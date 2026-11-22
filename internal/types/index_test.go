package types

import (
	"strings"
	"testing"
)

// indexed checks src as file "m.lunar" and returns the index.
func indexed(t *testing.T, src string) (*Index, string) {
	t.Helper()
	c := NewChecker()
	for _, e := range c.Check(parseFile(t, "m.lunar", src)) {
		t.Fatalf("unexpected type error: %s", e.Message)
	}
	return c.Index(), src
}

// symbolAt resolves the n-th (1-based) occurrence of word in src.
func symbolAt(t *testing.T, ix *Index, src, word string, n int) *Symbol {
	t.Helper()
	offset := -1
	for i := 0; i < n; {
		next := strings.Index(src[offset+1:], word)
		if next < 0 {
			t.Fatalf("%q occurs fewer than %d times", word, n)
		}
		offset += next + 1
		before := offset == 0 || !isWordByte(src[offset-1])
		after := offset+len(word) >= len(src) || !isWordByte(src[offset+len(word)])
		if before && after {
			i++
		}
	}
	occ, ok := ix.At("m.lunar", offset)
	if !ok {
		t.Fatalf("nothing indexed at occurrence %d of %q (offset %d)", n, word, offset)
	}
	return occ.Sym
}

func isWordByte(b byte) bool {
	return b == '_' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}

func expectDecl(t *testing.T, sym *Symbol, kind SymbolKind, line, col int) {
	t.Helper()
	if sym == nil {
		t.Fatal("no symbol")
	}
	if sym.Kind != kind || sym.Decl.Line != line || sym.Decl.Column != col {
		t.Errorf("got %s %q declared at %d:%d, want %s at %d:%d", sym.Kind, sym.Name, sym.Decl.Line, sym.Decl.Column, kind, line, col)
	}
}

func TestIndexLocalsFunctionsAndParameters(t *testing.T) {
	ix, src := indexed(t, `local hp = 10
function heal(amount: number): number
    return amount + hp
end
local n: number = heal(3)
`)
	expectDecl(t, symbolAt(t, ix, src, "hp", 2), SymVariable, 1, 7)
	expectDecl(t, symbolAt(t, ix, src, "amount", 2), SymParameter, 2, 15)
	heal := symbolAt(t, ix, src, "heal", 2)
	expectDecl(t, heal, SymFunction, 2, 10)
	if fn, ok := heal.Type.(*FunctionType); !ok || len(fn.ParamNames) != 1 || fn.ParamNames[0] != "amount" {
		t.Errorf("heal's type has param names %v", heal.Type)
	}
}

func TestIndexClassMembersTypesAndInheritance(t *testing.T) {
	ix, src := indexed(t, `class Enemy
    hp: number
    constructor(hp: number)
        self.hp = hp
    end
    hit(n: number): void
        self.hp = self.hp - n
    end
end
class Boss extends Enemy
    constructor(hp: number)
        super(hp)
    end
end
local e: Enemy = Enemy(3)
e.hit(1)
local b: Boss = Boss(9)
local left: number = b.hp
`)
	occ, ok := ix.At("m.lunar", strings.Index(src, "b.hp")+2) // inherited from Enemy
	if !ok {
		t.Fatal("b.hp is not indexed")
	}
	hp := occ.Sym
	expectDecl(t, hp, SymProperty, 2, 5)
	if hp.Container != "Enemy" {
		t.Errorf("container %q", hp.Container)
	}
	expectDecl(t, symbolAt(t, ix, src, "hit", 2), SymMethod, 6, 5)
	expectDecl(t, symbolAt(t, ix, src, "Enemy", 4), SymClass, 1, 7) // the annotation in `local e: Enemy`
}

func TestIndexEnumMembers(t *testing.T) {
	ix, src := indexed(t, "enum Color\n    Red\n    Blue\nend\nlocal c: Color = Color.Blue\n")
	expectDecl(t, symbolAt(t, ix, src, "Blue", 2), SymEnumMember, 3, 5)
}

// A narrowed variable is the same variable; a shadowing local is another.
func TestIndexNarrowingAndShadowing(t *testing.T) {
	ix, src := indexed(t, `function f(b: number?): number
    if b ~= nil then
        return b + 1
    end
    local x = 1
    do
        local x = 2
        return x
    end
end
`)
	expectDecl(t, symbolAt(t, ix, src, "b", 3), SymParameter, 1, 12)
	expectDecl(t, symbolAt(t, ix, src, "x", 3), SymVariable, 7, 15)
}
