package core

import (
	"fmt"

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

var TypeBuiltinClosure = ValueTypeDescr{
	Pin:        func(a *Arena, v Value) { a.PinBuiltinClosureValue(v) },
	Retain:     func(a *Arena, v Value) { a.RetainBuiltinClosureValue(v) },
	Release:    func(a *Arena, v Value) { a.ReleaseBuiltinClosureValue(v) },
	Name:       builtinClosureTypeName,
	String:     func(a *Arena, v Value) string { return builtinClosureTypeName(a, v) },
	IsTrue:     ConstHook(true),
	IsCallable: ConstHook(true),
	IsVariadic: builtinClosureTypeIsVariadic,
	Arity:      builtinClosureTypeArity,
	Call:       builtinClosureTypeCall,
	MethodCall: builtinClosureTypeMethodCall,
}

func builtinClosureTypeName(a *Arena, v Value) string {
	o := (*BuiltinClosure)(v.Ptr)
	if o.Variadic {
		return fmt.Sprintf("<builtin-closure:%s/%d+>", o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-closure:%s/%d>", o.Name, o.Arity)
}

func builtinClosureTypeIsVariadic(a *Arena, v Value) bool {
	return (*BuiltinClosure)(v.Ptr).Variadic
}

func builtinClosureTypeArity(a *Arena, v Value) int8 {
	return (*BuiltinClosure)(v.Ptr).Arity
}

func builtinClosureTypeCall(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	return (*BuiltinClosure)(v.Ptr).Func(a, vm, args)
}

func builtinClosureTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
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
