package core

import (
	"fmt"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
)

// BuiltinClosure is a native function built at run time (a Go closure handed to a script). Build its value with
// NewBuiltinClosureValue.
type BuiltinClosure struct {
	Func NativeFunc
	Name string
	Sig  Signature // what Func accepts
}

// Set fills the closure in place.
func (f *BuiltinClosure) Set(fn NativeFunc, name string, sig Signature) {
	f.Func = fn
	f.Name = name
	f.Sig = sig
}

// NewBuiltinClosureValue boxes fn as a builtin closure named name. sig is pre-built — MustSignature held in a
// package variable, ParseSignature, or SignatureOf when there is no template — so nothing is parsed on this path.
func NewBuiltinClosureValue(name string, sig Signature, fn NativeFunc) Value {
	o := &BuiltinClosure{}
	o.Set(fn, name, sig)
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

	Methods: []Method{
		members.IsTrue: NewPureMethod("()", builtinClosureIsTrue),
		members.Format: NewPureMethod("([spec])", builtinClosureFormat),
		members.Copy:   NewPureMethod("()", builtinClosureCopy),
		members.Freeze: NewPureMethod("()", builtinClosureFreeze),
	},
}

// builtinClosureIsTrue is a builtin closure's is_true(): the type's IsTrue hook, as a bool.
// PURE by contract
func builtinClosureIsTrue(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	t, err := ValueTypes[v.Type].IsTrue(v)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(t), nil
}

// builtinClosureFormat is a builtin closure's format([spec]).
// PURE by contract
func builtinClosureFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return FormatWithSpec(v, id, args)
}

// builtinClosureCopy is a builtin closure's copy(): the receiver — a builtin closure is immutable, so a copy of any depth is the
// value itself.
// PURE by contract
func builtinClosureCopy(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
}

// builtinClosureFreeze is a builtin closure's freeze(): the receiver — a builtin closure is immutable already.
// PURE by contract
func builtinClosureFreeze(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
}

func builtinClosureTypeName(v Value) string {
	o := (*BuiltinClosure)(v.Ptr)
	if o.Sig.IsVariadic() {
		return fmt.Sprintf("<builtin-closure:%s/%d+>", o.Name, o.Sig.Arity())
	}
	return fmt.Sprintf("<builtin-closure:%s/%d>", o.Name, o.Sig.Arity())
}

func builtinClosureTypeIsVariadic(v Value) bool {
	return (*BuiltinClosure)(v.Ptr).Sig.IsVariadic()
}

func builtinClosureTypeArity(v Value) int {
	return (*BuiltinClosure)(v.Ptr).Sig.Arity()
}

// CALLABLE-DEPENDENT: purity depends on the underlying builtin and the captured environment. Not folded by the
// optimizer unless a future analysis proves both are pure. See docs/purity.md.
func builtinClosureTypeCall(vm VM, v Value, args []Value) (Value, error) {
	return (*BuiltinClosure)(v.Ptr).Call(vm, args)
}

// Call runs the closure: the count check against Sig, then Func, then the translation of a body's ErrArgCount —
// the path every call of a builtin closure takes (the Call hook and the VM's fast track).
// CALLABLE-DEPENDENT by contract.
func (f *BuiltinClosure) Call(vm VM, args []Value) (Value, error) {
	if !f.Sig.admitsFloor(len(args)) {
		return closureCountRefused(f, len(args))
	}
	res, err := f.Func(vm, args)
	if err != nil {
		return closureResult(f, len(args), res, err)
	}
	return res, nil
}

// closureCountRefused and closureResult are Call's error paths, out of line so Call stays small enough to inline.
func closureCountRefused(f *BuiltinClosure, n int) (Value, error) {
	return Undefined, ArgCountError(f.Sig, f.Name, n)
}

func closureResult(f *BuiltinClosure, n int, res Value, err error) (Value, error) {
	return countResult(f.Sig, f.Name, n, res, err)
}
