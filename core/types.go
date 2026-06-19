package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

type VM interface {
	Abort()                             // aborts execution of the current script
	IsStackEmpty() bool                 // returns true if there are no frames on the call stack
	Call(Value, []Value) (Value, error) // calls a compiled function
	Run() error                         // runs the VM until completion
	Recover() Value                     // returns the in-flight error if in "deferred-for" frame
}

type NativeFunc = func(*Arena, VM, []Value) (Value, error)
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
	VT_BUILTIN_CLOSURE    = uint8(3)
	VT_COMPILED_FUNCTION  = uint8(4)
	VT_FORMAT_SPEC        = uint8(5)
	VT_ERROR              = uint8(6)
	VT_BOOL               = uint8(7)
	VT_BYTE               = uint8(8)
	VT_RUNE               = uint8(9)
	VT_INT                = uint8(10)
	VT_FLOAT              = uint8(11)
	VT_DECIMAL            = uint8(12)
	VT_TIME               = uint8(13)
	VT_STRING             = uint8(14)
	VT_RUNES              = uint8(15)
	VT_BYTES              = uint8(16)
	VT_ARRAY              = uint8(17)
	VT_RECORD             = uint8(18)
	VT_DICT               = uint8(19)
	VT_INT_RANGE          = uint8(20)
	VT_RUNES_ITERATOR     = uint8(21)
	VT_BYTES_ITERATOR     = uint8(22)
	VT_ARRAY_ITERATOR     = uint8(23)
	VT_DICT_ITERATOR      = uint8(24)
	VT_INT_RANGE_ITERATOR = uint8(25)
	VT_USER_DEFINED       = uint8(26) // must be last

	ModuleSlotSize = 128
	MaxModules     = 32
)

var BuiltinFunctions [MaxModules * ModuleSlotSize]*BuiltinFunction

// ValueTypeDescr is a Kavun data type descriptor structure.
type ValueTypeDescr struct {
	Pin     func(a *Arena, v Value)
	Retain  func(a *Arena, v Value)
	Release func(a *Arena, v Value)

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
	Access     func(a *Arena, v Value, index Value, mode opcode.Opcode) (Value, error)
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

// DefaultValueType provides default implementations for all ValueType hooks.
var DefaultValueType = ValueTypeDescr{
	Pin:     func(*Arena, Value) {},
	Retain:  func(*Arena, Value) {},
	Release: func(*Arena, Value) {},

	Name:         func(_ *Arena, v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },
	String:       func(a *Arena, v Value) string { return v.TypeName(a) },
	Format:       defaultFormat,
	Interface:    func(_ *Arena, _ Value) any { return nil },
	EncodeJSON:   func(a *Arena, v Value) ([]byte, error) { return nil, errs.NewJSONEncodingError(v.TypeName(a)) },
	EncodeBinary: func(a *Arena, v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName(a)) },
	DecodeBinary: func(a *Arena, v *Value, _ []byte) error { return errs.NewBinaryEncodingError(v.TypeName(a)) },
	IsTrue:       ConstHook(false),
	Clone:        func(_ *Arena, v Value) (Value, error) { return v, nil },
	Equal:        func(_ *Arena, v Value, r Value) bool { return v == r },

	UnaryOp:    defaultUnaryOp,
	BinaryOp:   defaultBinaryOp,
	MethodCall: defaultMethodCall,

	IsIterable: ConstHook(false),
	Contains:   func(*Arena, Value, Value) bool { return false },
	Len:        ConstHook(int64(0)),
	Iterator:   ValueHook(Undefined, nil),
	Assign:     func(a *Arena, v Value, _, _ Value) error { return errs.NewNotAssignableError(v.TypeName(a)) },
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
var ValueTypes [256]ValueTypeDescr

// SetValueType registers a user-defined value type descriptor for the given type ID.
func SetValueType(t uint8, f ValueTypeDescr) error {
	if t < VT_USER_DEFINED {
		return fmt.Errorf("cannot set value type for built-in type %d", t)
	}
	setValueType(t, f)
	return nil
}

func setValueType(t uint8, f ValueTypeDescr) {
	fv := reflect.ValueOf(&f).Elem()
	dv := reflect.ValueOf(DefaultValueType)

	for i := 0; i < fv.NumField(); i++ {
		field := fv.Field(i)
		if field.IsNil() {
			field.Set(dv.Field(i))
		}
	}

	ValueTypes[t] = f
}
