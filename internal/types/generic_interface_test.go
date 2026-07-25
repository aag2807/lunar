package types

import "testing"

// An interface can take type parameters, the way classes and functions do.
func TestGenericInterfaceResolves(t *testing.T) {
	input := `
interface Container<T>
	value: T
end

function unwrap(box: Container<number>): number
	return box.value
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestGenericInterfaceWithSeveralParameters(t *testing.T) {
	input := `
interface Pair<A, B>
	first: A
	second: B
end

function label(p: Pair<string, number>): string
	return p.first .. tostring(p.second)
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// The substitution has to be real: Container<number> holds a number.
func TestGenericInterfaceSubstitutionIsChecked(t *testing.T) {
	errors := checkSource(t, "interface Box<T>\n\tvalue: T\nend\n\nlocal b: Box<number> = { value = \"text\" }")

	if len(errors) == 0 {
		t.Error("expected a string value to be rejected for Box<number>")
	}
}

func TestGenericInterfaceArityIsChecked(t *testing.T) {
	errors := checkSource(t, "interface Box<T>\n\tvalue: T\nend\n\nlocal b: Box<number, string> = { value = 1 }")

	if len(errors) == 0 {
		t.Error("expected the wrong number of type arguments to be reported")
	}
}

// A declared generic function has its own parameters in scope, which is how the
// testing vendor library declares expect<T>.
func TestGenericAmbientFunctionDeclaration(t *testing.T) {
	input := `
declare interface Expectation<T>
	toBe: function(expected: T): void
end

declare function expect<T>(value: T): Expectation<T> end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// An interface may name itself, as crypto's DigestContext does.
func TestSelfReferentialInterface(t *testing.T) {
	input := `
interface Digest
	update: function(data: string): Digest
	final: function(): string
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
