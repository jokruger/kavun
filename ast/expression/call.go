package expression

import (
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Call represents a function call expression.
type Call struct {
	Func     ast.Expression
	LParen   core.Pos
	Args     []ast.Expression
	Ellipsis core.Pos
	RParen   core.Pos
}

func (e *Call) Pos() core.Pos {
	return e.Func.Pos()
}

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

func (e *Call) IsUndefinedLiteral() bool {
	return false
}

func (e *Call) IsScalarLiteral() bool {
	return false
}

func (e *Call) IsCompositeLiteral() bool {
	return false
}

func (e *Call) IsCallExpression() bool {
	return true
}

func (e *Call) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
