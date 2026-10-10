package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_dated_rates's members: one function per entry of TypeFinDatedRates.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func finDatedRatesString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinDatedRatesOf(v)
	return finTableString(v, t.String(), args)
}

// PURE by contract
func finDatedRatesBands(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t := FinDatedRatesOf(v)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]Value, len(bs))
	for i, b := range bs {
		out[i] = NewRecordValue(map[string]Value{"from": DateValue(b.From), "rate": NewDecimalValue(b.Rate)}, false)
	}
	return NewArrayValue(out, false), nil
}

// PURE by contract
func finDatedRatesAt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinDatedRatesOf(v)
	// at(on[, fallback]): the rate in force; the fallback answers ONLY a date before the first band
	if err := finArgCount(name, args, 1, 2); err != nil {
		return Undefined, err
	}
	on, err := finDateArg(name, "on", args[0])
	if err != nil {
		return Undefined, err
	}
	if len(args) == 1 {
		d, err := t.At(on)
		return finDecimal(name, d, err)
	}
	fallback, err := decimalOperandArg(name, "fallback", args[1])
	if err != nil {
		return Undefined, err
	}
	d, err := t.AtOr(on, fallback)
	return finDecimal(name, d, err)
}

// PURE by contract
func finDatedRatesApply(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinDatedRatesOf(v)
	// apply(amount, on, [fallback,] scale, mode): amount × the rate in force, rounded once
	if err := finArgCount(name, args, 4, 5); err != nil {
		return Undefined, err
	}
	amount, err := decimalOperandArg(name, "amount", args[0])
	if err != nil {
		return Undefined, err
	}
	on, err := finDateArg(name, "on", args[1])
	if err != nil {
		return Undefined, err
	}
	out, err := finRoundingArgs(name, args, len(args)-2)
	if err != nil {
		return Undefined, err
	}
	if len(args) == 4 {
		d, err := t.Apply(amount, on, out)
		return finDecimal(name, d, err)
	}
	fallback, err := decimalOperandArg(name, "fallback", args[2])
	if err != nil {
		return Undefined, err
	}
	d, err := t.ApplyOr(amount, on, fallback, out)
	return finDecimal(name, d, err)
}

// PURE by contract
func finDatedRatesAccrue(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinDatedRatesOf(v)
	a, err := finAccrualArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	d, err := t.Accrue(a.principal, a.start, a.end, a.conv, a.out)
	return finDecimal(name, d, err)
}

// PURE by contract
func finDatedRatesAccrueParts(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	t := FinDatedRatesOf(v)
	a, err := finAccrualArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	parts, err := t.AccrueParts(a.principal, a.start, a.end, a.conv, a.out)
	if err != nil {
		return Undefined, FinError(name, err)
	}
	res := make([]Value, len(parts))
	for i, p := range parts {
		res[i] = NewRecordValue(map[string]Value{
			"start":  DateValue(p.Start),
			"end":    DateValue(p.End),
			"rate":   NewDecimalValue(p.Rate),
			"amount": NewDecimalValue(p.Amount),
		}, false)
	}
	return NewArrayValue(res, false), nil
}
