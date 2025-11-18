package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestGenericClassDeclaration(t *testing.T) {
	input := `
class Box<T>
	private value: T

	constructor(val: T)
		self.value = val
	end

	getValue(): T
		return self.value
	end

	setValue(val: T): void
		self.value = val
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
}

func TestGenericClassInstantiation(t *testing.T) {
	input := `
class Box<T>
	private value: T

	constructor(val: T)
		self.value = val
	end

	getValue(): T
		return self.value
	end
end

local stringBox: Box<string> = Box<string>("hello")
local numberBox: Box<number> = Box<number>(42)
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

func TestGenericClassMultipleParameters(t *testing.T) {
	input := `
class Pair<K, V>
	private key: K
	private value: V

	constructor(k: K, v: V)
		self.key = k
		self.value = v
	end

	getKey(): K
		return self.key
	end

	getValue(): V
		return self.value
	end
end

local pair: Pair<string, number> = Pair<string, number>("age", 25)
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

func TestGenericClassStack(t *testing.T) {
	input := `
class Stack<T>
	private items: T[]

	constructor()
		self.items = {}
	end

	push(item: T): void
		table.insert(self.items, item)
	end

	pop(): T?
		if #self.items > 0 then
			return table.remove(self.items)
		end
		return nil
	end

	isEmpty(): boolean
		return #self.items == 0
	end
end

local numberStack: Stack<number> = Stack<number>()
numberStack.push(1)
numberStack.push(2)
local value: number? = numberStack.pop()
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

func TestGenericClassWithInheritance(t *testing.T) {
	input := `
class Container<T>
	protected value: T

	constructor(val: T)
		self.value = val
	end

	getValue(): T
		return self.value
	end
end

class PrintableContainer<T> extends Container<T>
	print(): void
		local x: T = self.value
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
}

func TestGenericClassTypeError(t *testing.T) {
	input := `
class Box<T>
	private value: T

	constructor(val: T)
		self.value = val
	end
end

local box: Box<string> = Box<string>(42)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have error: passing number to Box<string>
	if len(errors) < 1 {
		t.Errorf("Expected at least 1 type error, got %d:", len(errors))
	}
}
