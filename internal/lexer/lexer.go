package lexer

import "fmt"

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()

	return l
}

func (l *Lexer) readChar() {
	fmt.Printf("Position: %d, Next char: %c\n", l.position, l.ch)
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII code for "NUL"
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition++

	if l.ch == '\n' {
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) NextToken() Token {
	fmt.Printf("NextToken() starting at pos %d, char %q\n", l.position, l.ch)
	var tok Token
	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '+':
		tok = newToken(PLUS, l.ch, l.line, l.column)
	case '-':
		if l.peekChar() == '-' {
			l.skipComment()
			return l.NextToken()
		}
		tok = newToken(MINUS, l.ch, l.line, l.column)
	case '~':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: NOT_EQ_LUA, Literal: "~=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: "!=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: EQ, Literal: "==", Line: l.line, Column: l.column}
		} else {
			tok = newToken(ASSIGN, l.ch, l.line, l.column)
		}
	case '*':
		tok = newToken(ASTERISK, l.ch, l.line, l.column)
	case '/':
		tok = newToken(SLASH, l.ch, l.line, l.column)
	case '%':
		tok = newToken(MODULO, l.ch, l.line, l.column)
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			tok = Token{Type: CONCAT, Literal: "..", Line: l.line, Column: l.column}
		} else {
			tok = newToken(DOT, l.ch, l.line, l.column)
		}
	case ',':
		tok = newToken(COMMA, l.ch, l.line, l.column)
	case ':':
		tok = newToken(COLON, l.ch, l.line, l.column)
	case '(':
		tok = newToken(LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(RPAREN, l.ch, l.line, l.column)
	case '[':
		tok = newToken(LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(RBRACKET, l.ch, l.line, l.column)
	case '{':
		tok = newToken(LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(RBRACE, l.ch, l.line, l.column)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		return tok
	case 0:
		tok.Type = EOF
		tok.Literal = ""
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	l.readChar() // skip first '-'
	l.readChar() // skip second '-'

	// Check for multiline comment
	if l.ch == '[' && l.peekChar() == '[' {
		l.skipMultiLineComment()
	} else {
		l.skipSingleLineComment()
	}
}

func (l *Lexer) skipMultiLineComment() {
	l.readChar() // consume first '['
	for {
		if l.ch == 0 {
			return
		}

		if l.ch == ']' && l.peekChar() == ']' {
			l.readChar() // consume first ']'
			l.readChar() // consume second ']'
			return
		}

		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) skipSingleLineComment() {
	// Skip until newline but don't consume it
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}

	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' && isDigit(l.peekChar()) {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	var result []byte

	for {
		l.readChar()

		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case '"':
				result = append(result, '"')
			case '\\':
				result = append(result, '\\')
			default:
				result = append(result, l.ch)
			}
			continue
		}

		if l.ch == '"' {
			l.readChar()
			break
		}

		if l.ch != 0 {
			result = append(result, l.ch)
		}
	}

	return string(result)
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}

	return l.input[l.readPosition]
}

func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Line:    line,
		Column:  column,
	}
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
