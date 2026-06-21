package stdlib

import (
	"math"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/errs"
)

func init() {
	InitModule("math", module.Math, nil,
		map[string]core.Value{
			"e":                        core.FloatValue(math.E),
			"pi":                       core.FloatValue(math.Pi),
			"phi":                      core.FloatValue(math.Phi),
			"sqrt2":                    core.FloatValue(math.Sqrt2),
			"sqrt_e":                   core.FloatValue(math.SqrtE),
			"sqrt_pi":                  core.FloatValue(math.SqrtPi),
			"sqrt_phi":                 core.FloatValue(math.SqrtPhi),
			"ln2":                      core.FloatValue(math.Ln2),
			"log2e":                    core.FloatValue(math.Log2E),
			"ln10":                     core.FloatValue(math.Ln10),
			"log10e":                   core.FloatValue(math.Log10E),
			"max_float32":              core.FloatValue(math.MaxFloat32),
			"smallest_nonzero_float32": core.FloatValue(math.SmallestNonzeroFloat32),
			"max_float64":              core.FloatValue(math.MaxFloat64),
			"smallest_nonzero_float64": core.FloatValue(math.SmallestNonzeroFloat64),
			"max_int":                  core.IntValue(math.MaxInt),
			"min_int":                  core.IntValue(math.MinInt),
			"max_int8":                 core.IntValue(math.MaxInt8),
			"min_int8":                 core.IntValue(math.MinInt8),
			"max_int16":                core.IntValue(math.MaxInt16),
			"min_int16":                core.IntValue(math.MinInt16),
			"max_int32":                core.IntValue(math.MaxInt32),
			"min_int32":                core.IntValue(math.MinInt32),
			"max_int64":                core.IntValue(math.MaxInt64),
			"min_int64":                core.IntValue(math.MinInt64),
		},
		// 53..127 reserved
		map[uint64]*core.BuiltinFunction{
			0:  core.NewBuiltinFunction("abs", mathAbs, 1, false),
			1:  core.NewBuiltinFunction("acos", mathAcos, 1, false),
			2:  core.NewBuiltinFunction("acosh", mathAcosh, 1, false),
			3:  core.NewBuiltinFunction("asin", mathAsin, 1, false),
			4:  core.NewBuiltinFunction("asinh", mathAsinh, 1, false),
			5:  core.NewBuiltinFunction("atan", mathAtan, 1, false),
			6:  core.NewBuiltinFunction("atan2", mathAtan2, 2, false),
			7:  core.NewBuiltinFunction("atanh", mathAtanh, 1, false),
			8:  core.NewBuiltinFunction("cbrt", mathCbrt, 1, false),
			9:  core.NewBuiltinFunction("ceil", mathCeil, 1, false),
			10: core.NewBuiltinFunction("copy_sign", mathCopysign, 2, false),
			11: core.NewBuiltinFunction("cos", mathCos, 1, false),
			12: core.NewBuiltinFunction("cosh", mathCosh, 1, false),
			13: core.NewBuiltinFunction("dim", mathDim, 2, false),
			14: core.NewBuiltinFunction("erf", mathErf, 1, false),
			15: core.NewBuiltinFunction("erfc", mathErfc, 1, false),
			16: core.NewBuiltinFunction("exp", mathExp, 1, false),
			17: core.NewBuiltinFunction("exp2", mathExp2, 1, false),
			18: core.NewBuiltinFunction("expm1", mathExpm1, 1, false),
			19: core.NewBuiltinFunction("floor", mathFloor, 1, false),
			20: core.NewBuiltinFunction("gamma", mathGamma, 1, false),
			21: core.NewBuiltinFunction("hypot", mathHypot, 2, false),
			22: core.NewBuiltinFunction("ilogb", mathIlogb, 1, false),
			23: core.NewBuiltinFunction("inf", mathInf, 1, false),
			24: core.NewBuiltinFunction("is_inf", mathIsInf, 2, false),
			25: core.NewBuiltinFunction("is_nan", mathIsNaN, 1, false),
			26: core.NewBuiltinFunction("j0", mathJ0, 1, false),
			27: core.NewBuiltinFunction("j1", mathJ1, 1, false),
			28: core.NewBuiltinFunction("jn", mathJn, 2, false),
			29: core.NewBuiltinFunction("ldexp", mathLdexp, 2, false),
			30: core.NewBuiltinFunction("log", mathLog, 1, false),
			31: core.NewBuiltinFunction("log10", mathLog10, 1, false),
			32: core.NewBuiltinFunction("log1p", mathLog1p, 1, false),
			33: core.NewBuiltinFunction("log2", mathLog2, 1, false),
			34: core.NewBuiltinFunction("logb", mathLogb, 1, false),
			35: core.NewBuiltinFunction("max", mathMax, 2, false),
			36: core.NewBuiltinFunction("min", mathMin, 2, false),
			37: core.NewBuiltinFunction("mod", mathMod, 2, false),
			38: core.NewBuiltinFunction("nan", mathNaN, 0, false),
			39: core.NewBuiltinFunction("next_after", mathNextafter, 2, false),
			40: core.NewBuiltinFunction("pow", mathPow, 2, false),
			41: core.NewBuiltinFunction("pow10", mathPow10, 1, false),
			42: core.NewBuiltinFunction("remainder", mathRemainder, 2, false),
			43: core.NewBuiltinFunction("signbit", mathSignbit, 1, false),
			44: core.NewBuiltinFunction("sin", mathSin, 1, false),
			45: core.NewBuiltinFunction("sinh", mathSinh, 1, false),
			46: core.NewBuiltinFunction("sqrt", mathSqrt, 1, false),
			47: core.NewBuiltinFunction("tan", mathTan, 1, false),
			48: core.NewBuiltinFunction("tanh", mathTanh, 1, false),
			49: core.NewBuiltinFunction("trunc", mathTrunc, 1, false),
			50: core.NewBuiltinFunction("y0", mathY0, 1, false),
			51: core.NewBuiltinFunction("y1", mathY1, 1, false),
			52: core.NewBuiltinFunction("yn", mathYn, 2, false),
		})
}

func mathSignbit(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.signbit", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.signbit", "first", "float(compatible)", args[0].TypeName())
	}
	return core.BoolValue(math.Signbit(f1)), nil
}

func mathIsNaN(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.is_nan", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.is_nan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.BoolValue(math.IsNaN(f1)), nil
}

func mathIsInf(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.is_inf", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.is_inf", "first", "float(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.is_inf", "second", "int(compatible)", args[1].TypeName())
	}
	return core.BoolValue(math.IsInf(f1, int(i2))), nil
}

func mathLdexp(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.ldexp", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ldexp", "first", "float(compatible)", args[0].TypeName())
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ldexp", "second", "int(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Ldexp(f1, int(i2))), nil
}

func mathYn(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.yn", "2", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.yn", "first", "int(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.yn", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Yn(int(i1), f2)), nil
}

func mathJn(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.jn", "2", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.jn", "first", "int(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.jn", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Jn(int(i1), f2)), nil
}

func mathIlogb(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.ilogb", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ilogb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.IntValue(int64(math.Ilogb(f1))), nil
}

func mathPow10(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.pow10", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.pow10", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Pow10(int(i1))), nil
}

func mathInf(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.inf", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.inf", "first", "int(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Inf(int(i1))), nil
}

func mathAbs(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.abs", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.abs", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Abs(f1)), nil
}

func mathAcos(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.acos", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acos(f1)), nil
}

func mathAcosh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.acosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.acosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Acosh(f1)), nil
}

func mathAsin(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.asin", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asin(f1)), nil
}

func mathAsinh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.asinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.asinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Asinh(f1)), nil
}

func mathAtan(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atan", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atan(f1)), nil
}

func mathAtanh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Atanh(f1)), nil
}

func mathCbrt(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cbrt", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cbrt", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cbrt(f1)), nil
}

func mathCeil(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.ceil", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.ceil", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Ceil(f1)), nil
}

func mathCos(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cos", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cos", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cos(f1)), nil
}

func mathCosh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.cosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.cosh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Cosh(f1)), nil
}

func mathErf(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.erf", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erf", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erf(f1)), nil
}

func mathErfc(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.erfc", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.erfc", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Erfc(f1)), nil
}

func mathExp(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.exp", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.exp", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Exp(f1)), nil
}

func mathExp2(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.exp2", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.exp2", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Exp2(f1)), nil
}

func mathExpm1(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.expm1", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.expm1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Expm1(f1)), nil
}

func mathFloor(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.floor", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.floor", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Floor(f1)), nil
}

func mathGamma(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.gamma", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.gamma", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Gamma(f1)), nil
}

func mathJ0(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.j0", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J0(f1)), nil
}

func mathJ1(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.j1", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.j1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.J1(f1)), nil
}

func mathLog(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.log", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log(f1)), nil
}

func mathLog10(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.log10", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log10", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log10(f1)), nil
}

func mathLog1p(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.log1p", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log1p", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log1p(f1)), nil
}

func mathLog2(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.log2", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.log2", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Log2(f1)), nil
}

func mathLogb(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.logb", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.logb", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Logb(f1)), nil
}

func mathSin(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.sin", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sin", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sin(f1)), nil
}

func mathSinh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.sinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sinh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sinh(f1)), nil
}

func mathSqrt(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.sqrt", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.sqrt", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Sqrt(f1)), nil
}

func mathTan(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.tan", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tan", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tan(f1)), nil
}

func mathTanh(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.tanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.tanh", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Tanh(f1)), nil
}

func mathTrunc(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.trunc", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.trunc", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Trunc(f1)), nil
}

func mathY0(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.y0", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y0", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y0(f1)), nil
}

func mathY1(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.y1", "1", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.y1", "first", "float(compatible)", args[0].TypeName())
	}
	return core.FloatValue(math.Y1(f1)), nil
}

func mathAtan2(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.atan2", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan2", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.atan2", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Atan2(f1, f2)), nil
}

func mathCopysign(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.copy_sign", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.copy_sign", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.copy_sign", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Copysign(f1, f2)), nil
}

func mathDim(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.dim", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.dim", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.dim", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Dim(f1, f2)), nil
}

func mathHypot(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.hypot", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.hypot", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.hypot", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Hypot(f1, f2)), nil
}

func mathMax(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.max", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.max", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.max", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Max(f1, f2)), nil
}

func mathMin(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.min", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.min", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.min", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Min(f1, f2)), nil
}

func mathMod(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.mod", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.mod", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.mod", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Mod(f1, f2)), nil
}

func mathNextafter(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.next_after", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.next_after", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.next_after", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Nextafter(f1, f2)), nil
}

func mathPow(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.pow", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.pow", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.pow", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Pow(f1, f2)), nil
}

func mathRemainder(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.remainder", "2", len(args))
	}
	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.remainder", "first", "float(compatible)", args[0].TypeName())
	}
	f2, ok := args[1].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("math.remainder", "second", "float(compatible)", args[1].TypeName())
	}
	return core.FloatValue(math.Remainder(f1, f2)), nil
}

func mathNaN(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("math.nan", "0", len(args))
	}
	return core.FloatValue(math.NaN()), nil
}
