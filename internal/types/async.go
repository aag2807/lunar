package types

import (
	"fmt"
	"sort"
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
	// what, if set, is what is not async, for messages: "the initial
	// value of 'x'".
	what string
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
	case c.fn.what != "":
		return c.fn.what + " is not async"
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
	return exprDescription(e)
}

// exprDescription is an expression as written, near enough for a message:
// wait, g.walk, g:walk, num() ^ 2.
func exprDescription(e ast.Expression) string {
	switch n := e.(type) {
	case *ast.Identifier:
		return n.Value
	case *ast.SuperExpression:
		return "super"
	case *ast.DotExpression:
		sep := "."
		if n.IsMethodCall {
			sep = ":"
		}
		return exprDescription(n.Left) + sep + exprDescription(n.Right)
	case *ast.CallExpression:
		return exprDescription(n.Function) + "()"
	}
	if isNilNode(e) {
		return "?"
	}
	text := e.String()
	if _, infix := e.(*ast.InfixExpression); infix && strings.HasPrefix(text, "(") && strings.HasSuffix(text, ")") {
		text = text[1 : len(text)-1]
	}
	return text
}

// headCall is the call an expression begins with: num() in num() ^ 2 and
// in f().x, or nil.
func headCall(e ast.Expression) *ast.CallExpression {
	switch n := e.(type) {
	case *ast.CallExpression:
		return n
	case *ast.InfixExpression:
		return headCall(n.Left)
	case *ast.DotExpression:
		return headCall(n.Left)
	case *ast.IndexExpression:
		return headCall(n.Left)
	}
	return nil
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
	return exprDescription(e)
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
	// In await num() ^ 2, await applies to all of num() ^ 2, as - would;
	// its error says so, in place of one about calling num.
	head := headCall(node.Expression)
	if c.awaited == nil {
		c.awaited = make(map[*ast.CallExpression]calleeKind)
	}
	if head != nil {
		if _, seen := c.awaited[head]; !seen {
			c.awaited[head] = calleeUnchecked
		}
	}
	t := c.checkExpression(node.Expression)
	switch {
	case !c.fn.async:
		c.addErrorCode("async", "'await' is only allowed inside an async function; "+c.notAsync(), node.Token)
	case !isCall && head != nil && c.awaited[head] == calleeAsync:
		c.addErrorCode("async", fmt.Sprintf("'await' applies to all of '%s', which is not a call; write '(await %s)' for the call's result",
			exprDescription(node.Expression), exprDescription(head)), node.Token)
	case !isCall:
		c.addErrorCode("async", fmt.Sprintf("'await' needs a call to an async function: '%s' is not a call", exprDescription(node.Expression)), node.Token)
	case c.awaited[call] == calleePlain:
		c.addErrorCode("async", fmt.Sprintf("'await' needs a call to an async function: '%s' does not pause", calleeName(call.Function)), node.Token)
	}
	return t
}

// reportAsyncMismatch reports an async function given where a plain one is
// expected, when being async is what keeps it from fitting, and says
// whether it did.
func (c *Checker) reportAsyncMismatch(value, target Type, tok lexer.Token) bool {
	if iface, ok := target.(*InterfaceType); ok {
		return c.reportAsyncMember(value, iface, tok)
	}
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

// reportAsyncMember reports a value given where an interface is expected
// whose member is async where the interface's is plain, a method or a
// function field, and says whether there was one.
func (c *Checker) reportAsyncMember(value Type, iface *InterfaceType, tok lexer.Token) bool {
	member := func(name string) Type {
		switch v := value.(type) {
		case *ClassType:
			if m, ok := v.GetMethod(name); ok {
				return m
			}
			if p, ok := v.GetProperty(name); ok {
				return p
			}
		case *InterfaceType:
			if m, ok := v.Methods[name]; ok {
				return m
			}
			if p, ok := v.Properties[name]; ok {
				return p
			}
		}
		return nil
	}
	// The interface's members, its own over those it inherits.
	expected := make(map[string]Type)
	var collect func(i *InterfaceType)
	collect = func(i *InterfaceType) {
		if i == nil {
			return
		}
		for name, m := range i.Methods {
			if _, seen := expected[name]; !seen {
				expected[name] = m
			}
		}
		for name, p := range i.Properties {
			if _, seen := expected[name]; !seen {
				expected[name] = p
			}
		}
		for _, parent := range i.Extends {
			collect(parent)
		}
	}
	collect(iface)
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		want, ok := expected[name].(*FunctionType)
		if !ok || want.Async {
			continue
		}
		if got, ok := member(name).(*FunctionType); ok && got.Async {
			what := "this table"
			if cl, isClass := value.(*ClassType); isClass {
				what = "'" + cl.Name + "'"
			}
			c.addErrorCode("async", fmt.Sprintf("%s cannot be used as '%s': its '%s' is async, and callers of '%s' would call it without await",
				what, iface.Name, name, iface.Name), tok)
			return true
		}
	}
	return false
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

// checkAsyncInterfaceExtends refuses an interface that makes a method it
// inherits async: code written against the parent calls it without await.
func (c *Checker) checkAsyncInterfaceExtends(iface *InterfaceType, tok lexer.Token) {
	for name, m := range iface.Methods {
		if !m.Async {
			continue
		}
		for _, parent := range iface.Extends {
			if base, ok := interfaceMethod(parent, name); ok {
				c.checkAsyncOverride(name, m, iface.Name, base, parent.Name, tok)
			}
		}
	}
}

// interfaceMethod finds a method on an interface or what it extends.
func interfaceMethod(iface *InterfaceType, name string) (*FunctionType, bool) {
	if iface == nil {
		return nil, false
	}
	if m, ok := iface.Methods[name]; ok {
		return m, true
	}
	for _, parent := range iface.Extends {
		if m, ok := interfaceMethod(parent, name); ok {
			return m, true
		}
	}
	return nil, false
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

// checkFieldValues checks the initial values of a class's fields against
// their types. An instance field's value is set by the constructor, so it
// is checked as the constructor's code (with self, and not async); a
// static field's where the class is defined.
func (c *Checker) checkFieldValues(node *ast.ClassDeclaration, classType *ClassType) {
	for _, prop := range node.Properties {
		if prop == nil || prop.Name == nil || isNilNode(prop.Value) {
			continue
		}
		name := prop.Name.Value
		declared, ok := classType.StaticProperties[name]
		if !prop.IsStatic {
			declared, ok = classType.Properties[name]
		}
		prevEnv := c.env
		c.env = NewEnclosedEnvironment(prevEnv)
		for _, gp := range node.GenericParams {
			c.env.Set(gp.Value, Any)
		}
		leave := func() {}
		if !prop.IsStatic {
			c.env.Set("self", classType)
			prev := c.fn
			c.fn = fnContext{inFunction: true, what: fmt.Sprintf("the initial value of '%s'", name)}
			leave = func() { c.fn = prev }
		}
		valueType := c.checkExpression(prop.Value)
		if ok && prop.Type != nil && !c.reportLiteralFields(prop.Value, declared) &&
			!valueType.IsAssignableTo(declared) && !c.reportAsyncMismatch(valueType, declared, prop.Name.Token) {
			c.addErrorCode("field-type", fmt.Sprintf("Field '%s' of '%s' must be '%s', got '%s'",
				name, classType.Name, declared.String(), widenLiteralType(valueType).String()), prop.Name.Token)
		}
		leave()
		c.env = prevEnv
	}
}

// addOverload adds a declared signature to a function's overloads. They are
// all async or none: whether a call has to be awaited cannot depend on
// which overload its arguments pick, nor on which one a value of the
// function is taken to be.
func (c *Checker) addOverload(fn, overload *FunctionType, tok lexer.Token) {
	if overload.Async != fn.Async {
		c.addErrorCode("async", fmt.Sprintf("'%s' is async in one overload and not in another: declare every overload async, or none",
			tok.Literal), tok)
	}
	fn.Overloads = append(fn.Overloads, overload)
}
