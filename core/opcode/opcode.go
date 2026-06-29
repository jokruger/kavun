package opcode

type Opcode byte

func (op Opcode) Byte() byte {
	return byte(op)
}

func (op Opcode) String() string {
	return Opcodes[op].Name
}

func (op Opcode) Width() int8 {
	return Opcodes[op].Width
}

func (op Opcode) Class() OpClass {
	return Opcodes[op].Class
}

func (op Opcode) Operands() []int8 {
	return Opcodes[op].Operands
}

const (
	AbortCheck                   = Opcode(0)  // Poll VM abort flag; return control to host when set
	Suspend                      = Opcode(1)  // Suspend VM
	Return                       = Opcode(2)  // Return
	Pop                          = Opcode(3)  // Pop
	PushTrue                     = Opcode(4)  // Push true
	PushFalse                    = Opcode(5)  // Push false
	PushUndefined                = Opcode(6)  // Push undefined
	UnaryNeg                     = Opcode(7)  // Unary negation -
	UnaryNot                     = Opcode(8)  // Logical not !
	UnaryBitNot                  = Opcode(9)  // Unary bitwise NOT
	Equal                        = Opcode(10) // Equal ==
	NotEqual                     = Opcode(11) // Not equal !=
	Contains                     = Opcode(12) // Contains operation (x in y)
	Immutable                    = Opcode(13) // Immutable object
	AccessIndex                  = Opcode(14) // Index operation
	AccessSelector               = Opcode(15) // Select operation
	Slice                        = Opcode(16) // Slice operation
	SliceStep                    = Opcode(17) // Slice with step
	IterInit                     = Opcode(18) // Iterator init
	IterNext                     = Opcode(19) // Iterator next
	IterKey                      = Opcode(20) // Iterator key
	IterValue                    = Opcode(21) // Iterator value
	FormatRuntimeSpec            = Opcode(22) // Format value with runtime-built FormatSpec string popped from the stack
	FormatStaticSpec             = Opcode(23) // Format value with pre-parsed FormatSpec static
	BinaryOp                     = Opcode(24) // Binary operation
	ImportBuiltinModule          = Opcode(25) // Import builtin module by static ID
	DefineLocal                  = Opcode(26) // Define local variable
	LoadLocal                    = Opcode(27) // Get local variable
	StoreLocal                   = Opcode(28) // Set local variable
	StoreIndexedLocal            = Opcode(29) // Set local variable using selectors
	LoadFree                     = Opcode(30) // Get free variables
	StoreFree                    = Opcode(31) // Set free variables
	StoreIndexedFree             = Opcode(32) // Set free variables using selectors
	LoadLocalPtr                 = Opcode(33) // Get local variable as a pointer
	LoadFreePtr                  = Opcode(34) // Get free variable pointer object
	LoadBuiltinFunction8         = Opcode(35) // Get builtin function, 8-bit index
	LoadBuiltinFunction16        = Opcode(36) // Get builtin function, 16-bit index
	MakeClosure8                 = Opcode(37) // Push closure, 8-bit static function index, 8-bit free var count
	MakeClosure16                = Opcode(38) // Push closure, 16-bit static function index, 8-bit free var count
	LoadGlobal8                  = Opcode(39) // Get global variable, 8-bit index
	LoadGlobal16                 = Opcode(40) // Get global variable, 16-bit index
	StoreGlobal8                 = Opcode(41) // Set global variable, 8-bit index
	StoreGlobal16                = Opcode(42) // Set global variable, 16-bit index
	StoreIndexedGlobal8          = Opcode(43) // Set global variable using selectors, 8-bit index
	StoreIndexedGlobal16         = Opcode(44) // Set global variable using selectors, 16-bit index
	MakeArray8                   = Opcode(45) // Array object, 8-bit element count
	MakeArray16                  = Opcode(46) // Array object, 16-bit element count
	MakeRecord8                  = Opcode(47) // Record object, 8-bit element count
	MakeRecord16                 = Opcode(48) // Record object, 16-bit element count
	CallFunction                 = Opcode(49) // Call function
	CallMethod8                  = Opcode(50) // Call method on object, 8-bit method name index
	CallMethod16                 = Opcode(51) // Call method on object, 16-bit method name index
	Defer                        = Opcode(52) // Register deferred call: pop callee + N args, store on current frame
	DeferMethod8                 = Opcode(53) // Register deferred method call: pop receiver + N args; 8-bit method name from static
	DeferMethod16                = Opcode(54) // Register deferred method call: pop receiver + N args; 18-bit method name from static
	Jump8                        = Opcode(55) // Jump, 8-bit address
	Jump16                       = Opcode(56) // Jump, 16-bit address
	JumpFalsy                    = Opcode(57) // Jump if falsy, 16-bit address
	AndJump                      = Opcode(58) // Logical AND jump, 16-bit address
	OrJump                       = Opcode(59) // Logical OR jump, 16-bit address
	LoadStaticDecimal8           = Opcode(60) // Push static decimal value, 8-bit index
	LoadStaticDecimal16          = Opcode(61) // Push static decimal value, 16-bit index
	LoadStaticString8            = Opcode(62) // Push static string value, 8-bit index
	LoadStaticString16           = Opcode(63) // Push static string value, 16-bit index
	LoadStaticRunes8             = Opcode(64) // Push static runes value, 8-bit index
	LoadStaticRunes16            = Opcode(65) // Push static runes value, 16-bit index
	LoadStaticBytes8             = Opcode(66) // Push static bytes value, 8-bit index
	LoadStaticBytes16            = Opcode(67) // Push static bytes value, 16-bit index
	LoadStaticTime8              = Opcode(68) // Push static time value, 8-bit index
	LoadStaticTime16             = Opcode(69) // Push static time value, 16-bit index
	LoadStaticFormatSpec8        = Opcode(70) // Push static FormatSpec value, 8-bit index
	LoadStaticFormatSpec16       = Opcode(71) // Push static FormatSpec value, 16-bit index
	LoadStaticCompiledFunction8  = Opcode(72) // Push static compiled function value, 8-bit index
	LoadStaticCompiledFunction16 = Opcode(73) // Push static compiled function value, 16-bit index
	LoadStaticPrimitive8         = Opcode(74) // Push static primitive value, 8-bit index
	LoadStaticPrimitive16        = Opcode(75) // Push static primitive value, 16-bit index
	// ...255 are reserved for future use
)

type OpClass byte

const (
	OpFallThrough   = OpClass(0) // Proceed to next instruction
	OpConditional   = OpClass(1) // Conditional jump
	OpUnconditional = OpClass(2) // Unconditional jump
	OpTerminating   = OpClass(3) // Terminating instruction (return, abort, etc.)
)

type OpDescr struct {
	Name     string
	Operands []int8  // Number of bytes for each operand
	Width    int8    // Total width of the instruction in bytes (including opcode and operands)
	Class    OpClass // Instruction class (fall-through, conditional jump, etc.)
}

var Opcodes = [...]OpDescr{
	AbortCheck:                   {"ABORT_CHECK", []int8{}, 1, OpFallThrough},
	Suspend:                      {"SUSPEND", []int8{}, 1, OpTerminating},
	Return:                       {"RETURN", []int8{1}, 2, OpTerminating}, // op: has result (0 or 1)
	Pop:                          {"POP", []int8{}, 1, OpFallThrough},
	PushTrue:                     {"PUSH_TRUE", []int8{}, 1, OpFallThrough},
	PushFalse:                    {"PUSH_FALSE", []int8{}, 1, OpFallThrough},
	PushUndefined:                {"PUSH_UNDEFINED", []int8{}, 1, OpFallThrough},
	UnaryNeg:                     {"UNARY_NEG", []int8{}, 1, OpFallThrough},
	UnaryNot:                     {"UNARY_NOT", []int8{}, 1, OpFallThrough},
	UnaryBitNot:                  {"UNARY_BITNOT", []int8{}, 1, OpFallThrough},
	Equal:                        {"EQUAL", []int8{}, 1, OpFallThrough},
	NotEqual:                     {"NOT_EQUAL", []int8{}, 1, OpFallThrough},
	Contains:                     {"CONTAINS", []int8{}, 1, OpFallThrough},
	Immutable:                    {"IMMUTABLE", []int8{}, 1, OpFallThrough},
	AccessIndex:                  {"ACCESS_INDEX", []int8{}, 1, OpFallThrough},
	AccessSelector:               {"ACCESS_SELECTOR", []int8{}, 1, OpFallThrough},
	Slice:                        {"SLICE", []int8{}, 1, OpFallThrough},
	SliceStep:                    {"SLICE_STEP", []int8{}, 1, OpFallThrough},
	IterInit:                     {"ITER_INIT", []int8{}, 1, OpFallThrough},
	IterNext:                     {"ITER_NEXT", []int8{}, 1, OpFallThrough},
	IterKey:                      {"ITER_KEY", []int8{}, 1, OpFallThrough},
	IterValue:                    {"ITER_VALUE", []int8{}, 1, OpFallThrough},
	FormatRuntimeSpec:            {"FORMAT_RUNTIME_SPEC", []int8{}, 1, OpFallThrough},
	FormatStaticSpec:             {"FORMAT_STATIC_SPEC", []int8{2}, 3, OpFallThrough},               // format spec constant 16-bit index
	BinaryOp:                     {"BINARY_OP", []int8{1}, 2, OpFallThrough},                        // token
	ImportBuiltinModule:          {"IMPORT_BUILTIN_MODULE", []int8{1}, 2, OpFallThrough},            // module static ID, 8-bit index
	DefineLocal:                  {"DEFINE_LOCAL", []int8{1}, 2, OpFallThrough},                     // local index, 8-bit index
	LoadLocal:                    {"LOAD_LOCAL", []int8{1}, 2, OpFallThrough},                       // local index, 8-bit index
	StoreLocal:                   {"STORE_LOCAL", []int8{1}, 2, OpFallThrough},                      // local index, 8-bit index
	StoreIndexedLocal:            {"STORE_INDEXED_LOCAL", []int8{1, 1}, 3, OpFallThrough},           // local 8-bit index, num selectors (max 255)
	LoadFree:                     {"LOAD_FREE", []int8{1}, 2, OpFallThrough},                        // free index, 8-bit index
	StoreFree:                    {"STORE_FREE", []int8{1}, 2, OpFallThrough},                       // free index, 8-bit index
	StoreIndexedFree:             {"STORE_INDEXED_FREE", []int8{1, 1}, 3, OpFallThrough},            // free 8-bit index, num selectors (max 255)
	LoadLocalPtr:                 {"LOAD_LOCAL_PTR", []int8{1}, 2, OpFallThrough},                   // local index, 8-bit index
	LoadFreePtr:                  {"LOAD_FREE_PTR", []int8{1}, 2, OpFallThrough},                    // free index, 8-bit index
	LoadBuiltinFunction8:         {"LOAD_BUILTIN_FUNCTION_8", []int8{1}, 2, OpFallThrough},          // builtin function 8-bit index
	LoadBuiltinFunction16:        {"LOAD_BUILTIN_FUNCTION_16", []int8{2}, 3, OpFallThrough},         // builtin function 16-bit index
	MakeClosure8:                 {"MAKE_CLOSURE_8", []int8{1, 1}, 3, OpFallThrough},                // static function 8-bit index, num free vars
	MakeClosure16:                {"MAKE_CLOSURE_16", []int8{2, 1}, 4, OpFallThrough},               // static function 16-bit index, num free vars
	LoadGlobal8:                  {"LOAD_GLOBAL_8", []int8{1}, 2, OpFallThrough},                    // global variable 8-bit index
	LoadGlobal16:                 {"LOAD_GLOBAL_16", []int8{2}, 3, OpFallThrough},                   // global variable 16-bit index
	StoreGlobal8:                 {"STORE_GLOBAL_8", []int8{1}, 2, OpFallThrough},                   // global variable 8-bit index
	StoreGlobal16:                {"STORE_GLOBAL_16", []int8{2}, 3, OpFallThrough},                  // global variable 16-bit index
	StoreIndexedGlobal8:          {"STORE_INDEXED_GLOBAL_8", []int8{1, 1}, 3, OpFallThrough},        // global variable 8-bit index, num selectors (max 255)
	StoreIndexedGlobal16:         {"STORE_INDEXED_GLOBAL_16", []int8{2, 1}, 4, OpFallThrough},       // global variable 16-bit index, num selectors (max 255)
	MakeArray8:                   {"MAKE_ARRAY_8", []int8{1}, 2, OpFallThrough},                     // 8-bit num elements
	MakeArray16:                  {"MAKE_ARRAY_16", []int8{2}, 3, OpFallThrough},                    // 16-bit num elements
	MakeRecord8:                  {"MAKE_RECORD_8", []int8{1}, 2, OpFallThrough},                    // 8-bit num elements
	MakeRecord16:                 {"MAKE_RECORD_16", []int8{2}, 3, OpFallThrough},                   // 16-bit num elements
	CallFunction:                 {"CALL_FUNCTION", []int8{1, 1}, 3, OpFallThrough},                 // num args, is spread (0 or 1)
	CallMethod8:                  {"CALL_METHOD_8", []int8{1, 1, 1}, 4, OpFallThrough},              // 8-bit method const index, num args, spread
	CallMethod16:                 {"CALL_METHOD_16", []int8{2, 1, 1}, 5, OpFallThrough},             // 16-bit method const index, num args, spread
	Defer:                        {"DEFER", []int8{1}, 2, OpFallThrough},                            // num args (callee + args popped from stack at runtime)
	DeferMethod8:                 {"DEFER_METHOD_8", []int8{1, 1}, 3, OpFallThrough},                // 8-bit method const index, num args
	DeferMethod16:                {"DEFER_METHOD_16", []int8{2, 1}, 4, OpFallThrough},               // 16-bit method const index, num args
	Jump8:                        {"JUMP_8", []int8{1}, 2, OpUnconditional},                         // 8-bit address
	Jump16:                       {"JUMP_16", []int8{2}, 3, OpUnconditional},                        // 16-bit address
	JumpFalsy:                    {"JUMP_FALSY", []int8{2}, 3, OpConditional},                       // 16-bit address
	AndJump:                      {"AND_JUMP", []int8{2}, 3, OpConditional},                         // 16-bit address
	OrJump:                       {"OR_JUMP", []int8{2}, 3, OpConditional},                          // 16-bit address
	LoadStaticDecimal8:           {"LOAD_STATIC_DECIMAL_8", []int8{1}, 2, OpFallThrough},            // 8-bit index
	LoadStaticDecimal16:          {"LOAD_STATIC_DECIMAL_16", []int8{2}, 3, OpFallThrough},           // 16-bit index
	LoadStaticString8:            {"LOAD_STATIC_STRING_8", []int8{1}, 2, OpFallThrough},             // 8-bit index
	LoadStaticString16:           {"LOAD_STATIC_STRING_16", []int8{2}, 3, OpFallThrough},            // 16-bit index
	LoadStaticRunes8:             {"LOAD_STATIC_RUNES_8", []int8{1}, 2, OpFallThrough},              // 8-bit index
	LoadStaticRunes16:            {"LOAD_STATIC_RUNES_16", []int8{2}, 3, OpFallThrough},             // 16-bit index
	LoadStaticBytes8:             {"LOAD_STATIC_BYTES_8", []int8{1}, 2, OpFallThrough},              // 8-bit index
	LoadStaticBytes16:            {"LOAD_STATIC_BYTES_16", []int8{2}, 3, OpFallThrough},             // 16-bit index
	LoadStaticTime8:              {"LOAD_STATIC_TIME_8", []int8{1}, 2, OpFallThrough},               // 8-bit index
	LoadStaticTime16:             {"LOAD_STATIC_TIME_16", []int8{2}, 3, OpFallThrough},              // 16-bit index
	LoadStaticFormatSpec8:        {"LOAD_STATIC_FORMAT_SPEC_8", []int8{1}, 2, OpFallThrough},        // 8-bit index
	LoadStaticFormatSpec16:       {"LOAD_STATIC_FORMAT_SPEC_16", []int8{2}, 3, OpFallThrough},       // 16-bit index
	LoadStaticCompiledFunction8:  {"LOAD_STATIC_COMPILED_FUNCTION_8", []int8{1}, 2, OpFallThrough},  // 8-bit index
	LoadStaticCompiledFunction16: {"LOAD_STATIC_COMPILED_FUNCTION_16", []int8{2}, 3, OpFallThrough}, // 16-bit index
	LoadStaticPrimitive8:         {"LOAD_STATIC_PRIMITIVE_8", []int8{1}, 2, OpFallThrough},          // 8-bit index
	LoadStaticPrimitive16:        {"LOAD_STATIC_PRIMITIVE_16", []int8{2}, 3, OpFallThrough},         // 16-bit index
}
