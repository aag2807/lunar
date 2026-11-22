package project

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"lunar/internal/ast"
	"lunar/internal/diag"
	"lunar/internal/lexer"
	"lunar/internal/types"
)

const noLibraries = "data files are loaded with no libraries opened"

// isDataFile reports whether a root-relative path matches a dataFiles glob.
// '*' matches within a folder name, '**' across folders.
func (p *Project) isDataFile(rel string) bool {
	for _, pattern := range p.DataFiles {
		if globRegexp(pattern).MatchString(rel) {
			return true
		}
	}
	return false
}

func globRegexp(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case c == '*' && i+1 < len(pattern) && pattern[i+1] == '*':
			sb.WriteString(".*")
			i++
		case c == '*':
			sb.WriteString("[^/]*")
		case c == '?':
			sb.WriteString("[^/]")
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}

// dataFileDiagnostics reports what in a data file needs the Lua state to
// supply something: a global from a declaration, or a construct the code
// generator implements with library functions.
func dataFileDiagnostics(m *Module, checker *types.Checker) diag.List {
	var diags diag.List
	add := func(tok lexer.Token, msg string) {
		diags = append(diags, diag.Diagnostic{File: m.Path, Line: tok.Line, Column: tok.Column, EndLine: tok.EndLine,
			EndColumn: tok.EndColumn, Severity: diag.Error, Code: "data-file", Message: msg + "; " + noLibraries})
	}
	for _, tok := range checker.GlobalUses() {
		if tok.File == m.Path {
			add(tok, fmt.Sprintf("Data file uses the global '%s'", tok.Literal))
		}
	}
	for _, stmt := range m.Statements {
		walk(reflect.ValueOf(stmt), func(n any) {
			switch node := n.(type) {
			case *ast.ClassDeclaration:
				add(node.Token, "Data file declares a class, which needs setmetatable")
			case *ast.SpreadExpression:
				add(node.Token, "Data file uses a spread, which needs table.unpack or a helper")
			case *ast.TemplateLiteral:
				add(node.Token, "Data file uses a template string, which needs tostring")
			case *ast.MatchExpression:
				add(node.Token, "Data file uses match, which needs error")
			}
		})
	}
	return diags
}

var nodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()

// walk visits every AST node reachable from v.
func walk(v reflect.Value, visit func(any)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			walk(v.Elem(), visit)
		}
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Type().Implements(nodeType) {
			visit(v.Interface())
		}
		walk(v.Elem(), visit)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(lexer.Token{}) {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			walk(v.Field(i), visit)
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i), visit)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			walk(iter.Key(), visit)
			walk(iter.Value(), visit)
		}
	}
}
