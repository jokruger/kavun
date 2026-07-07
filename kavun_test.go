package kavun_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

const testOut = "out"

type MAP = map[string]any
type ARR = []any

type customError struct {
	err error
	str string
}

func (c *customError) Error() string {
	return c.str
}

func (c *customError) Unwrap() error {
	return c.err
}

func formatGlobals(globals []core.Value) (formatted []string) {
	for idx, global := range globals {
		if global.Type == value.Undefined {
			return
		}
		formatted = append(formatted, fmt.Sprintf("[% 3d] %s (%s|%v)", idx, global.String(), global.TypeName(), global))
	}
	return
}

type vmTracer struct {
	Out []string
}

func (o *vmTracer) Write(p []byte) (n int, err error) {
	o.Out = append(o.Out, string(p))
	return len(p), nil
}

func errorObject(v any) core.Value {
	if s, ok := v.(string); ok {
		return core.NewErrorValue(core.NewStringValue(s), core.KindUser, false)
	}
	return core.NewErrorValue(kavun.MustValueOf(v), core.KindUser, false)
}

func traceCompileRun(
	file *parser.File,
	symbols map[string]core.Value,
	customModules map[string][]byte,
	customBuiltinModules map[string]module,
) (res map[string]core.Value, trace []string, err error) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %v", e)

			// stack trace
			var stackTrace []string
			for i := 2; ; i += 1 {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				stackTrace = append(stackTrace, fmt.Sprintf("  %s:%d", file, line))
			}

			trace = append(trace, fmt.Sprintf("[Error Trace]\n\n  %s\n", strings.Join(stackTrace, "\n  ")))
		}
	}()

	globals := make([]core.Value, vm.GlobalsSize)

	symTable := compiler.NewSymbolTable()
	for name, value := range symbols {
		sym := symTable.Define(name)
		globals[sym.Index] = value
	}
	for idx, name := range vm.BuiltinFunctionNames {
		symTable.DefineBuiltin(idx, name)
	}

	idx := 0
	for name, mod := range customBuiltinModules {
		stdlib.InitModule(name, kavun.UsedDefinedModule+uint8(idx), mod.cs, mod.fns)
		idx++
	}
	defer func() {
		for name := range customBuiltinModules {
			stdlib.RemoveModule(name)
		}
	}()

	tr := &vmTracer{}
	c := compiler.NewCompiler(nil, nil, file.InputFile, symTable, nil, customModules, tr)
	err = c.Compile(file)
	trace = append(trace, fmt.Sprintf("\n[Compiler Trace]\n\n%s", strings.Join(tr.Out, "")))
	if err != nil {
		return
	}

	bytecode := c.Bytecode()
	trace = append(trace, fmt.Sprintf("\n[Compiled Constants]\n\n%s", strings.Join(bytecode.MustFormatStatics(), "\n")))
	trace = append(trace, fmt.Sprintf("\n[Compiled Instructions]\n\n%s\n", strings.Join(bytecode.MustFormatInstructions(), "\n")))

	machine.Reset(bytecode, globals)
	err = machine.Run()
	{
		res = make(map[string]core.Value)
		for name := range symbols {
			sym, depth, ok := symTable.Resolve(name, false)
			if !ok || depth != 0 {
				err = fmt.Errorf("symbol not found: %s", name)
				return
			}
			res[name] = globals[sym.Index]
		}
		trace = append(trace, fmt.Sprintf("\n[Globals]\n\n%s", strings.Join(formatGlobals(globals), "\n")))
	}
	if err == nil && !machine.IsStackEmpty() {
		err = errors.New("non empty stack after execution")
	}

	return
}

func parse(t *testing.T, input string) *parser.File {
	testFileSet := parser.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	p := parser.NewParser(testFile, []byte(input), nil)
	file, err := p.ParseFile()
	require.NoError(t, err)
	return file
}

type module struct {
	cs  map[string]core.Value
	fns map[uint64]*core.BuiltinFunction
}

type testOpts struct {
	customModules        map[string][]byte
	customBuiltinModules map[string]module
	symbols              map[string]core.Value
	skip2ndPass          bool
}

func Opts() *testOpts {
	return &testOpts{
		customModules:        make(map[string][]byte),
		customBuiltinModules: make(map[string]module),
		symbols:              make(map[string]core.Value),
		skip2ndPass:          false,
	}
}

func (o *testOpts) copy() *testOpts {
	c := &testOpts{
		customModules:        make(map[string][]byte),
		customBuiltinModules: make(map[string]module),
		symbols:              make(map[string]core.Value),
		skip2ndPass:          o.skip2ndPass,
	}
	maps.Copy(c.customModules, o.customModules)
	maps.Copy(c.customBuiltinModules, o.customBuiltinModules)
	maps.Copy(c.symbols, o.symbols)
	return c
}

func (o *testOpts) Module(name string, mod string) *testOpts {
	c := o.copy()
	c.customModules[name] = []byte(mod)
	return c
}

func (o *testOpts) BuiltinModule(name string, mod module) *testOpts {
	c := o.copy()
	c.customBuiltinModules[name] = mod
	return c
}

func (o *testOpts) Symbol(name string, value core.Value) *testOpts {
	c := o.copy()
	c.symbols[name] = value
	return c
}

func (o *testOpts) Skip2ndPass() *testOpts {
	c := o.copy()
	c.skip2ndPass = true
	return c
}

func expectErrorAs(t *testing.T, input string, opts *testOpts, expected any) {
	if opts == nil {
		opts = Opts()
	}

	// parse
	program := parse(t, input)
	if program == nil {
		return
	}

	// compiler/VM
	_, trace, err := traceCompileRun(program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, errors.As(err, expected), "expected error as: %v, got: %v\n%s", expected, err, strings.Join(trace, "\n"))
}

func expectErrorIs(t *testing.T, input string, opts *testOpts, expected error) {
	if opts == nil {
		opts = Opts()
	}

	// parse
	program := parse(t, input)
	if program == nil {
		return
	}

	// compiler/VM
	_, trace, err := traceCompileRun(program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, errors.Is(err, expected), "expected error is: %s, got: %s\n%s", expected.Error(), err.Error(), strings.Join(trace, "\n"))
}

func expectError(t *testing.T, input string, opts *testOpts, expected string) {
	if opts == nil {
		opts = Opts()
	}

	expected = strings.TrimSpace(expected)
	if expected == "" {
		panic("expected must not be empty")
	}

	// parse
	program := parse(t, input)
	if program == nil {
		return
	}

	// compiler/VM
	_, trace, err := traceCompileRun(program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, strings.Contains(err.Error(), expected), "expected error string: %s, got: %s\n%s", expected, err.Error(), strings.Join(trace, "\n"))
}

func expectRun(t *testing.T, input string, opts *testOpts, expected any) {
	if opts == nil {
		opts = Opts()
	}

	symbols := opts.symbols
	if symbols == nil {
		symbols = make(map[string]core.Value)
	}
	symbols[testOut] = core.Undefined

	// first pass: run the code normally
	{
		// parse
		file := parse(t, input)
		if file == nil {
			return
		}

		// compiler/VM
		res, trace, err := traceCompileRun(file, symbols, opts.customModules, opts.customBuiltinModules)
		require.NoError(t, err, "\n"+strings.Join(trace, "\n"))
		a := res[testOut]
		e := kavun.MustValueOf(expected)
		require.Equal(t, e, a, "\n"+strings.Join(trace, "\n"))
	}

	// second pass: run the code as import module
	if !opts.skip2ndPass {
		file := parse(t, `out = import("__code__")`)
		if file == nil {
			return
		}

		symbols[testOut] = core.Undefined
		modules := maps.Clone(opts.customModules)
		modules["__code__"] = []byte(fmt.Sprintf("out := undefined; %s; export out", input))
		res, trace, err := traceCompileRun(file, symbols, modules, opts.customBuiltinModules)
		require.NoError(t, err, "\n"+strings.Join(trace, "\n"))
		a := res[testOut]
		e := kavun.MustValueOf(expected)
		require.Equal(t, e, a, "\n"+strings.Join(trace, "\n"))
	}
}

func TestUndefined(t *testing.T) {
	expectRun(t, `out = undefined`, nil, core.Undefined)
	expectRun(t, `out = undefined.a`, nil, core.Undefined)
	expectRun(t, `out = undefined[1]`, nil, core.Undefined)
	expectRun(t, `out = undefined.a.b`, nil, core.Undefined)
	expectRun(t, `out = undefined[1][2]`, nil, core.Undefined)
	expectRun(t, `out = undefined ? 1 : 2`, nil, 2)
	expectRun(t, `out = undefined == undefined`, nil, true)
	expectRun(t, `out = undefined == 1`, nil, false)
	expectRun(t, `out = 1 == undefined`, nil, false)
	expectRun(t, `out = undefined == float([])`, nil, true)
	expectRun(t, `out = float([]) == undefined`, nil, true)
	expectRun(t, `out = undefined.format("v")`, nil, "undefined")

	u := core.Undefined
	s, _ := u.AsString()
	require.Equal(t, "", s)
	require.Equal(t, "undefined", u.String())

	expectRun(t, fmt.Sprintf(`out = undefined == %s`, u.String()), nil, true)
}

func TestBoolean(t *testing.T) {
	expectRun(t, `out = bool()`, nil, false)
	expectRun(t, `out = bool(true)`, nil, true)
	expectRun(t, `out = bool(false)`, nil, false)

	expectRun(t, `out = true`, nil, true)
	expectRun(t, `out = false`, nil, false)

	expectRun(t, `out = 1 < 2`, nil, true)
	expectRun(t, `out = 1 > 2`, nil, false)
	expectRun(t, `out = 1 < 1`, nil, false)
	expectRun(t, `out = 1 > 2`, nil, false)
	expectRun(t, `out = 1 == 1`, nil, true)
	expectRun(t, `out = 1 != 1`, nil, false)
	expectRun(t, `out = 1 == 2`, nil, false)
	expectRun(t, `out = 1 != 2`, nil, true)
	expectRun(t, `out = 1 <= 2`, nil, true)
	expectRun(t, `out = 1 >= 2`, nil, false)
	expectRun(t, `out = 1 <= 1`, nil, true)
	expectRun(t, `out = 1 >= 2`, nil, false)

	expectRun(t, `out = true == true`, nil, true)
	expectRun(t, `out = false == false`, nil, true)
	expectRun(t, `out = true == false`, nil, false)
	expectRun(t, `out = true != false`, nil, true)
	expectRun(t, `out = false != true`, nil, true)
	expectRun(t, `out = (1 < 2) == true`, nil, true)
	expectRun(t, `out = (1 < 2) == false`, nil, false)
	expectRun(t, `out = (1 > 2) == true`, nil, false)
	expectRun(t, `out = (1 > 2) == false`, nil, true)
	expectRun(t, `out = 5 + true`, nil, 6)
	expectRun(t, `out = 5 + true; 5`, nil, 6)

	expectError(t, `-true`, nil, "invalid_unary_operator: - bool")
	expectError(t, `true + false`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `5; true + false; 5`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `if (10 > 1) { true + false; }`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, `
func() {
	if (10 > 1) {
		if (10 > 1) {
			return true + false;
		}

		return 1;
	}
}()
`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, `if (true + false) { 10 }`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `10 + (true + false)`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `(true + false) + 20`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `!(true + false)`, nil, "invalid_binary_operator: bool + bool")

	var v core.Value

	v = core.True
	s, _ := v.AsString()
	require.Equal(t, "true", s)
	v = core.True
	require.Equal(t, "true", v.String())

	v = core.True
	expectRun(t, fmt.Sprintf(`out = true == %s`, v.String()), nil, true)
	v = core.False
	expectRun(t, fmt.Sprintf(`out = false == %s`, v.String()), nil, true)

	expectRun(t, `out = true.bool()`, nil, true)
	expectRun(t, `out = false.bool()`, nil, false)
	expectRun(t, `out = true.byte()`, nil, byte(1))
	expectRun(t, `out = false.byte()`, nil, byte(0))
	expectRun(t, `out = true.int()`, nil, 1)
	expectRun(t, `out = false.int()`, nil, 0)
	expectRun(t, `out = true.string()`, nil, "true")
	expectRun(t, `out = false.string()`, nil, "false")
	expectRun(t, `out = false.format()`, nil, "false")
	expectRun(t, `out = false.format("v")`, nil, "false")
}

func TestByte(t *testing.T) {
	var v core.Value

	expectRun(t, `out = byte(5)`, nil, byte(5))
	expectRun(t, `out = byte(true)`, nil, byte(1))
	expectRun(t, `out = byte(false)`, nil, byte(0))
	expectRun(t, `out = byte('A')`, nil, byte(65))
	expectRun(t, `out = byte("12")`, nil, byte(12))
	expectRun(t, `out = byte(u"12")`, nil, byte(12))
	expectRun(t, `out = byte(u"300", byte(7))`, nil, byte(7))
	expectRun(t, `out = byte(255) + 1`, nil, byte(0))
	expectRun(t, `out = byte(255) + 2`, nil, byte(1))
	expectRun(t, `out = byte(0) - 1`, nil, byte(255))
	expectRun(t, `out = 1 + byte(255)`, nil, int64(256))

	v = core.ByteValue(0)
	expectRun(t, fmt.Sprintf(`out = byte(0) == %s`, v.String()), nil, true)
	v = core.ByteValue(1)
	expectRun(t, fmt.Sprintf(`out = byte(1) == %s`, v.String()), nil, true)
	v = core.ByteValue(123)
	expectRun(t, fmt.Sprintf(`out = byte(123) == %s`, v.String()), nil, true)

	expectRun(t, `out = byte(123).int()`, nil, 123)
	expectRun(t, `out = byte(0).bool()`, nil, false)
	expectRun(t, `out = byte(10).bool()`, nil, true)
	expectRun(t, `out = byte(48).rune()`, nil, '0')
	expectRun(t, `out = byte(48).float()`, nil, 48.0)
	expectRun(t, `out = byte(48).string()`, nil, "48")
	expectRun(t, `out = byte(48).format()`, nil, "48")
	expectRun(t, `out = byte(48).format("v")`, nil, "byte(48)")
}

func TestInteger(t *testing.T) {
	var v core.Value

	expectRun(t, `out = 5`, nil, 5)
	expectRun(t, `out = 10`, nil, 10)
	expectRun(t, `out = -5`, nil, -5)
	expectRun(t, `out = -10`, nil, -10)
	expectRun(t, `out = 5 + 5 + 5 + 5 - 10`, nil, 10)
	expectRun(t, `out = 2 * 2 * 2 * 2 * 2`, nil, 32)
	expectRun(t, `out = -50 + 100 + -50`, nil, 0)
	expectRun(t, `out = 5 * 2 + 10`, nil, 20)
	expectRun(t, `out = 5 + 2 * 10`, nil, 25)
	expectRun(t, `out = 20 + 2 * -10`, nil, 0)
	expectRun(t, `out = 50 / 2 * 2 + 10`, nil, 60)
	expectRun(t, `out = 2 * (5 + 10)`, nil, 30)
	expectRun(t, `out = 3 * 3 * 3 + 10`, nil, 37)
	expectRun(t, `out = 3 * (3 * 3) + 10`, nil, 37)
	expectRun(t, `out = (5 + 10 * 2 + 15 /3) * 2 + -10`, nil, 50)
	expectRun(t, `out = 5 % 3`, nil, 2)
	expectRun(t, `out = 5 % 3 + 4`, nil, 6)
	expectRun(t, `out = +5`, nil, 5)
	expectRun(t, `out = +5 + -5`, nil, 0)

	expectRun(t, `out = 9 + '0'`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, `out = '9' - 5`, nil, 52) // '9' is 57 in ASCII

	v = core.IntValue(0)
	expectRun(t, fmt.Sprintf(`out = 0 == %s`, v.String()), nil, true)
	v = core.IntValue(1)
	expectRun(t, fmt.Sprintf(`out = 1 == %s`, v.String()), nil, true)
	v = core.IntValue(1234567890)
	expectRun(t, fmt.Sprintf(`out = 1234567890 == %s`, v.String()), nil, true)

	expectRun(t, `out = 5 + "-5"`, nil, 0)
	expectRun(t, `out = 5 + "5"`, nil, 10)

	expectRun(t, `out = (12).int()`, nil, 12)
	expectRun(t, `out = (0).bool()`, nil, false)
	expectRun(t, `out = (10).bool()`, nil, true)
	expectRun(t, `out = (48).rune()`, nil, '0')
	expectRun(t, `out = (48).float()`, nil, 48.0)
	expectRun(t, `out = (48).string()`, nil, "48")
	expectRun(t, `out = (1234567890).time().utc().string()`, nil, "2009-02-13 23:31:30 +0000 UTC")
	expectRun(t, `out = (48).byte()`, nil, byte(48))
	expectRun(t, `out = (48).format()`, nil, "48")
	expectRun(t, `out = (48).format("v")`, nil, "48")
}

func TestFloat(t *testing.T) {
	expectRun(t, `out = 0.0`, nil, 0.0)
	expectRun(t, `out = -10.3`, nil, -10.3)
	expectRun(t, `out = 3.2 + 2.0 * -4.0`, nil, -4.8)
	expectRun(t, `out = 4 + 2.3`, nil, 6.3)
	expectRun(t, `out = 2.3 + 4`, nil, 6.3)
	expectRun(t, `out = +5.0`, nil, 5.0)
	expectRun(t, `out = -5.0 + +5.0`, nil, 0.0)

	v := core.FloatValue(0.0)
	expectRun(t, fmt.Sprintf(`out = 0.0 == %s`, v.String()), nil, true)
	v = core.FloatValue(1.0)
	expectRun(t, fmt.Sprintf(`out = 1.0 == %s`, v.String()), nil, true)
	v = core.FloatValue(12345.6789)
	expectRun(t, fmt.Sprintf(`out = 12345.6789 == %s`, v.String()), nil, true)

	expectRun(t, `out = 5.0 + "-5.0"`, nil, 0.0)
	expectRun(t, `out = 5.0 + "5.0"`, nil, 10.0)

	expectRun(t, `out = (1.5).float()`, nil, 1.5)
	expectRun(t, `out = (1.5).int()`, nil, 1)
	expectRun(t, `out = (1.5).string()`, nil, "1.5")

	// f-suffix float literals
	expectRun(t, `out = 1f`, nil, 1.0)
	expectRun(t, `out = 1.5f`, nil, 1.5)
	expectRun(t, `out = type_name(1f)`, nil, "float")
	expectRun(t, `out = type_name(1.5f)`, nil, "float")
	expectRun(t, `out = 2f + 3f`, nil, 5.0)
}

func TestDecimal(t *testing.T) {
	expectRun(t, `out = decimal(123)`, nil, dec128.FromInt64(123))
	expectRun(t, `out = decimal(1.23)`, nil, dec128.FromFloat64(1.23))
	expectRun(t, `out = decimal("1.23")`, nil, dec128.FromString("1.23"))

	expectRun(t, `out = (123).decimal()`, nil, dec128.FromInt64(123))
	expectRun(t, `out = (1.23).decimal()`, nil, dec128.FromFloat64(1.23))
	expectRun(t, `out = "1.23".decimal()`, nil, dec128.FromString("1.23"))

	expectRun(t, `out = decimal(1) + decimal(2)`, nil, dec128.FromString("3"))
	expectRun(t, `out = decimal(1) + 2`, nil, dec128.FromString("3"))
	expectRun(t, `out = 1 + decimal(2)`, nil, dec128.FromString("3"))

	expectRun(t, `out = 1.0 + decimal(2)`, nil, 3.0)
	expectRun(t, `out = decimal(1) + 2.0`, nil, dec128.FromString("3"))

	expectRun(t, `out = 1d`, nil, dec128.FromInt64(1))
	expectRun(t, `out = 1.23d`, nil, dec128.FromString("1.23"))
	expectRun(t, `out = type_name(1d)`, nil, "decimal")
	expectRun(t, `out = type_name(1.23d)`, nil, "decimal")
	expectRun(t, `out = 1d + 2d`, nil, dec128.FromString("3"))
	expectRun(t, `out = 1d + 2`, nil, dec128.FromString("3"))
	expectRun(t, `out = 1 + 2d`, nil, dec128.FromString("3"))
	expectRun(t, `out = 1.5d + 0.5d`, nil, dec128.FromString("2"))
	expectRun(t, `out = -1d`, nil, dec128.FromInt64(-1))

	expectRun(t, `out = (1.23d).decimal()`, nil, dec128.FromString("1.23"))
	expectRun(t, `out = (123d).float().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, `out = (123d).int().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, `out = (1.23d).string()`, nil, "1.23")
	expectRun(t, `out = (1.23d).is_zero()`, nil, false)
	expectRun(t, `out = (0d).is_zero()`, nil, true)
	expectRun(t, `out = (0d).is_negative()`, nil, false)
	expectRun(t, `out = (1d).is_negative()`, nil, false)
	expectRun(t, `out = (-1d).is_negative()`, nil, true)
	expectRun(t, `out = (0d).is_positive()`, nil, false)
	expectRun(t, `out = (1d).is_positive()`, nil, true)
	expectRun(t, `out = (-1d).is_positive()`, nil, false)
	expectRun(t, `out = (0d).sign()`, nil, 0)
	expectRun(t, `out = (1d).sign()`, nil, 1)
	expectRun(t, `out = (-1d).sign()`, nil, -1)
	expectRun(t, `out = (123d).rescale(2).scale()`, nil, 2)
	expectRun(t, `out = (123d).rescale(2).canonical().scale()`, nil, 0)
	expectRun(t, `out = (1.23d).format()`, nil, "1.23")
	expectRun(t, `out = (1.23d).format("v")`, nil, "1.23d")
}

func TestRune(t *testing.T) {
	expectRun(t, `out = 'a'`, nil, 'a')
	expectRun(t, `out = 'あ'`, nil, rune(12354))
	expectRun(t, `out = 'Æ'`, nil, rune(198))

	expectRun(t, `out = '0' + '9'`, nil, rune(105))
	expectRun(t, `out = '0' + 9`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, `out = '9' - 4`, nil, 53) // '9' is 57 in ASCII
	expectRun(t, `out = '0' == '0'`, nil, true)
	expectRun(t, `out = '0' != '0'`, nil, false)
	expectRun(t, `out = '2' < '4'`, nil, true)
	expectRun(t, `out = '2' > '4'`, nil, false)
	expectRun(t, `out = '2' <= '4'`, nil, true)
	expectRun(t, `out = '2' >= '4'`, nil, false)
	expectRun(t, `out = '4' < '4'`, nil, false)
	expectRun(t, `out = '4' > '4'`, nil, false)
	expectRun(t, `out = '4' <= '4'`, nil, true)
	expectRun(t, `out = '4' >= '4'`, nil, true)

	v := core.RuneValue('A')
	s, _ := v.AsString()
	require.Equal(t, "A", s)
	v = core.RuneValue('A')
	require.Equal(t, "'A'", v.String())

	v = core.RuneValue('0')
	expectRun(t, fmt.Sprintf(`out = '0' == %s`, v.String()), nil, true)
	v = core.RuneValue('A')
	expectRun(t, fmt.Sprintf(`out = 'A' == %s`, v.String()), nil, true)
	v = core.RuneValue('₴')
	expectRun(t, fmt.Sprintf(`out = '₴' == %s`, v.String()), nil, true)
	v = core.RuneValue('\'')
	expectRun(t, fmt.Sprintf(`out = '\'' == %s`, v.String()), nil, true)

	expectRun(t, `out = '4' + 4`, nil, 56) // '4' is 52 in ASCII
	expectRun(t, `out = '4' + "4"`, nil, "44")
	expectError(t, `'4' - "4"`, nil, "invalid_binary_operator: rune - string")

	expectRun(t, `out = '4'.rune()`, nil, '4')
	expectRun(t, `out = '4'.bool()`, nil, true)
	expectRun(t, `out = '4'.int()`, nil, 52)
	expectRun(t, `out = '4'.string()`, nil, "4")
	expectRun(t, `out = '4'.format()`, nil, "4")
	expectRun(t, `out = '4'.format("v")`, nil, "'4'")
}

func TestString(t *testing.T) {
	expectRun(t, `out = "Hello World!"`, nil, "Hello World!")
	expectRun(t, `out = "Hello" + " " + "World!"`, nil, "Hello World!")

	expectRun(t, `out = "Hello" == "Hello"`, nil, true)
	expectRun(t, `out = "Hello" == "World"`, nil, false)
	expectRun(t, `out = "Hello" != "Hello"`, nil, false)
	expectRun(t, `out = "Hello" != "World"`, nil, true)

	expectRun(t, `out = "Hello" > "World"`, nil, false)
	expectRun(t, `out = "World" < "Hello"`, nil, false)
	expectRun(t, `out = "Hello" < "World"`, nil, true)
	expectRun(t, `out = "World" > "Hello"`, nil, true)
	expectRun(t, `out = "Hello" >= "World"`, nil, false)
	expectRun(t, `out = "Hello" <= "World"`, nil, true)
	expectRun(t, `out = "Hello" >= "Hello"`, nil, true)
	expectRun(t, `out = "World" <= "World"`, nil, true)
	expectRun(t, `out = "el" in "Hello"`, nil, true)
	expectRun(t, `out = "Hello".contains("el")`, nil, true)
	expectRun(t, `out = 'e' in "Hello"`, nil, true)
	expectRun(t, `out = "Hello".contains('e')`, nil, true)
	expectRun(t, `out = "z" in "Hello"`, nil, false)
	expectRun(t, `out = "Hello".contains("z")`, nil, false)
	expectRun(t, `out = "z" not in "Hello"`, nil, true)

	// index operator
	str := "abcdef"
	strStr := `"abcdef"`
	strLen := 6
	for idx := range strLen {
		expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, idx), nil, str[idx])
		expectRun(t, fmt.Sprintf("out = %s[0 + %d]", strStr, idx), nil, str[idx])
		expectRun(t, fmt.Sprintf("out = %s[1 + %d - 1]", strStr, idx), nil, str[idx])
		expectRun(t, fmt.Sprintf("idx = %d; out = %s[idx]", idx, strStr), nil, str[idx])
		expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, -idx-1), nil, str[strLen-idx-1])
	}

	expectError(t, fmt.Sprintf("%s[%d]", strStr, -strLen-1), nil, "index_out_of_bounds")
	expectError(t, fmt.Sprintf("%s[%d]", strStr, strLen), nil, "index_out_of_bounds")
	expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, -2), nil, str[strLen-2])

	// slice operator
	for low := 0; low <= strLen; low++ {
		expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, low, low), nil, "")
		for high := low; high <= strLen; high++ {
			expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, low, high), nil, str[low:high])
			expectRun(t, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", strStr, low, high), nil, str[low:high])
			expectRun(t, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", strStr, low, high), nil, str[low:high])
			expectRun(t, fmt.Sprintf("out = %s[:%d]", strStr, high), nil, str[:high])
			expectRun(t, fmt.Sprintf("out = %s[%d:]", strStr, low), nil, str[low:])
		}
	}

	expectRun(t, fmt.Sprintf("out = %s[:]", strStr), nil, str[:])
	expectRun(t, fmt.Sprintf("out = %s[:]", strStr), nil, str)
	expectRun(t, fmt.Sprintf("out = %s[%d:]", strStr, -1), nil, str[strLen-1:])
	expectRun(t, fmt.Sprintf("out = %s[:%d]", strStr, strLen+1), nil, str)
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 2), nil, "")
	expectRun(t, fmt.Sprintf("out = %s[:%d]", strStr, -1), nil, str[:strLen-1])
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, 0, -1), nil, str[:strLen-1])
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, -3, -1), nil, str[strLen-3:strLen-1])
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, 1, -1), nil, str[1:strLen-1])
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 1), nil, "")
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, 10, 20), nil, "")
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", strStr, -100, 100), nil, str)
	expectRun(t, fmt.Sprintf("out = %s[1:5:2]", strStr), nil, "bd")
	expectRun(t, fmt.Sprintf("out = %s[1:5:-1]", strStr), nil, "")
	expectRun(t, fmt.Sprintf("out = %s[5:1:-1]", strStr), nil, "fedc")
	expectRun(t, fmt.Sprintf("out = %s[0:%d:2]", strStr, strLen), nil, "ace")
	expectRun(t, fmt.Sprintf("out = %s[::-1]", strStr), nil, "fedcba")
	expectError(t, fmt.Sprintf("out = %s[::0]", strStr), nil, "step cannot be zero")

	// string concatenation with other types
	expectRun(t, `out = "foo" + 1`, nil, "foo1")
	// Float.string() returns the smallest number of digits necessary such that ParseFloat will return f exactly.
	expectRun(t, `out = "foo" + 1.0`, nil, "foo1") // <- note '1' instead of '1.0'
	expectRun(t, `out = "foo" + 1.5`, nil, "foo1.5")
	expectRun(t, `out = "foo" + true`, nil, "footrue")
	expectRun(t, `out = "foo" + 'X'`, nil, "fooX")
	expectRun(t, `out = "foo" + error(5)`, nil, "foo5")
	expectRun(t, `out = "foo" + [100, 101]`, nil, "foode")
	// also works with "+=" operator
	expectRun(t, `out = "foo"; out += 1.5`, nil, "foo1.5")

	// string concat works only when string is LHS
	expectError(t, `1 + "foo"`, nil, "invalid_binary_operator: int + string")

	// there is no '-' operator for string
	expectError(t, `"foo" - "bar"`, nil, "invalid_binary_operator: string - string")

	// undefined cannot be added to string
	expectError(t, `"foo" + undefined`, nil, "invalid_binary_operator: string + undefined")

	v := core.NewStringValue("abc")
	s, _ := v.AsString()
	require.Equal(t, "abc", s)
	v = core.NewStringValue("abc")
	require.Equal(t, `"abc"`, v.String())

	v = core.NewStringValue("")
	expectRun(t, fmt.Sprintf(`out = "" == %s`, v.String()), nil, true)
	v = core.NewStringValue("hello")
	expectRun(t, fmt.Sprintf(`out = "hello" == %s`, v.String()), nil, true)
	v = core.NewStringValue("hello \"world\"")
	expectRun(t, fmt.Sprintf(`out = "hello \"world\"" == %s`, v.String()), nil, true)
	v = core.NewStringValue("123₴")
	expectRun(t, fmt.Sprintf(`out = "123₴" == %s`, v.String()), nil, true)

	expectRun(t, `out = "".is_empty()`, nil, true)
	expectRun(t, `out = "abcd".is_empty()`, nil, false)
	expectRun(t, `out = "abcd".len()`, nil, 4)
	expectRun(t, `out = "Abcd".lower()`, nil, "abcd")
	expectRun(t, `out = "Abcd".upper()`, nil, "ABCD")
	expectRun(t, `out = "abcd ".trim()`, nil, "abcd")
	expectRun(t, `out = "abcd".trim("ad")`, nil, "bc")
	expectRun(t, `out = "".reverse()`, nil, "")
	expectRun(t, `out = "a".reverse()`, nil, "a")
	expectRun(t, `out = "hello".reverse()`, nil, "olleh")
	expectRun(t, `out = "їЇґҐ".reverse()`, nil, "ҐґЇї")
	expectRun(t, `out = "こんにちは".reverse()`, nil, "はちにんこ")

	expectRun(t, `out = "abc".string()`, nil, "abc")
	expectRun(t, `out = "abc".array()`, nil, ARR{int64('a'), int64('b'), int64('c')})
	expectRun(t, `out = "abc".array().string()`, nil, "abc")
	expectRun(t, `out = "true".bool()`, nil, true)
	expectRun(t, `out = "false".bool()`, nil, false)
	expectRun(t, `out = "abc".bool()`, nil, false)
	expectRun(t, `out = "true".bool().string()`, nil, "true")
	expectRun(t, `out = "abc".bytes()`, nil, core.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, `out = "abc".bytes().string()`, nil, "abc")
	expectRun(t, `out = "1.2".float()`, nil, 1.2)
	expectRun(t, `out = "1.2".float().string()`, nil, "1.2")
	expectRun(t, `out = "12".byte()`, nil, byte(12))
	expectRun(t, `out = u"12".byte()`, nil, byte(12))
	expectRun(t, `out = "12".int()`, nil, 12)
	expectRun(t, `out = "12".float().string()`, nil, "12")
	expectRun(t, `out = "abc".int()`, nil, 0)
	expectRun(t, `out = "abc".record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, `out = "abc".dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, `out = "abc".format()`, nil, "abc")
	expectRun(t, `out = "abc".format("v")`, nil, `"abc"`)

	expectRun(t, `out = " їЇґҐ ".trim()`, nil, "їЇґҐ")
	expectRun(t, `out = "їЇґҐ".upper()`, nil, "ЇЇҐҐ")
	expectRun(t, `out = "їЇґҐ".lower()`, nil, "їїґґ")
	expectRun(t, `out = "こんにちはさ"[1]`, nil, byte(129)) // byte index, not rune index
	expectRun(t, `out = "こんにちはさ"[1:2]`, nil, "\x81")  // byte slice, not rune slice
	expectRun(t, `out = "こんにちはさ"[0:3]`, nil, "こ")     // byte slice, not rune slice

	expectRun(t, `out = len("")`, nil, 0)
	expectRun(t, `out = len("hello")`, nil, 5)
	expectRun(t, `out = len("їЇґҐ")`, nil, 8)    // byte length, not rune length
	expectRun(t, `out = len("こんにちはさ")`, nil, 18) // byte length, not rune length

	expectRun(t, `out = "hello".filter(x => x > 'e')`, nil, "hllo")
	expectRun(t, `out = "hello".filter((i, x) => i > 2)`, nil, "lo")
	expectRun(t, `out = "hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, `out = "hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, `out = "hello".all(x => x > 'a')`, nil, true)
	expectRun(t, `out = "hello".all(x => x > 'e')`, nil, false)
	expectRun(t, `out = "hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, `out = "hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, `out = "hello".any(x => x == 'e')`, nil, true)
	expectRun(t, `out = "hello".any(x => x == 'z')`, nil, false)
	expectRun(t, `out = "hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, `out = "hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, `out = "hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, `out = "hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = "hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = "hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = "".find(x => true)`, nil, core.Undefined)
	expectError(t, `out = "x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `out = "x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `out = "x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, `
out = ""
ignored := "hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, `
out = 0
ignored := "abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunes(t *testing.T) {
	expectRun(t, `out = u"Hello World!"`, nil, []rune("Hello World!"))
	expectRun(t, `out = u"Hello" + u" " + "World!"`, nil, []rune("Hello World!"))

	expectRun(t, `out = u"Hello" == "Hello"`, nil, true)
	expectRun(t, `out = u"Hello" == u"Hello"`, nil, true)
	expectRun(t, `out = u"Hello" == u"World"`, nil, false)
	expectRun(t, `out = u"Hello" != u"Hello"`, nil, false)
	expectRun(t, `out = u"Hello" != u"World"`, nil, true)

	expectRun(t, `out = u"Hello" > u"World"`, nil, false)
	expectRun(t, `out = u"World" < u"Hello"`, nil, false)
	expectRun(t, `out = u"Hello" < u"World"`, nil, true)
	expectRun(t, `out = u"World" > u"Hello"`, nil, true)
	expectRun(t, `out = u"Hello" >= u"World"`, nil, false)
	expectRun(t, `out = u"Hello" <= u"World"`, nil, true)
	expectRun(t, `out = u"Hello" >= u"Hello"`, nil, true)
	expectRun(t, `out = u"World" <= u"World"`, nil, true)
	expectRun(t, `out = u"el" in u"Hello"`, nil, true)
	expectRun(t, `out = runes("Hello").contains(u"el")`, nil, true)
	expectRun(t, `out = 'e' in u"Hello"`, nil, true)
	expectRun(t, `out = runes("Hello").contains('e')`, nil, true)
	expectRun(t, `out = runes("z") in u"Hello"`, nil, false)
	expectRun(t, `out = runes("Hello").contains(u"z")`, nil, false)
	expectRun(t, `out = runes("z") not in u"Hello"`, nil, true)

	expectRun(t, `out = runes("").is_empty()`, nil, true)
	expectRun(t, `out = runes("abcd").is_empty()`, nil, false)
	expectRun(t, `out = runes("abcd").len()`, nil, 4)
	expectRun(t, `out = runes("abcd").first()`, nil, 'a')
	expectRun(t, `out = runes("abcd").last()`, nil, 'd')
	expectRun(t, `out = runes("Abcd").lower()`, nil, []rune("abcd"))
	expectRun(t, `out = runes("Abcd").upper()`, nil, []rune("ABCD"))
	expectRun(t, `out = runes("abcd ").trim()`, nil, []rune("abcd"))
	expectRun(t, `out = runes("abcd").trim("ad")`, nil, []rune("bc"))
	expectRun(t, `out = runes("").reverse()`, nil, []rune(""))
	expectRun(t, `out = runes("hello").reverse()`, nil, []rune("olleh"))
	expectRun(t, `out = u"hello".reverse()`, nil, []rune("olleh"))
	expectRun(t, `out = u"їЇґҐ".reverse()`, nil, []rune("ҐґЇї"))
	expectRun(t, `out = u"こんにちは".reverse()`, nil, []rune("はちにんこ"))

	expectRun(t, `out = runes("abc").string()`, nil, "abc")
	expectRun(t, `out = runes("abc").array()`, nil, ARR{'a', 'b', 'c'})
	expectRun(t, `out = runes("abc").array().string()`, nil, "abc")
	expectRun(t, `out = runes("true").bool()`, nil, true)
	expectRun(t, `out = runes("false").bool()`, nil, false)
	expectRun(t, `out = runes("abc").bool()`, nil, false)
	expectRun(t, `out = runes("true").bool().string()`, nil, "true")
	expectRun(t, `out = runes("abc").bytes()`, nil, core.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, `out = runes("abc").bytes().string()`, nil, "abc")
	expectRun(t, `out = runes("1.2").float()`, nil, 1.2)
	expectRun(t, `out = runes("1.2").float().string()`, nil, "1.2")
	expectRun(t, `out = runes("12").int()`, nil, 12)
	expectRun(t, `out = runes("12").float().string()`, nil, "12")
	expectRun(t, `out = runes("abc").int()`, nil, 0)
	expectRun(t, `out = runes("abc").record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, `out = runes("abc").dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})

	expectRun(t, `out = runes(" їЇґҐ ").trim()`, nil, []rune("їЇґҐ"))
	expectRun(t, `out = u" їЇґҐ ".trim()`, nil, []rune("їЇґҐ"))

	expectRun(t, `out = u"їЇґҐ".upper()`, nil, []rune("ЇЇҐҐ"))
	expectRun(t, `out = u"їЇґҐ".lower()`, nil, []rune("їїґґ"))
	expectRun(t, `out = u"їЇґҐ"[1]`, nil, 'Ї')
	expectRun(t, `out = u"їЇґҐ"[-1]`, nil, 'Ґ')
	expectRun(t, `out = u"їЇґҐ"[-2]`, nil, 'ґ')
	expectRun(t, `out = u"їЇґҐ"[1:2]`, nil, []rune("Ї"))
	expectRun(t, `out = u"їЇґҐ"[1:3]`, nil, []rune("Їґ"))
	expectRun(t, `out = u"їЇґҐ"[:-1]`, nil, []rune("їЇґ"))
	expectRun(t, `out = u"їЇґҐ"[1:-1]`, nil, []rune("Їґ"))
	expectRun(t, `out = u"їЇґҐ"[-3:-1]`, nil, []rune("Їґ"))
	expectRun(t, `out = u"їЇґҐ"[10:20]`, nil, []rune(""))
	expectRun(t, `out = u"їЇґҐ"[1:4:2]`, nil, []rune("ЇҐ"))
	expectRun(t, `out = u"їЇґҐ"[1:4:-1]`, nil, []rune(""))
	expectRun(t, `out = u"їЇґҐ"[3:0:-1]`, nil, []rune("ҐґЇ"))
	expectRun(t, `out = u"їЇґҐ"[0:4:2]`, nil, []rune("їґ"))
	expectRun(t, `out = u"їЇґҐ"[::-1]`, nil, []rune("ҐґЇї"))
	expectError(t, `out = u"їЇґҐ"[::0]`, nil, "step cannot be zero")
	expectRun(t, `out = u"こんにちはさ"[1]`, nil, 'ん')
	expectRun(t, `out = u"こんにちはさ"[1:2]`, nil, []rune("ん"))
	expectRun(t, `out = u"こんにちはさ"[1:3]`, nil, []rune("んに"))
	expectRun(t, `out = u"こんにちはさ"[-2:]`, nil, []rune("はさ"))
	expectError(t, `out = u"こんにちはさ"[-7]`, nil, "index_out_of_bounds")

	expectRun(t, `out = len(u"")`, nil, 0)
	expectRun(t, `out = len(u"hello")`, nil, 5)
	expectRun(t, `out = len(u"їЇґҐ")`, nil, 4)
	expectRun(t, `out = len(u"こんにちはさ")`, nil, 6)

	expectRun(t, `out = runes("abc").format()`, nil, "abc")
	expectRun(t, `out = runes("abc").format("v")`, nil, `u"abc"`)

	expectRun(t, `out = u"hello".sort()`, nil, []rune("ehllo"))
	expectRun(t, `out = u"".dedup()`, nil, []rune(""))
	expectRun(t, `out = u"aabbccd".dedup()`, nil, []rune("abcd"))
	expectRun(t, `out = u"abc".dedup()`, nil, []rune("abc"))
	expectRun(t, `out = u"aaaa".dedup()`, nil, []rune("a"))
	expectRun(t, `out = u"abab".dedup()`, nil, []rune("abab"))
	expectRun(t, `out = u"hello".sort().dedup()`, nil, []rune("ehlo"))
	expectRun(t, `out = u"їЇїЇ".dedup()`, nil, []rune("їЇїЇ"))
	expectRun(t, `out = u"їїЇЇ".dedup()`, nil, []rune("їЇ"))
	expectRun(t, `out = u"".unique()`, nil, []rune(""))
	expectRun(t, `out = u"abc".unique()`, nil, []rune("abc"))
	expectRun(t, `out = u"hello".unique()`, nil, []rune("helo"))
	expectRun(t, `out = u"abab".unique()`, nil, []rune("ab"))
	expectRun(t, `out = u"їЇїЇ".unique()`, nil, []rune("їЇ"))
	expectRun(t, `out = u"".chunk(2)`, nil, ARR{})
	expectRun(t, `out = u"hello".chunk(2)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, `out = u"hello".chunk(2, true)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, `out = u"hello".chunk(10)`, nil, ARR{[]rune("hello")})
	expectRun(t, `out = u"hello".filter(x => x > 'e')`, nil, []rune("hllo"))
	expectRun(t, `out = u"hello".filter((i, x) => i > 2)`, nil, []rune("lo"))
	expectRun(t, `out = u"hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, `out = u"hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, `out = u"hello".all(x => x > 'a')`, nil, true)
	expectRun(t, `out = u"hello".all(x => x > 'e')`, nil, false)
	expectRun(t, `out = u"hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, `out = u"hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, `out = u"hello".any(x => x == 'e')`, nil, true)
	expectRun(t, `out = u"hello".any(x => x == 'z')`, nil, false)
	expectRun(t, `out = u"hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, `out = u"hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, `out = u"hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, `out = u"hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = u"hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = u"hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = u"".find(x => true)`, nil, core.Undefined)
	expectError(t, `out = u"x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `out = u"x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `out = u"x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, `out = u"hello".min()`, nil, 'e')
	expectRun(t, `out = u"hello".max()`, nil, 'o')
	expectRun(t, `
out = ""
ignored := u"hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, `
out = 0
ignored := u"abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunesMutability(t *testing.T) {
	// index assignment
	expectRun(t, `r := runes("hello"); r[0] = 'H'; out = r`, nil, []rune("Hello"))
	expectRun(t, `r := runes("hello"); r[-2] = '!'; out = r`, nil, []rune("hel!o"))
	expectRun(t, `r := runes("hello"); r[0] = 0x41; out = r`, nil, []rune("Aello"))

	// append
	expectRun(t, `r := runes("ab"); r2 := append(r, 'c'); out = r2`, nil, []rune("abc"))
	expectRun(t, `r := runes("ab"); r2 := append(r, 'c', 'd'); out = r2`, nil, []rune("abcd"))
	expectRun(t, `r := runes("ab"); r2 := append(r, runes("cd")); out = r2`, nil, []rune("abcd"))
	expectRun(t, `r := runes("ab"); r2 := append(r, 'c'); out = r`, nil, []rune("ab"))

	// sum / avg / map / reduce
	expectRun(t, `out = runes("abc").sum()`, nil, 97+98+99)
	expectRun(t, `out = runes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, `out = runes("").sum()`, nil, core.Undefined)
	expectRun(t, `out = runes("").avg()`, nil, core.Undefined)
	expectRun(t, `out = runes("abc").map(func(r) { return r + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, `out = runes("abc").map(func(i, r) { return [i, r] })`, nil, ARR{ARR{0, 'a'}, ARR{1, 'b'}, ARR{2, 'c'}})
	expectRun(t, `out = runes("abc").reduce(0, func(acc, r) { return acc + r })`, nil, int64('a'+'b'+'c'))
	expectRun(t, `out = runes("abc").reduce("", func(acc, i, r) { return acc + i.string() + r.string() })`, nil, "0a1b2c")

	// type names
	expectRun(t, `out = type_name(runes("abc"))`, nil, "runes")
	expectRun(t, `out = type_name(immutable(runes("abc")))`, nil, "immutable-runes")

	// immutable rejects writes
	expectError(t, `r := immutable(runes("abc")); r[0] = 'X'`, nil, "not_assignable: type immutable-runes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, `out = type_name(immutable(runes("abcd"))[1:3])`, nil, "immutable-runes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, `out = type_name(immutable(runes("abcd"))[::-1])`, nil, "runes")
	// slice of mutable stays mutable
	expectRun(t, `out = type_name(runes("abcd")[1:3])`, nil, "runes")

	// copy of immutable yields mutable
	expectRun(t, `r := immutable(runes("abc")); c := copy(r); c[0] = 'X'; out = c`, nil, []rune("Xbc"))

	// append on immutable returns a fresh mutable value (does not mutate source)
	expectRun(t, `r := immutable(runes("ab")); r2 := append(r, 'c'); r2[0] = 'X'; out = r2`, nil, []rune("Xbc"))
	expectRun(t, `r := immutable(runes("ab")); r2 := append(r, 'c'); out = type_name(r2)`, nil, "runes")

	// invalid assignment values
	expectError(t, `r := runes("abc"); r[0] = "xy"`, nil, "invalid_index_type: (index assign value) expected rune, got string")
	expectError(t, `r := runes("abc"); r[10] = 'X'`, nil, "index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestError(t *testing.T) {
	expectError(t, `out = error()`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 0")
	expectRun(t, `out = error(1)`, nil, errorObject(1))
	expectRun(t, `out = error(1).value()`, nil, 1)
	expectRun(t, `out = error("some error")`, nil, errorObject("some error"))
	expectRun(t, `out = error("some" + " error")`, nil, errorObject("some error"))
	expectRun(t, `out = func() { return error(5) }()`, nil, errorObject(5))
	expectRun(t, `out = error(error("foo"))`, nil, errorObject(errorObject("foo")))
	expectRun(t, `out = error("some error")`, nil, errorObject("some error"))
	expectRun(t, `out = error("some error").value()`, nil, "some error")
	expectRun(t, `out = error("some error").string()`, nil, "some error")
	expectRun(t, `out = error("some error").format()`, nil, "some error")
	expectRun(t, `out = error("some error").format("v")`, nil, `error("some error")`)

	expectRun(t, `out = error("x").is_fatal()`, nil, false)
	expectRun(t, `out = error("x", false).is_fatal()`, nil, false)
	expectRun(t, `out = error("x", true).is_fatal()`, nil, true)
	expectError(t, `out = error("x").is_fatal(1)`, nil, "wrong_num_arguments: (is_fatal) expected 0 argument(s), got 1")

	expectError(t, `error("error").err`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, `error("error").value_`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, `error([1,2,3])[1]`, nil, "not_accessible: type error does not support indexing or field access")

	s, _ := core.NewErrorValue(core.NewStringValue("abc"), core.KindUser, false).AsString()
	require.Equal(t, "abc", s)
	require.Equal(t, `error("abc")`, core.NewErrorValue(core.NewStringValue("abc"), core.KindUser, false).String())

	v := core.NewErrorValue(core.Undefined, core.KindUser, false)
	require.Equal(t, "error()", v.String())
	expectRun(t, `out = error(undefined) == error(undefined)`, nil, true)
	v = core.NewErrorValue(core.NewStringValue("some error"), core.KindUser, false)
	expectRun(t, fmt.Sprintf(`out = error("some error") == %s`, v.String()), nil, true)
}

func TestArray(t *testing.T) {
	expectRun(t, `out = [1, 2 * 2, 3 + 3]`, nil, ARR{1, 4, 6})

	// array copy-by-reference
	expectRun(t, `a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2`, nil, ARR{5, 2, 3})
	expectRun(t, `func () { a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2 }()`, nil, ARR{5, 2, 3})

	// array index set
	expectError(t, `a1 := [1, 2, 3]; a1[3] = 5`, nil, "index_out_of_bounds")

	// index operator
	arr := ARR{1, 2, 3, 4, 5, 6}
	arrStr := `[1, 2, 3, 4, 5, 6]`
	arrLen := 6
	for idx := 0; idx < arrLen; idx++ {
		expectRun(t, fmt.Sprintf("out = %s[%d]", arrStr, idx), nil, arr[idx])
		expectRun(t, fmt.Sprintf("out = %s[0 + %d]", arrStr, idx), nil, arr[idx])
		expectRun(t, fmt.Sprintf("out = %s[1 + %d - 1]", arrStr, idx), nil, arr[idx])
		expectRun(t, fmt.Sprintf("idx := %d; out = %s[idx]", idx, arrStr), nil, arr[idx])
		expectRun(t, fmt.Sprintf("out = %s[%d]", arrStr, -idx-1), nil, arr[arrLen-idx-1])
	}

	expectError(t, fmt.Sprintf("%s[%d]", arrStr, -arrLen-1), nil, "index_out_of_bounds")
	expectError(t, fmt.Sprintf("%s[%d]", arrStr, arrLen), nil, "index_out_of_bounds")
	expectRun(t, fmt.Sprintf("out = %s[%d]", arrStr, -2), nil, arr[arrLen-2])
	expectRun(t, `a1 := [1, 2, 3]; a1[-1] = 5; out = a1[2]`, nil, 5)
	expectError(t, `a1 := [1, 2, 3]; a1[-4] = 5`, nil, "index_out_of_bounds")

	// slice operator
	for low := 0; low < arrLen; low++ {
		expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, low), nil, ARR{})
		for high := low; high <= arrLen; high++ {
			expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, fmt.Sprintf("out = %s[:%d]", arrStr, high), nil, arr[:high])
			expectRun(t, fmt.Sprintf("out = %s[%d:]", arrStr, low), nil, arr[low:])
		}
	}

	expectRun(t, fmt.Sprintf("out = %s[:]", arrStr), nil, arr)
	expectRun(t, fmt.Sprintf("out = %s[%d:]", arrStr, -1), nil, ARR{6})
	expectRun(t, fmt.Sprintf("out = %s[:%d]", arrStr, arrLen+1), nil, arr)
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 2), nil, ARR{})
	expectRun(t, fmt.Sprintf("out = %s[:%d]", arrStr, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, 0, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, 1, -1), nil, ARR{2, 3, 4, 5})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, -3, -1), nil, ARR{4, 5})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 1), nil, ARR{})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, 10, 20), nil, ARR{})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d]", arrStr, -100, 100), nil, arr)
	expectRun(t, fmt.Sprintf("out = %s[1:5:2]", arrStr), nil, ARR{2, 4})
	expectRun(t, fmt.Sprintf("out = %s[1:5:-1]", arrStr), nil, ARR{})
	expectRun(t, fmt.Sprintf("out = %s[5:1:-1]", arrStr), nil, ARR{6, 5, 4, 3})
	expectRun(t, fmt.Sprintf("out = %s[%d:%d:%d]", arrStr, 0, arrLen, 2), nil, ARR{1, 3, 5})
	expectRun(t, fmt.Sprintf("out = %s[::-1]", arrStr), nil, ARR{6, 5, 4, 3, 2, 1})
	expectError(t, fmt.Sprintf("out = %s[::0]", arrStr), nil, "step cannot be zero")

	v := core.NewArrayValue(nil, false)
	expectRun(t, fmt.Sprintf(`out = [] == %s`, v.String()), nil, true)
	v = core.NewArrayValue(nil, true)
	expectRun(t, fmt.Sprintf(`out = [] == %s`, v.String()), nil, true)

	v = core.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.Undefined,
		core.NewStringValue("3"),
	}, false)
	expectRun(t, fmt.Sprintf(`out = [1, undefined, "3"] == %s`, v.String()), nil, true)

	expectError(t, `[1, 2, 3].q`, nil, "Runtime Error: invalid_selector: type array has no property \"q\"\n\tat test:1:11")

	expectRun(t, `t := []; out = t.sort()`, nil, ARR{})
	expectRun(t, `t := [1, 2, 3]; out = t.sort()`, nil, ARR{1, 2, 3})
	expectRun(t, `t := [3, 2, 1]; out = t.sort()`, nil, ARR{1, 2, 3})

	expectRun(t, `out = [].dedup()`, nil, ARR{})
	expectRun(t, `out = [1].dedup()`, nil, ARR{1})
	expectRun(t, `out = [1, 1, 2, 2, 3, 3, 3, 1].dedup()`, nil, ARR{1, 2, 3, 1})
	expectRun(t, `out = [1, 2, 3].dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [1, 2, 1, 2].dedup()`, nil, ARR{1, 2, 1, 2})
	expectRun(t, `out = [3, 1, 2, 1, 3, 2].sort().dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = ["a", "a", "b", "a"].dedup()`, nil, ARR{"a", "b", "a"})
	expectRun(t, `out = [1, 1.0, "1"].dedup()`, nil, ARR{1})
	expectRun(t, `out = [[1, 2], [1, 2], [3]].dedup()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, `out = [].unique()`, nil, ARR{})
	expectRun(t, `out = [1].unique()`, nil, ARR{1})
	expectRun(t, `out = [1, 2, 3].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [1, 1, 2, 2, 3, 3, 3, 1].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [3, 1, 2, 1, 3, 2].unique()`, nil, ARR{3, 1, 2})
	expectRun(t, `out = ["a", "b", "a", "c", "b"].unique()`, nil, ARR{"a", "b", "c"})
	expectRun(t, `out = [1, 1.0, "1"].unique()`, nil, ARR{1})
	expectRun(t, `out = [[1, 2], [3], [1, 2]].unique()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, `out = [].reverse()`, nil, ARR{})
	expectRun(t, `out = [1].reverse()`, nil, ARR{1})
	expectRun(t, `out = [1, 2, 3].reverse()`, nil, ARR{3, 2, 1})
	expectRun(t, `out = ["a", "b", "c"].reverse()`, nil, ARR{"c", "b", "a"})
	expectRun(t, `out = [1, 2, 3].reverse().reverse()`, nil, ARR{1, 2, 3})

	expectRun(t, `t := []; out = t.is_empty()`, nil, true)
	expectRun(t, `t := [1, 2, 3]; out = t.is_empty()`, nil, false)

	expectRun(t, `t := []; out = t.len()`, nil, 0)
	expectRun(t, `t := [1, 2, 3]; out = t.len()`, nil, 3)

	expectRun(t, `out = [].first()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].first()`, nil, 1)

	expectRun(t, `out = [].last()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].last()`, nil, 3)

	expectRun(t, `out = [].min()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].min()`, nil, 1)

	expectRun(t, `out = [].max()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].max()`, nil, 3)

	expectRun(t, `out = [].sum()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].sum()`, nil, 6)

	expectRun(t, `out = [].avg()`, nil, core.Undefined)
	expectRun(t, `out = [1, 2, 3].avg()`, nil, 2)

	expectRun(t, `out = [].count(x => x > 0)`, nil, 0)
	expectRun(t, `out = [1, 2, 3, -10].count(x => x > 0)`, nil, 3)
	expectRun(t, `out = [1, 2, 3, -10].count((i, x) => x == i+1)`, nil, 3)

	expectRun(t, `out = [1, 2, 3].filter(x => x == 2)`, nil, ARR{2})
	expectRun(t, `out = [1, 2, 3].filter(x => x != 2)`, nil, ARR{1, 3})
	expectRun(t, `out = [1, undefined, 2, undefined, 3].filter()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [].filter()`, nil, ARR{})
	expectRun(t, `out = [undefined, undefined].filter()`, nil, ARR{})

	expectRun(t, `out = [].all(x => x > 0)`, nil, true)
	expectRun(t, `out = [1, 2, 3, -10].all(x => x > 0)`, nil, false)
	expectRun(t, `out = [1, 2, 3, -10].all(x => x > -100)`, nil, true)
	expectRun(t, `out = [1, 2, 3, -10].all((i, x) => x == i+1)`, nil, false)
	expectRun(t, `out = [1, 2, 3, 4].all((i, x) => x == i+1)`, nil, true)

	expectRun(t, `out = [].any(x => x > 0)`, nil, false)
	expectRun(t, `out = [1, 2, 3, -10].any(x => x < 0)`, nil, true)
	expectRun(t, `out = [1, 2, 3, -10].any(x => x < -100)`, nil, false)
	expectRun(t, `out = [1, 2, 3, -10].any((i, x) => x != i+1)`, nil, true)
	expectRun(t, `out = [1, 2, 3, 4].any((i, x) => x != i+1)`, nil, false)

	expectRun(t, `out = [].map(x => x * x)`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3].map(x => x * x)`, nil, ARR{1, 4, 9})

	expectRun(t, `out = [].chunk(2)`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3, 4].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}})
	expectRun(t, `out = [1, 2, 3, 4, 5].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}, ARR{5}})
	expectRun(t, `out = [1, 2, 3].chunk(10)`, nil, ARR{ARR{1, 2, 3}})
	expectRun(t, `a := [1, 2, 3]; c := a.chunk(2); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, `a := [1, 2, 3]; c := a.chunk(2, false); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, `a := [1, 2, 3]; c := a.chunk(2, true); c[0][0] = 9; out = a`, nil, ARR{1, 2, 3})
	expectError(t, `out = [1, 2, 3].chunk()`, nil, "wrong_num_arguments: (chunk) expected 1 or 2 argument(s), got 0")
	expectError(t, `out = [1, 2, 3].chunk("x")`, nil, "invalid_argument_type: (chunk) argument first expects type int, got string")
	expectError(t, `out = [1, 2, 3].chunk(2, 1)`, nil, "invalid_argument_type: (chunk) argument second expects type bool, got int")
	expectError(t, `out = [1, 2, 3].chunk(0)`, nil, "invalid_value: chunk size must be positive")
	expectError(t, `out = [1, 2, 3].chunk(-1)`, nil, "invalid_value: chunk size must be positive")

	expectRun(t, `
out = 0
ignored := [1, 2, 3, 4].for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)

	expectRun(t, `
out = 0
ignored := [10, 20, 30].for_each(func(i, v) {
	out += i * v
	return true
})
`, nil, 80)

	expectRun(t, `out = [1].for_each(func(v) { return true })`, nil, core.Undefined)
	expectError(t, `out = [1].for_each()`, nil, "wrong_num_arguments: (for_each) expected 1 argument(s), got 0")
	expectError(t, `out = [1].for_each(1)`, nil, "invalid_argument_type: (for_each) argument first expects type non-variadic function, got int")
	expectError(t, `out = [1].for_each(func() { return true })`, nil, "invalid_argument_type: (for_each) argument first expects type f/1 or f/2")

	expectRun(t, `out = [10, 20, 30].find(x => x == 20)`, nil, 1)
	expectRun(t, `out = [10, 20, 30].find(x => x == 99)`, nil, core.Undefined)
	expectRun(t, `out = [10, 20, 30].find((i, v) => i == 2)`, nil, 2)
	expectRun(t, `out = [10, 20, 30].find((i, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, `out = [].find(x => true)`, nil, core.Undefined)
	expectError(t, `out = [1].find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `out = [1].find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `out = [1].find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, `out = [].reduce(0, (a, v) => a + v)`, nil, 0)
	expectRun(t, `out = [1, 2, 3].reduce(0, (a, v) => a + v)`, nil, 6)
	expectRun(t, `out = [1, 2, 3].reduce(0, (a, i, v) => a + i)`, nil, 3)
	expectRun(t, `out = [1, 2].reduce(0, (a, v) => a + [10, 20].reduce(0, (b, w) => b + w) + v)`, nil, 63)

	expectRun(t, `out = [1, 2, 3].array()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [48, 49, -1].bytes()`, nil, core.NewBytesValue([]byte{48, 49, 255}, false))
	expectRun(t, `out = [48, 49, -1].record()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, `out = [48, 49, -1].dict()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, `out = [48, 49, 50].string()`, nil, "012")
	expectRun(t, `out = [48, 49, 50].format("v")`, nil, "[48, 49, 50]")
	expectRun(t, `out = [48, 49, 50].format()`, nil, "[48, 49, 50]")

	expectRun(t, `out = 2 in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [1, 2, 3].contains(2)`, nil, true)
	expectRun(t, `out = "2" in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [1, 2, 3].contains("2")`, nil, true)
	expectRun(t, `out = "z" in [1, 2, 3]`, nil, false)
	expectRun(t, `out = [1, 2, 3].contains("z")`, nil, false)
	expectRun(t, `out = [2, 3] in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [1, 2, 3].contains([2, 3])`, nil, true)
	expectRun(t, `out = [] in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [1, 2, 3].contains([])`, nil, true)
	expectRun(t, `out = [1, 3] in [1, 2, 3]`, nil, false)
	expectRun(t, `out = [1, 2, 3].contains([1, 3])`, nil, false)
	expectRun(t, `out = [1, 3] not in [1, 2, 3]`, nil, true)
}

func TestRecord(t *testing.T) {
	expectRun(t, `
out = {
	one: 10 - 9,
	two: 1 + 1,
	three: 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, `
out = {
	"one": 10 - 9,
	"two": 1 + 1,
	"three": 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, `out = {foo: 5}["foo"]`, nil, 5)
	expectRun(t, `out = {foo: 5}["bar"]`, nil, core.Undefined)
	expectRun(t, `key := "foo"; out = {foo: 5}[key]`, nil, 5)
	expectRun(t, `out = {}["foo"]`, nil, core.Undefined)

	expectRun(t, `
m := {
	foo: func(x) {
		return x * 2
	}
}
out = m["foo"](2) + m["foo"](3)
`, nil, 10)

	expectRun(t, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1`, nil, 5)
	expectRun(t, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1`, nil, 3)
	expectRun(t, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1 }()`, nil, 5)
	expectRun(t, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1 }()`, nil, 3)

	v := core.NewRecordValue(nil, false)
	expectRun(t, fmt.Sprintf(`out = {} == %s`, v.String()), nil, true)
	v = core.NewRecordValue(nil, true)
	expectRun(t, fmt.Sprintf(`out = {} == %s`, v.String()), nil, true)

	v = core.NewRecordValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": core.NewStringValue("3"),
	}, false)
	expectRun(t, fmt.Sprintf(`out = {a: 1, b: undefined, c: "3"} == %s`, v.String()), nil, true)

	expectRun(t, `out = {a: 1, b: 2}["b"]`, nil, 2)
	expectRun(t, `out = {a: 1, b: 2}["q"]`, nil, core.Undefined)
	expectRun(t, `out = {a: 1, b: 2}.b`, nil, 2)
	expectRun(t, `out = {a: 1, b: 2}.q`, nil, core.Undefined)
	expectRun(t, `out = "a" in {a: 1, b: 2}`, nil, true)
	expectRun(t, `out = "q" in {a: 1, b: 2}`, nil, false)
	expectRun(t, `out = "q" not in {a: 1, b: 2}`, nil, true)
	expectRun(t, `t := {a: 1, b: 2}; t["a"] = 3; out = t.a`, nil, 3)
	expectRun(t, `t := {a: 1, b: 2}; t.a = 3; out = t["a"]`, nil, 3)
}

func TestDict(t *testing.T) {
	expectRun(t, fmt.Sprintf(`out = dict() == %s`, core.NewDictValue(nil, false).String()), nil, true)
	expectRun(t, fmt.Sprintf(`out = dict() == %s`, core.NewDictValue(nil, true).String()), nil, true)

	expectRun(t, fmt.Sprintf(`out = dict({a: 1, b: undefined, c: "3"}) == %s`, core.NewDictValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": core.NewStringValue("3"),
	}, false).String()), nil, true)

	expectRun(t, `out = dict({a: 1, b: 2})["b"]`, nil, 2)
	expectRun(t, `out = dict({a: 1, b: 2}).record().b`, nil, 2)
	expectRun(t, `out = dict({a: 1, b: 2})["q"]`, nil, core.Undefined)
	expectRun(t, `out = "a" in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, `out = "q" in dict({a: 1, b: 2})`, nil, false)
	expectRun(t, `out = "q" not in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, `t := dict({a: 1, b: 2}); t["a"] = 3; out = t["a"]`, nil, 3)
	expectError(t, `dict({a: 1, b: 2}).q`, nil, "Runtime Error: invalid_selector: type dict has no property q\n\tat test:1:20")

	expectRun(t, `t := dict({a: 1, b: 2}); out = t.is_empty()`, nil, false)
	expectRun(t, `t := dict(); out = t.is_empty()`, nil, true)

	expectRun(t, `t := dict({a: 1, b: 2}); out = t.len()`, nil, 2)
	expectRun(t, `t := dict(); out = t.len()`, nil, 0)

	expectRun(t, `t := dict({a: 1, b: 2}); out = t.keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, `t := dict({a: 1, b: 2}); out = t.values().sort()`, nil, ARR{1, 2})

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.filter(k => k != "b").keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.filter((k, v) => v > 1).keys().sort()`, nil, ARR{"b", "c"})
	expectRun(t, `t := dict({a: 1, b: undefined, c: 3, d: undefined}); out = t.filter().keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, `t := dict(); out = t.filter().len()`, nil, 0)

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.count(k => k != "b")`, nil, 2)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.count((k, v) => v > 1)`, nil, 2)

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "b")`, nil, false)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "q")`, nil, true)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 1)`, nil, false)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 0)`, nil, true)

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "b")`, nil, true)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "q")`, nil, false)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 1)`, nil, true)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 10)`, nil, false)

	expectRun(t, `
out = 0
d = dict({a: 1, b: 2, c: 3})
ignored = d.for_each(func(k) {
	out += d[k]
	return true
})
`, nil, 6)

	expectRun(t, `
items = []
ignored = dict({a: 1, b: 2}).for_each(func(k, v) {
	items = append(items, k + v.string())
	return true
})
out = items.sort()
`, nil, ARR{"a1", "b2"})

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "b")`, nil, "b")
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "q")`, nil, core.Undefined)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 2)`, nil, "b")
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, `t := dict(); out = t.find(k => true)`, nil, core.Undefined)
	expectError(t, `dict({a: 1}).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `dict({a: 1}).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `dict({a: 1}).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, `out = "a" in dict({a: 1, b: 2, c: 3})`, nil, true)
	expectRun(t, `out = dict({a: 1, b: 2, c: 3}).contains("a")`, nil, true)
	expectRun(t, `out = "q" in dict({a: 1, b: 2, c: 3})`, nil, false)
	expectRun(t, `out = dict({a: 1, b: 2, c: 3}).contains("q")`, nil, false)
	expectRun(t, `out = "q" not in dict({a: 1, b: 2, c: 3})`, nil, true)

	//there is a problem with keys order (it is random) so we cannot test it now
	//expectRun(t, `out = dict({a: 1, b: 2}).format("v")`, nil, `dict({"a": 1, "b": 2})`)
	//expectRun(t, `out = dict({a: 1, b: 2}).format()`, nil, `dict({"a": 1, "b": 2})`)
}

func TestTime(t *testing.T) {
	o := core.NewTimeValue(time.Date(2020, 6, 20, 1, 2, 3, 4, time.UTC))
	s, _ := o.AsString()
	require.Equal(t, "2020-06-20 01:02:03.000000004 +0000 UTC", s)
	require.Equal(t, `time("2020-06-20T01:02:03.000000004Z")`, o.String())

	expectRun(t, `out = t"2020-06-20T01:02:03.000000004Z"`, nil, time.Date(2020, 6, 20, 1, 2, 3, 4, time.UTC))
	expectRun(t, `out = t"2020-06-20T01:02:03.000000004Z" == time("2020-06-20 01:02:03.000000004 UTC")`, nil, true)
	expectRun(t, `out = t"2020-06-20T01:02:03.000000004Z".year()`, nil, 2020)

	expectRun(t, fmt.Sprintf(`out = time("2020-06-20 01:02:03.000000004 UTC") == %s`, o.String()), nil, true)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").year()`, nil, 2020)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").month()`, nil, 6)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").day()`, nil, 20)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").hour()`, nil, 1)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").minute()`, nil, 2)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").second()`, nil, 3)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").nanosecond()`, nil, 4)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix()`, nil, 1592614923)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_nano()`, nil, 1592614923000000004)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day()`, nil, 6)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day_name()`, nil, "Saturday")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").month_name()`, nil, "June")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").year_day()`, nil, 172) // June 20 is the 172nd day of the year (173rd in leap years)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format_date()`, nil, "2020-06-20")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format_time()`, nil, "01:02:03")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format_datetime()`, nil, "2020-06-20 01:02:03")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").utc().string()`, nil, "2020-06-19 23:02:03.000000004 +0000 UTC")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").zone_offset()`, nil, 7200)

	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").string()`, nil, "2020-06-20 01:02:03.000000004 +0200 +0200")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").int().time().utc().string()`, nil, "2020-06-19 23:02:03 +0000 UTC")

	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format()`, nil, "2020-06-20T01:02:03+02:00")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format("v")`, nil, `time("2020-06-20T01:02:03.000000004+02:00")`)
}

func TestDictRecord(t *testing.T) {
	expectRun(t, `out = len({})`, nil, 0)
	expectRun(t, `out = len(dict())`, nil, 0)
	expectRun(t, `out = len(dict({}))`, nil, 0)

	expectRun(t, `out = len({a: 1})`, nil, 1)
	expectRun(t, `out = len(dict({a: 1}))`, nil, 1)

	expectRun(t, `out = len({a: 1, b: 2})`, nil, 2)
	expectRun(t, `out = len(dict({a: 1, b: 2}))`, nil, 2)

	expectRun(t, `out = dict() == ""`, nil, false)
	expectRun(t, `out = dict() == {}`, nil, true)
	expectRun(t, `out = dict({a: 1}) == {a: 1}`, nil, true)
	expectRun(t, `out = dict({a: 1}) == {a: 1, b: 1}`, nil, false)

	expectRun(t, `out = {a: 1}["a"]`, nil, 1)
	expectRun(t, `out = {a: 1}.a`, nil, 1)

	expectRun(t, `out = dict({a: 1})["a"]`, nil, 1)
}

func TestBytes(t *testing.T) {
	expectRun(t, `out = b'A'`, nil, byte(65))
	expectRun(t, `out = b'\x00'`, nil, byte(0))
	expectRun(t, `out = b'\n'`, nil, byte('\n'))

	expectRun(t, `out = b"Hello World!"`, nil, []byte("Hello World!"))
	expectRun(t, `out = b"Hello" + b" " + b"World!"`, nil, []byte("Hello World!"))
	expectRun(t, `out = b"abc" == bytes("abc")`, nil, true)
	expectRun(t, `out = b"abc"[1]`, nil, byte(98))

	expectRun(t, `out = bytes("Hello World!")`, nil, []byte("Hello World!"))
	expectRun(t, `out = bytes("Hello") + bytes(" ") + bytes("World!")`, nil, []byte("Hello World!"))

	// bytes[] -> byte
	expectRun(t, `out = bytes("abcde")[0]`, nil, byte(97))
	expectRun(t, `out = bytes("abcde")[1]`, nil, byte(98))
	expectRun(t, `out = bytes("abcde")[4]`, nil, byte(101))
	expectRun(t, `out = bytes("abcde")[-1]`, nil, byte(101))
	expectRun(t, `out = bytes("abcde")[-2]`, nil, byte(100))
	expectError(t, `out = bytes("abcde")[-6]`, nil, "index_out_of_bounds")
	expectError(t, `out = bytes("abcde")[10]`, nil, "index_out_of_bounds")

	// bytes[a:b] -> bytes
	expectRun(t, `out = bytes("abcde")[1:4]`, nil, []byte("bcd"))
	expectRun(t, `out = bytes("abcde")[:-1]`, nil, []byte("abcd"))
	expectRun(t, `out = bytes("abcde")[1:-1]`, nil, []byte("bcd"))
	expectRun(t, `out = bytes("abcde")[-2:]`, nil, []byte("de"))
	expectRun(t, `out = bytes("abcde")[-3:-1]`, nil, []byte("cd"))
	expectRun(t, `out = bytes("abcde")[3:1]`, nil, []byte{})
	expectRun(t, `out = bytes("abcde")[10:20]`, nil, []byte{})
	expectRun(t, `out = bytes("abcde")[1:5:2]`, nil, []byte("bd"))
	expectRun(t, `out = bytes("abcde")[1:5:-1]`, nil, []byte(""))
	expectRun(t, `out = bytes("abcde")[4:0:-1]`, nil, []byte("edcb"))
	expectRun(t, `out = bytes("abcde")[0:5:2]`, nil, []byte("ace"))
	expectRun(t, `out = bytes("abcde")[::-1]`, nil, []byte("edcba"))
	expectError(t, `out = bytes("abcde")[::0]`, nil, "step cannot be zero")

	o := core.NewBytesValue([]byte("Hello World!"), false)
	s, _ := o.AsString()
	require.Equal(t, "Hello World!", s)
	require.Equal(t, "bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33])", o.String())

	expectRun(t, fmt.Sprintf(`out = bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33]) == %s`, o.String()), nil, true)

	v := core.NewBytesValue([]byte("hello"), false)
	expectRun(t, fmt.Sprintf(`out = bytes("hello") == %s`, v.String()), nil, true)

	expectRun(t, `out = bytes("abcde").len()`, nil, 5)
	expectRun(t, `out = bytes("abcde").is_empty()`, nil, false)
	expectRun(t, `out = bytes().is_empty()`, nil, true)
	expectRun(t, `out = bytes("abcde").first()`, nil, byte(97))
	expectRun(t, `out = bytes("abcde").last()`, nil, byte(101))

	expectRun(t, `out = bytes("abc").array()`, nil, ARR{97, 98, 99})
	expectRun(t, `out = bytes("abc").record()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, `out = bytes("abc").dict()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, `out = bytes("abc").string()`, nil, "abc")
	expectRun(t, `out = "abc".bytes().array().string()`, nil, "abc")
	expectRun(t, `out = bytes("abc").format()`, nil, "abc")
	expectRun(t, `out = bytes("abc").format("v")`, nil, "bytes([97, 98, 99])")

	expectRun(t, `out = 98 in bytes("abc")`, nil, true)
	expectRun(t, `out = bytes("abc").contains(98)`, nil, true)
	expectRun(t, `out = 255 in bytes("abc")`, nil, false)
	expectRun(t, `out = bytes("abc").contains(255)`, nil, false)
	expectRun(t, `out = bytes("bc") in bytes("abc")`, nil, true)
	expectRun(t, `out = bytes("abc").contains(bytes("bc"))`, nil, true)
	expectRun(t, `out = bytes("bd") in bytes("abc")`, nil, false)
	expectRun(t, `out = bytes("abc").contains(bytes("bd"))`, nil, false)
	expectRun(t, `out = bytes("bd") not in bytes("abc")`, nil, true)
	expectRun(t, `out = bytes("hello").sort()`, nil, []byte("ehllo"))
	expectRun(t, `out = bytes("").dedup()`, nil, []byte(""))
	expectRun(t, `out = bytes("a").dedup()`, nil, []byte("a"))
	expectRun(t, `out = bytes("aabbccd").dedup()`, nil, []byte("abcd"))
	expectRun(t, `out = bytes("abc").dedup()`, nil, []byte("abc"))
	expectRun(t, `out = bytes("abab").dedup()`, nil, []byte("abab"))
	expectRun(t, `out = bytes("hello").sort().dedup()`, nil, []byte("ehlo"))
	expectRun(t, `out = bytes([1, 1, 2, 2, 3]).dedup()`, nil, []byte{1, 2, 3})
	expectRun(t, `out = bytes("").unique()`, nil, []byte(""))
	expectRun(t, `out = bytes("abc").unique()`, nil, []byte("abc"))
	expectRun(t, `out = bytes("hello").unique()`, nil, []byte("helo"))
	expectRun(t, `out = bytes("abab").unique()`, nil, []byte("ab"))
	expectRun(t, `out = bytes([3, 1, 2, 1, 3, 2]).unique()`, nil, []byte{3, 1, 2})
	expectRun(t, `out = bytes("").reverse()`, nil, []byte(""))
	expectRun(t, `out = bytes("hello").reverse()`, nil, []byte("olleh"))
	expectRun(t, `out = bytes([1, 2, 3]).reverse()`, nil, []byte{3, 2, 1})
	expectRun(t, `out = bytes("").chunk(2)`, nil, ARR{})
	expectRun(t, `out = bytes("hello").chunk(2)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, `out = bytes("hello").chunk(2, true)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, `out = bytes("hello").chunk(10)`, nil, ARR{[]byte("hello")})
	expectRun(t, `out = bytes("hello").filter(x => x > 'e')`, nil, []byte("hllo"))
	expectRun(t, `out = bytes("hello").filter((i, x) => i > 2)`, nil, []byte("lo"))
	expectRun(t, `out = bytes("hello").count(x => x > 'e')`, nil, 4)
	expectRun(t, `out = bytes("hello").count((i, x) => i > 2)`, nil, 2)
	expectRun(t, `out = bytes("hello").all(x => x > 'a')`, nil, true)
	expectRun(t, `out = bytes("hello").all(x => x > 'e')`, nil, false)
	expectRun(t, `out = bytes("hello").all((i, x) => i < 5)`, nil, true)
	expectRun(t, `out = bytes("hello").all((i, x) => i < 3)`, nil, false)
	expectRun(t, `out = bytes("hello").any(x => x == 'e')`, nil, true)
	expectRun(t, `out = bytes("hello").any(x => x == 'z')`, nil, false)
	expectRun(t, `out = bytes("hello").any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, `out = bytes("hello").any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, `out = bytes("hello").find(x => x == 'l')`, nil, 2)
	expectRun(t, `out = bytes("hello").find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = bytes("hello").find((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = bytes("hello").find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = bytes("").find(x => true)`, nil, core.Undefined)
	expectError(t, `out = bytes("x").find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `out = bytes("x").find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `out = bytes("x").find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, `out = bytes("hello").min()`, nil, byte('e'))
	expectRun(t, `out = bytes("hello").max()`, nil, byte('o'))
	expectRun(t, `
out = 0
ignored := bytes("abc").for_each(func(b) {
	out += b
	return b < 'b'
})
`, nil, 195)
	expectRun(t, `
items := []
ignored := bytes("ABC").for_each(func(i, b) {
	items = append(items, i, b)
	return true
})
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
	expectRun(t, `
items := []
for i, b in bytes("ABC") {
	items = append(items, i, b)
}
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
}

func TestBytesMutability(t *testing.T) {
	// index assignment
	expectRun(t, `b := bytes("hello"); b[0] = 'H'; out = b`, nil, []byte("Hello"))
	expectRun(t, `b := bytes("hello"); b[-2] = '!'; out = b`, nil, []byte("hel!o"))
	expectRun(t, `b := bytes("abc"); b[0] = 65; out = b`, nil, []byte("Abc"))

	// append
	expectRun(t, `b := bytes("ab"); b2 := append(b, 'c'); out = b2`, nil, []byte("abc"))
	expectRun(t, `b := bytes("ab"); b2 := append(b, 'c', 'd'); out = b2`, nil, []byte("abcd"))
	expectRun(t, `b := bytes("ab"); b2 := append(b, bytes("cd")); out = b2`, nil, []byte("abcd"))
	expectRun(t, `b := bytes("ab"); b2 := append(b, 99); out = b2`, nil, []byte("abc"))
	expectRun(t, `b := bytes("ab"); b2 := append(b, 'c'); out = b`, nil, []byte("ab"))

	// sum / avg / map / reduce
	expectRun(t, `out = bytes("abc").sum()`, nil, 97+98+99)
	expectRun(t, `out = bytes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, `out = bytes().sum()`, nil, core.Undefined)
	expectRun(t, `out = bytes().avg()`, nil, core.Undefined)
	expectRun(t, `out = bytes("abc").map(func(b) { return b + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, `out = bytes("abc").map(func(i, b) { return [i, b] })`, nil,
		ARR{ARR{0, byte('a')}, ARR{1, byte('b')}, ARR{2, byte('c')}})
	expectRun(t, `out = bytes("abc").reduce(0, func(acc, b) { return acc + b })`, nil, 97+98+99)
	expectRun(t, `out = bytes("abc").reduce("", func(acc, i, b) { return acc + i.string() + b.string() })`, nil, "097198299")

	// type names
	expectRun(t, `out = type_name(bytes("abc"))`, nil, "bytes")
	expectRun(t, `out = type_name(immutable(bytes("abc")))`, nil, "immutable-bytes")

	// immutable rejects writes
	expectError(t, `b := immutable(bytes("abc")); b[0] = 'X'`, nil, "not_assignable: type immutable-bytes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, `out = type_name(immutable(bytes("abcd"))[1:3])`, nil, "immutable-bytes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, `out = type_name(immutable(bytes("abcd"))[::-1])`, nil, "bytes")
	// slice of mutable stays mutable
	expectRun(t, `out = type_name(bytes("abcd")[1:3])`, nil, "bytes")

	// copy of immutable yields mutable
	expectRun(t, `b := immutable(bytes("abc")); c := copy(b); c[0] = 'X'; out = c`, nil, []byte("Xbc"))

	// append on immutable returns fresh mutable (does not mutate source)
	expectRun(t, `b := immutable(bytes("ab")); b2 := append(b, 'c'); b2[0] = 'X'; out = b2`, nil, []byte("Xbc"))
	expectRun(t, `b := immutable(bytes("ab")); b2 := append(b, 'c'); out = type_name(b2)`, nil, "bytes")

	// invalid assignment values
	expectError(t, `b := bytes("abc"); b[0] = "xy"`, nil,
		"invalid_index_type: (index assign value) expected byte, got string")
	expectError(t, `b := bytes("abc"); b[0] = 256`, nil,
		"invalid_index_type: (index assign value) expected byte, got int")
	expectError(t, `b := bytes("abc"); b[10] = 'X'`, nil,
		"index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestArrayIterator(t *testing.T) {
	expectRun(t, `
x := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
y := x[2:5]
sum1 := 0
for v in x {
	sum1 += v
}
sum2 := 0
for v in y {
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{55, 12})

	expectRun(t, `
x := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
y := x[2:5]
isum1 := 0
sum1 := 0
for i, v in x {
	isum1 += i
	sum1 += v
}
isum2 := 0
sum2 := 0
for i, v in y {
	isum2 += i
	sum2 += v
}
out = [isum1, sum1, isum2, sum2]
`, nil, ARR{45, 55, 3, 12})
}

func TestStringIterator(t *testing.T) {
	expectRun(t, `
x := "abcdefg"
y := x[2:5]
res1 := ""
for v in x {
	res1 += v
}
res2 := ""
for v in y {
	res2 += v
}
out = [res1, res2]
`, nil, ARR{"abcdefg", "cde"})

	expectRun(t, `
x := "abcdefg"
y := x[2:5]
isum1 := 0
res1 := ""
for i, v in x {
	isum1 += i
	res1 += v
}
isum2 := 0
res2 := ""
for i, v in y {
	isum2 += i
	res2 += v
}
out = [isum1, res1, isum2, res2]
`, nil, ARR{21, "abcdefg", 3, "cde"})
}

func TestBytesIterator(t *testing.T) {
	expectRun(t, `
x := bytes("abcdefg")
y := x[2:5]
res1 := ""
for v in x {
	res1 += v.rune()
}
res2 := ""
for v in y {
	res2 += v.rune()
}
out = [res1, res2]
`, nil, ARR{"abcdefg", "cde"})

	expectRun(t, `
x := bytes("abcdefg")
y := x[2:5]
isum1 := 0
res1 := ""
for i, v in x {
	isum1 += i
	res1 += v.rune()
}
isum2 := 0
res2 := ""
for i, v in y {
	isum2 += i
	res2 += v.rune()
}
out = [isum1, res1, isum2, res2]
`, nil, ARR{21, "abcdefg", 3, "cde"})
}

func TestRecordIterator(t *testing.T) {
	expectRun(t, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
sum2 := 0
for k, v in m {
	sum1 += k[0] - 'a'
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{45, 55})
}

func TestDictIterator(t *testing.T) {
	expectRun(t, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
sum2 := 0
for k, v in m {
	sum1 += k[0] - 'a'
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{45, 55})
}

func TestRange(t *testing.T) {
	expectRun(t, `out = range(97, 103, 1).bytes().string()`, nil, "abcdef")
	expectRun(t, `out = range(103, 97, 1).bytes().string()`, nil, "gfedcb")
	expectRun(t, `out = range(97, 103, 1).string()`, nil, "abcdef")
	expectRun(t, `out = range(103, 97, 1).string()`, nil, "gfedcb")
	expectRun(t, `out = range(1, 3, 1).record()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, `out = range(1, 3, 1).dict()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, `
out = 0
ignored := range(1, 5, 1).for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)
	expectRun(t, `
out = 0
ignored := range(10, 13, 1).for_each(func(i, v) {
	out += i + v
	return true
})
`, nil, 36)

	expectRun(t, `out = range(10, 20, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, `out = range(10, 20, 1).find(v => v == 99)`, nil, core.Undefined)
	expectRun(t, `out = range(10, 20, 1).find((i, v) => i == 3)`, nil, 3)
	expectRun(t, `out = range(20, 10, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, `out = range(0, 0, 1).find(v => true)`, nil, core.Undefined)
	expectError(t, `out = range(0, 5, 1).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, `out = range(0, 5, 1).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, `out = range(0, 5, 1).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, `r := range(0, 10, 1); out = r.len()`, nil, 10)
	expectRun(t, `r := range(0, 10, 2); out = r.len()`, nil, 5)
	expectRun(t, `r := range(0, 10, 3); out = r.len()`, nil, 4)
	expectRun(t, `r := range(0, 10, 4); out = r.len()`, nil, 3)
	expectRun(t, `r := range(0, 10, 5); out = r.len()`, nil, 2)
	expectRun(t, `r := range(0, 10, 6); out = r.len()`, nil, 2)
	expectRun(t, `r := range(0, 10, 7); out = r.len()`, nil, 2)
	expectRun(t, `r := range(0, 10, 8); out = r.len()`, nil, 2)
	expectRun(t, `r := range(0, 10, 9); out = r.len()`, nil, 2)
	expectRun(t, `r := range(0, 10, 10); out = r.len()`, nil, 1)
	expectRun(t, `r := range(0, 10, 11); out = r.len()`, nil, 1)
	expectRun(t, `r := range(0, 10, 100); out = r.len()`, nil, 1)

	expectRun(t, `r := range(0, 100, 1); out = len(r)`, nil, 100)
	expectRun(t, `r := range(0, 100, 2); out = len(r)`, nil, 50)
	expectRun(t, `r := range(0, 100, 3); out = len(r)`, nil, 34)
	expectRun(t, `r := range(0, 100, 5); out = len(r)`, nil, 20)
	expectRun(t, `r := range(0, 100, 10); out = len(r)`, nil, 10)

	expectRun(t, `r := range(0, 100, 1); out = r.len()`, nil, 100)
	expectRun(t, `r := range(0, 100, 2); out = r.len()`, nil, 50)
	expectRun(t, `r := range(0, 100, 3); out = r.len()`, nil, 34)
	expectRun(t, `r := range(0, 100, 5); out = r.len()`, nil, 20)
	expectRun(t, `r := range(0, 100, 10); out = r.len()`, nil, 10)

	expectRun(t, `r := range(100, 0, 1); out = len(r)`, nil, 100)
	expectRun(t, `r := range(100, 0, 2); out = len(r)`, nil, 50)
	expectRun(t, `r := range(100, 0, 3); out = len(r)`, nil, 34)
	expectRun(t, `r := range(100, 0, 5); out = len(r)`, nil, 20)
	expectRun(t, `r := range(100, 0, 10); out = len(r)`, nil, 10)

	expectRun(t, `r := range(100, 0, 1); out = r.len()`, nil, 100)
	expectRun(t, `r := range(100, 0, 2); out = r.len()`, nil, 50)
	expectRun(t, `r := range(100, 0, 3); out = r.len()`, nil, 34)
	expectRun(t, `r := range(100, 0, 5); out = r.len()`, nil, 20)
	expectRun(t, `r := range(100, 0, 10); out = r.len()`, nil, 10)

	expectRun(t, `r := range(0, 5, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4})
	expectRun(t, `r := range(5, 0, 1); out = r.array()`, nil, ARR{5, 4, 3, 2, 1})
	expectRun(t, `r := range(-5, 5, 1); out = r.array()`, nil, ARR{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4})

	expectRun(t, `r := range(0, 10, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	expectRun(t, `r := range(0, 10, 2); out = r.array()`, nil, ARR{0, 2, 4, 6, 8})
	expectRun(t, `r := range(0, 10, 3); out = r.array()`, nil, ARR{0, 3, 6, 9})
	expectRun(t, `r := range(0, 10, 4); out = r.array()`, nil, ARR{0, 4, 8})
	expectRun(t, `r := range(0, 10, 5); out = r.array()`, nil, ARR{0, 5})

	expectRun(t, `r := range(10, 0, 1); out = r.array()`, nil, ARR{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	expectRun(t, `r := range(10, 0, 2); out = r.array()`, nil, ARR{10, 8, 6, 4, 2})
	expectRun(t, `r := range(10, 0, 3); out = r.array()`, nil, ARR{10, 7, 4, 1})
	expectRun(t, `r := range(10, 0, 4); out = r.array()`, nil, ARR{10, 6, 2})
	expectRun(t, `r := range(10, 0, 5); out = r.array()`, nil, ARR{10, 5})

	expectRun(t, `r := range(0, 100, 1); out = r[0]`, nil, 0)
	expectRun(t, `r := range(0, 100, 1); out = r[1]`, nil, 1)
	expectRun(t, `r := range(0, 100, 1); out = r[2]`, nil, 2)
	expectRun(t, `r := range(0, 100, 1); out = r[3]`, nil, 3)
	expectRun(t, `r := range(0, 100, 1); out = r[10]`, nil, 10)

	expectRun(t, `r := range(0, 100, 2); out = r[0]`, nil, 0)
	expectRun(t, `r := range(0, 100, 2); out = r[1]`, nil, 2)
	expectRun(t, `r := range(0, 100, 2); out = r[2]`, nil, 4)
	expectRun(t, `r := range(0, 100, 2); out = r[3]`, nil, 6)
	expectRun(t, `r := range(0, 100, 2); out = r[10]`, nil, 20)

	expectRun(t, `r := range(0, 100, 3); out = r[0]`, nil, 0)
	expectRun(t, `r := range(0, 100, 3); out = r[1]`, nil, 3)
	expectRun(t, `r := range(0, 100, 3); out = r[2]`, nil, 6)
	expectRun(t, `r := range(0, 100, 3); out = r[3]`, nil, 9)
	expectRun(t, `r := range(0, 100, 3); out = r[10]`, nil, 30)
	expectRun(t, `r := range(0, 100, 3); out = r[-1]`, nil, 99)
	expectRun(t, `r := range(10, 0, 2); out = r[-1]`, nil, 2)
	expectError(t, `r := range(0, 100, 3); out = r[-35]`, nil, "index_out_of_bounds")
	expectError(t, `r := range(0, 100, 3); out = r[34]`, nil, "index_out_of_bounds")

	expectRun(t, `r := range(0, 10, 1); out = r.contains(0)`, nil, true)
	expectRun(t, `r := range(0, 10, 1); out = r.contains(5)`, nil, true)
	expectRun(t, `r := range(0, 10, 1); out = r.contains(10)`, nil, false)
	expectRun(t, `r := range(0, 10, 2); out = r.contains(0)`, nil, true)
	expectRun(t, `r := range(0, 10, 2); out = r.contains(1)`, nil, false)
	expectRun(t, `r := range(0, 10, 2); out = r.contains(2)`, nil, true)

	expectRun(t, `r := range(10, 0, 1); out = r.contains(0)`, nil, false)
	expectRun(t, `r := range(10, 0, 1); out = r.contains(5)`, nil, true)
	expectRun(t, `r := range(10, 0, 1); out = r.contains(10)`, nil, true)
	expectRun(t, `r := range(10, 0, 2); out = r.contains(10)`, nil, true)
	expectRun(t, `r := range(10, 0, 2); out = r.contains(9)`, nil, false)
	expectRun(t, `r := range(10, 0, 2); out = r.contains(8)`, nil, true)
	expectRun(t, `out = 11 not in range(0, 10, 1)`, nil, true)

	expectRun(t, `
out = 0
for e in range(1, 10, 1) {
	out += e
}
`, nil, 45)

	expectRun(t, `
out = 0
for i, e in range(1, 10, 1) {
	out += i
}
`, nil, 36)

	expectRun(t, `
out = 0
for e in range(1, 10, 2) {
	out += e
}
`, nil, 25)

	expectRun(t, `
out = 0
for i, e in range(1, 10, 2) {
	out += i
}
`, nil, 10)

	expectRun(t, `
r := range(-10, 10, 1)
a := r.array()
s1 := 0
s2 := 0
for i, e in r {
	s1 += r[i] == e
	s2 += a[i] == e
}
out = [s1, s2]
`, nil, ARR{20, 20})

	expectRun(t, `
r := range(10, -10, 1)
a := r.array()
s1 := 0
s2 := 0
for i, e in r {
	s1 += r[i] == e
	s2 += a[i] == e
}
out = [s1, s2]
`, nil, ARR{20, 20})
}

func TestAssignment(t *testing.T) {
	expectRun(t, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, `a := 1; a = a + 4; out = a`, nil, 5)
	expectRun(t, `a := 1; f1 := func() { a = 2; return a }; out = f1()`, nil, 2)
	expectRun(t, `a := 1; f1 := func() { a := 3; a = 2; return a }; out = f1()`, nil, 2)

	expectRun(t, `a := 1; out = a`, nil, 1)
	expectRun(t, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, `a := 1; func() { a = 2 }(); out = a`, nil, 2)
	expectRun(t, `a := 1; func() { a := 2 }(); out = a`, nil, 1) // "a := 2" defines a new local variable 'a'
	expectRun(t, `a := 1; func() { b := 2; out = b }()`, nil, 2)

	expectRun(t, `
out = func() {
	a := 2
	func() {
		a = 3 // captured from outer scope
	}()
	return a
}()
`, nil, 3)

	expectRun(t, `
func() {
	a := 5
	out = func() {
		a := 4
		return a
	}()
}()`, nil, 4)

	expectError(t, `a := 1; a := 2`, nil, "redeclared")              // redeclared in the same scope
	expectError(t, `func() { a := 1; a := 2 }()`, nil, "redeclared") // redeclared in the same scope

	expectRun(t, `a := 1; a += 2; out = a`, nil, 3)
	expectRun(t, `a := 1; a += 4 - 2;; out = a`, nil, 3)
	expectRun(t, `a := 3; a -= 1;; out = a`, nil, 2)
	expectRun(t, `a := 3; a -= 5 - 4;; out = a`, nil, 2)
	expectRun(t, `a := 2; a *= 4;; out = a`, nil, 8)
	expectRun(t, `a := 2; a *= 1 + 3;; out = a`, nil, 8)
	expectRun(t, `a := 10; a /= 2;; out = a`, nil, 5)
	expectRun(t, `a := 10; a /= 5 - 3;; out = a`, nil, 5)

	// compound assignment operator does not define new variable
	expectError(t, `a += 4`, nil, "unresolved reference")
	expectError(t, `a -= 4`, nil, "unresolved reference")
	expectError(t, `a *= 4`, nil, "unresolved reference")
	expectError(t, `a /= 4`, nil, "unresolved reference")

	expectRun(t, `
f1 := func() {
	f2 := func() {
		a := 1
		a += 2    // it's a statement, not an expression
		return a
	};

	return f2();
};

out = f1();`, nil, 3)

	expectRun(t, `f1 := func() { f2 := func() { a := 1; a += 4 - 2; return a }; return f2(); }; out = f1()`, nil, 3)
	expectRun(t, `f1 := func() { f2 := func() { a := 3; a -= 1; return a }; return f2(); }; out = f1()`, nil, 2)
	expectRun(t, `f1 := func() { f2 := func() { a := 3; a -= 5 - 4; return a }; return f2(); }; out = f1()`, nil, 2)
	expectRun(t, `f1 := func() { f2 := func() { a := 2; a *= 4; return a }; return f2(); }; out = f1()`, nil, 8)
	expectRun(t, `f1 := func() { f2 := func() { a := 2; a *= 1 + 3; return a }; return f2(); }; out = f1()`, nil, 8)
	expectRun(t, `f1 := func() { f2 := func() { a := 10; a /= 2; return a }; return f2(); }; out = f1()`, nil, 5)
	expectRun(t, `f1 := func() { f2 := func() { a := 10; a /= 5 - 3; return a }; return f2(); }; out = f1()`, nil, 5)

	expectRun(t, `a := 1; f1 := func() { f2 := func() { a += 2; return a }; return f2(); }; out = f1()`, nil, 3)

	expectRun(t, `
	f1 := func(a) {
		return func(b) {
			c := a
			c += b * 2
			return c
		}
	}

	out = f1(3)(4)
	`, nil, 11)

	expectRun(t, `
	out = func() {
		a := 1
		func() {
			a = 2
			func() {
				a = 3
				func() {
					a := 4 // declared new
				}()
			}()
		}()
		return a
	}()
	`, nil, 3)

	// write on free variables
	expectRun(t, `
	f1 := func() {
		a := 5

		return func() {
			a += 3
			return a
		}()
	}
	out = f1()
	`, nil, 8)

	expectRun(t, `
    out = func() {
        f1 := func() {
            a := 5
            add1 := func() { a += 1 }
            add2 := func() { a += 2 }
            a += 3
            return func() { a += 4; add1(); add2(); a += 5; return a }
        }
        return f1()
    }()()
    `, nil, 20)

	expectRun(t, `
		it := func(seq, fn) {
			fn(seq[0])
			fn(seq[1])
			fn(seq[2])
		}

		foo := func(a) {
			b := 0
			it([1, 2, 3], func(x) {
				b = x + a
			})
			return b
		}

		out = foo(2)
		`, nil, 5)

	expectRun(t, `
		it := func(seq, fn) {
			fn(seq[0])
			fn(seq[1])
			fn(seq[2])
		}

		foo := func(a) {
			b := 0
			it([1, 2, 3], func(x) {
				b += x + a
			})
			return b
		}

		out = foo(2)
		`, nil, 12)

	expectRun(t, `
out = func() {
	a := 1
	func() {
		a = 2
	}()
	return a
}()
`, nil, 2)

	expectRun(t, `
f := func() {
	a := 1
	return {
		b: func() { a += 3 },
		c: func() { a += 2 },
		d: func() { return a }
	}
}
m := f()
m.b()
m.c()
out = m.d()
`, nil, 6)

	expectRun(t, `
each := func(s, x) { for i:=0; i<len(s); i++ { x(s[i]) } }

out = func() {
	a := 100
	each([1, 2, 3], func(x) {
		a += x
	})
	a += 10
	return func(b) {
		return a + b
	}
}()(20)
`, nil, 136)

	// assigning different type value
	expectRun(t, `a := 1; a = "foo"; out = a`, nil, "foo")              // global
	expectRun(t, `func() { a := 1; a = "foo"; out = a }()`, nil, "foo") // local

	expectRun(t, `
out = func() {
	a := 5
	return func() {
		a = "foo"
		return a
	}()
}()`, nil, "foo") // free

	// variables declared in if/for blocks
	expectRun(t, `for a:=0; a<5; a++ {}; a := "foo"; out = a`, nil, "foo")
	expectRun(t, `func() { for a:=0; a<5; a++ {}; a := "foo"; out = a }()`, nil, "foo")

	// selectors
	expectRun(t, `a:=[1,2,3]; a[1] = 5; out = a[1]`, nil, 5)
	expectRun(t, `a:=[1,2,3]; a[1] += 5; out = a[1]`, nil, 7)
	expectRun(t, `a:={b:1,c:2}; a.b = 5; out = a.b`, nil, 5)
	expectRun(t, `a:={b:1,c:2}; a.b += 5; out = a.b`, nil, 6)
	expectRun(t, `a:={b:1,c:2}; a.b += a.c; out = a.b`, nil, 3)
	expectRun(t, `a:={b:1,c:2}; a.b += a.c; out = a.c`, nil, 2)

	expectRun(t, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.c.f[1] += 2
out = a["c"]["f"][1]
`, nil, 10)

	expectRun(t, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.c.h = "bar"
out = a.c.h
`, nil, "bar")

	expectError(t, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.x.e = "bar"`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
}

func TestBitwise(t *testing.T) {
	expectRun(t, `out = 1 & 1`, nil, 1)
	expectRun(t, `out = 1 & 0`, nil, 0)
	expectRun(t, `out = 0 & 1`, nil, 0)
	expectRun(t, `out = 0 & 0`, nil, 0)
	expectRun(t, `out = 1 | 1`, nil, 1)
	expectRun(t, `out = 1 | 0`, nil, 1)
	expectRun(t, `out = 0 | 1`, nil, 1)
	expectRun(t, `out = 0 | 0`, nil, 0)
	expectRun(t, `out = 1 ^ 1`, nil, 0)
	expectRun(t, `out = 1 ^ 0`, nil, 1)
	expectRun(t, `out = 0 ^ 1`, nil, 1)
	expectRun(t, `out = 0 ^ 0`, nil, 0)
	expectRun(t, `out = 1 &^ 1`, nil, 0)
	expectRun(t, `out = 1 &^ 0`, nil, 1)
	expectRun(t, `out = 0 &^ 1`, nil, 0)
	expectRun(t, `out = 0 &^ 0`, nil, 0)
	expectRun(t, `out = 1 << 2`, nil, 4)
	expectRun(t, `out = 16 >> 2`, nil, 4)

	expectRun(t, `out = 1; out &= 1`, nil, 1)
	expectRun(t, `out = 1; out |= 0`, nil, 1)
	expectRun(t, `out = 1; out ^= 0`, nil, 1)
	expectRun(t, `out = 1; out &^= 0`, nil, 1)
	expectRun(t, `out = 1; out <<= 2`, nil, 4)
	expectRun(t, `out = 16; out >>= 2`, nil, 4)

	expectRun(t, `out = ^0`, nil, ^0)
	expectRun(t, `out = ^1`, nil, ^1)
	expectRun(t, `out = ^55`, nil, ^55)
	expectRun(t, `out = ^-55`, nil, ^-55)
}

func TestFormatting(t *testing.T) {
	// f-string shapes (docs/f-strings.md)
	expectRun(t, `x = 1; y = 2; z = "hello"; out = f"{z}, {x}, {y}"`, nil, "hello, 1, 2")
	expectRun(t, `name = "world"; n = 42; out = f"hello, {name}! n={n:5d}"`, nil, "hello, world! n=   42")
	expectRun(t, `out = f""`, nil, "")
	expectRun(t, `out = f"hello"`, nil, "hello")
	expectRun(t, `x = 10; out = f"{x}"`, nil, "10")
	expectRun(t, `x = 10; out = f"prefix {x}"`, nil, "prefix 10")
	expectRun(t, `x = 10; out = f"{x} suffix"`, nil, "10 suffix")
	expectRun(t, `x = 10; y = 20; out = f"{x}{y}"`, nil, "1020")
	expectRun(t, `x = 1; y = 2; z = 3; out = f"a={x} b={y} c={z}"`, nil, "a=1 b=2 c=3")
	expectRun(t, `a = 1; b = 2; c = 3; out = f"<{a}{b}>{c}"`, nil, "<12>3")

	// escapes inside f-string body (docs/f-strings.md)
	expectRun(t, `p = "/tmp"; out = f"path = \"{p}\""`, nil, `path = "/tmp"`)
	expectRun(t, `out = f"set = {{1, 2, 3}}"`, nil, "set = {1, 2, 3}")
	expectRun(t, `x = 1; out = f"newline -> {x}\n"`, nil, "newline -> 1\n")

	// format specs in f-strings (docs/f-strings.md)
	expectRun(t, `pi = 3.14159; out = f"{pi:.2f}"`, nil, "3.14")
	expectRun(t, `n = 42; out = f"{n:05d}"`, nil, "00042")
	expectRun(t, `x = -42; out = f"{x:05d}"`, nil, "-0042")
	expectRun(t, `n = 1234; out = f"{n:>10,}"`, nil, "     1,234")
	expectRun(t, `x = 255; out = f"{x:06x}"`, nil, "0x00ff")
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#date}"`, nil, "2020-06-20")

	// expressions inside `{...}` (docs/f-strings.md)
	expectRun(t, `x = 1; y = 2; out = f"{x + y}"`, nil, "3")
	expectRun(t, `users = [{name: "alice"}, {name: "bob"}]; i = 1; out = f"{users[i].name}"`, nil, "bob")
	expectRun(t, `out = f"{ dict({a: 1}).values() :v}"`, nil, "[1]")
	expectRun(t, `out = f"{ {a: 1} }"`, nil, `{"a": 1}`)
	expectRun(t, `out = f"{ {a: 1} :v}"`, nil, `{"a": 1}`)
	expectRun(t, `out = f"{[1,2,3]:v}"`, nil, "[1, 2, 3]")
	expectRun(t, `out = f"{[1,2,3]}"`, nil, "[1, 2, 3]")

	// Format Mini-Language: time #-tail templates (docs/format-mini-language.md)
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#%Y-%m-%d %H:%M:%S}"`, nil, "2020-06-20 01:02:03")
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#%Y-%j}"`, nil, "2020-172")
	expectRun(t, `t = time("2020-06-20 13:02:03 +0200"); out = f"{t:#%I:%M %p}"`, nil, "01:02 PM")

	// int / byte verbs
	expectRun(t, `out = (255).format("x")`, nil, "0xff")
	expectRun(t, `out = (255).format("X")`, nil, "0xFF")
	expectRun(t, `out = (42).format("b")`, nil, "0b101010")
	expectRun(t, `out = (42).format("o")`, nil, "0o52")
	expectRun(t, `out = (65).format("c")`, nil, "A")
	expectRun(t, `out = (42).format("d")`, nil, "42")

	// float verbs
	expectRun(t, `out = (1.5).format("e")`, nil, "1.500000e+00")
	expectRun(t, `out = (0.5).format("%")`, nil, "50.000000%")
	expectRun(t, `out = (1.234d).format("s")`, nil, "1.234")

	// bool verbs
	expectRun(t, `out = true.format("t")`, nil, "true")
	expectRun(t, `out = true.format("T")`, nil, "bool")
	expectRun(t, `out = true.format("d")`, nil, "1")
	expectRun(t, `out = false.format("d")`, nil, "0")

	// universal T verb prints the type name
	expectRun(t, `out = (42).format("T")`, nil, "int")
	expectRun(t, `out = (1.5).format("T")`, nil, "float")
	expectRun(t, `out = "abc".format("T")`, nil, "string")
	expectRun(t, `out = 'A'.format("T")`, nil, "rune")

	// rune verbs
	expectRun(t, `out = 'A'.format("d")`, nil, "65")
	expectRun(t, `out = 'A'.format("U")`, nil, "U+0041")
	expectRun(t, `out = 'A'.format("q")`, nil, "'A'")

	// string verbs
	expectRun(t, `out = "abc".format("v")`, nil, `"abc"`)
	expectRun(t, `out = "hello".format("q")`, nil, `"hello"`)
	expectRun(t, `out = "hello".format("b")`, nil, "aGVsbG8=")
	expectRun(t, `out = "hello".format("B")`, nil, "aGVsbG8")
	expectRun(t, `out = "hi".format("x")`, nil, "6869")
	expectRun(t, `out = "a b/c".format("u")`, nil, "a%20b%2Fc")

	// time verbs / aliases
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#date")`, nil, "2020-06-20")
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#time")`, nil, "01:02:03")
	expectRun(t, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#unix")`, nil, "1592607723")

	// container Kavun-source form via 'v' (docs/format-mini-language.md default-vs-v table)
	expectRun(t, `out = [1, 2, 3].format("v")`, nil, "[1, 2, 3]")

	// --- Edge cases: expressions with conflicting symbols (`:`, `{`, `}`, `?`) with and without fspec ---

	// Slicing uses `:` inside `[]`
	expectRun(t, `a = [1,2,3,4,5]; out = f"{a[1:3]}"`, nil, "[2, 3]")
	expectRun(t, `a = [1,2,3,4,5]; out = f"{a[1:3]:v}"`, nil, "[2, 3]")
	expectRun(t, `a = [1,2,3,4,5]; out = f"{a[::-1]:v}"`, nil, "[5, 4, 3, 2, 1]")
	expectRun(t, `s = "hello"; out = f"{s[1:4]}"`, nil, "ell")
	expectRun(t, `s = "hello"; out = f"{s[1:4]:>6}"`, nil, "   ell")

	// Record literal `{...}` (with internal `:`) directly in expression
	expectRun(t, `out = f"{ {a: 1} }"`, nil, `{"a": 1}`)
	expectRun(t, `out = f"{ {a: 1} :v}"`, nil, `{"a": 1}`)
	expectRun(t, `out = f"{ {a: 1}.a }"`, nil, "1")
	expectRun(t, `out = f"{ {a: 1}.a :>3}"`, nil, "  1")
	expectRun(t, `out = f"{ {a: {b: 1}}.a.b }"`, nil, "1")
	expectRun(t, `out = f"{ {a: {b: 1}}.a.b :05d}"`, nil, "00001")

	// Dict literal expression
	expectRun(t, `out = f"{ dict({a: 1}) :v}"`, nil, `dict({"a": 1})`)
	expectRun(t, `out = f"{ dict({a: 1}).values() }"`, nil, "[1]")

	// Ternary (uses `?` and `:`) — without spec, with spec, nested, chained
	expectRun(t, `cond = true; out = f"{cond ? \"yes\" : \"no\"}"`, nil, "yes")
	expectRun(t, `cond = false; out = f"{cond ? \"yes\" : \"no\"}"`, nil, "no")
	expectRun(t, `cond = true; out = f"{cond ? \"yes\" : \"no\":>5}"`, nil, "  yes")
	expectRun(t, `cond = true; out = f"{cond ? 42 : 7 :>5d}"`, nil, "   42")
	expectRun(t, `cond = false; out = f"{cond ? 42 : 7 :>5d}"`, nil, "    7")
	expectRun(t, `cond = true; out = f"{(cond ? 1 : 2) + 10}"`, nil, "11")
	expectRun(t, `cond = false; out = f"{(cond ? 1 : 2) + 10:>5}"`, nil, "   12")
	expectRun(t, `a = true; b = false; out = f"{a ? (b ? 1 : 2) : 3}"`, nil, "2")
	expectRun(t, `a = true; b = false; out = f"{a ? (b ? 1 : 2) : 3:>5d}"`, nil, "    2")
	expectRun(t, `a = false; b = true; out = f"{a ? 1 : b ? 2 : 3 :>5d}"`, nil, "    2")

	// Strings inside expressions containing `{`, `}`, `:`
	expectRun(t, `s = "{not}"; out = f"prefix {s} suffix"`, nil, "prefix {not} suffix")
	expectRun(t, `s = "a:b"; out = f"{s}"`, nil, "a:b")
	expectRun(t, `s = "a:b"; out = f"{s:>10}"`, nil, "       a:b")
	expectRun(t, `out = f"{\"hi\"}"`, nil, "hi")
	expectRun(t, `out = f"{\"hi\":>5}"`, nil, "   hi")
	expectRun(t, `out = f"{\"a:b\"}"`, nil, "a:b")
	expectRun(t, `out = f"{\"a:b\":>5}"`, nil, "  a:b")

	// Rune literals containing `:`, `{`, `}`
	expectRun(t, `out = f"{':'}"`, nil, ":")
	expectRun(t, `out = f"{'{'}"`, nil, "{")
	expectRun(t, `out = f"{'}'}"`, nil, "}")
	expectRun(t, `out = f"{':':>3}"`, nil, "  :")

	// Multiple interpolations mixing fspec and non-fspec
	expectRun(t, `a = 1; b = 2; out = f"{a} {b:03d} {a + b:>4d}"`, nil, "1 002    3")

	// Function call with embedded string-literal args
	expectRun(t, `out = f"{int(\"42\") + 1}"`, nil, "43")
	expectRun(t, `out = f"{int(\"42\") + 1:>5d}"`, nil, "   43")

	// Literal `{{`/`}}` adjacent to interpolations
	expectRun(t, `x = 5; out = f"{{{x}}}"`, nil, "{5}")
	expectRun(t, `x = 5; out = f"{{{x:03d}}}"`, nil, "{005}")

	// --- Real-world usage patterns ---

	// Log-style messages
	expectRun(t, `id = 42; name = "alice"; out = f"user {name} (id={id}) logged in"`, nil, "user alice (id=42) logged in")
	expectRun(t, `path = "/etc/foo"; err = "permission denied"; out = f"failed to open {path}: {err}"`, nil, "failed to open /etc/foo: permission denied")

	// Tabular alignment
	expectRun(t, `name = "alice"; age = 30; email = "a@x"; out = f"{name:<10} {age:>3} {email}"`, nil, "alice       30 a@x")
	expectRun(t, `out = f"{\"name\":<10}{\"age\":>5}"`, nil, "name        age")
	expectRun(t, `out = f"{\"title\":-^15}"`, nil, "-----title-----")

	// Currency / thousands grouping
	expectRun(t, `amount = 1234567.89; out = f"${amount:,.2f}"`, nil, "$1,234,567.89")
	expectRun(t, `n = 1000000; out = f"{n:,}"`, nil, "1,000,000")
	expectRun(t, `n = 1234567; out = f"{n:_}"`, nil, "1_234_567")

	// Percentage
	expectRun(t, `r = 0.875; out = f"{r:.1%}"`, nil, "87.5%")
	expectRun(t, `r = 0.5; out = f"{r:6.2%}"`, nil, "50.00%")

	// Sign control
	expectRun(t, `x = 42; out = f"{x:+d}"`, nil, "+42")
	expectRun(t, `x = -42; out = f"{x:+d}"`, nil, "-42")
	expectRun(t, `x = 42; out = f"{x: d}"`, nil, " 42")

	// Hex dump style
	expectRun(t, `addr = 255; out = f"{addr:08x}"`, nil, "0x0000ff")
	expectRun(t, `b = 0xab; out = f"{b:02X}"`, nil, "0xAB")

	// Padding identifiers / progress
	expectRun(t, `n = 7; out = f"ID-{n:06d}"`, nil, "ID-000007")
	expectRun(t, `i = 3; total = 100; out = f"[{i:>3}/{total}] processing..."`, nil, "[  3/100] processing...")

	// Building paths and URLs
	expectRun(t, `dir = "/tmp"; name = "foo"; ext = "txt"; out = f"{dir}/{name}.{ext}"`, nil, "/tmp/foo.txt")
	expectRun(t, `host = "example.com"; port = 8080; path = "/api"; out = f"http://{host}:{port}{path}"`, nil, "http://example.com:8080/api")

	// Floating-point precision
	expectRun(t, `pi = 3.14159265358979; out = f"pi = {pi:.4f}"`, nil, "pi = 3.1416")
	expectRun(t, `x = 1234567.89; out = f"{x:.3e}"`, nil, "1.235e+06")
	expectRun(t, `x = 0.00012345; out = f"{x:.2g}"`, nil, "0.00012")

	// Date/time formatting (real-world templates)
	expectRun(t, `ts = time("2026-05-05 18:42:07 +0200"); out = f"[{ts:#%Y-%m-%d %H:%M:%S}] log message"`, nil, "[2026-05-05 18:42:07] log message")
	expectRun(t, `ts = time("2026-05-05 18:42:07 +0200"); out = f"{ts:#%a, %d %b %Y}"`, nil, "Tue, 05 May 2026")
	expectRun(t, `ts = time("2026-05-05 09:42:00 +0200"); out = f"{ts:#%I:%M %p}"`, nil, "09:42 AM")

	// Multi-line via \n inside f-string body
	expectRun(t, `name = "bob"; n = 3; out = f"name: {name}\ncount: {n}"`, nil, "name: bob\ncount: 3")

	// Booleans / mixed types
	expectRun(t, `ok = true; n = 0; out = f"ok={ok} n={n}"`, nil, "ok=true n=0")

	// Method chain (simple)
	expectRun(t, `name = "ALICE"; out = f"hello, {name.lower()}"`, nil, "hello, alice")
	expectRun(t, `s = "  hello  "; out = f"[{s.trim()}]"`, nil, "[hello]")

	// len / common builtins
	expectRun(t, `xs = [1,2,3,4,5]; out = f"got {len(xs)} items"`, nil, "got 5 items")

	// Array rendering inside a sentence
	expectRun(t, `xs = [1, 2, 3]; out = f"items: {xs}"`, nil, "items: [1, 2, 3]")
	expectRun(t, `xs = [1, 2, 3]; out = f"items: {xs:v}"`, nil, "items: [1, 2, 3]")

	// Negative-zero suppression with `~`
	expectRun(t, `x = -0.0001; out = f"{x:.2f}"`, nil, "-0.00")
	expectRun(t, `x = -0.0001; out = f"{x:.2~f}"`, nil, "0.00")

	// Centered text with default fill
	expectRun(t, `s = "ok"; out = f"|{s:^6}|"`, nil, "|  ok  |")

	// Concatenation of multiple f-strings
	expectRun(t, `a = 1; b = 2; out = f"a={a}" + " " + f"b={b}"`, nil, "a=1 b=2")

	// --- Dynamic format specs (Python-style nested `{...}` inside the spec) ---

	// width / precision from variables
	expectRun(t, `v = 3.14159; w = 10; p = 3; out = f"[{v:{w}.{p}f}]"`, nil, "[     3.142]")
	expectRun(t, `v = 3.14159; w = 10; p = 3; out = f"[{v:>{w}.{p}f}]"`, nil, "[     3.142]")

	// fill, align, width all dynamic
	expectRun(t, `n = 42; w = 10; fill = "*"; align = ">"; out = f"[{n:{fill}{align}{w}}]"`, nil, "[********42]")

	// arithmetic in nested spec expression
	expectRun(t, `n = 1; w = 3; out = f"[{n:{w*2}d}]"`, nil, "[     1]")

	// zero-pad via "0" + width
	expectRun(t, `n = 7; w = 4; out = f"[{n:0{w}d}]"`, nil, "[0007]")

	// runtime spec built from a single variable holding the entire spec text
	expectRun(t, `n = 42; spec = "05d"; out = f"[{n:{spec}}]"`, nil, "[00042]")

	// dynamic spec mixed with static specs in the same f-string
	expectRun(t, `x = 1; y = 2; w = 4; out = f"a={x:03d} b={y:{w}d}"`, nil, "a=001 b=   2")

	// dynamic spec where the inner expression returns the empty string -> default formatting
	expectRun(t, `n = 7; s = ""; out = f"[{n:{s}}]"`, nil, "[7]")

	// dynamic-spec fast path is consistent across iterations (cache hit semantics)
	expectRun(t, `w = 5; out = ""; for i in [1, 2, 3] { out += f"[{i:{w}d}]" }`, nil, "[    1][    2][    3]")

	// runtime error when the dynamic spec resolves to invalid fspec text
	expectError(t, `bad = "zzz"; out = f"{1:{bad}}"`, nil, `f-string format spec "zzz"`)
}

func TestFStringDynamicSpecParseErrors(t *testing.T) {
	// Parse-time errors are reported by the parser itself (not by expectError, which uses require.NoError on parse).
	parseErr := func(input, want string) {
		t.Helper()
		fs := parser.NewFileSet()
		f := fs.AddFile("test", -1, len(input))
		p := parser.NewParser(f, []byte(input), nil)
		_, err := p.ParseFile()
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), want), "expected error to contain %q, got: %s", want, err.Error())
	}

	// nested `{` inside a dynamic-spec placeholder is forbidden (only one level of nesting)
	parseErr(`x = f"{1:{{w}}}"`, "fspec")

	// empty placeholder inside a format spec
	parseErr(`x = f"{1:{}}"`, "empty expression in format spec")

	// missing closing `}` inside a format spec
	parseErr(`x = f"{1:{w}"`, "missing")

	// invalid expression inside a dynamic spec
	parseErr(`x = f"{1:{1+}}"`, "f-string")
}

func TestBuiltinFunctionLen(t *testing.T) {
	expectRun(t, `out = len("")`, nil, 0)
	expectRun(t, `out = len("four")`, nil, 4)
	expectRun(t, `out = len("hello world")`, nil, 11)
	expectRun(t, `out = len([])`, nil, 0)
	expectRun(t, `out = len([1, 2, 3])`, nil, 3)
	expectRun(t, `out = len({})`, nil, 0)
	expectRun(t, `out = len({a:1, b:2})`, nil, 2)
	expectRun(t, `out = len(immutable([]))`, nil, 0)
	expectRun(t, `out = len(immutable([1, 2, 3]))`, nil, 3)
	expectRun(t, `out = len(immutable({}))`, nil, 0)
	expectRun(t, `out = len(immutable({a:1, b:2}))`, nil, 2)
	expectRun(t, `out = len(undefined)`, nil, 0)
	expectRun(t, `out = len(0)`, nil, 1)
	expectRun(t, `out = len(1)`, nil, 1)
	expectError(t, `len("one", "two")`, nil, "wrong_num_arguments")

	// builtins can be reassigned at the top level (smart assignment mode)
	expectRun(t, `len = 10; out = len`, nil, 10)
	expectRun(t, `len := 10; out = len`, nil, 10)
	expectRun(t, `len = func(x) { return 42 }; out = len("hi")`, nil, 42)

	// builtins can be shadowed in function-local scopes; outer scope still sees builtin
	expectRun(t, `f := func() { len := 10; return len }; out = f()`, nil, 10)
	expectRun(t, `f := func() { len := 10; return len }; out = f() + len("hi")`, nil, 12)

	// shadowing in an if-block: outer reference still resolves to builtin
	expectRun(t, `out = 0; if true { len := 10; out = len }`, nil, 10)
	expectRun(t, `if true { len := 10 }; out = len("hi")`, nil, 2)

	// reassignment changes resolution from this point onward; earlier
	// references compiled to OpGetBuiltin keep the builtin semantics
	expectRun(t, `a := len("ab"); len = 99; b := len; out = a + b`, nil, 101)

	// compound assignment to a builtin remains disallowed (no storage)
	expectError(t, `len += 1`, nil, "cannot assign to builtin 'len'")
	expectError(t, `len -= 1`, nil, "cannot assign to builtin 'len'")
}

func TestBuiltinFunctionCopy(t *testing.T) {
	expectRun(t, `out = copy(1)`, nil, 1)
	expectError(t, `copy(1, 2)`, nil, "wrong_num_arguments")
}

func TestBuiltinFunctionAppend(t *testing.T) {
	expectRun(t, `out = append([1, 2, 3], 4)`, nil, ARR{1, 2, 3, 4})
	expectRun(t, `out = append([1, 2, 3], 4, 5, 6)`, nil, ARR{1, 2, 3, 4, 5, 6})
	expectRun(t, `out = append([1, 2, 3], "foo", false)`, nil, ARR{1, 2, 3, "foo", false})
}

func TestBuiltinFunctionInt(t *testing.T) {
	expectRun(t, `out = int(1)`, nil, 1)
	expectRun(t, `out = int(1.8)`, nil, 1)
	expectRun(t, `out = int("-522")`, nil, -522)
	expectRun(t, `out = int(true)`, nil, 1)
	expectRun(t, `out = int(false)`, nil, 0)
	expectRun(t, `out = int('8')`, nil, 56)
	expectRun(t, `out = int([1])`, nil, core.Undefined)
	expectRun(t, `out = int({a: 1})`, nil, core.Undefined)
	expectRun(t, `out = int(time(1))`, nil, 1)
	expectRun(t, `out = int(undefined)`, nil, core.Undefined)
	expectRun(t, `out = int("-522", 1)`, nil, -522)
	expectRun(t, `out = int(undefined, 1)`, nil, 1)
	expectRun(t, `out = int(undefined, 1.8)`, nil, 1.8)
	expectRun(t, `out = int(undefined, string(1))`, nil, "1")
	expectRun(t, `out = int(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionString(t *testing.T) {
	expectRun(t, `out = string(1)`, nil, "1")
	expectRun(t, `out = string(1.8)`, nil, "1.8")
	expectRun(t, `out = string("-522")`, nil, "-522")
	expectRun(t, `out = string(true)`, nil, "true")
	expectRun(t, `out = string(false)`, nil, "false")
	expectRun(t, `out = string('8')`, nil, "8")
	expectRun(t, `out = string([100, 101, 102])`, nil, "def")
	expectRun(t, `out = string({b: "foo"})`, nil, `{"b": "foo"}`)
	expectRun(t, `out = string(undefined)`, nil, core.Undefined) // not "undefined"
	expectRun(t, `out = string(1, "-522")`, nil, "1")
	expectRun(t, `out = string(undefined, "-522")`, nil, "-522") // not "undefined"
}

func TestBuiltinFunctionFloat(t *testing.T) {
	expectRun(t, `out = float(1)`, nil, 1.0)
	expectRun(t, `out = float(1.8)`, nil, 1.8)
	expectRun(t, `out = float("-52.2")`, nil, -52.2)
	expectRun(t, `out = float(true)`, nil, core.Undefined)
	expectRun(t, `out = float(false)`, nil, core.Undefined)
	expectRun(t, `out = float('8')`, nil, core.Undefined)
	expectRun(t, `out = float([1,8.1,true,3])`, nil, core.Undefined)
	expectRun(t, `out = float({a: 1, b: "foo"})`, nil, core.Undefined)
	expectRun(t, `out = float(undefined)`, nil, core.Undefined)
	expectRun(t, `out = float("-52.2", 1.8)`, nil, -52.2)
	expectRun(t, `out = float(undefined, 1)`, nil, 1)
	expectRun(t, `out = float(undefined, 1.8)`, nil, 1.8)
	expectRun(t, `out = float(undefined, "-52.2")`, nil, "-52.2")
	expectRun(t, `out = float(undefined, rune(56))`, nil, '8')
	expectRun(t, `out = float(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionRune(t *testing.T) {
	expectRun(t, `out = rune(56)`, nil, '8')
	expectRun(t, `out = rune(1.8)`, nil, core.Undefined)
	expectRun(t, `out = rune("-52.2")`, nil, core.Undefined)
	expectRun(t, `out = rune(true)`, nil, core.Undefined)
	expectRun(t, `out = rune(false)`, nil, core.Undefined)
	expectRun(t, `out = rune('8')`, nil, '8')
	expectRun(t, `out = rune([1,8.1,true,3])`, nil, core.Undefined)
	expectRun(t, `out = rune({a: 1, b: "foo"})`, nil, core.Undefined)
	expectRun(t, `out = rune(undefined)`, nil, core.Undefined)
	expectRun(t, `out = rune(56, 'a')`, nil, '8')
	expectRun(t, `out = rune(undefined, '8')`, nil, '8')
	expectRun(t, `out = rune(undefined, 56)`, nil, 56)
	expectRun(t, `out = rune(undefined, "-52.2")`, nil, "-52.2")
	expectRun(t, `out = rune(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionBool(t *testing.T) {
	expectRun(t, `out = bool(1)`, nil, true)          // non-zero integer: true
	expectRun(t, `out = bool(0)`, nil, false)         // zero: true
	expectRun(t, `out = bool(1.8)`, nil, true)        // all floats (except for NaN): true
	expectRun(t, `out = bool(0.0)`, nil, true)        // all floats (except for NaN): true
	expectRun(t, `out = bool("false")`, nil, false)   // parsed boolean string: false
	expectRun(t, `out = bool("true")`, nil, true)     // parsed boolean string: true
	expectRun(t, `out = bool("")`, nil, false)        // empty string: false
	expectRun(t, `out = bool(true)`, nil, true)       // true: true
	expectRun(t, `out = bool(false)`, nil, false)     // false: false
	expectRun(t, `out = bool('8')`, nil, true)        // non-zero chars: true
	expectRun(t, `out = bool(rune(0))`, nil, false)   // zero rune: false
	expectRun(t, `out = bool([1])`, nil, true)        // non-empty arrays: true
	expectRun(t, `out = bool([])`, nil, false)        // empty array: false
	expectRun(t, `out = bool({a: 1})`, nil, true)     // non-empty maps: true
	expectRun(t, `out = bool({})`, nil, false)        // empty maps: false
	expectRun(t, `out = bool(undefined)`, nil, false) // undefined: false
}

func TestBuiltinFunctionBytes(t *testing.T) {
	expectRun(t, `out = bytes(1)`, nil, []byte{0})
	expectRun(t, `out = bytes(1.8)`, nil, core.Undefined)
	expectRun(t, `out = bytes("-522")`, nil, []byte{'-', '5', '2', '2'})
	expectRun(t, `out = bytes(true)`, nil, core.Undefined)
	expectRun(t, `out = bytes(false)`, nil, core.Undefined)
	expectRun(t, `out = bytes('8')`, nil, core.Undefined)
	expectRun(t, `out = bytes([1])`, nil, []byte{1})
	expectRun(t, `out = bytes({a: 1})`, nil, core.Undefined)
	expectRun(t, `out = bytes(undefined)`, nil, core.Undefined)
	expectRun(t, `out = bytes("-522", ['8'])`, nil, []byte{'-', '5', '2', '2'})
	expectRun(t, `out = bytes(undefined, "-522")`, nil, "-522")
	expectRun(t, `out = bytes(undefined, 1)`, nil, 1)
	expectRun(t, `out = bytes(undefined, 1.8)`, nil, 1.8)
	expectRun(t, `out = bytes(undefined, int("-522"))`, nil, -522)
	expectRun(t, `out = bytes(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionIs(t *testing.T) {
	expectRun(t, `out = is_error(error(1))`, nil, true)
	expectRun(t, `out = is_error(1)`, nil, false)

	expectRun(t, `out = is_undefined(undefined)`, nil, true)
	expectRun(t, `out = is_undefined(error(1))`, nil, false)

	// is_function
	expectRun(t, `out = is_function(1)`, nil, false)
	expectRun(t, `out = is_function(func() {})`, nil, true)
	expectRun(t, `out = is_function(func(x) { return x })`, nil, true)
	expectRun(t, `out = is_function(len)`, nil, true)                                               // builtin function
	expectRun(t, `a := func(x) { return func() { return x } }; out = is_function(a)`, nil, true)    // function
	expectRun(t, `a := func(x) { return func() { return x } }; out = is_function(a(5))`, nil, true) // closure

	expectRun(t, `out = is_function(x)`,
		Opts().Symbol("x", kavun.MustValueOf([]string{"foo", "bar"})).Skip2ndPass(),
		false) // user object

	// is_callable
	expectRun(t, `out = is_callable(1)`, nil, false)
	expectRun(t, `out = is_callable(func() {})`, nil, true)
	expectRun(t, `out = is_callable(func(x) { return x })`, nil, true)
	expectRun(t, `out = is_callable(len)`, nil, true)                                               // builtin function
	expectRun(t, `a := func(x) { return func() { return x } }; out = is_callable(a)`, nil, true)    // function
	expectRun(t, `a := func(x) { return func() { return x } }; out = is_callable(a(5))`, nil, true) // closure

	expectRun(t, `out = is_callable(x)`,
		Opts().Symbol("x", kavun.MustValueOf([]string{"foo", "bar"})).Skip2ndPass(), false) // user object
}

func TestBuiltinFunctionTypeName(t *testing.T) {
	expectRun(t, `out = type_name(1)`, nil, "int")
	expectRun(t, `out = type_name(1.1)`, nil, "float")
	expectRun(t, `out = type_name("a")`, nil, "string")
	expectRun(t, `out = type_name([1,2,3])`, nil, "array")
	expectRun(t, `out = type_name({k:1})`, nil, "record")
	expectRun(t, `out = type_name('a')`, nil, "rune")
	expectRun(t, `out = type_name(true)`, nil, "bool")
	expectRun(t, `out = type_name(false)`, nil, "bool")
	expectRun(t, `out = type_name(bytes( 1))`, nil, "bytes")
	expectRun(t, `out = type_name(undefined)`, nil, "undefined")
	expectRun(t, `out = type_name(error("err"))`, nil, "error")
	expectRun(t, `out = type_name(func() {})`, nil, "<compiled-function/0>")
	expectRun(t, `a := func(x) { return func() { return x } }; out = type_name(a(5))`, nil, "<compiled-function/0>") // closure
}

func TestBuiltinFunctionFormat(t *testing.T) {
	// --- argument validation ---
	expectError(t, `format()`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 0")
	expectError(t, `format("x")`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 1")
	expectError(t, `format("x", [], [])`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 3")
	expectError(t, `format(1, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got int")
	expectError(t, `format(1.0, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got float")
	expectError(t, `format(undefined, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got undefined")
	expectError(t, `format("x", 1)`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got int")
	expectError(t, `format("x", "y")`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got string")
	expectError(t, `format("x", undefined)`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got undefined")

	// --- pure literal templates (no placeholders) accept any args container ---
	expectRun(t, `out = format("", [])`, nil, "")
	expectRun(t, `out = format("", {})`, nil, "")
	expectRun(t, `out = format("hello", [])`, nil, "hello")
	expectRun(t, `out = format("hello", {})`, nil, "hello")

	// --- {{ and }} brace escapes ---
	expectRun(t, `out = format("a {{ b }} c", [])`, nil, "a { b } c")
	expectRun(t, `out = format("{{}}", [])`, nil, "{}")
	expectRun(t, `out = format("set = {{ {x} }}", {x: 1})`, nil, "set = { 1 }")

	// --- examples from docs/format-function.md ---
	expectRun(t, `out = format("hello {x} from {y}!", {x: "kavun", y: "Kherson"})`, nil, "hello kavun from Kherson!")
	expectRun(t, `out = format("hello {0} from {1}!", ["kavun", "Kherson"])`, nil, "hello kavun from Kherson!")
	expectRun(t, `out = format("pi = {x:.3f}", {x: 3.14159})`, nil, "pi = 3.142")
	expectRun(t, `out = format("n = {x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "n = 00042")
	expectRun(t, `out = format("{x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "00042")
	expectRun(t, `out = format("{0:{1}}", [42, "05d"])`, nil, "00042")

	// --- examples from docs/language.md "Built-in functions" section ---
	expectRun(t, `out = format("hello {x} from {y}!", {x: "kavun", y: "Kherson"})`, nil, "hello kavun from Kherson!")
	expectRun(t, `out = format("hello {0} from {1}!", ["kavun", "Kherson"])`, nil, "hello kavun from Kherson!")
	expectRun(t, `out = format("pi = {x:.3f}", {x: 3.14159})`, nil, "pi = 3.142")
	expectRun(t, `out = format("n = {x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "n = 00042")

	// --- dict and record behave identically for named lookup ---
	expectRun(t, `out = format("hi {x}", dict({x: "world"}))`, nil, "hi world")
	expectRun(t, `out = format("hi {x}", {x: "world"})`, nil, "hi world")

	// --- repeated placeholders, multi-segment templates ---
	expectRun(t, `out = format("{0}-{1}-{0}", ["a", "b"])`, nil, "a-b-a")
	expectRun(t, `out = format("{a}+{b}={a}+{b}", {a: 1, b: 2})`, nil, "1+2=1+2")

	// --- literal fspec variants ---
	expectRun(t, `out = format("{x:>5}", {x: "hi"})`, nil, "   hi")
	expectRun(t, `out = format("{x:*^7}", {x: "hi"})`, nil, "**hi***")

	// --- "Mode is determined by args type" mismatch errors ---
	expectError(t, `format("{x}", [1, 2])`, nil, "invalid_argument_type: (format) argument args expects type dict or record, got array")
	expectError(t, `format("{0}", {a: 1})`, nil, "invalid_argument_type: (format) argument args expects type array, got record")
	expectError(t, `format("{0}", dict({a: 1}))`, nil, "invalid_argument_type: (format) argument args expects type array, got dict")

	// --- "Mixing named and indexed placeholders is an error" ---
	expectError(t, `format("{0} and {x}", [])`, nil, "unsupported_format_spec: format: cannot mix named and indexed placeholders at offset 8")
	expectError(t, `format("{x} and {0}", {})`, nil, "unsupported_format_spec: format: cannot mix named and indexed placeholders at offset 8")

	// --- template syntax errors ---
	expectError(t, `format("a }", [])`, nil, "unsupported_format_spec: format: unmatched '}' at offset 2 (use '}}' for a literal '}')")
	expectError(t, `format("{}", [])`, nil, "unsupported_format_spec: format: empty placeholder '{}' at offset 0 (auto-numbering is not supported)")
	expectError(t, `format("{x", {})`, nil, "unsupported_format_spec: format: unterminated placeholder starting at offset 0")
	expectError(t, `format("{1bad}", {})`, nil, `unsupported_format_spec: format: invalid placeholder "1bad" at offset 0`)
	expectError(t, `format("{x+1}", {})`, nil, `unsupported_format_spec: format: invalid placeholder "x+1" at offset 0`)
	expectError(t, `format("{ x }", {})`, nil, `unsupported_format_spec: format: invalid placeholder " x " at offset 0`)

	// --- spec parse error in literal spec ---
	expectError(t, `format("{x:zzz}", {x: 1})`, nil, `unsupported_format_spec: format: fspec: trailing characters "zz" in "zzz"`)

	// --- nested-{ref} restrictions ---
	expectError(t, `format("{x:>{w}}", {x: 1, w: 5})`, nil, "unsupported_format_spec: format: '{ref}' inside a format spec must stand alone (offset 4)")
	expectError(t, `format("{x:{a}{b}}", {x: 1, a: "0", b: "5d"})`, nil, "unsupported_format_spec: format: '{ref}' inside a format spec must stand alone (offset 6)")
	expectError(t, `format("{x:{}}", {x: 1})`, nil, "unsupported_format_spec: format: empty '{}' inside format spec at offset 3")

	// --- runtime lookup errors ---
	expectError(t, `format("{x}", {})`, nil, `invalid_value: format: missing key "x"`)
	expectError(t, `format("{0}", [])`, nil, "index_out_of_bounds: (format) 0 out of range [0, 0]")
	expectError(t, `format("{2}", ["a", "b"])`, nil, "index_out_of_bounds: (format) 2 out of range [0, 2]")

	// --- spec-by-reference runtime errors ---
	expectError(t, `format("{x:{fmt}}", {x: 1})`, nil, `invalid_value: format: missing spec ref key "fmt"`)
	expectError(t, `format("{0:{1}}", [1])`, nil, "index_out_of_bounds: (format spec ref) 1 out of range [0, 1]")
	expectError(t, `format("{x:{fmt}}", {x: 1, fmt: 2})`, nil, "invalid_argument_type: (format) argument spec ref expects type string, got int")
	expectError(t, `format("{x:{fmt}}", {x: 1, fmt: "zzz"})`, nil, `unsupported_format_spec: format: fspec: trailing characters "zz" in "zzz"`)

	// --- type's Format method rejects an unsupported spec ---
	expectError(t, `format("{x:.2f}", {x: "hi"})`, nil, `unsupported_format_spec: type string does not support format spec {0 0 0 false false 0 0 2 true false false 102 }`)
}

func TestBuiltinFunctionDelete(t *testing.T) {
	expectError(t, `delete()`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 0")
	expectError(t, `delete(1)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 1")
	expectError(t, `delete(1, 2, 3)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 3")
	expectError(t, `delete({}, "", 3)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 3")
	expectError(t, `delete(1, 1)`, nil, `not_deletable: type int does not support delete`)
	expectError(t, `delete(1.0, 1)`, nil, `not_deletable: type float does not support delete`)
	expectError(t, `delete("str", 1)`, nil, `not_deletable: type string does not support delete`)
	expectError(t, `delete(bytes("str"), 1)`, nil, `not_deletable: type bytes does not support delete`)
	expectError(t, `delete(error("err"), 1)`, nil, `not_deletable: type error does not support delete`)
	expectError(t, `delete(true, 1)`, nil, `not_deletable: type bool does not support delete`)
	expectError(t, `delete(rune('c'), 1)`, nil, `not_deletable: type rune does not support delete`)
	expectError(t, `delete(undefined, 1)`, nil, `not_deletable: type undefined does not support delete`)
	expectError(t, `delete(time(1257894000), 1)`, nil, `not_deletable: type time does not support delete`)
	expectError(t, `delete(immutable({}), "key")`, nil, `not_deletable: type immutable-record does not support delete`)
	expectError(t, `delete(immutable([]), "")`, nil, `not_deletable: type immutable-array does not support delete`)
	expectError(t, `delete([], "")`, nil, `not_deletable: type array does not support delete`)
	expectError(t, `delete({}, undefined)`, nil, `invalid_index_type: (delete key) expected string, got undefined`)

	expectRun(t, `out = delete({}, "")`, nil, MAP{})
	expectRun(t, `out = {key1: 1}; delete(out, "key1")`, nil, MAP{})
	expectRun(t, `out = {key1: 1, key2: "2"}; delete(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `out = dict({key1: 1}); delete(out, "key1")`, nil, MAP{})
	expectRun(t, `out = dict({key1: 1, key2: "2"}); delete(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `out = [1, "2", {a: "b", c: 10}]; delete(out[2], "c")`, nil, ARR{1, "2", MAP{"a": "b"}})
}

func TestBuiltinFunctionSplice(t *testing.T) {
	expectError(t, `splice()`, nil, "wrong_num_arguments: (splice) expected at least 1 argument(s), got 0")
	expectError(t, `splice(1)`, nil, `invalid_argument_type: (splice) argument first expects type array, got int`)
	expectError(t, `splice(1.0)`, nil, `invalid_argument_type: (splice) argument first expects type array, got float`)
	expectError(t, `splice("str")`, nil, `invalid_argument_type: (splice) argument first expects type array, got string`)
	expectError(t, `splice(bytes("str"))`, nil, `invalid_argument_type: (splice) argument first expects type array, got bytes`)
	expectError(t, `splice(error("err"))`, nil, `invalid_argument_type: (splice) argument first expects type array, got error`)
	expectError(t, `splice(true)`, nil, `invalid_argument_type: (splice) argument first expects type array, got bool`)
	expectError(t, `splice(rune('c'))`, nil, `invalid_argument_type: (splice) argument first expects type array, got rune`)
	expectError(t, `splice(undefined)`, nil, `invalid_argument_type: (splice) argument first expects type array, got undefined`)
	expectError(t, `splice(time(1257894000))`, nil, `invalid_argument_type: (splice) argument first expects type array, got time`)
	expectError(t, `splice(immutable({}))`, nil, `invalid_argument_type: (splice) argument first expects type array, got immutable-record`)
	expectError(t, `splice(immutable([]))`, nil, `invalid_argument_type: (splice) argument first expects type mutable array, got immutable-array`)
	expectError(t, `splice({})`, nil, `invalid_argument_type: (splice) argument first expects type array, got record`)
	expectError(t, `splice([], "str")`, nil, `invalid_argument_type: (splice) argument second expects type int, got string`)
	expectError(t, `splice([], bytes("str"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got bytes`)
	expectError(t, `splice([], error("error"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got error`)
	expectError(t, `splice([], undefined)`, nil, `invalid_argument_type: (splice) argument second expects type int, got undefined`)
	//expectError(t, `splice([], time(0))`, nil, `invalid_argument_type: (splice) argument second expects type int, got time`)
	expectError(t, `splice([], [])`, nil, `invalid_argument_type: (splice) argument second expects type int, got array`)
	expectError(t, `splice([], {})`, nil, `invalid_argument_type: (splice) argument second expects type int, got record`)
	expectError(t, `splice([], immutable([]))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-array`)
	expectError(t, `splice([], immutable({}))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-record`)
	expectError(t, `splice([], 0, "string")`, nil, `invalid_argument_type: (splice) argument third expects type int, got string`)
	expectError(t, `splice([], 0, bytes("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got bytes`)
	expectError(t, `splice([], 0, error("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got error`)
	expectError(t, `splice([], 0, undefined)`, nil, `invalid_argument_type: (splice) argument third expects type int, got undefined`)
	//expectError(t, `splice([], 0, time(0))`, nil, `invalid_argument_type: (splice) argument third expects type int, got time`)
	expectError(t, `splice([], 0, [])`, nil, `invalid_argument_type: (splice) argument third expects type int, got array`)
	expectError(t, `splice([], 0, {})`, nil, `invalid_argument_type: (splice) argument third expects type int, got record`)
	expectError(t, `splice([], 0, immutable([]))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-array`)
	expectError(t, `splice([], 0, immutable({}))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-record`)
	expectError(t, `splice([], 1)`, nil, "index_out_of_bounds")
	expectError(t, `splice([1, 2, 3], 0, -1)`, nil, "invalid_value: splice delete count must be non-negative")
	expectError(t, `splice([1, 2, 3], 99, 0, "a", "b")`, nil, "index_out_of_bounds")
	expectRun(t, `out = []; splice(out)`, nil, ARR{})
	expectRun(t, `out = ["a"]; splice(out, 1)`, nil, ARR{"a"})
	expectRun(t, `out = ["a"]; out = splice(out, 1)`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3]; splice(out, 0, 1)`, nil, ARR{2, 3})
	expectRun(t, `out = [1, 2, 3]; out = splice(out, 0, 1)`, nil, ARR{1})
	expectRun(t, `out = [1, 2, 3]; splice(out, 0, 0, "a", "b")`, nil, ARR{"a", "b", 1, 2, 3})
	expectRun(t, `out = [1, 2, 3]; out = splice(out, 0, 0, "a", "b")`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3]; splice(out, 1, 0, "a", "b")`, nil, ARR{1, "a", "b", 2, 3})
	expectRun(t, `out = [1, 2, 3]; out = splice(out, 1, 0, "a", "b")`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3]; splice(out, 1, 0, "a", "b")`, nil, ARR{1, "a", "b", 2, 3})
	expectRun(t, `out = [1, 2, 3]; splice(out, 2, 0, "a", "b")`, nil, ARR{1, 2, "a", "b", 3})
	expectRun(t, `out = [1, 2, 3]; splice(out, 3, 0, "a", "b")`, nil, ARR{1, 2, 3, "a", "b"})

	expectRun(t, `array := [1, 2, 3]; deleted := splice(array, 1, 1, "a", "b");
				out = [deleted, array]`, nil, ARR{ARR{2}, ARR{1, "a", "b", 3}})

	expectRun(t, `array := [1, 2, 3]; deleted := splice(array, 1);
		out = [deleted, array]`, nil, ARR{ARR{2, 3}, ARR{1}})

	expectRun(t, `out = []; splice(out, 0, 0, "a", "b")`, nil, ARR{"a", "b"})
	expectRun(t, `out = []; splice(out, 0, 1, "a", "b")`, nil, ARR{"a", "b"})
	expectRun(t, `out = []; out = splice(out, 0, 0, "a", "b")`, nil, ARR{})
	expectRun(t, `out = splice(splice([1, 2, 3], 0, 3), 1, 3)`, nil, ARR{2, 3})

	// splice doc examples
	expectRun(t, `v := [1, 2, 3]; deleted := splice(v, 0);
		out = [deleted, v]`, nil, ARR{ARR{1, 2, 3}, ARR{}})

	expectRun(t, `v := [1, 2, 3]; deleted := splice(v, 1);
		out = [deleted, v]`, nil, ARR{ARR{2, 3}, ARR{1}})

	expectRun(t, `v := [1, 2, 3]; deleted := splice(v, 0, 1);
		out = [deleted, v]`, nil, ARR{ARR{1}, ARR{2, 3}})

	expectRun(t, `v := ["a", "b", "c"]; deleted := splice(v, 1, 2);
		out = [deleted, v]`, nil, ARR{ARR{"b", "c"}, ARR{"a"}})

	expectRun(t, `v := ["a", "b", "c"]; deleted := splice(v, 2, 1, "d");
		out = [deleted, v]`, nil, ARR{ARR{"c"}, ARR{"a", "b", "d"}})

	expectRun(t, `v := ["a", "b", "c"]; deleted := splice(v, 0, 0, "d", "e");
		out = [deleted, v]`, nil, ARR{ARR{}, ARR{"d", "e", "a", "b", "c"}})

	expectRun(t, `v := ["a", "b", "c"]; deleted := splice(v, 1, 1, "d", "e");
		out = [deleted, v]`, nil, ARR{ARR{"b"}, ARR{"a", "d", "e", "c"}})
}

func TestImmutable(t *testing.T) {
	// primitive types are already immutable values
	// immutable expression has no effects.
	expectRun(t, `a := immutable(1); out = a`, nil, 1)
	expectRun(t, `a := 5; b := immutable(a); out = b`, nil, 5)
	expectRun(t, `a := immutable(1); a = 5; out = a`, nil, 5)

	// array
	expectError(t, `a := immutable([1, 2, 3]); a[1] = 5`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := immutable(["foo", [1,2,3]]); a[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `a := immutable(["foo", [1,2,3]]); a[1][1] = "bar"; out = a`, nil, ARR{"foo", ARR{1, "bar", 3}})
	expectError(t, `a := immutable(["foo", immutable([1,2,3])]); a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := ["foo", immutable([1,2,3])]; a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `a := immutable([1,2,3]); b := copy(a); b[1] = 5; out = b`, nil, ARR{1, 5, 3})
	expectRun(t, `a := immutable([1,2,3]); b := copy(a); b[1] = 5; out = a`, nil, ARR{1, 2, 3})
	expectRun(t, `out = immutable([1,2,3]) == [1,2,3]`, nil, true)
	expectRun(t, `out = immutable([1,2,3]) == immutable([1,2,3])`, nil, true)
	expectRun(t, `out = [1,2,3] == immutable([1,2,3])`, nil, true)
	expectRun(t, `out = immutable([1,2,3]) == [1,2]`, nil, false)
	expectRun(t, `out = immutable([1,2,3]) == immutable([1,2])`, nil, false)
	expectRun(t, `out = [1,2,3] == immutable([1,2])`, nil, false)
	expectRun(t, `out = immutable([1, 2, 3, 4])[1]`, nil, 2)
	expectRun(t, `out = immutable([1, 2, 3, 4])[1:3]`, nil, ARR{2, 3})
	expectRun(t, `a := immutable([1,2,3]); a = 5; out = a`, nil, 5)

	// map
	expectError(t, `a := immutable({b: 1, c: 2}); a.b = 5`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectError(t, `a := immutable({b: 1, c: 2}); a["b"] = "bar"`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectRun(t, `a := immutable({b: 1, c: [1,2,3]}); a.c[1] = "bar"; out = a`, nil, MAP{"b": 1, "c": ARR{1, "bar", 3}})
	expectError(t, `a := immutable({b: 1, c: immutable([1,2,3])}); a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := {b: 1, c: immutable([1,2,3])}; a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `out = immutable({a:1,b:2}) == {a:1,b:2}`, nil, true)
	expectRun(t, `out = immutable({a:1,b:2}) == immutable({a:1,b:2})`, nil, true)
	expectRun(t, `out = {a:1,b:2} == immutable({a:1,b:2})`, nil, true)
	expectRun(t, `out = immutable({a:1,b:2}) == {a:1,b:3}`, nil, false)
	expectRun(t, `out = immutable({a:1,b:2}) == immutable({a:1,b:3})`, nil, false)
	expectRun(t, `out = {a:1,b:2} == immutable({a:1,b:3})`, nil, false)
	expectRun(t, `out = immutable({a:1,b:2}).b`, nil, 2)
	expectRun(t, `out = immutable({a:1,b:2})["b"]`, nil, 2)
	expectRun(t, `a := immutable({a:1,b:2}); a = 5; out = 5`, nil, 5)
	expectRun(t, `a := immutable({a:1,b:2}); out = a.c`, nil, core.Undefined)

	expectRun(t, `a := immutable({b: 5, c: "foo"}); out = a.b`, nil, 5)
	expectError(t, `a := immutable({b: 5, c: "foo"}); a.b = 10`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
}

func TestBytesN(t *testing.T) {
	expectRun(t, `out = bytes(0)`, nil, make([]byte, 0))
	expectRun(t, `out = bytes(10)`, nil, make([]byte, 10))
	expectRun(t, `out = bytes(1000)`, nil, make([]byte, 1000))
}

func TestCall(t *testing.T) {
	expectRun(t, `a := { b: func(x) { return x + 2 } }; out = a.b(5)`, nil, 7)
	expectRun(t, `a := { b: { c: func(x) { return x + 2 } } }; out = a.b.c(5)`, nil, 7)
	expectRun(t, `a := { b: { c: func(x) { return x + 2 } } }; out = a["b"].c(5)`, nil, 7)
	expectError(t, `a := 1
b := func(a, c) {
   c(a)
}

c := func(a) {
   a()
}
b(a, c)
`, nil, "Runtime Error: not_callable: type int is not callable\n\tat test:7:4\n\tat test:3:6\n\tat test:9:6")
}

func TestCondExpr(t *testing.T) {
	expectRun(t, `out = true ? 5 : 10`, nil, 5)
	expectRun(t, `out = false ? 5 : 10`, nil, 10)
	expectRun(t, `out = (1 == 1) ? 2 + 3 : 12 - 2`, nil, 5)
	expectRun(t, `out = (1 != 1) ? 2 + 3 : 12 - 2`, nil, 10)
	expectRun(t, `out = (1 == 1) ? true ? 10 - 8 : 1 + 3 : 12 - 2`, nil, 2)
	expectRun(t, `out = (1 == 1) ? false ? 10 - 8 : 1 + 3 : 12 - 2`, nil, 4)

	expectRun(t, `
out = 0
f1 := func() { out += 10 }
f2 := func() { out = -out }
true ? f1() : f2()
`, nil, 10)
	expectRun(t, `
out = 5
f1 := func() { out += 10 }
f2 := func() { out = -out }
false ? f1() : f2()
`, nil, -5)
	expectRun(t, `
f1 := func(a) { return a + 2 }
f2 := func(a) { return a - 2 }
f3 := func(a) { return a + 10 }
f4 := func(a) { return -a }

f := func(c) {
	return c == 0 ? f1(c) : f2(c) ? f3(c) : f4(c)
}

out = [f(0), f(1), f(2)]
`, nil, ARR{2, 11, -2})

	expectRun(t, `f := func(a) { return -a }; out = f(true ? 5 : 3)`, nil, -5)
	expectRun(t, `out = [false?5:10, true?1:2]`, nil, ARR{10, 1})

	expectRun(t, `
out = 1 > 2 ?
	1 + 2 + 3 :
	10 - 5`, nil, 5)
}

func TestEquality(t *testing.T) {
	testEquality(t, `1`, `1`, true)
	testEquality(t, `1`, `2`, false)

	testEquality(t, `1.0`, `1.0`, true)
	testEquality(t, `1.0`, `1.1`, false)

	testEquality(t, `true`, `true`, true)
	testEquality(t, `true`, `false`, false)

	testEquality(t, `"foo"`, `"foo"`, true)
	testEquality(t, `"foo"`, `"bar"`, false)

	testEquality(t, `'f'`, `'f'`, true)
	testEquality(t, `'f'`, `'b'`, false)

	testEquality(t, `[]`, `[]`, true)
	testEquality(t, `[1]`, `[1]`, true)
	testEquality(t, `[1]`, `[1, 2]`, false)
	testEquality(t, `["foo", "bar"]`, `["foo", "bar"]`, true)
	testEquality(t, `["foo", "bar"]`, `["bar", "foo"]`, false)

	testEquality(t, `{}`, `{}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2, a: 1}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2}`, false)
	testEquality(t, `{a: 1, b: {}}`, `{b: {}, a: 1}`, true)

	testEquality(t, `1`, `"foo"`, false)

	expectRun(t, "out = true == true", nil, true)
	expectRun(t, "out = true != false", nil, true)
	expectRun(t, "out = false != true", nil, true)

	expectRun(t, "out = true == 1", nil, true)
	expectRun(t, "out = 1 == true", nil, true)

	expectRun(t, "out = true == 2", nil, true)
	expectRun(t, "out = 2 != true", nil, true)
	expectRun(t, "out = true != 2", nil, false)
	expectRun(t, "out = 2 == true", nil, false)

	expectRun(t, "out = 0 == false", nil, true)
	expectRun(t, "out = 0 != true", nil, true)
	expectRun(t, "out = false == 0", nil, true)
	expectRun(t, "out = true != 0", nil, true)

	expectRun(t, `out = [1] == ["1"]`, nil, true)
	expectRun(t, `out = [1] != ["2"]`, nil, true)

	expectRun(t, `out = [1, [2]] == [1, ["2"]]`, nil, true)
	expectRun(t, `out = [1, [2]] != [1, ["3"]]`, nil, true)

	expectRun(t, `out = {a: 1} == {a: "1"}`, nil, true)
	expectRun(t, `out = {a: 1} != {a: "2"}`, nil, true)

	expectRun(t, `out = {a: 1, b: {c: 2}} == {a: 1, b: {c: "2"}}`, nil, true)
	expectRun(t, `out = {a: 1, b: {c: 2}} != {a: 1, b: {c: "3"}}`, nil, true)
}

func testEquality(t *testing.T, lhs, rhs string, expected bool) {
	// 1. equality is commutative
	// 2. equality and inequality must be always opposite
	expectRun(t, fmt.Sprintf("out = %s == %s", lhs, rhs), nil, expected)
	expectRun(t, fmt.Sprintf("out = %s == %s", rhs, lhs), nil, expected)
	expectRun(t, fmt.Sprintf("out = %s != %s", lhs, rhs), nil, !expected)
	expectRun(t, fmt.Sprintf("out = %s != %s", rhs, lhs), nil, !expected)
}

func TestForIn(t *testing.T) {
	// array
	expectRun(t, `out = 0; for x in [1, 2, 3] { out += x }`, nil, 6)                     // value
	expectRun(t, `out = 0; for i, x in [1, 2, 3] { out += i + x }`, nil, 9)              // index, value
	expectRun(t, `out = 0; func() { for i, x in [1, 2, 3] { out += i + x } }()`, nil, 9) // index, value
	expectRun(t, `out = 0; for i, _ in [1, 2, 3] { out += i }`, nil, 3)                  // index, _
	expectRun(t, `out = 0; func() { for i, _ in [1, 2, 3] { out += i  } }()`, nil, 3)    // index, _

	// record
	expectRun(t, `out = 0; for v in {a:2,b:3,c:4} { out += v }`, nil, 9)                                      // value
	expectRun(t, `out = ""; for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } }`, nil, "b")              // key, value
	expectRun(t, `out = ""; for k, _ in {a:2} { out += k }`, nil, "a")                                        // key, _
	expectRun(t, `out = 0; for _, v in {a:2,b:3,c:4} { out += v }`, nil, 9)                                   // _, value
	expectRun(t, `out = ""; func() { for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } } }()`, nil, "b") // key, value

	// string
	expectRun(t, `out = ""; for c in "abcde" { out += c }`, nil, "abcde")
	expectRun(t, `out = ""; for i, c in "abcde" { if i == 2 { continue }; out += c }`, nil, "abde")
}

func TestFor(t *testing.T) {
	expectRun(t, `
	out = 0
	for {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, `
	out = 0
	for {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, `
	out = 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}`, nil, 7) // 1 + 2 + 4

	expectRun(t, `
	out = 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, `
	out = 0
	for true {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, `
	a := 0
	for true {
		a++
		if a == 5 {
			break
		}
	}
	out = a`, nil, 5)

	expectRun(t, `
	out = 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}`, nil, 7) // 1 + 2 + 4

	expectRun(t, `
	out = 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, `
	out = 0
	func() {
		for true {
			out++
			if out == 5 {
				return
			}
		}
	}()`, nil, 5)

	expectRun(t, `
	out = 0
	for a:=1; a<=10; a++ {
		out += a
	}`, nil, 55)

	expectRun(t, `
	out = 0
	for a:=1; a<=3; a++ {
		for b:=3; b<=6; b++ {
			out += b
		}
	}`, nil, 54)

	expectRun(t, `
	out = 0
	func() {
		for {
			out++
			if out == 5 {
				break
			}
		}
	}()`, nil, 5)

	expectRun(t, `
	out = 0
	func() {
		for true {
			out++
			if out == 5 {
				break
			}
		}
	}()`, nil, 5)

	expectRun(t, `
	out = func() {
		a := 0
		for {
			a++
			if a == 5 {
				break
			}
		}
		return a
	}()`, nil, 5)

	expectRun(t, `
	out = func() {
		a := 0
		for true {
			a++
			if a== 5 {
				break
			}
		}
		return a
	}()`, nil, 5)

	expectRun(t, `
	out = func() {
		a := 0
		func() {
			for {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, 5)

	expectRun(t, `
	out = func() {
		a := 0
		func() {
			for true {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, 5)

	expectRun(t, `
	out = func() {
		sum := 0
		for a:=1; a<=10; a++ {
			sum += a
		}
		return sum
	}()`, nil, 55)

	expectRun(t, `
	out = func() {
		sum := 0
		for a:=1; a<=4; a++ {
			for b:=3; b<=5; b++ {
				sum += b
			}
		}
		return sum
	}()`, nil, 48) // (3+4+5) * 4

	expectRun(t, `
	a := 1
	for ; a<=10; a++ {
		if a == 5 {
			break
		}
	}
	out = a`, nil, 5)

	expectRun(t, `
	out = 0
	for a:=1; a<=10; a++ {
		if a == 3 {
			continue
		}
		out += a
		if a == 5 {
			break
		}
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, `
	out = 0
	for a:=1; a<=10; {
		if a == 3 {
			a++
			continue
		}
		out += a
		if a == 5 {
			break
		}
		a++
	}`, nil, 12) // 1 + 2 + 4 + 5
}

func TestFunction(t *testing.T) {
	// function with no "return" statement returns "invalid" value.
	expectRun(t, `f1 := func() {}; out = f1();`, nil, core.Undefined)
	expectRun(t, `f1 := func() {}; f2 := func() { return f1(); }; f1(); out = f2();`, nil, core.Undefined)
	expectRun(t, `f := func(x) { x; }; out = f(5);`, nil, core.Undefined)

	expectRun(t, `f := func(...x) { return x; }; out = f(1,2,3);`, nil, ARR{1, 2, 3})
	expectRun(t, `f := func(a, b, ...x) { return [a, b, x]; }; out = f(8,9,1,2,3);`, nil, ARR{8, 9, ARR{1, 2, 3}})
	expectRun(t, `f := func(v) { x := 2; return func(a, ...b){ return [a, b, v+x]}; }; out = f(5)("a", "b");`, nil, ARR{"a", ARR{"b"}, 7})
	expectRun(t, `f := func(...x) { return x; }; out = f();`, nil, core.NewArrayValue([]core.Value{}, true))
	expectRun(t, `f := func(a, b, ...x) { return [a, b, x]; }; out = f(8, 9);`, nil, ARR{8, 9, ARR{}})
	expectRun(t, `f := func(v) { x := 2; return func(a, ...b){ return [a, b, v+x]}; }; out = f(5)("a");`, nil, ARR{"a", ARR{}, 7})

	expectError(t, `f := func(a, b, ...x) { return [a, b, x]; }; f();`, nil, "Runtime Error: wrong_num_arguments: (call) expected >=2 argument(s), got 0\n\tat test:1:46")
	expectError(t, `f := func(a, b, ...x) { return [a, b, x]; }; f(1);`, nil, "Runtime Error: wrong_num_arguments: (call) expected >=2 argument(s), got 1\n\tat test:1:48")

	expectRun(t, `f := func(x) { return x; }; out = f(5);`, nil, 5)
	expectRun(t, `f := func(x) { return x * 2; }; out = f(5);`, nil, 10)
	expectRun(t, `f := func(x, y) { return x + y; }; out = f(5, 5);`, nil, 10)
	expectRun(t, `f := func(x, y) { return x + y; }; out = f(5 + 5, f(5, 5));`, nil, 20)
	expectRun(t, `out = func(x) { return x; }(5)`, nil, 5)
	expectRun(t, `x := 10; f := func(x) { return x; }; f(5); out = x;`, nil, 10)

	expectRun(t, `
	f2 := func(a) {
		f1 := func(a) {
			return a * 2;
		};

		return f1(a) * 3;
	};

	out = f2(10);
	`, nil, 60)

	expectRun(t, `
		f1 := func(f) {
			a := [undefined]
			a[0] = func() { return f(a) }
			return a[0]()
		}

		out = f1(func(a) { return 2 })
	`, nil, 2)

	// closures
	expectRun(t, `
		newAdder := func(x) {
			return func(y) { return x + y };
		};

		add2 := newAdder(2);
		out = add2(5);
		`, nil, 7)
	expectRun(t, `
		m := {a: 1}
		for k,v in m {
			func(){
				out = k
			}()
		}
		`, nil, "a")

	expectRun(t, `
		m := {a: 1}
		for k,v in m {
			func(){
				out = v
			}()
		}
		`, nil, 1)
	// function as a argument
	expectRun(t, `
	add := func(a, b) { return a + b };
	sub := func(a, b) { return a - b };
	applyFunc := func(a, b, f) { return f(a, b) };

	out = applyFunc(applyFunc(2, 2, add), 3, sub);
	`, nil, 1)

	expectRun(t, `f1 := func() { return 5 + 10; }; out = f1();`, nil, 15)
	expectRun(t, `f1 := func() { return 1 }; f2 := func() { return 2 }; out = f1() + f2()`, nil, 3)
	expectRun(t, `f1 := func() { return 1 }; f2 := func() { return f1() + 2 }; f3 := func() { return f2() + 3 }; out = f3()`, nil, 6)
	expectRun(t, `f1 := func() { return 99; 100 }; out = f1();`, nil, 99)
	expectRun(t, `f1 := func() { return 99; return 100 }; out = f1();`, nil, 99)
	expectRun(t, `f1 := func() { return 33; }; f2 := func() { return f1 }; out = f2()();`, nil, 33)
	expectRun(t, `one := func() { one = 1; return one }; out = one()`, nil, 1)
	expectRun(t, `three := func() { one := 1; two := 2; return one + two }; out = three()`, nil, 3)
	expectRun(t, `three := func() { one := 1; two := 2; return one + two }; seven := func() { three := 3; four := 4; return three + four }; out = three() + seven()`, nil, 10)

	expectRun(t, `
	foo1 := func() {
		foo := 50
		return foo
	}
	foo2 := func() {
		foo := 100
		return foo
	}
	out = foo1() + foo2()`, nil, 150)
	expectRun(t, `
	g := 50;
	minusOne := func() {
		n := 1;
		return g - n;
	};
	minusTwo := func() {
		n := 2;
		return g - n;
	};
	out = minusOne() + minusTwo()
	`, nil, 97)
	expectRun(t, `
	f1 := func() {
		f2 := func() { return 1; }
		return f2
	};
	out = f1()()
	`, nil, 1)

	expectRun(t, `
	f1 := func(a) { return a; };
	out = f1(4)`, nil, 4)
	expectRun(t, `
	f1 := func(a, b) { return a + b; };
	out = f1(1, 2)`, nil, 3)

	expectRun(t, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	out = sum(1, 2);`, nil, 3)

	expectRun(t, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	out = sum(1, 2) + sum(3, 4);`, nil, 10)

	expectRun(t, `
	sum := func(a, b) {
		c := a + b
		return c
	};
	outer := func() {
		return sum(1, 2) + sum(3, 4)
	};
	out = outer();`, nil, 10)

	expectRun(t, `
	g := 10;

	sum := func(a, b) {
		c := a + b;
		return c + g;
	}

	outer := func() {
		return sum(1, 2) + sum(3, 4) + g;
	}

	out = outer() + g
	`, nil, 50)

	expectError(t, `func() { return 1; }(1)`, nil, "wrong_num_arguments")
	expectError(t, `func(a) { return a; }()`, nil, "wrong_num_arguments")
	expectError(t, `func(a, b) { return a + b; }(1)`, nil, "wrong_num_arguments")

	expectRun(t, `
		f1 := func(a) {
			return func() { return a; };
		};
		f2 := f1(99);
		out = f2()
		`, nil, 99)

	expectRun(t, `
		f1 := func(a, b) {
			return func(c) { return a + b + c };
		};

		f2 := f1(1, 2);
		out = f2(8);
		`, nil, 11)
	expectRun(t, `
		f1 := func(a, b) {
			c := a + b;
			return func(d) { return c + d };
		};
		f2 := f1(1, 2);
		out = f2(8);
		`, nil, 11)
	expectRun(t, `
		f1 := func(a, b) {
			c := a + b;
			return func(d) {
				e := d + c;
				return func(f) { return e + f };
			}
		};
		f2 := f1(1, 2);
		f3 := f2(3);
		out = f3(8);
		`, nil, 14)
	expectRun(t, `
		a := 1;
		f1 := func(b) {
			return func(c) {
				return func(d) { return a + b + c + d }
			};
		};
		f2 := f1(2);
		f3 := f2(3);
		out = f3(8);
		`, nil, 14)
	expectRun(t, `
		f1 := func(a, b) {
			one := func() { return a; };
			two := func() { return b; };
			return func() { return one() + two(); }
		};
		f2 := f1(9, 90);
		out = f2();
		`, nil, 99)

	// global function recursion
	expectRun(t, `
		fib := func(x) {
			if x == 0 {
				return 0
			} else if x == 1 {
				return 1
			} else {
				return fib(x-1) + fib(x-2)
			}
		}
		out = fib(15)`, nil, 610)

	// local function recursion
	expectRun(t, `
out = func() {
	sum := func(x) {
		return x == 0 ? 0 : x + sum(x-1)
	}
	return sum(5)
}()`, nil, 15)

	expectError(t, `return 5`, nil, "return not allowed outside function")

	// closure and block scopes
	expectRun(t, `
func() {
	a := 10
	func() {
		b := 5
		if true {
			out = a + 5
		}
	}()
}()`, nil, 15)
	expectRun(t, `
func() {
	a := 10
	b := func() { return 5 }
	func() {
		if b() {
			out = a + b()
		}
	}()
}()`, nil, 15)
	expectRun(t, `
func() {
	a := 10
	func() {
		b := func() { return 5 }
		func() {
			if true {
				out = a + b()
			}
		}()
	}()
}()`, nil, 15)

	// function skipping return
	expectRun(t, `out = func() {}()`, nil, core.Undefined)
	expectRun(t, `out = func(v) { if v { return true } }(1)`, nil, true)
	expectRun(t, `out = func(v) { if v { return true } }(0)`, nil, core.Undefined)
	expectRun(t, `out = func(v) { if v { } else { return true } }(1)`, nil, core.Undefined)
	expectRun(t, `out = func(v) { if v { return } }(1)`, nil, core.Undefined)
	expectRun(t, `out = func(v) { if v { return } }(0)`, nil, core.Undefined)
	expectRun(t, `out = func(v) { if v { } else { return } }(1)`, nil, core.Undefined)
	expectRun(t, `out = func(v) { for ;;v++ { if v == 3 { return true } } }(1)`, nil, true)
	expectRun(t, `out = func(v) { for ;;v++ { if v == 3 { break } } }(1)`, nil, core.Undefined)

	// 'f' in RHS at line 4 must reference global variable 'f'
	expectRun(t, `
f := func() { return 2 }
out = (func() {
	f := f()
	return f
})()
	`, nil, 2)
}

func TestBlocksInGlobalScope(t *testing.T) {
	expectRun(t, `
f := undefined
if true {
	a := 1
	f = func() {
		a = 2
	}
}
b := 3
f()
out = b`,
		nil, 3)

	expectRun(t, `
func() {
	f := undefined
	if true {
		a := 10
		f = func() {
			a = 20
		}
	}
	b := 5
	f()
	out = b
}()
	`,
		nil, 5)

	expectRun(t, `
f := undefined
if true {
	a := 1
	b := 2
	f = func() {
		a = 3
		b = 4
	}
}
c := 5
d := 6
f()
out = c + d`,
		nil, 11)

	expectRun(t, `
fn := undefined
if true {
	a := 1
	b := 2
	if true {
		c := 3
		d := 4
		fn = func() {
			a = 5
			b = 6
			c = 7
			d = 8
		}
	}
}
e := 9
f := 10
fn()
out = e + f`,
		nil, 19)

	expectRun(t, `
out = 0
func() {
	for x in [1, 2, 3] {
		out += x
	}
}()`,
		nil, 6)

	expectRun(t, `
out = 0
for x in [1, 2, 3] {
	out += x
}`,
		nil, 6)
}

func TestIf(t *testing.T) {
	expectRun(t, `if (true) { out = 10 }`, nil, 10)
	expectRun(t, `if (false) { out = 10 }`, nil, core.Undefined)
	expectRun(t, `if (false) { out = 10 } else { out = 20 }`, nil, 20)
	expectRun(t, `if (1) { out = 10 }`, nil, 10)
	expectRun(t, `if (0) { out = 10 } else { out = 20 }`, nil, 20)
	expectRun(t, `if (1 < 2) { out = 10 }`, nil, 10)
	expectRun(t, `if (1 > 2) { out = 10 }`, nil, core.Undefined)
	expectRun(t, `if (1 < 2) { out = 10 } else { out = 20 }`, nil, 10)
	expectRun(t, `if (1 > 2) { out = 10 } else { out = 20 }`, nil, 20)

	expectRun(t, `if (1 < 2) { out = 10 } else if (1 > 2) { out = 20 } else { out = 30 }`, nil, 10)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 < 2) { out = 20 } else { out = 30 }`, nil, 20)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30 }`, nil, 30)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else if (1 < 2) { out = 30 } else { out = 40 }`, nil, 30)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 < 2) { out = 20; out = 21; out = 22 } else { out = 30 }`, nil, 22)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30; out = 31; out = 32}`, nil, 32)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else { out = 22 } } else { out = 30 }`, nil, 22)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }`, nil, 23)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 == 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }`, nil, 30)
	expectRun(t, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { if (1 == 2) { out = 31 } else if (2 == 3) { out = 32 } else { out = 33 } }`, nil, 33)

	expectRun(t, `if a:=0; a<1 { out = 10 }`, nil, 10)
	expectRun(t, `a:=0; if a++; a==1 { out = 10 }`, nil, 10)

	expectRun(t, `
func() {
	a := 1
	if a++; a > 1 {
		out = a
	}
}()
`, nil, 2)
	expectRun(t, `
func() {
	a := 1
	if a++; a == 1 {
		out = 10
	} else {
		out = 20
	}
}()
`, nil, 20)
	expectRun(t, `
func() {
	a := 1

	func() {
		if a++; a > 1 {
			a++
		}
	}()

	out = a
}()
`, nil, 3)

	// expression statement in init (should not leave objects on stack)
	expectRun(t, `a := 1; if a; a { out = a }`, nil, 1)
	expectRun(t, `a := 1; if a + 4; a { out = a }`, nil, 1)

	// dead code elimination
	expectRun(t, `
out = func() {
	if false { return 1 }

	a := undefined

	a = 2
	if !a {
		b := func() {
			return is_callable(a) ? a(8) : a
		}()
		if is_error(b) {
			return b
		} else if !is_undefined(b) {
			return immutable(b)
		}
	}

	a = 3
	if a {
		b := func() {
			return is_callable(a) ? a(9) : a
		}()
		if is_error(b) {
			return b
		} else if !is_undefined(b) {
			return immutable(b)
		}
	}

	return a
}()
`, nil, 3)
}

func TestIncDec(t *testing.T) {
	expectRun(t, `out = 0; out++`, nil, 1)
	expectRun(t, `out = 0; out--`, nil, -1)
	expectRun(t, `a := 0; a++; out = a`, nil, 1)
	expectRun(t, `a := 0; a++; a--; out = a`, nil, 0)

	// this seems strange but it works because 'a += b' is
	// translated into 'a = a + b' and string type takes other types for + operator.
	expectRun(t, `a := "foo"; a++; out = a`, nil, "foo1")
	expectError(t, `a := "foo"; a--`, nil, "invalid_binary_operator: string - int")

	expectError(t, `a++`, nil, "unresolved reference") // not declared
	expectError(t, `a--`, nil, "unresolved reference") // not declared
	expectError(t, `4++`, nil, "unresolved reference")
}

func TestLogical(t *testing.T) {
	expectRun(t, `out = true && true`, nil, true)
	expectRun(t, `out = true && false`, nil, false)
	expectRun(t, `out = false && true`, nil, false)
	expectRun(t, `out = false && false`, nil, false)
	expectRun(t, `out = !true && true`, nil, false)
	expectRun(t, `out = !true && false`, nil, false)
	expectRun(t, `out = !false && true`, nil, true)
	expectRun(t, `out = !false && false`, nil, false)

	expectRun(t, `out = true || true`, nil, true)
	expectRun(t, `out = true || false`, nil, true)
	expectRun(t, `out = false || true`, nil, true)
	expectRun(t, `out = false || false`, nil, false)
	expectRun(t, `out = !true || true`, nil, true)
	expectRun(t, `out = !true || false`, nil, false)
	expectRun(t, `out = !false || true`, nil, true)
	expectRun(t, `out = !false || false`, nil, true)

	expectRun(t, `out = 1 && 2`, nil, 2)
	expectRun(t, `out = 1 || 2`, nil, 1)
	expectRun(t, `out = 1 && 0`, nil, 0)
	expectRun(t, `out = 1 || 0`, nil, 1)
	expectRun(t, `out = 1 && (0 || 2)`, nil, 2)
	expectRun(t, `out = 0 || (0 || 2)`, nil, 2)
	expectRun(t, `out = 0 || (0 && 2)`, nil, 0)
	expectRun(t, `out = 0 || (2 && 0)`, nil, 0)

	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() && f()`, nil, 7)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() && t()`, nil, 7)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() || t()`, nil, 3)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() || f()`, nil, 3)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() && f()`, nil, 3)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() && t()`, nil, 3)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() || t()`, nil, 7)
	expectRun(t, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() || f()`, nil, 7)
}

func TestBangOperator(t *testing.T) {
	expectRun(t, `out = !true`, nil, false)
	expectRun(t, `out = !false`, nil, true)
	expectRun(t, `out = !0`, nil, true)
	expectRun(t, `out = !5`, nil, false)
	expectRun(t, `out = !!true`, nil, true)
	expectRun(t, `out = !!false`, nil, false)
	expectRun(t, `out = !!5`, nil, true)
}

func TestReturn(t *testing.T) {
	expectRun(t, `out = func() { return 10; }()`, nil, 10)
	expectRun(t, `out = func() { return 10; return 9; }()`, nil, 10)
	expectRun(t, `out = func() { return 2 * 5; return 9 }()`, nil, 10)
	expectRun(t, `out = func() { 9; return 2 * 5; return 9 }()`, nil, 10)

	expectRun(t, `
	out = func() {
		if (10 > 1) {
			if (10 > 1) {
				return 10;
	  		}

	  		return 1;
		}
	}()`, nil, 10)

	expectRun(t, `f1 := func() { return 2 * 5; }; out = f1()`, nil, 10)
}

func TestVMScopes(t *testing.T) {
	// shadowed global variable
	expectRun(t, `
c := 5
if a := 3; a {
	c := 6
} else {
	c := 7
}
out = c
`, nil, 5)

	// shadowed local variable
	expectRun(t, `
func() {
	c := 5
	if a := 3; a {
		c := 6
	} else {
		c := 7
	}
	out = c
}()
`, nil, 5)

	// 'b' is declared in 2 separate blocks
	expectRun(t, `
c := 5
if a := 3; a {
	b := 8
	c = b
} else {
	b := 9
	c = b
}
out = c
`, nil, 8)

	// shadowing inside for statement
	expectRun(t, `
a := 4
b := 5
for i:=0;i<3;i++ {
	b := 6
	for j:=0;j<2;j++ {
		b := 7
		a = i*j
	}
}
out = a`, nil, 2)

	// shadowing inside for statement with var init
	expectRun(t, `
a := 0
for var i = 0; i < 3; i++ {
	a += i
}
out = a`, nil, 3)

	// shadowing variable declared in init statement
	expectRun(t, `
if a := 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, `
a := 4
if a := 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, `
a := 4
if a := 0; a {
	a := 6
	out = a
} else {
	a := 7
	out = a
}`, nil, 7)
	expectRun(t, `
a := 4
if a := 0; a {
	out = a
} else {
	out = a
}`, nil, 0)

	// shadowing variable declared in init statement using var
	expectRun(t, `
a := 4
if var a = 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, `
a := 4
if var a = 0; a {
	out = 1
} else {
	out = a
}`, nil, 0)

	// shadowing function level
	expectRun(t, `
a := 5
func() {
	a := 6
	a = 7
}()
out = a
`, nil, 5)
	expectRun(t, `
a := 5
func() {
	if a := 7; true {
		a = 8
	}
}()
out = a
`, nil, 5)
}

func TestSelector(t *testing.T) {
	expectRun(t, `a := {k1: 5, k2: "foo"}; out = a.k1`, nil, 5)
	expectRun(t, `a := {k1: 5, k2: "foo"}; out = a.k2`, nil, "foo")
	expectRun(t, `a := {k1: 5, k2: "foo"}; out = a.k3`, nil, core.Undefined)

	expectRun(t, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
out = a.b.c`, nil, 4)

	expectRun(t, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
b := a.x.c`, nil, core.Undefined)

	expectRun(t, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
b := a.x.y`, nil, core.Undefined)

	expectRun(t, `a := {b: 1, c: "foo"}; a.b = 2; out = a.b`, nil, 2)
	expectRun(t, `a := {b: 1, c: "foo"}; a.c = 2; out = a.c`, nil, 2) // type not checked on sub-field
	expectRun(t, `a := {b: {c: 1}}; a.b.c = 2; out = a.b.c`, nil, 2)
	expectRun(t, `a := {b: 1}; a.c = 2; out = a`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, `a := {b: {c: 1}}; a.b.d = 2; out = a`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, `func() { a := {b: 1, c: "foo"}; a.b = 2; out = a.b }()`, nil, 2)
	expectRun(t, `func() { a := {b: 1, c: "foo"}; a.c = 2; out = a.c }()`, nil, 2) // type not checked on sub-field
	expectRun(t, `func() { a := {b: {c: 1}}; a.b.c = 2; out = a.b.c }()`, nil, 2)
	expectRun(t, `func() { a := {b: 1}; a.c = 2; out = a }()`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, `func() { a := {b: {c: 1}}; a.b.d = 2; out = a }()`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, `func() { a := {b: 1, c: "foo"}; func() { a.b = 2 }(); out = a.b }()`, nil, 2)
	expectRun(t, `func() { a := {b: 1, c: "foo"}; func() { a.c = 2 }(); out = a.c }()`, nil, 2) // type not checked on sub-field
	expectRun(t, `func() { a := {b: {c: 1}}; func() { a.b.c = 2 }(); out = a.b.c }()`, nil, 2)
	expectRun(t, `func() { a := {b: 1}; func() { a.c = 2 }(); out = a }()`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, `func() { a := {b: {c: 1}}; func() { a.b.d = 2 }(); out = a }()`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
out = [a.b[2], a.c.d, a.c.e, a.c.f[1]]
`, nil, ARR{3, 8, "foo", 8})

	expectRun(t, `
func() {
	a := [1, 2, 3]
	b := 9
	a[1] = b
	b = 7     // make sure a[1] has a COPY of value of 'b'
	out = a[1]
}()
`, nil, 9)

	expectError(t, `a := {b: {c: 1}}; a.d.c = 2`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
	expectError(t, `a := [1, 2, 3]; a.b = 2`, nil, "invalid_index_type: (index assign) expected int, got string")
	expectError(t, `a := "foo"; a.b = 2`, nil, "not_assignable: type string does not support assignment via indexing or field access")
	expectError(t, `func() { a := {b: {c: 1}}; a.d.c = 2 }()`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
	expectError(t, `func() { a := [1, 2, 3]; a.b = 2 }()`, nil, "invalid_index_type")
	expectError(t, `func() { a := "foo"; a.b = 2 }()`, nil, "not_assignable: type string does not support assignment via indexing or field access")
}

func TestVMNewStackOverflowError(t *testing.T) {
	expectError(t, `f := func() { return f() + 1 }; f()`, nil, "stack_overflow")
}

func TestTailCall(t *testing.T) {
	expectRun(t, `
	fac := func(n, a) {
		if n == 1 {
			return a
		}
		return fac(n-1, n*a)
	}
	out = fac(5, 1)`, nil, 120)

	expectRun(t, `
	fac := func(n, a) {
		if n == 1 {
			return a
		}
		x := {foo: fac} // indirection for test
		return x.foo(n-1, n*a)
	}
	out = fac(5, 1)`, nil, 120)

	expectRun(t, `
	fib := func(x, s) {
		if x == 0 {
			return 0 + s
		} else if x == 1 {
			return 1 + s
		}
		return fib(x-1, fib(x-2, s))
	}
	out = fib(15, 0)`, nil, 610)

	expectRun(t, `
	fib := func(n, a, b) {
		if n == 0 {
			return a
		} else if n == 1 {
			return b
		}
		return fib(n-1, b, a + b)
	}
	out = fib(15, 0, 1)`, nil, 610)

	// global variable and no return value
	expectRun(t, `
			out = 0
			foo := func(a) {
			   if a == 0 {
			       return
			   }
			   out += a
			   foo(a-1)
			}
			foo(10)`, nil, 55)

	expectRun(t, `
	f1 := func() {
		f2 := 0    // TODO: this might be fixed in the future
		f2 = func(n, s) {
			if n == 0 { return s }
			return f2(n-1, n + s)
		}
		return f2(5, 0)
	}
	out = f1()`, nil, 15)

	// tail-call replacing loop
	// without tail-call optimization, this code will cause stack_overflow
	expectRun(t, `
iter := func(n, max) {
	if n == max {
		return n
	}

	return iter(n+1, max)
}
out = iter(0, 9999)
`, nil, 9999)
	expectRun(t, `
c := 0
iter := func(n, max) {
	if n == max {
		return
	}

	c++
	iter(n+1, max)
}
iter(0, 9999)
out = c
`, nil, 9999)
}

// tail call with free vars
func TestTailCallFreeVars(t *testing.T) {
	expectRun(t, `
func() {
	a := 10
	f2 := 0
	f2 = func(n, s) {
		if n == 0 {
			return s + a
		}
		return f2(n-1, n+s)
	}
	out = f2(5, 0)
}()`, nil, 25)
}

func TestSpread(t *testing.T) {
	expectRun(t, `
	f := func(...a) {
		return append(a, 3)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, `
	f := func(a, ...b) {
		return append([a], append(b, 3)...)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, `
	f := func(a, ...b) {
		return append(append([a], b), 3)
	}
	out = f(1, [2]...)
	`, nil, ARR{1, ARR{2}, 3})

	expectRun(t, `
	f1 := func(...a){
		return append([3], a...)
	}
	f2 := func(a, ...b) {
		return f1(append([a], b...)...)
	}
	out = f2([1, 2]...)
	`, nil, ARR{3, 1, 2})

	expectRun(t, `
	f := func(a, ...b) {
		return func(...a) {
			return append([3], append(a, 4)...)
		}(a, b...)
	}
	out = f([1, 2]...)
	`, nil, ARR{3, 1, 2, 4})

	expectRun(t, `
	f := func(a, ...b) {
		c := append(b, 4)
		return func(){
			return append(append([a], b...), c...)
		}()
	}
	out = f(1, immutable([2, 3])...)
	`, nil, ARR{1, 2, 3, 2, 3, 4})

	expectError(t, `func(a) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 1 argument(s), got 2")
	expectError(t, `func(a, b, c) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 3 argument(s), got 2")
}

func TestSliceIndex(t *testing.T) {
	expectError(t, `undefined[:1]`, nil, "Runtime Error: not_sliceable: type undefined does not support slicing")
	expectError(t, `123[-1:2]`, nil, "Runtime Error: not_sliceable: type int does not support slicing")
	expectError(t, `{}[:]`, nil, "Runtime Error: not_sliceable: type record does not support slicing")
	expectError(t, `a := 123[-1:2] ; a += 1`, nil, "Runtime Error: not_sliceable: type int does not support slicing")
}

func TestLambdas(t *testing.T) {
	expectRun(t, `
	foo := (a, b) => { return a + b }
	out = foo(1, 2)`, nil, 3)

	expectRun(t, `
	foo := (a) => { return a + 2 }
	out = foo(1)`, nil, 3)

	expectRun(t, `
	foo := a => { return a + 2 }
	out = foo(1)`, nil, 3)

	expectRun(t, `
	foo := () => { return 3 }
	out = foo()`, nil, 3)

	expectRun(t, `
	foo := (a, b) => a + b
	out = foo(1, 2)`, nil, 3)

	expectRun(t, `
	foo := (a) => a + 2
	out = foo(1)`, nil, 3)

	expectRun(t, `
	foo := a => a + 2
	out = foo(1)`, nil, 3)

	expectRun(t, `
	foo := () => 3
	out = foo()`, nil, 3)

	expectRun(t, `
	foo := (a, f) => f(a)
	out = foo(3, x => x*2)`, nil, 6)

	expectRun(t, `
	foo := (f, a) => f(a)
	out = foo(x => x*2, 3)`, nil, 6)
}

func TestIntegrity(t *testing.T) {
	expectRun(t, `
		x := [9, 8, 7, 6, 5, 4, 3, 2, 1]
		r1 := x.sort().filter(e => e % 2 == 0).last()
		y := dict({a: 1, b: 2, c: 3})
		r2 := y.values().sort().filter(e => e == 2).first()

		out = string([r1, r2])
	`, nil, string([]byte{8, 2}))

	expectRun(t, `
		x = [9, 8, 7, 6, 5, 4, 3, 2, 1]
		r1 = x.sort().filter(e => e % 2 == 0).last()
		y = dict({a: 1, b: 2, c: 3})
		r2 = y.values().sort().filter(e => e == 2).first()

		out = string([r1, r2])
	`, nil, string([]byte{8, 2}))

	expectRun(t, `
		out = [1, 2, 3]
			.sort()
			.filter(e => e > 1)
			.sum()
	`, nil, 5)
}

func TestInSyntax(t *testing.T) {
	// element iterator
	expectRun(t, `
		y := [1, 2, 3]
		out = 0
		for x in y {
			out += x
		}
	`, nil, 6)

	// index and element iterator
	expectRun(t, `
		y := [1, 2, 3]
		s1 := 0
		s2 := 0
		for i, x in y {
			s1 += i
			s2 += x
		}
		out = [s1, s2]
	`, nil, ARR{3, 6})

	// loop with condition
	expectRun(t, `
		y := {a: 1, b: 2, c: 3}
		c := 0
		s := 0
		ks := ["a", "b", "c"]
		for i, x in ks {
			if !(x in y) { break }
			c += 1
			s += y[x]
			delete(y, x)
		}
		out = [c, s]
	`, nil, ARR{3, 6})

	// condition
	expectRun(t, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if x in y {
			out = 1
		} else {
			out = 0
		}
	`, nil, 1)

	expectRun(t, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if (x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 1)

	expectRun(t, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if !(x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 0)

	expectRun(t, `
		y := {a: 1, b: 2, c: 3}
		x := "z"
		if (x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 0)
}

func TestVarSyntax(t *testing.T) {
	expectRun(t, `
		var x = 1
		var y = 2
		out = x + y
	`, nil, 3)

	expectRun(t, `
		var x = 1
		x = 2
		out = x
	`, nil, 2)

	expectRun(t, `
		var x
		x = 2
		out = x
	`, nil, 2)

	expectRun(t, `
		var x = 1
		func() {
			x = 2
		}()
		out = x
	`, nil, 2)

	expectRun(t, `
		var x = 1
		func() {
			var x = 2
			out = x
		}()
	`, nil, 2)

	expectRun(t, `
		var x = 1
		func() {
			var x = 2
			func() {
				x = 3
			}()
			out = x
		}()
	`, nil, 3)
}

func TestDivBy0(t *testing.T) {
	expectRun(t, `out = 1.0 / 0.0`, nil, math.Inf(0))
	expectRun(t, `out = 1.0 / 0`, nil, math.Inf(0))
	expectRun(t, `out = 1 / 0.0`, nil, math.Inf(0))
	expectError(t, `1 / 0`, nil, "division_by_zero")
}

func TestExamples(t *testing.T) {
	expectRun(t, `
out = {a: 1, b: 2}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, `
out = {a: 1,
	b: 2}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, `
out = {
	a: 1,
	b: 2
}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, `
out = {
	a: 1,
	b: 2,
}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, `
out = [1, 2, 3].sum()
`, nil, 6)

	expectRun(t, `
out = [1, 2, 3]
	.sum()
`, nil, 6)

	expectRun(t, `
out = [1, 2, 3].map(x => x*x).sum()
`, nil, 14)

	expectRun(t, `
out = [1, 2, 3]
	.map(x => x*x)
	.sum()
`, nil, 14)

	expectRun(t, `
out = [1, 2, 3]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
out = [1,
	2,
	3]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
out = [1,
	2,
	3]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
out = [
	1,
	2,
	3
]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
out = [
	1,
	2,
	3,
]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
out =
	[
		1,
		2,
		3,
	]
`, nil, ARR{1, 2, 3})

	expectRun(t, `
result := [1, 2, 3, 4, 5, 6]
  .filter(x => x % 2 == 0)
  .map(x => x * x)
  .reduce(0, (sum, x) => sum + x)
out = result
`, nil, 56)

	expectRun(t, `
orders := [
  {customer: "Ada", total: 120, paid: true},
  {customer: "Linus", total: 75, paid: false},
  {customer: "Grace", total: 210, paid: true},
  {customer: "Ken", total: 95, paid: true},
]

paid_total := orders
  .filter(order => order.paid)
  .map(order => order.total)
  .sum()

vip_customers := orders
  .filter(order => order.total >= 100)
  .map(order => order.customer)

out = [paid_total, vip_customers]
`, nil, ARR{425, ARR{"Ada", "Grace"}})
}

func TestVariableDeclarationAndShadowing(t *testing.T) {
	expectRun(t, `
x := 1
out = x
`, nil, 1)

	expectRun(t, `
x = 1
out = x
`, nil, 1)

	expectRun(t, `
x := 1
for i in [0, 1, 2] {
	x = i // assignment to outer variable
}
out = x
`, nil, 2)

	expectRun(t, `
x = 1
for i in [0, 1, 2] {
	x = i // assignment to outer variable
}
out = x
`, nil, 2)

	expectRun(t, `
x := 1
for i in [0, 1, 2] {
	x := i // declaration of new variable that shadows outer variable, so outer variable is not modified
}
out = x
`, nil, 1)

	expectRun(t, `
x = 1
for i in [0, 1, 2] {
	x := i // declaration of new variable that shadows outer variable, so outer variable is not modified
}
out = x
`, nil, 1)

	expectRun(t, `
x := 1
foo := func() {
	x = 2 // assignment to outer variable
}
foo()
out = x
`, nil, 2)

	expectRun(t, `
x = 1
foo = func() {
	x = 2 // assignment to outer variable
}
foo()
out = x
`, nil, 2)

	expectRun(t, `
x := 1
foo := func() {
	x := 2 // declaration of new variable that shadows outer variable, so outer variable is not modified
}
foo()
out = x
`, nil, 1)

	expectRun(t, `
x = 1
foo = func() {
	x := 2 // declaration of new variable that shadows outer variable, so outer variable is not modified
}
foo()
out = x
`, nil, 1)

	expectRun(t, `
x = 0
y = 0
if x = 10; x > 0 {
    y = 1
} else {
    y = 2
}
out = [x, y]
`, nil, ARR{10, 1}) // x == 10, y == 1 (= modifies outer x)

	expectRun(t, `
x = 0
y = 0
if x := 10; x > 0 {
    y = 1
} else {
    y = 2
}
out = [x, y]
`, nil, ARR{0, 1}) // x == 0, y == 1 (:= declares new local x in if block)
}

func TestRepeat(t *testing.T) {
	// Scalars -> array of n copies
	expectRun(t, `x := 1; out = x.repeat(3)`, nil, ARR{1, 1, 1})
	expectRun(t, `x := 0; out = x.repeat(0)`, nil, ARR{})
	expectRun(t, `x := 7; out = x.repeat(1)`, nil, ARR{7})
	expectRun(t, `b := true; out = b.repeat(2)`, nil, ARR{true, true})
	expectRun(t, `f := 1.5; out = f.repeat(2)`, nil, ARR{1.5, 1.5})
	expectRun(t, `out = undefined.repeat(3)`, nil, ARR{core.Undefined, core.Undefined, core.Undefined})

	// decimal & time -> array of n copies (reference scalars are immutable in user-land)
	expectRun(t, `d := decimal("1.5"); out = d.repeat(2).len()`, nil, 2)
	expectRun(t, `d := decimal("1.5"); out = d.repeat(2)[0] == d`, nil, true)
	expectRun(t, `d := decimal("1.5"); out = d.repeat(2)[1] == d`, nil, true)
	expectRun(t, `d := decimal("0").repeat(0); out = d`, nil, ARR{})
	expectRun(t, `t := time(0); out = t.repeat(3).len()`, nil, 3)

	// byte -> bytes (specialized concat)
	expectRun(t, `out = byte(65).repeat(3)`, nil, []byte{65, 65, 65})
	expectRun(t, `out = byte(0).repeat(0)`, nil, []byte{})
	expectRun(t, `out = byte(255).repeat(2)`, nil, []byte{255, 255})

	// rune -> runes (specialized concat)
	expectRun(t, `out = 'a'.repeat(3)`, nil, []rune("aaa"))
	expectRun(t, `out = 'a'.repeat(0)`, nil, []rune(""))
	expectRun(t, `out = 'こ'.repeat(2)`, nil, []rune("ここ"))

	// string -> string concat
	expectRun(t, `out = "ab".repeat(3)`, nil, "ababab")
	expectRun(t, `out = "".repeat(5)`, nil, "")
	expectRun(t, `out = "x".repeat(0)`, nil, "")
	expectRun(t, `out = "-".repeat(5)`, nil, "-----")
	expectRun(t, `out = "їЇ".repeat(2)`, nil, "їЇїЇ")

	// bytes -> bytes concat
	expectRun(t, `out = "AB".bytes().repeat(3)`, nil, []byte{65, 66, 65, 66, 65, 66})
	expectRun(t, `out = "".bytes().repeat(5)`, nil, []byte{})
	expectRun(t, `out = "x".bytes().repeat(0)`, nil, []byte{})

	// runes -> runes concat
	expectRun(t, `out = u"ab".repeat(3)`, nil, []rune("ababab"))
	expectRun(t, `out = u"".repeat(5)`, nil, []rune(""))
	expectRun(t, `out = u"x".repeat(0)`, nil, []rune(""))

	// array -> array concat
	expectRun(t, `out = [1, 2].repeat(3)`, nil, ARR{1, 2, 1, 2, 1, 2})
	expectRun(t, `out = [].repeat(5)`, nil, ARR{})
	expectRun(t, `out = [1, 2, 3].repeat(0)`, nil, ARR{})
	expectRun(t, `out = [1].repeat(1)`, nil, ARR{1})

	// chains and idioms
	expectRun(t, `out = "ab".repeat(3).len()`, nil, 6)
	expectRun(t, `out = [1, 2].repeat(3).sum()`, nil, 9)

	// negative count -> error
	expectError(t, `"ab".repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, `[1].repeat(-2)`, nil, "repeat count must be non-negative")
	expectError(t, `byte(1).repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, `'a'.repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, `(1).repeat(-1)`, nil, "repeat count must be non-negative")

	// wrong arity / arg type
	expectError(t, `"ab".repeat()`, nil, "wrong_num_arguments")
	expectError(t, `"ab".repeat(1, 2)`, nil, "wrong_num_arguments")
	expectError(t, `"ab".repeat([])`, nil, "invalid_argument_type")
}

func TestJoin(t *testing.T) {
	// array seq with string sep
	expectRun(t, `out = [1, 2, 3].join(", ")`, nil, "1, 2, 3")
	// string sep, array arg (sep-as-receiver)
	expectRun(t, `out = ", ".join([1, 2, 3])`, nil, "1, 2, 3")
	// default sep
	expectRun(t, `out = [1, 2, 3].join()`, nil, "123")
	// empty seq
	expectRun(t, `out = [].join(", ")`, nil, "")
	expectRun(t, `out = ", ".join([])`, nil, "")
	// single element
	expectRun(t, `out = [42].join(", ")`, nil, "42")
	// mixed types stringified via AsString (same as `+` operator)
	expectRun(t, `out = [1, "a", true].join(" | ")`, nil, "1 | a | true")
	// undefined is not string-coercible (consistent with `+`)
	expectError(t, `[1, undefined].join(",")`, nil, "cannot convert undefined to string")

	// runes sep (both directions) -> runes result; encode to bytes("aXbXc")
	expectRun(t, `out = bytes([1, 2, 3].join(u","))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, `out = bytes(u",".join([1, 2, 3]))`, nil, []byte{'1', ',', '2', ',', '3'})

	// rune sep -> runes result
	expectRun(t, `out = bytes([1, 2, 3].join(','))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, `out = bytes(','.join([1, 2, 3]))`, nil, []byte{'1', ',', '2', ',', '3'})

	// byte sep -> bytes result
	expectRun(t, `out = [1, 2, 3].join(byte(0x2C))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, `out = byte(0x2C).join([1, 2, 3])`, nil, []byte{'1', ',', '2', ',', '3'})

	// range as seq
	expectRun(t, `out = range(1, 4).join(",")`, nil, "1,2,3")
	expectRun(t, `out = ",".join(range(1, 4))`, nil, "1,2,3")
	expectRun(t, `out = range(0, 0).join(",")`, nil, "")

	// errors: wrong sep type for array.join
	expectError(t, `[1, 2].join(123)`, nil, "invalid_argument_type")
	// errors: wrong seq type for sep.join
	expectError(t, `", ".join("ab")`, nil, "invalid_argument_type")
	expectError(t, `", ".join(123)`, nil, "invalid_argument_type")
	// errors: arity
	expectError(t, `", ".join()`, nil, "wrong_num_arguments")
	expectError(t, `", ".join([1], [2])`, nil, "wrong_num_arguments")
	expectError(t, `[1, 2].join(",", "x")`, nil, "wrong_num_arguments")
}

func TestSplit(t *testing.T) {
	// string.split — basic literal
	expectRun(t, `out = "a,b,c".split(",")`, nil, ARR{"a", "b", "c"})
	expectRun(t, `out = "a,b,c".split(",", 1)`, nil, ARR{"a", "b,c"})
	expectRun(t, `out = "a,b,c".split(",", 0)`, nil, ARR{"a,b,c"})
	expectRun(t, `out = "a,b,c".split(",", -1)`, nil, ARR{"a", "b", "c"})
	// string.split — whitespace default
	expectRun(t, `out = "  hello  world  ".split()`, nil, ARR{"hello", "world"})
	// string.split — leading/trailing/consecutive seps preserved
	expectRun(t, `out = ",a,".split(",")`, nil, ARR{"", "a", ""})
	expectRun(t, `out = "a,,b".split(",")`, nil, ARR{"a", "", "b"})
	// string.split — sep not found
	expectRun(t, `out = "abc".split("x")`, nil, ARR{"abc"})
	// string.split — empty receiver
	expectRun(t, `out = "".split(",")`, nil, ARR{})
	expectRun(t, `out = "".split()`, nil, ARR{})
	// string.split — cross-type sep
	expectRun(t, `out = "a,b".split(',')`, nil, ARR{"a", "b"})
	expectRun(t, `out = "a,b".split(byte(0x2C))`, nil, ARR{"a", "b"})
	expectRun(t, `out = "a,b".split(u",")`, nil, ARR{"a", "b"})

	// runes.split
	expectRun(t, `out = bytes(u"a,b,c".split(",")[1])`, nil, []byte{'b'})
	expectRun(t, `out = u"a b c".split().len()`, nil, int64(3))
	expectRun(t, `out = u"".split(",").len()`, nil, int64(0))

	// bytes.split
	expectRun(t, `out = bytes("a,b,c").split(",").len()`, nil, int64(3))
	expectRun(t, `out = bytes("a,b,c").split(byte(0x2C)).len()`, nil, int64(3))
	expectRun(t, `out = bytes("a b c").split().len()`, nil, int64(3))
	expectRun(t, `out = bytes("").split(",").len()`, nil, int64(0))
	expectRun(t, `out = bytes("a,b,c").split(",", 1)[1]`, nil, []byte("b,c"))

	// errors
	expectError(t, `"a,b".split("")`, nil, "split separator must not be empty")
	expectError(t, `"a,b".split([])`, nil, "invalid_argument_type")
	expectError(t, `"a,b".split(",", "x")`, nil, "invalid_argument_type")
	expectError(t, `"a,b".split(",", 1, 2)`, nil, "wrong_num_arguments")
	expectError(t, `bytes("a,b").split([])`, nil, "invalid_argument_type")
}

func TestSplitLines(t *testing.T) {
	expectRun(t, `out = "a\nb\nc".split_lines()`, nil, ARR{"a", "b", "c"})
	expectRun(t, `out = "a\r\nb\rc\nd".split_lines()`, nil, ARR{"a", "b", "c", "d"})
	expectRun(t, `out = "trail\n".split_lines()`, nil, ARR{"trail"})
	expectRun(t, `out = "no_newline".split_lines()`, nil, ARR{"no_newline"})
	expectRun(t, `out = "".split_lines()`, nil, ARR{})
	expectRun(t, `out = "\n\n".split_lines()`, nil, ARR{"", ""})

	// runes / bytes
	expectRun(t, `out = u"a\nb".split_lines().len()`, nil, int64(2))
	expectRun(t, `out = bytes("a\nb").split_lines().len()`, nil, int64(2))

	expectError(t, `"x".split_lines("y")`, nil, "wrong_num_arguments")
}

func TestPartition(t *testing.T) {
	expectRun(t, `out = "a=1=b".partition("=")`, nil, ARR{"a", "=", "1=b"})
	expectRun(t, `out = "abc".partition("x")`, nil, ARR{"abc", "", ""})
	expectRun(t, `out = "".partition(",")`, nil, ARR{"", "", ""})
	expectRun(t, `out = "a,b".partition(',')`, nil, ARR{"a", ",", "b"})
	expectRun(t, `out = "a,b".partition(byte(0x2C))`, nil, ARR{"a", ",", "b"})

	// runes
	expectRun(t, `out = u"a=b".partition("=").len()`, nil, int64(3))
	expectRun(t, `out = bytes(u"a=b".partition("=")[1])`, nil, []byte{'='})

	// bytes
	expectRun(t, `out = bytes("k=v").partition("=").len()`, nil, int64(3))
	expectRun(t, `out = bytes("k=v").partition("=")[0]`, nil, []byte("k"))
	expectRun(t, `out = bytes("k=v").partition("=")[1]`, nil, []byte("="))
	expectRun(t, `out = bytes("k=v").partition("=")[2]`, nil, []byte("v"))
	expectRun(t, `out = bytes("abc").partition("x")[0]`, nil, []byte("abc"))

	// errors
	expectError(t, `"a".partition("")`, nil, "partition separator must not be empty")
	expectError(t, `"a".partition([])`, nil, "invalid_argument_type")
	expectError(t, `"a".partition()`, nil, "wrong_num_arguments")
	expectError(t, `bytes("a").partition([])`, nil, "invalid_argument_type")
}

func TestFlatten(t *testing.T) {
	// no nested arrays — no-op (but still produces a fresh array)
	expectRun(t, `out = [1, 2, 3].flatten()`, nil, ARR{int64(1), int64(2), int64(3)})
	// one level nesting
	expectRun(t, `out = [[1, 2], [3, 4]].flatten()`, nil, ARR{int64(1), int64(2), int64(3), int64(4)})
	// default depth = 1: deeper nesting preserved
	expectRun(t, `out = [1, [2, 3], [4, [5, 6]]].flatten()`, nil, ARR{int64(1), int64(2), int64(3), int64(4), ARR{int64(5), int64(6)}})
	// explicit depth
	expectRun(t, `out = [1, [2, 3], [4, [5, 6]]].flatten(2)`, nil, ARR{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)})
	// unbounded (negative)
	expectRun(t, `out = [1, [[2, [[3]]]]].flatten(-1)`, nil, ARR{int64(1), int64(2), int64(3)})
	expectRun(t, `out = [1, [[2, [[3]]]]].flatten(-100)`, nil, ARR{int64(1), int64(2), int64(3)})
	// depth 0 = shallow copy (no unwrap)
	expectRun(t, `out = [1, [2, [3]]].flatten(0)`, nil, ARR{int64(1), ARR{int64(2), ARR{int64(3)}}})
	// empty
	expectRun(t, `out = [].flatten()`, nil, ARR{})
	expectRun(t, `out = [].flatten(5)`, nil, ARR{})
	// non-array elements stay intact
	expectRun(t, `out = ["ab", [1, 2]].flatten()`, nil, ARR{"ab", int64(1), int64(2)})
	expectRun(t, `out = [[1], "abc", [[2, 3]]].flatten(1)`, nil, ARR{int64(1), "abc", ARR{int64(2), int64(3)}})
	// fresh top-level array (mutating result doesn't affect original)
	expectRun(t, `
		x = [[1, 2], [3, 4]]
		y = x.flatten()
		y[0] = 99
		out = x[0][0]
	`, nil, int64(1))

	// errors
	expectError(t, `[1, 2].flatten("x")`, nil, "invalid_argument_type")
	expectError(t, `[1, 2].flatten(1, 2)`, nil, "wrong_num_arguments")
}

func TestVMErrorInfo(t *testing.T) {
	expectError(t, `a := 5
a + "boo"`,
		nil, "Runtime Error: invalid_binary_operator: int + string\n\tat test:2:5")

	expectError(t, `a := 5
b := a(5)`,
		nil, "Runtime Error: not_callable: type int is not callable\n\tat test:2:8")

	expectError(t, `a := 5
b := {}
b.x.y = 10`,
		nil, "Runtime Error: not_assignable: type undefined does not support assignment via indexing or field access\n\tat test:3:3")

	expectError(t, `
a := func() {
	b := 5
	b += "foo"
}
a()`,
		nil, "Runtime Error: invalid_binary_operator: int + string\n\tat test:4:7\n\tat test:6:1")

	expectError(t, `a := 5
a + import("mod1")`, Opts().Module(
		"mod1", `export "foo"`,
	), ": invalid_binary_operator: int + string\n\tat test:2:5")

	expectError(t, `a := import("mod1")()`,
		Opts().Module(
			"mod1", `
export func() {
	b := 5
	return b + "foo"
}`), "Runtime Error: invalid_binary_operator: int + string\n\tat mod1:4:13\n\tat test:1:6")

	expectError(t, `a := import("mod1")()`,
		Opts().Module(
			"mod1", `export import("mod2")()`).
			Module(
				"mod2", `
export func() {
	b := 5
	return b + "foo"
}`), "Runtime Error: invalid_binary_operator: int + string\n\tat mod2:4:13\n\tat mod1:1:8\n\tat test:1:6")

	expectError(t, `a := [1, 2, 3]; b := a[:"invalid"];`, nil, "Runtime Error: invalid_index_type: (slice) expected int, got string")

	//expectError(t, `a := immutable([4, 5, 6]); b := a[:false];`, nil, "Runtime Error: invalid slice index type: bool")
	expectRun(t, `a := immutable([4, 5, 6]); out = string(a[:false]);`, nil, "")

	//expectError(t, `a := "hello"; b := a[:1.23];`, nil, "Runtime Error: invalid slice index type: float")
	expectRun(t, `a := "hello"; out = a[:1.23];`, nil, "h")

	//expectError(t, `a := bytes("world"); b := a[:time(1)];`, nil, "Runtime Error: invalid slice index type: time")
	expectRun(t, `a := bytes("world"); out = string(a[:time(1)]);`, nil, "w")
}

func TestVMErrorUnwrap(t *testing.T) {
	userErr := errors.New("user runtime error")

	userFunc := func(err error) core.Value {
		return core.NewBuiltinClosureValue(
			"user_func",
			func(v core.VM, args []core.Value) (core.Value, error) {
				return core.Undefined, err
			},
			0,
			false,
		)
	}

	expectError(t, `user_func()`, Opts().Symbol("user_func", userFunc(userErr)), "Runtime Error: "+userErr.Error())
	expectErrorIs(t, `user_func()`, Opts().Symbol("user_func", userFunc(userErr)), userErr)

	wrapUserErr := &customError{err: userErr, str: "custom error"}
	expectErrorIs(t, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), wrapUserErr)
	expectErrorIs(t, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), userErr)

	var asErr1 *customError
	expectErrorAs(t, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), &asErr1)
	require.True(t, asErr1.Error() == wrapUserErr.Error(), "expected error as:%v, got:%v", wrapUserErr, asErr1)

	userModule := func(err error) module {
		return module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"afunction",
					func(v core.VM, a []core.Value) (core.Value, error) {
						return core.Undefined, err
					},
					0,
					false,
					false,
				),
			},
		}
	}

	expectError(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(userErr)), "Runtime Error: "+userErr.Error())
	expectErrorIs(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(userErr)), userErr)
	expectError(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), "Runtime Error: "+wrapUserErr.Error())
	expectErrorIs(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), wrapUserErr)
	expectErrorIs(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), userErr)

	var asErr2 *customError
	expectErrorAs(t, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), &asErr2)
	require.True(t, asErr2.Error() == wrapUserErr.Error(), "expected error as:%v, got:%v", wrapUserErr, asErr2)
}

func TestCustomBuiltin(t *testing.T) {
	m := Opts().BuiltinModule("math1",
		module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"abs",
					func(v core.VM, a []core.Value) (core.Value, error) {
						r, _ := a[0].AsFloat()
						return core.FloatValue(math.Abs(r)), nil
					},
					1,
					false,
					false,
				),
			},
		})

	// builtin
	expectRun(t, `math := import("math1"); out = math.abs(1)`, m, 1.0)
	expectRun(t, `math := import("math1"); out = math.abs(-1)`, m, 1.0)
	expectRun(t, `math := import("math1"); out = math.abs(1.0)`, m, 1.0)
	expectRun(t, `math := import("math1"); out = math.abs(-1.0)`, m, 1.0)
}

func TestUserModules(t *testing.T) {
	// export none
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `fn := func() { return 5.0 }; a := 2`),
		core.Undefined)

	// export values
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `export 5`), 5)
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `export "foo"`), "foo")

	// export compound types
	expectRun(t, `out = import("mod1")`, Opts().Module("mod1", `export [1, 2, 3]`), ARR{1, 2, 3})
	expectRun(t, `out = import("mod1")`, Opts().Module("mod1", `export {a: 1, b: 2}`), MAP{"a": 1, "b": 2})

	// export value is immutable
	expectError(t, `m1 := import("mod1"); m1.a = 5`, Opts().Module("mod1", `export {a: 1, b: 2}`), "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectError(t, `m1 := import("mod1"); m1[1] = 5`, Opts().Module("mod1", `export [1, 2, 3]`), "not_assignable: type immutable-array does not support assignment via indexing or field access")

	// code after export statement will not be executed
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `a := 10; export a; a = 20`), 10)
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `a := 10; export a; a = 20; export a`), 10)

	// export function
	expectRun(t, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { return 5.0 }`), 5.0)
	// export function that reads module-global variable
	expectRun(t, `out = import("mod1")()`,
		Opts().Module("mod1", `a := 1.5; export func() { return a + 5.0 }`), 6.5)
	// export function that read local variable
	expectRun(t, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { a := 1.5; return a + 5.0 }`), 6.5)
	// export function that read free variables
	expectRun(t, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { a := 1.5; return func() { return a + 5.0 }() }`), 6.5)

	// recursive function in module
	expectRun(t, `out = import("mod1")`,
		Opts().Module(
			"mod1", `
a := func(x) {
	return x == 0 ? 0 : x + a(x-1)
}

export a(5)
`), 15)
	expectRun(t, `out = import("mod1")`,
		Opts().Module(
			"mod1", `
export func() {
	a := func(x) {
		return x == 0 ? 0 : x + a(x-1)
	}

	return a(5)
}()
`), 15)

	// (main) -> mod1 -> mod2
	expectRun(t, `out = import("mod1")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export func() { return 5.0 }`),
		5.0)
	// (main) -> mod1 -> mod2
	//        -> mod2
	expectRun(t, `import("mod1"); out = import("mod2")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export func() { return 5.0 }`),
		5.0)
	// (main) -> mod1 -> mod2 -> mod3
	//        -> mod2 -> mod3
	expectRun(t, `import("mod1"); out = import("mod2")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export import("mod3")`).
			Module("mod3", `export func() { return 5.0 }`),
		5.0)

	// cyclic imports
	// (main) -> mod1 -> mod2 -> mod1
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod1")`),
		"Compile Error: cyclic module import: mod1\n\tat mod2:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod1
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod1")`),
		"Compile Error: cyclic module import: mod1\n\tat mod3:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod2
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod2")`),
		"Compile Error: cyclic module import: mod2\n\tat mod3:1:1")

	// unknown modules
	expectError(t, `import("mod0")`,
		Opts().Module("mod1", `a := 5`), "module 'mod0' not found")
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`), "module 'mod2' not found")

	// module is immutable but its variables is not necessarily immutable.
	expectRun(t, `m1 := import("mod1"); m1.a.b = 5; out = m1.a.b`,
		Opts().Module("mod1", `export {a: {b: 3}}`),
		5)

	// make sure module has same builtin functions
	expectRun(t, `out = import("mod1")`,
		Opts().Module("mod1", `export func() { return type_name(0) }()`),
		"int")

	// 'export' statement is ignored outside module
	expectRun(t, `a := 5; export func() { a = 10 }(); out = a`,
		Opts().Skip2ndPass(), 5)

	// 'export' must be in the top-level
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `func() { export 5 }()`),
		"Compile Error: export not allowed inside function\n\tat mod1:1:10")
	expectError(t, `import("mod1")`,
		Opts().Module("mod1", `func() { func() { export 5 }() }()`),
		"Compile Error: export not allowed inside function\n\tat mod1:1:19")

	// module cannot access outer scope
	expectError(t, `a := 5; import("mod1")`,
		Opts().Module("mod1", `export a`),
		"Compile Error: unresolved reference 'a'\n\tat mod1:1:8")

	// runtime error within modules
	expectError(t, `
a := 1;
b := import("mod1");
b(a)`,
		Opts().Module("mod1", `
export func(a) {
   a()
}
`), "Runtime Error: not_callable: type int is not callable\n\tat mod1:3:4\n\tat test:4:3")

	// module skipping export
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", ``), core.Undefined)
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", `if 1 { export true }`), true)
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", `if 0 { export true }`),
		core.Undefined)
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", `if 1 { } else { export true }`),
		core.Undefined)
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", `for v:=0;;v++ { if v == 3 { export true } }`),
		true)
	expectRun(t, `out = import("mod0")`,
		Opts().Module("mod0", `for v:=0;;v++ { if v == 3 { break } }`),
		core.Undefined)

	// duplicate compiled functions
	// NOTE: module "mod" has a function with some local variable, and it's
	//  imported twice by the main script. That causes the same CompiledFunction
	//  put in constants twice and the Bytecode optimization (removing duplicate
	//  constants) should still work correctly.
	expectRun(t, `
m1 := import("mod")
m2 := import("mod")
out = m1.x
	`,
		Opts().Module("mod", `
f1 := func(a, b) {
	c := a + b + 1
	return a + b + 1
}
export { x: 1 }
`),
		1)
}

func TestCustomModuleBlockScopes(t *testing.T) {
	m := Opts().BuiltinModule("rand1",
		module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"intn",
					func(v core.VM, a []core.Value) (core.Value, error) {
						r, _ := a[0].AsInt()
						return core.IntValue(rand.Int63n(r)), nil
					},
					1,
					false,
					false,
				),
			},
		})

	// block scopes in module
	expectRun(t, `out = import("mod1")()`, m.Module(
		"mod1", `
	rand := import("rand1")
	foo := func() { return 1 }
	export func() {
		rand.intn(3)
		return foo()
	}`), 1)

	expectRun(t, `out = import("mod1")()`, m.Module(
		"mod1", `
rand := import("rand1")
foo := func() { return 1 }
export func() {
	rand.intn(3)
	if foo() {}
	return 10
}
`), 10)

	expectRun(t, `out = import("mod1")()`, m.Module(
		"mod1", `
	rand := import("rand1")
	foo := func() { return 1 }
	export func() {
		rand.intn(3)
		if true { foo() }
		return 10
	}
	`), 10)
}

func TestNamedReturn_DefaultUndefined(t *testing.T) {
	expectRun(t, `
		f := func() res {
			// no assignment to res — bare return yields undefined
			return
		}
		out = is_undefined(f())
	`, nil, true)
}

func TestNamedReturn_AssignThenBareReturn(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = 42
			return
		}
		out = f()
	`, nil, 42)
}

func TestNamedReturn_AssignNoReturnStmt(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = "hello"
		}
		out = f()
	`, nil, "hello")
}

func TestNamedReturn_ExplicitReturnOverridesNamed(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = "named"
			return "explicit"
		}
		out = f()
	`, nil, "explicit")
}

func TestNamedReturn_ParameterCollision_Errors(t *testing.T) {
	expectError(t, `
		f := func(x) x { return }
		out = f(1)
	`, nil, "named result")
}

func TestNamedReturn_UnderscoreNotAllowed(t *testing.T) {
	expectError(t, `
		f := func() _ { return }
		out = f()
	`, nil, "named result cannot be '_'")
}

// Regression: each call must reset the named-result slot to undefined.
// Previously the slot reused whatever stack value the previous call left behind, so a function that didn't assign its
// named result could observe a stale value from an unrelated earlier call.
func TestNamedReturn_SlotResetBetweenCalls(t *testing.T) {
	expectRun(t, `
		sign := func(x) s {
			if x > 0 { s = 1 }
			if x < 0 { s = -1 }
			if x == 0 { s = 0 }
		}
		maybe := func(x) r {
			if x { return }
			r = "set"
		}
		_ = sign(0)         // leaves 0 in the slot region
		out = is_undefined(maybe(true))
	`, nil, true)
}

func TestNamedReturn_ReadBeforeAssignIsUndefined(t *testing.T) {
	expectRun(t, `
		f := func() r {
			before := r
			r = 5
			return before
		}
		out = is_undefined(f())
	`, nil, true)
}

func TestNamedReturn_RecursionUsesOwnSlot(t *testing.T) {
	expectRun(t, `
		fact := func(n) r {
			if n <= 1 { r = 1; return }
			r = n * fact(n - 1)
		}
		out = fact(6)
	`, nil, 720)
}

func TestNamedReturn_ConditionalAssignment(t *testing.T) {
	expectRun(t, `
		sign := func(x) s {
			if x > 0 { s = 1 }
			if x < 0 { s = -1 }
			if x == 0 { s = 0 }
		}
		out = [sign(-7), sign(0), sign(3)]
	`, nil, ARR{-1, 0, 1})
}

func TestNamedReturn_ShadowedInInnerBlock(t *testing.T) {
	// A `:=` inside a nested block introduces a new local that shadows the named-result symbol; the outer slot is
	// untouched.
	expectRun(t, `
		f := func() r {
			r = "outer"
			if true {
				r := "inner"
				_ = r
			}
		}
		out = f()
	`, nil, "outer")
}

func TestNamedReturn_MutateThroughReference(t *testing.T) {
	expectRun(t, `
		build := func() obj {
			obj = {a: 1}
			obj.b = 2
		}
		r := build()
		out = [r.a, r.b]
	`, nil, ARR{1, 2})
}

func TestNamedReturn_CapturedByClosure(t *testing.T) {
	// The named result holds a closure that captures a sibling local.
	// Each invocation of the returned closure must observe the same captured environment (closure-over-local, not over
	// slot value).
	expectRun(t, `
		counter := func() c {
			n := 0
			c = func() { n = n + 1; return n }
		}
		inc := counter()
		out = [inc(), inc(), inc()]
	`, nil, ARR{1, 2, 3})
}

func TestNamedReturn_ImmediatelyInvoked(t *testing.T) {
	expectRun(t, `
		out = (func() r { r = 99 })()
	`, nil, 99)
}

func TestNamedReturn_ForLoopAccumulation(t *testing.T) {
	expectRun(t, `
		sumto := func(n) total {
			total = 0
			for i := 1; i <= n; i = i + 1 { total = total + i }
		}
		out = sumto(10)
	`, nil, 55)
}

func TestNamedReturn_VariadicWithNamedResult(t *testing.T) {
	expectRun(t, `
		joinall := func(sep, ...xs) joined {
			joined = ""
			for x in xs {
				if joined == "" { joined = string(x) } else { joined = joined + sep + string(x) }
			}
		}
		out = joinall(",", 1, 2, 3)
	`, nil, "1,2,3")
}

func TestNamedReturn_NameMayShadowBuiltin(t *testing.T) {
	// The named-result identifier is just a local symbol; it can use the same spelling as a builtin (here `len`)
	// without ambiguity.
	expectRun(t, `
		f := func() len {
			len = 7
		}
		out = f()
	`, nil, 7)
}

func TestNamedReturn_BareReturnInLoopUsesNamedSlot(t *testing.T) {
	// A bare `return` inside a loop must yield the current named-result value, not what the call stack happens to hold.
	expectRun(t, `
		find := func(arr, target) idx {
			idx = -1
			for i := 0; i < len(arr); i = i + 1 {
				if arr[i] == target { idx = i; return }
			}
		}
		out = [find([10, 20, 30, 40], 30), find([10, 20, 30, 40], 99)]
	`, nil, ARR{2, -1})
}

func TestNamedReturn_ExplicitReturnExprIgnoresNamedSlot(t *testing.T) {
	expectRun(t, `
		f := func() r {
			r = 1
			return r + 100  // expression value wins
		}
		out = f()
	`, nil, 101)
}

func TestNamedReturn_ReassignMultipleTimes(t *testing.T) {
	expectRun(t, `
		f := func() r {
			r = 1
			r = r + 10
			r = r * 2
		}
		out = f()
	`, nil, 22)
}

func TestDefer_RunsOnExit(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "a") }()
			log = append(log, "b")
		}
		f()
		out = log
	`, nil, ARR{"b", "a"})
}

func TestDefer_LIFOOrder(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, 1) }()
			defer func() { log = append(log, 2) }()
			defer func() { log = append(log, 3) }()
		}
		f()
		out = log
	`, nil, ARR{3, 2, 1})
}

func TestDefer_ArgsCapturedAtDeferTime(t *testing.T) {
	// Plain-call defer evaluates its argument expressions at defer statement time, not at call time.
	expectRun(t, `
		seen := undefined
		record := func(v) { seen = v }
		f := func() {
			x := 10
			defer record(x)
			x = 20
		}
		f()
		out = seen
	`, nil, 10)
}

func TestDefer_RunsOnExplicitReturn(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "deferred") }()
			return
		}
		f()
		out = log
	`, nil, ARR{"deferred"})
}

func TestDefer_OutsideFunction_Errors(t *testing.T) {
	expectError(t, `defer foo()`, nil, "defer not allowed outside function")
}

func TestDefer_NonCall_Errors(t *testing.T) {
	testFileSet := parser.NewFileSet()
	src := `f := func() { defer 1+1 }`
	testFile := testFileSet.AddFile("test", -1, len(src))
	p := parser.NewParser(testFile, []byte(src), nil)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatal("expected parse error for non-call defer, got none")
	}
}

func TestRecover_OutsideDeferred_ReturnsUndefined(t *testing.T) {
	expectRun(t, `
		f := func() { return is_undefined(recover()) }
		out = f()
	`, nil, true)
}

func TestRecover_NoErrorInDeferred_ReturnsUndefined(t *testing.T) {
	expectRun(t, `
		got := undefined
		f := func() {
			defer func() {
				got = recover()
			}()
		}
		f()
		out = is_undefined(got)
	`, nil, true)
}

func TestRecover_CatchesVMError(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = "caught"
				}
			}()
			x := 1 / 0
			res = "no_error"
		}
		out = f()
	`, nil, "caught")
}

func TestRecover_VMError_IsRuntime(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.is_runtime()
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, true)
}

func TestRecover_VMError_HasKind(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.kind()
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, "division_by_zero")
}

func TestRecover_RaiseUserError(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.value()
				}
			}()
			raise(error({code: "boom"}))
		}
		v := f()
		out = v.code
	`, nil, "boom")
}

func TestRecover_RaisedUserError_IsNotRuntime(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.is_runtime()
				}
			}()
			raise(error("nope"))
		}
		out = f()
	`, nil, false)
}

func TestRecover_OnlyDirectlyInDeferred(t *testing.T) {
	// recover() must be called directly from the deferred function; indirection through another call returns undefined,
	// so the raised error is not cleared and propagates out.
	expectError(t, `
		inner := func() { return recover() }
		f := func() {
			defer func() {
				inner()
			}()
			raise(error("escapes_through_inner"))
		}
		f()
	`, nil, "escapes_through_inner")
}

func TestRecover_ErrorEscapesIfNotRecovered(t *testing.T) {
	expectError(t, `
		f := func() {
			defer func() {
				// don't call recover()
			}()
			raise(error("escapes"))
		}
		f()
	`, nil, "escapes")
}

func TestDefer_RunsBeforeUnrecoveredErrorEscapes(t *testing.T) {
	expectError(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "did defer") }()
			raise(error("oops"))
		}
		f()
	`, nil, "oops")
}

func TestRecover_NamedResultUpdatedByDefer(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				if recover() != undefined {
					res = "rescued"
				}
			}()
			res = "ok"
			raise(error("bang"))
		}
		out = f()
	`, nil, "rescued")
}

func TestDefer_AccessAndModifyNamedResult(t *testing.T) {
	expectRun(t, `
		f := func(x) res {
			defer func() {
				res = res + 100
			}()
			res = x
		}
		out = f(5)
	`, nil, 105)
}

func TestDefer_LaterDeferStillRunsAfterRecover(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() res {
			defer func() { log = append(log, "outer") }()
			defer func() {
				if recover() != undefined {
					log = append(log, "recovered")
				}
			}()
			raise(error("boom"))
		}
		f()
		out = log
	`, nil, ARR{"recovered", "outer"})
}

func TestDefer_RaisedInsideDefer_CanBeRecoveredByEarlierDefer(t *testing.T) {
	// defers run LIFO; an earlier-registered defer (= later to run) can recover an error raised by a later-registered
	// defer (= run earlier).
	expectRun(t, `
		f := func() res {
			defer func() {
				if recover() != undefined {
					res = "outer_caught"
				}
			}()
			defer func() {
				raise(error("from_inner_defer"))
			}()
			res = "ok"
		}
		out = f()
	`, nil, "outer_caught")
}

// is_runtime() returns false for user errors and true for runtime ones.
func TestRecover_IsRuntime_ForRuntimeError(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.is_runtime()
    }
  }()
  x := 1 / 0
}
out = f()
`, nil, true)
}

func TestRecover_IsRuntime_ForUserError(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.is_runtime()
    }
  }()
  raise(error("oops"))
}
out = f()
`, nil, false)
}

// kind() reports specific runtime error kinds; new "not_iterable" tag should surface when iterating a non-iterable value.
func TestRecover_NotIterable_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  for i in true {  // bool is not_iterable
    _ = i
  }
}
out = f()
`, nil, "not_iterable")
}

// not_callable kind is exposed via recover().
func TestRecover_NotCallable_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  x := 42
  x()
}
out = f()
`, nil, "not_callable")
}

// wrong_num_arguments is exposed via recover().
func TestRecover_WrongNumArguments_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  g := func(a, b) { return a + b }
  g(1)
}
out = f()
`, nil, "wrong_num_arguments")
}

// User-raised errors carry an empty kind (kind() returns "").
func TestRecover_UserError_KindIsUser(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  raise(error("boom"))
}
out = f()
`, nil, "user")
}

// Critical (Fatal) Go errors raised by host-supplied builtins must bypass deferred recover() and escape directly to the host.
func TestRecover_FatalErrorBypassesRecover(t *testing.T) {
	fatalBuiltin := core.NewBuiltinClosureValue(
		"do_fatal",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, errs.NewFatalError("custom_fatal", "host requested abort")
		}, 0, false)

	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // tries to swallow but cannot
  do_fatal()
}
f()
`,
		Opts().Symbol("do_fatal", fatalBuiltin).Skip2ndPass(),
		"custom_fatal: host requested abort",
	)
}

// Recoverable Go errors raised by host-supplied builtins are caught by deferred recover().
func TestRecover_RecoverableErrorIsCaught(t *testing.T) {
	recBuiltin := core.NewBuiltinClosureValue(
		"do_logical",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, errs.NewRecoverableError("custom_kind", "user level mistake")
		}, 0, false)

	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined { res = e.kind() }
  }()
  do_logical()
}
out = f()
`,
		Opts().Symbol("do_logical", recBuiltin).Skip2ndPass(),
		"custom_kind",
	)
}

// Script-level fatal errors raised via `error(payload, true)` must bypass deferred recover() and escape directly to the host.
func TestRecover_ScriptFatalErrorBypassesRecover(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // tries to swallow but cannot
  raise(error("boom", true))
}
f()
`,
		Opts().Skip2ndPass(),
		"boom",
	)
}

// raise(err, true) promotes an otherwise-recoverable error to fatal so recover() cannot catch it.
func TestRecover_RaiseFatalFlagPromotesToFatal(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()
  raise(error("boom"), true)
}
f()
`,
		Opts().Skip2ndPass(),
		"boom",
	)
}

// raise(non_error, true) wraps the payload in a fatal error.
func TestRecover_RaiseFatalFlagOnRawPayload(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()
  raise("plain", true)
}
f()
`,
		Opts().Skip2ndPass(),
		"plain",
	)
}

// raise(err, false) demotes a fatal error back to recoverable so recover() catches it; the original error value is
// left unchanged.
func TestRecover_RaiseFalseFlagDemotesToRecoverable(t *testing.T) {
	expectRun(t, `
e := error("boom", true)
f := func() res {
  defer func() {
    r := recover()
    if r != undefined { res = r.is_fatal() }
  }()
  raise(e, false)
}
out = [f(), e.is_fatal()]
`, nil, ARR{false, true})
}

// Script-level error with explicit fatal=false is still recoverable (matches default).
func TestRecover_ScriptExplicitNonFatalIsRecovered(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined { res = e.kind() }
  }()
  raise(error("boom", false))
}
out = f()
`, nil, "user")
}

// `return EXPR` in a function with a named result is sugar for `name = EXPR; return`. Defers can observe and mutate
// the returned value through the named result. Matches Go semantics.
func TestReturnExpr_NamedResult_DeferMutates(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { r = r + 1 }()
			return 41
		}
		out = f()
	`, nil, 42)
}

func TestReturnExpr_NamedResult_DeferOverrides(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { r = "deferred" }()
			return "explicit"
		}
		out = f()
	`, nil, "deferred")
}

func TestReturnExpr_NamedResult_NoDefer_UnaffectedByNamedSlot(t *testing.T) {
	// Without defers, `return EXPR` should still produce EXPR — writing to the named-result slot is a no-op for
	// the visible return value when there are no defers to observe it.
	expectRun(t, `
		f := func() r {
			r = "init"
			return "explicit"
		}
		out = f()
	`, nil, "explicit")
}

func TestReturnExpr_NoNamedResult_DeferIrrelevant(t *testing.T) {
	expectRun(t, `
		f := func() {
			defer func() {}()
			return 7
		}
		out = f()
	`, nil, 7)
}

// `defer obj.method()` calls the method when the surrounding function exits. recover() inside such a method does
// NOT catch a raised error (the method dispatch path doesn't push a Kavun-level deferred-for frame). This codifies
// the current limitation; if/when method-call defers gain recover support, this test should be updated.
func TestDeferMethodCall_DoesNotEnableRecover(t *testing.T) {
	expectError(t, `
		// `+"`recover_helper`"+` is reachable as a method of nothing — we just verify recover() inside a deferred
		// method call (acting on a value) cannot swallow a raised error.
		f := func() {
			arr := [1,2,3]
			defer arr.sort()  // a valid deferred method call; sort() can't recover()
			raise(error("escapes_through_method_defer"))
		}
		f()
	`, nil, "escapes_through_method_defer")
}

// recover() invoked from inside a host builtin running as a defer returns Undefined (the builtin is not a Kavun
// deferred-for frame). Therefore the raised error escapes.
func TestRecover_FromHostBuiltinAsDefer_IsIneffective(t *testing.T) {
	probe := core.NewBuiltinClosureValue(
		"probe_recover",
		func(v core.VM, args []core.Value) (core.Value, error) {
			// Try to recover from inside a deferred builtin — must return Undefined.
			return v.Recover(), nil
		}, 0, false)

	expectError(t, `
f := func() {
  defer probe_recover()
  raise(error("escapes_past_builtin_defer"))
}
f()
`,
		Opts().Symbol("probe_recover", probe).Skip2ndPass(),
		"escapes_past_builtin_defer",
	)
}

// A host builtin that returns a raw (non-*errs.Error) Go error is classified Fatal and bypasses recover(). This
// matches the documented severity policy: any non-*errs.Error defaults to Fatal.
func TestRecover_RawGoErrorFromBuiltin_IsFatal(t *testing.T) {
	rawBuiltin := core.NewBuiltinClosureValue(
		"do_raw",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, fmt.Errorf("plain go error")
		}, 0, false)

	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // cannot catch — error is Fatal
  do_raw()
}
f()
`,
		Opts().Symbol("do_raw", rawBuiltin).Skip2ndPass(),
		"plain go error",
	)
}

// Stress: many defers (1000) all run in LIFO order; the first-registered defer (running last) sees the accumulated
// counter. Exercises allocated args slice and per-defer state cleanup at scale.
func TestDefer_ManyDefers_AllRun(t *testing.T) {
	expectRun(t, `
		f := func() res {
			counter := 0
			defer func() { res = counter }()  // registered FIRST → runs LAST → sees final counter
			for i := 0; i < 1000; i = i + 1 {
				defer func() { counter = counter + 1 }()
			}
		}
		out = f()
	`, nil, 1000)
}

// Common real-world idiom: a defer recovers, decides based on the error kind whether to swallow it, and re-raises
// otherwise. The protection is targeted (e.g. division_by_zero) and unrelated errors must propagate unchanged.

// Selective recover: the error kind matches the protected one and is swallowed.
func TestRecover_SelectiveReraise_MatchingKindSwallowed(t *testing.T) {
	expectRun(t, `
		safe_div := func(a, b) res {
			defer func() {
				e := recover()
				if e != undefined {
					if e.kind() == "division_by_zero" {
						res = -1
					} else {
						raise(e)
					}
				}
			}()
			res = a / b
		}
		out = [safe_div(10, 2), safe_div(10, 0)]
	`, nil, ARR{5, -1})
}

// Selective recover: the recovered error is of a different kind, so it is re-raised and escapes the function. The
// caller observes the original error (kind preserved, message preserved).
func TestRecover_SelectiveReraise_NonMatchingReraised(t *testing.T) {
	expectError(t, `
		safe_div := func(a, b) res {
			defer func() {
				e := recover()
				if e != undefined {
					if e.kind() == "division_by_zero" {
						res = -1
					} else {
						raise(e)  // not the kind we protect against — propagate
					}
				}
			}()
			arr := [1, 2, 3]
			_ = arr[a + b]  // index_out_of_bounds, NOT division_by_zero
		}
		safe_div(99, 0)
	`, nil, "index_out_of_bounds")
}

// The re-raised error preserves its original kind so an outer defer can still classify it correctly.
func TestRecover_SelectiveReraise_KindPreservedForOuterRecover(t *testing.T) {
	expectRun(t, `
		outer_kind := ""
		safe_div := func(a, b) res {
			defer func() {
				e := recover()
				if e != undefined {
					if e.kind() == "division_by_zero" {
						res = -1
					} else {
						raise(e)
					}
				}
			}()
			arr := [1, 2, 3]
			_ = arr[10]  // index_out_of_bounds
		}
		g := func() {
			defer func() {
				e := recover()
				if e != undefined { outer_kind = e.kind() }
			}()
			safe_div(1, 1)
		}
		g()
		out = outer_kind
	`, nil, "index_out_of_bounds")
}

// User-raised errors aren't filtered by kind here ("user"): they too can be selectively re-raised based on payload.
func TestRecover_SelectiveReraise_UserErrorByPayload(t *testing.T) {
	// Code "expected" is swallowed; code "fatal" is re-raised with its original payload intact.
	expectError(t, `
		guarded := func(payload) res {
			defer func() {
				e := recover()
				if e != undefined {
					v := e.value()
					if v.code == "expected" {
						res = "handled"
					} else {
						raise(e)
					}
				}
			}()
			raise(error(payload))
		}
		_ = guarded({code: "expected"})           // swallowed
		_ = guarded({code: "unexpected_boom"})    // re-raised
	`, nil, "unexpected_boom")
}

// Re-raising the recovered error from inside a defer is itself catchable by an *earlier-registered* defer
// (which runs later). This mirrors the LIFO interaction already tested for fresh raises.
func TestRecover_SelectiveReraise_CaughtByEarlierDefer(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				// outermost — runs last; catches the re-raised error.
				e := recover()
				if e != undefined && e.kind() == "division_by_zero" {
					res = "outer_caught"
				}
			}()
			defer func() {
				// inner — runs first; recovers, inspects, re-raises because it only handles "not_iterable".
				e := recover()
				if e != undefined {
					if e.kind() == "not_iterable" {
						res = "inner_swallowed"
					} else {
						raise(e)
					}
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, "outer_caught")
}

// recover() called from a nested *non-deferred* helper function returns undefined and the error propagates.
// This is the contrapositive of TestRecover_OnlyDirectlyInDeferred phrased in terms of the new Recover() guard.
func TestRecover_NestedHelper_ReturnsUndefined(t *testing.T) {
	expectError(t, `
		helper := func() { _ = recover() }
		f := func() {
			defer func() { helper() }()
			raise(error("nested_helper_cannot_recover"))
		}
		f()
	`, nil, "nested_helper_cannot_recover")
}

func TestBuiltinIsPredicates(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want bool
	}{
		// is_string
		{"is_string/string", `is_string("a")`, true},
		{"is_string/runes", `is_string(runes("a"))`, false},
		{"is_string/int", `is_string(1)`, false},

		// is_runes
		{"is_runes/runes", `is_runes(runes("a"))`, true},
		{"is_runes/string", `is_runes("a")`, false},

		// is_int
		{"is_int/int", `is_int(1)`, true},
		{"is_int/float", `is_int(1.0)`, false},

		// is_float
		{"is_float/float", `is_float(1.0)`, true},
		{"is_float/int", `is_float(1)`, false},

		// is_decimal
		{"is_decimal/decimal", `is_decimal(decimal("1.5"))`, true},
		{"is_decimal/float", `is_decimal(1.5)`, false},

		// is_bool
		{"is_bool/true", `is_bool(true)`, true},
		{"is_bool/int", `is_bool(0)`, false},

		// is_byte
		{"is_byte/byte", `is_byte(byte(0))`, true},
		{"is_byte/int", `is_byte(0)`, false},

		// is_rune
		{"is_rune/rune", `is_rune('a')`, true},
		{"is_rune/int", `is_rune(97)`, false},

		// is_bytes
		{"is_bytes/bytes", `is_bytes(bytes("a"))`, true},
		{"is_bytes/string", `is_bytes("a")`, false},

		// is_array
		{"is_array/array", `is_array([])`, true},
		{"is_array/dict", `is_array({})`, false},

		// is_record
		{"is_record/record", `is_record({})`, true},
		{"is_record/dict", `is_record(dict({}))`, false},

		// is_dict
		{"is_dict/dict", `is_dict(dict({}))`, true},
		{"is_dict/record", `is_dict({})`, false},

		// is_range
		{"is_range/range", `is_range(range(0, 5, 1))`, true},
		{"is_range/array", `is_range([])`, false},

		// is_immutable
		{"is_immutable/immutable", `is_immutable(immutable([1, 2]))`, true},
		{"is_immutable/mutable", `is_immutable([1, 2])`, false},
		{"is_immutable/string", `is_immutable("x")`, true},
		{"is_immutable/int", `is_immutable(1)`, true},

		// is_time
		{"is_time/time", `is_time(time())`, true},
		{"is_time/int", `is_time(1)`, false},

		// is_error
		{"is_error/error", `is_error(error("oops"))`, true},
		{"is_error/string", `is_error("x")`, false},

		// is_undefined
		{"is_undefined/undef", `is_undefined(undefined)`, true},
		{"is_undefined/zero", `is_undefined(0)`, false},

		// is_function
		{"is_function/lambda", `is_function(func(){})`, true},
		{"is_function/builtin", `is_function(len)`, true},
		{"is_function/int", `is_function(1)`, false},

		// is_callable
		{"is_callable/lambda", `is_callable(func(){})`, true},
		{"is_callable/builtin", `is_callable(len)`, true},
		{"is_callable/int", `is_callable(1)`, false},

		// is_iterable
		{"is_iterable/array", `is_iterable([])`, true},
		{"is_iterable/string", `is_iterable("a")`, true},
		{"is_iterable/range", `is_iterable(range(0, 1, 1))`, true},
		{"is_iterable/int", `is_iterable(1)`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectRun(t, "out = "+c.expr, nil, c.want)
		})
	}
}

func TestBuiltinIsPredicates_WrongArity(t *testing.T) {
	for _, name := range []string{
		"is_string", "is_runes", "is_int", "is_float", "is_decimal",
		"is_bool", "is_byte", "is_rune", "is_bytes", "is_array",
		"is_record", "is_dict", "is_range", "is_immutable", "is_time",
		"is_error", "is_undefined", "is_function", "is_callable", "is_iterable",
	} {
		t.Run(name, func(t *testing.T) {
			expectError(t, name+"()", nil, fmt.Sprintf("wrong_num_arguments: (%s) expected 1 argument(s), got 0", name))
		})
	}
}

func TestBuiltinTypeName(t *testing.T) {
	expectRun(t, `out = type_name(1)`, nil, "int")
	expectRun(t, `out = type_name(1.0)`, nil, "float")
	expectRun(t, `out = type_name("x")`, nil, "string")
	expectRun(t, `out = type_name([])`, nil, "array")
	expectRun(t, `out = type_name({})`, nil, "record")
	expectRun(t, `out = type_name(dict({}))`, nil, "dict")
	expectRun(t, `out = type_name(undefined)`, nil, "undefined")
	expectRun(t, `out = type_name(error("x"))`, nil, "error")
	expectRun(t, `out = type_name(func(){})`, nil, "<compiled-function/0>")
	expectRun(t, `out = type_name(len)`, nil, "<builtin-function:len/1>")
	expectError(t, `type_name()`, nil, "wrong_num_arguments: (type_name) expected 1 argument(s), got 0")
}

func TestSpread_EmptyArray_OnVariadic(t *testing.T) {
	expectRun(t, `f := func(...a) { return a }; out = f([]...)`, nil, ARR{})
	expectRun(t, `f := func(a, ...b) { return [a, b] }; out = f(1, []...)`, nil, ARR{1, ARR{}})
}

func TestSpread_EmptyArray_OnFixedArity(t *testing.T) {
	expectRun(t, `f := func() { return 42 }; out = f([]...)`, nil, 42)
	expectError(t, `f := func(a) { return a }; f([]...)`, nil, "wrong_num_arguments")
}

func TestSpread_NonArray(t *testing.T) {
	expectError(t, `f := func(a) { return a }; r := {a:1}; f(r...)`, nil, "invalid_argument_type: (...) argument spread expects type array, got record")
	expectError(t, `f := func(a) { return a }; s := "abc"; f(s...)`, nil, "invalid_argument_type: (...) argument spread expects type array, got string")
	expectError(t, `f := func(a) { return a }; n := 1; f(n...)`, nil, "invalid_argument_type: (...) argument spread expects type array, got int")
}

func TestSpread_MethodCall_EmptyArray_WrongArgsRaised(t *testing.T) {
	// for_each requires exactly 1 fn argument. An empty spread degrades to zero args.
	expectError(t, `[1,2].for_each([]...)`, nil, "wrong_num_arguments: (for_each)")
}

func TestSpread_MethodCall_NonArray(t *testing.T) {
	expectError(t, `[1,2].for_each({a:1}...)`, nil, "invalid_argument_type: (...) argument spread expects type array, got record")
}

// Spread expansion of a large array must raise a recoverable stack_overflow
// error, NOT a Go runtime panic. The compile-time MaxStack analyzer cannot
// model the data-driven growth of `f(arr...)`, so the VM bounds-checks the
// spread destination before expanding. (DefaultStackSize == 2048.)
func TestSpread_LargeArray_OpCall_StackOverflow(t *testing.T) {
	src := `
		f := func(...args) { return len(args) }
		big := []
		for i := 0; i < 5000; i = i + 1 { big = append(big, i) }
		out = f(big...)
	`
	expectError(t, src, nil, "stack_overflow")
}

func TestSpread_LargeArray_OpMethodCall_StackOverflow(t *testing.T) {
	// Stress OpMethodCall's spread path. `d.keys` is a no-arg method, but the
	// spread expansion happens before arg-count validation, so a huge array
	// still trips the bounds check.
	src := `
		big := []
		for i := 0; i < 5000; i = i + 1 { big = append(big, i) }
		d := {}
		out = len(d.keys(big...))
	`
	expectError(t, src, nil, "stack_overflow")
}

// Sanity: a reasonable spread (well under DefaultStackSize) works normally.
// Pins down the boundary between rejected and accepted behavior.
func TestSpread_SmallArray_OK(t *testing.T) {
	src := `
		f := func(...args) { return len(args) }
		big := []
		for i := 0; i < 500; i = i + 1 { big = append(big, i) }
		out = f(big...)
	`
	expectRun(t, src, nil, 500)
}

func TestSplice_HugeDeleteCountClamps(t *testing.T) {
	// Regression: large positive count must be clamped, not overflow startIdx+delCount.
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := splice(a, 2, 9223372036854775807)
		out = [a, d]
	`, nil, ARR{ARR{1, 2}, ARR{3, 4, 5}})
}

func TestSplice_HugeDeleteCountWithInsertClamps(t *testing.T) {
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := splice(a, 1, 9223372036854775807, "x", "y")
		out = [a, d]
	`, nil, ARR{ARR{1, "x", "y"}, ARR{2, 3, 4, 5}})
}

func TestSplice_NegativeStart(t *testing.T) {
	expectError(t, `splice([1,2,3], -1)`, nil, "index_out_of_bounds: (splice, start index)")
}

func TestSplice_StartBeyondLen(t *testing.T) {
	expectError(t, `splice([1,2,3], 4)`, nil, "index_out_of_bounds: (splice, start index)")
}

func TestSplice_NegativeCount_Recoverable(t *testing.T) {
	// Bug fix: negative-count error is now Recoverable so deferred recover() can catch it.
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			splice([1,2,3], 0, -1)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestSplice_OnConstArray_Errors(t *testing.T) {
	expectError(t, `splice(immutable([1,2,3]), 0)`, nil,
		"invalid_argument_type: (splice) argument first expects type mutable array")
}

func TestRange_StepZero_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			range(0, 5, 0)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestRange_NegativeStep_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			range(0, 5, -1)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestRange_WrongArity(t *testing.T) {
	expectError(t, `range()`, nil, "wrong_num_arguments: (range) expected 2 or 3")
	expectError(t, `range(1)`, nil, "wrong_num_arguments: (range) expected 2 or 3")
	expectError(t, `range(1,2,3,4)`, nil, "wrong_num_arguments: (range) expected 2 or 3")
}

func TestRange_NonIntArgs(t *testing.T) {
	expectError(t, `range("a", 1, 1)`, nil, "invalid_argument_type: (range) argument start expects type int")
	expectError(t, `range(0, "b", 1)`, nil, "invalid_argument_type: (range) argument stop expects type int")
	expectError(t, `range(0, 1, "c")`, nil, "invalid_argument_type: (range) argument step expects type int")
}

func TestConstructorFallback_Defaults(t *testing.T) {
	// Use values that are NOT convertible to the target type, so the fallback kicks in.
	expectRun(t, `out = int("nope", 42)`, nil, 42)
	expectRun(t, `out = float("nope", 1.5)`, nil, 1.5)
	expectRun(t, `out = string(len, "alt")`, nil, "alt")
}

func TestConstructorFallback_NoFallback_ReturnsUndefined(t *testing.T) {
	expectRun(t, `out = is_undefined(int("nope"))`, nil, true)
	expectRun(t, `out = is_undefined(float("nope"))`, nil, true)
}

func TestConstructorWrongArity(t *testing.T) {
	expectError(t, `int(1, 2, 3)`, nil, "wrong_num_arguments: (int) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `float(1, 2, 3)`, nil, "wrong_num_arguments: (float) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `bool(1, 2, 3)`, nil, "wrong_num_arguments: (bool) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `byte(1, 2, 3)`, nil, "wrong_num_arguments: (byte) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `rune(1, 2, 3)`, nil, "wrong_num_arguments: (rune) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `string(1, 2, 3)`, nil, "wrong_num_arguments: (string) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `runes(1, 2, 3)`, nil, "wrong_num_arguments: (runes) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `bytes(1, 2, 3)`, nil, "wrong_num_arguments: (bytes) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `decimal(1, 2, 3)`, nil, "wrong_num_arguments: (decimal) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `time(1, 2, 3)`, nil, "wrong_num_arguments: (time) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `dict(1, 2, 3)`, nil, "wrong_num_arguments: (dict) expected 0, 1 or 2 argument(s), got 3")
}

func TestBuiltinDict_FromInvalidType(t *testing.T) {
	expectError(t, `dict(123)`, nil, "invalid_argument_type: (dict) argument first expects type dict or record")
}

func TestError_FatalFlag(t *testing.T) {
	// error(payload, true) creates a fatal error which bypasses recover.
	expectError(t, `
		f := func() {
			defer func() { recover() }()
			raise(error("boom", true))
		}
		f()
	`, nil, "boom")
}

func TestError_RecoverableFlag(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			raise(error("boom", false))
		}
		out = f()
	`, nil, "rescued")
}

func TestError_WrongFlagType(t *testing.T) {
	// A builtin function value has no AsBool conversion -> triggers the type check.
	expectError(t, `error("x", len)`, nil,
		"invalid_argument_type: (error) argument second expects type bool")
}

func TestError_WrongArity(t *testing.T) {
	expectError(t, `error()`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 0")
	expectError(t, `error("a", true, "extra")`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 3")
}

func TestRaise_PayloadGetsWrapped(t *testing.T) {
	// raise of non-error wraps it.
	expectRun(t, `
		f := func() r {
			defer func() {
				e := recover()
				if is_error(e) { r = "wrapped" }
			}()
			raise("plain")
		}
		out = f()
	`, nil, "wrapped")
}

func TestRaise_FatalFlag_BypassesRecover(t *testing.T) {
	expectError(t, `
		f := func() {
			defer func() { recover() }()
			raise("boom", true)
		}
		f()
	`, nil, "boom")
}

func TestRaise_DemoteFatalFlagToRecoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			raise(error("boom", true), false) // demote
		}
		out = f()
	`, nil, "rescued")
}

func TestRaise_WrongArity(t *testing.T) {
	expectError(t, `raise()`, nil, "wrong_num_arguments: (raise) expected 1 or 2 argument(s), got 0")
	expectError(t, `raise("x", true, "extra")`, nil, "wrong_num_arguments: (raise) expected 1 or 2 argument(s), got 3")
}

func TestRaise_WrongFlagType(t *testing.T) {
	expectError(t, `raise("x", len)`, nil, "invalid_argument_type: (raise) argument second expects type bool")
}

func TestRecover_WrongArity(t *testing.T) {
	expectError(t, `func() { defer func() { recover(1) }(); raise("x") }()`, nil, "wrong_num_arguments: (recover) expected 0 argument(s), got 1")
}

func TestDefer_DeepRecursionWithDefers(t *testing.T) {
	// Each call registers a defer; verifies that the deferred-call slice is correctly
	// reset on each frame across many levels and that recover-eligible frames don't
	// leak in-flight errors between calls.
	expectRun(t, `
		log := []
		f := func() {}
		walker := 0
		walker = func(n) {
			defer f()
			if n > 0 {
				walker(n-1)
			}
			log = append(log, n)
		}
		walker(20)
		out = len(log)
	`, nil, 21)
}

func TestDefer_LaterDeferRunsAfterEarlierRaisedAndRecovered(t *testing.T) {
	// First defer (LIFO last) raises. Earlier defer recovers it; the function returns normally.
	expectRun(t, `
		log := []
		f := func() r {
			defer func() {
				log = append(log, "defer1")
				e := recover()
				if e != undefined { log = append(log, "rescued") }
			}()
			defer func() {
				log = append(log, "defer2")
				raise("from-defer2")
			}()
			r = "ok"
		}
		_ = f()
		out = log
	`, nil, ARR{"defer2", "defer1", "rescued"})
}

func TestDefer_NestedFunctionCallRecoverFails(t *testing.T) {
	// recover() called from a helper INSIDE a defer must return undefined (Go parity).
	expectRun(t, `
		out = "untouched"
		f := func() {
			defer func() {
				helper := func() { return recover() }
				e := helper()
				if e == undefined { out = "no_recover_through_helper" }
			}()
			raise("err")
		}
		// f re-raises since helper.recover() returned undefined.
		// Wrap to swallow.
		g := func() {
			defer func() { recover() }()
			f()
		}
		g()
	`, nil, "no_recover_through_helper")
}

func TestDefer_VariadicDeferredFunction(t *testing.T) {
	expectRun(t, `
		log := []
		f := func(...args) { log = append(log, args) }
		g := func() {
			defer f(1, 2, 3)
		}
		g()
		out = log[0]
	`, nil, ARR{1, 2, 3})
}

func TestTailCall_DeepRecursionDoesNotOverflow(t *testing.T) {
	// 100k iterations: only TCO keeps this within DefaultMaxFrames.
	expectRun(t, `
		f := func(n) {
			if n == 0 { return "done" }
			return f(n-1)
		}
		out = f(100000)
	`, nil, "done")
}

func TestTailCall_DisabledWhenDefersPresent(t *testing.T) {
	// With a defer registered, TCO must be skipped — otherwise the defer slice
	// would leak across the recursive call, doubling-firing or losing entries.
	expectRun(t, `
		log := []
		f := 0
		f = func(n) {
			defer func() { log = append(log, n) }()
			if n == 0 { return }
			f(n-1)
		}
		f(3)
		out = log
	`, nil, ARR{0, 1, 2, 3})
}

func TestClosure_DeferMutatesCapturedVariable(t *testing.T) {
	expectRun(t, `
		x := 1
		f := func() {
			defer func() { x = 99 }()
		}
		f()
		out = x
	`, nil, 99)
}

func TestClosure_NamedResultViaClosure(t *testing.T) {
	// Defer mutates named result through closure capture.
	expectRun(t, `
		f := func() r {
			r = 10
			defer func() { r = r * 2 }()
			return
		}
		out = f()
	`, nil, 20)
}

func TestHostCallback_CallScriptFunction(t *testing.T) {
	// A host-registered builtin that invokes a script function via VM.Call.
	caller := core.NewBuiltinClosureValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			if len(args) != 2 {
				return core.Undefined, fmt.Errorf("invoke expects (fn, arg)")
			}
			fnVal := args[0]
			if fnVal.Type != value.CompiledFunction {
				return core.Undefined, fmt.Errorf("invoke: arg 1 not a function")
			}
			return v.Call(fnVal, []core.Value{args[1]})
		}, 2, false)

	expectRun(t, `f := func(x) { return x * 3 }; out = invoke(f, 7)`, Opts().Symbol("invoke", caller).Skip2ndPass(), 21)
}

func TestHostCallback_PropagatesRaisedError(t *testing.T) {
	// Errors raised by the script callback must bubble back through VM.Call to the host.
	caller := core.NewBuiltinClosureValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call(fnVal, nil)
		}, 1, false)

	expectError(t, `f := func() { raise("script-side") }; invoke(f)`, Opts().Symbol("invoke", caller).Skip2ndPass(), "script-side")
}

func TestHostCallback_RecoveredByOuterScript(t *testing.T) {
	// If the host-invoked script function defers a recover, the error must be
	// caught at the trampoline boundary and returned cleanly to the host.
	caller := core.NewBuiltinClosureValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call(fnVal, nil)
		}, 1, false)

	expectRun(t, `
		f := func() r {
			defer func() {
				e := recover()
				if e != undefined { r = "rescued" }
			}()
			raise("oops")
		}
		out = invoke(f)
	`, Opts().Symbol("invoke", caller).Skip2ndPass(), "rescued")
}

func TestHostCallback_VarargsAndArity(t *testing.T) {
	caller := core.NewBuiltinClosureValue("invoke3",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call(fnVal, []core.Value{core.IntValue(1), core.IntValue(2), core.IntValue(3)})
		}, 1, false)

	// Variadic script function via host VM.Call.
	expectRun(t, `
		f := func(...xs) {
			s := 0
			for _, x in xs { s += x }
			return s
		}
		out = invoke3(f)
	`, Opts().Symbol("invoke3", caller).Skip2ndPass(), 6)

	// Wrong arity from host-side.
	wrong := core.NewBuiltinClosureValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call(fnVal, nil)
		}, 1, false)
	expectError(t, `f := func(a) { return a }; invoke(f)`, Opts().Symbol("invoke", wrong).Skip2ndPass(), "wrong_num_arguments: (call) expected 1 argument(s), got 0")
}

func TestStackOverflow_MutualRecursion(t *testing.T) {
	expectError(t, `
		f := 0
		g := 0
		f = func(n) { return g(n+1) }
		g = func(n) { return f(n+1) }
		f(0)
	`, nil, "stack_overflow")
}

func TestStackOverflow_HostCallback_RespectsFrameLimit(t *testing.T) {
	// Build a small VM with very few frames, then invoke a host-callback that
	// wants to call back into the VM. Eventually exhaust frames.
	machine := vm.NewVM(8, 1024) // tiny frame stack

	var caller core.Value
	callerFn := func(v core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, fmt.Errorf("invoke needs 1 arg")
		}
		return v.Call(args[0], []core.Value{args[0]})
	}
	caller = core.NewBuiltinClosureValue("invoke", callerFn, 1, false)

	s := kavun.NewScript([]byte(`f := func(self) { return invoke(self) }; out = invoke(f)`), "out", "invoke")
	c, err := s.Compile()
	require.NoError(t, err)
	c.Set("out", core.Undefined)
	c.Set("invoke", caller)
	err = c.Run(machine)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "stack_overflow"), "expected stack_overflow, got %v", err)
}

func TestIterator_OnNonIterable(t *testing.T) {
	expectError(t, `for x in 1 { _ = x }`, nil, "not_iterable")
	expectError(t, `for k, v in true { _ = k; _ = v }`, nil, "not_iterable")
}

func TestFormatDyn_BadSpec_Recoverable(t *testing.T) {
	// f"{x:{spec}}" with an invalid dynamic spec must produce a recoverable error.
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			x := 42; spec := "@"
			_ = f"{x:{spec}}"
		}
		out = f()
	`, nil, "rescued")
}

func TestFormatDyn_NonStringSpec(t *testing.T) {
	// The dynamic-spec inner expression is always coerced to a string by the compiler
	// (via OpFormat with empty spec), so this guard is mostly defensive — verify that
	// even purely non-string-looking values produce a valid (or recoverable) result
	// rather than panicking. Numeric specs parse as width.
	expectRun(t, `x := 1; spec := 5; out = f"{x:{spec}}"`, nil, "    1")
}

func TestBuiltinFormat_TemplateModeMismatch(t *testing.T) {
	expectError(t, `format("{a}", [1])`, nil, "invalid_argument_type: (format) argument args expects type dict or record, got array")
	expectError(t, `format("{0}", {a:1})`, nil, "invalid_argument_type: (format) argument args expects type array, got record")
}

func TestBuiltinFormat_MissingKey(t *testing.T) {
	expectError(t, `format("{missing}", {a:1})`, nil, "missing key")
}

func TestBuiltinFormat_IndexOutOfRange(t *testing.T) {
	expectError(t, `format("{5}", [1])`, nil, "out of range")
}

func TestBuiltinFormat_BytesAsTemplate(t *testing.T) {
	expectRun(t, `out = format(bytes("hi {0}!"), ["world"])`, nil, "hi world!")
}

// Regression: format() errors used to be NewInternalError (fatal). They are now
// recoverable so deferred recover() can catch them.
func TestBuiltinFormat_ErrorsAreRecoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			_ = format("{missing}", {a:1})
		}
		out = f()
	`, nil, "rescued")
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			_ = format("{0}", [])
		}
		out = f()
	`, nil, "rescued")
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			_ = format("{unterminated", {})
		}
		out = f()
	`, nil, "rescued")
}

func TestArrayChunk_NonPositiveSize_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			_ = [1,2,3].chunk(0)
		}
		out = f()
	`, nil, "rescued")
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			_ = [1,2,3].chunk(-5)
		}
		out = f()
	`, nil, "rescued")
}

func TestBuiltinFormat_RunesAsTemplate(t *testing.T) {
	expectRun(t, `out = format(runes("hi {0}!"), ["world"])`, nil, "hi world!")
}

func TestBuiltinFormat_NonStringTemplate(t *testing.T) {
	expectError(t, `format(123, [])`, nil,
		"invalid_argument_type: (format) argument template expects type string")
}

func TestBuiltinFormat_WrongArity(t *testing.T) {
	expectError(t, `format("x")`, nil, "wrong_num_arguments: (format) expected 2")
}

func TestRecordLiteral_StringKey_OK(t *testing.T) {
	expectRun(t, `out = {"a": 1, "b": 2}`, nil, MAP{"a": 1, "b": 2})
}

func TestArith_DivisionByZero_Int(t *testing.T) {
	expectError(t, `1 / 0`, nil, "division_by_zero")
	expectError(t, `1 % 0`, nil, "division_by_zero")
}

func TestArith_DivisionByZero_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { if recover() != undefined { r = "rescued" } }()
			_ = 1 / 0
			return "no"
		}
		out = f()
	`, nil, "rescued")
}

func TestArith_NegateMinInt_Wraps(t *testing.T) {
	// -MinInt64 wraps to MinInt64 (two's complement); document the behavior.
	expectRun(t, `
		min := -9223372036854775807 - 1
		out = -min == min
	`, nil, true)
}

func TestArith_BitwiseComplement_Int(t *testing.T) {
	expectRun(t, `out = ^0`, nil, -1)
	expectRun(t, `out = ^(-1)`, nil, 0)
}

func TestNotCallable(t *testing.T) {
	expectError(t, `1()`, nil, "not_callable: type int is not callable")
	expectError(t, `({})()`, nil, "not_callable")
	expectError(t, `"x"()`, nil, "not_callable")
}

func TestSelectorAssign_GlobalRecord(t *testing.T) {
	expectRun(t, `
		g := {a: {b: 1}}
		g.a.b = 99
		out = g.a.b
	`, nil, 99)
}

func TestSelectorAssign_LocalRecord(t *testing.T) {
	expectRun(t, `
		f := func() {
			x := {a: {b: 1}}
			x.a.b = 99
			return x.a.b
		}
		out = f()
	`, nil, 99)
}

func TestSelectorAssign_FreeVar(t *testing.T) {
	expectRun(t, `
		f := func() {
			x := {a: {b: 1}}
			g := func() { x.a.b = 99 }
			g()
			return x.a.b
		}
		out = f()
	`, nil, 99)
}

func TestSpread_MethodCall_EmptyArray(t *testing.T) {
	// `arr.method(args...)` where args is an empty array — combined with a method
	// that accepts variable arity. dict has a `keys()` method that takes 0 args.
	expectRun(t, `
		d := dict({a:1, b:2})
		out = len(d.keys([]...))
	`, nil, 2)
}

func TestHostErrorBoundary_ErrorsIsWorks(t *testing.T) {
	s := kavun.NewScript([]byte("1 / 0"))
	c, err := s.Compile()
	require.NoError(t, err)

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	err = c.Run(machine)
	require.Error(t, err)
	require.True(t, errors.Is(err, errs.ErrDivisionByZero), "expected errors.Is(err, ErrDivisionByZero), got: %v", err)
}

func TestVM_Abort_StopsExecution(t *testing.T) {
	s := kavun.NewScript([]byte("for true { _ = 1 }"))
	c, err := s.Compile()
	require.NoError(t, err)

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		runErr = c.Run(machine)
	}()
	time.Sleep(20 * time.Millisecond)
	machine.Abort()
	wg.Wait()
	// VM stopped cleanly via Abort: no error propagated.
	require.NoError(t, runErr)
}

func TestVM_Clear_ZerosOutSlots(t *testing.T) {
	s := kavun.NewScript([]byte(`out = "ok"`), "out")
	c, err := s.Compile()
	require.NoError(t, err)

	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	require.NoError(t, c.Run(machine))
	// Should not panic, should not leak references.
	machine.Clear()
	require.True(t, machine.IsStackEmpty())
}

func TestVM_ReuseAfterAbort(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	// 1: abort an infinite loop
	s1 := kavun.NewScript([]byte(`for true { _ = 1 }`))
	c1, err := s1.Compile()
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = c1.Run(machine)
	}()
	time.Sleep(10 * time.Millisecond)
	machine.Abort()
	wg.Wait()

	// 2: reuse same VM for a fresh program — must not be poisoned.
	s2 := kavun.NewScript([]byte(`out = 7`), "out")
	c2, err := s2.Compile()
	require.NoError(t, err)
	require.NoError(t, c2.Run(machine))
	require.Equal(t, core.IntValue(7), c2.Get("out"))
}

func TestRunContext_CancelMidExecution(t *testing.T) {
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	s := kavun.NewScript([]byte(`for true {}`))
	c, err := s.Compile()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	err = c.RunContext(ctx, machine)
	require.Equal(t, context.Canceled, err)
}
