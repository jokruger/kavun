package core

import (
	"encoding/binary"
	"fmt"
	"github.com/jokruger/kavun/core/member/members"

	"github.com/jokruger/kavun/core/value"
)

// NOTE: builtin functions are addressed by the index stored in Value.Data, not by Value.Ptr!

type BuiltinFunction struct {
	Func     NativeFunc
	Module   string
	Name     string
	Arity    int
	Variadic bool
	Pure     bool
}

// NewBuiltinFunction creates a new builtin function object.
// name is the name of the function.
// fn is the native function to be called.
// arity is the number of arguments the function takes.
// variadic is true if the function takes a variable number of arguments, in which case arity is the minimum number of arguments required.
// pure is true if the function does not have side effects, does not rely on external state, and always returns the same output for the same input.
func NewBuiltinFunction(name string, fn NativeFunc, arity int, variadic bool, pure bool) *BuiltinFunction {
	return &BuiltinFunction{
		Func:     fn,
		Name:     name,
		Arity:    arity,
		Variadic: variadic,
		Pure:     pure,
	}
}

// BuiltinFunctionValue creates new boxed builtin function value.
func BuiltinFunctionValue(id uint64) Value {
	return Value{
		Type:      value.BuiltinFunction,
		Immutable: true,
		Data:      id,
	}
}

var TypeBuiltinFunction = ValueTypeDescr{
	Name:         builtinFunctionTypeName,                                    // PURE by contract
	String:       func(v Value) string { return builtinFunctionTypeName(v) }, // PURE by contract
	Format:       callableFormat,                                             // PURE by contract
	EncodeBinary: builtinFunctionTypeEncodeBinary,                            // PURE by contract
	DecodeBinary: builtinFunctionTypeDecodeBinary,                            // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](true, nil),                         // PURE by contract
	IsCallable:   ConstHook(true),                                            // PURE by contract
	IsVariadic:   builtinFunctionTypeIsVariadic,                              // PURE by contract
	Arity:        builtinFunctionTypeArity,                                   // PURE by contract
	Call:         builtinFunctionTypeCall,                                    // CALLABLE-DEPENDENT by contract

	Methods: []Method{
		members.IsTrue: NewPureMethod(MemberIsTrue, 0, Fixed),
		members.Format: NewPureMethod(MemberFormat, 0, Variadic),
		members.Copy:   NewPureMethod(MemberSelf, 0, Fixed),
		members.Freeze: NewPureMethod(MemberSelf, 0, Fixed),
	},
}

// builtinFunctionAt answers the function registered at slot, or nil for a slot out of range or empty.
func builtinFunctionAt(slot uint64) *BuiltinFunction {
	if slot >= uint64(len(BuiltinFunctions)) {
		return nil
	}
	return BuiltinFunctions[slot]
}

func builtinFunctionTypeName(v Value) string {
	o := builtinFunctionAt(v.Data)
	if o == nil {
		return "<builtin-function>" // only a refused decode names a value that holds no function
	}
	if o.Variadic {
		if o.Module != "" {
			return fmt.Sprintf("<builtin-function:%s.%s/%d+>", o.Module, o.Name, o.Arity)
		}
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.Name, o.Arity)
	}
	if o.Module != "" {
		return fmt.Sprintf("<builtin-function:%s.%s/%d>", o.Module, o.Name, o.Arity)
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.Name, o.Arity)
}

func builtinFunctionTypeEncodeBinary(v Value) ([]byte, error) {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint64(out, v.Data)
	return out, nil
}

func builtinFunctionTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) != 8 {
		return fmt.Errorf("builtin function: expected 8 bytes, got %d", len(data))
	}
	slot := binary.LittleEndian.Uint64(data)
	if builtinFunctionAt(slot) == nil {
		return fmt.Errorf("builtin function: no function registered at slot %d", slot)
	}
	v.Data = slot
	return nil
}

func builtinFunctionTypeIsVariadic(v Value) bool {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Variadic
}

func builtinFunctionTypeArity(v Value) int {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Arity
}

// CALLABLE-DEPENDENT: purity is a property of the specific BuiltinFunction being called (see the Pure field). The
// optimizer folds Call only when the callable is a BuiltinFunction with Pure=true. See docs/purity.md.
func builtinFunctionTypeCall(vm VM, v Value, args []Value) (Value, error) {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Func(vm, args)
}
