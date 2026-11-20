package native

import (
	"strings"
	"testing"
	"thief/internal/reader"
)

func TestNativePageDoesNotIncludeClippedParagraph(t *testing.T) {
	text := "\n山风穿过竹林，远处的灯火刚刚亮起。他把信折好，收进衣袋，沿着石阶慢慢向前走去。\n\n转过山口，一座小小的茶亭出现在路旁。炉上的水已经沸了，白汽消散在暮色里。\n\n老人放下茶碗，指了指山下的道路。"
	n, e := (Measurer{}).Fit(text, reader.Layout{Width: 315, Height: 156, FontSize: 16, LineSpacing: 4})
	if e != nil {
		t.Fatal(e)
	}
	boundary := len([]rune(strings.Split(text, "老人")[0]))
	if n > boundary {
		t.Fatalf("included clipped third paragraph: end %d, boundary %d", n, boundary)
	}
}

func TestNativeMeasurement(t *testing.T) {
	m := Measurer{}
	text := "甲😀é👨‍👩‍👧‍👦乙"
	l := reader.Layout{Width: 800, Height: 100, FontSize: 16}
	n, e := m.Fit(text, l)
	if e != nil || n != len([]rune(text)) {
		t.Fatal(n, e)
	}
	n, e = m.Fit(text, reader.Layout{Width: 40, Height: 30, FontSize: 16, SingleLine: true})
	if e != nil || n <= 0 || n >= len([]rune(text)) {
		t.Fatal(n, e)
	}
	if n == 3 || n > 4 && n < 11 {
		t.Fatal("split composed character", n)
	}
}

func TestNativeMeasurementPreservesEmbeddedNUL(t *testing.T) {
	n, e := (Measurer{}).Fit("甲\x00乙", reader.Layout{Width: 400, Height: 100, FontSize: 16})
	if e != nil || n != 3 {
		t.Fatalf("truncated at NUL: %d %v", n, e)
	}
}
