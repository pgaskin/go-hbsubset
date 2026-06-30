package hbsubset

import (
	"errors"
	"math"
	"sync"
)

// Face is a loaded Harfbuzz font. It is safe for concurrent use, but all
// methods lock a mutex.
type Face struct {
	mu   sync.Mutex
	x    *instance
	data uint32
	face uint32
}

// NewFace loads an sfnt (TrueType/OpenType) font face from data. The index
// should be 0 unless data is a font collection (TTC/OTC).
func NewFace(data []byte, index int) (*Face, error) {
	if index < 0 || int64(index) > math.MaxUint32 {
		return nil, errors.New("invalid face index")
	}
	x := newInstance()
	dataPtr, err := x.copy(data)
	if err != nil {
		return nil, err
	}
	facePtr := uint32(x.m.Xhbw_face_create(int32(dataPtr), int32(uint32(len(data))), int32(uint32(index))))
	if facePtr == 0 {
		return nil, errors.New("failed to load face (not a valid sfnt font?)")
	}
	return &Face{
		x:    x,
		data: dataPtr,
		face: facePtr,
	}, nil
}

// Subset is a helper to call [NewFace] and [Face.Subset].
func Subset(font []byte, index int, opts *Options) ([]byte, error) {
	face, err := NewFace(font, index)
	if err != nil {
		return nil, err
	}
	return face.Subset(opts)
}

// SubsetWithMapping is like [Subset] but also returns the [Mapping].
func SubsetWithMapping(font []byte, index int, opts *Options) ([]byte, Mapping, error) {
	face, err := NewFace(font, index)
	if err != nil {
		return nil, Mapping{}, err
	}
	return face.SubsetWithMapping(opts)
}

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

// Preprocess prepares the font to optimize repeated subset calls.
func (f *Face) Preprocess() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	pre := uint32(f.x.m.Xhb_subset_preprocess(int32(f.face)))
	if pre == 0 {
		return errors.New("preprocess failed")
	}
	f.x.m.Xhb_face_destroy(int32(f.face))
	f.face = pre
	return nil
}

// Subset subsets the face according to opts and returns the new sfnt.
func (f *Face) Subset(opts *Options) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return subset(f.x, f.face, opts)
}

// SubsetWithMapping is like [Face.Subset] but also returns the [Mapping].
func (f *Face) SubsetWithMapping(opts *Options) ([]byte, Mapping, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return subsetPlan(f.x, f.face, opts)
}

// GlyphCount returns the number of glyphs in the face.
func (f *Face) GlyphCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int(uint32(f.x.m.Xhb_face_get_glyph_count(int32(f.face))))
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

// TableTags returns the sfnt table tags present in the face.
func (f *Face) TableTags() []Tag {
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

func subset(x *instance, face uint32, opts *Options) ([]byte, error) {
	input, err := makeInput(x, face, opts)
	if err != nil {
		return nil, err
	}
	defer x.m.Xhb_subset_input_destroy(int32(input))

	res := uint32(x.m.Xhb_subset_or_fail(int32(face), int32(input)))
	if res == 0 {
		return nil, errors.New("subsetting failed")
	}
	defer x.m.Xhb_face_destroy(int32(res))

	return x.blob(x.m.Xhbw_face_blob(int32(res)))
}

func subsetPlan(x *instance, face uint32, opts *Options) ([]byte, Mapping, error) {
	input, err := makeInput(x, face, opts)
	if err != nil {
		return nil, Mapping{}, err
	}
	defer x.m.Xhb_subset_input_destroy(int32(input))

	plan := uint32(x.m.Xhb_subset_plan_create_or_fail(int32(face), int32(input)))
	if plan == 0 {
		return nil, Mapping{}, errors.New("subsetting failed")
	}
	defer x.m.Xhb_subset_plan_destroy(int32(plan))

	res := uint32(x.m.Xhb_subset_plan_execute_or_fail(int32(plan)))
	if res == 0 {
		return nil, Mapping{}, errors.New("subsetting failed")
	}
	defer x.m.Xhb_face_destroy(int32(res))

	readMap := func(mapPtr uint32) (map[uint32]uint32, error) {
		pop := uint32(x.m.Xhb_map_get_population(int32(mapPtr)))
		if pop == 0 {
			return map[uint32]uint32{}, nil
		}
		keys, err := x.alloc(int(pop) * 4)
		if err != nil {
			return nil, err
		}
		defer x.free(keys)
		vals, err := x.alloc(int(pop) * 4)
		if err != nil {
			return nil, err
		}
		defer x.free(vals)

		got := uint32(x.m.Xhbw_map_entries(int32(mapPtr), int32(keys), int32(vals), int32(pop)))
		out := make(map[uint32]uint32, got)
		for i := range got {
			out[x.u32(keys+i*4)] = x.u32(vals + i*4)
		}
		return out, nil
	}

	m := Mapping{}
	if m.oldToNew, err = readMap(uint32(x.m.Xhb_subset_plan_old_to_new_glyph_mapping(int32(plan)))); err != nil {
		return nil, Mapping{}, err
	}
	if m.newToOld, err = readMap(uint32(x.m.Xhb_subset_plan_new_to_old_glyph_mapping(int32(plan)))); err != nil {
		return nil, Mapping{}, err
	}
	if m.uniToOld, err = readMap(uint32(x.m.Xhb_subset_plan_unicode_to_old_glyph_mapping(int32(plan)))); err != nil {
		return nil, Mapping{}, err
	}

	b, err := x.blob(x.m.Xhbw_face_blob(int32(res)))
	if err != nil {
		return nil, Mapping{}, err
	}
	return b, m, nil
}

func makeInput(x *instance, face uint32, opts *Options) (uint32, error) {
	if opts == nil {
		opts = &Options{}
	}
	input := uint32(x.m.Xhb_subset_input_create_or_fail())
	if input == 0 {
		return 0, errors.New("failed to create subset input")
	}
	if err := opts.apply(x, input, face); err != nil {
		x.m.Xhb_subset_input_destroy(int32(input))
		return 0, err
	}
	return input, nil
}
