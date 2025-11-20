package formatter

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"lunar/internal/parser"
)

// LexerAdapter wraps the lexer for use in the formatter
type LexerAdapter struct {
	lexer *lexer.Lexer
}

// NewLexerAdapter creates a new lexer adapter
func NewLexerAdapter(source string) *LexerAdapter {
	return &LexerAdapter{lexer: lexer.New(source)}
}

// ParserAdapter wraps the parser for use in the formatter
type ParserAdapter struct {
	parser *parser.Parser
}

// NewParserAdapter creates a new parser adapter
func NewParserAdapter(l *LexerAdapter) *ParserAdapter {
	return &ParserAdapter{parser: parser.New(l.lexer)}
}

// Parse parses the source and returns the AST
func (p *ParserAdapter) Parse() []ast.Statement {
	return p.parser.Parse()
}

// Errors returns any parsing errors
func (p *ParserAdapter) Errors() []string {
	var errs []string
	for _, err := range p.parser.Errors() {
		errs = append(errs, err.Error())
	}
	return errs
}
