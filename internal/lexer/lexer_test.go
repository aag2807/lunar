package lexer

import (
	"testing"
)

func TestReadChar(t *testing.T) {
	input := "class\npoint"
	l := New(input)

	if l.ch != 'c' {
		t.Errorf("first char wrong. expected='c', got='%c'", l.ch)
	}

	if l.line != 1 {
		t.Errorf("line number wrong. expected=1 got=%d", l.line)
	}

	for i := 0; i < 5; i++ {
		l.readChar()
	}

	if l.line != 1 {
		t.Errorf("line number after newline wrong. expected=2 got=%d", l.line)
	}

	if l.column != 0 {
		t.Errorf("column after newline wrong. expected=0, got=%d", l.column)
	}
}

func TestNextToken(t *testing.T) {
	input := `class Point
	private x: number`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(CLASS), "class", 1},
		{TokenType(IDENT), "Point", 1},
		{TokenType(PRIVATE), "private", 2},
		{TokenType(IDENT), "x", 2},
		{TokenType(COLON), ":", 2},
		{TokenType(IDENT), "number", 2},
		{TokenType(EOF), "", 2},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype is wrong, expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal is wrong, expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Fatalf("tests[%d] - line is wrong, expected=%q, got=%q",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestNumberTokens(t *testing.T) {
	input := `42
	3.14
	100
	0.123`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(NUMBER), "42", 1},
		{TokenType(NUMBER), "3.14", 2},
		{TokenType(NUMBER), "100", 3},
		{TokenType(NUMBER), "0.123", 4},
		{TokenType(EOF), "", 4},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Fatalf("tests[%d] - line is wrong, expected=%q, got=%q",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestStringTokens(t *testing.T) {
	input := `"simple string"
    "string with \"quotes\""
    "string with \n newline"
    "string with \t tab"
    "multiple
    lines"
    "escaped \\backslash"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(STRING), "simple string", 1},
		{TokenType(STRING), "string with \"quotes\"", 2},
		{TokenType(STRING), "string with \n newline", 3},
		{TokenType(STRING), "string with \t tab", 4},
		{TokenType(STRING), "multiple\n    lines", 5},
		{TokenType(STRING), "escaped \\backslash", 6},
		{TokenType(EOF), "", 6},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Errorf("tests[%d] - line number wrong. expected=%d, got=%d",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestComments(t *testing.T) {
	input := `-- Single line comment
local x = 5 -- Inline comment
--[[ Multi
line
comment ]]
local y = 10`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(LOCAL), "local", 2},
		{TokenType(IDENT), "x", 2},
		{TokenType(ASSIGN), "=", 2},
		{TokenType(NUMBER), "5", 2},
		{TokenType(LOCAL), "local", 6},
		{TokenType(IDENT), "y", 6},
		{TokenType(ASSIGN), "=", 6},
		{TokenType(NUMBER), "10", 6},
		{TokenType(EOF), "", 6},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Errorf("tests[%d] - line number wrong. expected=%d, got=%d",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / %
== ~= != < > <= >=
and or not
.. "concat" .. "strings"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenType(PLUS), "+"},
		{TokenType(MINUS), "-"},
		{TokenType(ASTERISK), "*"},
		{TokenType(SLASH), "/"},
		{TokenType(MODULO), "%"},
		{TokenType(EQ), "=="},
		{TokenType(NOT_EQ_LUA), "~="},
		{TokenType(NOT_EQ), "!="},
		{TokenType(LT), "<"},
		{TokenType(GT), ">"},
		{TokenType(LT_EQ), "<="},
		{TokenType(GT_EQ), ">="},
		{TokenType(AND), "and"},
		{TokenType(OR), "or"},
		{TokenType(NOT), "not"},
		{TokenType(CONCAT), ".."},
		{TokenType(STRING), "concat"},
		{TokenType(CONCAT), ".."},
		{TokenType(STRING), "strings"},
		{TokenType(EOF), ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestDelimiters(t *testing.T) {
	input := `([]),:
local point: Point = {x: 10, y: 20}`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenType(LPAREN), "("},   //0
		{TokenType(LBRACKET), "["}, //1
		{TokenType(RBRACKET), "]"}, //2
		{TokenType(RPAREN), ")"},   //3
		{TokenType(COMMA), ","},    //4
		{TokenType(COLON), ":"},    //5

		{TokenType(LOCAL), "local"}, //6
		{TokenType(IDENT), "point"}, //7
		{TokenType(COLON), ":"},     //8
		{TokenType(IDENT), "Point"}, //9
		{TokenType(ASSIGN), "="},    //10

		{TokenType(LBRACE), "{"},  //11
		{TokenType(IDENT), "x"},   //12
		{TokenType(COLON), ":"},   //13
		{TokenType(NUMBER), "10"}, //14
		{TokenType(COMMA), ","},   //15

		{TokenType(IDENT), "y"},   //16
		{TokenType(COLON), ":"},   // 17
		{TokenType(NUMBER), "20"}, //18
		{TokenType(RBRACE), "}"},  // 19
		{TokenType(EOF), ""},      // 20
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestInterfaceDeclaration(t *testing.T) {
	input := `interface Vehicle
		brand: string 
		year: number
		start(): void
		stop(): void
	end
 
	interface ElectricVehicle extends Vehicle
		batteryLevel: number
		charge(duration: number): void
	end`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{INTERFACE, "interface"}, // 1
		{IDENT, "Vehicle"},       // 2
		{IDENT, "brand"},         // 3
		{COLON, ":"},             // 4
		{IDENT, "string"},        // 5
		{IDENT, "year"},          // 6
		{COLON, ":"},             // 7
		{IDENT, "number"},        // 8
		{IDENT, "start"},         // 9
		{LPAREN, "("},            // 10
		{RPAREN, ")"},            // 11
		{COLON, ":"},             // 12
		{VOID, "void"},           // 13
		{IDENT, "stop"},          // 14
		{LPAREN, "("},            // 15
		{RPAREN, ")"},            // 16
		{COLON, ":"},             // 17
		{VOID, "void"},           // 18
		{END, "end"},             // 19

		{INTERFACE, "interface"},   // 20
		{IDENT, "ElectricVehicle"}, // 21
		{EXTENDS, "extends"},       // 22
		{IDENT, "Vehicle"},         // 23
		{IDENT, "batteryLevel"},    // 24
		{COLON, ":"},               // 25
		{IDENT, "number"},          // 26
		{IDENT, "charge"},          // 27
		{LPAREN, "("},              // 28
		{IDENT, "duration"},        // 29
		{COLON, ":"},               // 30
		{IDENT, "number"},          // 31
		{RPAREN, ")"},              // 32
		{COLON, ":"},               // 33
		{VOID, "void"},             // 34
		{END, "end"},               // 35
		{EOF, ""},                  // 36
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}

func TestClassDeclaration(t *testing.T) {
	input := `class Car implements Vehicle
		private brand: string
		private year: number
		private running: boolean

		constructor(brand: string, year: number)
			self.brand = brand
			self.year = year
			self.running = false
		end

		public start(): void 
			self.running = true
		end
	end`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenType(CLASS), "class"},
		{TokenType(IDENT), "Car"},
		{TokenType(IMPLEMENTS), "implements"},
		{TokenType(IDENT), "Vehicle"},

		{TokenType(PRIVATE), "private"},
		{TokenType(IDENT), "brand"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "string"},

		{TokenType(PRIVATE), "private"},
		{TokenType(IDENT), "year"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "number"},

		{TokenType(PRIVATE), "private"},
		{TokenType(IDENT), "running"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "boolean"},

		{TokenType(CONSTRUCTOR), "constructor"},
		{TokenType(LPAREN), "("},
		{TokenType(IDENT), "brand"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "string"},
		{TokenType(COMMA), ","},
		{TokenType(IDENT), "year"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "number"},
		{TokenType(RPAREN), ")"},

		{TokenType(SELF), "self"},
		{TokenType(DOT), "."},
		{TokenType(IDENT), "brand"},
		{TokenType(ASSIGN), "="},
		{TokenType(IDENT), "brand"},

		{TokenType(SELF), "self"},
		{TokenType(DOT), "."},
		{TokenType(IDENT), "year"},
		{TokenType(ASSIGN), "="},
		{TokenType(IDENT), "year"},

		{TokenType(SELF), "self"},
		{TokenType(DOT), "."},
		{TokenType(IDENT), "running"},
		{TokenType(ASSIGN), "="},
		{TokenType(IDENT), "false"},

		{TokenType(END), "end"},

		{TokenType(PUBLIC), "public"},
		{TokenType(IDENT), "start"},
		{TokenType(LPAREN), "("},
		{TokenType(RPAREN), ")"},
		{TokenType(COLON), ":"},
		{TokenType(VOID), "void"},

		{TokenType(SELF), "self"},
		{TokenType(DOT), "."},
		{TokenType(IDENT), "running"},
		{TokenType(ASSIGN), "="},
		{TokenType(IDENT), "true"},
		{TokenType(END), "end"},

		{TokenType(END), "end"},
		{TokenType(EOF), ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}
