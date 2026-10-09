package core

import (
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.tiered_charges is an immutable amount-banded fee table: per band a rate plus a fixed amount, and the computed
// charge clamped to [min, max] (a max of 0 means no cap). It has no rate() and no accrue(): a fixed component is
// money, not a rate (fin128's own reasoning). Built only by fin.tiered_charges(text) or
// fin.tiered_charges(bands, min, max).

const finTieredChargesTypeName = "fin.tiered_charges"

// FinTieredChargesDefault is the type's default value, the zero-rate table "0:0" with no bounds.
var FinTieredChargesDefault = func() fin128.TieredCharges {
	t, _ := fin128.ParseTieredCharges("0:0")
	return t
}()

func NewFinTieredChargesValue(t fin128.TieredCharges) Value {
	return Value{Type: value.FinTieredCharges, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func FinTieredChargesOf(v Value) fin128.TieredCharges {
	return *(*fin128.TieredCharges)(v.Ptr)
}

// NewFinTieredChargesFrom builds the table from its text form (bounds included: "…; min=25, max=500"), or from an
// array of {from, rate, fixed} records plus the bounds.
func NewFinTieredChargesFrom(ctx string, args []Value) (Value, error) {
	if len(args) == 1 && (args[0].Type == value.String || args[0].Type == value.Runes) {
		text, err := finTableText(ctx, args[0])
		if err != nil {
			return Undefined, err
		}
		t, err := fin128.ParseTieredCharges(text)
		if err != nil {
			return Undefined, FinError(ctx, err)
		}
		return NewFinTieredChargesValue(t), nil
	}
	if len(args) != 3 {
		return Undefined, errs.NewWrongNumArgumentsError(ctx, "1 (text) or 3 (bands, min, max)", len(args))
	}
	var bands []fin128.ChargeBand
	err := finBandArray(ctx, args[0], func(at string, e Value) error {
		m, err := finRecordFields(ctx, at, e, "from", "rate", "fixed")
		if err != nil {
			return err
		}
		var f [3]dec128.Dec128
		for i, k := range []string{"from", "rate", "fixed"} {
			if f[i], err = decimalOperandArg(ctx, at+"."+k, m[k]); err != nil {
				return err
			}
		}
		bands = append(bands, fin128.ChargeBand{From: f[0], Rate: f[1], Fixed: f[2]})
		return nil
	})
	if err != nil {
		return Undefined, err
	}
	minimum, err := decimalOperandArg(ctx, "min", args[1])
	if err != nil {
		return Undefined, err
	}
	maximum, err := decimalOperandArg(ctx, "max", args[2])
	if err != nil {
		return Undefined, err
	}
	t, err := fin128.NewTieredCharges(bands, minimum, maximum)
	if err != nil {
		return Undefined, FinError(ctx, err)
	}
	return NewFinTieredChargesValue(t), nil
}

func finTieredChargesEqual(a, b fin128.TieredCharges) bool {
	x, y := a.Bands(), b.Bands()
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if !decimalsEqual(x[i].From, y[i].From) || !decimalsEqual(x[i].Rate, y[i].Rate) || !decimalsEqual(x[i].Fixed, y[i].Fixed) {
			return false
		}
	}
	amin, amax := a.Bounds()
	bmin, bmax := b.Bounds()
	return decimalsEqual(amin, bmin) && decimalsEqual(amax, bmax)
}

func finChargeBandValue(b fin128.ChargeBand) Value {
	return NewRecordValue(map[string]Value{
		"from":  NewDecimalValue(b.From),
		"rate":  NewDecimalValue(b.Rate),
		"fixed": NewDecimalValue(b.Fixed),
	}, false)
}

// TypeFinTieredCharges is the fin.tiered_charges type descriptor.
var TypeFinTieredCharges = ValueTypeDescr{
	Name:   ConstHook(finTieredChargesTypeName),                                                                      // PURE by contract
	String: func(v Value) string { return finTableSource(finTieredChargesTypeName, FinTieredChargesOf(v).String()) }, // PURE by contract
	Format: func(v Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return finTableFormat(v, finTieredChargesTypeName, FinTieredChargesOf(v).String(), sp)
	},
	Interface:    func(v Value) any { return FinTieredChargesOf(v) },                                         // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return finTableJSON(FinTieredChargesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v Value) ([]byte, error) { return []byte(FinTieredChargesOf(v).String()), nil },       // PURE by contract
	DecodeBinary: func(v *Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewFinTieredChargesFrom(finTieredChargesTypeName, []Value{NewStringValue(string(data))})
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v Value) (bool, error) { // PURE by contract
		return !finTieredChargesEqual(FinTieredChargesOf(v), FinTieredChargesDefault), nil
	},
	Len: ConstHook(int64(1)), // PURE by contract
	Equal: func(v Value, other Value, final bool) bool { // PURE by contract: band by band and the bounds, numerically
		if other.Type == value.FinTieredCharges {
			return finTieredChargesEqual(FinTieredChargesOf(v), FinTieredChargesOf(other))
		}
		if final {
			return false
		}
		return ValueTypes[other.Type].Equal(other, v, true)
	},
	CallNamedMethod:   finTieredChargesCallNamedMethod,                                              // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AsString:          func(v Value) (string, bool) { return FinTieredChargesOf(v).String(), true }, // PURE by contract
	IsNamedMethodPure: func(string) bool { return true },                                            // All methods are expected to be pure.
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func finTieredChargesCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	t := FinTieredChargesOf(v)
	switch name {
	case "copy", "freeze":
		// identities on an immutable value
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil
	case "string":
		return finTableString(v, t.String(), args)
	case "format":
		return finTableFormatMember(v, t.String(), args)
	case "bands":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		bs := t.Bands()
		out := make([]Value, len(bs))
		for i, b := range bs {
			out[i] = finChargeBandValue(b)
		}
		return NewArrayValue(out, false), nil
	case "bounds":
		// {min, max}; a max of 0 means no cap
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		lo, hi := t.Bounds()
		return NewRecordValue(map[string]Value{"min": NewDecimalValue(lo), "max": NewDecimalValue(hi)}, false), nil
	case "at":
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
	case "charge":
		amount, rule, out, err := finChargeArgs(name, args)
		if err != nil {
			return Undefined, err
		}
		d, err := t.Charge(amount, rule, out)
		return finDecimal(name, d, err)
	case "charge_parts":
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
	return CallMemberByLookup(vm, v, name, args)
}
