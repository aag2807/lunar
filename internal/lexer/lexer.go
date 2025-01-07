package lexer

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
	var tok Token

	//skip whitespace
	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '"':
		tok.Type = TokenType(STRING)
		tok.Literal = l.readString()
		l.readChar() //closing quote
		return tok
	case '=':
		tok = newToken(TokenType(ASSIGN), l.ch, l.line, l.column)
		break
	case ':':
		tok = newToken(TokenType(COLON), l.ch, l.line, l.column)
		break
	case '(':
		tok = newToken(TokenType(LPAREN), l.ch, l.line, l.column)
		break
	case ')':
		tok = newToken(TokenType(RPAREN), l.ch, l.line, l.column)
		break
	case ',':
		tok = newToken(TokenType(COMMA), l.ch, l.line, l.column)
		break
	case 0:
		tok.Literal = ""
		tok.Type = TokenType(EOF)
		break
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok

		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			tok.Type = TokenType(NUMBER)
		} else {
			tok = newToken(TokenType(ILLEGAL), l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for {
		switch l.ch {
		case ' ', '\t', '\r':
			l.readChar()
		case '\n':
			l.line++
			l.readChar()
		case '-':
			if l.peekChar() == '-' {
				l.skipComment()
				if l.ch == '\n' {
					l.line++
					l.readChar()
				}
			} else {
				return
			}
		default:
			return
		}
	}
}

func (l *Lexer) skipComment() {
	if l.ch == '-' && l.peekChar() == '-' {
		l.readChar() // move past first '-'
		l.readChar() // move past '--'

		if l.ch == '[' && l.peekChar() == '[' {
			l.readChar() // consume second '['
			l.skipMultiLineComment()
		} else {
			l.skipSingleLineComment()
		}
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
	// Read until end of line or EOF
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	// Now we're at the newline or EOF
	// Don't consume the newline - let skipWhitespace handle it
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
	position := l.position + 1 // skips the opening quote
	escapeNext := false
	var result []byte

	for {
		l.readChar()
		if l.ch == 0 {
			return string(result)
		}

		if escapeNext {
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '"':
				result = append(result, '"')
			case '\\':
				result = append(result, '\\')
			default:
				// Invalid escape sequence, just add the character
				result = append(result, l.ch)
			}
			escapeNext = false
			continue
		}

		if l.ch == '\\' {
			escapeNext = true
			continue
		}

		if l.ch == '"' {
			break
		}

		result = append(result, l.ch)
	}

	if l.ch == 0 {
		//unterminated string
		return l.input[position:l.position]
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
