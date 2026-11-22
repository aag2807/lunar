package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Messages keep '<' and '>' as written, so a regex over the raw line (a VS
// Code problem matcher, a console) shows '<table literal>'.
func TestJSONKeepsAngleBrackets(t *testing.T) {
	dir := writeFiles(t, map[string]string{"a.lunar": "local t: number = { a = 1 }\n"})
	stdout, _, _ := runLunar(t, dir, "--json", "a.lunar")
	if !strings.Contains(stdout, "'<table literal>'") || strings.Contains(stdout, "\\u00") {
		t.Errorf("stdout: %s", stdout)
	}
}

// --json=true is JSON mode from the first line, early failures included.
func TestJSONEqualsTrueIsJSONMode(t *testing.T) {
	dir := writeFiles(t, map[string]string{})
	stdout, stderr, code := runLunar(t, dir, "--json=true", "missing.lunar")
	if code != 2 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if lines := parseLines(t, stdout); lines[0].Code != "usage" {
		t.Errorf("stdout: %s", stdout)
	}
}

// Flags may come after the file name, as with most compilers.
func TestFlagsAfterTheFileName(t *testing.T) {
	dir := writeFiles(t, map[string]string{"game.lunar": "print(1)\n"})
	_, stderr, code := runLunar(t, dir, "game.lunar", "-o", "out.lua")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "out.lua")); err != nil {
		t.Errorf("out.lua not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "game.lua")); err == nil {
		t.Error("game.lua written; -o was ignored")
	}
}

// -o may name a device or a pipe: the Lua is written to it directly.
func TestOutputToADevice(t *testing.T) {
	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skip("no /dev/null here")
	}
	dir := writeFiles(t, map[string]string{"game.lunar": "print(\"from lunar\")\n"})
	if _, stderr, code := runLunar(t, dir, "game.lunar", "-o", "/dev/null"); code != 0 {
		t.Errorf("-o /dev/null: exit %d: %s", code, stderr)
	}
	stdout, stderr, code := runLunar(t, dir, "game.lunar", "-o", "/dev/stdout")
	if code != 0 || strings.TrimSpace(stdout) != `print("from lunar")` {
		t.Errorf("-o /dev/stdout: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}
