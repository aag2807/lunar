package codegen

import "testing"

// A member without a value follows the previous one, as in TypeScript and C:
// after `Elite = 10` comes 11, not the member's position.
func TestEnumAutoValueFollowsPreviousValue(t *testing.T) {
	out := generate(t, "enum Rank\n    Grunt\n    Elite = 10\n    Boss\n    Final\nend\n")
	assertContains(t, out, "Grunt = 0,")
	assertContains(t, out, "Elite = 10,")
	assertContains(t, out, "Boss = 11,")
	assertContains(t, out, "Final = 12,")
}

func TestEnumAutoValueAfterNegativeValue(t *testing.T) {
	out := generate(t, "enum Offset\n    Left = -1\n    Center\n    Right\nend\n")
	assertContains(t, out, "Center = 0,")
	assertContains(t, out, "Right = 1,")
}
