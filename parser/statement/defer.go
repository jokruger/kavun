package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Defer represents a defer statement.
type Defer struct {
	DeferPos core.Pos
	Call     ast.Expression
}

func (s *Defer) StatementNode() {}

func (s *Defer) Pos() core.Pos {
	return s.DeferPos
}

func (s *Defer) End() core.Pos {
	return s.Call.End()
}

func (s *Defer) String() string {
	return "defer " + s.Call.String()
}
