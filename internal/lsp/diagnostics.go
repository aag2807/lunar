package lsp

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
)

// DiagnosticsEngine provides diagnostics for Lunar code
type DiagnosticsEngine struct {
}

// NewDiagnosticsEngine creates a new diagnostics engine
func NewDiagnosticsEngine() *DiagnosticsEngine {
	return &DiagnosticsEngine{}
}

// Analyze analyzes source code and returns diagnostics
func (de *DiagnosticsEngine) Analyze(uri string, content string) []Diagnostic {
	diagnostics := []Diagnostic{}

	// Parse the source
	l := lexer.New(content)
	p := parser.New(l)
	statements := p.Parse()

	// Add parse errors
	for _, err := range p.Errors() {
		// Use line/column info from ParseError (convert to 0-based)
		line := err.Line - 1
		if line < 0 {
			line = 0
		}
		col := err.Column - 1
		if col < 0 {
			col = 0
		}

		diagnostic := Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: col},
				End:   Position{Line: line, Character: col + 1},
			},
			Severity: SeverityError,
			Source:   "lunar",
			Message:  err.Message,
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	// If there are parse errors, don't run type checking
	if len(p.Errors()) > 0 {
		return diagnostics
	}

	// Run type checker
	stdlibPath := getStdlibPathForLSP()
	checker := types.NewChecker(stdlibPath)
	typeErrors := checker.Check(statements)

	for _, err := range typeErrors {
		// Convert 1-based line to 0-based
		line := err.Line - 1
		if line < 0 {
			line = 0
		}
		col := err.Column - 1
		if col < 0 {
			col = 0
		}

		diagnostic := Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: col},
				End:   Position{Line: line, Character: col + 10}, // Approximate end
			},
			Severity: SeverityError,
			Source:   "lunar",
			Message:  err.Message,
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics
}
