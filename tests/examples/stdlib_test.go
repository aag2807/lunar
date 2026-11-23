package examples

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"

	"lunar/internal/types"
	"lunar/stdlib"
)

// libraries are the standard library tables checked member by member.
var libraries = []string{"string", "table", "math", "os", "io", "debug", "utf8", "coroutine", "package"}

// The declarations match what Lua has: every member of every library, every
// global, every file-handle method that lua5.3 and lua5.4 report is
// declared for that target, and nothing is declared that they lack.
func TestStdlibMatchesLua(t *testing.T) {
	dump := `for _, lib in ipairs{"string","table","math","os","io","debug","utf8","coroutine","package"} do
  for k in pairs(_G[lib]) do print(lib .. "." .. k) end
end
for k in pairs(_G) do print("_G." .. k) end
for k in pairs(getmetatable(io.stdout).__index) do print("file." .. k) end`
	for _, lua := range interpreters(t) {
		target := "lua53"
		if lua.version == "5.4" {
			target = "lua54"
		}
		out, err := exec.Command(lua.path, "-e", dump).Output()
		if err != nil {
			t.Fatalf("%s: %v", lua.path, err)
		}
		real := map[string]bool{}
		for _, line := range strings.Fields(string(out)) {
			real[line] = true
		}
		declared := declaredStdlib(t, target)

		var missing, phantom []string
		for name := range real {
			// Globals only the standalone interpreter sets, and the
			// library tables themselves, which are declared as values.
			// A file's metatable entries (__gc, __name) are not methods.
			if name == "_G.arg" || strings.HasPrefix(name, "file.__") {
				continue
			}
			if !declared[name] {
				missing = append(missing, name)
			}
		}
		for name := range declared {
			if name == "_G._ENV" { // the environment itself, not a field of _G
				continue
			}
			if !real[name] {
				phantom = append(phantom, name)
			}
		}
		sort.Strings(missing)
		sort.Strings(phantom)
		if len(missing) > 0 {
			t.Errorf("Lua %s has, undeclared: %s", lua.version, strings.Join(missing, " "))
		}
		if len(phantom) > 0 {
			t.Errorf("declared for %s, absent from Lua %s: %s", target, lua.version, strings.Join(phantom, " "))
		}
	}
}

var topLevelDecl = regexp.MustCompile(`(?m)^declare\s+(?:function|const|local)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// declaredStdlib lists what the declarations for target give: lib.member,
// _G.name and file.method.
func declaredStdlib(t *testing.T, target string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	files := append(append([]string{}, stdlib.Core...), stdlib.ForTarget(target)...)
	for _, f := range files {
		src, err := stdlib.Files.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range topLevelDecl.FindAllStringSubmatch(string(src), -1) {
			out["_G."+m[1]] = true
		}
	}
	c := types.NewCheckerWithOptions(types.Options{Target: target})
	if errs := c.Check(nil); len(errs) > 0 {
		t.Fatalf("stdlib for %s: %v", target, errs[0].Message)
	}
	env := c.GetEnv()
	for _, lib := range libraries {
		typ, ok := env.Get(lib)
		iface, isIface := typ.(*types.InterfaceType)
		if !ok || !isIface {
			t.Errorf("%s is declared as %v", lib, typ)
			continue
		}
		out["_G."+lib] = true
		for name := range iface.Properties {
			out[lib+"."+name] = true
		}
		for name := range iface.Methods {
			out[lib+"."+name] = true
		}
		if lib == "io" {
			if file, ok := iface.Properties["stdout"].(*types.InterfaceType); ok {
				for name := range file.Methods {
					out["file."+name] = true
				}
			}
		}
	}
	return out
}
