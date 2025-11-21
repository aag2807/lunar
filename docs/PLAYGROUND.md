# Lunar Playground / Online IDE

This document provides comprehensive instructions for building a web-based Lunar compiler and IDE.

## Architecture Overview

There are three main approaches for implementing the Lunar playground:

### Option 1: Server-Side Compilation (Recommended for MVP)
**Pros:** Simple to implement, full compiler features, easier to maintain
**Cons:** Requires server resources, network latency

### Option 2: WebAssembly Compilation (Best User Experience)
**Pros:** Client-side execution, no network latency, offline capable
**Cons:** More complex build process, larger initial download

### Option 3: Hybrid Approach
**Pros:** Best of both worlds, fallback options
**Cons:** Most complex to implement

## Option 1: Server-Side Compilation

### Backend API Design

Create a REST API with the following endpoints:

#### 1. Compile Endpoint
```http
POST /api/compile
Content-Type: application/json

{
  "code": "local x: number = 42\nprint(x)",
  "options": {
    "typecheck": true,
    "optimize": false,
    "minify": false,
    "target": "lua51"
  }
}

Response:
{
  "success": true,
  "output": "local x = 42\nprint(x)",
  "errors": [],
  "warnings": [],
  "sourceMap": "...",
  "stats": {
    "compileTime": 23,
    "lines": 2,
    "size": 28
  }
}
```

#### 2. Type Check Endpoint
```http
POST /api/typecheck
Content-Type: application/json

{
  "code": "local x: number = 'wrong'"
}

Response:
{
  "success": false,
  "errors": [
    {
      "line": 1,
      "column": 19,
      "message": "Cannot assign type 'string' to variable 'x' of type 'number'",
      "severity": "error"
    }
  ]
}
```

#### 3. Format Endpoint
```http
POST /api/format
Content-Type: application/json

{
  "code": "local x:number=42;print(x)"
}

Response:
{
  "success": true,
  "formatted": "local x: number = 42\nprint(x)"
}
```

#### 4. Share Snippet Endpoint
```http
POST /api/snippets
Content-Type: application/json

{
  "code": "local x: number = 42\nprint(x)",
  "title": "Hello World Example",
  "description": "Basic number example"
}

Response:
{
  "id": "abc123xyz",
  "url": "https://playground.lunar-lang.dev/share/abc123xyz",
  "shortUrl": "lunar.dev/s/abc123"
}
```

#### 5. Get Snippet Endpoint
```http
GET /api/snippets/:id

Response:
{
  "id": "abc123xyz",
  "code": "local x: number = 42\nprint(x)",
  "title": "Hello World Example",
  "description": "Basic number example",
  "createdAt": "2024-01-15T10:30:00Z",
  "views": 123
}
```

### Backend Implementation (Go)

```go
package main

import (
    "encoding/json"
    "lunar/internal/compiler"
    "net/http"
    "github.com/gin-gonic/gin"
)

type CompileRequest struct {
    Code    string          `json:"code"`
    Options CompileOptions  `json:"options"`
}

type CompileOptions struct {
    TypeCheck bool   `json:"typecheck"`
    Optimize  bool   `json:"optimize"`
    Minify    bool   `json:"minify"`
    Target    string `json:"target"`
}

type CompileResponse struct {
    Success   bool              `json:"success"`
    Output    string            `json:"output,omitempty"`
    Errors    []CompilerError   `json:"errors"`
    Warnings  []CompilerWarning `json:"warnings"`
    SourceMap string            `json:"sourceMap,omitempty"`
    Stats     CompileStats      `json:"stats"`
}

func handleCompile(c *gin.Context) {
    var req CompileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Compile the code
    result, err := compiler.Compile(req.Code, req.Options)

    response := CompileResponse{
        Success:  err == nil,
        Output:   result.Output,
        Errors:   result.Errors,
        Warnings: result.Warnings,
        Stats:    result.Stats,
    }

    c.JSON(200, response)
}

func main() {
    r := gin.Default()

    // Enable CORS
    r.Use(corsMiddleware())

    // API routes
    api := r.Group("/api")
    {
        api.POST("/compile", handleCompile)
        api.POST("/typecheck", handleTypeCheck)
        api.POST("/format", handleFormat)
        api.POST("/snippets", handleCreateSnippet)
        api.GET("/snippets/:id", handleGetSnippet)
    }

    r.Run(":8080")
}
```

### Frontend Implementation

#### React + Monaco Editor Example

```typescript
// src/components/Playground.tsx
import React, { useState, useCallback } from 'react';
import Editor from '@monaco-editor/react';
import { debounce } from 'lodash';

interface CompileResult {
  success: boolean;
  output?: string;
  errors?: CompilerError[];
  warnings?: CompilerWarning[];
}

const Playground: React.FC = () => {
  const [code, setCode] = useState('local x: number = 42\nprint(x)');
  const [output, setOutput] = useState('');
  const [errors, setErrors] = useState<CompilerError[]>([]);
  const [isCompiling, setIsCompiling] = useState(false);

  const compile = useCallback(async (sourceCode: string) => {
    setIsCompiling(true);
    try {
      const response = await fetch('/api/compile', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code: sourceCode,
          options: {
            typecheck: true,
            optimize: false,
            minify: false,
            target: 'lua51'
          }
        })
      });

      const result: CompileResult = await response.json();

      if (result.success) {
        setOutput(result.output || '');
        setErrors([]);
      } else {
        setErrors(result.errors || []);
      }
    } catch (error) {
      console.error('Compilation error:', error);
    } finally {
      setIsCompiling(false);
    }
  }, []);

  // Debounce compilation
  const debouncedCompile = useCallback(
    debounce((code: string) => compile(code), 500),
    [compile]
  );

  const handleEditorChange = (value: string | undefined) => {
    const newCode = value || '';
    setCode(newCode);
    debouncedCompile(newCode);
  };

  return (
    <div className="playground">
      <div className="editor-panel">
        <h3>Lunar Code</h3>
        <Editor
          height="600px"
          language="lua"
          theme="vs-dark"
          value={code}
          onChange={handleEditorChange}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            scrollBeyondLastLine: false,
            automaticLayout: true,
          }}
        />
        {errors.length > 0 && (
          <div className="errors">
            {errors.map((error, i) => (
              <div key={i} className="error">
                Line {error.line}: {error.message}
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="output-panel">
        <h3>Compiled Lua</h3>
        <Editor
          height="600px"
          language="lua"
          theme="vs-dark"
          value={output}
          options={{
            readOnly: true,
            minimap: { enabled: false },
            fontSize: 14,
          }}
        />
      </div>
    </div>
  );
};

export default Playground;
```

#### Monaco Editor Language Configuration

```typescript
// src/monaco/lunar.ts
import * as monaco from 'monaco-editor';

export function registerLunarLanguage() {
  monaco.languages.register({ id: 'lunar' });

  monaco.languages.setMonarchTokensProvider('lunar', {
    keywords: [
      'local', 'function', 'end', 'if', 'then', 'else', 'elseif',
      'for', 'while', 'do', 'return', 'break', 'continue',
      'class', 'interface', 'type', 'const', 'declare',
      'public', 'private', 'protected', 'static', 'readonly',
      'async', 'await', 'extends', 'implements'
    ],

    typeKeywords: [
      'number', 'string', 'boolean', 'any', 'void', 'nil',
      'table', 'array', 'union', 'intersection'
    ],

    operators: [
      '=', '>', '<', '!', '~', '?', ':', '==', '<=', '>=', '!=',
      '&&', '||', '++', '--', '+', '-', '*', '/', '&', '|', '^', '%',
      '<<', '>>', '>>>', '+=', '-=', '*=', '/=', '&=', '|=', '^=',
      '%=', '<<=', '>>=', '>>>='
    ],

    symbols: /[=><!~?:&|+\-*\/\^%]+/,

    escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{1,4}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,

    tokenizer: {
      root: [
        // Type annotations
        [/:\s*[a-zA-Z_]\w*/, 'type'],

        // Identifiers and keywords
        [/[a-z_$][\w$]*/, {
          cases: {
            '@typeKeywords': 'type.identifier',
            '@keywords': 'keyword',
            '@default': 'identifier'
          }
        }],

        // Whitespace
        { include: '@whitespace' },

        // Strings
        [/"([^"\\]|\\.)*$/, 'string.invalid'],
        [/"/, { token: 'string.quote', bracket: '@open', next: '@string' }],

        // Numbers
        [/\d*\.\d+([eE][\-+]?\d+)?/, 'number.float'],
        [/0[xX][0-9a-fA-F]+/, 'number.hex'],
        [/\d+/, 'number'],

        // Operators
        [/@symbols/, {
          cases: {
            '@operators': 'operator',
            '@default': ''
          }
        }],
      ],

      comment: [
        [/[^\-]+/, 'comment'],
        [/--/, 'comment', '@pop'],
        [/./, 'comment']
      ],

      string: [
        [/[^\\"]+/, 'string'],
        [/@escapes/, 'string.escape'],
        [/\\./, 'string.escape.invalid'],
        [/"/, { token: 'string.quote', bracket: '@close', next: '@pop' }]
      ],

      whitespace: [
        [/[ \t\r\n]+/, 'white'],
        [/--\[\[/, 'comment', '@comment'],
        [/--.*$/, 'comment'],
      ],
    },
  });

  // Auto-completion
  monaco.languages.registerCompletionItemProvider('lunar', {
    provideCompletionItems: (model, position) => {
      const suggestions = [
        {
          label: 'function',
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: 'function ${1:name}(${2:params}): ${3:void}\n\t${4}\nend',
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: 'Define a function'
        },
        {
          label: 'class',
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: 'class ${1:ClassName}\n\t${2}\nend',
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          documentation: 'Define a class'
        },
        // Add more snippets...
      ];

      return { suggestions };
    }
  });
}
```

## Option 2: WebAssembly Compilation

### Building Lunar for WebAssembly

#### 1. Install TinyGo (for smaller WASM binaries)
```bash
curl -L https://github.com/tinygo-org/tinygo/releases/download/v0.30.0/tinygo_0.30.0_amd64.deb -o tinygo.deb
sudo dpkg -i tinygo.deb
```

#### 2. Create WASM Wrapper

```go
// cmd/lunar-wasm/main.go
// +build wasm

package main

import (
    "encoding/json"
    "lunar/internal/compiler"
    "syscall/js"
)

func compile(this js.Value, args []js.Value) interface{} {
    if len(args) < 1 {
        return map[string]interface{}{
            "success": false,
            "error": "No code provided",
        }
    }

    code := args[0].String()

    // Parse options if provided
    options := compiler.DefaultOptions()
    if len(args) > 1 {
        optionsJSON := args[1].String()
        json.Unmarshal([]byte(optionsJSON), &options)
    }

    // Compile
    result, err := compiler.Compile(code, options)

    response := map[string]interface{}{
        "success": err == nil,
        "output": result.Output,
        "errors": result.Errors,
    }

    responseJSON, _ := json.Marshal(response)
    return string(responseJSON)
}

func main() {
    c := make(chan struct{})

    // Register functions
    js.Global().Set("lunarCompile", js.FuncOf(compile))

    // Keep alive
    <-c
}
```

#### 3. Build WASM Module
```bash
# Standard Go (larger file)
GOOS=js GOARCH=wasm go build -o lunar.wasm cmd/lunar-wasm/main.go

# TinyGo (smaller file, recommended)
tinygo build -o lunar.wasm -target wasm cmd/lunar-wasm/main.go
```

#### 4. Load in Browser

```html
<!-- public/index.html -->
<!DOCTYPE html>
<html>
<head>
    <title>Lunar Playground</title>
    <script src="wasm_exec.js"></script>
</head>
<body>
    <script>
        const go = new Go();

        WebAssembly.instantiateStreaming(
            fetch("lunar.wasm"),
            go.importObject
        ).then((result) => {
            go.run(result.instance);

            // Now you can call lunarCompile()
            const code = 'local x: number = 42\nprint(x)';
            const result = lunarCompile(code, JSON.stringify({
                typecheck: true,
                optimize: false
            }));

            console.log('Compilation result:', JSON.parse(result));
        });
    </script>
</body>
</html>
```

#### 5. React Integration

```typescript
// src/hooks/useLunarWasm.ts
import { useEffect, useState } from 'react';

declare global {
  function lunarCompile(code: string, options?: string): string;
}

export function useLunarWasm() {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const go = new (window as any).Go();

    WebAssembly.instantiateStreaming(
      fetch('/lunar.wasm'),
      go.importObject
    ).then((result) => {
      go.run(result.instance);
      setReady(true);
    });
  }, []);

  const compile = (code: string, options = {}) => {
    if (!ready) return null;

    try {
      const resultJSON = (window as any).lunarCompile(
        code,
        JSON.stringify(options)
      );
      return JSON.parse(resultJSON);
    } catch (error) {
      return { success: false, error: error.message };
    }
  };

  return { ready, compile };
}
```

## Security Considerations

### 1. Rate Limiting
```go
import "github.com/ulule/limiter/v3"

// Limit to 10 compilations per minute per IP
rate := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  10,
}

middleware := limiter.NewMiddleware(limiter.New(
    memory.NewStore(),
    rate,
))

r.Use(middleware)
```

### 2. Code Sandboxing
- Never execute compiled Lua code on the server
- If running Lua (for output preview), use Docker containers with resource limits
- Implement timeout for compilation (max 5 seconds)

### 3. Input Validation
```go
func validateCode(code string) error {
    // Max code length (100KB)
    if len(code) > 100000 {
        return errors.New("Code too long")
    }

    // Detect obvious malicious patterns
    if strings.Contains(code, "os.execute") ||
       strings.Contains(code, "io.popen") {
        return errors.New("Forbidden functions")
    }

    return nil
}
```

## Database Schema (for snippets)

```sql
CREATE TABLE snippets (
    id VARCHAR(12) PRIMARY KEY,
    code TEXT NOT NULL,
    title VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    views INT DEFAULT 0,
    is_public BOOLEAN DEFAULT true,
    user_id VARCHAR(36),
    INDEX idx_created (created_at),
    INDEX idx_user (user_id)
);

CREATE TABLE snippet_tags (
    snippet_id VARCHAR(12),
    tag VARCHAR(50),
    PRIMARY KEY (snippet_id, tag),
    FOREIGN KEY (snippet_id) REFERENCES snippets(id) ON DELETE CASCADE
);
```

## Deployment

### Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o lunar-api cmd/playground-api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/lunar-api .

EXPOSE 8080
CMD ["./lunar-api"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/lunar
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=lunar
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./frontend/build:/usr/share/nginx/html

volumes:
  postgres_data:
  redis_data:
```

## Features Checklist

### MVP Features
- [ ] Code editor with syntax highlighting
- [ ] Real-time compilation
- [ ] Error display with line numbers
- [ ] Basic code sharing (unique URLs)
- [ ] Copy to clipboard
- [ ] Download compiled output

### Advanced Features
- [ ] User accounts and saved snippets
- [ ] Multiple file support
- [ ] Code templates/examples gallery
- [ ] Lua execution with output preview
- [ ] Collaborative editing (WebSockets)
- [ ] Version history for snippets
- [ ] Comments and ratings
- [ ] Embed snippets in documentation
- [ ] Dark/light theme toggle
- [ ] Mobile-responsive design

## Performance Optimization

### 1. Caching Strategy
```go
// Cache compiled results
var cache = NewLRUCache(1000) // Keep 1000 most recent compilations

func handleCompile(c *gin.Context) {
    codeHash := sha256Hash(req.Code + req.Options.String())

    // Check cache
    if cached, ok := cache.Get(codeHash); ok {
        c.JSON(200, cached)
        return
    }

    // Compile and cache
    result := compile(req.Code, req.Options)
    cache.Set(codeHash, result, 1*time.Hour)

    c.JSON(200, result)
}
```

### 2. CDN for WASM
- Host lunar.wasm on CDN (Cloudflare, CloudFront)
- Enable compression (Brotli/Gzip)
- Use versioned URLs for cache busting

### 3. Worker Threads (for WASM)
```typescript
// Run compilation in Web Worker to avoid blocking UI
const worker = new Worker('/compiler-worker.js');

worker.postMessage({ code, options });

worker.onmessage = (e) => {
  const result = e.data;
  setOutput(result.output);
};
```

## Example Implementations

### Full Stack Example Repository Structure
```
lunar-playground/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go
│   │   └── wasm/main.go
│   ├── internal/
│   │   ├── compiler/
│   │   ├── snippets/
│   │   └── rate-limit/
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Editor.tsx
│   │   │   ├── Output.tsx
│   │   │   └── Toolbar.tsx
│   │   ├── hooks/
│   │   │   └── useLunarCompiler.ts
│   │   ├── monaco/
│   │   │   └── lunar-language.ts
│   │   └── App.tsx
│   ├── public/
│   │   ├── lunar.wasm
│   │   └── wasm_exec.js
│   └── package.json
├── docker-compose.yml
└── README.md
```

## Getting Started (For Implementation)

1. **Choose your approach** (Server-side recommended for MVP)
2. **Set up backend API** with endpoints above
3. **Create React frontend** with Monaco Editor
4. **Implement compilation** using existing Lunar compiler
5. **Add snippet storage** using PostgreSQL/MongoDB
6. **Deploy** using Docker Compose
7. **Optimize** with caching and CDN
8. **Monitor** with logging and analytics

## Support and Resources

- Monaco Editor: https://microsoft.github.io/monaco-editor/
- React Monaco Editor: https://github.com/suren-atoyan/monaco-react
- Go WASM: https://github.com/golang/go/wiki/WebAssembly
- TinyGo: https://tinygo.org/docs/guides/webassembly/

This documentation provides everything you need to build a production-ready Lunar playground!
