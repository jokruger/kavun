package core

import (
	"encoding/binary"
	"fmt"

	"github.com/jokruger/kavun/errs"
)

type BuiltinFunction struct {
	Func     NativeFunc
	Name     string
	Arity    int8
	Variadic bool
}

type BuiltinModule struct {
	Name string
}

const (
	BuiltinFunctionModuleGlobal = uint8(0)
	BuiltinFunctionModuleAuto   = uint8(255)
	BuiltinFunctionMaxModules   = 256
	BuiltinFunctionMaxPerModule = 256
)

var builtinModules [BuiltinFunctionMaxModules]BuiltinModule
var builtinModuleSet [BuiltinFunctionMaxModules]bool

var builtinFunctions [BuiltinFunctionMaxModules][BuiltinFunctionMaxPerModule]BuiltinFunction
var builtinFunctionSet [BuiltinFunctionMaxModules][BuiltinFunctionMaxPerModule]bool

var autoBuiltinFuncID uint8

func init() {
	RegisterBuiltinModule(BuiltinFunctionModuleGlobal, "global")
	RegisterBuiltinModule(BuiltinFunctionModuleAuto, "auto")
}

func NewBuiltinFunction(name string, fn NativeFunc, arity int8, variadic bool) BuiltinFunction {
	return BuiltinFunction{
		Func:     fn,
		Name:     name,
		Arity:    arity,
		Variadic: variadic,
	}
}

func (f *BuiltinFunction) Set(fn NativeFunc, name string, arity int8, variadic bool) {
	f.Func = fn
	f.Name = name
	f.Arity = arity
	f.Variadic = variadic
}

// BuiltinFunctionID builds a packed static builtin function ID using
// lower 8 bits for in-module function index and next 8 bits for module ID.
func BuiltinFunctionID(moduleID uint8, functionID uint8) uint64 {
	return (uint64(moduleID) << 8) | uint64(functionID)
}

// BuiltinFunctionIDParts decodes packed builtin function ID into module/function indexes.
func BuiltinFunctionIDParts(id uint64) (moduleID uint8, functionID uint8) {
	return uint8((id >> 8) & 0xff), uint8(id & 0xff)
}

// RegisterBuiltinModule sets metadata for a builtin module slot.
func RegisterBuiltinModule(id uint8, name string) {
	builtinModules[id].Name = name
	builtinModuleSet[id] = true
}

// RegisterBuiltinFunction registers a static builtin function and returns its ID.
func RegisterBuiltinFunction(name string, fn NativeFunc, arity int8, variadic bool) uint64 {
	if autoBuiltinFuncID == 255 && builtinFunctionSet[BuiltinFunctionModuleAuto][255] {
		panic("builtin function auto registry full (max 256)")
	}
	id := BuiltinFunctionID(BuiltinFunctionModuleAuto, autoBuiltinFuncID)
	RegisterBuiltinFunctionAt(id, name, fn, arity, variadic)
	if autoBuiltinFuncID < 255 {
		autoBuiltinFuncID++
	}
	return id
}

// RegisterBuiltinFunctionAt registers a static builtin function at a specific ID.
func RegisterBuiltinFunctionAt(id uint64, name string, fn NativeFunc, arity int8, variadic bool) {
	modID, fnID := BuiltinFunctionIDParts(id)
	builtinFunctions[modID][fnID].Set(fn, name, arity, variadic)
	builtinFunctionSet[modID][fnID] = true
}

// GetBuiltinFunction returns a builtin function descriptor by static ID.
func GetBuiltinFunction(id uint64) (BuiltinFunction, bool) {
	modID, fnID := BuiltinFunctionIDParts(id)
	if !builtinFunctionSet[modID][fnID] {
		return BuiltinFunction{}, false
	}
	return builtinFunctions[modID][fnID], true
}

// ResolveBuiltinFunction resolves a static builtin function from a value.
func ResolveBuiltinFunction(v Value) (BuiltinFunction, bool) {
	if v.Type != VT_BUILTIN_FUNCTION {
		return BuiltinFunction{}, false
	}
	return GetBuiltinFunction(v.Data)
}

// BuiltinFunctionName returns the printable builtin function name for VT_BUILTIN_FUNCTION values.
func BuiltinFunctionName(v Value) (string, bool) {
	b, ok := ResolveBuiltinFunction(v)
	if !ok {
		return "", false
	}
	return b.Name, true
}

// BuiltinFunctionValue creates new boxed builtin function value.
func BuiltinFunctionValue(id uint64) Value {
	return Value{
		Type:      VT_BUILTIN_FUNCTION,
		Immutable: true,
		Data:      id,
	}
}

// NewBuiltinFunctionValue creates new (heap-allocated) builtin function value.
func NewBuiltinFunctionValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	id := RegisterBuiltinFunction(name, fn, arity, variadic)
	return BuiltinFunctionValue(id)
}

// NewBuiltinFunctionValueAt creates a static builtin function value with an explicit ID.
func NewBuiltinFunctionValueAt(id uint64, name string, fn NativeFunc, arity int8, variadic bool) Value {
	RegisterBuiltinFunctionAt(id, name, fn, arity, variadic)
	return BuiltinFunctionValue(id)
}

var TypeBuiltinFunction = ValueType{
	Name:         builtinFunctionTypeName,
	String:       func(a *Arena, v Value) string { return builtinFunctionTypeName(a, v) },
	EncodeBinary: builtinFunctionTypeEncodeBinary,
	DecodeBinary: builtinFunctionTypeDecodeBinary,
	IsTrue:       ConstHook(true),
	IsCallable:   ConstHook(true),
	IsVariadic:   builtinFunctionTypeIsVariadic,
	Arity:        builtinFunctionTypeArity,
	Call:         builtinFunctionTypeCall,
	MethodCall:   builtinFunctionTypeMethodCall,
}

func builtinFunctionTypeName(a *Arena, v Value) string {
	o, ok := ResolveBuiltinFunction(v)
	if !ok {
		return "<builtin-function:invalid>"
	}
	if o.Variadic {
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.Name, o.Arity)
}

func builtinFunctionTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, v.Data)
	if _, ok := GetBuiltinFunction(v.Data); !ok {
		return nil, fmt.Errorf("builtin function: invalid id %d", v.Data)
	}
	return out, nil
}

func builtinFunctionTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	if len(data) != 8 {
		return fmt.Errorf("builtin function: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.BigEndian.Uint64(data)
	if _, ok := GetBuiltinFunction(v.Data); !ok {
		return fmt.Errorf("builtin function: unknown id %d", v.Data)
	}
	return nil
}

func builtinFunctionTypeIsVariadic(a *Arena, v Value) bool {
	o, ok := ResolveBuiltinFunction(v)
	return ok && o.Variadic
}

func builtinFunctionTypeArity(a *Arena, v Value) int8 {
	o, ok := ResolveBuiltinFunction(v)
	if !ok {
		return 0
	}
	return o.Arity
}

func builtinFunctionTypeCall(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	o, ok := ResolveBuiltinFunction(v)
	if !ok {
		return Undefined, fmt.Errorf("builtin function: invalid id %d", v.Data)
	}
	if o.Func == nil {
		return Undefined, fmt.Errorf("builtin function: nil function for id %d", v.Data)
	}
	return o.Func(a, vm, args)
}

func builtinFunctionTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}
