package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestGenericFunctionDeclaration(t *testing.T) {
	input := `
function identity<T>(value: T): T
	return value
end

local str: string = identity<string>("hello")
local num: number = identity<number>(42)
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

func TestGenericFunctionWithMultipleParameters(t *testing.T) {
	input := `
function map<T, U>(array: T[], fn: (item: T) => U): U[]
	local result: U[] = {}
	for _, item in ipairs(array) do
		table.insert(result, fn(item))
	end
	return result
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

func TestGenericFunctionInference(t *testing.T) {
	input := `
function identity<T>(value: T): T
	return value
end

local str: string = identity("hello")
local num: number = identity(42)
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
		t.Errorf("Expected no type errors (type inference), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestGenericFunctionFilter(t *testing.T) {
	input := `
function filter<T>(array: T[], predicate: (item: T) => boolean): T[]
	local result: T[] = {}
	for _, item in ipairs(array) do
		if predicate(item) then
			table.insert(result, item)
		end
	end
	return result
end

function isEven(n: number): boolean
	return n % 2 == 0
end

local numbers: number[] = {1, 2, 3, 4, 5, 6}
local evens: number[] = filter<number>(numbers, isEven)
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

func TestGenericFunctionReduce(t *testing.T) {
	input := `
function reduce<T, U>(array: T[], fn: (acc: U, item: T) => U, initial: U): U
	local result: U = initial
	for _, item in ipairs(array) do
		result = fn(result, item)
	end
	return result
end

function add(acc: number, item: number): number
	return acc + item
end

local numbers: number[] = {1, 2, 3, 4, 5}
local sum: number = reduce<number, number>(numbers, add, 0)
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

func TestGenericFunctionFind(t *testing.T) {
	input := `
function find<T>(array: T[], predicate: (item: T) => boolean): T?
	for _, item in ipairs(array) do
		if predicate(item) then
			return item
		end
	end
	return nil
end

function isPositive(n: number): boolean
	return n > 0
end

local numbers: number[] = {-1, -2, 3, -4, 5}
local first: number? = find<number>(numbers, isPositive)
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

func TestGenericFunctionSwap(t *testing.T) {
	input := `
function swap<T, U>(a: T, b: U): (U, T)
	return b, a
end

local x: number, y: string = swap<string, number>("hello", 42)
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

func TestGenericFunctionTypeError(t *testing.T) {
	input := `
function identity<T>(value: T): T
	return value
end

local str: string = identity<number>(42)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have error: assigning number to string
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
