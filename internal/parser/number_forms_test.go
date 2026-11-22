package parser

import (
	"lunar/internal/ast"
	"testing"
)

func TestParseLuaNumberValues(t *testing.T) {
	cases := map[string]float64{
		"0xFF":    255,
		"0x10":    16,
		"0xA.8p1": 21,
		"1e-9":    1e-9,
		"3E+2":    300,
		".5":      0.5,
	}
	for src, want := range cases {
		statements := parseSource(t, "local n = "+src)
		decl, ok := statements[0].(*ast.VariableDeclaration)
		if !ok {
			t.Fatalf("%s: expected VariableDeclaration, got %T", src, statements[0])
		}
		lit, ok := decl.Values[0].(*ast.NumberLiteral)
		if !ok {
			t.Fatalf("%s: expected NumberLiteral, got %T", src, decl.Values[0])
		}
		if lit.Value != want {
			t.Errorf("%s: value %v, want %v", src, lit.Value, want)
		}
	}
}
