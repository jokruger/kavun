package token

import (
	"strconv"

	"github.com/jokruger/kavun/core/token/tokens"
)

var keywords map[string]Token

type Token uint8

// Block boundaries of the token enum (core/token/tokens): IsOperator/IsLiteral/IsKeyword test against them.
const (
	operatorBeg Token = 10
	operatorEnd Token = 131
	literalBeg  Token = 132
	literalEnd  Token = 153
	keywordBeg  Token = 154
	keywordEnd  Token = 255
)

var spellings = [...]string{
	tokens.Illegal: "ILLEGAL",
	tokens.EOF:     "EOF",
	tokens.Comment: "COMMENT",

	operatorBeg:         "",
	tokens.Add:          "+",
	tokens.Sub:          "-",
	tokens.Mul:          "*",
	tokens.Quo:          "/",
	tokens.Rem:          "%",
	tokens.And:          "&",
	tokens.Or:           "|",
	tokens.Xor:          "^",
	tokens.Shl:          "<<",
	tokens.Shr:          ">>",
	tokens.AndNot:       "&^",
	tokens.AddAssign:    "+=",
	tokens.SubAssign:    "-=",
	tokens.MulAssign:    "*=",
	tokens.QuoAssign:    "/=",
	tokens.RemAssign:    "%=",
	tokens.AndAssign:    "&=",
	tokens.OrAssign:     "|=",
	tokens.XorAssign:    "^=",
	tokens.ShlAssign:    "<<=",
	tokens.ShrAssign:    ">>=",
	tokens.AndNotAssign: "&^=",
	tokens.LAnd:         "&&",
	tokens.LOr:          "||",
	tokens.Inc:          "++",
	tokens.Dec:          "--",
	tokens.Equal:        "==",
	tokens.Less:         "<",
	tokens.Greater:      ">",
	tokens.Assign:       "=",
	tokens.Not:          "!",
	tokens.NotEqual:     "!=",
	tokens.LessEq:       "<=",
	tokens.GreaterEq:    ">=",
	tokens.Define:       ":=",
	tokens.Ellipsis:     "...",
	tokens.LParen:       "(",
	tokens.LBrack:       "[",
	tokens.LBrace:       "{",
	tokens.Comma:        ",",
	tokens.Period:       ".",
	tokens.RParen:       ")",
	tokens.RBrack:       "]",
	tokens.RBrace:       "}",
	tokens.Semicolon:    ";",
	tokens.Colon:        ":",
	tokens.Question:     "?",
	tokens.DotDot:       "..",
	operatorEnd:         "",

	literalBeg:         "",
	tokens.Ident:       "IDENT",
	tokens.Int:         "INT",
	tokens.Float:       "FLOAT",
	tokens.Char:        "CHAR",
	tokens.String:      "STRING",
	tokens.Decimal:     "DECIMAL",
	tokens.RunesString: "RUNESSTRING",
	tokens.BytesString: "BYTESSTRING",
	tokens.ByteChar:    "BYTECHAR",
	tokens.TimeString:  "TIMESTRING",
	tokens.RawString:   "RAWSTRING",
	tokens.FString:     "FSTRING",
	tokens.DateString:  "DATESTRING",
	literalEnd:         "",

	keywordBeg:       "",
	tokens.Break:     "break",
	tokens.Continue:  "continue",
	tokens.Else:      "else",
	tokens.For:       "for",
	tokens.Func:      "func",
	tokens.Arrow:     "=>",
	tokens.If:        "if",
	tokens.Return:    "return",
	tokens.Export:    "export",
	tokens.True:      "true",
	tokens.False:     "false",
	tokens.In:        "in",
	tokens.NotKw:     "not",
	tokens.Undefined: "undefined",
	tokens.Import:    "import",
	tokens.Var:       "var",
	tokens.Defer:     "defer",
	keywordEnd:       "",
}

func (tok Token) String() string {
	s := spellings[tok]
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
	case tokens.LOr:
		return 1
	case tokens.LAnd:
		return 2
	case tokens.Equal, tokens.NotEqual, tokens.Less, tokens.LessEq, tokens.Greater, tokens.GreaterEq, tokens.In:
		return 3
	case tokens.Add, tokens.Sub, tokens.Or, tokens.Xor:
		return 4
	case tokens.Mul, tokens.Quo, tokens.Rem, tokens.Shl, tokens.Shr, tokens.And, tokens.AndNot:
		return 5
	}
	return LowestPrec
}

// IsLiteral returns true if the token is a literal.
func (tok Token) IsLiteral() bool {
	return literalBeg < tok && tok < literalEnd
}

// IsOperator returns true if the token is an operator.
func (tok Token) IsOperator() bool {
	return operatorBeg < tok && tok < operatorEnd
}

// IsKeyword returns true if the token is a keyword.
func (tok Token) IsKeyword() bool {
	return keywordBeg < tok && tok < keywordEnd
}

// Lookup returns corresponding keyword if ident is a keyword.
func Lookup(ident string) Token {
	if tok, isKeyword := keywords[ident]; isKeyword {
		return tok
	}
	return tokens.Ident
}

func init() {
	keywords = make(map[string]Token)
	for i := keywordBeg + 1; i < keywordEnd; i++ {
		if spellings[i] == "" { // a retired keyword leaves its number unused
			continue
		}
		keywords[spellings[i]] = i
	}
}
