package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic async function declaration
func TestAsyncFunctionBasic(t *testing.T) {
	input := `
async function fetchData(): string
	return "data"
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

// Test async function with await
func TestAsyncFunctionWithAwait(t *testing.T) {
	input := `
async function fetchData(): string
	return "user data"
end

async function fetchUser(): string
	local data = await fetchData()
	return data
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

// Test await expression
func TestAwaitExpression(t *testing.T) {
	input := `
async function computeValue(): number
	return 42
end

async function loadData(): number
	local result = await computeValue()
	return result + 1
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

// Test multiple async functions
func TestMultipleAsyncFunctions(t *testing.T) {
	input := `
async function fetchUsers(): string
	return "users"
end

async function fetchPosts(): string
	return "posts"
end

async function loadAll(): string
	local users = await fetchUsers()
	local posts = await fetchPosts()
	return users
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

// Test async function with parameters
func TestAsyncFunctionWithParams(t *testing.T) {
	input := `
async function fetchById(id: number): string
	return "item"
end

async function processData(data: string, count: number): boolean
	local result = await fetchById(count)
	return true
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

// Test await in control flow
func TestAwaitInControlFlow(t *testing.T) {
	input := `
async function checkCondition(): boolean
	return true
end

async function process(): number
	if await checkCondition() then
		return 1
	else
		return 0
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
