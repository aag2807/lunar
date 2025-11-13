package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestPrivatePropertyAccessFromOutside(t *testing.T) {
	input := `
class BankAccount
	private balance: number

	constructor(initial: number)
		self.balance = initial
	end

	getBalance(): number
		return self.balance
	end
end

local account: BankAccount = BankAccount(100)
local b: number = account.balance
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: cannot access private property
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Cannot access private property 'balance' of class 'BankAccount'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestPrivateMethodAccessFromOutside(t *testing.T) {
	input := `
class BankAccount
	private validateAmount(amount: number): boolean
		return amount > 0
	end

	deposit(amount: number): void
		if self.validateAmount(amount) then
			local x: number = 1
		end
	end
end

local account: BankAccount = BankAccount()
local valid: boolean = account.validateAmount(50)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: cannot access private method
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Cannot access private method 'validateAmount' of class 'BankAccount'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestPrivatePropertyAccessFromSameClass(t *testing.T) {
	input := `
class BankAccount
	private balance: number

	constructor(initial: number)
		self.balance = initial
	end

	getBalance(): number
		return self.balance
	end
end

local account: BankAccount = BankAccount(100)
local b: number = account.getBalance()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have no errors: private member accessed from within same class
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestProtectedPropertyAccessFromChild(t *testing.T) {
	input := `
class Animal
	protected name: string
end

class Dog extends Animal
	bark(): void
		local myName: string = self.name
	end
end

local dog: Dog = Dog()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have no errors: protected member accessed from child class
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestProtectedPropertyAccessFromOutside(t *testing.T) {
	input := `
class Animal
	protected name: string
end

local animal: Animal = Animal()
local n: string = animal.name
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: cannot access protected property from outside
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Cannot access protected property 'name' of class 'Animal'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestPublicPropertyAccessFromAnywhere(t *testing.T) {
	input := `
class Person
	public name: string
	public age: number
end

local person: Person = Person()
local n: string = person.name
local a: number = person.age
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have no errors: public members accessible from anywhere
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestPrivateStaticPropertyAccess(t *testing.T) {
	input := `
class MathUtil
	private static PI: number

	static getPI(): number
		return MathUtil.PI
	end
end

local pi: number = MathUtil.PI
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: cannot access private static property
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Cannot access private static property 'PI' of class 'MathUtil'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestPrivateStaticMethodAccess(t *testing.T) {
	input := `
class Helper
	private static validate(): boolean
		return true
	end

	static doSomething(): void
		local valid: boolean = Helper.validate()
	end
end

local result: boolean = Helper.validate()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: cannot access private static method
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Cannot access private static method 'validate' of class 'Helper'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}
