package core

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const decimalTypeName = "decimal"

func NewStaticDecimalValue(d *dec128.Dec128) Value {
	return Value{Type: value.Decimal, Immutable: true, Ptr: unsafe.Pointer(d)}
}

func NewDecimalValue(d dec128.Dec128) Value {
	return Value{Type: value.Decimal, Immutable: true, Ptr: unsafe.Pointer(&d)}
}

var TypeDecimal = ValueTypeDescr{
	Name:         ConstHook(decimalTypeName),
	String:       decimalTypeString,
	Format:       decimalTypeFormat,
	Interface:    func(v Value) any { return *(*dec128.Dec128)(v.Ptr) },
	EncodeJSON:   func(v Value) ([]byte, error) { return (*dec128.Dec128)(v.Ptr).MarshalJSON() },
	EncodeBinary: decimalTypeEncodeBinary,
	DecodeBinary: decimalTypeDecodeBinary,
	IsTrue:       func(v Value) bool { return !(*dec128.Dec128)(v.Ptr).IsZero() },
	Equal:        decimalTypeEqual,
	Len:          ConstHook(int64(1)),
	UnaryOp:      decimalTypeUnaryOp,
	BinaryOp:     decimalTypeBinaryOp,
	MethodCall:   decimalTypeMethodCall,
	AsString:     func(v Value) (string, bool) { return (*dec128.Dec128)(v.Ptr).String(), true },
	AsInt:        decimalTypeAsInt,
	AsFloat:      decimalTypeAsFloat,
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *(*dec128.Dec128)(v.Ptr), true },
	AsBool:       func(v Value) (bool, bool) { return !(*dec128.Dec128)(v.Ptr).IsZero(), true },
}

func decimalTypeEncodeBinary(v Value) ([]byte, error) {
	return (*dec128.Dec128)(v.Ptr).MarshalBinary()
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
	if o.IsNaN() {
		return `decimal("NaN")`
	}
	return o.String() + "d"
}

func decimalTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return decimalTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(decimalTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	d := *(*dec128.Dec128)(v.Ptr)

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
	o := (*dec128.Dec128)(v.Ptr)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsDecimal()
	if !ok {
		return false
	}
	l := (*dec128.Dec128)(v.Ptr)
	return l.Equal(r)
}

func decimalTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

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
		return NewStringValue(o.String()), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := decimalTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

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
		return NewErrorValue(NewStringValue(o.ErrorDetails().Error()), KindUser, false), nil

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
		return NewDecimalValue(o.ToScale(uint8(scale))), nil

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.Canonical()), nil

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.NextUp()), nil

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.NextDown()), nil

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.Abs()), nil

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.Neg()), nil

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewDecimalValue(o.Sqrt()), nil

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
		return NewDecimalValue(o.RoundDown(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundUp(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundTowardZero(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundAwayFromZero(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundHalfTowardZero(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundHalfAwayFromZero(uint8(scale))), nil

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
		return NewDecimalValue(o.RoundBank(uint8(scale))), nil

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
		return NewDecimalValue(o.Trunc(uint8(scale))), nil

	case "repeat":
		return repeatScalarToArray(v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, decimalTypeName)
	}
}

func decimalTypeUnaryOp(v Value, op token.Token) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch op {
	case token.Sub:
		return NewDecimalValue(o.Neg()), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func decimalTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsDecimal()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := (*dec128.Dec128)(v.Ptr)
	switch op {
	case token.Add:
		return NewDecimalValue(l.Add(r)), nil
	case token.Sub:
		return NewDecimalValue(l.Sub(r)), nil
	case token.Mul:
		return NewDecimalValue(l.Mul(r)), nil
	case token.Quo:
		return NewDecimalValue(l.Div(r)), nil
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
