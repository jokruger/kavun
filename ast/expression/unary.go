package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Unary represents an unary operator expression.
type Unary struct {
	Expr     ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (e *Unary) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Unary) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Unary) End() core.Pos {
	return e.Expr.End()
}

func (e *Unary) String() string {
	return "(" + e.Token.String() + e.Expr.String() + ")"
}
