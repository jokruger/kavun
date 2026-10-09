package core

import (
	"unsafe"

	"github.com/jokruger/fin128"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// fin.dated_charges is an immutable date-banded fee table: a standing charge that changes on given dates. It has no
// apply(): multiplying an amount by an amount has no meaning (fin128's own reasoning). Built only by
// fin.dated_charges(text | bands).

const finDatedChargesTypeName = "fin.dated_charges"

// FinDatedChargesDefault is the type's default value, the zero-amount table "0001-01-01:0".
var FinDatedChargesDefault = func() fin128.DatedCharges {
	t, _ := fin128.ParseDatedCharges("0001-01-01:0")
	return t
}()

func NewFinDatedChargesValue(t fin128.DatedCharges) Value {
	return Value{Type: value.FinDatedCharges, Immutable: true, Ptr: unsafe.Pointer(&t)}
}

func FinDatedChargesOf(v Value) fin128.DatedCharges {
	return *(*fin128.DatedCharges)(v.Ptr)
}

// NewFinDatedChargesFrom builds the table from its text form or an array of {from: date, amount} records.
func NewFinDatedChargesFrom(ctx string, src Value) (Value, error) {
	switch src.Type {
	case value.String, value.Runes:
		text, err := finTableText(ctx, src)
		if err != nil {
			return Undefined, err
		}
		t, err := fin128.ParseDatedCharges(text)
		if err != nil {
			return Undefined, FinError(ctx, err)
		}
		return NewFinDatedChargesValue(t), nil
	}
	var bands []fin128.DatedAmountBand
	err := finBandArray(ctx, src, func(at string, e Value) error {
		m, err := finRecordFields(ctx, at, e, "from", "amount")
		if err != nil {
			return err
		}
		from, err := finDateArg(ctx, at+".from", m["from"])
		if err != nil {
			return err
		}
		amount, err := decimalOperandArg(ctx, at+".amount", m["amount"])
		if err != nil {
			return err
		}
		bands = append(bands, fin128.DatedAmountBand{From: from, Amount: amount})
		return nil
	})
	if err != nil {
		return Undefined, err
	}
	t, err := fin128.NewDatedCharges(bands)
	if err != nil {
		return Undefined, FinError(ctx, err)
	}
	return NewFinDatedChargesValue(t), nil
}

func finDatedChargesEqual(a, b fin128.DatedCharges) bool {
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

// TypeFinDatedCharges is the fin.dated_charges type descriptor.
var TypeFinDatedCharges = ValueTypeDescr{
	Name:   ConstHook(finDatedChargesTypeName),                                                                     // PURE by contract
	String: func(v Value) string { return finTableSource(finDatedChargesTypeName, FinDatedChargesOf(v).String()) }, // PURE by contract
	Format: func(v Value, sp fspec.FormatSpec) (string, error) { // PURE by contract
		return finTableFormat(v, finDatedChargesTypeName, FinDatedChargesOf(v).String(), sp)
	},
	Interface:    func(v Value) any { return FinDatedChargesOf(v) },                                         // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return finTableJSON(FinDatedChargesOf(v).String()), nil }, // PURE by contract: the text form, as a JSON string
	EncodeBinary: func(v Value) ([]byte, error) { return []byte(FinDatedChargesOf(v).String()), nil },       // PURE by contract
	DecodeBinary: func(v *Value, data []byte) error { // IMPURE by contract (mutates target): re-validated through the constructor
		r, err := NewFinDatedChargesFrom(finDatedChargesTypeName, NewStringValue(string(data)))
		if err != nil {
			return err
		}
		*v = r
		return nil
	},
	IsTrue: func(v Value) (bool, error) { // PURE by contract
		return !finDatedChargesEqual(FinDatedChargesOf(v), FinDatedChargesDefault), nil
	},
	Len: ConstHook(int64(1)), // PURE by contract
	Equal: func(v Value, other Value, final bool) bool { // PURE by contract: band by band, numerically
		if other.Type == value.FinDatedCharges {
			return finDatedChargesEqual(FinDatedChargesOf(v), FinDatedChargesOf(other))
		}
		if final {
			return false
		}
		return ValueTypes[other.Type].Equal(other, v, true)
	},
	CallNamedMethod:   finDatedChargesCallNamedMethod,                                              // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AsString:          func(v Value) (string, bool) { return FinDatedChargesOf(v).String(), true }, // PURE by contract
	IsNamedMethodPure: func(string) bool { return true },                                           // All methods are expected to be pure.
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func finDatedChargesCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	t := FinDatedChargesOf(v)
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
			out[i] = NewRecordValue(map[string]Value{"from": DateValue(b.From), "amount": NewDecimalValue(b.Amount)}, false)
		}
		return NewArrayValue(out, false), nil
	case "at":
		// at(on): the amount in force (fin128 has no fallback form for a charge table)
		if err := finArgCount(name, args, 1); err != nil {
			return Undefined, err
		}
		on, err := finDateArg(name, "on", args[0])
		if err != nil {
			return Undefined, err
		}
		d, err := t.At(on)
		return finDecimal(name, d, err)
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}
