package ast

import (
	"strings"

	"github.com/jokruger/kavun/core"
)

// File represents a file unit.
type File struct {
	InputFile *SourceFile
	Stmts     []Statement
}

func (n *File) Pos() core.Pos {
	return core.Pos(n.InputFile.Base)
}

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

func (n *File) Name() string {
	if n.InputFile != nil {
		return n.InputFile.Name
	}
	return "<nil>"
}
