package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/parser/ast"
)

// FStringPart is a single segment of an f-string.
// Exactly one of Literal or Expr is set: when Expr == nil the part is a verbatim literal text; when Expr != nil the
// part is an interpolation that must be Format()-ed with the pre-parsed Spec at run time.
type FStringPart struct {
	// Literal text segment. Used when Expr == nil. Already unescaped from the f-string body (with `{{` / `}}` collapsed
	// to single braces and the usual `\n`, `\"`, ... escapes processed).
	Literal string

	// Interpolated expression (parsed Kavun expression). Nil for literal segments.
	Expr ast.Expression

	// Pre-parsed format spec for the interpolation. Always valid for static interpolation parts; for literal parts and
	// dynamic-spec interpolation parts it is the zero FormatSpec.
	Spec fspec.FormatSpec

	// Original spec text (the substring after the `:` inside `{...}`), without leading colon. Empty when no `:` was
	// present or when the fspec was empty. For dynamic specs this is the raw template (including `{...}` placeholders);
	// it is only used for de-duplication and disassembly.
	SpecText string

	// Dynamic format-spec template. Set only when the spec contains nested `{expr}` placeholders. The runtime spec
	// string is built by interleaving SpecLiterals[i] with str(SpecExprs[i]) and ending with
	// SpecLiterals[len(SpecExprs)]. When SpecExprs is non-empty, Spec is unused and the spec is parsed at run time.
	SpecLiterals []string
	SpecExprs    []ast.Expression
}

// FString represents f-string literal: f"text {expr:fspec} ...".
// All format specs and literal text segments are resolved at parse time so the runtime cost of an f-string is the cost
// of its expression evaluations plus per-interpolation Format calls and string concatenation.
type FString struct {
	Parts    []FStringPart
	ValuePos core.Pos
	EndPos   core.Pos
	Literal  string // original source text, including surrounding quotes
}

func (e *FString) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FString) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FString) End() core.Pos {
	return e.EndPos
}

func (e *FString) String() string {
	return "f" + e.Literal
}
