package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestIndexedAccessInterface(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type PersonName = Person["name"]
local n: PersonName = "John"
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

func TestIndexedAccessClass(t *testing.T) {
	input := `
class User
	public name: string
	public age: number
	public active: boolean
end

type UserAge = User["age"]
local age: UserAge = 25
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

func TestIndexedAccessUnionOfKeys(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
	email: string
end

type StringFields = Person["name" | "email"]
local field: StringFields = "test@example.com"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Person["name" | "email"] = string | string = string
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessWithKeyof(t *testing.T) {
	input := `
interface Config
	host: string
	port: number
	debug: boolean
end

type ConfigValue = Config[keyof Config]
local value1: ConfigValue = "localhost"
local value2: ConfigValue = 8080
local value3: ConfigValue = true
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Config[keyof Config] = string | number | boolean
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessArray(t *testing.T) {
	input := `
type StringArray = string[]
type ArrayElement = StringArray[number]
local element: ArrayElement = "test"
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// string[number] = string
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessTuple(t *testing.T) {
	input := `
type Pair = (string, number)
type First = Pair[0]
type Second = Pair[1]

local first: First = "hello"
local second: Second = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Pair[0] = string, Pair[1] = number
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessTupleWithNumber(t *testing.T) {
	input := `
type Triple = (string, number, boolean)
type TripleElement = Triple[number]

local elem1: TripleElement = "hello"
local elem2: TripleElement = 42
local elem3: TripleElement = true
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Triple[number] = string | number | boolean
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessTable(t *testing.T) {
	input := `
type StringMap = table<string, number>
type MapValue = StringMap[string]

local value: MapValue = 42
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// table<string, number>[string] = number
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestIndexedAccessNestedObjects(t *testing.T) {
	input := `
interface Address
	street: string
	city: string
	zip: number
end

interface Person
	name: string
	address: Address
end

type AddressType = Person["address"]
type CityType = Person["address"]["city"]

local addr: AddressType
local city: CityType = "NYC"
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

func TestIndexedAccessWithFunction(t *testing.T) {
	input := `
interface API
	getName: () => string
	getAge: () => number
end

type GetNameType = API["getName"]

function test(): GetNameType
	return function(): string
		return "test"
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

	// API["getName"] = () => string
	if len(errors) > 0 {
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
