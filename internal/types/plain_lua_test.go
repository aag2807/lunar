package types

import (
	"strings"
	"testing"
)

// An untyped function expression infers its return type from its own
// returns, whatever its body, and a function inside it keeps its returns
// to itself.
func TestFunctionExpressionsInferTheirReturns(t *testing.T) {
	for _, err := range checkSource(t, `
local function wrap(t)
    return function(u) return #t + #u end
end
local f = wrap({ 1 })
local n: number = f({ 2 })
local h = function(x: number)
    local z = x * 2
    return z
end
local k: number = h(3)
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// obj:m(x) passes obj as the first argument of a function stored in a
// field, as Lua does.
func TestColonCallOnAFunctionField(t *testing.T) {
	for _, err := range checkSource(t, `
local obj = { tag = function(self, t) return t[1] end }
local v = obj:tag({ "first" })
local w = obj.tag(obj, { "second" })
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	errs := checkSource(t, `
local obj = { tag = function(self, t: number): number return t end }
local v = obj:tag("one")
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Argument 1") {
		t.Errorf("a wrong argument after the receiver: %v", messages(errs))
	}
}

// In Lua, and/or return one of their operands: a or b is a when a is
// truthy and b otherwise; a and b is a when a is falsy and b otherwise.
func TestAndOrReturnAnOperand(t *testing.T) {
	for _, err := range checkSource(t, `
local function bump(c)
    c.attempt = (c.attempt or 0) + 1
end
local function pick(a: number?, b: string?, cond: boolean)
    local n: number = (a or 0) + 1
    local s: string = b or "none"
    local label: string = cond and "yes" or "no"
    local doubled = a and a * 2
    local both: boolean = a ~= nil and b ~= nil
    local five: number = nil or 5
    local first = a or b
    return n, s, label, doubled, both, five, first
end
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	got := strings.Join(messages(checkSource(t, `
local function f(b: string?)
    local n: number = b or "x"
end
local function g(a: number?)
    local m: number = a and a * 2
end
`)), "\n")
	for _, want := range []string{"Cannot assign type 'string' to variable 'n'", "to variable 'm' of type 'number'"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// Untyped Lua is checked like Lua: what is not annotated is not narrowed
// to the first value it held.
func TestUntypedLuaIdioms(t *testing.T) {
	for name, src := range map[string]string{
		"a local declared empty takes a value later": `
local x
x = 5
local best = nil
for _, v in ipairs({ 3, 9, 4 }) do
    if not best or v > best then best = v end
end
print(x + 1, best)`,
		"a table built from a literal gains fields": `
local t = { a = 1 }
t.b = 2
t.name = "x"
print(t.a + t.b, t.name, t.missing)`,
		"an untyped parameter may be left out": `
local function f(a, b)
    b = b or 2
    return a * b
end
print(f(3), f(3, 3))`,
		"functions defined into tables": `
local M = {}
function M.add(a, b) return a + b end
function M.twice(a) return M.add(a, a) end
local Point = {}
Point.__index = Point
function Point.new(x, y)
    return setmetatable({ x = x, y = y }, Point)
end
function Point:len()
    return math.sqrt(self.x ^ 2 + self.y ^ 2)
end
local p = Point.new(3, 4)
print(M.twice(4), p:len())`,
		"table.pack and table.unpack": `
local packed = table.pack(1, 2, 3)
print(packed.n, table.unpack(packed, 1, packed.n))`,
	} {
		for _, err := range checkSource(t, src) {
			t.Errorf("%s: %s", name, err.Message)
		}
	}
}

// What is annotated stays checked.
func TestAnnotatedCodeStaysStrict(t *testing.T) {
	got := strings.Join(messages(checkSource(t, `
interface P
    hp: number
end
local p: P = { hp = 1 }
p.hpp = 2
local function f(a: number, b: number): number
    return a + b
end
local n = f(1)
`)), "\n")
	for _, want := range []string{"'hpp'", "expects at least 2 arguments"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A script may define a global that a declaration file describes, so the
// scripts that call it know its type. An untyped definition takes the
// declared types; a declared non-function cannot be defined as one.
func TestDefiningADeclaredGlobal(t *testing.T) {
	for name, src := range map[string]string{
		"declare function, untyped definition": `
declare function ring_run(n: number): number
function ring_run(n)
    return n + 1
end
local r: number = ring_run(2)`,
		"declare const with a function type": `
declare const ring_spin: function(n: number): number
function ring_spin(n)
    return n * 2
end`,
		"declared any": `
declare const helper: any
function helper(a, b)
    return a
end`,
	} {
		for _, err := range checkSource(t, src) {
			t.Errorf("%s: %s", name, err.Message)
		}
	}
	for src, want := range map[string]string{
		"declare function on_hit(target: string): void\nfunction on_hit(target)\n    local n: number = target\nend\n": "Cannot assign type 'string' to variable 'n'",
		"declare function ring_run(n: number): number\nfunction ring_run(n)\n    return \"x\"\nend\n":                 "Cannot return type",
		"declare const count: number\nfunction count()\nend\n":                                                        "'count' is declared as 'number'",
	} {
		got := strings.Join(messages(checkSource(t, src)), "\n")
		if !strings.Contains(got, want) {
			t.Errorf("%q: want %q, got %q", src, want, got)
		}
	}
}
