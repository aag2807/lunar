package types

import (
	"fmt"

	"lunar/internal/ast"
)

// checkDecoratorTargets checks a target against the decorators applied to
// it. The Lua is `f = deco(f)`, the last decorator first, and the name keeps
// the target's type: so the target has to fit what each decorator takes,
// and what the decorator returns, which replaces it, has to fit the target.
func (c *Checker) checkDecoratorTargets(decorators []*ast.Decorator, target *FunctionType, name string) {
	for _, d := range decorators {
		fn := c.decoratorFunction(d, target)
		if fn == nil {
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
		if fn, ok = fn.ReturnType.(*FunctionType); !ok {
			return nil
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
