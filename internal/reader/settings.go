package reader

import (
	"fmt"
	"math"
	"regexp"
)

type Rect struct{ X, Y, Width, Height float64 }
type Binding struct{ KeyCode, Modifiers uint32 }
type Settings struct {
	Mode                                                                                     string
	Frame                                                                                    Rect
	BackgroundColor, TextColor                                                               string
	BackgroundTransparency, TextTransparency, FontSize, LineSpacing, MenuWidth, MenuFontSize float64
	AutoCollapse                                                                             bool
	CollapseDelayMS                                                                          int
	Hotkeys                                                                                  map[string]Binding
}

func DefaultSettings() Settings {
	return Settings{Mode: "menu", Frame: Rect{100, 100, 355, 236}, BackgroundColor: "#F5F3EB", TextColor: "#343B43", BackgroundTransparency: 14, FontSize: 16, LineSpacing: 4, MenuWidth: 320, MenuFontSize: 13, AutoCollapse: true, CollapseDelayMS: 400, Hotkeys: map[string]Binding{"previous": {123, 6144}, "next": {124, 6144}, "previousChapter": {126, 6144}, "nextChapter": {125, 6144}, "toggleHidden": {4, 6144}, "toggleMode": {46, 6144}}}
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s Settings) Validate() error {
	if s.Mode != "menu" && s.Mode != "float" {
		return fmt.Errorf("阅读模式无效")
	}
	if !hexColor.MatchString(s.BackgroundColor) || !hexColor.MatchString(s.TextColor) {
		return fmt.Errorf("颜色请输入 # 加六位十六进制字符")
	}
	for _, v := range []struct {
		n         string
		v, lo, hi float64
	}{{"宽度", s.Frame.Width, 200, 1600}, {"高度", s.Frame.Height, 120, 1200}, {"字号", s.FontSize, 10, 48}, {"行距", s.LineSpacing, 0, 32}, {"菜单栏宽度", s.MenuWidth, 80, 800}, {"菜单栏字号", s.MenuFontSize, 10, 24}, {"背景透明度", s.BackgroundTransparency, 0, 100}, {"文字透明度", s.TextTransparency, 0, 100}, {"X", s.Frame.X, -100000, 100000}, {"Y", s.Frame.Y, -100000, 100000}} {
		if math.IsNaN(v.v) || math.IsInf(v.v, 0) || v.v < v.lo || v.v > v.hi {
			return fmt.Errorf("%s超出范围 %.0f–%.0f", v.n, v.lo, v.hi)
		}
	}
	if s.CollapseDelayMS < 0 || s.CollapseDelayMS > 3000 {
		return fmt.Errorf("收起延迟范围为 0–3000 毫秒")
	}
	seen := map[Binding]bool{}
	for _, b := range s.Hotkeys {
		if b.Modifiers == 0 || b.KeyCode > 127 || seen[b] {
			return fmt.Errorf("快捷键无效或重复")
		}
		seen[b] = true
	}
	return nil
}
