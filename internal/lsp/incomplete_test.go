package lsp

import (
	"strings"
	"testing"
)

// Typing passes through text that does not parse. The server used to die
// on `export function` and freeze on an unclosed backtick; it has to keep
// publishing syntax errors and answering requests.
func TestTypingIncompleteCodeKeepsTheServerAlive(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	base := c.text("main.lunar")
	for _, typed := range []string{"export function", "function", "local s = `abc", "if e then", "class", "local t = {"} {
		c.change("main.lunar", base+typed+"\n")
		d, _ := c.diagnosticsFor("main.lunar")
		if len(d) == 0 || d[0].Code != "syntax" {
			t.Errorf("%q: diagnostics %+v", typed, d)
		}
		// The rest of the file still answers from the last good analysis.
		h := decode[Hover](t, c.request("textDocument/hover", c.pos("main.lunar", "e.hit", 1)))
		if !strings.Contains(h.Contents.Value, "Enemy") {
			t.Errorf("%q: hover %q", typed, h.Contents.Value)
		}
	}
}
