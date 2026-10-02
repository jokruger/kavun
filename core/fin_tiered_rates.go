package core

import (
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.tiered_rates is an immutable amount-banded rate table (a savings ladder): bands from 0, each with a rate,
// open at the top. A thin wrapper over fin128.TieredRates; built only by fin.tiered_rates(text | bands).

const finTieredRatesTypeName = "fin.tiered_rates"

// FinTieredRatesDefault is the type's default value, the zero-rate table "0:0": every computation on it is 0.
var FinTieredRatesDefault = func() fin128.TieredRates {
	t, _ := fin128.ParseTieredRates("0:0")
	return t
}()

func NewFinTieredRatesValue(t fin128.TieredRates) Value {
	return Value{Type: value.FinTieredRates, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func FinTieredRatesOf(v Value) fin128.TieredRates {
	return *(*fin128.TieredRates)(v.Ptr)
}

// NewFinTieredRatesFrom builds the table from its text form or an array of {from, rate} records.
func NewFinTieredRatesFrom(ctx string, src Value) (Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := finTableText(ctx, src)
		if err != nil {
			return Undefined, err
		}
		t, err := fin128.ParseTieredRates(text)
		if err != nil {
			return Undefined, FinError(ctx, err)
		}
		return NewFinTieredRatesValue(t), nil
	}
	var bands []fin128.RateBand
	err := finBandArray(ctx, src, func(at string, e Value) error {
		m, err := finRecordFields(ctx, at, e, "from", "rate")
		if err != nil {
			return err
		}
		from, err := decimalOperandArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		rate, err := decimalOperandArg(ctx, at+".rate", m["rate"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.RateBand{From: from, Rate: rate})
		return nil
	})
	if err != nil {
		return Undefined, err
	}
	t, err := fin128.NewTieredRates(bands)
	if err != nil {
		return Undefined, FinError(ctx, err)
	}
	return NewFinTieredRatesValue(t), nil
}

func finTieredRatesEqual(a, b fin128.TieredRates) bool {
	x, y := a.Bands(), b.Bands()
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if !decimalsEqual(x[i].From, y[i].From) || !decimalsEqual(x[i].Rate, y[i].Rate) {
			return false
		}
	}
	return true
}

func finRateBandValue(b fin128.RateBand) Value {
	return NewRecordValue(map[string]Value{"from": NewDecimalValue(b.From), "rate": NewDecimalValue(b.Rate)}, false)
}

// TypeFinTieredRates is the fin.tiered_rates type descriptor.
var TypeFinTieredRates = ValueTypeDescr{
	Name:   ConstHook(finTieredRatesTypeName),                                                                    // PURE by contract
	String: func(v Value) string { return finTableSource(finTieredRatesTypeName, FinTieredRatesOf(v).String()) }, // PURE by contract
	Format: func(v Value, sp fspec.FormatSpec) (string, error) {
		return finTableFormat(v, finTieredRatesTypeName, FinTieredRatesOf(v).String(), sp)
	}, // PURE by contract
	Interface:    func(v Value) any { return FinTieredRatesOf(v) },                                         // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return finTableJSON(FinTieredRatesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v Value) ([]byte, error) { return []byte(FinTieredRatesOf(v).String()), nil },       // PURE by contract
	DecodeBinary: func(v *Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewFinTieredRatesFrom(finTieredRatesTypeName, NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v Value) (bool, error) {
		return !finTieredRatesEqual(FinTieredRatesOf(v), FinTieredRatesDefault), nil
	}, // PURE by contract
	Len: ConstHook(int64(1)), // PURE by contract
	Equal: func(v Value, other Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinTieredRates {
			return finTieredRatesEqual(FinTieredRatesOf(v), FinTieredRatesOf(other))
		}
		if final {
			return false
		}
		return ValueTypes[other.Type].Equal(other, v, true)
	},
	MethodCall:   finTieredRatesMethodCall,                                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return FinTieredRatesOf(v).String(), true }, // PURE by contract
	IsMethodPure: func(string) bool { return true },                                          // All methods are expected to be pure.
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func finTieredRatesMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	t := FinTieredRatesOf(v)
	if r, ok, err := finTableCommonMember(v, name, t.String(), args); ok {
		return r, err
	}
	switch name {
	case "bands":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		bs := t.Bands()
		out := make([]Value, len(bs))
		for i, b := range bs {
			out[i] = finRateBandValue(b)
		}
		return NewArrayValue(out, false), nil
	case "at":
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
	case "charge", "rate", "charge_parts":
		// (amount, rule, scale, mode)
		if err := finArgCount(name, args, 4); err != nil {
			return Undefined, err
		}
		amount, err := decimalOperandArg(name, "amount", args[0])
		if err != nil {
			return Undefined, err
		}
		rule, err := finRuleArg(name, "rule", args[1])
		if err != nil {
			return Undefined, err
		}
		out, err := finRoundingArgs(name, args, 2)
		if err != nil {
			return Undefined, err
		}
		switch name {
		case "charge":
			d, err := t.Charge(amount, rule, out)
			return finDecimal(name, d, err)
		case "rate":
			d, err := t.Rate(amount, rule, out)
			return finDecimal(name, d, err)
		}
		parts, err := t.ChargeParts(amount, rule, out)
		if err != nil {
			return Undefined, FinError(name, err)
		}
		return finTierPartsValue(parts), nil
	case "accrue":
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
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}
