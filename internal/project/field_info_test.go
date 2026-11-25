package project

import (
	"path/filepath"
	"strings"
	"testing"
)

// A field decorator's info names the script as the host knows it: its path
// under rootDir, with forward slashes, whatever folder the build runs from.
func TestFieldInfoNamesTheScriptUnderRootDir(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"enemies/guard.lunar": "local function export(owner: any, name: string, info: any): void\nend\n" +
			"class Guard\n    @export\n    speed: number = 2\nend\n",
	})
	build(t, root)
	lua := readFile(t, filepath.Join(root, "..", "build", "enemies", "guard.lua"))
	if !strings.Contains(lua, `line = 5, file = "enemies/guard.lunar"`) {
		t.Errorf("want line 5 and file enemies/guard.lunar in:\n%s", lua)
	}
}
