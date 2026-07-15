package expression

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

func (e *Identifiers) Pos() core.Pos {
	if e.LParen.IsValid() {
		return e.LParen
	}
	if len(e.List) > 0 {
		return e.List[0].Pos()
	}
	return core.NoPos
}

func (e *Identifiers) End() core.Pos {
	if e.RParen.IsValid() {
		return e.RParen + 1
	}
	if l := len(e.List); l > 0 {
		return e.List[l-1].End()
	}
	return core.NoPos
}

func (e *Identifiers) NumFields() int {
	if e == nil {
		return 0
	}
	return len(e.List)
}

func (e *Identifiers) String() string {
	list := make([]string, 0, len(e.List))
	for i, x := range e.List {
		if e.VarArgs && i == len(e.List)-1 {
			list = append(list, "..."+x.String())
		} else {
			list = append(list, x.String())
		}
	}
	return "(" + strings.Join(list, ", ") + ")"
}

func (e *Identifiers) IsUndefinedLiteral() bool {
	return false
}

func (e *Identifiers) IsScalarLiteral() bool {
	return false
}

func (e *Identifiers) IsCompositeLiteral() bool {
	return false
}

func (e *Identifiers) IsCallExpression() bool {
	return false
}

func (e *Identifiers) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
