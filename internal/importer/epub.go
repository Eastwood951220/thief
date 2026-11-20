package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"thief/internal/book"
)

type epubItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	Properties string `xml:"properties,attr"`
	Media      string `xml:"media-type,attr"`
}
type epubPackage struct {
	Title string     `xml:"metadata>title"`
	Items []epubItem `xml:"manifest>item"`
	Spine []struct {
		ID string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}
type archive struct {
	files map[string]*zip.File
	total int64
}

func (a *archive) read(name string) ([]byte, error) {
	f := a.files[name]
	if f == nil {
		return nil, fmt.Errorf("EPUB 缺少资源 %s", name)
	}
	if f.UncompressedSize64 > 32<<20 {
		return nil, fmt.Errorf("EPUB 单资源过大")
	}
	r, e := f.Open()
	if e != nil {
		return nil, e
	}
	defer r.Close()
	b, e := io.ReadAll(io.LimitReader(r, 32<<20+1))
	a.total += int64(len(b))
	if len(b) > 32<<20 || a.total > 256<<20 {
		return nil, fmt.Errorf("EPUB 展开内容超过限制")
	}
	return b, e
}
func resolve(base, href string) (string, string, error) {
	u, e := url.Parse(href)
	if e != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") {
		return "", "", fmt.Errorf("EPUB 路径无效: %s", href)
	}
	p := path.Clean(path.Join(path.Dir(base), u.Path))
	if p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, "\\") {
		return "", "", fmt.Errorf("EPUB 路径越界")
	}
	return p, u.Fragment, nil
}
func ParseEPUB(ctx context.Context, data []byte, title string) (book.Book, error) {
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return book.Book{}, fmt.Errorf("EPUB 文件损坏: %w", e)
	}
	if len(z.File) > 20000 {
		return book.Book{}, fmt.Errorf("EPUB 条目过多")
	}
	a := archive{files: map[string]*zip.File{}}
	for _, f := range z.File {
		n := path.Clean(f.Name)
		if strings.HasPrefix(f.Name, "/") || n == ".." || strings.HasPrefix(n, "../") || strings.Contains(f.Name, "\\") {
			return book.Book{}, fmt.Errorf("EPUB 路径越界")
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if a.files[n] != nil {
			return book.Book{}, fmt.Errorf("EPUB 重复资源路径")
		}
		a.files[n] = f
	}
	raw, e := a.read("META-INF/container.xml")
	if e != nil {
		return book.Book{}, e
	}
	var container struct {
		Roots []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if e = xml.Unmarshal(raw, &container); e != nil || len(container.Roots) == 0 {
		return book.Book{}, fmt.Errorf("EPUB 容器无效")
	}
	opf, _, e := resolve("root", container.Roots[0].Path)
	if e != nil {
		return book.Book{}, e
	}
	raw, e = a.read(opf)
	if e != nil {
		return book.Book{}, e
	}
	var pkg epubPackage
	if e = xml.Unmarshal(raw, &pkg); e != nil {
		return book.Book{}, e
	}
	items := map[string]epubItem{}
	for _, item := range pkg.Items {
		items[item.ID] = item
	}
	toc := map[string]map[string]string{}
	for _, item := range pkg.Items {
		if strings.Contains(" "+item.Properties+" ", " nav ") || item.Media == "application/x-dtbncx+xml" {
			p, _, err := resolve(opf, item.Href)
			if err != nil {
				return book.Book{}, err
			}
			raw, err := a.read(p)
			if err != nil {
				return book.Book{}, err
			}
			if item.Media == "application/x-dtbncx+xml" {
				readNCX(raw, p, toc)
			} else {
				readNav(raw, p, toc)
			}
		}
	}
	encrypted := map[string]bool{}
	if a.files["META-INF/encryption.xml"] != nil {
		raw, e = a.read("META-INF/encryption.xml")
		if e != nil {
			return book.Book{}, e
		}
		dec := xml.NewDecoder(bytes.NewReader(raw))
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return book.Book{}, err
			}
			if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "CipherReference" {
				for _, at := range start.Attr {
					if at.Name.Local == "URI" {
						p, _, er := resolve("root", at.Value)
						if er != nil {
							return book.Book{}, er
						}
						encrypted[p] = true
					}
				}
			}
		}
	}
	b := book.Book{Title: strings.TrimSpace(pkg.Title), Format: "EPUB"}
	if b.Title == "" {
		b.Title = title
	}
	for _, ref := range pkg.Spine {
		if e = ctx.Err(); e != nil {
			return book.Book{}, e
		}
		item, ok := items[ref.ID]
		if !ok {
			return book.Book{}, fmt.Errorf("EPUB spine 引用不存在")
		}
		p, _, er := resolve(opf, item.Href)
		if er != nil {
			return book.Book{}, er
		}
		if encrypted[p] {
			return book.Book{}, fmt.Errorf("EPUB 正文已加密，不支持 DRM")
		}
		raw, e = a.read(p)
		if e != nil {
			return book.Book{}, e
		}
		chapters, er := extractEPUBText(raw, toc[p])
		if er != nil {
			return book.Book{}, er
		}
		for _, c := range chapters {
			c.ID = fmt.Sprintf("chapter-%d", len(b.Chapters)+1)
			if c.Title == "" {
				c.Title = fmt.Sprintf("章节 %d", len(b.Chapters)+1)
			}
			b.Chapters = append(b.Chapters, c)
		}
	}
	if len(b.Chapters) == 0 {
		return book.Book{}, fmt.Errorf("EPUB 没有可读文字")
	}
	return b, nil
}
