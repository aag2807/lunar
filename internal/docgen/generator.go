package docgen

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
)

// Generator generates documentation from Lunar source code
type Generator struct {
	output strings.Builder
}

// New creates a new documentation generator
func New() *Generator {
	return &Generator{}
}

// Generate generates documentation from source code
func (g *Generator) Generate(filename, source string) (string, error) {
	g.output.Reset()

	// Parse the source
	l := lexer.New(source)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors: %v", p.Errors())
	}

	// Write header
	g.writeHeader(filename)

	// Extract and write documentation for each statement
	for _, stmt := range statements {
		g.processStatement(stmt)
	}

	return g.output.String(), nil
}

// writeHeader writes the document header
func (g *Generator) writeHeader(filename string) {
	g.output.WriteString(fmt.Sprintf("# %s\n\n", filename))
	g.output.WriteString("## API Documentation\n\n")
}

// processStatement processes a statement and extracts documentation
func (g *Generator) processStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ClassDeclaration:
		g.writeClassDoc(s)
	case *ast.FunctionDeclaration:
		// Only document top-level functions (not class methods)
		if s.Visibility == "" {
			g.writeFunctionDoc(s)
		}
	case *ast.InterfaceDeclaration:
		g.writeInterfaceDoc(s)
	case *ast.EnumDeclaration:
		g.writeEnumDoc(s)
	case *ast.TypeDeclaration:
		g.writeTypeAliasDoc(s)
	case *ast.VariableDeclaration:
		// Only document exported variables (uppercase first letter)
		if len(s.Names) > 0 && s.Names[0] != nil && len(s.Names[0].Value) > 0 {
			first := s.Names[0].Value[0]
			if first >= 'A' && first <= 'Z' {
				g.writeVariableDoc(s)
			}
		}
	}
}

// writeClassDoc writes documentation for a class
func (g *Generator) writeClassDoc(stmt *ast.ClassDeclaration) {
	g.output.WriteString("### ")

	// Write class keyword with modifiers
	if stmt.IsAbstract {
		g.output.WriteString("abstract ")
	}
	g.output.WriteString("class ")

	// Write class name
	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	// Write generic parameters if any
	if len(stmt.GenericParams) > 0 {
		g.output.WriteString("<")
		for i, tp := range stmt.GenericParams {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(tp.Value)
		}
		g.output.WriteString(">")
	}

	g.output.WriteString("\n\n")

	// Write extends clause
	if stmt.Extends != nil {
		g.output.WriteString(fmt.Sprintf("**Extends:** `%s`\n\n", stmt.Extends.String()))
	}

	// Write implements clause
	if len(stmt.Implements) > 0 {
		g.output.WriteString("**Implements:** ")
		for i, iface := range stmt.Implements {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(fmt.Sprintf("`%s`", iface.String()))
		}
		g.output.WriteString("\n\n")
	}

	// Write properties
	if len(stmt.Properties) > 0 {
		g.output.WriteString("**Properties:**\n\n")
		for _, prop := range stmt.Properties {
			g.writePropertyDoc(prop)
		}
	}

	// Write methods
	if len(stmt.Methods) > 0 {
		g.output.WriteString("\n**Methods:**\n\n")
		for _, method := range stmt.Methods {
			g.writeMethodDoc(method)
		}
	}

	// Write constructor
	if stmt.Constructor != nil {
		g.output.WriteString("\n**Constructor:**\n\n")
		g.writeConstructorDoc(stmt.Constructor)
	}

	g.output.WriteString("\n---\n\n")
}

// writePropertyDoc writes documentation for a class property
func (g *Generator) writePropertyDoc(prop *ast.PropertyDeclaration) {
	g.output.WriteString("- ")

	// Write modifiers
	if prop.Visibility != "" {
		g.output.WriteString(fmt.Sprintf("`%s` ", prop.Visibility))
	}
	if prop.IsStatic {
		g.output.WriteString("`static` ")
	}
	if prop.IsReadonly {
		g.output.WriteString("`readonly` ")
	}

	// Write property name and type
	if prop.Name != nil {
		g.output.WriteString(fmt.Sprintf("**%s**", prop.Name.Value))
	}
	if prop.Type != nil {
		g.output.WriteString(fmt.Sprintf(": `%s`", prop.Type.String()))
	}

	g.output.WriteString("\n")
}

// writeMethodDoc writes documentation for a class method
func (g *Generator) writeMethodDoc(method *ast.FunctionDeclaration) {
	g.output.WriteString("- ")

	// Write modifiers
	if method.Visibility != "" {
		g.output.WriteString(fmt.Sprintf("`%s` ", method.Visibility))
	}
	if method.IsStatic {
		g.output.WriteString("`static` ")
	}
	if method.IsAbstract {
		g.output.WriteString("`abstract` ")
	}
	if method.IsAsync {
		g.output.WriteString("`async` ")
	}

	// Write method signature
	g.output.WriteString("**")
	if method.Name != nil {
		g.output.WriteString(method.Name.Value)
	}
	g.output.WriteString("**(")

	// Write parameters
	for i, param := range method.Parameters {
		if i > 0 {
			g.output.WriteString(", ")
		}
		g.output.WriteString(param.Name.Value)
		if param.Type != nil {
			g.output.WriteString(fmt.Sprintf(": `%s`", param.Type.String()))
		}
	}

	g.output.WriteString(")")

	// Write return type
	if method.ReturnType != nil {
		g.output.WriteString(fmt.Sprintf(": `%s`", method.ReturnType.String()))
	}

	g.output.WriteString("\n")
}

// writeConstructorDoc writes documentation for a constructor
func (g *Generator) writeConstructorDoc(constructor *ast.ConstructorDeclaration) {
	g.output.WriteString("- **constructor**(")

	// Write parameters
	for i, param := range constructor.Parameters {
		if i > 0 {
			g.output.WriteString(", ")
		}
		g.output.WriteString(param.Name.Value)
		if param.Type != nil {
			g.output.WriteString(fmt.Sprintf(": `%s`", param.Type.String()))
		}
	}

	g.output.WriteString(")\n")
}

// writeFunctionDoc writes documentation for a function
func (g *Generator) writeFunctionDoc(stmt *ast.FunctionDeclaration) {
	g.output.WriteString("### function ")

	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	// Write generic parameters
	if len(stmt.GenericParams) > 0 {
		g.output.WriteString("<")
		for i, tp := range stmt.GenericParams {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(tp.Value)
		}
		g.output.WriteString(">")
	}

	g.output.WriteString("\n\n")

	// Write signature
	g.output.WriteString("```lunar\n")
	if stmt.IsAsync {
		g.output.WriteString("async ")
	}
	g.output.WriteString("function ")
	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	// Write generic parameters
	if len(stmt.GenericParams) > 0 {
		g.output.WriteString("<")
		for i, tp := range stmt.GenericParams {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(tp.Value)
		}
		g.output.WriteString(">")
	}

	g.output.WriteString("(")

	// Write parameters
	for i, param := range stmt.Parameters {
		if i > 0 {
			g.output.WriteString(", ")
		}
		g.output.WriteString(param.Name.Value)
		if param.Type != nil {
			g.output.WriteString(fmt.Sprintf(": %s", param.Type.String()))
		}
	}

	g.output.WriteString(")")

	// Write return type
	if stmt.ReturnType != nil {
		g.output.WriteString(fmt.Sprintf(": %s", stmt.ReturnType.String()))
	}

	g.output.WriteString("\n```\n\n")
	g.output.WriteString("---\n\n")
}

// writeInterfaceDoc writes documentation for an interface
func (g *Generator) writeInterfaceDoc(stmt *ast.InterfaceDeclaration) {
	g.output.WriteString("### interface ")

	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	g.output.WriteString("\n\n")

	// Write extends clause
	if len(stmt.Extends) > 0 {
		g.output.WriteString("**Extends:** ")
		for i, ext := range stmt.Extends {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(fmt.Sprintf("`%s`", ext.String()))
		}
		g.output.WriteString("\n\n")
	}

	// Write properties
	if len(stmt.Properties) > 0 {
		g.output.WriteString("**Properties:**\n\n")
		for _, prop := range stmt.Properties {
			g.output.WriteString(fmt.Sprintf("- **%s**", prop.Name.Value))
			if prop.Type != nil {
				g.output.WriteString(fmt.Sprintf(": `%s`", prop.Type.String()))
			}
			g.output.WriteString("\n")
		}
	}

	// Write methods
	if len(stmt.Methods) > 0 {
		g.output.WriteString("\n**Methods:**\n\n")
		for _, method := range stmt.Methods {
			g.output.WriteString(fmt.Sprintf("- **%s**(", method.Name.Value))

			// Write parameters
			for i, param := range method.Parameters {
				if i > 0 {
					g.output.WriteString(", ")
				}
				g.output.WriteString(param.Name.Value)
				if param.Type != nil {
					g.output.WriteString(fmt.Sprintf(": `%s`", param.Type.String()))
				}
			}

			g.output.WriteString(")")

			if method.ReturnType != nil {
				g.output.WriteString(fmt.Sprintf(": `%s`", method.ReturnType.String()))
			}

			g.output.WriteString("\n")
		}
	}

	g.output.WriteString("\n---\n\n")
}

// writeEnumDoc writes documentation for an enum
func (g *Generator) writeEnumDoc(stmt *ast.EnumDeclaration) {
	g.output.WriteString("### enum ")

	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	g.output.WriteString("\n\n")

	// Write members
	if len(stmt.Members) > 0 {
		g.output.WriteString("**Values:**\n\n")
		for _, member := range stmt.Members {
			g.output.WriteString(fmt.Sprintf("- `%s`", member.Name.Value))
			if member.Value != nil {
				g.output.WriteString(fmt.Sprintf(" = `%s`", member.Value.String()))
			}
			g.output.WriteString("\n")
		}
	}

	g.output.WriteString("\n---\n\n")
}

// writeTypeAliasDoc writes documentation for a type alias
func (g *Generator) writeTypeAliasDoc(stmt *ast.TypeDeclaration) {
	g.output.WriteString("### type ")

	if stmt.Name != nil {
		g.output.WriteString(stmt.Name.Value)
	}

	// Write generic parameters
	if len(stmt.GenericParams) > 0 {
		g.output.WriteString("<")
		for i, tp := range stmt.GenericParams {
			if i > 0 {
				g.output.WriteString(", ")
			}
			g.output.WriteString(tp.Value)
		}
		g.output.WriteString(">")
	}

	g.output.WriteString("\n\n")

	// Write type
	if stmt.Type != nil {
		g.output.WriteString("```lunar\n")
		g.output.WriteString(fmt.Sprintf("type %s", stmt.Name.Value))

		if len(stmt.GenericParams) > 0 {
			g.output.WriteString("<")
			for i, tp := range stmt.GenericParams {
				if i > 0 {
					g.output.WriteString(", ")
				}
				g.output.WriteString(tp.Value)
			}
			g.output.WriteString(">")
		}

		g.output.WriteString(fmt.Sprintf(" = %s\n", stmt.Type.String()))
		g.output.WriteString("```\n\n")
	}

	g.output.WriteString("---\n\n")
}

// writeVariableDoc writes documentation for a variable
func (g *Generator) writeVariableDoc(stmt *ast.VariableDeclaration) {
	g.output.WriteString("### ")

	// Write first variable name
	if len(stmt.Names) > 0 && stmt.Names[0] != nil {
		g.output.WriteString(stmt.Names[0].Value)
	}

	g.output.WriteString("\n\n")

	// Write type if available
	if len(stmt.Types) > 0 && stmt.Types[0] != nil {
		g.output.WriteString(fmt.Sprintf("**Type:** `%s`\n\n", stmt.Types[0].String()))
	}

	g.output.WriteString("---\n\n")
}
