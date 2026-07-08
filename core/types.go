package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jokruger/dec128"
	bc "github.com/jokruger/kavun/core/bytecode"
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
	Times             []time.Time
	FormatSpecs       []FormatSpec
	CompiledFunctions []CompiledFunction
}

// ValueTypeDescr is a Kavun data type descriptor structure.
// See docs/purity.md for purity contract.
type ValueTypeDescr struct {
	Name         func(v Value) string                                           // PURE by contract
	String       func(v Value) string                                           // PURE by contract
	Format       func(v Value, sp fspec.FormatSpec) (string, error)             // PURE by contract
	Interface    func(v Value) any                                              // PURE by contract
	EncodeJSON   func(v Value) ([]byte, error)                                  // PURE by contract
	EncodeBinary func(v Value) ([]byte, error)                                  // PURE by contract
	DecodeBinary func(v *Value, data []byte) error                              // IMPURE by contract (mutates target)
	IsTrue       func(v Value) bool                                             // PURE by contract
	Clone        func(v Value) (Value, error)                                   // PURE by contract
	Equal        func(v Value, r Value) bool                                    // PURE by contract
	UnaryOp      func(v Value, op token.Token) (Value, error)                   // PURE by contract
	BinaryOp     func(v Value, r Value, op token.Token) (Value, error)          // PURE by contract
	MethodCall   func(vm VM, v Value, name string, args []Value) (Value, error) // PURE by contract with higher-order rule caveat (see docs/purity.md)

	IsIterable func(v Value) bool                                         // PURE by contract
	Contains   func(v Value, e Value) bool                                // PURE by contract
	Len        func(v Value) int64                                        // PURE by contract
	Iterator   func(v Value) (Value, error)                               // PURE by contract (constructs fresh iterator)
	Access     func(v Value, index Value, mode bc.Opcode) (Value, error)  // PURE by contract
	Assign     func(v Value, index Value, r Value) error                  // IMPURE by contract (mutates target)
	Append     func(v Value, args []Value) (Value, error)                 // GO-STYLE by contract (may share receiver storage)
	Slice      func(v Value, s Value, e Value) (Value, error)             // PURE by contract
	Delete     func(v Value, key Value) (Value, error)                    // IMPURE by contract (mutates target)
	SliceStep  func(v Value, s Value, e Value, step Value) (Value, error) // PURE by contract

	IsCallable func(v Value) bool                                // PURE by contract
	IsVariadic func(v Value) bool                                // PURE by contract
	Arity      func(v Value) int                                 // PURE by contract
	Call       func(vm VM, v Value, args []Value) (Value, error) // CALLABLE-DEPENDENT by contract

	Next  func(v Value) bool           // LOCALISED-STATE by contract (advances iterator cursor)
	Key   func(v Value) (Value, error) // LOCALISED-STATE by contract (reads iterator cursor)
	Value func(v Value) (Value, error) // LOCALISED-STATE by contract (reads iterator cursor)

	AsBool    func(v Value) (bool, bool)             // PURE by contract
	AsByte    func(v Value) (byte, bool)             // PURE by contract
	AsRune    func(v Value) (rune, bool)             // PURE by contract
	AsInt     func(v Value) (int64, bool)            // PURE by contract
	AsFloat   func(v Value) (float64, bool)          // PURE by contract
	AsDecimal func(v Value) (dec128.Dec128, bool)    // PURE by contract
	AsTime    func(v Value) (time.Time, bool)        // PURE by contract
	AsString  func(v Value) (string, bool)           // PURE by contract
	AsRunes   func(v Value) ([]rune, bool)           // PURE by contract
	AsBytes   func(v Value) ([]byte, bool)           // PURE by contract
	AsArray   func(v Value) ([]Value, bool)          // PURE by contract
	AsDict    func(v Value) (map[string]Value, bool) // PURE by contract
}

// DefaultValueType provides default implementations for all ValueType hooks.
var DefaultValueType = ValueTypeDescr{
	Name:         func(v Value) string { return fmt.Sprintf("<unknown:%d>", v.Type) },                     // PURE by contract
	String:       func(v Value) string { return v.TypeName() },                                            // PURE by contract
	Format:       defaultFormat,                                                                           // PURE by contract
	Interface:    func(_ Value) any { return nil },                                                        // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return nil, errs.NewJSONEncodingError(v.TypeName()) },   // PURE by contract
	EncodeBinary: func(v Value) ([]byte, error) { return nil, errs.NewBinaryEncodingError(v.TypeName()) }, // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { return errs.NewBinaryEncodingError(v.TypeName()) },     // IMPURE by contract (mutates target)
	IsTrue:       ConstHook(false),                                                                        // PURE by contract
	Clone:        func(v Value) (Value, error) { return v, nil },                                          // PURE by contract
	Equal:        func(v Value, r Value) bool { return v == r },                                           // PURE by contract

	UnaryOp:    defaultUnaryOp,    // PURE by contract
	BinaryOp:   defaultBinaryOp,   // PURE by contract
	MethodCall: defaultMethodCall, // PURE by contract with higher-order rule caveat (see docs/purity.md)

	IsIterable: ConstHook(false),                                                                    // PURE by contract
	Contains:   func(Value, Value) bool { return false },                                            // PURE by contract
	Len:        ConstHook(int64(0)),                                                                 // PURE by contract
	Iterator:   ValueHook(Undefined, nil),                                                           // PURE by contract (constructs fresh iterator)
	Assign:     func(v Value, _, _ Value) error { return errs.NewNotAssignableError(v.TypeName()) }, // IMPURE by contract
	Delete:     defaultDelete,                                                                       // IMPURE by contract

	Access:    defaultAccess,    // PURE by contract
	Append:    defaultAppend,    // GO-STYLE by contract (may share receiver storage)
	Slice:     defaultSlice,     // PURE by contract
	SliceStep: defaultSliceStep, // PURE by contract

	IsCallable: ConstHook(false), // PURE by contract
	IsVariadic: ConstHook(false), // PURE by contract
	Arity:      ConstHook(0),     // PURE by contract

	Call: defaultCall, // CALLABLE-DEPENDENT by contract

	Next:  ConstHook(false),          // LOCALISED-STATE by contract (advances iterator cursor)
	Key:   ValueHook(Undefined, nil), // LOCALISED-STATE by contract (reads iterator cursor)
	Value: ValueHook(Undefined, nil), // LOCALISED-STATE by contract (reads iterator cursor)

	AsBool:    Const2Hook(false, false),                                   // PURE by contract
	AsByte:    Const2Hook(byte(0), false),                                 // PURE by contract
	AsRune:    Const2Hook(rune(0), false),                                 // PURE by contract
	AsInt:     Const2Hook(int64(0), false),                                // PURE by contract
	AsFloat:   Const2Hook(float64(0), false),                              // PURE by contract
	AsDecimal: Const2Hook(dec128.Decimal0, false),                         // PURE by contract
	AsTime:    Const2Hook(time.Time{}, false),                             // PURE by contract
	AsString:  Const2Hook("", false),                                      // PURE by contract
	AsBytes:   Const2Hook[[]byte](nil, false),                             // PURE by contract
	AsArray:   func(Value) ([]Value, bool) { return nil, false },          // PURE by contract
	AsDict:    func(Value) (map[string]Value, bool) { return nil, false }, // PURE by contract
	AsRunes:   defaultAsRunes,                                             // PURE by contract
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
