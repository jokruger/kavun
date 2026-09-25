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
	Name:         ConstHook(decimalTypeName),                                                          // PURE by contract
	String:       decimalTypeString,                                                                   // PURE by contract
	Format:       decimalTypeFormat,                                                                   // PURE by contract
	Interface:    func(v Value) any { return *(*dec128.Dec128)(v.Ptr) },                               // PURE by contract
	EncodeJSON:   func(v Value) ([]byte, error) { return (*dec128.Dec128)(v.Ptr).MarshalJSON() },      // PURE by contract
	EncodeBinary: decimalTypeEncodeBinary,                                                             // PURE by contract
	DecodeBinary: decimalTypeDecodeBinary,                                                             // IMPURE by contract (mutates target)
	IsTrue:       decimalTypeIsTrue,                                                                   // PURE by contract
	Equal:        decimalTypeEqual,                                                                    // PURE by contract
	BinaryOp:     decimalTypeBinaryOp,                                                                 // PURE by contract
	UnaryOp:      decimalTypeUnaryOp,                                                                  // PURE by contract
	Len:          ConstHook(int64(1)),                                                                 // PURE by contract
	MethodCall:   decimalTypeMethodCall,                                                               // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return (*dec128.Dec128)(v.Ptr).StringFixed(), true }, // PURE by contract
	AsInt:        decimalTypeAsInt,                                                                    // PURE by contract
	AsFloat:      decimalTypeAsFloat,                                                                  // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *(*dec128.Dec128)(v.Ptr), true },       // PURE by contract
	AsTime:       decimalTypeAsTime,                                                                   // PURE by contract
	AsBool:       decimalTypeAsBool,                                                                   // PURE by contract
	IsMethodPure: func(string) bool { return true },                                                   // All methods are expected to be pure.
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
	// UnmarshalBinary accepts a NaN payload and reports no error, so the error alone is not enough of a check.
	// No operation in the language can produce a NaN decimal; letting the codec introduce one would put a value
	// into a script that every other path is built to rule out.
	if d.IsNaN() {
		return fmt.Errorf("failed to decode decimal: encoded value is not a number")
	}
	*v = NewDecimalValue(d)
	return nil
}

// The repr carries the SCALE: 1.50d and 1.5d are two different constants (they hash apart in the static pool),
// so a reading that collapsed them would not round-trip.
func decimalTypeString(v Value) string {
	o := (*dec128.Dec128)(v.Ptr)
	if o.IsNaN() {
		return `decimal("NaN")`
	}
	return o.StringFixed() + "d"
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
	case 0, 's':
		// A decimal's scale is part of the value — "1.50" states cents precision, "1.5" does not — so the
		// default rendering keeps it, matching json.encode. '!' asks for the trimmed reading; 's' is the
		// explicit spelling of this same default, kept so existing format strings still say what they mean.
		if sp.Bare {
			raw = abs.String()
		} else {
			raw = abs.StringFixed()
		}

	case 'f', 'F':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = decimalFixedString(abs, prec)

	case '%':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		raw = decimalFixedString(abs.Mul(dec128.FromInt64(100)), prec) + "%"

	case 'e', 'E':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		sig, exp := decimalSigDigits(abs, prec+1)
		raw = decimalSciFromSig(sig, exp, verb == 'E')

	case 'g', 'G':
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		// "shortest" means the shorter of the two EXACT readings, fixed and scientific — not the shortest
		// approximation. 1e-19 stays "1e-19" rather than spelling out nineteen zeros, and a value that reads
		// better in full stays in full.
		want := 0 // 0 = every digit the value has
		if sp.HasPrec {
			want = max(int(sp.Precision), 1) // a 'g' precision counts SIGNIFICANT digits, as it does in Go
		}
		sig, exp := decimalSigDigits(abs, want)
		sig = strings.TrimRight(sig, "0")
		if sig == "" {
			sig = "0"
		}
		fixed := decimalFixedFromSig(sig, exp)
		sci := decimalSciFromSig(sig, exp, verb == 'G')
		raw = fixed
		if len(sci) < len(fixed) {
			raw = sci
		}

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

// decimalSigDigits splits a NON-NEGATIVE decimal into its significant decimal digits and the exponent of the
// first one: the value is sig[0] . sig[1:] x 10^exp. When want > 0 exactly that many digits are returned,
// rounded half-away-from-zero; want <= 0 keeps every digit the value has, with trailing zeros trimmed.
//
// The digits come from the decimal's own coefficient. Nothing here goes through float64, which is the whole
// point: a float64 round trip invents digits past the 17th significant place.
//
// PURE by contract.
func decimalSigDigits(d dec128.Dec128, want int) (string, int) {
	text := d.StringFixed()
	intPart, fracPart := text, ""
	if i := strings.IndexByte(text, '.'); i >= 0 {
		intPart, fracPart = text[:i], text[i+1:]
	}
	digits := intPart + fracPart

	first := -1
	for i := 0; i < len(digits); i++ {
		if digits[i] != '0' {
			first = i
			break
		}
	}
	if first < 0 { // the value is zero: no significant digit to anchor an exponent on
		return strings.Repeat("0", max(want, 1)), 0
	}

	exp := len(intPart) - 1 - first
	sig := digits[first:]

	if want <= 0 {
		sig = strings.TrimRight(sig, "0")
		if sig == "" {
			sig = "0"
		}
		return sig, exp
	}
	if len(sig) <= want {
		return sig + strings.Repeat("0", want-len(sig)), exp
	}

	head := []byte(sig[:want])
	if sig[want] >= '5' {
		i := len(head) - 1
		for ; i >= 0; i-- {
			if head[i] != '9' {
				head[i]++
				break
			}
			head[i] = '0'
		}
		if i < 0 {
			// the carry ran off the front (999 -> 1000): one more place, same digit count
			head = append([]byte{'1'}, head[:want-1]...)
			exp++
		}
	}
	return string(head), exp
}

// decimalSciFromSig renders significant digits + exponent as d.dddde±XX, with the exponent zero-padded to at
// least two digits as strconv.FormatFloat does.
//
// PURE by contract.
func decimalSciFromSig(sig string, exp int, upper bool) string {
	var b strings.Builder
	b.WriteByte(sig[0])
	if len(sig) > 1 {
		b.WriteByte('.')
		b.WriteString(sig[1:])
	}
	if upper {
		b.WriteByte('E')
	} else {
		b.WriteByte('e')
	}
	if exp < 0 {
		b.WriteByte('-')
		exp = -exp
	} else {
		b.WriteByte('+')
	}
	if exp < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.Itoa(exp))
	return b.String()
}

// decimalFixedFromSig renders significant digits + exponent in plain positional notation.
//
// PURE by contract.
func decimalFixedFromSig(sig string, exp int) string {
	if exp >= 0 {
		if len(sig) <= exp+1 {
			return sig + strings.Repeat("0", exp+1-len(sig))
		}
		return sig[:exp+1] + "." + sig[exp+1:]
	}
	return "0." + strings.Repeat("0", -exp-1) + sig
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

// decimalResult is the SINGLE guard on producing a decimal from a dec128 computation.
//
// dec128 signals every failure the same way: it answers NaN. That is an ERROR STATE, not a value — unlike an IEEE
// NaN it even compares equal to itself, so it behaves as an ordinary number everywhere downstream and a wrong
// answer propagates silently through a whole rules script. Kavun has one failure mode, so a NaN result raises
// here instead of travelling on, matching what float arithmetic already does.
//
// PURE by contract.
func decimalResult(name string, d dec128.Dec128) (Value, error) {
	if !d.IsNaN() {
		return NewDecimalValue(d), nil
	}
	if details := d.ErrorDetails(); details != nil {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) %s", name, details.Error()))
	}
	return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) decimal result is not a number", name))
}

// decimalArithResult is decimalResult for the binary operators. A zero divisor is reported as division_by_zero,
// the same kind `int` answers for `1 / 0`, rather than as a generic bad value.
//
// PURE by contract.
func decimalArithResult(d dec128.Dec128, op token.Token, rZero bool) (Value, error) {
	if d.IsNaN() && rZero && (op == token.Quo || op == token.Rem) {
		return Undefined, errs.NewDivisionByZeroError()
	}
	return decimalResult(op.String(), d)
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
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case token.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case token.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case token.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case token.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
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
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case token.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case token.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case token.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case token.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
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
		return decimalResult("-", o.Neg())

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

// decimalScaleArg reads a target-scale argument: a whole number in 0..dec128.MaxScale. It goes through
// parseIntArg, so a lossless float or decimal spelling is accepted (rescale(2.0)) and a fractional one raises
// rather than silently truncating (rescale(2.5)), exactly as every other count-shaped argument behaves.
//
// PURE by contract.
func decimalScaleArg(name, pos string, a Value) (uint8, error) {
	scale, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if scale < 0 || scale > int64(dec128.MaxScale) {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) scale must be between 0 and %d", name, dec128.MaxScale))
	}
	return uint8(scale), nil
}

// decimalOperandArg reads the other operand of a two-operand member. It accepts exactly what the arithmetic
// OPERATORS accept — decimal and int — so div_round_bank(y, n) and `/` agree on their domain. float is refused
// here for the same reason `decimal / float` is refused: there is no automatic winner between the two
// representations.
//
// PURE by contract.
func decimalOperandArg(name, pos string, a Value) (dec128.Dec128, error) {
	switch a.Type {
	case value.Decimal:
		return *(*dec128.Dec128)(a.Ptr), nil
	case value.Int:
		return dec128.FromInt64(int64(a.Data)), nil
	}
	return dec128.Dec128{}, errs.NewInvalidArgumentTypeError(name, pos, "decimal or int", a.TypeName())
}

// decimalRoundingModes maps a member name's POLICY SUFFIX to the dec128 rounding mode it selects. The seven
// policies are the same set the round_* family already exposes, so a reader who knows round_bank(n) knows
// rescale_bank(n) and div_round_bank(y, n) too — the policy is always spelled in the name, never passed as a
// mode argument.
var decimalRoundingModes = map[string]dec128.RoundingMode{
	"down":                dec128.ROUND_DOWN,
	"up":                  dec128.ROUND_UP,
	"toward_zero":         dec128.ROUND_TOWARD_ZERO,
	"away_from_zero":      dec128.ROUND_AWAY_FROM_ZERO,
	"half_toward_zero":    dec128.ROUND_HALF_TOWARD_ZERO,
	"half_away_from_zero": dec128.ROUND_HALF_AWAY_FROM_ZERO,
	"bank":                dec128.ROUND_BANK,
}

// decimalRoundingMode resolves the mode a member name selects, given the family prefix its case label carries.
// The case labels enumerate exactly the seven suffixes above, so the lookup cannot miss; it is checked anyway
// because a zero value here would be ROUND_TOWARD_ZERO — a silently wrong answer rather than a loud one.
//
// PURE by contract.
func decimalRoundingMode(name, prefix string) (dec128.RoundingMode, error) {
	mode, ok := decimalRoundingModes[strings.TrimPrefix(name, prefix)]
	if !ok {
		return 0, errs.NewInternalError(fmt.Sprintf("decimal: no rounding mode for member %q", name))
	}
	return mode, nil
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
		// scale-preserving, like every other rendering; canonical() is the value-level way to drop the zeros
		return NewStringValue(o.StringFixed()), nil

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
			return Undefined, errs.FromFormatSpecError(name, err)
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
			return NewErrorValue(NewStringValue(details.Error()), KindUser, errs.CategoryUser, false), nil
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
		scale, err := decimalScaleArg(name, "scale", args[0])
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.ToScale(scale))

	case "canonical":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.Canonical())

	case "next_up":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.NextUp())

	case "next_down":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.NextDown())

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.Abs())

	case "negate":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.Neg())

	case "sqrt":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return decimalResult(name, o.Sqrt())

	case "pow":
		// Integer exponent only. A negative one is the reciprocal (2d.pow(-1) is 0.5d), so a zero base with a
		// negative exponent is a division by zero and is reported as one rather than as a bad value.
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		exp, err := parseIntArg(name, "exponent", args[0])
		if err != nil {
			return Undefined, err
		}
		if exp < 0 && o.IsZero() {
			return Undefined, errs.NewDivisionByZeroError()
		}
		return decimalResult(name, o.PowInt64(exp))

	case "round_down", "round_up", "round_toward_zero", "round_away_from_zero",
		"round_half_toward_zero", "round_half_away_from_zero", "round_bank":
		// Round to AT MOST the given scale: a value that is already shorter is answered unchanged. The policy
		// lives in the name; see rescale_* for the "exactly n places" spelling.
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, err := decimalScaleArg(name, "scale", args[0])
		if err != nil {
			return Undefined, err
		}
		mode, err := decimalRoundingMode(name, "round_")
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.Round(scale, mode))

	case "rescale_down", "rescale_up", "rescale_toward_zero", "rescale_away_from_zero",
		"rescale_half_toward_zero", "rescale_half_away_from_zero", "rescale_bank":
		// Exactly the given scale, rounding on the way down: the money spelling. rescale() pads and truncates,
		// round_*() stops at "at most n places" and leaves a shorter value alone — only this family answers a
		// value that is guaranteed to carry n fractional digits.
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, err := decimalScaleArg(name, "scale", args[0])
		if err != nil {
			return Undefined, err
		}
		mode, err := decimalRoundingMode(name, "rescale_")
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.RescaleRound(scale, mode))

	case "div_round_down", "div_round_up", "div_round_toward_zero", "div_round_away_from_zero",
		"div_round_half_toward_zero", "div_round_half_away_from_zero", "div_round_bank":
		// Divide straight to the target scale. The difference from `(x / y).round_bank(2)` that always shows is
		// the SCALE: round_* stops at "at most n places", so 10d / 4d answers 2.5 at scale 1, while
		// div_round_bank(4d, 2) answers 2.50 at scale 2 — a money result that carries its cents.
		//
		// The rounding decision is also made against the exact quotient rather than an already-truncated
		// scale-19 intermediate. That can change the last digit in principle; measured, it is vanishingly rare
		// (no divergence in 4M randomized operand/scale/mode cases), so it is not the reason to reach for this.
		if len(args) != 2 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
		}
		other, err := decimalOperandArg(name, "first", args[0])
		if err != nil {
			return Undefined, err
		}
		scale, err := decimalScaleArg(name, "second", args[1])
		if err != nil {
			return Undefined, err
		}
		mode, err := decimalRoundingMode(name, "div_round_")
		if err != nil {
			return Undefined, err
		}
		if other.IsZero() {
			return Undefined, errs.NewDivisionByZeroError()
		}
		return decimalResult(name, o.DivRound(other, scale, mode))

	case "mul_round_down", "mul_round_up", "mul_round_toward_zero", "mul_round_away_from_zero",
		"mul_round_half_toward_zero", "mul_round_half_away_from_zero", "mul_round_bank":
		if len(args) != 2 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
		}
		other, err := decimalOperandArg(name, "first", args[0])
		if err != nil {
			return Undefined, err
		}
		scale, err := decimalScaleArg(name, "second", args[1])
		if err != nil {
			return Undefined, err
		}
		mode, err := decimalRoundingMode(name, "mul_round_")
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.MulRound(other, scale, mode))

	case "sqrt_round_down", "sqrt_round_up", "sqrt_round_toward_zero", "sqrt_round_away_from_zero",
		"sqrt_round_half_toward_zero", "sqrt_round_half_away_from_zero", "sqrt_round_bank":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, err := decimalScaleArg(name, "scale", args[0])
		if err != nil {
			return Undefined, err
		}
		mode, err := decimalRoundingMode(name, "sqrt_round_")
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.SqrtRound(scale, mode))

	case "trunc":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		scale, err := decimalScaleArg(name, "scale", args[0])
		if err != nil {
			return Undefined, err
		}
		return decimalResult(name, o.Trunc(scale))

	default:
		return Undefined, errs.NewInvalidMethodError(name, decimalTypeName)
	}
}
