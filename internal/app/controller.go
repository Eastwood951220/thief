package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"thief/internal/book"
	"thief/internal/importer"
	"thief/internal/library"
	"thief/internal/reader"
	"time"
)

type Command struct {
	Kind, BookID, ChapterID, Path, Query string
	Options                              importer.Options
	Settings                             *reader.Settings
	Frame                                *reader.Rect
	RequestID                            uint64
}
type ChapterRow struct {
	ID, Title, Volume string
	Number            int
	Current           bool
}
type Snapshot struct {
	Books                           []book.Metadata
	Settings                        reader.Settings
	Page                            reader.Page
	BookID, BookTitle               string
	Hidden, Collapsed, Busy         bool
	Error, Notice                   string
	Chapters                        []ChapterRow
	HotkeyStatus                    map[string]string
	Revision, RequestID, ReadAction uint64
	PendingSettings                 *reader.Settings
}
type Controller struct {
	mu                              sync.Mutex
	store                           *library.Store
	measure                         reader.Measurer
	settings                        reader.Settings
	session                         *reader.Session
	hidden, collapsed, busy, closed bool
	interactions                    int
	error, notice                   string
	timer                           stopper
	after                           func(time.Duration, func()) stopper
	epoch                           uint64
	revision, requestID, readAction uint64
	pendingSettings                 *reader.Settings
	notify                          func()
	ctx                             context.Context
	cancel                          context.CancelFunc
}
type stopper interface{ Stop() bool }

func NewController(s *library.Store, m reader.Measurer) (*Controller, error) {
	ctx, cancel := context.WithCancel(context.Background())
	settings, e := s.LoadSettings()
	c := &Controller{store: s, measure: m, settings: settings, hidden: true, ctx: ctx, cancel: cancel}
	c.after = func(d time.Duration, fn func()) stopper { return time.AfterFunc(d, fn) }
	if e != nil {
		c.error = e.Error()
	}
	if id := s.LastBook(); id != "" {
		if e = c.load(id); e != nil {
			c.error = e.Error()
		}
	}
	return c, nil
}
func (c *Controller) SetNotify(fn func()) { c.mu.Lock(); defer c.mu.Unlock(); c.notify = fn }
func (c *Controller) changed() {
	c.mu.Lock()
	fn := c.notify
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}
func (c *Controller) load(id string) error {
	b, e := c.store.Load(id)
	if e != nil {
		return e
	}
	p, e := c.store.LoadPosition(id)
	if e != nil {
		return e
	}
	session, e := reader.New(b, p, c.measure)
	if e != nil {
		return e
	}
	c.session = session
	return nil
}
func (c *Controller) layout() reader.Layout {
	if c.settings.Mode == "menu" {
		return reader.Layout{Width: c.settings.MenuWidth - 12, Height: 24, FontSize: c.settings.MenuFontSize, SingleLine: true}
	}
	return reader.Layout{Width: c.settings.Frame.Width - 40, Height: c.settings.Frame.Height - 80, FontSize: c.settings.FontSize, LineSpacing: c.settings.LineSpacing}
}
func (c *Controller) save() error {
	if c.session != nil {
		return c.store.SavePosition(c.session.Book.ID, c.session.Position())
	}
	return nil
}
func (c *Controller) Dispatch(cmd Command) error {
	c.mu.Lock()
	e := c.dispatch(cmd)
	c.revision++
	if e != nil {
		c.error = e.Error()
	}
	c.mu.Unlock()
	c.changed()
	return e
}
func (c *Controller) dispatch(cmd Command) error {
	if c.closed {
		return fmt.Errorf("应用已关闭")
	}
	c.error = ""
	c.notice = ""
	switch cmd.Kind {
	case "import":
		if c.busy {
			return fmt.Errorf("正在导入，请稍后")
		}
		if cmd.Path == "" {
			return fmt.Errorf("请选择文件")
		}
		c.busy = true
		go func() {
			b, e := importer.Parse(c.ctx, cmd.Path, cmd.Options)
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			if e == nil {
				var added bool
				added, e = c.store.Import(b)
				if e == nil {
					if added {
						c.notice = "已导入《" + b.Title + "》"
					} else {
						c.notice = "书架中已存在这本书，阅读进度已保留"
					}
				}
			}
			if e != nil {
				c.error = e.Error()
			}
			c.busy = false
			c.revision++
			c.mu.Unlock()
			c.changed()
		}()
		return nil
	case "continue", "jump":
		c.epoch++
		if c.timer != nil {
			c.timer.Stop()
		}
		if e := c.save(); e != nil {
			return e
		}
		if cmd.BookID != "" {
			if e := c.load(cmd.BookID); e != nil {
				return e
			}
		}
		if c.session == nil {
			return fmt.Errorf("请先从书架选择书籍")
		}
		if cmd.Kind == "jump" {
			if e := c.session.Jump(cmd.ChapterID); e != nil {
				return e
			}
		}
		c.hidden = false
		c.collapsed = false
		if e := c.save(); e != nil {
			return e
		}
		c.readAction++
		return nil
	case "remove":
		if e := c.store.Remove(cmd.BookID); e != nil {
			return e
		}
		if c.session != nil && c.session.Book.ID == cmd.BookID {
			c.session = nil
			c.hidden = true
		}
		return nil
	case "next", "previous", "nextChapter", "previousChapter":
		if c.session == nil {
			return nil
		}
		var e error
		switch cmd.Kind {
		case "next":
			e = c.session.Next(c.layout())
		case "previous":
			e = c.session.Previous(c.layout())
		case "nextChapter":
			e = c.session.Chapter(1)
		case "previousChapter":
			e = c.session.Chapter(-1)
		}
		if e != nil {
			return e
		}
		return c.save()
	case "toggleHidden":
		c.epoch++
		if c.session == nil {
			return fmt.Errorf("请先选择一本书")
		}
		c.hidden = !c.hidden
		c.collapsed = false
		if c.timer != nil {
			c.timer.Stop()
		}
	case "toggleMode":
		c.epoch++
		c.pendingSettings = nil
		c.requestID++
		if c.settings.Mode == "menu" {
			c.settings.Mode = "float"
		} else {
			c.settings.Mode = "menu"
		}
		return c.store.SaveSettings(c.settings)
	case "saveSettings":
		if cmd.Settings == nil {
			return fmt.Errorf("缺少设置")
		}
		if e := cmd.Settings.Validate(); e != nil {
			return e
		}
		c.requestID++
		copy := *cmd.Settings
		c.pendingSettings = &copy
	case "settingsAccepted":
		if c.pendingSettings == nil || cmd.RequestID != c.requestID {
			return nil
		}
		next := *c.pendingSettings
		c.pendingSettings = nil
		if e := c.store.SaveSettings(next); e != nil {
			return e
		}
		c.settings = next
		c.epoch++
		if !next.AutoCollapse {
			c.collapsed = false
		}
	case "settingsRejected":
		if c.pendingSettings != nil && cmd.RequestID == c.requestID {
			c.pendingSettings = nil
			c.error = cmd.Query
		}
	case "setFrame":
		if cmd.Frame == nil {
			return nil
		}
		next := c.settings
		next.Frame = *cmd.Frame
		if e := next.Validate(); e != nil {
			return e
		}
		if e := c.store.SaveSettings(next); e != nil {
			return e
		}
		c.settings = next
	case "hoverEnter":
		c.epoch++
		if c.timer != nil {
			c.timer.Stop()
		}
		if !c.hidden {
			c.collapsed = false
		}
	case "hoverLeave":
		c.scheduleCollapse()
	case "interactionBegin":
		c.epoch++
		c.interactions++
		if c.timer != nil {
			c.timer.Stop()
		}
	case "interactionEnd":
		if c.interactions > 0 {
			c.interactions--
		}
	case "clearError":
		c.error = ""
	case "reportError":
		c.error = cmd.Query
	}
	return nil
}
func (c *Controller) scheduleCollapse() {
	if c.hidden || !c.settings.AutoCollapse || c.interactions > 0 || c.settings.Mode != "float" {
		return
	}
	if c.timer != nil {
		c.timer.Stop()
	}
	c.epoch++
	epoch := c.epoch
	c.timer = c.after(time.Duration(c.settings.CollapseDelayMS)*time.Millisecond, func() {
		c.mu.Lock()
		if c.epoch == epoch && !c.hidden && c.interactions == 0 && !c.closed && c.settings.AutoCollapse && c.settings.Mode == "float" {
			c.collapsed = true
			c.revision++
		}
		c.mu.Unlock()
		c.changed()
	})
}
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	books, e := c.store.List()
	s := Snapshot{Books: books, Settings: c.settings, Hidden: c.hidden, Collapsed: c.collapsed, Busy: c.busy, Error: c.error, Notice: c.notice, Revision: c.revision, RequestID: c.requestID, ReadAction: c.readAction, PendingSettings: c.pendingSettings}
	if e != nil {
		s.Error = e.Error()
	}
	if c.session != nil {
		s.BookID = c.session.Book.ID
		s.BookTitle = c.session.Book.Title
		s.Page, e = c.session.Page(c.layout())
		if e != nil {
			s.Error = e.Error()
		}
	}
	return s
}
func (c *Controller) Chapters(id, query string) ([]ChapterRow, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, e := c.store.Load(id)
	if e != nil {
		return nil, e
	}
	p, _ := c.store.LoadPosition(id)
	rows := []ChapterRow{}
	for i, ch := range b.Chapters {
		if query != "" && !strings.Contains(strings.ToLower(ch.Title), strings.ToLower(query)) && !strings.Contains(fmt.Sprint(i+1), query) {
			continue
		}
		rows = append(rows, ChapterRow{ID: ch.ID, Title: ch.Title, Volume: ch.Volume, Number: i + 1, Current: p.ChapterID == ch.ID})
	}
	return rows, nil
}
func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.save(); e != nil {
		return e
	}
	c.closed = true
	c.cancel()
	if c.timer != nil {
		c.timer.Stop()
	}
	return nil
}
