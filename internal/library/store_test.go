package library

import (
	"os"
	"path/filepath"
	"testing"
	"thief/internal/book"
	"thief/internal/reader"
)

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
