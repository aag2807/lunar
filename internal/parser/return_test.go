package parser

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strings"
	"testing"
)

func parseWithErrors(input string) ([]ast.Statement, []*ParseError) {
	p := New(lexer.New(input))
	statements := p.Parse()
	return statements, p.Errors()
}

// A data file ends in `return {` with the table on the following lines;
// the value belongs to the return, as it does in Lua.
func TestTopLevelReturnOfMultilineTable(t *testing.T) {
	statements := parseSource(t, "local hp = 10\nreturn {\n    name = \"goblin\",\n    hp = hp,\n}\n")
	ret, ok := statements[len(statements)-1].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("expected the last statement to be a return, got %T", statements[len(statements)-1])
	}
	if len(ret.ReturnValues) != 1 {
		t.Fatalf("return has %d values, want 1", len(ret.ReturnValues))
	}
	if _, ok := ret.ReturnValues[0].(*ast.TableLiteral); !ok {
		t.Fatalf("returned %T, want a table literal", ret.ReturnValues[0])
	}
}

func TestReturnValueOnNextLine(t *testing.T) {
	statements := parseSource(t, "function f(): number\n    return\n        42\nend\n")
	fn := statements[0].(*ast.FunctionDeclaration)
	ret := fn.Body.Statements[0].(*ast.ReturnStatement)
	if len(ret.ReturnValues) != 1 {
		t.Fatalf("return has %d values, want 1", len(ret.ReturnValues))
	}
}

// An empty return must leave the block's terminator for the block.
func TestEmptyReturnBeforeTerminators(t *testing.T) {
	for _, src := range []string{
		"function f()\n    return\nend\nlocal after = 1\n",
		"function f() return end\nlocal after = 1\n",
		"function f(x: boolean)\n    if x then return else return end\nend\nlocal after = 1\n",
		"function f(x: boolean)\n    if x then return elseif x then return end\nend\nlocal after = 1\n",
		"local i = 0\nrepeat\n    i = i + 1\n    if i > 2 then break end\nuntil i > 5\nlocal after = 1\n",
		"function f() return; end\nlocal after = 1\n",
	} {
		statements := parseSource(t, src)
		last, ok := statements[len(statements)-1].(*ast.VariableDeclaration)
		if !ok || last.Names[0].Value != "after" {
			t.Errorf("%q: the statement after the function was lost; got %d statements", src, len(statements))
		}
	}
}

// Lua rejects code after a return in the same block, so Lunar does too
// rather than emitting Lua that does not load.
func TestStatementAfterReturnIsAnError(t *testing.T) {
	for _, src := range []string{
		"function f(): number\n    return 1\n    print(2)\nend\n",
		"return 1\nprint(2)\n",
	} {
		_, errs := parseWithErrors(src)
		if len(errs) != 1 || !strings.Contains(errs[0].Message, "'return' must be the last statement in a block") {
			t.Errorf("%q: got errors %v", src, errs)
		}
	}
}

// Lua's ';' is an empty statement and may separate statements on one line.
func TestSemicolonSeparatesStatements(t *testing.T) {
	statements := parseSource(t, "local a = 1; local b = 2;\n;\nlocal c = 3")
	if len(statements) != 3 {
		t.Fatalf("got %d statements, want 3", len(statements))
	}
}
