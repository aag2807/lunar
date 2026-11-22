package lsp

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

// A workspace folder with no lunar.config.json (a repository's root) is
// not one program: a stray declaration file elsewhere in it does not
// change how a script is checked, and a project inside it is checked as
// that project, open file or not.
func TestWorkspaceWithoutAConfig(t *testing.T) {
	root := writeTree(t, map[string]string{
		"tools/testdata/lua.d.lunar": "declare function assert(v: number, n: number): void\n",
		"game/main.lunar":            "assert(true)\nprint(\"ok\")\n",
		"proj/lunar.config.json":     `{"outDir": "../out"}`,
		"proj/bad.lunar":             "local n: number = \"x\"\n",
	})
	c := newClientAt(t, root)
	c.open("game/main.lunar")
	if diags, _ := c.diagnosticsFor("game/main.lunar"); len(diags) != 0 {
		t.Errorf("game/main.lunar: %+v", diags)
	}
	diags, found := c.diagnosticsFor("proj/bad.lunar")
	if !found || len(diags) != 1 || !strings.Contains(diags[0].Message, "number") {
		t.Errorf("proj/bad.lunar (not open): %+v", diags)
	}
}

// Every workspace folder is looked at, not only the first.
func TestEveryWorkspaceFolder(t *testing.T) {
	one := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": "print(1)\n"})
	two := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "b.lunar": "local n: number = \"x\"\n"})
	c := newClientWith(t, one, map[string]any{"workspaceFolders": []map[string]any{
		{"uri": pathToURI(one), "name": "one"}, {"uri": pathToURI(two), "name": "two"}}})
	c.flushNow()
	found := false
	for _, note := range c.notes {
		if strings.Contains(string(note["params"]), pathToURI(two)+"/b.lunar") && strings.Contains(string(note["params"]), "number") {
			found = true
		}
	}
	if !found {
		t.Errorf("no diagnostics for the second folder's b.lunar")
	}
}

// newClientWith starts a server with these initialize params.
func newClientWith(t *testing.T, root string, params map[string]any) *client {
	t.Helper()
	c := &client{t: t, out: &bytes.Buffer{}, root: root}
	c.s = NewServer()
	c.s.writer = c.out
	c.s.logger = log.New(io.Discard, "", 0)
	c.s.debounce = 0
	c.request("initialize", params)
	c.notify("initialized", map[string]any{})
	return c
}

// flushNow has the server analyse and collects what it published.
func (c *client) flushNow() {
	c.call("workspace/symbol", map[string]any{"query": ""})
}
