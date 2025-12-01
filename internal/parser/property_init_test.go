package parser

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"testing"
)

func TestPropertyWithInitialValue(t *testing.T) {
	input := `
class Person
	name = "John"
	age: number = 25
	city: string
end
`

	l := lexer.New(input)
	p := New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	if len(statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(statements))
	}

	classStmt, ok := statements[0].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("Expected ClassDeclaration, got %T", statements[0])
	}

	if len(classStmt.Properties) != 3 {
		t.Fatalf("Expected 3 properties, got %d", len(classStmt.Properties))
	}

	// Test property with initial value but no type
	if classStmt.Properties[0].Name.Value != "name" {
		t.Errorf("Expected property name 'name', got '%s'", classStmt.Properties[0].Name.Value)
	}
	if classStmt.Properties[0].Type != nil {
		t.Errorf("Expected no type annotation for 'name' property")
	}
	if classStmt.Properties[0].Value == nil {
		t.Errorf("Expected initial value for 'name' property")
	}

	// Test property with both type and initial value
	if classStmt.Properties[1].Name.Value != "age" {
		t.Errorf("Expected property name 'age', got '%s'", classStmt.Properties[1].Name.Value)
	}
	if classStmt.Properties[1].Type == nil {
		t.Errorf("Expected type annotation for 'age' property")
	}
	if classStmt.Properties[1].Value == nil {
		t.Errorf("Expected initial value for 'age' property")
	}

	// Test property with type but no initial value
	if classStmt.Properties[2].Name.Value != "city" {
		t.Errorf("Expected property name 'city', got '%s'", classStmt.Properties[2].Name.Value)
	}
	if classStmt.Properties[2].Type == nil {
		t.Errorf("Expected type annotation for 'city' property")
	}
	if classStmt.Properties[2].Value != nil {
		t.Errorf("Expected no initial value for 'city' property")
	}
}

func TestPropertyVisibilityWithInitialValue(t *testing.T) {
	input := `
class MyClass
	private secret = 42
	public name: string = "test"
	protected internal: boolean = true
end
`

	l := lexer.New(input)
	p := New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	classStmt := statements[0].(*ast.ClassDeclaration)

	if len(classStmt.Properties) != 3 {
		t.Fatalf("Expected 3 properties, got %d", len(classStmt.Properties))
	}

	// Check visibility modifiers are preserved
	if classStmt.Properties[0].Visibility != "private" {
		t.Errorf("Expected visibility 'private', got '%s'", classStmt.Properties[0].Visibility)
	}
	if classStmt.Properties[1].Visibility != "public" {
		t.Errorf("Expected visibility 'public', got '%s'", classStmt.Properties[1].Visibility)
	}
	if classStmt.Properties[2].Visibility != "protected" {
		t.Errorf("Expected visibility 'protected', got '%s'", classStmt.Properties[2].Visibility)
	}

	// Check all have initial values
	for i, prop := range classStmt.Properties {
		if prop.Value == nil {
			t.Errorf("Property %d should have initial value", i)
		}
	}
}

func TestStaticPropertyWithInitialValue(t *testing.T) {
	input := `
class Counter
	static count = 0
	static maxCount: number = 100
end
`

	l := lexer.New(input)
	p := New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	classStmt := statements[0].(*ast.ClassDeclaration)

	if len(classStmt.Properties) != 2 {
		t.Fatalf("Expected 2 properties, got %d", len(classStmt.Properties))
	}

	// Check both are static
	if !classStmt.Properties[0].IsStatic {
		t.Errorf("Expected first property to be static")
	}
	if !classStmt.Properties[1].IsStatic {
		t.Errorf("Expected second property to be static")
	}

	// Check both have initial values
	if classStmt.Properties[0].Value == nil {
		t.Errorf("Expected initial value for 'count' property")
	}
	if classStmt.Properties[1].Value == nil {
		t.Errorf("Expected initial value for 'maxCount' property")
	}
}
