package types

import (
	"fmt"
	"io/fs"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/stdlib"
	"os"
	"reflect"
	"sort"
	"strings"
)

// TypeError represents a type error
type TypeError struct {
	Message string
	Line    int
	Column  int
	// File is the path of the source the error is in; EndLine and
	// EndColumn are just past the offending span.
	File      string
	EndLine   int
	EndColumn int
	// Code is a stable kebab-case name for the kind of error, "type" by
	// default; see addErrorCode.
	Code string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("Type error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// Environment represents a scope with type bindings
type Environment struct {
	store     map[string]Type
	constVars map[string]bool // tracks which variables are const
	// narrow holds what a check has refined a variable to in this scope
	// (x ~= nil, not x, x and ...), over its declared type in store.
	narrow map[string]Type
	// inferred marks the locals declared without a type annotation.
	inferred map[string]bool
	outer    *Environment
	syms     map[string]*Symbol // definitions made in this scope, for the index
}

// NewEnvironment creates a new environment
func NewEnvironment() *Environment {
	return &Environment{
		store:     make(map[string]Type),
		constVars: make(map[string]bool),
		outer:     nil,
	}
}

// NewEnclosedEnvironment creates a new environment with an outer scope
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a type from the environment
func (e *Environment) Get(name string) (Type, bool) {
	if typ, ok := e.narrow[name]; ok {
		return typ, true
	}
	typ, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return typ, ok
}

// Narrow records what a variable is known to be from here on in this
// scope, without changing what it was declared as.
func (e *Environment) Narrow(name string, typ Type) {
	if e.narrow == nil {
		e.narrow = map[string]Type{}
	}
	e.narrow[name] = typ
}

// DeclaredType is a variable's type as declared, ignoring narrowing: what
// an assignment to it has to fit.
func (e *Environment) DeclaredType(name string) (Type, bool) {
	for env := e; env != nil; env = env.outer {
		if typ, ok := env.store[name]; ok {
			return typ, true
		}
	}
	return nil, false
}

// Set sets a type in the environment
func (e *Environment) Set(name string, typ Type) {
	e.store[name] = typ
	delete(e.narrow, name)
}

// SetLocal binds a new, assignable local: one declared over a const of
// the same scope (local table at the top of a module) is not const.
func (e *Environment) SetLocal(name string, typ Type) {
	e.Set(name, typ)
	delete(e.constVars, name)
}

// SetConst sets a variable as const in the environment
func (e *Environment) SetConst(name string, typ Type) {
	e.store[name] = typ
	e.constVars[name] = true
}

// IsConst checks if a variable is const: the one the name refers to here,
// so a parameter named string is not the const string library.
func (e *Environment) IsConst(name string) bool {
	if _, ok := e.store[name]; ok {
		return e.constVars[name]
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

// MarkInferred records that the local name was declared without a type.
func (e *Environment) MarkInferred(name string) {
	if e.inferred == nil {
		e.inferred = map[string]bool{}
	}
	e.inferred[name] = true
}

// IsInferred says whether the variable the name refers to here was
// declared without a type.
func (e *Environment) IsInferred(name string) bool {
	if _, ok := e.store[name]; ok {
		return e.inferred[name]
	}
	if e.outer != nil {
		return e.outer.IsInferred(name)
	}
	return false
}

// GetAllNames returns all variable names in this environment and outer scopes
func (e *Environment) GetAllNames() []string {
	names := make([]string, 0, len(e.store))
	seen := make(map[string]bool)

	// Collect names from current scope
	for name := range e.store {
		names = append(names, name)
		seen[name] = true
	}

	// Collect names from outer scopes (avoiding duplicates)
	if e.outer != nil {
		for _, name := range e.outer.GetAllNames() {
			if !seen[name] {
				names = append(names, name)
				seen[name] = true
			}
		}
	}

	return names
}

// GetAll returns all type bindings in this environment and outer scopes
func (e *Environment) GetAll() map[string]Type {
	result := make(map[string]Type)

	// Collect from outer scopes first (so inner scopes override)
	if e.outer != nil {
		for name, typ := range e.outer.GetAll() {
			result[name] = typ
		}
	}

	// Collect from current scope (overrides outer)
	for name, typ := range e.store {
		result[name] = typ
	}

	return result
}

// levenshteinDistance calculates the edit distance between two strings
// Used for "Did you mean?" suggestions
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix for dynamic programming
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// findSimilarNames finds names similar to the target name using Levenshtein distance
// Returns up to maxSuggestions names with distance <= maxDistance
func findSimilarNames(target string, candidates []string, maxDistance int, maxSuggestions int) []string {
	type suggestion struct {
		name     string
		distance int
	}

	suggestions := make([]suggestion, 0)

	// Calculate distances for all candidates
	for _, candidate := range candidates {
		distance := levenshteinDistance(target, candidate)
		if distance <= maxDistance && distance > 0 {
			suggestions = append(suggestions, suggestion{candidate, distance})
		}
	}

	// Closest first, and alphabetically among equals: the candidates
	// often come from a map, whose order changes from run to run, so the
	// suggestion did too (the editor and the CLI disagreed).
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].distance != suggestions[j].distance {
			return suggestions[i].distance < suggestions[j].distance
		}
		return suggestions[i].name < suggestions[j].name
	})

	// Return top suggestions
	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].name)
	}

	return result
}

// Checker performs type checking on an AST
type Checker struct {
	env    *Environment
	errors []*TypeError

	// Type definitions (classes, interfaces, enums, type aliases)
	classes           map[string]*ClassType
	interfaces        map[string]*InterfaceType
	genericInterfaces map[string]*GenericInterfaceType
	// accessedDeclarer is the class declaring the member being accessed,
	// for the private check.
	accessedDeclarer *ClassType
	// laterLocals are local functions further down a block than their
	// first use, for saying why such a use is undefined.
	laterLocals map[string]lexer.Token
	// namespaces are the types of the namespace declarations seen.
	namespaces map[*ast.NamespaceDeclaration]*NamespaceType
	// sharedMemberSyms are the member symbols of the generic interface an
	// instance is being built from.
	sharedMemberSyms   map[string]*Symbol
	enums              map[string]*EnumType
	typeAliases        map[string]Type
	genericTypeAliases map[string]*GenericTypeAlias

	// Module system
	exports       map[string]Type        // Tracks exported names and their types
	modules       map[string]*ModuleInfo // Tracks loaded modules
	currentModule string                 // Current module being checked

	// Standard library path
	stdlibPath string // Directory to read stdlib declarations from; "" for the bundled copy
	target     string // Lua version being compiled for

	// Current function return type (for checking return statements)
	currentFunctionReturnType Type
	// inferringReturn is set while checking a function that declared no return
	// type; the return statements found are collected in inferredReturns.
	inferringReturn bool
	inferredReturns []Type
	// hoisted holds the placeholder signatures registered ahead of their
	// declaration, so the real declaration replaces them rather than being
	// treated as an overload of itself.
	hoisted map[string]*FunctionType

	// declaredFuncs are the function types that came from `declare`, which
	// an implementation may provide; funcDefs records where each
	// implemented function was defined.
	declaredFuncs map[*FunctionType]bool
	funcDefs      map[*FunctionType]lexer.Token

	// importResolver finds the module an import names; without one, every
	// import is unresolved. boundImports are those pass zero has bound.
	importResolver ImportResolver
	boundImports   map[*ast.ImportStatement]bool

	// index records definitions and their uses for editors; scopeEnd is
	// where the function body being checked ends (-1 at the top level).
	index    Index
	scopeEnd int
	typeSyms map[string]*Symbol

	exportSyms map[string]*Symbol

	// ambient maps each name a declaration bound (the stdlib, .d.lunar
	// files) to its type; globalUses are the identifiers that resolved to
	// one of those bindings, which is what a data file may not do.
	ambient    map[string]Type
	globalUses []lexer.Token
	// stringMethodUses are calls through a string's metatable (s:upper()),
	// which only exists once the string library is opened.
	stringMethodUses []lexer.Token
	// stringDots are the s.m expressions that resolved through the string
	// library; a call through one may be plain Lua's s.m(s, ...).
	stringDots map[*ast.DotExpression]*FunctionType
	// resolvingTypeof: a type alias using typeof is being resolved where it
	// is declared, not in the early registration pass.
	resolvingTypeof bool
	// classInstances are the instantiations of generic classes made so
	// far, by class and arguments.
	classInstances map[string]*ClassType

	// exprTypes records the type computed for each expression checked.
	exprTypes map[ast.Expression]Type
	// callee is the expression a call is about to check as its function,
	// so a method named there is known to be called, not read as a value.
	callee ast.Expression

	// chunkReturns are the return statements of the chunk itself, outside
	// any function.
	chunkReturns []*ast.ReturnStatement

	// shells holds the empty interface, class and enum types published by
	// predeclareTypeNames, keyed by name, until their declaration fills them.
	shells map[string]Type

	// Current class context (for visibility checking)
	currentClass *ClassType

	// Track if we're inside a constructor (for readonly assignment checking)
	inConstructor bool
}

// ModuleInfo represents information about a module
type ModuleInfo struct {
	Path    string
	Exports map[string]Type
	// Symbols are where each export is defined, for editors.
	Symbols map[string]*Symbol
	// Broken is set when the module does not parse: its own syntax errors
	// are reported, and what it would export is taken as 'any' here.
	Broken bool
}

// ImportResolver returns the module an import statement names, or an error
// whose message is reported at the import.
type ImportResolver func(node *ast.ImportStatement) (*ModuleInfo, error)

// SetImportResolver installs how this checker finds imported modules. The
// project resolves them against the module graph, relative to the importing
// file; nothing else in the checker looks at the file system.
func (c *Checker) SetImportResolver(r ImportResolver) {
	c.importResolver = r
}

// Exports returns the names the checked module exports and their types.
func (c *Checker) Exports() map[string]Type {
	return c.exports
}

// ExportSymbols returns where each exported name is defined.
func (c *Checker) ExportSymbols() map[string]*Symbol {
	return c.exportSyms
}

// NewChecker creates a new type checker
// If stdlibPath is provided, it will be used to locate stdlib files
// Otherwise, it will search relative to the current directory (for backwards compatibility)
func NewChecker(stdlibPath ...string) *Checker {
	var dir string
	if len(stdlibPath) > 0 {
		dir = stdlibPath[0]
	}
	return NewCheckerWithOptions(Options{StdlibDir: dir})
}

// Options configures a Checker.
type Options struct {
	// StdlibDir reads the standard library declarations from a directory
	// instead of the copy compiled into lunar.
	StdlibDir string
	// Target is the Lua version, "lua53" (the default) or "lua54"; it
	// selects the declarations only that version has.
	Target string
}

// NewCheckerWithOptions creates a checker with the standard library loaded.
func NewCheckerWithOptions(opts Options) *Checker {
	env := NewEnvironment()

	// Register built-in types
	env.Set("number", Number)
	env.Set("string", String)
	env.Set("boolean", Boolean)
	env.Set("nil", Nil)
	env.Set("void", Void)
	env.Set("any", Any)
	env.Set("never", Never)
	env.Set("unknown", Unknown)

	checker := &Checker{
		env:                env,
		errors:             []*TypeError{},
		classes:            make(map[string]*ClassType),
		interfaces:         make(map[string]*InterfaceType),
		genericInterfaces:  make(map[string]*GenericInterfaceType),
		namespaces:         make(map[*ast.NamespaceDeclaration]*NamespaceType),
		enums:              make(map[string]*EnumType),
		typeAliases:        make(map[string]Type),
		genericTypeAliases: make(map[string]*GenericTypeAlias),
		exports:            make(map[string]Type),
		modules:            make(map[string]*ModuleInfo),
		currentModule:      "",
		stdlibPath:         opts.StdlibDir,
		target:             opts.Target,
		scopeEnd:           -1,
		typeSyms:           make(map[string]*Symbol),
		exportSyms:         make(map[string]*Symbol),
	}

	// Register built-in utility types
	checker.registerUtilityTypes()

	// Load standard library declarations
	checker.loadStdlib()

	return checker
}

// registerUtilityTypes registers built-in utility types like Exclude, Extract, etc.
func (c *Checker) registerUtilityTypes() {
	// These will be handled specially in resolveTypeExpression when instantiated
	// We just mark them as existing so they can be referenced
	c.env.Set("Exclude", Any)     // Exclude<T, U>
	c.env.Set("Extract", Any)     // Extract<T, U>
	c.env.Set("NonNullable", Any) // NonNullable<T>
	c.env.Set("Pick", Any)        // Pick<T, K>
	c.env.Set("Omit", Any)        // Omit<T, K>
	c.env.Set("Record", Any)      // Record<K, V>
	c.env.Set("Partial", Any)     // Partial<T>
	c.env.Set("Required", Any)    // Required<T>
	c.env.Set("Readonly", Any)    // Readonly<T>
	c.env.Set("ReturnType", Any)  // ReturnType<F>
	c.env.Set("Parameters", Any)  // Parameters<F>
}

// loadStdlib loads the Lua standard library type definitions
func (c *Checker) loadStdlib() {
	// Lua's own standard libraries are always in scope; they are compiled
	// into lunar, or read from StdlibDir when one is given.
	var files fs.FS = stdlib.Files
	if c.stdlibPath != "" {
		files = os.DirFS(c.stdlibPath)
	}

	names := append(append([]string{}, stdlib.Core...), stdlib.ForTarget(c.target)...)
	for _, filename := range names {
		content, err := fs.ReadFile(files, filename)
		if err != nil {
			continue // a test directory may hold only some of them
		}

		// Stdlib errors are reported as <stdlib>/<file>, wherever it lives.
		l := lexer.NewFile("<stdlib>/"+filename, string(content))
		p := parser.New(l)
		statements := p.Parse()

		if len(p.Errors()) > 0 {
			// A broken stdlib file would leave its names undefined with no
			// hint why; report its parse errors instead.
			for _, e := range p.Errors() {
				c.errors = append(c.errors, &TypeError{
					Message: e.Message, Line: e.Line, Column: e.Column,
					File: e.File, EndLine: e.EndLine, EndColumn: e.EndColumn, Code: "syntax",
				})
			}
			continue
		}

		// Register stdlib declarations
		c.predeclareTypeNames(statements)
		for _, stmt := range statements {
			c.registerTypeDefinition(stmt)
		}

		// Check stdlib statements to register functions and variables
		for _, stmt := range statements {
			c.checkStatement(stmt)
		}
	}
}

// Check performs type checking on a list of statements
func (c *Checker) Check(statements []ast.Statement) []*TypeError {
	// Imports come first, so imported types can be named in annotations and
	// `extends` clauses that the next passes resolve.
	for _, stmt := range statements {
		if imp, ok := stmt.(*ast.ImportStatement); ok {
			c.bindImport(imp)
		}
	}

	// Pass zero: publish every interface, class and enum name, so a
	// declaration can refer to itself or to one further down.
	c.predeclareTypeNames(statements)

	// First pass: register all type definitions
	for _, stmt := range statements {
		c.registerTypeDefinition(stmt)
	}

	// Second pass: publish function signatures so a body can call a function
	// declared further down the file.
	c.hoistFunctionSignatures(statements)

	// Third pass: check all statements
	for _, stmt := range statements {
		c.checkStatement(stmt)
	}

	c.checkModuleShape(statements)
	c.checkGotos(statements)

	return c.errors
}

// attribOf is the Lua 5.4 attribute of a declaration's i-th name, or "".
func attribOf(node *ast.VariableDeclaration, i int) string {
	if i < len(node.Attribs) {
		return node.Attribs[i]
	}
	return ""
}

// checkAttribs checks Lua 5.4's <const> and <close>.
func (c *Checker) checkAttribs(node *ast.VariableDeclaration) {
	for i, attrib := range node.Attribs {
		switch attrib {
		case "", "const":
		case "close":
			if c.target != "lua54" {
				c.addError("'<close>' needs Lua 5.4: set \"target\": \"lua54\"", node.Names[i].Token)
			}
		default:
			c.addError(fmt.Sprintf("unknown attribute '%s' (Lua has <const> and <close>)", attrib), node.Names[i].Token)
		}
	}
}

// checkGotos checks every goto against the labels Lua would let it reach,
// as Lua does when it loads the file: a label in the same block or one
// around it, in the same function, not past the start of a local's scope.
func (c *Checker) checkGotos(statements []ast.Statement) {
	for _, g := range c.gotosIn(statements) {
		c.addError(fmt.Sprintf("no visible label '%s' for goto", g.Label.Value), g.Token)
	}
}

// gotosIn resolves the gotos in a block, nested blocks included, against
// the block's labels and returns those left for an enclosing block.
func (c *Checker) gotosIn(stmts []ast.Statement) []*ast.GotoStatement {
	labels := map[string]int{}
	for i, stmt := range stmts {
		if l, ok := stmt.(*ast.LabelStatement); ok {
			if _, dup := labels[l.Name.Value]; dup {
				c.addError(fmt.Sprintf("label '%s' already defined", l.Name.Value), l.Name.Token)
				continue
			}
			labels[l.Name.Value] = i
		}
	}
	var unresolved []*ast.GotoStatement
	for i, stmt := range stmts {
		// Declarations and types hold no code, so no goto: walking them
		// cost as much as checking a large engine API.
		switch stmt.(type) {
		case *ast.DeclareStatement, *ast.InterfaceDeclaration, *ast.TypeDeclaration, *ast.EnumDeclaration, *ast.ImportStatement:
			continue
		}
		var found []*ast.GotoStatement
		if g, ok := stmt.(*ast.GotoStatement); ok {
			found = []*ast.GotoStatement{g}
		} else {
			found = c.nestedGotos(reflect.ValueOf(stmt))
		}
		for _, g := range found {
			j, ok := labels[g.Label.Value]
			if !ok {
				unresolved = append(unresolved, g)
				continue
			}
			if j > i && !labelEndsBlock(stmts, j) {
				if name := localBetween(stmts, i, j); name != "" {
					c.addError(fmt.Sprintf("goto %s jumps into the scope of local '%s'", g.Label.Value, name), g.Token)
				}
			}
		}
	}
	return unresolved
}

// nestedGotos finds the gotos inside a statement that its own blocks do
// not resolve. A function inside it is checked on its own: a goto cannot
// leave a function.
func (c *Checker) nestedGotos(v reflect.Value) []*ast.GotoStatement {
	return c.nestedGotosOnce(v, map[uintptr]bool{})
}

// nestedGotosOnce visits each node once: a table literal's fields are
// reached both through its pairs and its entries, which made the walk
// exponential in nesting depth.
func (c *Checker) nestedGotosOnce(v reflect.Value, seen map[uintptr]bool) []*ast.GotoStatement {
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Ptr {
			if seen[v.Pointer()] {
				return nil
			}
			seen[v.Pointer()] = true
		}
		switch n := v.Interface().(type) {
		case *ast.BlockStatement:
			return c.gotosIn(n.Statements)
		case *ast.FunctionDeclaration, *ast.FunctionExpression,
			*ast.ConstructorDeclaration, *ast.GetterDeclaration, *ast.SetterDeclaration:
			if body := reflect.Indirect(reflect.ValueOf(n)).FieldByName("Body"); body.IsValid() {
				if block, ok := body.Interface().(*ast.BlockStatement); ok && block != nil {
					c.checkGotos(block.Statements)
				}
			}
			return nil
		}
		return c.nestedGotosOnce(v.Elem(), seen)
	case reflect.Struct:
		var out []*ast.GotoStatement
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				out = append(out, c.nestedGotosOnce(v.Field(i), seen)...)
			}
		}
		return out
	case reflect.Slice:
		var out []*ast.GotoStatement
		for i := 0; i < v.Len(); i++ {
			out = append(out, c.nestedGotosOnce(v.Index(i), seen)...)
		}
		return out
	case reflect.Map:
		var out []*ast.GotoStatement
		for _, k := range v.MapKeys() {
			out = append(out, c.nestedGotosOnce(v.MapIndex(k), seen)...)
		}
		return out
	}
	return nil
}

// labelEndsBlock says whether only labels follow the label at j: Lua lets
// a goto jump there past locals, since their scope ends with the block.
func labelEndsBlock(stmts []ast.Statement, j int) bool {
	for _, stmt := range stmts[j+1:] {
		if _, isLabel := stmt.(*ast.LabelStatement); !isLabel && !isNilNode(stmt) {
			return false
		}
	}
	return true
}

// localBetween names a local declared directly in stmts between i and j.
func localBetween(stmts []ast.Statement, i, j int) string {
	for _, stmt := range stmts[i+1 : j] {
		switch n := stmt.(type) {
		case *ast.VariableDeclaration:
			if len(n.Names) > 0 {
				return n.Names[0].Value
			}
		case *ast.FunctionDeclaration:
			if n.IsLocal && n.Name != nil {
				return n.Name.Value
			}
		}
	}
	return ""
}

// checkModuleShape rejects a file that both exports names and returns a
// value from its chunk: the module's value is one or the other, and the
// generated Lua would otherwise end in two returns.
func (c *Checker) checkModuleShape(statements []ast.Statement) {
	if len(c.chunkReturns) == 0 {
		return
	}
	for _, stmt := range statements {
		if _, ok := stmt.(*ast.ExportStatement); ok {
			for _, ret := range c.chunkReturns {
				if len(ret.ReturnValues) > 0 {
					c.addError("A module cannot both 'export' names and 'return' a value", ret.Token)
					return
				}
			}
			return
		}
	}
}

// hoistFunctionSignatures registers a placeholder signature for every function
// declared in a statement list, before any body is checked. A Lua function
// declaration binds a global, so calling one declared later in the file is
// normal; without this, forward references and mutual recursion both fail with
// "Undefined variable".
func (c *Checker) hoistFunctionSignatures(statements []ast.Statement) {
	c.hoistFunctions(statements, false)
}

// hoistFunctions makes the functions of a block callable before their
// declarations. A local function is in scope only after it, as in Lua,
// unless withLocals (a namespace, whose block declares its locals first).
func (c *Checker) hoistFunctions(statements []ast.Statement, withLocals bool) {
	// A name the file declares is published by its declaration, which an
	// implementation is then checked against; a placeholder would get in
	// the way.
	declared := map[string]bool{}
	for _, stmt := range statements {
		if ds, ok := stmt.(*ast.DeclareStatement); ok {
			if fn, ok := ds.Declaration.(*ast.FunctionDeclaration); ok && fn.Name != nil {
				declared[fn.Name.Value] = true
			}
		}
	}

	for _, stmt := range statements {
		if exp, ok := stmt.(*ast.ExportStatement); ok {
			stmt = exp.Statement
		}
		fn, ok := stmt.(*ast.FunctionDeclaration)
		if !ok || fn.Name == nil || fn.Receiver != nil || declared[fn.Name.Value] {
			continue
		}
		if fn.IsLocal && !withLocals {
			if c.laterLocals == nil {
				c.laterLocals = map[string]lexer.Token{}
			}
			if _, seen := c.laterLocals[fn.Name.Value]; !seen {
				c.laterLocals[fn.Name.Value] = fn.Name.Token
			}
			continue
		}

		// A name that already resolves is left alone: it may be an overload set
		// being built up, or a local shadowing something.
		if _, exists := c.env.Get(fn.Name.Value); exists {
			continue
		}

		placeholder := c.hoistedSignature(fn)
		if c.hoisted == nil {
			c.hoisted = make(map[string]*FunctionType)
		}
		c.hoisted[fn.Name.Value] = placeholder
		c.env.Set(fn.Name.Value, placeholder)
	}
}

// hoistedSignature builds the placeholder type for a hoisted function. Types
// that cannot be resolved yet fall back to 'any'; the real declaration replaces
// the whole signature when it is checked.
func (c *Checker) hoistedSignature(fn *ast.FunctionDeclaration) *FunctionType {
	// The signature's own type parameters have to be in scope while its
	// parameter and return types resolve.
	prevEnv := c.env
	if len(fn.GenericParams) > 0 {
		c.env = NewEnclosedEnvironment(prevEnv)
		for _, genericParam := range fn.GenericParams {
			c.env.Set(genericParam.Value, &GenericParamType{Name: genericParam.Value})
		}
		defer func() { c.env = prevEnv }()
	}

	params := make([]Type, len(fn.Parameters))
	optionalParams := make([]bool, len(fn.Parameters))
	hasRestParam := false

	for i, param := range fn.Parameters {
		paramType := c.paramType(param)
		if param.IsRest {
			hasRestParam = true
			if _, ok := paramType.(*ArrayType); !ok {
				paramType = &ArrayType{ElementType: paramType}
			}
		}
		optionalParams[i] = paramIsOptional(param, paramType)
		params[i] = paramType
	}

	// An unannotated return type is only known once the body is checked.
	var returnType Type = Any
	if fn.ReturnType != nil {
		returnType = c.resolveTypeExpression(fn.ReturnType)
	}

	genericParams := make([]string, len(fn.GenericParams))
	for i, param := range fn.GenericParams {
		genericParams[i] = param.Value
	}

	return &FunctionType{
		Parameters:       params,
		ReturnType:       returnType,
		GenericParams:    genericParams,
		HasRestParameter: hasRestParam,
		OptionalParams:   optionalParams,
		Untyped:          untypedSignature(fn.Parameters, fn.ReturnType, len(fn.GenericParams)),
	}
}

// untypedSignature says whether a function is written with no annotation
// at all, as plain Lua is: it is called as Lua calls it, missing arguments
// nil and extra ones dropped.
func untypedSignature(params []*ast.Parameter, ret ast.Expression, generics int) bool {
	if ret != nil || generics > 0 {
		return false
	}
	for _, p := range params {
		if p.Type != nil || p.IsOptional {
			return false
		}
	}
	return true
}

// GetEnv returns the type environment for LSP integration
func (c *Checker) GetEnv() *Environment {
	return c.env
}

// PreloadDeclarations adds types from a declaration environment to the checker
// This is used by the LSP to inject types from .d.lunar files before checking
func (c *Checker) PreloadDeclarations(declEnv *Environment) {
	if declEnv == nil {
		return
	}

	// Copy all types from the declaration environment to the checker's environment
	for name, typ := range declEnv.GetAll() {
		c.env.Set(name, typ)

		// Also register in appropriate internal maps based on type
		switch t := typ.(type) {
		case *InterfaceType:
			c.interfaces[name] = t
		case *ClassType:
			c.classes[name] = t
		case *EnumType:
			c.enums[name] = t
		}
	}
}

// predeclareTypeNames publishes an empty interface, class or enum type for
// every such declaration that is not yet known. Member signatures resolved
// later point at these objects, and the register functions fill the same
// objects in place, so `update: function(): DigestContext` inside
// DigestContext, or a reference to an interface declared further down,
// resolves to the finished type.
func (c *Checker) predeclareTypeNames(statements []ast.Statement) {
	if c.shells == nil {
		c.shells = make(map[string]Type)
	}
	for _, stmt := range statements {
		switch node := stmt.(type) {
		case *ast.InterfaceDeclaration:
			name := node.Name.Value
			if len(node.GenericParams) > 0 {
				c.declareGenericInterface(node)
				continue
			}
			if _, known := c.interfaces[name]; !known {
				shell := &InterfaceType{Name: name}
				c.interfaces[name] = shell
				c.env.Set(name, shell)
				c.shells[name] = shell
			}
		case *ast.ClassDeclaration:
			name := node.Name.Value
			if _, known := c.classes[name]; !known {
				shell := &ClassType{Name: name}
				c.classes[name] = shell
				c.env.Set(name, shell)
				c.shells[name] = shell
			}
		case *ast.EnumDeclaration:
			name := node.Name.Value
			if _, known := c.enums[name]; !known {
				shell := &EnumType{Name: name}
				c.enums[name] = shell
				c.env.Set(name, shell)
				c.shells[name] = shell
			}
		case *ast.DeclareStatement:
			if node.Declaration != nil {
				c.predeclareTypeNames([]ast.Statement{node.Declaration})
			}
		case *ast.ExportStatement:
			c.predeclareTypeNames([]ast.Statement{node.Statement})
		case *ast.NamespaceDeclaration:
			c.predeclareTypeNames(node.Statements)
		}
	}
}

// fillInterfaceShell returns the predeclared shell for t's name, now holding
// t's contents, or t itself when no shell is waiting.
func (c *Checker) fillInterfaceShell(t *InterfaceType) *InterfaceType {
	if shell, ok := c.shells[t.Name].(*InterfaceType); ok {
		delete(c.shells, t.Name)
		*shell = *t
		return shell
	}
	return t
}

func (c *Checker) fillClassShell(t *ClassType) *ClassType {
	if shell, ok := c.shells[t.Name].(*ClassType); ok {
		delete(c.shells, t.Name)
		*shell = *t
		return shell
	}
	return t
}

func (c *Checker) fillEnumShell(t *EnumType) *EnumType {
	if shell, ok := c.shells[t.Name].(*EnumType); ok {
		delete(c.shells, t.Name)
		*shell = *t
		return shell
	}
	return t
}

// didYouMean returns " Did you mean 'x'?" for the closest candidate, or "".
func didYouMean(name string, candidates []string) string {
	if similar := findSimilarNames(name, candidates, suggestionDistance(name), 1); len(similar) > 0 {
		return fmt.Sprintf(" Did you mean '%s'?", similar[0])
	}
	return ""
}

// registerTypeDefinition registers classes, interfaces, enums, and type aliases
func (c *Checker) registerTypeDefinition(stmt ast.Statement) {
	switch node := stmt.(type) {
	case *ast.ClassDeclaration:
		c.registerClass(node)
	case *ast.InterfaceDeclaration:
		c.registerInterface(node)
	case *ast.EnumDeclaration:
		c.registerEnum(node)
	case *ast.TypeDeclaration:
		c.registerTypeAlias(node)
	case *ast.NamespaceDeclaration:
		c.registerNamespace(node)
	case *ast.DeclareStatement:
		// Ambient declarations - register the underlying declaration
		if ns, ok := node.Declaration.(*ast.NamespaceDeclaration); ok {
			nsType := c.registerAmbientNamespace(ns)
			c.env.Set(ns.Name.Value, nsType)
			c.markAmbient(ns.Name.Value, nsType)
			c.define(ns.Name.Token, SymNamespace, nsType)
		} else if node.Declaration != nil {
			c.registerTypeDefinition(node.Declaration)
		}
	case *ast.ExportStatement:
		// Handle exported type definitions
		c.registerTypeDefinition(node.Statement)
	}
}

// registerClass registers a class type
func (c *Checker) registerClass(node *ast.ClassDeclaration) {
	// Extract generic parameter names
	genericParams := make([]string, len(node.GenericParams))
	for i, param := range node.GenericParams {
		genericParams[i] = param.Value
	}

	classType := &ClassType{
		Name:               node.Name.Value,
		Properties:         make(map[string]Type),
		Methods:            make(map[string]*FunctionType),
		StaticProperties:   make(map[string]Type),
		StaticMethods:      make(map[string]*FunctionType),
		ReadonlyProps:      make(map[string]bool),
		AbstractMethods:    make(map[string]bool),
		PropertyVisibility: make(map[string]string),
		MethodVisibility:   make(map[string]string),
		Implements:         []*InterfaceType{},
		IsAbstract:         node.IsAbstract,
		GenericParams:      genericParams,
		Getters:            make(map[string]*FunctionType),
		Setters:            make(map[string]*FunctionType),
		GetterVisibility:   make(map[string]string),
		SetterVisibility:   make(map[string]string),
	}

	// Publish the (still empty) class before resolving its members so that a
	// member signature can name the class itself, as in
	// `static open(owner: string): Account`. Members fill in the same pointer.
	classType.Ambient = node.IsAmbient
	classType = c.fillClassShell(classType)
	c.classes[classType.Name] = classType
	c.env.Set(classType.Name, classType)
	c.typeSyms[classType.Name] = c.define(node.Name.Token, SymClass, classType)

	// ClassName.new is the constructor Lunar generates, so a Lunar class
	// cannot define its own. A declared class describes foreign Lua and may.
	if !node.IsAmbient {
		reserved := func(tok lexer.Token) {
			c.addError(fmt.Sprintf("'new' is reserved for the constructor of class '%s'", node.Name.Value), tok)
		}
		for _, m := range node.Methods {
			if m != nil && m.Name != nil && m.Name.Value == "new" {
				reserved(m.Name.Token)
			}
		}
		for _, prop := range node.Properties {
			if prop != nil && prop.Name != nil && prop.Name.Value == "new" {
				reserved(prop.Name.Token)
			}
		}
	}

	// Add generic type parameters to scope temporarily
	prevEnv := c.env
	if len(node.GenericParams) > 0 {
		c.env = NewEnclosedEnvironment(prevEnv)
		for _, genericParam := range node.GenericParams {
			// Store as GenericParamType so we can substitute it later
			c.env.Set(genericParam.Value, &GenericParamType{Name: genericParam.Value})
		}
	}

	// Register properties
	for _, prop := range node.Properties {
		propType := c.resolveTypeExpression(prop.Type)
		if prop.IsOptional {
			propType = withNil(propType)
		}
		propName := prop.Name.Value

		// Track static vs instance properties
		if prop.IsStatic {
			classType.StaticProperties[propName] = propType
		} else {
			classType.Properties[propName] = propType
		}
		setMemberSym(&classType.MemberSyms, propName, c.defineMember(prop.Name.Token, SymProperty, propType, classType.Name))

		// Track readonly properties
		if prop.IsReadonly {
			classType.ReadonlyProps[propName] = true
		}

		// Track property visibility (default to public if not specified)
		visibility := prop.Visibility
		if visibility == "" {
			visibility = "public"
		}
		classType.PropertyVisibility[propName] = visibility
	}

	// Register constructor
	if node.Constructor != nil {
		params := make([]Type, len(node.Constructor.Parameters))
		ctorOptional := make([]bool, len(node.Constructor.Parameters))
		ctorHasRest := false
		for i, param := range node.Constructor.Parameters {
			if param.Type != nil {
				params[i] = c.paramType(param)
			} else {
				params[i] = Any
			}

			if param.IsRest {
				ctorHasRest = true
				if _, ok := params[i].(*ArrayType); !ok {
					params[i] = &ArrayType{ElementType: params[i]}
				}
			}
			ctorOptional[i] = paramIsOptional(param, params[i])

			// Handle constructor parameter properties
			if param.Visibility != "" || param.IsReadonly {
				propName := param.Name.Value
				propType := params[i]

				// Add as class property, defined where the parameter is
				classType.Properties[propName] = propType
				setMemberSym(&classType.MemberSyms, propName, c.defineMember(param.Name.Token, SymProperty, propType, classType.Name))

				// Track readonly
				if param.IsReadonly {
					classType.ReadonlyProps[propName] = true
				}

				// Track visibility
				classType.PropertyVisibility[propName] = param.Visibility
			}
		}
		classType.Constructor = &FunctionType{
			Parameters:       params,
			ReturnType:       classType,  // Constructor returns an instance of the class
			GenericParams:    []string{}, // Constructors are not separately generic
			OptionalParams:   ctorOptional,
			HasRestParameter: ctorHasRest,
			ParamNames:       parameterNames(node.Constructor.Parameters),
		}
		// Class.new(...) is the constructor: it goes to `constructor`.
		if classType.MemberSyms == nil || classType.MemberSyms["new"] == nil {
			setMemberSym(&classType.MemberSyms, "new", &Symbol{Name: "new", Kind: SymMethod, Type: classType.Constructor,
				Decl: node.Constructor.Token, Container: classType.Name, ScopeEnd: -1})
		}
	}

	// Register methods
	for _, method := range node.Methods {
		// A generic method's type parameters are in scope for its signature.
		prevSigEnv := c.env
		methodGenerics := make([]string, len(method.GenericParams))
		if len(method.GenericParams) > 0 {
			c.env = NewEnclosedEnvironment(prevSigEnv)
			for i, gp := range method.GenericParams {
				c.env.Set(gp.Value, &GenericParamType{Name: gp.Value})
				methodGenerics[i] = gp.Value
			}
		}
		params := make([]Type, len(method.Parameters))
		optionalParams := make([]bool, len(method.Parameters))
		hasRestParam := false
		for i, param := range method.Parameters {
			params[i] = c.paramType(param)
			if param.IsRest {
				hasRestParam = true
				if _, ok := params[i].(*ArrayType); !ok {
					params[i] = &ArrayType{ElementType: params[i]}
				}
			}
			optionalParams[i] = paramIsOptional(param, params[i])
		}
		var returnType Type = Void
		if method.ReturnType != nil {
			returnType = c.resolveTypeExpression(method.ReturnType)
		}
		c.env = prevSigEnv
		funcType := &FunctionType{
			Parameters:       params,
			ReturnType:       returnType,
			GenericParams:    methodGenerics,
			OptionalParams:   optionalParams,
			HasRestParameter: hasRestParam,
		}
		methodName := method.Name.Value

		// Track static vs instance methods
		if method.IsStatic {
			classType.StaticMethods[methodName] = funcType
		} else {
			classType.Methods[methodName] = funcType
		}
		funcType.ParamNames = parameterNames(method.Parameters)
		setMemberSym(&classType.MemberSyms, methodName, c.defineMember(method.Name.Token, SymMethod, funcType, classType.Name))

		// Track abstract methods
		if method.IsAbstract {
			classType.AbstractMethods[methodName] = true
		}

		// Track method visibility (default to public if not specified)
		visibility := method.Visibility
		if visibility == "" {
			visibility = "public"
		}
		classType.MethodVisibility[methodName] = visibility
	}

	// Register getters
	for _, getter := range node.Getters {
		var returnType Type = Any
		if getter.ReturnType != nil {
			returnType = c.resolveTypeExpression(getter.ReturnType)
		}
		funcType := &FunctionType{
			Parameters:    []Type{}, // Getters have no parameters
			ReturnType:    returnType,
			GenericParams: []string{},
		}
		getterName := getter.Name.Value
		classType.Getters[getterName] = funcType
		setMemberSym(&classType.MemberSyms, getterName, c.defineMember(getter.Name.Token, SymProperty, funcType.ReturnType, classType.Name))

		// Track getter visibility (default to public if not specified)
		visibility := getter.Visibility
		if visibility == "" {
			visibility = "public"
		}
		classType.GetterVisibility[getterName] = visibility
	}

	// Register setters
	for _, setter := range node.Setters {
		var paramType Type = Any
		if setter.Parameter != nil && setter.Parameter.Type != nil {
			paramType = c.resolveTypeExpression(setter.Parameter.Type)
		}
		funcType := &FunctionType{
			Parameters:    []Type{paramType},
			ReturnType:    Void, // Setters return void
			GenericParams: []string{},
		}
		setterName := setter.Name.Value
		classType.Setters[setterName] = funcType
		if classType.MemberSyms[setterName] == nil {
			setMemberSym(&classType.MemberSyms, setterName, c.defineMember(setter.Name.Token, SymProperty, Any, classType.Name))
		}

		// Track setter visibility (default to public if not specified)
		visibility := setter.Visibility
		if visibility == "" {
			visibility = "public"
		}
		classType.SetterVisibility[setterName] = visibility
	}

	// Register index signature if present
	if node.IndexSignature != nil {
		keyType := c.resolveTypeExpression(node.IndexSignature.KeyType)
		valueType := c.resolveTypeExpression(node.IndexSignature.ValueType)
		classType.IndexSignature = &IndexSignature{
			KeyType:   keyType,
			ValueType: valueType,
		}
	}

	// Resolve extends clause (parent class)
	if node.Extends != nil {
		if ident, ok := node.Extends.(*ast.Identifier); ok {
			if parentClass, exists := c.classes[ident.Value]; exists {
				classType.Parent = parentClass
			} else {
				c.addError(fmt.Sprintf("Parent class '%s' not found", ident.Value), ident.Token)
			}
		} else if parentClass, ok := c.resolveTypeExpression(node.Extends).(*ClassType); ok {
			// extends Geo.Base
			classType.Parent = parentClass
		}
	}

	// Resolve implements clause
	for _, impl := range node.Implements {
		if ident, ok := impl.(*ast.Identifier); ok {
			if interfaceType, exists := c.interfaces[ident.Value]; exists {
				classType.Implements = append(classType.Implements, interfaceType)
			} else {
				c.addError(fmt.Sprintf("Interface '%s' not found", ident.Value), ident.Token)
			}
		}
	}

	// Restore environment
	if len(node.GenericParams) > 0 {
		c.env = prevEnv
	}
}

// registerInterface registers an interface type
func (c *Checker) registerInterface(node *ast.InterfaceDeclaration) {
	if len(node.GenericParams) > 0 {
		c.declareGenericInterface(node)
		return
	}
	interfaceType := c.buildInterface(node, node.Name.Value, nil)
	// A second declaration of an interface adds to the first, so a later
	// declaration file can extend one (Lua 5.4's coroutine.close).
	if existing, ok := c.interfaces[node.Name.Value]; ok && existing.Properties != nil && c.shells[node.Name.Value] == nil {
		mergeInterface(existing, interfaceType)
		return
	}
	interfaceType = c.fillInterfaceShell(interfaceType)
	c.interfaces[interfaceType.Name] = interfaceType
	c.env.Set(interfaceType.Name, interfaceType)
	c.typeSyms[interfaceType.Name] = c.define(node.Name.Token, SymInterface, interfaceType)
}

// mergeInterface adds more's members to into.
func mergeInterface(into, more *InterfaceType) {
	for name, t := range more.Properties {
		into.Properties[name] = t
	}
	if into.Methods == nil {
		into.Methods = map[string]*FunctionType{}
	}
	for name, m := range more.Methods {
		into.Methods[name] = m
	}
	for _, flags := range []struct {
		dst *map[string]bool
		src map[string]bool
	}{
		{&into.OptionalProps, more.OptionalProps}, {&into.StaticProps, more.StaticProps}, {&into.StaticMethods, more.StaticMethods},
	} {
		for name, v := range flags.src {
			if *flags.dst == nil {
				*flags.dst = map[string]bool{}
			}
			(*flags.dst)[name] = v
		}
	}
	for name, sym := range more.MemberSyms {
		setMemberSym(&into.MemberSyms, name, sym)
	}
	into.Extends = append(into.Extends, more.Extends...)
	if into.IndexSignature == nil {
		into.IndexSignature = more.IndexSignature
	}
}

// memberSym defines a member's symbol, or reuses the shared one while an
// instance of a generic interface is built.
func (c *Checker) memberSym(tok lexer.Token, kind SymbolKind, typ Type, container string) *Symbol {
	if c.sharedMemberSyms != nil {
		if sym := c.sharedMemberSyms[tok.Literal]; sym != nil {
			return sym
		}
	}
	return c.defineMember(tok, kind, typ, container)
}

// declareGenericInterface records interface Box<T> as a template; each
// Box<X> written later is built from it on first use.
func (c *Checker) declareGenericInterface(node *ast.InterfaceDeclaration) {
	name := node.Name.Value
	if _, done := c.genericInterfaces[name]; done {
		return
	}
	params := make([]string, len(node.GenericParams))
	for i, p := range node.GenericParams {
		params[i] = p.Value
	}
	tmpl := &GenericInterfaceType{Name: name, Params: params}
	c.genericInterfaces[name] = tmpl
	c.env.Set(name, tmpl)
	c.typeSyms[name] = c.define(node.Name.Token, SymInterface, tmpl)

	instances := map[string]*InterfaceType{}
	var memberSyms map[string]*Symbol
	build := func(args []Type) *InterfaceType {
		strs := make([]string, len(args))
		for i, a := range args {
			strs[i] = a.String()
		}
		key := name + "<" + strings.Join(strs, ", ") + ">"
		if inst, ok := instances[key]; ok {
			return inst
		}
		// Registered before it is filled, so Node<T> may refer to itself.
		inst := &InterfaceType{Name: key}
		instances[key] = inst
		prevEnv := c.env
		c.env = NewEnclosedEnvironment(prevEnv)
		for i, p := range params {
			c.env.Set(p, args[i])
		}
		*inst = *c.buildInterface(node, key, memberSyms)
		c.env = prevEnv
		return inst
	}
	// The template's own members, with the parameters unbound, own the
	// symbols every instance shares, so references and rename see them all.
	generic := make([]Type, len(params))
	for i, p := range params {
		generic[i] = &GenericParamType{Name: p}
	}
	tmpl.Instantiate = build // before building: Node<T> may name itself
	memberSyms = build(generic).MemberSyms
}

// buildInterface makes the interface type a declaration describes, named
// name. With syms, members take those symbols instead of defining new ones.
func (c *Checker) buildInterface(node *ast.InterfaceDeclaration, name string, syms map[string]*Symbol) *InterfaceType {
	prevSyms := c.sharedMemberSyms
	c.sharedMemberSyms = syms
	defer func() { c.sharedMemberSyms = prevSyms }()

	interfaceType := &InterfaceType{
		Name:          name,
		Methods:       make(map[string]*FunctionType),
		Properties:    make(map[string]Type),
		Extends:       []*InterfaceType{},
		StaticProps:   make(map[string]bool),
		StaticMethods: make(map[string]bool),
	}

	// Register properties
	for _, prop := range node.Properties {
		c.addInterfaceProperty(interfaceType, prop)
	}

	// Register methods
	for _, method := range node.Methods {
		prevSigEnv := c.env
		methodGenerics := make([]string, len(method.GenericParams))
		if len(method.GenericParams) > 0 {
			c.env = NewEnclosedEnvironment(prevSigEnv)
			for i, gp := range method.GenericParams {
				c.env.Set(gp.Value, &GenericParamType{Name: gp.Value})
				methodGenerics[i] = gp.Value
			}
		}
		// Optional and rest parameters count as they do for functions:
		// seek(whence?: string, offset?: number) may be called with none.
		params := make([]Type, len(method.Parameters))
		optional := make([]bool, len(method.Parameters))
		hasRest := false
		for i, param := range method.Parameters {
			params[i] = c.paramType(param)
			if param.IsRest {
				hasRest = true
				if _, ok := params[i].(*ArrayType); !ok {
					params[i] = &ArrayType{ElementType: params[i]}
				}
			}
			optional[i] = paramIsOptional(param, params[i])
		}
		var returnType Type = Void
		if method.ReturnType != nil {
			returnType = c.resolveTypeExpression(method.ReturnType)
		}
		c.env = prevSigEnv
		interfaceType.Methods[method.Name.Value] = &FunctionType{
			Parameters:       params,
			ReturnType:       returnType,
			GenericParams:    methodGenerics,
			ParamNames:       parameterNames(method.Parameters),
			OptionalParams:   optional,
			HasRestParameter: hasRest,
		}
		setMemberSym(&interfaceType.MemberSyms, method.Name.Value,
			c.memberSym(method.Name.Token, SymMethod, interfaceType.Methods[method.Name.Value], interfaceType.Name))
		if method.IsStatic {
			interfaceType.StaticMethods[method.Name.Value] = true
		}
	}

	// Register index signature if present
	if node.IndexSignature != nil {
		keyType := c.resolveTypeExpression(node.IndexSignature.KeyType)
		valueType := c.resolveTypeExpression(node.IndexSignature.ValueType)
		interfaceType.IndexSignature = &IndexSignature{
			KeyType:   keyType,
			ValueType: valueType,
		}
	}

	// Resolve extends clause
	for _, ext := range node.Extends {
		if ident, ok := ext.(*ast.Identifier); ok {
			if extInterface, exists := c.interfaces[ident.Value]; exists {
				interfaceType.Extends = append(interfaceType.Extends, extInterface)
			} else {
				c.addError(fmt.Sprintf("Interface '%s' not found", ident.Value), ident.Token)
			}
		}
	}

	return interfaceType
}

// registerEnum registers an enum type
func (c *Checker) registerEnum(node *ast.EnumDeclaration) {
	enumType := &EnumType{
		Name:    node.Name.Value,
		Members: make(map[string]Type),
	}

	// First, register the enum type itself so members can reference it
	enumType = c.fillEnumShell(enumType)
	c.enums[enumType.Name] = enumType
	c.env.Set(enumType.Name, enumType)
	c.typeSyms[enumType.Name] = c.define(node.Name.Token, SymEnum, enumType)

	if _, blocked, previous := node.ImplicitValues(); blocked != nil {
		c.addError(fmt.Sprintf("Enum member '%s' needs an explicit value because '%s' is not a number literal",
			blocked.Name.Value, previous.Name.Value), blocked.Token)
	}

	for _, member := range node.Members {
		if member.Value != nil {
			// Validate the value expression (should be number or string)
			if IsStringType(c.checkExpression(member.Value)) {
				enumType.Strings = true
			}
		}
		// All enum members have the enum type itself, not the value type
		// This ensures type safety: Color.Red has type Color, not number
		enumType.Members[member.Name.Value] = enumType
		setMemberSym(&enumType.MemberSyms, member.Name.Value, c.defineMember(member.Name.Token, SymEnumMember, enumType, enumType.Name))
	}
}

// addInterfaceProperty records one property of an interface or object
// shape. `name?: T` is stored as T | nil, the same as `name: T?`, so reading
// it gives an optional either way; both spellings may be left out.
func (c *Checker) addInterfaceProperty(iface *InterfaceType, prop *ast.PropertyDeclaration) {
	propType := c.resolveTypeExpression(prop.Type)
	if prop.IsOptional {
		propType = withNil(propType)
	}
	name := prop.Name.Value
	iface.Properties[name] = propType
	if prop.IsReadonly {
		if iface.ReadonlyProps == nil {
			iface.ReadonlyProps = map[string]bool{}
		}
		iface.ReadonlyProps[name] = true
	}
	setMemberSym(&iface.MemberSyms, name, c.memberSym(prop.Name.Token, SymProperty, propType, iface.Name))
	if prop.IsStatic {
		if iface.StaticProps == nil {
			iface.StaticProps = make(map[string]bool)
		}
		iface.StaticProps[name] = true
	}
	if _, admitsNil := removeNil(propType); admitsNil {
		if iface.OptionalProps == nil {
			iface.OptionalProps = make(map[string]bool)
		}
		iface.OptionalProps[name] = true
	}
}

// withNil returns t | nil, or t itself when it already admits nil.
func withNil(t Type) Type {
	if _, admitsNil := removeNil(t); admitsNil {
		return t
	}
	// any and unknown already include nil.
	switch t.(type) {
	case *AnyType, *UnknownType, *NilType:
		return t
	}
	return &UnionType{Types: []Type{t, Nil}}
}

// registerTypeAlias registers a type alias
func (c *Checker) registerTypeAlias(node *ast.TypeDeclaration) {
	// type T = typeof u names a local that does not exist yet when types
	// are registered: T is any until the declaration is reached, then
	// resolved there (checkStatement), with the locals in scope.
	if !c.resolvingTypeof && node.Type != nil && containsTypeof(reflect.ValueOf(node.Type)) {
		if node.Name != nil {
			c.typeAliases[node.Name.Value] = Any
		}
		return
	}
	// Check if this is a generic type alias
	if len(node.GenericParams) > 0 {
		// Generic type alias: type Name<T, U> = Type
		typeParams := make([]string, len(node.GenericParams))
		for i, param := range node.GenericParams {
			typeParams[i] = param.Value
		}

		genericAlias := &GenericTypeAlias{
			Name:       node.Name.Value,
			TypeParams: typeParams,
			Body:       node.Type,
		}

		c.genericTypeAliases[node.Name.Value] = genericAlias
		c.env.Set(node.Name.Value, genericAlias)
		return
	}

	var aliasType Type

	if node.Type != nil {
		// Regular type alias: type Name = Type
		aliasType = c.resolveTypeExpression(node.Type)
	} else if len(node.Properties) > 0 {
		// Object shape: type Name ... end
		interfaceType := &InterfaceType{
			Name:        node.Name.Value,
			Properties:  make(map[string]Type),
			Methods:     make(map[string]*FunctionType),
			Extends:     []*InterfaceType{},
			StaticProps: make(map[string]bool),
		}

		// Register properties
		for _, prop := range node.Properties {
			c.addInterfaceProperty(interfaceType, prop)
		}

		aliasType = interfaceType
	} else {
		aliasType = Any
	}

	c.typeAliases[node.Name.Value] = aliasType
	c.env.Set(node.Name.Value, aliasType)
	c.typeSyms[node.Name.Value] = c.define(node.Name.Token, SymTypeAlias, aliasType)
}

// resolveTypeExpression resolves a type expression to a Type
func (c *Checker) resolveTypeExpression(expr ast.Expression) Type {
	if expr == nil {
		return Any
	}

	switch node := expr.(type) {
	case *ast.Identifier:
		// Check for primitive types FIRST (before checking environment)
		// This ensures primitive types take precedence over variables with the same name
		switch node.Value {
		case "string":
			return String
		case "number":
			return Number
		case "boolean":
			return Boolean
		case "nil":
			return Nil
		case "any":
			return Any
		case "void":
			return Void
		case "never":
			return Never
		case "unknown":
			return Unknown
		}

		// Check for user-defined types
		c.reference(node.Token, c.typeSyms[node.Value])
		if classType, ok := c.classes[node.Value]; ok {
			return classType
		}
		if interfaceType, ok := c.interfaces[node.Value]; ok {
			return interfaceType
		}
		if enumType, ok := c.enums[node.Value]; ok {
			return enumType
		}
		if aliasType, ok := c.typeAliases[node.Value]; ok {
			return aliasType
		}

		// Check environment last (for other types registered in environment)
		if typ, ok := c.env.Get(node.Value); ok {
			if tmpl, isGeneric := typ.(*GenericInterfaceType); isGeneric {
				c.addError(fmt.Sprintf("Generic type '%s' needs type arguments: %s<...>", tmpl.Name, tmpl.Name), node.Token)
				return Any
			}
			return typ
		}

		c.addError(fmt.Sprintf("Unknown type '%s'", node.Value), node.Token)
		return Any

	case *ast.DotExpression:
		// Geo.Pos: a type a namespace or a module (import * as m) holds.
		owner := c.qualifiedOwner(node.Left)
		name, _ := node.Right.(*ast.Identifier)
		if owner != nil && name != nil {
			if member, ok := owner.Members[name.Value]; ok {
				return member
			}
			c.addError(fmt.Sprintf("'%s' has no type '%s'", owner.Name, name.Value), name.Token)
			return Any
		}
		c.addError(fmt.Sprintf("'%s' is not a namespace or module", typeExprText(node.Left)), node.Token)
		return Any

	case *ast.ArrayType:
		elementType := c.resolveTypeExpression(node.ElementType)
		return &ArrayType{ElementType: elementType}

	case *ast.TableType:
		keyType := c.resolveTypeExpression(node.KeyType)
		valueType := c.resolveTypeExpression(node.ValueType)
		return &TableType{KeyType: keyType, ValueType: valueType}

	case *ast.UnionType:
		types := make([]Type, 0, len(node.Types))
		for _, t := range node.Types {
			resolvedType := c.resolveTypeExpression(t)
			// Flatten nested unions
			if unionType, isUnion := resolvedType.(*UnionType); isUnion {
				types = append(types, unionType.Types...)
			} else {
				types = append(types, resolvedType)
			}
		}
		return &UnionType{Types: types}

	case *ast.IntersectionType:
		types := make([]Type, 0, len(node.Types))
		for _, t := range node.Types {
			resolvedType := c.resolveTypeExpression(t)
			// Flatten nested intersections
			if intersectionType, isIntersection := resolvedType.(*IntersectionType); isIntersection {
				types = append(types, intersectionType.Types...)
			} else {
				types = append(types, resolvedType)
			}
		}
		return &IntersectionType{Types: types}

	case *ast.TupleType:
		elements := make([]Type, len(node.Types))
		for i, elem := range node.Types {
			elements[i] = c.resolveTypeExpression(elem)
		}
		return &TupleType{Elements: elements}

	case *ast.FunctionType:
		// A bare `function` type constrains nothing beyond being callable.
		if node.Parameters == nil && node.ReturnType == nil {
			return Any
		}

		params := make([]Type, len(node.Parameters))
		optionalParams := make([]bool, len(node.Parameters))
		hasRestParam := false
		for i, param := range node.Parameters {
			paramType := c.paramType(param)
			if param.IsRest {
				hasRestParam = true
				if _, ok := paramType.(*ArrayType); !ok {
					paramType = &ArrayType{ElementType: paramType}
				}
			}
			optionalParams[i] = paramIsOptional(param, paramType)
			params[i] = paramType
		}
		var returnType Type = Void
		if node.ReturnType != nil {
			returnType = c.resolveTypeExpression(node.ReturnType)
		}
		return &FunctionType{
			Parameters:       params,
			ReturnType:       returnType,
			GenericParams:    []string{}, // Function types in annotations are not generic
			OptionalParams:   optionalParams,
			HasRestParameter: hasRestParam,
		}

	case *ast.GenericType:
		// A generic interface, declared here or imported: Box<number>.
		if baseIdent, ok := node.BaseType.(*ast.Identifier); ok {
			if typ, found := c.env.Get(baseIdent.Value); found {
				if tmpl, isGeneric := typ.(*GenericInterfaceType); isGeneric {
					c.reference(baseIdent.Token, c.typeSyms[baseIdent.Value])
					args := make([]Type, len(node.TypeArguments))
					for i, arg := range node.TypeArguments {
						args[i] = c.resolveTypeExpression(arg)
					}
					if len(args) != len(tmpl.Params) {
						c.addError(fmt.Sprintf("Generic type '%s' expects %d type argument(s), got %d", tmpl.Name, len(tmpl.Params), len(args)), baseIdent.Token)
						return Any
					}
					return tmpl.Instantiate(args)
				}
			}
		}
		// Check if this is a generic type alias instantiation like Nullable<string>
		if baseIdent, ok := node.BaseType.(*ast.Identifier); ok {
			if genericAlias, exists := c.genericTypeAliases[baseIdent.Value]; exists {
				// Resolve type arguments
				typeArgs := make([]Type, len(node.TypeArguments))
				for i, arg := range node.TypeArguments {
					typeArgs[i] = c.resolveTypeExpression(arg)
				}

				// Check parameter count matches
				if len(typeArgs) != len(genericAlias.TypeParams) {
					c.addError(
						fmt.Sprintf("Generic type '%s' expects %d type arguments, got %d",
							genericAlias.Name, len(genericAlias.TypeParams), len(typeArgs)),
						baseIdent.Token,
					)
					return Any
				}

				// Create substitution map and resolve the body
				return c.substituteTypeParams(genericAlias.Body, genericAlias.TypeParams, typeArgs)
			}
		}

		// Check for built-in utility types
		if baseIdent, ok := node.BaseType.(*ast.Identifier); ok {
			// Resolve type arguments
			typeArgs := make([]Type, len(node.TypeArguments))
			for i, arg := range node.TypeArguments {
				typeArgs[i] = c.resolveTypeExpression(arg)
			}

			// Handle utility types
			switch baseIdent.Value {
			case "Exclude":
				if len(typeArgs) != 2 {
					c.addError("Exclude<T, U> expects 2 type arguments", baseIdent.Token)
					return Any
				}
				return c.excludeTypes(typeArgs[0], typeArgs[1])

			case "Extract":
				if len(typeArgs) != 2 {
					c.addError("Extract<T, U> expects 2 type arguments", baseIdent.Token)
					return Any
				}
				return c.extractTypes(typeArgs[0], typeArgs[1])

			case "NonNullable":
				if len(typeArgs) != 1 {
					c.addError("NonNullable<T> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.nonNullable(typeArgs[0])

			case "Pick":
				if len(typeArgs) != 2 {
					c.addError("Pick<T, K> expects 2 type arguments", baseIdent.Token)
					return Any
				}
				return c.pickType(typeArgs[0], typeArgs[1])

			case "Omit":
				if len(typeArgs) != 2 {
					c.addError("Omit<T, K> expects 2 type arguments", baseIdent.Token)
					return Any
				}
				return c.omitType(typeArgs[0], typeArgs[1])

			case "Record":
				if len(typeArgs) != 2 {
					c.addError("Record<K, V> expects 2 type arguments", baseIdent.Token)
					return Any
				}
				return c.recordType(typeArgs[0], typeArgs[1])

			case "Partial":
				if len(typeArgs) != 1 {
					c.addError("Partial<T> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.partialType(typeArgs[0])

			case "Required":
				if len(typeArgs) != 1 {
					c.addError("Required<T> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.requiredType(typeArgs[0])

			case "Readonly":
				if len(typeArgs) != 1 {
					c.addError("Readonly<T> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.readonlyType(typeArgs[0])

			case "ReturnType":
				if len(typeArgs) != 1 {
					c.addError("ReturnType<F> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.returnType(typeArgs[0])

			case "Parameters":
				if len(typeArgs) != 1 {
					c.addError("Parameters<F> expects 1 type argument", baseIdent.Token)
					return Any
				}
				return c.parametersType(typeArgs[0])
			}

			// Check if it's a generic class
			if classType, exists := c.classes[baseIdent.Value]; exists {
				if len(classType.GenericParams) > 0 {
					if len(typeArgs) != len(classType.GenericParams) {
						c.addError(
							fmt.Sprintf("Class '%s' expects %d type arguments, got %d",
								classType.Name, len(classType.GenericParams), len(typeArgs)),
							baseIdent.Token,
						)
						return classType
					}
					return c.instantiateGenericClass(classType, typeArgs)
				}
			}
		}

		// Fall back to regular type resolution (ignoring type arguments)
		baseType := c.resolveTypeExpression(node.BaseType)
		return baseType

	case *ast.StringLiteral:
		// String literal in type position becomes a literal type
		return &StringLiteralType{Value: node.Value}

	case *ast.NumberLiteral:
		// Number literal in type position becomes a literal type
		return &NumberLiteralType{Value: node.Value}

	case *ast.BooleanLiteral:
		// Boolean literal in type position becomes a literal type
		return &BooleanLiteralType{Value: node.Value}

	case *ast.TypeGuardType:
		// Type guard like "value is string"
		// Returns a special TypeGuardType that acts like boolean but carries guard info
		guardedType := c.resolveTypeExpression(node.GuardType)
		return &TypeGuardType{GuardedType: guardedType}

	case *ast.OptionalType:
		// Optional type T? is syntactic sugar for T | nil
		baseType := c.resolveTypeExpression(node.Type)
		return &UnionType{Types: []Type{baseType, Nil}}

	case *ast.KeyofType:
		// keyof T returns a union of string literal types of T's keys
		objectType := c.resolveTypeExpression(node.ObjectType)
		return c.resolveKeyof(objectType)

	case *ast.TypeofExpression:
		// typeof value returns the type of the value
		return c.resolveTypeof(node.Expression)

	case *ast.ConditionalType:
		// T extends U ? X : Y
		checkType := c.resolveTypeExpression(node.CheckType)
		extendsType := c.resolveTypeExpression(node.ExtendsType)
		trueType := c.resolveTypeExpression(node.TrueType)
		falseType := c.resolveTypeExpression(node.FalseType)
		return c.resolveConditionalType(checkType, extendsType, trueType, falseType)

	case *ast.IndexExpression:
		// Indexed access type T[K]
		objectType := c.resolveTypeExpression(node.Left)
		indexType := c.resolveTypeExpression(node.Index)
		return c.resolveIndexedAccess(objectType, indexType)

	case *ast.MappedType:
		// Mapped type { [K in T]: U }
		// Resolve the constraint type (what K iterates over)
		constraintType := c.resolveTypeExpression(node.Constraint)

		// For now, create a generic mapped type that will be instantiated later
		return c.resolveMappedType(node, constraintType)

	case *ast.TemplateLiteralType:
		// Template literal type `Hello ${string}` or `${T}_${U}`
		return c.resolveTemplateLiteralType(node)

	default:
		c.addError(fmt.Sprintf("Cannot resolve type expression: %T", expr), exprToken(expr, lexer.Token{}))
		return Any
	}
}

// resolveIndexedAccess resolves indexed access types like T[K]
func (c *Checker) resolveIndexedAccess(objectType Type, indexType Type) Type {
	switch obj := objectType.(type) {
	case *ClassType:
		// Handle class property access
		return c.resolveClassIndexedAccess(obj, indexType)

	case *InterfaceType:
		// Handle interface property access
		return c.resolveInterfaceIndexedAccess(obj, indexType)

	case *ArrayType:
		// Array[number] or Array[0] returns element type
		return obj.ElementType

	case *TupleType:
		// Tuple[number literal] returns type at that index
		if numLiteral, ok := indexType.(*NumberLiteralType); ok {
			index := int(numLiteral.Value)
			if index >= 0 && index < len(obj.Elements) {
				return obj.Elements[index]
			}
			return Any
		}
		// Tuple[number] returns union of all element types
		if _, ok := indexType.(*NumberType); ok {
			return &UnionType{Types: obj.Elements}
		}
		return Any

	case *TableType:
		// Table<K, V>[K] returns V
		return obj.ValueType

	default:
		// For other types, indexed access returns any
		return Any
	}
}

// resolveClassIndexedAccess resolves T[K] where T is a class type
func (c *Checker) resolveClassIndexedAccess(classType *ClassType, indexType Type) Type {
	// Handle string literal index
	if strLiteral, ok := indexType.(*StringLiteralType); ok {
		// Check properties
		if propType, ok := classType.Properties[strLiteral.Value]; ok {
			return propType
		}
		// Check methods
		if methodType, ok := classType.Methods[strLiteral.Value]; ok {
			return methodType
		}
		return Any
	}

	// Handle union of string literals
	if unionType, ok := indexType.(*UnionType); ok {
		var types []Type
		for _, t := range unionType.Types {
			propType := c.resolveClassIndexedAccess(classType, t)
			if propType != Any {
				types = append(types, propType)
			}
		}
		if len(types) == 0 {
			return Any
		}
		if len(types) == 1 {
			return types[0]
		}
		return &UnionType{Types: types}
	}

	// For string type, return union of all property types
	if _, ok := indexType.(*StringType); ok {
		var types []Type
		for _, name := range memberNames(classType.Properties) {
			types = append(types, classType.Properties[name])
		}
		for _, name := range memberNames(classType.Methods) {
			types = append(types, classType.Methods[name])
		}
		if len(types) == 0 {
			return Never
		}
		if len(types) == 1 {
			return types[0]
		}
		return &UnionType{Types: types}
	}

	return Any
}

// resolveInterfaceIndexedAccess resolves T[K] where T is an interface type
func (c *Checker) resolveInterfaceIndexedAccess(interfaceType *InterfaceType, indexType Type) Type {
	// Handle string literal index
	if strLiteral, ok := indexType.(*StringLiteralType); ok {
		// Check properties
		if propType, ok := interfaceType.Properties[strLiteral.Value]; ok {
			return propType
		}
		// Check methods
		if methodType, ok := interfaceType.Methods[strLiteral.Value]; ok {
			return methodType
		}
		return Any
	}

	// Handle union of string literals
	if unionType, ok := indexType.(*UnionType); ok {
		var types []Type
		for _, t := range unionType.Types {
			propType := c.resolveInterfaceIndexedAccess(interfaceType, t)
			if propType != Any {
				types = append(types, propType)
			}
		}
		if len(types) == 0 {
			return Any
		}
		if len(types) == 1 {
			return types[0]
		}
		return &UnionType{Types: types}
	}

	// For string type, return union of all property types
	if _, ok := indexType.(*StringType); ok {
		var types []Type
		for _, name := range memberNames(interfaceType.Properties) {
			types = append(types, interfaceType.Properties[name])
		}
		for _, name := range memberNames(interfaceType.Methods) {
			types = append(types, interfaceType.Methods[name])
		}
		if len(types) == 0 {
			return Never
		}
		if len(types) == 1 {
			return types[0]
		}
		return &UnionType{Types: types}
	}

	return Any
}

// excludeTypes implements Exclude<T, U> utility type
// Removes all types from T that are assignable to U
func (c *Checker) excludeTypes(t Type, u Type) Type {
	// If T is a union, filter out types assignable to U
	if unionType, ok := t.(*UnionType); ok {
		var remaining []Type
		for _, member := range unionType.Types {
			if !member.IsAssignableTo(u) {
				remaining = append(remaining, member)
			}
		}
		if len(remaining) == 0 {
			return Never
		}
		if len(remaining) == 1 {
			return remaining[0]
		}
		return &UnionType{Types: remaining}
	}

	// If T is not a union, check if it's assignable to U
	if t.IsAssignableTo(u) {
		return Never
	}
	return t
}

// extractTypes implements Extract<T, U> utility type
// Keeps only types from T that are assignable to U
func (c *Checker) extractTypes(t Type, u Type) Type {
	// If T is a union, keep only types assignable to U
	if unionType, ok := t.(*UnionType); ok {
		var extracted []Type
		for _, member := range unionType.Types {
			if member.IsAssignableTo(u) {
				extracted = append(extracted, member)
			}
		}
		if len(extracted) == 0 {
			return Never
		}
		if len(extracted) == 1 {
			return extracted[0]
		}
		return &UnionType{Types: extracted}
	}

	// If T is not a union, check if it's assignable to U
	if t.IsAssignableTo(u) {
		return t
	}
	return Never
}

// nonNullable implements NonNullable<T> utility type
// Removes nil and undefined from T
func (c *Checker) nonNullable(t Type) Type {
	// If T is a union, remove nil
	if unionType, ok := t.(*UnionType); ok {
		var nonNull []Type
		for _, member := range unionType.Types {
			if _, isNil := member.(*NilType); !isNil {
				nonNull = append(nonNull, member)
			}
		}
		if len(nonNull) == 0 {
			return Never
		}
		if len(nonNull) == 1 {
			return nonNull[0]
		}
		return &UnionType{Types: nonNull}
	}

	// If T is nil, return never
	if _, isNil := t.(*NilType); isNil {
		return Never
	}

	// Otherwise return T as is
	return t
}

// pickType implements Pick<T, K> utility type
// Constructs a type by picking a set of properties K from T
func (c *Checker) pickType(t Type, keys Type) Type {
	switch objType := t.(type) {
	case *InterfaceType:
		picked := &InterfaceType{
			Name:        objType.Name + "_Picked",
			Properties:  make(map[string]Type),
			Methods:     make(map[string]*FunctionType),
			StaticProps: make(map[string]bool),
		}

		// Extract keys from the keys type. Fields and methods inherited
		// through extends count, and keep whether they may be left out.
		keyList := c.extractKeyNames(keys)
		props, methods := allProperties(objType), allMethods(objType)
		for _, key := range keyList {
			if propType, ok := props[key]; ok {
				picked.Properties[key] = propType
				if objType.StaticProps != nil && objType.StaticProps[key] {
					picked.StaticProps[key] = true
				}
				if optionalProperty(objType, key) {
					if picked.OptionalProps == nil {
						picked.OptionalProps = map[string]bool{}
					}
					picked.OptionalProps[key] = true
				}
				if objType.ReadonlyProps[key] {
					if picked.ReadonlyProps == nil {
						picked.ReadonlyProps = map[string]bool{}
					}
					picked.ReadonlyProps[key] = true
				}
			}
			if methodType, ok := methods[key]; ok {
				picked.Methods[key] = methodType
			}
		}
		return picked

	case *ClassType:
		picked := &ClassType{
			Name:       objType.Name + "_Picked",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		keyList := c.extractKeyNames(keys)
		for _, key := range keyList {
			if propType, ok := objType.Properties[key]; ok {
				picked.Properties[key] = propType
			}
			if methodType, ok := objType.Methods[key]; ok {
				picked.Methods[key] = methodType
			}
		}
		return picked

	default:
		return Any
	}
}

// omitType implements Omit<T, K> utility type
// Constructs a type by omitting a set of properties K from T
func (c *Checker) omitType(t Type, keys Type) Type {
	// Omit<T, K> = Pick<T, Exclude<keyof T, K>>
	allKeys := c.resolveKeyof(t)
	remainingKeys := c.excludeTypes(allKeys, keys)
	return c.pickType(t, remainingKeys)
}

// allProperties are an interface's properties, those of the interfaces
// it extends included.
func allMethods(t *InterfaceType) map[string]*FunctionType {
	methods := map[string]*FunctionType{}
	for _, parent := range t.Extends {
		if parent == nil {
			continue
		}
		for name, m := range allMethods(parent) {
			methods[name] = m
		}
	}
	for name, m := range t.Methods {
		methods[name] = m
	}
	return methods
}

func allProperties(t *InterfaceType) map[string]Type {
	props := map[string]Type{}
	for _, parent := range t.Extends {
		for name, p := range allProperties(parent) {
			props[name] = p
		}
	}
	for name, p := range t.Properties {
		props[name] = p
	}
	return props
}

// optionalProperty says whether t or an interface it extends lets name be
// left out.
func optionalProperty(t *InterfaceType, name string) bool {
	if t.OptionalProps[name] {
		return true
	}
	for _, parent := range t.Extends {
		if optionalProperty(parent, name) {
			return true
		}
	}
	return false
}

// recordType implements Record<K, V> utility type
// Constructs a type with a set of properties K of type V
func (c *Checker) recordType(keys Type, valueType Type) Type {
	// Record<string, V> is a table from any string to V; only a union of
	// literal keys lists its fields.
	switch keys.(type) {
	case *StringType, *NumberType, *AnyType:
		return &TableType{KeyType: keys, ValueType: valueType}
	}
	record := &InterfaceType{
		Name:        "Record",
		Properties:  make(map[string]Type),
		Methods:     make(map[string]*FunctionType),
		StaticProps: make(map[string]bool),
	}

	keyList := c.extractKeyNames(keys)
	for _, key := range keyList {
		record.Properties[key] = valueType
	}

	return record
}

// partialType implements Partial<T> utility type
// Makes all properties in T optional
// readonlyType implements Readonly<T>: the same shape, with every property
// marked readonly so assigning to one is reported.
func (c *Checker) readonlyType(t Type) Type {
	switch objType := t.(type) {
	case *InterfaceType:
		readonly := &InterfaceType{
			Name:        objType.Name + "_Readonly",
			Properties:  make(map[string]Type),
			Methods:     make(map[string]*FunctionType),
			StaticProps: make(map[string]bool),
			Extends:     []*InterfaceType{},
		}

		readonly.OptionalProps = map[string]bool{}
		readonly.ReadonlyProps = map[string]bool{}
		for name, propType := range allProperties(objType) {
			readonly.Properties[name] = propType
			readonly.ReadonlyProps[name] = true
			if optionalProperty(objType, name) {
				readonly.OptionalProps[name] = true
			}
			if objType.StaticProps != nil && objType.StaticProps[name] {
				readonly.StaticProps[name] = true
			}
		}
		for name, methodType := range objType.Methods {
			readonly.Methods[name] = methodType
		}

		return readonly

	case *ClassType:
		readonly := &ClassType{
			Name:          objType.Name + "_Readonly",
			Properties:    make(map[string]Type),
			Methods:       make(map[string]*FunctionType),
			ReadonlyProps: make(map[string]bool),
		}

		for name, propType := range objType.Properties {
			readonly.Properties[name] = propType
			readonly.ReadonlyProps[name] = true
		}
		for name, methodType := range objType.Methods {
			readonly.Methods[name] = methodType
		}

		return readonly
	}

	return t
}

func (c *Checker) partialType(t Type) Type {
	switch objType := t.(type) {
	case *InterfaceType:
		partial := &InterfaceType{
			Name:        objType.Name + "_Partial",
			Properties:  make(map[string]Type),
			Methods:     make(map[string]*FunctionType),
			StaticProps: make(map[string]bool),
		}

		// Make all properties optional (T | nil), inherited ones included
		partial.OptionalProps = map[string]bool{}
		for name, propType := range allProperties(objType) {
			partial.Properties[name] = withNil(propType)
			partial.OptionalProps[name] = true
			if objType.StaticProps != nil && objType.StaticProps[name] {
				partial.StaticProps[name] = true
			}
		}
		for name, methodType := range objType.Methods {
			partial.Methods[name] = methodType
		}
		return partial

	case *ClassType:
		partial := &ClassType{
			Name:       objType.Name + "_Partial",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		for name, propType := range objType.Properties {
			partial.Properties[name] = &UnionType{Types: []Type{propType, Nil}}
		}
		for name, methodType := range objType.Methods {
			partial.Methods[name] = methodType
		}
		return partial

	default:
		return t
	}
}

// requiredType implements Required<T> utility type
// Makes all properties in T required (removes optionality)
func (c *Checker) requiredType(t Type) Type {
	switch objType := t.(type) {
	case *InterfaceType:
		required := &InterfaceType{
			Name:        objType.Name + "_Required",
			Properties:  make(map[string]Type),
			Methods:     make(map[string]*FunctionType),
			StaticProps: make(map[string]bool),
		}

		// Remove optionality from properties
		for name, propType := range objType.Properties {
			required.Properties[name] = c.nonNullable(propType)
			if objType.StaticProps != nil && objType.StaticProps[name] {
				required.StaticProps[name] = true
			}
		}
		for name, methodType := range objType.Methods {
			required.Methods[name] = methodType
		}
		return required

	case *ClassType:
		required := &ClassType{
			Name:       objType.Name + "_Required",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		for name, propType := range objType.Properties {
			required.Properties[name] = c.nonNullable(propType)
		}
		for name, methodType := range objType.Methods {
			required.Methods[name] = methodType
		}
		return required

	default:
		return t
	}
}

// returnType implements ReturnType<F> utility type
// Extracts the return type from a function type
func (c *Checker) returnType(f Type) Type {
	if funcType, ok := f.(*FunctionType); ok {
		return funcType.ReturnType
	}
	return Any
}

// parametersType implements Parameters<F> utility type
// Extracts parameter types from a function type as a tuple
func (c *Checker) parametersType(f Type) Type {
	if funcType, ok := f.(*FunctionType); ok {
		return &TupleType{Elements: funcType.Parameters}
	}
	return Never
}

// extractKeyNames extracts string literal names from a key type
func (c *Checker) extractKeyNames(keys Type) []string {
	var names []string

	switch k := keys.(type) {
	case *StringLiteralType:
		names = append(names, k.Value)
	case *UnionType:
		for _, member := range k.Types {
			if strLit, ok := member.(*StringLiteralType); ok {
				names = append(names, strLit.Value)
			}
		}
	}

	return names
}

// resolveKeyof extracts keys from an object type and returns a union of string literals
func (c *Checker) resolveKeyof(objectType Type) Type {
	var keys []Type

	switch t := objectType.(type) {
	case *ClassType:
		// Extract property names from class
		for _, propName := range memberNames(t.Properties) {
			keys = append(keys, &StringLiteralType{Value: propName})
		}
		for _, methodName := range memberNames(t.Methods) {
			keys = append(keys, &StringLiteralType{Value: methodName})
		}

	case *InterfaceType:
		// Extract property names from interface, inherited ones included
		for _, propName := range memberNames(allProperties(t)) {
			keys = append(keys, &StringLiteralType{Value: propName})
		}
		for _, methodName := range memberNames(allMethods(t)) {
			keys = append(keys, &StringLiteralType{Value: methodName})
		}

	case *TableType:
		// For table types, keyof returns the key type
		return t.KeyType

	case *TupleType:
		// For tuples, return numeric literal types for indices
		for i := range t.Elements {
			keys = append(keys, &NumberLiteralType{Value: float64(i)})
		}

	default:
		// For other types, keyof returns never (no keys)
		return Never
	}

	if len(keys) == 0 {
		return Never
	}

	if len(keys) == 1 {
		return keys[0]
	}

	return &UnionType{Types: keys}
}

// containsTypeof says whether a type expression uses typeof.
func containsTypeof(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return false
		}
		if _, ok := v.Interface().(*ast.TypeofExpression); ok {
			return true
		}
		return containsTypeof(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && containsTypeof(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if containsTypeof(v.Index(i)) {
				return true
			}
		}
	}
	return false
}

// resolveTypeof resolves the typeof operator, returning the type of an expression
func (c *Checker) resolveTypeof(expr ast.Expression) Type {
	// Check the type of the expression
	exprType := c.checkExpression(expr)
	return exprType
}

// resolveConditionalType resolves conditional types: T extends U ? X : Y
func (c *Checker) resolveConditionalType(checkType, extendsType, trueType, falseType Type) Type {
	// Check if checkType extends (is assignable to) extendsType
	if checkType.IsAssignableTo(extendsType) {
		return trueType
	}
	return falseType
}

// substituteTypeParams substitutes type parameters in a type expression
// For example: substituting T with string in (nil | T) yields (nil | string)
func (c *Checker) substituteTypeParams(body ast.Expression, typeParams []string, typeArgs []Type) Type {
	if body == nil {
		return Any
	}

	// Create a substitution map
	substitutions := make(map[string]Type)
	for i, param := range typeParams {
		substitutions[param] = typeArgs[i]
	}

	// Create a new environment with substitutions
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	for param, typ := range substitutions {
		c.env.Set(param, typ)
	}

	// Resolve the body with the substituted environment
	result := c.resolveTypeExpression(body)

	// Restore environment
	c.env = prevEnv

	return result
}

// instantiateGenericClass creates a new ClassType with type parameters substituted
func (c *Checker) instantiateGenericClass(classType *ClassType, typeArgs []Type) *ClassType {
	if classType.GenericOrigin != nil {
		classType = classType.GenericOrigin
	}
	// Create substitution map
	substitutions := make(map[string]Type)
	for i, param := range classType.GenericParams {
		if i < len(typeArgs) {
			substitutions[param] = typeArgs[i]
		}
	}
	// One instance per concrete set of arguments, registered before its
	// members are substituted: a method returning Box<T> refers back to it.
	concrete := true
	key := fmt.Sprintf("%p", classType)
	for _, a := range typeArgs {
		key += "|" + a.String()
		if containsGenericParam(a) {
			concrete = false
		}
	}
	if concrete {
		if inst, ok := c.classInstances[key]; ok {
			return inst
		}
	}

	// Create a new class type with substituted types
	instantiated := &ClassType{
		Name:               classType.Name,
		Properties:         make(map[string]Type),
		Methods:            make(map[string]*FunctionType),
		StaticProperties:   make(map[string]Type),
		StaticMethods:      make(map[string]*FunctionType),
		Constructor:        nil,
		Parent:             classType.Parent,
		Implements:         classType.Implements,
		IsAbstract:         classType.IsAbstract,
		AbstractMethods:    classType.AbstractMethods,
		GenericParams:      []string{}, // Instantiated classes are not generic
		PropertyVisibility: classType.PropertyVisibility,
		MethodVisibility:   classType.MethodVisibility,
		MemberSyms:         classType.MemberSyms,
		GenericOrigin:      classType,
		TypeArgs:           typeArgs,
	}
	if concrete {
		if c.classInstances == nil {
			c.classInstances = map[string]*ClassType{}
		}
		c.classInstances[key] = instantiated
	}

	// Substitute properties
	for name, typ := range classType.Properties {
		instantiated.Properties[name] = c.substituteType(typ, substitutions)
	}

	// Substitute methods
	for name, funcType := range classType.Methods {
		instantiated.Methods[name] = c.substituteFunctionType(funcType, substitutions)
	}

	// Substitute static properties
	for name, typ := range classType.StaticProperties {
		instantiated.StaticProperties[name] = c.substituteType(typ, substitutions)
	}

	// Substitute static methods
	for name, funcType := range classType.StaticMethods {
		instantiated.StaticMethods[name] = c.substituteFunctionType(funcType, substitutions)
	}

	// Substitute constructor
	if classType.Constructor != nil {
		substitutedConstructor := c.substituteFunctionType(classType.Constructor, substitutions)
		// The constructor should return the instantiated class, not the generic class
		substitutedConstructor.ReturnType = instantiated
		instantiated.Constructor = substitutedConstructor
	}

	return instantiated
}

// instantiateGenericFunction creates a new FunctionType with type parameters substituted
func (c *Checker) instantiateGenericFunction(funcType *FunctionType, typeArgs []Type) *FunctionType {
	// Create substitution map
	substitutions := make(map[string]Type)
	for i, param := range funcType.GenericParams {
		substitutions[param] = typeArgs[i]
	}

	return c.substituteFunctionType(funcType, substitutions)
}

// substituteFunctionType creates a new FunctionType with type parameters substituted
func (c *Checker) substituteFunctionType(funcType *FunctionType, substitutions map[string]Type) *FunctionType {
	newParams := make([]Type, len(funcType.Parameters))
	for i, param := range funcType.Parameters {
		newParams[i] = c.substituteType(param, substitutions)
	}

	newReturnType := c.substituteType(funcType.ReturnType, substitutions)

	return &FunctionType{
		Parameters:       newParams,
		ReturnType:       newReturnType,
		GenericParams:    []string{}, // Instantiated functions are not generic
		OptionalParams:   funcType.OptionalParams,
		HasRestParameter: funcType.HasRestParameter,
		ParamNames:       funcType.ParamNames,
		Untyped:          funcType.Untyped,
	}
}

// containsGenericParam says whether a type still names a type parameter.
func containsGenericParam(t Type) bool {
	switch v := t.(type) {
	case *GenericParamType:
		return true
	case *ArrayType:
		return containsGenericParam(v.ElementType)
	case *OptionalType:
		return containsGenericParam(v.BaseType)
	case *UnionType:
		for _, m := range v.Types {
			if containsGenericParam(m) {
				return true
			}
		}
	case *TableType:
		return containsGenericParam(v.KeyType) || containsGenericParam(v.ValueType)
	case *ClassType:
		for _, a := range v.TypeArgs {
			if containsGenericParam(a) {
				return true
			}
		}
	}
	return false
}

// substituteType replaces type parameter references with actual types
func (c *Checker) substituteType(typ Type, substitutions map[string]Type) Type {
	if typ == nil {
		return nil
	}

	switch t := typ.(type) {
	case *GenericParamType:
		// This is a type parameter reference - substitute it
		if subst, exists := substitutions[t.Name]; exists {
			return subst
		}
		return t

	case *ArrayType:
		return &ArrayType{ElementType: c.substituteType(t.ElementType, substitutions)}

	case *TableType:
		return &TableType{
			KeyType:   c.substituteType(t.KeyType, substitutions),
			ValueType: c.substituteType(t.ValueType, substitutions),
		}

	case *FunctionType:
		return c.substituteFunctionType(t, substitutions)

	case *UnionType:
		newTypes := make([]Type, len(t.Types))
		for i, typ := range t.Types {
			newTypes[i] = c.substituteType(typ, substitutions)
		}
		return &UnionType{Types: newTypes}

	case *OptionalType:
		return &OptionalType{BaseType: c.substituteType(t.BaseType, substitutions)}

	case *TupleType:
		newElements := make([]Type, len(t.Elements))
		for i, elem := range t.Elements {
			newElements[i] = c.substituteType(elem, substitutions)
		}
		return &TupleType{Elements: newElements}

	case *ClassType:
		// Box<T> inside Box: instantiated again with what T now is.
		if t.GenericOrigin != nil && containsGenericParam(t) {
			args := make([]Type, len(t.TypeArgs))
			for i, a := range t.TypeArgs {
				args[i] = c.substituteType(a, substitutions)
			}
			return c.instantiateGenericClass(t.GenericOrigin, args)
		}
		return t

	default:
		// For all other types (primitives, etc.), return as-is
		return t
	}
}

// checkStatement checks a statement
func (c *Checker) checkStatement(stmt ast.Statement) {
	if isNilNode(stmt) {
		return
	}
	if stmt == nil {
		return
	}

	switch node := stmt.(type) {
	case *ast.VariableDeclaration:
		c.checkVariableDeclaration(node)
	case *ast.FunctionDeclaration:
		c.checkFunctionDeclaration(node)
	case *ast.ExpressionStatement:
		c.checkExpression(node.Expression)
		// assert(x) leaves x what a true x proves for the rest of the block.
		if call, ok := node.Expression.(*ast.CallExpression); ok && len(call.Arguments) > 0 {
			if fn, ok := call.Function.(*ast.Identifier); ok && fn.Value == "assert" {
				c.applyFacts(c.narrowings(call.Arguments[0], true))
			}
		}
	case *ast.ReturnStatement:
		c.checkReturnStatement(node)
	case *ast.IfStatement:
		c.checkIfStatement(node)
	case *ast.WhileStatement:
		c.checkWhileStatement(node)
	case *ast.RepeatStatement:
		c.checkRepeatStatement(node)
	case *ast.ForStatement:
		c.checkForStatement(node)
	case *ast.DoStatement:
		c.checkDoStatement(node)
	case *ast.BreakStatement, *ast.GotoStatement, *ast.LabelStatement:
		// Checked elsewhere (gotos against labels, by checkGotos)
	case *ast.BlockStatement:
		c.checkBlockStatement(node)
	case *ast.AssignmentStatement:
		c.checkAssignmentStatement(node)
	case *ast.ClassDeclaration:
		c.checkClassDeclaration(node)
	case *ast.InterfaceDeclaration:
		// Interface declarations don't need runtime checking; the names
		// they extend are uses of those interfaces.
		for _, ext := range node.Extends {
			c.referenceTypeName(ext)
		}
	case *ast.EnumDeclaration:
		// Enum declarations don't need runtime checking
	case *ast.TypeDeclaration:
		// Type declarations don't need runtime checking, except one that
		// takes the type of a value (typeof), resolved here.
		if node.Type != nil && containsTypeof(reflect.ValueOf(node.Type)) {
			c.resolvingTypeof = true
			c.registerTypeAlias(node)
			c.resolvingTypeof = false
		}
	case *ast.NamespaceDeclaration:
		c.checkNamespaceDeclaration(node)
	case *ast.DeclareStatement:
		// Ambient declarations - register without checking implementation
		c.checkDeclareStatement(node)
	case *ast.ExportStatement:
		c.checkExportStatement(node)
	case *ast.ImportStatement:
		c.checkImportStatement(node)
	}
}

// checkVariableDeclaration checks a variable declaration
func (c *Checker) checkVariableDeclaration(node *ast.VariableDeclaration) {
	c.checkAttribs(node)
	// Resolve declared types for each variable
	declaredTypes := make([]Type, len(node.Names))
	for i, typ := range node.Types {
		if typ != nil {
			declaredTypes[i] = c.resolveTypeExpression(typ)
		}
	}

	// A table literal written for an interface: an untyped self in the
	// functions it gives for the interface's methods is that interface, so
	// calls through self get their ':'.
	for i, value := range node.Values {
		if i < len(node.Types) && node.Types[i] != nil {
			if iface, ok := declaredTypes[i].(*InterfaceType); ok {
				if lit, ok := value.(*ast.TableLiteral); ok {
					typeMethodSelf(lit, iface, node.Types[i])
				}
			}
		}
	}

	// Handle no value assignment
	if len(node.Values) == 0 {
		// Register variables with their declared types (or nil)
		for i, name := range node.Names {
			var varType Type
			if declaredTypes[i] != nil {
				varType = declaredTypes[i]
			} else {
				// `local x` with no type: it takes whatever it is given.
				varType = Any
			}

			if node.IsConstant || attribOf(node, i) == "const" {
				c.env.SetConst(name.Value, varType)
			} else {
				c.env.SetLocal(name.Value, varType)
			}
			c.define(name.Token, SymVariable, varType)
		}
		return
	}

	// Get value types
	var valueTypes []Type

	// If there's only one value and it's a call expression, it might return a tuple
	if len(node.Values) == 1 {
		valueType := c.checkExpression(node.Values[0])

		// Check if it's a tuple type (multiple return values)
		if tupleType, ok := valueType.(*TupleType); ok {
			valueTypes = tupleType.Elements
		} else {
			valueTypes = []Type{valueType}
		}
	} else {
		// Multiple values - evaluate each
		valueTypes = make([]Type, len(node.Values))
		for i, val := range node.Values {
			valueTypes[i] = c.checkExpression(val)
		}
	}

	// A call or '...' can yield more values than its type records, which is how
	// `local ok, err = pcall(f)` is written, so extra names take 'any'.
	if len(node.Names) > len(valueTypes) && len(node.Values) == 1 && yieldsMultipleValues(node.Values[0]) {
		for len(valueTypes) < len(node.Names) {
			valueTypes = append(valueTypes, Any)
		}
	}

	// Lua adjusts the values to the names: a missing one is nil and an
	// extra one is dropped (local a, b = 1; local ok = pcall(f)). A name
	// typed as non-nil that is left without a value is reported below.
	for len(valueTypes) < len(node.Names) {
		valueTypes = append(valueTypes, Nil)
	}
	valueTypes = valueTypes[:len(node.Names)]

	// Register each variable with type checking
	for i, name := range node.Names {
		declaredType := declaredTypes[i]
		valueType := valueTypes[i]

		// Determine final type and check assignability
		var finalType Type
		if declaredType != nil {
			// Type is declared - check if value is assignable. A table
			// literal is checked field by field, naming the field at fault.
			if i < len(node.Values) && c.reportLiteralFields(node.Values[i], declaredType) {
				// reported
			} else if !valueType.IsAssignableTo(declaredType) {
				c.addError(
					fmt.Sprintf("Cannot assign type '%s' to variable '%s' of type '%s'",
						valueType.String(), name.Value, declaredType.String()),
					node.Token,
				)
			}
			finalType = declaredType
		} else {
			// Infer type from value
			finalType = valueType

			// `local x = nil` with no type declares a variable to fill later.
			if IsNilType(finalType) && !node.IsConstant {
				finalType = Any
			}
			// A mutable binding widens a literal to its base type unless the
			// value explicitly asked to stay literal with `as const`.
			if !node.IsConstant && (i >= len(node.Values) || !isConstAssertion(node.Values[i])) {
				finalType = widenLiteralType(finalType)
			}
		}

		// Register variable
		if node.IsConstant || attribOf(node, i) == "const" {
			c.env.SetConst(name.Value, finalType)
		} else {
			c.env.SetLocal(name.Value, finalType)
			if declaredTypes[i] == nil {
				c.env.MarkInferred(name.Value)
			}
		}
		c.define(name.Token, SymVariable, finalType)
	}
}

// requiredParamCount returns how many arguments a signature demands, which is
// the number of parameters before the first optional one.
//
// A parameter that may be nil (b: number?) can be left out only when every
// parameter after it can be too; before a required one it must be passed,
// if only as nil.
func requiredParamCount(fnType *FunctionType) int {
	required := len(fnType.Parameters)
	if fnType.HasRestParameter && required > 0 {
		required-- // the rest takes any number, none included
	}
	for required > 0 {
		i := required - 1
		if i >= len(fnType.OptionalParams) || !fnType.OptionalParams[i] {
			break
		}
		required--
	}
	return required
}

// yieldsMultipleValues reports whether an expression can produce more than one
// value in Lua. Only calls and the vararg expression can.
func yieldsMultipleValues(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.CallExpression, *ast.VarargExpression:
		return true
	}

	return false
}

// paramIsOptional reports whether a parameter may be omitted at a call site.
// Both spellings count: `b?: number` and the documented `b: number?`, which is
// the same thing in Lua, where a missing argument arrives as nil.
func paramIsOptional(param *ast.Parameter, resolved Type) bool {
	if param.IsOptional {
		return true
	}
	// An untyped parameter is a Lua parameter: left out, it is nil.
	if param.Type == nil && !param.IsRest {
		return true
	}

	// `number?` resolves to the union `number | nil`, so ask whether nil is one
	// of the type's members rather than looking for a specific representation.
	_, admitsNil := removeNil(resolved)

	return admitsNil
}

// widenLiteralType generalises a literal type to its base type. A mutable
// binding declared without an annotation has to accept later values of the same
// base type, so `local state = "idle"` is a string rather than the type "idle".
// An explicit annotation or `as const` is how a literal type is kept.
func widenLiteralType(t Type) Type {
	switch typ := t.(type) {
	case *StringLiteralType:
		return String
	case *NumberLiteralType:
		return Number
	case *BooleanLiteralType:
		return Boolean
	case *ArrayType:
		// `local xs = {1, 2, 3}` holds numbers, not just 1, 2 and 3.
		if elem := widenLiteralType(typ.ElementType); elem != typ.ElementType {
			return &ArrayType{ElementType: elem}
		}
	case *UnionType:
		var members []Type
		seen := map[string]bool{}
		changed := false
		for _, m := range typ.Types {
			w := widenLiteralType(m)
			changed = changed || w != m
			if key := w.String(); !seen[key] {
				seen[key] = true
				members = append(members, w)
			}
		}
		if !changed {
			return t
		}
		if len(members) == 1 {
			return members[0]
		}
		return &UnionType{Types: members}
	}

	return t
}

// isConstAssertion reports whether an expression ends in `as const`, which asks
// for the literal type to be preserved.
func isConstAssertion(expr ast.Expression) bool {
	assertion, ok := expr.(*ast.TypeAssertion)
	if !ok {
		return false
	}

	ident, ok := assertion.TargetType.(*ast.Identifier)
	return ok && ident.Value == "const"
}

// checkFunctionDeclaration checks a function declaration
func (c *Checker) checkFunctionDeclaration(node *ast.FunctionDeclaration) {
	if node.Receiver != nil {
		c.checkFieldFunction(node)
		return
	}
	if node.IsAsync {
		c.addErrorCode("not-supported", "'async' functions are not supported: nothing would resume them. Use coroutine.create and coroutine.resume", node.Token)
	}

	// Check decorators
	for _, decorator := range node.Decorators {
		c.checkDecorator(decorator)
	}

	// Add generic type parameters to current scope first (for type resolution)
	prevEnv := c.env
	if len(node.GenericParams) > 0 {
		c.env = NewEnclosedEnvironment(prevEnv)
		for _, genericParam := range node.GenericParams {
			// Store as GenericParamType so we can substitute it later
			c.env.Set(genericParam.Value, &GenericParamType{Name: genericParam.Value})
		}
	}

	// Create function type
	params := make([]Type, len(node.Parameters))
	optionalParams := make([]bool, len(node.Parameters))
	hasOptional := false

	for i, param := range node.Parameters {
		// Validate rest parameters
		if param.IsRest {
			// Rest parameter must be last
			if i != len(node.Parameters)-1 {
				c.addError("Rest parameter must be the last parameter", param.Token)
			}
			// A rest parameter may be annotated with the element type (...n: number)
			// or with the array it collects (...n: number[]); function type
			// annotations already accept both, so declarations do too.
			if param.Type != nil {
				paramType := c.paramType(param)
				if _, ok := paramType.(*ArrayType); !ok {
					paramType = &ArrayType{ElementType: paramType}
				}
				params[i] = paramType
			} else {
				// Default to any[] for rest parameters
				params[i] = &ArrayType{ElementType: Any}
			}
		} else {
			// Validate optional parameter ordering. Only `name?:` says a
			// parameter may be left out; `name: T?` may be nil, and before a
			// required parameter it still has to be passed.
			if paramIsOptional(param, c.paramType(param)) {
				hasOptional = hasOptional || param.IsOptional
				optionalParams[i] = true
			} else if hasOptional && !param.IsRest {
				// Required parameter after optional parameter
				c.addError("Required parameters cannot follow optional parameters", param.Token)
			}

			if param.Type != nil {
				params[i] = c.paramType(param)
			} else {
				params[i] = Any
			}
		}
	}

	var returnType Type = Void
	if node.ReturnType != nil {
		returnType = c.resolveTypeExpression(node.ReturnType)
	}

	// Extract generic parameter names
	genericParams := make([]string, len(node.GenericParams))
	for i, param := range node.GenericParams {
		genericParams[i] = param.Value
	}

	// Check if the last parameter is a rest parameter
	hasRestParam := false
	if len(node.Parameters) > 0 && node.Parameters[len(node.Parameters)-1].IsRest {
		hasRestParam = true
	}

	funcType := &FunctionType{
		Parameters:       params,
		ReturnType:       returnType,
		GenericParams:    genericParams,
		HasRestParameter: hasRestParam,
		OptionalParams:   optionalParams,
		Untyped:          untypedSignature(node.Parameters, node.ReturnType, len(node.GenericParams)),
	}

	// Restore environment and register function
	if len(node.GenericParams) > 0 {
		c.env = prevEnv
	}

	// A placeholder this checker hoisted is replaced, not overloaded: the
	// declaration being checked now is the same function.
	if placeholder, wasHoisted := c.hoisted[node.Name.Value]; wasHoisted {
		if existingType, ok := c.env.Get(node.Name.Value); ok && existingType == Type(placeholder) {
			delete(c.hoisted, node.Name.Value)
			c.env.Set(node.Name.Value, funcType)
		}
	}

	// A script may define a global that a declaration file describes, so
	// the scripts calling it know its type; an untyped parameter or return
	// takes the declared type: `function ring_run(n)` for
	// `declare function ring_run(n: number): number`.
	contextual := false
	if declared, ok := c.env.Get(node.Name.Value); ok && !node.IsLocal {
		if fn, isFn := declared.(*FunctionType); isFn && len(fn.Overloads) == 0 &&
			(c.declaredFuncs[fn] || c.ambient[node.Name.Value] == declared) {
			for i, param := range node.Parameters {
				if param.Type == nil && !param.IsRest && i < len(fn.Parameters) {
					params[i] = fn.Parameters[i]
					funcType.Parameters[i] = fn.Parameters[i]
					optional := i < len(fn.OptionalParams) && fn.OptionalParams[i]
					if i < len(funcType.OptionalParams) {
						funcType.OptionalParams[i] = optional
					}
				}
			}
			if node.ReturnType == nil {
				returnType = fn.ReturnType
				funcType.ReturnType = fn.ReturnType
				contextual = true
			}
		}
	}

	// A name can have one implementation. Overloads describe external APIs
	// and are written as `declare function` signatures; two bodies for one
	// global would leave only the last at run time.
	if existingType, ok := c.env.Get(node.Name.Value); ok && existingType != Type(funcType) {
		existingFunc, isFunc := existingType.(*FunctionType)
		_, inThisScope := c.env.store[node.Name.Value]
		declaredGlobal := c.ambient[node.Name.Value] == existingType && !node.IsLocal
		switch {
		case node.IsLocal:
			// local function assert(v) is a new local, shadowing whatever the
			// name was, as any local does.
			c.env.SetLocal(node.Name.Value, funcType)
		case declaredGlobal && !isFunc:
			// Declared as a value: any takes the function, nothing else can.
			if _, isAny := existingType.(*AnyType); !isAny {
				c.addError(fmt.Sprintf("'%s' is declared as '%s', so a function cannot define it", node.Name.Value, existingType.String()),
					node.Name.Token)
			}
			c.env.Set(node.Name.Value, funcType)
		case declaredGlobal && !c.declaredFuncs[existingFunc]:
			// Declared as a function-typed value: the definition has to fit.
			if !funcType.IsAssignableTo(existingFunc) {
				c.addError(fmt.Sprintf("Function '%s' does not match its declaration '%s'",
					node.Name.Value, existingFunc.String()), node.Name.Token)
			}
		case isFunc && c.declaredFuncs[existingFunc]:
			// Implementing a declared function: it has to fit the declaration,
			// which stays the type callers see.
			if len(existingFunc.Overloads) == 0 && !funcType.IsAssignableTo(existingFunc) {
				c.addError(fmt.Sprintf("Function '%s' does not match its declaration '%s'",
					node.Name.Value, existingFunc.String()), node.Name.Token)
			}
		case isFunc && inThisScope && !node.IsLocal && c.funcDefs[existingFunc].Line > 0:
			c.addErrorCode("duplicate-function", fmt.Sprintf("Function '%s' is already defined at line %d. Overloads can only be declared with 'declare function'",
				node.Name.Value, c.funcDefs[existingFunc].Line), node.Name.Token)
		case !isFunc && inThisScope && !node.IsLocal && !existingType.Equals(Any):
			// `function f()` assigns f; a local f holding a number cannot
			// take a function. (local f then function f() is Lua's
			// forward declaration, and local function f() a new local.)
			c.addError(
				fmt.Sprintf("Cannot define function '%s': '%s' is a '%s' here", node.Name.Value, node.Name.Value, existingType.String()),
				node.Token,
			)
			c.env.Set(node.Name.Value, funcType)
		default:
			// A new binding, possibly shadowing one from an outer scope.
			c.env.Set(node.Name.Value, funcType)
		}
	} else if !node.IsLocal {
		// `function f()` with no f in scope defines a global, wherever it
		// is written: one in a do block is visible after it.
		root := c.env
		for root.outer != nil {
			root = root.outer
		}
		root.Set(node.Name.Value, funcType)
	} else {
		c.env.Set(node.Name.Value, funcType)
	}
	if c.funcDefs == nil {
		c.funcDefs = make(map[*FunctionType]lexer.Token)
	}
	c.funcDefs[funcType] = node.Name.Token
	funcType.ParamNames = parameterNames(node.Parameters)
	c.define(node.Name.Token, SymFunction, funcType)

	// Check function body in new scope
	prevReturnType := c.currentFunctionReturnType
	c.env = NewEnclosedEnvironment(c.env)
	c.currentFunctionReturnType = returnType

	// Without an annotation the return type is inferred from the body rather
	// than forced to void, so plain Lua functions keep working; one that
	// defines a declared function returns what the declaration says.
	prevInferring, prevInferred := c.inferringReturn, c.inferredReturns
	c.inferringReturn = node.ReturnType == nil && !contextual
	c.inferredReturns = nil

	if c.inferringReturn {
		// The signature is already visible to the body, so a recursive call
		// would otherwise be checked against the not-yet-inferred placeholder
		// and report things like "'*' cannot be applied to type 'void'".
		funcType.ReturnType = Any
	}

	// Add generic type parameters to scope
	for _, genericParam := range node.GenericParams {
		c.env.Set(genericParam.Value, Any)
	}

	// Parameters and the body's locals are in scope until the body ends.
	prevScopeEnd := c.scopeEnd
	if node.Body != nil {
		c.scopeEnd = node.Body.EndToken.End
	}

	// Add parameters to scope
	for i, param := range node.Parameters {
		c.env.Set(param.Name.Value, params[i])
		c.define(param.Name.Token, SymParameter, params[i])
	}

	// Check body
	c.checkBlockStatement(node.Body)
	c.scopeEnd = prevScopeEnd

	if c.inferringReturn {
		// funcType is already registered; callers see the inferred type because
		// they hold the same pointer.
		returns := make([]Type, len(c.inferredReturns))
		for i, t := range c.inferredReturns {
			returns[i] = withoutSelf(t, funcType)
		}
		funcType.ReturnType = combineReturnTypes(returns)
		// A plain Lua function returning different kinds of value has
		// callers that know which one they get: it returns any.
		if _, isUnion := funcType.ReturnType.(*UnionType); isUnion && funcType.Untyped {
			funcType.ReturnType = Any
		}
	}

	c.inferringReturn, c.inferredReturns = prevInferring, prevInferred
	c.env = prevEnv
	c.currentFunctionReturnType = prevReturnType
}

// parameterNames lists a signature's parameter names, for signature help.
func parameterNames(params []*ast.Parameter) []string {
	names := make([]string, len(params))
	for i, p := range params {
		if p != nil && p.Name != nil {
			names[i] = p.Name.Value
		}
	}
	return names
}

// combineReturnTypes reduces the return types seen in a body to one type: void
// when nothing is returned, the single type when they agree, otherwise a union.
// withoutSelf replaces fn inside a type its own body returns (`return f`)
// by a copy returning any: a function type containing itself never ends
// when printed or compared.
func withoutSelf(t Type, fn *FunctionType) Type {
	switch v := t.(type) {
	case *FunctionType:
		if v == fn {
			flat := *fn
			flat.ReturnType = Any
			return &flat
		}
	case *UnionType:
		members := make([]Type, len(v.Types))
		for i, m := range v.Types {
			members[i] = withoutSelf(m, fn)
		}
		return &UnionType{Types: members}
	case *TupleType:
		elems := make([]Type, len(v.Elements))
		for i, e := range v.Elements {
			elems[i] = withoutSelf(e, fn)
		}
		return &TupleType{Elements: elems}
	case *OptionalType:
		return &OptionalType{BaseType: withoutSelf(v.BaseType, fn)}
	case *ArrayType:
		return &ArrayType{ElementType: withoutSelf(v.ElementType, fn), EmptyLiteral: v.EmptyLiteral}
	}
	return t
}

func combineReturnTypes(types []Type) Type {
	if len(types) == 0 {
		return Void
	}

	unique := make([]Type, 0, len(types))
	seen := make(map[string]bool)
	for _, t := range types {
		// An inferred return widens its literals, the same as an inferred
		// variable: `function f() return 1 end` returns number, not 1.
		t = widenLiteralType(t)
		if key := t.String(); !seen[key] {
			seen[key] = true
			unique = append(unique, t)
		}
	}

	if len(unique) == 1 {
		return unique[0]
	}

	// A union that includes 'any' is just 'any'. This is what a recursive call
	// contributes while its own return type is still being inferred.
	for _, t := range unique {
		if t.Equals(Any) {
			return Any
		}
	}

	return &UnionType{Types: unique}
}

// checkReturnStatement checks a return statement
func (c *Checker) checkReturnStatement(node *ast.ReturnStatement) {
	if c.currentFunctionReturnType == nil {
		// A return in the chunk itself: Lua runs a file as a function, so a
		// data file ends in `return { ... }` and a script may return early.
		for _, value := range node.ReturnValues {
			c.checkExpression(value)
		}
		c.chunkReturns = append(c.chunkReturns, node)
		return
	}

	// The enclosing function declared no return type: collect what it actually
	// returns instead of checking against a type nobody wrote.
	if c.inferringReturn {
		switch len(node.ReturnValues) {
		case 0:
			c.inferredReturns = append(c.inferredReturns, Void)
		case 1:
			c.inferredReturns = append(c.inferredReturns, c.checkExpression(node.ReturnValues[0]))
		default:
			elements := make([]Type, len(node.ReturnValues))
			for i, val := range node.ReturnValues {
				elements[i] = c.checkExpression(val)
			}
			c.inferredReturns = append(c.inferredReturns, &TupleType{Elements: elements})
		}
		return
	}

	// Handle empty return
	if len(node.ReturnValues) == 0 {
		if !IsVoidType(c.currentFunctionReturnType) {
			c.addError(
				fmt.Sprintf("Function must return a value of type '%s'",
					c.currentFunctionReturnType.String()),
				node.Token,
			)
		}
		return
	}

	// Handle single return value
	if len(node.ReturnValues) == 1 {
		returnType := c.checkExpression(node.ReturnValues[0])
		if c.reportLiteralFields(node.ReturnValues[0], c.currentFunctionReturnType) {
			return
		}
		if !returnType.IsAssignableTo(c.currentFunctionReturnType) {
			c.addError(
				fmt.Sprintf("Cannot return type '%s' from function with return type '%s'",
					returnType.String(), c.currentFunctionReturnType.String()),
				node.Token,
			)
		}
		return
	}

	// Handle multiple return values - must match TupleType
	returnTypes := make([]Type, len(node.ReturnValues))
	for i, val := range node.ReturnValues {
		returnTypes[i] = c.checkExpression(val)
	}

	// Check if expected return type is a tuple
	tupleType, ok := c.currentFunctionReturnType.(*TupleType)
	if !ok {
		c.addError(
			fmt.Sprintf("Cannot return multiple values from function with return type '%s'",
				c.currentFunctionReturnType.String()),
			node.Token,
		)
		return
	}

	// Check tuple element count
	if len(returnTypes) != len(tupleType.Elements) {
		c.addError(
			fmt.Sprintf("Function expects %d return values, got %d",
				len(tupleType.Elements), len(returnTypes)),
			node.Token,
		)
		return
	}

	// Check each return value type
	for i, retType := range returnTypes {
		if !retType.IsAssignableTo(tupleType.Elements[i]) {
			c.addError(
				fmt.Sprintf("Return value %d: cannot return type '%s', expected '%s'",
					i+1, retType.String(), tupleType.Elements[i].String()),
				node.Token,
			)
		}
	}
}

// checkCondition accepts any value as a condition, as Lua does: everything
// but nil and false is true. Only an expression with no value at all, such as
// a call to a function returning void, cannot be tested.
func (c *Checker) checkCondition(condType Type, kind string, tok lexer.Token) {
	if IsVoidType(condType) {
		c.addError(fmt.Sprintf("%s condition has no value: the expression returns nothing", kind), tok)
	}
}

// checkIfStatement checks an if statement
func (c *Checker) checkIfStatement(node *ast.IfStatement) {
	condType := c.checkExpression(node.Condition)
	c.checkCondition(condType, "If", node.Token)

	// What the condition proves holds in its branch, and its opposite in
	// the other; a branch that always exits leaves the other's facts for
	// the rest of the block: `if x == nil then return end`.
	whenTrue := c.narrowings(node.Condition, true)
	whenFalse := c.narrowings(node.Condition, false)

	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	c.applyFacts(whenTrue)
	c.checkBlockStatement(node.Consequence)
	c.env = prevEnv
	if node.Alternative != nil {
		c.env = NewEnclosedEnvironment(prevEnv)
		c.applyFacts(whenFalse)
		c.checkBlockStatement(node.Alternative)
		c.env = prevEnv
	}

	thenExits := blockAlwaysExits(node.Consequence)
	elseExits := node.Alternative != nil && blockAlwaysExits(node.Alternative)
	switch {
	case thenExits && !elseExits:
		c.applyFacts(whenFalse)
	case elseExits && !thenExits:
		c.applyFacts(whenTrue)
	case node.Alternative == nil:
		// `if not x then x = default end`: either x was set, or the branch
		// assigned it a value; x is what the false side proves.
		for name, t := range whenFalse {
			if blockAssigns(node.Consequence, name) {
				c.env.Narrow(name, t)
			}
		}
	}
}

// facts are what a condition proves about variables: their narrowed types.
type facts map[string]Type

// applyFacts narrows the variables in the current scope.
func (c *Checker) applyFacts(f facts) {
	for name, t := range f {
		c.env.Narrow(name, t)
	}
}

// narrowings works out what cond proves when it is truthy (when) or falsy:
// x, not x, x == nil, x ~= nil, a and b, a or b, a type guard isT(x),
// type(x) == "string", and a discriminant field s.kind == "circle".
func (c *Checker) narrowings(cond ast.Expression, when bool) facts {
	switch e := cond.(type) {
	case *ast.Identifier:
		t, ok := c.env.Get(e.Value)
		if !ok {
			return nil
		}
		if when {
			if base, optional := removeNil(t); optional {
				return facts{e.Value: base}
			}
		} else if falsyPart(t) != nil && truthyPart(t) != nil {
			if f := falsyPart(t); f != nil && IsNilType(f) {
				return facts{e.Value: Nil}
			}
		}
		return nil
	case *ast.PrefixExpression:
		if e.Operator == "not" || e.Operator == "!" {
			return c.narrowings(e.Right, !when)
		}
		return nil
	case *ast.CallExpression:
		name, narrowed := c.extractTypeNarrowing(e)
		if name == "" || narrowed == nil {
			return nil
		}
		if when {
			return facts{name: narrowed}
		}
		if original, ok := c.env.Get(name); ok {
			if rest := subtractType(original, narrowed); rest != nil {
				return facts{name: rest}
			}
		}
		return nil
	case *ast.InfixExpression:
		switch e.Operator {
		case "and", "&&":
			if !when {
				return nil
			}
			left := c.narrowings(e.Left, true)
			prev := c.env
			c.env = NewEnclosedEnvironment(prev)
			c.applyFacts(left)
			right := c.narrowings(e.Right, true)
			c.env = prev
			return mergeFacts(left, right)
		case "or", "||":
			if when {
				return nil
			}
			left := c.narrowings(e.Left, false)
			prev := c.env
			c.env = NewEnclosedEnvironment(prev)
			c.applyFacts(left)
			right := c.narrowings(e.Right, false)
			c.env = prev
			return mergeFacts(left, right)
		case "==", "~=", "!=":
			holds := (e.Operator == "==") == when
			if ident, ok := identComparedToNil(e.Left, e.Right); ok {
				return c.nilFacts(ident, holds)
			}
			if ident, ok := identComparedToNil(e.Right, e.Left); ok {
				return c.nilFacts(ident, holds)
			}
			if f := c.typeofFacts(e.Left, e.Right, holds); f != nil {
				return f
			}
			if f := c.typeofFacts(e.Right, e.Left, holds); f != nil {
				return f
			}
			if name, narrowed := c.extractDiscriminantNarrowing(e.Left, e.Right, holds); name != "" && narrowed != nil {
				return facts{name: narrowed}
			}
			if name, narrowed := c.extractTypeNarrowing(e); name != "" && narrowed != nil && holds {
				return facts{name: narrowed}
			}
		}
	}
	return nil
}

// nilFacts is what x == nil (holds) or x ~= nil (not holds) proves.
func (c *Checker) nilFacts(ident *ast.Identifier, isNil bool) facts {
	t, ok := c.env.Get(ident.Value)
	if !ok {
		return nil
	}
	base, optional := removeNil(t)
	if !optional {
		return nil
	}
	if isNil {
		return facts{ident.Value: Nil}
	}
	return facts{ident.Value: base}
}

// typeofFacts is what type(x) == "name" proves, when call is type(x) and
// lit the string.
func (c *Checker) typeofFacts(call, lit ast.Expression, holds bool) facts {
	ce, ok := call.(*ast.CallExpression)
	if !ok || len(ce.Arguments) != 1 {
		return nil
	}
	fn, ok := ce.Function.(*ast.Identifier)
	if !ok || fn.Value != "type" {
		return nil
	}
	ident, ok := ce.Arguments[0].(*ast.Identifier)
	str, okStr := lit.(*ast.StringLiteral)
	if !ok || !okStr {
		return nil
	}
	t, found := c.env.Get(ident.Value)
	if !found {
		return nil
	}
	isKind := func(m Type) bool {
		switch str.Value {
		case "string":
			return IsStringType(m)
		case "number":
			return IsNumericType(m)
		case "boolean":
			switch m.(type) {
			case *BooleanType, *BooleanLiteralType:
				return true
			}
		case "nil":
			return IsNilType(m)
		case "function":
			_, isFn := m.(*FunctionType)
			return isFn
		case "table":
			switch m.(type) {
			case *ArrayType, *TableType, *InterfaceType, *ClassType, *TupleType:
				return true
			}
		}
		return false
	}
	var members []Type
	if u, ok := t.(*UnionType); ok {
		members = u.Types
	} else if opt, ok := t.(*OptionalType); ok {
		members = []Type{opt.BaseType, Nil}
	} else {
		switch t.(type) {
		case *AnyType, *UnknownType:
			if holds {
				switch str.Value {
				case "string":
					return facts{ident.Value: String}
				case "number":
					return facts{ident.Value: Number}
				case "boolean":
					return facts{ident.Value: Boolean}
				}
			}
			return nil
		}
		members = []Type{t}
	}
	var kept []Type
	for _, m := range members {
		if isKind(m) == holds {
			kept = append(kept, m)
		}
	}
	if len(kept) == 0 || len(kept) == len(members) {
		return nil
	}
	return facts{ident.Value: unionOf(kept...)}
}

// subtractType is t without the members that fit taken (nil if none
// remain, or t is not a union).
func subtractType(t, taken Type) Type {
	u, ok := t.(*UnionType)
	if !ok {
		return nil
	}
	var rest []Type
	for _, m := range u.Types {
		if m != nil && !m.IsAssignableTo(taken) {
			rest = append(rest, m)
		}
	}
	if len(rest) == 0 || len(rest) == len(u.Types) {
		return nil
	}
	return unionOf(rest...)
}

// mergeFacts joins two sets of facts; later ones win.
func mergeFacts(a, b facts) facts {
	if len(a) == 0 {
		return b
	}
	out := facts{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// nilCheck describes an `x == nil` / `x ~= nil` test on an optional variable.
type nilCheck struct {
	varName string
	// nonNil is the variable's type with nil removed.
	nonNil Type
	// nilInThen is true when the then-branch is the one where x is nil.
	nilInThen bool
}

// nilNarrowing recognises a comparison of a variable against nil and reports how
// each branch refines it. Without this an optional parameter stays 'T | nil'
// after its own nil check and cannot be used.
func (c *Checker) nilNarrowing(expr ast.Expression) (nilCheck, bool) {
	// `if x then` and `if not x then` test truthiness, which for an optional
	// is the same as testing it against nil.
	switch e := expr.(type) {
	case *ast.Identifier:
		return c.optionalNamed(e, false)
	case *ast.PrefixExpression:
		if ident, ok := e.Right.(*ast.Identifier); ok && e.Operator == "not" {
			return c.optionalNamed(ident, true)
		}
		return nilCheck{}, false
	}

	infix, ok := expr.(*ast.InfixExpression)
	if !ok {
		return nilCheck{}, false
	}

	var equality bool
	switch infix.Operator {
	case "==":
		equality = true
	case "~=", "!=":
		equality = false
	default:
		return nilCheck{}, false
	}

	// Accept the comparison written either way round.
	ident, nilOK := identComparedToNil(infix.Left, infix.Right)
	if !nilOK {
		ident, nilOK = identComparedToNil(infix.Right, infix.Left)
	}
	if !nilOK {
		return nilCheck{}, false
	}

	return c.optionalNamed(ident, equality)
}

// optionalNamed describes the refinement of ident when it holds an optional;
// nilInThen says which branch sees nil.
func (c *Checker) optionalNamed(ident *ast.Identifier, nilInThen bool) (nilCheck, bool) {
	current, exists := c.env.Get(ident.Value)
	if !exists {
		return nilCheck{}, false
	}

	nonNil, isOptional := removeNil(current)
	if !isOptional {
		return nilCheck{}, false
	}

	return nilCheck{varName: ident.Value, nonNil: nonNil, nilInThen: nilInThen}, true
}

// identComparedToNil returns the identifier when one side of a comparison is a
// plain name and the other is the nil literal.
func identComparedToNil(candidate, other ast.Expression) (*ast.Identifier, bool) {
	ident, ok := candidate.(*ast.Identifier)
	if !ok {
		return nil, false
	}

	if _, isNil := other.(*ast.NilLiteral); !isNil {
		return nil, false
	}

	return ident, true
}

// removeNil strips nil from an optional or a union that includes it.
func removeNil(t Type) (Type, bool) {
	if optType, ok := t.(*OptionalType); ok {
		return optType.BaseType, true
	}

	union, ok := t.(*UnionType)
	if !ok {
		return t, false
	}

	remaining := make([]Type, 0, len(union.Types))
	foundNil := false
	for _, member := range union.Types {
		if IsNilType(member) {
			foundNil = true
			continue
		}
		remaining = append(remaining, member)
	}

	if !foundNil || len(remaining) == 0 {
		return t, false
	}

	if len(remaining) == 1 {
		return remaining[0], true
	}

	return &UnionType{Types: remaining}, true
}

// checkIfWithNilNarrowing checks both branches with the variable refined, and
// carries the refinement past the statement when the nil branch always exits.
func (c *Checker) checkIfWithNilNarrowing(node *ast.IfStatement, check nilCheck) {
	prevEnv := c.env

	thenType, elseType := check.nonNil, Type(Nil)
	if check.nilInThen {
		thenType, elseType = Nil, check.nonNil
	}

	c.env = NewEnclosedEnvironment(prevEnv)
	c.env.Narrow(check.varName, thenType)
	c.checkBlockStatement(node.Consequence)
	c.env = prevEnv

	if node.Alternative != nil {
		c.env = NewEnclosedEnvironment(prevEnv)
		c.env.Narrow(check.varName, elseType)
		c.checkBlockStatement(node.Alternative)
		c.env = prevEnv
	}

	// `if x == nil then return end`, and `if not x then x = default end`,
	// leave x non-nil for everything after the statement.
	nilBranch := node.Alternative
	if check.nilInThen {
		nilBranch = node.Consequence
	}
	if nilBranch != nil && (blockAlwaysExits(nilBranch) || blockAssigns(nilBranch, check.varName)) {
		c.env.Narrow(check.varName, check.nonNil)
	}
}

// blockAssigns reports whether a block assigns name a value other than nil
// as one of its own statements, as `if not x then x = default end` does.
func blockAssigns(block *ast.BlockStatement, name string) bool {
	for _, stmt := range block.Statements {
		as, ok := stmt.(*ast.AssignmentStatement)
		if !ok || len(as.Targets) != 1 || len(as.Values) != 1 {
			continue
		}
		if ident, ok := as.Targets[0].(*ast.Identifier); ok && ident.Value == name {
			if _, isNil := as.Values[0].(*ast.NilLiteral); !isNil {
				return true
			}
		}
	}
	return false
}

// blockAlwaysExits reports whether a block ends in a statement that leaves the
// enclosing function or loop, so the code after it only runs otherwise.
func blockAlwaysExits(block *ast.BlockStatement) bool {
	if block == nil || len(block.Statements) == 0 {
		return false
	}

	switch last := block.Statements[len(block.Statements)-1].(type) {
	case *ast.ReturnStatement, *ast.BreakStatement:
		return true
	case *ast.ExpressionStatement:
		// A call to error() does not return either.
		call, ok := last.Expression.(*ast.CallExpression)
		if !ok {
			return false
		}
		ident, ok := call.Function.(*ast.Identifier)
		return ok && ident.Value == "error"
	}

	return false
}

// extractTypeNarrowing checks if the expression is a type guard call or discriminated union check
// and extracts narrowing info. Returns the variable name and the narrowed type, or empty string and nil
func (c *Checker) extractTypeNarrowing(expr ast.Expression) (string, Type) {
	// Check if this is a call expression (type guard function)
	if callExpr, ok := expr.(*ast.CallExpression); ok && len(callExpr.Arguments) == 1 {
		// Get the function being called
		var funcType *FunctionType
		if ident, ok := callExpr.Function.(*ast.Identifier); ok {
			typ, exists := c.env.Get(ident.Value)
			if exists {
				funcType, ok = typ.(*FunctionType)
				if ok {
					// Check if return type is a type guard
					guardType, ok := funcType.ReturnType.(*TypeGuardType)
					if ok {
						// Extract the variable being guarded from the argument
						if argIdent, ok := callExpr.Arguments[0].(*ast.Identifier); ok {
							return argIdent.Value, guardType.GuardedType
						}
					}
				}
			}
		}
	}

	// Check if this is a discriminated union check (e.g., obj.status === "success")
	if infixExpr, ok := expr.(*ast.InfixExpression); ok {
		if infixExpr.Operator == "==" || infixExpr.Operator == "!=" {
			// Try both directions: prop === literal or literal === prop
			varName, narrowedType := c.extractDiscriminantNarrowing(infixExpr.Left, infixExpr.Right, infixExpr.Operator == "==")
			if varName != "" && narrowedType != nil {
				return varName, narrowedType
			}
			// Try reverse
			varName, narrowedType = c.extractDiscriminantNarrowing(infixExpr.Right, infixExpr.Left, infixExpr.Operator == "==")
			if varName != "" && narrowedType != nil {
				return varName, narrowedType
			}
		}
	}

	return "", nil
}

// extractDiscriminantNarrowing checks if we're comparing a property to a literal value
// and narrows a union type based on that discriminant property
func (c *Checker) extractDiscriminantNarrowing(propExpr, literalExpr ast.Expression, isEquality bool) (string, Type) {
	// Check if propExpr is a property access (IndexExpression with identifier base)
	indexExpr, ok := propExpr.(*ast.IndexExpression)
	if !ok {
		return "", nil
	}

	// Get the object being accessed (must be an identifier)
	objIdent, ok := indexExpr.Left.(*ast.Identifier)
	if !ok {
		return "", nil
	}

	// Get the property name (must be a string literal or identifier used as property)
	var propName string
	if stringLit, ok := indexExpr.Index.(*ast.StringLiteral); ok {
		propName = stringLit.Value
	} else if ident, ok := indexExpr.Index.(*ast.Identifier); ok {
		// Property access like obj.prop is parsed as obj[prop] with identifier
		propName = ident.Value
	} else {
		return "", nil
	}

	// Get the literal value being compared
	var literalType Type
	switch lit := literalExpr.(type) {
	case *ast.StringLiteral:
		literalType = &StringLiteralType{Value: lit.Value}
	case *ast.NumberLiteral:
		literalType = &NumberLiteralType{Value: lit.Value}
	case *ast.BooleanLiteral:
		literalType = &BooleanLiteralType{Value: lit.Value}
	default:
		return "", nil
	}

	// Get the type of the object
	objType, exists := c.env.Get(objIdent.Value)
	if !exists {
		return "", nil
	}

	// If it's a union type, narrow it based on the discriminant property
	unionType, ok := objType.(*UnionType)
	if !ok {
		return "", nil
	}

	// Filter union variants that match the discriminant
	narrowedTypes := []Type{}
	for _, variantType := range unionType.Types {
		// Check if this variant has the discriminant property with the matching value
		if c.variantMatchesDiscriminant(variantType, propName, literalType, isEquality) {
			narrowedTypes = append(narrowedTypes, variantType)
		}
	}

	// If no variants match, return nil (this will cause a type error later)
	if len(narrowedTypes) == 0 {
		return "", nil
	}

	// If only one variant matches, return that type directly
	if len(narrowedTypes) == 1 {
		return objIdent.Value, narrowedTypes[0]
	}

	// Multiple variants match, return a narrowed union
	return objIdent.Value, &UnionType{Types: narrowedTypes}
}

// variantMatchesDiscriminant checks if a type variant has a property with the expected value
func (c *Checker) variantMatchesDiscriminant(variantType Type, propName string, expectedValue Type, isEquality bool) bool {
	// Get the property type from the variant
	var propType Type

	switch v := variantType.(type) {
	case *InterfaceType:
		var exists bool
		propType, exists = v.Properties[propName]
		if !exists {
			return false
		}
	case *ClassType:
		// Check if the class has the property
		var exists bool
		propType, exists = v.Properties[propName]
		if !exists {
			return false
		}
	case *TableType:
		// For tables, we can't determine the specific property value
		return false
	default:
		return false
	}

	// Check if the property type matches the expected literal type
	if isEquality {
		// For ==, the property must equal the expected value
		return propType.Equals(expectedValue)
	} else {
		// For !=, the property must NOT equal the expected value
		return !propType.Equals(expectedValue)
	}
}

// checkWhileStatement checks a while statement
// checkRepeatStatement checks a repeat ... until loop. Lua scopes the
// condition inside the body, so names declared in the body are visible to it.
func (c *Checker) checkRepeatStatement(node *ast.RepeatStatement) {
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)

	if node.Body != nil {
		for _, stmt := range node.Body.Statements {
			c.checkStatement(stmt)
		}
	}

	condType := c.checkExpression(node.Condition)
	c.checkCondition(condType, "Repeat", node.Token)

	c.env = prevEnv
}

func (c *Checker) checkWhileStatement(node *ast.WhileStatement) {
	condType := c.checkExpression(node.Condition)
	c.checkCondition(condType, "While", node.Token)

	c.checkBlockStatement(node.Body)
}

// checkForStatement checks a for statement
// iterationTypes works out what a generic for loop's key and value variables
// hold. The iterator is usually ipairs(t) or pairs(t), whose declared return
// type says nothing, so the collection being passed is what gets inspected.
func (c *Checker) iterationTypes(iterator ast.Expression, iterType Type) (Type, Type) {
	collection := iterType

	if call, ok := iterator.(*ast.CallExpression); ok && len(call.Arguments) == 1 {
		if ident, ok := call.Function.(*ast.Identifier); ok {
			switch ident.Value {
			case "ipairs", "pairs":
				collection = c.checkExpression(call.Arguments[0])
			default:
				return Any, Any
			}
		}
	}

	switch typ := collection.(type) {
	case *ArrayType:
		return Number, typ.ElementType
	case *TableType:
		return typ.KeyType, typ.ValueType
	}

	return Any, Any
}

func (c *Checker) checkForStatement(node *ast.ForStatement) {
	// Create new scope for loop
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)

	if node.IsGeneric {
		// Generic for loop (for-in)
		// The expressions are checked outside the loop's scope: the loop
		// variables are not visible in them.
		c.env = prevEnv
		iterType := c.checkExpression(node.Iterator)
		for _, e := range node.IteratorRest {
			c.checkExpression(e)
		}
		c.env = NewEnclosedEnvironment(prevEnv)
		// A call yielding (f, s, c) iterates with f.
		if tuple, ok := iterType.(*TupleType); ok && len(tuple.Elements) > 0 {
			iterType = tuple.Elements[0]
		}

		// Give the loop variables the key and value types of whatever is being
		// iterated. Leaving them as 'any' means a method call on the element
		// cannot be resolved, and codegen then picks the wrong call syntax.
		keyType, valueType := c.iterationTypes(node.Iterator, iterType)
		vars := []Type{keyType, valueType}
		// An iterator function's results are the loop variables; the loop
		// ends at the first nil, so the first is never nil inside.
		if fn, ok := iterType.(*FunctionType); ok {
			vars = nil
			if results, isTuple := fn.ReturnType.(*TupleType); isTuple {
				vars = append(vars, results.Elements...)
			} else if fn.ReturnType != nil && !fn.ReturnType.Equals(Void) {
				vars = append(vars, fn.ReturnType)
			}
			if len(vars) > 0 {
				vars[0] = WithoutNil(vars[0])
			}
		}
		for i, variable := range node.Variables {
			typ := Type(Any)
			if i < len(vars) && vars[i] != nil {
				typ = vars[i]
			}
			c.env.Set(variable.Value, typ)
			c.define(variable.Token, SymVariable, typ)
		}

		// Lua calls what comes after 'in': a number, string or boolean
		// cannot be iterated.
		switch iterType.(type) {
		case *NumberType, *StringType, *BooleanType, *NumberLiteralType, *StringLiteralType, *BooleanLiteralType:
			c.addError(
				fmt.Sprintf("Cannot iterate over type '%s'", iterType.String()),
				node.Token,
			)
		}
	} else {
		for _, variable := range node.Variables {
			c.env.Set(variable.Value, Number)
			c.define(variable.Token, SymVariable, Number)
		}

		// Numeric for loop
		startType := c.checkExpression(node.Start)
		endType := c.checkExpression(node.End)

		if !IsNumericType(startType) && !startType.Equals(Any) {
			c.addError(
				fmt.Sprintf("For loop start must be number, got '%s'", startType.String()),
				node.Token,
			)
		}
		if !IsNumericType(endType) && !endType.Equals(Any) {
			c.addError(
				fmt.Sprintf("For loop end must be number, got '%s'", endType.String()),
				node.Token,
			)
		}

		if node.Step != nil {
			stepType := c.checkExpression(node.Step)
			if !IsNumericType(stepType) && !stepType.Equals(Any) {
				c.addError(
					fmt.Sprintf("For loop step must be number, got '%s'", stepType.String()),
					node.Token,
				)
			}
		}
	}

	c.checkBlockStatement(node.Body)
	c.env = prevEnv
}

// checkDoStatement checks a do statement
func (c *Checker) checkDoStatement(node *ast.DoStatement) {
	c.checkBlockStatement(node.Body)
}

// checkBlockStatement checks a block statement
func (c *Checker) checkBlockStatement(node *ast.BlockStatement) {
	if node == nil {
		return
	}

	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	// A local in a do, if or loop block is visible to the block's end.
	if node.EndToken.End > 0 {
		prevScopeEnd := c.scopeEnd
		c.scopeEnd = node.EndToken.End
		defer func() { c.scopeEnd = prevScopeEnd }()
	}

	// Types declared inside the block, usable throughout it.
	c.predeclareTypeNames(node.Statements)
	for _, stmt := range node.Statements {
		switch stmt.(type) {
		case *ast.InterfaceDeclaration, *ast.TypeDeclaration, *ast.EnumDeclaration, *ast.ClassDeclaration:
			c.registerTypeDefinition(stmt)
		}
	}

	for _, stmt := range node.Statements {
		c.checkStatement(stmt)
	}

	c.env = prevEnv
}

// checkAssignmentStatement checks an assignment statement, which may have
// several targets (a, b = b, a).
func (c *Checker) checkAssignmentStatement(node *ast.AssignmentStatement) {
	// One expression can fill several targets, as in `n, s = two()` where two()
	// returns (number, string).
	if len(node.Targets) > 1 && len(node.Values) == 1 {
		valueType := c.checkExpression(node.Values[0])

		if tuple, ok := valueType.(*TupleType); ok {
			for i, target := range node.Targets {
				var elem Type
				if i < len(tuple.Elements) {
					elem = tuple.Elements[i]
				}
				c.checkAssignmentTarget(node, target, elem)
			}
			return
		}

		// Anything else assigns to the first target and leaves the rest nil.
		for i, target := range node.Targets {
			if i == 0 {
				c.checkAssignmentTarget(node, target, valueType)
			} else {
				c.checkAssignmentTarget(node, target, nil)
			}
		}
		return
	}

	for i, target := range node.Targets {
		var valueType Type
		if i < len(node.Values) {
			valueType = c.checkExpression(node.Values[i])
		}
		c.checkAssignmentTarget(node, target, valueType)
	}

	// Values beyond the target count still need checking for their own errors.
	for i := len(node.Targets); i < len(node.Values); i++ {
		c.checkExpression(node.Values[i])
	}
}

// checkAssignmentTarget checks one target of an assignment. A nil valueType
// means the target is filled from a multi-value expression and is not checked.
func (c *Checker) checkAssignmentTarget(node *ast.AssignmentStatement, target ast.Expression, valueType Type) {
	// Assigning a name defined nowhere defines a global, as in Lua:
	// counter = 0.
	if ident, ok := target.(*ast.Identifier); ok {
		if _, found := c.env.Get(ident.Value); !found {
			root := c.env
			for root.outer != nil {
				root = root.outer
			}
			root.Set(ident.Value, Any)
			c.define(ident.Token, SymVariable, Any)
		}
	}

	// Check if trying to assign to a const variable
	if ident, ok := target.(*ast.Identifier); ok {
		if c.env.IsConst(ident.Value) {
			c.addError(
				fmt.Sprintf("Cannot assign to const variable '%s'", ident.Value),
				node.Token,
			)
			return
		}
	}

	// Check if trying to assign to a readonly property
	if dotExpr, ok := target.(*ast.DotExpression); ok {
		leftType := c.checkExpression(dotExpr.Left)
		if classType, ok := leftType.(*ClassType); ok {
			if rightIdent, ok := dotExpr.Right.(*ast.Identifier); ok {
				propName := rightIdent.Value
				if classType.IsReadonly(propName) {
					// Allow readonly assignment in constructor if assigning to self
					allowAssignment := false
					if c.inConstructor && c.currentClass != nil {
						// Check if we're assigning to self
						if leftIdent, ok := dotExpr.Left.(*ast.Identifier); ok {
							if leftIdent.Value == "self" && c.currentClass == classType {
								allowAssignment = true
							}
						}
					}

					if !allowAssignment {
						c.addError(
							fmt.Sprintf("Cannot assign to readonly property '%s'", propName),
							node.Token,
						)
						return
					}
				}
			}
		}
	}

	// Enum members are constants, and a property with only a getter cannot
	// be set (the assignment would hide the getter for good).
	if dot, ok := target.(*ast.DotExpression); ok {
		if field, ok := dot.Right.(*ast.Identifier); ok {
			switch owner := c.checkExpression(dot.Left).(type) {
			case *EnumType:
				c.addError(fmt.Sprintf("Cannot assign to enum member '%s'", field.Value), field.Token)
				return
			case *InterfaceType:
				if owner.ReadonlyProps[field.Value] {
					c.addError(fmt.Sprintf("Cannot assign to readonly property '%s'", field.Value), field.Token)
					return
				}
			case *ClassType:
				if _, isProp := owner.GetProperty(field.Value); !isProp {
					_, hasGetter := owner.GetGetter(field.Value)
					_, hasSetter := owner.GetSetter(field.Value)
					if hasGetter && !hasSetter {
						c.addError(fmt.Sprintf("'%s' has a getter but no setter, so it cannot be assigned", field.Value), field.Token)
						return
					}
				}
			}
		}
	}

	// Assigning a field an unannotated table literal lacks adds it, as in
	// Lua: local t = { a = 1 }; t.b = 2.
	if dot, ok := target.(*ast.DotExpression); ok && !dot.IsOptional {
		if field, ok := dot.Right.(*ast.Identifier); ok {
			leftType := c.checkExpression(dot.Left)
			// `local M = {}` then function M.add: the field is defined there.
			if arr, ok := leftType.(*ArrayType); ok && arr.EmptyLiteral && arr.MemberSyms[field.Value] == nil {
				t := Type(Any)
				if valueType != nil {
					t = widenLiteralType(valueType)
				}
				setMemberSym(&arr.MemberSyms, field.Value, c.defineMember(field.Token, SymProperty, t, exprText(dot.Left)))
			}
			if rec, ok := leftType.(*InterfaceType); ok && rec.Open {
				current, hasProp := rec.Properties[field.Value]
				_, hasMethod := rec.Methods[field.Value]
				// A field given a literal (n = 1) takes other values of its
				// kind (t.n = 2), as an unannotated local does.
				if hasProp && valueType != nil && isLiteralType(current) &&
					widenLiteralType(valueType).Equals(widenLiteralType(current)) {
					rec.Properties[field.Value] = widenLiteralType(current)
				}
				if !hasProp && !hasMethod {
					added := Type(Any)
					if valueType != nil {
						added = widenLiteralType(valueType)
					}
					rec.Properties[field.Value] = added
					// Defined here: M.add later goes to `function M.add`.
					setMemberSym(&rec.MemberSyms, field.Value, c.defineMember(field.Token, SymProperty, added, exprText(dot.Left)))
				}
			}
		}
	}

	targetType := c.checkExpression(target)
	// A variable takes values of the type it was declared with, whatever a
	// check has narrowed it to: `if not name then name = "x" end`.
	if ident, ok := target.(*ast.Identifier); ok {
		if declared, ok := c.env.DeclaredType(ident.Value); ok {
			targetType = declared
		}
	}
	if valueType == nil {
		return
	}

	if len(node.Targets) == len(node.Values) {
		for i, t := range node.Targets {
			if t == target && c.reportLiteralFields(node.Values[i], targetType) {
				return
			}
		}
	}

	// A local declared without a type can be given up, as Lua code does
	// to release what it held: cache = nil.
	if ident, ok := target.(*ast.Identifier); ok && c.env.IsInferred(ident.Value) {
		if _, isNil := valueType.(*NilType); isNil {
			return
		}
	}
	// `{}` is an empty table: it can be replaced by any other table, or nil.
	if arr, ok := targetType.(*ArrayType); ok && arr.EmptyLiteral {
		return
	}

	if !valueType.IsAssignableTo(targetType) {
		c.addError(
			fmt.Sprintf("Cannot assign type '%s' to type '%s'",
				valueType.String(), targetType.String()),
			node.Token,
		)
	}
}

// checkClassDeclaration checks a class declaration
func (c *Checker) checkClassDeclaration(node *ast.ClassDeclaration) {
	classType, ok := c.classes[node.Name.Value]
	if !ok {
		return
	}
	// The names after extends and implements are uses of those types.
	if node.Extends != nil {
		c.referenceTypeName(node.Extends)
	}
	for _, impl := range node.Implements {
		c.referenceTypeName(impl)
	}

	// The parent is needed when the class is set up, so a parent declared
	// further down the same file is still nil then.
	if parentName := classNameOf(node.Extends); parentName != "" {
		if sym := c.typeSyms[parentName]; sym != nil && sym.Decl.File == node.Token.File && sym.Decl.Offset > node.Token.Offset {
			c.addError(fmt.Sprintf("'%s' has to be declared before '%s', which extends it", parentName, node.Name.Value), node.Extends.(*ast.Identifier).Token)
		}
	}

	// Check decorators
	for _, decorator := range node.Decorators {
		c.checkDecorator(decorator)
	}

	// Check abstract methods: they should not have a body in abstract classes
	// For now, we'll allow abstract methods to have no body
	for _, method := range node.Methods {
		if method.IsAbstract {
			// Abstract methods shouldn't have implementation
			if method.Body != nil && len(method.Body.Statements) > 0 {
				c.addError(
					fmt.Sprintf("Abstract method '%s' should not have an implementation", method.Name.Value),
					method.Token,
				)
			}
			// Abstract methods can only exist in abstract classes
			if !node.IsAbstract {
				c.addError(
					fmt.Sprintf("Abstract method '%s' can only be declared in an abstract class", method.Name.Value),
					method.Token,
				)
			}
		}
	}

	// Check that non-abstract classes implement all abstract methods from parent
	if !node.IsAbstract && classType.Parent != nil {
		c.checkAbstractMethodsImplemented(classType, classType.Parent, node.Token)
	}

	// Check that method overrides have compatible signatures
	if classType.Parent != nil {
		c.checkMethodOverrides(classType, classType.Parent, node.Token)
	}

	// Check constructor if present
	if node.Constructor != nil {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		prevClass := c.currentClass
		c.env = NewEnclosedEnvironment(prevEnv)
		c.currentFunctionReturnType = Void
		c.currentClass = classType

		// Add generic type parameters to scope
		for _, genericParam := range node.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}

		// Add self to scope
		c.env.Set("self", classType)
		c.env.defineSym("self", selfSymbol(node.Name.Token, classType))

		// Add parameters to scope
		for _, param := range node.Constructor.Parameters {
			var paramType Type = Any
			if param.Type != nil {
				paramType = c.paramType(param)
			}
			c.env.Set(param.Name.Value, paramType)
			c.define(param.Name.Token, SymParameter, paramType)
		}

		// A subclass's constructor runs its parent's with super(...): the
		// object is built in place, so without it the parent's fields are
		// never set.
		if parent, ok := c.classes[classNameOf(node.Extends)]; ok && !parent.Ambient && node.Constructor.Body != nil {
			calls := false
			for _, stmt := range node.Constructor.Body.Statements {
				if es, ok := stmt.(*ast.ExpressionStatement); ok {
					if call, ok := es.Expression.(*ast.CallExpression); ok {
						if _, isSuper := call.Function.(*ast.SuperExpression); isSuper {
							calls = true
						}
					}
				}
			}
			if !calls {
				c.addError(fmt.Sprintf("The constructor of '%s' must call super(...) to run '%s''s constructor", node.Name.Value, parent.Name), node.Constructor.Token)
			}
		}

		// Check constructor body (allow readonly assignments in constructor)
		prevInConstructor := c.inConstructor
		c.inConstructor = true
		prevScopeEnd := c.scopeEnd
		if node.Constructor.Body != nil {
			c.scopeEnd = node.Constructor.Body.EndToken.End
		}
		c.checkBlockStatement(node.Constructor.Body)
		c.scopeEnd = prevScopeEnd
		c.inConstructor = prevInConstructor

		c.env = prevEnv
		c.currentFunctionReturnType = prevReturnType
		c.currentClass = prevClass
	}

	// Check methods
	for _, method := range node.Methods {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		prevClass := c.currentClass
		c.env = NewEnclosedEnvironment(prevEnv)
		c.currentClass = classType

		// Add generic type parameters to scope: the class's, and the
		// method's own (pass<U>).
		for _, genericParam := range node.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}
		for _, genericParam := range method.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}

		// Get method's return type
		var returnType Type = Void
		if method.ReturnType != nil {
			returnType = c.resolveTypeExpression(method.ReturnType)
		}
		c.currentFunctionReturnType = returnType

		// Add self to scope
		c.env.Set("self", classType)
		c.env.defineSym("self", selfSymbol(node.Name.Token, classType))

		// Add parameters to scope
		for _, param := range method.Parameters {
			var paramType Type = Any
			if param.Type != nil {
				paramType = c.paramType(param)
			}
			c.env.Set(param.Name.Value, paramType)
			c.define(param.Name.Token, SymParameter, paramType)
		}

		// Check method body
		prevScopeEnd := c.scopeEnd
		if method.Body != nil {
			c.scopeEnd = method.Body.EndToken.End
		}
		c.checkBlockStatement(method.Body)
		c.scopeEnd = prevScopeEnd

		c.env = prevEnv
		c.currentFunctionReturnType = prevReturnType
		c.currentClass = prevClass
	}

	// Check getters
	for _, getter := range node.Getters {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		prevClass := c.currentClass
		c.env = NewEnclosedEnvironment(prevEnv)
		c.currentClass = classType

		// Add generic type parameters to scope
		for _, genericParam := range node.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}

		// Get getter's return type
		var returnType Type = Any
		if getter.ReturnType != nil {
			returnType = c.resolveTypeExpression(getter.ReturnType)
		}
		c.currentFunctionReturnType = returnType

		// Add self to scope
		c.env.Set("self", classType)
		c.env.defineSym("self", selfSymbol(node.Name.Token, classType))

		// Check getter body
		c.checkBlockStatement(getter.Body)

		c.env = prevEnv
		c.currentFunctionReturnType = prevReturnType
		c.currentClass = prevClass
	}

	// Check setters
	for _, setter := range node.Setters {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		prevClass := c.currentClass
		c.env = NewEnclosedEnvironment(prevEnv)
		c.currentClass = classType

		// Add generic type parameters to scope
		for _, genericParam := range node.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}

		// Setters return void
		c.currentFunctionReturnType = Void

		// Add self to scope
		c.env.Set("self", classType)
		c.env.defineSym("self", selfSymbol(node.Name.Token, classType))

		// Add setter parameter to scope
		if setter.Parameter != nil {
			var paramType Type = Any
			if setter.Parameter.Type != nil {
				paramType = c.resolveTypeExpression(setter.Parameter.Type)
			}
			c.env.Set(setter.Parameter.Name.Value, paramType)
		}

		// Check setter body
		c.checkBlockStatement(setter.Body)

		c.env = prevEnv
		c.currentFunctionReturnType = prevReturnType
		c.currentClass = prevClass
	}

	// Check if class implements all interface methods
	for _, impl := range classType.Implements {
		c.checkClassImplementsInterface(classType, impl, node.Token)
	}
}

// checkClassImplementsInterface verifies a class implements an interface
func (c *Checker) checkClassImplementsInterface(class *ClassType, iface *InterfaceType, token lexer.Token) {
	// Check all interface methods are implemented
	for methodName, ifaceMethod := range iface.Methods {
		classMethod, ok := class.GetMethod(methodName)
		if !ok {
			c.addError(
				fmt.Sprintf("Class '%s' does not implement method '%s' from interface '%s'",
					class.Name, methodName, iface.Name),
				token,
			)
			continue
		}

		// Check method signature matches
		if !ifaceMethod.Equals(classMethod) {
			c.addError(
				fmt.Sprintf("Method '%s' in class '%s' has signature '%s' but interface '%s' requires '%s'",
					methodName, class.Name, classMethod.String(), iface.Name, ifaceMethod.String()),
				token,
			)
		}
	}

	// Check all interface properties are present
	for propName, ifaceProp := range iface.Properties {
		classProp, ok := class.GetProperty(propName)
		if !ok {
			// A function-typed property and a method are different things in Lua:
			// the property is a field called as f(...), the method takes an
			// implicit self and is called as obj:f(...). Say so, because the two
			// look identical at the call site.
			if _, isFunc := ifaceProp.(*FunctionType); isFunc {
				if _, hasMethod := class.GetMethod(propName); hasMethod {
					c.addError(
						fmt.Sprintf("Class '%s' implements '%s' as a method, but interface '%s' declares it as a function-typed property. "+
							"A property is called as %s(...) while a method takes an implicit self and is called as obj:%s(...). "+
							"Declare it in the interface as a method ('%s(...)') to match the class.",
							class.Name, propName, iface.Name, propName, propName, propName),
						token,
					)
					continue
				}
			}

			c.addError(
				fmt.Sprintf("Class '%s' does not implement property '%s' from interface '%s'",
					class.Name, propName, iface.Name),
				token,
			)
			continue
		}

		// Check property type matches
		if !classProp.Equals(ifaceProp) {
			c.addError(
				fmt.Sprintf("Property '%s' in class '%s' has type '%s' but interface '%s' requires '%s'",
					propName, class.Name, classProp.String(), iface.Name, ifaceProp.String()),
				token,
			)
		}
	}

	// Recursively check extended interfaces
	for _, ext := range iface.Extends {
		c.checkClassImplementsInterface(class, ext, token)
	}
}

// checkAbstractMethodsImplemented verifies that a class implements all abstract methods from its parent
func (c *Checker) checkAbstractMethodsImplemented(class *ClassType, parent *ClassType, token lexer.Token) {
	// Recursively check all ancestors
	if parent == nil {
		return
	}

	// Check this parent's abstract methods: the class, or any class between
	// it and this parent, has to implement each one.
	for methodName := range parent.AbstractMethods {
		implemented := false
		for cl := class; cl != nil && cl != parent; cl = cl.Parent {
			if _, ok := cl.Methods[methodName]; ok && !cl.AbstractMethods[methodName] {
				implemented = true
				break
			}
		}
		if !implemented {
			c.addError(
				fmt.Sprintf("Class '%s' must implement abstract method '%s' from parent class '%s'",
					class.Name, methodName, parent.Name),
				token,
			)
		}
	}

	// Check grandparent
	if parent.Parent != nil {
		c.checkAbstractMethodsImplemented(class, parent.Parent, token)
	}
}

// checkMethodOverrides verifies that overridden methods have compatible signatures
func (c *Checker) checkMethodOverrides(class *ClassType, parent *ClassType, token lexer.Token) {
	if parent == nil {
		return
	}

	// Check each method in the child class
	for methodName, childMethod := range class.Methods {
		// Check if parent has a method with the same name
		parentMethod, hasParentMethod := parent.GetMethod(methodName)
		if !hasParentMethod {
			continue // Not an override, skip
		}

		// Validate parameter count
		if len(childMethod.Parameters) != len(parentMethod.Parameters) {
			c.addError(
				fmt.Sprintf("Method '%s' override has %d parameters, but parent method has %d parameters",
					methodName, len(childMethod.Parameters), len(parentMethod.Parameters)),
				token,
			)
			continue
		}

		// Validate parameter types
		for i := 0; i < len(childMethod.Parameters); i++ {
			childParamType := childMethod.Parameters[i]
			parentParamType := parentMethod.Parameters[i]

			// Parameters should have the same type (contravariance not supported yet)
			if !childParamType.Equals(parentParamType) {
				c.addError(
					fmt.Sprintf("Method '%s' override parameter %d has type '%s', but parent method expects '%s'",
						methodName, i+1, childParamType.String(), parentParamType.String()),
					token,
				)
			}
		}

		// Validate return type (covariance: child can return more specific type)
		if !childMethod.ReturnType.IsAssignableTo(parentMethod.ReturnType) {
			c.addError(
				fmt.Sprintf("Method '%s' override has return type '%s', but parent method returns '%s'",
					methodName, childMethod.ReturnType.String(), parentMethod.ReturnType.String()),
				token,
			)
		}

		// Validate visibility (cannot be more restrictive)
		childVisibility := class.MethodVisibility[methodName]
		if childVisibility == "" {
			childVisibility = "public"
		}

		// Find which parent class defines this method to get its visibility
		var parentVisibility string
		current := parent
		for current != nil {
			if _, ok := current.Methods[methodName]; ok {
				parentVisibility = current.MethodVisibility[methodName]
				if parentVisibility == "" {
					parentVisibility = "public"
				}
				break
			}
			current = current.Parent
		}

		// Check if child is more restrictive than parent
		if !c.isVisibilityCompatible(parentVisibility, childVisibility) {
			c.addError(
				fmt.Sprintf("Method '%s' override cannot reduce visibility from %s to %s",
					methodName, parentVisibility, childVisibility),
				token,
			)
		}
	}

	// Recursively check grandparent
	if parent.Parent != nil {
		c.checkMethodOverrides(class, parent.Parent, token)
	}
}

// isVisibilityCompatible checks if child visibility is compatible with parent visibility
// Child can be same or less restrictive (public > protected > private)
func (c *Checker) isVisibilityCompatible(parentVis, childVis string) bool {
	visibilityLevel := map[string]int{
		"private":   1,
		"protected": 2,
		"public":    3,
	}

	parentLevel := visibilityLevel[parentVis]
	childLevel := visibilityLevel[childVis]

	// Child must be same or less restrictive (higher level)
	return childLevel >= parentLevel
}

// checkExpression checks an expression and returns its type
func (c *Checker) checkExpression(expr ast.Expression) Type {
	if isNilNode(expr) {
		return Void
	}
	t := c.checkExpressionKind(expr)
	if c.exprTypes == nil {
		c.exprTypes = make(map[ast.Expression]Type)
	}
	c.exprTypes[expr] = t
	return t
}

// memberNames are a member map's names in order, so that types built from
// them (keyof T, T[string]) print the same way on every run.
func memberNames[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// literalKeyName is the field a table-literal key names: a bare name, or a
// string in brackets (`["a"] = 1`). Other keys are computed.
func literalKeyName(key ast.Expression) (*ast.Identifier, bool) {
	switch k := key.(type) {
	case *ast.Identifier:
		return k, true
	case *ast.StringLiteral:
		return &ast.Identifier{Token: k.Token, Value: k.Value}, true
	}
	return nil, false
}

// checkFieldFunction checks `function M.f(...)` and `function t:m(...)`,
// which Lua defines as `M.f = function(...)` and `t.m = function(self,
// ...)`: the function is checked as that expression, then assigned to the
// field, so an unannotated table gains it and a declared one must have it.
func (c *Checker) checkFieldFunction(node *ast.FunctionDeclaration) {
	if node.IsLocal {
		c.addError("a local function has a plain name: local function f(), or M.f = function() ... end", node.Token)
	}
	params := node.Parameters
	if node.IsMethod {
		// The implicit self has no place in the source: given the method
		// name's token, hovering the name showed "(parameter) len".
		selfTok := lexer.Token{Type: lexer.IDENT, Literal: "self", File: node.Name.Token.File, Offset: -1, End: -1}
		self := &ast.Parameter{Token: selfTok, Name: &ast.Identifier{Token: selfTok, Value: "self"}}
		params = append([]*ast.Parameter{self}, params...)
	}
	fnType := c.checkFunctionExpression(&ast.FunctionExpression{Token: node.Token, GenericParams: node.GenericParams,
		Parameters: params, ReturnType: node.ReturnType, Body: node.Body})
	if fn, ok := fnType.(*FunctionType); ok {
		fn.ParamNames = parameterNames(params)
	}
	target := &ast.DotExpression{Token: node.Name.Token, Left: node.Receiver, Right: node.Name}
	c.checkAssignmentTarget(&ast.AssignmentStatement{Token: node.Name.Token, Targets: []ast.Expression{target}},
		target, fnType)
}

// referenceTypeName records a use of the class or interface a name in an
// extends or implements clause names (Base, or Base<T>).
func (c *Checker) referenceTypeName(e ast.Expression) {
	switch n := e.(type) {
	case *ast.Identifier:
		c.reference(n.Token, c.typeSyms[n.Value])
	case *ast.GenericType:
		c.referenceTypeName(n.BaseType)
	}
}

// classNameOf is the name in an extends clause, or "".
func classNameOf(e ast.Expression) string {
	if ident, ok := e.(*ast.Identifier); ok {
		return ident.Value
	}
	return ""
}

// paramType is a parameter's annotated type; `name?: T` may be left out,
// so in the body, and to a caller passing nil, it is T or nil.
func (c *Checker) paramType(param *ast.Parameter) Type {
	t := c.resolveTypeExpression(param.Type)
	if param.IsOptional && !param.IsRest {
		return withNil(t)
	}
	return t
}

// WithoutNil is t with nil taken out, for showing an optional parameter
// as `name?: T` rather than `name?: T | nil`.
func WithoutNil(t Type) Type {
	if base, ok := removeNil(t); ok {
		return base
	}
	return t
}

// eraseTypeParams is fn with its unbound type parameters taken as any.
func eraseTypeParams(fn *FunctionType) *FunctionType {
	erased := *fn
	erased.Parameters = make([]Type, len(fn.Parameters))
	for i, p := range fn.Parameters {
		erased.Parameters[i] = eraseTypeParam(p)
	}
	return &erased
}

func eraseTypeParam(t Type) Type {
	switch typ := t.(type) {
	case *GenericParamType:
		return Any
	case *ArrayType:
		return &ArrayType{ElementType: eraseTypeParam(typ.ElementType)}
	case *OptionalType:
		return &OptionalType{BaseType: eraseTypeParam(typ.BaseType)}
	case *UnionType:
		members := make([]Type, len(typ.Types))
		for i, m := range typ.Types {
			members[i] = eraseTypeParam(m)
		}
		return &UnionType{Types: members}
	}
	return t
}

// inferTypeArgs matches a parameter's type against an argument's and binds
// the type parameters it finds there that are not bound yet.
func inferTypeArgs(param, arg Type, names []string, args []Type, inferred []bool) {
	if arg == nil {
		return
	}
	switch p := param.(type) {
	case *GenericParamType:
		for j, name := range names {
			if name == p.Name && !inferred[j] {
				args[j], inferred[j] = widenLiteralType(arg), true
			}
		}
	case *ArrayType:
		if a, ok := arg.(*ArrayType); ok {
			inferTypeArgs(p.ElementType, a.ElementType, names, args, inferred)
		}
	case *TableType:
		if a, ok := arg.(*TableType); ok {
			inferTypeArgs(p.KeyType, a.KeyType, names, args, inferred)
			inferTypeArgs(p.ValueType, a.ValueType, names, args, inferred)
		}
	case *OptionalType:
		inferTypeArgs(p.BaseType, WithoutNil(arg), names, args, inferred)
	case *UnionType:
		if base, ok := removeNil(p); ok {
			inferTypeArgs(base, WithoutNil(arg), names, args, inferred)
		}
	case *FunctionType:
		if a, ok := arg.(*FunctionType); ok {
			for i := range p.Parameters {
				if i < len(a.Parameters) {
					inferTypeArgs(p.Parameters[i], a.Parameters[i], names, args, inferred)
				}
			}
			inferTypeArgs(p.ReturnType, a.ReturnType, names, args, inferred)
		}
	}
}

// typeMethodSelf annotates an untyped first parameter named self, in each
// function a literal gives for one of iface's methods, with the type the
// literal is written for (annotation).
func typeMethodSelf(lit *ast.TableLiteral, iface *InterfaceType, annotation ast.Expression) {
	for key, value := range lit.Pairs {
		name, ok := key.(*ast.Identifier)
		if !ok {
			continue
		}
		if _, isMethod := iface.methodType(name.Value); !isMethod {
			continue
		}
		fn, ok := value.(*ast.FunctionExpression)
		if !ok || len(fn.Parameters) == 0 {
			continue
		}
		if self := fn.Parameters[0]; self.Name != nil && self.Name.Value == "self" && self.Type == nil {
			self.Type = annotation
		}
	}
}

// qualifiedOwner is the namespace a qualified type name's left side names.
func (c *Checker) qualifiedOwner(e ast.Expression) *NamespaceType {
	switch n := e.(type) {
	case *ast.Identifier:
		if t, ok := c.env.Get(n.Value); ok {
			ns, _ := t.(*NamespaceType)
			return ns
		}
	case *ast.DotExpression:
		if outer := c.qualifiedOwner(n.Left); outer != nil {
			if name, ok := n.Right.(*ast.Identifier); ok {
				ns, _ := outer.Members[name.Value].(*NamespaceType)
				return ns
			}
		}
	}
	return nil
}

func typeExprText(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Value
	case *ast.DotExpression:
		return typeExprText(n.Left) + "." + typeExprText(n.Right)
	}
	return "?"
}

// withoutFirstParameter is fn as called with ':', which supplies the first
// argument itself.
func withoutFirstParameter(fn *FunctionType) *FunctionType {
	// function(...) called with ':' gets the table as one of its varargs;
	// the vararg parameter stays.
	if fn.HasRestParameter && len(fn.Parameters) == 1 {
		return fn
	}
	rest := *fn
	rest.Parameters = fn.Parameters[1:]
	if len(fn.OptionalParams) > 0 {
		rest.OptionalParams = fn.OptionalParams[1:]
	}
	if len(fn.ParamNames) > 0 {
		rest.ParamNames = fn.ParamNames[1:]
	}
	return &rest
}

// isNilNode reports whether a node is absent, including a nil pointer of a
// concrete node type inside the interface, as a failed parse leaves.
func isNilNode(n any) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

// checkExpressionKind computes an expression's type; checkExpression records
// the result so later checks can ask for it without checking twice.
func (c *Checker) checkExpressionKind(expr ast.Expression) Type {
	switch node := expr.(type) {
	case *ast.Identifier:
		return c.checkIdentifier(node)
	case *ast.SuperExpression:
		return c.checkSuperExpression(node)
	case *ast.NumberLiteral:
		// Number literals infer as literal types for precision
		return &NumberLiteralType{Value: node.Value}
	case *ast.StringLiteral:
		// String literals infer as literal types for precision
		return &StringLiteralType{Value: node.Value}
	case *ast.TemplateLiteral:
		return c.checkTemplateLiteral(node)
	case *ast.ParenExpression:
		return c.checkExpression(node.Expression)
	case *ast.BooleanLiteral:
		// Boolean literals infer as literal types for discriminated unions
		return &BooleanLiteralType{Value: node.Value}
	case *ast.NilLiteral:
		return Nil
	case *ast.TableLiteral:
		return c.checkTableLiteral(node)
	case *ast.FunctionExpression:
		return c.checkFunctionExpression(node)
	case *ast.PrefixExpression:
		return c.checkPrefixExpression(node)
	case *ast.InfixExpression:
		return c.checkInfixExpression(node)
	case *ast.CallExpression:
		result := c.checkCallExpression(node)
		// (f()) is f's first value only.
		if tuple, ok := result.(*TupleType); ok && node.Parenthesized {
			if len(tuple.Elements) == 0 {
				return Nil
			}
			return tuple.Elements[0]
		}
		return result
	case *ast.DotExpression:
		return c.checkDotExpression(node)
	case *ast.IndexExpression:
		return c.checkIndexExpression(node)
	case *ast.GenericType:
		return c.checkGenericInstantiation(node)
	case *ast.SpreadExpression:
		return c.checkSpreadExpression(node)
	case *ast.VarargExpression:
		// Lua's '...' expands to a value list whose contents are not tracked.
		return Any
	case *ast.TypeAssertion:
		return c.checkTypeAssertion(node)
	case *ast.MatchExpression:
		return c.checkMatchExpression(node)
	case *ast.AwaitExpression:
		return c.checkAwaitExpression(node)
	default:
		return Any
	}
}

// checkIdentifier checks an identifier and returns its type
func (c *Checker) checkIdentifier(node *ast.Identifier) Type {
	typ, ok := c.env.Get(node.Value)
	// super outside a class is an ordinary name, so normally undefined.
	if !ok && node.Value == "super" {
		c.addError("'super' can only be used inside a class", node.Token)
		return Any
	}
	if ok && c.ambient[node.Value] == typ {
		c.globalUses = append(c.globalUses, node.Token)
	}
	if ok {
		c.reference(node.Token, c.env.lookupSym(node.Value))
	}
	if !ok {
		// Build error message with suggestions
		errorMsg := fmt.Sprintf("Undefined variable '%s'", node.Value)
		if later, ok := c.laterLocals[node.Value]; ok && later.File == node.Token.File && later.Offset > node.Token.Offset {
			c.addError(fmt.Sprintf("'%s' is declared further down (line %d) as a local, so it is not in scope here: declare it before this use, or write `local %s` first",
				node.Value, later.Line, node.Value), node.Token)
			return Any
		}

		// Collect all available names from environment, classes, interfaces, etc.
		candidates := c.env.GetAllNames()

		// Add class names
		for className := range c.classes {
			candidates = append(candidates, className)
		}

		// Add interface names
		for interfaceName := range c.interfaces {
			candidates = append(candidates, interfaceName)
		}

		// Add enum names
		for enumName := range c.enums {
			candidates = append(candidates, enumName)
		}

		// Add type alias names
		for aliasName := range c.typeAliases {
			candidates = append(candidates, aliasName)
		}

		// Find similar names
		suggestions := findSimilarNames(node.Value, candidates, 3, 3)

		// Add suggestions to error message
		if len(suggestions) > 0 {
			errorMsg += ". Did you mean"
			if len(suggestions) == 1 {
				errorMsg += fmt.Sprintf(" '%s'?", suggestions[0])
			} else {
				errorMsg += ":"
				for i, suggestion := range suggestions {
					if i == len(suggestions)-1 {
						errorMsg += fmt.Sprintf(" or '%s'?", suggestion)
					} else {
						errorMsg += fmt.Sprintf(" '%s',", suggestion)
					}
				}
			}
		}

		c.addError(errorMsg, node.Token)
		return Any
	}
	return typ
}

// checkSuperExpression checks a super expression
func (c *Checker) checkSuperExpression(node *ast.SuperExpression) Type {
	// Super can only be used inside a class context
	if c.currentClass == nil {
		c.addError("'super' can only be used inside a class", node.Token)
		return Any
	}

	// Class must have a parent
	if c.currentClass.Parent == nil {
		c.addError(
			fmt.Sprintf("Class '%s' has no parent class, cannot use 'super'", c.currentClass.Name),
			node.Token,
		)
		return Any
	}

	// Return the parent class type
	return c.currentClass.Parent
}

// checkTableLiteral checks a table literal
func (c *Checker) checkTableLiteral(node *ast.TableLiteral) Type {
	// Check if this is a record-like table (all keys are string identifiers)
	if len(node.Values) == 0 && len(node.Pairs) > 0 {
		properties := make(map[string]Type)
		methods := make(map[string]*FunctionType)
		keySyms := make(map[string]lexer.Token)
		isRecord := true

		for key, value := range node.Pairs {
			// Check if key is an identifier (field name), or a string in
			// brackets, which names a field the same way.
			if ident, ok := literalKeyName(key); ok {
				// A function in a table literal is a plain field, called
				// with '.', as in `local M = { greet = function() end }`.
				// (Methods, called with ':', are declared by interfaces
				// and classes.)
				properties[ident.Value] = c.checkExpression(value)
				keySyms[ident.Value] = ident.Token
			} else {
				// Not a simple identifier key, treat as regular table
				isRecord = false
				break
			}
		}

		// If all keys are identifiers, create a structural interface type
		if isRecord {
			rec := &InterfaceType{
				Name:       "<table literal>",
				Properties: properties,
				Methods:    methods,
				Extends:    []*InterfaceType{},
				Open:       true,
			}
			// Each key defines its field: cfg.from goes to `from = ...`.
			for name, tok := range keySyms {
				setMemberSym(&rec.MemberSyms, name, c.defineMember(tok, SymProperty, properties[name], "<table>"))
			}
			return rec
		}
	}

	// A table built only from spreads of records is itself a record: the merge
	// of their fields, with later spreads winning, as in {...defaults, ...opts}.
	if len(node.Pairs) == 0 && len(node.Values) > 0 {
		if merged := c.mergeSpreadRecords(node.Values); merged != nil {
			return merged
		}
	}

	// Check if this is an array-style table (has sequential Values, or empty with no Pairs)
	if len(node.Pairs) == 0 {
		// Infer element type from the values: each distinct type once, in
		// the order the values first show it.
		var types []Type
		seen := make(map[string]bool)
		for _, value := range node.Values {
			valueType := c.checkExpression(value)
			// If it's a spread expression, unwrap the array type
			if _, ok := value.(*ast.SpreadExpression); ok {
				if arrayType, isArray := valueType.(*ArrayType); isArray {
					valueType = arrayType.ElementType
				}
			}
			if key := valueType.String(); !seen[key] {
				seen[key] = true
				types = append(types, valueType)
			}
		}

		// If all elements have the same type, use that type
		if len(types) == 1 {
			return &ArrayType{ElementType: types[0]}
		}

		// If elements have different types, create a union type; one of
		// them any makes the element any.
		if len(types) > 1 {
			for _, t := range types {
				if t.Equals(Any) {
					return &ArrayType{ElementType: Any}
				}
			}
			return &ArrayType{ElementType: &UnionType{Types: types}}
		}

		// Empty array defaults to any[]
		return &ArrayType{ElementType: Any, EmptyLiteral: true}
	}

	// For mixed or key-value tables, return a generic table type
	return &TableType{KeyType: Any, ValueType: Any}
}

// mergeSpreadRecords merges the fields of table values that are all spreads of
// record-like tables. It returns nil when the values are anything else, leaving
// the caller to infer an array type.
func (c *Checker) mergeSpreadRecords(values []ast.Expression) Type {
	properties := make(map[string]Type)
	methods := make(map[string]*FunctionType)

	for _, value := range values {
		spread, ok := value.(*ast.SpreadExpression)
		if !ok {
			return nil
		}

		record, ok := c.checkExpression(spread.Value).(*InterfaceType)
		if !ok {
			return nil
		}

		for name, propType := range record.Properties {
			properties[name] = propType
		}
		for name, methodType := range record.Methods {
			methods[name] = methodType
		}
	}

	return &InterfaceType{
		Name:       "<table literal>",
		Properties: properties,
		Methods:    methods,
		Extends:    []*InterfaceType{},
		Open:       true,
	}
}

// checkFunctionExpression checks an anonymous function expression
func (c *Checker) checkFunctionExpression(node *ast.FunctionExpression) Type {
	// Create new environment for function scope
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)

	// Resolve parameter types and add to environment
	paramTypes := make([]Type, len(node.Parameters))
	hasRestParam := false
	optionalParams := make([]bool, len(node.Parameters))
	for i, param := range node.Parameters {
		paramType := c.paramType(param)
		if param.IsRest {
			hasRestParam = true
			if _, ok := paramType.(*ArrayType); !ok {
				paramType = &ArrayType{ElementType: paramType}
			}
		}
		optionalParams[i] = paramIsOptional(param, paramType)
		paramTypes[i] = paramType
		if param.Name != nil {
			c.env.Set(param.Name.Value, paramType)
			c.define(param.Name.Token, SymParameter, paramType)
		}
	}

	// Resolve return type. Without an annotation it is inferred from the
	// body's own returns, as for a function declaration; a function nested
	// in this one collects its returns for itself.
	var returnType Type = Void
	prevInferring, prevInferred := c.inferringReturn, c.inferredReturns
	c.inferringReturn = node.ReturnType == nil
	c.inferredReturns = nil
	if node.ReturnType != nil {
		returnType = c.resolveTypeExpression(node.ReturnType)
	}

	// Set currentFunctionReturnType for return statement checking
	prevReturnType := c.currentFunctionReturnType
	c.currentFunctionReturnType = returnType

	// Type check the body
	prevScopeEnd := c.scopeEnd
	if node.Body != nil {
		c.scopeEnd = node.Body.EndToken.End
	}
	c.checkBlockStatement(node.Body)
	c.scopeEnd = prevScopeEnd
	if c.inferringReturn {
		returnType = combineReturnTypes(c.inferredReturns)
		if _, isUnion := returnType.(*UnionType); isUnion && untypedSignature(node.Parameters, node.ReturnType, 0) {
			returnType = Any
		}
	}
	c.inferringReturn, c.inferredReturns = prevInferring, prevInferred

	// Restore environment and return type
	c.env = prevEnv
	c.currentFunctionReturnType = prevReturnType

	// Return the function type
	return &FunctionType{
		Parameters:       paramTypes,
		ReturnType:       returnType,
		GenericParams:    []string{}, // Function expressions are not generic
		HasRestParameter: hasRestParam,
		OptionalParams:   optionalParams,
		Untyped:          untypedSignature(node.Parameters, node.ReturnType, 0),
	}
}

// checkPrefixExpression checks a prefix expression
func (c *Checker) checkPrefixExpression(node *ast.PrefixExpression) Type {
	rightType := c.checkExpression(node.Right)
	if hasMetatable(rightType) && node.Operator != "not" {
		return Any
	}
	if node.Operator != "not" && c.usesUnknown(rightType, node.Token) {
		return Any
	}

	switch node.Operator {
	case "-":
		if !IsNumericType(rightType) && !rightType.Equals(Any) {
			c.addError(
				fmt.Sprintf("Unary operator '-' cannot be applied to type '%s'", rightType.String()),
				node.Token,
			)
		}
		return Number
	case "not":
		return Boolean
	default:
		return Any
	}
}

// checkLogical checks a and b / a or b. In Lua they return one of their
// operands: a or b is a when a is truthy, otherwise b; a and b is a when
// a is falsy, otherwise b. The right side is checked knowing how the left
// turned out, as an if statement would: `a and a * 2`, `x ~= nil and x.y`.
func (c *Checker) checkLogical(node *ast.InfixExpression, isAnd bool) Type {
	leftType := c.checkExpression(node.Left)

	// The right side runs where the left is truthy (and) or falsy (or).
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	c.applyFacts(c.narrowings(node.Left, isAnd))
	rightType := c.checkExpression(node.Right)
	c.env = prevEnv

	if isAnyType(leftType) || isAnyType(rightType) {
		return Any
	}
	if isAnd {
		falsy := falsyPart(leftType)
		if falsy == nil {
			return rightType // the left is never falsy
		}
		return unionOf(falsy, rightType)
	}
	truthy := truthyPart(leftType)
	if truthy == nil {
		return rightType // the left is always nil or false
	}
	if falsyPart(leftType) == nil {
		return leftType // the left is never falsy: b is never reached
	}
	return unionOf(truthy, rightType)
}

func isAnyType(t Type) bool {
	_, ok := t.(*AnyType)
	return ok
}

// truthyPart is what of t is neither nil nor false (nil if nothing is).
func truthyPart(t Type) Type {
	switch typ := t.(type) {
	case *NilType, *VoidType:
		return nil
	case *BooleanLiteralType:
		if typ.Value {
			return typ
		}
		return nil
	case *BooleanType:
		return &BooleanLiteralType{Value: true}
	case *OptionalType:
		return truthyPart(typ.BaseType)
	case *UnionType:
		var parts []Type
		for _, m := range typ.Types {
			if part := truthyPart(m); part != nil {
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return unionOf(parts...)
	}
	return t
}

// falsyPart is what of t is nil or false (nil if nothing is).
func falsyPart(t Type) Type {
	switch typ := t.(type) {
	case *NilType, *VoidType:
		return Nil
	case *BooleanLiteralType:
		if typ.Value {
			return nil
		}
		return typ
	case *BooleanType:
		return &BooleanLiteralType{Value: false}
	case *OptionalType:
		if part := falsyPart(typ.BaseType); part != nil {
			return unionOf(Nil, part)
		}
		return Nil
	case *UnionType:
		var parts []Type
		for _, m := range typ.Types {
			if part := falsyPart(m); part != nil {
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return unionOf(parts...)
	case *UnknownType:
		return t
	}
	return nil
}

// unionOf joins types into one, flattening unions and dropping repeats;
// true | false is boolean.
func unionOf(types ...Type) Type {
	var members []Type
	seen := map[string]bool{}
	var add func(Type)
	add = func(t Type) {
		if u, ok := t.(*UnionType); ok {
			for _, m := range u.Types {
				add(m)
			}
			return
		}
		if opt, ok := t.(*OptionalType); ok {
			add(opt.BaseType)
			add(Nil)
			return
		}
		if key := t.String(); !seen[key] {
			seen[key] = true
			members = append(members, t)
		}
	}
	for _, t := range types {
		add(t)
	}
	if seen["true"] && seen["false"] {
		kept := members[:0]
		for _, m := range members {
			if lit, ok := m.(*BooleanLiteralType); ok {
				if lit.Value {
					kept = append(kept, Boolean)
				}
				continue
			}
			kept = append(kept, m)
		}
		members = kept
	}
	// A literal beside its own base type adds nothing: string | "x" is
	// string.
	kept := members[:0]
	for _, m := range members {
		switch m.(type) {
		case *BooleanLiteralType:
			if seen["boolean"] {
				continue
			}
		case *StringLiteralType:
			if seen["string"] {
				continue
			}
		case *NumberLiteralType:
			if seen["number"] {
				continue
			}
		}
		kept = append(kept, m)
	}
	members = kept
	if len(members) == 1 {
		return members[0]
	}
	return &UnionType{Types: members}
}

// checkMatchExpression types a match: its value is what its arms give,
// and each arm's bindings have the matched value's type (or the type a
// type pattern names).
func (c *Checker) checkMatchExpression(node *ast.MatchExpression) Type {
	valueType := c.checkExpression(node.Value)
	var results []Type
	for _, mc := range node.Cases {
		prev := c.env
		c.env = NewEnclosedEnvironment(prev)
		c.checkPattern(mc.Pattern, valueType)
		if mc.Guard != nil {
			c.checkExpression(mc.Guard)
		}
		if mc.Body != nil {
			results = append(results, c.checkExpression(mc.Body))
		}
		c.env = prev
	}
	if len(results) == 0 {
		return Any
	}
	return combineReturnTypes(results)
}

// checkPattern binds what a pattern binds, and refuses a literal pattern
// that a value of type t can never equal.
func (c *Checker) checkPattern(pattern ast.Pattern, t Type) {
	switch pt := pattern.(type) {
	case *ast.BindingPattern:
		c.env.Set(pt.Name, t)
		c.define(pt.Token, SymVariable, t)
	case *ast.LiteralPattern:
		lit := c.checkExpression(pt.Value)
		if !patternCanMatch(lit, t) {
			c.addError(fmt.Sprintf("Pattern %s can never match a value of type '%s'", pt.Value.String(), t.String()), pt.Token)
		}
	case *ast.ValuePattern:
		c.checkExpression(pt.Value)
	case *ast.OrPattern:
		for _, alt := range pt.Alternatives {
			c.checkPattern(alt, t)
		}
	case *ast.TypePattern:
		if pt.Binding == "" {
			return
		}
		var bound Type = Any
		switch pt.TypeName {
		case "number":
			bound = Number
		case "string":
			bound = String
		case "boolean":
			bound = Boolean
		default:
			if iface, ok := c.interfaces[pt.TypeName]; ok {
				bound = iface
			} else if cl, ok := c.classes[pt.TypeName]; ok {
				bound = cl
			}
		}
		c.env.Set(pt.Binding, bound)
	case *ast.StructPattern:
		for name, field := range pt.Fields {
			ft := Type(Any)
			if iface, ok := t.(*InterfaceType); ok {
				if p, ok := iface.GetProperty(name); ok {
					ft = p
				}
			}
			c.checkPattern(field, ft)
		}
	}
}

// patternCanMatch says whether a literal of type lit can equal a value of
// type t: "a" never equals a number.
func patternCanMatch(lit, t Type) bool {
	switch t.(type) {
	case *AnyType, *UnknownType, *EnumType:
		return true
	}
	if _, isNil := lit.(*NilType); isNil {
		return true // nil is how a missing value is matched
	}
	return lit.IsAssignableTo(t) || widenLiteralType(lit).IsAssignableTo(widenLiteralType(t))
}

// usesUnknown reports using a value of type unknown before narrowing it,
// which is what unknown is for, and says whether it did.
func (c *Checker) usesUnknown(t Type, tok lexer.Token) bool {
	if _, ok := t.(*UnknownType); !ok {
		return false
	}
	c.addErrorCode("unknown", "A value of type 'unknown' has to be narrowed before it is used: check it first (type(v) == \"table\", v ~= nil), or assert a type with 'as'", tok)
	return true
}

// hasMetatable says whether t is a table given a metatable, whose
// operators and calls come from it.
func hasMetatable(t Type) bool {
	rec, ok := t.(*InterfaceType)
	return ok && rec.Metatable
}

// checkInfixExpression checks an infix expression
func (c *Checker) checkInfixExpression(node *ast.InfixExpression) Type {
	switch node.Operator {
	case "and", "&&":
		return c.checkLogical(node, true)
	case "or", "||":
		return c.checkLogical(node, false)
	}
	leftType := c.checkExpression(node.Left)
	rightType := c.checkExpression(node.Right)
	if hasMetatable(leftType) || hasMetatable(rightType) {
		return Any
	}
	// == and ~= compare anything; every other operator needs to know.
	if node.Operator != "==" && node.Operator != "~=" && node.Operator != "!=" {
		if c.usesUnknown(leftType, node.Token) || c.usesUnknown(rightType, node.Token) {
			return Any
		}
	}

	switch node.Operator {
	case "+", "-", "*", "/", "%", "^":
		// Arithmetic operators require numbers
		if !IsNumericType(leftType) && !leftType.Equals(Any) {
			c.addError(
				fmt.Sprintf("Operator '%s' cannot be applied to type '%s'", node.Operator, leftType.String()),
				node.Token,
			)
		}
		if !IsNumericType(rightType) && !rightType.Equals(Any) {
			c.addError(
				fmt.Sprintf("Operator '%s' cannot be applied to type '%s'", node.Operator, rightType.String()),
				node.Token,
			)
		}
		return Number

	case "==", "!=", "<", "<=", ">", ">=":
		// Comparison operators return boolean
		return Boolean

	case "and", "or":
		// Logical operators return boolean
		return Boolean

	case "..":
		// String concatenation
		return String

	case "??":
		// Nullish coalescing - returns right value if left is nil
		// If left is T?, result is T | R (where R is right type)
		// In practice, if left is optional, unwrap it and union with right
		if _, ok := leftType.(*OptionalType); ok {
			// Left is optional, so result is the unwrapped left type OR right type
			// For simplicity, return right type (as it's the fallback)
			return rightType
		}
		// If left is not optional, ?? still works (returns left if not nil, otherwise right)
		// Result type is left | right, but for simplicity return right type
		return rightType

	default:
		return Any
	}
}

// checkCallExpression checks a function call
func (c *Checker) checkCallExpression(node *ast.CallExpression) Type {
	c.callee = node.Function
	funcType := c.checkExpression(node.Function)

	// setmetatable(Stack, mt) can give Stack __call, __add and the rest.
	if ident, ok := node.Function.(*ast.Identifier); ok && ident.Value == "setmetatable" && len(node.Arguments) > 0 {
		if arg, ok := node.Arguments[0].(*ast.Identifier); ok {
			if t, found := c.env.Get(arg.Value); found {
				if rec, ok := t.(*InterfaceType); ok && rec.Open {
					rec.Metatable = true
				}
			}
		}
	}

	// s.upper() is Lunar for s:upper(), but plain Lua's s.f(...) is
	// string.f(...) and stays as written: when the string is passed as its
	// own first argument (s.sub(s, 2, 3)), or when the arguments already
	// fill every parameter, leaving no room for the receiver
	// (s.upper("xyz") is string.upper("xyz")).
	if dot, ok := node.Function.(*ast.DotExpression); ok && !dot.IsMethodCall {
		if lib := c.stringDots[dot]; lib != nil {
			explicitSelf := len(node.Arguments) > 0 && exprText(node.Arguments[0]) == exprText(dot.Left)
			full := !lib.HasRestParameter && len(node.Arguments) >= len(lib.Parameters)
			if explicitSelf || full {
				dot.MethodDispatch = ast.DispatchPlain
			}
		}
	}

	// obj:f(x) on a function stored in a field passes obj as f's first
	// argument, so the arguments written fill the parameters after it.
	if dot, ok := node.Function.(*ast.DotExpression); ok && dot.IsMethodCall && dot.MethodDispatch == ast.DispatchPlain {
		if fn, ok := funcType.(*FunctionType); ok && len(fn.Parameters) > 0 {
			funcType = withoutFirstParameter(fn)
		}
	}

	// An optional call tolerates a missing callee, so unwrap the optional before
	// checking the signature and make the result optional in turn.
	if node.IsOptional {
		if optType, ok := funcType.(*OptionalType); ok {
			funcType = optType.BaseType
		}
		return &OptionalType{BaseType: c.checkCallWithType(node, funcType)}
	}

	// e?.hit(1): the method of an optional receiver is called when the
	// receiver is there, and the call yields nil otherwise.
	if opt, ok := funcType.(*OptionalType); ok && inOptionalChain(node.Function) {
		if _, isFn := opt.BaseType.(*FunctionType); isFn {
			return withNil(c.checkCallWithType(node, opt.BaseType))
		}
	}
	return c.checkCallWithType(node, funcType)
}

// inOptionalChain says whether a member chain has a ?. or ?[ link.
func inOptionalChain(e ast.Expression) bool {
	switch n := e.(type) {
	case *ast.DotExpression:
		return n.IsOptional || inOptionalChain(n.Left)
	case *ast.IndexExpression:
		return n.IsOptional || inOptionalChain(n.Left)
	case *ast.CallExpression:
		return n.IsOptional || inOptionalChain(n.Function)
	}
	return false
}

// checkCallWithType checks a call whose callee type has already been resolved.
func (c *Checker) checkCallWithType(node *ast.CallExpression, funcType Type) Type {

	// Check if it's a class type (constructor call)
	if classType, ok := funcType.(*ClassType); ok {
		// super(...) runs the parent's constructor from a subclass; that is
		// exactly what an abstract base class exists for, so it is not an
		// instantiation of one.
		_, viaSuper := node.Function.(*ast.SuperExpression)

		// Enemy(3) constructs through Enemy.new; recording it here means
		// codegen gets it right for classes from other modules too.
		node.ConstructorCall = !viaSuper && !classType.Ambient

		// Check if trying to instantiate an abstract class
		if classType.IsAbstract && !viaSuper {
			c.addError(
				fmt.Sprintf("Cannot instantiate abstract class '%s'", classType.Name),
				node.Token,
			)
			return classType
		}

		// Check for constructor (or allow default constructor with no args)
		ctor := classType.EffectiveConstructor()
		// Box(4) on a generic class: its type parameters are not given, so
		// they take what the arguments are.
		if ctor != nil && len(classType.GenericParams) > 0 {
			ctor = eraseTypeParams(ctor)
		}
		if ctor == nil {
			// No explicit constructor - allow instantiation with zero arguments only
			if len(node.Arguments) > 0 {
				c.addError(
					fmt.Sprintf("Class '%s' has no constructor and cannot accept arguments", classType.Name),
					node.Token,
				)
			}
			return classType
		}

		fnType := ctor

		// Check if any argument is a spread expression
		hasSpread := false
		for _, arg := range node.Arguments {
			if _, ok := arg.(*ast.SpreadExpression); ok {
				hasSpread = true
				break
			}
		}

		// Check argument count (skip if there are spread arguments). Trailing
		// optional parameters may be omitted, as with any other call.
		if !hasSpread && (len(node.Arguments) < requiredParamCount(fnType) || len(node.Arguments) > len(fnType.Parameters)) {
			c.addError(
				fmt.Sprintf("Constructor expects %d arguments, got %d",
					len(fnType.Parameters), len(node.Arguments)),
				node.Token,
			)
			return fnType.ReturnType
		}

		// Check argument types
		for i, arg := range node.Arguments {
			argType := c.checkExpression(arg)
			if c.reportLiteralFields(arg, fnType.Parameters[i]) {
				continue
			}
			if !argType.IsAssignableTo(fnType.Parameters[i]) {
				c.addError(
					fmt.Sprintf("Argument %d: cannot pass type '%s' to parameter of type '%s'",
						i+1, argType.String(), fnType.Parameters[i].String()),
					node.Token,
				)
			}
		}

		return fnType.ReturnType
	}

	// Check if it's a function type
	fnType, ok := funcType.(*FunctionType)
	if !ok {
		if !funcType.Equals(Any) && !hasMetatable(funcType) {
			c.addError(
				fmt.Sprintf("Cannot call type '%s'", funcType.String()),
				node.Token,
			)
		}
		return Any
	}

	// Check if any argument is a spread expression (used for multiple checks)
	hasSpreadArgs := false
	for _, arg := range node.Arguments {
		if _, ok := arg.(*ast.SpreadExpression); ok {
			hasSpreadArgs = true
			break
		}
	}

	// Resolve overloads if the function has multiple signatures
	if len(fnType.Overloads) > 0 && !hasSpreadArgs {
		fnType = c.resolveOverload(fnType, node)
	}

	// If the function is generic and called without explicit type arguments, try to infer them
	if len(fnType.GenericParams) > 0 {
		// Check argument count first (skip if there are spread arguments or function has rest parameter)
		if !hasSpreadArgs && !fnType.HasRestParameter {
			// Calculate minimum required arguments (non-optional parameters)
			minArgs := len(fnType.Parameters)
			if len(fnType.OptionalParams) > 0 {
				// Find the first optional parameter
				for i, isOptional := range fnType.OptionalParams {
					if isOptional {
						minArgs = i
						break
					}
				}
			}

			maxArgs := len(fnType.Parameters)
			argCount := len(node.Arguments)

			if argCount < minArgs || argCount > maxArgs {
				if minArgs == maxArgs {
					c.addError(
						fmt.Sprintf("Function expects %d arguments, got %d",
							maxArgs, argCount),
						node.Token,
					)
				} else {
					c.addError(
						fmt.Sprintf("Function expects %d-%d arguments, got %d",
							minArgs, maxArgs, argCount),
						node.Token,
					)
				}
				return fnType.ReturnType
			}
		}

		// If function has rest parameter, check minimum argument count
		if !hasSpreadArgs && fnType.HasRestParameter {
			minArgs := len(fnType.Parameters) - 1
			if len(node.Arguments) < minArgs {
				c.addError(
					fmt.Sprintf("Function expects at least %d arguments, got %d",
						minArgs, len(node.Arguments)),
					node.Token,
				)
				return fnType.ReturnType
			}
		}

		// Infer type arguments from call arguments
		typeArgs := make([]Type, len(fnType.GenericParams))
		inferred := make([]bool, len(fnType.GenericParams))

		for i, arg := range node.Arguments {
			if i >= len(fnType.Parameters) {
				break
			}
			argType := c.checkExpression(arg)
			// T is found wherever it appears in the parameter's type:
			// x: T, xs: T[], f: (T) -> U, t: table<K, V>, v: T?.
			inferTypeArgs(fnType.Parameters[i], argType, fnType.GenericParams, typeArgs, inferred)
		}

		// Check if all type parameters were inferred
		allInferred := true
		for i, inf := range inferred {
			if !inf {
				c.addError(
					fmt.Sprintf("Could not infer type parameter '%s'", fnType.GenericParams[i]),
					node.Token,
				)
				allInferred = false
				typeArgs[i] = Any // Fallback to Any
			}
		}

		if allInferred {
			// Instantiate the function with inferred types
			fnType = c.instantiateGenericFunction(fnType, typeArgs)
		}
	}

	// A last argument of ... or a call can be any number of values: it
	// may fill the parameters left.
	lastIsMulti := false
	if n := len(node.Arguments); n > 0 {
		switch last := node.Arguments[n-1].(type) {
		case *ast.VarargExpression:
			lastIsMulti = true
		case *ast.CallExpression:
			lastIsMulti = !last.Parenthesized
		}
	}

	// Check argument count (skip if there are spread arguments or function has rest parameter)
	if !hasSpreadArgs && !fnType.HasRestParameter && !fnType.Untyped {
		// Calculate minimum required arguments (non-optional parameters)
		minArgs := len(fnType.Parameters)
		if len(fnType.OptionalParams) > 0 {
			// Find the first optional parameter
			for i, isOptional := range fnType.OptionalParams {
				if isOptional {
					minArgs = i
					break
				}
			}
		}

		maxArgs := len(fnType.Parameters)
		argCount := len(node.Arguments)

		if argCount < minArgs && !lastIsMulti {
			c.addError(
				fmt.Sprintf("Function expects at least %d arguments, got %d",
					minArgs, argCount),
				node.Token,
			)
			return fnType.ReturnType
		}

		if argCount > maxArgs {
			c.addError(
				fmt.Sprintf("Function expects at most %d arguments, got %d",
					maxArgs, argCount),
				node.Token,
			)
			return fnType.ReturnType
		}
	}

	// If function has rest parameter, check minimum argument count
	if !hasSpreadArgs && fnType.HasRestParameter && !fnType.Untyped && !lastIsMulti {
		minArgs := len(fnType.Parameters) - 1 // All params except the rest parameter
		if len(node.Arguments) < minArgs {
			c.addError(
				fmt.Sprintf("Function expects at least %d arguments, got %d",
					minArgs, len(node.Arguments)),
				node.Token,
			)
			return fnType.ReturnType
		}
	}

	// Check argument types (basic check, skip detailed checks with spread)
	if !hasSpreadArgs {
		for i, arg := range node.Arguments {
			argType := c.checkExpression(arg)

			// Determine which parameter type to check against
			var paramType Type
			if fnType.HasRestParameter && i >= len(fnType.Parameters)-1 {
				// This argument goes to the rest parameter - unwrap array type
				restParamType := fnType.Parameters[len(fnType.Parameters)-1]
				if arrayType, ok := restParamType.(*ArrayType); ok {
					paramType = arrayType.ElementType
				} else {
					paramType = Any
				}
			} else if i < len(fnType.Parameters) {
				paramType = fnType.Parameters[i]
			} else {
				// This should not happen if our count checking is correct
				continue
			}

			if c.reportLiteralFields(arg, paramType) {
				continue
			}
			if !argType.IsAssignableTo(paramType) {
				c.addError(
					fmt.Sprintf("Argument %d: cannot pass type '%s' to parameter of type '%s'",
						i+1, argType.String(), paramType.String()),
					node.Token,
				)
			}
		}
	}

	return fnType.ReturnType
}

// resolveOverload finds the best matching overload for a function call
func (c *Checker) resolveOverload(fnType *FunctionType, node *ast.CallExpression) *FunctionType {
	// Collect all signatures: the primary signature plus all overloads
	allSignatures := make([]*FunctionType, 0, len(fnType.Overloads)+1)
	allSignatures = append(allSignatures, fnType)
	allSignatures = append(allSignatures, fnType.Overloads...)

	// Get argument types
	argTypes := make([]Type, len(node.Arguments))
	for i, arg := range node.Arguments {
		argTypes[i] = c.checkExpression(arg)
	}

	// Find matching overloads
	var matches []*FunctionType
	for _, sig := range allSignatures {
		if c.overloadMatches(sig, argTypes) {
			matches = append(matches, sig)
		}
	}

	// If no matches found, return the primary signature (will generate error later)
	if len(matches) == 0 {
		c.addError(
			fmt.Sprintf("No overload matches this call with %d arguments", len(argTypes)),
			node.Token,
		)
		return fnType
	}

	// If exactly one match, use it
	if len(matches) == 1 {
		return matches[0]
	}

	// Multiple matches - find the most specific one
	best := matches[0]
	for _, match := range matches[1:] {
		if c.isMoreSpecific(match, best, argTypes) {
			best = match
		}
	}

	return best
}

// overloadMatches checks if a function signature matches the given arguments
func (c *Checker) overloadMatches(fnType *FunctionType, argTypes []Type) bool {
	// Check argument count
	if fnType.HasRestParameter {
		// Need at least (params - 1) arguments
		if len(argTypes) < len(fnType.Parameters)-1 {
			return false
		}
	} else {
		// Must have exact number of arguments
		if len(argTypes) != len(fnType.Parameters) {
			return false
		}
	}

	// Check if each argument is assignable to its parameter
	for i, argType := range argTypes {
		var paramType Type
		if fnType.HasRestParameter && i >= len(fnType.Parameters)-1 {
			// Rest parameter - get element type
			restParamType := fnType.Parameters[len(fnType.Parameters)-1]
			if arrayType, ok := restParamType.(*ArrayType); ok {
				paramType = arrayType.ElementType
			} else {
				paramType = Any
			}
		} else if i < len(fnType.Parameters) {
			paramType = fnType.Parameters[i]
		} else {
			return false
		}

		if !argType.IsAssignableTo(paramType) {
			return false
		}
	}

	return true
}

// isMoreSpecific returns true if sig1 is more specific than sig2 for the given arguments
func (c *Checker) isMoreSpecific(sig1, sig2 *FunctionType, argTypes []Type) bool {
	// A signature is more specific if its parameter types are narrower
	score1 := 0
	score2 := 0

	for i, argType := range argTypes {
		if i >= len(sig1.Parameters) || i >= len(sig2.Parameters) {
			break
		}

		param1 := sig1.Parameters[i]
		param2 := sig2.Parameters[i]

		// If param1 equals the argument type exactly, it's more specific
		if param1.Equals(argType) && !param2.Equals(argType) {
			score1++
		} else if param2.Equals(argType) && !param1.Equals(argType) {
			score2++
		}

		// 'any' is less specific than other types
		if param1.Equals(Any) && !param2.Equals(Any) {
			score2++
		} else if param2.Equals(Any) && !param1.Equals(Any) {
			score1++
		}
	}

	return score1 > score2
}

// checkDotExpression checks a dot expression (property access)
func (c *Checker) checkDotExpression(node *ast.DotExpression) Type {
	called := c.callee == node
	leftType := c.checkExpression(node.Left)

	// Right side must be an identifier
	rightIdent, ok := node.Right.(*ast.Identifier)
	if !ok {
		c.addError("Right side of dot expression must be an identifier", node.Token)
		return Any
	}

	propertyName := rightIdent.Value

	// Handle optional chaining - unwrap optional types
	isLeftOptional := false
	actualLeftType := leftType
	if optType, ok := leftType.(*OptionalType); ok {
		isLeftOptional = true
		actualLeftType = optType.BaseType
	}

	// Helper function to wrap result if needed
	wrapIfOptional := func(resultType Type) Type {
		if node.IsOptional || isLeftOptional {
			return &OptionalType{BaseType: resultType}
		}
		return resultType
	}

	// Record how the member resolved so codegen knows whether a call through
	// this expression needs Lua's implicit-self ':' or a plain '.'.
	resolved := func(dispatch string, resultType Type) Type {
		// An explicit obj:method() in the source already states its intent
		// for codegen; the dispatch is still recorded, so the call can count
		// obj as the first argument of a plain function field.
		node.MethodDispatch = dispatch
		return wrapIfOptional(resultType)
	}

	c.reference(rightIdent.Token, memberSymbol(actualLeftType, propertyName))

	// A method named without a call loses its receiver: `local f = e.hit`
	// then f(3) passes 3 as self. Lunar has no bound methods, so refuse it.
	method := func(owner string, methodType Type) Type {
		if !called && !node.IsMethodCall {
			c.addErrorCode("method-value", fmt.Sprintf("Method '%s' of '%s' has to be called where it is named (%s.%s(...)): read as a value it loses its receiver; wrap it in a function to pass it around",
				propertyName, owner, exprText(node.Left), propertyName), rightIdent.Token)
		}
		return resolved(ast.DispatchSelf, methodType)
	}

	// A string literal type (const s = "abc") is still a string receiver.
	if _, ok := actualLeftType.(*StringLiteralType); ok {
		actualLeftType = String
	}
	if c.usesUnknown(actualLeftType, rightIdent.Token) {
		return Any
	}

	// Check if left type has the property
	switch typ := actualLeftType.(type) {
	case *UnionType:
		dispatch, member, ok := c.unionMember(typ, propertyName, rightIdent.Token)
		switch {
		case !ok:
			return wrapIfOptional(Any)
		case dispatch == ast.DispatchSelf:
			return method(typ.String(), member)
		default:
			return resolved(dispatch, member)
		}
	case *NamespaceType:
		if member, ok := typ.Members[propertyName]; ok {
			return resolved(ast.DispatchPlain, member)
		}
		{
			names := make([]string, 0, len(typ.Members))
			for name := range typ.Members {
				names = append(names, name)
			}
			sort.Strings(names)
			c.addErrorCode("unknown-member", fmt.Sprintf("Namespace '%s' has no member '%s'.%s", typ.Name, propertyName, didYouMean(propertyName, names)), rightIdent.Token)
		}
		return wrapIfOptional(Any)
	case *ClassType:
		c.accessedDeclarer = declaringClass(typ, propertyName)
		// The class's own name on the left means the class itself rather than
		// an instance; one type stands for both.
		// So does a namespace or module member of that name: Game.Enemy,
		// m.Enemy.
		viaClassName := false
		switch left := node.Left.(type) {
		case *ast.Identifier:
			viaClassName = left.Value == typ.Name
		case *ast.DotExpression:
			if right, ok := left.Right.(*ast.Identifier); ok {
				viaClassName = right.Value == typ.Name
			}
		}

		// ClassName.new(...) is the constructor every Lunar class gets.
		if propertyName == "new" && !typ.Ambient {
			if !viaClassName {
				c.addError(fmt.Sprintf("'new' is static; call %s.new(...) or %s(...)", typ.Name, typ.Name), rightIdent.Token)
				return wrapIfOptional(Any)
			}
			if typ.IsAbstract {
				c.addError(fmt.Sprintf("Cannot instantiate abstract class '%s'", typ.Name), rightIdent.Token)
			}
			ctor := typ.EffectiveConstructor()
			if ctor == nil {
				ctor = &FunctionType{Parameters: []Type{}, ReturnType: typ}
			}
			return resolved(ast.DispatchPlain, ctor)
		}

		// A static member belongs to the class, an instance member to its
		// instances; reaching one through the other finds nothing at run time.
		if !viaClassName && !typ.Ambient {
			if _, isStatic := typ.GetStaticProperty(propertyName); isStatic {
				c.addError(fmt.Sprintf("'%s' is static: write %s.%s", propertyName, typ.Name, propertyName), rightIdent.Token)
			} else if _, isStatic := typ.GetStaticMethod(propertyName); isStatic {
				c.addError(fmt.Sprintf("'%s' is static: write %s.%s(...)", propertyName, typ.Name, propertyName), rightIdent.Token)
			}
		}
		if viaClassName && !typ.Ambient {
			if _, isMethod := typ.GetMethod(propertyName); isMethod {
				if _, isStatic := typ.GetStaticMethod(propertyName); !isStatic {
					c.addError(fmt.Sprintf("'%s' is an instance method: call it on a %s, not on the class", propertyName, typ.Name), rightIdent.Token)
					return wrapIfOptional(Any)
				}
			}
		}

		// Check static properties first (when accessing class directly like Math.PI)
		if propType, ok := typ.GetStaticProperty(propertyName); ok {
			// Check visibility for static properties
			visibility := typ.PropertyVisibility[propertyName]
			if visibility == "" {
				visibility = "public"
			}
			if !c.canAccessMember(typ, visibility) {
				c.addError(
					fmt.Sprintf("Cannot access %s static property '%s' of class '%s'", visibility, propertyName, typ.Name),
					node.Token,
				)
			}
			return resolved(ast.DispatchPlain, propType)
		}
		// Check static methods
		if methodType, ok := typ.GetStaticMethod(propertyName); ok {
			// Check visibility for static methods
			visibility := typ.MethodVisibility[propertyName]
			if visibility == "" {
				visibility = "public"
			}
			if !c.canAccessMember(typ, visibility) {
				c.addError(
					fmt.Sprintf("Cannot access %s static method '%s' of class '%s'", visibility, propertyName, typ.Name),
					node.Token,
				)
			}
			return resolved(ast.DispatchPlain, methodType)
		}
		// Check instance properties
		if propType, ok := typ.GetProperty(propertyName); ok {
			// Check visibility for instance properties
			visibility := typ.GetPropertyVisibility(propertyName)
			if !c.canAccessMember(typ, visibility) {
				c.addError(
					fmt.Sprintf("Cannot access %s property '%s' of class '%s'", visibility, propertyName, typ.Name),
					node.Token,
				)
			}
			return resolved(ast.DispatchPlain, propType)
		}
		// Check instance methods
		if methodType, ok := typ.GetMethod(propertyName); ok {
			// Check visibility for instance methods
			visibility := typ.GetMethodVisibility(propertyName)
			if !c.canAccessMember(typ, visibility) {
				c.addError(
					fmt.Sprintf("Cannot access %s method '%s' of class '%s'", visibility, propertyName, typ.Name),
					node.Token,
				)
			}
			return method(typ.Name, methodType)
		}
		// Accessors read and write like fields.
		if getter, ok := typ.GetGetter(propertyName); ok {
			return resolved(ast.DispatchPlain, getter.ReturnType)
		}
		if setter, ok := typ.GetSetter(propertyName); ok && len(setter.Parameters) == 1 {
			return resolved(ast.DispatchPlain, setter.Parameters[0])
		}
		// A declared class lists what the engine's table has; anything else
		// is a mistake, as for a declared interface.
		if typ.Ambient {
			c.addErrorCode("unknown-member", fmt.Sprintf("Class '%s' has no member '%s'.%s", typ.Name, propertyName, didYouMean(propertyName, typ.MemberNames())), rightIdent.Token)
			return wrapIfOptional(Any)
		}
		// Probing for a subclass's member is how a type guard tells them
		// apart (`return value.breed ~= nil`); it reads as optional here.
		if member, ok := c.subclassMember(typ, propertyName); ok {
			return resolved(ast.DispatchPlain, withNil(member))
		}
		c.addErrorCode("unknown-member", fmt.Sprintf("Class '%s' has no member '%s'.%s", typ.Name, propertyName, didYouMean(propertyName, typ.MemberNames())), rightIdent.Token)
		return wrapIfOptional(Any)

	case *InterfaceType:
		// Check properties. A function-typed property is a plain field holding a
		// function (this is how the stdlib declares table.insert, math.floor, ...),
		// so it is called with '.' and never takes an implicit self.
		if propType, ok := typ.GetProperty(propertyName); ok {
			return resolved(ast.DispatchPlain, propType)
		}
		// Check methods
		if methodType, ok := typ.GetMethod(propertyName); ok {
			// A declared interface describes a Lua or engine table, whose
			// functions are values like any other: io.stdout.write ~= nil.
			if sym := typ.MemberSyms[propertyName]; sym != nil && strings.HasSuffix(sym.Decl.File, ".d.lunar") && !called && !node.IsMethodCall {
				return resolved(ast.DispatchSelf, methodType)
			}
			return method(typ.String(), methodType)
		}
		// A string index signature covers every other name.
		if sig := typ.IndexSignature; sig != nil && sig.ValueType != nil {
			if _, isString := sig.KeyType.(*StringType); isString {
				return resolved(ast.DispatchPlain, sig.ValueType)
			}
		}
		// A table from an unannotated literal is a Lua table: a field it
		// does not have is nil, or one set somewhere the checker cannot see.
		if typ.Open {
			return wrapIfOptional(Any)
		}
		c.addError(
			fmt.Sprintf("Type '%s' has no property or method '%s'", typ.String(), propertyName),
			node.Token,
		)
		return wrapIfOptional(Any)

	case *EnumType:
		// Check enum members
		if memberType, ok := typ.GetMemberType(propertyName); ok {
			return wrapIfOptional(memberType)
		}
		c.addError(
			fmt.Sprintf("Enum '%s' has no member '%s'", typ.String(), propertyName),
			node.Token,
		)
		return wrapIfOptional(Any)

	case *StringType:
		// Lua exposes the string library through every string value's metatable,
		// so s.upper() is really string.upper(s) and needs implicit-self syntax.
		// The member type stays 'any': the declared StringLib signatures include
		// the receiver and mark every parameter as required, so checking calls
		// against them would reject correct code like s:match(pattern).
		if lib, ok := c.interfaces["StringLib"]; ok {
			prop, isProp := lib.GetProperty(propertyName)
			_, isMethod := lib.GetMethod(propertyName)
			if isProp || isMethod {
				c.stringMethodUses = append(c.stringMethodUses, rightIdent.Token)
				if c.stringDots == nil {
					c.stringDots = map[*ast.DotExpression]*FunctionType{}
				}
				fn, _ := prop.(*FunctionType)
				if fn == nil {
					fn = &FunctionType{HasRestParameter: true}
				}
				c.stringDots[node] = fn
				return resolved(ast.DispatchSelf, Any)
			}
			c.addErrorCode("unknown-member", fmt.Sprintf("A string has no method '%s'.%s", propertyName, didYouMean(propertyName, interfaceMemberNames(lib))), rightIdent.Token)
		}
		return wrapIfOptional(Any)

	case *TableType:
		// t.key on table<string, V> is t["key"]: a V (or nil, if the key
		// is missing, which a table type cannot rule out).
		if IsStringType(typ.KeyType) || isAnyType(typ.KeyType) {
			return resolved(ast.DispatchPlain, typ.ValueType)
		}
		return wrapIfOptional(Any)

	default:
		// For other types, allow any property access (could be table access)
		return wrapIfOptional(Any)
	}
}

// unionMember resolves a member on every type in a union. ok is false when
// some type's members are not known (any, a table, a type parameter...): the
// expression is then emitted as written. When the types disagree, or one
// lacks the member, that is reported, since no one Lua call fits them all.
func (c *Checker) unionMember(u *UnionType, name string, tok lexer.Token) (dispatch string, member Type, ok bool) {
	type part struct {
		owner    string
		dispatch string
		member   Type
	}
	var parts []part
	for _, t := range u.Types {
		if opt, isOpt := t.(*OptionalType); isOpt {
			t = opt.BaseType
		}
		if _, isNil := t.(*NilType); isNil || t == nil {
			continue
		}
		d, m, known, has := c.memberOf(t, name)
		if !known {
			return "", nil, false
		}
		if !has {
			c.addErrorCode("unknown-member", fmt.Sprintf("'%s' has no member '%s' (in '%s')", t.String(), name, u.String()), tok)
			return "", nil, false
		}
		parts = append(parts, part{t.String(), d, m})
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	first := parts[0]
	member = first.member
	for _, p := range parts[1:] {
		if p.dispatch != first.dispatch {
			describe := func(p part) string {
				if p.dispatch == ast.DispatchSelf {
					return "a method of '" + p.owner + "'"
				}
				return "a plain field of '" + p.owner + "'"
			}
			c.addErrorCode("union-dispatch", fmt.Sprintf("'%s' is %s but %s, so no one call works for '%s'",
				name, describe(first), describe(p), u.String()), tok)
			return "", nil, false
		}
		if member != nil && !p.member.Equals(member) {
			member = nil
		}
	}
	if member == nil {
		member = Any
	}
	return first.dispatch, member, true
}

// memberOf looks a member up on one type without reporting anything. known
// is false for types whose members cannot be listed.
func (c *Checker) memberOf(t Type, name string) (dispatch string, member Type, known, has bool) {
	switch typ := t.(type) {
	case *ClassType:
		if m, ok := typ.GetMethod(name); ok {
			return ast.DispatchSelf, m, true, true
		}
		if p, ok := typ.GetProperty(name); ok {
			return ast.DispatchPlain, p, true, true
		}
		if g, ok := typ.GetGetter(name); ok {
			return ast.DispatchPlain, g.ReturnType, true, true
		}
		if typ.Ambient {
			return "", nil, false, false
		}
		return "", nil, true, false
	case *InterfaceType:
		if p, ok := typ.GetProperty(name); ok {
			return ast.DispatchPlain, p, true, true
		}
		if m, ok := typ.GetMethod(name); ok {
			return ast.DispatchSelf, m, true, true
		}
		return "", nil, true, false
	case *StringType, *StringLiteralType:
		if lib, ok := c.interfaces["StringLib"]; ok {
			_, isProp := lib.GetProperty(name)
			_, isMethod := lib.GetMethod(name)
			return ast.DispatchSelf, Any, true, isProp || isMethod
		}
	}
	return "", nil, false, false
}

// exprText is a short rendering of a receiver for messages.
func exprText(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Value
	case *ast.DotExpression:
		return exprText(n.Left) + "." + exprText(n.Right)
	}
	return "obj"
}

// subclassMember finds a property, method or getter on some class that
// extends base, directly or not.
func (c *Checker) subclassMember(base *ClassType, name string) (Type, bool) {
	names := make([]string, 0, len(c.classes))
	for n := range c.classes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		cl := c.classes[n]
		if cl == base {
			continue
		}
		extends := false
		for p := cl.Parent; p != nil; p = p.Parent {
			if p == base {
				extends = true
				break
			}
		}
		if !extends {
			continue
		}
		if t, ok := cl.Properties[name]; ok {
			return t, true
		}
		if m, ok := cl.Methods[name]; ok {
			return m, true
		}
		if g, ok := cl.Getters[name]; ok {
			return g.ReturnType, true
		}
	}
	return nil, false
}

// checkIndexExpression checks an index expression
func (c *Checker) checkIndexExpression(node *ast.IndexExpression) Type {
	leftType := c.checkExpression(node.Left)
	indexType := c.checkExpression(node.Index)

	// Optional index access unwraps an optional receiver and makes the result
	// optional, mirroring how '?.' is handled.
	isLeftOptional := false
	if optType, ok := leftType.(*OptionalType); ok {
		isLeftOptional = true
		leftType = optType.BaseType
	}

	result := func(resultType Type) Type {
		if node.IsOptional || isLeftOptional {
			return &OptionalType{BaseType: resultType}
		}
		return resultType
	}

	if c.usesUnknown(leftType, node.Token) {
		return Any
	}

	switch typ := leftType.(type) {
	case *ArrayType:
		// `{}` is an empty table, not only an array: seen[name] = true.
		if typ.EmptyLiteral && !IsNumericType(indexType) {
			return result(Any)
		}
		// Index must be a number
		if !IsNumericType(indexType) && !indexType.Equals(Any) {
			c.addError(
				fmt.Sprintf("Array index must be number, got '%s'", indexType.String()),
				node.Token,
			)
		}
		return result(typ.ElementType)

	case *TableType:
		// Index must match key type
		if !indexType.IsAssignableTo(typ.KeyType) {
			c.addError(
				fmt.Sprintf("Table key must be '%s', got '%s'", typ.KeyType.String(), indexType.String()),
				node.Token,
			)
		}
		return result(typ.ValueType)

	default:
		// For other types, allow any index access
		return Any
	}
}

// checkGenericInstantiation checks a generic type instantiation like Box<string> or identity<number>
func (c *Checker) checkGenericInstantiation(node *ast.GenericType) Type {
	// Get the base type (should be an identifier)
	baseIdent, ok := node.BaseType.(*ast.Identifier)
	if !ok {
		c.addError("Generic instantiation base must be an identifier", node.Token)
		return Any
	}
	// first<number>(...) and Box<number>(...) are uses of first and Box.
	if sym := c.env.lookupSym(baseIdent.Value); sym != nil {
		c.reference(baseIdent.Token, sym)
	} else {
		c.reference(baseIdent.Token, c.typeSyms[baseIdent.Value])
	}

	// Resolve type arguments
	typeArgs := make([]Type, len(node.TypeArguments))
	for i, arg := range node.TypeArguments {
		typeArgs[i] = c.resolveTypeExpression(arg)
	}

	// Check if it's a generic class
	if classType, exists := c.classes[baseIdent.Value]; exists {
		if len(classType.GenericParams) == 0 {
			c.addError(
				fmt.Sprintf("Class '%s' is not generic", classType.Name),
				node.Token,
			)
			return classType
		}

		// Check type argument count
		if len(typeArgs) != len(classType.GenericParams) {
			c.addError(
				fmt.Sprintf("Class '%s' expects %d type arguments, got %d",
					classType.Name, len(classType.GenericParams), len(typeArgs)),
				node.Token,
			)
			return classType
		}

		// Create an instantiated class type with substituted type parameters
		return c.instantiateGenericClass(classType, typeArgs)
	}

	// Check if it's a generic function
	if typ, exists := c.env.Get(baseIdent.Value); exists {
		if funcType, ok := typ.(*FunctionType); ok {
			if len(funcType.GenericParams) == 0 {
				c.addError(
					fmt.Sprintf("Function '%s' is not generic", baseIdent.Value),
					node.Token,
				)
				return funcType
			}

			// Check type argument count
			if len(typeArgs) != len(funcType.GenericParams) {
				c.addError(
					fmt.Sprintf("Function '%s' expects %d type arguments, got %d",
						baseIdent.Value, len(funcType.GenericParams), len(typeArgs)),
					node.Token,
				)
				return funcType
			}

			// Create an instantiated function type with substituted type parameters
			return c.instantiateGenericFunction(funcType, typeArgs)
		}
	}

	c.addError(fmt.Sprintf("'%s' is not a generic type or function", baseIdent.Value), node.Token)
	return Any
}

// checkSpreadExpression checks a spread expression
func (c *Checker) checkSpreadExpression(node *ast.SpreadExpression) Type {
	valueType := c.checkExpression(node.Value)

	// Spread expression should contain an array or table
	if arrayType, ok := valueType.(*ArrayType); ok {
		return arrayType
	}

	if tableType, ok := valueType.(*TableType); ok {
		return tableType
	}

	// A record-like table literal is typed structurally as an interface; those
	// spread too, merging their key/value pairs.
	if interfaceType, ok := valueType.(*InterfaceType); ok {
		return interfaceType
	}

	// If it's Any, allow it
	if valueType.Equals(Any) {
		return Any
	}

	c.addError(
		fmt.Sprintf("Cannot spread type '%s', expected an array, table or record", valueType.String()),
		node.Token,
	)
	return Any
}

// checkTypeAssertion checks a type assertion (value as Type)
func (c *Checker) checkTypeAssertion(node *ast.TypeAssertion) Type {
	// Check if this is a const assertion (as const)
	if ident, ok := node.TargetType.(*ast.Identifier); ok && ident.Value == "const" {
		// This is a const assertion - apply literal type inference
		return c.applyConstAssertion(node.Expression)
	}

	// Check the expression being asserted (ensures it's valid)
	c.checkExpression(node.Expression)

	// Resolve the target type
	targetType := c.resolveTypeExpression(node.TargetType)

	// Type assertions are generally allowed, but we can warn about impossible casts
	// For now, we'll be permissive - the developer knows what they're doing
	// We could add stricter checking here in the future

	// Return the target type - that's what the assertion claims it is
	return targetType
}

// applyConstAssertion applies const assertion logic to an expression
// This makes literals as specific as possible and objects/arrays readonly
func (c *Checker) applyConstAssertion(expr ast.Expression) Type {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		// Already returns NumberLiteralType
		return &NumberLiteralType{Value: e.Value}

	case *ast.StringLiteral:
		// Already returns StringLiteralType
		return &StringLiteralType{Value: e.Value}

	case *ast.BooleanLiteral:
		// Already returns BooleanLiteralType
		return &BooleanLiteralType{Value: e.Value}

	case *ast.TableLiteral:
		// For arrays: convert to readonly tuple with literal types
		// For objects: make all properties readonly with literal types
		return c.applyConstToTableLiteral(e)

	default:
		// For other expressions, just check normally
		return c.checkExpression(expr)
	}
}

// applyConstToTableLiteral applies const assertion to a table literal
func (c *Checker) applyConstToTableLiteral(node *ast.TableLiteral) Type {
	// Check if this is an array (only Values, no Pairs)
	if len(node.Pairs) == 0 && len(node.Values) > 0 {
		// Convert to readonly tuple with literal element types
		elementTypes := make([]Type, len(node.Values))
		for i, val := range node.Values {
			elementTypes[i] = c.applyConstAssertion(val)
		}
		// Return a tuple type (which is like a readonly array with fixed length)
		return &TupleType{Elements: elementTypes}
	}

	// For objects (key-value pairs), create an interface with readonly properties
	if len(node.Pairs) > 0 {
		properties := make(map[string]Type)
		readonlyProps := make(map[string]bool)

		for key, value := range node.Pairs {
			// Check if key is an identifier (property name)
			if ident, ok := key.(*ast.Identifier); ok {
				// Apply const assertion to get literal type for the value
				valueType := c.applyConstAssertion(value)
				properties[ident.Value] = valueType
				readonlyProps[ident.Value] = true
			}
		}

		// Return an interface type with readonly properties
		return &InterfaceType{
			Name:          "<const table literal>",
			Properties:    properties,
			Methods:       make(map[string]*FunctionType),
			Extends:       []*InterfaceType{},
			ReadonlyProps: readonlyProps,
		}
	}

	// Empty table - return as-is
	return c.checkTableLiteral(node)
}

// checkAwaitExpression checks an await expression
func (c *Checker) checkAwaitExpression(node *ast.AwaitExpression) Type {
	c.addErrorCode("not-supported", "'await' is not supported: use coroutine.yield inside a coroutine", node.Token)

	// Check the expression being awaited
	exprType := c.checkExpression(node.Expression)

	// If it's a PromiseType, unwrap it
	if promiseType, ok := exprType.(*PromiseType); ok {
		return promiseType.ResolveType
	}

	// If it's a function that returns a Promise, unwrap the return type
	if funcType, ok := exprType.(*FunctionType); ok {
		if promiseType, ok := funcType.ReturnType.(*PromiseType); ok {
			return promiseType.ResolveType
		}
	}

	// Allow awaiting Any type
	if exprType.Equals(Any) {
		return Any
	}

	// Warn if awaiting a non-promise type (but still allow it)
	// In Lua, you can coroutine.yield anything
	return exprType
}

func (c *Checker) checkTemplateLiteral(node *ast.TemplateLiteral) Type {
	// Type check all expressions inside ${...}
	for _, expr := range node.Expressions {
		c.checkExpression(expr)
		// We don't enforce that expressions are strings - they'll be converted at runtime
	}

	// Template literals always result in a string
	return String
}

// canAccessMember checks if the current context can access a member with the given visibility
// declaringClass is the class in typ's chain that declares member name.
func declaringClass(typ *ClassType, name string) *ClassType {
	for cl := typ; cl != nil; cl = cl.Parent {
		if _, ok := cl.Properties[name]; ok {
			return cl
		}
		if _, ok := cl.Methods[name]; ok {
			return cl
		}
		if _, ok := cl.StaticProperties[name]; ok {
			return cl
		}
		if _, ok := cl.StaticMethods[name]; ok {
			return cl
		}
	}
	return typ
}

func (c *Checker) canAccessMember(targetClass *ClassType, visibility string) bool {
	// Public members are always accessible
	if visibility == "public" || visibility == "" {
		return true
	}

	// If no current class context, can only access public members
	if c.currentClass == nil {
		return false
	}

	// Private members can only be accessed from the class that declares
	// them, not from a subclass
	if visibility == "private" {
		return c.currentClass.Equals(targetClass) && c.currentClass == c.accessedDeclarer
	}

	// Protected members can be accessed from the same class or child classes
	if visibility == "protected" {
		return c.currentClass.Equals(targetClass) || c.currentClass.IsChildOf(targetClass)
	}

	return false
}

// addError adds a type error to the checker
func (c *Checker) addError(message string, token lexer.Token) {
	c.addErrorCode("type", message, token)
}

// addErrorCode records an error with a specific code. Codes are part of the
// JSON output that editors match on, so once released they do not change.
func (c *Checker) addErrorCode(code, message string, token lexer.Token) {
	endLine, endColumn := token.EndLine, token.EndColumn
	if endLine == 0 {
		endLine, endColumn = token.Line, token.Column
	}
	c.errors = append(c.errors, &TypeError{
		Message:   message,
		Line:      token.Line,
		Column:    token.Column,
		File:      token.File,
		EndLine:   endLine,
		EndColumn: endColumn,
		Code:      code,
	})
}

// registerNamespace registers all declarations in a namespace
func (c *Checker) registerNamespace(node *ast.NamespaceDeclaration) {
	nsType := c.buildNamespace(node)
	c.namespaces[node] = nsType
	c.env.Set(node.Name.Value, nsType)
}

// buildNamespace makes the type of a namespace from its declarations: its
// exported members, with the types their declarations give. Functions and
// variables get their checked types when the body is checked.
func (c *Checker) buildNamespace(node *ast.NamespaceDeclaration) *NamespaceType {
	nsName := node.Name.Value

	// Create a namespace type to track its contents
	nsType := &NamespaceType{
		Name:    nsName,
		Members: make(map[string]Type),
	}

	// Register all declarations in the namespace with prefixed names
	for _, stmt := range node.Statements {
		// `export` inside a namespace marks what the namespace's table
		// holds; the declaration is registered the same either way.
		if exp, ok := stmt.(*ast.ExportStatement); ok {
			stmt = exp.Statement
		}
		switch s := stmt.(type) {
		case *ast.ClassDeclaration:
			c.registerClass(s)
			// Store reference in namespace
			if classType, ok := c.classes[s.Name.Value]; ok {
				nsType.Members[s.Name.Value] = classType
			}
		case *ast.InterfaceDeclaration:
			c.registerInterface(s)
			if ifaceType, ok := c.interfaces[s.Name.Value]; ok {
				nsType.Members[s.Name.Value] = ifaceType
			}
		case *ast.EnumDeclaration:
			c.registerEnum(s)
			if enumType, ok := c.enums[s.Name.Value]; ok {
				nsType.Members[s.Name.Value] = enumType
			}
		case *ast.TypeDeclaration:
			c.registerTypeAlias(s)
			if aliasType, ok := c.typeAliases[s.Name.Value]; ok {
				nsType.Members[s.Name.Value] = aliasType
			}
		case *ast.FunctionDeclaration:
			// Register function in namespace
			params := make([]Type, len(s.Parameters))
			for i, param := range s.Parameters {
				if param.Type != nil {
					params[i] = c.paramType(param)
				} else {
					params[i] = Any
				}
			}
			var returnType Type = Void
			if s.ReturnType != nil {
				returnType = c.resolveTypeExpression(s.ReturnType)
			}
			optional := make([]bool, len(s.Parameters))
			hasRest := false
			for i, param := range s.Parameters {
				optional[i] = paramIsOptional(param, params[i])
				if param.IsRest {
					hasRest = true
					if _, ok := params[i].(*ArrayType); !ok {
						params[i] = &ArrayType{ElementType: params[i]}
					}
				}
			}
			funcType := &FunctionType{
				Parameters:       params,
				ReturnType:       returnType,
				OptionalParams:   optional,
				HasRestParameter: hasRest,
				ParamNames:       parameterNames(s.Parameters),
				Untyped:          untypedSignature(s.Parameters, s.ReturnType, len(s.GenericParams)),
			}
			nsType.Members[s.Name.Value] = funcType
			setMemberSym(&nsType.MemberSyms, s.Name.Value, c.defineMember(s.Name.Token, SymFunction, funcType, nsName))
		case *ast.VariableDeclaration:
			for i, name := range s.Names {
				var t Type = Any
				if i < len(s.Types) && s.Types[i] != nil {
					t = c.resolveTypeExpression(s.Types[i])
				}
				nsType.Members[name.Value] = t
				setMemberSym(&nsType.MemberSyms, name.Value, c.defineMember(name.Token, SymVariable, t, nsName))
			}
		case *ast.NamespaceDeclaration:
			inner := c.buildNamespace(s)
			c.namespaces[s] = inner
			nsType.Members[s.Name.Value] = inner
		}
	}
	return nsType
}

// registerAmbientNamespace builds the type of a `declare namespace`, such as
// a C API exposed to scripts as a table of functions. Its functions and
// constants are reached only through the namespace; types declared inside it
// are registered globally, as for ordinary namespaces.
func (c *Checker) registerAmbientNamespace(node *ast.NamespaceDeclaration) *NamespaceType {
	ns := &NamespaceType{Name: node.Name.Value, Members: make(map[string]Type), Ambient: true}

	for _, stmt := range node.Statements {
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			fn := c.declaredFunctionType(s)
			fn.ParamNames = parameterNames(s.Parameters)
			if existing, ok := ns.Members[s.Name.Value].(*FunctionType); ok {
				existing.Overloads = append(existing.Overloads, fn)
			} else {
				ns.Members[s.Name.Value] = fn
				setMemberSym(&ns.MemberSyms, s.Name.Value, c.defineMember(s.Name.Token, SymFunction, fn, ns.Name))
			}
		case *ast.VariableDeclaration:
			for i, name := range s.Names {
				var t Type = Any
				if i < len(s.Types) && s.Types[i] != nil {
					t = c.resolveTypeExpression(s.Types[i])
				}
				ns.Members[name.Value] = t
				setMemberSym(&ns.MemberSyms, name.Value, c.defineMember(name.Token, SymVariable, t, ns.Name))
			}
		case *ast.NamespaceDeclaration:
			ns.Members[s.Name.Value] = c.registerAmbientNamespace(s)
		case *ast.ClassDeclaration:
			c.registerTypeDefinition(s)
			ns.Members[s.Name.Value] = c.classes[s.Name.Value]
		case *ast.InterfaceDeclaration, *ast.EnumDeclaration, *ast.TypeDeclaration:
			c.registerTypeDefinition(s)
		}
	}

	return ns
}

// checkNamespaceDeclaration checks all statements in a namespace
func (c *Checker) checkNamespaceDeclaration(node *ast.NamespaceDeclaration) {
	nsType := c.namespaces[node]
	// The body is a scope of its own: its members are names there (calling
	// each other in any order), not in the scope around it.
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	defer func() { c.env = prevEnv }()
	if nsType != nil {
		for name, t := range nsType.Members {
			c.env.Set(name, t)
		}
	}
	c.hoistFunctions(node.Statements, true)

	for _, stmt := range node.Statements {
		exported := false
		if exp, ok := stmt.(*ast.ExportStatement); ok && exp.Statement != nil {
			stmt, exported = exp.Statement, true
		}
		c.checkStatement(stmt)
		if !exported || nsType == nil {
			continue
		}
		// The member's type is what checking its declaration gave.
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			if t, ok := c.env.Get(s.Name.Value); ok {
				nsType.Members[s.Name.Value] = t
			}
		case *ast.VariableDeclaration:
			for _, name := range s.Names {
				if t, ok := c.env.Get(name.Value); ok {
					nsType.Members[name.Value] = t
				}
			}
		}
	}
}

// checkExportStatement checks an export statement
func (c *Checker) checkExportStatement(node *ast.ExportStatement) {
	// Type check the underlying statement
	c.checkStatement(node.Statement)

	// Track what's being exported
	switch stmt := node.Statement.(type) {
	case *ast.VariableDeclaration:
		// Export variables
		for i, name := range stmt.Names {
			var varType Type
			if i < len(stmt.Types) && stmt.Types[i] != nil {
				varType = c.resolveTypeExpression(stmt.Types[i])
			} else if i < len(stmt.Values) && stmt.Values[i] != nil {
				varType = c.checkExpression(stmt.Values[i])
			} else {
				varType = Any
			}
			c.exports[name.Value] = varType
		}

	case *ast.FunctionDeclaration:
		// Export functions, with the type checking gave them (a generic
		// one's type parameters are only in scope there).
		var funcType Type
		if t, ok := c.env.Get(stmt.Name.Value); ok {
			funcType = t
		} else {
			funcType = c.getFunctionType(stmt)
		}
		c.exports[stmt.Name.Value] = funcType

	case *ast.ClassDeclaration:
		// Export classes
		if classType, ok := c.classes[stmt.Name.Value]; ok {
			c.exports[stmt.Name.Value] = classType
		}

	case *ast.InterfaceDeclaration:
		// Export interfaces
		if interfaceType, ok := c.interfaces[stmt.Name.Value]; ok {
			c.exports[stmt.Name.Value] = interfaceType
		}
		if tmpl, ok := c.genericInterfaces[stmt.Name.Value]; ok {
			c.exports[stmt.Name.Value] = tmpl
		}

	case *ast.EnumDeclaration:
		// Export enums
		if enumType, ok := c.enums[stmt.Name.Value]; ok {
			c.exports[stmt.Name.Value] = enumType
		}

	case *ast.TypeDeclaration:
		// Export type aliases
		if aliasType, ok := c.typeAliases[stmt.Name.Value]; ok {
			c.exports[stmt.Name.Value] = aliasType
		}
		if genericAlias, ok := c.genericTypeAliases[stmt.Name.Value]; ok {
			// Export generic type aliases as a special marker
			c.exports[stmt.Name.Value] = &GenericType{
				Name: genericAlias.Name,
			}
		}
	}

	// Remember where each export is defined, so an importing module's uses
	// lead back to it.
	for name := range c.exports {
		if c.exportSyms[name] != nil {
			continue
		}
		if sym := c.typeSyms[name]; sym != nil {
			c.exportSyms[name] = sym
		} else if sym := c.env.lookupSym(name); sym != nil {
			c.exportSyms[name] = sym
		}
	}
}

// checkImportStatement checks an import statement
func (c *Checker) checkImportStatement(node *ast.ImportStatement) {
	if !c.boundImports[node] {
		c.bindImport(node)
	}
}

// bindImport resolves an import and puts the names it brings in into scope.
// An unresolved module or a missing export is an error; the names are then
// bound as 'any' so they do not cascade into more errors.
func (c *Checker) bindImport(node *ast.ImportStatement) {
	if c.boundImports == nil {
		c.boundImports = make(map[*ast.ImportStatement]bool)
	}
	c.boundImports[node] = true

	var info *ModuleInfo
	var err error
	if c.importResolver != nil {
		info, err = c.importResolver(node)
	} else {
		err = fmt.Errorf("Cannot find module '%s'", node.Module)
	}
	if err != nil {
		c.addErrorCode("import", err.Error(), node.Token)
		if node.Alias != nil {
			c.env.Set(node.Alias.Value, Any)
		}
		for _, name := range node.Names {
			c.env.Set(name.Value, Any)
		}
		return
	}

	if node.Alias != nil {
		ns := &NamespaceType{Name: node.Alias.Value, Members: info.Exports, Ambient: true, MemberSyms: info.Symbols}
		c.env.Set(node.Alias.Value, ns)
		c.define(node.Alias.Token, SymNamespace, ns)
		return
	}
	for _, name := range node.Names {
		typ, ok := info.Exports[name.Value]
		if !ok && info.Broken {
			c.env.Set(name.Value, Any)
			continue
		}
		if !ok {
			c.addErrorCode("import", fmt.Sprintf("Module '%s' does not export '%s'", node.Module, name.Value), name.Token)
			c.env.Set(name.Value, Any)
			continue
		}
		c.env.Set(name.Value, typ)
		if sym := info.Symbols[name.Value]; sym != nil && (sym.Kind == SymInterface || sym.Kind == SymTypeAlias) {
			if node.TypeNames == nil {
				node.TypeNames = map[string]bool{}
			}
			node.TypeNames[name.Value] = true
		}
		// Uses of an imported name are uses of the export's definition.
		if sym := info.Symbols[name.Value]; sym != nil {
			c.env.defineSym(name.Value, sym)
			c.reference(name.Token, sym)
			switch typ.(type) {
			case *ClassType, *InterfaceType, *EnumType:
				c.typeSyms[name.Value] = sym
			}
		}
		// Imported types are named like local ones.
		switch t := typ.(type) {
		case *ClassType:
			c.classes[name.Value] = t
		case *InterfaceType:
			c.interfaces[name.Value] = t
		case *EnumType:
			c.enums[name.Value] = t
		}
	}
}

// markAmbient records that a declaration bound name to typ.
func (c *Checker) markAmbient(name string, typ Type) {
	if c.ambient == nil {
		c.ambient = make(map[string]Type)
	}
	c.ambient[name] = typ
}

// GlobalUses returns the identifiers that referred to something a
// declaration provides, such as tostring or the engine's API: globals the
// Lua state has to supply at run time.
func (c *Checker) GlobalUses() []lexer.Token {
	return c.globalUses
}

// StringMethodUses returns the string methods called through a string
// value's metatable, as in s:upper().
func (c *Checker) StringMethodUses() []lexer.Token {
	return c.stringMethodUses
}

// declaredFunctionType builds the type of an ambient function signature.
func (c *Checker) declaredFunctionType(decl *ast.FunctionDeclaration) *FunctionType {
	params := make([]Type, len(decl.Parameters))
	optionalParams := make([]bool, len(decl.Parameters))
	hasRestParam := false
	for i, param := range decl.Parameters {
		if param.Type != nil {
			paramType := c.paramType(param)
			optionalParams[i] = !param.IsRest && paramIsOptional(param, paramType)
			// If this is a rest parameter, wrap in array type if not already
			if param.IsRest {
				hasRestParam = true
				if _, ok := paramType.(*ArrayType); !ok {
					paramType = &ArrayType{ElementType: paramType}
				}
			}
			params[i] = paramType
		} else {
			if param.IsRest {
				hasRestParam = true
				params[i] = &ArrayType{ElementType: Any}
			} else {
				params[i] = Any
			}
		}
	}

	var returnType Type = Void
	if decl.ReturnType != nil {
		returnType = c.resolveTypeExpression(decl.ReturnType)
	}

	funcType := &FunctionType{
		Parameters:       params,
		OptionalParams:   optionalParams,
		ReturnType:       returnType,
		GenericParams:    []string{}, // Ambient function declarations are not generic
		HasRestParameter: hasRestParam,
	}

	return funcType
}

// checkDeclareStatement handles ambient declarations
func (c *Checker) checkDeclareStatement(node *ast.DeclareStatement) {
	if node.Declaration == nil {
		return
	}

	if isNilNode(node.Declaration) {
		return
	}
	// For ambient declarations, we only register types, not check implementations
	switch decl := node.Declaration.(type) {
	case *ast.VariableDeclaration:
		// Register each variable with its declared type
		for i, name := range decl.Names {
			var varType Type
			if i < len(decl.Types) && decl.Types[i] != nil {
				varType = c.resolveTypeExpression(decl.Types[i])
			} else {
				// No type annotation on ambient declaration - use any
				varType = Any
			}

			if decl.IsConstant {
				c.env.SetConst(name.Value, varType)
			} else {
				c.env.Set(name.Value, varType)
			}
			c.markAmbient(name.Value, varType)
			c.define(name.Token, SymVariable, varType)
		}

	case *ast.FunctionDeclaration:
		funcType := c.declaredFunctionType(decl)
		if c.declaredFuncs == nil {
			c.declaredFuncs = make(map[*FunctionType]bool)
		}
		c.declaredFuncs[funcType] = true

		// Check if function with same name already exists (for method overloading)
		if existingType, ok := c.env.Get(decl.Name.Value); ok {
			if existingFunc, ok := existingType.(*FunctionType); ok {
				// Add this as an overload to the existing function
				existingFunc.Overloads = append(existingFunc.Overloads, funcType)
				// Don't re-set in environment, the existing function reference is updated
			} else {
				// Name exists but is not a function - just set the new function type
				c.env.Set(decl.Name.Value, funcType)
				c.markAmbient(decl.Name.Value, funcType)
			}
		} else {
			c.env.Set(decl.Name.Value, funcType)
			c.markAmbient(decl.Name.Value, funcType)
		}
		funcType.ParamNames = parameterNames(decl.Parameters)
		c.define(decl.Name.Token, SymFunction, funcType)

		// Class, Interface, Enum, Type declarations are already handled in registerTypeDefinition
	}
}

// Check is the main entry point for type checking
func Check(statements []ast.Statement) []*TypeError {
	checker := NewChecker()
	return checker.Check(statements)
}

// loadModule loads a module and returns its type information
// getFunctionType extracts the function type from a function declaration
func (c *Checker) getFunctionType(stmt *ast.FunctionDeclaration) Type {
	paramTypes := make([]Type, 0, len(stmt.Parameters))
	for _, param := range stmt.Parameters {
		if param.Type != nil {
			paramTypes = append(paramTypes, c.paramType(param))
		} else {
			paramTypes = append(paramTypes, Any)
		}
	}

	var returnType Type
	if stmt.ReturnType != nil {
		returnType = c.resolveTypeExpression(stmt.ReturnType)
	} else {
		returnType = Void
	}

	genericParams := make([]string, 0, len(stmt.GenericParams))
	for _, gp := range stmt.GenericParams {
		genericParams = append(genericParams, gp.Value)
	}

	return &FunctionType{
		Parameters:    paramTypes,
		ReturnType:    returnType,
		GenericParams: genericParams,
	}
}

// resolveMappedType resolves mapped types like { [K in T]: U }
func (c *Checker) resolveMappedType(node *ast.MappedType, constraintType Type) Type {
	// The constraint type should be a union of string literals (from keyof)
	// For each key in the constraint, we create a property with the value type

	var keys []string

	// Extract keys from the constraint type
	switch constraint := constraintType.(type) {
	case *UnionType:
		// Union of string literals
		for _, t := range constraint.Types {
			if strLiteral, ok := t.(*StringLiteralType); ok {
				keys = append(keys, strLiteral.Value)
			}
		}
	case *StringLiteralType:
		// Single string literal
		keys = append(keys, constraint.Value)
	default:
		// For other types, we can't iterate yet
		// Return Any for now
		return Any
	}

	// Now we need to create a new object type with properties for each key
	// We need to resolve the value type for each key
	// The value type can reference K (the type parameter)

	properties := make(map[string]Type)

	// Create a temporary environment for the type parameter K
	// We'll substitute K with each key when resolving the value type
	for _, key := range keys {
		// Create a type substitution for K -> key
		// For now, we'll resolve the value type with K substituted
		valueType := c.resolveMappedValueType(node.ValueType, node.TypeParam.Value, &StringLiteralType{Value: key})
		properties[key] = valueType
	}

	// Create an interface type with these properties
	interfaceType := &InterfaceType{
		Name:       "", // Anonymous mapped type
		Properties: properties,
	}

	return interfaceType
}

// resolveMappedValueType resolves the value type of a mapped type with a substitution for the type parameter
func (c *Checker) resolveMappedValueType(valueTypeExpr ast.Expression, typeParamName string, keyType Type) Type {
	// We need to resolve the value type expression with K substituted
	// For example, if the value type is T[K], we need to substitute K with the current key

	// Create a temporary environment with K bound to the key type
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)
	c.env.Set(typeParamName, keyType)

	// Resolve the value type expression with K in the environment
	result := c.resolveTypeExpression(valueTypeExpr)

	// Restore environment
	c.env = prevEnv

	return result
}

// resolveTemplateLiteralType resolves template literal types like `Hello ${string}` or `${T}_${U}`
func (c *Checker) resolveTemplateLiteralType(node *ast.TemplateLiteralType) Type {
	// Resolve all type expressions in the template
	resolvedTypes := make([]Type, len(node.Types))
	for i, typeExpr := range node.Types {
		resolvedTypes[i] = c.resolveTypeExpression(typeExpr)
	}

	// Generate all possible string combinations
	// Start with the first part
	combinations := []string{node.Parts[0]}

	// For each type expression, generate new combinations
	for i, typ := range resolvedTypes {
		var newCombinations []string

		// Extract possible string values from the type
		stringValues := c.extractStringValues(typ)

		// For each existing combination and each string value, create new combinations
		for _, combo := range combinations {
			for _, value := range stringValues {
				newCombo := combo + value + node.Parts[i+1]
				newCombinations = append(newCombinations, newCombo)
			}
		}

		combinations = newCombinations
	}

	// If we couldn't generate any combinations, or if any type is `string` (unbounded),
	// return a general string type
	if len(combinations) == 0 {
		return String
	}

	// Check if we have an unbounded string type
	for _, typ := range resolvedTypes {
		if _, isString := typ.(*StringType); isString {
			// Unbounded - return string type
			return String
		}
	}

	// Create a union of string literal types
	if len(combinations) == 1 {
		return &StringLiteralType{Value: combinations[0]}
	}

	literalTypes := make([]Type, len(combinations))
	for i, combo := range combinations {
		literalTypes[i] = &StringLiteralType{Value: combo}
	}

	return &UnionType{Types: literalTypes}
}

// extractStringValues extracts possible string values from a type
func (c *Checker) extractStringValues(typ Type) []string {
	switch t := typ.(type) {
	case *StringLiteralType:
		return []string{t.Value}
	case *UnionType:
		var values []string
		for _, memberType := range t.Types {
			values = append(values, c.extractStringValues(memberType)...)
		}
		return values
	case *StringType:
		// Unbounded string - return empty to signal this
		return []string{"<string>"}
	default:
		// For non-string types, convert to string representation
		return []string{t.String()}
	}
}

// checkDecorator checks a decorator usage
func (c *Checker) checkDecorator(decorator *ast.Decorator) {
	// Look up the decorator function
	decoratorName := decorator.Name.Value
	decoratorType, ok := c.env.Get(decoratorName)

	if !ok {
		c.addError(
			fmt.Sprintf("Undefined decorator '%s'", decoratorName),
			decorator.Token,
		)
		return
	}

	// Check that it's a function
	funcType, ok := decoratorType.(*FunctionType)
	if !ok {
		c.addError(
			fmt.Sprintf("Decorator '%s' is not a function", decoratorName),
			decorator.Token,
		)
		return
	}

	// For decorators, the semantics are:
	// - @decorator - simple decorator, the target is passed as first arg implicitly
	// - @decorator(args) - decorator factory, args are passed first, returns a function that takes the target

	// If no arguments in decorator syntax, it's a simple decorator
	// The function should take at least 1 parameter (the target)
	if len(decorator.Arguments) == 0 {
		if len(funcType.Parameters) == 0 {
			c.addError(
				fmt.Sprintf("Decorator '%s' must accept at least one parameter (the target)", decoratorName),
				decorator.Token,
			)
		}
		return
	}

	// For decorator factories, check the argument types
	// The decorator factory receives the decorator arguments
	expectedArgs := len(funcType.Parameters)
	actualArgs := len(decorator.Arguments)

	// Count required parameters (non-optional)
	requiredParams := 0
	for _, param := range funcType.Parameters {
		if _, isOptional := param.(*OptionalType); !isOptional {
			requiredParams++
		}
	}

	if actualArgs < requiredParams || actualArgs > expectedArgs {
		if requiredParams == expectedArgs {
			c.addError(
				fmt.Sprintf("Decorator '%s' expects %d arguments, got %d", decoratorName, expectedArgs, actualArgs),
				decorator.Token,
			)
		} else {
			c.addError(
				fmt.Sprintf("Decorator '%s' expects %d-%d arguments, got %d", decoratorName, requiredParams, expectedArgs, actualArgs),
				decorator.Token,
			)
		}
		return
	}

	// Check argument types
	for i, arg := range decorator.Arguments {
		argType := c.checkExpression(arg)
		if i < len(funcType.Parameters) {
			paramType := funcType.Parameters[i]
			// Unwrap optional type for comparison
			if optType, ok := paramType.(*OptionalType); ok {
				paramType = optType.BaseType
			}
			if !argType.IsAssignableTo(paramType) {
				c.addError(
					fmt.Sprintf("Decorator argument type mismatch: expected %s, got %s", paramType.String(), argType.String()),
					decorator.Token,
				)
			}
		}
	}
}
