package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestConstructorParameterPropertyPublic(t *testing.T) {
	input := `
class Person
	function constructor(public name: string, public age: number)
	end
end

local person = Person("Alice", 30)
local name: string = person.name
local age: number = person.age
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

func TestConstructorParameterPropertyPrivate(t *testing.T) {
	input := `
class Secret
	function constructor(private value: string)
	end

	function getValue(): string
		return self.value
	end
end

local secret = Secret("hidden")
local result: string = secret.getValue()
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

func TestConstructorParameterPropertyReadonly(t *testing.T) {
	input := `
class Immutable
	function constructor(public readonly id: number)
	end
end

local obj = Immutable(42)
local id: number = obj.id
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

func TestConstructorParameterPropertyMixed(t *testing.T) {
	input := `
class MixedClass
	function constructor(public name: string, private secret: number, regularParam: boolean)
		-- regularParam is just a parameter, not a property
		local temp = regularParam
	end

	function getSecret(): number
		return self.secret
	end
end

local obj = MixedClass("test", 42, true)
local name: string = obj.name
local secret: number = obj.getSecret()
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

func TestConstructorParameterPropertyReadonlyCannotAssign(t *testing.T) {
	input := `
class ReadonlyTest
	function constructor(public readonly value: number)
	end

	function tryModify(): void
		self.value = 100
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

	if len(errors) == 0 {
		t.Error("Expected type error for assigning to readonly property, got none")
	}

	// Should have error about readonly
	foundError := false
	for _, err := range errors {
		if err.Message == "Cannot assign to readonly property 'value'" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'Cannot assign to readonly property' error, got: %v", errors)
	}
}

func TestConstructorParameterPropertyProtected(t *testing.T) {
	input := `
class Base
	function constructor(protected value: number)
	end
end

class Derived extends Base
	function constructor(value: number)
		super(value)
	end

	function getValue(): number
		return self.value
	end
end

local obj = Derived(42)
local result: number = obj.getValue()
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

func TestConstructorParameterPropertyAllVisibilities(t *testing.T) {
	input := `
class AllVisibilities
	function constructor(
		public pubVal: string,
		private privVal: number,
		protected protVal: boolean
	)
	end

	function getPrivate(): number
		return self.privVal
	end

	function getProtected(): boolean
		return self.protVal
	end
end

local obj = AllVisibilities("public", 42, true)
local pub: string = obj.pubVal
local priv: number = obj.getPrivate()
local prot: boolean = obj.getProtected()
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

func TestConstructorParameterPropertyWithInheritance(t *testing.T) {
	input := `
class Animal
	function constructor(public name: string)
	end
end

class Dog extends Animal
	function constructor(name: string, public breed: string)
		super(name)
	end
end

local dog = Dog("Rex", "German Shepherd")
local name: string = dog.name
local breed: string = dog.breed
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

func TestConstructorParameterPropertyReadonlyFirst(t *testing.T) {
	input := `
class ReadonlyFirst
	function constructor(readonly public name: string)
	end
end

local obj = ReadonlyFirst("test")
local name: string = obj.name
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
