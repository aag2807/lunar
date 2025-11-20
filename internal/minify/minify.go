package minify

import (
	"regexp"
	"strings"
)

// Options controls minification behavior
type Options struct {
	RemoveComments      bool
	RemoveWhitespace    bool
	ShortenVariables    bool
	PreserveLineNumbers bool // For source maps
}

// DefaultOptions returns default minification options
func DefaultOptions() Options {
	return Options{
		RemoveComments:      true,
		RemoveWhitespace:    true,
		ShortenVariables:    false, // Conservative default
		PreserveLineNumbers: false,
	}
}

// Minify minifies Lua code according to the given options
func Minify(code string, opts Options) string {
	if opts.RemoveComments {
		code = removeComments(code)
	}

	if opts.RemoveWhitespace {
		code = removeExtraWhitespace(code)
	}

	// Variable shortening is more complex and optional
	if opts.ShortenVariables {
		code = shortenVariables(code)
	}

	return code
}

// removeComments removes Lua comments from code
func removeComments(code string) string {
	var result strings.Builder
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		// Check for block comments
		if strings.Contains(line, "--[[") {
			// Find end of block comment
			endIdx := strings.Index(line, "]]")
			if endIdx != -1 {
				// Block comment ends on same line
				before := line[:strings.Index(line, "--[[")]
				after := line[endIdx+2:]
				result.WriteString(before + after + "\n")
			} else {
				// Block comment continues - skip
				result.WriteString("\n")
			}
			continue
		}

		// Check for single-line comments
		commentIdx := strings.Index(line, "--")
		if commentIdx != -1 {
			// Make sure it's not inside a string
			if !isInString(line, commentIdx) {
				line = line[:commentIdx]
			}
		}

		result.WriteString(line + "\n")
	}

	return result.String()
}

// removeExtraWhitespace removes unnecessary whitespace while preserving syntax
func removeExtraWhitespace(code string) string {
	// Replace multiple spaces with single space
	multiSpace := regexp.MustCompile(`[ \t]+`)
	code = multiSpace.ReplaceAllString(code, " ")

	// Remove spaces around operators where safe
	code = strings.ReplaceAll(code, " = ", "=")
	code = strings.ReplaceAll(code, " + ", "+")
	code = strings.ReplaceAll(code, " - ", "-")
	code = strings.ReplaceAll(code, " * ", "*")
	code = strings.ReplaceAll(code, " / ", "/")
	code = strings.ReplaceAll(code, " , ", ",")

	// Remove trailing whitespace
	lines := strings.Split(code, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != "" || len(result) > 0 {
			result = append(result, trimmed)
		}
	}

	// Remove empty lines between statements (but keep at least one newline)
	output := strings.Join(result, "\n")
	multiNewline := regexp.MustCompile(`\n\n+`)
	output = multiNewline.ReplaceAllString(output, "\n")

	return output
}

// shortenVariables renames local variables to shorter names
// This is a simplified implementation - full implementation would need AST analysis
func shortenVariables(code string) string {
	// For now, this is a placeholder
	// A full implementation would:
	// 1. Parse the AST
	// 2. Build scope information
	// 3. Rename local variables to a, b, c, etc.
	// 4. Preserve global variables and function names
	// 5. Handle closures correctly

	// Simplified: just return the code as-is
	// This would need AST-level transformation to be safe
	return code
}

// isInString checks if an index is inside a string literal
func isInString(line string, idx int) bool {
	inString := false
	stringChar := byte(0)

	for i := 0; i < idx; i++ {
		c := line[i]
		if !inString {
			if c == '"' || c == '\'' {
				inString = true
				stringChar = c
			}
		} else {
			if c == stringChar && (i == 0 || line[i-1] != '\\') {
				inString = false
			}
		}
	}

	return inString
}

// Stats returns statistics about minification
type Stats struct {
	OriginalSize   int
	MinifiedSize   int
	SavingsBytes   int
	SavingsPercent float64
}

// CompareSize compares original and minified code sizes
func CompareSize(original, minified string) Stats {
	origSize := len(original)
	minSize := len(minified)
	savings := origSize - minSize
	percent := 0.0
	if origSize > 0 {
		percent = (float64(savings) / float64(origSize)) * 100
	}

	return Stats{
		OriginalSize:   origSize,
		MinifiedSize:   minSize,
		SavingsBytes:   savings,
		SavingsPercent: percent,
	}
}
