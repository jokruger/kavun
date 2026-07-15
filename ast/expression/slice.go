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

func (e *Slice) Pos() core.Pos {
	return e.Expr.Pos()
}

func (e *Slice) End() core.Pos {
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
	return false
}
