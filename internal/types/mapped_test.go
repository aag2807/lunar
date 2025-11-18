package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic mapped type parsing
func TestMappedTypeBasic(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type ReadonlyPerson = { readonly [K in keyof Person]: Person[K] }
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

	// Verify the type alias was created
	if _, exists := checker.typeAliases["ReadonlyPerson"]; !exists {
		t.Error("Expected ReadonlyPerson type alias to exist")
	}
}

// Test mapped type with optional modifier
func TestMappedTypeOptional(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type PartialPerson = { [K in keyof Person]?: Person[K] }
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

// Test mapped type transforming value types
func TestMappedTypeTransform(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type Nullable<T> = { [K in keyof T]: T[K] | nil }

type NullablePerson = Nullable<Person>
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

// Test Record utility type (simple mapped type)
func TestMappedTypeRecord(t *testing.T) {
	input := `
type Record<K, V> = { [P in K]: V }

type StringRecord = Record<"foo" | "bar", number>
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

// Test Partial utility type
func TestMappedTypePartial(t *testing.T) {
	input := `
type Partial<T> = { [K in keyof T]?: T[K] }

interface Config
	host: string
	port: number
	debug: boolean
end

type PartialConfig = Partial<Config>
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

// Test Readonly utility type
func TestMappedTypeReadonly(t *testing.T) {
	input := `
type Readonly<T> = { readonly [K in keyof T]: T[K] }

interface Point
	x: number
	y: number
end

type ReadonlyPoint = Readonly<Point>
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
