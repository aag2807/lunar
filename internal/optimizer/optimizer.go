package optimizer

import (
	"lunar/internal/ast"
	"strconv"
)

// Options controls optimization behavior
type Options struct {
	ConstantFolding    bool
	DeadCodeElimination bool
	RemoveUnusedLocals bool
}

// DefaultOptions returns default optimization options
func DefaultOptions() Options {
	return Options{
		ConstantFolding:    true,
		DeadCodeElimination: true,
		RemoveUnusedLocals: false, // Conservative default
	}
}

// Optimizer performs AST-level optimizations
type Optimizer struct {
	opts Options
}

// New creates a new optimizer
func New(opts Options) *Optimizer {
	return &Optimizer{opts: opts}
}

// Optimize optimizes a list of statements
func (o *Optimizer) Optimize(statements []ast.Statement) []ast.Statement {
	if o.opts.DeadCodeElimination {
		statements = o.removeDeadCode(statements)
	}

	if o.opts.ConstantFolding {
		statements = o.foldConstants(statements)
	}

	return statements
}

// removeDeadCode removes unreachable code after return statements
func (o *Optimizer) removeDeadCode(statements []ast.Statement) []ast.Statement {
	result := make([]ast.Statement, 0, len(statements))

	for _, stmt := range statements {
		result = append(result, stmt)

		// Stop after return statement at top level
		if _, isReturn := stmt.(*ast.ReturnStatement); isReturn {
			// Any code after return is unreachable
			break
		}

		// Process function bodies
		if fnDecl, ok := stmt.(*ast.FunctionDeclaration); ok {
			if fnDecl.Body != nil {
				fnDecl.Body.Statements = o.optimizeFunctionBody(fnDecl.Body.Statements)
			}
		}

		// Process class methods
		if classDecl, ok := stmt.(*ast.ClassDeclaration); ok {
			for _, method := range classDecl.Methods {
				if method.Body != nil {
					method.Body.Statements = o.optimizeFunctionBody(method.Body.Statements)
				}
			}
		}
	}

	return result
}

// optimizeFunctionBody removes dead code in a function body
func (o *Optimizer) optimizeFunctionBody(body []ast.Statement) []ast.Statement {
	result := make([]ast.Statement, 0, len(body))

	for _, stmt := range body {
		result = append(result, stmt)

		// Stop after return
		if _, isReturn := stmt.(*ast.ReturnStatement); isReturn {
			break
		}
	}

	return result
}

// foldConstants performs constant folding on expressions
func (o *Optimizer) foldConstants(statements []ast.Statement) []ast.Statement {
	for i, stmt := range statements {
		statements[i] = o.foldConstantsInStatement(stmt)
	}
	return statements
}

// foldConstantsInStatement folds constants in a statement
func (o *Optimizer) foldConstantsInStatement(stmt ast.Statement) ast.Statement {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		for i, val := range s.Values {
			s.Values[i] = o.foldConstantsInExpression(val)
		}

	case *ast.FunctionDeclaration:
		if s.Body != nil {
			for i, bodyStmt := range s.Body.Statements {
				s.Body.Statements[i] = o.foldConstantsInStatement(bodyStmt)
			}
		}

	case *ast.ReturnStatement:
		for i, val := range s.ReturnValues {
			s.ReturnValues[i] = o.foldConstantsInExpression(val)
		}

	case *ast.ExpressionStatement:
		s.Expression = o.foldConstantsInExpression(s.Expression)
	}

	return stmt
}

// foldConstantsInExpression folds constant expressions
func (o *Optimizer) foldConstantsInExpression(expr ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.InfixExpression:
		// Fold operands first
		e.Left = o.foldConstantsInExpression(e.Left)
		e.Right = o.foldConstantsInExpression(e.Right)

		// Try to fold if both operands are literals
		leftNum, leftIsNum := e.Left.(*ast.NumberLiteral)
		rightNum, rightIsNum := e.Right.(*ast.NumberLiteral)

		if leftIsNum && rightIsNum {
			return o.foldNumericOperation(leftNum, rightNum, e.Operator)
		}

		// String concatenation
		leftStr, leftIsStr := e.Left.(*ast.StringLiteral)
		rightStr, rightIsStr := e.Right.(*ast.StringLiteral)

		if leftIsStr && rightIsStr && e.Operator == ".." {
			return &ast.StringLiteral{
				Token: leftStr.Token,
				Value: leftStr.Value + rightStr.Value,
			}
		}

	case *ast.PrefixExpression:
		e.Right = o.foldConstantsInExpression(e.Right)

		// Fold -number
		if e.Operator == "-" {
			if numLit, ok := e.Right.(*ast.NumberLiteral); ok {
				return &ast.NumberLiteral{
					Token: numLit.Token,
					Value: -numLit.Value,
				}
			}
		}

		// Fold not true/false
		if e.Operator == "not" {
			if boolLit, ok := e.Right.(*ast.BooleanLiteral); ok {
				return &ast.BooleanLiteral{
					Token: boolLit.Token,
					Value: !boolLit.Value,
				}
			}
		}

	case *ast.CallExpression:
		// Fold arguments
		for i, arg := range e.Arguments {
			e.Arguments[i] = o.foldConstantsInExpression(arg)
		}
	}

	return expr
}

// foldNumericOperation folds arithmetic operations on number literals
func (o *Optimizer) foldNumericOperation(left, right *ast.NumberLiteral, operator string) ast.Expression {
	var result float64

	switch operator {
	case "+":
		result = left.Value + right.Value
	case "-":
		result = left.Value - right.Value
	case "*":
		result = left.Value * right.Value
	case "/":
		if right.Value == 0 {
			// Don't fold division by zero
			return &ast.InfixExpression{
				Token:    left.Token,
				Left:     left,
				Operator: operator,
				Right:    right,
			}
		}
		result = left.Value / right.Value
	case "%":
		if right.Value == 0 {
			// Don't fold modulo by zero
			return &ast.InfixExpression{
				Token:    left.Token,
				Left:     left,
				Operator: operator,
				Right:    right,
			}
		}
		result = float64(int(left.Value) % int(right.Value))
	case "^":
		// For simplicity, don't fold power operations
		return &ast.InfixExpression{
			Token:    left.Token,
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	default:
		// Can't fold this operator
		return &ast.InfixExpression{
			Token:    left.Token,
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return &ast.NumberLiteral{
		Token: left.Token,
		Value: result,
	}
}

// Stats returns optimization statistics
type Stats struct {
	ConstantsFolded  int
	DeadCodeRemoved  int
	UnusedLocals     int
}

// String formats stats as a string
func (s Stats) String() string {
	return "Optimizations: " + strconv.Itoa(s.ConstantsFolded) + " constants folded, " +
		strconv.Itoa(s.DeadCodeRemoved) + " dead code blocks removed, " +
		strconv.Itoa(s.UnusedLocals) + " unused locals removed"
}
