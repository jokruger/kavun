package statement

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser/ast"
)

// If represents an if statement.
type If struct {
	IfPos core.Pos
	Init  ast.Statement
	Cond  ast.Expression
	Body  *Block
	Else  ast.Statement // else branch; or nil
}

func (s *If) StatementNode() {}

// Pos returns the position of first character belonging to the node.
func (s *If) Pos() core.Pos {
	return s.IfPos
}

// End returns the position of first character immediately after the node.
func (s *If) End() core.Pos {
	if s.Else != nil {
		return s.Else.End()
	}
	return s.Body.End()
}

func (s *If) String() string {
	var initStmt, elseStmt string
	if s.Init != nil {
		initStmt = s.Init.String() + "; "
	}
	if s.Else != nil {
		elseStmt = " else " + s.Else.String()
	}
	return "if " + initStmt + s.Cond.String() + " " + s.Body.String() + elseStmt
}
