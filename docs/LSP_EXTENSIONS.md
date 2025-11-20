# Language Server Protocol Extensions

Lunar extends the standard Language Server Protocol with custom methods for enhanced code intelligence, project analysis, and developer productivity.

## Table of Contents

- [Overview](#overview)
- [Custom Methods](#custom-methods)
- [Type System Extensions](#type-system-extensions)
- [Project Analysis](#project-analysis)
- [Smart Completion](#smart-completion)
- [Code Actions](#code-actions)
- [Editor Integration](#editor-integration)

## Overview

Lunar's LSP extensions provide:

- **Advanced type checking** - Deep type analysis and inference
- **Project-wide analysis** - Dependency graphs, metrics, issue detection
- **Smart completion** - Context-aware, type-driven suggestions
- **Refactoring** - Extract function/variable, inline, rename
- **Migration tools** - Lua to Lunar conversion
- **JIT hints** - Performance optimization assistance

## Custom Methods

All custom methods use the `lunar/` prefix to avoid conflicts with standard LSP methods.

### Type System

- `lunar/typeCheck` - Full type checking with detailed diagnostics
- `lunar/inferType` - Infer type at cursor position
- `lunar/typeHierarchy` - Show type inheritance hierarchy
- `lunar/expandType` - Expand complex types to show full definition

### Project Analysis

- `lunar/projectAnalysis` - Full project analysis
- `lunar/dependencyGraph` - Module dependency visualization
- `lunar/importAnalysis` - Import statement analysis
- `lunar/moduleGraph` - Module relationship mapping

### Code Intelligence

- `lunar/smartCompletion` - Context-aware completion
- `lunar/callHierarchy` - Function call graph
- `lunar/usageAnalysis` - Symbol usage analysis
- `lunar/deadCode` - Dead code detection

### Refactoring

- `lunar/extractFunction` - Extract code to function
- `lunar/extractVariable` - Extract expression to variable
- `lunar/inlineFunction` - Inline function calls
- `lunar/renameSymbol` - Smart symbol renaming

### Migration & Optimization

- `lunar/migrateLua` - Convert Lua to Lunar
- `lunar/addTypes` - Add type annotations
- `lunar/optimize` - Code optimization suggestions
- `lunar/generateJIT` - Generate JIT hints

## Type System Extensions

### Full Type Checking

**Method:** `lunar/typeCheck`

Performs comprehensive type checking with detailed diagnostics and metrics.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "lunar/typeCheck",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lunar"
    },
    "options": {
      "strict": true,
      "noImplicitAny": true,
      "strictNullCheck": true,
      "unusedVars": true
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "diagnostics": [
      {
        "range": { "start": { "line": 10, "character": 5 }, "end": { "line": 10, "character": 15 } },
        "severity": 1,
        "message": "Type 'string' is not assignable to type 'number'"
      }
    ],
    "typeInfo": [
      {
        "range": { "start": { "line": 5, "character": 10 }, "end": { "line": 5, "character": 14 } },
        "symbol": "user",
        "type": "User",
        "inferred": false,
        "complexity": 3
      }
    ],
    "metrics": {
      "totalSymbols": 45,
      "typedSymbols": 42,
      "inferredSymbols": 20,
      "anyTypes": 3,
      "coverage": 93.3
    }
  }
}
```

### Type Inference

**Method:** `lunar/inferType`

Infers the type of an expression at the cursor position.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "lunar/inferType",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lunar"
    },
    "position": {
      "line": 15,
      "character": 20
    },
    "deep": true
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "type": "array<User>",
    "confidence": 0.95,
    "alternatives": [
      {
        "type": "table<number, User>",
        "confidence": 0.75,
        "reason": "Could be a table if keys are numeric"
      }
    ],
    "reasoning": "Inferred from assignment and usage patterns"
  }
}
```

### Type Hierarchy

**Method:** `lunar/typeHierarchy`

Shows the inheritance hierarchy for a type.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "lunar/typeHierarchy",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lunar"
    },
    "position": {
      "line": 8,
      "character": 12
    },
    "direction": "both"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "root": {
      "name": "User",
      "kind": "interface",
      "location": {
        "uri": "file:///path/to/file.lunar",
        "range": { "start": { "line": 5, "character": 0 }, "end": { "line": 10, "character": 3 } }
      },
      "members": [
        { "name": "id", "kind": "property" },
        { "name": "name", "kind": "property" }
      ],
      "extends": [],
      "implements": []
    },
    "parents": [],
    "children": [
      {
        "name": "AdminUser",
        "kind": "interface",
        "extends": ["User"]
      }
    ]
  }
}
```

## Project Analysis

### Full Project Analysis

**Method:** `lunar/projectAnalysis`

Analyzes the entire project for metrics, dependencies, and issues.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "lunar/projectAnalysis",
  "params": {
    "rootUri": "file:///path/to/project",
    "options": {
      "includeDependencies": true,
      "includeMetrics": true,
      "includeIssues": true,
      "includeComplexity": true,
      "maxDepth": 10
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "files": [
      {
        "uri": "file:///path/to/project/main.lunar",
        "symbols": 25,
        "functions": 8,
        "interfaces": 3,
        "classes": 2,
        "loc": 342,
        "complexity": 15,
        "dependencies": ["./utils.lunar", "./models.lunar"],
        "exports": ["main", "init"]
      }
    ],
    "dependencies": [
      {
        "source": "file:///path/to/project/main.lunar",
        "target": "file:///path/to/project/utils.lunar",
        "type": "import",
        "symbols": ["helper", "format"],
        "location": {
          "uri": "file:///path/to/project/main.lunar",
          "range": { "start": { "line": 1, "character": 0 }, "end": { "line": 1, "character": 35 } }
        }
      }
    ],
    "metrics": {
      "totalFiles": 12,
      "totalLOC": 3456,
      "totalFunctions": 89,
      "totalInterfaces": 15,
      "totalClasses": 8,
      "averageComplexity": 8.5,
      "maxComplexity": 25,
      "typeCoverage": 92.5,
      "dependencyDepth": 5,
      "circularDeps": 0
    },
    "issues": [
      {
        "severity": "warning",
        "category": "complexity",
        "message": "Function 'processData' has high complexity (25)",
        "locations": [
          {
            "uri": "file:///path/to/project/data.lunar",
            "range": { "start": { "line": 45, "character": 0 }, "end": { "line": 85, "character": 3 } }
          }
        ],
        "suggestion": "Consider refactoring to reduce complexity",
        "autoFixable": false
      }
    ],
    "summary": {
      "status": "warning",
      "duration": 1250,
      "filesScanned": 12,
      "errorCount": 0,
      "warningCount": 3,
      "infoCount": 7
    }
  }
}
```

### Dependency Graph

**Method:** `lunar/dependencyGraph`

Builds a visual dependency graph for the project.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "lunar/dependencyGraph",
  "params": {
    "rootUri": "file:///path/to/project",
    "maxDepth": 5
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "nodes": [
      {
        "id": "main.lunar",
        "uri": "file:///path/to/project/main.lunar",
        "type": "module",
        "exports": ["main", "init"],
        "imports": 3,
        "importedBy": 0
      },
      {
        "id": "utils.lunar",
        "uri": "file:///path/to/project/utils.lunar",
        "type": "module",
        "exports": ["helper", "format"],
        "imports": 1,
        "importedBy": 5
      }
    ],
    "edges": [
      {
        "from": "main.lunar",
        "to": "utils.lunar",
        "symbols": ["helper", "format"],
        "weight": 10
      }
    ],
    "circular": [],
    "entryPoints": ["main.lunar"],
    "orphans": []
  }
}
```

## Smart Completion

**Method:** `lunar/smartCompletion`

Provides context-aware, type-driven completion suggestions.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "lunar/smartCompletion",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lunar"
    },
    "position": {
      "line": 20,
      "character": 15
    },
    "context": {
      "triggerKind": 2,
      "triggerCharacter": ".",
      "expectedType": "string",
      "inFunction": true,
      "inClass": false,
      "inInterface": false
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "result": {
    "items": [
      {
        "label": "toUpperCase",
        "kind": 2,
        "detail": "function(): string",
        "documentation": "Converts string to uppercase",
        "insertText": "toUpperCase()",
        "relevance": 0.95,
        "typeMatch": true,
        "imports": [],
        "snippet": "toUpperCase()",
        "examples": ["const upper = text.toUpperCase()"]
      },
      {
        "label": "trim",
        "kind": 2,
        "detail": "function(): string",
        "documentation": "Removes whitespace from both ends",
        "insertText": "trim()",
        "relevance": 0.85,
        "typeMatch": true,
        "imports": [],
        "snippet": "trim()"
      }
    ],
    "suggestions": [
      {
        "title": "Convert to const",
        "description": "This variable is never reassigned",
        "edit": {
          "changes": {
            "file:///path/to/file.lunar": [
              {
                "range": { "start": { "line": 15, "character": 0 }, "end": { "line": 15, "character": 5 } },
                "newText": "const"
              }
            ]
          }
        },
        "priority": 5
      }
    ]
  }
}
```

## Code Actions

### Extract Function

**Method:** `lunar/extractFunction`

Extracts selected code into a new function.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "lunar/extractFunction",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lunar"
    },
    "range": {
      "start": { "line": 10, "character": 4 },
      "end": { "line": 15, "character": 5 }
    },
    "functionName": "calculateTotal",
    "options": {
      "generateTypes": true,
      "inlineVars": false
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "edit": {
      "changes": {
        "file:///path/to/file.lunar": [
          {
            "range": { "start": { "line": 8, "character": 0 }, "end": { "line": 8, "character": 0 } },
            "newText": "function calculateTotal(items: array<number>): number\n    local sum: number = 0\n    for _, item in ipairs(items) do\n        sum = sum + item\n    end\n    return sum\nend\n\n"
          },
          {
            "range": { "start": { "line": 10, "character": 4 }, "end": { "line": 15, "character": 5 } },
            "newText": "const total = calculateTotal(items)"
          }
        ]
      }
    },
    "function": {
      "name": "calculateTotal",
      "signature": "function calculateTotal(items: array<number>): number",
      "body": "...",
      "location": {
        "uri": "file:///path/to/file.lunar",
        "range": { "start": { "line": 8, "character": 0 }, "end": { "line": 13, "character": 3 } }
      }
    },
    "parameters": [
      {
        "name": "items",
        "type": "array<number>"
      }
    ],
    "returnType": "number"
  }
}
```

### Migrate Lua to Lunar

**Method:** `lunar/migrateLua`

Converts Lua code to Lunar with type annotations.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "method": "lunar/migrateLua",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.lua"
    },
    "options": {
      "addTypes": true,
      "useConst": true,
      "generateInterfaces": true,
      "modernize": true,
      "strictMode": false
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 8,
  "result": {
    "code": "const name: string = \"Alice\"\n\nfunction greet(person: string): string\n    return \"Hello, \" .. person\nend",
    "changes": [
      {
        "range": { "start": { "line": 0, "character": 0 }, "end": { "line": 0, "character": 5 } },
        "oldText": "local",
        "newText": "const",
        "description": "Changed to const (immutable)"
      },
      {
        "range": { "start": { "line": 0, "character": 11 }, "end": { "line": 0, "character": 11 } },
        "oldText": "",
        "newText": ": string",
        "description": "Added type annotation"
      }
    ],
    "suggestions": [
      "Consider creating a User interface for better type safety"
    ],
    "stats": {
      "typesAdded": 5,
      "functionsUpdated": 3,
      "constUsed": 8,
      "interfacesAdded": 0
    }
  }
}
```

## Editor Integration

### VS Code Extension

```json
// settings.json
{
  "lunar.lsp.extensions.enabled": true,
  "lunar.lsp.typeCheck.strict": true,
  "lunar.lsp.projectAnalysis.enabled": true,
  "lunar.lsp.smartCompletion.enabled": true,
  "lunar.lsp.autoMigrate": false
}
```

### Neovim Configuration

```lua
-- init.lua
require('lspconfig').lunar_lsp.setup{
  settings = {
    lunar = {
      lsp = {
        extensions = {
          enabled = true,
          typeCheck = {
            strict = true,
            noImplicitAny = true
          },
          projectAnalysis = {
            enabled = true,
            onSave = true
          },
          smartCompletion = {
            enabled = true,
            typeMatch = true
          }
        }
      }
    }
  },
  on_attach = function(client, bufnr)
    -- Custom commands
    vim.api.nvim_buf_create_user_command(bufnr, 'LunarTypeCheck', function()
      vim.lsp.buf.execute_command({
        command = 'lunar.typeCheck',
        arguments = { vim.api.nvim_buf_get_name(bufnr) }
      })
    end, {})

    vim.api.nvim_buf_create_user_command(bufnr, 'LunarAnalyze', function()
      vim.lsp.buf.execute_command({
        command = 'lunar.projectAnalysis',
        arguments = { vim.fn.getcwd() }
      })
    end, {})
  end
}
```

### Emacs Configuration

```elisp
;; lunar-mode.el
(use-package lsp-mode
  :hook (lunar-mode . lsp)
  :config
  (setq lsp-lunar-extensions-enabled t)
  (setq lsp-lunar-type-check-strict t)
  (setq lsp-lunar-smart-completion-enabled t)

  ;; Custom commands
  (defun lunar-type-check ()
    (interactive)
    (lsp-request "lunar/typeCheck"
                 (list :textDocument (lsp--text-document-identifier))))

  (defun lunar-project-analysis ()
    (interactive)
    (lsp-request "lunar/projectAnalysis"
                 (list :rootUri (lsp-workspace-root))))

  (define-key lunar-mode-map (kbd "C-c t") 'lunar-type-check)
  (define-key lunar-mode-map (kbd "C-c a") 'lunar-project-analysis))
```

## Performance Considerations

LSP extensions are designed for performance:

- **Incremental analysis** - Only reanalyze changed files
- **Caching** - Results cached until files change
- **Background processing** - Long operations don't block editor
- **Streaming results** - Large results sent incrementally

## Summary

Lunar's LSP extensions provide:

- ✅ 20+ custom methods for advanced features
- ✅ Deep type system integration
- ✅ Project-wide analysis and visualization
- ✅ Smart, context-aware completion
- ✅ Powerful refactoring tools
- ✅ Seamless editor integration

Enable in your editor to unlock the full power of Lunar! 🚀
