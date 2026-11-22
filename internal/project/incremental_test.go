package project

import (
	"path/filepath"
	"testing"
)

func moduleNamed(t *testing.T, prog *Program, rel string) *Module {
	t.Helper()
	for _, m := range prog.Modules {
		if m.Rel == rel {
			return m
		}
	}
	t.Fatalf("no module %s", rel)
	return nil
}

func sameDiagnostics(t *testing.T, step string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: diagnostics %v, a full check gives %v", step, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: diagnostics %v, a full check gives %v", step, got, want)
		}
	}
}

// After an edit, only the edited module and the modules that import it are
// checked again; the rest are taken from the previous check, and the
// diagnostics are exactly those of a full check.
func TestCheckReusingRechecksOnlyWhatChanged(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"api.d.lunar":       "declare function spawn(): number\n",
		"a.lunar":           "export function f(): number\n    return spawn()\nend\n",
		"b.lunar":           "import { f } from \"./a\"\nlocal n: string = f()\n",
		"c.lunar":           "local z: number = 1\n",
	})
	at := func(rel string) string { return filepath.Join(root, rel) }
	step := func(prev *Program, overlay map[string]string) (*Program, []string) {
		p := open(t, Options{Dir: root, Overlay: overlay})
		prog, diags := p.CheckReusing(prev)
		_, full := open(t, Options{Dir: root, Overlay: overlay}).Check()
		sameDiagnostics(t, "step", messagesOf(diags), messagesOf(full))
		return prog, messagesOf(diags)
	}

	first, diags := step(nil, nil)
	if len(diags) != 1 {
		t.Fatalf("want b's type error, got %v", diags)
	}

	// Editing c leaves a and b as they were.
	second, _ := step(first, map[string]string{at("c.lunar"): "local z: number = 2\n"})
	for _, rel := range []string{"a.lunar", "b.lunar"} {
		if moduleNamed(t, second, rel).Index != moduleNamed(t, first, rel).Index {
			t.Errorf("%s was checked again after an edit to c.lunar", rel)
		}
	}
	if moduleNamed(t, second, "c.lunar").Index == moduleNamed(t, first, "c.lunar").Index {
		t.Error("c.lunar was not checked again")
	}

	// Editing a checks a and b, its importer, again; b's error goes away.
	third, diags := step(second, map[string]string{at("c.lunar"): "local z: number = 2\n",
		at("a.lunar"): "export function f(): string\n    return \"x\"\nend\n"})
	if len(diags) != 0 {
		t.Errorf("after fixing a: %v", diags)
	}
	if moduleNamed(t, third, "b.lunar").Index == moduleNamed(t, second, "b.lunar").Index {
		t.Error("b.lunar was not checked again after its import changed")
	}
	if moduleNamed(t, third, "c.lunar").Index != moduleNamed(t, second, "c.lunar").Index {
		t.Error("c.lunar was checked again")
	}

	// A change to the declarations checks everything again.
	fourth, _ := step(third, map[string]string{at("c.lunar"): "local z: number = 2\n",
		at("a.lunar"):     "export function f(): string\n    return \"x\"\nend\n",
		at("api.d.lunar"): "declare function spawn(): number\ndeclare function other(): number\n"})
	if moduleNamed(t, fourth, "c.lunar").Index == moduleNamed(t, third, "c.lunar").Index {
		t.Error("c.lunar was reused after the declarations changed")
	}
}
