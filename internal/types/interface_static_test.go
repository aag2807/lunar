package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestInterfaceStaticMethods(t *testing.T) {
	input := `
interface GraphicsModule
    static clear(r: number, g: number, b: number): void
    static print(text: string, x: number, y: number): void
    width: number
    height: number
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Check that the interface type has StaticProps map populated
	env := checker.GetEnv()
	graphicsType, ok := env.Get("GraphicsModule")
	if !ok {
		t.Fatalf("GraphicsModule not found in environment")
	}

	interfaceType, ok := graphicsType.(*InterfaceType)
	if !ok {
		t.Fatalf("GraphicsModule is not an InterfaceType, got %T", graphicsType)
	}

	if interfaceType.StaticMethods == nil {
		t.Fatal("StaticMethods map is nil")
	}

	// Verify static methods are marked correctly
	if !interfaceType.StaticMethods["clear"] {
		t.Error("clear should be marked as static")
	}

	if !interfaceType.StaticMethods["print"] {
		t.Error("print should be marked as static")
	}

	// Verify all methods exist in the Methods map
	if len(interfaceType.Methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(interfaceType.Methods))
	}

	// Verify properties exist (width, height)
	if len(interfaceType.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(interfaceType.Properties))
	}
}

func TestInterfaceWithoutStaticMembers(t *testing.T) {
	input := `
interface Love
    graphics: any
    window: any
    load(): void
    update(dt: number): void
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	env := checker.GetEnv()
	loveType, ok := env.Get("Love")
	if !ok {
		t.Fatalf("Love not found in environment")
	}

	interfaceType, ok := loveType.(*InterfaceType)
	if !ok {
		t.Fatalf("Love is not an InterfaceType, got %T", loveType)
	}

	// StaticMethods should be initialized but empty
	if interfaceType.StaticMethods == nil {
		t.Fatal("StaticMethods map should be initialized")
	}

	// All methods should be non-static
	for name := range interfaceType.Methods {
		if interfaceType.StaticMethods[name] {
			t.Errorf("method %s should not be static", name)
		}
	}
}

func TestInterfaceMixedStaticAndInstance(t *testing.T) {
	input := `
interface MixedModule
    static create(name: string): any
    static destroy(id: number): void
    instance: any
    getName(): string
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	env := checker.GetEnv()
	mixedType, ok := env.Get("MixedModule")
	if !ok {
		t.Fatalf("MixedModule not found in environment")
	}

	interfaceType, ok := mixedType.(*InterfaceType)
	if !ok {
		t.Fatalf("MixedModule is not an InterfaceType, got %T", mixedType)
	}

	// Verify static methods
	staticMethods := []string{"create", "destroy"}
	for _, name := range staticMethods {
		if !interfaceType.StaticMethods[name] {
			t.Errorf("%s should be marked as static method", name)
		}
	}

	// Verify non-static method
	if interfaceType.StaticMethods["getName"] {
		t.Error("getName should not be marked as static")
	}

	// Verify property is not marked static
	if interfaceType.StaticProps["instance"] {
		t.Error("instance property should not be marked as static")
	}
}

// Note: type declarations only support properties, not methods
// Static keyword only makes sense for interface methods
