package formatter

import (
	"lunar/internal/ast"
	"strings"
)

// Options configures the formatter behavior
type Options struct {
	IndentStyle   string // "tab" or "space"
	IndentSize    int    // number of spaces (if IndentStyle is "space")
	MaxLineLength int    // max line length before wrapping (0 = no limit)
}

// DefaultOptions returns the default formatting options
func DefaultOptions() Options {
	return Options{
		IndentStyle:   "tab",
		IndentSize:    4,
		MaxLineLength: 120,
	}
}

// Formatter formats Lunar source code
type Formatter struct {
	options Options
	indent  int
	output  strings.Builder
}

// New creates a new formatter with the given options
func New(options Options) *Formatter {
	return &Formatter{
		options: options,
		indent:  0,
	}
}

// Format formats a program (list of statements)
func (f *Formatter) Format(statements []ast.Statement) string {
	f.output.Reset()
	f.indent = 0

	var lines []string
	for _, stmt := range statements {
		formatted := f.formatStatement(stmt)
		if formatted != "" {
			lines = append(lines, formatted)
		}
	}

	return strings.Join(lines, "\n\n") + "\n"
}

// formatStatement formats a single statement
func (f *Formatter) formatStatement(stmt ast.Statement) string {
	// Use the AST's String() method as base
	raw := stmt.String()

	// Apply consistent indentation
	return f.normalizeIndentation(raw)
}

// normalizeIndentation ensures consistent indentation throughout the code
func (f *Formatter) normalizeIndentation(code string) string {
	lines := strings.Split(code, "\n")
	var result []string

	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// Decrease indent for closing keywords
		if f.isClosingKeyword(trimmed) {
			indent--
			if indent < 0 {
				indent = 0
			}
		}

		// Apply current indentation
		indentStr := f.getIndentString(indent)
		result = append(result, indentStr+trimmed)

		// Increase indent for opening keywords
		if f.isOpeningKeyword(trimmed) {
			indent++
		}
	}

	return strings.Join(result, "\n")
}

// getIndentString returns the indentation string for the given level
func (f *Formatter) getIndentString(level int) string {
	if f.options.IndentStyle == "tab" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", level*f.options.IndentSize)
}

// isOpeningKeyword checks if a line opens a new block
func (f *Formatter) isOpeningKeyword(line string) bool {
	// Check for keywords that open blocks
	openers := []string{
		"function ",
		"class ",
		"interface ",
		"enum ",
		"namespace ",
		"if ",
		"elseif ",
		"else",
		"while ",
		"for ",
		"do",
		"repeat",
	}

	for _, opener := range openers {
		if strings.HasPrefix(line, opener) {
			return true
		}
	}

	// Also check for lines ending with "then" or "do"
	if strings.HasSuffix(line, " then") || strings.HasSuffix(line, " do") {
		return true
	}

	return false
}

// isClosingKeyword checks if a line closes a block
func (f *Formatter) isClosingKeyword(line string) bool {
	closers := []string{
		"end",
		"else",
		"elseif ",
		"until ",
	}

	for _, closer := range closers {
		if line == closer || strings.HasPrefix(line, closer) {
			return true
		}
	}

	return false
}

// FormatSource formats Lunar source code string
func FormatSource(source string) (string, error) {
	// Import lexer and parser
	l := NewLexerAdapter(source)
	p := NewParserAdapter(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return "", &FormatError{Errors: p.Errors()}
	}

	formatter := New(DefaultOptions())
	return formatter.Format(statements), nil
}

// FormatError represents formatting errors
type FormatError struct {
	Errors []string
}

func (e *FormatError) Error() string {
	return strings.Join(e.Errors, "\n")
}
