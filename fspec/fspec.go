package fspec

type Align byte

const (
	AlignNone   Align = 0
	AlignLeft   Align = '<'
	AlignRight  Align = '>'
	AlignCenter Align = '^'
	AlignSign   Align = '='
)

type Sign byte

const (
	SignDefault Sign = 0
	SignPlus    Sign = '+'
	SignMinus   Sign = '-'
	SignSpace   Sign = ' '
)

// FormatSpec is the fully parsed format spec.
type FormatSpec struct {
	// generic
	Fill     rune  // 0 = unset
	Align    Align // 0 = type default
	Width    int16
	HasWidth bool
	ZeroPad  bool // leading-0 shortcut

	// numeric / shared
	Sign       Sign
	Grouping   byte // 0, ',' or '_'
	Precision  int16
	HasPrec    bool
	CoerceZero bool // '~' — for float / decimal: coerce -0 to +0 after rounding
	Bare       bool // '!' — suppress conventional prefix ("0b", "0o", "0x", "0X") on integer prefix-emitting verbs

	// discriminator
	Verb byte   // 0 = default; one ASCII letter; or '#' when a tail is present
	Tail string // anything after '#'; "" if absent
}
