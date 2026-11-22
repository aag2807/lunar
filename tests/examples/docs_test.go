package examples

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var fence = regexp.MustCompile("(?s)```lunar([^\n]*)\n(.*?)```")

// TestDocumentationExamplesCompile compiles every ```lunar block in the
// README and docs/, so the documentation cannot describe syntax the
// compiler rejects. A block marked ```lunar fragment is part of a larger
// program and is skipped.
func TestDocumentationExamplesCompile(t *testing.T) {
	docs, _ := filepath.Glob(filepath.Join(repoRoot, "docs", "*.md"))
	docs = append([]string{filepath.Join(repoRoot, "README.md")}, docs...)

	count := 0
	for _, doc := range docs {
		data, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		for i, m := range fence.FindAllStringSubmatch(string(data), -1) {
			if strings.Contains(m[1], "fragment") {
				continue
			}
			count++
			name := filepath.Base(doc)
			dir := t.TempDir()
			src := filepath.Join(dir, "example.lunar")
			if err := os.WriteFile(src, []byte(m[2]), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(compilerBin, "--target", "lua54", "example.lunar")
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("%s, lunar block %d does not compile:\n%s\n%s", name, i+1, m[2], out)
			}
		}
	}
	if count == 0 {
		t.Fatal("found no lunar blocks to check")
	}
}
