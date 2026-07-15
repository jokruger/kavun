package composite

import (
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Array represents an array literal.
type Array struct {
	Elements []ast.Expression
	LBrack   core.Pos
	RBrack   core.Pos
}

func (e *Array) Pos() core.Pos {
	return e.LBrack
}

func (e *Array) End() core.Pos {
	return e.RBrack + 1
}

func (e *Array) String() string {
	elements := make([]string, 0, len(e.Elements))
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

func (e *Array) IsUndefinedLiteral() bool {
	return false
}

func (e *Array) IsScalarLiteral() bool {
	return false
}

func (e *Array) IsCompositeLiteral() bool {
	return true
}

func (e *Array) IsCallExpression() bool {
	return false
}
