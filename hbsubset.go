package hbsubset

import (
	"errors"
	"math"
	"slices"

	"github.com/pgaskin/go-hbsubset/internal"
)

//go:generate docker build --platform amd64 --progress plain --output . src

// TODO: is it possible to get better error messages?

type instance struct {
	m *internal.Module
}

func newInstance() *instance {
	x := &instance{m: internal.New()}
	x.m.X_initialize() // run constructors
	return x
}

func (x *instance) alloc(n int) (uint32, error) {
	if n < 0 || int64(n) > math.MaxInt32 {
		return 0, errors.New("out of wasm memory")
	}
	if n == 0 {
		n = 1
	}
	p := x.m.Xmalloc(int32(n))
	if p == 0 {
		return 0, errors.New("out of wasm memory")
	}
	return uint32(p), nil
}

func (x *instance) free(ptr uint32) {
	if ptr != 0 {
		x.m.Xfree(int32(ptr))
	}
}

func (x *instance) copy(b []byte) (uint32, error) {
	p, err := x.alloc(len(b))
	if err != nil {
		return 0, err
	}
	mem := *x.m.Xmemory().Slice()
	copy(mem[p:p+uint32(len(b))], b)
	return p, nil
}

func (x *instance) mem(ptr, n uint32) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	mem := *x.m.Xmemory().Slice()
	if ptr == 0 || ptr > uint32(len(mem)) || n > uint32(len(mem))-ptr {
		return nil, errors.New("invalid wasm pointer")
	}
	return mem[ptr : ptr+n], nil // only valid until the next wasm call
}

func (x *instance) u32(ptr uint32) uint32 {
	mem := *x.m.Xmemory().Slice()
	return uint32(mem[ptr]) | uint32(mem[ptr+1])<<8 | uint32(mem[ptr+2])<<16 | uint32(mem[ptr+3])<<24
}

func (x *instance) blob(ptr, n, handle int32) ([]byte, error) {
	if handle == 0 {
		return nil, errors.New("null hb_blob_t")
	}
	defer x.m.Xhb_blob_destroy(handle) // decrements the refcount, possibly freeing it
	if n == 0 {
		return nil, nil
	}
	b, err := x.mem(uint32(ptr), uint32(n))
	if err != nil {
		return nil, err
	}
	return slices.Clone(b), nil
}
