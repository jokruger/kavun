package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
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

type NativeFunc = func(VM, []Value) (Value, error)
type Pos int

func (p Pos) IsValid() bool {
	return p != NoPos
}

const (
	// Pos constants
	NoPos Pos = 0

	// Builtin module/function slot constants
	ModuleSlotSize = 128
	MaxModules     = 32
)

// Builtin module/function registry
var BuiltinFunctions [MaxModules * ModuleSlotSize]*BuiltinFunction

// Primitive value (used in static storage)
type Primitive struct {
	Type uint8
	Data uint64
}

func (p Primitive) Value() Value {
	return Value{Type: p.Type, Immutable: true, Data: p.Data}
}

// Static variables
type Static struct {
	Primitives        []Primitive
	Decimals          []dec128.Dec128
	Strings           []string
	Runes             []Runes
	Bytes             []Bytes
	FormatSpecs       []FormatSpec
	CompiledFunctions []CompiledFunction
}

// ValueTypeDescr is a Kavun data type descriptor structure.
type ValueTypeDescr struct {
	Name         func(v Value) string
	String       func(v Value) string
	Format       func(v Value, sp fspec.FormatSpec) (string, error)
	Interface    func(v Value) any
	EncodeJSON   func(v Value) ([]byte, error)
	EncodeBinary func(v Value) ([]byte, error)
	DecodeBinary func(v *Value, data []byte) error
	IsTrue       func(v Value) bool
	Clone        func(v Value) (Value, error)
	Equal        func(v Value, r Value) bool
	UnaryOp      func(v Value, op token.Token) (Value, error)
	BinaryOp     func(v Value, r Value, op token.Token) (Value, error)
	MethodCall   func(vm VM, v Value, name string, args []Value) (Value, error)

	IsIterable func(v Value) bool
	Contains   func(v Value, e Value) bool
	Len        func(v Value) int64
	Iterator   func(v Value) (Value, error)
	Access     func(v Value, index Value, mode opcode.Opcode) (Value, error)
	Assign     func(v Value, index Value, r Value) error
	Append     func(v Value, args []Value) (Value, error)
	Slice      func(v Value, s Value, e Value) (Value, error)
	Delete     func(v Value, key Value) (Value, error)
	SliceStep  func(v Value, s Value, e Value, step Value) (Value, error)

	IsCallable func(v Value) bool
	IsVariadic func(v Value) bool
	Arity      func(v Value) int8
	Call       func(vm VM, v Value, args []Value) (Value, error)

	Next  func(v Value) bool
	Key   func(v Value) (Value, error)
	Value func(v Value) (Value, error)

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
	AsArray   func(v Value) ([]Value, bool)
	AsDict    func(v Value) (map[string]Value, bool)
}

// DefaultValueType provides default implementations for all ValueType hooks.
var DefaultValueType = ValueTypeDescr{
	Name:         func(v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },
	String:       func(v Value) string { return v.TypeName() },
	Format:       defaultFormat,
	Interface:    func(_ Value) any { return nil },
	EncodeJSON:   func(v Value) ([]byte, error) { return nil, errs.NewJSONEncodingError(v.TypeName()) },
	EncodeBinary: func(v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName()) },
	DecodeBinary: func(v *Value, _ []byte) error { return errs.NewBinaryEncodingError(v.TypeName()) },
	IsTrue:       ConstHook(false),
	Clone:        func(v Value) (Value, error) { return v, nil },
	Equal:        func(v Value, r Value) bool { return v == r },

	UnaryOp:    defaultUnaryOp,
	BinaryOp:   defaultBinaryOp,
	MethodCall: defaultMethodCall,

	IsIterable: ConstHook(false),
	Contains:   func(Value, Value) bool { return false },
	Len:        ConstHook(int64(0)),
	Iterator:   ValueHook(Undefined, nil),
	Assign:     func(v Value, _, _ Value) error { return errs.NewNotAssignableError(v.TypeName()) },
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
	AsArray:   func(Value) ([]Value, bool) { return nil, false },
	AsDict:    func(Value) (map[string]Value, bool) { return nil, false },
	AsRunes:   defaultAsRunes,
}

// ValueTypes is the global registry of value type descriptors, indexed by type ID.
var ValueTypes [256]ValueTypeDescr

// SetValueType registers a user-defined value type descriptor for the given type ID.
func SetValueType(t uint8, f ValueTypeDescr) error {
	if t < value.FirstUserDefinedType {
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
