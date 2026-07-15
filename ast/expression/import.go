package expression

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
)

// Import represents an import expression
type Import struct {
	ModuleName string
	Token      token.Token
	TokenPos   core.Pos
}

func (e *Import) ExpressionNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Import) Pos() core.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *Import) End() core.Pos {
	// import("moduleName")
	return core.Pos(int(e.TokenPos) + 10 + len(e.ModuleName))
}

func (e *Import) String() string {
	return `import("` + e.ModuleName + `")`
}
