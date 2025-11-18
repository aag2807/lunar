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
	classes          map[string]bool   // Track defined classes for constructor calls
	classParents     map[string]string // Track parent class names for super
	currentClassName string            // Current class being generated
	exports          []string          // Track exported names for module generation
}

// New creates a new code generator
func New() *Generator {
	return &Generator{
		indent:       0,
		classes:      make(map[string]bool),
		classParents: make(map[string]string),
		exports:      make([]string, 0),
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
		classParents:     make(map[string]string),
		exports:          make([]string, 0),
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

	// If there are exports, generate a return statement
	if len(g.exports) > 0 {
		g.write("\n\n")
		output.WriteString("\n\n")
		returnStmt := g.generateModuleReturn()
		g.write(returnStmt)
		output.WriteString(returnStmt)
	}

	return output.String()
}

// generateModuleReturn generates the module return statement for exports
func (g *Generator) generateModuleReturn() string {
	if len(g.exports) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("return {\n")

	for i, name := range g.exports {
		output.WriteString(fmt.Sprintf("    %s = %s", name, name))
		if i < len(g.exports)-1 {
			output.WriteString(",")
		}
		output.WriteString("\n")
	}

	output.WriteString("}\n")
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
	case *ast.TemplateLiteral:
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

	// Write all variable names
	for i, name := range node.Names {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(name.Value)
	}

	// Write values if present
	if len(node.Values) > 0 {
		output.WriteString(" = ")
		for i, val := range node.Values {
			if i > 0 {
				output.WriteString(", ")
			}
			output.WriteString(g.generateExpression(val))
		}
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
	params := make([]string, 0, len(node.Parameters))
	var restParam *ast.Parameter
	for i, param := range node.Parameters {
		if param.IsRest {
			restParam = param
			// Add ... to parameter list
			params = append(params, "...")
			break
		}
		params = append(params, param.Name.Value)
		// Check if next param is rest, if so don't include regular params after
		if i+1 < len(node.Parameters) && node.Parameters[i+1].IsRest {
			restParam = node.Parameters[i+1]
			params = append(params, "...")
			break
		}
	}
	output.WriteString(strings.Join(params, ", "))
	output.WriteString(")\n")

	// Body
	g.indent++

	// If there's a rest parameter, pack the varargs into a table
	if restParam != nil {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("local %s = {...}\n", restParam.Name.Value))
	}

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

	if len(node.ReturnValues) > 0 {
		output.WriteString(" ")
		for i, val := range node.ReturnValues {
			if i > 0 {
				output.WriteString(", ")
			}
			output.WriteString(g.generateExpression(val))
		}
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

	// Write all loop variables
	for i, v := range node.Variables {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(v.Value)
	}

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

	// Track parent class name for super
	if node.Extends != nil {
		if parentIdent, ok := node.Extends.(*ast.Identifier); ok {
			g.classParents[className] = parentIdent.Value
		}
	}

	// Set current class context
	prevClassName := g.currentClassName
	g.currentClassName = className
	defer func() { g.currentClassName = prevClassName }()

	// Create class table
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("local %s = {}\n", className))

	// Set up inheritance if there's a parent class
	if node.Extends != nil {
		if parentIdent, ok := node.Extends.(*ast.Identifier); ok {
			parentName := parentIdent.Value
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("setmetatable(%s, {__index = %s})\n", className, parentName))
		}
	}

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

	// Generate static properties
	for _, prop := range node.Properties {
		if prop.IsStatic {
			// Static properties go directly on the class table
			// They would need initialization in constructor or elsewhere
			// For now, we'll just declare them as nil (can be set later)
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s.%s = nil\n", className, prop.Name.Value))
		}
	}
	if len(node.Properties) > 0 {
		hasStatic := false
		for _, prop := range node.Properties {
			if prop.IsStatic {
				hasStatic = true
				break
			}
		}
		if hasStatic {
			output.WriteString("\n")
		}
	}

	// Generate methods (both static and instance)
	for _, method := range node.Methods {
		// Skip abstract methods (no implementation)
		if method.IsAbstract {
			continue
		}

		output.WriteString(g.generateIndent())
		if method.IsStatic {
			// Static methods use dot notation (no self parameter)
			output.WriteString(fmt.Sprintf("function %s.%s(", className, method.Name.Value))
		} else {
			// Instance methods use colon notation (implicit self parameter)
			output.WriteString(fmt.Sprintf("function %s:%s(", className, method.Name.Value))
		}

		params := make([]string, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = param.Name.Value
		}
		output.WriteString(strings.Join(params, ", "))
		output.WriteString(")\n")

		g.indent++
		if method.Body != nil {
			for _, stmt := range method.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
		}
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		output.WriteString("\n")
	}

	// Generate getters
	for _, getter := range node.Getters {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s:_get_%s()\n", className, getter.Name.Value))

		g.indent++
		if getter.Body != nil {
			for _, stmt := range getter.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
		}
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		output.WriteString("\n")
	}

	// Generate setters
	for _, setter := range node.Setters {
		output.WriteString(g.generateIndent())
		paramName := "value"
		if setter.Parameter != nil {
			paramName = setter.Parameter.Name.Value
		}
		output.WriteString(fmt.Sprintf("function %s:_set_%s(%s)\n", className, setter.Name.Value, paramName))

		g.indent++
		if setter.Body != nil {
			for _, stmt := range setter.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
		}
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		output.WriteString("\n")
	}

	// Generate metamethod dispatching for getters/setters if any exist
	if len(node.Getters) > 0 || len(node.Setters) > 0 {
		// Generate __index metamethod for getters
		if len(node.Getters) > 0 {
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("local %s_mt = getmetatable(%s) or {}\n", className, className))
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s_mt.__index = function(self, key)\n", className))
			g.indent++

			for _, getter := range node.Getters {
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("if key == \"%s\" then\n", getter.Name.Value))
				g.indent++
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("return %s._get_%s(self)\n", className, getter.Name.Value))
				g.indent--
				output.WriteString(g.generateIndent())
				output.WriteString("end\n")
			}

			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("return %s[key]\n", className))
			g.indent--
			output.WriteString(g.generateIndent())
			output.WriteString("end\n")
		}

		// Generate __newindex metamethod for setters
		if len(node.Setters) > 0 {
			if len(node.Getters) == 0 {
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("local %s_mt = getmetatable(%s) or {}\n", className, className))
			}
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s_mt.__newindex = function(self, key, value)\n", className))
			g.indent++

			for _, setter := range node.Setters {
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("if key == \"%s\" then\n", setter.Name.Value))
				g.indent++
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("%s._set_%s(self, value)\n", className, setter.Name.Value))
				output.WriteString(g.generateIndent())
				output.WriteString("return\n")
				g.indent--
				output.WriteString(g.generateIndent())
				output.WriteString("end\n")
			}

			output.WriteString(g.generateIndent())
			output.WriteString("rawset(self, key, value)\n")
			g.indent--
			output.WriteString(g.generateIndent())
			output.WriteString("end\n")
		}

		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("setmetatable(%s, %s_mt)\n", className, className))
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
	case *ast.SuperExpression:
		// Generate parent class name
		if g.currentClassName != "" {
			if parentName, ok := g.classParents[g.currentClassName]; ok {
				return parentName
			}
		}
		// Fallback (should not happen if type checker passes)
		return "super"
	case *ast.NumberLiteral:
		return node.Token.Literal
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", node.Value)
	case *ast.TemplateLiteral:
		return g.generateTemplateLiteral(node)
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
	case *ast.FunctionExpression:
		return g.generateFunctionExpression(node)
	case *ast.TypeAssertion:
		// Type assertions are compile-time only, just return the expression
		return g.generateExpression(node.Expression)
	default:
		return ""
	}
}

// generateTableLiteral generates code for a table literal
func (g *Generator) generateTableLiteral(node *ast.TableLiteral) string {
	// Check if any value is a spread expression
	hasSpread := false
	for _, val := range node.Values {
		if _, ok := val.(*ast.SpreadExpression); ok {
			hasSpread = true
			break
		}
	}

	// If there are spread expressions in array values, we need special handling
	if hasSpread {
		var output strings.Builder
		output.WriteString("(function() local __temp = {}; ")

		for _, val := range node.Values {
			if spread, ok := val.(*ast.SpreadExpression); ok {
				// Insert all elements from the spread array
				spreadValue := g.generateExpression(spread.Value)
				output.WriteString(fmt.Sprintf("for _, __v in ipairs(%s) do table.insert(__temp, __v) end; ", spreadValue))
			} else {
				// Insert single element
				output.WriteString(fmt.Sprintf("table.insert(__temp, %s); ", g.generateExpression(val)))
			}
		}

		output.WriteString("return __temp end)()")
		return output.String()
	}

	// No spread - use normal table literal generation
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

	// Handle nullish coalescing operator (??)
	if operator == "??" {
		// Generate: (function() local __temp = left; if __temp ~= nil then return __temp else return right end end)()
		return fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp else return %s end end)()", left, right)
	}

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

	// Check if any argument is a spread expression
	hasSpread := false
	for _, arg := range node.Arguments {
		if _, ok := arg.(*ast.SpreadExpression); ok {
			hasSpread = true
			break
		}
	}

	// If there are spread arguments, we need special handling
	if hasSpread {
		// Simple case: single spread argument
		if len(node.Arguments) == 1 {
			if spread, ok := node.Arguments[0].(*ast.SpreadExpression); ok {
				spreadValue := g.generateExpression(spread.Value)
				return fmt.Sprintf("%s(table.unpack(%s))", function, spreadValue)
			}
		}

		// Complex case: mixed regular and spread arguments
		// Build a table with all args and unpack it
		var tableBuilder strings.Builder
		tableBuilder.WriteString("{")
		for i, arg := range node.Arguments {
			if i > 0 {
				tableBuilder.WriteString(", ")
			}
			if spread, ok := arg.(*ast.SpreadExpression); ok {
				// Use table.unpack to spread the array
				tableBuilder.WriteString(fmt.Sprintf("table.unpack(%s)", g.generateExpression(spread.Value)))
			} else {
				tableBuilder.WriteString(g.generateExpression(arg))
			}
		}
		tableBuilder.WriteString("}")
		return fmt.Sprintf("%s(table.unpack(%s))", function, tableBuilder.String())
	}

	// No spread arguments - normal case
	args := make([]string, len(node.Arguments))
	for i, arg := range node.Arguments {
		args[i] = g.generateExpression(arg)
	}

	return fmt.Sprintf("%s(%s)", function, strings.Join(args, ", "))
}

// generateTemplateLiteral generates code for a template literal
func (g *Generator) generateTemplateLiteral(node *ast.TemplateLiteral) string {
	// Template literals are converted to Lua string concatenation
	// `Hello ${name}` becomes "Hello " .. tostring(name)

	if len(node.Parts) == 1 && len(node.Expressions) == 0 {
		// No interpolation, just a plain string
		return fmt.Sprintf("\"%s\"", node.Parts[0])
	}

	var parts []string
	for i, part := range node.Parts {
		// Add the string part if it's not empty
		if part != "" {
			parts = append(parts, fmt.Sprintf("\"%s\"", part))
		}

		// Add the expression (converted to string) if there's one at this position
		if i < len(node.Expressions) {
			exprCode := g.generateExpression(node.Expressions[i])
			// Use tostring() to convert the expression to a string
			parts = append(parts, fmt.Sprintf("tostring(%s)", exprCode))
		}
	}

	// Filter out empty parts
	nonEmptyParts := []string{}
	for _, part := range parts {
		if part != "" && part != "\"\"" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}

	if len(nonEmptyParts) == 0 {
		return "\"\""
	}

	if len(nonEmptyParts) == 1 {
		return nonEmptyParts[0]
	}

	// Concatenate all parts with ..
	return strings.Join(nonEmptyParts, " .. ")
}

// generateDotExpression generates code for a dot expression
func (g *Generator) generateDotExpression(node *ast.DotExpression) string {
	left := g.generateExpression(node.Left)
	right := g.generateExpression(node.Right)

	// Handle optional chaining (?.)
	if node.IsOptional {
		// Generate safe navigation: (function() if left ~= nil then return left.right else return nil end end)()
		// For efficiency, use a simpler pattern when left is a simple identifier
		return fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp.%s else return nil end end)()", left, right)
	}

	return fmt.Sprintf("%s.%s", left, right)
}

// generateIndexExpression generates code for an index expression
func (g *Generator) generateIndexExpression(node *ast.IndexExpression) string {
	left := g.generateExpression(node.Left)
	index := g.generateExpression(node.Index)

	return fmt.Sprintf("%s[%s]", left, index)
}

// generateFunctionExpression generates code for an anonymous function expression
func (g *Generator) generateFunctionExpression(node *ast.FunctionExpression) string {
	var output strings.Builder

	output.WriteString("function(")

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
	output.WriteString("end")

	return output.String()
}

// generateIndent generates the current indentation
func (g *Generator) generateIndent() string {
	return strings.Repeat("    ", g.indent)
}

// generateExportStatement generates code for an export statement
func (g *Generator) generateExportStatement(node *ast.ExportStatement) string {
	// In Lua, exports are handled via return tables at the end of modules
	// Generate the underlying statement and track exported names
	code := g.generateStatement(node.Statement)

	// Track what's being exported
	switch stmt := node.Statement.(type) {
	case *ast.VariableDeclaration:
		// Export variables
		for _, name := range stmt.Names {
			g.exports = append(g.exports, name.Value)
		}
	case *ast.FunctionDeclaration:
		// Export functions
		g.exports = append(g.exports, stmt.Name.Value)
	case *ast.ClassDeclaration:
		// Export classes
		g.exports = append(g.exports, stmt.Name.Value)
	case *ast.EnumDeclaration:
		// Export enums
		g.exports = append(g.exports, stmt.Name.Value)
	case *ast.TypeDeclaration:
		// Type declarations don't generate runtime code, skip tracking
	case *ast.InterfaceDeclaration:
		// Interface declarations don't generate runtime code, skip tracking
	}

	return code
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
