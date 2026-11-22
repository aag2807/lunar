package project

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// Paths in lunar.config.json may be absolute: they are used as given, not
// joined under the config's folder.
func TestAbsolutePathsInConfig(t *testing.T) {
	root := tree(t, map[string]string{
		"engine/api/engine.d.lunar": "declare function spawn(kind: string): number\n",
		"game/scripts/main.lunar":   "spawn(\"goblin\")\n",
	})
	out := filepath.Join(root, "elsewhere", "build")
	cfg := fmt.Sprintf(`{"rootDir": %q, "outDir": %q, "declarations": [%q]}`,
		filepath.Join(root, "game", "scripts"), out, filepath.Join(root, "engine", "api"))
	os.WriteFile(filepath.Join(root, "game", ConfigName), []byte(cfg), 0o644)
	build(t, filepath.Join(root, "game"))
	readFile(t, filepath.Join(out, "main.lua"))
}

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// Folders reached through symlinks are scanned: the root itself, a
// subfolder and a declarations folder.
func TestSymlinkedFoldersAreScanned(t *testing.T) {
	root := tree(t, map[string]string{
		"real/main.lunar":        "import { V } from \"lib/util\"\nprint(V, spawn(\"x\"))\n",
		"ext/util.lunar":         "export const V = 1\n",
		"api/engine.d.lunar":     "declare function spawn(kind: string): number\n",
		"proj/lunar.config.json": `{"rootDir": "scripts", "outDir": "../build", "declarations": ["../apilink"]}`,
	})
	symlinkOrSkip(t, filepath.Join(root, "real"), filepath.Join(root, "proj", "scripts"))
	symlinkOrSkip(t, filepath.Join(root, "ext"), filepath.Join(root, "real", "lib"))
	symlinkOrSkip(t, filepath.Join(root, "api"), filepath.Join(root, "apilink"))
	res := build(t, filepath.Join(root, "proj"))
	if res.Files != 2 {
		t.Errorf("compiled %d files, want 2", res.Files)
	}
	readFile(t, filepath.Join(root, "build", "main.lua"))
	readFile(t, filepath.Join(root, "build", "lib", "util.lua"))
}

// A symlink loop is not followed forever.
func TestSymlinkLoopIsScannedOnce(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"main.lunar":        "print(1)\n",
	})
	symlinkOrSkip(t, root, filepath.Join(root, "again"))
	if res := build(t, root); res.Files != 1 {
		t.Errorf("compiled %d files, want 1", res.Files)
	}
}

// The same project reached through another path is the same project.
func TestSameProjectThroughALink(t *testing.T) {
	root := tree(t, map[string]string{
		"game/lunar.config.json": `{"outDir": "../build"}`,
		"game/main.lunar":        "print(1)\n",
	})
	build(t, filepath.Join(root, "game"))
	symlinkOrSkip(t, filepath.Join(root, "game"), filepath.Join(root, "via"))
	build(t, filepath.Join(root, "via"))
}

// The manifest is part of the build: replacing a read-only one is not a
// half-written build.
func TestReadOnlyManifestIsReplaced(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only files cannot be replaced on Windows")
	}
	root := tree(t, map[string]string{
		"game/lunar.config.json": `{"outDir": "../build"}`,
		"game/a.lunar":           "print(1)\n",
	})
	build(t, filepath.Join(root, "game"))
	manifest := filepath.Join(root, "build", manifestName)
	os.Chmod(manifest, 0o444)
	os.WriteFile(filepath.Join(root, "game", "a.lunar"), []byte("print(2)\n"), 0o644)
	os.WriteFile(filepath.Join(root, "game", "b.lunar"), []byte("print(3)\n"), 0o644)
	res := build(t, filepath.Join(root, "game"))
	if !res.Written || !strings.Contains(readFile(t, manifest), "b.lua") {
		t.Errorf("result %+v, manifest %s", res, readFile(t, manifest))
	}
}

// A project with nothing to write still records an empty build.
func TestEmptyProjectBuilds(t *testing.T) {
	root := tree(t, map[string]string{
		"game/lunar.config.json": `{"outDir": "../build"}`,
		"game/api.d.lunar":       "declare function spawn(kind: string): number\n",
	})
	build(t, filepath.Join(root, "game"))
}

// Two builds of one project at once: the second waits for the first, and
// every output is there after both.
func TestConcurrentBuildsKeepEveryOutput(t *testing.T) {
	files := map[string]string{"game/lunar.config.json": `{"outDir": "../build"}`}
	for i := 0; i < 40; i++ {
		files[fmt.Sprintf("game/m%02d.lunar", i)] = fmt.Sprintf("export const V = %d\n", i)
	}
	root := tree(t, files)
	dir := filepath.Join(root, "game")
	build(t, dir)
	for round := 0; round < 8; round++ {
		var wg sync.WaitGroup
		errs := make(chan string, 2)
		for k := 0; k < 2; k++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p, diags := Open(Options{Dir: dir})
				if diags.HasErrors() {
					errs <- diags[0].Message
					return
				}
				if _, diags := Build(p); diags.HasErrors() {
					errs <- diags[0].Message
				}
			}()
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			t.Errorf("round %d: %s", round, e)
		}
		for i := 0; i < 40; i++ {
			if _, err := os.Stat(filepath.Join(root, "build", fmt.Sprintf("m%02d.lua", i))); err != nil {
				t.Fatalf("round %d: %v", round, err)
			}
		}
	}
}
