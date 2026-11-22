package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gameProject(t *testing.T) string {
	t.Helper()
	return writeFiles(t, map[string]string{
		"engine/api.d.lunar":         "declare function engine_log(msg: string): void\n",
		"scripts/lunar.config.json":  `{"outDir": "../build/scripts", "target": "lua54", "declarations": ["../engine"]}`,
		"scripts/actors/enemy.lunar": "export class Enemy\n    hp: number\n    constructor(hp: number)\n        self.hp = hp\n    end\n    hit(n: number): void\n        self.hp = self.hp - n\n    end\nend\n",
		"scripts/main.lunar":         "import { Enemy } from \"./actors/enemy\"\nimport { greet } from \"lib/greet\"\nlocal e: Enemy = Enemy(10)\ne.hit(3)\nassert(e.hp == 7)\nassert(greet(\"imp\") == \"hi imp\")\nprint(\"ok\")\n",
		"scripts/lib/greet.lunar":    "export function greet(name: string): string\n    return \"hi \" .. name\nend\n",
		"scripts/data/goblin.lunar":  "return { name = \"goblin\", hp = 10 }\n",
	})
}

// One run compiles the whole scripts folder, from anywhere in the project,
// and the result runs: imported classes construct, require names resolve
// against the output folder.
func TestBuildCompilesAndRunsAProject(t *testing.T) {
	root := gameProject(t)
	stdout, stderr, code := runLunar(t, filepath.Join(root, "scripts", "actors"), "build", "--json")
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	lines := parseLines(t, stdout)
	if len(lines) != 1 || lines[0].Kind != "summary" || lines[0].Files != 4 || !lines[0].Written {
		t.Fatalf("stdout:\n%s", stdout)
	}

	out := filepath.Join(root, "build", "scripts")
	for _, lua := range []string{"lua5.4"} {
		bin, err := exec.LookPath(lua)
		if err != nil {
			if os.Getenv("LUNAR_REQUIRE_LUA") == "1" {
				t.Fatalf("%s not found", lua)
			}
			t.Skipf("%s not found", lua)
		}
		cmd := exec.Command(bin, "main.lua")
		cmd.Dir = out
		cmd.Env = append(os.Environ(), "LUA_PATH=./?.lua;./?/init.lua")
		got, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(got)) != "ok" {
			t.Errorf("%s main.lua: %v\n%s", lua, err, got)
		}
	}
}

// A cross-file error fails the whole build and leaves the previous output.
func TestBuildCatchesCrossFileErrors(t *testing.T) {
	root := gameProject(t)
	if _, _, code := runLunar(t, filepath.Join(root, "scripts"), "build"); code != 0 {
		t.Fatal("first build failed")
	}
	main := filepath.Join(root, "build", "scripts", "main.lua")
	before, _ := os.ReadFile(main)

	os.WriteFile(filepath.Join(root, "scripts", "main.lunar"), []byte("import { greet } from \"lib/greet\"\nlocal n: number = greet(\"imp\")\n"), 0o644)
	stdout, _, code := runLunar(t, filepath.Join(root, "scripts"), "build", "--json")
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	lines := parseLines(t, stdout)
	if len(lines) != 2 || !strings.HasSuffix(lines[0].File, filepath.Join("scripts", "main.lunar")) || lines[0].Line != 2 || lines[1].Written {
		t.Errorf("stdout:\n%s", stdout)
	}
	after, _ := os.ReadFile(main)
	if string(after) != string(before) {
		t.Error("main.lua changed after a failed build")
	}
}

func TestBuildWithoutOutDirIsAUsageError(t *testing.T) {
	dir := writeFiles(t, map[string]string{"lunar.config.json": `{}`, "a.lunar": "print(1)\n"})
	stdout, _, code := runLunar(t, dir, "build", "--json")
	if code != 2 {
		t.Errorf("exit %d, want 2\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "no output folder") {
		t.Errorf("stdout:\n%s", stdout)
	}
}

// --decl points at a shared set of declarations from the command line.
func TestDeclFlag(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"api/engine.d.lunar": "declare function spawn(kind: string): number\n",
		"game/main.lunar":    "local id: number = spawn(\"imp\")\n",
	})
	if _, stderr, code := runLunar(t, filepath.Join(dir, "game"), "main.lunar"); code != 1 {
		t.Fatalf("without --decl spawn should be undefined; exit %d %s", code, stderr)
	}
	if out, stderr, code := runLunar(t, filepath.Join(dir, "game"), "--decl", "../api", "main.lunar"); code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, stderr)
	}
}
