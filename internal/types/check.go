package types

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"os"
)

// TypeError represents a type error
type TypeError struct {
	Message string
	Line    int
	Column  int
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("Type error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// Environment represents a scope with type bindings
type Environment struct {
	store     map[string]Type
	constVars map[string]bool // tracks which variables are const
	outer     *Environment
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
	typ, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return typ, ok
}

// Set sets a type in the environment
func (e *Environment) Set(name string, typ Type) {
	e.store[name] = typ
}

// SetConst sets a variable as const in the environment
func (e *Environment) SetConst(name string, typ Type) {
	e.store[name] = typ
	e.constVars[name] = true
}

// IsConst checks if a variable is const
func (e *Environment) IsConst(name string) bool {
	isConst, ok := e.constVars[name]
	if ok && isConst {
		return true
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
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

	// Sort by distance (simple bubble sort for small lists)
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].distance < suggestions[i].distance {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

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
	classes            map[string]*ClassType
	interfaces         map[string]*InterfaceType
	enums              map[string]*EnumType
	typeAliases        map[string]Type
	genericTypeAliases map[string]*GenericTypeAlias

	// Module system
	exports        map[string]Type // Tracks exported names and their types
	modules        map[string]*ModuleInfo // Tracks loaded modules
	currentModule  string // Current module being checked

	// Current function return type (for checking return statements)
	currentFunctionReturnType Type

	// Current class context (for visibility checking)
	currentClass *ClassType
}

// ModuleInfo represents information about a module
type ModuleInfo struct {
	Path    string
	Exports map[string]Type
}

// NewChecker creates a new type checker
func NewChecker() *Checker {
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
		enums:              make(map[string]*EnumType),
		typeAliases:        make(map[string]Type),
		genericTypeAliases: make(map[string]*GenericTypeAlias),
		exports:            make(map[string]Type),
		modules:            make(map[string]*ModuleInfo),
		currentModule:      "",
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
	c.env.Set("Exclude", Any)       // Exclude<T, U>
	c.env.Set("Extract", Any)       // Extract<T, U>
	c.env.Set("NonNullable", Any)   // NonNullable<T>
	c.env.Set("Pick", Any)          // Pick<T, K>
	c.env.Set("Omit", Any)          // Omit<T, K>
	c.env.Set("Record", Any)        // Record<K, V>
	c.env.Set("Partial", Any)       // Partial<T>
	c.env.Set("Required", Any)      // Required<T>
	c.env.Set("ReturnType", Any)    // ReturnType<F>
	c.env.Set("Parameters", Any)    // Parameters<F>
}

// loadStdlib loads the Lua standard library type definitions
func (c *Checker) loadStdlib() {
	// List of stdlib files to load
	stdlibFiles := []string{"lua.d.lunar", "table.d.lunar", "string.d.lunar", "math.d.lunar"}

	// Try to find stdlib directory relative to the current directory
	stdlibDirs := []string{
		"stdlib",
		"../stdlib",
		"../../stdlib",
		"../../../stdlib",
	}

	var foundDir string
	for _, dir := range stdlibDirs {
		if _, err := os.Stat(dir); err == nil {
			foundDir = dir
			break
		}
	}

	if foundDir == "" {
		// If stdlib not found, silently continue (tests can still define what they need)
		return
	}

	// Load each stdlib file
	for _, filename := range stdlibFiles {
		filepath := foundDir + "/" + filename
		content, err := os.ReadFile(filepath)
		if err != nil {
			continue // Skip files that don't exist
		}

		// Parse the stdlib file
		l := lexer.New(string(content))
		p := parser.New(l)
		statements := p.Parse()

		if len(p.Errors()) > 0 {
			// Stdlib file has parse errors, silently continue
			continue
		}

		// Register stdlib declarations
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
	// First pass: register all type definitions
	for _, stmt := range statements {
		c.registerTypeDefinition(stmt)
	}

	// Second pass: check all statements
	for _, stmt := range statements {
		c.checkStatement(stmt)
	}

	return c.errors
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
	case *ast.DeclareStatement:
		// Ambient declarations - register the underlying declaration
		if node.Declaration != nil {
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
		Name:                node.Name.Value,
		Properties:          make(map[string]Type),
		Methods:             make(map[string]*FunctionType),
		StaticProperties:    make(map[string]Type),
		StaticMethods:       make(map[string]*FunctionType),
		ReadonlyProps:       make(map[string]bool),
		AbstractMethods:     make(map[string]bool),
		PropertyVisibility:  make(map[string]string),
		MethodVisibility:    make(map[string]string),
		Implements:          []*InterfaceType{},
		IsAbstract:          node.IsAbstract,
		GenericParams:       genericParams,
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
		propName := prop.Name.Value

		// Track static vs instance properties
		if prop.IsStatic {
			classType.StaticProperties[propName] = propType
		} else {
			classType.Properties[propName] = propType
		}

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
		for i, param := range node.Constructor.Parameters {
			if param.Type != nil {
				params[i] = c.resolveTypeExpression(param.Type)
			} else {
				params[i] = Any
			}
		}
		classType.Constructor = &FunctionType{
			Parameters:    params,
			ReturnType:    classType, // Constructor returns an instance of the class
			GenericParams: []string{}, // Constructors are not separately generic
		}
	}

	// Register methods
	for _, method := range node.Methods {
		params := make([]Type, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = c.resolveTypeExpression(param.Type)
		}
		var returnType Type = Void
		if method.ReturnType != nil {
			returnType = c.resolveTypeExpression(method.ReturnType)
		}
		funcType := &FunctionType{
			Parameters:    params,
			ReturnType:    returnType,
			GenericParams: []string{}, // Methods are not separately generic
		}
		methodName := method.Name.Value

		// Track static vs instance methods
		if method.IsStatic {
			classType.StaticMethods[methodName] = funcType
		} else {
			classType.Methods[methodName] = funcType
		}

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

	c.classes[classType.Name] = classType
	c.env.Set(classType.Name, classType)
}

// registerInterface registers an interface type
func (c *Checker) registerInterface(node *ast.InterfaceDeclaration) {
	interfaceType := &InterfaceType{
		Name:       node.Name.Value,
		Methods:    make(map[string]*FunctionType),
		Properties: make(map[string]Type),
		Extends:    []*InterfaceType{},
	}

	// Register properties
	for _, prop := range node.Properties {
		propType := c.resolveTypeExpression(prop.Type)
		interfaceType.Properties[prop.Name.Value] = propType
	}

	// Register methods
	for _, method := range node.Methods {
		params := make([]Type, len(method.Parameters))
		for i, param := range method.Parameters {
			params[i] = c.resolveTypeExpression(param.Type)
		}
		var returnType Type = Void
		if method.ReturnType != nil {
			returnType = c.resolveTypeExpression(method.ReturnType)
		}
		interfaceType.Methods[method.Name.Value] = &FunctionType{
			Parameters:    params,
			ReturnType:    returnType,
			GenericParams: []string{}, // Interface methods are not separately generic
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

	c.interfaces[interfaceType.Name] = interfaceType
	c.env.Set(interfaceType.Name, interfaceType)
}

// registerEnum registers an enum type
func (c *Checker) registerEnum(node *ast.EnumDeclaration) {
	enumType := &EnumType{
		Name:    node.Name.Value,
		Members: make(map[string]Type),
	}

	// First, register the enum type itself so members can reference it
	c.enums[enumType.Name] = enumType
	c.env.Set(enumType.Name, enumType)

	for _, member := range node.Members {
		if member.Value != nil {
			// Validate the value expression (should be number or string)
			_ = c.checkExpression(member.Value)
		}
		// All enum members have the enum type itself, not the value type
		// This ensures type safety: Color.Red has type Color, not number
		enumType.Members[member.Name.Value] = enumType
	}
}

// registerTypeAlias registers a type alias
func (c *Checker) registerTypeAlias(node *ast.TypeDeclaration) {
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
			Name:       node.Name.Value,
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
			Extends:    []*InterfaceType{},
		}

		// Register properties
		for _, prop := range node.Properties {
			propType := c.resolveTypeExpression(prop.Type)
			interfaceType.Properties[prop.Name.Value] = propType
		}

		aliasType = interfaceType
	} else {
		aliasType = Any
	}

	c.typeAliases[node.Name.Value] = aliasType
	c.env.Set(node.Name.Value, aliasType)
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
			return typ
		}

		c.addError(fmt.Sprintf("Unknown type '%s'", node.Value), node.Token)
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
		params := make([]Type, len(node.Parameters))
		for i, param := range node.Parameters {
			params[i] = c.resolveTypeExpression(param.Type)
		}
		var returnType Type = Void
		if node.ReturnType != nil {
			returnType = c.resolveTypeExpression(node.ReturnType)
		}
		return &FunctionType{
			Parameters:    params,
			ReturnType:    returnType,
			GenericParams: []string{}, // Function types in annotations are not generic
		}

	case *ast.GenericType:
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
						lexer.Token{},
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
					c.addError("Exclude<T, U> expects 2 type arguments", lexer.Token{})
					return Any
				}
				return c.excludeTypes(typeArgs[0], typeArgs[1])

			case "Extract":
				if len(typeArgs) != 2 {
					c.addError("Extract<T, U> expects 2 type arguments", lexer.Token{})
					return Any
				}
				return c.extractTypes(typeArgs[0], typeArgs[1])

			case "NonNullable":
				if len(typeArgs) != 1 {
					c.addError("NonNullable<T> expects 1 type argument", lexer.Token{})
					return Any
				}
				return c.nonNullable(typeArgs[0])

			case "Pick":
				if len(typeArgs) != 2 {
					c.addError("Pick<T, K> expects 2 type arguments", lexer.Token{})
					return Any
				}
				return c.pickType(typeArgs[0], typeArgs[1])

			case "Omit":
				if len(typeArgs) != 2 {
					c.addError("Omit<T, K> expects 2 type arguments", lexer.Token{})
					return Any
				}
				return c.omitType(typeArgs[0], typeArgs[1])

			case "Record":
				if len(typeArgs) != 2 {
					c.addError("Record<K, V> expects 2 type arguments", lexer.Token{})
					return Any
				}
				return c.recordType(typeArgs[0], typeArgs[1])

			case "Partial":
				if len(typeArgs) != 1 {
					c.addError("Partial<T> expects 1 type argument", lexer.Token{})
					return Any
				}
				return c.partialType(typeArgs[0])

			case "Required":
				if len(typeArgs) != 1 {
					c.addError("Required<T> expects 1 type argument", lexer.Token{})
					return Any
				}
				return c.requiredType(typeArgs[0])

			case "ReturnType":
				if len(typeArgs) != 1 {
					c.addError("ReturnType<F> expects 1 type argument", lexer.Token{})
					return Any
				}
				return c.returnType(typeArgs[0])

			case "Parameters":
				if len(typeArgs) != 1 {
					c.addError("Parameters<F> expects 1 type argument", lexer.Token{})
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
							lexer.Token{},
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

	default:
		c.addError(fmt.Sprintf("Cannot resolve type expression: %T", expr), lexer.Token{})
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
		for _, propType := range classType.Properties {
			types = append(types, propType)
		}
		for _, methodType := range classType.Methods {
			types = append(types, methodType)
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
		for _, propType := range interfaceType.Properties {
			types = append(types, propType)
		}
		for _, methodType := range interfaceType.Methods {
			types = append(types, methodType)
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
			Name:       objType.Name + "_Picked",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		// Extract keys from the keys type
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

// recordType implements Record<K, V> utility type
// Constructs a type with a set of properties K of type V
func (c *Checker) recordType(keys Type, valueType Type) Type {
	record := &InterfaceType{
		Name:       "Record",
		Properties: make(map[string]Type),
		Methods:    make(map[string]*FunctionType),
	}

	keyList := c.extractKeyNames(keys)
	for _, key := range keyList {
		record.Properties[key] = valueType
	}

	return record
}

// partialType implements Partial<T> utility type
// Makes all properties in T optional
func (c *Checker) partialType(t Type) Type {
	switch objType := t.(type) {
	case *InterfaceType:
		partial := &InterfaceType{
			Name:       objType.Name + "_Partial",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		// Make all properties optional (T | nil)
		for name, propType := range objType.Properties {
			partial.Properties[name] = &UnionType{Types: []Type{propType, Nil}}
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
			Name:       objType.Name + "_Required",
			Properties: make(map[string]Type),
			Methods:    make(map[string]*FunctionType),
		}

		// Remove optionality from properties
		for name, propType := range objType.Properties {
			required.Properties[name] = c.nonNullable(propType)
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
		for propName := range t.Properties {
			keys = append(keys, &StringLiteralType{Value: propName})
		}
		for methodName := range t.Methods {
			keys = append(keys, &StringLiteralType{Value: methodName})
		}

	case *InterfaceType:
		// Extract property names from interface
		for propName := range t.Properties {
			keys = append(keys, &StringLiteralType{Value: propName})
		}
		for methodName := range t.Methods {
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
	// Create substitution map
	substitutions := make(map[string]Type)
	for i, param := range classType.GenericParams {
		substitutions[param] = typeArgs[i]
	}

	// Create a new class type with substituted types
	instantiated := &ClassType{
		Name:              classType.Name,
		Properties:        make(map[string]Type),
		Methods:           make(map[string]*FunctionType),
		StaticProperties:  make(map[string]Type),
		StaticMethods:     make(map[string]*FunctionType),
		Constructor:       nil,
		Parent:            classType.Parent,
		Implements:        classType.Implements,
		IsAbstract:        classType.IsAbstract,
		AbstractMethods:   classType.AbstractMethods,
		GenericParams:     []string{}, // Instantiated classes are not generic
		PropertyVisibility: classType.PropertyVisibility,
		MethodVisibility:  classType.MethodVisibility,
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
		Parameters:    newParams,
		ReturnType:    newReturnType,
		GenericParams: []string{}, // Instantiated functions are not generic
	}
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

	default:
		// For all other types (primitives, classes, etc.), return as-is
		return t
	}
}

// checkStatement checks a statement
func (c *Checker) checkStatement(stmt ast.Statement) {
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
	case *ast.ReturnStatement:
		c.checkReturnStatement(node)
	case *ast.IfStatement:
		c.checkIfStatement(node)
	case *ast.WhileStatement:
		c.checkWhileStatement(node)
	case *ast.ForStatement:
		c.checkForStatement(node)
	case *ast.DoStatement:
		c.checkDoStatement(node)
	case *ast.BreakStatement:
		// Nothing to check for break
	case *ast.BlockStatement:
		c.checkBlockStatement(node)
	case *ast.AssignmentStatement:
		c.checkAssignmentStatement(node)
	case *ast.ClassDeclaration:
		c.checkClassDeclaration(node)
	case *ast.InterfaceDeclaration:
		// Interface declarations don't need runtime checking
	case *ast.EnumDeclaration:
		// Enum declarations don't need runtime checking
	case *ast.TypeDeclaration:
		// Type declarations don't need runtime checking
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
	// Resolve declared types for each variable
	declaredTypes := make([]Type, len(node.Names))
	for i, typ := range node.Types {
		if typ != nil {
			declaredTypes[i] = c.resolveTypeExpression(typ)
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
				varType = Nil
			}

			if node.IsConstant {
				c.env.SetConst(name.Value, varType)
			} else {
				c.env.Set(name.Value, varType)
			}
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

	// Check that the number of variables matches number of values
	if len(node.Names) != len(valueTypes) {
		c.addError(
			fmt.Sprintf("Number of variables (%d) does not match number of values (%d)",
				len(node.Names), len(valueTypes)),
			node.Token,
		)
		// Register variables anyway to avoid cascading errors
		for i, name := range node.Names {
			var varType Type = Any
			if i < len(declaredTypes) && declaredTypes[i] != nil {
				varType = declaredTypes[i]
			} else if i < len(valueTypes) {
				varType = valueTypes[i]
			}

			if node.IsConstant {
				c.env.SetConst(name.Value, varType)
			} else {
				c.env.Set(name.Value, varType)
			}
		}
		return
	}

	// Register each variable with type checking
	for i, name := range node.Names {
		declaredType := declaredTypes[i]
		valueType := valueTypes[i]

		// Determine final type and check assignability
		var finalType Type
		if declaredType != nil {
			// Type is declared - check if value is assignable
			if !valueType.IsAssignableTo(declaredType) {
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
		}

		// Register variable
		if node.IsConstant {
			c.env.SetConst(name.Value, finalType)
		} else {
			c.env.Set(name.Value, finalType)
		}
	}
}

// checkFunctionDeclaration checks a function declaration
func (c *Checker) checkFunctionDeclaration(node *ast.FunctionDeclaration) {
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
	for i, param := range node.Parameters {
		// Validate rest parameters
		if param.IsRest {
			// Rest parameter must be last
			if i != len(node.Parameters)-1 {
				c.addError("Rest parameter must be the last parameter", param.Token)
			}
			// Rest parameter type should be an array
			if param.Type != nil {
				paramType := c.resolveTypeExpression(param.Type)
				if _, ok := paramType.(*ArrayType); !ok && !paramType.Equals(Any) {
					c.addError(
						fmt.Sprintf("Rest parameter must have array type, got '%s'", paramType.String()),
						param.Token,
					)
				}
				params[i] = paramType
			} else {
				// Default to any[] for rest parameters
				params[i] = &ArrayType{ElementType: Any}
			}
		} else {
			if param.Type != nil {
				params[i] = c.resolveTypeExpression(param.Type)
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
	}

	// Restore environment and register function
	if len(node.GenericParams) > 0 {
		c.env = prevEnv
	}
	c.env.Set(node.Name.Value, funcType)

	// Check function body in new scope
	prevReturnType := c.currentFunctionReturnType
	c.env = NewEnclosedEnvironment(c.env)
	c.currentFunctionReturnType = returnType

	// Add generic type parameters to scope
	for _, genericParam := range node.GenericParams {
		c.env.Set(genericParam.Value, Any)
	}

	// Add parameters to scope
	for i, param := range node.Parameters {
		c.env.Set(param.Name.Value, params[i])
	}

	// Check body
	c.checkBlockStatement(node.Body)

	c.env = prevEnv
	c.currentFunctionReturnType = prevReturnType
}

// checkReturnStatement checks a return statement
func (c *Checker) checkReturnStatement(node *ast.ReturnStatement) {
	if c.currentFunctionReturnType == nil {
		c.addError("Return statement outside of function", node.Token)
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

// checkIfStatement checks an if statement
func (c *Checker) checkIfStatement(node *ast.IfStatement) {
	condType := c.checkExpression(node.Condition)
	if !IsBooleanType(condType) && !condType.Equals(Any) {
		c.addError(
			fmt.Sprintf("If condition must be boolean, got '%s'", condType.String()),
			node.Token,
		)
	}

	// Check for type narrowing from type guard functions
	varName, narrowedType := c.extractTypeNarrowing(node.Condition)

	if varName != "" && narrowedType != nil {
		// Type narrowing applies: create new scope with narrowed type
		prevEnv := c.env

		// Then block: apply narrowed type
		c.env = NewEnclosedEnvironment(prevEnv)
		originalType, _ := prevEnv.Get(varName)
		c.env.Set(varName, narrowedType)
		c.checkBlockStatement(node.Consequence)
		c.env = prevEnv

		// Else block: apply complementary type if possible
		if node.Alternative != nil {
			c.env = NewEnclosedEnvironment(prevEnv)
			if unionType, ok := originalType.(*UnionType); ok {
				// Remove the narrowed type from the union
				remainingTypes := []Type{}
				for _, t := range unionType.Types {
					if !t.Equals(narrowedType) {
						remainingTypes = append(remainingTypes, t)
					}
				}
				if len(remainingTypes) == 1 {
					c.env.Set(varName, remainingTypes[0])
				} else if len(remainingTypes) > 1 {
					c.env.Set(varName, &UnionType{Types: remainingTypes})
				}
			}
			c.checkBlockStatement(node.Alternative)
			c.env = prevEnv
		}
	} else {
		// No type narrowing: check blocks normally
		c.checkBlockStatement(node.Consequence)
		if node.Alternative != nil {
			c.checkBlockStatement(node.Alternative)
		}
	}
}

// extractTypeNarrowing checks if the expression is a type guard call and extracts narrowing info
// Returns the variable name and the narrowed type, or empty string and nil if not a type guard
func (c *Checker) extractTypeNarrowing(expr ast.Expression) (string, Type) {
	// Check if this is a call expression
	callExpr, ok := expr.(*ast.CallExpression)
	if !ok || len(callExpr.Arguments) != 1 {
		return "", nil
	}

	// Get the function being called
	var funcType *FunctionType
	if ident, ok := callExpr.Function.(*ast.Identifier); ok {
		typ, exists := c.env.Get(ident.Value)
		if !exists {
			return "", nil
		}
		funcType, ok = typ.(*FunctionType)
		if !ok {
			return "", nil
		}
	} else {
		return "", nil
	}

	// Check if return type is a type guard
	guardType, ok := funcType.ReturnType.(*TypeGuardType)
	if !ok {
		return "", nil
	}

	// Extract the variable being guarded from the argument
	if argIdent, ok := callExpr.Arguments[0].(*ast.Identifier); ok {
		return argIdent.Value, guardType.GuardedType
	}

	return "", nil
}

// checkWhileStatement checks a while statement
func (c *Checker) checkWhileStatement(node *ast.WhileStatement) {
	condType := c.checkExpression(node.Condition)
	if !IsBooleanType(condType) && !condType.Equals(Any) {
		c.addError(
			fmt.Sprintf("While condition must be boolean, got '%s'", condType.String()),
			node.Token,
		)
	}

	c.checkBlockStatement(node.Body)
}

// checkForStatement checks a for statement
func (c *Checker) checkForStatement(node *ast.ForStatement) {
	// Create new scope for loop
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)

	// Check loop variables
	for _, variable := range node.Variables {
		// For generic for loops, variables can be any type (index, value, etc.)
		// For numeric for loops, variable is a number (loop counter)
		if node.IsGeneric {
			c.env.Set(variable.Value, Any)
		} else {
			c.env.Set(variable.Value, Number)
		}
	}

	if node.IsGeneric {
		// Generic for loop (for-in)
		iterType := c.checkExpression(node.Iterator)
		// Check if iterator is iterable (array or table)
		if _, isArray := iterType.(*ArrayType); !isArray {
			if _, isTable := iterType.(*TableType); !isTable {
				if !iterType.Equals(Any) {
					c.addError(
						fmt.Sprintf("Cannot iterate over type '%s'", iterType.String()),
						node.Token,
					)
				}
			}
		}
	} else {
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

	for _, stmt := range node.Statements {
		c.checkStatement(stmt)
	}

	c.env = prevEnv
}

// checkAssignmentStatement checks an assignment statement
func (c *Checker) checkAssignmentStatement(node *ast.AssignmentStatement) {
	// Check if trying to assign to a const variable
	if ident, ok := node.Name.(*ast.Identifier); ok {
		if c.env.IsConst(ident.Value) {
			c.addError(
				fmt.Sprintf("Cannot assign to const variable '%s'", ident.Value),
				node.Token,
			)
			return
		}
	}

	// Check if trying to assign to a readonly property
	if dotExpr, ok := node.Name.(*ast.DotExpression); ok {
		leftType := c.checkExpression(dotExpr.Left)
		if classType, ok := leftType.(*ClassType); ok {
			if rightIdent, ok := dotExpr.Right.(*ast.Identifier); ok {
				propName := rightIdent.Value
				if classType.IsReadonly(propName) {
					c.addError(
						fmt.Sprintf("Cannot assign to readonly property '%s'", propName),
						node.Token,
					)
					return
				}
			}
		}
	}

	targetType := c.checkExpression(node.Name)
	valueType := c.checkExpression(node.Value)

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

		// Add parameters to scope
		for _, param := range node.Constructor.Parameters {
			var paramType Type = Any
			if param.Type != nil {
				paramType = c.resolveTypeExpression(param.Type)
			}
			c.env.Set(param.Name.Value, paramType)
		}

		// Check constructor body
		c.checkBlockStatement(node.Constructor.Body)

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

		// Add generic type parameters to scope
		for _, genericParam := range node.GenericParams {
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

		// Add parameters to scope
		for _, param := range method.Parameters {
			var paramType Type = Any
			if param.Type != nil {
				paramType = c.resolveTypeExpression(param.Type)
			}
			c.env.Set(param.Name.Value, paramType)
		}

		// Check method body
		c.checkBlockStatement(method.Body)

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

	// Check this parent's abstract methods
	for methodName := range parent.AbstractMethods {
		// Check if child class has this method
		if _, ok := class.Methods[methodName]; !ok {
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
	if expr == nil {
		return Void
	}

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
	case *ast.BooleanLiteral:
		return Boolean
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
		return c.checkCallExpression(node)
	case *ast.DotExpression:
		return c.checkDotExpression(node)
	case *ast.IndexExpression:
		return c.checkIndexExpression(node)
	case *ast.GenericType:
		return c.checkGenericInstantiation(node)
	case *ast.SpreadExpression:
		return c.checkSpreadExpression(node)
	case *ast.TypeAssertion:
		return c.checkTypeAssertion(node)
	default:
		return Any
	}
}

// checkIdentifier checks an identifier and returns its type
func (c *Checker) checkIdentifier(node *ast.Identifier) Type {
	typ, ok := c.env.Get(node.Value)
	if !ok {
		// Build error message with suggestions
		errorMsg := fmt.Sprintf("Undefined variable '%s'", node.Value)

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
		isRecord := true

		for key, value := range node.Pairs {
			// Check if key is an identifier (field name)
			if ident, ok := key.(*ast.Identifier); ok {
				valueType := c.checkExpression(value)
				// Distinguish between methods (functions) and properties
				if funcType, ok := valueType.(*FunctionType); ok {
					methods[ident.Value] = funcType
				} else {
					properties[ident.Value] = valueType
				}
			} else {
				// Not a simple identifier key, treat as regular table
				isRecord = false
				break
			}
		}

		// If all keys are identifiers, create a structural interface type
		if isRecord {
			return &InterfaceType{
				Name:       "<table literal>",
				Properties: properties,
				Methods:    methods,
				Extends:    []*InterfaceType{},
			}
		}
	}

	// Check if this is an array-style table (has sequential Values, or empty with no Pairs)
	if len(node.Pairs) == 0 {
		// Infer element type from the values
		// Collect all unique types
		seenTypes := make(map[string]Type)
		for _, value := range node.Values {
			valueType := c.checkExpression(value)
			// If it's a spread expression, unwrap the array type
			if _, ok := value.(*ast.SpreadExpression); ok {
				if arrayType, isArray := valueType.(*ArrayType); isArray {
					valueType = arrayType.ElementType
				}
			}
			seenTypes[valueType.String()] = valueType
		}

		// If all elements have the same type, use that type
		if len(seenTypes) == 1 {
			for _, t := range seenTypes {
				return &ArrayType{ElementType: t}
			}
		}

		// If elements have different types, create a union type
		if len(seenTypes) > 1 {
			types := make([]Type, 0, len(seenTypes))
			for _, t := range seenTypes {
				types = append(types, t)
			}
			return &ArrayType{ElementType: &UnionType{Types: types}}
		}

		// Empty array defaults to any[]
		return &ArrayType{ElementType: Any}
	}

	// For mixed or key-value tables, return a generic table type
	return &TableType{KeyType: Any, ValueType: Any}
}

// checkFunctionExpression checks an anonymous function expression
func (c *Checker) checkFunctionExpression(node *ast.FunctionExpression) Type {
	// Create new environment for function scope
	prevEnv := c.env
	c.env = NewEnclosedEnvironment(prevEnv)

	// Resolve parameter types and add to environment
	paramTypes := make([]Type, len(node.Parameters))
	for i, param := range node.Parameters {
		paramType := c.resolveTypeExpression(param.Type)
		paramTypes[i] = paramType
		c.env.Set(param.Name.Value, paramType)
	}

	// Resolve return type
	var returnType Type = Void
	if node.ReturnType != nil {
		returnType = c.resolveTypeExpression(node.ReturnType)
	} else {
		// Try to infer return type for arrow functions with single expression body
		if len(node.Body.Statements) == 1 {
			if retStmt, ok := node.Body.Statements[0].(*ast.ReturnStatement); ok {
				if len(retStmt.ReturnValues) == 1 {
					// Infer from the single return expression
					returnType = c.checkExpression(retStmt.ReturnValues[0])
				}
			}
		}
	}

	// Set currentFunctionReturnType for return statement checking
	prevReturnType := c.currentFunctionReturnType
	c.currentFunctionReturnType = returnType

	// Type check the body
	c.checkBlockStatement(node.Body)

	// Restore environment and return type
	c.env = prevEnv
	c.currentFunctionReturnType = prevReturnType

	// Return the function type
	return &FunctionType{
		Parameters:    paramTypes,
		ReturnType:    returnType,
		GenericParams: []string{}, // Function expressions are not generic
	}
}

// checkPrefixExpression checks a prefix expression
func (c *Checker) checkPrefixExpression(node *ast.PrefixExpression) Type {
	rightType := c.checkExpression(node.Right)

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

// checkInfixExpression checks an infix expression
func (c *Checker) checkInfixExpression(node *ast.InfixExpression) Type {
	leftType := c.checkExpression(node.Left)
	rightType := c.checkExpression(node.Right)

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
	funcType := c.checkExpression(node.Function)

	// Check if it's a class type (constructor call)
	if classType, ok := funcType.(*ClassType); ok {
		// Check if trying to instantiate an abstract class
		if classType.IsAbstract {
			c.addError(
				fmt.Sprintf("Cannot instantiate abstract class '%s'", classType.Name),
				node.Token,
			)
			return classType
		}

		// Check for constructor (or allow default constructor with no args)
		if classType.Constructor == nil {
			// No explicit constructor - allow instantiation with zero arguments only
			if len(node.Arguments) > 0 {
				c.addError(
					fmt.Sprintf("Class '%s' has no constructor and cannot accept arguments", classType.Name),
					node.Token,
				)
			}
			return classType
		}

		fnType := classType.Constructor

		// Check if any argument is a spread expression
		hasSpread := false
		for _, arg := range node.Arguments {
			if _, ok := arg.(*ast.SpreadExpression); ok {
				hasSpread = true
				break
			}
		}

		// Check argument count (skip if there are spread arguments)
		if !hasSpread && len(node.Arguments) != len(fnType.Parameters) {
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
		if !funcType.Equals(Any) {
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

	// If the function is generic and called without explicit type arguments, try to infer them
	if len(fnType.GenericParams) > 0 {
		// Check argument count first (skip if there are spread arguments or function has rest parameter)
		if !hasSpreadArgs && !fnType.HasRestParameter && len(node.Arguments) != len(fnType.Parameters) {
			c.addError(
				fmt.Sprintf("Function expects %d arguments, got %d",
					len(fnType.Parameters), len(node.Arguments)),
				node.Token,
			)
			return fnType.ReturnType
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
			argType := c.checkExpression(arg)
			paramType := fnType.Parameters[i]

			// If the parameter type is a GenericParamType, infer from the argument
			if genericParam, ok := paramType.(*GenericParamType); ok {
				// Find which generic parameter this is
				for j, gpName := range fnType.GenericParams {
					if genericParam.Name == gpName {
						if !inferred[j] {
							typeArgs[j] = argType
							inferred[j] = true
						}
						break
					}
				}
			}
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

	// Check argument count (skip if there are spread arguments or function has rest parameter)
	if !hasSpreadArgs && !fnType.HasRestParameter && len(node.Arguments) != len(fnType.Parameters) {
		c.addError(
			fmt.Sprintf("Function expects %d arguments, got %d",
				len(fnType.Parameters), len(node.Arguments)),
			node.Token,
		)
		return fnType.ReturnType
	}

	// If function has rest parameter, check minimum argument count
	if !hasSpreadArgs && fnType.HasRestParameter {
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

// checkDotExpression checks a dot expression (property access)
func (c *Checker) checkDotExpression(node *ast.DotExpression) Type {
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

	// Check if left type has the property
	switch typ := actualLeftType.(type) {
	case *ClassType:
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
			return wrapIfOptional(propType)
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
			return wrapIfOptional(methodType)
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
			return wrapIfOptional(propType)
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
			return wrapIfOptional(methodType)
		}
		// In Lua, accessing a non-existent property returns nil at runtime
		// Allow this for type guards and dynamic property access
		// Return 'any' to be permissive (the property might exist in subclasses)
		return wrapIfOptional(Any)

	case *InterfaceType:
		// Check properties
		if propType, ok := typ.GetProperty(propertyName); ok {
			return wrapIfOptional(propType)
		}
		// Check methods
		if methodType, ok := typ.GetMethod(propertyName); ok {
			return wrapIfOptional(methodType)
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

	default:
		// For other types, allow any property access (could be table access)
		return wrapIfOptional(Any)
	}
}

// checkIndexExpression checks an index expression
func (c *Checker) checkIndexExpression(node *ast.IndexExpression) Type {
	leftType := c.checkExpression(node.Left)
	indexType := c.checkExpression(node.Index)

	switch typ := leftType.(type) {
	case *ArrayType:
		// Index must be a number
		if !IsNumericType(indexType) && !indexType.Equals(Any) {
			c.addError(
				fmt.Sprintf("Array index must be number, got '%s'", indexType.String()),
				node.Token,
			)
		}
		return typ.ElementType

	case *TableType:
		// Index must match key type
		if !indexType.IsAssignableTo(typ.KeyType) {
			c.addError(
				fmt.Sprintf("Table key must be '%s', got '%s'", typ.KeyType.String(), indexType.String()),
				node.Token,
			)
		}
		return typ.ValueType

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

	// If it's Any, allow it
	if valueType.Equals(Any) {
		return Any
	}

	c.addError(
		fmt.Sprintf("Cannot spread type '%s', expected array or table", valueType.String()),
		node.Token,
	)
	return Any
}

// checkTypeAssertion checks a type assertion (value as Type)
func (c *Checker) checkTypeAssertion(node *ast.TypeAssertion) Type {
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
func (c *Checker) canAccessMember(targetClass *ClassType, visibility string) bool {
	// Public members are always accessible
	if visibility == "public" || visibility == "" {
		return true
	}

	// If no current class context, can only access public members
	if c.currentClass == nil {
		return false
	}

	// Private members can only be accessed from the same class
	if visibility == "private" {
		return c.currentClass.Equals(targetClass)
	}

	// Protected members can be accessed from the same class or child classes
	if visibility == "protected" {
		return c.currentClass.Equals(targetClass) || c.currentClass.IsChildOf(targetClass)
	}

	return false
}

// addError adds a type error to the checker
func (c *Checker) addError(message string, token lexer.Token) {
	c.errors = append(c.errors, &TypeError{
		Message: message,
		Line:    token.Line,
		Column:  token.Column,
	})
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
		// Export functions
		funcType := c.getFunctionType(stmt)
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
}

// checkImportStatement checks an import statement
func (c *Checker) checkImportStatement(node *ast.ImportStatement) {
	// Try to load the module
	moduleInfo, err := c.loadModule(node.Module)
	if err != nil {
		// Module not found - for now, just add names as 'any' type
		// This allows for gradual typing and external modules
		if node.IsWildcard {
			// For wildcard imports, we can't add anything without knowing module exports
			c.addError(fmt.Sprintf("Cannot resolve module '%s'", node.Module), node.Token)
			return
		}

		for _, name := range node.Names {
			c.env.Set(name.Value, Any)
		}
		return
	}

	// Module loaded successfully - import the requested names
	if node.IsWildcard {
		// Import all exports
		for name, typ := range moduleInfo.Exports {
			c.env.Set(name, typ)
		}
	} else {
		// Import specific names
		for _, name := range node.Names {
			if typ, ok := moduleInfo.Exports[name.Value]; ok {
				c.env.Set(name.Value, typ)
			} else {
				c.addError(fmt.Sprintf("Module '%s' does not export '%s'", node.Module, name.Value), name.Token)
				c.env.Set(name.Value, Any) // Add as 'any' to prevent cascading errors
			}
		}
	}
}

// checkDeclareStatement handles ambient declarations
func (c *Checker) checkDeclareStatement(node *ast.DeclareStatement) {
	if node.Declaration == nil {
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
		}

	case *ast.FunctionDeclaration:
		// Register the function signature without checking the body
		params := make([]Type, len(decl.Parameters))
		for i, param := range decl.Parameters {
			if param.Type != nil {
				params[i] = c.resolveTypeExpression(param.Type)
			} else {
				params[i] = Any
			}
		}

		var returnType Type = Void
		if decl.ReturnType != nil {
			returnType = c.resolveTypeExpression(decl.ReturnType)
		}

		funcType := &FunctionType{
			Parameters:    params,
			ReturnType:    returnType,
			GenericParams: []string{}, // Ambient function declarations are not generic
		}
		c.env.Set(decl.Name.Value, funcType)

	// Class, Interface, Enum, Type declarations are already handled in registerTypeDefinition
	}
}

// Check is the main entry point for type checking
func Check(statements []ast.Statement) []*TypeError {
	checker := NewChecker()
	return checker.Check(statements)
}

// loadModule loads a module and returns its type information
func (c *Checker) loadModule(modulePath string) (*ModuleInfo, error) {
	// Check if module is already loaded
	if mod, ok := c.modules[modulePath]; ok {
		return mod, nil
	}

	// Try to find and parse the module file
	// Support both relative and absolute paths
	possiblePaths := []string{
		modulePath,
		modulePath + ".lunar",
		"./" + modulePath + ".lunar",
		"./" + modulePath,
	}

	var moduleFile string
	var found bool
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			moduleFile = path
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("module not found: %s", modulePath)
	}

	// Parse the module file
	source, err := os.ReadFile(moduleFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read module: %w", err)
	}

	l := lexer.New(string(source))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in module %s", modulePath)
	}

	// Create a new checker for the module
	moduleChecker := NewChecker()
	moduleChecker.currentModule = modulePath

	// Check the module and collect exports
	for _, stmt := range statements {
		moduleChecker.checkStatement(stmt)
	}

	// Store module info
	moduleInfo := &ModuleInfo{
		Path:    modulePath,
		Exports: moduleChecker.exports,
	}
	c.modules[modulePath] = moduleInfo

	return moduleInfo, nil
}

// getFunctionType extracts the function type from a function declaration
func (c *Checker) getFunctionType(stmt *ast.FunctionDeclaration) Type {
	paramTypes := make([]Type, 0, len(stmt.Parameters))
	for _, param := range stmt.Parameters {
		if param.Type != nil {
			paramTypes = append(paramTypes, c.resolveTypeExpression(param.Type))
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
