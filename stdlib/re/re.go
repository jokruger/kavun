package re

import (
	"regexp"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/stdlib/internal/modkit"
)

// Regular expressions.
//
// The module's one function is the constructor of the type it owns: `re.pattern(src)` compiles the expression once, and
// the [`re.pattern`](../types/re_pattern.md) answers `match`, `find`, `replace` and `split`. Pure: importable by
// default.
//
// ```go
// re := import("re")
// digits := re.pattern("[0-9]+")
// digits.match("abc123")              // => true
// digits.find("a1b22")                // => [[{"begin": 1, "end": 2, "text": "1"}]] — the first match, group 0 first
// digits.find("a1b22", -1).len()      // => 2 — every match
// digits.find("abc")                  // => [] — no match is an empty array, never undefined
// digits.replace("a1b22", "#")        // => "a#b#"
// re.pattern(",\\s*").split("a, b,c")  // => ["a", "b", "c"]
// ```
//
// The pattern's members (see [types/re_pattern.md](../types/re_pattern.md)):
//
// - `p.match(s) -> bool`: Whether the expression matches anywhere in `s`.
// - `p.find(s, count?) -> [match]`: The first match (or up to `count`; `-1` for all), each an array of frozen
// `{text, begin, end}` group records, group 0 first; `[]` when nothing matches.
// - `p.replace(s, repl) -> string`: Every match replaced; `$1` / `${name}` expand groups. A replacement that would
// grow past the sequence ceiling raises `invalid_value`.
// - `p.split(s, count?) -> [string]`: `s` cut at the matches.
// - `p.string() -> string`: The source text (the pattern IS text); `format(p)` renders `re.pattern("…")`.
//
// **Failures.** An invalid expression raises kind `invalid_value` — a bad pattern is a bad argument:
// `(re.pattern) error parsing regexp: missing closing ): `(``.
//
//kavun:module re pure

// moduleExtras declares what the //kavun:fn directives do not: the module's constants and owned types.
func moduleExtras(m *core.ModuleBuilder) {
	m.Type(value.RePattern, TypePattern)
}

// Compile the regular expression `src`; a malformed expression raises `invalid_value`.
//
// `src` is a `string`.
//
// The expression is compiled where `re.pattern(…)` runs — hoist a pattern used in a loop.
//
//kavun:fn pattern slot=0 pure (src) -> re.pattern
func rePattern(_ core.VM, args []core.Value) (core.Value, error) {
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
