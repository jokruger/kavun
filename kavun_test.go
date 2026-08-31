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
	"github.com/jokruger/kavun/ast"
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
	file *ast.File,
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
	c := compiler.NewCompiler(compiler.O3(), nil, file.InputFile, symTable, nil, customModules, tr)
	var node ast.Node
	node, err = c.Optimize(file)
	if err != nil {
		return
	}
	err = c.CompileNode(node)
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

func parse(t *testing.T, input string) *ast.File {
	testFileSet := ast.NewFileSet()
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
	expectError(t, `out = undefined == float([])`, nil, "conversion: cannot convert array to float") // no edge — raises even inside ==
	expectRun(t, `out = undefined.format("v")`, nil, "undefined")

	// undefined propagates through every operator except ==/!= — "we don't know" contaminates
	// arithmetic, bitwise, ordering, and unary alike, regardless of which side it's on.
	expectRun(t, `out = undefined + 1`, nil, core.Undefined)
	expectRun(t, `out = 1 + undefined`, nil, core.Undefined)
	expectRun(t, `out = undefined + "foo"`, nil, core.Undefined)
	expectRun(t, `out = undefined < 1`, nil, core.Undefined)
	expectRun(t, `out = 1 < undefined`, nil, core.Undefined)
	expectRun(t, `out = -undefined`, nil, core.Undefined)
	expectRun(t, `out = ^undefined`, nil, core.Undefined)
	expectRun(t, `out = !undefined`, nil, true)

	// the ACTION plane raises — iteration is an action, while the chained reads above propagate
	expectError(t, `for x in undefined {}`, nil, "not_iterable")
	expectRun(t, `out = is_iterable(undefined)`, nil, false)
	expectRun(t, `out = is_immutable(undefined)`, nil, true) // true unless the value can be mutated — exceptionless

	// the maybe-missing rescue: undefined carries every conversion member, default-mandatory
	expectRun(t, `out = undefined.array([])`, nil, ARR{})
	expectRun(t, `out = undefined.dict(dict({a: 1})).keys()`, nil, ARR{"a"})
	expectRun(t, `out = undefined.record({a: 1}).a`, nil, 1)
	expectRun(t, `out = undefined.byte(b'\x07')`, nil, byte(7))
	expectRun(t, `out = undefined.runes(u"x").string()`, nil, "x")
	expectRun(t, `out = undefined.decimal(decimal("1.5")) == decimal("1.5")`, nil, true)
	expectRun(t, `out = undefined.time(time(5)) == time(5)`, nil, true)
	expectError(t, `undefined.int()`, nil, "value is missing")

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
	// bool arithmetic is deferred entirely, same-type included — only ordering is defined.
	expectError(t, `out = 5 + true`, nil, "invalid_binary_operator: int + bool")
	expectError(t, `out = 5 + true; 5`, nil, "invalid_binary_operator: int + bool")

	expectError(t, `-true`, nil, "invalid_unary_operator: - bool")
	expectError(t, `true + false`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `5; true + false; 5`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, `if (10 > 1) { true + false; }`, nil, "invalid_binary_operator: bool + bool")

	// bool ordering: false < true, the natural 0/1 convention — motivates sortable []bool.
	expectRun(t, `out = false < true`, nil, true)
	expectRun(t, `out = true < false`, nil, false)
	expectRun(t, `out = false <= false`, nil, true)
	expectRun(t, `out = true > false`, nil, true)
	expectRun(t, `out = true >= true`, nil, true)

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
	expectError(t, `out = true.byte()`, nil, "invalid_method: type bool has no method byte") // int is the gateway
	expectRun(t, `out = true.int().byte()`, nil, byte(1))
	expectRun(t, `out = false.int().byte()`, nil, byte(0))
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
	expectError(t, `out = byte(true)`, nil, "conversion: cannot convert bool to byte: no conversion exists")
	expectRun(t, `out = byte(true.int())`, nil, byte(1)) // int is the gateway
	expectRun(t, `out = byte('A')`, nil, byte(65))
	expectError(t, `out = byte("12")`, nil, "conversion: cannot convert string to byte") // text parses into numerics only
	expectRun(t, `out = "12".int().byte()`, nil, byte(12))
	expectError(t, `out = byte(u"12")`, nil, "to byte: no conversion exists") // text parses into numerics only
	expectRun(t, `out = byte(u"12".int())`, nil, byte(12))
	expectError(t, `out = byte(300, byte(7))`, nil, "wrong_num_arguments: (byte) expected 0 or 1 argument(s), got 2") // no free default form
	expectRun(t, `out = (300).byte(byte(7))`, nil, byte(7))                                                           // the member default is the fallible-conversion spelling

	// a byte's canonical TEXT is its symbol, matching .string(): every convert-to-string surface agrees
	expectRun(t, `out = ["x", b'A'].join("-")`, nil, "x-A")
	expectError(t, `out = ["x", b'\xFF'].join("-")`, nil, "cannot convert byte to string")
	expectRun(t, `out = "A" == b'A'`, nil, true) // equality reads the canonical text, like rune
	expectRun(t, `out = b'A' == "A"`, nil, true)
	expectRun(t, `out = "65" == b'A'`, nil, false)
	// a high octet has no text form and equals no text — in either direction
	expectRun(t, `out = "ÿ" == b'\xFF'`, nil, false)
	expectRun(t, `out = b'\xFF' == "ÿ"`, nil, false)
	expectRun(t, `out = b"\xFF" == "ÿ"`, nil, false)
	expectRun(t, `out = b'A' in {A: 1}`, nil, true)
	// display renders stay numeric — they are renders, not conversions
	expectRun(t, `out = f"{b'A'}"`, nil, "65")
	expectRun(t, `out = format(b'A')`, nil, "65")
	expectRun(t, `out = byte(255) + 1`, nil, byte(0))
	expectRun(t, `out = byte(255) + 2`, nil, byte(1))
	expectRun(t, `out = byte(0) - 1`, nil, byte(255))
	// byte owns this pairing symmetrically now (the old "byte never widens" asymmetry is fixed):
	// int declines and byte's ring arithmetic wins regardless of which side byte is on.
	expectRun(t, `out = 1 + byte(255)`, nil, byte(0))
	// byte is a genuine ring — int - byte is symmetric with byte - int (both wrap), unlike the
	// directional rune/time case. Computed via runtime int vars (not constants) so Go's own
	// byte(...) conversion performs the same truncating wraparound being tested, without tripping
	// a compile-time constant-overflow error.
	five, threeHundred, three := 5, 300, 3
	expectRun(t, `out = byte(5) - 300`, nil, byte(five-threeHundred)) // wraps for any magnitude
	expectRun(t, `out = 300 - byte(5)`, nil, byte(threeHundred-five))
	expectRun(t, `out = 3 - byte(5)`, nil, byte(three-five)) // 3-5 = -2, wraps to 254

	// ordering: same-type, plus against int (widens, doesn't truncate)
	expectRun(t, `out = byte(1) < byte(2)`, nil, true)
	expectRun(t, `out = byte(200) < 300`, nil, true) // widens the int side; never truncates byte
	expectRun(t, `out = 300 > byte(200)`, nil, true)

	// byte vs. rune: rune safely accepts byte (rule 2, Latin-1 bijection); + inherits rune+rune's
	// rejection, - becomes a genuine code-point distance, ordering/equality widen the same way.
	expectError(t, `out = byte(65) + 'A'`, nil, "invalid_binary_operator: byte + rune")
	expectError(t, `out = 'A' + byte(65)`, nil, "invalid_binary_operator: rune + byte")
	expectRun(t, `out = byte(65) - 'B'`, nil, -1)
	expectRun(t, `out = 'B' - byte(65)`, nil, 1)
	expectRun(t, `out = byte(65) < 'B'`, nil, true)
	expectRun(t, `out = byte(65) == 'A'`, nil, true)
	expectRun(t, `out = 'A' == byte(65)`, nil, true)

	v = core.ByteValue(0)
	expectRun(t, fmt.Sprintf(`out = byte(0) == %s`, v.String()), nil, true)
	v = core.ByteValue(1)
	expectRun(t, fmt.Sprintf(`out = byte(1) == %s`, v.String()), nil, true)
	v = core.ByteValue(123)
	expectRun(t, fmt.Sprintf(`out = byte(123) == %s`, v.String()), nil, true)

	expectRun(t, `out = byte(123).int()`, nil, 123)
	expectError(t, `out = byte(0).bool()`, nil, "invalid_method: type byte has no method bool") // int is the gateway
	expectRun(t, `out = byte(10).int().bool()`, nil, true)
	expectRun(t, `out = byte(48).rune()`, nil, '0')
	expectError(t, `out = byte(48).float()`, nil, "invalid_method: type byte has no method float") // int is the gateway
	expectRun(t, `out = byte(48).int().float()`, nil, 48.0)
	// .string() is the byte's TEXT CONTENT (its ASCII symbol); the render keeps
	// the digits — the one scalar where the two deliberately disagree
	expectRun(t, `out = byte(48).string()`, nil, "0")
	expectRun(t, `out = byte(65).string()`, nil, "A")
	expectRun(t, `out = byte(65).runes().string()`, nil, "A")
	expectError(t, `out = byte(200).string()`, nil, "conversion: cannot convert byte to string")
	expectRun(t, `out = byte(200).string("?")`, nil, "?")
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

	// rune owns int+rune arithmetic (offset, stays rune) — int declines and reflects to rune's hook.
	expectRun(t, `out = 9 + '0'`, nil, rune(57)) // '0' is 48 in ASCII; 9 + 48 = 57 = '9'
	expectRun(t, `out = '9' - 5`, nil, rune(52)) // '9' is 57 in ASCII; 57 - 5 = 52 = '4'

	v = core.IntValue(0)
	expectRun(t, fmt.Sprintf(`out = 0 == %s`, v.String()), nil, true)
	v = core.IntValue(1)
	expectRun(t, fmt.Sprintf(`out = 1 == %s`, v.String()), nil, true)
	v = core.IntValue(1234567890)
	expectRun(t, fmt.Sprintf(`out = 1234567890 == %s`, v.String()), nil, true)

	// int has no relationship with string at all — no implicit numeric-string parsing via '+'.
	expectError(t, `out = 5 + "-5"`, nil, "invalid_binary_operator: int + string")
	expectError(t, `out = 5 + "5"`, nil, "invalid_binary_operator: int + string")

	expectRun(t, `out = (12).int()`, nil, 12)
	expectRun(t, `out = (0).bool()`, nil, false)
	expectRun(t, `out = (10).bool()`, nil, true)
	expectRun(t, `out = (48).rune()`, nil, '0')
	expectRun(t, `out = (48).float()`, nil, 48.0)
	expectRun(t, `out = (48).string()`, nil, "48")
	expectRun(t, `out = (1234567890).time().utc().string()`, nil, "2009-02-13T23:31:30Z")
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

	// The DISPLAY form of a whole float keeps its point, so it never reads back as an int; the text
	// CONVERSION stays bare, because 3 == 3.0 and equal values must key and join alike.
	expectRun(t, `out = format("{0}", [[3.0, 3]])`, nil, "[3.0, 3]")
	expectRun(t, `out = format("{0}", [[0.0, -0.0, 2.5]])`, nil, "[0.0, -0.0, 2.5]")
	expectRun(t, `out = (3.0).string()`, nil, "3")
	expectRun(t, `out = string(3.0)`, nil, "3")
	expectRun(t, `out = f"{3.0}"`, nil, "3")
	expectRun(t, `d := dict(); d[3] = "i"; d[3.0] = "f"; out = d.len()`, nil, 1) // one key: 3 == 3.0

	// float has no relationship with string at all — no implicit numeric-string parsing via '+'.
	expectError(t, `out = 5.0 + "-5.0"`, nil, "invalid_binary_operator: float + string")
	expectError(t, `out = 5.0 + "5.0"`, nil, "invalid_binary_operator: float + string")

	expectRun(t, `out = (1.5).float()`, nil, 1.5)
	expectRun(t, `out = (1.5).int()`, nil, 1)
	expectRun(t, `out = (1.5).string()`, nil, "1.5")

	// f-suffix float literals
	expectRun(t, `out = 1f`, nil, 1.0)
	expectRun(t, `out = 1.5f`, nil, 1.5)
	expectRun(t, `out = type_name(1f)`, nil, "float")
	expectRun(t, `out = type_name(1.5f)`, nil, "float")
	expectRun(t, `out = 2f + 3f`, nil, 5.0)

	// truthiness is inequality with the type's zero, so 0.0 is falsy like 0 and
	// decimal(0); previously 0.0 was truthy, the one type disagreeing with the rule
	expectRun(t, `out = !!0.0`, nil, false)
	expectRun(t, `out = !!(-0.0)`, nil, false)
	expectRun(t, `out = !!0.5`, nil, true)
	expectRun(t, `if 0.0 { out = "t" } else { out = "f" }`, nil, "f")
	// NaN is an error state: a boolean context raises instead of answering
	expectError(t, `x := float("nan"); if x { out = 1 }`, nil, "invalid_value")
	expectError(t, `out = !!float("nan")`, nil, "invalid_value")
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

	// float and decimal are declared incompatible — no automatic winner between the two
	// representations, a vm error in both directions. This is the original bug that motivated the
	// whole type/operator redesign: 0.1 + 2.5d used to silently produce float 2.6, dropping
	// decimal's exactness.
	expectError(t, `out = 1.0 + decimal(2)`, nil, "invalid_binary_operator: float + decimal")
	expectError(t, `out = decimal(1) + 2.0`, nil, "invalid_binary_operator: decimal + float")

	// regression: error_details() on a VALID decimal dereferenced a nil
	// ErrorDetails() and panicked the Go host; the honest answer is undefined
	expectRun(t, `out = decimal("1.23").error_details()`, nil, core.Undefined)
	expectRun(t, `out = decimal(123).error_details()`, nil, core.Undefined)

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

	// rune + rune is deliberately undefined — "the sum of two code points" corresponds to nothing
	// real (unlike byte + byte's wraparound-counter reading, since rune isn't a ring).
	expectError(t, `out = '0' + '9'`, nil, "invalid_binary_operator: rune + rune")
	// rune owns rune + int / rune - int (offset, stays rune) — int declines and reflects here.
	expectRun(t, `out = '0' + 9`, nil, rune(57)) // '0' is 48 in ASCII; 48 + 9 = 57 = '9'
	expectRun(t, `out = '9' - 4`, nil, rune(53)) // '9' is 57 in ASCII; 57 - 4 = 53 = '5'
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

	expectRun(t, `out = '4' + 4`, nil, rune(56)) // '4' is 52 in ASCII; 52 + 4 = 56 = '8'
	expectRun(t, `out = '4' + "4"`, nil, "44")
	expectError(t, `'4' - "4"`, nil, "invalid_binary_operator: rune - string")

	expectRun(t, `out = '4'.rune()`, nil, '4')
	expectError(t, `out = '4'.bool()`, nil, "invalid_method: type rune has no method bool") // int is the gateway
	expectRun(t, `out = '4'.int().bool()`, nil, true)
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

	// index operator — s[i] yields the i-th SYMBOL as a rune; the result type
	// is rune even on ASCII, never a byte
	str := "abcdef"
	strStr := `"abcdef"`
	strLen := 6
	for idx := range strLen {
		expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, idx), nil, rune(str[idx]))
		expectRun(t, fmt.Sprintf("out = %s[0 + %d]", strStr, idx), nil, rune(str[idx]))
		expectRun(t, fmt.Sprintf("out = %s[1 + %d - 1]", strStr, idx), nil, rune(str[idx]))
		expectRun(t, fmt.Sprintf("idx = %d; out = %s[idx]", idx, strStr), nil, rune(str[idx]))
		expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, -idx-1), nil, rune(str[strLen-idx-1]))
	}

	expectError(t, fmt.Sprintf("%s[%d]", strStr, -strLen-1), nil, "index_out_of_bounds")
	expectError(t, fmt.Sprintf("%s[%d]", strStr, strLen), nil, "index_out_of_bounds")
	expectRun(t, fmt.Sprintf("out = %s[%d]", strStr, -2), nil, rune(str[strLen-2]))

	// multibyte: indexing counts symbols, not bytes
	expectRun(t, `out = "héllo"[1]`, nil, 'é')
	expectRun(t, `out = "héllo"[2]`, nil, 'l')
	expectRun(t, `out = "héllo"[-1]`, nil, 'o')
	expectRun(t, `out = "héllo".len()`, nil, 5)
	expectRun(t, `out = len("héllo")`, nil, 5)
	expectError(t, `out = "héllo"[5]`, nil, "index_out_of_bounds")

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

	// No implicit stringification: '+' never silently converts an unrelated type to string, even
	// when string is lhs — this is the exact typo'd-'+' footgun the type/operator redesign rejects.
	// 'rune' is the one legitimate scalar exception (a rune is genuinely a piece of text, not an
	// unrelated type being coerced) — see docs/types.md's "Sequence/text" section.
	expectError(t, `out = "foo" + 1`, nil, "invalid_binary_operator: string + int")
	expectError(t, `out = "foo" + 1.0`, nil, "invalid_binary_operator: string + float")
	expectError(t, `out = "foo" + 1.5`, nil, "invalid_binary_operator: string + float")
	expectError(t, `out = "foo" + true`, nil, "invalid_binary_operator: string + bool")
	expectRun(t, `out = "foo" + 'X'`, nil, "fooX")
	expectError(t, `out = "foo" + error(5)`, nil, "invalid_binary_operator: string + error")
	// string has no reading for an array, so it hands the operation over and the array prepends
	// it as one element — the mirror of [100, 101] + "foo"
	expectRun(t, `out = "foo" + [100, 101]`, nil, ARR{"foo", 100, 101})
	// also works with "+=" operator
	expectError(t, `out = "foo"; out += 1.5`, nil, "invalid_binary_operator: string + float")

	// int + string is a vm error in both directions — no implicit stringification either way
	expectError(t, `1 + "foo"`, nil, "invalid_binary_operator: int + string")

	// '-' now exists for string: same-type removes every occurrence of the substring (native
	// substring removal, non-mutating) — new capability, not previously supported.
	expectRun(t, `out = "foo" - "bar"`, nil, "foo") // "bar" isn't a substring of "foo"
	expectRun(t, `out = "foobar" - "bar"`, nil, "foo")
	expectRun(t, `out = "foofoofoo" - "foo"`, nil, "")
	expectRun(t, `out = "foo" - 'o'`, nil, "f")
	expectRun(t, `out = "foobar" - runes("oa")`, nil, "foobar") // "oa" isn't a contiguous substring
	expectRun(t, `out = "banana" - runes("an")`, nil, "ba")
	// `-`'s acceptance equals `+`'s: octets are symbols in ASCII, bytes decode as UTF-8
	expectRun(t, `out = "foo" - b'o'`, nil, "f")
	expectRun(t, `out = "foo" - bytes("f")`, nil, "oo")
	expectError(t, `"foo" - byte(200)`, nil, "invalid_value") // a non-ASCII octet is no symbol
	// '-' is lhs-only, no reflected direction — rune - string was never defined (mirrors '+',
	// which also doesn't define rune "owning" a pairing with string for removal).
	expectError(t, `out = 'f' - "foo"`, nil, "invalid_binary_operator: rune - string")

	// undefined propagates through everything except ==/!=, string concatenation included
	expectRun(t, `out = "foo" + undefined`, nil, core.Undefined)

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
	expectRun(t, `out = "abcd".trim('a', 'd')`, nil, "bc")            // the set form is variadic ELEMENTS
	expectError(t, `"abcd".trim("ad")`, nil, "invalid_argument_type") // a run points at remove_prefix/remove_suffix
	expectRun(t, `out = "".reverse()`, nil, "")
	expectRun(t, `out = "a".reverse()`, nil, "a")
	expectRun(t, `out = "hello".reverse()`, nil, "olleh")
	expectRun(t, `out = "їЇґҐ".reverse()`, nil, "ҐґЇї")
	expectRun(t, `out = "こんにちは".reverse()`, nil, "はちにんこ")

	expectRun(t, `out = "abc".string()`, nil, "abc")
	expectRun(t, `out = "abc".array()`, nil, ARR{int64('a'), int64('b'), int64('c')})
	expectRun(t, `out = "abc".array().string()`, nil, "abc")

	// regression: a multibyte string with TRAILING content used to panic the
	// Go host — the byte offset was used as a rune index, so the test input
	// must be multibyte-then-more ("hé" alone cannot reproduce it)
	expectRun(t, `out = "héllo".array()`, nil, ARR{int64('h'), int64('é'), int64('l'), int64('l'), int64('o')})
	expectRun(t, `out = "héllo".array().string()`, nil, "héllo")
	expectRun(t, `out = "héllo".runes().string()`, nil, "héllo")
	expectRun(t, `out = "héllo".runes().len()`, nil, 5)
	// same defect class in the map conversions: keys must be rune ordinals,
	// not byte offsets ("héllo" byte offsets are 0,1,3,4,5 — key "4" was 'l')
	expectError(t, `out = "héllo".dict()`, nil, "invalid_method: type string has no method dict") // elements are never entries
	expectError(t, `out = "héllo".record()`, nil, "invalid_method: type string has no method record")
	expectRun(t, `out = "true".bool()`, nil, true)
	expectRun(t, `out = "false".bool()`, nil, false)
	expectError(t, `out = "abc".bool()`, nil, "conversion: cannot convert string to bool")
	expectRun(t, `out = "abc".bool(false)`, nil, false)
	expectRun(t, `out = "true".bool().string()`, nil, "true")
	expectRun(t, `out = "abc".bytes()`, nil, core.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, `out = "abc".bytes().string()`, nil, "abc")
	expectRun(t, `out = "1.2".float()`, nil, 1.2)
	expectRun(t, `out = "1.2".float().string()`, nil, "1.2")
	expectError(t, `out = "12".byte()`, nil, "invalid_method: type string has no method byte") // text parses into numerics only
	expectRun(t, `out = "12".int().byte()`, nil, byte(12))
	expectRun(t, `out = "12".int()`, nil, 12)
	expectRun(t, `out = "12".float().string()`, nil, "12")
	expectError(t, `out = "abc".int()`, nil, "conversion: cannot convert string to int")
	expectRun(t, `out = "abc".int(0)`, nil, 0)
	expectError(t, `out = "abc".record()`, nil, "invalid_method") // elements are never entries
	expectError(t, `out = "abc".dict()`, nil, "invalid_method")
	expectRun(t, `out = "abc".format()`, nil, "abc")
	expectRun(t, `out = "abc".format("v")`, nil, `"abc"`)

	expectRun(t, `out = " їЇґҐ ".trim()`, nil, "їЇґҐ")
	expectRun(t, `out = "їЇґҐ".upper()`, nil, "ЇЇҐҐ")
	expectRun(t, `out = "їЇґҐ".lower()`, nil, "їїґґ")
	expectRun(t, `out = "こんにちはさ"[1]`, nil, 'ん')     // symbol index, never a byte
	expectRun(t, `out = "こんにちはさ"[1:2]`, nil, "ん")   // symbol slice — can never split a rune
	expectRun(t, `out = "こんにちはさ"[0:3]`, nil, "こんに") // symbol slice, not byte slice

	expectRun(t, `out = len("")`, nil, 0)
	expectRun(t, `out = len("hello")`, nil, 5)
	expectRun(t, `out = len("їЇґҐ")`, nil, 4)   // symbol length, never bytes
	expectRun(t, `out = len("こんにちはさ")`, nil, 6) // symbol length; octet count is .bytes().len()

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
	expectRun(t, `out = "hello".index(x => x == 'l')`, nil, 2)
	expectRun(t, `out = "hello".index(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = "hello".index((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = "hello".index((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = "".index(x => true)`, nil, core.Undefined)
	expectRun(t, `out = "x".index()`, nil, 0) // no-arg = first significant (non-blank) symbol
	expectRun(t, `out = "  x".index()`, nil, 2)
	expectRun(t, `out = " ".index()`, nil, core.Undefined) // all blank
	expectRun(t, `out = "x".index('x')`, nil, 0)           // element reading
	expectRun(t, `out = "hello".index("ll")`, nil, 2)      // run reading, symbol offsets
	expectRun(t, `out = "hello".index_last('l')`, nil, 3)
	expectError(t, `out = "x".index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")
	expectRun(t, `
out = ""
ignored := "hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hello") // full pass — the falsy return no longer stops the loop
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

	// the RECEIVER — the left operand — decides the result type; content order still
	// respects which side was written first
	expectRun(t, `out = "World!" + u"Hello "`, nil, "World!Hello ")
	expectRun(t, `out = u"Hello " + "World!"`, nil, []rune("Hello World!"))
	expectRun(t, `out = runes("cd") + bytes("ab")`, nil, []rune("cdab")) // valid UTF-8 decodes
	expectRun(t, `out = "cd" > u"ab"`, nil, true)
	expectRun(t, `out = u"ab" > "cd"`, nil, false)

	// a scalar on the left takes the sequence's type
	expectRun(t, `out = 'A' + runes("bc")`, nil, []rune("Abc"))
	expectRun(t, `out = runes("bc") + 'A'`, nil, []rune("bcA"))
	// an octet is a symbol in ASCII; beyond it, no symbol exists to add
	expectRun(t, `out = b'A' + runes("bc")`, nil, []rune("Abc"))
	expectRun(t, `out = runes("bc") + b'A'`, nil, []rune("bcA"))
	expectError(t, `runes("bc") + byte(200)`, nil, "invalid_value")
	// `-`'s acceptance equals `+`'s
	expectRun(t, `out = u"banana" - "an"`, nil, []rune("ba"))
	expectRun(t, `out = u"foo" - b'o'`, nil, []rune("f"))
	expectRun(t, `out = u"foo" - bytes("f")`, nil, []rune("oo"))

	// removal: an element operand removes every equal symbol, a run every occurrence
	expectRun(t, `out = runes("banana") - 'a'`, nil, []rune("bnn"))
	expectRun(t, `out = runes("banana") - runes("an")`, nil, []rune("ba"))
	expectRun(t, `out = runes("abc") - "b"`, nil, []rune("ac"))

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
	expectRun(t, `out = runes("abcd").trim('a', 'd')`, nil, []rune("bc")) // the set form is variadic ELEMENTS
	expectError(t, `runes("abcd").trim("ad")`, nil, "invalid_argument_type")
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
	expectError(t, `out = runes("abc").bool()`, nil, "conversion: cannot convert runes to bool")
	expectRun(t, `out = runes("abc").bool(false)`, nil, false)
	expectRun(t, `out = runes("true").bool().string()`, nil, "true")
	expectRun(t, `out = runes("abc").bytes()`, nil, core.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, `out = runes("abc").bytes().string()`, nil, "abc")
	expectRun(t, `out = runes("1.2").float()`, nil, 1.2)
	expectRun(t, `out = runes("1.2").float().string()`, nil, "1.2")
	expectRun(t, `out = runes("12").int()`, nil, 12)
	expectRun(t, `out = runes("12").float().string()`, nil, "12")
	expectError(t, `out = runes("abc").int()`, nil, "conversion: cannot convert runes to int")
	expectRun(t, `out = runes("abc").int(0)`, nil, 0)
	expectError(t, `out = runes("abc").record()`, nil, "invalid_method") // elements are never entries
	expectError(t, `out = runes("abc").dict()`, nil, "invalid_method")

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
	expectRun(t, `out = u"hello".index(x => x == 'l')`, nil, 2)
	expectRun(t, `out = u"hello".index(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = u"hello".index((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = u"hello".index((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = u"".index(x => true)`, nil, core.Undefined)
	expectRun(t, `out = u"x".index()`, nil, 0)
	expectRun(t, `out = u"héllo".index(u"ll")`, nil, 2) // run reading, rune offsets
	expectRun(t, `out = u"héllo".index('é')`, nil, 1)   // element reading
	expectError(t, `out = u"x".index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")
	expectRun(t, `out = u"hello".min()`, nil, 'e')
	expectRun(t, `out = u"hello".max()`, nil, 'o')
	expectRun(t, `
out = ""
ignored := u"hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hello") // full pass — the falsy return no longer stops the loop
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
	expectRun(t, `r := runes("ab"); r2 := r.append('c'); out = r2`, nil, []rune("abc"))
	expectRun(t, `r := runes("ab"); r2 := r.append('c', 'd'); out = r2`, nil, []rune("abcd"))
	expectRun(t, `r := runes("ab"); r2 := r.append(runes("cd")); out = r2`, nil, []rune("abcd"))
	expectRun(t, `r := runes("ab"); r2 := r.append('c'); out = r`, nil, []rune("ab"))

	// sum/avg are gone: they widened rune to int against the type's own
	// arithmetic; reduce or .array() covers the checksum case
	expectError(t, `out = runes("abc").sum()`, nil, "invalid_method")
	expectError(t, `out = runes("abc").avg()`, nil, "invalid_method")
	expectRun(t, `out = runes("abc").reduce(0, func(acc, r) { return acc + r.int() })`, nil, 97+98+99)
	expectError(t, `out = runes("").avg()`, nil, "invalid_method")
	// map is strictly 1:1 and answers the RECEIVER'S type — a runes receiver
	// answers runes, and a sequence or undefined callback result raises
	expectRun(t, `out = runes("abc").map(func(r) { return (r.int() + 1).rune() })`, nil, []rune("bcd"))
	expectRun(t, `out = runes("abc").map(func(i, r) { return (r.int() + i).rune() })`, nil, []rune("ace"))
	expectError(t, `runes("abc").map(func(i, r) { return [i, r] })`, nil, "invalid_argument_type") // an array is not text content
	expectError(t, `runes("abc").map(func(r) { return u"xx" })`, nil, "invalid_value")             // a text run is flat_map's
	expectRun(t, `out = runes("abc").reduce(0, func(acc, r) { return acc + r.int() })`, nil, int64('a'+'b'+'c'))
	expectRun(t, `out = runes("abc").reduce("", func(acc, i, r) { return acc + i.string() + r.string() })`, nil, "0a1b2c")

	// type names
	expectRun(t, `out = type_name(runes("abc"))`, nil, "runes")
	expectRun(t, `out = type_name(freeze(runes("abc")))`, nil, "immutable-runes")

	// immutable rejects writes
	expectError(t, `r := freeze(runes("abc")); r[0] = 'X'`, nil, "not_assignable: type immutable-runes does not support assignment via indexing or field access")

	// slice always produces a fresh independent buffer now (P4-002, closing P01/P02), so it is mutable
	// regardless of the source's mutability — same convention as copy() (see below)
	expectRun(t, `out = type_name(freeze(runes("abcd"))[1:3])`, nil, "runes")
	// stepped slice was already a fresh independent buffer before P4-002, so it is mutable
	expectRun(t, `out = type_name(freeze(runes("abcd"))[::-1])`, nil, "runes")
	// slice_view() is the explicit opt-in for the old sharing behavior, so it still propagates immutability
	expectRun(t, `out = type_name(freeze(runes("abcd")).slice_view(1, 3))`, nil, "immutable-runes")
	// slice of mutable stays mutable
	expectRun(t, `out = type_name(runes("abcd")[1:3])`, nil, "runes")

	// copy of immutable yields mutable
	expectRun(t, `r := freeze(runes("abc")); c := copy(r); c[0] = 'X'; out = c`, nil, []rune("Xbc"))

	// append on immutable returns a fresh mutable value (does not mutate source)
	expectRun(t, `r := freeze(runes("ab")); r2 := r.append('c'); r2[0] = 'X'; out = r2`, nil, []rune("Xbc"))
	expectRun(t, `r := freeze(runes("ab")); r2 := r.append('c'); out = type_name(r2)`, nil, "runes")

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

	// error is unconditionally truthy (regardless of kind/payload/fatal), and .bool() mirrors that
	// exactly, no divergence between implicit truthiness and the explicit conversion — supports the
	// "undefined on success, error on failure" idiom reading naturally as `if x`/`if !x`.
	expectRun(t, `out = !error(1)`, nil, false)
	expectRun(t, `out = !error(undefined)`, nil, false)
	expectRun(t, `out = error(1).bool()`, nil, true)
	expectRun(t, `out = error(1) ? "has error" : "no error"`, nil, "has error")

	// undefined always wins over error, per the implementor contract: error declines to undefined
	// rather than claiming its own "vm error on everything but ==/!=" catch-all.
	expectRun(t, `out = error(1) + undefined`, nil, core.Undefined)
	expectRun(t, `out = undefined + error(1)`, nil, core.Undefined)

	// error otherwise has no arithmetic/bitwise/ordering pairing with anything, unary included.
	expectError(t, `out = error(1) + 1`, nil, "invalid_binary_operator: error + int")
	expectError(t, `out = -error(1)`, nil, "invalid_unary_operator: - error")
	expectError(t, `out = ^error(1)`, nil, "invalid_unary_operator: ^ error")

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

	// array's ONLY operator pairing is same-type '+' (concatenate) — no scalar append/prepend, no
	// '-' at all, deliberately, since an array element can be any type (including another array),
	// so "append one element" vs. "concatenate" would silently mean different things depending on
	// incidental operand type. Confirmed as a checked fact here (step 5's regression test), not an
	// assumption carried over from the audit that found nothing needed to change.
	expectRun(t, `out = [1, 2] + [3, 4]`, nil, ARR{1, 2, 3, 4})
	// member ≡ operator: + takes exactly append's reading (own family spreads, anything
	// else is one element), - takes remove's (element, or every occurrence of the run)
	expectRun(t, `out = [1, 2] + 3`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [1, 2] + "ab"`, nil, ARR{1, 2, "ab"})
	expectRun(t, `out = [9] + range(1, 4)`, nil, ARR{9, core.NewIntRangeValue(1, 4, 1)}) // one element: materializing is spelled .array()
	expectRun(t, `out = [9] + range(1, 4).array()`, nil, ARR{9, 1, 2, 3})
	expectRun(t, `out = [1, 2, 1] - 1`, nil, ARR{2})
	expectRun(t, `out = [1, 2, 1, 2] - [1, 2]`, nil, ARR{})     // every occurrence of the run
	expectRun(t, `out = [1, 2, 3, 2] - [3, 2]`, nil, ARR{1, 2}) // never set difference
	expectRun(t, `out = [1, 2] - []`, nil, ARR{1, 2})           // the empty run removes nothing
	// an operand with no reading of its own for an array hands the operation over, and the array
	// prepends it: `x + a` is exactly `a.prepend(x)`, the mirror of `a + x` = `a.append(x)`
	expectRun(t, `out = 3 + [1, 2]`, nil, ARR{3, 1, 2})
	expectRun(t, `out = range(1, 4) + [9]`, nil, ARR{core.NewIntRangeValue(1, 4, 1), 9})
	expectRun(t, `out = true + [1]`, nil, ARR{true, 1})
	expectRun(t, `out = {a: 1} + [1]`, nil, ARR{MAP{"a": 1}, 1})
	// only + has a reflected form — only the add side has a front spelling. Removal has no front
	// member, so - and every other operator raise rather than inventing one
	expectError(t, `out = 3 - [1, 2]`, nil, "invalid_binary_operator: int - array")
	expectError(t, `out = 3 * [1, 2]`, nil, "invalid_binary_operator: int * array")
	expectError(t, `out = 3 < [1, 2]`, nil, "invalid_binary_operator: int < array")
	// the universal contracts outrank the element reading on this side too
	expectRun(t, `out = undefined + [1]`, nil, core.Undefined)
	expectError(t, `out = error("x") + [1]`, nil, "invalid_binary_operator: error + array")
	// a nested array is an entirely ordinary thing to want to append as one element — and that's
	// exactly the ambiguity '+' avoids by not defining array + scalar at all: [[1], [2]] + [3]
	// unambiguously concatenates (both operands are arrays), it never means "append [3] as a
	// single nested element."
	expectRun(t, `out = [[1], [2]] + [3]`, nil, ARR{ARR{1}, ARR{2}, 3})

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

	expectRun(t, `out = []`, nil, ARR{})
	expectRun(t, `out = array()`, nil, ARR{})
	// array(x) represents x as a sequence: a non-sequence value is one element (the sizing form is gone)
	expectRun(t, `out = array(3)`, nil, ARR{3})
	expectRun(t, `out = array(-1)`, nil, ARR{-1})
	expectError(t, `out = array(undefined)`, nil, "value is missing")
	// array(x, n) is n copies of x AS ONE ELEMENT (never spreads) — exactly [x].repeat(n)
	expectRun(t, `out = array(0, 2)`, nil, ARR{0, 0})
	expectRun(t, `out = array("ab", 2)`, nil, ARR{"ab", "ab"})
	expectRun(t, `out = array([1, 2], 2)`, nil, ARR{ARR{1, 2}, ARR{1, 2}})
	expectRun(t, `out = array(undefined, 3)`, nil, ARR{nil, nil, nil}) // the explicit preallocation
	expectRun(t, `out = array("ab", 0)`, nil, ARR{})
	expectError(t, `out = array("ab", -1)`, nil, "repeat count must be non-negative")
	expectError(t, `out = array("ab", 1.5)`, nil, "must be a whole number")
	// the decomposing repetition is spelled convert-then-repeat
	expectRun(t, `out = array("ab").repeat(2)`, nil, ARR{'a', 'b', 'a', 'b'})
	// the two arities are two operations: arity 1 decomposes a convertible, arity 2 keeps it whole
	expectRun(t, `out = array("ab")`, nil, ARR{'a', 'b'})
	expectRun(t, `out = array("ab", 1)`, nil, ARR{"ab"})

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
	expectRun(t, `t := array(); out = t.is_empty()`, nil, true)
	expectRun(t, `t := [1, 2, 3]; out = t.is_empty()`, nil, false)
	expectRun(t, `t := array(3); out = t.is_empty()`, nil, false)

	expectRun(t, `t := []; out = t.len()`, nil, 0)
	expectRun(t, `t := array(); out = t.len()`, nil, 0)
	expectRun(t, `t := [1, 2, 3]; out = t.len()`, nil, 3)
	expectRun(t, `t := array(undefined, 3); out = t.len()`, nil, 3)

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
	// chunk() always copies now (P4-002, closing P01/P02) — its `copy` bool parameter is retired; chunk_view()
	// is the explicit opt-in for sharing (see TestMemberFunctionSliceViewChunkView).
	expectRun(t, `a := [1, 2, 3]; c := a.chunk(2); c[0][0] = 9; out = a`, nil, ARR{1, 2, 3})
	expectError(t, `a := [1, 2, 3]; c := a.chunk(2, false); c[0][0] = 9; out = a`, nil, "wrong_num_arguments")
	expectError(t, `a := [1, 2, 3]; c := a.chunk(2, true); c[0][0] = 9; out = a`, nil, "wrong_num_arguments")
	expectError(t, `out = [1, 2, 3].chunk()`, nil, "wrong_num_arguments: (chunk) expected 1 argument(s), got 0")
	expectError(t, `out = [1, 2, 3].chunk("x")`, nil, "invalid_argument_type: (chunk) argument first expects type int, got string")
	expectError(t, `out = [1, 2, 3].chunk(2, 1)`, nil, "wrong_num_arguments: (chunk) expected 1 argument(s), got 2")
	expectError(t, `out = [1, 2, 3].chunk(0)`, nil, "invalid_value: chunk size must be positive")
	expectError(t, `out = [1, 2, 3].chunk(-1)`, nil, "invalid_value: chunk size must be positive")

	// for_each makes a FULL pass and ignores the callback's return — the falsy
	// return no longer stops the loop (a forgotten return used to silently visit
	// exactly one element); early exit is for/break or a search member
	expectRun(t, `
out = 0
ignored := [1, 2, 3, 4].for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 10)
	expectRun(t, `out = [1, 2].for_each(func(v) {}) == [1, 2]`, nil, true) // returns the receiver, so it chains

	expectRun(t, `
out = 0
ignored := [10, 20, 30].for_each(func(i, v) {
	out += i * v
	return true
})
`, nil, 80)

	expectRun(t, `out = [1].for_each(func(v) { return true }) == [1]`, nil, true) // returns the receiver
	expectError(t, `out = [1].for_each()`, nil, "wrong_num_arguments: (for_each) expected 1 argument(s), got 0")
	expectError(t, `out = [1].for_each(1)`, nil, "invalid_argument_type: (for_each) argument first expects type function, got int")
	expectError(t, `out = [1].for_each(func() { return true })`, nil, "invalid_argument_type: (for_each) argument first expects type f/1 or f/2")

	expectRun(t, `out = [10, 20, 30].index(x => x == 20)`, nil, 1)
	expectRun(t, `out = [10, 20, 30].index(x => x == 99)`, nil, core.Undefined)
	expectRun(t, `out = [10, 20, 30].index((i, v) => i == 2)`, nil, 2)
	expectRun(t, `out = [10, 20, 30].index((i, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, `out = [].index(x => true)`, nil, core.Undefined)
	expectRun(t, `out = [0, 0, 5].index()`, nil, 2)             // no-arg = first significant (non-blank) element
	expectRun(t, `out = [10, 20].index(20)`, nil, 1)            // element reading
	expectRun(t, `out = [1, 2, 3, 2, 3].index([2, 3])`, nil, 1) // run reading — the receiver's own kind
	expectRun(t, `out = [1, 2, 3, 2, 3].index_last([2, 3])`, nil, 3)
	expectRun(t, `out = [1, 2].index(9, "-")`, nil, "-") // miss -> the optional default
	expectError(t, `out = [1].index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")

	expectRun(t, `out = [].reduce(0, (a, v) => a + v)`, nil, 0)
	expectRun(t, `out = [1, 2, 3].reduce(0, (a, v) => a + v)`, nil, 6)
	expectRun(t, `out = [1, 2, 3].reduce(0, (a, i, v) => a + i)`, nil, 3)
	expectRun(t, `out = [1, 2].reduce(0, (a, v) => a + [10, 20].reduce(0, (b, w) => b + w) + v)`, nil, 63)

	expectRun(t, `out = [1, 2, 3].array()`, nil, ARR{1, 2, 3})
	// element-wise, all-or-nothing: -1 is not an octet, so the mod-256 wrap is gone
	expectError(t, `out = [48, 49, -1].bytes()`, nil, "conversion: cannot convert array to bytes")
	expectRun(t, `out = [48, 49].bytes()`, nil, core.NewBytesValue([]byte{48, 49}, false))
	// the ENTRIES reading: an entry is exactly a 2-element array
	expectRun(t, `out = [["a", 1], ["b", 2]].dict()`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `out = [["a", 1], ["b", 2]].record()`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `out = [["a", 1], ["a", 9]].dict()`, nil, MAP{"a": 9})                      // duplicates: last wins
	expectError(t, `out = [48, 49].dict()`, nil, "conversion: cannot convert array to dict") // decomposition is gone
	expectError(t, `out = ["ab", "cd"].dict()`, nil, "conversion")                           // a 2-element TEXT sequence is not an entry
	expectRun(t, `out = [["a", 1]].dict().array()`, nil, ARR{ARR{"a", 1}})                   // round-trip, key-sorted
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
	// A map RENDERS and ENCODES key-sorted, so a display, a JSON payload and a binary blob are the same
	// on every run. (Iteration order over a map stays deliberately undefined — see the type page.)
	expectRun(t, `out = format("{0}", [{b: 2, a: 1, c: 3}])`, nil, `{"a": 1, "b": 2, "c": 3}`)
	expectRun(t, `out = format("{0}", [dict({b: 2, a: 1})])`, nil, `dict({"a": 1, "b": 2})`)
	expectRun(t, `json := import("json"); out = json.encode({b: 2, a: 1}).string()`, nil, `{"a":1,"b":2}`)

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
	// selector access is the record's feature: a dict refuses the dot in BOTH directions
	expectError(t, `d := dict({a: 1}); out = d.a`, nil, "has no property a")
	expectError(t, `d := dict({a: 1}); d.a = 5`, nil, "has no property a")
	expectError(t, `d := dict({a: 1}); d.keys = 1`, nil, "has no property keys")
	expectRun(t, `d := dict({a: 1}); d["a"] = 5; out = d["a"]`, nil, 5)
	expectError(t, `d := dict({x: dict()}); d.x.y = 1`, nil, "has no property x") // the walk sees the spelling too
	expectRun(t, `d := dict({x: dict()}); d["x"]["y"] = 1; out = d["x"]["y"]`, nil, 1)

	// a non-string key stores under its canonical text (the .string() reading); a byte's is its symbol
	expectRun(t, `d := dict(); d[b'A'] = 1; out = d.keys()`, nil, ARR{"A"})
	expectRun(t, `d := dict(); d[b'A'] = 1; out = d["A"]`, nil, 1)
	expectRun(t, `d := dict(); d[65] = 1; out = d.keys()`, nil, ARR{"65"})
	expectError(t, `d := dict(); d[b'\xFF'] = 1`, nil, "expected string, got byte") // a high octet has no symbol

	// a sequence-typed key raises — a transcode is not a key
	expectError(t, `d := dict(); d[[1, 2]] = 1`, nil, "expected string, got array")
	expectError(t, `d := dict(); d[range(1, 3)] = 1`, nil, "expected string, got range")
	expectError(t, `d := dict({a: 1}); out = [1, 2] in d`, nil, "in")

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

	// keys()/values() must return keys in a deterministic (lexically sorted) order without needing .sort() to mask
	// Go's randomized map iteration order — regression test for the dict.go sortedKeys() fix.
	expectRun(t, `t := dict({z: 1, a: 2, m: 3}); out = t.keys()`, nil, ARR{"a", "m", "z"})
	expectRun(t, `t := dict({z: 1, a: 2, m: 3}); out = t.values()`, nil, ARR{2, 3, 1})
	expectRun(t, `t := dict({z: 1, a: 2, m: 3}); out = t.keys() == t.keys()`, nil, true)
	expectRun(t, `t := dict({z: 1, a: 2, m: 3}); out = t.values() == t.values()`, nil, true)
	expectRun(t, `t := dict({z: 1, a: 2, m: 3}); out = f"{t}"`, nil, `dict({"a": 2, "m": 3, "z": 1})`)

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.filter(k => k != "b").keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.filter((k, v) => v > 1).keys().sort()`, nil, ARR{"b", "c"})
	expectError(t, `t := dict({a: 1, b: undefined}); t.filter()`, nil, "wrong_num_arguments") // no blank reading on a map: two axes
	expectRun(t, `t := dict({a: 1, b: undefined, c: 3}); out = t.remove(func(k, v) { return v == undefined }).keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.filter("a", "c").keys().sort()`, nil, ARR{"a", "c"}) // variadic key set
	expectRun(t, `t := dict({a: 1, b: 2}); out = t.contains("x", "b")`, nil, true)                            // set = any-of
	expectRun(t, `t := dict({a: 1, b: 2}); out = t.all("a", "b")`, nil, true)
	expectRun(t, `t := dict({a: 1, b: 2}); out = t.all("a", "x")`, nil, false)

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
	items = items.append(k + v.string())
	return true
})
out = items.sort()
`, nil, ARR{"a1", "b2"})

	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.index(k => k == "b")`, nil, "b")
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.index(k => k == "q")`, nil, core.Undefined)
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.index((k, v) => v == 2)`, nil, "b")
	expectRun(t, `t := dict({a: 1, b: 2, c: 3}); out = t.index((k, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, `t := dict(); out = t.index(k => true)`, nil, core.Undefined)
	expectError(t, `dict({a: 1}).index()`, nil, "wrong_num_arguments") // no absent reading on a map: keys are identities, never filler
	expectRun(t, `out = dict({a: 1}).index("a")`, nil, "a")            // the value reading answers the KEY
	expectRun(t, `out = dict({a: 1}).index("z", "-")`, nil, "-")
	expectError(t, `dict({a: 1}).index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")

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
	// one text form: RFC3339 with the fraction the instant carries — Go's
	// default text form (with its own layout) is gone
	require.Equal(t, "2020-06-20T01:02:03.000000004Z", s)
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
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_ms()`, nil, 1592614923000)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_micro()`, nil, 1592614923000000)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_nano()`, nil, 1592614923000000004)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day()`, nil, 6)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day_name()`, nil, "Saturday")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").month_name()`, nil, "June")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").year_day()`, nil, 172) // June 20 is the 172nd day of the year (173rd in leap years)
	// one type, one render surface: the format_* trio is gone; the specs name the layouts
	expectError(t, `time(0).format_date()`, nil, "invalid_method")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format("#date")`, nil, "2020-06-20")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format("#datetime")`, nil, "2020-06-20 01:02:03")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").utc().string()`, nil, "2020-06-19T23:02:03.000000004Z")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").zone_offset()`, nil, 7200)

	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").string()`, nil, "2020-06-20T01:02:03.000000004+02:00") // one text form: RFC3339Nano
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").int().time().utc().string()`, nil, "2020-06-19T23:02:03Z")

	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format()`, nil, "2020-06-20T01:02:03.000000004+02:00") // precision-preserving; #iso truncates
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format("#iso")`, nil, "2020-06-20T01:02:03+02:00")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 +0200").format("v")`, nil, `time("2020-06-20T01:02:03.000000004+02:00")`)

	// int -> time: in conversion context an int is a unix timestamp, in the encoding the method
	// names. Each is the exact inverse of the time accessor with the matching suffix, and each
	// produces UTC so the result never depends on the host's timezone. See docs/types/time.md.
	expectRun(t, `out = (1592614923).time().string()`, nil, "2020-06-20T01:02:03Z")
	expectRun(t, `out = (1592614923123).time_ms().string()`, nil, "2020-06-20T01:02:03.123Z")
	expectRun(t, `out = (1592614923123456).time_micro().string()`, nil, "2020-06-20T01:02:03.123456Z")
	expectRun(t, `out = (1592614923000000004).time_nano().string()`, nil, "2020-06-20T01:02:03.000000004Z")

	// the round trip that was impossible before the *_nano/_ms/_micro pairs existed: the only int
	// constructor read seconds, so anything sub-second could not survive a conversion out and back.
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_nano().time_nano() == time("2020-06-20 01:02:03.000000004 UTC")`, nil, true)
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").int().time() == time("2020-06-20 01:02:03.000000004 UTC")`, nil, false) // seconds encoding truncates

	// operator context is the other role: int is a duration in NANOSECONDS, never a timestamp.
	expectRun(t, `out = (t"2020-06-20T01:02:03Z" + 1000000000).unix() - t"2020-06-20T01:02:03Z".unix()`, nil, 1)
	expectRun(t, `out = t"2020-06-20T01:02:03Z" + 1 == t"2020-06-20T01:02:03Z"`, nil, false)
	expectRun(t, `out = (t"2020-06-20T01:02:03Z" + 1).unix_nano() - t"2020-06-20T01:02:03Z".unix_nano()`, nil, 1)

	// the two roles are deliberately NOT bridged: comparing an instant against a bare int would have
	// to pick one of them, so it stays an error. Convert explicitly instead.
	expectError(t, `t"2020-06-20T01:02:03Z" < 1592614923`, nil, "invalid_binary_operator")
	expectError(t, `1592614923 < t"2020-06-20T01:02:03Z"`, nil, "invalid_binary_operator")
	expectRun(t, `out = t"2020-06-20T01:02:03Z" == 1592614923`, nil, false)
	expectRun(t, `out = t"2020-06-20T01:02:03Z" < (1592614924).time()`, nil, true)
	expectRun(t, `out = t"2020-06-20T01:02:03Z".unix() < 1592614924`, nil, true)

	// float/decimal in conversion position: a unix timestamp read as sec.frac -- integer part is
	// seconds, fraction is the sub-second part (the encoding Python's time.time() produces).
	expectRun(t, `out = time(1704067200.5).string()`, nil, "2024-01-01T00:00:00.5Z")
	expectRun(t, `out = (1704067200.5).time().string()`, nil, "2024-01-01T00:00:00.5Z")
	expectRun(t, `out = time(1704067200.123456789d).string()`, nil, "2024-01-01T00:00:00.123456789Z")
	expectRun(t, `out = (1704067200.123456789d).time().string()`, nil, "2024-01-01T00:00:00.123456789Z")
	expectRun(t, `out = time(1704067200.123456789d).unix_nano()`, nil, 1704067200123456789)
	expectRun(t, `out = time(0.0).string()`, nil, "1970-01-01T00:00:00Z")
	expectRun(t, `out = time(-0.5).string()`, nil, "1969-12-31T23:59:59.5Z") // negative: pre-epoch
	expectRun(t, `out = time(-1.5d).string()`, nil, "1969-12-31T23:59:58.5Z")
	expectRun(t, `out = time(1704067200d) == time(1704067200)`, nil, true)

	// decimal is the exact path (base 10); float64 cannot spell most sub-second values, and the
	// documented consequence is visible here rather than hidden.
	expectRun(t, `out = time(1704067200.123).string()`, nil, "2024-01-01T00:00:00.122999907Z")
	expectRun(t, `out = time(1704067200.123d).string()`, nil, "2024-01-01T00:00:00.123Z")

	// declines: NaN/Inf and out-of-range raise, or answer the time(x, fallback) default
	expectError(t, `out = time(1e300)`, nil, "conversion: cannot convert float to time")
	expectError(t, `out = time(float("nan"))`, nil, "conversion: cannot convert float to time")
	expectError(t, `out = decimal("NaN")`, nil, "conversion: cannot convert string to decimal")                            // a parse never produces NaN
	expectError(t, `out = time(1e300, "fallback")`, nil, "wrong_num_arguments: (time) expected 0 or 1 argument(s), got 2") // no free default form
	expectRun(t, `out = (1e300).time("fallback")`, nil, "fallback")                                                        // the member default is the fallible-conversion spelling

	// every int-shaped construction path is UTC, so wall-clock accessors never depend on the host's
	// timezone. A numeric string used to come back in local time (time("1704067200").hour() was the
	// machine's offset); an explicit zone in the input is data and is still preserved.
	expectRun(t, `out = time("1704067200").hour()`, nil, 0)
	expectRun(t, `out = time(1704067200).hour()`, nil, 0)
	expectRun(t, `out = time(1704067200.0).hour()`, nil, 0)
	expectRun(t, `out = time("2024-01-01T12:00:00+05:30").hour()`, nil, 12)

	// f-string / format() int-encoding specs, one per accessor
	expectRun(t, `ts = t"2020-06-20T01:02:03.000000004Z"; out = f"{ts:#unix}"`, nil, "1592614923")
	expectRun(t, `ts = t"2020-06-20T01:02:03.000000004Z"; out = f"{ts:#unixms}"`, nil, "1592614923000")
	expectRun(t, `ts = t"2020-06-20T01:02:03.000000004Z"; out = f"{ts:#unixmicro}"`, nil, "1592614923000000")
	expectRun(t, `ts = t"2020-06-20T01:02:03.000000004Z"; out = f"{ts:#unixnano}"`, nil, "1592614923000000004")
	expectRun(t, `out = time("2020-06-20 01:02:03.000000004 UTC").format("#unixnano")`, nil, "1592614923000000004")
}

func TestDictRecord(t *testing.T) {
	// merge via '+' — new capability, dict/record had no BinaryOp hook at all before this redesign.
	// rhs always wins key collisions (last-writer-wins); record + record stays record, but dict
	// wins the moment either side is dict (dict is the more general of the pair).
	expectRun(t, `out = (dict({a: 1}) + dict({b: 2})).keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, `out = (dict({a: 1, b: 1}) + dict({b: 2}))["b"]`, nil, 2)
	expectRun(t, `r := {a: 1} + {b: 2}; out = [r.a, r.b]`, nil, ARR{1, 2})
	expectRun(t, `out = ({a: 1, b: 1} + {b: 2}).b`, nil, 2)
	expectRun(t, `out = type_name({a: 1} + {b: 2})`, nil, "record")
	expectRun(t, `out = ({a: 1} + dict({b: 2})).keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, `out = (dict({a: 1}) + {b: 2}).keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, `out = type_name({a: 1} + dict({b: 2}))`, nil, "dict")
	expectRun(t, `out = type_name(dict({a: 1}) + {b: 2})`, nil, "dict")
	// rhs still wins collisions regardless of which side is dict vs. record.
	expectRun(t, `out = ({a: 1, b: 1} + dict({b: 2}))["b"]`, nil, 2)
	expectRun(t, `out = (dict({a: 1, b: 1}) + {b: 2})["b"]`, nil, 2)

	// dict - string => dict, remove that key, non-mutating — parallels dict's own remove() member.
	expectRun(t, `out = (dict({a: 1, b: 2}) - "a").keys()`, nil, ARR{"b"})
	expectRun(t, `out = (dict({a: 1}) - "z").keys()`, nil, ARR{"a"}) // missing key: no-op
	// record has no '-' at all — its removal goes through remove()/remove_in_place() instead.
	expectError(t, `out = {a: 1} - "a"`, nil, "invalid_binary_operator: record - string")

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

	// bytes ordering — same-type, a new capability that was missing entirely before this redesign.
	expectRun(t, `out = bytes("abc") < bytes("abd")`, nil, true)
	expectRun(t, `out = bytes("abd") > bytes("abc")`, nil, true)
	expectRun(t, `out = bytes("abc") <= bytes("abc")`, nil, true)
	expectRun(t, `out = bytes("abc") >= bytes("abc")`, nil, true)

	// fixed rank bytes > runes > string, order-independent for the TYPE, but content order still
	// respects which side was written first (concatenation is never commutative in content).
	expectRun(t, `out = bytes("ab") + runes("cd")`, nil, []byte("abcd"))
	expectRun(t, `out = runes("cd") + bytes("ab")`, nil, []rune("cdab")) // receiver decides: runes
	expectRun(t, `out = bytes("ab") + "cd"`, nil, []byte("abcd"))
	expectRun(t, `out = "cd" + bytes("ab")`, nil, "cdab") // receiver decides: string
	expectRun(t, `out = bytes("ab") > runes("cd")`, nil, false)
	expectRun(t, `out = runes("cd") > bytes("ab")`, nil, true)
	expectRun(t, `out = bytes("ab") > "cd"`, nil, false)

	// byte/rune scalars joining bytes — bytes owns both, never the scalar.
	expectRun(t, `out = b'A' + bytes("bc")`, nil, []byte("Abc"))
	expectRun(t, `out = bytes("bc") + b'A'`, nil, []byte("bcA"))
	expectRun(t, `out = 'A' + bytes("bc")`, nil, []byte("Abc"))
	expectRun(t, `out = bytes("bc") + 'A'`, nil, []byte("bcA"))

	// removal family: byte/rune/string/bytes/runes all accepted (bytes owns every pairing it's in,
	// same as + and ordering — the rhs just needs a byte encoding, which every one of these already
	// has), non-mutating. lhs-only, no reflected form (scalar/sequence - bytes is never defined).
	expectRun(t, `out = bytes("banana") - b'a'`, nil, []byte("bnn"))
	expectRun(t, `out = bytes("banana") - bytes("an")`, nil, []byte("ba"))
	expectRun(t, `out = bytes("abc") - "b"`, nil, []byte("ac"))
	expectRun(t, `out = bytes("abc") - 'b'`, nil, []byte("ac"))
	expectRun(t, `out = bytes("banana") - runes("an")`, nil, []byte("ba"))
	expectError(t, `out = b'a' - bytes("banana")`, nil, "invalid_binary_operator: byte - bytes")

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
	expectError(t, `out = bytes("abc").record()`, nil, "invalid_method") // elements are never entries
	expectError(t, `out = bytes("abc").dict()`, nil, "invalid_method")
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
	expectRun(t, `out = bytes("hello").index(x => x == 'l')`, nil, 2)
	expectRun(t, `out = bytes("hello").index(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, `out = bytes("hello").index((i, x) => i == 3)`, nil, 3)
	expectRun(t, `out = bytes("hello").index((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, `out = bytes("").index(x => true)`, nil, core.Undefined)
	expectRun(t, `out = bytes("x").index()`, nil, 0)                // first non-blank octet
	expectRun(t, `out = bytes("hello").index(bytes("ll"))`, nil, 2) // run reading, octet offsets
	expectRun(t, `out = bytes("hello").index("ll")`, nil, 2)        // text encodes into the receiver's representation
	expectRun(t, `out = bytes("hello").index_last(b'l')`, nil, 3)   // element reading
	expectError(t, `out = bytes("x").index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")
	expectRun(t, `out = bytes("hello").min()`, nil, byte('e'))
	expectRun(t, `out = bytes("hello").max()`, nil, byte('o'))
	// int + byte now owns the pairing (byte's ring arithmetic wins, int declines) — out becomes
	// byte after the first accumulation, not int as it used to.
	expectRun(t, `
out = 0
ignored := bytes("abc").for_each(func(b) {
	out += b // byte ring arithmetic: 97+98+99 wraps mod 256
	return b < 'b' // ignored — full pass
})
`, nil, byte(38))
	expectRun(t, `
items := []
ignored := bytes("ABC").for_each(func(i, b) {
	items = items.append(i, b)
	return true
})
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
	expectRun(t, `
items := []
for i, b in bytes("ABC") {
	items = items.append(i, b)
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
	expectRun(t, `b := bytes("ab"); b2 := b.append('c'); out = b2`, nil, []byte("abc"))
	expectRun(t, `b := bytes("ab"); b2 := b.append('c', 'd'); out = b2`, nil, []byte("abcd"))
	expectRun(t, `b := bytes("ab"); b2 := b.append(bytes("cd")); out = b2`, nil, []byte("abcd"))
	expectRun(t, `b := bytes("ab"); b2 := b.append(99); out = b2`, nil, []byte("abc"))
	expectRun(t, `b := bytes("ab"); b2 := b.append('c'); out = b`, nil, []byte("ab"))

	// sum/avg are gone: they widened byte to int against the type's own arithmetic
	expectError(t, `out = bytes("abc").sum()`, nil, "invalid_method")
	expectError(t, `out = bytes("abc").avg()`, nil, "invalid_method")
	expectRun(t, `out = bytes("abc").reduce(0, func(acc, b) { return acc + b.int() })`, nil, 97+98+99)
	expectError(t, `out = bytes().avg()`, nil, "invalid_method")
	// map is strictly 1:1 and answers the RECEIVER'S type — a bytes receiver
	// answers bytes; a result outside the octet domain, a sequence, or undefined raises
	expectRun(t, `out = bytes("abc").map(func(b) { return b + 1 })`, nil, []byte("bcd"))
	expectError(t, `bytes("abc").map(func(i, b) { return [i, b] })`, nil, "invalid_argument_type") // an array is not text content
	expectError(t, `bytes("abc").map(func(b) { return bytes("xx") })`, nil, "invalid_value")       // a text run is flat_map's
	expectError(t, `bytes([200, 100]).map(func(b) { return b.int() * 2 })`, nil, "invalid_value")  // 400 leaves the octet domain
	// byte + int now owns the pairing (wraps mod 256) — converted explicitly to keep this an
	// unwrapped int sum, matching the original intent rather than relying on byte accumulation.
	expectRun(t, `out = bytes("abc").reduce(0, func(acc, b) { return acc + b.int() })`, nil, 97+98+99)
	expectRun(t, `out = bytes("abc").reduce("", func(acc, i, b) { return acc + i.string() + b.string() })`, nil, "0a1b2c") // b.string() is the symbol now

	// type names
	expectRun(t, `out = type_name(bytes("abc"))`, nil, "bytes")
	expectRun(t, `out = type_name(freeze(bytes("abc")))`, nil, "immutable-bytes")

	// immutable rejects writes
	expectError(t, `b := freeze(bytes("abc")); b[0] = 'X'`, nil, "not_assignable: type immutable-bytes does not support assignment via indexing or field access")

	// slice always produces a fresh independent buffer now (P4-002, closing P01/P02), so it is mutable
	// regardless of the source's mutability — same convention as copy() (see below)
	expectRun(t, `out = type_name(freeze(bytes("abcd"))[1:3])`, nil, "bytes")
	// stepped slice was already a fresh independent buffer before P4-002, so it is mutable
	expectRun(t, `out = type_name(freeze(bytes("abcd"))[::-1])`, nil, "bytes")
	// slice_view() is the explicit opt-in for the old sharing behavior, so it still propagates immutability
	expectRun(t, `out = type_name(freeze(bytes("abcd")).slice_view(1, 3))`, nil, "immutable-bytes")
	// slice of mutable stays mutable
	expectRun(t, `out = type_name(bytes("abcd")[1:3])`, nil, "bytes")

	// copy of immutable yields mutable
	expectRun(t, `b := freeze(bytes("abc")); c := copy(b); c[0] = 'X'; out = c`, nil, []byte("Xbc"))

	// append on immutable returns fresh mutable (does not mutate source)
	expectRun(t, `b := freeze(bytes("ab")); b2 := b.append('c'); b2[0] = 'X'; out = b2`, nil, []byte("Xbc"))
	expectRun(t, `b := freeze(bytes("ab")); b2 := b.append('c'); out = type_name(b2)`, nil, "bytes")

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
	// the single-variable form yields KEYS — a map's element is its key; the
	// values are `for _, v in m` or the two-variable form
	expectRun(t, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
for k in m {
	sum1 += k[0] - 'a'
}
out = sum1
`, nil, 45)

	expectRun(t, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
for _, v in m {
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
	// the single-variable form yields KEYS — a map's element is its key
	expectRun(t, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
for k in m {
	sum1 += k[0] - 'a'
}
out = sum1
`, nil, 45)

	expectRun(t, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
for _, v in m {
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

func TestIsTrue(t *testing.T) {
	// is_true is the boolean-context test — the same answer as !!x and `if x` —
	// in both spellings: the free builtin and the universal member. It is NOT
	// equality with `true`: is_true([1]) is true while [1] == true is false.
	expectRun(t, `out = is_true(0)`, nil, false)
	expectRun(t, `out = is_true(1)`, nil, true)
	expectRun(t, `out = is_true(0.0)`, nil, false)
	expectRun(t, `out = is_true("")`, nil, false)
	expectRun(t, `out = is_true("x")`, nil, true)
	expectRun(t, `out = is_true([])`, nil, false)
	expectRun(t, `out = is_true([1])`, nil, true)
	expectRun(t, `out = is_true([1]) != ([1] == true)`, nil, true)
	expectRun(t, `out = is_true(dict({}))`, nil, false)
	expectRun(t, `out = is_true(undefined)`, nil, false)
	expectRun(t, `out = is_true(error("boom"))`, nil, true) // every error is truthy
	expectRun(t, `out = is_true(func(){})`, nil, true)      // callables have no zero value

	expectRun(t, `out = (0).is_true()`, nil, false)
	expectRun(t, `out = (1).is_true()`, nil, true)
	expectRun(t, `out = "".is_true()`, nil, false)
	expectRun(t, `out = [1].is_true()`, nil, true)
	expectRun(t, `out = dict({}).is_true()`, nil, false)
	expectRun(t, `out = undefined.is_true()`, nil, false)
	expectRun(t, `out = time().is_true()`, nil, false) // the zero instant
	expectRun(t, `f := func(){}; out = f.is_true()`, nil, true)

	// the callable form is the point: truthiness as a first-class function
	expectRun(t, `out = [0, 1, "", "x", []].filter(is_true)`, nil, ARR{1, "x"})

	// error states raise instead of answering
	expectError(t, `out = is_true(float("nan"))`, nil, "invalid_value")
	expectError(t, `out = float("nan").is_true()`, nil, "invalid_value")
}

func TestRange(t *testing.T) {
	// range() is the type's zero form — the empty range
	expectRun(t, `out = range().len()`, nil, 0)
	expectRun(t, `out = range().is_empty()`, nil, true)
	expectRun(t, `out = is_true(range())`, nil, false)
	expectRun(t, `out = range(97, 103, 1).bytes().string()`, nil, "abcdef")
	expectRun(t, `out = range(103, 97, 1).bytes().string()`, nil, "gfedcb")
	expectRun(t, `out = range(97, 103, 1).string()`, nil, "abcdef")
	expectRun(t, `out = range(103, 97, 1).string()`, nil, "gfedcb")
	expectError(t, `out = range(1, 3, 1).record()`, nil, "invalid_method") // elements are never entries
	expectError(t, `out = range(1, 3, 1).dict()`, nil, "invalid_method")
	// the components map is the way back instead
	expectRun(t, `out = range(2, 10, 3).components()["step"]`, nil, 3)
	expectRun(t, `
out = 0
ignored := range(1, 5, 1).for_each(func(v) {
	out += v
	return v < 3 // ignored — full pass
})
`, nil, 10)
	expectRun(t, `
out = 0
ignored := range(10, 13, 1).for_each(func(i, v) {
	out += i + v
	return true
})
`, nil, 36)

	expectRun(t, `out = range(10, 20, 1).index(v => v == 15)`, nil, 5)
	expectRun(t, `out = range(10, 20, 1).index(v => v == 99)`, nil, core.Undefined)
	expectRun(t, `out = range(10, 20, 1).index((i, v) => i == 3)`, nil, 3)
	expectRun(t, `out = range(20, 10, 1).index(v => v == 15)`, nil, 5)
	expectRun(t, `out = range(0, 0, 1).index(v => true)`, nil, core.Undefined)
	expectRun(t, `out = range(0, 5, 1).index()`, nil, 1)  // no-arg = first non-zero element (0 is int's blank)
	expectRun(t, `out = range(0, 5, 1).index(3)`, nil, 3) // element reading
	expectRun(t, `out = range(0, 5, 1).index_last(3)`, nil, 3)
	// the RUN reading is deferred until the vectorised int sequence type exists — never approximated by array
	expectError(t, `range(0, 5, 1).index([1, 2])`, nil, "not_implemented")
	expectError(t, `out = range(0, 5, 1).index(func() { return true })`, nil, "invalid_argument_type: (index) argument first expects type f/1 or f/2")

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
	s1 += (r[i] == e).int()
	s2 += (a[i] == e).int()
}
out = [s1, s2]
`, nil, ARR{20, 20})

	expectRun(t, `
r := range(10, -10, 1)
a := r.array()
s1 := 0
s2 := 0
for i, e in r {
	s1 += (r[i] == e).int()
	s2 += (a[i] == e).int()
}
out = [s1, s2]
`, nil, ARR{20, 20})
}

// TestRangeSyntax covers "low..high" / "low..high:step" as sugar for range(low, high[, step]) — see TestRange for
// the underlying builtin's semantics, which this syntax inherits unchanged (exclusive stop, direction auto-detected,
// step defaults to 1 via the builtin itself).
func TestRangeSyntax(t *testing.T) {
	// bare range literal, as a value and in a for-in header
	expectRun(t, `out = (1..5).array()`, nil, ARR{1, 2, 3, 4})
	expectRun(t, `out = (5..1).array()`, nil, ARR{5, 4, 3, 2})
	expectRun(t, `out = (1..5:2).array()`, nil, ARR{1, 3})
	expectRun(t, `out = (1..1).array()`, nil, ARR{})

	expectRun(t, `
out = 0
for x in 1..5 { out += x }
`, nil, 10)
	expectRun(t, `
out = 0
for x in 5..1 { out += x }
`, nil, 14)
	expectRun(t, `
out = 0
for x in 1..5:2 { out += x }
`, nil, 4)

	// equivalent to calling the builtin directly
	expectRun(t, `out = (1..5) == range(1, 5)`, nil, true)
	expectRun(t, `out = (1..5:2) == range(1, 5, 2)`, nil, true)

	// expression operands, not just literals
	expectRun(t, `n := 3; out = (1..n+2).array()`, nil, ARR{1, 2, 3, 4})
	expectRun(t, `a := 1; b := 5; s := 2; out = (a..b:s).array()`, nil, ARR{1, 3})
	expectRun(t, `f := func(x) { return x }; out = (f(1)..f(4)).array()`, nil, ARR{1, 2, 3})

	// precedence: comparison/equality wrap the whole range rather than being swallowed into the high bound. '<'
	// isn't defined for a range value, so "(1..n) < 5" is a runtime type error — which itself proves the range was
	// built as a whole (1..10) first rather than swallowing "n < 5" into the high bound.
	expectError(t, `n := 10; out = (1..n < 5)`, nil, "invalid_binary_operator: range < int")
	expectRun(t, `out = 1..2+3 == range(1, 5)`, nil, true) // additive binds tighter than range: 1..(2+3)

	// "1..5" is a language construct, not a reference to the identifier "range" — like a b"..." bytes literal never
	// referencing "byte", it is immune to a local `range := ...` reassignment and always calls the true builtin.
	// range(...) written explicitly, by contrast, is a normal (shadowable) call and honors the reassignment.
	expectRun(t, `range := func(a, b) { return a + b }; out = (1..5).array()`, nil, ARR{1, 2, 3, 4})
	expectRun(t, `range := func(a, b) { return a + b }; out = range(1, 5)`, nil, 6)
	expectRun(t, `range := func(a, b) { return a + b }; out = (1..5) != range(1, 5)`, nil, true)

	// arr[low..high] and arr[low..high:step] must behave exactly like the existing arr[low:high]/arr[low:high:step]
	// slice syntax — same underlying *expression.Slice, only the low/high separator spelling differs.
	arr := `arr := [10, 20, 30, 40, 50]; `
	expectRun(t, arr+`out = arr[1..3]`, nil, ARR{20, 30})
	expectRun(t, arr+`out = arr[1:3]`, nil, ARR{20, 30})
	expectRun(t, arr+`out = arr[1..4:2]`, nil, ARR{20, 40})
	expectRun(t, arr+`out = arr[1:4:2]`, nil, ARR{20, 40})

	// omitted bounds work inside brackets exactly like the existing ':' spelling (arr[:3], arr[1:])
	expectRun(t, arr+`out = arr[..3]`, nil, ARR{10, 20, 30})
	expectRun(t, arr+`out = arr[1..]`, nil, ARR{20, 30, 40, 50})

	// a bare range literal has no receiver to default a missing bound against, unlike a bracketed slice — this is a
	// parse error, covered in parser.TestParseRange.
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

func TestUnpack(t *testing.T) {
	// array unpack: positional, exact-or-more length; := and =
	expectRun(t, `a, b, c := [10, 20, 30]; out = [a, b, c]`, nil, ARR{10, 20, 30})
	expectRun(t, `a := 0; b := 0; c := 0; a, b, c = [10, 20, 30]; out = [a, b, c]`, nil, ARR{10, 20, 30})
	expectRun(t, `a, b := [1, 2, 3]; out = [a, b]`, nil, ARR{1, 2}) // extra elements ignored

	// array unpack: too few elements is an out-of-bounds error, same as arr[i]
	expectError(t, `a, b, c := [1, 2]`, nil, "index_out_of_bounds")

	// array unpack: '_' discards a value but still consumes/bounds-checks a position
	expectRun(t, `a, _, c := [1, 2, 3]; out = [a, c]`, nil, ARR{1, 3})
	expectRun(t, `a, _, _, c := [1, 2, 3, 4]; out = [a, c]`, nil, ARR{1, 4})
	expectError(t, `a, _, c := [1, 2]`, nil, "index_out_of_bounds") // '_' at position 1 is still out of range

	// dict/record unpack: keyed by LHS name, missing keys fill undefined, extra keys ignored
	expectRun(t, `a, b := {a: 1, b: 2, c: 3}; out = [a, b]`, nil, ARR{1, 2})
	expectRun(t, `a, b := {a: 1, c: 2}; out = [a, b]`, nil, ARR{1, core.Undefined})
	expectRun(t, `a, b := dict({a: 1, b: 2, c: 3}); out = [a, b]`, nil, ARR{1, 2})
	expectRun(t, `a, b := dict({a: 1}); out = [a, b]`, nil, ARR{1, core.Undefined})

	// dict/record unpack: '_' is inert filler, no key lookup, no requirement it exists
	expectRun(t, `a, _ := {a: 1}; out = a`, nil, 1)
	expectRun(t, `a, _ := {a: 1, z: 99}; out = a`, nil, 1)

	// '_' is never a real variable: repeats freely, never readable, and a plain '_ = x' is a no-op
	expectRun(t, `_ = 1; _ := 2; out = 3`, nil, 3)
	expectError(t, `_ = 1; out = _`, nil, "unresolved reference '_'")

	// duplicate real names in one destructuring statement are a compile error
	expectError(t, `a, c, c, b := [1, 2, 3, 4]`, nil, "'c' used more than once in destructuring assignment")

	// LHS must be plain identifiers - no selectors/nested targets
	expectError(t, `x := {a: 1}; a, x.a := [1, 2]`, nil, "destructuring target must be a plain identifier")

	// arity mismatch between LHS and RHS (neither destructuring - RHS isn't a single expression - nor a valid
	// parallel assignment, since the counts differ) is a compile error
	expectError(t, `a, b = 1, 2, 3`, nil, "assignment mismatch: 2 name(s) on the left, 3 value(s) on the right")
	expectError(t, `a = 1, 2`, nil, "assignment mismatch: 1 name(s) on the left, 2 value(s) on the right")

	// non-collection RHS is a runtime error
	expectError(t, `a, b := 5`, nil, "cannot destructure value of type int")
	expectError(t, `a, b := "hi"`, nil, "cannot destructure value of type string")

	// works for locals/closures too, not just globals
	expectRun(t, `out = func() { a, b := [1, 2]; return a + b }()`, nil, 3)
	expectRun(t, `f1 := func() { a, b := [10, 20]; return func() { return a + b } }; out = f1()()`, nil, 30)

	// regression: the pre-existing "_ = expr()" discard idiom must keep evaluating its RHS for side effects
	expectRun(t, `out = 0; f := func() { out = 1; return 2 }; _ = f(); `, nil, 1)
}

func TestParallelAssignment(t *testing.T) {
	// basic positional pairing, := and =
	expectRun(t, `a, b := 1, 2; out = [a, b]`, nil, ARR{1, 2})
	expectRun(t, `a := 0; b := 0; a, b = 1, 2; out = [a, b]`, nil, ARR{1, 2})
	expectRun(t, `a, b, c := 1, 2, 3; out = [a, b, c]`, nil, ARR{1, 2, 3})

	// all right-hand expressions are evaluated before any left-hand target is stored - swap works
	expectRun(t, `a := 1; b := 2; a, b = b, a; out = [a, b]`, nil, ARR{2, 1})
	expectRun(t, `a := 1; b := 2; c := 3; a, b, c = c, a, b; out = [a, b, c]`, nil, ARR{3, 1, 2})

	// right-hand expressions can be arbitrary, not just identifiers
	expectRun(t, `a, b := 1 + 1, 2 * 3; out = [a, b]`, nil, ARR{2, 6})

	// arity mismatch is a compile error, checked statically (not a runtime unpack failure)
	expectError(t, `a, b := 1, 2, 3`, nil, "assignment mismatch: 2 name(s) on the left, 3 value(s) on the right")
	expectError(t, `a, b, c := 1, 2`, nil, "assignment mismatch: 3 name(s) on the left, 2 value(s) on the right")

	// '_' discards a position, same convention as destructuring
	expectRun(t, `a, _, c := 1, 2, 3; out = [a, c]`, nil, ARR{1, 3})
	expectRun(t, `a := 1; b := 2; _, a = a, b; out = a`, nil, 2) // discard the old 'a', keep the swap semantics

	// duplicate real names in one statement are a compile error
	expectError(t, `a, a := 1, 2`, nil, "'a' used more than once in assignment")

	// targets must be plain identifiers - no selectors/nested targets
	expectError(t, `x := {a: 1}; a, x.a := 1, 2`, nil, "assignment target must be a plain identifier")

	// works for locals/closures too, not just globals
	expectRun(t, `out = func() { a, b := 1, 2; return a + b }()`, nil, 3)
	expectRun(t, `f1 := func() { a, b := 10, 20; return func() { return a + b } }; out = f1()()`, nil, 30)
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

	// byte bitwise — same-type only, no cross-type mixing with int (would reopen "which width
	// wins"), and rune is excluded entirely (checked directly, no real meaning for a code point).
	expectRun(t, `out = byte(0xF0) & byte(0x0F)`, nil, byte(0))
	expectRun(t, `out = byte(0xF0) | byte(0x0F)`, nil, byte(0xFF))
	expectRun(t, `out = byte(0xFF) ^ byte(0x0F)`, nil, byte(0xF0))
	expectRun(t, `out = byte(0xFF) &^ byte(0x0F)`, nil, byte(0xF0))
	expectRun(t, `out = byte(1) << 4`, nil, byte(16))
	expectRun(t, `out = byte(16) >> 4`, nil, byte(1))
	expectRun(t, `out = ^byte(0)`, nil, byte(0xFF))
	expectError(t, `out = byte(1) & 1`, nil, "invalid_binary_operator: byte & int")
	expectError(t, `out = 1 & byte(1)`, nil, "invalid_binary_operator: int & byte")
	expectError(t, `out = 'a' & 'a'`, nil, "invalid_binary_operator: rune & rune")
	expectError(t, `out = byte(1) & 'a'`, nil, "invalid_binary_operator: byte & rune")
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
		fs := ast.NewFileSet()
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
	expectRun(t, `out = len(freeze_shallow([]))`, nil, 0)
	expectRun(t, `out = len(freeze_shallow([1, 2, 3]))`, nil, 3)
	expectRun(t, `out = len(freeze_shallow({}))`, nil, 0)
	expectRun(t, `out = len(freeze_shallow({a:1, b:2}))`, nil, 2)
	// the free len shares the member's domain: types that have a length.
	// len(5) answering 1 was total and nonsensical
	expectError(t, `out = len(undefined)`, nil, "invalid_argument_type")
	expectError(t, `out = len(0)`, nil, "invalid_argument_type")
	expectError(t, `out = len(func(){})`, nil, "invalid_argument_type")
	expectRun(t, `out = len(range(0, 3))`, nil, 3)
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

// TestMemberFunctionCopyShallow checks P3-002's copy_shallow(): top-level container is independent (same
// assertion as copy()'s existing deep-clone behavior), but nested containers still alias the source (the
// opposite assertion) — for every type family that has a real deep/shallow distinction (array, dict, error),
// plus confirms copy()'s own deep-clone behavior still holds after the Clone->Copy(deep bool) hook refactor,
// and that types with no nested Values (bytes/runes/scalars) treat copy_shallow identically to copy.
func TestMemberFunctionCopyShallow(t *testing.T) {
	// array: top-level independent, nested shared
	expectRun(t, `a := [1, 2, 3]; b := a.copy_shallow(); b[0] = 99; out = a`, nil, ARR{1, 2, 3})
	expectRun(t, `a := [[1, 2], [3, 4]]; b := a.copy_shallow(); b[0][0] = 99; out = a[0][0]`, nil, 99)
	// array: copy() stays fully deep after the refactor
	expectRun(t, `a := [[1, 2], [3, 4]]; b := a.copy(); b[0][0] = 99; out = a[0][0]`, nil, 1)

	// dict: top-level independent, nested shared
	expectRun(t, `d := dict({a: 1}); d2 := d.copy_shallow(); d2["a"] = 99; out = d["a"]`, nil, 1)
	expectRun(t, `d := dict({a: [1, 2]}); d2 := d.copy_shallow(); d2["a"][0] = 99; out = d["a"][0]`, nil, 99)
	expectRun(t, `d := dict({a: [1, 2]}); d2 := d.copy(); d2["a"][0] = 99; out = d["a"][0]`, nil, 1)

	// error has no _shallow twin: the twins live only where elements can be
	// containers (array/dict/record); copy() stays the real deep operation
	expectError(t, `error([1]).copy_shallow()`, nil, "invalid_method")
	expectRun(t, `e := error([1, 2, 3]); e2 := e.copy(); p := e2.value(); p[0] = 99; out = e.value()[0]`, nil, 1)

	// bytes/runes have no _shallow twins: their elements are scalars, so the
	// distinction the suffix names is unobservable
	expectError(t, `bytes("abc").copy_shallow()`, nil, "invalid_method")
	expectError(t, `runes("abc").copy_shallow()`, nil, "invalid_method")

	// scalars and string have no _shallow twin any more: nothing to observe
	expectError(t, `(5).copy_shallow()`, nil, "invalid_method")
	expectError(t, `"x".copy_shallow()`, nil, "invalid_method")

	// arity errors, mirroring copy()'s
	expectError(t, `[1, 2, 3].copy_shallow(1)`, nil, "wrong_num_arguments")
	expectError(t, `dict({}).copy_shallow(1)`, nil, "wrong_num_arguments")
	expectError(t, `bytes("x").copy_shallow(1)`, nil, "invalid_method") // the twin left this type
	expectError(t, `runes("x").copy_shallow(1)`, nil, "invalid_method")
	expectError(t, `error("x").copy_shallow(1)`, nil, "invalid_method")
	expectError(t, `(5).copy_shallow(1)`, nil, "invalid_method")

	// record has no member functions at all (see P14/function-matrix.md) — copy_shallow is deliberately not
	// reachable on it, same as copy() today.
	expectError(t, `{}.copy_shallow()`, nil, "type record has no method copy_shallow")
}

// TestMemberFunctionSliceViewChunkView checks P3-003's slice_view()/chunk_view()/is_view(): both _view twins
// reproduce today's two-part-slice/chunk sharing behavior exactly (same assertion style as the existing
// aliasing tests for `a[i:j]` and `chunk(size)`), and is_view() reports true only for values actually produced
// by one of the two _view constructors — not for plain values, not for copy()'d values, and (deliberately, for
// now — see ROADMAP.md P4-002) not for chunk()'s own still-sharing default either.
func TestMemberFunctionSliceViewChunkView(t *testing.T) {
	// slice_view: shares backing storage, same as a[i:j] does today
	expectRun(t, `a := [1, 2, 3, 4, 5]; b := a.slice_view(1, 3); b[0] = 99; out = a[1]`, nil, 99)
	expectRun(t, `a := [1, 2, 3]; b := a.slice_view(); b[0] = 99; out = a[0]`, nil, 99)
	expectRun(t, `a := [1, 2, 3, 4, 5]; out = a.slice_view(1, 3) == a[1:3]`, nil, true)
	expectRun(t, `b := bytes("hello").slice_view(1, 3); b[0] = 'X'; out = b`, nil, []byte("Xl"))
	expectRun(t, `b := bytes("hello"); s := b.slice_view(1, 3); s[0] = 'X'; out = b`, nil, []byte("hXllo"))
	expectRun(t, `r := runes("hello"); s := r.slice_view(1, 3); s[0] = 'X'; out = r`, nil, []rune("hXllo"))
	expectError(t, `[1, 2, 3].slice_view(0, 1, 2)`, nil, "wrong_num_arguments")
	expectError(t, `[1, 2, 3].slice_view("x")`, nil, "invalid_index_type")

	// is_view is a FREE builtin with universal domain: storage-state predicates
	// read the header and must answer on every value (record views included,
	// which no member could reach)
	expectRun(t, `out = is_view([1, 2, 3])`, nil, false)
	expectRun(t, `out = is_view([1, 2, 3].slice_view(1, 2))`, nil, true)
	expectRun(t, `out = is_view([1, 2, 3].slice_view(1, 2).copy())`, nil, false)
	expectRun(t, `out = is_view(5)`, nil, false)
	expectRun(t, `out = is_view(dict({a: 1}).record_view())`, nil, true)
	expectError(t, `[1].is_view()`, nil, "invalid_method")
	expectRun(t, `out = is_view([1, 2, 3].slice_view(1, 2).copy_shallow())`, nil, false)
	expectRun(t, `out = is_view(bytes("abc").slice_view(0, 1))`, nil, true)
	expectRun(t, `out = is_view(runes("abc").slice_view(0, 1))`, nil, true)
	expectError(t, `is_view()`, nil, "wrong_num_arguments")

	// chunk_view: shares backing storage per-chunk, same as chunk(size) does today (chunk_view takes no bool arg)
	expectRun(t, `a := [1, 2, 3, 4]; c := a.chunk_view(2); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3, 4})
	expectRun(t, `out = [1, 2, 3, 4].chunk_view(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}})
	expectRun(t, `out = is_view([1, 2, 3, 4].chunk_view(2)[0])`, nil, true)
	expectError(t, `[1, 2, 3].chunk_view()`, nil, "wrong_num_arguments: (chunk_view) expected 1 argument(s), got 0")
	expectError(t, `[1, 2, 3].chunk_view(2, true)`, nil, "wrong_num_arguments: (chunk_view) expected 1 argument(s), got 2")
	expectError(t, `[1, 2, 3].chunk_view(0)`, nil, "invalid_value: chunk size must be positive")
	expectRun(t, `out = bytes("hello").chunk_view(2)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, `out = u"hello".chunk_view(2)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})

	// today's chunk()/its default (no bool, or explicit false) already shares storage, same as chunk_view, but
	// is deliberately NOT tagged is_view() yet — that rename is P4-002's job, not this step's (see ROADMAP.md).
	expectRun(t, `out = is_view([1, 2, 3, 4].chunk(2)[0])`, nil, false)

	// dict/record are map-shaped, not Seq-shaped — no slice/chunk concept, so none of these exist there.
	expectError(t, `dict({}).is_view()`, nil, "type dict has no method is_view")
	expectError(t, `dict({}).slice_view()`, nil, "type dict has no method slice_view")
	expectError(t, `{}.is_view()`, nil, "type record has no method is_view")
}

// TestMemberFunctionFreeze checks P3-004's freeze()/freeze_shallow(): freeze() always detaches first (deep
// copy + deep immutable-marking of the fresh, not-yet-observable clone) so it never affects the source or any
// existing alias into it; freeze_shallow() skips the detach (today's ToImmutable(), now also member-callable)
// and so does NOT protect the frozen reference's shared body from another still-mutable alias into the same
// data — that's the documented danger motivating freeze() as the safe default.
func TestMemberFunctionFreeze(t *testing.T) {
	// freeze(): detaches, so the source and any pre-existing alias are unaffected
	expectRun(t, `a := [1, 2, 3]; f := a.freeze(); out = is_immutable(f)`, nil, true)
	expectRun(t, `a := [1, 2, 3]; f := a.freeze(); out = is_immutable(a)`, nil, false)
	expectRun(t, `a := [1, 2, 3]; f := a.freeze(); a[0] = 99; out = f`, nil, ARR{1, 2, 3})
	expectError(t, `a := [1, 2, 3]; f := a.freeze(); f[0] = 99`, nil, "not_assignable")

	// freeze() marks nested containers immutable too (deep), and the source's own nested containers stay
	// mutable and independent (deep copy already fully detached them)
	expectError(t, `a := [[1, 2], [3, 4]]; f := a.freeze(); f[0][0] = 99`, nil, "not_assignable")
	expectRun(t, `a := [[1, 2], [3, 4]]; f := a.freeze(); a[0][0] = 99; out = f[0][0]`, nil, 1)

	// freeze_shallow(): today's ToImmutable(), now member-callable; requires reassignment to affect the
	// caller's own variable, same as every other member method in this language
	expectRun(t, `a := [1, 2, 3]; a = a.freeze_shallow(); out = is_immutable(a)`, nil, true)
	expectError(t, `a := [1, 2, 3]; a = a.freeze_shallow(); a[0] = 99`, nil, "not_assignable")

	// The documented danger: freezing `a` does not protect the shared body from a pre-existing sibling alias
	// `b` — b's own header was copied before the freeze and stays independently mutable, and mutating through
	// b is still visible through the now-"frozen" a, since both still point at the same underlying storage.
	expectRun(t, `a := [1, 2, 3]; b := a; a = a.freeze_shallow(); out = is_immutable(b)`, nil, false)
	expectRun(t, `a := [1, 2, 3]; b := a; a = a.freeze_shallow(); b[0] = 99; out = a[0]`, nil, 99)

	// copy_shallow().freeze_shallow() composes to the "shallow freeze" Rule 6 says needs no third name:
	// top level detached and frozen, nested structure still shared with (and mutable through) the source.
	expectRun(t, `a := [[1, 2], [3, 4]]; f := a.copy_shallow().freeze_shallow(); out = is_immutable(f)`, nil, true)
	expectError(t, `a := [[1, 2], [3, 4]]; f := a.copy_shallow().freeze_shallow(); f[0] = [9, 9]`, nil,
		"not_assignable")
	expectRun(t, `a := [[1, 2], [3, 4]]; f := a.copy_shallow().freeze_shallow(); f[0][0] = 99; out = a[0][0]`,
		nil, 99)

	// bytes/runes/dict: same shape
	expectRun(t, `b := bytes("abc"); f := b.freeze(); b[0] = 'X'; out = f`, nil, []byte("abc"))
	expectError(t, `f := bytes("abc").freeze(); f[0] = 'X'`, nil, "not_assignable")
	expectRun(t, `r := runes("abc"); f := r.freeze(); r[0] = 'X'; out = f`, nil, []rune("abc"))
	expectRun(t, `d := dict({a: [1, 2]}); f := d.freeze(); d["a"][0] = 99; out = f["a"][0]`, nil, 1)
	expectError(t, `f := dict({}).freeze(); f["a"] = 1`, nil, "not_assignable")

	// error: freeze() deep-copies and freezes the payload too, without affecting the source's own payload
	expectRun(t, `e := error([1, 2, 3]); f := e.freeze(); p := f.value(); out = is_immutable(p)`, nil, true)
	expectRun(t, `e := error([1, 2, 3]); f := e.freeze(); p := e.value(); p[0] = 77; out = f.value()[0]`, nil, 1)

	// arity errors, mirroring copy()'s
	expectError(t, `[1, 2, 3].freeze(1)`, nil, "wrong_num_arguments")
	expectError(t, `[1, 2, 3].freeze_shallow(1)`, nil, "wrong_num_arguments")
	expectError(t, `dict({}).freeze(1)`, nil, "wrong_num_arguments")
	expectError(t, `bytes("x").freeze_shallow(1)`, nil, "invalid_method") // the twin left this type
	expectError(t, `runes("x").freeze(1)`, nil, "wrong_num_arguments")
	expectError(t, `error("x").freeze_shallow(1)`, nil, "invalid_method")
	expectError(t, `(5).freeze(1)`, nil, "wrong_num_arguments")

	// record has no member functions at all (see P14/function-matrix.md) — freeze/freeze_shallow have no
	// member-call form on it, same as copy()/copy_shallow() today; see TestBuiltinFunctionFreeze/
	// TestBuiltinFunctionFreezeShallow for record's free-function path (added 2026-08-17).
	expectError(t, `{}.freeze()`, nil, "type record has no method freeze")
	expectError(t, `{}.freeze_shallow()`, nil, "type record has no method freeze_shallow")
}

// TestSliceCopyByDefault checks P4-002's flip: the `a[i:j]` operator and the new `.slice(start, end)` member
// function (Rule 10's "slice gets a real member-function name too") both now produce an independently-owned
// copy, closing P01/P02 (the one confirmed engine-level bug in the whole redesign) — mutating through either
// side no longer affects the other, matching copy()'s existing convention exactly (including that the result
// is always mutable, regardless of the source's mutability). slice_view() (P3-003) is unaffected and remains
// the explicit opt-in for the old sharing behavior.
func TestSliceCopyByDefault(t *testing.T) {
	// a[i:j] operator: no longer shares storage
	expectRun(t, `a := [1, 2, 3, 4, 5]; b := a[1:3]; b[0] = 99; out = a[1]`, nil, 2)
	expectRun(t, `a := [1, 2, 3, 4, 5]; b := a[1:3]; a[1] = 99; out = b[0]`, nil, 2)
	expectRun(t, `b := bytes("hello")[1:3]; b[0] = 'X'; out = bytes("hello")[1:3]`, nil, []byte("el"))
	expectRun(t, `s := bytes("hello"); b := s[1:3]; s[1] = 'X'; out = b`, nil, []byte("el"))
	expectRun(t, `s := runes("hello"); r := s[1:3]; s[1] = 'X'; out = r`, nil, []rune("el"))

	// .slice(start, end): same operation, member spelling, same copying behavior
	expectRun(t, `a := [1, 2, 3, 4, 5]; b := a.slice(1, 3); b[0] = 99; out = a[1]`, nil, 2)
	expectRun(t, `out = [1, 2, 3, 4, 5].slice(1, 3) == [1, 2, 3, 4, 5][1:3]`, nil, true)
	expectRun(t, `out = [1, 2, 3].slice()`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [1, 2, 3, 4].slice(2)`, nil, ARR{3, 4})
	expectRun(t, `out = [1, 2, 3, 4].slice(0, 2)`, nil, ARR{1, 2})
	expectRun(t, `out = bytes("hello").slice(1, 3)`, nil, []byte("el"))
	expectRun(t, `out = runes("hello").slice(1, 3)`, nil, []rune("el"))
	expectError(t, `[1, 2, 3].slice(0, 1, 2)`, nil, "wrong_num_arguments")
	expectError(t, `[1, 2, 3].slice("x")`, nil, "invalid_index_type")

	// result is always mutable, regardless of source mutability — same convention as copy()
	expectRun(t, `out = type_name(freeze_shallow([1, 2, 3])[1:3])`, nil, "array")
	expectRun(t, `out = type_name(freeze_shallow([1, 2, 3]).slice(1, 3))`, nil, "array")

	// slice results are real copies, never views
	expectRun(t, `out = is_view([1, 2, 3][1:2])`, nil, false)
	expectRun(t, `out = is_view([1, 2, 3].slice(1, 2))`, nil, false)

	// slice_view() is unaffected: still the explicit opt-in for sharing
	expectRun(t, `a := [1, 2, 3, 4, 5]; b := a.slice_view(1, 3); b[0] = 99; out = a[1]`, nil, 99)
}

func TestBuiltinFunctionMin(t *testing.T) {
	expectError(t, `min()`, nil, "wrong_num_arguments") // a zero-argument selection has no answer
	expectRun(t, `out = min(5)`, nil, 5)
	expectRun(t, `out = min(undefined)`, nil, core.Undefined)
	expectRun(t, `out = min(3, 1, 2)`, nil, 1)
	expectRun(t, `out = min(3.5, 1, 2)`, nil, 1)
	expectRun(t, `out = min("banana", "apple", "cherry")`, nil, "apple")
	expectError(t, `min([]...)`, nil, "wrong_num_arguments") // an empty spread leaves nothing to select among
	expectRun(t, `out = min([7]...)`, nil, 7)
	expectRun(t, `out = min([3, 1, 2]...)`, nil, 1)
	expectRun(t, `out = min([3, 1, 2]...) == [3, 1, 2].min()`, nil, true)
	expectError(t, `min([1], [2])`, nil, "invalid_binary_operator")
}

func TestBuiltinFunctionMax(t *testing.T) {
	expectError(t, `max()`, nil, "wrong_num_arguments")
	expectRun(t, `out = max(5)`, nil, 5)
	expectRun(t, `out = max(undefined)`, nil, core.Undefined)
	expectRun(t, `out = max(3, 1, 2)`, nil, 3)
	expectRun(t, `out = max(3.5, 1, 2)`, nil, 3.5)
	expectRun(t, `out = max("banana", "apple", "cherry")`, nil, "cherry")
	expectError(t, `max([]...)`, nil, "wrong_num_arguments")
	expectRun(t, `out = max([7]...)`, nil, 7)
	expectRun(t, `out = max([3, 1, 2]...)`, nil, 3)
	expectRun(t, `out = max([3, 1, 2]...) == [3, 1, 2].max()`, nil, true)
	expectError(t, `max([1], [2])`, nil, "invalid_binary_operator")
}

// TestBuiltinFunctionMinMaxSpreadEquality checks that min(x...) == x.min() and max(x...) == x.max() hold across
// every container type that has a .min()/.max() member function (array, bytes, runes), for the corner cases where
// the two call paths could plausibly diverge: empty (0 args after spread), singleton (1 arg, no comparison
// performed), and multiple elements with duplicates/unsorted/negative values. Only array spreads directly
// ("..." only accepts array at the VM level); bytes/runes go through .array() first.
func TestBuiltinFunctionMinMaxSpreadEquality(t *testing.T) {
	cases := []string{
		`[7]`,
		`[3, 1, 4, 1, 5, -9, 2, 6]`,
	}
	// the empty case diverges by design: a.min() answers undefined (absence is
	// data, rescuable via a.min(d)), while min() — a zero-argument selection —
	// raises, since spreading an empty array leaves nothing to select among
	expectError(t, `a := []; out = min(a...)`, nil, "wrong_num_arguments")
	expectRun(t, `a := []; out = a.min()`, nil, core.Undefined)
	for _, c := range cases {
		expectRun(t, fmt.Sprintf(`a := %s; out = min(a...) == a.min()`, c), nil, true)
		expectRun(t, fmt.Sprintf(`a := %s; out = max(a...) == a.max()`, c), nil, true)
	}

	byteCases := []string{
		`bytes("x")`,
		`bytes("banana")`,
	}
	expectError(t, `b := bytes(); out = min(b.array()...)`, nil, "wrong_num_arguments")
	for _, c := range byteCases {
		expectRun(t, fmt.Sprintf(`b := %s; out = min(b.array()...) == b.min()`, c), nil, true)
		expectRun(t, fmt.Sprintf(`b := %s; out = max(b.array()...) == b.max()`, c), nil, true)
	}

	runeCases := []string{
		`u"x"`,
		`u"héllo wörld"`,
	}
	expectError(t, `r := runes(); out = min(r.array()...)`, nil, "wrong_num_arguments")
	for _, c := range runeCases {
		expectRun(t, fmt.Sprintf(`r := %s; out = min(r.array()...) == r.min()`, c), nil, true)
		expectRun(t, fmt.Sprintf(`r := %s; out = max(r.array()...) == r.max()`, c), nil, true)
	}
}

func TestBuiltinFunctionInt(t *testing.T) {
	expectRun(t, `out = int(1)`, nil, 1)
	expectRun(t, `out = int(1.8)`, nil, 1)
	expectRun(t, `out = int("-522")`, nil, -522)
	expectRun(t, `out = int(true)`, nil, 1)
	expectRun(t, `out = int(false)`, nil, 0)
	expectRun(t, `out = int('8')`, nil, 56)
	expectError(t, `out = int([1])`, nil, "conversion: cannot convert array to int: no conversion exists")
	expectError(t, `out = int({a: 1})`, nil, "conversion: cannot convert record to int: no conversion exists")
	expectError(t, `out = int([1], 0)`, nil, "wrong_num_arguments: (int) expected 0 or 1 argument(s), got 2") // no free default form
	expectRun(t, `out = int(time(1))`, nil, 1)
	expectError(t, `out = int(undefined)`, nil, "conversion: cannot convert undefined to int: value is missing")
	expectError(t, `out = int("-522", 1)`, nil, "wrong_num_arguments") // no free default form
	expectRun(t, `out = "-522".int(1)`, nil, -522)                     // the member default is the fallible-conversion spelling
	expectRun(t, `out = "nope".int(1)`, nil, 1)
	expectRun(t, `out = undefined.int(1)`, nil, 1)     // the undefined rescue member
	expectRun(t, `out = undefined.int(1.8)`, nil, 1.8) // the default is an explicit opt-out, not type-checked
	expectRun(t, `out = undefined.int(undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionString(t *testing.T) {
	expectRun(t, `out = string(1)`, nil, "1")
	expectRun(t, `out = string(1.8)`, nil, "1.8")
	expectRun(t, `out = string("-522")`, nil, "-522")
	expectRun(t, `out = string(true)`, nil, "true")
	expectRun(t, `out = string(false)`, nil, "false")
	expectRun(t, `out = string('8')`, nil, "8")
	expectRun(t, `out = string([100, 101, 102])`, nil, "def")
	// .string() is a conversion, not the render: no text content -> raise; format() renders
	expectError(t, `out = string({b: "foo"})`, nil, "conversion: cannot convert record to string: no conversion exists")
	expectRun(t, `out = format({b: "foo"})`, nil, `{"b": "foo"}`)
	expectError(t, `out = string(undefined)`, nil, "value is missing")
	// string(x, n) is the count form (there is no free default): convert, then repeat the content
	expectRun(t, `out = string("ab", 2)`, nil, "abab")
	expectRun(t, `out = string('x', 3)`, nil, "xxx")
	expectRun(t, `out = string(65, 2)`, nil, "6565")
	expectRun(t, `out = string("ab", 0)`, nil, "")
	expectError(t, `out = string(1, "x")`, nil, "expects type int") // the second slot is a count now
	expectError(t, `out = string(1, -1)`, nil, "repeat count must be non-negative")
	expectError(t, `out = string(undefined, "-522")`, nil, "value is missing")
	// the maybe-missing rescue is the member's: undefined carries the conversion members, default-mandatory
	expectRun(t, `out = undefined.string("-522")`, nil, "-522")
	expectError(t, `out = undefined.string()`, nil, "value is missing")
}

func TestBuiltinFunctionFloat(t *testing.T) {
	expectRun(t, `out = float(1)`, nil, 1.0)
	expectRun(t, `out = float(1.8)`, nil, 1.8)
	expectRun(t, `out = float("-52.2")`, nil, -52.2)
	expectError(t, `out = float(true)`, nil, "conversion: cannot convert bool to float: no conversion exists")
	expectError(t, `out = float('8')`, nil, "conversion: cannot convert rune to float: no conversion exists")
	expectRun(t, `out = float(true.int())`, nil, 1.0) // int is the gateway
	expectError(t, `out = float([1,8.1,true,3])`, nil, "no conversion exists")
	expectError(t, `out = float({a: 1, b: "foo"})`, nil, "no conversion exists")
	expectError(t, `out = float(undefined)`, nil, "value is missing")
	expectError(t, `out = float("-52.2", 1.8)`, nil, "wrong_num_arguments") // no free default form
	expectRun(t, `out = "-52.2".float(1.8)`, nil, -52.2)
	expectRun(t, `out = undefined.float(1.8)`, nil, 1.8)         // the undefined rescue member
	expectRun(t, `out = undefined.float("-52.2")`, nil, "-52.2") // the default is an explicit opt-out, not type-checked
	expectRun(t, `out = undefined.float(undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionRune(t *testing.T) {
	expectRun(t, `out = rune(56)`, nil, '8')
	expectError(t, `out = rune(1.8)`, nil, "conversion: cannot convert float to rune: no conversion exists")
	expectError(t, `out = rune("-52.2")`, nil, "conversion: cannot convert string to rune: no conversion exists")
	expectError(t, `out = rune(true)`, nil, "no conversion exists")
	expectRun(t, `out = rune('8')`, nil, '8')
	expectError(t, `out = rune([1,8.1,true,3])`, nil, "no conversion exists")
	expectError(t, `out = rune({a: 1, b: "foo"})`, nil, "no conversion exists")
	expectError(t, `out = rune(undefined)`, nil, "value is missing")
	expectError(t, `out = rune(56, 'a')`, nil, "wrong_num_arguments") // no free default form
	expectRun(t, `out = (56).rune('a')`, nil, '8')
	expectRun(t, `out = undefined.rune('8')`, nil, '8') // the undefined rescue member
	expectRun(t, `out = undefined.rune(56)`, nil, 56)
	expectRun(t, `out = undefined.rune(undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionBool(t *testing.T) {
	// bool(x) is the CONVERSION — a numeric zero check or a text parse — not
	// truthiness; truthiness is is_true(x) / !!x, which covers every type
	expectRun(t, `out = bool(1)`, nil, true)                                           // non-zero integer
	expectRun(t, `out = bool(0)`, nil, false)                                          // zero
	expectRun(t, `out = bool(1.8)`, nil, true)                                         // zero check on floats too
	expectRun(t, `out = bool(0.0)`, nil, false)                                        // 0.0 is zero
	expectRun(t, `out = bool("false")`, nil, false)                                    // parsed boolean literal
	expectRun(t, `out = bool("true")`, nil, true)                                      // parsed boolean literal
	expectError(t, `out = bool("")`, nil, "conversion: cannot convert string to bool") // not a boolean literal
	expectRun(t, `out = bool(true)`, nil, true)
	expectRun(t, `out = bool(false)`, nil, false)
	expectError(t, `out = bool('8')`, nil, "no conversion exists")     // int is the gateway
	expectError(t, `out = bool(rune(0))`, nil, "no conversion exists") // int is the gateway
	expectError(t, `out = bool([1])`, nil, "no conversion exists")     // truthiness is is_true([1])
	expectError(t, `out = bool({})`, nil, "no conversion exists")
	expectError(t, `out = bool(undefined)`, nil, "value is missing")
	expectError(t, `out = bool(undefined, false)`, nil, "wrong_num_arguments") // no free default form
	expectRun(t, `out = undefined.bool(false)`, nil, false)                    // the undefined rescue member
}

func TestBuiltinFunctionBytes(t *testing.T) {
	// bytes(int) raises: bytes plays a double role (ASCII text vs memory chunk), so an int is ambiguous --
	// spell the octet (bytes(b'\x01')) or the text (bytes("1")); the sizing form is gone
	expectError(t, `out = bytes(1)`, nil, "conversion: cannot convert int to bytes: no conversion exists")
	expectError(t, `out = bytes(1.8)`, nil, "conversion: cannot convert float to bytes: no conversion exists")
	expectRun(t, `out = bytes("-522")`, nil, []byte{'-', '5', '2', '2'})
	expectError(t, `out = bytes(true)`, nil, "no conversion exists")
	expectError(t, `out = bytes(false)`, nil, "no conversion exists")
	expectRun(t, `out = bytes(b'\x07')`, nil, []byte{7}) // a byte is one octet
	expectRun(t, `out = bytes('8')`, nil, []byte{'8'})   // a rune converts as its UTF-8 encoding, agreeing with '8'.bytes()
	expectRun(t, `out = bytes('\u00e9')`, nil, []byte{0xC3, 0xA9})
	expectRun(t, `out = bytes([1])`, nil, []byte{1})
	expectError(t, `out = bytes({a: 1})`, nil, "no conversion exists")
	expectError(t, `out = bytes(undefined)`, nil, "value is missing")
	// bytes(x, n) is the count form (there is no free default): convert, then repeat the content
	expectRun(t, `out = bytes(b'\x00', 3)`, nil, []byte{0, 0, 0}) // the old sizing spelling, with the fill explicit
	expectRun(t, `out = bytes("ab", 2)`, nil, []byte{'a', 'b', 'a', 'b'})
	expectError(t, `out = bytes("-522", ['8'])`, nil, "expects type int")
	expectError(t, `out = bytes(undefined, "-522")`, nil, "value is missing")
	expectRun(t, `out = undefined.bytes(b"x")`, nil, []byte{'x'}) // the undefined rescue member
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
	expectRun(t, `out = type_name(bytes(b'\x01'))`, nil, "bytes")
	expectRun(t, `out = type_name(undefined)`, nil, "undefined")
	expectRun(t, `out = type_name(error("err"))`, nil, "error")
	// classification names types: all three function kinds answer "function";
	// the kind/arity detail lives in the render (format(), f-strings)
	expectRun(t, `out = type_name(func() {})`, nil, "function")
	expectRun(t, `a := func(x) { return func() { return x } }; out = type_name(a(5))`, nil, "function") // closure
	expectRun(t, `out = type_name(len)`, nil, "function")                                               // builtin
	expectRun(t, `out = format(func() {})`, nil, "<compiled-function/0>")
	expectRun(t, `out = format(len)`, nil, "<builtin-function:len/1>")
	expectRun(t, `g := func(a, b) { return a }; out = f"{g}"`, nil, "<compiled-function/2>")
	expectRun(t, `g := func(a, b) { return a }; out = g.format()`, nil, "<compiled-function/2>")
}

func TestBuiltinFunctionFormat(t *testing.T) {
	// --- argument validation ---
	expectError(t, `format()`, nil, "wrong_num_arguments: (format) expected 1 or 2 argument(s), got 0")
	expectRun(t, `out = format("x")`, nil, "x") // format(x) renders any value — a template with nothing to fill is its own rendering
	expectRun(t, `out = format(5)`, nil, "5")
	expectRun(t, `out = format({a: 1})`, nil, `{"a": 1}`)
	expectRun(t, `out = format(undefined)`, nil, "undefined")
	expectRun(t, `out = [1, 2].map(format)`, nil, ARR{"1", "2"}) // the render's callable form
	expectError(t, `format("x", [], [])`, nil, "wrong_num_arguments: (format) expected 1 or 2 argument(s), got 3")
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

// TestBuiltinFunctionDelete checks the free remove() builtin, kept alive specifically because record has no
// member functions at all (see P14/function-matrix.md) — remove() is the only way to remove a key from a
// record. remove() is pure now (P4-004/P4-005): it never mutates the receiver, works regardless of the
// receiver's mutability, and requires reassignment to see its effect on the caller's own variable, same as
// every other member/builtin in this language. See TestBuiltinFunctionDeleteInPlace for the mutating twin.
func TestBuiltinFunctionDelete(t *testing.T) {
	expectError(t, `remove()`, nil, "wrong_num_arguments: (remove) expected 2 argument(s), got 0")
	expectError(t, `remove(1)`, nil, "wrong_num_arguments: (remove) expected 2 argument(s), got 1")
	expectError(t, `remove(1, 2, 3)`, nil, "wrong_num_arguments: (remove) expected 2 argument(s), got 3")
	expectError(t, `remove({}, "", 3)`, nil, "wrong_num_arguments: (remove) expected 2 argument(s), got 3")
	expectError(t, `remove(1, 1)`, nil, `not_deletable: type int does not support delete`)
	expectError(t, `remove(1.0, 1)`, nil, `not_deletable: type float does not support delete`)
	expectError(t, `remove("str", 1)`, nil, `not_deletable: type string does not support delete`)
	expectError(t, `remove(bytes("str"), 1)`, nil, `not_deletable: type bytes does not support delete`)
	expectError(t, `remove(error("err"), 1)`, nil, `not_deletable: type error does not support delete`)
	expectError(t, `remove(true, 1)`, nil, `not_deletable: type bool does not support delete`)
	expectError(t, `remove(rune('c'), 1)`, nil, `not_deletable: type rune does not support delete`)
	expectError(t, `remove(undefined, 1)`, nil, `not_deletable: type undefined does not support delete`)
	expectError(t, `remove(time(1257894000), 1)`, nil, `not_deletable: type time does not support delete`)
	expectError(t, `remove(freeze_shallow([]), "")`, nil, `not_deletable: type immutable-array does not support delete`)
	expectError(t, `remove([], "")`, nil, `not_deletable: type array does not support delete`)
	expectError(t, `remove({}, undefined)`, nil, `invalid_index_type: (delete key) expected string, got undefined`)

	// pure: works on an immutable record/dict too, since nothing is mutated
	expectRun(t, `out = remove(freeze_shallow({a: 1}), "a")`, nil, MAP{})

	expectRun(t, `out = remove({}, "")`, nil, MAP{})
	expectRun(t, `out = {key1: 1}; out = remove(out, "key1")`, nil, MAP{})
	expectRun(t, `out = {key1: 1, key2: "2"}; out = remove(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `out = dict({key1: 1}); out = remove(out, "key1")`, nil, MAP{})
	expectRun(t, `out = dict({key1: 1, key2: "2"}); out = remove(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `out = [1, "2", {a: "b", c: 10}]; out[2] = remove(out[2], "c")`, nil, ARR{1, "2", MAP{"a": "b"}})

	// pure: the receiver itself is left untouched unless reassigned
	expectRun(t, `r := {key1: 1}; remove(r, "key1"); out = r`, nil, MAP{"key1": 1})
	expectRun(t, `d := dict({key1: 1}); remove(d, "key1"); out = d`, nil, MAP{"key1": 1})
}

// TestBuiltinFunctionDeleteInPlace checks the free remove_in_place() builtin — the mutating twin, added
// alongside pure remove() in P4-004/P4-005. Same arity/type-error surface as remove(), but mutates the
// receiver directly and rejects an immutable receiver (unlike the pure form).
func TestBuiltinFunctionDeleteInPlace(t *testing.T) {
	expectError(t, `remove_in_place()`, nil, "wrong_num_arguments: (remove_in_place) expected 2 argument(s), got 0")
	expectError(t, `remove_in_place(1, 1)`, nil, `not_deletable: type int does not support delete`)
	expectError(t, `remove_in_place([], "")`, nil, `not_deletable: type array does not support delete`)
	expectError(t, `remove_in_place({}, undefined)`, nil, `invalid_index_type: (delete key) expected string, got undefined`)
	expectError(t, `remove_in_place(freeze_shallow({}), "key")`, nil, `not_mutable: (remove_in_place) type immutable-record is immutable`)
	expectError(t, `remove_in_place(freeze_shallow(dict({})), "key")`, nil, `not_mutable: (remove_in_place) type immutable-dict is immutable`)

	expectRun(t, `out = remove_in_place({}, "")`, nil, MAP{})
	expectRun(t, `out = {key1: 1}; remove_in_place(out, "key1")`, nil, MAP{})
	expectRun(t, `out = {key1: 1, key2: "2"}; remove_in_place(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `out = dict({key1: 1}); remove_in_place(out, "key1")`, nil, MAP{})
	expectRun(t, `out = [1, "2", {a: "b", c: 10}]; remove_in_place(out[2], "c")`, nil, ARR{1, "2", MAP{"a": "b"}})

	// mutates in place: no reassignment needed to see the effect
	expectRun(t, `r := {key1: 1}; remove_in_place(r, "key1"); out = r`, nil, MAP{})
	expectRun(t, `d := dict({key1: 1}); remove_in_place(d, "key1"); out = d`, nil, MAP{})
}

// TestBuiltinFunctionFreeze checks the free freeze() builtin, added 2026-08-17 specifically because record has
// no member functions at all (see P14/function-matrix.md) — before this, a record could not be frozen by any
// spelling, member-call or free-function. freeze() is pure: detaches first (deep copy), then marks the fresh
// clone immutable throughout, so the source and any pre-existing alias are unaffected. See
// TestBuiltinFunctionFreezeShallow for its shallow, no-detach twin.
func TestBuiltinFunctionFreeze(t *testing.T) {
	expectError(t, `freeze()`, nil, "wrong_num_arguments: (freeze) expected 1 argument(s), got 0")
	expectError(t, `freeze(1, 2)`, nil, "wrong_num_arguments: (freeze) expected 1 argument(s), got 2")

	// the new capability: record can now be frozen, deeply, via the free function
	expectRun(t, `out = is_immutable(freeze({a: 1}))`, nil, true)
	expectRun(t, `r := {a: 1}; freeze(r); out = is_immutable(r)`, nil, false)            // pure: source untouched
	expectRun(t, `r := {a: [1, 2]}; f := freeze(r); out = is_immutable(f.a)`, nil, true) // deep
	expectError(t, `f := freeze({a: 1}); f.a = 2`, nil, "not_assignable")

	// same shape as the member forms for types that already had them
	expectRun(t, `a := [1, 2, 3]; f := freeze(a); a[0] = 99; out = f`, nil, ARR{1, 2, 3})
	expectRun(t, `out = is_immutable(freeze(dict({a: 1})))`, nil, true)
	expectRun(t, `out = is_immutable(freeze(5))`, nil, true) // scalars already report immutable regardless; freeze is a no-op
	expectRun(t, `out = freeze(5)`, nil, 5)
	expectRun(t, `out = freeze("x")`, nil, "x")
}

// TestBuiltinFunctionFreezeShallow checks the free freeze_shallow() builtin — freeze()'s shallow, no-detach
// twin (renamed from freeze_in_place 2026-08-17: that name wrongly implied it mutates without reassignment,
// like append_in_place/splice_in_place/delete_in_place do; it structurally can't, since `Immutable` lives on
// the `Value` header, not the shared body). It's genuinely pure — same as copy_shallow() — so, exactly like
// `x.freeze_shallow()`, the caller must reassign the result to see the effect on their own variable; a
// pre-existing sibling alias that never gets reassigned stays independently mutable, and mutating through it
// stays visible through the "frozen" variable too, since both still share the same body.
func TestBuiltinFunctionFreezeShallow(t *testing.T) {
	expectError(t, `freeze_shallow()`, nil, "wrong_num_arguments: (freeze_shallow) expected 1 argument(s), got 0")
	expectError(t, `freeze_shallow(1, 2)`, nil, "wrong_num_arguments: (freeze_shallow) expected 1 argument(s), got 2")

	// the new capability: record can now be frozen in place via the free function (requires reassignment,
	// same as the member-call form on every other type)
	expectRun(t, `r := {a: 1}; r = freeze_shallow(r); out = is_immutable(r)`, nil, true)
	expectError(t, `r := {a: 1}; r = freeze_shallow(r); r.a = 2`, nil, "not_assignable")

	// the documented danger, same as the member-call form: a pre-existing sibling alias is unaffected and its
	// mutations remain visible through the "frozen" variable too, since both still share the same body
	expectRun(t, `r := {a: 1}; s := r; r = freeze_shallow(r); out = is_immutable(s)`, nil, false)
	expectRun(t, `r := {a: 1}; s := r; r = freeze_shallow(r); s.a = 99; out = r.a`, nil, 99)

	expectRun(t, `a := [1, 2, 3]; a = freeze_shallow(a); out = is_immutable(a)`, nil, true)
	expectRun(t, `d := dict({a: 1}); d = freeze_shallow(d); out = is_immutable(d)`, nil, true)
	expectError(t, `freeze_shallow(5)`, nil, "invalid_argument_type: (freeze_shallow) argument first expects type array, dict, or record") // twins only where observable
}

// TestRetiredFreeBuiltins confirms append()/splice() were retired outright (P4-005), not deprecated: calling
// either as a free function is a compile-time error, with no fallback behavior. copy()/copy_shallow()/remove()/
// remove_in_place() were kept as free functions instead (see P4-003's decision) — confirmed still callable.
func TestRetiredFreeBuiltins(t *testing.T) {
	expectError(t, `out = append([1, 2, 3], 4)`, nil, "unresolved reference 'append'")
	expectError(t, `out = splice([1, 2, 3], 0)`, nil, "unresolved reference 'splice'")

	expectRun(t, `out = copy([1, 2, 3])`, nil, ARR{1, 2, 3})
	expectRun(t, `out = copy_shallow([1, 2, 3])`, nil, ARR{1, 2, 3})
	expectRun(t, `out = remove({a: 1}, "a")`, nil, MAP{})
	expectRun(t, `r := {a: 1}; remove_in_place(r, "a"); out = r`, nil, MAP{})
}

// TestDictRecordConversionViews checks P19's resolution: dict.record()/dict(record_val) now build an
// independent shallow copy (matching every other type's own .record()/.dict() conversion), and the new
// dict.record_view()/dict_view(record_val)/record(dict_val)/record_view(dict_val) family covers the pure
// (copy) vs _view (share) split symmetrically in both directions. record()/record_view()/dict_view() are new
// free constructors (record has no MethodCall switch, so record()/record_view() are its only spellings;
// dict_view() completes the family alongside the pre-existing dict()).
func TestDictRecordConversionViews(t *testing.T) {
	// dict.record(): now an independent SHALLOW copy — same convention as array/bytes/runes/string's own
	// .record() conversions. Top-level key set is independent; nested containers are still shared (shallow,
	// not deep) — same shape already established for copy_shallow().
	expectRun(t, `d := dict({a: 1}); r := d.record(); r.b = 2; out = d`, nil, MAP{"a": 1}) // top level independent
	expectRun(t, `d := dict({a: 1}); r := d.record(); d["c"] = 3; out = r`, nil, MAP{"a": 1})
	expectRun(t, `d := dict({a: [1, 2]}); r := d.record(); r.a[0] = 99; out = d["a"][0]`, nil, 99) // nested shared
	expectRun(t, `out = dict({a: 1, b: 2}).record()`, nil, MAP{"a": 1, "b": 2})

	// dict.record_view(): the explicit full-sharing opt-in — today's original dict.record() behavior. Top
	// level and nested are both the same underlying map, so both directions of mutation are visible.
	expectRun(t, `d := dict({a: 1}); r := d.record_view(); r.b = 2; out = d`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `d := dict({a: 1}); r := d.record_view(); d["c"] = 3; out = r`, nil, MAP{"a": 1, "c": 3})
	expectRun(t, `d := dict({a: [1, 2]}); r := d.record_view(); r.a[0] = 99; out = d["a"][0]`, nil, 99)
	expectRun(t, `out = type_name(freeze_shallow(dict({a: 1})).record_view())`, nil, "immutable-record")
	expectRun(t, `out = type_name(freeze_shallow(dict({a: 1})).record())`, nil, "record") // copy: always mutable

	// free dict(record_val): now an independent shallow copy too (P19 fix; same shape as .record() above,
	// opposite direction)
	expectRun(t, `r := {a: 1}; d := dict(r); d["b"] = 2; out = r`, nil, MAP{"a": 1})
	expectRun(t, `r := {a: [1, 2]}; d := dict(r); d["a"][0] = 99; out = r.a[0]`, nil, 99) // nested still shared

	// free dict_view(record_val): the sharing opt-in for the free constructor
	expectRun(t, `r := {a: 1}; d := dict_view(r); d["b"] = 2; out = r`, nil, MAP{"a": 1, "b": 2})

	// free record(dict_val) / record_view(dict_val): same copy-vs-share split, other direction — same
	// operation as dict_val.record()/dict_val.record_view(), just the free-constructor spelling
	expectRun(t, `d := dict({a: 1}); r := record(d); r.b = 2; out = d`, nil, MAP{"a": 1})
	expectRun(t, `d := dict({a: 1}); r := record_view(d); r.b = 2; out = d`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `out = record(dict({a: 1})) == dict({a: 1}).record()`, nil, true)

	// identity cases: converting to your own type is a no-op, same as dict()/array()/etc.
	expectRun(t, `d := dict({a: 1}); out = dict(d) == d`, nil, true)
	expectRun(t, `d := dict({a: 1}); out = dict_view(d) == d`, nil, true)
	expectRun(t, `r := {a: 1}; out = record(r) == r`, nil, true)
	expectRun(t, `r := {a: 1}; out = record_view(r) == r`, nil, true)

	// zero-arg construction
	expectRun(t, `out = record()`, nil, MAP{})
	expectRun(t, `out = record_view()`, nil, MAP{})
	expectRun(t, `out = dict_view()`, nil, MAP{})

	// arity/type errors, mirroring dict()'s existing shape
	expectError(t, `record(1)`, nil, "conversion: cannot convert int to record: no conversion exists")
	expectError(t, `record_view(1)`, nil, "invalid_argument_type: (record_view) argument first expects type dict or record, got int")
	expectError(t, `dict_view(1)`, nil, "invalid_argument_type: (dict_view) argument first expects type dict or record, got int")
	expectError(t, `record(1, 2)`, nil, "wrong_num_arguments: (record) expected 0 or 1 argument(s), got 2") // no free default form
	expectError(t, `record_view(1, 2)`, nil, "wrong_num_arguments: (record_view) expected 0 or 1 argument(s), got 2")
	expectError(t, `dict_view(1, 2)`, nil, "wrong_num_arguments: (dict_view) expected 0 or 1 argument(s), got 2")
	expectError(t, `dict({}).record_view(1)`, nil, "wrong_num_arguments")
}

// TestMemberFunctionSpliceInPlace checks array.splice_in_place() — the mutating twin, renamed from what used
// to be the only splice()/free splice() behavior (P4-004/P4-005: splice is member-only now, and splice() itself
// is pure — see TestMemberFunctionAppendDeleteSplice for that side). The "argument first expects type array"
// error family from the old free-function form is gone: a member receiver is already guaranteed to be an array
// by dispatch, so that failure mode isn't reachable this way anymore.
func TestMemberFunctionSpliceInPlace(t *testing.T) {
	expectError(t, `[].splice_in_place("str")`, nil, `invalid_argument_type: (splice) argument second expects type int, got string`)
	expectError(t, `[].splice_in_place(bytes("str"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got bytes`)
	expectError(t, `[].splice_in_place(error("error"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got error`)
	expectError(t, `[].splice_in_place(undefined)`, nil, `invalid_argument_type: (splice) argument second expects type int, got undefined`)
	expectError(t, `[].splice_in_place([])`, nil, `invalid_argument_type: (splice) argument second expects type int, got array`)
	expectError(t, `[].splice_in_place({})`, nil, `invalid_argument_type: (splice) argument second expects type int, got record`)
	expectError(t, `[].splice_in_place(freeze_shallow([]))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-array`)
	expectError(t, `[].splice_in_place(freeze_shallow({}))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-record`)
	expectError(t, `[].splice_in_place(0, "string")`, nil, `invalid_argument_type: (splice) argument third expects type int, got string`)
	expectError(t, `[].splice_in_place(0, bytes("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got bytes`)
	expectError(t, `[].splice_in_place(0, error("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got error`)
	expectError(t, `[].splice_in_place(0, undefined)`, nil, `invalid_argument_type: (splice) argument third expects type int, got undefined`)
	expectError(t, `[].splice_in_place(0, [])`, nil, `invalid_argument_type: (splice) argument third expects type int, got array`)
	expectError(t, `[].splice_in_place(0, {})`, nil, `invalid_argument_type: (splice) argument third expects type int, got record`)
	expectError(t, `[].splice_in_place(0, freeze_shallow([]))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-array`)
	expectError(t, `[].splice_in_place(0, freeze_shallow({}))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-record`)
	expectError(t, `[].splice_in_place(1)`, nil, "index_out_of_bounds")
	expectError(t, `[1, 2, 3].splice_in_place(0, -1)`, nil, "invalid_value: splice delete count must be non-negative")
	expectError(t, `[1, 2, 3].splice_in_place(99, 0, "a", "b")`, nil, "index_out_of_bounds")
	expectError(t, `freeze_shallow([1, 2, 3]).splice_in_place(0)`, nil,
		`not_mutable: (splice_in_place) type immutable-array is immutable`)

	// splice_in_place returns the RECEIVER (side-effecting members chain and the
	// twins correspond); the removed run is x.slice(i, j) taken beforehand
	expectRun(t, `out = []; out.splice_in_place()`, nil, ARR{})
	expectRun(t, `out = ["a"]; out.splice_in_place(1)`, nil, ARR{"a"})
	expectRun(t, `out = ["a"]; res := out.splice_in_place(1); out = res`, nil, ARR{"a"})
	expectRun(t, `out = [1, 2, 3]; out.splice_in_place(0, 1)`, nil, ARR{2, 3})
	expectRun(t, `out = [1, 2, 3]; res := out.splice_in_place(0, 1); out = res == [2, 3]`, nil, true)
	expectRun(t, `a := [1, 2, 3]; removed := a.slice(0, 1); a.splice_in_place(0, 1); out = removed`, nil, ARR{1})
	expectRun(t, `out = [1, 2, 3]; out.splice_in_place(0, 0, "a", "b")`, nil, ARR{"a", "b", 1, 2, 3})
	expectRun(t, `out = [1, 2, 3]; out.splice_in_place(1, 0, "a", "b")`, nil, ARR{1, "a", "b", 2, 3})
	expectRun(t, `out = [1, 2, 3]; out.splice_in_place(2, 0, "a", "b")`, nil, ARR{1, 2, "a", "b", 3})
	expectRun(t, `out = [1, 2, 3]; out.splice_in_place(3, 0, "a", "b")`, nil, ARR{1, 2, 3, "a", "b"})

	expectRun(t, `array := [1, 2, 3]; res := array.splice_in_place(1, 1, "a", "b");
				out = [res, array]`, nil, ARR{ARR{1, "a", "b", 3}, ARR{1, "a", "b", 3}})

	expectRun(t, `array := [1, 2, 3]; res := array.splice_in_place(1);
		out = [res, array]`, nil, ARR{ARR{1}, ARR{1}})

	// splice_in_place doc examples: it returns the receiver
	expectRun(t, `v := [1, 2, 3]; res := v.splice_in_place(0);
		out = [res, v]`, nil, ARR{ARR{}, ARR{}})

	expectRun(t, `v := [1, 2, 3]; res := v.splice_in_place(1);
		out = [res, v]`, nil, ARR{ARR{1}, ARR{1}})

	expectRun(t, `v := [1, 2, 3]; res := v.splice_in_place(0, 1);
		out = [res, v]`, nil, ARR{ARR{2, 3}, ARR{2, 3}})

	expectRun(t, `v := ["a", "b", "c"]; res := v.splice_in_place(1, 2);
		out = [res, v]`, nil, ARR{ARR{"a"}, ARR{"a"}})

	expectRun(t, `v := ["a", "b", "c"]; res := v.splice_in_place(2, 1, "d");
		out = [res, v]`, nil, ARR{ARR{"a", "b", "d"}, ARR{"a", "b", "d"}})

	expectRun(t, `v := ["a", "b", "c"]; res := v.splice_in_place(0, 0, "d", "e");
		out = [res, v]`, nil, ARR{ARR{"d", "e", "a", "b", "c"}, ARR{"d", "e", "a", "b", "c"}})

	expectRun(t, `v := ["a", "b", "c"]; res := v.splice_in_place(1, 1, "d", "e");
		out = [res, v]`, nil, ARR{ARR{"a", "d", "e", "c"}, ARR{"a", "d", "e", "c"}})
}

// TestMemberFunctionAppendDeleteSplice checks that the new member-call spellings for append/delete/splice
// (P3-001) produce results identical to the existing builtin forms — a new spelling, no behavior change. The
// append() cases were rewritten for P12/P5-001: append() is now pure/copy-default on all three types (was
// Go-style/capacity-dependent for array only), 0 items is a legal no-op that still returns an independent copy,
// and works regardless of the receiver's mutability — see TestMemberFunctionAppendInPlace for the mutating twin.
func TestMemberFunctionAppendDeleteSplice(t *testing.T) {
	// append: array/bytes/runes
	expectRun(t, `a := [1, 2, 3]; out = a.append(4) == a.append(4)`, nil, true)
	expectRun(t, `a := [1, 2, 3]; out = a.append(4, 5, 6) == a.append(4, 5, 6)`, nil, true)
	expectRun(t, `out = bytes("ab").append('c') == bytes("ab").append('c')`, nil, true)
	expectRun(t, `out = runes("ab").append('c') == runes("ab").append('c')`, nil, true)
	expectRun(t, `a := [1, 2, 3]; a.append(4); out = a`, nil, ARR{1, 2, 3}) // pure: never mutates the receiver

	expectRun(t, `out = [1, 2, 3].append()`, nil, ARR{1, 2, 3}) // 0 items: legal no-op, still returns a copy
	expectRun(t, `out = bytes("ab").append()`, nil, []byte("ab"))
	expectRun(t, `out = runes("ab").append()`, nil, []rune("ab"))
	expectRun(t, `a := [1, 2, 3]; b := a.append(); b[0] = 99; out = a`, nil, ARR{1, 2, 3}) // 0 items still detaches
	expectRun(t, `out = freeze_shallow([1, 2, 3]).append(4)`, nil, ARR{1, 2, 3, 4})        // pure: works on immutable too
	expectRun(t, `out = freeze(bytes("ab")).append('c')`, nil, []byte("abc"))
	expectError(t, `bytes("ab").append({})`, nil, "invalid_argument_type")
	expectError(t, `runes("ab").append({})`, nil, "invalid_argument_type")

	// the add side's three readings on array (member ≡ + operator): an argument of the receiver's
	// OWN KIND — another array — is a run and spreads; every other value is one element; the element
	// spelling for a nested array is the wrap — or push, which never spreads
	expectRun(t, `out = [1, 2].append([3, 4])`, nil, ARR{1, 2, 3, 4})                          // own kind: spreads
	expectRun(t, `out = [1, 2].append("ab")`, nil, ARR{1, 2, "ab"})                            // cross-family: one element
	expectRun(t, `out = [1, 2].append([[3, 4]])`, nil, ARR{1, 2, ARR{3, 4}})                   // the wrap
	expectRun(t, `out = [9].append(range(1, 4))`, nil, ARR{9, core.NewIntRangeValue(1, 4, 1)}) // range: one element like any non-array
	expectRun(t, `out = [9].append(range(1, 4).array())`, nil, ARR{9, 1, 2, 3})                // materializing is spelled at the call site
	expectRun(t, `a := [1, 2]; a.append_in_place([3, 4]); out = a`, nil, ARR{1, 2, 3, 4})      // the twin agrees
	expectRun(t, `out = [1, 2].append([3], 4, [5, 6])`, nil, ARR{1, 2, 3, 4, 5, 6})            // operands in order

	// on the text triple every accepted argument is text content, encoded into the receiver's
	// representation; element and run operands mix freely on the add side (x.append("ab", 'c') ≡ x + "ab" + 'c')
	expectRun(t, `out = bytes("ab").append("cd", 'x')`, nil, []byte("abcdx"))
	expectRun(t, `out = bytes("ab").append(99)`, nil, []byte("abc")) // an in-range int is one octet
	expectRun(t, `out = runes("ab").append("cd", 'x')`, nil, []rune("abcdx"))
	expectError(t, `bytes("ab").append(1.5)`, nil, "invalid_argument_type") // no fractional reading of text content

	// string carries the whole add side too, unsuffixed only (immutable by construction)
	expectRun(t, `out = "ab".append("cd", 'x')`, nil, "abcdx")
	expectRun(t, `out = "ab".append(bytes("cd"))`, nil, "abcd") // valid-UTF-8 bytes are text content
	expectRun(t, `out = "ab".append(99)`, nil, "abc")           // a valid code point is one symbol
	expectError(t, `"ab".append_in_place("c")`, nil, "type string has no method append_in_place")

	// range has no add member at all: a lazy sequence never answers a new sequence of its own elements
	expectError(t, `range(0, 3).append(1)`, nil, "type range has no method append")

	// delete: dict (record intentionally out of scope — record has no MethodCall switch at all, so it has no
	// member-call form of delete). remove() is pure: never mutates the receiver, works regardless of the
	// receiver's mutability. remove_in_place() is the mutating twin.
	expectRun(t, `d := dict({key1: 1, key2: "2"}); out = d.remove("key1")`, nil, MAP{"key2": "2"})
	expectRun(t, `d := dict({key1: 1}); d.remove("key1"); out = d`, nil, MAP{"key1": 1}) // pure: receiver untouched
	expectRun(t, `d := dict({key1: 1}); d.remove_in_place("key1"); out = d`, nil, MAP{}) // _in_place: mutates
	expectRun(t, `out = freeze_shallow(dict({a: 1})).remove("a")`, nil, MAP{})           // pure: works on immutable too
	expectError(t, `dict({}).remove()`, nil, "wrong_num_arguments")
	expectRun(t, `out = dict({a: 1, b: 2, c: 3}).remove("a", "b")`, nil, MAP{"c": 3}) // variadic key set
	expectRun(t, `out = dict({a: 1, b: 2}).remove(k => k == "a")`, nil, MAP{"b": 2})  // predicate
	expectError(t, `dict({a: 1}).remove(undefined)`, nil, "invalid_argument_type")
	expectError(t, `freeze_shallow(dict({a: 1})).remove_in_place("a")`, nil, "not_mutable")
	expectError(t, `{}.remove("x")`, nil, "type record has no method remove")
	expectError(t, `{}.remove_in_place("x")`, nil, "type record has no method remove_in_place")

	// splice: array (bytes/runes generalized in P5-002 — see TestMemberFunctionSpliceBytesRunes). splice() is
	// pure now (P4-004/P4-005): returns the modified array, doesn't mutate the receiver, doesn't return the
	// deleted items, and works regardless of the receiver's mutability. splice_in_place() is the mutating twin,
	// returning deleted items — see TestMemberFunctionSpliceInPlace.
	expectRun(t, `v := [1, 2, 3]; result := v.splice(0, 1); out = [result, v]`, nil, ARR{ARR{2, 3}, ARR{1, 2, 3}})
	expectRun(t, `v := [1, 2, 3]; result := v.splice(1, 0, "a", "b");
		out = [result, v]`, nil, ARR{ARR{1, "a", "b", 2, 3}, ARR{1, 2, 3}})
	expectRun(t, `out = [1, 2, 3].splice(0, 1) == [1, 2, 3].splice_in_place(0, 1)`, nil, true) // the twins correspond now
	expectRun(t, `out = freeze_shallow([1, 2, 3]).splice(0, 1)`, nil, ARR{2, 3})               // pure: works on immutable too
	expectError(t, `[1, 2, 3].splice(0, -1)`, nil, "invalid_value: splice delete count must be non-negative")
	expectError(t, `[1, 2, 3].splice(99)`, nil, "index_out_of_bounds")
}

// TestMemberFunctionAppendInPlace checks append_in_place() (P12/P5-001) — the mutating twin added alongside
// append()'s new pure/copy-default behavior on all three Seq-shaped types. It mutates the receiver's own shared
// struct directly (via Seq.Set), so the mutation is visible through every existing alias without needing
// reassignment — unlike freeze_shallow(), whose Immutable flag lives on the Value header rather than behind
// Ptr (see TestMemberFunctionFreeze). Today's old array.append's capacity-dependent reuse-vs-reallocate detail
// is Go-internal and not itself asserted here (it's not part of the language contract) — only the guaranteed,
// deterministic behavior (shared-struct mutation, immutable-receiver rejection, 0-arg no-op) is.
func TestMemberFunctionAppendInPlace(t *testing.T) {
	// array
	expectRun(t, `a := [1, 2, 3]; a.append_in_place(4); out = a`, nil, ARR{1, 2, 3, 4})
	expectRun(t, `a := [1, 2, 3]; b := a; a.append_in_place(4); out = b`, nil, ARR{1, 2, 3, 4})  // shared struct: b sees it too
	expectRun(t, `a := [1, 2, 3]; out = a.append_in_place(4, 5, 6)`, nil, ARR{1, 2, 3, 4, 5, 6}) // returns the (now-mutated) receiver
	expectRun(t, `a := [1, 2, 3]; a.append_in_place(); out = a`, nil, ARR{1, 2, 3})              // 0 items: true no-op
	expectError(t, `freeze_shallow([1, 2, 3]).append_in_place(4)`, nil,
		"not_mutable: (append_in_place) type immutable-array is immutable")

	// bytes — genuinely new capability (P12): no mutating/sharing append form existed for bytes before this
	expectRun(t, `a := bytes("ab"); a.append_in_place('c'); out = a`, nil, []byte("abc"))
	expectRun(t, `a := bytes("ab"); b := a; a.append_in_place('c'); out = b`, nil, []byte("abc"))
	expectRun(t, `a := bytes("ab"); a.append_in_place(); out = a`, nil, []byte("ab"))
	expectError(t, `freeze(bytes("ab")).append_in_place('c')`, nil,
		"not_mutable: (append_in_place) type immutable-bytes is immutable")
	expectError(t, `bytes("ab").append_in_place({})`, nil, "invalid_argument_type")

	// runes — same, genuinely new capability
	expectRun(t, `a := runes("ab"); a.append_in_place('c'); out = a`, nil, []rune("abc"))
	expectRun(t, `a := runes("ab"); b := a; a.append_in_place('c'); out = b`, nil, []rune("abc"))
	expectRun(t, `a := runes("ab"); a.append_in_place(); out = a`, nil, []rune("ab"))
	expectError(t, `freeze(runes("ab")).append_in_place('c')`, nil,
		"not_mutable: (append_in_place) type immutable-runes is immutable")
	expectError(t, `runes("ab").append_in_place({})`, nil, "invalid_argument_type")
}

// TestMemberFunctionPushPrepend checks the rest of the add side: prepend (whole-operand concatenation at the
// front, arguments staying in order — x.prepend(a, b) ≡ a + b + x) and push/push_first (each argument is ONE
// element whatever its type — the spelling that never spreads; on the text triple they VALIDATE: a sequence
// argument raises even at length 1). push/push_first are not a twin pair — two operations at two ends; the
// twins are push/push_in_place and push_first/push_first_in_place.
func TestMemberFunctionPushPrepend(t *testing.T) {
	// prepend: array — same three readings as append, at the front
	expectRun(t, `out = [1].prepend([2, 3])`, nil, ARR{2, 3, 1})                                // own kind: spreads
	expectRun(t, `out = [1].prepend(2, 3)`, nil, ARR{2, 3, 1})                                  // arguments in order at the front
	expectRun(t, `out = [1].prepend("ab")`, nil, ARR{"ab", 1})                                  // cross-family: one element
	expectRun(t, `out = [9].prepend(range(1, 3))`, nil, ARR{core.NewIntRangeValue(1, 3, 1), 9}) // range: one element like any non-array
	expectRun(t, `out = [1].prepend()`, nil, ARR{1})                                            // 0 items: legal no-op
	expectRun(t, `a := [1]; a.prepend(2); out = a`, nil, ARR{1})                                // pure: receiver untouched
	expectRun(t, `a := [1]; b := a; a.prepend_in_place(2, 3); out = b`, nil, ARR{2, 3, 1})      // twin: shared struct
	expectRun(t, `a := [1]; out = a.prepend_in_place(2)`, nil, ARR{2, 1})                       // twin returns the receiver
	expectError(t, `freeze_shallow([1]).prepend_in_place(2)`, nil, "not_mutable")

	// prepend: the text triple + string (unsuffixed only on string)
	expectRun(t, `out = bytes("ab").prepend("cd", 'x')`, nil, []byte("cdxab"))
	expectRun(t, `out = runes("ab").prepend('x')`, nil, []rune("xab"))
	expectRun(t, `out = "ab".prepend("cd", 'x')`, nil, "cdxab")
	expectRun(t, `a := bytes("ab"); b := a; a.prepend_in_place(b'x'); out = b`, nil, []byte("xab"))
	expectRun(t, `a := runes("ab"); a.prepend_in_place('x'); out = a`, nil, []rune("xab"))
	expectError(t, `freeze(bytes("ab")).prepend_in_place(b'x')`, nil, "not_mutable")
	expectError(t, `"ab".prepend_in_place("c")`, nil, "type string has no method prepend_in_place")

	// push: each argument is one element whatever its type — the postconditions are the contract
	expectRun(t, `out = [1, 2].push([3, 4])`, nil, ARR{1, 2, ARR{3, 4}}) // never spreads
	expectRun(t, `out = [1].push([9]).last() == [9]`, nil, true)         // a.push(x) ⟹ a.last() == x
	expectRun(t, `a := [1].push(range(1, 3)); out = a.len()`, nil, 2)    // a range is one element here too
	expectRun(t, `out = [1].push(dict({a: 1})).len()`, nil, 2)
	expectRun(t, `out = [1, 2].push()`, nil, ARR{1, 2}) // 0 items: legal no-op
	expectRun(t, `out = [1].push(2, 3)`, nil, ARR{1, 2, 3})
	expectRun(t, `out = [1].push_first([9]).first() == [9]`, nil, true)                  // a.push_first(x) ⟹ a.first() == x
	expectRun(t, `out = [1].push_first(2, 3)`, nil, ARR{2, 3, 1})                        // arguments in order at the front
	expectRun(t, `a := [1]; a.push(2); out = a`, nil, ARR{1})                            // pure: receiver untouched
	expectRun(t, `a := [1]; b := a; a.push_in_place([2]); out = b`, nil, ARR{1, ARR{2}}) // twin: shared struct
	expectRun(t, `a := [1]; out = a.push_first_in_place(2, 3)`, nil, ARR{2, 3, 1})       // twin returns the receiver
	expectError(t, `freeze_shallow([1]).push_in_place(2)`, nil, "not_mutable")
	expectError(t, `freeze_shallow([1]).push_first_in_place(2)`, nil, "not_mutable")

	// push on the text triple VALIDATES: element type only — a sequence argument raises even at
	// length 1 (the refusal is the member's purpose), and so does an element that widens
	expectRun(t, `out = bytes("ab").push(b'c', 99)`, nil, []byte("abc"+string(byte(99))))
	expectRun(t, `out = bytes("ab").push('c')`, nil, []byte("abc"))              // an ASCII rune is one octet
	expectError(t, `bytes("ab").push(bytes("c"))`, nil, "invalid_argument_type") // sequence, even length 1
	expectError(t, `bytes("ab").push("c")`, nil, "invalid_argument_type")
	expectError(t, `bytes("ab").push('é')`, nil, "invalid_value") // two octets do not fit one element
	expectRun(t, `out = runes("ab").push('c')`, nil, []rune("abc"))
	expectError(t, `runes("ab").push(runes("c"))`, nil, "invalid_argument_type")
	expectRun(t, `out = runes("ab").push_first('c')`, nil, []rune("cab"))
	expectRun(t, `a := bytes("ab"); a.push_in_place(b'c'); out = a`, nil, []byte("abc"))
	expectRun(t, `a := runes("ab"); a.push_first_in_place('c'); out = a`, nil, []rune("cab"))
	expectError(t, `freeze(bytes("ab")).push_in_place(b'c')`, nil, "not_mutable")

	// string: unsuffixed only
	expectRun(t, `out = "ab".push('c')`, nil, "abc")
	expectRun(t, `out = "ab".push_first('c')`, nil, "cab")
	expectError(t, `"ab".push("c")`, nil, "invalid_argument_type")
	expectError(t, `"ab".push_in_place('c')`, nil, "type string has no method push_in_place")
}

// TestMemberFunctionMergeRemoveInPlace checks dict's whole add side — merge/merge_in_place, variadic over maps
// (dict and record), entries applied in argument order with last-wins on key collision, exactly the + operator's
// rule — and the remove_in_place twins on the sequence types and dict, which run remove's own dispatch (blank
// set / element / run / key set / predicate) and apply it to the receiver, returning the receiver.
func TestMemberFunctionMergeRemoveInPlace(t *testing.T) {
	// merge: pure form
	expectRun(t, `out = dict({a: 1}).merge(dict({b: 2}))`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `out = dict({a: 1}).merge({b: 2})`, nil, MAP{"a": 1, "b": 2})             // a record is the map family too
	expectRun(t, `out = dict({a: 1}).merge(dict({a: 2}), dict({a: 3}))`, nil, MAP{"a": 3}) // last wins, in argument order
	expectRun(t, `out = dict({a: 1}).merge()`, nil, MAP{"a": 1})                           // 0 maps: legal no-op
	expectRun(t, `d := dict({a: 1}); d.merge(dict({b: 2})); out = d`, nil, MAP{"a": 1})    // pure: receiver untouched
	expectRun(t, `out = freeze_shallow(dict({a: 1})).merge(dict({b: 2}))`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `out = dict({a: 1}).merge(dict([["x", 9]]))`, nil, MAP{"a": 1, "x": 9}) // the non-mutating one-entry spelling
	expectError(t, `dict({}).merge(1)`, nil, "invalid_argument_type")
	expectError(t, `dict({}).merge([["a", 1]])`, nil, "invalid_argument_type") // entries need the constructor, not a bare array

	// merge_in_place: mutates the receiver's own map, returns the receiver
	expectRun(t, `d := dict({a: 1}); out = d.merge_in_place(dict({b: 2}))`, nil, MAP{"a": 1, "b": 2})
	expectRun(t, `d := dict({a: 1}); e := d; d.merge_in_place(dict({b: 2})); out = e`, nil, MAP{"a": 1, "b": 2})
	expectError(t, `freeze_shallow(dict({a: 1})).merge_in_place(dict({b: 2}))`, nil, "not_mutable")
	expectError(t, `{}.merge({a: 1})`, nil, "type record has no method merge") // record has no member surface

	// remove_in_place on the sequence types: remove's dispatch, applied to the receiver
	expectRun(t, `a := [1, 2, 1]; out = a.remove_in_place(1)`, nil, ARR{2})               // returns the receiver
	expectRun(t, `a := [1, 2, 1]; b := a; a.remove_in_place(1); out = b`, nil, ARR{2})    // shared struct
	expectRun(t, `a := [1, 2, 3]; a.remove_in_place(x => x > 1); out = a`, nil, ARR{1})   // predicate
	expectRun(t, `a := [0, 1, 0]; a.remove_in_place(); out = a`, nil, ARR{1})             // no-arg drops the blanks
	expectRun(t, `a := [1, 2, 3, 2, 3]; a.remove_in_place([2, 3]); out = a`, nil, ARR{1}) // run reading
	expectError(t, `freeze_shallow([1]).remove_in_place(1)`, nil, "not_mutable")
	expectRun(t, `a := bytes("aab"); a.remove_in_place(b'a'); out = a`, nil, []byte("b"))
	expectRun(t, `a := runes("aab"); a.remove_in_place('a'); out = a`, nil, []rune("b"))
	expectRun(t, `a := runes("xaby"); a.remove_in_place("ab"); out = a`, nil, []rune("xy")) // run reading
	expectError(t, `freeze(bytes("a")).remove_in_place(b'a')`, nil, "not_mutable")
	expectError(t, `freeze(runes("a")).remove_in_place('a')`, nil, "not_mutable")

	// remove_in_place on dict mirrors remove: key set or predicate, returning the receiver
	expectRun(t, `d := dict({a: 1, b: 2, c: 3}); out = d.remove_in_place("a", "b")`, nil, MAP{"c": 3})
	expectRun(t, `d := dict({a: 1, b: 2}); e := d; d.remove_in_place(k => k == "a"); out = e`, nil, MAP{"b": 2})
	expectError(t, `dict({a: 1}).remove_in_place()`, nil, "wrong_num_arguments") // no blank reading on a map
}

// TestMemberFunctionTextStructural checks the text-structural family — the trim family, the anchored pair
// (has_/remove_ prefix/suffix), replace, and the pads — as SEQUENCE members: array carries every one of them
// alongside the text triple, with the same readings, sets, and match semantics, plus _in_place twins on the
// mutable receivers (array/bytes/runes; string is immutable by construction and carries the unsuffixed forms
// only). split/partition stay the triple's own: a lazy or general sequence has other spellings.
func TestMemberFunctionTextStructural(t *testing.T) {
	// trim: element set, repeat-while, both ends; no-arg = the blank set
	expectRun(t, `out = "xxabcx".trim('x')`, nil, "abc")
	expectRun(t, `out = "xyaby".trim('x', 'y')`, nil, "ab")
	expectRun(t, `out = "  ab  ".trim_start()`, nil, "ab  ")
	expectRun(t, `out = "  ab  ".trim_end()`, nil, "  ab")
	expectRun(t, `out = bytes("  ab ").trim()`, nil, []byte("ab"))
	expectRun(t, `out = u"xab".trim('x')`, nil, []rune("ab"))
	expectRun(t, `out = [0, undefined, 5, 0].trim()`, nil, ARR{5})     // array's blank set: undefined ∪ the zero
	expectRun(t, `out = [0, 5, 0].trim(undefined)`, nil, ARR{0, 5, 0}) // "zeros are data": name your own set
	expectRun(t, `out = [9, 1, 9, 9].trim(9)`, nil, ARR{1})
	expectError(t, `[1, 2].trim([1])`, nil, "invalid_argument_type")               // a run points at remove_prefix/remove_suffix
	expectError(t, `"ab".trim(x => true)`, nil, "invalid_argument_type")           // no predicate reading
	expectRun(t, `a := bytes("  ab"); out = a.trim_in_place()`, nil, []byte("ab")) // twin returns the receiver
	expectRun(t, `a := [9, 1]; b := a; a.trim_start_in_place(9); out = b`, nil, ARR{1})
	expectError(t, `freeze_shallow([9, 1]).trim_in_place(9)`, nil, "not_mutable")
	expectError(t, `"ab".trim_in_place('a')`, nil, "type string has no method trim_in_place")

	// has_prefix / has_suffix: element | run | variadic run set (any-of); no predicate, no absent
	expectRun(t, `out = "http://x".has_prefix("http://", "https://")`, nil, true)
	expectRun(t, `out = "abc".has_prefix('a')`, nil, true)
	expectRun(t, `out = "abc".has_suffix("bc")`, nil, true)
	expectRun(t, `out = bytes("abc").has_suffix(b'c')`, nil, true)
	expectRun(t, `out = u"abc".has_prefix(u"ab")`, nil, true)
	expectRun(t, `out = [1, 2, 3].has_prefix([1, 2])`, nil, true)              // a run on array is its own family
	expectRun(t, `out = [1, 2, 3].has_prefix(1)`, nil, true)                   // element form
	expectRun(t, `out = [1, 2, 3].has_suffix(range(2, 4))`, nil, false)        // range: one element, and no element equals it
	expectRun(t, `out = [1, 2, 3].has_suffix(range(2, 4).array())`, nil, true) // the run reading is spelled by materializing
	expectRun(t, `out = "abc".has_prefix("")`, nil, true)                      // the empty run is anchored everywhere
	expectError(t, `"abc".has_prefix()`, nil, "wrong_num_arguments")
	expectError(t, `"abc".has_prefix(x => true)`, nil, "invalid_argument_type") // index(f) == 0 is the spelling

	// remove_prefix / remove_suffix: one exact anchored run, ONCE; absent → unchanged; longest in a set wins
	expectRun(t, `out = "xxab".remove_prefix("xx")`, nil, "ab")
	expectRun(t, `out = "xxxxab".remove_prefix("xx")`, nil, "xxab") // once, not repeat-while
	expectRun(t, `out = "ab.txt".remove_suffix(".txt")`, nil, "ab")
	expectRun(t, `out = "ab".remove_prefix("zz")`, nil, "ab")         // absent → unchanged
	expectRun(t, `out = "abcd".remove_prefix("ab", "abc")`, nil, "d") // the longest matching run wins
	expectRun(t, `out = [1, 2, 3].remove_prefix([1, 2])`, nil, ARR{3})
	expectRun(t, `out = bytes("xab").remove_prefix(b'x')`, nil, []byte("ab"))
	expectRun(t, `a := runes("xab"); a.remove_prefix_in_place(u"x"); out = a`, nil, []rune("ab"))
	expectError(t, `freeze(bytes("xa")).remove_suffix_in_place(b'a')`, nil, "not_mutable")

	// replace: element or run in both positions, every occurrence, leftmost non-overlapping
	expectRun(t, `out = "a-b-c".replace('-', '+')`, nil, "a+b+c")
	expectRun(t, `out = "ab ab".replace("ab", "xyz")`, nil, "xyz xyz") // cross-length
	expectRun(t, `out = "aaa".replace("aa", "b")`, nil, "ba")          // non-overlapping
	expectRun(t, `out = "abc".replace("b", "")`, nil, "ac")            // removal spelling
	expectRun(t, `out = "abc".replace("", "x")`, nil, "abc")           // an empty old run matches nothing
	expectRun(t, `out = bytes("a-b").replace(b'-', b'+')`, nil, []byte("a+b"))
	expectRun(t, `out = u"a-b".replace('-', "--")`, nil, []rune("a--b"))
	expectRun(t, `out = [1, 0, 2].replace(0, 9)`, nil, ARR{1, 9, 2})
	expectRun(t, `out = [1, 2, 3].replace([2, 3], [9])`, nil, ARR{1, 9})            // run→run on array
	expectRun(t, `out = [1, 2, 3].replace(range(2, 4), 0)`, nil, ARR{1, 2, 3})      // a range is one element: no match
	expectRun(t, `out = [1, 2, 3].replace(range(2, 4).array(), 0)`, nil, ARR{1, 0}) // the run reading, spelled
	expectRun(t, `a := [1, 0]; b := a; a.replace_in_place(0, 9); out = b`, nil, ARR{1, 9})
	expectError(t, `"ab".replace("a")`, nil, "wrong_num_arguments")              // never variadic: position 2 is the replacement
	expectError(t, `"ab".replace(x => true, "y")`, nil, "invalid_argument_type") // never a predicate
	expectError(t, `freeze_shallow([1]).replace_in_place(1, 2)`, nil, "not_mutable")

	// pads: element width, one-element fill, default = the blank set's canonical member; short width = no-op
	expectRun(t, `out = "ab".pad_start(4)`, nil, "  ab")
	expectRun(t, `out = "ab".pad_end(4, '.')`, nil, "ab..")
	expectRun(t, `out = "їЇ".pad_start(4, 'x')`, nil, "xxїЇ") // width counts SYMBOLS, never octets
	expectRun(t, `out = "abc".pad_start(2)`, nil, "abc")      // width below length is a no-op
	expectRun(t, `out = bytes("7").pad_start(3, b'0')`, nil, []byte("007"))
	expectRun(t, `out = u"ab".pad_end(3)`, nil, []rune("ab "))
	expectRun(t, `out = [1].pad_end(3)`, nil, ARR{1, core.Undefined, core.Undefined}) // array's canonical blank
	expectRun(t, `out = [1].pad_start(3, 0)`, nil, ARR{0, 0, 1})
	expectRun(t, `a := bytes("7"); a.pad_start_in_place(2, b'0'); out = a`, nil, []byte("07"))
	expectError(t, `"ab".pad_end(4, "..")`, nil, "invalid_argument_type") // a run fill hides a truncation rule
	expectError(t, `bytes("a").pad_end(3, 'é')`, nil, "invalid_value")    // two octets do not fit one element
	expectError(t, `"ab".pad_start()`, nil, "wrong_num_arguments")
	expectError(t, `freeze_shallow([1]).pad_end_in_place(3, 0)`, nil, "not_mutable")

	// split/partition stay the triple's: array has other spellings (chunk, filter, the locators)
	expectError(t, `[1, 2].split(0)`, nil, "type array has no method split")
	expectError(t, `[1, 2].partition(0)`, nil, "type array has no method partition")
	expectError(t, `range(0, 5).trim(0)`, nil, "type range has no method trim") // a formula has no incidental ends
}

// TestMemberFunctionMapFlatMap checks map's 1:1 contract (the result is the RECEIVER'S type; on a text receiver
// a sequence or undefined callback result raises) and flat_map, the map-then-concatenate member: each callback
// result is read like an add-side operand — a run concatenates, undefined contributes nothing, anything else is
// one element. flat_map(f) ≠ map(f).flatten(): flatten is single-level but flattens ANY nested element.
func TestMemberFunctionMapFlatMap(t *testing.T) {
	// map on array: unchanged — a nested result nests, undefined stays (1:1)
	expectRun(t, `out = [1, 2].map(x => [x, x])`, nil, ARR{ARR{1, 1}, ARR{2, 2}})
	expectRun(t, `out = [1, 2, 3].map(func(x) { if x > 1 { return x } })`, nil, ARR{core.Undefined, 2, 3})
	// map on the triple: the receiver's type
	expectRun(t, `out = "abc".map(func(c) { return c == 'b' ? 'x' : c })`, nil, "axc")
	expectRun(t, `out = u"abc".map(func(c) { return (c.int() + 1).rune() })`, nil, []rune("bcd"))
	expectError(t, `"abc".map(func(c) { return "xx" })`, nil, "invalid_value")              // 1:1 — a run is flat_map's
	expectError(t, `"abc".map(func(c) { if c != 'b' { return c } })`, nil, "invalid_value") // dropping is flat_map's

	// flat_map: run splices, undefined drops, element stays
	expectRun(t, `out = "abc".flat_map(func(c) { return c == 'b' ? "xxx" : c })`, nil, "axxxc")
	expectRun(t, `out = "abc".flat_map(func(c) { if c != 'b' { return c } })`, nil, "ac")
	expectRun(t, `out = bytes("ab").flat_map(func(b) { return bytes([b, b]) })`, nil, []byte("aabb"))
	expectRun(t, `out = [1, 2].flat_map(x => [x, x])`, nil, ARR{1, 1, 2, 2})              // own kind spreads
	expectRun(t, `out = [1, 2].flat_map(x => x * 10)`, nil, ARR{10, 20})                  // anything else is one element
	expectRun(t, `out = [1, 2].flat_map(func(x) { if x > 1 { return x } })`, nil, ARR{2}) // undefined contributes nothing
	expectRun(t, `out = [1, 2].flat_map(x => range(0, x))`, nil,                          // a range is one element
		ARR{core.NewIntRangeValue(0, 1, 1), core.NewIntRangeValue(0, 2, 1)})
	expectRun(t, `out = [[1, [2]]].flat_map(x => x)`, nil, ARR{1, ARR{2}})         // one level, unlike flatten(-1)
	expectError(t, `range(0, 3).map(x => x)`, nil, "type range has no method map") // spell it .array().map(...)
	expectError(t, `range(0, 3).flat_map(x => x)`, nil, "type range has no method flat_map")
}

// TestMemberFunctionCasingFamily checks the casing family — ONE word segmenter, five renderings — plus
// case_fold and fields, all symbol-class members: string and runes carry them, bytes never does.
func TestMemberFunctionCasingFamily(t *testing.T) {
	// the segmenter: closed boundary set (whitespace/_/-), lower→upper, upper-run+lower
	expectRun(t, `out = "hello_world".snake_case()`, nil, "hello_world")
	expectRun(t, `out = "helloWorld".snake_case()`, nil, "hello_world")
	expectRun(t, `out = "hello-world now".kebab_case()`, nil, "hello-world-now")
	expectRun(t, `out = "HTTPServer".snake_case()`, nil, "http_server")
	expectRun(t, `out = "parseXMLFile".snake_case()`, nil, "parse_xml_file")
	expectRun(t, `out = "hello_world".camel_case()`, nil, "helloWorld")
	expectRun(t, `out = "hello_world".pascal_case()`, nil, "HelloWorld")
	expectRun(t, `out = "__a__b".snake_case()`, nil, "a_b") // empty words dropped
	// rule 4: the boundary set is CLOSED — digits, apostrophes, periods stay inside the word
	expectRun(t, `out = "don't stop".title_case()`, nil, "Don't Stop")
	expectRun(t, `out = "v1.2 item2".title_case()`, nil, "V1.2 Item2")
	// the two policies: identifiers normalise the interior, the label preserves it
	expectRun(t, `out = "ATM fee".title_case()`, nil, "ATM Fee")
	expectRun(t, `out = "ATM fee".snake_case()`, nil, "atm_fee")
	// the label rendering keeps the WRITTEN boundaries: case transitions stay inside words —
	// a label preserves the author's emphasis, word boundaries included
	expectRun(t, `out = "hELLO world".title_case()`, nil, "HELLO World")
	expectRun(t, `out = "iPhone".title_case()`, nil, "IPhone")
	// title_case re-segments: an identifier turns into a label
	expectRun(t, `out = "atm_fee-total".title_case()`, nil, "Atm Fee Total")
	expectRun(t, `out = u"helloWorld".kebab_case()`, nil, []rune("hello-world"))
	expectError(t, `bytes("ab").snake_case()`, nil, "invalid_method") // symbol class — never on bytes
	expectError(t, `"ab".title_case("x")`, nil, "wrong_num_arguments")

	// case_fold: a transform (composes as a key/sort basis), the fold ≠ lower in both directions
	expectRun(t, `out = "Straße".case_fold() == "STRASSE".lower().case_fold()`, nil, false) // ß folds to ß, not ss (simple fold)
	expectRun(t, `out = "ſtop".case_fold() == "Stop".case_fold()`, nil, true)               // ſ and S fold together
	expectRun(t, `out = "ſtop".lower() == "Stop".lower()`, nil, false)                      // lower cannot see it
	expectRun(t, `out = "HeLLo".case_fold() == "hello".case_fold()`, nil, true)
	// the canonical representative is the smallest LOWERCASE member of the fold orbit (the
	// minimum when none exists), so the render reads like every mainstream casefold while
	// equality stays exactly EqualFold — İ keeps its own orbit, distinct from i
	expectRun(t, `out = u"Hi".case_fold()`, nil, []rune("hi"))
	expectRun(t, `out = "İ".case_fold() == "i".case_fold()`, nil, false)
	expectError(t, `bytes("ab").case_fold()`, nil, "invalid_method")

	// fields() is GONE: the blank set is one notion — NUL ∪ whitespace — projected into each
	// element domain, so string/runes split() already splits on Unicode whitespace and fields()
	// would be a near-duplicate; bytes keeps the ASCII projection (all the whitespace an octet
	// can express)
	expectError(t, `"a b".fields()`, nil, "invalid_method")
	expectRun(t, `out = "  a  b ".split()`, nil, ARR{"a", "b"})
	expectRun(t, "out = \"a\u00A0b\".split()", nil, ARR{"a", "b"})         // NBSP splits on string
	expectRun(t, "out = \"\u00A0x\u00A0\".trim()", nil, "x")               // ... and trims
	expectRun(t, `out = bytes("a b").split().len()`, nil, int64(2))        // octets: ASCII whitespace
	expectRun(t, "out = bytes(\"a\u00A0b\").split().len()", nil, int64(1)) // NBSP octets are content
}

// TestMemberFunctionRosterCompletions checks the per-type roster completions: string's gains (its roster is
// runes' minus the twins, the views, sum/avg, the _shallow pair and the byte/rune conversions), range's
// closed-form roster (computed on start/stop/step, nothing materialised), dict's transform block, and the
// remaining derived _in_place twins (filter/dedup/unique).
func TestMemberFunctionRosterCompletions(t *testing.T) {
	// string's gains — unsuffixed only
	expectRun(t, `out = "bca".sort()`, nil, "abc")
	expectRun(t, `out = "aabca".dedup()`, nil, "abca")
	expectRun(t, `out = "aabca".unique()`, nil, "abc")
	expectRun(t, `out = "abc".first()`, nil, 'a')
	expectRun(t, `out = "".first("-")`, nil, "-")
	expectRun(t, `out = "abc".last()`, nil, 'c')
	expectRun(t, `out = "bca".min()`, nil, 'a')
	expectRun(t, `out = "bca".max()`, nil, 'c')
	expectRun(t, `out = "abcd".slice(1, 3)`, nil, "bc")
	expectRun(t, `out = "abcd".slice(1, 99)`, nil, "bcd") // slice clamps
	expectRun(t, `out = "abcde".chunk(2)`, nil, ARR{"ab", "cd", "e"})
	expectRun(t, `out = "abcd".splice(1, 2, "xy")`, nil, "axyd")
	expectRun(t, `out = "abc".reduce("", func(acc, c) { return c.string() + acc })`, nil, "cba")
	expectError(t, `"abc".sum()`, nil, "invalid_method") // no Numeric elements, no aggregation
	expectError(t, `"abc".sort_in_place()`, nil, "type string has no method sort_in_place")
	expectError(t, `"abc".slice_view(0, 1)`, nil, "invalid_method") // sharing is unobservable: slice IS slice_view

	// range's closed forms — every transform answers a range, chunk an array of ranges
	expectRun(t, `out = range(0, 10, 3).reverse() == range(9, -1, 3)`, nil, true)
	expectRun(t, `out = range(10, 0, 3).sort() == range(1, 11, 3)`, nil, true)
	expectRun(t, `out = range(0, 5).sort() == range(0, 5)`, nil, true)
	expectRun(t, `out = range(0, 5).dedup() == range(0, 5)`, nil, true) // the identity: a progression never repeats
	expectRun(t, `out = range(0, 5).unique() == range(0, 5)`, nil, true)
	expectRun(t, `out = range(0, 10).slice(2, 5) == range(2, 5)`, nil, true)
	expectRun(t, `out = range(0, 10, 3).slice(1, 3) == range(3, 9, 3)`, nil, true)
	expectRun(t, `out = range(0, 10).slice(8, 99) == range(8, 10)`, nil, true)     // clamps
	expectRun(t, `out = range(10, 0, 3).slice(1, 3) == range(7, 1, 3)`, nil, true) // direction kept
	expectRun(t, `c := range(0, 5).chunk(2); out = [c.len(), c[0] == range(0, 2), c[2] == range(4, 5)]`, nil,
		ARR{3, true, true})
	expectRun(t, `out = range(0, 10, 3).first()`, nil, 0)
	expectRun(t, `out = range(0, 10, 3).last()`, nil, 9)
	expectRun(t, `out = range(10, 0, 3).min()`, nil, 1)
	expectRun(t, `out = range(10, 0, 3).max()`, nil, 10)
	expectRun(t, `out = range(0, 0).min("-")`, nil, "-")
	expectRun(t, `out = range(1, 5).sum()`, nil, 10)
	expectRun(t, `out = range(1, 5).avg()`, nil, 2) // the same int division array's avg performs
	expectRun(t, `out = range(1, 4).reduce(0, func(acc, x) { return acc + x })`, nil, 6)
	expectError(t, `range(0, 5).filter(x => true)`, nil, "invalid_method") // result type would depend on the data
	expectError(t, `range(0, 5).slice_view(0, 1)`, nil, "invalid_method")

	// dict's transform block: map transforms the ATTACHMENT, keys fixed (f/1 = key, f/2 = key and value);
	// reduce folds over sorted keys
	expectRun(t, `out = dict({a: 1, b: 2}).map(func(k, v) { return v * 10 })`, nil, MAP{"a": 10, "b": 20})
	expectRun(t, `out = dict({a: 1}).map(func(k) { return k.upper() })`, nil, MAP{"a": "A"})
	expectRun(t, `out = dict({b: 2, a: 1}).reduce("", func(acc, k) { return acc + k })`, nil, "ab") // sorted keys
	expectRun(t, `out = dict({a: 1, b: 2}).reduce(0, func(acc, k, v) { return acc + v })`, nil, 3)
	expectError(t, `dict({a: 1}).flat_map(func(k) { return k })`, nil, "invalid_method") // a map has no run to concatenate

	// the remaining derived twins: filter/dedup/unique in place, returning the receiver
	expectRun(t, `a := [1, 2, 3]; b := a; a.filter_in_place(x => x > 1); out = b`, nil, ARR{2, 3})
	expectRun(t, `a := [1, 1, 2]; out = a.dedup_in_place()`, nil, ARR{1, 2})
	expectRun(t, `a := [2, 1, 2]; a.unique_in_place(); out = a`, nil, ARR{2, 1})
	expectRun(t, `a := bytes("aab"); a.dedup_in_place(); out = a`, nil, []byte("ab"))
	expectRun(t, `a := runes("aba"); a.unique_in_place(); out = a`, nil, []rune("ab"))
	expectRun(t, `a := bytes("a b"); a.filter_in_place(b' '); out = a`, nil, []byte(" "))
	expectRun(t, `d := dict({a: 1, b: 2}); d.filter_in_place("a"); out = d`, nil, MAP{"a": 1})
	expectError(t, `freeze_shallow([1]).filter_in_place(x => true)`, nil, "not_mutable")
	expectError(t, `freeze_shallow([1]).dedup_in_place()`, nil, "not_mutable")
	expectError(t, `freeze(bytes("a")).unique_in_place()`, nil, "not_mutable")
	expectError(t, `freeze_shallow(dict({a: 1})).filter_in_place("a")`, nil, "not_mutable")
}

// TestMemberFunctionSpliceBytesRunes checks P5-002's generalization of splice()/splice_in_place() from array-only
// to bytes/runes, via the new shared core.SeqSplice — same argument shape and pure/mutating split as array's
// (see TestMemberFunctionAppendDeleteSplice's array splice() cases and TestMemberFunctionSpliceInPlace). Insert
// items are converted the same way append()'s are (bytesAppendItems/runesAppendItems), so passing a bytes/runes
// value as one of splice's insert items spreads it, rather than erroring or nesting it as one opaque element.
func TestMemberFunctionSpliceBytesRunes(t *testing.T) {
	// bytes: splice() is pure
	expectRun(t, `v := bytes("abc"); result := v.splice(0, 1); out = [result, v]`, nil,
		ARR{[]byte("bc"), []byte("abc")})
	expectRun(t, `v := bytes("abc"); out = v.splice(1, 0, 'x')`, nil, []byte("axbc"))
	expectRun(t, `v := bytes("abc"); out = v.splice(1, 0, bytes("xy"))`, nil, []byte("axybc"))
	expectRun(t, `out = freeze(bytes("abc")).splice(0, 1)`, nil, []byte("bc")) // pure: works on immutable too
	expectError(t, `bytes("abc").splice(0, -1)`, nil, "invalid_value: splice delete count must be non-negative")

	// bytes: splice_in_place() mutates and returns the receiver
	expectRun(t, `v := bytes("abc"); res := v.splice_in_place(0, 1); out = [res, v]`, nil,
		ARR{[]byte("bc"), []byte("bc")})
	expectError(t, `freeze(bytes("abc")).splice_in_place(0)`, nil,
		"not_mutable: (splice_in_place) type immutable-bytes is immutable")

	// runes: splice() is pure
	expectRun(t, `v := runes("abc"); result := v.splice(0, 1); out = [result, v]`, nil,
		ARR{[]rune("bc"), []rune("abc")})
	expectRun(t, `v := runes("abc"); out = v.splice(1, 0, 'x')`, nil, []rune("axbc"))
	expectRun(t, `v := runes("abc"); out = v.splice(1, 0, runes("xy"))`, nil, []rune("axybc"))
	expectRun(t, `out = freeze(runes("abc")).splice(0, 1)`, nil, []rune("bc")) // pure: works on immutable too

	// runes: splice_in_place() mutates and returns the receiver
	expectRun(t, `v := runes("abc"); res := v.splice_in_place(0, 1); out = [res, v]`, nil,
		ARR{[]rune("bc"), []rune("bc")})
	expectError(t, `freeze(runes("abc")).splice_in_place(0)`, nil,
		"not_mutable: (splice_in_place) type immutable-runes is immutable")
}

// TestMemberFunctionSortInPlaceReverseInPlace checks the sort_in_place()/reverse_in_place() mutating twins,
// added 2026-08-17 alongside the freeze_in_place->freeze_shallow rename (found missing during that audit: sort
// and reverse were the only two Seq-shaped operations with no `_in_place` twin despite `docs/conventions.md`
// citing them by name as the canonical example of the convention — a stale example describing behavior that
// didn't actually exist until now). Both mutate the receiver's own backing storage directly, visible through
// every existing alias without reassignment (same shape as append_in_place/splice_in_place), and reject an
// immutable receiver.
func TestMemberFunctionSortInPlaceReverseInPlace(t *testing.T) {
	// array
	expectRun(t, `a := [3, 1, 2]; a.sort_in_place(); out = a`, nil, ARR{1, 2, 3})
	expectRun(t, `a := [3, 1, 2]; b := a; a.sort_in_place(); out = b`, nil, ARR{1, 2, 3}) // visible via alias, no reassignment
	expectError(t, `freeze_shallow([3, 1, 2]).sort_in_place()`, nil, "not_mutable")
	expectError(t, `[1].sort_in_place(1)`, nil, "wrong_num_arguments")

	expectRun(t, `a := [1, 2, 3]; a.reverse_in_place(); out = a`, nil, ARR{3, 2, 1})
	expectRun(t, `a := [1, 2, 3]; b := a; a.reverse_in_place(); out = b`, nil, ARR{3, 2, 1})
	expectError(t, `freeze_shallow([1, 2, 3]).reverse_in_place()`, nil, "not_mutable")
	expectError(t, `[1].reverse_in_place(1)`, nil, "wrong_num_arguments")

	// bytes
	expectRun(t, `b := bytes("cba"); b.sort_in_place(); out = b`, nil, []byte("abc"))
	expectError(t, `freeze(bytes("cba")).sort_in_place()`, nil, "not_mutable")
	expectRun(t, `b := bytes("abc"); b.reverse_in_place(); out = b`, nil, []byte("cba"))
	expectError(t, `freeze(bytes("abc")).reverse_in_place()`, nil, "not_mutable")

	// runes
	expectRun(t, `r := runes("cba"); r.sort_in_place(); out = r`, nil, []rune("abc"))
	expectError(t, `freeze(runes("cba")).sort_in_place()`, nil, "not_mutable")
	expectRun(t, `r := runes("abc"); r.reverse_in_place(); out = r`, nil, []rune("cba"))
	expectError(t, `freeze(runes("abc")).reverse_in_place()`, nil, "not_mutable")

	// pure sort()/reverse() are unaffected: still return a fresh copy, source untouched
	expectRun(t, `a := [3, 1, 2]; b := a.sort(); out = a`, nil, ARR{3, 1, 2})
	expectRun(t, `a := [1, 2, 3]; b := a.reverse(); out = a`, nil, ARR{1, 2, 3})
}

func TestImmutable(t *testing.T) {
	// scalars are already immutable values: the predicate says so, and the shallow-freeze twin
	// refuses them rather than pretending to act (see TestBuiltinFunctionFreezeShallow)
	expectRun(t, `out = is_immutable(1)`, nil, true)
	expectRun(t, `a := 5; out = is_immutable(a)`, nil, true)
	expectError(t, `freeze_shallow(1)`, nil, "invalid_argument_type: (freeze_shallow) argument first expects type array, dict, or record")

	// `immutable(x)` is retired: it did exactly what freeze_shallow does, so the name is gone
	expectError(t, `immutable([1, 2])`, nil, "unresolved reference 'immutable'")
	expectError(t, `out = immutable([1, 2])`, nil, "unresolved reference 'immutable'")

	// array
	expectError(t, `a := freeze_shallow([1, 2, 3]); a[1] = 5`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := freeze_shallow(["foo", [1,2,3]]); a[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `a := freeze_shallow(["foo", [1,2,3]]); a[1][1] = "bar"; out = a`, nil, ARR{"foo", ARR{1, "bar", 3}})
	expectError(t, `a := freeze_shallow(["foo", freeze_shallow([1,2,3])]); a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := ["foo", freeze_shallow([1,2,3])]; a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `a := freeze_shallow([1,2,3]); b := copy(a); b[1] = 5; out = b`, nil, ARR{1, 5, 3})
	expectRun(t, `a := freeze_shallow([1,2,3]); b := copy(a); b[1] = 5; out = a`, nil, ARR{1, 2, 3})
	expectRun(t, `out = freeze_shallow([1,2,3]) == [1,2,3]`, nil, true)
	expectRun(t, `out = freeze_shallow([1,2,3]) == freeze_shallow([1,2,3])`, nil, true)
	expectRun(t, `out = [1,2,3] == freeze_shallow([1,2,3])`, nil, true)
	expectRun(t, `out = freeze_shallow([1,2,3]) == [1,2]`, nil, false)
	expectRun(t, `out = freeze_shallow([1,2,3]) == freeze_shallow([1,2])`, nil, false)
	expectRun(t, `out = [1,2,3] == freeze_shallow([1,2])`, nil, false)
	expectRun(t, `out = freeze_shallow([1, 2, 3, 4])[1]`, nil, 2)
	expectRun(t, `out = freeze_shallow([1, 2, 3, 4])[1:3]`, nil, ARR{2, 3})
	expectRun(t, `a := freeze_shallow([1,2,3]); a = 5; out = a`, nil, 5)

	// map
	expectError(t, `a := freeze_shallow({b: 1, c: 2}); a.b = 5`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectError(t, `a := freeze_shallow({b: 1, c: 2}); a["b"] = "bar"`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectRun(t, `a := freeze_shallow({b: 1, c: [1,2,3]}); a.c[1] = "bar"; out = a`, nil, MAP{"b": 1, "c": ARR{1, "bar", 3}})
	expectError(t, `a := freeze_shallow({b: 1, c: freeze_shallow([1,2,3])}); a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, `a := {b: 1, c: freeze_shallow([1,2,3])}; a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `out = freeze_shallow({a:1,b:2}) == {a:1,b:2}`, nil, true)
	expectRun(t, `out = freeze_shallow({a:1,b:2}) == freeze_shallow({a:1,b:2})`, nil, true)
	expectRun(t, `out = {a:1,b:2} == freeze_shallow({a:1,b:2})`, nil, true)
	expectRun(t, `out = freeze_shallow({a:1,b:2}) == {a:1,b:3}`, nil, false)
	expectRun(t, `out = freeze_shallow({a:1,b:2}) == freeze_shallow({a:1,b:3})`, nil, false)
	expectRun(t, `out = {a:1,b:2} == freeze_shallow({a:1,b:3})`, nil, false)
	expectRun(t, `out = freeze_shallow({a:1,b:2}).b`, nil, 2)
	expectRun(t, `out = freeze_shallow({a:1,b:2})["b"]`, nil, 2)
	expectRun(t, `a := freeze_shallow({a:1,b:2}); a = 5; out = 5`, nil, 5)
	expectRun(t, `a := freeze_shallow({a:1,b:2}); out = a.c`, nil, core.Undefined)

	expectRun(t, `a := freeze_shallow({b: 5, c: "foo"}); out = a.b`, nil, 5)
	expectError(t, `a := freeze_shallow({b: 5, c: "foo"}); a.b = 10`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
}

func TestBytesN(t *testing.T) {
	// the sizing form is gone; preallocation spells the fill explicitly
	expectRun(t, `out = bytes(b'\x00', 0)`, nil, make([]byte, 0))
	expectRun(t, `out = bytes(b'\x00', 10)`, nil, make([]byte, 10))
	expectRun(t, `out = bytes(b'\x00', 1000)`, nil, make([]byte, 1000))
	expectError(t, `out = bytes(b'\x00', -1)`, nil, "repeat count must be non-negative")
	expectError(t, `out = bytes(-1)`, nil, "conversion: cannot convert int to bytes")
}

func TestRunesN(t *testing.T) {
	// runes(n) is the CONVERSION — runes("10") — never sizing: the conversion
	// claims the spelling. Pre-filled runes are u"\0".repeat(n)
	expectRun(t, `out = runes(0)`, nil, []rune("0"))
	expectRun(t, `out = runes(10)`, nil, []rune("10"))
	expectRun(t, `out = runes(1000).len()`, nil, 4)
	expectRun(t, `out = u"\u0000".repeat(10)`, nil, make([]rune, 10))
	expectRun(t, `out = runes(-1)`, nil, []rune("-1")) // the conversion — runes(n) never sizes; the conversion claims the spelling
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

	// Exact-chain cross-type equality: bool < byte < rune < int < decimal, each recognizes every
	// type below it directly (flattened, not chained), commutative both directions.
	testEquality(t, `true`, `byte(1)`, true)
	testEquality(t, `false`, `byte(0)`, true)
	testEquality(t, `byte(1)`, `rune(1)`, true)
	testEquality(t, `rune(65)`, `65`, true)
	testEquality(t, `true`, `1`, true)
	testEquality(t, `true`, `2`, false)
	testEquality(t, `5`, `decimal("5")`, true)
	testEquality(t, `true`, `decimal("1")`, true)
	testEquality(t, `byte(5)`, `decimal("5")`, true)
	testEquality(t, `rune(65)`, `decimal("65")`, true)

	// float: always exact against bool/byte/rune (whole range fits the float64 mantissa); exact via
	// math/big.Rat (not a lossy float64 round-trip) against int/decimal -- this is the whole reason
	// the round-trip-check design was dropped in favor of big.Rat (see docs/types.md).
	testEquality(t, `true`, `1.0`, true)
	testEquality(t, `byte(1)`, `1.0`, true)
	testEquality(t, `rune(65)`, `65.0`, true)
	testEquality(t, `9007199254740992`, `float(9007199254740992)`, true)
	testEquality(t, `9007199254740993`, `float(9007199254740992)`, false) // no silent 2^53 collapse
	testEquality(t, `decimal("0.5")`, `0.5`, true)                        // 0.5 has an exact binary form
	testEquality(t, `decimal("0.1")`, `0.1`, false)                       // float 0.1 isn't exactly a tenth

	// NaN/Inf: decimal's NaN and a NaN float are the same "unique minimum" concept from both
	// directions; float's own same-type NaN is a total order now too (NaN == NaN is true).
	testEquality(t, `0d / 0d`, `float("nan")`, true) // float arithmetic cannot produce NaN now; the parse still can
	testEquality(t, `0d / 0d`, `5.0`, false)
	testEquality(t, `float("nan")`, `float("nan")`, true)

	// Text tier: string/runes/bytes all recognize the exact chain + float via canonical text form.
	testEquality(t, `5`, `"5"`, true)
	testEquality(t, `true`, `"true"`, true)
	testEquality(t, `true`, `"false"`, false) // the actual old bug: AsBool()-truthiness used to leak this true
	testEquality(t, `decimal("2.5")`, `"2.5"`, true)
	testEquality(t, `bytes("hello")`, `"hello"`, true)
	testEquality(t, `bytes("hello")`, `runes("hello")`, true)
	testEquality(t, `bytes("5")`, `5`, true)
	testEquality(t, `bytes("hello")`, `[104, 101, 108, 108, 111]`, false) // bytes no longer leaks into array equality

	expectRun(t, "out = true == true", nil, true)
	expectRun(t, "out = true != false", nil, true)
	expectRun(t, "out = false != true", nil, true)

	expectRun(t, "out = true == 1", nil, true)
	expectRun(t, "out = 1 == true", nil, true)

	expectRun(t, "out = true == 2", nil, false)
	expectRun(t, "out = 2 != true", nil, true)
	expectRun(t, "out = true != 2", nil, true)
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

	// record: the single-variable form yields KEYS (a map's element is its key)
	expectRun(t, `out = 0; for k in {a:2,b:3,c:4} { out += k[0] - 'a' }`, nil, 3)                             // key (order-free)
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
			return b
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
			return b
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

	// 'a += b' desugars to 'a = a + b' — no implicit stringification means neither direction works.
	expectError(t, `a := "foo"; a++`, nil, "invalid_binary_operator: string + int")
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

	// variables declared in if-init cannot be redeclared with := inside body/else
	expectError(t, `
if a := 5; a {
	a := 6
	out = a
}`, nil, "'a' redeclared in this block")
	expectError(t, `
a := 4
if a := 5; a {
	a := 6
	out = a
}`, nil, "'a' redeclared in this block")
	expectError(t, `
a := 4
if a := 0; a {
	a := 6
	out = a
} else {
	a := 7
	out = a
	}`, nil, "'a' redeclared in this block")
	expectRun(t, `
a := 4
if a := 0; a {
	out = a
} else {
	out = a
}`, nil, 0)

	// same rule applies when init uses var syntax
	expectError(t, `
a := 4
if var a = 5; a {
	a := 6
	out = a
	}`, nil, "'a' redeclared in this block")
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

	// A header (function/lambda parameters and named result; if-init; for-init; for-in key/value)
	// shares one scope with its own corresponding block(s): reusing the header's name with := or
	// var directly in that block is a redeclaration error; = always reassigns it instead.
	expectError(t, `
foo := func(x) { x := 10; return x }
out = foo(1)
`, nil, "'x' redeclared in this block")
	expectRun(t, `
foo := func(x) { x = 10; return x }
out = foo(1)
`, nil, 10)
	expectError(t, `
foo := func(x) result { result := 5 }
out = foo(1)
`, nil, "'result' redeclared in this block")
	expectRun(t, `
foo := func(x) result { result = 5 }
out = foo(1)
`, nil, 5)
	expectError(t, `
for k, v in [1, 2, 3] {
	k := 99
	out = k
}
`, nil, "'k' redeclared in this block")
	expectRun(t, `
result := 0
for k, v in [1, 2, 3] {
	k = 99
	result = k
}
out = result
`, nil, 99)

	// A block nested one level deeper than a header's own corresponding block is always a fresh
	// scope: it can freely reuse the header's name, whether via an ordinary := or via its own
	// nested header (if/for-init, else-if, or a nested func's own parameter list).
	expectRun(t, `
result := 0
if x := 10; true {
	if x := 20; true {
		result = x
	}
}
out = result
`, nil, 20)
	expectRun(t, `
foo := func(x) {
	if true {
		x := 99
		return x
	}
	return x
}
out = foo(1)
`, nil, 99)
	expectRun(t, `
result := 0
if x := 10; false {
	result = x
} else if x := 20; true {
	result = x
}
out = result
`, nil, 20)

	// An if-init variable is always free to shadow an outer variable of the same name (its own
	// header is just a fresh nested scope like any other); reassigning it in the body only
	// affects the shadowed copy, never the outer one.
	expectRun(t, `
x := 1
if x := 2; true {
	x = 10
}
out = x
`, nil, 1)
	// A bare if (no init clause of its own) is not a header at all: its body reusing an outer
	// name with := is ordinary shadowing, same as any other nested block.
	expectRun(t, `
x := 1
if true {
	x := 20
}
out = x
`, nil, 1)
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
		return a.append(3)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, `
	f := func(a, ...b) {
		return [a].append(b.append(3)...)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, `
	f := func(a, ...b) {
		return [a].append(b).append(3)
	}
	out = f(1, [2]...)
	`, nil, ARR{1, 2, 3})

	// the element spelling: push never spreads, so the collected variadic pack stays one row
	expectRun(t, `
	f := func(a, ...b) {
		return [a].push(b).append(3)
	}
	out = f(1, [2]...)
	`, nil, ARR{1, ARR{2}, 3})

	expectRun(t, `
	f1 := func(...a){
		return [3].append(a...)
	}
	f2 := func(a, ...b) {
		return f1([a].append(b...)...)
	}
	out = f2([1, 2]...)
	`, nil, ARR{3, 1, 2})

	expectRun(t, `
	f := func(a, ...b) {
		return func(...a) {
			return [3].append(a.append(4)...)
		}(a, b...)
	}
	out = f([1, 2]...)
	`, nil, ARR{3, 1, 2, 4})

	expectRun(t, `
	f := func(a, ...b) {
		c := b.append(4)
		return func(){
			return [a].append(b...).append(c...)
		}()
	}
	out = f(1, freeze_shallow([2, 3])...)
	`, nil, ARR{1, 2, 3, 2, 3, 4})

	expectError(t, `func(a) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 1 argument(s), got 2")
	expectError(t, `func(a, b, c) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 3 argument(s), got 2")
}

func TestSliceIndex(t *testing.T) {
	// slicing is chaining-shaped, so it propagates like undefined[0]
	expectRun(t, `out = undefined[:1]`, nil, core.Undefined)
	expectRun(t, `out = undefined[1:2]`, nil, core.Undefined)
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

func TestPlaceholder(t *testing.T) {
	t.Run("call_argument", func(t *testing.T) {
		expectRun(t, `
		add := func(a, b, c) { return a + b + c }
		w := add(1, _, 3)
		out = w(10)`, nil, 14)
	})

	t.Run("callee_position", func(t *testing.T) {
		expectRun(t, `
		mul := func(a, b) { return a * b }
		w := _(2, 5)
		out = w(mul)`, nil, 10)
	})

	t.Run("callee_and_argument_both_placeholders", func(t *testing.T) {
		// general 2-arg "apply": (f, x) => f(x)
		expectRun(t, `
		mul10 := func(a) { return a * 10 }
		w := _(_)
		out = w(mul10, 5)`, nil, 50)
	})

	t.Run("method_receiver_and_argument", func(t *testing.T) {
		expectRun(t, `
		w := _.filter(_ > 1)
		out = w([1, 2, 3])`, nil, ARR{2, 3})
	})

	t.Run("selector", func(t *testing.T) {
		expectRun(t, `
		getname := _.name
		out = getname({name: "kavun"})`, nil, "kavun")
	})

	t.Run("binary_both_sides", func(t *testing.T) {
		expectRun(t, `
		w := _ + _
		out = w(2, 5)`, nil, 7)
	})

	t.Run("unary", func(t *testing.T) {
		expectRun(t, `
		w := !_
		out = w(false)`, nil, true)
	})

	t.Run("index", func(t *testing.T) {
		expectRun(t, `
		a := [10, 20, 30]
		w := a[_]
		out = w(1)`, nil, 20)
	})

	t.Run("index_receiver_only", func(t *testing.T) {
		expectRun(t, `
		a := [10, 20, 30]
		w := _[1]
		out = w(a)`, nil, 20)
	})

	t.Run("index_both_operands_are_placeholders", func(t *testing.T) {
		expectRun(t, `
		a := [10, 20, 30]
		w := _[_]
		out = w(a, 1)`, nil, 20)
	})

	t.Run("slice", func(t *testing.T) {
		expectRun(t, `
		a := [10, 20, 30]
		w := a[_:2]
		out = w(1)`, nil, ARR{20})
	})

	t.Run("ternary", func(t *testing.T) {
		expectRun(t, `
		w := _ ? "yes" : "no"
		out = [w(true), w(false)]`, nil, ARR{"yes", "no"})
	})

	t.Run("multiple_placeholders_left_to_right", func(t *testing.T) {
		expectRun(t, `
		f := func(a, b, c) { return [a, b, c] }
		w := f(_, "mid", _)
		out = w(1, 2)`, nil, ARR{1, "mid", 2})
	})

	t.Run("nested_call_binds_to_innermost", func(t *testing.T) {
		// '_' binds to bar(_, 2), NOT to foo -- foo receives an already-built lambda as its 2nd argument.
		expectRun(t, `
		bar := func(x, y) { return x + y }
		foo := func(a, b) { return [a, b] }
		r := foo(1, bar(_, 2))
		out = [r[0], is_callable(r[1]), r[1](5)]`, nil, ARR{1, true, 7})
	})

	t.Run("iife_degenerate_but_well_defined", func(t *testing.T) {
		w := `
		iife := func(x) { return x }(_)
		out = iife(42)`
		expectRun(t, w, nil, 42)
	})

	t.Run("bare_underscore_outside_placeholder_position_still_errors", func(t *testing.T) {
		expectError(t, `out = _`, nil, "unresolved reference '_'")
	})

	t.Run("discard_in_destructuring_unaffected", func(t *testing.T) {
		expectRun(t, `
		a, _, c := [1, 2, 3]
		out = [a, c]`, nil, ARR{1, 3})
	})

	// --- Exception: method/field NAMES are never placeholder targets (see docs/language.md). ---

	t.Run("underscore_is_a_legal_field_name_untouched_by_placeholder", func(t *testing.T) {
		// 'x' isn't a placeholder, so nothing is rewritten here -- '_' is just a literal field name.
		expectRun(t, `
		r := {_: 99}
		out = r._`, nil, 99)
	})

	t.Run("selector_receiver_placeholder_field_name_literal", func(t *testing.T) {
		// receiver becomes the param; the field name '_' after the dot is untouched.
		expectRun(t, `
		r := {_: 99}
		w := _._
		out = w(r)`, nil, 99)
	})

	t.Run("method_name_is_never_a_placeholder_target", func(t *testing.T) {
		// desugars fine (3 placeholders: receiver + 2 args) -- the method NAME stays literal '_'.
		// Records have no real method dispatch table (fields only), so 'a._(b, c)' means "look up field '_',
		// then call it" -- since the field holds an int, this fails at CALL time, not desugar time.
		expectRun(t, `
		r := {_: 99}
		w := _._(_, _)
		out = is_callable(w)`, nil, true)
		expectError(t, `
		r := {_: 99}
		w := _._(_, _)
		out = w(r, 1, 2)`, nil, "not callable")
	})

	// --- Corner cases for what IS and ISN'T a placeholder position. ---

	t.Run("whole_spread_source_can_be_a_placeholder", func(t *testing.T) {
		// '_...' binds the param to the WHOLE iterable being spread, not to one of its elements.
		expectRun(t, `
		f := func(a, ...rest) { return [a, rest] }
		w := f(_, _...)
		out = w(1, [2, 3])`, nil, ARR{1, ARR{2, 3}})
	})

	t.Run("placeholder_inside_composite_literal_not_covered", func(t *testing.T) {
		// Array/Record literals aren't in the rewrite's node list at all -- a bare '_' inside one is just an
		// ordinary unresolved reference, same as '_' anywhere outside a qualifying node.
		expectError(t, `out = [1, _, 3]`, nil, "unresolved reference '_'")
	})

	t.Run("parenthesized_placeholder_not_detected", func(t *testing.T) {
		// The placeholder must be bare -- wrapping it in parens hides it from the rewrite.
		expectError(t, `out = (_) + 1`, nil, "unresolved reference '_'")
	})
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
			remove(y, x)
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
	// division by zero raises on every numeric type — float no longer answers Inf
	expectError(t, `out = 1.0 / 0.0`, nil, "float overflow or division by zero")
	expectError(t, `out = 1.0 / 0`, nil, "float overflow or division by zero")
	expectError(t, `out = 1 / 0.0`, nil, "float overflow or division by zero")
	expectError(t, `1 / 0`, nil, "division_by_zero")
	// overflow raises too; NaN/Inf stay reachable from parses only
	expectError(t, `out = 1e300 * 1e300`, nil, "float overflow or division by zero")
	expectError(t, `out = float("inf") + 1.0`, nil, "float overflow or division by zero")
	expectError(t, `out = float("inf") - float("inf")`, nil, "invalid float arithmetic (NaN result)")
	expectError(t, `out = -float("inf")`, nil, "float overflow or division by zero")
	expectError(t, `out = decimal("340282366920938463463374607431768211455") * 100d`, nil, "decimal overflow") // 2^128-1 — the coefficient ceiling
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
	// repeat is sequence self-concatenation ONLY: the scalar form is gone, and
	// its spellings are the wrap or the count constructor
	expectError(t, `(1).repeat(3)`, nil, "invalid_method: type int has no method repeat")
	expectRun(t, `out = [1].repeat(3)`, nil, ARR{1, 1, 1})
	expectRun(t, `out = array(7, 3)`, nil, ARR{7, 7, 7}) // n copies of x-as-element
	expectError(t, `true.repeat(2)`, nil, "invalid_method")
	expectError(t, `(1.5).repeat(2)`, nil, "invalid_method")
	expectError(t, `undefined.repeat(3)`, nil, "invalid_method")
	expectRun(t, `out = array(true, 2)`, nil, ARR{true, true}) // the count constructor is the spelling

	// decimal & time lost repeat too; the count constructor is the spelling
	expectError(t, `decimal("1.5").repeat(2)`, nil, "invalid_method")
	expectError(t, `time(0).repeat(3)`, nil, "invalid_method")
	expectRun(t, `d := decimal("1.5"); out = array(d, 2).len()`, nil, 2)
	expectRun(t, `d := decimal("1.5"); out = array(d, 2)[0] == d`, nil, true)

	// the element-scalar promoting forms are gone: promotion happens through the
	// sequence constructors, and repeat is sequence self-concatenation only
	expectError(t, `byte(65).repeat(3)`, nil, "invalid_method")
	expectError(t, `'a'.repeat(3)`, nil, "invalid_method")
	expectRun(t, `out = bytes(byte(65), 3)`, nil, []byte{65, 65, 65})

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
	expectError(t, `byte(1).repeat(-1)`, nil, "invalid_method") // repeat left the scalars
	expectError(t, `(1).repeat(-1)`, nil, "invalid_method")

	// wrong arity / arg type
	expectError(t, `"ab".repeat()`, nil, "wrong_num_arguments")
	expectError(t, `"ab".repeat(1, 2)`, nil, "wrong_num_arguments")
	expectError(t, `"ab".repeat([])`, nil, "invalid_argument_type")

	// a count past the ceiling answers a catchable error; it used to panic the host (makeslice), and an
	// empty receiver used to spin the count instead of answering the empty result
	expectError(t, `[1, 2].repeat(9223372036854775807)`, nil, "past the 4294967296 limit")
	expectError(t, `"ab".repeat(9223372036854775807)`, nil, "past the 4294967296 limit")
	expectError(t, `u"ab".repeat(9223372036854775807)`, nil, "past the 4294967296 limit")
	expectError(t, `bytes("ab").repeat(9223372036854775807)`, nil, "past the 4294967296 limit")
	expectRun(t, `out = [].repeat(9223372036854775807)`, nil, ARR{})
	expectRun(t, `out = "".repeat(9223372036854775807)`, nil, "")
}

// TestRepeatOperator pins `*` as repeat's operator form on the four buildable sequences: the right operand is
// a COUNT, not an element, and there is NO reflected direction — `seq * n` reads as "apply n to the
// sequence", while `n * seq` has no such reading.
func TestRepeatOperator(t *testing.T) {
	// member ≡ operator on every type that has repeat
	expectRun(t, `out = [1, 2] * 3`, nil, ARR{1, 2, 1, 2, 1, 2})
	expectRun(t, `out = "ab" * 3`, nil, "ababab")
	expectRun(t, `out = u"ab" * 3`, nil, []rune("ababab"))
	expectRun(t, `out = bytes("ab") * 3`, nil, []byte("ababab"))
	expectRun(t, `out = ([1, 2] * 3) == [1, 2].repeat(3)`, nil, true)
	expectRun(t, `out = ("ab" * 3) == "ab".repeat(3)`, nil, true)
	expectRun(t, `out = (u"ab" * 3) == u"ab".repeat(3)`, nil, true)
	expectRun(t, `out = (bytes("ab") * 3) == bytes("ab").repeat(3)`, nil, true)
	expectRun(t, `out = "-" * 40`, nil, strings.Repeat("-", 40))

	// the count slot is repeat's, verbatim
	expectRun(t, `out = [1, 2] * 0`, nil, ARR{})
	expectRun(t, `out = [1, 2] * 1`, nil, ARR{1, 2})
	expectRun(t, `out = [1, 2] * 2.0`, nil, ARR{1, 2, 1, 2}) // a lossless numeric is a count
	expectError(t, `out = [1, 2] * 2.5`, nil, "must be a whole number")
	expectError(t, `out = [1, 2] * -1`, nil, "repeat count must be non-negative")
	expectError(t, `out = bytes("ab") * 9223372036854775807`, nil, "past the 4294967296 limit")

	// NO reflected direction: a count on the left has no reading
	expectError(t, `out = 3 * [1, 2]`, nil, "invalid_binary_operator: int * array")
	expectError(t, `out = 3 * "ab"`, nil, "invalid_binary_operator: int * string")
	expectError(t, `out = 3 * u"ab"`, nil, "invalid_binary_operator: int * immutable-runes")
	expectError(t, `out = 3 * bytes("ab")`, nil, "invalid_binary_operator: int * bytes")

	// only a number is a count; anything else is not a `*` at all
	expectError(t, `out = [1, 2] * "ab"`, nil, "invalid_binary_operator: array * string")
	expectError(t, `out = [1, 2] * [3]`, nil, "invalid_binary_operator: array * array")
	expectError(t, `out = "ab" * "cd"`, nil, "invalid_binary_operator: string * string")
	expectError(t, `out = "ab" * 'c'`, nil, "invalid_binary_operator: string * rune")
	expectError(t, `out = bytes("ab") * b'c'`, nil, "invalid_binary_operator: bytes * byte")

	// the types without repeat have no `*` either
	expectError(t, `out = range(1, 3) * 3`, nil, "invalid_binary_operator: range * int")
	expectError(t, `out = dict({a: 1}) * 3`, nil, "invalid_binary_operator: dict * int")

	// the universal contracts still come first
	expectRun(t, `out = [1, 2] * undefined`, nil, core.Undefined)
	expectError(t, `out = [1, 2] * error("x")`, nil, "invalid_binary_operator: array * error")

	// `*=` is the usual sugar
	expectRun(t, `a := [1, 2]; a *= 3; out = a`, nil, ARR{1, 2, 1, 2, 1, 2})
	expectRun(t, `s := "ab"; s *= 3; out = s`, nil, "ababab")
}

func TestJoin(t *testing.T) {
	// join is collection-as-receiver ONLY: array and range carry it, the separator never does
	expectRun(t, `out = [1, 2, 3].join(", ")`, nil, "1, 2, 3")
	// default sep
	expectRun(t, `out = [1, 2, 3].join()`, nil, "123")
	// empty seq
	expectRun(t, `out = [].join(", ")`, nil, "")
	// single element
	expectRun(t, `out = [42].join(", ")`, nil, "42")
	// mixed types stringified via AsString (same as `+` operator)
	expectRun(t, `out = [1, "a", true].join(" | ")`, nil, "1 | a | true")
	// undefined is not string-coercible (consistent with `+`)
	expectError(t, `[1, undefined].join(",")`, nil, "cannot convert undefined to string")
	// a nested collection has no canonical text — it raises, never joins silently
	expectError(t, `[[1], [2]].join("-")`, nil, "not renderable in join")

	// the separator-as-receiver forms are GONE — the collection is the receiver
	expectError(t, `", ".join([1, 2, 3])`, nil, "type string has no method join")
	expectError(t, `','.join([1, 2, 3])`, nil, "type rune has no method join")
	expectError(t, `byte(0x2C).join([1, 2, 3])`, nil, "type byte has no method join")
	expectError(t, `u",".join([1, 2, 3])`, nil, "has no method join")

	// runes sep -> runes result; encode to bytes("aXbXc")
	expectRun(t, `out = bytes([1, 2, 3].join(u","))`, nil, []byte{'1', ',', '2', ',', '3'})

	// rune sep -> runes result
	expectRun(t, `out = bytes([1, 2, 3].join(','))`, nil, []byte{'1', ',', '2', ',', '3'})

	// byte sep -> bytes result; a bytes sep too (the result follows the separator's type)
	expectRun(t, `out = [1, 2, 3].join(byte(0x2C))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, `out = [1, 2].join(bytes(", "))`, nil, []byte("1, 2"))

	// range as seq
	expectRun(t, `out = range(1, 4).join(",")`, nil, "1,2,3")
	expectRun(t, `out = range(0, 0).join(",")`, nil, "")

	// errors: wrong sep type for array.join
	expectError(t, `[1, 2].join(123)`, nil, "invalid_argument_type")
	// errors: arity
	expectError(t, `[1, 2].join(",", "x")`, nil, "wrong_num_arguments")
}

func TestSplit(t *testing.T) {
	// string.split — basic literal
	expectRun(t, `out = "a,b,c".split(",")`, nil, ARR{"a", "b", "c"})
	// the limit argument is GONE: a second scalar is another separator now, so the
	// old (sep, limit) spelling raises one way or another rather than silently shifting
	expectError(t, `"a,b,c".split(",", 1)`, nil, "HOMOGENEOUS")    // run + element: mixed set
	expectError(t, `"a,b,c".split(",", -1)`, nil, "invalid_value") // -1 is no code point
	// string.split — whitespace default (the blank set: maximal runs of significant content)
	expectRun(t, `out = "  hello  world  ".split()`, nil, ARR{"hello", "world"})
	// string.split — leading/trailing/consecutive explicit seps keep their empty pieces
	expectRun(t, `out = ",a,".split(",")`, nil, ARR{"", "a", ""})
	expectRun(t, `out = "a,,b".split(",")`, nil, ARR{"a", "", "b"})
	// string.split — sep not found
	expectRun(t, `out = "abc".split("x")`, nil, ARR{"abc"})
	// string.split — empty receiver: n matches answer n+1 pieces, so one empty piece
	expectRun(t, `out = "".split(",")`, nil, ARR{""})
	expectRun(t, `out = "".split()`, nil, ARR{}) // the blank form has no significant content to answer
	// string.split — cross-type sep
	expectRun(t, `out = "a,b".split(',')`, nil, ARR{"a", "b"})
	expectRun(t, `out = "a,b".split(byte(0x2C))`, nil, ARR{"a", "b"})
	expectRun(t, `out = "a,b".split(u",")`, nil, ARR{"a", "b"})

	// the variadic set: every argument one reading — all elements, or all runs
	expectRun(t, `out = "a,b;c".split(',', ';')`, nil, ARR{"a", "b", "c"})
	expectRun(t, `out = "a--b::c".split("--", "::")`, nil, ARR{"a", "b", "c"})
	// runs match leftmost-longest: the longer separator wins at the same position
	expectRun(t, `out = "axxxb".split("xx", "xxx")`, nil, ARR{"a", "b"})
	// the element-level predicate (f/1 or f/2)
	expectRun(t, `out = "a1b2c".split(func(c) { return c == '1' || c == '2' })`, nil, ARR{"a", "b", "c"})
	expectRun(t, `out = "xaxb".split(func(i, c) { return i % 2 == 0 })`, nil, ARR{"", "a", "b"})

	// runes.split
	expectRun(t, `out = bytes(u"a,b,c".split(",")[1])`, nil, []byte{'b'})
	expectRun(t, `out = u"a b c".split().len()`, nil, int64(3))
	expectRun(t, `out = u"".split(",").len()`, nil, int64(1))

	// bytes.split
	expectRun(t, `out = bytes("a,b,c").split(",").len()`, nil, int64(3))
	expectRun(t, `out = bytes("a,b,c").split(byte(0x2C)).len()`, nil, int64(3))
	expectRun(t, `out = bytes("a b c").split().len()`, nil, int64(3))
	expectRun(t, `out = bytes("").split(",").len()`, nil, int64(1))

	// errors and edges
	expectRun(t, `out = "a,b".split("")`, nil, ARR{"a,b"}) // an empty run matches nothing
	expectError(t, `"a,b".split([])`, nil, "invalid_argument_type")
	expectError(t, `"a,b".split(",", func(c) { return true })`, nil, "invalid_argument_type") // a function among several arguments
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

	// the variadic set — leftmost hit wins, longest at that position
	expectRun(t, `out = "a=b:c".partition(":", "=")`, nil, ARR{"a", "=", "b:c"})
	expectRun(t, `out = "axxxb".partition("xx", "xxx")`, nil, ARR{"a", "xxx", "b"})
	// the element-level predicate
	expectRun(t, `out = "ab1cd".partition(func(c) { return c == '1' })`, nil, ARR{"ab", "1", "cd"})
	// the blank no-argument form: the separator is the whole maximal run of filler
	expectRun(t, `out = "key  value more".partition()`, nil, ARR{"key", "  ", "value more"})
	expectRun(t, `out = "abc".partition()`, nil, ARR{"abc", "", ""})

	// errors and edges
	expectRun(t, `out = "a".partition("")`, nil, ARR{"a", "", ""}) // an empty run matches nothing
	expectError(t, `"a".partition([])`, nil, "invalid_argument_type")
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

	//expectError(t, `a := freeze_shallow([4, 5, 6]); b := a[:false];`, nil, "Runtime Error: invalid slice index type: bool")
	expectRun(t, `a := freeze_shallow([4, 5, 6]); out = string(a[:false]);`, nil, "")

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

func TestO3_ModuleExportBindingsAreNotEliminated(t *testing.T) {
	expectRun(t,
		`out = import("iface_mod")`,
		Opts().Module("iface_mod", `
res := 6 * 7
export res
`),
		int64(42),
	)
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

	// module export is deeply immutable: a nested container reachable through it is protected too, not just
	// the top-level exported value.
	expectError(t, `m1 := import("mod1"); m1.a.b = 5`,
		Opts().Module("mod1", `export {a: {b: 3}}`),
		"not_assignable: type immutable-record does not support assignment via indexing or field access")
	// protection reaches arbitrarily deep (array nested two levels down inside a record), not just one level.
	expectError(t, `m1 := import("mod1"); m1.a.b[0] = 99`,
		Opts().Module("mod1", `export {a: {b: [1, 2, 3]}}`),
		"not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, `m1 := import("mod1"); out = is_immutable(m1.a.b)`,
		Opts().Module("mod1", `export {a: {b: [1, 2, 3]}}`), true)

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
			defer func() { log = log.append("a") }()
			log = log.append("b")
		}
		f()
		out = log
	`, nil, ARR{"b", "a"})
}

func TestDefer_LIFOOrder(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = log.append(1) }()
			defer func() { log = log.append(2) }()
			defer func() { log = log.append(3) }()
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
			defer func() { log = log.append("deferred") }()
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
	testFileSet := ast.NewFileSet()
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

// regression: errors raised from core/ as bare fmt.Errorf were FATAL —
// they bypassed recover() and stopped the VM. Each case below is one converted
// class: argument validation, conversion failure, and render/format failure.
func TestRecover_CatchesCoreErrors(t *testing.T) {
	catch := func(expr string) string {
		return `
			f := func() res {
				defer func() {
					e := recover()
					if e != undefined {
						res = "caught: " + e.string()
					}
				}()
				x := ` + expr + `
				res = "no_error"
			}
			out = f()
		`
	}
	// argument validation (the step-16 probe pair: chunk(0) was catchable, repeat(-1) was not)
	expectRun(t, catch(`[1, 2, 3].repeat(-1)`), nil, "caught: (repeat) repeat count must be non-negative, got -1")
	expectRun(t, catch(`"a,b".trim("ab")`), nil,
		"caught: (trim) argument expects type a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading), got string")
	expectRun(t, catch(`bytes("ab").push('é')`), nil, "caught: (push) the value does not fit a single element of the receiver")
	expectRun(t, catch(`decimal("1.5").rescale(99)`), nil, "caught: (rescale) scale must be between 0 and 19")
	// conversion failure
	expectRun(t, catch(`decimal("99999999999999999999999999999999999.9").int()`), nil,
		"caught: cannot convert decimal to int")
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
			defer func() { log = log.append("did defer") }()
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
			defer func() { log = log.append("outer") }()
			defer func() {
				if recover() != undefined {
					log = log.append("recovered")
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
		{"is_immutable/immutable", `is_immutable(freeze_shallow([1, 2]))`, true},
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
	expectRun(t, `out = type_name(func(){})`, nil, "function")
	expectRun(t, `out = type_name(len)`, nil, "function")
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
		for i := 0; i < 5000; i = i + 1 { big = big.append(i) }
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
		for i := 0; i < 5000; i = i + 1 { big = big.append(i) }
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
		for i := 0; i < 500; i = i + 1 { big = big.append(i) }
		out = f(big...)
	`
	expectRun(t, src, nil, 500)
}

func TestSplice_HugeDeleteCountClamps(t *testing.T) {
	// Regression: large positive count must be clamped, not overflow startIdx+delCount.
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := a.splice_in_place(2, 9223372036854775807)
		out = [a, d]
	`, nil, ARR{ARR{1, 2}, ARR{1, 2}})
}

func TestSplice_HugeDeleteCountWithInsertClamps(t *testing.T) {
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := a.splice_in_place(1, 9223372036854775807, "x", "y")
		out = [a, d]
	`, nil, ARR{ARR{1, "x", "y"}, ARR{1, "x", "y"}})
}

func TestSplice_NegativeStart(t *testing.T) {
	// negative indices count from the end, like every positional slot
	expectRun(t, `out = [1,2,3].splice(-1)`, nil, ARR{1, 2})
	expectError(t, `[1,2,3].splice(-4)`, nil, "index_out_of_bounds: (splice, start index)")
}

func TestSplice_StartBeyondLen(t *testing.T) {
	expectError(t, `[1,2,3].splice(4)`, nil, "index_out_of_bounds: (splice, start index)")
}

func TestSplice_NegativeCount_Recoverable(t *testing.T) {
	// Bug fix: negative-count error is now Recoverable so deferred recover() can catch it.
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			[1,2,3].splice(0, -1)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestSplice_OnConstArray_Errors(t *testing.T) {
	// splice() is pure now (P4-004/P4-005) and works regardless of receiver mutability; splice_in_place() is
	// the twin that still requires a mutable receiver.
	expectError(t, `freeze_shallow([1,2,3]).splice_in_place(0)`, nil,
		"not_mutable: (splice_in_place) type immutable-array is immutable")
	expectRun(t, `out = freeze_shallow([1,2,3]).splice(0)`, nil, ARR{})
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
	// range() is valid — the type's zero form (the empty range)
	expectRun(t, `out = range() == range(0, 0)`, nil, true)
	expectError(t, `range(1)`, nil, "wrong_num_arguments: (range) expected 0, 2 or 3")
	expectError(t, `range(1,2,3,4)`, nil, "wrong_num_arguments: (range) expected 0, 2 or 3")
}

func TestRange_NonIntArgs(t *testing.T) {
	expectError(t, `range("a", 1, 1)`, nil, "invalid_argument_type: (range) argument start expects type int")
	expectError(t, `range(0, "b", 1)`, nil, "invalid_argument_type: (range) argument stop expects type int")
	expectError(t, `range(0, 1, "c")`, nil, "invalid_argument_type: (range) argument step expects type int")
}

func TestConstructorFallback_Defaults(t *testing.T) {
	// there is no free default form — the fallible-conversion idiom is the MEMBER's, where the
	// receiver opts into recovery
	expectError(t, `out = int("nope", 42)`, nil, "wrong_num_arguments: (int) expected 0 or 1 argument(s), got 2")
	expectError(t, `out = float("nope", 1.5)`, nil, "wrong_num_arguments: (float) expected 0 or 1 argument(s), got 2")
	expectRun(t, `out = "nope".int(42)`, nil, 42)
	expectRun(t, `out = "nope".float(1.5)`, nil, 1.5)
}

func TestConstructorFallback_NoFallback_Raises(t *testing.T) {
	// a failed conversion raises — the silent undefined is gone
	expectError(t, `int("nope")`, nil, "conversion: cannot convert string to int")
	expectError(t, `float("nope")`, nil, "conversion: cannot convert string to float")
}

func TestConstructorWrongArity(t *testing.T) {
	// scalar constructors are T() | T(x); the four buildable sequences add T(x, count)
	expectError(t, `int(1, 2)`, nil, "wrong_num_arguments: (int) expected 0 or 1 argument(s), got 2")
	expectError(t, `float(1, 2)`, nil, "wrong_num_arguments: (float) expected 0 or 1 argument(s), got 2")
	expectError(t, `bool(1, 2)`, nil, "wrong_num_arguments: (bool) expected 0 or 1 argument(s), got 2")
	expectError(t, `byte(1, 2)`, nil, "wrong_num_arguments: (byte) expected 0 or 1 argument(s), got 2")
	expectError(t, `rune(1, 2)`, nil, "wrong_num_arguments: (rune) expected 0 or 1 argument(s), got 2")
	expectError(t, `decimal(1, 2)`, nil, "wrong_num_arguments: (decimal) expected 0 or 1 argument(s), got 2")
	expectError(t, `time(1, 2)`, nil, "wrong_num_arguments: (time) expected 0 or 1 argument(s), got 2")
	expectError(t, `dict(1, 2)`, nil, "wrong_num_arguments: (dict) expected 0 or 1 argument(s), got 2")
	expectError(t, `string(1, 2, 3)`, nil, "wrong_num_arguments: (string) expected 0 or 1 argument(s), got 3")
	expectError(t, `runes(1, 2, 3)`, nil, "wrong_num_arguments: (runes) expected 0 or 1 argument(s), got 3")
	expectError(t, `bytes(1, 2, 3)`, nil, "wrong_num_arguments: (bytes) expected 0 or 1 argument(s), got 3")
	expectError(t, `array(1, 2, 3)`, nil, "wrong_num_arguments")
}

func TestBuiltinDict_FromInvalidType(t *testing.T) {
	expectError(t, `dict(123)`, nil, "conversion: cannot convert int to dict: no conversion exists")
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
			log = log.append(n)
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
				log = log.append("defer1")
				e := recover()
				if e != undefined { log = log.append("rescued") }
			}()
			defer func() {
				log = log.append("defer2")
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
		f := func(...args) { log = log.push(args) }
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
			defer func() { log = log.append(n) }()
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

func TestBuiltinFormat_OneArg_Renders(t *testing.T) {
	expectRun(t, `out = format("x")`, nil, "x")
	expectError(t, `format()`, nil, "wrong_num_arguments: (format) expected 1 or 2")
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

func TestArith_IntOverflowRaises(t *testing.T) {
	// int is a CHECKED numeric — overflow raises, catchable; wide modular
	// arithmetic left the language (byte is the only modular type)
	expectError(t, `min := -9223372036854775807 - 1; out = -min`, nil, "int overflow")
	expectError(t, `out = 9223372036854775807 + 1`, nil, "int overflow")
	expectError(t, `out = -9223372036854775807 - 2`, nil, "int overflow")
	expectError(t, `out = 9223372036854775807 * 2`, nil, "int overflow")
	expectError(t, `min := -9223372036854775807 - 1; out = min / -1`, nil, "int overflow")
	expectError(t, `out = 1 << 64`, nil, "int overflow")
	expectError(t, `out = 1 << 63`, nil, "int overflow") // the sign bit is not a value bit
	expectRun(t, `out = 1 << 62`, nil, int64(1)<<62)
	expectRun(t, `out = 9223372036854775806 + 1`, nil, int64(9223372036854775807))
	expectRun(t, `min := -9223372036854775807 - 1; out = min % -1`, nil, 0)
	// the raise is catchable, like every argument/value failure
	expectRun(t, `
		f := func() res {
			defer func() { if recover() != undefined { res = "caught" } }()
			return 9223372036854775807 + 1
		}
		out = f()
	`, nil, "caught")
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
	v, err := c2.Get("out")
	require.NoError(t, err)
	require.Equal(t, core.IntValue(7), v)
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

// TestArith_RuneOverflowRaises pins rune's overflow policy on the operator side: arithmetic whose
// result leaves the code-point space (or lands in the surrogate range) raises — it never silently
// becomes U+FFFD. Surrogates are rejected by the same rule, not as a special case.
func TestArith_RuneOverflowRaises(t *testing.T) {
	expectRun(t, `out = 'a' + 1`, nil, 'b')
	expectRun(t, `out = 'b' - 1`, nil, 'a')
	expectError(t, `out = '\U0010FFFF' + 1`, nil, "rune overflow")
	expectError(t, `out = '\x00' - 1`, nil, "rune overflow")
	expectError(t, `out = '퟿' + 1`, nil, "rune overflow") // the first surrogate
}

// TestDictKeyRule pins the index operator's key rule: a key is accepted iff its own string
// conversion exists — so map-, record- and callable-shaped keys raise instead of being silently
// keyed by their render.
func TestDictKeyRule(t *testing.T) {
	expectRun(t, `d := dict(); d[1] = "x"; out = d["1"]`, nil, "x") // an int converts
	expectError(t, `d := dict(); d[dict({})] = 1`, nil, "invalid_index_type")
	expectError(t, `d := dict(); d[{a: 1}] = 1`, nil, "invalid_index_type")
	expectError(t, `d := dict(); d[func(){}] = 1`, nil, "invalid_index_type")
	expectError(t, `d := dict({a: 1}); x := d[dict({})]`, nil, "invalid_index_type")
	expectError(t, `d := dict(); d[undefined] = 1`, nil, "invalid_index_type")
}

// TestForInSoloYieldsElement pins the single-variable for-in binding: the container's ELEMENT —
// the value everywhere except maps, whose element is the KEY. The
// two-variable form is unchanged: (key, value) / (index, element).
func TestForInSoloYieldsElement(t *testing.T) {
	expectRun(t, `out = []; for k in dict({a: 1, b: 2}) { out = out.push(k) }; out = out.sort()`, nil, ARR{"a", "b"})
	expectRun(t, `out = 0; for k in dict({a: 1, b: 2}) { out += k.len() }`, nil, 2)
	expectRun(t, `out = []; for k, v in dict({a: 1}) { out = [k, v] }`, nil, ARR{"a", 1}) // two-var unchanged
	expectRun(t, `out = 0; for _, v in dict({a: 1, b: 2}) { out += v }`, nil, 3)          // values: the _, v form
	expectRun(t, `out = 0; for x in [10, 20] { out += x }`, nil, 30)                      // sequences: the element
	expectRun(t, `out = ""; for c in "ab" { out += c }`, nil, "ab")
	expectRun(t, `out = 0; for x in range(1, 4) { out += x }`, nil, 6)
}

// TestAnyAllDispatchForms pins the three argument forms of `any`/`all` on every receiver that carries
// them — value, predicate, and blank — plus the two boundaries that keep the pair from drifting: a
// sequence argument RAISES (the contiguous-run query belongs to `contains`, not here), and the two
// documented synonyms hold, so a later refactor cannot silently split `contains(f)` from `any(f)` or
// `contains()` from `any()`.
func TestAnyAllDispatchForms(t *testing.T) {
	// the value form — one element, on each receiver
	expectRun(t, `out = [1, 2].any(2)`, nil, true)
	expectRun(t, `out = [1, 2].any(3)`, nil, false)
	expectRun(t, `out = [1, 2].all(1)`, nil, false)
	expectRun(t, `out = [1, 1].all(1)`, nil, true)
	expectRun(t, `out = "abc".any('b')`, nil, true)
	expectRun(t, `out = "abc".all('a')`, nil, false)
	expectRun(t, `out = u"abc".any('b')`, nil, true)
	expectRun(t, `out = bytes("abc").any(b'b')`, nil, true)
	expectRun(t, `out = range(1, 4).any(2)`, nil, true)
	expectRun(t, `out = range(1, 4).all(1)`, nil, false)
	expectRun(t, `out = dict({a: 1}).any("a")`, nil, true) // a map matches on keys
	expectRun(t, `out = dict({a: 1}).all("a")`, nil, true)

	// the blank form — truthiness of the elements; a map has no blank reading
	expectRun(t, `out = [0, 1].any()`, nil, true)
	expectRun(t, `out = [0, 0].any()`, nil, false)
	expectRun(t, `out = [].any()`, nil, false)
	expectRun(t, `out = [1, 2].all()`, nil, true)
	expectRun(t, `out = [].all()`, nil, true) // vacuous truth
	expectRun(t, `out = "abc".any()`, nil, true)
	expectRun(t, `out = "".any()`, nil, false)
	expectRun(t, `out = u"abc".any()`, nil, true)
	expectRun(t, `out = bytes("abc").any()`, nil, true)
	expectRun(t, `out = range(0, 3).any()`, nil, true) // 0 is falsy, 1 and 2 are not
	expectError(t, `out = dict({a: 1}).any()`, nil, "a map has no blank reading")
	expectError(t, `out = dict({a: 1}).all()`, nil, "a map has no blank reading")

	// a sequence argument raises — the run query is contains'
	expectError(t, `out = [1, 2].any([1, 2])`, nil, "this member declares no run reading")
	expectError(t, `out = [1, 2].all([1, 2])`, nil, "this member declares no run reading")
	expectError(t, `out = "abc".any("ab")`, nil, "the contiguous-run query is contains's")
	expectError(t, `out = u"abc".any(u"ab")`, nil, "the contiguous-run query is contains's")
	expectError(t, `out = bytes("abc").any(bytes("ab"))`, nil, "the contiguous-run query is contains's")
	expectError(t, `out = range(1, 4).any(range(1, 3))`, nil, "this member declares no run reading")
	expectError(t, `out = dict({a: 1}).any(dict({a: 1}))`, nil, "the submap reading is deferred")

	// the synonyms: contains(f) == any(f) and contains() == any()
	expectRun(t, `out = [1, 2].contains(x => x > 1) == [1, 2].any(x => x > 1)`, nil, true)
	expectRun(t, `out = [1, 2].contains(x => x > 9) == [1, 2].any(x => x > 9)`, nil, true)
	expectRun(t, `out = [0, 1].contains() == [0, 1].any()`, nil, true)
	expectRun(t, `out = [0, 0].contains() == [0, 0].any()`, nil, true)
	expectRun(t, `out = "abc".contains(c => c == 'b') == "abc".any(c => c == 'b')`, nil, true)
	expectRun(t, `out = "abc".contains() == "abc".any()`, nil, true)
}

// TestInOperatorReadings pins the `in` operator to contains' VALUE readings: element | run,
// with the member's full acceptance — an unacceptable operand RAISES, never answers a silent false —
// and a callable operand raises, because an operator operand is always a value (the predicate reading
// is the member's: contains(f) ≡ any(f)).
func TestInOperatorReadings(t *testing.T) {
	// element and run, same answers as the member
	expectRun(t, `out = 2 in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [2, 3] in [1, 2, 3]`, nil, true)
	expectRun(t, `out = [3, 2] in [1, 2, 3]`, nil, false)
	expectRun(t, `out = range(2, 4) in [1, 2, 3]`, nil, false)        // a range is one element on an array
	expectRun(t, `out = range(2, 4).array() in [1, 2, 3]`, nil, true) // the run reading is spelled by materializing
	expectRun(t, `out = [] in [1, 2]`, nil, true)                     // the empty run is contained everywhere
	expectRun(t, `out = undefined in [1, undefined]`, nil, true)      // an array can hold undefined
	expectRun(t, `out = "bc" in "abc"`, nil, true)
	expectRun(t, `out = b'a' in "abc"`, nil, true)     // an ASCII octet is a symbol
	expectRun(t, `out = 98 in bytes("ab")`, nil, true) // an in-range int is one octet
	expectRun(t, `out = u"bc" in bytes("abc")`, nil, true)
	expectRun(t, `out = 2 in range(1, 4)`, nil, true) // closed form, nothing materialised
	expectRun(t, `out = "a" in dict({a: 1})`, nil, true)
	expectRun(t, `out = "a" in {a: 1}`, nil, true)

	// unacceptable operands raise — the silent false is gone
	expectError(t, `1.5 in "abc"`, nil, "invalid_argument_type")         // no fractional symbol
	expectError(t, `300 in bytes("ab")`, nil, "invalid_value")           // out of the octet range
	expectError(t, `byte(200) in "abc"`, nil, "invalid_value")           // no symbol beyond ASCII
	expectError(t, `[1, 2] in range(0, 9)`, nil, "not_implemented")      // the run reading on a range is deferred
	expectError(t, `dict({}) in dict({a: 1})`, nil, "not_implemented")   // the submap reading is deferred
	expectRun(t, `d := dict(); d[1.5] = "x"; out = 1.5 in d`, nil, true) // a float converts to its key
	expectError(t, `1 in 5`, nil, "invalid_binary_operator")             // no membership on a scalar
	expectError(t, `1 in undefined`, nil, "invalid_binary_operator")     // an absent container is an error, not false
	expectError(t, `"x" in time(0)`, nil, "invalid_binary_operator")

	// a callable operand raises: an operator operand is always a value
	expectError(t, `func(x){ return true } in [1, 2]`, nil, "an operator operand is always a value")
	expectError(t, `func(x){ return true } in "ab"`, nil, "an operator operand is always a value")
	expectError(t, `func(x){ return true } in dict({a: 1})`, nil, "an operator operand is always a value")

	// operator ≡ member on the shared readings
	expectRun(t, `out = ([2, 3] in [1, 2, 3]) == [1, 2, 3].contains([2, 3])`, nil, true)
	expectRun(t, `out = ("bc" in "abc") == "abc".contains("bc")`, nil, true)
	expectRun(t, `out = ("x" in dict({a: 1})) == dict({a: 1}).contains("x")`, nil, true)
}

// TestMemberFunctionInsertSplice pins the positional add pair: splice's inserts take the ADD-SIDE
// reading (an item of the receiver's own kind spreads, anything else is one element — the wrap
// spells the element), and insert(i, ...items) is the element-inserting sibling — each item is ONE
// element, never spreads; on the text triple it VALIDATES (a sequence item raises even at length 1).
// Both are positional EDITS: the position raises out of range.
func TestMemberFunctionInsertSplice(t *testing.T) {
	// splice inserts: the add-side reading (this is the batch's one silent flip on array)
	expectRun(t, `out = [1, 2].splice(1, 0, [8, 9])`, nil, ARR{1, 8, 9, 2})
	expectRun(t, `out = [1, 2].splice(1, 0, range(8, 10))`, nil, ARR{1, core.NewIntRangeValue(8, 10, 1), 2}) // one element
	expectRun(t, `out = [1, 2].splice(1, 0, range(8, 10).array())`, nil, ARR{1, 8, 9, 2})
	expectRun(t, `out = [1, 2].splice(1, 0, "ab")`, nil, ARR{1, "ab", 2})
	expectRun(t, `out = [1, 2].splice(1, 0, [[8, 9]])`, nil, ARR{1, ARR{8, 9}, 2}) // the wrap
	expectRun(t, `a := [1, 2]; a.splice_in_place(1, 0, [8, 9]); out = a`, nil, ARR{1, 8, 9, 2})

	// insert: one element each, at a position; arguments stay in order
	expectRun(t, `out = [1, 2].insert(1, [8, 9])`, nil, ARR{1, ARR{8, 9}, 2}) // never spreads
	expectRun(t, `out = [1, 2].insert(1, 8, 9)`, nil, ARR{1, 8, 9, 2})
	expectRun(t, `out = [1, 2].insert(2, 9)`, nil, ARR{1, 2, 9})                             // i == len appends
	expectRun(t, `out = [1, 2].insert(-1, 9)`, nil, ARR{1, 9, 2})                            // negative counts from the end
	expectRun(t, `out = [1, 2].insert(0)`, nil, ARR{1, 2})                                   // no items: a legal no-op
	expectRun(t, `a := [1, 2]; a.insert(1, 9); out = a`, nil, ARR{1, 2})                     // pure: receiver untouched
	expectRun(t, `a := [1, 2]; b := a; a.insert_in_place(1, 9); out = b`, nil, ARR{1, 9, 2}) // twin: shared struct
	expectRun(t, `a := [1]; out = a.insert_in_place(0, 9)`, nil, ARR{9, 1})                  // twin returns the receiver
	expectError(t, `[1, 2].insert(3, 9)`, nil, "index_out_of_bounds")                        // an edit past the end raises
	expectError(t, `[1, 2].insert()`, nil, "wrong_num_arguments")
	expectError(t, `freeze_shallow([1]).insert_in_place(0, 9)`, nil, "not_mutable")

	// insert on the triple validates: element only, a sequence raises even at length 1
	expectRun(t, `out = bytes("ab").insert(1, b'x')`, nil, []byte("axb"))
	expectRun(t, `out = runes("ab").insert(1, 'x')`, nil, []rune("axb"))
	expectRun(t, `out = "ab".insert(1, 'x')`, nil, "axb")
	expectError(t, `bytes("ab").insert(1, "xy")`, nil, "invalid_argument_type") // splice takes runs
	expectError(t, `bytes("ab").insert(1, 'é')`, nil, "invalid_value")          // two octets, one slot
	expectRun(t, `a := runes("ab"); a.insert_in_place(1, 'x'); out = a`, nil, []rune("axb"))
	expectError(t, `"ab".insert_in_place(1, 'x')`, nil, "type string has no method insert_in_place")
}

// TestNotMutableKind pins the unified refusal: every mutating member on an immutable receiver
// raises ONE kind — not_mutable — whatever the verb, so a script catches "I mutated a frozen
// value" with a single kind test. The assignment statement keeps not_assignable.
func TestNotMutableKind(t *testing.T) {
	expectRun(t, `
		kinds := []
		probes := [
			func() { freeze_shallow([1]).append_in_place(2) },
			func() { freeze_shallow([1]).sort_in_place() },
			func() { freeze_shallow([1]).reverse_in_place() },
			func() { freeze_shallow([1]).remove_in_place(1) },
			func() { freeze_shallow([1]).splice_in_place(0, 1) },
			func() { freeze_shallow([1, 0]).trim_in_place() },
			func() { freeze(bytes("ab")).push_in_place(b'c') },
			func() { freeze_shallow(dict({a: 1})).merge_in_place(dict({b: 2})) },
			func() { freeze_shallow(dict({a: 1})).remove_in_place("a") },
		]
		probes.for_each(func(p) {
			func() {
				defer func() { kinds = kinds.push(recover().kind()) }()
				p()
			}()
		})
		out = kinds.unique()
	`, nil, ARR{"not_mutable"})
	// the assignment STATEMENT keeps not_assignable — it also covers types with no index assignment
	expectError(t, `f := freeze([1]); f[0] = 9`, nil, "not_assignable")
}

// TestLossyCountRaises pins the count/width/position slots: any numeric is accepted iff the
// conversion is lossless — 2.0 is 2, 1.5 raises instead of silently truncating.
func TestLossyCountRaises(t *testing.T) {
	expectRun(t, `out = "ab".repeat(2.0)`, nil, "abab")
	expectError(t, `"ab".repeat(1.5)`, nil, "must be a whole number")
	expectError(t, `[1, 2, 3, 4].chunk(1.5)`, nil, "must be a whole number")
	expectError(t, `"ab".pad_start(3.5)`, nil, "must be a whole number")
	expectError(t, `[1, 2].insert(0.5, 9)`, nil, "must be a whole number")
	expectRun(t, `out = [1, 2, 3, 4].chunk(2.0).len()`, nil, 2)
}

// TestLocatorNeedleAcceptance pins the locators to the receiver's acceptance: a needle the receiver
// cannot read as an element RAISES — exactly as it does on contains/count — instead of scanning to a
// silent miss.
func TestLocatorNeedleAcceptance(t *testing.T) {
	expectError(t, `"abc".index(55296)`, nil, "must be a valid code point") // a surrogate is no symbol
	expectError(t, `"abc".index(1.5)`, nil, "invalid_argument_type")        // no fractional symbol
	expectError(t, `bytes("ab").index(300)`, nil, "must be in [0, 255]")    // out of the octet range
	expectError(t, `u"ab".index_last(byte(200))`, nil, "invalid_value")     // no symbol beyond ASCII
	expectError(t, `range(0, 5).index("x")`, nil, "invalid_argument_type")
	expectRun(t, `out = "abc".index(98)`, nil, 1)       // a valid code point is the element
	expectRun(t, `out = "abc".index('z', -1)`, nil, -1) // a genuine miss still answers the default
	expectRun(t, `out = [1, "x"].index("x")`, nil, 1)   // an array reads anything as an element
}

// TestDictSubParity pins `-`'s key reading to remove's: any operand whose string conversion exists
// names a key, so d - 1 ≡ d.remove(1).
func TestDictSubParity(t *testing.T) {
	expectRun(t, `d := dict(); d[1] = "x"; out = (d - 1).len()`, nil, 0)
	expectRun(t, `d := dict({a: 1}); out = (d - "a").len()`, nil, 0)
	expectError(t, `dict({a: 1}) - dict({})`, nil, "invalid_binary_operator") // a submap is not a key
	expectRun(t, `out = (dict({a: 1}) - undefined) == undefined`, nil, true)  // undefined propagates
}

// TestSpliceNegativeStart pins splice's start to the uniform rule: negative indices count from the end.
func TestSpliceNegativeStart(t *testing.T) {
	expectRun(t, `out = [1, 2, 3].splice(-1, 1)`, nil, ARR{1, 2})
	expectRun(t, `out = [1, 2, 3].splice(-2, 1, 9)`, nil, ARR{1, 9, 3})
	expectError(t, `[1, 2, 3].splice(-4, 1)`, nil, "index_out_of_bounds")
	expectRun(t, `out = "abc".splice(-1, 1)`, nil, "ab")
}

// TestRuneRosterParity pins the element-scalar twins' shared cells: rune carries runes() (the text
// targets compose through string), int's abs() raises on the one value whose magnitude does not fit,
// and every total conversion carries the never-firing default slot.
func TestRuneRosterParity(t *testing.T) {
	expectRun(t, `out = 'є'.runes()`, nil, []rune("є"))
	expectRun(t, `out = 'A'.runes() == b'A'.runes()`, nil, true)
	expectRun(t, `out = 'є'.string("?")`, nil, "є") // the default is carried, never fires
	expectError(t, `min := -9223372036854775807 - 1; out = min.abs()`, nil, "int overflow")
	expectRun(t, `out = (-5).abs()`, nil, 5)
}
