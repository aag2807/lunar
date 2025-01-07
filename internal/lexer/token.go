package lexer

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// identifiers + literals
	IDENT  = "IDENT"
	NUMBER = "NUMBER"
	STRING = "STRING"

	//operators
	ASSIGN    = "="
	PLUS      = "+"
	MINUS     = "-"
	BANG      = "!"
	ASTEROISK = "*"
	SLASH     = "/"

	//delimeters
	COMMA    = ","
	COLON    = ":"
	DOT      = "."
	LPAREN   = "("
	RPAREN   = ")"
	LBRACKET = "["
	RBRACKET = "]"

	// keywords specific to lunar
	CLASS       = "class"
	INTERFACE   = "interface"
	ENUM        = "enum"
	TYPE        = "type"
	END         = "end"
	PUBLIC      = "public"
	PRIVATE     = "private"
	FUNCTION    = "function"
	LOCAL       = "local"
	CONST       = "const"
	RETURN      = "return"
	IF          = "if"
	ELSE        = "else"
	THEN        = "then"
	FOR         = "for"
	IN          = "in"
	EXTENDS     = "extends"
	IMPLEMENTS  = "implements"
	CONSTRUCTOR = "constructor"
	SELF        = "self"
)

// Map of keywords
var keywords = map[string]TokenType{
	"class":       CLASS,
	"interface":   INTERFACE,
	"enum":        ENUM,
	"type":        TYPE,
	"end":         END,
	"public":      PUBLIC,
	"private":     PRIVATE,
	"function":    FUNCTION,
	"local":       LOCAL,
	"const":       CONST,
	"return":      RETURN,
	"if":          IF,
	"else":        ELSE,
	"then":        THEN,
	"for":         FOR,
	"in":          IN,
	"extends":     EXTENDS,
	"implements":  IMPLEMENTS,
	"constructor": CONSTRUCTOR,
	"self":        SELF,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}

	return IDENT
}
