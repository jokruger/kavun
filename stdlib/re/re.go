package re

import (
	"regexp"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The re module is regular expressions. Its one function is the constructor of the type it owns: re.pattern(src)
// compiles once, and the pattern's members match, find, replace and split. Pure: importable without a grant.
func init() {
	m := core.NewModule(value.ModuleRe, "re", true)
	m.Type(value.RePattern, TypePattern)
	m.PureFn(members.Pattern, 0, rePattern, 1, core.Fixed)
	// 1..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

// re.pattern(src): the compiled pattern; a malformed expression raises invalid_value.
func rePattern(_ core.VM, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError("re.pattern", "1", len(args))
	}
	src, ok := args[0].AsString()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("re.pattern", "first", "string(compatible)", args[0].TypeName())
	}
	r, err := regexp.Compile(src)
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "re.pattern", err)
	}
	return NewPatternValue(r), nil
}
