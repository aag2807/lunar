package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/internal/ast"
	"lunar/internal/diag"
)

// tree writes files under a fresh directory and returns it.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func open(t *testing.T, opts Options) *Project {
	t.Helper()
	p, diags := Open(opts)
	if diags.HasErrors() {
		t.Fatalf("Open: %v", messagesOf(diags))
	}
	return p
}

func messagesOf(l diag.List) []string {
	out := make([]string, len(l))
	for i, d := range l {
		out[i] = d.Location() + ": " + d.Message
	}
	return out
}

func checkClean(t *testing.T, p *Project) *Program {
	t.Helper()
	prog, diags := p.Check()
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", messagesOf(diags))
	}
	return prog
}

func singleDiag(t *testing.T, p *Project) diag.Diagnostic {
	t.Helper()
	_, diags := p.Check()
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", messagesOf(diags))
	}
	return diags[0]
}

func importNamed(prog *Program, rel, spec string) *ast.ImportStatement {
	for _, m := range prog.Modules {
		if m.Rel != rel {
			continue
		}
		for _, stmt := range m.Statements {
			if imp, ok := stmt.(*ast.ImportStatement); ok && imp.Module == spec {
				return imp
			}
		}
	}
	return nil
}

// Relative imports resolve from the importing file, bare ones from the
// project root, and require names are always root-relative, because that
// is how the engine's package.path will find them.
func TestImportsResolveFromTheImportingFile(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"ai/brain.lunar":    "import { clamp } from \"./util\"\nimport { lerp } from \"../shared/math\"\nlocal x: number = clamp(lerp(1, 2, 0.5))\n",
		"ai/util.lunar":     "export function clamp(n: number): number\n    return n\nend\n",
		"shared/math.lunar": "export function lerp(a: number, b: number, t: number): number\n    return a + (b - a) * t\nend\n",
	})
	prog := checkClean(t, open(t, Options{Dir: root}))
	for spec, want := range map[string]string{"./util": "ai.util", "../shared/math": "shared.math"} {
		imp := importNamed(prog, "ai/brain.lunar", spec)
		if imp == nil || imp.RequireName != want {
			t.Errorf("import %q: require name %v, want %q", spec, imp, want)
		}
	}
}

func TestUnresolvedImportIsAnError(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"ai/brain.lunar":    "\nimport { x } from \"./missing\"\n",
	})
	d := singleDiag(t, open(t, Options{Dir: root}))
	if d.Message != "Cannot find module './missing' (looked for ai/missing.lunar)" || d.Line != 2 || !strings.HasSuffix(d.File, filepath.FromSlash("ai/brain.lunar")) {
		t.Errorf("got %s: %s", d.Location(), d.Message)
	}
}

func TestImportCycleIsAnError(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"a.lunar":           "import { b } from \"./b\"\nexport function a(): number\n    return 1\nend\n",
		"b.lunar":           "import { a } from \"./a\"\nexport function b(): number\n    return 2\nend\n",
	})
	_, diags := open(t, Options{Dir: root}).Check()
	found := false
	for _, d := range diags {
		if d.Message == "Import cycle: a.lunar → b.lunar → a.lunar" || d.Message == "Import cycle: b.lunar → a.lunar → b.lunar" {
			found = true
		}
	}
	if !found {
		t.Errorf("no cycle error in %v", messagesOf(diags))
	}
}

func TestImportLeavingTheRootIsAnError(t *testing.T) {
	root := tree(t, map[string]string{
		"scripts/lunar.config.json": `{"outDir": "../build"}`,
		"scripts/main.lunar":        "import { x } from \"../outside\"\n",
		"outside.lunar":             "export const x: number = 1\n",
	})
	d := singleDiag(t, open(t, Options{Dir: filepath.Join(root, "scripts")}))
	if d.Message != "Cannot import '../outside': it is outside the project root" {
		t.Errorf("got %s", d.Message)
	}
}

// An error that crosses files is caught, wherever lunar runs from.
func TestCrossFileErrorsAreCaught(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"mod/math.lunar":    "export function double(n: number): number\n    return n * 2\nend\n",
		"main.lunar":        "import { double, nonexistent } from \"./mod/math\"\nlocal s: string = double(4)\n",
	})
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(t.TempDir()) // not the project: resolution must not depend on it

	_, diags := open(t, Options{Dir: root}).Check()
	msgs := strings.Join(messagesOf(diags), "\n")
	if !strings.Contains(msgs, "Module './mod/math' does not export 'nonexistent'") || !strings.Contains(msgs, "Cannot assign type 'number' to variable 's' of type 'string'") || len(diags) != 2 {
		t.Errorf("got:\n%s", msgs)
	}
}

func TestImportedClassAndInterfaceAreUsable(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"enemy.lunar": `export interface Stats
    atk: number
end

export class Enemy
    hp: number
    constructor(hp: number)
        self.hp = hp
    end
    hit(n: number): void
        self.hp = self.hp - n
    end
end
`,
		"main.lunar": `import { Enemy, Stats } from "./enemy"

class Boss extends Enemy
    constructor()
        super(100)
    end
end

local s: Stats = { atk = 3 }
local e: Enemy = Enemy(3)
local f: Enemy = Enemy.new(4)
e.hit(1)
local b: Boss = Boss()
`,
	})
	prog := checkClean(t, open(t, Options{Dir: root}))
	outs, diags := prog.Emit()
	if len(diags) > 0 {
		t.Fatalf("emit: %v", messagesOf(diags))
	}
	main := string(outputNamed(t, outs, "main.lua"))
	for _, want := range []string{`require("enemy")`, "Enemy.new(3)", "Enemy.new(4)", "e:hit(1)", "setmetatable(Boss, {__index = Enemy})"} {
		if !strings.Contains(main, want) {
			t.Errorf("main.lua lacks %q:\n%s", want, main)
		}
	}
}

func TestWildcardImportBindsAModule(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"mod.lunar":         "export function double(n: number): number\n    return n * 2\nend\n",
		"main.lunar":        "import * as m from \"./mod\"\nlocal x: number = m.double(2)\nlocal y = m.tripple(2)\n",
	})
	d := singleDiag(t, open(t, Options{Dir: root}))
	if !strings.Contains(d.Message, "'tripple'") {
		t.Errorf("got %s", d.Message)
	}
}

func outputNamed(t *testing.T, outs []Output, rel string) []byte {
	t.Helper()
	for _, o := range outs {
		if o.Rel == rel {
			return o.Content
		}
	}
	t.Fatalf("no output %s among %d", rel, len(outs))
	return nil
}

// Shared declarations come from the config (any path) and from every
// .d.lunar under the root.
func TestDeclarationsFromConfigAndRoot(t *testing.T) {
	root := tree(t, map[string]string{
		"engine/api.d.lunar":               "declare function spawn(kind: string): number\n",
		"game/scripts/lunar.config.json":   `{"outDir": "../build", "declarations": ["../../engine"]}`,
		"game/scripts/types/local.d.lunar": "declare function tick(dt: number): void\n",
		"game/scripts/ai/brain.lunar":      "local id: number = spawn(\"imp\")\ntick(0.016)\nprint(id)\n",
	})
	checkClean(t, open(t, Options{Dir: filepath.Join(root, "game", "scripts")}))
}

func TestDeclFlagAddsDeclarations(t *testing.T) {
	root := tree(t, map[string]string{
		"api/engine.d.lunar": "declare function spawn(kind: string): number\n",
		"s/main.lunar":       "local id: number = spawn(\"imp\")\n",
	})
	checkClean(t, open(t, Options{Dir: filepath.Join(root, "s"), OutDir: filepath.Join(root, "out"), Decls: []string{filepath.Join(root, "api")}}))
}

func TestConfigRejectsUnknownKeys(t *testing.T) {
	root := tree(t, map[string]string{"lunar.config.json": `{"rootdir": "scripts"}`})
	_, diags := Open(Options{Dir: root})
	if !diags.HasErrors() || !strings.Contains(diags[0].Message, "rootdir") {
		t.Errorf("got %v", messagesOf(diags))
	}
}

// An imported name's uses lead to its definition in the exporting file,
// which is what go-to-definition, references and rename need.
func TestImportedNamesResolveToTheirDefinition(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"enemy.lunar":       "export class Enemy\n    hp: number\n    constructor(hp: number)\n        self.hp = hp\n    end\nend\n",
		"main.lunar":        "import { Enemy } from \"./enemy\"\nlocal e: Enemy = Enemy(3)\nlocal n: number = e.hp\n",
	})
	prog := checkClean(t, open(t, Options{Dir: root}))
	var main *Module
	for _, m := range prog.Modules {
		if m.Rel == "main.lunar" {
			main = m
		}
	}
	src := main.Source
	for _, at := range []int{strings.Index(src, "Enemy(3)"), strings.LastIndex(src, "hp")} {
		occ, ok := main.Index.At(main.Path, at)
		if !ok {
			t.Fatalf("nothing indexed at %d", at)
		}
		if !strings.HasSuffix(occ.Sym.Decl.File, "enemy.lunar") {
			t.Errorf("%q resolves to %s:%d", occ.Tok.Literal, occ.Sym.Decl.File, occ.Sym.Decl.Line)
		}
	}
}

// A class reached through `import * as m` constructs like one imported by
// name.
func TestNamespaceImportedClassConstructs(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"enemy.lunar": `export class Enemy
    hp: number
    constructor(hp: number)
        self.hp = hp
    end
    hit(n: number): number
        self.hp = self.hp - n
        return self.hp
    end
end
`,
		"main.lunar": `import * as m from "./enemy"
local e = m.Enemy(3)
local f = m.Enemy.new(4)
print(e.hit(1), f.hit(1))
`,
	})
	prog := checkClean(t, open(t, Options{Dir: root}))
	outs, diags := prog.Emit()
	if len(diags) > 0 {
		t.Fatalf("emit: %v", messagesOf(diags))
	}
	main := string(outputNamed(t, outs, "main.lua"))
	for _, want := range []string{"m.Enemy.new(3)", "m.Enemy.new(4)", "e:hit(1)", "f:hit(1)"} {
		if !strings.Contains(main, want) {
			t.Errorf("main.lua lacks %q:\n%s", want, main)
		}
	}
}
