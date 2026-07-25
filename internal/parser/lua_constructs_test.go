package parser

import (
	"lunar/internal/ast"
	"testing"
)

// `local function f()` is core Lua and has to parse as a function declaration.
func TestParseLocalFunction(t *testing.T) {
	statements := parseSource(t, "local function helper(n: number): number\n\treturn n + 1\nend")

	fn, ok := statements[0].(*ast.FunctionDeclaration)
	if !ok {
		t.Fatalf("expected FunctionDeclaration, got %T", statements[0])
	}

	if !fn.IsLocal {
		t.Error("expected IsLocal to be true")
	}

	if fn.Name.Value != "helper" {
		t.Errorf("expected name 'helper', got %q", fn.Name.Value)
	}
}

func TestParseRepeatUntil(t *testing.T) {
	statements := parseSource(t, "local i = 0\nrepeat\n\ti = i + 1\nuntil i >= 3")

	loop, ok := statements[1].(*ast.RepeatStatement)
	if !ok {
		t.Fatalf("expected RepeatStatement, got %T", statements[1])
	}

	if len(loop.Body.Statements) != 1 {
		t.Errorf("expected 1 statement in the body, got %d", len(loop.Body.Statements))
	}

	if loop.Condition == nil {
		t.Error("expected a condition")
	}
}

// An abstract method has no body, so parsing must stop at the signature and
// leave the members that follow it alone.
func TestParseAbstractMethodWithoutBody(t *testing.T) {
	statements := parseSource(t, "abstract class Vehicle\n\tabstract sound(): string\n\n\tdescribe(): string\n\t\treturn self.sound()\n\tend\nend")

	class, ok := statements[0].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("expected ClassDeclaration, got %T", statements[0])
	}

	if len(class.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(class.Methods))
	}

	if !class.Methods[0].IsAbstract {
		t.Error("expected the first method to be abstract")
	}

	if class.Methods[0].Body != nil {
		t.Error("expected the abstract method to have no body")
	}

	if class.Methods[1].Body == nil {
		t.Error("expected the concrete method to keep its body")
	}
}

// A method with an empty body still has an 'end', and the class parser has to
// step past it or the class terminates early.
func TestParseEmptyMethodBody(t *testing.T) {
	statements := parseSource(t, "class C\n\tnoop(): void\n\tend\n\n\tvalue(): number\n\t\treturn 1\n\tend\nend")

	class, ok := statements[0].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("expected ClassDeclaration, got %T", statements[0])
	}

	if len(class.Methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(class.Methods))
	}

	if class.Methods[0].Body == nil || len(class.Methods[0].Body.Statements) != 0 {
		t.Error("expected the first method to have an empty body")
	}
}

// Hex literals reach the parser as one token and have to be converted with the
// right base; Go's ParseFloat rejects them.
func TestParseHexLiteralValue(t *testing.T) {
	statements := parseSource(t, `local flags = 0xFF`)

	decl := statements[0].(*ast.VariableDeclaration)
	number, ok := decl.Values[0].(*ast.NumberLiteral)
	if !ok {
		t.Fatalf("expected NumberLiteral, got %T", decl.Values[0])
	}

	if number.Value != 255 {
		t.Errorf("expected 0xFF to be 255, got %v", number.Value)
	}
}

// `table` on its own is a documented type meaning a table of anything.
func TestParseBareTableType(t *testing.T) {
	parseSource(t, "local t: table = {}")
	parseSource(t, "local t: table<string, number> = {}")
}

// Imported names may be words Lunar reserves but Lua does not.
func TestImportKeywordNames(t *testing.T) {
	statements := parseSource(t, `import { get, post, type } from "vendor/http"`)

	imp, ok := statements[0].(*ast.ImportStatement)
	if !ok {
		t.Fatalf("expected ImportStatement, got %T", statements[0])
	}

	if len(imp.Names) != 3 {
		t.Errorf("expected 3 imported names, got %d", len(imp.Names))
	}
}

// get and set are only accessor keywords inside a class body.
func TestGetSetUsableAsValues(t *testing.T) {
	parseSource(t, `local value = get("url")`)
	parseSource(t, `set("key", 1)`)
}
