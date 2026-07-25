package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Every Lua standard library has type declarations in stdlib/; they all need to
// be loaded, not just the four the checker used to name explicitly.
func TestLuaStandardLibraryGlobalsAreDefined(t *testing.T) {
	for _, global := range []string{"table", "string", "math", "os", "io", "coroutine", "debug", "package"} {
		t.Run(global, func(t *testing.T) {
			l := lexer.New("local x: any = " + global)
			p := parser.New(l)
			statements := p.Parse()

			if len(p.Errors()) > 0 {
				t.Fatalf("parse errors: %v", p.Errors())
			}

			checker := NewChecker("../../stdlib")
			for _, err := range checker.Check(statements) {
				t.Errorf("unexpected type error: %s", err.Message)
			}
		})
	}
}
