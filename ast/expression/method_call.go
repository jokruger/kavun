package expression

import (
	"strings"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// MethodCall represents a method call expression.
type MethodCall struct {
	Object     ast.Expression
	MethodName string
	MethodPos  core.Pos
	LParen     core.Pos
	Args       []ast.Expression
	Ellipsis   core.Pos
	RParen     core.Pos
}

func (e *MethodCall) Pos() core.Pos {
	return e.Object.Pos()
}

func (e *MethodCall) End() core.Pos {
	return e.RParen + 1
}

func (e *MethodCall) String() string {
	args := make([]string, 0, len(e.Args))
	for _, a := range e.Args {
		args = append(args, a.String())
	}
	if len(args) > 0 && e.Ellipsis.IsValid() {
		args[len(args)-1] = args[len(args)-1] + "..."
	}
	return e.Object.String() + "." + e.MethodName + "(" + strings.Join(args, ", ") + ")"
}

func (e *MethodCall) IsUndefinedLiteral() bool {
	return false
}

func (e *MethodCall) IsScalarLiteral() bool {
	return false
}

func (e *MethodCall) IsCompositeLiteral() bool {
	return false
}

func (e *MethodCall) IsCallExpression() bool {
	return true
}

func (e *MethodCall) LiteralToValue() (core.Value, bool) {
	return core.Undefined, false
}
