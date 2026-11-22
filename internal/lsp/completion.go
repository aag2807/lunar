package lsp

import (
	"sort"
	"strings"

	"lunar/internal/lexer"
	"lunar/internal/project"
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
			// a?.b reads as a.b
			if text[start-1] == '.' && end > 0 && text[end-1] == '?' {
				end--
			}
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
	return lookupIn(m.Index, path, name, offset)
}

// lookupIn is lookupName in a given index.
func lookupIn(ix *types.Index, path, name string, offset int) *types.Symbol {
	if ix == nil {
		return nil
	}
	var best *types.Occurrence
	for i := range ix.Occurrences {
		o := &ix.Occurrences[i]
		if o.Tok.File == path && o.Tok.Literal == name && o.Tok.Offset < offset && o.Sym.Container == "" {
			if best == nil || o.Tok.Offset > best.Tok.Offset {
				best = o
			}
		}
	}
	if best != nil {
		return best.Sym
	}
	for _, sym := range ix.Symbols {
		if sym.Name == name && sym.Container == "" && sym.ScopeEnd == -1 {
			return sym
		}
	}
	return nil
}

// typeOfChain finds the type of a dotted expression such as self.pos.
func (s *Server) typeOfChain(path string, chain []string, offset int) types.Type {
	if m := s.modules[path]; m != nil {
		t, _ := typeOfChainIn(m.Index, path, chain, offset)
		return t
	}
	return nil
}

// typeOfChainIn is typeOfChain in a given index; it also returns the
// symbol the chain starts with.
func typeOfChainIn(ix *types.Index, path string, chain []string, offset int) (types.Type, *types.Symbol) {
	if len(chain) == 0 {
		return nil, nil
	}
	sym := lookupIn(ix, path, chain[0], offset)
	if sym == nil {
		return nil, nil
	}
	t := sym.Type
	for _, member := range chain[1:] {
		if t = memberType(t, member); t == nil {
			return nil, sym
		}
	}
	return t, sym
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
	case *types.ArrayType:
		// `local M = {}` and the fields given to it since.
		for name, sym := range typ.MemberSyms {
			kind := FieldCompletion
			if _, isFn := sym.Type.(*types.FunctionType); isFn {
				kind = FunctionCompletion
			}
			if !methodsOnly || kind == FunctionCompletion {
				seen[name] = true
				out = append(out, member{name, sym.Type, kind})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// staticMembers lists what follows `Class.`: new, and the statics.
func staticMembers(cl *types.ClassType) []member {
	out := []member{{"new", cl.Constructor, FunctionCompletion}}
	seen := map[string]bool{"new": true}
	for c := cl; c != nil; c = c.Parent {
		for name, m := range c.StaticMethods {
			if !seen[name] {
				seen[name] = true
				out = append(out, member{name, m, FunctionCompletion})
			}
		}
		for name, p := range c.StaticProperties {
			if !seen[name] {
				seen[name] = true
				out = append(out, member{name, p, FieldCompletion})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// inCommentOrString says whether offset is inside a comment or a string,
// where completion offers nothing.
func inCommentOrString(text string, offset int) bool {
	if offset > len(text) {
		offset = len(text)
	}
	l := lexer.New(text)
	prevEnd := 0
	for i := 0; i < 1_000_000; i++ {
		tok := l.NextToken()
		if tok.Type == lexer.EOF || tok.Offset >= offset {
			// Only whitespace and comments lie between the last token and
			// offset: offset is in a comment if one is open there.
			gap := text[prevEnd:offset]
			at := strings.LastIndex(gap, "--")
			if at < 0 {
				return false
			}
			rest := gap[at:]
			if strings.HasPrefix(rest, "--[") {
				return !strings.Contains(rest, "]")
			}
			return !strings.Contains(rest, "\n")
		}
		unterminated := tok.Type == lexer.ILLEGAL && strings.HasPrefix(tok.Literal, "unterminated string")
		if (tok.Type == lexer.STRING || unterminated) && tok.Offset < offset && (offset < tok.End || unterminated && offset <= tok.End) {
			return true
		}
		prevEnd = tok.End
	}
	return false
}

// typePosition says whether a name starting at start is where a type is
// written: after `name: ` or `): `, after `<`, extends, implements or as.
func typePosition(text string, start int) bool {
	i := start
	for i > 0 && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	if i == 0 {
		return false
	}
	switch text[i-1] {
	case ':':
		return i < start // `x: T`; `obj:m` is a method call
	case '<', '|', '&':
		return true
	}
	word, _ := identBefore(text, i)
	return word == "extends" || word == "implements" || word == "as"
}

// builtinTypes are the type names that are not symbols.
var builtinTypes = []string{"any", "boolean", "never", "nil", "number", "string", "table", "unknown", "void"}

// completionIndex is the index completion reads at offset in path: the
// last analysis, or, while the file does not parse as typed (the usual
// case: "h2." is not a statement), the same text with the line being typed
// blanked, and an `end` added for each block still open, checked again.
func (s *Server) completionIndex(path string, offset int) *types.Index {
	m := s.modules[path]
	if m != nil && !s.broken[path] {
		return m.Index
	}
	var prog *project.Program
	for _, p := range s.progs {
		if p != nil && p.Has(path) {
			prog = p
			break
		}
	}
	if prog != nil {
		text := s.file(path).Text
		if offset > len(text) {
			offset = len(text)
		}
		lineStart := strings.LastIndexByte(text[:offset], '\n') + 1
		lineEnd := len(text)
		if nl := strings.IndexByte(text[offset:], '\n'); nl >= 0 {
			lineEnd = offset + nl
		}
		repaired := text[:lineStart] + strings.Repeat(" ", lineEnd-lineStart) + text[lineEnd:]
		for i := 0; i < 6; i++ {
			if fixed, ok := prog.WithText(path, repaired); ok {
				return fixed.Index
			}
			repaired += "\nend"
		}
	}
	if m != nil {
		return m.Index
	}
	return nil
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
	if inCommentOrString(text, offset) {
		return CompletionList{Items: items}, nil
	}
	ix := s.completionIndex(path, offset)

	// After '.' or ':': the members of what is on the left.
	if start > 0 && (text[start-1] == '.' || text[start-1] == ':') {
		end := start - 1
		if text[end] == '.' && end > 0 && text[end-1] == '?' {
			end-- // e?.
		}
		chain := chainBefore(text, end)
		t, sym := typeOfChainIn(ix, path, chain, end)
		var list []member
		switch typ := t.(type) {
		case *types.ClassType:
			if sym != nil && sym.Kind == types.SymClass && len(chain) == 1 {
				list = staticMembers(typ) // Enemy.new, Enemy.count
			} else {
				list = members(t, text[start-1] == ':')
			}
		case *types.StringType, *types.StringLiteralType:
			// A string's methods are the string library's functions.
			if lib := lookupIn(ix, path, "string", end); lib != nil {
				list = members(lib.Type, false)
			}
		default:
			list = members(t, text[start-1] == ':')
		}
		for _, m := range list {
			if strings.HasPrefix(strings.ToLower(m.name), lowered) {
				items = append(items, CompletionItem{Label: m.name, Kind: m.kind, Detail: typeString(m.typ)})
			}
		}
		return CompletionList{Items: items}, nil
	}

	// Otherwise every name in scope here, innermost first, then what the
	// file uses from elsewhere (imports, globals), and keywords. Where a
	// type goes, only types.
	wantType := typePosition(text, start)
	isType := func(k types.SymbolKind) bool {
		return k == types.SymInterface || k == types.SymTypeAlias || k == types.SymClass || k == types.SymEnum
	}
	seen := map[string]bool{}
	offer := func(sym *types.Symbol) {
		if seen[sym.Name] || !strings.HasPrefix(strings.ToLower(sym.Name), lowered) {
			return
		}
		if wantType && !isType(sym.Kind) || !wantType && (sym.Kind == types.SymInterface || sym.Kind == types.SymTypeAlias) {
			return
		}
		seen[sym.Name] = true
		items = append(items, CompletionItem{Label: sym.Name, Kind: completionKind(sym.Kind), Detail: describe(sym)})
	}
	if ix != nil {
		syms := append([]*types.Symbol{}, ix.Symbols...)
		sort.SliceStable(syms, func(i, j int) bool { return syms[i].ScopeEnd != -1 && syms[j].ScopeEnd == -1 })
		for _, sym := range syms {
			if sym.Container != "" {
				continue
			}
			if sym.ScopeEnd != -1 && !(sym.Decl.File == path && sym.Decl.Offset < offset && offset <= sym.ScopeEnd) {
				continue // a local of another function, or not declared yet
			}
			offer(sym)
		}
		for _, o := range ix.Occurrences {
			if o.Tok.File == path && o.Sym.Container == "" && o.Sym.Decl.File != path && o.Sym.ScopeEnd == -1 {
				offer(o.Sym)
			}
		}
	}
	if wantType {
		for _, name := range builtinTypes {
			if !seen[name] && strings.HasPrefix(name, lowered) {
				items = append(items, CompletionItem{Label: name, Kind: InterfaceKind})
			}
		}
		return CompletionList{Items: items}, nil
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
		fn = typ.EffectiveConstructor()
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
