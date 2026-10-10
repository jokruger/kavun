package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_dated_charges's members: one function per entry of TypeFinDatedCharges.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func finDatedChargesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinDatedChargesOf(v)
	return finTableString(v, t.String(), args)
}

// PURE by contract
func finDatedChargesBands(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinDatedChargesOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]Value, len(bs))
	for i, b := range bs {
		out[i] = NewRecordValue(map[string]Value{"from": DateValue(b.From), "amount": NewDecimalValue(b.Amount)}, false)
	}
	return NewArrayValue(out, false), nil
}

// PURE by contract
func finDatedChargesAt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinDatedChargesOf(v)
	// at(on): the amount in force (fin128 has no fallback form for a charge table)
	if err := finArgCount(name, args, 1); err != nil {
		return Undefined, err
	}
	on, err := finDateArg(name, "on", args[0])
	if err != nil {
		return Undefined, err
	}
	d, err := t.At(on)
	return finDecimal(name, d, err)
}
