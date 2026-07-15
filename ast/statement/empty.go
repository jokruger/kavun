package statement

import "github.com/jokruger/kavun/core"

// Empty represents an empty statement.
type Empty struct {
	Semicolon core.Pos
	Implicit  bool
}

func (s *Empty) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *Empty) Pos() core.Pos {
	return s.Semicolon
}

// End returns the position of first character immediately after the node.
func (s *Empty) End() core.Pos {
	if s.Implicit {
		return s.Semicolon
	}
	return s.Semicolon + 1
}

func (s *Empty) String() string {
	return ";"
}
