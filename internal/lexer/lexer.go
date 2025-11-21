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
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: THIN_ARROW, Literal: "->", Line: l.line, Column: l.column}
		} else {
			tok = newToken(MINUS, l.ch, l.line, l.column)
		}
	case '~':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: NOT_EQ_LUA, Literal: "~=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(TILDE, l.ch, l.line, l.column)
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: NOT_EQ, Literal: "!=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(BANG, l.ch, l.line, l.column)
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: EQ, Literal: "==", Line: l.line, Column: l.column}
		} else if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: ARROW, Literal: "=>"}
		} else {
			tok = newToken(ASSIGN, l.ch, l.line, l.column)
		}
	case '?':
		if l.peekChar() == '.' {
			l.readChar()
			tok = Token{Type: OPTIONAL_CHAIN, Literal: "?.", Line: l.line, Column: l.column}
		} else if l.peekChar() == '?' {
			l.readChar()
			tok = Token{Type: NULLISH_COALESCE, Literal: "??", Line: l.line, Column: l.column}
		} else {
			tok = newToken(QUESTION, l.ch, l.line, l.column)
		}
	case '|':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: PIPE_OP, Literal: "|>", Line: l.line, Column: l.column}
		} else {
			tok = newToken(PIPE, l.ch, l.line, l.column)
		}
	case '<':
		if l.peekChar() == '<' {
			l.readChar()
			tok = Token{Type: LEFT_SHIFT, Literal: "<<", Line: l.line, Column: l.column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: LT_EQ, Literal: "<=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(LT, l.ch, l.line, l.column)
		}
	case '>':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: RIGHT_SHIFT, Literal: ">>", Line: l.line, Column: l.column}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: GT_EQ, Literal: ">=", Line: l.line, Column: l.column}
		} else {
			tok = newToken(GT, l.ch, l.line, l.column)
		}
	case '*':
		tok = newToken(ASTERISK, l.ch, l.line, l.column)
	case '/':
		if l.peekChar() == '/' {
			l.readChar()
			tok = Token{Type: FLOOR_DIV, Literal: "//", Line: l.line, Column: l.column}
		} else {
			tok = newToken(SLASH, l.ch, l.line, l.column)
		}
	case '%':
		tok = newToken(MODULO, l.ch, l.line, l.column)
	case '#':
		tok = newToken(HASH, l.ch, l.line, l.column)
	case '&':
		tok = newToken(AMPERSAND, l.ch, l.line, l.column)
	case '^':
		tok = newToken(CARET, l.ch, l.line, l.column)
	case '@':
		tok = newToken(AT, l.ch, l.line, l.column)
	case '.':
		if l.peekChar() == '.' {
			// Check if it's ... or ..
			l.readChar() // consume second dot
			if l.peekChar() == '.' {
				l.readChar() // consume third dot
				tok = Token{Type: ELLIPSIS, Literal: "...", Line: l.line, Column: l.column}
			} else {
				tok = Token{Type: CONCAT, Literal: "..", Line: l.line, Column: l.column}
			}
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
		// Check for long string [[...]]
		if l.peekChar() == '[' || l.peekChar() == '=' {
			tok.Type = STRING
			tok.Literal = l.readLongString()
			return tok
		}
		tok = newToken(LBRACKET, l.ch, l.line, l.column)
	case ']':
		tok = newToken(RBRACKET, l.ch, l.line, l.column)
	case '{':
		tok = newToken(LBRACE, l.ch, l.line, l.column)
	case '}':
		tok = newToken(RBRACE, l.ch, l.line, l.column)
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString('"')
		return tok
	case '\'':
		tok.Type = STRING
		tok.Literal = l.readString('\'')
		return tok
	case '`':
		tok.Type = TEMPLATE_STRING
		tok.Literal = l.readTemplateString()
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

func (l *Lexer) readString(delimiter byte) string {
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
			case 'r':
				result = append(result, '\r')
			case 'b':
				result = append(result, '\b')
			case 'f':
				result = append(result, '\f')
			case 'v':
				result = append(result, '\v')
			case '0':
				result = append(result, '\x00')
			case '"':
				result = append(result, '"')
			case '\'':
				result = append(result, '\'')
			case '\\':
				result = append(result, '\\')
			case 'x':
				// Hexadecimal escape: \xHH
				hex := l.readHexEscape(2)
				if hex != "" {
					result = append(result, byte(l.parseHex(hex)))
				}
			case 'u':
				// Unicode escape: \uHHHH or \u{HHHHHH}
				if l.peekChar() == '{' {
					l.readChar() // consume '{'
					hex := l.readUntil('}')
					if hex != "" && len(hex) <= 6 {
						codePoint := l.parseHex(hex)
						result = append(result, []byte(string(rune(codePoint)))...)
					}
				} else {
					hex := l.readHexEscape(4)
					if hex != "" {
						codePoint := l.parseHex(hex)
						result = append(result, []byte(string(rune(codePoint)))...)
					}
				}
			default:
				result = append(result, l.ch)
			}
			continue
		}

		if l.ch == delimiter {
			l.readChar()
			break
		}

		if l.ch != 0 {
			result = append(result, l.ch)
		}
	}

	return string(result)
}

func (l *Lexer) readTemplateString() string {
	var result []byte

	for {
		l.readChar()

		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case 'b':
				result = append(result, '\b')
			case 'f':
				result = append(result, '\f')
			case 'v':
				result = append(result, '\v')
			case '0':
				result = append(result, '\x00')
			case '`':
				result = append(result, '`')
			case '\\':
				result = append(result, '\\')
			case '$':
				// Allow escaping ${} in templates
				result = append(result, '$')
			case 'x':
				// Hexadecimal escape: \xHH
				hex := l.readHexEscape(2)
				if hex != "" {
					result = append(result, byte(l.parseHex(hex)))
				}
			case 'u':
				// Unicode escape: \uHHHH or \u{HHHHHH}
				if l.peekChar() == '{' {
					l.readChar() // consume '{'
					hex := l.readUntil('}')
					if hex != "" && len(hex) <= 6 {
						codePoint := l.parseHex(hex)
						result = append(result, []byte(string(rune(codePoint)))...)
					}
				} else {
					hex := l.readHexEscape(4)
					if hex != "" {
						codePoint := l.parseHex(hex)
						result = append(result, []byte(string(rune(codePoint)))...)
					}
				}
			default:
				result = append(result, l.ch)
			}
			continue
		}

		// End of template string
		if l.ch == '`' {
			l.readChar()
			break
		}

		// Preserve ${} expressions as-is (parser will handle them)
		if l.ch != 0 {
			result = append(result, l.ch)
		}
	}

	return string(result)
}

// readLongString reads Lua-style long strings: [[...]] or [=[...]=]
func (l *Lexer) readLongString() string {
	// Count opening '=' characters
	equalCount := 0
	l.readChar() // skip initial '['
	for l.ch == '=' {
		equalCount++
		l.readChar()
	}

	// Should now be at second '['
	if l.ch != '[' {
		// Invalid long string syntax, return empty
		return ""
	}

	var result []byte

	// Read content until we find the matching closing delimiter
	for {
		l.readChar()

		if l.ch == 0 {
			// End of input
			break
		}

		// Check for potential closing delimiter
		if l.ch == ']' {
			// Save position in case this isn't the closing delimiter
			savedPos := l.position
			savedReadPos := l.readPosition
			savedCh := l.ch
			savedLine := l.line
			savedColumn := l.column

			// Check if we have matching '=' characters
			matchCount := 0
			l.readChar()
			for l.ch == '=' && matchCount < equalCount {
				matchCount++
				l.readChar()
			}

			// If we found the right number of '=' and a closing ']', we're done
			if matchCount == equalCount && l.ch == ']' {
				l.readChar() // consume final ']'
				break
			}

			// Not the closing delimiter, restore position and add ']' to result
			l.position = savedPos
			l.readPosition = savedReadPos
			l.ch = savedCh
			l.line = savedLine
			l.column = savedColumn
		}

		result = append(result, l.ch)
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

// LexerState represents a saved state of the lexer for lookahead
type LexerState struct {
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

// SaveState saves the current lexer state for lookahead
func (l *Lexer) SaveState() LexerState {
	return LexerState{
		position:     l.position,
		readPosition: l.readPosition,
		ch:           l.ch,
		line:         l.line,
		column:       l.column,
	}
}

// RestoreState restores a previously saved lexer state
func (l *Lexer) RestoreState(state LexerState) {
	l.position = state.position
	l.readPosition = state.readPosition
	l.ch = state.ch
	l.line = state.line
	l.column = state.column
}

// readHexEscape reads n hexadecimal digits for escape sequences
func (l *Lexer) readHexEscape(n int) string {
	var result []byte
	for i := 0; i < n; i++ {
		l.readChar()
		if !isHexDigit(l.ch) {
			return ""
		}
		result = append(result, l.ch)
	}
	return string(result)
}

// readUntil reads until the specified delimiter
func (l *Lexer) readUntil(delimiter byte) string {
	var result []byte
	for {
		l.readChar()
		if l.ch == delimiter || l.ch == 0 {
			break
		}
		result = append(result, l.ch)
	}
	return string(result)
}

// parseHex converts a hexadecimal string to an integer
func (l *Lexer) parseHex(hex string) int {
	result := 0
	for _, ch := range hex {
		result *= 16
		if ch >= '0' && ch <= '9' {
			result += int(ch - '0')
		} else if ch >= 'a' && ch <= 'f' {
			result += int(ch - 'a' + 10)
		} else if ch >= 'A' && ch <= 'F' {
			result += int(ch - 'A' + 10)
		}
	}
	return result
}

// isHexDigit checks if a character is a hexadecimal digit
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
