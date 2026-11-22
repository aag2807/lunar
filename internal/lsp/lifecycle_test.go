package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
)

// A client that can watch files is asked to: changes on disk (a git
// checkout, generated declarations) were otherwise only seen at the next
// edit, in Neovim and Zed.
func TestRegistersFileWatching(t *testing.T) {
	s := NewServer()
	out := &bytes.Buffer{}
	s.writer, s.logger, s.debounce = out, log.New(io.Discard, "", 0), 0
	send := func(msg map[string]any) {
		raw, _ := json.Marshal(msg)
		if err := s.handleMessage(raw); err != nil {
			t.Fatal(err)
		}
	}
	send(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{
		"rootUri": pathToURI(t.TempDir()),
		"capabilities": map[string]any{"workspace": map[string]any{
			"didChangeWatchedFiles": map[string]any{"dynamicRegistration": true}}}}})
	send(map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}})
	got := out.String()
	if !strings.Contains(got, `"method":"client/registerCapability"`) || !strings.Contains(got, `"workspace/didChangeWatchedFiles"`) {
		t.Fatalf("no registration in %s", got)
	}
	// The client's answer to that request is not a request to answer.
	out.Reset()
	send(map[string]any{"jsonrpc": "2.0", "id": "lunar-watch", "result": nil})
	if out.Len() != 0 {
		t.Errorf("answered the client's response: %s", out.String())
	}
}

func frame(msgs ...map[string]any) string {
	var sb strings.Builder
	for _, m := range msgs {
		raw, _ := json.Marshal(m)
		fmt.Fprintf(&sb, "Content-Length: %d\r\n\r\n%s", len(raw), raw)
	}
	return sb.String()
}

func runServer(input string) (string, error) {
	s := NewServer()
	out := &bytes.Buffer{}
	s.writer, s.logger, s.debounce = out, log.New(io.Discard, "", 0), 0
	s.reader = bufio.NewReader(strings.NewReader(input))
	err := s.Run()
	return out.String(), err
}

// shutdown, then exit: the server answers requests in between with an
// error and ends at exit. exit without shutdown is exit code 1.
func TestShutdownThenExit(t *testing.T) {
	init := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"rootUri": pathToURI(t.TempDir())}}
	out, err := runServer(frame(init,
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"},
		map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/hover", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
		map[string]any{"jsonrpc": "2.0", "id": 4, "method": "shutdown"}))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, `"id":3,"error"`) || strings.Contains(out, `"id":4`) {
		t.Errorf("output: %s", out)
	}
	_, err = runServer(frame(init, map[string]any{"jsonrpc": "2.0", "method": "exit"}))
	if !errors.Is(err, ErrExitWithoutShutdown) {
		t.Errorf("exit without shutdown: %v", err)
	}
}
