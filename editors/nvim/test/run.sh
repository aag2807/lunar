#!/bin/sh
# Runs the Neovim smoke test against a freshly built lunar and parser.
# Needs Neovim 0.11+, Go, the tree-sitter CLI and a C compiler.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/bin" "$work/site/parser" "$work/site/queries/lunar"
(cd "$repo" && go build -o "$work/bin/lunar" ./cmd/lunar)
tree-sitter build -o "$work/site/parser/lunar.so" "$repo/editors/tree-sitter-lunar" >/dev/null
cp "$repo"/editors/tree-sitter-lunar/queries/*.scm "$work/site/queries/lunar/"

PATH="$work/bin:$PATH" LUNAR_TEST_PROJECT="$repo/internal/lsp/testdata/projects/game" \
  nvim --headless --clean \
    --cmd "set rtp^=$repo/editors/nvim,$work/site" \
    --cmd "runtime! ftdetect/lunar.lua plugin/lunar.lua" \
    -c "luafile $here/smoke.lua"
