package lsp

import (
	"sort"
	"strings"

	"lunar/internal/types"
)

var keywords = []string{
	"and", "break", "class", "const", "constructor", "declare", "do", "else", "elseif", "end",
	"enum", "export", "extends", "false", "for", "function", "if", "implements", "import",
	"in", "interface", "local", "match", "namespace", "nil", "not", "or", "private",
	"protected", "public", "readonly", "repeat", "return", "self", "static", "then", "true",
	"type", "until", "while", "with",
}

func isIdentByte(b byte) bool {
	return b == '_' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}

// identBefore returns the identifier that ends at end in text, and where
// it starts.
func identBefore(text string, end int) (string, int) {
	start := end
	for start > 0 && isIdentByte(text[start-1]) {
		start--
	}
	return text[start:end], start
}

// chainBefore reads the dotted names that end at end: for "self.pos.x" it
// returns [self pos x]. It stops at anything else, such as a call.
func chainBefore(text string, end int) []string {
	var chain []string
	for {
		name, start := identBefore(text, end)
		if name == "" {
			return nil
		}
		chain = append([]string{name}, chain...)
		if start > 0 && (text[start-1] == '.' || text[start-1] == ':') {
			end = start - 1
			continue
		}
		return chain
	}
}

// lookupName resolves a name as of offset in path: the closest occurrence
// of it before offset in this file, else any top-level symbol by that name.
func (s *Server) lookupName(path, name string, offset int) *types.Symbol {
	m := s.modules[path]
	if m == nil {
		return nil
	}
	var best *types.Occurrence
	for i := range m.Index.Occurrences {
		o := &m.Index.Occurrences[i]
		if o.Tok.File == path && o.Tok.Literal == name && o.Tok.Offset < offset && o.Sym.Container == "" {
			if best == nil || o.Tok.Offset > best.Tok.Offset {
				best = o
			}
		}
	}
	if best != nil {
		return best.Sym
	}
	for _, sym := range m.Index.Symbols {
		if sym.Name == name && sym.Container == "" && sym.ScopeEnd == -1 {
			return sym
		}
	}
	return nil
}

// typeOfChain finds the type of a dotted expression such as self.pos.
func (s *Server) typeOfChain(path string, chain []string, offset int) types.Type {
	if len(chain) == 0 {
		return nil
	}
	sym := s.lookupName(path, chain[0], offset)
	if sym == nil {
		return nil
	}
	t := sym.Type
	for _, member := range chain[1:] {
		if t = memberType(t, member); t == nil {
			return nil
		}
	}
	return t
}

// unwrap strips nil from an optional, so `enemy?.` completes like `enemy.`.
func unwrap(t types.Type) types.Type {
	switch typ := t.(type) {
	case *types.OptionalType:
		return typ.BaseType
	case *types.UnionType:
		var rest []types.Type
		for _, m := range typ.Types {
			if m.String() != "nil" {
				rest = append(rest, m)
			}
		}
		if len(rest) == 1 {
			return rest[0]
		}
	}
	return t
}

// memberType is the type of a member reached through a value of type t.
func memberType(t types.Type, name string) types.Type {
	for _, item := range members(t, false) {
		if item.name == name {
			return item.typ
		}
	}
	return nil
}

type member struct {
	name string
	typ  types.Type
	kind CompletionItemKind
}

// members lists what can follow a '.' (or, with methodsOnly, a ':') on a
// value of type t.
func members(t types.Type, methodsOnly bool) []member {
	var out []member
	seen := map[string]bool{}
	add := func(name string, typ types.Type, kind CompletionItemKind) {
		if !seen[name] && (!methodsOnly || kind == MethodCompletion) {
			seen[name] = true
			out = append(out, member{name, typ, kind})
		}
	}
	switch typ := unwrap(t).(type) {
	case *types.ClassType:
		for cl := typ; cl != nil; cl = cl.Parent {
			for name, m := range cl.Methods {
				add(name, m, MethodCompletion)
			}
			for name, p := range cl.Properties {
				add(name, p, FieldCompletion)
			}
			for name, g := range cl.Getters {
				add(name, g.ReturnType, PropertyCompletion)
			}
			for name, m := range cl.StaticMethods {
				add(name, m, FunctionCompletion)
			}
			for name, p := range cl.StaticProperties {
				add(name, p, FieldCompletion)
			}
		}
	case *types.InterfaceType:
		var walk func(*types.InterfaceType)
		walk = func(it *types.InterfaceType) {
			for name, m := range it.Methods {
				add(name, m, MethodCompletion)
			}
			for name, p := range it.Properties {
				kind := FieldCompletion
				if _, isFn := p.(*types.FunctionType); isFn {
					kind = FunctionCompletion
				}
				add(name, p, kind)
			}
			for _, parent := range it.Extends {
				if parent != nil {
					walk(parent)
				}
			}
		}
		walk(typ)
	case *types.NamespaceType:
		for name, m := range typ.Members {
			kind := VariableCompletion
			if _, isFn := m.(*types.FunctionType); isFn {
				kind = FunctionCompletion
			}
			add(name, m, kind)
		}
	case *types.EnumType:
		for name, m := range typ.Members {
			add(name, m, EnumMemberKind)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func completionKind(k types.SymbolKind) CompletionItemKind {
	switch k {
	case types.SymFunction:
		return FunctionCompletion
	case types.SymClass:
		return ClassCompletion
	case types.SymInterface, types.SymTypeAlias:
		return InterfaceKind
	case types.SymEnum:
		return EnumCompletion
	case types.SymNamespace:
		return ModuleCompletion
	}
	return VariableCompletion
}

func (s *Server) completion(uri string, pos Position) (interface{}, error) {
	path := uriToPath(uri)
	text := s.file(path).Text
	offset := s.offsetAt(path, pos)
	prefix, start := identBefore(text, offset)
	lowered := strings.ToLower(prefix)
	items := []CompletionItem{}

	// After '.' or ':': the members of what is on the left.
	if start > 0 && (text[start-1] == '.' || text[start-1] == ':') {
		t := s.typeOfChain(path, chainBefore(text, start-1), start-1)
		for _, m := range members(t, text[start-1] == ':') {
			if strings.HasPrefix(strings.ToLower(m.name), lowered) {
				items = append(items, CompletionItem{Label: m.name, Kind: m.kind, Detail: typeString(m.typ)})
			}
		}
		return CompletionList{Items: items}, nil
	}

	// Otherwise every name in scope here, innermost first, and keywords.
	seen := map[string]bool{}
	if m := s.modules[path]; m != nil {
		syms := append([]*types.Symbol{}, m.Index.Symbols...)
		sort.SliceStable(syms, func(i, j int) bool { return syms[i].ScopeEnd != -1 && syms[j].ScopeEnd == -1 })
		for _, sym := range syms {
			if sym.Container != "" || seen[sym.Name] || !strings.HasPrefix(strings.ToLower(sym.Name), lowered) {
				continue
			}
			if sym.ScopeEnd != -1 && !(sym.Decl.File == path && sym.Decl.Offset < offset && offset <= sym.ScopeEnd) {
				continue // a local of another function, or not declared yet
			}
			seen[sym.Name] = true
			items = append(items, CompletionItem{Label: sym.Name, Kind: completionKind(sym.Kind), Detail: describe(sym)})
		}
	}
	for _, kw := range keywords {
		if !seen[kw] && strings.HasPrefix(kw, lowered) {
			items = append(items, CompletionItem{Label: kw, Kind: KeywordCompletion})
		}
	}
	return CompletionList{Items: items}, nil
}

// signatureHelp finds the call the cursor is in and shows its parameters,
// with the one being typed active.
func (s *Server) signatureHelp(uri string, pos Position) (interface{}, error) {
	path := uriToPath(uri)
	text := s.file(path).Text
	offset := s.offsetAt(path, pos)

	depth, active := 0, 0
	open := -1
scan:
	for i := offset - 1; i >= 0; i-- {
		switch text[i] {
		case ')', ']', '}':
			depth++
		case '(', '[', '{':
			if depth == 0 {
				if text[i] == '(' {
					open = i
				}
				break scan
			}
			depth--
		case ',':
			if depth == 0 {
				active++
			}
		case '\n':
			if i < offset-200 {
				break scan
			}
		}
	}
	if open < 0 {
		return nil, nil
	}

	chain := chainBefore(text, open)
	t := s.typeOfChain(path, chain, open)
	var fn *types.FunctionType
	switch typ := t.(type) {
	case *types.FunctionType:
		fn = typ
	case *types.ClassType:
		fn = typ.Constructor
		if fn == nil {
			fn = &types.FunctionType{ReturnType: typ}
		}
	}
	if fn == nil {
		return nil, nil
	}

	name := strings.Join(chain, ".")
	params := paramLabels(fn)
	info := SignatureInformation{Label: signature(name, fn)}
	for _, p := range params {
		info.Parameters = append(info.Parameters, ParameterInformation{Label: p})
	}
	if active >= len(params) && len(params) > 0 {
		active = len(params) - 1
	}
	return SignatureHelp{Signatures: []SignatureInformation{info}, ActiveParameter: active}, nil
}
