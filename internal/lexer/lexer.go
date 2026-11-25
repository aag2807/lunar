package lexer

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
	file         string
	// unterminated names what reached the end of input unclosed, if
	// anything did; NextToken turns it into an ILLEGAL token.
	unterminated string
}

// NewFile lexes input, recording path as every token's File.
func NewFile(path, input string) *Lexer {
	l := New(input)
	l.file = path
	return l
}

// Source is the text being lexed.
func (l *Lexer) Source() string { return l.input }

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	// As lua does: a UTF-8 byte-order mark, then a first line starting
	// with '#' (#!/usr/bin/env lua), are not part of the chunk.
	if strings.HasPrefix(input, "\xef\xbb\xbf") {
		l.readChar()
		l.readChar()
		l.readChar()
		l.column = 1
	}
	if l.ch == '#' {
		for !l.eof() && l.ch != '\n' {
			l.readChar()
		}
	}

	return l
}

// eof says whether the input is used up. A NUL byte is a character like
// any other: inside a string it is part of the string, elsewhere an error.
func (l *Lexer) eof() bool {
	return l.position >= len(l.input)
}

// fail records a lexical error; the token being read becomes ILLEGAL
// with this message. The first error in a token wins.
func (l *Lexer) fail(msg string) {
	if l.unterminated == "" {
		l.unterminated = msg
	}
}

// readChar advances one byte. Moving past a newline starts the next line,
// so line and column stay right inside strings and comments too.
func (l *Lexer) readChar() {
	// A line ends at \n, or at a \r not followed by one (CR-only files).
	if l.ch == '\n' || (l.ch == '\r' && (l.readPosition >= len(l.input) || l.input[l.readPosition] != '\n')) {
		l.line++
		l.column = 0
	}

	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII code for "NUL"
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition++
	l.column++
}

// NextToken returns the next token, with its start and end positions and
// byte offsets recorded in one place, after whitespace and comments.
func (l *Lexer) NextToken() Token {
	for {
		l.skipWhitespace()
		if l.ch == '-' && l.peekChar() == '-' {
			l.skipComment()
			continue
		}
		break
	}

	start, line, column := l.position, l.line, l.column
	tok := l.scanToken()
	if l.unterminated != "" {
		tok.Type, tok.Literal = ILLEGAL, l.unterminated
		l.unterminated = ""
	}

	end := l.position
	if tok.Type == EOF || start >= len(l.input) {
		start, end = len(l.input), len(l.input)
	}
	if end > len(l.input) {
		end = len(l.input)
	}
	tok.File = l.file
	tok.Offset, tok.End = start, end
	tok.Line, tok.Column = line, column

	text := l.input[start:end]
	if nl := strings.LastIndexByte(text, '\n'); nl < 0 {
		tok.EndLine, tok.EndColumn = line, column+len(text)
	} else {
		tok.EndLine = line + strings.Count(text, "\n")
		tok.EndColumn = len(text) - nl
	}
	return tok
}

// scanToken reads one token starting at the current character.
func (l *Lexer) scanToken() Token {
	var tok Token

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
		if isDigit(l.peekChar()) {
			// A numeral with a leading point, such as .5
			line, column := l.line, l.column
			return Token{Type: NUMBER, Literal: l.readNumber(), Line: line, Column: column}
		}
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
	case ';':
		tok = newToken(SEMICOLON, l.ch, l.line, l.column)
	case ':':
		tok = newToken(COLON, l.ch, l.line, l.column)
	case '(':
		tok = newToken(LPAREN, l.ch, l.line, l.column)
	case ')':
		tok = newToken(RPAREN, l.ch, l.line, l.column)
	case '[':
		// Check for long string [[...]]
		if level := l.longBracketLevel(); level >= 0 {
			tok.Type = STRING
			tok.Literal = l.readLongBracket(level, "unterminated long string")
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
		if !l.eof() {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
			break
		}
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
		} else if l.ch >= 0x80 {
			// One character, however many bytes it takes: é is one error.
			_, size := utf8.DecodeRuneInString(l.input[l.position:])
			tok = Token{Type: ILLEGAL, Literal: l.input[l.position : l.position+size], Line: l.line, Column: l.column}
			for i := 1; i < size; i++ {
				l.readChar()
			}
		} else {
			tok = newToken(ILLEGAL, l.ch, l.line, l.column)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' || l.ch == '\f' || l.ch == '\v' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	l.readChar() // skip first '-'
	l.readChar() // skip second '-'

	// --[[ ]], --[==[ ]==]: a long comment of that level. Anything else,
	// ---[[ included, is a line comment.
	if l.ch == '[' {
		if level := l.longBracketLevel(); level >= 0 {
			l.readLongBracket(level, "unterminated long comment")
			return
		}
	}
	l.skipSingleLineComment()
}

func (l *Lexer) skipSingleLineComment() {
	// Skip until newline but don't consume it
	for l.ch != '\n' && l.ch != '\r' && !l.eof() {
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

// readNumber reads a Lua numeral: decimal with optional fraction and
// exponent (3, 3.14, .5, 1e-9), or hexadecimal with optional fraction and
// binary exponent (0xFF, 0xA.8p1). The caller has seen a digit, or a '.'
// followed by a digit.
func (l *Lexer) readNumber() string {
	position := l.position

	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		l.readChar() // 0
		l.readChar() // x
		for isHexDigit(l.ch) {
			l.readChar()
		}
		if l.ch == '.' && l.peekChar() != '.' {
			l.readChar()
			for isHexDigit(l.ch) {
				l.readChar()
			}
		}
		l.readExponent('p', 'P')
		return l.malformedNumber(position)
	}

	for isDigit(l.ch) {
		l.readChar()
	}
	// 4. and 2.E-1 are numerals; 1..2 is 1, '..', 2.
	if l.ch == '.' && l.peekChar() != '.' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	l.readExponent('e', 'E')
	return l.malformedNumber(position)
}

// malformedNumber returns the numeral read from position, or, when
// letters or digits run on after it (5print, 0x1g, 3e), reads them too and
// reports it as Lua does.
func (l *Lexer) malformedNumber(position int) string {
	if !isLetter(l.ch) && !isDigit(l.ch) {
		return l.input[position:l.position]
	}
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	text := l.input[position:l.position]
	l.fail(fmt.Sprintf("malformed number near '%s'", text))
	return text
}

// readExponent consumes an exponent marker, optional sign and digits, but
// only when digits actually follow: "2e" stays the number 2 and the name e.
func (l *Lexer) readExponent(lower, upper byte) {
	if l.ch != lower && l.ch != upper {
		return
	}
	next := l.peekChar()
	digitAt := 1
	if next == '+' || next == '-' {
		digitAt = 2
	}
	if l.readPosition+digitAt-1 >= len(l.input) || !isDigit(l.input[l.readPosition+digitAt-1]) {
		return
	}
	for i := 0; i < digitAt; i++ {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
}

func (l *Lexer) readString(delimiter byte) string {
	var result []byte

	for {
		l.readChar()

		if l.eof() {
			l.fail("unterminated string")
			break
		}
		if l.ch == '\\' {
			l.readChar()
			result = l.escape(result, 0)
			continue
		}
		if l.ch == delimiter {
			l.readChar()
			break
		}
		if l.ch == '\n' || l.ch == '\r' {
			// As in Lua, a short string ends at its line.
			l.fail("unterminated string")
			break
		}
		result = append(result, l.ch)
	}

	return string(result)
}

// escape decodes the escape sequence whose first character (after the
// backslash) is current, as Lua does, appending its bytes to out. It
// leaves the last character of the sequence current. extra is one more
// character that escapes to itself (a template's backtick), or 0.
func (l *Lexer) escape(out []byte, extra byte) []byte {
	switch c := l.ch; {
	case c == 'a':
		return append(out, '\a')
	case c == 'b':
		return append(out, '\b')
	case c == 'f':
		return append(out, '\f')
	case c == 'n':
		return append(out, '\n')
	case c == 'r':
		return append(out, '\r')
	case c == 't':
		return append(out, '\t')
	case c == 'v':
		return append(out, '\v')
	case c == '\\', c == '"', c == '\'', extra != 0 && c == extra:
		return append(out, c)
	case c == '\n' || c == '\r':
		// A backslash before a line break keeps it, as one "\n" whatever
		// the line ending.
		if p := l.peekChar(); (p == '\n' || p == '\r') && p != c {
			l.readChar()
		}
		return append(out, '\n')
	case c == 'z':
		// \z skips the whitespace that follows, line breaks included.
		for {
			switch l.peekChar() {
			case ' ', '\t', '\n', '\r', '\f', '\v':
				l.readChar()
				continue
			}
			return out
		}
	case c == 'x':
		v := 0
		for i := 0; i < 2; i++ {
			if !isHexDigit(l.peekChar()) {
				l.fail("hexadecimal digit expected")
				return out
			}
			l.readChar()
			v = v*16 + hexValue(l.ch)
		}
		return append(out, byte(v))
	case isDigit(c):
		v := int(c - '0')
		for i := 0; i < 2 && isDigit(l.peekChar()); i++ {
			l.readChar()
			v = v*10 + int(l.ch-'0')
		}
		if v > 255 {
			l.fail("decimal escape too large")
			return out
		}
		return append(out, byte(v))
	case c == 'u':
		if l.peekChar() != '{' {
			// \uHHHH, four digits: kept from earlier Lunar (Lua has only
			// \u{...}).
			v := 0
			for i := 0; i < 4; i++ {
				if !isHexDigit(l.peekChar()) {
					l.fail("missing '{' in \\u{xxxx}")
					return out
				}
				l.readChar()
				v = v*16 + hexValue(l.ch)
			}
			return appendUTF8(out, v)
		}
		l.readChar() // '{'
		if !isHexDigit(l.peekChar()) {
			l.fail("hexadecimal digit expected")
			return out
		}
		v := 0
		for isHexDigit(l.peekChar()) {
			l.readChar()
			v = v*16 + hexValue(l.ch)
			if v > 0x7FFFFFFF {
				l.fail("UTF-8 value too large")
				return out
			}
		}
		if l.peekChar() != '}' {
			l.fail("missing '}' in \\u{xxxx}")
			return out
		}
		l.readChar() // '}'
		return appendUTF8(out, v)
	case l.eof():
		return out
	default:
		l.fail(fmt.Sprintf("invalid escape sequence '\\%c'", c))
		return append(out, c)
	}
}

// appendUTF8 encodes v as Lua does: UTF-8 extended to 31 bits, surrogates
// included (Go would write U+FFFD for both).
func appendUTF8(out []byte, v int) []byte {
	if v < 0x80 {
		return append(out, byte(v))
	}
	var buf [6]byte
	n := 0
	limit := 0x3f // largest value that fits in the first byte
	for v > limit {
		buf[5-n] = byte(0x80 | (v & 0x3f))
		n++
		v >>= 6
		limit >>= 1
	}
	buf[5-n] = byte((^limit << 1) | v)
	n++
	return append(out, buf[6-n:]...)
}

func hexValue(ch byte) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'f':
		return int(ch-'a') + 10
	default:
		return int(ch-'A') + 10
	}
}

func (l *Lexer) readTemplateString() string {
	var result []byte

	for {
		l.readChar()

		if l.eof() {
			l.fail("unterminated template string")
			break
		}
		if l.ch == '\\' {
			l.readChar()
			// \$ keeps a literal ${ out of interpolation.
			if l.ch == '$' {
				result = append(result, '$')
				continue
			}
			result = l.escape(result, '`')
			continue
		}
		if l.ch == '`' {
			l.readChar()
			break
		}
		// Preserve ${} expressions as-is (parser will handle them)
		result = append(result, l.ch)
	}

	return string(result)
}

// longBracketLevel says whether a long bracket opens at the current '[':
// [[ is level 0, [==[ level 2. It is -1 when none does; nothing is read.
func (l *Lexer) longBracketLevel() int {
	i := l.position + 1
	for i < len(l.input) && l.input[i] == '=' {
		i++
	}
	if i < len(l.input) && l.input[i] == '[' {
		return i - l.position - 1
	}
	return -1
}

// readLongBracket reads a long string or comment of the given level,
// starting on its first '['. As in Lua, a line break right after the
// opening bracket is dropped and every line ending becomes "\n".
func (l *Lexer) readLongBracket(level int, unterminated string) string {
	for i := 0; i < level+2; i++ {
		l.readChar() // [ = ... = [
	}
	var result []byte
	first := true
	for {
		if l.eof() {
			l.fail(unterminated)
			return string(result)
		}
		if l.ch == ']' && l.closesLongBracket(level) {
			for i := 0; i < level+2; i++ {
				l.readChar()
			}
			return string(result)
		}
		if l.ch == '\n' || l.ch == '\r' {
			if p := l.peekChar(); (p == '\n' || p == '\r') && p != l.ch {
				l.readChar()
			}
			if !first {
				result = append(result, '\n')
			}
		} else {
			result = append(result, l.ch)
		}
		first = false
		l.readChar()
	}
}

// closesLongBracket says whether ]=...=] of the level starts here.
func (l *Lexer) closesLongBracket(level int) bool {
	i := l.position + 1
	for n := 0; n < level; n++ {
		if i >= len(l.input) || l.input[i] != '=' {
			return false
		}
		i++
	}
	return i < len(l.input) && l.input[i] == ']'
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
		Literal: string([]byte{ch}), // the byte, not the rune it numbers
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

// isHexDigit checks if a character is a hexadecimal digit
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
