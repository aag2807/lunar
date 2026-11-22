package types

import (
	"strings"
	"testing"
)

// A function in a table literal is a plain field, as in every Lua module:
// M.greet("bob") is a '.' call.
func TestLiteralFunctionFieldsArePlainFields(t *testing.T) {
	for _, err := range checkSource(t, `
local M = { greet = function(name: string): string
    return "hi " .. name
end }
local s: string = M.greet("bob")
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A function-typed property accepts a literal function.
func TestFunctionTypedPropertyAcceptsALiteralFunction(t *testing.T) {
	for _, err := range checkSource(t, `
interface Handlers
    onHit: function(n: number): void
end
local h: Handlers = { onHit = function(n: number): void
end }
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// An interface method is called with ':', so a literal implementing one
// takes self first, and leaving it out is reported as such.
func TestInterfaceMethodLiteralTakesSelf(t *testing.T) {
	ok := `
interface Enemy
    name: string
    attack(target: string): number
end
local imp: Enemy = { name = "imp", attack = function(self, target: string): number
    return #target
end }
local n: number = imp.attack("orc")
`
	for _, err := range checkSource(t, ok) {
		t.Errorf("unexpected type error: %s", err.Message)
	}

	errs := checkSource(t, strings.Replace(ok, "function(self, target: string)", "function(target: string)", 1))
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Field 'attack' of 'Enemy' is a method") || !strings.Contains(errs[0].Message, "self") {
		t.Fatalf("got %v", messages(errs))
	}
}
