package types

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
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

	// Current function return type (for checking return statements)
	currentFunctionReturnType Type
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

	return &Checker{
		env:                env,
		errors:             []*TypeError{},
		classes:            make(map[string]*ClassType),
		interfaces:         make(map[string]*InterfaceType),
		enums:              make(map[string]*EnumType),
		typeAliases:        make(map[string]Type),
		genericTypeAliases: make(map[string]*GenericTypeAlias),
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
	}
}

// registerClass registers a class type
func (c *Checker) registerClass(node *ast.ClassDeclaration) {
	classType := &ClassType{
		Name:       node.Name.Value,
		Properties: make(map[string]Type),
		Methods:    make(map[string]*FunctionType),
		Implements: []*InterfaceType{},
	}

	// Add generic type parameters to scope temporarily
	prevEnv := c.env
	if len(node.GenericParams) > 0 {
		c.env = NewEnclosedEnvironment(prevEnv)
		for _, genericParam := range node.GenericParams {
			c.env.Set(genericParam.Value, Any)
		}
	}

	// Register properties
	for _, prop := range node.Properties {
		propType := c.resolveTypeExpression(prop.Type)
		classType.Properties[prop.Name.Value] = propType
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
		classType.Methods[method.Name.Value] = &FunctionType{
			Parameters: params,
			ReturnType: returnType,
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
			Parameters: params,
			ReturnType: returnType,
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
		// Check for built-in types
		if typ, ok := c.env.Get(node.Value); ok {
			return typ
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
		return &FunctionType{Parameters: params, ReturnType: returnType}

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

		// Not a generic type alias, try regular type resolution
		baseType := c.resolveTypeExpression(node.BaseType)
		return baseType

	case *ast.StringLiteral:
		// String literal in type position becomes a literal type
		return &StringLiteralType{Value: node.Value}

	case *ast.NumberLiteral:
		// Number literal in type position becomes a literal type
		return &NumberLiteralType{Value: node.Value}

	default:
		c.addError(fmt.Sprintf("Cannot resolve type expression: %T", expr), lexer.Token{})
		return Any
	}
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
	var declaredType Type
	if node.Type != nil {
		declaredType = c.resolveTypeExpression(node.Type)
	}

	var valueType Type
	if node.Value != nil {
		valueType = c.checkExpression(node.Value)
	} else {
		valueType = Nil
	}

	// If type is declared, check if value is assignable
	if declaredType != nil {
		if !valueType.IsAssignableTo(declaredType) {
			c.addError(
				fmt.Sprintf("Cannot assign type '%s' to variable of type '%s'",
					valueType.String(), declaredType.String()),
				node.Token,
			)
		}
		// Use SetConst if variable is declared as const
		if node.IsConstant {
			c.env.SetConst(node.Name.Value, declaredType)
		} else {
			c.env.Set(node.Name.Value, declaredType)
		}
	} else {
		// Infer type from value
		if node.IsConstant {
			c.env.SetConst(node.Name.Value, valueType)
		} else {
			c.env.Set(node.Name.Value, valueType)
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
			c.env.Set(genericParam.Value, Any)
		}
	}

	// Create function type
	params := make([]Type, len(node.Parameters))
	for i, param := range node.Parameters {
		if param.Type != nil {
			params[i] = c.resolveTypeExpression(param.Type)
		} else {
			params[i] = Any
		}
	}

	var returnType Type = Void
	if node.ReturnType != nil {
		returnType = c.resolveTypeExpression(node.ReturnType)
	}

	funcType := &FunctionType{
		Parameters: params,
		ReturnType: returnType,
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

	if node.ReturnValue == nil {
		if !IsVoidType(c.currentFunctionReturnType) {
			c.addError(
				fmt.Sprintf("Function must return a value of type '%s'",
					c.currentFunctionReturnType.String()),
				node.Token,
			)
		}
		return
	}

	returnType := c.checkExpression(node.ReturnValue)
	if !returnType.IsAssignableTo(c.currentFunctionReturnType) {
		c.addError(
			fmt.Sprintf("Cannot return type '%s' from function with return type '%s'",
				returnType.String(), c.currentFunctionReturnType.String()),
			node.Token,
		)
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

	c.checkBlockStatement(node.Consequence)
	if node.Alternative != nil {
		c.checkBlockStatement(node.Alternative)
	}
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

	// Check loop variable
	c.env.Set(node.Variable.Value, Number)

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

	// Check constructor if present
	if node.Constructor != nil {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		c.env = NewEnclosedEnvironment(prevEnv)
		c.currentFunctionReturnType = Void

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
	}

	// Check methods
	for _, method := range node.Methods {
		prevEnv := c.env
		prevReturnType := c.currentFunctionReturnType
		c.env = NewEnclosedEnvironment(prevEnv)

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

// checkExpression checks an expression and returns its type
func (c *Checker) checkExpression(expr ast.Expression) Type {
	if expr == nil {
		return Void
	}

	switch node := expr.(type) {
	case *ast.Identifier:
		return c.checkIdentifier(node)
	case *ast.NumberLiteral:
		// Number literals infer as literal types for precision
		return &NumberLiteralType{Value: node.Value}
	case *ast.StringLiteral:
		// String literals infer as literal types for precision
		return &StringLiteralType{Value: node.Value}
	case *ast.BooleanLiteral:
		return Boolean
	case *ast.NilLiteral:
		return Nil
	case *ast.TableLiteral:
		return c.checkTableLiteral(node)
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
	default:
		return Any
	}
}

// checkIdentifier checks an identifier and returns its type
func (c *Checker) checkIdentifier(node *ast.Identifier) Type {
	typ, ok := c.env.Get(node.Value)
	if !ok {
		c.addError(fmt.Sprintf("Undefined variable '%s'", node.Value), node.Token)
		return Any
	}
	return typ
}

// checkTableLiteral checks a table literal
func (c *Checker) checkTableLiteral(node *ast.TableLiteral) Type {
	// Check if this is a record-like table (all keys are string identifiers)
	if len(node.Values) == 0 && len(node.Pairs) > 0 {
		properties := make(map[string]Type)
		isRecord := true

		for key, value := range node.Pairs {
			// Check if key is an identifier (field name)
			if ident, ok := key.(*ast.Identifier); ok {
				valueType := c.checkExpression(value)
				properties[ident.Value] = valueType
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
				Methods:    make(map[string]*FunctionType),
				Extends:    []*InterfaceType{},
			}
		}
	}

	// For array-style or mixed tables, return a generic table type
	return &TableType{KeyType: Any, ValueType: Any}
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

	default:
		return Any
	}
}

// checkCallExpression checks a function call
func (c *Checker) checkCallExpression(node *ast.CallExpression) Type {
	funcType := c.checkExpression(node.Function)

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

	// Check argument count
	if len(node.Arguments) != len(fnType.Parameters) {
		c.addError(
			fmt.Sprintf("Function expects %d arguments, got %d",
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

	// Check if left type has the property
	switch typ := leftType.(type) {
	case *ClassType:
		// Check properties
		if propType, ok := typ.GetProperty(propertyName); ok {
			return propType
		}
		// Check methods
		if methodType, ok := typ.GetMethod(propertyName); ok {
			return methodType
		}
		c.addError(
			fmt.Sprintf("Type '%s' has no property or method '%s'", typ.String(), propertyName),
			node.Token,
		)
		return Any

	case *InterfaceType:
		// Check properties
		if propType, ok := typ.GetProperty(propertyName); ok {
			return propType
		}
		// Check methods
		if methodType, ok := typ.GetMethod(propertyName); ok {
			return methodType
		}
		c.addError(
			fmt.Sprintf("Type '%s' has no property or method '%s'", typ.String(), propertyName),
			node.Token,
		)
		return Any

	case *EnumType:
		// Check enum members
		if memberType, ok := typ.GetMemberType(propertyName); ok {
			return memberType
		}
		c.addError(
			fmt.Sprintf("Enum '%s' has no member '%s'", typ.String(), propertyName),
			node.Token,
		)
		return Any

	default:
		// For other types, allow any property access (could be table access)
		return Any
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
}

// checkImportStatement checks an import statement
func (c *Checker) checkImportStatement(node *ast.ImportStatement) {
	// For now, we skip type checking imports since we don't have module resolution
	// In a full implementation, we would:
	// 1. Resolve the module path
	// 2. Load the module's type information
	// 3. Add the imported names to the environment with their types

	// For now, just add imported names as 'any' type so they don't cause undefined variable errors
	for _, name := range node.Names {
		c.env.Set(name.Value, Any)
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
		// Register the variable with its declared type
		if decl.Type != nil {
			declaredType := c.resolveTypeExpression(decl.Type)
			if decl.IsConstant {
				c.env.SetConst(decl.Name.Value, declaredType)
			} else {
				c.env.Set(decl.Name.Value, declaredType)
			}
		} else {
			// No type annotation on ambient declaration - use any
			c.env.Set(decl.Name.Value, Any)
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
			Parameters: params,
			ReturnType: returnType,
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
