// Package stdlib holds the declaration files for Lua's standard library.
// They are compiled into the lunar binary, so the type checker knows print,
// string, table and the rest wherever lunar is installed.
package stdlib

import "embed"

// Files holds every *.d.lunar in this directory.
//
//go:embed *.d.lunar
var Files embed.FS

// Core lists the declarations every script sees, whatever its target.
var Core = []string{
	"lua.d.lunar",
	"table.d.lunar",
	"string.d.lunar",
	"math.d.lunar",
	"utf8.d.lunar",
	"os.d.lunar",
	"io.d.lunar",
	"coroutine.d.lunar",
	"debug.d.lunar",
	"package.d.lunar",
}

// ForTarget lists the extra declarations for one Lua version.
func ForTarget(target string) []string {
	if target == "lua54" {
		return []string{"lua54.d.lunar"}
	}
	return nil
}
