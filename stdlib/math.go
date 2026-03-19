package stdlib

import (
	"math"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

var mathModule = map[string]core.Object{
	"e":                      value.NewFloat(math.E),
	"pi":                     value.NewFloat(math.Pi),
	"phi":                    value.NewFloat(math.Phi),
	"sqrt2":                  value.NewFloat(math.Sqrt2),
	"sqrtE":                  value.NewFloat(math.SqrtE),
	"sqrtPi":                 value.NewFloat(math.SqrtPi),
	"sqrtPhi":                value.NewFloat(math.SqrtPhi),
	"ln2":                    value.NewFloat(math.Ln2),
	"log2E":                  value.NewFloat(math.Log2E),
	"ln10":                   value.NewFloat(math.Ln10),
	"log10E":                 value.NewFloat(math.Log10E),
	"maxFloat32":             value.NewFloat(math.MaxFloat32),
	"smallestNonzeroFloat32": value.NewFloat(math.SmallestNonzeroFloat32),
	"maxFloat64":             value.NewFloat(math.MaxFloat64),
	"smallestNonzeroFloat64": value.NewFloat(math.SmallestNonzeroFloat64),
	"maxInt":                 value.NewInt(math.MaxInt),
	"minInt":                 value.NewInt(math.MinInt),
	"maxInt8":                value.NewInt(math.MaxInt8),
	"minInt8":                value.NewInt(math.MinInt8),
	"maxInt16":               value.NewInt(math.MaxInt16),
	"minInt16":               value.NewInt(math.MinInt16),
	"maxInt32":               value.NewInt(math.MaxInt32),
	"minInt32":               value.NewInt(math.MinInt32),
	"maxInt64":               value.NewInt(math.MaxInt64),
	"minInt64":               value.NewInt(math.MinInt64),

	"abs":       value.NewBuiltinFunction("abs", mathAbs, 1, false),
	"acos":      value.NewBuiltinFunction("acos", mathAcos, 1, false),
	"acosh":     value.NewBuiltinFunction("acosh", mathAcosh, 1, false),
	"asin":      value.NewBuiltinFunction("asin", mathAsin, 1, false),
	"asinh":     value.NewBuiltinFunction("asinh", mathAsinh, 1, false),
	"atan":      value.NewBuiltinFunction("atan", mathAtan, 1, false),
	"atan2":     value.NewBuiltinFunction("atan2", mathAtan2, 2, false),
	"atanh":     value.NewBuiltinFunction("atanh", mathAtanh, 1, false),
	"cbrt":      value.NewBuiltinFunction("cbrt", mathCbrt, 1, false),
	"ceil":      value.NewBuiltinFunction("ceil", mathCeil, 1, false),
	"copysign":  value.NewBuiltinFunction("copysign", mathCopysign, 2, false),
	"cos":       value.NewBuiltinFunction("cos", mathCos, 1, false),
	"cosh":      value.NewBuiltinFunction("cosh", mathCosh, 1, false),
	"dim":       value.NewBuiltinFunction("dim", mathDim, 2, false),
	"erf":       value.NewBuiltinFunction("erf", mathErf, 1, false),
	"erfc":      value.NewBuiltinFunction("erfc", mathErfc, 1, false),
	"exp":       value.NewBuiltinFunction("exp", mathExp, 1, false),
	"exp2":      value.NewBuiltinFunction("exp2", mathExp2, 1, false),
	"expm1":     value.NewBuiltinFunction("expm1", mathExpm1, 1, false),
	"floor":     value.NewBuiltinFunction("floor", mathFloor, 1, false),
	"gamma":     value.NewBuiltinFunction("gamma", mathGamma, 1, false),
	"hypot":     value.NewBuiltinFunction("hypot", mathHypot, 2, false),
	"ilogb":     value.NewBuiltinFunction("ilogb", mathIlogb, 1, false),
	"inf":       value.NewBuiltinFunction("inf", mathInf, 1, false),
	"is_inf":    value.NewBuiltinFunction("is_inf", mathIsInf, 2, false),
	"is_nan":    value.NewBuiltinFunction("is_nan", mathIsNaN, 1, false),
	"j0":        value.NewBuiltinFunction("j0", mathJ0, 1, false),
	"j1":        value.NewBuiltinFunction("j1", mathJ1, 1, false),
	"jn":        value.NewBuiltinFunction("jn", mathJn, 2, false),
	"ldexp":     value.NewBuiltinFunction("ldexp", mathLdexp, 2, false),
	"log":       value.NewBuiltinFunction("log", mathLog, 1, false),
	"log10":     value.NewBuiltinFunction("log10", mathLog10, 1, false),
	"log1p":     value.NewBuiltinFunction("log1p", mathLog1p, 1, false),
	"log2":      value.NewBuiltinFunction("log2", mathLog2, 1, false),
	"logb":      value.NewBuiltinFunction("logb", mathLogb, 1, false),
	"max":       value.NewBuiltinFunction("max", mathMax, 2, false),
	"min":       value.NewBuiltinFunction("min", mathMin, 2, false),
	"mod":       value.NewBuiltinFunction("mod", mathMod, 2, false),
	"nan":       value.NewBuiltinFunction("nan", mathNaN, 0, false),
	"nextafter": value.NewBuiltinFunction("nextafter", mathNextafter, 2, false),
	"pow":       value.NewBuiltinFunction("pow", mathPow, 2, false),
	"pow10":     value.NewBuiltinFunction("pow10", mathPow10, 1, false),
	"remainder": value.NewBuiltinFunction("remainder", mathRemainder, 2, false),
	"signbit":   value.NewBuiltinFunction("signbit", mathSignbit, 1, false),
	"sin":       value.NewBuiltinFunction("sin", mathSin, 1, false),
	"sinh":      value.NewBuiltinFunction("sinh", mathSinh, 1, false),
	"sqrt":      value.NewBuiltinFunction("sqrt", mathSqrt, 1, false),
	"tan":       value.NewBuiltinFunction("tan", mathTan, 1, false),
	"tanh":      value.NewBuiltinFunction("tanh", mathTanh, 1, false),
	"trunc":     value.NewBuiltinFunction("trunc", mathTrunc, 1, false),
	"y0":        value.NewBuiltinFunction("y0", mathY0, 1, false),
	"y1":        value.NewBuiltinFunction("y1", mathY1, 1, false),
	"yn":        value.NewBuiltinFunction("yn", mathYn, 2, false),
}

func mathSignbit(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.signbit", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.signbit", "first", "float(compatible)", args[0])
	}
	if math.Signbit(f1) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathIsNaN(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.is_nan", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.is_nan", "first", "float(compatible)", args[0])
	}
	if math.IsNaN(f1) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathIsInf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.is_inf", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.is_inf", "first", "float(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.is_inf", "second", "int(compatible)", args[1])
	}
	if math.IsInf(f1, int(i2)) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathLdexp(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.ldexp", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.ldexp", "first", "float(compatible)", args[0])
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.ldexp", "second", "int(compatible)", args[1])
	}
	return value.NewFloat(math.Ldexp(f1, int(i2))), nil
}

func mathYn(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.yn", "2", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.yn", "first", "int(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.yn", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Yn(int(i1), f2)), nil
}

func mathJn(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.jn", "2", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.jn", "first", "int(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.jn", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Jn(int(i1), f2)), nil
}

func mathIlogb(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.ilogb", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.ilogb", "first", "float(compatible)", args[0])
	}
	return value.NewInt(int64(math.Ilogb(f1))), nil
}

func mathPow10(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.pow10", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.pow10", "first", "int(compatible)", args[0])
	}
	return value.NewFloat(math.Pow10(int(i1))), nil
}

func mathInf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.inf", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.inf", "first", "int(compatible)", args[0])
	}
	return value.NewFloat(math.Inf(int(i1))), nil
}

func mathAbs(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.abs", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.abs", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Abs(f1)), nil
}

func mathAcos(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.acos", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.acos", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Acos(f1)), nil
}

func mathAcosh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.acosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.acosh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Acosh(f1)), nil
}

func mathAsin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.asin", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.asin", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Asin(f1)), nil
}

func mathAsinh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.asinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.asinh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Asinh(f1)), nil
}

func mathAtan(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.atan", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.atan", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Atan(f1)), nil
}

func mathAtanh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.atanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.atanh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Atanh(f1)), nil
}

func mathCbrt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.cbrt", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.cbrt", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Cbrt(f1)), nil
}

func mathCeil(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.ceil", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.ceil", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Ceil(f1)), nil
}

func mathCos(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.cos", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.cos", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Cos(f1)), nil
}

func mathCosh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.cosh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.cosh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Cosh(f1)), nil
}

func mathErf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.erf", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.erf", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Erf(f1)), nil
}

func mathErfc(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.erfc", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.erfc", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Erfc(f1)), nil
}

func mathExp(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.exp", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.exp", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Exp(f1)), nil
}

func mathExp2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.exp2", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.exp2", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Exp2(f1)), nil
}

func mathExpm1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.expm1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.expm1", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Expm1(f1)), nil
}

func mathFloor(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.floor", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.floor", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Floor(f1)), nil
}

func mathGamma(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.gamma", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.gamma", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Gamma(f1)), nil
}

func mathJ0(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.j0", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.j0", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.J0(f1)), nil
}

func mathJ1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.j1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.j1", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.J1(f1)), nil
}

func mathLog(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.log", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.log", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Log(f1)), nil
}

func mathLog10(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.log10", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.log10", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Log10(f1)), nil
}

func mathLog1p(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.log1p", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.log1p", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Log1p(f1)), nil
}

func mathLog2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.log2", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.log2", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Log2(f1)), nil
}

func mathLogb(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.logb", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.logb", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Logb(f1)), nil
}

func mathSin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.sin", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.sin", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Sin(f1)), nil
}

func mathSinh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.sinh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.sinh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Sinh(f1)), nil
}

func mathSqrt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.sqrt", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.sqrt", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Sqrt(f1)), nil
}

func mathTan(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.tan", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.tan", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Tan(f1)), nil
}

func mathTanh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.tanh", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.tanh", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Tanh(f1)), nil
}

func mathTrunc(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.trunc", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.trunc", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Trunc(f1)), nil
}

func mathY0(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.y0", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.y0", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Y0(f1)), nil
}

func mathY1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("math.y1", "1", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.y1", "first", "float(compatible)", args[0])
	}
	return value.NewFloat(math.Y1(f1)), nil
}

func mathAtan2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.atan2", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.atan2", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.atan2", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Atan2(f1, f2)), nil
}

func mathCopysign(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.copysign", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.copysign", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.copysign", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Copysign(f1, f2)), nil
}

func mathDim(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.dim", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.dim", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.dim", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Dim(f1, f2)), nil
}

func mathHypot(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.hypot", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.hypot", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.hypot", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Hypot(f1, f2)), nil
}

func mathMax(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.max", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.max", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.max", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Max(f1, f2)), nil
}

func mathMin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.min", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.min", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.min", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Min(f1, f2)), nil
}

func mathMod(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.mod", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.mod", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.mod", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Mod(f1, f2)), nil
}

func mathNextafter(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.nextafter", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.nextafter", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.nextafter", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Nextafter(f1, f2)), nil
}

func mathPow(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.pow", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.pow", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.pow", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Pow(f1, f2)), nil
}

func mathRemainder(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("math.remainder", "2", len(args))
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.remainder", "first", "float(compatible)", args[0])
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("math.remainder", "second", "float(compatible)", args[1])
	}
	return value.NewFloat(math.Remainder(f1, f2)), nil
}

func mathNaN(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, core.NewWrongNumArgumentsError("math.nan", "0", len(args))
	}
	return value.NewFloat(math.NaN()), nil
}
