package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/jokruger/kavun/core/member/members"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
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
	UnaryOp:      decimalTypeUnaryOp,                                                                  // PURE by contract
	BinaryOp:     decimalTypeBinaryOp,                                                                 // PURE by contract
	Len:          ConstHook(int64(1)),                                                                 // PURE by contract
	AsString:     func(v Value) (string, bool) { return (*dec128.Dec128)(v.Ptr).StringFixed(), true }, // PURE by contract
	AsInt:        decimalTypeAsInt,                                                                    // PURE by contract
	AsFloat:      decimalTypeAsFloat,                                                                  // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return *(*dec128.Dec128)(v.Ptr), true },       // PURE by contract
	AsTime:       decimalTypeAsTime,                                                                   // PURE by contract
	AsBool:       decimalTypeAsBool,                                                                   // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:            {Fn: memberIsTrue, Pure: true},
		members.String:            {Fn: decimalString, Pure: true},
		members.Format:            {Fn: memberFormat, Pure: true},
		members.Copy:              {Fn: memberSelf, Pure: true},
		members.Freeze:            {Fn: memberSelf, Pure: true},
		members.Runes:             {Fn: decimalRunes, Pure: true},
		members.Int:               {Fn: decimalInt, Pure: true},
		members.Bool:              {Fn: decimalBool, Pure: true},
		members.Float:             {Fn: decimalFloat, Pure: true},
		members.Time:              {Fn: decimalTime, Pure: true},
		members.Decimal:           {Fn: decimalDecimal, Pure: true},
		members.Split:             {Fn: decimalSplit, Pure: true},
		members.Abs:               {Fn: decimalAbs, Pure: true},
		members.Sign:              {Fn: decimalSign, Pure: true},
		members.IsZero:            {Fn: decimalIsZero, Pure: true},
		members.IsPositive:        {Fn: decimalIsPositive, Pure: true},
		members.IsNegative:        {Fn: decimalIsNegative, Pure: true},
		members.IsNaN:             {Fn: decimalIsNaN, Pure: true},
		members.IsInf:             {Fn: decimalIsInf, Pure: true},
		members.Clamp:             {Fn: decimalClamp, Pure: true},
		members.Sqrt:              {Fn: decimalSqrt, Pure: true},
		members.Pow:               {Fn: decimalPow, Pure: true},
		members.CopySign:          {Fn: decimalCopySign, Pure: true},
		members.Round:             {Fn: decimalRound, Pure: true},
		members.RoundHalfEven:     {Fn: decimalRoundHalfEven, Pure: true},
		members.RoundHalfUp:       {Fn: decimalRoundHalfUp, Pure: true},
		members.RoundHalfDown:     {Fn: decimalRoundHalfDown, Pure: true},
		members.RoundUp:           {Fn: decimalRoundUp, Pure: true},
		members.RoundDown:         {Fn: decimalRoundDown, Pure: true},
		members.RoundCeiling:      {Fn: decimalRoundCeiling, Pure: true},
		members.RoundFloor:        {Fn: decimalRoundFloor, Pure: true},
		members.RoundSignificant:  {Fn: decimalRoundSignificant, Pure: true},
		members.RoundToMultiple:   {Fn: decimalRoundToMultiple, Pure: true},
		members.Rescale:           {Fn: decimalRescale, Pure: true},
		members.Scale:             {Fn: decimalScale, Pure: true},
		members.ScaleByPow10:      {Fn: decimalScaleByPow10, Pure: true},
		members.Canonical:         {Fn: decimalCanonical, Pure: true},
		members.IntegerDigits:     {Fn: decimalIntegerDigits, Pure: true},
		members.SignificantDigits: {Fn: decimalSignificantDigits, Pure: true},
		members.IsInteger:         {Fn: decimalIsInteger, Pure: true},
		members.Negate:            {Fn: decimalNegate, Pure: true},
		members.NextUp:            {Fn: decimalNextUp, Pure: true},
		members.NextDown:          {Fn: decimalNextDown, Pure: true},
		members.CanFit:            {Fn: decimalCanFit, Pure: true},
		members.QuoRem:            {Fn: decimalQuoRem, Pure: true},
		members.DivRound:          {Fn: decimalDivRound, Pure: true},
		members.MulRound:          {Fn: decimalMulRound, Pure: true},
		members.MulAddRound:       {Fn: decimalMulAddRound, Pure: true},
		members.MulDivRound:       {Fn: decimalMulDivRound, Pure: true},
		members.MulPercentRound:   {Fn: decimalMulPercentRound, Pure: true},
		members.PowRound:          {Fn: decimalPowRound, Pure: true},
		members.PowRationalRound:  {Fn: decimalPowRationalRound, Pure: true},
		members.SqrtRound:         {Fn: decimalSqrtRound, Pure: true},
		members.NthRootRound:      {Fn: decimalNthRootRound, Pure: true},
		members.ExpRound:          {Fn: decimalExpRound, Pure: true},
		members.LnRound:           {Fn: decimalLnRound, Pure: true},
		members.Log2Round:         {Fn: decimalLog2Round, Pure: true},
		members.Log10Round:        {Fn: decimalLog10Round, Pure: true},
		members.Allocate:          {Fn: decimalAllocate, Pure: true},
		members.AllocateResidual:  {Fn: decimalAllocateResidual, Pure: true},
		members.SplitResidual:     {Fn: decimalSplitResidual, Pure: true},
		members.ErrorDetails:      {Fn: decimalErrorDetails, Pure: true},
	},
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
//
// A precision past MaxScale is honoured by padding: a decimal never has more than MaxScale places, so every digit
// beyond is a zero and writing it is exact. Only the rounding step is bounded by the type.
func decimalFixedString(d dec128.Dec128, prec int) string {
	if prec < 0 {
		return d.String()
	}
	rounded := d.RoundHalfAwayFromZero(uint8(min(prec, int(dec128.MaxScale))))
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
// here instead of travelling on, matching what float arithmetic already does. A NaN that carries dec128's
// division-by-zero reason raises division_by_zero, the kind `int` answers for `1 / 0`.
//
// PURE by contract.
func decimalResult(name string, d dec128.Dec128) (Value, error) {
	if !d.IsNaN() {
		return NewDecimalValue(d), nil
	}
	if errors.Is(d.ErrorDetails(), state.DivisionByZero.Error()) {
		return Undefined, errs.NewDivisionByZeroError()
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
	if d.IsNaN() && rZero && (op == tokens.Quo || op == tokens.Rem) {
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
			case tokens.Less:
				return BoolValue(l.LessThan(*r)), nil
			case tokens.Greater:
				return BoolValue(l.GreaterThan(*r)), nil
			case tokens.LessEq:
				return BoolValue(l.LessThanOrEqual(*r)), nil
			case tokens.GreaterEq:
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
		case tokens.Add:
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case tokens.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case tokens.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case tokens.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case tokens.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Int:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case tokens.Add:
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case tokens.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case tokens.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case tokens.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case tokens.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Bool, value.Byte, value.Rune:
		r := dec128.FromInt64(int64(other.Data))
		switch op {
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func decimalTypeUnaryOp(v Value, op token.Token) (Value, error) {
	o := (*dec128.Dec128)(v.Ptr)

	switch op {
	case tokens.Sub:
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
// OPERATORS accept — decimal and int — so div_round(y, n, mode) and `/` agree on their domain. float is refused
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

// decimalRoundingModes maps a rounding-mode NAME to the dec128 mode it selects. The names are Python's `decimal`
// module's, lowercased and without the ROUND_ prefix, because that is the reference a reader brings: `down` is
// toward zero and `up` away from it, while `floor` / `ceiling` are the directions toward −∞ / +∞. A mode is a
// string argument because in a real application it is configuration, not code; the round_<mode>(n) twins spell
// the same names as suffixes for the common case where it is fixed.
//
// dec128's ROUND_NAN has no name here on purpose: refusing a loss is rescale(n)'s job, and it is meaningless for
// the division-shaped operations, which lose digits by nature.
var decimalRoundingModes = map[string]dec128.RoundingMode{
	"ceiling":   dec128.ROUND_UP,
	"floor":     dec128.ROUND_DOWN,
	"down":      dec128.ROUND_TOWARD_ZERO,
	"up":        dec128.ROUND_AWAY_FROM_ZERO,
	"half_down": dec128.ROUND_HALF_TOWARD_ZERO,
	"half_up":   dec128.ROUND_HALF_AWAY_FROM_ZERO,
	"half_even": dec128.ROUND_BANK,
}

// decimalRoundingModeNames lists the accepted names in the order the docs present them, for the error message.
const decimalRoundingModeNames = "ceiling, floor, down, up, half_down, half_up, half_even"

// decimalModeArg reads a rounding-mode argument: a string naming one of the seven modes, exactly. An unknown name
// raises and lists the valid ones — a mode read from configuration must fail loudly, never fall back to a default.
//
// PURE by contract.
func decimalModeArg(name, pos string, a Value) (dec128.RoundingMode, error) {
	if a.Type != value.String {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "string", a.TypeName())
	}
	s, _ := a.AsString()
	mode, ok := decimalRoundingModes[s]
	if !ok {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown rounding mode %q, expected one of: %s", name, s, decimalRoundingModeNames))
	}
	return mode, nil
}

// decimalPlacesArg reads round()'s place count: 0..MaxScale fractional places, or a negative count that rounds to
// tens, hundreds, … down to the 10^38 place, past which no coefficient reaches.
//
// PURE by contract.
func decimalPlacesArg(name, pos string, a Value) (int, error) {
	n, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if n < -38 || n > int64(dec128.MaxScale) {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) places must be between -38 and %d", name, dec128.MaxScale))
	}
	return int(n), nil
}

// decimalRoundTo is round(n, mode): EXACTLY n places for n >= 0 (1.5 at 2 is 1.50, as Python's round(Decimal, n)
// answers), and a whole number at scale 0 for a negative n (1234.5 at -2 is 1200).
//
// PURE by contract.
func decimalRoundTo(d dec128.Dec128, places int, mode dec128.RoundingMode) dec128.Dec128 {
	if places >= 0 {
		return d.RescaleRound(uint8(places), mode)
	}
	return d.RoundToPlaces(int8(places), mode)
}

// decimalCountArg reads a strictly positive int-shaped argument (a root degree, a share count, a digit count).
//
// PURE by contract.
func decimalCountArg(name, pos string, a Value) (int64, error) {
	n, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) argument %s must be positive, got %d", name, pos, n))
	}
	return n, nil
}

// decimalMaxRootDegree is dec128's own bound on a root degree (and on a reduced pow_rational exponent), repeated
// here so an over-large degree raises with a message that names the limit rather than a generic domain error. It
// is a measured cost ceiling — 40 years of daily rests fit, a century does not.
const decimalMaxRootDegree = 16384

// decimalScaleModeArgs reads the trailing (scale, mode) pair every *_round member ends with, starting at args[i].
//
// PURE by contract.
func decimalScaleModeArgs(name string, args []Value, i int) (uint8, dec128.RoundingMode, error) {
	scale, err := decimalScaleArg(name, "scale", args[i])
	if err != nil {
		return 0, 0, err
	}
	mode, err := decimalModeArg(name, "mode", args[i+1])
	if err != nil {
		return 0, 0, err
	}
	return scale, mode, nil
}

// decimalRatiosArg reads allocate()'s ratios: a non-empty array of decimal or int, none negative and not all zero.
// These are checked here, before dec128 is called, because its Allocate answers a bare `false` with no reason.
//
// PURE by contract.
func decimalRatiosArg(name string, a Value) ([]dec128.Dec128, error) {
	if a.Type != value.Array {
		return nil, errs.NewInvalidArgumentTypeError(name, "ratios", "array", a.TypeName())
	}
	elems := (*Array)(a.Ptr).Elements
	if len(elems) == 0 {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratios must not be empty", name))
	}
	if len(elems) > MaxSequenceLen {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) result would be %d elements, past the %d limit", name, len(elems), MaxSequenceLen))
	}
	ratios := make([]dec128.Dec128, len(elems))
	allZero := true
	for i, e := range elems {
		r, err := decimalOperandArg(name, "ratios", e)
		if err != nil {
			return nil, err
		}
		if r.IsNegative() {
			return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratio at index %d is negative", name, i))
		}
		if !r.IsZero() {
			allZero = false
		}
		ratios[i] = r
	}
	if allZero {
		return nil, errs.NewInvalidValueError(fmt.Sprintf("(%s) ratios must not all be zero", name))
	}
	return ratios, nil
}

// decimalSharesCountArg reads split()'s share count: positive, and within the sequence-allocation ceiling, because
// the count sizes the array that dec128 builds.
//
// PURE by contract.
func decimalSharesCountArg(name string, a Value) (int, error) {
	n, err := decimalCountArg(name, "count", a)
	if err != nil {
		return 0, err
	}
	if n > MaxSequenceLen {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) result would be %d elements, past the %d limit", name, n, MaxSequenceLen))
	}
	return int(n), nil
}

// decimalResidualIndexArg reads the index of the share that absorbs the rounding. Like an array index, a negative
// one counts from the end (-1 is the last share).
//
// PURE by contract.
func decimalResidualIndexArg(name string, a Value, count int) (int, error) {
	idx, err := parseIntArg(name, "index", a)
	if err != nil {
		return 0, err
	}
	i := idx
	if i < 0 {
		i += int64(count)
	}
	if i < 0 || i >= int64(count) {
		return 0, errs.NewIndexOutOfBoundsError(name, int(idx), count-1)
	}
	return int(i), nil
}

// decimalShares finishes allocate/split: dec128's `ok == false` is turned into a raise, naming the one cause the
// argument checks cannot rule out in advance cheaply (the amount's own scale), and otherwise the representation
// limit. The shares come back as a new, mutable array.
//
// PURE by contract.
func decimalShares(name string, d dec128.Dec128, scale uint8, shares []dec128.Dec128, ok bool) (Value, error) {
	if !ok {
		if scale < d.Scale() {
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf(
				"(%s) scale %d is below the amount's own scale %d; round the amount first", name, scale, d.Scale()))
		}
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the shares cannot be represented at scale %d", name, scale))
	}
	out := make([]Value, len(shares))
	for i, s := range shares {
		out[i] = NewDecimalValue(s)
	}
	return NewArrayValue(out, false), nil
}

// The decimal argument rules, exported for the modules that compute in decimal (fin): one vocabulary, one set of
// messages. DecimalOperandArg accepts decimal or int (float raises); ScaleModeArgs reads the trailing
// (scale, mode) pair at args[i]; DecimalResult turns a NaN result into its raise (division_by_zero or
// invalid_value).

// PURE by contract.
func DecimalOperandArg(name, pos string, a Value) (dec128.Dec128, error) {
	return decimalOperandArg(name, pos, a)
}

// PURE by contract.
func ScaleModeArgs(name string, args []Value, i int) (uint8, dec128.RoundingMode, error) {
	return decimalScaleModeArgs(name, args, i)
}

// PURE by contract.
func DecimalResult(name string, d dec128.Dec128) (Value, error) {
	return decimalResult(name, d)
}
