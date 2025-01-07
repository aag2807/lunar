package ast

import "lunar/internal/lexer"

type Node interface {
	TokenLiteral() string
}

type Expression interface {
	Node
	expressionNode()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

type NumberLiteral struct {
	Token lexer.Token
	Value float64
}

type StringLiteral struct {
	Token lexer.Token
	Value string
}

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}
