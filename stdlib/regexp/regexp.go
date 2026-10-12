package regexp

import (
	"regexp"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

func init() {
	m := core.NewModule(value.ModuleRe, "regexp", true)
	// The regexp module is what remains of the former text module: everything
	// string-shaped became member functions on string/runes/bytes, and only the
	// five regex functions stay module-shaped.
	m.PureFn(members.ReMatch, 0, regexpREMatch, 2, core.Fixed)     // re_match(pattern, text) => bool/error
	m.PureFn(members.ReFind, 1, regexpREFind, 2, core.Variadic)    // re_find(pattern, text [,count]) => [[{text:,begin:,end:}]]/undefined
	m.PureFn(members.ReReplace, 2, regexpREReplace, 3, core.Fixed) // re_replace(pattern, text, repl) => string/error
	m.PureFn(members.ReSplit, 3, regexpRESplit, 2, core.Variadic)  // re_split(pattern, text [,count]) => [string]/error
	m.PureFn(members.ReCompile, 4, regexpRECompile, 1, core.Fixed) // re_compile(pattern) => Regexp/error
	if err := m.Register(); err != nil {
		panic(err)
	}
}

func regexpREMatch(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError("regexp.re_match", "2", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_match", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_match", "second", "string(compatible)", args[1].TypeName())
	}

	matched, err := regexp.MatchString(s1, s2)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "regexp.re_match", err)
	}

	return core.BoolValue(matched), nil
}

func regexpREFind(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("regexp.re_find", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_find", "first", "string(compatible)", args[0].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "regexp.re_find", err)
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_find", "second", "string(compatible)", args[1].TypeName())
	}

	if numArgs < 3 {
		m := re.FindStringSubmatchIndex(s2)
		if m == nil {
			return core.Undefined, nil
		}

		arr := make([]core.Value, 0, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				t := core.NewRecordValue(map[string]core.Value{
					"text":  core.NewStringValue(s2[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, false)
				arr = append(arr, t)
			}
		}

		return core.NewArrayValue([]core.Value{core.NewArrayValue(arr, false)}, false), nil
	}

	i3, ok := args[2].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_find", "third", "int(compatible)", args[2].TypeName())
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
				t := core.NewRecordValue(map[string]core.Value{
					"text":  core.NewStringValue(s2[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, true)
				subMatch = append(subMatch, t)
			}
		}
		arr = append(arr, core.NewArrayValue(subMatch, false))
	}

	return core.NewArrayValue(arr, false), nil
}

func regexpREReplace(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("regexp.re_replace", "3", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_replace", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_replace", "second", "string(compatible)", args[1].TypeName())
	}

	s3, ok := args[2].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_replace", "third", "string(compatible)", args[2].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "regexp.re_replace", err)
	}

	s, ok := doRegexpReplace(re, s2, s3)
	if !ok {
		return core.Undefined, errs.NewInvalidValueError(
			"(regexp.re_replace) expansion would exceed the replacement size limit")
	}

	return core.NewStringValue(s), nil
}

func regexpRESplit(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	if numArgs != 2 && numArgs != 3 {
		return core.Undefined, errs.NewWrongNumArgumentsError("regexp.re_split", "2 or 3", numArgs)
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_split", "first", "string(compatible)", args[0].TypeName())
	}

	s2, ok := args[1].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_split", "second", "string(compatible)", args[1].TypeName())
	}

	var i3 = -1
	if numArgs > 2 {
		var i3t int64
		i3t, ok = args[2].AsInt()
		i3 = int(i3t)
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_split", "third", "int(compatible)", args[2].TypeName())
		}
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "regexp.re_split", err)
	}

	spl := re.Split(s2, i3)
	arr := make([]core.Value, 0, len(spl))
	for _, s := range spl {
		arr = append(arr, core.NewStringValue(s))
	}

	return core.NewArrayValue(arr, false), nil
}

func regexpRECompile(vm core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("regexp.re_compile", "1", len(args))
	}

	s1, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("regexp.re_compile", "first", "string(compatible)", args[0].TypeName())
	}

	re, err := regexp.Compile(s1)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "regexp.re_compile", err)
	}

	return makeCompiledRegexp(vm, re)
}
