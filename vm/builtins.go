package vm

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/formatter"
)

// do not change builtin function indexes as it will break compatibility
// 32..99 are reserved for future builtin functions
var BuiltinFuncs = map[int]core.Value{
	7:  core.NewBuiltinFunctionValue("bool", builtinBool, 0, true),
	9:  core.NewBuiltinFunctionValue("char", builtinChar, 0, true),
	6:  core.NewBuiltinFunctionValue("int", builtinInt, 0, true),
	8:  core.NewBuiltinFunctionValue("float", builtinFloat, 0, true),
	5:  core.NewBuiltinFunctionValue("string", builtinString, 0, true),
	10: core.NewBuiltinFunctionValue("bytes", builtinBytes, 0, true),
	11: core.NewBuiltinFunctionValue("time", builtinTime, 0, true),
	21: core.NewBuiltinFunctionValue("map", builtinMap, 0, true),

	15: core.NewBuiltinFunctionValue("is_bool", builtinIsBool, 1, false),
	16: core.NewBuiltinFunctionValue("is_char", builtinIsChar, 1, false),
	12: core.NewBuiltinFunctionValue("is_int", builtinIsInt, 1, false),
	13: core.NewBuiltinFunctionValue("is_float", builtinIsFloat, 1, false),
	14: core.NewBuiltinFunctionValue("is_string", builtinIsString, 1, false),
	17: core.NewBuiltinFunctionValue("is_bytes", builtinIsBytes, 1, false),
	23: core.NewBuiltinFunctionValue("is_time", builtinIsTime, 1, false),
	18: core.NewBuiltinFunctionValue("is_array", builtinIsArray, 1, false),
	20: core.NewBuiltinFunctionValue("is_record", builtinIsRecord, 1, false),
	31: core.NewBuiltinFunctionValue("is_map", builtinIsMap, 1, false),

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
	30: core.NewBuiltinFunctionValue("range", builtinRange, 2, true),
	28: core.NewBuiltinFunctionValue("type_name", builtinTypeName, 1, false),
}

func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return vm.Allocator().NewStringValue(args[0].TypeName()), nil
}

func builtinIsString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	return core.BoolValue(args[0].IsString()), nil
}

func builtinIsInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	return core.BoolValue(args[0].IsInt()), nil
}

func builtinIsFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	return core.BoolValue(args[0].IsFloat()), nil
}

func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	return core.BoolValue(args[0].IsBool()), nil
}

func builtinIsChar(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_char", "1", len(args))
	}
	return core.BoolValue(args[0].IsChar()), nil
}

func builtinIsBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	return core.BoolValue(args[0].IsBytes()), nil
}

func builtinIsArray(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	return core.BoolValue(args[0].IsArray()), nil
}

func builtinIsRecord(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	return core.BoolValue(args[0].IsRecord()), nil
}

func builtinIsMap(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_map", "1", len(args))
	}
	return core.BoolValue(args[0].IsMap()), nil
}

func builtinIsImmutable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return core.BoolValue(args[0].IsImmutable()), nil
}

func builtinIsTime(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	return core.BoolValue(args[0].IsTime()), nil
}

func builtinIsError(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	return core.BoolValue(args[0].IsError()), nil
}

func builtinIsUndefined(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	return core.BoolValue(args[0].IsUndefined()), nil
}

func builtinIsFunction(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_function", "1", len(args))
	}
	return core.BoolValue(args[0].IsCompiledFunction()), nil
}

func builtinIsCallable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return core.BoolValue(args[0].IsCallable()), nil
}

func builtinIsIterable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return core.BoolValue(args[0].IsIterable()), nil
}

// len(obj object) => int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("len", "1", len(args))
	}

	arg := args[0]
	switch arg.Type {
	case core.VT_ARRAY:
		o := (*core.Array)(arg.Ptr)
		return core.IntValue(int64(len(o.Elements))), nil
	case core.VT_STRING:
		o := (*core.String)(arg.Ptr)
		return core.IntValue(int64(o.Len())), nil
	case core.VT_BYTES:
		o := (*core.Bytes)(arg.Ptr)
		return core.IntValue(int64(len(o.Elements))), nil
	case core.VT_RECORD:
		o := (*core.Record)(arg.Ptr)
		return core.IntValue(int64(len(o.Elements))), nil
	case core.VT_MAP:
		o := (*core.Map)(arg.Ptr)
		return core.IntValue(int64(len(o.Elements))), nil
	case core.VT_INT_RANGE:
		o := (*core.IntRange)(arg.Ptr)
		return core.IntValue(o.Len()), nil
	default:
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", arg.TypeName())
	}
}

// range(start, stop[, step])
func builtinRange(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
	}

	start, ok := args[0].AsInt()
	if !ok {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("range", "start", "int", args[0].TypeName())
	}

	stop, ok := args[1].AsInt()
	if !ok {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("range", "stop", "int", args[1].TypeName())
	}

	step := int64(1)
	if numArgs == 3 {
		step, ok = args[2].AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("range", "step", "int", args[2].TypeName())
		}
		if step <= 0 {
			return core.UndefinedValue(), errs.NewLogicError(fmt.Sprintf("range step must be greater than 0, got %d", step))
		}
	}

	return vm.Allocator().NewIntRangeValue(start, stop, step), nil
}

func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("format", "at least 1", numArgs)
	}
	format, ok := args[0].AsString()
	if !ok {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("format", "first", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		return vm.Allocator().NewStringValue(format), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.UndefinedValue(), err
	}
	return vm.Allocator().NewStringValue(s), nil
}

func builtinCopy(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Copy(vm.Allocator()), nil
}

func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return vm.Allocator().NewStringValue(""), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsString()
	if ok {
		if len(v) > core.MaxStringLen {
			return core.UndefinedValue(), errs.NewStringLimitError("string constructor")
		}
		return vm.Allocator().NewStringValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.IntValue(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsInt()
	if ok {
		return core.IntValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.FloatValue(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsFloat()
	if ok {
		return core.FloatValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.BoolValue(false), nil
	}

	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("bool", "0 or 1", len(args))
	}

	v, ok := args[0].AsBool()
	if ok {
		return core.BoolValue(v), nil
	}

	return core.UndefinedValue(), nil
}

func builtinChar(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.CharValue(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("char", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsChar()
	if ok {
		return core.CharValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return vm.Allocator().NewBytesValue([]byte{}), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	// bytes(N) => create a new bytes with given size N
	if args[0].IsInt() {
		n := core.ToInt(args[0])
		if n > int64(core.MaxBytesLen) {
			return core.UndefinedValue(), errs.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytesValue(make([]byte, int(n))), nil
	}

	if v, ok := args[0].AsBytes(); ok {
		if len(v) > core.MaxBytesLen {
			return core.UndefinedValue(), errs.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytesValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinTime(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return vm.Allocator().NewTimeValue(time.Time{}), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	if v, ok := args[0].AsTime(); ok {
		return vm.Allocator().NewTimeValue(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.UndefinedValue(), nil
}

func builtinMap(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return vm.Allocator().NewMapValue(nil, false), nil
	}

	if len(args) != 1 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("map", "0 or 1", len(args))
	}

	alloc := vm.Allocator()
	arg := args[0]
	switch arg.Type {
	case core.VT_MAP:
		m := (*core.Map)(arg.Ptr)
		v := make(map[string]core.Value, len(m.Elements))
		for k, o := range m.Elements {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMapValue(v, false), nil

	case core.VT_RECORD:
		r := (*core.Record)(arg.Ptr)
		v := make(map[string]core.Value, len(r.Elements))
		for k, o := range r.Elements {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMapValue(v, false), nil

	default:
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("map", "first", "map or record", arg.TypeName())
	}
}

// append(arr, items...)
func builtinAppend(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) < 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}

	arg := args[0]
	switch arg.Type {
	case core.VT_ARRAY:
		o := (*core.Array)(arg.Ptr)
		return vm.Allocator().NewArrayValue(append(o.Elements, args[1:]...), false), nil

	default:
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("append", "first", "array", arg.TypeName())
	}
}

// builtinDelete deletes Map keys
// usage: delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("delete", "2", argsLen)
	}

	if args[0].IsImmutable() {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("delete", "first", "mutable record or map", args[0].TypeName())
	}

	arg := args[0]
	switch arg.Type {
	case core.VT_RECORD:
		if key, ok := args[1].AsString(); ok {
			o := (*core.Record)(arg.Ptr)
			delete(o.Elements, key)
			return core.UndefinedValue(), nil
		}
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	case core.VT_MAP:
		if key, ok := args[1].AsString(); ok {
			o := (*core.Map)(arg.Ptr)
			delete(o.Elements, key)
			return core.UndefinedValue(), nil
		}
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	default:
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("delete", "first", "record or map", arg.TypeName())
	}
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage: deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return core.UndefinedValue(), errs.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}

	if !args[0].IsArray() {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}
	arr := (*core.Array)(args[0].Ptr)

	if args[0].IsImmutable() {
		return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName())
	}

	arrayLen := len(arr.Elements)

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName())
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return core.UndefinedValue(), errs.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName())
		}
		delCount = int(arg2)
		if delCount < 0 {
			return core.UndefinedValue(), errs.NewLogicError("splice delete count must be non-negative")
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
	return vm.Allocator().NewArrayValue(deleted, false), nil
}
