package types

import "testing"

// `x?: any` may be left out, as `x?: string` may: any? is any, which is
// no union with nil, so the ? itself has to count.
func TestOptionalAnyFieldMayBeLeftOut(t *testing.T) {
	if errs := checkSource(t, `
interface Info
	default?: any
	has_default: boolean
end
local a: Info = { has_default = false }
`); len(errs) > 0 {
		t.Errorf("got %v", messages(errs))
	}
}
