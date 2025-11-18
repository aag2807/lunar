package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic decorator on function
func TestDecoratorBasic(t *testing.T) {
	input := `
function log(fn: any): any
	return fn
end

@log
function greet(): string
	return "hello"
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

// Test decorator with arguments
func TestDecoratorWithArguments(t *testing.T) {
	input := `
function route(path: string): any
	return function(fn: any): any
		return fn
	end
end

@route("/users")
function getUsers(): string
	return "users"
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

// Test multiple decorators on function
func TestMultipleDecorators(t *testing.T) {
	input := `
function auth(fn: any): any
	return fn
end

function log(fn: any): any
	return fn
end

@auth
@log
function secureEndpoint(): string
	return "secret"
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

// Test decorator on class
func TestDecoratorOnClass(t *testing.T) {
	input := `
function component(cls: any): any
	return cls
end

@component
class Button
	label: string

	function constructor(l: string)
		self.label = l
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

// Test class decorator with arguments
func TestClassDecoratorWithArguments(t *testing.T) {
	input := `
function injectable(name: string): any
	return function(cls: any): any
		return cls
	end
end

@injectable("UserService")
class UserService
	function getUser(): string
		return "user"
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

// Test undefined decorator error
func TestUndefinedDecoratorError(t *testing.T) {
	input := `
@nonexistent
function test(): void
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
		t.Error("Expected error for undefined decorator")
	}
}

// Test decorator on async function
func TestDecoratorOnAsyncFunction(t *testing.T) {
	input := `
function cache(fn: any): any
	return fn
end

@cache
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
