package stdlib

import (
	"regexp"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeTextRegexp(vm core.VM, re *regexp.Regexp) (core.Value, error) {
	alloc := vm.Allocator()

	// match(text) => bool
	reMatch := alloc.NewBuiltinFunctionValue("match", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.match", "1", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.match", "first", "string(compatible)", args[0].TypeName())
		}

		return core.BoolValue(re.MatchString(s1)), nil
	}, 1, false)

	// find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
	reFind := alloc.NewBuiltinFunctionValue("find", func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.find", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.find", "first", "string(compatible)", args[0].TypeName())
		}

		alloc := vm.Allocator()

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return core.Undefined, nil
			}

			arr := alloc.NewArray(len(m)/2, false)
			for i := 0; i < len(m); i += 2 {
				txt := alloc.NewStringValue(s1[m[i]:m[i+1]])
				t := alloc.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				arr = append(arr, t)
			}

			t := alloc.NewArrayValue(arr, false)
			return alloc.NewArrayValue([]core.Value{t}, false), nil
		}

		i2, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.find", "second", "int(compatible)", args[1].TypeName())
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return core.Undefined, nil
		}

		arr := alloc.NewArray(len(m), false)
		for _, m := range m {
			subMatch := alloc.NewArray(len(m)/2, false)
			for i := 0; i < len(m); i += 2 {
				txt := alloc.NewStringValue(s1[m[i]:m[i+1]])
				t := alloc.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				subMatch = append(subMatch, t)
			}
			t := alloc.NewArrayValue(subMatch, false)
			arr = append(arr, t)
		}

		return alloc.NewArrayValue(arr, false), nil
	}, 1, true)

	// replace(src, repl) => string
	reReplace := alloc.NewBuiltinFunctionValue("replace", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.replace", "2", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.replace", "first", "string(compatible)", args[0].TypeName())
		}

		s2, ok := args[1].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.replace", "second", "string(compatible)", args[1].TypeName())
		}

		s, ok := doTextRegexpReplace(re, s1, s2)
		if !ok {
			return core.Undefined, errs.NewStringLimitError("text.regexp.replace")
		}

		return vm.Allocator().NewStringValue(s), nil
	}, 2, false)

	// split(text[,maxCount]) => array(string)
	reSplit := alloc.NewBuiltinFunctionValue("split", func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.split", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.split", "first", "string(compatible)", args[0].TypeName())
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt()
			i2 = int(i2t)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.split", "second", "int(compatible)", args[1].TypeName())
			}
		}

		spl := re.Split(s1, i2)
		alloc := vm.Allocator()
		arr := alloc.NewArray(len(spl), false)
		for _, s := range spl {
			t := alloc.NewStringValue(s)
			arr = append(arr, t)
		}

		return alloc.NewArrayValue(arr, false), nil
	}, 1, true)

	m := vm.Allocator().NewRecordValue(map[string]core.Value{
		"match":   reMatch,
		"find":    reFind,
		"replace": reReplace,
		"split":   reSplit,
	}, true)

	return m, nil
}

// Size-limit checking implementation of regexp.ReplaceAllString.
func doTextRegexpReplace(re *regexp.Regexp, src, repl string) (string, bool) {
	idx := 0
	out := ""
	for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
		var exp []byte
		exp = re.ExpandString(exp, repl, src, m)
		out += src[idx:m[0]] + string(exp)
		idx = m[1]
	}
	if idx < len(src) {
		out += src[idx:]
	}
	return out, true
}
