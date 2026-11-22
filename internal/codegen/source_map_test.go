package codegen

import (
	"strings"

	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"testing"
)

// The source-map generator must track the same receiver state as the plain
// one: a typed array of class instances used to write to a nil map and panic.
func TestSourceMapGenerationMatchesPlainGeneration(t *testing.T) {
	input := `class Item
    n: number
    constructor(n: number)
        self.n = n
    end
    get(): number
        return self.n
    end
end
local xs: Item[] = { Item(1) }
print(xs[1].get())
`
	p := parser.New(lexer.New(input))
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	types.NewChecker().Check(statements)

	plain := GenerateWithTarget(statements, "lua54")
	mapped, sm := GenerateWithSourceMapAndTarget(statements, "items.lunar", "items.lua", "lua54")
	if sm == nil {
		t.Fatal("no source map returned")
	}
	if mapped != plain {
		t.Errorf("source-map output differs from plain output\nplain:\n%s\nmapped:\n%s", plain, mapped)
	}
	assertContains(t, mapped, "local xs = {Item.new(1)}")
}

// decodeLines decodes a source map's mappings into generated line ->
// source line (both 1-based), failing on a malformed segment.
func decodeLines(t *testing.T, mappings string) map[int]int {
	t.Helper()
	const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	out := map[int]int{}
	srcLine := 0
	for gen, group := range strings.Split(mappings, ";") {
		if group == "" {
			continue
		}
		for _, seg := range strings.Split(group, ",") {
			if seg == "" {
				t.Fatalf("empty segment on generated line %d: %q", gen+1, mappings)
			}
			var vals []int
			shift, value := 0, 0
			for _, ch := range seg {
				d := strings.IndexRune(b64, ch)
				value += (d & 31) << shift
				if d&32 != 0 {
					shift += 5
					continue
				}
				if value&1 != 0 {
					value = -(value >> 1)
				} else {
					value >>= 1
				}
				vals = append(vals, value)
				shift, value = 0, 0
			}
			if len(vals) >= 3 {
				srcLine += vals[2]
				if _, seen := out[gen+1]; !seen {
					out[gen+1] = srcLine + 1
				}
			}
		}
	}
	return out
}

// Every statement maps to its line, nested ones too: a runtime error
// inside a function body is found through the map.
func TestSourceMapCoversNestedStatements(t *testing.T) {
	input := "local function f(n)\n    if n > 1 then\n        error(\"boom\")\n    end\n    return n\nend\nprint(f(1))\n"
	p := parser.New(lexer.New(input))
	statements := p.Parse()
	types.NewChecker().Check(statements)
	lua, sm := GenerateWithSourceMapAndTarget(statements, "f.lunar", "f.lua", "lua54")
	lines := decodeLines(t, sm.Mappings)
	for i, line := range strings.Split(lua, "\n") {
		if strings.Contains(line, "error(\"boom\")") {
			if lines[i+1] != 3 {
				t.Errorf("error(\"boom\") on generated line %d maps to %d, want 3; mappings %v\n%s", i+1, lines[i+1], lines, lua)
			}
			return
		}
	}
	t.Fatalf("no error call in:\n%s", lua)
}
