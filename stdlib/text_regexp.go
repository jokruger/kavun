package stdlib

import (
	"regexp"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

func makeTextRegexp(re *regexp.Regexp) *value.Map {
	reMatch := func(args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, gse.ErrWrongNumArguments
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.match", "first", "string(compatible)", args[0])
		}

		if re.MatchString(s1) {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	}

	reFind := func(args ...core.Object) (core.Object, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return nil, gse.ErrWrongNumArguments
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.find", "first", "string(compatible)", args[0])
		}

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return value.UndefinedValue, nil
			}

			arr := value.NewArray(nil, false)
			for i := 0; i < len(m); i += 2 {
				arr.Append(value.NewMap(map[string]core.Object{
					"text":  value.NewString(s1[m[i]:m[i+1]]),
					"begin": value.NewInt(int64(m[i])),
					"end":   value.NewInt(int64(m[i+1])),
				}, true))
			}

			return value.NewArray([]core.Object{arr}, false), nil
		}

		i2, ok := args[1].AsInt()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.find", "second", "int(compatible)", args[1])
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return value.UndefinedValue, nil
		}

		arr := value.NewArray(nil, false)
		for _, m := range m {
			subMatch := value.NewArray(nil, false)
			for i := 0; i < len(m); i += 2 {
				subMatch.Append(value.NewMap(map[string]core.Object{
					"text":  value.NewString(s1[m[i]:m[i+1]]),
					"begin": value.NewInt(int64(m[i])),
					"end":   value.NewInt(int64(m[i+1])),
				}, true))
			}
			arr.Append(subMatch)
		}

		return arr, nil
	}

	reReplace := func(args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, gse.ErrWrongNumArguments
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.replace", "first", "string(compatible)", args[0])
		}

		s2, ok := args[1].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.replace", "second", "string(compatible)", args[1])
		}

		s, ok := doTextRegexpReplace(re, s1, s2)
		if !ok {
			return nil, core.StringLimit("text.regexp.replace")
		}

		return value.NewString(s), nil
	}

	reSplit := func(args ...core.Object) (core.Object, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return nil, gse.ErrWrongNumArguments
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.InvalidArgumentType("text.regexp.split", "first", "string(compatible)", args[0])
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt()
			i2 = int(i2t)
			if !ok {
				return nil, core.InvalidArgumentType("text.regexp.split", "second", "int(compatible)", args[1])
			}
		}

		spl := re.Split(s1, i2)
		arr := make([]core.Object, 0, len(spl))
		for _, s := range spl {
			arr = append(arr, value.NewString(s))
		}

		return value.NewArray(arr, false), nil
	}

	return value.NewMap(map[string]core.Object{
		"match":   value.NewBuiltinFunction("match", reMatch, 1, false),     // match(text) => bool
		"find":    value.NewBuiltinFunction("find", reFind, 1, true),        // find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
		"replace": value.NewBuiltinFunction("replace", reReplace, 2, false), // replace(src, repl) => string
		"split":   value.NewBuiltinFunction("split", reSplit, 1, true),      // split(text[,maxCount]) => array(string)
	}, true)
}

// Size-limit checking implementation of regexp.ReplaceAllString.
func doTextRegexpReplace(re *regexp.Regexp, src, repl string) (string, bool) {
	idx := 0
	out := ""
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		var exp []byte
		exp = re.ExpandString(exp, repl, src, m)
		if len(out)+m[0]-idx+len(exp) > core.MaxStringLen {
			return "", false
		}
		out += src[idx:m[0]] + string(exp)
		idx = m[1]
	}
	if idx < len(src) {
		if len(out)+len(src)-idx > core.MaxStringLen {
			return "", false
		}
		out += src[idx:]
	}
	return out, true
}
