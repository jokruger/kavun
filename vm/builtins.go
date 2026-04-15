package vm

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/formatter"
)

// do not change builtin function indexes as it will break compatibility
// 34..99 are reserved for future builtin functions
var BuiltinFuncs = map[int]core.Value{
	7:  core.NewBuiltinFunctionValue("bool", builtinBool, 0, true),
	9:  core.NewBuiltinFunctionValue("char", builtinChar, 0, true),
	6:  core.NewBuiltinFunctionValue("int", builtinInt, 0, true),
	8:  core.NewBuiltinFunctionValue("float", builtinFloat, 0, true),
	11: core.NewBuiltinFunctionValue("time", builtinTime, 0, true),
	5:  core.NewBuiltinFunctionValue("string", builtinString, 0, true),
	10: core.NewBuiltinFunctionValue("bytes", builtinBytes, 0, true),
	21: core.NewBuiltinFunctionValue("map", builtinMap, 0, true),
	30: core.NewBuiltinFunctionValue("range", builtinRange, 2, true),
	33: core.NewBuiltinFunctionValue("error", builtinError, 0, true),

	15: core.NewBuiltinFunctionValue("is_bool", builtinIsBool, 1, false),
	16: core.NewBuiltinFunctionValue("is_char", builtinIsChar, 1, false),
	12: core.NewBuiltinFunctionValue("is_int", builtinIsInt, 1, false),
	13: core.NewBuiltinFunctionValue("is_float", builtinIsFloat, 1, false),
	23: core.NewBuiltinFunctionValue("is_time", builtinIsTime, 1, false),
	14: core.NewBuiltinFunctionValue("is_string", builtinIsString, 1, false),
	17: core.NewBuiltinFunctionValue("is_bytes", builtinIsBytes, 1, false),
	18: core.NewBuiltinFunctionValue("is_array", builtinIsArray, 1, false),
	31: core.NewBuiltinFunctionValue("is_map", builtinIsMap, 1, false),
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
	29: core.NewBuiltinFunctionValue("format", builtinFormat, 1, true),
	28: core.NewBuiltinFunctionValue("type_name", builtinTypeName, 1, false),
}

func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return vm.Allocator().NewStringValue(args[0].TypeName())
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

func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	if args[0].Type == core.VT_BOOL {
		return core.True, nil
	}
	return core.False, nil
}

func builtinIsChar(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_char", "1", len(args))
	}
	if args[0].Type == core.VT_CHAR {
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

func builtinIsMap(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("is_map", "1", len(args))
	}
	if args[0].Type == core.VT_MAP {
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
	return vm.Allocator().NewErrorValue(payload)
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

	return vm.Allocator().NewIntRangeValue(start, stop, step)
}

func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.Undefined, errs.NewWrongNumArgumentsError("format", "at least 1", numArgs)
	}
	format, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "first", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		return vm.Allocator().NewStringValue(format)
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.Undefined, err
	}
	return vm.Allocator().NewStringValue(s)
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
		return vm.Allocator().NewStringValue("")
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_STRING:
		return args[0], nil

	default:
		if v, ok := args[0].AsString(); ok {
			if len(v) > core.MaxStringLen {
				return core.Undefined, errs.NewStringLimitError("string constructor")
			}
			return vm.Allocator().NewStringValue(v)
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

func builtinChar(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.CharValue(0), nil
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("char", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_CHAR:
		return args[0], nil

	default:
		if v, ok := args[0].AsChar(); ok {
			return core.CharValue(v), nil
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
		return vm.Allocator().NewBytesValue([]byte{})
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_BYTES:
		return args[0], nil

	case core.VT_INT:
		n := core.ToInt(args[0])
		if n > int64(core.MaxBytesLen) {
			return core.Undefined, errs.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytesValue(make([]byte, int(n)))

	default:
		if v, ok := args[0].AsBytes(); ok {
			if len(v) > core.MaxBytesLen {
				return core.Undefined, errs.NewBytesLimitError("bytes constructor")
			}
			return vm.Allocator().NewBytesValue(v)
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinTime(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return vm.Allocator().NewTimeValue(time.Time{})
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_TIME:
		return args[0], nil

	default:
		if v, ok := args[0].AsTime(); ok {
			return vm.Allocator().NewTimeValue(v)
		}
		if l == 2 {
			return args[1], nil
		}
		return core.Undefined, nil
	}
}

func builtinMap(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return vm.Allocator().NewMapValue(nil, false)
	}
	if l > 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("map", "0, 1 or 2", len(args))
	}

	switch args[0].Type {
	case core.VT_MAP:
		return args[0], nil

	case core.VT_RECORD:
		r := (*core.Record)(args[0].Ptr)
		return vm.Allocator().NewMapValue(r.Elements, r.Immutable)

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("map", "first", "map or record", args[0].TypeName())
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
	arr := (*core.Array)(args[0].Ptr)

	if args[0].IsImmutable() {
		return core.Undefined, errs.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName())
	}

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

	head := arr.Elements[:startIdx]
	var items []core.Value
	if argsLen > 3 {
		items = make([]core.Value, 0, argsLen-3)
		for i := 3; i < argsLen; i++ {
			items = append(items, args[i])
		}
	}
	items = append(items, arr.Elements[endIdx:]...)
	arr.Set(append(head, items...), false)

	// return deleted items
	return vm.Allocator().NewArrayValue(deleted, false)
}
