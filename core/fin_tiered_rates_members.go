package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// fin_tiered_rates's members: one function per entry of TypeFinTieredRates.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func finTieredRatesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinTieredRatesOf(v)
	return finTableString(v, t.String(), args)
}

// PURE by contract
func finTieredRatesBands(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinTieredRatesOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]Value, len(bs))
	for i, b := range bs {
		out[i] = finRateBandValue(b)
	}
	return NewArrayValue(out, false), nil
}

// PURE by contract
func finTieredRatesAt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredRatesOf(v)
	// at(amount) -> {from, rate}: the band the amount falls in, no arithmetic
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
	return finRateBandValue(b), nil
}

// PURE by contract
func finTieredRatesCharge(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredRatesOf(v)
	amount, rule, out, err := finChargeArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	d, err := t.Charge(amount, rule, out)
	return finDecimal(name, d, err)
}

// PURE by contract
func finTieredRatesRate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredRatesOf(v)
	amount, rule, out, err := finChargeArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	d, err := t.Rate(amount, rule, out)
	return finDecimal(name, d, err)
}

// PURE by contract
func finTieredRatesChargeParts(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredRatesOf(v)
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

// PURE by contract
func finTieredRatesAccrue(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinTieredRatesOf(v)
	// accrue(principal, f, rule, scale, mode): interest on principal at the table's rates over year fraction f
	if err := finArgCount(name, args, 5); err != nil {
		return Undefined, err
	}
	principal, err := decimalOperandArg(name, "principal", args[0])
	if err != nil {
		return Undefined, err
	}
	if args[1].Type != value.FinYearFraction {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "f", "fin.year_fraction", args[1].TypeName())
	}
	rule, err := finRuleArg(name, "rule", args[2])
	if err != nil {
		return Undefined, err
	}
	out, err := finRoundingArgs(name, args, 3)
	if err != nil {
		return Undefined, err
	}
	d, err := t.Accrue(principal, FinYearFractionOf(args[1]), rule, out)
	return finDecimal(name, d, err)
}
