MAKEFLAGS += -r

WASM2GO_VERSION  := 7cf5bf3860ec207b09027f2d568e08ff143f39d3
HARFBUZZ_VERSION := $(shell cut -c1-10 harfbuzz/COMMIT)

CXX      := wasm32-wasip1-clang++
GO       := $(if $(GOROOT),$(GOROOT)/bin/,)go
WASM_OPT := wasm-opt
WASM2GO  := $(GO) run github.com/ncruces/wasm2go@$(WASM2GO_VERSION)

OUTDIR := ../internal/

comma := ,
EXPORTS := $(shell grep -vE '^\s*(#.*)?$$' exports.txt)

LDFLAGS := \
	-mexec-model=reactor \
	$(addprefix -Wl$(comma)--export=,$(EXPORTS)) \
	-Wl,--no-entry \
	-Wl,--gc-sections

OPTFLAGS := \
	-O3 \
	--enable-bulk-memory \
	--enable-mutable-globals \
	--enable-sign-ext \
	--enable-nontrapping-float-to-int \
	--enable-multivalue \
	--low-memory-unused \
	--strip-debug \
	--strip-dwarf

CXXFLAGS := \
	-ffile-prefix-map== \
	-ffile-prefix-map=harfbuzz=github.com/harfbuzz/harfbuzz@$(HARFBUZZ_VERSION) \
	-ffile-compilation-dir=. \
	-std=c++17 \
	-O2 \
	-fno-exceptions \
	-fno-rtti \
	-fno-threadsafe-statics \
	-fvisibility=hidden \
	-DNDEBUG \
	-DHB_NO_MT \
	-DHB_NO_MMAP \
	-Wall \
	-Wno-unused-function \
	-Iharfbuzz/src

WRAPPER_FLAGS := \
	-mmultivalue \
	-Xclang -target-abi \
	-Xclang experimental-mv

OBJS := \
	harfbuzz/src/harfbuzz-subset.o \
	wrapper.o

.PHONY: all clean distclean download
all: $(OUTDIR)hbsubset.go $(OUTDIR)const.go $(OUTDIR)COPYING

clean:
	rm -rf $(OBJS) *.wasm

distclean: clean
	rm -rf harfbuzz

download:
	CGO_ENABLED=0 $(GO) run ./download.go

$(OUTDIR)hbsubset.go: hbsubset.opt.wasm
	CGO_ENABLED=0 $(WASM2GO) -embed -pkg internal -o $@ $<

$(OUTDIR)const.go: harfbuzz/src/hb-subset.h
	CGO_ENABLED=0 $(GO) run ./constgen.go -o $@ $<

$(OUTDIR)COPYING: harfbuzz/COPYING
	cp $^ $@

hbsubset.opt.wasm: hbsubset.wasm
	$(WASM_OPT) $(OPTFLAGS) -o $@ $<

hbsubset.wasm: $(OBJS)
	$(CXX) $(LDFLAGS) -o $@ $(OBJS)

wrapper.o: CXXFLAGS += -mmultivalue -Xclang -target-abi -Xclang experimental-mv

%.o: %.cc
	$(CXX) $(CXXFLAGS) -c -o $@ $<
