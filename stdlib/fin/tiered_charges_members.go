package fin

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_tiered_charges's members: one function per entry of TypeTieredCharges.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func tieredChargesString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := TieredChargesOf(v)
	return tableString(v, t.String(), args)
}

// PURE by contract
func tieredChargesBands(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := TieredChargesOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]core.Value, len(bs))
	for i, b := range bs {
		out[i] = chargeBandValue(b)
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract
func tieredChargesBounds(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := TieredChargesOf(v)
	// {min, max}; a max of 0 means no cap
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	lo, hi := t.Bounds()
	return core.NewRecordValue(map[string]core.Value{"min": core.NewDecimalValue(lo), "max": core.NewDecimalValue(hi)}, false), nil
}

// PURE by contract
func tieredChargesAt(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredChargesOf(v)
	if err := argCount(name, args, 1); err != nil {
		return core.Undefined, err
	}
	amount, err := core.DecimalOperandArg(name, "amount", args[0])
	if err != nil {
		return core.Undefined, err
	}
	b, err := t.At(amount)
	if err != nil {
		return core.Undefined, Error(name, err)
	}
	return chargeBandValue(b), nil
}

// PURE by contract
func tieredChargesCharge(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredChargesOf(v)
	amount, rule, out, err := chargeArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.Charge(amount, rule, out)
	return decimal(name, d, err)
}

// PURE by contract
func tieredChargesChargeParts(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredChargesOf(v)
	amount, rule, out, err := chargeArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	parts, err := t.ChargeParts(amount, rule, out)
	if err != nil {
		return core.Undefined, Error(name, err)
	}
	return tierPartsValue(parts), nil
}
