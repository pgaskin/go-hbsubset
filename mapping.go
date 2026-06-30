package hbsubset

import "iter"

// Mapping describes renumbered glyphs.
type Mapping struct {
	oldToNew map[uint32]uint32
	newToOld map[uint32]uint32
	uniToOld map[uint32]uint32
}

// NewGlyph returns the subset glyph id corresponding to a source glyph id,
// reporting whether that source glyph was retained.
func (m Mapping) NewGlyph(oldGID uint32) (uint32, bool) {
	v, ok := m.oldToNew[oldGID]
	return v, ok
}

// OldGlyph returns the source glyph id corresponding to a subset glyph id,
// reporting whether such a glyph exists in the subset.
func (m Mapping) OldGlyph(newGID uint32) (uint32, bool) {
	v, ok := m.newToOld[newGID]
	return v, ok
}

// RuneGlyph returns the source glyph id that a unicode codepoint maps to,
// reporting whether the codepoint was retained. Combine it with
// [Mapping.NewGlyph] to obtain the codepoint's glyph id in the subset.
func (m Mapping) RuneGlyph(r rune) (uint32, bool) {
	v, ok := m.uniToOld[uint32(r)]
	return v, ok
}

// NumGlyphs returns the number of glyphs retained in the subset.
func (m Mapping) NumGlyphs() int {
	return len(m.oldToNew)
}

// Glyphs iterates over the retained glyphs as (source glyph id, subset glyph
// id) pairs, in an arbitrary order.
func (m Mapping) Glyphs() iter.Seq2[uint32, uint32] {
	return func(yield func(uint32, uint32) bool) {
		for oldGID, newGID := range m.oldToNew {
			if !yield(oldGID, newGID) {
				return
			}
		}
	}
}

// Runes iterates over the retained unicode codepoints as (codepoint, source
// glyph id) pairs, in an arbitrary order.
func (m Mapping) Runes() iter.Seq2[rune, uint32] {
	return func(yield func(rune, uint32) bool) {
		for cp, old := range m.uniToOld {
			if !yield(rune(cp), old) {
				return
			}
		}
	}
}
