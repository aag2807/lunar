package types

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"lunar/internal/ast"
	"lunar/internal/lexer"
)

// reportLiteralFields checks a table literal written where a known interface
// (or an array of one) is expected, field by field, and reports which field is
// wrong: a value of the wrong type, a field the interface does not have, or
// required fields left out. It returns true when it checked the literal this
// way, whether or not it found anything: the literal's fit has then been
// decided, and the caller skips its generic "Cannot assign" check.
func (c *Checker) reportLiteralFields(expr ast.Expression, target Type) bool {
	lit, ok := expr.(*ast.TableLiteral)
	if !ok || target == nil {
		return false
	}
	handled, _ := c.checkLiteralAgainst(lit, target, target.String(), "")
	return handled
}

// checkLiteralAgainst reports whether it could check the literal against
// target (handled), and whether it found anything wrong (reported).
func (c *Checker) checkLiteralAgainst(lit *ast.TableLiteral, target Type, root, path string) (handled, reported bool) {
	if nonNil, optional := removeNil(target); optional {
		target = nonNil
	}

	switch t := target.(type) {
	case *InterfaceType:
		if t.Name == "<table literal>" || t.Name == "" {
			return false, false
		}
		return c.checkRecordLiteral(lit, t, root, path)
	case *ArrayType:
		if len(lit.Pairs) > 0 {
			return false, false
		}
		// Every element has to be a literal checked against the element
		// type for the array to count as handled.
		handled = true
		for i, value := range lit.Values {
			inner, ok := value.(*ast.TableLiteral)
			if !ok {
				handled = false
				continue
			}
			h, r := c.checkLiteralAgainst(inner, t.ElementType, root, fmt.Sprintf("%s[%d]", path, i+1))
			handled = handled && h
			reported = reported || r
		}
		return handled || reported, reported
	}
	return false, false
}

type literalField struct {
	key   *ast.Identifier
	value ast.Expression
	// quoted is set for a key written ["a"], which rename leaves alone.
	quoted bool
}

func (c *Checker) checkRecordLiteral(lit *ast.TableLiteral, iface *InterfaceType, root, path string) (handled, reported bool) {
	if len(lit.Values) > 0 {
		return false, false // positional values: not a record literal
	}

	fields := make([]literalField, 0, len(lit.Pairs))
	for key, value := range lit.Pairs {
		ident, ok := literalKeyName(key)
		if !ok {
			return false, false // computed keys: leave it to the generic check
		}
		_, quoted := key.(*ast.StringLiteral)
		fields = append(fields, literalField{ident, value, quoted})
	}
	// Pairs is a map; report in source order.
	sort.Slice(fields, func(i, j int) bool {
		a, b := fields[i].key.Token, fields[j].key.Token
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})

	written := map[string]bool{}
	for _, f := range fields {
		written[f.key.Value] = true
	}
	// Suggest only fields the literal does not already have.
	var members []string
	for _, name := range interfaceMemberNames(iface) {
		if !written[name] {
			members = append(members, name)
		}
	}
	required := iface.requiredProperties()
	for name, m := range iface.requiredMethods() {
		required[name] = m
	}
	present := map[string]bool{}
	explained := map[string]bool{}

	for _, f := range fields {
		name := f.key.Value
		present[name] = true
		label := joinPath(path, name)
		// The key names the interface's field, for references and rename.
		// (A quoted key's span includes its quotes, so it is not renamed.)
		if !f.quoted {
			c.reference(f.key.Token, memberSymbol(iface, name))
		}

		expected, known := iface.propertyType(name)
		isMethod := false
		if !known {
			if method, ok := iface.methodType(name); ok {
				// A method is called with ':', so the function a literal
				// gives for it receives the table first.
				expected, known, isMethod = withSelfParameter(method), true, true
			}
		}
		if !known && iface.IndexSignature != nil {
			expected, known = iface.IndexSignature.ValueType, true
		}
		if !known {
			msg := fmt.Sprintf("'%s' is not a field of '%s'", joinPath(path, name), root)
			if similar := findSimilarNames(name, members, suggestionDistance(name), 1); len(similar) > 0 {
				msg += fmt.Sprintf(". Did you mean '%s'?", joinPath(path, similar[0]))
				explained[similar[0]] = true
			}
			c.addErrorCode("unknown-field", msg, f.key.Token)
			reported = true
			continue
		}

		if inner, ok := f.value.(*ast.TableLiteral); ok {
			if h, r := c.checkLiteralAgainst(inner, expected, root, label); h {
				reported = reported || r
				continue
			}
		}

		actual := c.typeOf(f.value)
		if isMethod {
			fn, isFn := actual.(*FunctionType)
			params := len(expected.(*FunctionType).Parameters) - 1
			// With no parameters the receiver ':' passes is simply unused.
			if isFn && params == 0 && len(fn.Parameters) == 0 {
				continue
			}
			if isFn && len(fn.Parameters) == params {
				c.addErrorCode("field-type", fmt.Sprintf("Field '%s' of '%s' is a method, called with ':': its function takes self first, function(self, ...)",
					label, root), exprToken(f.value, f.key.Token))
				reported = true
				continue
			}
		}
		if actual != nil && !actual.IsAssignableTo(expected) {
			c.addErrorCode("field-type", fmt.Sprintf("Field '%s' of '%s' must be '%s', got '%s'",
				label, root, expected.String(), widenLiteralType(actual).String()), exprToken(f.value, f.key.Token))
			reported = true
		}
	}

	var missing []string
	for name := range required {
		if !present[name] && !explained[name] {
			missing = append(missing, "'"+joinPath(path, name)+"'")
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		c.addErrorCode("missing-fields", fmt.Sprintf("Missing field(s) %s required by '%s'", strings.Join(missing, ", "), root), lit.Token)
		reported = true
	}

	return true, reported
}

// withSelfParameter is a method's type as the function a table literal
// supplies for it: with the receiver as an extra first parameter.
func withSelfParameter(method *FunctionType) *FunctionType {
	fn := *method
	fn.Parameters = append([]Type{Any}, method.Parameters...)
	if len(method.OptionalParams) > 0 {
		fn.OptionalParams = append([]bool{false}, method.OptionalParams...)
	}
	if len(method.ParamNames) > 0 {
		fn.ParamNames = append([]string{"self"}, method.ParamNames...)
	}
	return &fn
}

// typeOf returns the type checkExpression computed for expr, checking it
// now only if it was never checked.
func (c *Checker) typeOf(expr ast.Expression) Type {
	if t, ok := c.exprTypes[expr]; ok {
		return t
	}
	return c.checkExpression(expr)
}

// interfaceMemberNames lists the properties and methods of an interface and
// everything it extends, sorted, for "did you mean" suggestions.
func interfaceMemberNames(iface *InterfaceType) []string {
	seen := map[string]bool{}
	var walk func(*InterfaceType)
	walk = func(it *InterfaceType) {
		if it == nil {
			return
		}
		for name := range it.Properties {
			seen[name] = true
		}
		for name := range it.Methods {
			seen[name] = true
		}
		for _, parent := range it.Extends {
			walk(parent)
		}
	}
	walk(iface)
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// suggestionDistance is how many edits a misspelling may be from the real
// name: none for one letter, where any other letter would match, up to two.
func suggestionDistance(name string) int {
	if d := len(name) / 2; d < 2 {
		return d
	}
	return 2
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// exprToken returns the token where an expression starts, or fallback.
func exprToken(expr ast.Expression, fallback lexer.Token) lexer.Token {
	switch e := expr.(type) {
	case *ast.InfixExpression:
		return exprToken(e.Left, fallback)
	case *ast.DotExpression:
		return exprToken(e.Left, fallback)
	case *ast.CallExpression:
		return exprToken(e.Function, fallback)
	case *ast.IndexExpression:
		return exprToken(e.Left, fallback)
	}
	v := reflect.ValueOf(expr)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		if f := v.Elem().FieldByName("Token"); f.IsValid() {
			if tok, ok := f.Interface().(lexer.Token); ok && tok.Line > 0 {
				return tok
			}
		}
	}
	return fallback
}
