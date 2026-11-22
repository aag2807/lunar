package lsp

import (
	"strings"
	"testing"
)

// While a file does not parse, its positions come from an older text, so a
// rename would edit the wrong characters. It is refused until it parses.
func TestRenameIsRefusedWhileAFileDoesNotParse(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	c.change("main.lunar", "local s = 1\n"+c.text("main.lunar")+"function\n")
	c.diagnosticsFor("main.lunar")

	p := c.pos("main.lunar", "e.hp", 1)
	p["position"].(map[string]int)["character"] += 2
	p["newName"] = "health"
	if _, errMsg := c.call("textDocument/rename", p); !strings.Contains(errMsg, "main.lunar does not parse") {
		t.Errorf("rename with a broken file: %q", errMsg)
	}
	if _, errMsg := c.call("textDocument/prepareRename", p); !strings.Contains(errMsg, "main.lunar does not parse") {
		t.Errorf("prepareRename with a broken file: %q", errMsg)
	}
}

// A field written as a key in a table literal is an occurrence of it.
func TestRenameIncludesTableLiteralKeys(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	text := "interface Stats\n    atk: number\nend\nlocal s: Stats = { atk = 1 }\nprint(s.atk)\n"
	c.change("main.lunar", text)
	c.diagnosticsFor("main.lunar")

	edit := decode[WorkspaceEdit](t, c.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, "atk", 1, 0), "newName": "attack"}))
	if n := len(edit.Changes[c.uri("main.lunar")]); n != 3 {
		t.Errorf("main.lunar edits: %d, want 3 (declaration, literal key, s.atk): %+v", n, edit.Changes)
	}
}
