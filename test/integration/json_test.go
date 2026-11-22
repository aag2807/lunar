package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type jsonLine struct {
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Errors    int    `json:"errors"`
	Files     int    `json:"files"`
	Written   bool   `json:"written"`
}

// runLunar runs the compiler in dir and returns stdout, stderr and the exit code.
func runLunar(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatalf("running lunar: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

// parseLines decodes NDJSON output, failing on any line that is not JSON.
func parseLines(t *testing.T, out string) []jsonLine {
	t.Helper()
	var lines []jsonLine
	for _, raw := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		var l jsonLine
		if err := json.Unmarshal([]byte(raw), &l); err != nil {
			t.Fatalf("stdout line is not JSON: %q (%v)\nall stdout:\n%s", raw, err, out)
		}
		lines = append(lines, l)
	}
	return lines
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// The editor reads errors as JSON lines from stdout; nothing else may be
// mixed in, and the exit code says whether the build failed.
func TestJSONReportsDiagnosticsOnStdoutOnly(t *testing.T) {
	dir := writeFiles(t, map[string]string{"enemy.lunar": "interface Enemy\n    hp: number\nend\nlocal e: Enemy = { hp = \"ten\" }\n"})
	stdout, stderr, code := runLunar(t, dir, "--json", "enemy.lunar")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	if stderr != "" {
		t.Errorf("stderr should be empty with --json, got %q", stderr)
	}
	lines := parseLines(t, stdout)
	if len(lines) != 2 {
		t.Fatalf("want one diagnostic and a summary, got %d lines:\n%s", len(lines), stdout)
	}
	d := lines[0]
	want := filepath.Join(dir, "enemy.lunar")
	if d.Kind != "diagnostic" || d.File != want || d.Line != 4 || d.Column != 25 || d.EndLine != 4 || d.EndColumn != 30 ||
		d.Severity != "error" || d.Code != "field-type" || d.Message != "Field 'hp' of 'Enemy' must be 'number', got 'string'" {
		t.Errorf("diagnostic = %+v", d)
	}
	s := lines[1]
	if s.Kind != "summary" || s.Errors != 1 || s.Files != 1 || s.Written {
		t.Errorf("summary = %+v", s)
	}
	if _, err := os.Stat(filepath.Join(dir, "enemy.lua")); err == nil {
		t.Error("a failed compile wrote enemy.lua")
	}
}

func TestJSONSuccessPrintsOnlyTheSummary(t *testing.T) {
	dir := writeFiles(t, map[string]string{"ok.lunar": "local x: number = 1\n"})
	stdout, stderr, code := runLunar(t, dir, "--json", "ok.lunar")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	lines := parseLines(t, stdout)
	if len(lines) != 1 || lines[0].Kind != "summary" || lines[0].Errors != 0 || !lines[0].Written || lines[0].Files != 1 {
		t.Fatalf("stdout:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "ok.lua")); err != nil {
		t.Errorf("ok.lua not written: %v", err)
	}
}

// An error in a declaration file is reported against that file.
func TestJSONDeclarationErrorNamesTheDeclarationFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"api.d.lunar": "\n\ndeclare function spawn(kind: Monster): number\n",
		"main.lunar":  "local id: number = spawn(\"imp\")\n",
	})
	stdout, _, code := runLunar(t, dir, "--json", "main.lunar")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	lines := parseLines(t, stdout)
	if lines[0].File != filepath.Join(dir, "api.d.lunar") || lines[0].Line != 3 {
		t.Errorf("diagnostic = %+v", lines[0])
	}
}

// When a compile fails, the last good Lua stays where it is, so the game
// keeps running on it.
func TestFailedCompileLeavesOutputUntouched(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"game.lunar": "local x: number = \"no\"\n",
		"game.lua":   "-- last good build\n",
	})
	if _, _, code := runLunar(t, dir, "game.lunar"); code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	data, err := os.ReadFile(filepath.Join(dir, "game.lua"))
	if err != nil || string(data) != "-- last good build\n" {
		t.Errorf("game.lua changed: %q %v", data, err)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	dir := writeFiles(t, map[string]string{"ok.lunar": "local x = 1\n"})
	cases := [][]string{
		{"--json"},                  // no input file
		{"--json", "missing.lunar"}, // input does not exist
		{"--json", "--target", "lua51", "ok.lunar"},
		{"--json", "--no-such-flag", "ok.lunar"},
	}
	for _, args := range cases {
		stdout, stderr, code := runLunar(t, dir, args...)
		if code != 2 {
			t.Errorf("%v: exit code %d, want 2", args, code)
			continue
		}
		if stderr != "" {
			t.Errorf("%v: stderr should be empty with --json, got %q", args, stderr)
		}
		lines := parseLines(t, stdout)
		if len(lines) != 2 || lines[0].Code != "usage" || lines[0].Message == "" || lines[1].Kind != "summary" {
			t.Errorf("%v: stdout:\n%s", args, stdout)
		}
	}
}

func TestOutputWriteFailureExitsThree(t *testing.T) {
	dir := writeFiles(t, map[string]string{"ok.lunar": "local x = 1\n", "blocker": "a file, not a directory\n"})
	stdout, _, code := runLunar(t, dir, "--json", "-o", filepath.Join("blocker", "ok.lua"), "ok.lunar")
	if code != 3 {
		t.Errorf("exit code %d, want 3\n%s", code, stdout)
	}
	lines := parseLines(t, stdout)
	if lines[0].Code != "io" {
		t.Errorf("diagnostic = %+v", lines[0])
	}
}

// Without --json, errors are for people: on stderr, with source context.
func TestHumanErrorsGoToStderr(t *testing.T) {
	dir := writeFiles(t, map[string]string{"bad.lunar": "local x: number = \"no\"\n"})
	stdout, stderr, code := runLunar(t, dir, "bad.lunar")
	if code != 1 || stdout != "" {
		t.Errorf("exit %d, stdout %q", code, stdout)
	}
	if !strings.Contains(stderr, "bad.lunar:1:") || !strings.Contains(stderr, "local x: number") {
		t.Errorf("stderr:\n%s", stderr)
	}
}
