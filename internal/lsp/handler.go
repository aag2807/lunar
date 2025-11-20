package lsp

import (
	"encoding/json"
	"fmt"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"strings"
)

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(content json.RawMessage) error {
	var request struct {
		ID     interface{}      `json:"id"`
		Params InitializeParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(request.ID, InvalidParams, err.Error())
	}

	s.rootURI = request.Params.RootURI
	s.logger.Printf("Initializing with root URI: %s", s.rootURI)

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    SyncFull,
				Save: &SaveOptions{
					IncludeText: true,
				},
			},
			HoverProvider:      true,
			DefinitionProvider: true,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{".", ":"},
				ResolveProvider:   false,
			},
			ReferencesProvider: true,
		},
	}

	return s.sendResponse(request.ID, result)
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized() error {
	s.initialized = true
	s.logger.Println("Server initialized")
	return nil
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(id interface{}) error {
	s.shutdown = true
	return s.sendResponse(id, nil)
}

// handleExit handles the exit notification
func (s *Server) handleExit() error {
	if s.shutdown {
		// Normal exit
		return nil
	}
	// Abnormal exit
	return fmt.Errorf("exit without shutdown")
}

// handleDidOpen handles textDocument/didOpen
func (s *Server) handleDidOpen(content json.RawMessage) error {
	var params struct {
		Params DidOpenTextDocumentParams `json:"params"`
	}

	if err := json.Unmarshal(content, &params); err != nil {
		return err
	}

	doc := params.Params.TextDocument
	s.documents.Open(doc.URI, doc.Version, doc.Text)
	s.logger.Printf("Opened document: %s", doc.URI)

	// Publish diagnostics
	s.publishDiagnostics(doc.URI)

	return nil
}

// handleDidChange handles textDocument/didChange
func (s *Server) handleDidChange(content json.RawMessage) error {
	var params struct {
		Params DidChangeTextDocumentParams `json:"params"`
	}

	if err := json.Unmarshal(content, &params); err != nil {
		return err
	}

	doc := params.Params.TextDocument
	changes := params.Params.ContentChanges

	// We use full sync, so take the last change
	if len(changes) > 0 {
		lastChange := changes[len(changes)-1]
		s.documents.Update(doc.URI, doc.Version, lastChange.Text)
	}

	// Publish diagnostics
	s.publishDiagnostics(doc.URI)

	return nil
}

// handleDidClose handles textDocument/didClose
func (s *Server) handleDidClose(content json.RawMessage) error {
	var params struct {
		Params DidCloseTextDocumentParams `json:"params"`
	}

	if err := json.Unmarshal(content, &params); err != nil {
		return err
	}

	s.documents.Close(params.Params.TextDocument.URI)
	s.logger.Printf("Closed document: %s", params.Params.TextDocument.URI)

	// Clear diagnostics
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         params.Params.TextDocument.URI,
		Diagnostics: []Diagnostic{},
	})

	return nil
}

// handleDidSave handles textDocument/didSave
func (s *Server) handleDidSave(content json.RawMessage) error {
	var params struct {
		Params DidSaveTextDocumentParams `json:"params"`
	}

	if err := json.Unmarshal(content, &params); err != nil {
		return err
	}

	// If text is included, update the document
	if params.Params.Text != "" {
		s.documents.Update(params.Params.TextDocument.URI, 0, params.Params.Text)
	}

	// Publish diagnostics
	s.publishDiagnostics(params.Params.TextDocument.URI)

	return nil
}

// handleHover handles textDocument/hover
func (s *Server) handleHover(content json.RawMessage, id interface{}) error {
	var request struct {
		Params TextDocumentPositionParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, nil)
	}

	word := doc.GetWordAtPosition(request.Params.Position)
	if word == "" {
		return s.sendResponse(id, nil)
	}

	// Get type information
	typeInfo := s.getTypeInfo(doc.Content, word, request.Params.Position)
	if typeInfo == "" {
		return s.sendResponse(id, nil)
	}

	hover := Hover{
		Contents: MarkupContent{
			Kind:  Markdown,
			Value: typeInfo,
		},
	}

	return s.sendResponse(id, hover)
}

// handleDefinition handles textDocument/definition
func (s *Server) handleDefinition(content json.RawMessage, id interface{}) error {
	var request struct {
		Params TextDocumentPositionParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, nil)
	}

	word := doc.GetWordAtPosition(request.Params.Position)
	if word == "" {
		return s.sendResponse(id, nil)
	}

	// Find definition location
	location := s.findDefinition(doc.Content, word, request.Params.TextDocument.URI)
	if location == nil {
		return s.sendResponse(id, nil)
	}

	return s.sendResponse(id, location)
}

// handleReferences handles textDocument/references
func (s *Server) handleReferences(content json.RawMessage, id interface{}) error {
	var request struct {
		Params ReferenceParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, []Location{})
	}

	word := doc.GetWordAtPosition(request.Params.Position)
	if word == "" {
		return s.sendResponse(id, []Location{})
	}

	// Find all references to this symbol
	locations := s.findReferences(doc.Content, word, request.Params.TextDocument.URI, request.Params.Context.IncludeDeclaration)

	return s.sendResponse(id, locations)
}

// handleCompletion handles textDocument/completion
func (s *Server) handleCompletion(content json.RawMessage, id interface{}) error {
	var request struct {
		Params CompletionParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, CompletionList{Items: []CompletionItem{}})
	}

	items := s.getCompletions(doc, request.Params.Position)

	result := CompletionList{
		IsIncomplete: false,
		Items:        items,
	}

	return s.sendResponse(id, result)
}

// getTypeInfo returns type information for a symbol
func (s *Server) getTypeInfo(content string, word string, pos Position) string {
	l := lexer.New(content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return ""
	}

	checker := types.NewChecker()
	checker.Check(statements)

	// Look up the symbol in the type environment
	if typ, ok := checker.GetEnv().Get(word); ok {
		return formatTypeInfo(word, typ)
	}

	return ""
}

// findDefinition finds the definition location of a symbol
func (s *Server) findDefinition(content string, word string, uri string) *Location {
	l := lexer.New(content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return nil
	}

	// Search for definition in AST
	for _, stmt := range statements {
		if loc := findDefinitionInStatement(stmt, word, uri); loc != nil {
			return loc
		}
	}

	return nil
}

// getCompletions returns completion items for a position
func (s *Server) getCompletions(doc *Document, pos Position) []CompletionItem {
	items := []CompletionItem{}

	l := lexer.New(doc.Content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		// Still provide keyword completions
		return s.getKeywordCompletions()
	}

	checker := types.NewChecker()
	checker.Check(statements)

	// Get line content to determine context
	lineContent := doc.GetLineContent(pos.Line)
	beforeCursor := ""
	if pos.Character <= len(lineContent) {
		beforeCursor = lineContent[:pos.Character]
	}

	// Check if we're completing after a dot
	if strings.HasSuffix(strings.TrimSpace(beforeCursor), ".") {
		// Get the object before the dot
		trimmed := strings.TrimSpace(beforeCursor)
		objName := getObjectBeforeDot(trimmed)
		if objName != "" {
			items = append(items, s.getMemberCompletions(checker.GetEnv(), objName)...)
		}
	} else {
		// Get all symbols in scope
		for name, typ := range checker.GetEnv().GetAll() {
			item := CompletionItem{
				Label:  name,
				Detail: types.TypeString(typ),
				Kind:   getCompletionKind(typ),
			}
			items = append(items, item)
		}

		// Add keyword completions
		items = append(items, s.getKeywordCompletions()...)
	}

	return items
}

// getKeywordCompletions returns keyword completion items
func (s *Server) getKeywordCompletions() []CompletionItem {
	keywords := []string{
		"local", "function", "end", "if", "then", "else", "elseif",
		"while", "do", "for", "in", "repeat", "until", "return",
		"break", "continue", "class", "extends", "implements",
		"interface", "enum", "namespace", "import", "export",
		"public", "private", "protected", "static", "readonly",
		"abstract", "override", "async", "await", "nil", "true", "false",
	}

	items := make([]CompletionItem, len(keywords))
	for i, kw := range keywords {
		items[i] = CompletionItem{
			Label: kw,
			Kind:  KeywordCompletion,
		}
	}

	return items
}

// getMemberCompletions returns member completions for an object
func (s *Server) getMemberCompletions(env *types.Environment, objName string) []CompletionItem {
	items := []CompletionItem{}

	typ, ok := env.Get(objName)
	if !ok {
		return items
	}

	switch t := typ.(type) {
	case *types.ClassType:
		// Add properties
		for name, propType := range t.Properties {
			items = append(items, CompletionItem{
				Label:  name,
				Detail: types.TypeString(propType),
				Kind:   PropertyCompletion,
			})
		}
		// Add methods
		for name, methodType := range t.Methods {
			items = append(items, CompletionItem{
				Label:  name,
				Detail: types.TypeString(methodType),
				Kind:   MethodCompletion,
			})
		}
	}

	return items
}

// Helper functions

func formatTypeInfo(name string, typ types.Type) string {
	switch t := typ.(type) {
	case *types.FunctionType:
		params := make([]string, len(t.Parameters))
		for i, p := range t.Parameters {
			params[i] = types.TypeString(p)
		}
		returnType := types.TypeString(t.ReturnType)
		return fmt.Sprintf("```lunar\nfunction %s(%s): %s\n```",
			name, strings.Join(params, ", "), returnType)
	case *types.ClassType:
		return fmt.Sprintf("```lunar\nclass %s\n```", t.Name)
	default:
		return fmt.Sprintf("```lunar\n%s: %s\n```", name, types.TypeString(typ))
	}
}

func getCompletionKind(typ types.Type) CompletionItemKind {
	switch typ.(type) {
	case *types.FunctionType:
		return FunctionCompletion
	case *types.ClassType:
		return ClassCompletion
	case *types.InterfaceType:
		return InterfaceKind
	case *types.EnumType:
		return EnumCompletion
	default:
		return VariableCompletion
	}
}

func getObjectBeforeDot(s string) string {
	// Remove the trailing dot
	s = strings.TrimSuffix(s, ".")
	// Get the last word
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func findDefinitionInStatement(stmt interface{}, word string, uri string) *Location {
	// This is a simplified implementation
	// A full implementation would traverse the entire AST

	switch s := stmt.(type) {
	case interface{ GetName() string }:
		if name, ok := s.(interface{ GetName() string }); ok {
			if name.GetName() == word {
				if tok, ok := s.(interface{ GetToken() interface{ GetLine() int } }); ok {
					line := tok.GetToken().GetLine() - 1
					return &Location{
						URI: uri,
						Range: Range{
							Start: Position{Line: line, Character: 0},
							End:   Position{Line: line, Character: len(word)},
						},
					}
				}
			}
		}
	}

	return nil
}

// findReferences finds all references to a symbol in the document
func (s *Server) findReferences(content string, symbol string, uri string, includeDeclaration bool) []Location {
	locations := []Location{}
	lines := strings.Split(content, "\n")

	// Find all occurrences of the symbol
	for lineNum, line := range lines {
		// Simple approach: find all word occurrences
		// This can be improved with proper AST traversal
		startIdx := 0
		for {
			idx := strings.Index(line[startIdx:], symbol)
			if idx == -1 {
				break
			}

			actualIdx := startIdx + idx

			// Check if this is a complete word (not part of another identifier)
			beforeOK := actualIdx == 0 || !isIdentifierChar(line[actualIdx-1])
			afterOK := actualIdx+len(symbol) >= len(line) || !isIdentifierChar(line[actualIdx+len(symbol)])

			if beforeOK && afterOK {
				// Check if this is a reference or declaration
				isDecl := isDeclarationLine(line, actualIdx, symbol)

				if includeDeclaration || !isDecl {
					locations = append(locations, Location{
						URI: uri,
						Range: Range{
							Start: Position{Line: lineNum, Character: actualIdx},
							End:   Position{Line: lineNum, Character: actualIdx + len(symbol)},
						},
					})
				}
			}

			startIdx = actualIdx + 1
		}
	}

	return locations
}

// isIdentifierChar checks if a byte is a valid identifier character
func isIdentifierChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_'
}

// isDeclarationLine checks if a line contains a declaration of the symbol
func isDeclarationLine(line string, idx int, symbol string) bool {
	// Simple heuristic: check for common declaration patterns
	trimmedBefore := strings.TrimSpace(line[:idx])

	// Check for variable declaration: local name, local name:, const name, name =
	if strings.HasSuffix(trimmedBefore, "local") ||
		strings.HasSuffix(trimmedBefore, "const") ||
		strings.Contains(line, "function "+symbol) ||
		strings.Contains(line, "class "+symbol) ||
		strings.Contains(line, "interface "+symbol) ||
		strings.Contains(line, "enum "+symbol) {
		return true
	}

	return false
}
