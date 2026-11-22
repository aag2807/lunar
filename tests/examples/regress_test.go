package examples

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestRegressions runs the small programs in tests/regress, one per bug
// found: those in run/ must compile with no diagnostics and pass their own
// asserts under every interpreter their target allows ("-- needs: lua54"
// on the first line limits a file to lua54); those in refuse/ must be
// refused, with a diagnostic containing each "-- error: <text>" line's text.
func TestRegressions(t *testing.T) {
	interps := interpreters(t)
	dir := filepath.Join(repoRoot, "tests", "regress")
	runs, _ := filepath.Glob(filepath.Join(dir, "run", "*.lunar"))
	for _, file := range runs {
		src := readFile(t, file)
		name := filepath.Base(file)
		for _, target := range []string{"lua53", "lua54"} {
			if target == "lua53" && strings.HasPrefix(src, "-- needs: lua54") {
				continue
			}
			t.Run("run/"+name+"/"+target, func(t *testing.T) {
				out := filepath.Join(t.TempDir(), "out.lua")
				msgs, code := compileJSON(t, file, target, out)
				if code != 0 || len(msgs) > 0 {
					t.Fatalf("compile: exit %d\n%s", code, strings.Join(msgs, "\n"))
				}
				for _, lua := range interps {
					if target == "lua54" && lua.version != "5.4" {
						continue
					}
					var stderr bytes.Buffer
					cmd := exec.Command(lua.path, out)
					cmd.Stderr = &stderr
					if err := cmd.Run(); err != nil || stderr.Len() > 0 {
						t.Errorf("%s: %v %s\n--- Lua\n%s", lua.path, err, stderr.String(), readFile(t, out))
					}
				}
			})
		}
	}
	refusals, _ := filepath.Glob(filepath.Join(dir, "refuse", "*.lunar"))
	for _, file := range refusals {
		src := readFile(t, file)
		t.Run("refuse/"+filepath.Base(file), func(t *testing.T) {
			var wants []string
			for _, line := range strings.Split(src, "\n") {
				if w, ok := strings.CutPrefix(strings.TrimSpace(line), "-- error: "); ok {
					wants = append(wants, w)
				}
			}
			if len(wants) == 0 {
				t.Fatal("no '-- error:' lines")
			}
			out := filepath.Join(t.TempDir(), "out.lua")
			msgs, code := compileJSON(t, file, "lua54", out)
			if code != 1 {
				t.Errorf("exit %d, want 1: %v", code, msgs)
			}
			if _, err := os.Stat(out); err == nil {
				t.Error("output written")
			}
			joined := strings.Join(msgs, "\n")
			for _, w := range wants {
				if !strings.Contains(joined, w) {
					t.Errorf("no diagnostic contains %q; got:\n%s", w, joined)
				}
			}
		})
	}
}

// compileJSON compiles one file with --json and returns its diagnostics as
// "line:col: message" and the exit code.
func compileJSON(t *testing.T, file, target, out string) ([]string, int) {
	t.Helper()
	cmd := exec.Command(compilerBin, "--json", "--target", target, filepath.Base(file), "-o", out)
	cmd.Dir = filepath.Dir(file)
	stdout, err := cmd.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	}
	var msgs []string
	for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
		var d struct {
			Kind, Message string
			Line, Column  int
		}
		json.Unmarshal([]byte(line), &d)
		if d.Kind == "diagnostic" {
			msgs = append(msgs, strconv.Itoa(d.Line)+":"+strconv.Itoa(d.Column)+": "+d.Message)
		}
	}
	return msgs, code
}
