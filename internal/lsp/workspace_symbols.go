package lsp

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"os"
	"path/filepath"
	"strings"
)

// getWorkspaceSymbols searches for symbols across the workspace
func (s *Server) getWorkspaceSymbols(query string) []SymbolInformation {
	if s.rootURI == "" {
		return []SymbolInformation{}
	}

	rootPath := strings.TrimPrefix(s.rootURI, "file://")
	symbols := []SymbolInformation{}

	// Walk through .lunar files in the workspace
	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		// Skip directories and non-.lunar files
		if info.IsDir() || !strings.HasSuffix(path, ".lunar") {
			return nil
		}

		// Extract symbols from this file
		fileSymbols := s.extractWorkspaceSymbolsFromFile(path, query)
		symbols = append(symbols, fileSymbols...)

		return nil
	})

	return symbols
}

// extractWorkspaceSymbolsFromFile extracts symbols from a file matching the query
func (s *Server) extractWorkspaceSymbolsFromFile(filePath string, query string) []SymbolInformation {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return []SymbolInformation{}
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return []SymbolInformation{}
	}

	// Convert file path to URI
	uri := "file://" + filepath.ToSlash(filePath)

	symbols := []SymbolInformation{}
	for _, stmt := range statements {
		fileSymbols := s.extractWorkspaceSymbolsFromStatement(stmt, uri, query)
		symbols = append(symbols, fileSymbols...)
	}

	return symbols
}

// extractWorkspaceSymbolsFromStatement extracts symbols from a statement
func (s *Server) extractWorkspaceSymbolsFromStatement(stmt ast.Statement, uri string, query string) []SymbolInformation {
	symbols := []SymbolInformation{}

	switch node := stmt.(type) {
	case *ast.FunctionDeclaration:
		if node.Name != nil && s.matchesQuery(node.Name.Value, query) {
			line := node.Name.Token.Line
			symbols = append(symbols, SymbolInformation{
				Name: node.Name.Value,
				Kind: FunctionSymbol,
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: line - 1, Character: 0},
						End:   Position{Line: line, Character: 0},
					},
				},
			})
		}

	case *ast.ClassDeclaration:
		if node.Name != nil && s.matchesQuery(node.Name.Value, query) {
			line := node.Name.Token.Line
			symbols = append(symbols, SymbolInformation{
				Name: node.Name.Value,
				Kind: ClassSymbol,
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: line - 1, Character: 0},
						End:   Position{Line: line, Character: 0},
					},
				},
			})
		}

		// Add methods
		for _, method := range node.Methods {
			if method.Name != nil && s.matchesQuery(method.Name.Value, query) {
				methodLine := method.Name.Token.Line
				symbols = append(symbols, SymbolInformation{
					Name:          method.Name.Value,
					Kind:          MethodSymbol,
					ContainerName: node.Name.Value,
					Location: Location{
						URI: uri,
						Range: Range{
							Start: Position{Line: methodLine - 1, Character: 0},
							End:   Position{Line: methodLine, Character: 0},
						},
					},
				})
			}
		}

	case *ast.InterfaceDeclaration:
		if node.Name != nil && s.matchesQuery(node.Name.Value, query) {
			line := node.Name.Token.Line
			symbols = append(symbols, SymbolInformation{
				Name: node.Name.Value,
				Kind: InterfaceSymbol,
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: line - 1, Character: 0},
						End:   Position{Line: line, Character: 0},
					},
				},
			})
		}

		// Add methods
		for _, method := range node.Methods {
			if method.Name != nil && s.matchesQuery(method.Name.Value, query) {
				methodLine := method.Name.Token.Line
				symbols = append(symbols, SymbolInformation{
					Name:          method.Name.Value,
					Kind:          MethodSymbol,
					ContainerName: node.Name.Value,
					Location: Location{
						URI: uri,
						Range: Range{
							Start: Position{Line: methodLine - 1, Character: 0},
							End:   Position{Line: methodLine, Character: 0},
						},
					},
				})
			}
		}

	case *ast.EnumDeclaration:
		if node.Name != nil && s.matchesQuery(node.Name.Value, query) {
			line := node.Name.Token.Line
			symbols = append(symbols, SymbolInformation{
				Name: node.Name.Value,
				Kind: EnumSymbol,
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: line - 1, Character: 0},
						End:   Position{Line: line, Character: 0},
					},
				},
			})
		}

	case *ast.VariableDeclaration:
		if node.IsConstant && len(node.Names) > 0 && s.matchesQuery(node.Names[0].Value, query) {
			line := node.Names[0].Token.Line
			symbols = append(symbols, SymbolInformation{
				Name: node.Names[0].Value,
				Kind: ConstantSymbol,
				Location: Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: line - 1, Character: 0},
						End:   Position{Line: line, Character: 0},
					},
				},
			})
		}
	}

	return symbols
}

// matchesQuery checks if a symbol name matches the search query
func (s *Server) matchesQuery(name string, query string) bool {
	if query == "" {
		return true // Empty query matches everything
	}

	// Case-insensitive substring match
	nameLower := strings.ToLower(name)
	queryLower := strings.ToLower(query)

	// Check for exact match first
	if nameLower == queryLower {
		return true
	}

	// Check if query is a substring
	if strings.Contains(nameLower, queryLower) {
		return true
	}

	// Check if query matches start of name
	if strings.HasPrefix(nameLower, queryLower) {
		return true
	}

	// Check for camelCase matching (e.g., "gFC" matches "getUserFromCache")
	if s.matchesCamelCase(name, query) {
		return true
	}

	return false
}

// matchesCamelCase checks if query matches the camelCase pattern of name
func (s *Server) matchesCamelCase(name string, query string) bool {
	if query == "" {
		return true
	}

	queryIndex := 0
	queryUpper := strings.ToUpper(query)

	for _, ch := range name {
		if queryIndex >= len(queryUpper) {
			return true
		}

		// Match uppercase letters or start of word
		if ch >= 'A' && ch <= 'Z' {
			if ch == rune(queryUpper[queryIndex]) {
				queryIndex++
			}
		} else if queryIndex == 0 && ch == rune(strings.ToLower(string(queryUpper[0]))[0]) {
			// Match first character even if lowercase
			queryIndex++
		}
	}

	return queryIndex >= len(queryUpper)
}
