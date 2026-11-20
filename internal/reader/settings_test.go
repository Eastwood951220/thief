package reader

import (
	"math"
	"testing"
)

func TestSettings(t *testing.T) {
	s := DefaultSettings()
	if e := s.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, color := range []string{"#000000", "#aBcDeF", "#FFFFFF"} {
		s.TextColor = color
		if e := s.Validate(); e != nil {
			t.Fatal(e)
		}
	}
	s.TextColor = "#GG0000"
	if s.Validate() == nil {
		t.Fatal("bad color accepted")
	}
	s = DefaultSettings()
	s.FontSize = math.NaN()
	if s.Validate() == nil {
		t.Fatal("NaN accepted")
	}
	s = DefaultSettings()
	s.BackgroundTransparency = 100
	s.TextTransparency = 0
	if e := s.Validate(); e != nil {
		t.Fatal(e)
	}
	s.Frame.Width = 0
	if s.Validate() == nil {
		t.Fatal("zero area accepted")
	}
}
