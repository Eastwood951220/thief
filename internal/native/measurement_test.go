package native

import (
	"testing"
	"thief/internal/reader"
)

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
