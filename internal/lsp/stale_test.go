package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// While a file does not parse, an answer from its last good analysis is
// given only where the token is still the same.
func TestNoAnswerFromMovedText(t *testing.T) {
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": "local alpha = 1\nlocal beta = 2\nprint(alpha, beta)\n"})
	c := newClientAt(t, root)
	c.open("a.lunar")
	moved := "-- note\nlocal alpha = 1\nlocal beta = 2\nprint(alpha, beta)\nlocal x =\n"
	c.change("a.lunar", moved)
	h := decode[Hover](t, c.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, moved, "alpha", 1, 0)}))
	if strings.Contains(h.Contents.Value, "beta") {
		t.Errorf("hover on alpha after the text moved: %q", h.Contents.Value)
	}
}

// A file deleted from the project leaves the index: rename does not edit it.
func TestDeletedFileLeavesTheIndex(t *testing.T) {
	root := writeTree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../out"}`,
		"math.lunar":        "export function clamp(n: number): number\n    return n\nend\n",
		"a.lunar":           "import { clamp } from \"./math\"\nprint(clamp(1))\n",
	})
	c := newClientAt(t, root)
	c.open("math.lunar")
	os.Remove(filepath.Join(root, "a.lunar"))
	c.notify("workspace/didChangeWatchedFiles", map[string]any{"changes": []map[string]any{{"uri": c.uri("a.lunar"), "type": 3}}})
	text := c.text("math.lunar")
	edit := decode[WorkspaceEdit](t, c.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("math.lunar")}, "position": at(t, text, "clamp", 1, 0), "newName": "limit"}))
	if _, edited := edit.Changes[c.uri("a.lunar")]; edited {
		t.Errorf("rename edits the deleted a.lunar: %+v", edit.Changes)
	}
}
