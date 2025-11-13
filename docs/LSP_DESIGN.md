# Lunar Language Server Protocol (LSP) Design

## Overview

This document outlines the design and implementation plan for the Lunar Language Server, which will provide IDE features like autocomplete, go-to-definition, diagnostics, and more.

## Architecture

### High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Editor (Neovim/VSCode)                  │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │            LSP Client (built-in)                    │    │
│  └────────────────────┬───────────────────────────────┘    │
└───────────────────────┼──────────────────────────────────────┘
                        │ JSON-RPC over stdio/TCP
                        │
┌───────────────────────▼──────────────────────────────────────┐
│                   Lunar Language Server                       │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Protocol Handler (LSP Message Router)               │  │
│  └────────┬─────────────────────────────────────────────┘  │
│           │                                                  │
│  ┌────────▼──────────┬──────────────┬────────────────────┐ │
│  │  Document Manager │  Diagnostics │  Workspace Manager │ │
│  └────────┬──────────┴──────┬───────┴────────────────────┘ │
│           │                  │                               │
│  ┌────────▼──────────────────▼──────────────────────────┐  │
│  │           Lunar Compiler Integration                  │  │
│  │  ┌──────────┬──────────┬──────────┬────────────────┐ │  │
│  │  │  Lexer   │  Parser  │  Types   │  Semantic      │ │  │
│  │  │          │          │ Checker  │  Analysis      │ │  │
│  │  └──────────┴──────────┴──────────┴────────────────┘ │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. **Protocol Handler**
- **Purpose**: Handle LSP JSON-RPC protocol messages
- **Responsibilities**:
  - Parse incoming requests/notifications
  - Route to appropriate handlers
  - Format and send responses
  - Handle initialization/shutdown lifecycle

#### 2. **Document Manager**
- **Purpose**: Track open documents and their state
- **Responsibilities**:
  - Maintain in-memory representation of files
  - Handle `textDocument/didOpen`, `didChange`, `didClose`
  - Manage document versions
  - Provide efficient document lookup

#### 3. **Diagnostics Engine**
- **Purpose**: Provide real-time error/warning reporting
- **Responsibilities**:
  - Run type checker on document changes
  - Convert compiler errors to LSP diagnostics
  - Send `textDocument/publishDiagnostics` notifications
  - Debounce rapid changes

#### 4. **Workspace Manager**
- **Purpose**: Manage project-wide state
- **Responsibilities**:
  - Track workspace folders
  - Manage declaration files (`.d.lunar`)
  - Handle cross-file type information
  - Cache type definitions

## LSP Features Implementation Plan

### Phase 1: Core Features (MVP)

#### 1.1 Diagnostics (Error Checking)
```go
// internal/lsp/diagnostics.go
type DiagnosticsEngine struct {
    documents *DocumentManager
    checker   *types.Checker
}

func (d *DiagnosticsEngine) CheckDocument(uri string) []Diagnostic {
    doc := d.documents.Get(uri)

    // Parse
    lexer := lexer.New(doc.Content)
    parser := parser.New(lexer)
    ast := parser.ParseProgram()

    // Type check
    checker := types.NewChecker()
    checker.Check(ast)

    // Convert errors to LSP diagnostics
    return convertErrors(checker.Errors())
}
```

**LSP Methods**:
- `textDocument/publishDiagnostics` - Send errors/warnings to editor

#### 1.2 Go to Definition
```go
// internal/lsp/definition.go
func (s *Server) FindDefinition(uri string, position Position) Location {
    doc := s.documents.Get(uri)
    symbol := s.findSymbolAtPosition(doc, position)

    if symbol.DefinitionLocation != nil {
        return *symbol.DefinitionLocation
    }

    return Location{}
}
```

**LSP Methods**:
- `textDocument/definition` - Jump to where symbol is defined

#### 1.3 Hover Information
```go
// internal/lsp/hover.go
func (s *Server) GetHover(uri string, position Position) Hover {
    symbol := s.findSymbolAtPosition(uri, position)

    return Hover{
        Contents: MarkupContent{
            Kind:  "markdown",
            Value: formatTypeInfo(symbol),
        },
    }
}
```

**LSP Methods**:
- `textDocument/hover` - Show type information on hover

### Phase 2: Enhanced Features

#### 2.1 Autocompletion
```go
// internal/lsp/completion.go
func (s *Server) GetCompletions(uri string, position Position) []CompletionItem {
    doc := s.documents.Get(uri)
    context := s.getCompletionContext(doc, position)

    suggestions := []CompletionItem{}

    // Add local variables
    suggestions = append(suggestions, s.getLocalVariables(context)...)

    // Add class members
    if context.InClassScope {
        suggestions = append(suggestions, s.getClassMembers(context)...)
    }

    // Add stdlib functions
    suggestions = append(suggestions, s.getStdlibCompletions(context)...)

    return suggestions
}
```

**LSP Methods**:
- `textDocument/completion` - Autocomplete suggestions
- `completionItem/resolve` - Get detailed completion info

#### 2.2 Find References
```go
// internal/lsp/references.go
func (s *Server) FindReferences(uri string, position Position) []Location {
    symbol := s.findSymbolAtPosition(uri, position)
    return s.workspace.FindAllReferences(symbol)
}
```

**LSP Methods**:
- `textDocument/references` - Find all usages of symbol

#### 2.3 Rename Symbol
```go
// internal/lsp/rename.go
func (s *Server) RenameSymbol(uri string, position Position, newName string) WorkspaceEdit {
    symbol := s.findSymbolAtPosition(uri, position)
    locations := s.workspace.FindAllReferences(symbol)

    edits := make([]TextEdit, len(locations))
    for i, loc := range locations {
        edits[i] = TextEdit{
            Range:   loc.Range,
            NewText: newName,
        }
    }

    return WorkspaceEdit{
        Changes: map[string][]TextEdit{
            uri: edits,
        },
    }
}
```

**LSP Methods**:
- `textDocument/rename` - Rename symbol across workspace
- `textDocument/prepareRename` - Validate rename

### Phase 3: Advanced Features

#### 3.1 Code Actions
**LSP Methods**:
- `textDocument/codeAction` - Quick fixes, refactorings
- Examples:
  - "Add missing type annotation"
  - "Import declaration file"
  - "Implement interface"

#### 3.2 Document Symbols
**LSP Methods**:
- `textDocument/documentSymbol` - Outline view
- Show classes, functions, variables in file

#### 3.3 Workspace Symbols
**LSP Methods**:
- `workspace/symbol` - Search symbols across project

#### 3.4 Semantic Tokens
**LSP Methods**:
- `textDocument/semanticTokens/full` - Syntax highlighting
- Better than regex-based highlighting

#### 3.5 Inlay Hints
**LSP Methods**:
- `textDocument/inlayHint` - Show inferred types inline

## File Structure

```
cmd/
  lunar-lsp/          # LSP server binary
    main.go           # Entry point

internal/
  lsp/
    server.go         # Main LSP server
    protocol.go       # LSP message types
    handler.go        # Request/notification handlers

    documents.go      # Document manager
    diagnostics.go    # Diagnostics engine
    workspace.go      # Workspace manager

    completion.go     # Autocompletion
    definition.go     # Go to definition
    hover.go          # Hover information
    references.go     # Find references
    rename.go         # Rename symbol
    symbols.go        # Document/workspace symbols

    semantic.go       # Semantic analysis
    utils.go          # Helper functions
```

## Key Data Structures

```go
// Document represents an open file
type Document struct {
    URI     string
    Version int
    Content string
    AST     *ast.Program
    Types   *types.TypeInfo
}

// Symbol represents an identifier in the code
type Symbol struct {
    Name     string
    Kind     SymbolKind  // Variable, Function, Class, etc.
    Type     types.Type
    Location Location
    Scope    *Scope
}

// Workspace tracks project-wide information
type Workspace struct {
    RootURI     string
    Documents   map[string]*Document
    Symbols     *SymbolTable
    DeclFiles   []*Document  // .d.lunar files
}
```

## Implementation Steps

### Step 1: Project Setup
```bash
mkdir -p cmd/lunar-lsp internal/lsp
```

Create basic server structure with stdio transport.

### Step 2: Basic LSP Lifecycle
Implement:
- `initialize`
- `initialized`
- `shutdown`
- `exit`

### Step 3: Document Management
Implement:
- `textDocument/didOpen`
- `textDocument/didChange`
- `textDocument/didClose`

### Step 4: Diagnostics
Implement:
- Real-time type checking
- `textDocument/publishDiagnostics`

### Step 5: Core Features
Implement in order:
1. Hover (easiest, shows type info)
2. Go to Definition
3. Completion (most complex)

### Step 6: Testing
- Unit tests for each feature
- Integration tests with real editors
- Performance testing

## Performance Considerations

### Caching Strategy
- **AST Cache**: Cache parsed ASTs per document
- **Type Cache**: Cache type check results
- **Symbol Index**: Build symbol index for fast lookups

### Incremental Updates
- Only re-parse/re-check changed documents
- Use incremental text updates from `textDocument/didChange`
- Debounce rapid changes (300ms delay)

### Concurrent Processing
- Process requests concurrently where possible
- Use goroutines for long-running operations
- Maintain document consistency with mutexes

## Error Handling

### Graceful Degradation
- If type checking fails, still provide basic features
- Always respond to LSP requests (never hang)
- Log errors but don't crash

### Recovery
- Recover from panics in handlers
- Continue serving even if one feature breaks

## Configuration

### Server Configuration
```json
{
  "lunar": {
    "diagnostics": {
      "enable": true,
      "debounce": 300
    },
    "completion": {
      "enable": true,
      "triggerCharacters": [".", ":", "("]
    },
    "trace": {
      "server": "verbose"
    }
  }
}
```

## Next Steps

1. Create basic server skeleton
2. Implement initialization
3. Add document management
4. Implement diagnostics
5. Add hover/definition
6. Test with Neovim
7. Expand features incrementally

## References

- [LSP Specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)
- [Go LSP Library](https://github.com/TobiasYin/go-lsp)
- [Neovim LSP Client](https://neovim.io/doc/user/lsp.html)
