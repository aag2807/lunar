package codegen

import (
	"strings"
	"testing"
)

// A struct pattern must only match when its named fields are present,
// otherwise { name: n, age: a } also matches a table carrying no age.
func TestStructPatternRequiresItsFields(t *testing.T) {
	output := generate(t, `
function greet(person: any): string
	return match person with
		| { name: n, age: a } -> n
		| { name: n } -> n
		| _ -> "stranger"
	end
end
`)

	assertContains(t, output, "__match_value.age ~= nil")
	assertContains(t, output, "__match_value.name ~= nil")
}

// Field order must not depend on Go map iteration order.
func TestStructPatternFieldOrderIsStable(t *testing.T) {
	source := `
function f(e: any): string
	return match e with
		| { zebra: z, alpha: a, middle: m } -> z
		| _ -> "none"
	end
end
`

	first := generate(t, source)
	for i := 0; i < 8; i++ {
		if got := generate(t, source); got != first {
			t.Fatalf("generated output differs between runs:\n%s\n---\n%s", first, got)
		}
	}
}

// A guard's bindings live in the guard's own closure, so the arm body needs
// its own copies or it reads a nil global.
func TestGuardBindingsReachTheBody(t *testing.T) {
	output := generate(t, `
function classify(n: any): string
	return match n with
		| code when code >= 500 -> tostring(code)
		| _ -> "other"
	end
end
`)

	// Once for the guard, once for the body.
	if got := strings.Count(output, "local code = __match_value"); got != 2 {
		t.Errorf("expected the binding both in the guard and the body, got %d occurrences\n%s", got, output)
	}
}

// A field matched against nil explicitly must not also require presence.
func TestStructPatternNilFieldIsNotRequired(t *testing.T) {
	output := generate(t, `
function f(e: any): string
	return match e with
		| { value: nil } -> "empty"
		| _ -> "other"
	end
end
`)

	assertNotContains(t, output, "__match_value.value ~= nil")
	assertContains(t, output, "__match_value.value == nil")
}
