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
