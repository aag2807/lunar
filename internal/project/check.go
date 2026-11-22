package project

import (
	"crypto/sha256"
	"encoding/hex"
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
	// checked is set once its exports are known; reused when they were
	// taken from an earlier check.
	checked, reused bool
	// Diags are the module's own check diagnostics; parseDiags its syntax
	// errors.
	Diags      diag.List
	parseDiags diag.List

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

	// What a later check may reuse: everything a module's check depends
	// on besides its own text and imports, the declarations' own
	// diagnostics, and the modules by path.
	contextKey string
	declDiags  diag.List
	byPath     map[string]*Module
	// declStatements are the declaration files, parsed once: every
	// module's checker reads them (it writes only into expressions, which
	// declarations do not have, and the same values each time).
	declStatements []ast.Statement
}

// Check parses and checks the declarations and every module, following
// imports. Diagnostics come back sorted; the Program is usable for Emit
// only when none of them is an error.
func (p *Project) Check() (*Program, diag.List) {
	return p.CheckReusing(nil)
}

// CheckReusing is Check, taking from prev (an earlier check of the same
// project, as the language server makes on every edit) each module whose
// text is unchanged and whose imports were all taken from prev too, as
// long as the declarations and settings are the same. Its results are
// those of a full check.
func (p *Project) CheckReusing(prev *Program) (*Program, diag.List) {
	var diags diag.List

	// Declarations are checked once, together; per module they are only
	// context, so their errors are not repeated for every script.
	declSources := make([]string, len(p.DeclFiles))
	var declIO diag.List
	key := sha256.New()
	fmt.Fprintf(key, "%s\x00%s\x00%q\x00", p.Root, p.Target, p.DataFiles)
	for i, path := range p.DeclFiles {
		text, err := p.read(path)
		if err != nil {
			declIO = append(declIO, ioDiag(path, err))
			continue
		}
		declSources[i] = text
		fmt.Fprintf(key, "%s\x00%d\x00%s\x00", path, len(text), text)
	}
	contextKey := hex.EncodeToString(key.Sum(nil))
	if prev != nil && prev.contextKey != contextKey {
		prev = nil // other declarations or settings: nothing carries over
	}

	var declDiags diag.List
	var declIndex *types.Index
	var declStatements []ast.Statement
	if prev != nil {
		declDiags, declIndex, declStatements = prev.declDiags, prev.DeclIndex, prev.declStatements
	} else {
		for i, path := range p.DeclFiles {
			statements, parseDiags := parseSource(path, declSources[i])
			declDiags = append(declDiags, parseDiags...)
			declStatements = append(declStatements, statements...)
		}
		declChecker := p.newChecker()
		declDiags = append(declDiags, typeDiags(declChecker.Check(declStatements))...)
		declIndex = declChecker.Index()
	}
	diags = append(diags, declIO...)
	diags = append(diags, declDiags...)

	// Load every module reachable from the sources.
	byPath := map[string]*Module{}
	byName := map[string]*Module{}
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
		// Unchanged text needs no parsing again; the earlier tree also
		// carries the checker's annotations, should the module be reused.
		var statements []ast.Statement
		var parseDiags diag.List
		if pm := prev.module(path); pm != nil && pm.Source == text {
			statements, parseDiags = pm.Statements, pm.parseDiags
		} else {
			statements, parseDiags = parseSource(path, text)
		}
		m.Statements = statements
		m.parseDiags = parseDiags
		m.Broken = parseDiags.HasErrors()
		diags = append(diags, parseDiags...)

		name, nameErr := requireName(m.Rel)
		m.RequireName = name
		if nameErr != nil {
			diags = append(diags, diag.Diagnostic{File: path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1,
				Severity: diag.Error, Code: "module-name", Message: nameErr.Error()})
		}
		// a.lunar and a/init.lunar are both require("a"), and Lua loads
		// whichever its path finds first.
		if other, taken := byName[name]; taken && nameErr == nil {
			diags = append(diags, diag.Diagnostic{File: path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1,
				Severity: diag.Error, Code: "module-name", Message: fmt.Sprintf("%s and %s are both require(\"%s\"), and Lua loads only one of them; rename one", m.Rel, other.Rel, name)})
		} else if nameErr == nil {
			byName[name] = m
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
		if pm := prev.module(m.Path); pm != nil && reusable(m, pm) {
			m.Exports, m.Symbols, m.Index, m.Diags = pm.Exports, pm.Symbols, pm.Index, pm.Diags
			m.checked, m.reused = true, true
			diags = append(diags, m.Diags...)
			continue
		}
		m.Diags = p.checkModule(m, checker, declStatements)
		diags = append(diags, m.Diags...)
		m.checked = true
	}

	diags = append(diags, p.unmatchedDataFiles(sorted)...)
	diags.Sort()
	return &Program{Project: p, Modules: sorted, DeclIndex: declIndex,
		contextKey: contextKey, declDiags: declDiags, byPath: byPath, declStatements: declStatements}, diags
}

// module is the earlier check's module at path, if any.
func (prog *Program) module(path string) *Module {
	if prog == nil {
		return nil
	}
	return prog.byPath[path]
}

// reusable says whether m, about to be checked, would come out the same as
// pm did: the same text, checked cleanly before, and every import resolved
// to the same module, itself reused.
func reusable(m, pm *Module) bool {
	if pm.Source != m.Source || !pm.checked || pm.Broken || len(pm.imports) != len(m.imports) {
		return false
	}
	for i, ri := range m.imports {
		before := pm.imports[i]
		if ri.err != nil || before.err != nil || ri.target == nil || before.target == nil ||
			ri.target.Path != before.target.Path || !ri.target.reused {
			return false
		}
	}
	return true
}

// checkModule checks one module with the declarations, recording its
// exports and index. A checker panic is turned into an internal error on
// the module rather than taking the process down.
func (p *Project) checkModule(m *Module, checker *types.Checker, declStatements []ast.Statement) (diags diag.List) {
	defer func() {
		if r := recover(); r != nil {
			diags = append(diags, diag.Diagnostic{File: m.Path, Line: 1, Column: 1, EndLine: 1, EndColumn: 1,
				Severity: diag.Error, Code: "internal", Message: fmt.Sprintf("internal compiler error checking this file: %v", r)})
			m.Broken = true
		}
	}()

	statements := make([]ast.Statement, 0, len(declStatements)+len(m.Statements))
	statements = append(statements, declStatements...)
	statements = append(statements, m.Statements...)

	diags = append(diags, moduleDiags(checker.Check(statements), m.Path)...)
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

// moduleDiags keeps the errors in the module's own file, and places one
// with no file at all (raised on a token the parser did not produce) at the
// top of the module rather than losing it. Errors in declaration files are
// reported once, by the declarations' own check.
func moduleDiags(errs []*types.TypeError, path string) diag.List {
	var out diag.List
	for _, e := range errs {
		switch {
		case e.File == path:
			out = append(out, typeDiag(e))
		case e.File == "":
			d := typeDiag(e)
			d.File = path
			if d.Line < 1 {
				d.Line, d.Column, d.EndLine, d.EndColumn = 1, 1, 1, 1
			}
			out = append(out, d)
		}
	}
	return out
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
