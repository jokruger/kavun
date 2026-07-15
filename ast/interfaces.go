package ast

// Expression represents an expression node in the AST.
type Expression interface {
	Node
	IsUndefinedLiteral() bool
	IsScalarLiteral() bool
	IsCompositeLiteral() bool
	IsCallExpression() bool
}

// Statement represents a statement in the AST.
type Statement interface {
	Node
	StatementNode()
}
