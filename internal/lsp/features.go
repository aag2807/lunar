package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"lunar/internal/lexer"
	"lunar/internal/types"
)

// occurrenceAt finds the indexed name under the cursor.
func (s *Server) occurrenceAt(uri string, pos Position) (types.Occurrence, bool) {
	path := uriToPath(uri)
	offset := s.offsetAt(path, pos)
	if m := s.modules[path]; m != nil {
		if occ, ok := m.Index.At(path, offset); ok {
			return occ, true
		}
	}
	if s.prog != nil && s.prog.DeclIndex != nil {
		if occ, ok := s.prog.DeclIndex.At(path, offset); ok {
			return occ, true
		}
	}
	return types.Occurrence{}, false
}

// indexes are every index the last analysis produced.
func (s *Server) indexes() []*types.Index {
	var out []*types.Index
	paths := make([]string, 0, len(s.modules))
	for path := range s.modules {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		out = append(out, s.modules[path].Index)
	}
	if s.prog != nil && s.prog.DeclIndex != nil {
		out = append(out, s.prog.DeclIndex)
	}
	return out
}

func (s *Server) tokenRange(tok lexer.Token) Range {
	return s.rangeOf(tok.File, tok.Line, tok.Column, tok.EndLine, tok.EndColumn)
}

func (s *Server) location(tok lexer.Token) Location {
	return Location{URI: pathToURI(tok.File), Range: s.tokenRange(tok)}
}

func (s *Server) definition(uri string, pos Position) (interface{}, error) {
	occ, ok := s.occurrenceAt(uri, pos)
	if !ok {
		return nil, nil
	}
	return s.location(occ.Sym.Decl), nil
}

// occurrencesOf lists every place a symbol appears, in every file,
// declaration first, without duplicates.
func (s *Server) occurrencesOf(sym *types.Symbol, includeDecl bool) []types.Occurrence {
	key := sym.Key()
	seen := map[types.Key]bool{}
	var out []types.Occurrence
	for _, ix := range s.indexes() {
		for _, o := range ix.Occurrences {
			if o.Sym.Key() != key || (o.Decl && !includeDecl) {
				continue
			}
			at := types.Key{File: o.Tok.File, Offset: o.Tok.Offset, Name: o.Tok.Literal}
			if seen[at] {
				continue
			}
			seen[at] = true
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Decl != out[j].Decl {
			return out[i].Decl
		}
		if out[i].Tok.File != out[j].Tok.File {
			return out[i].Tok.File < out[j].Tok.File
		}
		return out[i].Tok.Offset < out[j].Tok.Offset
	})
	return out
}

func (s *Server) references(uri string, pos Position, includeDecl bool) []Location {
	occ, ok := s.occurrenceAt(uri, pos)
	if !ok {
		return []Location{}
	}
	locs := []Location{}
	for _, o := range s.occurrencesOf(occ.Sym, includeDecl) {
		locs = append(locs, s.location(o.Tok))
	}
	return locs
}

func (s *Server) hover(uri string, pos Position) (interface{}, error) {
	occ, ok := s.occurrenceAt(uri, pos)
	if !ok {
		return nil, nil
	}
	r := s.tokenRange(occ.Tok)
	return Hover{
		Contents: MarkupContent{Kind: "markdown", Value: "```lunar\n" + describe(occ.Sym) + "\n```"},
		Range:    &r,
	}, nil
}

// describe renders a symbol the way it would be declared.
func describe(sym *types.Symbol) string {
	qualified := sym.Name
	if sym.Container != "" {
		qualified = sym.Container + "." + sym.Name
	}
	switch sym.Kind {
	case types.SymFunction:
		if fn, ok := sym.Type.(*types.FunctionType); ok {
			return "function " + signature(qualified, fn)
		}
	case types.SymMethod:
		if fn, ok := sym.Type.(*types.FunctionType); ok {
			return "(method) " + signature(qualified, fn)
		}
	case types.SymProperty:
		return fmt.Sprintf("(property) %s: %s", qualified, typeString(sym.Type))
	case types.SymParameter:
		return fmt.Sprintf("(parameter) %s: %s", sym.Name, typeString(sym.Type))
	case types.SymClass, types.SymInterface, types.SymEnum, types.SymNamespace:
		return sym.Kind.String() + " " + sym.Name
	case types.SymTypeAlias:
		return fmt.Sprintf("type %s = %s", sym.Name, typeString(sym.Type))
	case types.SymEnumMember:
		return "(enum member) " + qualified
	}
	return fmt.Sprintf("%s: %s", qualified, typeString(sym.Type))
}

func typeString(t types.Type) string {
	if t == nil {
		return "any"
	}
	return t.String()
}

// signature renders "name(a: number, b?: string): string".
func signature(name string, fn *types.FunctionType) string {
	return name + "(" + strings.Join(paramLabels(fn), ", ") + "): " + typeString(fn.ReturnType)
}

func paramLabels(fn *types.FunctionType) []string {
	labels := make([]string, len(fn.Parameters))
	for i, p := range fn.Parameters {
		name := fmt.Sprintf("arg%d", i+1)
		if i < len(fn.ParamNames) && fn.ParamNames[i] != "" {
			name = fn.ParamNames[i]
		}
		optional := i < len(fn.OptionalParams) && fn.OptionalParams[i]
		rest := fn.HasRestParameter && i == len(fn.Parameters)-1
		switch {
		case rest:
			if arr, ok := p.(*types.ArrayType); ok {
				p = arr.ElementType
			}
			labels[i] = "..." + name + ": " + typeString(p)
		case optional:
			labels[i] = name + "?: " + typeString(p)
		default:
			labels[i] = name + ": " + typeString(p)
		}
	}
	return labels
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// renamable says whether a symbol can be renamed, and why not.
func renamable(sym *types.Symbol) error {
	switch {
	case sym.Name == "self":
		return fmt.Errorf("'self' cannot be renamed")
	case strings.HasPrefix(sym.Decl.File, "<stdlib>"):
		return fmt.Errorf("'%s' is part of Lua's standard library", sym.Name)
	case sym.Ambient():
		return fmt.Errorf("'%s' is declared in %s; rename it there", sym.Name, sym.Decl.File)
	}
	return nil
}

// renameBlocked refuses a rename while any file does not parse: its
// positions are from an older text, and editing at them would change the
// wrong characters.
func (s *Server) renameBlocked() error {
	if len(s.broken) == 0 {
		return nil
	}
	var names []string
	for path := range s.broken {
		names = append(names, s.relName(path))
	}
	sort.Strings(names)
	return fmt.Errorf("%s does not parse; fix it before renaming", strings.Join(names, ", "))
}

func (s *Server) prepareRename(uri string, pos Position) (interface{}, error) {
	if err := s.renameBlocked(); err != nil {
		return nil, err
	}
	occ, ok := s.occurrenceAt(uri, pos)
	if !ok {
		return nil, nil
	}
	if err := renamable(occ.Sym); err != nil {
		return nil, err
	}
	return PrepareRenameResult{Range: s.tokenRange(occ.Tok), Placeholder: occ.Tok.Literal}, nil
}

func (s *Server) rename(uri string, pos Position, newName string) (interface{}, error) {
	if err := s.renameBlocked(); err != nil {
		return nil, err
	}
	occ, ok := s.occurrenceAt(uri, pos)
	if !ok {
		return nil, fmt.Errorf("there is nothing to rename here")
	}
	if err := renamable(occ.Sym); err != nil {
		return nil, err
	}
	if !identifierPattern.MatchString(newName) || lexer.LookupIdent(newName) != lexer.IDENT {
		return nil, fmt.Errorf("'%s' is not a valid name", newName)
	}
	edit := WorkspaceEdit{Changes: map[string][]TextEdit{}}
	for _, o := range s.occurrencesOf(occ.Sym, true) {
		u := pathToURI(o.Tok.File)
		edit.Changes[u] = append(edit.Changes[u], TextEdit{Range: s.tokenRange(o.Tok), NewText: newName})
	}
	return edit, nil
}

func symbolKind(k types.SymbolKind) SymbolKind {
	switch k {
	case types.SymFunction:
		return FunctionSymbol
	case types.SymClass:
		return ClassSymbol
	case types.SymInterface:
		return InterfaceSymbol
	case types.SymEnum:
		return EnumSymbol
	case types.SymNamespace:
		return NamespaceSymbol
	case types.SymProperty:
		return PropertySymbol
	case types.SymMethod:
		return MethodSymbol
	case types.SymEnumMember:
		return ConstantSymbol
	case types.SymTypeAlias:
		return InterfaceSymbol
	}
	return VariableSymbol
}

// ownSymbols are the symbols a module defines in its own file.
func ownSymbols(path string, ix *types.Index) []*types.Symbol {
	var out []*types.Symbol
	for _, sym := range ix.Symbols {
		if sym.Decl.File == path && sym.Kind != types.SymParameter {
			out = append(out, sym)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Decl.Offset < out[j].Decl.Offset })
	return out
}

func (s *Server) documentSymbols(uri string) []DocumentSymbol {
	path := uriToPath(uri)
	var ix *types.Index
	if m := s.modules[path]; m != nil {
		ix = m.Index
	} else if s.prog != nil {
		ix = s.prog.DeclIndex
	}
	out := []DocumentSymbol{}
	if ix == nil {
		return out
	}
	containers := map[string]int{}
	for _, sym := range ownSymbols(path, ix) {
		ds := DocumentSymbol{Name: sym.Name, Detail: typeString(sym.Type), Kind: symbolKind(sym.Kind),
			Range: s.tokenRange(sym.Decl), SelectionRange: s.tokenRange(sym.Decl)}
		if sym.Container != "" {
			if i, ok := containers[sym.Container]; ok {
				out[i].Children = append(out[i].Children, ds)
			}
			continue
		}
		if sym.ScopeEnd != -1 {
			continue // a local
		}
		switch sym.Kind {
		case types.SymClass, types.SymInterface, types.SymEnum, types.SymNamespace:
			containers[sym.Name] = len(out)
		}
		out = append(out, ds)
	}
	return out
}

func (s *Server) workspaceSymbols(query string) []SymbolInformation {
	query = strings.ToLower(query)
	out := []SymbolInformation{}
	seen := map[types.Key]bool{}
	for path, m := range s.modules {
		for _, sym := range ownSymbols(path, m.Index) {
			if seen[sym.Key()] || (sym.Container == "" && sym.ScopeEnd != -1) {
				continue
			}
			if !strings.Contains(strings.ToLower(sym.Name), query) {
				continue
			}
			seen[sym.Key()] = true
			out = append(out, SymbolInformation{Name: sym.Name, Kind: symbolKind(sym.Kind),
				Location: s.location(sym.Decl), ContainerName: sym.Container})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// relName is a file's path relative to the workspace, for messages.
func (s *Server) relName(path string) string {
	if s.root != "" {
		if rel, err := filepath.Rel(s.root, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.Base(path)
}
