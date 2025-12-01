package codegen

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

func TestInstanceMethodCallUsesColonSyntax(t *testing.T) {
	input := `
class Person
	name: string

	constructor(n: string)
		self.name = n
	end

	sayHello(): void
		print("Hello " .. self.name)
	end
end

local p = Person("John")
p.sayHello()
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

	// Instance method call should use colon syntax
	if !strings.Contains(output, `p:sayHello()`) {
		t.Errorf("Expected instance method call to use colon syntax: p:sayHello()")
		t.Logf("Output:\n%s", output)
	}

	// Should NOT contain dot syntax for the method call
	if strings.Contains(output, `p.sayHello()`) && !strings.Contains(output, `p:sayHello()`) {
		t.Errorf("Instance method call should not use dot syntax")
	}
}

func TestStaticMethodCallUsesDotSyntax(t *testing.T) {
	input := `
class Utils
	static greet(name: string): string
		return "Hello " .. name
	end
end

local message = Utils.greet("World")
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

	// Static method call should use dot syntax
	if !strings.Contains(output, `Utils.greet("World")`) {
		t.Errorf("Expected static method call to use dot syntax: Utils.greet(\"World\")")
		t.Logf("Output:\n%s", output)
	}

	// Should NOT contain colon syntax for static method call
	if strings.Contains(output, `Utils:greet`) {
		t.Errorf("Static method call should not use colon syntax")
	}
}

func TestMixedInstanceAndStaticMethodCalls(t *testing.T) {
	input := `
class Counter
	count: number

	constructor()
		self.count = 0
	end

	increment(): void
		self.count = self.count + 1
	end

	static reset(c: Counter): void
		c.count = 0
	end
end

local c = Counter()
c.increment()
Counter.reset(c)
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

	// Instance method should use colon
	if !strings.Contains(output, `c:increment()`) {
		t.Errorf("Expected instance method call to use colon syntax: c:increment()")
	}

	// Static method should use dot
	if !strings.Contains(output, `Counter.reset(c)`) {
		t.Errorf("Expected static method call to use dot syntax: Counter.reset(c)")
	}

	// Check that reset doesn't use colon (it's static)
	if strings.Contains(output, `Counter:reset`) {
		t.Errorf("Static method should not use colon syntax")
	}
}

func TestMethodCallWithArguments(t *testing.T) {
	input := `
class Calculator
	add(a: number, b: number): number
		return a + b
	end
end

local calc = Calculator()
local result = calc.add(5, 3)
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

	// Instance method call with arguments should use colon
	if !strings.Contains(output, `calc:add(5, 3)`) {
		t.Errorf("Expected instance method call with arguments to use colon syntax: calc:add(5, 3)")
		t.Logf("Output:\n%s", output)
	}
}

func TestMethodDefinitionsUseCorrectSyntax(t *testing.T) {
	input := `
class Example
	instanceMethod(): void
		print("instance")
	end

	static staticMethod(): void
		print("static")
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

	// Instance method definition should use colon
	if !strings.Contains(output, `function Example:instanceMethod()`) {
		t.Errorf("Expected instance method definition to use colon syntax")
	}

	// Static method definition should use dot
	if !strings.Contains(output, `function Example.staticMethod()`) {
		t.Errorf("Expected static method definition to use dot syntax")
	}
}

func TestPropertyAccessDoesNotChangeToColon(t *testing.T) {
	input := `
class Person
	name: string

	constructor(n: string)
		self.name = n
	end
end

local p = Person("John")
local n = p.name
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

	// Property access should remain as dot syntax (not a function call)
	if !strings.Contains(output, `p.name`) {
		t.Errorf("Expected property access to use dot syntax: p.name")
	}

	// Should NOT have colon for property access
	// (This checks that we only change calls, not all dot accesses)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "local n =") && strings.Contains(line, "p:name") {
			t.Errorf("Property access should not use colon syntax")
		}
	}
}
