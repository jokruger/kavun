package stdlib

import (
	"fmt"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

// FuncAIFRF transform a function of 'func(int, float64) float64' signature
// into CallableFunc type.
func FuncAIFRF(fn func(int, float64) float64) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		return &value.Float{Value: fn(int(i1), f2)}, nil
	}
}

// FuncAFIRF transform a function of 'func(float64, int) float64' signature
// into CallableFunc type.
func FuncAFIRF(fn func(float64, int) float64) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		return &value.Float{Value: fn(f1, int(i2))}, nil
	}
}

// FuncAFIRB transform a function of 'func(float64, int) bool' signature
// into CallableFunc type.
func FuncAFIRB(fn func(float64, int) bool) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		if fn(f1, int(i2)) {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}
}

// FuncAFRB transform a function of 'func(float64) bool' signature
// into CallableFunc type.
func FuncAFRB(fn func(float64) bool) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		if fn(f1) {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}
}

// FuncASRS transform a function of 'func(string) string' signature into
// CallableFunc type. User function will return 'true' if underlying native
// function returns nil.
func FuncASRS(fn func(string) string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s := fn(s1)
		if len(s) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: s}, nil
	}
}

// FuncASRSs transform a function of 'func(string) []string' signature into
// CallableFunc type.
func FuncASRSs(fn func(string) []string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res := fn(s1)
		arr := &value.Array{}
		for _, elem := range res {
			if len(elem) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			arr.Value = append(arr.Value, &value.String{Value: elem})
		}
		return arr, nil
	}
}

// FuncASRSE transform a function of 'func(string) (string, error)' signature
// into CallableFunc type. User function will return 'true' if underlying
// native function returns nil.
func FuncASRSE(fn func(string) (string, error)) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: res}, nil
	}
}

// FuncASRE transform a function of 'func(string) error' signature into
// CallableFunc type. User function will return 'true' if underlying native
// function returns nil.
func FuncASRE(fn func(string) error) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return wrapError(fn(s1)), nil
	}
}

// FuncASSRE transform a function of 'func(string, string) error' signature
// into CallableFunc type. User function will return 'true' if underlying
// native function returns nil.
func FuncASSRE(fn func(string, string) error) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return wrapError(fn(s1, s2)), nil
	}
}

// FuncASSRSs transform a function of 'func(string, string) []string'
// signature into CallableFunc type.
func FuncASSRSs(fn func(string, string) []string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		arr := &value.Array{}
		for _, res := range fn(s1, s2) {
			if len(res) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			arr.Value = append(arr.Value, &value.String{Value: res})
		}
		return arr, nil
	}
}

// FuncASSIRSs transform a function of 'func(string, string, int) []string'
// signature into CallableFunc type.
func FuncASSIRSs(fn func(string, string, int) []string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 3 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		i3, ok := args[2].AsInt()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
		}
		arr := &value.Array{}
		for _, res := range fn(s1, s2, int(i3)) {
			if len(res) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			arr.Value = append(arr.Value, &value.String{Value: res})
		}
		return arr, nil
	}
}

// FuncASSRI transform a function of 'func(string, string) int' signature into
// CallableFunc type.
func FuncASSRI(fn func(string, string) int) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		return &value.Int{Value: int64(fn(s1, s2))}, nil
	}
}

// FuncASSRS transform a function of 'func(string, string) string' signature
// into CallableFunc type.
func FuncASSRS(fn func(string, string) string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		s := fn(s1, s2)
		if len(s) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: s}, nil
	}
}

// FuncASSRB transform a function of 'func(string, string) bool' signature
// into CallableFunc type.
func FuncASSRB(fn func(string, string) bool) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		if fn(s1, s2) {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}
}

// FuncASsSRS transform a function of 'func([]string, string) string' signature
// into CallableFunc type.
func FuncASsSRS(fn func([]string, string) string) core.NativeFunc {
	return func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		var ss1 []string
		switch arg0 := args[0].(type) {
		case *value.Array:
			for idx, a := range arg0.Value {
				as, ok := a.AsString()
				if !ok {
					return nil, gse.ErrInvalidArgumentType{
						Name:     fmt.Sprintf("first[%d]", idx),
						Expected: "string(compatible)",
						Found:    a.TypeName(),
					}
				}
				ss1 = append(ss1, as)
			}
		case *value.ImmutableArray:
			for idx, a := range arg0.Value {
				as, ok := a.AsString()
				if !ok {
					return nil, gse.ErrInvalidArgumentType{
						Name:     fmt.Sprintf("first[%d]", idx),
						Expected: "string(compatible)",
						Found:    a.TypeName(),
					}
				}
				ss1 = append(ss1, as)
			}
		default:
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "array",
				Found:    args[0].TypeName(),
			}
		}
		s2, ok := args[1].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "string(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		s := fn(ss1, s2)
		if len(s) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: s}, nil
	}
}

// FuncASI64RE transform a function of 'func(string, int64) error' signature
// into CallableFunc type.
func FuncASI64RE(fn func(string, int64) error) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
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
		return wrapError(fn(s1, i2)), nil
	}
}

// FuncAIIRE transform a function of 'func(int, int) error' signature
// into CallableFunc type.
func FuncAIIRE(fn func(int, int) error) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		i2, ok := args[1].AsInt()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int(compatible)",
				Found:    args[1].TypeName(),
			}
		}
		return wrapError(fn(int(i1), int(i2))), nil
	}
}

// FuncASIRS transform a function of 'func(string, int) string' signature
// into CallableFunc type.
func FuncASIRS(fn func(string, int) string) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
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
		s := fn(s1, int(i2))
		if len(s) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: s}, nil
	}
}

// FuncASIIRE transform a function of 'func(string, int, int) error' signature
// into CallableFunc type.
func FuncASIIRE(fn func(string, int, int) error) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 3 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
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
		i3, ok := args[2].AsInt()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
		}
		return wrapError(fn(s1, int(i2), int(i3))), nil
	}
}

// FuncAYRIE transform a function of 'func([]byte) (int, error)' signature
// into CallableFunc type.
func FuncAYRIE(fn func([]byte) (int, error)) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		y1, ok := args[0].AsByteSlice()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "bytes(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(y1)
		if err != nil {
			return wrapError(err), nil
		}
		return &value.Int{Value: int64(res)}, nil
	}
}

// FuncAYRS transform a function of 'func([]byte) string' signature into
// CallableFunc type.
func FuncAYRS(fn func([]byte) string) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		y1, ok := args[0].AsByteSlice()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "bytes(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res := fn(y1)
		return &value.String{Value: res}, nil
	}
}

// FuncASRIE transform a function of 'func(string) (int, error)' signature
// into CallableFunc type.
func FuncASRIE(fn func(string) (int, error)) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		return &value.Int{Value: int64(res)}, nil
	}
}

// FuncASRYE transform a function of 'func(string) ([]byte, error)' signature
// into CallableFunc type.
func FuncASRYE(fn func(string) ([]byte, error)) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}
		s1, ok := args[0].AsString()
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "first",
				Expected: "string(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		res, err := fn(s1)
		if err != nil {
			return wrapError(err), nil
		}
		if len(res) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return &value.Bytes{Value: res}, nil
	}
}

// FuncAIRSsE transform a function of 'func(int) ([]string, error)' signature
// into CallableFunc type.
func FuncAIRSsE(fn func(int) ([]string, error)) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		res, err := fn(int(i1))
		if err != nil {
			return wrapError(err), nil
		}
		arr := &value.Array{}
		for _, r := range res {
			if len(r) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			arr.Value = append(arr.Value, &value.String{Value: r})
		}
		return arr, nil
	}
}

// FuncAIRS transform a function of 'func(int) string' signature into
// CallableFunc type.
func FuncAIRS(fn func(int) string) core.NativeFunc {
	return func(args ...core.Object) (ret core.Object, err error) {
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
		s := fn(int(i1))
		if len(s) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: s}, nil
	}
}
