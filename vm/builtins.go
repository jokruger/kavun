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
var BuiltinFuncs = map[int]*value.BuiltinFunction{
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

func builtinTypeName(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("type_name", "1", len(args))
	}
	return vm.Allocator().NewString(args[0].TypeName()), nil
}

func builtinIsString(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_string", "1", len(args))
	}
	_, ok := args[0].(*value.String)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsInt(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_int", "1", len(args))
	}
	_, ok := args[0].(*value.Int)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsFloat(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_float", "1", len(args))
	}
	_, ok := args[0].(*value.Float)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsBool(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_bool", "1", len(args))
	}
	_, ok := args[0].(*value.Bool)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsChar(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_char", "1", len(args))
	}
	_, ok := args[0].(*value.Char)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsBytes(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_bytes", "1", len(args))
	}
	_, ok := args[0].(*value.Bytes)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsArray(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_array", "1", len(args))
	}
	_, ok := args[0].(*value.Array)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsRecord(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_record", "1", len(args))
	}
	_, ok := args[0].(*value.Record)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsMap(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_map", "1", len(args))
	}
	_, ok := args[0].(*value.Map)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsImmutable(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_immutable", "1", len(args))
	}
	return vm.Allocator().NewBool(args[0].IsImmutable()), nil
}

func builtinIsTime(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_time", "1", len(args))
	}
	_, ok := args[0].(*value.Time)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsError(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_error", "1", len(args))
	}
	_, ok := args[0].(*value.Error)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsUndefined(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_undefined", "1", len(args))
	}
	return vm.Allocator().NewBool(args[0].IsUndefined()), nil
}

func builtinIsFunction(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_function", "1", len(args))
	}
	_, ok := args[0].(*value.CompiledFunction)
	return vm.Allocator().NewBool(ok), nil
}

func builtinIsCallable(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_callable", "1", len(args))
	}
	return vm.Allocator().NewBool(args[0].IsCallable()), nil
}

func builtinIsIterable(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("is_iterable", "1", len(args))
	}
	return vm.Allocator().NewBool(args[0].IsIterable()), nil
}

// len(obj object) => int
func builtinLen(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("len", "1", len(args))
	}
	switch arg := args[0].(type) {
	case *value.Array:
		return vm.Allocator().NewInt(int64(arg.Len())), nil
	case *value.String:
		return vm.Allocator().NewInt(int64(arg.Len())), nil
	case *value.Bytes:
		return vm.Allocator().NewInt(int64(arg.Len())), nil
	case *value.Record:
		return vm.Allocator().NewInt(int64(arg.Len())), nil
	case *value.Map:
		return vm.Allocator().NewInt(int64(arg.Len())), nil
	default:
		return nil, core.NewInvalidArgumentTypeError("len", "first", "record/map/array/string/bytes", arg)
	}
}

// range(start, stop[, step])
func builtinRange(vm core.VM, args ...core.Object) (core.Object, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return nil, core.NewWrongNumArgumentsError("range", "2 or 3", numArgs)
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
			return nil, core.NewInvalidArgumentTypeError("range", name, "int", arg)
		}

		if i == 2 && v <= 0 {
			return nil, core.NewLogicError(fmt.Sprintf("range step must be greater than 0, got %d", v))
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

func buildRange(alloc core.Allocator, start, stop, step int64) core.Object {
	array := make([]core.Object, 0)
	if start <= stop {
		for i := start; i < stop; i += step {
			array = append(array, alloc.NewInt(i))
		}
	} else {
		for i := start; i > stop; i -= step {
			array = append(array, alloc.NewInt(i))
		}
	}
	return alloc.NewArray(array, false)
}

func builtinFormat(vm core.VM, args ...core.Object) (core.Object, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, core.NewWrongNumArgumentsError("format", "at least 1", numArgs)
	}
	format, ok := args[0].AsString()
	if !ok {
		return nil, core.NewInvalidArgumentTypeError("format", "first", "string", args[0])
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return vm.Allocator().NewString(format), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return nil, err
	}
	return vm.Allocator().NewString(s), nil
}

func builtinCopy(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("copy", "1", len(args))
	}
	return args[0].Copy(vm.Allocator()), nil
}

func builtinString(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewString(""), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("string", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.String); ok {
		return args[0], nil
	}

	v, ok := args[0].AsString()
	if ok {
		if len(v) > core.MaxStringLen {
			return nil, core.NewStringLimitError("string constructor")
		}
		return vm.Allocator().NewString(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinInt(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewInt(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("int", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.Int); ok {
		return args[0], nil
	}

	v, ok := args[0].AsInt()
	if ok {
		return vm.Allocator().NewInt(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinFloat(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewFloat(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("float", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.Float); ok {
		return args[0], nil
	}

	v, ok := args[0].AsFloat()
	if ok {
		return vm.Allocator().NewFloat(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinBool(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewBool(false), nil
	}

	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("bool", "0 or 1", len(args))
	}

	if _, ok := args[0].(*value.Bool); ok {
		return args[0], nil
	}

	v, ok := args[0].AsBool()
	if ok {
		return vm.Allocator().NewBool(v), nil
	}

	return vm.Allocator().NewUndefined(), nil
}

func builtinChar(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewChar(0), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("char", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.Char); ok {
		return args[0], nil
	}

	v, ok := args[0].AsRune()
	if ok {
		return vm.Allocator().NewChar(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinBytes(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewBytes([]byte{}), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("bytes", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.Bytes); ok {
		return args[0], nil
	}

	// bytes(N) => create a new bytes with given size N
	if n, ok := args[0].(*value.Int); ok {
		if n.Value() > int64(core.MaxBytesLen) {
			return nil, core.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytes(make([]byte, int(n.Value()))), nil
	}

	v, ok := args[0].AsBytes()
	if ok {
		if len(v) > core.MaxBytesLen {
			return nil, core.NewBytesLimitError("bytes constructor")
		}
		return vm.Allocator().NewBytes(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinTime(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewTime(time.Time{}), nil
	}

	if len(args) != 1 && len(args) != 2 {
		return nil, core.NewWrongNumArgumentsError("time", "0, 1 or 2", len(args))
	}

	if _, ok := args[0].(*value.Time); ok {
		return args[0], nil
	}

	v, ok := args[0].AsTime()
	if ok {
		return vm.Allocator().NewTime(v), nil
	}

	if len(args) == 2 {
		return args[1], nil
	}
	return vm.Allocator().NewUndefined(), nil
}

func builtinMap(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) == 0 {
		return vm.Allocator().NewMap(nil, false), nil
	}

	if len(args) != 1 {
		return nil, core.NewWrongNumArgumentsError("map", "0 or 1", len(args))
	}

	alloc := vm.Allocator()
	switch arg := args[0].(type) {
	case *value.Map:
		v := make(map[string]core.Object, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMap(v, false), nil
	case *value.Record:
		v := make(map[string]core.Object, arg.Len())
		for k, o := range arg.Value() {
			v[k] = o.Copy(alloc)
		}
		return vm.Allocator().NewMap(v, false), nil
	default:
		return nil, core.NewInvalidArgumentTypeError("map", "first", "map or record", arg)
	}
}

// append(arr, items...)
func builtinAppend(vm core.VM, args ...core.Object) (core.Object, error) {
	if len(args) < 2 {
		return nil, core.NewWrongNumArgumentsError("append", "at least 2", len(args))
	}
	switch arg := args[0].(type) {
	case *value.Array:
		return vm.Allocator().NewArray(append(arg.Value(), args[1:]...), false), nil
	default:
		return nil, core.NewInvalidArgumentTypeError("append", "first", "array", arg)
	}
}

// builtinDelete deletes Map keys
// usage: delete(map, "key")
// key must be a string
func builtinDelete(vm core.VM, args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return nil, core.NewWrongNumArgumentsError("delete", "2", argsLen)
	}
	switch arg := args[0].(type) {
	case *value.Record:
		if arg.IsImmutable() {
			return nil, core.NewInvalidArgumentTypeError("delete", "first", "mutable record", arg)
		}
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return vm.Allocator().NewUndefined(), nil
		}
		return nil, core.NewInvalidArgumentTypeError("delete", "second", "string", args[1])
	case *value.Map:
		if arg.IsImmutable() {
			return nil, core.NewInvalidArgumentTypeError("delete", "first", "mutable record", arg)
		}
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return vm.Allocator().NewUndefined(), nil
		}
		return nil, core.NewInvalidArgumentTypeError("delete", "second", "string", args[1])
	default:
		return nil, core.NewInvalidArgumentTypeError("delete", "first", "record", arg)
	}
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage:
// deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(vm core.VM, args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return nil, core.NewWrongNumArgumentsError("splice", "at least 1", argsLen)
	}

	array, ok := args[0].(*value.Array)
	if !ok || array.IsImmutable() {
		return nil, core.NewInvalidArgumentTypeError("splice", "first", "mutable array", args[0])
	}

	arrayLen := int(array.Len())

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("splice", "second", "int", args[1])
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return nil, core.NewIndexOutOfBoundsError("splice, start index", startIdx, arrayLen)
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("splice", "third", "int", args[2])
		}
		delCount = int(arg2)
		if delCount < 0 {
			return nil, core.NewLogicError("splice delete count must be non-negative")
		}
	}
	// if count of to be deleted items is bigger than expected, truncate it
	if startIdx+delCount > arrayLen {
		delCount = arrayLen - startIdx
	}
	// delete items
	endIdx := startIdx + delCount
	deleted := append([]core.Object{}, array.Slice(startIdx, endIdx)...)

	head := array.Slice(0, startIdx)
	var items []core.Object
	if argsLen > 3 {
		items = make([]core.Object, 0, argsLen-3)
		for i := 3; i < argsLen; i++ {
			items = append(items, args[i])
		}
	}
	items = append(items, array.Slice(endIdx, array.Len())...)
	array.Set(append(head, items...), false)

	// return deleted items
	return vm.Allocator().NewArray(deleted, false), nil
}
