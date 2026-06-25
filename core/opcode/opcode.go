package opcode

type Opcode byte

const (
	AbortCheck                  = Opcode(0)  // Poll VM abort flag; return control to host when set
	UnaryBitNot                 = Opcode(1)  // Unary bitwise NOT
	Pop                         = Opcode(2)  // Pop
	PushTrue                    = Opcode(3)  // Push true
	PushFalse                   = Opcode(4)  // Push false
	Equal                       = Opcode(5)  // Equal ==
	NotEqual                    = Opcode(6)  // Not equal !=
	UnaryNeg                    = Opcode(7)  // Unary negation -
	UnaryNot                    = Opcode(8)  // Logical not !
	JumpFalsy                   = Opcode(9)  // Jump if falsy
	AndJump                     = Opcode(10) // Logical AND jump
	OrJump                      = Opcode(11) // Logical OR jump
	Jump                        = Opcode(12) // Jump
	PushUndefined               = Opcode(13) // Push undefined
	Array                       = Opcode(14) // Array object
	Record                      = Opcode(15) // Record object
	Contains                    = Opcode(16) // Contains operation (x in y)
	Immutable                   = Opcode(17) // Immutable object
	Index                       = Opcode(18) // Index operation
	SliceIndex                  = Opcode(19) // Slice operation
	Call                        = Opcode(20) // Call function
	Return                      = Opcode(21) // Return
	LoadGlobal                  = Opcode(22) // Get global variable
	StoreGlobal                 = Opcode(23) // Set global variable
	SetSelGlobal                = Opcode(24) // Set global variable using selectors
	LoadLocal                   = Opcode(25) // Get local variable
	StoreLocal                  = Opcode(26) // Set local variable
	DefineLocal                 = Opcode(27) // Define local variable
	SetSelLocal                 = Opcode(28) // Set local variable using selectors
	GetFreePtr                  = Opcode(29) // Get free variable pointer object
	LoadFree                    = Opcode(30) // Get free variables
	StoreFree                   = Opcode(31) // Set free variables
	GetLocalPtr                 = Opcode(32) // Get local variable as a pointer
	SetSelFree                  = Opcode(33) // Set free variables using selectors
	GetBuiltinFunction          = Opcode(34) // Get builtin function
	Closure                     = Opcode(35) // Push closure
	IteratorInit                = Opcode(36) // Iterator init
	IteratorNext                = Opcode(37) // Iterator next
	IteratorKey                 = Opcode(38) // Iterator key
	IteratorValue               = Opcode(39) // Iterator value
	BinaryOp                    = Opcode(40) // Binary operation
	Suspend                     = Opcode(41) // Suspend VM
	Select                      = Opcode(42) // Select operation
	MethodCall                  = Opcode(43) // Call method on object
	SliceIndexStep              = Opcode(44) // Slice with step
	Format                      = Opcode(45) // Format value with pre-parsed FormatSpec static
	FormatDyn                   = Opcode(46) // Format value with runtime-built FormatSpec string popped from the stack
	Defer                       = Opcode(47) // Register deferred call: pop callee + N args, store on current frame
	DeferMethod                 = Opcode(48) // Register deferred method call: pop receiver + N args; method name from static strings[methodIdx]
	ImportBuiltinModule         = Opcode(49) // Import builtin module by static ID
	StaticPrimitiveValue        = Opcode(50) // Push static primitive value (bool, nil)
	StaticDecimalValue          = Opcode(51) // Push static decimal value
	StaticStringValue           = Opcode(52) // Push static string value
	StaticRunesValue            = Opcode(53) // Push static runes value
	StaticBytesValue            = Opcode(54) // Push static bytes value
	StaticTimeValue             = Opcode(55) // Push static time value
	StaticFormatSpecValue       = Opcode(56) // Push static FormatSpec value
	StaticCompiledFunctionValue = Opcode(57) // Push static compiled function value
	// 58...255 are reserved for future use
)

var names = [...]string{
	AbortCheck:                  "ABORTCHK",
	Pop:                         "POP",
	PushTrue:                    "TRUE",
	PushFalse:                   "FALSE",
	UnaryBitNot:                 "BITNOT",
	Equal:                       "EQL",
	NotEqual:                    "NEQ",
	UnaryNeg:                    "NEG",
	UnaryNot:                    "NOT",
	JumpFalsy:                   "JMPF",
	AndJump:                     "ANDJMP",
	OrJump:                      "ORJMP",
	Jump:                        "JMP",
	PushUndefined:               "UNDEF",
	LoadGlobal:                  "GETG",
	StoreGlobal:                 "SETG",
	SetSelGlobal:                "SETSG",
	Array:                       "ARR",
	Record:                      "RECORD",
	Immutable:                   "IMMUT",
	Index:                       "INDEX",
	SliceIndex:                  "SLICE",
	Call:                        "CALL",
	SliceIndexStep:              "SLICESTEP",
	Return:                      "RET",
	LoadLocal:                   "GETL",
	StoreLocal:                  "SETL",
	DefineLocal:                 "DEFL",
	SetSelLocal:                 "SETSL",
	GetBuiltinFunction:          "BUILTIN",
	Closure:                     "CLOSURE",
	GetFreePtr:                  "GETFP",
	LoadFree:                    "GETF",
	StoreFree:                   "SETF",
	GetLocalPtr:                 "GETLP",
	SetSelFree:                  "SETSF",
	IteratorInit:                "ITER",
	IteratorNext:                "ITNXT",
	IteratorKey:                 "ITKEY",
	IteratorValue:               "ITVAL",
	BinaryOp:                    "BINARYOP",
	Suspend:                     "SUSPEND",
	Select:                      "SELECT",
	MethodCall:                  "MCALL",
	Contains:                    "CONTAINS",
	Format:                      "FMT",
	FormatDyn:                   "FMTDYN",
	Defer:                       "DEFER",
	DeferMethod:                 "DEFERM",
	ImportBuiltinModule:         "IMPMOD",
	StaticPrimitiveValue:        "STATICPRIM",
	StaticDecimalValue:          "STATICDEC",
	StaticStringValue:           "STATICSTR",
	StaticRunesValue:            "STATICRUNES",
	StaticBytesValue:            "STATICBYTES",
	StaticTimeValue:             "STATICTIME",
	StaticFormatSpecValue:       "STATICFMTSPEC",
	StaticCompiledFunctionValue: "STATICCOMPFN",
}

// describes the number and shape of opcode operands
var operands = [...][]int{
	AbortCheck:                  {},
	Pop:                         {},
	PushTrue:                    {},
	PushFalse:                   {},
	UnaryBitNot:                 {},
	Equal:                       {},
	NotEqual:                    {},
	UnaryNeg:                    {},
	UnaryNot:                    {},
	JumpFalsy:                   {2}, // new pos
	AndJump:                     {2}, // new pos
	OrJump:                      {2}, // new pos
	Jump:                        {2}, // new pos
	PushUndefined:               {},
	LoadGlobal:                  {2},    // index
	StoreGlobal:                 {2},    // index
	SetSelGlobal:                {2, 1}, // index, num selectors
	Array:                       {2},    // num elements (inline init)
	Record:                      {2},    // num elements (inline init)
	Immutable:                   {},
	Index:                       {},
	SliceIndex:                  {},
	Call:                        {1, 1}, // num args, is spread (0 or 1)
	SliceIndexStep:              {},
	Return:                      {1},    // has result (0 or 1)
	LoadLocal:                   {1},    // index
	StoreLocal:                  {1},    // index
	DefineLocal:                 {1},    // index
	SetSelLocal:                 {1, 1}, // index, num selectors
	GetBuiltinFunction:          {1},    // index
	Closure:                     {2, 1}, // num args, is spread (0 or 1)
	GetFreePtr:                  {1},    // index
	LoadFree:                    {1},    // index
	StoreFree:                   {1},    // index
	GetLocalPtr:                 {1},    // index
	SetSelFree:                  {1, 1}, // index, num selectors
	IteratorInit:                {},
	IteratorNext:                {},
	IteratorKey:                 {},
	IteratorValue:               {},
	BinaryOp:                    {1}, // token
	Suspend:                     {},
	Select:                      {},
	MethodCall:                  {2, 1, 1}, // method const index, num args, spread
	Contains:                    {},
	Format:                      {2},    // format spec constant index
	FormatDyn:                   {},     // pops spec string and value from stack, pushes formatted string
	Defer:                       {1},    // numArgs (callee + args popped from stack at runtime)
	DeferMethod:                 {2, 1}, // method const index, num args
	ImportBuiltinModule:         {1},    // module static ID
	StaticPrimitiveValue:        {2},    // index
	StaticDecimalValue:          {2},    // index
	StaticStringValue:           {2},    // index
	StaticRunesValue:            {2},    // index
	StaticBytesValue:            {2},    // index
	StaticTimeValue:             {2},    // index
	StaticFormatSpecValue:       {2},    // index
	StaticCompiledFunctionValue: {2},    // index
}

var widths [256]int

func (op Opcode) Width() int {
	return widths[op]
}

func (op Opcode) Byte() byte {
	return byte(op)
}

func (op Opcode) String() string {
	return names[op]
}

func (op Opcode) Operands() []int {
	return operands[op]
}

func init() {
	for op, ws := range operands {
		for _, w := range ws {
			widths[op] += w
		}
	}
}
