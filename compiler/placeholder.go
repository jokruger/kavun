package compiler

import (
	"fmt"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/statement"
)

// desugarPlaceholders rewrites a bare '_' used as a direct operand of a call, method call, selector, unary/binary
// operator, index, slice, or ternary into an arrow lambda wrapping that single node - see docs/language.md's
// placeholder syntax section.
//
//	foo(_, a)     -> x => foo(x, a)
//	_ > a         -> x => x > a
//	_.name        -> x => x.name
//	arr[_]        -> x => arr[x]
//
// Each distinct '_' among a qualifying node's direct operands becomes a fresh, unique parameter, assigned in
// left-to-right order; only that one node is wrapped - a '_' nested inside a deeper qualifying node (e.g.
// `foo(1, bar(_, 2))`) binds to that inner node only, never bubbling out. This falls out for free from walkFile's
// post-order traversal: children are rewritten (and, if eligible, already turned into a Function literal) before
// their parent is checked, so by the time a parent node is examined, an inner placeholder has already been
// consumed and is no longer a bare Identifier.
//
// This must run before any optimizer pass or pre-optimization validation sees the AST: '_' is deliberately never a
// readable identifier elsewhere in the language (see compiler/optimizer.go and compiler/compiler_impl.go), so a
// bare '_' left in expression position after this pass would surface as "unresolved reference '_'". Because the
// output is an ordinary Function literal wrapping the original (rewritten) node, every later stage - optimizer
// purity/copy-propagation analysis, codegen - sees exactly what a hand-written arrow lambda would produce; nothing
// downstream needs to know this pass exists.
//
// A '_' wrapped in parentheses (e.g. `(_) + 1`) is NOT detected - the placeholder must be bare. This is a
// deliberate limitation, not an oversight: write an arrow lambda by hand for anything beyond the trivial single
// bare '_' case.
func desugarPlaceholders(node ast.Node) ast.Node {
	next := 0
	var rewriteExpr exprRewriteFn
	rewriteExpr = func(e ast.Expression) (ast.Expression, bool) {
		slots := placeholderOperandSlots(e)
		if slots == nil {
			return e, false
		}

		var params []*expression.Identifier
		for _, slot := range slots {
			if !isBarePlaceholder(*slot) {
				continue
			}
			next++
			pos := (*slot).Pos()
			name := fmt.Sprintf("_ph%d", next)
			params = append(params, &expression.Identifier{Name: name, NamePos: pos})
			*slot = &expression.Identifier{Name: name, NamePos: pos}
		}
		if len(params) == 0 {
			return e, false
		}

		return wrapAsPlaceholderLambda(params, e), true
	}

	n, _ := walkFile(node, nil, rewriteExpr)
	return n
}

// placeholderOperandSlots returns pointers to the direct operand fields of e that are eligible to hold a bare '_'
// placeholder, in left-to-right syntax order. Returns nil for any node kind that doesn't participate in placeholder
// desugaring.
func placeholderOperandSlots(e ast.Expression) []*ast.Expression {
	switch n := e.(type) {
	case *expression.Call:
		slots := make([]*ast.Expression, 0, len(n.Args)+1)
		slots = append(slots, &n.Func)
		for i := range n.Args {
			slots = append(slots, &n.Args[i])
		}
		return slots
	case *expression.MethodCall:
		slots := make([]*ast.Expression, 0, len(n.Args)+1)
		slots = append(slots, &n.Object)
		for i := range n.Args {
			slots = append(slots, &n.Args[i])
		}
		return slots
	case *expression.Selector:
		return []*ast.Expression{&n.Expr}
	case *expression.Binary:
		return []*ast.Expression{&n.LHS, &n.RHS}
	case *expression.Unary:
		return []*ast.Expression{&n.Expr}
	case *expression.Index:
		return []*ast.Expression{&n.Expr, &n.Index}
	case *expression.Slice:
		slots := []*ast.Expression{&n.Expr}
		if n.Low != nil {
			slots = append(slots, &n.Low)
		}
		if n.High != nil {
			slots = append(slots, &n.High)
		}
		if n.Step != nil {
			slots = append(slots, &n.Step)
		}
		return slots
	case *expression.Ternary:
		return []*ast.Expression{&n.Cond, &n.True, &n.False}
	default:
		return nil
	}
}

// isBarePlaceholder reports whether expr is a bare '_' identifier (as opposed to, e.g., a discard target elsewhere
// in the grammar, or '_' nested inside a parenthesized/composite sub-expression, which this pass does not unwrap).
func isBarePlaceholder(expr ast.Expression) bool {
	id, ok := expr.(*expression.Identifier)
	return ok && id.Name == "_"
}

// wrapAsPlaceholderLambda builds the same Function/FunctionType/Block/Return shape the parser produces for a
// hand-written arrow lambda (see parser.parseLambda), so nothing downstream needs to special-case placeholder-
// derived functions.
func wrapAsPlaceholderLambda(params []*expression.Identifier, body ast.Expression) *expression.Function {
	pos := body.Pos()
	return &expression.Function{
		Type: &expression.FunctionType{
			FuncPos: pos,
			Params: &expression.Identifiers{
				LParen: pos,
				List:   params,
				RParen: pos,
			},
		},
		Body: &statement.Block{
			LBrace: pos,
			RBrace: body.End(),
			Stmts: []ast.Statement{
				&statement.Return{ReturnPos: pos, Result: body},
			},
		},
	}
}
