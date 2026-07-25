package parser

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strconv"
	"strings"
)

const (
	_            int = iota
	LOWEST
	PIPE_PREC    // |> pipeline operator
	OR_PREC      // or
	BITWISE_OR   // |
	BITWISE_XOR  // ^
	BITWISE_AND  // &
	AND_PREC     // and
	EQUALS       // ==
	LESSGREATER  // > OR <
	SHIFT        // << >>
	SUM          // +
	PRODUCT      // * / %
	PREFIX       // -X OR !X OR not OR ~X
	DOT          // foo.bar
	CALL         // function(x)
)

var precedences = map[lexer.TokenType]int{
	lexer.AS:               PREFIX,    // type assertions have same precedence as prefix operators
	lexer.PIPE_OP:          PIPE_PREC, // |> pipeline operator
	lexer.NULLISH_COALESCE: OR_PREC,   // ?? has same precedence as ||
	lexer.OR:               OR_PREC,
	lexer.PIPE:             BITWISE_OR,
	lexer.CARET:            PRODUCT, // exponentiation, as in Lua
	lexer.TILDE:            BITWISE_XOR,
	lexer.AMPERSAND:        BITWISE_AND,
	lexer.AND:              AND_PREC,
	lexer.EQ:               EQUALS,
	lexer.NOT_EQ:           EQUALS,
	lexer.NOT_EQ_LUA:       EQUALS,
	lexer.LT:               LESSGREATER,
	lexer.GT:               LESSGREATER,
	lexer.LT_EQ:            LESSGREATER,
	lexer.GT_EQ:            LESSGREATER,
	lexer.LEFT_SHIFT:       SHIFT,
	lexer.RIGHT_SHIFT:      SHIFT,
	lexer.PLUS:             SUM,
	lexer.MINUS:            SUM,
	lexer.ASTERISK:         PRODUCT,
	lexer.SLASH:            PRODUCT,
	lexer.FLOOR_DIV:        PRODUCT,
	lexer.MODULO:           PRODUCT,
	lexer.DOT:              DOT,
	lexer.OPTIONAL_CHAIN:   DOT, // ?. has same precedence as .
	lexer.LBRACKET:         CALL, // index has same precedence as function call
	lexer.LPAREN:           CALL,
	lexer.CONCAT:           SUM,
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

type Parser struct {
	l *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors []*ParseError

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

// addError adds a parse error with location information
func (p *Parser) addError(message string, token lexer.Token) {
	p.errors = append(p.errors, &ParseError{
		Message: message,
		Line:    token.Line,
		Column:  token.Column,
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
	// `get` and `set` only mean accessors inside a class body; everywhere else
	// they are ordinary names, and common ones (a module exporting get/set).
	p.registerPrefix(lexer.GET, p.parseIdentifier)
	p.registerPrefix(lexer.SET, p.parseIdentifier)
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
	p.registerPrefix(lexer.AWAIT, p.parseAwaitExpression)
	p.registerPrefix(lexer.MATCH, p.parseMatchExpression)

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
		// Split >> into two > tokens
		p.curToken = lexer.Token{Type: lexer.GT, Literal: ">", Line: p.curToken.Line, Column: p.curToken.Column}
		p.peekToken = lexer.Token{Type: lexer.GT, Literal: ">", Line: p.curToken.Line, Column: p.curToken.Column + 1}
		return true
	}

	p.peekError(lexer.GT)
	return false
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseSuperExpression() ast.Expression {
	return &ast.SuperExpression{
		Token: p.curToken,
	}
}

// parseNumericLiteral converts a number literal's text to a value. Go's
// ParseFloat rejects Lua's hexadecimal form, so 0xFF is parsed separately.
func parseNumericLiteral(literal string) (float64, error) {
	if len(literal) > 2 && literal[0] == '0' && (literal[1] == 'x' || literal[1] == 'X') {
		value, err := strconv.ParseUint(literal[2:], 16, 64)
		return float64(value), err
	}

	return strconv.ParseFloat(literal, 64)
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	value, err := parseNumericLiteral(p.curToken.Literal)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as number", p.curToken.Literal)
		p.addError(msg, p.curToken)
		return nil
	}

	return &ast.NumberLiteral{Token: p.curToken, Value: value}
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
	var currentPart string
	i := 0

	for i < len(raw) {
		// Look for ${ pattern
		if i < len(raw)-1 && raw[i] == '$' && raw[i+1] == '{' {
			// Add the current part
			template.Parts = append(template.Parts, currentPart)
			currentPart = ""

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

			// Parse the expression
			exprStr := raw[exprStart:i]
			exprLexer := lexer.New(exprStr)
			exprParser := New(exprLexer)
			expr := exprParser.parseExpression(LOWEST)

			if len(exprParser.Errors()) > 0 {
				for _, err := range exprParser.Errors() {
					p.addError("Template expression error: "+err.Message, token)
				}
				return nil
			}

			template.Expressions = append(template.Expressions, expr)
			i++ // skip the closing }
		} else {
			currentPart += string(raw[i])
			i++
		}
	}

	// Add the final part
	template.Parts = append(template.Parts, currentPart)

	return template
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

	for precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	// We're at '('. Need to disambiguate between:
	// - Grouped expression: (a + b)
	// - Arrow function: (x: number) => x * 2

	// Save state for lookahead
	savedState := p.SaveState()

	// Quick checks for obvious arrow functions
	if p.peekTokenIs(lexer.RPAREN) || p.peekTokenIs(lexer.ELLIPSIS) {
		// () => or (...args) => pattern
		return p.parseArrowFunction()
	}

	// Look ahead to detect arrow function patterns
	// Pattern: (ident : type) or (ident , ...) or (ident) =>
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken() // move from '(' to ident

		// Now curToken = ident, check what follows
		if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.COMMA) {
			// (ident : ...) or (ident , ...) - arrow function
			p.RestoreState(savedState)
			return p.parseArrowFunction()
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
	p.nextToken() // consume '('

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
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
		// (...args) pattern - rest parameter definitely arrow function
		return true
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
	return p.peekTokenIs(lexer.LPAREN)
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
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.addError(msg, p.curToken)
}

func (p *Parser) Errors() []*ParseError {
	return p.errors
}

// Parse parses the entire program and returns a slice of statements
func (p *Parser) Parse() []ast.Statement {
	statements := []ast.Statement{}

	// Note: New() already initializes curToken and peekToken by calling nextToken() twice
	// So we don't need to call nextToken() here

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
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
	// is a spread, as in {...items}.
	if !p.peekStartsExpression() {
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
		if !p.curTokenIs(lexer.IDENT) {
			p.addError("expected identifier in tuple destructuring", p.curToken)
			return nil
		}
		decl.Names = append(decl.Names, p.parseIdentifierOrContextual())

		// Parse type annotation if present
		if p.peekTokenIs(lexer.COLON) {
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

			if !p.curTokenIs(lexer.IDENT) {
				p.addError(fmt.Sprintf("expected identifier after comma in tuple, got %s", p.curToken.Type), p.curToken)
				return nil
			}

			decl.Names = append(decl.Names, p.parseIdentifierOrContextual())

			// Parse type annotation if present
			if p.peekTokenIs(lexer.COLON) {
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

		// Parse type annotation if present
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // consume :
			p.nextToken() // move to type
			decl.Types = append(decl.Types, p.parseType())
		}

		// Parse additional variables after commas
		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume previous token/type
			p.nextToken() // consume comma, now on next identifier

			if !p.curTokenIs(lexer.IDENT) {
				p.addError(fmt.Sprintf("expected identifier after comma, got %s", p.curToken.Type), p.curToken)
				return nil
			}

			decl.Names = append(decl.Names, p.parseIdentifierOrContextual())

			// Parse type annotation if present
			if p.peekTokenIs(lexer.COLON) {
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
		return p.parseTupleOrFunctionType()
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
		value, _ := parseNumericLiteral(p.curToken.Literal)
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
		typeExpr = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
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
		return p.parseTupleOrFunctionType()
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
	currentType := baseType

	// First pass: handle high-precedence suffixes (arrays, generics, optional)
	// These bind tighter than union types
	for {
		switch {
		case p.peekTokenIs(lexer.LBRACKET):
			// Could be array type T[] or indexed access type T[K]
			p.nextToken() // consume '['
			p.nextToken() // move to content or ']'

			if p.curTokenIs(lexer.RBRACKET) {
				// Empty brackets: T[] is array type
				currentType = &ast.ArrayType{
					Token:       baseType.(*ast.Identifier).Token,
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
				Token:         baseType.(*ast.Identifier).Token,
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
		return p.parseTupleOrFunctionType()
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
		value, _ := parseNumericLiteral(p.curToken.Literal)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.TRUE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case lexer.FALSE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		typeExpr = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
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
					Token:       typeExpr.(*ast.Identifier).Token,
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
				Token:         typeExpr.(*ast.Identifier).Token,
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
		return p.parseTupleOrFunctionType()
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
		value, _ := parseNumericLiteral(p.curToken.Literal)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.TRUE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: true}
	case lexer.FALSE:
		typeExpr = &ast.BooleanLiteral{Token: p.curToken, Value: false}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL, lexer.NEVER, lexer.UNKNOWN:
		typeExpr = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
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
				Token:         typeExpr.(*ast.Identifier).Token,
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

	// A bare `table` is a table of anything, as documented. Only `table<K, V>`
	// carries type arguments.
	if !p.peekTokenIs(lexer.LT) {
		return &ast.TableType{
			Token:     tableToken,
			KeyType:   &ast.Identifier{Token: tableToken, Value: "any"},
			ValueType: &ast.Identifier{Token: tableToken, Value: "any"},
		}
	}

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
				// It's a tuple type
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
func (p *Parser) isContextualKeyword(t lexer.TokenType) bool {
	return t == lexer.STRING_TYPE || t == lexer.TABLE || t == lexer.TYPE || t == lexer.GET || t == lexer.SET
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
}

// SaveState saves the current parser state for lookahead
func (p *Parser) SaveState() ParserState {
	return ParserState{
		curToken:   p.curToken,
		peekToken:  p.peekToken,
		lexerState: p.l.SaveState(),
	}
}

// RestoreState restores a previously saved parser state
func (p *Parser) RestoreState(state ParserState) {
	p.curToken = state.curToken
	p.peekToken = state.peekToken
	p.l.RestoreState(state.lexerState)
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

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{
		Token: p.curToken,
	}

	// Check for visibility modifiers (for constructor parameter properties)
	switch p.curToken.Type {
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
	fd := &ast.FunctionDeclaration{
		Token: p.curToken,
	}

	//parse function name - allows context-aware keywords
	if !p.expectPeekIdentOrContextual() {
		return nil
	}
	fd.Name = p.parseIdentifierOrContextual()

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

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() //consume :
		p.nextToken() // move onto return type
		fd.ReturnType = p.parseType()
	}

	fd.Body = p.parseBlockStatement()

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

	// Parse optional return type
	if p.peekTokenIs(lexer.COLON) {
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

	// Parse optional return type
	if p.peekTokenIs(lexer.COLON) {
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

	for !p.curTokenIs(terminator) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken() // move past 'return'

	// Parse multiple return values separated by commas
	stmt.ReturnValues = []ast.Expression{}

	// If we hit a newline or end, return empty
	if p.curTokenIs(lexer.EOF) || p.curToken.Line != stmt.Token.Line {
		return stmt
	}

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

	// Otherwise, it's just an expression statement
	return &ast.ExpressionStatement{
		Token:      p.curToken,
		Expression: expr,
	}
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
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
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

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

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	return &ast.BreakStatement{Token: p.curToken}
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
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

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// Try to parse as key-value pair first - allows context-aware keywords
		// Check for [key] = value syntax
		if p.curTokenIs(lexer.LBRACKET) {
			p.nextToken() // consume '['
			key := p.parseExpression(LOWEST)
			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}
			if !p.expectPeek(lexer.ASSIGN) {
				return nil
			}
			p.nextToken() // move past '='
			value := p.parseExpression(LOWEST)
			table.Pairs[key] = value
		} else if p.curTokenIsIdentOrContextual() && p.peekTokenIs(lexer.ASSIGN) {
			// Key-value pair with identifier key
			key := p.parseIdentifierOrContextual()
			p.nextToken() // consume identifier
			p.nextToken() // consume '='

			value := p.parseExpression(LOWEST)
			table.Pairs[key] = value
		} else {
			// Array-style value
			value := p.parseExpression(LOWEST)
			table.Values = append(table.Values, value)
		}

		// Check for comma or end
		if !p.peekTokenIs(lexer.RBRACE) {
			if !p.expectPeek(lexer.COMMA) {
				return nil
			}
			p.nextToken() // move past comma
		} else {
			p.nextToken() // move to '}'
		}
	}

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
	class := &ast.ClassDeclaration{
		Token:      p.curToken,
		Properties: []*ast.PropertyDeclaration{},
		Methods:    []*ast.FunctionDeclaration{},
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
		class.Extends = &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
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
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
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

		// Track modifiers for current member
		isStatic := false
		isAbstract := false
		isReadonly := false
		visibility := "public" // default visibility

		// Parse modifiers
		for p.curToken.Type == lexer.PUBLIC || p.curToken.Type == lexer.PRIVATE || p.curToken.Type == lexer.PROTECTED ||
			p.curToken.Type == lexer.STATIC || p.curToken.Type == lexer.ABSTRACT || p.curToken.Type == lexer.READONLY {
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

		// Now parse the actual member
		switch p.curToken.Type {
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
			if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.ASSIGN) {
				// It's a property (with type annotation or initial value)
				prop := p.parsePropertyDeclaration()
				prop.Visibility = visibility
				prop.IsStatic = isStatic
				prop.IsReadonly = isReadonly
				class.Properties = append(class.Properties, prop)
			} else if p.peekTokenIs(lexer.LPAREN) {
				// It's a method
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
	}

	return class
}

func (p *Parser) parsePropertyDeclaration() *ast.PropertyDeclaration {
	prop := &ast.PropertyDeclaration{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
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
	if isAbstract && !p.abstractMethodHasBody() {
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

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume '<'
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
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LBRACKET) {
			// Index signature: [key: KeyType]: ValueType
			indexSig := p.parseIndexSignature()
			if indexSig != nil {
				iface.IndexSignature = indexSig
			}
			p.nextToken() // move past index signature
		} else if p.curTokenIs(lexer.STATIC) {
			// Handle static keyword
			p.nextToken() // move past 'static'
			if p.curTokenIsIdentOrContextual() {
				if p.peekTokenIs(lexer.COLON) {
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
					p.nextToken()
				}
			}
		} else if p.curTokenIsIdentOrContextual() {
			if p.peekTokenIs(lexer.COLON) {
				// Property
				prop := p.parsePropertyDeclaration()
				iface.Properties = append(iface.Properties, prop)
			} else if p.peekTokenIs(lexer.LPAREN) {
				// Method signature
				method := p.parseInterfaceMethod()
				iface.Methods = append(iface.Methods, method)
			} else {
				p.nextToken()
			}
		} else {
			p.nextToken()
		}
	}

	return iface
}

func (p *Parser) parseInterfaceMethod() *ast.InterfaceMethod {
	method := &ast.InterfaceMethod{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
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
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
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
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIsIdentOrContextual() {
				prop := &ast.PropertyDeclaration{
					Token: p.curToken,
					Name:  p.parseIdentifierOrContextual(),
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
		importStmt.IsWildcard = true
		p.nextToken() // move past '*'
	} else if p.curTokenIs(lexer.LBRACE) {
		// Named imports: import { name1, name2 } from "module"
		p.nextToken() // move past '{'

		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			// An imported name is whatever the module exported, which may be a
			// word Lunar reserves but Lua does not, such as `get` or `type`.
			if !lexer.CanBeFieldName(p.curToken) {
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
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			ns.Statements = append(ns.Statements, stmt)
		}
		p.nextToken()
	}

	return ns
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
		declareStmt.Declaration = p.parseFunctionDeclaration()
	case lexer.CLASS:
		declareStmt.Declaration = p.parseClassDeclaration()
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

	p.nextToken() // move past 'with' to first case

	// After advancing, curToken should be the first '|' or 'end'

	// Parse match cases
	// Each case starts with |
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
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
	// Use BITWISE_OR precedence to prevent consuming the next '|' as an operator
	matchCase.Body = p.parseExpression(BITWISE_OR)
	if matchCase.Body == nil {
		p.addError("expected expression after '->' in match case", p.curToken)
		return nil
	}

	return matchCase
}

// parsePattern parses a pattern for pattern matching
func (p *Parser) parsePattern() ast.Pattern {
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
