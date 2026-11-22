package types

import "testing"

// The owner's duel phase has no speed: a field typed `number?` may be left
// out of a literal, the same as `speed?: number`.
func TestOptionalFieldsMayBeLeftOut(t *testing.T) {
	input := `
interface Phase
    name: string
    speed: number?
    delay?: number
end
local duel: Phase = { name = "duel" }
local chase: Phase = { name = "chase", speed = 3, delay = 1 }
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Reading an optional field gives T | nil, whichever way it was written.
func TestOptionalFieldReadsAsOptional(t *testing.T) {
	ok := `
interface Phase
    delay?: number
end
local p: Phase = {}
local d: number? = p.delay
`
	for _, err := range checkSource(t, ok) {
		t.Errorf("unexpected type error: %s", err.Message)
	}

	bad := `
interface Phase
    delay?: number
end
local p: Phase = {}
local d: number = p.delay
`
	if errs := checkSource(t, bad); len(errs) != 1 {
		t.Errorf("expected one error assigning number | nil to number, got %v", messages(errs))
	}
}

func TestRequiredFieldIsStillRequired(t *testing.T) {
	errs := checkSource(t, `
interface Phase
    name: string
    speed?: number
end
local p: Phase = { speed = 2 }
`)
	if len(errs) != 1 {
		t.Fatalf("expected one error for the missing name, got %v", messages(errs))
	}
}

// Fields inherited through extends are part of the interface a literal has
// to satisfy.
func TestInheritedFieldsAreRequired(t *testing.T) {
	src := `
interface Base
    id: number
    note?: string
end
interface Child extends Base
    name: string
end
local good: Child = { id = 1, name = "x" }
`
	for _, err := range checkSource(t, src) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	if errs := checkSource(t, src+"local bad: Child = { name = \"x\" }\n"); len(errs) != 1 {
		t.Errorf("expected one error for the missing inherited id, got %v", messages(errs))
	}
}
