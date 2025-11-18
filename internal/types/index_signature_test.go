package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestInterfaceIndexSignatureString(t *testing.T) {
	input := `
interface StringMap
	[key: string]: number
end

local map: StringMap
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the interface has an index signature
	stringMapType, ok := checker.interfaces["StringMap"]
	if !ok {
		t.Fatal("Expected StringMap interface to be registered")
	}

	if stringMapType.IndexSignature == nil {
		t.Fatal("Expected StringMap to have an index signature")
	}

	// Check key type is string
	if stringMapType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", stringMapType.IndexSignature.KeyType)
	}

	// Check value type is number
	if stringMapType.IndexSignature.ValueType != Number {
		t.Errorf("Expected value type to be number, got %v", stringMapType.IndexSignature.ValueType)
	}
}

func TestInterfaceIndexSignatureNumber(t *testing.T) {
	input := `
interface NumberArray
	[index: number]: string
end

local arr: NumberArray
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the interface has an index signature
	numberArrayType, ok := checker.interfaces["NumberArray"]
	if !ok {
		t.Fatal("Expected NumberArray interface to be registered")
	}

	if numberArrayType.IndexSignature == nil {
		t.Fatal("Expected NumberArray to have an index signature")
	}

	// Check key type is number
	if numberArrayType.IndexSignature.KeyType != Number {
		t.Errorf("Expected key type to be number, got %v", numberArrayType.IndexSignature.KeyType)
	}

	// Check value type is string
	if numberArrayType.IndexSignature.ValueType != String {
		t.Errorf("Expected value type to be string, got %v", numberArrayType.IndexSignature.ValueType)
	}
}

func TestInterfaceIndexSignatureWithProperties(t *testing.T) {
	input := `
interface Dictionary
	[key: string]: string
	length: number
end

local dict: Dictionary
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the interface has both property and index signature
	dictType, ok := checker.interfaces["Dictionary"]
	if !ok {
		t.Fatal("Expected Dictionary interface to be registered")
	}

	// Check property
	lengthProp, ok := dictType.Properties["length"]
	if !ok {
		t.Fatal("Expected Dictionary to have 'length' property")
	}
	if lengthProp != Number {
		t.Errorf("Expected length property to be number, got %v", lengthProp)
	}

	// Check index signature
	if dictType.IndexSignature == nil {
		t.Fatal("Expected Dictionary to have an index signature")
	}
	if dictType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", dictType.IndexSignature.KeyType)
	}
	if dictType.IndexSignature.ValueType != String {
		t.Errorf("Expected value type to be string, got %v", dictType.IndexSignature.ValueType)
	}
}

func TestClassIndexSignature(t *testing.T) {
	input := `
class Storage
	[key: string]: any
end

local storage: Storage
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the class has an index signature
	storageType, ok := checker.classes["Storage"]
	if !ok {
		t.Fatal("Expected Storage class to be registered")
	}

	if storageType.IndexSignature == nil {
		t.Fatal("Expected Storage to have an index signature")
	}

	// Check key type is string
	if storageType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", storageType.IndexSignature.KeyType)
	}

	// Check value type is any
	if storageType.IndexSignature.ValueType != Any {
		t.Errorf("Expected value type to be any, got %v", storageType.IndexSignature.ValueType)
	}
}

func TestClassIndexSignatureWithProperties(t *testing.T) {
	input := `
class DataStore
	[key: string]: number
	name: string

	constructor(n: string)
	end
end

local store: DataStore
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the class has both property and index signature
	storeType, ok := checker.classes["DataStore"]
	if !ok {
		t.Fatal("Expected DataStore class to be registered")
	}

	// Check property
	nameProp, ok := storeType.Properties["name"]
	if !ok {
		t.Fatal("Expected DataStore to have 'name' property")
	}
	if nameProp != String {
		t.Errorf("Expected name property to be string, got %v", nameProp)
	}

	// Check index signature
	if storeType.IndexSignature == nil {
		t.Fatal("Expected DataStore to have an index signature")
	}
	if storeType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", storeType.IndexSignature.KeyType)
	}
	if storeType.IndexSignature.ValueType != Number {
		t.Errorf("Expected value type to be number, got %v", storeType.IndexSignature.ValueType)
	}

	// Check constructor
	if storeType.Constructor == nil {
		t.Fatal("Expected DataStore to have a constructor")
	}
}

func TestInterfaceIndexSignatureWithMethods(t *testing.T) {
	input := `
interface ApiClient
	[key: string]: any
	get(url: string): string
end

local client: ApiClient
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the interface has both method and index signature
	clientType, ok := checker.interfaces["ApiClient"]
	if !ok {
		t.Fatal("Expected ApiClient interface to be registered")
	}

	// Check method
	getMethod, ok := clientType.Methods["get"]
	if !ok {
		t.Fatal("Expected ApiClient to have 'get' method")
	}
	if len(getMethod.Parameters) != 1 || getMethod.Parameters[0] != String {
		t.Errorf("Expected get method to have one string parameter")
	}
	if getMethod.ReturnType != String {
		t.Errorf("Expected get method to return string, got %v", getMethod.ReturnType)
	}

	// Check index signature
	if clientType.IndexSignature == nil {
		t.Fatal("Expected ApiClient to have an index signature")
	}
	if clientType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", clientType.IndexSignature.KeyType)
	}
	if clientType.IndexSignature.ValueType != Any {
		t.Errorf("Expected value type to be any, got %v", clientType.IndexSignature.ValueType)
	}
}

func TestClassIndexSignatureWithMethods(t *testing.T) {
	input := `
class Cache
	[key: string]: string

	set(key: string, value: string): void
		local x: number = 1
	end

	get(key: string): string
		return "test"
	end
end

local cache: Cache
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the class has both methods and index signature
	cacheType, ok := checker.classes["Cache"]
	if !ok {
		t.Fatal("Expected Cache class to be registered")
	}

	// Check set method
	setMethod, ok := cacheType.Methods["set"]
	if !ok {
		t.Fatal("Expected Cache to have 'set' method")
	}
	if len(setMethod.Parameters) != 2 {
		t.Errorf("Expected set method to have two parameters")
	}
	if setMethod.ReturnType != Void {
		t.Errorf("Expected set method to return void, got %v", setMethod.ReturnType)
	}

	// Check get method
	getMethod, ok := cacheType.Methods["get"]
	if !ok {
		t.Fatal("Expected Cache to have 'get' method")
	}
	if len(getMethod.Parameters) != 1 || getMethod.Parameters[0] != String {
		t.Errorf("Expected get method to have one string parameter")
	}
	if getMethod.ReturnType != String {
		t.Errorf("Expected get method to return string, got %v", getMethod.ReturnType)
	}

	// Check index signature
	if cacheType.IndexSignature == nil {
		t.Fatal("Expected Cache to have an index signature")
	}
	if cacheType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", cacheType.IndexSignature.KeyType)
	}
	if cacheType.IndexSignature.ValueType != String {
		t.Errorf("Expected value type to be string, got %v", cacheType.IndexSignature.ValueType)
	}
}

func TestIndexSignatureWithUnionTypes(t *testing.T) {
	input := `
interface FlexibleMap
	[key: string]: string | number | boolean
end

local flexMap: FlexibleMap
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify the interface has an index signature with union type
	flexMapType, ok := checker.interfaces["FlexibleMap"]
	if !ok {
		t.Fatal("Expected FlexibleMap interface to be registered")
	}

	if flexMapType.IndexSignature == nil {
		t.Fatal("Expected FlexibleMap to have an index signature")
	}

	// Check key type is string
	if flexMapType.IndexSignature.KeyType != String {
		t.Errorf("Expected key type to be string, got %v", flexMapType.IndexSignature.KeyType)
	}

	// Check value type is a union
	unionType, ok := flexMapType.IndexSignature.ValueType.(*UnionType)
	if !ok {
		t.Fatalf("Expected value type to be a union, got %T", flexMapType.IndexSignature.ValueType)
	}

	if len(unionType.Types) != 3 {
		t.Errorf("Expected union to have 3 types, got %d", len(unionType.Types))
	}
}

func TestMultipleInterfacesWithIndexSignatures(t *testing.T) {
	input := `
interface StringDict
	[key: string]: string
end

interface NumberDict
	[key: string]: number
end

local strDict: StringDict
local numDict: NumberDict
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
		t.Errorf("Expected no errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}

	// Verify StringDict
	strDictType, ok := checker.interfaces["StringDict"]
	if !ok {
		t.Fatal("Expected StringDict interface to be registered")
	}
	if strDictType.IndexSignature == nil {
		t.Fatal("Expected StringDict to have an index signature")
	}
	if strDictType.IndexSignature.ValueType != String {
		t.Errorf("Expected StringDict value type to be string, got %v", strDictType.IndexSignature.ValueType)
	}

	// Verify NumberDict
	numDictType, ok := checker.interfaces["NumberDict"]
	if !ok {
		t.Fatal("Expected NumberDict interface to be registered")
	}
	if numDictType.IndexSignature == nil {
		t.Fatal("Expected NumberDict to have an index signature")
	}
	if numDictType.IndexSignature.ValueType != Number {
		t.Errorf("Expected NumberDict value type to be number, got %v", numDictType.IndexSignature.ValueType)
	}
}
