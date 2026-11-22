package types

import (
	"strings"
	"testing"
)

// Two bodies for one global name: the second replaces the first at run time,
// so add(1, 2) returned "12". Overloads are for declarations only.
func TestImplementationOverloadIsAnError(t *testing.T) {
	errs := checkSource(t, `
function add(a: number, b: number): number
    return a + b
end

function add(a: string, b: string): string
    return a .. b
end
`)
	if len(errs) != 1 || errs[0].Message != "Function 'add' is already defined at line 2. Overloads can only be declared with 'declare function'" {
		t.Fatalf("got %v", messages(errs))
	}
}

// Implementing something a declaration file describes is fine when the
// implementation fits the declared signature.
func TestImplementingADeclaredFunction(t *testing.T) {
	for _, err := range checkSource(t, `
declare function greet(name: string): string
function greet(name: string): string
    return "hi " .. name
end
local s: string = greet("ada")
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}

	errs := checkSource(t, `
declare function greet(name: string): string
function greet(name: number): number
    return name
end
`)
	if len(errs) != 1 || !strings.HasPrefix(errs[0].Message, "Function 'greet' does not match its declaration") {
		t.Fatalf("got %v", messages(errs))
	}
}

// A local function may shadow a name from an outer scope, as in Lua.
func TestLocalFunctionMayShadow(t *testing.T) {
	for _, err := range checkSource(t, `
function add(a: number, b: number): number
    return a + b
end
function wrapper(): number
    local function add(a: number): number
        return a
    end
    return add(1)
end
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
