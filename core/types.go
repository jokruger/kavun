package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

type VM interface {
	Allocator() *Arena                              // returns the arena allocator used by this VM
	Abort()                                         // aborts execution of the current script
	IsStackEmpty() bool                             // returns true if there are no frames on the call stack
	Call(*CompiledFunction, []Value) (Value, error) // calls a compiled function
	Run() error                                     // runs the VM until completion
	Recover() Value                                 // returns the in-flight error if in "deferred-for" frame
}

type NativeFunc = func(VM, []Value) (Value, error)
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

// ValueType is a Kavun data type descriptor structure.
type ValueType struct {
	Name         func(a *Arena, v Value) string
	String       func(a *Arena, v Value) string
	Format       func(a *Arena, v Value, sp fspec.FormatSpec) (string, error)
	Interface    func(a *Arena, v Value) any
	EncodeJSON   func(a *Arena, v Value) ([]byte, error)
	EncodeBinary func(a *Arena, v Value) ([]byte, error)
	DecodeBinary func(a *Arena, v *Value, data []byte) error
	IsTrue       func(a *Arena, v Value) bool
	Clone        func(a *Arena, v Value) (Value, error)
	Equal        func(a *Arena, v Value, r Value) bool
	UnaryOp      func(a *Arena, v Value, op token.Token) (Value, error)
	BinaryOp     func(a *Arena, v Value, r Value, op token.Token) (Value, error)
	MethodCall   func(a *Arena, vm VM, v Value, name string, args []Value) (Value, error)

	IsIterable func(a *Arena, v Value) bool
	Contains   func(a *Arena, v Value, e Value) bool
	Len        func(a *Arena, v Value) int64
	Iterator   func(a *Arena, v Value) (Value, error)
	Access     func(a *Arena, v Value, index Value, mode bc.Opcode) (Value, error)
	Assign     func(a *Arena, v Value, index Value, r Value) error
	Append     func(a *Arena, v Value, args []Value) (Value, error)
	Slice      func(a *Arena, v Value, s Value, e Value) (Value, error)
	Delete     func(a *Arena, v Value, key Value) (Value, error)
	SliceStep  func(a *Arena, v Value, s Value, e Value, step Value) (Value, error)

	IsCallable func(a *Arena, v Value) bool
	IsVariadic func(a *Arena, v Value) bool
	Arity      func(a *Arena, v Value) int8
	Call       func(a *Arena, vm VM, v Value, args []Value) (Value, error)

	Next  func(a *Arena, v Value) bool
	Key   func(a *Arena, v Value) (Value, error)
	Value func(a *Arena, v Value) (Value, error)

	AsBool    func(a *Arena, v Value) (bool, bool)
	AsByte    func(a *Arena, v Value) (byte, bool)
	AsRune    func(a *Arena, v Value) (rune, bool)
	AsInt     func(a *Arena, v Value) (int64, bool)
	AsFloat   func(a *Arena, v Value) (float64, bool)
	AsDecimal func(a *Arena, v Value) (dec128.Dec128, bool)
	AsTime    func(a *Arena, v Value) (time.Time, bool)
	AsString  func(a *Arena, v Value) (string, bool)
	AsRunes   func(a *Arena, v Value) ([]rune, bool)
	AsBytes   func(a *Arena, v Value) ([]byte, bool)
	AsArray   func(a *Arena, v Value) ([]Value, bool)
	AsDict    func(a *Arena, v Value) (map[string]Value, bool)
}

// ValueTypeDefaults provides default implementations for all ValueType hooks.
var ValueTypeDefaults = ValueType{
	Name:         func(_ *Arena, v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },
	String:       func(_ *Arena, v Value) string { return v.TypeName() },
	Format:       defaultFormat,
	Interface:    func(_ *Arena, _ Value) any { return nil },
	EncodeJSON:   func(_ *Arena, v Value) ([]byte, error) { return nil, errs.NewJSONEncodingError(v.TypeName()) },
	EncodeBinary: func(_ *Arena, v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName()) },
	DecodeBinary: func(_ *Arena, v *Value, _ []byte) error { return errs.NewBinaryEncodingError(v.TypeName()) },
	IsTrue:       ConstHook(false),
	Clone:        func(_ *Arena, v Value) (Value, error) { return v, nil },
	Equal:        func(_ *Arena, v Value, r Value) bool { return v.Type == r.Type && v.Data == r.Data && v.Ptr == r.Ptr }, // ignore immutability

	UnaryOp:    defaultUnaryOp,
	BinaryOp:   defaultBinaryOp,
	MethodCall: defaultMethodCall,

	IsIterable: ConstHook(false),
	Contains:   func(*Arena, Value, Value) bool { return false },
	Len:        ConstHook(int64(0)),
	Iterator:   ValueHook(Undefined, nil),
	Assign:     func(_ *Arena, v Value, _, _ Value) error { return errs.NewNotAssignableError(v.TypeName()) },
	Delete:     defaultDelete,

	Access:    defaultAccess,
	Append:    defaultAppend,
	Slice:     defaultSlice,
	SliceStep: defaultSliceStep,

	IsCallable: ConstHook(false),
	IsVariadic: ConstHook(false),
	Arity:      ConstHook(int8(0)),

	Call: defaultCall,

	Next:  ConstHook(false),
	Key:   ValueHook(Undefined, nil),
	Value: ValueHook(Undefined, nil),

	AsBool:    Const2Hook(false, false),
	AsByte:    Const2Hook(byte(0), false),
	AsRune:    Const2Hook(rune(0), false),
	AsInt:     Const2Hook(int64(0), false),
	AsFloat:   Const2Hook(float64(0), false),
	AsDecimal: Const2Hook(dec128.Decimal0, false),
	AsTime:    Const2Hook(time.Time{}, false),
	AsString:  Const2Hook("", false),
	AsBytes:   Const2Hook[[]byte](nil, false),
	AsArray:   func(*Arena, Value) ([]Value, bool) { return nil, false },
	AsDict:    func(*Arena, Value) (map[string]Value, bool) { return nil, false },
	AsRunes:   defaultAsRunes,
}

// ValueTypes is the global registry of value type descriptors, indexed by type ID.
var ValueTypes [256]ValueType

// SetValueType registers a user-defined value type descriptor for the given type ID.
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

func defaultFormat(_ *Arena, v Value, _ fspec.FormatSpec) (string, error) {
	return "", errs.NewNoFormattingError(v.TypeName())
}

func defaultUnaryOp(_ *Arena, v Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

func defaultBinaryOp(_ *Arena, v Value, r Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func defaultMethodCall(_ *Arena, _ VM, v Value, name string, _ []Value) (Value, error) {
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

func defaultDelete(_ *Arena, v Value, _ Value) (Value, error) {
	return Undefined, errs.NewNotDeletableError(v.TypeName())
}

func defaultAccess(_ *Arena, v Value, _ Value, _ bc.Opcode) (Value, error) {
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

func defaultAppend(_ *Arena, v Value, _ []Value) (Value, error) {
	return Undefined, errs.NewNotAppendableError(v.TypeName())
}

func defaultSlice(_ *Arena, v Value, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

func defaultSliceStep(_ *Arena, v Value, _, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

func defaultCall(_ *Arena, _ VM, v Value, _ []Value) (Value, error) {
	return Undefined, errs.NewNotCallableError(v.TypeName())
}

func defaultAsRunes(_ *Arena, v Value) ([]rune, bool) {
	s, ok := v.AsString()
	if !ok {
		return nil, false
	}
	return []rune(s), true
}
