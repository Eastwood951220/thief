package app

import (
	"testing"
	"thief/internal/book"
	"thief/internal/library"
	"thief/internal/reader"
)

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
