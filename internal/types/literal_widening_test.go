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
