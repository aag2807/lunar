package types

import (
	"strings"
	"testing"
)

// Data files end in `return { ... }`: the chunk itself is a function in Lua.
func TestTopLevelReturnIsAllowed(t *testing.T) {
	for _, src := range []string{
		"return {\n    name = \"goblin\",\n    hp = 10,\n}\n",
		"local defs = { hp = 10 }\nreturn defs\n",
		"local ready: boolean = false\nif not ready then\n    return\nend\nprint(\"ready\")\n",
	} {
		for _, err := range checkSource(t, src) {
			t.Errorf("%q: unexpected type error: %s", src, err.Message)
		}
	}
}

// A module's value is either its exports or what it returns; codegen would
// otherwise emit two returns.
func TestModuleCannotBothExportAndReturn(t *testing.T) {
	errs := checkSource(t, "export function double(n: number): number\n    return n * 2\nend\nreturn {}\n")
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "A module cannot both 'export' names and 'return' a value") {
		t.Fatalf("got %v", messages(errs))
	}
}

// Returns inside functions are still checked against the function.
func TestReturnInFunctionStillChecked(t *testing.T) {
	errs := checkSource(t, "function f(): number\n    return \"x\"\nend\nreturn {}\n")
	if len(errs) != 1 {
		t.Fatalf("expected one error for the function's return, got %v", messages(errs))
	}
}
