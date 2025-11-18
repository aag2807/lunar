package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestExcludeType(t *testing.T) {
	input := `
type AllTypes = string | number | boolean
type NoStrings = Exclude<AllTypes, string>

local x: NoStrings = 42
local y: NoStrings = true
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Exclude<string | number | boolean, string> = number | boolean
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestExcludeStringFromUnion(t *testing.T) {
	input := `
type AllTypes = string | number | boolean
type NoStrings = Exclude<AllTypes, string>

local x: NoStrings = "hello"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should error: string is not in number | boolean
	if len(errors) == 0 {
		t.Error("Expected type error for assigning string to NoStrings")
	}
}

func TestExcludeMultipleTypes(t *testing.T) {
	input := `
type All = string | number | boolean | nil
type OnlyNumber = Exclude<All, string | boolean | nil>

local x: OnlyNumber = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestExtractType(t *testing.T) {
	input := `
type AllTypes = string | number | boolean
type OnlyNumbers = Extract<AllTypes, number>

local x: OnlyNumbers = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Extract<string | number | boolean, number> = number
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestExtractStringFromUnion(t *testing.T) {
	input := `
type AllTypes = string | number | boolean
type OnlyStrings = Extract<AllTypes, string>

local x: OnlyStrings = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should error: 42 is not assignable to string
	if len(errors) == 0 {
		t.Error("Expected type error for assigning number to OnlyStrings")
	}
}

func TestNonNullableType(t *testing.T) {
	input := `
type MaybeString = string | nil
type DefinitelyString = NonNullable<MaybeString>

local x: DefinitelyString = "hello"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// NonNullable<string | nil> = string
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestNonNullableRemovesNil(t *testing.T) {
	input := `
type MaybeNumber = number | nil
type DefinitelyNumber = NonNullable<MaybeNumber>

local x: DefinitelyNumber = nil
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should error: nil is not in number
	if len(errors) == 0 {
		t.Error("Expected type error for assigning nil to NonNullable type")
	}
}

func TestNonNullableComplexUnion(t *testing.T) {
	input := `
type MaybeValues = string | number | boolean | nil
type NonNullValues = NonNullable<MaybeValues>

local s: NonNullValues = "test"
local n: NonNullValues = 42
local b: NonNullValues = true
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// NonNullable<string | number | boolean | nil> = string | number | boolean
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestExcludeWithLiterals(t *testing.T) {
	input := `
type Status = "loading" | "success" | "error" | "idle"
type ActiveStatus = Exclude<Status, "idle">

local status: ActiveStatus = "loading"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Exclude<"loading" | "success" | "error" | "idle", "idle">
	// = "loading" | "success" | "error"
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestExtractWithLiterals(t *testing.T) {
	input := `
type Response = "success" | "error" | 200 | 404
type StringResponses = Extract<Response, string>

local resp: StringResponses = "success"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Extract<"success" | "error" | 200 | 404, string>
	// = "success" | "error"
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
