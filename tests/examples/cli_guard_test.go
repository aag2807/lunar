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

func exitCode(err error) int {
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	if err != nil {
		return -1
	}
	return 0
}

// Usage mistakes are exit 2: lunar build with no config to find, and a
// flag of the other mode (which used to be ignored).
func TestUsageMistakes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.lunar"), []byte("print(1)\n"), 0o644)
	for _, args := range [][]string{
		{"build", "--json"},
		{"build", "--json", "-o", "x.lua"},
		{"--json", "a.lunar", "--out", "build"},
		{"--json", "a.lunar", "--project", "lunar.config.json"},
	} {
		cmd := exec.Command(compilerBin, args...)
		cmd.Dir = dir
		if code := exitCode(cmd.Run()); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

// -o /dev/stdout with stdout redirected to a file writes the Lua there.
func TestOutputToStdoutRedirectedToAFile(t *testing.T) {
	if _, err := os.Stat("/dev/stdout"); err != nil {
		t.Skip("no /dev/stdout")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.lunar"), []byte("print(1)\n"), 0o644)
	out, _ := os.Create(filepath.Join(dir, "out.lua"))
	cmd := exec.Command(compilerBin, "a.lunar", "-o", "/dev/stdout")
	cmd.Dir, cmd.Stdout = dir, out
	err := cmd.Run()
	out.Close()
	if code := exitCode(err); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "out.lua")); string(data) != "print(1)\n" {
		t.Errorf("out.lua = %q", data)
	}
}
