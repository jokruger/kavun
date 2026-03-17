package vm

import (
	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/formatter"
	"github.com/jokruger/gs/value"
)

var BuiltinFuncs = []*value.BuiltinFunction{
	value.NewBuiltinFunction("len", builtinLen, 1, false),
	value.NewBuiltinFunction("copy", builtinCopy, 1, false),
	value.NewBuiltinFunction("append", builtinAppend, 2, true),
	value.NewBuiltinFunction("delete", builtinDelete, 2, false),
	value.NewBuiltinFunction("splice", builtinSplice, 1, true),
	value.NewBuiltinFunction("string", builtinString, 1, true),
	value.NewBuiltinFunction("int", builtinInt, 1, true),
	value.NewBuiltinFunction("bool", builtinBool, 1, true),
	value.NewBuiltinFunction("float", builtinFloat, 1, true),
	value.NewBuiltinFunction("char", builtinChar, 1, true),
	value.NewBuiltinFunction("bytes", builtinBytes, 1, true),
	value.NewBuiltinFunction("time", builtinTime, 1, true),
	value.NewBuiltinFunction("is_int", builtinIsInt, 1, false),
	value.NewBuiltinFunction("is_float", builtinIsFloat, 1, false),
	value.NewBuiltinFunction("is_string", builtinIsString, 1, false),
	value.NewBuiltinFunction("is_bool", builtinIsBool, 1, false),
	value.NewBuiltinFunction("is_char", builtinIsChar, 1, false),
	value.NewBuiltinFunction("is_bytes", builtinIsBytes, 1, false),
	value.NewBuiltinFunction("is_array", builtinIsArray, 1, false),
	value.NewBuiltinFunction("is_immutable_array", builtinIsImmutableArray, 1, false),
	value.NewBuiltinFunction("is_map", builtinIsMap, 1, false),
	value.NewBuiltinFunction("is_immutable_map", builtinIsImmutableMap, 1, false),
	value.NewBuiltinFunction("is_iterable", builtinIsIterable, 1, false),
	value.NewBuiltinFunction("is_time", builtinIsTime, 1, false),
	value.NewBuiltinFunction("is_error", builtinIsError, 1, false),
	value.NewBuiltinFunction("is_undefined", builtinIsUndefined, 1, false),
	value.NewBuiltinFunction("is_function", builtinIsFunction, 1, false),
	value.NewBuiltinFunction("is_callable", builtinIsCallable, 1, false),
	value.NewBuiltinFunction("type_name", builtinTypeName, 1, false),
	value.NewBuiltinFunction("format", builtinFormat, 1, true),
	value.NewBuiltinFunction("range", builtinRange, 2, true),
}

// GetAllBuiltinFunctions returns all builtin function objects.
func GetAllBuiltinFunctions() []*value.BuiltinFunction {
	return append([]*value.BuiltinFunction{}, BuiltinFuncs...)
}

func builtinTypeName(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	return value.NewString(args[0].TypeName()), nil
}

func builtinIsString(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.String); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsInt(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Int); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsFloat(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Float); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsBool(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Bool); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsChar(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Char); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsBytes(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Bytes); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsArray(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Array); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsImmutableArray(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if !args[0].IsImmutable() {
		return value.FalseValue, nil
	}
	if _, ok := args[0].(*value.Array); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsMap(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Map); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsImmutableMap(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if !args[0].IsImmutable() {
		return value.FalseValue, nil
	}
	if _, ok := args[0].(*value.Map); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsTime(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Time); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsError(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Error); ok {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsUndefined(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if args[0] == value.UndefinedValue {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsFunction(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	switch args[0].(type) {
	case *CompiledFunction:
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsCallable(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if args[0].IsCallable() {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsIterable(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if args[0].IsIterable() {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

// len(obj object) => int
func builtinLen(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	switch arg := args[0].(type) {
	case *value.Array:
		return value.NewInt(int64(arg.Len())), nil
	case *value.String:
		return value.NewInt(int64(arg.Len())), nil
	case *value.Bytes:
		return value.NewInt(int64(arg.Len())), nil
	case *value.Map:
		return value.NewInt(int64(arg.Len())), nil
	default:
		return nil, &gse.InvalidArgumentTypeError{
			Name:     "first",
			Expected: "array/string/bytes/map",
			Found:    arg.TypeName(),
		}
	}
}

// range(start, stop[, step])
func builtinRange(args ...core.Object) (core.Object, error) {
	numArgs := len(args)
	if numArgs < 2 || numArgs > 3 {
		return nil, gse.ErrWrongNumArguments
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
			return nil, &gse.InvalidArgumentTypeError{Name: name, Expected: "int", Found: arg.TypeName()}
		}

		if i == 2 && v <= 0 {
			return nil, gse.ErrInvalidRangeStep
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

	return buildRange(start, stop, step), nil
}

func buildRange(start, stop, step int64) *value.Array {
	array := make([]core.Object, 0)
	if start <= stop {
		for i := start; i < stop; i += step {
			array = append(array, value.NewInt(i))
		}
	} else {
		for i := start; i > stop; i -= step {
			array = append(array, value.NewInt(i))
		}
	}
	return value.NewArray(array, false)
}

func builtinFormat(args ...core.Object) (core.Object, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, gse.ErrWrongNumArguments
	}
	format, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "format", Expected: "string", Found: args[0].TypeName()}
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return value.NewString(format), nil
	}
	s, err := formatter.Format(format, args[1:]...)
	if err != nil {
		return nil, err
	}
	return value.NewString(s), nil
}

func builtinCopy(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	return args[0].Copy(), nil
}

func builtinString(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.String); ok {
		return args[0], nil
	}
	v, ok := args[0].AsString()
	if ok {
		if len(v) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return value.NewString(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

func builtinInt(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Int); ok {
		return args[0], nil
	}
	v, ok := args[0].AsInt()
	if ok {
		return value.NewInt(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

func builtinFloat(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Float); ok {
		return args[0], nil
	}
	v, ok := args[0].AsFloat()
	if ok {
		return value.NewFloat(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

func builtinBool(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Bool); ok {
		return args[0], nil
	}
	v, ok := args[0].AsBool()
	if ok {
		if v {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}
	return value.UndefinedValue, nil
}

func builtinChar(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Char); ok {
		return args[0], nil
	}
	v, ok := args[0].AsRune()
	if ok {
		return value.NewChar(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

func builtinBytes(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}

	// bytes(N) => create a new bytes with given size N
	if n, ok := args[0].(*value.Int); ok {
		if n.Value() > int64(core.MaxBytesLen) {
			return nil, gse.ErrBytesLimit
		}
		return value.NewBytes(make([]byte, int(n.Value()))), nil
	}
	v, ok := args[0].AsByteSlice()
	if ok {
		if len(v) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return value.NewBytes(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

func builtinTime(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, gse.ErrWrongNumArguments
	}
	if _, ok := args[0].(*value.Time); ok {
		return args[0], nil
	}
	v, ok := args[0].AsTime()
	if ok {
		return value.NewTime(v), nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return value.UndefinedValue, nil
}

// append(arr, items...)
func builtinAppend(args ...core.Object) (core.Object, error) {
	if len(args) < 2 {
		return nil, gse.ErrWrongNumArguments
	}
	switch arg := args[0].(type) {
	case *value.Array:
		return value.NewArray(append(arg.Value(), args[1:]...), false), nil
	default:
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "array", Found: arg.TypeName()}
	}
}

// builtinDelete deletes Map keys
// usage: delete(map, "key")
// key must be a string
func builtinDelete(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if argsLen != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	switch arg := args[0].(type) {
	case *value.Map:
		if arg.IsImmutable() {
			return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "map", Found: arg.TypeName()}
		}
		if key, ok := args[1].AsString(); ok {
			arg.Delete(key)
			return value.UndefinedValue, nil
		}
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string", Found: args[1].TypeName()}
	default:
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "map", Found: arg.TypeName()}
	}
}

// builtinSplice deletes and changes given Array, returns deleted items.
// usage:
// deleted_items := splice(array[,start[,delete_count[,item1[,item2[,...]]]])
func builtinSplice(args ...core.Object) (core.Object, error) {
	argsLen := len(args)
	if argsLen == 0 {
		return nil, gse.ErrWrongNumArguments
	}

	array, ok := args[0].(*value.Array)
	if !ok || array.IsImmutable() {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "array", Found: args[0].TypeName()}
	}

	arrayLen := int(array.Len())

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].AsInt()
		if !ok {
			return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "int", Found: args[1].TypeName()}
		}
		startIdx = int(arg1)
		if startIdx < 0 || startIdx > arrayLen {
			return nil, gse.ErrIndexOutOfBounds
		}
	}

	delCount := arrayLen
	if argsLen > 2 {
		arg2, ok := args[2].AsInt()
		if !ok {
			return nil, &gse.InvalidArgumentTypeError{Name: "third", Expected: "int", Found: args[2].TypeName()}
		}
		delCount = int(arg2)
		if delCount < 0 {
			return nil, gse.ErrIndexOutOfBounds
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
	return value.NewArray(deleted, false), nil
}
