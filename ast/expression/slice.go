package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Slice represents a slice expression.
type Slice struct {
	Expr   ast.Expression
	LBrack core.Pos
	Low    ast.Expression
	High   ast.Expression
	Step   ast.Expression
	RBrack core.Pos
}

// Pos and End report the receiver's brackets for an indexed slice (arr[low:high]). A bare range literal
// (Expr == nil, e.g. "low..high") has no brackets, so they fall back to the Low/High/Step operands instead.

func (e *Slice) Pos() core.Pos {
	if e.Expr == nil {
		return e.Low.Pos()
	}
	return e.Expr.Pos()
}

func (e *Slice) End() core.Pos {
	if e.Expr == nil {
		if e.Step != nil {
			return e.Step.End()
		}
		return e.High.End()
	}
	return e.RBrack + 1
}

func (e *Slice) String() string {
	var low, high string
	if e.Low != nil {
		low = e.Low.String()
	}
	if e.High != nil {
		high = e.High.String()
	}
	if e.Expr == nil {
		// bare range literal: always renders as "low..high[:step]", never "low:high:step"
		if e.Step != nil {
			return low + ".." + high + ":" + e.Step.String()
		}
		return low + ".." + high
	}
	if e.Step != nil {
		return e.Expr.String() + "[" + low + ":" + high + ":" + e.Step.String() + "]"
	}
	return e.Expr.String() + "[" + low + ":" + high + "]"
}

func (e *Slice) IsUndefinedLiteral() bool {
	return false
}

func (e *Slice) IsScalarLiteral() bool {
	return false
}

func (e *Slice) IsCompositeLiteral() bool {
	return false
}

func (e *Slice) IsCallExpression() bool {
	// A bare range literal (Expr == nil) de-sugars to a call to the range() builtin.
	return e.Expr == nil
}

func (e *Slice) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
