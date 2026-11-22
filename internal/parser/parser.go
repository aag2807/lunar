package parser

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"reflect"
	"strconv"
	"strings"
)

// Lua's operator precedence, lowest first (Lua 5.4 manual, 3.4.8). '..'
// and '^' are right associative.
const (
	_ int = iota
	LOWEST
	PIPE_PREC   // |> pipeline operator
	OR_PREC     // or, ??
	AND_PREC    // and
	COMPARE     // < > <= >= ~= ==
	BITWISE_OR  // |
	BITWISE_XOR // ~
	BITWISE_AND // &
	SHIFT       // << >>
	CONCAT      // ..
	SUM         // + -
	PRODUCT     // * / // %
	PREFIX      // not # - ~ (unary), as
	POWER       // ^
	DOT         // foo.bar
	CALL        // function(x)
)

var precedences = map[lexer.TokenType]int{
	lexer.AS:               PREFIX,    // type assertions have same precedence as prefix operators
	lexer.PIPE_OP:          PIPE_PREC, // |> pipeline operator
	lexer.NULLISH_COALESCE: OR_PREC,   // ?? has same precedence as or
	lexer.OR:               OR_PREC,
	lexer.AND:              AND_PREC,
	lexer.EQ:               COMPARE,
	lexer.NOT_EQ:           COMPARE,
	lexer.NOT_EQ_LUA:       COMPARE,
	lexer.LT:               COMPARE,
	lexer.GT:               COMPARE,
	lexer.LT_EQ:            COMPARE,
	lexer.GT_EQ:            COMPARE,
	lexer.PIPE:             BITWISE_OR,
	lexer.TILDE:            BITWISE_XOR,
	lexer.AMPERSAND:        BITWISE_AND,
	lexer.LEFT_SHIFT:       SHIFT,
	lexer.RIGHT_SHIFT:      SHIFT,
	lexer.CONCAT:           CONCAT,
	lexer.PLUS:             SUM,
	lexer.MINUS:            SUM,
	lexer.ASTERISK:         PRODUCT,
	lexer.SLASH:            PRODUCT,
	lexer.FLOOR_DIV:        PRODUCT,
	lexer.MODULO:           PRODUCT,
	lexer.CARET:            POWER,
	lexer.DOT:              DOT,
	lexer.OPTIONAL_CHAIN:   DOT,  // ?. has same precedence as .
	lexer.LBRACKET:         CALL, // index has same precedence as function call
	lexer.LPAREN:           CALL,
	lexer.LBRACE:           CALL, // f{...}, a call with one table
	lexer.STRING:           CALL, // f"...", a call with one string
}

// rightAssociative operators group from the right: a .. b .. c is
// a .. (b .. c), and 2 ^ 3 ^ 2 is 2 ^ 9.
var rightAssociative = map[lexer.TokenType]bool{lexer.CONCAT: true, lexer.CARET: true}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

type Parser struct {
	l *lexer.Lexer

	// pending is a token read ahead of peekToken, when >> is split into
	// two > tokens to close nested generics.
	pending *lexer.Token

	// classDepth counts the class bodies being parsed, where super names
	// the parent class.
	classDepth int

	// stopAtPipe is set while a match arm's body is parsed, where a bare '|'
	// begins the next arm; brackets and blocks clear it (clearStopAtPipe).
	stopAtPipe bool

	// bodilessDeclare is the last `declare function` written without an
	// `end`, so a stray `end` after it can be explained as its body.
	bodilessDeclare *ast.FunctionDeclaration

	curToken  lexer.Token
	peekToken lexer.Token

	errors []*ParseError

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn

	// ambient is set inside `declare class` and `declare namespace`, where
	// functions, methods, constructors and accessors are signatures with no
	// body: the enclosing declaration's 'end' is the only one.
	ambient bool
	// matchWith is set once a match expression reached its `with`.
	matchWith bool
}

// addError adds a parse error with location information
func (p *Parser) addError(message string, token lexer.Token) {
	endLine, endColumn := token.EndLine, token.EndColumn
	if endLine == 0 {
		endLine, endColumn = token.Line, token.Column
	}
	p.errors = append(p.errors, &ParseError{
		Message:   message,
		Line:      token.Line,
		Column:    token.Column,
		File:      token.File,
		EndLine:   endLine,
		EndColumn: endColumn,
		Code:      "syntax",
	})
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []*ParseError{},
	}

	//register prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.SELF, p.parseIdentifier) // self is like an identifier
	p.registerPrefix(lexer.SUPER, p.parseSuperExpression)
	// Context-aware keywords can be used as identifiers in value contexts
	p.registerPrefix(lexer.STRING_TYPE, p.parseIdentifier)
	p.registerPrefix(lexer.TABLE, p.parseIdentifier)
	p.registerPrefix(lexer.TYPE, p.parseIdentifier)
	p.registerPrefix(lexer.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TEMPLATE_STRING, p.parseTemplateLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.HASH, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.TILDE, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACE, p.parseTableLiteral)
	p.registerPrefix(lexer.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(lexer.ELLIPSIS, p.parseSpreadExpression)
	p.registerPrefix(lexer.AWAIT, p.parseAwaitOrName)
	p.registerPrefix(lexer.MATCH, p.parseMatchOrName)
	// Every other keyword of Lunar's own is a name in an expression.
	for _, t := range lexer.SoftKeywordTypes() {
		if _, taken := p.prefixParseFns[t]; !taken {
			p.registerPrefix(t, p.parseIdentifier)
		}
	}

	//register infix operators
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.FLOOR_DIV, p.parseInfixExpression)
	p.registerInfix(lexer.MODULO, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ_LUA, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseGenericOrLessThan)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.GT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LEFT_SHIFT, p.parseInfixExpression)
	p.registerInfix(lexer.RIGHT_SHIFT, p.parseInfixExpression)
	p.registerInfix(lexer.AMPERSAND, p.parseInfixExpression)
	p.registerInfix(lexer.PIPE, p.parseInfixExpression)
	p.registerInfix(lexer.PIPE_OP, p.parsePipeExpression)
	p.registerInfix(lexer.CARET, p.parseInfixExpression)
	p.registerInfix(lexer.TILDE, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.LBRACE, p.parseSingleArgumentCall)
	p.registerInfix(lexer.STRING, p.parseSingleArgumentCall)
	p.registerInfix(lexer.DOT, p.parseDotExpression)
	p.registerInfix(lexer.OPTIONAL_CHAIN, p.parseDotExpression)
	p.registerInfix(lexer.COLON, p.parseMethodCallExpression)
	p.registerInfix(lexer.QUESTION, p.parseOptionalIndexExpression)
	p.registerInfix(lexer.CONCAT, p.parseInfixExpression)
	p.registerInfix(lexer.NULLISH_COALESCE, p.parseInfixExpression)
	p.registerInfix(lexer.AS, p.parseTypeAssertion)

	// read to tokens to initialize curtoken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	if p.pending != nil {
		p.peekToken, p.pending = *p.pending, nil
		return
	}
	p.peekToken = p.l.NextToken()
}

// expectPeekGT expects a GT token or splits RIGHT_SHIFT for nested generics
func (p *Parser) expectPeekGT() bool {
	if p.peekTokenIs(lexer.GT) {
		p.nextToken()
		return true
	}

	// Handle >> as two > tokens for nested generics like Box<Box<T>>
	if p.peekTokenIs(lexer.RIGHT_SHIFT) {
		p.nextToken() // move to >>
		// Split >> into two > tokens, keeping the token after it for later
		after := p.peekToken
		p.pending = &after
		shift := p.curToken
		p.curToken = lexer.Token{Type: lexer.GT, Literal: ">", File: shift.File, Line: shift.Line, Column: shift.Column,
			Offset: shift.Offset, End: shift.Offset + 1, EndLine: shift.Line, EndColumn: shift.Column + 1}
		p.peekToken = lexer.Token{Type: lexer.GT, Literal: ">", File: shift.File, Line: shift.Line, Column: shift.Column + 1,
			Offset: shift.Offset + 1, End: shift.Offset + 2, EndLine: shift.Line, EndColumn: shift.Column + 2}
		return true
	}

	p.peekError(lexer.GT)
	return false
}

// startsDeclaration says whether the keyword in curToken begins one of
// Lunar's declarations here, judging by the token after it.
func (p *Parser) startsDeclaration() bool {
	name := p.peekTokenIsIdentOrContextual()
	switch p.curToken.Type {
	case lexer.CLASS, lexer.INTERFACE, lexer.ENUM, lexer.NAMESPACE, lexer.TYPE:
		return name
	case lexer.CONST:
		return name || p.peekTokenIs(lexer.LBRACE) || p.peekTokenIs(lexer.LPAREN)
	case lexer.ABSTRACT:
		return p.peekTokenIs(lexer.CLASS)
	case lexer.ASYNC:
		return p.peekTokenIs(lexer.FUNCTION)
	case lexer.IMPORT:
		return p.peekTokenIs(lexer.LBRACE) || p.peekTokenIs(lexer.ASTERISK)
	case lexer.EXPORT, lexer.DECLARE:
		switch p.peekToken.Type {
		case lexer.FUNCTION, lexer.LOCAL, lexer.CONST, lexer.CLASS, lexer.INTERFACE, lexer.ENUM,
			lexer.TYPE, lexer.NAMESPACE, lexer.ABSTRACT, lexer.DECLARE, lexer.ASYNC:
			return true
		}
		return false
	}
	return false
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseSuperExpression() ast.Expression {
	// Outside a class, super is an ordinary name.
	if p.classDepth == 0 {
		return p.parseIdentifier()
	}
	return &ast.SuperExpression{
		Token: p.curToken,
	}
}

// parseMatchOrName parses `match x with ...`, or the name match when what
// follows is not a match expression (match = 1, s:match(...), match(x)).
func (p *Parser) parseMatchOrName() ast.Expression {
	state := p.SaveState()
	tok := p.curToken
	errorsBefore := len(p.errors)
	prevWith := p.matchWith
	p.matchWith = false
	m := p.parseMatchExpression()
	sawWith := p.matchWith
	p.matchWith = prevWith
	if !isNilNode(m) && len(p.errors) == errorsBefore {
		return m
	}
	// `match v with` is a match: its own errors say what is wrong, where
	// reading match as a name gave "'v' is not a statement" and more.
	if sawWith {
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			p.nextToken()
		}
		if isNilNode(m) {
			return &ast.MatchExpression{Token: tok}
		}
		return m
	}
	p.RestoreState(state)
	return p.parseIdentifier()
}

// parseAwaitOrName parses `await x`, which is refused later, or the name
// await anywhere else.
func (p *Parser) parseAwaitOrName() ast.Expression {
	// `local u = await` then a new line is the name.
	if p.peekTokenIsIdentOrContextual() && p.peekToken.Line == p.curToken.EndLine {
		return p.parseAwaitExpression()
	}
	return p.parseIdentifier()
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	value, err := parseLuaNumber(p.curToken.Literal)
	// 1e309 is out of range for a float: Lua reads it as inf, as Go does.
	if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
		err = nil
	}
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as number", p.curToken.Literal)
		p.addError(msg, p.curToken)
		return nil
	}

	return &ast.NumberLiteral{Token: p.curToken, Value: value}
}

// parseLuaNumber converts a Lua numeral to its value. Go's ParseFloat reads
// hexadecimal only with a binary exponent, so one is supplied when absent.
func parseLuaNumber(literal string) (float64, error) {
	lower := strings.ToLower(literal)
	if strings.HasPrefix(lower, "0x") && !strings.Contains(lower, "p") {
		literal += "p0"
	}
	return strconv.ParseFloat(literal, 64)
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseTemplateLiteral() ast.Expression {
	token := p.curToken
	template := &ast.TemplateLiteral{
		Token:       token,
		Parts:       []string{},
		Expressions: []ast.Expression{},
	}

	// Parse the template string and extract parts and expressions
	raw := token.Literal
	var currentPart strings.Builder
	i := 0

	for i < len(raw) {
		// Look for ${ pattern
		if i < len(raw)-1 && raw[i] == '$' && raw[i+1] == '{' {
			// Add the current part
			template.Parts = append(template.Parts, currentPart.String())
			currentPart.Reset()

			// Find the matching }
			i += 2 // skip ${
			braceCount := 1
			exprStart := i

			for i < len(raw) && braceCount > 0 {
				if raw[i] == '{' {
					braceCount++
				} else if raw[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			// Parse the expression, then place its tokens where it is in
			// the file (it was lexed on its own, from line 1 of no file),
			// so errors, hover and rename point inside the template.
			exprStr := raw[exprStart:i]
			exprLexer := lexer.NewFile(token.File, exprStr)
			exprParser := New(exprLexer)
			expr := exprParser.parseExpression(LOWEST)
			line, col := token.Line, token.Column+1 // just past the backtick
			for _, ch := range []byte(raw[:exprStart]) {
				if ch == '\n' {
					line, col = line+1, 1
				} else {
					col++
				}
			}
			placeTokens(reflect.ValueOf(expr), token.Offset+1+exprStart, line, col)

			if len(exprParser.Errors()) > 0 {
				for _, err := range exprParser.Errors() {
					p.addError("Template expression error: "+err.Message, token)
				}
				return nil
			}

			template.Expressions = append(template.Expressions, expr)
			i++ // skip the closing }
		} else {
			// Bytes, not characters: string(raw[i]) turned each byte of a
			// UTF-8 character into a character of its own.
			currentPart.WriteByte(raw[i])
			i++
		}
	}

	// Add the final part
	template.Parts = append(template.Parts, currentPart.String())

	return template
}

// placeTokens moves every token in a node parsed from part of a line (a
// template's ${...}) to where that part is: offset, line and column of its
// first character.
func placeTokens(v reflect.Value, offset, line, col int) {
	placeTokensOnce(v, offset, line, col, map[uintptr]bool{})
}

// placeTokensOnce visits each node once: a table's fields are reached both
// through its pairs and its entries.
func placeTokensOnce(v reflect.Value, offset, line, col int, seen map[uintptr]bool) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			placeTokensOnce(v.Elem(), offset, line, col, seen)
		}
	case reflect.Ptr:
		if v.IsNil() || seen[v.Pointer()] {
			return
		}
		seen[v.Pointer()] = true
		placeTokensOnce(v.Elem(), offset, line, col, seen)
	case reflect.Struct:
		if tok, ok := v.Addr().Interface().(*lexer.Token); ok {
			if tok.Line == 1 {
				tok.Column += col - 1
			}
			if tok.EndLine == 1 {
				tok.EndColumn += col - 1
			}
			tok.Line += line - 1
			tok.EndLine += line - 1
			tok.Offset += offset
			tok.End += offset
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				placeTokensOnce(v.Field(i), offset, line, col, seen)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			placeTokensOnce(v.Index(i), offset, line, col, seen)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			placeTokensOnce(k, offset, line, col, seen)
			placeTokensOnce(v.MapIndex(k), offset, line, col, seen)
		}
	}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{
		Token: p.curToken,
		Value: p.curToken.Type == lexer.TRUE,
	}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{Token: p.curToken}
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()
	if isNilNode(leftExp) {
		return nil
	}

	for precedence < p.peekPrecedence() {
		// Inside a match arm, a bare '|' starts the next arm.
		if p.stopAtPipe && p.peekTokenIs(lexer.PIPE) {
			return leftExp
		}
		// A call (f(), f{...}, f"...") applies only to what Lua can call: a
		// name, a field, an index, a call or a parenthesised expression.
		// After anything else, as `local t = {}` then `(f)()` on the next
		// line, the '(' starts a new statement, as in Lua.
		if (p.peekTokenIs(lexer.LBRACE) || p.peekTokenIs(lexer.STRING) || p.peekTokenIs(lexer.LPAREN)) && !isCallable(leftExp) {
			return leftExp
		}
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
		if isNilNode(leftExp) {
			return nil
		}
	}

	return leftExp
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	defer p.clearStopAtPipe()()
	// We're at '('. Need to disambiguate between:
	// - Grouped expression: (a + b)
	// - Arrow function: (x: number) => x * 2

	// Save state for lookahead
	savedState := p.SaveState()

	// Quick checks for obvious arrow functions: () => and (...args) =>.
	// (...) on its own is Lua's first vararg value.
	if p.peekTokenIs(lexer.RPAREN) || p.peekTokenIs(lexer.ELLIPSIS) && p.looksLikeArrowFunction() {
		return p.parseArrowFunction()
	}

	// Look ahead to detect arrow function patterns
	// Pattern: (ident : type) or (ident , ...) or (ident) =>
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken() // move from '(' to ident

		// Now curToken = ident, check what follows. (x: number) => and
		// (a, b) => are arrow functions, but so does (s:len() + 1) begin:
		// try the arrow function, and parse a grouped expression instead
		// if that fails.
		if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.COMMA) {
			p.RestoreState(savedState)
			errorsBefore := len(p.errors)
			if arrow := p.parseArrowFunction(); !isNilNode(arrow) && len(p.errors) == errorsBefore {
				return arrow
			}
			p.RestoreState(savedState)
			return p.parseGroupedInner()
		}

		if p.peekTokenIs(lexer.RPAREN) {
			// (ident) - check if followed by =>
			p.nextToken() // move to ')'
			if p.peekTokenIs(lexer.ARROW) {
				// (ident) => - arrow function
				p.RestoreState(savedState)
				return p.parseArrowFunction()
			}
		}
	}

	// Restore state and parse as grouped expression
	p.RestoreState(savedState)
	return p.parseGroupedInner()
}

// parseGroupedInner parses ( expression ), starting on the '('.
func (p *Parser) parseGroupedInner() ast.Expression {
	open := p.curToken
	p.nextToken() // consume '('

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}
	// (f()) keeps only f's first value, so these parentheses stay.
	if call, ok := exp.(*ast.CallExpression); ok {
		call.Parenthesized = true
	}
	// (...) is the first vararg value only: the parentheses stay.
	if v, ok := exp.(*ast.VarargExpression); ok {
		return &ast.ParenExpression{Token: open, Expression: v}
	}
	// Called, indexed or reached into, the parentheses decide what that
	// applies to: (t.f or g)(2), (a or b)[1], (function() end)().
	// (t.b) = 1 is not an assignment in Lua: the parentheses stay, for the
	// assignment to refuse.
	if p.peekTokenIs(lexer.ASSIGN) && !isNilNode(exp) {
		return &ast.ParenExpression{Token: open, Expression: exp}
	}
	if !isCallable(exp) || isNilNode(exp) {
		switch p.peekToken.Type {
		case lexer.LPAREN, lexer.LBRACKET, lexer.DOT, lexer.COLON, lexer.STRING, lexer.LBRACE:
			if !isNilNode(exp) {
				return &ast.ParenExpression{Token: open, Expression: exp}
			}
		}
	}

	return exp
}

// looksLikeArrowFunction does a quick lookahead to see if this looks like an arrow function
func (p *Parser) looksLikeArrowFunction() bool {
	// We're currently at '(', check only the immediate next token (peekToken)
	// This is a simple, safe lookahead without state manipulation

	// Definite arrow function indicators:
	// - () => (empty params)
	// - (...args) => (rest parameter)

	if p.peekTokenIs(lexer.RPAREN) {
		// () pattern - empty params almost always means arrow function
		return true
	}

	if p.peekTokenIs(lexer.ELLIPSIS) {
		// (...args) => is an arrow function; (...) alone is Lua's first
		// vararg value, unless => (or a return type) follows.
		state := p.SaveState()
		defer p.RestoreState(state)
		p.nextToken() // '...'
		if !p.peekTokenIs(lexer.RPAREN) {
			return true
		}
		p.nextToken() // ')'
		return p.peekTokenIs(lexer.ARROW) || (p.peekTokenIs(lexer.COLON) && !p.isColonMethodCall())
	}

	// For other cases like (x: type) or (x, y) or (x) =>
	// we can't easily detect without deeper lookahead
	// Return false to default to grouped expression
	// Users will need to use one of the above patterns or we'll parse as grouped expr
	return false
}

func (p *Parser) parseArrowFunction() ast.Expression {
	// Save current position
	startToken := p.curToken

	// Debug: log what tokens we're starting with
	// fmt.Printf("parseArrowFunction: curToken=%s, peekToken=%s\n", p.curToken.Literal, p.peekToken.Literal)

	// We're at '(', so parse parameters
	p.nextToken() // consume '('

	// Debug: log after consuming (
	// fmt.Printf("after consuming (: curToken=%s, peekToken=%s\n", p.curToken.Literal, p.peekToken.Literal)

	// Parse parameters - could be empty ()
	params := []*ast.Parameter{}

	if !p.curTokenIs(lexer.RPAREN) {
		// Parse first parameter
		param := p.parseParameter()
		if param == nil {
			return nil
		}
		params = append(params, param)

		// Parse remaining parameters
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next param
			param = p.parseParameter()
			if param == nil {
				return nil
			}
			params = append(params, param)
		}

		// Expect ')' after parameters
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}
	// If we started at ')', we're already there, no need to expect it again

	// Parse optional return type annotation
	var returnType ast.Expression
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume :
		p.nextToken() // move to type
		returnType = p.parseType()
	}

	// Expect '=>'
	if !p.expectPeek(lexer.ARROW) {
		return nil
	}

	p.nextToken() // move to body

	// Create arrow function expression
	fe := &ast.FunctionExpression{
		Token:      startToken,
		Parameters: params,
		ReturnType: returnType,
	}

	// Parse body - can be single expression or block
	if p.curTokenIs(lexer.LBRACE) {
		// Block body
		fe.Body = p.parseBlockStatement()
	} else {
		// Single expression body - wrap in return statement
		expr := p.parseExpression(LOWEST)
		fe.Body = &ast.BlockStatement{
			Token: p.curToken,
			Statements: []ast.Statement{
				&ast.ReturnStatement{
					Token:        p.curToken,
					ReturnValues: []ast.Expression{expr},
				},
			},
		}
	}

	return fe
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}

	p.peekError(t)
	return false
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{
		Token:     p.curToken,
		Function:  function,
		Arguments: p.parseExpressionList(lexer.RPAREN),
	}
	return exp
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	defer p.clearStopAtPipe()()
	list := []ast.Expression{}

	if p.peekToken.Type == end {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekToken.Type == lexer.COMMA {

		p.nextToken() //consume comma
		p.nextToken() // move unto next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	// '?.' followed by '(' is an optional call: fn?.(args)
	if p.curTokenIs(lexer.OPTIONAL_CHAIN) && p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // move onto '('
		call, ok := p.parseCallExpression(left).(*ast.CallExpression)
		if !ok {
			return nil
		}
		call.IsOptional = true
		return call
	}

	exp := &ast.DotExpression{
		Token:      p.curToken,
		Left:       left,
		IsOptional: p.curToken.Type == lexer.OPTIONAL_CHAIN,
	}

	// Right side of dot expression must be an identifier - allows context-aware keywords
	if !p.expectPeekFieldName() {
		return nil
	}

	exp.Right = p.parseFieldName()

	return exp
}

// parseMethodCallExpression handles Lua's method call syntax (obj:method(...)).
// It is only reached when isColonMethodCall has confirmed the ':' starts a method
// call, so the receiver is kept as-is and the call itself is parsed by the caller.
func (p *Parser) parseMethodCallExpression(left ast.Expression) ast.Expression {
	exp := &ast.DotExpression{
		Token:        p.curToken,
		Left:         left,
		IsMethodCall: true,
	}

	if !p.expectPeekFieldName() {
		return nil
	}

	exp.Right = p.parseFieldName()

	return exp
}

// isColonMethodCall reports whether a ':' in peek position starts a Lua method
// call (obj:method(...)) rather than a type annotation, a match arm's type
// pattern, or the ':' of a conditional type. It only says yes for the exact
// shape ': name (', so every other use of ':' keeps its current meaning.
func (p *Parser) isColonMethodCall() bool {
	if !p.peekTokenIs(lexer.COLON) {
		return false
	}

	state := p.SaveState()
	defer p.RestoreState(state)

	p.nextToken() // curToken = ':'
	if !lexer.CanBeFieldName(p.peekToken) {
		return false
	}

	p.nextToken() // curToken = method name
	// obj:m(...), obj:m{...} and obj:m"..."
	return p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.LBRACE) || p.peekTokenIs(lexer.STRING)
}

// exprToken is the token an expression starts with, or fallback.
func exprToken(e ast.Expression, fallback lexer.Token) lexer.Token {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Token
	case *ast.StringLiteral:
		return n.Token
	case *ast.NumberLiteral:
		return n.Token
	case *ast.BooleanLiteral:
		return n.Token
	case *ast.DotExpression:
		return exprToken(n.Left, n.Token)
	}
	return fallback
}

// genericBaseToken is the token of what type arguments follow, which has
// to name a type: 1<A> and "s"<A> are errors, not crashes.
func (p *Parser) genericBaseToken(base ast.Expression) lexer.Token {
	switch base.(type) {
	case *ast.Identifier, *ast.DotExpression:
	default:
		p.addError(fmt.Sprintf("type arguments follow a type name, not '%s'", base.String()), exprToken(base, p.curToken))
	}
	return exprToken(base, p.curToken)
}

// peekStartsAnnotation says whether a ':' in peek position starts a type
// annotation: `local x` then a `::label::` on the next line is a label.
func (p *Parser) peekStartsAnnotation() bool {
	if !p.peekTokenIs(lexer.COLON) {
		return false
	}
	state := p.SaveState()
	defer p.RestoreState(state)
	p.nextToken() // curToken = ':'
	return !p.peekTokenIs(lexer.COLON)
}

// isOptionalIndex reports whether a '?' in peek position starts an optional
// index access (items?[1]).
func (p *Parser) isOptionalIndex() bool {
	if !p.peekTokenIs(lexer.QUESTION) {
		return false
	}

	state := p.SaveState()
	defer p.RestoreState(state)

	p.nextToken() // curToken = '?'
	return p.peekTokenIs(lexer.LBRACKET)
}

// parseOptionalIndexExpression parses optional index access: items?[1].
func (p *Parser) parseOptionalIndexExpression(left ast.Expression) ast.Expression {
	if !p.expectPeek(lexer.LBRACKET) {
		return nil
	}

	exp := p.parseIndexExpression(left)
	if indexExpr, ok := exp.(*ast.IndexExpression); ok {
		indexExpr.IsOptional = true
	}

	return exp
}

// parsePipeExpression handles the pipeline operator (|>)
// Transforms: value |> func(args) into func(value, args)
func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	// Parse the right side with current precedence
	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)

	// If right is a call expression, prepend left as first argument
	if callExpr, ok := right.(*ast.CallExpression); ok {
		// Insert left expression as first argument by creating new call expression
		newArgs := make([]ast.Expression, 0, len(callExpr.Arguments)+1)
		newArgs = append(newArgs, left)
		newArgs = append(newArgs, callExpr.Arguments...)

		return &ast.CallExpression{
			Token:     callExpr.Token,
			Function:  callExpr.Function,
			Arguments: newArgs,
		}
	}

	// If right is just an identifier (function name without parens)
	// Create a call expression with left as the only argument
	return &ast.CallExpression{
		Token:     p.curToken,
		Function:  right,
		Arguments: []ast.Expression{left},
	}
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.addError(msg, p.peekToken)
}

func (p *Parser) peekPrecedence() int {
	// ':' only binds as an operator when it starts a method call; everywhere else
	// (type annotations, conditional types, match patterns) it must stay inert.
	if p.peekToken.Type == lexer.COLON {
		if p.isColonMethodCall() {
			return DOT
		}
		return LOWEST
	}

	// '?' binds only in '?[', the optional index operator. Optional types and
	// conditional types use '?' too, and must not be treated as operators.
	if p.peekToken.Type == lexer.QUESTION {
		if p.isOptionalIndex() {
			return CALL
		}
		return LOWEST
	}

	// `x as T` is on one line: a statement starting with a variable named
	// as (`local y = 5` then `as = y`) is not an assertion on the line
	// before.
	if p.peekToken.Type == lexer.AS && p.peekToken.Line != p.curToken.EndLine {
		return LOWEST
	}

	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	if t == lexer.ILLEGAL {
		lit := p.curToken.Literal
		switch {
		case strings.Contains(lit, " "):
			// The lexer's own message: "unterminated string", "invalid
			// escape sequence '\q'", ...
			p.addError(lit, p.curToken)
		case lit == "\x00":
			p.addError("unexpected NUL byte", p.curToken)
		case len(lit) == 1 && (lit[0] < ' ' || lit[0] >= 0x7f):
			p.addError(fmt.Sprintf("unexpected byte 0x%02X", lit[0]), p.curToken)
		default:
			p.addError(fmt.Sprintf("unexpected symbol '%s'", lit), p.curToken)
		}
		return
	}
	if t == lexer.END {
		// Most often the body of a declare function, which has none.
		if fd := p.bodilessDeclare; fd != nil && fd.Name != nil {
			p.addError(fmt.Sprintf("'declare function %s' cannot have a body: a declaration is only the signature of a function implemented elsewhere (such as in the engine). Remove the body and this 'end', or drop 'declare' to implement it here",
				fd.Name.Value), p.curToken)
			p.bodilessDeclare = nil
			return
		}
		p.addError("unexpected 'end': it closes nothing", p.curToken)
		return
	}
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.addError(msg, p.curToken)
}

func (p *Parser) Errors() []*ParseError {
	return p.errors
}

// Parse parses the entire program and returns a slice of statements
func (p *Parser) Parse() []ast.Statement {
	// Note: New() already initializes curToken and peekToken by calling nextToken() twice
	// So we don't need to call nextToken() here

	return p.parseStatements(lexer.EOF)
}

// parseStatements parses statements up to terminator or end of input. As in
// Lua, a return has to be the last statement of its block; anything after it
// is reported once, since the emitted Lua would not load.
func (p *Parser) parseStatements(terminator lexer.TokenType) []ast.Statement {
	defer p.clearStopAtPipe()()
	statements := []ast.Statement{}
	var returned *ast.ReturnStatement
	reported := false

	for !p.curTokenIs(terminator) && !p.curTokenIs(lexer.EOF) {
		start := p.curToken
		stmt := p.parseStatement()
		if !isNilNode(stmt) {
			// A stray 'end' is reported on its own.
			if returned != nil && !reported && start.Type != lexer.END {
				p.addError("'return' must be the last statement in a block", start)
				reported = true
			}
			if ret, ok := stmt.(*ast.ReturnStatement); ok {
				returned = ret
			}
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	if rightAssociative[p.curToken.Type] {
		precedence-- // the right side may hold the same operator again
	}
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseTypeAssertion(left ast.Expression) ast.Expression {
	assertion := &ast.TypeAssertion{
		Token:      p.curToken,
		Expression: left,
	}

	p.nextToken()
	// `as const` keeps literal types; const is not a type of its own.
	if p.curTokenIs(lexer.CONST) {
		assertion.TargetType = &ast.Identifier{Token: p.curToken, Value: "const"}
		return assertion
	}
	assertion.TargetType = p.parseType()

	return assertion
}

// parseGenericOrLessThan disambiguates between generic type instantiation and less-than operator
func (p *Parser) parseGenericOrLessThan(left ast.Expression) ast.Expression {
	// Only try to parse as generic if left is an identifier or dot expression
	switch left.(type) {
	case *ast.Identifier, *ast.DotExpression:
		// Try to parse as generic type instantiation
		if genericExpr := p.tryParseGenericType(left); genericExpr != nil {
			return genericExpr
		}
	}

	// Fall back to less-than operator
	return p.parseInfixExpression(left)
}

// tryParseGenericType attempts to parse generic type arguments
// Returns nil if this isn't a generic type instantiation
func (p *Parser) tryParseGenericType(baseExpr ast.Expression) ast.Expression {
	// Save the full parser state, lexer included: backtracking by restoring the
	// two lookahead tokens alone would silently drop every token the attempt
	// pulled from the lexer, turning `x < 10 then` into `x < 10` + lost `then`.
	startState := p.SaveState()
	startToken := p.curToken

	// Current token is '<', move past it
	p.nextToken()

	// Try to parse type arguments
	typeArgs := []ast.Expression{}

	// Parse first type argument
	if !p.isTypeToken(p.curToken.Type) {
		// Not a type token, this is a less-than operator
		p.RestoreState(startState)
		return nil
	}

	firstArg := p.parseType()
	if firstArg == nil {
		p.RestoreState(startState)
		return nil
	}
	typeArgs = append(typeArgs, firstArg)

	// Parse additional type arguments
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next type
		arg := p.parseType()
		if arg == nil {
			p.RestoreState(startState)
			return nil
		}
		typeArgs = append(typeArgs, arg)
	}

	// Must end with '>' (or >> which we split for nested generics)
	if !p.peekTokenIs(lexer.GT) && !p.peekTokenIs(lexer.RIGHT_SHIFT) {
		p.RestoreState(startState)
		return nil
	}

	// Type arguments in an expression are only for a call or a member:
	// Box<number>(1), Pool<Enemy>.new(). `print(a < b, c > d)` and
	// `a < b >> 1` compare.
	if p.peekTokenIs(lexer.GT) || p.peekTokenIs(lexer.RIGHT_SHIFT) {
		after := p.SaveState()
		p.nextToken() // '>'
		called := p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.DOT)
		p.RestoreState(after)
		if !called {
			p.RestoreState(startState)
			return nil
		}
	}

	// Handle >> as two > tokens for nested generics like Box<Box<T>>
	if p.peekTokenIs(lexer.RIGHT_SHIFT) {
		p.nextToken() // move to >>
		// Split >> into two > tokens: consume first >, keep second for next iteration
		p.curToken = lexer.Token{Type: lexer.GT, Literal: ">", Line: p.curToken.Line, Column: p.curToken.Column}
		p.peekToken = lexer.Token{Type: lexer.GT, Literal: ">", Line: p.curToken.Line, Column: p.curToken.Column + 1}
	} else {
		p.nextToken() // consume '>'
	}

	// Successfully parsed generic type
	token := startToken
	if ident, ok := baseExpr.(*ast.Identifier); ok {
		token = ident.Token
	}

	return &ast.GenericType{
		Token:         token,
		BaseType:      baseExpr,
		TypeArguments: typeArgs,
	}
}

// isTypeToken checks if a token can start a type expression
func (p *Parser) isTypeToken(tokenType lexer.TokenType) bool {
	switch tokenType {
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN,
		lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN, lexer.TABLE, lexer.LPAREN:
		return true
	default:
		return false
	}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseSpreadExpression() ast.Expression {
	// A bare '...' is Lua's vararg expression; only '...' followed by an operand
	// on the same line is a spread, as in {...items}. `local m = ...` then a
	// line starting with a name is the vararg, then a new statement.
	// What follows a spread is a name, a call, a parenthesised expression or
	// a table: `... - 1` and `... ~ 1` are operators on the vararg.
	spreadable := p.peekTokenIsIdentOrContextual() || p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.LBRACE)
	if !spreadable || p.peekToken.Line != p.curToken.Line {
		return &ast.VarargExpression{Token: p.curToken}
	}

	expression := &ast.SpreadExpression{
		Token: p.curToken,
	}

	p.nextToken()
	expression.Value = p.parseExpression(LOWEST)

	return expression
}

// peekStartsExpression reports whether the next token could begin an operand.
func (p *Parser) peekStartsExpression() bool {
	_, ok := p.prefixParseFns[p.peekToken.Type]
	return ok
}

func (p *Parser) parseAwaitExpression() ast.Expression {
	expression := &ast.AwaitExpression{
		Token: p.curToken, // 'await' token
	}

	p.nextToken()
	expression.Expression = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseVariableDeclaration() *ast.VariableDeclaration {
	decl := &ast.VariableDeclaration{
		Token:      p.curToken,
		IsConstant: p.curToken.Type == lexer.CONST,
		Names:      []*ast.Identifier{},
		Types:      []ast.Expression{},
		Values:     []ast.Expression{},
	}

	// Check for tuple destructuring: local (a, b) = func()
	if p.peekTokenIs(lexer.LPAREN) {
		decl.IsTupleDestructuring = true
		p.nextToken() // consume local/const
		p.nextToken() // consume (, now on first identifier

		// Parse first identifier
		if !p.curTokenIsIdentOrContextual() {
			p.addError("expected identifier in tuple destructuring", p.curToken)
			return nil
		}
		decl.Names = append(decl.Names, p.parseIdentifierOrContextual())

		// Parse type annotation if present
		if p.peekStartsAnnotation() {
			p.nextToken() // consume identifier
			p.nextToken() // consume :
			decl.Types = append(decl.Types, p.parseType())
		} else {
			decl.Types = append(decl.Types, nil)
		}

		// Parse additional variables in tuple
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume previous token/type
			p.nextToken() // consume comma, now on next identifier

			if !p.curTokenIsIdentOrContextual() {
				p.addError(fmt.Sprintf("expected identifier after comma in tuple, got %s", p.curToken.Type), p.curToken)
				return nil
			}

			decl.Names = append(decl.Names, p.parseIdentifierOrContextual())

			// Parse type annotation if present
			if p.peekStartsAnnotation() {
				p.nextToken() // consume identifier
				p.nextToken() // consume :
				decl.Types = append(decl.Types, p.parseType())
			} else {
				decl.Types = append(decl.Types, nil)
			}
		}

		// Expect closing paren
		if !p.expectPeek(lexer.RPAREN) {
			p.addError("expected ) after tuple destructuring", p.curToken)
			return nil
		}
	} else {
		// Regular variable declaration: local a, b = 1, 2
		// Parse first identifier (name) - allows context-aware keywords
		if !p.expectPeekIdentOrContextual() {
			return nil
		}
		decl.Names = append(decl.Names, p.parseIdentifierOrContextual())
		decl.Attribs = append(decl.Attribs, p.parseAttrib())

		// Parse type annotation if present
		if p.peekStartsAnnotation() {
			p.nextToken() // consume :
			p.nextToken() // move to type
			decl.Types = append(decl.Types, p.parseType())
		}

		// Parse additional variables after commas
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume previous token/type
			p.nextToken() // consume comma, now on next identifier

			if !p.curTokenIsIdentOrContextual() {
				p.addError(fmt.Sprintf("expected identifier after comma, got %s", p.curToken.Type), p.curToken)
				return nil
			}

			decl.Names = append(decl.Names, p.parseIdentifierOrContextual())
			decl.Attribs = append(decl.Attribs, p.parseAttrib())

			// Parse type annotation if present
			if p.peekStartsAnnotation() {
				p.nextToken() // consume :
				p.nextToken() // move to type
				decl.Types = append(decl.Types, p.parseType())
			} else {
				// No type for this variable
				decl.Types = append(decl.Types, nil)
			}
		}
	}

	// Parse initializer(s) if present
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to expression

		// Parse first value
		decl.Values = append(decl.Values, p.parseExpression(LOWEST))

		// Parse additional values after commas (for multiple assignment)
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume current expression
			p.nextToken() // consume comma
			decl.Values = append(decl.Values, p.parseExpression(LOWEST))
		}
	}

	return decl
}

func (p *Parser) parseType() ast.Expression {
	// Check for type guard syntax: paramName is Type
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.IS) {
		paramName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken() // consume identifier
		p.nextToken() // consume 'is', now on guard type

		guardType := p.parseType()
		if guardType == nil {
			return nil
		}

		return &ast.TypeGuardType{
			Token:     paramName.Token,
			ParamName: paramName,
			GuardType: guardType,
		}
	}

	var typeExpr ast.Expression

	switch p.curToken.Type {
	case lexer.LPAREN:
		// Could be tuple type or function type
		return p.parseTypeSuffix(p.parseTupleOrFunctionType())
	case lexer.FUNCTION:
		// function(params): Return
		return p.parseKeywordFunctionType()
	case lexer.LBRACE:
		// Mapped type: { [K in T]: U }
		return p.parseMappedType()
	case lexer.TABLE:
		// table<K, V>
		typeExpr = p.parseTableType()
	case lexer.KEYOF:
		// keyof T
		return p.parseKeyofType()
	case lexer.TYPEOF:
		// typeof value
		return p.parseTypeofExpression()
	case lexer.STRING:
		// String literal in type position (for literal types)
		typeExpr = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		// Number literal in type position (for literal types)
		value, _ := parseLuaNumber(p.curToken.Literal)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.TRUE:
		// Boolean literal type: true
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case lexer.FALSE:
		// Boolean literal type: false
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case lexer.TEMPLATE_STRING:
		// Template literal type: `Hello ${string}` or `${T}_${U}`
		return p.parseTemplateLiteralType()
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		typeExpr = p.parseTypeName()
	default:
		p.addError(fmt.Sprintf("expected a type, got '%s'", p.curToken.Literal), p.curToken)
		return nil
	}

	// Check for suffixes and modifiers
	return p.parseTypeSuffix(typeExpr)
}

// parseMappedType parses mapped types: { [K in T]: U }
// Also supports: { readonly [K in T]: U }, { [K in T]?: U }, { readonly [K in T]?: U }
func (p *Parser) parseMappedType() ast.Expression {
	mappedType := &ast.MappedType{
		Token: p.curToken, // '{'
	}

	// Check for 'readonly' modifier
	if p.peekTokenIs(lexer.READONLY) {
		p.nextToken() // consume 'readonly'
		mappedType.IsReadonly = true
	}

	if !p.expectPeek(lexer.LBRACKET) {
		return nil
	}
	p.nextToken() // move past '['

	// Parse the type parameter (K)
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}
	mappedType.TypeParam = &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	// Expect 'in' keyword
	if !p.expectPeek(lexer.IN) {
		return nil
	}

	// Parse the constraint (what K iterates over)
	p.nextToken() // move to constraint type
	mappedType.Constraint = p.parseType()
	if mappedType.Constraint == nil {
		return nil
	}

	// Expect ']'
	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	// Check for optional '?' modifier
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken() // consume '?'
		mappedType.IsOptional = true
	}

	// Expect ':'
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse the value type
	p.nextToken() // move to value type
	mappedType.ValueType = p.parseType()
	if mappedType.ValueType == nil {
		return nil
	}

	// Expect '}'
	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return mappedType
}

// parseTemplateLiteralType parses template literal types: `Hello ${string}` or `${T}_${U}`
func (p *Parser) parseTemplateLiteralType() ast.Expression {
	templateLiteral := &ast.TemplateLiteralType{
		Token:      p.curToken,
		RawLiteral: p.curToken.Literal,
		Parts:      []string{},
		Types:      []ast.Expression{},
	}

	// Parse the template string to extract parts and type expressions
	// The template string is in p.curToken.Literal
	content := p.curToken.Literal
	var parts []string
	var types []ast.Expression
	var currentPart strings.Builder

	i := 0
	for i < len(content) {
		if i < len(content)-1 && content[i] == '$' && content[i+1] == '{' {
			// Found ${...} - save current part and parse the type
			parts = append(parts, currentPart.String())
			currentPart.Reset()

			// Find the matching }
			i += 2 // Skip ${
			start := i
			braceCount := 1
			for i < len(content) && braceCount > 0 {
				if content[i] == '{' {
					braceCount++
				} else if content[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				// Unmatched braces
				return nil
			}

			// Parse the type expression
			typeStr := content[start:i]
			// Create a mini-parser for the type expression
			typeLexer := lexer.New(typeStr)
			typeParser := New(typeLexer)
			typeParser.nextToken() // Initialize parser
			typeExpr := typeParser.parseType()
			if typeExpr == nil {
				return nil
			}
			types = append(types, typeExpr)

			i++ // Skip closing }
		} else {
			// Regular character
			currentPart.WriteByte(content[i])
			i++
		}
	}

	// Add the final part
	parts = append(parts, currentPart.String())

	templateLiteral.Parts = parts
	templateLiteral.Types = types

	return templateLiteral
}

func (p *Parser) parseSimpleType() ast.Expression {
	switch p.curToken.Type {
	case lexer.LPAREN:
		return p.parseTypeSuffix(p.parseTupleOrFunctionType())
	case lexer.FUNCTION:
		return p.parseKeywordFunctionType()
	case lexer.TABLE:
		return p.parseTableType()
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
		return nil
	}
}

// parseKeywordFunctionType parses the `function(params): Return` type form used
// by declaration files, as in `insert: function(t: any, pos: number?): void`.
// Without it the whole annotation was skipped and the member silently became
// 'any' -- and the parameter names leaked out as extra interface properties.
func (p *Parser) parseKeywordFunctionType() ast.Expression {
	funcType := &ast.FunctionType{
		Token: p.curToken, // 'function'
	}

	// A bare `function` means any callable; leaving Parameters nil marks it as
	// unconstrained, which an empty (but non-nil) list would not.
	if !p.peekTokenIs(lexer.LPAREN) {
		return funcType
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	funcType.Parameters = p.parseFunctionParameters()
	if funcType.Parameters == nil {
		return nil
	}

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // move to return type
		funcType.ReturnType = p.parseType()
	}

	return funcType
}

func (p *Parser) parseTypeSuffix(baseType ast.Expression) ast.Expression {
	if isNilNode(baseType) {
		return nil
	}
	currentType := baseType

	// First pass: handle high-precedence suffixes (arrays, generics, optional)
	// These bind tighter than union types
	for {
		switch {
		case p.peekTokenIs(lexer.LBRACKET) && p.peekToken.Line == p.curToken.Line:
			// Could be array type T[] or indexed access type T[K]. A '[' on
			// the next line starts something else (an index signature).
			p.nextToken() // consume '['
			p.nextToken() // move to content or ']'

			if p.curTokenIs(lexer.RBRACKET) {
				// Empty brackets: T[] is array type
				currentType = &ast.ArrayType{
					Token:       p.curToken,
					ElementType: currentType,
				}
			} else {
				// Non-empty brackets: T[K] is indexed access type
				indexType := p.parseType()
				if !p.expectPeek(lexer.RBRACKET) {
					return nil
				}
				currentType = &ast.IndexExpression{
					Token: p.curToken,
					Left:  currentType,
					Index: indexType,
				}
			}

		case p.peekTokenIs(lexer.LT):
			// Generic type: T<U>
			p.nextToken() // consume '<'
			p.nextToken() // move to first type argument

			typeArgs := []ast.Expression{}
			typeArgs = append(typeArgs, p.parseType())

			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next type
				typeArgs = append(typeArgs, p.parseType())
			}

			if !p.expectPeekGT() {
				return nil
			}

			currentType = &ast.GenericType{
				Token:         p.genericBaseToken(baseType),
				BaseType:      baseType,
				TypeArguments: typeArgs,
			}

		case p.peekTokenIs(lexer.QUESTION):
			// Optional type: T?
			p.nextToken()
			currentType = &ast.OptionalType{
				Token: p.curToken,
				Type:  currentType,
			}

		default:
			// No more high-precedence suffixes
			goto checkIntersection
		}
	}

checkIntersection:
	// Second pass: handle intersection types (higher precedence than unions)
	if p.peekTokenIs(lexer.AMPERSAND) {
		types := []ast.Expression{currentType}
		intersectionToken := p.peekToken
		for p.peekTokenIs(lexer.AMPERSAND) {
			p.nextToken() // consume '&'
			p.nextToken() // move to next type
			// Parse the next type WITHOUT processing intersections or unions
			// skipOptional=false since parseTypeSuffix is called from normal parseType context
			nextType := p.parseNonUnionIntersectionType(false)
			if nextType != nil {
				types = append(types, nextType)
			}
		}
		currentType = &ast.IntersectionType{
			Token: intersectionToken,
			Types: types,
		}
	}

	// Third pass: handle union types (lowest precedence)
	if p.peekTokenIs(lexer.PIPE) {
		types := []ast.Expression{currentType}
		unionToken := p.peekToken
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken() // consume '|'
			p.nextToken() // move to next type
			// Parse the next type WITHOUT processing unions
			// But we DO process intersections, so A | B & C parses as A | (B & C)
			// skipOptional=false since parseTypeSuffix is called from normal parseType context
			nextType := p.parseNonUnionIntersectionType(false)
			// Check if this type has intersection suffix
			if p.peekTokenIs(lexer.AMPERSAND) {
				intersectionTypes := []ast.Expression{nextType}
				intersectionToken := p.peekToken
				for p.peekTokenIs(lexer.AMPERSAND) {
					p.nextToken() // consume '&'
					p.nextToken() // move to next type
					intersectionMember := p.parseNonUnionIntersectionType(false)
					if intersectionMember != nil {
						intersectionTypes = append(intersectionTypes, intersectionMember)
					}
				}
				nextType = &ast.IntersectionType{
					Token: intersectionToken,
					Types: intersectionTypes,
				}
			}
			if nextType != nil {
				types = append(types, nextType)
			}
		}
		currentType = &ast.UnionType{
			Token: unionToken,
			Types: types,
		}
	}

	// Fourth pass: handle conditional types (lowest precedence)
	// T extends U ? X : Y
	if p.peekTokenIs(lexer.EXTENDS) {
		checkType := currentType
		p.nextToken() // consume 'extends'
		p.nextToken() // move to extends type

		// Parse the extends type (without allowing nested conditionals)
		// skipOptional=true to prevent consuming the ? that's part of the conditional syntax
		extendsType := p.parseSimpleTypeWithSuffixes(true)

		// After parseSimpleTypeWithSuffixes, we're positioned at the last token
		// of the extends type. We need to check if the NEXT token is ?
		if !p.peekTokenIs(lexer.QUESTION) {
			return nil
		}
		p.nextToken() // consume ?

		p.nextToken() // move to true type
		// Use parseType() to allow nested conditional types in true branch
		trueType := p.parseType()

		if !p.peekTokenIs(lexer.COLON) {
			return nil
		}
		p.nextToken() // consume :

		p.nextToken() // move to false type
		// Use parseType() to allow nested conditional types in false branch
		falseType := p.parseType()

		// After parsing falseType, we're positioned correctly at its last token
		// Don't advance further - let the caller handle positioning

		// Get the token from checkType
		var condToken lexer.Token
		switch ct := checkType.(type) {
		case *ast.Identifier:
			condToken = ct.Token
		case *ast.UnionType:
			condToken = ct.Token
		case *ast.IntersectionType:
			condToken = ct.Token
		default:
			condToken = p.curToken
		}

		currentType = &ast.ConditionalType{
			Token:       condToken,
			CheckType:   checkType,
			ExtendsType: extendsType,
			TrueType:    trueType,
			FalseType:   falseType,
		}
	}

	return currentType
}

// parseSimpleTypeWithSuffixes parses a type with all suffixes including union and intersection,
// but NOT conditional types (to avoid infinite recursion in conditional type parsing)
// skipOptional: if true, don't parse ? as optional type (used when parsing conditional type extends clause)
func (p *Parser) parseSimpleTypeWithSuffixes(skipOptional bool) ast.Expression {
	var typeExpr ast.Expression

	switch p.curToken.Type {
	case lexer.LPAREN:
		return p.parseTypeSuffix(p.parseTupleOrFunctionType())
	case lexer.FUNCTION:
		return p.parseKeywordFunctionType()
	case lexer.TABLE:
		typeExpr = p.parseTableType()
	case lexer.KEYOF:
		return p.parseKeyofType()
	case lexer.TYPEOF:
		return p.parseTypeofExpression()
	case lexer.STRING:
		typeExpr = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		value, _ := parseLuaNumber(p.curToken.Literal)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.TRUE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case lexer.FALSE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		typeExpr = p.parseTypeName()
	default:
		return nil
	}

	currentType := typeExpr

	// Handle high-precedence suffixes (arrays, generics, optional)
	for {
		switch {
		case p.peekTokenIs(lexer.LBRACKET):
			p.nextToken() // consume '['
			p.nextToken() // move to content or ']'

			if p.curTokenIs(lexer.RBRACKET) {
				// Empty brackets: T[] is array type
				currentType = &ast.ArrayType{
					Token:       exprToken(typeExpr, p.curToken),
					ElementType: currentType,
				}
			} else {
				// Non-empty brackets: T[K] is indexed access type
				indexType := p.parseType()
				if !p.expectPeek(lexer.RBRACKET) {
					return nil
				}
				currentType = &ast.IndexExpression{
					Token: p.curToken,
					Left:  currentType,
					Index: indexType,
				}
			}

		case p.peekTokenIs(lexer.LT):
			p.nextToken() // consume '<'
			p.nextToken() // move to first type argument

			typeArgs := []ast.Expression{}
			typeArgs = append(typeArgs, p.parseType())

			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next type
				typeArgs = append(typeArgs, p.parseType())
			}

			if !p.expectPeekGT() {
				return nil
			}

			currentType = &ast.GenericType{
				Token:         p.genericBaseToken(typeExpr),
				BaseType:      typeExpr,
				TypeArguments: typeArgs,
			}

		case p.peekTokenIs(lexer.QUESTION) && !skipOptional:
			p.nextToken()
			currentType = &ast.OptionalType{
				Token: p.curToken,
				Type:  currentType,
			}

		default:
			goto checkIntersection
		}
	}

checkIntersection:
	// Handle intersection types
	if p.peekTokenIs(lexer.AMPERSAND) {
		types := []ast.Expression{currentType}
		intersectionToken := p.peekToken
		for p.peekTokenIs(lexer.AMPERSAND) {
			p.nextToken() // consume '&'
			p.nextToken() // move to next type
			// Pass through skipOptional parameter
			nextType := p.parseNonUnionIntersectionType(skipOptional)
			if nextType != nil {
				types = append(types, nextType)
			}
		}
		currentType = &ast.IntersectionType{
			Token: intersectionToken,
			Types: types,
		}
	}

	// Handle union types
	if p.peekTokenIs(lexer.PIPE) {
		types := []ast.Expression{currentType}
		unionToken := p.peekToken
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken() // consume '|'
			p.nextToken() // move to next type
			// Pass through skipOptional parameter
			nextType := p.parseNonUnionIntersectionType(skipOptional)
			// Check for intersection in this union member
			if p.peekTokenIs(lexer.AMPERSAND) {
				intersectionTypes := []ast.Expression{nextType}
				intersectionToken := p.peekToken
				for p.peekTokenIs(lexer.AMPERSAND) {
					p.nextToken() // consume '&'
					p.nextToken() // move to next type
					intersectionMember := p.parseNonUnionIntersectionType(skipOptional)
					if intersectionMember != nil {
						intersectionTypes = append(intersectionTypes, intersectionMember)
					}
				}
				nextType = &ast.IntersectionType{
					Token: intersectionToken,
					Types: intersectionTypes,
				}
			}
			if nextType != nil {
				types = append(types, nextType)
			}
		}
		currentType = &ast.UnionType{
			Token: unionToken,
			Types: types,
		}
	}

	return currentType
}

// parseNonUnionIntersectionType parses a type with all suffixes EXCEPT union and intersection types
// This is used when parsing union/intersection members to avoid nested structures
// skipOptional: if true, don't parse ? as optional type (used when parsing conditional type extends clause)
func (p *Parser) parseNonUnionIntersectionType(skipOptional bool) ast.Expression {
	var typeExpr ast.Expression

	switch p.curToken.Type {
	case lexer.LPAREN:
		// Could be tuple type or function type
		return p.parseTypeSuffix(p.parseTupleOrFunctionType())
	case lexer.FUNCTION:
		// function(params): Return, e.g. `string | function(s: string): string`
		return p.parseKeywordFunctionType()
	case lexer.TABLE:
		// table<K, V>
		typeExpr = p.parseTableType()
	case lexer.STRING:
		// String literal in type position (for literal types)
		typeExpr = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		// Number literal in type position (for literal types)
		value, _ := parseLuaNumber(p.curToken.Literal)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.TRUE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case lexer.FALSE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		typeExpr = p.parseTypeName()
	default:
		return nil
	}

	currentType := typeExpr

	// Handle high-precedence suffixes (arrays, generics, optional) but NOT unions
	for {
		switch {
		case p.peekTokenIs(lexer.LBRACKET):
			// Array type: T[]
			p.nextToken() // consume '['
			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}
			currentType = &ast.ArrayType{
				Token:       p.curToken,
				ElementType: currentType,
			}

		case p.peekTokenIs(lexer.LT):
			// Generic type: T<U>
			p.nextToken() // consume '<'
			p.nextToken() // move to first type argument

			typeArgs := []ast.Expression{}
			typeArgs = append(typeArgs, p.parseType())

			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next type
				typeArgs = append(typeArgs, p.parseType())
			}

			if !p.expectPeekGT() {
				return nil
			}

			currentType = &ast.GenericType{
				Token:         p.genericBaseToken(typeExpr),
				BaseType:      typeExpr,
				TypeArguments: typeArgs,
			}

		case p.peekTokenIs(lexer.QUESTION) && !skipOptional:
			// Optional type: T?
			p.nextToken() // consume '?'
			currentType = &ast.OptionalType{
				Token: p.curToken,
				Type:  currentType,
			}

		default:
			// No more high-precedence suffixes, return without processing unions
			return currentType
		}
	}
}

func (p *Parser) parseTableType() ast.Expression {
	tableToken := p.curToken

	// Expect '<'
	if !p.expectPeek(lexer.LT) {
		return nil
	}

	p.nextToken() // move to key type
	keyType := p.parseType()

	// Expect ','
	if !p.expectPeek(lexer.COMMA) {
		return nil
	}

	p.nextToken() // move to value type
	valueType := p.parseType()

	// Expect '>'
	if !p.expectPeekGT() {
		return nil
	}

	return &ast.TableType{
		Token:     tableToken,
		KeyType:   keyType,
		ValueType: valueType,
	}
}

func (p *Parser) parseKeyofType() ast.Expression {
	keyofToken := p.curToken

	p.nextToken() // move to the object type
	objectType := p.parseType()

	if objectType == nil {
		return nil
	}

	return &ast.KeyofType{
		Token:      keyofToken,
		ObjectType: objectType,
	}
}

func (p *Parser) parseTypeofExpression() ast.Expression {
	typeofToken := p.curToken

	p.nextToken() // move to the expression
	// Parse the expression (variable name, member access, etc.)
	expr := p.parseExpression(LOWEST)

	if expr == nil {
		return nil
	}

	return &ast.TypeofExpression{
		Token:      typeofToken,
		Expression: expr,
	}
}

func (p *Parser) parseTupleOrFunctionType() ast.Expression {
	parenToken := p.curToken

	// Parse parameter-like list
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		// Empty parameter list
		p.nextToken()
	} else {
		p.nextToken() // move past '('

		// Check if this is a named parameter (function type) or just types (tuple)
		isNamedParam := p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON)

		if isNamedParam {
			// Function type
			param := p.parseParameter()
			params = append(params, param)

			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next param
				params = append(params, p.parseParameter())
			}

			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}
		} else {
			// Tuple type - just types, no names
			types := []ast.Expression{}
			types = append(types, p.parseType())

			for p.peekTokenIs(lexer.COMMA) {
				p.nextToken() // consume comma
				p.nextToken() // move to next type
				types = append(types, p.parseType())
			}

			if !p.expectPeek(lexer.RPAREN) {
				return nil
			}

			// Check if this is followed by => (making it a function type)
			if p.peekTokenIs(lexer.ARROW) {
				// Convert types to anonymous parameters
				for _, t := range types {
					params = append(params, &ast.Parameter{
						Token: parenToken,
						Type:  t,
					})
				}
			} else {
				// (T) is T, grouped: (string | number)[]. Several types are
				// a tuple.
				if len(types) == 1 {
					return types[0]
				}
				return &ast.TupleType{
					Token: parenToken,
					Types: types,
				}
			}
		}
	}

	// Check for arrow (function type)
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // consume '=>'
		p.nextToken() // move to return type

		returnType := p.parseType()

		return &ast.FunctionType{
			Token:      parenToken,
			Parameters: params,
			ReturnType: returnType,
		}
	}

	// Just parenthesized type or empty tuple
	if len(params) == 0 {
		return &ast.TupleType{
			Token: parenToken,
			Types: []ast.Expression{},
		}
	}

	// Single parameter without arrow - error?
	return nil
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// Context-aware keyword helpers
// These keywords can be used as type names OR as identifiers depending on context
// isContextualKeyword reports whether a keyword token may be a name: every
// keyword of Lunar's own, since in Lua they are all ordinary names.
func (p *Parser) isContextualKeyword(t lexer.TokenType) bool {
	return lexer.IsSoftKeyword(t)
}

func (p *Parser) curTokenIsIdentOrContextual() bool {
	return p.curToken.Type == lexer.IDENT || p.isContextualKeyword(p.curToken.Type)
}

func (p *Parser) peekTokenIsIdentOrContextual() bool {
	return p.peekToken.Type == lexer.IDENT || p.isContextualKeyword(p.peekToken.Type)
}

// ParserState represents a saved state of the parser for lookahead
type ParserState struct {
	curToken   lexer.Token
	peekToken  lexer.Token
	lexerState lexer.LexerState
	errorCount int
	pending    *lexer.Token
}

// SaveState saves the current parser state for lookahead
func (p *Parser) SaveState() ParserState {
	return ParserState{
		curToken:   p.curToken,
		peekToken:  p.peekToken,
		lexerState: p.l.SaveState(),
		errorCount: len(p.errors),
		pending:    p.pending,
	}
}

// RestoreState restores a previously saved parser state
func (p *Parser) RestoreState(state ParserState) {
	p.curToken = state.curToken
	p.peekToken = state.peekToken
	p.pending = state.pending
	p.l.RestoreState(state.lexerState)
	// A look-ahead that is abandoned leaves no errors behind.
	if state.errorCount < len(p.errors) {
		p.errors = p.errors[:state.errorCount]
	}
}

// luaReserved are Lua's own keywords, which can never name anything.
var luaReserved = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true,
	"false": true, "for": true, "function": true, "goto": true, "if": true, "in": true,
	"local": true, "nil": true, "not": true, "or": true, "repeat": true, "return": true,
	"then": true, "true": true, "until": true, "while": true,
}

// isWord reports whether s has the shape of an identifier.
func isWord(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || (i > 0 && '0' <= r && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// unclosedAtEOF reports a construct that the end of input cut off: Lua
// would refuse it, and filling in the missing token silently would compile
// code that was never finished.
func (p *Parser) unclosedAtEOF(closer string, opener lexer.Token) {
	if !p.curTokenIs(lexer.EOF) {
		return
	}
	what := fmt.Sprintf("the block that starts on line %d", opener.Line)
	if lexer.LookupIdent(opener.Literal) != lexer.IDENT || opener.Literal == "{" {
		what = fmt.Sprintf("the '%s' on line %d", opener.Literal, opener.Line)
	}
	p.addError(fmt.Sprintf("missing '%s' to close %s", closer, what), opener)
}

func (p *Parser) expectPeekIdentOrContextual() bool {
	if p.peekTokenIsIdentOrContextual() {
		p.nextToken()
		return true
	}

	p.peekError(lexer.IDENT)
	return false
}

// expectPeekFieldName advances when the next token can serve as a field name.
// Used where a name follows '.' or ':', which is unambiguous enough to accept
// any Lunar keyword Lua does not reserve (s:match(...), event.type, ...).
func (p *Parser) expectPeekFieldName() bool {
	if lexer.CanBeFieldName(p.peekToken) {
		p.nextToken()
		return true
	}

	p.peekError(lexer.IDENT)
	return false
}

// parseFieldName builds an identifier from a field-name token.
func (p *Parser) parseFieldName() *ast.Identifier {
	if !lexer.CanBeFieldName(p.curToken) {
		return nil
	}

	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseIdentifierOrContextual() *ast.Identifier {
	if p.curTokenIsIdentOrContextual() {
		return &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
	}
	return nil
}

// peekStartsMember says whether the token after a word shows the word is a
// member's name: name: T, name = v, name?: T, name(...), name<T>(...).
func (p *Parser) peekStartsMember() bool {
	switch p.peekToken.Type {
	case lexer.COLON, lexer.ASSIGN, lexer.QUESTION, lexer.LPAREN, lexer.LT:
		return true
	}
	return false
}

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{
		Token: p.curToken,
	}

	// Check for visibility modifiers (for constructor parameter properties).
	// A modifier word followed by no name is the parameter's own name, as in
	// Lua: function f(public, private).
	modifier := p.curToken.Type
	if !p.peekTokenIsIdentOrContextual() {
		modifier = lexer.IDENT
	}
	switch modifier {
	case lexer.PUBLIC, lexer.PRIVATE, lexer.PROTECTED:
		param.Visibility = p.curToken.Literal
		p.nextToken()
		// Check for readonly after visibility
		if p.curToken.Type == lexer.READONLY {
			param.IsReadonly = true
			p.nextToken()
		}
	case lexer.READONLY:
		param.IsReadonly = true
		p.nextToken()
		// Check for visibility after readonly (readonly public is also valid)
		switch p.curToken.Type {
		case lexer.PUBLIC, lexer.PRIVATE, lexer.PROTECTED:
			param.Visibility = p.curToken.Literal
			p.nextToken()
		}
	}

	// Check for rest parameter (...name) or Lua's bare vararg (... / ...: T)
	if p.curTokenIs(lexer.ELLIPSIS) {
		param.IsRest = true

		if !p.peekTokenIsIdentOrContextual() {
			// Bare vararg: the name stays "..." so it passes straight through to Lua.
			param.Name = &ast.Identifier{Token: p.curToken, Value: "..."}

			if p.peekTokenIs(lexer.COLON) {
				p.nextToken() // consume :
				p.nextToken() // move onto type
				param.Type = p.parseType()
			} else if p.isTypeToken(p.peekToken.Type) {
				// Unnamed typed vararg written without a colon, as in `...any`.
				p.nextToken()
				param.Type = p.parseType()
			}

			return param
		}

		p.nextToken()
	}

	// Any word will do, Lunar's own keywords included (string.gmatch's
	// declaration names a parameter `match`): only Lua's reserved words and
	// punctuation are refused.
	if !isWord(p.curToken.Literal) || luaReserved[p.curToken.Literal] {
		p.addError(fmt.Sprintf("expected a parameter name, got '%s'", p.curToken.Literal), p.curToken)
	}
	param.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for optional parameter (name?: type)
	if p.peekTokenIs(lexer.QUESTION) {
		param.IsOptional = true
		p.nextToken() // consume ?
	}

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consumes :
		p.nextToken() // moves onto type
		param.Type = p.parseType()
	}

	return param
}

func (p *Parser) parseFunctionParameters() []*ast.Parameter {
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()

	//first param
	param := p.parseParameter()
	params = append(params, param)

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		param = p.parseParameter()
		params = append(params, param)
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseFunctionDeclaration() *ast.FunctionDeclaration {
	fd := p.parseFunctionSignature()
	if fd == nil || p.ambient {
		return fd
	}

	fd.Body = p.parseBlockStatement()

	return fd
}

// parseFunctionSignature parses `function name<T>(params): ReturnType` and
// stops on its last token, leaving any body to the caller.
func (p *Parser) parseFunctionSignature() *ast.FunctionDeclaration {
	fd := &ast.FunctionDeclaration{
		Token: p.curToken,
	}

	//parse function name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	fd.Name = p.parseIdentifierOrContextual()

	// function M.add(), function a.b.c:m(): what comes before the last
	// name is the table the function goes into.
	for p.peekTokenIs(lexer.DOT) || p.peekTokenIs(lexer.COLON) {
		isMethod := p.peekTokenIs(lexer.COLON)
		p.nextToken() // '.' or ':'
		sep := p.curToken
		if !p.expectPeekFieldName() {
			return nil
		}
		if fd.Receiver == nil {
			fd.Receiver = fd.Name
		} else {
			fd.Receiver = &ast.DotExpression{Token: sep, Left: fd.Receiver, Right: fd.Name}
		}
		fd.Name = p.parseFieldName()
		if isMethod {
			fd.IsMethod = true
			break
		}
	}

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		fd.GenericParams = p.parseGenericParameters()
	}

	//parse the parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	fd.Parameters = p.parseFunctionParameters()

	if p.peekStartsAnnotation() {
		p.nextToken() //consume :
		p.nextToken() // move onto return type
		fd.ReturnType = p.parseType()
	}

	return fd
}

// parseAsyncFunctionDeclaration parses async function declarations
func (p *Parser) parseAsyncFunctionDeclaration() *ast.FunctionDeclaration {
	// Current token is 'async'
	asyncToken := p.curToken

	// Expect 'function' keyword
	if !p.expectPeek(lexer.FUNCTION) {
		return nil
	}

	fd := &ast.FunctionDeclaration{
		Token:   asyncToken,
		IsAsync: true,
	}

	// Parse function name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	fd.Name = p.parseIdentifierOrContextual()

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		fd.GenericParams = p.parseGenericParameters()
	}

	// Parse the parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	fd.Parameters = p.parseFunctionParameters()

	// Parse optional return type (a `::label::` first in the body is not one)
	if p.peekStartsAnnotation() {
		p.nextToken() // consume :
		p.nextToken() // move onto return type
		fd.ReturnType = p.parseType()
	}

	fd.Body = p.parseBlockStatement()

	return fd
}

// parseFunctionExpression parses anonymous function expressions like: function(x: number): number return x end
func (p *Parser) parseFunctionExpression() ast.Expression {
	fe := &ast.FunctionExpression{
		Token: p.curToken, // 'function' token
	}

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		fe.GenericParams = p.parseGenericParameters()
	}

	// Parse the parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	fe.Parameters = p.parseFunctionParameters()

	// Parse optional return type (a `::label::` first in the body is not one)
	if p.peekStartsAnnotation() {
		p.nextToken() // consume :
		p.nextToken() // move onto return type
		fe.ReturnType = p.parseType()
	}

	// Parse the body
	fe.Body = p.parseBlockStatement()

	return fe
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	return p.parseBlockStatementUntil(lexer.END)
}

// parseBlockStatementUntil parses statements up to a terminator, which is 'end'
// for most blocks and 'until' for a repeat loop.
func (p *Parser) parseBlockStatementUntil(terminator lexer.TokenType) *ast.BlockStatement {
	block := &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken()
	block.Statements = p.parseStatements(terminator)
	block.EndToken = p.curToken
	if terminator == lexer.UNTIL {
		p.unclosedAtEOF("until", block.Token)
	} else {
		p.unclosedAtEOF("end", block.Token)
	}

	return block
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	stmt.ReturnValues = []ast.Expression{}

	// A return has no values when the block ends right after it. Anything
	// else, on this line or the next, is its value list, as in Lua: a data
	// file's `return` is often followed by its table on the next line.
	switch p.peekToken.Type {
	case lexer.END, lexer.ELSE, lexer.ELSEIF, lexer.UNTIL, lexer.EOF:
		return stmt
	case lexer.SEMICOLON:
		p.nextToken()
		return stmt
	}

	p.nextToken() // move past 'return'

	// Parse first return value
	stmt.ReturnValues = append(stmt.ReturnValues, p.parseExpression(LOWEST))

	// Parse additional return values after commas
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume current expression
		p.nextToken() // consume comma
		stmt.ReturnValues = append(stmt.ReturnValues, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	// Try to parse as expression first
	start := p.curToken
	expr := p.parseExpression(LOWEST)

	// Lua allows several assignment targets: a, b = b, a
	targets := []ast.Expression{expr}
	if p.peekTokenIs(lexer.COMMA) {
		state := p.SaveState()

		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume ','
			p.nextToken() // move to next target
			targets = append(targets, p.parseExpression(LOWEST))
		}

		if !p.peekTokenIs(lexer.ASSIGN) {
			// Not an assignment after all; leave the comma to the caller.
			p.RestoreState(state)
			return &ast.ExpressionStatement{
				Token:      p.curToken,
				Expression: expr,
			}
		}
	}

	// Check if this is an assignment
	if p.peekTokenIs(lexer.ASSIGN) {
		assignToken := p.peekToken
		// Lua assigns to a name, a field or an index only: `x + 1 = 2`
		// and `f() = {}` were written out and did not load.
		for _, target := range targets {
			switch t := target.(type) {
			case *ast.Identifier:
			case *ast.DotExpression:
				if t.IsOptional || t.IsMethodCall {
					p.addError("cannot assign through '?.'; assign to the field of a value known to be there", t.Token)
				}
			case *ast.IndexExpression:
				if t.IsOptional {
					p.addError("cannot assign through '?['; assign to the element of a value known to be there", t.Token)
				}
			default:
				p.addError(fmt.Sprintf("cannot assign to '%s': only a name, a field or an index can be assigned", target.String()), assignToken)
			}
		}
		p.nextToken() // consume '='
		p.nextToken() // move to first value expression

		values := []ast.Expression{p.parseExpression(LOWEST)}
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume ','
			p.nextToken() // move to next value
			values = append(values, p.parseExpression(LOWEST))
		}

		return &ast.AssignmentStatement{
			Token:   assignToken,
			Targets: targets,
			Values:  values,
		}
	}

	// Otherwise it is an expression on its own, which Lua allows only for
	// a call (Lunar adds match, and await, which is refused later).
	switch expr.(type) {
	case *ast.CallExpression, *ast.MatchExpression, *ast.AwaitExpression:
	default:
		if !isNilNode(expr) {
			p.addError(fmt.Sprintf("'%s' is not a statement: Lua runs only calls and assignments", start.Literal), start)
		}
	}
	return &ast.ExpressionStatement{
		Token:      p.curToken,
		Expression: expr,
	}
}

// isNilNode reports whether a node is nil, including a nil pointer of a
// concrete node type wrapped in the interface: every parseX function returns
// its own pointer type, and its nil must not reach the checker as a node.
func isNilNode(n any) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// parseStatement parses one statement, or returns a true nil when there is
// none (an empty statement, or one that failed to parse and was reported).
func (p *Parser) parseStatement() ast.Statement {
	stmt := p.parseStatementNode()
	if isNilNode(stmt) {
		return nil
	}
	return stmt
}

func (p *Parser) parseStatementNode() ast.Statement {
	// goto name and ::name::
	if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "goto" && p.peekTokenIsIdentOrContextual() {
		stmt := &ast.GotoStatement{Token: p.curToken}
		p.nextToken()
		stmt.Label = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		return stmt
	}
	if p.curTokenIs(lexer.COLON) && p.peekTokenIs(lexer.COLON) {
		return p.parseLabelStatement()
	}

	// A word like class, import or const begins a declaration only when
	// what follows fits one; otherwise it is a name, as in Lua.
	if lexer.IsSoftKeyword(p.curToken.Type) && !p.startsDeclaration() {
		return p.parseExpressionStatement()
	}
	switch p.curToken.Type {
	case lexer.SEMICOLON:
		// Lua's empty statement.
		return nil
	case lexer.FUNCTION:
		return p.parseFunctionDeclaration()
	case lexer.ASYNC:
		return p.parseAsyncFunctionDeclaration()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.LOCAL, lexer.CONST:
		// `local function f() ... end` is a function declaration, not a variable.
		if p.curTokenIs(lexer.LOCAL) && p.peekTokenIs(lexer.FUNCTION) {
			p.nextToken() // move onto 'function'
			fn := p.parseFunctionDeclaration()
			if fn != nil {
				fn.IsLocal = true
			}
			return fn
		}
		return p.parseVariableDeclaration()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.REPEAT:
		return p.parseRepeatStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.DO:
		return p.parseDoStatement()
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.ABSTRACT:
		// abstract class declaration
		return p.parseAbstractClassDeclaration()
	case lexer.CLASS:
		return p.parseClassDeclaration()
	case lexer.INTERFACE:
		return p.parseInterfaceDeclaration()
	case lexer.ENUM:
		return p.parseEnumDeclaration()
	case lexer.TYPE:
		return p.parseTypeDeclaration()
	case lexer.NAMESPACE:
		return p.parseNamespaceDeclaration()
	case lexer.EXPORT:
		return p.parseExportStatement()
	case lexer.IMPORT:
		return p.parseImportStatement()
	case lexer.DECLARE:
		return p.parseDeclareStatement()
	case lexer.AT:
		return p.parseDecoratedStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.curToken}

	p.nextToken() // move to condition

	// Parse condition
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect 'then'
	if !p.expectPeek(lexer.THEN) {
		return nil
	}

	// Parse consequence block (stops at 'else', 'elseif' or 'end')
	stmt.Consequence = p.parseIfBlockStatement()

	// Check for elseif (treat as nested if in alternative)
	if p.curTokenIs(lexer.ELSEIF) {
		// Recursively parse elseif as a new if statement
		elseIfStmt := p.parseIfStatement()
		stmt.Alternative = &ast.BlockStatement{
			Token:      p.curToken,
			Statements: []ast.Statement{elseIfStmt},
		}
	} else if p.curTokenIs(lexer.ELSE) {
		// Check for else
		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseIfBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken()

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.ELSE) && !p.curTokenIs(lexer.ELSEIF) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if !isNilNode(stmt) {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	p.unclosedAtEOF("end", block.Token)

	return block
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	p.nextToken() // move to condition

	// Parse condition
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect 'do'
	if !p.expectPeek(lexer.DO) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseRepeatStatement() *ast.RepeatStatement {
	stmt := &ast.RepeatStatement{Token: p.curToken}

	// The body runs up to 'until' rather than 'end'.
	stmt.Body = p.parseBlockStatementUntil(lexer.UNTIL)

	if !p.curTokenIs(lexer.UNTIL) {
		p.addError(fmt.Sprintf("expected 'until' to close repeat, got %s", p.curToken.Type), p.curToken)
		return nil
	}

	p.nextToken() // move onto the condition
	stmt.Condition = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}

	// Parse loop variables (can be multiple for generic for loops)
	stmt.Variables = []*ast.Identifier{}

	// Expect first variable name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	stmt.Variables = append(stmt.Variables, p.parseIdentifierOrContextual())

	// Parse additional variables if comma-separated
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // consume comma
		if !p.expectPeekIdentOrContextual() {
			return nil
		}
		stmt.Variables = append(stmt.Variables, p.parseIdentifierOrContextual())
	}

	// Check if it's a generic for (for...in) or numeric for (for...=)
	if p.peekTokenIs(lexer.IN) {
		stmt.IsGeneric = true
		p.nextToken() // consume 'in'
		p.nextToken() // move to iterator expression

		stmt.Iterator = p.parseExpression(LOWEST)
		// for k, v in next, t do: the iterator, its state and control value
		// (and, in Lua 5.4, a closing value) may be listed.
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			stmt.IteratorRest = append(stmt.IteratorRest, p.parseExpression(LOWEST))
		}
	} else if p.peekTokenIs(lexer.ASSIGN) {
		stmt.IsGeneric = false
		p.nextToken() // consume '='
		p.nextToken() // move to start expression

		// Parse start value
		stmt.Start = p.parseExpression(LOWEST)

		// Expect comma
		if !p.expectPeek(lexer.COMMA) {
			return nil
		}

		p.nextToken() // move to end expression
		stmt.End = p.parseExpression(LOWEST)

		// Optional step value
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to step expression
			stmt.Step = p.parseExpression(LOWEST)
		}
	} else {
		msg := fmt.Sprintf("expected 'in' or '=' after for variable, got %s", p.peekToken.Type)
		p.addError(msg, p.peekToken)
		return nil
	}

	// Expect 'do'
	if !p.expectPeek(lexer.DO) {
		return nil
	}

	// Parse body
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseDoStatement() *ast.DoStatement {
	stmt := &ast.DoStatement{Token: p.curToken}

	// Parse body
	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseTypeName reads a type's name, qualified by the namespaces or
// modules it is reached through: Pos, Geo.Pos, m.Shapes.Circle.
func (p *Parser) parseTypeName() ast.Expression {
	var name ast.Expression = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	for p.peekTokenIs(lexer.DOT) {
		state := p.SaveState()
		p.nextToken()
		dot := p.curToken
		if !lexer.CanBeFieldName(p.peekToken) {
			p.RestoreState(state)
			break
		}
		p.nextToken()
		name = &ast.DotExpression{Token: dot, Left: name, Right: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
	}
	return name
}

// parseLabelStatement parses ::name::, starting on the first ':'.
func (p *Parser) parseLabelStatement() ast.Statement {
	stmt := &ast.LabelStatement{Token: p.curToken}
	p.nextToken() // the second ':'
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(lexer.COLON) || !p.expectPeek(lexer.COLON) {
		return nil
	}
	return stmt
}

// parseAttrib reads Lua 5.4's <const> or <close> after a local's name, if
// there is one, and returns it ("" if none).
func (p *Parser) parseAttrib() string {
	if !p.peekTokenIs(lexer.LT) {
		return ""
	}
	state := p.SaveState()
	p.nextToken() // '<'
	if !p.peekTokenIsIdentOrContextual() {
		p.RestoreState(state)
		return ""
	}
	p.nextToken() // the attribute
	attrib := p.curToken.Literal
	if !p.expectPeek(lexer.GT) {
		return attrib
	}
	return attrib
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	return &ast.BreakStatement{Token: p.curToken}
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	defer p.clearStopAtPipe()()
	exp := &ast.IndexExpression{
		Token: p.curToken,
		Left:  left,
	}

	p.nextToken() // move past '['
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseTableLiteral() ast.Expression {
	defer p.clearStopAtPipe()()
	table := &ast.TableLiteral{
		Token:  p.curToken,
		Pairs:  make(map[ast.Expression]ast.Expression),
		Values: []ast.Expression{},
	}

	// Empty table
	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return table
	}

	p.nextToken() // move past '{'

	var lastOffset1 int
	var lastType1 lexer.TokenType
	guardStarted1 := false
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset1 && guardType == lastType1 && guardStarted1 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted1, lastOffset1, lastType1 = true, p.curToken.Offset, p.curToken.Type
		// Try to parse as key-value pair first - allows context-aware keywords
		// Check for [key] = value syntax
		if p.curTokenIs(lexer.LBRACKET) {
			open := p.curToken
			p.nextToken() // consume '['
			key := p.parseExpression(LOWEST)
			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}
			// [KEY] = v is the value of KEY, not the field "KEY": only a
			// literal key in brackets names a field directly.
			switch key.(type) {
			case *ast.StringLiteral, *ast.NumberLiteral, *ast.BooleanLiteral:
			default:
				key = &ast.ParenExpression{Token: open, Expression: key}
			}
			if !p.expectPeek(lexer.ASSIGN) {
				return nil
			}
			p.nextToken() // move past '='
			value := p.parseExpression(LOWEST)
			table.Pairs[key] = value
			table.Entries = append(table.Entries, ast.TableEntry{Key: key, Value: value})
		} else if p.curTokenIsIdentOrContextual() && p.peekTokenIs(lexer.ASSIGN) {
			// Key-value pair with identifier key
			key := p.parseIdentifierOrContextual()
			p.nextToken() // consume identifier
			p.nextToken() // consume '='

			value := p.parseExpression(LOWEST)
			table.Pairs[key] = value
			table.Entries = append(table.Entries, ast.TableEntry{Key: key, Value: value})
		} else {
			// Array-style value
			value := p.parseExpression(LOWEST)
			table.Values = append(table.Values, value)
			table.Entries = append(table.Entries, ast.TableEntry{Value: value})
		}

		// Fields are separated by ',' or ';', as in Lua
		if !p.peekTokenIs(lexer.RBRACE) {
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			} else if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken() // move past the separator
		} else {
			p.nextToken() // move to '}'
		}
	}
	p.unclosedAtEOF("}", table.Token)

	return table
}

func (p *Parser) parseAbstractClassDeclaration() *ast.ClassDeclaration {
	// Consume 'abstract' keyword
	p.nextToken()

	// Expect 'class' keyword
	if !p.curTokenIs(lexer.CLASS) {
		p.addError(fmt.Sprintf("expected 'class' after 'abstract', got %s", p.curToken.Type), p.curToken)
		return nil
	}

	// Parse class declaration normally
	class := p.parseClassDeclaration()
	if class != nil {
		class.IsAbstract = true
	}
	return class
}

func (p *Parser) parseClassDeclaration() *ast.ClassDeclaration {
	p.classDepth++
	defer func() { p.classDepth-- }()
	class := &ast.ClassDeclaration{
		Token:      p.curToken,
		Properties: []*ast.PropertyDeclaration{},
		Methods:    []*ast.FunctionDeclaration{},
		IsAmbient:  p.ambient,
	}

	// Parse class name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	class.Name = p.parseIdentifierOrContextual()

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		class.GenericParams = p.parseGenericParameters()
	}

	// Parse extends clause (single inheritance)
	if p.peekTokenIs(lexer.EXTENDS) {
		p.nextToken() // consume 'extends'
		p.nextToken() // move to parent class name
		var parent ast.Expression = &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		// extends Geo.Base: a class reached through a namespace or module
		for p.peekTokenIs(lexer.DOT) {
			p.nextToken()
			dot := p.curToken
			if !p.expectPeekFieldName() {
				break
			}
			parent = &ast.DotExpression{Token: dot, Left: parent, Right: p.parseFieldName()}
		}
		class.Extends = parent
	}

	// Parse implements clause
	if p.peekTokenIs(lexer.IMPLEMENTS) {
		p.nextToken() // consume 'implements'
		p.nextToken() // move to first interface

		class.Implements = append(class.Implements, &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		})

		// Multiple interfaces
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next interface
			class.Implements = append(class.Implements, &ast.Identifier{
				Token: p.curToken,
				Value: p.curToken.Literal,
			})
		}
	}

	p.nextToken() // move past class header

	// Parse class body
	var lastOffset2 int
	var lastType2 lexer.TokenType
	guardStarted2 := false
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset2 && guardType == lastType2 && guardStarted2 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted2, lastOffset2, lastType2 = true, p.curToken.Offset, p.curToken.Type
		// Check for index signature first (before modifiers)
		if p.curTokenIs(lexer.LBRACKET) {
			// Index signature: [key: KeyType]: ValueType
			indexSig := p.parseIndexSignature()
			if indexSig != nil {
				class.IndexSignature = indexSig
			}
			p.nextToken() // move past index signature
			continue
		}

		// Decorators on a method: @logged before it.
		var memberDecorators []*ast.Decorator
		for p.curTokenIs(lexer.AT) {
			if d := p.parseDecorator(); d != nil {
				memberDecorators = append(memberDecorators, d)
			}
			p.nextToken()
		}
		methodsBefore := len(class.Methods)

		// Track modifiers for current member
		isStatic := false
		isAbstract := false
		isReadonly := false
		visibility := "public" // default visibility

		// Parse modifiers. A modifier word followed by ':', '=', '?', '(' or
		// '<' is the member's own name instead: static: boolean, public().
		for (p.curToken.Type == lexer.PUBLIC || p.curToken.Type == lexer.PRIVATE || p.curToken.Type == lexer.PROTECTED ||
			p.curToken.Type == lexer.STATIC || p.curToken.Type == lexer.ABSTRACT || p.curToken.Type == lexer.READONLY) &&
			!p.peekStartsMember() {
			switch p.curToken.Type {
			case lexer.PUBLIC, lexer.PRIVATE, lexer.PROTECTED:
				visibility = p.curToken.Literal
			case lexer.STATIC:
				isStatic = true
			case lexer.ABSTRACT:
				isAbstract = true
			case lexer.READONLY:
				isReadonly = true
			}
			p.nextToken()
		}

		// Now parse the actual member. Any word of Lunar's own names a member
		// when a member follows it (from: number, match(x)); get, set and
		// constructor keep their meaning, as before.
		memberType := p.curToken.Type
		if lexer.IsSoftKeyword(memberType) && memberType != lexer.CONSTRUCTOR && memberType != lexer.GET &&
			memberType != lexer.SET && p.peekStartsMember() {
			memberType = lexer.IDENT
		}
		switch memberType {
		case lexer.CONSTRUCTOR:
			class.Constructor = p.parseConstructorDeclaration()
			p.nextToken()

		case lexer.GET:
			// Check if it's a getter declaration (get name()) or a method named "get"
			if p.peekTokenIs(lexer.LPAREN) {
				// It's a method named "get"
				method := p.parseMethodDeclaration(isAbstract)
				method.IsStatic = isStatic
				method.IsAbstract = isAbstract
				method.Visibility = visibility
				class.Methods = append(class.Methods, method)
				// Advance past the method's own 'end'. An empty body still has
				// one, so testing for statements left the parser sitting on it
				// and ended the class early.
				if method.Body != nil {
					p.nextToken()
				}
			} else {
				// It's a getter declaration
				getter := p.parseGetterDeclaration()
				if getter != nil {
					getter.Visibility = visibility
					class.Getters = append(class.Getters, getter)
					p.nextToken() // Advance past 'end'
				}
			}

		case lexer.SET:
			// Check if it's a setter declaration (set name(param)) or a method named "set"
			if p.peekTokenIs(lexer.LPAREN) {
				// It's a method named "set"
				method := p.parseMethodDeclaration(isAbstract)
				method.IsStatic = isStatic
				method.IsAbstract = isAbstract
				method.Visibility = visibility
				class.Methods = append(class.Methods, method)
				// Advance past the method's own 'end'. An empty body still has
				// one, so testing for statements left the parser sitting on it
				// and ended the class early.
				if method.Body != nil {
					p.nextToken()
				}
			} else {
				// It's a setter declaration
				setter := p.parseSetterDeclaration()
				if setter != nil {
					setter.Visibility = visibility
					class.Setters = append(class.Setters, setter)
					p.nextToken() // Advance past 'end'
				}
			}

		case lexer.IDENT, lexer.STRING_TYPE, lexer.TABLE, lexer.TYPE:
			// Could be property or method
			if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.ASSIGN) || p.peekTokenIs(lexer.QUESTION) {
				// It's a property (with type annotation or initial value)
				prop := p.parsePropertyDeclaration()
				prop.Visibility = visibility
				prop.IsStatic = isStatic
				prop.IsReadonly = isReadonly
				class.Properties = append(class.Properties, prop)
			} else if p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.LT) {
				// It's a method (generic if '<' follows its name)
				method := p.parseMethodDeclaration(isAbstract)
				method.IsStatic = isStatic
				method.IsAbstract = isAbstract
				method.Visibility = visibility
				class.Methods = append(class.Methods, method)
				// Advance past the method's own 'end'. An empty body still has
				// one, so testing for statements left the parser sitting on it
				// and ended the class early.
				if method.Body != nil {
					p.nextToken()
				}
			} else {
				p.nextToken()
			}

		default:
			p.nextToken()
		}
		if len(memberDecorators) > 0 {
			if len(class.Methods) > methodsBefore {
				class.Methods[len(class.Methods)-1].Decorators = memberDecorators
			} else {
				p.addError("decorators apply to functions, classes and methods", memberDecorators[0].Token)
			}
		}
	}
	p.unclosedAtEOF("end", class.Token)

	return class
}

func (p *Parser) parsePropertyDeclaration() *ast.PropertyDeclaration {
	prop := &ast.PropertyDeclaration{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	// `name?: T` marks the field optional.
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken()
		prop.IsOptional = true
	}

	// Check for type annotation (: type)
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // move to type
		prop.Type = p.parseType()
	}

	// Check for initial value (= value)
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume '='
		p.nextToken() // move to value
		prop.Value = p.parseExpression(LOWEST)
	}

	p.nextToken() // move past property
	return prop
}

func (p *Parser) parseMethodDeclaration(isAbstract bool) *ast.FunctionDeclaration {
	method := &ast.FunctionDeclaration{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	// A generic method: pass<U>(x: U): U
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		method.GenericParams = p.parseGenericParameters()
	}

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	method.Parameters = p.parseFunctionParameters()

	// Parse return type
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // move to return type
		method.ReturnType = p.parseType()
	}

	method.IsAbstract = isAbstract

	// An abstract method declares a signature and stops there; parsing a body
	// would swallow the members that follow it, up to the class's own 'end'.
	// A body written anyway is still consumed, so the checker can report it
	// rather than the parser choking on the leftovers.
	if p.ambient || (isAbstract && !p.abstractMethodHasBody()) {
		return method
	}

	method.Body = p.parseBlockStatement()

	return method
}

// abstractMethodHasBody reports whether an abstract method signature is
// followed by statements rather than by the next member or the class's end.
func (p *Parser) abstractMethodHasBody() bool {
	switch p.peekToken.Type {
	case lexer.END, lexer.EOF, lexer.CONSTRUCTOR, lexer.PUBLIC, lexer.PRIVATE,
		lexer.PROTECTED, lexer.STATIC, lexer.ABSTRACT, lexer.READONLY, lexer.GET, lexer.SET:
		return false
	}

	if !lexer.CanBeFieldName(p.peekToken) {
		// A keyword Lua reserves starts a statement, so this is a body.
		return true
	}

	// A name followed by '(' or ':' is the next method or property.
	state := p.SaveState()
	defer p.RestoreState(state)

	p.nextToken()

	return !p.peekTokenIs(lexer.LPAREN) && !p.peekTokenIs(lexer.COLON)
}

func (p *Parser) parseConstructorDeclaration() *ast.ConstructorDeclaration {
	constructor := &ast.ConstructorDeclaration{
		Token: p.curToken,
	}

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	constructor.Parameters = p.parseFunctionParameters()

	if p.ambient {
		return constructor
	}

	// Parse body
	constructor.Body = p.parseBlockStatement()

	return constructor
}

func (p *Parser) parseGetterDeclaration() *ast.GetterDeclaration {
	getter := &ast.GetterDeclaration{
		Token: p.curToken, // 'get' token
	}

	// Parse property name
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	getter.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Expect empty parentheses
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Parse return type
	if !p.expectPeek(lexer.COLON) {
		return nil
	}
	p.nextToken() // move to return type
	getter.ReturnType = p.parseType()

	if p.ambient {
		return getter
	}

	// Parse body
	getter.Body = p.parseBlockStatement()

	return getter
}

func (p *Parser) parseSetterDeclaration() *ast.SetterDeclaration {
	setter := &ast.SetterDeclaration{
		Token: p.curToken, // 'set' token
	}

	// Parse property name
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	setter.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse parameter
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	if !p.expectPeekIdentOrContextual() {
		return nil
	}

	paramName := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	var paramType ast.Expression

	// Parse parameter type if present
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // move to type
		paramType = p.parseType()
	}

	setter.Parameter = &ast.Parameter{
		Token: paramName.Token,
		Name:  paramName,
		Type:  paramType,
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	// Parse body
	if p.ambient {
		return setter
	}

	setter.Body = p.parseBlockStatement()

	return setter
}

func (p *Parser) parseInterfaceDeclaration() *ast.InterfaceDeclaration {
	iface := &ast.InterfaceDeclaration{
		Token:      p.curToken,
		Methods:    []*ast.InterfaceMethod{},
		Properties: []*ast.PropertyDeclaration{},
	}

	// Parse interface name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	iface.Name = p.parseIdentifierOrContextual()

	// Type parameters: interface Box<T>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		iface.GenericParams = p.parseGenericParameters()
	}

	// Parse extends clause
	if p.peekTokenIs(lexer.EXTENDS) {
		p.nextToken() // consume 'extends'
		p.nextToken() // move to first parent

		iface.Extends = append(iface.Extends, &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		})

		// Multiple parents
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next parent
			iface.Extends = append(iface.Extends, &ast.Identifier{
				Token: p.curToken,
				Value: p.curToken.Literal,
			})
		}
	}

	p.nextToken() // move past interface header

	// Parse interface body - allows context-aware keywords
	var lastOffset3 int
	var lastType3 lexer.TokenType
	guardStarted3 := false
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset3 && guardType == lastType3 && guardStarted3 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted3, lastOffset3, lastType3 = true, p.curToken.Offset, p.curToken.Type
		if p.curTokenIs(lexer.LBRACKET) {
			// Index signature: [key: KeyType]: ValueType
			indexSig := p.parseIndexSignature()
			if indexSig != nil {
				iface.IndexSignature = indexSig
			}
			p.nextToken() // move past index signature
		} else if p.curTokenIs(lexer.READONLY) && p.peekTokenIsIdentOrContextual() {
			// readonly name: T
			p.nextToken()
			prop := p.parsePropertyDeclaration()
			if prop != nil {
				prop.IsReadonly = true
				iface.Properties = append(iface.Properties, prop)
			}
		} else if p.curTokenIs(lexer.STATIC) {
			// Handle static keyword
			p.nextToken() // move past 'static'
			if !p.curTokenIsIdentOrContextual() {
				p.addError("expected a member after 'static'", p.curToken)
			}
			if p.curTokenIsIdentOrContextual() {
				if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.QUESTION) {
					// Static property
					prop := p.parsePropertyDeclaration()
					prop.IsStatic = true
					iface.Properties = append(iface.Properties, prop)
				} else if p.peekTokenIs(lexer.LPAREN) {
					// Static method signature
					method := p.parseInterfaceMethod()
					method.IsStatic = true
					iface.Methods = append(iface.Methods, method)
				} else {
					p.addError(fmt.Sprintf("unexpected '%s' in an interface", p.curToken.Literal), p.curToken)
					p.nextToken()
				}
			}
		} else if p.curTokenIsIdentOrContextual() {
			if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.QUESTION) {
				// Property
				prop := p.parsePropertyDeclaration()
				iface.Properties = append(iface.Properties, prop)
			} else if p.peekTokenIs(lexer.LPAREN) || p.peekTokenIs(lexer.LT) {
				// Method signature (generic if '<' follows its name)
				method := p.parseInterfaceMethod()
				iface.Methods = append(iface.Methods, method)
			} else {
				p.addError(fmt.Sprintf("unexpected '%s' in an interface", p.curToken.Literal), p.curToken)
				p.nextToken()
			}
		} else {
			p.addError(fmt.Sprintf("unexpected '%s' in an interface", p.curToken.Literal), p.curToken)
			p.nextToken()
		}
	}
	p.unclosedAtEOF("end", iface.Token)

	return iface
}

func (p *Parser) parseInterfaceMethod() *ast.InterfaceMethod {
	method := &ast.InterfaceMethod{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	// A generic method: pass<U>(x: U): U
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		method.GenericParams = p.parseGenericParameters()
	}

	// Parse parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	method.Parameters = p.parseFunctionParameters()

	// Parse return type
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume ':'
		p.nextToken() // move to return type
		method.ReturnType = p.parseType()
	}

	p.nextToken() // move past method signature
	return method
}

func (p *Parser) parseIndexSignature() *ast.IndexSignatureDeclaration {
	indexSig := &ast.IndexSignatureDeclaration{
		Token: p.curToken, // '['
	}

	p.nextToken() // move to key name

	if !p.curTokenIsIdentOrContextual() {
		msg := fmt.Sprintf("expected identifier for index signature key name, got %s instead", p.curToken.Type)
		p.addError(msg, p.curToken)
		return nil
	}

	indexSig.KeyName = &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // move to key type
	indexSig.KeyType = p.parseType()

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // move to value type
	indexSig.ValueType = p.parseType()

	// Don't call nextToken() here - let the caller decide when to advance
	// p.nextToken() // move past index signature
	return indexSig
}

func (p *Parser) parseEnumDeclaration() *ast.EnumDeclaration {
	enum := &ast.EnumDeclaration{
		Token:   p.curToken,
		Members: []*ast.EnumMember{},
	}

	// Parse enum name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	enum.Name = p.parseIdentifierOrContextual()

	p.nextToken() // move past enum name

	// Parse enum members - allows context-aware keywords
	var lastOffset4 int
	var lastType4 lexer.TokenType
	guardStarted4 := false
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset4 && guardType == lastType4 && guardStarted4 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted4, lastOffset4, lastType4 = true, p.curToken.Offset, p.curToken.Type
		if p.curTokenIsIdentOrContextual() {
			member := &ast.EnumMember{
				Token: p.curToken,
				Name:  p.parseIdentifierOrContextual(),
			}

			// Check for value assignment
			if p.peekTokenIs(lexer.ASSIGN) {
				p.nextToken() // consume '='
				p.nextToken() // move to value
				member.Value = p.parseExpression(LOWEST)
			}

			enum.Members = append(enum.Members, member)
		}

		p.nextToken()
	}
	p.unclosedAtEOF("end", enum.Token)

	return enum
}

func (p *Parser) parseTypeDeclaration() *ast.TypeDeclaration {
	typeDecl := &ast.TypeDeclaration{
		Token: p.curToken,
	}

	// Parse type name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	typeDecl.Name = p.parseIdentifierOrContextual()

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		typeDecl.GenericParams = p.parseGenericParameters()
	}

	p.nextToken() // move past name (or generic params)

	// Check if it's an object shape declaration (type Name ... end) or alias (type Name = Type)
	if p.curTokenIs(lexer.ASSIGN) {
		// Type alias: type Name = Type
		p.nextToken() // move to type definition
		typeDecl.Type = p.parseType()
	} else {
		// Object shape: type Name ... end
		// Parse properties similar to interface (no braces needed)
		var lastOffset5 int
		var lastType5 lexer.TokenType
		guardStarted5 := false
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			// An iteration that consumed nothing (a half-typed member the
			// branch below could not parse) would run forever: skip the token.
			if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset5 && guardType == lastType5 && guardStarted5 {
				p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
				p.nextToken()
				continue
			}
			guardStarted5, lastOffset5, lastType5 = true, p.curToken.Offset, p.curToken.Type
			if p.curTokenIsIdentOrContextual() {
				prop := &ast.PropertyDeclaration{
					Token: p.curToken,
					Name:  p.parseIdentifierOrContextual(),
				}

				if p.peekTokenIs(lexer.QUESTION) {
					p.nextToken()
					prop.IsOptional = true
				}

				if !p.expectPeek(lexer.COLON) {
					return nil
				}

				p.nextToken() // move to type
				prop.Type = p.parseType()
				typeDecl.Properties = append(typeDecl.Properties, prop)
			}
			p.nextToken()
		}
		p.unclosedAtEOF("end", typeDecl.Token)
	}

	return typeDecl
}

func (p *Parser) parseExportStatement() *ast.ExportStatement {
	exportStmt := &ast.ExportStatement{
		Token: p.curToken,
	}

	p.nextToken() // move past 'export'

	// Parse the statement being exported
	exportStmt.Statement = p.parseStatement()

	return exportStmt
}

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	importStmt := &ast.ImportStatement{
		Token: p.curToken,
	}

	p.nextToken() // move past 'import'

	// Check for wildcard import (import * from "module")
	if p.curTokenIs(lexer.ASTERISK) {
		// import * as name from "module": the module's exports through a name.
		importStmt.IsWildcard = true
		if !p.peekTokenIs(lexer.AS) {
			p.addError("import * needs a name: import * as name from \"...\"", p.curToken)
			return nil
		}
		p.nextToken() // move to 'as'
		if !p.expectPeekIdentOrContextual() {
			return nil
		}
		importStmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken() // move past the name
	} else if p.curTokenIs(lexer.LBRACE) {
		// Named imports: import { name1, name2 } from "module"
		p.nextToken() // move past '{'

		var lastOffset6 int
		var lastType6 lexer.TokenType
		guardStarted6 := false
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			// An iteration that consumed nothing (a half-typed member the
			// branch below could not parse) would run forever: skip the token.
			if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset6 && guardType == lastType6 && guardStarted6 {
				p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
				p.nextToken()
				continue
			}
			guardStarted6, lastOffset6, lastType6 = true, p.curToken.Offset, p.curToken.Type
			if !p.curTokenIs(lexer.IDENT) {
				p.peekError(lexer.IDENT)
				return nil
			}

			importStmt.Names = append(importStmt.Names, &ast.Identifier{
				Token: p.curToken,
				Value: p.curToken.Literal,
			})

			p.nextToken()

			if p.curTokenIs(lexer.COMMA) {
				p.nextToken() // move past comma
			}
		}
		p.unclosedAtEOF("}", importStmt.Token)

		if !p.curTokenIs(lexer.RBRACE) {
			p.addError("expected '}' after import names", p.curToken)
			return nil
		}

		p.nextToken() // move past '}'
	}

	// Expect 'from' keyword
	if !p.curTokenIs(lexer.FROM) {
		p.addError("expected 'from' after import statement", p.curToken)
		return nil
	}

	p.nextToken() // move past 'from'

	// Expect string literal for module path
	if !p.curTokenIs(lexer.STRING) {
		p.addError("expected string literal for module path", p.curToken)
		return nil
	}

	importStmt.Module = p.curToken.Literal

	return importStmt
}

// parseNamespaceDeclaration parses a namespace declaration
func (p *Parser) parseNamespaceDeclaration() *ast.NamespaceDeclaration {
	ns := &ast.NamespaceDeclaration{
		Token: p.curToken, // 'namespace' token
	}

	// Parse namespace name
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	ns.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // move past name

	// Parse namespace body until 'end'
	var lastOffset7 int
	var lastType7 lexer.TokenType
	guardStarted7 := false
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset7 && guardType == lastType7 && guardStarted7 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted7, lastOffset7, lastType7 = true, p.curToken.Offset, p.curToken.Type
		stmt := p.parseStatement()
		if !isNilNode(stmt) {
			ns.Statements = append(ns.Statements, stmt)
		}
		p.nextToken()
	}
	p.unclosedAtEOF("end", ns.Token)

	return ns
}

// parseAmbient parses one declaration with bodies switched off.
func (p *Parser) parseAmbient(parse func() ast.Statement) ast.Statement {
	saved := p.ambient
	p.ambient = true
	defer func() { p.ambient = saved }()

	stmt := parse()
	// A nil *ClassDeclaration in an interface is not a nil Statement.
	switch d := stmt.(type) {
	case *ast.ClassDeclaration:
		if d == nil {
			return nil
		}
	case *ast.NamespaceDeclaration:
		if d == nil {
			return nil
		}
	}
	return stmt
}

// parseDeclareStatement parses ambient declarations like: declare const name: Type
func (p *Parser) parseDeclareStatement() *ast.DeclareStatement {
	declareStmt := &ast.DeclareStatement{
		Token: p.curToken,
	}

	p.nextToken() // move past 'declare'

	// Parse the underlying declaration (const, function, class, interface, etc.)
	switch p.curToken.Type {
	case lexer.CONST, lexer.LOCAL:
		declareStmt.Declaration = p.parseVariableDeclaration()
	case lexer.FUNCTION:
		// A declaration has no body. The trailing 'end' that older
		// declaration files write is accepted but not required, so leaving
		// it out can no longer swallow the declarations that follow.
		fd := p.parseFunctionSignature()
		if fd == nil {
			return nil
		}
		p.bodilessDeclare = fd
		if p.peekTokenIs(lexer.END) {
			p.nextToken()
			p.bodilessDeclare = nil
		}
		declareStmt.Declaration = fd
	case lexer.CLASS:
		declareStmt.Declaration = p.parseAmbient(func() ast.Statement { return p.parseClassDeclaration() })
	case lexer.NAMESPACE:
		declareStmt.Declaration = p.parseAmbient(func() ast.Statement { return p.parseNamespaceDeclaration() })
	case lexer.INTERFACE:
		declareStmt.Declaration = p.parseInterfaceDeclaration()
	case lexer.ENUM:
		declareStmt.Declaration = p.parseEnumDeclaration()
	case lexer.TYPE:
		declareStmt.Declaration = p.parseTypeDeclaration()
	default:
		p.addError(fmt.Sprintf("expected declaration after 'declare', got %s", p.curToken.Type), p.curToken)
		return nil
	}

	// A declaration that did not parse leaves nothing to declare.
	if isNilNode(declareStmt.Declaration) {
		return nil
	}
	return declareStmt
}

// parseGenericParameters parses generic type parameters: <T, U, V>
func (p *Parser) parseGenericParameters() []*ast.Identifier {
	params := []*ast.Identifier{}

	p.nextToken() // move past '<' to first parameter

	for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIsIdentOrContextual() {
			p.peekError(lexer.IDENT)
			return nil
		}

		params = append(params, p.parseIdentifierOrContextual())

		p.nextToken()

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // move past comma to next parameter
		} else if !p.curTokenIs(lexer.GT) {
			if p.curTokenIs(lexer.EXTENDS) {
				p.addError("constraints on type parameters (T extends X) are not supported; write <T>", p.curToken)
			} else {
				p.addError(fmt.Sprintf("expected ',' or '>' after a type parameter, got '%s'", p.curToken.Literal), p.curToken)
			}
			return nil
		}
	}

	if !p.curTokenIs(lexer.GT) {
		p.addError("expected '>' after generic parameters", p.curToken)
		return nil
	}

	return params
}

// parseDecorator parses a single decorator like @name or @name(args)
func (p *Parser) parseDecorator() *ast.Decorator {
	decorator := &ast.Decorator{
		Token: p.curToken, // '@' token
	}

	// Expect identifier after @
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	decorator.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for arguments
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // move to '('
		decorator.Arguments = p.parseExpressionList(lexer.RPAREN)
	}

	return decorator
}

// parseDecoratedStatement parses decorators followed by a class or function
func (p *Parser) parseDecoratedStatement() ast.Statement {
	var decorators []*ast.Decorator

	// Parse all decorators
	for p.curTokenIs(lexer.AT) {
		decorator := p.parseDecorator()
		if decorator != nil {
			decorators = append(decorators, decorator)
		}
		p.nextToken()
	}

	// Now parse the decorated statement
	switch p.curToken.Type {
	case lexer.CLASS:
		class := p.parseClassDeclaration()
		if class != nil {
			class.Decorators = decorators
		}
		return class
	case lexer.ABSTRACT:
		class := p.parseAbstractClassDeclaration()
		if class != nil {
			class.Decorators = decorators
		}
		return class
	case lexer.FUNCTION:
		fn := p.parseFunctionDeclaration()
		if fn != nil {
			fn.Decorators = decorators
		}
		return fn
	case lexer.ASYNC:
		fn := p.parseAsyncFunctionDeclaration()
		if fn != nil {
			fn.Decorators = decorators
		}
		return fn
	default:
		p.addError(fmt.Sprintf("decorators can only be applied to classes and functions, got %s", p.curToken.Type), p.curToken)
		return nil
	}
}

// ============================================
// Pattern Matching Parser
// ============================================

// parseMatchExpression parses a match expression
// Syntax: match value | pattern -> expression | pattern -> expression end
func (p *Parser) parseMatchExpression() ast.Expression {
	matchExpr := &ast.MatchExpression{
		Token: p.curToken, // 'match' token
		Cases: []ast.MatchCase{},
	}

	p.nextToken() // move to value expression

	// Parse the value being matched
	matchExpr.Value = p.parseExpression(LOWEST)

	if matchExpr.Value == nil {
		p.addError("expected expression after 'match'", p.curToken)
		return nil
	}

	// Expect 'with' keyword (check peek and advance)
	if !p.expectPeek(lexer.WITH) {
		return nil
	}
	p.matchWith = true

	p.nextToken() // move past 'with' to first case

	// After advancing, curToken should be the first '|' or 'end'

	// Parse match cases
	// Each case starts with |
	var lastOffset8 int
	var lastType8 lexer.TokenType
	guardStarted8 := false
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		// An iteration that consumed nothing (a half-typed member the
		// branch below could not parse) would run forever: skip the token.
		if guardOffset, guardType := p.curToken.Offset, p.curToken.Type; guardOffset == lastOffset8 && guardType == lastType8 && guardStarted8 {
			p.addError(fmt.Sprintf("unexpected '%s' here", p.curToken.Literal), p.curToken)
			p.nextToken()
			continue
		}
		guardStarted8, lastOffset8, lastType8 = true, p.curToken.Offset, p.curToken.Type
		// Expect | for case
		if !p.curTokenIs(lexer.PIPE) {
			p.addError(fmt.Sprintf("expected '|' or 'end' in match expression, got %s", p.curToken.Type), p.curToken)
			return nil
		}

		matchCase := p.parseMatchCase()
		if matchCase != nil {
			matchExpr.Cases = append(matchExpr.Cases, *matchCase)
		}
		// parseMatchCase leaves us on the last token of the body expression
		// Move to the next token (either next '|' or 'end')
		p.nextToken()
	}
	p.unclosedAtEOF("end", matchExpr.Token)

	if len(matchExpr.Cases) == 0 {
		p.addError("match expression must have at least one case", matchExpr.Token)
		return nil
	}

	return matchExpr
}

// parseMatchCase parses a single match case
// Syntax: | pattern [when condition] -> expression
func (p *Parser) parseMatchCase() *ast.MatchCase {
	matchCase := &ast.MatchCase{
		Token: p.curToken, // '|' token
	}

	p.nextToken() // move to pattern

	// Parse the pattern
	matchCase.Pattern = p.parsePattern()
	if matchCase.Pattern == nil {
		return nil
	}

	p.nextToken() // move past pattern

	// | 1 | 2 -> ...: before the '->', a '|' starts another alternative.
	if p.curTokenIs(lexer.PIPE) {
		or := &ast.OrPattern{Token: matchCase.Token, Alternatives: []ast.Pattern{matchCase.Pattern}}
		for p.curTokenIs(lexer.PIPE) {
			p.nextToken()
			alt := p.parsePattern()
			if alt == nil {
				return nil
			}
			if patternBinds(alt) || patternBinds(or.Alternatives[0]) {
				p.addError("an alternative of '|' patterns cannot bind a name: each would bind it differently", p.curToken)
				return nil
			}
			or.Alternatives = append(or.Alternatives, alt)
			p.nextToken()
		}
		matchCase.Pattern = or
	}

	// Check for optional guard (when clause)
	hasGuard := false
	if p.curTokenIs(lexer.IDENT) && p.curToken.Literal == "when" {
		hasGuard = true
		p.nextToken() // move to guard condition
		matchCase.Guard = p.parseExpression(LOWEST)
		// parseExpression leaves us on last token of guard, -> is in peek
	}

	// Expect ->
	// If we parsed a guard, curToken is on last token of guard, -> is next
	// If no guard, curToken should already be on ->
	if hasGuard {
		if !p.expectPeek(lexer.THIN_ARROW) {
			return nil
		}
	} else {
		if !p.curTokenIs(lexer.THIN_ARROW) {
			p.addError(fmt.Sprintf("expected '->' in match case, got %s", p.curToken.Type), p.curToken)
			return nil
		}
	}

	p.nextToken() // move to body expression

	// Parse the body expression
	// The body ends at the next '|', which starts the next arm; a bitwise
	// or inside an arm goes in parentheses.
	saved := p.stopAtPipe
	p.stopAtPipe = true
	matchCase.Body = p.parseExpression(LOWEST)
	p.stopAtPipe = saved
	if matchCase.Body == nil {
		p.addError("expected expression after '->' in match case", p.curToken)
		return nil
	}

	return matchCase
}

// patternBinds says whether a pattern binds a name.
func patternBinds(pat ast.Pattern) bool {
	switch pt := pat.(type) {
	case *ast.BindingPattern:
		return true
	case *ast.TypePattern:
		return pt.Binding != ""
	case *ast.StructPattern:
		for _, f := range pt.Fields {
			if patternBinds(f) {
				return true
			}
		}
	}
	return false
}

// parsePattern parses a pattern for pattern matching
func (p *Parser) parsePattern() ast.Pattern {
	// State.Idle, Color.Red: a value the subject is compared with.
	if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.DOT) {
		tok := p.curToken
		var value ast.Expression = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		for p.peekTokenIs(lexer.DOT) {
			p.nextToken() // '.'
			dotTok := p.curToken
			p.nextToken()
			if !lexer.CanBeFieldName(p.curToken) {
				p.addError(fmt.Sprintf("expected a name after '.' in a pattern, got %s", p.curToken.Type), p.curToken)
				return nil
			}
			value = &ast.DotExpression{Token: dotTok, Left: value, Right: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
		}
		return &ast.ValuePattern{Token: tok, Value: value}
	}
	// -1: a negative number.
	if p.curTokenIs(lexer.MINUS) && p.peekTokenIs(lexer.NUMBER) {
		tok := p.curToken
		p.nextToken()
		num := p.parseNumberLiteral()
		return &ast.LiteralPattern{Token: tok, Value: &ast.PrefixExpression{Token: tok, Operator: "-", Right: num}}
	}
	switch p.curToken.Type {
	case lexer.IDENT:
		// Could be: wildcard (_), binding (x), or type pattern
		if p.curToken.Literal == "_" {
			return &ast.WildcardPattern{
				Token: p.curToken,
			}
		}

		// Check if it's a type pattern (identifier followed by : or just a capitalized type name)
		// For now, treat as binding pattern
		// If peek is COLON, it's a type pattern with binding
		if p.peekTokenIs(lexer.COLON) {
			name := p.curToken.Literal
			p.nextToken() // move to :
			p.nextToken() // move to type name

			// Accept IDENT or type keywords (number, string, boolean, etc.)
			var typeName string
			if p.curTokenIs(lexer.IDENT) {
				typeName = p.curToken.Literal
			} else if p.curTokenIs(lexer.NUMBER_TYPE) {
				typeName = "number"
			} else if p.curTokenIs(lexer.STRING_TYPE) {
				typeName = "string"
			} else if p.curTokenIs(lexer.BOOLEAN) {
				typeName = "boolean"
			} else if p.curTokenIs(lexer.ANY) {
				typeName = "any"
			} else if p.curTokenIs(lexer.TABLE) {
				typeName = "table"
			} else {
				p.addError(fmt.Sprintf("expected type name after ':', got %s", p.curToken.Type), p.curToken)
				return nil
			}

			return &ast.TypePattern{
				Token:    p.curToken,
				TypeName: typeName,
				Binding:  name,
			}
		}

		// Check if it's a standalone type (capitalized identifier)
		// For simplicity, if it starts with uppercase, treat as type pattern
		if len(p.curToken.Literal) > 0 && p.curToken.Literal[0] >= 'A' && p.curToken.Literal[0] <= 'Z' {
			return &ast.TypePattern{
				Token:    p.curToken,
				TypeName: p.curToken.Literal,
				Binding:  "",
			}
		}

		// Otherwise it's a binding pattern
		return &ast.BindingPattern{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}

	case lexer.NUMBER, lexer.STRING, lexer.TRUE, lexer.FALSE, lexer.NIL:
		// Literal pattern
		var value ast.Expression
		switch p.curToken.Type {
		case lexer.NUMBER:
			value = p.parseNumberLiteral()
		case lexer.STRING:
			value = p.parseStringLiteral()
		case lexer.TRUE, lexer.FALSE:
			value = p.parseBooleanLiteral()
		case lexer.NIL:
			value = p.parseNilLiteral()
		}

		return &ast.LiteralPattern{
			Token: p.curToken,
			Value: value,
		}

	case lexer.LBRACE:
		// Struct pattern { field1: pattern1, field2: pattern2 }
		return p.parseStructPattern()

	default:
		p.addError(fmt.Sprintf("unexpected token in pattern: %s", p.curToken.Type), p.curToken)
		return nil
	}
}

// parseStructPattern parses a struct/object destructuring pattern
// Syntax: { field1: pattern1, field2: pattern2, ... }
func (p *Parser) parseStructPattern() ast.Pattern {
	pattern := &ast.StructPattern{
		Token:  p.curToken, // '{' token
		Fields: make(map[string]ast.Pattern),
	}

	p.nextToken() // move past {

	// Empty struct pattern
	if p.curTokenIs(lexer.RBRACE) {
		return pattern
	}

	// Parse fields
	for {
		// Expect field name. Lunar keywords that Lua does not reserve are valid
		// field names, so discriminated unions can match on { type: "click" }.
		if !lexer.CanBeFieldName(p.curToken) {
			p.addError(fmt.Sprintf("expected field name in struct pattern, got %s", p.curToken.Type), p.curToken)
			return nil
		}

		fieldName := p.curToken.Literal
		p.nextToken() // move past field name

		// Expect :
		if !p.curTokenIs(lexer.COLON) {
			p.addError(fmt.Sprintf("expected ':' after field name in struct pattern, got %s", p.curToken.Type), p.curToken)
			return nil
		}

		p.nextToken() // move to pattern

		// Parse field pattern
		fieldPattern := p.parsePattern()
		if fieldPattern == nil {
			return nil
		}

		pattern.Fields[fieldName] = fieldPattern

		p.nextToken() // move past pattern

		// Check for more fields or end
		if p.curTokenIs(lexer.RBRACE) {
			break
		}

		if !p.curTokenIs(lexer.COMMA) {
			p.addError(fmt.Sprintf("expected ',' or '}' in struct pattern, got %s", p.curToken.Type), p.curToken)
			return nil
		}

		p.nextToken() // move past comma

		// Allow trailing comma
		if p.curTokenIs(lexer.RBRACE) {
			break
		}
	}

	return pattern
}

// clearStopAtPipe makes '|' an operator again inside brackets or a block,
// even within a match arm, and returns what restores the arm's setting:
// defer p.clearStopAtPipe()().
func (p *Parser) clearStopAtPipe() func() {
	saved := p.stopAtPipe
	p.stopAtPipe = false
	return func() { p.stopAtPipe = saved }
}

// parseSingleArgumentCall parses Lua's f{...} and f"...": a call whose one
// argument is a table constructor or a string, written without parentheses.
func (p *Parser) parseSingleArgumentCall(left ast.Expression) ast.Expression {
	call := &ast.CallExpression{Token: p.curToken, Function: left}
	var arg ast.Expression
	if p.curTokenIs(lexer.LBRACE) {
		arg = p.parseTableLiteral()
	} else {
		arg = p.parseStringLiteral()
	}
	if isNilNode(arg) {
		return nil
	}
	call.Arguments = []ast.Expression{arg}
	return call
}

// isCallable says whether Lua allows a call on e without parentheses
// around it: a name, a field or method, an index, or a call.
func isCallable(e ast.Expression) bool {
	switch n := e.(type) {
	case *ast.Identifier, *ast.DotExpression, *ast.IndexExpression, *ast.SuperExpression,
		*ast.ParenExpression, *ast.GenericType:
		return true
	case *ast.CallExpression:
		return true
	default:
		_ = n
		return false
	}
}
