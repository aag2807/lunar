package examples

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlainLuaCompilesUnchanged takes the plain Lua in tests/luacompat,
// compiles each file unchanged as a .lunar script, and runs both the
// original and the compiled Lua: the compile must report nothing at all,
// and the two must print the same. A file whose first line says
// "-- needs: lua54" is compiled for lua54 only and run under Lua 5.4 only.
//
// Lunar is Lua with types, so any Lua script is meant to pass through; to
// add one, drop it in tests/luacompat.
func TestPlainLuaCompilesUnchanged(t *testing.T) {
	interps := interpreters(t)
	corpus := filepath.Join(repoRoot, "tests", "luacompat")
	files, _ := filepath.Glob(filepath.Join(corpus, "*.lua"))
	if len(files) == 0 {
		t.Fatal("no Lua files in tests/luacompat")
	}
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(filepath.Base(file), ".lua")
		needs54 := strings.HasPrefix(string(src), "-- needs: lua54")
		for _, target := range []string{"lua53", "lua54"} {
			if needs54 && target == "lua53" {
				continue
			}
			t.Run(name+"/"+target, func(t *testing.T) {
				dir := t.TempDir()
				os.MkdirAll(filepath.Join(dir, "src"), 0o755)
				os.WriteFile(filepath.Join(dir, "src", name+".lunar"), src, 0o644)
				os.WriteFile(filepath.Join(dir, "lunar.config.json"),
					[]byte(`{"rootDir": "src", "outDir": "out", "target": "`+target+`"}`), 0o644)
				cmd := exec.Command(compilerBin, "build", "--json")
				cmd.Dir = dir
				out, _ := cmd.Output()
				for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
					var d struct {
						Kind, Code, Message string
						Line, Column        int
					}
					json.Unmarshal([]byte(line), &d)
					if d.Kind == "diagnostic" {
						t.Errorf("%s.lua:%d:%d: [%s] %s", name, d.Line, d.Column, d.Code, d.Message)
					}
				}
				if t.Failed() {
					return
				}
				compiled := filepath.Join(dir, "out", name+".lua")
				for _, lua := range interps {
					if (target == "lua54" || needs54) && lua.version != "5.4" {
						continue
					}
					want, wantErr := runLua(t, lua.path, corpus, file)
					got, gotErr := runLua(t, lua.path, filepath.Dir(compiled), compiled)
					if wantErr != "" {
						t.Fatalf("%s: the original fails under %s: %s", name, lua.path, wantErr)
					}
					if gotErr != "" || got != want {
						t.Errorf("%s under %s:\n--- original\n%s--- compiled\n%s%s\n--- compiled Lua\n%s",
							name, lua.path, want, got, gotErr, readFile(t, compiled))
					}
				}
			})
		}
	}
}

func runLua(t *testing.T, lua, dir, file string) (string, string) {
	t.Helper()
	cmd := exec.Command(lua, file)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil && stderr.Len() == 0 {
		stderr.WriteString(err.Error())
	}
	return stdout.String(), stderr.String()
}

func readFile(t *testing.T, path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
