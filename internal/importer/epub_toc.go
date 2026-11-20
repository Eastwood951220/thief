package importer

import (
	"bytes"
	"encoding/xml"
	"golang.org/x/net/html"
	"strings"
)

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}
func addTOC(toc map[string]map[string]string, base, href, title string) {
	p, f, e := resolve(base, href)
	if e != nil {
		return
	}
	if toc[p] == nil {
		toc[p] = map[string]string{}
	}
	if toc[p][f] == "" {
		toc[p][f] = strings.TrimSpace(title)
	}
}
func readNav(raw []byte, base string, toc map[string]map[string]string) {
	doc, e := html.Parse(bytes.NewReader(raw))
	if e != nil {
		return
	}
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inNav bool) {
		if n.Type == html.ElementNode && n.Data == "nav" {
			kind := attr(n, "epub:type")
			if kind == "" {
				kind = attr(n, "type")
			}
			if kind != "" && !strings.Contains(kind, "toc") {
				return
			}
			inNav = true
		}
		if inNav && n.Type == html.ElementNode && n.Data == "a" {
			addTOC(toc, base, attr(n, "href"), nodeText(n))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inNav)
		}
	}
	walk(doc, false)
}

type ncxPoint struct {
	Text    string `xml:"navLabel>text"`
	Content struct {
		Source string `xml:"src,attr"`
	} `xml:"content"`
	Children []ncxPoint `xml:"navPoint"`
}

func readNCX(raw []byte, base string, toc map[string]map[string]string) {
	var n struct {
		Points []ncxPoint `xml:"navMap>navPoint"`
	}
	if xml.Unmarshal(raw, &n) != nil {
		return
	}
	var walk func([]ncxPoint)
	walk = func(points []ncxPoint) {
		for _, p := range points {
			addTOC(toc, base, p.Content.Source, p.Text)
			walk(p.Children)
		}
	}
	walk(n.Points)
}
