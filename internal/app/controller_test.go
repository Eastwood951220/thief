package app

import (
	"os"
	"path/filepath"
	"testing"
	"thief/internal/book"
	"thief/internal/library"
	"thief/internal/reader"
	"time"
)

func TestImportCompletionDoesNotSwitchBook(t *testing.T) {
	s, _ := library.Open(t.TempDir())
	s.Import(book.Book{ID: "existing", Title: "已在读", Chapters: []book.Chapter{{ID: "c", Text: "甲乙丙丁戊己"}}})
	c, _ := NewController(s, measure{})
	c.Dispatch(Command{Kind: "continue", BookID: "existing"})
	c.Dispatch(Command{Kind: "next"})
	p := filepath.Join(t.TempDir(), "new.txt")
	os.WriteFile(p, []byte("第一章 初见\n正文。"), 0600)
	changed := make(chan struct{}, 10)
	c.SetNotify(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if e := c.Dispatch(Command{Kind: "import", Path: p}); e != nil {
		t.Fatal(e)
	}
	deadline := time.After(3 * time.Second)
	for c.Snapshot().Busy {
		select {
		case <-changed:
		case <-deadline:
			t.Fatal("import timed out")
		}
	}
	got := c.Snapshot()
	if got.BookID != "existing" || got.Page.Start != 3 || len(got.Books) != 2 {
		t.Fatal(got)
	}
}

func TestCollapseInteraction(t *testing.T) {
	s, _ := library.Open(t.TempDir())
	s.Import(book.Book{ID: "book", Chapters: []book.Chapter{{ID: "c", Text: "甲乙丙丁"}}})
	c, _ := NewController(s, measure{})
	var fire func()
	var delay time.Duration
	c.after = func(d time.Duration, fn func()) stopper { delay = d; fire = fn; return &fakeTimer{} }
	c.Dispatch(Command{Kind: "continue", BookID: "book"})
	c.Dispatch(Command{Kind: "toggleMode"})
	c.Dispatch(Command{Kind: "hoverLeave"})
	if delay != 400*time.Millisecond || c.Snapshot().Collapsed {
		t.Fatal("incorrect delay")
	}
	c.Dispatch(Command{Kind: "interactionBegin"})
	fire()
	if c.Snapshot().Collapsed {
		t.Fatal("collapsed during menu or drag")
	}
	c.Dispatch(Command{Kind: "interactionEnd"})
	c.Dispatch(Command{Kind: "hoverLeave"})
	fire()
	if !c.Snapshot().Collapsed {
		t.Fatal("not collapsed")
	}
	c.Dispatch(Command{Kind: "hoverEnter"})
	if c.Snapshot().Collapsed {
		t.Fatal("did not restore")
	}
	c.Dispatch(Command{Kind: "toggleHidden"})
	fire()
	c.Dispatch(Command{Kind: "hoverEnter"})
	if !c.Snapshot().Hidden || c.Snapshot().Collapsed {
		t.Fatal("hidden restored")
	}
}

type fakeTimer struct{}

func (*fakeTimer) Stop() bool { return true }

func TestHotkeyReplacementRollback(t *testing.T) {
	s, _ := library.Open(t.TempDir())
	c, _ := NewController(s, measure{})
	original := c.Snapshot()
	settings := original.Settings
	settings.Hotkeys = map[string]reader.Binding{}
	for k, v := range original.Settings.Hotkeys {
		settings.Hotkeys[k] = v
	}
	settings.Hotkeys["next"] = reader.Binding{KeyCode: 38, Modifiers: 6144}
	if e := c.Dispatch(Command{Kind: "saveSettings", Settings: &settings}); e != nil {
		t.Fatal(e)
	}
	pending := c.Snapshot()
	stored, _ := s.LoadSettings()
	if stored.Hotkeys["next"].KeyCode != 124 || pending.PendingSettings == nil || pending.RequestID == 0 {
		t.Fatal("saved before registration", pending)
	}
	c.Dispatch(Command{Kind: "settingsRejected", RequestID: pending.RequestID, Query: "快捷键已占用"})
	if c.Snapshot().Settings.Hotkeys["next"].KeyCode != 124 {
		t.Fatal("old binding lost")
	}
	c.Dispatch(Command{Kind: "saveSettings", Settings: &settings})
	newer := c.Snapshot()
	c.Dispatch(Command{Kind: "settingsAccepted", RequestID: pending.RequestID})
	if c.Snapshot().PendingSettings == nil {
		t.Fatal("stale confirmation accepted")
	}
	c.Dispatch(Command{Kind: "settingsAccepted", RequestID: newer.RequestID})
	stored, _ = s.LoadSettings()
	if stored.Hotkeys["next"].KeyCode != 38 {
		t.Fatal("success not persisted")
	}
	if c.Snapshot().Revision <= original.Revision {
		t.Fatal("snapshot ordering missing")
	}
}

type measure struct{}

func (measure) Fit(text string, l reader.Layout) (int, error) {
	r := []rune(text)
	if len(r) > 3 {
		return 3, nil
	}
	return len(r), nil
}
func TestHiddenState(t *testing.T) {
	s, _ := library.Open(t.TempDir())
	s.Import(book.Book{ID: "book", Title: "书", Chapters: []book.Chapter{{ID: "c", Title: "一", Text: "甲乙丙丁戊己"}}})
	c, e := NewController(s, measure{})
	if e != nil {
		t.Fatal(e)
	}
	c.Dispatch(Command{Kind: "continue", BookID: "book"})
	c.Dispatch(Command{Kind: "toggleHidden"})
	c.Dispatch(Command{Kind: "next"})
	c.Dispatch(Command{Kind: "toggleMode"})
	c.Dispatch(Command{Kind: "hoverEnter"})
	if !c.Snapshot().Hidden {
		t.Fatal("hidden text restored")
	}
	c.Dispatch(Command{Kind: "toggleHidden"})
	if c.Snapshot().Hidden {
		t.Fatal("restore failed")
	}
	c.Close()
	again, e := NewController(s, measure{})
	if e != nil {
		t.Fatal(e)
	}
	if !again.Snapshot().Hidden || again.Snapshot().Page.Start != 3 {
		t.Fatal(again.Snapshot())
	}
}
func TestChapterSearch(t *testing.T) {
	s, _ := library.Open(t.TempDir())
	s.Import(book.Book{ID: "book", Title: "书", Chapters: []book.Chapter{{ID: "a", Title: "初见", Text: "甲"}, {ID: "b", Title: "归途", Text: "乙"}}})
	c, _ := NewController(s, measure{})
	rows, e := c.Chapters("book", "2")
	if e != nil || len(rows) != 1 || rows[0].ID != "b" {
		t.Fatal(rows, e)
	}
	c.Dispatch(Command{Kind: "jump", BookID: "book", ChapterID: "b"})
	if c.Snapshot().Page.ChapterTitle != "归途" || c.Snapshot().Page.Start != 0 {
		t.Fatal(c.Snapshot())
	}
}
