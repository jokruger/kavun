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
	7:  value.NewStaticBuiltinFunction("bool", builtinBool, 0, true),
	9:  value.NewStaticBuiltinFunction("char", builtinChar, 0, true),
	6:  value.NewStaticBuiltinFunction("int", builtinInt, 0, true),
	8:  value.NewStaticBuiltinFunction("float", builtinFloat, 0, true),
	5:  value.NewStaticBuiltinFunction("string", builtinString, 0, true),
	10: value.NewStaticBuiltinFunction("bytes", builtinBytes, 0, true),
	11: value.NewStaticBuiltinFunction("time", builtinTime, 0, true),
	21: value.NewStaticBuiltinFunction("map", builtinMap, 0, true),

	15: value.NewStaticBuiltinFunction("is_bool", builtinIsBool, 1, false),
	16: value.NewStaticBuiltinFunction("is_char", builtinIsChar, 1, false),
	12: value.NewStaticBuiltinFunction("is_int", builtinIsInt, 1, false),
	13: value.NewStaticBuiltinFunction("is_float", builtinIsFloat, 1, false),
	14: value.NewStaticBuiltinFunction("is_string", builtinIsString, 1, false),
	17: value.NewStaticBuiltinFunction("is_bytes", builtinIsBytes, 1, false),
	23: value.NewStaticBuiltinFunction("is_time", builtinIsTime, 1, false),
	18: value.NewStaticBuiltinFunction("is_array", builtinIsArray, 1, false),
	20: value.NewStaticBuiltinFunction("is_record", builtinIsRecord, 1, false),
	31: value.NewStaticBuiltinFunction("is_map", builtinIsMap, 1, false),

	24: value.NewStaticBuiltinFunction("is_error", builtinIsError, 1, false),
	25: value.NewStaticBuiltinFunction("is_undefined", builtinIsUndefined, 1, false),
	26: value.NewStaticBuiltinFunction("is_function", builtinIsFunction, 1, false),
	27: value.NewStaticBuiltinFunction("is_callable", builtinIsCallable, 1, false),
	22: value.NewStaticBuiltinFunction("is_iterable", builtinIsIterable, 1, false),
	19: value.NewStaticBuiltinFunction("is_immutable", builtinIsImmutable, 1, false),

	0:  value.NewStaticBuiltinFunction("len", builtinLen, 1, false),
	1:  value.NewStaticBuiltinFunction("copy", builtinCopy, 1, false),
	2:  value.NewStaticBuiltinFunction("append", builtinAppend, 2, true),
	3:  value.NewStaticBuiltinFunction("delete", builtinDelete, 2, false),
	4:  value.NewStaticBuiltinFunction("splice", builtinSplice, 1, true),
	29: value.NewStaticBuiltinFunction("format", builtinFormat, 1, true),
	30: value.NewStaticBuiltinFunction("range", builtinRange, 2, true),
	28: value.NewStaticBuiltinFunction("type_name", builtinTypeName, 1, false),
}

func builtinTypeName(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return core.NewObject(vm.Allocator().NewString(args[0].TypeName()), false), nil
}

func builtinIsString(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	return core.NewBool(args[0].IsString()), nil
}

func builtinIsInt(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	return core.NewBool(args[0].IsInt()), nil
}

func builtinIsFloat(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	return core.NewBool(args[0].IsFloat()), nil
}

func builtinIsBool(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	return core.NewBool(args[0].IsBool()), nil
}

func builtinIsChar(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_char", "1", len(args))
	}
	return core.NewBool(args[0].IsChar()), nil
}

func builtinIsBytes(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	return core.NewBool(args[0].IsBytes()), nil
}

func builtinIsArray(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	return core.NewBool(args[0].IsArray()), nil
}

func builtinIsRecord(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	return core.NewBool(args[0].IsRecord()), nil
}

func builtinIsMap(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_map", "1", len(args))
	}
	return core.NewBool(args[0].IsMap()), nil
}

func builtinIsImmutable(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return core.NewBool(args[0].IsImmutable()), nil
}

func builtinIsTime(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	return core.NewBool(args[0].IsTime()), nil
}

func builtinIsError(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	return core.NewBool(args[0].IsError()), nil
}

func builtinIsUndefined(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	return core.NewBool(args[0].IsUndefined()), nil
}

func builtinIsFunction(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_function", "1", len(args))
	}
	return core.NewBool(args[0].IsCompiledFunction()), nil
}

func builtinIsCallable(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return core.NewBool(args[0].IsCallable()), nil
}

func builtinIsIterable(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return core.NewBool(args[0].IsIterable()), nil
}

// len(obj object) => int
func builtinLen(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("len", "1", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Array:
		return core.NewInt(int64(arg.Len())), nil
	case *value.String:
		return core.NewInt(int64(arg.Len())), nil
	case *value.Bytes:
		return core.NewInt(int64(arg.Len())), nil
	case *value.Record:
		return core.NewInt(int64(arg.Len())), nil
	case *value.Map:
		return core.NewInt(int64(arg.Len())), nil
	default:
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", arg.TypeName())
	}
}

// range(start, stop[, step])
func builtinRange(vm core.VM, args ...core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
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
			return core.NewUndefined(), core.NewInvalidArgumentTypeError("range", name, "int", arg.TypeName())
		}

		if i == 2 && v <= 0 {
			return core.NewUndefined(), core.NewLogicError(fmt.Sprintf("range step must be greater than 0, got %d", v))
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
			array = append(array, core.NewInt(i))
		}
		return alloc.NewArrayValue(array, false)
	}

	array := make([]core.Value, 0, (start-stop+step-1)/step)
	for i := start; i > stop; i -= step {
		array = append(array, core.NewInt(i))
	}
	return alloc.NewArrayValue(array, false)
}

func builtinFormat(vm core.VM, args ...core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("format", "at least 1", numArgs)
	}
	format, ok := args[0].AsString()
	if !ok {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("format", "first", "string", args[0].TypeName())
	}
	if numArgs == 1 {
		return core.NewObject(vm.Allocator().NewString(format), false), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return core.NewUndefined(), err
	}
	return core.NewObject(vm.Allocator().NewString(s), false), nil
}

func builtinCopy(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Copy(vm.Allocator()), nil
}

func builtinString(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewObject(vm.Allocator().NewString(""), false), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsString()
	if ok {
		if len(v) > core.MaxStringLen {
			return core.NewUndefined(), core.NewStringLimitError("string constructor")
		}
		return core.NewObject(vm.Allocator().NewString(v), false), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinInt(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewInt(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsInt()
	if ok {
		return core.NewInt(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinFloat(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewFloat(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsFloat()
	if ok {
		return core.NewFloat(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinBool(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewBool(false), nil
	}

	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("bool", "0 or 1", len(args))
	}

	v, ok := args[0].AsBool()
	if ok {
		return core.NewBool(v), nil
	}

	return core.NewUndefined(), nil
}

func builtinChar(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewChar(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("char", "0, 1 or 2", len(args))
	}

	v, ok := args[0].AsChar()
	if ok {
		return core.NewChar(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinBytes(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		return core.NewObject(vm.Allocator().NewBytes([]byte{}), false), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	// bytes(N) => create a new bytes with given size N
	if args[0].IsInt() {
		n := args[0].Int()
		if n > int64(core.MaxBytesLen) {
			return core.NewUndefined(), core.NewBytesLimitError("bytes constructor")
		}
		t := vm.Allocator().NewBytes(make([]byte, int(n)))
		return core.NewObject(t, false), nil
	}

	if v, ok := args[0].AsBytes(); ok {
		if len(v) > core.MaxBytesLen {
			return core.NewUndefined(), core.NewBytesLimitError("bytes constructor")
		}
		t := vm.Allocator().NewBytes(v)
		return core.NewObject(t, false), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinTime(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		t := vm.Allocator().NewTime(time.Time{})
		return core.NewObject(t, false), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	if v, ok := args[0].AsTime(); ok {
		t := vm.Allocator().NewTime(v)
		return core.NewObject(t, false), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return core.NewUndefined(), nil
}

func builtinMap(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) == 0 {
		t := vm.Allocator().NewMap(nil, false)
		return core.NewObject(t, false), nil
	}

	if len(args) != 1 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("map", "0 or 1", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("map", "first", "record or map", args[0].TypeName())
	}

	alloc := vm.Allocator()
	switch arg := args[0].Object().(type) {
	case *value.Map:
		v := make(map[string]core.Value, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		t := vm.Allocator().NewMap(v, false)
		return core.NewObject(t, false), nil

	case *value.Record:
		v := make(map[string]core.Value, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		t := vm.Allocator().NewMap(v, false)
		return core.NewObject(t, false), nil

	default:
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("map", "first", "map or record", arg.TypeName())
	}
}

// append(arr, items...)
func builtinAppend(vm core.VM, args ...core.Value) (core.Value, error) {
	if len(args) < 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("append", "first", "array", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Array:
		t := vm.Allocator().NewArray(append(arg.Value(), args[1:]...), false)
		return core.NewObject(t, false), nil

	default:
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("append", "first", "array", arg.TypeName())
	}
}

// builtinDelete deletes Map keys
// usage: delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args ...core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("delete", "2", argsLen)
	}

	if args[0].Kind() != core.V_OBJECT {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("delete", "first", "record or map", args[0].TypeName())
	}

	if args[0].IsImmutable() {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("delete", "first", "mutable record or map", args[0].TypeName())
	}

	switch arg := args[0].Object().(type) {
	case *value.Record:
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return core.NewUndefined(), nil
		}
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	case *value.Map:
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return core.NewUndefined(), nil
		}
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("delete", "second", "string", args[1].TypeName())

	default:
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("delete", "first", "record or map", arg.TypeName())
	}
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage: deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(vm core.VM, args ...core.Value) (core.Value, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return core.NewUndefined(), core.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}

	if !args[0].IsObject() {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}

	array, ok := args[0].Object().(*value.Array)
	if !ok {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("splice", "first", "array", args[0].TypeName())
	}

	if array.IsImmutable() {
		return core.NewUndefined(), core.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0].TypeName())
	}

	arrayLen := int(array.Len())

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return core.NewUndefined(), core.NewInvalidArgumentTypeError("splice", "second", "int", args[1].TypeName())
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return core.NewUndefined(), core.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return core.NewUndefined(), core.NewInvalidArgumentTypeError("splice", "third", "int", args[2].TypeName())
		}
		delCount = int(arg2)
		if delCount < 0 {
			return core.NewUndefined(), core.NewLogicError("splice delete count must be non-negative")
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
	t := vm.Allocator().NewArray(deleted, false)
	return core.NewObject(t, false), nil
}
