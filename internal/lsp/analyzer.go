package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
)

// ProjectAnalyzer performs project-wide analysis
type ProjectAnalyzer struct {
	rootPath     string
	files        map[string]*FileInfo
	dependencies map[string][]string
	symbols      map[string]*SymbolInfo
	mu           sync.RWMutex
}

// FileInfo contains information about a source file
type FileInfo struct {
	URI          string
	Content      string
	AST          []ast.Statement
	Symbols      []SymbolInfo
	Dependencies []string
	Exports      []string
	Metrics      FileMetrics
	LastModified time.Time
}

// SymbolInfo contains information about a symbol
type SymbolInfo struct {
	Name       string
	Kind       SymbolKind
	Type       string
	Location   Location
	References []Location
	Scope      string
	Exported   bool
}

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymbolKindFunction SymbolKind = iota
	SymbolKindVariable
	SymbolKindClass
	SymbolKindInterface
	SymbolKindType
	SymbolKindConstant
	SymbolKindParameter
	SymbolKindProperty
)

// FileMetrics contains metrics for a file
type FileMetrics struct {
	LOC        int // Lines of code
	Functions  int
	Classes    int
	Interfaces int
	Complexity int // Cyclomatic complexity
}

// NewProjectAnalyzer creates a new project analyzer
func NewProjectAnalyzer(rootPath string) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		rootPath:     rootPath,
		files:        make(map[string]*FileInfo),
		dependencies: make(map[string][]string),
		symbols:      make(map[string]*SymbolInfo),
	}
}

// Analyze performs full project analysis
func (a *ProjectAnalyzer) Analyze(params ProjectAnalysisParams) (*ProjectAnalysisResult, error) {
	startTime := time.Now()

	// Scan project files
	files, err := a.scanFiles(params.RootURI, params.IncludeFiles, params.ExcludeFiles)
	if err != nil {
		return nil, err
	}

	result := &ProjectAnalysisResult{
		Files:        make([]FileAnalysis, 0),
		Dependencies: make([]Dependency, 0),
		Issues:       make([]ProjectIssue, 0),
	}

	// Analyze each file
	var wg sync.WaitGroup
	fileChan := make(chan FileAnalysis, len(files))
	errorCount := 0
	warningCount := 0
	infoCount := 0

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()

			analysis, err := a.analyzeFile(f, params.Options)
			if err != nil {
				return
			}

			// Count issues
			for _, issue := range analysis.Issues {
				switch issue.Severity {
				case SeverityError:
					errorCount++
				case SeverityWarning:
					warningCount++
				case SeverityInformation:
					infoCount++
				}
			}

			fileChan <- analysis
		}(file)
	}

	// Wait for all files to be analyzed
	go func() {
		wg.Wait()
		close(fileChan)
	}()

	// Collect results
	for analysis := range fileChan {
		result.Files = append(result.Files, analysis)
	}

	// Build dependency graph if requested
	if params.Options.IncludeDependencies {
		result.Dependencies = a.buildDependencyGraph(result.Files)
	}

	// Calculate metrics if requested
	if params.Options.IncludeMetrics {
		result.Metrics = a.calculateMetrics(result.Files, result.Dependencies)
	}

	// Detect issues if requested
	if params.Options.IncludeIssues {
		result.Issues = a.detectIssues(result.Files, result.Dependencies)
	}

	// Build summary
	result.Summary = AnalysisSummary{
		Status:       a.determineStatus(errorCount, warningCount),
		Duration:     time.Since(startTime).Milliseconds(),
		FilesScanned: len(files),
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		InfoCount:    infoCount,
	}

	return result, nil
}

// scanFiles scans for Lunar source files
func (a *ProjectAnalyzer) scanFiles(rootURI string, include, exclude []string) ([]string, error) {
	files := make([]string, 0)

	err := filepath.Walk(rootURI, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file should be included
		if !strings.HasSuffix(path, ".lunar") {
			return nil
		}

		// Check exclusions
		for _, pattern := range exclude {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// analyzeFile analyzes a single file
func (a *ProjectAnalyzer) analyzeFile(filePath string, options ProjectAnalysisOptions) (FileAnalysis, error) {
	analysis := FileAnalysis{
		URI:          "file://" + filePath,
		Dependencies: make([]string, 0),
		Exports:      make([]string, 0),
		Issues:       make([]Diagnostic, 0),
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return analysis, err
	}

	// Parse file
	l := lexer.New(string(content))
	p := parser.New(l)
	statements := p.Parse()

	// Count symbols
	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *ast.FunctionDeclaration:
			analysis.Functions++
		case *ast.InterfaceDeclaration:
			analysis.Interfaces++
			analysis.Exports = append(analysis.Exports, s.Name.Value)
		case *ast.ClassDeclaration:
			analysis.Classes++
			analysis.Exports = append(analysis.Exports, s.Name.Value)
		}
	}

	// Calculate LOC
	analysis.LOC = strings.Count(string(content), "\n") + 1

	// Calculate complexity if requested
	if options.IncludeComplexity {
		analysis.Complexity = a.calculateComplexity(statements)
	}

	// Extract dependencies
	analysis.Dependencies = a.extractDependencies(statements)

	// Store file info
	a.mu.Lock()
	a.files[filePath] = &FileInfo{
		URI:          analysis.URI,
		Content:      string(content),
		AST:          statements,
		Dependencies: analysis.Dependencies,
		Exports:      analysis.Exports,
		LastModified: time.Now(),
	}
	a.mu.Unlock()

	return analysis, nil
}

// extractDependencies extracts import/require dependencies
func (a *ProjectAnalyzer) extractDependencies(statements []ast.Statement) []string {
	deps := make([]string, 0)

	// Look for import statements and require calls
	for range statements {
		// This would need to be extended based on actual AST structure
		// For now, return empty list
	}

	return deps
}

// calculateComplexity calculates cyclomatic complexity
func (a *ProjectAnalyzer) calculateComplexity(statements []ast.Statement) int {
	complexity := 0

	var walkStatements func([]ast.Statement)
	walkStatements = func(stmts []ast.Statement) {
		for _, stmt := range stmts {
			switch s := stmt.(type) {
			case *ast.IfStatement:
				complexity++
				walkStatements(s.Consequence.Statements)
				if s.Alternative != nil {
					walkStatements(s.Alternative.Statements)
				}
			case *ast.WhileStatement:
				complexity++
				walkStatements(s.Body.Statements)
			case *ast.ForStatement:
				complexity++
				walkStatements(s.Body.Statements)
			case *ast.FunctionDeclaration:
				complexity++
				walkStatements(s.Body.Statements)
			}
		}
	}

	walkStatements(statements)
	return complexity
}

// buildDependencyGraph builds the dependency graph
func (a *ProjectAnalyzer) buildDependencyGraph(files []FileAnalysis) []Dependency {
	dependencies := make([]Dependency, 0)

	for _, file := range files {
		for _, dep := range file.Dependencies {
			dependencies = append(dependencies, Dependency{
				Source: file.URI,
				Target: dep,
				Type:   "import",
			})
		}
	}

	return dependencies
}

// calculateMetrics calculates project-wide metrics
func (a *ProjectAnalyzer) calculateMetrics(files []FileAnalysis, deps []Dependency) ProjectMetrics {
	metrics := ProjectMetrics{
		TotalFiles: len(files),
	}

	totalComplexity := 0

	for _, file := range files {
		metrics.TotalLOC += file.LOC
		metrics.TotalFunctions += file.Functions
		metrics.TotalInterfaces += file.Interfaces
		metrics.TotalClasses += file.Classes
		totalComplexity += file.Complexity

		if file.Complexity > metrics.MaxComplexity {
			metrics.MaxComplexity = file.Complexity
		}
	}

	if metrics.TotalFunctions > 0 {
		metrics.AverageComplexity = float64(totalComplexity) / float64(metrics.TotalFunctions)
	}

	// Detect circular dependencies
	metrics.CircularDeps = a.detectCircularDependencies(deps)

	return metrics
}

// detectCircularDependencies detects circular dependencies
func (a *ProjectAnalyzer) detectCircularDependencies(deps []Dependency) int {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, dep := range deps {
		graph[dep.Source] = append(graph[dep.Source], dep.Target)
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	cycles := 0

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					cycles++
					return true
				}
			} else if recStack[neighbor] {
				cycles++
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// detectIssues detects project-wide issues
func (a *ProjectAnalyzer) detectIssues(files []FileAnalysis, deps []Dependency) []ProjectIssue {
	issues := make([]ProjectIssue, 0)

	// Check for circular dependencies
	if cycles := a.detectCircularDependencies(deps); cycles > 0 {
		issues = append(issues, ProjectIssue{
			Severity:    "warning",
			Category:    "architecture",
			Message:     fmt.Sprintf("Found %d circular dependencies", cycles),
			Suggestion:  "Refactor code to remove circular dependencies",
			AutoFixable: false,
		})
	}

	// Check for files with high complexity
	for _, file := range files {
		if file.Complexity > 20 {
			issues = append(issues, ProjectIssue{
				Severity: "warning",
				Category: "complexity",
				Message:  fmt.Sprintf("High complexity in %s (complexity: %d)", file.URI, file.Complexity),
				Locations: []Location{
					{URI: file.URI},
				},
				Suggestion:  "Consider refactoring to reduce complexity",
				AutoFixable: false,
			})
		}
	}

	// Check for files with no exports (potential dead code)
	for _, file := range files {
		if len(file.Exports) == 0 && file.Functions > 0 {
			issues = append(issues, ProjectIssue{
				Severity: "info",
				Category: "dead-code",
				Message:  fmt.Sprintf("File %s has no exports", file.URI),
				Locations: []Location{
					{URI: file.URI},
				},
				Suggestion:  "Consider removing unused file or adding exports",
				AutoFixable: false,
			})
		}
	}

	return issues
}

// determineStatus determines overall project status
func (a *ProjectAnalyzer) determineStatus(errors, warnings int) string {
	if errors > 0 {
		return "error"
	}
	if warnings > 0 {
		return "warning"
	}
	return "success"
}

// BuildDependencyGraph builds a visual dependency graph
func (a *ProjectAnalyzer) BuildDependencyGraph(params DependencyGraphParams) (*DependencyGraphResult, error) {
	result := &DependencyGraphResult{
		Nodes:       make([]DependencyNode, 0),
		Edges:       make([]DependencyEdge, 0),
		Circular:    make([][]string, 0),
		EntryPoints: make([]string, 0),
		Orphans:     make([]string, 0),
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Build nodes
	for path, file := range a.files {
		node := DependencyNode{
			ID:         path,
			URI:        file.URI,
			Type:       "file",
			Exports:    file.Exports,
			Imports:    len(file.Dependencies),
			ImportedBy: 0,
		}

		result.Nodes = append(result.Nodes, node)
	}

	// Build edges
	for path, file := range a.files {
		for _, dep := range file.Dependencies {
			edge := DependencyEdge{
				From: path,
				To:   dep,
			}
			result.Edges = append(result.Edges, edge)

			// Update imported-by count
			for i := range result.Nodes {
				if result.Nodes[i].ID == dep {
					result.Nodes[i].ImportedBy++
					break
				}
			}
		}
	}

	// Find entry points (files with no importers)
	for _, node := range result.Nodes {
		if node.ImportedBy == 0 {
			result.EntryPoints = append(result.EntryPoints, node.ID)
		}
		if node.Imports == 0 && node.ImportedBy == 0 {
			result.Orphans = append(result.Orphans, node.ID)
		}
	}

	return result, nil
}

// AnalyzeTypeHierarchy analyzes type hierarchy at a position
func (a *ProjectAnalyzer) AnalyzeTypeHierarchy(params TypeHierarchyParams) (*TypeHierarchyResult, error) {
	result := &TypeHierarchyResult{
		Parents:  make([]TypeNode, 0),
		Children: make([]TypeNode, 0),
	}

	// Implementation would involve:
	// 1. Find type at position
	// 2. Walk AST to find parent types (extends/implements)
	// 3. Find child types that extend/implement this type
	// 4. Build hierarchy tree

	return result, nil
}

// DetectDeadCode detects unused code
func (a *ProjectAnalyzer) DetectDeadCode(params DeadCodeParams) (*DeadCodeResult, error) {
	result := &DeadCodeResult{
		DeadFunctions: make([]DeadCodeEntry, 0),
		DeadVariables: make([]DeadCodeEntry, 0),
		DeadImports:   make([]DeadCodeEntry, 0),
		UnusedParams:  make([]DeadCodeEntry, 0),
	}

	// Implementation would involve:
	// 1. Build symbol table with all definitions
	// 2. Build usage map with all references
	// 3. Find symbols with no references
	// 4. Account for exported symbols (can't be dead if exported)

	return result, nil
}

// AnalyzeUsage analyzes symbol usage
func (a *ProjectAnalyzer) AnalyzeUsage(params UsageAnalysisParams) (*UsageAnalysisResult, error) {
	result := &UsageAnalysisResult{
		Reads:    make([]Location, 0),
		Writes:   make([]Location, 0),
		Calls:    make([]Location, 0),
		HotSpots: make([]UsageHotSpot, 0),
	}

	// Implementation would involve:
	// 1. Find symbol at position
	// 2. Scan all files for references
	// 3. Classify references as read/write/call
	// 4. Identify hot spots (frequently used locations)

	return result, nil
}
