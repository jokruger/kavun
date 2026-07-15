package ast

import "github.com/jokruger/kavun/core"

// Node represents a node in the AST.
type Node interface {
	Pos() core.Pos  // Returns the position of first character belonging to the node.
	End() core.Pos  // Returns the position of first character immediately after the node.
	String() string // Returns a string representation of the node.
}
