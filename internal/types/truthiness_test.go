package types

import (
	"strings"
	"testing"
)

// Lua tests truthiness, not booleans: every value except nil and false is
// true. Game loops are `while true do`, and optionals are tested as `if x`.
func TestAnyValueIsACondition(t *testing.T) {
	input := `
local i: number = 0
while true do
	i = i + 1
	if i > 3 then
		break
	end
end
repeat
	i = i - 1
until true
local name: string = "x"
if name then
	i = 0
end
local count: number? = 2
while count do
	count = nil
end
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestTruthinessNarrowsOptionalInThenBranch(t *testing.T) {
	input := `
function use(b: number?): number
	if b then
		return b + 1
	end
	return 0
end
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestNotNarrowsOptionalAfterEarlyReturn(t *testing.T) {
	input := `
function use(b: number?): number
	if not b then
		return 0
	end
	return b * 2
end
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A call that returns nothing is the one condition with no value to test.
func TestVoidConditionIsAnError(t *testing.T) {
	input := `
function nothing(): void
end
if nothing() then
	print("never")
end
`
	errs := checkSource(t, input)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "no value") {
		t.Fatalf("expected one 'no value' error, got %v", messages(errs))
	}
}

func messages(errs []*TypeError) []string {
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Message
	}
	return out
}
