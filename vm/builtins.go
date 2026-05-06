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
// 40..99 are reserved for future builtin functions
var BuiltinFuncs = map[int]core.Value{
	7:  core.NewBuiltinFunctionValue("bool", builtinBool, 0, true),
	38: core.NewBuiltinFunctionValue("byte", builtinByte, 0, true),
	9:  core.NewBuiltinFunctionValue("rune", builtinRune, 0, true),
	6:  core.NewBuiltinFunctionValue("int", builtinInt, 0, true),
	8:  core.NewBuiltinFunctionValue("float", builtinFloat, 0, true),
	34: core.NewBuiltinFunctionValue("decimal", builtinDecimal, 0, true),
	11: core.NewBuiltinFunctionValue("time", builtinTime, 0, true),
	5:  core.NewBuiltinFunctionValue("string", builtinString, 0, true),
	36: core.NewBuiltinFunctionValue("runes", builtinRunes, 0, true),
	10: core.NewBuiltinFunctionValue("bytes", builtinBytes, 0, true),
	21: core.NewBuiltinFunctionValue("dict", builtinDict, 0, true),
	30: core.NewBuiltinFunctionValue("range", builtinRange, 2, true),
	33: core.NewBuiltinFunctionValue("error", builtinError, 0, true),

	15: core.NewBuiltinFunctionValue("is_bool", builtinIsBool, 1, false),
	39: core.NewBuiltinFunctionValue("is_byte", builtinIsByte, 1, false),
	16: core.NewBuiltinFunctionValue("is_rune", builtinIsRune, 1, false),
	12: core.NewBuiltinFunctionValue("is_int", builtinIsInt, 1, false),
	13: core.NewBuiltinFunctionValue("is_float", builtinIsFloat, 1, false),
	35: core.NewBuiltinFunctionValue("is_decimal", builtinIsDecimal, 1, false),
	23: core.NewBuiltinFunctionValue("is_time", builtinIsTime, 1, false),
	14: core.NewBuiltinFunctionValue("is_string", builtinIsString, 1, false),
	37: core.NewBuiltinFunctionValue("is_runes", builtinIsRunes, 1, false),
	17: core.NewBuiltinFunctionValue("is_bytes", builtinIsBytes, 1, false),
	18: core.NewBuiltinFunctionValue("is_array", builtinIsArray, 1, false),
	31: core.NewBuiltinFunctionValue("is_dict", builtinIsDict, 1, false),
	20: core.NewBuiltinFunctionValue("is_record", builtinIsRecord, 1, false),
	32: core.NewBuiltinFunctionValue("is_range", builtinIsRange, 1, false),
	24: core.NewBuiltinFunctionValue("is_error", builtinIsError, 1, false),

	25: core.NewBuiltinFunctionValue("is_undefined", builtinIsUndefined, 1, false),
	26: core.NewBuiltinFunctionValue("is_function", builtinIsFunction, 1, false),
	27: core.NewBuiltinFunctionValue("is_callable", builtinIsCallable, 1, false),
	22: core.NewBuiltinFunctionValue("is_iterable", builtinIsIterable, 1, false),
	19: core.NewBuiltinFunctionValue("is_immutable", builtinIsImmutable, 1, false),

	0:  core.NewBuiltinFunctionValue("len", builtinLen, 1, false),
	1:  core.NewBuiltinFunctionValue("copy", builtinCopy, 1, false),
	2:  core.NewBuiltinFunctionValue("append", builtinAppend, 2, true),
	3:  core.NewBuiltinFunctionValue("delete", builtinDelete, 2, false),
	4:  core.NewBuiltinFunctionValue("splice", builtinSplice, 1, true),
	29: core.NewBuiltinFunctionValue("format", builtinFormat, 2, false),
	28: core.NewBuiltinFunctionValue("type_name", builtinTypeName, 1, false),
}

func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return vm.Allocator().NewStringValue(args[0].TypeName()), nil
}

func builtinIsString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	if args[0].Type == core.VT_STRING {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRunes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_runes", "1", len(args))
	}
	if args[0].Type == core.VT_RUNES {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	if args[0].Type == core.VT_INT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	if args[0].Type == core.VT_FLOAT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_decimal", "1", len(args))
	}
	if args[0].Type == core.VT_DECIMAL {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	if args[0].Type == core.VT_BOOL {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsByte(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_byte", "1", len(args))
	}
	if args[0].Type == core.VT_BYTE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRune(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_rune", "1", len(args))
	}
	if args[0].Type == core.VT_RUNE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	if args[0].Type == core.VT_BYTES {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsArray(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	if args[0].Type == core.VT_ARRAY {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRecord(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	if args[0].Type == core.VT_RECORD {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsDict(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_dict", "1", len(args))
	}
	if args[0].Type == core.VT_DICT {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsRange(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_range", "1", len(args))
	}
	if args[0].Type == core.VT_INT_RANGE {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsImmutable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return core.BoolValue(args[0].IsImmutable()), nil
}

func builtinIsTime(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	if args[0].Type == core.VT_TIME {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsError(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	if args[0].Type == core.VT_ERROR {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsUndefined(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	if args[0].Type == core.VT_UNDEFINED {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsFunction(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_function", "1", len(args))
	}

	switch args[0].Type {
	case core.VT_BUILTIN_FUNCTION, core.VT_COMPILED_FUNCTION:
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

// error([payload]) => error
func builtinError(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) > 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("error", "0 or 1", len(args))
	}
	var payload core.Value
	if len(args) == 1 {
		payload = args[0]
	}
	return vm.Allocator().NewErrorValue(payload), nil
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
			return core.Undefined, errs.NewLogicError(fmt.Sprintf("range step must be greater than 0, got %d", step))
		}
	}

	return vm.Allocator().NewIntRangeValue(start, stop, step), nil
}

func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("format", "2", len(args))
	}
	if args[0].Type != core.VT_STRING && args[0].Type != core.VT_RUNES && args[0].Type != core.VT_BYTES {
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "template", "string", args[0].TypeName())
	}
	tmplStr, _ := args[0].AsString()

	var arr []core.Value
	var dict map[string]core.Value
	switch args[1].Type {
	case core.VT_ARRAY:
		arr = (*core.Array)(args[1].Ptr).Elements
	case core.VT_DICT, core.VT_RECORD:
		dict = (*core.Dict)(args[1].Ptr).Elements
	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array, dict, or record", args[1].TypeName())
	}

	tmpl, err := fspec.ParseTemplate(tmplStr)
	if err != nil {
		return core.Undefined, errs.NewLogicError(err.Error())
	}

	switch tmpl.Mode {
	case fspec.TemplateModeIndexed:
		if args[1].Type != core.VT_ARRAY {
			return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: template uses indexed placeholders but args is %s (expected array)", args[1].TypeName()))
		}
	case fspec.TemplateModeNamed:
		if args[1].Type == core.VT_ARRAY {
			return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: template uses named placeholders but args is %s (expected dict or record)", args[1].TypeName()))
		}
	}

	lookup := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.Index < 0 || seg.Index >= len(arr) {
				return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: index %d out of range [0, %d)", seg.Index, len(arr)))
			}
			return arr[seg.Index], nil
		}
		v, ok := dict[seg.Name]
		if !ok {
			return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: missing key %q", seg.Name))
		}
		return v, nil
	}

	lookupRef := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.SpecRefIndex < 0 || seg.SpecRefIndex >= len(arr) {
				return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: spec ref index %d out of range [0, %d)", seg.SpecRefIndex, len(arr)))
			}
			return arr[seg.SpecRefIndex], nil
		}
		v, ok := dict[seg.SpecRefName]
		if !ok {
			return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: missing spec ref key %q", seg.SpecRefName))
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
				return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: spec reference must be a string, got %s", refVal.TypeName()))
			}
			specStr, _ := refVal.AsString()
			parsed, ferr := fspec.Parse(specStr)
			if ferr != nil {
				return core.Undefined, errs.NewLogicError(fmt.Sprintf("format: %v", ferr))
			}
			spec = parsed
		}
		out, ferr := val.Format(spec)
		if ferr != nil {
			return core.Undefined, ferr
		}
		sb.WriteString(out)
	}
	return vm.Allocator().NewStringValue(sb.String()), nil
}

func builtinCopy(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Copy(vm.Allocator())
}

func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return vm.Allocator().NewStringValue(""), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_STRING:
		return args[0], nil

	default:
		if v, ok := args[0].AsString(); ok {
			return vm.Allocator().NewStringValue(v), nil
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinRunes(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	alloc := vm.Allocator()

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
		if v, ok := args[0].AsRunes(); ok {
			return alloc.NewRunesValue(v, false), nil
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
	case core.VT_INT:
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
	case core.VT_FLOAT:
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
		d := vm.Allocator().NewDecimal()
		*d = dec128.Decimal0
		return core.DecimalValue(d), nil
	}

	switch args[0].Type {
	case core.VT_DECIMAL:
		return args[0], nil

	default:
		v, ok := args[0].AsDecimal()
		if !ok && l == 2 {
			return args[1], nil
		}
		d := vm.Allocator().NewDecimal()
		*d = v
		return core.DecimalValue(d), nil
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
	case core.VT_BOOL:
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
	case core.VT_BYTE:
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
	case core.VT_RUNE:
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
	alloc := vm.Allocator()

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
		if v, ok := args[0].AsBytes(); ok {
			return alloc.NewBytesValue(v, false), nil
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
		d := vm.Allocator().NewTime()
		*d = time.Time{}
		return core.TimeValue(d), nil
	}

	switch args[0].Type {
	case core.VT_TIME:
		return args[0], nil

	default:
		if v, ok := args[0].AsTime(); ok {
			d := vm.Allocator().NewTime()
			*d = v
			return core.TimeValue(d), nil
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
		return vm.Allocator().NewDictValue(nil, false), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("dict", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_DICT:
		return args[0], nil

	case core.VT_RECORD:
		r := (*core.Dict)(args[0].Ptr)
		return vm.Allocator().NewDictValue(r.Elements, args[0].Const), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("dict", "first", "dict or record", args[0].TypeName())
	}
}

// append(arr, items...)
func builtinAppend(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) < 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}
	return args[0].Append(vm.Allocator(), args[1:])
}

// builtinDelete deletes Map keys inplace
// usage: delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("delete", "2", argsLen)
	}
	return args[0].Delete(args[1])
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage: deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}
	if args[0].Type != core.VT_ARRAY {
		return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}
	if args[0].Const {
		return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName())
	}

	arr := (*core.Array)(args[0].Ptr)
	arrayLen := len(arr.Elements)

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName())
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return core.Undefined, errs.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName())
		}
		delCount = int(arg2)
		if delCount < 0 {
			return core.Undefined, errs.NewLogicError("splice delete count must be non-negative")
		}
	}
	// if count of to be deleted items is bigger than expected, truncate it
	if startIdx+delCount > arrayLen {
		delCount = arrayLen - startIdx
	}
	// delete items
	endIdx := startIdx + delCount
	deleted := append([]core.Value{}, arr.Elements[startIdx:endIdx]...)

	alloc := vm.Allocator()
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
