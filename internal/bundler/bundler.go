package bundler

import (
	"fmt"
	"io/ioutil"
	"lunar/internal/ast"
	"lunar/internal/codegen"
	"lunar/internal/config"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"os"
	"path/filepath"
	"strings"
)

// Module represents a parsed Lunar module
type Module struct {
	Path        string          // Resolved file path
	ModuleID    string          // Module identifier for require()
	Source      string          // Source code
	Statements  []ast.Statement // Parsed AST
	Imports     []string        // Module IDs this module depends on
	Exports     []string        // Exported names
	UsedExports map[string]bool // Exports that are actually imported (for tree shaking)
}

// Bundler handles bundling multiple Lunar files into a single output
type Bundler struct {
	entryFile    string
	baseDir      string
	modules      map[string]*Module // moduleID -> Module
	loadOrder    []string           // Topologically sorted module IDs
	typeCheck    bool
	declStmts    []ast.Statement // Declaration statements for type checking
	config       *config.Config  // Project configuration
}

// New creates a new Bundler
func New(entryFile string, typeCheck bool) *Bundler {
	absPath, _ := filepath.Abs(entryFile)
	return &Bundler{
		entryFile: absPath,
		baseDir:   filepath.Dir(absPath),
		modules:   make(map[string]*Module),
		typeCheck: typeCheck,
		config:    config.DefaultConfig(),
	}
}

// NewWithConfig creates a new Bundler with configuration
func NewWithConfig(entryFile string, typeCheck bool, cfg *config.Config) *Bundler {
	absPath, _ := filepath.Abs(entryFile)
	return &Bundler{
		entryFile: absPath,
		baseDir:   filepath.Dir(absPath),
		modules:   make(map[string]*Module),
		typeCheck: typeCheck,
		config:    cfg,
	}
}

// Bundle builds the dependency graph and generates bundled output
func (b *Bundler) Bundle() (string, error) {
	// Load declaration files for type checking
	if b.typeCheck {
		declFiles, err := b.discoverDeclarationFiles()
		if err != nil {
			return "", fmt.Errorf("failed to discover declaration files: %w", err)
		}
		for _, declFile := range declFiles {
			stmts, err := b.parseFile(declFile)
			if err != nil {
				return "", fmt.Errorf("failed to parse declaration file %s: %w", declFile, err)
			}
			b.declStmts = append(b.declStmts, stmts...)
		}
	}

	// Build dependency graph starting from entry file
	entryModuleID := "./" + filepath.Base(strings.TrimSuffix(b.entryFile, ".lunar"))
	if err := b.loadModule(b.entryFile, entryModuleID); err != nil {
		return "", err
	}

	// Topological sort to determine load order
	if err := b.topologicalSort(); err != nil {
		return "", err
	}

	// Type check all modules together
	if b.typeCheck {
		if err := b.typeCheckAll(); err != nil {
			return "", err
		}
	}

	// Analyze used exports for tree shaking
	if b.config.CompilerOptions.TreeShake {
		b.analyzeUsedExports()
	}

	// Generate bundled output
	return b.generateBundle(entryModuleID), nil
}

// analyzeUsedExports marks which exports are actually used by analyzing imports
func (b *Bundler) analyzeUsedExports() {
	// Initialize UsedExports maps
	for _, module := range b.modules {
		module.UsedExports = make(map[string]bool)
	}

	// Analyze each module's imports
	for _, module := range b.modules {
		for _, stmt := range module.Statements {
			if importStmt, ok := stmt.(*ast.ImportStatement); ok {
				// Find the target module
				depModule := b.findModuleByImport(importStmt.Module, module.Path)
				if depModule == nil {
					continue
				}

				if importStmt.IsWildcard {
					// Wildcard import uses all exports
					for _, export := range depModule.Exports {
						depModule.UsedExports[export] = true
					}
				} else {
					// Named imports - mark specific exports as used
					for _, name := range importStmt.Names {
						depModule.UsedExports[name.Value] = true
					}
				}
			}
		}
	}
}

// loadModule loads and parses a module, recursively loading its dependencies
func (b *Bundler) loadModule(filePath, moduleID string) error {
	// Skip if already loaded
	if _, exists := b.modules[moduleID]; exists {
		return nil
	}

	// Read and parse the file
	source, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	statements, err := b.parseFile(filePath)
	if err != nil {
		return err
	}

	// Extract imports and exports
	imports := b.extractImports(statements)
	exports := b.extractExports(statements)

	// Create module
	module := &Module{
		Path:       filePath,
		ModuleID:   moduleID,
		Source:     string(source),
		Statements: statements,
		Imports:    imports,
		Exports:    exports,
	}
	b.modules[moduleID] = module

	// Recursively load dependencies
	for _, importPath := range imports {
		depFile, depModuleID, err := b.resolveModule(importPath, filePath)
		if err != nil {
			return fmt.Errorf("in %s: %w", filePath, err)
		}
		if err := b.loadModule(depFile, depModuleID); err != nil {
			return err
		}
	}

	return nil
}

// parseFile parses a Lunar file and returns its statements
func (b *Bundler) parseFile(filePath string) ([]ast.Statement, error) {
	source, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	l := lexer.New(string(source))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in %s:\n%s", filePath, strings.Join(p.Errors(), "\n"))
	}

	return statements, nil
}

// extractImports extracts import module paths from statements
func (b *Bundler) extractImports(statements []ast.Statement) []string {
	var imports []string
	for _, stmt := range statements {
		if importStmt, ok := stmt.(*ast.ImportStatement); ok {
			imports = append(imports, importStmt.Module)
		}
	}
	return imports
}

// extractExports extracts exported names from statements
func (b *Bundler) extractExports(statements []ast.Statement) []string {
	var exports []string
	for _, stmt := range statements {
		if exportStmt, ok := stmt.(*ast.ExportStatement); ok {
			switch s := exportStmt.Statement.(type) {
			case *ast.VariableDeclaration:
				for _, name := range s.Names {
					exports = append(exports, name.Value)
				}
			case *ast.FunctionDeclaration:
				exports = append(exports, s.Name.Value)
			case *ast.ClassDeclaration:
				exports = append(exports, s.Name.Value)
			case *ast.EnumDeclaration:
				exports = append(exports, s.Name.Value)
			}
		}
	}
	return exports
}

// resolveModule resolves an import path to a file path and module ID
func (b *Bundler) resolveModule(importPath, fromFile string) (string, string, error) {
	fromDir := filepath.Dir(fromFile)

	// Apply path aliases from config
	resolvedPath := b.config.ResolvePath(importPath, fromDir)
	if resolvedPath != importPath {
		// Path was aliased, resolve from base directory
		// Ensure the path is relative
		if !strings.HasPrefix(resolvedPath, "./") && !strings.HasPrefix(resolvedPath, "../") {
			resolvedPath = "./" + resolvedPath
		}
		importPath = resolvedPath
		fromDir = b.baseDir
	}

	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		resolved := filepath.Join(fromDir, importPath)

		// Try different resolution strategies
		absPath, err := b.resolveFile(resolved)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve %s: %w", importPath, err)
		}

		// Create module ID relative to base dir
		relPath, err := filepath.Rel(b.baseDir, absPath)
		if err != nil {
			relPath = absPath
		}
		moduleID := "./" + strings.TrimSuffix(relPath, ".lunar")
		moduleID = strings.ReplaceAll(moduleID, "\\", "/") // Normalize path separators

		return absPath, moduleID, nil
	}

	// For non-relative imports (e.g., "lua" stdlib), don't bundle them
	// They'll be left as regular require() calls
	return "", importPath, fmt.Errorf("cannot bundle external module: %s", importPath)
}

// resolveFile tries to resolve a file path with various extensions and index files
func (b *Bundler) resolveFile(basePath string) (string, error) {
	// Try exact path with .lunar extension
	if !strings.HasSuffix(basePath, ".lunar") {
		lunarPath := basePath + ".lunar"
		if _, err := os.Stat(lunarPath); err == nil {
			return filepath.Abs(lunarPath)
		}
	} else {
		if _, err := os.Stat(basePath); err == nil {
			return filepath.Abs(basePath)
		}
	}

	// Try as directory with index.lunar
	indexPath := filepath.Join(basePath, "index.lunar")
	if _, err := os.Stat(indexPath); err == nil {
		return filepath.Abs(indexPath)
	}

	// Try without extension (in case it's already .lunar)
	if strings.HasSuffix(basePath, ".lunar") {
		if _, err := os.Stat(basePath); err == nil {
			return filepath.Abs(basePath)
		}
	}

	return "", fmt.Errorf("module not found: %s", basePath)
}

// topologicalSort sorts modules in dependency order
func (b *Bundler) topologicalSort() error {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var order []string
	var path []string // Track the current path for better error messages

	var visit func(moduleID string) error
	visit = func(moduleID string) error {
		if inStack[moduleID] {
			// Build circular dependency chain
			cycle := []string{moduleID}
			for i := len(path) - 1; i >= 0; i-- {
				cycle = append([]string{path[i]}, cycle...)
				if path[i] == moduleID {
					break
				}
			}
			return fmt.Errorf("circular dependency detected:\n  %s", strings.Join(cycle, " -> "))
		}
		if visited[moduleID] {
			return nil
		}

		inStack[moduleID] = true
		path = append(path, moduleID)
		module := b.modules[moduleID]
		if module != nil {
			for _, dep := range module.Imports {
				// Resolve to module ID
				if depModule := b.findModuleByImport(dep, module.Path); depModule != nil {
					if err := visit(depModule.ModuleID); err != nil {
						return err
					}
				}
			}
		}
		path = path[:len(path)-1]
		inStack[moduleID] = false
		visited[moduleID] = true
		order = append(order, moduleID)
		return nil
	}

	for moduleID := range b.modules {
		if err := visit(moduleID); err != nil {
			return err
		}
	}

	b.loadOrder = order
	return nil
}

// findModuleByImport finds a module by its import path from a given file
func (b *Bundler) findModuleByImport(importPath, fromFile string) *Module {
	_, moduleID, err := b.resolveModule(importPath, fromFile)
	if err != nil {
		return nil
	}
	return b.modules[moduleID]
}

// typeCheckAll type checks all modules together
func (b *Bundler) typeCheckAll() error {
	// Combine all statements
	var allStatements []ast.Statement
	allStatements = append(allStatements, b.declStmts...)

	for _, moduleID := range b.loadOrder {
		module := b.modules[moduleID]
		allStatements = append(allStatements, module.Statements...)
	}

	typeErrors := types.Check(allStatements)
	if len(typeErrors) > 0 {
		var errMsgs []string
		for _, err := range typeErrors {
			errMsgs = append(errMsgs, fmt.Sprintf("Line %d: %s", err.Line, err.Message))
		}
		return fmt.Errorf("type errors:\n%s", strings.Join(errMsgs, "\n"))
	}

	return nil
}

// discoverDeclarationFiles finds all .d.lunar files
func (b *Bundler) discoverDeclarationFiles() ([]string, error) {
	pattern := filepath.Join(b.baseDir, "*.d.lunar")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// generateBundle generates the bundled Lua code
func (b *Bundler) generateBundle(entryModuleID string) string {
	var output strings.Builder

	// Write bundle header with module system
	output.WriteString(`-- Lunar bundled output
local __modules = {}
local __loaded = {}

local function __require(name)
    if __loaded[name] then
        return __loaded[name]
    end
    if not __modules[name] then
        -- Fall back to standard require for external modules
        return require(name)
    end
    __loaded[name] = __modules[name]()
    return __loaded[name]
end

`)

	// Generate each module in dependency order
	for _, moduleID := range b.loadOrder {
		module := b.modules[moduleID]
		output.WriteString(b.generateModuleWrapper(module))
		output.WriteString("\n")
	}

	// Execute entry point
	output.WriteString(fmt.Sprintf("-- Execute entry point\n__require(\"%s\")\n", entryModuleID))

	return output.String()
}

// generateModuleWrapper generates a module wrapped in a function
func (b *Bundler) generateModuleWrapper(module *Module) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("__modules[\"%s\"] = function()\n", module.ModuleID))

	// Generate module code
	gen := codegen.New()

	// Transform import statements to use __require
	transformedStatements := b.transformImports(module.Statements, module.Path)

	// Generate code for non-import statements
	var moduleStatements []ast.Statement
	for _, stmt := range transformedStatements {
		// Skip import statements - they're handled specially
		if _, ok := stmt.(*ast.ImportStatement); ok {
			// Generate __require call instead
			importStmt := stmt.(*ast.ImportStatement)
			depModule := b.findModuleByImport(importStmt.Module, module.Path)

			if depModule != nil {
				// Internal module - use __require
				if importStmt.IsWildcard {
					parts := strings.Split(importStmt.Module, "/")
					varName := strings.TrimSuffix(parts[len(parts)-1], ".lunar")
					output.WriteString(fmt.Sprintf("    local %s = __require(\"%s\")\n", varName, depModule.ModuleID))
				} else {
					tempVar := "_" + strings.ReplaceAll(importStmt.Module, "/", "_")
					tempVar = strings.ReplaceAll(tempVar, ".", "_")
					tempVar = strings.ReplaceAll(tempVar, "-", "_")
					output.WriteString(fmt.Sprintf("    local %s = __require(\"%s\")\n", tempVar, depModule.ModuleID))
					for _, name := range importStmt.Names {
						output.WriteString(fmt.Sprintf("    local %s = %s.%s\n", name.Value, tempVar, name.Value))
					}
				}
			} else {
				// External module - use regular require
				if importStmt.IsWildcard {
					parts := strings.Split(importStmt.Module, "/")
					varName := strings.TrimSuffix(parts[len(parts)-1], ".lunar")
					output.WriteString(fmt.Sprintf("    local %s = require(\"%s\")\n", varName, importStmt.Module))
				} else {
					tempVar := "_" + strings.ReplaceAll(importStmt.Module, "/", "_")
					tempVar = strings.ReplaceAll(tempVar, ".", "_")
					output.WriteString(fmt.Sprintf("    local %s = require(\"%s\")\n", tempVar, importStmt.Module))
					for _, name := range importStmt.Names {
						output.WriteString(fmt.Sprintf("    local %s = %s.%s\n", name.Value, tempVar, name.Value))
					}
				}
			}
			continue
		}
		// Tree shaking: skip unused exports
		if b.config.CompilerOptions.TreeShake && module.UsedExports != nil {
			if exportStmt, ok := stmt.(*ast.ExportStatement); ok {
				var exportName string
				switch s := exportStmt.Statement.(type) {
				case *ast.VariableDeclaration:
					if len(s.Names) > 0 {
						exportName = s.Names[0].Value
					}
				case *ast.FunctionDeclaration:
					exportName = s.Name.Value
				case *ast.ClassDeclaration:
					exportName = s.Name.Value
				case *ast.EnumDeclaration:
					exportName = s.Name.Value
				}
				if exportName != "" && !module.UsedExports[exportName] {
					// Skip unused export
					continue
				}
			}
		}
		moduleStatements = append(moduleStatements, stmt)
	}

	// Generate the rest of the module code
	moduleCode := gen.Generate(moduleStatements)

	// Indent the module code
	lines := strings.Split(moduleCode, "\n")
	for _, line := range lines {
		if line != "" {
			output.WriteString("    " + line + "\n")
		}
	}

	// Add return {} if no exports (codegen already handles exports)
	if len(module.Exports) == 0 {
		output.WriteString("    return {}\n")
	}

	output.WriteString("end\n")

	return output.String()
}

// transformImports transforms import statements (placeholder for future use)
func (b *Bundler) transformImports(statements []ast.Statement, fromFile string) []ast.Statement {
	return statements
}

// GetAllFiles returns all file paths that were bundled
func (b *Bundler) GetAllFiles() []string {
	var files []string
	for _, module := range b.modules {
		files = append(files, module.Path)
	}
	return files
}
