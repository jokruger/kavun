package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// For represents a for statement.
type For struct {
	ForPos core.Pos
	Init   ast.Statement
	Cond   ast.Expression
	Post   ast.Statement
	Body   *Block
}

func (s *For) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *For) Pos() core.Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *For) End() core.Pos {
	return s.Body.End()
}

func (s *For) String() string {
	var init, cond, post string

	if s.Init != nil {
		init = s.Init.String()
	}

	if s.Cond != nil {
		cond = s.Cond.String() + " "
	}

	if s.Post != nil {
		post = s.Post.String()
	}

	if init != "" || post != "" {
		return "for " + init + " ; " + cond + " ; " + post + s.Body.String()
	}

	return "for " + cond + s.Body.String()
}
