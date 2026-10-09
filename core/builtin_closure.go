package core

import (
	"fmt"
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
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
	Format:     callableFormat,                                            // PURE by contract
	IsTrue:     Const2Hook[bool, error](true, nil),                        // PURE by contract
	IsCallable: ConstHook(true),                                           // PURE by contract
	IsVariadic: builtinClosureTypeIsVariadic,                              // PURE by contract
	Arity:      builtinClosureTypeArity,                                   // PURE by contract
	Call:       builtinClosureTypeCall,                                    // CALLABLE-DEPENDENT by contract

	Methods: []MethodEntry{
		members.IsTrue: {Fn: memberIsTrue, Pure: true},
		members.Format: {Fn: memberFormat, Pure: true},
		members.Copy:   {Fn: memberSelf, Pure: true},
		members.Freeze: {Fn: memberSelf, Pure: true},
	},
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
