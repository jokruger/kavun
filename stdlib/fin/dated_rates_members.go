package fin

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_dated_rates's members: one function per entry of TypeDatedRates.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func datedRatesString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := DatedRatesOf(v)
	return tableString(v, t.String(), args)
}

// PURE by contract
func datedRatesBands(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := DatedRatesOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]core.Value, len(bs))
	for i, b := range bs {
		out[i] = core.NewRecordValue(map[string]core.Value{"from": core.DateValue(b.From), "rate": core.NewDecimalValue(b.Rate)}, false)
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract
func datedRatesAt(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := DatedRatesOf(v)
	// at(on[, fallback]): the rate in force; the fallback answers ONLY a date before the first band
	if err := argCount(name, args, 1, 2); err != nil {
		return core.Undefined, err
	}
	on, err := dateArg(name, "on", args[0])
	if err != nil {
		return core.Undefined, err
	}
	if len(args) == 1 {
		d, err := t.At(on)
		return decimal(name, d, err)
	}
	fallback, err := core.DecimalOperandArg(name, "fallback", args[1])
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.AtOr(on, fallback)
	return decimal(name, d, err)
}

// PURE by contract
func datedRatesApply(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := DatedRatesOf(v)
	// apply(amount, on, [fallback,] scale, mode): amount × the rate in force, rounded once
	if err := argCount(name, args, 4, 5); err != nil {
		return core.Undefined, err
	}
	amount, err := core.DecimalOperandArg(name, "amount", args[0])
	if err != nil {
		return core.Undefined, err
	}
	on, err := dateArg(name, "on", args[1])
	if err != nil {
		return core.Undefined, err
	}
	out, err := roundingArgs(name, args, len(args)-2)
	if err != nil {
		return core.Undefined, err
	}
	if len(args) == 4 {
		d, err := t.Apply(amount, on, out)
		return decimal(name, d, err)
	}
	fallback, err := core.DecimalOperandArg(name, "fallback", args[2])
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.ApplyOr(amount, on, fallback, out)
	return decimal(name, d, err)
}

// PURE by contract
func datedRatesAccrue(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := DatedRatesOf(v)
	a, err := accrualArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.Accrue(a.principal, a.start, a.end, a.conv, a.out)
	return decimal(name, d, err)
}

// PURE by contract
func datedRatesAccrueParts(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := DatedRatesOf(v)
	a, err := accrualArgs(name, args)
	if err != nil {
		return core.Undefined, err
	}
	parts, err := t.AccrueParts(a.principal, a.start, a.end, a.conv, a.out)
	if err != nil {
		return core.Undefined, Error(name, err)
	}
	res := make([]core.Value, len(parts))
	for i, p := range parts {
		res[i] = core.NewRecordValue(map[string]core.Value{
			"start":  core.DateValue(p.Start),
			"end":    core.DateValue(p.End),
			"rate":   core.NewDecimalValue(p.Rate),
			"amount": core.NewDecimalValue(p.Amount),
		}, false)
	}
	return core.NewArrayValue(res, false), nil
}
