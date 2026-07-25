package parser

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"testing"
)

func parseSource(t *testing.T, input string) []ast.Statement {
	t.Helper()

	l := lexer.New(input)
	p := New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	return statements
}

func TestParseColonMethodCall(t *testing.T) {
	statements := parseSource(t, `local n = s:len()`)

	decl, ok := statements[0].(*ast.VariableDeclaration)
	if !ok {
		t.Fatalf("expected VariableDeclaration, got %T", statements[0])
	}

	call, ok := decl.Values[0].(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", decl.Values[0])
	}

	dot, ok := call.Function.(*ast.DotExpression)
	if !ok {
		t.Fatalf("expected DotExpression, got %T", call.Function)
	}

	if !dot.IsMethodCall {
		t.Error("expected IsMethodCall to be true for s:len()")
	}

	if got := dot.Right.String(); got != "len" {
		t.Errorf("expected method name 'len', got %q", got)
	}
}

func TestParseChainedColonMethodCalls(t *testing.T) {
	// The receiver of the second call is the result of the first.
	statements := parseSource(t, `local x = s:trim():upper()`)

	decl := statements[0].(*ast.VariableDeclaration)
	outer, ok := decl.Values[0].(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", decl.Values[0])
	}

	outerDot, ok := outer.Function.(*ast.DotExpression)
	if !ok || !outerDot.IsMethodCall {
		t.Fatalf("expected outer method call, got %T", outer.Function)
	}

	if _, ok := outerDot.Left.(*ast.CallExpression); !ok {
		t.Errorf("expected inner call as receiver, got %T", outerDot.Left)
	}
}

// ':' keeps every one of its other meanings; only ': name (' is a method call.
func TestColonStillParsesNonMethodUses(t *testing.T) {
	sources := []struct {
		name  string
		input string
	}{
		{"variable type annotation", `local s: string = "hi"`},
		{"function return type", "function f(): number\n\treturn 1\nend"},
		{"parameter type", "function f(a: number, b: string): void\nend"},
		{"class member types", "class C\n\tprivate x: number\n\n\tm(a: number): number\n\t\treturn a\n\tend\nend"},
		{"interface members", "interface I\n\tname: string\n\tgreet(): string\nend"},
		{"conditional type", `type T = number extends string ? number : string`},
		{"type alias to function", `type Handler = (a: number) => void`},
		{"interface with function-typed property", "interface Lib\n\tinsert: function(t: any, v: any): void\nend"},
		{"optional parameter", "function f(a: number?): void\nend"},
	}

	for _, tc := range sources {
		t.Run(tc.name, func(t *testing.T) {
			parseSource(t, tc.input)
		})
	}
}

// A ':' followed by an identifier that is not a call must stay a type
// annotation, so declarations with class types keep parsing.
func TestColonBeforeNonCallIsNotAMethodCall(t *testing.T) {
	statements := parseSource(t, `local p: Person = other`)

	decl, ok := statements[0].(*ast.VariableDeclaration)
	if !ok {
		t.Fatalf("expected VariableDeclaration, got %T", statements[0])
	}

	if len(decl.Types) == 0 || decl.Types[0] == nil {
		t.Fatal("expected a type annotation on the declaration")
	}

	if got := decl.Types[0].String(); got != "Person" {
		t.Errorf("expected type 'Person', got %q", got)
	}
}

func TestParseColonMethodCallOnSelf(t *testing.T) {
	statements := parseSource(t, "class C\n\tm(): void\n\t\tself:other(1)\n\tend\nend")

	class, ok := statements[0].(*ast.ClassDeclaration)
	if !ok {
		t.Fatalf("expected ClassDeclaration, got %T", statements[0])
	}

	stmt := class.Methods[0].Body.Statements[0].(*ast.ExpressionStatement)
	call := stmt.Expression.(*ast.CallExpression)

	dot, ok := call.Function.(*ast.DotExpression)
	if !ok || !dot.IsMethodCall {
		t.Fatalf("expected self:other() to parse as a method call, got %T", call.Function)
	}
}

// Lunar keywords that Lua does not reserve are legal field names, so a
// discriminated union can be matched on its `type` tag.
func TestStructPatternAcceptsKeywordFieldNames(t *testing.T) {
	statements := parseSource(t, "function f(e: any): string\n\treturn match e with\n\t\t| { type: \"click\", button: b } -> \"click\"\n\t\t| _ -> \"other\"\n\tend\nend")

	if len(statements) == 0 {
		t.Fatal("expected a parsed function declaration")
	}
}

func TestMemberAccessAcceptsKeywordNames(t *testing.T) {
	for _, input := range []string{
		`local x = event.type`,
		`local y = s:match("a")`,
		`local z = obj.from`,
		`local w = obj:with(1)`,
	} {
		t.Run(input, func(t *testing.T) {
			parseSource(t, input)
		})
	}
}

// tryParseGenericType backtracks when '<' turns out to be a comparison; the
// rollback has to restore the lexer too, or the tokens it read are dropped.
func TestLessThanIsNotMistakenForGenerics(t *testing.T) {
	for _, input := range []string{
		"if x < 10 then\n\tprint(1)\nend",
		`local b = x < 3`,
		"while i < n do\n\ti = i + 1\nend",
		"function f(x: any): string\n\treturn match x with\n\t\t| c when c < 300 -> \"small\"\n\t\t| _ -> \"other\"\n\tend\nend",
	} {
		t.Run(input, func(t *testing.T) {
			parseSource(t, input)
		})
	}
}

// Real generic instantiations must still parse after the backtracking fix.
func TestGenericInstantiationStillParses(t *testing.T) {
	for _, input := range []string{
		`local b: Box<number> = other`,
		`local m: Map<string, number> = other`,
		`local n: Box<Box<number>> = other`,
	} {
		t.Run(input, func(t *testing.T) {
			parseSource(t, input)
		})
	}
}

// '?' is an operator only in '?[', so optional types keep parsing.
func TestOptionalIndexAndOptionalTypesCoexist(t *testing.T) {
	for _, input := range []string{
		`local x = items?[1]`,
		`local y = items?[1] ?? 0`,
		`local z: number? = nil`,
		"function f(a: number?): string?\n\treturn nil\nend",
		`type Conditional = number extends string ? number : string`,
	} {
		t.Run(input, func(t *testing.T) {
			parseSource(t, input)
		})
	}
}

func TestParseOptionalIndexSetsFlag(t *testing.T) {
	statements := parseSource(t, `local x = items?[1]`)

	decl := statements[0].(*ast.VariableDeclaration)
	index, ok := decl.Values[0].(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", decl.Values[0])
	}

	if !index.IsOptional {
		t.Error("expected IsOptional to be true for items?[1]")
	}
}

func TestParseOptionalCallSetsFlag(t *testing.T) {
	statements := parseSource(t, `local x = fn?.(1)`)

	decl := statements[0].(*ast.VariableDeclaration)
	call, ok := decl.Values[0].(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", decl.Values[0])
	}

	if !call.IsOptional {
		t.Error("expected IsOptional to be true for fn?.(1)")
	}

	if len(call.Arguments) != 1 {
		t.Errorf("expected 1 argument, got %d", len(call.Arguments))
	}
}
