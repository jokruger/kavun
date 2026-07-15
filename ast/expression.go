package ast

// Expression represents an expression node in the AST.
type Expression interface {
	Node
	ExpressionNode()
}
