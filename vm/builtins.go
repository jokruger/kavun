package vm

import (
	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/formatter"
	"github.com/jokruger/gs/value"
)

var BuiltinFuncs = []*value.BuiltinFunction{
	{Name: "len", Value: builtinLen},
	{Name: "copy", Value: builtinCopy},
	{Name: "append", Value: builtinAppend},
	{Name: "delete", Value: builtinDelete},
	{Name: "splice", Value: builtinSplice},
	{Name: "string", Value: builtinString},
	{Name: "int", Value: builtinInt},
	{Name: "bool", Value: builtinBool},
	{Name: "float", Value: builtinFloat},
	{Name: "char", Value: builtinChar},
	{Name: "bytes", Value: builtinBytes},
	{Name: "time", Value: builtinTime},
	{Name: "is_int", Value: builtinIsInt},
	{Name: "is_float", Value: builtinIsFloat},
	{Name: "is_string", Value: builtinIsString},
	{Name: "is_bool", Value: builtinIsBool},
	{Name: "is_char", Value: builtinIsChar},
	{Name: "is_bytes", Value: builtinIsBytes},
	{Name: "is_array", Value: builtinIsArray},
	{Name: "is_immutable_array", Value: builtinIsImmutableArray},
	{Name: "is_map", Value: builtinIsMap},
	{Name: "is_immutable_map", Value: builtinIsImmutableMap},
	{Name: "is_iterable", Value: builtinIsIterable},
	{Name: "is_time", Value: builtinIsTime},
	{Name: "is_error", Value: builtinIsError},
	{Name: "is_undefined", Value: builtinIsUndefined},
	{Name: "is_function", Value: builtinIsFunction},
	{Name: "is_callable", Value: builtinIsCallable},
	{Name: "type_name", Value: builtinTypeName},
	{Name: "format", Value: builtinFormat},
	{Name: "range", Value: builtinRange},
}

// GetAllBuiltinFunctions returns all builtin function objects.
func GetAllBuiltinFunctions() []*value.BuiltinFunction {
	return append([]*value.BuiltinFunction{}, BuiltinFuncs...)
}

func builtinTypeName(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	return &value.String{Value: args[0].TypeName()}, nil
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
	if _, ok := args[0].(*value.ImmutableArray); ok {
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
	if _, ok := args[0].(*value.ImmutableMap); ok {
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
	case *value.CompiledFunction:
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsCallable(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if args[0].CanCall() {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func builtinIsIterable(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	if args[0].CanIterate() {
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
		return &value.Int{Value: int64(len(arg.Value))}, nil
	case *value.ImmutableArray:
		return &value.Int{Value: int64(len(arg.Value))}, nil
	case *value.String:
		return &value.Int{Value: int64(len(arg.Value))}, nil
	case *value.Bytes:
		return &value.Int{Value: int64(len(arg.Value))}, nil
	case *value.Map:
		return &value.Int{Value: int64(len(arg.Value))}, nil
	case *value.ImmutableMap:
		return &value.Int{Value: int64(len(arg.Value))}, nil
	default:
		return nil, gse.ErrInvalidArgumentType{
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
	var start, stop, step *value.Int

	for i, arg := range args {
		v, ok := args[i].(*value.Int)
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

			return nil, gse.ErrInvalidArgumentType{
				Name:     name,
				Expected: "int",
				Found:    arg.TypeName(),
			}
		}
		if i == 2 && v.Value <= 0 {
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

	if step == nil {
		step = &value.Int{Value: int64(1)}
	}

	return buildRange(start.Value, stop.Value, step.Value), nil
}

func buildRange(start, stop, step int64) *value.Array {
	array := &value.Array{}
	if start <= stop {
		for i := start; i < stop; i += step {
			array.Value = append(array.Value, &value.Int{
				Value: i,
			})
		}
	} else {
		for i := start; i > stop; i -= step {
			array.Value = append(array.Value, &value.Int{
				Value: i,
			})
		}
	}
	return array
}

func builtinFormat(args ...core.Object) (core.Object, error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, gse.ErrWrongNumArguments
	}
	format, ok := args[0].(*value.String)
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return format, nil
	}
	s, err := formatter.Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}
	return &value.String{Value: s}, nil
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
	v, ok := args[0].ToString()
	if ok {
		if len(v) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: v}, nil
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
	v, ok := args[0].ToInt64()
	if ok {
		return &value.Int{Value: v}, nil
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
	v, ok := args[0].ToFloat64()
	if ok {
		return &value.Float{Value: v}, nil
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
	v, ok := args[0].ToBool()
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
	v, ok := args[0].ToRune()
	if ok {
		return &value.Char{Value: v}, nil
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
		if n.Value > int64(core.MaxBytesLen) {
			return nil, gse.ErrBytesLimit
		}
		return &value.Bytes{Value: make([]byte, int(n.Value))}, nil
	}
	v, ok := args[0].ToByteSlice()
	if ok {
		if len(v) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return &value.Bytes{Value: v}, nil
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
	v, ok := args[0].ToTime()
	if ok {
		return &value.Time{Value: v}, nil
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
		return &value.Array{Value: append(arg.Value, args[1:]...)}, nil
	case *value.ImmutableArray:
		return &value.Array{Value: append(arg.Value, args[1:]...)}, nil
	default:
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array",
			Found:    arg.TypeName(),
		}
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
		if key, ok := args[1].(*value.String); ok {
			delete(arg.Value, key.Value)
			return value.UndefinedValue, nil
		}
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string",
			Found:    args[1].TypeName(),
		}
	default:
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "map",
			Found:    arg.TypeName(),
		}
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
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array",
			Found:    args[0].TypeName(),
		}
	}
	arrayLen := len(array.Value)

	var startIdx int
	if argsLen > 1 {
		arg1, ok := args[1].(*value.Int)
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "second",
				Expected: "int",
				Found:    args[1].TypeName(),
			}
		}
		startIdx = int(arg1.Value)
		if startIdx < 0 || startIdx > arrayLen {
			return nil, gse.ErrIndexOutOfBounds
		}
	}

	delCount := len(array.Value)
	if argsLen > 2 {
		arg2, ok := args[2].(*value.Int)
		if !ok {
			return nil, gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int",
				Found:    args[2].TypeName(),
			}
		}
		delCount = int(arg2.Value)
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
	deleted := append([]core.Object{}, array.Value[startIdx:endIdx]...)

	head := array.Value[:startIdx]
	var items []core.Object
	if argsLen > 3 {
		items = make([]core.Object, 0, argsLen-3)
		for i := 3; i < argsLen; i++ {
			items = append(items, args[i])
		}
	}
	items = append(items, array.Value[endIdx:]...)
	array.Value = append(head, items...)

	// return deleted items
	return &value.Array{Value: deleted}, nil
}
