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
