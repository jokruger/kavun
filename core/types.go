package core

import (
	"time"

	"github.com/jokruger/gs/token"
)

type Opcode = byte
type NativeFunc = func(VM, []Value) (Value, error)

type Allocator interface {
	NewBuiltinFunctionValue(name string, val NativeFunc, arity int, variadic bool) Value
	NewErrorValue(e Value) Value
	NewTimeValue(t time.Time) Value
	NewStringValue(s string) Value
	NewStringIteratorValue(s []rune) Value
	NewBytesValue(b []byte) Value
	NewBytesIteratorValue(b []byte) Value
	NewArrayValue(arr []Value, immutable bool) Value
	NewArrayIteratorValue(arr []Value) Value
	NewMapValue(m map[string]Value, immutable bool) Value
	NewMapIteratorValue(m map[string]Value) Value
	NewRecordValue(m map[string]Value, immutable bool) Value
	NewIntRangeValue(start, stop, step int64) Value
	NewIntRangeIteratorValue(start, stop, step int64) Value
}

type VM interface {
	Allocator() Allocator
	Abort()
	IsStackEmpty() bool
	Call(*CompiledFunction, []Value) (Value, error)
	Run() error
}

type Pos int

func (p Pos) IsValid() bool {
	return p != NoPos
}

const (
	// Pos constants
	NoPos Pos = 0

	// Value type constants
	VT_UNDEFINED          = uint8(0) // must be first (zero)
	VT_VALUE_PTR          = uint8(1)
	VT_BUILTIN_FUNCTION   = uint8(2)
	VT_COMPILED_FUNCTION  = uint8(3)
	VT_ERROR              = uint8(4)
	VT_BOOL               = uint8(5)
	VT_CHAR               = uint8(6)
	VT_INT                = uint8(7)
	VT_FLOAT              = uint8(8)
	VT_TIME               = uint8(9)
	VT_STRING             = uint8(10)
	VT_BYTES              = uint8(11)
	VT_ARRAY              = uint8(12)
	VT_RECORD             = uint8(13)
	VT_MAP                = uint8(14)
	VT_STRING_ITERATOR    = uint8(15)
	VT_BYTES_ITERATOR     = uint8(16)
	VT_ARRAY_ITERATOR     = uint8(17)
	VT_MAP_ITERATOR       = uint8(18)
	VT_INT_RANGE          = uint8(19)
	VT_INT_RANGE_ITERATOR = uint8(20)
	VT_USER_DEFINED       = uint8(21) // must be last
)

var (
	// MaxStringLen is the maximum byte-length for string value. Note this limit applies to all compiler/VM instances in the process.
	MaxStringLen = 2147483647

	// MaxBytesLen is the maximum length for bytes value. Note this limit applies to all compiler/VM instances in the process.
	MaxBytesLen = 2147483647

	// Type function tables
	TypeName         [256]func(v Value) string
	TypeEncodeJSON   [256]func(v Value) ([]byte, error)
	TypeEncodeBinary [256]func(v Value) ([]byte, error)
	TypeDecodeBinary [256]func(v *Value, data []byte) error
	TypeString       [256]func(v Value) string
	TypeInterface    [256]func(v Value) any

	TypeIsTrue      [256]func(v Value) bool
	TypeIsImmutable [256]func(v Value) bool
	TypeIsIterable  [256]func(v Value) bool
	TypeIsCallable  [256]func(v Value) bool
	TypeContains    [256]func(v Value, e Value) bool

	TypeAsBool   [256]func(v Value) (bool, bool)
	TypeAsChar   [256]func(v Value) (rune, bool)
	TypeAsInt    [256]func(v Value) (int64, bool)
	TypeAsFloat  [256]func(v Value) (float64, bool)
	TypeAsTime   [256]func(v Value) (time.Time, bool)
	TypeAsString [256]func(v Value) (string, bool)
	TypeAsBytes  [256]func(v Value) ([]byte, bool)

	TypeCopy       [256]func(v Value, a Allocator) Value
	TypeEqual      [256]func(v Value, r Value) bool
	TypeBinaryOp   [256]func(v Value, a Allocator, op token.Token, r Value) (Value, error)
	TypeMethodCall [256]func(v Value, vm VM, name string, args []Value) (Value, error)

	TypeAccess   [256]func(v Value, a Allocator, index Value, mode Opcode) (Value, error)
	TypeAssign   [256]func(v Value, index Value, r Value) error
	TypeIterator [256]func(v Value, a Allocator) Value

	TypeNext  [256]func(v *Value) bool
	TypeKey   [256]func(v Value, a Allocator) Value
	TypeValue [256]func(v Value, a Allocator) Value

	TypeArity      [256]func(v Value) int
	TypeIsVariadic [256]func(v Value) bool
	TypeCall       [256]func(v Value, vm VM, args []Value) (Value, error)
)

// List of opcodes
const (
	OpConstant      = Opcode(0)  // Load constant
	OpBComplement   = Opcode(1)  // bitwise complement
	OpPop           = Opcode(2)  // Pop
	OpTrue          = Opcode(3)  // Push true
	OpFalse         = Opcode(4)  // Push false
	OpEqual         = Opcode(5)  // Equal ==
	OpNotEqual      = Opcode(6)  // Not equal !=
	OpMinus         = Opcode(7)  // Minus -
	OpLNot          = Opcode(8)  // Logical not !
	OpJumpFalsy     = Opcode(9)  // Jump if falsy
	OpAndJump       = Opcode(10) // Logical AND jump
	OpOrJump        = Opcode(11) // Logical OR jump
	OpJump          = Opcode(12) // Jump
	OpNull          = Opcode(13) // Push null
	OpArray         = Opcode(14) // Array object
	OpRecord        = Opcode(15) // Record object
	OpError         = Opcode(16) // Error object
	OpImmutable     = Opcode(17) // Immutable object
	OpIndex         = Opcode(18) // Index operation
	OpSliceIndex    = Opcode(19) // Slice operation
	OpCall          = Opcode(20) // Call function
	OpReturn        = Opcode(21) // Return
	OpGetGlobal     = Opcode(22) // Get global variable
	OpSetGlobal     = Opcode(23) // Set global variable
	OpSetSelGlobal  = Opcode(24) // Set global variable using selectors
	OpGetLocal      = Opcode(25) // Get local variable
	OpSetLocal      = Opcode(26) // Set local variable
	OpDefineLocal   = Opcode(27) // Define local variable
	OpSetSelLocal   = Opcode(28) // Set local variable using selectors
	OpGetFreePtr    = Opcode(29) // Get free variable pointer object
	OpGetFree       = Opcode(30) // Get free variables
	OpSetFree       = Opcode(31) // Set free variables
	OpGetLocalPtr   = Opcode(32) // Get local variable as a pointer
	OpSetSelFree    = Opcode(33) // Set free variables using selectors
	OpGetBuiltin    = Opcode(34) // Get builtin function
	OpClosure       = Opcode(35) // Push closure
	OpIteratorInit  = Opcode(36) // Iterator init
	OpIteratorNext  = Opcode(37) // Iterator next
	OpIteratorKey   = Opcode(38) // Iterator key
	OpIteratorValue = Opcode(39) // Iterator value
	OpBinaryOp      = Opcode(40) // Binary operation
	OpSuspend       = Opcode(41) // Suspend VM
	OpSelect        = Opcode(42) // Select operation
	OpMethodCall    = Opcode(43) // Call method on object
	OpContains      = Opcode(44) // Contains operation (x in y)
	// 45...255 are reserved for future use
)

// OpcodeNames are string representation of opcodes.
var OpcodeNames = [...]string{
	OpConstant:      "CONST",
	OpPop:           "POP",
	OpTrue:          "TRUE",
	OpFalse:         "FALSE",
	OpBComplement:   "NEG",
	OpEqual:         "EQL",
	OpNotEqual:      "NEQ",
	OpMinus:         "NEG",
	OpLNot:          "NOT",
	OpJumpFalsy:     "JMPF",
	OpAndJump:       "ANDJMP",
	OpOrJump:        "ORJMP",
	OpJump:          "JMP",
	OpNull:          "NULL",
	OpGetGlobal:     "GETG",
	OpSetGlobal:     "SETG",
	OpSetSelGlobal:  "SETSG",
	OpArray:         "ARR",
	OpRecord:        "RECORD",
	OpError:         "ERROR",
	OpImmutable:     "IMMUT",
	OpIndex:         "INDEX",
	OpSliceIndex:    "SLICE",
	OpCall:          "CALL",
	OpReturn:        "RET",
	OpGetLocal:      "GETL",
	OpSetLocal:      "SETL",
	OpDefineLocal:   "DEFL",
	OpSetSelLocal:   "SETSL",
	OpGetBuiltin:    "BUILTIN",
	OpClosure:       "CLOSURE",
	OpGetFreePtr:    "GETFP",
	OpGetFree:       "GETF",
	OpSetFree:       "SETF",
	OpGetLocalPtr:   "GETLP",
	OpSetSelFree:    "SETSF",
	OpIteratorInit:  "ITER",
	OpIteratorNext:  "ITNXT",
	OpIteratorKey:   "ITKEY",
	OpIteratorValue: "ITVAL",
	OpBinaryOp:      "BINARYOP",
	OpSuspend:       "SUSPEND",
	OpSelect:        "SELECT",
	OpMethodCall:    "MCALL",
	OpContains:      "CONTAINS",
}

// OpcodeOperands is the number of operands.
var OpcodeOperands = [...][]int{
	OpConstant:      {2},
	OpPop:           {},
	OpTrue:          {},
	OpFalse:         {},
	OpBComplement:   {},
	OpEqual:         {},
	OpNotEqual:      {},
	OpMinus:         {},
	OpLNot:          {},
	OpJumpFalsy:     {4},
	OpAndJump:       {4},
	OpOrJump:        {4},
	OpJump:          {4},
	OpNull:          {},
	OpGetGlobal:     {2},
	OpSetGlobal:     {2},
	OpSetSelGlobal:  {2, 1},
	OpArray:         {2},
	OpRecord:        {2},
	OpError:         {},
	OpImmutable:     {},
	OpIndex:         {},
	OpSliceIndex:    {},
	OpCall:          {1, 1},
	OpReturn:        {1},
	OpGetLocal:      {1},
	OpSetLocal:      {1},
	OpDefineLocal:   {1},
	OpSetSelLocal:   {1, 1},
	OpGetBuiltin:    {1},
	OpClosure:       {2, 1},
	OpGetFreePtr:    {1},
	OpGetFree:       {1},
	OpSetFree:       {1},
	OpGetLocalPtr:   {1},
	OpSetSelFree:    {1, 1},
	OpIteratorInit:  {},
	OpIteratorNext:  {},
	OpIteratorKey:   {},
	OpIteratorValue: {},
	OpBinaryOp:      {1},
	OpSuspend:       {},
	OpSelect:        {},
	OpMethodCall:    {2, 1, 1}, // method const index, numArgs, spread
	OpContains:      {},
}

// ReadOperands reads operands from the bytecode.
func ReadOperands(numOperands []int, ins []byte) ([]int, int) {
	operands := make([]int, 0, len(numOperands))
	var offset int
	for _, width := range numOperands {
		switch width {
		case 1:
			operands = append(operands, int(ins[offset]))
		case 2:
			operands = append(operands, int(ins[offset+1])|int(ins[offset])<<8)
		case 4:
			operands = append(operands, int(ins[offset+3])|int(ins[offset+2])<<8|int(ins[offset+1])<<16|int(ins[offset])<<24)
		}
		offset += width
	}
	return operands, offset
}
