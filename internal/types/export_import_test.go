package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
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

// An import that does not resolve is an error; it used to bind its names
// as 'any' silently, so a cross-file mistake compiled clean. The names are
// still bound as 'any' so they do not cascade into more errors.
func TestImportNonExistentModule(t *testing.T) {
	checker := NewChecker()
	errs := checker.Check(parseFile(t, "main.lunar", "import { greet } from \"nonexistent_module\"\ngreet()\n"))
	if len(errs) != 1 || errs[0].Message != "Cannot find module 'nonexistent_module'" {
		t.Fatalf("got %v", messages(errs))
	}
	if typ, ok := checker.env.Get("greet"); !ok || typ != Any {
		t.Errorf("greet should be bound as any, got %v", typ)
	}
}

func TestImportWildcardNonExistentModule(t *testing.T) {
	errs := NewChecker().Check(parseFile(t, "main.lunar", "import * as m from \"nonexistent_module\"\n"))
	if len(errs) != 1 || errs[0].Message != "Cannot find module 'nonexistent_module'" {
		t.Fatalf("got %v", messages(errs))
	}
}

// `import * from` named nothing to bind the module to; the checker bound
// every export while codegen bound one local, so the two disagreed.
func TestWildcardImportNeedsAName(t *testing.T) {
	p := parser.New(lexer.New("import * from \"mod\"\n"))
	p.Parse()
	if len(p.Errors()) == 0 || !strings.Contains(p.Errors()[0].Message, "import * as name") {
		t.Fatalf("got %v", p.Errors())
	}
}
