package parser

import (
	"strings"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

// Expr represents an expression node in the AST.
type Expr interface {
	Node
	exprNode()
}

// ArrayLit represents an array literal.
type ArrayLit struct {
	Elements []Expr
	LBrack   core.Pos
	RBrack   core.Pos
}

func (e *ArrayLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ArrayLit) Pos() core.Pos {
	return e.LBrack
}

// End returns the position of first character immediately after the node.
func (e *ArrayLit) End() core.Pos {
	return e.RBrack + 1
}

func (e *ArrayLit) String() string {
	elements := make([]string, 0, len(e.Elements))
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// BadExpr represents a bad expression.
type BadExpr struct {
	From core.Pos
	To   core.Pos
}

func (e *BadExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BadExpr) Pos() core.Pos {
	return e.From
}

// End returns the position of first character immediately after the node.
func (e *BadExpr) End() core.Pos {
	return e.To
}

func (e *BadExpr) String() string {
	return "<bad expression>"
}

// BinaryExpr represents a binary operator expression.
type BinaryExpr struct {
	LHS      Expr
	RHS      Expr
	Token    token.Token
	TokenPos core.Pos
}

func (e *BinaryExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BinaryExpr) Pos() core.Pos {
	return e.LHS.Pos()
}

// End returns the position of first character immediately after the node.
func (e *BinaryExpr) End() core.Pos {
	return e.RHS.End()
}

func (e *BinaryExpr) String() string {
	return "(" + e.LHS.String() + " " + e.Token.String() + " " + e.RHS.String() + ")"
}

// BoolLit represents a boolean literal.
type BoolLit struct {
	Value    bool
	ValuePos core.Pos
	Literal  string
}

func (e *BoolLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *BoolLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *BoolLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *BoolLit) String() string {
	return e.Literal
}

// CallExpr represents a function call expression.
type CallExpr struct {
	Func     Expr
	LParen   core.Pos
	Args     []Expr
	Ellipsis core.Pos
	RParen   core.Pos
}

func (e *CallExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CallExpr) Pos() core.Pos {
	return e.Func.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CallExpr) End() core.Pos {
	return e.RParen + 1
}

func (e *CallExpr) String() string {
	args := make([]string, 0, len(e.Args))
	for _, e := range e.Args {
		args = append(args, e.String())
	}
	if len(args) > 0 && e.Ellipsis.IsValid() {
		args[len(args)-1] = args[len(args)-1] + "..."
	}
	return e.Func.String() + "(" + strings.Join(args, ", ") + ")"
}

// MethodCallExpr represents a method call expression.
type MethodCallExpr struct {
	Object     Expr
	MethodName string
	MethodPos  core.Pos
	LParen     core.Pos
	Args       []Expr
	Ellipsis   core.Pos
	RParen     core.Pos
}

func (e *MethodCallExpr) exprNode() {}

func (e *MethodCallExpr) Pos() core.Pos {
	return e.Object.Pos()
}

func (e *MethodCallExpr) End() core.Pos {
	return e.RParen + 1
}

func (e *MethodCallExpr) String() string {
	args := make([]string, 0, len(e.Args))
	for _, a := range e.Args {
		args = append(args, a.String())
	}
	if len(args) > 0 && e.Ellipsis.IsValid() {
		args[len(args)-1] = args[len(args)-1] + "..."
	}
	return e.Object.String() + "." + e.MethodName + "(" + strings.Join(args, ", ") + ")"
}

// CharLit represents a character literal.
type CharLit struct {
	Value    rune
	ValuePos core.Pos
	Literal  string
}

func (e *CharLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CharLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *CharLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *CharLit) String() string {
	return e.Literal
}

// CondExpr represents a ternary conditional expression.
type CondExpr struct {
	Cond        Expr
	True        Expr
	False       Expr
	QuestionPos core.Pos
	ColonPos    core.Pos
}

func (e *CondExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *CondExpr) Pos() core.Pos {
	return e.Cond.Pos()
}

// End returns the position of first character immediately after the node.
func (e *CondExpr) End() core.Pos {
	return e.False.End()
}

func (e *CondExpr) String() string {
	return "(" + e.Cond.String() + " ? " + e.True.String() + " : " + e.False.String() + ")"
}

// FloatLit represents a floating point literal.
type FloatLit struct {
	Value    float64
	ValuePos core.Pos
	Literal  string
}

func (e *FloatLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FloatLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *FloatLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *FloatLit) String() string {
	return e.Literal
}

// DecimalLit represents a decimal literal.
type DecimalLit struct {
	Value    core.Decimal
	ValuePos core.Pos
	Literal  string
}

func (e *DecimalLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *DecimalLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *DecimalLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *DecimalLit) String() string {
	return e.Literal
}

// FuncLit represents a function literal.
type FuncLit struct {
	Type *FuncType
	Body *BlockStmt
}

func (e *FuncLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncLit) Pos() core.Pos {
	return e.Type.Pos()
}

// End returns the position of first character immediately after the node.
func (e *FuncLit) End() core.Pos {
	return e.Body.End()
}

func (e *FuncLit) String() string {
	return "func" + e.Type.Params.String() + " " + e.Body.String()
}

// FuncType represents a function type definition.
type FuncType struct {
	FuncPos core.Pos
	Params  *IdentList
}

func (e *FuncType) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *FuncType) Pos() core.Pos {
	return e.FuncPos
}

// End returns the position of first character immediately after the node.
func (e *FuncType) End() core.Pos {
	return e.Params.End()
}

func (e *FuncType) String() string {
	return "func" + e.Params.String()
}

// Ident represents an identifier.
type Ident struct {
	Name    string
	NamePos core.Pos
}

func (e *Ident) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *Ident) Pos() core.Pos {
	return e.NamePos
}

// End returns the position of first character immediately after the node.
func (e *Ident) End() core.Pos {
	return core.Pos(int(e.NamePos) + len(e.Name))
}

func (e *Ident) String() string {
	if e != nil {
		return e.Name
	}
	return nullRep
}

// ImmutableExpr represents an immutable expression
type ImmutableExpr struct {
	Expr   Expr
	IPos   core.Pos
	LParen core.Pos
	RParen core.Pos
}

func (e *ImmutableExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ImmutableExpr) Pos() core.Pos {
	return e.IPos
}

// End returns the position of first character immediately after the node.
func (e *ImmutableExpr) End() core.Pos {
	return e.RParen
}

func (e *ImmutableExpr) String() string {
	return "immutable(" + e.Expr.String() + ")"
}

// ImportExpr represents an import expression
type ImportExpr struct {
	ModuleName string
	Token      token.Token
	TokenPos   core.Pos
}

func (e *ImportExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ImportExpr) Pos() core.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *ImportExpr) End() core.Pos {
	// import("moduleName")
	return core.Pos(int(e.TokenPos) + 10 + len(e.ModuleName))
}

func (e *ImportExpr) String() string {
	return `import("` + e.ModuleName + `")`
}

// IndexExpr represents an index expression.
type IndexExpr struct {
	Expr   Expr
	LBrack core.Pos
	Index  Expr
	RBrack core.Pos
}

func (e *IndexExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IndexExpr) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *IndexExpr) End() core.Pos {
	return e.RBrack + 1
}

func (e *IndexExpr) String() string {
	var index string
	if e.Index != nil {
		index = e.Index.String()
	}
	return e.Expr.String() + "[" + index + "]"
}

// IntLit represents an integer literal.
type IntLit struct {
	Value    int64
	ValuePos core.Pos
	Literal  string
}

func (e *IntLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IntLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *IntLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *IntLit) String() string {
	return e.Literal
}

// RecordElementLit represents a record element.
type RecordElementLit struct {
	Key      string
	KeyPos   core.Pos
	ColonPos core.Pos
	Value    Expr
}

func (e *RecordElementLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RecordElementLit) Pos() core.Pos {
	return e.KeyPos
}

// End returns the position of first character immediately after the node.
func (e *RecordElementLit) End() core.Pos {
	return e.Value.End()
}

func (e *RecordElementLit) String() string {
	return e.Key + ": " + e.Value.String()
}

// RecordLit represents a record literal.
type RecordLit struct {
	LBrace   core.Pos
	Elements []*RecordElementLit
	RBrace   core.Pos
}

func (e *RecordLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *RecordLit) Pos() core.Pos {
	return e.LBrace
}

// End returns the position of first character immediately after the node.
func (e *RecordLit) End() core.Pos {
	return e.RBrace + 1
}

func (e *RecordLit) String() string {
	elements := make([]string, 0, len(e.Elements))
	for _, m := range e.Elements {
		elements = append(elements, m.String())
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

// ParenExpr represents a parenthesis wrapped expression.
type ParenExpr struct {
	Expr   Expr
	LParen core.Pos
	RParen core.Pos
}

func (e *ParenExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *ParenExpr) Pos() core.Pos {
	return e.LParen
}

// End returns the position of first character immediately after the node.
func (e *ParenExpr) End() core.Pos {
	return e.RParen + 1
}

func (e *ParenExpr) String() string {
	return "(" + e.Expr.String() + ")"
}

// SelectorExpr represents a selector expression.
type SelectorExpr struct {
	Expr Expr
	Sel  Expr
}

func (e *SelectorExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SelectorExpr) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SelectorExpr) End() core.Pos {
	return e.Sel.End()
}

func (e *SelectorExpr) String() string {
	return e.Expr.String() + "." + e.Sel.String()
}

// SliceExpr represents a slice expression.
type SliceExpr struct {
	Expr   Expr
	LBrack core.Pos
	Low    Expr
	High   Expr
	RBrack core.Pos
}

func (e *SliceExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *SliceExpr) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *SliceExpr) End() core.Pos {
	return e.RBrack + 1
}

func (e *SliceExpr) String() string {
	var low, high string
	if e.Low != nil {
		low = e.Low.String()
	}
	if e.High != nil {
		high = e.High.String()
	}
	return e.Expr.String() + "[" + low + ":" + high + "]"
}

// StringLit represents a string literal.
type StringLit struct {
	Value    string
	ValuePos core.Pos
	Literal  string
}

func (e *StringLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *StringLit) Pos() core.Pos {
	return e.ValuePos
}

// End returns the position of first character immediately after the node.
func (e *StringLit) End() core.Pos {
	return core.Pos(int(e.ValuePos) + len(e.Literal))
}

func (e *StringLit) String() string {
	return e.Literal
}

// UnaryExpr represents an unary operator expression.
type UnaryExpr struct {
	Expr     Expr
	Token    token.Token
	TokenPos core.Pos
}

func (e *UnaryExpr) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UnaryExpr) Pos() core.Pos {
	return e.Expr.Pos()
}

// End returns the position of first character immediately after the node.
func (e *UnaryExpr) End() core.Pos {
	return e.Expr.End()
}

func (e *UnaryExpr) String() string {
	return "(" + e.Token.String() + e.Expr.String() + ")"
}

// UndefinedLit represents an undefined literal.
type UndefinedLit struct {
	TokenPos core.Pos
}

func (e *UndefinedLit) exprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *UndefinedLit) Pos() core.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *UndefinedLit) End() core.Pos {
	return e.TokenPos + 9 // len(undefined) == 9
}

func (e *UndefinedLit) String() string {
	return "undefined"
}
