package core

import (
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
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
	AsString: func(v Value) (string, bool) { return FinDatedRatesOf(v).String(), true }, // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:      {Fn: memberIsTrue, Pure: true},
		members.String:      {Fn: finDatedRatesString, Pure: true},
		members.Format:      {Fn: memberFormat, Pure: true},
		members.Copy:        {Fn: memberSelf, Pure: true},
		members.Freeze:      {Fn: memberSelf, Pure: true},
		members.At:          {Fn: finDatedRatesAt, Pure: true},
		members.Bands:       {Fn: finDatedRatesBands, Pure: true},
		members.Accrue:      {Fn: finDatedRatesAccrue, Pure: true},
		members.AccrueParts: {Fn: finDatedRatesAccrueParts, Pure: true},
		members.Apply:       {Fn: finDatedRatesApply, Pure: true},
	},
}
