package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

func checkSource(t *testing.T, input string) []*TypeError {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	return NewChecker("../../stdlib").Check(statements)
}

// `function(params): Return` in type position used to be skipped entirely,
// leaving the member typed as 'any' and leaking the parameter names as extra
// interface properties.
func TestFunctionTypeAnnotationResolves(t *testing.T) {
	input := `
interface Lib
	apply: function(a: number, b: string): boolean
end
`
	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	checker.Check(statements)

	lib, ok := checker.GetEnv().Get("Lib")
	if !ok {
		t.Fatal("interface Lib not registered")
	}

	iface, ok := lib.(*InterfaceType)
	if !ok {
		t.Fatalf("Lib is %T, want *InterfaceType", lib)
	}

	if len(iface.Properties) != 1 {
		t.Errorf("expected exactly 1 property, got %d: %v", len(iface.Properties), iface.Properties)
	}

	fnType, ok := iface.Properties["apply"].(*FunctionType)
	if !ok {
		t.Fatalf("property 'apply' is %T, want *FunctionType", iface.Properties["apply"])
	}

	if len(fnType.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fnType.Parameters))
	}
	if !fnType.ReturnType.Equals(Boolean) {
		t.Errorf("expected boolean return type, got %s", fnType.ReturnType.String())
	}
}

// With real types, wrong arguments to library functions are finally caught.
func TestStdlibCallsAreTypeChecked(t *testing.T) {
	errors := checkSource(t, `local x: number = math.floor("not a number")`)

	if len(errors) == 0 {
		t.Fatal("expected an error for math.floor(string)")
	}
}

// Optional parameters in the declarations keep correct calls quiet.
func TestStdlibOptionalArgumentsAccepted(t *testing.T) {
	input := `
local t: string[] = {}
table.insert(t, "x")
table.insert(t, 1, "y")
table.sort(t)
local joined: string = table.concat(t, ",")
local r: number = math.random()
local r2: number = math.random(1, 10)
local now: number = os.time()
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A function type inside a union must parse, as the stdlib's gsub uses one.
func TestFunctionTypeInsideUnion(t *testing.T) {
	input := `
interface Lib
	gsub: function(s: string, repl: string | function(m: string): string): string
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A method cannot stand in for a function-typed property: they are dispatched
// differently in Lua. The diagnostic has to say why.
func TestMethodForPropertyExplainsDispatch(t *testing.T) {
	errors := checkSource(t, "interface Shape\n\tarea: function(): number\nend\n\nclass Circle implements Shape\n\tarea(): number\n\t\treturn 1\n\tend\nend")

	if len(errors) == 0 {
		t.Fatal("expected a conformance error")
	}

	if !strings.Contains(errors[0].Message, "implicit self") {
		t.Errorf("expected the error to explain method vs property dispatch, got: %s", errors[0].Message)
	}
}

// Declaring it as a method satisfies the interface.
func TestMethodSatisfiesInterfaceMethod(t *testing.T) {
	input := "interface Shape\n\tarea(): number\nend\n\nclass Circle implements Shape\n\tarea(): number\n\t\treturn 1\n\tend\nend"

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A class must be able to name itself in its own member signatures.
func TestClassCanReferenceItsOwnType(t *testing.T) {
	input := `
class Account
	private balance: number

	constructor(balance: number)
		self.balance = balance
	end

	static open(): Account
		return Account(0)
	end

	merge(other: Account): Account
		return Account(self.balance + other.balance)
	end
end
`

	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
