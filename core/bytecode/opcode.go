package bytecode

import "fmt"

type Opcode byte

func (op Opcode) String() string {
	if int(op) >= len(Opcodes) {
		return fmt.Sprintf("UNKNOWN_OPCODE %d", op)
	}
	return Opcodes[op].Name
}

func (op Opcode) Class() OpClass {
	if int(op) >= len(Opcodes) {
		return OpUnknown
	}
	return Opcodes[op].Class
}

const (
	AbortCheck                 = Opcode(0)  // Poll VM abort flag; return control to host when set; no operands
	Suspend                    = Opcode(1)  // Suspend VM; no operands
	Return                     = Opcode(2)  // Return; Op1 = has result (0 or 1)
	Pop                        = Opcode(3)  // Pop; no operands
	UnaryNeg                   = Opcode(4)  // Unary negation; no operands
	UnaryNot                   = Opcode(5)  // Logical not; no operands
	UnaryBitNot                = Opcode(6)  // Unary bitwise not; no operands
	Equal                      = Opcode(7)  // Equal; no operands
	NotEqual                   = Opcode(8)  // Not equal; no operands
	Contains                   = Opcode(9)  // Contains operation (x in y); no operands
	Immutable                  = Opcode(10) // Immutable object; no operands
	AccessIndex                = Opcode(11) // Index operation; no operands
	AccessSelector             = Opcode(12) // Select operation; no operands
	Slice                      = Opcode(13) // Slice operation; no operands
	SliceStep                  = Opcode(14) // Slice with step; no operands
	IterInit                   = Opcode(15) // Iterator init; no operands
	IterNext                   = Opcode(16) // Iterator next; no operands
	IterKey                    = Opcode(17) // Iterator key; no operands
	IterValue                  = Opcode(18) // Iterator value; no operands
	FormatRuntimeSpec          = Opcode(19) // Format value with runtime-built FormatSpec string popped from the stack; no operands
	FormatStaticSpec           = Opcode(20) // Format value with pre-parsed FormatSpec static; Op3 = format spec constant index
	BinaryOp                   = Opcode(21) // Binary operation; Op1 = token
	ImportBuiltinModule        = Opcode(22) // Import builtin module by static ID; Op3 = module static ID
	DefineLocal                = Opcode(23) // Define local variable; Op3 = local index
	LoadLocal                  = Opcode(24) // Get local variable; Op3 = local index
	StoreLocal                 = Opcode(25) // Set local variable; Op3 = local index
	StoreIndexedLocal          = Opcode(26) // Set local variable using selectors; Op3 = local index, Op2 = num selectors
	LoadFree                   = Opcode(27) // Get free variables; Op3 = free index
	StoreFree                  = Opcode(28) // Set free variables; Op3 = free index
	StoreIndexedFree           = Opcode(29) // Set free variables using selectors; Op3 = free index, Op2 = num selectors
	LoadLocalPtr               = Opcode(30) // Get local variable as a pointer; Op3 = local index
	LoadFreePtr                = Opcode(31) // Get free variable pointer object; Op3 = free index
	LoadBuiltinFunction        = Opcode(32) // Get builtin function; Op3 = builtin function ID
	MakeClosure                = Opcode(33) // Push closure; Op3 = static function index, Op2 = num free vars
	LoadGlobal                 = Opcode(34) // Get global variable; Op3 = global variable index
	StoreGlobal                = Opcode(35) // Set global variable; Op3 = global variable index
	StoreIndexedGlobal         = Opcode(36) // Set global variable using selectors; Op3 = global variable index, Op2 = num selectors
	MakeArray                  = Opcode(37) // Array object; Op3 = num elements (on stack)
	MakeRecord                 = Opcode(38) // Record object; Op3 = num fields (on stack) = 2 * num K/V pairs
	CallFunction               = Opcode(39) // Call function; Op2 = num args, Op1 = is spread (0 or 1)
	CallMethod                 = Opcode(40) // Call method on object; Op3 = method const index, Op2 = num args, Op1 = is spread
	Defer                      = Opcode(41) // Register deferred call; Op2 = num args (callee + args popped from stack at runtime)
	DeferMethod                = Opcode(42) // Register deferred method call; Op3 = method const index, Op2 = num args
	Jump                       = Opcode(43) // Jump; Op3 = target ip
	JumpFalsy                  = Opcode(44) // Jump if falsy; Op3 = target ip
	AndJump                    = Opcode(45) // Logical AND jump; Op3 = target ip
	OrJump                     = Opcode(46) // Logical OR jump; Op3 = target ip
	PushUndefined              = Opcode(47) // Push undefined; no operands
	PushBool                   = Opcode(48) // Push boolean; Op1 = 0 (false) or 1 (true)
	PushByte                   = Opcode(49) // Push byte; Op1 = byte value
	PushRune                   = Opcode(50) // Push rune; Op3 = rune value
	PushInt                    = Opcode(51) // Push integer; Op3 = integer value (signed 32-bit)
	LoadStaticDecimal          = Opcode(52) // Push static decimal value; Op3 = static decimal index
	LoadStaticString           = Opcode(53) // Push static string value; Op3 = static string index
	LoadStaticRunes            = Opcode(54) // Push static runes value; Op3 = static runes index
	LoadStaticBytes            = Opcode(55) // Push static bytes value; Op3 = static bytes index
	LoadStaticTime             = Opcode(56) // Push static time value; Op3 = static time index
	LoadStaticFormatSpec       = Opcode(57) // Push static FormatSpec value; Op3 = static FormatSpec index
	LoadStaticCompiledFunction = Opcode(58) // Push static compiled function value; Op3 = static compiled function index
	LoadStaticPrimitive        = Opcode(59) // Push static primitive value; Op3 = static primitive index
	// ...255 are reserved for future use
)

type OpClass byte

const (
	OpUnknown       = OpClass(0)
	OpFallThrough   = OpClass(1) // Proceed to next instruction
	OpConditional   = OpClass(2) // Conditional jump
	OpUnconditional = OpClass(3) // Unconditional jump
	OpTerminating   = OpClass(4) // Terminating instruction (return, abort, etc.)
)

type OpDescr struct {
	Name  string
	Class OpClass // Instruction class (fall-through, conditional jump, etc.)
}

var Opcodes = [...]OpDescr{
	AbortCheck:                 {"ABORT_CHECK", OpFallThrough},
	Suspend:                    {"SUSPEND", OpTerminating},
	Return:                     {"RETURN", OpTerminating},
	Pop:                        {"POP", OpFallThrough},
	UnaryNeg:                   {"UNARY_NEG", OpFallThrough},
	UnaryNot:                   {"UNARY_NOT", OpFallThrough},
	UnaryBitNot:                {"UNARY_BITNOT", OpFallThrough},
	Equal:                      {"EQUAL", OpFallThrough},
	NotEqual:                   {"NOT_EQUAL", OpFallThrough},
	Contains:                   {"CONTAINS", OpFallThrough},
	Immutable:                  {"IMMUTABLE", OpFallThrough},
	AccessIndex:                {"ACCESS_INDEX", OpFallThrough},
	AccessSelector:             {"ACCESS_SELECTOR", OpFallThrough},
	Slice:                      {"SLICE", OpFallThrough},
	SliceStep:                  {"SLICE_STEP", OpFallThrough},
	IterInit:                   {"ITER_INIT", OpFallThrough},
	IterNext:                   {"ITER_NEXT", OpFallThrough},
	IterKey:                    {"ITER_KEY", OpFallThrough},
	IterValue:                  {"ITER_VALUE", OpFallThrough},
	FormatRuntimeSpec:          {"FORMAT_RUNTIME_SPEC", OpFallThrough},
	FormatStaticSpec:           {"FORMAT_STATIC_SPEC", OpFallThrough},
	BinaryOp:                   {"BINARY_OP", OpFallThrough},
	ImportBuiltinModule:        {"IMPORT_BUILTIN_MODULE", OpFallThrough},
	DefineLocal:                {"DEFINE_LOCAL", OpFallThrough},
	LoadLocal:                  {"LOAD_LOCAL", OpFallThrough},
	StoreLocal:                 {"STORE_LOCAL", OpFallThrough},
	StoreIndexedLocal:          {"STORE_INDEXED_LOCAL", OpFallThrough},
	LoadFree:                   {"LOAD_FREE", OpFallThrough},
	StoreFree:                  {"STORE_FREE", OpFallThrough},
	StoreIndexedFree:           {"STORE_INDEXED_FREE", OpFallThrough},
	LoadLocalPtr:               {"LOAD_LOCAL_PTR", OpFallThrough},
	LoadFreePtr:                {"LOAD_FREE_PTR", OpFallThrough},
	LoadBuiltinFunction:        {"LOAD_BUILTIN_FUNCTION", OpFallThrough},
	MakeClosure:                {"MAKE_CLOSURE", OpFallThrough},
	LoadGlobal:                 {"LOAD_GLOBAL", OpFallThrough},
	StoreGlobal:                {"STORE_GLOBAL", OpFallThrough},
	StoreIndexedGlobal:         {"STORE_INDEXED_GLOBAL", OpFallThrough},
	MakeArray:                  {"MAKE_ARRAY", OpFallThrough},
	MakeRecord:                 {"MAKE_RECORD", OpFallThrough},
	CallFunction:               {"CALL_FUNCTION", OpFallThrough},
	CallMethod:                 {"CALL_METHOD", OpFallThrough},
	Defer:                      {"DEFER", OpFallThrough},
	DeferMethod:                {"DEFER_METHOD", OpFallThrough},
	Jump:                       {"JUMP", OpUnconditional},
	JumpFalsy:                  {"JUMP_FALSY", OpConditional},
	AndJump:                    {"AND_JUMP", OpConditional},
	OrJump:                     {"OR_JUMP", OpConditional},
	PushUndefined:              {"PUSH_UNDEFINED", OpFallThrough},
	PushBool:                   {"PUSH_BOOL", OpFallThrough},
	PushByte:                   {"PUSH_BYTE", OpFallThrough},
	PushRune:                   {"PUSH_RUNE", OpFallThrough},
	PushInt:                    {"PUSH_INT", OpFallThrough},
	LoadStaticDecimal:          {"LOAD_STATIC_DECIMAL", OpFallThrough},
	LoadStaticString:           {"LOAD_STATIC_STRING", OpFallThrough},
	LoadStaticRunes:            {"LOAD_STATIC_RUNES", OpFallThrough},
	LoadStaticBytes:            {"LOAD_STATIC_BYTES", OpFallThrough},
	LoadStaticTime:             {"LOAD_STATIC_TIME", OpFallThrough},
	LoadStaticFormatSpec:       {"LOAD_STATIC_FORMAT_SPEC", OpFallThrough},
	LoadStaticCompiledFunction: {"LOAD_STATIC_COMPILED_FUNCTION", OpFallThrough},
	LoadStaticPrimitive:        {"LOAD_STATIC_PRIMITIVE", OpFallThrough},
}
