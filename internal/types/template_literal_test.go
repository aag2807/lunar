package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic template literal type
func TestTemplateLiteralBasic(t *testing.T) {
	input := "type Greeting = `Hello ${string}`\n\nconst g: Greeting = \"Hello world\""

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

// Test template literal type with union
func TestTemplateLiteralUnion(t *testing.T) {
	input := "type Direction = \"left\" | \"right\"\ntype ButtonType = `${Direction}Button`\n\n-- Should resolve to \"leftButton\" | \"rightButton\"\nconst btn1: ButtonType = \"leftButton\"\nconst btn2: ButtonType = \"rightButton\""

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

// Test template literal with multiple substitutions
func TestTemplateLiteralMultiple(t *testing.T) {
	input := "type Size = \"small\" | \"large\"\ntype Color = \"red\" | \"blue\"\ntype ClassName = `${Size}-${Color}-button`\n\n-- Should resolve to \"small-red-button\" | \"small-blue-button\" | \"large-red-button\" | \"large-blue-button\"\nconst c1: ClassName = \"small-red-button\"\nconst c2: ClassName = \"large-blue-button\""

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

// Test template literal with generic
func TestTemplateLiteralGeneric(t *testing.T) {
	input := "type WithId<T> = `${T}Id`\n\ntype UserId = WithId<\"user\">  -- should be \"userId\"\ntype PostId = WithId<\"post\">  -- should be \"postId\""

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

// Test template literal parsing
func TestTemplateLiteralParsing(t *testing.T) {
	input := "type EmailLocale = `${string}_${string}@${string}.${string}`"

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
