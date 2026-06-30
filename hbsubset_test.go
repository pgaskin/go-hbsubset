package hbsubset

import (
	"testing"
)

// TODO: test subsetting, reproducibility, error handling, etc

func TestTag(t *testing.T) {
	for _, tc := range []struct {
		s   string
		tag uint32
	}{
		{"glyf", 0x676c7966},
		{"OS/2", 0x4f532f32},
		{"a", 0x61202020},       // space-padded
		{"", 0x20202020},        // all spaces
		{"toolong", 0x746f6f6c}, // truncated to 4
	} {
		if got := tagValue(MakeTag(tc.s)); got != tc.tag {
			t.Errorf("tagValue(MakeTag(%q)) = %#x, want %#x", tc.s, uint32(got), uint32(tc.tag))
		}
	}
	if got := MakeTag("wght").String(); got != "wght" {
		t.Errorf("String = %q", got)
	}
	if got := MakeTag("a").String(); got != "a" {
		t.Errorf("String trim = %q, want %q", got, "a")
	}
}
