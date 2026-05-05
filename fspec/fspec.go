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
	CoerceZero bool // 'z'

	// discriminator
	Verb byte   // 0 = default; one ASCII letter
	Tail string // anything after '#'; "" if absent
}
