package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

type BuiltinClosure struct {
	Func     NativeFunc
	Name     string
	Arity    int
	Variadic bool
}

func (f *BuiltinClosure) Set(fn NativeFunc, name string, arity int, variadic bool) {
	f.Func = fn
	f.Name = name
	f.Arity = arity
	f.Variadic = variadic
}

func NewBuiltinClosureValue(name string, fn NativeFunc, arity int, variadic bool) Value {
	o := &BuiltinClosure{}
	o.Set(fn, name, arity, variadic)
	return Value{Type: value.BuiltinClosure, Immutable: true, Ptr: unsafe.Pointer(o)}
}

var TypeBuiltinClosure = ValueTypeDescr{
	Name:       builtinClosureTypeName,                                    // PURE by contract
	String:     func(v Value) string { return builtinClosureTypeName(v) }, // PURE by contract
	IsTrue:     ConstHook(true),                                           // PURE by contract
	IsCallable: ConstHook(true),                                           // PURE by contract
	IsVariadic: builtinClosureTypeIsVariadic,                              // PURE by contract
	Arity:      builtinClosureTypeArity,                                   // PURE by contract
	Call:       builtinClosureTypeCall,                                    // CALLABLE-DEPENDENT by contract
	MethodCall: builtinClosureTypeMethodCall,                              // PURE by contract with higher-order rule caveat (see docs/purity.md)
}

func builtinClosureTypeName(v Value) string {
	o := (*BuiltinClosure)(v.Ptr)
	if o.Variadic {
		return fmt.Sprintf("<builtin-closure:%s/%d+>", o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-closure:%s/%d>", o.Name, o.Arity)
}

func builtinClosureTypeIsVariadic(v Value) bool {
	return (*BuiltinClosure)(v.Ptr).Variadic
}

func builtinClosureTypeArity(v Value) int {
	return (*BuiltinClosure)(v.Ptr).Arity
}

// CALLABLE-DEPENDENT: purity depends on the underlying builtin and the captured environment. Not folded by the
// optimizer unless a future analysis proves both are pure. See docs/purity.md.
func builtinClosureTypeCall(vm VM, v Value, args []Value) (Value, error) {
	return (*BuiltinClosure)(v.Ptr).Func(vm, args)
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
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
