package lsp

import (
	"strings"
	"testing"
)

// Completion asked for with the cursor inside a word (pr|int) panicked,
// and the panic took the whole server down.
func TestCompletionInsideAWord(t *testing.T) {
	text := "local value = 1\nprint(value)\n"
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": text})
	c := newClientAt(t, root)
	c.open("a.lunar")
	for _, delta := range []int{1, 2, 3} {
		c.request("textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, "print", 1, delta)})
	}
	if got := hoverText(t, c, text, "value)", 0); !strings.Contains(got, "value") {
		t.Errorf("server not answering after completion: %q", got)
	}
}

// A request that panics is answered with an error; the server lives on.
func TestPanickingRequestIsAnError(t *testing.T) {
	text := "local value = 1\n"
	root := writeTree(t, map[string]string{"lunar.config.json": `{"outDir": "../out"}`, "a.lunar": text})
	c := newClientAt(t, root)
	c.open("a.lunar")
	c.s.testPanic = "textDocument/hover"
	if _, errMsg := c.call("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": map[string]int{"line": 0, "character": 7}}); !strings.Contains(errMsg, "internal error") {
		t.Errorf("want an internal error response, got %q", errMsg)
	}
	c.s.testPanic = ""
	if got := hoverText(t, c, text, "value", 0); !strings.Contains(got, "value") {
		t.Errorf("server not answering after a panic: %q", got)
	}
}

func hoverText(t *testing.T, c *client, text, word string, delta int) string {
	t.Helper()
	h := decode[Hover](t, c.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": c.uri("a.lunar")}, "position": at(t, text, word, 1, delta)}))
	return h.Contents.Value
}
