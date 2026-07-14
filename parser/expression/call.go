package expression

import (
	"strings"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// Call represents a function call expression.
type Call struct {
	Func     ast.Expression
	LParen   core.Pos
	Args     []ast.Expression
	Ellipsis core.Pos
	RParen   core.Pos
}

func (e *Call) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Call) Pos() core.Pos {
	return e.Func.Pos()
}

// End returns the position of first character immediately after the node.
func (e *Call) End() core.Pos {
	return e.RParen + 1
}

func (e *Call) String() string {
	args := make([]string, 0, len(e.Args))
	for _, e := range e.Args {
		args = append(args, e.String())
	}
	if len(args) > 0 && e.Ellipsis.IsValid() {
		args[len(args)-1] = args[len(args)-1] + "..."
	}
	return e.Func.String() + "(" + strings.Join(args, ", ") + ")"
}
