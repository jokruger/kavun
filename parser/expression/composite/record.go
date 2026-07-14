package composite

import (
	"strings"

	"github.com/jokruger/kavun/core"
)

// Record represents a record literal.
type Record struct {
	LBrace   core.Pos
	Elements []*RecordElement
	RBrace   core.Pos
}

func (e *Record) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Record) Pos() core.Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *Record) End() core.Pos {
	return e.RBrace + 1
}

func (e *Record) String() string {
	elements := make([]string, 0, len(e.Elements))
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "{" + strings.Join(elements, ", ") + "}"
}
