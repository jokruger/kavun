package stdlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

var textModule = map[string]core.Object{
	"re_match":       value.NewBuiltinFunction("re_match", textREMatch, 2, false),               // re_match(pattern, text) => bool/error
	"re_find":        value.NewBuiltinFunction("re_find", textREFind, 3, false),                 // re_find(pattern, text, count) => [[{text:,begin:,end:}]]/undefined
	"re_replace":     value.NewBuiltinFunction("re_replace", textREReplace, 3, false),           // re_replace(pattern, text, repl) => string/error
	"re_split":       value.NewBuiltinFunction("re_split", textRESplit, 3, false),               // re_split(pattern, text, count) => [string]/error
	"re_compile":     value.NewBuiltinFunction("re_compile", textRECompile, 1, false),           // re_compile(pattern) => Regexp/error
	"compare":        value.NewBuiltinFunction("compare", stringsCompare, 2, false),             // compare(a, b) => int
	"contains":       value.NewBuiltinFunction("contains", textContains, 2, false),              // contains(s, substr) => bool
	"contains_any":   value.NewBuiltinFunction("contains_any", textContainsAny, 2, false),       // contains_any(s, chars) => bool
	"count":          value.NewBuiltinFunction("count", stringsCount, 2, false),                 // count(s, substr) => int
	"equal_fold":     value.NewBuiltinFunction("equal_fold", textEqualFold, 2, false),           // "equal_fold(s, t) => bool
	"fields":         value.NewBuiltinFunction("fields", stringsFields, 1, false),               // fields(s) => [string]
	"has_prefix":     value.NewBuiltinFunction("has_prefix", textHasPrefix, 2, false),           // has_prefix(s, prefix) => bool
	"has_suffix":     value.NewBuiltinFunction("has_suffix", textHasSuffix, 2, false),           // has_suffix(s, suffix) => bool
	"index":          value.NewBuiltinFunction("index", stringsIndex, 2, false),                 // index(s, substr) => int
	"index_any":      value.NewBuiltinFunction("index_any", stringsIndexAny, 2, false),          // index_any(s, chars) => int
	"join":           value.NewBuiltinFunction("join", textJoin, 2, false),                      // join(arr, sep) => string
	"last_index":     value.NewBuiltinFunction("last_index", stringsLastIndex, 2, false),        // last_index(s, substr) => int
	"last_index_any": value.NewBuiltinFunction("last_index_any", stringsLastIndexAny, 2, false), // last_index_any(s, chars) => int
	"repeat":         value.NewBuiltinFunction("repeat", textRepeat, 2, false),                  // repeat(s, count) => string
	"replace":        value.NewBuiltinFunction("replace", textReplace, 4, false),                // replace(s, old, new, n) => string
	"substr":         value.NewBuiltinFunction("substr", textSubstring, 3, false),               // substr(s, lower, upper) => string
	"split":          value.NewBuiltinFunction("split", stringsSplit, 2, false),                 // split(s, sep) => [string]
	"split_after":    value.NewBuiltinFunction("split_after", stringsSplitAfter, 2, false),      // split_after(s, sep) => [string]
	"split_after_n":  value.NewBuiltinFunction("split_after_n", stringsSplitAfterN, 3, false),   // split_after_n(s, sep, n) => [string]
	"split_n":        value.NewBuiltinFunction("split_n", stringsSplitN, 3, false),              // split_n(s, sep, n) => [string]
	"title":          value.NewBuiltinFunction("title", stringsTitle, 1, false),                 // title(s) => string
	"to_lower":       value.NewBuiltinFunction("to_lower", stringsToLower, 1, false),            // to_lower(s) => string
	"to_title":       value.NewBuiltinFunction("to_title", stringsToTitle, 1, false),            // to_title(s) => string
	"to_upper":       value.NewBuiltinFunction("to_upper", stringsToUpper, 1, false),            // to_upper(s) => string
	"pad_left":       value.NewBuiltinFunction("pad_left", textPadLeft, 3, false),               // pad_left(s, pad_len, pad_with) => string
	"pad_right":      value.NewBuiltinFunction("pad_right", textPadRight, 3, false),             // pad_right(s, pad_len, pad_with) => string
	"trim":           value.NewBuiltinFunction("trim", stringsTrim, 2, false),                   // trim(s, cutset) => string
	"trim_left":      value.NewBuiltinFunction("trim_left", stringsTrimLeft, 2, false),          // trim_left(s, cutset) => string
	"trim_prefix":    value.NewBuiltinFunction("trim_prefix", stringsTrimPrefix, 2, false),      // trim_prefix(s, prefix) => string
	"trim_right":     value.NewBuiltinFunction("trim_right", stringsTrimRight, 2, false),        // trim_right(s, cutset) => string
	"trim_space":     value.NewBuiltinFunction("trim_space", stringsTrimSpace, 1, false),        // trim_space(s) => string
	"trim_suffix":    value.NewBuiltinFunction("trim_suffix", stringsTrimSuffix, 2, false),      // trim_suffix(s, suffix) => string
	"atoi":           value.NewBuiltinFunction("atoi", strconvAtoi, 1, false),                   // atoi(str) => int/error
	"format_bool":    value.NewBuiltinFunction("format_bool", textFormatBool, 1, false),         // format_bool(b) => string
	"format_float":   value.NewBuiltinFunction("format_float", textFormatFloat, 4, false),       // format_float(f, fmt, prec, bits) => string
	"format_int":     value.NewBuiltinFunction("format_int", textFormatInt, 2, false),           // format_int(i, base) => string
	"itoa":           value.NewBuiltinFunction("itoa", strconvItoa, 1, false),                   // itoa(i) => string
	"parse_bool":     value.NewBuiltinFunction("parse_bool", textParseBool, 1, false),           // parse_bool(str) => bool/error
	"parse_float":    value.NewBuiltinFunction("parse_float", textParseFloat, 2, false),         // parse_float(str, bits) => float/error
	"parse_int":      value.NewBuiltinFunction("parse_int", textParseInt, 3, false),             // parse_int(str, base, bits) => int/error
	"quote":          value.NewBuiltinFunction("quote", strconvQuote, 1, false),                 // quote(str) => string
	"unquote":        value.NewBuiltinFunction("unquote", strconvUnquote, 1, false),             // unquote(str) => string/error
}

func strconvItoa(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	i1, ok := args[0].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	s := strconv.Itoa(int(i1))
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.itoa")
	}
	return value.NewString(s), nil
}

func strconvAtoi(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := strconv.Atoi(s1)
	if err != nil {
		return wrapError(err), nil
	}
	return value.NewInt(int64(res)), nil
}

func stringsTrimSuffix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	s := strings.TrimSuffix(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim_suffix")
	}
	return value.NewString(s), nil
}

func stringsTrimRight(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	s := strings.TrimRight(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim_right")
	}
	return value.NewString(s), nil
}

func stringsTrimPrefix(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	s := strings.TrimPrefix(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim_prefix")
	}
	return value.NewString(s), nil
}

func stringsTrimLeft(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	s := strings.TrimLeft(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim_left")
	}
	return value.NewString(s), nil
}

func stringsTrim(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	s := strings.Trim(s1, s2)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim")
	}
	return value.NewString(s), nil
}

func stringsLastIndexAny(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.LastIndexAny(s1, s2))), nil
}

func stringsLastIndex(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.LastIndex(s1, s2))), nil
}

func stringsIndexAny(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.IndexAny(s1, s2))), nil
}

func stringsIndex(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.Index(s1, s2))), nil
}

func stringsCount(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.Count(s1, s2))), nil
}

func stringsCompare(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	return value.NewInt(int64(strings.Compare(s1, s2))), nil
}

func stringsSplitN(args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
	}
	spl := strings.SplitN(s1, s2, int(i3))
	arr := make([]core.Object, 0, len(spl))
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return nil, core.StringLimit("text.split_n")
		}
		arr = append(arr, value.NewString(res))
	}
	return value.NewArray(arr, false), nil
}

func stringsSplitAfterN(args ...core.Object) (core.Object, error) {
	if len(args) != 3 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	i3, ok := args[2].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
	}
	spl := strings.SplitAfterN(s1, s2, int(i3))
	arr := make([]core.Object, 0, len(spl))
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return nil, core.StringLimit("text.split_after_n")
		}
		arr = append(arr, value.NewString(res))
	}
	return value.NewArray(arr, false), nil
}

func stringsSplitAfter(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	spl := strings.SplitAfter(s1, s2)
	arr := make([]core.Object, 0, len(spl))
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return nil, core.StringLimit("text.split_after")
		}
		arr = append(arr, value.NewString(res))
	}
	return value.NewArray(arr, false), nil
}

func stringsSplit(args ...core.Object) (core.Object, error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	spl := strings.Split(s1, s2)
	arr := make([]core.Object, 0, len(spl))
	for _, res := range spl {
		if len(res) > core.MaxStringLen {
			return nil, core.StringLimit("text.split")
		}
		arr = append(arr, value.NewString(res))
	}
	return value.NewArray(arr, false), nil
}

func strconvUnquote(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res, err := strconv.Unquote(s1)
	if err != nil {
		return wrapError(err), nil
	}
	if len(res) > core.MaxStringLen {
		return nil, core.StringLimit("text.unquote")
	}
	return value.NewString(res), nil
}

func stringsFields(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	res := strings.Fields(s1)
	arr := make([]core.Object, 0, len(res))
	for _, elem := range res {
		if len(elem) > core.MaxStringLen {
			return nil, core.StringLimit("text.fields")
		}
		arr = append(arr, value.NewString(elem))
	}
	return value.NewArray(arr, false), nil
}

func strconvQuote(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strconv.Quote(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.quote")
	}
	return value.NewString(s), nil
}

func stringsTrimSpace(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strings.TrimSpace(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.trim_space")
	}
	return value.NewString(s), nil
}

func stringsToTitle(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strings.ToTitle(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.to_title")
	}
	return value.NewString(s), nil
}

func stringsToUpper(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strings.ToUpper(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.to_upper")
	}
	return value.NewString(s), nil
}

func stringsToLower(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strings.ToLower(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.to_lower")
	}
	return value.NewString(s), nil
}

func stringsTitle(args ...core.Object) (core.Object, error) {
	if len(args) != 1 {
		return nil, gse.ErrWrongNumArguments
	}
	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s := strings.Title(s1)
	if len(s) > core.MaxStringLen {
		return nil, core.StringLimit("text.title")
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
		return
	}

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			ret = value.UndefinedValue
			return
		}

		arr := value.NewArray(nil, false)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				arr.Append(value.NewMap(map[string]core.Object{
					"text":  value.NewString(s2[m[i]:m[i+1]]),
					"begin": value.NewInt(int64(m[i])),
					"end":   value.NewInt(int64(m[i+1])),
				}, true))
			}
		}

		ret = value.NewArray([]core.Object{arr}, false)

		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
		return
	}
	m := re.FindAllStringSubmatchIndex(s2, int(i3))
	if m == nil {
		ret = value.UndefinedValue
		return
	}

	arr := value.NewArray(nil, false)
	for _, m := range m {
		subMatch := value.NewArray(nil, false)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				subMatch.Append(value.NewMap(map[string]core.Object{
					"text":  value.NewString(s2[m[i]:m[i+1]]),
					"begin": value.NewInt(int64(m[i])),
					"end":   value.NewInt(int64(m[i+1])),
				}, true))
			}
		}
		arr.Append(subMatch)
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
		return
	}

	s3, ok := args[2].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "string(compatible)", Found: args[2].TypeName()}
		return
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
	} else {
		s, ok := doTextRegexpReplace(re, s2, s3)
		if !ok {
			return nil, core.StringLimit("text.re_replace")
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
		return
	}

	var i3 = -1
	if numArgs > 2 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
			return
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	spl := re.Split(s2, i3)
	arr := make([]core.Object, 0, len(spl))
	for _, s := range spl {
		arr = append(arr, value.NewString(s))
	}

	ret = value.NewArray(arr, false)

	return
}

func textRECompile(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
		return
	}

	s3, ok := args[2].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "string(compatible)", Found: args[2].TypeName()}
		return
	}

	i4, ok := args[3].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "fourth", Expected: "int(compatible)", Found: args[3].TypeName()}
		return
	}

	s, ok := doTextReplace(s1, s2, s3, int(i4))
	if !ok {
		err = core.StringLimit("text.replace")
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	strlen := len(s1)
	i3 := strlen
	if argslen == 3 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
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

	ret = value.NewString(s1[i2:i3])

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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	if int(i2) > core.MaxStringLen {
		return nil, core.StringLimit("text.pad_left")
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		ret = value.NewString(s1)
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "string(compatible)", Found: args[2].TypeName()}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = value.NewString(s1)
		return
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := strings.Repeat(s3, padCount) + s1
	ret = value.NewString(retStr[len(retStr)-int(i2):])

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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	if int(i2) > core.MaxStringLen {
		return nil, core.StringLimit("text.pad_right")
	}

	sLen := len(s1)
	if sLen >= int(i2) {
		ret = value.NewString(s1)
		return
	}

	s3 := " "
	if argslen == 3 {
		s3, ok = args[2].AsString()
		if !ok {
			err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "string(compatible)", Found: args[2].TypeName()}
			return
		}
	}

	padStrLen := len(s3)
	if padStrLen == 0 {
		ret = value.NewString(s1)
		return
	}

	padCount := ((int(i2) - padStrLen) / padStrLen) + 1
	retStr := s1 + strings.Repeat(s3, padCount)
	ret = value.NewString(retStr[:i2])

	return
}

func textRepeat(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}

	s1, ok := args[0].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}

	i2, ok := args[1].AsInt()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}

	if len(s1)*int(i2) > core.MaxStringLen {
		return nil, core.StringLimit("text.repeat")
	}

	return value.NewString(strings.Repeat(s1, int(i2))), nil
}

func textJoin(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		return nil, gse.ErrWrongNumArguments
	}

	var slen int
	var ss1 []string
	switch arg0 := args[0].(type) {
	case *value.Array:
		for idx, a := range arg0.Value() {
			as, ok := a.AsString()
			if !ok {
				return nil, &gse.InvalidArgumentTypeError{Name: fmt.Sprintf("first[%d]", idx), Expected: "string(compatible)", Found: a.TypeName()}
			}
			slen += len(as)
			ss1 = append(ss1, as)
		}
	default:
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "array", Found: args[0].TypeName()}
	}

	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}

	// make sure output length does not exceed the limit
	if slen+len(s2)*(len(ss1)-1) > core.MaxStringLen {
		return nil, core.StringLimit("text.join")
	}

	return value.NewString(strings.Join(ss1, s2)), nil
}

func textFormatBool(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	b1, ok := args[0].(*value.Bool)
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "bool", Found: args[0].TypeName()}
		return
	}

	if b1 == value.TrueValue {
		ret = value.NewString("true")
	} else {
		ret = value.NewString("false")
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "float", Found: args[0].TypeName()}
		return
	}

	s2, ok := args[1].AsString()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
		return
	}

	i4, ok := args[3].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	ret = value.NewString(strconv.FormatFloat(f1.Value(), s2[0], int(i3), int(i4)))

	return
}

func textFormatInt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 2 {
		err = gse.ErrWrongNumArguments
		return
	}

	i1, ok := args[0].(*value.Int)
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "int", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	ret = value.NewString(strconv.FormatInt(i1.Value(), int(i2)))

	return
}

func textParseBool(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 1 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*value.String)
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string", Found: args[0].TypeName()}
		return
	}

	parsed, err := strconv.ParseBool(s1.Value())
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
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	parsed, err := strconv.ParseFloat(s1.Value(), int(i2))
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = value.NewFloat(parsed)

	return
}

func textParseInt(args ...core.Object) (ret core.Object, err error) {
	if len(args) != 3 {
		err = gse.ErrWrongNumArguments
		return
	}

	s1, ok := args[0].(*value.String)
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "first", Expected: "string", Found: args[0].TypeName()}
		return
	}

	i2, ok := args[1].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
		return
	}

	i3, ok := args[2].AsInt()
	if !ok {
		err = &gse.InvalidArgumentTypeError{Name: "third", Expected: "int(compatible)", Found: args[2].TypeName()}
		return
	}

	parsed, err := strconv.ParseInt(s1.Value(), int(i2), int(i3))
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = value.NewInt(parsed)

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
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
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
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
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
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
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
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
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
		return nil, &gse.InvalidArgumentTypeError{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	s2, ok := args[1].AsString()
	if !ok {
		return nil, &gse.InvalidArgumentTypeError{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	if strings.HasSuffix(s1, s2) {
		return value.TrueValue, nil
	}
	return value.FalseValue, nil
}
