// Package tokens is the enum of lexical tokens: one untyped constant per token. It holds constants only — the kind
// (token.Token), the spelling table, the operator/literal/keyword ranges and every function live in core/token.
package tokens

const (
	Illegal = 0
	EOF     = 1
	Comment = 2
	// 3..9 are reserved for future use

	// 10: Operators block start
	Add          = 11 // +
	Sub          = 12 // -
	Mul          = 13 // *
	Quo          = 14 // /
	Rem          = 15 // %
	And          = 16 // &
	Or           = 17 // |
	Xor          = 18 // ^
	Shl          = 19 // <<
	Shr          = 20 // >>
	AndNot       = 21 // &^
	AddAssign    = 22 // +=
	SubAssign    = 23 // -=
	MulAssign    = 24 // *=
	QuoAssign    = 25 // /=
	RemAssign    = 26 // %=
	AndAssign    = 27 // &=
	OrAssign     = 28 // |=
	XorAssign    = 29 // ^=
	ShlAssign    = 30 // <<=
	ShrAssign    = 31 // >>=
	AndNotAssign = 32 // &^=
	LAnd         = 33 // &&
	LOr          = 34 // ||
	Inc          = 35 // ++
	Dec          = 36 // --
	Equal        = 37 // ==
	Less         = 38 // <
	Greater      = 39 // >
	Assign       = 40 // =
	Not          = 41 // !
	NotEqual     = 42 // !=
	LessEq       = 43 // <=
	GreaterEq    = 44 // >=
	Define       = 45 // :=
	Ellipsis     = 46 // ...
	LParen       = 47 // (
	LBrack       = 48 // [
	LBrace       = 49 // {
	Comma        = 50 // ,
	Period       = 51 // .
	RParen       = 52 // )
	RBrack       = 53 // ]
	RBrace       = 54 // }
	Semicolon    = 55 // ;
	Colon        = 56 // :
	Question     = 57 // ?
	DotDot       = 58 // ..
	// 59..130 are reserved for future operators
	// 131: Operators block end

	// 132: Literals block start
	Ident       = 133
	Int         = 134
	Float       = 135
	Char        = 136
	String      = 137
	Decimal     = 138
	RunesString = 139 // u"..."
	BytesString = 140 // b"..."
	ByteChar    = 141 // b'...'
	TimeString  = 142 // t"..."
	RawString   = 143 // r"..."
	FString     = 144 // f"..."
	DateString  = 145 // d"..."
	// 146..152 are reserved for future literal types
	// 153: Literals block end

	// 154: Keywords block start
	Break    = 155
	Continue = 156
	Else     = 157
	For      = 158
	Func     = 159
	Arrow    = 160 // => (behaves as a keyword)
	// 161 was `immutable`, removed: freeze_shallow(x) is the one spelling of a header-only freeze
	If        = 162
	Return    = 163
	Export    = 164
	True      = 165
	False     = 166
	In        = 167
	NotKw     = 168
	Undefined = 169
	Import    = 170
	Var       = 171
	Defer     = 172
	// 173..254 are reserved for future keywords
	// 255: Keywords block end
)
