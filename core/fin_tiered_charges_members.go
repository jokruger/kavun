package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_tiered_charges's members: one function per entry of TypeFinTieredCharges.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func finTieredChargesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinTieredChargesOf(v)
	return finTableString(v, t.String(), args)
}

// PURE by contract
func finTieredChargesBands(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinTieredChargesOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]Value, len(bs))
	for i, b := range bs {
		out[i] = finChargeBandValue(b)
	}
	return NewArrayValue(out, false), nil
}

// PURE by contract
func finTieredChargesBounds(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinTieredChargesOf(v)
	// {min, max}; a max of 0 means no cap
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	lo, hi := t.Bounds()
	return NewRecordValue(map[string]Value{"min": NewDecimalValue(lo), "max": NewDecimalValue(hi)}, false), nil
}

// PURE by contract
func finTieredChargesAt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredChargesOf(v)
	if err := finArgCount(name, args, 1); err != nil {
		return Undefined, err
	}
	amount, err := decimalOperandArg(name, "amount", args[0])
	if err != nil {
		return Undefined, err
	}
	b, err := t.At(amount)
	if err != nil {
		return Undefined, FinError(name, err)
	}
	return finChargeBandValue(b), nil
}

// PURE by contract
func finTieredChargesCharge(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredChargesOf(v)
	amount, rule, out, err := finChargeArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	d, err := t.Charge(amount, rule, out)
	return finDecimal(name, d, err)
}

// PURE by contract
func finTieredChargesChargeParts(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredChargesOf(v)
	amount, rule, out, err := finChargeArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	parts, err := t.ChargeParts(amount, rule, out)
	if err != nil {
		return Undefined, FinError(name, err)
	}
	return finTierPartsValue(parts), nil
}
