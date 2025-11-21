package lsp

// Lunar-specific LSP protocol extensions
// These provide additional functionality beyond the standard LSP

// =======================
// Custom Method Names
// =======================

const (
	// Type System Extensions
	MethodLunarTypeCheck       = "lunar/typeCheck"       // Full type checking
	MethodLunarInferType       = "lunar/inferType"       // Type inference
	MethodLunarTypeHierarchy   = "lunar/typeHierarchy"   // Show type hierarchy
	MethodLunarExpandType      = "lunar/expandType"      // Expand complex types

	// Code Intelligence
	MethodLunarSmartCompletion = "lunar/smartCompletion" // Context-aware completion
	MethodLunarCallHierarchy   = "lunar/callHierarchy"   // Function call graph
	MethodLunarUsageAnalysis   = "lunar/usageAnalysis"   // Variable usage analysis
	MethodLunarDeadCode        = "lunar/deadCode"        // Dead code detection

	// Project-Wide Analysis
	MethodLunarProjectAnalysis = "lunar/projectAnalysis" // Full project analysis
	MethodLunarDependencyGraph = "lunar/dependencyGraph" // Module dependencies
	MethodLunarImportAnalysis  = "lunar/importAnalysis"  // Import analysis
	MethodLunarModuleGraph     = "lunar/moduleGraph"     // Module relationships

	// Refactoring
	MethodLunarExtractFunction = "lunar/extractFunction" // Extract to function
	MethodLunarExtractVariable = "lunar/extractVariable" // Extract to variable
	MethodLunarInlineFunction  = "lunar/inlineFunction"  // Inline function
	MethodLunarRenameSymbol    = "lunar/renameSymbol"    // Smart rename

	// Migration & Transformation
	MethodLunarMigrateLua      = "lunar/migrateLua"      // Migrate Lua code
	MethodLunarAddTypes        = "lunar/addTypes"        // Add type annotations
	MethodLunarOptimize        = "lunar/optimize"        // Code optimization
	MethodLunarGenerateJIT     = "lunar/generateJIT"     // Generate JIT hints

	// Documentation
	MethodLunarGenerateDocs    = "lunar/generateDocs"    // Generate documentation
	MethodLunarTypeSignature   = "lunar/typeSignature"   // Get full type signature
	MethodLunarExamples        = "lunar/examples"        // Usage examples
)

// =======================
// Type System Extensions
// =======================

// TypeCheckParams represents full type check parameters
type TypeCheckParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      TypeCheckOptions       `json:"options,omitempty"`
}

// TypeCheckOptions configures type checking behavior
type TypeCheckOptions struct {
	Strict          bool `json:"strict"`           // Strict type checking
	NoImplicitAny   bool `json:"noImplicitAny"`    // Disallow implicit any
	StrictNullCheck bool `json:"strictNullCheck"`  // Strict null checking
	UnusedVars      bool `json:"unusedVars"`       // Check for unused variables
}

// TypeCheckResult represents type check results
type TypeCheckResult struct {
	Diagnostics []Diagnostic      `json:"diagnostics"`
	TypeInfo    []TypeInformation `json:"typeInfo"`
	Metrics     TypeCheckMetrics  `json:"metrics"`
}

// TypeInformation represents type information for a symbol
type TypeInformation struct {
	Range      Range  `json:"range"`
	Symbol     string `json:"symbol"`
	Type       string `json:"type"`
	Inferred   bool   `json:"inferred"`
	Complexity int    `json:"complexity,omitempty"`
}

// TypeCheckMetrics represents type checking metrics
type TypeCheckMetrics struct {
	TotalSymbols    int     `json:"totalSymbols"`
	TypedSymbols    int     `json:"typedSymbols"`
	InferredSymbols int     `json:"inferredSymbols"`
	AnyTypes        int     `json:"anyTypes"`
	Coverage        float64 `json:"coverage"` // Percentage
}

// InferTypeParams represents type inference parameters
type InferTypeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Deep         bool                   `json:"deep,omitempty"` // Deep inference
}

// InferTypeResult represents inferred type information
type InferTypeResult struct {
	Type         string            `json:"type"`
	Confidence   float64           `json:"confidence"` // 0-1
	Alternatives []TypeAlternative `json:"alternatives,omitempty"`
	Reasoning    string            `json:"reasoning,omitempty"`
}

// TypeAlternative represents an alternative type interpretation
type TypeAlternative struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// TypeHierarchyParams represents type hierarchy parameters
type TypeHierarchyParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Direction    string                 `json:"direction"` // "up" | "down" | "both"
}

// TypeHierarchyResult represents type hierarchy
type TypeHierarchyResult struct {
	Root     TypeNode   `json:"root"`
	Parents  []TypeNode `json:"parents,omitempty"`
	Children []TypeNode `json:"children,omitempty"`
}

// TypeNode represents a node in the type hierarchy
type TypeNode struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"` // "interface" | "class" | "type" | "enum"
	Location   Location   `json:"location"`
	Members    []TypeNode `json:"members,omitempty"`
	Extends    []string   `json:"extends,omitempty"`
	Implements []string   `json:"implements,omitempty"`
}

// =======================
// Project-Wide Analysis
// =======================

// ProjectAnalysisParams represents project analysis parameters
type ProjectAnalysisParams struct {
	RootURI       string                  `json:"rootUri"`
	Options       ProjectAnalysisOptions  `json:"options,omitempty"`
	IncludeFiles  []string                `json:"includeFiles,omitempty"`
	ExcludeFiles  []string                `json:"excludeFiles,omitempty"`
}

// ProjectAnalysisOptions configures project analysis
type ProjectAnalysisOptions struct {
	IncludeDependencies bool `json:"includeDependencies"` // Analyze dependencies
	IncludeMetrics      bool `json:"includeMetrics"`      // Calculate metrics
	IncludeIssues       bool `json:"includeIssues"`       // Find issues
	IncludeComplexity   bool `json:"includeComplexity"`   // Complexity analysis
	MaxDepth            int  `json:"maxDepth,omitempty"`  // Max dependency depth
}

// ProjectAnalysisResult represents project analysis results
type ProjectAnalysisResult struct {
	Files        []FileAnalysis    `json:"files"`
	Dependencies []Dependency      `json:"dependencies,omitempty"`
	Metrics      ProjectMetrics    `json:"metrics,omitempty"`
	Issues       []ProjectIssue    `json:"issues,omitempty"`
	Summary      AnalysisSummary   `json:"summary"`
}

// FileAnalysis represents analysis of a single file
type FileAnalysis struct {
	URI          string             `json:"uri"`
	Symbols      int                `json:"symbols"`
	Functions    int                `json:"functions"`
	Interfaces   int                `json:"interfaces"`
	Classes      int                `json:"classes"`
	LOC          int                `json:"loc"`          // Lines of code
	Complexity   int                `json:"complexity"`   // Cyclomatic complexity
	Dependencies []string           `json:"dependencies"` // Imported modules
	Exports      []string           `json:"exports"`      // Exported symbols
	Issues       []Diagnostic       `json:"issues,omitempty"`
}

// Dependency represents a module dependency
type Dependency struct {
	Source   string   `json:"source"`   // Source file/module
	Target   string   `json:"target"`   // Target file/module
	Type     string   `json:"type"`     // "import" | "require" | "type"
	Symbols  []string `json:"symbols"`  // Imported symbols
	Location Location `json:"location"` // Import location
}

// ProjectMetrics represents project-level metrics
type ProjectMetrics struct {
	TotalFiles          int     `json:"totalFiles"`
	TotalLOC            int     `json:"totalLOC"`
	TotalFunctions      int     `json:"totalFunctions"`
	TotalInterfaces     int     `json:"totalInterfaces"`
	TotalClasses        int     `json:"totalClasses"`
	AverageComplexity   float64 `json:"averageComplexity"`
	MaxComplexity       int     `json:"maxComplexity"`
	TypeCoverage        float64 `json:"typeCoverage"`        // Percentage
	DependencyDepth     int     `json:"dependencyDepth"`     // Max depth
	CircularDeps        int     `json:"circularDeps"`        // Circular dependencies
}

// ProjectIssue represents a project-level issue
type ProjectIssue struct {
	Severity    string     `json:"severity"`    // "error" | "warning" | "info"
	Category    string     `json:"category"`    // "type" | "style" | "performance" | etc
	Message     string     `json:"message"`
	Locations   []Location `json:"locations"`   // Affected locations
	Suggestion  string     `json:"suggestion,omitempty"`
	AutoFixable bool       `json:"autoFixable"`
}

// AnalysisSummary represents analysis summary
type AnalysisSummary struct {
	Status       string `json:"status"` // "success" | "warning" | "error"
	Duration     int64  `json:"duration"` // milliseconds
	FilesScanned int    `json:"filesScanned"`
	ErrorCount   int    `json:"errorCount"`
	WarningCount int    `json:"warningCount"`
	InfoCount    int    `json:"infoCount"`
}

// DependencyGraphParams represents dependency graph parameters
type DependencyGraphParams struct {
	RootURI     string   `json:"rootUri"`
	StartModule string   `json:"startModule,omitempty"` // Starting module
	MaxDepth    int      `json:"maxDepth,omitempty"`
	Filters     []string `json:"filters,omitempty"` // Filter patterns
}

// DependencyGraphResult represents dependency graph
type DependencyGraphResult struct {
	Nodes         []DependencyNode `json:"nodes"`
	Edges         []DependencyEdge `json:"edges"`
	Circular      [][]string       `json:"circular,omitempty"` // Circular dependency chains
	EntryPoints   []string         `json:"entryPoints"`
	Orphans       []string         `json:"orphans"` // Files with no dependencies
}

// DependencyNode represents a node in dependency graph
type DependencyNode struct {
	ID          string   `json:"id"`
	URI         string   `json:"uri"`
	Type        string   `json:"type"` // "module" | "file"
	Exports     []string `json:"exports"`
	Imports     int      `json:"imports"`  // Number of imports
	ImportedBy  int      `json:"importedBy"` // Number of importers
}

// DependencyEdge represents an edge in dependency graph
type DependencyEdge struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Symbols []string `json:"symbols,omitempty"`
	Weight  int      `json:"weight,omitempty"` // Usage frequency
}

// =======================
// Smart Completion
// =======================

// SmartCompletionParams represents smart completion parameters
type SmartCompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      SmartCompletionContext `json:"context"`
}

// SmartCompletionContext represents smart completion context
type SmartCompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
	ExpectedType     string `json:"expectedType,omitempty"` // Expected return type
	InFunction       bool   `json:"inFunction"`
	InClass          bool   `json:"inClass"`
	InInterface      bool   `json:"inInterface"`
}

// SmartCompletionResult represents smart completion results
type SmartCompletionResult struct {
	Items       []SmartCompletionItem `json:"items"`
	Suggestions []CodeSuggestion      `json:"suggestions,omitempty"`
}

// SmartCompletionItem extends CompletionItem with additional info
type SmartCompletionItem struct {
	CompletionItem
	Relevance   float64  `json:"relevance"`   // 0-1 relevance score
	TypeMatch   bool     `json:"typeMatch"`   // Matches expected type
	Imports     []string `json:"imports,omitempty"` // Required imports
	Snippet     string   `json:"snippet,omitempty"` // Code snippet
	Examples    []string `json:"examples,omitempty"` // Usage examples
}

// CodeSuggestion represents a code suggestion
type CodeSuggestion struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Edit        WorkspaceEdit  `json:"edit"`
	Priority    int            `json:"priority"` // Higher = more important
}

// =======================
// Code Analysis
// =======================

// CallHierarchyParams represents call hierarchy parameters
type CallHierarchyParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Direction    string                 `json:"direction"` // "incoming" | "outgoing" | "both"
	Depth        int                    `json:"depth,omitempty"` // Max depth
}

// CallHierarchyResult represents call hierarchy
type CallHierarchyResult struct {
	Root     CallNode   `json:"root"`
	Incoming []CallNode `json:"incoming,omitempty"` // Functions that call this
	Outgoing []CallNode `json:"outgoing,omitempty"` // Functions called by this
}

// CallNode represents a node in call hierarchy
type CallNode struct {
	Name      string     `json:"name"`
	Kind      string     `json:"kind"` // "function" | "method"
	Signature string     `json:"signature"`
	Location  Location   `json:"location"`
	Calls     int        `json:"calls"` // Number of calls
	Children  []CallNode `json:"children,omitempty"`
}

// UsageAnalysisParams represents usage analysis parameters
type UsageAnalysisParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	IncludeWrite bool                   `json:"includeWrite"` // Include write refs
	IncludeRead  bool                   `json:"includeRead"`  // Include read refs
}

// UsageAnalysisResult represents usage analysis results
type UsageAnalysisResult struct {
	Symbol      string          `json:"symbol"`
	Reads       []Location      `json:"reads"`
	Writes      []Location      `json:"writes"`
	Calls       []Location      `json:"calls,omitempty"`
	Total       int             `json:"total"`
	HotSpots    []UsageHotSpot  `json:"hotSpots,omitempty"`
}

// UsageHotSpot represents a frequently used location
type UsageHotSpot struct {
	Location Location `json:"location"`
	Count    int      `json:"count"`
	Context  string   `json:"context"`
}

// DeadCodeParams represents dead code detection parameters
type DeadCodeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Aggressive   bool                   `json:"aggressive,omitempty"` // Aggressive detection
}

// DeadCodeResult represents dead code detection results
type DeadCodeResult struct {
	DeadFunctions []DeadCodeEntry `json:"deadFunctions,omitempty"`
	DeadVariables []DeadCodeEntry `json:"deadVariables,omitempty"`
	DeadImports   []DeadCodeEntry `json:"deadImports,omitempty"`
	UnusedParams  []DeadCodeEntry `json:"unusedParams,omitempty"`
}

// DeadCodeEntry represents a dead code entry
type DeadCodeEntry struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Location Location `json:"location"`
	Reason   string   `json:"reason"`
}

// =======================
// Refactoring Extensions
// =======================

// ExtractFunctionParams represents extract function parameters
type ExtractFunctionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	FunctionName string                 `json:"functionName"`
	Options      ExtractOptions         `json:"options,omitempty"`
}

// ExtractOptions configures extraction behavior
type ExtractOptions struct {
	GenerateTypes bool `json:"generateTypes"` // Generate type annotations
	InlineVars    bool `json:"inlineVars"`    // Inline simple variables
	Position      *Position `json:"position,omitempty"` // Where to place function
}

// ExtractFunctionResult represents extraction result
type ExtractFunctionResult struct {
	Edit       WorkspaceEdit `json:"edit"`
	Function   FunctionInfo  `json:"function"`
	Parameters []ParamInfo   `json:"parameters"`
	ReturnType string        `json:"returnType,omitempty"`
}

// FunctionInfo represents extracted function information
type FunctionInfo struct {
	Name      string   `json:"name"`
	Signature string   `json:"signature"`
	Body      string   `json:"body"`
	Location  Location `json:"location"`
}

// ParamInfo represents parameter information
type ParamInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// =======================
// Migration & Transform
// =======================

// MigrateLuaParams represents Lua migration parameters
type MigrateLuaParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      MigrationOptions       `json:"options"`
}

// MigrationOptions configures migration behavior
type MigrationOptions struct {
	AddTypes           bool `json:"addTypes"`
	UseConst           bool `json:"useConst"`
	GenerateInterfaces bool `json:"generateInterfaces"`
	Modernize          bool `json:"modernize"`
	StrictMode         bool `json:"strictMode"`
}

// MigrateLuaResult represents migration result
type MigrateLuaResult struct {
	Code        string          `json:"code"`
	Changes     []MigrationEdit `json:"changes"`
	Suggestions []string        `json:"suggestions,omitempty"`
	Stats       MigrationStats  `json:"stats"`
}

// MigrationEdit represents a migration edit
type MigrationEdit struct {
	Range       Range  `json:"range"`
	OldText     string `json:"oldText"`
	NewText     string `json:"newText"`
	Description string `json:"description"`
}

// MigrationStats represents migration statistics
type MigrationStats struct {
	TypesAdded       int `json:"typesAdded"`
	FunctionsUpdated int `json:"functionsUpdated"`
	ConstUsed        int `json:"constUsed"`
	InterfacesAdded  int `json:"interfacesAdded"`
}

// GenerateJITParams represents JIT hint generation parameters
type GenerateJITParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      JITOptions             `json:"options,omitempty"`
}

// JITOptions configures JIT hint generation
type JITOptions struct {
	EnableProfiling bool `json:"enableProfiling"`
	OptimizeHotPath bool `json:"optimizeHotPath"`
	AddAnnotations  bool `json:"addAnnotations"`
}

// GenerateJITResult represents JIT generation result
type GenerateJITResult struct {
	Code        string     `json:"code"`
	Hints       []JITHint  `json:"hints"`
	HotPaths    []Location `json:"hotPaths,omitempty"`
}

// JITHint represents a JIT optimization hint
type JITHint struct {
	Location    Location `json:"location"`
	Type        string   `json:"type"` // "on" | "off" | "hotpath" | etc
	Description string   `json:"description"`
}
