package lsp

import (
	"strings"
	"testing"
)

const navSource = `interface Named
    name: string
end
interface Living extends Named
    hp: number
end
type Id = number
class Base
    constructor(public x: number)
    end
    size(): number
        return self.x
    end
end
class Point extends Base implements Named
    name: string = "p"
end
function first<T>(xs: T[]): T
    return xs[1]
end
namespace Geometry
    export function area(w: number, h: number): number
        return w * h
    end
end
local id: Id = 1
local p = Base.new(1)
print(first<number>({1}), Geometry.area(1, 2), p.x, id)
local M = {}
function M.add(a, b) return a + b end
print(M.add(1, 2))
`

// Definition works on every kind of name: those after extends and
// implements, type aliases, generic calls, namespace members, Class.new,
// constructor parameter properties, self.x, and plain Lua table fields.
func TestDefinitionOfEveryKindOfName(t *testing.T) {
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": navSource})
	c := newClientAt(t, root)
	c.open("a.lunar")
	cases := []struct {
		word         string
		nth, delta   int
		wantDeclLine int
	}{
		{"Named", 2, 0, 0},   // interface Living extends Named
		{"Named", 3, 0, 0},   // implements Named
		{"Base", 2, 0, 7},    // class Point extends Base
		{"Id = 1", 1, 0, 6},  // local id: Id
		{"first<", 2, 0, 17}, // first<number>(...)
		{"area(1", 1, 0, 21}, // Geometry.area
		{"new(1)", 1, 0, 8},  // Base.new: the constructor
		{"p.x", 1, 2, 8},     // constructor(public x)
		{"self.x", 1, 5, 8},  // self.x
		{"add(1", 1, 0, 29},  // M.add
	}
	for _, tc := range cases {
		params := map[string]any{"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, navSource, tc.word, tc.nth, tc.delta)}
		res, errMsg := c.call("textDocument/definition", params)
		var loc Location
		if errMsg == "" {
			loc = decode[Location](t, res)
		}
		if errMsg != "" || loc.URI == "" || loc.Range.Start.Line != tc.wantDeclLine {
			t.Errorf("%q #%d: definition %+v %s, want line %d", tc.word, tc.nth, loc, errMsg, tc.wantDeclLine)
		}
	}
}

// In `function Point:len()` the name is the method, not the implicit self.
func TestMethodNameIsNotSelf(t *testing.T) {
	text := "local Point = {}\nPoint.__index = Point\nfunction Point:len()\n    return 1\nend\nprint(Point:len())\n"
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": text})
	c := newClientAt(t, root)
	c.open("a.lunar")
	h := decode[Hover](t, c.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, "len()", 1, 0)}))
	if strings.Contains(h.Contents.Value, "parameter") {
		t.Errorf("hover on the method name: %q", h.Contents.Value)
	}
	loc := decode[Location](t, c.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, "len()", 2, 0)}))
	if loc.Range.Start.Line != 2 {
		t.Errorf("definition of Point:len() at line %d", loc.Range.Start.Line)
	}
}

// Workspace symbols include what the project's declaration files declare,
// and not every key of every table literal.
func TestWorkspaceSymbolsIncludeDeclarations(t *testing.T) {
	root := writeTree(t, map[string]string{
		"lunar.config.json":  `{"outDir": "../out"}`,
		"api/engine.d.lunar": "declare function spawn(kind: string): number\n",
		"a.lunar":            "local cfg = { spawnRate = 1 }\nprint(spawn(\"x\"), cfg.spawnRate)\n",
	})
	c := newClientAt(t, root)
	c.open("a.lunar")
	syms := decode[[]SymbolInformation](t, c.request("workspace/symbol", map[string]any{"query": "spawn"}))
	var names []string
	for _, s := range syms {
		names = append(names, s.Name)
	}
	if strings.Join(names, ",") != "spawn" {
		t.Errorf("workspace symbols for spawn: %v", names)
	}
}

// Hover shows a property as declared (scale?:), a generic member with the
// instance's type, and a setter-only property with the setter's type.
func TestHoverDetails(t *testing.T) {
	text := "interface Opts\n    scale?: number\nend\nlocal o: Opts = {}\nprint(o.scale)\n" +
		"class Slot<T>\n    item: T\n    constructor(item: T)\n        self.item = item\n    end\nend\nlocal s = Slot<number>(1)\nprint(s.item)\n" +
		"class Knob\n    set level(v: number)\n    end\nend\nlocal k = Knob()\nk.level = 3\n"
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": text})
	c := newClientAt(t, root)
	c.open("a.lunar")
	hover := func(word string, nth, delta int) string {
		h := decode[Hover](t, c.request("textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, word, nth, delta)}))
		return h.Contents.Value
	}
	if got := hover("o.scale", 1, 2); !strings.Contains(got, "scale?: number") {
		t.Errorf("o.scale: %q", got)
	}
	if got := hover("s.item", 1, 2); !strings.Contains(got, "item: number") {
		t.Errorf("s.item: %q", got)
	}
	if got := hover("k.level", 1, 2); !strings.Contains(got, "level: number") {
		t.Errorf("k.level: %q", got)
	}
}
