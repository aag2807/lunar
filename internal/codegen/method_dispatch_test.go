package codegen

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

// generate is a helper that compiles a snippet without type checking, which is
// how these tests exercise the generator's own receiver inference.
func generate(t *testing.T, input string) string {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	return New().Generate(statements)
}

func assertContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Errorf("expected output to contain %q\ngot:\n%s", want, output)
	}
}

func assertNotContains(t *testing.T, output, unwanted string) {
	t.Helper()
	if strings.Contains(output, unwanted) {
		t.Errorf("expected output NOT to contain %q\ngot:\n%s", unwanted, output)
	}
}

// Library functions are plain fields; calling them with ':' would pass the
// library table itself as the first argument.
func TestLibraryCallsUseDotSyntax(t *testing.T) {
	output := generate(t, `
local nums = {3, 1, 2}
table.insert(nums, 4)
table.sort(nums)
local x = math.floor(1.5)
local s = string.rep("a", 3)
`)

	for _, want := range []string{
		"table.insert(nums, 4)",
		"table.sort(nums)",
		"math.floor(1.5)",
		`string.rep("a", 3)`,
	} {
		assertContains(t, output, want)
	}

	for _, unwanted := range []string{"table:insert", "table:sort", "math:floor", "string:rep"} {
		assertNotContains(t, output, unwanted)
	}
}

// A receiver whose type annotation names a class dispatches through ':'.
func TestAnnotatedInstanceUsesColonSyntax(t *testing.T) {
	output := generate(t, `
class Greeter
	greet(): string
		return "hi"
	end
end

local g: Greeter = Greeter()
g.greet()
`)

	assertContains(t, output, "g:greet()")
}

// Parameters carrying a class type are instances too.
func TestClassTypedParameterUsesColonSyntax(t *testing.T) {
	output := generate(t, `
class Greeter
	greet(): string
		return "hi"
	end
end

function useIt(g: Greeter): string
	return g.greet()
end
`)

	assertContains(t, output, "g:greet()")
}

// Parameter tracking must not leak past the function that declared it.
func TestParameterReceiverIsScopedToItsFunction(t *testing.T) {
	output := generate(t, `
class Greeter
	greet(): string
		return "hi"
	end
end

function useIt(lib: Greeter): string
	return lib.greet()
end

local lib = {}
lib.greet()
`)

	assertContains(t, output, "lib:greet()")
	assertContains(t, output, "lib.greet()")
}

// Strings dispatch their methods through the string metatable.
func TestStringReceiverUsesColonSyntax(t *testing.T) {
	output := generate(t, `
local s: string = "a,b"
s.upper()
local literal = "hello"
literal.upper()
`)

	assertContains(t, output, "s:upper()")
	assertContains(t, output, "literal:upper()")
}

// self and super always take an implicit self.
func TestSelfCallsUseColonSyntax(t *testing.T) {
	output := generate(t, `
class Greeter
	greet(): string
		return "hi"
	end

	loud(): string
		return self.greet()
	end
end
`)

	assertContains(t, output, "self:greet()")
}

// Explicit Lua method syntax in the source is preserved verbatim.
func TestExplicitColonSyntaxIsPreserved(t *testing.T) {
	output := generate(t, `
local s = "a,b"
local up = s:upper()
local t = {}
local ok = t:custom(1)
`)

	assertContains(t, output, "s:upper()")
	assertContains(t, output, "t:custom(1)")
}

// An unknown receiver defaults to '.', which is what Lua modules and plain
// tables need; guessing ':' silently corrupts the argument list.
func TestUnknownReceiverDefaultsToDotSyntax(t *testing.T) {
	output := generate(t, `
local mod = require("mod")
mod.doThing(1)
`)

	assertContains(t, output, "mod.doThing(1)")
	assertNotContains(t, output, "mod:doThing")
}

// The element of a typed array is an instance, so its methods take self.
func TestLoopElementUsesColonSyntax(t *testing.T) {
	output := generate(t, `
class Ball
	draw(): void
	end
end

local balls: Ball[] = {}

for _, ball in ipairs(balls) do
	ball.draw()
end
`)

	assertContains(t, output, "ball:draw()")
}
