package lsp

import (
	"io/fs"
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
}

// NewDeclarationManager creates a new declaration manager
func NewDeclarationManager() *DeclarationManager {
	return &DeclarationManager{
		declarations: make(map[string]*types.Environment),
	}
}

// SetRootPath sets the workspace root path and scans for declarations
func (dm *DeclarationManager) SetRootPath(rootPath string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.rootPath = rootPath
	dm.scanDeclarations()
}

// scanDeclarations scans the workspace for .d.lunar files
func (dm *DeclarationManager) scanDeclarations() {
	if dm.rootPath == "" {
		return
	}

	// Scan for .d.lunar files
	filepath.WalkDir(dm.rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on error
		}

		// Skip node_modules and hidden directories
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Process .d.lunar files
		if strings.HasSuffix(path, ".d.lunar") {
			dm.loadDeclarationFile(path)
		}

		return nil
	})
}

// loadDeclarationFile loads a declaration file and extracts type information
func (dm *DeclarationManager) loadDeclarationFile(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Parse the declaration file
	l := lexer.New(string(content))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return
	}

	// Type check to extract declarations
	checker := types.NewChecker()
	checker.Check(statements)

	// Store the environment
	uri := "file://" + path
	dm.declarations[uri] = checker.GetEnv()
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
