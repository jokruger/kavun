package expression

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Binary represents a binary operator expression.
type Binary struct {
	LHS      ast.Expression
	RHS      ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (e *Binary) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Binary) Pos() core.Pos {
	return e.LHS.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Binary) End() core.Pos {
	return e.RHS.End()
}

func (e *Binary) String() string {
	return "(" + e.LHS.String() + " " + e.Token.String() + " " + e.RHS.String() + ")"
}
