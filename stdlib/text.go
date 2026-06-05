package stdlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func init() {
	// 47..127 reserved
	InitModule("text", core.BI_MOD_TEXT, nil, nil, map[uint64]*core.BuiltinFunction{
		0:  core.NewBuiltinFunction("re_match", textREMatch, 2, false),               // re_match(pattern, text) => bool/error
		1:  core.NewBuiltinFunction("re_find", textREFind, 2, true),                  // re_find(pattern, text [,count]) => [[{text:,begin:,end:}]]/undefined
		2:  core.NewBuiltinFunction("re_replace", textREReplace, 3, false),           // re_replace(pattern, text, repl) => string/error
		3:  core.NewBuiltinFunction("re_split", textRESplit, 2, true),                // re_split(pattern, text [,count]) => [string]/error
		4:  core.NewBuiltinFunction("re_compile", textRECompile, 1, false),           // re_compile(pattern) => Regexp/error
		5:  core.NewBuiltinFunction("compare", stringsCompare, 2, false),             // compare(a, b) => int
		6:  core.NewBuiltinFunction("contains", textContains, 2, false),              // contains(s, substr) => bool
		7:  core.NewBuiltinFunction("contains_any", textContainsAny, 2, false),       // contains_any(s, chars) => bool
		8:  core.NewBuiltinFunction("count", stringsCount, 2, false),                 // count(s, substr) => int
		9:  core.NewBuiltinFunction("equal_fold", textEqualFold, 2, false),           // "equal_fold(s, t) => bool
		10: core.NewBuiltinFunction("fields", stringsFields, 1, false),               // fields(s) => [string]
		11: core.NewBuiltinFunction("has_prefix", textHasPrefix, 2, false),           // has_prefix(s, prefix) => bool
		12: core.NewBuiltinFunction("has_suffix", textHasSuffix, 2, false),           // has_suffix(s, suffix) => bool
		13: core.NewBuiltinFunction("index", stringsIndex, 2, false),                 // index(s, substr) => int
		14: core.NewBuiltinFunction("index_any", stringsIndexAny, 2, false),          // index_any(s, chars) => int
		15: core.NewBuiltinFunction("join", textJoin, 2, false),                      // join(arr, sep) => string
		16: core.NewBuiltinFunction("last_index", stringsLastIndex, 2, false),        // last_index(s, substr) => int
		17: core.NewBuiltinFunction("last_index_any", stringsLastIndexAny, 2, false), // last_index_any(s, chars) => int
		18: core.NewBuiltinFunction("repeat", textRepeat, 2, false),                  // repeat(s, count) => string
		19: core.NewBuiltinFunction("replace", textReplace, 4, false),                // replace(s, old, new, n) => string
		20: core.NewBuiltinFunction("substr", textSubstring, 2, true),                // substr(s, lower [,upper]) => string
		21: core.NewBuiltinFunction("split", stringsSplit, 2, false),                 // split(s, sep) => [string]
		22: core.NewBuiltinFunction("split_after", stringsSplitAfter, 2, false),      // split_after(s, sep) => [string]
		23: core.NewBuiltinFunction("split_after_n", stringsSplitAfterN, 3, false),   // split_after_n(s, sep, n) => [string]
		24: core.NewBuiltinFunction("split_n", stringsSplitN, 3, false),              // split_n(s, sep, n) => [string]
		25: core.NewBuiltinFunction("title", stringsTitle, 1, false),                 // title(s) => string
		26: core.NewBuiltinFunction("to_lower", stringsToLower, 1, false),            // to_lower(s) => string
		27: core.NewBuiltinFunction("to_title", stringsToTitle, 1, false),            // to_title(s) => string
		28: core.NewBuiltinFunction("to_upper", stringsToUpper, 1, false),            // to_upper(s) => string
		29: core.NewBuiltinFunction("pad_left", textPadLeft, 2, true),                // pad_left(s, pad_len [,pad_with]) => string
		30: core.NewBuiltinFunction("pad_right", textPadRight, 2, true),              // pad_right(s, pad_len [,pad_with]) => string
		31: core.NewBuiltinFunction("trim", stringsTrim, 2, false),                   // trim(s, cutset) => string
		32: core.NewBuiltinFunction("trim_left", stringsTrimLeft, 2, false),          // trim_left(s, cutset) => string
		33: core.NewBuiltinFunction("trim_prefix", stringsTrimPrefix, 2, false),      // trim_prefix(s, prefix) => string
		34: core.NewBuiltinFunction("trim_right", stringsTrimRight, 2, false),        // trim_right(s, cutset) => string
		35: core.NewBuiltinFunction("trim_space", stringsTrimSpace, 1, false),        // trim_space(s) => string
		36: core.NewBuiltinFunction("trim_suffix", stringsTrimSuffix, 2, false),      // trim_suffix(s, suffix) => string
		37: core.NewBuiltinFunction("atoi", strconvAtoi, 1, false),                   // atoi(str) => int/error
		38: core.NewBuiltinFunction("format_bool", textFormatBool, 1, false),         // format_bool(b) => string
		39: core.NewBuiltinFunction("format_float", textFormatFloat, 4, false),       // format_float(f, fmt, prec, bits) => string
		40: core.NewBuiltinFunction("format_int", textFormatInt, 2, false),           // format_int(i, base) => string
		41: core.NewBuiltinFunction("itoa", strconvItoa, 1, false),                   // itoa(i) => string
		42: core.NewBuiltinFunction("parse_bool", textParseBool, 1, false),           // parse_bool(str) => bool/error
		43: core.NewBuiltinFunction("parse_float", textParseFloat, 2, false),         // parse_float(str, bits) => float/error
		44: core.NewBuiltinFunction("parse_int", textParseInt, 3, false),             // parse_int(str, base, bits) => int/error
		45: core.NewBuiltinFunction("quote", strconvQuote, 1, false),                 // quote(str) => string
		46: core.NewBuiltinFunction("unquote", strconvUnquote, 1, false),             // unquote(str) => string/error
	})
}

func strconvItoa(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.itoa", "1", len(args))
	}
	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.itoa", "first", "int(compatible)", args[0].TypeName(a))
	}
	s := strconv.Itoa(int(i1))
	return a.NewStringValue(s), nil
}

func strconvAtoi(a *core.Arena, vm core.VM, args []core.Value) (ret core.Value, err error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.atoi", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.atoi", "first", "string(compatible)", args[0].TypeName(a))
	}
	res, err := strconv.Atoi(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return core.IntValue(int64(res)), nil
}

func stringsTrimSuffix(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_suffix", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_suffix", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_suffix", "second", "string(compatible)", args[1].TypeName(a))
	}
	s := strings.TrimSuffix(s1, s2)
	return a.NewStringValue(s), nil
}

func stringsTrimRight(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_right", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_right", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_right", "second", "string(compatible)", args[1].TypeName(a))
	}
	s := strings.TrimRight(s1, s2)
	return a.NewStringValue(s), nil
}

func stringsTrimPrefix(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_prefix", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_prefix", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_prefix", "second", "string(compatible)", args[1].TypeName(a))
	}
	s := strings.TrimPrefix(s1, s2)
	return a.NewStringValue(s), nil
}

func stringsTrimLeft(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_left", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_left", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_left", "second", "string(compatible)", args[1].TypeName(a))
	}
	s := strings.TrimLeft(s1, s2)
	return a.NewStringValue(s), nil
}

func stringsTrim(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim", "second", "string(compatible)", args[1].TypeName(a))
	}
	s := strings.Trim(s1, s2)
	return a.NewStringValue(s), nil
}

func stringsLastIndexAny(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.last_index_any", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index_any", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index_any", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.LastIndexAny(s1, s2))), nil
}

func stringsLastIndex(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.last_index", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.last_index", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.LastIndex(s1, s2))), nil
}

func stringsIndexAny(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.index_any", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index_any", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index_any", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.IndexAny(s1, s2))), nil
}

func stringsIndex(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.index", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.index", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.Index(s1, s2))), nil
}

func stringsCount(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.count", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.count", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.count", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.Count(s1, s2))), nil
}

func stringsCompare(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.compare", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.compare", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.compare", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.IntValue(int64(strings.Compare(s1, s2))), nil
}

func stringsSplitN(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_n", "3", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "second", "string(compatible)", args[1].TypeName(a))
	}
	i3, ok := args[2].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_n", "third", "int(compatible)", args[2].TypeName(a))
	}
	spl := strings.SplitN(s1, s2, int(i3))
	arr := a.NewArray(len(spl), false)
	for _, res := range spl {
		t := a.NewStringValue(res)
		arr = append(arr, t)
	}
	return a.NewArrayValue(arr, false), nil
}

func stringsSplitAfterN(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_after_n", "3", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "second", "string(compatible)", args[1].TypeName(a))
	}
	i3, ok := args[2].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after_n", "third", "int(compatible)", args[2].TypeName(a))
	}
	spl := strings.SplitAfterN(s1, s2, int(i3))
	arr := a.NewArray(len(spl), false)
	for _, res := range spl {
		t := a.NewStringValue(res)
		arr = append(arr, t)
	}
	return a.NewArrayValue(arr, false), nil
}

func stringsSplitAfter(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split_after", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split_after", "second", "string(compatible)", args[1].TypeName(a))
	}
	spl := strings.SplitAfter(s1, s2)
	arr := a.NewArray(len(spl), false)
	for _, res := range spl {
		t := a.NewStringValue(res)
		arr = append(arr, t)
	}
	return a.NewArrayValue(arr, false), nil
}

func stringsSplit(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.split", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.split", "second", "string(compatible)", args[1].TypeName(a))
	}
	spl := strings.Split(s1, s2)
	arr := a.NewArray(len(spl), false)
	for _, res := range spl {
		t := a.NewStringValue(res)
		arr = append(arr, t)
	}
	return a.NewArrayValue(arr, false), nil
}

func strconvUnquote(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.unquote", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.unquote", "first", "string(compatible)", args[0].TypeName(a))
	}
	res, err := strconv.Unquote(s1)
	if err != nil {
		return wrapError(a, err)
	}
	return a.NewStringValue(res), nil
}

func stringsFields(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.fields", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.fields", "first", "string(compatible)", args[0].TypeName(a))
	}
	res := strings.Fields(s1)
	arr := a.NewArray(len(res), false)
	for _, elem := range res {
		t := a.NewStringValue(elem)
		arr = append(arr, t)
	}
	return a.NewArrayValue(arr, false), nil
}

func strconvQuote(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.quote", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.quote", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strconv.Quote(s1)
	return a.NewStringValue(s), nil
}

func stringsTrimSpace(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.trim_space", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.trim_space", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strings.TrimSpace(s1)
	return a.NewStringValue(s), nil
}

func stringsToTitle(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_title", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_title", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strings.ToTitle(s1)
	return a.NewStringValue(s), nil
}

func stringsToUpper(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_upper", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_upper", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strings.ToUpper(s1)
	return a.NewStringValue(s), nil
}

func stringsToLower(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.to_lower", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.to_lower", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strings.ToLower(s1)
	return a.NewStringValue(s), nil
}

func stringsTitle(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.title", "1", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.title", "first", "string(compatible)", args[0].TypeName(a))
	}
	s := strings.Title(s1)
	return a.NewStringValue(s), nil
}

func textREMatch(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_match", "2", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_match", "first", "string(compatible)", args[0].TypeName(a))
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_match", "second", "string(compatible)", args[1].TypeName(a))
	}

	matched, err := regexp.MatchString(s1, s2)
	if err != nil {
		return wrapError(a, err)
	}

	return core.BoolValue(matched), nil
}

func textREFind(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_find", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "first", "string(compatible)", args[0].TypeName(a))
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(a, err)
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "second", "string(compatible)", args[1].TypeName(a))
	}

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			return core.Undefined, nil
		}

		arr := a.NewArray(len(m)/2, false)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				txt := a.NewStringValue(s2[m[i]:m[i+1]])
				t := a.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				arr = append(arr, t)
			}
		}

		t := a.NewArrayValue(arr, false)
		return a.NewArrayValue([]core.Value{t}, false), nil
	}

	i3, ok := args[2].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_find", "third", "int(compatible)", args[2].TypeName(a))
	}
	m := re.FindAllStringSubmatchIndex(s2, int(i3))
	if m == nil {
		return core.Undefined, nil
	}

	arr := a.NewArray(len(m), false)
	for _, m := range m {
		subMatch := a.NewArray(len(m)/2, false)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				txt := a.NewStringValue(s2[m[i]:m[i+1]])
				t := a.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, true)
				subMatch = append(subMatch, t)
			}
		}
		t := a.NewArrayValue(subMatch, false)
		arr = append(arr, t)
	}

	return a.NewArrayValue(arr, false), nil
}

func textREReplace(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_replace", "3", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "first", "string(compatible)", args[0].TypeName(a))
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "second", "string(compatible)", args[1].TypeName(a))
	}

	s3, ok := args[2].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_replace", "third", "string(compatible)", args[2].TypeName(a))
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(a, err)
	}

	s, ok := doTextRegexpReplace(re, s2, s3)
	if !ok {
		return core.Undefined, errs.NewResourceLimitError("text.re_replace")
	}

	return a.NewStringValue(s), nil
}

func textRESplit(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_split", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "first", "string(compatible)", args[0].TypeName(a))
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "second", "string(compatible)", args[1].TypeName(a))
	}

	var i3 = -1
	if numArgs > 2 {
		var i3t int64
		i3t, ok = args[2].AsInt(a)
		i3 = int(i3t)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_split", "third", "int(compatible)", args[2].TypeName(a))
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(a, err)
	}

	spl := re.Split(s2, i3)
	arr := a.NewArray(len(spl), false)
	for _, s := range spl {
		t := a.NewStringValue(s)
		arr = append(arr, t)
	}

	return a.NewArrayValue(arr, false), nil
}

func textRECompile(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.re_compile", "1", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.re_compile", "first", "string(compatible)", args[0].TypeName(a))
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return wrapError(a, err)
	}

	return makeTextRegexp(a, vm, re)
}

func textReplace(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.replace", "4", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "first", "string(compatible)", args[0].TypeName(a))
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "second", "string(compatible)", args[1].TypeName(a))
	}

	s3, ok := args[2].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "third", "string(compatible)", args[2].TypeName(a))
	}

	i4, ok := args[3].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.replace", "fourth", "int(compatible)", args[3].TypeName(a))
	}

	s, ok := doTextReplace(s1, s2, s3, int(i4))
	if !ok {
		return core.Undefined, errs.NewResourceLimitError("text.replace")
	}

	return a.NewStringValue(s), nil
}

func textSubstring(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.substr", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "first", "string(compatible)", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "second", "int(compatible)", args[1].TypeName(a))
	}

	strlen := len(s1)
	i3 := strlen
	if argslen == 3 {
		var i3t int64
		i3t, ok = args[2].AsInt(a)
		i3 = int(i3t)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.substr", "third", "int(compatible)", args[2].TypeName(a))
		}
	}

	if int(i2) > i3 {
		return core.Undefined, errs.NewInvalidValueError("text.substring expected second argument to be less than or equal to third argument")
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

	return a.NewStringValue(s1[i2:i3]), nil
}

func textPadLeft(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.pad_left", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "first", "string(compatible)", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "second", "int(compatible)", args[1].TypeName(a))
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		return a.NewStringValue(s1), nil
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_left", "third", "string(compatible)", args[2].TypeName(a))
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		return a.NewStringValue(s1), nil
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := strings.Repeat(s3, padCount) + s1
	return a.NewStringValue(retStr[len(retStr)-int(i2):]), nil
}

func textPadRight(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.pad_right", "2 or 3", argslen)
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "first", "string(compatible)", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "second", "int(compatible)", args[1].TypeName(a))
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		return a.NewStringValue(s1), nil
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.pad_right", "third", "string(compatible)", args[2].TypeName(a))
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		return a.NewStringValue(s1), nil
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := s1 + strings.Repeat(s3, padCount)
	return a.NewStringValue(retStr[:i2]), nil
}

func textRepeat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.repeat", "2", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.repeat", "first", "string(compatible)", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.repeat", "second", "int(compatible)", args[1].TypeName(a))
	}

	return a.NewStringValue(strings.Repeat(s1, int(i2))), nil
}

func textJoin(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.join", "2", len(args))
	}
	if args[0].Type != core.VT_ARRAY {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", "first", "array", args[0].TypeName(a))
	}
	arr := (*core.Array)(args[0].Ptr)
	val := arr.Elements
	ss1 := make([]string, 0, len(val))
	var slen int
	for idx, e := range val {
		as, ok := e.AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", fmt.Sprintf("first[%d]", idx), "string(compatible)", e.TypeName(a))
		}
		slen += len(as)
		ss1 = append(ss1, as)
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.join", "second", "string(compatible)", args[1].TypeName(a))
	}

	return a.NewStringValue(strings.Join(ss1, s2)), nil
}

func textFormatBool(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_bool", "1", len(args))
	}

	b, ok := args[0].AsBool(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_bool", "first", "bool", args[0].TypeName(a))
	}

	var s string
	if b {
		s = "true"
	} else {
		s = "false"
	}

	return a.NewStringValue(s), nil
}

func textFormatFloat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 4 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_float", "4", len(args))
	}

	f1, ok := args[0].AsFloat(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "first", "float", args[0].TypeName(a))
	}

	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "second", "string(compatible)", args[1].TypeName(a))
	}

	i3, ok := args[2].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "third", "int(compatible)", args[2].TypeName(a))
	}

	i4, ok := args[3].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_float", "fourth", "int(compatible)", args[3].TypeName(a))
	}

	return a.NewStringValue(strconv.FormatFloat(f1, s2[0], int(i3), int(i4))), nil
}

func textFormatInt(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.format_int", "2", len(args))
	}

	i1, ok := args[0].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_int", "first", "int", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.format_int", "second", "int(compatible)", args[1].TypeName(a))
	}

	return a.NewStringValue(strconv.FormatInt(i1, int(i2))), nil
}

func textParseBool(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_bool", "1", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_bool", "first", "string", args[0].TypeName(a))
	}

	parsed, err := strconv.ParseBool(s1)
	if err != nil {
		return wrapError(a, err)
	}

	return core.BoolValue(parsed), nil
}

func textParseFloat(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_float", "2", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_float", "first", "string", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_float", "second", "int(compatible)", args[1].TypeName(a))
	}

	parsed, err := strconv.ParseFloat(s1, int(i2))
	if err != nil {
		return wrapError(a, err)
	}

	return core.FloatValue(parsed), nil
}

func textParseInt(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.parse_int", "3", len(args))
	}

	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "first", "string", args[0].TypeName(a))
	}

	i2, ok := args[1].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "second", "int(compatible)", args[1].TypeName(a))
	}

	i3, ok := args[2].AsInt(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.parse_int", "third", "int(compatible)", args[2].TypeName(a))
	}

	parsed, err := strconv.ParseInt(s1, int(i2), int(i3))
	if err != nil {
		return wrapError(a, err)
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
		w += copy(t[w:], ssj)
		w += copy(t[w:], new)
		start = j + len(old)
	}

	ss := s[start:]
	w += copy(t[w:], ss)

	return string(t[0:w]), true
}

func textContains(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.contains", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.BoolValue(strings.Contains(s1, s2)), nil
}

func textContainsAny(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.contains_any", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains_any", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.contains_any", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.BoolValue(strings.ContainsAny(s1, s2)), nil
}

func textEqualFold(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.equal_fold", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.equal_fold", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.equal_fold", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.BoolValue(strings.EqualFold(s1, s2)), nil
}

func textHasPrefix(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.has_prefix", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_prefix", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_prefix", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.BoolValue(strings.HasPrefix(s1, s2)), nil
}

func textHasSuffix(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("text.has_suffix", "2", len(args))
	}
	s1, ok := args[0].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_suffix", "first", "string(compatible)", args[0].TypeName(a))
	}
	s2, ok := args[1].AsString(a)
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("text.has_suffix", "second", "string(compatible)", args[1].TypeName(a))
	}
	return core.BoolValue(strings.HasSuffix(s1, s2)), nil
}
