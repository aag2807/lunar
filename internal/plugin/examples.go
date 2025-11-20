package plugin

import (
	"fmt"
	"strings"

	"lunar/internal/ast"
)

// ExampleTypeCheckerPlugin demonstrates a custom type checker plugin
type ExampleTypeCheckerPlugin struct {
	name        string
	version     string
	description string
}

// NewExampleTypeChecker creates a new example type checker
func NewExampleTypeChecker() *ExampleTypeCheckerPlugin {
	return &ExampleTypeCheckerPlugin{
		name:        "example-type-checker",
		version:     "1.0.0",
		description: "Example type checker plugin that enforces custom type rules",
	}
}

func (p *ExampleTypeCheckerPlugin) Name() string        { return p.name }
func (p *ExampleTypeCheckerPlugin) Version() string     { return p.version }
func (p *ExampleTypeCheckerPlugin) Description() string { return p.description }

func (p *ExampleTypeCheckerPlugin) Initialize(config interface{}) error {
	// Initialize plugin with configuration
	return nil
}

func (p *ExampleTypeCheckerPlugin) Shutdown() error {
	// Cleanup resources
	return nil
}

func (p *ExampleTypeCheckerPlugin) CheckType(node ast.Expression, context *TypeContext) (*TypeResult, error) {
	result := &TypeResult{
		Valid:      true,
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
		Confidence: 1.0,
	}

	// Example: Check for unsafe number operations
	if binOp, ok := node.(*ast.BinaryExpression); ok {
		if binOp.Operator == "/" {
			result.Warnings = append(result.Warnings, "Division operation detected - consider checking for division by zero")
			result.Confidence = 0.9
		}
	}

	return result, nil
}

func (p *ExampleTypeCheckerPlugin) ValidateType(typeExpr ast.Expression) error {
	// Validate custom type syntax
	return nil
}

func (p *ExampleTypeCheckerPlugin) InferType(expr ast.Expression, context *TypeContext) (string, error) {
	// Infer type from expression
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		return "number", nil
	case *ast.StringLiteral:
		return "string", nil
	case *ast.BooleanLiteral:
		return "boolean", nil
	default:
		_ = e
		return "any", nil
	}
}

func (p *ExampleTypeCheckerPlugin) RegisterCustomTypes() []CustomType {
	// Register custom types
	return []CustomType{
		{
			Name:       "PositiveNumber",
			Definition: "number", // Base type
			Methods: map[string]string{
				"increment": "function(): PositiveNumber",
			},
		},
	}
}

// ExampleTransformerPlugin demonstrates AST transformation
type ExampleTransformerPlugin struct {
	name        string
	version     string
	description string
	priority    int
}

// NewExampleTransformer creates a new example transformer
func NewExampleTransformer() *ExampleTransformerPlugin {
	return &ExampleTransformerPlugin{
		name:        "example-transformer",
		version:     "1.0.0",
		description: "Example transformer that optimizes code",
		priority:    100,
	}
}

func (p *ExampleTransformerPlugin) Name() string        { return p.name }
func (p *ExampleTransformerPlugin) Version() string     { return p.version }
func (p *ExampleTransformerPlugin) Description() string { return p.description }
func (p *ExampleTransformerPlugin) Priority() int       { return p.priority }

func (p *ExampleTransformerPlugin) Initialize(config interface{}) error {
	return nil
}

func (p *ExampleTransformerPlugin) Shutdown() error {
	return nil
}

func (p *ExampleTransformerPlugin) ShouldTransform(node ast.Statement) bool {
	// Only transform if statements
	_, isIf := node.(*ast.IfStatement)
	return isIf
}

func (p *ExampleTransformerPlugin) Transform(node ast.Statement, context *TransformContext) (ast.Statement, error) {
	// Example: Simplify if statements with constant conditions
	if ifStmt, ok := node.(*ast.IfStatement); ok {
		if boolLit, ok := ifStmt.Condition.(*ast.BooleanLiteral); ok {
			if boolLit.Value {
				// If condition is always true, return just the consequence
				// This is a simplified example - in practice would return block statements
				return ifStmt, nil
			} else {
				// If condition is always false, return alternative or empty
				if ifStmt.Alternative != nil {
					return ifStmt, nil
				}
			}
		}
	}

	return node, nil
}

// ExampleLinterPlugin demonstrates custom linting rules
type ExampleLinterPlugin struct {
	name        string
	version     string
	description string
	rules       []LintRule
}

// NewExampleLinter creates a new example linter
func NewExampleLinter() *ExampleLinterPlugin {
	return &ExampleLinterPlugin{
		name:        "example-linter",
		version:     "1.0.0",
		description: "Example linter with custom rules",
		rules: []LintRule{
			{
				Name:        "no-magic-numbers",
				Description: "Disallow magic numbers",
				Category:    "style",
				Severity:    SeverityWarning,
				Enabled:     true,
			},
			{
				Name:        "prefer-const",
				Description: "Prefer const for immutable variables",
				Category:    "style",
				Severity:    SeverityInfo,
				Enabled:     true,
			},
		},
	}
}

func (p *ExampleLinterPlugin) Name() string        { return p.name }
func (p *ExampleLinterPlugin) Version() string     { return p.version }
func (p *ExampleLinterPlugin) Description() string { return p.description }

func (p *ExampleLinterPlugin) Initialize(config interface{}) error {
	return nil
}

func (p *ExampleLinterPlugin) Shutdown() error {
	return nil
}

func (p *ExampleLinterPlugin) Rules() []LintRule {
	return p.rules
}

func (p *ExampleLinterPlugin) Lint(statements []ast.Statement, context *LintContext) ([]LintIssue, error) {
	issues := make([]LintIssue, 0)

	for _, stmt := range statements {
		// Check for magic numbers
		p.checkMagicNumbers(stmt, context, &issues)

		// Check for const usage
		p.checkConstUsage(stmt, context, &issues)
	}

	return issues, nil
}

func (p *ExampleLinterPlugin) checkMagicNumbers(stmt ast.Statement, context *LintContext, issues *[]LintIssue) {
	// Walk AST looking for number literals in non-declaration contexts
	// This is a simplified example
}

func (p *ExampleLinterPlugin) checkConstUsage(stmt ast.Statement, context *LintContext, issues *[]LintIssue) {
	// Check if variable should be const
	if assign, ok := stmt.(*ast.AssignmentStatement); ok {
		if ident, ok := assign.Name.(*ast.Identifier); ok {
			// If variable is never reassigned, suggest const
			*issues = append(*issues, LintIssue{
				Rule:        "prefer-const",
				Severity:    SeverityInfo,
				Message:     fmt.Sprintf("Variable '%s' is never reassigned - consider using const", ident.Value),
				File:        context.CurrentFile,
				Suggestion:  fmt.Sprintf("Change 'local %s' to 'const %s'", ident.Value, ident.Value),
				AutoFixable: true,
			})
		}
	}
}

func (p *ExampleLinterPlugin) CanAutoFix(issue LintIssue) bool {
	return issue.AutoFixable
}

func (p *ExampleLinterPlugin) AutoFix(issue LintIssue, statements []ast.Statement) ([]ast.Statement, error) {
	// Auto-fix implementation
	return statements, nil
}

// ExampleBuildHookPlugin demonstrates build process hooks
type ExampleBuildHookPlugin struct {
	name        string
	version     string
	description string
}

// NewExampleBuildHook creates a new example build hook
func NewExampleBuildHook() *ExampleBuildHookPlugin {
	return &ExampleBuildHookPlugin{
		name:        "example-build-hook",
		version:     "1.0.0",
		description: "Example build hook plugin",
	}
}

func (p *ExampleBuildHookPlugin) Name() string        { return p.name }
func (p *ExampleBuildHookPlugin) Version() string     { return p.version }
func (p *ExampleBuildHookPlugin) Description() string { return p.description }

func (p *ExampleBuildHookPlugin) Initialize(config interface{}) error {
	return nil
}

func (p *ExampleBuildHookPlugin) Shutdown() error {
	return nil
}

func (p *ExampleBuildHookPlugin) PreBuild(context *BuildContext) error {
	// Pre-build actions
	fmt.Printf("Starting build for %s\n", context.RootDir)
	fmt.Printf("Building %d files\n", len(context.Files))

	// Could validate configuration, check dependencies, etc.
	return nil
}

func (p *ExampleBuildHookPlugin) PostBuild(context *BuildContext, result *BuildResult) error {
	// Post-build actions
	if result.Success {
		fmt.Printf("Build completed successfully in %dms\n", result.Duration)
		fmt.Printf("Generated %d output files\n", len(result.OutputFiles))

		// Could run additional validation, generate reports, etc.
	} else {
		fmt.Printf("Build failed with %d errors\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	return nil
}

func (p *ExampleBuildHookPlugin) OnFileProcessed(file string, statements []ast.Statement) error {
	// Called for each processed file
	fmt.Printf("Processed file: %s (%d statements)\n", file, len(statements))
	return nil
}

// PluginTemplate provides a template for creating new plugins
const PluginTemplate = `package main

import (
	"lunar/internal/plugin"
	"lunar/internal/ast"
)

// MyPlugin is a custom Lunar plugin
type MyPlugin struct {
	name        string
	version     string
	description string
}

// Plugin is the exported symbol that Lunar will load
var Plugin plugin.Plugin = &MyPlugin{
	name:        "my-plugin",
	version:     "1.0.0",
	description: "My custom Lunar plugin",
}

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
	// Initialize your plugin with configuration
	return nil
}

func (p *MyPlugin) Shutdown() error {
	// Cleanup resources
	return nil
}

// Implement additional interfaces as needed:
// - plugin.TypeCheckerPlugin
// - plugin.TransformerPlugin
// - plugin.BuildHookPlugin
// - plugin.LinterPlugin

// Example type checker implementation:
func (p *MyPlugin) CheckType(node ast.Expression, context *plugin.TypeContext) (*plugin.TypeResult, error) {
	result := &plugin.TypeResult{
		Valid:      true,
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
		Confidence: 1.0,
	}

	// Implement your custom type checking logic here

	return result, nil
}

func (p *MyPlugin) ValidateType(typeExpr ast.Expression) error {
	return nil
}

func (p *MyPlugin) InferType(expr ast.Expression, context *plugin.TypeContext) (string, error) {
	return "any", nil
}

func (p *MyPlugin) RegisterCustomTypes() []plugin.CustomType {
	return []plugin.CustomType{}
}
`

// BuildScriptTemplate provides a template for building plugins
const BuildScriptTemplate = `#!/bin/bash
# Build script for Lunar plugins

PLUGIN_NAME="my-plugin"
OUTPUT="plugins/${PLUGIN_NAME}.so"

echo "Building ${PLUGIN_NAME}..."

go build -buildmode=plugin -o "${OUTPUT}" ./plugins/${PLUGIN_NAME}

if [ $? -eq 0 ]; then
    echo "✓ Plugin built successfully: ${OUTPUT}"
else
    echo "✗ Plugin build failed"
    exit 1
fi
`

// PluginConfigTemplate provides a template for plugin configuration
const PluginConfigTemplate = `{
  "plugins": {
    "directory": "./plugins",
    "autoLoad": true,
    "enabled": [
      "example-type-checker",
      "example-linter"
    ],
    "config": {
      "example-type-checker": {
        "strictMode": true,
        "checkDivisionByZero": true
      },
      "example-linter": {
        "rules": {
          "no-magic-numbers": {
            "enabled": true,
            "severity": "warning",
            "ignore": [0, 1, -1]
          },
          "prefer-const": {
            "enabled": true,
            "severity": "info"
          }
        }
      }
    }
  }
}
`

// GeneratePluginSkeleton generates a new plugin skeleton
func GeneratePluginSkeleton(name string, pluginType string) (string, error) {
	template := strings.ReplaceAll(PluginTemplate, "my-plugin", name)
	template = strings.ReplaceAll(template, "MyPlugin", toPascalCase(name))

	return template, nil
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
