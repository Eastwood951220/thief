package importer

import (
	"strings"
	"testing"
)

func TestTXTHeadings(t *testing.T) {
	cases := []struct {
		name, text string
		titles     []string
		body       string
	}{
		{"chapters", "第一章 开始\n正文。\n第二章 继续\n结束。", []string{"第一章 开始", "第二章 继续"}, "正文。"},
		{"body", "第一章 开始\n第一节课上完，休息间隙很长。\n第二节课开始。", []string{"第一章 开始"}, "第一节课上完，休息间隙很长。\n\n第二节课开始。"},
		{"quote", "第一章 开始\n第1章 “便是改革天皇制度”\n正文继续。", []string{"第一章 开始"}, "第1章 “便是改革天皇制度”\n\n正文继续。"},
		{"intro", "前言。\n\n第一卷 少年\n第壹章 开端\n甲。\n第贰拾叁回 归途\n乙。", []string{"前言", "第壹章 开端", "第贰拾叁回 归途"}, "前言。"},
		{"numbered", "第1卷\n1、起点\n甲。\n2、归途\n乙。", []string{"1、起点", "2、归途"}, "甲。"},
		{"english", "Chapter IV: Night\nHello.\n\nChapter 5 Road\nWorld.", []string{"Chapter IV: Night", "Chapter 5 Road"}, "Hello."},
		{"wrapped", "第一章 开始\n这是一个被窄屏\n硬折行拆开的句子。\n\n下一段。", []string{"第一章 开始"}, "这是一个被窄屏硬折行拆开的句子。\n\n下一段。"},
		{"plain", "没有章节的正文", []string{"书名"}, "没有章节的正文"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := ParseTXT(tc.text, "书名", Options{})
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Chapters) != len(tc.titles) {
				t.Fatalf("titles: %#v", b.Chapters)
			}
			for i, title := range tc.titles {
				if b.Chapters[i].Title != title {
					t.Fatalf("title %q want %q", b.Chapters[i].Title, title)
				}
			}
			if b.Chapters[0].Text != tc.body {
				t.Fatalf("body %q want %q", b.Chapters[0].Text, tc.body)
			}
		})
	}
}
func TestTXTCustomAndLongLine(t *testing.T) {
	b, e := ParseTXT("卷一\n第001话 起点\n正文。\n第002话 远行\n结束。", "书", Options{ChapterPattern: `^第\d+话`, VolumePattern: `^卷[一二]$`})
	if e != nil || len(b.Chapters) != 2 || b.Chapters[1].Volume != "卷一" {
		t.Fatalf("%+v %v", b, e)
	}
	for _, p := range []string{"[", ".*"} {
		if _, e := ParseTXT("正文", "书", Options{ChapterPattern: p}); e == nil {
			t.Fatal("accepted invalid pattern", p)
		}
	}
	b, e = ParseTXT(strings.Repeat("长", 100000), "书", Options{})
	if e != nil || len([]rune(b.Chapters[0].Text)) != 100000 {
		t.Fatal("long line lost")
	}
	if _, e := ParseTXT(" \n\t", "书", Options{}); e == nil {
		t.Fatal("empty accepted")
	}
}
func TestDecode(t *testing.T) {
	for _, data := range [][]byte{[]byte("中文"), {0xef, 0xbb, 0xbf, 0xe4, 0xb8, 0xad, 0xe6, 0x96, 0x87}, {0xff, 0xfe, 0x2d, 0x4e, 0x87, 0x65}, {0xfe, 0xff, 0x4e, 0x2d, 0x65, 0x87}, {0xd6, 0xd0, 0xce, 0xc4}} {
		s, e := Decode(data, "")
		if e != nil || s != "中文" {
			t.Fatalf("%x: %q %v", data, s, e)
		}
	}
	if _, e := Decode([]byte{0xff, 0xfe, 0x2d}, ""); e == nil {
		t.Fatal("truncated UTF16 accepted")
	}
	if _, e := Decode([]byte{0x81}, "GB18030"); e == nil {
		t.Fatal("truncated GB accepted")
	}
}
