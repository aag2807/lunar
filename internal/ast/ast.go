package ast

import (
	"bytes"
	"fmt"
	"lunar/internal/lexer"
	"strings"
)

type Node interface {
	TokenLiteral() string
	String() string
}

type Expression interface {
	Node
	expressionNode()
}

type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// SuperExpression represents the 'super' keyword for accessing parent class members
type SuperExpression struct {
	Token lexer.Token // The 'super' token
}

func (se *SuperExpression) expressionNode()      {}
func (se *SuperExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SuperExpression) String() string       { return "super" }

type NumberLiteral struct {
	Token lexer.Token
	Value float64
}

func (i *NumberLiteral) expressionNode()      {}
func (i *NumberLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *NumberLiteral) String() string       { return i.Token.Literal }

type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (i *StringLiteral) expressionNode()      {}
func (i *StringLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *StringLiteral) String() string {
	return fmt.Sprintf("\"%s\"", i.Value)
}

type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (i *BooleanLiteral) expressionNode()      {}
func (i *BooleanLiteral) TokenLiteral() string { return i.Token.Literal }
func (i *BooleanLiteral) String() string       { return i.Token.Literal }

type NilLiteral struct {
	Token lexer.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return "nil" }

type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) expressionNode()      {}
func (i *InfixExpression) TokenLiteral() string { return i.Token.Literal }
func (i *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)",
		i.Left.String(),
		i.Operator,
		i.Right.String(),
	)
}

type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String())
}

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

type DotExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	return fmt.Sprintf("%s.%s", de.Left.String(), de.Right.String())
}

type IndexExpression struct {
	Token lexer.Token // '[' token
	Left  Expression  // the object being indexed
	Index Expression  // the index expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	return fmt.Sprintf("%s[%s]", ie.Left.String(), ie.Index.String())
}

type TableLiteral struct {
	Token  lexer.Token // '{' token
	Pairs  map[Expression]Expression // for key-value pairs
	Values []Expression // for array-style values
}

func (tl *TableLiteral) expressionNode()      {}
func (tl *TableLiteral) TokenLiteral() string { return tl.Token.Literal }
func (tl *TableLiteral) String() string {
	var out strings.Builder
	out.WriteString("{")

	// Print array-style values first
	for i, val := range tl.Values {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(val.String())
	}

	// Print key-value pairs
	if len(tl.Pairs) > 0 && len(tl.Values) > 0 {
		out.WriteString(", ")
	}

	pairStrs := []string{}
	for key, val := range tl.Pairs {
		pairStrs = append(pairStrs, fmt.Sprintf("%s = %s", key.String(), val.String()))
	}
	out.WriteString(strings.Join(pairStrs, ", "))

	out.WriteString("}")
	return out.String()
}

type Statement interface {
	Node
	statementNode()
}

type VariableDeclaration struct {
	Token      lexer.Token
	Name       *Identifier
	Type       Expression
	Value      Expression
	IsConstant bool
}

func (vd *VariableDeclaration) statementNode()       {}
func (vd *VariableDeclaration) TokenLiteral() string { return vd.Token.Literal }
func (vd *VariableDeclaration) String() string {
	var out strings.Builder

	if vd.IsConstant {
		out.WriteString("const ")
	} else {
		out.WriteString("local ")
	}

	out.WriteString(vd.Name.String())

	// Type annotation
	if vd.Type != nil {
		out.WriteString(": ")
		out.WriteString(vd.Type.String())
	}

	// Value assignment
	if vd.Value != nil {
		out.WriteString(" = ")
		out.WriteString(vd.Value.String())
	}

	return out.String()
}

type OptionalType struct {
	Token lexer.Token
	Type  Expression
}

func (ot *OptionalType) expressionNode()      {}
func (ot *OptionalType) TokenLiteral() string { return ot.Token.Literal }
func (ot *OptionalType) String() string       { return ot.Type.String() + "?" }

type ArrayType struct {
	Token       lexer.Token // the element type token
	ElementType Expression
}

func (at *ArrayType) expressionNode()      {}
func (at *ArrayType) TokenLiteral() string { return at.Token.Literal }
func (at *ArrayType) String() string       { return at.ElementType.String() + "[]" }

type TableType struct {
	Token     lexer.Token // 'table' token
	KeyType   Expression
	ValueType Expression
}

func (tt *TableType) expressionNode()      {}
func (tt *TableType) TokenLiteral() string { return tt.Token.Literal }
func (tt *TableType) String() string {
	return fmt.Sprintf("table<%s, %s>", tt.KeyType.String(), tt.ValueType.String())
}

type UnionType struct {
	Token lexer.Token // '|' token
	Types []Expression
}

func (ut *UnionType) expressionNode()      {}
func (ut *UnionType) TokenLiteral() string { return ut.Token.Literal }
func (ut *UnionType) String() string {
	typeStrs := []string{}
	for _, t := range ut.Types {
		if t != nil {
			typeStrs = append(typeStrs, t.String())
		}
	}
	return strings.Join(typeStrs, " | ")
}

type TupleType struct {
	Token lexer.Token // '(' token
	Types []Expression
}

func (tt *TupleType) expressionNode()      {}
func (tt *TupleType) TokenLiteral() string { return tt.Token.Literal }
func (tt *TupleType) String() string {
	typeStrs := []string{}
	for _, t := range tt.Types {
		if t != nil {
			typeStrs = append(typeStrs, t.String())
		}
	}
	return fmt.Sprintf("(%s)", strings.Join(typeStrs, ", "))
}

type FunctionType struct {
	Token      lexer.Token // '(' or first param token
	Parameters []*Parameter
	ReturnType Expression
}

func (ft *FunctionType) expressionNode()      {}
func (ft *FunctionType) TokenLiteral() string { return ft.Token.Literal }
func (ft *FunctionType) String() string {
	paramStrs := []string{}
	for _, p := range ft.Parameters {
		paramStrs = append(paramStrs, p.String())
	}
	return fmt.Sprintf("(%s) => %s", strings.Join(paramStrs, ", "), ft.ReturnType.String())
}

type GenericType struct {
	Token         lexer.Token // the base type token
	BaseType      Expression
	TypeArguments []Expression
}

func (gt *GenericType) expressionNode()      {}
func (gt *GenericType) TokenLiteral() string { return gt.Token.Literal }
func (gt *GenericType) String() string {
	argStrs := []string{}
	for _, arg := range gt.TypeArguments {
		argStrs = append(argStrs, arg.String())
	}
	return fmt.Sprintf("%s<%s>", gt.BaseType.String(), strings.Join(argStrs, ", "))
}

type Parameter struct {
	Token lexer.Token
	Name  *Identifier
	Type  Expression
}

func (p *Parameter) expressionNode()      {}
func (p *Parameter) TokenLiteral() string { return p.Token.Literal }
func (p *Parameter) String() string {
	var out strings.Builder
	out.WriteString(p.Name.String())
	if p.Type != nil {
		out.WriteString(": ")
		out.WriteString(p.Type.String())
	}
	return out.String()
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out strings.Builder
	for _, s := range bs.Statements {
		out.WriteString("    ") // Add indentation
		out.WriteString(s.String())
		if s != bs.Statements[len(bs.Statements)-1] {
			out.WriteString("\n")
		}
	}
	return out.String()
}

type FunctionDeclaration struct {
	Token         lexer.Token
	Name          *Identifier
	GenericParams []*Identifier // generic type parameters like <T, U>
	Parameters    []*Parameter
	ReturnType    Expression
	Body          *BlockStatement
	IsStatic      bool   // static method (when used in class)
	IsAbstract    bool   // abstract method (when used in class)
	Visibility    string // visibility modifier: public, private, protected (when used in class)
}

func (fd *FunctionDeclaration) statementNode()       {}
func (fd *FunctionDeclaration) TokenLiteral() string { return fd.Token.Literal }
func (fd *FunctionDeclaration) String() string {
	var out strings.Builder

	params := []string{}
	for _, p := range fd.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("function ")
	out.WriteString(fd.Name.String())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")

	if fd.ReturnType != nil {
		out.WriteString(": ")
		out.WriteString(fd.ReturnType.String())
	}

	out.WriteString("\n")
	out.WriteString(fd.Body.String())
	out.WriteString("\nend")

	return out.String()
}

type ReturnStatement struct {
	Token       lexer.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString("return ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	return out.String()
}

type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type IfStatement struct {
	Token       lexer.Token // 'if' token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement // can be nil
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var out strings.Builder

	out.WriteString("if ")
	out.WriteString(is.Condition.String())
	out.WriteString(" then\n")
	out.WriteString(is.Consequence.String())

	if is.Alternative != nil {
		out.WriteString("\nelse\n")
		out.WriteString(is.Alternative.String())
	}

	out.WriteString("\nend")
	return out.String()
}

type WhileStatement struct {
	Token     lexer.Token // 'while' token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	var out strings.Builder

	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" do\n")
	out.WriteString(ws.Body.String())
	out.WriteString("\nend")

	return out.String()
}

type ForStatement struct {
	Token    lexer.Token // 'for' token
	Variable *Identifier
	Start    Expression // for numeric: start value
	End      Expression // for numeric: end value
	Step     Expression // for numeric: step value (optional)
	Iterator Expression // for generic: iterator expression
	Body     *BlockStatement
	IsGeneric bool // true if generic for, false if numeric for
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out strings.Builder

	out.WriteString("for ")
	out.WriteString(fs.Variable.String())

	if fs.IsGeneric {
		out.WriteString(" in ")
		out.WriteString(fs.Iterator.String())
	} else {
		out.WriteString(" = ")
		out.WriteString(fs.Start.String())
		out.WriteString(", ")
		out.WriteString(fs.End.String())
		if fs.Step != nil {
			out.WriteString(", ")
			out.WriteString(fs.Step.String())
		}
	}

	out.WriteString(" do\n")
	out.WriteString(fs.Body.String())
	out.WriteString("\nend")

	return out.String()
}

type DoStatement struct {
	Token lexer.Token // 'do' token
	Body  *BlockStatement
}

func (ds *DoStatement) statementNode()       {}
func (ds *DoStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DoStatement) String() string {
	var out strings.Builder

	out.WriteString("do\n")
	out.WriteString(ds.Body.String())
	out.WriteString("\nend")

	return out.String()
}

type BreakStatement struct {
	Token lexer.Token // 'break' token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

type AssignmentStatement struct {
	Token lexer.Token // '=' token
	Name  Expression  // left side (can be identifier, dot expression, index expression)
	Value Expression  // right side
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignmentStatement) String() string {
	var out strings.Builder
	out.WriteString(as.Name.String())
	out.WriteString(" = ")
	out.WriteString(as.Value.String())
	return out.String()
}

type ClassDeclaration struct {
	Token         lexer.Token // 'class' token
	Name          *Identifier
	GenericParams []*Identifier           // generic type parameters like <T, U>
	Extends       Expression              // parent class name (single inheritance)
	Properties    []*PropertyDeclaration
	Methods       []*FunctionDeclaration
	Constructor   *ConstructorDeclaration
	Implements    []Expression // interface names
	IsAbstract    bool         // abstract class
}

func (cd *ClassDeclaration) statementNode()       {}
func (cd *ClassDeclaration) TokenLiteral() string { return cd.Token.Literal }
func (cd *ClassDeclaration) String() string {
	var out strings.Builder

	out.WriteString("class ")
	out.WriteString(cd.Name.String())

	if len(cd.Implements) > 0 {
		out.WriteString(" implements ")
		impls := []string{}
		for _, impl := range cd.Implements {
			impls = append(impls, impl.String())
		}
		out.WriteString(strings.Join(impls, ", "))
	}

	out.WriteString("\n")

	// Properties
	for _, prop := range cd.Properties {
		out.WriteString("    ")
		out.WriteString(prop.String())
		out.WriteString("\n")
	}

	// Constructor
	if cd.Constructor != nil {
		out.WriteString("\n")
		out.WriteString("    ")
		out.WriteString(cd.Constructor.String())
		out.WriteString("\n")
	}

	// Methods
	for _, method := range cd.Methods {
		out.WriteString("\n")
		// Indent method
		methodStr := method.String()
		lines := strings.Split(methodStr, "\n")
		for _, line := range lines {
			out.WriteString("    ")
			out.WriteString(line)
			out.WriteString("\n")
		}
	}

	out.WriteString("end")
	return out.String()
}

type PropertyDeclaration struct {
	Token      lexer.Token // property name token
	Visibility string      // "public", "private", "protected"
	IsStatic   bool        // static property
	IsReadonly bool        // readonly property
	Name       *Identifier
	Type       Expression
}

func (pd *PropertyDeclaration) statementNode()       {}
func (pd *PropertyDeclaration) TokenLiteral() string { return pd.Token.Literal }
func (pd *PropertyDeclaration) String() string {
	var out strings.Builder
	if pd.Visibility != "" {
		out.WriteString(pd.Visibility)
		out.WriteString(" ")
	}
	out.WriteString(pd.Name.String())
	out.WriteString(": ")
	out.WriteString(pd.Type.String())
	return out.String()
}

type ConstructorDeclaration struct {
	Token      lexer.Token // 'constructor' token
	Parameters []*Parameter
	Body       *BlockStatement
}

func (cd *ConstructorDeclaration) statementNode()       {}
func (cd *ConstructorDeclaration) TokenLiteral() string { return cd.Token.Literal }
func (cd *ConstructorDeclaration) String() string {
	var out strings.Builder

	params := []string{}
	for _, p := range cd.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("constructor(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")\n")
	out.WriteString(cd.Body.String())
	out.WriteString("\nend")

	return out.String()
}

type InterfaceDeclaration struct {
	Token      lexer.Token // 'interface' token
	Name       *Identifier
	Methods    []*InterfaceMethod
	Properties []*PropertyDeclaration
	Extends    []Expression // parent interface names
}

func (id *InterfaceDeclaration) statementNode()       {}
func (id *InterfaceDeclaration) TokenLiteral() string { return id.Token.Literal }
func (id *InterfaceDeclaration) String() string {
	var out strings.Builder

	out.WriteString("interface ")
	out.WriteString(id.Name.String())

	if len(id.Extends) > 0 {
		out.WriteString(" extends ")
		exts := []string{}
		for _, ext := range id.Extends {
			exts = append(exts, ext.String())
		}
		out.WriteString(strings.Join(exts, ", "))
	}

	out.WriteString("\n")

	// Properties
	for _, prop := range id.Properties {
		out.WriteString("    ")
		out.WriteString(prop.String())
		out.WriteString("\n")
	}

	// Methods
	for _, method := range id.Methods {
		out.WriteString("    ")
		out.WriteString(method.String())
		out.WriteString("\n")
	}

	out.WriteString("end")
	return out.String()
}

type InterfaceMethod struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType Expression
}

func (im *InterfaceMethod) statementNode()       {}
func (im *InterfaceMethod) TokenLiteral() string { return im.Token.Literal }
func (im *InterfaceMethod) String() string {
	params := []string{}
	for _, p := range im.Parameters {
		params = append(params, p.String())
	}

	var out strings.Builder
	out.WriteString(im.Name.String())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")

	if im.ReturnType != nil {
		out.WriteString(": ")
		out.WriteString(im.ReturnType.String())
	}

	return out.String()
}

type EnumDeclaration struct {
	Token   lexer.Token // 'enum' token
	Name    *Identifier
	Members []*EnumMember
}

func (ed *EnumDeclaration) statementNode()       {}
func (ed *EnumDeclaration) TokenLiteral() string { return ed.Token.Literal }
func (ed *EnumDeclaration) String() string {
	var out strings.Builder

	out.WriteString("enum ")
	out.WriteString(ed.Name.String())
	out.WriteString("\n")

	for _, member := range ed.Members {
		out.WriteString("    ")
		out.WriteString(member.String())
		out.WriteString("\n")
	}

	out.WriteString("end")
	return out.String()
}

type EnumMember struct {
	Token lexer.Token
	Name  *Identifier
	Value Expression // optional - can be nil
}

func (em *EnumMember) statementNode()       {}
func (em *EnumMember) TokenLiteral() string { return em.Token.Literal }
func (em *EnumMember) String() string {
	if em.Value != nil {
		return fmt.Sprintf("%s = %s", em.Name.String(), em.Value.String())
	}
	return em.Name.String()
}

type TypeDeclaration struct {
	Token         lexer.Token // 'type' token
	Name          *Identifier
	GenericParams []*Identifier            // generic type parameters (e.g., T, U)
	Type          Expression               // the type being aliased (for type Name = Type)
	Properties    []*PropertyDeclaration // for object shape (type Name ... end)
}

func (td *TypeDeclaration) statementNode()       {}
func (td *TypeDeclaration) TokenLiteral() string { return td.Token.Literal }
func (td *TypeDeclaration) String() string {
	if td.Type != nil {
		return fmt.Sprintf("type %s = %s", td.Name.String(), td.Type.String())
	}
	// Object shape type
	return fmt.Sprintf("type %s { ... }", td.Name.String())
}

// ObjectShapeType represents an inline object shape for type declarations
type ObjectShapeType struct {
	Token      lexer.Token
	Properties []*PropertyDeclaration
}

func (ost *ObjectShapeType) expressionNode()      {}
func (ost *ObjectShapeType) TokenLiteral() string { return ost.Token.Literal }
func (ost *ObjectShapeType) String() string {
	return "{ object shape }"
}

// ExportStatement wraps another statement to mark it as exported
type ExportStatement struct {
	Token     lexer.Token // 'export' token
	Statement Statement   // the statement being exported
}

func (es *ExportStatement) statementNode()       {}
func (es *ExportStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExportStatement) String() string {
	return fmt.Sprintf("export %s", es.Statement.String())
}

// ImportStatement represents an import declaration
type ImportStatement struct {
	Token   lexer.Token   // 'import' token
	Names   []*Identifier // names being imported
	Module  string        // module path (string literal)
	IsWildcard bool       // true if using * import
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	if is.IsWildcard {
		return fmt.Sprintf("import * from \"%s\"", is.Module)
	}
	names := []string{}
	for _, name := range is.Names {
		names = append(names, name.String())
	}
	return fmt.Sprintf("import { %s } from \"%s\"", strings.Join(names, ", "), is.Module)
}

// DeclareStatement represents an ambient declaration (no implementation)
// Used in .d.lunar files to declare external APIs
type DeclareStatement struct {
	Token       lexer.Token // 'declare' token
	Declaration Statement   // the declaration (variable, function, class, etc.)
}

func (ds *DeclareStatement) statementNode()       {}
func (ds *DeclareStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DeclareStatement) String() string {
	if ds.Declaration != nil {
		return fmt.Sprintf("declare %s", ds.Declaration.String())
	}
	return "declare"
}
