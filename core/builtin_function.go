package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/errs"
)

type BuiltinFunction struct {
	Func     NativeFunc
	Name     string
	Arity    int8
	Variadic bool
}

func (f *BuiltinFunction) Set(fn NativeFunc, name string, arity int8, variadic bool) {
	f.Func = fn
	f.Name = name
	f.Arity = arity
	f.Variadic = variadic
}

// BuiltinFunctionValue creates new boxed builtin function value.
func BuiltinFunctionValue(f *BuiltinFunction) Value {
	return Value{
		Type:      VT_BUILTIN_FUNCTION,
		Immutable: true,
		Ptr:       unsafe.Pointer(f),
	}
}

// NewBuiltinFunctionValue creates new (heap-allocated) builtin function value.
func NewBuiltinFunctionValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	t := &BuiltinFunction{}
	t.Set(fn, name, arity, variadic)
	return BuiltinFunctionValue(t)
}

var TypeBuiltinFunction = ValueType{
	Name:         builtinFunctionTypeName,
	String:       func(v Value) string { return builtinFunctionTypeName(v) },
	EncodeBinary: builtinFunctionTypeEncodeBinary,
	DecodeBinary: builtinFunctionTypeDecodeBinary,
	IsTrue:       ConstHook(true),
	IsCallable:   ConstHook(true),
	IsVariadic:   func(v Value) bool { return (*BuiltinFunction)(v.Ptr).Variadic },
	Arity:        func(v Value) int8 { return (*BuiltinFunction)(v.Ptr).Arity },
	Call:         func(v Value, vm VM, args []Value) (Value, error) { return (*BuiltinFunction)(v.Ptr).Func(vm, args) },
	MethodCall:   builtinFunctionTypeMethodCall,
}

func builtinFunctionTypeName(v Value) string {
	o := (*BuiltinFunction)(v.Ptr)
	if o.Variadic {
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.Name, o.Arity)
}

func builtinFunctionTypeEncodeBinary(v Value) ([]byte, error) {
	f := (*BuiltinFunction)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(f); err != nil {
		return nil, fmt.Errorf("builtin function: %w", err)
	}
	return buf.Bytes(), nil
}

func builtinFunctionTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var f BuiltinFunction
	if err := dec.Decode(&f); err != nil {
		return fmt.Errorf("builtin function: %w", err)
	}
	v.Ptr = unsafe.Pointer(&f)
	return nil
}

func builtinFunctionTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}
