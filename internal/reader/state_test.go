package reader

import (
	"errors"
	"strings"
	"testing"
	"thief/internal/book"
)

type fixedMeasure struct{ n int }

func (m fixedMeasure) Fit(text string, l Layout) (int, error) {
	n := len([]rune(text))
	cap := m.n
	if l.Width > 0 {
		cap = int(l.Width)
	}
	if n < cap {
		return n, nil
	}
	return cap, nil
}
func testBook() book.Book {
	return book.Book{ID: "b", Chapters: []book.Chapter{{ID: "a", Title: "第一章", Text: "甲乙丙丁戊己庚辛壬癸"}, {ID: "b", Title: "第二章", Text: "一二三四五六"}}}
}
func TestPaginationContinuity(t *testing.T) {
	s, e := New(testBook(), book.Position{}, fixedMeasure{5})
	if e != nil {
		t.Fatal(e)
	}
	var got strings.Builder
	l := Layout{Width: 5}
	for i := 0; i < 4; i++ {
		p, e := s.Page(l)
		if e != nil {
			t.Fatal(e)
		}
		got.WriteString(p.Text)
		if i < 3 {
			if e = s.Next(l); e != nil {
				t.Fatal(e)
			}
		}
	}
	if got.String() != "甲乙丙丁戊己庚辛壬癸一二三四五六" {
		t.Fatal(got.String())
	}
	end := s.Position()
	s.Next(l)
	if s.Position() != end {
		t.Fatal("looped at end")
	}
	s.Previous(l)
	p, _ := s.Page(l)
	if p.Text != "一二三四五" {
		t.Fatal(p)
	}
	s.Jump("a")
	s.Next(l)
	p, _ = s.Page(Layout{Width: 3})
	if p.Start != 5 || p.Text != "己庚辛" {
		t.Fatal("anchor lost", p)
	}
	s.Previous(Layout{Width: 3})
	if s.Position().Offset >= 5 {
		t.Fatal("did not move back")
	}
}
func TestUnicodePageBoundary(t *testing.T) {
	b := book.Book{Chapters: []book.Chapter{{ID: "c", Text: "甲😀乙"}}}
	s, _ := New(b, book.Position{}, fixedMeasure{1})
	var all string
	for i := 0; i < 3; i++ {
		p, e := s.Page(Layout{})
		if e != nil {
			t.Fatal(e)
		}
		all += p.Text
		s.Next(Layout{})
	}
	if all != "甲😀乙" {
		t.Fatal(all)
	}
	s, _ = New(b, book.Position{}, zeroMeasure{})
	if _, e := s.Page(Layout{}); e == nil {
		t.Fatal("zero progress accepted")
	}
}

type zeroMeasure struct{}

func TestPreviousAfterSwitchingLayoutUsesCurrentAnchor(t *testing.T) {
	s, _ := New(testBook(), book.Position{}, fixedMeasure{5})
	s.Next(Layout{Width: 5})
	s.Next(Layout{Width: 3})
	if s.Position().Offset != 8 {
		t.Fatal(s.Position())
	}
	s.Previous(Layout{Width: 5})
	if s.Position().Offset != 5 {
		t.Fatalf("went to stale history: %+v", s.Position())
	}
}

func (zeroMeasure) Fit(string, Layout) (int, error) { return 0, nil }
func TestInvalidPosition(t *testing.T) {
	s, e := New(testBook(), book.Position{ChapterID: "gone", Offset: 999}, fixedMeasure{5})
	if e != nil {
		t.Fatal(e)
	}
	if s.Position().Offset != 0 {
		t.Fatal(s.Position())
	}
	if !errors.Is(s.Jump("missing"), ErrChapter) {
		t.Fatal("missing chapter accepted")
	}
}
