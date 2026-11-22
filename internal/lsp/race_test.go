package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Edits arrive while the debounce timer is analysing on its own goroutine;
// run with -race, this checks the two never touch state unguarded, and
// that every request is still answered.
func TestEditsAndRequestsWhileAnalysing(t *testing.T) {
	root := gameRoot(t)
	inR, inW := io.Pipe()
	out := &lockedBuffer{}
	s := NewServer()
	s.reader = bufio.NewReader(inR)
	s.writer = out
	s.logger = log.New(io.Discard, "", 0)
	s.debounce = 2 * time.Millisecond

	done := make(chan error, 1)
	go func() { done <- s.Run() }()

	send := func(msg map[string]any) {
		raw, _ := json.Marshal(msg)
		fmt.Fprintf(inW, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	}
	uri := pathToURI(root + "/main.lunar")
	base := "import { Enemy } from \"./actors/enemy\"\nlocal e: Enemy = Enemy(10)\n"
	send(map[string]any{"jsonrpc": "2.0", "id": 0, "method": "initialize", "params": map[string]any{"rootUri": pathToURI(root)}})
	send(map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}})
	send(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "lunar", "version": 1, "text": base}}})
	const rounds = 40
	for i := 1; i <= rounds; i++ {
		text := base
		if i%2 == 0 {
			text += "local s: string = 1\n"
		}
		send(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": i + 1}, "contentChanges": []map[string]any{{"text": text}}}})
		send(map[string]any{"jsonrpc": "2.0", "id": i, "method": "textDocument/hover", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}, "position": map[string]int{"line": 1, "character": 6}}})
		time.Sleep(time.Millisecond)
	}
	send(map[string]any{"jsonrpc": "2.0", "id": rounds + 1, "method": "shutdown"})
	inW.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for i := 0; i <= rounds+1; i++ {
		if !strings.Contains(got, fmt.Sprintf(`"id":%d,`, i)) {
			t.Errorf("no response to request %d", i)
		}
	}
}
