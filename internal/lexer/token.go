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
	PROTECTED   = "protected"
	STATIC      = "static"
	ABSTRACT    = "abstract"
	READONLY    = "readonly"
	FUNCTION    = "function"
	LOCAL       = "local"
	CONST       = "const"
	RETURN      = "return"
	IF          = "if"
	ELSE        = "else"
	THEN        = "then"
	FOR         = "for"
	WHILE       = "while"
	DO          = "do"
	BREAK       = "break"
	IN          = "in"
	EXTENDS     = "extends"
	IMPLEMENTS  = "implements"
	CONSTRUCTOR = "constructor"
	SELF        = "self"
	SUPER       = "super"
	VOID        = "void"
	EXPORT      = "export"
	IMPORT      = "import"
	FROM        = "from"
	DECLARE     = "declare"

	//types
	ANY         = "any"
	STRING_TYPE = "string"
	NUMBER_TYPE = "number"
	BOOLEAN     = "boolean"
	NIL         = "nil"
	TRUE        = "true"
	FALSE       = "false"

	ARROW    = "=>"
	QUESTION = "?"
	TABLE    = "table"
	PIPE     = "|"
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
	"protected":   PROTECTED,
	"static":      STATIC,
	"abstract":    ABSTRACT,
	"readonly":    READONLY,
	"function":    FUNCTION,
	"local":       LOCAL,
	"const":       CONST,
	"return":      RETURN,
	"if":          IF,
	"else":        ELSE,
	"then":        THEN,
	"for":         FOR,
	"while":       WHILE,
	"do":          DO,
	"break":       BREAK,
	"in":          IN,
	"extends":     EXTENDS,
	"implements":  IMPLEMENTS,
	"constructor": CONSTRUCTOR,
	"self":        SELF,
	"super":       SUPER,
	"and":         AND,
	"or":          OR,
	"not":         NOT,
	"void":        VOID,
	"export":      EXPORT,
	"import":      IMPORT,
	"from":        FROM,
	"declare":     DECLARE,
	"table":       TABLE,
	"any":         ANY,
	"string":      STRING_TYPE,
	"number":      NUMBER_TYPE,
	"boolean":     BOOLEAN,
	"nil":         NIL,
	"true":        TRUE,
	"false":       FALSE,
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
