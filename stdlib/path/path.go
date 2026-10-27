package path

import (
	"fmt"
	"path/filepath"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// Path TEXT with the host's separator — joining, splitting, cleaning.
//
// It never touches the file system (that is [`fs`](fs.md)), so it is pure and importable by default.
//
// ```go
// path := import("path")
// path.join("a", "b", "../c")     // => "a/c"
// path.split("/x/y.txt")          // => ["/x/", "y.txt"]
// ```
//
// Failures: `invalid_value` (`rel`, `match`), `invalid_argument_type` (a non-string), `wrong_num_arguments`.
//
//kavun:module path pure

// pathArgs reads the string arguments of a fixed-count path function (one or two: the dispatcher has checked the
// count against the template).
func pathArgs(name string, args []core.Value) ([]string, error) {
	out := make([]string, len(args))
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

// Join the parts with the separator and clean the result; `""` for no parts.
//
//kavun:fn join slot=0 pure (...parts) -> string
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

// Answer the last element (`"."` for `""`).
//
// A root answers the separator itself.
//
//kavun:fn base slot=1 pure (p) -> string
func pathBase(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.base", args)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Base(p[0])), nil
}

// Answer everything but the last element, cleaned.
//
//kavun:fn dir slot=2 pure (p) -> string
func pathDir(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.dir", args)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Dir(p[0])), nil
}

// Answer the extension of the last element, dot included (`""` when none).
//
//kavun:fn ext slot=3 pure (p) -> string
func pathExt(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.ext", args)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Ext(p[0])), nil
}

// Answer the shortest equivalent path.
//
// The result has no `.` or `..` elements and no doubled separators.
//
//kavun:fn clean slot=4 pure (p) -> string
func pathClean(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.clean", args)
	if err != nil {
		return core.Undefined, err
	}
	return core.NewStringValue(filepath.Clean(p[0])), nil
}

// Split `p` into directory and file, `dir` keeping its trailing separator so `dir + file == p`.
//
//kavun:fn split slot=5 pure (p) -> [dir, file]
func pathSplit(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.split", args)
	if err != nil {
		return core.Undefined, err
	}
	dir, file := filepath.Split(p[0])
	return core.NewArrayValue([]core.Value{core.NewStringValue(dir), core.NewStringValue(file)}, false), nil
}

// Answer `target` relative to `base`; raises `invalid_value` when there is none.
//
// There is no such path when one of `base` and `target` is absolute and the other is not.
//
//kavun:fn rel slot=6 pure (base, target) -> string
func pathRel(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.rel", args)
	if err != nil {
		return core.Undefined, err
	}
	r, err := filepath.Rel(p[0], p[1])
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "path.rel", err)
	}
	return core.NewStringValue(r), nil
}

// Report whether `p` is absolute.
//
//kavun:fn is_abs slot=7 pure (p) -> bool
func pathIsAbs(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.is_abs", args)
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(filepath.IsAbs(p[0])), nil
}

// Report whether `name` matches the shell `pattern` (`*`, `?`, `[...]`).
//
// A malformed pattern raises `invalid_value`.
//
//kavun:fn match slot=8 pure (pattern, name) -> bool
func pathMatch(_ core.VM, args []core.Value) (core.Value, error) {
	p, err := pathArgs("path.match", args)
	if err != nil {
		return core.Undefined, err
	}
	ok, err := filepath.Match(p[0], p[1])
	if err != nil {
		return modkit.RaiseGo(errs.KindInvalidValue, "path.match", err)
	}
	return core.BoolValue(ok), nil
}
