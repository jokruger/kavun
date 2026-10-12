package fin

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// fin_tiered_rates's members: one function per entry of TypeTieredRates.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func tieredRatesString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := TieredRatesOf(v)
	return tableString(v, t.String(), args)
}

// PURE by contract
func tieredRatesBands(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := TieredRatesOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]core.Value, len(bs))
	for i, b := range bs {
		out[i] = rateBandValue(b)
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract
func tieredRatesAt(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredRatesOf(v)
	// at(amount) -> {from, rate}: the band the amount falls in, no arithmetic
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
	return rateBandValue(b), nil
}

// PURE by contract
func tieredRatesCharge(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredRatesOf(v)
	amount, rule, out, err := chargeArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.Charge(amount, rule, out)
	return decimal(name, d, err)
}

// PURE by contract
func tieredRatesRate(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredRatesOf(v)
	amount, rule, out, err := chargeArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.Rate(amount, rule, out)
	return decimal(name, d, err)
}

// PURE by contract
func tieredRatesChargeParts(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredRatesOf(v)
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

// PURE by contract
func tieredRatesAccrue(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := TieredRatesOf(v)
	// accrue(principal, f, rule, scale, mode): interest on principal at the table's rates over year fraction f
	if err := argCount(name, args, 5); err != nil {
		return core.Undefined, err
	}
	principal, err := core.DecimalOperandArg(name, "principal", args[0])
	if err != nil {
		return core.Undefined, err
	}
	if args[1].Type != value.FinYearFraction {
		return core.Undefined, errs.NewInvalidArgumentTypeError(name, "f", "fin.year_fraction", args[1].TypeName())
	}
	rule, err := ruleArg(name, "rule", args[2])
	if err != nil {
		return core.Undefined, err
	}
	out, err := roundingArgs(name, args, 3)
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.Accrue(principal, YearFractionOf(args[1]), rule, out)
	return decimal(name, d, err)
}
