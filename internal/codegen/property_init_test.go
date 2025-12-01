package codegen

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

func TestInstancePropertyInitialization(t *testing.T) {
	input := `
class Person
	name = "John"
	age: number = 25
	city: string
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	g := New()
	output := g.Generate(statements)

	// Check that the constructor initializes all instance properties
	if !strings.Contains(output, `self.name = "John"`) {
		t.Errorf("Expected instance property 'name' to be initialized with \"John\"")
	}
	if !strings.Contains(output, `self.age = 25`) {
		t.Errorf("Expected instance property 'age' to be initialized with 25")
	}
	if !strings.Contains(output, `self.city = nil`) {
		t.Errorf("Expected instance property 'city' to be initialized with nil")
	}

	// Check that properties are initialized in the new() function
	if !strings.Contains(output, "function Person.new()") {
		t.Errorf("Expected Person.new() function to be generated")
	}
}

func TestPrivateAndPublicPropertiesGenerated(t *testing.T) {
	input := `
class MyClass
	private secret = 42
	public name: string = "test"
	protected value: boolean = true
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	g := New()
	output := g.Generate(statements)

	// All properties should be generated regardless of visibility
	if !strings.Contains(output, `self.secret = 42`) {
		t.Errorf("Expected private property 'secret' to be generated")
	}
	if !strings.Contains(output, `self.name = "test"`) {
		t.Errorf("Expected public property 'name' to be generated")
	}
	if !strings.Contains(output, `self.value = true`) {
		t.Errorf("Expected protected property 'value' to be generated")
	}
}

func TestStaticPropertyInitialization(t *testing.T) {
	input := `
class Counter
	static count = 0
	static maxCount: number = 100
	static name: string
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	g := New()
	output := g.Generate(statements)

	// Static properties should be on the class table, not in new()
	if !strings.Contains(output, `Counter.count = 0`) {
		t.Errorf("Expected static property 'count' to be initialized")
	}
	if !strings.Contains(output, `Counter.maxCount = 100`) {
		t.Errorf("Expected static property 'maxCount' to be initialized")
	}
	if !strings.Contains(output, `Counter.name = nil`) {
		t.Errorf("Expected static property 'name' to be initialized with nil")
	}

	// Static properties should NOT be in the constructor
	constructorStart := strings.Index(output, "function Counter.new()")
	constructorEnd := strings.Index(output[constructorStart:], "end")
	if constructorStart != -1 && constructorEnd != -1 {
		constructor := output[constructorStart : constructorStart+constructorEnd]
		if strings.Contains(constructor, "Counter.count") {
			t.Errorf("Static properties should not be in constructor")
		}
	}
}

func TestMixedInstanceAndStaticProperties(t *testing.T) {
	input := `
class Example
	instanceProp = 1
	static staticProp = 2

	constructor()
	end
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	g := New()
	output := g.Generate(statements)

	// Instance property should be in constructor
	if !strings.Contains(output, `self.instanceProp = 1`) {
		t.Errorf("Expected instance property to be initialized in constructor")
	}

	// Static property should be on class table
	if !strings.Contains(output, `Example.staticProp = 2`) {
		t.Errorf("Expected static property to be on class table")
	}
}

func TestClassWithNoConstructorButWithProperties(t *testing.T) {
	input := `
class NoConstructor
	name = "test"
	value = 42
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser had %d errors:", len(p.Errors()))
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
	}

	g := New()
	output := g.Generate(statements)

	// Should generate a constructor even if none was defined
	if !strings.Contains(output, "function NoConstructor.new()") {
		t.Errorf("Expected constructor to be generated for class with properties")
	}

	// Properties should be initialized
	if !strings.Contains(output, `self.name = "test"`) {
		t.Errorf("Expected property 'name' to be initialized")
	}
	if !strings.Contains(output, `self.value = 42`) {
		t.Errorf("Expected property 'value' to be initialized")
	}
}
