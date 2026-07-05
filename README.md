# go-hbsubset

[![Go Reference](https://pkg.go.dev/badge/github.com/pgaskin/go-hbsubset.svg)](https://pkg.go.dev/github.com/pgaskin/go-hbsubset)
[![Test](https://github.com/pgaskin/go-hbsubset/actions/workflows/test.yml/badge.svg)](https://github.com/pgaskin/go-hbsubset/actions/workflows/test.yml)
[![Attest hbsubset build](https://github.com/pgaskin/go-hbsubset/actions/workflows/attest.yml/badge.svg)](https://github.com/pgaskin/go-hbsubset/actions/workflows/attest.yml)

Go bindings for [HarfBuzz](https://github.com/harfbuzz/harfbuzz)'s font subsetter (`hb-subset`) without cgo.

This library wraps a WebAssembly build of HarfBuzz transpiled to Go using [wasm2go](https://github.com/ncruces/wasm2go).

> [!WARNING]
>
> These bindings are still experimental and are subject to change. They have not been tested extensively yet.

The wasm2go blob is fully [reproducible](./src/Dockerfile) and [verified](https://github.com/pgaskin/go-hbsubset/attestations).

To have working IDE integration while working on the bindings, use `bear -- make -C src distclean download all CXX=/path/to/wasi-sdk/bin/wasm32-wasip1-clang++ WASM_OPT=/path/to/binaryen/bin/wasm-opt` to download the Harfbuzz source and generate the `compile_commands.json`.

## Usage

To subset a single font:

```go
import "github.com/pgaskin/go-hbsubset"

out, err := hbsubset.Subset(font, 0, &hbsubset.Options{
    Unicodes: []rune("some text"),
})
```

You can specify other options too:

```go
import (
    "github.com/pgaskin/go-gfsubsets"
    "golang.org/x/text/unicode/rangetable"
)

out, err := hbsubset.Subset(data, 0, &hbsubset.Options{
    UnicodeRanges: rangetable.Merge(
        gfsubsets.Latin,
        &unicode.RangeTable{
            R16: []unicode.Range16{
                {Stride: 1, Lo: '\u2002', Hi: '\u201E'}, // spaces, smart punctuation
                {Stride: 1, Lo: '\u2022', Hi: '\u2022'}, // bullet
                {Stride: 1, Lo: '\u2026', Hi: '\u2026'}, // ellipsis
            },
        },
    ),
    PinAllAxesToDefault: true,
    AxisRanges: map[hbsubset.Tag]hbsubset.AxisRange{
        hbsubset.MakeTag("wght"): {Min: 100, Max: 900, Default: 400},
    },
    LayoutFeatures: []hbsubset.Tag{
        hbsubset.MakeTag("kern"),
        hbsubset.MakeTag("mkmk"),
        hbsubset.MakeTag("size"),
        hbsubset.MakeTag("liga"),
    },
    NameIDs: []hbsubset.NameID{
        hbsubset.NameUniqueID,
        hbsubset.NameCopyright,
        hbsubset.NameFontFamily,
        hbsubset.NameFontSubfamily,
        hbsubset.NameTypographicFamily,
        hbsubset.NameTypographicSubfamily,
        hbsubset.NameFullName,
        hbsubset.NameMacFullName,
        hbsubset.NamePostscriptName,
        hbsubset.NameManufacturer,
        hbsubset.NameDescription,
        hbsubset.NameVariationsPSPrefix,
    },
    LayoutScripts: []hbsubset.Tag{
        hbsubset.MakeTag("latn"), // latin
    },
    NameLanguages: []uint32{
        1033, // english
    },
})
```

You can reuse a font for multiple subsets (this also lets you access the font metadata):

```go
face, err := hbsubset.NewFace(font, 0)
if err != nil {
    return err
}
face.Preprocess()

for _, page := range pages {
    out, err := face.Subset(&hbsubset.Options{Unicodes: page})
    // ...
}
```

You can use `SubsetWithMapping` to get the renumbered glyphs.

```go
out, m, err := hbsubset.SubsetWithMapping(font, &hbsubset.Options{
    Unicodes: []rune("Hi"),
})

new, ok := m.NewGlyph(oldGID)

for old, new := range m.Glyphs() {
    // ...
}
```
