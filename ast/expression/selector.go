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

func (e *Selector) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Selector) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Selector) End() core.Pos {
	return e.Sel.End()
}

func (e *Selector) String() string {
	return e.Expr.String() + "." + e.Sel.String()
}
