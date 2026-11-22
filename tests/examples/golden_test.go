package examples

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden .lua snapshots from the current compiler output")

// TestGoldenOutput pins the generated Lua for every example, so any change in
// code generation shows up as a reviewable diff of testdata/golden.
// Regenerate with: go test ./tests/examples -run TestGoldenOutput -update
func TestGoldenOutput(t *testing.T) {
	out := t.TempDir()
	for _, rel := range exampleSources(t) {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			got := filepath.Join(out, filepath.FromSlash(rel)+".lua")
			compileExample(t, rel, got, "lua53")
			gotBytes, err := os.ReadFile(got)
			if err != nil {
				t.Fatal(err)
			}

			golden := filepath.Join("testdata", "golden", filepath.FromSlash(rel)+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, gotBytes, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("no snapshot for %s (run with -update): %v", rel, err)
			}
			if string(want) != string(gotBytes) {
				t.Errorf("generated Lua for %s differs from %s\n--- want\n%s\n--- got\n%s", rel, golden, want, gotBytes)
			}
		})
	}
}
