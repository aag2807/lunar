package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func build(t *testing.T, dir string) Result {
	t.Helper()
	p, diags := Open(Options{Dir: dir})
	if diags.HasErrors() {
		t.Fatalf("Open: %v", messagesOf(diags))
	}
	res, diags := Build(p)
	if diags.HasErrors() {
		t.Fatalf("Build: %v", messagesOf(diags))
	}
	return res
}

// One build compiles the whole scripts folder into the output folder,
// mirroring its layout, and copies plain Lua scripts through.
func TestBuildWritesTheWholeFolder(t *testing.T) {
	root := tree(t, map[string]string{
		"scripts/lunar.config.json": `{"outDir": "../build/scripts"}`,
		"scripts/main.lunar":        "import { think } from \"./ai/brain\"\nthink()\n",
		"scripts/ai/brain.lunar":    "export function think(): void\n    print(\"hmm\")\nend\n",
		"scripts/legacy/util.lua":   "return { old = true }\n",
		"scripts/types/api.d.lunar": "declare function spawn(kind: string): number\n",
	})
	res := build(t, filepath.Join(root, "scripts"))
	if !res.Written || res.Files != 2 {
		t.Errorf("result %+v", res)
	}
	out := filepath.Join(root, "build", "scripts")
	if got := readFile(t, filepath.Join(out, "main.lua")); !strings.Contains(got, `require("ai.brain")`) {
		t.Errorf("main.lua:\n%s", got)
	}
	readFile(t, filepath.Join(out, "ai", "brain.lua"))
	if got := readFile(t, filepath.Join(out, "legacy", "util.lua")); got != "return { old = true }\n" {
		t.Errorf("util.lua not copied through: %q", got)
	}
	if _, err := os.Stat(filepath.Join(out, "types", "api.lua")); err == nil {
		t.Error("a declaration file produced output")
	}
	entries, _ := os.ReadDir(out)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".lunar-tmp") {
			t.Errorf("staging file left behind: %s", e.Name())
		}
	}
}

// If anything fails, nothing is written: the last good Lua stays in place
// and the game keeps running on it.
func TestFailedBuildWritesNothing(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"a.lunar":           "print(\"a\")\n",
		"b.lunar":           "print(\"b\")\n",
	})
	build(t, root)
	out := filepath.Join(filepath.Dir(root), "build")
	before := readFile(t, filepath.Join(out, "a.lua"))
	stat, _ := os.Stat(filepath.Join(out, "a.lua"))

	time.Sleep(20 * time.Millisecond)
	os.WriteFile(filepath.Join(root, "a.lunar"), []byte("print(\"changed\")\n"), 0o644)
	os.WriteFile(filepath.Join(root, "b.lunar"), []byte("local x: number = \"oops\"\n"), 0o644)

	p, _ := Open(Options{Dir: root})
	res, diags := Build(p)
	if !diags.HasErrors() || res.Written {
		t.Fatalf("expected a failed build, got %+v %v", res, messagesOf(diags))
	}
	if after := readFile(t, filepath.Join(out, "a.lua")); after != before {
		t.Errorf("a.lua changed on a failed build:\n%s", after)
	}
	if again, _ := os.Stat(filepath.Join(out, "a.lua")); !again.ModTime().Equal(stat.ModTime()) {
		t.Error("a.lua was rewritten on a failed build")
	}
}

// A script deleted from the project loses its output, but nothing lunar did
// not write is ever removed.
func TestBuildRemovesOnlyItsOwnStaleOutputs(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"keep.lunar":        "print(\"k\")\n",
		"old.lunar":         "print(\"o\")\n",
	})
	build(t, root)
	out := filepath.Join(filepath.Dir(root), "build")
	os.WriteFile(filepath.Join(out, "notes.txt"), []byte("mine\n"), 0o644)
	os.Remove(filepath.Join(root, "old.lunar"))

	build(t, root)
	if _, err := os.Stat(filepath.Join(out, "old.lua")); !os.IsNotExist(err) {
		t.Error("old.lua was not removed after old.lunar was deleted")
	}
	readFile(t, filepath.Join(out, "keep.lua"))
	if readFile(t, filepath.Join(out, "notes.txt")) != "mine\n" {
		t.Error("a file lunar did not write was touched")
	}
}

func TestLuaNextToLunarIsAnError(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"enemy.lunar":       "print(\"a\")\n",
		"enemy.lua":         "print(\"b\")\n",
	})
	p, _ := Open(Options{Dir: root})
	_, diags := Build(p)
	if !diags.HasErrors() || !strings.Contains(diags[0].Message, "enemy.lua") {
		t.Errorf("got %v", messagesOf(diags))
	}
}

// An output folder inside the scripts folder is not read back as input.
func TestOutDirInsideRootIsNotScanned(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "build"}`,
		"main.lunar":        "print(\"m\")\n",
	})
	build(t, root)
	res := build(t, root)
	if res.Files != 1 {
		t.Errorf("second build compiled %d files", res.Files)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "build")); err == nil {
		t.Error("the output folder was copied into itself")
	}
}
