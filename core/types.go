package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

type Decimal = dec128.Dec128
type Time = time.Time
type Opcode = byte
type NativeFunc = func(VM, []Value) (Value, error)

// Allocator creates values and may reuse their storage when possible.
//
// For simple values whose allocation is limited to the value object itself,
// the lifecycle is straightforward: the caller obtains the value through the
// corresponding New...Value method, uses it, and calls ReleaseValue when the
// value is no longer needed. At that point the allocator may either free the
// underlying storage or keep it for future reuse.
//
// For complex values, ReleaseValue applies only to the value envelope managed
// by the allocator, not to resources stored inside that envelope. For example,
// a compiled function may reference instruction buffers, free variables,
// source maps, and similar data provided by the caller. Releasing the compiled
// function value allows the allocator to recycle only the compiled function
// object itself; the referenced resources remain owned by the caller and are
// reclaimed by the GC unless they are managed separately.
//
// The intended allocation strategy is therefore split by ownership boundary:
// envelope objects are created with New...Value and returned with ReleaseValue,
// while internal resources that have their own allocator support must be
// released independently.
//
// In case of StringValue, the allocator may reuse the string wrapper objects,
// but the underlying string data is immutable and shared by all wrappers,
// so it is not managed by the allocator.
//
// In case of ArrayValue, BytesValue, MapValue, etc., the allocator may reuse
// the wrapper objects as well as the underlying data buffers, which are managed
// by their own New/Release methods.
type Allocator interface {
	NewDecimal() (*Decimal, error)
	ReleaseDecimal(d *Decimal)

	NewTime() (*Time, error)
	ReleaseTime(t *Time)

	NewRunes(capacity int) ([]rune, error)
	ReleaseRunes(r []rune)

	NewBuiltinFunctionValue(name string, val NativeFunc, arity int8, variadic bool) (Value, error)
	NewCompiledFunctionValue(instructions []byte, free []*Value, sourceMap map[int]Pos, numLocals int, numParameters int8, varArgs bool) (Value, error)
	NewErrorValue(e Value) (Value, error)
	NewStringValue(s string) (Value, error)
	NewRunesValue(r []rune) (Value, error)
	NewIntRangeValue(start, stop, step int64) (Value, error)
	NewRunesIteratorValue(s []rune) (Value, error)
	NewBytesIteratorValue(b []byte) (Value, error)
	NewArrayIteratorValue(arr []Value) (Value, error)
	NewMapIteratorValue(m map[string]Value) (Value, error)
	NewIntRangeIteratorValue(start, stop, step int64) (Value, error)

	ReleaseValue(v Value)

	NewBytesValue(b []byte) (Value, error)
	NewArrayValue(arr []Value, immutable bool) (Value, error)
	NewMapValue(m map[string]Value, immutable bool) (Value, error)
	NewRecordValue(m map[string]Value, immutable bool) (Value, error)
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
	VT_RUNE               = uint8(6)
	VT_INT                = uint8(7)
	VT_FLOAT              = uint8(8)
	VT_DECIMAL            = uint8(9)
	VT_TIME               = uint8(10)
	VT_STRING             = uint8(11)
	VT_RUNES              = uint8(12)
	VT_BYTES              = uint8(13)
	VT_ARRAY              = uint8(14)
	VT_RECORD             = uint8(15)
	VT_MAP                = uint8(16)
	VT_INT_RANGE          = uint8(17)
	VT_RUNES_ITERATOR     = uint8(18)
	VT_BYTES_ITERATOR     = uint8(19)
	VT_ARRAY_ITERATOR     = uint8(20)
	VT_MAP_ITERATOR       = uint8(21)
	VT_INT_RANGE_ITERATOR = uint8(22)
	VT_USER_DEFINED       = uint8(23) // must be last
)

type ValueType struct {
	Name         func(v Value) string
	String       func(v Value) string
	Interface    func(v Value) any
	EncodeJSON   func(v Value) ([]byte, error)
	EncodeBinary func(v Value) ([]byte, error)
	DecodeBinary func(v *Value, data []byte) error
	IsTrue       func(v Value) bool
	Copy         func(v Value, a Allocator) (Value, error)
	Equal        func(v Value, r Value) bool
	UnaryOp      func(v Value, a Allocator, op token.Token) (Value, error)
	BinaryOp     func(v Value, a Allocator, op token.Token, r Value) (Value, error)
	MethodCall   func(v Value, vm VM, name string, args []Value) (Value, error)

	IsIterable func(v Value) bool
	Contains   func(v Value, e Value) bool
	Len        func(v Value) int64
	Iterator   func(v Value, a Allocator) (Value, error)
	Access     func(v Value, a Allocator, index Value, mode Opcode) (Value, error)
	Assign     func(v Value, index Value, r Value) error
	Append     func(v Value, a Allocator, args []Value) (Value, error)
	Slice      func(v Value, a Allocator, s Value, e Value) (Value, error)
	Delete     func(v Value, key Value) (Value, error)

	IsCallable func(v Value) bool
	IsVariadic func(v Value) bool
	Arity      func(v Value) int8
	Call       func(v Value, vm VM, args []Value) (Value, error)

	Next  func(v Value) bool
	Key   func(v Value, a Allocator) (Value, error)
	Value func(v Value, a Allocator) (Value, error)

	AsBool    func(v Value) (bool, bool)
	AsRune    func(v Value) (rune, bool)
	AsInt     func(v Value) (int64, bool)
	AsFloat   func(v Value) (float64, bool)
	AsDecimal func(v Value) (Decimal, bool)
	AsTime    func(v Value) (Time, bool)
	AsString  func(v Value) (string, bool)
	AsRunes   func(v Value) ([]rune, bool)
	AsBytes   func(v Value) ([]byte, bool)
	AsArray   func(v Value, a Allocator) ([]Value, bool)
	AsMap     func(v Value, a Allocator) (map[string]Value, bool)
}

var ValueTypeDefaults = ValueType{
	Name:         defaultTypeName,
	String:       defaultTypeString,
	Interface:    defaultTypeInterface,
	EncodeJSON:   defaultTypeEncodeJSON,
	EncodeBinary: defaultTypeEncodeBinary,
	DecodeBinary: defaultTypeDecodeBinary,
	IsTrue:       defaultFalse,
	Copy:         defaultSelf,
	Equal:        defaultTypeEqualPrimitive,
	UnaryOp:      defaultTypeUnaryOp,
	BinaryOp:     defaultTypeBinaryOp,
	MethodCall:   defaultTypeMethodCall,

	IsIterable: defaultFalse,
	Contains:   defaultTypeContains,
	Len:        default0,
	Iterator:   defaultUndefined,
	Access:     defaultTypeAccess,
	Assign:     defaultTypeAssign,
	Append:     defaultTypeAppend,
	Slice:      defaultTypeSlice,
	Delete:     defaultTypeDelete,

	IsCallable: defaultFalse,
	IsVariadic: defaultFalse,
	Arity:      defaultTypeArity,
	Call:       defaultTypeCall,

	Next:  defaultFalse,
	Key:   defaultUndefined,
	Value: defaultUndefined,

	AsBool:    defaultTypeAsBool,
	AsRune:    defaultTypeAsRune,
	AsInt:     defaultTypeAsInt,
	AsFloat:   defaultTypeAsFloat,
	AsDecimal: defaultTypeAsDecimal,
	AsTime:    defaultTypeAsTime,
	AsString:  defaultTypeAsString,
	AsRunes:   defaultTypeAsRunes,
	AsBytes:   defaultTypeAsBytes,
	AsArray:   defaultTypeAsArray,
	AsMap:     defaultTypeAsMap,
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
	OpContains      = Opcode(16) // Contains operation (x in y)
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
	// 44...255 are reserved for future use
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

func defaultUndefined(v Value, a Allocator) (Value, error) {
	return Undefined, nil
}

func defaultSelf(v Value, a Allocator) (Value, error) {
	return v, nil
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

func defaultTypeAsRune(v Value) (rune, bool) {
	return 0, false
}

func defaultTypeAsInt(v Value) (int64, bool) {
	return 0, false
}

func defaultTypeAsFloat(v Value) (float64, bool) {
	return 0, false
}

func defaultTypeAsDecimal(v Value) (Decimal, bool) {
	return dec128.Decimal0, false
}

func defaultTypeAsTime(v Value) (Time, bool) {
	return Time{}, false
}

func defaultTypeAsString(v Value) (string, bool) {
	return "", false
}

func defaultTypeAsRunes(v Value) ([]rune, bool) {
	s, ok := v.AsString()
	if !ok {
		return nil, false
	}
	return []rune(s), true
}

func defaultTypeAsBytes(v Value) ([]byte, bool) {
	return nil, false
}

func defaultTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	return nil, false
}

func defaultTypeAsMap(v Value, a Allocator) (map[string]Value, bool) {
	return nil, false
}

func defaultTypeEqualPrimitive(v Value, r Value) bool {
	// ignore immutability flag
	return v.Type == r.Type && v.Data == r.Data && v.Ptr == r.Ptr
}

func defaultTypeUnaryOp(v Value, a Allocator, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
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

func defaultTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	return Undefined, errs.NewInvalidSliceError(v.TypeName())
}
