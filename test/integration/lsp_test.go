package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// `lunar lsp` is the language server, so an editor needs one binary.
func TestLunarLspAnswersInitialize(t *testing.T) {
	cmd := exec.Command(compilerBin, "lsp")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":null}}`
	fmt.Fprintf(stdin, "Content-Length: %d\r\n\r\n%s", len(body), body)

	result := make(chan string, 1)
	go func() {
		r := bufio.NewReader(stdout)
		line, _ := r.ReadString('\n')
		n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
		r.ReadString('\n')
		buf := make([]byte, n)
		io.ReadFull(r, buf)
		result <- string(buf)
	}()
	select {
	case got := <-result:
		var resp struct {
			Result struct {
				ServerInfo struct{ Name string } `json:"serverInfo"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(got), &resp); err != nil || resp.Result.ServerInfo.Name != "lunar" {
			t.Errorf("initialize response: %s (%v)", got, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no response from lunar lsp")
	}
}
