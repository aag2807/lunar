package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// The editor opens a folder above the game; each script is analysed with
// the lunar.config.json found from its own folder, as `lunar build` would.
func TestConfigIsFoundFromEachDocument(t *testing.T) {
	root := writeTree(t, map[string]string{
		"game/lunar.config.json":  `{"rootDir": "scripts", "outDir": "../build"}`,
		"game/scripts/util.lunar": "export function f(): number\n    return 1\nend\n",
		"game/scripts/main.lunar": "import { f } from \"util\"\nlocal n: number = f()\n",
		"loose.lunar":             "local x: number = \"s\"\n",
	})
	c := newClientAt(t, root)
	c.open("game/scripts/main.lunar")
	c.open("loose.lunar")
	if d, _ := c.diagnosticsFor("game/scripts/main.lunar"); len(d) != 0 {
		t.Errorf("main.lunar: %+v", d)
	}
	if d, _ := c.diagnosticsFor("loose.lunar"); len(d) != 1 || !strings.Contains(d[0].Message, "Cannot assign") {
		t.Errorf("loose.lunar: %+v", d)
	}
}

// A broken config is shown on the config file and as a message, instead of
// every diagnostic silently disappearing.
func TestConfigErrorIsShown(t *testing.T) {
	root := writeTree(t, map[string]string{
		"lunar.config.json": `{"rootdir": "scripts"}`,
		"main.lunar":        "print(1)\n",
	})
	c := newClientAt(t, root)
	c.open("main.lunar")
	d, _ := c.diagnosticsFor("lunar.config.json")
	if len(d) != 1 || !strings.Contains(d[0].Message, "rootdir") {
		t.Errorf("lunar.config.json diagnostics: %+v", d)
	}
	shown := false
	for _, note := range c.notes {
		if string(note["method"]) == `"window/showMessage"` {
			var p struct{ Message string }
			json.Unmarshal(note["params"], &p)
			shown = shown || strings.Contains(p.Message, "rootdir")
		}
	}
	if !shown {
		t.Error("no window/showMessage about the config")
	}
}

// A script outside its project's rootDir is told so in the editor.
func TestFileOutsideRootDirIsToldSo(t *testing.T) {
	root := writeTree(t, map[string]string{
		"lunar.config.json":  `{"rootDir": "scripts", "outDir": "../build"}`,
		"scripts/main.lunar": "print(1)\n",
		"tools/x.lunar":      "print(2)\n",
	})
	c := newClientAt(t, root)
	c.open("tools/x.lunar")
	if d, _ := c.diagnosticsFor("tools/x.lunar"); len(d) != 1 || !strings.Contains(d[0].Message, "is outside rootDir") {
		t.Errorf("tools/x.lunar: %+v", d)
	}
}
