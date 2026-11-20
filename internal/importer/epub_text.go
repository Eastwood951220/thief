package importer

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
	"thief/internal/book"
)

func extractEPUBText(raw []byte, toc map[string]string) ([]book.Chapter, error) {
	doc, e := html.Parse(bytes.NewReader(raw))
	if e != nil {
		return nil, e
	}
	var out []book.Chapter
	current := book.Chapter{Title: toc[""]}
	var paras []string
	var pending strings.Builder
	flushP := func() {
		s := strings.TrimSpace(pending.String())
		pending.Reset()
		if s != "" {
			paras = append(paras, s)
		}
	}
	flushC := func() {
		flushP()
		if len(paras) > 0 {
			current.Text = strings.Join(paras, "\n\n")
			out = append(out, current)
		}
		paras = nil
	}
	begin := func(title string) { flushC(); current = book.Chapter{Title: title} }
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			if pending.Len() == 0 && strings.TrimSpace(n.Data) == "" {
				return
			}
			text := n.Data
			if strings.TrimSpace(text) == "" {
				if pending.Len() > 0 {
					pending.WriteByte(' ')
				}
				return
			}
			pending.WriteString(strings.Join(strings.Fields(text), " "))
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "head", "script", "style", "img", "svg", "noscript", "nav":
				return
			}
			if title := toc[attr(n, "id")]; attr(n, "id") != "" && title != "" {
				begin(title)
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				title := nodeText(n)
				if title != "" {
					if current.Title == title && len(paras) == 0 && pending.Len() == 0 {
					} else {
						begin(title)
					}
				}
				return
			}
			if n.Data == "br" {
				pending.WriteByte('\n')
				return
			}
			if strings.Contains(" p div section article blockquote li pre ", " "+n.Data+" ") {
				flushP()
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				flushP()
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	flushC()
	return out, nil
}
