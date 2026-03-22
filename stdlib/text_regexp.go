package stdlib

import (
	"regexp"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

func makeTextRegexp(vm core.VM, re *regexp.Regexp) *value.Record {
	reMatch := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 1 {
			return nil, core.NewWrongNumArgumentsError("text.regexp.match", "1", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.match", "first", "string(compatible)", args[0])
		}

		return vm.Allocator().NewBool(re.MatchString(s1)), nil
	}

	reFind := func(vm core.VM, args ...core.Object) (core.Object, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return nil, core.NewWrongNumArgumentsError("text.regexp.find", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.find", "first", "string(compatible)", args[0])
		}

		alloc := vm.Allocator()

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return alloc.NewUndefined(), nil
			}

			arr := alloc.NewArray(nil, false).(*value.Array)
			for i := 0; i < len(m); i += 2 {
				arr.Append(alloc.NewRecord(map[string]core.Object{
					"text":  alloc.NewString(s1[m[i]:m[i+1]]),
					"begin": alloc.NewInt(int64(m[i])),
					"end":   alloc.NewInt(int64(m[i+1])),
				}, true))
			}

			return alloc.NewArray([]core.Object{arr}, false), nil
		}

		i2, ok := args[1].AsInt()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.find", "second", "int(compatible)", args[1])
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return alloc.NewUndefined(), nil
		}

		arr := alloc.NewArray(nil, false).(*value.Array)
		for _, m := range m {
			subMatch := alloc.NewArray(nil, false).(*value.Array)
			for i := 0; i < len(m); i += 2 {
				subMatch.Append(alloc.NewRecord(map[string]core.Object{
					"text":  alloc.NewString(s1[m[i]:m[i+1]]),
					"begin": alloc.NewInt(int64(m[i])),
					"end":   alloc.NewInt(int64(m[i+1])),
				}, true))
			}
			arr.Append(subMatch)
		}

		return arr, nil
	}

	reReplace := func(vm core.VM, args ...core.Object) (core.Object, error) {
		if len(args) != 2 {
			return nil, core.NewWrongNumArgumentsError("text.regexp.replace", "2", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.replace", "first", "string(compatible)", args[0])
		}

		s2, ok := args[1].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.replace", "second", "string(compatible)", args[1])
		}

		s, ok := doTextRegexpReplace(re, s1, s2)
		if !ok {
			return nil, core.NewStringLimitError("text.regexp.replace")
		}

		return vm.Allocator().NewString(s), nil
	}

	reSplit := func(vm core.VM, args ...core.Object) (core.Object, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return nil, core.NewWrongNumArgumentsError("text.regexp.split", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return nil, core.NewInvalidArgumentTypeError("text.regexp.split", "first", "string(compatible)", args[0])
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt()
			i2 = int(i2t)
			if !ok {
				return nil, core.NewInvalidArgumentTypeError("text.regexp.split", "second", "int(compatible)", args[1])
			}
		}

		spl := re.Split(s1, i2)
		arr := make([]core.Object, 0, len(spl))
		alloc := vm.Allocator()
		for _, s := range spl {
			arr = append(arr, alloc.NewString(s))
		}

		return alloc.NewArray(arr, false), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecord(map[string]core.Object{
		"match":   alloc.NewBuiltinFunction("match", reMatch, 1, false),     // match(text) => bool
		"find":    alloc.NewBuiltinFunction("find", reFind, 1, true),        // find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
		"replace": alloc.NewBuiltinFunction("replace", reReplace, 2, false), // replace(src, repl) => string
		"split":   alloc.NewBuiltinFunction("split", reSplit, 1, true),      // split(text[,maxCount]) => array(string)
	}, true).(*value.Record)
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
