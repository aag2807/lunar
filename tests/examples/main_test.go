// Package examples compiles every program under examples/ with the real
// compiler binary, compares the Lua against stored snapshots, and runs the
// feature programs under each available Lua interpreter.
package examples

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var (
	compilerBin string
	repoRoot    string
)

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	wd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	repoRoot = filepath.Clean(filepath.Join(wd, "..", ".."))

	tmp, err := os.MkdirTemp("", "lunar-examples-bin")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)

	compilerBin = filepath.Join(tmp, "lunar")
	if runtime.GOOS == "windows" {
		compilerBin += ".exe"
	}
	build := exec.Command("go", "build", "-o", compilerBin, "./cmd/lunar")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("building lunar: %v\n%s", err, out)
	}
	return m.Run(), nil
}

// exampleSources lists every .lunar program under examples/ (declaration
// files excluded), relative to examples/, in a stable order.
func exampleSources(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(repoRoot, "examples")
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".lunar") || strings.HasSuffix(path, ".d.lunar") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

// compileExample compiles examples/<rel> to outFile. The compiler runs from
// the example's own directory so relative imports resolve as they would for
// a user compiling in place.
func compileExample(t *testing.T, rel, outFile, target string) {
	t.Helper()
	src := filepath.Join(repoRoot, "examples", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(compilerBin, "--target", target, "-o", outFile, filepath.Base(src))
	cmd.Dir = filepath.Dir(src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compiling %s: %v\n%s", rel, err, out)
	}
}
