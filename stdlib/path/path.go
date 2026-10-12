package path

import (
	"fmt"
	"path/filepath"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// The path module is path TEXT: joining, splitting and cleaning file paths with the host's separator. It never
// touches the file system (that is fs), so it is pure and importable without a grant.
func init() {
	m := core.NewModule(value.ModulePath, "path", true)
	m.PureFn(members.Join, 0, pathJoin, 0, core.Variadic)
	m.PureFn(members.Base, 1, pathBase, 1, core.Fixed)
	m.PureFn(members.Dir, 2, pathDir, 1, core.Fixed)
	m.PureFn(members.Ext, 3, pathExt, 1, core.Fixed)
	m.PureFn(members.Clean, 4, pathClean, 1, core.Fixed)
	m.PureFn(members.Split, 5, pathSplit, 1, core.Fixed)
	m.PureFn(members.Rel, 6, pathRel, 2, core.Fixed)
	m.PureFn(members.IsAbs, 7, pathIsAbs, 1, core.Fixed)
	m.PureFn(members.Match, 8, pathMatch, 2, core.Fixed)
	// 9..127 reserved
	if err := m.Register(); err != nil {
		panic(err)
	}
}

// pathArgs reads n string arguments of a path function.
func pathArgs(name string, args []core.Value, n int) ([]string, error) {
	if len(args) != n {
		return nil, errs.NewWrongNumArgumentsError(name, fmt.Sprint(n), len(args))
	}
	out := make([]string, n)
	for i, a := range args {
		if a.Type != value.String {
			return nil, errs.NewInvalidArgumentTypeError(name, ordinal(i), "string", a.TypeName())
		}
		out[i], _ = a.AsString()
	}
	return out, nil
}

func ordinal(i int) string {
	return [...]string{"first", "second"}[i]
}

// join(parts...): the parts joined with the separator and cleaned; "" for no parts.
func pathJoin(_ core.VM, args []core.Value) (core.Value, error) {
	parts := make([]string, len(args))
	for i, a := range args {
		if a.Type != value.String {
			return core.Undefined, errs.NewInvalidArgumentTypeError("path.join", fmt.Sprintf("parts[%d]", i), "string", a.TypeName())
		}
		parts[i], _ = a.AsString()
	}
	return core.NewStringValue(filepath.Join(parts...)), nil
}

// base(p): the last element ("." for "", the separator for a root).
func pathBase(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.base", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Base(p[0])), nil
}

// dir(p): everything but the last element, cleaned.
func pathDir(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.dir", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Dir(p[0])), nil
}

// ext(p): the extension of the last element, dot included ("" when none).
func pathExt(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.ext", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Ext(p[0])), nil
}

// clean(p): the shortest equivalent path (no "." or "..", no doubled separators).
func pathClean(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.clean", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Clean(p[0])), nil
}

// split(p): [dir, file], with dir keeping its trailing separator, so dir + file == p.
func pathSplit(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.split", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	dir, file := filepath.Split(p[0])
	return core.NewArrayValue([]core.Value{core.NewStringValue(dir), core.NewStringValue(file)}, false), nil
}

// rel(base, target): target relative to base; raises when there is no such path (one absolute, one not).
func pathRel(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.rel", args, 2)
	if err != nil {
		return core.Undefined, err
	}
	r, err := filepath.Rel(p[0], p[1])
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "path.rel", err)
	}
	return core.NewStringValue(r), nil
}

// is_abs(p): whether p is absolute.
func pathIsAbs(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.is_abs", args, 1)
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(filepath.IsAbs(p[0])), nil
}

// match(pattern, name): whether name matches the shell pattern (* ? [...]); a malformed pattern raises.
func pathMatch(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.match", args, 2)
	if err != nil {
		return core.Undefined, err
	}
	ok, err := filepath.Match(p[0], p[1])
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "path.match", err)
	}
	return core.BoolValue(ok), nil
}
