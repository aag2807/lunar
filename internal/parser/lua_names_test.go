package parser

import "testing"

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
