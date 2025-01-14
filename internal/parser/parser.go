package parser

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strconv"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > OR <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X OR !X
	DOT         // foo.bar
	CALL        // function(x)
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ:         EQUALS,
	lexer.NOT_EQ:     EQUALS,
	lexer.NOT_EQ_LUA: EQUALS,
	lexer.LT:         LESSGREATER,
	lexer.GT:         LESSGREATER,
	lexer.PLUS:       SUM,
	lexer.MINUS:      SUM,
	lexer.ASTERISK:   PRODUCT,
	lexer.DOT:        DOT,
	lexer.LPAREN:     CALL,
	lexer.CONCAT:     SUM,
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

type Parser struct {
	l *lexer.Lexer

	curToken  lexer.Token
	peekToken lexer.Token

	errors []string

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	//register prefix parse functions
	p.prefixParseFns = make(map[lexer.TokenType]prefixParseFn)
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)

	//register infix operators
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.DOT, p.parseDotExpression)
	p.registerInfix(lexer.CONCAT, p.parseInfixExpression)

	// read to tokens to initialize curtoken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as number", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	return &ast.NumberLiteral{Token: p.curToken, Value: value}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
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
	p.nextToken() // consumes the first '('

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return exp
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
	exp := &ast.DotExpression{
		Token: p.curToken,
		Left:  left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
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
	p.errors = append(p.errors, msg)
}

func (p *Parser) Errors() []string {
	return p.errors
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

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseVariableDeclaration() *ast.VariableDeclaration {
	decl := &ast.VariableDeclaration{
		Token:      p.curToken,
		IsConstant: p.curToken.Type == lexer.CONST,
	}

	// Parse identifier (name)
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	decl.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse type annotation if present
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consume :
		p.nextToken() // move to type

		typeExp := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Check for optional type (?)
		if p.peekTokenIs(lexer.QUESTION) {
			p.nextToken()
			decl.Type = &ast.OptionalType{
				Token: p.curToken,
				Type:  typeExp,
			}
		} else {
			decl.Type = typeExp
		}
	}

	// Parse initializer if present
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // consume =
		p.nextToken() // move to expression
		decl.Value = p.parseExpression(LOWEST)
	}

	return decl
}

func (p *Parser) parseType() ast.Expression {
	if p.curTokenIs(lexer.IDENT) {
		return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}
	return nil
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // consumes :
		p.nextToken() // moves onto type
		param.Type = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
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

	//parse function name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	fd.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	//parse the parameters
	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	fd.Parameters = p.parseFunctionParameters()

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() //consume :
		p.nextToken() // move onto return type
		fd.ReturnType = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	fd.Body = p.parseBlockStatement()

	return fd
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{
		Token:      p.curToken,
		Statements: []ast.Statement{},
	}

	p.nextToken()

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
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

	stmt.ReturnValue = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{
		Token:      p.curToken,
		Expression: p.parseExpression(LOWEST),
	}

	return stmt
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.FUNCTION:
		return p.parseFunctionDeclaration()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.LOCAL, lexer.CONST:
		return p.parseVariableDeclaration()
	default:
		return p.parseExpressionStatement()
	}
}
