package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// A Lua function declaration binds a global, so calling one declared further
// down the file is normal and must type check.
func TestForwardReferenceToLaterFunction(t *testing.T) {
	input := `
function caller(): number
	return callee()
end

function callee(): number
	return 1
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestMutualRecursionWithAnnotations(t *testing.T) {
	input := `
function isEven(n: number): boolean
	if n == 0 then
		return true
	end
	return isOdd(n - 1)
end

function isOdd(n: number): boolean
	if n == 0 then
		return false
	end
	return isEven(n - 1)
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestMutualRecursionWithoutAnnotations(t *testing.T) {
	input := `
function isEven(n: number)
	if n == 0 then
		return true
	end
	return isOdd(n - 1)
end

function isOdd(n: number)
	if n == 0 then
		return false
	end
	return isEven(n - 1)
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A recursive call must not be checked against the placeholder return type
// that is still being inferred.
func TestDirectRecursionWithInferredReturnType(t *testing.T) {
	input := `
function fact(n: number)
	if n <= 1 then
		return 1
	end
	return n * fact(n - 1)
end

local result: number = fact(5)
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Hoisting must not hide real mistakes.
func TestForwardReferenceStillChecksArguments(t *testing.T) {
	errors := checkSource(t, "function caller(): string\n\treturn callee(\"wrong\")\nend\n\nfunction callee(n: number): string\n\treturn tostring(n)\nend")

	if len(errors) == 0 {
		t.Error("expected passing a string to a number parameter to fail")
	}
}

func TestForwardReferenceStillChecksReturnType(t *testing.T) {
	errors := checkSource(t, "function caller(): number\n\treturn callee()\nend\n\nfunction callee(): string\n\treturn \"text\"\nend")

	if len(errors) == 0 {
		t.Error("expected returning a string as a number to fail")
	}
}

func TestUndefinedFunctionIsStillReported(t *testing.T) {
	errors := checkSource(t, "function caller(): number\n\treturn nowhere()\nend")

	if len(errors) == 0 {
		t.Error("expected a call to an undefined function to fail")
	}
}

// The declaration replaces its own hoisted placeholder rather than being
// recorded as an overload of itself.
func TestHoistedFunctionIsNotItsOwnOverload(t *testing.T) {
	input := `
function solo(n: number): number
	return n
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker("../../stdlib")
	checker.Check(statements)

	typ, ok := checker.GetEnv().Get("solo")
	if !ok {
		t.Fatal("solo not registered")
	}

	fnType, ok := typ.(*FunctionType)
	if !ok {
		t.Fatalf("solo is %T, want *FunctionType", typ)
	}

	if len(fnType.Overloads) != 0 {
		t.Errorf("expected no overloads, got %d", len(fnType.Overloads))
	}
}
