package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/internal/diag"
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

// A data file may import types: that needs nothing at run time, so no
// require is written. Importing a value needs require, which a state with
// no libraries does not have.
func TestDataFilesMayImportOnlyTypes(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build", "dataFiles": ["data/**"]}`,
		"types.lunar":       "export interface EnemyDef\n    name: string\nend\nexport type Id = number\nexport const SPEED = 3\n",
		"data/goblin.lunar": "import { EnemyDef, Id } from \"../types\"\nlocal id: Id = 1\nlocal g: EnemyDef = { name = \"goblin\" }\nreturn g\n",
		"data/orc.lunar":    "import { EnemyDef, SPEED } from \"../types\"\nlocal o: EnemyDef = { name = \"orc\" }\nreturn { def = o, speed = SPEED }\n",
	})
	p := open(t, Options{Dir: root})
	_, diags := p.Check()
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Location()+" "+d.Message)
	}
	if len(diags) != 1 || !strings.HasSuffix(diags[0].File, "orc.lunar") ||
		!strings.Contains(diags[0].Message, "Data file imports the value 'SPEED' from '../types', which needs require") {
		t.Fatalf("diagnostics:\n%s", strings.Join(msgs, "\n"))
	}

	os.WriteFile(filepath.Join(root, "data", "orc.lunar"), []byte("return { name = \"orc\" }\n"), 0o644)
	build(t, root)
	if got := readFile(t, filepath.Join(root, "..", "build", "data", "goblin.lua")); strings.Contains(got, "require") {
		t.Errorf("goblin.lua:\n%s", got)
	}
}

// dataFiles globs match the way globs usually do: ** matches no folder
// too, and a leading ./ or a bare folder name means what it says.
func TestDataFileGlobs(t *testing.T) {
	p := &Project{}
	cases := map[string][]string{
		"data/**/*.lunar": {"data/a.lunar", "data/x/a.lunar"},
		"**/*.lunar":      {"a.lunar", "data/a.lunar"},
		"./data/**":       {"data/a.lunar", "data/x/a.lunar"},
		"data/":           {"data/a.lunar", "data/x/a.lunar"},
		"data":            {"data/a.lunar", "data/x/a.lunar"},
		"data/*.lunar":    {"data/a.lunar"},
	}
	for pattern, rels := range cases {
		p.DataFiles = []string{pattern}
		for _, rel := range rels {
			if !p.isDataFile(rel) {
				t.Errorf("%q does not match %q", pattern, rel)
			}
		}
		if strings.Contains(pattern, "data") && p.isDataFile("other/a.lunar") {
			t.Errorf("%q matches other/a.lunar", pattern)
		}
	}
	p.DataFiles = []string{"data/*.lunar"}
	if p.isDataFile("data/x/a.lunar") {
		t.Errorf("data/*.lunar matches data/x/a.lunar")
	}
}

// A dataFiles pattern that matches no file is a warning: otherwise the
// check is silently off.
func TestDataFilePatternMatchingNothingWarns(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build", "dataFiles": ["dat/**"]}`,
		"data/a.lunar":      "return { a = 1 }\n",
	})
	_, diags := open(t, Options{Dir: root}).Check()
	if len(diags) != 1 || diags[0].Severity != diag.Warning || !strings.Contains(diags[0].Message, "'dat/**' matches no file") {
		t.Errorf("diagnostics: %+v", diags)
	}
}
