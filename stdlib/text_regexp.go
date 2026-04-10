package stdlib

import (
	"regexp"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/errs"
)

func makeTextRegexp(vm core.VM, re *regexp.Regexp) core.Value {
	reMatch := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.UndefinedValue(), errs.NewWrongNumArgumentsError("text.regexp.match", "1", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.match", "first", "string(compatible)", args[0].TypeName())
		}

		return core.BoolValue(re.MatchString(s1)), nil
	}

	reFind := func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.UndefinedValue(), errs.NewWrongNumArgumentsError("text.regexp.find", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.find", "first", "string(compatible)", args[0].TypeName())
		}

		alloc := vm.Allocator()

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return core.UndefinedValue(), nil
			}

			arr := make([]core.Value, 0, len(m)/2)
			for i := 0; i < len(m); i += 2 {
				t := alloc.NewRecordValue(map[string]core.Value{
					"text":  alloc.NewStringValue(s1[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				arr = append(arr, t)
			}

			return alloc.NewArrayValue([]core.Value{alloc.NewArrayValue(arr, false)}, false), nil
		}

		i2, ok := args[1].AsInt()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.find", "second", "int(compatible)", args[1].TypeName())
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return core.UndefinedValue(), nil
		}

		arr := make([]core.Value, 0, len(m))
		for _, m := range m {
			subMatch := make([]core.Value, 0, len(m)/2)
			for i := 0; i < len(m); i += 2 {
				t := alloc.NewRecordValue(map[string]core.Value{
					"text":  alloc.NewStringValue(s1[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				subMatch = append(subMatch, t)
			}
			arr = append(arr, alloc.NewArrayValue(subMatch, false))
		}

		return alloc.NewArrayValue(arr, false), nil
	}

	reReplace := func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.UndefinedValue(), errs.NewWrongNumArgumentsError("text.regexp.replace", "2", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.replace", "first", "string(compatible)", args[0].TypeName())
		}

		s2, ok := args[1].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.replace", "second", "string(compatible)", args[1].TypeName())
		}

		s, ok := doTextRegexpReplace(re, s1, s2)
		if !ok {
			return core.UndefinedValue(), errs.NewStringLimitError("text.regexp.replace")
		}

		return vm.Allocator().NewStringValue(s), nil
	}

	reSplit := func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.UndefinedValue(), errs.NewWrongNumArgumentsError("text.regexp.split", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.split", "first", "string(compatible)", args[0].TypeName())
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt()
			i2 = int(i2t)
			if !ok {
				return core.UndefinedValue(), errs.NewInvalidArgumentTypeError("text.regexp.split", "second", "int(compatible)", args[1].TypeName())
			}
		}

		spl := re.Split(s1, i2)
		arr := make([]core.Value, 0, len(spl))
		alloc := vm.Allocator()
		for _, s := range spl {
			arr = append(arr, alloc.NewStringValue(s))
		}

		return alloc.NewArrayValue(arr, false), nil
	}

	alloc := vm.Allocator()
	return vm.Allocator().NewRecordValue(map[string]core.Value{
		"match":   alloc.NewBuiltinFunctionValue("match", reMatch, 1, false),     // match(text) => bool
		"find":    alloc.NewBuiltinFunctionValue("find", reFind, 1, true),        // find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
		"replace": alloc.NewBuiltinFunctionValue("replace", reReplace, 2, false), // replace(src, repl) => string
		"split":   alloc.NewBuiltinFunctionValue("split", reSplit, 1, true),      // split(text[,maxCount]) => array(string)
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
