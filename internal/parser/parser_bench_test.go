package parser

import (
	"lunar/internal/lexer"
	"strings"
	"testing"
)

// BenchmarkParserSimple benchmarks parsing a simple program
func BenchmarkParserSimple(b *testing.B) {
	input := `
local x = 42
local y = "hello"
function add(a, b)
	return a + b
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}

// BenchmarkParserComplex benchmarks parsing a complex program
func BenchmarkParserComplex(b *testing.B) {
	input := `
interface Person
	name: string
	age: number
end

class Employee
	private salary: number

	function constructor(name: string)
		self.salary = 0
	end

	function getSalary(): number
		return self.salary
	end
end

type Nullable<T> = T | nil
type Result<T> = T | "error"

function process<T>(value: T): Result<T>
	if value == nil then
		return "error"
	else
		return value
	end
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}

// BenchmarkParserAdvancedTypes benchmarks parsing advanced type features
func BenchmarkParserAdvancedTypes(b *testing.B) {
	input := `
type Partial<T> = { [K in keyof T]?: T[K] }
type Readonly<T> = { readonly [K in keyof T]: T[K] }
type Record<K, V> = { [P in K]: V }

type IsString<T> = T extends string ? true : false
type ButtonType = "left" | "right"
type EventName = ` + "`${ButtonType}Click`" + `

interface Config
	host: string
	port: number
end

type PartialConfig = Partial<Config>
type ReadonlyConfig = Readonly<Config>
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}

// BenchmarkParserLargeFile benchmarks parsing a large file
func BenchmarkParserLargeFile(b *testing.B) {
	var builder strings.Builder
	for i := 0; i < 50; i++ {
		builder.WriteString(`
interface Data` + string(rune('0'+i%10)) + `
	value: number
	name: string
end

class Processor` + string(rune('0'+i%10)) + `
	function process(data: Data0): number
		return data.value * 2
	end
end
`)
	}

	input := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}

// BenchmarkParserNestedExpressions benchmarks parsing deeply nested expressions
func BenchmarkParserNestedExpressions(b *testing.B) {
	input := `
function calculate()
	local result = ((1 + 2) * (3 - 4)) / ((5 + 6) - (7 * 8))
	local complex = (a and b) or (c and (d or e))
	return result + complex
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}

// BenchmarkParserGenericTypes benchmarks parsing generic types
func BenchmarkParserGenericTypes(b *testing.B) {
	input := `
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
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		p.Parse()
	}
}
