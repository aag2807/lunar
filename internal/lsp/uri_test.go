package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

// Windows editors send drive letters in either case (VS Code writes
// file:///c%3A/...), and files under WSL as \\wsl$ shares.
func TestWindowsURIs(t *testing.T) {
	for uri, want := range map[string]string{
		"file:///c%3A/Users/red/game/main.lunar":        `C:\Users\red\game\main.lunar`,
		"file:///C:/Users/red/game/main.lunar":          `C:\Users\red\game\main.lunar`,
		"file:///d%3A/":                                 `D:\`,
		"file://wsl%24/Ubuntu/home/red/game/main.lunar": `\\wsl$\Ubuntu\home\red\game\main.lunar`,
		"file://wsl.localhost/Ubuntu/home/red/a.lunar":  `\\wsl.localhost\Ubuntu\home\red\a.lunar`,
	} {
		if got := uriToPathOn("windows", uri); got != want {
			t.Errorf("%s -> %q, want %q", uri, got, want)
		}
	}
	for _, path := range []string{`C:\Users\red\my game\main.lunar`, `\\wsl$\Ubuntu\home\red\main.lunar`} {
		uri := pathToURIOn("windows", path)
		if back := uriToPathOn("windows", uri); back != path {
			t.Errorf("%s -> %s -> %s", path, uri, back)
		}
	}
	if got := pathToURIOn("windows", `C:\Users\red\my game\main.lunar`); got != "file:///C:/Users/red/my%20game/main.lunar" {
		t.Errorf("drive path URI %s", got)
	}
	if got := uriToPathOn("linux", "file:///home/red/my%20game/a.lunar"); got != "/home/red/my game/a.lunar" {
		t.Errorf("linux path %s", got)
	}
}

// The server answers in the editor's own spelling of a file's URI, so the
// editor matches diagnostics to the buffer whatever it encoded.
func TestDiagnosticsUseTheEditorsURI(t *testing.T) {
	c := newClient(t)
	spelled := strings.Replace(c.uri("main.lunar"), "main.lunar", "main%2Elunar", 1)
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": spelled, "languageId": "lunar", "version": 1, "text": "local x: number = \"s\"\n"}})
	found := false
	for _, note := range c.notes {
		if string(note["method"]) != `"textDocument/publishDiagnostics"` {
			continue
		}
		var p PublishDiagnosticsParams
		json.Unmarshal(note["params"], &p)
		if strings.HasSuffix(p.URI, "main.lunar") && len(p.Diagnostics) > 0 {
			t.Errorf("published to %s, not the editor's %s", p.URI, spelled)
		}
		found = found || (p.URI == spelled && len(p.Diagnostics) > 0)
	}
	if !found {
		t.Errorf("no diagnostics published to %s", spelled)
	}
}
