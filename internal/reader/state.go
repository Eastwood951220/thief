package reader

import (
	"errors"
	"fmt"
	"thief/internal/book"
)

type Layout struct {
	Width, Height, FontSize, LineSpacing float64
	SingleLine                           bool
}
type Measurer interface {
	Fit(string, Layout) (int, error)
}
type Page struct {
	Text, ChapterID, ChapterTitle string
	Start, End                    int
}

var ErrChapter = errors.New("章节不存在")

type Session struct {
	Book            book.Book
	chapter, offset int
	measure         Measurer
	history         map[Layout][]book.Position
}

func New(b book.Book, p book.Position, m Measurer) (*Session, error) {
	if len(b.Chapters) == 0 {
		return nil, fmt.Errorf("没有可读章节")
	}
	s := &Session{Book: b, measure: m, history: map[Layout][]book.Position{}}
	for i, c := range b.Chapters {
		if c.ID == p.ChapterID {
			s.chapter = i
			if p.Offset >= 0 && p.Offset < len([]rune(c.Text)) {
				s.offset = p.Offset
			}
			break
		}
	}
	return s, nil
}
func (s *Session) Position() book.Position {
	return book.Position{ChapterID: s.Book.Chapters[s.chapter].ID, Offset: s.offset}
}
func (s *Session) Page(l Layout) (Page, error) {
	c := s.Book.Chapters[s.chapter]
	r := []rune(c.Text)
	if s.offset < 0 || s.offset > len(r) {
		return Page{}, fmt.Errorf("阅读位置无效")
	}
	if len(r) == 0 {
		return Page{ChapterID: c.ID, ChapterTitle: c.Title}, nil
	}
	n, e := s.measure.Fit(string(r[s.offset:]), l)
	if e != nil {
		return Page{}, e
	}
	if n <= 0 || n > len(r)-s.offset {
		return Page{}, fmt.Errorf("区域无法容纳文字，请增大阅读区域或减小字号")
	}
	return Page{ChapterID: c.ID, ChapterTitle: c.Title, Text: string(r[s.offset : s.offset+n]), Start: s.offset, End: s.offset + n}, nil
}
func (s *Session) Next(l Layout) error {
	p, e := s.Page(l)
	if e != nil {
		return e
	}
	old := s.Position()
	if p.End < len([]rune(s.Book.Chapters[s.chapter].Text)) {
		s.offset = p.End
	} else {
		for i := s.chapter + 1; i < len(s.Book.Chapters); i++ {
			if s.Book.Chapters[i].Text != "" {
				s.chapter = i
				s.offset = 0
				break
			}
		}
	}
	if old != s.Position() {
		s.history[l] = append(s.history[l], old)
	}
	return nil
}
func (s *Session) Previous(l Layout) error {
	if h := s.history[l]; len(h) > 0 {
		p := h[len(h)-1]
		s.history[l] = h[:len(h)-1]
		for i, c := range s.Book.Chapters {
			if c.ID == p.ChapterID {
				s.chapter = i
				s.offset = p.Offset
				return nil
			}
		}
	}
	target := s.offset
	if target == 0 {
		found := false
		for i := s.chapter - 1; i >= 0; i-- {
			if s.Book.Chapters[i].Text != "" {
				s.chapter = i
				target = len([]rune(s.Book.Chapters[i].Text))
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	s.offset = 0
	for {
		p, e := s.Page(l)
		if e != nil {
			return e
		}
		if p.End >= target || p.End <= s.offset {
			return nil
		}
		s.offset = p.End
	}
}
func (s *Session) Jump(id string) error {
	for i, c := range s.Book.Chapters {
		if c.ID == id {
			s.chapter = i
			s.offset = 0
			s.history = map[Layout][]book.Position{}
			return nil
		}
	}
	return ErrChapter
}
func (s *Session) Chapter(delta int) error {
	for i := s.chapter + delta; i >= 0 && i < len(s.Book.Chapters); i += delta {
		if s.Book.Chapters[i].Text != "" {
			return s.Jump(s.Book.Chapters[i].ID)
		}
	}
	return nil
}
