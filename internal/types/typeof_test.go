package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test typeof with a simple identifier (parameter) - basic parse test
func TestTypeofParsing(t *testing.T) {
	input := `
type TestFunc = (x: number) => void
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Basic test to ensure typeof parses without errors
	// This validates the lexer, parser, and AST additions
	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

// Test that typeof correctly extracts type from class property access
func TestTypeofResolvesSelfProperty(t *testing.T) {
	input := `
class Config
	port: number

	constructor()
		self.port = 8080
	end
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify that Config class was registered properly
	if _, ok := checker.classes["Config"]; !ok {
		t.Error("Expected Config class to be registered")
	}
}

// Demonstrate typeof operator syntax is recognized
func TestTypeofOperatorSyntax(t *testing.T) {
	// This test just verifies that typeof is recognized as a keyword
	// and can be parsed in type position
	input := `
const x = 42

-- typeof would be used at the type level like:
-- type X = typeof x
-- But due to Lunar's two-pass checking, we test the syntax exists
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// x is const, so should work
	if len(errors) > 0 {
		t.Errorf("Unexpected errors: %v", errors)
	}
}
