package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

type Opcode = byte
type NativeFunc = func(VM, []Value) (Value, error)

type Allocator interface {
	NewBuiltinFunctionValue(name string, val NativeFunc, arity int8, variadic bool) Value
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

type ValueType struct {
	Name         func(v Value) string
	EncodeJSON   func(v Value) ([]byte, error)
	EncodeBinary func(v Value) ([]byte, error)
	DecodeBinary func(v *Value, data []byte) error
	String       func(v Value) string
	Interface    func(v Value) any

	IsTrue      func(v Value) bool
	IsImmutable func(v Value) bool
	IsIterable  func(v Value) bool
	IsCallable  func(v Value) bool
	Contains    func(v Value, e Value) bool

	AsBool   func(v Value) (bool, bool)
	AsChar   func(v Value) (rune, bool)
	AsInt    func(v Value) (int64, bool)
	AsFloat  func(v Value) (float64, bool)
	AsTime   func(v Value) (time.Time, bool)
	AsString func(v Value) (string, bool)
	AsBytes  func(v Value) ([]byte, bool)

	Len        func(v Value) int64
	Copy       func(v Value, a Allocator) Value
	Equal      func(v Value, r Value) bool
	BinaryOp   func(v Value, a Allocator, op token.Token, r Value) (Value, error)
	MethodCall func(v Value, vm VM, name string, args []Value) (Value, error)

	Access   func(v Value, a Allocator, index Value, mode Opcode) (Value, error)
	Assign   func(v Value, index Value, r Value) error
	Iterator func(v Value, a Allocator) Value
	Append   func(v Value, a Allocator, args []Value) (Value, error)
	Delete   func(v Value, key Value) (Value, error)

	Next  func(v Value) bool
	Key   func(v Value, a Allocator) Value
	Value func(v Value, a Allocator) Value

	Arity      func(v Value) int8
	IsVariadic func(v Value) bool
	Call       func(v Value, vm VM, args []Value) (Value, error)
}

var ValueTypeDefaults = ValueType{
	Name:         defaultTypeName,
	EncodeJSON:   defaultTypeEncodeJSON,
	EncodeBinary: defaultTypeEncodeBinary,
	DecodeBinary: defaultTypeDecodeBinary,
	String:       defaultTypeString,
	Interface:    defaultTypeInterface,

	IsTrue:      defaultFalse,
	IsImmutable: defaultFalse,
	IsIterable:  defaultFalse,
	IsCallable:  defaultFalse,
	Contains:    defaultTypeContains,

	AsBool:   defaultTypeAsBool,
	AsChar:   defaultTypeAsChar,
	AsInt:    defaultTypeAsInt,
	AsFloat:  defaultTypeAsFloat,
	AsTime:   defaultTypeAsTime,
	AsString: defaultTypeAsString,
	AsBytes:  defaultTypeAsBytes,

	Len:        default0,
	Copy:       defaultTypeCopy,
	Equal:      defaultTypeEqualPrimitive,
	BinaryOp:   defaultTypeBinaryOp,
	MethodCall: defaultTypeMethodCall,

	Access:   defaultTypeAccess,
	Assign:   defaultTypeAssign,
	Iterator: defaultUndefined,
	Append:   defaultTypeAppend,
	Delete:   defaultTypeDelete,

	Next:  defaultFalse,
	Key:   defaultUndefined,
	Value: defaultUndefined,

	Arity:      defaultTypeArity,
	IsVariadic: defaultFalse,
	Call:       defaultTypeCall,
}

var ValueTypes [256]ValueType

func SetValueType(t uint8, f ValueType) {
	fv := reflect.ValueOf(&f).Elem()
	dv := reflect.ValueOf(ValueTypeDefaults)

	for i := 0; i < fv.NumField(); i++ {
		field := fv.Field(i)
		if field.IsNil() {
			field.Set(dv.Field(i))
		}
	}

	ValueTypes[t] = f
}

var (
	// Value shortcuts
	True      = BoolValue(true)
	False     = BoolValue(false)
	Undefined = UndefinedValue()

	// MaxStringLen is the maximum byte-length for string value. Note this limit applies to all compiler/VM instances in the process.
	MaxStringLen = 2147483647

	// MaxBytesLen is the maximum length for bytes value. Note this limit applies to all compiler/VM instances in the process.
	MaxBytesLen = 2147483647
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

func default0(v Value) int64 {
	return 0
}

func default1(v Value) int64 {
	return 1
}

func defaultTrue(v Value) bool {
	return true
}

func defaultFalse(v Value) bool {
	return false
}

func defaultUndefined(v Value, a Allocator) Value {
	return Undefined
}

func defaultTypeName(v Value) string {
	return fmt.Sprintf("<unknown:%d>", v.Type)
}

func defaultTypeEncodeJSON(v Value) ([]byte, error) {
	return nil, fmt.Errorf("value type %s does not support JSON encoding", v.TypeName())
}

func defaultTypeEncodeBinary(v Value) ([]byte, error) {
	return nil, fmt.Errorf("value type %s does not support binary encoding", v.TypeName())
}

func defaultTypeDecodeBinary(v *Value, data []byte) error {
	return fmt.Errorf("value type %s does not support binary decoding", v.TypeName())
}

func defaultTypeString(v Value) string {
	return v.TypeName()
}

func defaultTypeInterface(v Value) any {
	return nil
}

func defaultTypeAsBool(v Value) (bool, bool) {
	return false, false
}

func defaultTypeAsChar(v Value) (rune, bool) {
	return 0, false
}

func defaultTypeAsInt(v Value) (int64, bool) {
	return 0, false
}

func defaultTypeAsFloat(v Value) (float64, bool) {
	return 0, false
}

func defaultTypeAsTime(v Value) (time.Time, bool) {
	return time.Time{}, false
}

func defaultTypeAsString(v Value) (string, bool) {
	return "", false
}

func defaultTypeAsBytes(v Value) ([]byte, bool) {
	return nil, false
}

func defaultTypeCopy(v Value, a Allocator) Value {
	// by default copy as primitive value (used by Int, Float, etc)
	return v
}

func defaultTypeEqualPrimitive(v Value, r Value) bool {
	return v == r
}

func defaultTypeBinaryOp(v Value, a Allocator, op token.Token, r Value) (Value, error) {
	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func defaultTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

func defaultTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

func defaultTypeAssign(v Value, index Value, r Value) error {
	return errs.NewNotAssignableError(v.TypeName())
}

func defaultTypeArity(v Value) int8 {
	return 0
}

func defaultTypeCall(v Value, vm VM, args []Value) (Value, error) {
	return Undefined, errs.NewNotCallableError(v.TypeName())
}

func defaultTypeContains(v Value, item Value) bool {
	return false
}

func defaultTypeAppend(v Value, a Allocator, args []Value) (Value, error) {
	return Undefined, errs.NewInvalidAppendError(v.TypeName())
}

func defaultTypeDelete(v Value, key Value) (Value, error) {
	return Undefined, errs.NewInvalidDeleteError(v.TypeName())
}
