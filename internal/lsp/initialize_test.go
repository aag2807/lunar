package lsp

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"
	"testing"

	"lunar/internal/version"
)

// The server reports the same version as `lunar --version`, so an editor can
// tell when the LSP and the compiler on PATH have drifted apart.
func TestInitializeReportsServerInfoVersion(t *testing.T) {
	var out bytes.Buffer
	s := NewServer()
	s.writer = &out
	s.logger = log.New(io.Discard, "", 0)

	if err := s.handleInitialize(json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)); err != nil {
		t.Fatalf("handleInitialize: %v", err)
	}

	body := out.String()
	if i := strings.Index(body, "\r\n\r\n"); i >= 0 {
		body = body[i+4:]
	}
	var resp struct {
		Result struct {
			ServerInfo *struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decoding response %q: %v", body, err)
	}
	if resp.Result.ServerInfo == nil {
		t.Fatalf("initialize result has no serverInfo: %s", body)
	}
	if resp.Result.ServerInfo.Name != "lunar" || resp.Result.ServerInfo.Version != version.Version {
		t.Fatalf("serverInfo = %+v, want name %q version %q", *resp.Result.ServerInfo, "lunar", version.Version)
	}
}
