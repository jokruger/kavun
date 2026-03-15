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
		Value: FuncAFFRF(math.Atan2),
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
		Value: FuncAFFRF(math.Copysign),
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
		Value: FuncAFFRF(math.Dim),
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
		Value: FuncAFFRF(math.Hypot),
	},
	"ilogb": &value.BuiltinFunction{
		Name:  "ilogb",
		Value: FuncAFRI(math.Ilogb),
	},
	"inf": &value.BuiltinFunction{
		Name:  "inf",
		Value: FuncAIRF(math.Inf),
	},
	"is_inf": &value.BuiltinFunction{
		Name:  "is_inf",
		Value: FuncAFIRB(math.IsInf),
	},
	"is_nan": &value.BuiltinFunction{
		Name:  "is_nan",
		Value: FuncAFRB(math.IsNaN),
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
		Value: FuncAIFRF(math.Jn),
	},
	"ldexp": &value.BuiltinFunction{
		Name:  "ldexp",
		Value: FuncAFIRF(math.Ldexp),
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
		Value: FuncAFFRF(math.Max),
	},
	"min": &value.BuiltinFunction{
		Name:  "min",
		Value: FuncAFFRF(math.Min),
	},
	"mod": &value.BuiltinFunction{
		Name:  "mod",
		Value: FuncAFFRF(math.Mod),
	},
	"nan": &value.BuiltinFunction{
		Name:  "nan",
		Value: mathNaN,
	},
	"nextafter": &value.BuiltinFunction{
		Name:  "nextafter",
		Value: FuncAFFRF(math.Nextafter),
	},
	"pow": &value.BuiltinFunction{
		Name:  "pow",
		Value: FuncAFFRF(math.Pow),
	},
	"pow10": &value.BuiltinFunction{
		Name:  "pow10",
		Value: FuncAIRF(math.Pow10),
	},
	"remainder": &value.BuiltinFunction{
		Name:  "remainder",
		Value: FuncAFFRF(math.Remainder),
	},
	"signbit": &value.BuiltinFunction{
		Name:  "signbit",
		Value: FuncAFRB(math.Signbit),
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
		Value: FuncAIFRF(math.Yn),
	},
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

func mathNaN(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 0 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.Float{Value: math.NaN()}, nil
}
