package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Range represents a bare range literal: "low..high" or "low..high:step". Unlike Slice, it has no receiver — it is
// a value-producing expression, sugar for range(low, high[, step]), not an operation on a container. It is a
// distinct node type from Slice (rather than a Slice with a nil Expr) so that the two forms keep their own String()
// rendering and so the optimizer/compiler never have to guard on whether a receiver is present.
type Range struct {
	Low  ast.Expression
	High ast.Expression
	Step ast.Expression
}

func (e *Range) Pos() core.Pos {
	return e.Low.Pos()
}

func (e *Range) End() core.Pos {
	if e.Step != nil {
		return e.Step.End()
	}
	return e.High.End()
}

func (e *Range) String() string {
	if e.Step != nil {
		return e.Low.String() + ".." + e.High.String() + ":" + e.Step.String()
	}
	return e.Low.String() + ".." + e.High.String()
}

func (e *Range) IsUndefinedLiteral() bool {
	return false
}

func (e *Range) IsScalarLiteral() bool {
	return false
}

func (e *Range) IsCompositeLiteral() bool {
	return false
}

func (e *Range) IsCallExpression() bool {
	// De-sugars to a call to the range() builtin.
	return true
}

func (e *Range) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
