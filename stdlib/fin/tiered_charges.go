package fin

import (
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.tiered_charges is an immutable amount-banded fee table: per band a rate plus a fixed amount, and the computed
// charge clamped to [min, max] (a max of 0 means no cap). It has no rate() and no accrue(): a fixed component is
// money, not a rate (fin128's own reasoning). Built only by fin.tiered_charges(text) or
// fin.tiered_charges(bands, min, max).

const tieredChargesTypeName = "fin.tiered_charges"

// TieredChargesDefault is the type's default value, the zero-rate table "0:0" with no bounds.
var TieredChargesDefault = func() fin128.TieredCharges {
	t, _ := fin128.ParseTieredCharges("0:0")
	return t
}()

func NewTieredChargesValue(t fin128.TieredCharges) core.Value {
	return core.Value{Type: value.FinTieredCharges, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func TieredChargesOf(v core.Value) fin128.TieredCharges {
	return *(*fin128.TieredCharges)(v.Ptr)
}

// NewTieredChargesFrom builds the table from its text form (bounds included: "…; min=25, max=500"), or from an
// array of {from, rate, fixed} records plus the bounds.
func NewTieredChargesFrom(ctx string, args []core.Value) (core.Value, error) {
	if len(args) == 1 && (args[0].Type == value.String || args[0].Type == value.Runes) {
		text, err := tableText(ctx, args[0])
		if err != nil {
			return core.Undefined, err
		}
		t, err := fin128.ParseTieredCharges(text)
		if err != nil {
			return core.Undefined, Error(ctx, err)
		}
		return NewTieredChargesValue(t), nil
	}
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError(ctx, "1 (text) or 3 (bands, min, max)", len(args))
	}
	var bands []fin128.ChargeBand
	err := bandArray(ctx, args[0], func(at string, e core.Value) error {
		m, err := recordFields(ctx, at, e, "from", "rate", "fixed")
		if err != nil {
			return err
		}
		var f [3]dec128.Dec128
		for i, k := range []string{"from", "rate", "fixed"} {
			if f[i], err = core.DecimalOperandArg(ctx, at+"."+k, m[k]); err != nil {
				return err
			}
		}
		bands = append(bands, fin128.ChargeBand{From: f[0], Rate: f[1], Fixed: f[2]})
		return nil
	})
	if err != nil {
		return core.Undefined, err
	}
	minimum, err := core.DecimalOperandArg(ctx, "min", args[1])
	if err != nil {
		return core.Undefined, err
	}
	maximum, err := core.DecimalOperandArg(ctx, "max", args[2])
	if err != nil {
		return core.Undefined, err
	}
	t, err := fin128.NewTieredCharges(bands, minimum, maximum)
	if err != nil {
		return core.Undefined, Error(ctx, err)
	}
	return NewTieredChargesValue(t), nil
}

func tieredChargesEqual(a, b fin128.TieredCharges) bool {
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

func chargeBandValue(b fin128.ChargeBand) core.Value {
	return core.NewRecordValue(map[string]core.Value{
		"from":  core.NewDecimalValue(b.From),
		"rate":  core.NewDecimalValue(b.Rate),
		"fixed": core.NewDecimalValue(b.Fixed),
	}, false)
}

// TypeTieredCharges is the fin.tiered_charges type descriptor.
var TypeTieredCharges = core.ValueTypeDescr{
	Name: core.ConstHook(tieredChargesTypeName), // PURE by contract
	String: func(v core.Value) string {
		return tableSource(tieredChargesTypeName, TieredChargesOf(v).String())
	}, // PURE by contract
	Format: func(v core.Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return tableFormat(v, tieredChargesTypeName, TieredChargesOf(v).String(), sp)
	},
	Interface:    func(v core.Value) any { return TieredChargesOf(v) },                                      // PURE by contract
	EncodeJSON:   func(v core.Value) ([]byte, error) { return tableJSON(TieredChargesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v core.Value) ([]byte, error) { return []byte(TieredChargesOf(v).String()), nil },    // PURE by contract
	DecodeBinary: func(v *core.Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewTieredChargesFrom(tieredChargesTypeName, []core.Value{core.NewStringValue(string(data))})
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v core.Value) (bool, error) { // PURE by contract
		return !tieredChargesEqual(TieredChargesOf(v), TieredChargesDefault), nil
	},
	Len: core.ConstHook(int64(1)), // PURE by contract
	Equal: func(v core.Value, other core.Value, final bool) bool { // PURE by contract: band by band and the bounds, numerically
		if other.Type == value.FinTieredCharges {
			return tieredChargesEqual(TieredChargesOf(v), TieredChargesOf(other))
		}
		if final {
			return false
		}
		return core.ValueTypes[other.Type].Equal(other, v, true)
	},
	AsString: func(v core.Value) (string, bool) { return TieredChargesOf(v).String(), true }, // PURE by contract

	Methods: []core.Method{
		members.IsTrue:      core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.String:      core.NewPureMethod(tieredChargesString, 0, core.Variadic),
		members.Format:      core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:        core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:      core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.At:          core.NewPureMethod(tieredChargesAt, 1, core.Fixed),
		members.Bands:       core.NewPureMethod(tieredChargesBands, 0, core.Fixed),
		members.Charge:      core.NewPureMethod(tieredChargesCharge, 4, core.Fixed),
		members.ChargeParts: core.NewPureMethod(tieredChargesChargeParts, 4, core.Fixed),
		members.Bounds:      core.NewPureMethod(tieredChargesBounds, 0, core.Fixed),
	},
}
