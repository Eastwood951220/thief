package importer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"thief/internal/book"
)

func Parse(ctx context.Context, path string, opts Options) (book.Book, error) {
	if e := ctx.Err(); e != nil {
		return book.Book{}, e
	}
	f, e := os.Open(path)
	if e != nil {
		return book.Book{}, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, 256<<20+1))
	if e != nil {
		return book.Book{}, e
	}
	if len(data) > 256<<20 {
		return book.Book{}, fmt.Errorf("文件超过 256 MiB")
	}
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var b book.Book
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt":
		var text string
		text, e = Decode(data, opts.Encoding)
		if e == nil {
			b, e = ParseTXT(text, title, opts)
		}
	case ".epub":
		b, e = ParseEPUB(ctx, data, title)
	default:
		e = fmt.Errorf("请选择 TXT 或 EPUB 文件")
	}
	if e != nil {
		return b, e
	}
	if e = ctx.Err(); e != nil {
		return book.Book{}, e
	}
	b.ID = fmt.Sprintf("%x", sha256.Sum256(data))
	return b, nil
}
