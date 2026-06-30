package hbsubset

import (
	"encoding/binary"
	"fmt"

	"github.com/pgaskin/go-hbsubset/internal"
)

// Flags is a bitmask of boolean options.
type Flags uint32

const (
	FlagDefault                 = Flags(internal.HB_SUBSET_FLAGS_DEFAULT)                  // empty flags
	FlagNoHinting               = Flags(internal.HB_SUBSET_FLAGS_NO_HINTING)               // drop hinting instructions
	FlagRetainGIDs              = Flags(internal.HB_SUBSET_FLAGS_RETAIN_GIDS)              // make dropped glyphs empty instead of renumbering
	FlagDesubroutinize          = Flags(internal.HB_SUBSET_FLAGS_DESUBROUTINIZE)           // remove subroutines when subsetting a CFF font
	FlagNameLegacy              = Flags(internal.HB_SUBSET_FLAGS_NAME_LEGACY)              // retain non-unicode name records
	FlagSetOverlapsFlag         = Flags(internal.HB_SUBSET_FLAGS_SET_OVERLAPS_FLAG)        // set the OVERLAP_SIMPLE flag on each simple glyph
	FlagPassthroughUnrecognized = Flags(internal.HB_SUBSET_FLAGS_PASSTHROUGH_UNRECOGNIZED) // don't drop unrecognized tables
	FlagNotdefOutline           = Flags(internal.HB_SUBSET_FLAGS_NOTDEF_OUTLINE)           // retain the outline of the .notdef glyph
	FlagGlyphNames              = Flags(internal.HB_SUBSET_FLAGS_GLYPH_NAMES)              // retain PostScript glyph names
	FlagNoPruneUnicodeRanges    = Flags(internal.HB_SUBSET_FLAGS_NO_PRUNE_UNICODE_RANGES)  // prevent OS/2 unicode ranges from being recalculated
	FlagNoLayoutClosure         = Flags(internal.HB_SUBSET_FLAGS_NO_LAYOUT_CLOSURE)        // skip glyph closure over GSUB layout substitution rules
	FlagOptimizeIUPDeltas       = Flags(internal.HB_SUBSET_FLAGS_OPTIMIZE_IUP_DELTAS)      // perform IUP delta optimization on remaining gvar deltas
	FlagNoBidiClosure           = Flags(internal.HB_SUBSET_FLAGS_NO_BIDI_CLOSURE)          // do not pull mirrored versions of input codepoints into the subset
	FlagDowngradeCFF2           = Flags(internal.HB_SUBSET_FLAGS_DOWNGRADE_CFF2)           // convert an instanced CFF2 table to CFF1 for compatibility with older renderers
)

// Tag is a four-byte identifier, for example, a sfnt table tag ("glyf"), a
// layout feature tag ("liga"), or a variation axis tag ("wght").
type Tag [4]byte

// MakeTag builds a Tag from a string. The string is truncated or space-padded
// to exactly four bytes, matching HB_TAG.
func MakeTag(s string) Tag {
	var t Tag
	for i := copy(t[:], s); i < len(t); i++ {
		t[i] = ' '
	}
	return t
}

// String returns the trimmed tag.
func (t Tag) String() string {
	n := len(t)
	for n > 0 && t[n-1] == ' ' {
		n--
	}
	return string(t[:n])
}

func tagValue(t Tag) uint32 {
	return binary.BigEndian.Uint32(t[:])
}

func valueTag(n uint32) Tag {
	var t Tag
	binary.BigEndian.PutUint32(t[:], n)
	return t
}

// RuneRange is an inclusive range of unicode codepoints [Lo, Hi].
type RuneRange struct {
	Lo, Hi rune
}

// AxisRange partially instances a variable font by limiting a variation axis to
// [Min, Max] with the given Default.
type AxisRange struct {
	Min, Max, Default float32
}

// Options is the configuration for a subset plan.
//
// The zero value retains only the minimal structure of the font. Use
// [Options.KeepEverything] to start from a full font and remove from it
// instead.
type Options struct {
	// KeepEverything starts from a configuration that retains the entire font,
	// so that the other fields remove or override from a full font rather than
	// adding to an empty one. Note that it implicitly enables several [Flags].
	KeepEverything bool

	// Flags is a bitmask of boolean options. It is OR'd with any flags implied
	// by [Options.KeepEverything].
	Flags Flags

	// Unicodes adds unicode codepoints to retain glyphs for.
	Unicodes []rune

	// UnicodesRanges adds unicode codepoints to retain glyphs for.
	UnicodeRanges []RuneRange

	// Glyphs are glyph indices to retain in addition to those reached from the
	// requested unicode codepoints.
	Glyphs []uint32

	// LayoutFeatures are the OpenType layout feature tags to retain.
	LayoutFeatures []Tag

	// LayoutScripts are the OpenType script tags to retain. It defaults to all
	// scripts when empty.
	LayoutScripts []Tag

	// NameIDs are the name-table name IDs to retain.
	NameIDs []uint32

	// NameIDs are the name-table language IDs to retain.
	NameLanguages []uint32

	// NoSubsetTables are table tags to pass through unchanged rather than
	// subsetting.
	NoSubsetTables []Tag

	// DropTables are table tags to remove entirely.
	DropTables []Tag

	// GlyphMapping forces specific old-glyph-id to new-glyph-id assignments in
	// the subset. This is applied after [Options.Glyphs].
	GlyphMapping map[uint32]uint32

	// PinAllAxesToDefault pins every axis to its default, fully instancing a
	// variable font.
	PinAllAxesToDefault bool

	// PinAxesToDefault pins the named axes to their default value. This is
	// applied after [Options.PinAllAxesToDefault].
	PinAxesToDefault []Tag

	// PinAxes pins variation axes to specific values. This is applied after
	// [Options.PinAllAxesToDefault] and [Options.PinAxesToDefault].
	PinAxes map[Tag]float32

	// AxisRanges limits variation axes to sub-ranges, partially instancing a
	// variable font. This is applied after
	// [Options.PinAllAxesToDefault], [Options.PinAxesToDefault], and [Options.PinAxes].
	AxisRanges map[Tag]AxisRange
}

func (o *Options) apply(x *instance, input, face uint32) error {
	if o.KeepEverything {
		x.m.Xhb_subset_input_keep_everything(int32(input))
	}
	if o.Flags != 0 {
		cur := Flags(uint32(x.m.Xhb_subset_input_get_flags(int32(input))))
		x.m.Xhb_subset_input_set_flags(int32(input), int32(uint32(cur|o.Flags)))
	}

	if len(o.Unicodes) > 0 || len(o.UnicodeRanges) > 0 {
		s := uint32(x.m.Xhb_subset_input_unicode_set(int32(input)))
		for _, r := range o.Unicodes {
			x.m.Xhb_set_add(int32(s), int32(uint32(r)))
		}
		for _, rr := range o.UnicodeRanges {
			x.m.Xhb_set_add_range(int32(s), int32(uint32(rr.Lo)), int32(uint32(rr.Hi)))
		}
	}
	if len(o.Glyphs) > 0 {
		s := uint32(x.m.Xhb_subset_input_glyph_set(int32(input)))
		for _, g := range o.Glyphs {
			x.m.Xhb_set_add(int32(s), int32(g))
		}
	}

	addTags := func(input uint32, which int, tags []Tag) {
		if len(tags) == 0 {
			return
		}
		s := uint32(x.m.Xhb_subset_input_set(int32(input), int32(which)))
		for _, t := range tags {
			x.m.Xhb_set_add(int32(s), int32(tagValue(t)))
		}
	}

	addValues := func(input uint32, which int, vals []uint32) {
		if len(vals) == 0 {
			return
		}
		s := uint32(x.m.Xhb_subset_input_set(int32(input), int32(which)))
		for _, v := range vals {
			x.m.Xhb_set_add(int32(s), int32(v))
		}
	}

	addTags(input, internal.HB_SUBSET_SETS_LAYOUT_FEATURE_TAG, o.LayoutFeatures)
	addTags(input, internal.HB_SUBSET_SETS_LAYOUT_SCRIPT_TAG, o.LayoutScripts)
	addTags(input, internal.HB_SUBSET_SETS_NO_SUBSET_TABLE_TAG, o.NoSubsetTables)
	addTags(input, internal.HB_SUBSET_SETS_DROP_TABLE_TAG, o.DropTables)
	addValues(input, internal.HB_SUBSET_SETS_NAME_ID, o.NameIDs)
	addValues(input, internal.HB_SUBSET_SETS_NAME_LANG_ID, o.NameLanguages)

	if len(o.GlyphMapping) > 0 {
		m := uint32(x.m.Xhb_subset_input_old_to_new_glyph_mapping(int32(input)))
		for oldGID, newGID := range o.GlyphMapping {
			x.m.Xhb_map_set(int32(m), int32(oldGID), int32(newGID))
		}
	}

	if o.PinAllAxesToDefault {
		if x.m.Xhb_subset_input_pin_all_axes_to_default(int32(input), int32(face)) == 0 {
			return fmt.Errorf("hbsubset: failed to pin all axes to default")
		}
	}
	for _, tag := range o.PinAxesToDefault {
		if x.m.Xhb_subset_input_pin_axis_to_default(int32(input), int32(face), int32(tagValue(tag))) == 0 {
			return fmt.Errorf("hbsubset: failed to pin axis %q to default", tag)
		}
	}
	for tag, v := range o.PinAxes {
		if x.m.Xhb_subset_input_pin_axis_location(int32(input), int32(face), int32(tagValue(tag)), v) == 0 {
			return fmt.Errorf("hbsubset: failed to pin axis %q", tag)
		}
	}
	for tag, ar := range o.AxisRanges {
		if x.m.Xhb_subset_input_set_axis_range(int32(input), int32(face), int32(tagValue(tag)), ar.Min, ar.Max, ar.Default) == 0 {
			return fmt.Errorf("hbsubset: failed to set range for axis %q", tag)
		}
	}
	return nil
}
