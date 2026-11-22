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

func renameIn(t *testing.T, text, name string, nth int, newName string) string {
	t.Helper()
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": text})
	c := newClientAt(t, root)
	c.open("a.lunar")
	c.change("a.lunar", text)
	c.diagnosticsFor("a.lunar")
	_, errMsg := c.call("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, name, nth, 0), "newName": newName})
	return errMsg
}

// A rename that would change what a name refers to is refused: a to b
// next to another b would make print(a + b) print(b + b).
func TestRenameRefusesACapture(t *testing.T) {
	cases := []string{
		"local a = 1\nlocal b = 2\nprint(a + b)\n",
		"local b = 2\nlocal a = 1\nprint(a + b)\n",
		"local a = 1\nlocal function f()\n    local b = 2\n    return a + b\nend\nprint(f())\n",
	}
	for _, text := range cases {
		if errMsg := renameIn(t, text, "a = 1", 1, "b"); !strings.Contains(errMsg, "'b'") {
			t.Errorf("%q: rename a to b gave %q", text, errMsg)
		}
	}
}

// Shadowing that changes nothing is fine: the inner a becomes an inner b.
func TestRenameAllowsHarmlessShadowing(t *testing.T) {
	text := "local b = 1\ndo\n    local a = 2\n    print(a)\nend\nprint(b)\n"
	if errMsg := renameIn(t, text, "a = 2", 1, "b"); errMsg != "" {
		t.Errorf("rename a to b: %q", errMsg)
	}
}

// Lunar's own keywords are names in Lua, and rename takes them; Lua's
// reserved words it refuses.
func TestRenameToContextualKeywords(t *testing.T) {
	text := "local a = 1\nprint(a)\n"
	if errMsg := renameIn(t, text, "a = 1", 1, "from"); errMsg != "" {
		t.Errorf("rename to from: %q", errMsg)
	}
	if errMsg := renameIn(t, text, "a = 1", 1, "end"); !strings.Contains(errMsg, "not a valid name") {
		t.Errorf("rename to end: %q", errMsg)
	}
}
