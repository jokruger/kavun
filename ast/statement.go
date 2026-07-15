package ast

// Statement represents a statement in the AST.
type Statement interface {
	Node
	StatementNode()
}
