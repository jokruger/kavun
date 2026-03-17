package stdlib

import (
	"fmt"
	"strings"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var textModule = map[string]core.Object{
	/*
		"re_match": &value.BuiltinFunction{
			Name:  "re_match",
			Value: textREMatch,
		}, // re_match(pattern, text) => bool/error
		"re_find": &value.BuiltinFunction{
			Name:  "re_find",
			Value: textREFind,
		}, // re_find(pattern, text, count) => [[{text:,begin:,end:}]]/undefined
		"re_replace": &value.BuiltinFunction{
			Name:  "re_replace",
			Value: textREReplace,
		}, // re_replace(pattern, text, repl) => string/error
		"re_split": &value.BuiltinFunction{
			Name:  "re_split",
			Value: textRESplit,
		}, // re_split(pattern, text, count) => [string]/error
		"re_compile": &value.BuiltinFunction{
			Name:  "re_compile",
			Value: textRECompile,
		}, // re_compile(pattern) => Regexp/error
		"compare": &value.BuiltinFunction{
			Name:  "compare",
			Value: stringsCompare,
		}, // compare(a, b) => int
		"contains": &value.BuiltinFunction{
			Name:  "contains",
			Value: textContains,
		}, // contains(s, substr) => bool
		"contains_any": &value.BuiltinFunction{
			Name:  "contains_any",
			Value: textContainsAny,
		}, // contains_any(s, chars) => bool
		"count": &value.BuiltinFunction{
			Name:  "count",
			Value: stringsCount,
		}, // count(s, substr) => int
		"equal_fold": &value.BuiltinFunction{
			Name:  "equal_fold",
			Value: textEqualFold,
		}, // "equal_fold(s, t) => bool
		"fields": &value.BuiltinFunction{
			Name:  "fields",
			Value: stringsFields,
		}, // fields(s) => [string]
		"has_prefix": &value.BuiltinFunction{
			Name:  "has_prefix",
			Value: textHasPrefix,
		}, // has_prefix(s, prefix) => bool
		"has_suffix": &value.BuiltinFunction{
			Name:  "has_suffix",
			Value: textHasSuffix,
		}, // has_suffix(s, suffix) => bool
		"index": &value.BuiltinFunction{
			Name:  "index",
			Value: stringsIndex,
		}, // index(s, substr) => int
		"index_any": &value.BuiltinFunction{
			Name:  "index_any",
			Value: stringsIndexAny,
		}, // index_any(s, chars) => int
	*/
	"join": value.NewBuiltinFunction("join", textJoin, 2, false), // join(arr, sep) => string
	/*
		"last_index": &value.BuiltinFunction{
			Name:  "last_index",
			Value: stringsLastIndex,
		}, // last_index(s, substr) => int
		"last_index_any": &value.BuiltinFunction{
			Name:  "last_index_any",
			Value: stringsLastIndexAny,
		}, // last_index_any(s, chars) => int
		"repeat": &value.BuiltinFunction{
			Name:  "repeat",
			Value: textRepeat,
		}, // repeat(s, count) => string
		"replace": &value.BuiltinFunction{
			Name:  "replace",
			Value: textReplace,
		}, // replace(s, old, new, n) => string
		"substr": &value.BuiltinFunction{
			Name:  "substr",
			Value: textSubstring,
		}, // substr(s, lower, upper) => string
		"split": &value.BuiltinFunction{
			Name:  "split",
			Value: stringsSplit,
		}, // split(s, sep) => [string]
		"split_after": &value.BuiltinFunction{
			Name:  "split_after",
			Value: stringsSplitAfter,
		}, // split_after(s, sep) => [string]
		"split_after_n": &value.BuiltinFunction{
			Name:  "split_after_n",
			Value: stringsSplitAfterN,
		}, // split_after_n(s, sep, n) => [string]
		"split_n": &value.BuiltinFunction{
			Name:  "split_n",
			Value: stringsSplitN,
		}, // split_n(s, sep, n) => [string]
		"title": &value.BuiltinFunction{
			Name:  "title",
			Value: stringsTitle,
		}, // title(s) => string
		"to_lower": &value.BuiltinFunction{
			Name:  "to_lower",
			Value: stringsToLower,
		}, // to_lower(s) => string
		"to_title": &value.BuiltinFunction{
			Name:  "to_title",
			Value: stringsToTitle,
		}, // to_title(s) => string
		"to_upper": &value.BuiltinFunction{
			Name:  "to_upper",
			Value: stringsToUpper,
		}, // to_upper(s) => string
		"pad_left": &value.BuiltinFunction{
			Name:  "pad_left",
			Value: textPadLeft,
		}, // pad_left(s, pad_len, pad_with) => string
		"pad_right": &value.BuiltinFunction{
			Name:  "pad_right",
			Value: textPadRight,
		}, // pad_right(s, pad_len, pad_with) => string
		"trim": &value.BuiltinFunction{
			Name:  "trim",
			Value: stringsTrim,
		}, // trim(s, cutset) => string
		"trim_left": &value.BuiltinFunction{
			Name:  "trim_left",
			Value: stringsTrimLeft,
		}, // trim_left(s, cutset) => string
		"trim_prefix": &value.BuiltinFunction{
			Name:  "trim_prefix",
			Value: stringsTrimPrefix,
		}, // trim_prefix(s, prefix) => string
		"trim_right": &value.BuiltinFunction{
			Name:  "trim_right",
			Value: stringsTrimRight,
		}, // trim_right(s, cutset) => string
		"trim_space": &value.BuiltinFunction{
			Name:  "trim_space",
			Value: stringsTrimSpace,
		}, // trim_space(s) => string
		"trim_suffix": &value.BuiltinFunction{
			Name:  "trim_suffix",
			Value: stringsTrimSuffix,
		}, // trim_suffix(s, suffix) => string
		"atoi": &value.BuiltinFunction{
			Name:  "atoi",
			Value: strconvAtoi,
		}, // atoi(str) => int/error
		"format_bool": &value.BuiltinFunction{
			Name:  "format_bool",
			Value: textFormatBool,
		}, // format_bool(b) => string
		"format_float": &value.BuiltinFunction{
			Name:  "format_float",
			Value: textFormatFloat,
		}, // format_float(f, fmt, prec, bits) => string
		"format_int": &value.BuiltinFunction{
			Name:  "format_int",
			Value: textFormatInt,
		}, // format_int(i, base) => string
		"itoa": &value.BuiltinFunction{
			Name:  "itoa",
			Value: strconvItoa,
		}, // itoa(i) => string
		"parse_bool": &value.BuiltinFunction{
			Name:  "parse_bool",
			Value: textParseBool,
		}, // parse_bool(str) => bool/error
		"parse_float": &value.BuiltinFunction{
			Name:  "parse_float",
			Value: textParseFloat,
		}, // parse_float(str, bits) => float/error
		"parse_int": &value.BuiltinFunction{
			Name:  "parse_int",
			Value: textParseInt,
		}, // parse_int(str, base, bits) => int/error
		"quote": &value.BuiltinFunction{
			Name:  "quote",
			Value: strconvQuote,
		}, // quote(str) => string
		"unquote": &value.BuiltinFunction{
			Name:  "unquote",
			Value: strconvUnquote,
		}, // unquote(str) => string/error
	*/
}

/*
func strconvItoa(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strconv.Itoa(int(i1))
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func strconvAtoi(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := strconv.Atoi(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return &value.Int{Value: int64(res)}, nil
}

func stringsTrimSuffix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	s := strings.TrimSuffix(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTrimRight(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	s := strings.TrimRight(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTrimPrefix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	s := strings.TrimPrefix(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTrimLeft(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	s := strings.TrimLeft(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTrim(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	s := strings.Trim(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsLastIndexAny(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.LastIndexAny(s1, s2))}, nil
}

func stringsLastIndex(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.LastIndex(s1, s2))}, nil
}

func stringsIndexAny(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.IndexAny(s1, s2))}, nil
}

func stringsIndex(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.Index(s1, s2))}, nil
}

func stringsCount(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.Count(s1, s2))}, nil
}

func stringsCompare(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return &value.Int{Value: int64(strings.Compare(s1, s2))}, nil
}

func stringsSplitN(args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	arr := &value.Array{}
	for _, res := range strings.SplitN(s1, s2, int(i3)) {
		if len(res) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, value.NewString(res))
	}
	return arr, nil
}

func stringsSplitAfterN(args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
	}
	arr := &value.Array{}
	for _, res := range strings.SplitAfterN(s1, s2, int(i3)) {
		if len(res) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, value.NewString(res))
	}
	return arr, nil
}

func stringsSplitAfter(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	arr := &value.Array{}
	for _, res := range strings.SplitAfter(s1, s2) {
		if len(res) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, value.NewString(res))
	}
	return arr, nil
}

func stringsSplit(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	arr := &value.Array{}
	for _, res := range strings.Split(s1, s2) {
		if len(res) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, value.NewString(res))
	}
	return arr, nil
}

func strconvUnquote(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res, err := strconv.Unquote(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(res), nil
}

func stringsFields(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	res := strings.Fields(s1)
	arr := &value.Array{}
	for _, elem := range res {
		if len(elem) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		arr.Value = append(arr.Value, &value.String{Value: elem})
	}
	return arr, nil
}

func strconvQuote(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strconv.Quote(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTrimSpace(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strings.TrimSpace(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsToTitle(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strings.ToTitle(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsToUpper(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strings.ToUpper(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsToLower(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strings.ToLower(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func stringsTitle(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s := strings.Title(s1)
	if len(s) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}
	return value.NewString(s), nil
}

func textREMatch(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	matched, err := regexp.MatchString(s1, s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	if matched {
		ret = value.TrueValue
	} else {
		ret = value.FalseValue
	}

	return
}

func textREFind(args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			ret = value.UndefinedValue
			return
		}

		arr := &value.Array{}
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				arr.Value = append(arr.Value,
					&value.ImmutableMap{Value: map[string]core.Object{
						"text":  &value.String{Value: s2[m[i]:m[i+1]]},
						"begin": &value.Int{Value: int64(m[i])},
						"end":   &value.Int{Value: int64(m[i+1])},
					}})
			}
		}

		ret = &value.Array{Value: []core.Object{arr}}

		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}
	m := re.FindAllStringSubmatchIndex(s2, int(i3))
	if m == nil {
		ret = value.UndefinedValue
		return
	}

	arr := &value.Array{}
	for _, m := range m {
		subMatch := &value.Array{}
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				subMatch.Value = append(subMatch.Value,
					&value.ImmutableMap{Value: map[string]core.Object{
						"text":  &value.String{Value: s2[m[i]:m[i+1]]},
						"begin": &value.Int{Value: int64(m[i])},
						"end":   &value.Int{Value: int64(m[i+1])},
					}})
			}
		}

		arr.Value = append(arr.Value, subMatch)
	}

	ret = arr

	return
}

func textREReplace(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s3, ok := args[2].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
	} else {
		s, ok := doTextRegexpReplace(re, s2, s3)
		if !ok {
			return nil, gse.ErrStringLimit
		}

		ret = value.NewString(s)
	}

	return
}

func textRESplit(args ...core.Object) (ret core.Object, err error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	var i3 = -1
	if numArgs > 2 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			err = gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	arr := &value.Array{}
	for _, s := range re.Split(s2, i3) {
		arr.Value = append(arr.Value, value.NewString(s))
	}

	ret = arr

	return
}

func textRECompile(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
	} else {
		ret = makeTextRegexp(re)
	}

	return
}

func textReplace(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 4 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s3, ok := args[2].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "string(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := args[3].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	s, ok := doTextReplace(s1, s2, s3, int(i4))
	if !ok {
		err = gse.ErrStringLimit
		return
	}

	ret = value.NewString(s)

	return
}

func textSubstring(args ...core.Object) (ret core.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	strlen := len(s1)
	i3 := strlen
	if argslen == 3 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			err = gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "int(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	if int(i2) > i3 {
		err = gse.ErrInvalidIndexType
		return
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

	ret = &value.String{Value: s1[i2:i3]}

	return
}

func textPadLeft(args ...core.Object) (ret core.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if int(i2) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		ret = &value.String{Value: s1}
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			err = gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "string(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = &value.String{Value: s1}
		return
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := strings.Repeat(s3, padCount) + s1
	ret = &value.String{Value: retStr[len(retStr)-int(i2):]}

	return
}

func textPadRight(args ...core.Object) (ret core.Object, err error) {
	argslen := len(args)
	if argslen != 2 && argslen != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if int(i2) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		ret = &value.String{Value: s1}
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			err = gse.ErrInvalidArgumentType{
				Name:     "third",
				Expected: "string(compatible)",
				Found:    args[2].TypeName(),
			}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = &value.String{Value: s1}
		return
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := s1 + strings.Repeat(s3, padCount)
	ret = &value.String{Value: retStr[:i2]}

	return
}

func textRepeat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
	}

	if len(s1)*int(i2) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}

	return &value.String{Value: strings.Repeat(s1, int(i2))}, nil
}
*/

func textJoin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}

	var slen int
	var ss1 []string
	switch arg0 := args[0].(type) {
	case *value.Array:
		for idx, a := range arg0.Native() {
			as, ok := a.AsString()
			if !ok {
				return nil, gse.ErrInvalidArgumentType{
					Name:     fmt.Sprintf("first[%d]", idx),
					Expected: "string(compatible)",
					Found:    a.TypeName(),
				}
			}
			slen += len(as)
			ss1 = append(ss1, as)
		}
	default:
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "array",
			Found:    args[0].TypeName(),
		}
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}

	// make sure output length does not exceed the limit
	if slen+len(s2)*(len(ss1)-1) > core.MaxStringLen {
		return nil, gse.ErrStringLimit
	}

	return value.NewString(strings.Join(ss1, s2)), nil
}

/*
func textFormatBool(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	b1, ok := args[0].(*value.Bool)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "bool",
			Found:    args[0].TypeName(),
		}
		return
	}

	if b1 == value.TrueValue {
		ret = &value.String{Value: "true"}
	} else {
		ret = &value.String{Value: "false"}
	}

	return
}

func textFormatFloat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 4 {
		err = gse.ErrWrongNumArguments
		return
	}

	f1, ok := args[0].(*value.Float)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "float",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := args[3].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	ret = &value.String{Value: strconv.FormatFloat(f1.Value, s2[0], int(i3), int(i4))}

	return
}

func textFormatInt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].(*value.Int)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &value.String{Value: strconv.FormatInt(i1.Value, int(i2))}

	return
}

func textParseBool(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*value.String)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseBool(s1.Value)
	if err != nil {
		ret = wrapError(err)
		return
	}

	if parsed {
		ret = value.TrueValue
	} else {
		ret = value.FalseValue
	}

	return
}

func textParseFloat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*value.String)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseFloat(s1.Value, int(i2))
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &value.Float{Value: parsed}

	return
}

func textParseInt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*value.String)
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = gse.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	parsed, err := strconv.ParseInt(s1.Value, int(i2), int(i3))
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &value.Int{Value: parsed}

	return
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

func textContains(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if strings.Contains(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func textContainsAny(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if strings.ContainsAny(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func textEqualFold(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if strings.EqualFold(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func textHasPrefix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if strings.HasPrefix(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}

func textHasSuffix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, gse.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
	}
	if strings.HasSuffix(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}
*/
