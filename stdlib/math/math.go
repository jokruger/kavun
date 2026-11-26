package math

import (
	"math"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// Constants, trigonometry and the special functions.
//
// A numeric operation that `decimal` has is a **float member** instead — `x.sqrt()`, `x.pow(y)`, `x.exp()`, `x.ln()`,
// `x.log2()`, `x.log10()`, `x.round(n, mode)` and its twins, `x.copy_sign(y)`, `x.quo_rem(y)`, … (see
// [reference/types/float.md § Overview](kavun:type/float#overview)) — so no function exists both ways. `min`/`max` are
// not module functions either: selection over arguments is the free variadic `min(a, b, ...)` / `max(a, b, ...)`, and
// aggregation over elements is the member (`arr.min()`).
//
// Example:
//
// ```go
// math = import("math")
// math.hypot(3, 4)       // => 5.0
// math.cbrt(27)          // => 3.0
// math.pi                // => 3.141592653589793
// ```
//
// The constants are the numeric constants [`pi`](#pi), [`e`](#e), [`phi`](#phi), their square roots and the
// logarithms [`ln2`](#ln2), [`log2e`](#log2e), [`ln10`](#ln10), [`log10e`](#log10e); the float bounds
// [`max_float64`](#max_float64), [`smallest_nonzero_float64`](#smallest_nonzero_float64) and their 32-bit pair; and
// the integer bounds, [`max_int64`](#max_int64) / [`min_int64`](#min_int64) down to the 8-bit pair.
//
// **Failures.** A non-numeric argument raises kind `invalid_argument_type`; a wrong count, `wrong_num_arguments`. A
// result outside the reals is the float's own error state, answered rather than raised (`math.acos(2)` → `NaN`,
// `math.inf(1)` → `+Inf`): arithmetic on it raises, and `x.is_nan()` / `x.is_inf()` ask about it.
//
//kavun:module math id=value.ModuleMath pure

// The ratio of a circle's circumference to its diameter.
//
//kavun:const pi -> float
var constPi = core.FloatValue(math.Pi)

// Euler's number, the base of the natural logarithm.
//
//kavun:const e -> float
var constE = core.FloatValue(math.E)

// The golden ratio, (1 + √5) / 2.
//
//kavun:const phi -> float
var constPhi = core.FloatValue(math.Phi)

// The square root of 2.
//
//kavun:const sqrt2 -> float
var constSqrt2 = core.FloatValue(math.Sqrt2)

// The square root of [`e`](#e).
//
//kavun:const sqrt_e -> float
var constSqrtE = core.FloatValue(math.SqrtE)

// The square root of [`pi`](#pi).
//
//kavun:const sqrt_pi -> float
var constSqrtPi = core.FloatValue(math.SqrtPi)

// The square root of [`phi`](#phi).
//
//kavun:const sqrt_phi -> float
var constSqrtPhi = core.FloatValue(math.SqrtPhi)

// The natural logarithm of 2.
//
//kavun:const ln2 -> float
var constLn2 = core.FloatValue(math.Ln2)

// The base-2 logarithm of [`e`](#e), 1 / [`ln2`](#ln2).
//
//kavun:const log2e -> float
var constLog2e = core.FloatValue(math.Log2E)

// The natural logarithm of 10.
//
//kavun:const ln10 -> float
var constLn10 = core.FloatValue(math.Ln10)

// The base-10 logarithm of [`e`](#e), 1 / [`ln10`](#ln10).
//
//kavun:const log10e -> float
var constLog10e = core.FloatValue(math.Log10E)

// The largest integer of the host's word size: [`max_int64`](#max_int64) on a 64-bit platform.
//
//kavun:const max_int -> int
var constMaxInt = core.IntValue(math.MaxInt)

// The smallest integer of the host's word size: [`min_int64`](#min_int64) on a 64-bit platform.
//
//kavun:const min_int -> int
var constMinInt = core.IntValue(math.MinInt)

// The largest 8-bit signed integer, 127.
//
//kavun:const max_int8 -> int
var constMaxInt8 = core.IntValue(math.MaxInt8)

// The smallest 8-bit signed integer, -128.
//
//kavun:const min_int8 -> int
var constMinInt8 = core.IntValue(math.MinInt8)

// The largest 16-bit signed integer, 32767.
//
//kavun:const max_int16 -> int
var constMaxInt16 = core.IntValue(math.MaxInt16)

// The smallest 16-bit signed integer, -32768.
//
//kavun:const min_int16 -> int
var constMinInt16 = core.IntValue(math.MinInt16)

// The largest 32-bit signed integer, 2147483647.
//
//kavun:const max_int32 -> int
var constMaxInt32 = core.IntValue(math.MaxInt32)

// The smallest 32-bit signed integer, -2147483648.
//
//kavun:const min_int32 -> int
var constMinInt32 = core.IntValue(math.MinInt32)

// The largest 64-bit signed integer, 9223372036854775807: the largest `int`.
//
//kavun:const max_int64 -> int
var constMaxInt64 = core.IntValue(math.MaxInt64)

// The smallest 64-bit signed integer, -9223372036854775808: the smallest `int`.
//
//kavun:const min_int64 -> int
var constMinInt64 = core.IntValue(math.MinInt64)

// The largest finite 32-bit float, about 3.4e38.
//
//kavun:const max_float32 -> float
var constMaxFloat32 = core.FloatValue(math.MaxFloat32)

// The smallest positive 32-bit float, about 1.4e-45 (a subnormal).
//
//kavun:const smallest_nonzero_float32 -> float
var constSmallestNonzeroFloat32 = core.FloatValue(math.SmallestNonzeroFloat32)

// The largest finite `float`, about 1.8e308.
//
//kavun:const max_float64 -> float
var constMaxFloat64 = core.FloatValue(math.MaxFloat64)

// The smallest positive `float`, about 4.9e-324 (a subnormal).
//
//kavun:const smallest_nonzero_float64 -> float
var constSmallestNonzeroFloat64 = core.FloatValue(math.SmallestNonzeroFloat64)

// Answer `frac * 2**exp`.
//
// `frac` is a `float` and `exp` is an `int`.
//
//kavun:fn ldexp slot=11 pure (frac, exp) -> float
func mathLdexp(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ldexp", "first", "float(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ldexp", "second", "int(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Ldexp(f1, int(i2))), nil
}

// Answer the Bessel function Yn of `x`.
//
// `n` is an `int` and `x` is a `float`.
//
//kavun:fn yn slot=35 pure (n, x) -> float
func mathYn(vm core.VM, args []core.Value) (ret core.Value, err error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.yn", "first", "int(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.yn", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Yn(int(i1), f2)), nil
}

// Answer the Bessel function Jn of `x`.
//
// `n` is an `int` and `x` is a `float`.
//
//kavun:fn jn slot=32 pure (n, x) -> float
func mathJn(vm core.VM, args []core.Value) (ret core.Value, err error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.jn", "first", "int(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.jn", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Jn(int(i1), f2)), nil
}

// Answer the binary exponent of `x`, as an `int`.
//
// `x` is a `float`.
//
//kavun:fn ilogb slot=13 pure (x) -> int
func mathIlogb(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ilogb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(math.Ilogb(f1))), nil
}

// Answer 10 to the power `n`.
//
// `n` is an `int`.
//
//kavun:fn pow10 slot=6 pure (n) -> float
func mathPow10(vm core.VM, args []core.Value) (ret core.Value, err error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.pow10", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Pow10(int(i1))), nil
}

// Answer positive infinity for `sign >= 0`, negative infinity otherwise.
//
// `sign` is an `int`.
//
//kavun:fn inf slot=0 pure (sign) -> float
func mathInf(vm core.VM, args []core.Value) (ret core.Value, err error) {
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.inf", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Inf(int(i1))), nil
}

// Answer the arc cosine of `x`.
//
// `x` is a `float`.
//
//kavun:fn acos slot=18 pure (x) -> float
func mathAcos(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acos(f1)), nil
}

// Answer the inverse hyperbolic cosine of `x`.
//
// `x` is a `float`.
//
//kavun:fn acosh slot=25 pure (x) -> float
func mathAcosh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acosh(f1)), nil
}

// Answer the arc sine of `x`.
//
// `x` is a `float`.
//
//kavun:fn asin slot=17 pure (x) -> float
func mathAsin(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asin(f1)), nil
}

// Answer the inverse hyperbolic sine of `x`.
//
// `x` is a `float`.
//
//kavun:fn asinh slot=24 pure (x) -> float
func mathAsinh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asinh(f1)), nil
}

// Answer the arc tangent of `x`.
//
// `x` is a `float`.
//
//kavun:fn atan slot=19 pure (x) -> float
func mathAtan(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atan(f1)), nil
}

// Answer the inverse hyperbolic tangent of `x`.
//
// `x` is a `float`.
//
//kavun:fn atanh slot=26 pure (x) -> float
func mathAtanh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atanh(f1)), nil
}

// Answer the cube root of `x`.
//
// `x` is a `float`.
//
//kavun:fn cbrt slot=2 pure (x) -> float
func mathCbrt(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cbrt", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cbrt(f1)), nil
}

// Answer the cosine of `x`.
//
// `x` is a `float`.
//
//kavun:fn cos slot=15 pure (x) -> float
func mathCos(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cos(f1)), nil
}

// Answer the hyperbolic cosine of `x`.
//
// `x` is a `float`.
//
//kavun:fn cosh slot=22 pure (x) -> float
func mathCosh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cosh(f1)), nil
}

// Answer the error function of `x`.
//
// `x` is a `float`.
//
//kavun:fn erf slot=27 pure (x) -> float
func mathErf(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erf", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erf(f1)), nil
}

// Answer the complementary error function of `x`.
//
// `x` is a `float`.
//
//kavun:fn erfc slot=28 pure (x) -> float
func mathErfc(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erfc", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erfc(f1)), nil
}

// Answer 2 to the power `x`.
//
// `x` is a `float`.
//
//kavun:fn exp2 slot=3 pure (x) -> float
func mathExp2(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.exp2", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Exp2(f1)), nil
}

// Answer `e**x - 1`, precise for small `x`.
//
// `x` is a `float`.
//
//kavun:fn expm1 slot=4 pure (x) -> float
func mathExpm1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.expm1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Expm1(f1)), nil
}

// Answer the gamma function of `x`.
//
// `x` is a `float`.
//
//kavun:fn gamma slot=29 pure (x) -> float
func mathGamma(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.gamma", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Gamma(f1)), nil
}

// Answer the Bessel function J0 of `x`.
//
// `x` is a `float`.
//
//kavun:fn j0 slot=30 pure (x) -> float
func mathJ0(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J0(f1)), nil
}

// Answer the Bessel function J1 of `x`.
//
// `x` is a `float`.
//
//kavun:fn j1 slot=31 pure (x) -> float
func mathJ1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J1(f1)), nil
}

// Answer `ln(1 + x)`, precise for small `x`.
//
// `x` is a `float`.
//
//kavun:fn log1p slot=5 pure (x) -> float
func mathLog1p(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log1p", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log1p(f1)), nil
}

// Answer the binary exponent of `x`, as a `float`.
//
// `x` is a `float`.
//
//kavun:fn logb slot=12 pure (x) -> float
func mathLogb(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.logb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Logb(f1)), nil
}

// Answer the sine of `x`.
//
// `x` is a `float`.
//
//kavun:fn sin slot=14 pure (x) -> float
func mathSin(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sin(f1)), nil
}

// Answer the hyperbolic sine of `x`.
//
// `x` is a `float`.
//
//kavun:fn sinh slot=21 pure (x) -> float
func mathSinh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sinh(f1)), nil
}

// Answer the tangent of `x`.
//
// `x` is a `float`.
//
//kavun:fn tan slot=16 pure (x) -> float
func mathTan(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tan(f1)), nil
}

// Answer the hyperbolic tangent of `x`.
//
// `x` is a `float`.
//
//kavun:fn tanh slot=23 pure (x) -> float
func mathTanh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tanh(f1)), nil
}

// Answer the Bessel function Y0 of `x`.
//
// `x` is a `float`.
//
//kavun:fn y0 slot=33 pure (x) -> float
func mathY0(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y0(f1)), nil
}

// Answer the Bessel function Y1 of `x`.
//
// `x` is a `float`.
//
//kavun:fn y1 slot=34 pure (x) -> float
func mathY1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y1(f1)), nil
}

// Answer the arc tangent of `y/x`, with the quadrant from the signs of both.
//
// `y` and `x` are `float`s.
//
//kavun:fn atan2 slot=20 pure (y, x) -> float
func mathAtan2(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan2", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan2", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Atan2(f1, f2)), nil
}

// Answer the positive difference `max(x - y, 0)`.
//
// `x` and `y` are `float`s.
//
//kavun:fn dim slot=8 pure (x, y) -> float
func mathDim(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.dim", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.dim", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Dim(f1, f2)), nil
}

// Answer `sqrt(p*p + q*q)`.
//
// `p` and `q` are `float`s.
//
//kavun:fn hypot slot=7 pure (p, q) -> float
func mathHypot(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.hypot", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.hypot", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Hypot(f1, f2)), nil
}

// Answer the next representable float after `x` toward `y`.
//
// `x` and `y` are `float`s.
//
//kavun:fn next_after slot=10 pure (x, y) -> float
func mathNextafter(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.next_after", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.next_after", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Nextafter(f1, f2)), nil
}

// Answer the IEEE 754 remainder of `x / y`.
//
// `x` and `y` are `float`s.
//
//kavun:fn remainder slot=9 pure (x, y) -> float
func mathRemainder(vm core.VM, args []core.Value) (ret core.Value, err error) {
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.remainder", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.remainder", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Remainder(f1, f2)), nil
}

// Answer a NaN value.
//
//kavun:fn nan slot=1 pure () -> float
func mathNaN(vm core.VM, args []core.Value) (ret core.Value, err error) {
	return core.FloatValue(math.NaN()), nil
}
