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
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	MODULO   = "%"

	//comparison
	EQ         = "=="
	NOT_EQ_LUA = "~="
	NOT_EQ     = "!="
	LT         = "<"
	GT         = ">"
	LT_EQ      = "<="
	GT_EQ      = ">="

	//logical
	AND = "and"
	OR  = "or"
	NOT = "not"

	//concat operator
	CONCAT = ".."

	//delimeters
	COMMA    = ","
	COLON    = ":"
	DOT      = "."
	LPAREN   = "("
	RPAREN   = ")"
	LBRACKET = "["
	RBRACKET = "]"
	LBRACE   = "{"
	RBRACE   = "}"

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
	"and":         AND,
	"or":          OR,
	"not":         NOT,
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
