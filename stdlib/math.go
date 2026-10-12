package stdlib

import (
	"math"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

func init() {
	m := core.NewModule(value.ModuleMath, "math", true)
	m.Const(members.Pi, core.FloatValue(math.Pi))
	m.Const(members.E, core.FloatValue(math.E))
	m.Const(members.Phi, core.FloatValue(math.Phi))
	m.Const(members.Sqrt2, core.FloatValue(math.Sqrt2))
	m.Const(members.SqrtE, core.FloatValue(math.SqrtE))
	m.Const(members.SqrtPi, core.FloatValue(math.SqrtPi))
	m.Const(members.SqrtPhi, core.FloatValue(math.SqrtPhi))
	m.Const(members.Ln2, core.FloatValue(math.Ln2))
	m.Const(members.Log2e, core.FloatValue(math.Log2E))
	m.Const(members.Ln10, core.FloatValue(math.Ln10))
	m.Const(members.Log10e, core.FloatValue(math.Log10E))
	m.Const(members.MaxInt, core.IntValue(math.MaxInt))
	m.Const(members.MinInt, core.IntValue(math.MinInt))
	m.Const(members.MaxInt8, core.IntValue(math.MaxInt8))
	m.Const(members.MinInt8, core.IntValue(math.MinInt8))
	m.Const(members.MaxInt16, core.IntValue(math.MaxInt16))
	m.Const(members.MinInt16, core.IntValue(math.MinInt16))
	m.Const(members.MaxInt32, core.IntValue(math.MaxInt32))
	m.Const(members.MinInt32, core.IntValue(math.MinInt32))
	m.Const(members.MaxInt64, core.IntValue(math.MaxInt64))
	m.Const(members.MinInt64, core.IntValue(math.MinInt64))
	m.Const(members.MaxFloat32, core.FloatValue(math.MaxFloat32))
	m.Const(members.SmallestNonzeroFloat32, core.FloatValue(math.SmallestNonzeroFloat32))
	m.Const(members.MaxFloat64, core.FloatValue(math.MaxFloat64))
	m.Const(members.SmallestNonzeroFloat64, core.FloatValue(math.SmallestNonzeroFloat64))
	// functions with no receiver, or no numeric-member reading (a float member answers sqrt, pow, exp, ln, log2,
	// log10, the rounding family, copy_sign, quo_rem and the predicates: no function exists twice)
	m.PureFn(members.Inf, 0, mathInf, 1, core.Fixed)
	m.PureFn(members.NaN, 1, mathNaN, 0, core.Fixed)
	m.PureFn(members.Cbrt, 2, mathCbrt, 1, core.Fixed)
	m.PureFn(members.Exp2, 3, mathExp2, 1, core.Fixed)
	m.PureFn(members.Expm1, 4, mathExpm1, 1, core.Fixed)
	m.PureFn(members.Log1p, 5, mathLog1p, 1, core.Fixed)
	m.PureFn(members.Pow10, 6, mathPow10, 1, core.Fixed)
	m.PureFn(members.Hypot, 7, mathHypot, 2, core.Fixed)
	m.PureFn(members.Dim, 8, mathDim, 2, core.Fixed)
	m.PureFn(members.Remainder, 9, mathRemainder, 2, core.Fixed)
	m.PureFn(members.NextAfter, 10, mathNextafter, 2, core.Fixed)
	m.PureFn(members.Ldexp, 11, mathLdexp, 2, core.Fixed)
	m.PureFn(members.Logb, 12, mathLogb, 1, core.Fixed)
	m.PureFn(members.Ilogb, 13, mathIlogb, 1, core.Fixed)
	m.PureFn(members.Sin, 14, mathSin, 1, core.Fixed)
	m.PureFn(members.Cos, 15, mathCos, 1, core.Fixed)
	m.PureFn(members.Tan, 16, mathTan, 1, core.Fixed)
	m.PureFn(members.Asin, 17, mathAsin, 1, core.Fixed)
	m.PureFn(members.Acos, 18, mathAcos, 1, core.Fixed)
	m.PureFn(members.Atan, 19, mathAtan, 1, core.Fixed)
	m.PureFn(members.Atan2, 20, mathAtan2, 2, core.Fixed)
	m.PureFn(members.Sinh, 21, mathSinh, 1, core.Fixed)
	m.PureFn(members.Cosh, 22, mathCosh, 1, core.Fixed)
	m.PureFn(members.Tanh, 23, mathTanh, 1, core.Fixed)
	m.PureFn(members.Asinh, 24, mathAsinh, 1, core.Fixed)
	m.PureFn(members.Acosh, 25, mathAcosh, 1, core.Fixed)
	m.PureFn(members.Atanh, 26, mathAtanh, 1, core.Fixed)
	m.PureFn(members.Erf, 27, mathErf, 1, core.Fixed)
	m.PureFn(members.Erfc, 28, mathErfc, 1, core.Fixed)
	m.PureFn(members.Gamma, 29, mathGamma, 1, core.Fixed)
	m.PureFn(members.J0, 30, mathJ0, 1, core.Fixed)
	m.PureFn(members.J1, 31, mathJ1, 1, core.Fixed)
	m.PureFn(members.Jn, 32, mathJn, 2, core.Fixed)
	m.PureFn(members.Y0, 33, mathY0, 1, core.Fixed)
	m.PureFn(members.Y1, 34, mathY1, 1, core.Fixed)
	m.PureFn(members.Yn, 35, mathYn, 2, core.Fixed)
	// 36..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func mathLdexp(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.ldexp", "2", len(args))
	}
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

func mathYn(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.yn", "2", len(args))
	}
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

func mathJn(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.jn", "2", len(args))
	}
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

func mathIlogb(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.ilogb", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ilogb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(math.Ilogb(f1))), nil
}

func mathPow10(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.pow10", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.pow10", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Pow10(int(i1))), nil
}

func mathInf(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.inf", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.inf", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Inf(int(i1))), nil
}

func mathAcos(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.acos", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acos(f1)), nil
}

func mathAcosh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.acosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acosh(f1)), nil
}

func mathAsin(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.asin", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asin(f1)), nil
}

func mathAsinh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.asinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asinh(f1)), nil
}

func mathAtan(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atan", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atan(f1)), nil
}

func mathAtanh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atanh(f1)), nil
}

func mathCbrt(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cbrt", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cbrt", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cbrt(f1)), nil
}

func mathCos(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cos", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cos(f1)), nil
}

func mathCosh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cosh(f1)), nil
}

func mathErf(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.erf", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erf", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erf(f1)), nil
}

func mathErfc(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.erfc", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erfc", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erfc(f1)), nil
}

func mathExp2(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.exp2", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.exp2", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Exp2(f1)), nil
}

func mathExpm1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.expm1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.expm1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Expm1(f1)), nil
}

func mathGamma(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.gamma", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.gamma", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Gamma(f1)), nil
}

func mathJ0(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.j0", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J0(f1)), nil
}

func mathJ1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.j1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J1(f1)), nil
}

func mathLog1p(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.log1p", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log1p", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log1p(f1)), nil
}

func mathLogb(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.logb", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.logb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Logb(f1)), nil
}

func mathSin(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.sin", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sin(f1)), nil
}

func mathSinh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.sinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sinh(f1)), nil
}

func mathTan(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.tan", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tan(f1)), nil
}

func mathTanh(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.tanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tanh(f1)), nil
}

func mathY0(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.y0", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y0(f1)), nil
}

func mathY1(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.y1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y1(f1)), nil
}

func mathAtan2(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atan2", "2", len(args))
	}
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

func mathDim(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.dim", "2", len(args))
	}
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

func mathHypot(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.hypot", "2", len(args))
	}
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

func mathNextafter(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.next_after", "2", len(args))
	}
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

func mathRemainder(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.remainder", "2", len(args))
	}
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

func mathNaN(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.nan", "0", len(args))
	}
	return core.FloatValue(math.NaN()), nil
}
