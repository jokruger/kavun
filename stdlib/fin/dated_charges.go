package fin

import (
	"github.com/jokruger/kavun/core/member/members"
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/fspec"
)

// fin.dated_charges is an immutable date-banded fee table: a standing charge that changes on given dates. It has no
// apply(): multiplying an amount by an amount has no meaning (fin128's own reasoning). Built only by
// fin.dated_charges(text | bands).

const datedChargesTypeName = "fin.dated_charges"

// DatedChargesDefault is the type's default value, the zero-amount table "0001-01-01:0".
var DatedChargesDefault = func() fin128.DatedCharges {
	t, _ := fin128.ParseDatedCharges("0001-01-01:0")
	return t
}()

func NewDatedChargesValue(t fin128.DatedCharges) core.Value {
	return core.Value{Type: value.FinDatedCharges, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func DatedChargesOf(v core.Value) fin128.DatedCharges {
	return *(*fin128.DatedCharges)(v.Ptr)
}

// NewDatedChargesFrom builds the table from its text form or an array of {from: date, amount} records.
func NewDatedChargesFrom(ctx string, src core.Value) (core.Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := tableText(ctx, src)
		if err != nil {
			return core.Undefined, err
		}
		t, err := fin128.ParseDatedCharges(text)
		if err != nil {
			return core.Undefined, Error(ctx, err)
		}
		return NewDatedChargesValue(t), nil
	}
	var bands []fin128.DatedAmountBand
	err := bandArray(ctx, src, func(at string, e core.Value) error {
		m, err := recordFields(ctx, at, e, "from", "amount")
		if err != nil {
			return err
		}
		from, err := dateArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		amount, err := core.DecimalOperandArg(ctx, at+".amount", m["amount"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.DatedAmountBand{From: from, Amount: amount})
		return nil
	})
	if err != nil {
		return core.Undefined, err
	}
	t, err := fin128.NewDatedCharges(bands)
	if err != nil {
		return core.Undefined, Error(ctx, err)
	}
	return NewDatedChargesValue(t), nil
}

func datedChargesEqual(a, b fin128.DatedCharges) bool {
	x, y := a.Bands(), b.Bands()
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i].From != y[i].From || !decimalsEqual(x[i].Amount, y[i].Amount) {
			return false
		}
	}
	return true
}

// TypeDatedCharges is the fin.dated_charges type descriptor.
var TypeDatedCharges = core.ValueTypeDescr{
	Name: core.ConstHook(datedChargesTypeName), // PURE by contract
	String: func(v core.Value) string {
		return tableSource(datedChargesTypeName, DatedChargesOf(v).String())
	}, // PURE by contract
	Format: func(v core.Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return tableFormat(v, datedChargesTypeName, DatedChargesOf(v).String(), sp)
	},
	Interface:    func(v core.Value) any { return DatedChargesOf(v) },                                      // PURE by contract
	EncodeJSON:   func(v core.Value) ([]byte, error) { return tableJSON(DatedChargesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v core.Value) ([]byte, error) { return []byte(DatedChargesOf(v).String()), nil },    // PURE by contract
	DecodeBinary: func(v *core.Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewDatedChargesFrom(datedChargesTypeName, core.NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v core.Value) (bool, error) { // PURE by contract
		return !datedChargesEqual(DatedChargesOf(v), DatedChargesDefault), nil
	},
	Len: core.ConstHook(int64(1)), // PURE by contract
	Equal: func(v core.Value, other core.Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinDatedCharges {
			return datedChargesEqual(DatedChargesOf(v), DatedChargesOf(other))
		}
		if final {
			return false
		}
		return core.ValueTypes[other.Type].Equal(other, v, true)
	},
	AsString: func(v core.Value) (string, bool) { return DatedChargesOf(v).String(), true }, // PURE by contract

	Methods: []core.Method{
		members.IsTrue: core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.String: core.NewPureMethod(datedChargesString, 0, core.Variadic),
		members.Format: core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:   core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze: core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.At:     core.NewPureMethod(datedChargesAt, 1, core.Fixed),
		members.Bands:  core.NewPureMethod(datedChargesBands, 0, core.Fixed),
	},
}
