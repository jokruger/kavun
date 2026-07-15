package ast

import (
	"strings"

	"github.com/jokruger/kavun/core"
)

// Identifiers represents a list of identifiers.
type Identifiers struct {
	LParen  core.Pos
	VarArgs bool
	List    []*Identifier
	RParen  core.Pos
}

func (n *Identifiers) Pos() core.Pos {
	if n.LParen.IsValid() {
		return n.LParen
	}
	if len(n.List) > 0 {
		return n.List[0].Pos()
	}
	return core.NoPos
}

func (n *Identifiers) End() core.Pos {
	if n.RParen.IsValid() {
		return n.RParen + 1
	}
	if l := len(n.List); l > 0 {
		return n.List[l-1].End()
	}
	return core.NoPos
}

func (n *Identifiers) NumFields() int {
	if n == nil {
		return 0
	}
	return len(n.List)
}

func (n *Identifiers) String() string {
	list := make([]string, 0, len(n.List))
	for i, e := range n.List {
		if n.VarArgs && i == len(n.List)-1 {
			list = append(list, "..."+e.String())
		} else {
			list = append(list, e.String())
		}
	}
	return "(" + strings.Join(list, ", ") + ")"
}

func (n *Identifiers) IsUndefinedLiteral() bool {
	return false
}

func (n *Identifiers) IsScalarLiteral() bool {
	return false
}

func (n *Identifiers) IsCompositeLiteral() bool {
	return false
}

func (n *Identifiers) IsCallExpression() bool {
	return false
}
