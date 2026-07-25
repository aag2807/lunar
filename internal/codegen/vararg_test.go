package codegen

import "testing"

// Lua's bare vararg passes through untouched and is never packed into a local.
func TestBareVarargIsPassedThrough(t *testing.T) {
	output := generate(t, `
function sum(...: any): number
	return select("#", ...)
end
`)

	assertContains(t, output, "function sum(...)")
	assertNotContains(t, output, "local ... =")
	assertContains(t, output, `select("#", ...)`)
}

// A named rest parameter still becomes '...' in the signature plus a local.
func TestNamedRestParameterIsPacked(t *testing.T) {
	output := generate(t, `
function tagged(label: string, ...args: any): number
	return #args
end
`)

	assertContains(t, output, "function tagged(label, ...)")
	assertContains(t, output, "local args = {...}")
}

// Function expressions used to drop rest parameters entirely.
func TestRestParameterInFunctionExpression(t *testing.T) {
	output := generate(t, `
local wrap = function(...args: any): any
	return #args
end
`)

	assertContains(t, output, "function(...)")
	assertContains(t, output, "local args = {...}")
}

func TestBareVarargInFunctionExpression(t *testing.T) {
	output := generate(t, `
local wrap = function(...: any): any
	return select("#", ...)
end
`)

	assertContains(t, output, "function(...)")
	assertNotContains(t, output, "local ... =")
}

// Class methods take varargs too.
func TestRestParameterInClassMethod(t *testing.T) {
	output := generate(t, `
class Logger
	log(prefix: string, ...args: any): void
		print(prefix, #args)
	end
end
`)

	assertContains(t, output, "function Logger:log(prefix, ...)")
	assertContains(t, output, "local args = {...}")
}

// '...' inside a table constructor is a vararg expression, not a spread.
func TestVarargInsideTableConstructor(t *testing.T) {
	output := generate(t, `
function pack(...: any): any
	return {...}
end
`)

	assertContains(t, output, "{...}")
}
