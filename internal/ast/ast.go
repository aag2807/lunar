package ast

import (
	"bytes"
	"fmt"
	"lunar/internal/lexer"
	"strings"
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

type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String())
}

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

type DotExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	return fmt.Sprintf("%s.%s", de.Left.String(), de.Right.String())
}

type Statement interface {
	Node
	statementNode()
}

type VariableDeclaration struct {
	Token      lexer.Token
	Name       *Identifier
	Type       Expression
	Value      Expression
	IsConstant bool
}

func (vd *VariableDeclaration) statementNode()       {}
func (vd *VariableDeclaration) TokenLiteral() string { return vd.Token.Literal }
func (vd *VariableDeclaration) String() string {
	var out strings.Builder

	if vd.IsConstant {
		out.WriteString("const ")
	} else {
		out.WriteString("local ")
	}

	out.WriteString(vd.Name.String())

	// Type annotation
	if vd.Type != nil {
		out.WriteString(": ")
		out.WriteString(vd.Type.String())
	}

	// Value assignment
	if vd.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vd.Value.String())
	}

	return out.String()
}

type OptionalType struct {
	Token lexer.Token
	Type  Expression
}

func (ot *OptionalType) expressionNode()      {}
func (ot *OptionalType) TokenLiteral() string { return ot.Token.Literal }
func (ot *OptionalType) String() string       { return ot.Type.String() + "?" }
