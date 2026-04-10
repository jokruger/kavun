package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"
)

type BuiltinFunction struct {
	Func     NativeFunc
	Name     string
	Arity    int
	Variadic bool
}

func (f *BuiltinFunction) Set(fn NativeFunc, name string, arity int, variadic bool) {
	f.Func = fn
	f.Name = name
	f.Arity = arity
	f.Variadic = variadic
}

func BuiltinFunctionValue(f *BuiltinFunction) Value {
	var v Value
	v.Ptr = unsafe.Pointer(f)
	v.Type = VT_BUILTIN_FUNCTION
	return v
}

func NewBuiltinFunctionValue(name string, fn NativeFunc, arity int, variadic bool) Value {
	t := &BuiltinFunction{}
	t.Set(fn, name, arity, variadic)
	return BuiltinFunctionValue(t)
}

func toBuiltinFunction(v Value) *BuiltinFunction {
	return (*BuiltinFunction)(v.Ptr)
}

func builtinFunctionTypeEqual(v Value, r Value) bool {
	return v == r
}

func builtinFunctionTypeArity(v Value) int {
	o := (*BuiltinFunction)(v.Ptr)
	return o.Arity
}

func builtinFunctionTypeIsVariadic(v Value) bool {
	o := (*BuiltinFunction)(v.Ptr)
	return o.Variadic
}

func builtinFunctionTypeName(v Value) string {
	o := (*BuiltinFunction)(v.Ptr)
	if builtinFunctionTypeIsVariadic(v) {
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.Name, builtinFunctionTypeArity(v))
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.Name, builtinFunctionTypeArity(v))
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

func builtinFunctionTypeString(v Value) string {
	return builtinFunctionTypeName(v)
}

func builtinFunctionTypeIsTrue(v Value) bool {
	return true
}

func builtinFunctionTypeIsCallable(v Value) bool {
	return true
}

func builtinFunctionTypeCall(v Value, vm VM, args []Value) (Value, error) {
	return (*BuiltinFunction)(v.Ptr).Func(vm, args)
}
