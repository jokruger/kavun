package stdlib

import (
	"math"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var mathModule = map[string]core.Object{
	"e":                      &value.Float{Value: math.E},
	"pi":                     &value.Float{Value: math.Pi},
	"phi":                    &value.Float{Value: math.Phi},
	"sqrt2":                  &value.Float{Value: math.Sqrt2},
	"sqrtE":                  &value.Float{Value: math.SqrtE},
	"sqrtPi":                 &value.Float{Value: math.SqrtPi},
	"sqrtPhi":                &value.Float{Value: math.SqrtPhi},
	"ln2":                    &value.Float{Value: math.Ln2},
	"log2E":                  &value.Float{Value: math.Log2E},
	"ln10":                   &value.Float{Value: math.Ln10},
	"log10E":                 &value.Float{Value: math.Log10E},
	"maxFloat32":             &value.Float{Value: math.MaxFloat32},
	"smallestNonzeroFloat32": &value.Float{Value: math.SmallestNonzeroFloat32},
	"maxFloat64":             &value.Float{Value: math.MaxFloat64},
	"smallestNonzeroFloat64": &value.Float{Value: math.SmallestNonzeroFloat64},
	"maxInt":                 &value.Int{Value: math.MaxInt},
	"minInt":                 &value.Int{Value: math.MinInt},
	"maxInt8":                &value.Int{Value: math.MaxInt8},
	"minInt8":                &value.Int{Value: math.MinInt8},
	"maxInt16":               &value.Int{Value: math.MaxInt16},
	"minInt16":               &value.Int{Value: math.MinInt16},
	"maxInt32":               &value.Int{Value: math.MaxInt32},
	"minInt32":               &value.Int{Value: math.MinInt32},
	"maxInt64":               &value.Int{Value: math.MaxInt64},
	"minInt64":               &value.Int{Value: math.MinInt64},

	"abs": &value.BuiltinFunction{
		Name:  "abs",
		Value: mathAbs,
	},
	"acos": &value.BuiltinFunction{
		Name:  "acos",
		Value: mathAcos,
	},
	"acosh": &value.BuiltinFunction{
		Name:  "acosh",
		Value: mathAcosh,
	},
	"asin": &value.BuiltinFunction{
		Name:  "asin",
		Value: mathAsin,
	},
	"asinh": &value.BuiltinFunction{
		Name:  "asinh",
		Value: mathAsinh,
	},
	"atan": &value.BuiltinFunction{
		Name:  "atan",
		Value: mathAtan,
	},
	"atan2": &value.BuiltinFunction{
		Name:  "atan2",
		Value: mathAtan2,
	},
	"atanh": &value.BuiltinFunction{
		Name:  "atanh",
		Value: mathAtanh,
	},
	"cbrt": &value.BuiltinFunction{
		Name:  "cbrt",
		Value: mathCbrt,
	},
	"ceil": &value.BuiltinFunction{
		Name:  "ceil",
		Value: mathCeil,
	},
	"copysign": &value.BuiltinFunction{
		Name:  "copysign",
		Value: mathCopysign,
	},
	"cos": &value.BuiltinFunction{
		Name:  "cos",
		Value: mathCos,
	},
	"cosh": &value.BuiltinFunction{
		Name:  "cosh",
		Value: mathCosh,
	},
	"dim": &value.BuiltinFunction{
		Name:  "dim",
		Value: mathDim,
	},
	"erf": &value.BuiltinFunction{
		Name:  "erf",
		Value: mathErf,
	},
	"erfc": &value.BuiltinFunction{
		Name:  "erfc",
		Value: mathErfc,
	},
	"exp": &value.BuiltinFunction{
		Name:  "exp",
		Value: mathExp,
	},
	"exp2": &value.BuiltinFunction{
		Name:  "exp2",
		Value: mathExp2,
	},
	"expm1": &value.BuiltinFunction{
		Name:  "expm1",
		Value: mathExpm1,
	},
	"floor": &value.BuiltinFunction{
		Name:  "floor",
		Value: mathFloor,
	},
	"gamma": &value.BuiltinFunction{
		Name:  "gamma",
		Value: mathGamma,
	},
	"hypot": &value.BuiltinFunction{
		Name:  "hypot",
		Value: mathHypot,
	},
	"ilogb": &value.BuiltinFunction{
		Name:  "ilogb",
		Value: mathIlogb,
	},
	"inf": &value.BuiltinFunction{
		Name:  "inf",
		Value: mathInf,
	},
	"is_inf": &value.BuiltinFunction{
		Name:  "is_inf",
		Value: mathIsInf,
	},
	"is_nan": &value.BuiltinFunction{
		Name:  "is_nan",
		Value: mathIsNaN,
	},
	"j0": &value.BuiltinFunction{
		Name:  "j0",
		Value: mathJ0,
	},
	"j1": &value.BuiltinFunction{
		Name:  "j1",
		Value: mathJ1,
	},
	"jn": &value.BuiltinFunction{
		Name:  "jn",
		Value: mathJn,
	},
	"ldexp": &value.BuiltinFunction{
		Name:  "ldexp",
		Value: mathLdexp,
	},
	"log": &value.BuiltinFunction{
		Name:  "log",
		Value: mathLog,
	},
	"log10": &value.BuiltinFunction{
		Name:  "log10",
		Value: mathLog10,
	},
	"log1p": &value.BuiltinFunction{
		Name:  "log1p",
		Value: mathLog1p,
	},
	"log2": &value.BuiltinFunction{
		Name:  "log2",
		Value: mathLog2,
	},
	"logb": &value.BuiltinFunction{
		Name:  "logb",
		Value: mathLogb,
	},
	"max": &value.BuiltinFunction{
		Name:  "max",
		Value: mathMax,
	},
	"min": &value.BuiltinFunction{
		Name:  "min",
		Value: mathMin,
	},
	"mod": &value.BuiltinFunction{
		Name:  "mod",
		Value: mathMod,
	},
	"nan": &value.BuiltinFunction{
		Name:  "nan",
		Value: mathNaN,
	},
	"nextafter": &value.BuiltinFunction{
		Name:  "nextafter",
		Value: mathNextafter,
	},
	"pow": &value.BuiltinFunction{
		Name:  "pow",
		Value: mathPow,
	},
	"pow10": &value.BuiltinFunction{
		Name:  "pow10",
		Value: mathPow10,
	},
	"remainder": &value.BuiltinFunction{
		Name:  "remainder",
		Value: mathRemainder,
	},
	"signbit": &value.BuiltinFunction{
		Name:  "signbit",
		Value: mathSignbit,
	},
	"sin": &value.BuiltinFunction{
		Name:  "sin",
		Value: mathSin,
	},
	"sinh": &value.BuiltinFunction{
		Name:  "sinh",
		Value: mathSinh,
	},
	"sqrt": &value.BuiltinFunction{
		Name:  "sqrt",
		Value: mathSqrt,
	},
	"tan": &value.BuiltinFunction{
		Name:  "tan",
		Value: mathTan,
	},
	"tanh": &value.BuiltinFunction{
		Name:  "tanh",
		Value: mathTanh,
	},
	"trunc": &value.BuiltinFunction{
		Name:  "trunc",
		Value: mathTrunc,
	},
	"y0": &value.BuiltinFunction{
		Name:  "y0",
		Value: mathY0,
	},
	"y1": &value.BuiltinFunction{
		Name:  "y1",
		Value: mathY1,
	},
	"yn": &value.BuiltinFunction{
		Name:  "yn",
		Value: mathYn,
	},
}

func mathSignbit(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	if math.Signbit(f1) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathIsNaN(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	if math.IsNaN(f1) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathIsInf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if math.IsInf(f1, int(i2)) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func mathLdexp(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	i2, ok := args[1].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Ldexp(f1, int(i2))}, nil
}

func mathYn(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Yn(int(i1), f2)}, nil
}

func mathJn(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Jn(int(i1), f2)}, nil
}

func mathIlogb(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(math.Ilogb(f1))}, nil
}

func mathPow10(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Pow10(int(i1))}, nil
}

func mathInf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Inf(int(i1))}, nil
}

func mathAbs(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Abs(f1)}, nil
}

func mathAcos(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Acos(f1)}, nil
}

func mathAcosh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Acosh(f1)}, nil
}

func mathAsin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Asin(f1)}, nil
}

func mathAsinh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Asinh(f1)}, nil
}

func mathAtan(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Atan(f1)}, nil
}

func mathAtanh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Atanh(f1)}, nil
}

func mathCbrt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Cbrt(f1)}, nil
}

func mathCeil(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Ceil(f1)}, nil
}

func mathCos(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Cos(f1)}, nil
}

func mathCosh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Cosh(f1)}, nil
}

func mathErf(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Erf(f1)}, nil
}

func mathErfc(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Erfc(f1)}, nil
}

func mathExp(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Exp(f1)}, nil
}

func mathExp2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Exp2(f1)}, nil
}

func mathExpm1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Expm1(f1)}, nil
}

func mathFloor(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Floor(f1)}, nil
}

func mathGamma(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Gamma(f1)}, nil
}

func mathJ0(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.J0(f1)}, nil
}

func mathJ1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.J1(f1)}, nil
}

func mathLog(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Log(f1)}, nil
}

func mathLog10(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Log10(f1)}, nil
}

func mathLog1p(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Log1p(f1)}, nil
}

func mathLog2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Log2(f1)}, nil
}

func mathLogb(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Logb(f1)}, nil
}

func mathSin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Sin(f1)}, nil
}

func mathSinh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Sinh(f1)}, nil
}

func mathSqrt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Sqrt(f1)}, nil
}

func mathTan(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Tan(f1)}, nil
}

func mathTanh(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Tanh(f1)}, nil
}

func mathTrunc(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Trunc(f1)}, nil
}

func mathY0(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Y0(f1)}, nil
}

func mathY1(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Float{Value: math.Y1(f1)}, nil
}

func mathAtan2(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Atan2(f1, f2)}, nil
}

func mathCopysign(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Copysign(f1, f2)}, nil
}

func mathDim(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Dim(f1, f2)}, nil
}

func mathHypot(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Hypot(f1, f2)}, nil
}

func mathMax(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Max(f1, f2)}, nil
}

func mathMin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Min(f1, f2)}, nil
}

func mathMod(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Mod(f1, f2)}, nil
}

func mathNextafter(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Nextafter(f1, f2)}, nil
}

func mathPow(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Pow(f1, f2)}, nil
}

func mathRemainder(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	f1, ok := args[0].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	f2, ok := args[1].AsFloat()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "float(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	return &value.Float{Value: math.Remainder(f1, f2)}, nil
}

func mathNaN(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Float{Value: math.NaN()}, nil
}
