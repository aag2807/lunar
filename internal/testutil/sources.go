// Package testutil holds helpers for the test packages that build and run
// the compiler binary instead of importing it.
package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TouchCompilerSources reads every file the compiler is built from: the Go
// sources under cmd/ and internal/, go.mod and the stdlib declarations.
//
// Packages such as tests/examples run `go build` in a subprocess and import
// nothing from the compiler, so Go's test cache does not know their results
// depend on it and replays a stale "ok" after compiler changes. Files a test
// opens are part of the cache key, so reading them here makes the cache
// invalidate when any of them changes. Only files opened while tests run
// count, not those opened in TestMain, so call this from a test.
func TouchCompilerSources(repoRoot string) error {
	for _, dir := range []string{"cmd", "internal", "stdlib"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !(strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".lunar")) {
				return nil
			}
			_, err = os.ReadFile(path)
			return err
		})
		if err != nil {
			return err
		}
	}
	_, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	return err
}
