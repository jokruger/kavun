package statement

import "github.com/jokruger/kavun/core"

// Invalid represents invalid statement.
type Invalid struct {
	From core.Pos
	To   core.Pos
}

func (s *Invalid) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Invalid) Pos() core.Pos {
	return s.From
}

// End returns the position of first character immediately after the node.
func (s *Invalid) End() core.Pos {
	return s.To
}

func (s *Invalid) String() string {
	return "<bad statement>"
}
