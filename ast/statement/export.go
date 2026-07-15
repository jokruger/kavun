package statement

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Export represents an export statement.
type Export struct {
	ExportPos core.Pos
	Result    ast.Expression
}

func (s *Export) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Export) Pos() core.Pos {
	return s.ExportPos
}

// End returns the position of first character immediately after the node.
func (s *Export) End() core.Pos {
	return s.Result.End()
}

func (s *Export) String() string {
	return "export " + s.Result.String()
}
