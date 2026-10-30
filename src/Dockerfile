FROM golang:1.26-trixie AS base

ARG WASI_SDK_VERSION=34
ARG BINARYEN_VERSION=133

ARG WASI_SDK_CHECKSUM=sha256:b761e3a0721dbae9c09a0059e5fdb2bf917d1b4a8a7b430fb3b5aafb0984b2c4
ARG BINARYEN_CHECKSUM=sha256:2dc9c7813f5375db93d96ead4b78222fcc3e2677bbb832297af4797782a37489

ARG WASI_SDK_URL=https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${WASI_SDK_VERSION}/wasi-sdk-${WASI_SDK_VERSION}.0-x86_64-linux.tar.gz
ARG BINARYEN_URL=https://github.com/WebAssembly/binaryen/releases/download/version_${BINARYEN_VERSION}/binaryen-version_${BINARYEN_VERSION}-x86_64-linux.tar.gz

FROM base AS wasi-sdk
ADD --checksum=${WASI_SDK_CHECKSUM} ${WASI_SDK_URL} ./wasi-sdk.tar.gz
RUN mkdir /opt/wasi-sdk && tar -C /opt/wasi-sdk --strip-components=1 -xf wasi-sdk.tar.gz

FROM base AS binaryen
ADD --checksum=${BINARYEN_CHECKSUM} ${BINARYEN_URL} ./binaryen.tar.gz
RUN mkdir /opt/binaryen && tar -C /opt/binaryen --strip-components=1 -xf binaryen.tar.gz

FROM base AS wasm2go-version
COPY --link Makefile /
RUN grep -oE 'WASM2GO_VERSION\s*:=\s*[a-fv0-9.]+$' /Makefile | grep -oE '[a-fv0-9.]+$' > /wasm2go-version.txt

FROM base AS wasm2go
COPY --link --from=wasm2go-version /wasm2go-version.txt /
RUN GOBIN=/ go install -trimpath -v github.com/ncruces/wasm2go@$(cat /wasm2go-version.txt)

FROM base AS sdk
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y make
COPY --link --from=wasi-sdk /opt /opt
COPY --link --from=binaryen /opt /opt
COPY --link --from=wasm2go /wasm2go /usr/local/bin/
ENV PATH="/opt/wasi-sdk/bin:/opt/binaryen/bin:$PATH"

FROM base AS download
WORKDIR /build/src
COPY --link download.go /build/src/
RUN go run download.go

FROM sdk AS build
WORKDIR /build

COPY --link --from=download /build/src/harfbuzz/ /build/src/harfbuzz/
COPY --link Makefile exports.txt wrapper.cc constgen.go /build/src/
RUN mkdir -p /build/internal && make -j"$(nproc)" -Csrc all WASM2GO=wasm2go

FROM scratch
COPY --link --from=build /build/internal/ /internal/
