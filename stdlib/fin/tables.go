package fin

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

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// Shared machinery of the fin module's types (fin.year_fraction and the four rate/charge tables) and functions:
// the single fin128 → Kavun error translation, the argument readers the table members share with the module, and
// the band ↔ record codec the table constructors and bands() use.

// Error is the one translation point from a fin128 or dec128 error to a Kavun error, for the fin module's
// functions and its types' members alike. ctx names the operation ("fin.payment", "charge").
func Error(ctx string, err error) error {
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

// decimal answers a fin128 (value, error) pair as a Kavun decimal or the translated error.
func decimal(ctx string, d dec128.Dec128, err error) (core.Value, error) {
	if err != nil {
		return core.Undefined, Error(ctx, err)
	}
	return core.DecimalResult(ctx, d)
}

// ConventionArg reads a day-count convention by name through daycount.ByName — the one fin enum that keeps
// its market spellings and aliases ("ACT/365F", "30/360 US", ...), because products store those names verbatim.
func ConventionArg(ctx, pos string, v core.Value) (daycount.Convention, error) {
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

// ruleArg reads a banding rule: exactly "whole" (the band the amount lands in applies to all of it) or
// "marginal" (each slice at its own band's rate).
func ruleArg(ctx, pos string, v core.Value) (fin128.Rule, error) {
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

// roundingArgs reads the trailing (scale, mode) pair at args[i] — decimal's own parser and messages.
func roundingArgs(ctx string, args []core.Value, i int) (fin128.Rounding, error) {
	scale, mode, err := core.ScaleModeArgs(ctx, args, i)
	if err != nil {
		return fin128.Rounding{}, err
	}
	return fin128.NewRounding(scale, mode), nil
}

func dateArg(ctx, pos string, v core.Value) (civil.Date, error) {
	if v.Type != value.Date {
		return civil.Date{}, errs.NewInvalidArgumentTypeError(ctx, pos, "date", v.TypeName())
	}
	return core.DateOf(v), nil
}

// recordFields reads a band record (or dict) with EXACTLY the given keys: a missing or unknown key raises.
func recordFields(ctx, at string, v core.Value, keys ...string) (map[string]core.Value, error) {
	var m map[string]core.Value
	switch v.Type {
	case value.Record:
		m = (*core.Record)(v.Ptr).Elements
	case value.Dict:
		m = (*core.Dict)(v.Ptr).Elements
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

// bandArray reads a non-empty array argument of band records, calling read for each.
func bandArray(ctx string, v core.Value, read func(at string, e core.Value) error) error {
	if v.Type != value.Array {
		return errs.NewInvalidArgumentTypeError(ctx, "bands", "string or array", v.TypeName())
	}
	elems := (*core.Array)(v.Ptr).Elements
	for k, e := range elems {
		if err := read(fmt.Sprintf("bands[%d]", k), e); err != nil {
			return err
		}
	}
	return nil
}

// tableText reads a table's text form, refusing the empty (or blank) text: fin128 parses it to its zero value,
// an undefined table, which a Kavun value never is.
func tableText(ctx string, v core.Value) (string, error) {
	s, _ := v.AsString()
	if strings.TrimSpace(s) == "" {
		return "", Error(ctx, fin128.ErrEmpty)
	}
	return s, nil
}

// tableSource is a table's source form, `fin.tiered_rates("0:0.005, 1000:0.007")` — its constructor call.
func tableSource(typeName, text string) string {
	return fmt.Sprintf("%s(%q)", typeName, text)
}

// tableJSON is a table's JSON: its text form as a JSON string. The text grammar has no quote, backslash or
// control character, so it needs no escaping.
func tableJSON(text string) []byte {
	return []byte(`"` + text + `"`)
}

// decimalsEqual compares two decimals numerically (1.50 == 1.5).
func decimalsEqual(a, b dec128.Dec128) bool {
	return a.Equal(b)
}

// tableFormat renders a table: the text form, or the source form under `v`; only the generic width/alignment
// fields apply.
func tableFormat(v core.Value, typeName, text string, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return tableSource(typeName, text), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(typeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 || sp.HasUnconsumedTail() || sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.HasPrec || sp.ZeroPad || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(text, sp, fspec.AlignLeft), nil
}

// The members every table has — copy/freeze (identities on an immutable value), string(), format([spec]) — are
// entries in each table's member table; copy/freeze/format are the shared MemberSelf/MemberFormat, string is this.

// tableString is string([default]): the table's canonical text, the form its constructor reads back.
func tableString(v core.Value, text string, args []core.Value) (core.Value, error) {
	return core.ConvMember("string", v.TypeName(), args, true, core.NewStringValue(text))
}

// chargeArgs reads the amount-banded tables' (amount, rule, scale, mode): charge, rate, charge_parts.
func chargeArgs(name string, args []core.Value) (dec128.Dec128, fin128.Rule, fin128.Rounding, error) {
	if err := argCount(name, args, 4); err != nil {
		return dec128.Dec128{}, 0, fin128.Rounding{}, err
	}
	amount, err := core.DecimalOperandArg(name, "amount", args[0])
	if err != nil {
		return dec128.Dec128{}, 0, fin128.Rounding{}, err
	}
	rule, err := ruleArg(name, "rule", args[1])
	if err != nil {
		return dec128.Dec128{}, 0, fin128.Rounding{}, err
	}
	out, err := roundingArgs(name, args, 2)
	if err != nil {
		return dec128.Dec128{}, 0, fin128.Rounding{}, err
	}
	return amount, rule, out, nil
}

// accrual is the date-banded rate table's accrual request: (principal, start, end, convention, scale, mode) —
// interest over [start, end), the rate changing at band boundaries, each stretch measured by the convention.
type accrual struct {
	principal dec128.Dec128
	start     civil.Date
	end       civil.Date
	conv      daycount.Convention
	out       fin128.Rounding
}

// accrualArgs reads accrue/accrue_parts' arguments.
func accrualArgs(name string, args []core.Value) (accrual, error) {
	var a accrual
	if err := argCount(name, args, 6); err != nil {
		return a, err
	}
	var err error
	if a.principal, err = core.DecimalOperandArg(name, "principal", args[0]); err != nil {
		return a, err
	}
	if a.start, err = dateArg(name, "start", args[1]); err != nil {
		return a, err
	}
	if a.end, err = dateArg(name, "end", args[2]); err != nil {
		return a, err
	}
	if a.conv, err = ConventionArg(name, "convention", args[3]); err != nil {
		return a, err
	}
	if a.out, err = roundingArgs(name, args, 4); err != nil {
		return a, err
	}
	return a, nil
}

// tierPartsValue answers ChargeParts' result: an array of {from, to, rate, fixed, amount}.
func tierPartsValue(parts []fin128.TierPart) core.Value {
	out := make([]core.Value, len(parts))
	for i, p := range parts {
		out[i] = core.NewRecordValue(map[string]core.Value{
			"from":   core.NewDecimalValue(p.From),
			"to":     core.NewDecimalValue(p.To),
			"rate":   core.NewDecimalValue(p.Rate),
			"fixed":  core.NewDecimalValue(p.Fixed),
			"amount": core.NewDecimalValue(p.Amount),
		}, false)
	}
	return core.NewArrayValue(out, false)
}

func argCount(name string, args []core.Value, counts ...int) error {
	if slices.Contains(counts, len(args)) {
		return nil
	}
	want := fmt.Sprint(counts[0])
	if len(counts) == 2 {
		want = fmt.Sprintf("%d or %d", counts[0], counts[1])
	}
	return errs.NewWrongNumArgumentsError(name, want, len(args))
}
