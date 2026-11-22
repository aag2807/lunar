package lsp

import (
	"strings"
	"testing"
)

// A name used inside ${...} is an occurrence in the file, at its place.
func TestRenameInsideATemplate(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	text := "local name = \"ada\"\nprint(`hi ${name}`)\n"
	c.change("main.lunar", text)
	c.diagnosticsFor("main.lunar")
	edit := decode[WorkspaceEdit](t, c.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, "name", 1, 0), "newName": "who"}))
	for uri := range edit.Changes {
		if !strings.HasPrefix(uri, "file:///") {
			t.Errorf("edit for %q", uri)
		}
	}
	edits := edit.Changes[c.uri("main.lunar")]
	if len(edits) != 2 || edits[1].Range.Start.Line != 1 || edits[1].Range.Start.Character != 12 {
		t.Errorf("edits %+v", edits)
	}
}
