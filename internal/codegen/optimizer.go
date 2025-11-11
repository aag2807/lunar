package codegen

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strconv"
)

// Optimizer performs compile-time optimizations on the AST
type Optimizer struct {
	enabled bool
}

// NewOptimizer creates a new optimizer
func NewOptimizer(enabled bool) *Optimizer {
	return &Optimizer{enabled: enabled}
}

// OptimizeStatements optimizes a list of statements
func (o *Optimizer) OptimizeStatements(statements []ast.Statement) []ast.Statement {
	if !o.enabled {
		return statements
	}

	optimized := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		if opt := o.optimizeStatement(stmt); opt != nil {
			optimized = append(optimized, opt)
		}
	}
	return optimized
}

// optimizeStatement optimizes a single statement
func (o *Optimizer) optimizeStatement(stmt ast.Statement) ast.Statement {
	if stmt == nil {
		return nil
	}

	switch node := stmt.(type) {
	case *ast.VariableDeclaration:
		// Optimize the value expression
		if node.Value != nil {
			node.Value = o.optimizeExpression(node.Value)
		}
		return node

	case *ast.ReturnStatement:
		if node.ReturnValue != nil {
			node.ReturnValue = o.optimizeExpression(node.ReturnValue)
		}
		return node

	case *ast.ExpressionStatement:
		node.Expression = o.optimizeExpression(node.Expression)
		return node

	case *ast.AssignmentStatement:
		node.Value = o.optimizeExpression(node.Value)
		return node

	case *ast.IfStatement:
		// Optimize condition
		node.Condition = o.optimizeExpression(node.Condition)

		// Constant condition optimization
		if boolLit, ok := node.Condition.(*ast.BooleanLiteral); ok {
			if boolLit.Value {
				// Condition is always true, replace with consequence
				return &ast.BlockStatement{
					Token:      node.Token,
					Statements: node.Consequence.Statements,
				}
			} else if node.Alternative != nil {
				// Condition is always false, replace with alternative
				return &ast.BlockStatement{
					Token:      node.Token,
					Statements: node.Alternative.Statements,
				}
			} else {
				// Condition is always false and no alternative, remove statement
				return nil
			}
		}

		// Optimize blocks
		node.Consequence = o.optimizeBlock(node.Consequence)
		if node.Alternative != nil {
			node.Alternative = o.optimizeBlock(node.Alternative)
		}
		return node

	case *ast.WhileStatement:
		node.Condition = o.optimizeExpression(node.Condition)
		node.Body = o.optimizeBlock(node.Body)
		return node

	case *ast.ForStatement:
		if node.Start != nil {
			node.Start = o.optimizeExpression(node.Start)
		}
		if node.End != nil {
			node.End = o.optimizeExpression(node.End)
		}
		if node.Step != nil {
			node.Step = o.optimizeExpression(node.Step)
		}
		if node.Iterator != nil {
			node.Iterator = o.optimizeExpression(node.Iterator)
		}
		node.Body = o.optimizeBlock(node.Body)
		return node

	case *ast.BlockStatement:
		return o.optimizeBlock(node)

	default:
		return stmt
	}
}

// optimizeBlock optimizes a block statement
func (o *Optimizer) optimizeBlock(block *ast.BlockStatement) *ast.BlockStatement {
	if block == nil {
		return nil
	}

	optimized := make([]ast.Statement, 0, len(block.Statements))
	reachable := true

	for _, stmt := range block.Statements {
		if !reachable {
			// Dead code after return/break
			break
		}

		if opt := o.optimizeStatement(stmt); opt != nil {
			optimized = append(optimized, opt)

			// Check if this statement makes subsequent code unreachable
			if _, isReturn := stmt.(*ast.ReturnStatement); isReturn {
				reachable = false
			}
			if _, isBreak := stmt.(*ast.BreakStatement); isBreak {
				reachable = false
			}
		}
	}

	block.Statements = optimized
	return block
}

// optimizeExpression optimizes an expression
func (o *Optimizer) optimizeExpression(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.InfixExpression:
		return o.optimizeInfixExpression(node)

	case *ast.PrefixExpression:
		return o.optimizePrefixExpression(node)

	case *ast.CallExpression:
		// Optimize arguments
		for i, arg := range node.Arguments {
			node.Arguments[i] = o.optimizeExpression(arg)
		}
		return node

	default:
		return expr
	}
}

// optimizeInfixExpression performs constant folding on infix expressions
func (o *Optimizer) optimizeInfixExpression(node *ast.InfixExpression) ast.Expression {
	// Optimize left and right first
	node.Left = o.optimizeExpression(node.Left)
	node.Right = o.optimizeExpression(node.Right)

	// Try constant folding
	leftNum, leftIsNum := node.Left.(*ast.NumberLiteral)
	rightNum, rightIsNum := node.Right.(*ast.NumberLiteral)

	if leftIsNum && rightIsNum {
		return o.foldNumericOperation(leftNum, rightNum, node.Operator, node.Token)
	}

	// String concatenation
	leftStr, leftIsStr := node.Left.(*ast.StringLiteral)
	rightStr, rightIsStr := node.Right.(*ast.StringLiteral)

	if leftIsStr && rightIsStr && node.Operator == ".." {
		return &ast.StringLiteral{
			Token: node.Token,
			Value: leftStr.Value + rightStr.Value,
		}
	}

	// Boolean constant folding
	if node.Operator == "&&" || node.Operator == "and" {
		if leftBool, ok := node.Left.(*ast.BooleanLiteral); ok {
			if !leftBool.Value {
				// false && x => false
				return &ast.BooleanLiteral{Token: node.Token, Value: false}
			} else {
				// true && x => x
				return node.Right
			}
		}
		if rightBool, ok := node.Right.(*ast.BooleanLiteral); ok {
			if !rightBool.Value {
				// x && false => false
				return &ast.BooleanLiteral{Token: node.Token, Value: false}
			} else {
				// x && true => x
				return node.Left
			}
		}
	}

	if node.Operator == "||" || node.Operator == "or" {
		if leftBool, ok := node.Left.(*ast.BooleanLiteral); ok {
			if leftBool.Value {
				// true || x => true
				return &ast.BooleanLiteral{Token: node.Token, Value: true}
			} else {
				// false || x => x
				return node.Right
			}
		}
		if rightBool, ok := node.Right.(*ast.BooleanLiteral); ok {
			if rightBool.Value {
				// x || true => true
				return &ast.BooleanLiteral{Token: node.Token, Value: true}
			} else {
				// x || false => x
				return node.Left
			}
		}
	}

	return node
}

// foldNumericOperation performs constant folding on numeric operations
func (o *Optimizer) foldNumericOperation(left, right *ast.NumberLiteral, operator string, token lexer.Token) ast.Expression {
	leftVal, _ := strconv.ParseFloat(left.Token.Literal, 64)
	rightVal, _ := strconv.ParseFloat(right.Token.Literal, 64)

	var result float64
	switch operator {
	case "+":
		result = leftVal + rightVal
	case "-":
		result = leftVal - rightVal
	case "*":
		result = leftVal * rightVal
	case "/":
		if rightVal == 0 {
			// Don't fold division by zero
			return &ast.InfixExpression{
				Token:    token,
				Left:     left,
				Operator: operator,
				Right:    right,
			}
		}
		result = leftVal / rightVal
	case "%":
		if rightVal == 0 {
			// Don't fold modulo by zero
			return &ast.InfixExpression{
				Token:    token,
				Left:     left,
				Operator: operator,
				Right:    right,
			}
		}
		result = float64(int(leftVal) % int(rightVal))
	case "^":
		// Lua power operator
		result = 1
		for i := 0; i < int(rightVal); i++ {
			result *= leftVal
		}
	case "==":
		return &ast.BooleanLiteral{Token: token, Value: leftVal == rightVal}
	case "!=":
		return &ast.BooleanLiteral{Token: token, Value: leftVal != rightVal}
	case "<":
		return &ast.BooleanLiteral{Token: token, Value: leftVal < rightVal}
	case "<=":
		return &ast.BooleanLiteral{Token: token, Value: leftVal <= rightVal}
	case ">":
		return &ast.BooleanLiteral{Token: token, Value: leftVal > rightVal}
	case ">=":
		return &ast.BooleanLiteral{Token: token, Value: leftVal >= rightVal}
	default:
		// Unknown operator, don't fold
		return &ast.InfixExpression{
			Token:    token,
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	// Return folded number
	return &ast.NumberLiteral{
		Token: lexer.Token{Literal: formatNumber(result)},
		Value: result,
	}
}

// optimizePrefixExpression optimizes prefix expressions
func (o *Optimizer) optimizePrefixExpression(node *ast.PrefixExpression) ast.Expression {
	node.Right = o.optimizeExpression(node.Right)

	// Constant folding for 'not'
	if node.Operator == "!" || node.Operator == "not" {
		if boolLit, ok := node.Right.(*ast.BooleanLiteral); ok {
			return &ast.BooleanLiteral{
				Token: node.Token,
				Value: !boolLit.Value,
			}
		}
	}

	// Constant folding for unary minus
	if node.Operator == "-" {
		if numLit, ok := node.Right.(*ast.NumberLiteral); ok {
			return &ast.NumberLiteral{
				Token: node.Token,
				Value: -numLit.Value,
			}
		}
	}

	return node
}

// formatNumber formats a number for output
func formatNumber(n float64) string {
	// If it's an integer, format without decimal point
	if n == float64(int(n)) {
		return fmt.Sprintf("%d", int(n))
	}
	return fmt.Sprintf("%g", n)
}
