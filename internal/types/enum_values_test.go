package types

import (
	"strings"
	"testing"
)

func TestEnumMemberAfterStringValueNeedsExplicitValue(t *testing.T) {
	input := `
enum Tag
	Red = "red"
	Blue
end
`
	errs := checkSource(t, input)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'Blue' needs an explicit value") || !strings.Contains(errs[0].Message, "'Red'") {
		t.Fatalf("expected one error naming Blue and Red, got %v", messages(errs))
	}
}

func TestEnumAutoValuesCheckCleanly(t *testing.T) {
	input := `
enum Rank
	Grunt
	Elite = 10
	Boss
end
local r: Rank = Rank.Boss
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
