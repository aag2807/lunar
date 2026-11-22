package codegen

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/sourcemap"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Generator generates Lua code from an AST
type Generator struct {
	indent           int
	sourceMapBuilder *sourcemap.Builder
	currentLine      int
	currentColumn    int
	sourceFile       string
	classParents     map[string]string // Track parent class names for super
	// varargs says, for each function being generated, whether it takes
	// '...'; empty means the main chunk.
	varargs []bool
	// namespaceLocals are the members of the namespaces being generated,
	// declared local at the start of each namespace's block.
	namespaceLocals map[string]bool
	// forwardDeclared are the top-level classes and enums declared local at
	// the start of the chunk, which their declarations then assign.
	forwardDeclared  map[string]bool
	currentClassName string   // Current class being generated
	exports          []string // Track exported names for module generation
	target           string   // Target Lua version: lua53 or lua54
	// needMetamethodList: a subclass copies its parent's metamethods,
	// so the module declares the list of their names.
	needMetamethodList bool
}

// newGenerator is the one constructor every public variant goes through, so
// the plain and source-mapped generators can never drift apart in state.
func newGenerator(target string, sm *sourcemap.Builder, sourceFile string) *Generator {
	if target == "" {
		target = "lua53"
	}
	g := &Generator{
		classParents: make(map[string]string),
		exports:      make([]string, 0),
		target:       target,
	}
	if sm != nil {
		g.sourceMapBuilder = sm
		g.currentLine = 1
		g.sourceFile = sourceFile
	}
	return g
}

// New creates a new code generator
func New() *Generator {
	return newGenerator("", nil, "")
}

// NewWithTarget creates a new code generator with specified target
func NewWithTarget(target string) *Generator {
	return newGenerator(target, nil, "")
}

// NewWithSourceMap creates a new code generator with source map support
func NewWithSourceMap(sourceFile, generatedFile string) *Generator {
	return NewWithSourceMapAndTarget(sourceFile, generatedFile, "")
}

// NewWithSourceMapAndTarget creates a new code generator with source map support and target
func NewWithSourceMapAndTarget(sourceFile, generatedFile string, target string) *Generator {
	return newGenerator(target, sourcemap.NewBuilder(sourceFile, generatedFile), sourceFile)
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

	// Exported functions are module members, not globals. Declaring them all
	// local up front keeps forward references working: `function a()` then
	// assigns the local, and a body can call one defined further down.
	// So are the file's classes and enums: a function above a class may
	// name it, and runs after it is set up.
	names := exportedFunctionNames(statements)
	g.forwardDeclared = map[string]bool{}
	for _, stmt := range statements {
		if exp, ok := stmt.(*ast.ExportStatement); ok {
			stmt = exp.Statement
		}
		switch n := stmt.(type) {
		case *ast.ClassDeclaration:
			names = append(names, n.Name.Value)
			g.forwardDeclared[n.Name.Value] = true
		case *ast.EnumDeclaration:
			names = append(names, n.Name.Value)
			g.forwardDeclared[n.Name.Value] = true
		case *ast.NamespaceDeclaration:
			names = append(names, n.Name.Value)
			g.forwardDeclared[n.Name.Value] = true
		}
	}
	if len(names) > 0 {
		decl := "local " + strings.Join(names, ", ") + "\n\n"
		g.write(decl)
		output.WriteString(decl)
	}

	for i, stmt := range statements {
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

	code := output.String()
	if g.needMetamethodList {
		code = metamethodList + code
	}
	if g.sourceMapBuilder != nil {
		return g.placeMappings(code)
	}
	return code
}

// defaultsPrologue gives each parameter with a default its value when the
// argument is left out or nil: x: number = 4.
func (g *Generator) defaultsPrologue(params []*ast.Parameter) string {
	var out strings.Builder
	for _, p := range params {
		if p.Default == nil || p.Name == nil {
			continue
		}
		out.WriteString(g.generateIndent())
		out.WriteString(fmt.Sprintf("if %s == nil then %s = %s end\n", p.Name.Value, p.Name.Value, g.generateExpression(p.Default)))
	}
	return out.String()
}

// metamethodList names Lua's metamethods, for subclasses to copy from
// their parent (Lua does not find them through __index).
const metamethodList = `local __metamethods = {"__add", "__sub", "__mul", "__div", "__mod", "__pow", "__unm", "__idiv", "__band", "__bor", "__bxor", "__shl", "__shr", "__bnot", "__concat", "__len", "__eq", "__lt", "__le", "__call", "__tostring", "__close", "__gc", "__mode", "__name", "__pairs", "__metatable"}

`

// exportedFunctionNames lists the top-level exported functions in order.
func exportedFunctionNames(statements []ast.Statement) []string {
	var names []string
	for _, stmt := range statements {
		if exp, ok := stmt.(*ast.ExportStatement); ok {
			if fn, ok := exp.Statement.(*ast.FunctionDeclaration); ok && fn.Name != nil {
				names = append(names, fn.Name.Value)
			}
		}
	}
	return names
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
	case *ast.RepeatStatement:
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
	code := g.generateStatementCode(stmt)
	// With a source map, each statement, nested ones too, carries a marker
	// of where it is in the source; Generate turns them into mappings.
	if g.sourceMapBuilder != nil && code != "" {
		if tok, ok := statementToken(stmt); ok && tok.Line > 0 {
			code = fmt.Sprintf("\x01%d:%d\x02", tok.Line, tok.Column) + code
		}
	}
	return code
}

// statementToken is the token a statement starts with.
func statementToken(stmt ast.Statement) (lexer.Token, bool) {
	v := reflect.ValueOf(stmt)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return lexer.Token{}, false
	}
	f := v.Elem().FieldByName("Token")
	if !f.IsValid() {
		return lexer.Token{}, false
	}
	tok, ok := f.Interface().(lexer.Token)
	return tok, ok
}

// placeMappings removes the statements' markers from the code and maps
// each place one was to the source position it names.
func (g *Generator) placeMappings(code string) string {
	var out strings.Builder
	line, col := 1, 0
	for i := 0; i < len(code); {
		c := code[i]
		if c == '\x01' {
			end := strings.IndexByte(code[i:], '\x02')
			var srcLine, srcCol int
			fmt.Sscanf(code[i+1:i+end], "%d:%d", &srcLine, &srcCol)
			g.sourceMapBuilder.AddMapping(line, col, srcLine, srcCol-1, "")
			i += end + 1
			continue
		}
		out.WriteByte(c)
		if c == '\n' {
			line, col = line+1, 0
		} else if c < 0x80 || c >= 0xC0 {
			col++ // a character, not a UTF-8 continuation byte
		}
		i++
	}
	return out.String()
}

func (g *Generator) generateStatementCode(stmt ast.Statement) string {
	if stmt == nil {
		return ""
	}

	switch node := stmt.(type) {
	case *ast.VariableDeclaration:
		return g.generateVariableDeclaration(node)
	case *ast.FunctionDeclaration:
		return g.generateFunctionDeclaration(node)
	case *ast.ExpressionStatement:
		code := g.generateExpression(node.Expression)
		// A statement starting with '(' would continue the one before it
		// (`f\n(x)` is a call of f): Lua's ';' keeps it apart.
		if strings.HasPrefix(code, "(") {
			code = ";" + code
		}
		return g.generateIndent() + code + "\n"
	case *ast.ReturnStatement:
		return g.generateReturnStatement(node)
	case *ast.IfStatement:
		return g.generateIfStatement(node)
	case *ast.WhileStatement:
		return g.generateWhileStatement(node)
	case *ast.RepeatStatement:
		return g.generateRepeatStatement(node)
	case *ast.ForStatement:
		return g.generateForStatement(node)
	case *ast.DoStatement:
		return g.generateDoStatement(node)
	case *ast.BreakStatement:
		return g.generateIndent() + "break\n"
	case *ast.GotoStatement:
		return g.generateIndent() + "goto " + node.Label.Value + "\n"
	case *ast.LabelStatement:
		return g.generateIndent() + "::" + node.Name.Value + "::\n"
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
	output.WriteString(g.generateIndent())
	output.WriteString("local ")

	// Write all variable names, with Lua 5.4's attributes. For 5.3 a
	// <const> is dropped (the checker enforces it); <close> is refused.
	for i, name := range node.Names {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(name.Value)
		if i < len(node.Attribs) && node.Attribs[i] != "" && (g.target == "lua54" || node.Attribs[i] != "const") {
			output.WriteString(" <" + node.Attribs[i] + ">")
		}
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

	output.WriteString(g.generateIndent())
	if node.IsLocal {
		output.WriteString("local ")
	}
	output.WriteString("function ")
	if node.Receiver != nil {
		// function M.add() and function Point:len(), as written.
		output.WriteString(g.generateExpression(node.Receiver))
		if node.IsMethod {
			output.WriteString(":")
		} else {
			output.WriteString(".")
		}
	}
	output.WriteString(node.Name.Value)
	output.WriteString("(")

	// Parameters (without type annotations)
	paramList, restParam := g.renderParameters(node.Parameters)
	output.WriteString(paramList)
	output.WriteString(")\n")
	defer g.enterFunction(strings.Contains(paramList, "..."))()

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
	output.WriteString(g.defaultsPrologue(node.Parameters))

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
	// An elseif chain is written as one: nested else-if blocks overflowed
	// Lua's C stack at about 200 branches, and took quadratic time.
	for cur := node; ; {
		output.WriteString(g.generateExpression(cur.Condition))
		output.WriteString(" then\n")
		g.indent++
		for _, stmt := range cur.Consequence.Statements {
			output.WriteString(g.generateStatement(stmt))
		}
		g.indent--

		alt := cur.Alternative
		if alt == nil {
			break
		}
		if len(alt.Statements) == 1 {
			if next, ok := alt.Statements[0].(*ast.IfStatement); ok && next.Token.Literal == "elseif" {
				output.WriteString(g.generateIndent())
				output.WriteString("elseif ")
				cur = next
				continue
			}
		}
		output.WriteString(g.generateIndent())
		output.WriteString("else\n")
		g.indent++
		for _, stmt := range alt.Statements {
			output.WriteString(g.generateStatement(stmt))
		}
		g.indent--
		break
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

// generateRepeatStatement generates code for a repeat ... until loop
func (g *Generator) generateRepeatStatement(node *ast.RepeatStatement) string {
	var output strings.Builder

	output.WriteString(g.generateIndent())
	output.WriteString("repeat\n")

	g.indent++
	for _, stmt := range node.Body.Statements {
		output.WriteString(g.generateStatement(stmt))
	}
	g.indent--

	output.WriteString(g.generateIndent())
	output.WriteString("until ")
	output.WriteString(g.generateExpression(node.Condition))
	output.WriteString("\n")

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
		for _, e := range node.IteratorRest {
			output.WriteString(", ")
			output.WriteString(g.generateExpression(e))
		}
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

	// super.x = v runs the parent's setter for x with this object as self,
	// or sets the field when the parent has none.
	if len(node.Targets) == 1 && len(node.Values) == 1 {
		if dot, ok := node.Targets[0].(*ast.DotExpression); ok {
			if _, isSuper := dot.Left.(*ast.SuperExpression); isSuper {
				if name, ok := dot.Right.(*ast.Identifier); ok {
					output.WriteString(fmt.Sprintf("do local __v = %s; local __s = %s.__setters; __s = __s and __s.%s; if __s then __s(self, __v) else rawset(self, %q, __v) end end\n",
						g.generateExpression(node.Values[0]), g.generateExpression(dot.Left), name.Value, name.Value))
					return output.String()
				}
			}
		}
	}

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

// localUnlessForward is "local " for a class or enum that was not declared
// at the start of the chunk.
func (g *Generator) localUnlessForward(name string) string {
	if g.indent == 0 && g.forwardDeclared[name] {
		return ""
	}
	if g.namespaceLocals[name] && g.indent > 0 {
		return ""
	}
	return "local "
}

// optionalLink finds the nearest ?. ?[ or ?.() link down a member chain.
func optionalLink(e ast.Expression) ast.Expression {
	switch n := e.(type) {
	case *ast.DotExpression:
		if n.IsOptional {
			return n
		}
		return optionalLink(n.Left)
	case *ast.IndexExpression:
		if n.IsOptional {
			return n
		}
		return optionalLink(n.Left)
	case *ast.CallExpression:
		if n.IsOptional || n.Parenthesized {
			if n.IsOptional {
				return n
			}
			return nil
		}
		return optionalLink(n.Function)
	}
	return nil
}

// shortCircuit generates outer, a chain containing the optional link, so a
// nil before the link makes the whole chain nil: the value before the link
// is kept in __o, and the rest of the chain runs on it only when it is set.
func (g *Generator) shortCircuit(outer ast.Expression, link ast.Expression) string {
	holder := &ast.Identifier{Value: "__o"}
	var base string
	switch n := link.(type) {
	case *ast.DotExpression:
		base = g.generateExpression(n.Left)
		left := n.Left
		n.Left, n.IsOptional = holder, false
		defer func() { n.Left, n.IsOptional = left, true }()
	case *ast.IndexExpression:
		base = g.generateExpression(n.Left)
		left := n.Left
		n.Left, n.IsOptional = holder, false
		defer func() { n.Left, n.IsOptional = left, true }()
	case *ast.CallExpression:
		base = g.generateExpression(n.Function)
		fn := n.Function
		n.Function, n.IsOptional = holder, false
		defer func() { n.Function, n.IsOptional = fn, true }()
	}
	return g.iife(fmt.Sprintf("(function() local __o = %s; if __o == nil then return nil end; return %s end)()", base, g.generateExpression(outer)))
}

// spreadArguments renders call arguments with a spread that is not the
// only one: every element of each spread is passed, in order, wherever it
// stands. ({table.unpack(a), 9} would keep one element of a: a call in
// the middle of a constructor gives one value.)
func (g *Generator) spreadArguments(args []ast.Expression) string {
	var b strings.Builder
	b.WriteString("(function() local __t, __n = {}, 0; ")
	for _, arg := range args {
		if spread, ok := arg.(*ast.SpreadExpression); ok {
			b.WriteString(fmt.Sprintf("local __s = %s; for __i = 1, #__s do __n = __n + 1; __t[__n] = __s[__i] end; ", g.generateExpression(spread.Value)))
		} else {
			b.WriteString(fmt.Sprintf("__n = __n + 1; __t[__n] = %s; ", g.generateExpression(arg)))
		}
	}
	b.WriteString("return table.unpack(__t, 1, __n) end)()")
	return g.iife(b.String())
}

// enterFunction records that a function whose parameters take '...' (or
// not) is being generated, and returns what leaves it.
func (g *Generator) enterFunction(vararg bool) func() {
	g.varargs = append(g.varargs, vararg)
	return func() { g.varargs = g.varargs[:len(g.varargs)-1] }
}

// inVararg says whether '...' can be used where code is being generated:
// in a function that takes it, or in the main chunk, which always does.
func (g *Generator) inVararg() bool {
	return len(g.varargs) == 0 || g.varargs[len(g.varargs)-1]
}

// iifeOpen and iifeClose bracket code the generator wraps in a function
// it calls at once. Inside a vararg function they pass '...' through, so
// the wrapped code can use the enclosing function's varargs.
func (g *Generator) iifeOpen() string {
	if g.inVararg() {
		return "(function(...)"
	}
	return "(function()"
}

func (g *Generator) iifeClose() string {
	if g.inVararg() {
		return "end)(...)"
	}
	return "end)()"
}

// iife turns a generated "(function() ... end)()" into the form iifeOpen
// and iifeClose give: only its own outer brackets, not those of anything
// already generated inside it.
func (g *Generator) iife(code string) string {
	if !g.inVararg() || !strings.HasPrefix(code, "(function()") || !strings.HasSuffix(code, "end)()") {
		return code
	}
	return "(function(...)" + strings.TrimPrefix(code, "(function()")[:len(code)-len("(function()")-len("end)()")] + "end)(...)"
}

// isSuperCall reports whether a statement is a call of super(...).
func isSuperCall(stmt ast.Statement) bool {
	es, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		return false
	}
	call, ok := es.Expression.(*ast.CallExpression)
	if !ok {
		return false
	}
	_, isSuper := call.Function.(*ast.SuperExpression)
	return isSuper
}

// accessorTables creates a class's __getters and __setters tables, chained to
// its parent's when it has one.
func (g *Generator) accessorTables(className, parentName string) string {
	var out strings.Builder
	for _, kind := range []string{"__getters", "__setters"} {
		out.WriteString(g.generateIndent())
		if parentName != "" {
			out.WriteString(fmt.Sprintf("%s.%s = setmetatable({}, {__index = %s.%s})\n", className, kind, parentName, kind))
		} else {
			out.WriteString(fmt.Sprintf("%s.%s = {}\n", className, kind))
		}
	}
	return out.String()
}

// accessorMetamethods installs the __index and __newindex that consult a
// class's accessors before its methods and the instance's own fields.
// Methods are looked up on the class table itself, whose metatable leads to
// the parent, so this never calls itself.
func (g *Generator) accessorMetamethods(className string) string {
	ind := g.generateIndent()
	lines := []string{
		"{C}.__index = function(self, key)",
		"    local get = {C}.__getters[key]",
		"    if get ~= nil then",
		"        return get(self)",
		"    end",
		"    return {C}[key]",
		"end",
		"{C}.__newindex = function(self, key, value)",
		"    local set = {C}.__setters[key]",
		"    if set ~= nil then",
		"        set(self, value)",
		"        return",
		"    end",
		"    rawset(self, key, value)",
		"end",
	}
	var out strings.Builder
	for _, line := range lines {
		out.WriteString(ind)
		out.WriteString(strings.ReplaceAll(line, "{C}", className))
		out.WriteString("\n")
	}
	return out.String()
}

// generateClassDeclaration generates code for a class (transpiled to Lua table with metatable)
func (g *Generator) generateClassDeclaration(node *ast.ClassDeclaration) string {
	var output strings.Builder
	className := node.Name.Value

	// The parent as written: Base, or Geo.Base through a namespace.
	parentName := ""
	if node.Extends != nil {
		parentName = g.generateExpression(node.Extends)
		g.classParents[className] = parentName
	}

	// Set current class context
	prevClassName := g.currentClassName
	g.currentClassName = className
	defer func() { g.currentClassName = prevClassName }()

	// Create class table
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("%s%s = {}\n", g.localUnlessForward(className), className))

	// Set up inheritance if there's a parent class. Lua finds metamethods
	// on the metatable itself, never through __index, so the parent's
	// (__tostring, __eq, __add, ...) are copied; the class's own, defined
	// below, replace them.
	if parentName != "" {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("setmetatable(%s, {__index = %s})\n", className, parentName))
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("for _, k in ipairs(__metamethods) do %s[k] = %s[k] end\n", className, parentName))
		g.needMetamethodList = true
	}

	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("%s.__index = %s\n", className, className))

	// A class with accessors keeps them in __getters/__setters tables,
	// chained to the parent's so inherited accessors are found too.
	hasAccessors := len(node.Getters) > 0 || len(node.Setters) > 0
	if hasAccessors {
		output.WriteString(g.accessorTables(className, parentName))
	}
	output.WriteString("\n")

	// Construction happens in place. __init(self, ...) runs the field
	// initialisers and the constructor body on the object it is given;
	// super(...) runs the parent's __init on that same object, so the
	// instance is one table throughout: a subclass's field values win, the
	// parent sees the real self, and methods it calls dispatch to the
	// subclass. new(...) makes the object and runs __init. Every class
	// needs one: `Calculator()` compiles to `Calculator.new()`.
	{
		inheritsConstructor := node.Constructor == nil && parentName != ""

		// Field initialisers and constructor parameter properties, run
		// after the parent's constructor (TypeScript's order).
		var inits strings.Builder
		g.indent++
		for _, prop := range node.Properties {
			if !prop.IsStatic && prop.Value != nil {
				inits.WriteString(g.generateIndent())
				inits.WriteString(fmt.Sprintf("self.%s = %s\n", prop.Name.Value, g.generateExpression(prop.Value)))
			}
		}
		if node.Constructor != nil {
			for _, param := range node.Constructor.Parameters {
				if param.Visibility != "" || param.IsReadonly {
					inits.WriteString(g.generateIndent())
					inits.WriteString(fmt.Sprintf("self.%s = %s\n", param.Name.Value, param.Name.Value))
				}
			}
		}
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s.__init(self", className))
		switch {
		case node.Constructor != nil:
			for _, param := range node.Constructor.Parameters {
				output.WriteString(", " + param.Name.Value)
			}
		case inheritsConstructor:
			output.WriteString(", ...")
		}
		output.WriteString(")\n")
		g.indent++
		if node.Constructor != nil {
			output.WriteString(g.defaultsPrologue(node.Constructor.Parameters))
		}
		if inheritsConstructor {
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s.__init(self, ...)\n", parentName))
		}
		if node.Constructor == nil {
			output.WriteString(inits.String())
		} else {
			// The initialisers go right after the super(...) call, or first
			// when there is none.
			superAt := -1
			for i, stmt := range node.Constructor.Body.Statements {
				if isSuperCall(stmt) {
					superAt = i
					break
				}
			}
			if superAt < 0 {
				output.WriteString(inits.String())
			}
			for i, stmt := range node.Constructor.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
				if i == superAt {
					output.WriteString(inits.String())
				}
			}
		}
		g.indent--
		output.WriteString(g.generateIndent())
		output.WriteString("end\n")

		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s.new(...)\n", className))
		g.indent++
		output.WriteString(g.generateIndent())
		output.WriteString("local self = setmetatable({}, " + className + ")\n")
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("%s.__init(self, ...)\n", className))
		output.WriteString(g.generateIndent())
		output.WriteString("return self\n")
		g.indent--
		output.WriteString(g.generateIndent())
		output.WriteString("end\n\n")
	}

	// Generate static properties
	for _, prop := range node.Properties {
		if prop.IsStatic && prop.Value != nil {
			// Static properties go directly on the class table; one with
			// no initial value is nil until set.
			output.WriteString(g.generateIndent())
			value := g.generateExpression(prop.Value)
			output.WriteString(fmt.Sprintf("%s.%s = %s\n", className, prop.Name.Value, value))
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

		methodParams, methodRest := g.renderParameters(method.Parameters)
		output.WriteString(methodParams)
		output.WriteString(")\n")
		leave := g.enterFunction(strings.Contains(methodParams, "..."))

		g.indent++
		if methodRest != nil {
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("local %s = {...}\n", methodRest.Name.Value))
		}
		output.WriteString(g.defaultsPrologue(method.Parameters))
		if method.Body != nil {
			for _, stmt := range method.Body.Statements {
				output.WriteString(g.generateStatement(stmt))
			}
		}
		leave()
		g.indent--

		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
		// Decorators wrap the method, the first one outermost.
		for i := len(method.Decorators) - 1; i >= 0; i-- {
			d := method.Decorators[i]
			ref := className + "." + method.Name.Value
			call := d.Name.Value
			if len(d.Arguments) > 0 {
				args := make([]string, len(d.Arguments))
				for j, arg := range d.Arguments {
					args[j] = g.generateExpression(arg)
				}
				call += "(" + strings.Join(args, ", ") + ")"
			}
			output.WriteString(g.generateIndent())
			output.WriteString(fmt.Sprintf("%s = %s(%s)\n", ref, call, ref))
		}
		output.WriteString("\n")
	}

	// Generate getters
	for _, getter := range node.Getters {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("function %s.__getters.%s(self)\n", className, getter.Name.Value))

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
		output.WriteString(fmt.Sprintf("function %s.__setters.%s(self, %s)\n", className, setter.Name.Value, paramName))

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

	// Route instance reads and writes through the accessors. A subclass with
	// none of its own still needs this when its parent has some, which is
	// only known at run time when the parent comes from another module.
	if hasAccessors {
		output.WriteString(g.accessorMetamethods(className))
		output.WriteString("\n")
	} else if parentName != "" {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("if %s.__getters ~= nil then\n", parentName))
		g.indent++
		output.WriteString(g.accessorTables(className, parentName))
		output.WriteString(g.accessorMetamethods(className))
		g.indent--
		output.WriteString(g.generateIndent())
		output.WriteString("end\n")
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
	output.WriteString(fmt.Sprintf("%s%s = {\n", g.localUnlessForward(enumName), enumName))

	implicit := map[*ast.EnumMember]float64{}
	values, _, _ := node.ImplicitValues()
	for _, v := range values {
		implicit[v.Member] = v.Value
	}

	g.indent++
	for _, member := range node.Members {
		output.WriteString(g.generateIndent())
		output.WriteString(member.Name.Value)
		output.WriteString(" = ")

		if member.Value != nil {
			output.WriteString(g.generateExpression(member.Value))
		} else {
			output.WriteString(strconv.FormatFloat(implicit[member], 'g', -1, 64))
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

	// The namespace is a table; its body runs in a block of its own, where
	// every member is a local (so members call each other by name, in any
	// order, private ones included) and the exported ones are also put in
	// the table.
	output.WriteString(g.generateIndent())
	output.WriteString(fmt.Sprintf("%s%s = {}\n", g.localUnlessForward(nsName), nsName))
	output.WriteString(g.generateIndent())
	output.WriteString("do\n")
	g.indent++

	var names []string
	for _, stmt := range node.Statements {
		if exported, ok := stmt.(*ast.ExportStatement); ok && exported.Statement != nil {
			stmt = exported.Statement
		}
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			if s.Receiver == nil && s.Name != nil {
				names = append(names, s.Name.Value)
			}
		case *ast.ClassDeclaration:
			names = append(names, s.Name.Value)
		case *ast.EnumDeclaration:
			names = append(names, s.Name.Value)
		case *ast.NamespaceDeclaration:
			names = append(names, s.Name.Value)
		}
	}
	if len(names) > 0 {
		output.WriteString(g.generateIndent())
		output.WriteString("local " + strings.Join(names, ", ") + "\n")
	}
	prevLocals := g.namespaceLocals
	g.namespaceLocals = map[string]bool{}
	for _, name := range names {
		g.namespaceLocals[name] = true
	}
	defer func() { g.namespaceLocals = prevLocals }()

	for _, stmt := range node.Statements {
		exported := false
		if exp, ok := stmt.(*ast.ExportStatement); ok && exp.Statement != nil {
			stmt, exported = exp.Statement, true
		}
		publish := func(name string) {
			if exported {
				output.WriteString(g.generateIndent())
				output.WriteString(fmt.Sprintf("%s.%s = %s\n", nsName, name, name))
			}
		}
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			if s.Receiver != nil {
				output.WriteString(g.generateStatement(s))
				continue
			}
			// The local declared above is assigned: function f() ... end.
			local := s.IsLocal
			s.IsLocal = false
			output.WriteString(g.generateFunctionDeclaration(s))
			s.IsLocal = local
			publish(s.Name.Value)
		case *ast.ClassDeclaration:
			output.WriteString(g.generateClassDeclaration(s))
			publish(s.Name.Value)
		case *ast.EnumDeclaration:
			output.WriteString(g.generateEnumDeclaration(s))
			publish(s.Name.Value)
		case *ast.NamespaceDeclaration:
			output.WriteString(g.generateNamespaceDeclaration(s))
			publish(s.Name.Value)
		case *ast.VariableDeclaration:
			output.WriteString(g.generateStatement(s))
			for _, name := range s.Names {
				publish(name.Value)
			}
		case *ast.InterfaceDeclaration, *ast.TypeDeclaration:
			// Type-only declarations don't generate code
		default:
			output.WriteString(g.generateStatement(stmt))
		}
	}

	g.indent--
	output.WriteString(g.generateIndent())
	output.WriteString("end\n")
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
	case *ast.ParenExpression:
		return "(" + g.generateExpression(node.Expression) + ")"
	case *ast.BooleanLiteral:
		if node.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "nil"
	case *ast.VarargExpression:
		return "..."
	case *ast.GenericType:
		// Type arguments are erased: Stack<number>() is just Stack() in Lua.
		return g.generateExpression(node.BaseType)
	case *ast.TableLiteral:
		return g.generateTableLiteral(node)
	case *ast.PrefixExpression:
		return g.generatePrefixExpression(node)
	case *ast.InfixExpression:
		return g.generateInfixExpression(node)
	case *ast.CallExpression:
		if node.Parenthesized {
			return "(" + g.generateCallExpression(node) + ")"
		}
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

	// The fields in the order written, when the parser recorded it.
	entries := node.Entries
	if len(entries) != len(node.Pairs)+len(node.Values) {
		entries = nil
	}

	// If we have key-value pairs AND spread expressions, treat spreads as object spreads
	// Example: {...obj1, key = val, ...obj2}
	if len(node.Pairs) > 0 && hasSpread {
		var output strings.Builder
		output.WriteString(g.iifeOpen() + " local __temp = {}; ")

		// In source order, a later field or spread overrides an earlier one.
		if entries != nil {
			for _, e := range entries {
				switch {
				case e.Key != nil:
					key, val := g.tableKey(e.Key), g.generateExpression(e.Value)
					if computed, ok := strings.CutPrefix(key, "\x00"); ok {
						output.WriteString(fmt.Sprintf("__temp[%s] = %s; ", computed, val))
					} else if isLuaIdentifier(key) {
						output.WriteString(fmt.Sprintf("__temp[\"%s\"] = %s; ", key, val))
					} else {
						output.WriteString(fmt.Sprintf("__temp[%s] = %s; ", key, val))
					}
				default:
					if spread, ok := e.Value.(*ast.SpreadExpression); ok {
						output.WriteString(spreadMerge(g.generateExpression(spread.Value)))
					} else {
						output.WriteString(fmt.Sprintf("table.insert(__temp, %s); ", g.generateExpression(e.Value)))
					}
				}
			}
			output.WriteString("return __temp " + g.iifeClose())
			return output.String()
		}

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

		output.WriteString("return __temp " + g.iifeClose())
		return output.String()
	}

	// If there are spread expressions in array-only context, treat as array spread
	if hasSpread && len(node.Pairs) == 0 {
		var output strings.Builder
		output.WriteString(g.iifeOpen() + " local __temp = {}; ")

		for _, val := range node.Values {
			if spread, ok := val.(*ast.SpreadExpression); ok {
				// Merge the spread source, whatever shape it has
				output.WriteString(spreadMerge(g.generateExpression(spread.Value)))
			} else {
				// Insert single element
				output.WriteString(fmt.Sprintf("table.insert(__temp, %s); ", g.generateExpression(val)))
			}
		}

		output.WriteString("return __temp " + g.iifeClose())
		return output.String()
	}

	// No spread - use normal table literal generation
	var output strings.Builder
	output.WriteString("{")

	if entries != nil {
		fields := make([]string, len(entries))
		for i, e := range entries {
			if e.Key == nil {
				fields[i] = g.generateExpression(e.Value)
				continue
			}
			fields[i] = g.tableField(g.tableKey(e.Key), g.generateExpression(e.Value))
		}
		output.WriteString(strings.Join(fields, ", "))
		output.WriteString("}")
		return output.String()
	}

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

// receiverNeedsParens reports whether a method call's receiver has to be
// wrapped in parentheses. Lua only allows a name, an index, a call or a
// parenthesized expression before ':' or '.', so `"a,b":gmatch(...)` is a
// syntax error while `("a,b"):gmatch(...)` is fine.
func receiverNeedsParens(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.StringLiteral, *ast.NumberLiteral, *ast.BooleanLiteral, *ast.NilLiteral,
		*ast.TemplateLiteral, *ast.TableLiteral, *ast.InfixExpression, *ast.PrefixExpression,
		*ast.FunctionExpression, *ast.VarargExpression, *ast.TypeAssertion:
		return true
	}

	return false
}

// asReceiver renders an expression for use before ':' or '.'.
func (g *Generator) asReceiver(expr ast.Expression) string {
	rendered := g.generateExpression(expr)
	if receiverNeedsParens(expr) {
		return "(" + rendered + ")"
	}

	return rendered
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
// tableKey renders a table literal's key: a name as itself, anything else
// as the expression inside [ ].
func (g *Generator) tableKey(key ast.Expression) string {
	if ident, ok := key.(*ast.Identifier); ok {
		return ident.Value
	}
	// A computed key, [KEY]: always in brackets, never a field name.
	if paren, ok := key.(*ast.ParenExpression); ok {
		return "\x00" + g.generateExpression(paren.Expression)
	}
	return g.generateExpression(key)
}

// tableField renders one key = value field of a table constructor.
func (g *Generator) tableField(key, val string) string {
	if computed, ok := strings.CutPrefix(key, "\x00"); ok {
		return fmt.Sprintf("[%s] = %s", computed, val)
	}
	if isLuaIdentifier(key) {
		return fmt.Sprintf("%s = %s", key, val)
	}
	return fmt.Sprintf("[%s] = %s", key, val)
}

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

	// Bitwise not is written next to its operand: ~x, ~(a | b).
	if operator == "~" {
		if needsUnaryParens(node.Right) {
			return fmt.Sprintf("~(%s)", right)
		}
		return fmt.Sprintf("~%s", right)
	}

	// The operand needs parentheses unless it binds tighter than a unary
	// operator (only '^' does): -(2 + 3), but -2 ^ 2.
	if needsUnaryParens(node.Right) {
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
		return g.iife(fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp else return %s end end)()", left, right))
	}

	// '//' and the bitwise operators are Lua 5.3's own, so they go through
	// the same parenthesisation as every other operator.

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

// usesImplicitSelf decides between Lua's ':' and '.' for a member call. The
// type checker records which one each member needs; without that, the call
// is emitted as written: `obj:m()` asks for self, and `obj.m()` does not.
func (g *Generator) usesImplicitSelf(dotExpr *ast.DotExpression, property string) bool {
	if dotExpr.IsMethodCall {
		return true
	}
	return dotExpr.MethodDispatch == ast.DispatchSelf
}

// generateCallExpression generates code for a function call
func (g *Generator) generateCallExpression(node *ast.CallExpression) string {
	// super(...) inside a constructor runs the parent's constructor and copies
	// the fields it set onto the instance being built. Without this it compiled
	// to a bare `Parent(args)` call, which is not even a function in the
	// generated Lua, so any subclass crashed on construction.
	// e?.hit(1): the ?. stops the call too, and the method gets its self.
	if !node.IsOptional {
		if link := optionalLink(node.Function); link != nil {
			return g.shortCircuit(node, link)
		}
	}

	if _, isSuper := node.Function.(*ast.SuperExpression); isSuper {
		parent := g.generateExpression(node.Function)
		args := []string{"self"}
		for _, arg := range node.Arguments {
			args = append(args, g.generateExpression(arg))
		}
		return fmt.Sprintf("%s.__init(%s)", parent, strings.Join(args, ", "))
	}

	// super.m(...) is the parent's m with this object as self.
	if dot, ok := node.Function.(*ast.DotExpression); ok {
		if _, isSuper := dot.Left.(*ast.SuperExpression); isSuper {
			if name, ok := dot.Right.(*ast.Identifier); ok {
				args := []string{"self"}
				for _, arg := range node.Arguments {
					args = append(args, g.generateExpression(arg))
				}
				return fmt.Sprintf("%s.%s(%s)", g.generateExpression(dot.Left), name.Value, strings.Join(args, ", "))
			}
		}
	}

	// Optional call (fn?.()) evaluates the callee once and yields nil when it
	// is absent, instead of erroring on a nil call.
	if node.IsOptional {
		function := g.generateExpression(node.Function)
		args := make([]string, len(node.Arguments))
		for i, arg := range node.Arguments {
			args[i] = g.generateExpression(arg)
		}
		return g.iife(fmt.Sprintf("(function() local __fn = %s; if __fn ~= nil then return __fn(%s) else return nil end end)()",
			function, strings.Join(args, ", ")))
	}

	// Check if this is a method call on an object (DotExpression)
	// In Lunar: object.method() should compile to Lua: object:method() or object.method()
	// depending on whether it's an instance method or static method
	// m.Enemy(3) constructs; it is not a call of a member named Enemy.
	if dotExpr, ok := node.Function.(*ast.DotExpression); ok && !node.ConstructorCall {
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

		object := g.asReceiver(dotExpr.Left)

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
			return fmt.Sprintf("%s%s%s(%s)", object, methodOp, property, g.spreadArguments(node.Arguments))
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

	// The checker marks calls that construct a Lunar class, Enemy(3) or
	// Stack<number>(), which go through the class's constructor.
	if node.ConstructorCall {
		function = function + ".new"
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
		return fmt.Sprintf("%s(%s)", function, g.spreadArguments(node.Arguments))
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
	// a?.b.c: the ?. stops the whole chain at a nil a.
	if !node.IsOptional {
		if link := optionalLink(node.Left); link != nil {
			return g.shortCircuit(node, link)
		}
	}
	// super.x is the parent's getter for x with this object as self, or the
	// field itself.
	if _, isSuper := node.Left.(*ast.SuperExpression); isSuper && !node.IsMethodCall {
		if name, ok := node.Right.(*ast.Identifier); ok {
			return fmt.Sprintf("(function(o) local g = %s.__getters; g = g and g.%s; if g then return g(o) end return rawget(o, %q) end)(self)",
				g.generateExpression(node.Left), name.Value, name.Value)
		}
	}
	left := g.asReceiver(node.Left)
	right := g.generateExpression(node.Right)

	// Handle optional chaining (?.)
	if node.IsOptional {
		// Generate safe navigation: (function() if left ~= nil then return left.right else return nil end end)()
		// For efficiency, use a simpler pattern when left is a simple identifier
		return g.iife(fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp.%s else return nil end end)()", left, right))
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
		return g.iife(fmt.Sprintf("(function() local __temp = %s; if __temp ~= nil then return __temp[%s] else return nil end end)()", left, index))
	}

	return fmt.Sprintf("%s[%s]", left, index)
}

// generateFunctionExpression generates code for an anonymous function expression
func (g *Generator) generateFunctionExpression(node *ast.FunctionExpression) string {
	var output strings.Builder

	output.WriteString("function(")

	// Parameters (without type annotations)
	paramList, restParam := g.renderParameters(node.Parameters)
	output.WriteString(paramList)
	output.WriteString(")\n")
	defer g.enterFunction(strings.Contains(paramList, "..."))()

	// Body
	g.indent++
	if restParam != nil {
		output.WriteString(g.generateIndent())
		output.WriteString(fmt.Sprintf("local %s = {...}\n", restParam.Name.Value))
	}
	output.WriteString(g.defaultsPrologue(node.Parameters))
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
// luaModuleName turns an import path into the dotted name Lua's require expects:
// "./mod/math_utils" becomes "mod.math_utils". Passing the path through
// verbatim produced a require that could never resolve.
func luaModuleName(path string) string {
	name := strings.TrimSuffix(path, ".lunar")
	name = strings.TrimPrefix(name, "./")

	// A parent-directory hop has no dotted equivalent; leave those alone rather
	// than inventing a name that resolves to the wrong module.
	if strings.HasPrefix(name, "../") {
		return name
	}

	return strings.ReplaceAll(name, "/", ".")
}

func (g *Generator) generateImportStatement(node *ast.ImportStatement) string {
	var output strings.Builder
	output.WriteString(g.generateIndent())

	// The project records the root-relative name require() takes; without
	// one (a snippet compiled on its own) the path is converted locally.
	moduleName := node.RequireName
	if moduleName == "" {
		moduleName = luaModuleName(node.Module)
	}

	if node.IsWildcard && node.Alias != nil {
		// import * as m from "module" -> local m = require("module")
		output.WriteString(fmt.Sprintf("local %s = require(\"%s\")\n", node.Alias.Value, moduleName))
	} else {
		// import { name1, name2 } from "module"
		// -> local __module = require("module")
		// -> local name1 = __module.name1
		// Names that are only types need nothing at run time; an import of
		// nothing but types is no require at all (as in TypeScript).
		var values []*ast.Identifier
		for _, name := range node.Names {
			if !node.TypeNames[name.Value] {
				values = append(values, name)
			}
		}
		if len(values) == 0 && len(node.Names) > 0 {
			return ""
		}
		tempVar := "__" + strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(moduleName)

		output.WriteString(fmt.Sprintf("local %s = require(\"%s\")\n", tempVar, moduleName))

		for _, name := range values {
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

// GenerateWithSourceMap generates Lua code and source map
func GenerateWithSourceMap(statements []ast.Statement, sourceFile, generatedFile string) (string, *sourcemap.SourceMap) {
	return GenerateWithSourceMapAndTarget(statements, sourceFile, generatedFile, "")
}

// GenerateWithSourceMapAndTarget generates Lua code and source map for a specific target
func GenerateWithSourceMapAndTarget(statements []ast.Statement, sourceFile, generatedFile string, target string) (string, *sourcemap.SourceMap) {
	generator := NewWithSourceMapAndTarget(sourceFile, generatedFile, target)
	code := generator.Generate(statements)
	return code, generator.GetSourceMap()
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

// needsUnaryParens says whether a unary operator's operand needs
// parentheses: any binary operation but '^'.
func needsUnaryParens(expr ast.Expression) bool {
	infix, ok := expr.(*ast.InfixExpression)
	return ok && getOperatorPrecedence(infix.Operator) <= unaryPrecedence
}

// needsParensInInfix says whether an operand of an infix operator needs
// parentheses in Lua: an operator that binds less tightly always does; one
// of the same level does on the right of a left-associative operator and on
// the left of a right-associative one ('..', '^'). A unary operand on the
// left of '^' does too, since -2 ^ 2 is -(2 ^ 2).
func needsParensInInfix(expr ast.Expression, parentOp string, isLeft bool) bool {
	parentPrec := getOperatorPrecedence(parentOp)
	if _, unary := expr.(*ast.PrefixExpression); unary {
		return isLeft && parentPrec > unaryPrecedence
	}
	infixExpr, ok := expr.(*ast.InfixExpression)
	if !ok {
		return false
	}
	childPrec := getOperatorPrecedence(infixExpr.Operator)
	if childPrec != parentPrec {
		return childPrec < parentPrec
	}
	if parentOp == ".." || parentOp == "^" {
		return isLeft
	}
	return !isLeft
}

// unaryPrecedence is the level of not, #, - and ~ as prefixes.
const unaryPrecedence = 11

// getOperatorPrecedence returns the level of a binary operator in Lua
// (higher binds tighter), as in the Lua 5.4 manual, section 3.4.8.
func getOperatorPrecedence(op string) int {
	switch op {
	case "or", "||":
		return 1
	case "and", "&&":
		return 2
	case "<", ">", "<=", ">=", "~=", "!=", "==":
		return 3
	case "|":
		return 4
	case "~":
		return 5
	case "&":
		return 6
	case "<<", ">>":
		return 7
	case "..":
		return 8
	case "+", "-":
		return 9
	case "*", "/", "//", "%":
		return 10
	case "^":
		return 12
	default:
		return 0
	}
}

// escapeLuaString writes a string's bytes as the inside of a "..." Lua
// literal. Lua strings are bytes: valid UTF-8 is written as is, anything
// else (a lone 0xE9, control bytes) as a three-digit decimal escape, which
// a following digit cannot extend.
func (g *Generator) escapeLuaString(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '\n':
			result.WriteString("\\n")
		case '\t':
			result.WriteString("\\t")
		case '\r':
			result.WriteString("\\r")
		case '"':
			result.WriteString("\\\"")
		case '\\':
			result.WriteString("\\\\")
		default:
			if c >= 0x80 {
				if r, size := utf8.DecodeRuneInString(s[i:]); r != utf8.RuneError || size > 1 {
					result.WriteString(s[i : i+size])
					i += size
					continue
				}
			}
			if c < ' ' || c >= 0x7f {
				fmt.Fprintf(&result, "\\%03d", c)
			} else {
				result.WriteByte(c)
			}
		}
		i++
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
	output.WriteString(g.iifeOpen())
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
			// The pattern is checked before its bindings read the value:
			// { x: x } on a number must not index it.
			output.WriteString(g.iifeOpen() + "\n")
			output.WriteString("      if not (" + condition + ") then return false end\n")
			for _, binding := range bindings {
				output.WriteString("      ")
				output.WriteString(binding)
				output.WriteString("\n")
			}
			output.WriteString("      return (")
			output.WriteString(g.generateExpression(matchCase.Guard))
			output.WriteString(")\n    " + g.iifeClose() + " then\n")
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
	output.WriteString(g.iifeClose())

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

	case *ast.OrPattern:
		conds := make([]string, len(p.Alternatives))
		for i, alt := range p.Alternatives {
			cond, _ := g.generatePatternCondition(alt, valueName)
			conds[i] = "(" + cond + ")"
		}
		return strings.Join(conds, " or "), bindings

	case *ast.ValuePattern:
		return fmt.Sprintf("%s == %s", valueName, g.generateExpression(p.Value)), bindings

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
