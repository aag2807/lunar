package types

import (
	"testing"

	"lunar/internal/lexer"
	"lunar/internal/parser"
)

// Every async mistake has the code tools rely on, "async".
func TestAsyncErrorsHaveTheirCode(t *testing.T) {
	errs := checkSource(t, `
declare async function wait(seconds: number): void
function update(): void
	wait(1)
	await wait(1)
end
local cb: function(): void = async function() end
`)
	if len(errs) != 3 {
		t.Fatalf("got %v", messages(errs))
	}
	for _, e := range errs {
		if e.Code != "async" {
			t.Errorf("%q has code %q", e.Message, e.Code)
		}
	}
}

// An async function's type says so, and keeps it through assignment.
func TestAsyncFunctionType(t *testing.T) {
	p := parser.New(lexer.New(`
async function fetch(name: string): number
	return 1
end
local f = fetch
`))
	statements := p.Parse()
	c := NewChecker("../../stdlib")
	errs := c.Check(statements)
	if len(errs) != 0 {
		t.Fatalf("got %v", messages(errs))
	}
	f, _ := c.GetEnv().Get("f")
	if got := f.String(); got != "async (string) -> number" {
		t.Errorf("type of f: %s", got)
	}
}

// An async member where an interface has a plain one is an async error,
// with its code, wherever the value goes: a class instance, a table
// literal's field, a table held in a name, a returned value.
func TestStructuralAsyncMismatchHasItsCode(t *testing.T) {
	errs := checkSource(t, `
interface Mover
	move(): void
end
interface Ticker
	tick: () => void
end
class Walker
	async move(): void
	end
end
async function later(): void
end
local m: Mover = Walker()
local function use(x: Mover): void
end
use(Walker())
local t: Ticker = { tick = later }
local r = { tick = later }
local u: Ticker = r
local function make(): Ticker
	return r
end
interface Runner extends Mover
	speed: number
end
class Sprinter
	speed: number = 2
	async move(): void
	end
end
local s: Runner = Sprinter()
`)
	if len(errs) != 6 {
		t.Fatalf("want 6 errors, got %v", messages(errs))
	}
	for _, e := range errs {
		if e.Code != "async" {
			t.Errorf("%d: %q has code %q", e.Line, e.Message, e.Code)
		}
	}
}
