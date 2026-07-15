package compiler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/expression/composite"
	"github.com/jokruger/kavun/ast/expression/scalar"
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/vm"
)

type exprRewriteFn func(ast.Expression) (ast.Expression, bool)
type stmtRewriteFn func(ast.Statement) (ast.Statement, bool)

func walkExprWithStmt(e ast.Expression, stmtFn stmtRewriteFn, fn exprRewriteFn) (ast.Expression, bool) {
	if e == nil {
		return nil, false
	}

	var changed bool
	var c bool
	switch n := e.(type) {
	case *expression.Parenthesis:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c

	case *expression.Binary:
		n.LHS, c = walkExprWithStmt(n.LHS, stmtFn, fn)
		changed = changed || c
		n.RHS, c = walkExprWithStmt(n.RHS, stmtFn, fn)
		changed = changed || c

	case *expression.Unary:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c

	case *expression.Ternary:
		n.Cond, c = walkExprWithStmt(n.Cond, stmtFn, fn)
		changed = changed || c
		n.True, c = walkExprWithStmt(n.True, stmtFn, fn)
		changed = changed || c
		n.False, c = walkExprWithStmt(n.False, stmtFn, fn)
		changed = changed || c

	case *expression.Call:
		n.Func, c = walkExprWithStmt(n.Func, stmtFn, fn)
		changed = changed || c
		for i, a := range n.Args {
			n.Args[i], c = walkExprWithStmt(a, stmtFn, fn)
			changed = changed || c
		}

	case *expression.MethodCall:
		n.Object, c = walkExprWithStmt(n.Object, stmtFn, fn)
		changed = changed || c
		for i, a := range n.Args {
			n.Args[i], c = walkExprWithStmt(a, stmtFn, fn)
			changed = changed || c
		}

	case *expression.Index:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
		n.Index, c = walkExprWithStmt(n.Index, stmtFn, fn)
		changed = changed || c

	case *expression.Selector:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c
		// Sel is an Ident/expression selector; we do not fold it.

	case *expression.Slice:
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

	case *composite.Array:
		for i, elem := range n.Elements {
			n.Elements[i], c = walkExprWithStmt(elem, stmtFn, fn)
			changed = changed || c
		}

	case *composite.Record:
		for _, el := range n.Elements {
			el.Value, c = walkExprWithStmt(el.Value, stmtFn, fn)
			changed = changed || c
		}

	case *expression.Function:
		newBody, bc := walkStmt(n.Body, stmtFn, fn)
		if bc {
			if bs, ok := newBody.(*statement.Block); ok {
				n.Body = bs
			}
		}
		changed = changed || bc

	case *expression.Immutable:
		n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, fn)
		changed = changed || c

	case *expression.FString:
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

// walkStmt walks s bottom-up. If stmtFn is non-nil it is applied to every Stmt after recursing. If exprFn is non-nil,
// expressions embedded in statements are rewritten as well.
func walkStmt(s ast.Statement, stmtFn stmtRewriteFn, exprFn exprRewriteFn) (ast.Statement, bool) {
	if s == nil {
		return nil, false
	}

	var changed bool
	var c bool
	switch n := s.(type) {
	case *statement.Block:
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

	case *statement.Expression:
		if exprFn != nil || stmtFn != nil {
			n.Expr, c = walkExprWithStmt(n.Expr, stmtFn, exprFn)
			changed = changed || c
		}

	case *statement.Assign:
		if exprFn != nil || stmtFn != nil {
			for i, e := range n.LHS {
				// LHS may be an ident being assigned; do not rewrite the plain
				// ident target itself (folding a target would break semantics),
				// but do descend into index/selector targets.
				switch e.(type) {
				case *expression.Identifier:
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

	case *statement.If:
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
			if bs, ok := r.(*statement.Block); ok {
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

	case *statement.For:
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
			if bs, ok := r.(*statement.Block); ok {
				n.Body = bs
			}
			changed = true
		}

	case *statement.ForIn:
		if exprFn != nil || stmtFn != nil {
			n.Iterable, c = walkExprWithStmt(n.Iterable, stmtFn, exprFn)
			changed = changed || c
		}
		if r, bc := walkStmt(n.Body, stmtFn, exprFn); bc {
			if bs, ok := r.(*statement.Block); ok {
				n.Body = bs
			}
			changed = true
		}

	case *statement.Return:
		if n.Result != nil && (exprFn != nil || stmtFn != nil) {
			n.Result, c = walkExprWithStmt(n.Result, stmtFn, exprFn)
			changed = changed || c
		}

	case *statement.Export:
		if exprFn != nil || stmtFn != nil {
			n.Result, c = walkExprWithStmt(n.Result, stmtFn, exprFn)
			changed = changed || c
		}

	case *statement.Defer:
		if exprFn != nil || stmtFn != nil {
			n.Call, c = walkExprWithStmt(n.Call, stmtFn, exprFn)
			changed = changed || c
		}

	case *statement.IncDec:
		// Do NOT rewrite the LHS of ++/-- (it is an assignment target).
		// Only descend into IndexExpr / SelectorExpr targets to fold their sub-parts.
		if exprFn != nil || stmtFn != nil {
			switch t := n.Expr.(type) {
			case *expression.Index:
				t.Index, c = walkExprWithStmt(t.Index, stmtFn, exprFn)
				changed = changed || c
			case *expression.Selector:
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

// walkFile dispatches walking to the appropriate helper based on the root node type. Root is typically *parser.File
// but may be a bare Stmt/Expr.
func walkFile(n ast.Node, stmtFn stmtRewriteFn, exprFn exprRewriteFn) (ast.Node, bool) {
	if n == nil {
		return nil, false
	}

	switch t := n.(type) {
	case *ast.File:
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

	case ast.Statement:
		return walkStmt(t, stmtFn, exprFn)

	case ast.Expression:
		return walkExprWithStmt(t, stmtFn, exprFn)
	}

	return n, false
}

// isLiteralExpr returns true if the expression is a scalar literal AST node that can be safely used as a compile-time
// constant.
func isLiteralExpr(e ast.Expression) bool {
	switch e.(type) {
	case *scalar.Int, *scalar.Float, *scalar.Decimal,
		*scalar.Bool, *scalar.String, *scalar.Rune,
		*scalar.Byte, *expression.Undefined,
		*scalar.Bytes, *scalar.Runes, *scalar.Time:
		return true
	}
	return false
}

// literalToValue converts a literal expression to a core.Value. Returns (Undefined, false) when the node is not a
// scalar literal we can evaluate at compile time.
func literalToValue(e ast.Expression) (core.Value, bool) {
	switch n := e.(type) {
	case *scalar.Int:
		return core.IntValue(n.Value), true
	case *scalar.Float:
		return core.FloatValue(n.Value), true
	case *scalar.Decimal:
		return core.NewDecimalValue(n.Value), true
	case *scalar.Bool:
		if n.Value {
			return core.True, true
		}
		return core.False, true
	case *scalar.String:
		return core.NewStringValue(n.Value), true
	case *scalar.Rune:
		return core.RuneValue(n.Value), true
	case *scalar.Byte:
		return core.ByteValue(n.Value), true
	case *expression.Undefined:
		return core.Undefined, true
	case *scalar.Runes:
		return core.NewRunesValue(n.Value, true), true
	case *scalar.Bytes:
		return core.NewBytesValue(n.Value, true), true
	case *scalar.Time:
		return core.NewTimeValue(n.Value), true
	}
	return core.Undefined, false
}

// safeValueToLiteral converts a runtime value back into an AST literal, if a safe round-trip is possible. Only
// scalar / immutable types are supported so we never introduce shared mutable containers as constants.
func safeValueToLiteral(v core.Value, pos core.Pos) (ast.Expression, bool) {
	switch v.Type {
	case value.Undefined:
		return &expression.Undefined{TokenPos: pos}, true

	case value.Bool:
		b := v.Data != 0
		lit := "false"
		if b {
			lit = "true"
		}
		return &scalar.Bool{Value: b, ValuePos: pos, Literal: lit}, true

	case value.Int:
		i := int64(v.Data)
		return &scalar.Int{Value: i, ValuePos: pos, Literal: strconv.FormatInt(i, 10)}, true

	case value.Float:
		if f, ok := v.AsFloat(); ok {
			return &scalar.Float{Value: f, ValuePos: pos, Literal: strconv.FormatFloat(f, 'g', -1, 64)}, true
		}

	case value.Decimal:
		if d, ok := v.AsDecimal(); ok {
			return &scalar.Decimal{Value: d, ValuePos: pos, Literal: d.String() + "d"}, true
		}

	case value.String:
		if s, ok := v.AsString(); ok {
			return &scalar.String{Value: s, ValuePos: pos, Literal: strconv.Quote(s)}, true
		}

	case value.Rune:
		r := rune(v.Data)
		return &scalar.Rune{Value: r, ValuePos: pos, Literal: strconv.QuoteRune(r)}, true

	case value.Byte:
		return &scalar.Byte{Value: byte(v.Data), ValuePos: pos, Literal: fmt.Sprintf("'\\x%02x'", byte(v.Data))}, true

	case value.Time:
		if t, ok := v.AsTime(); ok {
			return &scalar.Time{Value: t, ValuePos: pos, Literal: `"` + t.Format(time.RFC3339Nano) + `"`}, true
		}
	}

	// Container / iterator / function-shaped values are intentionally not converted back — a folded shared reference
	// would break identity / mutability semantics.
	return nil, false
}

// isTruthyLiteral returns (truthy, isConst). isConst==true iff e is a literal whose truthiness is known at
// compile time. Falls back to Kavun's runtime truthiness table (see docs/language.md).
func isTruthyLiteral(e ast.Expression) (bool, bool) {
	v, ok := literalToValue(e)
	if !ok {
		return false, false
	}
	return v.IsTrue(), true
}

// isBuiltinName reports whether name is a globally-defined pure builtin function. Used to allow calls like `len("abc")`
// inside foldable subtrees.
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

// shadowedBuiltinsIn returns the set of builtin names that are assigned anywhere within root. Any call to such a name
// at runtime may resolve to a user value and must NOT be constant-folded.
func shadowedBuiltinsIn(root ast.Node) map[string]bool {
	out := make(map[string]bool)
	usage := collectNameUsage(root)
	for name, u := range usage {
		if u.writes > 0 && isBuiltinPureName(name) {
			out[name] = true
		}
	}
	return out
}

// isFoldableExpr checks whether every leaf of the subtree is either a scalar literal or a call to a pure builtin, and
// that only pure operator/method nodes appear internally. Nothing that could observe external state or mutable
// references is allowed.
//
// shadowed is a set of identifier names that must NOT be treated as builtin callables even though vm.BuiltinFunctions
// knows a builtin by that name — they have been re-assigned somewhere in the enclosing scope and the runtime value at
// the call site may be an arbitrary user value.
func isFoldableExpr(e ast.Expression, shadowed map[string]bool) bool {
	if e == nil {
		return false
	}

	switch n := e.(type) {
	case *scalar.Int, *scalar.Float, *scalar.Decimal,
		*scalar.Bool, *scalar.String, *scalar.Rune,
		*scalar.Byte, *expression.Undefined,
		*scalar.Bytes, *scalar.Runes, *scalar.Time:
		return true

	case *expression.Parenthesis:
		return isFoldableExpr(n.Expr, shadowed)

	case *expression.Unary:
		return isFoldableExpr(n.Expr, shadowed)

	case *expression.Binary:
		return isFoldableExpr(n.LHS, shadowed) && isFoldableExpr(n.RHS, shadowed)

	case *expression.Ternary:
		return isFoldableExpr(n.Cond, shadowed) && isFoldableExpr(n.True, shadowed) && isFoldableExpr(n.False, shadowed)

	case *expression.Index:
		return isFoldableExpr(n.Expr, shadowed) && isFoldableExpr(n.Index, shadowed)

	case *expression.Slice:
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

	case *expression.MethodCall:
		// Receiver must be foldable, and every argument must be foldable. The higher-order rule from docs/purity.md is
		// enforced implicitly: FuncLit / Ident / CallExpr / MethodCallExpr arguments that carry a function value never
		// satisfy isFoldableExpr, so any callback would disqualify the tree here.
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

	case *expression.Call:
		// Only pure builtin function calls are allowed. The callee must be a bare identifier naming a
		// globally-registered pure builtin that has NOT been shadowed by any assignment in the surrounding scope.
		id, ok := n.Func.(*expression.Identifier)
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

	case *expression.FString:
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

// evalConstantExpr speculatively compiles and runs expr in an isolated compiler + VM sandbox. Returns the runtime value
// on success. The sandbox has:
//   - Fresh symbol table with only builtin function names.
//   - Empty allowed-modules set (no imports).
//   - No custom modules.
//   - Optimization disabled (avoids recursive folding).
//   - A cancellation deadline enforced by aborting the VM.
func evalConstantExpr(expr ast.Expression, fset *ast.SourceFileSet) (core.Value, bool) {
	// Defensive recover: any panic in the isolated compiler/VM stack is treated as "not foldable" and leaves the
	// original subtree untouched.
	var result core.Value
	var ok bool
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

func evalConstantExprUnsafe(expr ast.Expression, fset *ast.SourceFileSet) (core.Value, bool) {
	if fset == nil {
		fset = ast.NewFileSet()
	}
	srcFile := fset.AddFile("<opt>", -1, 0)

	// Isolated symbol table: only builtins are visible. Importantly, we do NOT copy any parent symbols so identifier
	// references (which are excluded by isFoldableExpr) would fail to compile if one slipped through.
	symTable := NewSymbolTable()

	// Empty allowed-modules set (not nil) so imports are disallowed.
	// isFoldable already excludes ImportExpr but this is defense-in-depth.
	emptyAllowed := []string{}

	c := NewCompiler(O0(), nil, srcFile, symTable, emptyAllowed, nil, nil)
	c.SetAssignmentMode(AssignmentModeSmart)

	// Build a synthetic AST: `__opt_result__ := expr`
	pos := expr.Pos()
	target := &expression.Identifier{Name: "__opt_result__", NamePos: pos}
	assign := &statement.Assign{
		LHS:      []ast.Expression{target},
		RHS:      []ast.Expression{expr},
		Token:    token.Define,
		TokenPos: pos,
	}
	file := &ast.File{InputFile: srcFile, Stmts: []ast.Statement{assign}}
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

// stmtToBlock ensures s is a *BlockStmt.
func stmtToBlock(s ast.Statement, at core.Pos) *statement.Block {
	if s == nil {
		return &statement.Block{LBrace: at, RBrace: at, Stmts: nil}
	}
	if b, ok := s.(*statement.Block); ok {
		return b
	}
	return &statement.Block{
		LBrace: s.Pos(),
		RBrace: s.End(),
		Stmts:  []ast.Statement{s},
	}
}

// isTerminatorStmt returns true when s always exits the containing block.
func isTerminatorStmt(s ast.Statement) bool {
	switch t := s.(type) {
	case *statement.Return:
		return true
	case *statement.Branch:
		return t.Token == token.Break || t.Token == token.Continue
	}
	return false
}

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

// markLHSAddressed marks every ident inside a compound LHS target as "addressed" — such idents refer to a container
// that is about to be mutated. Propagation must never replace them with a literal value.
func markLHSAddressed(e ast.Expression, get func(string) *nameUsage) {
	switch n := e.(type) {
	case *expression.Identifier:
		u := get(n.Name)
		u.addressed = true
	case *expression.Selector:
		markLHSAddressed(n.Expr, get)
	case *expression.Index:
		markLHSAddressed(n.Expr, get)
	case *expression.Slice:
		markLHSAddressed(n.Expr, get)
	case *expression.Parenthesis:
		markLHSAddressed(n.Expr, get)
	}
}

// collectNameUsage walks the AST and records how each named identifier is used. Used for very conservative
// propagation / dead-code checks: whenever we plan to remove or replace a binding we require its usage record to
// satisfy a strict pattern (read-only, no closure/loop/defer).
func collectNameUsage(root ast.Node) map[string]*nameUsage {
	usage := make(map[string]*nameUsage)
	get := func(name string) *nameUsage {
		u, ok := usage[name]
		if !ok {
			u = &nameUsage{}
			usage[name] = u
		}
		return u
	}

	var inFunc int
	var inLoop int
	var inDefer int
	var walkE func(e ast.Expression, isRead bool)
	var walkS func(s ast.Statement)

	walkE = func(e ast.Expression, isRead bool) {
		if e == nil {
			return
		}

		switch n := e.(type) {
		case *expression.Identifier:
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

		case *expression.Parenthesis:
			walkE(n.Expr, isRead)

		case *expression.Binary:
			walkE(n.LHS, true)
			walkE(n.RHS, true)

		case *expression.Unary:
			walkE(n.Expr, true)

		case *expression.Ternary:
			walkE(n.Cond, true)
			walkE(n.True, true)
			walkE(n.False, true)

		case *expression.Call:
			walkE(n.Func, true)
			for _, a := range n.Args {
				walkE(a, true)
			}

		case *expression.MethodCall:
			walkE(n.Object, true)
			for _, a := range n.Args {
				walkE(a, true)
			}

		case *expression.Index:
			walkE(n.Expr, true)
			walkE(n.Index, true)

		case *expression.Selector:
			walkE(n.Expr, true)

		case *expression.Slice:
			walkE(n.Expr, true)
			walkE(n.Low, true)
			walkE(n.High, true)
			walkE(n.Step, true)

		case *composite.Array:
			for _, elem := range n.Elements {
				walkE(elem, true)
			}

		case *composite.Record:
			for _, el := range n.Elements {
				walkE(el.Value, true)
			}

		case *expression.Function:
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

		case *expression.Immutable:
			walkE(n.Expr, true)

		case *expression.FString:
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

	walkS = func(s ast.Statement) {
		if s == nil {
			return
		}

		switch n := s.(type) {
		case *statement.Block:
			for _, sub := range n.Stmts {
				walkS(sub)
			}

		case *statement.Expression:
			walkE(n.Expr, true)

		case *statement.Assign:
			// LHS ident is a write; other targets recurse as reads (index/selector receivers are read to locate the
			// target). For compound LHS targets (SelectorExpr / IndexExpr), any embedded ident is being used as the
			// base of a mutation — mark it addressed so propagation refuses to replace it with a literal (which would
			// break the store target).
			for _, lh := range n.LHS {
				switch t := lh.(type) {
				case *expression.Identifier:
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
					if id, ok := lh.(*expression.Identifier); ok {
						u := get(id.Name)
						u.reads++
						u.addressed = true
					}
				}
			}

		case *statement.If:
			walkS(n.Init)
			walkE(n.Cond, true)
			walkS(n.Body)
			walkS(n.Else)

		case *statement.For:
			inLoop++
			walkS(n.Init)
			walkE(n.Cond, true)
			walkS(n.Post)
			walkS(n.Body)
			inLoop--

		case *statement.ForIn:
			inLoop++
			u := get(n.Key.String())
			u.writes++
			u.takenAsAssignTarget = true
			if n.Value != nil {
				u := get(n.Value.String())
				u.writes++
				u.takenAsAssignTarget = true
			}
			walkE(n.Iterable, true)
			walkS(n.Body)
			inLoop--

		case *statement.Return:
			if n.Result != nil {
				walkE(n.Result, true)
			}

		case *statement.Export:
			walkE(n.Result, true)

		case *statement.Defer:
			inDefer++
			walkE(n.Call, true)
			inDefer--

		case *statement.IncDec:
			if id, ok := n.Expr.(*expression.Identifier); ok {
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
	case *ast.File:
		for _, s := range t.Stmts {
			walkS(s)
		}

	case ast.Statement:
		walkS(t)

	case ast.Expression:
		walkE(t, true)
	}

	return usage
}
