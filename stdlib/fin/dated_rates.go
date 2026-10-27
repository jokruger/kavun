package fin

import (
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
)

// fin.dated_rates is an immutable date-banded rate table (a teaser rate, then a reversion rate): each band starts on
// a date and runs until the next; it is open at both ends, and makes no claim about dates before its first band.
// Built only by fin.dated_rates(text | bands).

const datedRatesTypeName = "fin.dated_rates"

// DatedRatesDefault is the type's default value, the zero-rate table "0001-01-01:0": one band from the first
// civil date, so no date falls before it, and every computation on it is 0.
var DatedRatesDefault = func() fin128.DatedRates {
	t, _ := fin128.ParseDatedRates("0001-01-01:0")
	return t
}()

func NewDatedRatesValue(t fin128.DatedRates) core.Value {
	return core.Value{Type: value.FinDatedRates, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func DatedRatesOf(v core.Value) fin128.DatedRates {
	return *(*fin128.DatedRates)(v.Ptr)
}

// NewDatedRatesFrom builds the table from its text form or an array of {from: date, rate} records.
func NewDatedRatesFrom(ctx string, src core.Value) (core.Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := tableText(ctx, src)
		if err != nil {
			return core.Undefined, err
		}
		t, err := fin128.ParseDatedRates(text)
		if err != nil {
			return core.Undefined, Error(ctx, err)
		}
		return NewDatedRatesValue(t), nil
	}
	var bands []fin128.DatedRateBand
	err := bandArray(ctx, src, func(at string, e core.Value) error {
		m, err := recordFields(ctx, at, e, "from", "rate")
		if err != nil {
			return err
		}
		from, err := dateArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		rate, err := core.DecimalOperandArg(ctx, at+".rate", m["rate"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.DatedRateBand{From: from, Rate: rate})
		return nil
	})
	if err != nil {
		return core.Undefined, err
	}
	t, err := fin128.NewDatedRates(bands)
	if err != nil {
		return core.Undefined, Error(ctx, err)
	}
	return NewDatedRatesValue(t), nil
}

func datedRatesEqual(a, b fin128.DatedRates) bool {
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

// An immutable date-banded rate table, such as a teaser rate that reverts.
//
// ## Overview
//
// `fin.dated_rates` is owned by the [`fin`](../stdlib/fin.md) module (see [Rate and charge
// tables](../stdlib/fin.md#rate-and-charge-tables) for what the four tables share). It is built from a one-line text
// form or from band records:
//
// | text form | band record |
// | --- | --- |
// | `"2026-01-01:0.04, 2026-07-01:0.02"` — each band from a date to the next | `{from: date, rate}` |
//
//kavun:type fin.dated_rates

// TypeDatedRates is the fin.dated_rates type descriptor.
var TypeDatedRates = core.ValueTypeDescr{
	Name:   core.ConstHook(datedRatesTypeName),                                                             // PURE by contract
	String: func(v core.Value) string { return tableSource(datedRatesTypeName, DatedRatesOf(v).String()) }, // PURE by contract
	Format: func(v core.Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return tableFormat(v, datedRatesTypeName, DatedRatesOf(v).String(), sp)
	},
	Interface:    func(v core.Value) any { return DatedRatesOf(v) },                                      // PURE by contract
	EncodeJSON:   func(v core.Value) ([]byte, error) { return tableJSON(DatedRatesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v core.Value) ([]byte, error) { return []byte(DatedRatesOf(v).String()), nil },    // PURE by contract
	DecodeBinary: func(v *core.Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewDatedRatesFrom(datedRatesTypeName, core.NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v core.Value) (bool, error) {
		return !datedRatesEqual(DatedRatesOf(v), DatedRatesDefault), nil
	}, // PURE by contract
	Len: core.ConstHook(int64(1)), // PURE by contract
	Equal: func(v core.Value, other core.Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinDatedRates {
			return datedRatesEqual(DatedRatesOf(v), DatedRatesOf(other))
		}
		if final {
			return false
		}
		return core.ValueTypes[other.Type].Equal(other, v, true)
	},
	AsString: func(v core.Value) (string, bool) { return DatedRatesOf(v).String(), true }, // PURE by contract

	Methods: datedRatesMethods,
}
