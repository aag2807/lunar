package types

import (
	"strings"
	"testing"
)

// A match's value is what its arms give, and bindings have the matched
// value's type: both used to be any.
func TestMatchIsTyped(t *testing.T) {
	refused := map[string]string{
		"local function f(n: number): string\n    return match n with\n        | 1 -> 5\n        | _ -> 6\n    end\nend\n":     "number",
		"local s: string = match 3 with\n    | x -> x\nend\n":                                                                  "number",
		"local function g(n: number): number\n    return match n with\n        | \"a\" -> 1\n        | _ -> 2\n    end\nend\n": "never match",
	}
	for src, want := range refused {
		errs := checkSource(t, src)
		if len(errs) == 0 || !strings.Contains(strings.Join(messages(errs), "\n"), want) {
			t.Errorf("%q: %v, want an error with %q", src, messages(errs), want)
		}
	}
	accepted := []string{
		"local function g(n: number): number\n    return match n with\n        | x -> x + 1\n    end\nend\n",
		"local function h(v: number | string): string\n    return match v with\n        | 1 | 2 -> \"low\"\n        | \"a\" -> \"letter\"\n        | _ -> \"other\"\n    end\nend\n",
	}
	for _, src := range accepted {
		if errs := checkSource(t, src); len(errs) != 0 {
			t.Errorf("%q: %v", src, messages(errs))
		}
	}
}
