package types

import (
	"strings"
	"testing"

	"lunar/internal/lexer"
	"lunar/internal/parser"
)

func checkTarget(t *testing.T, target, src string) []string {
	t.Helper()
	c := NewCheckerWithOptions(Options{Target: target})
	p := parser.New(lexer.New(src))
	stmts := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return messages(c.Check(stmts))
}

// Lua refuses to load a goto without a visible label, or one that jumps
// into the scope of a local; so does the checker, rather than writing Lua
// that will not load.
func TestGotoRules(t *testing.T) {
	ok := `
for i = 1, 3 do
    if i == 2 then goto continue end
    local shown = i
    print(shown)
    ::continue::
end
do
    local k = 0
    ::top::
    k = k + 1
    if k < 3 then goto top end
end
`
	if errs := checkTarget(t, "lua54", ok); len(errs) != 0 {
		t.Errorf("valid gotos: %v", errs)
	}
	for src, want := range map[string]string{
		"goto nowhere\n": "no visible label 'nowhere' for goto",
		"local function f()\n    goto out\nend\n::out::\n": "no visible label 'out' for goto",
		"goto f\nlocal x = 1\n::f::\nprint(x)\n":           "jumps into the scope of local 'x'",
		"::a::\n::a::\n":                                   "label 'a' already defined",
	} {
		errs := strings.Join(checkTarget(t, "lua54", src), "\n")
		if !strings.Contains(errs, want) {
			t.Errorf("%q: want %q, got %q", src, want, errs)
		}
	}
}

// Lua 5.4's attributes: <const> is enforced like const, <close> needs 5.4.
func TestLocalAttributes(t *testing.T) {
	if errs := checkTarget(t, "lua54", "local x <const> = 5\nlocal h <close> = nil\nprint(x)\n"); len(errs) != 0 {
		t.Errorf("valid attributes: %v", errs)
	}
	for src, want := range map[string]string{
		"local x <const> = 5\nx = 6\n": "Cannot assign to const variable 'x'",
		"local y <foo> = 1\n":          "unknown attribute 'foo'",
	} {
		errs := strings.Join(checkTarget(t, "lua54", src), "\n")
		if !strings.Contains(errs, want) {
			t.Errorf("%q: want %q, got %q", src, want, errs)
		}
	}
	if errs := strings.Join(checkTarget(t, "lua53", "local h <close> = nil\n"), "\n"); !strings.Contains(errs, "'<close>' needs Lua 5.4") {
		t.Errorf("<close> on lua53: %q", errs)
	}
}
