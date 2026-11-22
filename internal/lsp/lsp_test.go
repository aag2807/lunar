package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// client drives a Server in-process through its JSON-RPC handler.
type client struct {
	t     *testing.T
	s     *Server
	out   *bytes.Buffer
	root  string
	id    int
	notes []map[string]json.RawMessage // notifications received
}

func gameRoot(t *testing.T) string {
	root, err := filepath.Abs(filepath.Join("testdata", "projects", "game"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func newClient(t *testing.T) *client {
	t.Helper()
	c := &client{t: t, out: &bytes.Buffer{}, root: gameRoot(t)}
	c.s = NewServer()
	c.s.writer = c.out
	c.s.logger = log.New(io.Discard, "", 0)
	c.s.debounce = 0 // analyse on every change, synchronously
	c.request("initialize", map[string]any{"rootUri": pathToURI(c.root)})
	c.notify("initialized", map[string]any{})
	return c
}

func (c *client) uri(rel string) string { return pathToURI(filepath.Join(c.root, rel)) }

func (c *client) text(rel string) string {
	data, err := os.ReadFile(filepath.Join(c.root, rel))
	if err != nil {
		c.t.Fatal(err)
	}
	return string(data)
}

func (c *client) send(msg map[string]any) {
	c.t.Helper()
	raw, _ := json.Marshal(msg)
	if err := c.s.handleMessage(raw); err != nil {
		c.t.Fatalf("handleMessage: %v", err)
	}
}

func (c *client) notify(method string, params any) {
	c.t.Helper()
	c.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
	c.drain()
}

// request sends a request and returns its result, failing on an error
// response unless wantErr is set.
func (c *client) request(method string, params any) json.RawMessage {
	c.t.Helper()
	res, errMsg := c.call(method, params)
	if errMsg != "" {
		c.t.Fatalf("%s: error %s", method, errMsg)
	}
	return res
}

func (c *client) call(method string, params any) (json.RawMessage, string) {
	c.t.Helper()
	c.id++
	id := c.id
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	for _, msg := range c.drain() {
		if string(msg["id"]) == strconv.Itoa(id) {
			if e, ok := msg["error"]; ok {
				return nil, string(e)
			}
			return msg["result"], ""
		}
	}
	c.t.Fatalf("%s: no response", method)
	return nil, ""
}

// drain parses everything written so far; notifications are kept.
func (c *client) drain() []map[string]json.RawMessage {
	var msgs []map[string]json.RawMessage
	r := bufio.NewReader(bytes.NewReader(c.out.Bytes()))
	c.out.Reset()
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
		r.ReadString('\n')
		body := make([]byte, n)
		io.ReadFull(r, body)
		var msg map[string]json.RawMessage
		json.Unmarshal(body, &msg)
		if _, isNote := msg["method"]; isNote {
			c.notes = append(c.notes, msg)
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func (c *client) open(rel string) {
	c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
		"uri": c.uri(rel), "languageId": "lunar", "version": 1, "text": c.text(rel)}})
}

func (c *client) change(rel, text string) {
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": c.uri(rel), "version": 2},
		"contentChanges": []map[string]any{{"text": text}}})
}

// at is the LSP position of the n-th (1-based) occurrence of word in text,
// offset by delta characters into it.
func at(t *testing.T, text, word string, n, delta int) map[string]int {
	t.Helper()
	idx := -1
	for i := 0; i < n; i++ {
		next := strings.Index(text[idx+1:], word)
		if next < 0 {
			t.Fatalf("%q occurs fewer than %d times", word, n)
		}
		idx += next + 1
	}
	idx += delta
	line := strings.Count(text[:idx], "\n")
	col := idx - (strings.LastIndex(text[:idx], "\n") + 1)
	return map[string]int{"line": line, "character": col}
}

func (c *client) pos(rel, word string, n int) map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": c.uri(rel)}, "position": at(c.t, c.text(rel), word, n, 0)}
}

// diagnosticsFor returns the last diagnostics published for a file.
func (c *client) diagnosticsFor(rel string) ([]Diagnostic, bool) {
	var last []Diagnostic
	found := false
	for _, note := range c.notes {
		if string(note["method"]) != `"textDocument/publishDiagnostics"` {
			continue
		}
		var p PublishDiagnosticsParams
		json.Unmarshal(note["params"], &p)
		if p.URI == c.uri(rel) {
			last, found = p.Diagnostics, true
		}
	}
	return last, found
}

func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return v
}

func TestDiagnosticsArePublishedAndCleared(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	if d, found := c.diagnosticsFor("main.lunar"); found && len(d) > 0 {
		t.Fatalf("a clean project published %v", d)
	}

	broken := strings.Replace(c.text("main.lunar"), `spawn("imp")`, `spawn(42)`, 1)
	c.change("main.lunar", broken)
	d, _ := c.diagnosticsFor("main.lunar")
	if len(d) != 1 || !strings.Contains(d[0].Message, "Argument 1") {
		t.Fatalf("diagnostics: %+v", d)
	}
	want := at(t, broken, "spawn(42)", 1, 0)
	if d[0].Range.Start.Line != want["line"] {
		t.Errorf("range %+v, want line %d", d[0].Range, want["line"])
	}

	c.change("main.lunar", c.text("main.lunar"))
	if d, _ := c.diagnosticsFor("main.lunar"); len(d) != 0 {
		t.Errorf("fixed file still has %v", d)
	}
}

func TestDefinitionAcrossFiles(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	loc := decode[Location](t, c.request("textDocument/definition", c.pos("main.lunar", "Enemy(10)", 1)))
	if loc.URI != c.uri("actors/enemy.lunar") || loc.Range.Start.Line != 0 || loc.Range.Start.Character != 13 {
		t.Errorf("definition of Enemy: %+v", loc)
	}

	loc = decode[Location](t, c.request("textDocument/definition", c.pos("main.lunar", "spawn", 1)))
	if loc.URI != c.uri("types/api.d.lunar") || loc.Range.Start.Character != 17 {
		t.Errorf("definition of spawn: %+v", loc)
	}
}

// The standard library opens too: it is written to the cache folder.
func TestDefinitionIntoTheStdlib(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	loc := decode[Location](t, c.request("textDocument/definition", c.pos("main.lunar", "print", 1)))
	path := uriToPath(loc.URI)
	if !strings.HasSuffix(path, filepath.Join("stdlib", "lua.d.lunar")) {
		t.Fatalf("definition of print: %+v", loc)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "declare function print") {
		t.Errorf("stdlib copy at %s: %v", path, err)
	}
}

func TestReferencesAndRenameAcrossFiles(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")

	refs := decode[[]Location](t, c.request("textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, c.text("main.lunar"), "hit", 1, 0),
		"context": map[string]any{"includeDeclaration": true}}))
	if len(refs) != 2 || refs[0].URI != c.uri("actors/enemy.lunar") || refs[1].URI != c.uri("main.lunar") {
		t.Errorf("references to hit: %+v", refs)
	}

	pr := decode[PrepareRenameResult](t, c.request("textDocument/prepareRename", c.pos("main.lunar", "e.hp", 1)))
	if pr.Placeholder != "e" {
		t.Errorf("prepareRename: %+v", pr)
	}

	p := c.pos("main.lunar", "e.hp", 1)
	p["position"].(map[string]int)["character"] += 2
	p["newName"] = "health"
	edit := decode[WorkspaceEdit](t, c.request("textDocument/rename", p))
	if n := len(edit.Changes[c.uri("actors/enemy.lunar")]); n != 4 {
		t.Errorf("enemy.lunar edits: %d (%+v)", n, edit.Changes)
	}
	if n := len(edit.Changes[c.uri("main.lunar")]); n != 1 {
		t.Errorf("main.lunar edits: %d", n)
	}
	for _, edits := range edit.Changes {
		for _, e := range edits {
			if e.NewText != "health" || e.Range.End.Character-e.Range.Start.Character != 2 {
				t.Errorf("edit %+v", e)
			}
		}
	}
}

func TestRenameRefusesTheStdlib(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	if _, errMsg := c.call("textDocument/prepareRename", c.pos("main.lunar", "print", 1)); !strings.Contains(errMsg, "standard library") {
		t.Errorf("prepareRename on print: %q", errMsg)
	}
}

func TestHoverShowsTypes(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	for word, want := range map[string]string{
		"e.hit":  "e: Enemy",
		"hit(3)": "(method) Enemy.hit(n: number): void",
		"spawn":  "function spawn(kind: string, count?: number): number",
	} {
		h := decode[Hover](t, c.request("textDocument/hover", c.pos("main.lunar", word, 1)))
		if !strings.Contains(h.Contents.Value, want) {
			t.Errorf("hover on %s: %q, want %q", word, h.Contents.Value, want)
		}
	}
}

// Completion works while the line being typed does not parse yet.
func TestCompletionOfMembersAndNames(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	text := c.text("main.lunar") + "e.\n"
	c.change("main.lunar", text)
	items := decode[CompletionList](t, c.request("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, "e.\n", 1, 2)}))
	labels := map[string]bool{}
	for _, it := range items.Items {
		labels[it.Label] = true
	}
	if !labels["hp"] || !labels["hit"] || len(items.Items) != 2 {
		t.Errorf("members of e: %v", labels)
	}

	text = c.text("main.lunar") + "sp\n"
	c.change("main.lunar", text)
	items = decode[CompletionList](t, c.request("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, "sp\n", 1, 2)}))
	found := false
	for _, it := range items.Items {
		found = found || it.Label == "spawn"
	}
	if !found {
		t.Errorf("'sp' did not complete to spawn: %+v", items.Items)
	}
}

func TestSignatureHelp(t *testing.T) {
	c := newClient(t)
	c.open("main.lunar")
	text := c.text("main.lunar") + "spawn(\"orc\", \n"
	c.change("main.lunar", text)
	sh := decode[SignatureHelp](t, c.request("textDocument/signatureHelp", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("main.lunar")}, "position": at(t, text, "\"orc\", ", 1, 7)}))
	if len(sh.Signatures) != 1 || sh.Signatures[0].Label != "spawn(kind: string, count?: number): number" || sh.ActiveParameter != 1 {
		t.Errorf("signature help: %+v", sh)
	}
}

func TestDocumentAndWorkspaceSymbols(t *testing.T) {
	c := newClient(t)
	c.open("actors/enemy.lunar")
	syms := decode[[]DocumentSymbol](t, c.request("textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("actors/enemy.lunar")}}))
	if len(syms) != 1 || syms[0].Name != "Enemy" || len(syms[0].Children) != 2 {
		t.Fatalf("document symbols: %+v", syms)
	}

	ws := decode[[]SymbolInformation](t, c.request("workspace/symbol", map[string]any{"query": "ene"}))
	if len(ws) == 0 || ws[0].Name != "Enemy" {
		t.Errorf("workspace symbols: %+v", ws)
	}
}
