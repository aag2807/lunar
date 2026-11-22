package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bigProject writes a game-sized project: modules scripts, each importing
// the one before it, and an engine API of about 3,400 declaration lines.
func bigProject(tb testing.TB, modules int) string {
	tb.Helper()
	root := tb.TempDir()
	write := func(rel, text string) {
		path := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	write("lunar.config.json", `{"outDir": "../build"}`)
	var api strings.Builder
	for i := 0; i < 850; i++ {
		fmt.Fprintf(&api, "declare function engine_call%d(id: number, name: string, scale?: number): number\n", i)
		fmt.Fprintf(&api, "declare interface Thing%d\n    id: number\n    name: string\nend\n", i)
	}
	write("api/engine.d.lunar", api.String())
	for i := 0; i < modules; i++ {
		var src strings.Builder
		if i > 0 {
			fmt.Fprintf(&src, "import { Mob%d } from \"./mob%d\"\n", i-1, i-1)
		}
		fmt.Fprintf(&src, "export class Mob%d\n    hp: number = %d\n    hit(n: number): number\n        self.hp = self.hp - n\n        return engine_call%d(self.hp, \"mob\")\n    end\nend\n", i, i, i%850)
		for j := 0; j < 20; j++ {
			fmt.Fprintf(&src, "local function helper%d(x: number): number\n    local t: Thing%d = { id = x, name = \"t\" }\n    return t.id + %d\nend\n", j, (i+j)%850, j)
		}
		write(fmt.Sprintf("mob%d.lunar", i), src.String())
	}
	return root
}

func BenchmarkCheckAfterOneEdit(b *testing.B) {
	root := bigProject(b, 200)
	leaf := filepath.Join(root, "mob199.lunar")
	text, _ := os.ReadFile(leaf)
	p, diags := Open(Options{Dir: root})
	if diags.HasErrors() {
		b.Fatal(diags[0].Message)
	}
	prev, d := p.Check()
	if d.HasErrors() {
		b.Fatalf("%s %s", d[0].Location(), d[0].Message)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		edited := string(text) + fmt.Sprintf("local edit%d = 1\n", i)
		p, _ := Open(Options{Dir: root, Overlay: map[string]string{leaf: edited}})
		prev, _ = p.CheckReusing(prev)
	}
}
