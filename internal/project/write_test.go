package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// When one file cannot be replaced (on Windows, a file another program has
// open), every file already replaced is put back: the folder keeps the
// last good build, never half of each.
func TestFailedReplaceLeavesTheLastGoodBuild(t *testing.T) {
	out := t.TempDir()
	old := []Output{{Rel: "a.lua", Content: []byte("old a")}, {Rel: "b.lua", Content: []byte("old b")}, {Rel: "c.lua", Content: []byte("old c")}}
	if err := writeOutputs(out, "/root", old); err != nil {
		t.Fatal(err)
	}

	real := renameFile
	defer func() { renameFile = real }()
	renameFile = func(from, to string) error {
		if strings.HasSuffix(from, "b.lua.lunar-tmp") {
			return errors.New("the file is in use")
		}
		return real(from, to)
	}
	err := writeOutputs(out, "/root", []Output{{Rel: "a.lua", Content: []byte("new a")}, {Rel: "b.lua", Content: []byte("new b")}, {Rel: "c.lua", Content: []byte("new c")}})
	if err == nil || !strings.Contains(err.Error(), "the file is in use") || !strings.Contains(err.Error(), "nothing was changed") {
		t.Fatalf("error: %v", err)
	}

	entries, _ := os.ReadDir(out)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, " ") != ".lunar-manifest.json a.lua b.lua c.lua" {
		t.Errorf("folder holds %v", names)
	}
	for _, name := range []string{"a.lua", "b.lua", "c.lua"} {
		data, _ := os.ReadFile(filepath.Join(out, name))
		if string(data) != "old "+name[:1] {
			t.Errorf("%s = %q", name, data)
		}
	}
}

// A build killed while replacing files leaves its staged copies and the
// old files it moved aside. The next build puts the old files back and
// removes the rest before anything else, even if it then finds errors.
func TestLeftoversOfAKilledBuildAreRecovered(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"main.lunar":        "local x: number = \"broken\"\n",
	})
	out := filepath.Join(filepath.Dir(root), "build")
	os.MkdirAll(filepath.Join(out, "ai"), 0o755)
	os.WriteFile(filepath.Join(out, "main.lua.lunar-old"), []byte("last good main"), 0o644) // main.lua itself is gone
	os.WriteFile(filepath.Join(out, "ai", "brain.lua"), []byte("current brain"), 0o644)
	os.WriteFile(filepath.Join(out, "ai", "brain.lua.lunar-old"), []byte("older brain"), 0o644)
	os.WriteFile(filepath.Join(out, "ai", "brain.lua.lunar-tmp"), []byte("half"), 0o644)

	if _, diags := Build(open(t, Options{Dir: root})); !diags.HasErrors() {
		t.Fatal("expected the type error")
	}
	for path, want := range map[string]string{"main.lua": "last good main", "ai/brain.lua": "current brain"} {
		if data, _ := os.ReadFile(filepath.Join(out, path)); string(data) != want {
			t.Errorf("%s = %q, want %q", path, data, want)
		}
	}
	for _, gone := range []string{"main.lua.lunar-old", "ai/brain.lua.lunar-old", "ai/brain.lua.lunar-tmp"} {
		if _, err := os.Stat(filepath.Join(out, gone)); err == nil {
			t.Errorf("%s left behind", gone)
		}
	}
}

// Two projects writing one outDir would each delete the other's outputs
// as stale. The second is refused.
func TestOutDirOfAnotherProjectIsRefused(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"a", "b"} {
		os.MkdirAll(filepath.Join(base, name), 0o755)
		os.WriteFile(filepath.Join(base, name, "lunar.config.json"), []byte(`{"outDir": "../shared"}`), 0o644)
		os.WriteFile(filepath.Join(base, name, name+".lunar"), []byte("print(1)\n"), 0o644)
	}
	if _, diags := Build(open(t, Options{Dir: filepath.Join(base, "a")})); len(diags) != 0 {
		t.Fatalf("a: %v", messagesOf(diags))
	}
	res, diags := Build(open(t, Options{Dir: filepath.Join(base, "b")}))
	if res.Written || len(diags) != 1 || diags[0].Code != "config" || !strings.Contains(diags[0].Message, "holds the build of another project") {
		t.Fatalf("b: written %v, %v", res.Written, messagesOf(diags))
	}
	if _, err := os.Stat(filepath.Join(base, "shared", "a.lua")); err != nil {
		t.Errorf("a.lua was removed: %v", err)
	}
	// Building a again still works.
	if _, diags := Build(open(t, Options{Dir: filepath.Join(base, "a")})); len(diags) != 0 {
		t.Errorf("a again: %v", messagesOf(diags))
	}
}
