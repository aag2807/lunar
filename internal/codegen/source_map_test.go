package codegen

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
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
