package examples

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type luaInterpreter struct {
	path    string
	version string // "5.3", "5.4"
}

// interpreters returns the Lua binaries to run under: LUNAR_LUA_BINS
// (space-separated) when set, otherwise whichever of lua5.4 and lua5.3 are on
// PATH. With none available the test skips, unless LUNAR_REQUIRE_LUA=1, which
// CI sets so that a missing interpreter is a failure rather than a pass.
func interpreters(t *testing.T) []luaInterpreter {
	t.Helper()
	names := strings.Fields(os.Getenv("LUNAR_LUA_BINS"))
	if len(names) == 0 {
		names = []string{"lua5.4", "lua5.3"}
	}
	var found []luaInterpreter
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			if os.Getenv("LUNAR_LUA_BINS") != "" {
				t.Fatalf("LUNAR_LUA_BINS names %q, which is not on PATH", name)
			}
			continue
		}
		out, err := exec.Command(path, "-e", "io.write(_VERSION)").Output()
		if err != nil {
			t.Fatalf("asking %s for its version: %v", path, err)
		}
		found = append(found, luaInterpreter{path: path, version: strings.TrimPrefix(string(out), "Lua ")})
	}
	if len(found) == 0 {
		if os.Getenv("LUNAR_REQUIRE_LUA") == "1" {
			t.Fatal("LUNAR_REQUIRE_LUA=1 but no Lua interpreter was found")
		}
		t.Skip("no Lua interpreter on PATH (set LUNAR_REQUIRE_LUA=1 to make this a failure)")
	}
	return found
}

// runtimePrograms are the examples that are complete programs for plain Lua:
// every top-level file in examples/features, plus the todo app.
func runtimePrograms(t *testing.T) []string {
	t.Helper()
	var progs []string
	for _, rel := range exampleSources(t) {
		if (strings.HasPrefix(rel, "features/") && strings.Count(rel, "/") == 1) || rel == "todo_app.lunar" {
			progs = append(progs, rel)
		}
	}
	if len(progs) == 0 {
		t.Fatal("found no runtime programs")
	}
	return progs
}

// TestExamplesRun compiles each runtime program for every supported target
// and runs it under every interpreter that target supports. A program fails
// on a non-zero exit (a runtime error or a failed assert) or on any output to
// stderr.
func TestExamplesRun(t *testing.T) {
	luas := interpreters(t)
	targets := map[string]func(luaInterpreter) bool{
		"lua53": func(l luaInterpreter) bool { return l.version == "5.3" || l.version == "5.4" },
		"lua54": func(l luaInterpreter) bool { return l.version == "5.4" },
	}

	for target, runsOn := range targets {
		out := filepath.Join(t.TempDir(), target)
		// Compile everything first: programs may require modules from
		// sibling directories (features/mod).
		for _, rel := range exampleSources(t) {
			if strings.HasPrefix(rel, "features/") || rel == "todo_app.lunar" {
				compileExample(t, rel, filepath.Join(out, strings.TrimSuffix(filepath.FromSlash(rel), ".lunar")+".lua"), target)
			}
		}

		for _, rel := range runtimePrograms(t) {
			for _, lua := range luas {
				if !runsOn(lua) {
					continue
				}
				rel, lua := rel, lua
				t.Run(target+"/lua"+lua.version+"/"+rel, func(t *testing.T) {
					file := filepath.Join(out, strings.TrimSuffix(filepath.FromSlash(rel), ".lunar")+".lua")
					cmd := exec.Command(lua.path, filepath.Base(file))
					cmd.Dir = filepath.Dir(file)
					cmd.Env = append(os.Environ(), "LUA_PATH=./?.lua;./?/init.lua", "LUA_INIT=", "LUA_INIT_5_3=", "LUA_INIT_5_4=")
					var stdout, stderr bytes.Buffer
					cmd.Stdout, cmd.Stderr = &stdout, &stderr
					err := cmd.Run()
					if err != nil || stderr.Len() > 0 {
						t.Errorf("%s under %s: %v\nstderr:\n%s\nstdout:\n%s", rel, lua.path, err, stderr.String(), stdout.String())
					}
				})
			}
		}
	}
}
