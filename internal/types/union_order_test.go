package types

import (
	"strings"
	"testing"
)

// Types built from an interface's fields list them in a fixed order, so a
// message reads the same on every run.
func TestKeyofAndIndexedAccessPrintInAFixedOrder(t *testing.T) {
	src := `
interface Stats
    zeta: number
    alpha: string
    mid: boolean
end
type K = keyof Stats
type V = Stats[string]
local k: K = "nope"
local v: V = {}
`
	for i := 0; i < 30; i++ {
		errs := checkSource(t, src)
		got := strings.Join(messages(errs), "\n")
		if !strings.Contains(got, `'"alpha" | "mid" | "zeta"'`) || !strings.Contains(got, `'string | boolean | number'`) {
			t.Fatalf("run %d:\n%s", i, got)
		}
	}
}

// An array of mixed values lists its element types in source order.
func TestArrayElementUnionInSourceOrder(t *testing.T) {
	for i := 0; i < 30; i++ {
		errs := checkSource(t, "local xs = {1, \"a\", true}\nlocal n: number = xs\n")
		got := strings.Join(messages(errs), "\n")
		if !strings.Contains(got, "'(number | string | boolean)[]'") {
			t.Fatalf("run %d: %s", i, got)
		}
	}
}
