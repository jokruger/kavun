package vm

import (
	"fmt"
	"strings"
	"time"

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
		3:  core.NewBuiltinFunction("delete", builtinDelete, 2, false, true), // pure: returns a container without the key
		4:  core.NewBuiltinFunction("delete_in_place", builtinDeleteInPlace, 2, false, false),
		29: core.NewBuiltinFunction("format", builtinFormat, 2, false, true),
		28: core.NewBuiltinFunction("type_name", builtinTypeName, 1, false, true),
		40: core.NewBuiltinFunction("raise", builtinRaise, 1, true, false),
		41: core.NewBuiltinFunction("recover", builtinRecover, 0, false, false),
		43: core.NewBuiltinFunction("min", builtinMin, 0, true, true),
		44: core.NewBuiltinFunction("max", builtinMax, 0, true, true),
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
	return core.BoolValue(args[0].Immutable), nil
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

// len(obj object) => int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("len", "1", len(args))
	}
	return core.IntValue(args[0].Len()), nil
}

// min(args...) => smallest argument, by BinaryOp(Less); 0 args => undefined, 1 arg => that arg unchanged.
func builtinMin(vm core.VM, args []core.Value) (core.Value, error) {
	return minMaxReduce(args, token.Less)
}

// max(args...) => largest argument, by BinaryOp(Greater); 0 args => undefined, 1 arg => that arg unchanged.
func builtinMax(vm core.VM, args []core.Value) (core.Value, error) {
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
		if better.IsTrue() {
			best = args[i]
		}
	}

	return best, nil
}

// error(val) creates a (recoverable) Kavun error value with the given payload.
// error(val, fatal) — if fatal is true, the resulting error, when raised, bypasses recover() and stops the VM,
// propagating to the host caller.
func builtinError(vm core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 1:
		return core.NewErrorValue(args[0], core.KindUser, false), nil
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("error", "second", "bool", args[1].TypeName())
		}
		if fatal {
			return core.NewErrorValue(args[0], core.KindUser, true), nil
		}
		return core.NewErrorValue(args[0], core.KindUser, false), nil
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
			val = core.NewErrorValue(val, core.KindUser, false)
		}
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("raise", "second", "bool", args[1].TypeName())
		}
		if args[0].Type == value.Error {
			o := (*core.Error)(args[0].Ptr)
			val = core.NewErrorValue(o.Payload, o.Kind, fatal)
		} else if fatal {
			val = core.NewErrorValue(args[0], core.KindUser, true)
		} else {
			val = core.NewErrorValue(args[0], core.KindUser, false)
		}
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("raise", "1 or 2", len(args))
	}
	return core.Undefined, newRaisedError(val)
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

// range(start, stop[, step])
func builtinRange(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
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
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("format", "2", len(args))
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
		return core.Undefined, errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, err.Error())
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
				return core.Undefined, errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, fmt.Sprintf("format: %v", ferr))
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
	return args[0].Copy(true)
}

func builtinCopyShallow(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("copy_shallow", "1", len(args))
	}
	return args[0].Copy(false)
}

func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.EmptyString, nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.String:
		return args[0], nil

	default:
		if v, ok := args[0].AsString(); ok {
			return core.NewStringValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinRunes(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)

	if l == 0 {
		return core.NewRunesValue(make([]rune, 0), false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("runes", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Runes:
		return args[0], nil

	case value.Int:
		n := int64(args[0].Data)
		if n < 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, fmt.Sprintf("runes size must be non-negative, got %d", n))
		}
		return core.NewRunesValue(make([]rune, n), false), nil

	default:
		if v, ok := args[0].AsRunes(); ok {
			return core.NewRunesValue(v, false), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinInt(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.IntValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Int:
		return args[0], nil

	default:
		if v, ok := args[0].AsInt(); ok {
			return core.IntValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinFloat(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.FloatValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Float:
		return args[0], nil

	default:
		if v, ok := args[0].AsFloat(); ok {
			return core.FloatValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("decimal", "0, 1 or 2", len(args))
	}

	if l == 0 {
		return core.NewDecimalValue(dec128.Decimal0), nil
	}

	switch args[0].Type {
	case value.Decimal:
		return args[0], nil

	default:
		v, ok := args[0].AsDecimal()
		if !ok && l == 2 {
			return args[1], nil
		}
		return core.NewDecimalValue(v), nil
	}
}

func builtinBool(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.False, nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("bool", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Bool:
		return args[0], nil

	default:
		if v, ok := args[0].AsBool(); ok {
			return core.BoolValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinByte(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.ByteValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("byte", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Byte:
		return args[0], nil

	default:
		if v, ok := args[0].AsByte(); ok {
			return core.ByteValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinRune(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.RuneValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rune", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Rune:
		return args[0], nil

	default:
		if v, ok := args[0].AsRune(); ok {
			return core.RuneValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinBytes(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)

	if l == 0 {
		return core.NewBytesValue(make([]byte, 0), false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Bytes:
		return args[0], nil

	case value.Int:
		n := int64(args[0].Data)
		if n < 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, fmt.Sprintf("bytes size must be non-negative, got %d", n))
		}
		return core.NewBytesValue(make([]byte, n), false), nil

	default:
		if v, ok := args[0].AsBytes(); ok {
			return core.NewBytesValue(v, false), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinArray(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewArrayValue(nil, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("array", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Array:
		return args[0], nil

	case value.Int:
		n := int64(args[0].Data)
		if n < 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, fmt.Sprintf("array size must be non-negative, got %d", n))
		}
		return core.NewArrayValue(make([]core.Value, n), false), nil

	default:
		if v, ok := args[0].AsArray(); ok {
			return core.NewArrayValue(v, false), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinTime(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	if l == 0 {
		return core.NewTimeValue(time.Time{}), nil
	}

	switch args[0].Type {
	case value.Time:
		return args[0], nil

	default:
		if v, ok := args[0].AsTime(); ok {
			return core.NewTimeValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinDict(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewDictValue(nil, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("dict", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case value.Dict:
		return args[0], nil

	case value.Record:
		r := (*core.Record)(args[0].Ptr)
		return core.NewDictValue(r.Elements, args[0].Immutable), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("dict", "first", "dict or record", args[0].TypeName())
	}
}

// builtinDelete returns a dict/record without the given key, without mutating the receiver.
// usage: out := delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("delete", "2", argsLen)
	}
	return args[0].Delete(args[1], false)
}

// builtinDeleteInPlace deletes a dict/record key in place, mutating the receiver.
// usage: delete_in_place(map, "key")
// key must be a string
func builtinDeleteInPlace(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("delete_in_place", "2", argsLen)
	}
	return args[0].Delete(args[1], true)
}
