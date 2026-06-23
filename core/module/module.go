package module

const (
	// Builtin modules
	Global = uint8(0)
	Base64 = uint8(1)
	Fmt    = uint8(2)
	Hex    = uint8(3)
	Json   = uint8(4)
	Math   = uint8(5)
	OS     = uint8(6)
	Rand   = uint8(7)
	Text   = uint8(8)
	Times  = uint8(9)
	// 10..15 reserved for future built-in modules
	UserDefined = uint8(16) // 16..31 reserved for user-defined builtin modules
)
