package hbsubset

import (
	"encoding/binary"
	"fmt"
	"unicode"

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

// NameID is an OpenType name ID.
type NameID uint32

const (
	NameCopyright            = NameID(internal.HB_OT_NAME_ID_COPYRIGHT)
	NameFontFamily           = NameID(internal.HB_OT_NAME_ID_FONT_FAMILY)
	NameFontSubfamily        = NameID(internal.HB_OT_NAME_ID_FONT_SUBFAMILY)
	NameUniqueID             = NameID(internal.HB_OT_NAME_ID_UNIQUE_ID)
	NameFullName             = NameID(internal.HB_OT_NAME_ID_FULL_NAME)
	NameVersionString        = NameID(internal.HB_OT_NAME_ID_VERSION_STRING)
	NamePostscriptName       = NameID(internal.HB_OT_NAME_ID_POSTSCRIPT_NAME)
	NameTrademark            = NameID(internal.HB_OT_NAME_ID_TRADEMARK)
	NameManufacturer         = NameID(internal.HB_OT_NAME_ID_MANUFACTURER)
	NameDesigner             = NameID(internal.HB_OT_NAME_ID_DESIGNER)
	NameDescription          = NameID(internal.HB_OT_NAME_ID_DESCRIPTION)
	NameVendorURL            = NameID(internal.HB_OT_NAME_ID_VENDOR_URL)
	NameDesignerURL          = NameID(internal.HB_OT_NAME_ID_DESIGNER_URL)
	NameLicense              = NameID(internal.HB_OT_NAME_ID_LICENSE)
	NameLicenseURL           = NameID(internal.HB_OT_NAME_ID_LICENSE_URL)
	NameTypographicFamily    = NameID(internal.HB_OT_NAME_ID_TYPOGRAPHIC_FAMILY)
	NameTypographicSubfamily = NameID(internal.HB_OT_NAME_ID_TYPOGRAPHIC_SUBFAMILY)
	NameMacFullName          = NameID(internal.HB_OT_NAME_ID_MAC_FULL_NAME)
	NameSampleText           = NameID(internal.HB_OT_NAME_ID_SAMPLE_TEXT)
	NameCidFindfontName      = NameID(internal.HB_OT_NAME_ID_CID_FINDFONT_NAME)
	NameWWSFamily            = NameID(internal.HB_OT_NAME_ID_WWS_FAMILY)
	NameWWSSubfamily         = NameID(internal.HB_OT_NAME_ID_WWS_SUBFAMILY)
	NameLightBackground      = NameID(internal.HB_OT_NAME_ID_LIGHT_BACKGROUND)
	NameDarkBackground       = NameID(internal.HB_OT_NAME_ID_DARK_BACKGROUND)
	NameVariationsPSPrefix   = NameID(internal.HB_OT_NAME_ID_VARIATIONS_PS_PREFIX)
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

	// UnicodesRanges adds unicode codepoints to retain glyphs for. You can make
	// these directly (LatinOffset is ignored) or use
	// golang.org/x/text/unicode/rangetable.
	UnicodeRanges *unicode.RangeTable

	// Glyphs are glyph indices to retain in addition to those reached from the
	// requested unicode codepoints.
	Glyphs []uint32

	// LayoutFeatures are the OpenType layout feature tags to retain.
	LayoutFeatures []Tag

	// LayoutScripts are the OpenType script tags to retain. It defaults to all
	// scripts when empty.
	LayoutScripts []Tag

	// NameIDs are the name-table name IDs to retain.
	NameIDs []NameID

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

	if len(o.Unicodes) > 0 || o.UnicodeRanges != nil {
		s := uint32(x.m.Xhb_subset_input_unicode_set(int32(input)))
		for _, r := range o.Unicodes {
			x.m.Xhb_set_add(int32(s), int32(uint32(r)))
		}
		if o.UnicodeRanges != nil {
			for _, r := range o.UnicodeRanges.R16 {
				if r.Lo > r.Hi || r.Stride < 1 {
					return fmt.Errorf("invalid unicode range %d..%d..%d", r.Lo, r.Hi, r.Stride)
				}
				if r.Stride != 1 {
					for i := range r.Hi + 1 - r.Lo {
						if i%r.Stride == 0 {
							x.m.Xhb_set_add(int32(s), int32(uint32(r.Lo+i)))
						}
					}
				} else {
					x.m.Xhb_set_add_range(int32(s), int32(uint32(r.Lo)), int32(uint32(r.Hi)))
				}
			}
			for _, r := range o.UnicodeRanges.R32 {
				if r.Lo > r.Hi || r.Stride < 1 {
					return fmt.Errorf("invalid unicode range %d..%d..%d", r.Lo, r.Hi, r.Stride)
				}
				if r.Stride != 1 {
					for i := range r.Hi + 1 - r.Lo {
						if i%r.Stride == 0 {
							x.m.Xhb_set_add(int32(s), int32(uint32(r.Lo+i)))
						}
					}
				} else {
					x.m.Xhb_set_add_range(int32(s), int32(uint32(r.Lo)), int32(uint32(r.Hi)))
				}
			}
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

	addTags(input, internal.HB_SUBSET_SETS_LAYOUT_FEATURE_TAG, o.LayoutFeatures)
	addTags(input, internal.HB_SUBSET_SETS_LAYOUT_SCRIPT_TAG, o.LayoutScripts)
	addTags(input, internal.HB_SUBSET_SETS_NO_SUBSET_TABLE_TAG, o.NoSubsetTables)
	addTags(input, internal.HB_SUBSET_SETS_DROP_TABLE_TAG, o.DropTables)
	addValues(x.m, input, internal.HB_SUBSET_SETS_NAME_ID, o.NameIDs)
	addValues(x.m, input, internal.HB_SUBSET_SETS_NAME_LANG_ID, o.NameLanguages)

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

func addValues[T ~uint32](m *internal.Module, input uint32, which int, vals []T) {
	if len(vals) == 0 {
		return
	}
	s := uint32(m.Xhb_subset_input_set(int32(input), int32(which)))
	for _, v := range vals {
		m.Xhb_set_add(int32(s), int32(v))
	}
}
