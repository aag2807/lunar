package parser

import "fmt"

// ParseError represents a parsing error with location information
type ParseError struct {
	Message string
	Line    int
	Column  int
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}
