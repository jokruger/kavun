package core

import (
	"fmt"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"

	"github.com/jokruger/kavun/core/value"
)

// NOTE: builtin functions are addressed by the index stored in Value.Data, not by Value.Ptr!

// BuiltinFunction is a native function a script calls by name: a free builtin (vm.BuiltinFunctions) or a module
// function. Build it with NewBuiltinFunction.
type BuiltinFunction struct {
	Func   NativeFunc
	Module string    // the owning module's name; "" for a free builtin
	Name   string    // the name a script calls it by
	Sig    Signature // what Func accepts
	Pure   bool      // the optimizer may fold a call (docs/purity.md)
}

// NewBuiltinFunction creates a builtin function named name. template is its signature template ("(x[, default])");
// a malformed one panics, like regexp.MustCompile — it is a programming error caught at init. pure is true when
// the function has no side effects, reads no external state, and always answers the same output for the same input.
func NewBuiltinFunction(name string, template string, fn NativeFunc, pure bool) *BuiltinFunction {
	return &BuiltinFunction{Func: fn, Name: name, Sig: MustSignature(template), Pure: pure}
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
	EncodeBinary: codeEncodeBinary,                                           // PURE by contract (refuses: code, not data)
	DecodeBinary: codeDecodeBinary,                                           // IMPURE by contract (refuses: code, not data)
	IsTrue:       Const2Hook[bool, error](true, nil),                         // PURE by contract
	IsCallable:   ConstHook(true),                                            // PURE by contract
	IsVariadic:   builtinFunctionTypeIsVariadic,                              // PURE by contract
	Arity:        builtinFunctionTypeArity,                                   // PURE by contract
	Call:         builtinFunctionTypeCall,                                    // CALLABLE-DEPENDENT by contract

	Methods: []Method{
		members.IsTrue: NewPureMethod("()", builtinFunctionIsTrue),
		members.Format: NewPureMethod("([spec])", builtinFunctionFormat),
		members.Copy:   NewPureMethod("()", builtinFunctionCopy),
		members.Freeze: NewPureMethod("()", builtinFunctionFreeze),
	},
}

// builtinFunctionIsTrue is a builtin function's is_true(): the type's IsTrue hook, as a bool.
// PURE by contract
func builtinFunctionIsTrue(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	t, err := ValueTypes[v.Type].IsTrue(v)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(t), nil
}

// builtinFunctionFormat is a builtin function's format([spec]).
// PURE by contract
func builtinFunctionFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return FormatWithSpec(v, id, args)
}

// builtinFunctionCopy is a builtin function's copy(): the receiver — a builtin function is immutable, so a copy of any depth is the
// value itself.
// PURE by contract
func builtinFunctionCopy(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
}

// builtinFunctionFreeze is a builtin function's freeze(): the receiver — a builtin function is immutable already.
// PURE by contract
func builtinFunctionFreeze(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
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
	if o.Sig.IsVariadic() {
		if o.Module != "" {
			return fmt.Sprintf("<builtin-function:%s.%s/%d+>", o.Module, o.Name, o.Sig.Arity())
		}
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.Name, o.Sig.Arity())
	}
	if o.Module != "" {
		return fmt.Sprintf("<builtin-function:%s.%s/%d>", o.Module, o.Name, o.Sig.Arity())
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.Name, o.Sig.Arity())
}

func builtinFunctionTypeIsVariadic(v Value) bool {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Sig.IsVariadic()
}

func builtinFunctionTypeArity(v Value) int {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Sig.Arity()
}

// CALLABLE-DEPENDENT: purity is a property of the specific BuiltinFunction being called (see the Pure field). The
// optimizer folds Call only when the callable is a BuiltinFunction with Pure=true. See docs/purity.md.
func builtinFunctionTypeCall(vm VM, v Value, args []Value) (Value, error) {
	// the slot holds a function: every value is built from a registered slot, and decode refuses any other
	return BuiltinFunctions[v.Data].Call(vm, args)
}

// Call runs the function: the count check against Sig, then Func, then the translation of a body's ErrArgCount —
// the path every call of a builtin or module function takes (the Call hook and the VM's fast track).
// CALLABLE-DEPENDENT by contract: purity is the function's (Pure).
func (f *BuiltinFunction) Call(vm VM, args []Value) (Value, error) {
	if !f.Sig.admitsFloor(len(args)) {
		return builtinCountRefused(f, len(args))
	}
	res, err := f.Func(vm, args)
	if err != nil {
		return builtinResult(f, len(args), res, err)
	}
	return res, nil
}

// builtinCountRefused and builtinResult are Call's error paths, out of line so Call stays small enough to inline.
func builtinCountRefused(f *BuiltinFunction, n int) (Value, error) {
	return Undefined, ArgCountError(f.Sig, f.QualifiedName(), n)
}

func builtinResult(f *BuiltinFunction, n int, res Value, err error) (Value, error) {
	return countResult(f.Sig, f.QualifiedName(), n, res, err)
}

// QualifiedName answers the name a script calls the function by: "min" for a free builtin, "math.abs" for a module
// function.
func (f *BuiltinFunction) QualifiedName() string {
	if f.Module != "" {
		return f.Module + "." + f.Name
	}
	return f.Name
}
