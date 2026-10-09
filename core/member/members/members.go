// Package members is the enum of builtin member ids: one untyped constant per member name a builtin type or stdlib
// module answers (or is planned to answer). It holds constants only — the kind (member.ID), the id ↔ name table and
// every function live in core/member — so an identifier here can only ever collide with another member name.
//
// FROZEN: a released id is never changed, removed or reused. A new name takes the next free id inside its block's
// reserved range, or follows the last block once that range is full; a retired name keeps its line, commented out.
// Blocks run from names shared by many owners to names owned by one, because a type's table is as long as the
// highest id it uses. "planned" marks a name reserved for a member that does not exist yet.
package members

const (
	// universal — every builtin type
	IsTrue      = 1
	String      = 2
	Format      = 3
	Copy        = 4
	Freeze      = 5
	Or          = 6 // planned
	TypeName    = 7 // planned
	IsImmutable = 8 // planned
	// 9…16 reserved

	// common domain properties — identity, money, ranges (embedder and future domain types; one id shared by all)
	ID               = 17 // domain property; embedder use
	UID              = 18 // domain property; embedder use
	UUID             = 19 // domain property; embedder use
	CID              = 20 // domain property; embedder use
	PID              = 21
	Hash             = 22 // domain property; embedder use
	TID              = 23 // domain property; embedder use
	TransactionID    = 24 // domain property; embedder use
	ReferenceID      = 25 // domain property; embedder use
	OriginationID    = 26 // domain property; embedder use
	AccountID        = 27 // domain property; embedder use
	AccountNumber    = 28 // domain property; embedder use
	ProductBalanceID = 29 // domain property; embedder use
	ParentID         = 30 // domain property; embedder use
	CustomerID       = 31 // domain property; embedder use
	ProductID        = 32 // domain property; embedder use
	Amount           = 33 // domain property; embedder use
	Currency         = 34 // domain property; embedder use
	Quantity         = 35 // domain property; embedder use
	Price            = 36 // domain property; embedder use
	Fee              = 37 // domain property; embedder use
	Interest         = 38 // domain property; embedder use
	Principal        = 39 // domain property; embedder use
	Tax              = 40 // domain property; embedder use
	Total            = 41 // domain property; embedder use
	Balance          = 42
	Debit            = 43 // domain property; embedder use
	Credit           = 44 // domain property; embedder use
	Ledger           = 45 // domain property; embedder use
	Reversal         = 46 // domain property; embedder use
	Scheduled        = 47 // domain property; embedder use
	ValueTime        = 48 // domain property; embedder use
	Status           = 49 // domain property; embedder use
	State            = 50 // domain property; embedder use
	CreatedAt        = 51 // domain property; embedder use
	UpdatedAt        = 52 // domain property; embedder use
	EffectiveDate    = 53 // domain property; embedder use
	PostingDate      = 54 // domain property; embedder use
	DueDate          = 55 // domain property; embedder use
	MaturityDate     = 56 // domain property; embedder use
	StartDate        = 57 // domain property; embedder use
	EndDate          = 58 // domain property; embedder use
	From             = 59 // domain property; embedder use
	To               = 60 // domain property; embedder use
	Type             = 61 // domain property; embedder use
	Description      = 62 // domain property; embedder use
	Body             = 63 // domain property; embedder use
	DateStr          = 64 // domain property; embedder use
	TimeStr          = 65 // domain property; embedder use
	DateTimeStr      = 66 // domain property; embedder use
	MonthStr         = 67 // domain property; embedder use
	Code             = 68 // domain property; embedder use
	Key              = 69 // domain property; embedder use
	Version          = 70 // domain property; embedder use
	Label            = 71 // domain property; embedder use
	Note             = 72 // domain property; embedder use
	Tags             = 73 // domain property; embedder use
	Metadata         = 74 // domain property; embedder use
	// 75…104 reserved

	// conversions — by breadth
	Runes       = 105
	Int         = 106
	Bool        = 107
	Float       = 108
	Time        = 109
	Decimal     = 110
	Date        = 111
	Array       = 112
	Bytes       = 113
	Byte        = 114
	Rune        = 115
	Dict        = 116
	Record      = 117
	Range       = 118
	Set         = 119 // planned
	Ints        = 120 // planned
	Floats      = 121 // planned
	BigInt      = 122 // planned
	BigFloat    = 123 // planned
	BigRational = 124 // planned
	Duration    = 125 // planned
	Regex       = 126 // planned
	// 127…144 reserved

	// collections — size, search, callbacks (array, bytes, runes, string, range, dict)
	Len      = 145
	IsEmpty  = 146
	Contains = 147
	Index    = 148
	Find     = 149
	Count    = 150
	All      = 151
	Any      = 152
	None     = 153 // planned
	ForEach  = 154
	Reduce   = 155
	Keep     = 156
	Map      = 157
	GroupBy  = 158 // planned
	CountBy  = 159 // planned
	MinBy    = 160 // planned
	MaxBy    = 161 // planned
	Remove   = 162
	// 163…176 reserved

	// sequences — order, ends, slices (array, bytes, runes, string, range)
	First       = 177
	Last        = 178
	IndexLast   = 179
	Min         = 180
	Max         = 181
	Slice       = 182
	Reverse     = 183
	Sort        = 184
	IsSorted    = 185 // planned
	Unique      = 186
	Dedup       = 187
	Chunk       = 188
	Take        = 189 // planned
	Drop        = 190 // planned
	Pop         = 191 // planned
	PopFirst    = 192 // planned
	Window      = 193 // planned
	Enumerate   = 194 // planned
	Zip         = 195 // planned
	Unzip       = 196 // planned
	Intersperse = 197 // planned
	Cycle       = 198 // planned
	SortBy      = 199 // planned
	Shuffle     = 200 // planned
	Rotate      = 201 // planned
	// 202…216 reserved

	// sequences — edit and text-trait verbs (array, bytes, runes, string)
	Append       = 217
	Prepend      = 218
	Push         = 219
	PushFirst    = 220
	Insert       = 221
	Splice       = 222
	Repeat       = 223
	PadStart     = 224
	PadEnd       = 225
	PadCenter    = 226 // planned
	Trim         = 227
	TrimStart    = 228
	TrimEnd      = 229
	HasPrefix    = 230
	HasSuffix    = 231
	RemovePrefix = 232
	RemoveSuffix = 233
	RemoveAt     = 234 // planned
	Replace      = 235
	Split        = 236
	FlatMap      = 237
	// 238…248 reserved

	// text (string, runes; is_ascii/is_valid also byte, rune, bytes)
	IsASCII     = 249
	IsValid     = 250
	IsLetter    = 251 // planned
	IsDigit     = 252 // planned
	IsNumber    = 253 // planned
	IsSpace     = 254 // planned
	IsUpper     = 255 // planned
	IsLower     = 256 // planned
	IsTitle     = 257 // planned
	IsPunct     = 258 // planned
	IsSymbol    = 259 // planned
	IsControl   = 260 // planned
	IsGraphic   = 261 // planned
	IsPrint     = 262 // planned
	IsMark      = 263 // planned
	SplitLines  = 264
	Words       = 265 // planned
	Partition   = 266
	Lower       = 267
	Upper       = 268
	CaseFold    = 269
	Normalize   = 270 // planned
	TitleCase   = 271
	CamelCase   = 272
	PascalCase  = 273
	SnakeCase   = 274
	KebabCase   = 275
	Truncate    = 276
	Graphemes   = 277 // planned
	GraphemeLen = 278 // planned
	Quote       = 279 // planned
	Unquote     = 280 // planned
	ParseArray  = 281 // planned
	Hex         = 282 // planned
	Base64      = 283 // planned
	// 284…304 reserved

	// containers — array, dict, range extras
	CopyShallow   = 305
	FreezeShallow = 306
	Flatten       = 307
	Join          = 308
	Sum           = 309
	Avg           = 310
	Keys          = 311
	Values        = 312
	Items         = 313 // planned
	Merge         = 314
	MergeInPlace  = 315
	Pick          = 316 // planned
	Omit          = 317 // planned
	Invert        = 318 // planned
	RecordView    = 319
	Union         = 320 // planned
	Intersect     = 321 // planned
	Diff          = 322 // planned
	IsSubsetOf    = 323 // planned
	IsSupersetOf  = 324 // planned
	// 325…336 reserved

	// sequences — _in_place twins and views (array, bytes, runes; keep/remove also dict)
	KeepInPlace         = 337
	RemoveInPlace       = 338
	AppendInPlace       = 339
	PrependInPlace      = 340
	PushInPlace         = 341
	PushFirstInPlace    = 342
	InsertInPlace       = 343
	SpliceInPlace       = 344
	PadStartInPlace     = 345
	PadEndInPlace       = 346
	TrimInPlace         = 347
	TrimStartInPlace    = 348
	TrimEndInPlace      = 349
	RemovePrefixInPlace = 350
	RemoveSuffixInPlace = 351
	ReplaceInPlace      = 352
	ReverseInPlace      = 353
	SortInPlace         = 354
	UniqueInPlace       = 355
	DedupInPlace        = 356
	PadCenterInPlace    = 357 // planned
	SortByInPlace       = 358 // planned
	ShuffleInPlace      = 359 // planned
	RotateInPlace       = 360 // planned
	SliceView           = 361
	ChunkView           = 362
	// 363…376 reserved

	// numbers — shared (int, float, decimal, fin.year_fraction; abs/is_nan/is_inf/sqrt/pow/copy_sign also math)
	Abs        = 377
	Sign       = 378
	IsZero     = 379
	IsPositive = 380
	IsNegative = 381
	IsEven     = 382 // planned
	IsOdd      = 383 // planned
	IsNaN      = 384
	IsInf      = 385
	IsFinite   = 386 // planned
	Clamp      = 387
	Sqrt       = 388
	Pow        = 389
	CopySign   = 390
	Floor      = 391
	Ceil       = 392
	Trunc      = 393
	Round      = 394
	Gcd        = 395 // planned
	Lcm        = 396 // planned
	// 397…408 reserved

	// functions (builtin_function, compiled_function, builtin_closure; name also obj os.file)
	Name       = 409
	Arity      = 410 // planned
	IsVariadic = 411 // planned
	IsPure     = 412 // planned
	Memoize    = 413 // planned
	// 414…416 reserved

	// error
	Kind          = 417
	Value         = 418
	Message       = 419 // planned
	Trace         = 420 // planned
	IsUser        = 421
	IsRuntime     = 422
	IsRequirement = 423
	// 424…432 reserved

	// fin.* types (rate also fin module)
	At          = 433
	Bands       = 434
	Rate        = 435
	Charge      = 436
	ChargeParts = 437
	Accrue      = 438
	AccrueParts = 439
	Apply       = 440
	Bounds      = 441
	Terms       = 442
	// 443…448 reserved

	// time and date (time, date; *_in_month/_year/leap also times; hour…nanosecond also times constants)
	Components      = 449
	Year            = 450
	Month           = 451
	Day             = 452
	Hour            = 453
	Minute          = 454
	Second          = 455
	Nanosecond      = 456
	WeekDay         = 457
	WeekDayName     = 458
	MonthName       = 459
	YearDay         = 460
	Quarter         = 461 // planned
	ISOWeek         = 462 // planned
	DaysInMonth     = 463
	DaysInYear      = 464
	IsLeapYear      = 465
	IsWeekend       = 466 // planned
	IsWeekday       = 467 // planned
	IsHoliday       = 468 // planned
	IsBusinessDay   = 469 // planned
	AddYears        = 470
	AddMonths       = 471
	AddQuarters     = 472 // planned
	AddWeeks        = 473 // planned
	AddDays         = 474
	AddBusinessDays = 475 // planned
	AddHours        = 476 // planned
	AddMinutes      = 477 // planned
	AddSeconds      = 478 // planned
	StartOfYear     = 479 // planned
	EndOfYear       = 480 // planned
	StartOfQuarter  = 481 // planned
	EndOfQuarter    = 482 // planned
	StartOfMonth    = 483
	EndOfMonth      = 484
	StartOfWeek     = 485 // planned
	EndOfWeek       = 486 // planned
	StartOfDay      = 487 // planned
	EndOfDay        = 488 // planned
	IsEndOfMonth    = 489
	YearsSince      = 490 // planned
	MonthsSince     = 491
	DaysSince       = 492 // planned
	Unix            = 493
	UnixMs          = 494
	UnixMicro       = 495
	UnixNano        = 496
	UTC             = 497
	InZone          = 498
	ZoneName        = 499
	ZoneOffset      = 500
	DateIn          = 501
	TimeIn          = 502
	TimeMs          = 503
	TimeMicro       = 504
	TimeNano        = 505
	// 506…536 reserved

	// numbers — decimal
	RoundHalfEven     = 537
	RoundHalfUp       = 538
	RoundHalfDown     = 539
	RoundUp           = 540
	RoundDown         = 541
	RoundCeiling      = 542
	RoundFloor        = 543
	RoundSignificant  = 544
	RoundToMultiple   = 545
	Rescale           = 546
	Scale             = 547
	ScaleByPow10      = 548
	Canonical         = 549
	IntegerDigits     = 550
	SignificantDigits = 551
	IsInteger         = 552
	Negate            = 553
	NextUp            = 554
	NextDown          = 555
	CanFit            = 556
	QuoRem            = 557
	DivRound          = 558
	MulRound          = 559
	MulAddRound       = 560
	MulDivRound       = 561
	MulPercentRound   = 562
	PowRound          = 563
	PowRationalRound  = 564
	SqrtRound         = 565
	NthRootRound      = 566
	ExpRound          = 567
	LnRound           = 568
	Log2Round         = 569
	Log10Round        = 570
	Allocate          = 571
	AllocateResidual  = 572
	SplitResidual     = 573
	ErrorDetails      = 574
	// 575…584 reserved

	// modules — shared by several (base64, hex, json)
	Encode = 585
	Decode = 586
	// 587…592 reserved

	// module fmt
	Print   = 593
	Println = 594
	// 595…600 reserved

	// module json
	Indent     = 601
	HTMLEscape = 602
	// 603…608 reserved

	// module regexp
	ReMatch   = 609
	ReFind    = 610
	ReReplace = 611
	ReSplit   = 612
	ReCompile = 613
	// 614…616 reserved

	// module base64
	URLEncode    = 617
	URLDecode    = 618
	RawEncode    = 619
	RawDecode    = 620
	RawURLEncode = 621
	RawURLDecode = 622
	// 623…624 reserved

	// module rand
	Seed      = 625
	Rand      = 626
	IntN      = 627
	ExpFloat  = 628
	NormFloat = 629
	Perm      = 630
	Read      = 631
	// 632…640 reserved

	// module times — functions
	Now                 = 641
	Since               = 642
	Until               = 643
	Sleep               = 644
	ParseDuration       = 645
	DurationString      = 646
	DurationHours       = 647
	DurationMinutes     = 648
	DurationSeconds     = 649
	DurationNanoseconds = 650
	FromUnixMs          = 651
	FromUnixMicro       = 652
	FromUnixNano        = 653
	// 654…664 reserved

	// module times — constants
	Millisecond = 665
	Microsecond = 666
	January     = 667
	February    = 668
	March       = 669
	April       = 670
	May         = 671
	June        = 672
	July        = 673
	August      = 674
	September   = 675
	October     = 676
	November    = 677
	December    = 678
	// 679…688 reserved

	// module math — functions
	Mod       = 689
	Remainder = 690
	Dim       = 691
	Hypot     = 692
	Cbrt      = 693
	Exp       = 694
	Exp2      = 695
	Expm1     = 696
	Log       = 697
	Log10     = 698
	Log2      = 699
	Log1p     = 700
	Logb      = 701
	Ilogb     = 702
	Ldexp     = 703
	Pow10     = 704
	Sin       = 705
	Cos       = 706
	Tan       = 707
	Asin      = 708
	Acos      = 709
	Atan      = 710
	Atan2     = 711
	Sinh      = 712
	Cosh      = 713
	Tanh      = 714
	Asinh     = 715
	Acosh     = 716
	Atanh     = 717
	Signbit   = 718
	NextAfter = 719
	Inf       = 720
	NaN       = 721
	Erf       = 722
	Erfc      = 723
	Gamma     = 724
	J0        = 725
	J1        = 726
	Jn        = 727
	Y0        = 728
	Y1        = 729
	Yn        = 730
	// 731…744 reserved

	// module math — constants
	Pi                     = 745
	E                      = 746
	Phi                    = 747
	Sqrt2                  = 748
	SqrtE                  = 749
	SqrtPi                 = 750
	SqrtPhi                = 751
	Ln2                    = 752
	Log2e                  = 753
	Ln10                   = 754
	Log10e                 = 755
	MaxInt                 = 756
	MinInt                 = 757
	MaxInt8                = 758
	MinInt8                = 759
	MaxInt16               = 760
	MinInt16               = 761
	MaxInt32               = 762
	MinInt32               = 763
	MaxInt64               = 764
	MinInt64               = 765
	MaxFloat32             = 766
	SmallestNonzeroFloat32 = 767
	MaxFloat64             = 768
	SmallestNonzeroFloat64 = 769
	// 770…776 reserved

	// module fin — functions
	YearFraction             = 777
	YearFractionBetween      = 778
	YearFractionBetweenFinal = 779
	DaysBetween              = 780
	DaysBetweenFinal         = 781
	Conventions              = 782
	ToPercent                = 783
	FromPercent              = 784
	ToBasisPoints            = 785
	FromBasisPoints          = 786
	ApplyRate                = 787
	ExactProduct             = 788
	PerYear                  = 789
	NominalToEffective       = 790
	EffectiveToNominal       = 791
	CompoundFactor           = 792
	CompoundFactorFor        = 793
	DiscountFactor           = 794
	DiscountFactorFor        = 795
	AccrueSimple             = 796
	AccrueCompound           = 797
	AnnuityFactorFv          = 798
	AnnuityFactorPv          = 799
	Payment                  = 800
	PresentValue             = 801
	FutureValue              = 802
	Periods                  = 803
	PrincipalPart            = 804
	ChargePart               = 805
	Npv                      = 806
	Irr                      = 807
	Xnpv                     = 808
	Xirr                     = 809
	Mirr                     = 810
	Root                     = 811
	DefaultSolver            = 812
	DiscountPrice            = 813
	DiscountRate             = 814
	DiscountYield            = 815
	StraightLine             = 816
	DecliningBalance         = 817
	DecliningRateFromFactor  = 818
	DecliningRateFromSalvage = 819
	SumOfDigits              = 820
	TieredRates              = 821
	TieredCharges            = 822
	DatedRates               = 823
	DatedCharges             = 824
	Brackets                 = 825
	IsYearFraction           = 826
	IsTieredRates            = 827
	IsTieredCharges          = 828
	IsDatedRates             = 829
	IsDatedCharges           = 830
	// 831…848 reserved

	// module os — functions
	Args         = 849
	Exit         = 850
	Environ      = 851
	GetEnv       = 852
	SetEnv       = 853
	UnsetEnv     = 854
	LookupEnv    = 855
	ExpandEnv    = 856
	ClearEnv     = 857
	Hostname     = 858
	TempDir      = 859
	GetWd        = 860
	Chdir        = 861
	Open         = 862
	OpenFile     = 863
	Create       = 864
	ReadFile     = 865
	Stat         = 866
	RemoveAll    = 867
	Rename       = 868
	Mkdir        = 869
	MkdirAll     = 870
	Link         = 871
	Symlink      = 872
	ReadLink     = 873
	Chmod        = 874
	Chown        = 875
	Lchown       = 876
	Exec         = 877
	ExecLookPath = 878
	StartProcess = 879
	FindProcess  = 880
	GetPID       = 881
	GetPpid      = 882
	GetUID       = 883
	GetEuid      = 884
	GetGid       = 885
	GetEgid      = 886
	GetGroups    = 887
	GetPageSize  = 888
	// 889…904 reserved

	// module os — constants
	Platform          = 905
	Arch              = 906
	PathSeparator     = 907
	PathListSeparator = 908
	DevNull           = 909
	ORd               = 910
	OWr               = 911
	ORdwr             = 912
	OAppend           = 913
	OCreate           = 914
	OExcl             = 915
	OSync             = 916
	OTrunc            = 917
	SeekSet           = 918
	SeekCur           = 919
	SeekEnd           = 920
	ModeDir           = 921
	ModeAppend        = 922
	ModeExclusive     = 923
	ModeTemporary     = 924
	ModeSymlink       = 925
	ModeDevice        = 926
	ModeNamedPipe     = 927
	ModeSocket        = 928
	ModeSetUID        = 929
	ModeSetGui        = 930
	ModeCharDevice    = 931
	ModeSticky        = 932
	ModeType          = 933
	ModePerm          = 934
	// 935…944 reserved

	// objects — compiled regexp, os file/process/cmd (record-backed today; members once they become types)
	Match          = 945
	Close          = 946
	Write          = 947
	WriteString    = 948
	Sync           = 949
	Seek           = 950
	ReadDirNames   = 951
	Kill           = 952
	Signal         = 953
	Wait           = 954
	Release        = 955
	Exited         = 956
	Success        = 957
	Run            = 958
	Start          = 959
	Output         = 960
	CombinedOutput = 961
	Process        = 962
	SetDir         = 963
	SetPath        = 964
	// 965…976 reserved

	// alias spellings — embedder use only; never builtin members (builtins keep the one convention spelling shown)
	Empty     = 977 // alias of is_empty; embedder use only
	Filter    = 978 // alias of keep; embedder use only
	Sorted    = 979 // alias of sort; embedder use only
	Immutable = 980 // alias of freeze / is_immutable(x); embedder use only
	LeapYear  = 981 // alias of is_leap_year; embedder use only
	Pos       = 982 // alias of is_positive; embedder use only
	Positive  = 983 // alias of is_positive; embedder use only
	Neg       = 984 // alias of is_negative / negate; embedder use only
	Negative  = 985 // alias of is_negative / negate; embedder use only
	Sod       = 986 // alias of start_of_day; embedder use only
	Som       = 987 // alias of start_of_month; embedder use only
	Soy       = 988 // alias of start_of_year; embedder use only
	Eod       = 989 // alias of end_of_day; embedder use only
	Eom       = 990 // alias of end_of_month; embedder use only
	Eoy       = 991 // alias of end_of_year; embedder use only
	// 992…1000 reserved
)
