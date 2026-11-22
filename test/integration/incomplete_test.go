package integration

import (
	"strings"
	"testing"
)

// Half-typed code is a syntax error, never a crash: these used to exit 3
// with "internal compiler error" (and kill the language server).
func TestIncompleteCodeIsASyntaxError(t *testing.T) {
	for _, src := range []string{
		"function", "local function", "if x", "class", "interface", "enum",
		"import", "import { a } from", "export class", "export function", "declare",
		"for i = 1, do", "function g<T extends>() end", "local { a, b } = t",
		"print(1)\nfunction\nprint(2)\n", "local x: = 1", "function f()\n    return 1\n",
		"local t = { a = 1,", "x = (1 + ",
	} {
		dir := writeFiles(t, map[string]string{"half.lunar": src})
		stdout, stderr, code := runLunar(t, dir, "--json", "half.lunar")
		if code != 1 {
			t.Errorf("%q: exit %d, want 1\n%s%s", src, code, stdout, stderr)
			continue
		}
		lines := parseLines(t, stdout)
		if lines[0].Code != "syntax" || !strings.HasSuffix(lines[0].File, "half.lunar") || lines[len(lines)-1].Written {
			t.Errorf("%q: %s", src, stdout)
		}
	}
}
