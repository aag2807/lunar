package codegen

import (
	"strings"
	"testing"
)

// An exported function belongs to the module's return table, not to the
// global environment of the engine's Lua state.
func TestExportedFunctionsAreLocals(t *testing.T) {
	out := generateChecked(t, `export function a(): number
    return b()
end

export function b(): number
    return 1
end

function helper(): number
    return 2
end
`)
	// Declared up front, so a can call b before b's definition runs.
	first := strings.SplitN(strings.TrimSpace(out), "\n", 2)[0]
	if first != "local a, b" {
		t.Errorf("first line = %q, want the exported names declared local up front\n%s", first, out)
	}
	assertContains(t, out, "function a()")
	assertContains(t, out, "return {\n    a = a,\n    b = b\n}")
	// A function that is not exported keeps Lua's meaning: a global.
	assertContains(t, out, "function helper()")
	assertNotContains(t, out, "local function helper")
}
