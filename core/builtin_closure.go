package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

type BuiltinClosure struct {
	Func     NativeFunc
	Name     string
	Arity    int8
	Variadic bool
}

func (f *BuiltinClosure) Set(fn NativeFunc, name string, arity int8, variadic bool) {
	f.Func = fn
	f.Name = name
	f.Arity = arity
	f.Variadic = variadic
}

func (a *Arena) MustNewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) Value {
	v, err := a.NewBuiltinClosureValue(name, fn, arity, variadic)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewBuiltinClosureValue(name string, fn NativeFunc, arity int8, variadic bool) (Value, error) {
	if ref, p, ok := a.arena.New(value.BuiltinClosure); ok {
		(*BuiltinClosure)(p).Set(fn, name, arity, variadic)
		return Value{Type: value.BuiltinClosure, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError("builtin-closure")
}

var TypeBuiltinClosure = ValueTypeDescr{
	Name:       builtinClosureTypeName,
	String:     func(v Value) string { return builtinClosureTypeName(a, v) },
	IsTrue:     ConstHook(true),
	IsCallable: ConstHook(true),
	IsVariadic: builtinClosureTypeIsVariadic,
	Arity:      builtinClosureTypeArity,
	Call:       builtinClosureTypeCall,
	MethodCall: builtinClosureTypeMethodCall,
}

func builtinClosureTypeName(v Value) string {
	o := a.ResolveBuiltinClosureValue(v)
	if o.Variadic {
		return fmt.Sprintf("<builtin-closure:%s/%d+>", o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-closure:%s/%d>", o.Name, o.Arity)
}

func builtinClosureTypeIsVariadic(v Value) bool {
	return a.ResolveBuiltinClosureValue(v).Variadic
}

func builtinClosureTypeArity(v Value) int8 {
	return a.ResolveBuiltinClosureValue(v).Arity
}

func builtinClosureTypeCall(vm VM, v Value, args []Value) (Value, error) {
	return a.ResolveBuiltinClosureValue(v).Func(a, vm, args)
}

func builtinClosureTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
