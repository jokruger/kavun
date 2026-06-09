package opcode

type Opcode byte

const (
	// 0 is reserved for future use
	BComplement                 = Opcode(1)  // bitwise complement
	Pop                         = Opcode(2)  // Pop
	True                        = Opcode(3)  // Push true
	False                       = Opcode(4)  // Push false
	Equal                       = Opcode(5)  // Equal ==
	NotEqual                    = Opcode(6)  // Not equal !=
	Minus                       = Opcode(7)  // Minus -
	LNot                        = Opcode(8)  // Logical not !
	JumpFalsy                   = Opcode(9)  // Jump if falsy
	AndJump                     = Opcode(10) // Logical AND jump
	OrJump                      = Opcode(11) // Logical OR jump
	Jump                        = Opcode(12) // Jump
	Null                        = Opcode(13) // Push null
	Array                       = Opcode(14) // Array object
	Record                      = Opcode(15) // Record object
	Contains                    = Opcode(16) // Contains operation (x in y)
	Immutable                   = Opcode(17) // Immutable object
	Index                       = Opcode(18) // Index operation
	SliceIndex                  = Opcode(19) // Slice operation
	Call                        = Opcode(20) // Call function
	Return                      = Opcode(21) // Return
	GetGlobal                   = Opcode(22) // Get global variable
	SetGlobal                   = Opcode(23) // Set global variable
	SetSelGlobal                = Opcode(24) // Set global variable using selectors
	GetLocal                    = Opcode(25) // Get local variable
	SetLocal                    = Opcode(26) // Set local variable
	DefineLocal                 = Opcode(27) // Define local variable
	SetSelLocal                 = Opcode(28) // Set local variable using selectors
	GetFreePtr                  = Opcode(29) // Get free variable pointer object
	GetFree                     = Opcode(30) // Get free variables
	SetFree                     = Opcode(31) // Set free variables
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
	StaticFormatSpecValue       = Opcode(54) // Push static FormatSpec value
	StaticCompiledFunctionValue = Opcode(55) // Push static compiled function value
	// 56...255 are reserved for future use
)

var names = [...]string{
	Pop:                         "POP",
	True:                        "TRUE",
	False:                       "FALSE",
	BComplement:                 "NEG",
	Equal:                       "EQL",
	NotEqual:                    "NEQ",
	Minus:                       "NEG",
	LNot:                        "NOT",
	JumpFalsy:                   "JMPF",
	AndJump:                     "ANDJMP",
	OrJump:                      "ORJMP",
	Jump:                        "JMP",
	Null:                        "NULL",
	GetGlobal:                   "GETG",
	SetGlobal:                   "SETG",
	SetSelGlobal:                "SETSG",
	Array:                       "ARR",
	Record:                      "RECORD",
	Immutable:                   "IMMUT",
	Index:                       "INDEX",
	SliceIndex:                  "SLICE",
	Call:                        "CALL",
	SliceIndexStep:              "SLICESTEP",
	Return:                      "RET",
	GetLocal:                    "GETL",
	SetLocal:                    "SETL",
	DefineLocal:                 "DEFL",
	SetSelLocal:                 "SETSL",
	GetBuiltinFunction:          "BUILTIN",
	Closure:                     "CLOSURE",
	GetFreePtr:                  "GETFP",
	GetFree:                     "GETF",
	SetFree:                     "SETF",
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
	StaticFormatSpecValue:       "STATICFMTSPEC",
	StaticCompiledFunctionValue: "STATICCOMPFN",
}

// describes the number and shape of opcode operands
var operands = [...][]int{
	Pop:                         {},
	True:                        {},
	False:                       {},
	BComplement:                 {},
	Equal:                       {},
	NotEqual:                    {},
	Minus:                       {},
	LNot:                        {},
	JumpFalsy:                   {2}, // new pos
	AndJump:                     {2}, // new pos
	OrJump:                      {2}, // new pos
	Jump:                        {2}, // new pos
	Null:                        {},
	GetGlobal:                   {2},    // index
	SetGlobal:                   {2},    // index
	SetSelGlobal:                {2, 1}, // index, num selectors
	Array:                       {2},    // num elements (inline init)
	Record:                      {2},    // num elements (inline init)
	Immutable:                   {},
	Index:                       {},
	SliceIndex:                  {},
	Call:                        {1, 1}, // num args, is spread (0 or 1)
	SliceIndexStep:              {},
	Return:                      {1},    // has result (0 or 1)
	GetLocal:                    {1},    // index
	SetLocal:                    {1},    // index
	DefineLocal:                 {1},    // index
	SetSelLocal:                 {1, 1}, // index, num selectors
	GetBuiltinFunction:          {1},    // index
	Closure:                     {2, 1}, // num args, is spread (0 or 1)
	GetFreePtr:                  {1},    // index
	GetFree:                     {1},    // index
	SetFree:                     {1},    // index
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
