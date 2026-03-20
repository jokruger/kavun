package token

import "strconv"

var keywords map[string]Token

// Token represents a token.
type Token int

// List of tokens
const (
	Illegal = Token(0)
	EOF     = Token(1)
	Comment = Token(2)
	// 3..9 are reserved for future use

	_operatorBeg = Token(10) // Operators block start
	Add          = Token(11) // +
	Sub          = Token(12) // -
	Mul          = Token(13) // *
	Quo          = Token(14) // /
	Rem          = Token(15) // %
	And          = Token(16) // &
	Or           = Token(17) // |
	Xor          = Token(18) // ^
	Shl          = Token(19) // <<
	Shr          = Token(20) // >>
	AndNot       = Token(21) // &^
	AddAssign    = Token(22) // +=
	SubAssign    = Token(23) // -=
	MulAssign    = Token(24) // *=
	QuoAssign    = Token(25) // /=
	RemAssign    = Token(26) // %=
	AndAssign    = Token(27) // &=
	OrAssign     = Token(28) // |=
	XorAssign    = Token(29) // ^=
	ShlAssign    = Token(30) // <<=
	ShrAssign    = Token(31) // >>=
	AndNotAssign = Token(32) // &^=
	LAnd         = Token(33) // &&
	LOr          = Token(34) // ||
	Inc          = Token(35) // ++
	Dec          = Token(36) // --
	Equal        = Token(37) // ==
	Less         = Token(38) // <
	Greater      = Token(39) // >
	Assign       = Token(40) // =
	Not          = Token(41) // !
	NotEqual     = Token(42) // !=
	LessEq       = Token(43) // <=
	GreaterEq    = Token(44) // >=
	Define       = Token(45) // :=
	Ellipsis     = Token(46) // ...
	LParen       = Token(47) // (
	LBrack       = Token(48) // [
	LBrace       = Token(49) // {
	Comma        = Token(50) // ,
	Period       = Token(51) // .
	RParen       = Token(52) // )
	RBrack       = Token(53) // ]
	RBrace       = Token(54) // }
	Semicolon    = Token(55) // ;
	Colon        = Token(56) // :
	Question     = Token(57) // ?
	// 58..130 are reserved for future operators
	_operatorEnd = Token(131) // Operators block end

	_literalBeg = Token(132) // Literals block start
	Ident       = Token(133)
	Int         = Token(134)
	Float       = Token(135)
	Char        = Token(136)
	String      = Token(137)
	// 138..152 are reserved for future literal types
	_literalEnd = Token(153) // Literals block end

	_keywordBeg = Token(154) // Keywords block start
	Break       = Token(155)
	Continue    = Token(156)
	Else        = Token(157)
	For         = Token(158)
	Func        = Token(159)
	Error       = Token(160)
	Immutable   = Token(161)
	If          = Token(162)
	Return      = Token(163)
	Export      = Token(164)
	True        = Token(165)
	False       = Token(166)
	In          = Token(167)
	Undefined   = Token(168)
	Import      = Token(169)
	Arrow       = Token(170) // => (behaves as a keyword)
	// 171..254 are reserved for future keywords
	_keywordEnd = Token(255) // Keywords block end
)

var tokens = [...]string{
	Illegal: "ILLEGAL",
	EOF:     "EOF",
	Comment: "COMMENT",

	_operatorBeg: "",
	Add:          "+",
	Sub:          "-",
	Mul:          "*",
	Quo:          "/",
	Rem:          "%",
	And:          "&",
	Or:           "|",
	Xor:          "^",
	Shl:          "<<",
	Shr:          ">>",
	AndNot:       "&^",
	AddAssign:    "+=",
	SubAssign:    "-=",
	MulAssign:    "*=",
	QuoAssign:    "/=",
	RemAssign:    "%=",
	AndAssign:    "&=",
	OrAssign:     "|=",
	XorAssign:    "^=",
	ShlAssign:    "<<=",
	ShrAssign:    ">>=",
	AndNotAssign: "&^=",
	LAnd:         "&&",
	LOr:          "||",
	Inc:          "++",
	Dec:          "--",
	Equal:        "==",
	Less:         "<",
	Greater:      ">",
	Assign:       "=",
	Not:          "!",
	NotEqual:     "!=",
	LessEq:       "<=",
	GreaterEq:    ">=",
	Define:       ":=",
	Ellipsis:     "...",
	LParen:       "(",
	LBrack:       "[",
	LBrace:       "{",
	Comma:        ",",
	Period:       ".",
	RParen:       ")",
	RBrack:       "]",
	RBrace:       "}",
	Semicolon:    ";",
	Colon:        ":",
	Question:     "?",
	_operatorEnd: "",

	_literalBeg: "",
	Ident:       "IDENT",
	Int:         "INT",
	Float:       "FLOAT",
	Char:        "CHAR",
	String:      "STRING",
	_literalEnd: "",

	_keywordBeg: "",
	Break:       "break",
	Continue:    "continue",
	Else:        "else",
	For:         "for",
	Func:        "func",
	Error:       "error",
	Immutable:   "immutable",
	If:          "if",
	Return:      "return",
	Export:      "export",
	True:        "true",
	False:       "false",
	In:          "in",
	Undefined:   "undefined",
	Import:      "import",
	Arrow:       "=>",
	_keywordEnd: "",
}

func (tok Token) String() string {
	s := ""

	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}

	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}

	return s
}

// LowestPrec represents lowest operator precedence.
const LowestPrec = 0

// Precedence returns the precedence for the operator token.
func (tok Token) Precedence() int {
	switch tok {
	case LOr:
		return 1
	case LAnd:
		return 2
	case Equal, NotEqual, Less, LessEq, Greater, GreaterEq:
		return 3
	case Add, Sub, Or, Xor:
		return 4
	case Mul, Quo, Rem, Shl, Shr, And, AndNot:
		return 5
	}
	return LowestPrec
}

// IsLiteral returns true if the token is a literal.
func (tok Token) IsLiteral() bool {
	return _literalBeg < tok && tok < _literalEnd
}

// IsOperator returns true if the token is an operator.
func (tok Token) IsOperator() bool {
	return _operatorBeg < tok && tok < _operatorEnd
}

// IsKeyword returns true if the token is a keyword.
func (tok Token) IsKeyword() bool {
	return _keywordBeg < tok && tok < _keywordEnd
}

// Lookup returns corresponding keyword if ident is a keyword.
func Lookup(ident string) Token {
	if tok, isKeyword := keywords[ident]; isKeyword {
		return tok
	}
	return Ident
}

func init() {
	keywords = make(map[string]Token)
	for i := _keywordBeg + 1; i < _keywordEnd; i++ {
		keywords[tokens[i]] = i
	}
}
