package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"lunar/internal/ast"
	"lunar/internal/lexer"
)

// PluginSystem manages all plugins
type PluginSystem struct {
	plugins       map[string]Plugin
	typeCheckers  []TypeCheckerPlugin
	transformers  []TransformerPlugin
	buildHooks    []BuildHookPlugin
	linters       []LinterPlugin
	mu            sync.RWMutex
	config        *Config
}

// Config represents plugin system configuration
type Config struct {
	PluginDir      string   // Directory containing plugins
	EnabledPlugins []string // List of enabled plugin names
	PluginConfig   map[string]interface{} // Per-plugin configuration
	AutoLoad       bool     // Auto-load plugins from directory
}

// NewPluginSystem creates a new plugin system
func NewPluginSystem(config *Config) *PluginSystem {
	if config == nil {
		config = DefaultConfig()
	}

	return &PluginSystem{
		plugins:      make(map[string]Plugin),
		typeCheckers: make([]TypeCheckerPlugin, 0),
		transformers: make([]TransformerPlugin, 0),
		buildHooks:   make([]BuildHookPlugin, 0),
		linters:      make([]LinterPlugin, 0),
		config:       config,
	}
}

// DefaultConfig returns default plugin configuration
func DefaultConfig() *Config {
	return &Config{
		PluginDir:      "./plugins",
		EnabledPlugins: make([]string, 0),
		PluginConfig:   make(map[string]interface{}),
		AutoLoad:       true,
	}
}

// Plugin is the base interface all plugins must implement
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Description returns a description of the plugin
	Description() string

	// Initialize initializes the plugin with configuration
	Initialize(config interface{}) error

	// Shutdown performs cleanup when plugin is unloaded
	Shutdown() error
}

// TypeCheckerPlugin extends the type checker
type TypeCheckerPlugin interface {
	Plugin

	// CheckType performs custom type checking
	CheckType(node ast.Expression, context *TypeContext) (*TypeResult, error)

	// ValidateType validates a type annotation
	ValidateType(typeExpr ast.Expression) error

	// InferType infers the type of an expression
	InferType(expr ast.Expression, context *TypeContext) (string, error)

	// RegisterCustomTypes registers custom types provided by this plugin
	RegisterCustomTypes() []CustomType
}

// TransformerPlugin performs AST transformations
type TransformerPlugin interface {
	Plugin

	// Transform transforms an AST node
	Transform(node ast.Statement, context *TransformContext) (ast.Statement, error)

	// Priority returns transformation priority (higher = earlier)
	Priority() int

	// ShouldTransform determines if this transformer should run
	ShouldTransform(node ast.Statement) bool
}

// BuildHookPlugin hooks into the build process
type BuildHookPlugin interface {
	Plugin

	// PreBuild runs before build starts
	PreBuild(context *BuildContext) error

	// PostBuild runs after build completes
	PostBuild(context *BuildContext, result *BuildResult) error

	// OnFileProcessed runs when a file is processed
	OnFileProcessed(file string, ast []ast.Statement) error
}

// LinterPlugin provides custom linting rules
type LinterPlugin interface {
	Plugin

	// Lint performs linting on AST
	Lint(ast []ast.Statement, context *LintContext) ([]LintIssue, error)

	// Rules returns the linting rules provided by this plugin
	Rules() []LintRule

	// CanAutoFix returns true if issues can be auto-fixed
	CanAutoFix(issue LintIssue) bool

	// AutoFix attempts to auto-fix an issue
	AutoFix(issue LintIssue, ast []ast.Statement) ([]ast.Statement, error)
}

// TypeContext provides context for type checking
type TypeContext struct {
	CurrentFile   string
	SymbolTable   map[string]string // symbol -> type
	Imports       map[string]string
	Scope         string
	InFunction    bool
	InClass       bool
	InInterface   bool
	StrictMode    bool
}

// TypeResult represents the result of type checking
type TypeResult struct {
	Type       string
	Valid      bool
	Errors     []string
	Warnings   []string
	Confidence float64 // 0-1
}

// CustomType represents a custom type definition
type CustomType struct {
	Name        string
	Definition  string
	Constructor func(args ...interface{}) interface{}
	Methods     map[string]string // method name -> signature
}

// TransformContext provides context for transformations
type TransformContext struct {
	CurrentFile string
	Options     map[string]interface{}
	Metadata    map[string]interface{}
}

// BuildContext provides context for build hooks
type BuildContext struct {
	RootDir     string
	OutputDir   string
	Files       []string
	Options     BuildOptions
	StartTime   int64
}

// BuildOptions represents build options
type BuildOptions struct {
	Optimize      bool
	Target        string // "lua" | "wasm" | etc
	OutputFormat  string
	SourceMaps    bool
	Minify        bool
}

// BuildResult represents build results
type BuildResult struct {
	Success      bool
	OutputFiles  []string
	Errors       []string
	Warnings     []string
	Duration     int64 // milliseconds
}

// LintContext provides context for linting
type LintContext struct {
	CurrentFile string
	Rules       []string // Enabled rules
	FixMode     bool     // Auto-fix mode
}

// LintIssue represents a linting issue
type LintIssue struct {
	Rule        string
	Severity    Severity
	Message     string
	File        string
	Line        int
	Column      int
	EndLine     int
	EndColumn   int
	Suggestion  string
	AutoFixable bool
}

// LintRule represents a linting rule
type LintRule struct {
	Name        string
	Description string
	Category    string   // "style" | "error" | "performance" | etc
	Severity    Severity
	Enabled     bool
}

// Severity represents issue severity
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
	SeverityHint
)

// LoadPlugin loads a plugin from a file
func (ps *PluginSystem) LoadPlugin(path string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Load plugin using Go's plugin package
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for exported Plugin symbol
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Plugin symbol: %w", path, err)
	}

	// Type assert to Plugin interface
	plug, ok := symPlugin.(Plugin)
	if !ok {
		return fmt.Errorf("plugin %s does not implement Plugin interface", path)
	}

	// Initialize plugin
	config := ps.config.PluginConfig[plug.Name()]
	if err := plug.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", plug.Name(), err)
	}

	// Register plugin
	ps.plugins[plug.Name()] = plug

	// Register specialized interfaces
	if tc, ok := plug.(TypeCheckerPlugin); ok {
		ps.typeCheckers = append(ps.typeCheckers, tc)
	}
	if tr, ok := plug.(TransformerPlugin); ok {
		ps.transformers = append(ps.transformers, tr)
	}
	if bh, ok := plug.(BuildHookPlugin); ok {
		ps.buildHooks = append(ps.buildHooks, bh)
	}
	if ln, ok := plug.(LinterPlugin); ok {
		ps.linters = append(ps.linters, ln)
	}

	return nil
}

// LoadAllPlugins loads all plugins from the plugin directory
func (ps *PluginSystem) LoadAllPlugins() error {
	if !ps.config.AutoLoad {
		return nil
	}

	// Check if plugin directory exists
	if _, err := os.Stat(ps.config.PluginDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}

	// Find all .so files in plugin directory
	files, err := filepath.Glob(filepath.Join(ps.config.PluginDir, "*.so"))
	if err != nil {
		return err
	}

	for _, file := range files {
		// Check if plugin is enabled
		pluginName := filepath.Base(file)
		if len(ps.config.EnabledPlugins) > 0 {
			enabled := false
			for _, name := range ps.config.EnabledPlugins {
				if name == pluginName {
					enabled = true
					break
				}
			}
			if !enabled {
				continue
			}
		}

		if err := ps.LoadPlugin(file); err != nil {
			return err
		}
	}

	return nil
}

// UnloadPlugin unloads a plugin
func (ps *PluginSystem) UnloadPlugin(name string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	plug, exists := ps.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Shutdown plugin
	if err := plug.Shutdown(); err != nil {
		return err
	}

	// Remove from registry
	delete(ps.plugins, name)

	// Remove from specialized lists
	ps.typeCheckers = filterTypeCheckers(ps.typeCheckers, name)
	ps.transformers = filterTransformers(ps.transformers, name)
	ps.buildHooks = filterBuildHooks(ps.buildHooks, name)
	ps.linters = filterLinters(ps.linters, name)

	return nil
}

// GetPlugin retrieves a plugin by name
func (ps *PluginSystem) GetPlugin(name string) (Plugin, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	plug, exists := ps.plugins[name]
	return plug, exists
}

// ListPlugins lists all loaded plugins
func (ps *PluginSystem) ListPlugins() []PluginInfo {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	info := make([]PluginInfo, 0, len(ps.plugins))
	for _, plug := range ps.plugins {
		info = append(info, PluginInfo{
			Name:        plug.Name(),
			Version:     plug.Version(),
			Description: plug.Description(),
		})
	}

	return info
}

// PluginInfo represents plugin information
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Type        []string // Types of plugin interfaces implemented
}

// RunTypeCheckers runs all type checker plugins
func (ps *PluginSystem) RunTypeCheckers(node ast.Expression, context *TypeContext) (*TypeResult, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := &TypeResult{
		Valid:      true,
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
		Confidence: 1.0,
	}

	for _, tc := range ps.typeCheckers {
		r, err := tc.CheckType(node, context)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Valid = false
			continue
		}

		if !r.Valid {
			result.Valid = false
		}

		result.Errors = append(result.Errors, r.Errors...)
		result.Warnings = append(result.Warnings, r.Warnings...)

		if r.Confidence < result.Confidence {
			result.Confidence = r.Confidence
		}

		if r.Type != "" && result.Type == "" {
			result.Type = r.Type
		}
	}

	return result, nil
}

// RunTransformers runs all transformer plugins
func (ps *PluginSystem) RunTransformers(statements []ast.Statement, context *TransformContext) ([]ast.Statement, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	// Sort transformers by priority
	sortedTransformers := make([]TransformerPlugin, len(ps.transformers))
	copy(sortedTransformers, ps.transformers)

	// Simple bubble sort by priority
	for i := 0; i < len(sortedTransformers); i++ {
		for j := i + 1; j < len(sortedTransformers); j++ {
			if sortedTransformers[j].Priority() > sortedTransformers[i].Priority() {
				sortedTransformers[i], sortedTransformers[j] = sortedTransformers[j], sortedTransformers[i]
			}
		}
	}

	// Apply transformers
	result := statements
	for _, tr := range sortedTransformers {
		transformed := make([]ast.Statement, 0, len(result))
		for _, stmt := range result {
			if tr.ShouldTransform(stmt) {
				newStmt, err := tr.Transform(stmt, context)
				if err != nil {
					return nil, err
				}
				transformed = append(transformed, newStmt)
			} else {
				transformed = append(transformed, stmt)
			}
		}
		result = transformed
	}

	return result, nil
}

// RunBuildHooks runs build hook plugins
func (ps *PluginSystem) RunPreBuild(context *BuildContext) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, bh := range ps.buildHooks {
		if err := bh.PreBuild(context); err != nil {
			return err
		}
	}

	return nil
}

// RunPostBuild runs post-build hooks
func (ps *PluginSystem) RunPostBuild(context *BuildContext, result *BuildResult) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, bh := range ps.buildHooks {
		if err := bh.PostBuild(context, result); err != nil {
			return err
		}
	}

	return nil
}

// RunLinters runs all linter plugins
func (ps *PluginSystem) RunLinters(statements []ast.Statement, context *LintContext) ([]LintIssue, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	allIssues := make([]LintIssue, 0)

	for _, linter := range ps.linters {
		issues, err := linter.Lint(statements, context)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// Helper functions to filter plugin lists

func filterTypeCheckers(list []TypeCheckerPlugin, name string) []TypeCheckerPlugin {
	result := make([]TypeCheckerPlugin, 0)
	for _, tc := range list {
		if tc.Name() != name {
			result = append(result, tc)
		}
	}
	return result
}

func filterTransformers(list []TransformerPlugin, name string) []TransformerPlugin {
	result := make([]TransformerPlugin, 0)
	for _, tr := range list {
		if tr.Name() != name {
			result = append(result, tr)
		}
	}
	return result
}

func filterBuildHooks(list []BuildHookPlugin, name string) []BuildHookPlugin {
	result := make([]BuildHookPlugin, 0)
	for _, bh := range list {
		if bh.Name() != name {
			result = append(result, bh)
		}
	}
	return result
}

func filterLinters(list []LinterPlugin, name string) []LinterPlugin {
	result := make([]LinterPlugin, 0)
	for _, ln := range list {
		if ln.Name() != name {
			result = append(result, ln)
		}
	}
	return result
}

// PluginRegistry maintains a global registry of available plugins
type PluginRegistry struct {
	available map[string]PluginMetadata
	mu        sync.RWMutex
}

// PluginMetadata contains metadata about an available plugin
type PluginMetadata struct {
	Name        string
	Version     string
	Author      string
	Description string
	Repository  string
	License     string
	Tags        []string
}

// GlobalRegistry is the global plugin registry
var GlobalRegistry = &PluginRegistry{
	available: make(map[string]PluginMetadata),
}

// Register registers a plugin in the global registry
func (pr *PluginRegistry) Register(metadata PluginMetadata) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.available[metadata.Name] = metadata
}

// Get retrieves plugin metadata
func (pr *PluginRegistry) Get(name string) (PluginMetadata, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	metadata, exists := pr.available[name]
	return metadata, exists
}

// List lists all available plugins
func (pr *PluginRegistry) List() []PluginMetadata {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	list := make([]PluginMetadata, 0, len(pr.available))
	for _, metadata := range pr.available {
		list = append(list, metadata)
	}

	return list
}
