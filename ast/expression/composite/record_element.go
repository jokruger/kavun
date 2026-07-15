package composite

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// RecordElement represents a record element.
type RecordElement struct {
	Key      string
	KeyPos   core.Pos
	ColonPos core.Pos
	Value    ast.Expression
}

func (e *RecordElement) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RecordElement) Pos() core.Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *RecordElement) End() core.Pos {
	return e.Value.End()
}

func (e *RecordElement) String() string {
	return e.Key + ": " + e.Value.String()
}
