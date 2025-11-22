package lsp

import (
	"io/fs"
	"log"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DeclarationManager manages global declaration files
type DeclarationManager struct {
	declarations map[string]*types.Environment // URI -> environment
	mu           sync.RWMutex
	rootPath     string
	logger       *log.Logger
}

// NewDeclarationManager creates a new declaration manager
func NewDeclarationManager() *DeclarationManager {
	return &DeclarationManager{
		declarations: make(map[string]*types.Environment),
		logger:       log.New(os.Stderr, "[lunar-lsp-decl] ", log.LstdFlags),
	}
}

// SetRootPath sets the workspace root path and scans for declarations
func (dm *DeclarationManager) SetRootPath(rootPath string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.rootPath = rootPath
	dm.logger.Printf("Setting root path: %s", rootPath)
	dm.scanDeclarations()
}

// scanDeclarations scans the workspace for .d.lunar files
func (dm *DeclarationManager) scanDeclarations() {
	if dm.rootPath == "" {
		dm.logger.Println("Root path is empty, skipping scan")
		return
	}

	dm.logger.Printf("Scanning for .d.lunar files in: %s", dm.rootPath)
	filesFound := 0

	// Scan for .d.lunar files
	filepath.WalkDir(dm.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			dm.logger.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue on error
		}

		// Skip node_modules and hidden directories
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
				dm.logger.Printf("Skipping directory: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Process .d.lunar files
		if strings.HasSuffix(path, ".d.lunar") {
			dm.logger.Printf("Found .d.lunar file: %s", path)
			filesFound++
			dm.loadDeclarationFile(path)
		}

		return nil
	})

	dm.logger.Printf("Scan complete. Found %d .d.lunar files", filesFound)
}

// loadDeclarationFile loads a declaration file and extracts type information
func (dm *DeclarationManager) loadDeclarationFile(path string) {
	dm.logger.Printf("Loading declaration file: %s", path)

	content, err := os.ReadFile(path)
	if err != nil {
		dm.logger.Printf("Failed to read file %s: %v", path, err)
		return
	}

	// Parse the declaration file
	l := lexer.New(string(content))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		dm.logger.Printf("Parse errors in %s:", path)
		for _, err := range p.Errors() {
			dm.logger.Printf("  %s", err)
		}
		return
	}

	// Type check to extract declarations
	checker := types.NewChecker()
	typeErrors := checker.Check(statements)

	if len(typeErrors) > 0 {
		dm.logger.Printf("Type errors in %s:", path)
		for _, err := range typeErrors {
			dm.logger.Printf("  %s", err.Message)
		}
		// Continue loading even with type errors
	}

	// Store the environment
	uri := "file://" + path
	env := checker.GetEnv()
	dm.declarations[uri] = env

	// Log what was loaded
	symbols := env.GetAll()
	dm.logger.Printf("Loaded %d symbols from %s", len(symbols), path)
	for name := range symbols {
		dm.logger.Printf("  - %s", name)
	}
}

// GetDeclaredType looks up a declared constant or type in all declaration files
func (dm *DeclarationManager) GetDeclaredType(name string) (types.Type, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Search all declaration environments
	for _, env := range dm.declarations {
		if typ, ok := env.Get(name); ok {
			return typ, true
		}
	}

	return nil, false
}

// GetAllDeclaredSymbols returns all declared symbols from all declaration files
func (dm *DeclarationManager) GetAllDeclaredSymbols() map[string]types.Type {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	symbols := make(map[string]types.Type)

	// Collect all symbols from all declaration files
	for _, env := range dm.declarations {
		for name, typ := range env.GetAll() {
			symbols[name] = typ
		}
	}

	return symbols
}

// GetMembersOf returns the members of a declared interface or class
func (dm *DeclarationManager) GetMembersOf(typeName string) map[string]types.Type {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	members := make(map[string]types.Type)

	// Find the type in declarations
	for _, env := range dm.declarations {
		if typ, ok := env.Get(typeName); ok {
			// Extract members based on type
			switch t := typ.(type) {
			case *types.InterfaceType:
				// Add interface properties and methods
				for name, propType := range t.Properties {
					members[name] = propType
				}
				for name, methodType := range t.Methods {
					members[name] = methodType
				}
			case *types.ClassType:
				// Add class properties and methods
				for name, propType := range t.Properties {
					members[name] = propType
				}
				for name, methodType := range t.Methods {
					members[name] = methodType
				}
			}
			break
		}
	}

	return members
}

// Refresh rescans all declaration files
func (dm *DeclarationManager) Refresh() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.declarations = make(map[string]*types.Environment)
	dm.scanDeclarations()
}
