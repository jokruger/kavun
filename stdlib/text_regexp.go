package stdlib

import (
	"regexp"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeTextRegexp(a *core.Arena, vm core.VM, re *regexp.Regexp) (core.Value, error) {
	// match(text) => bool
	reMatch, err := a.NewBuiltinClosureValue("match", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.match", "1", len(args))
		}

		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.match", "first", "string(compatible)", args[0].TypeName(a))
		}

		return core.BoolValue(re.MatchString(s1)), nil
	}, 1, false)
	if err != nil {
		return core.Undefined, err
	}

	// find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
	reFind, err := a.NewBuiltinClosureValue("find", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.find", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.find", "first", "string(compatible)", args[0].TypeName(a))
		}

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return core.Undefined, nil
			}

			arr := a.NewArray(len(m)/2, false)
			for i := 0; i < len(m); i += 2 {
				txt, err := a.NewStringValue(s1[m[i]:m[i+1]])
				if err != nil {
					return core.Undefined, err
				}
				a.PinAllocated(txt)
				t, err := a.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				if err != nil {
					return core.Undefined, err
				}
				a.PinAllocated(t)
				arr = append(arr, t)
			}

			t, err := a.NewArrayValue(arr, false)
			if err != nil {
				return core.Undefined, err
			}
			a.PinAllocated(t)
			return a.NewArrayValue([]core.Value{t}, false)
		}

		i2, ok := args[1].AsInt(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.find", "second", "int(compatible)", args[1].TypeName(a))
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return core.Undefined, nil
		}

		arr := a.NewArray(len(m), false)
		for _, m := range m {
			subMatch := a.NewArray(len(m)/2, false)
			for i := 0; i < len(m); i += 2 {
				txt, err := a.NewStringValue(s1[m[i]:m[i+1]])
				if err != nil {
					return core.Undefined, err
				}
				a.PinAllocated(txt)
				t, err := a.NewRecordValue(map[string]core.Value{
					"text":  txt,
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				if err != nil {
					return core.Undefined, err
				}
				a.PinAllocated(t)
				subMatch = append(subMatch, t)
			}
			t, err := a.NewArrayValue(subMatch, false)
			if err != nil {
				return core.Undefined, err
			}
			a.PinAllocated(t)
			arr = append(arr, t)
		}

		return a.NewArrayValue(arr, false)
	}, 1, true)
	if err != nil {
		return core.Undefined, err
	}

	// replace(src, repl) => string
	reReplace, err := a.NewBuiltinClosureValue("replace", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.replace", "2", len(args))
		}

		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.replace", "first", "string(compatible)", args[0].TypeName(a))
		}

		s2, ok := args[1].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.replace", "second", "string(compatible)", args[1].TypeName(a))
		}

		s, ok := doTextRegexpReplace(re, s1, s2)
		if !ok {
			return core.Undefined, errs.NewResourceLimitError("text.regexp.replace")
		}

		return a.NewStringValue(s)
	}, 2, false)
	if err != nil {
		return core.Undefined, err
	}

	// split(text[,maxCount]) => array(string)
	reSplit, err := a.NewBuiltinClosureValue("split", func(a *core.Arena, vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("text.regexp.split", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString(a)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.split", "first", "string(compatible)", args[0].TypeName(a))
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt(a)
			i2 = int(i2t)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("text.regexp.split", "second", "int(compatible)", args[1].TypeName(a))
			}
		}

		spl := re.Split(s1, i2)
		arr := a.NewArray(len(spl), false)
		for _, s := range spl {
			t, err := a.NewStringValue(s)
			if err != nil {
				return core.Undefined, err
			}
			arr = append(arr, t)
		}

		return a.NewArrayValue(arr, false)
	}, 1, true)
	if err != nil {
		return core.Undefined, err
	}

	m, err := a.NewRecordValue(map[string]core.Value{
		"match":   reMatch,
		"find":    reFind,
		"replace": reReplace,
		"split":   reSplit,
	}, true)
	if err != nil {
		return core.Undefined, err
	}

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
