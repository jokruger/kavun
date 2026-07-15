package expression

import "github.com/jokruger/kavun/core"

// Identifier represents an identifier.
type Identifier struct {
	Name    string
	NamePos core.Pos
}

func (e *Identifier) IdentifierNode() {}

func (e *Identifier) Pos() core.Pos {
	return e.NamePos
}

func (e *Identifier) End() core.Pos {
	return core.Pos(int(e.NamePos) + len(e.Name))
}

func (e *Identifier) String() string {
	if e != nil {
		return e.Name
	}
	return ""
}

func (e *Identifier) IsUndefinedLiteral() bool {
	return false
}

func (e *Identifier) IsScalarLiteral() bool {
	return false
}

func (e *Identifier) IsCompositeLiteral() bool {
	return false
}

func (e *Identifier) IsCallExpression() bool {
	return false
}
