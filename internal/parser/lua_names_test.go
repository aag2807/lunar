package parser

import (
	"strings"
	"testing"
)

// Lua code where the meaning depends on a line break, which the
// highlighter's grammar cannot see: an `as` starting a line is a variable,
// not an assertion on the line before.
func TestStatementsSplitAtLineBreaks(t *testing.T) {
	cases := map[string]int{
		"local as = 5\nas = as + 1\n": 2,
		"local y = 5\nas.x = 1\n":     2,
		"local x = y as number\n":     1,
	}
	for src, want := range cases {
		stmts, errs := parseWithErrors(src)
		if len(errs) > 0 || len(stmts) != want {
			t.Errorf("%q: %d statements, errors %v", src, len(stmts), errs)
		}
	}
}

// A match that has reached its `with` reports its own mistake, instead of
// being read as the name match and failing on everything after it.
func TestBrokenMatchSaysWhy(t *testing.T) {
	_, errs := parseWithErrors("local v = 1\nlocal s = match v with\n    | 1 => \"one\"\n    | _ -> \"other\"\nend\n")
	if len(errs) == 0 || !strings.Contains(errs[0].Message, "'->'") {
		t.Errorf("errors: %v", errs)
	}
	for _, e := range errs {
		if strings.Contains(e.Message, "is not a statement") {
			t.Errorf("misleading: %s", e.Message)
		}
	}
}
