package scalar

import "github.com/jokruger/kavun/core"

// Bytes represents a bytes string literal (b"...").
type Bytes struct {
	Value    []byte
	ValuePos core.Pos
	Literal  string
}

func (e *Bytes) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Bytes) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *Bytes) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal) + 1) // +1 for the 'b' prefix
}

func (e *Bytes) String() string {
	return "b" + e.Literal
}
