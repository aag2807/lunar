package types

import "testing"

// Declarations describe C and stdlib APIs, where trailing arguments are
// routinely optional (assert's message, an engine call's flags). Both
// spellings must let the caller leave them out.
func TestDeclaredFunctionOptionalParameterMayBeOmitted(t *testing.T) {
	input := `
declare function spawn(name: string, x?: number, y: number?): number end
local a: number = spawn("goblin")
local b: number = spawn("goblin", 1)
local c: number = spawn("goblin", 1, 2)
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestDeclaredFunctionRequiredParameterIsStillRequired(t *testing.T) {
	input := `
declare function spawn(name: string, x?: number): number end
local a: number = spawn()
`
	if errs := checkSource(t, input); len(errs) == 0 {
		t.Error("expected an error for a missing required argument")
	}
}

// assert(cond) with no message is the most common call in Lua.
func TestStdlibAssertTakesOneArgument(t *testing.T) {
	input := `
local x: number = 3
assert(x == 3)
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
