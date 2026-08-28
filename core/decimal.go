package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
	Name:         ConstHook(decimalTypeName),                                                     // PURE by contract
	String:       decimalTypeString,                                                              // PURE by contract
	Format:       decimalTypeFormat,                                                              // PURE by contract
	Interface:    func(v Value) any { return *(*dec128.Dec128)(v.Ptr) },                          // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return (*dec128.Dec128)(v.Ptr).MarshalJSON() }, // PURE by contract
	EncodeBinary: decimalTypeEncodeBinary,                                                        // PURE by contract
	DecodeBinary: decimalTypeDecodeBinary,                                                        // IMPURE by contract (mutates target)
	IsTrue:       decimalTypeIsTrue,                                                              // PURE by contract
	Equal:        decimalTypeEqual,                                                               // PURE by contract
	BinaryOp:     decimalTypeBinaryOp,                                                            // PURE by contract
	UnaryOp:      decimalTypeUnaryOp,                                                             // PURE by contract
	Len:          ConstHook(int64(1)),                                                            // PURE by contract
	MethodCall:   decimalTypeMethodCall,                                                          // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return (*dec128.Dec128)(v.Ptr).String(), true }, // PURE by contract
	AsInt:        decimalTypeAsInt,                                                               // PURE by contract
	AsFloat:      decimalTypeAsFloat,                                                             // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *(*dec128.Dec128)(v.Ptr), true },  // PURE by contract
	AsTime:       decimalTypeAsTime,                                                              // PURE by contract
	AsBool:       decimalTypeAsBool,                                                              // PURE by contract
	IsMethodPure: func(string) bool { return true },                                              // All methods are expected to be pure.
}

// decimal NaN is an error state, not a domain value; a boolean context refuses
// the question rather than answering it (as arithmetic and conversion already do)
func decimalTypeIsTrue(v Value) (bool, error) {
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return false, errs.NewInvalidValueError("decimal NaN is neither true nor false in a boolean context")
	}
	return !o.IsZero(), nil
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
			return "", errs.NewFormattingError(fmt.Sprintf("decimal: cannot format %s with verb %c: %s", d.String(), verb, err))
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

// PURE by contract. A decimal in conversion context is a unix timestamp read as sec.frac: the
// integer part is seconds since epoch, the fraction is the sub-second part. Always UTC, like every
// other int-shaped conversion.
//
// This is the EXACT sec.frac path -- dec128 is base-10, so decimal("1704067200.123456789") converts
// with every digit intact, where the float64 spelling of the same number cannot. Anything finer than
// nanoseconds truncates (that is the resolution of a time value, not of the decimal). NaN and
// out-of-int64-range values decline, surfacing as time(x) -> undefined or the time(x, fallback)
// default.
func decimalTypeAsTime(v Value) (time.Time, bool) {
	d := *(*dec128.Dec128)(v.Ptr)
	if d.IsNaN() {
		return time.Time{}, false
	}
	whole := d.Trunc(0)
	sec, err := whole.Int64()
	if err != nil {
		return time.Time{}, false
	}
	nsec, err := d.Sub(whole).MulInt64(1_000_000_000).Trunc(0).Int64()
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, nsec).UTC(), true
}

func decimalTypeAsInt(v Value) (int64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	i, err := o.Int64()
	if err != nil {
		return 0, false
	}
	return i, true
}

func decimalTypeAsBool(v Value) (bool, bool) {
	// the conversion is a zero check; NaN is an error state and declines
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return false, false
	}
	return !o.IsZero(), true
}

func decimalTypeAsFloat(v Value) (float64, bool) {
	o := (*dec128.Dec128)(v.Ptr)
	f, err := o.InexactFloat64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func decimalTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Decimal:
		l := (*dec128.Dec128)(v.Ptr)
		r := (*dec128.Dec128)(other.Ptr)
		return l.Equal(*r)
	case value.Int, value.Rune, value.Byte, value.Bool:
		l := (*dec128.Dec128)(v.Ptr)
		r, _ := other.AsInt() // always succeeds and is exact for Int/Rune/Byte/Bool
		return l.Equal(dec128.FromInt64(r))
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.

// decimalArithResult guards the overflow class of decimal arithmetic: a result
// that is NaN while both operands were domain values is an out-of-range result
// (dec128 is a fixed 128-bit decimal) and raises. Division by zero still
// answers NaN for now — its treatment is deliberately deferred (see the NaN
// item in TODO.md) and must be decided together with the error-handling policy.
func decimalArithResult(d dec128.Dec128, lNaN, rNaN bool, division bool) (Value, error) {
	if d.IsNaN() && !lNaN && !rNaN && !division {
		return Undefined, errs.NewInvalidValueError("decimal overflow")
	}
	return NewDecimalValue(d), nil
}

func decimalTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		r := (*dec128.Dec128)(v.Ptr)
		switch other.Type {
		case value.Bool, value.Byte, value.Rune:
			l := dec128.FromInt64(int64(other.Data))
			switch op {
			case token.Less:
				return BoolValue(l.LessThan(*r)), nil
			case token.Greater:
				return BoolValue(l.GreaterThan(*r)), nil
			case token.LessEq:
				return BoolValue(l.LessThanOrEqual(*r)), nil
			case token.GreaterEq:
				return BoolValue(l.GreaterThanOrEqual(*r)), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	l := (*dec128.Dec128)(v.Ptr)
	switch other.Type {
	case value.Decimal:
		r := *(*dec128.Dec128)(other.Ptr)
		switch op {
		case token.Add:
			return decimalArithResult(l.Add(r), l.IsNaN(), r.IsNaN(), false)
		case token.Sub:
			return decimalArithResult(l.Sub(r), l.IsNaN(), r.IsNaN(), false)
		case token.Mul:
			return decimalArithResult(l.Mul(r), l.IsNaN(), r.IsNaN(), false)
		case token.Quo:
			return decimalArithResult(l.Div(r), l.IsNaN(), r.IsNaN(), true)
		case token.Rem:
			return decimalArithResult(l.Mod(r), l.IsNaN(), r.IsNaN(), true)
		case token.Less:
			return BoolValue(l.LessThan(r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Int:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case token.Add:
			return decimalArithResult(l.Add(r), l.IsNaN(), r.IsNaN(), false)
		case token.Sub:
			return decimalArithResult(l.Sub(r), l.IsNaN(), r.IsNaN(), false)
		case token.Mul:
			return decimalArithResult(l.Mul(r), l.IsNaN(), r.IsNaN(), false)
		case token.Quo:
			return decimalArithResult(l.Div(r), l.IsNaN(), r.IsNaN(), true)
		case token.Rem:
			return decimalArithResult(l.Mod(r), l.IsNaN(), r.IsNaN(), true)
		case token.Less:
			return BoolValue(l.LessThan(r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Bool, value.Byte, value.Rune:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case token.Less:
			return BoolValue(l.LessThan(r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func decimalTypeUnaryOp(v Value, op token.Token) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch op {
	case token.Sub:
		return NewDecimalValue(o.Neg()), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func decimalTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value regardless of copy depth
		return v, nil

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable already, so freeze/freeze_shallow are no-ops
		return v, nil

	case "decimal":
		return convMember(name, decimalTypeName, args, true, v)

	case "float":
		f, ok := v.AsFloat()
		return convMember(name, decimalTypeName, args, ok, FloatValue(f))

	case "int":
		i, ok := v.AsInt()
		return convMember(name, decimalTypeName, args, ok, IntValue(i))

	case "time":
		t, ok := v.AsTime()
		return convMember(name, decimalTypeName, args, ok, NewTimeValue(t))

	case "bool":
		b, ok := v.AsBool()
		return convMember(name, decimalTypeName, args, ok, BoolValue(b))

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(o.String()), nil

	case "runes":
		s, ok := v.AsString()
		return convMember(name, decimalTypeName, args, ok, NewRunesValue([]rune(s), false))

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

	case "is_inf":
		// constant false: dec128 has no Inf representation at all
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return False, nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsNaN()), nil

	case "error_details":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// a valid decimal has no error details; ErrorDetails() is nil then and must not be dereferenced
		if !o.IsNaN() {
			return Undefined, nil
		}
		if details := o.ErrorDetails(); details != nil {
			return NewErrorValue(NewStringValue(details.Error()), KindUser, false), nil
		}
		return Undefined, nil

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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
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
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
		}
		return NewDecimalValue(o.Trunc(uint8(scale))), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, decimalTypeName)
	}
}
