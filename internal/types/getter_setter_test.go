package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test basic getter declaration
func TestGetterBasic(t *testing.T) {
	input := `
class Point
	_x: number

	function constructor(x: number)
		self._x = x
	end

	get x(): number
		return self._x
	end
end

local p = Point(10)
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

	// Verify getter is registered
	pointClass, ok := checker.classes["Point"]
	if !ok {
		t.Fatal("Expected Point class to be registered")
	}

	if len(pointClass.Getters) != 1 {
		t.Fatalf("Expected 1 getter, got %d", len(pointClass.Getters))
	}

	getter, ok := pointClass.Getters["x"]
	if !ok {
		t.Fatal("Expected getter 'x' to be registered")
	}

	if getter.ReturnType != Number {
		t.Errorf("Expected getter return type to be number, got %v", getter.ReturnType)
	}
}

// Test basic setter declaration
func TestSetterBasic(t *testing.T) {
	input := `
class Point
	_x: number

	function constructor(x: number)
		self._x = x
	end

	set x(value: number)
		self._x = value
	end
end

local p = Point(10)
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

	// Verify setter is registered
	pointClass, ok := checker.classes["Point"]
	if !ok {
		t.Fatal("Expected Point class to be registered")
	}

	if len(pointClass.Setters) != 1 {
		t.Fatalf("Expected 1 setter, got %d", len(pointClass.Setters))
	}

	setter, ok := pointClass.Setters["x"]
	if !ok {
		t.Fatal("Expected setter 'x' to be registered")
	}

	if len(setter.Parameters) != 1 {
		t.Errorf("Expected setter to have 1 parameter, got %d", len(setter.Parameters))
	}

	if setter.Parameters[0] != Number {
		t.Errorf("Expected setter parameter to be number, got %v", setter.Parameters[0])
	}
}

// Test getter and setter together
func TestGetterSetterPair(t *testing.T) {
	input := `
class Counter
	_count: number

	function constructor()
		self._count = 0
	end

	get count(): number
		return self._count
	end

	set count(value: number)
		self._count = value
	end

	function increment(): void
		self._count = self._count + 1
	end
end

local c = Counter()
c.increment()
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

	// Verify both getter and setter are registered
	counterClass, ok := checker.classes["Counter"]
	if !ok {
		t.Fatal("Expected Counter class to be registered")
	}

	if len(counterClass.Getters) != 1 {
		t.Errorf("Expected 1 getter, got %d", len(counterClass.Getters))
	}

	if len(counterClass.Setters) != 1 {
		t.Errorf("Expected 1 setter, got %d", len(counterClass.Setters))
	}

	// Verify they have matching names
	_, hasGetter := counterClass.Getters["count"]
	_, hasSetter := counterClass.Setters["count"]

	if !hasGetter {
		t.Error("Expected getter 'count' to be registered")
	}
	if !hasSetter {
		t.Error("Expected setter 'count' to be registered")
	}
}

// Test multiple getters and setters
func TestMultipleGettersSetters(t *testing.T) {
	input := `
class Rectangle
	_width: number
	_height: number

	function constructor(w: number, h: number)
		self._width = w
		self._height = h
	end

	get width(): number
		return self._width
	end

	set width(value: number)
		self._width = value
	end

	get height(): number
		return self._height
	end

	set height(value: number)
		self._height = value
	end

	get area(): number
		return self._width * self._height
	end
end

local r = Rectangle(10, 20)
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

	// Verify all getters and setters are registered
	rectClass, ok := checker.classes["Rectangle"]
	if !ok {
		t.Fatal("Expected Rectangle class to be registered")
	}

	if len(rectClass.Getters) != 3 {
		t.Errorf("Expected 3 getters, got %d", len(rectClass.Getters))
	}

	if len(rectClass.Setters) != 2 {
		t.Errorf("Expected 2 setters, got %d", len(rectClass.Setters))
	}
}

// Test getter with wrong return type in body
func TestGetterTypeError(t *testing.T) {
	input := `
class Point
	_x: number

	get x(): number
		return "not a number"
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

	// Should have a type error for returning string when number expected
	if len(errors) == 0 {
		t.Error("Expected type error for wrong return type in getter")
	}
}

// Test visibility on getters/setters
func TestGetterSetterVisibility(t *testing.T) {
	input := `
class Account
	_balance: number

	function constructor()
		self._balance = 0
	end

	private get balance(): number
		return self._balance
	end

	protected set balance(value: number)
		self._balance = value
	end
end

local a = Account()
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

	// Verify visibility is tracked
	accountClass, ok := checker.classes["Account"]
	if !ok {
		t.Fatal("Expected Account class to be registered")
	}

	if accountClass.GetterVisibility["balance"] != "private" {
		t.Errorf("Expected getter visibility to be private, got %s", accountClass.GetterVisibility["balance"])
	}

	if accountClass.SetterVisibility["balance"] != "protected" {
		t.Errorf("Expected setter visibility to be protected, got %s", accountClass.SetterVisibility["balance"])
	}
}
