package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"thief/internal/book"
	"thief/internal/reader"
	"time"
)

type index struct {
	Version   int
	Books     []book.Metadata
	Positions map[string]book.Position
	LastBook  string
}
type Store struct {
	mu    sync.Mutex
	root  string
	index index
}

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func Open(root string) (*Store, error) {
	if e := os.MkdirAll(filepath.Join(root, "books"), 0700); e != nil {
		return nil, e
	}
	s := &Store{root: root, index: index{Version: 1, Books: []book.Metadata{}, Positions: map[string]book.Position{}}}
	raw, e := os.ReadFile(filepath.Join(root, "index.json"))
	if e == nil {
		if e = json.Unmarshal(raw, &s.index); e != nil {
			return nil, fmt.Errorf("书架索引损坏: %w", e)
		}
		if s.index.Version != 1 {
			return nil, fmt.Errorf("不支持的书架版本")
		}
		if s.index.Positions == nil {
			s.index.Positions = map[string]book.Position{}
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	return s, nil
}
func (s *Store) path(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", fmt.Errorf("无效书籍标识")
	}
	return filepath.Join(s.root, "books", id+".json"), nil
}
func cloneIndex(i index) index {
	c := i
	c.Books = append([]book.Metadata{}, i.Books...)
	c.Positions = map[string]book.Position{}
	for k, v := range i.Positions {
		c.Positions[k] = v
	}
	return c
}
func (s *Store) save(next index) error {
	if e := atomicJSON(filepath.Join(s.root, "index.json"), next); e != nil {
		return e
	}
	s.index = next
	return nil
}
func (s *Store) List() ([]book.Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]book.Metadata{}, s.index.Books...), nil
}
func (s *Store) Import(b book.Book) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, e := s.path(b.ID)
	if e != nil {
		return false, e
	}
	for _, m := range s.index.Books {
		if m.ID == b.ID {
			return false, nil
		}
	}
	if len(b.Chapters) == 0 {
		return false, fmt.Errorf("书籍没有正文")
	}
	if e = atomicJSON(p, b); e != nil {
		return false, e
	}
	next := cloneIndex(s.index)
	next.Books = append(next.Books, book.Metadata{ID: b.ID, Title: b.Title, Format: b.Format, ImportedAt: time.Now()})
	if e = s.save(next); e != nil {
		os.Remove(p)
		return false, e
	}
	return true, nil
}
func (s *Store) Load(id string) (book.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, e := s.path(id)
	if e != nil {
		return book.Book{}, e
	}
	raw, e := os.ReadFile(p)
	if e != nil {
		return book.Book{}, fmt.Errorf("读取书籍失败: %w", e)
	}
	var b book.Book
	e = json.Unmarshal(raw, &b)
	return b, e
}
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, e := s.path(id)
	if e != nil {
		return e
	}
	next := cloneIndex(s.index)
	next.Books = nil
	for _, m := range s.index.Books {
		if m.ID != id {
			next.Books = append(next.Books, m)
		}
	}
	delete(next.Positions, id)
	if next.LastBook == id {
		next.LastBook = ""
	}
	if e = s.save(next); e != nil {
		return e
	}
	if e = os.Remove(p); e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return nil
}
func (s *Store) SavePosition(id string, p book.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneIndex(s.index)
	found := false
	for i, m := range next.Books {
		if m.ID == id {
			found = true
			raw, e := os.ReadFile(filepath.Join(s.root, "books", id+".json"))
			if e != nil {
				return e
			}
			var b book.Book
			if e = json.Unmarshal(raw, &b); e != nil {
				return e
			}
			total, done := 0, 0
			passed := false
			for _, c := range b.Chapters {
				n := len([]rune(c.Text))
				total += n
				if c.ID == p.ChapterID {
					next.Books[i].CurrentChapter = c.Title
					done += p.Offset
					passed = true
				} else if !passed {
					done += n
				}
			}
			if total > 0 {
				next.Books[i].Progress = float64(done) / float64(total)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("书籍不存在")
	}
	next.Positions[id] = p
	next.LastBook = id
	return s.save(next)
}
func (s *Store) LoadPosition(id string) (book.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.index.Positions[id], nil
}
func (s *Store) LastBook() string { s.mu.Lock(); defer s.mu.Unlock(); return s.index.LastBook }
func (s *Store) SaveSettings(v reader.Settings) error {
	if e := v.Validate(); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicJSON(filepath.Join(s.root, "settings.json"), v)
}
func (s *Store) LoadSettings() (reader.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := reader.DefaultSettings()
	raw, e := os.ReadFile(filepath.Join(s.root, "settings.json"))
	if errors.Is(e, os.ErrNotExist) {
		return v, nil
	}
	if e != nil {
		return v, e
	}
	if e = json.Unmarshal(raw, &v); e != nil {
		return reader.DefaultSettings(), e
	}
	if e = v.Validate(); e != nil {
		return reader.DefaultSettings(), e
	}
	return v, nil
}
