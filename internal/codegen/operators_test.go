package codegen

import "testing"

// '^' is exponentiation in Lua. Compiling it as XOR silently produced wrong
// numbers on 5.3/5.4 and bit.bxor on 5.1, so the same source meant two
// different things depending on the target.
func TestCaretIsExponentiation(t *testing.T) {
	output := generate(t, `local x: number = 2 ^ 8`)

	assertContains(t, output, "2 ^ 8")
	assertNotContains(t, output, "bxor")
}

// Binary '~' is XOR.
func TestTildeIsBitwiseXor(t *testing.T) {
	output := generate(t, `local x: number = 12 ~ 10`)

	assertContains(t, output, "12 ~ 10")
}

// A literal receiver has to be parenthesised or the Lua is a syntax error.
func TestLiteralReceiverIsParenthesised(t *testing.T) {
	output := generate(t, `local words = ("a b"):gmatch("%a+")`)

	assertContains(t, output, `("a b"):gmatch`)
}

// Namespace members are attached to the namespace table, including exported ones.
func TestNamespaceExportsAttachToTable(t *testing.T) {
	output := generate(t, "namespace Geometry\n\texport function area(w: number, h: number): number\n\t\treturn w * h\n\tend\nend")

	assertContains(t, output, "function Geometry.area(w, h)")
}

// Import paths become dotted module names require can resolve.
func TestImportPathBecomesLuaModuleName(t *testing.T) {
	output := generate(t, `import { double } from "./mod/math_utils"`)

	assertContains(t, output, `require("mod.math_utils")`)
}
