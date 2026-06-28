package opcode

type Opcode byte

const (
	AbortCheck                 = Opcode(0)  // Poll VM abort flag; return control to host when set
	UnaryBitNot                = Opcode(1)  // Unary bitwise NOT
	Pop                        = Opcode(2)  // Pop
	PushTrue                   = Opcode(3)  // Push true
	PushFalse                  = Opcode(4)  // Push false
	Equal                      = Opcode(5)  // Equal ==
	NotEqual                   = Opcode(6)  // Not equal !=
	UnaryNeg                   = Opcode(7)  // Unary negation -
	UnaryNot                   = Opcode(8)  // Logical not !
	JumpFalsy                  = Opcode(9)  // Jump if falsy
	AndJump                    = Opcode(10) // Logical AND jump
	OrJump                     = Opcode(11) // Logical OR jump
	Jump                       = Opcode(12) // Jump
	PushUndefined              = Opcode(13) // Push undefined
	MakeArray                  = Opcode(14) // Array object
	MakeRecord                 = Opcode(15) // Record object
	Contains                   = Opcode(16) // Contains operation (x in y)
	Immutable                  = Opcode(17) // Immutable object
	AccessIndex                = Opcode(18) // Index operation
	Slice                      = Opcode(19) // Slice operation
	CallFunction               = Opcode(20) // Call function
	Return                     = Opcode(21) // Return
	LoadGlobal                 = Opcode(22) // Get global variable
	StoreGlobal                = Opcode(23) // Set global variable
	StoreIndexedGlobal         = Opcode(24) // Set global variable using selectors
	LoadLocal                  = Opcode(25) // Get local variable
	StoreLocal                 = Opcode(26) // Set local variable
	DefineLocal                = Opcode(27) // Define local variable
	StoreIndexedLocal          = Opcode(28) // Set local variable using selectors
	LoadFreePtr                = Opcode(29) // Get free variable pointer object
	LoadFree                   = Opcode(30) // Get free variables
	StoreFree                  = Opcode(31) // Set free variables
	LoadLocalPtr               = Opcode(32) // Get local variable as a pointer
	StoreIndexedFree           = Opcode(33) // Set free variables using selectors
	LoadBuiltinFunction        = Opcode(34) // Get builtin function
	MakeClosure                = Opcode(35) // Push closure
	IterInit                   = Opcode(36) // Iterator init
	IterNext                   = Opcode(37) // Iterator next
	IterKey                    = Opcode(38) // Iterator key
	IterValue                  = Opcode(39) // Iterator value
	BinaryOp                   = Opcode(40) // Binary operation
	Suspend                    = Opcode(41) // Suspend VM
	AccessSelector             = Opcode(42) // Select operation
	CallMethod                 = Opcode(43) // Call method on object
	SliceStep                  = Opcode(44) // Slice with step
	FormatStaticSpec           = Opcode(45) // Format value with pre-parsed FormatSpec static
	FormatRuntimeSpec          = Opcode(46) // Format value with runtime-built FormatSpec string popped from the stack
	Defer                      = Opcode(47) // Register deferred call: pop callee + N args, store on current frame
	DeferMethod                = Opcode(48) // Register deferred method call: pop receiver + N args; method name from static strings[methodIdx]
	ImportBuiltinModule        = Opcode(49) // Import builtin module by static ID
	LoadStaticPrimitive        = Opcode(50) // Push static primitive value (bool, nil)
	LoadStaticDecimal          = Opcode(51) // Push static decimal value
	LoadStaticString           = Opcode(52) // Push static string value
	LoadStaticRunes            = Opcode(53) // Push static runes value
	LoadStaticBytes            = Opcode(54) // Push static bytes value
	LoadStaticTime             = Opcode(55) // Push static time value
	LoadStaticFormatSpec       = Opcode(56) // Push static FormatSpec value
	LoadStaticCompiledFunction = Opcode(57) // Push static compiled function value
	// 58...255 are reserved for future use
)

var names = [...]string{
	AbortCheck:                 "ABORTCHK",
	Pop:                        "POP",
	PushTrue:                   "TRUE",
	PushFalse:                  "FALSE",
	UnaryBitNot:                "BITNOT",
	Equal:                      "EQL",
	NotEqual:                   "NEQ",
	UnaryNeg:                   "NEG",
	UnaryNot:                   "NOT",
	JumpFalsy:                  "JMPF",
	AndJump:                    "ANDJMP",
	OrJump:                     "ORJMP",
	Jump:                       "JMP",
	PushUndefined:              "UNDEF",
	LoadGlobal:                 "GETG",
	StoreGlobal:                "SETG",
	StoreIndexedGlobal:         "SETIG",
	MakeArray:                  "ARR",
	MakeRecord:                 "RECORD",
	Immutable:                  "IMMUT",
	AccessIndex:                "INDEX",
	Slice:                      "SLICE",
	CallFunction:               "CALL",
	SliceStep:                  "SLICESTEP",
	Return:                     "RET",
	LoadLocal:                  "GETL",
	StoreLocal:                 "SETL",
	DefineLocal:                "DEFL",
	StoreIndexedLocal:          "SETIL",
	LoadBuiltinFunction:        "BUILTIN",
	MakeClosure:                "CLOSURE",
	LoadFreePtr:                "GETFP",
	LoadFree:                   "GETF",
	StoreFree:                  "SETF",
	LoadLocalPtr:               "GETLP",
	StoreIndexedFree:           "SETIF",
	IterInit:                   "ITER",
	IterNext:                   "ITNXT",
	IterKey:                    "ITKEY",
	IterValue:                  "ITVAL",
	BinaryOp:                   "BINARYOP",
	Suspend:                    "SUSPEND",
	AccessSelector:             "SELECT",
	CallMethod:                 "MCALL",
	Contains:                   "CONTAINS",
	FormatStaticSpec:           "FMTS",
	FormatRuntimeSpec:          "FMTRT",
	Defer:                      "DEFER",
	DeferMethod:                "DEFERM",
	ImportBuiltinModule:        "IMPMOD",
	LoadStaticPrimitive:        "STATICPRIM",
	LoadStaticDecimal:          "STATICDEC",
	LoadStaticString:           "STATICSTR",
	LoadStaticRunes:            "STATICRUNES",
	LoadStaticBytes:            "STATICBYTES",
	LoadStaticTime:             "STATICTIME",
	LoadStaticFormatSpec:       "STATICFMTSPEC",
	LoadStaticCompiledFunction: "STATICCOMPFN",
}

// describes the number and shape of opcode operands
var operands = [...][]int{
	AbortCheck:                 {},
	Pop:                        {},
	PushTrue:                   {},
	PushFalse:                  {},
	UnaryBitNot:                {},
	Equal:                      {},
	NotEqual:                   {},
	UnaryNeg:                   {},
	UnaryNot:                   {},
	JumpFalsy:                  {2}, // new pos
	AndJump:                    {2}, // new pos
	OrJump:                     {2}, // new pos
	Jump:                       {2}, // new pos
	PushUndefined:              {},
	LoadGlobal:                 {2},    // index
	StoreGlobal:                {2},    // index
	StoreIndexedGlobal:         {2, 1}, // index, num selectors
	MakeArray:                  {2},    // num elements (inline init)
	MakeRecord:                 {2},    // num elements (inline init)
	Immutable:                  {},
	AccessIndex:                {},
	Slice:                      {},
	CallFunction:               {1, 1}, // num args, is spread (0 or 1)
	SliceStep:                  {},
	Return:                     {1},    // has result (0 or 1)
	LoadLocal:                  {1},    // index
	StoreLocal:                 {1},    // index
	DefineLocal:                {1},    // index
	StoreIndexedLocal:          {1, 1}, // index, num selectors
	LoadBuiltinFunction:        {1},    // index
	MakeClosure:                {2, 1}, // num args, is spread (0 or 1)
	LoadFreePtr:                {1},    // index
	LoadFree:                   {1},    // index
	StoreFree:                  {1},    // index
	LoadLocalPtr:               {1},    // index
	StoreIndexedFree:           {1, 1}, // index, num selectors
	IterInit:                   {},
	IterNext:                   {},
	IterKey:                    {},
	IterValue:                  {},
	BinaryOp:                   {1}, // token
	Suspend:                    {},
	AccessSelector:             {},
	CallMethod:                 {2, 1, 1}, // method const index, num args, spread
	Contains:                   {},
	FormatStaticSpec:           {2},    // format spec constant index
	FormatRuntimeSpec:          {},     // pops spec string and value from stack, pushes formatted string
	Defer:                      {1},    // numArgs (callee + args popped from stack at runtime)
	DeferMethod:                {2, 1}, // method const index, num args
	ImportBuiltinModule:        {1},    // module static ID
	LoadStaticPrimitive:        {2},    // index
	LoadStaticDecimal:          {2},    // index
	LoadStaticString:           {2},    // index
	LoadStaticRunes:            {2},    // index
	LoadStaticBytes:            {2},    // index
	LoadStaticTime:             {2},    // index
	LoadStaticFormatSpec:       {2},    // index
	LoadStaticCompiledFunction: {2},    // index
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
