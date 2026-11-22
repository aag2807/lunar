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
	if err := writeOutputs(out, old); err != nil {
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
	err := writeOutputs(out, []Output{{Rel: "a.lua", Content: []byte("new a")}, {Rel: "b.lua", Content: []byte("new b")}, {Rel: "c.lua", Content: []byte("new c")}})
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
