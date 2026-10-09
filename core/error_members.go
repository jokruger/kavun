package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// error's members: one function per entry of TypeError.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func errorBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	b, _ := v.AsBool()
	return BoolValue(b), nil
}

// PURE by contract
func errorCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return errorTypeCopy(v, true)
}

// PURE by contract
func errorFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.Freeze()
}

// PURE by contract
func errorValue(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Error)(v.Ptr)
	return o.Payload, nil
}

// PURE by contract
func errorKind(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Error)(v.Ptr)
	return NewStringValue(o.Kind), nil

	// The three categories a script can ever hold. The fourth, system, is always fatal and therefore never
	// reaches a script, so it has no predicate here.
}

// PURE by contract
func errorIsRuntime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Error)(v.Ptr)
	return BoolValue(o.Category == errs.CategoryRuntime), nil
}

// PURE by contract
func errorIsUser(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Error)(v.Ptr)
	return BoolValue(o.Category == errs.CategoryUser), nil
}

// PURE by contract
func errorIsRequirement(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*Error)(v.Ptr)
	return BoolValue(o.Category == errs.CategoryRequirement), nil
}

// PURE by contract
func errorString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the payload's RENDER — the same path format() and f-strings use — so a
	// container payload answers its rendering instead of an empty string
	o := (*Error)(v.Ptr)
	s, err := o.Payload.Format(fspec.FormatSpec{})
	if err != nil {
		return Undefined, err
	}
	return convMember(id.String(), errorTypeName, args, true, NewStringValue(s))
}

// PURE by contract
func errorRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Error)(v.Ptr)
	s, err := o.Payload.Format(fspec.FormatSpec{})
	if err != nil {
		return Undefined, err
	}
	return convMember(id.String(), errorTypeName, args, true, NewRunesValue([]rune(s), false))
}
