package lexer

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// identifiers + literals
	IDENT           = "IDENT"
	NUMBER          = "NUMBER"
	STRING          = "STRING"
	TEMPLATE_STRING = "TEMPLATE_STRING" // backtick string with ${} interpolation

	//operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK  = "*"
	SLASH     = "/"
	FLOOR_DIV = "//"
	MODULO    = "%"
	HASH      = "#"
	AMPERSAND = "&" // for intersection types Type1 & Type2, and bitwise AND
	AT        = "@" // for decorators @decoratorName
	CARET     = "^" // for bitwise XOR
	TILDE     = "~" // for bitwise NOT
	LEFT_SHIFT  = "<<"
	RIGHT_SHIFT = ">>"

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
	CONCAT   = ".."
	ELLIPSIS = "..."

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
	GET         = "get"
	SET         = "set"
	NAMESPACE   = "namespace"
	ASYNC       = "async"
	AWAIT       = "await"
	FUNCTION    = "function"
	LOCAL       = "local"
	CONST       = "const"
	RETURN      = "return"
	IF          = "if"
	ELSEIF      = "elseif"
	ELSE        = "else"
	THEN        = "then"
	FOR         = "for"
	WHILE       = "while"
	REPEAT      = "repeat"
	UNTIL       = "until"
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
	IS          = "is" // for type guards: value is Type
	AS          = "as" // for type assertions: value as Type
	KEYOF       = "keyof" // for keyof operator: keyof T
	TYPEOF      = "typeof" // for typeof operator: typeof value
	MATCH       = "match" // for pattern matching: match value with | pattern -> expr end
	WITH        = "with"  // for pattern matching: separates value from cases

	//types
	ANY         = "any"
	STRING_TYPE = "string"
	NUMBER_TYPE = "number"
	BOOLEAN     = "boolean"
	NIL         = "nil"
	TRUE        = "true"
	FALSE       = "false"
	NEVER       = "never"
	UNKNOWN     = "unknown"

	ARROW      = "=>"
	THIN_ARROW = "->" // For pattern matching: | pattern -> expression
	QUESTION   = "?"
	TABLE      = "table"
	PIPE       = "|"
	PIPE_OP    = "|>" // Pipeline operator

	// Optional chaining and nullish coalescing
	OPTIONAL_CHAIN     = "?."
	NULLISH_COALESCE   = "??"
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
	"get":         GET,
	"set":         SET,
	"namespace":   NAMESPACE,
	"async":       ASYNC,
	"await":       AWAIT,
	"function":    FUNCTION,
	"local":       LOCAL,
	"const":       CONST,
	"return":      RETURN,
	"if":          IF,
	"elseif":      ELSEIF,
	"else":        ELSE,
	"then":        THEN,
	"for":         FOR,
	"while":       WHILE,
	"repeat":      REPEAT,
	"until":       UNTIL,
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
	"is":          IS,
	"as":          AS,
	"keyof":       KEYOF,
	"typeof":      TYPEOF,
	"match":       MATCH,
	"with":        WITH,
	"table":       TABLE,
	"any":         ANY,
	"string":      STRING_TYPE,
	"number":      NUMBER_TYPE,
	"boolean":     BOOLEAN,
	"nil":         NIL,
	"true":        TRUE,
	"false":       FALSE,
	"never":       NEVER,
	"unknown":     UNKNOWN,
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

// luaReserved lists the words Lua itself reserves. Most Lunar keywords (type,
// match, class, ...) are ordinary identifiers to Lua, so they remain legal
// field names in generated code.
var luaReserved = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true,
	"end": true, "false": true, "for": true, "function": true, "goto": true,
	"if": true, "in": true, "local": true, "nil": true, "not": true,
	"or": true, "repeat": true, "return": true, "then": true, "true": true,
	"until": true, "while": true,
}

// IsLuaReserved reports whether a word is reserved by Lua itself.
func IsLuaReserved(word string) bool {
	return luaReserved[word]
}

// IsKeyword reports whether a word is a Lunar keyword.
func IsKeyword(word string) bool {
	_, ok := keywords[word]
	return ok
}

// CanBeFieldName reports whether a token may stand in for a field name in
// positions where no ambiguity is possible, such as after '.' or as a struct
// pattern's field. Identifiers always qualify; keywords qualify unless Lua
// reserves them, since the name has to survive into the generated Lua.
func CanBeFieldName(tok Token) bool {
	if tok.Type == IDENT {
		return true
	}

	return IsKeyword(tok.Literal) && !IsLuaReserved(tok.Literal)
}
