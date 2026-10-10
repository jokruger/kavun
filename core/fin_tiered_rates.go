package core

import (
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
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
	AsString: func(v Value) (string, bool) { return FinTieredRatesOf(v).String(), true }, // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:      {Fn: memberIsTrue, Pure: true},
		members.String:      {Fn: finTieredRatesString, Pure: true},
		members.Format:      {Fn: memberFormat, Pure: true},
		members.Copy:        {Fn: memberSelf, Pure: true},
		members.Freeze:      {Fn: memberSelf, Pure: true},
		members.At:          {Fn: finTieredRatesAt, Pure: true},
		members.Bands:       {Fn: finTieredRatesBands, Pure: true},
		members.Rate:        {Fn: finTieredRatesRate, Pure: true},
		members.Charge:      {Fn: finTieredRatesCharge, Pure: true},
		members.ChargeParts: {Fn: finTieredRatesChargeParts, Pure: true},
		members.Accrue:      {Fn: finTieredRatesAccrue, Pure: true},
	},
}
