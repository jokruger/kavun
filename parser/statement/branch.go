package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/parser/ast"
)

// Branch represents a branch statement.
type Branch struct {
	Token    token.Token
	TokenPos core.Pos
	Label    *ast.Identifier
}

func (s *Branch) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Branch) Pos() core.Pos {
	return s.TokenPos
}

// End returns the position of first character immediately after the node.
func (s *Branch) End() core.Pos {
	if s.Label != nil {
		return s.Label.End()
	}

	return core.Pos(int(s.TokenPos) + len(s.Token.String()))
}

func (s *Branch) String() string {
	var label string
	if s.Label != nil {
		label = " " + s.Label.Name
	}
	return s.Token.String() + label
}
