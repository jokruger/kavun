package ast

import "github.com/jokruger/kavun/core"

// Identifier represents an identifier.
type Identifier struct {
	Name    string
	NamePos core.Pos
}

func (e *Identifier) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Identifier) Pos() core.Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *Identifier) End() core.Pos {
	return core.Pos(int(e.NamePos) + len(e.Name))
}

func (e *Identifier) String() string {
	if e != nil {
		return e.Name
	}
	return "<null>"
}
