package codegen

import "testing"

// `Calculator()` compiles to `Calculator.new()`, so every class needs a new(),
// including one that declares only methods.
func TestClassWithoutConstructorStillGetsNew(t *testing.T) {
	output := generate(t, `
class Calculator
	add(a: number, b: number): number
		return a + b
	end
end

local c = Calculator()
`)

	assertContains(t, output, "function Calculator.new()")
	assertContains(t, output, "setmetatable({}, Calculator)")
	assertContains(t, output, "Calculator.new()")
}

// Type arguments are erased; the callee must survive.
func TestGenericCallKeepsItsCallee(t *testing.T) {
	output := generate(t, `
function map<T, U>(item: T, mapper: (val: T) => U): U
	return mapper(item)
end

local result = map<number, string>(42, tostring)
`)

	assertContains(t, output, "map(42, tostring)")
}

// A generic class instantiation is still a constructor call.
func TestGenericClassInstantiationCallsNew(t *testing.T) {
	output := generate(t, `
class Box<T>
	private value: T

	constructor(value: T)
		self.value = value
	end

	get(): T
		return self.value
	end
end

local b = Box<number>(1)
b.get()
`)

	assertContains(t, output, "Box.new(1)")
	assertContains(t, output, "b:get()")
}
