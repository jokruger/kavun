package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Return represents a return statement.
type Return struct {
	ReturnPos core.Pos
	Result    ast.Expression
}

func (s *Return) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Return) Pos() core.Pos {
	return s.ReturnPos
}

// End returns the position of first character immediately after the node.
func (s *Return) End() core.Pos {
	if s.Result != nil {
		return s.Result.End()
	}
	return s.ReturnPos + 6
}

func (s *Return) String() string {
	if s.Result != nil {
		return "return " + s.Result.String()
	}
	return "return"
}
