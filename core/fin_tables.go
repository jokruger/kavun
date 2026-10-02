package core

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/fin128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/fin128/daycount"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// Shared machinery of the fin module's types (fin.year_fraction and the four rate/charge tables) and functions:
// the single fin128 → Kavun error translation, the argument readers the table members share with the module, and
// the band ↔ record codec the table constructors and bands() use.

// FinError is the one translation point from a fin128 or dec128 error to a Kavun error, for the fin module's
// functions and its types' members alike. ctx names the operation ("fin.payment", "charge").
func FinError(ctx string, err error) error {
	msg := strings.TrimPrefix(err.Error(), "fin128: ")
	switch {
	case errors.Is(err, state.DivisionByZero.Error()):
		return errs.NewDivisionByZeroError()
	case errors.Is(err, fin128.ErrSyntax):
		return errs.NewConversionError("string", ctx, msg)
	case errors.Is(err, fin128.ErrRoundingUnset), errors.Is(err, fin128.ErrNotBuilt), errors.Is(err, fin128.ErrTiming),
		errors.Is(err, fin128.ErrRule), errors.Is(err, fin128.ErrSolverSpec):
		// the binding validates all of these before calling fin128: reaching one is a defect, not a script fault
		return errs.NewInternalError(fmt.Sprintf("(%s) unexpected %v", ctx, err))
	}
	return errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", ctx, msg))
}

// finDecimal answers a fin128 (value, error) pair as a Kavun decimal or the translated error.
func finDecimal(ctx string, d dec128.Dec128, err error) (Value, error) {
	if err != nil {
		return Undefined, FinError(ctx, err)
	}
	return decimalResult(ctx, d)
}

// FinConventionArg reads a day-count convention by name through daycount.ByName — the one fin enum that keeps
// its market spellings and aliases ("ACT/365F", "30/360 US", ...), because products store those names verbatim.
func FinConventionArg(ctx, pos string, v Value) (daycount.Convention, error) {
	if v.Type != value.String {
		return daycount.Convention{}, errs.NewInvalidArgumentTypeError(ctx, pos, "string", v.TypeName())
	}
	s, _ := v.AsString()
	c, ok := daycount.ByName(s)
	if !ok {
		return daycount.Convention{}, errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown day-count convention %q, expected one of: %s (or a market alias)", ctx, s, strings.Join(daycount.Names(), ", ")))
	}
	return c, nil
}

// finRuleArg reads a banding rule: exactly "whole" (the band the amount lands in applies to all of it) or
// "marginal" (each slice at its own band's rate).
func finRuleArg(ctx, pos string, v Value) (fin128.Rule, error) {
	if v.Type != value.String {
		return 0, errs.NewInvalidArgumentTypeError(ctx, pos, "string", v.TypeName())
	}
	s, _ := v.AsString()
	switch s {
	case "whole":
		return fin128.Whole, nil
	case "marginal":
		return fin128.Marginal, nil
	}
	return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown rule %q, expected one of: whole, marginal", ctx, s))
}

// finRoundingArgs reads the trailing (scale, mode) pair at args[i] — decimal's own parser and messages.
func finRoundingArgs(ctx string, args []Value, i int) (fin128.Rounding, error) {
	scale, mode, err := decimalScaleModeArgs(ctx, args, i)
	if err != nil {
		return fin128.Rounding{}, err
	}
	return fin128.NewRounding(scale, mode), nil
}

func finDateArg(ctx, pos string, v Value) (civil.Date, error) {
	if v.Type != value.Date {
		return civil.Date{}, errs.NewInvalidArgumentTypeError(ctx, pos, "date", v.TypeName())
	}
	return dateOf(v), nil
}

// finRecordFields reads a band record (or dict) with EXACTLY the given keys: a missing or unknown key raises.
func finRecordFields(ctx, at string, v Value, keys ...string) (map[string]Value, error) {
	var m map[string]Value
	switch v.Type {
	case value.Record:
		m = (*Record)(v.Ptr).Elements
	case value.Dict:
		m = (*Dict)(v.Ptr).Elements
	default:
		return nil, errs.NewInvalidArgumentTypeError(ctx, at, "record or dict", v.TypeName())
	}
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if !slices.Contains(keys, k) {
			return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s: unknown key %q, expected: %s", ctx, at, k, strings.Join(keys, ", ")))
		}
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s: missing key %q (a band has: %s)", ctx, at, k, strings.Join(keys, ", ")))
		}
	}
	return m, nil
}

// finBandArray reads a non-empty array argument of band records, calling read for each.
func finBandArray(ctx string, v Value, read func(at string, e Value) error) error {
	if v.Type != value.Array {
		return errs.NewInvalidArgumentTypeError(ctx, "bands", "string or array", v.TypeName())
	}
	elems := (*Array)(v.Ptr).Elements
	for k, e := range elems {
		if err := read(fmt.Sprintf("bands[%d]", k), e); err != nil {
			return err
		}
	}
	return nil
}

// finTableText reads a table's text form, refusing the empty (or blank) text: fin128 parses it to its zero value,
// an undefined table, which a Kavun value never is.
func finTableText(ctx string, v Value) (string, error) {
	s, _ := v.AsString()
	if strings.TrimSpace(s) == "" {
		return "", FinError(ctx, fin128.ErrEmpty)
	}
	return s, nil
}

// finTableSource is a table's source form, `fin.tiered_rates("0:0.005, 1000:0.007")` — its constructor call.
func finTableSource(typeName, text string) string {
	return fmt.Sprintf("%s(%q)", typeName, text)
}

// finTableJSON is a table's JSON: its text form as a JSON string. The text grammar has no quote, backslash or
// control character, so it needs no escaping.
func finTableJSON(text string) []byte {
	return []byte(`"` + text + `"`)
}

// decimalsEqual compares two decimals numerically (1.50 == 1.5).
func decimalsEqual(a, b dec128.Dec128) bool {
	return a.Equal(b)
}

// finTableFormat renders a table: the text form, or the source form under `v`; only the generic width/alignment
// fields apply.
func finTableFormat(v Value, typeName, text string, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return finTableSource(typeName, text), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(typeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 || sp.HasUnconsumedTail() || sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(text, sp, fspec.AlignLeft), nil
}

// finTableCommonMember answers the members every table has: copy/freeze (identities on an immutable value),
// string(), format([spec]). ok is false for any other name.
func finTableCommonMember(v Value, name, text string, args []Value) (Value, bool, error) {
	switch name {
	case "copy", "freeze":
		if len(args) != 0 {
			return Undefined, true, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, true, nil
	case "string":
		r, err := convMember(name, v.TypeName(), args, true, NewStringValue(text))
		return r, true, err
	case "format":
		if len(args) > 1 {
			return Undefined, true, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		spec := ""
		if len(args) == 1 {
			if args[0].Type != value.String {
				return Undefined, true, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
			spec, _ = args[0].AsString()
		}
		sp, err := fspec.Parse(spec)
		if err != nil {
			return Undefined, true, errs.FromFormatSpecError(name, err)
		}
		s, err := finTableFormat(v, v.TypeName(), text, sp)
		if err != nil {
			return Undefined, true, err
		}
		return NewStringValue(s), true, nil
	}
	return Undefined, false, nil
}

// finTierPartsValue answers ChargeParts' result: an array of {from, to, rate, fixed, amount}.
func finTierPartsValue(parts []fin128.TierPart) Value {
	out := make([]Value, len(parts))
	for i, p := range parts {
		out[i] = NewRecordValue(map[string]Value{
			"from":   NewDecimalValue(p.From),
			"to":     NewDecimalValue(p.To),
			"rate":   NewDecimalValue(p.Rate),
			"fixed":  NewDecimalValue(p.Fixed),
			"amount": NewDecimalValue(p.Amount),
		}, false)
	}
	return NewArrayValue(out, false)
}

func finArgCount(name string, args []Value, counts ...int) error {
	if slices.Contains(counts, len(args)) {
		return nil
	}
	want := fmt.Sprint(counts[0])
	if len(counts) == 2 {
		want = fmt.Sprintf("%d or %d", counts[0], counts[1])
	}
	return errs.NewWrongNumArgumentsError(name, want, len(args))
}
