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

func (e *Array) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Array) Pos() core.Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
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
