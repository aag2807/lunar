package lexer

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// identifiers + literals
	IDENT  = "IDENT"
	NUMBER = "NUMBER"
	STRING = "STRING"

	// keywords specific to lunar
	CLASS     = "class"
	INTERFACE = "interface"
	END       = "end"
	TYPE      = "type"
	ENUM      = "enum"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}
