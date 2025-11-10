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
	OR_PREC     // or
	AND_PREC    // and
	EQUALS      // ==
	LESSGREATER // > OR <
	SUM         // +
	PRODUCT     // * / %
	PREFIX      // -X OR !X OR not
	DOT         // foo.bar
	CALL        // function(x)
)

var precedences = map[lexer.TokenType]int{
	lexer.OR:         OR_PREC,
	lexer.AND:        AND_PREC,
	lexer.EQ:         EQUALS,
	lexer.NOT_EQ:     EQUALS,
	lexer.NOT_EQ_LUA: EQUALS,
	lexer.LT:         LESSGREATER,
	lexer.GT:         LESSGREATER,
	lexer.LT_EQ:      LESSGREATER,
	lexer.GT_EQ:      LESSGREATER,
	lexer.PLUS:       SUM,
	lexer.MINUS:      SUM,
	lexer.ASTERISK:   PRODUCT,
	lexer.SLASH:      PRODUCT,
	lexer.MODULO:     PRODUCT,
	lexer.DOT:        DOT,
	lexer.LBRACKET:   CALL, // index has same precedence as function call
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
	p.registerPrefix(lexer.SELF, p.parseIdentifier) // self is like an identifier
	p.registerPrefix(lexer.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.NIL, p.parseNilLiteral)
	p.registerPrefix(lexer.BANG, p.parsePrefixExpression)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(lexer.LBRACE, p.parseTableLiteral)

	//register infix operators
	p.infixParseFns = make(map[lexer.TokenType]infixParseFn)
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.ASTERISK, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.MODULO, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NOT_EQ_LUA, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.GT_EQ, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
	p.registerInfix(lexer.LBRACKET, p.parseIndexExpression)
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
		decl.Type = p.parseType()
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
	var typeExpr ast.Expression

	switch p.curToken.Type {
	case lexer.LPAREN:
		// Could be tuple type or function type
		return p.parseTupleOrFunctionType()
	case lexer.TABLE:
		// table<K, V>
		typeExpr = p.parseTableType()
	case lexer.STRING:
		// String literal in type position (for literal types)
		typeExpr = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		// Number literal in type position (for literal types)
		value, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL:
		typeExpr = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
		return nil
	}

	// Check for suffixes and modifiers
	return p.parseTypeSuffix(typeExpr)
}

func (p *Parser) parseSimpleType() ast.Expression {
	switch p.curToken.Type {
	case lexer.LPAREN:
		return p.parseTupleOrFunctionType()
	case lexer.TABLE:
		return p.parseTableType()
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL:
		return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	default:
		return nil
	}
}

func (p *Parser) parseTypeSuffix(baseType ast.Expression) ast.Expression {
	currentType := baseType

	// First pass: handle high-precedence suffixes (arrays, generics, optional)
	// These bind tighter than union types
	for {
		switch {
		case p.peekTokenIs(lexer.LBRACKET):
			// Array type: T[]
			p.nextToken() // consume '['
			if !p.expectPeek(lexer.RBRACKET) {
				return nil
			}
			currentType = &ast.ArrayType{
				Token:       baseType.(*ast.Identifier).Token,
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

			if !p.expectPeek(lexer.GT) {
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
			goto checkUnion
		}
	}

checkUnion:
	// Second pass: handle union types (lowest precedence)
	if p.peekTokenIs(lexer.PIPE) {
		types := []ast.Expression{currentType}
		unionToken := p.peekToken
		for p.peekTokenIs(lexer.PIPE) {
			p.nextToken() // consume '|'
			p.nextToken() // move to next type
			// Parse the next type WITHOUT processing unions (to avoid nested unions)
			nextType := p.parseNonUnionType()
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

// parseNonUnionType parses a type with all suffixes EXCEPT union types
// This is used when parsing union members to avoid nested union structures
func (p *Parser) parseNonUnionType() ast.Expression {
	var typeExpr ast.Expression

	switch p.curToken.Type {
	case lexer.LPAREN:
		// Could be tuple type or function type
		return p.parseTupleOrFunctionType()
	case lexer.TABLE:
		// table<K, V>
		typeExpr = p.parseTableType()
	case lexer.STRING:
		// String literal in type position (for literal types)
		typeExpr = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	case lexer.NUMBER:
		// Number literal in type position (for literal types)
		value, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		typeExpr = &ast.NumberLiteral{Token: p.curToken, Value: value}
	case lexer.IDENT, lexer.STRING_TYPE, lexer.NUMBER_TYPE, lexer.BOOLEAN, lexer.ANY, lexer.VOID, lexer.NIL:
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

			if !p.expectPeek(lexer.GT) {
				return nil
			}

			currentType = &ast.GenericType{
				Token:         typeExpr.(*ast.Identifier).Token,
				BaseType:      typeExpr,
				TypeArguments: typeArgs,
			}

		case p.peekTokenIs(lexer.QUESTION):
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
	if !p.expectPeek(lexer.GT) {
		return nil
	}

	return &ast.TableType{
		Token:     tableToken,
		KeyType:   keyType,
		ValueType: valueType,
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

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{
		Token: p.curToken,
		Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
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

	//parse function name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	fd.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

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

func (p *Parser) parseExpressionStatement() ast.Statement {
	// Try to parse as expression first
	expr := p.parseExpression(LOWEST)

	// Check if this is an assignment
	if p.peekTokenIs(lexer.ASSIGN) {
		assignToken := p.peekToken
		p.nextToken() // consume '='
		p.nextToken() // move to value expression

		return &ast.AssignmentStatement{
			Token: assignToken,
			Name:  expr,
			Value: p.parseExpression(LOWEST),
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
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.LOCAL, lexer.CONST:
		return p.parseVariableDeclaration()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.DO:
		return p.parseDoStatement()
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.CLASS:
		return p.parseClassDeclaration()
	case lexer.INTERFACE:
		return p.parseInterfaceDeclaration()
	case lexer.ENUM:
		return p.parseEnumDeclaration()
	case lexer.TYPE:
		return p.parseTypeDeclaration()
	case lexer.EXPORT:
		return p.parseExportStatement()
	case lexer.IMPORT:
		return p.parseImportStatement()
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

	// Parse consequence block (stops at 'else' or 'end')
	stmt.Consequence = p.parseIfBlockStatement()

	// Check for else
	if p.curTokenIs(lexer.ELSE) {
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

	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.ELSE) && !p.curTokenIs(lexer.EOF) {
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

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}

	// Expect variable name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

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
		p.errors = append(p.errors, msg)
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
		// Try to parse as key-value pair first
		// Look ahead to see if this is a key = value pattern
		if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.ASSIGN) {
			// Key-value pair
			key := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
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

func (p *Parser) parseClassDeclaration() *ast.ClassDeclaration {
	class := &ast.ClassDeclaration{
		Token:      p.curToken,
		Properties: []*ast.PropertyDeclaration{},
		Methods:    []*ast.FunctionDeclaration{},
	}

	// Parse class name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	class.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Parse generic parameters if present: <T, U>
	if p.peekTokenIs(lexer.LT) {
		p.nextToken() // consume <
		class.GenericParams = p.parseGenericParameters()
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
		switch p.curToken.Type {
		case lexer.PUBLIC, lexer.PRIVATE:
			// Property or method with visibility
			visibility := p.curToken.Literal
			p.nextToken()

			if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
				// It's a property
				prop := p.parsePropertyDeclaration()
				prop.Visibility = visibility
				class.Properties = append(class.Properties, prop)
			} else if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.LPAREN) {
				// It's a method
				method := p.parseMethodDeclaration()
				class.Methods = append(class.Methods, method)
			} else {
				p.nextToken()
			}

		case lexer.CONSTRUCTOR:
			class.Constructor = p.parseConstructorDeclaration()
			p.nextToken()

		case lexer.IDENT:
			// Property without visibility modifier
			if p.peekTokenIs(lexer.COLON) {
				prop := p.parsePropertyDeclaration()
				class.Properties = append(class.Properties, prop)
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

	// Expect colon
	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	p.nextToken() // move to type
	prop.Type = p.parseType()

	p.nextToken() // move past type
	return prop
}

func (p *Parser) parseMethodDeclaration() *ast.FunctionDeclaration {
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

	// Parse body
	method.Body = p.parseBlockStatement()

	return method
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

func (p *Parser) parseInterfaceDeclaration() *ast.InterfaceDeclaration {
	iface := &ast.InterfaceDeclaration{
		Token:      p.curToken,
		Methods:    []*ast.InterfaceMethod{},
		Properties: []*ast.PropertyDeclaration{},
	}

	// Parse interface name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	iface.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

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

	// Parse interface body
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
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

func (p *Parser) parseEnumDeclaration() *ast.EnumDeclaration {
	enum := &ast.EnumDeclaration{
		Token:   p.curToken,
		Members: []*ast.EnumMember{},
	}

	// Parse enum name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	enum.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // move past enum name

	// Parse enum members
	for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			member := &ast.EnumMember{
				Token: p.curToken,
				Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
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

	// Parse type name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	typeDecl.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // move past name

	// Check if it's an object shape declaration (type Name ... end) or alias (type Name = Type)
	if p.curTokenIs(lexer.ASSIGN) {
		// Type alias: type Name = Type
		p.nextToken() // move to type definition
		typeDecl.Type = p.parseType()
	} else {
		// Object shape: type Name { properties } end
		// Parse properties similar to interface
		for !p.curTokenIs(lexer.END) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.IDENT) {
				prop := &ast.PropertyDeclaration{
					Token: p.curToken,
					Name:  &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
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

		if !p.curTokenIs(lexer.RBRACE) {
			p.errors = append(p.errors, "expected '}' after import names")
			return nil
		}

		p.nextToken() // move past '}'
	}

	// Expect 'from' keyword
	if !p.curTokenIs(lexer.FROM) {
		p.errors = append(p.errors, "expected 'from' after import statement")
		return nil
	}

	p.nextToken() // move past 'from'

	// Expect string literal for module path
	if !p.curTokenIs(lexer.STRING) {
		p.errors = append(p.errors, "expected string literal for module path")
		return nil
	}

	importStmt.Module = p.curToken.Literal

	return importStmt
}

// parseGenericParameters parses generic type parameters: <T, U, V>
func (p *Parser) parseGenericParameters() []*ast.Identifier {
	params := []*ast.Identifier{}

	p.nextToken() // move past '<' to first parameter

	for !p.curTokenIs(lexer.GT) && !p.curTokenIs(lexer.EOF) {
		if !p.curTokenIs(lexer.IDENT) {
			p.peekError(lexer.IDENT)
			return nil
		}

		params = append(params, &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		})

		p.nextToken()

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken() // move past comma to next parameter
		}
	}

	if !p.curTokenIs(lexer.GT) {
		p.errors = append(p.errors, "expected '>' after generic parameters")
		return nil
	}

	return params
}
