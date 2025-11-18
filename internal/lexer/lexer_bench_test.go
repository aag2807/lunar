package lexer

import (
	"strings"
	"testing"
)

// BenchmarkLexerSimple benchmarks lexing a simple program
func BenchmarkLexerSimple(b *testing.B) {
	input := `
local x = 42
local y = "hello"
function add(a, b)
	return a + b
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkLexerComplex benchmarks lexing a complex program with types
func BenchmarkLexerComplex(b *testing.B) {
	input := `
interface Person
	name: string
	age: number
	active: boolean
end

class Employee
	private salary: number
	public name: string

	function constructor(name: string, salary: number)
		self.name = name
		self.salary = salary
	end

	function getSalary(): number
		return self.salary
	end

	function promote(increase: number): void
		self.salary = self.salary + increase
	end
end

function processEmployee<T extends Person>(emp: T): string
	if emp.active then
		return emp.name
	else
		return "inactive"
	end
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkLexerLargeFile benchmarks lexing a large file
func BenchmarkLexerLargeFile(b *testing.B) {
	// Generate a large program with many functions
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString(`
function func` + string(rune('0'+i%10)) + `(x: number, y: number): number
	local result = x + y
	if result > 0 then
		return result
	else
		return 0
	end
end
`)
	}

	input := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkLexerTemplateStrings benchmarks lexing template strings
func BenchmarkLexerTemplateStrings(b *testing.B) {
	input := "local x = `Hello ${name}, you are ${age} years old`\nlocal y = `Result: ${x + y}`"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkLexerComments benchmarks lexing with comments
func BenchmarkLexerComments(b *testing.B) {
	input := `
-- This is a comment
local x = 42 -- inline comment

-- Multi-line
-- comments
-- here

function process(value)
	-- Complex logic here
	return value * 2
end
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}
