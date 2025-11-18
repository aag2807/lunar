package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestKeyofInterface(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type PersonKeys = keyof Person
local key: PersonKeys = "name"
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

func TestKeyofClass(t *testing.T) {
	input := `
class User
	public name: string
	public email: string
	public age: number

	public function greet(): void
		local x: number = 1
	end
end

type UserKeys = keyof User
local key1: UserKeys = "name"
local key2: UserKeys = "email"
local key3: UserKeys = "greet"
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

func TestKeyofInvalidKey(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type PersonKeys = keyof Person
local key: PersonKeys = "invalid"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have an error: "invalid" is not a key of Person
	if len(errors) == 0 {
		t.Error("Expected type error for invalid key")
	}
}

func TestKeyofTable(t *testing.T) {
	input := `
type StringTable = table<string, number>
type TableKeys = keyof StringTable
local key: TableKeys = "anyString"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// keyof table<string, number> = string
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestKeyofTuple(t *testing.T) {
	input := `
type Tuple = (string, number, boolean)
type TupleKeys = keyof Tuple
local key: TupleKeys = 0
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// keyof tuple returns numeric literal types for indices
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestKeyofEmptyInterface(t *testing.T) {
	input := `
interface Empty
end

type EmptyKeys = keyof Empty
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// keyof empty interface = never
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestKeyofPrimitiveType(t *testing.T) {
	input := `
type NumberKeys = keyof number
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// keyof primitive type = never
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestKeyofInFunctionParameter(t *testing.T) {
	input := `
interface Config
	host: string
	port: number
end

function updateConfig(key: keyof Config, value: any): void
	local x: number = 1
end

updateConfig("host", "localhost")
updateConfig("port", 8080)
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

func TestKeyofWithGenericClass(t *testing.T) {
	input := `
class Box<T>
	public value: T
	public id: number

	public function getValue(): T
		return self.value
	end
end

type BoxKeys = keyof Box<string>
local key1: BoxKeys = "value"
local key2: BoxKeys = "id"
local key3: BoxKeys = "getValue"
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
