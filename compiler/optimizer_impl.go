package compiler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// -----------------------------------------------------------------------------
// AST walker helpers (bottom-up rewriters)
// -----------------------------------------------------------------------------

// exprRewriteFn is applied bottom-up to every Expr. It returns the replacement
// node and true when it changed the node.
type exprRewriteFn func(parser.Expr) (parser.Expr, bool)

// stmtRewriteFn is applied bottom-up to every Stmt. It returns the replacement
// (may be nil to drop the statement) and true when it changed the node.
type stmtRewriteFn func(parser.Stmt) (parser.Stmt, bool)

// walkExpr walks e bottom-up, applying fn after recursing into children.
// stmtFn is applied to statements inside function-literal bodies encountered
// during traversal (pass nil when the caller does not care).
func walkExpr(e parser.Expr, fn exprRewriteFn) (parser.Expr, bool) {
	return walkExprWithStmt(e, nil, fn)
}

func walkExprWithStmt(e parser.Expr, stmtFn stmtRewriteFn, fn exprRewriteFn) (parser.Expr, bool) {
	if e == nil {
		return nil, false
	}
	var changed bool
	var c bool
	switch n := e.(type) {
	case *parser.ParenExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
	case *parser.BinaryExpr:
		n.LHS, c = walkExprWithStmt(n.LHS, stmtFn, fn)
		changed = changed || c
		n.RHS, c = walkExprWithStmt(n.RHS, stmtFn, fn)
		changed = changed || c
	case *parser.UnaryExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
	case *parser.CondExpr:
		n.Cond, c = walkExprWithStmt(n.Cond, stmtFn, fn)
		changed = changed || c
		n.True, c = walkExprWithStmt(n.True, stmtFn, fn)
		changed = changed || c
		n.False, c = walkExprWithStmt(n.False, stmtFn, fn)
		changed = changed || c
	case *parser.CallExpr:
		n.Func, c = walkExprWithStmt(n.Func, stmtFn, fn)
		changed = changed || c
		for i, a := range n.Args {
			n.Args[i], c = walkExprWithStmt(a, stmtFn, fn)
			changed = changed || c
		}
	case *parser.MethodCallExpr:
		n.Object, c = walkExprWithStmt(n.Object, stmtFn, fn)
		changed = changed || c
		for i, a := range n.Args {
			n.Args[i], c = walkExprWithStmt(a, stmtFn, fn)
			changed = changed || c
		}
	case *parser.IndexExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
		n.Index, c = walkExprWithStmt(n.Index, stmtFn, fn)
		changed = changed || c
	case *parser.SelectorExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
		// Sel is an Ident/expression selector; we do not fold it.
	case *parser.SliceExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
		if n.Low != nil {
			n.Low, c = walkExprWithStmt(n.Low, stmtFn, fn)
			changed = changed || c
		}
		if n.High != nil {
			n.High, c = walkExprWithStmt(n.High, stmtFn, fn)
			changed = changed || c
		}
		if n.Step != nil {
			n.Step, c = walkExprWithStmt(n.Step, stmtFn, fn)
			changed = changed || c
		}
	case *parser.ArrayLit:
		for i, elem := range n.Elements {
			n.Elements[i], c = walkExprWithStmt(elem, stmtFn, fn)
			changed = changed || c
		}
	case *parser.RecordLit:
		for _, el := range n.Elements {
			el.Value, c = walkExprWithStmt(el.Value, stmtFn, fn)
			changed = changed || c
		}
	case *parser.FuncLit:
		newBody, bc := walkStmt(n.Body, stmtFn, fn)
		if bc {
			if bs, ok := newBody.(*parser.BlockStmt); ok {
				n.Body = bs
			}
		}
		changed = changed || bc
	case *parser.ImmutableExpr:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
	case *parser.FStringLit:
		for i := range n.Parts {
			if n.Parts[i].Expr != nil {
				n.Parts[i].Expr, c = walkExprWithStmt(n.Parts[i].Expr, stmtFn, fn)
				changed = changed || c
			}
			for j := range n.Parts[i].SpecExprs {
				n.Parts[i].SpecExprs[j], c = walkExprWithStmt(n.Parts[i].SpecExprs[j], stmtFn, fn)
				changed = changed || c
			}
		}
	}
	if fn != nil {
		if r, rc := fn(e); rc {
			return r, true
		}
	}
	return e, changed
}

// walkStmt walks s bottom-up. If stmtFn is non-nil it is applied to every Stmt
// after recursing. If exprFn is non-nil, expressions embedded in statements
// are rewritten as well.
func walkStmt(s parser.Stmt, stmtFn stmtRewriteFn, exprFn exprRewriteFn) (parser.Stmt, bool) {
	if s == nil {
		return nil, false
	}
	var changed bool
	var c bool
	switch n := s.(type) {
	case *parser.BlockStmt:
		out := n.Stmts[:0]
		for _, sub := range n.Stmts {
			r, rc := walkStmt(sub, stmtFn, exprFn)
			if rc {
				changed = true
			}
			if r != nil {
				out = append(out, r)
			}
		}
		n.Stmts = out
	case *parser.ExprStmt:
		if exprFn != nil || stmtFn != nil {
			n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, exprFn)
			changed = changed || c
		}
	case *parser.AssignStmt:
		if exprFn != nil || stmtFn != nil {
			for i, e := range n.LHS {
				// LHS may be an ident being assigned; do not rewrite the plain
				// ident target itself (folding a target would break semantics),
				// but do descend into index/selector targets.
				switch e.(type) {
				case *parser.Ident:
					// skip
				default:
					n.LHS[i], c = walkExprWithStmt(e, stmtFn, exprFn)
					changed = changed || c
				}
			}
			for i, e := range n.RHS {
				n.RHS[i], c = walkExprWithStmt(e, stmtFn, exprFn)
				changed = changed || c
			}
		}
	case *parser.IfStmt:
		if n.Init != nil {
			r, ic := walkStmt(n.Init, stmtFn, exprFn)
			if ic {
				n.Init = r
				changed = true
			}
		}
		if exprFn != nil || stmtFn != nil {
			n.Cond, c = walkExprWithStmt(n.Cond, stmtFn, exprFn)
			changed = changed || c
		}
		if r, bc := walkStmt(n.Body, stmtFn, exprFn); bc {
			if bs, ok := r.(*parser.BlockStmt); ok {
				n.Body = bs
			}
			changed = true
		}
		if n.Else != nil {
			if r, ec := walkStmt(n.Else, stmtFn, exprFn); ec {
				n.Else = r
				changed = true
			}
		}
	case *parser.ForStmt:
		if n.Init != nil {
			if r, ic := walkStmt(n.Init, stmtFn, exprFn); ic {
				n.Init = r
				changed = true
			}
		}
		if n.Cond != nil && (exprFn != nil || stmtFn != nil) {
			n.Cond, c = walkExprWithStmt(n.Cond, stmtFn, exprFn)
			changed = changed || c
		}
		if n.Post != nil {
			if r, pc := walkStmt(n.Post, stmtFn, exprFn); pc {
				n.Post = r
				changed = true
			}
		}
		if r, bc := walkStmt(n.Body, stmtFn, exprFn); bc {
			if bs, ok := r.(*parser.BlockStmt); ok {
				n.Body = bs
			}
			changed = true
		}
	case *parser.ForInStmt:
		if exprFn != nil || stmtFn != nil {
			n.Iterable, c = walkExprWithStmt(n.Iterable, stmtFn, exprFn)
			changed = changed || c
		}
		if r, bc := walkStmt(n.Body, stmtFn, exprFn); bc {
			if bs, ok := r.(*parser.BlockStmt); ok {
				n.Body = bs
			}
			changed = true
		}
	case *parser.ReturnStmt:
		if n.Result != nil && (exprFn != nil || stmtFn != nil) {
			n.Result, c = walkExprWithStmt(n.Result, stmtFn, exprFn)
			changed = changed || c
		}
	case *parser.ExportStmt:
		if exprFn != nil || stmtFn != nil {
			n.Result, c = walkExprWithStmt(n.Result, stmtFn, exprFn)
			changed = changed || c
		}
	case *parser.DeferStmt:
		if exprFn != nil || stmtFn != nil {
			n.Call, c = walkExprWithStmt(n.Call, stmtFn, exprFn)
			changed = changed || c
		}
	case *parser.IncDecStmt:
		// Do NOT rewrite the LHS of ++/-- (it is an assignment target).
		// Only descend into IndexExpr / SelectorExpr targets to fold their sub-parts.
		if exprFn != nil || stmtFn != nil {
			switch t := n.Expr.(type) {
			case *parser.IndexExpr:
				t.Index, c = walkExprWithStmt(t.Index, stmtFn, exprFn)
				changed = changed || c
			case *parser.SelectorExpr:
				t.Expr, c = walkExprWithStmt(t.Expr, stmtFn, exprFn)
				changed = changed || c
			}
		}
	}
	if stmtFn != nil {
		if r, rc := stmtFn(s); rc {
			return r, true
		}
	}
	return s, changed
}

// walkFile dispatches walking to the appropriate helper based on the root
// node type. Root is typically *parser.File but may be a bare Stmt/Expr.
func walkFile(n parser.Node, stmtFn stmtRewriteFn, exprFn exprRewriteFn) (parser.Node, bool) {
	if n == nil {
		return nil, false
	}
	switch t := n.(type) {
	case *parser.File:
		var changed bool
		out := t.Stmts[:0]
		for _, s := range t.Stmts {
			r, rc := walkStmt(s, stmtFn, exprFn)
			if rc {
				changed = true
			}
			if r != nil {
				out = append(out, r)
			}
		}
		t.Stmts = out
		return t, changed
	case parser.Stmt:
		return walkStmt(t, stmtFn, exprFn)
	case parser.Expr:
		return walkExprWithStmt(t, stmtFn, exprFn)
	}
	return n, false
}

// -----------------------------------------------------------------------------
// Literal / value helpers
// -----------------------------------------------------------------------------

// isLiteralExpr returns true if the expression is a scalar literal AST node
// that can be safely used as a compile-time constant.
func isLiteralExpr(e parser.Expr) bool {
	switch e.(type) {
	case *parser.IntLit, *parser.FloatLit, *parser.DecimalLit,
		*parser.BoolLit, *parser.StringLit, *parser.RuneLit,
		*parser.ByteLit, *parser.UndefinedLit,
		*parser.BytesLit, *parser.RunesLit, *parser.TimeLit:
		return true
	}
	return false
}

// literalToValue converts a literal expression to a core.Value. Returns
// (Undefined, false) when the node is not a scalar literal we can evaluate at
// compile time.
func literalToValue(e parser.Expr) (core.Value, bool) {
	switch n := e.(type) {
	case *parser.IntLit:
		return core.IntValue(n.Value), true
	case *parser.FloatLit:
		return core.FloatValue(n.Value), true
	case *parser.DecimalLit:
		return core.NewDecimalValue(n.Value), true
	case *parser.BoolLit:
		if n.Value {
			return core.True, true
		}
		return core.False, true
	case *parser.StringLit:
		return core.NewStringValue(n.Value), true
	case *parser.RuneLit:
		return core.RuneValue(n.Value), true
	case *parser.ByteLit:
		return core.ByteValue(n.Value), true
	case *parser.UndefinedLit:
		return core.Undefined, true
	case *parser.RunesLit:
		return core.NewRunesValue(n.Value, true), true
	case *parser.BytesLit:
		return core.NewBytesValue(n.Value, true), true
	case *parser.TimeLit:
		return core.NewTimeValue(n.Value), true
	}
	return core.Undefined, false
}

// safeValueToLiteral converts a runtime value back into an AST literal, if a safe
// round-trip is possible. Only scalar / immutable types are supported so we
// never introduce shared mutable containers as constants.
func safeValueToLiteral(v core.Value, pos core.Pos) (parser.Expr, bool) {
	switch v.Type {
	case value.Undefined:
		return &parser.UndefinedLit{TokenPos: pos}, true
	case value.Bool:
		b := v.Data != 0
		lit := "false"
		if b {
			lit = "true"
		}
		return &parser.BoolLit{Value: b, ValuePos: pos, Literal: lit}, true
	case value.Int:
		i := int64(v.Data)
		return &parser.IntLit{Value: i, ValuePos: pos, Literal: strconv.FormatInt(i, 10)}, true
	case value.Float:
		if f, ok := v.AsFloat(); ok {
			return &parser.FloatLit{Value: f, ValuePos: pos, Literal: strconv.FormatFloat(f, 'g', -1, 64)}, true
		}
	case value.Decimal:
		if d, ok := v.AsDecimal(); ok {
			return &parser.DecimalLit{Value: d, ValuePos: pos, Literal: d.String() + "d"}, true
		}
	case value.String:
		if s, ok := v.AsString(); ok {
			return &parser.StringLit{Value: s, ValuePos: pos, Literal: strconv.Quote(s)}, true
		}
	case value.Rune:
		r := rune(v.Data)
		return &parser.RuneLit{Value: r, ValuePos: pos, Literal: strconv.QuoteRune(r)}, true
	case value.Byte:
		return &parser.ByteLit{Value: byte(v.Data), ValuePos: pos, Literal: fmt.Sprintf("'\\x%02x'", byte(v.Data))}, true
	case value.Time:
		if t, ok := v.AsTime(); ok {
			return &parser.TimeLit{Value: t, ValuePos: pos, Literal: `"` + t.Format(time.RFC3339Nano) + `"`}, true
		}
	}
	// Container / iterator / function-shaped values are intentionally not
	// converted back — a folded shared reference would break identity /
	// mutability semantics.
	return nil, false
}

// isTruthyLiteral returns (truthy, isConst). isConst==true iff e is a literal
// whose truthiness is known at compile time. Falls back to Kavun's runtime
// truthiness table (see docs/language.md).
func isTruthyLiteral(e parser.Expr) (bool, bool) {
	v, ok := literalToValue(e)
	if !ok {
		return false, false
	}
	return v.IsTrue(), true
}

// -----------------------------------------------------------------------------
// Eligibility for speculative evaluation
// -----------------------------------------------------------------------------

// isBuiltinName reports whether name is a globally-defined pure builtin
// function. Used to allow calls like `len("abc")` inside foldable subtrees.
func isBuiltinPureName(name string) bool {
	v, ok := vm.BuiltinFunctions[name]
	if !ok {
		return false
	}
	if v.Type != value.BuiltinFunction {
		return false
	}
	fn := core.BuiltinFunctions[v.Data]
	if fn == nil {
		return false
	}
	return fn.Pure
}

// shadowedBuiltinsIn returns the set of builtin names that are assigned
// anywhere within root. Any call to such a name at runtime may resolve to a
// user value and must NOT be constant-folded.
func shadowedBuiltinsIn(root parser.Node) map[string]bool {
	out := make(map[string]bool)
	usage := collectNameUsage(root)
	for name, u := range usage {
		if u.writes > 0 && isBuiltinPureName(name) {
			out[name] = true
		}
	}
	return out
}

// isFoldableExpr checks whether every leaf of the subtree is either a scalar
// literal or a call to a pure builtin, and that only pure operator/method
// nodes appear internally. Nothing that could observe external state or
// mutable references is allowed.
//
// shadowed is a set of identifier names that must NOT be treated as builtin
// callables even though vm.BuiltinFunctions knows a builtin by that name —
// they have been re-assigned somewhere in the enclosing scope and the runtime
// value at the call site may be an arbitrary user value.
func isFoldableExpr(e parser.Expr, shadowed map[string]bool) bool {
	if e == nil {
		return false
	}
	switch n := e.(type) {
	case *parser.IntLit, *parser.FloatLit, *parser.DecimalLit,
		*parser.BoolLit, *parser.StringLit, *parser.RuneLit,
		*parser.ByteLit, *parser.UndefinedLit,
		*parser.BytesLit, *parser.RunesLit, *parser.TimeLit:
		return true
	case *parser.ParenExpr:
		return isFoldableExpr(n.Expr, shadowed)
	case *parser.UnaryExpr:
		return isFoldableExpr(n.Expr, shadowed)
	case *parser.BinaryExpr:
		return isFoldableExpr(n.LHS, shadowed) && isFoldableExpr(n.RHS, shadowed)
	case *parser.CondExpr:
		return isFoldableExpr(n.Cond, shadowed) && isFoldableExpr(n.True, shadowed) && isFoldableExpr(n.False, shadowed)
	case *parser.IndexExpr:
		return isFoldableExpr(n.Expr, shadowed) && isFoldableExpr(n.Index, shadowed)
	case *parser.SliceExpr:
		if !isFoldableExpr(n.Expr, shadowed) {
			return false
		}
		if n.Low != nil && !isFoldableExpr(n.Low, shadowed) {
			return false
		}
		if n.High != nil && !isFoldableExpr(n.High, shadowed) {
			return false
		}
		if n.Step != nil && !isFoldableExpr(n.Step, shadowed) {
			return false
		}
		return true
	case *parser.MethodCallExpr:
		// Receiver must be foldable, and every argument must be foldable.
		// The higher-order rule from docs/purity.md is enforced implicitly:
		// FuncLit / Ident / CallExpr / MethodCallExpr arguments that carry a
		// function value never satisfy isFoldableExpr, so any callback would
		// disqualify the tree here.
		if !isFoldableExpr(n.Object, shadowed) {
			return false
		}
		for _, a := range n.Args {
			if !isFoldableExpr(a, shadowed) {
				return false
			}
		}
		// Reject spread arguments — those alias the underlying array.
		if n.Ellipsis.IsValid() {
			return false
		}
		return true
	case *parser.CallExpr:
		// Only pure builtin function calls are allowed. The callee must be a
		// bare identifier naming a globally-registered pure builtin that has
		// NOT been shadowed by any assignment in the surrounding scope.
		id, ok := n.Func.(*parser.Ident)
		if !ok || !isBuiltinPureName(id.Name) {
			return false
		}
		if shadowed[id.Name] {
			return false
		}
		for _, a := range n.Args {
			if !isFoldableExpr(a, shadowed) {
				return false
			}
		}
		if n.Ellipsis.IsValid() {
			return false
		}
		return true
	case *parser.FStringLit:
		for _, p := range n.Parts {
			if p.Expr != nil && !isFoldableExpr(p.Expr, shadowed) {
				return false
			}
			for _, se := range p.SpecExprs {
				if !isFoldableExpr(se, shadowed) {
					return false
				}
			}
		}
		return true
	}
	return false
}

// -----------------------------------------------------------------------------
// Speculative evaluation
// -----------------------------------------------------------------------------

// evalConstantExpr speculatively compiles and runs expr in an isolated
// compiler + VM sandbox. Returns the runtime value on success.
// The sandbox has:
//   - Fresh symbol table with only builtin function names.
//   - Empty allowed-modules set (no imports).
//   - No custom modules.
//   - Optimization disabled (avoids recursive folding).
//   - A cancellation deadline enforced by aborting the VM.
func evalConstantExpr(expr parser.Expr, fset *parser.SourceFileSet) (core.Value, bool) {
	// Defensive recover: any panic in the isolated compiler/VM stack is
	// treated as "not foldable" and leaves the original subtree untouched.
	var (
		result core.Value
		ok     bool
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				ok = false
			}
		}()
		result, ok = evalConstantExprUnsafe(expr, fset)
	}()
	return result, ok
}

func evalConstantExprUnsafe(expr parser.Expr, fset *parser.SourceFileSet) (core.Value, bool) {
	if fset == nil {
		fset = parser.NewFileSet()
	}
	srcFile := fset.AddFile("<opt>", -1, 0)

	// Isolated symbol table: only builtins are visible. Importantly, we do
	// NOT copy any parent symbols so identifier references (which are excluded
	// by isFoldableExpr) would fail to compile if one slipped through.
	symTable := NewSymbolTable()

	// Empty allowed-modules set (not nil) so imports are disallowed. isFoldable
	// already excludes ImportExpr but this is defense-in-depth.
	emptyAllowed := []string{}

	c := NewCompiler(O0(), nil, srcFile, symTable, emptyAllowed, nil, nil)
	c.SetAssignmentMode(AssignmentModeSmart)

	// Build a synthetic AST: `__opt_result__ := expr`
	pos := expr.Pos()
	target := &parser.Ident{Name: "__opt_result__", NamePos: pos}
	assign := &parser.AssignStmt{
		LHS:      []parser.Expr{target},
		RHS:      []parser.Expr{expr},
		Token:    token.Define,
		TokenPos: pos,
	}
	file := &parser.File{InputFile: srcFile, Stmts: []parser.Stmt{assign}}
	if err := c.CompileNode(file); err != nil {
		return core.Undefined, false
	}
	bc := c.Bytecode()

	// Locate the result slot before running.
	sym, _, resolved := symTable.Resolve("__opt_result__", false)
	if !resolved {
		return core.Undefined, false
	}

	// Cancel after a small deadline to bound worst-case cost.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	globals := make([]core.Value, vm.GlobalsSize)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	machine.Reset(bc, globals)

	ch := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- fmt.Errorf("panic: %v", r)
			}
		}()
		ch <- machine.Run()
	}()

	var runErr error
	select {
	case <-ctx.Done():
		machine.Abort()
		<-ch
		runErr = ctx.Err()
	case runErr = <-ch:
	}
	if runErr != nil {
		return core.Undefined, false
	}
	return globals[sym.Index], true
}

// -----------------------------------------------------------------------------
// Pass: simplifyBooleanIdentities
// -----------------------------------------------------------------------------

// isProvablyBool returns true when e is provably a bool-typed expression by
// AST-only inspection (bool literal, comparison, `!`, `&&`, `||`, `in`,
// `not in`). We do NOT trust identifiers because their type at that program
// point cannot be inferred without full type analysis.
func isProvablyBool(e parser.Expr) bool {
	switch n := e.(type) {
	case *parser.BoolLit:
		return true
	case *parser.ParenExpr:
		return isProvablyBool(n.Expr)
	case *parser.UnaryExpr:
		return n.Token == token.Not
	case *parser.BinaryExpr:
		switch n.Token {
		case token.Equal, token.NotEqual,
			token.Less, token.LessEq, token.Greater, token.GreaterEq,
			token.In:
			return true
		case token.LAnd, token.LOr:
			// Kavun's && / || return one of the operands. The result is bool
			// only when both operands are provably bool.
			return isProvablyBool(n.LHS) && isProvablyBool(n.RHS)
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Pass: simplifyConstantConditions (also handles eliminateDeadBranches via
// natural composition with `if / else if / else`).
// -----------------------------------------------------------------------------

// stmtToBlock ensures s is a *BlockStmt.
func stmtToBlock(s parser.Stmt, at core.Pos) *parser.BlockStmt {
	if s == nil {
		return &parser.BlockStmt{LBrace: at, RBrace: at, Stmts: nil}
	}
	if b, ok := s.(*parser.BlockStmt); ok {
		return b
	}
	return &parser.BlockStmt{
		LBrace: s.Pos(),
		RBrace: s.End(),
		Stmts:  []parser.Stmt{s},
	}
}

func (c *Compiler) runSimplifyConstantConditions(node parser.Node) (parser.Node, bool, error) {
	// Statement-level: fold if-statements whose condition is a scalar literal.
	stmtFn := func(s parser.Stmt) (parser.Stmt, bool) {
		is, ok := s.(*parser.IfStmt)
		if !ok {
			return s, false
		}
		// A non-nil Init may declare variables referenced later via smart `=`
		// or may have side effects; we cannot silently drop it. If it is
		// present, we conservatively refuse to rewrite the whole statement
		// unless we can safely keep the Init as a sibling. Init is always a
		// simple statement in Kavun (usually AssignStmt); when we keep it,
		// wrap it with the surviving branch in a fresh BlockStmt.
		truthy, isConst := isTruthyLiteral(is.Cond)
		if !isConst {
			return s, false
		}
		var chosen parser.Stmt
		if truthy {
			chosen = is.Body
		} else if is.Else != nil {
			chosen = is.Else
		} else {
			// No else and condition falsy: drop the branch. Preserve Init if any.
			if is.Init != nil {
				return &parser.BlockStmt{
					LBrace: is.IfPos,
					RBrace: is.End(),
					Stmts:  []parser.Stmt{is.Init},
				}, true
			}
			return &parser.BlockStmt{LBrace: is.IfPos, RBrace: is.End()}, true
		}
		if is.Init != nil {
			block := stmtToBlock(chosen, is.IfPos)
			// Prepend the Init statement to preserve any variable it declares.
			newStmts := make([]parser.Stmt, 0, len(block.Stmts)+1)
			newStmts = append(newStmts, is.Init)
			newStmts = append(newStmts, block.Stmts...)
			return &parser.BlockStmt{
				LBrace: is.IfPos,
				RBrace: is.End(),
				Stmts:  newStmts,
			}, true
		}
		return chosen, true
	}
	// Expression-level: fold ternary `c ? a : b` with a constant c.
	exprFn := func(e parser.Expr) (parser.Expr, bool) {
		ce, ok := e.(*parser.CondExpr)
		if !ok {
			return e, false
		}
		truthy, isConst := isTruthyLiteral(ce.Cond)
		if !isConst {
			return e, false
		}
		if truthy {
			return ce.True, true
		}
		return ce.False, true
	}
	n, changed := walkFile(node, stmtFn, exprFn)
	return n, changed, nil
}

// -----------------------------------------------------------------------------
// Pass: simplifyIfExprToBool
// -----------------------------------------------------------------------------

// singleBoolValue returns (boolLit, true) when block contains exactly one
// bool-literal-producing statement (ExprStmt of a BoolLit, or ReturnStmt of a
// BoolLit).
func singleBoolValue(s parser.Stmt) (*parser.BoolLit, parser.Stmt, bool) {
	block, ok := s.(*parser.BlockStmt)
	if ok {
		if len(block.Stmts) != 1 {
			return nil, nil, false
		}
		s = block.Stmts[0]
	}
	switch t := s.(type) {
	case *parser.ExprStmt:
		if bl, ok := t.Expr.(*parser.BoolLit); ok {
			return bl, t, true
		}
	case *parser.ReturnStmt:
		if t.Result != nil {
			if bl, ok := t.Result.(*parser.BoolLit); ok {
				return bl, t, true
			}
		}
	}
	return nil, nil, false
}

func (c *Compiler) runSimplifyIfExprToBool(node parser.Node) (parser.Node, bool, error) {
	stmtFn := func(s parser.Stmt) (parser.Stmt, bool) {
		is, ok := s.(*parser.IfStmt)
		if !ok || is.Else == nil || is.Init != nil {
			return s, false
		}
		if !isProvablyBool(is.Cond) {
			return s, false
		}
		trueBl, trueKind, tok := singleBoolValue(is.Body)
		if !tok {
			return s, false
		}
		falseBl, falseKind, fok := singleBoolValue(is.Else)
		if !fok {
			return s, false
		}
		// Both branches must be the same kind (return-return or expr-expr) —
		// otherwise the surrounding statement's control-flow semantics differ.
		if fmt.Sprintf("%T", trueKind) != fmt.Sprintf("%T", falseKind) {
			return s, false
		}
		// Both bools identical → not simplifiable (both branches trivially
		// equal to the same constant; that's a different optimization).
		if trueBl.Value == falseBl.Value {
			return s, false
		}
		// The final expression is either `cond` or `!cond` depending on which
		// branch produced `true`.
		var expr parser.Expr = is.Cond
		if !trueBl.Value {
			expr = &parser.UnaryExpr{Expr: is.Cond, Token: token.Not, TokenPos: is.IfPos}
		}
		switch trueKind.(type) {
		case *parser.ReturnStmt:
			return &parser.ReturnStmt{ReturnPos: is.IfPos, Result: expr}, true
		case *parser.ExprStmt:
			return &parser.ExprStmt{Expr: expr}, true
		}
		return s, false
	}
	n, changed := walkFile(node, stmtFn, nil)
	return n, changed, nil
}

// -----------------------------------------------------------------------------
// Pass: eliminateDeadBranches
// -----------------------------------------------------------------------------

// This handles chained `if / else if / ...` where an early branch has a
// constant-true condition (making later branches unreachable) or where an
// early branch is constant-false (making that branch removable and letting the
// else-if chain shift up). simplifyConstantConditions already handles the
// single-branch case; this pass focuses on the chained variant to keep the
// two behaviors independently testable.
func (c *Compiler) runEliminateDeadBranches(node parser.Node) (parser.Node, bool, error) {
	var globalChanged bool
	var simplify func(is *parser.IfStmt) (parser.Stmt, bool)
	simplify = func(is *parser.IfStmt) (parser.Stmt, bool) {
		// Recurse first so inner ifs are simplified.
		if is.Else != nil {
			if inner, ok := is.Else.(*parser.IfStmt); ok {
				if r, c := simplify(inner); c {
					is.Else = r
				}
			}
		}
		// Only touch chained else-if forms with a compile-time-constant
		// condition. Ignore ifs with Init (see simplifyConstantConditions).
		if is.Init != nil {
			return is, false
		}
		truthy, isConst := isTruthyLiteral(is.Cond)
		if !isConst {
			return is, false
		}
		if truthy {
			// Whole `if / else if / else` collapses to the body of this branch.
			return is.Body, true
		}
		// Condition constant-false: drop this arm, promote else.
		if is.Else != nil {
			return is.Else, true
		}
		return &parser.BlockStmt{LBrace: is.IfPos, RBrace: is.End()}, true
	}
	stmtFn := func(s parser.Stmt) (parser.Stmt, bool) {
		is, ok := s.(*parser.IfStmt)
		if !ok {
			return s, false
		}
		if r, c := simplify(is); c {
			globalChanged = true
			return r, true
		}
		return s, false
	}
	n, changed := walkFile(node, stmtFn, nil)
	return n, changed || globalChanged, nil
}

// -----------------------------------------------------------------------------
// Pass: eliminateUnreachableAfterTerminator
// -----------------------------------------------------------------------------

// isTerminatorStmt returns true when s always exits the containing block.
func isTerminatorStmt(s parser.Stmt) bool {
	switch t := s.(type) {
	case *parser.ReturnStmt:
		return true
	case *parser.BranchStmt:
		return t.Token == token.Break || t.Token == token.Continue
	}
	return false
}

func (c *Compiler) runEliminateUnreachableAfterTerminator(node parser.Node) (parser.Node, bool, error) {
	var changed bool
	stmtFn := func(s parser.Stmt) (parser.Stmt, bool) {
		block, ok := s.(*parser.BlockStmt)
		if !ok {
			return s, false
		}
		for i, sub := range block.Stmts {
			if isTerminatorStmt(sub) && i+1 < len(block.Stmts) {
				block.Stmts = block.Stmts[:i+1]
				changed = true
				return block, true
			}
		}
		return s, false
	}
	n, walkChanged := walkFile(node, stmtFn, nil)
	return n, changed || walkChanged, nil
}

// -----------------------------------------------------------------------------
// Scope / usage analysis helpers (used by conservative propagation & DCE)
// -----------------------------------------------------------------------------

// nameUsage tracks how a local name is used within a scope.
type nameUsage struct {
	reads               int
	writes              int
	insideFuncLit       bool
	insideLoop          bool
	insideDefer         bool
	addressed           bool // taken by ++ / -- / compound assign / as base of an LHS mutation
	takenAsAssignTarget bool
}

// markLHSAddressed marks every ident inside a compound LHS target as
// "addressed" — such idents refer to a container that is about to be
// mutated. Propagation must never replace them with a literal value.
func markLHSAddressed(e parser.Expr, get func(string) *nameUsage) {
	switch n := e.(type) {
	case *parser.Ident:
		u := get(n.Name)
		u.addressed = true
	case *parser.SelectorExpr:
		markLHSAddressed(n.Expr, get)
	case *parser.IndexExpr:
		markLHSAddressed(n.Expr, get)
	case *parser.SliceExpr:
		markLHSAddressed(n.Expr, get)
	case *parser.ParenExpr:
		markLHSAddressed(n.Expr, get)
	}
}

// collectNameUsage walks the AST and records how each named identifier is
// used. Used for very conservative propagation / dead-code checks: whenever
// we plan to remove or replace a binding we require its usage record to
// satisfy a strict pattern (read-only, no closure/loop/defer).
func collectNameUsage(root parser.Node) map[string]*nameUsage {
	usage := make(map[string]*nameUsage)
	get := func(name string) *nameUsage {
		u, ok := usage[name]
		if !ok {
			u = &nameUsage{}
			usage[name] = u
		}
		return u
	}
	var (
		inFunc  int
		inLoop  int
		inDefer int
	)
	var (
		walkE func(e parser.Expr, isRead bool)
		walkS func(s parser.Stmt)
	)
	walkE = func(e parser.Expr, isRead bool) {
		if e == nil {
			return
		}
		switch n := e.(type) {
		case *parser.Ident:
			u := get(n.Name)
			if isRead {
				u.reads++
			} else {
				u.writes++
				u.takenAsAssignTarget = true
			}
			if inFunc > 0 {
				u.insideFuncLit = true
			}
			if inLoop > 0 {
				u.insideLoop = true
			}
			if inDefer > 0 {
				u.insideDefer = true
			}
		case *parser.ParenExpr:
			walkE(n.Expr, isRead)
		case *parser.BinaryExpr:
			walkE(n.LHS, true)
			walkE(n.RHS, true)
		case *parser.UnaryExpr:
			walkE(n.Expr, true)
		case *parser.CondExpr:
			walkE(n.Cond, true)
			walkE(n.True, true)
			walkE(n.False, true)
		case *parser.CallExpr:
			walkE(n.Func, true)
			for _, a := range n.Args {
				walkE(a, true)
			}
		case *parser.MethodCallExpr:
			walkE(n.Object, true)
			for _, a := range n.Args {
				walkE(a, true)
			}
		case *parser.IndexExpr:
			walkE(n.Expr, true)
			walkE(n.Index, true)
		case *parser.SelectorExpr:
			walkE(n.Expr, true)
		case *parser.SliceExpr:
			walkE(n.Expr, true)
			walkE(n.Low, true)
			walkE(n.High, true)
			walkE(n.Step, true)
		case *parser.ArrayLit:
			for _, elem := range n.Elements {
				walkE(elem, true)
			}
		case *parser.RecordLit:
			for _, el := range n.Elements {
				walkE(el.Value, true)
			}
		case *parser.FuncLit:
			inFunc++
			// Parameters and named result declare local names; record their
			// writes so outer scope's usage is not polluted.
			for _, p := range n.Type.Params.List {
				u := get(p.Name)
				u.writes++
				u.takenAsAssignTarget = true
			}
			if n.Type.Result != nil {
				u := get(n.Type.Result.Name)
				u.writes++
				u.takenAsAssignTarget = true
			}
			walkS(n.Body)
			inFunc--
		case *parser.ImmutableExpr:
			walkE(n.Expr, true)
		case *parser.FStringLit:
			for _, p := range n.Parts {
				if p.Expr != nil {
					walkE(p.Expr, true)
				}
				for _, se := range p.SpecExprs {
					walkE(se, true)
				}
			}
		}
	}
	walkS = func(s parser.Stmt) {
		if s == nil {
			return
		}
		switch n := s.(type) {
		case *parser.BlockStmt:
			for _, sub := range n.Stmts {
				walkS(sub)
			}
		case *parser.ExprStmt:
			walkE(n.Expr, true)
		case *parser.AssignStmt:
			// LHS ident is a write; other targets recurse as reads (index/selector
			// receivers are read to locate the target). For compound LHS targets
			// (SelectorExpr / IndexExpr), any embedded ident is being used as the
			// base of a mutation — mark it addressed so propagation refuses to
			// replace it with a literal (which would break the store target).
			for _, lh := range n.LHS {
				switch t := lh.(type) {
				case *parser.Ident:
					walkE(t, false)
				default:
					walkE(t, true)
					markLHSAddressed(t, get)
				}
			}
			for _, rh := range n.RHS {
				walkE(rh, true)
			}
			// Compound assignments (+=, etc.) also read the LHS.
			if n.Token != token.Assign && n.Token != token.Define {
				for _, lh := range n.LHS {
					if id, ok := lh.(*parser.Ident); ok {
						u := get(id.Name)
						u.reads++
						u.addressed = true
					}
				}
			}
		case *parser.IfStmt:
			walkS(n.Init)
			walkE(n.Cond, true)
			walkS(n.Body)
			walkS(n.Else)
		case *parser.ForStmt:
			inLoop++
			walkS(n.Init)
			walkE(n.Cond, true)
			walkS(n.Post)
			walkS(n.Body)
			inLoop--
		case *parser.ForInStmt:
			inLoop++
			u := get(n.Key.Name)
			u.writes++
			u.takenAsAssignTarget = true
			if n.Value != nil {
				u := get(n.Value.Name)
				u.writes++
				u.takenAsAssignTarget = true
			}
			walkE(n.Iterable, true)
			walkS(n.Body)
			inLoop--
		case *parser.ReturnStmt:
			if n.Result != nil {
				walkE(n.Result, true)
			}
		case *parser.ExportStmt:
			walkE(n.Result, true)
		case *parser.DeferStmt:
			inDefer++
			walkE(n.Call, true)
			inDefer--
		case *parser.IncDecStmt:
			if id, ok := n.Expr.(*parser.Ident); ok {
				u := get(id.Name)
				u.reads++
				u.writes++
				u.addressed = true
			} else {
				walkE(n.Expr, true)
			}
		}
	}
	switch t := root.(type) {
	case *parser.File:
		for _, s := range t.Stmts {
			walkS(s)
		}
	case parser.Stmt:
		walkS(t)
	case parser.Expr:
		walkE(t, true)
	}
	return usage
}

// -----------------------------------------------------------------------------
// Pass: propagateConstants
// -----------------------------------------------------------------------------

// runPropagateConstants replaces reads of variables declared as `x := <literal>`
// (or `var x = <literal>`) with the literal itself, but only in strictly safe
// cases:
//   - The declaration is a single-LHS := / assign with a literal RHS.
//   - The variable is NEVER written after that point (only reads).
//   - The variable is NEVER referenced from inside a FuncLit (closures capture
//     by reference).
//   - The variable is NEVER referenced from inside a defer.
//   - The variable is NEVER a for-loop-in bound (each iteration re-binds).
//
// This is conservative: it only fires for top-level declarations in the File
// stmt list (walk semantics guarantee sequential scope), and it does not
// propagate across function boundaries. Even so, it composes with folding to
// turn `x := 2; y := x + 3` into `x := 2; y := 5` in one optimization cycle.
func (c *Compiler) runPropagateConstants(node parser.Node) (parser.Node, bool, error) {
	file, ok := node.(*parser.File)
	if !ok {
		return node, false, nil
	}
	usage := collectNameUsage(file)
	consts := make(map[string]parser.Expr) // name → literal
	for _, s := range file.Stmts {
		as, ok := s.(*parser.AssignStmt)
		if !ok {
			continue
		}
		if len(as.LHS) != 1 || len(as.RHS) != 1 {
			continue
		}
		if as.Token != token.Define && as.Token != token.Assign {
			continue
		}
		id, ok := as.LHS[0].(*parser.Ident)
		if !ok {
			continue
		}
		if !isLiteralExpr(as.RHS[0]) {
			continue
		}
		// Never propagate builtin names — the identifier may be used as a
		// function callee elsewhere (`len(x)`), and replacing it with a value
		// literal would change program semantics.
		if _, isBuiltin := vm.BuiltinFunctions[id.Name]; isBuiltin {
			continue
		}
		u, ok := usage[id.Name]
		if !ok {
			continue
		}
		// Must be single-assignment (exactly the declaration counts), no
		// closure captures, no defer, no loop rebinding.
		if u.writes != 1 || u.insideFuncLit || u.insideDefer || u.addressed {
			continue
		}
		consts[id.Name] = as.RHS[0]
	}
	if len(consts) == 0 {
		return node, false, nil
	}
	// Rewrite reads of tracked idents to the corresponding literal. Avoid
	// rewriting occurrences at LHS positions (walker already skips plain-ident
	// LHS in AssignStmt / IncDecStmt).
	changed := false
	fn := func(e parser.Expr) (parser.Expr, bool) {
		id, ok := e.(*parser.Ident)
		if !ok {
			return e, false
		}
		lit, ok := consts[id.Name]
		if !ok {
			return e, false
		}
		// Clone the literal so each use has its own position (safe since
		// literals are immutable value carriers).
		if v, ok := literalToValue(lit); ok {
			if cloned, ok := safeValueToLiteral(v, id.Pos()); ok {
				changed = true
				return cloned, true
			}
		}
		return e, false
	}
	n, walkChanged := walkFile(file, nil, fn)
	return n, changed || walkChanged, nil
}

// -----------------------------------------------------------------------------
// Pass: eliminateDeadAssignments
// -----------------------------------------------------------------------------

// runEliminateDeadAssignments removes declarations of the form `x := <literal>`
// or `x := <ident>` where x is:
//   - Never read anywhere.
//   - Never captured by a FuncLit.
//   - Never referenced by a defer.
//   - Never re-assigned or addressed (single-assignment).
//   - Its RHS is side-effect-free (a literal or another identifier).
//
// When the RHS has side effects we do NOT remove the statement (to preserve
// observable behavior). Side-effect-free RHS means: literal or bare identifier.
func (c *Compiler) runEliminateDeadAssignments(node parser.Node) (parser.Node, bool, error) {
	file, ok := node.(*parser.File)
	if !ok {
		return node, false, nil
	}
	usage := collectNameUsage(file)
	changed := false
	out := file.Stmts[:0]
	for _, s := range file.Stmts {
		if as, ok := s.(*parser.AssignStmt); ok {
			if len(as.LHS) == 1 && len(as.RHS) == 1 && as.Token == token.Define {
				if id, ok := as.LHS[0].(*parser.Ident); ok {
					u, uok := usage[id.Name]
					sideEffectFree := isLiteralExpr(as.RHS[0])
					if !sideEffectFree {
						if _, ok := as.RHS[0].(*parser.Ident); ok {
							sideEffectFree = true
						}
					}
					if uok && sideEffectFree && u.reads == 0 && u.writes == 1 && !u.insideFuncLit && !u.insideDefer && !u.addressed {
						changed = true
						continue
					}
				}
			}
		}
		out = append(out, s)
	}
	file.Stmts = out
	return file, changed, nil
}
