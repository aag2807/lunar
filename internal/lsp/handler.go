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

	// Scan for declaration files if root URI is provided
	if s.rootURI != "" {
		rootPath := strings.TrimPrefix(s.rootURI, "file://")
		s.declarations.SetRootPath(rootPath)
		s.logger.Println("Scanned for .d.lunar declaration files")
	}

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
			SignatureHelpProvider: &SignatureHelpOptions{
				TriggerCharacters:   []string{"(", ","},
				RetriggerCharacters: []string{","},
			},
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			ReferencesProvider:      true,
			RenameProvider:          true,
			CodeActionProvider: &CodeActionOptions{
				CodeActionKinds: []string{
					CodeActionKindQuickFix,
					CodeActionKindRefactor,
					CodeActionKindRefactorExtract,
					CodeActionKindRefactorInline,
					CodeActionKindRefactorRewrite,
					CodeActionKindSource,
					CodeActionKindSourceOrganizeImports,
				},
			},
			InlayHintProvider: &InlayHintOptions{
				ResolveProvider: false,
			},
			SemanticTokensProvider: &SemanticTokensOptions{
				Legend: SemanticTokensLegend{
					TokenTypes: []string{
						SemanticTokenTypeKeyword,
						SemanticTokenTypeType,
						SemanticTokenTypeClass,
						SemanticTokenTypeInterface,
						SemanticTokenTypeFunction,
						SemanticTokenTypeVariable,
						SemanticTokenTypeParameter,
						SemanticTokenTypeProperty,
						SemanticTokenTypeString,
						SemanticTokenTypeNumber,
						SemanticTokenTypeComment,
						SemanticTokenTypeOperator,
						SemanticTokenTypeModifier,
					},
					TokenModifiers: []string{
						SemanticTokenModifierDeclaration,
						SemanticTokenModifierDefinition,
						SemanticTokenModifierReadonly,
					},
				},
				Full: true,
			},
		},
	}

	// Mark server as initialized after sending response
	s.initialized = true
	s.logger.Println("Server initialization complete")

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

// handleRename handles textDocument/rename
func (s *Server) handleRename(content json.RawMessage, id interface{}) error {
	var request struct {
		Params RenameParams `json:"params"`
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

	// Find all references to this symbol (including declaration)
	locations := s.findReferences(doc.Content, word, request.Params.TextDocument.URI, true)

	// Create text edits for all locations
	edits := make([]TextEdit, len(locations))
	for i, loc := range locations {
		edits[i] = TextEdit{
			Range:   loc.Range,
			NewText: request.Params.NewName,
		}
	}

	// Create workspace edit
	workspaceEdit := WorkspaceEdit{
		Changes: map[string][]TextEdit{
			request.Params.TextDocument.URI: edits,
		},
	}

	return s.sendResponse(id, workspaceEdit)
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

// handleSignatureHelp handles textDocument/signatureHelp
func (s *Server) handleSignatureHelp(content json.RawMessage, id interface{}) error {
	var request struct {
		Params SignatureHelpParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendError(id, InvalidParams, "Document not found")
	}

	result := s.getSignatureHelp(doc, request.Params)
	return s.sendResponse(id, result)
}

// handleDocumentSymbol handles textDocument/documentSymbol
func (s *Server) handleDocumentSymbol(content json.RawMessage, id interface{}) error {
	var request struct {
		Params DocumentSymbolParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendError(id, InvalidParams, "Document not found")
	}

	result := s.getDocumentSymbols(doc)
	return s.sendResponse(id, result)
}

// handleWorkspaceSymbol handles workspace/symbol
func (s *Server) handleWorkspaceSymbol(content json.RawMessage, id interface{}) error {
	var request struct {
		Params WorkspaceSymbolParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	result := s.getWorkspaceSymbols(request.Params.Query)
	return s.sendResponse(id, result)
}

// handleCodeAction handles textDocument/codeAction
func (s *Server) handleCodeAction(content json.RawMessage, id interface{}) error {
	var request struct {
		Params CodeActionParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, []CodeAction{})
	}

	actions := s.getCodeActions(doc, request.Params)

	return s.sendResponse(id, actions)
}

// getCodeActions returns code actions for a range
func (s *Server) getCodeActions(doc *Document, params CodeActionParams) []CodeAction {
	actions := []CodeAction{}

	// Get diagnostics in the range
	for _, diag := range params.Context.Diagnostics {
		// Check if diagnostic is in the requested range
		if rangesOverlap(diag.Range, params.Range) {
			actions = append(actions, s.getQuickFixesForDiagnostic(doc, diag, params.TextDocument.URI)...)
		}
	}

	// Add general refactoring actions
	actions = append(actions, s.getRefactoringActions(doc, params)...)

	return actions
}

// getQuickFixesForDiagnostic returns quick fixes for a diagnostic
func (s *Server) getQuickFixesForDiagnostic(doc *Document, diag Diagnostic, uri string) []CodeAction {
	actions := []CodeAction{}

	// Check for undefined variable errors
	if strings.Contains(diag.Message, "undefined") {
		// Extract variable name from diagnostic message
		if varName := extractUndefinedVariable(diag.Message); varName != "" {
			// Quick fix: Add local declaration
			action := CodeAction{
				Title: fmt.Sprintf("Declare local variable '%s'", varName),
				Kind:  CodeActionKindQuickFix,
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						uri: {
							{
								Range: Range{
									Start: Position{Line: diag.Range.Start.Line, Character: 0},
									End:   Position{Line: diag.Range.Start.Line, Character: 0},
								},
								NewText: fmt.Sprintf("local %s\n", varName),
							},
						},
					},
				},
			}
			actions = append(actions, action)

			// Quick fix: Add type annotation
			action2 := CodeAction{
				Title: fmt.Sprintf("Declare '%s: any'", varName),
				Kind:  CodeActionKindQuickFix,
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						uri: {
							{
								Range: Range{
									Start: Position{Line: diag.Range.Start.Line, Character: 0},
									End:   Position{Line: diag.Range.Start.Line, Character: 0},
								},
								NewText: fmt.Sprintf("local %s: any\n", varName),
							},
						},
					},
				},
			}
			actions = append(actions, action2)
		}
	}

	// Check for type mismatch errors
	if strings.Contains(diag.Message, "type mismatch") || strings.Contains(diag.Message, "expected") {
		// Quick fix: Add type cast
		lineContent := doc.GetLineContent(diag.Range.Start.Line)
		word := getWordAtRange(lineContent, diag.Range)
		if word != "" {
			action := CodeAction{
				Title: "Add explicit type cast",
				Kind:  CodeActionKindQuickFix,
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						uri: {
							{
								Range:   diag.Range,
								NewText: fmt.Sprintf("(%s as any)", word),
							},
						},
					},
				},
			}
			actions = append(actions, action)
		}
	}

	// Check for missing import errors
	if strings.Contains(diag.Message, "module") && strings.Contains(diag.Message, "not found") {
		if moduleName := extractModuleName(diag.Message); moduleName != "" {
			action := CodeAction{
				Title: fmt.Sprintf("Import module '%s'", moduleName),
				Kind:  CodeActionKindQuickFix,
				Edit: &WorkspaceEdit{
					Changes: map[string][]TextEdit{
						uri: {
							{
								Range: Range{
									Start: Position{Line: 0, Character: 0},
									End:   Position{Line: 0, Character: 0},
								},
								NewText: fmt.Sprintf("import \"%s\"\n", moduleName),
							},
						},
					},
				},
			}
			actions = append(actions, action)
		}
	}

	return actions
}

// getRefactoringActions returns general refactoring actions
func (s *Server) getRefactoringActions(doc *Document, params CodeActionParams) []CodeAction {
	actions := []CodeAction{}

	// Only provide refactoring actions if a range is selected
	if params.Range.Start.Line == params.Range.End.Line &&
		params.Range.Start.Character == params.Range.End.Character {
		return actions
	}

	// Extract to function
	action := CodeAction{
		Title: "Extract to function",
		Kind:  CodeActionKindRefactorExtract,
		Edit: &WorkspaceEdit{
			Changes: map[string][]TextEdit{
				params.TextDocument.URI: {
					{
						Range:   params.Range,
						NewText: "extracted()",
					},
					{
						Range: Range{
							Start: Position{Line: params.Range.End.Line + 1, Character: 0},
							End:   Position{Line: params.Range.End.Line + 1, Character: 0},
						},
						NewText: "\nfunction extracted()\n    -- TODO: extracted code\nend\n",
					},
				},
			},
		},
	}
	actions = append(actions, action)

	return actions
}

// handleInlayHint handles textDocument/inlayHint
func (s *Server) handleInlayHint(content json.RawMessage, id interface{}) error {
	var request struct {
		Params InlayHintParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, []InlayHint{})
	}

	hints := s.getInlayHints(doc, request.Params)

	return s.sendResponse(id, hints)
}

// getInlayHints returns inlay hints for a range
func (s *Server) getInlayHints(doc *Document, params InlayHintParams) []InlayHint {
	hints := []InlayHint{}

	// Parse the document
	l := lexer.New(doc.Content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		// Return empty hints if parsing fails
		return hints
	}

	// Run type checker
	stdlibPath := getStdlibPathForLSP()
	checker := types.NewChecker(stdlibPath)
	checker.Check(statements)

	// Get all type information
	env := checker.GetEnv()

	// Add type hints for variables with inferred types
	lines := strings.Split(doc.Content, "\n")
	for lineNum, line := range lines {
		// Skip lines outside the requested range
		if lineNum < params.Range.Start.Line || lineNum > params.Range.End.Line {
			continue
		}

		// Add hints for variable declarations without explicit types
		hints = append(hints, s.getVariableTypeHints(line, lineNum, env)...)
	}

	return hints
}

// getVariableTypeHints returns type hints for variable declarations
func (s *Server) getVariableTypeHints(line string, lineNum int, env *types.Environment) []InlayHint {
	hints := []InlayHint{}

	// Check for variable declarations without type annotations
	// Pattern: local name = value (without : type)
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "local ") && !strings.Contains(trimmed, ":") {
		// Extract variable name
		rest := strings.TrimPrefix(trimmed, "local ")
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			varName := parts[0]
			// Remove any trailing = or ,
			varName = strings.TrimSuffix(varName, "=")
			varName = strings.TrimSuffix(varName, ",")

			// Look up the type in the environment
			if typ, ok := env.Get(varName); ok {
				typeStr := types.TypeString(typ)
				// Skip if type is 'any' or 'unknown'
				if typeStr != "any" && typeStr != "unknown" {
					// Find position after variable name
					idx := strings.Index(line, varName)
					if idx != -1 {
						pos := idx + len(varName)
						hint := InlayHint{
							Position: Position{
								Line:      lineNum,
								Character: pos,
							},
							Label:        fmt.Sprintf(": %s", typeStr),
							Kind:         InlayHintKindType,
							PaddingLeft:  false,
							PaddingRight: true,
						}
						hints = append(hints, hint)
					}
				}
			}
		}
	}

	// Check for function return types
	if strings.Contains(trimmed, "function ") && !strings.Contains(trimmed, "):") {
		// Function declaration without explicit return type
		if strings.Contains(trimmed, "function ") {
			// Extract function name
			start := strings.Index(trimmed, "function ") + 9
			rest := trimmed[start:]
			nameEnd := strings.Index(rest, "(")
			if nameEnd > 0 {
				funcName := strings.TrimSpace(rest[:nameEnd])
				if typ, ok := env.Get(funcName); ok {
					if fnType, ok := typ.(*types.FunctionType); ok {
						returnTypeStr := types.TypeString(fnType.ReturnType)
						if returnTypeStr != "void" && returnTypeStr != "any" {
							// Find position after closing parenthesis
							parenIdx := strings.Index(line, ")")
							if parenIdx != -1 {
								hint := InlayHint{
									Position: Position{
										Line:      lineNum,
										Character: parenIdx + 1,
									},
									Label:        fmt.Sprintf(": %s", returnTypeStr),
									Kind:         InlayHintKindType,
									PaddingLeft:  false,
									PaddingRight: true,
								}
								hints = append(hints, hint)
							}
						}
					}
				}
			}
		}
	}

	return hints
}

// Helper functions for code actions

func rangesOverlap(r1, r2 Range) bool {
	// Check if ranges overlap
	if r1.End.Line < r2.Start.Line || r2.End.Line < r1.Start.Line {
		return false
	}
	if r1.End.Line == r2.Start.Line && r1.End.Character < r2.Start.Character {
		return false
	}
	if r2.End.Line == r1.Start.Line && r2.End.Character < r1.Start.Character {
		return false
	}
	return true
}

func extractUndefinedVariable(message string) string {
	// Extract variable name from "undefined variable 'x'" or similar
	if strings.Contains(message, "'") {
		parts := strings.Split(message, "'")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func extractModuleName(message string) string {
	// Extract module name from error messages
	if strings.Contains(message, "'") {
		parts := strings.Split(message, "'")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func getWordAtRange(line string, r Range) string {
	if r.Start.Character >= len(line) || r.End.Character > len(line) {
		return ""
	}
	return line[r.Start.Character:r.End.Character]
}

// getTypeInfo returns type information for a symbol
func (s *Server) getTypeInfo(content string, word string, pos Position) string {
	l := lexer.New(content)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return ""
	}

	stdlibPath := getStdlibPathForLSP()
	checker := types.NewChecker(stdlibPath)
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

	// Get stdlib path for type checker
	stdlibPath := getStdlibPathForLSP()

	if len(p.Errors()) > 0 {
		// Still provide keyword completions and snippets
		items = append(items, s.getKeywordCompletions()...)
		items = append(items, s.addSnippetCompletions(&EnhancedCompletionContext{CursorPosition: pos})...)
		return items
	}

	checker := types.NewChecker(stdlibPath)
	checker.Check(statements)

	// Get line content to determine context
	lineContent := doc.GetLineContent(pos.Line)
	beforeCursor := ""
	if pos.Character <= len(lineContent) {
		beforeCursor = lineContent[:pos.Character]
	}

	// Check if we're completing after a dot or colon
	trimmedBefore := strings.TrimSpace(beforeCursor)
	isMethodCall := strings.HasSuffix(trimmedBefore, ":")
	isDotAccess := strings.HasSuffix(trimmedBefore, ".")

	// Extract query (partial word being typed)
	query := s.extractQueryFromLine(lineContent, pos.Character)

	// Build completion context
	context := &EnhancedCompletionContext{
		IsAfterDot:     isDotAccess,
		IsAfterColon:   isMethodCall,
		Query:          query,
		CursorPosition: pos,
	}

	if isMethodCall || isDotAccess {
		// Get the object before the dot/colon
		objName := getObjectBeforeDotOrColon(trimmedBefore)
		context.ObjectName = objName

		if objName != "" {
			// First try local scope
			items = append(items, s.getMemberCompletions(checker.GetEnv(), objName, isMethodCall)...)

			// Then try declared globals from .d.lunar files
			if declaredType, ok := s.declarations.GetDeclaredType(objName); ok {
				items = append(items, s.getMemberCompletionsFromType(objName, declaredType, isMethodCall)...)
			}
		}
	} else {
		// Get all symbols in scope
		for name, typ := range checker.GetEnv().GetAll() {
			item := CompletionItem{
				Label:  name,
				Detail: types.TypeString(typ),
				Kind:   getCompletionKind(typ),
			}
			// Enhance with documentation
			item = s.enhanceCompletionItem(item, typ, context)
			items = append(items, item)
		}

		// Add declared globals from .d.lunar files
		for name, typ := range s.declarations.GetAllDeclaredSymbols() {
			item := CompletionItem{
				Label:  name,
				Detail: types.TypeString(typ),
				Kind:   getCompletionKind(typ),
			}
			// Enhance with documentation
			item = s.enhanceCompletionItem(item, typ, context)
			items = append(items, item)
		}

		// Add keyword completions
		items = append(items, s.getKeywordCompletions()...)

		// Add snippet completions
		items = append(items, s.addSnippetCompletions(context)...)
	}

	// Sort and rank items
	items = s.sortAndRankCompletions(items, context)

	return items
}

// extractQueryFromLine extracts the partial word being typed at the cursor
func (s *Server) extractQueryFromLine(line string, cursorPos int) string {
	if cursorPos > len(line) {
		cursorPos = len(line)
	}

	// Find start of word
	start := cursorPos - 1
	for start >= 0 && (isAlphanumeric(line[start]) || line[start] == '_') {
		start--
	}
	start++

	return line[start:cursorPos]
}

// isAlphanumeric checks if a character is alphanumeric
func isAlphanumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
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
func (s *Server) getMemberCompletions(env *types.Environment, objName string, methodsOnly bool) []CompletionItem {
	items := []CompletionItem{}

	typ, ok := env.Get(objName)
	if !ok {
		return items
	}

	switch t := typ.(type) {
	case *types.ClassType:
		// Add properties (only if not method call with :)
		if !methodsOnly {
			for name, propType := range t.Properties {
				items = append(items, CompletionItem{
					Label:  name,
					Detail: types.TypeString(propType),
					Kind:   PropertyCompletion,
				})
			}
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

// getMemberCompletionsFromType returns member completions for a declared type
func (s *Server) getMemberCompletionsFromType(objName string, typ types.Type, methodsOnly bool) []CompletionItem {
	items := []CompletionItem{}

	switch t := typ.(type) {
	case *types.InterfaceType:
		// Add interface properties (only if not method call with :)
		if !methodsOnly {
			for name, propType := range t.Properties {
				isStatic := t.StaticProps != nil && t.StaticProps[name]
				// Skip function types when using dot access on properties
				if _, isFunc := propType.(*types.FunctionType); !isFunc {
					// Only show non-static properties with dot access
					if !isStatic {
						items = append(items, CompletionItem{
							Label:  name,
							Detail: types.TypeString(propType),
							Kind:   PropertyCompletion,
						})
					}
				}
			}
		}
		// Add interface methods (function-type properties and methods)
		for name, propType := range t.Properties {
			if funcType, isFunc := propType.(*types.FunctionType); isFunc {
				isStatic := t.StaticProps != nil && t.StaticProps[name]
				// With : show non-static methods, with . show static methods
				if methodsOnly && !isStatic {
					items = append(items, CompletionItem{
						Label:  name,
						Detail: types.TypeString(funcType),
						Kind:   MethodCompletion,
					})
				} else if !methodsOnly && isStatic {
					items = append(items, CompletionItem{
						Label:  name,
						Detail: types.TypeString(funcType),
						Kind:   MethodCompletion,
					})
				}
			}
		}
		for name, methodType := range t.Methods {
			isStatic := t.StaticMethods != nil && t.StaticMethods[name]
			// With : show non-static methods, with . show static methods
			if methodsOnly && !isStatic {
				items = append(items, CompletionItem{
					Label:  name,
					Detail: types.TypeString(methodType),
					Kind:   MethodCompletion,
				})
			} else if !methodsOnly && isStatic {
				items = append(items, CompletionItem{
					Label:  name,
					Detail: types.TypeString(methodType),
					Kind:   MethodCompletion,
				})
			}
		}
	case *types.ClassType:
		// Add class properties (only if not method call with :)
		if !methodsOnly {
			for name, propType := range t.Properties {
				items = append(items, CompletionItem{
					Label:  name,
					Detail: types.TypeString(propType),
					Kind:   PropertyCompletion,
				})
			}
		}
		// Add class methods
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

func getObjectBeforeDotOrColon(s string) string {
	// Remove the trailing dot or colon
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSuffix(s, ":")
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

// handleSemanticTokens handles textDocument/semanticTokens/full
func (s *Server) handleSemanticTokens(content json.RawMessage, id interface{}) error {
	var request struct {
		Params SemanticTokensParams `json:"params"`
	}

	if err := json.Unmarshal(content, &request); err != nil {
		return s.sendError(id, InvalidParams, err.Error())
	}

	doc := s.documents.Get(request.Params.TextDocument.URI)
	if doc == nil {
		return s.sendResponse(id, SemanticTokens{Data: []uint32{}})
	}

	tokens := s.getSemanticTokens(doc)
	return s.sendResponse(id, tokens)
}

// getSemanticTokens returns semantic tokens for a document
func (s *Server) getSemanticTokens(doc *Document) SemanticTokens {
	tokens := []uint32{}
	
	l := lexer.New(doc.Content)
	
	// Token type indices (must match the legend order in handleInitialize)
	const (
		tokenTypeKeyword = 0
		tokenTypeType = 1
		tokenTypeClass = 2
		tokenTypeInterface = 3
		tokenTypeFunction = 4
		tokenTypeVariable = 5
		tokenTypeParameter = 6
		tokenTypeProperty = 7
		tokenTypeString = 8
		tokenTypeNumber = 9
		tokenTypeComment = 10
		tokenTypeOperator = 11
		tokenTypeModifier = 12
	)
	
	// Token modifier indices
	const (
		modifierDeclaration = 0
		modifierDefinition = 1
		modifierReadonly = 2
	)
	
	prevLine := 0
	prevChar := 0
	
	for {
		tok := l.NextToken()
		if tok.Type == lexer.EOF {
			break
		}
		
		// Get token position (line is 0-indexed in LSP)
		line := tok.Line - 1
		char := tok.Column
		
		var tokenType int = -1
		var tokenModifiers uint32 = 0
		
		// Map token types to semantic token types
		switch tok.Type {
		// Keywords
		case lexer.FUNCTION, lexer.END, lexer.IF, lexer.THEN, lexer.ELSE, lexer.ELSEIF,
			lexer.WHILE, lexer.DO, lexer.FOR, lexer.IN, lexer.RETURN, lexer.BREAK,
			lexer.LOCAL, lexer.CONST, lexer.IMPORT, lexer.EXPORT, lexer.FROM,
			lexer.ASYNC, lexer.AWAIT, lexer.MATCH, lexer.WITH:
			tokenType = tokenTypeKeyword

		// Declaration keywords
		case lexer.DECLARE:
			tokenType = tokenTypeKeyword
			tokenModifiers = 1 << modifierDeclaration

		case lexer.CLASS:
			tokenType = tokenTypeClass

		case lexer.INTERFACE:
			tokenType = tokenTypeInterface

		case lexer.PUBLIC, lexer.PRIVATE, lexer.PROTECTED, lexer.STATIC, lexer.READONLY:
			tokenType = tokenTypeModifier
			if tok.Type == lexer.READONLY {
				tokenModifiers = 1 << modifierReadonly
			}

		// Types
		case lexer.NUMBER_TYPE, lexer.STRING_TYPE, lexer.BOOLEAN,
			lexer.ANY, lexer.VOID, lexer.TABLE, lexer.NEVER, lexer.UNKNOWN:
			tokenType = tokenTypeType

		// Literals
		case lexer.STRING, lexer.TEMPLATE_STRING:
			tokenType = tokenTypeString

		case lexer.NUMBER:
			tokenType = tokenTypeNumber

		case lexer.NIL, lexer.TRUE, lexer.FALSE:
			tokenType = tokenTypeKeyword

		// Operators
		case lexer.PLUS, lexer.MINUS, lexer.ASTERISK, lexer.SLASH, lexer.FLOOR_DIV,
			lexer.EQ, lexer.NOT_EQ, lexer.NOT_EQ_LUA, lexer.LT, lexer.GT, lexer.LT_EQ, lexer.GT_EQ,
			lexer.AND, lexer.OR, lexer.NOT, lexer.CONCAT, lexer.AMPERSAND, lexer.PIPE,
			lexer.CARET, lexer.TILDE, lexer.LEFT_SHIFT, lexer.RIGHT_SHIFT:
			tokenType = tokenTypeOperator
		}
		
		// Skip tokens we don't want to highlight
		if tokenType == -1 {
			continue
		}
		
		// Calculate deltas (LSP semantic tokens use delta encoding)
		deltaLine := line - prevLine
		deltaChar := char
		if deltaLine == 0 {
			deltaChar = char - prevChar
		}
		
		length := len(tok.Literal)
		if length == 0 {
			length = 1
		}
		
		// Append token (format: deltaLine, deltaStart, length, tokenType, tokenModifiers)
		tokens = append(tokens, 
			uint32(deltaLine),
			uint32(deltaChar),
			uint32(length),
			uint32(tokenType),
			tokenModifiers,
		)
		
		prevLine = line
		prevChar = char
	}
	
	return SemanticTokens{Data: tokens}
}
