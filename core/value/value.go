package value

// Type ids. Only two hard constraints exist: Undefined is 0, and the data-only primitives are contiguous from 0
// (IsPrimitive gates the static pool). The rest is grouped by kind, each group with a reserve; the language owns
// 0–191 and embedders 192–255. Ids 112–191 belong to the builtin modules (112–143, module id == type id) and the
// types they own (144–191): they are FROZEN like member ids — alphabetical on first allocation, then append-only,
// never renumbered. Embedder types and embedder modules share one space, 192–255. A planned type is a COMMENT on
// its ReservedNN constant: a constant exists when the type does.
const (
	// 0–15: data-only primitives
	Undefined         = uint8(0) // must be zero
	Bool              = uint8(1)
	Byte              = uint8(2)
	Rune              = uint8(3)
	Int               = uint8(4)
	Float             = uint8(5)
	Date              = uint8(6)
	Reserved7         = uint8(7)
	Reserved8         = uint8(8)
	Reserved9         = uint8(9)
	Reserved10        = uint8(10)
	Reserved11        = uint8(11)
	Reserved12        = uint8(12)
	Reserved13        = uint8(13)
	Reserved14        = uint8(14)
	Reserved15        = uint8(15)
	LastPrimitiveType = Reserved15

	// 16–31: internal and callables
	ValuePtr         = uint8(16)
	BuiltinFunction  = uint8(17)
	BuiltinClosure   = uint8(18)
	CompiledFunction = uint8(19)
	BoundMember      = uint8(20)
	FormatSpec       = uint8(21)
	Reserved22       = uint8(22)
	Reserved23       = uint8(23)
	Reserved24       = uint8(24)
	Reserved25       = uint8(25)
	Reserved26       = uint8(26)
	Reserved27       = uint8(27)
	Reserved28       = uint8(28)
	Reserved29       = uint8(29)
	Reserved30       = uint8(30)
	Reserved31       = uint8(31)

	// 32–47: heap scalars
	String     = uint8(32)
	Decimal    = uint8(33)
	Time       = uint8(34)
	Error      = uint8(35)
	Reserved36 = uint8(36)
	Reserved37 = uint8(37)
	Reserved38 = uint8(38)
	Reserved39 = uint8(39)
	Reserved40 = uint8(40)
	Reserved41 = uint8(41)
	Reserved42 = uint8(42)
	Reserved43 = uint8(43)
	Reserved44 = uint8(44)
	Reserved45 = uint8(45)
	Reserved46 = uint8(46)
	Reserved47 = uint8(47)

	// 48–79: containers and vectors
	Array      = uint8(48)
	Dict       = uint8(49)
	Record     = uint8(50)
	IntRange   = uint8(51)
	Bytes      = uint8(52)
	Runes      = uint8(53)
	Reserved54 = uint8(54) // Set
	Reserved55 = uint8(55) // Bools
	Reserved56 = uint8(56) // Ints
	Reserved57 = uint8(57) // Floats
	Reserved58 = uint8(58) // Decimals
	Reserved59 = uint8(59) // Strings
	Reserved60 = uint8(60) // Times
	Reserved61 = uint8(61) // Dates
	Reserved62 = uint8(62)
	Reserved63 = uint8(63)
	Reserved64 = uint8(64)
	Reserved65 = uint8(65)
	Reserved66 = uint8(66)
	Reserved67 = uint8(67)
	Reserved68 = uint8(68)
	Reserved69 = uint8(69)
	Reserved70 = uint8(70)
	Reserved71 = uint8(71)
	Reserved72 = uint8(72)
	Reserved73 = uint8(73)
	Reserved74 = uint8(74)
	Reserved75 = uint8(75)
	Reserved76 = uint8(76)
	Reserved77 = uint8(77)
	Reserved78 = uint8(78)
	Reserved79 = uint8(79)

	// 80–111: iterators, each at its container + 32
	ArrayIterator    = uint8(80)
	DictIterator     = uint8(81)
	Reserved82       = uint8(82) // record iterates with the dict iterator
	IntRangeIterator = uint8(83)
	BytesIterator    = uint8(84)
	RunesIterator    = uint8(85)
	Reserved86       = uint8(86) // SetIterator
	Reserved87       = uint8(87)
	Reserved88       = uint8(88)
	Reserved89       = uint8(89)
	Reserved90       = uint8(90)
	Reserved91       = uint8(91)
	Reserved92       = uint8(92)
	Reserved93       = uint8(93)
	Reserved94       = uint8(94)
	Reserved95       = uint8(95)
	Reserved96       = uint8(96)
	Reserved97       = uint8(97)
	Reserved98       = uint8(98)
	Reserved99       = uint8(99)
	Reserved100      = uint8(100)
	Reserved101      = uint8(101)
	Reserved102      = uint8(102)
	Reserved103      = uint8(103)
	Reserved104      = uint8(104)
	Reserved105      = uint8(105)
	Reserved106      = uint8(106)
	Reserved107      = uint8(107)
	Reserved108      = uint8(108)
	Reserved109      = uint8(109)
	Reserved110      = uint8(110)
	Reserved111      = uint8(111)

	// 112–143: builtin module types (module id == type id)
	ModuleBase64 = uint8(112)
	ModuleClock  = uint8(113)
	ModuleExec   = uint8(114)
	ModuleFin    = uint8(115)
	ModuleFs     = uint8(116)
	ModuleHex    = uint8(117)
	ModuleIo     = uint8(118)
	ModuleJson   = uint8(119)
	ModuleMath   = uint8(120)
	ModuleOs     = uint8(121)
	ModulePath   = uint8(122)
	ModuleRand   = uint8(123)
	ModuleRe     = uint8(124)
	Reserved125  = uint8(125)
	Reserved126  = uint8(126)
	Reserved127  = uint8(127)
	Reserved128  = uint8(128)
	Reserved129  = uint8(129)
	Reserved130  = uint8(130)
	Reserved131  = uint8(131)
	Reserved132  = uint8(132)
	Reserved133  = uint8(133)
	Reserved134  = uint8(134)
	Reserved135  = uint8(135)
	Reserved136  = uint8(136)
	Reserved137  = uint8(137)
	Reserved138  = uint8(138)
	Reserved139  = uint8(139)
	Reserved140  = uint8(140)
	Reserved141  = uint8(141)
	Reserved142  = uint8(142)
	Reserved143  = uint8(143)

	// 144–191: module-owned types, registered in core, named <module>.<constructor>
	ExecProcess      = uint8(144) // exec.process
	FinYearFraction  = uint8(145) // fin.year_fraction
	FinTieredRates   = uint8(146) // fin.tiered_rates
	FinTieredCharges = uint8(147) // fin.tiered_charges
	FinDatedRates    = uint8(148) // fin.dated_rates
	FinDatedCharges  = uint8(149) // fin.dated_charges
	FsFile           = uint8(150) // fs.file
	RandGenerator    = uint8(151) // rand.generator
	RePattern        = uint8(152) // re.pattern
	ExecCommand      = uint8(153) // exec.command
	Reserved154      = uint8(154)
	Reserved155      = uint8(155)
	Reserved156      = uint8(156)
	Reserved157      = uint8(157)
	Reserved158      = uint8(158)
	Reserved159      = uint8(159)
	Reserved160      = uint8(160)
	Reserved161      = uint8(161)
	Reserved162      = uint8(162)
	Reserved163      = uint8(163)
	Reserved164      = uint8(164)
	Reserved165      = uint8(165)
	Reserved166      = uint8(166)
	Reserved167      = uint8(167)
	Reserved168      = uint8(168)
	Reserved169      = uint8(169)
	Reserved170      = uint8(170)
	Reserved171      = uint8(171)
	Reserved172      = uint8(172)
	Reserved173      = uint8(173)
	Reserved174      = uint8(174)
	Reserved175      = uint8(175)
	Reserved176      = uint8(176)
	Reserved177      = uint8(177)
	Reserved178      = uint8(178)
	Reserved179      = uint8(179)
	Reserved180      = uint8(180)
	Reserved181      = uint8(181)
	Reserved182      = uint8(182)
	Reserved183      = uint8(183)
	Reserved184      = uint8(184)
	Reserved185      = uint8(185)
	Reserved186      = uint8(186)
	Reserved187      = uint8(187)
	Reserved188      = uint8(188)
	Reserved189      = uint8(189)
	Reserved190      = uint8(190)
	Reserved191      = uint8(191)

	// 192–255: embedder types and embedder modules
	FirstUserDefinedType = uint8(192)
)
