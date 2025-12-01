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

func (dm *DeclarationManager) SetLogger(logger *log.Logger) {
	dm.logger = logger
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

	if dm.logger != nil {
		dm.logger.Printf("Scanning for .d.lunar files in: %s", dm.rootPath)
	}

	fileCount := 0
	// Scan for .d.lunar files recursively
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
			if dm.logger != nil {
				dm.logger.Printf("Found declaration file: %s", path)
			}
			dm.loadDeclarationFile(path)
			fileCount++
		}

		return nil
	})

	if dm.logger != nil {
		dm.logger.Printf("Loaded %d declaration file(s)", fileCount)
	}
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
	stdlibPath := getStdlibPathForLSP()
	checker := types.NewChecker(stdlibPath)
	checker.Check(statements)

	// Store the environment
	uri := pathToURI(path)
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

// GetAllEnvironments returns all declaration file environments
func (dm *DeclarationManager) GetAllEnvironments() []*types.Environment {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	envs := make([]*types.Environment, 0, len(dm.declarations))
	for _, env := range dm.declarations {
		envs = append(envs, env)
	}

	return envs
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

// pathToURI converts a filesystem path to a file:// URI
// Handles both Windows (C:\path) and Unix (/path) paths
func pathToURI(path string) string {
	// Convert backslashes to forward slashes
	path = filepath.ToSlash(path)

	// On Windows, paths start with a drive letter (e.g., C:/Users/...)
	// We need to prepend with file:/// (3 slashes)
	// On Unix, paths start with / (e.g., /home/...)
	// We need to prepend with file:// (2 slashes to make file:///)
	if len(path) > 1 && path[1] == ':' {
		// Windows path with drive letter
		return "file:///" + path
	}

	// Unix path
	return "file://" + path
}
