package unit

import (
	"errors"
	"fmt"
	"maps"
	_runtime "runtime"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/opcode"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/stdlib"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/token"
	"github.com/jokruger/kavun/vm"
)

type IARR []any
type IMAP map[string]any
type MAP = map[string]any
type ARR = []any

const testOut = "out"

var cta = core.NewArena(nil)
var rta = core.NewArena(nil)

var (
	VT_COUNTER               = core.VT_USER_DEFINED + 1
	VT_CUSTOM_NUMBER         = core.VT_USER_DEFINED + 2
	VT_STRING_ARRAY          = core.VT_USER_DEFINED + 3
	VT_STRING_CIRCLE         = core.VT_USER_DEFINED + 4
	VT_STRING_DICT           = core.VT_USER_DEFINED + 5
	VT_STRING_ARRAY_ITERATOR = core.VT_USER_DEFINED + 6
)

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

type vmTracer struct {
	Out []string
}

func (o *vmTracer) Write(p []byte) (n int, err error) {
	o.Out = append(o.Out, string(p))
	return len(p), nil
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

type Counter struct {
	value int64
}

func NewCounterValue(val int64) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&Counter{value: val}),
		Type: VT_COUNTER,
	}
}

func toCounter(a *core.Arena, v core.Value) *Counter {
	if v.Type != VT_COUNTER {
		panic(fmt.Sprintf("invalid type: expected Counter, got %s", v.TypeName(a)))
	}
	return (*Counter)(v.Ptr)
}

type CustomNumber struct {
	value int64
}

func NewCustomNumberValue(val int64) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&CustomNumber{value: val}),
		Type: VT_CUSTOM_NUMBER,
	}
}

func toCustomNumber(a *core.Arena, v core.Value) *CustomNumber {
	if v.Type != VT_CUSTOM_NUMBER {
		panic(fmt.Sprintf("invalid type: expected CustomNumber, got %s", v.TypeName(a)))
	}
	return (*CustomNumber)(v.Ptr)
}

type StringArray struct {
	Value []string
}

func NewStringArrayValue(vals []string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringArray{Value: vals}),
		Type: VT_STRING_ARRAY,
	}
}

func toStringArray(a *core.Arena, v core.Value) *StringArray {
	if v.Type != VT_STRING_ARRAY {
		panic(fmt.Sprintf("invalid type: expected StringArray, got %s", v.TypeName(a)))
	}
	return (*StringArray)(v.Ptr)
}

type StringCircle struct {
	Value []string
}

func NewStringCircleValue(vals []string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringCircle{Value: vals}),
		Type: VT_STRING_CIRCLE,
	}
}

func toStringCircle(v core.Value) *StringCircle {
	if v.Type != VT_STRING_CIRCLE {
		panic(fmt.Sprintf("invalid type: expected StringCircle, got %s", v.TypeName(rta)))
	}
	return (*StringCircle)(v.Ptr)
}

type StringDict struct {
	Value map[string]string
}

func NewStringDictValue(vals map[string]string) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringDict{Value: vals}),
		Type: VT_STRING_DICT,
	}
}

func toStringDict(v core.Value) *StringDict {
	if v.Type != VT_STRING_DICT {
		panic(fmt.Sprintf("invalid type: expected StringDict, got %s", v.TypeName(rta)))
	}
	return (*StringDict)(v.Ptr)
}

type StringArrayIterator struct {
	strArr *StringArray
	idx    int
}

func NewStringArrayIteratorValue(arr *StringArray) core.Value {
	return core.Value{
		Ptr:  unsafe.Pointer(&StringArrayIterator{strArr: arr, idx: 0}),
		Type: VT_STRING_ARRAY_ITERATOR,
	}
}

func toStringArrayIterator(v core.Value) *StringArrayIterator {
	if v.Type != VT_STRING_ARRAY_ITERATOR {
		panic(fmt.Sprintf("invalid type: expected StringArrayIterator, got %s", v.TypeName(rta)))
	}
	return (*StringArrayIterator)(v.Ptr)
}

func init() {
	// Register Counter
	core.SetValueType(VT_COUNTER, core.ValueTypeDescr{
		Interface: func(a *core.Arena, v core.Value) any { return toCounter(a, v) },
		Name:      func(a *core.Arena, v core.Value) string { return "counter" },
		String:    func(a *core.Arena, v core.Value) string { return fmt.Sprintf("Counter(%d)", toCounter(a, v).value) },
		AsString:  func(a *core.Arena, v core.Value) (string, bool) { return v.String(a), true },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == core.VT_INT {
				o := toCounter(a, v)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + int64(rhs.Data)), nil
				case token.Sub:
					return NewCounterValue(o.value - int64(rhs.Data)), nil
				}
			}
			if rhs.Type == VT_COUNTER {
				o := toCounter(a, v)
				r := toCounter(a, rhs)
				switch op {
				case token.Add:
					return NewCounterValue(o.value + r.value), nil
				case token.Sub:
					return NewCounterValue(o.value - r.value), nil
				}
			}
			return core.Undefined, errors.New("invalid operator")
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return toCounter(a, v).value != 0 },
		Equal: func(a *core.Arena, v core.Value, r core.Value) bool {
			if r.Type != VT_COUNTER {
				return false
			}
			return toCounter(a, v).value == toCounter(a, r).value
		},
		Clone: func(a *core.Arena, v core.Value) (core.Value, error) {
			return NewCounterValue(toCounter(a, v).value), nil
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			return core.IntValue(toCounter(a, v).value), nil
		},
		IsCallable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register CustomNumber
	core.SetValueType(VT_CUSTOM_NUMBER, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "Number" },
		String: func(a *core.Arena, v core.Value) string { return strconv.FormatInt(toCustomNumber(a, v).value, 10) },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			r, ok := rhs.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), rhs.TypeName(rta))
			}
			i := toCustomNumber(a, v).value
			switch op {
			case token.Less:
				return core.BoolValue(i < r), nil
			case token.Greater:
				return core.BoolValue(i > r), nil
			case token.LessEq:
				return core.BoolValue(i <= r), nil
			case token.GreaterEq:
				return core.BoolValue(i >= r), nil
			}
			t := core.IntValue(i)
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), t.TypeName(rta))
		},
	})

	// Register StringArray
	core.SetValueType(VT_STRING_ARRAY, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array" },
		String: func(a *core.Arena, v core.Value) string { return strings.Join(toStringArray(a, v).Value, ", ") },
		BinaryOp: func(a *core.Arena, v core.Value, rhs core.Value, op token.Token) (core.Value, error) {
			if rhs.Type == VT_STRING_ARRAY && op == token.Add {
				l := toStringArray(a, v)
				r := toStringArray(a, rhs)
				if len(r.Value) == 0 {
					return v, nil
				}
				return NewStringArrayValue(append(l.Value, r.Value...)), nil
			}
			return core.Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(rta), rhs.TypeName(rta))
		},
		IsTrue: func(a *core.Arena, v core.Value) bool { return len(toStringArray(a, v).Value) != 0 },
		Equal: func(a *core.Arena, v core.Value, rhs core.Value) bool {
			if rhs.Type == VT_STRING_ARRAY {
				l := toStringArray(a, v)
				r := toStringArray(a, rhs)
				if len(l.Value) != len(r.Value) {
					return false
				}
				for i, v := range l.Value {
					if v != r.Value[i] {
						return false
					}
				}
				return true
			}
			return false
		},
		Clone: func(a *core.Arena, v core.Value) (core.Value, error) {
			return NewStringArrayValue(append([]string{}, toStringArray(a, v).Value...)), nil
		},
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			o := toStringArray(a, v)
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					return a.NewStringValue(o.Value[intIdx]), nil
				}
				return core.Undefined, errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			strIdx, ok := index.AsString(a)
			if ok {
				for vidx, str := range o.Value {
					if strIdx == str {
						return core.IntValue(int64(vidx)), nil
					}
				}
				return core.Undefined, nil
			}
			return core.Undefined, errs.NewInvalidIndexTypeError("StringArray access", "int or string", index.TypeName(rta))
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			o := toStringArray(a, v)
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringArray assignment", "string(compatible)", value.TypeName(rta))
			}
			intIdx, ok := index.AsInt(a)
			if ok {
				if intIdx >= 0 && intIdx < int64(len(o.Value)) {
					o.Value[intIdx] = strVal
					return nil
				}
				return errs.NewIndexOutOfBoundsError("StringArray assignment", int(intIdx), len(o.Value))
			}
			return errs.NewInvalidIndexTypeError("StringArray assignment", "int", v.TypeName(rta))
		},
		Call: func(a *core.Arena, vm core.VM, v core.Value, args []core.Value) (core.Value, error) {
			if len(args) != 1 {
				return core.Undefined, errs.NewWrongNumArgumentsError("StringArray.Call", "1", len(args))
			}
			s1, ok := args[0].AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("StringArray.Call", "first", "string(compatible)", args[0].TypeName(rta))
			}
			o := toStringArray(a, v)
			for i, v := range o.Value {
				if v == s1 {
					return core.IntValue(int64(i)), nil
				}
			}
			return core.Undefined, nil
		},
		IsCallable: func(a *core.Arena, v core.Value) bool { return true },
		Iterator: func(a *core.Arena, v core.Value) (core.Value, error) {
			return NewStringArrayIteratorValue(toStringArray(a, v)), nil
		},
		IsIterable: func(a *core.Arena, v core.Value) bool { return true },
	})

	// Register StringCircle
	core.SetValueType(VT_STRING_CIRCLE, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-circle" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringCircle access", "int", index.TypeName(rta))
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			return a.NewStringValue(o.Value[r]), nil
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			intIdx, ok := index.AsInt(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "int", index.TypeName(rta))
			}
			o := toStringCircle(v)
			r := int(intIdx) % len(o.Value)
			if r < 0 {
				r = len(o.Value) + r
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringCircle assignment", "string(compatible)", value.TypeName(rta))
			}
			o.Value[r] = strVal
			return nil
		},
	})

	// Register StringDict
	core.SetValueType(VT_STRING_DICT, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-dict" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Access: func(a *core.Arena, v core.Value, index core.Value, mode opcode.Opcode) (core.Value, error) {
			strIdx, ok := index.AsString(a)
			if !ok {
				return core.Undefined, errs.NewInvalidIndexTypeError("StringDict access", "string", index.TypeName(rta))
			}
			o := toStringDict(v)
			for k, v := range o.Value {
				if strings.EqualFold(strIdx, k) {
					return a.NewStringValue(v), nil
				}
			}
			return core.Undefined, nil
		},
		Assign: func(a *core.Arena, v core.Value, index core.Value, value core.Value) error {
			strIdx, ok := index.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string", index.TypeName(rta))
			}
			strVal, ok := value.AsString(a)
			if !ok {
				return errs.NewInvalidIndexTypeError("StringDict assignment", "string(compatible)", value.TypeName(rta))
			}
			o := toStringDict(v)
			o.Value[strings.ToLower(strIdx)] = strVal
			return nil
		},
	})

	// Register StringArrayIterator
	core.SetValueType(VT_STRING_ARRAY_ITERATOR, core.ValueTypeDescr{
		Name:   func(a *core.Arena, v core.Value) string { return "string-array-iterator" },
		String: func(a *core.Arena, v core.Value) string { return "" },
		Next: func(a *core.Arena, v core.Value) bool {
			i := toStringArrayIterator(v)
			i.idx++
			return i.idx <= len(i.strArr.Value)
		},
		Key: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return core.IntValue(int64(i.idx - 1)), nil
		},
		Value: func(a *core.Arena, v core.Value) (core.Value, error) {
			i := toStringArrayIterator(v)
			return a.NewStringValue(i.strArr.Value[i.idx-1]), nil
		},
	})
}

func formatGlobals(globals []core.Value) (formatted []string) {
	for idx, global := range globals {
		if global.Type == core.VT_UNDEFINED {
			return
		}
		formatted = append(formatted, fmt.Sprintf("[% 3d] %s (%s|%v)", idx, global.String(rta), global.TypeName(rta), global))
	}
	return
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
				_, file, line, ok := _runtime.Caller(i)
				if !ok {
					break
				}
				stackTrace = append(stackTrace,
					fmt.Sprintf("  %s:%d", file, line))
			}

			trace = append(trace, fmt.Sprintf("[Error Trace]\n\n  %s\n", strings.Join(stackTrace, "\n  ")))
		}
	}()

	globals := make([]core.Value, vm.GlobalsSize)

	symTable := vm.NewSymbolTable()
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
	c := compiler.New(cta, file.InputFile, symTable, nil, customModules, tr)
	err = c.Compile(file)
	trace = append(trace, fmt.Sprintf("\n[Compiler Trace]\n\n%s", strings.Join(tr.Out, "")))
	if err != nil {
		return
	}

	bytecode := c.Bytecode()
	err = bytecode.RemoveDuplicates(rta)
	if err != nil {
		return
	}
	trace = append(trace, fmt.Sprintf("\n[Compiled Constants]\n\n%s", strings.Join(bytecode.MustFormatConstants(rta), "\n")))
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
		trace = append(trace, fmt.Sprintf("\n[Globals]\n\n%s", strings.Join(formatGlobals(globals), "\n")))
	}
	if err == nil && !machine.IsStackEmpty() {
		err = errors.New("non empty stack after execution")
	}

	return
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

func parse(t *testing.T, input string) *parser.File {
	testFileSet := parser.NewFileSet()
	testFile := testFileSet.AddFile("test", -1, len(input))

	p := parser.NewParser(testFile, []byte(input), nil)
	file, err := p.ParseFile()
	require.NoError(t, err)
	return file
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
	expectedObj := toObject(rta, expected)

	if symbols == nil {
		symbols = make(map[string]core.Value)
	}
	symbols[testOut] = objectZeroCopy(rta, expectedObj)

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
		require.Equal(t, rta, expectedObj, res[testOut], "\n"+strings.Join(trace, "\n"))
	}

	// second pass: run the code as import module
	if !opts.skip2ndPass {
		file := parse(t, `out = import("__code__")`)
		if file == nil {
			return
		}

		expectedObj := toObject(rta, expected)
		switch expectedObj.Type {
		case core.VT_ARRAY:
			eo := (*core.Array)(expectedObj.Ptr)
			expectedObj = rta.NewArrayValue(eo.Elements, true)
		case core.VT_RECORD:
			eo := (*core.Dict)(expectedObj.Ptr)
			expectedObj = rta.NewRecordValue(eo.Elements, true)
		case core.VT_DICT:
			eo := (*core.Dict)(expectedObj.Ptr)
			expectedObj = rta.NewDictValue(eo.Elements, true)
		}

		modules := maps.Clone(opts.customModules)
		modules["__code__"] = []byte(fmt.Sprintf("out := undefined; %s; export out", input))

		res, trace, err := traceCompileRun(rta, file, symbols, modules, opts.customBuiltinModules)
		require.NoError(t, err, "\n"+strings.Join(trace, "\n"))
		require.Equal(t, rta, expectedObj, res[testOut], "\n"+strings.Join(trace, "\n"))
	}
}

func errorObject(a *core.Arena, v any) core.Value {
	if s, ok := v.(string); ok {
		return a.NewErrorValue(a.NewStringValue(s), core.KindUser, false)
	}
	return a.NewErrorValue(toObject(a, v), core.KindUser, false)
}

func toObject(a *core.Arena, v any) core.Value {
	switch v := v.(type) {
	case core.Value:
		return v
	case nil:
		return core.Undefined
	case string:
		return a.NewStringValue(v)
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
		return a.NewDecimalValue(v)
	case []byte:
		return a.NewBytesValue(v, false)
	case []rune:
		return a.NewRunesValue(v, false)
	case MAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = toObject(a, v)
		}
		return a.NewRecordValue(objs, false)
	case ARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, toObject(a, e))
		}
		return a.NewArrayValue(objs, false)
	case IMAP:
		objs := make(map[string]core.Value)
		for k, v := range v {
			objs[k] = toObject(a, v)
		}
		return a.NewRecordValue(objs, true)
	case IARR:
		var objs []core.Value
		for _, e := range v {
			objs = append(objs, toObject(a, e))
		}
		return a.NewArrayValue(objs, true)
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
		return a.NewDecimalValue(dec128.Zero)

	case core.VT_RUNE:
		return core.RuneValue(0)

	case core.VT_STRING:
		return a.NewStringValue("")

	case core.VT_RUNES:
		return a.NewRunesValue([]rune(""), false)

	case core.VT_ARRAY:
		return a.NewArrayValue(nil, o.Immutable)

	case core.VT_RECORD:
		return a.NewRecordValue(nil, o.Immutable)

	case core.VT_DICT:
		return a.NewDictValue(nil, o.Immutable)

	case core.VT_ERROR:
		return a.NewErrorValue(core.Undefined, core.KindUser, false)

	case core.VT_BYTES:
		return a.NewBytesValue(nil, false)

	default:
		panic(fmt.Errorf("unknown value kind: %d", o.Type))
	}
}
