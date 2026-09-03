package stdlib

import (
	"regexp"
	"strings"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

func makeCompiledRegexp(vm core.VM, re *regexp.Regexp) (core.Value, error) {
	// match(text) => bool
	reMatch := core.NewBuiltinClosureValue("match", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, errs.NewWrongNumArgumentsError("regexp.match", "1", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.match", "first", "string(compatible)", args[0].TypeName())
		}

		return core.BoolValue(re.MatchString(s1)), nil
	}, 1, false)

	// find(text[,maxCount]) => array(array({text:,begin:,end:}))/undefined
	reFind := core.NewBuiltinClosureValue("find", func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("regexp.find", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.find", "first", "string(compatible)", args[0].TypeName())
		}

		if numArgs == 1 {
			m := re.FindStringSubmatchIndex(s1)
			if m == nil {
				return core.Undefined, nil
			}

			arr := make([]core.Value, 0, len(m)/2)
			for i := 0; i < len(m); i += 2 {
				t := core.NewRecordValue(map[string]core.Value{
					"text":  core.NewStringValue(s1[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				arr = append(arr, t)
			}

			return core.NewArrayValue([]core.Value{core.NewArrayValue(arr, false)}, false), nil
		}

		i2, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.find", "second", "int(compatible)", args[1].TypeName())
		}
		m := re.FindAllStringSubmatchIndex(s1, int(i2))
		if m == nil {
			return core.Undefined, nil
		}

		arr := make([]core.Value, 0, len(m))
		for _, m := range m {
			subMatch := make([]core.Value, 0, len(m)/2)
			for i := 0; i < len(m); i += 2 {
				t := core.NewRecordValue(map[string]core.Value{
					"text":  core.NewStringValue(s1[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				subMatch = append(subMatch, t)
			}
			arr = append(arr, core.NewArrayValue(subMatch, false))
		}

		return core.NewArrayValue(arr, false), nil
	}, 1, true)

	// replace(src, repl) => string
	reReplace := core.NewBuiltinClosureValue("replace", func(vm core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("regexp.replace", "2", len(args))
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.replace", "first", "string(compatible)", args[0].TypeName())
		}

		s2, ok := args[1].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.replace", "second", "string(compatible)", args[1].TypeName())
		}

		s, ok := doRegexpReplace(re, s1, s2)
		if !ok {
			return core.Undefined, errs.NewInvalidValueError(
				"(regexp.replace) expansion would exceed the replacement size limit")
		}

		return core.NewStringValue(s), nil
	}, 2, false)

	// split(text[,maxCount]) => array(string)
	reSplit := core.NewBuiltinClosureValue("split", func(vm core.VM, args []core.Value) (core.Value, error) {
		numArgs := len(args)
		if numArgs != 1 && numArgs != 2 {
			return core.Undefined, errs.NewWrongNumArgumentsError("regexp.split", "1 or 2", numArgs)
		}

		s1, ok := args[0].AsString()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.split", "first", "string(compatible)", args[0].TypeName())
		}

		var i2 = -1
		if numArgs > 1 {
			var i2t int64
			i2t, ok = args[1].AsInt()
			i2 = int(i2t)
			if !ok {
				return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.split", "second", "int(compatible)", args[1].TypeName())
			}
		}

		spl := re.Split(s1, i2)
		arr := make([]core.Value, 0, len(spl))
		for _, s := range spl {
			arr = append(arr, core.NewStringValue(s))
		}

		return core.NewArrayValue(arr, false), nil
	}, 1, true)

	m := core.NewRecordValue(map[string]core.Value{
		"match":   reMatch,
		"find":    reFind,
		"replace": reReplace,
		"split":   reSplit,
	}, true)

	return m, nil
}

// Size-limit checking implementation of regexp.ReplaceAllString.
// doRegexpReplace expands every match of re in src with repl. It answers false when the result would grow past
// core.MaxSequenceLen.
//
// The ceiling is needed because the result size is driven entirely by the script's own arguments: a template like
// "$0$0$0…" against a long subject multiplies the subject's size by the number of group references, so a handful
// of short strings can ask for terabytes. Every other count-driven allocation in Kavun checks the same ceiling
// before allocating (see core.MaxSequenceLen); this one has to, for the same reason.
//
// Two checks, because one is not enough on its own. The bound BEFORE the loop is arithmetic and conservative —
// every reference in the template can at most reproduce the whole subject — so an abusive call is rejected
// without doing any of the work. The running check inside the loop is exact, and is what actually guarantees the
// ceiling for a template the bound waved through.
func doRegexpReplace(re *regexp.Regexp, src, repl string) (string, bool) {
	matches := re.FindAllStringSubmatchIndex(src, -1)

	if n := len(matches); n > 0 {
		perMatch := len(repl) + strings.Count(repl, "$")*len(src)
		if perMatch > 0 && n > (core.MaxSequenceLen-len(src))/perMatch {
			return "", false
		}
	}

	idx := 0
	var out strings.Builder
	for _, m := range matches {
		var exp []byte
		exp = re.ExpandString(exp, repl, src, m)
		if out.Len()+(m[0]-idx)+len(exp) > core.MaxSequenceLen {
			return "", false
		}
		out.WriteString(src[idx:m[0]])
		out.Write(exp)
		idx = m[1]
	}
	if idx < len(src) {
		if out.Len()+(len(src)-idx) > core.MaxSequenceLen {
			return "", false
		}
		out.WriteString(src[idx:])
	}
	return out.String(), true
}
