package reader

import (
	"errors"
	"fmt"
	"github.com/rivo/uniseg"
	"sort"
	"thief/internal/book"
	"unicode/utf8"
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

type chapterText struct {
	runes  []rune
	bounds []int
}
type pageMove struct{ From, To book.Position }
type Session struct {
	Book            book.Book
	chapter, offset int
	measure         Measurer
	history         map[Layout][]pageMove
	texts           map[int]chapterText
}

func New(b book.Book, p book.Position, m Measurer) (*Session, error) {
	if len(b.Chapters) == 0 {
		return nil, fmt.Errorf("没有可读章节")
	}
	s := &Session{Book: b, measure: m, history: map[Layout][]pageMove{}, texts: map[int]chapterText{}}
	for i, c := range b.Chapters {
		if c.ID == p.ChapterID {
			s.chapter = i
			if p.Offset >= 0 && p.Offset < utf8.RuneCountInString(c.Text) {
				s.offset = p.Offset
			}
			break
		}
	}
	text := s.text()
	idx := sort.SearchInts(text.bounds, s.offset)
	if idx == len(text.bounds) || text.bounds[idx] != s.offset {
		s.offset = text.bounds[max(0, idx-1)]
	}
	return s, nil
}
func (s *Session) text() chapterText {
	if t, ok := s.texts[s.chapter]; ok {
		return t
	}
	text := s.Book.Chapters[s.chapter].Text
	t := chapterText{runes: []rune(text), bounds: []int{0}}
	g := uniseg.NewGraphemes(text)
	n := 0
	for g.Next() {
		n += utf8.RuneCountInString(g.Str())
		t.bounds = append(t.bounds, n)
	}
	s.texts[s.chapter] = t
	return t
}
func (s *Session) Position() book.Position {
	return book.Position{ChapterID: s.Book.Chapters[s.chapter].ID, Offset: s.offset}
}
func snippetLimit(l Layout) int {
	if l.FontSize <= 0 {
		return 512
	}
	rows := max(1, l.Height/l.FontSize)
	if l.SingleLine {
		rows = 1
	}
	return min(16384, max(256, int(l.Width/l.FontSize*rows*2)+32))
}
func (s *Session) Page(l Layout) (Page, error) {
	c := s.Book.Chapters[s.chapter]
	t := s.text()
	r := t.runes
	if s.offset < 0 || s.offset > len(r) {
		return Page{}, fmt.Errorf("阅读位置无效")
	}
	if len(r) == 0 {
		return Page{ChapterID: c.ID, ChapterTitle: c.Title}, nil
	}
	limit := snippetLimit(l)
	for {
		end := min(len(r), s.offset+limit)
		n, e := s.measure.Fit(string(r[s.offset:end]), l)
		if e != nil {
			return Page{}, e
		}
		if n < 0 || n > end-s.offset {
			return Page{}, fmt.Errorf("无效分页范围")
		}
		if n == end-s.offset && end < len(r) {
			limit *= 2
			continue
		}
		absolute := s.offset + n
		idx := sort.SearchInts(t.bounds, absolute)
		if idx == len(t.bounds) || t.bounds[idx] != absolute {
			absolute = t.bounds[max(0, idx-1)]
		}
		if absolute <= s.offset {
			return Page{}, fmt.Errorf("区域无法容纳文字，请增大阅读区域或减小字号")
		}
		return Page{ChapterID: c.ID, ChapterTitle: c.Title, Text: string(r[s.offset:absolute]), Start: s.offset, End: absolute}, nil
	}
}
func (s *Session) Next(l Layout) error {
	p, e := s.Page(l)
	if e != nil {
		return e
	}
	old := s.Position()
	if p.End < len(s.text().runes) {
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
		s.history[l] = append(s.history[l], pageMove{old, s.Position()})
		if len(s.history[l]) > 1024 {
			s.history[l] = s.history[l][len(s.history[l])-1024:]
		}
	}
	return nil
}
func (s *Session) Previous(l Layout) error {
	originalChapter, originalOffset := s.chapter, s.offset
	target := s.offset
	if target == 0 {
		found := false
		for i := s.chapter - 1; i >= 0; i-- {
			if s.Book.Chapters[i].Text != "" {
				s.chapter = i
				target = len(s.text().runes)
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	if h := s.history[l]; len(h) > 0 && h[len(h)-1].To == (book.Position{ChapterID: s.Book.Chapters[originalChapter].ID, Offset: originalOffset}) {
		p := h[len(h)-1].From
		s.history[l] = h[:len(h)-1]
		for i, c := range s.Book.Chapters {
			if c.ID == p.ChapterID {
				s.chapter = i
				s.offset = p.Offset
				return nil
			}
		}
	}
	// Reuse a known page covering the preceding anchor after a layout switch.
	candidate := -1
	for _, move := range s.history[l] {
		for _, p := range []book.Position{move.From, move.To} {
			if p.ChapterID == s.Book.Chapters[s.chapter].ID && p.Offset < target && p.Offset > candidate {
				candidate = p.Offset
			}
		}
	}
	if candidate >= 0 {
		s.offset = candidate
		p, e := s.Page(l)
		if e == nil && p.End >= target {
			return nil
		}
	}
	t := s.text()
	endIndex := sort.SearchInts(t.bounds, target)
	limit := snippetLimit(l)
	lowIndex := 0
	for {
		start := max(0, target-limit)
		lowIndex = sort.SearchInts(t.bounds, start)
		n, e := s.measure.Fit(string(t.runes[t.bounds[lowIndex]:target]), l)
		if e != nil {
			s.chapter = originalChapter
			s.offset = originalOffset
			return e
		}
		if n < target-t.bounds[lowIndex] || lowIndex == 0 {
			break
		}
		limit *= 2
	}
	lo, hi := lowIndex, endIndex
	for lo < hi {
		mid := (lo + hi) / 2
		start := t.bounds[mid]
		n, e := s.measure.Fit(string(t.runes[start:target]), l)
		if e != nil {
			s.chapter = originalChapter
			s.offset = originalOffset
			return e
		}
		if n >= target-start {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	if lo >= endIndex {
		s.chapter = originalChapter
		s.offset = originalOffset
		return fmt.Errorf("区域无法容纳文字")
	}
	s.offset = t.bounds[lo]
	return nil
}
func (s *Session) Jump(id string) error {
	for i, c := range s.Book.Chapters {
		if c.ID == id {
			s.chapter = i
			s.offset = 0
			s.history = map[Layout][]pageMove{}
			return nil
		}
	}
	return ErrChapter
}
func (s *Session) Chapter(delta int) error {
	if delta == 0 {
		return nil
	}
	for i := s.chapter + delta; i >= 0 && i < len(s.Book.Chapters); i += delta {
		if s.Book.Chapters[i].Text != "" {
			return s.Jump(s.Book.Chapters[i].ID)
		}
	}
	return nil
}
