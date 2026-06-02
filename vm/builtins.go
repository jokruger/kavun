package vm

import (
	"fmt"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// do not change builtin function indexes as it will break compatibility
// 42.. are reserved for future builtin functions
var BuiltinFuncs = map[int]core.Value{
	7: core.NewBuiltinFunctionValueAt(7, "bool", builtinBool, 0, true),
	38: core.NewBuiltinFunctionValueAt(38, "byte", builtinByte, 0, true),
	9: core.NewBuiltinFunctionValueAt(9, "rune", builtinRune, 0, true),
	6: core.NewBuiltinFunctionValueAt(6, "int", builtinInt, 0, true),
	8: core.NewBuiltinFunctionValueAt(8, "float", builtinFloat, 0, true),
	34: core.NewBuiltinFunctionValueAt(34, "decimal", builtinDecimal, 0, true),
	11: core.NewBuiltinFunctionValueAt(11, "time", builtinTime, 0, true),
	5: core.NewBuiltinFunctionValueAt(5, "string", builtinString, 0, true),
	36: core.NewBuiltinFunctionValueAt(36, "runes", builtinRunes, 0, true),
	10: core.NewBuiltinFunctionValueAt(10, "bytes", builtinBytes, 0, true),
	21: core.NewBuiltinFunctionValueAt(21, "dict", builtinDict, 0, true),
	30: core.NewBuiltinFunctionValueAt(30, "range", builtinRange, 2, true),
	33: core.NewBuiltinFunctionValueAt(33, "error", builtinError, 0, true),

	15: core.NewBuiltinFunctionValueAt(15, "is_bool", builtinIsBool, 1, false),
	39: core.NewBuiltinFunctionValueAt(39, "is_byte", builtinIsByte, 1, false),
	16: core.NewBuiltinFunctionValueAt(16, "is_rune", builtinIsRune, 1, false),
	12: core.NewBuiltinFunctionValueAt(12, "is_int", builtinIsInt, 1, false),
	13: core.NewBuiltinFunctionValueAt(13, "is_float", builtinIsFloat, 1, false),
	35: core.NewBuiltinFunctionValueAt(35, "is_decimal", builtinIsDecimal, 1, false),
	23: core.NewBuiltinFunctionValueAt(23, "is_time", builtinIsTime, 1, false),
	14: core.NewBuiltinFunctionValueAt(14, "is_string", builtinIsString, 1, false),
	37: core.NewBuiltinFunctionValueAt(37, "is_runes", builtinIsRunes, 1, false),
	17: core.NewBuiltinFunctionValueAt(17, "is_bytes", builtinIsBytes, 1, false),
	18: core.NewBuiltinFunctionValueAt(18, "is_array", builtinIsArray, 1, false),
	31: core.NewBuiltinFunctionValueAt(31, "is_dict", builtinIsDict, 1, false),
	20: core.NewBuiltinFunctionValueAt(20, "is_record", builtinIsRecord, 1, false),
	32: core.NewBuiltinFunctionValueAt(32, "is_range", builtinIsRange, 1, false),
	24: core.NewBuiltinFunctionValueAt(24, "is_error", builtinIsError, 1, false),

	25: core.NewBuiltinFunctionValueAt(25, "is_undefined", builtinIsUndefined, 1, false),
	26: core.NewBuiltinFunctionValueAt(26, "is_function", builtinIsFunction, 1, false),
	27: core.NewBuiltinFunctionValueAt(27, "is_callable", builtinIsCallable, 1, false),
	22: core.NewBuiltinFunctionValueAt(22, "is_iterable", builtinIsIterable, 1, false),
	19: core.NewBuiltinFunctionValueAt(19, "is_immutable", builtinIsImmutable, 1, false),

	0: core.NewBuiltinFunctionValueAt(0, "len", builtinLen, 1, false),
	1: core.NewBuiltinFunctionValueAt(1, "copy", builtinCopy, 1, false),
	2: core.NewBuiltinFunctionValueAt(2, "append", builtinAppend, 2, true),
	3: core.NewBuiltinFunctionValueAt(3, "delete", builtinDelete, 2, false),
	4: core.NewBuiltinFunctionValueAt(4, "splice", builtinSplice, 1, true),
	29: core.NewBuiltinFunctionValueAt(29, "format", builtinFormat, 2, false),
	28: core.NewBuiltinFunctionValueAt(28, "type_name", builtinTypeName, 1, false),
	40: core.NewBuiltinFunctionValueAt(40, "raise", builtinRaise, 1, true),
	41: core.NewBuiltinFunctionValueAt(41, "recover", builtinRecover, 0, false),
}

func builtinTypeName(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return a.NewStringValue(args[0].TypeName(a)), nil
}

func builtinIsString(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	if args[0].Type == core.VT_STRING {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRunes(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_runes", "1", len(args))
	}
	if args[0].Type == core.VT_RUNES {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsInt(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	if args[0].Type == core.VT_INT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFloat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	if args[0].Type == core.VT_FLOAT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDecimal(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_decimal", "1", len(args))
	}
	if args[0].Type == core.VT_DECIMAL {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBool(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	if args[0].Type == core.VT_BOOL {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsByte(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_byte", "1", len(args))
	}
	if args[0].Type == core.VT_BYTE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRune(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_rune", "1", len(args))
	}
	if args[0].Type == core.VT_RUNE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBytes(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	if args[0].Type == core.VT_BYTES {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsArray(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	if args[0].Type == core.VT_ARRAY {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRecord(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	if args[0].Type == core.VT_RECORD {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDict(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_dict", "1", len(args))
	}
	if args[0].Type == core.VT_DICT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRange(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_range", "1", len(args))
	}
	if args[0].Type == core.VT_INT_RANGE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsImmutable(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return core.BoolValue(args[0].Immutable), nil
}

func builtinIsTime(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	if args[0].Type == core.VT_TIME {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsError(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	if args[0].Type == core.VT_ERROR {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsUndefined(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	if args[0].Type == core.VT_UNDEFINED {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFunction(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_function", "1", len(args))
	}

	switch args[0].Type {
	case core.VT_BUILTIN_FUNCTION, core.VT_BUILTIN_CLOSURE, core.VT_COMPILED_FUNCTION:
		return core.True, nil
	default:
		return core.False, nil
	}
}

func builtinIsCallable(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return core.BoolValue(args[0].IsCallable(a)), nil
}

func builtinIsIterable(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return core.BoolValue(args[0].IsIterable(a)), nil
}

// len(obj object) => int
func builtinLen(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("len", "1", len(args))
	}
	return core.IntValue(args[0].Len(a)), nil
}

// error(val) creates a (recoverable) Kavun error value with the given payload.
// error(val, fatal) — if fatal is true, the resulting error, when raised, bypasses recover() and stops the VM,
// propagating to the host caller.
func builtinError(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 1:
		return core.NewErrorValue(args[0]), nil
	case 2:
		fatal, ok := args[1].AsBool(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("error", "second", "bool", args[1].TypeName(a))
		}
		if fatal {
			return core.NewFatalErrorValue(args[0]), nil
		}
		return core.NewErrorValue(args[0]), nil
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("error", "1 or 2", len(args))
	}
}

// raise(err) raises the given error so that surrounding deferred recover() calls can catch it. If `err` is not already
// an error value, it is wrapped in a fresh recoverable error.
// raise(err, fatal) — explicitly sets the severity of the raised error: a fatal error bypasses recover() and stops the
// VM. If `err` is an existing error value, a copy with the requested severity is raised (the original is left
// untouched).
func builtinRaise(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	var val core.Value
	switch len(args) {
	case 1:
		val = args[0]
		if val.Type != core.VT_ERROR {
			val = core.NewErrorValue(val)
		}
	case 2:
		fatal, ok := args[1].AsBool(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("raise", "second", "bool", args[1].TypeName(a))
		}
		if args[0].Type == core.VT_ERROR {
			o := (*core.Error)(args[0].Ptr)
			val = core.ErrorValue(&core.Error{Payload: o.Payload, Kind: o.Kind, Fatal: fatal})
		} else if fatal {
			val = core.NewFatalErrorValue(args[0])
		} else {
			val = core.NewErrorValue(args[0])
		}
	default:
		return core.Undefined, errs.NewWrongNumArgumentsError("raise", "1 or 2", len(args))
	}
	return core.Undefined, newRaisedError(a, val)
}

// recover() returns the in-flight Kavun error caught by a deferred function and clears it (so the surrounding function
// returns normally). Outside a deferred function, or when there is no error in flight, it returns undefined.
// Must be called directly inside a deferred function — any indirection returns undefined.
func builtinRecover(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("recover", "0", len(args))
	}
	return vm.Recover(), nil
}

// range(start, stop[, step])
func builtinRange(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
	}

	start, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "start", "int", args[0].TypeName(a))
	}

	stop, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "stop", "int", args[1].TypeName(a))
	}

	step := int64(1)
	if numArgs == 3 {
		step, ok = args[2].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("range", "step", "int", args[2].TypeName(a))
		}
		if step <= 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, fmt.Sprintf("range step must be greater than 0, got %d", step))
		}
	}

	return a.NewIntRangeValue(start, stop, step), nil
}

func builtinFormat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("format", "2", len(args))
	}
	if args[0].Type != core.VT_STRING && args[0].Type != core.VT_RUNES && args[0].Type != core.VT_BYTES {
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "template", "string", args[0].TypeName(a))
	}
	tmplStr, _ := args[0].AsString(a)

	var arr []core.Value
	var dict map[string]core.Value
	switch args[1].Type {
	case core.VT_ARRAY:
		arr = (*core.Array)(args[1].Ptr).Elements
	case core.VT_DICT, core.VT_RECORD:
		dict = (*core.Dict)(args[1].Ptr).Elements
	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array, dict, or record", args[1].TypeName(a))
	}

	tmpl, err := fspec.ParseTemplate(tmplStr)
	if err != nil {
		return core.Undefined, errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, err.Error())
	}

	switch tmpl.Mode {
	case fspec.TemplateModeIndexed:
		if args[1].Type != core.VT_ARRAY {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array", args[1].TypeName(a))
		}
	case fspec.TemplateModeNamed:
		if args[1].Type == core.VT_ARRAY {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "dict or record", args[1].TypeName(a))
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
			if refVal.Type != core.VT_STRING {
				return core.Undefined, errs.NewInvalidArgumentTypeError("format", "spec ref", "string", refVal.TypeName(a))
			}
			specStr, _ := refVal.AsString(a)
			parsed, ferr := fspec.Parse(specStr)
			if ferr != nil {
				return core.Undefined, errs.NewRecoverableError(errs.KindUnsupportedFormatSpec, fmt.Sprintf("format: %v", ferr))
			}
			spec = parsed
		}
		out, ferr := val.Format(a, spec)
		if ferr != nil {
			return core.Undefined, ferr
		}
		sb.WriteString(out)
	}
	return a.NewStringValue(sb.String()), nil
}

func builtinCopy(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Clone(a)
}

func builtinString(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return a.NewStringValue(""), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_STRING:
		return args[0], nil

	default:
		if v, ok := args[0].AsString(a); ok {
			return a.NewStringValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinRunes(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	alloc := a

	if l == 0 {
		rs := alloc.NewRunes(0, false)
		return alloc.NewRunesValue(rs, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("runes", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_RUNES:
		return args[0], nil

	case core.VT_INT:
		n := int(int64(args[0].Data))
		bs := alloc.NewRunes(n, true)
		return alloc.NewRunesValue(bs, false), nil

	default:
		if v, ok := args[0].AsRunes(a); ok {
			return alloc.NewRunesValue(v, false), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinInt(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.IntValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_INT:
		return args[0], nil

	default:
		if v, ok := args[0].AsInt(a); ok {
			return core.IntValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinFloat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.FloatValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_FLOAT:
		return args[0], nil

	default:
		if v, ok := args[0].AsFloat(a); ok {
			return core.FloatValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinDecimal(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("decimal", "0, 1 or 2", len(args))
	}

	if l == 0 {
		d := a.NewDecimal()
		*d = dec128.Decimal0
		return core.DecimalValue(d), nil
	}

	switch args[0].Type {
	case core.VT_DECIMAL:
		return args[0], nil

	default:
		v, ok := args[0].AsDecimal(a)
		if !ok && l == 2 {
			return args[1], nil
		}
		d := a.NewDecimal()
		*d = v
		return core.DecimalValue(d), nil
	}
}

func builtinBool(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.False, nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("bool", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_BOOL:
		return args[0], nil

	default:
		if v, ok := args[0].AsBool(a); ok {
			return core.BoolValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinByte(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.ByteValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("byte", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_BYTE:
		return args[0], nil

	default:
		if v, ok := args[0].AsByte(a); ok {
			return core.ByteValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinRune(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.RuneValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("rune", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_RUNE:
		return args[0], nil

	default:
		if v, ok := args[0].AsRune(a); ok {
			return core.RuneValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinBytes(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	alloc := a

	if l == 0 {
		bs := alloc.NewBytes(0, false)
		return alloc.NewBytesValue(bs, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_BYTES:
		return args[0], nil

	case core.VT_INT:
		n := int(int64(args[0].Data))
		bs := alloc.NewBytes(n, true)
		return alloc.NewBytesValue(bs, false), nil

	default:
		if v, ok := args[0].AsBytes(a); ok {
			return alloc.NewBytesValue(v, false), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinTime(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	if l == 0 {
		d := a.NewTime()
		*d = time.Time{}
		return core.TimeValue(d), nil
	}

	switch args[0].Type {
	case core.VT_TIME:
		return args[0], nil

	default:
		if v, ok := args[0].AsTime(a); ok {
			d := a.NewTime()
			*d = v
			return core.TimeValue(d), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinDict(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return a.NewDictValue(nil, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("dict", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_DICT:
		return args[0], nil

	case core.VT_RECORD:
		r := (*core.Dict)(args[0].Ptr)
		return a.NewDictValue(r.Elements, args[0].Immutable), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("dict", "first", "dict or record", args[0].TypeName(a))
	}
}

// append(arr, items...)
func builtinAppend(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) < 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}
	return args[0].Append(a, args[1:])
}

// builtinDelete deletes Map keys inplace
// usage: delete(map, "key")
// key must be a string
func builtinDelete(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("delete", "2", argsLen)
	}
	return args[0].Delete(a, args[1])
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage: deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}
	if args[0].Type != core.VT_ARRAY {
		return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName(a))
	}
	if args[0].Immutable {
		return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName(a))
	}

	arr := (*core.Array)(args[0].Ptr)
	arrayLen := len(arr.Elements)

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName(a))
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return core.Undefined, errs.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName(a))
		}
		if arg2 < 0 {
			return core.Undefined, errs.NewRecoverableError(errs.KindInvalidValue, "splice delete count must be non-negative")
		}
		// Clamp before converting to avoid signed integer overflow when computing startIdx+delCount.
		if arg2 > int64(arrayLen-startIdx) {
			delCount = arrayLen - startIdx
		} else {
			delCount = int(arg2)
		}
	} else if startIdx+delCount > arrayLen {
		// no count given; default to "from startIdx to end"
		delCount = arrayLen - startIdx
	}
	// delete items
	endIdx := startIdx + delCount
	deleted := append([]core.Value{}, arr.Elements[startIdx:endIdx]...)

	alloc := a
	head := arr.Elements[:startIdx]
	var items []core.Value
	if argsLen > 3 {
		items = alloc.NewArray(argsLen-3, false)
		for i := 3; i < argsLen; i++ {
			items = append(items, args[i])
		}
	}
	items = append(items, arr.Elements[endIdx:]...)
	arr.Set(append(head, items...))

	// return deleted items
	return alloc.NewArrayValue(deleted, false), nil
}
