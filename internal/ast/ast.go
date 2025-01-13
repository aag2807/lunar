package ast

import (
	"fmt"
	"lunar/internal/lexer"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Expression interface {
	Node
	expressionNode()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type NumberLiteral struct {
	Token lexer.Token
	Value float64
}

func (i *NumberLiteral) expressionNode()      {}
func (i *NumberLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *NumberLiteral) String() string       { return i.Token.Literal }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (i *StringLiteral) expressionNode()      {}
func (i *StringLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *StringLiteral) String() string       { return i.Token.Literal }

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (i *BooleanLiteral) expressionNode()      {}
func (i *BooleanLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *BooleanLiteral) String() string       { return i.Token.Literal }

type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) expressionNode()      {}
func (i *InfixExpression) TokenLiteral() string { return i.Token.Literal }
func (i *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)",
		i.Left.String(),
		i.Operator,
		i.Right.String(),
	)
}
