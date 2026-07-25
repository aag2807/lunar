package types

import "testing"

// A loop variable typed 'any' means a method call on it cannot be resolved,
// and codegen then has to guess the call syntax. Iterating a typed array has
// to give the element type.
func TestIpairsGivesElementType(t *testing.T) {
	input := `
class Ball
	bounce(): void
	end
end

local balls: Ball[] = {}

for _, ball in ipairs(balls) do
	ball.bounce()
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// The index variable of ipairs is a number.
func TestIpairsIndexIsNumber(t *testing.T) {
	input := `
local names: string[] = {"a", "b"}

for index, name in ipairs(names) do
	local n: number = index
	local s: string = name
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// pairs over a typed table gives its key and value types.
func TestPairsGivesKeyAndValueTypes(t *testing.T) {
	input := `
local scores: table<string, number> = {}

for name, score in pairs(scores) do
	local n: string = name
	local s: number = score
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Element typing must not hide a real mistake.
func TestLoopVariableTypeIsStillChecked(t *testing.T) {
	errors := checkSource(t, "local names: string[] = {\"a\"}\nfor _, name in ipairs(names) do\n\tlocal n: number = name\nend")

	if len(errors) == 0 {
		t.Error("expected assigning a string element to a number to fail")
	}
}

// A generic parameter fits a union that lists it, which is what `T?` is.
func TestGenericParamIsAssignableToUnionContainingIt(t *testing.T) {
	input := `
function find<T>(items: T[]): T?
	for _, item in ipairs(items) do
		return item
	end
	return nil
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
