package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"
)

func epubFixture(extra map[string]string) []byte {
	files := map[string]string{"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/book.opf"/></rootfiles></container>`, "OEBPS/book.opf": `<package><metadata><title>测试</title></metadata><manifest><item id="b" href="b.xhtml" media-type="application/xhtml+xml"/><item id="a" href="a.xhtml" media-type="application/xhtml+xml"/><item id="nav" href="nav.xhtml" properties="nav"/></manifest><spine><itemref idref="b"/><itemref idref="a"/></spine></package>`, "OEBPS/b.xhtml": `<html><body><h1 id="one">第一章</h1><p>甲 &amp; 乙。</p><img src="x"/><style>hidden</style><h1 id="two">第二章</h1><p>丙<br/>丁。</p></body></html>`, "OEBPS/a.xhtml": `<html><body><h1>第三章</h1><p>末尾。</p></body></html>`, "OEBPS/nav.xhtml": `<html><body><nav epub:type="toc"><ol><li><a href="b.xhtml#one">第一章</a></li><li><a href="b.xhtml#two">第二章</a></li><li><a href="a.xhtml">第三章</a></li></ol></nav></body></html>`}
	for k, v := range extra {
		files[k] = v
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for k, v := range files {
		f, _ := w.Create(k)
		f.Write([]byte(v))
	}
	w.Close()
	return buf.Bytes()
}
func TestEPUBSpineAndTOC(t *testing.T) {
	b, e := ParseEPUB(context.Background(), epubFixture(nil), "fallback")
	if e != nil {
		t.Fatal(e)
	}
	if len(b.Chapters) != 3 {
		t.Fatalf("%+v", b)
	}
	if b.Chapters[0].Title != "第一章" || b.Chapters[0].Text != "甲 & 乙。" || b.Chapters[1].Text != "丙\n丁。" || b.Chapters[2].Text != "末尾。" {
		t.Fatalf("%+v", b.Chapters)
	}
}
func TestEPUBPlainText(t *testing.T) {
	b, e := ParseEPUB(context.Background(), epubFixture(map[string]string{"OEBPS/nav.xhtml": "<html></html>"}), "fallback")
	if e != nil || len(b.Chapters) != 3 {
		t.Fatal(b, e)
	}
	for _, c := range b.Chapters {
		if strings.Contains(c.Text, "hidden") || strings.Contains(c.Text, "<") {
			t.Fatal(c)
		}
	}
}
func TestEPUBInvalid(t *testing.T) {
	for _, data := range [][]byte{[]byte("invalid"), epubFixture(map[string]string{"OEBPS/a.xhtml": "<html><body><img src='x'/></body></html>", "OEBPS/b.xhtml": "<html><body><img src='x'/></body></html>"}), epubFixture(map[string]string{"../bad": "escape"}), epubFixture(map[string]string{"META-INF/encryption.xml": `<encryption><EncryptedData><CipherData><CipherReference URI="OEBPS/b.xhtml"/></CipherData></EncryptedData></encryption>`})} {
		if _, e := ParseEPUB(context.Background(), data, "book"); e == nil {
			t.Fatal("invalid EPUB accepted")
		}
	}
}
