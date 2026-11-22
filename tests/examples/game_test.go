package examples

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGameScriptsBehaveTheSame builds the engine's game scripts as a Lunar
// project and runs each original and compiled script under the engine
// side's recording harness, which must print the same for both. See
// tests/luacompat/game/README.md.
func TestGameScriptsBehaveTheSame(t *testing.T) {
	var lua string
	for _, interp := range interpreters(t) {
		if interp.version == "5.4" {
			lua = interp.path
		}
	}
	if lua == "" {
		if os.Getenv("LUNAR_REQUIRE_LUA") == "1" {
			t.Fatal("the game scripts need Lua 5.4")
		}
		t.Skip("no Lua 5.4")
	}

	game := filepath.Join(repoRoot, "tests", "luacompat", "game")
	originals, _ := filepath.Glob(filepath.Join(game, "scripts", "*.lua"))
	if len(originals) != 14 {
		t.Fatalf("expected the 14 game scripts, found %d", len(originals))
	}
	dir := t.TempDir()
	for _, f := range originals {
		copyTo(t, f, filepath.Join(dir, "scripts", strings.TrimSuffix(filepath.Base(f), ".lua")+".lunar"))
	}
	decls, _ := filepath.Glob(filepath.Join(game, "api", "*.d.lunar"))
	for _, f := range decls {
		copyTo(t, f, filepath.Join(dir, "api", filepath.Base(f)))
	}
	os.WriteFile(filepath.Join(dir, "lunar.config.json"),
		[]byte(`{ "rootDir": "scripts", "outDir": "out", "target": "lua54", "declarations": ["api"] }`), 0o644)

	cmd := exec.Command(compilerBin, "build", "--json")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var summary struct {
		Kind                    string
		Errors, Warnings, Files int
		Written                 bool
	}
	json.Unmarshal([]byte(lines[len(lines)-1]), &summary)
	if err != nil || len(lines) != 1 || summary.Kind != "summary" || summary.Errors != 0 || summary.Warnings != 0 ||
		summary.Files != 14 || !summary.Written || stderr.Len() != 0 {
		t.Fatalf("lunar build: %v\n%s%s", err, out, stderr.String())
	}

	harness := filepath.Join(game, "harness.lua")
	for _, original := range originals {
		name := filepath.Base(original)
		want, wantDone := runHarness(t, lua, harness, original)
		got, gotDone := runHarness(t, lua, harness, filepath.Join(dir, "out", name))
		switch {
		case wantDone != gotDone:
			t.Errorf("%s: the original finished: %v, the compiled script finished: %v", name, wantDone, gotDone)
		case got != want:
			t.Errorf("%s behaves differently\n--- original\n%s--- compiled\n%s", name, want, got)
		}
	}
}

// runHarness runs one script under the harness, and says whether it
// finished within the time limit.
func runHarness(t *testing.T, lua, harness, script string) (string, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, lua, harness)
	cmd.Env = append(os.Environ(), "SCRIPT="+script)
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", false
	}
	if err != nil {
		t.Fatalf("running %s under the harness: %v\n%s", script, err, out)
	}
	return string(out), true
}

func copyTo(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(to), 0o755)
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
