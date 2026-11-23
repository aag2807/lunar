package lsp

import (
	"strings"
	"testing"
)

// sigAt returns the signature label and active parameter at the end of
// text (the call being typed).
func sigAt(t *testing.T, text string) (string, int) {
	t.Helper()
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "main.lunar": text})
	c := newClientAt(t, root)
	c.open("main.lunar")
	lines := strings.Split(text, "\n")
	last := len(lines) - 1
	res := c.request("textDocument/signatureHelp", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")},
		"position":     map[string]int{"line": last, "character": len(lines[last])}})
	if string(res) == "null" {
		return "", -1
	}
	h := decode[SignatureHelp](t, res)
	if len(h.Signatures) == 0 {
		return "", -1
	}
	return h.Signatures[0].Label, h.ActiveParameter
}

func TestSignatureHelpCases(t *testing.T) {
	head := "class Enemy\n    constructor(hp: number, name: string)\n    end\nend\nfunction pair(a: number, b: string): string\n    return b\nend\nfunction firstOf<T>(items: T[]): T\n    return items[1]\nend\n"
	cases := []struct {
		name, text, wantLabel string
		wantActive            int
	}{
		{"commas in a string do not count", head + `pair(1, "a, b, c`, "pair(", 1},
		{"a paren in a comment does not count", head + "pair(1, -- (x\n  ", "pair(", 1},
		{"Class.new", head + "Enemy.new(3, ", "new(", 1},
		{"a generic call", head + "firstOf<number>(", "firstOf(", 0},
		{"a string method through ':'", head + "local s = \"abc\"\nprint(s:sub(1, ", "sub(", 1},
		{"an async function says so", "declare async function wait(seconds: number): void\nasync function f(): void\n    await wait(", "async wait(seconds: number): void", 0},
	}
	for _, tc := range cases {
		label, active := sigAt(t, tc.text)
		if !strings.Contains(label, tc.wantLabel) || active != tc.wantActive {
			t.Errorf("%s: label %q active %d, want %q and %d", tc.name, label, active, tc.wantLabel, tc.wantActive)
		}
	}
}
