package examples

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// -o naming the source itself would overwrite it with the Lua: a usage
// error (exit 2), and the source is untouched.
func TestOutputOverTheSourceIsRefused(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.lua")
	os.WriteFile(src, []byte("print(1)\n"), 0o644)
	cmd := exec.Command(compilerBin, "--json", "a.lua", "-o", "a.lua")
	cmd.Dir = dir
	err := cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 {
		t.Errorf("exit: %v", err)
	}
	if data, _ := os.ReadFile(src); string(data) != "print(1)\n" {
		t.Errorf("source became %q", data)
	}
}
