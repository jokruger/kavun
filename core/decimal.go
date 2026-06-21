package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const decimalTypeName = "decimal"

func (a *Arena) MustNewDecimalValue(d dec128.Dec128) Value {
	v, err := a.NewDecimalValue(d)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewDecimalValue(d dec128.Dec128) (Value, error) {
	if ref, p, ok := a.arena.New(value.Decimal); ok {
		*(*dec128.Dec128)(p) = d
		return Value{Type: value.Decimal, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(decimalTypeName)
}

var TypeDecimal = ValueTypeDescr{
	Name:         ConstHook(decimalTypeName),
	String:       decimalTypeString,
	Format:       decimalTypeFormat,
	Interface:    func(v Value) any { return *a.ResolveDecimalValue(v) },
	EncodeJSON:   func(v Value) ([]byte, error) { return a.ResolveDecimalValue(v).MarshalJSON() },
	EncodeBinary: decimalTypeEncodeBinary,
	DecodeBinary: decimalTypeDecodeBinary,
	IsTrue:       func(v Value) bool { return !a.ResolveDecimalValue(v).IsZero() },
	Equal:        decimalTypeEqual,
	Len:          ConstHook(int64(1)),
	UnaryOp:      decimalTypeUnaryOp,
	BinaryOp:     decimalTypeBinaryOp,
	MethodCall:   decimalTypeMethodCall,
	AsString:     func(v Value) (string, bool) { return a.ResolveDecimalValue(v).String(), true },
	AsInt:        decimalTypeAsInt,
	AsFloat:      decimalTypeAsFloat,
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *a.ResolveDecimalValue(v), true },
	AsBool:       func(v Value) (bool, bool) { return !a.ResolveDecimalValue(v).IsZero(), true },
}

func decimalTypeEncodeBinary(v Value) ([]byte, error) {
	return a.ResolveDecimalValue(v).MarshalBinary()
}

func decimalTypeDecodeBinary(v *Value, data []byte) error {
	var d dec128.Dec128
	if err := d.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("failed to decode decimal: %w", err)
	}
	nv, err := a.NewDecimalValue(d)
	if err != nil {
		return fmt.Errorf("failed to decode decimal: %w", err)
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = nv
	return nil
}

func decimalTypeString(v Value) string {
	o := a.ResolveDecimalValue(v)
	if o.IsNaN() {
		return `decimal("NaN")`
	}
	return o.String() + "d"
}

func decimalTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return decimalTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(decimalTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	d := *a.ResolveDecimalValue(v)

	if sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	// NaN bypasses digit shaping.
	if d.IsNaN() {
		body := "NaN"
		switch sp.Verb {
		case 'F', 'E', 'G':
			body = "NAN"
		}
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
	}

	verb := sp.Verb
	prec := -1
	if sp.HasPrec {
		prec = int(sp.Precision)
	} else {
		switch verb {
		case 'f', 'F', '%', 'e', 'E':
			prec = 6
		}
	}

	negative := d.IsNegative()
	abs := d.Abs()
	var raw string // magnitude string, no leading sign

	switch verb {
	case 0:
		// default: canonical fixed-point string; trailing zeros trimmed.
		raw = abs.String()

	case 'f', 'F':
		raw = decimalFixedString(abs, prec)

	case '%':
		raw = decimalFixedString(abs.Mul(dec128.FromInt64(100)), prec) + "%"

	case 's':
		// Preserve source scale; no trim of trailing zeros.
		raw = abs.StringFixed()

	case 'e', 'E', 'g', 'G':
		// Fall back to float64 for scientific / shortest forms — adequate for the typical case where these verbs are
		// chosen for human-readable output rather than full precision.
		f, err := abs.InexactFloat64()
		if err != nil {
			return "", fmt.Errorf("decimal: cannot format %s with verb %c: %w", d.String(), verb, err)
		}
		raw = strconv.FormatFloat(f, byte(verb), prec, 64)

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	// 'z' coerce-zero: drop sign when the formatted magnitude is numerically zero.
	if sp.CoerceZero && negative && isAllZeroMagnitude(strings.TrimSuffix(raw, "%")) {
		negative = false
	}

	if sp.Grouping != 0 {
		if sp.Grouping != ',' && sp.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		hasPct := strings.HasSuffix(raw, "%")
		if hasPct {
			raw = raw[:len(raw)-1]
		}
		raw = groupFloatIntegral(raw, sp.Grouping)
		if hasPct {
			raw += "%"
		}
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

// decimalFixedString renders a non-negative Dec128 in fixed-point notation with exactly prec fractional digits (no
// trailing-zero trim). If prec < 0, the canonical representation is returned (trailing zeros trimmed).
func decimalFixedString(d dec128.Dec128, prec int) string {
	if prec < 0 {
		return d.String()
	}
	if prec > int(dec128.MaxScale) {
		prec = int(dec128.MaxScale)
	}
	rounded := d.RoundHalfAwayFromZero(uint8(prec))
	s := rounded.String()
	dot := strings.IndexByte(s, '.')
	var intp, fracp string
	if dot < 0 {
		intp, fracp = s, ""
	} else {
		intp, fracp = s[:dot], s[dot+1:]
	}
	if len(fracp) < prec {
		fracp += strings.Repeat("0", prec-len(fracp))
	} else if len(fracp) > prec {
		fracp = fracp[:prec]
	}
	if prec == 0 {
		return intp
	}
	return intp + "." + fracp
}

func decimalTypeAsInt(v Value) (int64, bool) {
	o := a.ResolveDecimalValue(v)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := a.ResolveDecimalValue(v)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsDecimal(a)
	if !ok {
		return false
	}
	l := a.ResolveDecimalValue(v)
	return l.Equal(r)
}

func decimalTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := a.ResolveDecimalValue(v)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, err := o.InexactFloat64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to float: %w", err)
		}
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, err := o.Int64()
		if err != nil {
			return Undefined, fmt.Errorf("failed to convert decimal to int: %w", err)
		}
		return IntValue(i), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(o.String())

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := decimalTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

	case "is_zero":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsZero()), nil

	case "is_negative":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsNegative()), nil

	case "is_positive":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsPositive()), nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsNaN()), nil

	case "error_details":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		nv, err := a.NewStringValue(o.ErrorDetails().Error())
		if err != nil {
			return Undefined, fmt.Errorf("failed to get decimal error details: %w", err)
		}
		return a.NewErrorValue(nv, KindUser, false)

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Sign())), nil

	case "scale":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(o.Scale())), nil

	case "rescale":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.ToScale(uint8(scale)))

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.Canonical())

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.NextUp())

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.NextDown())

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.Abs())

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.Neg())

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewDecimalValue(o.Sqrt())

	case "round_down":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundDown(uint8(scale)))

	case "round_up":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundUp(uint8(scale)))

	case "round_toward_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundTowardZero(uint8(scale)))

	case "round_away_from_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundAwayFromZero(uint8(scale)))

	case "round_half_toward_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundHalfTowardZero(uint8(scale)))

	case "round_half_away_from_zero":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundHalfAwayFromZero(uint8(scale)))

	case "round_bank":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.RoundBank(uint8(scale)))

	case "trunc":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "scale", "int", args[0].TypeName())
		}
		if scale < 0 || scale > int64(dec128.MaxScale) {
			return Undefined, fmt.Errorf("scale must be between 0 and %d", dec128.MaxScale)
		}
		return a.NewDecimalValue(o.Trunc(uint8(scale)))

	case "repeat":
		return repeatScalarToArray(a, v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, decimalTypeName)
	}
}

func decimalTypeUnaryOp(v Value, op token.Token) (Value, error) {
	o := a.ResolveDecimalValue(v)

	switch op {
	case token.Sub:
		return a.NewDecimalValue(o.Neg())

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func decimalTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsDecimal(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := a.ResolveDecimalValue(v)
	switch op {
	case token.Add:
		return a.NewDecimalValue(l.Add(r))
	case token.Sub:
		return a.NewDecimalValue(l.Sub(r))
	case token.Mul:
		return a.NewDecimalValue(l.Mul(r))
	case token.Quo:
		return a.NewDecimalValue(l.Div(r))
	case token.Less:
		return BoolValue(l.LessThan(r)), nil
	case token.Greater:
		return BoolValue(l.GreaterThan(r)), nil
	case token.LessEq:
		return BoolValue(l.LessThanOrEqual(r)), nil
	case token.GreaterEq:
		return BoolValue(l.GreaterThanOrEqual(r)), nil
	default:
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}
