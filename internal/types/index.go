package types

import (
	"strings"

	"lunar/internal/lexer"
)

// SymbolKind says what a name was defined as.
type SymbolKind int

const (
	SymVariable SymbolKind = iota
	SymFunction
	SymParameter
	SymClass
	SymInterface
	SymEnum
	SymTypeAlias
	SymNamespace
	SymProperty
	SymMethod
	SymEnumMember
)

func (k SymbolKind) String() string {
	return [...]string{"variable", "function", "parameter", "class", "interface", "enum",
		"type", "namespace", "property", "method", "enum member"}[k]
}

// Symbol is one definition: a variable, function, parameter, type or
// member. Its identity is where it is declared (Decl's file and offset), so
// the same declaration checked by two module checkers is one symbol.
type Symbol struct {
	Name string
	Kind SymbolKind
	Type Type
	// Decl is the token that names the symbol where it is defined.
	Decl lexer.Token
	// Container names the class, interface, enum or namespace a member
	// belongs to.
	Container string
	// ScopeEnd is the byte offset where a local's scope ends; -1 for
	// symbols visible to the whole file and beyond.
	ScopeEnd int
}

// Key identifies a symbol across checkers: its declaring file and offset,
// and its name, since `self` is placed at its class's name.
type Key struct {
	File   string
	Offset int
	Name   string
}

// Key returns the symbol's identity.
func (s *Symbol) Key() Key {
	return Key{File: s.Decl.File, Offset: s.Decl.Offset, Name: s.Name}
}

// Ambient reports whether the symbol comes from the standard library or a
// declaration file rather than from Lunar code.
func (s *Symbol) Ambient() bool {
	return strings.HasPrefix(s.Decl.File, "<stdlib>") || strings.HasSuffix(s.Decl.File, ".d.lunar")
}

// Occurrence is one place a symbol's name appears: where it is declared,
// or a use.
type Occurrence struct {
	Tok  lexer.Token
	Sym  *Symbol
	Decl bool
}

// Index is every symbol a check defined and every occurrence it resolved.
type Index struct {
	Symbols     []*Symbol
	Occurrences []Occurrence
}

// At returns the occurrence whose token covers offset in file.
func (ix *Index) At(file string, offset int) (Occurrence, bool) {
	for _, o := range ix.Occurrences {
		if o.Tok.File == file && o.Tok.Offset <= offset && offset <= o.Tok.End {
			return o, true
		}
	}
	return Occurrence{}, false
}

// Index returns what the last Check defined and resolved.
func (c *Checker) Index() *Index {
	return &c.index
}

// define records a definition named by tok, makes it what later uses of
// the name in the current scope resolve to, and returns it.
func (c *Checker) define(tok lexer.Token, kind SymbolKind, typ Type) *Symbol {
	sym := &Symbol{Name: tok.Literal, Kind: kind, Type: typ, Decl: tok, ScopeEnd: c.scopeEnd}
	c.index.Symbols = append(c.index.Symbols, sym)
	c.index.Occurrences = append(c.index.Occurrences, Occurrence{Tok: tok, Sym: sym, Decl: true})
	c.env.defineSym(tok.Literal, sym)
	return sym
}

// defineMember records a member of a class, interface, enum or namespace;
// it is reached through its container, not by name in scope.
func (c *Checker) defineMember(tok lexer.Token, kind SymbolKind, typ Type, container string) *Symbol {
	sym := &Symbol{Name: tok.Literal, Kind: kind, Type: typ, Decl: tok, Container: container, ScopeEnd: -1}
	c.index.Symbols = append(c.index.Symbols, sym)
	c.index.Occurrences = append(c.index.Occurrences, Occurrence{Tok: tok, Sym: sym, Decl: true})
	return sym
}

// reference records a use of sym at tok.
func (c *Checker) reference(tok lexer.Token, sym *Symbol) {
	if sym == nil || tok.Line == 0 {
		return
	}
	c.index.Occurrences = append(c.index.Occurrences, Occurrence{Tok: tok, Sym: sym})
}

func (e *Environment) defineSym(name string, sym *Symbol) {
	if e.syms == nil {
		e.syms = make(map[string]*Symbol)
	}
	e.syms[name] = sym
}

// lookupSym finds the symbol a name refers to in this scope or an outer one.
func (e *Environment) lookupSym(name string) *Symbol {
	for env := e; env != nil; env = env.outer {
		if _, defined := env.store[name]; defined {
			// The innermost binding wins, even a narrowing re-bind that
			// has no symbol of its own: keep looking outward for the
			// definition it refines.
			if sym, ok := env.syms[name]; ok {
				return sym
			}
		}
	}
	return nil
}

func setMemberSym(m *map[string]*Symbol, name string, sym *Symbol) {
	if *m == nil {
		*m = make(map[string]*Symbol)
	}
	(*m)[name] = sym
}

// memberSymbol finds where a member reached through a value of type t is
// defined: on a class or its parents, an interface or what it extends, an
// enum, or a namespace.
func memberSymbol(t Type, name string) *Symbol {
	switch typ := t.(type) {
	case *ClassType:
		for cl := typ; cl != nil; cl = cl.Parent {
			if sym := cl.MemberSyms[name]; sym != nil {
				return sym
			}
		}
	case *InterfaceType:
		if sym := typ.MemberSyms[name]; sym != nil {
			return sym
		}
		for _, parent := range typ.Extends {
			if sym := memberSymbol(parent, name); sym != nil {
				return sym
			}
		}
	case *EnumType:
		return typ.MemberSyms[name]
	case *NamespaceType:
		return typ.MemberSyms[name]
	case *OptionalType:
		return memberSymbol(typ.BaseType, name)
	case *UnionType:
		if nonNil, ok := removeNil(typ); ok {
			return memberSymbol(nonNil, name)
		}
	}
	return nil
}

// selfSymbol is what `self` refers to inside a class: the class, named at
// its declaration. It is not a definition of its own, so it is not listed
// among the index's symbols, but uses of `self` resolve to it.
func selfSymbol(classTok lexer.Token, classType *ClassType) *Symbol {
	return &Symbol{Name: "self", Kind: SymVariable, Type: classType, Decl: classTok, ScopeEnd: -1}
}
