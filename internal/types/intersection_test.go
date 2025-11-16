package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestIntersectionTypeBasic(t *testing.T) {
	input := `
interface HasName
	name: string
end

interface HasAge
	age: number
end

type Person = HasName & HasAge
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIntersectionTypeMultiple(t *testing.T) {
	input := `
interface A
	a: number
end

interface B
	b: string
end

interface C
	c: boolean
end

type ABC = A & B & C
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIntersectionWithUnion(t *testing.T) {
	input := `
interface Named
	name: string
end

interface Aged
	age: number
end

type PersonData = Named & Aged
type MaybePersonData = PersonData | nil
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIntersectionPrecedence(t *testing.T) {
	input := `
interface X
	x: number
end

interface Y
	y: string
end

interface Z
	z: boolean
end

-- A | B & C should parse as A | (B & C)
type Mixed = X | Y & Z
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIntersectionInFunction(t *testing.T) {
	input := `
interface Printable
	toString: () => string
end

interface Comparable
	compare: () => number
end

type PrintableComparable = Printable & Comparable

function process(obj: PrintableComparable): void
	print("processing")
end
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
