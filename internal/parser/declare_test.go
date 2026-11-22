package parser

import (
	"lunar/internal/ast"
	"testing"
)

func declaredNames(t *testing.T, statements []ast.Statement) []string {
	t.Helper()
	var names []string
	for _, stmt := range statements {
		ds, ok := stmt.(*ast.DeclareStatement)
		if !ok {
			t.Fatalf("expected DeclareStatement, got %T", stmt)
		}
		switch d := ds.Declaration.(type) {
		case *ast.FunctionDeclaration:
			names = append(names, d.Name.Value)
		case *ast.ClassDeclaration:
			names = append(names, d.Name.Value)
		case *ast.NamespaceDeclaration:
			names = append(names, d.Name.Value)
		default:
			t.Fatalf("unexpected declaration %T", d)
		}
	}
	return names
}

func assertNames(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got declarations %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got declarations %v, want %v", got, want)
		}
	}
}

// A declaration has no body, so the trailing 'end' is optional. Leaving it
// out used to swallow every following declaration as the body.
func TestDeclareFunctionEndIsOptional(t *testing.T) {
	statements := parseSource(t, `declare function spawn(name: string): number
declare function despawn(id: number): void end
declare function tick(dt: number): void
declare function log(...args: any): void end`)
	assertNames(t, declaredNames(t, statements), "spawn", "despawn", "tick", "log")
	for _, stmt := range statements {
		if fn := stmt.(*ast.DeclareStatement).Declaration.(*ast.FunctionDeclaration); fn.Body != nil {
			t.Errorf("declare function %s has a body", fn.Name.Value)
		}
	}
}

// Members of a declared class are signatures; the class's own 'end' closes it.
func TestDeclareClassMembersAreSignatures(t *testing.T) {
	statements := parseSource(t, `declare class Sprite
    x: number
    constructor(path: string)
    draw(): void
    static load(path: string): Sprite
    get width(): number
end
declare function after(): void`)
	assertNames(t, declaredNames(t, statements), "Sprite", "after")

	class := statements[0].(*ast.DeclareStatement).Declaration.(*ast.ClassDeclaration)
	if len(class.Properties) != 1 || len(class.Methods) != 2 || class.Constructor == nil || len(class.Getters) != 1 {
		t.Fatalf("got %d properties, %d methods, constructor %v, %d getters",
			len(class.Properties), len(class.Methods), class.Constructor != nil, len(class.Getters))
	}
	for _, m := range class.Methods {
		if m.Body != nil {
			t.Errorf("declared method %s has a body", m.Name.Value)
		}
	}
}

func TestDeclareNamespace(t *testing.T) {
	statements := parseSource(t, `declare namespace audio
    function play(name: string, volume?: number): void
    function stop(name: string): void
    const masterVolume: number
end
declare function after(): void`)
	assertNames(t, declaredNames(t, statements), "audio", "after")

	ns := statements[0].(*ast.DeclareStatement).Declaration.(*ast.NamespaceDeclaration)
	if len(ns.Statements) != 3 {
		t.Fatalf("namespace has %d members, want 3", len(ns.Statements))
	}
}
