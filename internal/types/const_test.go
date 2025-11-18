package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestConstDeclaration(t *testing.T) {
	input := `
const MAX_SIZE: number = 100
const APP_NAME: string = "Lunar"
const DEBUG: boolean = true
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

func TestConstReassignment(t *testing.T) {
	input := `
const MAX_SIZE: number = 100
MAX_SIZE = 200
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have error: cannot reassign const
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error (const reassignment), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	if !contains(errors[0].Message, "const") && !contains(errors[0].Message, "Cannot assign") {
		t.Errorf("Expected error about const reassignment, got: %s", errors[0].Message)
	}
}

func TestConstInFunction(t *testing.T) {
	input := `
function calculate(): number
	const MULTIPLIER: number = 10
	local value: number = 5
	return value * MULTIPLIER
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

func TestConstReassignmentInFunction(t *testing.T) {
	input := `
function test(): void
	const VALUE: number = 10
	VALUE = 20
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

	// Should have error: cannot reassign const
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error (const reassignment), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestConstWithTypeInference(t *testing.T) {
	input := `
const MAX_SIZE = 100
const APP_NAME = "Lunar"
const DEBUG = true
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

func TestLocalReassignment(t *testing.T) {
	input := `
local count: number = 0
count = 10
count = 20
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have no errors: local can be reassigned
	if len(errors) > 0 {
		t.Errorf("Expected no type errors (local can be reassigned), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
