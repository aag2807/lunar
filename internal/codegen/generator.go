package codegen

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/sourcemap"
	"strings"
)

// Generator generates Lua code from an AST
type Generator struct {
	indent           int
	sourceMapBuilder *sourcemap.Builder
	currentLine      int
	currentColumn    int
	sourceFile       string
	classes          map[string]bool // Track defined classes for constructor calls
}

// New creates a new code generator
func New() *Generator {
	return &Generator{
		indent:  0,
		classes: make(map[string]bool),
	}
}

// NewWithSourceMap creates a new code generator with source map support
func NewWithSourceMap(sourceFile, generatedFile string) *Generator {
	return &Generator{
		indent:           0,
		sourceMapBuilder: sourcemap.NewBuilder(sourceFile, generatedFile),
		currentLine:      1,
		currentColumn:    0,
		sourceFile:       sourceFile,
		classes:          make(map[string]bool),
	}
}

// GetSourceMap returns the built source map (if enabled)
func (g *Generator) GetSourceMap() *sourcemap.SourceMap {
	if g.sourceMapBuilder == nil {
		return nil
	}
	return g.sourceMapBuilder.Build()
}

// trackMapping adds a source mapping if source maps are enabled
func (g *Generator) trackMapping(sourceToken lexer.Token) {
	if g.sourceMapBuilder == nil {
		return
	}

	// Add mapping from current generated position to source position
	g.sourceMapBuilder.AddMapping(
		g.currentLine,
		g.currentColumn,
		sourceToken.Line,
		sourceToken.Column-1, // Source maps use 0-based columns
		"",
	)
}

// write outputs text and updates position tracking
func (g *Generator) write(text string) {
	if g.sourceMapBuilder == nil {
		return // Position tracking not needed without source maps
	}

	for _, char := range text {
		if char == '\n' {
			g.currentLine++
			g.currentColumn = 0
		} else {
			g.currentColumn++
		}
	}
}

// Generate generates Lua code from a list of statements
func (g *Generator) Generate(statements []ast.Statement) string {
	var output strings.Builder

	for i, stmt := range statements {
		// Track mapping at the start of each statement
		if g.sourceMapBuilder != nil {
			g.trackStatementMapping(stmt)
		}

		code := g.generateStatement(stmt)
		if code != "" {
			g.write(code)
			output.WriteString(code)
			// Add blank line between top-level declarations
			if i < len(statements)-1 {
				g.write("\n")
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

// trackStatementMapping tracks the source mapping for a statement
func (g *Generator) trackStatementMapping(stmt ast.Statement) {
	if stmt == nil {
		return
	}

	// Get the token from the statement
	var token lexer.Token
	switch node := stmt.(type) {
	case *ast.VariableDeclaration:
		token = node.Token
	case *ast.FunctionDeclaration:
		token = node.Token
	case *ast.ExpressionStatement:
		if node.Expression != nil {
			token = g.getExpressionToken(node.Expression)
		}
	case *ast.ReturnStatement:
		token = node.Token
	case *ast.IfStatement:
		token = node.Token
	case *ast.WhileStatement:
		token = node.Token
	case *ast.ForStatement:
		token = node.Token
	case *ast.DoStatement:
		token = node.Token
	case *ast.BreakStatement:
		token = node.Token
	case *ast.AssignmentStatement:
		if node.Name != nil {
			token = g.getExpressionToken(node.Name)
		}
	case *ast.ClassDeclaration:
		token = node.Token
	case *ast.EnumDeclaration:
		token = node.Token
	case *ast.ExportStatement:
		token = node.Token
	case *ast.ImportStatement:
		token = node.Token
	default:
		return
	}

	g.trackMapping(token)
}

// getExpressionToken gets the token from an expression
func (g *Generator) getExpressionToken(expr ast.Expression) lexer.Token {
	switch node := expr.(type) {
	case *ast.Identifier:
		return node.Token
	case *ast.NumberLiteral:
		return node.Token
	case *ast.StringLiteral:
		return node.Token
	case *ast.BooleanLiteral:
		return node.Token
	case *ast.CallExpression:
		return g.getExpressionToken(node.Function)
	case *ast.InfixExpression:
		return g.getExpressionToken(node.Left)
	case *ast.PrefixExpression:
		return node.Token
	case *ast.DotExpression:
		return g.getExpressionToken(node.Left)
	case *ast.IndexExpression:
		return g.getExpressionToken(node.Left)
	default:
		return lexer.Token{}
	}
}

// generateStatement generates Lua code for a statement
func (g *Generator) generateStatement(stmt ast.Statement) string {
	if stmt == nil {
		return ""
	}

	switch node := stmt.(type) {
	case *ast.VariableDeclaration:
		return g.generateVariableDeclaration(node)
	case *ast.FunctionDeclaration:
		return g.generateFunctionDeclaration(node)
	case *ast.ExpressionStatement:
		return g.generateIndent() + g.generateExpression(node.Expression) + "\n"
	case *ast.ReturnStatement:
		return g.generateReturnStatement(node)
	case *ast.IfStatement:
		return g.generateIfStatement(node)
	case *ast.WhileStatement:
		return g.generateWhileStatement(node)
	case *ast.ForStatement:
		return g.generateForStatement(node)
	case *ast.DoStatement:
		return g.generateDoStatement(node)
	case *ast.BreakStatement:
		return g.generateIndent() + "break\n"
	case *ast.BlockStatement:
		return g.generateBlockStatement(node)
	case *ast.AssignmentStatement:
		return g.generateAssignmentStatement(node)
	case *ast.ClassDeclaration:
		return g.generateClassDeclaration(node)
	case *ast.InterfaceDeclaration:
		// Interfaces are type-only, don't generate code
		return ""
	case *ast.EnumDeclaration:
		return g.generateEnumDeclaration(node)
	case *ast.TypeDeclaration:
		// Type aliases are type-only, don't generate code
		return ""
	case *ast.ExportStatement:
		return g.generateExportStatement(node)
	case *ast.ImportStatement:
		return g.generateImportStatement(node)
	default:
		return ""
	}
}

// generateVariableDeclaration generates code for a variable declaration
func (g *Generator) generateVariableDeclaration(node *ast.VariableDeclaration) string {
	var output strings.Builder
	output.WriteString(g.generateIndent())
	output.WriteString("local ")
	output.WriteString(node.Name.Value)

	if node.Value != nil {
		output.WriteString(" = ")
		output.WriteString(g.generateExpression(node.Value))
	}

	output.WriteString("\n")
	return output.String()
}

// generateFunctionDeclaration generates code for a function declaration
func (g *Generator) generateFunctionDeclaration(node *ast.FunctionDeclaration) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("function ")
	output.WriteString(node.Name.Value)
	output.WriteString("(")

	// Parameters (without type annotations)
	params := make([]string, len(node.Parameters))
	for i, param := range node.Parameters {
		params[i] = param.Name.Value
	}
	output.WriteString(strings.Join(params, ", "))
	output.WriteString(")\n")

	// Body
	g.indent++
	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	return output.String()
}

// generateReturnStatement generates code for a return statement
func (g *Generator) generateReturnStatement(node *ast.ReturnStatement) string {
	var output strings.Builder
	output.WriteString(g.generateIndent())
	output.WriteString("return")

	if node.ReturnValue != nil {
		output.WriteString(" ")
		output.WriteString(g.generateExpression(node.ReturnValue))
	}

	output.WriteString("\n")
	return output.String()
}

// generateIfStatement generates code for an if statement
func (g *Generator) generateIfStatement(node *ast.IfStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("if ")
	output.WriteString(g.generateExpression(node.Condition))
	output.WriteString(" then\n")

	// Consequence
	g.indent++
	for _, stmt := range node.Consequence.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	// Alternative (else)
	if node.Alternative != nil {
		output.WriteString(g.generateIndent())
		output.WriteString("else\n")

		g.indent++
		for _, stmt := range node.Alternative.Statements {
			output.WriteString(g.generateStatement(stmt))
		}
		g.indent--
	}

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	return output.String()
}

// generateWhileStatement generates code for a while statement
func (g *Generator) generateWhileStatement(node *ast.WhileStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("while ")
	output.WriteString(g.generateExpression(node.Condition))
	output.WriteString(" do\n")

	g.indent++
	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	return output.String()
}

// generateForStatement generates code for a for statement
func (g *Generator) generateForStatement(node *ast.ForStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("for ")
	output.WriteString(node.Variable.Value)

	if node.IsGeneric {
		// Generic for loop: for k, v in pairs(table) do
		output.WriteString(" in ")
		output.WriteString(g.generateExpression(node.Iterator))
	} else {
		// Numeric for loop: for i = start, end, step do
		output.WriteString(" = ")
		output.WriteString(g.generateExpression(node.Start))
		output.WriteString(", ")
		output.WriteString(g.generateExpression(node.End))

		if node.Step != nil {
			output.WriteString(", ")
			output.WriteString(g.generateExpression(node.Step))
		}
	}

	output.WriteString(" do\n")

	g.indent++
	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	return output.String()
}

// generateDoStatement generates code for a do statement
func (g *Generator) generateDoStatement(node *ast.DoStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("do\n")

	g.indent++
	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	return output.String()
}

// generateBlockStatement generates code for a block statement
func (g *Generator) generateBlockStatement(node *ast.BlockStatement) string {
	var output strings.Builder

	for _, stmt := range node.Statements {
		output.WriteString(g.generateStatement(stmt))
	}

	return output.String()
}

// generateAssignmentStatement generates code for an assignment
func (g *Generator) generateAssignmentStatement(node *ast.AssignmentStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString(g.generateExpression(node.Name))
	output.WriteString(" = ")
	output.WriteString(g.generateExpression(node.Value))
	output.WriteString("\n")

	return output.String()
}

// generateClassDeclaration generates code for a class (transpiled to Lua table with metatable)
func (g *Generator) generateClassDeclaration(node *ast.ClassDeclaration) string {
	var output strings.Builder
	className := node.Name.Value

	// Track this class for constructor calls
	g.classes[className] = true

	// Create class table
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("local %s = {}\n", className))
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("%s.__index = %s\n", className, className))
	output.WriteString("\n")

	// Generate constructor as new() function
	if node.Constructor != nil {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s.new(", className))

		params := make([]string, len(node.Constructor.Parameters))
		for i, param := range node.Constructor.Parameters {
			params[i] = param.Name.Value
		}
		output.WriteString(strings.Join(params, ", "))
		output.WriteString(")\n")

		g.indent++
		output.WriteString(g.generateIndent())
		output.WriteString("local self = setmetatable({}, " + className + ")\n")

		// Initialize properties from constructor body
		for _, stmt := range node.Constructor.Body.Statements {
			output.WriteString(g.generateStatement(stmt))
		}

		output.WriteString(g.generateIndent())
		output.WriteString("return self\n")
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		output.WriteString("\n")
	}

	// Generate methods
	for _, method := range node.Methods {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s:%s(", className, method.Name.Value))

		params := make([]string, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = param.Name.Value
		}
		output.WriteString(strings.Join(params, ", "))
		output.WriteString(")\n")

		g.indent++
		for _, stmt := range method.Body.Statements {
			output.WriteString(g.generateStatement(stmt))
		}
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		output.WriteString("\n")
	}

	return output.String()
}

// generateEnumDeclaration generates code for an enum (transpiled to Lua table)
func (g *Generator) generateEnumDeclaration(node *ast.EnumDeclaration) string {
	var output strings.Builder
	enumName := node.Name.Value

	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("local %s = {\n", enumName))

	g.indent++
	for i, member := range node.Members {
		output.WriteString(g.generateIndent())
		output.WriteString(member.Name.Value)
		output.WriteString(" = ")

		if member.Value != nil {
			output.WriteString(g.generateExpression(member.Value))
		} else {
			// Auto-increment starting from 0
			output.WriteString(fmt.Sprintf("%d", i))
		}

		output.WriteString(",\n")
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("}\n")

	return output.String()
}

// generateExpression generates code for an expression
func (g *Generator) generateExpression(expr ast.Expression) string {
	if expr == nil {
		return ""
	}

	switch node := expr.(type) {
	case *ast.Identifier:
		return node.Value
	case *ast.NumberLiteral:
		return node.Token.Literal
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", node.Value)
	case *ast.BooleanLiteral:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "nil"
	case *ast.TableLiteral:
		return g.generateTableLiteral(node)
	case *ast.PrefixExpression:
		return g.generatePrefixExpression(node)
	case *ast.InfixExpression:
		return g.generateInfixExpression(node)
	case *ast.CallExpression:
		return g.generateCallExpression(node)
	case *ast.DotExpression:
		return g.generateDotExpression(node)
	case *ast.IndexExpression:
		return g.generateIndexExpression(node)
	default:
		return ""
	}
}

// generateTableLiteral generates code for a table literal
func (g *Generator) generateTableLiteral(node *ast.TableLiteral) string {
	var output strings.Builder
	output.WriteString("{")

	// Generate array-style values
	if len(node.Values) > 0 {
		values := make([]string, len(node.Values))
		for i, val := range node.Values {
			values[i] = g.generateExpression(val)
		}
		output.WriteString(strings.Join(values, ", "))
	}

	// Generate key-value pairs
	if len(node.Pairs) > 0 {
		if len(node.Values) > 0 {
			output.WriteString(", ")
		}

		pairs := []string{}
		for key, val := range node.Pairs {
			keyStr := g.generateExpression(key)
			valStr := g.generateExpression(val)
			pairs = append(pairs, fmt.Sprintf("[%s] = %s", keyStr, valStr))
		}
		output.WriteString(strings.Join(pairs, ", "))
	}

	output.WriteString("}")
	return output.String()
}

// generatePrefixExpression generates code for a prefix expression
func (g *Generator) generatePrefixExpression(node *ast.PrefixExpression) string {
	operator := node.Operator
	right := g.generateExpression(node.Right)

	// Convert 'not' to Lua 'not'
	if operator == "!" {
		operator = "not"
	}

	// Only add parentheses if the right side is a complex expression
	if needsParentheses(node.Right) {
		return fmt.Sprintf("%s (%s)", operator, right)
	}
	return fmt.Sprintf("%s %s", operator, right)
}

// generateInfixExpression generates code for an infix expression
func (g *Generator) generateInfixExpression(node *ast.InfixExpression) string {
	left := g.generateExpression(node.Left)
	operator := node.Operator
	right := g.generateExpression(node.Right)

	// Convert operators to Lua equivalents
	switch operator {
	case "!=":
		operator = "~="
	case "&&":
		operator = "and"
	case "||":
		operator = "or"
	}

	// Smart parenthesization based on operator precedence
	leftNeedsParens := needsParensInInfix(node.Left, operator, true)
	rightNeedsParens := needsParensInInfix(node.Right, operator, false)

	if leftNeedsParens {
		left = "(" + left + ")"
	}
	if rightNeedsParens {
		right = "(" + right + ")"
	}

	return fmt.Sprintf("%s %s %s", left, operator, right)
}

// generateCallExpression generates code for a function call
func (g *Generator) generateCallExpression(node *ast.CallExpression) string {
	function := g.generateExpression(node.Function)

	// Check if calling a class constructor (simple identifier that's a known class)
	if ident, ok := node.Function.(*ast.Identifier); ok {
		if g.classes[ident.Value] {
			function = function + ".new"
		}
	}

	args := make([]string, len(node.Arguments))
	for i, arg := range node.Arguments {
		args[i] = g.generateExpression(arg)
	}

	return fmt.Sprintf("%s(%s)", function, strings.Join(args, ", "))
}

// generateDotExpression generates code for a dot expression
func (g *Generator) generateDotExpression(node *ast.DotExpression) string {
	left := g.generateExpression(node.Left)
	right := g.generateExpression(node.Right)

	return fmt.Sprintf("%s.%s", left, right)
}

// generateIndexExpression generates code for an index expression
func (g *Generator) generateIndexExpression(node *ast.IndexExpression) string {
	left := g.generateExpression(node.Left)
	index := g.generateExpression(node.Index)

	return fmt.Sprintf("%s[%s]", left, index)
}

// generateIndent generates the current indentation
func (g *Generator) generateIndent() string {
	return strings.Repeat("    ", g.indent)
}

// generateExportStatement generates code for an export statement
func (g *Generator) generateExportStatement(node *ast.ExportStatement) string {
	// In Lua, exports are handled via return tables at the end of modules
	// For now, just generate the underlying statement without special export handling
	// The exported names should be collected and returned at module end
	return g.generateStatement(node.Statement)
}

// generateImportStatement generates code for an import statement
func (g *Generator) generateImportStatement(node *ast.ImportStatement) string {
	var output strings.Builder
	output.WriteString(g.generateIndent())

	if node.IsWildcard {
		// import * from "module" -> local module = require("module")
		// Extract module name from path (last part before extension)
		moduleName := node.Module
		// Simple heuristic: use the last part of the path as variable name
		parts := strings.Split(moduleName, "/")
		varName := strings.TrimSuffix(parts[len(parts)-1], ".lunar")
		output.WriteString(fmt.Sprintf("local %s = require(\"%s\")\n", varName, moduleName))
	} else {
		// import { name1, name2 } from "module"
		// -> local _module = require("module")
		// -> local name1 = _module.name1
		// -> local name2 = _module.name2
		tempVar := "_" + strings.ReplaceAll(node.Module, "/", "_")
		tempVar = strings.ReplaceAll(tempVar, ".", "_")

		output.WriteString(fmt.Sprintf("local %s = require(\"%s\")\n", tempVar, node.Module))

		for _, name := range node.Names {
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("local %s = %s.%s\n", name.Value, tempVar, name.Value))
		}
	}

	return output.String()
}

// Generate is the main entry point for code generation
// Note: Optimizations disabled by default in v1.0 (enabled in future versions)
func Generate(statements []ast.Statement) string {
	return GenerateWithOptions(statements, false)
}

// GenerateWithOptions generates Lua code with configurable optimization
func GenerateWithOptions(statements []ast.Statement, optimize bool) string {
	// Run optimizer if enabled
	if optimize {
		optimizer := NewOptimizer(true)
		statements = optimizer.OptimizeStatements(statements)
	}

	generator := New()
	return generator.Generate(statements)
}

// GenerateWithSourceMap generates Lua code and source map
func GenerateWithSourceMap(statements []ast.Statement, sourceFile, generatedFile string, optimize bool) (string, *sourcemap.SourceMap) {
	// Run optimizer if enabled
	if optimize {
		optimizer := NewOptimizer(true)
		statements = optimizer.OptimizeStatements(statements)
	}

	generator := NewWithSourceMap(sourceFile, generatedFile)
	code := generator.Generate(statements)
	sourceMap := generator.GetSourceMap()

	return code, sourceMap
}

// needsParentheses determines if an expression needs parentheses
func needsParentheses(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.InfixExpression, *ast.PrefixExpression:
		return true
	default:
		return false
	}
}

// needsParensInInfix determines if parentheses are needed for an operand in an infix expression
func needsParensInInfix(expr ast.Expression, parentOp string, isLeft bool) bool {
	infixExpr, ok := expr.(*ast.InfixExpression)
	if !ok {
		return false
	}

	childOp := infixExpr.Operator
	parentPrec := getOperatorPrecedence(parentOp)
	childPrec := getOperatorPrecedence(childOp)

	// Need parentheses if child has lower precedence
	if childPrec < parentPrec {
		return true
	}

	// For same precedence, need parentheses on right for non-associative/right-associative operators
	if childPrec == parentPrec && !isLeft {
		// Most operators in Lua are left-associative, so right operand needs parentheses
		// Exception: power operator ^ is right-associative
		if parentOp != "^" {
			return true
		}
	}

	return false
}

// getOperatorPrecedence returns the precedence level of an operator (higher = tighter binding)
func getOperatorPrecedence(op string) int {
	switch op {
	case "or", "||":
		return 1
	case "and", "&&":
		return 2
	case "<", ">", "<=", ">=", "~=", "!=", "==":
		return 3
	case "..":
		return 4
	case "+", "-":
		return 5
	case "*", "/", "%":
		return 6
	case "not", "!", "unary-":
		return 7
	case "^":
		return 8
	default:
		return 0
	}
}
