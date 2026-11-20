package importer

import (
	"bytes"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	encunicode "golang.org/x/text/encoding/unicode"
	"strings"
	"unicode/utf8"
)

func Decode(data []byte, encoding string) (string, error) {
	name := strings.ToUpper(encoding)
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		data = data[3:]
		name = "UTF-8"
	}
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) {
		data = data[2:]
		name = "UTF-16LE"
	} else if bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		data = data[2:]
		name = "UTF-16BE"
	}
	if name == "" || name == "AUTO" {
		if utf8.Valid(data) {
			name = "UTF-8"
		} else {
			name = "GB18030"
		}
	}
	var out []byte
	var err error
	switch name {
	case "UTF-8":
		if !utf8.Valid(data) {
			return "", fmt.Errorf("无效 UTF-8，请选择文件编码")
		}
		return string(data), nil
	case "UTF-16LE", "UTF-16BE":
		if len(data)%2 != 0 {
			return "", fmt.Errorf("UTF-16 数据不完整")
		}
		order := encunicode.LittleEndian
		if name == "UTF-16BE" {
			order = encunicode.BigEndian
		}
		out, err = encunicode.UTF16(order, encunicode.IgnoreBOM).NewDecoder().Bytes(data)
	case "GB18030", "GBK":
		out, err = simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	default:
		return "", fmt.Errorf("不支持编码 %s", encoding)
	}
	if err != nil || bytes.Contains(out, []byte("\ufffd")) {
		return "", fmt.Errorf("编码解码失败，请手动选择文件编码")
	}
	return string(out), nil
}
