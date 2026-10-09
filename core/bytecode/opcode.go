package bytecode

import "fmt"

type Opcode byte

func (op Opcode) String() string {
	if int(op) >= len(Opcodes) || Opcodes[op].Name == "" {
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
	AbortCheck                 = Opcode(0)  // Poll the VM abort flag; return control to the host when set; no operands
	Suspend                    = Opcode(1)  // Suspend the VM, returning control to the host; no operands
	Return                     = Opcode(2)  // Return from the current function; Op1 = has result (0 or 1)
	Jump                       = Opcode(3)  // Jump unconditionally; Op3 = target ip
	JumpFalsy                  = Opcode(4)  // Pop the condition, jump if it is falsy; Op3 = target ip
	AndJump                    = Opcode(5)  // Logical AND: if the top is falsy keep it and jump, else pop it; Op3 = target ip
	OrJump                     = Opcode(6)  // Logical OR: if the top is truthy keep it and jump, else pop it; Op3 = target ip
	Pop                        = Opcode(7)  // Pop and discard the top of the stack; no operands
	Unpack                     = Opcode(8)  // Destructure RHS into N values; Op1 = N (count of LHS positions), Op3 = static name-list index
	Immutable                  = Opcode(9)  // Make the top of the stack immutable; Op1 = 1 deep (export), 0 shallow
	UnaryNeg                   = Opcode(10) // Unary negation (-x); no operands
	UnaryNot                   = Opcode(11) // Logical not (!x); no operands
	UnaryBitNot                = Opcode(12) // Unary bitwise not (^x); no operands
	Equal                      = Opcode(13) // Equality (x == y); no operands
	NotEqual                   = Opcode(14) // Inequality (x != y); no operands
	Contains                   = Opcode(15) // Membership (x in y); no operands
	Slice                      = Opcode(16) // Slice read x[lo:hi]; no operands
	SliceStep                  = Opcode(17) // Slice read with step x[lo:hi:step]; no operands
	IterInit                   = Opcode(18) // Replace the top of the stack with an iterator over it; no operands
	IterNext                   = Opcode(19) // Advance the iterator on top, replace it with whether there is an element; no operands
	IterKey                    = Opcode(20) // Replace the iterator on top with its current key; no operands
	IterValue                  = Opcode(21) // Replace the iterator on top with its current value; no operands
	IterElem                   = Opcode(22) // Replace the iterator on top with its current element — the single-variable for-in binding; no operands
	AccessIndex                = Opcode(23) // Index read x[k] (stack: recv, key); no operands
	AccessProperty             = Opcode(24) // Property read x.name; Op2 = member ID, Op3 = static name index
	AssignIndex                = Opcode(25) // Index write x[k] = value (stack: value, recv, key); no operands
	AssignProperty             = Opcode(26) // Property write x.name = value (stack: value, recv); Op2 = member ID, Op3 = static name index
	BinaryOp                   = Opcode(27) // Binary operator; Op1 = token
	CallFunction               = Opcode(28) // Call a function f(args); Op2 = num args, Op1 = is spread (0 or 1)
	CallMember                 = Opcode(29) // Call member function x.name(args); Op1 = num args (≤ 255), Op2 = member ID, Op3 = static name index
	CallMemberSpread           = Opcode(30) // CallMember whose last argument is an array to spread; operands as CallMember
	Defer                      = Opcode(31) // Register a deferred function call; Op2 = num args (callee is an implicit extra stack item)
	DeferMember                = Opcode(32) // Register a deferred member call; Op1 = num args (≤ 255), Op2 = member ID, Op3 = static name index
	FormatRuntimeSpec          = Opcode(33) // Format a value with a spec string computed at run time (stack: value, spec); no operands
	FormatStaticSpec           = Opcode(34) // Format the value on top with a pre-parsed spec; Op3 = static FormatSpec index
	ImportBuiltinModule        = Opcode(35) // Push a builtin module; Op3 = builtin module ID
	DefineLocal                = Opcode(36) // Pop into a newly defined local variable; Op3 = local index
	StoreLocal                 = Opcode(37) // Pop into a local variable; Op3 = local index
	StoreFree                  = Opcode(38) // Pop into a free variable; Op3 = free index
	StoreGlobal                = Opcode(39) // Pop into a global variable; Op3 = global variable index
	MakeClosure                = Opcode(40) // Make a closure from a static function and the free-variable pointers on the stack; Op3 = static function index, Op2 = num free vars
	MakeArray                  = Opcode(41) // Make an array from the elements on the stack; Op3 = num elements
	MakeRecord                 = Opcode(42) // Make a record from the key/value pairs on the stack; Op3 = 2 * num pairs
	PushUndefined              = Opcode(43) // Push undefined; no operands
	PushBool                   = Opcode(44) // Push a bool; Op1 = 0 (false) or 1 (true)
	PushByte                   = Opcode(45) // Push a byte; Op1 = byte value
	PushRune                   = Opcode(46) // Push a rune; Op3 = rune value
	PushInt                    = Opcode(47) // Push an int; Op3 = integer value (signed 32-bit)
	LoadLocal                  = Opcode(48) // Push a local variable; Op3 = local index
	LoadLocalPtr               = Opcode(49) // Push a pointer to a local variable; Op3 = local index
	LoadFree                   = Opcode(50) // Push a free variable; Op3 = free index
	LoadFreePtr                = Opcode(51) // Push a pointer to a free variable; Op3 = free index
	LoadBuiltinFunction        = Opcode(52) // Push a builtin function; Op3 = builtin function ID
	LoadGlobal                 = Opcode(53) // Push a global variable; Op3 = global variable index
	LoadStaticDecimal          = Opcode(54) // Push a static decimal; Op3 = static decimal index
	LoadStaticString           = Opcode(55) // Push a static string; Op3 = static string index
	LoadStaticRunes            = Opcode(56) // Push a static runes value; Op3 = static runes index
	LoadStaticBytes            = Opcode(57) // Push a static bytes value; Op3 = static bytes index
	LoadStaticTime             = Opcode(58) // Push a static time; Op3 = static time index
	LoadStaticFormatSpec       = Opcode(59) // Push a static FormatSpec; Op3 = static FormatSpec index
	LoadStaticRange            = Opcode(60) // Push a static int range; Op3 = static range index
	LoadStaticCompiledFunction = Opcode(61) // Push a static compiled function; Op3 = static compiled function index
	LoadStaticPrimitive        = Opcode(62) // Push a static primitive; Op3 = static primitive index
	// 63...255 are reserved for future use; adding, removing or renumbering an opcode requires a new vm.BytecodeMagic
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
	Jump:                       {"JUMP", OpUnconditional},
	JumpFalsy:                  {"JUMP_FALSY", OpConditional},
	AndJump:                    {"AND_JUMP", OpConditional},
	OrJump:                     {"OR_JUMP", OpConditional},
	Pop:                        {"POP", OpFallThrough},
	Unpack:                     {"UNPACK", OpFallThrough},
	Immutable:                  {"IMMUTABLE", OpFallThrough},
	UnaryNeg:                   {"UNARY_NEG", OpFallThrough},
	UnaryNot:                   {"UNARY_NOT", OpFallThrough},
	UnaryBitNot:                {"UNARY_BITNOT", OpFallThrough},
	Equal:                      {"EQUAL", OpFallThrough},
	NotEqual:                   {"NOT_EQUAL", OpFallThrough},
	Contains:                   {"CONTAINS", OpFallThrough},
	Slice:                      {"SLICE", OpFallThrough},
	SliceStep:                  {"SLICE_STEP", OpFallThrough},
	IterInit:                   {"ITER_INIT", OpFallThrough},
	IterNext:                   {"ITER_NEXT", OpFallThrough},
	IterKey:                    {"ITER_KEY", OpFallThrough},
	IterValue:                  {"ITER_VALUE", OpFallThrough},
	IterElem:                   {"ITER_ELEM", OpFallThrough},
	AccessIndex:                {"ACCESS_INDEX", OpFallThrough},
	AccessProperty:             {"ACCESS_PROPERTY", OpFallThrough},
	AssignIndex:                {"ASSIGN_INDEX", OpFallThrough},
	AssignProperty:             {"ASSIGN_PROPERTY", OpFallThrough},
	BinaryOp:                   {"BINARY_OP", OpFallThrough},
	CallFunction:               {"CALL_FUNCTION", OpFallThrough},
	CallMember:                 {"CALL_MEMBER", OpFallThrough},
	CallMemberSpread:           {"CALL_MEMBER_SPREAD", OpFallThrough},
	Defer:                      {"DEFER", OpFallThrough},
	DeferMember:                {"DEFER_MEMBER", OpFallThrough},
	FormatRuntimeSpec:          {"FORMAT_RUNTIME_SPEC", OpFallThrough},
	FormatStaticSpec:           {"FORMAT_STATIC_SPEC", OpFallThrough},
	ImportBuiltinModule:        {"IMPORT_BUILTIN_MODULE", OpFallThrough},
	DefineLocal:                {"DEFINE_LOCAL", OpFallThrough},
	StoreLocal:                 {"STORE_LOCAL", OpFallThrough},
	StoreFree:                  {"STORE_FREE", OpFallThrough},
	StoreGlobal:                {"STORE_GLOBAL", OpFallThrough},
	MakeClosure:                {"MAKE_CLOSURE", OpFallThrough},
	MakeArray:                  {"MAKE_ARRAY", OpFallThrough},
	MakeRecord:                 {"MAKE_RECORD", OpFallThrough},
	PushUndefined:              {"PUSH_UNDEFINED", OpFallThrough},
	PushBool:                   {"PUSH_BOOL", OpFallThrough},
	PushByte:                   {"PUSH_BYTE", OpFallThrough},
	PushRune:                   {"PUSH_RUNE", OpFallThrough},
	PushInt:                    {"PUSH_INT", OpFallThrough},
	LoadLocal:                  {"LOAD_LOCAL", OpFallThrough},
	LoadLocalPtr:               {"LOAD_LOCAL_PTR", OpFallThrough},
	LoadFree:                   {"LOAD_FREE", OpFallThrough},
	LoadFreePtr:                {"LOAD_FREE_PTR", OpFallThrough},
	LoadBuiltinFunction:        {"LOAD_BUILTIN_FUNCTION", OpFallThrough},
	LoadGlobal:                 {"LOAD_GLOBAL", OpFallThrough},
	LoadStaticDecimal:          {"LOAD_STATIC_DECIMAL", OpFallThrough},
	LoadStaticString:           {"LOAD_STATIC_STRING", OpFallThrough},
	LoadStaticRunes:            {"LOAD_STATIC_RUNES", OpFallThrough},
	LoadStaticBytes:            {"LOAD_STATIC_BYTES", OpFallThrough},
	LoadStaticTime:             {"LOAD_STATIC_TIME", OpFallThrough},
	LoadStaticFormatSpec:       {"LOAD_STATIC_FORMAT_SPEC", OpFallThrough},
	LoadStaticRange:            {"LOAD_STATIC_RANGE", OpFallThrough},
	LoadStaticCompiledFunction: {"LOAD_STATIC_COMPILED_FUNCTION", OpFallThrough},
	LoadStaticPrimitive:        {"LOAD_STATIC_PRIMITIVE", OpFallThrough},
}
