package project

import (
	"path/filepath"
	"strings"
	"testing"
)

// @singleton export class WorldState: the module exports what the
// decorator returned, so the decorator runs before the export table.
func TestDecoratedExportIsWhatTheModuleExports(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"world.lunar": "local function singleton(c: any): any\n    return c.new()\nend\n" +
			"@singleton\nexport class WorldState\n    tick: number = 0\nend\n",
	})
	build(t, root)
	lua := readFile(t, filepath.Join(root, "..", "build", "world.lua"))
	decorated := strings.Index(lua, "WorldState = singleton(WorldState)")
	exported := strings.Index(lua, "return {")
	if decorated < 0 || exported < 0 || decorated > exported {
		t.Errorf("want the decorator before the export table:\n%s", lua)
	}
}
