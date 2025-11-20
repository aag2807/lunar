# Lunar LSP Features Roadmap

This document tracks the implementation status of Language Server Protocol features in the Lunar LSP.

---

## Current Implementation Status

### ✅ Fully Implemented Features

| Feature | LSP Method | Status | Description |
|---------|-----------|--------|-------------|
| **Hover** | `textDocument/hover` | ✅ Complete | Shows type information on hover |
| **Go to Definition** | `textDocument/definition` | ✅ Complete | Jump to symbol definition |
| **Find References** | `textDocument/references` | ✅ Complete | Find all symbol usages |
| **Completion** | `textDocument/completion` | ✅ Complete | Auto-completion with context |
| **Rename Symbol** | `textDocument/rename` | ✅ Complete | Rename symbol across document |
| **Diagnostics** | `textDocument/publishDiagnostics` | ✅ Complete | Parse and type errors |
| **Document Sync** | `textDocument/did{Open,Change,Close,Save}` | ✅ Complete | Document lifecycle management |

### 🚧 Protocol Definitions Added (Ready for Implementation)

| Feature | LSP Method | Protocol Status | Handler Status | Description |
|---------|-----------|----------------|----------------|-------------|
| **Code Actions** | `textDocument/codeAction` | ✅ Added | ⏳ Pending | Quick fixes, refactorings |
| **Inlay Hints** | `textDocument/inlayHint` | ✅ Added | ⏳ Pending | Inline type hints |
| **Semantic Tokens** | `textDocument/semanticTokens/full` | ✅ Added | ⏳ Pending | Semantic highlighting |

### 📋 Future Enhancements

| Feature | LSP Method | Priority | Description |
|---------|-----------|----------|-------------|
| **Document Symbols** | `textDocument/documentSymbol` | High | Symbol outline |
| **Workspace Symbols** | `workspace/symbol` | High | Cross-file symbol search |
| **Signature Help** | `textDocument/signatureHelp` | Medium | Function parameter hints |
| **Call Hierarchy** | `textDocument/prepareCallHierarchy` | Medium | Call graph navigation |
| **Type Hierarchy** | `textDocument/prepareTypeHierarchy` | Low | Type inheritance tree |

---

## Protocol Definitions Added

### 1. Code Actions (protocol.go:355-386)

**Purpose:** Provide quick fixes, refactorings, and source actions.

**Types Added:**
```go
type CodeActionOptions struct {
    CodeActionKinds []string
}

type CodeActionParams struct {
    TextDocument TextDocumentIdentifier
    Range        Range
    Context      CodeActionContext
}

type CodeAction struct {
    Title       string
    Kind        string
    Diagnostics []Diagnostic
    Edit        *WorkspaceEdit
    Command     *Command
}
```

**Supported Action Kinds:**
- `quickfix` - Fix errors/warnings
- `refactor.extract` - Extract to function/variable
- `source.organizeImports` - Auto-organize imports

**Example Use Cases:**
1. **Add Missing Type Annotation**
   ```lunar
   local x = 10  // Suggest: Add explicit type
   → local x: number = 10
   ```

2. **Auto-Import Missing Symbol**
   ```lunar
   local user: User = {}  // User not imported
   → import { User } from "./types"
   ```

3. **Extract to Function**
   ```lunar
   // Selected code
   local result = a + b * 2
   → function calculate(a, b) return a + b * 2 end
   ```

---

### 2. Inlay Hints (protocol.go:388-413)

**Purpose:** Show inferred types and parameter names inline in the editor.

**Types Added:**
```go
type InlayHintOptions struct {
    ResolveProvider bool
}

type InlayHintParams struct {
    TextDocument TextDocumentIdentifier
    Range        Range
}

type InlayHint struct {
    Position     Position
    Label        string
    Kind         int  // 1=Type, 2=Parameter
    PaddingLeft  bool
    PaddingRight bool
    Tooltip      *MarkupContent
}
```

**Hint Kinds:**
- **Type Hints** (`Kind = 1`): Show inferred types
- **Parameter Hints** (`Kind = 2`): Show parameter names in function calls

**Example Use Cases:**

1. **Inferred Type Hints**
   ```lunar
   local result = compute(10, 20)
   // Editor shows:
   local result: number = compute(10, 20)
            ^^^^^^^^ (inlay hint)
   ```

2. **Parameter Name Hints**
   ```lunar
   createUser("Alice", 30, true)
   // Editor shows:
   createUser(name: "Alice", age: 30, active: true)
              ^^^^^          ^^^^       ^^^^^^^
   ```

3. **Generic Type Arguments**
   ```lunar
   local list = List.new()
   // Editor shows:
   local list = List<any>.new()
                     ^^^^^ (inlay hint)
   ```

---

### 3. Semantic Tokens (protocol.go:415-486)

**Purpose:** Provide rich syntax highlighting based on semantic meaning.

**Types Added:**
```go
type SemanticTokensOptions struct {
    Legend SemanticTokensLegend
    Range  bool
    Full   bool
}

type SemanticTokensLegend struct {
    TokenTypes     []string
    TokenModifiers []string
}

type SemanticTokens struct {
    Data []uint32  // Encoded token data
}
```

**Token Types (20 types):**
- `namespace`, `type`, `class`, `enum`, `interface`
- `parameter`, `variable`, `property`
- `function`, `method`
- `keyword`, `modifier`, `comment`
- `string`, `number`, `regexp`, `operator`

**Token Modifiers (10 modifiers):**
- `declaration`, `definition`
- `readonly`, `static`, `deprecated`, `abstract`, `async`
- `modification`, `documentation`, `defaultLibrary`

**Example Use Cases:**

1. **Distinguish Types from Values**
   ```lunar
   local User = User.new()
   //    ^^^^   ^^^^
   // variable  class (different colors)
   ```

2. **Highlight Unused Variables**
   ```lunar
   local unused = 10
   //    ^^^^^^ (grayed out + modification=unused)
   ```

3. **Readonly vs Mutable**
   ```lunar
   const PI = 3.14      // readonly modifier
   local x = 10         // no modifier (mutable)
   ```

---

## Implementation Architecture

### Current LSP Infrastructure

**Files:**
- `cmd/lunar-lsp/main.go` - LSP server entry point
- `internal/lsp/server.go` - Core server (message handling)
- `internal/lsp/handler.go` - Request handlers
- `internal/lsp/protocol.go` - LSP protocol types ✅ **Updated**
- `internal/lsp/documents.go` - Document management
- `internal/lsp/diagnostics.go` - Error reporting

**Type Checker Integration:**
- `internal/types/check.go` - Symbol resolution and type checking
- Type environment with scope chain
- Symbol tables for classes, interfaces, enums, functions

**AST with Position Tracking:**
- Every token has `Line` and `Column` (1-based)
- Full position information for all nodes

---

## Implementation Plan

### Phase 1: Code Actions 🎯

**Step 1: Add Handler Registration** (server.go)
```go
case "textDocument/codeAction":
    return s.handleCodeAction(content)
```

**Step 2: Implement Handler** (handler.go)
```go
func (s *Server) handleCodeAction(content json.RawMessage) error {
    // 1. Parse CodeActionParams
    // 2. Get document and diagnostics
    // 3. Generate applicable code actions:
    //    - Add missing types
    //    - Add missing imports
    //    - Extract to function
    // 4. Return []CodeAction
}
```

**Step 3: Update Initialize Response**
```go
CodeActionProvider: &CodeActionOptions{
    CodeActionKinds: []string{
        CodeActionKindQuickFix,
        CodeActionKindRefactorExtract,
        CodeActionKindSourceOrganizeImports,
    },
}
```

**Implementation Strategy:**
1. **Quick Fix: Add Missing Type**
   - Detect variables without type annotations
   - Infer type from initializer using type checker
   - Generate edit to add`: Type` annotation

2. **Quick Fix: Add Missing Import**
   - Detect undefined symbols
   - Search workspace for symbol definitions
   - Generate import statement

3. **Refactor: Extract to Function**
   - Take selected code range
   - Analyze variables used (read/write)
   - Generate function with parameters and return
   - Replace selection with function call

---

### Phase 2: Inlay Hints 🎯

**Step 1: Add Handler Registration** (server.go)
```go
case "textDocument/inlayHint":
    return s.handleInlayHint(content)
```

**Step 2: Implement Handler** (handler.go)
```go
func (s *Server) handleInlayHint(content json.RawMessage) error {
    // 1. Parse InlayHintParams
    // 2. Get document and AST
    // 3. Traverse AST in range:
    //    - Variable declarations → type hints
    //    - Function calls → parameter hints
    // 4. Return []InlayHint
}
```

**Step 3: Update Initialize Response**
```go
InlayHintProvider: &InlayHintOptions{
    ResolveProvider: false,
}
```

**Implementation Strategy:**
1. **Type Hints**
   - Find all `VariableDeclaration` without explicit type
   - Use type checker to infer type
   - Create hint at end of identifier with `: Type`

2. **Parameter Hints**
   - Find all `CallExpression` nodes
   - Get function signature from type checker
   - Match arguments to parameters
   - Create hints before each argument with `paramName:`

---

### Phase 3: Semantic Tokens 🎯

**Step 1: Add Handler Registration** (server.go)
```go
case "textDocument/semanticTokens/full":
    return s.handleSemanticTokens(content)
```

**Step 2: Implement Handler** (handler.go)
```go
func (s *Server) handleSemanticTokens(content json.RawMessage) error {
    // 1. Parse SemanticTokensParams
    // 2. Get document and AST
    // 3. Traverse AST and encode tokens:
    //    - Classes → class type
    //    - Functions → function/method type
    //    - Variables → variable type + modifiers
    // 4. Return SemanticTokens with encoded data
}
```

**Step 3: Update Initialize Response**
```go
SemanticTokensProvider: &SemanticTokensOptions{
    Legend: SemanticTokensLegend{
        TokenTypes: []string{
            "class", "interface", "enum", "function",
            "variable", "parameter", "property", "type",
            "keyword", "comment", "string", "number",
        },
        TokenModifiers: []string{
            "declaration", "readonly", "static",
            "deprecated", "abstract", "async",
        },
    },
    Full: true,
}
```

**Token Encoding Format:**
Semantic tokens are encoded as an array of uint32 values:
```
[deltaLine, deltaStart, length, tokenType, tokenModifiers]
```

Each token is 5 uint32 values (delta-encoded for compactness).

**Implementation Strategy:**
1. **Token Collection**
   - Traverse AST depth-first
   - Classify each identifier:
     - Check if it's a type name (class, interface, enum)
     - Check if it's a function/method
     - Check if it's a variable/parameter/property
     - Get modifiers from AST/type checker

2. **Token Encoding**
   - Sort tokens by line, then column
   - Delta-encode positions (relative to previous token)
   - Encode token type as index into legend
   - Encode modifiers as bit flags

3. **Modifier Detection**
   - `readonly`: Check for `const` keyword or `readonly` property
   - `static`: Check class member AST node
   - `deprecated`: Check for `@deprecated` JSDoc comment
   - `abstract`: Check class/method modifiers

---

## Type Checker Integration Points

### Symbol Resolution

The type checker maintains comprehensive symbol tables:

```go
checker.GetEnv().Get(name)          // Lookup symbol in scope
checker.GetEnv().GetAll()           // Get all symbols in scope
checker.classes[name]               // Lookup class definition
checker.interfaces[name]            // Lookup interface definition
checker.enums[name]                 // Lookup enum definition
checker.typeAliases[name]           // Lookup type alias
```

### Type Information

```go
checker.checkExpression(expr)       // Get type of expression
checker.getTypeString(typ)          // Convert type to string
```

### AST Traversal

All AST nodes have position information:

```go
node.Token.Line      // 1-based line number
node.Token.Column    // 1-based column number
```

---

## Testing Strategy

### Unit Tests

Create test files for each handler:

```go
// internal/lsp/codeaction_test.go
func TestCodeActionAddType(t *testing.T) {
    // Test adding type annotation
}

func TestCodeActionExtractFunction(t *testing.T) {
    // Test extract to function refactoring
}
```

### Integration Tests

Test with actual LSP clients:

1. **VS Code Extension**
   - Test all features in VS Code
   - Verify UI integration

2. **Neovim LSP Client**
   - Test with native LSP
   - Verify protocol compatibility

3. **LSP Test Suite**
   - Use official LSP test framework
   - Verify spec compliance

---

## Documentation

### User Documentation

Create guides for:
1. **Setting up Lunar LSP** in various editors
2. **Using Code Actions** - shortcuts and available actions
3. **Configuring Inlay Hints** - when to show/hide
4. **Understanding Semantic Highlighting** - token types and colors

### Developer Documentation

1. **Architecture Overview** - how LSP server works
2. **Adding New Features** - extending the LSP
3. **Type Checker Integration** - using symbol tables
4. **Protocol Reference** - custom extensions

---

## Performance Considerations

### 1. Incremental Updates

- Use incremental document sync for large files
- Cache AST and type checker results per document
- Invalidate cache only on document changes

### 2. Async Processing

- Process diagnostics asynchronously
- Don't block on expensive operations
- Use timeouts for long-running analyses

### 3. Workspace Indexing

- Build symbol index for all files in workspace
- Update incrementally on file changes
- Use for fast "Find All References" across files

---

## Configuration Options

Future LSP configuration (via `workspace/configuration`):

```json
{
  "lunar.inlayHints.parameterNames": true,
  "lunar.inlayHints.inferredTypes": true,
  "lunar.codeActions.autoImport": true,
  "lunar.semanticHighlighting": true,
  "lunar.diagnostics.typeCheck": true
}
```

---

## Summary

**Current Status:**
- ✅ Protocol types added for all three features
- ✅ LSP server compiles successfully
- ⏳ Handler implementations pending

**Next Steps:**
1. Implement Code Actions handler
2. Implement Inlay Hints handler
3. Implement Semantic Tokens handler
4. Add comprehensive tests
5. Update LSP initialization to advertise new capabilities
6. Create user and developer documentation

**Estimated Effort:**
- Code Actions: 2-3 hours (basic quick fixes)
- Inlay Hints: 1-2 hours (type and parameter hints)
- Semantic Tokens: 2-3 hours (token encoding)
- Testing: 2-3 hours
- Documentation: 1-2 hours

**Total: ~10-15 hours of implementation work**

---

**Version**: 1.0
**Last Updated**: 2025-01-20
**Status**: Protocol Definitions Complete, Handlers Pending
