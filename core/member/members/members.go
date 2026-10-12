// Package members is the enum of builtin member ids: one untyped constant per member name a builtin type or stdlib
// module answers — a name is bound only once something answers it (an embedder binds its own from 2048 with
// member.Define). It holds constants only — the kind (member.ID), the id ↔ name table and
// every function live in core/member — so an identifier here can only ever collide with another member name.
//
// FROZEN: a released id is never changed, removed or reused. A new name takes the next free id inside its block's
// reserved range, or follows the last block once that range is full; a retired name keeps its line, commented out.
// Blocks run from names shared by many owners to names owned by one, because a type's table is as long as the
// highest id it uses.
package members

const (
	// universal — every builtin type
	IsTrue = 1
	String = 2
	Format = 3
	Copy   = 4
	Freeze = 5
	// 6…16 reserved

	// shared property names (pid: exec.process; balance: fin)
	PID     = 17
	Balance = 18
	// 19…104 reserved

	// conversions — by breadth
	Runes   = 105
	Int     = 106
	Bool    = 107
	Float   = 108
	Time    = 109
	Decimal = 110
	Date    = 111
	Array   = 112
	Bytes   = 113
	Byte    = 114
	Rune    = 115
	Dict    = 116
	Record  = 117
	Range   = 118
	// 119…144 reserved

	// collections — size, search, callbacks (array, bytes, runes, string, range, dict)
	Len      = 145
	IsEmpty  = 146
	Contains = 147
	Index    = 148
	Find     = 149
	Count    = 150
	All      = 151
	Any      = 152
	ForEach  = 153
	Reduce   = 154
	Keep     = 155
	Map      = 156
	Remove   = 157
	// 158…176 reserved

	// sequences — order, ends, slices (array, bytes, runes, string, range)
	First     = 177
	Last      = 178
	IndexLast = 179
	Min       = 180
	Max       = 181
	Slice     = 182
	Reverse   = 183
	Sort      = 184
	Unique    = 185
	Dedup     = 186
	Chunk     = 187
	Shuffle   = 188
	// 189…216 reserved

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
	Trim         = 226
	TrimStart    = 227
	TrimEnd      = 228
	HasPrefix    = 229
	HasSuffix    = 230
	RemovePrefix = 231
	RemoveSuffix = 232
	Replace      = 233
	Split        = 234
	FlatMap      = 235
	// 236…248 reserved

	// text (string, runes; is_ascii/is_valid also byte, rune, bytes)
	IsASCII    = 249
	IsValid    = 250
	SplitLines = 251
	Partition  = 252
	Lower      = 253
	Upper      = 254
	CaseFold   = 255
	TitleCase  = 256
	CamelCase  = 257
	PascalCase = 258
	SnakeCase  = 259
	KebabCase  = 260
	Truncate   = 261
	// 262…304 reserved

	// containers — array, dict, range extras
	CopyShallow   = 305
	FreezeShallow = 306
	Flatten       = 307
	Join          = 308
	Sum           = 309
	Avg           = 310
	Keys          = 311
	Values        = 312
	Merge         = 313
	MergeInPlace  = 314
	RecordView    = 315
	// 316…336 reserved

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
	ShuffleInPlace      = 357
	SliceView           = 358
	ChunkView           = 359
	// 360…376 reserved

	// numbers — shared (int, float, decimal, fin.year_fraction)
	Abs        = 377
	Sign       = 378
	IsZero     = 379
	IsPositive = 380
	IsNegative = 381
	IsNaN      = 382
	IsInf      = 383
	Clamp      = 384
	Sqrt       = 385
	Pow        = 386
	CopySign   = 387
	Round      = 388
	Ln         = 389
	// 390…408 reserved

	// functions (builtin_function, compiled_function, builtin_closure; name also fs.file)
	Name = 409
	// 410…416 reserved

	// error
	Kind          = 417
	Value         = 418
	IsUser        = 419
	IsRuntime     = 420
	IsRequirement = 421
	// 422…432 reserved

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

	// time and date (time, date)
	Components   = 449
	Year         = 450
	Month        = 451
	Day          = 452
	Hour         = 453
	Minute       = 454
	Second       = 455
	Nanosecond   = 456
	WeekDay      = 457
	WeekDayName  = 458
	MonthName    = 459
	YearDay      = 460
	DaysInMonth  = 461
	DaysInYear   = 462
	IsLeapYear   = 463
	AddYears     = 464
	AddMonths    = 465
	AddDays      = 466
	StartOfMonth = 467
	EndOfMonth   = 468
	IsEndOfMonth = 469
	MonthsSince  = 470
	Unix         = 471
	UnixMs       = 472
	UnixMicro    = 473
	UnixNano     = 474
	UTC          = 475
	InZone       = 476
	ZoneName     = 477
	ZoneOffset   = 478
	DateIn       = 479
	TimeIn       = 480
	TimeMs       = 481
	TimeMicro    = 482
	TimeNano     = 483
	// 484…505 reserved

	// durations: an int counts nanoseconds when its member says so (int, float build; int reads; string parses)
	Nanoseconds    = 506
	Microseconds   = 507
	Milliseconds   = 508
	Seconds        = 509
	Minutes        = 510
	Hours          = 511
	Days           = 512
	Weeks          = 513
	InMicroseconds = 514
	InMilliseconds = 515
	InSeconds      = 516
	InMinutes      = 517
	InHours        = 518
	InDays         = 519
	InWeeks        = 520
	// 521…536 reserved

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

	// module io
	Print    = 593
	ReadLine = 594
	// 595…600 reserved

	// module json
	Indent     = 601
	HTMLEscape = 602
	// 603…608 reserved

	// module re
	Pattern = 609
	// 610…616 reserved

	// module base64
	URLEncode    = 617
	URLDecode    = 618
	RawEncode    = 619
	RawDecode    = 620
	RawURLEncode = 621
	RawURLDecode = 622
	// 623…624 reserved

	// module rand
	ExpFloat  = 625
	NormFloat = 626
	Perm      = 627
	Read      = 628
	Generator = 629
	// 630…640 reserved

	// module clock (formerly times); duration_string is int's
	Now            = 641
	Since          = 642
	Until          = 643
	Sleep          = 644
	DurationString = 645
	// 646…664 reserved

	// module math — functions
	Remainder = 665
	Dim       = 666
	Hypot     = 667
	Cbrt      = 668
	Exp       = 669
	Exp2      = 670
	Expm1     = 671
	Log10     = 672
	Log2      = 673
	Log1p     = 674
	Logb      = 675
	Ilogb     = 676
	Ldexp     = 677
	Pow10     = 678
	Sin       = 679
	Cos       = 680
	Tan       = 681
	Asin      = 682
	Acos      = 683
	Atan      = 684
	Atan2     = 685
	Sinh      = 686
	Cosh      = 687
	Tanh      = 688
	Asinh     = 689
	Acosh     = 690
	Atanh     = 691
	NextAfter = 692
	Inf       = 693
	NaN       = 694
	Erf       = 695
	Erfc      = 696
	Gamma     = 697
	J0        = 698
	J1        = 699
	Jn        = 700
	Y0        = 701
	Y1        = 702
	Yn        = 703
	// 704…744 reserved

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

	// module path (os block reserve)
	Base          = 889
	Dir           = 890
	Ext           = 891
	Clean         = 892
	Rel           = 893
	IsAbs         = 894
	RandGenerator = 895 // os.rand_generator(): a rand.generator over crypto/rand
	// 896…904 reserved

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

	// module-owned types — re.pattern, fs.file, exec.command, exec.process
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
	Run            = 956
	Start          = 957
	Output         = 958
	CombinedOutput = 959
	Process        = 960
	SetDir         = 961
	SetPath        = 962
	// 963…976 reserved

	// free
	// 977…1000 reserved
)
