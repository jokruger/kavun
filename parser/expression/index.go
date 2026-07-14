package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Index represents an index expression.
type Index struct {
	Expr   ast.Expression
	LBrack core.Pos
	Index  ast.Expression
	RBrack core.Pos
}

func (e *Index) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Index) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Index) End() core.Pos {
	return e.RBrack + 1
}

func (e *Index) String() string {
	var index string
	if e.Index != nil {
		index = e.Index.String()
	}
	return e.Expr.String() + "[" + index + "]"
}
