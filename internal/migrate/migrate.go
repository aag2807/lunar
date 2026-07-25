package migrate

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
)

// Migrator converts Lua code to Lunar with type annotations
type Migrator struct {
	// Type inference data
	variableTypes map[string]string
	functionTypes map[string]*FunctionSignature
	// Migration options
	options Options
}

// Options configures migration behavior
type Options struct {
	// Add type annotations
	AddTypes bool
	// Convert to strict mode (use const where possible)
	UseConst bool
	// Add interface definitions for tables
	GenerateInterfaces bool
	// Preserve comments
	PreserveComments bool
	// Modernize syntax
	Modernize bool
}

// FunctionSignature represents inferred function type
type FunctionSignature struct {
	Parameters []ParamType
	Returns    []string
}

// ParamType represents a function parameter with type
type ParamType struct {
	Name string
	Type string
}

// New creates a new migrator
func New(options Options) *Migrator {
	return &Migrator{
		variableTypes: make(map[string]string),
		functionTypes: make(map[string]*FunctionSignature),
		options:       options,
	}
}

// Migrate converts Lua source code to Lunar
func (m *Migrator) Migrate(luaCode string) (string, error) {
	// Parse Lua code
	l := lexer.New(luaCode)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors: %v", p.Errors())
	}

	// Phase 1: Type inference
	m.inferTypes(statements)

	// Phase 2: Generate Lunar code
	return m.generate(statements), nil
}

// inferTypes performs type inference on statements
func (m *Migrator) inferTypes(statements []ast.Statement) {
	for _, stmt := range statements {
		m.inferStatement(stmt)
	}
}

func (m *Migrator) inferStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		m.inferVariableTypes(s)

	case *ast.FunctionDeclaration:
		m.inferFunctionType(s)

	case *ast.AssignmentStatement:
		m.inferAssignmentType(s)

	case *ast.ForStatement:
		m.inferForLoopTypes(s)

	case *ast.IfStatement:
		m.inferStatement(s.Consequence)
		if s.Alternative != nil {
			m.inferStatement(s.Alternative)
		}

	case *ast.BlockStatement:
		for _, blockStmt := range s.Statements {
			m.inferStatement(blockStmt)
		}
	}
}

func (m *Migrator) inferVariableTypes(decl *ast.VariableDeclaration) {
	for i, name := range decl.Names {
		if i < len(decl.Values) {
			inferredType := m.inferExpressionType(decl.Values[i])
			m.variableTypes[name.Value] = inferredType
		}
	}
}

func (m *Migrator) inferFunctionType(decl *ast.FunctionDeclaration) {
	sig := &FunctionSignature{
		Parameters: make([]ParamType, 0),
		Returns:    []string{},
	}

	// Infer parameter types from usage
	for _, param := range decl.Parameters {
		paramType := m.inferParameterType(param.Name.Value, decl.Body)
		sig.Parameters = append(sig.Parameters, ParamType{
			Name: param.Name.Value,
			Type: paramType,
		})
	}

	// Infer return type
	sig.Returns = m.inferReturnTypes(decl.Body)

	m.functionTypes[decl.Name.Value] = sig
}

func (m *Migrator) inferAssignmentType(stmt *ast.AssignmentStatement) {
	for i, target := range stmt.Targets {
		ident, ok := target.(*ast.Identifier)
		if !ok {
			continue
		}
		// Only a matching value position tells us anything about the type;
		// extra targets take whatever a multi-value call returns.
		if i >= len(stmt.Values) {
			m.variableTypes[ident.Value] = "any"
			continue
		}
		m.variableTypes[ident.Value] = m.inferExpressionType(stmt.Values[i])
	}
}

func (m *Migrator) inferForLoopTypes(stmt *ast.ForStatement) {
	for _, variable := range stmt.Variables {
		if stmt.IsGeneric {
			m.variableTypes[variable.Value] = "any"
		} else {
			m.variableTypes[variable.Value] = "number"
		}
	}
}

func (m *Migrator) inferExpressionType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		return "number"
	case *ast.StringLiteral:
		return "string"
	case *ast.BooleanLiteral:
		return "boolean"
	case *ast.NilLiteral:
		return "nil"
	case *ast.TableLiteral:
		return m.inferTableType(e)
	case *ast.FunctionExpression:
		return "function"
	case *ast.Identifier:
		if typ, ok := m.variableTypes[e.Value]; ok {
			return typ
		}
		return "any"
	case *ast.InfixExpression:
		return m.inferInfixType(e)
	default:
		return "any"
	}
}

func (m *Migrator) inferTableType(table *ast.TableLiteral) string {
	if len(table.Pairs) == 0 && len(table.Values) > 0 {
		if len(table.Values) > 0 {
			elemType := m.inferExpressionType(table.Values[0])
			return fmt.Sprintf("array<%s>", elemType)
		}
		return "array<any>"
	}

	if len(table.Pairs) > 0 {
		return "table<string, any>"
	}

	return "table<any, any>"
}

func (m *Migrator) inferInfixType(infix *ast.InfixExpression) string {
	switch infix.Operator {
	case "+", "-", "*", "/", "%", "^":
		return "number"
	case "==", "!=", "<", "<=", ">", ">=", "and", "or":
		return "boolean"
	case "..":
		return "string"
	default:
		return "any"
	}
}

func (m *Migrator) inferParameterType(paramName string, body *ast.BlockStatement) string {
	// Simple heuristic: if used in arithmetic, it's a number
	// This is a basic implementation - could be enhanced
	return "any"
}

func (m *Migrator) inferReturnTypes(body *ast.BlockStatement) []string {
	types := []string{}

	for _, stmt := range body.Statements {
		if ret, ok := stmt.(*ast.ReturnStatement); ok {
			if len(ret.ReturnValues) > 0 {
				typ := m.inferExpressionType(ret.ReturnValues[0])
				types = append(types, typ)
			} else {
				types = append(types, "void")
			}
		}
	}

	if len(types) == 0 {
		return []string{"void"}
	}

	return types
}

// generate creates Lunar code from statements with type annotations
func (m *Migrator) generate(statements []ast.Statement) string {
	var output strings.Builder

	for _, stmt := range statements {
		output.WriteString(m.generateStatement(stmt))
		output.WriteString("\n")
	}

	return output.String()
}

func (m *Migrator) generateStatement(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		return m.generateVariableDeclaration(s)

	case *ast.FunctionDeclaration:
		return m.generateFunctionDeclaration(s)

	case *ast.AssignmentStatement:
		return m.generateAssignment(s)

	case *ast.ReturnStatement:
		return m.generateReturn(s)

	case *ast.IfStatement:
		return m.generateIf(s)

	case *ast.ForStatement:
		return m.generateFor(s)

	case *ast.WhileStatement:
		return m.generateWhile(s)

	case *ast.ExpressionStatement:
		return m.generateExpression(s.Expression)

	default:
		return stmt.String()
	}
}

func (m *Migrator) generateVariableDeclaration(decl *ast.VariableDeclaration) string {
	var output strings.Builder

	keyword := "local"
	if m.options.UseConst && decl.IsConstant {
		keyword = "const"
	}

	output.WriteString(keyword)
	output.WriteString(" ")

	for i, name := range decl.Names {
		if i > 0 {
			output.WriteString(", ")
		}

		output.WriteString(name.Value)

		// Add type annotation
		if m.options.AddTypes {
			if typ, ok := m.variableTypes[name.Value]; ok && typ != "any" {
				output.WriteString(": ")
				output.WriteString(typ)
			}
		}
	}

	if len(decl.Values) > 0 {
		output.WriteString(" = ")
		for i, val := range decl.Values {
			if i > 0 {
				output.WriteString(", ")
			}
			output.WriteString(m.generateExpression(val))
		}
	}

	return output.String()
}

func (m *Migrator) generateFunctionDeclaration(decl *ast.FunctionDeclaration) string {
	var output strings.Builder

	output.WriteString("function ")
	output.WriteString(decl.Name.Value)
	output.WriteString("(")

	sig, hasSig := m.functionTypes[decl.Name.Value]

	for i, param := range decl.Parameters {
		if i > 0 {
			output.WriteString(", ")
		}

		output.WriteString(param.Name.Value)

		// Add type annotation
		if m.options.AddTypes && hasSig && i < len(sig.Parameters) {
			paramType := sig.Parameters[i].Type
			if paramType != "any" {
				output.WriteString(": ")
				output.WriteString(paramType)
			}
		}
	}

	output.WriteString(")")

	// Add return type
	if m.options.AddTypes && hasSig && len(sig.Returns) > 0 {
		returnType := sig.Returns[0]
		if returnType != "any" && returnType != "void" {
			output.WriteString(": ")
			output.WriteString(returnType)
		}
	}

	output.WriteString("\n")

	// Generate body
	if decl.Body != nil {
		for _, stmt := range decl.Body.Statements {
			output.WriteString("\t")
			output.WriteString(m.generateStatement(stmt))
			output.WriteString("\n")
		}
	}

	output.WriteString("end")

	return output.String()
}

func (m *Migrator) generateAssignment(stmt *ast.AssignmentStatement) string {
	targets := make([]string, len(stmt.Targets))
	for i, target := range stmt.Targets {
		targets[i] = m.generateExpression(target)
	}

	values := make([]string, len(stmt.Values))
	for i, value := range stmt.Values {
		values[i] = m.generateExpression(value)
	}

	return fmt.Sprintf("%s = %s", strings.Join(targets, ", "), strings.Join(values, ", "))
}

func (m *Migrator) generateReturn(stmt *ast.ReturnStatement) string {
	if len(stmt.ReturnValues) == 0 {
		return "return"
	}

	var values []string
	for _, val := range stmt.ReturnValues {
		values = append(values, m.generateExpression(val))
	}

	return "return " + strings.Join(values, ", ")
}

func (m *Migrator) generateIf(stmt *ast.IfStatement) string {
	var output strings.Builder

	output.WriteString("if ")
	output.WriteString(m.generateExpression(stmt.Condition))
	output.WriteString(" then\n")

	for _, s := range stmt.Consequence.Statements {
		output.WriteString("\t")
		output.WriteString(m.generateStatement(s))
		output.WriteString("\n")
	}

	if stmt.Alternative != nil {
		output.WriteString("else\n")
		for _, s := range stmt.Alternative.Statements {
			output.WriteString("\t")
			output.WriteString(m.generateStatement(s))
			output.WriteString("\n")
		}
	}

	output.WriteString("end")

	return output.String()
}

func (m *Migrator) generateFor(stmt *ast.ForStatement) string {
	var output strings.Builder

	output.WriteString("for ")

	for i, variable := range stmt.Variables {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(variable.Value)

		// Add type annotation
		if m.options.AddTypes {
			if typ, ok := m.variableTypes[variable.Value]; ok && typ != "any" {
				output.WriteString(": ")
				output.WriteString(typ)
			}
		}
	}

	if stmt.IsGeneric {
		output.WriteString(" in ")
		output.WriteString(m.generateExpression(stmt.Iterator))
	} else {
		output.WriteString(" = ")
		output.WriteString(m.generateExpression(stmt.Start))
		output.WriteString(", ")
		output.WriteString(m.generateExpression(stmt.End))
		if stmt.Step != nil {
			output.WriteString(", ")
			output.WriteString(m.generateExpression(stmt.Step))
		}
	}

	output.WriteString(" do\n")

	for _, s := range stmt.Body.Statements {
		output.WriteString("\t")
		output.WriteString(m.generateStatement(s))
		output.WriteString("\n")
	}

	output.WriteString("end")

	return output.String()
}

func (m *Migrator) generateWhile(stmt *ast.WhileStatement) string {
	var output strings.Builder

	output.WriteString("while ")
	output.WriteString(m.generateExpression(stmt.Condition))
	output.WriteString(" do\n")

	for _, s := range stmt.Body.Statements {
		output.WriteString("\t")
		output.WriteString(m.generateStatement(s))
		output.WriteString("\n")
	}

	output.WriteString("end")

	return output.String()
}

func (m *Migrator) generateExpression(expr ast.Expression) string {
	return expr.String()
}
