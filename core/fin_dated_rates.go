package core

import (
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.dated_rates is an immutable date-banded rate table (a teaser rate, then a reversion rate): each band starts on
// a date and runs until the next; it is open at both ends, and makes no claim about dates before its first band.
// Built only by fin.dated_rates(text | bands).

const finDatedRatesTypeName = "fin.dated_rates"

// FinDatedRatesDefault is the type's default value, the zero-rate table "0001-01-01:0": one band from the first
// civil date, so no date falls before it, and every computation on it is 0.
var FinDatedRatesDefault = func() fin128.DatedRates {
	t, _ := fin128.ParseDatedRates("0001-01-01:0")
	return t
}()

func NewFinDatedRatesValue(t fin128.DatedRates) Value {
	return Value{Type: value.FinDatedRates, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func FinDatedRatesOf(v Value) fin128.DatedRates {
	return *(*fin128.DatedRates)(v.Ptr)
}

// NewFinDatedRatesFrom builds the table from its text form or an array of {from: date, rate} records.
func NewFinDatedRatesFrom(ctx string, src Value) (Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := finTableText(ctx, src)
		if err != nil {
			return Undefined, err
		}
		t, err := fin128.ParseDatedRates(text)
		if err != nil {
			return Undefined, FinError(ctx, err)
		}
		return NewFinDatedRatesValue(t), nil
	}
	var bands []fin128.DatedRateBand
	err := finBandArray(ctx, src, func(at string, e Value) error {
		m, err := finRecordFields(ctx, at, e, "from", "rate")
		if err != nil {
			return err
		}
		from, err := finDateArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		rate, err := decimalOperandArg(ctx, at+".rate", m["rate"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.DatedRateBand{From: from, Rate: rate})
		return nil
	})
	if err != nil {
		return Undefined, err
	}
	t, err := fin128.NewDatedRates(bands)
	if err != nil {
		return Undefined, FinError(ctx, err)
	}
	return NewFinDatedRatesValue(t), nil
}

func finDatedRatesEqual(a, b fin128.DatedRates) bool {
	x, y := a.Bands(), b.Bands()
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i].From != y[i].From || !decimalsEqual(x[i].Rate, y[i].Rate) {
			return false
		}
	}
	return true
}

// TypeFinDatedRates is the fin.dated_rates type descriptor.
var TypeFinDatedRates = ValueTypeDescr{
	Name:   ConstHook(finDatedRatesTypeName),                                                                   // PURE by contract
	String: func(v Value) string { return finTableSource(finDatedRatesTypeName, FinDatedRatesOf(v).String()) }, // PURE by contract
	Format: func(v Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return finTableFormat(v, finDatedRatesTypeName, FinDatedRatesOf(v).String(), sp)
	},
	Interface:    func(v Value) any { return FinDatedRatesOf(v) },                                         // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return finTableJSON(FinDatedRatesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v Value) ([]byte, error) { return []byte(FinDatedRatesOf(v).String()), nil },       // PURE by contract
	DecodeBinary: func(v *Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewFinDatedRatesFrom(finDatedRatesTypeName, NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v Value) (bool, error) { return !finDatedRatesEqual(FinDatedRatesOf(v), FinDatedRatesDefault), nil }, // PURE by contract
	Len:    ConstHook(int64(1)),                                                                                       // PURE by contract
	Equal: func(v Value, other Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinDatedRates {
			return finDatedRatesEqual(FinDatedRatesOf(v), FinDatedRatesOf(other))
		}
		if final {
			return false
		}
		return ValueTypes[other.Type].Equal(other, v, true)
	},
	MethodCall:   finDatedRatesMethodCall,                                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return FinDatedRatesOf(v).String(), true }, // PURE by contract
	IsMethodPure: func(string) bool { return true },                                         // All methods are expected to be pure.
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func finDatedRatesMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	t := FinDatedRatesOf(v)
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
			out[i] = NewRecordValue(map[string]Value{"from": DateValue(b.From), "rate": NewDecimalValue(b.Rate)}, false)
		}
		return NewArrayValue(out, false), nil
	case "at":
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
	case "apply":
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
	case "accrue", "accrue_parts":
		// (principal, start, end, convention, scale, mode): interest over [start, end), the rate changing at band
		// boundaries, each stretch measured by the convention
		if err := finArgCount(name, args, 6); err != nil {
			return Undefined, err
		}
		principal, err := decimalOperandArg(name, "principal", args[0])
		if err != nil {
			return Undefined, err
		}
		start, err := finDateArg(name, "start", args[1])
		if err != nil {
			return Undefined, err
		}
		end, err := finDateArg(name, "end", args[2])
		if err != nil {
			return Undefined, err
		}
		conv, err := FinConventionArg(name, "convention", args[3])
		if err != nil {
			return Undefined, err
		}
		out, err := finRoundingArgs(name, args, 4)
		if err != nil {
			return Undefined, err
		}
		if name == "accrue" {
			d, err := t.Accrue(principal, start, end, conv, out)
			return finDecimal(name, d, err)
		}
		parts, err := t.AccrueParts(principal, start, end, conv, out)
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
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}
