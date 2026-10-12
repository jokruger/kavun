package fin

import (
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
)

// fin.tiered_rates is an immutable amount-banded rate table (a savings ladder): bands from 0, each with a rate,
// open at the top. A thin wrapper over fin128.TieredRates; built only by fin.tiered_rates(text | bands).

const tieredRatesTypeName = "fin.tiered_rates"

// TieredRatesDefault is the type's default value, the zero-rate table "0:0": every computation on it is 0.
var TieredRatesDefault = func() fin128.TieredRates {
	t, _ := fin128.ParseTieredRates("0:0")
	return t
}()

func NewTieredRatesValue(t fin128.TieredRates) core.Value {
	return core.Value{Type: value.FinTieredRates, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func TieredRatesOf(v core.Value) fin128.TieredRates {
	return *(*fin128.TieredRates)(v.Ptr)
}

// NewTieredRatesFrom builds the table from its text form or an array of {from, rate} records.
func NewTieredRatesFrom(ctx string, src core.Value) (core.Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := tableText(ctx, src)
		if err != nil {
			return core.Undefined, err
		}
		t, err := fin128.ParseTieredRates(text)
		if err != nil {
			return core.Undefined, Error(ctx, err)
		}
		return NewTieredRatesValue(t), nil
	}
	var bands []fin128.RateBand
	err := bandArray(ctx, src, func(at string, e core.Value) error {
		m, err := recordFields(ctx, at, e, "from", "rate")
		if err != nil {
			return err
		}
		from, err := core.DecimalOperandArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		rate, err := core.DecimalOperandArg(ctx, at+".rate", m["rate"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.RateBand{From: from, Rate: rate})
		return nil
	})
	if err != nil {
		return core.Undefined, err
	}
	t, err := fin128.NewTieredRates(bands)
	if err != nil {
		return core.Undefined, Error(ctx, err)
	}
	return NewTieredRatesValue(t), nil
}

func tieredRatesEqual(a, b fin128.TieredRates) bool {
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

func rateBandValue(b fin128.RateBand) core.Value {
	return core.NewRecordValue(map[string]core.Value{"from": core.NewDecimalValue(b.From), "rate": core.NewDecimalValue(b.Rate)}, false)
}

// TypeTieredRates is the fin.tiered_rates type descriptor.
var TypeTieredRates = core.ValueTypeDescr{
	Name:   core.ConstHook(tieredRatesTypeName),                                                              // PURE by contract
	String: func(v core.Value) string { return tableSource(tieredRatesTypeName, TieredRatesOf(v).String()) }, // PURE by contract
	Format: func(v core.Value, sp fspec.FormatSpec) (string, error) {
		return tableFormat(v, tieredRatesTypeName, TieredRatesOf(v).String(), sp)
	}, // PURE by contract
	Interface:    func(v core.Value) any { return TieredRatesOf(v) },                                      // PURE by contract
	EncodeJSON:   func(v core.Value) ([]byte, error) { return tableJSON(TieredRatesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v core.Value) ([]byte, error) { return []byte(TieredRatesOf(v).String()), nil },    // PURE by contract
	DecodeBinary: func(v *core.Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewTieredRatesFrom(tieredRatesTypeName, core.NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v core.Value) (bool, error) {
		return !tieredRatesEqual(TieredRatesOf(v), TieredRatesDefault), nil
	}, // PURE by contract
	Len: core.ConstHook(int64(1)), // PURE by contract
	Equal: func(v core.Value, other core.Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinTieredRates {
			return tieredRatesEqual(TieredRatesOf(v), TieredRatesOf(other))
		}
		if final {
			return false
		}
		return core.ValueTypes[other.Type].Equal(other, v, true)
	},
	AsString: func(v core.Value) (string, bool) { return TieredRatesOf(v).String(), true }, // PURE by contract

	Methods: []core.Method{
		members.IsTrue:      core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.String:      core.NewPureMethod(tieredRatesString, 0, core.Variadic),
		members.Format:      core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:        core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:      core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.At:          core.NewPureMethod(tieredRatesAt, 1, core.Fixed),
		members.Bands:       core.NewPureMethod(tieredRatesBands, 0, core.Fixed),
		members.Rate:        core.NewPureMethod(tieredRatesRate, 4, core.Fixed),
		members.Charge:      core.NewPureMethod(tieredRatesCharge, 4, core.Fixed),
		members.ChargeParts: core.NewPureMethod(tieredRatesChargeParts, 4, core.Fixed),
		members.Accrue:      core.NewPureMethod(tieredRatesAccrue, 5, core.Fixed),
	},
}
