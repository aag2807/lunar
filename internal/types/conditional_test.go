package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestConditionalTypeBasic(t *testing.T) {
	input := `
type IsString<T> = T extends string ? true : false

type A = IsString<string>  -- should be true (boolean literal)
type B = IsString<number>  -- should be false (boolean literal)
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

func TestConditionalTypeWithNumber(t *testing.T) {
	input := `
type IsNumber<T> = T extends number ? "yes" : "no"

type A = IsNumber<number>  -- should be "yes"
type B = IsNumber<string>  -- should be "no"
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

func TestConditionalTypeWithUnion(t *testing.T) {
	input := `
type IsStringOrNumber<T> = T extends string | number ? true : false

type A = IsStringOrNumber<string>  -- should be true
type B = IsStringOrNumber<boolean> -- should be false
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

func TestConditionalTypeNested(t *testing.T) {
	input := `
type TypeName<T> =
	T extends string ? "string" :
	T extends number ? "number" :
	T extends boolean ? "boolean" :
	"other"

type A = TypeName<string>  -- should be "string"
type B = TypeName<number>  -- should be "number"
type C = TypeName<any>     -- should be "other" (any is not specifically string, number, or boolean)
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

func TestConditionalTypeWithComplexTypes(t *testing.T) {
	input := `
type IsArray<T> = T extends any[] ? true : false

type A = IsArray<string[]>  -- should be true
type B = IsArray<number>    -- should be false
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

func TestConditionalTypeReturnTypes(t *testing.T) {
	input := `
type ReturnString<T> = T extends string ? string : number

local x: ReturnString<string> = "hello"
local y: ReturnString<number> = 42
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

func TestConditionalTypeWithIntersection(t *testing.T) {
	input := `
interface HasName
	name: string
end

interface HasAge
	age: number
end

type HasBoth<T> = T extends HasName & HasAge ? true : false

-- This test validates the syntax; actual implementation depends on interface checking
type A = HasBoth<any>  -- should be false
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

func TestConditionalTypeExclude(t *testing.T) {
	// Exclude can be implemented using conditional types:
	// type Exclude<T, U> = T extends U ? never : T
	input := `
type MyExclude<T, U> = T extends U ? never : T

type A = MyExclude<string, string>  -- should be never
type B = MyExclude<string, number>  -- should be string
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

func TestConditionalTypeExtract(t *testing.T) {
	// Extract can be implemented using conditional types:
	// type Extract<T, U> = T extends U ? T : never
	input := `
type MyExtract<T, U> = T extends U ? T : never

type A = MyExtract<string, string>  -- should be string
type B = MyExtract<string, number>  -- should be never
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
