package re

import (
	"regexp"
	"unsafe"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
)

// re.pattern is a compiled regular expression, owned by the re module — built only by re.pattern(src), named after
// its constructor — and registered by it (core.ModuleBuilder.Type). It is immutable and its members are pure; two
// patterns are equal when their sources are. Its text content is the source, so it has string(), and it renders as
// the expression that builds it.

const patternTypeName = "re.pattern"

// NewPatternValue wraps a compiled expression.
func NewPatternValue(r *regexp.Regexp) core.Value {
	return core.Value{Type: value.RePattern, Immutable: true, Ptr: unsafe.Pointer(r)}
}

// PatternOf unwraps a re.pattern value.
func PatternOf(v core.Value) *regexp.Regexp {
	return (*regexp.Regexp)(v.Ptr)
}

// TypePattern is the re.pattern type descriptor.
var TypePattern = core.ValueTypeDescr{
	Name:     core.ConstHook(patternTypeName),                                          // PURE by contract
	String:   patternTypeString,                                                        // PURE by contract
	Format:   core.RenderFormat,                                                        // PURE by contract
	IsTrue:   core.Const2Hook[bool, error](true, nil),                                  // PURE by contract
	Equal:    patternTypeEqual,                                                         // PURE by contract
	AsString: func(v core.Value) (string, bool) { return PatternOf(v).String(), true }, // PURE by contract: the source

	Methods: []core.Method{
		members.IsTrue:  core.NewPureMethod(core.MemberIsTrue, 0, core.Fixed),
		members.String:  core.NewPureMethod(patternString, 0, core.Variadic),
		members.Format:  core.NewPureMethod(core.MemberFormat, 0, core.Variadic),
		members.Copy:    core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Freeze:  core.NewPureMethod(core.MemberSelf, 0, core.Fixed),
		members.Find:    core.NewPureMethod(patternFind, 1, core.Variadic),
		members.Replace: core.NewPureMethod(patternReplace, 2, core.Fixed),
		members.Split:   core.NewPureMethod(patternSplit, 1, core.Variadic),
		members.Match:   core.NewPureMethod(patternMatch, 1, core.Fixed),
	},
}

// A pattern renders as the expression that builds it: re.pattern("[0-9]+").
func patternTypeString(v core.Value) string {
	return patternTypeName + "(" + core.NewStringValue(PatternOf(v).String()).String() + ")"
}

func patternTypeEqual(v core.Value, other core.Value, final bool) bool {
	if other.Type == value.RePattern {
		return PatternOf(v).String() == PatternOf(other).String()
	}
	if final {
		return false
	}
	return core.ValueTypes[other.Type].Equal(other, v, true)
}
