package types

import "testing"

// A function without a return annotation infers its return type instead of
// being held to void, which would reject every plain Lua function.
func TestUnannotatedFunctionInfersReturnType(t *testing.T) {
	input := `
function double(x: number)
	return x * 2
end

local n: number = double(4)
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Returning nothing still infers void.
func TestUnannotatedFunctionWithNoReturnIsVoid(t *testing.T) {
	errors := checkSource(t, "function shout(s: string)\n\tprint(s)\nend\n\nlocal n: number = shout(\"hi\")")

	if len(errors) == 0 {
		t.Error("expected assigning a void call result to a number to fail")
	}
}

// Disagreeing branches infer a union, and it is still checked.
func TestUnannotatedFunctionInfersUnion(t *testing.T) {
	errors := checkSource(t, "function pick(flag: boolean)\n\tif flag then\n\t\treturn 1\n\tend\n\treturn \"text\"\nend\n\nlocal v: number = pick(true)")

	if len(errors) == 0 {
		t.Fatal("expected a union return type to be rejected for a number variable")
	}
}

// An explicit annotation is still enforced.
func TestAnnotatedReturnTypeIsStillChecked(t *testing.T) {
	errors := checkSource(t, "function f(): number\n\treturn \"text\"\nend")

	if len(errors) == 0 {
		t.Error("expected returning a string from a number function to fail")
	}
}

// A record with function-valued fields fits table<string, any>.
func TestRecordWithMethodsFitsTableType(t *testing.T) {
	input := `
function createPerson(name: string): table<string, any>
	return {name = name, greet = function() print(name) end}
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A record's keys are strings, so a numeric-keyed table must not accept it.
func TestRecordDoesNotFitNumericKeyedTable(t *testing.T) {
	errors := checkSource(t, "function f(): table<number, any>\n\treturn {name = \"x\"}\nend")

	if len(errors) == 0 {
		t.Error("expected a string-keyed record to be rejected for table<number, any>")
	}
}
