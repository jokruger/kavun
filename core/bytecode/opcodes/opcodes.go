// Package opcodes is the enum of VM instruction opcodes: one untyped constant per instruction. It holds constants
// only — the kind (bytecode.Opcode), the descriptor table (names, classes) and every function live in core/bytecode.
//
// Adding, removing or renumbering an opcode breaks bytecode compatibility: change vm.BytecodeMagic with it.
package opcodes

const (
	AbortCheck                 = 0  // Poll the VM abort flag; return control to the host when set; no operands
	Suspend                    = 1  // Suspend the VM, returning control to the host; no operands
	Return                     = 2  // Return from the current function; Op1 = has result (0 or 1)
	Jump                       = 3  // Jump unconditionally; Op3 = target ip
	JumpFalsy                  = 4  // Pop the condition, jump if it is falsy; Op3 = target ip
	AndJump                    = 5  // Logical AND: if the top is falsy keep it and jump, else pop it; Op3 = target ip
	OrJump                     = 6  // Logical OR: if the top is truthy keep it and jump, else pop it; Op3 = target ip
	Pop                        = 7  // Pop and discard the top of the stack; no operands
	Unpack                     = 8  // Destructure RHS into N values; Op1 = N (count of LHS positions), Op3 = static name-list index
	Immutable                  = 9  // Make the top of the stack immutable; Op1 = 1 deep (export), 0 shallow
	UnaryNeg                   = 10 // Unary negation (-x); no operands
	UnaryNot                   = 11 // Logical not (!x); no operands
	UnaryBitNot                = 12 // Unary bitwise not (^x); no operands
	Equal                      = 13 // Equality (x == y); no operands
	NotEqual                   = 14 // Inequality (x != y); no operands
	Contains                   = 15 // Membership (x in y); no operands
	Slice                      = 16 // Slice read x[lo:hi]; no operands
	SliceStep                  = 17 // Slice read with step x[lo:hi:step]; no operands
	IterInit                   = 18 // Replace the top of the stack with an iterator over it; no operands
	IterNext                   = 19 // Advance the iterator on top, replace it with whether there is an element; no operands
	IterKey                    = 20 // Replace the iterator on top with its current key; no operands
	IterValue                  = 21 // Replace the iterator on top with its current value; no operands
	IterElem                   = 22 // Replace the iterator on top with its current element — the single-variable for-in binding; no operands
	AccessIndex                = 23 // Index read x[k] (stack: recv, key); no operands
	AccessProperty             = 24 // Property read x.name; Op2 = member ID, Op3 = static name index
	AssignIndex                = 25 // Index write x[k] = value (stack: value, recv, key); no operands
	AssignProperty             = 26 // Property write x.name = value (stack: value, recv); Op2 = member ID, Op3 = static name index
	BinaryOp                   = 27 // Binary operator; Op1 = token
	CallFunction               = 28 // Call a function f(args); Op2 = num args, Op1 = is spread (0 or 1)
	CallMember                 = 29 // Call member function x.name(args); Op1 = num args (≤ 255), Op2 = member ID, Op3 = static name index
	CallMemberSpread           = 30 // CallMember whose last argument is an array to spread; operands as CallMember
	Defer                      = 31 // Register a deferred function call; Op2 = num args (callee is an implicit extra stack item)
	DeferMember                = 32 // Register a deferred member call; Op1 = num args (≤ 255), Op2 = member ID, Op3 = static name index
	FormatRuntimeSpec          = 33 // Format a value with a spec string computed at run time (stack: value, spec); no operands
	FormatStaticSpec           = 34 // Format the value on top with a pre-parsed spec; Op3 = static FormatSpec index
	ImportBuiltinModule        = 35 // Push a builtin module; Op3 = builtin module ID
	DefineLocal                = 36 // Pop into a newly defined local variable; Op3 = local index
	StoreLocal                 = 37 // Pop into a local variable; Op3 = local index
	StoreFree                  = 38 // Pop into a free variable; Op3 = free index
	StoreGlobal                = 39 // Pop into a global variable; Op3 = global variable index
	MakeClosure                = 40 // Make a closure from a static function and the free-variable pointers on the stack; Op3 = static function index, Op2 = num free vars
	MakeArray                  = 41 // Make an array from the elements on the stack; Op3 = num elements
	MakeRecord                 = 42 // Make a record from the key/value pairs on the stack; Op3 = 2 * num pairs
	PushUndefined              = 43 // Push undefined; no operands
	PushBool                   = 44 // Push a bool; Op1 = 0 (false) or 1 (true)
	PushByte                   = 45 // Push a byte; Op1 = byte value
	PushRune                   = 46 // Push a rune; Op3 = rune value
	PushInt                    = 47 // Push an int; Op3 = integer value (signed 32-bit)
	LoadLocal                  = 48 // Push a local variable; Op3 = local index
	LoadLocalPtr               = 49 // Push a pointer to a local variable; Op3 = local index
	LoadFree                   = 50 // Push a free variable; Op3 = free index
	LoadFreePtr                = 51 // Push a pointer to a free variable; Op3 = free index
	LoadBuiltinFunction        = 52 // Push a builtin function; Op3 = builtin function ID
	LoadGlobal                 = 53 // Push a global variable; Op3 = global variable index
	LoadStaticDecimal          = 54 // Push a static decimal; Op3 = static decimal index
	LoadStaticString           = 55 // Push a static string; Op3 = static string index
	LoadStaticRunes            = 56 // Push a static runes value; Op3 = static runes index
	LoadStaticBytes            = 57 // Push a static bytes value; Op3 = static bytes index
	LoadStaticTime             = 58 // Push a static time; Op3 = static time index
	LoadStaticFormatSpec       = 59 // Push a static FormatSpec; Op3 = static FormatSpec index
	LoadStaticRange            = 60 // Push a static int range; Op3 = static range index
	LoadStaticCompiledFunction = 61 // Push a static compiled function; Op3 = static compiled function index
	LoadStaticPrimitive        = 62 // Push a static primitive; Op3 = static primitive index
	// 63...255 are reserved for future use; adding, removing or renumbering an opcode requires a new vm.BytecodeMagic
)
