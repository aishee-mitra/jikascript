package lexer

// TokenType identifies the token category.
type TokenType string

const (
	EOF        TokenType = "EOF"
	ILLEGAL    TokenType = "ILLEGAL"
	IDENT      TokenType = "IDENT"
	NUMBER     TokenType = "NUMBER"
	STRING     TokenType = "STRING"
	BOOL       TokenType = "BOOL"
	KEYWORD    TokenType = "KEYWORD"
	PUNCT      TokenType = "PUNCT"
	OPERATOR   TokenType = "OPERATOR"
	ASSIGN     TokenType = "ASSIGN"
	COMMENT    TokenType = "COMMENT"
	WHITESPACE TokenType = "WHITESPACE"
)

// Token stores the lexical token and its location.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// Keywords are the reserved words in the initial v0.1 language subset.
var Keywords = map[string]TokenType{
	"let":       KEYWORD,
	"const":     KEYWORD,
	"function":  KEYWORD,
	"return":    KEYWORD,
	"if":        KEYWORD,
	"else":      KEYWORD,
	"while":     KEYWORD,
	"for":       KEYWORD,
	"break":     KEYWORD,
	"continue":  KEYWORD,
	"true":      BOOL,
	"false":     BOOL,
	"task":      KEYWORD,
	"pin":       KEYWORD,
	"scheduler": KEYWORD,
	"start":     KEYWORD,
}

// Punctuators and operators recognized by the lexer.
var SingleCharPunctuators = map[byte]string{
	'(': "(",
	')': ")",
	'{': "{",
	'}': "}",
	'[': "[",
	']': "]",
	';': ";",
	',': ",",
	'.': ".",
	':': ":",
}

var SingleCharOperators = map[byte]string{
	'+': "+",
	'-': "-",
	'*': "*",
	'/': "/",
	'%': "%",
	'=': "=",
	'!': "!",
	'<': "<",
	'>': ">",
	'&': "&",
	'|': "|",
}
