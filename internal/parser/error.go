package parser

import "fmt"

// ParseError represents a parsing error with location information
type ParseError struct {
	Message string
	Line    int
	Column  int
	// File is the path of the source; EndLine and EndColumn are just past
	// the offending token.
	File      string
	EndLine   int
	EndColumn int
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}
