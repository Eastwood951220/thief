package library

import (
	"os"
	"path/filepath"
	"testing"
	"thief/internal/book"
	"thief/internal/reader"
)

func TestStoreExclusiveDirectoryLock(t *testing.T) {
	root := t.TempDir()
	first, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	if second, e := Open(root); e == nil {
		second.Close()
		t.Fatal("two writers accepted")
	}
	other, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	other.Close()
	first.Close()
	again, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	again.Close()
}

func TestStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	b := book.Book{ID: "abc123", Title: "书", Format: "TXT", Chapters: []book.Chapter{{ID: "c1", Title: "第一章", Text: "正文。"}}}
	added, e := s.Import(b)
	if !added || e != nil {
		t.Fatal(added, e)
	}
	p := book.Position{ChapterID: "c1", Offset: 2}
	if e = s.SavePosition(b.ID, p); e != nil {
		t.Fatal(e)
	}
	added, e = s.Import(b)
	if added || e != nil {
		t.Fatal("duplicate", added, e)
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.LoadPosition(b.ID)
	if e != nil || got != p {
		t.Fatal(got, e)
	}
	loaded, e := s.Load(b.ID)
	if e != nil || loaded.Chapters[0].Text != "正文。" {
		t.Fatal(loaded, e)
	}
	settings := reader.DefaultSettings()
	settings.TextColor = "#000000"
	if e = s.SaveSettings(settings); e != nil {
		t.Fatal(e)
	}
	gotSettings, e := s.LoadSettings()
	if e != nil || gotSettings.TextColor != "#000000" {
		t.Fatal(e)
	}
	if e = s.Remove(b.ID); e != nil {
		t.Fatal(e)
	}
	list, _ := s.List()
	if len(list) != 0 {
		t.Fatal(list)
	}
}
func TestStoreFailurePreservesIndex(t *testing.T) {
	root := t.TempDir()
	s, _ := Open(root)
	b := book.Book{ID: "good", Title: "书", Chapters: []book.Chapter{{ID: "c", Text: "文"}}}
	if _, e := s.Import(b); e != nil {
		t.Fatal(e)
	}
	os.Remove(filepath.Join(root, "books", "good.json"))
	if _, e := s.Load("good"); e == nil {
		t.Fatal("missing book hidden")
	}
	list, e := s.List()
	if e != nil || len(list) != 1 {
		t.Fatal(list, e)
	}
	os.WriteFile(filepath.Join(root, "index.json"), []byte("broken"), 0600)
	if _, e := Open(root); e == nil {
		t.Fatal("corrupt index accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "index.json"))
	if string(raw) != "broken" {
		t.Fatal("index overwritten")
	}
	if _, e := s.Load("../escape"); e == nil {
		t.Fatal("traversal accepted")
	}
}

func TestIncompleteIndexNeverReplacesShelf(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"Version":1}`, `{"Version":1,"Books":null,"Positions":{}}`} {
		t.Run(raw, func(t *testing.T) {
			root := t.TempDir()
			os.MkdirAll(filepath.Join(root, "books"), 0700)
			os.WriteFile(filepath.Join(root, "index.json"), []byte(raw), 0600)
			if _, e := Open(root); e == nil {
				t.Fatal("incomplete index accepted")
			}
			after, _ := os.ReadFile(filepath.Join(root, "index.json"))
			if string(after) != raw {
				t.Fatal("overwrote index")
			}
		})
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "books"), 0700)
	os.WriteFile(filepath.Join(root, "books", "existing.json"), []byte(`{}`), 0600)
	if _, e := Open(root); e == nil {
		t.Fatal("missing shelf with existing book accepted")
	}
}
func TestPositionUsesLoadedSummary(t *testing.T) {
	root := t.TempDir()
	s, _ := Open(root)
	b := book.Book{ID: "book", Chapters: []book.Chapter{{ID: "a", Title: "一", Text: "甲乙"}, {ID: "b", Title: "二", Text: "丙丁戊"}}}
	s.Import(b)
	s.Load("book")
	os.WriteFile(filepath.Join(root, "books", "book.json"), []byte("temporarily unreadable"), 0600)
	if e := s.SavePosition("book", book.Position{ChapterID: "b", Offset: 1}); e != nil {
		t.Fatalf("page turn reread entire book: %v", e)
	}
	list, _ := s.List()
	if list[0].Progress != 0.6 || list[0].CurrentChapter != "二" {
		t.Fatal(list)
	}
	if e := s.SavePosition("book", book.Position{ChapterID: "missing", Offset: 0}); e == nil {
		t.Fatal("missing chapter accepted")
	}
}
