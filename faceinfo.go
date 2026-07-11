package hbsubset

import "encoding/binary"

// FaceCount returns the number of faces in a font collection. If the font is
// not a collection, the count will be 1. If the data is not a font, the count
// will be 0. An error will only be returned if the wasm memory allocation
// fails.
func FaceCount(data []byte) (int, error) {
	x := newInstance()
	p, err := x.copy(data)
	if err != nil {
		return 0, err
	}
	defer x.free(p)
	return int(x.m.Xhbw_face_count(int32(p), int32(uint32(len(data))))), nil
}

// GlyphCount returns the number of glyphs in the face.
func (f *Face) GlyphCount() uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return uint32(f.x.m.Xhb_face_get_glyph_count(int32(f.face)))
}

// UnitsPerEm returns the design units per em of the face.
func (f *Face) UnitsPerEm() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int(uint32(f.x.m.Xhb_face_get_upem(int32(f.face))))
}

// Index returns the face's index within its font collection.
func (f *Face) Index() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int(uint32(f.x.m.Xhb_face_get_index(int32(f.face))))
}

// Tables returns the sfnt table tags present in the face, in font order.
func (f *Face) Tables() []Tag {
	f.mu.Lock()
	defer f.mu.Unlock()
	const batch = 64
	p, err := f.x.alloc(batch * 4)
	if err != nil {
		return nil
	}
	defer f.x.free(p)
	var tags []Tag
	for start := uint32(0); ; {
		total := uint32(f.x.m.Xhbw_face_table_tags(int32(f.face), int32(start), int32(p), batch))
		n := min(total-start, batch)
		for i := range n {
			tags = append(tags, valueTag(f.x.u32(p+i*4)))
		}
		start += n
		if start >= total || n == 0 {
			break
		}
	}
	return tags
}

// Table returns the raw bytes of a single sfnt table, or nil if the table does
// not exist.
func (f *Face) Table(tag Tag) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ptr, n, handle := f.x.m.Xhbw_face_table(int32(f.face), int32(tagValue(tag)))
	return f.x.blob(ptr, n, handle)
}

// VariationAxes returns the fvar variation axes of the face and their value
// ranges.
func (f *Face) VariationAxes() map[Tag]AxisRange {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := uint32(f.x.m.Xhb_ot_var_get_axis_count(int32(f.face)))
	if total == 0 {
		return nil
	}
	const size = 32 // sizeof(hb_ot_var_axis_info_t) (TODO: codegen?)
	countPtr, err := f.x.alloc(4)
	if err != nil {
		return nil
	}
	defer f.x.free(countPtr)
	infoPtr, err := f.x.alloc(int(total) * size)
	if err != nil {
		return nil
	}
	defer f.x.free(infoPtr)
	f.x.putU32(countPtr, total)
	f.x.m.Xhb_ot_var_get_axis_infos(int32(f.face), 0, int32(countPtr), int32(infoPtr))
	axes := make(map[Tag]AxisRange, total)
	for i := range min(f.x.u32(countPtr), total) {
		p := infoPtr + i*size
		axes[valueTag(f.x.u32(p+4))] = AxisRange{
			Min:     f.x.f32(p + 16),
			Default: f.x.f32(p + 20),
			Max:     f.x.f32(p + 24),
		}
	}
	return axes
}

// GlyphMapping returns the nominal cmap mapping from unicode codepoints to
// glyph indices.
func (f *Face) GlyphMapping() map[rune]uint32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := uint32(f.x.m.Xhb_map_create())
	defer f.x.m.Xhb_map_destroy(int32(m))
	f.x.m.Xhb_face_collect_nominal_glyph_mapping(int32(f.face), int32(m), 0)
	pop := uint32(f.x.m.Xhb_map_get_population(int32(m)))
	if pop == 0 {
		return nil
	}
	keys, err := f.x.alloc(int(pop) * 4)
	if err != nil {
		return nil
	}
	defer f.x.free(keys)
	vals, err := f.x.alloc(int(pop) * 4)
	if err != nil {
		return nil
	}
	defer f.x.free(vals)
	got := uint32(f.x.m.Xhbw_map_entries(int32(m), int32(keys), int32(vals), int32(pop)))
	mapping := make(map[rune]uint32, got)
	for i := range got {
		mapping[rune(f.x.u32(keys+i*4))] = f.x.u32(vals + i*4)
	}
	return mapping
}

// VariationSelectors returns the unicode variation selectors in the face's
// cmap, in ascending order. Note that variation selectors are a completely
// different concept from variation axes.
func (f *Face) VariationSelectors() []rune {
	return f.collectUnicodes(func(set uint32) {
		f.x.m.Xhb_face_collect_variation_selectors(int32(f.face), int32(set))
	})
}

// VariationUnicodes returns the unicode codepoints which the face's cmap maps
// to a variant glyph under the given variation selector, in ascending order.
func (f *Face) VariationUnicodes(selector rune) []rune {
	return f.collectUnicodes(func(set uint32) {
		f.x.m.Xhb_face_collect_variation_unicodes(int32(f.face), int32(uint32(selector)), int32(set))
	})
}

func (f *Face) collectUnicodes(fill func(set uint32)) []rune {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := uint32(f.x.m.Xhb_set_create())
	defer f.x.m.Xhb_set_destroy(int32(s))
	fill(s)
	pop := uint32(f.x.m.Xhb_set_get_population(int32(s)))
	if pop == 0 {
		return nil
	}
	buf, err := f.x.alloc(int(pop) * 4)
	if err != nil {
		return nil
	}
	defer f.x.free(buf)
	got := uint32(f.x.m.Xhbw_set_values(int32(s), int32(buf), int32(pop)))
	runes := make([]rune, got)
	for i := range got {
		runes[i] = rune(f.x.u32(buf + i*4))
	}
	return runes
}

// NameEntry identifies a name-table entry by name ID and language.
type NameEntry struct {
	// ID is the OpenType name ID (e.g. 1 is the font family name).
	ID NameID

	// Language is the BCP 47 language tag of the entry (e.g. "en"), for use
	// with [Face.Name].
	Language string

	// LanguageID is the raw name-record language ID of the entry, as used by
	// [Options.NameLanguages]. It is zero if the record could not be resolved.
	LanguageID uint32
}

// Names returns the name-table entries of the face, in font order.
func (f *Face) Names() []NameEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	countPtr, err := f.x.alloc(4)
	if err != nil {
		return nil
	}
	defer f.x.free(countPtr)
	arr := uint32(f.x.m.Xhb_ot_name_list_names(int32(f.face), int32(countPtr)))
	n := f.x.u32(countPtr)
	if arr == 0 || n == 0 {
		return nil
	}
	const size = 12 // sizeof(hb_ot_name_entry_t)
	names := make([]NameEntry, n)
	records := make([]uint32, n)
	for i := range n {
		p := arr + i*size
		names[i] = NameEntry{
			ID:       NameID(f.x.u32(p)),
			Language: f.x.cstr(uint32(f.x.m.Xhb_language_to_string(int32(f.x.u32(p + 8))))),
		}
		// harfbuzz keeps the index of the record within the font's name table
		// in the private var field (entry_index in OT/name/name.hh); the
		// pinned harfbuzz commit must be checked when updating
		records[i] = uint32(f.x.u16(p + 6))
	}
	// resolve the raw language IDs from the name table records, which harfbuzz
	// does not expose
	if tbl, err := f.x.blob(f.x.m.Xhbw_face_table(int32(f.face), int32(tagValue(MakeTag("name"))))); err == nil {
		for i, rec := range records {
			if off := 6 + int(rec)*12; off+12 <= len(tbl) {
				names[i].LanguageID = uint32(binary.BigEndian.Uint16(tbl[off+4:]))
			}
		}
	}
	return names
}

// Name returns a name-table entry as UTF-8. The language is a BCP 47 tag from
// [Face.NameIDs], or empty for English. It returns false if the face has no
// matching non-empty name.
func (f *Face) Name(nameID uint32, language string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	lp, err := f.x.copy([]byte(language))
	if err != nil {
		return "", false
	}
	lang := uint32(f.x.m.Xhb_language_from_string(int32(lp), int32(uint32(len(language)))))
	f.x.free(lp)
	full := uint32(f.x.m.Xhb_ot_name_get_utf8(int32(f.face), int32(nameID), int32(lang), 0, 0))
	if full == 0 {
		return "", false
	}
	sizePtr, err := f.x.alloc(4)
	if err != nil {
		return "", false
	}
	defer f.x.free(sizePtr)
	buf, err := f.x.alloc(int(full) + 1)
	if err != nil {
		return "", false
	}
	defer f.x.free(buf)
	f.x.putU32(sizePtr, full+1)
	f.x.m.Xhb_ot_name_get_utf8(int32(f.face), int32(nameID), int32(lang), int32(sizePtr), int32(buf))
	b, err := f.x.mem(buf, min(f.x.u32(sizePtr), full))
	if err != nil {
		return "", false
	}
	return string(b), true
}
