package types

import (
	"os"
	"path/filepath"
	"testing"

	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
)

func parseFile(t *testing.T, path, src string) []ast.Statement {
	t.Helper()
	p := parser.New(lexer.NewFile(path, src))
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("%s: parser errors: %v", path, p.Errors())
	}
	return statements
}

// An error inside a declaration file is reported against that file, with
// its own line, not against the script being compiled.
func TestErrorsNameTheirOwnFile(t *testing.T) {
	decls := parseFile(t, "engine/api.d.lunar", "declare function spawn(kind: Monster): number\n")
	main := parseFile(t, "scripts/main.lunar", "local id: number = spawn(\"imp\")\nlocal s: string = 1\n")

	errs := NewChecker().Check(append(decls, main...))
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %v", messages(errs))
	}
	byFile := map[string]*TypeError{}
	for _, e := range errs {
		byFile[e.File] = e
	}
	api, ok := byFile["engine/api.d.lunar"]
	if !ok || api.Line != 1 || api.Column != 30 {
		t.Errorf("declaration error: %+v", api)
	}
	if api != nil && (api.EndLine != 1 || api.EndColumn != 37) {
		t.Errorf("declaration error span ends %d:%d, want 1:37 (after 'Monster')", api.EndLine, api.EndColumn)
	}
	if m, ok := byFile["scripts/main.lunar"]; !ok || m.Line != 2 {
		t.Errorf("script error: %+v", m)
	}
}

// The bundled declarations themselves must check cleanly; any error here
// would be reported against every script.
func TestStdlibChecksCleanly(t *testing.T) {
	if errs := NewChecker().Check(nil); len(errs) != 0 {
		t.Fatalf("stdlib produced %v", messages(errs))
	}
}

// A broken stdlib file used to be skipped silently, leaving print and the
// rest undefined with no hint why.
func TestBrokenStdlibFileIsReported(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lua.d.lunar"), []byte("declare function print(\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	errs := NewChecker(dir).Check(nil)
	if len(errs) == 0 {
		t.Fatal("a stdlib parse error was not reported")
	}
	if errs[0].File != "<stdlib>/lua.d.lunar" {
		t.Errorf("reported against %q, want <stdlib>/lua.d.lunar", errs[0].File)
	}
}
