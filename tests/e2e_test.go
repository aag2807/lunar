package tests

import (
	"lunar/internal/codegen"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// E2E test helper
func compileAndCheck(t *testing.T, lunarCode string) (string, []string) {
	// Lex
	l := lexer.New(lunarCode)

	// Parse
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Type check
	checker := types.NewChecker()
	typeErrors := checker.Check(program)

	var errorMessages []string
	for _, err := range typeErrors {
		errorMessages = append(errorMessages, err.Message)
	}

	// Generate code
	gen := codegen.New()
	luaCode := gen.Generate(program)

	return luaCode, errorMessages
}

// TestE2ESimpleProgram tests end-to-end compilation of a simple program
func TestE2ESimpleProgram(t *testing.T) {
	input := `
local x: number = 42
local y: number = 10
local result: number = x + y
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "local x = 42") {
		t.Errorf("Generated Lua doesn't contain expected code")
	}
}

// TestE2EFunctionDeclaration tests function compilation
func TestE2EFunctionDeclaration(t *testing.T) {
	input := `
function add(a: number, b: number): number
	return a + b
end

local result: number = add(5, 3)
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "function add") {
		t.Errorf("Generated Lua doesn't contain function declaration")
	}
}

// TestE2EClassDeclaration tests class compilation
func TestE2EClassDeclaration(t *testing.T) {
	input := `
class Counter
	value: number

	function constructor()
		self.value = 0
	end

	function increment(): void
		self.value = self.value + 1
	end

	function getValue(): number
		return self.value
	end
end

local counter = Counter()
counter.increment()
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "Counter = {}") {
		t.Errorf("Generated Lua doesn't contain class declaration")
	}
}

// TestE2EGenericFunction tests generic function compilation
func TestE2EGenericFunction(t *testing.T) {
	input := `
function identity<T>(x: T): T
	return x
end

local n: number = identity<number>(42)
local s: string = identity<string>("hello")
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "function identity") {
		t.Errorf("Generated Lua doesn't contain function")
	}
}

// TestE2ETypeAlias tests type alias usage
func TestE2ETypeAlias(t *testing.T) {
	input := `
type Nullable<T> = T | nil

local value: Nullable<number> = 42
local maybeString: Nullable<string> = "hello"
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	// Type aliases don't generate runtime code
	if !strings.Contains(lua, "local value = 42") {
		t.Errorf("Generated Lua doesn't contain variable")
	}
}

// TestE2EInterfaceImplementation tests interface and class declarations
func TestE2EInterfaceImplementation(t *testing.T) {
	input := `
interface Drawable
	x: number
	y: number
end

class Circle
	x: number
	y: number
	radius: number

	function constructor(x: number, y: number, r: number)
		self.x = x
		self.y = y
		self.radius = r
	end

	function getX(): number
		return self.x
	end
end

local circle = Circle(10, 20, 5)
local xCoord: number = circle.getX()
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "Circle = {}") {
		t.Errorf("Generated Lua doesn't contain class")
	}
}

// TestE2EConditionalTypes tests advanced conditional types
func TestE2EConditionalTypes(t *testing.T) {
	input := `
type IsString<T> = T extends string ? true : false

type Result1 = IsString<string>
type Result2 = IsString<number>

local x = 42
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	// Types don't generate runtime code, just verify compilation works
	if !strings.Contains(lua, "local x = 42") {
		t.Errorf("Generated Lua doesn't contain variable")
	}
}

// TestE2EMappedTypes tests mapped types compilation
func TestE2EMappedTypes(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

type Partial<T> = { [K in keyof T]?: T[K] }
type PartialPerson = Partial<Person>

local x = 42
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	// Mapped types don't generate runtime code
	if !strings.Contains(lua, "local x = 42") {
		t.Errorf("Generated Lua doesn't contain variable")
	}
}

// TestE2ETemplateLiteralTypes tests template literal types
func TestE2ETemplateLiteralTypes(t *testing.T) {
	input := "type Direction = \"left\" | \"right\"\n" +
		"type ButtonType = `${Direction}Button`\n\n" +
		"local x = 42"

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "local x = 42") {
		t.Errorf("Generated Lua doesn't contain variable")
	}
}

// TestE2ETypeErrorDetection tests that type errors are caught
func TestE2ETypeErrorDetection(t *testing.T) {
	input := `
local x: number = "string"
`

	_, errors := compileAndCheck(t, input)

	if len(errors) == 0 {
		t.Errorf("Expected type error but got none")
	}

	// Should contain error about type mismatch
	found := false
	for _, err := range errors {
		if strings.Contains(err, "Cannot assign") || strings.Contains(err, "not assignable") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected type mismatch error, got: %v", errors)
	}
}

// TestE2EComplexProgram tests a complex program with multiple features
func TestE2EComplexProgram(t *testing.T) {
	input := `
interface Config
	host: string
	port: number
end

type Partial<T> = { [K in keyof T]?: T[K] }

class Server
	host: string
	port: number

	function constructor(h: string, p: number)
		self.host = h
		self.port = p
	end

	function getUrl(): string
		return self.host
	end

	function getPort(): number
		return self.port
	end
end

function createServer(host: string, port: number): Server
	return Server(host, port)
end

local server = createServer("localhost", 8080)
local url: string = server.getUrl()
local port: number = server.getPort()
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Errorf("Unexpected type errors: %v", errors)
	}

	if !strings.Contains(lua, "Server = {}") {
		t.Errorf("Generated Lua doesn't contain Server class")
	}

	if !strings.Contains(lua, "function createServer") {
		t.Errorf("Generated Lua doesn't contain createServer function")
	}
}

// TestE2ELuaExecution tests that generated Lua actually runs (requires lua installed)
func TestE2ELuaExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Lua execution test in short mode")
	}

	// Check if lua is available
	if _, err := exec.LookPath("lua"); err != nil {
		t.Skip("Lua not installed, skipping execution test")
	}

	input := `
function factorial(n: number): number
	if n <= 1 then
		return 1
	else
		return n * factorial(n - 1)
	end
end

local result = factorial(5)
print(result)
`

	lua, errors := compileAndCheck(t, input)

	if len(errors) > 0 {
		t.Fatalf("Type errors: %v", errors)
	}

	// Write Lua code to temp file
	tmpfile, err := os.CreateTemp("", "lunar_test_*.lua")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(lua)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Execute with lua
	cmd := exec.Command("lua", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Lua execution failed: %v\nOutput: %s", err, output)
	}

	// Should print 120 (5 factorial)
	if !strings.Contains(string(output), "120") {
		t.Errorf("Expected output to contain 120, got: %s", output)
	}
}
