package parser

import (
	"strings"

	"github.com/jokruger/gs/core"
)

// File represents a file unit.
type File struct {
	InputFile *SourceFile
	Stmts     []Stmt
}

// Pos returns the position of first character belonging to the node.
func (n *File) Pos() core.Pos {
	return core.Pos(n.InputFile.Base)
}

// End returns the position of first character immediately after the node.
func (n *File) End() core.Pos {
	return core.Pos(n.InputFile.Base + n.InputFile.Size)
}

func (n *File) String() string {
	stmts := make([]string, 0, len(n.Stmts))
	for _, e := range n.Stmts {
		stmts = append(stmts, e.String())
	}
	return strings.Join(stmts, "; ")
}
