package types

import "testing"

// A mutable local declared from a literal widens to the base type, so it can be
// reassigned; `local state = "idle"` is a string, not the type "idle".
func TestMutableLocalWidensLiteralType(t *testing.T) {
	input := `
local state = "idle"
state = "running"

local count = 0
count = 10

local flag = true
flag = false
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// An explicit annotation still pins the literal type.
func TestAnnotatedLiteralTypeIsKept(t *testing.T) {
	errors := checkSource(t, "local state: \"idle\" = \"idle\"\nstate = \"running\"")

	if len(errors) == 0 {
		t.Error("expected assigning \"running\" to a variable of type \"idle\" to fail")
	}
}

// `as const` opts back in to the literal type.
func TestConstAssertionKeepsLiteralType(t *testing.T) {
	l := `local mode = "fast" as const`

	for _, err := range checkSource(t, l) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// An inferred array of literals is an array of their base type, as a
// single literal widens: `local xs = {1, 2, 3}` is number[].
func TestArrayOfLiteralsWidens(t *testing.T) {
	for _, err := range checkSource(t, `
local xs = {1, 2, 3}
local y: number = xs[1] + 1
xs[2] = 10
local names = {"a", "b"}
names[1] = "c"
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Arithmetic works on a union of numbers.
func TestArithmeticOnANumberLiteralUnion(t *testing.T) {
	for _, err := range checkSource(t, `
type Die = 1 | 2 | 3
local function step(d: Die): number
    return d + 1
end
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
