package value

const (
	FirstPrimitiveType   = Undefined // Primitive types (boxed as uint64 data)
	Undefined            = uint8(0)  // always zero
	Bool                 = uint8(1)
	Byte                 = uint8(2)
	Rune                 = uint8(3)
	Int                  = uint8(4)
	Float                = uint8(5)
	Reserved6            = uint8(6)
	Reserved7            = uint8(7)
	Reserved8            = uint8(8)
	Reserved9            = uint8(9)
	Reserved10           = uint8(10)
	Reserved11           = uint8(11)
	Reserved12           = uint8(12)
	Reserved13           = uint8(13)
	Reserved14           = uint8(14)
	Reserved15           = uint8(15)
	LastPrimitiveType    = Reserved15
	FirstAssocType       = Record // Associative types
	Record               = uint8(16)
	Dict                 = uint8(17)
	Reserved18           = uint8(18)
	Reserved19           = uint8(19)
	Reserved20           = uint8(20)
	Reserved21           = uint8(21)
	Reserved22           = uint8(22)
	Reserved23           = uint8(23)
	LastAssocType        = Reserved23
	FirstSeqType         = Array // Sequence types
	Array                = uint8(24)
	Bytes                = uint8(25)
	Runes                = uint8(26)
	Reserved27           = uint8(27)
	Reserved28           = uint8(28)
	Reserved29           = uint8(29)
	Reserved30           = uint8(30)
	Reserved31           = uint8(31)
	Reserved32           = uint8(32)
	Reserved33           = uint8(33)
	Reserved34           = uint8(34)
	Reserved35           = uint8(35)
	Reserved36           = uint8(36)
	Reserved37           = uint8(37)
	Reserved38           = uint8(38)
	Reserved39           = uint8(39)
	LastSeqType          = Reserved39
	FirstRangeType       = IntRange // Range types
	IntRange             = uint8(40)
	Reserved41           = uint8(41)
	Reserved42           = uint8(42)
	Reserved43           = uint8(43)
	Reserved44           = uint8(44)
	Reserved45           = uint8(45)
	Reserved46           = uint8(46)
	Reserved47           = uint8(47)
	LastRangeType        = Reserved47
	FirstIteratorType    = DictIterator // Iterator types
	DictIterator         = uint8(48)
	ArrayIterator        = uint8(49)
	BytesIterator        = uint8(50)
	RunesIterator        = uint8(51)
	IntRangeIterator     = uint8(52)
	Reserved53           = uint8(53)
	Reserved54           = uint8(54)
	Reserved55           = uint8(55)
	Reserved56           = uint8(56)
	Reserved57           = uint8(57)
	Reserved58           = uint8(58)
	Reserved59           = uint8(59)
	Reserved60           = uint8(60)
	Reserved61           = uint8(61)
	Reserved62           = uint8(62)
	Reserved63           = uint8(63)
	LastIteratorType     = Reserved63
	FirstFunctionType    = BuiltinFunction // Function types
	BuiltinFunction      = uint8(64)
	BuiltinClosure       = uint8(65)
	CompiledFunction     = uint8(66)
	Reserved67           = uint8(67)
	Reserved68           = uint8(68)
	Reserved69           = uint8(69)
	Reserved70           = uint8(70)
	Reserved71           = uint8(71)
	LastFunctionType     = Reserved71
	ValuePtr             = uint8(72)
	FormatSpec           = uint8(73)
	Error                = uint8(74)
	Decimal              = uint8(75)
	Time                 = uint8(76)
	String               = uint8(77)
	Reserved78           = uint8(78)
	Reserved79           = uint8(79)
	Reserved80           = uint8(80)
	Reserved81           = uint8(81)
	Reserved82           = uint8(82)
	Reserved83           = uint8(83)
	Reserved84           = uint8(84)
	Reserved85           = uint8(85)
	Reserved86           = uint8(86)
	Reserved87           = uint8(87)
	Reserved88           = uint8(88)
	Reserved89           = uint8(89)
	Reserved90           = uint8(90)
	Reserved91           = uint8(91)
	Reserved92           = uint8(92)
	Reserved93           = uint8(93)
	Reserved94           = uint8(94)
	Reserved95           = uint8(95)
	Reserved96           = uint8(96)
	Reserved97           = uint8(97)
	Reserved98           = uint8(98)
	Reserved99           = uint8(99)
	Reserved100          = uint8(100)
	Reserved101          = uint8(101)
	Reserved102          = uint8(102)
	Reserved103          = uint8(103)
	LastBuiltinType      = Reserved103
	FirstUserDefinedType = UserDefined // User-defined types
	UserDefined          = uint8(104)
)

func IsPrimitiveType(t uint8) bool {
	return t <= LastPrimitiveType
}

func IsAssocType(t uint8) bool {
	return t >= FirstAssocType && t <= LastAssocType
}

func IsSeqType(t uint8) bool {
	return t >= FirstSeqType && t <= LastSeqType
}

func IsRangeType(t uint8) bool {
	return t >= FirstRangeType && t <= LastRangeType
}

func IsIteratorType(t uint8) bool {
	return t >= FirstIteratorType && t <= LastIteratorType
}

func IsFunctionType(t uint8) bool {
	return t >= FirstFunctionType && t <= LastFunctionType
}

func IsBuiltinType(t uint8) bool {
	return t <= LastBuiltinType
}

func IsUserDefinedType(t uint8) bool {
	return t >= FirstUserDefinedType
}
