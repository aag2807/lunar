package types

import (
	"fmt"

	"lunar/internal/ast"
)

// checkDecoratorTargets checks a target against the decorators applied to
// it. The Lua is `f = deco(f)`, the last decorator first, and the name keeps
// the target's type: so the target has to fit what each decorator takes,
// and what the decorator returns, which replaces it, has to fit the target.
func (c *Checker) checkDecoratorTargets(decorators []*ast.Decorator, target *FunctionType, name, what string) {
	for _, d := range decorators {
		fn := c.decoratorFunction(d, target)
		if fn == nil {
			continue
		}
		if decoratesFields(fn) {
			c.addError(fmt.Sprintf("Decorator '%s' decorates fields: it takes (class, name, info). '%s' is a %s, which a decorator is given to replace",
				d.Name.Value, name, what), d.Token)
			continue
		}
		params, rest := splitRest(fn)
		var param Type
		switch {
		case len(params) > 0:
			param = params[0]
		case rest != nil:
			param = rest
		default:
			continue // checkDecorator says it takes no target
		}
		deco := d.Name.Value
		if pf, ok := param.(*FunctionType); ok && target.Async && !pf.Async {
			c.addErrorCode("async", fmt.Sprintf("'%s' is async, and decorator '%s' takes a plain function: it would call it without await",
				name, deco), d.Token)
		} else if !target.IsAssignableTo(param) {
			c.addError(fmt.Sprintf("Decorator '%s' takes '%s', which '%s' of type '%s' does not fit",
				deco, param.String(), name, target.String()), d.Token)
		}
		ret := fn.ReturnType
		if ret == nil {
			continue
		}
		if rf, ok := ret.(*FunctionType); ok && rf.Async && !target.Async {
			c.addErrorCode("async", fmt.Sprintf("Decorator '%s' returns an async function, which cannot replace '%s', a plain one: its callers would not await it",
				deco, name), d.Token)
		} else if !ret.IsAssignableTo(target) {
			c.addError(fmt.Sprintf("Decorator '%s' returns '%s', which cannot replace '%s' of type '%s'",
				deco, ret.String(), name, target.String()), d.Token)
		}
	}
}

// decoratorFunction is the function a decorator applies to its target: the
// decorator itself, or what @deco(args) returns, a generic one instantiated
// for the target (deco<T>(f: T): T). It is nil when that is not known (any)
// or not a function, which checkDecorator reports.
func (c *Checker) decoratorFunction(d *ast.Decorator, target Type) *FunctionType {
	t, ok := c.env.Get(d.Name.Value)
	if !ok {
		return nil
	}
	fn, ok := t.(*FunctionType)
	if !ok {
		return nil
	}
	if len(d.Arguments) > 0 {
		if fn, ok = c.decoratorOverload(fn, d).ReturnType.(*FunctionType); !ok {
			return nil
		}
	} else if len(fn.Overloads) > 0 {
		// A bare @deco declared with several overloads: the one for this
		// target, a field's (class, name, info) or a function.
		forField := target == nil
		for _, sig := range append([]*FunctionType{fn}, fn.Overloads...) {
			if decoratesFields(sig) == forField {
				fn = sig
				break
			}
		}
	}
	if len(fn.GenericParams) > 0 && len(fn.Parameters) > 0 {
		args := make([]Type, len(fn.GenericParams))
		inferred := make([]bool, len(fn.GenericParams))
		inferTypeArgs(fn.Parameters[0], target, fn.GenericParams, args, inferred)
		substitutions := make(map[string]Type, len(fn.GenericParams))
		for i, name := range fn.GenericParams {
			substitutions[name] = Any
			if inferred[i] {
				substitutions[name] = args[i]
			}
		}
		fn = c.substituteFunctionType(fn, substitutions)
	}
	return fn
}

// decoratorOverload is the signature of a declared decorator that
// @deco(args)'s arguments match, or its first when none does (whose errors
// checkDecorator then reports).
func (c *Checker) decoratorOverload(fn *FunctionType, d *ast.Decorator) *FunctionType {
	if len(fn.Overloads) == 0 {
		return fn
	}
	args := make([]Type, len(d.Arguments))
	for i, arg := range d.Arguments {
		args[i] = c.typeOf(arg)
	}
	for _, sig := range append([]*FunctionType{fn}, fn.Overloads...) {
		if c.overloadMatches(sig, args) {
			return sig
		}
	}
	return fn
}

// checkFieldDecorators checks the decorators of a class's fields. Each is
// called once, as the class is defined, with deco(Class, "name", info),
// and what it returns is ignored: it has to take those three.
func (c *Checker) checkFieldDecorators(node *ast.ClassDeclaration, classType *ClassType) {
	call := &FunctionType{Parameters: []Type{classType, String, fieldInfoType()}, ReturnType: Void}
	for _, prop := range node.Properties {
		if prop == nil || prop.Name == nil {
			continue
		}
		if len(prop.Decorators) > 0 {
			declared, ok := classType.Properties[prop.Name.Value]
			if prop.IsStatic {
				declared, ok = classType.StaticProperties[prop.Name.Value]
			}
			if ok && prop.Type != nil {
				prop.Choices = StringChoices(declared)
			}
		}
		for _, d := range prop.Decorators {
			c.checkDecorator(d)
			fn := c.decoratorFunction(d, nil)
			if fn == nil {
				continue
			}
			deco, field := d.Name.Value, prop.Name.Value
			switch {
			case !decoratesFields(fn) && decoratesFunctions(fn):
				c.addError(fmt.Sprintf("Decorator '%s' decorates functions: it takes the function it replaces. '%s' is a field, whose decorators take (class, name, info)",
					deco, field), d.Token)
			case fn.Async:
				c.addErrorCode("async", fmt.Sprintf("Decorator '%s' on field '%s' is async: it is called when the class is defined, where nothing awaits it",
					deco, field), d.Token)
			case !fn.parametersFit(call):
				c.addError(fmt.Sprintf("Decorator '%s' on field '%s' must accept (class: any, name: string, info: table), but takes '%s'",
					deco, field, fn.String()), d.Token)
			}
		}
	}
}

// StringChoices is what a union of string literals allows, in the order
// written ("a" | "b", an alias of one, either with ?), or nil.
func StringChoices(t Type) []string {
	if base, ok := removeNil(t); ok {
		t = base
	}
	if opt, ok := t.(*OptionalType); ok {
		t = opt.BaseType
	}
	union, ok := t.(*UnionType)
	if !ok {
		return nil
	}
	var choices []string
	for _, m := range union.Types {
		switch lit := m.(type) {
		case *StringLiteralType:
			choices = append(choices, lit.Value)
		case *NilType:
		default:
			return nil
		}
	}
	return choices
}

// fieldInfoType is the type of the info a field decorator is given:
// { type?: string, default?: any, has_default: boolean }.
func fieldInfoType() *InterfaceType {
	return &InterfaceType{
		Name: "field info",
		Properties: map[string]Type{"type": &OptionalType{BaseType: String}, "default": Any, "has_default": Boolean,
			"choices": &OptionalType{BaseType: &ArrayType{ElementType: String}}, "line": Number,
			"file": &OptionalType{BaseType: String}, "doc": &OptionalType{BaseType: String}},
		OptionalProps: map[string]bool{"type": true, "default": true, "choices": true, "file": true, "doc": true},
		Methods:       map[string]*FunctionType{},
		StaticProps:   map[string]bool{},
	}
}

// decoratesFields says whether a decorator takes what a field's is given,
// (class, name, info): a second parameter that has to be passed.
func decoratesFields(fn *FunctionType) bool {
	params, _ := splitRest(fn)
	if len(params) < 2 {
		return false
	}
	return fn.Untyped || len(fn.OptionalParams) < 2 || !fn.OptionalParams[1]
}

// decoratesFunctions says whether a decorator takes what a function's or
// a method's is given: the function, first.
func decoratesFunctions(fn *FunctionType) bool {
	params, _ := splitRest(fn)
	return len(params) >= 1
}

// methodValue is a method as the value a decorator is given: a function
// whose first parameter is self, unless the method is static.
func methodValue(method *FunctionType, self Type, static bool) *FunctionType {
	if static {
		return method
	}
	value := *method
	value.Parameters = append([]Type{self}, method.Parameters...)
	if len(method.OptionalParams) > 0 {
		value.OptionalParams = append([]bool{false}, method.OptionalParams...)
	}
	if len(method.ParamNames) > 0 {
		value.ParamNames = append([]string{"self"}, method.ParamNames...)
	}
	return &value
}
