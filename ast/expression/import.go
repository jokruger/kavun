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

func (e *Import) Pos() core.Pos {
	return e.TokenPos
}

func (e *Import) End() core.Pos {
	// import("moduleName")
	return core.Pos(int(e.TokenPos) + 10 + len(e.ModuleName))
}

func (e *Import) String() string {
	return `import("` + e.ModuleName + `")`
}

func (e *Import) IsUndefinedLiteral() bool {
	return false
}

func (e *Import) IsScalarLiteral() bool {
	return false
}

func (e *Import) IsCompositeLiteral() bool {
	return false
}

func (e *Import) IsCallExpression() bool {
	return false
}
