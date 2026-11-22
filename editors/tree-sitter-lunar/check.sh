#!/bin/sh
# Checks the tree-sitter grammar, as CI does:
#   1. the committed src/ is what grammar.js generates,
#   2. the corpus tests pass,
#   3. every .lunar file in the repository parses with no ERROR or MISSING
#      node, which cross-checks the grammar against the compiler's parser.
# Needs the tree-sitter CLI and a C compiler. Run from anywhere.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
cd "$here"

tree-sitter generate --abi 14
if ! git -C "$repo" diff --quiet -- editors/tree-sitter-lunar/src; then
    echo "src/ is out of date: run 'tree-sitter generate --abi 14' in editors/tree-sitter-lunar and commit it" >&2
    git -C "$repo" diff --stat -- editors/tree-sitter-lunar/src >&2
    exit 1
fi

tree-sitter test -p .

list=$(mktemp)
trap 'rm -f "$list"' EXIT
# Plain Lua is Lunar too: the compiler's Lua corpus must parse as well.
git -C "$repo" ls-files '*.lunar' 'tests/luacompat/*.lua' | sed "s|^|$repo/|" > "$list"
bad=$(tree-sitter parse -p . --quiet --paths "$list" 2>/dev/null | grep -E 'ERROR|MISSING' || true)
if [ -n "$bad" ]; then
    echo "these files do not parse cleanly:" >&2
    echo "$bad" >&2
    exit 1
fi
echo "grammar ok: $(wc -l < "$list") files parse cleanly"
