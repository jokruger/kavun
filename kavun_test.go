package kavun_test

import (
	"errors"
	"fmt"
	"maps"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/vm"
)

const testOut = "out"

type IARR []any
type IMAP map[string]any
type MAP = map[string]any
type ARR = []any

func formatGlobals(a *core.Arena, globals []core.Value) (formatted []string) {
	for idx, global := range globals {
		if global.Type == core.VT_UNDEFINED {
			return
		}
		formatted = append(formatted, fmt.Sprintf("[% 3d] %s (%s|%v)", idx, global.String(a), global.TypeName(a), global))
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

func errorObject(a *core.Arena, v any) core.Value {
	if s, ok := v.(string); ok {
		sv, err := a.NewStringValue(s)
		if err != nil {
			panic(fmt.Errorf("failed to create string value: %w", err))
		}
		nv, err := a.NewErrorValue(sv, core.KindUser, false)
		if err != nil {
			panic(fmt.Errorf("failed to create error value: %w", err))
		}
		return nv
	}
	nv, err := a.NewErrorValue(toObject(a, v), core.KindUser, false)
	if err != nil {
		panic(fmt.Errorf("failed to create error value: %w", err))
	}
	return nv
}

func toObject(a *core.Arena, v any) core.Value {
	switch v := v.(type) {
	case core.Value:
		return v
	case nil:
		return core.Undefined
	case string:
		return a.MustNewStringValue(v)
	case int64:
		return core.IntValue(v)
	case int:
		return core.IntValue(int64(v))
	case bool:
		return core.BoolValue(v)
	case rune:
		return core.RuneValue(v)
	case byte:
		return core.ByteValue(v)
	case float64:
		return core.FloatValue(v)
	case dec128.Dec128:
		return a.MustNewDecimalValue(v)
	case []byte:
		return a.MustNewBytesValue(v, false)
	case []rune:
		return a.MustNewRunesValue(v, false)
	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			t := toObject(a, v)
			t.Pin(a)
			objs[k] = t
		}
		return a.MustNewRecordValue(objs, false)
	case ARR:
		var objs []core.Value
		for _, e := range v {
			t := toObject(a, e)
			t.Pin(a)
			objs = append(objs, t)
		}
		return a.MustNewArrayValue(objs, false)
	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			t := toObject(a, v)
			t.Pin(a)
			objs[k] = t
		}
		return a.MustNewRecordValue(objs, true)
	case IARR:
		var objs []core.Value
		for _, e := range v {
			t := toObject(a, e)
			t.Pin(a)
			objs = append(objs, t)
		}
		return a.MustNewArrayValue(objs, true)
	}
	panic(fmt.Errorf("unknown type: %T", v))
}

func objectZeroCopy(a *core.Arena, o core.Value) core.Value {
	switch o.Type {
	case core.VT_UNDEFINED:
		return core.Undefined

	case core.VT_BOOL:
		return core.False

	case core.VT_INT:
		return core.IntValue(0)

	case core.VT_BYTE:
		return core.ByteValue(0)

	case core.VT_FLOAT:
		return core.FloatValue(0)

	case core.VT_DECIMAL:
		return a.MustNewDecimalValue(dec128.Zero)

	case core.VT_RUNE:
		return core.RuneValue(0)

	case core.VT_STRING:
		return a.MustNewStringValue("")

	case core.VT_RUNES:
		return a.MustNewRunesValue([]rune(""), false)

	case core.VT_ARRAY:
		return a.MustNewArrayValue(nil, o.Immutable)

	case core.VT_RECORD:
		return a.MustNewRecordValue(nil, o.Immutable)

	case core.VT_DICT:
		return a.MustNewDictValue(nil, o.Immutable)

	case core.VT_ERROR:
		return a.MustNewErrorValue(core.Undefined, core.KindUser, false)

	case core.VT_BYTES:
		return a.MustNewBytesValue(nil, false)

	default:
		panic(fmt.Errorf("unknown value kind: %d", o.Type))
	}
}

func traceCompileRun(
	rta *core.Arena,
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
		stdlib.InitModule(name, core.BI_MOD_USER_DEFINED+uint8(idx), mod.bmi, mod.cs, mod.fns)
		idx++
	}
	defer func() {
		for name := range customBuiltinModules {
			stdlib.RemoveModule(name)
		}
	}()

	tr := &vmTracer{}
	c := compiler.NewCompiler(nil, file.InputFile, symTable, nil, customModules, tr)
	err = c.Compile(file)
	trace = append(trace, fmt.Sprintf("\n[Compiler Trace]\n\n%s", strings.Join(tr.Out, "")))
	if err != nil {
		return
	}

	bytecode := c.Bytecode()
	trace = append(trace, fmt.Sprintf("\n[Compiled Constants]\n\n%s", strings.Join(bytecode.MustFormatStatics(), "\n")))
	trace = append(trace, fmt.Sprintf("\n[Compiled Instructions]\n\n%s\n", strings.Join(bytecode.MustFormatInstructions(), "\n")))

	machine.Reset(rta, bytecode, globals)
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
		trace = append(trace, fmt.Sprintf("\n[Globals]\n\n%s", strings.Join(formatGlobals(rta, globals), "\n")))
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
	bmi stdlib.BuiltinModuleInitializer
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

func expectErrorAs(t *testing.T, rta *core.Arena, input string, opts *testOpts, expected any) {
	if opts == nil {
		opts = Opts()
	}

	// parse
	program := parse(t, input)
	if program == nil {
		return
	}

	// compiler/VM
	_, trace, err := traceCompileRun(rta, program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, errors.As(err, expected), "expected error as: %v, got: %v\n%s", expected, err, strings.Join(trace, "\n"))
}

func expectErrorIs(t *testing.T, rta *core.Arena, input string, opts *testOpts, expected error) {
	if opts == nil {
		opts = Opts()
	}

	// parse
	program := parse(t, input)
	if program == nil {
		return
	}

	// compiler/VM
	_, trace, err := traceCompileRun(rta, program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, errors.Is(err, expected), "expected error is: %s, got: %s\n%s", expected.Error(), err.Error(), strings.Join(trace, "\n"))
}

func expectError(t *testing.T, rta *core.Arena, input string, opts *testOpts, expected string) {
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
	_, trace, err := traceCompileRun(rta, program, opts.symbols, opts.customModules, opts.customBuiltinModules)
	require.Error(t, err, "\n"+strings.Join(trace, "\n"))
	require.True(t, strings.Contains(err.Error(), expected), "expected error string: %s, got: %s\n%s", expected, err.Error(), strings.Join(trace, "\n"))
}

func expectRun(t *testing.T, rta *core.Arena, input string, opts *testOpts, expected any) {
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
		res, trace, err := traceCompileRun(rta, file, symbols, opts.customModules, opts.customBuiltinModules)
		require.NoError(t, err, "\n"+strings.Join(trace, "\n"))
		a := res[testOut]
		e := toObject(rta, expected)
		require.Equal(t, rta, e, a, "\n"+strings.Join(trace, "\n"))
	}

	// second pass: run the code as import module
	if !opts.skip2ndPass {
		file := parse(t, `out = import("__code__")`)
		if file == nil {
			return
		}

		symbols[testOut] = core.Undefined //objectZeroCopy(rta, expectedObj)
		modules := maps.Clone(opts.customModules)
		modules["__code__"] = []byte(fmt.Sprintf("out := undefined; %s; export out", input))
		res, trace, err := traceCompileRun(rta, file, symbols, modules, opts.customBuiltinModules)
		require.NoError(t, err, "\n"+strings.Join(trace, "\n"))
		a := res[testOut]
		e := toObject(rta, expected)
		require.Equal(t, rta, e, a, "\n"+strings.Join(trace, "\n"))
	}
}

func TestUndefined(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = undefined`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined.a`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined[1]`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined.a.b`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined[1][2]`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined ? 1 : 2`, nil, 2)
	expectRun(t, rta, `out = undefined == undefined`, nil, true)
	expectRun(t, rta, `out = undefined == 1`, nil, false)
	expectRun(t, rta, `out = 1 == undefined`, nil, false)
	expectRun(t, rta, `out = undefined == float([])`, nil, true)
	expectRun(t, rta, `out = float([]) == undefined`, nil, true)
	expectRun(t, rta, `out = undefined.format("v")`, nil, "undefined")

	u := core.Undefined
	s, _ := u.AsString(rta)
	require.Equal(t, rta, "", s)
	require.Equal(t, rta, "undefined", u.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = undefined == %s`, u.String(rta)), nil, true)
}

func TestBoolean(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = bool()`, nil, false)
	expectRun(t, rta, `out = bool(true)`, nil, true)
	expectRun(t, rta, `out = bool(false)`, nil, false)

	expectRun(t, rta, `out = true`, nil, true)
	expectRun(t, rta, `out = false`, nil, false)

	expectRun(t, rta, `out = 1 < 2`, nil, true)
	expectRun(t, rta, `out = 1 > 2`, nil, false)
	expectRun(t, rta, `out = 1 < 1`, nil, false)
	expectRun(t, rta, `out = 1 > 2`, nil, false)
	expectRun(t, rta, `out = 1 == 1`, nil, true)
	expectRun(t, rta, `out = 1 != 1`, nil, false)
	expectRun(t, rta, `out = 1 == 2`, nil, false)
	expectRun(t, rta, `out = 1 != 2`, nil, true)
	expectRun(t, rta, `out = 1 <= 2`, nil, true)
	expectRun(t, rta, `out = 1 >= 2`, nil, false)
	expectRun(t, rta, `out = 1 <= 1`, nil, true)
	expectRun(t, rta, `out = 1 >= 2`, nil, false)

	expectRun(t, rta, `out = true == true`, nil, true)
	expectRun(t, rta, `out = false == false`, nil, true)
	expectRun(t, rta, `out = true == false`, nil, false)
	expectRun(t, rta, `out = true != false`, nil, true)
	expectRun(t, rta, `out = false != true`, nil, true)
	expectRun(t, rta, `out = (1 < 2) == true`, nil, true)
	expectRun(t, rta, `out = (1 < 2) == false`, nil, false)
	expectRun(t, rta, `out = (1 > 2) == true`, nil, false)
	expectRun(t, rta, `out = (1 > 2) == false`, nil, true)
	expectRun(t, rta, `out = 5 + true`, nil, 6)
	expectRun(t, rta, `out = 5 + true; 5`, nil, 6)

	expectError(t, rta, `-true`, nil, "invalid_unary_operator: - bool")
	expectError(t, rta, `true + false`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `5; true + false; 5`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `if (10 > 1) { true + false; }`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, rta, `
func() {
	if (10 > 1) {
		if (10 > 1) {
			return true + false;
		}

		return 1;
	}
}()
`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, rta, `if (true + false) { 10 }`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `10 + (true + false)`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `(true + false) + 20`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `!(true + false)`, nil, "invalid_binary_operator: bool + bool")

	var v core.Value

	v = core.True
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "true", s)
	v = core.True
	require.Equal(t, rta, "true", v.String(rta))

	v = core.True
	expectRun(t, rta, fmt.Sprintf(`out = true == %s`, v.String(rta)), nil, true)
	v = core.False
	expectRun(t, rta, fmt.Sprintf(`out = false == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = true.bool()`, nil, true)
	expectRun(t, rta, `out = false.bool()`, nil, false)
	expectRun(t, rta, `out = true.byte()`, nil, byte(1))
	expectRun(t, rta, `out = false.byte()`, nil, byte(0))
	expectRun(t, rta, `out = true.int()`, nil, 1)
	expectRun(t, rta, `out = false.int()`, nil, 0)
	expectRun(t, rta, `out = true.string()`, nil, "true")
	expectRun(t, rta, `out = false.string()`, nil, "false")
	expectRun(t, rta, `out = false.format()`, nil, "false")
	expectRun(t, rta, `out = false.format("v")`, nil, "false")
}

func TestByte(t *testing.T) {
	rta := core.NewArena(nil)
	var v core.Value

	expectRun(t, rta, `out = byte(5)`, nil, byte(5))
	expectRun(t, rta, `out = byte(true)`, nil, byte(1))
	expectRun(t, rta, `out = byte(false)`, nil, byte(0))
	expectRun(t, rta, `out = byte('A')`, nil, byte(65))
	expectRun(t, rta, `out = byte("12")`, nil, byte(12))
	expectRun(t, rta, `out = byte(u"12")`, nil, byte(12))
	expectRun(t, rta, `out = byte(u"300", byte(7))`, nil, byte(7))
	expectRun(t, rta, `out = byte(255) + 1`, nil, byte(0))
	expectRun(t, rta, `out = byte(255) + 2`, nil, byte(1))
	expectRun(t, rta, `out = byte(0) - 1`, nil, byte(255))
	expectRun(t, rta, `out = 1 + byte(255)`, nil, int64(256))

	v = core.ByteValue(0)
	expectRun(t, rta, fmt.Sprintf(`out = byte(0) == %s`, v.String(rta)), nil, true)
	v = core.ByteValue(1)
	expectRun(t, rta, fmt.Sprintf(`out = byte(1) == %s`, v.String(rta)), nil, true)
	v = core.ByteValue(123)
	expectRun(t, rta, fmt.Sprintf(`out = byte(123) == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = byte(123).int()`, nil, 123)
	expectRun(t, rta, `out = byte(0).bool()`, nil, false)
	expectRun(t, rta, `out = byte(10).bool()`, nil, true)
	expectRun(t, rta, `out = byte(48).rune()`, nil, '0')
	expectRun(t, rta, `out = byte(48).float()`, nil, 48.0)
	expectRun(t, rta, `out = byte(48).string()`, nil, "48")
	expectRun(t, rta, `out = byte(48).format()`, nil, "48")
	expectRun(t, rta, `out = byte(48).format("v")`, nil, "byte(48)")
}

func TestInteger(t *testing.T) {
	rta := core.NewArena(nil)
	var v core.Value

	expectRun(t, rta, `out = 5`, nil, 5)
	expectRun(t, rta, `out = 10`, nil, 10)
	expectRun(t, rta, `out = -5`, nil, -5)
	expectRun(t, rta, `out = -10`, nil, -10)
	expectRun(t, rta, `out = 5 + 5 + 5 + 5 - 10`, nil, 10)
	expectRun(t, rta, `out = 2 * 2 * 2 * 2 * 2`, nil, 32)
	expectRun(t, rta, `out = -50 + 100 + -50`, nil, 0)
	expectRun(t, rta, `out = 5 * 2 + 10`, nil, 20)
	expectRun(t, rta, `out = 5 + 2 * 10`, nil, 25)
	expectRun(t, rta, `out = 20 + 2 * -10`, nil, 0)
	expectRun(t, rta, `out = 50 / 2 * 2 + 10`, nil, 60)
	expectRun(t, rta, `out = 2 * (5 + 10)`, nil, 30)
	expectRun(t, rta, `out = 3 * 3 * 3 + 10`, nil, 37)
	expectRun(t, rta, `out = 3 * (3 * 3) + 10`, nil, 37)
	expectRun(t, rta, `out = (5 + 10 * 2 + 15 /3) * 2 + -10`, nil, 50)
	expectRun(t, rta, `out = 5 % 3`, nil, 2)
	expectRun(t, rta, `out = 5 % 3 + 4`, nil, 6)
	expectRun(t, rta, `out = +5`, nil, 5)
	expectRun(t, rta, `out = +5 + -5`, nil, 0)

	expectRun(t, rta, `out = 9 + '0'`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, rta, `out = '9' - 5`, nil, 52) // '9' is 57 in ASCII

	v = core.IntValue(0)
	expectRun(t, rta, fmt.Sprintf(`out = 0 == %s`, v.String(rta)), nil, true)
	v = core.IntValue(1)
	expectRun(t, rta, fmt.Sprintf(`out = 1 == %s`, v.String(rta)), nil, true)
	v = core.IntValue(1234567890)
	expectRun(t, rta, fmt.Sprintf(`out = 1234567890 == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = 5 + "-5"`, nil, 0)
	expectRun(t, rta, `out = 5 + "5"`, nil, 10)

	expectRun(t, rta, `out = (12).int()`, nil, 12)
	expectRun(t, rta, `out = (0).bool()`, nil, false)
	expectRun(t, rta, `out = (10).bool()`, nil, true)
	expectRun(t, rta, `out = (48).rune()`, nil, '0')
	expectRun(t, rta, `out = (48).float()`, nil, 48.0)
	expectRun(t, rta, `out = (48).string()`, nil, "48")
	expectRun(t, rta, `out = (1234567890).time().utc().string()`, nil, "2009-02-13 23:31:30 +0000 UTC")
	expectRun(t, rta, `out = (48).byte()`, nil, byte(48))
	expectRun(t, rta, `out = (48).format()`, nil, "48")
	expectRun(t, rta, `out = (48).format("v")`, nil, "48")
}

func TestFloat(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = 0.0`, nil, 0.0)
	expectRun(t, rta, `out = -10.3`, nil, -10.3)
	expectRun(t, rta, `out = 3.2 + 2.0 * -4.0`, nil, -4.8)
	expectRun(t, rta, `out = 4 + 2.3`, nil, 6.3)
	expectRun(t, rta, `out = 2.3 + 4`, nil, 6.3)
	expectRun(t, rta, `out = +5.0`, nil, 5.0)
	expectRun(t, rta, `out = -5.0 + +5.0`, nil, 0.0)

	v := core.FloatValue(0.0)
	expectRun(t, rta, fmt.Sprintf(`out = 0.0 == %s`, v.String(rta)), nil, true)
	v = core.FloatValue(1.0)
	expectRun(t, rta, fmt.Sprintf(`out = 1.0 == %s`, v.String(rta)), nil, true)
	v = core.FloatValue(12345.6789)
	expectRun(t, rta, fmt.Sprintf(`out = 12345.6789 == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = 5.0 + "-5.0"`, nil, 0.0)
	expectRun(t, rta, `out = 5.0 + "5.0"`, nil, 10.0)

	expectRun(t, rta, `out = (1.5).float()`, nil, 1.5)
	expectRun(t, rta, `out = (1.5).int()`, nil, 1)
	expectRun(t, rta, `out = (1.5).string()`, nil, "1.5")

	// f-suffix float literals
	expectRun(t, rta, `out = 1f`, nil, 1.0)
	expectRun(t, rta, `out = 1.5f`, nil, 1.5)
	expectRun(t, rta, `out = type_name(1f)`, nil, "float")
	expectRun(t, rta, `out = type_name(1.5f)`, nil, "float")
	expectRun(t, rta, `out = 2f + 3f`, nil, 5.0)
}

func TestDecimal(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = decimal(123)`, nil, dec128.FromInt64(123))
	expectRun(t, rta, `out = decimal(1.23)`, nil, dec128.FromFloat64(1.23))
	expectRun(t, rta, `out = decimal("1.23")`, nil, dec128.FromString("1.23"))

	expectRun(t, rta, `out = (123).decimal()`, nil, dec128.FromInt64(123))
	expectRun(t, rta, `out = (1.23).decimal()`, nil, dec128.FromFloat64(1.23))
	expectRun(t, rta, `out = "1.23".decimal()`, nil, dec128.FromString("1.23"))

	expectRun(t, rta, `out = decimal(1) + decimal(2)`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = decimal(1) + 2`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1 + decimal(2)`, nil, dec128.FromString("3"))

	expectRun(t, rta, `out = 1.0 + decimal(2)`, nil, 3.0)
	expectRun(t, rta, `out = decimal(1) + 2.0`, nil, dec128.FromString("3"))

	expectRun(t, rta, `out = 1d`, nil, dec128.FromInt64(1))
	expectRun(t, rta, `out = 1.23d`, nil, dec128.FromString("1.23"))
	expectRun(t, rta, `out = type_name(1d)`, nil, "decimal")
	expectRun(t, rta, `out = type_name(1.23d)`, nil, "decimal")
	expectRun(t, rta, `out = 1d + 2d`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1d + 2`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1 + 2d`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1.5d + 0.5d`, nil, dec128.FromString("2"))
	expectRun(t, rta, `out = -1d`, nil, dec128.FromInt64(-1))

	expectRun(t, rta, `out = (1.23d).decimal()`, nil, dec128.FromString("1.23"))
	expectRun(t, rta, `out = (123d).float().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, rta, `out = (123d).int().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, rta, `out = (1.23d).string()`, nil, "1.23")
	expectRun(t, rta, `out = (1.23d).is_zero()`, nil, false)
	expectRun(t, rta, `out = (0d).is_zero()`, nil, true)
	expectRun(t, rta, `out = (0d).is_negative()`, nil, false)
	expectRun(t, rta, `out = (1d).is_negative()`, nil, false)
	expectRun(t, rta, `out = (-1d).is_negative()`, nil, true)
	expectRun(t, rta, `out = (0d).is_positive()`, nil, false)
	expectRun(t, rta, `out = (1d).is_positive()`, nil, true)
	expectRun(t, rta, `out = (-1d).is_positive()`, nil, false)
	expectRun(t, rta, `out = (0d).sign()`, nil, 0)
	expectRun(t, rta, `out = (1d).sign()`, nil, 1)
	expectRun(t, rta, `out = (-1d).sign()`, nil, -1)
	expectRun(t, rta, `out = (123d).rescale(2).scale()`, nil, 2)
	expectRun(t, rta, `out = (123d).rescale(2).canonical().scale()`, nil, 0)
	expectRun(t, rta, `out = (1.23d).format()`, nil, "1.23")
	expectRun(t, rta, `out = (1.23d).format("v")`, nil, "1.23d")
}

func TestRune(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = 'a'`, nil, 'a')
	expectRun(t, rta, `out = 'あ'`, nil, rune(12354))
	expectRun(t, rta, `out = 'Æ'`, nil, rune(198))

	expectRun(t, rta, `out = '0' + '9'`, nil, rune(105))
	expectRun(t, rta, `out = '0' + 9`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, rta, `out = '9' - 4`, nil, 53) // '9' is 57 in ASCII
	expectRun(t, rta, `out = '0' == '0'`, nil, true)
	expectRun(t, rta, `out = '0' != '0'`, nil, false)
	expectRun(t, rta, `out = '2' < '4'`, nil, true)
	expectRun(t, rta, `out = '2' > '4'`, nil, false)
	expectRun(t, rta, `out = '2' <= '4'`, nil, true)
	expectRun(t, rta, `out = '2' >= '4'`, nil, false)
	expectRun(t, rta, `out = '4' < '4'`, nil, false)
	expectRun(t, rta, `out = '4' > '4'`, nil, false)
	expectRun(t, rta, `out = '4' <= '4'`, nil, true)
	expectRun(t, rta, `out = '4' >= '4'`, nil, true)

	v := core.RuneValue('A')
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "A", s)
	v = core.RuneValue('A')
	require.Equal(t, rta, "'A'", v.String(rta))

	v = core.RuneValue('0')
	expectRun(t, rta, fmt.Sprintf(`out = '0' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('A')
	expectRun(t, rta, fmt.Sprintf(`out = 'A' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('₴')
	expectRun(t, rta, fmt.Sprintf(`out = '₴' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('\'')
	expectRun(t, rta, fmt.Sprintf(`out = '\'' == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = '4' + 4`, nil, 56) // '4' is 52 in ASCII
	expectRun(t, rta, `out = '4' + "4"`, nil, "44")
	expectError(t, rta, `'4' - "4"`, nil, "invalid_binary_operator: rune - string")

	expectRun(t, rta, `out = '4'.rune()`, nil, '4')
	expectRun(t, rta, `out = '4'.bool()`, nil, true)
	expectRun(t, rta, `out = '4'.int()`, nil, 52)
	expectRun(t, rta, `out = '4'.string()`, nil, "4")
	expectRun(t, rta, `out = '4'.format()`, nil, "4")
	expectRun(t, rta, `out = '4'.format("v")`, nil, "'4'")
}

func TestString(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = "Hello World!"`, nil, "Hello World!")
	expectRun(t, rta, `out = "Hello" + " " + "World!"`, nil, "Hello World!")

	expectRun(t, rta, `out = "Hello" == "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello" == "World"`, nil, false)
	expectRun(t, rta, `out = "Hello" != "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello" != "World"`, nil, true)

	expectRun(t, rta, `out = "Hello" > "World"`, nil, false)
	expectRun(t, rta, `out = "World" < "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello" < "World"`, nil, true)
	expectRun(t, rta, `out = "World" > "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello" >= "World"`, nil, false)
	expectRun(t, rta, `out = "Hello" <= "World"`, nil, true)
	expectRun(t, rta, `out = "Hello" >= "Hello"`, nil, true)
	expectRun(t, rta, `out = "World" <= "World"`, nil, true)
	expectRun(t, rta, `out = "el" in "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello".contains("el")`, nil, true)
	expectRun(t, rta, `out = 'e' in "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello".contains('e')`, nil, true)
	expectRun(t, rta, `out = "z" in "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello".contains("z")`, nil, false)
	expectRun(t, rta, `out = "z" not in "Hello"`, nil, true)

	// index operator
	str := "abcdef"
	strStr := `"abcdef"`
	strLen := 6
	for idx := range strLen {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("idx = %d; out = %s[idx]", idx, strStr), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, -idx-1), nil, str[strLen-idx-1])
	}

	expectError(t, rta, fmt.Sprintf("%s[%d]", strStr, -strLen-1), nil, "index_out_of_bounds")
	expectError(t, rta, fmt.Sprintf("%s[%d]", strStr, strLen), nil, "index_out_of_bounds")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, -2), nil, str[strLen-2])

	// slice operator
	for low := 0; low <= strLen; low++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, low, low), nil, "")
		for high := low; high <= strLen; high++ {
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, high), nil, str[:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", strStr, low), nil, str[low:])
		}
	}

	expectRun(t, rta, fmt.Sprintf("out = %s[:]", strStr), nil, str[:])
	expectRun(t, rta, fmt.Sprintf("out = %s[:]", strStr), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", strStr, -1), nil, str[strLen-1:])
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, strLen+1), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 2), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, -1), nil, str[:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 0, -1), nil, str[:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, -3, -1), nil, str[strLen-3:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 1, -1), nil, str[1:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 1), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 10, 20), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, -100, 100), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:2]", strStr), nil, "bd")
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:-1]", strStr), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[5:1:-1]", strStr), nil, "fedc")
	expectRun(t, rta, fmt.Sprintf("out = %s[0:%d:2]", strStr, strLen), nil, "ace")
	expectRun(t, rta, fmt.Sprintf("out = %s[::-1]", strStr), nil, "fedcba")
	expectError(t, rta, fmt.Sprintf("out = %s[::0]", strStr), nil, "step cannot be zero")

	// string concatenation with other types
	expectRun(t, rta, `out = "foo" + 1`, nil, "foo1")
	// Float.string() returns the smallest number of digits necessary such that ParseFloat will return f exactly.
	expectRun(t, rta, `out = "foo" + 1.0`, nil, "foo1") // <- note '1' instead of '1.0'
	expectRun(t, rta, `out = "foo" + 1.5`, nil, "foo1.5")
	expectRun(t, rta, `out = "foo" + true`, nil, "footrue")
	expectRun(t, rta, `out = "foo" + 'X'`, nil, "fooX")
	expectRun(t, rta, `out = "foo" + error(5)`, nil, "foo5")
	expectRun(t, rta, `out = "foo" + [100, 101]`, nil, "foode")
	// also works with "+=" operator
	expectRun(t, rta, `out = "foo"; out += 1.5`, nil, "foo1.5")

	// string concat works only when string is LHS
	expectError(t, rta, `1 + "foo"`, nil, "invalid_binary_operator: int + string")

	// there is no '-' operator for string
	expectError(t, rta, `"foo" - "bar"`, nil, "invalid_binary_operator: string - string")

	// undefined cannot be added to string
	expectError(t, rta, `"foo" + undefined`, nil, "invalid_binary_operator: string + undefined")

	v := rta.MustNewStringValue("abc")
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "abc", s)
	v = rta.MustNewStringValue("abc")
	require.Equal(t, rta, `"abc"`, v.String(rta))

	v = rta.MustNewStringValue("")
	expectRun(t, rta, fmt.Sprintf(`out = "" == %s`, v.String(rta)), nil, true)
	v = rta.MustNewStringValue("hello")
	expectRun(t, rta, fmt.Sprintf(`out = "hello" == %s`, v.String(rta)), nil, true)
	v = rta.MustNewStringValue("hello \"world\"")
	expectRun(t, rta, fmt.Sprintf(`out = "hello \"world\"" == %s`, v.String(rta)), nil, true)
	v = rta.MustNewStringValue("123₴")
	expectRun(t, rta, fmt.Sprintf(`out = "123₴" == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = "".is_empty()`, nil, true)
	expectRun(t, rta, `out = "abcd".is_empty()`, nil, false)
	expectRun(t, rta, `out = "abcd".len()`, nil, 4)
	expectRun(t, rta, `out = "Abcd".lower()`, nil, "abcd")
	expectRun(t, rta, `out = "Abcd".upper()`, nil, "ABCD")
	expectRun(t, rta, `out = "abcd ".trim()`, nil, "abcd")
	expectRun(t, rta, `out = "abcd".trim("ad")`, nil, "bc")
	expectRun(t, rta, `out = "".reverse()`, nil, "")
	expectRun(t, rta, `out = "a".reverse()`, nil, "a")
	expectRun(t, rta, `out = "hello".reverse()`, nil, "olleh")
	expectRun(t, rta, `out = "їЇґҐ".reverse()`, nil, "ҐґЇї")
	expectRun(t, rta, `out = "こんにちは".reverse()`, nil, "はちにんこ")

	expectRun(t, rta, `out = "abc".string()`, nil, "abc")
	expectRun(t, rta, `out = "abc".array()`, nil, ARR{int64('a'), int64('b'), int64('c')})
	expectRun(t, rta, `out = "abc".array().string()`, nil, "abc")
	expectRun(t, rta, `out = "true".bool()`, nil, true)
	expectRun(t, rta, `out = "false".bool()`, nil, false)
	expectRun(t, rta, `out = "abc".bool()`, nil, false)
	expectRun(t, rta, `out = "true".bool().string()`, nil, "true")
	expectRun(t, rta, `out = "abc".bytes()`, nil, rta.MustNewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, rta, `out = "abc".bytes().string()`, nil, "abc")
	expectRun(t, rta, `out = "1.2".float()`, nil, 1.2)
	expectRun(t, rta, `out = "1.2".float().string()`, nil, "1.2")
	expectRun(t, rta, `out = "12".byte()`, nil, byte(12))
	expectRun(t, rta, `out = u"12".byte()`, nil, byte(12))
	expectRun(t, rta, `out = "12".int()`, nil, 12)
	expectRun(t, rta, `out = "12".float().string()`, nil, "12")
	expectRun(t, rta, `out = "abc".int()`, nil, 0)
	expectRun(t, rta, `out = "abc".record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = "abc".dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = "abc".format()`, nil, "abc")
	expectRun(t, rta, `out = "abc".format("v")`, nil, `"abc"`)

	expectRun(t, rta, `out = " їЇґҐ ".trim()`, nil, "їЇґҐ")
	expectRun(t, rta, `out = "їЇґҐ".upper()`, nil, "ЇЇҐҐ")
	expectRun(t, rta, `out = "їЇґҐ".lower()`, nil, "їїґґ")
	expectRun(t, rta, `out = "こんにちはさ"[1]`, nil, byte(129)) // byte index, not rune index
	expectRun(t, rta, `out = "こんにちはさ"[1:2]`, nil, "\x81")  // byte slice, not rune slice
	expectRun(t, rta, `out = "こんにちはさ"[0:3]`, nil, "こ")     // byte slice, not rune slice

	expectRun(t, rta, `out = len("")`, nil, 0)
	expectRun(t, rta, `out = len("hello")`, nil, 5)
	expectRun(t, rta, `out = len("їЇґҐ")`, nil, 8)    // byte length, not rune length
	expectRun(t, rta, `out = len("こんにちはさ")`, nil, 18) // byte length, not rune length

	expectRun(t, rta, `out = "hello".filter(x => x > 'e')`, nil, "hllo")
	expectRun(t, rta, `out = "hello".filter((i, x) => i > 2)`, nil, "lo")
	expectRun(t, rta, `out = "hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = "hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = "hello".all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = "hello".all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = "hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = "hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = "hello".any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = "hello".any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = "hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = "hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = "hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = "hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = "hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = "hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = "".find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = "x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = "x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = "x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `
out = ""
ignored := "hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, rta, `
out = 0
ignored := "abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunes(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = u"Hello World!"`, nil, []rune("Hello World!"))
	expectRun(t, rta, `out = u"Hello" + u" " + "World!"`, nil, []rune("Hello World!"))

	expectRun(t, rta, `out = u"Hello" == "Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" == u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" == u"World"`, nil, false)
	expectRun(t, rta, `out = u"Hello" != u"Hello"`, nil, false)
	expectRun(t, rta, `out = u"Hello" != u"World"`, nil, true)

	expectRun(t, rta, `out = u"Hello" > u"World"`, nil, false)
	expectRun(t, rta, `out = u"World" < u"Hello"`, nil, false)
	expectRun(t, rta, `out = u"Hello" < u"World"`, nil, true)
	expectRun(t, rta, `out = u"World" > u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" >= u"World"`, nil, false)
	expectRun(t, rta, `out = u"Hello" <= u"World"`, nil, true)
	expectRun(t, rta, `out = u"Hello" >= u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"World" <= u"World"`, nil, true)
	expectRun(t, rta, `out = u"el" in u"Hello"`, nil, true)
	expectRun(t, rta, `out = runes("Hello").contains(u"el")`, nil, true)
	expectRun(t, rta, `out = 'e' in u"Hello"`, nil, true)
	expectRun(t, rta, `out = runes("Hello").contains('e')`, nil, true)
	expectRun(t, rta, `out = runes("z") in u"Hello"`, nil, false)
	expectRun(t, rta, `out = runes("Hello").contains(u"z")`, nil, false)
	expectRun(t, rta, `out = runes("z") not in u"Hello"`, nil, true)

	expectRun(t, rta, `out = runes("").is_empty()`, nil, true)
	expectRun(t, rta, `out = runes("abcd").is_empty()`, nil, false)
	expectRun(t, rta, `out = runes("abcd").len()`, nil, 4)
	expectRun(t, rta, `out = runes("abcd").first()`, nil, 'a')
	expectRun(t, rta, `out = runes("abcd").last()`, nil, 'd')
	expectRun(t, rta, `out = runes("Abcd").lower()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = runes("Abcd").upper()`, nil, []rune("ABCD"))
	expectRun(t, rta, `out = runes("abcd ").trim()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = runes("abcd").trim("ad")`, nil, []rune("bc"))
	expectRun(t, rta, `out = runes("").reverse()`, nil, []rune(""))
	expectRun(t, rta, `out = runes("hello").reverse()`, nil, []rune("olleh"))
	expectRun(t, rta, `out = u"hello".reverse()`, nil, []rune("olleh"))
	expectRun(t, rta, `out = u"їЇґҐ".reverse()`, nil, []rune("ҐґЇї"))
	expectRun(t, rta, `out = u"こんにちは".reverse()`, nil, []rune("はちにんこ"))

	expectRun(t, rta, `out = runes("abc").string()`, nil, "abc")
	expectRun(t, rta, `out = runes("abc").array()`, nil, ARR{'a', 'b', 'c'})
	expectRun(t, rta, `out = runes("abc").array().string()`, nil, "abc")
	expectRun(t, rta, `out = runes("true").bool()`, nil, true)
	expectRun(t, rta, `out = runes("false").bool()`, nil, false)
	expectRun(t, rta, `out = runes("abc").bool()`, nil, false)
	expectRun(t, rta, `out = runes("true").bool().string()`, nil, "true")
	expectRun(t, rta, `out = runes("abc").bytes()`, nil, rta.MustNewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, rta, `out = runes("abc").bytes().string()`, nil, "abc")
	expectRun(t, rta, `out = runes("1.2").float()`, nil, 1.2)
	expectRun(t, rta, `out = runes("1.2").float().string()`, nil, "1.2")
	expectRun(t, rta, `out = runes("12").int()`, nil, 12)
	expectRun(t, rta, `out = runes("12").float().string()`, nil, "12")
	expectRun(t, rta, `out = runes("abc").int()`, nil, 0)
	expectRun(t, rta, `out = runes("abc").record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = runes("abc").dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})

	expectRun(t, rta, `out = runes(" їЇґҐ ").trim()`, nil, []rune("їЇґҐ"))
	expectRun(t, rta, `out = u" їЇґҐ ".trim()`, nil, []rune("їЇґҐ"))

	expectRun(t, rta, `out = u"їЇґҐ".upper()`, nil, []rune("ЇЇҐҐ"))
	expectRun(t, rta, `out = u"їЇґҐ".lower()`, nil, []rune("їїґґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1]`, nil, 'Ї')
	expectRun(t, rta, `out = u"їЇґҐ"[-1]`, nil, 'Ґ')
	expectRun(t, rta, `out = u"їЇґҐ"[-2]`, nil, 'ґ')
	expectRun(t, rta, `out = u"їЇґҐ"[1:2]`, nil, []rune("Ї"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:3]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[:-1]`, nil, []rune("їЇґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:-1]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[-3:-1]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[10:20]`, nil, []rune(""))
	expectRun(t, rta, `out = u"їЇґҐ"[1:4:2]`, nil, []rune("ЇҐ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:4:-1]`, nil, []rune(""))
	expectRun(t, rta, `out = u"їЇґҐ"[3:0:-1]`, nil, []rune("ҐґЇ"))
	expectRun(t, rta, `out = u"їЇґҐ"[0:4:2]`, nil, []rune("їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[::-1]`, nil, []rune("ҐґЇї"))
	expectError(t, rta, `out = u"їЇґҐ"[::0]`, nil, "step cannot be zero")
	expectRun(t, rta, `out = u"こんにちはさ"[1]`, nil, 'ん')
	expectRun(t, rta, `out = u"こんにちはさ"[1:2]`, nil, []rune("ん"))
	expectRun(t, rta, `out = u"こんにちはさ"[1:3]`, nil, []rune("んに"))
	expectRun(t, rta, `out = u"こんにちはさ"[-2:]`, nil, []rune("はさ"))
	expectError(t, rta, `out = u"こんにちはさ"[-7]`, nil, "index_out_of_bounds")

	expectRun(t, rta, `out = len(u"")`, nil, 0)
	expectRun(t, rta, `out = len(u"hello")`, nil, 5)
	expectRun(t, rta, `out = len(u"їЇґҐ")`, nil, 4)
	expectRun(t, rta, `out = len(u"こんにちはさ")`, nil, 6)

	expectRun(t, rta, `out = runes("abc").format()`, nil, "abc")
	expectRun(t, rta, `out = runes("abc").format("v")`, nil, `u"abc"`)

	expectRun(t, rta, `out = u"hello".sort()`, nil, []rune("ehllo"))
	expectRun(t, rta, `out = u"".dedup()`, nil, []rune(""))
	expectRun(t, rta, `out = u"aabbccd".dedup()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = u"abc".dedup()`, nil, []rune("abc"))
	expectRun(t, rta, `out = u"aaaa".dedup()`, nil, []rune("a"))
	expectRun(t, rta, `out = u"abab".dedup()`, nil, []rune("abab"))
	expectRun(t, rta, `out = u"hello".sort().dedup()`, nil, []rune("ehlo"))
	expectRun(t, rta, `out = u"їЇїЇ".dedup()`, nil, []rune("їЇїЇ"))
	expectRun(t, rta, `out = u"їїЇЇ".dedup()`, nil, []rune("їЇ"))
	expectRun(t, rta, `out = u"".unique()`, nil, []rune(""))
	expectRun(t, rta, `out = u"abc".unique()`, nil, []rune("abc"))
	expectRun(t, rta, `out = u"hello".unique()`, nil, []rune("helo"))
	expectRun(t, rta, `out = u"abab".unique()`, nil, []rune("ab"))
	expectRun(t, rta, `out = u"їЇїЇ".unique()`, nil, []rune("їЇ"))
	expectRun(t, rta, `out = u"".chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = u"hello".chunk(2)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, rta, `out = u"hello".chunk(2, true)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, rta, `out = u"hello".chunk(10)`, nil, ARR{[]rune("hello")})
	expectRun(t, rta, `out = u"hello".filter(x => x > 'e')`, nil, []rune("hllo"))
	expectRun(t, rta, `out = u"hello".filter((i, x) => i > 2)`, nil, []rune("lo"))
	expectRun(t, rta, `out = u"hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = u"hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = u"hello".all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = u"hello".all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = u"hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = u"hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = u"hello".any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = u"hello".any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = u"hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = u"hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = u"hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = u"hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = u"hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = u"hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = u"".find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = u"x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = u"x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = u"x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `out = u"hello".min()`, nil, 'e')
	expectRun(t, rta, `out = u"hello".max()`, nil, 'o')
	expectRun(t, rta, `
out = ""
ignored := u"hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, rta, `
out = 0
ignored := u"abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunesMutability(t *testing.T) {
	rta := core.NewArena(nil)

	// index assignment
	expectRun(t, rta, `r := runes("hello"); r[0] = 'H'; out = r`, nil, []rune("Hello"))
	expectRun(t, rta, `r := runes("hello"); r[-2] = '!'; out = r`, nil, []rune("hel!o"))
	expectRun(t, rta, `r := runes("hello"); r[0] = 0x41; out = r`, nil, []rune("Aello"))

	// append
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c'); out = r2`, nil, []rune("abc"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c', 'd'); out = r2`, nil, []rune("abcd"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, runes("cd")); out = r2`, nil, []rune("abcd"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c'); out = r`, nil, []rune("ab"))

	// sum / avg / map / reduce
	expectRun(t, rta, `out = runes("abc").sum()`, nil, 97+98+99)
	expectRun(t, rta, `out = runes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, rta, `out = runes("").sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = runes("").avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = runes("abc").map(func(r) { return r + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, rta, `out = runes("abc").map(func(i, r) { return [i, r] })`, nil, ARR{ARR{0, 'a'}, ARR{1, 'b'}, ARR{2, 'c'}})
	expectRun(t, rta, `out = runes("abc").reduce(0, func(acc, r) { return acc + r })`, nil, int64('a'+'b'+'c'))
	expectRun(t, rta, `out = runes("abc").reduce("", func(acc, i, r) { return acc + i.string() + r.string() })`, nil, "0a1b2c")

	// type names
	expectRun(t, rta, `out = type_name(runes("abc"))`, nil, "runes")
	expectRun(t, rta, `out = type_name(immutable(runes("abc")))`, nil, "immutable-runes")

	// immutable rejects writes
	expectError(t, rta, `r := immutable(runes("abc")); r[0] = 'X'`, nil, "not_assignable: type immutable-runes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, rta, `out = type_name(immutable(runes("abcd"))[1:3])`, nil, "immutable-runes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, rta, `out = type_name(immutable(runes("abcd"))[::-1])`, nil, "runes")
	// slice of mutable stays mutable
	expectRun(t, rta, `out = type_name(runes("abcd")[1:3])`, nil, "runes")

	// copy of immutable yields mutable
	expectRun(t, rta, `r := immutable(runes("abc")); c := copy(r); c[0] = 'X'; out = c`, nil, []rune("Xbc"))

	// append on immutable returns a fresh mutable value (does not mutate source)
	expectRun(t, rta, `r := immutable(runes("ab")); r2 := append(r, 'c'); r2[0] = 'X'; out = r2`, nil, []rune("Xbc"))
	expectRun(t, rta, `r := immutable(runes("ab")); r2 := append(r, 'c'); out = type_name(r2)`, nil, "runes")

	// invalid assignment values
	expectError(t, rta, `r := runes("abc"); r[0] = "xy"`, nil, "invalid_index_type: (index assign value) expected rune, got string")
	expectError(t, rta, `r := runes("abc"); r[10] = 'X'`, nil, "index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestError(t *testing.T) {
	rta := core.NewArena(nil)

	expectError(t, rta, `out = error()`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 0")
	expectRun(t, rta, `out = error(1)`, nil, errorObject(rta, 1))
	expectRun(t, rta, `out = error(1).value()`, nil, 1)
	expectRun(t, rta, `out = error("some error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = error("some" + " error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = func() { return error(5) }()`, nil, errorObject(rta, 5))
	expectRun(t, rta, `out = error(error("foo"))`, nil, errorObject(rta, errorObject(rta, "foo")))
	expectRun(t, rta, `out = error("some error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = error("some error").value()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").string()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").format()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").format("v")`, nil, `error("some error")`)

	expectRun(t, rta, `out = error("x").is_fatal()`, nil, false)
	expectRun(t, rta, `out = error("x", false).is_fatal()`, nil, false)
	expectRun(t, rta, `out = error("x", true).is_fatal()`, nil, true)
	expectError(t, rta, `out = error("x").is_fatal(1)`, nil, "wrong_num_arguments: (is_fatal) expected 0 argument(s), got 1")

	expectError(t, rta, `error("error").err`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, rta, `error("error").value_`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, rta, `error([1,2,3])[1]`, nil, "not_accessible: type error does not support indexing or field access")

	s, _ := rta.MustNewErrorValue(rta.MustNewStringValue("abc"), core.KindUser, false).AsString(rta)
	require.Equal(t, rta, "abc", s)
	require.Equal(t, rta, `error("abc")`, rta.MustNewErrorValue(rta.MustNewStringValue("abc"), core.KindUser, false).String(rta))

	v := rta.MustNewErrorValue(core.Undefined, core.KindUser, false)
	require.Equal(t, rta, "error()", v.String(rta))
	expectRun(t, rta, `out = error(undefined) == error(undefined)`, nil, true)
	v = rta.MustNewErrorValue(rta.MustNewStringValue("some error"), core.KindUser, false)
	expectRun(t, rta, fmt.Sprintf(`out = error("some error") == %s`, v.String(rta)), nil, true)
}

func TestArray(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = [1, 2 * 2, 3 + 3]`, nil, ARR{1, 4, 6})

	// array copy-by-reference
	expectRun(t, rta, `a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2`, nil, ARR{5, 2, 3})
	expectRun(t, rta, `func () { a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2 }()`, nil, ARR{5, 2, 3})

	// array index set
	expectError(t, rta, `a1 := [1, 2, 3]; a1[3] = 5`, nil, "index_out_of_bounds")

	// index operator
	arr := ARR{1, 2, 3, 4, 5, 6}
	arrStr := `[1, 2, 3, 4, 5, 6]`
	arrLen := 6
	for idx := 0; idx < arrLen; idx++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("idx := %d; out = %s[idx]", idx, arrStr), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, -idx-1), nil, arr[arrLen-idx-1])
	}

	expectError(t, rta, fmt.Sprintf("%s[%d]", arrStr, -arrLen-1), nil, "index_out_of_bounds")
	expectError(t, rta, fmt.Sprintf("%s[%d]", arrStr, arrLen), nil, "index_out_of_bounds")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, -2), nil, arr[arrLen-2])
	expectRun(t, rta, `a1 := [1, 2, 3]; a1[-1] = 5; out = a1[2]`, nil, 5)
	expectError(t, rta, `a1 := [1, 2, 3]; a1[-4] = 5`, nil, "index_out_of_bounds")

	// slice operator
	for low := 0; low < arrLen; low++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, low), nil, ARR{})
		for high := low; high <= arrLen; high++ {
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, high), nil, arr[:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", arrStr, low), nil, arr[low:])
		}
	}

	expectRun(t, rta, fmt.Sprintf("out = %s[:]", arrStr), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", arrStr, -1), nil, ARR{6})
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, arrLen+1), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 2), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 0, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 1, -1), nil, ARR{2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, -3, -1), nil, ARR{4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 1), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 10, 20), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, -100, 100), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:2]", arrStr), nil, ARR{2, 4})
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:-1]", arrStr), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[5:1:-1]", arrStr), nil, ARR{6, 5, 4, 3})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d:%d]", arrStr, 0, arrLen, 2), nil, ARR{1, 3, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[::-1]", arrStr), nil, ARR{6, 5, 4, 3, 2, 1})
	expectError(t, rta, fmt.Sprintf("out = %s[::0]", arrStr), nil, "step cannot be zero")

	v := rta.MustNewArrayValue(nil, false)
	expectRun(t, rta, fmt.Sprintf(`out = [] == %s`, v.String(rta)), nil, true)
	v = rta.MustNewArrayValue(nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = [] == %s`, v.String(rta)), nil, true)

	v = rta.MustNewArrayValue([]core.Value{
		core.IntValue(1),
		core.Undefined,
		rta.MustNewStringValue("3"),
	}, false)
	expectRun(t, rta, fmt.Sprintf(`out = [1, undefined, "3"] == %s`, v.String(rta)), nil, true)

	expectError(t, rta, `[1, 2, 3].q`, nil, "Runtime Error: invalid_selector: type array has no property \"q\"\n\tat test:1:11")

	expectRun(t, rta, `t := []; out = t.sort()`, nil, ARR{})
	expectRun(t, rta, `t := [1, 2, 3]; out = t.sort()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `t := [3, 2, 1]; out = t.sort()`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `out = [].dedup()`, nil, ARR{})
	expectRun(t, rta, `out = [1].dedup()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 1, 2, 2, 3, 3, 3, 1].dedup()`, nil, ARR{1, 2, 3, 1})
	expectRun(t, rta, `out = [1, 2, 3].dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [1, 2, 1, 2].dedup()`, nil, ARR{1, 2, 1, 2})
	expectRun(t, rta, `out = [3, 1, 2, 1, 3, 2].sort().dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = ["a", "a", "b", "a"].dedup()`, nil, ARR{"a", "b", "a"})
	expectRun(t, rta, `out = [1, 1.0, "1"].dedup()`, nil, ARR{1})
	expectRun(t, rta, `out = [[1, 2], [1, 2], [3]].dedup()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, rta, `out = [].unique()`, nil, ARR{})
	expectRun(t, rta, `out = [1].unique()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 2, 3].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [1, 1, 2, 2, 3, 3, 3, 1].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [3, 1, 2, 1, 3, 2].unique()`, nil, ARR{3, 1, 2})
	expectRun(t, rta, `out = ["a", "b", "a", "c", "b"].unique()`, nil, ARR{"a", "b", "c"})
	expectRun(t, rta, `out = [1, 1.0, "1"].unique()`, nil, ARR{1})
	expectRun(t, rta, `out = [[1, 2], [3], [1, 2]].unique()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, rta, `out = [].reverse()`, nil, ARR{})
	expectRun(t, rta, `out = [1].reverse()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 2, 3].reverse()`, nil, ARR{3, 2, 1})
	expectRun(t, rta, `out = ["a", "b", "c"].reverse()`, nil, ARR{"c", "b", "a"})
	expectRun(t, rta, `out = [1, 2, 3].reverse().reverse()`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `t := []; out = t.is_empty()`, nil, true)
	expectRun(t, rta, `t := [1, 2, 3]; out = t.is_empty()`, nil, false)

	expectRun(t, rta, `t := []; out = t.len()`, nil, 0)
	expectRun(t, rta, `t := [1, 2, 3]; out = t.len()`, nil, 3)

	expectRun(t, rta, `out = [].first()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].first()`, nil, 1)

	expectRun(t, rta, `out = [].last()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].last()`, nil, 3)

	expectRun(t, rta, `out = [].min()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].min()`, nil, 1)

	expectRun(t, rta, `out = [].max()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].max()`, nil, 3)

	expectRun(t, rta, `out = [].sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].sum()`, nil, 6)

	expectRun(t, rta, `out = [].avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].avg()`, nil, 2)

	expectRun(t, rta, `out = [].count(x => x > 0)`, nil, 0)
	expectRun(t, rta, `out = [1, 2, 3, -10].count(x => x > 0)`, nil, 3)
	expectRun(t, rta, `out = [1, 2, 3, -10].count((i, x) => x == i+1)`, nil, 3)

	expectRun(t, rta, `out = [1, 2, 3].filter(x => x == 2)`, nil, ARR{2})
	expectRun(t, rta, `out = [1, 2, 3].filter(x => x != 2)`, nil, ARR{1, 3})
	expectRun(t, rta, `out = [1, undefined, 2, undefined, 3].filter()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [].filter()`, nil, ARR{})
	expectRun(t, rta, `out = [undefined, undefined].filter()`, nil, ARR{})

	expectRun(t, rta, `out = [].all(x => x > 0)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].all(x => x > 0)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].all(x => x > -100)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].all((i, x) => x == i+1)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, 4].all((i, x) => x == i+1)`, nil, true)

	expectRun(t, rta, `out = [].any(x => x > 0)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].any(x => x < 0)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].any(x => x < -100)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].any((i, x) => x != i+1)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, 4].any((i, x) => x != i+1)`, nil, false)

	expectRun(t, rta, `out = [].map(x => x * x)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3].map(x => x * x)`, nil, ARR{1, 4, 9})

	expectRun(t, rta, `out = [].chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3, 4].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}})
	expectRun(t, rta, `out = [1, 2, 3, 4, 5].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}, ARR{5}})
	expectRun(t, rta, `out = [1, 2, 3].chunk(10)`, nil, ARR{ARR{1, 2, 3}})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2, false); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2, true); c[0][0] = 9; out = a`, nil, ARR{1, 2, 3})
	expectError(t, rta, `out = [1, 2, 3].chunk()`, nil, "wrong_num_arguments: (chunk) expected 1 or 2 argument(s), got 0")
	expectError(t, rta, `out = [1, 2, 3].chunk("x")`, nil, "invalid_argument_type: (chunk) argument first expects type int, got string")
	expectError(t, rta, `out = [1, 2, 3].chunk(2, 1)`, nil, "invalid_argument_type: (chunk) argument second expects type bool, got int")
	expectError(t, rta, `out = [1, 2, 3].chunk(0)`, nil, "invalid_value: chunk size must be positive")
	expectError(t, rta, `out = [1, 2, 3].chunk(-1)`, nil, "invalid_value: chunk size must be positive")

	expectRun(t, rta, `
out = 0
ignored := [1, 2, 3, 4].for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)

	expectRun(t, rta, `
out = 0
ignored := [10, 20, 30].for_each(func(i, v) {
	out += i * v
	return true
})
`, nil, 80)

	expectRun(t, rta, `out = [1].for_each(func(v) { return true })`, nil, core.Undefined)
	expectError(t, rta, `out = [1].for_each()`, nil, "wrong_num_arguments: (for_each) expected 1 argument(s), got 0")
	expectError(t, rta, `out = [1].for_each(1)`, nil, "invalid_argument_type: (for_each) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = [1].for_each(func() { return true })`, nil, "invalid_argument_type: (for_each) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = [10, 20, 30].find(x => x == 20)`, nil, 1)
	expectRun(t, rta, `out = [10, 20, 30].find(x => x == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = [10, 20, 30].find((i, v) => i == 2)`, nil, 2)
	expectRun(t, rta, `out = [10, 20, 30].find((i, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = [].find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = [1].find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = [1].find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = [1].find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = [].reduce(0, (a, v) => a + v)`, nil, 0)
	expectRun(t, rta, `out = [1, 2, 3].reduce(0, (a, v) => a + v)`, nil, 6)
	expectRun(t, rta, `out = [1, 2, 3].reduce(0, (a, i, v) => a + i)`, nil, 3)
	expectRun(t, rta, `out = [1, 2].reduce(0, (a, v) => a + [10, 20].reduce(0, (b, w) => b + w) + v)`, nil, 63)

	expectRun(t, rta, `out = [1, 2, 3].array()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [48, 49, -1].bytes()`, nil, rta.MustNewBytesValue([]byte{48, 49, 255}, false))
	expectRun(t, rta, `out = [48, 49, -1].record()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, rta, `out = [48, 49, -1].dict()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, rta, `out = [48, 49, 50].string()`, nil, "012")
	expectRun(t, rta, `out = [48, 49, 50].format("v")`, nil, "[48, 49, 50]")
	expectRun(t, rta, `out = [48, 49, 50].format()`, nil, "[48, 49, 50]")

	expectRun(t, rta, `out = 2 in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains(2)`, nil, true)
	expectRun(t, rta, `out = "2" in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains("2")`, nil, true)
	expectRun(t, rta, `out = "z" in [1, 2, 3]`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3].contains("z")`, nil, false)
	expectRun(t, rta, `out = [2, 3] in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains([2, 3])`, nil, true)
	expectRun(t, rta, `out = [] in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains([])`, nil, true)
	expectRun(t, rta, `out = [1, 3] in [1, 2, 3]`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3].contains([1, 3])`, nil, false)
	expectRun(t, rta, `out = [1, 3] not in [1, 2, 3]`, nil, true)
}

func TestRecord(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `
out = {
	one: 10 - 9,
	two: 1 + 1,
	three: 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, rta, `
out = {
	"one": 10 - 9,
	"two": 1 + 1,
	"three": 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, rta, `out = {foo: 5}["foo"]`, nil, 5)
	expectRun(t, rta, `out = {foo: 5}["bar"]`, nil, core.Undefined)
	expectRun(t, rta, `key := "foo"; out = {foo: 5}[key]`, nil, 5)
	expectRun(t, rta, `out = {}["foo"]`, nil, core.Undefined)

	expectRun(t, rta, `
m := {
	foo: func(x) {
		return x * 2
	}
}
out = m["foo"](2) + m["foo"](3)
`, nil, 10)

	expectRun(t, rta, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1`, nil, 5)
	expectRun(t, rta, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1`, nil, 3)
	expectRun(t, rta, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1 }()`, nil, 5)
	expectRun(t, rta, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1 }()`, nil, 3)

	v := rta.MustNewRecordValue(nil, false)
	expectRun(t, rta, fmt.Sprintf(`out = {} == %s`, v.String(rta)), nil, true)
	v = rta.MustNewRecordValue(nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = {} == %s`, v.String(rta)), nil, true)

	v = rta.MustNewRecordValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": rta.MustNewStringValue("3"),
	}, false)
	expectRun(t, rta, fmt.Sprintf(`out = {a: 1, b: undefined, c: "3"} == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = {a: 1, b: 2}["b"]`, nil, 2)
	expectRun(t, rta, `out = {a: 1, b: 2}["q"]`, nil, core.Undefined)
	expectRun(t, rta, `out = {a: 1, b: 2}.b`, nil, 2)
	expectRun(t, rta, `out = {a: 1, b: 2}.q`, nil, core.Undefined)
	expectRun(t, rta, `out = "a" in {a: 1, b: 2}`, nil, true)
	expectRun(t, rta, `out = "q" in {a: 1, b: 2}`, nil, false)
	expectRun(t, rta, `out = "q" not in {a: 1, b: 2}`, nil, true)
	expectRun(t, rta, `t := {a: 1, b: 2}; t["a"] = 3; out = t.a`, nil, 3)
	expectRun(t, rta, `t := {a: 1, b: 2}; t.a = 3; out = t["a"]`, nil, 3)
}

func TestDict(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, fmt.Sprintf(`out = dict() == %s`, rta.MustNewDictValue(nil, false).String(rta)), nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = dict() == %s`, rta.MustNewDictValue(nil, true).String(rta)), nil, true)

	expectRun(t, rta, fmt.Sprintf(`out = dict({a: 1, b: undefined, c: "3"}) == %s`, rta.MustNewDictValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": rta.MustNewStringValue("3"),
	}, false).String(rta)), nil, true)

	expectRun(t, rta, `out = dict({a: 1, b: 2})["b"]`, nil, 2)
	expectRun(t, rta, `out = dict({a: 1, b: 2}).record().b`, nil, 2)
	expectRun(t, rta, `out = dict({a: 1, b: 2})["q"]`, nil, core.Undefined)
	expectRun(t, rta, `out = "a" in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, rta, `out = "q" in dict({a: 1, b: 2})`, nil, false)
	expectRun(t, rta, `out = "q" not in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2}); t["a"] = 3; out = t["a"]`, nil, 3)
	expectError(t, rta, `dict({a: 1, b: 2}).q`, nil, "Runtime Error: invalid_selector: type dict has no property q\n\tat test:1:20")

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.is_empty()`, nil, false)
	expectRun(t, rta, `t := dict(); out = t.is_empty()`, nil, true)

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.len()`, nil, 2)
	expectRun(t, rta, `t := dict(); out = t.len()`, nil, 0)

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.values().sort()`, nil, ARR{1, 2})

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.filter(k => k != "b").keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.filter((k, v) => v > 1).keys().sort()`, nil, ARR{"b", "c"})
	expectRun(t, rta, `t := dict({a: 1, b: undefined, c: 3, d: undefined}); out = t.filter().keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, rta, `t := dict(); out = t.filter().len()`, nil, 0)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.count(k => k != "b")`, nil, 2)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.count((k, v) => v > 1)`, nil, 2)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "b")`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "q")`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 1)`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 0)`, nil, true)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "b")`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "q")`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 1)`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 10)`, nil, false)

	expectRun(t, rta, `
out = 0
d = dict({a: 1, b: 2, c: 3})
ignored = d.for_each(func(k) {
	out += d[k]
	return true
})
`, nil, 6)

	expectRun(t, rta, `
items = []
ignored = dict({a: 1, b: 2}).for_each(func(k, v) {
	items = append(items, k + v.string())
	return true
})
out = items.sort()
`, nil, ARR{"a1", "b2"})

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "b")`, nil, "b")
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "q")`, nil, core.Undefined)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 2)`, nil, "b")
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `t := dict(); out = t.find(k => true)`, nil, core.Undefined)
	expectError(t, rta, `dict({a: 1}).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `dict({a: 1}).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `dict({a: 1}).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = "a" in dict({a: 1, b: 2, c: 3})`, nil, true)
	expectRun(t, rta, `out = dict({a: 1, b: 2, c: 3}).contains("a")`, nil, true)
	expectRun(t, rta, `out = "q" in dict({a: 1, b: 2, c: 3})`, nil, false)
	expectRun(t, rta, `out = dict({a: 1, b: 2, c: 3}).contains("q")`, nil, false)
	expectRun(t, rta, `out = "q" not in dict({a: 1, b: 2, c: 3})`, nil, true)

	//there is a problem with keys order (it is random) so we cannot test it now
	//expectRun(t, rta, `out = dict({a: 1, b: 2}).format("v")`, nil, `dict({"a": 1, "b": 2})`)
	//expectRun(t, rta, `out = dict({a: 1, b: 2}).format()`, nil, `dict({"a": 1, "b": 2})`)
}

func TestTime(t *testing.T) {
	rta := core.NewArena(nil)

	o := rta.MustNewTimeValue(time.Date(2020, 6, 20, 1, 2, 3, 4, time.UTC))
	s, _ := o.AsString(rta)
	require.Equal(t, rta, "2020-06-20 01:02:03.000000004 +0000 UTC", s)
	require.Equal(t, rta, `time("2020-06-20T01:02:03.000000004Z")`, o.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = time("2020-06-20 01:02:03.000000004 UTC") == %s`, o.String(rta)), nil, true)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").year()`, nil, 2020)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").month()`, nil, 6)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").day()`, nil, 20)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").hour()`, nil, 1)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").minute()`, nil, 2)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").second()`, nil, 3)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").nanosecond()`, nil, 4)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").unix()`, nil, 1592614923)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_nano()`, nil, 1592614923000000004)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day()`, nil, 6)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day_name()`, nil, "Saturday")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").month_name()`, nil, "June")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").year_day()`, nil, 172) // June 20 is the 172nd day of the year (173rd in leap years)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_date()`, nil, "2020-06-20")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_time()`, nil, "01:02:03")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_datetime()`, nil, "2020-06-20 01:02:03")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").utc().string()`, nil, "2020-06-19 23:02:03.000000004 +0000 UTC")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").zone_offset()`, nil, 7200)

	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").string()`, nil, "2020-06-20 01:02:03.000000004 +0200 +0200")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").int().time().utc().string()`, nil, "2020-06-19 23:02:03 +0000 UTC")

	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format()`, nil, "2020-06-20T01:02:03+02:00")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format("v")`, nil, `time("2020-06-20T01:02:03.000000004+02:00")`)
}

func TestBytes(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = bytes("Hello World!")`, nil, []byte("Hello World!"))
	expectRun(t, rta, `out = bytes("Hello") + bytes(" ") + bytes("World!")`, nil, []byte("Hello World!"))

	// bytes[] -> byte
	expectRun(t, rta, `out = bytes("abcde")[0]`, nil, byte(97))
	expectRun(t, rta, `out = bytes("abcde")[1]`, nil, byte(98))
	expectRun(t, rta, `out = bytes("abcde")[4]`, nil, byte(101))
	expectRun(t, rta, `out = bytes("abcde")[-1]`, nil, byte(101))
	expectRun(t, rta, `out = bytes("abcde")[-2]`, nil, byte(100))
	expectError(t, rta, `out = bytes("abcde")[-6]`, nil, "index_out_of_bounds")
	expectError(t, rta, `out = bytes("abcde")[10]`, nil, "index_out_of_bounds")

	// bytes[a:b] -> bytes
	expectRun(t, rta, `out = bytes("abcde")[1:4]`, nil, []byte("bcd"))
	expectRun(t, rta, `out = bytes("abcde")[:-1]`, nil, []byte("abcd"))
	expectRun(t, rta, `out = bytes("abcde")[1:-1]`, nil, []byte("bcd"))
	expectRun(t, rta, `out = bytes("abcde")[-2:]`, nil, []byte("de"))
	expectRun(t, rta, `out = bytes("abcde")[-3:-1]`, nil, []byte("cd"))
	expectRun(t, rta, `out = bytes("abcde")[3:1]`, nil, []byte{})
	expectRun(t, rta, `out = bytes("abcde")[10:20]`, nil, []byte{})
	expectRun(t, rta, `out = bytes("abcde")[1:5:2]`, nil, []byte("bd"))
	expectRun(t, rta, `out = bytes("abcde")[1:5:-1]`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("abcde")[4:0:-1]`, nil, []byte("edcb"))
	expectRun(t, rta, `out = bytes("abcde")[0:5:2]`, nil, []byte("ace"))
	expectRun(t, rta, `out = bytes("abcde")[::-1]`, nil, []byte("edcba"))
	expectError(t, rta, `out = bytes("abcde")[::0]`, nil, "step cannot be zero")

	o := rta.MustNewBytesValue([]byte("Hello World!"), false)
	s, _ := o.AsString(rta)
	require.Equal(t, rta, "Hello World!", s)
	require.Equal(t, rta, "bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33])", o.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33]) == %s`, o.String(rta)), nil, true)

	v := rta.MustNewBytesValue([]byte("hello"), false)
	expectRun(t, rta, fmt.Sprintf(`out = bytes("hello") == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = bytes("abcde").len()`, nil, 5)
	expectRun(t, rta, `out = bytes("abcde").is_empty()`, nil, false)
	expectRun(t, rta, `out = bytes().is_empty()`, nil, true)
	expectRun(t, rta, `out = bytes("abcde").first()`, nil, byte(97))
	expectRun(t, rta, `out = bytes("abcde").last()`, nil, byte(101))

	expectRun(t, rta, `out = bytes("abc").array()`, nil, ARR{97, 98, 99})
	expectRun(t, rta, `out = bytes("abc").record()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, rta, `out = bytes("abc").dict()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, rta, `out = bytes("abc").string()`, nil, "abc")
	expectRun(t, rta, `out = "abc".bytes().array().string()`, nil, "abc")
	expectRun(t, rta, `out = bytes("abc").format()`, nil, "abc")
	expectRun(t, rta, `out = bytes("abc").format("v")`, nil, "bytes([97, 98, 99])")

	expectRun(t, rta, `out = 98 in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("abc").contains(98)`, nil, true)
	expectRun(t, rta, `out = 255 in bytes("abc")`, nil, false)
	expectRun(t, rta, `out = bytes("abc").contains(255)`, nil, false)
	expectRun(t, rta, `out = bytes("bc") in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("abc").contains(bytes("bc"))`, nil, true)
	expectRun(t, rta, `out = bytes("bd") in bytes("abc")`, nil, false)
	expectRun(t, rta, `out = bytes("abc").contains(bytes("bd"))`, nil, false)
	expectRun(t, rta, `out = bytes("bd") not in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("hello").sort()`, nil, []byte("ehllo"))
	expectRun(t, rta, `out = bytes("").dedup()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("a").dedup()`, nil, []byte("a"))
	expectRun(t, rta, `out = bytes("aabbccd").dedup()`, nil, []byte("abcd"))
	expectRun(t, rta, `out = bytes("abc").dedup()`, nil, []byte("abc"))
	expectRun(t, rta, `out = bytes("abab").dedup()`, nil, []byte("abab"))
	expectRun(t, rta, `out = bytes("hello").sort().dedup()`, nil, []byte("ehlo"))
	expectRun(t, rta, `out = bytes([1, 1, 2, 2, 3]).dedup()`, nil, []byte{1, 2, 3})
	expectRun(t, rta, `out = bytes("").unique()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("abc").unique()`, nil, []byte("abc"))
	expectRun(t, rta, `out = bytes("hello").unique()`, nil, []byte("helo"))
	expectRun(t, rta, `out = bytes("abab").unique()`, nil, []byte("ab"))
	expectRun(t, rta, `out = bytes([3, 1, 2, 1, 3, 2]).unique()`, nil, []byte{3, 1, 2})
	expectRun(t, rta, `out = bytes("").reverse()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("hello").reverse()`, nil, []byte("olleh"))
	expectRun(t, rta, `out = bytes([1, 2, 3]).reverse()`, nil, []byte{3, 2, 1})
	expectRun(t, rta, `out = bytes("").chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = bytes("hello").chunk(2)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, rta, `out = bytes("hello").chunk(2, true)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, rta, `out = bytes("hello").chunk(10)`, nil, ARR{[]byte("hello")})
	expectRun(t, rta, `out = bytes("hello").filter(x => x > 'e')`, nil, []byte("hllo"))
	expectRun(t, rta, `out = bytes("hello").filter((i, x) => i > 2)`, nil, []byte("lo"))
	expectRun(t, rta, `out = bytes("hello").count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = bytes("hello").count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = bytes("hello").all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = bytes("hello").all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = bytes("hello").any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = bytes("hello").find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("hello").find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = bytes("hello").find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("").find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = bytes("x").find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = bytes("x").find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = bytes("x").find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `out = bytes("hello").min()`, nil, byte('e'))
	expectRun(t, rta, `out = bytes("hello").max()`, nil, byte('o'))
	expectRun(t, rta, `
out = 0
ignored := bytes("abc").for_each(func(b) {
	out += b
	return b < 'b'
})
`, nil, 195)
	expectRun(t, rta, `
items := []
ignored := bytes("ABC").for_each(func(i, b) {
	items = append(items, i, b)
	return true
})
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
	expectRun(t, rta, `
items := []
for i, b in bytes("ABC") {
	items = append(items, i, b)
}
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
}

func TestBytesMutability(t *testing.T) {
	rta := core.NewArena(nil)

	// index assignment
	expectRun(t, rta, `b := bytes("hello"); b[0] = 'H'; out = b`, nil, []byte("Hello"))
	expectRun(t, rta, `b := bytes("hello"); b[-2] = '!'; out = b`, nil, []byte("hel!o"))
	expectRun(t, rta, `b := bytes("abc"); b[0] = 65; out = b`, nil, []byte("Abc"))

	// append
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c'); out = b2`, nil, []byte("abc"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c', 'd'); out = b2`, nil, []byte("abcd"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, bytes("cd")); out = b2`, nil, []byte("abcd"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 99); out = b2`, nil, []byte("abc"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c'); out = b`, nil, []byte("ab"))

	// sum / avg / map / reduce
	expectRun(t, rta, `out = bytes("abc").sum()`, nil, 97+98+99)
	expectRun(t, rta, `out = bytes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, rta, `out = bytes().sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes().avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("abc").map(func(b) { return b + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, rta, `out = bytes("abc").map(func(i, b) { return [i, b] })`, nil,
		ARR{ARR{0, byte('a')}, ARR{1, byte('b')}, ARR{2, byte('c')}})
	expectRun(t, rta, `out = bytes("abc").reduce(0, func(acc, b) { return acc + b })`, nil, 97+98+99)
	expectRun(t, rta, `out = bytes("abc").reduce("", func(acc, i, b) { return acc + i.string() + b.string() })`, nil, "097198299")

	// type names
	expectRun(t, rta, `out = type_name(bytes("abc"))`, nil, "bytes")
	expectRun(t, rta, `out = type_name(immutable(bytes("abc")))`, nil, "immutable-bytes")

	// immutable rejects writes
	expectError(t, rta, `b := immutable(bytes("abc")); b[0] = 'X'`, nil, "not_assignable: type immutable-bytes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, rta, `out = type_name(immutable(bytes("abcd"))[1:3])`, nil, "immutable-bytes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, rta, `out = type_name(immutable(bytes("abcd"))[::-1])`, nil, "bytes")
	// slice of mutable stays mutable
	expectRun(t, rta, `out = type_name(bytes("abcd")[1:3])`, nil, "bytes")

	// copy of immutable yields mutable
	expectRun(t, rta, `b := immutable(bytes("abc")); c := copy(b); c[0] = 'X'; out = c`, nil, []byte("Xbc"))

	// append on immutable returns fresh mutable (does not mutate source)
	expectRun(t, rta, `b := immutable(bytes("ab")); b2 := append(b, 'c'); b2[0] = 'X'; out = b2`, nil, []byte("Xbc"))
	expectRun(t, rta, `b := immutable(bytes("ab")); b2 := append(b, 'c'); out = type_name(b2)`, nil, "bytes")

	// invalid assignment values
	expectError(t, rta, `b := bytes("abc"); b[0] = "xy"`, nil,
		"invalid_index_type: (index assign value) expected byte, got string")
	expectError(t, rta, `b := bytes("abc"); b[0] = 256`, nil,
		"invalid_index_type: (index assign value) expected byte, got int")
	expectError(t, rta, `b := bytes("abc"); b[10] = 'X'`, nil,
		"index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestArrayIterator(t *testing.T) {
	rta := core.NewArena(nil)

	expectRun(t, rta, `
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

	expectRun(t, rta, `
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
	rta := core.NewArena(nil)

	expectRun(t, rta, `
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

	expectRun(t, rta, `
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
	rta := core.NewArena(nil)

	expectRun(t, rta, `
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

	expectRun(t, rta, `
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
	rta := core.NewArena(nil)

	expectRun(t, rta, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, rta, `
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
	rta := core.NewArena(nil)

	expectRun(t, rta, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, rta, `
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
	rta := core.NewArena(nil)

	expectRun(t, rta, `out = range(97, 103, 1).bytes().string()`, nil, "abcdef")
	expectRun(t, rta, `out = range(103, 97, 1).bytes().string()`, nil, "gfedcb")
	expectRun(t, rta, `out = range(97, 103, 1).string()`, nil, "abcdef")
	expectRun(t, rta, `out = range(103, 97, 1).string()`, nil, "gfedcb")
	expectRun(t, rta, `out = range(1, 3, 1).record()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, rta, `out = range(1, 3, 1).dict()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, rta, `
out = 0
ignored := range(1, 5, 1).for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)
	expectRun(t, rta, `
out = 0
ignored := range(10, 13, 1).for_each(func(i, v) {
	out += i + v
	return true
})
`, nil, 36)

	expectRun(t, rta, `out = range(10, 20, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, rta, `out = range(10, 20, 1).find(v => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = range(10, 20, 1).find((i, v) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = range(20, 10, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, rta, `out = range(0, 0, 1).find(v => true)`, nil, core.Undefined)
	expectError(t, rta, `out = range(0, 5, 1).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = range(0, 5, 1).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = range(0, 5, 1).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `r := range(0, 10, 1); out = r.len()`, nil, 10)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.len()`, nil, 5)
	expectRun(t, rta, `r := range(0, 10, 3); out = r.len()`, nil, 4)
	expectRun(t, rta, `r := range(0, 10, 4); out = r.len()`, nil, 3)
	expectRun(t, rta, `r := range(0, 10, 5); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 6); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 7); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 8); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 9); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 10); out = r.len()`, nil, 1)
	expectRun(t, rta, `r := range(0, 10, 11); out = r.len()`, nil, 1)
	expectRun(t, rta, `r := range(0, 10, 100); out = r.len()`, nil, 1)

	expectRun(t, rta, `r := range(0, 100, 1); out = len(r)`, nil, 100)
	expectRun(t, rta, `r := range(0, 100, 2); out = len(r)`, nil, 50)
	expectRun(t, rta, `r := range(0, 100, 3); out = len(r)`, nil, 34)
	expectRun(t, rta, `r := range(0, 100, 5); out = len(r)`, nil, 20)
	expectRun(t, rta, `r := range(0, 100, 10); out = len(r)`, nil, 10)

	expectRun(t, rta, `r := range(0, 100, 1); out = r.len()`, nil, 100)
	expectRun(t, rta, `r := range(0, 100, 2); out = r.len()`, nil, 50)
	expectRun(t, rta, `r := range(0, 100, 3); out = r.len()`, nil, 34)
	expectRun(t, rta, `r := range(0, 100, 5); out = r.len()`, nil, 20)
	expectRun(t, rta, `r := range(0, 100, 10); out = r.len()`, nil, 10)

	expectRun(t, rta, `r := range(100, 0, 1); out = len(r)`, nil, 100)
	expectRun(t, rta, `r := range(100, 0, 2); out = len(r)`, nil, 50)
	expectRun(t, rta, `r := range(100, 0, 3); out = len(r)`, nil, 34)
	expectRun(t, rta, `r := range(100, 0, 5); out = len(r)`, nil, 20)
	expectRun(t, rta, `r := range(100, 0, 10); out = len(r)`, nil, 10)

	expectRun(t, rta, `r := range(100, 0, 1); out = r.len()`, nil, 100)
	expectRun(t, rta, `r := range(100, 0, 2); out = r.len()`, nil, 50)
	expectRun(t, rta, `r := range(100, 0, 3); out = r.len()`, nil, 34)
	expectRun(t, rta, `r := range(100, 0, 5); out = r.len()`, nil, 20)
	expectRun(t, rta, `r := range(100, 0, 10); out = r.len()`, nil, 10)

	expectRun(t, rta, `r := range(0, 5, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4})
	expectRun(t, rta, `r := range(5, 0, 1); out = r.array()`, nil, ARR{5, 4, 3, 2, 1})
	expectRun(t, rta, `r := range(-5, 5, 1); out = r.array()`, nil, ARR{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4})

	expectRun(t, rta, `r := range(0, 10, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	expectRun(t, rta, `r := range(0, 10, 2); out = r.array()`, nil, ARR{0, 2, 4, 6, 8})
	expectRun(t, rta, `r := range(0, 10, 3); out = r.array()`, nil, ARR{0, 3, 6, 9})
	expectRun(t, rta, `r := range(0, 10, 4); out = r.array()`, nil, ARR{0, 4, 8})
	expectRun(t, rta, `r := range(0, 10, 5); out = r.array()`, nil, ARR{0, 5})

	expectRun(t, rta, `r := range(10, 0, 1); out = r.array()`, nil, ARR{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	expectRun(t, rta, `r := range(10, 0, 2); out = r.array()`, nil, ARR{10, 8, 6, 4, 2})
	expectRun(t, rta, `r := range(10, 0, 3); out = r.array()`, nil, ARR{10, 7, 4, 1})
	expectRun(t, rta, `r := range(10, 0, 4); out = r.array()`, nil, ARR{10, 6, 2})
	expectRun(t, rta, `r := range(10, 0, 5); out = r.array()`, nil, ARR{10, 5})

	expectRun(t, rta, `r := range(0, 100, 1); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[1]`, nil, 1)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[2]`, nil, 2)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[3]`, nil, 3)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[10]`, nil, 10)

	expectRun(t, rta, `r := range(0, 100, 2); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[1]`, nil, 2)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[2]`, nil, 4)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[3]`, nil, 6)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[10]`, nil, 20)

	expectRun(t, rta, `r := range(0, 100, 3); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[1]`, nil, 3)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[2]`, nil, 6)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[3]`, nil, 9)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[10]`, nil, 30)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[-1]`, nil, 99)
	expectRun(t, rta, `r := range(10, 0, 2); out = r[-1]`, nil, 2)
	expectError(t, rta, `r := range(0, 100, 3); out = r[-35]`, nil, "index_out_of_bounds")
	expectError(t, rta, `r := range(0, 100, 3); out = r[34]`, nil, "index_out_of_bounds")

	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(0)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(5)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(10)`, nil, false)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(0)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(1)`, nil, false)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(2)`, nil, true)

	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(0)`, nil, false)
	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(5)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(10)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(10)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(9)`, nil, false)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(8)`, nil, true)
	expectRun(t, rta, `out = 11 not in range(0, 10, 1)`, nil, true)

	expectRun(t, rta, `
out = 0
for e in range(1, 10, 1) {
	out += e
}
`, nil, 45)

	expectRun(t, rta, `
out = 0
for i, e in range(1, 10, 1) {
	out += i
}
`, nil, 36)

	expectRun(t, rta, `
out = 0
for e in range(1, 10, 2) {
	out += e
}
`, nil, 25)

	expectRun(t, rta, `
out = 0
for i, e in range(1, 10, 2) {
	out += i
}
`, nil, 10)

	expectRun(t, rta, `
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

	expectRun(t, rta, `
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
