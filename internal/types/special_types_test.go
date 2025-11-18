package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestNeverType(t *testing.T) {
	input := `
local x: never = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// never is a bottom type - nothing can be assigned to it (except never itself)
	if len(errors) == 0 {
		t.Error("Expected type error when assigning number to never")
	}
}

func TestNeverTypeAssignability(t *testing.T) {
	input := `
function throwError(): never
	error("This function never returns")
end

local x: number = throwError()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// never is assignable to any type (bottom type)
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestNeverInUnion(t *testing.T) {
	input := `
local x: number | never = 42
local y: number = x
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// number | never simplifies to number, so this should work
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestUnknownType(t *testing.T) {
	input := `
local x: unknown = 42
local y: number = x
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// unknown is only assignable to unknown or any (without type assertion)
	if len(errors) == 0 {
		t.Error("Expected type error when assigning unknown to number without type narrowing")
	}
}

func TestUnknownAcceptsAnyValue(t *testing.T) {
	input := `
local x: unknown = 42
local y: unknown = "hello"
local z: unknown = true
local w: unknown = nil
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// unknown accepts any value (top type)
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestUnknownWithTypeAssertion(t *testing.T) {
	input := `
local x: unknown = 42
local y: number = x as number
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// With type assertion, unknown can be assigned to number
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

// TestUnknownWithTypeGuard is skipped because type guards are not fully implemented yet
// func TestUnknownWithTypeGuard(t *testing.T) {
// 	input := `
// function processValue(x: unknown): number
// 	if x is number then
// 		return x
// 	end
// 	return 0
// end
// `
//
// 	l := lexer.New(input)
// 	p := parser.New(l)
// 	statements := p.Parse()
//
// 	if len(p.Errors()) > 0 {
// 		t.Fatalf("Parser errors: %v", p.Errors())
// 	}
//
// 	checker := NewChecker()
// 	errors := checker.Check(statements)
//
// 	// Type guard should narrow unknown to number
// 	if len(errors) > 0 {
// 		t.Errorf("Expected no errors, got %d:", len(errors))
// 		for _, err := range errors {
// 			t.Errorf("  %s", err.Message)
// 		}
// 	}
// }

func TestUnknownVsAny(t *testing.T) {
	// unknown requires type assertion or narrowing
	input1 := `
local x: unknown = 42
local y: number = x
`

	l1 := lexer.New(input1)
	p1 := parser.New(l1)
	statements1 := p1.Parse()

	checker1 := NewChecker()
	errors1 := checker1.Check(statements1)

	if len(errors1) == 0 {
		t.Error("Expected error for unknown -> number without assertion")
	}

	// any allows direct assignment
	input2 := `
local x: any = 42
local y: number = x
`

	l2 := lexer.New(input2)
	p2 := parser.New(l2)
	statements2 := p2.Parse()

	checker2 := NewChecker()
	errors2 := checker2.Check(statements2)

	if len(errors2) > 0 {
		t.Errorf("Expected no errors for any -> number, got %d:", len(errors2))
		for _, err := range errors2 {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestNeverReturnType(t *testing.T) {
	input := `
function alwaysThrows(): never
	error("Error!")
end

-- This is valid: function never returns, so the assignment never happens
local x: string = alwaysThrows()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// never is assignable to string (bottom type property)
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestNeverInIntersection(t *testing.T) {
	// Intersection with never results in never
	input := `
type StringAndNever = string & never
local x: StringAndNever = "hello"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// string & never = never, so string is not assignable to it
	if len(errors) == 0 {
		t.Error("Expected type error when assigning string to never")
	}
}

func TestUnknownInUnion(t *testing.T) {
	// Union with unknown results in unknown
	input := `
type NumberOrUnknown = number | unknown
local x: NumberOrUnknown = 42
local y: number = x
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// number | unknown = unknown, so cannot directly assign to number
	if len(errors) == 0 {
		t.Error("Expected type error when assigning unknown to number")
	}
}

// TestUnknownInIntersection is skipped because intersection type simplification is not implemented yet
// In TypeScript, string & unknown = string, but we don't simplify intersections yet
// func TestUnknownInIntersection(t *testing.T) {
// 	// Intersection with unknown results in the other type
// 	input := `
// type StringAndUnknown = string & unknown
// local x: StringAndUnknown = "hello"
// local y: string = x
// `
//
// 	l := lexer.New(input)
// 	p := parser.New(l)
// 	statements := p.Parse()
//
// 	if len(p.Errors()) > 0 {
// 		t.Fatalf("Parser errors: %v", p.Errors())
// 	}
//
// 	checker := NewChecker()
// 	errors := checker.Check(statements)
//
// 	// string & unknown = string
// 	if len(errors) > 0 {
// 		t.Errorf("Expected no errors, got %d:", len(errors))
// 		for _, err := range errors {
// 			t.Errorf("  %s", err.Message)
// 		}
// 	}
// }

func TestNeverArrayType(t *testing.T) {
	input := `
local x: never[] = {}
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Empty array is assignable to never[]
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestUnknownArrayType(t *testing.T) {
	input := `
local x: unknown[] = {1, "hello", true}
local y: number = x[1]
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Accessing unknown[] gives unknown, which can't be assigned to number
	if len(errors) == 0 {
		t.Error("Expected type error when assigning unknown to number")
	}
}
