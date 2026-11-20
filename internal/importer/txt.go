package importer

import (
	"fmt"
	"regexp"
	"strings"
	"thief/internal/book"
	"unicode"
	"unicode/utf8"
)

type Options struct{ Encoding, ChapterPattern, VolumePattern string }

const numerals = "零〇一二三四五六七八九十百千万两壹贰叁肆伍陆柒捌玖拾佰仟萬"
const endings = "。！？!?…～：:“”‘’\"'「」『』（）()【】《》〈〉."

var english = regexp.MustCompile(`(?i)^chapter\s+([0-9]+|[ivxlcdm]+)([\s:.]|$)`)
var numbered = regexp.MustCompile(`^[0-9]{1,4}、\s*\S`)

func compilePattern(p string) (*regexp.Regexp, error) {
	if strings.TrimSpace(p) == "" {
		return nil, nil
	}
	r, e := regexp.Compile(p)
	if e != nil {
		return nil, fmt.Errorf("章节正则无效: %w", e)
	}
	if r.MatchString("") {
		return nil, fmt.Errorf("正则不能匹配空行")
	}
	return r, nil
}
func headingUnit(s string) rune {
	r := []rune(strings.TrimSpace(s))
	if len(r) < 3 || len(r) > 40 || r[0] != '第' || strings.ContainsRune("。；;，,、", r[len(r)-1]) {
		return 0
	}
	i := 1
	for i < len(r) && ((r[i] >= '0' && r[i] <= '9') || strings.ContainsRune(numerals, r[i])) {
		i++
	}
	if i == 1 || i >= len(r) || !strings.ContainsRune("章节回集卷部篇", r[i]) {
		return 0
	}
	unit := r[i]
	tail := r[i+1:]
	trim := strings.TrimSpace(string(tail))
	if trim != "" && strings.ContainsRune("“「『\"", []rune(trim)[0]) {
		return 0
	}
	if len(tail) == 0 {
		return unit
	}
	if unicode.IsSpace(tail[0]) || strings.ContainsRune("、，,：:·•.-—–~～「『【(《<", tail[0]) {
		return unit
	}
	if len(tail) <= 12 && !strings.ContainsAny(string(tail), "，。！？；：、,.!?;:") {
		return unit
	}
	return 0
}
func chapterHeading(s string) bool {
	u := headingUnit(s)
	if u != 0 && strings.ContainsRune("章节回集", u) {
		return true
	}
	if utf8.RuneCountInString(s) <= 80 && numbered.MatchString(s) && !strings.ContainsRune("。；;，,", []rune(s)[len([]rune(s))-1]) {
		return true
	}
	if english.MatchString(s) {
		return true
	}
	switch s {
	case "序章", "楔子", "引子", "尾声", "后记", "番外", "终章":
		return true
	}
	return false
}
func ends(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)
	return strings.ContainsRune(endings, r)
}
func ParseTXT(text, title string, opts Options) (book.Book, error) {
	cr, e := compilePattern(opts.ChapterPattern)
	if e != nil {
		return book.Book{}, e
	}
	vr, e := compilePattern(opts.VolumePattern)
	if e != nil {
		return book.Book{}, e
	}
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	b := book.Book{Title: title, Format: "TXT"}
	current := book.Chapter{Title: "前言"}
	var paras []string
	paragraph := ""
	volume := ""
	flushP := func() {
		if paragraph != "" {
			paras = append(paras, paragraph)
			paragraph = ""
		}
	}
	flushC := func() {
		flushP()
		if len(paras) > 0 {
			current.ID = fmt.Sprintf("chapter-%d", len(b.Chapters)+1)
			current.Text = strings.Join(paras, "\n\n")
			b.Chapters = append(b.Chapters, current)
		}
		paras = nil
	}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			flushP()
			continue
		}
		boundary := paragraph == "" || ends(paragraph)
		isVol := false
		isChap := false
		if boundary {
			if vr != nil {
				isVol = vr.MatchString(line)
			} else {
				u := headingUnit(line)
				isVol = u != 0 && strings.ContainsRune("卷部篇", u)
			}
			if cr != nil {
				isChap = cr.MatchString(line)
			} else {
				isChap = chapterHeading(line)
			}
		}
		if isVol || isChap {
			flushC()
			if isVol {
				volume = line
				current = book.Chapter{Title: line, Volume: volume}
			} else {
				current = book.Chapter{Title: line, Volume: volume}
			}
			continue
		}
		if paragraph != "" && ends(paragraph) {
			flushP()
		}
		if paragraph != "" {
			last, _ := utf8.DecodeLastRuneInString(paragraph)
			first, _ := utf8.DecodeRuneInString(line)
			if last < 128 && first < 128 && (unicode.IsLetter(last) || unicode.IsDigit(last)) && (unicode.IsLetter(first) || unicode.IsDigit(first)) {
				paragraph += " "
			}
		}
		paragraph += line
	}
	flushC()
	if len(b.Chapters) == 0 {
		return book.Book{}, fmt.Errorf("TXT 文件没有正文")
	}
	if len(b.Chapters) == 1 && b.Chapters[0].Title == "前言" {
		b.Chapters[0].Title = title
	}
	return b, nil
}
