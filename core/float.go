package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// FloatValue creates new boxed float value.
func FloatValue(f float64) Value {
	return Value{
		Type:  VT_FLOAT,
		Const: true,
		Data:  math.Float64bits(f),
	}
}

/* Float type methods */

func floatTypeName(v Value) string {
	return "float"
}

func floatTypeEncodeJSON(v Value) ([]byte, error) {
	var y []byte

	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) {
		return nil, errors.New("unsupported Inf value")
	}
	if math.IsNaN(f) {
		return nil, errors.New("unsupported NaN value")
	}

	// Convert as if by ES6 number to string conversion.
	// This matches most other JSON generators.
	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	y = strconv.AppendFloat(y, f, fmt, -1, 64)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(y)
		if n >= 4 && y[n-4] == 'e' && y[n-3] == '-' && y[n-2] == '0' {
			y[n-2] = y[n-1]
			y = y[:n-1]
		}
	}

	return y, nil
}

func floatTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v.Data)
	return b, nil
}

func floatTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("float: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.BigEndian.Uint64(data)
	return nil
}

func floatTypeString(v Value) string {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64)
}

func floatTypeFormat(v Value, s fspec.FormatSpec) (string, error) {
	if s.Verb == 'v' {
		return floatTypeString(v), nil
	}
	f := math.Float64frombits(v.Data)
	verb := s.Verb
	if verb == 0 {
		verb = 'g'
	}

	var (
		fmtVerb byte
		upper   bool
		percent bool
	)
	switch verb {
	case 'f':
		fmtVerb = 'f'
	case 'F':
		fmtVerb = 'f'
		upper = true
	case 'e':
		fmtVerb = 'e'
	case 'E':
		fmtVerb = 'E'
	case 'g':
		fmtVerb = 'g'
	case 'G':
		fmtVerb = 'G'
		upper = true
	case '%':
		fmtVerb = 'f'
		percent = true
	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	prec := -1
	if s.HasPrec {
		prec = int(s.Precision)
	} else {
		switch fmtVerb {
		case 'f':
			prec = 6
		case 'e', 'E':
			prec = 6
		case 'g', 'G':
			prec = -1
		}
	}

	if percent {
		f *= 100
	}

	negative := math.Signbit(f) && !math.IsNaN(f)

	// Special values: NaN / ±Inf bypass digit-shaping (no grouping, no zero-pad).
	if math.IsNaN(f) || math.IsInf(f, 0) {
		var body string
		switch {
		case math.IsNaN(f):
			body = "NaN"
			if upper {
				body = "NAN"
			}
		default: // Inf
			body = "Inf"
			if upper {
				body = "INF"
			}
			if negative {
				body = "-" + body
			} else {
				body = fspec.SignPrefix(s.Sign, false) + body
			}
		}
		return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil
	}

	// Render the magnitude; strconv emits its own leading '-' for negatives, which we strip and re-emit explicitly so
	// that grouping / sign-aware split work uniformly.
	raw := strconv.FormatFloat(f, fmtVerb, prec, 64)
	if upper {
		raw = strings.ToUpper(raw)
	}
	if strings.HasPrefix(raw, "-") {
		raw = raw[1:]
	}

	// 'z' flag: coerce -0 (and -0.000…) to +0 once rounding has produced an all-zero magnitude.
	if s.CoerceZero && negative && isAllZeroMagnitude(raw) {
		negative = false
	}

	// Grouping applies to the integral part only.
	if s.Grouping != 0 {
		if s.Grouping != ',' && s.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		raw = groupFloatIntegral(raw, s.Grouping)
	}

	if percent {
		raw += "%"
	}

	sign := fspec.SignPrefix(s.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil
}

// isAllZeroMagnitude reports whether a magnitude string (no leading sign) numerically equals zero. It accepts forms
// like "0", "0.000", "0e+00", "0.0E-05".
func isAllZeroMagnitude(s string) bool {
	for _, r := range s {
		switch r {
		case '0', '.':
			continue
		case 'e', 'E':
			return true // remainder is the exponent; mantissa was all zeros
		default:
			return false
		}
	}
	return true
}

// groupFloatIntegral inserts sep into the integral part of a magnitude string (no leading sign). The integral part is
// everything up to the first '.', 'e' or 'E'.
func groupFloatIntegral(s string, sep byte) string {
	end := len(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', 'e', 'E':
			end = i
			i = len(s)
		}
	}
	if end == 0 {
		return s
	}
	return fspec.GroupDigits(s[:end], sep, 3) + s[end:]
}

func floatTypeInterface(v Value) any {
	return math.Float64frombits(v.Data)
}

func floatTypeIsTrue(v Value) bool {
	return !math.IsNaN(math.Float64frombits(v.Data))
}

func floatTypeAsInt(v Value) (int64, bool) {
	return int64(math.Float64frombits(v.Data)), true
}

func floatTypeAsString(v Value) (string, bool) {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), true
}

func floatTypeAsFloat(v Value) (float64, bool) {
	return math.Float64frombits(v.Data), true
}

func floatTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return dec128.NaN(state.NaN), false
	}
	return dec128.FromFloat64(f), true
}

func floatTypeAsBool(v Value) (bool, bool) {
	return !math.IsNaN(math.Float64frombits(v.Data)), true
}

func floatTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsFloat()
	if !ok {
		return false
	}
	return math.Float64frombits(v.Data) == r
}

func floatTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f := math.Float64frombits(v.Data)
		alloc := vm.Allocator()
		d := alloc.NewDecimal()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			*d = dec128.NaN(state.NaN)
			return DecimalValue(d), nil
		}
		*d = dec128.FromFloat64(f)
		return DecimalValue(d), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := v.AsInt()
		return IntValue(i), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s), nil

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f := math.Float64frombits(v.Data)
		if math.IsNaN(f) {
			return IntValue(0), nil
		}
		if f > 0 {
			return IntValue(1), nil
		}
		if f < 0 {
			return IntValue(-1), nil
		}
		return IntValue(0), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "float")
	}
}

func floatTypeUnaryOp(v Value, a *Arena, op token.Token) (Value, error) {
	f := math.Float64frombits(v.Data)
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		return FloatValue(-f), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func floatTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsFloat()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := math.Float64frombits(v.Data)
	switch op {
	case token.Add:
		return FloatValue(l + r), nil
	case token.Sub:
		return FloatValue(l - r), nil
	case token.Mul:
		return FloatValue(l * r), nil
	case token.Quo:
		return FloatValue(l / r), nil
	case token.Less:
		return BoolValue(l < r), nil
	case token.Greater:
		return BoolValue(l > r), nil
	case token.LessEq:
		return BoolValue(l <= r), nil
	case token.GreaterEq:
		return BoolValue(l >= r), nil
	default:
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}
