package stdlib_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/stdlib"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/value"
)

func TestFuncAIR(t *testing.T) {
	uf := stdlib.FuncAIR(func(int) {})
	ret, err := funcCall(uf, &value.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t, value.UndefinedValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASRE(t *testing.T) {
	uf := stdlib.FuncASRE(func(a string) error { return nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	uf = stdlib.FuncASRE(func(a string) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASRS(t *testing.T) {
	uf := stdlib.FuncASRS(func(a string) string { return a })
	ret, err := funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "foo"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASRSs(t *testing.T) {
	uf := stdlib.FuncASRSs(func(a string) []string { return []string{a} })
	ret, err := funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, array(&value.String{Value: "foo"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASI64RE(t *testing.T) {
	uf := stdlib.FuncASI64RE(func(a string, b int64) error { return nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"}, &value.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	uf = stdlib.FuncASI64RE(func(a string, b int64) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"}, &value.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAIIRE(t *testing.T) {
	uf := stdlib.FuncAIIRE(func(a, b int) error { return nil })
	ret, err := funcCall(uf, &value.Int{Value: 5}, &value.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	uf = stdlib.FuncAIIRE(func(a, b int) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &value.Int{Value: 5}, &value.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASIIRE(t *testing.T) {
	uf := stdlib.FuncASIIRE(func(a string, b, c int) error { return nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"}, &value.Int{Value: 5},
		&value.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	uf = stdlib.FuncASIIRE(func(a string, b, c int) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"}, &value.Int{Value: 5},
		&value.Int{Value: 7})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASRSE(t *testing.T) {
	uf := stdlib.FuncASRSE(func(a string) (string, error) { return a, nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "foo"}, ret)
	uf = stdlib.FuncASRSE(func(a string) (string, error) {
		return a, errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSRE(t *testing.T) {
	uf := stdlib.FuncASSRE(func(a, b string) error { return nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"},
		&value.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	uf = stdlib.FuncASSRE(func(a, b string) error {
		return errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"},
		&value.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf, &value.String{Value: "foo"})
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASsRS(t *testing.T) {
	uf := stdlib.FuncASsSRS(func(a []string, b string) string {
		return strings.Join(a, b)
	})
	ret, err := funcCall(uf, array(&value.String{Value: "foo"},
		&value.String{Value: "bar"}), &value.String{Value: " "})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "foo bar"}, ret)
	_, err = funcCall(uf, &value.String{Value: "foo"})
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAIRF(t *testing.T) {
	uf := stdlib.FuncAIRF(func(a int) float64 {
		return float64(a)
	})
	ret, err := funcCall(uf, &value.Int{Value: 10.0})
	require.NoError(t, err)
	require.Equal(t, &value.Float{Value: 10.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAFRI(t *testing.T) {
	uf := stdlib.FuncAFRI(func(a float64) int {
		return int(a)
	})
	ret, err := funcCall(uf, &value.Float{Value: 10.5})
	require.NoError(t, err)
	require.Equal(t, &value.Int{Value: 10}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAFRB(t *testing.T) {
	uf := stdlib.FuncAFRB(func(a float64) bool {
		return a > 0.0
	})
	ret, err := funcCall(uf, &value.Float{Value: 0.1})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAFFRF(t *testing.T) {
	uf := stdlib.FuncAFFRF(func(a, b float64) float64 {
		return a + b
	})
	ret, err := funcCall(uf, &value.Float{Value: 10.0},
		&value.Float{Value: 20.0})
	require.NoError(t, err)
	require.Equal(t, &value.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASIRS(t *testing.T) {
	uf := stdlib.FuncASIRS(func(a string, b int) string {
		return strings.Repeat(a, b)
	})
	ret, err := funcCall(uf, &value.String{Value: "ab"}, &value.Int{Value: 2})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "abab"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAIFRF(t *testing.T) {
	uf := stdlib.FuncAIFRF(func(a int, b float64) float64 {
		return float64(a) + b
	})
	ret, err := funcCall(uf, &value.Int{Value: 10}, &value.Float{Value: 20.0})
	require.NoError(t, err)
	require.Equal(t, &value.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAFIRF(t *testing.T) {
	uf := stdlib.FuncAFIRF(func(a float64, b int) float64 {
		return a + float64(b)
	})
	ret, err := funcCall(uf, &value.Float{Value: 10.0}, &value.Int{Value: 20})
	require.NoError(t, err)
	require.Equal(t, &value.Float{Value: 30.0}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAFIRB(t *testing.T) {
	uf := stdlib.FuncAFIRB(func(a float64, b int) bool {
		return a < float64(b)
	})
	ret, err := funcCall(uf, &value.Float{Value: 10.0}, &value.Int{Value: 20})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
	_, err = funcCall(uf, value.TrueValue)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAIRSsE(t *testing.T) {
	uf := stdlib.FuncAIRSsE(func(a int) ([]string, error) {
		return []string{"foo", "bar"}, nil
	})
	ret, err := funcCall(uf, &value.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t, array(&value.String{Value: "foo"},
		&value.String{Value: "bar"}), ret)
	uf = stdlib.FuncAIRSsE(func(a int) ([]string, error) {
		return nil, errors.New("some error")
	})
	ret, err = funcCall(uf, &value.Int{Value: 10})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSRSs(t *testing.T) {
	uf := stdlib.FuncASSRSs(func(a, b string) []string {
		return []string{a, b}
	})
	ret, err := funcCall(uf, &value.String{Value: "foo"},
		&value.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, array(&value.String{Value: "foo"},
		&value.String{Value: "bar"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSIRSs(t *testing.T) {
	uf := stdlib.FuncASSIRSs(func(a, b string, c int) []string {
		return []string{a, b, strconv.Itoa(c)}
	})
	ret, err := funcCall(uf, &value.String{Value: "foo"},
		&value.String{Value: "bar"}, &value.Int{Value: 5})
	require.NoError(t, err)
	require.Equal(t, array(&value.String{Value: "foo"},
		&value.String{Value: "bar"}, &value.String{Value: "5"}), ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASRIE(t *testing.T) {
	uf := stdlib.FuncASRIE(func(a string) (int, error) { return 5, nil })
	ret, err := funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t, &value.Int{Value: 5}, ret)
	uf = stdlib.FuncASRIE(func(a string) (int, error) {
		return 0, errors.New("some error")
	})
	ret, err = funcCall(uf, &value.String{Value: "foo"})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAYRIE(t *testing.T) {
	uf := stdlib.FuncAYRIE(func(a []byte) (int, error) { return 5, nil })
	ret, err := funcCall(uf, &value.Bytes{Value: []byte("foo")})
	require.NoError(t, err)
	require.Equal(t, &value.Int{Value: 5}, ret)
	uf = stdlib.FuncAYRIE(func(a []byte) (int, error) {
		return 0, errors.New("some error")
	})
	ret, err = funcCall(uf, &value.Bytes{Value: []byte("foo")})
	require.NoError(t, err)
	require.Equal(t,
		&value.Error{Value: &value.String{Value: "some error"}}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSRI(t *testing.T) {
	uf := stdlib.FuncASSRI(func(a, b string) int { return len(a) + len(b) })
	ret, err := funcCall(uf,
		&value.String{Value: "foo"}, &value.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, &value.Int{Value: 6}, ret)
	_, err = funcCall(uf, &value.String{Value: "foo"})
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSRS(t *testing.T) {
	uf := stdlib.FuncASSRS(func(a, b string) string { return a + b })
	ret, err := funcCall(uf,
		&value.String{Value: "foo"}, &value.String{Value: "bar"})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "foobar"}, ret)
	_, err = funcCall(uf, &value.String{Value: "foo"})
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASSRB(t *testing.T) {
	uf := stdlib.FuncASSRB(func(a, b string) bool { return len(a) > len(b) })
	ret, err := funcCall(uf,
		&value.String{Value: "123"}, &value.String{Value: "12"})
	require.NoError(t, err)
	require.Equal(t, value.TrueValue, ret)
	_, err = funcCall(uf, &value.String{Value: "foo"})
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAIRS(t *testing.T) {
	uf := stdlib.FuncAIRS(func(a int) string { return strconv.Itoa(a) })
	ret, err := funcCall(uf, &value.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "55"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncAI64R(t *testing.T) {
	uf := stdlib.FuncAIR(func(a int) {})
	ret, err := funcCall(uf, &value.Int{Value: 55})
	require.NoError(t, err)
	require.Equal(t, value.UndefinedValue, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func TestFuncASsSRS(t *testing.T) {
	uf := stdlib.FuncASsSRS(func(a []string, b string) string {
		return strings.Join(a, b)
	})
	ret, err := funcCall(uf,
		array(&value.String{Value: "abc"}, &value.String{Value: "def"}),
		&value.String{Value: "-"})
	require.NoError(t, err)
	require.Equal(t, &value.String{Value: "abc-def"}, ret)
	_, err = funcCall(uf)
	require.Equal(t, gse.ErrWrongNumArguments, err)
}

func funcCall(fn core.NativeFunc, args ...core.Object) (core.Object, error) {
	userFunc := &value.BuiltinFunction{Value: fn}
	return userFunc.Call(nil, args...)
}

func array(elements ...core.Object) *value.Array {
	return &value.Array{Value: elements}
}
