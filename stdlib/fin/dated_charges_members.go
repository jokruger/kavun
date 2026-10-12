package fin

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// fin_dated_charges's members: one function per entry of TypeDatedCharges.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func datedChargesString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := DatedChargesOf(v)
	return tableString(v, t.String(), args)
}

// PURE by contract
func datedChargesBands(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	t := DatedChargesOf(v)
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	bs := t.Bands()
	out := make([]core.Value, len(bs))
	for i, b := range bs {
		out[i] = core.NewRecordValue(map[string]core.Value{"from": core.DateValue(b.From), "amount": core.NewDecimalValue(b.Amount)}, false)
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract
func datedChargesAt(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	name := id.String()
	t := DatedChargesOf(v)
	// at(on): the amount in force (fin128 has no fallback form for a charge table)
	if err := argCount(name, args, 1); err != nil {
		return core.Undefined, err
	}
	on, err := dateArg(name, "on", args[0])
	if err != nil {
		return core.Undefined, err
	}
	d, err := t.At(on)
	return decimal(name, d, err)
}
