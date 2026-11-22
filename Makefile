# Makefile for Lunar

GO       ?= go
GOFLAGS  ?=
DIST_DIR := dist

ifeq ($(OS),Windows_NT)
	EXE_EXT := .exe
	PREFIX  ?= $(USERPROFILE)
else
	EXE_EXT :=
	PREFIX  ?= /usr/local
endif

BINDIR    ?= $(PREFIX)/bin

LUNAR_BIN     := $(DIST_DIR)/lunar$(EXE_EXT)
LUNAR_LSP_BIN := $(DIST_DIR)/lunar-lsp$(EXE_EXT)

.PHONY: all build build-lunar build-lunar-lsp install uninstall test test-lua fmt vet clean help

all: build

## build: build dist/lunar and dist/lunar-lsp
build: build-lunar build-lunar-lsp

build-lunar:
	$(GO) build $(GOFLAGS) -o $(LUNAR_BIN) ./cmd/lunar

build-lunar-lsp:
	$(GO) build $(GOFLAGS) -o $(LUNAR_LSP_BIN) ./cmd/lunar-lsp

## install: copy the binaries to $(BINDIR); the stdlib declarations are built in
install: build
	mkdir -p "$(BINDIR)"
	cp $(LUNAR_BIN) $(LUNAR_LSP_BIN) "$(BINDIR)/"

## uninstall: remove what install copied
uninstall:
	rm -f "$(BINDIR)/lunar$(EXE_EXT)" "$(BINDIR)/lunar-lsp$(EXE_EXT)"

## test: run every Go test (runtime tests skip when Lua is missing)
test:
	$(GO) test ./...

## test-lua: like test, but fail instead of skipping when Lua 5.3/5.4 are missing
test-lua:
	LUNAR_REQUIRE_LUA=1 $(GO) test ./...

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

clean:
	rm -rf $(DIST_DIR)

help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  make /'
