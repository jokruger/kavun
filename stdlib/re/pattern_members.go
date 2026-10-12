package re

import (
	"regexp"
	"strings"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// re.pattern's members: one function per entry of TypePattern.Methods that is not a shared member. All are pure:
// the same pattern and text always answer the same result.

func patternContext(id member.ID) string {
	return "re.pattern." + id.String()
}

// textArg reads the subject text of a pattern member.
func textArg(id member.ID, pos string, a core.Value) (string, error) {
	s, ok := a.AsString()
	if !ok {
		return "", errs.NewInvalidArgumentTypeError(patternContext(id), pos, "string(compatible)", a.TypeName())
	}
	return s, nil
}

// PURE by contract — string([default]): the source text
func patternString(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	return core.ConvMember(id.String(), patternTypeName, args, true, core.NewStringValue(PatternOf(v).String()))
}

// PURE by contract — match(s): whether the pattern matches anywhere in s
func patternMatch(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 {
		return core.Undefined, errs.NewWrongNumArgumentsError(patternContext(id), "1", len(args))
	}
	s, err := textArg(id, "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(PatternOf(v).MatchString(s)), nil
}

// PURE by contract — find(s[, count]): the matches, first (count < 0: all; without count: the first only), each an
// array of frozen {text, begin, end} records, group 0 first; [] when nothing matches
func patternFind(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(patternContext(id), "1 or 2", len(args))
	}
	s, err := textArg(id, "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	count := 1
	if len(args) == 2 {
		n, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError(patternContext(id), "second", "int(compatible)", args[1].TypeName())
		}
		count = int(max(min(n, 1<<31), -1))
	}
	matches := PatternOf(v).FindAllStringSubmatchIndex(s, count)
	out := make([]core.Value, 0, len(matches))
	for _, m := range matches {
		groups := make([]core.Value, 0, len(m)/2)
		for i := 0; i < len(m); i += 2 {
			if m[i] >= 0 && m[i+1] >= 0 {
				groups = append(groups, core.NewRecordValue(map[string]core.Value{
					"text":  core.NewStringValue(s[m[i]:m[i+1]]),
					"begin": core.IntValue(int64(m[i])),
					"end":   core.IntValue(int64(m[i+1])),
				}, true))
			}
		}
		out = append(out, core.NewArrayValue(groups, false))
	}
	return core.NewArrayValue(out, false), nil
}

// PURE by contract — replace(s, repl): every match replaced, $1 / ${name} expanding groups; a result past
// core.MaxSequenceLen raises
func patternReplace(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(patternContext(id), "2", len(args))
	}
	s, err := textArg(id, "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	repl, err := textArg(id, "second", args[1])
	if err != nil {
		return core.Undefined, err
	}
	out, ok := expandReplace(PatternOf(v), s, repl)
	if !ok {
		return core.Undefined, errs.NewInvalidValueError("(" + patternContext(id) + ") expansion would exceed the replacement size limit")
	}
	return core.NewStringValue(out), nil
}

// PURE by contract — split(s[, count]): s cut at the matches (count < 0 or absent: all pieces)
func patternSplit(_ core.VM, v core.Value, id member.ID, args []core.Value) (core.Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return core.Undefined, errs.NewWrongNumArgumentsError(patternContext(id), "1 or 2", len(args))
	}
	s, err := textArg(id, "first", args[0])
	if err != nil {
		return core.Undefined, err
	}
	count := -1
	if len(args) == 2 {
		n, ok := args[1].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError(patternContext(id), "second", "int(compatible)", args[1].TypeName())
		}
		count = int(max(min(n, 1<<31), -1))
	}
	parts := PatternOf(v).Split(s, count)
	out := make([]core.Value, 0, len(parts))
	for _, p := range parts {
		out = append(out, core.NewStringValue(p))
	}
	return core.NewArrayValue(out, false), nil
}

// expandReplace expands every match of re in src with repl. It answers false when the result would grow past
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
func expandReplace(re *regexp.Regexp, src, repl string) (string, bool) {
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
