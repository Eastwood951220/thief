package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"thief/internal/book"
	"thief/internal/reader"
	"time"
	"unicode/utf8"
)

type index struct {
	Version   int
	Books     []book.Metadata
	Positions map[string]book.Position
	LastBook  string
}
type Store struct {
	mu        sync.Mutex
	root      string
	index     index
	summaries map[string]summary
	lockFile  *os.File
}
type chapterSummary struct {
	ID, Title     string
	Start, Length int
}
type summary struct {
	Chapters []chapterSummary
	Total    int
}

func summarize(b book.Book) summary {
	s := summary{}
	for _, c := range b.Chapters {
		n := utf8.RuneCountInString(c.Text)
		s.Chapters = append(s.Chapters, chapterSummary{c.ID, c.Title, s.Total, n})
		s.Total += n
	}
	return s
}

var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func Open(root string) (*Store, error) {
	if e := os.MkdirAll(filepath.Join(root, "books"), 0700); e != nil {
		return nil, e
	}
	lock, e := os.OpenFile(filepath.Join(root, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		lock.Close()
		return nil, fmt.Errorf("已有 thief 实例正在使用这个书架，请先退出另一个实例")
	}
	success := false
	defer func() {
		if !success {
			syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			lock.Close()
		}
	}()
	s := &Store{root: root, index: index{Version: 1, Books: []book.Metadata{}, Positions: map[string]book.Position{}}, summaries: map[string]summary{}}
	s.lockFile = lock
	raw, e := os.ReadFile(filepath.Join(root, "index.json"))
	if e == nil {
		var fields map[string]json.RawMessage
		if e = json.Unmarshal(raw, &fields); e != nil || fields == nil || fields["Version"] == nil || fields["Books"] == nil || string(fields["Books"]) == "null" || fields["Positions"] == nil || string(fields["Positions"]) == "null" {
			return nil, fmt.Errorf("书架索引不完整，已保留原文件，请从备份恢复")
		}
		if e = json.Unmarshal(raw, &s.index); e != nil {
			return nil, fmt.Errorf("书架索引损坏: %w", e)
		}
		if s.index.Version != 1 {
			return nil, fmt.Errorf("不支持的书架版本")
		}
		if s.index.Positions == nil {
			s.index.Positions = map[string]book.Position{}
		}
		seen := map[string]bool{}
		for _, m := range s.index.Books {
			if !validID.MatchString(m.ID) || seen[m.ID] {
				return nil, fmt.Errorf("书架包含无效或重复书籍标识")
			}
			seen[m.ID] = true
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return nil, e
	} else {
		entries, err := os.ReadDir(filepath.Join(root, "books"))
		if err != nil {
			return nil, err
		}
		if len(entries) > 0 {
			return nil, fmt.Errorf("书架索引缺失但书籍副本仍在，已停止写入，请从备份恢复")
		}
	}
	success = true
	return s, nil
}
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockFile == nil {
		return nil
	}
	syscall.Flock(int(s.lockFile.Fd()), syscall.LOCK_UN)
	e := s.lockFile.Close()
	s.lockFile = nil
	return e
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
	s.summaries[b.ID] = summarize(b)
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
	if e == nil {
		if b.ID != id || len(b.Chapters) == 0 {
			return book.Book{}, fmt.Errorf("书籍副本不完整")
		}
		s.summaries[id] = summarize(b)
	}
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
	next.Books = []book.Metadata{}
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
	delete(s.summaries, id)
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
			sum, ok := s.summaries[id]
			if !ok {
				raw, e := os.ReadFile(filepath.Join(s.root, "books", id+".json"))
				if e != nil {
					return e
				}
				var b book.Book
				if e = json.Unmarshal(raw, &b); e != nil {
					return e
				}
				sum = summarize(b)
				s.summaries[id] = sum
			}
			chapterFound := false
			for _, ch := range sum.Chapters {
				if ch.ID == p.ChapterID {
					if p.Offset < 0 || p.Offset > ch.Length {
						return fmt.Errorf("阅读偏移超出章节")
					}
					chapterFound = true
					next.Books[i].CurrentChapter = ch.Title
					if sum.Total > 0 {
						next.Books[i].Progress = float64(ch.Start+p.Offset) / float64(sum.Total)
					}
					break
				}
			}
			if !chapterFound {
				return fmt.Errorf("阅读章节不存在")
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
