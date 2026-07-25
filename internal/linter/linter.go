package linter

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
	"unicode"
)

// Severity represents the severity of a lint warning
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Issue represents a lint issue
type Issue struct {
	Message  string
	Line     int
	Column   int
	Severity Severity
	Rule     string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s [%s] (line %d, col %d): %s",
		i.Severity, i.Rule, i.Line, i.Column, i.Message)
}

// Options configures the linter
type Options struct {
	CheckNamingConventions bool
	CheckUnusedVariables   bool
	CheckMissingTypes      bool
	CheckEmptyBlocks       bool
}

// DefaultOptions returns default linter options
func DefaultOptions() Options {
	return Options{
		CheckNamingConventions: true,
		CheckUnusedVariables:   true,
		CheckMissingTypes:      true,
		CheckEmptyBlocks:       true,
	}
}

// Linter analyzes Lunar code for potential issues
type Linter struct {
	options    Options
	issues     []Issue
	variables  map[string]*variableInfo
	scopeStack []map[string]*variableInfo
}

type variableInfo struct {
	name    string
	line    int
	column  int
	used    bool
	isConst bool
}

// New creates a new linter
func New(options Options) *Linter {
	return &Linter{
		options:    options,
		issues:     []Issue{},
		variables:  make(map[string]*variableInfo),
		scopeStack: []map[string]*variableInfo{},
	}
}

// Lint analyzes a program and returns issues
func (l *Linter) Lint(statements []ast.Statement) []Issue {
	l.issues = []Issue{}
	l.variables = make(map[string]*variableInfo)
	l.scopeStack = []map[string]*variableInfo{}

	// Push global scope
	l.pushScope()

	// Analyze each statement
	for _, stmt := range statements {
		l.lintStatement(stmt)
	}

	// Check for unused variables in global scope
	if l.options.CheckUnusedVariables {
		l.checkUnusedVariables()
	}

	l.popScope()

	return l.issues
}

func (l *Linter) pushScope() {
	l.scopeStack = append(l.scopeStack, make(map[string]*variableInfo))
}

func (l *Linter) popScope() {
	if len(l.scopeStack) > 0 {
		l.scopeStack = l.scopeStack[:len(l.scopeStack)-1]
	}
}

func (l *Linter) currentScope() map[string]*variableInfo {
	if len(l.scopeStack) == 0 {
		return nil
	}
	return l.scopeStack[len(l.scopeStack)-1]
}

func (l *Linter) addVariable(name string, line, column int, isConst bool) {
	scope := l.currentScope()
	if scope != nil {
		scope[name] = &variableInfo{
			name:    name,
			line:    line,
			column:  column,
			used:    false,
			isConst: isConst,
		}
	}
}

func (l *Linter) markVariableUsed(name string) {
	// Search from innermost to outermost scope
	for i := len(l.scopeStack) - 1; i >= 0; i-- {
		if v, ok := l.scopeStack[i][name]; ok {
			v.used = true
			return
		}
	}
}

func (l *Linter) checkUnusedVariables() {
	for _, scope := range l.scopeStack {
		for _, v := range scope {
			if !v.used && !strings.HasPrefix(v.name, "_") {
				l.addIssue(Issue{
					Message:  fmt.Sprintf("Variable '%s' is declared but never used", v.name),
					Line:     v.line,
					Column:   v.column,
					Severity: SeverityWarning,
					Rule:     "unused-variable",
				})
			}
		}
	}
}

func (l *Linter) addIssue(issue Issue) {
	l.issues = append(l.issues, issue)
}

func (l *Linter) lintStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VariableDeclaration:
		l.lintVariableDeclaration(s)
	case *ast.FunctionDeclaration:
		l.lintFunctionDeclaration(s)
	case *ast.ClassDeclaration:
		l.lintClassDeclaration(s)
	case *ast.InterfaceDeclaration:
		l.lintInterfaceDeclaration(s)
	case *ast.EnumDeclaration:
		l.lintEnumDeclaration(s)
	case *ast.IfStatement:
		l.lintIfStatement(s)
	case *ast.WhileStatement:
		l.lintWhileStatement(s)
	case *ast.ForStatement:
		l.lintForStatement(s)
	case *ast.ReturnStatement:
		l.lintReturnStatement(s)
	case *ast.ExpressionStatement:
		l.lintExpression(s.Expression)
	case *ast.AssignmentStatement:
		for _, target := range s.Targets {
			l.lintExpression(target)
		}
		for _, value := range s.Values {
			l.lintExpression(value)
		}
	case *ast.BlockStatement:
		l.lintBlockStatement(s)
	}
}

func (l *Linter) lintVariableDeclaration(decl *ast.VariableDeclaration) {
	for i, name := range decl.Names {
		// Check naming convention for variables
		if l.options.CheckNamingConventions {
			if !isLowerCamelCase(name.Value) && !strings.HasPrefix(name.Value, "_") {
				l.addIssue(Issue{
					Message:  fmt.Sprintf("Variable '%s' should use camelCase naming", name.Value),
					Line:     name.Token.Line,
					Column:   name.Token.Column,
					Severity: SeverityInfo,
					Rule:     "naming-convention",
				})
			}
		}

		// Track variable for unused check
		l.addVariable(name.Value, name.Token.Line, name.Token.Column, decl.IsConstant)

		// Check for type annotation
		if l.options.CheckMissingTypes && (i >= len(decl.Types) || decl.Types[i] == nil) {
			// Only warn if there's no initializer (can't infer type)
			if i >= len(decl.Values) || decl.Values[i] == nil {
				l.addIssue(Issue{
					Message:  fmt.Sprintf("Variable '%s' has no type annotation", name.Value),
					Line:     name.Token.Line,
					Column:   name.Token.Column,
					Severity: SeverityInfo,
					Rule:     "missing-type",
				})
			}
		}
	}

	// Lint initializer expressions
	for _, val := range decl.Values {
		if val != nil {
			l.lintExpression(val)
		}
	}
}

func (l *Linter) lintFunctionDeclaration(fn *ast.FunctionDeclaration) {
	// Check naming convention
	if l.options.CheckNamingConventions && fn.Name != nil {
		if fn.Name.Value != "constructor" && !isLowerCamelCase(fn.Name.Value) {
			l.addIssue(Issue{
				Message:  fmt.Sprintf("Function '%s' should use camelCase naming", fn.Name.Value),
				Line:     fn.Name.Token.Line,
				Column:   fn.Name.Token.Column,
				Severity: SeverityInfo,
				Rule:     "naming-convention",
			})
		}
	}

	// Add function name to scope
	if fn.Name != nil {
		l.addVariable(fn.Name.Value, fn.Name.Token.Line, fn.Name.Token.Column, true)
	}

	// Check for return type annotation
	if l.options.CheckMissingTypes && fn.ReturnType == nil && fn.Name != nil && fn.Name.Value != "constructor" {
		l.addIssue(Issue{
			Message:  fmt.Sprintf("Function '%s' has no return type annotation", fn.Name.Value),
			Line:     fn.Token.Line,
			Column:   fn.Token.Column,
			Severity: SeverityInfo,
			Rule:     "missing-type",
		})
	}

	// Push scope for function body
	l.pushScope()

	// Add parameters to scope
	for _, param := range fn.Parameters {
		l.addVariable(param.Name.Value, param.Token.Line, param.Token.Column, false)
	}

	// Lint body
	if fn.Body != nil {
		// Check for empty function body
		if l.options.CheckEmptyBlocks && len(fn.Body.Statements) == 0 && !fn.IsAbstract {
			l.addIssue(Issue{
				Message:  fmt.Sprintf("Function '%s' has an empty body", fn.Name.Value),
				Line:     fn.Token.Line,
				Column:   fn.Token.Column,
				Severity: SeverityWarning,
				Rule:     "empty-block",
			})
		}

		for _, stmt := range fn.Body.Statements {
			l.lintStatement(stmt)
		}
	}

	// Check unused variables in function scope
	if l.options.CheckUnusedVariables {
		l.checkUnusedVariables()
	}

	l.popScope()
}

func (l *Linter) lintClassDeclaration(class *ast.ClassDeclaration) {
	// Check naming convention for class
	if l.options.CheckNamingConventions {
		if !isPascalCase(class.Name.Value) {
			l.addIssue(Issue{
				Message:  fmt.Sprintf("Class '%s' should use PascalCase naming", class.Name.Value),
				Line:     class.Name.Token.Line,
				Column:   class.Name.Token.Column,
				Severity: SeverityInfo,
				Rule:     "naming-convention",
			})
		}
	}

	// Add class to scope
	l.addVariable(class.Name.Value, class.Name.Token.Line, class.Name.Token.Column, true)

	// Lint methods
	for _, method := range class.Methods {
		l.lintFunctionDeclaration(method)
	}
}

func (l *Linter) lintInterfaceDeclaration(iface *ast.InterfaceDeclaration) {
	// Check naming convention
	if l.options.CheckNamingConventions {
		if !isPascalCase(iface.Name.Value) {
			l.addIssue(Issue{
				Message:  fmt.Sprintf("Interface '%s' should use PascalCase naming", iface.Name.Value),
				Line:     iface.Name.Token.Line,
				Column:   iface.Name.Token.Column,
				Severity: SeverityInfo,
				Rule:     "naming-convention",
			})
		}
	}

	// Add interface to scope
	l.addVariable(iface.Name.Value, iface.Name.Token.Line, iface.Name.Token.Column, true)
}

func (l *Linter) lintEnumDeclaration(enum *ast.EnumDeclaration) {
	// Check naming convention
	if l.options.CheckNamingConventions {
		if !isPascalCase(enum.Name.Value) {
			l.addIssue(Issue{
				Message:  fmt.Sprintf("Enum '%s' should use PascalCase naming", enum.Name.Value),
				Line:     enum.Name.Token.Line,
				Column:   enum.Name.Token.Column,
				Severity: SeverityInfo,
				Rule:     "naming-convention",
			})
		}
	}

	// Add enum to scope
	l.addVariable(enum.Name.Value, enum.Name.Token.Line, enum.Name.Token.Column, true)
}

func (l *Linter) lintIfStatement(stmt *ast.IfStatement) {
	l.lintExpression(stmt.Condition)

	// Check for empty if body
	if l.options.CheckEmptyBlocks && len(stmt.Consequence.Statements) == 0 {
		l.addIssue(Issue{
			Message:  "If statement has an empty body",
			Line:     stmt.Token.Line,
			Column:   stmt.Token.Column,
			Severity: SeverityWarning,
			Rule:     "empty-block",
		})
	}

	l.pushScope()
	for _, s := range stmt.Consequence.Statements {
		l.lintStatement(s)
	}
	l.popScope()

	if stmt.Alternative != nil {
		l.pushScope()
		for _, s := range stmt.Alternative.Statements {
			l.lintStatement(s)
		}
		l.popScope()
	}
}

func (l *Linter) lintWhileStatement(stmt *ast.WhileStatement) {
	l.lintExpression(stmt.Condition)

	// Check for empty while body
	if l.options.CheckEmptyBlocks && len(stmt.Body.Statements) == 0 {
		l.addIssue(Issue{
			Message:  "While loop has an empty body",
			Line:     stmt.Token.Line,
			Column:   stmt.Token.Column,
			Severity: SeverityWarning,
			Rule:     "empty-block",
		})
	}

	l.pushScope()
	for _, s := range stmt.Body.Statements {
		l.lintStatement(s)
	}
	l.popScope()
}

func (l *Linter) lintForStatement(stmt *ast.ForStatement) {
	l.pushScope()

	// Add loop variables to scope
	for _, v := range stmt.Variables {
		l.addVariable(v.Value, v.Token.Line, v.Token.Column, false)
	}

	// Check for empty for body
	if l.options.CheckEmptyBlocks && len(stmt.Body.Statements) == 0 {
		l.addIssue(Issue{
			Message:  "For loop has an empty body",
			Line:     stmt.Token.Line,
			Column:   stmt.Token.Column,
			Severity: SeverityWarning,
			Rule:     "empty-block",
		})
	}

	for _, s := range stmt.Body.Statements {
		l.lintStatement(s)
	}

	l.popScope()
}

func (l *Linter) lintReturnStatement(stmt *ast.ReturnStatement) {
	for _, val := range stmt.ReturnValues {
		l.lintExpression(val)
	}
}

func (l *Linter) lintBlockStatement(block *ast.BlockStatement) {
	l.pushScope()
	for _, stmt := range block.Statements {
		l.lintStatement(stmt)
	}
	if l.options.CheckUnusedVariables {
		l.checkUnusedVariables()
	}
	l.popScope()
}

func (l *Linter) lintExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		l.markVariableUsed(e.Value)
	case *ast.CallExpression:
		l.lintExpression(e.Function)
		for _, arg := range e.Arguments {
			l.lintExpression(arg)
		}
	case *ast.InfixExpression:
		l.lintExpression(e.Left)
		l.lintExpression(e.Right)
	case *ast.PrefixExpression:
		l.lintExpression(e.Right)
	case *ast.DotExpression:
		l.lintExpression(e.Left)
	case *ast.IndexExpression:
		l.lintExpression(e.Left)
		l.lintExpression(e.Index)
	}
}

// Helper functions for naming conventions

func isLowerCamelCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// First character should be lowercase
	if !unicode.IsLower(rune(s[0])) {
		return false
	}
	// No underscores (except at start)
	if strings.Contains(s[1:], "_") {
		return false
	}
	return true
}

func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// First character should be uppercase
	if !unicode.IsUpper(rune(s[0])) {
		return false
	}
	// No underscores
	if strings.Contains(s, "_") {
		return false
	}
	return true
}

// LintSource lints Lunar source code and returns issues
func LintSource(source string) ([]Issue, error) {
	l := lexer.New(source)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		var errMsgs []string
		for _, err := range p.Errors() {
			errMsgs = append(errMsgs, err.Error())
		}
		return nil, fmt.Errorf("parse errors:\n%s", strings.Join(errMsgs, "\n"))
	}

	linter := New(DefaultOptions())
	return linter.Lint(statements), nil
}
