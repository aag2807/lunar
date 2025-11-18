package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestExportVariable(t *testing.T) {
	input := `
export const MAX_SIZE: number = 100
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

	// Check that the export was tracked
	if len(checker.exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(checker.exports))
	}

	if _, ok := checker.exports["MAX_SIZE"]; !ok {
		t.Error("Expected MAX_SIZE to be exported")
	}
}

func TestExportFunction(t *testing.T) {
	input := `
export function greet(name: string): string
	return "Hello, " .. name
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

	if len(checker.exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(checker.exports))
	}

	if _, ok := checker.exports["greet"]; !ok {
		t.Error("Expected greet to be exported")
	}
}

func TestExportClass(t *testing.T) {
	input := `
export class Calculator
	static function add(a: number, b: number): number
		return a + b
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
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	if len(checker.exports) != 1 {
		t.Errorf("Expected 1 export, got %d", len(checker.exports))
	}

	if _, ok := checker.exports["Calculator"]; !ok {
		t.Error("Expected Calculator to be exported")
	}
}

func TestExportInterface(t *testing.T) {
	input := `
export interface User
	name: string
	age: number
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

	if _, ok := checker.exports["User"]; !ok {
		t.Error("Expected User to be exported")
	}
}

func TestExportEnum(t *testing.T) {
	input := `
export enum Direction
	North
	South
	East
	West
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

	if _, ok := checker.exports["Direction"]; !ok {
		t.Error("Expected Direction to be exported")
	}
}

func TestExportTypeAlias(t *testing.T) {
	input := `
export type UserId = number
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

	if _, ok := checker.exports["UserId"]; !ok {
		t.Error("Expected UserId to be exported")
	}
}

func TestMultipleExports(t *testing.T) {
	input := `
export const MAX_SIZE: number = 100
export const MIN_SIZE: number = 1

export function add(a: number, b: number): number
	return a + b
end

export class MyClass
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

	expectedExports := []string{"MAX_SIZE", "MIN_SIZE", "add", "MyClass"}
	if len(checker.exports) != len(expectedExports) {
		t.Errorf("Expected %d exports, got %d", len(expectedExports), len(checker.exports))
	}

	for _, name := range expectedExports {
		if _, ok := checker.exports[name]; !ok {
			t.Errorf("Expected %s to be exported", name)
		}
	}
}

func TestImportNonExistentModule(t *testing.T) {
	input := `
import { greet } from "nonexistent_module"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	_ = checker.Check(statements)

	// Should not error - module not found is allowed for gradual typing
	// The imported name should be added as 'any' type
	typ, ok := checker.env.Get("greet")
	if !ok {
		t.Error("Expected greet to be in environment")
	}

	if typ != Any {
		t.Errorf("Expected greet to have type 'any', got %s", typ.String())
	}
}

func TestImportWildcardNonExistentModule(t *testing.T) {
	input := `
import * from "nonexistent_module"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	typeErrors := checker.Check(statements)

	// Should error for wildcard imports when module not found
	if len(typeErrors) == 0 {
		t.Error("Expected error for wildcard import of non-existent module")
	}

	foundError := false
	for _, err := range typeErrors {
		if err.Message == "Cannot resolve module 'nonexistent_module'" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected 'Cannot resolve module' error")
	}
}
