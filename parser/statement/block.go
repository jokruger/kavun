package statement

import (
	"strings"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Block represents a block statement.
type Block struct {
	Stmts  []ast.Statement
	LBrace core.Pos
	RBrace core.Pos
}

func (s *Block) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Block) Pos() core.Pos {
	return s.LBrace
}

// End returns the position of first character immediately after the node.
func (s *Block) End() core.Pos {
	return s.RBrace + 1
}

func (s *Block) String() string {
	list := make([]string, 0, len(s.Stmts))
	for _, e := range s.Stmts {
		list = append(list, e.String())
	}
	return "{" + strings.Join(list, "; ") + "}"
}
