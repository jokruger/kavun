package stdlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

var textModule = map[string]core.Value{
	"re_match":       core.NewBuiltinFunctionValue("re_match", textREMatch, 2, false),               // re_match(pattern, text) => bool/error
	"re_find":        core.NewBuiltinFunctionValue("re_find", textREFind, 2, true),                  // re_find(pattern, text [,count]) => [[{text:,begin:,end:}]]/undefined
	"re_replace":     core.NewBuiltinFunctionValue("re_replace", textREReplace, 3, false),           // re_replace(pattern, text, repl) => string/error
	"re_split":       core.NewBuiltinFunctionValue("re_split", textRESplit, 2, true),                // re_split(pattern, text [,count]) => [string]/error
	"re_compile":     core.NewBuiltinFunctionValue("re_compile", textRECompile, 1, false),           // re_compile(pattern) => Regexp/error
	"compare":        core.NewBuiltinFunctionValue("compare", stringsCompare, 2, false),             // compare(a, b) => int
	"contains":       core.NewBuiltinFunctionValue("contains", textContains, 2, false),              // contains(s, substr) => bool
	"contains_any":   core.NewBuiltinFunctionValue("contains_any", textContainsAny, 2, false),       // contains_any(s, chars) => bool
	"count":          core.NewBuiltinFunctionValue("count", stringsCount, 2, false),                 // count(s, substr) => int
	"equal_fold":     core.NewBuiltinFunctionValue("equal_fold", textEqualFold, 2, false),           // "equal_fold(s, t) => bool
	"fields":         core.NewBuiltinFunctionValue("fields", stringsFields, 1, false),               // fields(s) => [string]
	"has_prefix":     core.NewBuiltinFunctionValue("has_prefix", textHasPrefix, 2, false),           // has_prefix(s, prefix) => bool
	"has_suffix":     core.NewBuiltinFunctionValue("has_suffix", textHasSuffix, 2, false),           // has_suffix(s, suffix) => bool
	"index":          core.NewBuiltinFunctionValue("index", stringsIndex, 2, false),                 // index(s, substr) => int
	"index_any":      core.NewBuiltinFunctionValue("index_any", stringsIndexAny, 2, false),          // index_any(s, chars) => int
	"join":           core.NewBuiltinFunctionValue("join", textJoin, 2, false),                      // join(arr, sep) => string
	"last_index":     core.NewBuiltinFunctionValue("last_index", stringsLastIndex, 2, false),        // last_index(s, substr) => int
	"last_index_any": core.NewBuiltinFunctionValue("last_index_any", stringsLastIndexAny, 2, false), // last_index_any(s, chars) => int
	"repeat":         core.NewBuiltinFunctionValue("repeat", textRepeat, 2, false),                  // repeat(s, count) => string
	"replace":        core.NewBuiltinFunctionValue("replace", textReplace, 4, false),                // replace(s, old, new, n) => string
	"substr":         core.NewBuiltinFunctionValue("substr", textSubstring, 2, true),                // substr(s, lower [,upper]) => string
	"split":          core.NewBuiltinFunctionValue("split", stringsSplit, 2, false),                 // split(s, sep) => [string]
	"split_after":    core.NewBuiltinFunctionValue("split_after", stringsSplitAfter, 2, false),      // split_after(s, sep) => [string]
	"split_after_n":  core.NewBuiltinFunctionValue("split_after_n", stringsSplitAfterN, 3, false),   // split_after_n(s, sep, n) => [string]
	"split_n":        core.NewBuiltinFunctionValue("split_n", stringsSplitN, 3, false),              // split_n(s, sep, n) => [string]
	"title":          core.NewBuiltinFunctionValue("title", stringsTitle, 1, false),                 // title(s) => string
	"to_lower":       core.NewBuiltinFunctionValue("to_lower", stringsToLower, 1, false),            // to_lower(s) => string
	"to_title":       core.NewBuiltinFunctionValue("to_title", stringsToTitle, 1, false),            // to_title(s) => string
	"to_upper":       core.NewBuiltinFunctionValue("to_upper", stringsToUpper, 1, false),            // to_upper(s) => string
	"pad_left":       core.NewBuiltinFunctionValue("pad_left", textPadLeft, 2, true),                // pad_left(s, pad_len [,pad_with]) => string
	"pad_right":      core.NewBuiltinFunctionValue("pad_right", textPadRight, 2, true),              // pad_right(s, pad_len [,pad_with]) => string
	"trim":           core.NewBuiltinFunctionValue("trim", stringsTrim, 2, false),                   // trim(s, cutset) => string
	"trim_left":      core.NewBuiltinFunctionValue("trim_left", stringsTrimLeft, 2, false),          // trim_left(s, cutset) => string
	"trim_prefix":    core.NewBuiltinFunctionValue("trim_prefix", stringsTrimPrefix, 2, false),      // trim_prefix(s, prefix) => string
	"trim_right":     core.NewBuiltinFunctionValue("trim_right", stringsTrimRight, 2, false),        // trim_right(s, cutset) => string
	"trim_space":     core.NewBuiltinFunctionValue("trim_space", stringsTrimSpace, 1, false),        // trim_space(s) => string
	"trim_suffix":    core.NewBuiltinFunctionValue("trim_suffix", stringsTrimSuffix, 2, false),      // trim_suffix(s, suffix) => string
	"atoi":           core.NewBuiltinFunctionValue("atoi", strconvAtoi, 1, false),                   // atoi(str) => int/error
	"format_bool":    core.NewBuiltinFunctionValue("format_bool", textFormatBool, 1, false),         // format_bool(b) => string
	"format_float":   core.NewBuiltinFunctionValue("format_float", textFormatFloat, 4, false),       // format_float(f, fmt, prec, bits) => string
	"format_int":     core.NewBuiltinFunctionValue("format_int", textFormatInt, 2, false),           // format_int(i, base) => string
	"itoa":           core.NewBuiltinFunctionValue("itoa", strconvItoa, 1, false),                   // itoa(i) => string
	"parse_bool":     core.NewBuiltinFunctionValue("parse_bool", textParseBool, 1, false),           // parse_bool(str) => bool/error
	"parse_float":    core.NewBuiltinFunctionValue("parse_float", textParseFloat, 2, false),         // parse_float(str, bits) => float/error
	"parse_int":      core.NewBuiltinFunctionValue("parse_int", textParseInt, 3, false),             // parse_int(str, base, bits) => int/error
	"quote":          core.NewBuiltinFunctionValue("quote", strconvQuote, 1, false),                 // quote(str) => string
	"unquote":        core.NewBuiltinFunctionValue("unquote", strconvUnquote, 1, false),             // unquote(str) => string/error
}

func strconvItoa(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.itoa", "1", len(args))
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.itoa", "first", "int(compatible)", args[0].TypeName())
	}
	s := strconv.Itoa(int(i1))
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.itoa")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func strconvAtoi(vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.atoi", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.atoi", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := strconv.Atoi(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	return core.IntValue(int64(res)), nil
}

func stringsTrimSuffix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_suffix", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_suffix", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_suffix", "second", "string(compatible)", args[1].TypeName())
	}
	s := strings.TrimSuffix(s1, s2)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim_suffix")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTrimRight(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_right", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_right", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_right", "second", "string(compatible)", args[1].TypeName())
	}
	s := strings.TrimRight(s1, s2)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim_right")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTrimPrefix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_prefix", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_prefix", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_prefix", "second", "string(compatible)", args[1].TypeName())
	}
	s := strings.TrimPrefix(s1, s2)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim_prefix")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTrimLeft(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_left", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_left", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_left", "second", "string(compatible)", args[1].TypeName())
	}
	s := strings.TrimLeft(s1, s2)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim_left")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTrim(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim", "second", "string(compatible)", args[1].TypeName())
	}
	s := strings.Trim(s1, s2)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsLastIndexAny(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.last_index_any", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index_any", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index_any", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.LastIndexAny(s1, s2))), nil
}

func stringsLastIndex(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.last_index", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.LastIndex(s1, s2))), nil
}

func stringsIndexAny(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.index_any", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index_any", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index_any", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.IndexAny(s1, s2))), nil
}

func stringsIndex(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.index", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.Index(s1, s2))), nil
}

func stringsCount(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.count", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.count", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.count", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.Count(s1, s2))), nil
}

func stringsCompare(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.compare", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.compare", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.compare", "second", "string(compatible)", args[1].TypeName())
	}
	return core.IntValue(int64(strings.Compare(s1, s2))), nil
}

func stringsSplitN(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_n", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "second", "string(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "third", "int(compatible)", args[2].TypeName())
	}
	spl := strings.SplitN(s1, s2, int(i3))
	arr := make([]core.Value, 0, len(spl))
	alloc := vm.Allocator()
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("text.split_n")
		}
		arr = append(arr, alloc.NewStringValue(res))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func stringsSplitAfterN(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_after_n", "3", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "second", "string(compatible)", args[1].TypeName())
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "third", "int(compatible)", args[2].TypeName())
	}
	spl := strings.SplitAfterN(s1, s2, int(i3))
	arr := make([]core.Value, 0, len(spl))
	alloc := vm.Allocator()
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("text.split_after_n")
		}
		arr = append(arr, alloc.NewStringValue(res))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func stringsSplitAfter(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_after", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after", "second", "string(compatible)", args[1].TypeName())
	}
	spl := strings.SplitAfter(s1, s2)
	arr := make([]core.Value, 0, len(spl))
	alloc := vm.Allocator()
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("text.split_after")
		}
		arr = append(arr, alloc.NewStringValue(res))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func stringsSplit(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split", "second", "string(compatible)", args[1].TypeName())
	}
	spl := strings.Split(s1, s2)
	arr := make([]core.Value, 0, len(spl))
	alloc := vm.Allocator()
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("text.split")
		}
		arr = append(arr, alloc.NewStringValue(res))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func strconvUnquote(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.unquote", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.unquote", "first", "string(compatible)", args[0].TypeName())
	}
	res, err := strconv.Unquote(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}
	if len(res) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.unquote")
	}
	return vm.Allocator().NewStringValue(res), nil
}

func stringsFields(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.fields", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.fields", "first", "string(compatible)", args[0].TypeName())
	}
	res := strings.Fields(s1)
	arr := make([]core.Value, 0, len(res))
	alloc := vm.Allocator()
	for _, elem := range res {
		if len(elem) > core.MaxStringLen {
			return core.Undefined, errs.NewStringLimitError("text.fields")
		}
		arr = append(arr, alloc.NewStringValue(elem))
	}
	return alloc.NewArrayValue(arr, false), nil
}

func strconvQuote(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.quote", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.quote", "first", "string(compatible)", args[0].TypeName())
	}
	s := strconv.Quote(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.quote")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTrimSpace(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_space", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_space", "first", "string(compatible)", args[0].TypeName())
	}
	s := strings.TrimSpace(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.trim_space")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsToTitle(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_title", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_title", "first", "string(compatible)", args[0].TypeName())
	}
	s := strings.ToTitle(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.to_title")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsToUpper(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_upper", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_upper", "first", "string(compatible)", args[0].TypeName())
	}
	s := strings.ToUpper(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.to_upper")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsToLower(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_lower", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_lower", "first", "string(compatible)", args[0].TypeName())
	}
	s := strings.ToLower(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.to_lower")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func stringsTitle(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.title", "1", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.title", "first", "string(compatible)", args[0].TypeName())
	}
	s := strings.Title(s1)
	if len(s) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.title")
	}
	return vm.Allocator().NewStringValue(s), nil
}

func textREMatch(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_match", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_match", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_match", "second", "string(compatible)", args[1].TypeName())
	}

	matched, err := regexp.MatchString(s1, s2)
	if err != nil {
		return wrapError(vm, err), nil
	}

	return core.BoolValue(matched), nil
}

func textREFind(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_find", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "first", "string(compatible)", args[0].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "second", "string(compatible)", args[1].TypeName())
	}

	alloc := vm.Allocator()

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			return core.Undefined, nil
		}

		arr := make([]core.Value, 0, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				t := map[string]core.Value{
					"text":  alloc.NewStringValue(s2[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}
				arr = append(arr, alloc.NewRecordValue(t, false))
			}
		}

		return alloc.NewArrayValue([]core.Value{alloc.NewArrayValue(arr, false)}, false), nil
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "third", "int(compatible)", args[2].TypeName())
	}
	m := re.FindAllStringSubmatchIndex(s2, int(i3))
	if m == nil {
		return core.Undefined, nil
	}

	arr := make([]core.Value, 0, len(m))
	for _, m := range m {
		subMatch := make([]core.Value, 0, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				t := alloc.NewRecordValue(map[string]core.Value{
					"text":  alloc.NewStringValue(s2[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, true)
				subMatch = append(subMatch, t)
			}
		}
		arr = append(arr, alloc.NewArrayValue(subMatch, false))
	}

	return alloc.NewArrayValue(arr, false), nil
}

func textREReplace(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_replace", "3", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "second", "string(compatible)", args[1].TypeName())
	}

	s3, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "third", "string(compatible)", args[2].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	s, ok := doTextRegexpReplace(re, s2, s3)
	if !ok {
		return core.Undefined, errs.NewStringLimitError("text.re_replace")
	}

	return vm.Allocator().NewStringValue(s), nil
}

func textRESplit(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_split", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "second", "string(compatible)", args[1].TypeName())
	}

	var i3 = -1
	if numArgs > 2 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "third", "int(compatible)", args[2].TypeName())
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	spl := re.Split(s2, i3)
	arr := make([]core.Value, 0, len(spl))
	alloc := vm.Allocator()
	for _, s := range spl {
		arr = append(arr, alloc.NewStringValue(s))
	}

	return alloc.NewArrayValue(arr, false), nil
}

func textRECompile(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_compile", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_compile", "first", "string(compatible)", args[0].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	return makeTextRegexp(vm, re), nil
}

func textReplace(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.replace", "4", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "second", "string(compatible)", args[1].TypeName())
	}

	s3, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "third", "string(compatible)", args[2].TypeName())
	}

	i4, ok := args[3].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "fourth", "int(compatible)", args[3].TypeName())
	}

	s, ok := doTextReplace(s1, s2, s3, int(i4))
	if !ok {
		return core.Undefined, errs.NewStringLimitError("text.replace")
	}

	return vm.Allocator().NewStringValue(s), nil
}

func textSubstring(vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.substr", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "first", "string(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "second", "int(compatible)", args[1].TypeName())
	}

	strlen := len(s1)
	i3 := strlen
	if argslen == 3 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "third", "int(compatible)", args[2].TypeName())
		}
	}

	if int(i2) > i3 {
		return core.Undefined, errs.NewLogicError("text.substring expected second argument to be less than or equal to third argument")
	}

	if i2 < 0 {
		i2 = 0
	} else if int(i2) > strlen {
		i2 = int64(strlen)
	}

	if i3 < 0 {
		i3 = 0
	} else if i3 > strlen {
		i3 = strlen
	}

	return vm.Allocator().NewStringValue(s1[i2:i3]), nil
}

func textPadLeft(vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.pad_left", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "first", "string(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "second", "int(compatible)", args[1].TypeName())
	}

	if int(i2) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.pad_left")
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		return vm.Allocator().NewStringValue(s1), nil
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "third", "string(compatible)", args[2].TypeName())
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		return vm.Allocator().NewStringValue(s1), nil
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := strings.Repeat(s3, padCount) + s1
	return vm.Allocator().NewStringValue(retStr[len(retStr)-int(i2):]), nil
}

func textPadRight(vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.pad_right", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "first", "string(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "second", "int(compatible)", args[1].TypeName())
	}

	if int(i2) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.pad_right")
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		return vm.Allocator().NewStringValue(s1), nil
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "third", "string(compatible)", args[2].TypeName())
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		return vm.Allocator().NewStringValue(s1), nil
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := s1 + strings.Repeat(s3, padCount)
	return vm.Allocator().NewStringValue(retStr[:i2]), nil
}

func textRepeat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.repeat", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.repeat", "first", "string(compatible)", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.repeat", "second", "int(compatible)", args[1].TypeName())
	}

	if len(s1)*int(i2) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.repeat")
	}

	return vm.Allocator().NewStringValue(strings.Repeat(s1, int(i2))), nil
}

func textJoin(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.join", "2", len(args))
	}

	if args[0].Type != core.VT_ARRAY {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", "first", "array", args[0].TypeName())
	}
	arr := (*core.Array)(args[0].Ptr)
	val := arr.Elements
	ss1 := make([]string, 0, len(val))
	var slen int
	for idx, a := range val {
		as, ok := a.AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", fmt.Sprintf("first[%d]", idx), "string(compatible)", a.TypeName())
		}
		slen += len(as)
		ss1 = append(ss1, as)
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", "second", "string(compatible)", args[1].TypeName())
	}

	// make sure output length does not exceed the limit
	if slen+len(s2)*(len(ss1)-1) > core.MaxStringLen {
		return core.Undefined, errs.NewStringLimitError("text.join")
	}

	return vm.Allocator().NewStringValue(strings.Join(ss1, s2)), nil
}

func textFormatBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_bool", "1", len(args))
	}

	b, ok := args[0].AsBool()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_bool", "first", "bool", args[0].TypeName())
	}

	var s string
	if b {
		s = "true"
	} else {
		s = "false"
	}

	return vm.Allocator().NewStringValue(s), nil
}

func textFormatFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_float", "4", len(args))
	}

	f1, ok := args[0].AsFloat()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "first", "float", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "second", "string(compatible)", args[1].TypeName())
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "third", "int(compatible)", args[2].TypeName())
	}

	i4, ok := args[3].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "fourth", "int(compatible)", args[3].TypeName())
	}

	return vm.Allocator().NewStringValue(strconv.FormatFloat(f1, s2[0], int(i3), int(i4))), nil
}

func textFormatInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_int", "2", len(args))
	}

	i1, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_int", "first", "int", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_int", "second", "int(compatible)", args[1].TypeName())
	}

	return vm.Allocator().NewStringValue(strconv.FormatInt(i1, int(i2))), nil
}

func textParseBool(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_bool", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_bool", "first", "string", args[0].TypeName())
	}

	parsed, err := strconv.ParseBool(s1)
	if err != nil {
		return wrapError(vm, err), nil
	}

	return core.BoolValue(parsed), nil
}

func textParseFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_float", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_float", "first", "string", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_float", "second", "int(compatible)", args[1].TypeName())
	}

	parsed, err := strconv.ParseFloat(s1, int(i2))
	if err != nil {
		return wrapError(vm, err), nil
	}

	return core.FloatValue(parsed), nil
}

func textParseInt(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_int", "3", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "first", "string", args[0].TypeName())
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "second", "int(compatible)", args[1].TypeName())
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "third", "int(compatible)", args[2].TypeName())
	}

	parsed, err := strconv.ParseInt(s1, int(i2), int(i3))
	if err != nil {
		return wrapError(vm, err), nil
	}

	return core.IntValue(parsed), nil
}

// Modified implementation of strings.Replace
// to limit the maximum length of output string.
func doTextReplace(s, old, new string, n int) (string, bool) {
	if old == new || n == 0 {
		return s, true // avoid allocation
	}

	// Compute number of replacements.
	if m := strings.Count(s, old); m == 0 {
		return s, true // avoid allocation
	} else if n < 0 || m < n {
		n = m
	}

	// Apply replacements to buffer.
	t := make([]byte, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if len(old) == 0 {
			if i > 0 {
				_, wid := utf8.DecodeRuneInString(s[start:])
				j += wid
			}
		} else {
			j += strings.Index(s[start:], old)
		}

		ssj := s[start:j]
		if w+len(ssj)+len(new) > core.MaxStringLen {
			return "", false
		}

		w += copy(t[w:], ssj)
		w += copy(t[w:], new)
		start = j + len(old)
	}

	ss := s[start:]
	if w+len(ss) > core.MaxStringLen {
		return "", false
	}

	w += copy(t[w:], ss)

	return string(t[0:w]), true
}

func textContains(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.contains", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains", "second", "string(compatible)", args[1].TypeName())
	}
	return core.BoolValue(strings.Contains(s1, s2)), nil
}

func textContainsAny(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.contains_any", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains_any", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains_any", "second", "string(compatible)", args[1].TypeName())
	}
	return core.BoolValue(strings.ContainsAny(s1, s2)), nil
}

func textEqualFold(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.equal_fold", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.equal_fold", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.equal_fold", "second", "string(compatible)", args[1].TypeName())
	}
	return core.BoolValue(strings.EqualFold(s1, s2)), nil
}

func textHasPrefix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.has_prefix", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_prefix", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_prefix", "second", "string(compatible)", args[1].TypeName())
	}
	return core.BoolValue(strings.HasPrefix(s1, s2)), nil
}

func textHasSuffix(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.has_suffix", "2", len(args))
	}
	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_suffix", "first", "string(compatible)", args[0].TypeName())
	}
	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_suffix", "second", "string(compatible)", args[1].TypeName())
	}
	return core.BoolValue(strings.HasSuffix(s1, s2)), nil
}
