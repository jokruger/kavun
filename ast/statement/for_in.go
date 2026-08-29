package statement

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// ForIn represents a for-in statement.
type ForIn struct {
	ForPos   core.Pos
	Key      ast.Identifier
	Value    ast.Identifier
	Iterable ast.Expression
	Body     *Block
	// Solo marks the single-variable form (`for x in seq`), which binds the container's
	// ELEMENT — the value everywhere except maps, whose element is the key. The
	// two-variable form always binds (key, value).
	Solo bool
}

func (s *ForIn) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ForIn) Pos() core.Pos {
	return s.ForPos
}

// End returns the position of first character immediately after the node.
func (s *ForIn) End() core.Pos {
	return s.Body.End()
}

func (s *ForIn) String() string {
	if s.Value != nil {
		return "for " + s.Key.String() + ", " + s.Value.String() + " in " + s.Iterable.String() + " " + s.Body.String()
	}
	return "for " + s.Key.String() + " in " + s.Iterable.String() + " " + s.Body.String()
}
