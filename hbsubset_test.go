package hbsubset

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TODO: test subsetting, reproducibility, error handling, etc

var testFonts = []string{
	"SourceSans3VF-Upright.ttf",
	"SourceSans3VF-Upright.otf",
}

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

func TestFaceMetadata(t *testing.T) {
	for _, name := range testFonts {
		t.Run(name, func(t *testing.T) {
			data := readFont(t, name)

			if n, err := FaceCount(data); err != nil || n != 1 {
				t.Fatalf("FaceCount = %d, %v; want 1, nil", n, err)
			}

			face, err := NewFace(data, 0)
			if err != nil {
				t.Fatal(err)
			}

			if got := face.GlyphCount(); got < 100 {
				t.Errorf("GlyphCount = %d, want a real font", got)
			}
			if got := face.UnitsPerEm(); got != 1000 {
				t.Errorf("UnitsPerEm = %d, want 1000", got)
			}
			if got := face.Index(); got != 0 {
				t.Errorf("Index = %d, want 0", got)
			}

			tags := face.Tables()
			if !slices.Contains(tags, MakeTag("head")) || !slices.Contains(tags, MakeTag("cmap")) {
				t.Errorf("Tables = %v, missing head/cmap", tags)
			}

			head, err := face.Table(MakeTag("head"))
			if err != nil {
				t.Fatalf("Table(head): %v", err)
			}
			if len(head) != 54 {
				t.Errorf("head table length = %d, want 54", len(head))
			}

			if missing, err := face.Table(MakeTag("ZZZZ")); err != nil || missing != nil {
				t.Errorf("Table(ZZZZ) = %v, %v; want nil, nil", missing, err)
			}
		})
	}
}

func TestVariationAxes(t *testing.T) {
	for _, name := range testFonts {
		t.Run(name, func(t *testing.T) {
			face, err := NewFace(readFont(t, name), 0)
			if err != nil {
				t.Fatal(err)
			}
			axes := face.VariationAxes()
			wght, ok := axes[MakeTag("wght")]
			if !ok {
				t.Fatalf("VariationAxes = %v, missing wght", axes)
			}
			if wght.Min >= wght.Max || wght.Default < wght.Min || wght.Default > wght.Max {
				t.Errorf("wght range = %+v, want a real range", wght)
			}
		})
	}
}

func TestNominalGlyphMapping(t *testing.T) {
	face, err := NewFace(readFont(t, "SourceSans3VF-Upright.ttf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	mapping := face.GlyphMapping()
	if mapping['A'] == 0 {
		t.Errorf("GlyphMapping is missing 'A' (got %d runes)", len(mapping))
	}
}

func TestVariationSelectors(t *testing.T) {
	face, err := NewFace(readFont(t, "SourceSans3VF-Upright.ttf"), 0)
	if err != nil {
		t.Fatal(err)
	}
	// note: Source Sans has no format 14 cmap subtable
	for _, vs := range face.VariationSelectors() {
		if vs < 0x180B || vs > 0xE01EF {
			t.Errorf("VariationSelectors returned %#x, not a variation selector", vs)
		}
		face.VariationUnicodes(vs)
	}
	for _, r := range face.VariationUnicodes(0xFE0F) {
		t.Errorf("VariationUnicodes(U+FE0F) returned %#x, want none", r)
	}
}

func TestNames(t *testing.T) {
	data := readFont(t, "SourceSans3VF-Upright.ttf")
	face, err := NewFace(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	names := face.Names()
	if len(names) == 0 {
		t.Fatal("NameIDs returned nothing")
	}
	family := NameEntry{ID: 1, Language: "en", LanguageID: 1033}
	if !slices.Contains(names, family) {
		t.Fatalf("NameIDs = %v, missing en family name %v", names, family)
	}

	for _, lang := range []string{"", "en"} {
		if got, ok := face.Name(1, lang); !ok || !strings.Contains(got, "SourceSans") {
			t.Errorf("Name(1, %q) = %q, %v, want the family name", lang, got, ok)
		}
	}
	if got, ok := face.Name(0xFFFE, ""); ok {
		t.Errorf("Name(0xFFFE, \"\") = %q, want no match", got)
	}
}

func readFont(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read font: %v", err)
	}
	return data
}
