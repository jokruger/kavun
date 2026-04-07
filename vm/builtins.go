package vm

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/formatter"
	"github.com/jokruger/gs/value"
)

// do not change builtin function indexes as it will break compatibility
// 32..99 are reserved for future builtin functions
var BuiltinFuncs = map[int]core.Value{
	7:  core.NewStaticBuiltinFunction("bool", builtinBool, 0, true),
	9:  core.NewStaticBuiltinFunction("char", builtinChar, 0, true),
	6:  core.NewStaticBuiltinFunction("int", builtinInt, 0, true),
	8:  core.NewStaticBuiltinFunction("float", builtinFloat, 0, true),
	5:  core.NewStaticBuiltinFunction("string", builtinString, 0, true),
	10: core.NewStaticBuiltinFunction("bytes", builtinBytes, 0, true),
	11: core.NewStaticBuiltinFunction("time", builtinTime, 0, true),
	21: core.NewStaticBuiltinFunction("map", builtinMap, 0, true),

	15: core.NewStaticBuiltinFunction("is_bool", builtinIsBool, 1, false),
	16: core.NewStaticBuiltinFunction("is_char", builtinIsChar, 1, false),
	12: core.NewStaticBuiltinFunction("is_int", builtinIsInt, 1, false),
	13: core.NewStaticBuiltinFunction("is_float", builtinIsFloat, 1, false),
	14: core.NewStaticBuiltinFunction("is_string", builtinIsString, 1, false),
	17: core.NewStaticBuiltinFunction("is_bytes", builtinIsBytes, 1, false),
	23: core.NewStaticBuiltinFunction("is_time", builtinIsTime, 1, false),
	18: core.NewStaticBuiltinFunction("is_array", builtinIsArray, 1, false),
	20: core.NewStaticBuiltinFunction("is_record", builtinIsRecord, 1, false),
	31: core.NewStaticBuiltinFunction("is_map", builtinIsMap, 1, false),

	24: core.NewStaticBuiltinFunction("is_error", builtinIsError, 1, false),
	25: core.NewStaticBuiltinFunction("is_undefined", builtinIsUndefined, 1, false),
	26: core.NewStaticBuiltinFunction("is_function", builtinIsFunction, 1, false),
	27: core.NewStaticBuiltinFunction("is_callable", builtinIsCallable, 1, false),
	22: core.NewStaticBuiltinFunction("is_iterable", builtinIsIterable, 1, false),
	19: core.NewStaticBuiltinFunction("is_immutable", builtinIsImmutable, 1, false),

	0:  core.NewStaticBuiltinFunction("len", builtinLen, 1, false),
	1:  core.NewStaticBuiltinFunction("copy", builtinCopy, 1, false),
	2:  core.NewStaticBuiltinFunction("append", builtinAppend, 2, true),
	3:  core.NewStaticBuiltinFunction("delete", builtinDelete, 2, false),
	4:  core.NewStaticBuiltinFunction("splice", builtinSplice, 1, true),
	29: core.NewStaticBuiltinFunction("format", builtinFormat, 1, true),
	30: core.NewStaticBuiltinFunction("range", builtinRange, 2, true),
	28: core.NewStaticBuiltinFunction("type_name", builtinTypeName, 1, false),
}

func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return vm.Allocator().NewStringValue(args[0].TypeName()), nil
}

func builtinIsString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	return core.BoolValue(args[0].IsString()), nil
}

func builtinIsInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	return core.BoolValue(args[0].IsInt()), nil
}

func builtinIsFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	return core.BoolValue(args[0].IsFloat()), nil
}

func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	return core.BoolValue(args[0].IsBool()), nil
}

func builtinIsChar(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_char", "1", len(args))
	}
	return core.BoolValue(args[0].IsChar()), nil
}

func builtinIsBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	return core.BoolValue(args[0].IsBytes()), nil
}

func builtinIsArray(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	return core.BoolValue(args[0].IsArray()), nil
}

func builtinIsRecord(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	return core.BoolValue(args[0].IsRecord()), nil
}

func builtinIsMap(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_map", "1", len(args))
	}
	return core.BoolValue(args[0].IsMap()), nil
}

func builtinIsImmutable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return core.BoolValue(args[0].IsImmutable()), nil
}

func builtinIsTime(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	return core.BoolValue(args[0].IsTime()), nil
}

func builtinIsError(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	return core.BoolValue(args[0].IsError()), nil
}

func builtinIsUndefined(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	return core.BoolValue(args[0].IsUndefined()), nil
}

func builtinIsFunction(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_function", "1", len(args))
	}
	return core.BoolValue(args[0].IsCompiledFunction()), nil
}

func builtinIsCallable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return core.BoolValue(args[0].IsCallable()), nil
}

func builtinIsIterable(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return core.BoolValue(args[0].IsIterable()), nil
}

// len(obj object) => int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("len", "1", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Array:
		return core.IntValue(int64(arg.Len())), nil
	case *value.String:
		return core.IntValue(int64(arg.Len())), nil
	case *value.Bytes:
		return core.IntValue(int64(arg.Len())), nil
	case *value.Record:
		return core.IntValue(int64(arg.Len())), nil
	case *value.Map:
		return core.IntValue(int64(arg.Len())), nil
	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", arg.TypeName())
	}
}

// range(start, stop[, step])
func builtinRange(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
	}

	var start, stop, step int64
	for i, arg := range args {
		v, ok := args[i].AsInt()
		if !ok {
			var name string
			switch i {
			case 0:
				name = "start"
			case 1:
				name = "stop"
			case 2:
				name = "step"
			}
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("range", name, "int", arg.TypeName())
		}

		if i == 2 && v <= 0 {
			return core.UndefinedValue(), core.NewLogicError(fmt.Sprintf("range step must be greater than 0, got %d", v))
		}

		switch i {
		case 0:
			start = v
		case 1:
			stop = v
		case 2:
			step = v
		}
	}

	if step == 0 {
		step = 1
	}

	return buildRange(vm.Allocator(), start, stop, step), nil
}

func buildRange(alloc core.Allocator, start, stop, step int64) core.Value {
	if start == stop {
		return alloc.NewArrayValue([]core.Value{}, false)
	}

	if start < stop {
		array := make([]core.Value, 0, (stop-start+step-1)/step)
		for i := start; i < stop; i += step {
			array = append(array, core.IntValue(i))
		}
		return alloc.NewArrayValue(array, false)
	}

	array := make([]core.Value, 0, (start-stop+step-1)/step)
	for i := start; i > stop; i -= step {
		array = append(array, core.IntValue(i))
	}
	return alloc.NewArrayValue(array, false)
}

func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("format", "at least 1", numArgs)
	}
	format, ok := args[0].AsString()
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("format", "first", "string", args[0].TypeName())
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Copy(vm.Allocator()), nil
}

func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) == 0 {
		return vm.Allocator().NewStringValue(""), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsString()
	if ok {
		if len(v) > core.MaxStringLen {
			return core.UndefinedValue(), core.NewStringLimitError("string constructor")
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("bool", "0 or 1", len(args))
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("char", "0, 1 or 2", len(args))
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	// bytes(N) => create a new bytes with given size N
	if args[0].IsInt() {
		n := args[0].Int()
		if n > int64(core.MaxBytesLen) {
			return core.UndefinedValue(), core.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytesValue(make([]byte, int(n))), nil
	}

	if v, ok := args[0].AsBytes(); ok {
		if len(v) > core.MaxBytesLen {
			return core.UndefinedValue(), core.NewBytesLimitError("bytes constructor")
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
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
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("map", "0 or 1", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("map", "first", "record or map", args[0].TypeName())
	}

	alloc := vm.Allocator()
	switch arg := args[0].Object().(type) {
	case *value.Map:
		v := make(map[string]core.Value, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMapValue(v, false), nil

	case *value.Record:
		v := make(map[string]core.Value, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMapValue(v, false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("map", "first", "map or record", arg.TypeName())
	}
}

// append(arr, items...)
func builtinAppend(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) < 2 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("append", "first", "array", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Array:
		return vm.Allocator().NewArrayValue(append(arg.Value(), args[1:]...), false), nil

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("append", "first", "array", arg.TypeName())
	}
}

// builtinDelete deletes Map keys
// usage: delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("delete", "2", argsLen)
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("delete", "first", "record or map", args[0].TypeName())
	}

	if args[0].IsImmutable() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("delete", "first", "mutable record or map", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Record:
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return core.UndefinedValue(), nil
		}
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	case *value.Map:
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return core.UndefinedValue(), nil
		}
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	default:
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("delete", "first", "record or map", arg.TypeName())
	}
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage: deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(vm core.VM, args []core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return core.UndefinedValue(), core.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}

	if !args[0].IsObject() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}

	array, ok := args[0].Object().(*value.Array)
	if !ok {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}

	if array.IsImmutable() {
		return core.UndefinedValue(), core.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName())
	}

	arrayLen := int(array.Len())

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName())
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return core.UndefinedValue(), core.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return core.UndefinedValue(), core.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName())
		}
		delCount = int(arg2)
		if delCount < 0 {
			return core.UndefinedValue(), core.NewLogicError("splice delete count must be non-negative")
		}
	}
	// if count of to be deleted items is bigger than expected, truncate it
	if startIdx+delCount > arrayLen {
		delCount = arrayLen - startIdx
	}
	// delete items
	endIdx := startIdx + delCount
	deleted := append([]core.Value{}, array.Slice(startIdx, endIdx)...)

	head := array.Slice(0, startIdx)
	var items []core.Value
	if argsLen > 3 {
		items = make([]core.Value, 0, argsLen-3)
		for i := 3; i < argsLen; i++ {
			items = append(items, args[i])
		}
	}
	items = append(items, array.Slice(endIdx, array.Len())...)
	array.Set(append(head, items...), false)

	// return deleted items
	return vm.Allocator().NewArrayValue(deleted, false), nil
}
