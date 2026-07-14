package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/parser/ast"
)

// IncDec represents increment or decrement statement.
type IncDec struct {
	Expr     ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (s *IncDec) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *IncDec) Pos() core.Pos {
	return s.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (s *IncDec) End() core.Pos {
	return core.Pos(int(s.TokenPos) + 2)
}

func (s *IncDec) String() string {
	return s.Expr.String() + s.Token.String()
}
