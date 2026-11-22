package project

import (
	"strings"
	"testing"
)

// Data files are loaded with no libraries opened, so anything that needs a
// global, or Lua the code generator builds on library functions, is an
// error in them instead of a crash when the engine loads them.
func TestDataFilesMayNotUseGlobals(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build", "dataFiles": ["data/**"]}`,
		"data/ok.lunar":     "interface Def\n    name: string\n    hp: number\nend\nlocal d: Def = { name = \"imp\", hp = 3 * 2 }\nreturn d\n",
		"data/bad.lunar": `local label = tostring(3)
local greeting = ` + "`hi ${label}`" + `
local merged = { ...{ a = 1 } }
class Thing
end
return { label = label }
`,
		"main.lunar": "print(tostring(1))\n",
	})
	_, diags := open(t, Options{Dir: root}).Check()
	var msgs []string
	for _, d := range diags {
		if !strings.HasSuffix(d.File, "bad.lunar") || d.Code != "data-file" {
			t.Errorf("unexpected: %s %s", d.Location(), d.Message)
		}
		msgs = append(msgs, d.Message)
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{
		"Data file uses the global 'tostring'; data files are loaded with no libraries opened",
		"Data file uses a template string, which needs tostring; data files are loaded with no libraries opened",
		"Data file uses a spread, which needs table.unpack or a helper; data files are loaded with no libraries opened",
		"Data file declares a class, which needs setmetatable; data files are loaded with no libraries opened",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}

// Strings get their methods from the string library's metatable, which a
// state with no libraries does not have.
func TestDataFilesMayNotCallStringMethods(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build", "dataFiles": ["data/**"]}`,
		"data/names.lunar":  "local n = \"imp\"\nconst k = \"orc\"\nreturn { a = n:upper(), b = k.lower() }\n",
	})
	_, diags := open(t, Options{Dir: root}).Check()
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Message)
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{
		"Data file calls the string method 'upper', which needs the string library",
		"Data file calls the string method 'lower', which needs the string library",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in:\n%s", want, joined)
		}
	}
}
