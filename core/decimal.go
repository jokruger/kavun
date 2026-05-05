package core

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// DecimalValue creates new boxed decimal value.
func DecimalValue(d *dec128.Dec128) Value {
	return Value{
		Type:  VT_DECIMAL,
		Const: true,
		Ptr:   unsafe.Pointer(d),
	}
}

// NewDecimalValue creates new (heap-allocated) boxed decimal value.
func NewDecimalValue(d dec128.Dec128) Value {
	o := &d
	return DecimalValue(o)
}

/* Decimal type methods */

func decimalTypeName(v Value) string {
	return "decimal"
}

func decimalTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*dec128.Dec128)(v.Ptr)
	return o.MarshalJSON()
}

func decimalTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*dec128.Dec128)(v.Ptr)
	return o.MarshalBinary()
}

func decimalTypeDecodeBinary(v *Value, data []byte) error {
	var d dec128.Dec128
	if err := d.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("failed to decode decimal: %w", err)
	}
	*v = NewDecimalValue(d)
	return nil
}

func decimalTypeString(v Value) string {
	o := (*dec128.Dec128)(v.Ptr)
	return o.String()
}

func decimalTypeFormat(v Value, s fspec.FormatSpec) (string, error) {
	d := *(*dec128.Dec128)(v.Ptr)

	// 'v' = Kavun source form, ignores other generic fields per the spec.
	if s.Verb == 'v' {
		return d.String() + "d", nil
	}

	// NaN bypasses digit shaping.
	if d.IsNaN() {
		body := "NaN"
		switch s.Verb {
		case 'F', 'E', 'G':
			body = "NAN"
		}
		return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil
	}

	verb := s.Verb
	prec := -1
	if s.HasPrec {
		prec = int(s.Precision)
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
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	// 'z' coerce-zero: drop sign when the formatted magnitude is numerically zero.
	if s.CoerceZero && negative && isAllZeroMagnitude(strings.TrimSuffix(raw, "%")) {
		negative = false
	}

	if s.Grouping != 0 {
		if s.Grouping != ',' && s.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		hasPct := strings.HasSuffix(raw, "%")
		if hasPct {
			raw = raw[:len(raw)-1]
		}
		raw = groupFloatIntegral(raw, s.Grouping)
		if hasPct {
			raw += "%"
		}
	}

	sign := fspec.SignPrefix(s.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil
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

func decimalTypeInterface(v Value) any {
	o := (*dec128.Dec128)(v.Ptr)
	return *o
}

func decimalTypeIsTrue(v Value) bool {
	o := (*dec128.Dec128)(v.Ptr)
	return !o.IsZero()
}

func decimalTypeAsInt(v Value) (int64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsString(v Value) (string, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	return o.String(), true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	return *o, true
}

func decimalTypeAsBool(v Value) (bool, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	return !o.IsZero(), true
}

func decimalTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsDecimal()
	if !ok {
		return false
	}
	l := (*dec128.Dec128)(v.Ptr)
	return l.Equal(r)
}

func decimalTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)
	alloc := vm.Allocator()

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
		return alloc.NewStringValue(o.String()), nil

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
		e := alloc.NewStringValue(o.ErrorDetails().Error())
		return vm.Allocator().NewErrorValue(e), nil

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
		d := alloc.NewDecimal()
		*d = o.ToScale(uint8(scale))
		return DecimalValue(d), nil

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.Canonical()
		return DecimalValue(d), nil

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.NextUp()
		return DecimalValue(d), nil

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.NextDown()
		return DecimalValue(d), nil

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.Abs()
		return DecimalValue(d), nil

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.Neg()
		return DecimalValue(d), nil

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d := alloc.NewDecimal()
		*d = o.Sqrt()
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundDown(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundUp(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundTowardZero(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundAwayFromZero(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundHalfTowardZero(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundHalfAwayFromZero(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.RoundBank(uint8(scale))
		return DecimalValue(d), nil

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
		d := alloc.NewDecimal()
		*d = o.Trunc(uint8(scale))
		return DecimalValue(d), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "decimal")
	}
}

func decimalTypeUnaryOp(v Value, a *Arena, op token.Token) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch op {
	case token.Sub:
		d := a.NewDecimal()
		*d = o.Neg()
		return DecimalValue(d), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func decimalTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsDecimal()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := (*dec128.Dec128)(v.Ptr)
	switch op {
	case token.Add:
		d := a.NewDecimal()
		*d = l.Add(r)
		return DecimalValue(d), nil
	case token.Sub:
		d := a.NewDecimal()
		*d = l.Sub(r)
		return DecimalValue(d), nil
	case token.Mul:
		d := a.NewDecimal()
		*d = l.Mul(r)
		return DecimalValue(d), nil
	case token.Quo:
		d := a.NewDecimal()
		*d = l.Div(r)
		return DecimalValue(d), nil
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
