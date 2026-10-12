package core

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// float's members: one function per entry of TypeFloat.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func floatFloat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return ConvMember(id.String(), floatTypeName, args, true, v)
}

// PURE by contract
func floatDecimal(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	f := math.Float64frombits(v.Data)
	// a NaN decimal is an error state, never a produced value: NaN/Inf decline, and so does a finite float
	// too large for dec128's 128-bit coefficient — FromFloat64 signals that overflow as NaN too
	ok := !math.IsInf(f, 0) && !math.IsNaN(f)
	var d Value
	if ok {
		dd := dec128.FromFloat64(f)
		if ok = !dd.IsNaN(); ok {
			d = NewDecimalValue(dd)
		}
	}
	return ConvMember(id.String(), floatTypeName, args, ok, d)
}

// PURE by contract
func floatInt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	i, ok := v.AsInt()
	return ConvMember(id.String(), floatTypeName, args, ok, IntValue(i))
}

// PURE by contract
func floatBool(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	b, ok := v.AsBool()
	return ConvMember(id.String(), floatTypeName, args, ok, BoolValue(b))
}

// PURE by contract
func floatTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	t, ok := v.AsTime()
	return ConvMember(id.String(), floatTypeName, args, ok, NewTimeValue(t))
}

// PURE by contract
func floatString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	s, _ := v.AsString()
	return NewStringValue(s), nil
}

// PURE by contract
func floatRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	s, ok := v.AsString()
	return ConvMember(id.String(), floatTypeName, args, ok, NewRunesValue(DecodeText(s), false))
}

// PURE by contract
func floatIsNaN(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.IsNaN(math.Float64frombits(v.Data))), nil
}

// PURE by contract
func floatIsInf(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// no sign argument: -Inf is x.is_inf() && x.sign() < 0
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.IsInf(math.Float64frombits(v.Data), 0)), nil
}

// PURE by contract
func floatIsZero(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) == 0), nil
}

// PURE by contract
func floatIsNegative(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) < 0), nil
}

// PURE by contract
func floatIsPositive(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(math.Float64frombits(v.Data) > 0), nil
}

// PURE by contract
func floatAbs(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return FloatValue(math.Abs(math.Float64frombits(v.Data))), nil
}

// PURE by contract
func floatSign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
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
}

// --- numeric members: the readings decimal has, on a float, in table order. No member may produce NaN or ±Inf
// (floatResult), exactly as no float operator may. The rounding members apply decimal's contract — its seven modes,
// through the shared roundingMode — to the float's shortest decimal text, the digits string() prints, so
// 2.675.round(2, "half_up") is 2.68. They round the text itself (floatRoundKeep), so no magnitude is out of range.

// PURE by contract
func floatClamp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	lo, err := floatOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	hi, err := floatOperandArg(name, "second", args[1])
	if err != nil {
		return Undefined, err
	}
	if lo > hi {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) lower bound %s is above upper bound %s", name, FloatValue(lo).String(), FloatValue(hi).String()))
	}
	return floatResult(name, math.Min(math.Max(math.Float64frombits(v.Data), lo), hi))
}

// PURE by contract
func floatSqrt(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return floatResult(name, math.Sqrt(math.Float64frombits(v.Data)))
}

// PURE by contract
func floatPow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	x := math.Float64frombits(v.Data)
	y, err := floatOperandArg(name, "exponent", args[0])
	if err != nil {
		return Undefined, err
	}
	if x == 0 && y < 0 { // the reciprocal of zero, reported as decimal reports it
		return Undefined, errs.NewDivisionByZeroError()
	}
	return floatResult(name, math.Pow(x, y))
}

// PURE by contract
func floatCopySign(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sign, err := floatOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	return floatResult(name, math.Copysign(math.Float64frombits(v.Data), sign))
}

// PURE by contract
func floatRound(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	places, err := floatPlacesArg(name, "places", args[0])
	if err != nil {
		return Undefined, err
	}
	mode, err := roundingModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	return floatRoundPlaces(name, v, places, mode)
}

// the fixed-mode twins of round(n, mode), each naming its mode through roundingModes

// PURE by contract
func floatLn(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatUnaryMath(v, id, args, math.Log)
}

// PURE by contract
func floatRoundHalfEven(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "half_even")
}

// PURE by contract
func floatRoundHalfUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "half_up")
}

// PURE by contract
func floatRoundHalfDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "half_down")
}

// PURE by contract
func floatRoundUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "up")
}

// PURE by contract
func floatRoundDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "down")
}

// PURE by contract
func floatRoundCeiling(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "ceiling")
}

// PURE by contract
func floatRoundFloor(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatRoundFixed(v, id, args, "floor")
}

// floatRoundFixed is round_<mode>(places): round(places, mode) with the mode fixed by the member.
func floatRoundFixed(v Value, id member.ID, args []Value, mode string) (Value, error) {
	name := id.String()
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	places, err := floatPlacesArg(name, "places", args[0])
	if err != nil {
		return Undefined, err
	}
	return floatRoundPlaces(name, v, places, roundingModes[mode])
}

// PURE by contract
func floatRoundSignificant(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// at most k significant digits; a shorter value is answered unchanged
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	digits, err := decimalCountArg(name, "digits", args[0])
	if err != nil {
		return Undefined, err
	}
	mode, err := roundingModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	f, err := floatRoundable(name, v)
	if err != nil {
		return Undefined, err
	}
	return floatResult(name, floatRoundKeep(f, mode, func(int) int64 { return digits }))
}

// PURE by contract
func floatRoundToMultiple(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// the nearest multiple of m: 0.05 cash rounding, "the next whole 10" — exact, over the shortest texts of the
	// value and of m (a multiple is a division, so this one rounds a big.Rat quotient)
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	f, err := floatRoundable(name, v)
	if err != nil {
		return Undefined, err
	}
	m, err := floatOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	if !(m > 0) || math.IsInf(m, 0) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the multiple must be positive, got %s", name, FloatValue(m).String()))
	}
	mode, err := roundingModeArg(name, "mode", args[1])
	if err != nil {
		return Undefined, err
	}
	x, mr := floatShortestRat(f), floatShortestRat(m)
	q := ratRoundToInt(new(big.Rat).Quo(x, mr), mode)
	r, _ := new(big.Rat).Mul(new(big.Rat).SetInt(q), mr).Float64()
	return floatResult(name, r)
}

// PURE by contract
func floatIsInteger(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	f := math.Float64frombits(v.Data)
	return BoolValue(!math.IsInf(f, 0) && f == math.Trunc(f)), nil // NaN fails the comparison: not an integer
}

// PURE by contract
func floatNegate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return floatResult(name, -math.Float64frombits(v.Data))
}

// PURE by contract
func floatNextUp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return floatResult(name, math.Nextafter(math.Float64frombits(v.Data), math.Inf(1)))
}

// PURE by contract
func floatNextDown(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return floatResult(name, math.Nextafter(math.Float64frombits(v.Data), math.Inf(-1)))
}

// PURE by contract
func floatQuoRem(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// [quotient, remainder]: the quotient truncated toward zero, the remainder with the receiver's sign — decimal's
	// reading; q*y + r == x up to float rounding of the quotient
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	x := math.Float64frombits(v.Data)
	y, err := floatOperandArg(name, "first", args[0])
	if err != nil {
		return Undefined, err
	}
	if y == 0 {
		return Undefined, errs.NewDivisionByZeroError()
	}
	r := math.Mod(x, y)
	q, err := floatResult(name, math.Trunc((x-r)/y))
	if err != nil {
		return Undefined, err
	}
	rv, err := floatResult(name, r)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue([]Value{q, rv}, false), nil
}

// PURE by contract
func floatExp(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatUnaryMath(v, id, args, math.Exp)
}

// PURE by contract
func floatLog10(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatUnaryMath(v, id, args, math.Log10)
}

// PURE by contract
func floatLog2(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return floatUnaryMath(v, id, args, math.Log2)
}

func floatUnaryMath(v Value, id member.ID, args []Value, f func(float64) float64) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	return floatResult(name, f(math.Float64frombits(v.Data)))
}

// --- shared leaves

// floatResult is every float member's result: like the operators (floatArithResult), no member may produce NaN or
// ±Inf. The message names the member.
func floatResult(name string, f float64) (Value, error) {
	if math.IsNaN(f) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the result is not a number", name))
	}
	if math.IsInf(f, 0) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the result is infinite", name)) // overflow, or the log of zero
	}
	return FloatValue(f), nil
}

// floatOperandArg reads a float member's numeric operand: a float, or an int read as a float.
func floatOperandArg(name, pos string, a Value) (float64, error) {
	switch a.Type {
	case value.Float:
		return math.Float64frombits(a.Data), nil
	case value.Int:
		return float64(int64(a.Data)), nil
	}
	return 0, errs.NewInvalidArgumentTypeError(name, pos, "float or int", a.TypeName())
}

// floatRoundable reads a rounding receiver: NaN and ±Inf have no digits to round.
func floatRoundable(name string, v Value) (float64, error) {
	f := math.Float64frombits(v.Data)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) cannot round %g", name, f))
	}
	return f, nil
}

// floatPlacesArg reads a float rounding place count: any int — negative rounds to tens, hundreds, … A count past
// float's range keeps every digit (or none), so it is clamped rather than refused.
func floatPlacesArg(name, pos string, a Value) (int64, error) {
	n, err := parseIntArg(name, pos, a)
	if err != nil {
		return 0, err
	}
	return max(min(n, 1000), -1000), nil
}

// floatRoundPlaces is round(places, mode) on a float.
func floatRoundPlaces(name string, v Value, places int64, mode roundingMode) (Value, error) {
	f, err := floatRoundable(name, v)
	if err != nil {
		return Undefined, err
	}
	// the digit at index i of the shortest text has place value 10^(exp-i); keep those at 10^-places and above
	return floatResult(name, floatRoundKeep(f, mode, func(exp int) int64 { return int64(exp) + places + 1 }))
}

// floatRoundKeep rounds f's shortest decimal text — the digits string() prints — keeping its first keep(exp) digits,
// where exp is the decimal exponent of the first digit. It works on the text, so no magnitude is out of range:
// 1e-30 rounded to 2 places is 0, 1e300 is unchanged.
func floatRoundKeep(f float64, mode roundingMode, keep func(exp int) int64) float64 {
	if f == 0 {
		return f
	}
	negative := f < 0
	text := strconv.FormatFloat(math.Abs(f), 'e', -1, 64) // d.ddde±XX, the shortest digits
	mantissa, expText, _ := strings.Cut(text, "e")
	digits := strings.Replace(mantissa, ".", "", 1)
	exp, _ := strconv.Atoi(expText)
	k := keep(exp)
	if k >= int64(len(digits)) {
		return f // nothing is dropped
	}

	// what is dropped, read as a fraction of the last kept unit
	kept, first, rest := "", byte('0'), true // k < 0: every digit is dropped, and below the first dropped place
	if k >= 0 {
		kept, first, rest = digits[:k], digits[k], strings.TrimRight(digits[k+1:], "0") != ""
	}
	half := 1
	switch {
	case first < '5':
		half = -1
	case first == '5' && !rest:
		half = 0
	}
	odd := kept != "" && (kept[len(kept)-1]-'0')%2 == 1
	if mode.roundsUp(negative, odd, half) {
		kept = incrementDigits(kept)
	}
	if kept == "" {
		return 0
	}
	r, _ := strconv.ParseFloat(kept+"e"+strconv.FormatInt(int64(exp)-k+1, 10), 64) // ±Inf on overflow: floatResult raises
	if negative {
		r = -r
	}
	return r
}

// incrementDigits adds one to a decimal digit string ("" is 0, so it answers "1"; "199" answers "200").
func incrementDigits(d string) string {
	b := []byte(d)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < '9' {
			b[i]++
			return string(b)
		}
		b[i] = '0'
	}
	return "1" + string(b)
}

// floatShortestRat is f's shortest decimal text as an exact rational.
func floatShortestRat(f float64) *big.Rat {
	r, _ := new(big.Rat).SetString(strconv.FormatFloat(f, 'g', -1, 64))
	return r
}

// ratRoundToInt rounds r to an integer with mode.
func ratRoundToInt(r *big.Rat, mode roundingMode) *big.Int {
	negative := r.Sign() < 0
	q, rem := new(big.Int).QuoRem(new(big.Int).Abs(r.Num()), r.Denom(), new(big.Int))
	if rem.Sign() != 0 && mode.roundsUp(negative, q.Bit(0) == 1, new(big.Int).Lsh(rem, 1).Cmp(r.Denom())) {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return q
}
