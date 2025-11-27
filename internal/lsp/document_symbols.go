package lsp

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"strings"
)

// getDocumentSymbols extracts all symbols from a document for outline view
func (s *Server) getDocumentSymbols(doc *Document) []DocumentSymbol {
	l := lexer.New(doc.Content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return []DocumentSymbol{}
	}

	symbols := []DocumentSymbol{}
	for _, stmt := range statements {
		symbol := s.extractSymbolFromStatement(stmt)
		if symbol != nil {
			symbols = append(symbols, *symbol)
		}
	}

	return symbols
}

// extractSymbolFromStatement extracts a DocumentSymbol from an AST statement
func (s *Server) extractSymbolFromStatement(stmt ast.Statement) *DocumentSymbol {
	switch node := stmt.(type) {
	case *ast.FunctionDeclaration:
		return s.createFunctionSymbol(node)

	case *ast.ClassDeclaration:
		return s.createClassSymbol(node)

	case *ast.InterfaceDeclaration:
		return s.createInterfaceSymbol(node)

	case *ast.EnumDeclaration:
		return s.createEnumSymbol(node)

	case *ast.VariableDeclaration:
		// Only show const/top-level variables
		if node.IsConstant && len(node.Names) > 0 {
			return s.createVariableSymbol(node)
		}

	case *ast.DeclareStatement:
		return s.createDeclareSymbol(node)
	}

	return nil
}

// createFunctionSymbol creates a symbol for a function declaration
func (s *Server) createFunctionSymbol(node *ast.FunctionDeclaration) *DocumentSymbol {
	funcName := "(anonymous)"
	if node.Name != nil {
		funcName = node.Name.Value
	}

	// Build function signature detail
	paramStrs := []string{}
	for _, param := range node.Parameters {
		if param.Name != nil {
			paramStrs = append(paramStrs, param.Name.Value)
		}
	}

	detail := "(" + strings.Join(paramStrs, ", ") + ")"
	if node.ReturnType != nil {
		detail += ": " + node.ReturnType.String()
	}

	line := 0
	if node.Name != nil {
		line = node.Name.Token.Line
	}

	symbol := &DocumentSymbol{
		Name:   funcName,
		Detail: detail,
		Kind:   FunctionSymbol,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len("function " + funcName)},
		},
		Children: []DocumentSymbol{},
	}

	// Add nested function symbols from the body
	if node.Body != nil {
		for _, stmt := range node.Body.Statements {
			if childSymbol := s.extractSymbolFromStatement(stmt); childSymbol != nil {
				symbol.Children = append(symbol.Children, *childSymbol)
			}
		}
	}

	return symbol
}

// createClassSymbol creates a symbol for a class declaration
func (s *Server) createClassSymbol(node *ast.ClassDeclaration) *DocumentSymbol {
	className := "(anonymous)"
	line := 0
	if node.Name != nil {
		className = node.Name.Value
		line = node.Name.Token.Line
	}

	symbol := &DocumentSymbol{
		Name:   className,
		Detail: "class",
		Kind:   ClassSymbol,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len("class " + className)},
		},
		Children: []DocumentSymbol{},
	}

	// Add properties and methods as children
	for _, prop := range node.Properties {
		if prop.Name != nil {
			propLine := prop.Name.Token.Line
			propName := prop.Name.Value
			propType := ""
			if prop.Type != nil {
				propType = prop.Type.String()
			}

			propSymbol := &DocumentSymbol{
				Name:   propName,
				Detail: propType,
				Kind:   PropertySymbol,
				Range: Range{
					Start: Position{Line: propLine - 1, Character: 0},
					End:   Position{Line: propLine, Character: 0},
				},
				SelectionRange: Range{
					Start: Position{Line: propLine - 1, Character: 0},
					End:   Position{Line: propLine - 1, Character: len(propName)},
				},
			}
			symbol.Children = append(symbol.Children, *propSymbol)
		}
	}

	for _, method := range node.Methods {
		methodSymbol := s.createFunctionSymbol(method)
		if methodSymbol != nil {
			methodSymbol.Kind = MethodSymbol
			symbol.Children = append(symbol.Children, *methodSymbol)
		}
	}

	return symbol
}

// createInterfaceSymbol creates a symbol for an interface declaration
func (s *Server) createInterfaceSymbol(node *ast.InterfaceDeclaration) *DocumentSymbol {
	interfaceName := "(anonymous)"
	line := 0
	if node.Name != nil {
		interfaceName = node.Name.Value
		line = node.Name.Token.Line
	}

	symbol := &DocumentSymbol{
		Name:   interfaceName,
		Detail: "interface",
		Kind:   InterfaceSymbol,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len("interface " + interfaceName)},
		},
		Children: []DocumentSymbol{},
	}

	// Add properties and methods
	for _, prop := range node.Properties {
		if prop.Name != nil {
			propLine := prop.Name.Token.Line
			propName := prop.Name.Value
			propType := ""
			if prop.Type != nil {
				propType = prop.Type.String()
			}

			propSymbol := &DocumentSymbol{
				Name:   propName,
				Detail: propType,
				Kind:   PropertySymbol,
				Range: Range{
					Start: Position{Line: propLine - 1, Character: 0},
					End:   Position{Line: propLine, Character: 0},
				},
				SelectionRange: Range{
					Start: Position{Line: propLine - 1, Character: 0},
					End:   Position{Line: propLine - 1, Character: len(propName)},
				},
			}
			symbol.Children = append(symbol.Children, *propSymbol)
		}
	}

	for _, method := range node.Methods {
		methodSymbol := s.createInterfaceMethodSymbol(method)
		if methodSymbol != nil {
			symbol.Children = append(symbol.Children, *methodSymbol)
		}
	}

	return symbol
}

// createInterfaceMethodSymbol creates a symbol for an interface method
func (s *Server) createInterfaceMethodSymbol(method *ast.InterfaceMethod) *DocumentSymbol {
	if method.Name == nil {
		return nil
	}

	methodName := method.Name.Value
	line := method.Name.Token.Line

	// Build method signature
	paramStrs := []string{}
	for _, param := range method.Parameters {
		if param.Name != nil {
			paramStrs = append(paramStrs, param.Name.Value)
		}
	}

	detail := "(" + strings.Join(paramStrs, ", ") + ")"
	if method.ReturnType != nil {
		detail += ": " + method.ReturnType.String()
	}

	return &DocumentSymbol{
		Name:   methodName,
		Detail: detail,
		Kind:   MethodSymbol,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len(methodName)},
		},
	}
}

// createEnumSymbol creates a symbol for an enum declaration
func (s *Server) createEnumSymbol(node *ast.EnumDeclaration) *DocumentSymbol {
	enumName := "(anonymous)"
	line := 0
	if node.Name != nil {
		enumName = node.Name.Value
		line = node.Name.Token.Line
	}

	symbol := &DocumentSymbol{
		Name:   enumName,
		Detail: "enum",
		Kind:   EnumSymbol,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len("enum " + enumName)},
		},
		Children: []DocumentSymbol{},
	}

	// Add enum members
	for _, member := range node.Members {
		if member.Name != nil {
			memberLine := member.Name.Token.Line
			memberName := member.Name.Value

			memberSymbol := &DocumentSymbol{
				Name: memberName,
				Kind: ConstantSymbol,
				Range: Range{
					Start: Position{Line: memberLine - 1, Character: 0},
					End:   Position{Line: memberLine, Character: 0},
				},
				SelectionRange: Range{
					Start: Position{Line: memberLine - 1, Character: 0},
					End:   Position{Line: memberLine - 1, Character: len(memberName)},
				},
			}
			symbol.Children = append(symbol.Children, *memberSymbol)
		}
	}

	return symbol
}

// createVariableSymbol creates a symbol for a variable declaration
func (s *Server) createVariableSymbol(node *ast.VariableDeclaration) *DocumentSymbol {
	kind := VariableSymbol
	if node.IsConstant {
		kind = ConstantSymbol
	}

	// Use first name
	if len(node.Names) == 0 {
		return nil
	}

	varName := node.Names[0].Value
	line := node.Names[0].Token.Line

	detail := ""
	if len(node.Types) > 0 && node.Types[0] != nil {
		detail = ": " + node.Types[0].String()
	}

	return &DocumentSymbol{
		Name:   varName,
		Detail: detail,
		Kind:   kind,
		Range: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line, Character: 0},
		},
		SelectionRange: Range{
			Start: Position{Line: line - 1, Character: 0},
			End:   Position{Line: line - 1, Character: len(varName)},
		},
	}
}

// createDeclareSymbol creates a symbol for a declare statement
func (s *Server) createDeclareSymbol(node *ast.DeclareStatement) *DocumentSymbol {
	switch declNode := node.Declaration.(type) {
	case *ast.FunctionDeclaration:
		return s.createFunctionSymbol(declNode)
	case *ast.VariableDeclaration:
		return s.createVariableSymbol(declNode)
	}
	return nil
}
