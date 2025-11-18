package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"testing"
)

// BenchmarkTypeCheckerSimple benchmarks type checking a simple program
func BenchmarkTypeCheckerSimple(b *testing.B) {
	input := `
local x: number = 42
local y: string = "hello"

function add(a: number, b: number): number
	return a + b
end

local result = add(1, 2)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerClassesAndInterfaces benchmarks checking classes and interfaces
func BenchmarkTypeCheckerClassesAndInterfaces(b *testing.B) {
	input := `
interface Person
	name: string
	age: number
	active: boolean
end

interface Employee extends Person
	salary: number
	department: string
end

class Worker
	name: string
	age: number
	active: boolean
	salary: number
	department: string

	function constructor(n: string, a: number)
		self.name = n
		self.age = a
		self.active = true
		self.salary = 0
		self.department = "none"
	end

	function promote(amount: number): void
		self.salary = self.salary + amount
	end

	function getInfo(): string
		return self.name
	end
end

local worker = Worker("John", 30)
worker.promote(1000)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerGenerics benchmarks checking generic types
func BenchmarkTypeCheckerGenerics(b *testing.B) {
	input := `
type Nullable<T> = T | nil

function identity<T>(x: T): T
	return x
end

function map<T, U>(arr: T[], fn: (T) -> U): U[]
	local result: U[] = []
	return result
end

class Box<T>
	value: T

	function constructor(val: T)
		self.value = val
	end

	function get(): T
		return self.value
	end
end

local stringBox = Box<string>("hello")
local numberBox = Box<number>(42)
local result1: string = stringBox.get()
local result2: number = numberBox.get()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerAdvancedTypes benchmarks advanced type features
func BenchmarkTypeCheckerAdvancedTypes(b *testing.B) {
	input := `
type Partial<T> = { [K in keyof T]?: T[K] }
type Readonly<T> = { readonly [K in keyof T]: T[K] }

interface Config
	host: string
	port: number
	debug: boolean
end

type PartialConfig = Partial<Config>
type ReadonlyConfig = Readonly<Config>

type Direction = "left" | "right"
type ButtonType = "` + "`${Direction}Button`" + `"

type IsString<T> = T extends string ? true : false
type IsNumber<T> = T extends number ? true : false

type Result1 = IsString<string>
type Result2 = IsNumber<number>
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerLargeProgram benchmarks a large program with many types
func BenchmarkTypeCheckerLargeProgram(b *testing.B) {
	var builder strings.Builder

	// Generate many interfaces
	for i := 0; i < 20; i++ {
		builder.WriteString(`
interface Data` + string(rune('A'+i%26)) + `
	id: number
	name: string
	value: number
	active: boolean
end
`)
	}

	// Generate many classes
	for i := 0; i < 20; i++ {
		builder.WriteString(`
class Processor` + string(rune('A'+i%26)) + `
	data: Data` + string(rune('A'+i%26)) + `

	function process(): number
		return self.data.value * 2
	end

	function getName(): string
		return self.data.name
	end
end
`)
	}

	// Generate many functions
	for i := 0; i < 20; i++ {
		builder.WriteString(`
function transform` + string(rune('A'+i%26)) + `(x: number): number
	return x + ` + string(rune('0'+i%10)) + `
end
`)
	}

	input := builder.String()
	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerConditionalTypes benchmarks conditional type resolution
func BenchmarkTypeCheckerConditionalTypes(b *testing.B) {
	input := `
type IsString<T> = T extends string ? true : false
type IsNumber<T> = T extends number ? true : false
type IsBoolean<T> = T extends boolean ? true : false

type TypeName<T> =
	T extends string ? "string" :
	T extends number ? "number" :
	T extends boolean ? "boolean" :
	"other"

type A = TypeName<string>
type B = TypeName<number>
type C = TypeName<boolean>
type D = TypeName<any>

type Exclude<T, U> = T extends U ? never : T
type Extract<T, U> = T extends U ? T : never

type Result1 = Exclude<"a" | "b" | "c", "a">
type Result2 = Extract<"a" | "b" | "c", "a" | "b">
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerMappedTypes benchmarks mapped type resolution
func BenchmarkTypeCheckerMappedTypes(b *testing.B) {
	input := `
interface Person
	name: string
	age: number
	email: string
	active: boolean
end

type Partial<T> = { [K in keyof T]?: T[K] }
type Readonly<T> = { readonly [K in keyof T]: T[K] }
type Nullable<T> = { [K in keyof T]: T[K] | nil }

type PartialPerson = Partial<Person>
type ReadonlyPerson = Readonly<Person>
type NullablePerson = Nullable<Person>

type Record<K, V> = { [P in K]: V }
type StringRecord = Record<"foo" | "bar" | "baz", number>
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}

// BenchmarkTypeCheckerTemplateLiterals benchmarks template literal type resolution
func BenchmarkTypeCheckerTemplateLiterals(b *testing.B) {
	input := "type Direction = \"left\" | \"right\" | \"up\" | \"down\"\n" +
		"type Size = \"small\" | \"medium\" | \"large\"\n" +
		"type Color = \"red\" | \"green\" | \"blue\"\n\n" +
		"type ButtonType = `${Direction}Button`\n" +
		"type ClassName = `${Size}-${Color}-button`\n" +
		"type EventName = `on${Direction}Click`"

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewChecker()
		checker.Check(statements)
	}
}
