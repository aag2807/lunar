package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lunar/internal/ast"
	"lunar/internal/diag"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
)

// Module is one checked .lunar file.
type Module struct {
	Path        string // absolute file path
	Rel         string // relative to the root, slash-separated: "ai/brain.lunar"
	RequireName string // what require() loads it by: "ai.brain"
	Source      string
	Statements  []ast.Statement
	Exports     map[string]types.Type
	Symbols     map[string]*types.Symbol // where each export is defined
	// Index is what the checker defined and resolved in this module.
	Index *types.Index
	// Emit is false for a module that was only loaded because something
	// imports it, when compiling a single file.
	Emit bool
	// Broken is set when the module does not parse: it is not checked or
	// emitted, and its importers take its names as 'any'.
	Broken bool
	// checked is set once its exports are known.
	checked bool

	imports []resolvedImport
}

type resolvedImport struct {
	node   *ast.ImportStatement
	target *Module
	err    error
}

// Program is a checked project: its modules in dependency order.
type Program struct {
	Project *Project
	Modules []*Module
	// DeclIndex covers the declaration files, checked on their own.
	DeclIndex *types.Index
}

// Check parses and checks the declarations and every module, following
// imports. Diagnostics come back sorted; the Program is usable for Emit
// only when none of them is an error.
func (p *Project) Check() (*Program, diag.List) {
	var diags diag.List

	// Declarations are checked once, together; per module they are only
	// context, so their errors are not repeated for every script.
	declSources := make([]string, len(p.DeclFiles))
	var declStatements []ast.Statement
	for i, path := range p.DeclFiles {
		text, err := p.read(path)
		if err != nil {
			diags = append(diags, ioDiag(path, err))
			continue
		}
		declSources[i] = text
		statements, parseDiags := parseSource(path, text)
		diags = append(diags, parseDiags...)
		declStatements = append(declStatements, statements...)
	}
	declChecker := p.newChecker()
	diags = append(diags, typeDiags(declChecker.Check(declStatements))...)

	// Load every module reachable from the sources.
	byPath := map[string]*Module{}
	var order []*Module
	var load func(path string, emit bool) *Module
	load = func(path string, emit bool) *Module {
		if m, ok := byPath[path]; ok {
			m.Emit = m.Emit || emit
			return m
		}
		m := &Module{Path: path, Rel: p.rel(path), Emit: emit}
		byPath[path] = m
		order = append(order, m)

		text, err := p.read(path)
		if err != nil {
			diags = append(diags, ioDiag(path, err))
			return m
		}
		m.Source = text
		m.Statements, err = nil, nil
		statements, parseDiags := parseSource(path, text)
		m.Statements = statements
		m.Broken = parseDiags.HasErrors()
		diags = append(diags, parseDiags...)

		name, nameErr := requireName(m.Rel)
		m.RequireName = name
		if nameErr != nil {
			diags = append(diags, diag.Diagnostic{File: path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1,
				Severity: diag.Error, Code: "module-name", Message: nameErr.Error()})
		}

		for _, stmt := range statements {
			imp, ok := stmt.(*ast.ImportStatement)
			if !ok {
				continue
			}
			targetPath, err := p.resolve(path, imp.Module)
			ri := resolvedImport{node: imp, err: err}
			if err == nil {
				ri.target = load(targetPath, false)
				imp.RequireName = ri.target.RequireName
			}
			m.imports = append(m.imports, ri)
		}
		return m
	}
	for _, src := range p.Sources {
		load(src, true)
	}

	// Check in dependency order, so a module's exports are known before
	// anything that imports it. A cycle is reported where it closes.
	sorted, cycleDiags := topoSort(order)
	diags = append(diags, cycleDiags...)

	for _, m := range sorted {
		checker := p.newChecker()
		checker.SetImportResolver(func(node *ast.ImportStatement) (*types.ModuleInfo, error) {
			for _, ri := range m.imports {
				if ri.node != node {
					continue
				}
				if ri.err != nil {
					return nil, ri.err
				}
				exports := ri.target.Exports
				if exports == nil {
					exports = map[string]types.Type{}
				}
				// A module not checked yet is on an import cycle, which is
				// reported already; its names are taken as 'any' meanwhile.
				unknown := ri.target.Broken || !ri.target.checked
				return &types.ModuleInfo{Path: ri.target.Path, Exports: exports, Symbols: ri.target.Symbols, Broken: unknown}, nil
			}
			return nil, fmt.Errorf("Cannot find module '%s'", node.Module)
		})

		// A module that does not parse is reported by its syntax errors
		// alone; checking a half-built tree only produces noise, or worse.
		if m.Broken {
			continue
		}
		diags = append(diags, p.checkModule(m, checker, declSources)...)
		m.checked = true
	}

	diags.Sort()
	return &Program{Project: p, Modules: sorted, DeclIndex: declChecker.Index()}, diags
}

// checkModule checks one module with the declarations, recording its
// exports and index. A checker panic is turned into an internal error on
// the module rather than taking the process down.
func (p *Project) checkModule(m *Module, checker *types.Checker, declSources []string) (diags diag.List) {
	defer func() {
		if r := recover(); r != nil {
			diags = append(diags, diag.Diagnostic{File: m.Path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1,
				Severity: diag.Error, Code: "internal", Message: fmt.Sprintf("internal compiler error checking this file: %v", r)})
			m.Broken = true
		}
	}()

	// The declarations again, freshly parsed: the checker annotates the
	// trees it checks, so modules do not share them.
	var statements []ast.Statement
	for i, path := range p.DeclFiles {
		decls, _ := parseSource(path, declSources[i])
		statements = append(statements, decls...)
	}
	statements = append(statements, m.Statements...)

	for _, e := range checker.Check(statements) {
		if e.File == m.Path {
			diags = append(diags, typeDiag(e))
		}
	}
	m.Exports = checker.Exports()
	m.Symbols = checker.ExportSymbols()
	m.Index = checker.Index()
	if p.isDataFile(m.Rel) {
		diags = append(diags, dataFileDiagnostics(m, checker)...)
	}
	return diags
}

func (p *Project) newChecker() *types.Checker {
	return types.NewCheckerWithOptions(types.Options{Target: p.Target})
}

// resolve finds the file an import names. "./x" and "../x" are relative to
// the importing file; anything else is relative to the root. Either way the
// module has to be a .lunar file inside the root.
func (p *Project) resolve(from, spec string) (string, error) {
	base := p.Root
	if strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		base = filepath.Dir(from)
	}
	target := filepath.Clean(filepath.Join(base, filepath.FromSlash(strings.TrimSuffix(spec, ".lunar"))))

	if rel, err := filepath.Rel(p.Root, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("Cannot import '%s': it is outside the project root", spec)
	}

	candidates := []string{target + ".lunar", filepath.Join(target, "init.lunar")}
	for _, c := range candidates {
		if _, ok := p.overlay[c]; ok {
			return c, nil
		}
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}
	if info, err := os.Stat(target + ".lua"); err == nil && !info.IsDir() {
		name, _ := requireName(p.rel(target + ".lua"))
		return "", fmt.Errorf("Cannot import '%s': it is a plain Lua file; load it with require(\"%s\")", spec, name)
	}
	return "", fmt.Errorf("Cannot find module '%s' (looked for %s)", spec, p.rel(candidates[0]))
}

// requireName turns a root-relative path into the dotted name require()
// takes: "ai/brain.lunar" is "ai.brain", and "ai/init.lunar" is "ai".
func requireName(rel string) (string, error) {
	name := strings.TrimSuffix(strings.TrimSuffix(rel, ".lunar"), ".lua")
	name = strings.TrimSuffix(name, "/init")
	for _, part := range strings.Split(name, "/") {
		if strings.Contains(part, ".") {
			return name, fmt.Errorf("'%s' cannot be loaded by require: '.' in a file or folder name means a subfolder", rel)
		}
	}
	return strings.ReplaceAll(name, "/", "."), nil
}

// topoSort orders modules so that each comes after what it imports, and
// reports every import that closes a cycle.
func topoSort(modules []*Module) ([]*Module, diag.List) {
	const (
		unvisited = iota
		visiting
		done
	)
	state := map[*Module]int{}
	var stack []*Module
	var sorted []*Module
	var diags diag.List

	var visit func(m *Module)
	visit = func(m *Module) {
		state[m] = visiting
		stack = append(stack, m)
		for _, ri := range m.imports {
			if ri.target == nil {
				continue
			}
			switch state[ri.target] {
			case unvisited:
				visit(ri.target)
			case visiting:
				var names []string
				for i := len(stack) - 1; i >= 0; i-- {
					names = append([]string{stack[i].Rel}, names...)
					if stack[i] == ri.target {
						break
					}
				}
				names = append(names, ri.target.Rel)
				tok := ri.node.Token
				diags = append(diags, diag.Diagnostic{File: tok.File, Line: tok.Line, Column: tok.Column,
					EndLine: tok.EndLine, EndColumn: tok.EndColumn, Severity: diag.Error, Code: "import-cycle",
					Message: "Import cycle: " + strings.Join(names, " → ")})
			}
		}
		stack = stack[:len(stack)-1]
		state[m] = done
		sorted = append(sorted, m)
	}
	for _, m := range modules {
		if state[m] == unvisited {
			visit(m)
		}
	}
	return sorted, diags
}

func parseSource(path, text string) ([]ast.Statement, diag.List) {
	p := parser.New(lexer.NewFile(path, text))
	statements := p.Parse()
	var diags diag.List
	for _, e := range p.Errors() {
		diags = append(diags, diag.Diagnostic{File: e.File, Line: e.Line, Column: e.Column, EndLine: e.EndLine,
			EndColumn: e.EndColumn, Severity: diag.Error, Code: e.Code, Message: e.Message})
	}
	return statements, diags
}

func typeDiag(e *types.TypeError) diag.Diagnostic {
	code := e.Code
	if code == "" {
		code = "type"
	}
	return diag.Diagnostic{File: e.File, Line: e.Line, Column: e.Column, EndLine: e.EndLine,
		EndColumn: e.EndColumn, Severity: diag.Error, Code: code, Message: e.Message}
}

func typeDiags(errs []*types.TypeError) diag.List {
	out := make(diag.List, len(errs))
	for i, e := range errs {
		out[i] = typeDiag(e)
	}
	return out
}

func ioDiag(path string, err error) diag.Diagnostic {
	var pathErr *os.PathError
	msg := err.Error()
	if errors.As(err, &pathErr) {
		msg = fmt.Sprintf("cannot read %s: %v", path, pathErr.Err)
	}
	return diag.Diagnostic{File: path, Severity: diag.Error, Code: "io", Message: msg}
}
