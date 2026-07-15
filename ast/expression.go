package ast

// Expression represents an expression node in the AST.
type Expression interface {
	Node
	IsUndefinedLiteral() bool
	IsScalarLiteral() bool
	IsCompositeLiteral() bool
	IsCallExpression() bool
}
