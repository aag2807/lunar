package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic namespace declaration
func TestNamespaceBasic(t *testing.T) {
	input := `
namespace MyApp
	class Server
		host: string

		function constructor(h: string)
			self.host = h
		end

		function getHost(): string
			return self.host
		end
	end

	function createServer(host: string): Server
		return Server(host)
	end
end

local server = MyApp.createServer("localhost")
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

// Test namespace with interface
func TestNamespaceWithInterface(t *testing.T) {
	input := `
namespace Models
	interface User
		name: string
		age: number
	end

	class Person
		name: string
		age: number

		function constructor(n: string, a: number)
			self.name = n
			self.age = a
		end
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
}

// Test namespace with enum
func TestNamespaceWithEnum(t *testing.T) {
	input := `
namespace Config
	enum LogLevel
		Debug,
		Info,
		Warning,
		Error
	end

	function getDefaultLevel(): number
		return 1
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
}

// Test namespace with type alias
func TestNamespaceWithTypeAlias(t *testing.T) {
	input := `
namespace Types
	type Nullable<T> = T | nil
	type StringOrNumber = string | number

	function processValue(v: StringOrNumber): string
		return "processed"
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
}

// Test multiple namespaces
func TestMultipleNamespaces(t *testing.T) {
	input := `
namespace Utils
	function add(a: number, b: number): number
		return a + b
	end
end

namespace Math
	function multiply(a: number, b: number): number
		return a * b
	end
end

local sum = Utils.add(1, 2)
local product = Math.multiply(3, 4)
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
