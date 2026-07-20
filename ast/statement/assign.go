package statement

import (
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Assign represents an assignment statement.
type Assign struct {
	LHS      []ast.Expression
	RHS      []ast.Expression
	Token    token.Token
	TokenPos core.Pos
}

func (s *Assign) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Assign) Pos() core.Pos {
	return s.LHS[0].Pos()
}

// End returns the position of first character immediately after the node.
func (s *Assign) End() core.Pos {
	return s.RHS[len(s.RHS)-1].End()
}

func (s *Assign) String() string {
	lhs := make([]string, 0, len(s.LHS))
	for _, e := range s.LHS {
		lhs = append(lhs, e.String())
	}
	rhs := make([]string, 0, len(s.RHS))
	for _, e := range s.RHS {
		rhs = append(rhs, e.String())
	}
	return strings.Join(lhs, ", ") + " " + s.Token.String() + " " + strings.Join(rhs, ", ")
}
