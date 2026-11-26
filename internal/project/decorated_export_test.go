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
		"world.lunar": "local function singleton<T>(cls: T): InstanceOf<T>\n    return (cls as any).new()\nend\n" +
			"@singleton\nexport class WorldState\n    tick: number = 0\n    advance(): void\n        self.tick = self.tick + 1\n    end\nend\n",
		// The importer uses it as the instance it is.
		"main.lunar": "import { WorldState } from \"./world\"\nWorldState:advance()\nlocal tick: number = WorldState.tick\nprint(tick)\n",
	})
	build(t, root)
	lua := readFile(t, filepath.Join(root, "..", "build", "world.lua"))
	decorated := strings.Index(lua, "WorldState = singleton(WorldState)")
	exported := strings.Index(lua, "return {")
	if decorated < 0 || exported < 0 || decorated > exported {
		t.Errorf("want the decorator before the export table:\n%s", lua)
	}
}
