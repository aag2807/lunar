package lsp

import (
	"strings"
	"testing"
)

// An unsaved new buffer (untitled:) is checked like a lone file.
func TestUntitledBufferGetsDiagnostics(t *testing.T) {
	c := newClientAt(t, t.TempDir())
	uri := "untitled:Untitled-1"
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": uri, "languageId": "lunar", "version": 1, "text": "local n: number = \"x\"\n"}})
	c.flushNow()
	found := false
	for _, note := range c.notes {
		if strings.Contains(string(note["params"]), uri) && strings.Contains(string(note["params"]), "number") {
			found = true
		}
	}
	if !found {
		t.Errorf("no diagnostics for %s", uri)
	}
}
