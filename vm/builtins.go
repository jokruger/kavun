package vm

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/module"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

var BuiltinFunctions = make(map[string]core.Value)
var BuiltinFunctionNames []string

func init() {
	// 45..127 reserved
	fns := map[uint64]*core.BuiltinFunction{
		7:  core.NewBuiltinFunction("bool", builtinBool, 0, true, true),
		38: core.NewBuiltinFunction("byte", builtinByte, 0, true, true),
		9:  core.NewBuiltinFunction("rune", builtinRune, 0, true, true),
		6:  core.NewBuiltinFunction("int", builtinInt, 0, true, true),
		8:  core.NewBuiltinFunction("float", builtinFloat, 0, true, true),
		34: core.NewBuiltinFunction("decimal", builtinDecimal, 0, true, true),
		11: core.NewBuiltinFunction("time", builtinTime, 0, true, true),
		5:  core.NewBuiltinFunction("string", builtinString, 0, true, true),
		36: core.NewBuiltinFunction("runes", builtinRunes, 0, true, true),
		10: core.NewBuiltinFunction("bytes", builtinBytes, 0, true, true),
		21: core.NewBuiltinFunction("dict", builtinDict, 0, true, true),
		45: core.NewBuiltinFunction("dict_view", builtinDictView, 0, true, true),
		46: core.NewBuiltinFunction("record", builtinRecord, 0, true, true),
		47: core.NewBuiltinFunction("record_view", builtinRecordView, 0, true, true),
		30: core.NewBuiltinFunction("range", builtinRange, 2, true, true),
		33: core.NewBuiltinFunction("error", builtinError, 0, true, true),
		42: core.NewBuiltinFunction("array", builtinArray, 0, true, true),

		15: core.NewBuiltinFunction("is_bool", builtinIsBool, 1, false, true),
		39: core.NewBuiltinFunction("is_byte", builtinIsByte, 1, false, true),
		16: core.NewBuiltinFunction("is_rune", builtinIsRune, 1, false, true),
		12: core.NewBuiltinFunction("is_int", builtinIsInt, 1, false, true),
		13: core.NewBuiltinFunction("is_float", builtinIsFloat, 1, false, true),
		35: core.NewBuiltinFunction("is_decimal", builtinIsDecimal, 1, false, true),
		23: core.NewBuiltinFunction("is_time", builtinIsTime, 1, false, true),
		14: core.NewBuiltinFunction("is_string", builtinIsString, 1, false, true),
		37: core.NewBuiltinFunction("is_runes", builtinIsRunes, 1, false, true),
		17: core.NewBuiltinFunction("is_bytes", builtinIsBytes, 1, false, true),
		18: core.NewBuiltinFunction("is_array", builtinIsArray, 1, false, true),
		31: core.NewBuiltinFunction("is_dict", builtinIsDict, 1, false, true),
		20: core.NewBuiltinFunction("is_record", builtinIsRecord, 1, false, true),
		32: core.NewBuiltinFunction("is_range", builtinIsRange, 1, false, true),
		24: core.NewBuiltinFunction("is_error", builtinIsError, 1, false, true),

		25: core.NewBuiltinFunction("is_undefined", builtinIsUndefined, 1, false, true),
		26: core.NewBuiltinFunction("is_function", builtinIsFunction, 1, false, true),
		27: core.NewBuiltinFunction("is_callable", builtinIsCallable, 1, false, true),
		22: core.NewBuiltinFunction("is_iterable", builtinIsIterable, 1, false, true),
		19: core.NewBuiltinFunction("is_immutable", builtinIsImmutable, 1, false, true),

		0:  core.NewBuiltinFunction("len", builtinLen, 1, false, true),
		1:  core.NewBuiltinFunction("copy", builtinCopy, 1, false, true),
		2:  core.NewBuiltinFunction("copy_shallow", builtinCopyShallow, 1, false, true),
		3:  core.NewBuiltinFunction("remove", builtinRemove, 2, false, true), // pure: returns a container without the key
		4:  core.NewBuiltinFunction("remove_in_place", builtinRemoveInPlace, 2, false, false),
		48: core.NewBuiltinFunction("freeze", builtinFreeze, 1, false, true),
		49: core.NewBuiltinFunction("freeze_shallow", builtinFreezeShallow, 1, false, true), // pure: never mutates shared storage, only returns a new header (see docs/purity.md)
		29: core.NewBuiltinFunction("format", builtinFormat, 1, true, true),
		28: core.NewBuiltinFunction("type_name", builtinTypeName, 1, false, true),
		40: core.NewBuiltinFunction("raise", builtinRaise, 1, true, false),
		41: core.NewBuiltinFunction("recover", builtinRecover, 0, false, false),
		43: core.NewBuiltinFunction("min", builtinMin, 0, true, true),
		44: core.NewBuiltinFunction("max", builtinMax, 0, true, true),
		50: core.NewBuiltinFunction("is_true", builtinIsTrue, 1, false, true),
		51: core.NewBuiltinFunction("is_view", builtinIsView, 1, false, true),
		52: core.NewBuiltinFunction("require", builtinRequire, 2, false, false),
	}

	for i, fn := range fns {
		id := uint64(module.Global) + i
		core.BuiltinFunctions[id] = fn
		BuiltinFunctions[fn.Name] = core.BuiltinFunctionValue(id)
	}

	BuiltinFunctionNames = make([]string, 0, len(fns))
	for id := module.Global; id < module.Global+core.ModuleSlotSize; id++ {
		if fn := core.BuiltinFunctions[id]; fn != nil {
			BuiltinFunctionNames = append(BuiltinFunctionNames, fn.Name)
		}
	}
}

func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	// classification names types: scripts branch on bare lowercase names, so the
	// three function kinds answer "function" uniformly and the kind/arity detail
	// lives in the render (format(), f-strings). Custom host-defined callable
	// types keep their own registered name — is_callable is their unifier.
	switch args[0].Type {
	case value.CompiledFunction, value.BuiltinFunction, value.BuiltinClosure:
		return core.NewStringValue("function"), nil
	}
	return core.NewStringValue(args[0].TypeName()), nil
}

func builtinIsString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	if args[0].Type == value.String {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRunes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_runes", "1", len(args))
	}
	if args[0].Type == value.Runes {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	if args[0].Type == value.Int {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	if args[0].Type == value.Float {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_decimal", "1", len(args))
	}
	if args[0].Type == value.Decimal {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	if args[0].Type == value.Bool {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsByte(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_byte", "1", len(args))
	}
	if args[0].Type == value.Byte {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRune(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_rune", "1", len(args))
	}
	if args[0].Type == value.Rune {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	if args[0].Type == value.Bytes {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsArray(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	if args[0].Type == value.Array {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRecord(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	if args[0].Type == value.Record {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDict(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_dict", "1", len(args))
	}
	if args[0].Type == value.Dict {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRange(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_range", "1", len(args))
	}
	if args[0].Type == value.IntRange {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsImmutable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	// true unless the value can be mutated — exceptionless; undefined cannot be, so it answers true
	// even though the constant's header does not carry the flag
	return core.BoolValue(args[0].Immutable || args[0].Type == value.Undefined), nil
}

func builtinIsTime(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	if args[0].Type == value.Time {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsError(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	if args[0].Type == value.Error {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsUndefined(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	if args[0].Type == value.Undefined {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFunction(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_function", "1", len(args))
	}

	switch args[0].Type {
	case value.BuiltinFunction, value.BuiltinClosure, value.CompiledFunction:
		return core.True, nil
	default:
		return core.False, nil
	}
}

func builtinIsCallable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return core.BoolValue(args[0].IsCallable()), nil
}

func builtinIsIterable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return core.BoolValue(args[0].IsIterable()), nil
}

// is_view(x) => bool — a storage-state predicate: does this value share backing storage with another value?
// Free-only with universal domain (is_view(5) is false, not an error): predicates ABOUT a value — type,
// capability, storage — read the header and must answer honestly on every value, record views included, which
// no member could reach.
func builtinIsView(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_view", "1", len(args))
	}
	switch args[0].Type {
	case value.Array:
		return core.BoolValue((*core.Array)(args[0].Ptr).IsView), nil
	case value.Bytes:
		return core.BoolValue((*core.Bytes)(args[0].Ptr).IsView), nil
	case value.Runes:
		return core.BoolValue((*core.Runes)(args[0].Ptr).IsView), nil
	case value.Dict:
		return core.BoolValue((*core.Dict)(args[0].Ptr).IsView), nil
	case value.Record:
		return core.BoolValue((*core.Record)(args[0].Ptr).IsView), nil
	}
	return core.False, nil
}

// is_true(x) => bool — the boolean-context test (the same answer as !!x and `if x`),
// as a callable form: arr.keep(is_true). NOT equality with `true`: is_true([1])
// is true while [1] == true is false. Raises on an error-state value (NaN).
func builtinIsTrue(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_true", "1", len(args))
	}
	t, err := args[0].IsTrue()
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(t), nil
}

// len(obj object) => int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("len", "1", len(args))
	}
	// the free form and the member are one operation with one domain: the types
	// that have a length. len(5) answering 1 was total and nonsensical — record
	// is why the free form exists at all (it can never have members)
	switch args[0].Type {
	case value.String, value.Runes, value.Bytes, value.Array, value.Dict, value.Record, value.IntRange:
		return core.IntValue(args[0].Len()), nil
	}
	if args[0].Type >= value.FirstUserDefinedType {
		return core.IntValue(args[0].Len()), nil
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("len", "first", "a value with a length", args[0].TypeName())
}

// min(args...) => smallest argument, by BinaryOp(Less); 0 args => undefined, 1 arg => that arg unchanged.
func builtinMin(vm core.VM, args []core.Value) (core.Value, error) {
	// variadic selection over ARGUMENTS (min(a, b, ...)); the member arr.min()
	// is the aggregation over elements — one meaning, two delivery mechanisms.
	// A zero-argument selection has no answer and raises.
	if len(args) == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("min", "1 or more", 0)
	}
	return minMaxReduce(args, token.Less)
}

// max(args...) => largest argument, by BinaryOp(Greater); 0 args => undefined, 1 arg => that arg unchanged.
func builtinMax(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("max", "1 or more", 0)
	}
	return minMaxReduce(args, token.Greater)
}

func minMaxReduce(args []core.Value, op token.Token) (core.Value, error) {
	if len(args) == 0 {
		return core.Undefined, nil
	}

	best := args[0]
	for i := 1; i < len(args); i++ {
		better, err := args[i].BinaryOp(op, best)
		if err != nil {
			return core.Undefined, err
		}
		bt, terr := better.IsTrue()
		if terr != nil {
			return core.Undefined, terr
		}
		if bt {
			best = args[i]
		}
	}

	return best, nil
}

// error(val) creates a (recoverable) Kavun error value with the given payload.
// error(val, fatal) — if fatal is true, the resulting error, when raised, bypasses recover() and stops the VM,
// propagating to the host caller.
// error(payload[, fatal]) wraps any value as a kind "user" error. An argument that is ALREADY an error is
// never re-wrapped: a bare error(err) added an unlabelled layer and relabelled the kind to "user", so a
// caught runtime error lost its own kind()/is_runtime() identity just by passing through here. It answers
// the error itself instead (error is immutable, so the receiver already IS the independent value — the same
// rule string follows), and error(err, fatal) answers a copy carrying the requested severity with the
// payload and kind intact — the same operation raise(err, fatal) already performed. An annotated chain is
// spelled with a payload that names the cause: error({msg: "...", cause: err}).
func builtinError(vm core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 1:
		if args[0].Type == value.Error {
			return args[0], nil
		}
		return core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, false), nil
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("error", "second", "bool", args[1].TypeName())
		}
		if args[0].Type == value.Error {
			o := (*core.Error)(args[0].Ptr)
			if o.Fatal == fatal {
				return args[0], nil
			}
			return core.NewErrorValue(o.Payload, o.Kind, o.Category, fatal), nil
		}
		return core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, fatal), nil
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("error", "1 or 2", len(args))
	}
}

// raise(err) raises the given error so that surrounding deferred recover() calls can catch it. If `err` is not already
// an error value, it is wrapped in a fresh recoverable error.
// raise(err, fatal) — explicitly sets the severity of the raised error: a fatal error bypasses recover() and stops the
// VM. If `err` is an existing error value, a copy with the requested severity is raised (the original is left
// untouched).
func builtinRaise(vm core.VM, args []core.Value) (core.Value, error) {
	var val core.Value
	switch len(args) {
	case 1:
		val = args[0]
		if val.Type != value.Error {
			val = core.NewErrorValue(val, core.KindUser, errs.CategoryUser, false)
		}
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("raise", "second", "bool", args[1].TypeName())
		}
		if args[0].Type == value.Error {
			o := (*core.Error)(args[0].Ptr)
			val = core.NewErrorValue(o.Payload, o.Kind, o.Category, fatal)
		} else if fatal {
			val = core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, true)
		} else {
			val = core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, false)
		}
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("raise", "1 or 2", len(args))
	}
	return core.Undefined, newRaisedError(val)
}

// require(cond, payload) is the input check that opens a script: if cond is true the call answers undefined and the
// script carries on; otherwise it raises a recoverable error of kind "requirement" carrying `payload` untouched, so
// a host reading RuntimeError.Payload gets whatever structure the script chose (a string, a dict, a record).
//
// Not pure: whether it raises depends on runtime state, so the optimizer must never fold it away.
func builtinRequire(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("require", "2", len(args))
	}
	ok, err := args[0].IsTrue()
	if err != nil {
		return core.Undefined, err
	}
	if ok {
		return core.Undefined, nil
	}
	return core.Undefined, newRaisedError(
		core.NewErrorValue(args[1], errs.KindRequirement, errs.CategoryRequirement, false))
}

// recover() returns the in-flight Kavun error caught by a deferred function and clears it (so the surrounding function
// returns normally). Outside a deferred function, or when there is no error in flight, it returns undefined.
// Must be called directly inside a deferred function — any indirection returns undefined.
func builtinRecover(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("recover", "0", len(args))
	}
	return vm.Recover(), nil
}

// range() | range(start, stop[, step])
func builtinRange(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	// range() is the type's zero form — the empty range — so generic code can
	// spell "the zero value of range" the way it can for every other type
	if numArgs == 0 {
		return core.NewIntRangeValue(0, 0, 1), nil
	}
	// a components MAP rebuilds the range: {start, stop[, step]}, strict keys
	if args[0].Type == value.Dict || args[0].Type == value.Record {
		if numArgs > 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("range", "1", numArgs)
		}
		var m map[string]core.Value
		if args[0].Type == value.Dict {
			m = (*core.Dict)(args[0].Ptr).Elements
		} else {
			m = (*core.Record)(args[0].Ptr).Elements
		}
		r, err := core.RangeFromComponents(m)
		if err != nil {
			return core.Undefined, err
		}
		return r, nil
	}
	if numArgs < 2 || numArgs > 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("range", "0, 2 or 3", numArgs)
	}

	start, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "start", "int", args[0].TypeName())
	}

	stop, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "stop", "int", args[1].TypeName())
	}

	step := int64(1)
	if numArgs == 3 {
		step, ok = args[2].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("range", "step", "int", args[2].TypeName())
		}
		if step <= 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, fmt.Sprintf("range step must be greater than 0, got %d", step))
		}
	}

	return core.NewIntRangeValue(start, stop, step), nil
}

func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	// one operation graded by arity: format(x) renders any value (a template
	// with nothing to fill is its own rendering) — the render's callable form,
	// and record's render spelling; format(tmpl, args) fills placeholders
	if len(args) == 1 {
		out, err := args[0].Format(fspec.FormatSpec{})
		if err != nil {
			return core.Undefined, err
		}
		return core.NewStringValue(out), nil
	}
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("format", "1 or 2", len(args))
	}
	if args[0].Type != value.String && args[0].Type != value.Runes && args[0].Type != value.Bytes {
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "template", "string", args[0].TypeName())
	}
	tmplStr, _ := args[0].AsString()

	var arr []core.Value
	var dict map[string]core.Value
	switch args[1].Type {
	case value.Array:
		arr = (*core.Array)(args[1].Ptr).Elements
	case value.Dict:
		dict = (*core.Dict)(args[1].Ptr).Elements
	case value.Record:
		dict = (*core.Record)(args[1].Ptr).Elements
	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array, dict, or record", args[1].TypeName())
	}

	tmpl, err := fspec.ParseTemplate(tmplStr)
	if err != nil {
		return core.Undefined, errs.FromFormatSpecError("format", err)
	}

	switch tmpl.Mode {
	case fspec.TemplateModeIndexed:
		if args[1].Type != value.Array {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array", args[1].TypeName())
		}
	case fspec.TemplateModeNamed:
		if args[1].Type == value.Array {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "dict or record", args[1].TypeName())
		}
	}

	lookup := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.Index < 0 || seg.Index >= len(arr) {
				return core.Undefined, errs.NewIndexOutOfBoundsError("format", seg.Index, len(arr))
			}
			return arr[seg.Index], nil
		}
		v, ok := dict[seg.Name]
		if !ok {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("format: missing key %q", seg.Name))
		}
		return v, nil
	}

	lookupRef := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.SpecRefIndex < 0 || seg.SpecRefIndex >= len(arr) {
				return core.Undefined, errs.NewIndexOutOfBoundsError("format spec ref", seg.SpecRefIndex, len(arr))
			}
			return arr[seg.SpecRefIndex], nil
		}
		v, ok := dict[seg.SpecRefName]
		if !ok {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("format: missing spec ref key %q", seg.SpecRefName))
		}
		return v, nil
	}

	var sb strings.Builder
	for _, seg := range tmpl.Segments {
		if seg.Kind == fspec.TemplateLiteral {
			sb.WriteString(seg.Literal)
			continue
		}
		val, err := lookup(seg)
		if err != nil {
			return core.Undefined, err
		}
		spec := seg.Spec
		if seg.HasSpec && seg.SpecIsRef {
			refVal, err := lookupRef(seg)
			if err != nil {
				return core.Undefined, err
			}
			if refVal.Type != value.String {
				return core.Undefined, errs.NewInvalidArgumentTypeError("format", "spec ref", "string", refVal.TypeName())
			}
			specStr, _ := refVal.AsString()
			parsed, ferr := fspec.Parse(specStr)
			if ferr != nil {
				return core.Undefined, errs.FromFormatSpecError("format", ferr)
			}
			spec = parsed
		}
		out, ferr := val.Format(spec)
		if ferr != nil {
			return core.Undefined, ferr
		}
		sb.WriteString(out)
	}
	return core.NewStringValue(sb.String()), nil
}

func builtinCopy(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	// absence has no identity to copy; handle missing data explicitly (is_undefined)
	if args[0].Type == value.Undefined {
		return core.Undefined, errs.NewInvalidArgumentTypeError("copy", "first", "a value", "undefined")
	}
	return args[0].Copy(true)
}

func builtinCopyShallow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("copy_shallow", "1", len(args))
	}
	// the twin exists only where the distinction is observable: the types whose
	// elements can themselves be containers
	switch args[0].Type {
	case value.Array, value.Dict, value.Record:
		return args[0].Copy(false)
	}
	if args[0].Type >= value.FirstUserDefinedType {
		return args[0].Copy(false)
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("copy_shallow", "first", "array, dict, or record", args[0].TypeName())
}

func builtinFreeze(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("freeze", "1", len(args))
	}
	// absence has no mutability; freezing it was a header-flag artifact
	if args[0].Type == value.Undefined {
		return core.Undefined, errs.NewInvalidArgumentTypeError("freeze", "first", "a value", "undefined")
	}
	return args[0].Freeze()
}

func builtinFreezeShallow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("freeze_shallow", "1", len(args))
	}
	switch args[0].Type {
	case value.Array, value.Dict, value.Record:
		return args[0].ToImmutable()
	}
	if args[0].Type >= value.FirstUserDefinedType {
		return args[0].ToImmutable()
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("freeze_shallow", "first", "array, dict, or record", args[0].TypeName())
}

func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	// .string() is a CONVERSION — the receiver's text content — not the render:
	// dict/record/callables have no text content and raise; format(x) renders.
	// string(x, n) is the count form: convert, then repeat the content n times.
	if len(args) == 2 {
		r, err := builtinString(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("string", args, core.EmptyString, func(t uint8) bool {
		switch t {
		case value.Bool, value.Byte, value.Rune, value.Int, value.Float, value.Decimal,
			value.String, value.Runes, value.Bytes, value.Array, value.IntRange, value.Time, value.Error:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.String:
			return src, true
		case value.Byte:
			s, ok := core.ByteSymbolString(byte(src.Data))
			return core.NewStringValue(s), ok
		case value.Bytes:
			// the UTF-8 decode is partial: invalid input declines
			b, _ := src.AsBytes()
			if !utf8.Valid(b) {
				return core.Undefined, false
			}
			return core.NewStringValue(string(b)), true
		case value.Array, value.IntRange:
			elems, _ := src.AsArray()
			rs, ok := core.ElementsToRunes(elems)
			return core.NewStringValue(string(rs)), ok
		case value.Error:
			o := (*core.Error)(src.Ptr)
			s, err := o.Payload.Format(fspec.FormatSpec{})
			return core.NewStringValue(s), err == nil
		}
		s, ok := src.AsString()
		return core.NewStringValue(s), ok
	})
}

func builtinRunes(vm core.VM, args []core.Value) (core.Value, error) {
	// mirrors .string() as symbols on every receiver that has text content —
	// note runes(65) is the CONVERSION runes("65"), never sizing; runes(x, n)
	// is the count form: convert, then repeat the content n times.
	if len(args) == 2 {
		r, err := builtinRunes(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("runes", args, core.NewRunesValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Bool, value.Byte, value.Rune, value.Int, value.Float, value.Decimal,
			value.String, value.Runes, value.Bytes, value.Array, value.IntRange, value.Time, value.Error:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Runes:
			return ctorSameType(src)
		case value.Byte:
			s, ok := core.ByteSymbolString(byte(src.Data))
			return core.NewRunesValue([]rune(s), false), ok
		case value.Bytes:
			b, _ := src.AsBytes()
			if !utf8.Valid(b) {
				return core.Undefined, false
			}
			return core.NewRunesValue([]rune(string(b)), false), true
		case value.Array, value.IntRange:
			elems, _ := src.AsArray()
			rs, ok := core.ElementsToRunes(elems)
			return core.NewRunesValue(rs, false), ok
		case value.Error:
			o := (*core.Error)(src.Ptr)
			s, err := o.Payload.Format(fspec.FormatSpec{})
			return core.NewRunesValue([]rune(s), false), err == nil
		}
		s, ok := src.AsString()
		return core.NewRunesValue([]rune(s), false), ok
	})
}

func builtinBytes(vm core.VM, args []core.Value) (core.Value, error) {
	// bytes(x) is the CONVERSION — x's content as octets; a byte is one octet, a rune its UTF-8
	// encoding. bytes(int) RAISES: bytes plays a double role (ASCII text vs memory chunk), so an
	// int argument is ambiguous — spell the octet explicitly (bytes(b'A')) or the text (bytes("65")).
	// bytes(x, n) is the count form: convert, then repeat the content n times — the old sizing
	// form's spelling is an explicit fill, bytes(b'\x00', 5).
	if len(args) == 2 {
		r, err := builtinBytes(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("bytes", args, core.NewBytesValue(nil, false), func(t uint8) bool {
		switch t {
		case value.String, value.Runes, value.Bytes, value.Array, value.Byte, value.Rune:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Bytes:
			return ctorSameType(src)
		case value.Array:
			elems, _ := src.AsArray()
			bs, ok := core.ElementsToBytes(elems)
			return core.NewBytesValue(bs, false), ok
		case value.Byte:
			return core.NewBytesValue([]byte{byte(src.Data)}, false), true
		case value.Rune:
			r := rune(src.Data)
			if !utf8.ValidRune(r) {
				return core.Undefined, false
			}
			return core.NewBytesValue([]byte(string(r)), false), true
		}
		b, ok := src.AsBytes()
		return core.NewBytesValue(b, false), ok
	})
}

func builtinArray(vm core.VM, args []core.Value) (core.Value, error) {
	// Two arities, two operations. array(x) is "represent x as a sequence": a convertible
	// (text, range, map entries) DECOMPOSES into its elements, anything else is one element —
	// array(5) is [5]; undefined raises (a conversion slot). array(x, n) is "n copies of x as
	// ONE element" — the never-spreading (push-family) reading, exactly [x].repeat(n):
	// array("ab", 2) is ["ab", "ab"], and undefined IS allowed there — array(undefined, 5)
	// is the explicit preallocation. The decomposing repetition is spelled convert-then-repeat:
	// array("ab").repeat(2).
	if len(args) == 2 {
		wrapped := core.NewArrayValue([]core.Value{args[0]}, false)
		return ctorRepeat(vm, wrapped, args[1])
	}
	return convertBuiltin("array", args, core.NewArrayValue(nil, false), func(t uint8) bool {
		// every present value has an array reading: decompose or wrap
		return true
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Array:
			return ctorSameType(src)
		case value.Dict:
			// a map's conversion elements are its entries, key-sorted
			return core.NewArrayValue(core.MapToSortedEntries((*core.Dict)(src.Ptr).Elements), false), true
		case value.Record:
			return core.NewArrayValue(core.MapToSortedEntries((*core.Record)(src.Ptr).Elements), false), true
		case value.String, value.Runes, value.Bytes, value.IntRange:
			elems, ok := src.AsArray()
			return core.NewArrayValue(elems, false), ok
		}
		if src.Type >= value.FirstUserDefinedType {
			if elems, ok := src.AsArray(); ok {
				return core.NewArrayValue(elems, false), true
			}
		}
		// a non-sequence value is one element
		return core.NewArrayValue([]core.Value{src}, false), true
	})
}

func builtinDict(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("dict", args, core.NewDictValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Dict, value.Record, value.Array:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Dict:
			return ctorSameType(src)
		case value.Record:
			return core.RecordToDict(src, false), true
		case value.Array:
			// the entries reading, agreeing with arr.dict()
			m, ok := core.ElementsToEntries((*core.Array)(src.Ptr).Elements)
			return core.NewDictValue(m, false), ok
		}
		return core.Undefined, false
	})
}

func builtinRecord(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("record", args, core.NewRecordValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Dict, value.Record, value.Array:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Record:
			return ctorSameType(src)
		case value.Dict:
			return core.DictToRecord(src, false), true
		case value.Array:
			m, ok := core.ElementsToEntries((*core.Array)(src.Ptr).Elements)
			return core.NewRecordValue(m, false), ok
		}
		return core.Undefined, false
	})
}

func builtinTime(vm core.VM, args []core.Value) (core.Value, error) {
	// a components MAP is a conversion and rebuilds the instant; unknown keys
	// raise inside TimeFromComponents, so a typo never silently means year 1
	if len(args) >= 1 && (args[0].Type == value.Dict || args[0].Type == value.Record) {
		if len(args) > 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("time", "0 or 1", len(args))
		}
		var m map[string]core.Value
		if args[0].Type == value.Dict {
			m = (*core.Dict)(args[0].Ptr).Elements
		} else {
			m = (*core.Record)(args[0].Ptr).Elements
		}
		t, err := core.TimeFromComponents(m)
		if err != nil {
			return core.Undefined, err
		}
		return core.NewTimeValue(t), nil
	}
	return convertBuiltin("time", args, core.NewTimeValue(time.Time{}), func(t uint8) bool {
		switch t {
		case value.Time, value.String, value.Runes, value.Int, value.Float, value.Decimal:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Time {
			return src, true
		}
		t, ok := src.AsTime()
		return core.NewTimeValue(t), ok
	})
}

// convertBuiltin implements the uniform free-constructor shape T([x]).
// T() is the zero value; T(x) builds the value from x's CONTENT — a safe, unambiguous
// conversion — and raises on failure: there is no free default form. The fallible-conversion
// idiom is the member's, where the receiver opts into recovery: x.T(default).
//
// Host-defined types always get an attempt: their hooks decide.
func convertBuiltin(name string, args []core.Value, zero core.Value, hasEdge func(uint8) bool, conv func(core.Value) (core.Value, bool)) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return zero, nil
	}
	if l > 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", l)
	}
	src := args[0]
	if src.Type == value.Undefined {
		return core.Undefined, errs.NewConversionError(src.TypeName(), name, "value is missing")
	}
	if !hasEdge(src.Type) && src.Type < value.FirstUserDefinedType {
		return core.Undefined, errs.NewConversionError(src.TypeName(), name, "no conversion exists")
	}
	if r, ok := conv(src); ok {
		return r, nil
	}
	return core.Undefined, errs.NewConversionError(src.TypeName(), name, "")
}

// ctorSameType answers a constructor's same-type argument. A constructor CONSTRUCTS: `T(x)` where `x` is
// already a `T` builds a NEW, independent, mutable value rather than handing back the argument, so
// `b := array(a)` never writes through to `a` and `bytes(b"ab")` gives a writable body from a literal's
// shared constant. The copy is SHALLOW — exactly `x.copy_shallow()`: the elements are the values handed in,
// and a frozen element stays frozen. The deep spelling is `x.copy()`. Immutable types (string, the scalars)
// have no observable form of this: they cannot be written to and carry no identity, so their converters
// return the argument.
// PURE by contract: a fresh value that no one else holds is not external state.
func ctorSameType(src core.Value) (core.Value, bool) {
	c, err := src.Copy(false)
	if err != nil {
		return core.Undefined, false
	}
	return c, true
}

// ctorRepeat implements the sequence constructors' count form: T(x, n) is n copies of x-as-element.
// For the text types the element reading degenerates to content repetition (their elements are scalars
// only), so it is exactly T(x).repeat(n); array never spreads, so it is exactly [x].repeat(n). Both are
// implemented AS that member call, so validation (lossless int count, non-negative) and semantics can
// never drift from repeat's.
func ctorRepeat(vm core.VM, seq core.Value, count core.Value) (core.Value, error) {
	return seq.MethodCall(vm, "repeat", []core.Value{count})
}

// the numeric targets share one source set: the numerics themselves, text, and time
func numericConvEdge(t uint8) bool {
	switch t {
	case value.Int, value.Float, value.Decimal, value.String, value.Runes, value.Time:
		return true
	}
	return false
}

func builtinInt(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("int", args, core.IntValue(0), func(t uint8) bool {
		return numericConvEdge(t) || t == value.Bool || t == value.Byte || t == value.Rune
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Int {
			return src, true
		}
		i, ok := src.AsInt()
		return core.IntValue(i), ok
	})
}

func builtinFloat(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("float", args, core.FloatValue(0), numericConvEdge, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Float {
			return src, true
		}
		f, ok := src.AsFloat()
		return core.FloatValue(f), ok
	})
}

func builtinDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("decimal", args, core.NewDecimalValue(dec128.Decimal0), numericConvEdge, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Decimal {
			return src, true
		}
		d, ok := src.AsDecimal()
		if ok && d.IsNaN() {
			// a NaN decimal is an error state, never a produced value
			return core.Undefined, false
		}
		return core.NewDecimalValue(d), ok
	})
}

func builtinBool(vm core.VM, args []core.Value) (core.Value, error) {
	// the conversion, not truthiness: a text receiver parses a boolean literal,
	// a numeric receiver is a zero check; everything else has no edge and raises.
	// Truthiness is is_true(x) / !!x.
	return convertBuiltin("bool", args, core.False, func(t uint8) bool {
		switch t {
		case value.Bool, value.Int, value.Float, value.Decimal, value.String, value.Runes:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Bool {
			return src, true
		}
		b, ok := src.AsBool()
		return core.BoolValue(b), ok
	})
}

func builtinByte(vm core.VM, args []core.Value) (core.Value, error) {
	// int is the sole gateway from the numeric domain; a rune converts iff its
	// UTF-8 form is one octet (ASCII); text does not parse into an ordinal type
	return convertBuiltin("byte", args, core.ByteValue(0), func(t uint8) bool {
		return t == value.Byte || t == value.Int || t == value.Rune
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Byte {
			return src, true
		}
		b, ok := src.AsByte()
		return core.ByteValue(b), ok
	})
}

func builtinRune(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("rune", args, core.RuneValue(0), func(t uint8) bool {
		return t == value.Rune || t == value.Int || t == value.Byte
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Rune {
			return src, true
		}
		c, ok := src.AsRune()
		return core.RuneValue(c), ok
	})
}

// dict(x): 0 args -> empty dict. dict already a dict -> unchanged. dict(record) -> independent shallow copy
// (P19: no longer shares storage with the source record). dict_view(record) is the explicit sharing opt-in.

// dict_view(x): the `_view` twin of dict() — dict_view(record) shares backing storage with the source record
// instead of copying (today's original dict(record) behavior, preserved under this new name; see P19).
func builtinDictView(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewDictValue(nil, false), nil
	}
	if l > 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("dict_view", "0 or 1", len(args))
	}

	switch args[0].Type {
	case value.Dict:
		return args[0], nil

	case value.Record:
		return core.RecordToDict(args[0], true), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("dict_view", "first", "dict or record", args[0].TypeName())
	}
}

// record(x): 0 args -> empty record. record already a record -> unchanged. record(dict) -> independent shallow
// copy, the same operation as dict_val.record(). Kept as a free function like dict() — record has no
// MethodCall switch (see P14), so this is also record's only reachable constructor-style spelling.

// record_view(x): the `_view` twin of record() — record_view(dict) shares backing storage with the source dict
// instead of copying, the same operation as dict_val.record_view().
func builtinRecordView(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewRecordValue(nil, false), nil
	}
	if l > 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("record_view", "0 or 1", len(args))
	}

	switch args[0].Type {
	case value.Record:
		return args[0], nil

	case value.Dict:
		return core.DictToRecord(args[0], true), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("record_view", "first", "dict or record", args[0].TypeName())
	}
}

// builtinDelete returns a dict/record without the given key, without mutating the receiver.
// usage: out := delete(map, "key")
// key must be a string
func builtinRemove(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("remove", "2", argsLen)
	}
	return args[0].Delete(args[1], false)
}

// builtinDeleteInPlace deletes a dict/record key in place, mutating the receiver.
// usage: delete_in_place(map, "key")
// key must be a string
func builtinRemoveInPlace(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("remove_in_place", "2", argsLen)
	}
	return args[0].Delete(args[1], true)
}
