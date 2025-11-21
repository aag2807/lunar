# Lunar Plugin System

Extend Lunar's capabilities with a powerful plugin system supporting custom type checkers, AST transformations, linters, and build hooks.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Plugin Types](#plugin-types)
- [Creating Plugins](#creating-plugins)
- [Plugin API](#plugin-api)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Publishing Plugins](#publishing-plugins)

## Overview

The Lunar plugin system allows you to:

- **Extend the type checker** - Add custom type validation rules
- **Transform AST** - Modify code during compilation
- **Custom linting** - Create project-specific linting rules
- **Build hooks** - Integrate with the build process
- **Code generation** - Generate boilerplate code

### Architecture

```
┌─────────────────────────────────────┐
│         Lunar Compiler              │
├─────────────────────────────────────┤
│  ┌───────────────────────────────┐  │
│  │      Plugin System            │  │
│  ├───────────────────────────────┤  │
│  │  • Type Checker Plugins       │  │
│  │  • Transformer Plugins        │  │
│  │  • Linter Plugins             │  │
│  │  • Build Hook Plugins         │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
           │        │
           ▼        ▼
      User         Official
    Plugins       Plugins
```

## Quick Start

### 1. Enable Plugins

```bash
# Initialize plugin directory
mkdir -p plugins

# Configure plugins
lunar --init-plugins
```

This creates `lunar.plugins.json`:

```json
{
  "plugins": {
    "directory": "./plugins",
    "autoLoad": true,
    "enabled": [],
    "config": {}
  }
}
```

### 2. Install a Plugin

```bash
# Install from repository
lunar plugin install typescript-strict

# Or install local plugin
lunar plugin install ./plugins/my-plugin.so
```

### 3. Use Plugin

```bash
# Plugins are automatically loaded
lunar build myapp.lunar

# List loaded plugins
lunar plugin list
```

## Plugin Types

### 1. Type Checker Plugins

Extend the type system with custom validation rules.

```go
type TypeCheckerPlugin interface {
    Plugin

    CheckType(node ast.Expression, context *TypeContext) (*TypeResult, error)
    ValidateType(typeExpr ast.Expression) error
    InferType(expr ast.Expression, context *TypeContext) (string, error)
    RegisterCustomTypes() []CustomType
}
```

**Use cases:**
- Domain-specific type validation
- Custom generic types
- Advanced type inference
- Type aliases and utilities

### 2. Transformer Plugins

Modify the AST during compilation.

```go
type TransformerPlugin interface {
    Plugin

    Transform(node ast.Statement, context *TransformContext) (ast.Statement, error)
    Priority() int
    ShouldTransform(node ast.Statement) bool
}
```

**Use cases:**
- Code optimization
- Syntactic sugar expansion
- Code generation
- Instrumentation

### 3. Linter Plugins

Add custom linting rules.

```go
type LinterPlugin interface {
    Plugin

    Lint(ast []ast.Statement, context *LintContext) ([]LintIssue, error)
    Rules() []LintRule
    CanAutoFix(issue LintIssue) bool
    AutoFix(issue LintIssue, ast []ast.Statement) ([]ast.Statement, error)
}
```

**Use cases:**
- Style enforcement
- Best practices checking
- Security auditing
- Performance linting

### 4. Build Hook Plugins

Hook into the build process.

```go
type BuildHookPlugin interface {
    Plugin

    PreBuild(context *BuildContext) error
    PostBuild(context *BuildContext, result *BuildResult) error
    OnFileProcessed(file string, ast []ast.Statement) error
}
```

**Use cases:**
- Asset generation
- Documentation generation
- Test running
- Deployment tasks

## Creating Plugins

### Plugin Template

```bash
# Generate plugin scaffold
lunar plugin create my-plugin --type type-checker

# This creates:
# plugins/my-plugin/
#   ├── main.go
#   ├── go.mod
#   ├── README.md
#   └── config.json
```

### Basic Plugin Structure

```go
// plugins/my-plugin/main.go
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

// MyPlugin implements a custom Lunar plugin
type MyPlugin struct {
    name        string
    version     string
    description string
    config      *Config
}

// Config holds plugin configuration
type Config struct {
    StrictMode bool   `json:"strictMode"`
    MaxErrors  int    `json:"maxErrors"`
}

// Plugin is the exported symbol Lunar will load
var Plugin plugin.Plugin = &MyPlugin{
    name:        "my-plugin",
    version:     "1.0.0",
    description: "My custom Lunar plugin",
}

// Implement Plugin interface
func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) Version() string {
    return p.version
}

func (p *MyPlugin) Description() string {
    return p.description
}

func (p *MyPlugin) Initialize(config interface{}) error {
    // Load configuration
    if cfg, ok := config.(*Config); ok {
        p.config = cfg
    }
    return nil
}

func (p *MyPlugin) Shutdown() error {
    // Cleanup resources
    return nil
}
```

### Building Plugins

```bash
# Build plugin as shared library
cd plugins/my-plugin
go build -buildmode=plugin -o ../my-plugin.so

# Or use Lunar's build command
lunar plugin build my-plugin
```

## Plugin API

### Type Checker Plugin

```go
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type StrictTypeChecker struct {
    plugin.BasePlugin
}

var Plugin plugin.Plugin = &StrictTypeChecker{}

func (p *StrictTypeChecker) CheckType(node ast.Expression, context *plugin.TypeContext) (*plugin.TypeResult, error) {
    result := &plugin.TypeResult{
        Valid:      true,
        Errors:     make([]string, 0),
        Warnings:   make([]string, 0),
        Confidence: 1.0,
    }

    // Example: Disallow 'any' type in strict mode
    if context.StrictMode {
        if typeAnnot, ok := node.(*ast.TypeAnnotation); ok {
            if typeAnnot.Type == "any" {
                result.Valid = false
                result.Errors = append(result.Errors,
                    "Type 'any' not allowed in strict mode")
            }
        }
    }

    return result, nil
}

func (p *StrictTypeChecker) InferType(expr ast.Expression, context *plugin.TypeContext) (string, error) {
    switch e := expr.(type) {
    case *ast.NumberLiteral:
        return "number", nil
    case *ast.StringLiteral:
        return "string", nil
    case *ast.BooleanLiteral:
        return "boolean", nil
    default:
        return "unknown", nil
    }
}

func (p *StrictTypeChecker) RegisterCustomTypes() []plugin.CustomType {
    return []plugin.CustomType{
        {
            Name:       "NonEmptyString",
            Definition: "string",
            Methods: map[string]string{
                "isEmpty": "function(): boolean",
            },
        },
    }
}
```

### Transformer Plugin

```go
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type OptimizingTransformer struct {
    plugin.BasePlugin
    priority int
}

var Plugin plugin.Plugin = &OptimizingTransformer{
    priority: 100, // Higher = runs earlier
}

func (p *OptimizingTransformer) Priority() int {
    return p.priority
}

func (p *OptimizingTransformer) ShouldTransform(node ast.Statement) bool {
    // Only transform constant expressions
    _, isConst := node.(*ast.ConstDeclaration)
    return isConst
}

func (p *OptimizingTransformer) Transform(node ast.Statement, context *plugin.TransformContext) (ast.Statement, error) {
    // Example: Fold constant expressions
    if constDecl, ok := node.(*ast.ConstDeclaration); ok {
        if binExpr, ok := constDecl.Value.(*ast.BinaryExpression); ok {
            // Check if both operands are number literals
            if left, ok := binExpr.Left.(*ast.NumberLiteral); ok {
                if right, ok := binExpr.Right.(*ast.NumberLiteral); ok {
                    // Perform constant folding
                    var result float64
                    switch binExpr.Operator {
                    case "+":
                        result = left.Value + right.Value
                    case "-":
                        result = left.Value - right.Value
                    case "*":
                        result = left.Value * right.Value
                    case "/":
                        if right.Value != 0 {
                            result = left.Value / right.Value
                        }
                    }

                    // Replace with folded constant
                    constDecl.Value = &ast.NumberLiteral{Value: result}
                }
            }
        }
    }

    return node, nil
}
```

### Linter Plugin

```go
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type StyleLinter struct {
    plugin.BasePlugin
    rules []plugin.LintRule
}

var Plugin plugin.Plugin = &StyleLinter{
    rules: []plugin.LintRule{
        {
            Name:        "no-unused-vars",
            Description: "Variables must be used",
            Category:    "error",
            Severity:    plugin.SeverityError,
            Enabled:     true,
        },
        {
            Name:        "prefer-const",
            Description: "Use const for immutable variables",
            Category:    "style",
            Severity:    plugin.SeverityWarning,
            Enabled:     true,
        },
    },
}

func (p *StyleLinter) Rules() []plugin.LintRule {
    return p.rules
}

func (p *StyleLinter) Lint(statements []ast.Statement, context *plugin.LintContext) ([]plugin.LintIssue, error) {
    issues := make([]plugin.LintIssue, 0)

    // Track declared and used variables
    declared := make(map[string]*ast.Identifier)
    used := make(map[string]bool)

    // First pass: collect declarations
    for _, stmt := range statements {
        if assign, ok := stmt.(*ast.AssignmentStatement); ok {
            if ident, ok := assign.Name.(*ast.Identifier); ok {
                declared[ident.Value] = ident
            }
        }
    }

    // Second pass: track usage
    // (simplified - real implementation would walk entire AST)

    // Third pass: find unused variables
    for name, ident := range declared {
        if !used[name] {
            issues = append(issues, plugin.LintIssue{
                Rule:        "no-unused-vars",
                Severity:    plugin.SeverityError,
                Message:     fmt.Sprintf("Variable '%s' is declared but never used", name),
                File:        context.CurrentFile,
                Line:        ident.Token.Line,
                Column:      ident.Token.Column,
                AutoFixable: true,
            })
        }
    }

    return issues, nil
}

func (p *StyleLinter) CanAutoFix(issue plugin.LintIssue) bool {
    return issue.AutoFixable
}

func (p *StyleLinter) AutoFix(issue plugin.LintIssue, statements []ast.Statement) ([]ast.Statement, error) {
    // Remove unused variable declarations
    fixed := make([]ast.Statement, 0)

    for _, stmt := range statements {
        // Check if this is the statement to remove
        shouldRemove := false

        if assign, ok := stmt.(*ast.AssignmentStatement); ok {
            if ident, ok := assign.Name.(*ast.Identifier); ok {
                if ident.Token.Line == issue.Line {
                    shouldRemove = true
                }
            }
        }

        if !shouldRemove {
            fixed = append(fixed, stmt)
        }
    }

    return fixed, nil
}
```

### Build Hook Plugin

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type DocGenerator struct {
    plugin.BasePlugin
    outputDir string
}

var Plugin plugin.Plugin = &DocGenerator{
    outputDir: "./docs/api",
}

func (p *DocGenerator) PreBuild(context *plugin.BuildContext) error {
    // Create output directory
    if err := os.MkdirAll(p.outputDir, 0755); err != nil {
        return err
    }

    fmt.Printf("Documentation will be generated in %s\n", p.outputDir)
    return nil
}

func (p *DocGenerator) PostBuild(context *plugin.BuildContext, result *plugin.BuildResult) error {
    if result.Success {
        fmt.Printf("Build successful. Generated docs for %d files\n", len(context.Files))

        // Generate index.html
        indexPath := filepath.Join(p.outputDir, "index.html")
        // ... generate documentation ...

        return nil
    }

    return nil
}

func (p *DocGenerator) OnFileProcessed(file string, statements []ast.Statement) error {
    // Extract documentation comments and generate docs for this file
    docs := extractDocs(statements)

    docFile := filepath.Join(p.outputDir, filepath.Base(file)+".md")
    // ... write documentation ...

    return nil
}
```

## Examples

### Example 1: React-Style JSX Plugin

```go
// Transform JSX-like syntax to Lunar
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type JSXTransformer struct {
    plugin.BasePlugin
}

var Plugin plugin.Plugin = &JSXTransformer{}

func (p *JSXTransformer) ShouldTransform(node ast.Statement) bool {
    // Check for JSX-like syntax
    // <Component prop="value">children</Component>
    return false // Simplified
}

func (p *JSXTransformer) Transform(node ast.Statement, context *plugin.TransformContext) (ast.Statement, error) {
    // Transform JSX to function calls
    // <Button onClick={handler}>Click</Button>
    // becomes:
    // createElement(Button, {onClick = handler}, "Click")

    return node, nil
}
```

### Example 2: SQL Query Builder

```go
// Type-safe SQL query builder
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type SQLPlugin struct {
    plugin.BasePlugin
}

var Plugin plugin.Plugin = &SQLPlugin{}

func (p *SQLPlugin) RegisterCustomTypes() []plugin.CustomType {
    return []plugin.CustomType{
        {
            Name: "SQLQuery",
            Methods: map[string]string{
                "select": "function(...: string): SQLQuery",
                "from": "function(table: string): SQLQuery",
                "where": "function(condition: string): SQLQuery",
                "execute": "function(): array<any>",
            },
        },
    }
}

// Example usage in Lunar:
// const query: SQLQuery = SQL
//     .select("name", "email")
//     .from("users")
//     .where("age > 18")
//
// const results = query.execute()
```

### Example 3: Performance Monitor

```go
// Auto-instrument code for performance monitoring
package main

import (
    "lunar/internal/plugin"
    "lunar/internal/ast"
)

type PerfMonitor struct {
    plugin.BasePlugin
}

var Plugin plugin.Plugin = &PerfMonitor{}

func (p *PerfMonitor) Transform(node ast.Statement, context *plugin.TransformContext) (ast.Statement, error) {
    // Wrap functions with timing code
    if fn, ok := node.(*ast.FunctionStatement); ok {
        // Insert: local __start = os.clock()
        // ... original function body ...
        // Insert: local __duration = os.clock() - __start
        // Insert: recordMetric(functionName, __duration)
    }

    return node, nil
}
```

## Best Practices

### 1. Version Compatibility

```go
// Check Lunar version in Initialize()
func (p *MyPlugin) Initialize(config interface{}) error {
    minVersion := "1.0.0"
    if !isCompatible(lunar.Version(), minVersion) {
        return fmt.Errorf("requires Lunar >= %s", minVersion)
    }
    return nil
}
```

### 2. Error Handling

```go
// Always return descriptive errors
func (p *MyPlugin) CheckType(node ast.Expression, context *TypeContext) (*TypeResult, error) {
    result := &TypeResult{Valid: true}

    if err := validateNode(node); err != nil {
        result.Valid = false
        result.Errors = append(result.Errors,
            fmt.Sprintf("Validation failed at %s:%d: %v",
                context.CurrentFile, node.Line(), err))
    }

    return result, nil
}
```

### 3. Performance

```go
// Cache expensive computations
type MyPlugin struct {
    cache map[string]*CachedResult
    mu    sync.RWMutex
}

func (p *MyPlugin) CheckType(node ast.Expression, context *TypeContext) (*TypeResult, error) {
    key := computeKey(node)

    p.mu.RLock()
    if cached, ok := p.cache[key]; ok {
        p.mu.RUnlock()
        return cached, nil
    }
    p.mu.RUnlock()

    // Compute result...
    result := expensiveComputation(node)

    p.mu.Lock()
    p.cache[key] = result
    p.mu.Unlock()

    return result, nil
}
```

### 4. Configuration

```go
// Provide sensible defaults
type Config struct {
    Enabled     bool     `json:"enabled"`
    StrictMode  bool     `json:"strictMode"`
    IgnoreFiles []string `json:"ignoreFiles"`
    MaxWarnings int      `json:"maxWarnings"`
}

func DefaultConfig() *Config {
    return &Config{
        Enabled:     true,
        StrictMode:  false,
        IgnoreFiles: []string{"*.test.lunar"},
        MaxWarnings: 100,
    }
}
```

## Publishing Plugins

### 1. Package Structure

```
my-plugin/
├── README.md
├── LICENSE
├── go.mod
├── go.sum
├── main.go
├── config.schema.json
├── examples/
│   └── basic.lunar
└── tests/
    └── plugin_test.go
```

### 2. Metadata

```json
// plugin.json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "author": "Your Name",
  "description": "Brief description",
  "repository": "https://github.com/you/my-plugin",
  "license": "MIT",
  "tags": ["type-checker", "linter"],
  "lunarVersion": ">=1.0.0",
  "dependencies": {}
}
```

### 3. Documentation

Include comprehensive README with:
- Installation instructions
- Configuration options
- Usage examples
- API reference
- Changelog

### 4. Testing

```go
// plugin_test.go
package main

import (
    "testing"
    "lunar/internal/plugin"
)

func TestPluginInitialize(t *testing.T) {
    p := Plugin.(*MyPlugin)
    config := DefaultConfig()

    if err := p.Initialize(config); err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }
}

func TestTypeChecking(t *testing.T) {
    // Test your plugin functionality
}
```

### 5. Publish

```bash
# Tag version
git tag v1.0.0
git push --tags

# Publish to plugin registry
lunar plugin publish

# Or publish to npm/GitHub
npm publish
```

## Plugin Registry

### Search Plugins

```bash
# Search for plugins
lunar plugin search type-checker

# View plugin details
lunar plugin info typescript-strict
```

### Install from Registry

```bash
# Install latest version
lunar plugin install typescript-strict

# Install specific version
lunar plugin install typescript-strict@1.2.0
```

### List Installed

```bash
# List all installed plugins
lunar plugin list

# List enabled plugins
lunar plugin list --enabled
```

## Summary

The Lunar plugin system provides:

- ✅ Four plugin types (type checker, transformer, linter, build hooks)
- ✅ Simple Go-based API
- ✅ Hot reload support
- ✅ Configuration management
- ✅ Plugin registry

Start building: `lunar plugin create my-plugin` 🔌
