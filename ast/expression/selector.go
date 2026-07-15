package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Selector represents a selector expression.
type Selector struct {
	Expr ast.Expression
	Sel  ast.Expression
}

func (e *Selector) Pos() core.Pos {
	return e.Expr.Pos()
}

func (e *Selector) End() core.Pos {
	return e.Sel.End()
}

func (e *Selector) String() string {
	return e.Expr.String() + "." + e.Sel.String()
}

func (e *Selector) IsUndefinedLiteral() bool {
	return false
}

func (e *Selector) IsScalarLiteral() bool {
	return false
}

func (e *Selector) IsCompositeLiteral() bool {
	return false
}

func (e *Selector) IsCallExpression() bool {
	return false
}
