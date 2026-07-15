package ast

import "github.com/jokruger/kavun/core"

// Expression represents an expression node in the AST.
type Expression interface {
	Node
	IsUndefinedLiteral() bool
	IsScalarLiteral() bool
	IsCompositeLiteral() bool
	IsCallExpression() bool
	LiteralToValue() (core.Value, bool)
}

// Statement represents a statement in the AST.
type Statement interface {
	Node
	StatementNode()
}

// Identifier represents an identifier node in the AST.
type Identifier interface {
	Expression
	IdentifierNode()
}
