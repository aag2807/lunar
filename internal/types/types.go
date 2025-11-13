package types

import (
	"fmt"
	"lunar/internal/ast"
	"strings"
)

// Type represents a type in the Lunar type system
type Type interface {
	String() string
	Equals(other Type) bool
	IsAssignableTo(other Type) bool
}

// Basic Types

// NumberType represents the number type
type NumberType struct{}

func (t *NumberType) String() string { return "number" }
func (t *NumberType) Equals(other Type) bool {
	_, ok := other.(*NumberType)
	return ok
}
func (t *NumberType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Check if other is a union type that contains number
	if unionType, isUnion := other.(*UnionType); isUnion {
		return unionType.Contains(t)
	}
	return false
}

// StringType represents the string type
type StringType struct{}

func (t *StringType) String() string { return "string" }
func (t *StringType) Equals(other Type) bool {
	_, ok := other.(*StringType)
	return ok
}
func (t *StringType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Check if other is a union type that contains string
	if unionType, isUnion := other.(*UnionType); isUnion {
		return unionType.Contains(t)
	}
	return false
}

// BooleanType represents the boolean type
type BooleanType struct{}

func (t *BooleanType) String() string { return "boolean" }
func (t *BooleanType) Equals(other Type) bool {
	_, ok := other.(*BooleanType)
	return ok
}
func (t *BooleanType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Check if other is a union type that contains boolean
	if unionType, isUnion := other.(*UnionType); isUnion {
		return unionType.Contains(t)
	}
	return false
}

// NilType represents the nil type
type NilType struct{}

func (t *NilType) String() string { return "nil" }
func (t *NilType) Equals(other Type) bool {
	_, ok := other.(*NilType)
	return ok
}
func (t *NilType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	// nil is assignable to optional types
	if opt, ok := other.(*OptionalType); ok {
		return opt != nil
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Check if other is a union type that contains nil
	if unionType, isUnion := other.(*UnionType); isUnion {
		return unionType.Contains(t)
	}
	return false
}

// VoidType represents the void type (for functions with no return)
type VoidType struct{}

func (t *VoidType) String() string { return "void" }
func (t *VoidType) Equals(other Type) bool {
	_, ok := other.(*VoidType)
	return ok
}
func (t *VoidType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	_, isAny := other.(*AnyType)
	return isAny
}

// StringLiteralType represents a specific string value as a type
type StringLiteralType struct {
	Value string
}

func (t *StringLiteralType) String() string { return fmt.Sprintf("\"%s\"", t.Value) }
func (t *StringLiteralType) Equals(other Type) bool {
	otherLiteral, ok := other.(*StringLiteralType)
	if !ok {
		return false
	}
	return t.Value == otherLiteral.Value
}
func (t *StringLiteralType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// String literal is assignable to string type
	if _, isString := other.(*StringType); isString {
		return true
	}
	// Check if other is a union type that contains this literal OR the base string type
	if unionType, isUnion := other.(*UnionType); isUnion {
		// First check if the literal itself is in the union
		if unionType.Contains(t) {
			return true
		}
		// Then check if the base string type is in the union
		for _, ut := range unionType.Types {
			if _, isString := ut.(*StringType); isString {
				return true
			}
		}
	}
	return false
}

// NumberLiteralType represents a specific number value as a type
type NumberLiteralType struct {
	Value float64
}

func (t *NumberLiteralType) String() string { return fmt.Sprintf("%g", t.Value) }
func (t *NumberLiteralType) Equals(other Type) bool {
	otherLiteral, ok := other.(*NumberLiteralType)
	if !ok {
		return false
	}
	return t.Value == otherLiteral.Value
}
func (t *NumberLiteralType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Number literal is assignable to number type
	if _, isNumber := other.(*NumberType); isNumber {
		return true
	}
	// Check if other is a union type that contains this literal OR the base number type
	if unionType, isUnion := other.(*UnionType); isUnion {
		// First check if the literal itself is in the union
		if unionType.Contains(t) {
			return true
		}
		// Then check if the base number type is in the union
		for _, ut := range unionType.Types {
			if _, isNumber := ut.(*NumberType); isNumber {
				return true
			}
		}
	}
	return false
}

// AnyType represents the any type (accepts all types)
type AnyType struct{}

func (t *AnyType) String() string { return "any" }
func (t *AnyType) Equals(other Type) bool {
	_, ok := other.(*AnyType)
	return ok
}
func (t *AnyType) IsAssignableTo(other Type) bool {
	return true // any is assignable to any type
}

// Complex Types

// ArrayType represents an array type with element type
type ArrayType struct {
	ElementType Type
}

func (t *ArrayType) String() string {
	return fmt.Sprintf("%s[]", t.ElementType.String())
}
func (t *ArrayType) Equals(other Type) bool {
	otherArray, ok := other.(*ArrayType)
	if !ok {
		return false
	}
	return t.ElementType.Equals(otherArray.ElementType)
}
func (t *ArrayType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Array is covariant in its element type
	if otherArray, ok := other.(*ArrayType); ok {
		return t.ElementType.IsAssignableTo(otherArray.ElementType)
	}
	return false
}

// TableType represents a table type with key and value types
type TableType struct {
	KeyType   Type
	ValueType Type
}

func (t *TableType) String() string {
	return fmt.Sprintf("table<%s, %s>", t.KeyType.String(), t.ValueType.String())
}
func (t *TableType) Equals(other Type) bool {
	otherTable, ok := other.(*TableType)
	if !ok {
		return false
	}
	return t.KeyType.Equals(otherTable.KeyType) && t.ValueType.Equals(otherTable.ValueType)
}
func (t *TableType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Table is covariant in both key and value types
	if otherTable, ok := other.(*TableType); ok {
		return t.KeyType.IsAssignableTo(otherTable.KeyType) &&
			t.ValueType.IsAssignableTo(otherTable.ValueType)
	}
	return false
}

// FunctionType represents a function type
type FunctionType struct {
	Parameters []Type
	ReturnType Type
}

func (t *FunctionType) String() string {
	params := make([]string, len(t.Parameters))
	for i, p := range t.Parameters {
		params[i] = p.String()
	}
	return fmt.Sprintf("(%s) -> %s", strings.Join(params, ", "), t.ReturnType.String())
}
func (t *FunctionType) Equals(other Type) bool {
	otherFunc, ok := other.(*FunctionType)
	if !ok {
		return false
	}
	if len(t.Parameters) != len(otherFunc.Parameters) {
		return false
	}
	for i, param := range t.Parameters {
		if !param.Equals(otherFunc.Parameters[i]) {
			return false
		}
	}
	return t.ReturnType.Equals(otherFunc.ReturnType)
}
func (t *FunctionType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Functions are contravariant in parameters and covariant in return type
	if otherFunc, ok := other.(*FunctionType); ok {
		if len(t.Parameters) != len(otherFunc.Parameters) {
			return false
		}
		for i, param := range t.Parameters {
			// Contravariance: other's parameter must be assignable to this parameter
			if !otherFunc.Parameters[i].IsAssignableTo(param) {
				return false
			}
		}
		// Covariance: this return type must be assignable to other's return type
		return t.ReturnType.IsAssignableTo(otherFunc.ReturnType)
	}
	return false
}

// UnionType represents a union of multiple types
type UnionType struct {
	Types []Type
}

func (t *UnionType) String() string {
	typeStrs := make([]string, 0, len(t.Types))
	for _, typ := range t.Types {
		if typ != nil {
			typeStrs = append(typeStrs, typ.String())
		}
	}
	return strings.Join(typeStrs, " | ")
}
func (t *UnionType) Equals(other Type) bool {
	otherUnion, ok := other.(*UnionType)
	if !ok {
		return false
	}
	if len(t.Types) != len(otherUnion.Types) {
		return false
	}
	// Check if all types match (order-independent)
	for _, typ := range t.Types {
		found := false
		for _, otherTyp := range otherUnion.Types {
			if typ.Equals(otherTyp) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func (t *UnionType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// A union type is assignable to another type if all its members are assignable
	for _, typ := range t.Types {
		if !typ.IsAssignableTo(other) {
			return false
		}
	}
	return true
}

// Contains checks if the union contains a specific type
func (t *UnionType) Contains(typ Type) bool {
	for _, ut := range t.Types {
		if ut.Equals(typ) {
			return true
		}
	}
	return false
}

// OptionalType represents an optional type (T | nil)
type OptionalType struct {
	BaseType Type
}

func (t *OptionalType) String() string {
	return fmt.Sprintf("%s?", t.BaseType.String())
}
func (t *OptionalType) Equals(other Type) bool {
	otherOpt, ok := other.(*OptionalType)
	if !ok {
		return false
	}
	return t.BaseType.Equals(otherOpt.BaseType)
}
func (t *OptionalType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Optional type is assignable to another optional with compatible base
	if otherOpt, ok := other.(*OptionalType); ok {
		return t.BaseType.IsAssignableTo(otherOpt.BaseType)
	}
	// Optional is NOT assignable to non-optional (must unwrap first)
	return false
}

// GenericTypeAlias represents a generic type alias like type Nullable<T> = T | nil
type GenericTypeAlias struct {
	Name       string
	TypeParams []string       // e.g., ["T", "U"]
	Body       ast.Expression // the type expression with type parameters
}

func (t *GenericTypeAlias) String() string {
	params := strings.Join(t.TypeParams, ", ")
	return fmt.Sprintf("%s<%s>", t.Name, params)
}
func (t *GenericTypeAlias) Equals(other Type) bool {
	otherGeneric, ok := other.(*GenericTypeAlias)
	if !ok {
		return false
	}
	if t.Name != otherGeneric.Name {
		return false
	}
	if len(t.TypeParams) != len(otherGeneric.TypeParams) {
		return false
	}
	for i, param := range t.TypeParams {
		if param != otherGeneric.TypeParams[i] {
			return false
		}
	}
	return true
}
func (t *GenericTypeAlias) IsAssignableTo(other Type) bool {
	// Generic type aliases cannot be assigned directly; they must be instantiated first
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	return false
}

// TupleType represents a tuple type
type TupleType struct {
	Elements []Type
}

func (t *TupleType) String() string {
	elemStrs := make([]string, len(t.Elements))
	for i, elem := range t.Elements {
		elemStrs[i] = elem.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(elemStrs, ", "))
}
func (t *TupleType) Equals(other Type) bool {
	otherTuple, ok := other.(*TupleType)
	if !ok {
		return false
	}
	if len(t.Elements) != len(otherTuple.Elements) {
		return false
	}
	for i, elem := range t.Elements {
		if !elem.Equals(otherTuple.Elements[i]) {
			return false
		}
	}
	return true
}
func (t *TupleType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	if otherTuple, ok := other.(*TupleType); ok {
		if len(t.Elements) != len(otherTuple.Elements) {
			return false
		}
		for i, elem := range t.Elements {
			if !elem.IsAssignableTo(otherTuple.Elements[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// User-Defined Types

// ClassType represents a class type
type ClassType struct {
	Name             string
	Properties       map[string]Type           // Instance properties
	Methods          map[string]*FunctionType  // Instance methods
	StaticProperties map[string]Type           // Static properties
	StaticMethods    map[string]*FunctionType  // Static methods
	ReadonlyProps    map[string]bool           // Readonly property names
	AbstractMethods  map[string]bool           // Abstract method names
	Constructor      *FunctionType             // Constructor signature
	Implements       []*InterfaceType
	IsAbstract       bool                      // Whether class is abstract
}

func (t *ClassType) String() string {
	return t.Name
}
func (t *ClassType) Equals(other Type) bool {
	otherClass, ok := other.(*ClassType)
	if !ok {
		return false
	}
	return t.Name == otherClass.Name
}
func (t *ClassType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Class is assignable to interfaces it implements
	if otherInterface, ok := other.(*InterfaceType); ok {
		for _, impl := range t.Implements {
			if impl.Equals(otherInterface) {
				return true
			}
		}
	}
	return false
}

// GetProperty returns the type of a property
func (t *ClassType) GetProperty(name string) (Type, bool) {
	typ, ok := t.Properties[name]
	return typ, ok
}

// GetMethod returns the type of a method
func (t *ClassType) GetMethod(name string) (*FunctionType, bool) {
	typ, ok := t.Methods[name]
	return typ, ok
}

// GetStaticProperty returns the type of a static property
func (t *ClassType) GetStaticProperty(name string) (Type, bool) {
	typ, ok := t.StaticProperties[name]
	return typ, ok
}

// GetStaticMethod returns the type of a static method
func (t *ClassType) GetStaticMethod(name string) (*FunctionType, bool) {
	typ, ok := t.StaticMethods[name]
	return typ, ok
}

// IsReadonly checks if a property is readonly
func (t *ClassType) IsReadonly(name string) bool {
	return t.ReadonlyProps[name]
}

// IsAbstractMethod checks if a method is abstract
func (t *ClassType) IsAbstractMethod(name string) bool {
	return t.AbstractMethods[name]
}

// InterfaceType represents an interface type
type InterfaceType struct {
	Name       string
	Methods    map[string]*FunctionType
	Properties map[string]Type
	Extends    []*InterfaceType
}

func (t *InterfaceType) String() string {
	return t.Name
}
func (t *InterfaceType) Equals(other Type) bool {
	otherInterface, ok := other.(*InterfaceType)
	if !ok {
		return false
	}
	return t.Name == otherInterface.Name
}
func (t *InterfaceType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// Interface is assignable to interfaces it extends
	if otherInterface, ok := other.(*InterfaceType); ok {
		for _, ext := range t.Extends {
			if ext.Equals(otherInterface) {
				return true
			}
		}

		// Structural compatibility: check if this interface has all required properties
		// This allows table literals to be assigned to interface types
		for propName, propType := range otherInterface.Properties {
			myPropType, hasProperty := t.Properties[propName]
			if !hasProperty {
				return false // Missing required property
			}
			if !myPropType.IsAssignableTo(propType) {
				return false // Property type mismatch
			}
		}

		// Check methods (if any required)
		for methodName, methodType := range otherInterface.Methods {
			myMethodType, hasMethod := t.Methods[methodName]
			if !hasMethod {
				return false // Missing required method
			}
			if !myMethodType.IsAssignableTo(methodType) {
				return false // Method type mismatch
			}
		}

		// If we have all required properties and methods, we're compatible
		return true
	}
	return false
}

// GetMethod returns the type of a method
func (t *InterfaceType) GetMethod(name string) (*FunctionType, bool) {
	// Check own methods
	if method, ok := t.Methods[name]; ok {
		return method, true
	}
	// Check extended interfaces
	for _, ext := range t.Extends {
		if method, ok := ext.GetMethod(name); ok {
			return method, true
		}
	}
	return nil, false
}

// GetProperty returns the type of a property
func (t *InterfaceType) GetProperty(name string) (Type, bool) {
	// Check own properties
	if prop, ok := t.Properties[name]; ok {
		return prop, true
	}
	// Check extended interfaces
	for _, ext := range t.Extends {
		if prop, ok := ext.GetProperty(name); ok {
			return prop, true
		}
	}
	return nil, false
}

// EnumType represents an enum type
type EnumType struct {
	Name    string
	Members map[string]Type
}

func (t *EnumType) String() string {
	return t.Name
}
func (t *EnumType) Equals(other Type) bool {
	otherEnum, ok := other.(*EnumType)
	if !ok {
		return false
	}
	return t.Name == otherEnum.Name
}
func (t *EnumType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	return false
}

// HasMember checks if the enum has a specific member
func (t *EnumType) HasMember(name string) bool {
	_, ok := t.Members[name]
	return ok
}

// GetMemberType returns the type of an enum member
func (t *EnumType) GetMemberType(name string) (Type, bool) {
	typ, ok := t.Members[name]
	return typ, ok
}

// GenericType represents a generic type parameter
type GenericType struct {
	Name       string
	Constraint Type // Optional constraint
}

func (t *GenericType) String() string {
	if t.Constraint != nil {
		return fmt.Sprintf("%s: %s", t.Name, t.Constraint.String())
	}
	return t.Name
}
func (t *GenericType) Equals(other Type) bool {
	otherGeneric, ok := other.(*GenericType)
	if !ok {
		return false
	}
	return t.Name == otherGeneric.Name
}
func (t *GenericType) IsAssignableTo(other Type) bool {
	if t.Equals(other) {
		return true
	}
	if _, isAny := other.(*AnyType); isAny {
		return true
	}
	// If there's a constraint, check if it's assignable
	if t.Constraint != nil {
		return t.Constraint.IsAssignableTo(other)
	}
	return false
}

// Utility functions

// IsNumericType checks if a type is numeric
func IsNumericType(t Type) bool {
	_, ok := t.(*NumberType)
	return ok
}

// IsStringType checks if a type is a string
func IsStringType(t Type) bool {
	_, ok := t.(*StringType)
	return ok
}

// IsBooleanType checks if a type is boolean
func IsBooleanType(t Type) bool {
	_, ok := t.(*BooleanType)
	return ok
}

// IsNilType checks if a type is nil
func IsNilType(t Type) bool {
	_, ok := t.(*NilType)
	return ok
}

// IsVoidType checks if a type is void
func IsVoidType(t Type) bool {
	_, ok := t.(*VoidType)
	return ok
}

// Commonly used type instances
var (
	Number  = &NumberType{}
	String  = &StringType{}
	Boolean = &BooleanType{}
	Nil     = &NilType{}
	Void    = &VoidType{}
	Any     = &AnyType{}
)
