package lsp

import (
	"strings"
	"testing"
)

// completeAt opens a file holding text (in a project that also has
// hero.lunar) and returns the labels completion offers at the n-th
// occurrence of marker (after it).
func completeAt(t *testing.T, text, marker string) []string {
	t.Helper()
	root := writeTree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../out"}`,
		"hero.lunar":        "export class Hero\n    hp: number = 3\n    static count: number = 0\n    constructor()\n    end\n    hit(n: number): void\n    end\nend\nexport function clamp(n: number): number\n    return n\nend\n",
		"main.lunar":        text,
	})
	c := newClientAt(t, root)
	c.open("main.lunar")
	c.change("main.lunar", text)
	i := strings.Index(text, marker)
	if i < 0 {
		t.Fatalf("no %q", marker)
	}
	params := map[string]any{"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, marker, 1, len(marker))}
	list := decode[CompletionList](t, c.request("textDocument/completion", params))
	var labels []string
	for _, it := range list.Items {
		labels = append(labels, it.Label)
	}
	return labels
}

func has(labels []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, l := range labels {
			if l == w {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestCompletionGaps(t *testing.T) {
	importLine := "import { Hero, clamp } from \"./hero\"\n"
	cases := []struct {
		name, text, marker string
		want               []string
		not                []string
	}{
		{"imported names", importLine + "local x = cla", "= cla", []string{"clamp"}, nil},
		{"a name typed since the file parsed", importLine + "local h2 = Hero()\nh2.", "h2.", []string{"hp", "hit"}, nil},
		{"inside a function being written", importLine + "local function f()\n    local h3 = Hero()\n    h3:", "h3:", []string{"hit"}, nil},
		{"string methods", "local s = \"abc\"\nprint(s:", "s:", []string{"upper", "sub"}, nil},
		{"through ?.", importLine + "local e: Hero? = nil\nprint(e?.", "e?.", []string{"hp"}, nil},
		{"a plain table's fields", "local M = {}\nfunction M.add(a, b) return a + b end\nprint(M.", "print(M.", []string{"add"}, nil},
		{"a class's statics", importLine + "print(Hero.", "Hero.", []string{"new", "count"}, []string{"hp", "hit"}},
		{"not in a comment", importLine + "local h = Hero()\nprint(h) -- h.", "-- h.", nil, []string{"hp"}},
		{"types where a type goes", importLine + "local x: ", "x: ", []string{"Hero", "number"}, []string{"clamp", "print"}},
		{"no interfaces where a value goes", "interface Shape\n    w: number\nend\nlocal v = ", "v = ", nil, []string{"Shape"}},
		{"not in a string", importLine + "local h = Hero()\nprint(\"h.", "\"h.", nil, []string{"hp"}},
	}
	for _, tc := range cases {
		labels := completeAt(t, tc.text, tc.marker)
		if !has(labels, tc.want...) {
			t.Errorf("%s: got %v, want %v", tc.name, labels, tc.want)
		}
		for _, n := range tc.not {
			if has(labels, n) {
				t.Errorf("%s: offered %q (%v)", tc.name, n, labels)
			}
		}
	}
}
