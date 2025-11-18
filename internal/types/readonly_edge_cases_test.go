package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestReadonlyPropertyCannotReassign(t *testing.T) {
	input := `
class Config
	readonly maxSize: number

	function constructor()
		self.maxSize = 100
	end

	function tryChange(): void
		self.maxSize = 200
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
		t.Error("Expected type error for reassigning readonly property, got none")
	}

	foundError := false
	for _, err := range errors {
		if err.Message == "Cannot assign to readonly property 'maxSize'" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'Cannot assign to readonly property' error, got: %v", errors)
	}
}

func TestReadonlyExternalAssignment(t *testing.T) {
	input := `
class Container
	readonly value: number

	function constructor()
		self.value = 42
	end
end

local c = Container()
c.value = 100
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
		t.Error("Expected type error for external assignment to readonly property, got none")
	}

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

func TestReadonlyMultipleProperties(t *testing.T) {
	input := `
class Point
	readonly x: number
	readonly y: number

	function constructor(x: number, y: number)
		self.x = x
		self.y = y
	end
end

local p = Point(10, 20)
local xVal: number = p.x
local yVal: number = p.y
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

func TestReadonlyMixedWithMutable(t *testing.T) {
	input := `
class User
	readonly id: number
	name: string

	function constructor(id: number, name: string)
		self.id = id
		self.name = name
	end

	function rename(newName: string): void
		self.name = newName
	end
end

local user = User(1, "Alice")
user.name = "Bob"
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

func TestReadonlyWithInheritance(t *testing.T) {
	input := `
class Base
	readonly id: number

	function constructor(id: number)
		self.id = id
	end
end

class Derived extends Base
	function constructor(id: number)
		super(id)
	end

	function tryModify(): void
		self.id = 999
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
		t.Error("Expected type error for modifying inherited readonly property, got none")
	}
}

func TestReadonlyStaticProperty(t *testing.T) {
	input := `
class Constants
	static readonly PI: number = 3.14159
	static readonly E: number = 2.71828
end

local pi: number = Constants.PI
local e: number = Constants.E
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

func TestReadonlyStaticCannotAssign(t *testing.T) {
	input := `
class Constants
	static readonly VERSION: string = "1.0"
end

Constants.VERSION = "2.0"
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
		t.Error("Expected type error for assigning to readonly static property, got none")
	}
}

func TestReadonlyInterfaceProperty(t *testing.T) {
	input := `
interface ReadonlyConfig
	readonly maxConnections: number
	readonly timeout: number
end

class ServerConfig implements ReadonlyConfig
	readonly maxConnections: number
	readonly timeout: number

	function constructor(max: number, time: number)
		self.maxConnections = max
		self.timeout = time
	end
end

local config: ReadonlyConfig = ServerConfig(100, 5000)
local max: number = config.maxConnections
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

func TestReadonlyIncrementDecrement(t *testing.T) {
	input := `
class Counter
	readonly value: number

	function constructor()
		self.value = 0
	end

	function tryIncrement(): void
		self.value = self.value + 1
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
		t.Error("Expected type error for incrementing readonly property, got none")
	}
}

func TestReadonlyArrayProperty(t *testing.T) {
	input := `
class Data
	readonly items: number[]

	function constructor()
		self.items = {1, 2, 3}
	end
end

local data = Data()
local items: number[] = data.items
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

func TestReadonlyArrayPropertyCannotReassign(t *testing.T) {
	input := `
class Data
	readonly items: number[]

	function constructor()
		self.items = {1, 2, 3}
	end

	function reset(): void
		self.items = {}
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
		t.Error("Expected type error for reassigning readonly array property, got none")
	}
}

func TestReadonlyWithGenericClass(t *testing.T) {
	input := `
class Box<T>
	readonly value: T

	function constructor(value: T)
		self.value = value
	end
end

local numBox = Box<number>(42)
local num: number = numBox.value

local strBox = Box<string>("hello")
local str: string = strBox.value
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

func TestReadonlyProtectedProperty(t *testing.T) {
	input := `
class Base
	protected readonly secret: string

	function constructor()
		self.secret = "hidden"
	end
end

class Derived extends Base
	function constructor()
		super()
	end

	function getSecret(): string
		return self.secret
	end

	function tryChangeSecret(): void
		self.secret = "exposed"
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

	// Should have error about modifying readonly property
	if len(errors) == 0 {
		t.Error("Expected type error for modifying readonly protected property, got none")
	}
}

func TestReadonlyPrivateProperty(t *testing.T) {
	input := `
class Secure
	private readonly token: string

	function constructor(token: string)
		self.token = token
	end

	function getToken(): string
		return self.token
	end
end

local secure = Secure("abc123")
local token: string = secure.getToken()
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
