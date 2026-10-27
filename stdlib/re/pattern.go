package re

import (
	"regexp"
	"unsafe"

	"github.com/jokruger/kavun/core"
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

// A compiled regular expression, as `re.pattern(src)` answers it.
//
// ## Overview
//
// `re.pattern` is owned by the [`re`](../stdlib/re.md) module and built only by its constructor. It is immutable,
// its members are pure (the same pattern and text always answer the same result), and its text content is its
// source — so unlike the world handles (`fs.file`, …) it has `string()`, and `string(p)` works too. It renders as the
// expression that builds it.
//
// ```go
// re := import("re")
// p := re.pattern("([a-z]+)([0-9]+)")
// type_name(p)                 // => "re.pattern"
// format(p)                    // `re.pattern("([a-z]+)([0-9]+)")`
// string(p)                    // => "([a-z]+)([0-9]+)"
// p.match("ab12")              // => true
// p.replace("ab12 cd34", "$2$1")   // => "12ab 34cd"
// ```
//
// The syntax is Go's RE2: no backreferences or lookaround, and matching time is linear in the input — a pattern
// cannot be made to run away on hostile text.
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `==` / `!=` | the same source text (`re.pattern("a+") == re.pattern("a+")`) |
// | `!` | a pattern is always truthy |
// | everything else | raises |
//
//kavun:type re.pattern

// TypePattern is the re.pattern type descriptor.
var TypePattern = core.ValueTypeDescr{
	Name:     core.ConstHook(patternTypeName),                                          // PURE by contract
	String:   patternTypeString,                                                        // PURE by contract
	Format:   core.RenderFormat,                                                        // PURE by contract
	IsTrue:   core.Const2Hook[bool, error](true, nil),                                  // PURE by contract
	Equal:    patternTypeEqual,                                                         // PURE by contract
	AsString: func(v core.Value) (string, bool) { return PatternOf(v).String(), true }, // PURE by contract: the source

	Methods: patternMethods,
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
