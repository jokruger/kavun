package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/hook"
	"github.com/jokruger/kavun/token"
)

type NativeFunc = func(VM, []Value) (Value, error)

type VM interface {
	Allocator() *Arena                              // returns the arena allocator used by this VM
	Abort()                                         // aborts execution of the current script
	IsStackEmpty() bool                             // returns true if there are no frames on the call stack
	Call(*CompiledFunction, []Value) (Value, error) // calls a compiled function
	Run() error                                     // runs the VM until completion
	Recover() Value                                 // returns the in-flight error if in "deferred-for" frame
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
	VT_FORMAT_SPEC        = uint8(4)
	VT_ERROR              = uint8(5)
	VT_BOOL               = uint8(6)
	VT_BYTE               = uint8(7)
	VT_RUNE               = uint8(8)
	VT_INT                = uint8(9)
	VT_FLOAT              = uint8(10)
	VT_DECIMAL            = uint8(11)
	VT_TIME               = uint8(12)
	VT_STRING             = uint8(13)
	VT_RUNES              = uint8(14)
	VT_BYTES              = uint8(15)
	VT_ARRAY              = uint8(16)
	VT_RECORD             = uint8(17)
	VT_DICT               = uint8(18)
	VT_INT_RANGE          = uint8(19)
	VT_RUNES_ITERATOR     = uint8(20)
	VT_BYTES_ITERATOR     = uint8(21)
	VT_ARRAY_ITERATOR     = uint8(22)
	VT_DICT_ITERATOR      = uint8(23)
	VT_INT_RANGE_ITERATOR = uint8(24)
	VT_USER_DEFINED       = uint8(25) // must be last
)

type ValueType struct {
	Name         func(v Value) string
	String       func(v Value) string
	Format       func(v Value, sp fspec.FormatSpec) (string, error)
	Interface    func(v Value) any
	EncodeJSON   func(v Value) ([]byte, error)
	EncodeBinary func(v Value) ([]byte, error)
	DecodeBinary func(v *Value, data []byte) error
	IsTrue       func(v Value) bool
	Copy         func(v Value, a *Arena) (Value, error)
	Equal        func(v Value, r Value) bool
	UnaryOp      func(v Value, a *Arena, op token.Token) (Value, error)
	BinaryOp     func(v Value, a *Arena, op token.Token, r Value) (Value, error)
	MethodCall   func(v Value, vm VM, name string, args []Value) (Value, error)

	IsIterable func(v Value) bool
	Contains   func(v Value, e Value) bool
	Len        func(v Value) int64
	Iterator   func(v Value, a *Arena) (Value, error)
	Access     func(v Value, a *Arena, index Value, mode bc.Opcode) (Value, error)
	Assign     func(v Value, index Value, r Value) error
	Append     func(v Value, a *Arena, args []Value) (Value, error)
	Slice      func(v Value, a *Arena, s Value, e Value) (Value, error)
	Delete     func(v Value, key Value) (Value, error)
	SliceStep  func(v Value, a *Arena, s Value, e Value, step Value) (Value, error)

	IsCallable func(v Value) bool
	IsVariadic func(v Value) bool
	Arity      func(v Value) int8
	Call       func(v Value, vm VM, args []Value) (Value, error)

	Next  func(v Value) bool
	Key   func(v Value, a *Arena) (Value, error)
	Value func(v Value, a *Arena) (Value, error)

	AsBool    func(v Value) (bool, bool)
	AsByte    func(v Value) (byte, bool)
	AsRune    func(v Value) (rune, bool)
	AsInt     func(v Value) (int64, bool)
	AsFloat   func(v Value) (float64, bool)
	AsDecimal func(v Value) (dec128.Dec128, bool)
	AsTime    func(v Value) (time.Time, bool)
	AsString  func(v Value) (string, bool)
	AsRunes   func(v Value) ([]rune, bool)
	AsBytes   func(v Value) ([]byte, bool)
	AsArray   func(v Value, a *Arena) ([]Value, bool)
	AsDict    func(v Value, a *Arena) (map[string]Value, bool)
}

var ValueTypeDefaults = ValueType{
	Name:         func(v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },
	String:       defaultTypeString,
	Format:       defaultTypeFormat,
	Interface:    defaultTypeInterface,
	EncodeJSON:   func(v Value) ([]byte, error) { return nil, errs.NewJSONEncodingError(v.TypeName()) },
	EncodeBinary: func(v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName()) },
	DecodeBinary: func(v *Value, data []byte) error { return errs.NewBinaryEncodingError(v.TypeName()) },
	IsTrue:       hook.Const[Value, bool](false),
	Copy:         hook.Self[Value, *Arena],
	Equal:        defaultTypeEqualPrimitive,
	UnaryOp:      defaultTypeUnaryOp,
	BinaryOp:     defaultTypeBinaryOp,
	MethodCall:   defaultTypeMethodCall,

	IsIterable: hook.Const[Value, bool](false),
	Contains:   defaultTypeContains,
	Len:        hook.Const[Value, int64](0),
	Iterator:   hook.Value[Value, *Arena](Undefined, nil),
	Access:     defaultTypeAccess,
	Assign:     defaultTypeAssign,
	Append:     defaultTypeAppend,
	Slice:      defaultTypeSlice,
	Delete:     defaultTypeDelete,
	SliceStep:  defaultTypeSliceStep,

	IsCallable: hook.Const[Value, bool](false),
	IsVariadic: hook.Const[Value, bool](false),
	Arity:      defaultTypeArity,
	Call:       defaultTypeCall,

	Next:  hook.Const[Value, bool](false),
	Key:   hook.Value[Value, *Arena](Undefined, nil),
	Value: hook.Value[Value, *Arena](Undefined, nil),

	AsBool:    defaultTypeAsBool,
	AsByte:    defaultTypeAsByte,
	AsRune:    defaultTypeAsRune,
	AsInt:     defaultTypeAsInt,
	AsFloat:   defaultTypeAsFloat,
	AsDecimal: defaultTypeAsDecimal,
	AsTime:    defaultTypeAsTime,
	AsString:  defaultTypeAsString,
	AsRunes:   defaultTypeAsRunes,
	AsBytes:   defaultTypeAsBytes,
	AsArray:   defaultTypeAsArray,
	AsDict:    defaultTypeAsDict,
}

var ValueTypes [256]ValueType

func SetValueType(t uint8, f ValueType) error {
	if t < VT_USER_DEFINED {
		return fmt.Errorf("cannot set value type for built-in type %d", t)
	}
	setValueType(t, f)
	return nil
}

func setValueType(t uint8, f ValueType) {
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

func defaultTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	return "", fmt.Errorf("value type %s does not support formatting", v.TypeName())
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

func defaultTypeAsByte(v Value) (byte, bool) {
	return 0, false
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

func defaultTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	return dec128.Decimal0, false
}

func defaultTypeAsTime(v Value) (time.Time, bool) {
	return time.Time{}, false
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

func defaultTypeAsArray(v Value, a *Arena) ([]Value, bool) {
	return nil, false
}

func defaultTypeAsDict(v Value, a *Arena) (map[string]Value, bool) {
	return nil, false
}

func defaultTypeEqualPrimitive(v Value, r Value) bool {
	// ignore immutability flag
	return v.Type == r.Type && v.Data == r.Data && v.Ptr == r.Ptr
}

func defaultTypeUnaryOp(v Value, a *Arena, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

func defaultTypeBinaryOp(v Value, a *Arena, op token.Token, r Value) (Value, error) {
	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func defaultTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

func defaultTypeAccess(v Value, a *Arena, index Value, mode bc.Opcode) (Value, error) {
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

func defaultTypeAppend(v Value, a *Arena, args []Value) (Value, error) {
	return Undefined, errs.NewNotAppendableError(v.TypeName())
}

func defaultTypeDelete(v Value, key Value) (Value, error) {
	return Undefined, errs.NewNotDeletableError(v.TypeName())
}

func defaultTypeSlice(v Value, a *Arena, s Value, e Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

func defaultTypeSliceStep(v Value, a *Arena, s Value, e Value, step Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}
