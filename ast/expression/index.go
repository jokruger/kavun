package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Index represents an index expression.
type Index struct {
	Expr   ast.Expression
	LBrack core.Pos
	Index  ast.Expression
	RBrack core.Pos
}

func (e *Index) Pos() core.Pos {
	return e.Expr.Pos()
}

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

func (e *Index) IsUndefinedLiteral() bool {
	return false
}

func (e *Index) IsScalarLiteral() bool {
	return false
}

func (e *Index) IsCompositeLiteral() bool {
	return false
}

func (e *Index) IsCallExpression() bool {
	return false
}

func (e *Index) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
