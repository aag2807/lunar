package codegen

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/sourcemap"
	"sort"
	"strings"
)

// Generator generates Lua code from an AST
type Generator struct {
	indent           int
	sourceMapBuilder *sourcemap.Builder
	currentLine      int
	currentColumn    int
	sourceFile       string
	classes          map[string]bool              // Track defined classes for constructor calls
	classParents     map[string]string            // Track parent class names for super
	staticMethods    map[string]map[string]bool   // Track static methods: class -> method -> isStatic
	currentClassName string                       // Current class being generated
	exports          []string                     // Track exported names for module generation
	target           string                       // Target Lua version: lua51, lua52, lua53, lua54, luajit
	selfReceivers    []map[string]bool            // Scoped set of locals whose methods take an implicit self
}

// New creates a new code generator
func New() *Generator {
	return &Generator{
		indent:        0,
		classes:       make(map[string]bool),
		classParents:  make(map[string]string),
		staticMethods: make(map[string]map[string]bool),
		exports:       make([]string, 0),
		target:        "lua53",
		selfReceivers: []map[string]bool{make(map[string]bool)},
	}
}

// NewWithTarget creates a new code generator with specified target
func NewWithTarget(target string) *Generator {
	if target == "" {
		target = "lua53"
	}
	return &Generator{
		indent:        0,
		classes:       make(map[string]bool),
		classParents:  make(map[string]string),
		staticMethods: make(map[string]map[string]bool),
		exports:       make([]string, 0),
		target:        target,
		selfReceivers: []map[string]bool{make(map[string]bool)},
	}
}

// IsLuaJIT returns true if targeting LuaJIT or Lua 5.1
func (g *Generator) IsLuaJIT() bool {
	return g.target == "luajit" || g.target == "lua51"
}

// NewWithSourceMap creates a new code generator with source map support
func NewWithSourceMap(sourceFile, generatedFile string) *Generator {
	return NewWithSourceMapAndTarget(sourceFile, generatedFile, "")
}

// NewWithSourceMapAndTarget creates a new code generator with source map support and target
func NewWithSourceMapAndTarget(sourceFile, generatedFile string, target string) *Generator {
	if target == "" {
		target = "lua53"
	}
	return &Generator{
		indent:           0,
		sourceMapBuilder: sourcemap.NewBuilder(sourceFile, generatedFile),
		currentLine:      1,
		currentColumn:    0,
		sourceFile:       sourceFile,
		classes:          make(map[string]bool),
		classParents:     make(map[string]string),
		staticMethods:    make(map[string]map[string]bool),
		exports:          make([]string, 0),
		target:           target,
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
		if first := node.Name(); first != nil {
			token = g.getExpressionToken(first)
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
	case *ast.NamespaceDeclaration:
		return g.generateNamespaceDeclaration(node)
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
	g.trackDeclaredReceivers(node.Names, node.Types, node.Values)
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
// renderParameters returns the Lua parameter list for a signature along with
// the rest parameter that needs unpacking, if any. Lua's bare vararg ("...")
// is passed straight through and never packed into a local.
func (g *Generator) renderParameters(parameters []*ast.Parameter) (string, *ast.Parameter) {
	params := make([]string, 0, len(parameters))
	var restParam *ast.Parameter

	for _, param := range parameters {
		if param.IsRest {
			params = append(params, "...")
			if param.Name != nil && param.Name.Value != "..." {
				restParam = param
			}
			// Lua allows nothing after the vararg.
			break
		}

		if param.Name != nil {
			params = append(params, param.Name.Value)
		}
	}

	return strings.Join(params, ", "), restParam
}

func (g *Generator) generateFunctionDeclaration(node *ast.FunctionDeclaration) string {
	var output strings.Builder

	g.pushReceiverScope()
	defer g.popReceiverScope()
	g.trackParameterReceivers(node.Parameters)

	output.WriteString(g.generateIndent())
	output.WriteString("function ")
	output.WriteString(node.Name.Value)
	output.WriteString("(")

	// Parameters (without type annotations)
	paramList, restParam := g.renderParameters(node.Parameters)
	output.WriteString(paramList)
	output.WriteString(")\n")

	// Body
	g.indent++

	// For async functions, wrap body in coroutine.create
	if node.IsAsync {
		output.WriteString(g.generateIndent())
		output.WriteString("return coroutine.create(function()\n")
		g.indent++
	}

	// If there's a rest parameter, pack the varargs into a table
	if restParam != nil {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("local %s = {...}\n", restParam.Name.Value))
	}

	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}

	// Close coroutine wrapper for async functions
	if node.IsAsync {
		g.indent--
		output.WriteString(g.generateIndent())
		output.WriteString("end)\n")
	}

	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("end\n")

	// Apply decorators in reverse order (first decorator is outermost)
	if len(node.Decorators) > 0 {
		funcName := node.Name.Value
		for i := len(node.Decorators) - 1; i >= 0; i-- {
			decorator := node.Decorators[i]
			output.WriteString(g.generateIndent())
			if len(decorator.Arguments) > 0 {
				// Decorator with arguments: funcName = decoratorName(args)(funcName)
				output.WriteString(fmt.Sprintf("%s = %s(", funcName, decorator.Name.Value))
				args := make([]string, len(decorator.Arguments))
				for j, arg := range decorator.Arguments {
					args[j] = g.generateExpression(arg)
				}
				output.WriteString(strings.Join(args, ", "))
				output.WriteString(fmt.Sprintf(")(%s)\n", funcName))
			} else {
				// Simple decorator: funcName = decoratorName(funcName)
				output.WriteString(fmt.Sprintf("%s = %s(%s)\n", funcName, decorator.Name.Value, funcName))
			}
		}
	}

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

	for i, target := range node.Targets {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(g.generateExpression(target))
	}

	output.WriteString(" = ")

	for i, value := range node.Values {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(g.generateExpression(value))
	}

	output.WriteString("\n")

	return output.String()
}

// generateClassDeclaration generates code for a class (transpiled to Lua table with metatable)
func (g *Generator) generateClassDeclaration(node *ast.ClassDeclaration) string {
	var output strings.Builder
	className := node.Name.Value

	// Track this class for constructor calls
	g.classes[className] = true

	// Initialize static methods tracking for this class
	if g.staticMethods[className] == nil {
		g.staticMethods[className] = make(map[string]bool)
	}

	// Track which methods are static
	for _, method := range node.Methods {
		if method.IsStatic {
			g.staticMethods[className][method.Name.Value] = true
		}
	}

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
	if node.Constructor != nil || len(node.Properties) > 0 {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s.new(", className))

		// If there's a constructor, use its parameters
		if node.Constructor != nil {
			params := make([]string, len(node.Constructor.Parameters))
			for i, param := range node.Constructor.Parameters {
				params[i] = param.Name.Value
			}
			output.WriteString(strings.Join(params, ", "))
		}
		output.WriteString(")\n")

		g.indent++
		output.WriteString(g.generateIndent())
		output.WriteString("local self = setmetatable({}, " + className + ")\n")

		// Initialize instance properties with their default values
		for _, prop := range node.Properties {
			if !prop.IsStatic {
				output.WriteString(g.generateIndent())
				if prop.Value != nil {
					// Property has an initial value
					value := g.generateExpression(prop.Value)
					output.WriteString(fmt.Sprintf("self.%s = %s\n", prop.Name.Value, value))
				} else {
					// Property has no initial value, initialize to nil
					output.WriteString(fmt.Sprintf("self.%s = nil\n", prop.Name.Value))
				}
			}
		}

		// Initialize constructor parameter properties (if there's a constructor)
		if node.Constructor != nil {
			for _, param := range node.Constructor.Parameters {
				if param.Visibility != "" {
					output.WriteString(g.generateIndent())
					output.WriteString(fmt.Sprintf("self.%s = %s\n", param.Name.Value, param.Name.Value))
				}
			}

			// Execute constructor body
			for _, stmt := range node.Constructor.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
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
			output.WriteString(g.generateIndent())
			if prop.Value != nil {
				// Static property has an initial value
				value := g.generateExpression(prop.Value)
				output.WriteString(fmt.Sprintf("%s.%s = %s\n", className, prop.Name.Value, value))
			} else {
				// Static property has no initial value, initialize to nil
				output.WriteString(fmt.Sprintf("%s.%s = nil\n", className, prop.Name.Value))
			}
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

		g.pushReceiverScope()
		g.trackParameterReceivers(method.Parameters)

		output.WriteString(g.generateIndent())
		if method.IsStatic {
			// Static methods use dot notation (no self parameter)
			output.WriteString(fmt.Sprintf("function %s.%s(", className, method.Name.Value))
		} else {
			// Instance methods use colon notation (implicit self parameter)
			output.WriteString(fmt.Sprintf("function %s:%s(", className, method.Name.Value))
		}

		methodParams, methodRest := g.renderParameters(method.Parameters)
		output.WriteString(methodParams)
		output.WriteString(")\n")

		g.indent++
		if methodRest != nil {
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("local %s = {...}\n", methodRest.Name.Value))
		}
		if method.Body != nil {
			for _, stmt := range method.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
		}
		g.indent--
		g.popReceiverScope()

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

	// Apply decorators in reverse order (first decorator is outermost)
	if len(node.Decorators) > 0 {
		for i := len(node.Decorators) - 1; i >= 0; i-- {
			decorator := node.Decorators[i]
			output.WriteString(g.generateIndent())
			if len(decorator.Arguments) > 0 {
				// Decorator with arguments: ClassName = decoratorName(args)(ClassName)
				output.WriteString(fmt.Sprintf("%s = %s(", className, decorator.Name.Value))
				args := make([]string, len(decorator.Arguments))
				for j, arg := range decorator.Arguments {
					args[j] = g.generateExpression(arg)
				}
				output.WriteString(strings.Join(args, ", "))
				output.WriteString(fmt.Sprintf(")(%s)\n", className))
			} else {
				// Simple decorator: ClassName = decoratorName(ClassName)
				output.WriteString(fmt.Sprintf("%s = %s(%s)\n", className, decorator.Name.Value, className))
			}
		}
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

// generateNamespaceDeclaration generates code for a namespace (transpiled to Lua table)
func (g *Generator) generateNamespaceDeclaration(node *ast.NamespaceDeclaration) string {
	var output strings.Builder
	nsName := node.Name.Value

	// Create namespace table
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("local %s = {}\n\n", nsName))

	// Generate all statements in the namespace
	for _, stmt := range node.Statements {
		switch s := stmt.(type) {
		case *ast.ClassDeclaration:
			// Generate class and assign to namespace
			classCode := g.generateClassDeclaration(s)
			output.WriteString(classCode)
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s.%s = %s\n\n", nsName, s.Name.Value, s.Name.Value))
		case *ast.FunctionDeclaration:
			// Generate function and assign to namespace
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("function %s.%s(", nsName, s.Name.Value))
			params := make([]string, len(s.Parameters))
			for i, param := range s.Parameters {
				params[i] = param.Name.Value
			}
			output.WriteString(strings.Join(params, ", "))
			output.WriteString(")\n")

			g.indent++
			if s.Body != nil {
				for _, bodyStmt := range s.Body.Statements {
					output.WriteString(g.generateStatement(bodyStmt))
				}
			}
			g.indent--

			output.WriteString(g.generateIndent())
			output.WriteString("end\n\n")
		case *ast.EnumDeclaration:
			// Generate enum and assign to namespace
			enumCode := g.generateEnumDeclaration(s)
			output.WriteString(enumCode)
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s.%s = %s\n\n", nsName, s.Name.Value, s.Name.Value))
		case *ast.InterfaceDeclaration, *ast.TypeDeclaration:
			// Type-only declarations don't generate code
		default:
			// Generate other statements normally
			output.WriteString(g.generateStatement(stmt))
		}
	}

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
		return fmt.Sprintf("\"%s\"", g.escapeLuaString(node.Value))
	case *ast.TemplateLiteral:
		return g.generateTemplateLiteral(node)
	case *ast.BooleanLiteral:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "nil"
	case *ast.VarargExpression:
		return "..."
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
	case *ast.AwaitExpression:
		// Await translates to coroutine.resume for the coroutine
		return fmt.Sprintf("coroutine.yield(%s)", g.generateExpression(node.Expression))
	case *ast.MatchExpression:
		return g.generateMatchExpression(node)
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

	// If we have key-value pairs AND spread expressions, treat spreads as object spreads
	// Example: {...obj1, key = val, ...obj2}
	if len(node.Pairs) > 0 && hasSpread {
		var output strings.Builder
		output.WriteString("(function() local __temp = {}; ")

		// First, merge all spread expressions
		for _, val := range node.Values {
			if spread, ok := val.(*ast.SpreadExpression); ok {
				output.WriteString(spreadMerge(g.generateExpression(spread.Value)))
			} else {
				// Non-spread values in mixed context are added as array elements
				output.WriteString(fmt.Sprintf("table.insert(__temp, %s); ", g.generateExpression(val)))
			}
		}

		// Then add explicit key-value pairs (they can override spread values)
		for _, pair := range g.sortedPairs(node.Pairs) {
			key, val := pair[0], pair[1]
			if isLuaIdentifier(key) {
				output.WriteString(fmt.Sprintf("__temp[\"%s\"] = %s; ", key, val))
			} else {
				output.WriteString(fmt.Sprintf("__temp[%s] = %s; ", key, val))
			}
		}

		output.WriteString("return __temp end)()")
		return output.String()
	}

	// If there are spread expressions in array-only context, treat as array spread
	if hasSpread && len(node.Pairs) == 0 {
		var output strings.Builder
		output.WriteString("(function() local __temp = {}; ")

		for _, val := range node.Values {
			if spread, ok := val.(*ast.SpreadExpression); ok {
				// Merge the spread source, whatever shape it has
				output.WriteString(spreadMerge(g.generateExpression(spread.Value)))
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
		for _, pair := range g.sortedPairs(node.Pairs) {
			key, val := pair[0], pair[1]

			// A plain identifier key needs no brackets
			if isLuaIdentifier(key) {
				pairs = append(pairs, fmt.Sprintf("%s = %s", key, val))
			} else {
				pairs = append(pairs, fmt.Sprintf("[%s] = %s", key, val))
			}
		}
		output.WriteString(strings.Join(pairs, ", "))
	}

	output.WriteString("}")
	return output.String()
}

// isLuaIdentifier reports whether a rendered table key is a bare name that Lua
// accepts without brackets. Anything else -- a quoted string, an expression, a
// word Lua reserves -- has to be written as [key].
func isLuaIdentifier(key string) bool {
	if key == "" || lexer.IsLuaReserved(key) {
		return false
	}

	for i, r := range key {
		isLetter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'

		if !isLetter && !(isDigit && i > 0) {
			return false
		}
	}

	return true
}

// spreadMerge emits the Lua that merges one spread source into __temp. It works
// for arrays, records and mixed tables: the sequence part is appended in order
// and every other key is copied. The source is bound to a local first so an
// expression with side effects is evaluated once.
func spreadMerge(source string) string {
	return fmt.Sprintf(
		"do local __src = %s; local __len = #__src; "+
			"for __i = 1, __len do table.insert(__temp, __src[__i]) end; "+
			"for __k, __v in pairs(__src) do if type(__k) ~= 'number' or __k > __len then __temp[__k] = __v end end "+
			"end; ",
		source)
}

// sortedPairs renders a table literal's key/value pairs in a stable order.
// Ranging over the map directly makes the same source compile to different
// output from run to run.
func (g *Generator) sortedPairs(pairs map[ast.Expression]ast.Expression) [][2]string {
	rendered := make([][2]string, 0, len(pairs))

	for key, val := range pairs {
		keyStr := ""
		if ident, ok := key.(*ast.Identifier); ok {
			keyStr = ident.Value
		} else {
			keyStr = g.generateExpression(key)
		}
		rendered = append(rendered, [2]string{keyStr, g.generateExpression(val)})
	}

	sort.Slice(rendered, func(i, j int) bool { return rendered[i][0] < rendered[j][0] })

	return rendered
}

// generatePrefixExpression generates code for a prefix expression
func (g *Generator) generatePrefixExpression(node *ast.PrefixExpression) string {
	operator := node.Operator
	right := g.generateExpression(node.Right)

	// Convert 'not' to Lua 'not'
	if operator == "!" {
		operator = "not"
	}

	// Handle bitwise NOT (~) for Lua 5.1/5.2/LuaJIT compatibility
	if operator == "~" {
		if g.target == "lua53" || g.target == "lua54" {
			// Lua 5.3+ supports ~ natively
			return fmt.Sprintf("~%s", right)
		} else if g.target == "lua52" {
			// Lua 5.2 uses bit32 library
			return fmt.Sprintf("bit32.bnot(%s)", right)
		} else {
			// LuaJIT and Lua 5.1 use bit library
			return fmt.Sprintf("bit.bnot(%s)", right)
		}
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

	// Handle integer division for LuaJIT compatibility
	if operator == "//" {
		if g.IsLuaJIT() {
			// LuaJIT doesn't support //, use math.floor(a/b)
			return fmt.Sprintf("math.floor(%s / %s)", left, right)
		}
		// Lua 5.3+ supports // natively
		return fmt.Sprintf("%s // %s", left, right)
	}

	// Handle bitwise operators for Lua 5.1/5.2/LuaJIT compatibility
	bitwiseOp := ""
	switch operator {
	case "&":
		bitwiseOp = "band"
	case "|":
		bitwiseOp = "bor"
	case "^":
		bitwiseOp = "bxor"
	case "<<":
		bitwiseOp = "lshift"
	case ">>":
		bitwiseOp = "rshift"
	}

	if bitwiseOp != "" {
		if g.target == "lua53" || g.target == "lua54" {
			// Lua 5.3+ supports bitwise operators natively
			return fmt.Sprintf("%s %s %s", left, operator, right)
		} else if g.target == "lua52" {
			// Lua 5.2 uses bit32 library
			return fmt.Sprintf("bit32.%s(%s, %s)", bitwiseOp, left, right)
		} else {
			// LuaJIT and Lua 5.1 use bit library
			return fmt.Sprintf("bit.%s(%s, %s)", bitwiseOp, left, right)
		}
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

// pushReceiverScope starts a new lexical scope for implicit-self tracking.
func (g *Generator) pushReceiverScope() {
	g.selfReceivers = append(g.selfReceivers, make(map[string]bool))
}

// popReceiverScope discards the innermost implicit-self scope.
func (g *Generator) popReceiverScope() {
	if len(g.selfReceivers) > 1 {
		g.selfReceivers = g.selfReceivers[:len(g.selfReceivers)-1]
	}
}

// markSelfReceiver records that calls on this name need Lua's ':' syntax.
func (g *Generator) markSelfReceiver(name string) {
	if len(g.selfReceivers) == 0 {
		g.selfReceivers = []map[string]bool{make(map[string]bool)}
	}
	g.selfReceivers[len(g.selfReceivers)-1][name] = true
}

// isSelfReceiver reports whether a name is known to hold a class instance or a
// string, both of which dispatch their methods through ':' in Lua.
func (g *Generator) isSelfReceiver(name string) bool {
	for i := len(g.selfReceivers) - 1; i >= 0; i-- {
		if g.selfReceivers[i][name] {
			return true
		}
	}
	return false
}

// typeNeedsSelfDispatch reports whether a declared type annotation describes a
// receiver whose methods are called with ':' (a class instance or a string).
func (g *Generator) typeNeedsSelfDispatch(typeExpr ast.Expression) bool {
	ident, ok := typeExpr.(*ast.Identifier)
	if !ok {
		return false
	}
	return g.classes[ident.Value] || ident.Value == "string"
}

// valueNeedsSelfDispatch reports whether an initializer obviously produces a
// class instance or a string, e.g. `local p = Person()` or `local s = "hi"`.
func (g *Generator) valueNeedsSelfDispatch(value ast.Expression) bool {
	switch v := value.(type) {
	case *ast.StringLiteral, *ast.TemplateLiteral:
		return true
	case *ast.CallExpression:
		if ident, ok := v.Function.(*ast.Identifier); ok {
			return g.classes[ident.Value]
		}
	}
	return false
}

// trackDeclaredReceivers records any names in a declaration whose methods will
// need ':' dispatch, so later calls on them generate the right syntax.
func (g *Generator) trackDeclaredReceivers(names []*ast.Identifier, typeExprs []ast.Expression, values []ast.Expression) {
	for i, name := range names {
		if name == nil {
			continue
		}
		if i < len(typeExprs) && typeExprs[i] != nil && g.typeNeedsSelfDispatch(typeExprs[i]) {
			g.markSelfReceiver(name.Value)
			continue
		}
		if i < len(values) && values[i] != nil && g.valueNeedsSelfDispatch(values[i]) {
			g.markSelfReceiver(name.Value)
		}
	}
}

// trackParameterReceivers records parameters that are class instances or strings.
func (g *Generator) trackParameterReceivers(params []*ast.Parameter) {
	for _, param := range params {
		if param == nil || param.Name == nil || param.Type == nil {
			continue
		}
		if g.typeNeedsSelfDispatch(param.Type) {
			g.markSelfReceiver(param.Name.Value)
		}
	}
}

// usesImplicitSelf decides between Lua's ':' and '.' call syntax for a member
// call. The type checker's resolution wins when available; otherwise the
// receiver is inferred from what the generator has seen so far, defaulting to
// '.' so that library calls like table.insert(t, v) stay correct.
func (g *Generator) usesImplicitSelf(dotExpr *ast.DotExpression, property string) bool {
	// obj:method() in the source says exactly what it wants.
	if dotExpr.IsMethodCall {
		return true
	}

	switch dotExpr.MethodDispatch {
	case ast.DispatchSelf:
		return true
	case ast.DispatchPlain:
		return false
	}

	// Untyped fallback (--no-typecheck, or a receiver the checker left as 'any').
	switch left := dotExpr.Left.(type) {
	case *ast.Identifier:
		// A static method reached through its class name takes no self.
		if g.classes[left.Value] {
			return !(g.staticMethods[left.Value] != nil && g.staticMethods[left.Value][property])
		}
		if left.Value == "self" || left.Value == "super" {
			return true
		}
		return g.isSelfReceiver(left.Value)
	case *ast.StringLiteral, *ast.TemplateLiteral:
		return true
	case *ast.DotExpression:
		// self.field.method(): the field's type is unknown here, but a method
		// call on an instance field is far more common than a module lookup
		// through a class instance.
		if inner, ok := left.Left.(*ast.Identifier); ok && (inner.Value == "self" || inner.Value == "super") {
			return true
		}
	}

	return false
}

// generateCallExpression generates code for a function call
func (g *Generator) generateCallExpression(node *ast.CallExpression) string {
	// Optional call (fn?.()) evaluates the callee once and yields nil when it
	// is absent, instead of erroring on a nil call.
	if node.IsOptional {
		function := g.generateExpression(node.Function)
		args := make([]string, len(node.Arguments))
		for i, arg := range node.Arguments {
			args[i] = g.generateExpression(arg)
		}
		return fmt.Sprintf("(function() local __fn = %s; if __fn ~= nil then return __fn(%s) else return nil end end)()",
			function, strings.Join(args, ", "))
	}

	// Check if this is a method call on an object (DotExpression)
	// In Lunar: object.method() should compile to Lua: object:method() or object.method()
	// depending on whether it's an instance method or static method
	if dotExpr, ok := node.Function.(*ast.DotExpression); ok {
		// Right side should be an identifier (the property/method name)
		var property string
		if rightIdent, ok := dotExpr.Right.(*ast.Identifier); ok {
			property = rightIdent.Value
		} else {
			// If right is not an identifier, generate normally
			function := g.generateExpression(node.Function)
			args := make([]string, len(node.Arguments))
			for i, arg := range node.Arguments {
				args[i] = g.generateExpression(arg)
			}
			return fmt.Sprintf("%s(%s)", function, strings.Join(args, ", "))
		}

		useColonSyntax := g.usesImplicitSelf(dotExpr, property)

		object := g.generateExpression(dotExpr.Left)

		// Check if any argument is a spread expression
		hasSpread := false
		for _, arg := range node.Arguments {
			if _, ok := arg.(*ast.SpreadExpression); ok {
				hasSpread = true
				break
			}
		}

		// Choose the appropriate syntax for method calls
		methodOp := ":"
		if !useColonSyntax {
			methodOp = "."
		}

		// Generate method call
		if hasSpread {
			// Handle spread arguments
			if len(node.Arguments) == 1 {
				if spread, ok := node.Arguments[0].(*ast.SpreadExpression); ok {
					spreadValue := g.generateExpression(spread.Value)
					return fmt.Sprintf("%s%s%s(table.unpack(%s))", object, methodOp, property, spreadValue)
				}
			}

			// Complex case: mixed regular and spread arguments
			var tableBuilder strings.Builder
			tableBuilder.WriteString("{")
			for i, arg := range node.Arguments {
				if i > 0 {
					tableBuilder.WriteString(", ")
				}
				if spread, ok := arg.(*ast.SpreadExpression); ok {
					tableBuilder.WriteString(fmt.Sprintf("table.unpack(%s)", g.generateExpression(spread.Value)))
				} else {
					tableBuilder.WriteString(g.generateExpression(arg))
				}
			}
			tableBuilder.WriteString("}")
			return fmt.Sprintf("%s%s%s(table.unpack(%s))", object, methodOp, property, tableBuilder.String())
		}

		// No spread arguments - normal method call
		args := make([]string, len(node.Arguments))
		for i, arg := range node.Arguments {
			args[i] = g.generateExpression(arg)
		}

		return fmt.Sprintf("%s%s%s(%s)", object, methodOp, property, strings.Join(args, ", "))
	}

	// Not a method call - generate normally
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
		return fmt.Sprintf("\"%s\"", g.escapeLuaString(node.Parts[0]))
	}

	var parts []string
	for i, part := range node.Parts {
		// Add the string part if it's not empty
		if part != "" {
			parts = append(parts, fmt.Sprintf("\"%s\"", g.escapeLuaString(part)))
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

	// Method call syntax (obj:method) is preserved verbatim; the parser only
	// produces it when a call follows, which generateCallExpression handles.
	if node.IsMethodCall {
		return fmt.Sprintf("%s:%s", left, right)
	}

	return fmt.Sprintf("%s.%s", left, right)
}

// generateIndexExpression generates code for an index expression
func (g *Generator) generateIndexExpression(node *ast.IndexExpression) string {
	left := g.generateExpression(node.Left)
	index := g.generateExpression(node.Index)

	// Optional index access (?[]) evaluates the receiver once and yields nil
	// instead of erroring when it is nil.
	if node.IsOptional {
		return fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp[%s] else return nil end end)()", left, index)
	}

	return fmt.Sprintf("%s[%s]", left, index)
}

// generateFunctionExpression generates code for an anonymous function expression
func (g *Generator) generateFunctionExpression(node *ast.FunctionExpression) string {
	var output strings.Builder

	g.pushReceiverScope()
	defer g.popReceiverScope()
	g.trackParameterReceivers(node.Parameters)

	output.WriteString("function(")

	// Parameters (without type annotations)
	paramList, restParam := g.renderParameters(node.Parameters)
	output.WriteString(paramList)
	output.WriteString(")\n")

	// Body
	g.indent++
	if restParam != nil {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("local %s = {...}\n", restParam.Name.Value))
	}
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
	return GenerateWithTarget(statements, "")
}

// GenerateWithTarget generates Lua code for a specific target
func GenerateWithTarget(statements []ast.Statement, target string) string {
	generator := NewWithTarget(target)
	return generator.Generate(statements)
}

// GenerateWithOptions generates Lua code with configurable optimization
func GenerateWithOptions(statements []ast.Statement, optimize bool) string {
	return GenerateWithOptionsAndTarget(statements, optimize, "")
}

// GenerateWithOptionsAndTarget generates Lua code with configurable optimization and target
func GenerateWithOptionsAndTarget(statements []ast.Statement, optimize bool, target string) string {
	// Run optimizer if enabled
	if optimize {
		optimizer := NewOptimizer(true)
		statements = optimizer.OptimizeStatements(statements)
	}

	generator := NewWithTarget(target)
	return generator.Generate(statements)
}

// GenerateWithSourceMap generates Lua code and source map
func GenerateWithSourceMap(statements []ast.Statement, sourceFile, generatedFile string, optimize bool) (string, *sourcemap.SourceMap) {
	return GenerateWithSourceMapAndTarget(statements, sourceFile, generatedFile, optimize, "")
}

// GenerateWithSourceMapAndTarget generates Lua code and source map for a specific target
func GenerateWithSourceMapAndTarget(statements []ast.Statement, sourceFile, generatedFile string, optimize bool, target string) (string, *sourcemap.SourceMap) {
	// Run optimizer if enabled
	if optimize {
		optimizer := NewOptimizer(true)
		statements = optimizer.OptimizeStatements(statements)
	}

	generator := NewWithSourceMapAndTarget(sourceFile, generatedFile, target)
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

// escapeLuaString escapes special characters in a string for Lua output
func (g *Generator) escapeLuaString(s string) string {
	var result strings.Builder
	for _, ch := range s {
		switch ch {
		case '\n':
			result.WriteString("\\n")
		case '\t':
			result.WriteString("\\t")
		case '\r':
			result.WriteString("\\r")
		case '\b':
			result.WriteString("\\b")
		case '\f':
			result.WriteString("\\f")
		case '\v':
			result.WriteString("\\v")
		case '\x00':
			result.WriteString("\\0")
		case '"':
			result.WriteString("\\\"")
		case '\\':
			result.WriteString("\\\\")
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// ============================================
// Pattern Matching Code Generation
// ============================================

// generateMatchExpression generates code for a match expression
// Compiles to an immediately invoked function with if/elseif chains
func (g *Generator) generateMatchExpression(node *ast.MatchExpression) string {
	var output strings.Builder

	// Start an immediately invoked function that returns the matched result
	output.WriteString("(function()")
	output.WriteString("\n")

	// Store the matched value in a local variable
	output.WriteString("  local __match_value = ")
	output.WriteString(g.generateExpression(node.Value))
	output.WriteString("\n")

	// Generate if/elseif chain for each case
	for i, matchCase := range node.Cases {
		if i == 0 {
			output.WriteString("  if ")
		} else {
			output.WriteString("  elseif ")
		}

		// Generate pattern matching condition
		condition, bindings := g.generatePatternCondition(matchCase.Pattern, "__match_value")

		// If there are bindings and a guard, we need to declare bindings before the guard
		// To do this, we wrap the condition in a separate scope
		if len(bindings) > 0 && matchCase.Guard != nil {
			output.WriteString("(function()\n")
			// Add pattern bindings first
			for _, binding := range bindings {
				output.WriteString("      ")
				output.WriteString(binding)
				output.WriteString("\n")
			}
			output.WriteString("      return ")
			output.WriteString(condition)
			output.WriteString(" and (")
			output.WriteString(g.generateExpression(matchCase.Guard))
			output.WriteString(")\n    end)() then\n")
		} else {
			output.WriteString(condition)

			// Add guard condition if present (and no bindings)
			if matchCase.Guard != nil {
				output.WriteString(" and (")
				output.WriteString(g.generateExpression(matchCase.Guard))
				output.WriteString(")")
			}

			output.WriteString(" then\n")
		}

		// Bind the pattern variables for the body. When a guard is present the
		// bindings above live inside the guard's own closure, so the body still
		// needs its own copies to see them.
		for _, binding := range bindings {
			output.WriteString("    ")
			output.WriteString(binding)
			output.WriteString("\n")
		}

		// Return the body expression
		output.WriteString("    return ")
		output.WriteString(g.generateExpression(matchCase.Body))
		output.WriteString("\n")
	}

	// Close the if statement
	output.WriteString("  end\n")

	// If no pattern matched, return nil (or could error)
	output.WriteString("  error('Pattern match failed: no case matched')\n")

	// End the function and invoke it
	output.WriteString("end)()")

	return output.String()
}

// generatePatternCondition generates the condition check and bindings for a pattern
// Returns (condition string, list of binding statements)
func (g *Generator) generatePatternCondition(pattern ast.Pattern, valueName string) (string, []string) {
	bindings := []string{}

	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard matches everything
		return "true", bindings

	case *ast.LiteralPattern:
		// Match against literal value
		literalValue := g.generateExpression(p.Value)
		condition := fmt.Sprintf("%s == %s", valueName, literalValue)
		return condition, bindings

	case *ast.BindingPattern:
		// Binding pattern always matches and binds the value to a variable
		binding := fmt.Sprintf("local %s = %s", p.Name, valueName)
		bindings = append(bindings, binding)
		return "true", bindings

	case *ast.TypePattern:
		// Type pattern - check if value has a specific type tag or structure
		// For discriminated unions, check the 'type' or 'kind' field
		var condition string

		if p.TypeName == "number" {
			condition = fmt.Sprintf("type(%s) == 'number'", valueName)
		} else if p.TypeName == "string" {
			condition = fmt.Sprintf("type(%s) == 'string'", valueName)
		} else if p.TypeName == "boolean" {
			condition = fmt.Sprintf("type(%s) == 'boolean'", valueName)
		} else if p.TypeName == "table" {
			condition = fmt.Sprintf("type(%s) == 'table'", valueName)
		} else {
			// For custom types, check the type tag field
			// Try both .type and .kind as common discriminator fields
			condition = fmt.Sprintf("(type(%s) == 'table' and (%s.type == '%s' or %s.kind == '%s'))",
				valueName, valueName, p.TypeName, valueName, p.TypeName)
		}

		// Add binding if specified
		if p.Binding != "" {
			binding := fmt.Sprintf("local %s = %s", p.Binding, valueName)
			bindings = append(bindings, binding)
		}

		return condition, bindings

	case *ast.StructPattern:
		// Struct pattern - check if value is a table and has matching fields
		var conditions []string
		conditions = append(conditions, fmt.Sprintf("type(%s) == 'table'", valueName))

		// Field order in the generated condition must not depend on map
		// iteration order, or the same source compiles to different output.
		fieldNames := make([]string, 0, len(p.Fields))
		for fieldName := range p.Fields {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)

		// Generate conditions and bindings for each field
		for _, fieldName := range fieldNames {
			fieldPattern := p.Fields[fieldName]
			fieldValue := fmt.Sprintf("%s.%s", valueName, fieldName)

			// A named field has to be present for the pattern to match, so
			// { name: n, age: a } does not match a table that has no age.
			// Matching a field against nil explicitly is the one exception.
			if !patternMatchesNil(fieldPattern) {
				conditions = append(conditions, fmt.Sprintf("%s ~= nil", fieldValue))
			}

			fieldCond, fieldBindings := g.generatePatternCondition(fieldPattern, fieldValue)

			// Only add non-trivial conditions (not "true")
			if fieldCond != "true" {
				conditions = append(conditions, fieldCond)
			}

			bindings = append(bindings, fieldBindings...)
		}

		condition := strings.Join(conditions, " and ")
		return condition, bindings

	default:
		// Unknown pattern type
		return "false", bindings
	}
}

// patternMatchesNil reports whether a pattern is written to match a nil value,
// in which case a struct field carrying it must not be required to be present.
func patternMatchesNil(pattern ast.Pattern) bool {
	literal, ok := pattern.(*ast.LiteralPattern)
	if !ok {
		return false
	}

	_, isNil := literal.Value.(*ast.NilLiteral)
	return isNil
}
