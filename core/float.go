package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const floatTypeName = "float"

func FloatValue(f float64) Value {
	return Value{
		Type:      value.Float,
		Immutable: true,
		Data:      math.Float64bits(f),
	}
}

var TypeFloat = ValueTypeDescr{
	Name:         ConstHook(floatTypeName),                                  // PURE by contract
	String:       floatTypeString,                                           // PURE by contract
	Format:       floatTypeFormat,                                           // PURE by contract
	Interface:    func(v Value) any { return math.Float64frombits(v.Data) }, // PURE by contract
	EncodeJSON:   floatTypeEncodeJSON,                                       // PURE by contract
	EncodeBinary: floatTypeEncodeBinary,                                     // PURE by contract
	DecodeBinary: floatTypeDecodeBinary,                                     // IMPURE by contract (mutates target)
	IsTrue:       floatTypeIsTrue,                                           // PURE by contract
	Len:          ConstHook(int64(1)),                                       // PURE by contract
	Equal:        floatTypeEqual,                                            // PURE by contract
	BinaryOp:     floatTypeBinaryOp,                                         // PURE by contract
	UnaryOp:      floatTypeUnaryOp,                                          // PURE by contract
	MethodCall:   floatTypeMethodCall,                                       // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsInt:        floatTypeAsInt,                                            // PURE by contract
	AsFloat:      floatTypeAsFloat,                                          // PURE by contract
	AsDecimal:    floatTypeAsDecimal,                                        // PURE by contract
	AsBool:       floatTypeAsBool,                                           // PURE by contract
	AsString:     floatTypeAsString,                                         // PURE by contract
	AsTime:       floatTypeAsTime,                                           // PURE by contract
	IsMethodPure: func(string) bool { return true },                         // All methods are expected to be pure.
}

func floatTypeIsTrue(v Value) (bool, error) {
	// truthiness is inequality with the type's zero value, so 0.0 is falsy like
	// 0 and decimal(0); NaN is an error state, not a domain value, and a boolean
	// context refuses the question rather than answering it
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) {
		return false, errs.NewInvalidValueError("float NaN is neither true nor false in a boolean context")
	}
	return f != 0, nil
}

func floatTypeString(v Value) string {
	f := math.Float64frombits(v.Data)
	s := strconv.FormatFloat(f, 'f', -1, 64)
	// A whole float keeps its point in the DISPLAY form, so `[3.0]` never reads back as an int.
	// The text CONVERSION (AsString, .string(), a dict key, join) stays bare on purpose: `3 == 3.0`
	// is true, so the two must render the same key text.
	if !math.IsInf(f, 0) && !math.IsNaN(f) && !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
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

	// Convert as if by ES6 number to string conversion. This matches most other JSON generators.
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
	binary.LittleEndian.PutUint64(b, v.Data)
	return b, nil
}

func floatTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("float: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.LittleEndian.Uint64(data)
	return nil
}

func floatTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(floatTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	f := math.Float64frombits(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'g'
	}

	if sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
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
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	prec := -1
	if sp.HasPrec {
		prec = int(sp.Precision)
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
				body = fspec.SignPrefix(sp.Sign, false) + body
			}
		}
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
	}

	// Render the magnitude; strconv emits its own leading '-' for negatives, which we strip and re-emit explicitly so
	// that grouping / sign-aware split work uniformly.
	raw := strings.TrimPrefix(strconv.FormatFloat(f, fmtVerb, prec, 64), "-")
	if upper {
		raw = strings.ToUpper(raw)
	}

	// 'z' flag: coerce -0 (and -0.000…) to +0 once rounding has produced an all-zero magnitude.
	if sp.CoerceZero && negative && isAllZeroMagnitude(raw) {
		negative = false
	}

	// Grouping applies to the integral part only.
	if sp.Grouping != 0 {
		if sp.Grouping != ',' && sp.Grouping != '_' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = groupFloatIntegral(raw, sp.Grouping)
	}

	if percent {
		raw += "%"
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + raw
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
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

func floatTypeAsInt(v Value) (int64, bool) {
	// truncation toward zero for in-range values is documented resolution loss;
	// NaN, ±Inf and values outside int64's range have no counterpart and decline
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) || math.IsInf(f, 0) || f >= math.MaxInt64 || f < math.MinInt64 {
		return 0, false
	}
	return int64(f), true
}

func floatTypeAsFloat(v Value) (float64, bool) {
	return math.Float64frombits(v.Data), true
}

func floatTypeAsBool(v Value) (bool, bool) {
	// the conversion is a zero check (the bool -> int edge read backwards);
	// NaN is an error state and declines
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) {
		return false, false
	}
	return f != 0, true
}

// PURE by contract. A float in conversion context is a unix timestamp read as sec.frac: the integer
// part is seconds since epoch, the fraction is the sub-second part (the encoding Python's
// time.time() produces). Always UTC, like every other int-shaped conversion.
//
// LOSSY, unavoidably: float64 has ~15-16 significant digits and a present-day timestamp already
// spends 10 on the seconds, so the fraction survives to roughly microsecond precision and not
// exactly -- 1704067200.123 lands on 1704067200.122999906. Use decimal (exact to nanoseconds) or
// int.time_nano() when the sub-second part has to be right. NaN/Inf and out-of-int64-range values
// decline, which surfaces as time(x) -> undefined or the time(x, fallback) default.
func floatTypeAsTime(v Value) (time.Time, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return time.Time{}, false
	}
	sec := math.Trunc(f)
	if sec >= 9.223372036854776e18 || sec <= -9.223372036854776e18 {
		return time.Time{}, false
	}
	nsec := math.Round((f - sec) * 1e9)
	return time.Unix(int64(sec), int64(nsec)).UTC(), true
}

func floatTypeAsString(v Value) (string, bool) {
	return strconv.FormatFloat(math.Float64frombits(v.Data), 'f', -1, 64), true
}

func floatTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	f := math.Float64frombits(v.Data)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return dec128.NaN(state.NaN), false
	}
	return dec128.FromFloat64(f), true
}

// compareExactAndFloat compares an exact, always-finite rational value against a float64 f,
// implementing the resolved ordering policy for the whole numeric family (docs/types.md):
// any real value is less than +Inf and greater than -Inf; nothing is ordered against NaN. exact
// itself is never NaN/Inf here — it comes from int/bool/byte/rune, or an already-NaN-checked
// decimal (see compareDecimalAndFloat). Returns a normal -1/0/1 Cmp result, or the sentinel -2 for
// "not ordered" (f is NaN).
func compareExactAndFloat(exact *big.Rat, f float64) int {
	switch {
	case math.IsNaN(f):
		return -2
	case math.IsInf(f, 1):
		return -1
	case math.IsInf(f, -1):
		return 1
	default:
		return exact.Cmp(new(big.Rat).SetFloat64(f))
	}
}

// compareDecimalAndFloat is compareExactAndFloat's decimal-specific counterpart: unlike int/bool/
// byte/rune, decimal itself can be NaN, which carries its own total-order placement — the unique
// minimum, equal only to another NaN (mirroring dec128's own Compare/Equal convention, and float's
// eventual same-type NaN change — see docs/types.md's "Resolved: NaN / Inf policy") — rather
// than "ordering against it is always false."
func compareDecimalAndFloat(d *dec128.Dec128, f float64) int {
	if d.IsNaN() {
		if math.IsNaN(f) {
			return 0 // both "NaN" in their respective domains: equal in the total order
		}
		return -1 // decimal NaN is the unique minimum, less than any real float (finite or ±Inf)
	}
	return compareExactAndFloat(decimalToExactRat(d), f)
}

// compareFloatTotalOrder implements the resolved NaN total-order convention for float's own
// same-type comparisons (docs/types.md, "Resolved: NaN / Inf policy"): NaN is the unique
// minimum, sorting below even -Inf, and equal only to another NaN — matching dec128's own
// Compare/Equal convention, rather than IEEE-754 unordered semantics (which made array.sort() on a
// float array containing NaN silently non-deterministic — see core/array.go's sort comparator).
// Inf needs no special-casing: Go's native operators already order it correctly.
func compareFloatTotalOrder(a, b float64) int {
	aNaN, bNaN := math.IsNaN(a), math.IsNaN(b)
	switch {
	case aNaN && bNaN:
		return 0
	case aNaN:
		return -1
	case bNaN:
		return 1
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// exactOrderFloat implements < > <= >= for "exact OP f", given cmp from compareExactAndFloat/
// compareDecimalAndFloat (the -2 NaN sentinel always resolves to false, for every op).
func exactOrderFloat(cmp int, op token.Token) (Value, error) {
	if cmp == -2 {
		return BoolValue(false), nil
	}
	switch op {
	case token.Less:
		return BoolValue(cmp < 0), nil
	case token.Greater:
		return BoolValue(cmp > 0), nil
	case token.LessEq:
		return BoolValue(cmp <= 0), nil
	default: // token.GreaterEq
		return BoolValue(cmp >= 0), nil
	}
}

// floatOrderExact is exactOrderFloat's mirror, for "f OP exact" — float as the left operand instead
// of the right (float's own non-reflected branch, where v is float and other is the exact side).
func floatOrderExact(cmp int, op token.Token) (Value, error) {
	if cmp == -2 {
		return BoolValue(false), nil
	}
	switch op {
	case token.Less:
		return BoolValue(cmp > 0), nil
	case token.Greater:
		return BoolValue(cmp < 0), nil
	case token.LessEq:
		return BoolValue(cmp >= 0), nil
	default: // token.GreaterEq
		return BoolValue(cmp <= 0), nil
	}
}

func floatTypeEqual(v Value, other Value, final bool) bool {
	f := math.Float64frombits(v.Data)
	switch other.Type {
	case value.Float:
		return compareFloatTotalOrder(f, math.Float64frombits(other.Data)) == 0
	case value.Bool, value.Byte, value.Rune:
		i := int64(other.Data)
		return f == float64(i)
	case value.Int:
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false // int is never NaN/Inf
		}
		i := int64(other.Data)
		fr := new(big.Rat).SetFloat64(f)
		return fr.Cmp(new(big.Rat).SetInt64(i)) == 0
	case value.Decimal:
		d := (*dec128.Dec128)(other.Ptr)
		if d.IsNaN() {
			return math.IsNaN(f)
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return false
		}
		fr := new(big.Rat).SetFloat64(f)
		return fr.Cmp(decimalToExactRat(d)) == 0
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func floatTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Bool, value.Byte, value.Rune:
			switch op {
			case token.Less, token.Greater, token.LessEq, token.GreaterEq:
				l := int64(other.Data)
				r := math.Float64frombits(v.Data)
				cmp := compareExactAndFloat(new(big.Rat).SetInt64(l), r)
				return exactOrderFloat(cmp, op)
			}
		case value.Decimal:
			switch op {
			case token.Less, token.Greater, token.LessEq, token.GreaterEq:
				l := (*dec128.Dec128)(other.Ptr)
				r := math.Float64frombits(v.Data)
				cmp := compareDecimalAndFloat(l, r)
				return exactOrderFloat(cmp, op)
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Float:
		l := math.Float64frombits(v.Data)
		r := math.Float64frombits(other.Data)
		switch op {
		case token.Add:
			return floatArithResult(l + r)
		case token.Sub:
			return floatArithResult(l - r)
		case token.Mul:
			return floatArithResult(l * r)
		case token.Quo:
			return floatArithResult(l / r)
		case token.Rem:
			return floatArithResult(math.Mod(l, r))
		case token.Less, token.Greater, token.LessEq, token.GreaterEq:
			cmp := compareFloatTotalOrder(l, r)
			switch op {
			case token.Less:
				return BoolValue(cmp < 0), nil
			case token.Greater:
				return BoolValue(cmp > 0), nil
			case token.LessEq:
				return BoolValue(cmp <= 0), nil
			default:
				return BoolValue(cmp >= 0), nil
			}
		}

	case value.Int:
		l := math.Float64frombits(v.Data)
		i := int64(other.Data)
		r := float64(i)
		switch op {
		case token.Add:
			return floatArithResult(l + r)
		case token.Sub:
			return floatArithResult(l - r)
		case token.Mul:
			return floatArithResult(l * r)
		case token.Quo:
			return floatArithResult(l / r)
		case token.Rem:
			return floatArithResult(math.Mod(l, r))
		case token.Less, token.Greater, token.LessEq, token.GreaterEq:
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(i), l)
			return floatOrderExact(cmp, op)
		}

	case value.Bool, value.Byte, value.Rune:
		switch op {
		case token.Less, token.Greater, token.LessEq, token.GreaterEq:
			l := math.Float64frombits(v.Data)
			r := int64(other.Data)
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(r), l)
			return floatOrderExact(cmp, op)
		}

	case value.Decimal:
		switch op {
		case token.Less, token.Greater, token.LessEq, token.GreaterEq:
			l := math.Float64frombits(v.Data)
			r := (*dec128.Dec128)(other.Ptr)
			cmp := compareDecimalAndFloat(r, l)
			return floatOrderExact(cmp, op)
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract

// floatArithResult guards every float arithmetic result: no arithmetic may
// produce NaN or ±Inf — division by zero and overflow raise, matching int's
// behaviour. The sentinels stay representable (parses, the host) and keep
// their total-order comparison semantics; only arithmetic refuses them.
func floatArithResult(f float64) (Value, error) {
	if math.IsNaN(f) {
		return Undefined, errs.NewInvalidValueError("invalid float arithmetic (NaN result)")
	}
	if math.IsInf(f, 0) {
		return Undefined, errs.NewInvalidValueError("float overflow or division by zero")
	}
	return FloatValue(f), nil
}

func floatTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		f := math.Float64frombits(v.Data)
		return floatArithResult(-f)
	}

	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func floatTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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

	case "float":
		return convMember(name, floatTypeName, args, true, v)

	case "decimal":
		f := math.Float64frombits(v.Data)
		// a NaN decimal is an error state, never a produced value: NaN/Inf decline
		ok := !math.IsInf(f, 0) && !math.IsNaN(f)
		var d Value
		if ok {
			d = NewDecimalValue(dec128.FromFloat64(f))
		}
		return convMember(name, floatTypeName, args, ok, d)

	case "int":
		i, ok := v.AsInt()
		return convMember(name, floatTypeName, args, ok, IntValue(i))

	case "bool":
		b, ok := v.AsBool()
		return convMember(name, floatTypeName, args, ok, BoolValue(b))

	case "time":
		t, ok := v.AsTime()
		return convMember(name, floatTypeName, args, ok, NewTimeValue(t))

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return NewStringValue(s), nil

	case "runes":
		s, ok := v.AsString()
		return convMember(name, floatTypeName, args, ok, NewRunesValue([]rune(s), false))

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
		s, err := floatTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(math.IsNaN(math.Float64frombits(v.Data))), nil

	case "is_inf":
		// no sign argument: -Inf is x.is_inf() && x.sign() < 0
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(math.IsInf(math.Float64frombits(v.Data), 0)), nil

	case "is_zero":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(math.Float64frombits(v.Data) == 0), nil

	case "is_negative":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(math.Float64frombits(v.Data) < 0), nil

	case "is_positive":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(math.Float64frombits(v.Data) > 0), nil

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return FloatValue(math.Abs(math.Float64frombits(v.Data))), nil

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
		return Undefined, errs.NewInvalidMethodError(name, floatTypeName)
	}
}
