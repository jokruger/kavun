package statement

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Expression represents an expression statement.
type Expression struct {
	Expr ast.Expression
}

func (s *Expression) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Expression) Pos() core.Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *Expression) End() core.Pos {
	return s.Expr.End()
}

func (s *Expression) String() string {
	return s.Expr.String()
}
