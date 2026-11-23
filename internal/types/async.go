package types

import (
	"fmt"
	"strings"

	"lunar/internal/ast"
	"lunar/internal/lexer"
)

// An async function may pause the coroutine it runs in (it yields, in the
// end, and something such as the engine resumes it later). Lua's
// coroutines pause from any depth of calls, so the Lua needs nothing: async
// and await are erased, as types are. What the checker adds is that an
// async function is only ever called with await, inside another async
// function, so that nothing pauses outside a coroutine.

// fnContext is the function the checker is inside.
type fnContext struct {
	inFunction bool
	async      bool
	name       string // "" for a function with no name
}

// calleeKind is what checking a call written after await found it calls.
type calleeKind int

const (
	calleeUnchecked calleeKind = iota
	calleeAsync
	calleePlain
	calleeUnknown // typed any: it may or may not pause
)

// enterFunction marks the start of a function body; the func it returns
// marks its end.
func (c *Checker) enterFunction(async bool, name string) func() {
	prev := c.fn
	c.fn = fnContext{inFunction: true, async: async, name: name}
	return func() { c.fn = prev }
}

// notAsync says why await, or a call to an async function, is not allowed
// where the checker is.
func (c *Checker) notAsync() string {
	switch {
	case !c.fn.inFunction:
		return "the top level of a file is not async"
	case c.fn.name != "":
		return fmt.Sprintf("'%s' is not async", c.fn.name)
	}
	return "the function it is in is not async"
}

// asyncCallee is the async function a callee's type is, or nil.
func asyncCallee(t Type) *FunctionType {
	switch f := t.(type) {
	case *FunctionType:
		if f.Async {
			return f
		}
	case *OptionalType:
		return asyncCallee(f.BaseType)
	}
	return nil
}

// calleeName is the name a call calls: wait in wait(1), walk in g:walk().
func calleeName(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Value
	case *ast.DotExpression:
		if id, ok := n.Right.(*ast.Identifier); ok {
			return id.Value
		}
	}
	return exprText(e)
}

// calleeToken is the token of the name a call calls, where an error about
// calling it belongs.
func calleeToken(node *ast.CallExpression) lexer.Token {
	switch n := node.Function.(type) {
	case *ast.Identifier:
		return n.Token
	case *ast.DotExpression:
		if id, ok := n.Right.(*ast.Identifier); ok {
			return id.Token
		}
	}
	return node.Token
}

// calleeText is a callee as written, for the fix a message suggests.
func calleeText(e ast.Expression) string {
	if dot, ok := e.(*ast.DotExpression); ok && dot.IsMethodCall {
		return exprText(dot.Left) + ":" + calleeName(dot)
	}
	return exprText(e)
}

// checkAsyncCall holds a call to an async function to being awaited. A
// call that is awaited is recorded instead, for checkAwaitExpression.
func (c *Checker) checkAsyncCall(node *ast.CallExpression, funcType Type) {
	kind := calleePlain
	if asyncCallee(funcType) != nil {
		kind = calleeAsync
	} else if _, isAny := funcType.(*AnyType); isAny {
		kind = calleeUnknown
	}
	if _, awaited := c.awaited[node]; awaited {
		c.awaited[node] = kind
		return
	}
	if kind != calleeAsync {
		return
	}
	name := calleeName(node.Function)
	if c.fn.async {
		c.addErrorCode("async", fmt.Sprintf("'%s' can pause: write 'await %s(...)'", name, calleeText(node.Function)), calleeToken(node))
		return
	}
	c.addErrorCode("async", fmt.Sprintf("'%s' can pause, so it can only be called with 'await' inside an async function; %s",
		name, c.notAsync()), calleeToken(node))
}

// checkAwaitExpression checks `await f()`: inside an async function, on a
// call to one. Its type is the call's, every result of it.
func (c *Checker) checkAwaitExpression(node *ast.AwaitExpression) Type {
	call, isCall := node.Expression.(*ast.CallExpression)
	if isCall {
		if c.awaited == nil {
			c.awaited = make(map[*ast.CallExpression]calleeKind)
		}
		if _, seen := c.awaited[call]; !seen {
			c.awaited[call] = calleeUnchecked
		}
	}
	t := c.checkExpression(node.Expression)
	switch {
	case !c.fn.async:
		c.addErrorCode("async", "'await' is only allowed inside an async function; "+c.notAsync(), node.Token)
	case !isCall:
		c.addErrorCode("async", fmt.Sprintf("'await' needs a call to an async function: '%s' is not a call", exprText(node.Expression)), node.Token)
	case c.awaited[call] == calleePlain:
		c.addErrorCode("async", fmt.Sprintf("'await' needs a call to an async function: '%s' does not pause", calleeName(call.Function)), node.Token)
	}
	return t
}

// reportAsyncMismatch reports an async function given where a plain one is
// expected, when being async is what keeps it from fitting, and says
// whether it did.
func (c *Checker) reportAsyncMismatch(value, target Type, tok lexer.Token) bool {
	fn, ok := value.(*FunctionType)
	if !ok || !fn.Async {
		return false
	}
	plain := *fn
	plain.Async = false
	if !plain.IsAssignableTo(target) {
		return false
	}
	c.addErrorCode("async", "an async function cannot be used where a plain function is expected: "+
		"whatever calls it would not await it, and it would pause outside a coroutine", tok)
	return true
}

// checkAsyncOverride refuses an async method in place of a plain one:
// code written against the parent or interface calls it without await.
func (c *Checker) checkAsyncOverride(name string, method Type, owner string, base Type, baseName string, tok lexer.Token) {
	m, ok := method.(*FunctionType)
	if !ok || !m.Async {
		return
	}
	if b, ok := base.(*FunctionType); ok && !b.Async {
		c.addErrorCode("async", fmt.Sprintf("'%s' is async in '%s' but not in '%s': callers of '%s' would call it without await",
			name, owner, baseName, baseName), tok)
	}
}

// checkAsyncClassMembers refuses async on what runs without a call that
// could await it: constructors, accessors and metamethods.
func (c *Checker) checkAsyncClassMembers(node *ast.ClassDeclaration) {
	if node.Constructor != nil && node.Constructor.IsAsync {
		c.addErrorCode("async", fmt.Sprintf("a constructor cannot be async: %s(...) and %s.new(...) are not awaited",
			node.Name.Value, node.Name.Value), node.Constructor.Token)
	}
	for _, g := range node.Getters {
		if g.IsAsync {
			c.addErrorCode("async", fmt.Sprintf("a getter cannot be async: reading '%s' cannot pause", g.Name.Value), g.Token)
		}
	}
	for _, s := range node.Setters {
		if s.IsAsync {
			c.addErrorCode("async", fmt.Sprintf("a setter cannot be async: assigning '%s' cannot pause", s.Name.Value), s.Token)
		}
	}
	for _, m := range node.Methods {
		if m.IsAsync && strings.HasPrefix(m.Name.Value, "__") {
			c.addErrorCode("async", fmt.Sprintf("a metamethod cannot be async: '%s' runs without a call that could await it", m.Name.Value), m.Name.Token)
		}
	}
}
