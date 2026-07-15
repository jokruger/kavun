package compiler

import (
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/ast/expression"
	"github.com/jokruger/kavun/ast/expression/composite"
	"github.com/jokruger/kavun/ast/statement"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/vm"
)

// OptimizationConfig controls which AST optimization passes are enabled and how many times the optimization loop
// runs. Each pass is independently gated by a boolean flag. MaxPasses caps the number of full optimization cycles;
// the loop exits early once a full cycle produces no changes.
//
// Design notes:
//   - Passes are grouped by cost/risk. O0 disables everything; O1 runs cheap value-preserving rewrites; O2 adds
//     dead-code and branch simplification; O3 adds interprocedural analysis.
//   - The unified constant-folding pass (FoldConstantSubexpressions) subsumes several patterns that would otherwise
//     be implemented as separate detectors (arithmetic, string concat, builtin calls with literal args, indexing on
//     literals, f-string collapse, ...). It works by speculatively compiling+running eligible subtrees, which keeps
//     the optimizer in sync with the runtime automatically as new types/operators/methods are added.
//   - OnPass, if set, is invoked after each pass with (pass name, changed?). Useful for tracing/regression debugging
//     when implementing pass bodies.
type OptimizationConfig struct {
	MaxPasses int
	OnPass    func(name string, changed bool)

	// Unified constant folding via speculative evaluation (O1). Subsumes what would otherwise be split into
	// FoldConstantExpressions, FoldConstantBuiltinCalls, FoldConstantIndexing, FoldConstantFString, and
	// FoldStringConcatChains.
	FoldConstantSubexpressions bool

	// Structural/logical simplifications that don't require full evaluation of operands (O1).
	FoldLogicalShortCircuit bool

	// Propagation of values and copies into use sites (O1-O2).
	PropagateConstants bool
	CopyPropagation    bool

	// Dead code and branch simplification (O2).
	SimplifyConstantConditions          bool
	EliminateDeadBranches               bool
	EliminateUnreachableAfterTerminator bool
	EliminateDeadAssignments            bool

	// Interprocedural optimizations (O3).
	FoldPureFunctionCalls bool
	InlinePureFunctions   bool
}

// SetO0 disables all optimizations; no passes run.
func (oc *OptimizationConfig) SetO0() {
	*oc = OptimizationConfig{}
}

// SetO1 enables the unified constant folder plus cheap structural/logical simplifications and constant propagation.
// MaxPasses = 2 so that folding→propagation→folding can converge in a single Optimize invocation.
func (oc *OptimizationConfig) SetO1() {
	oc.SetO0()
	oc.MaxPasses = 2
	oc.FoldConstantSubexpressions = true
	oc.FoldLogicalShortCircuit = true
	oc.PropagateConstants = true
}

// SetO2 adds copy propagation and dead-code/branch simplification on top of O1. MaxPasses = 3 for deeper convergence.
func (oc *OptimizationConfig) SetO2() {
	oc.SetO1()
	oc.MaxPasses = 3
	oc.CopyPropagation = true
	oc.SimplifyConstantConditions = true
	oc.EliminateDeadBranches = true
	oc.EliminateUnreachableAfterTerminator = true
	oc.EliminateDeadAssignments = true
}

// SetO3 enables interprocedural passes (pure-function folding and small-function inlining). MaxPasses = 10 to allow
// inlining to expose more folding opportunities across iterations.
func (oc *OptimizationConfig) SetO3() {
	oc.SetO2()
	oc.MaxPasses = 10
	oc.FoldPureFunctionCalls = true
	oc.InlinePureFunctions = true
}

func O0() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO0()
	return oc
}

func O1() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO1()
	return oc
}

func O2() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO2()
	return oc
}

func O3() *OptimizationConfig {
	oc := &OptimizationConfig{}
	oc.SetO3()
	return oc
}

// Optimize runs the AST optimization pipeline, re-iterating until no changes occur or MaxPasses is reached.
func (c *Compiler) Optimize(node ast.Node) (ast.Node, error) {
	if c.oc == nil || c.oc.MaxPasses <= 0 {
		return node, nil
	}

	var err error
	var changed bool
	pass := 0
	for {
		node, changed, err = c.optimize(node)
		if err != nil {
			return nil, err
		}
		if !changed {
			break
		}
		pass++
		if pass >= c.oc.MaxPasses {
			break
		}
	}

	return node, nil
}

// optimizationPass names and enables a single AST rewrite pass.
type optimizationPass struct {
	name    string
	enabled bool
	fn      func(ast.Node) (ast.Node, bool, error)
}

// passes returns the ordered pipeline for one optimization cycle.
func (c *Compiler) passes() []optimizationPass {
	return []optimizationPass{
		{"foldLogicalShortCircuit", c.oc.FoldLogicalShortCircuit, c.foldLogicalShortCircuit},
		{"foldConstantSubexpressions", c.oc.FoldConstantSubexpressions, c.foldConstantSubexpressions},
		{"copyPropagation", c.oc.CopyPropagation, c.copyPropagation},
		{"propagateConstants", c.oc.PropagateConstants, c.propagateConstants},
		{"simplifyConstantConditions", c.oc.SimplifyConstantConditions, c.simplifyConstantConditions},
		{"eliminateDeadBranches", c.oc.EliminateDeadBranches, c.eliminateDeadBranches},
		{"eliminateUnreachableAfterTerminator", c.oc.EliminateUnreachableAfterTerminator, c.eliminateUnreachableAfterTerminator},
		{"eliminateDeadAssignments", c.oc.EliminateDeadAssignments, c.eliminateDeadAssignments},
	}
}

// optimize runs one full cycle of all enabled AST passes in the order returned by passes().
// Returns the (possibly modified) node and a boolean indicating whether any pass reported changes.
func (c *Compiler) optimize(node ast.Node) (ast.Node, bool, error) {
	var any bool
	for _, p := range c.passes() {
		if !p.enabled {
			continue
		}
		var changed bool
		var err error
		node, changed, err = p.fn(node)
		if err != nil {
			return nil, false, err
		}
		if c.oc.OnPass != nil {
			c.oc.OnPass(p.name, changed)
		}
		any = any || changed
	}
	return node, any, nil
}

// foldConstantSubexpressions walks the AST and reduces any subtree that qualifies as a "constant sub-expression" to
// a single literal. This is a UNIFIED pass — instead of implementing a separate detector for each operator, builtin,
// or literal shape (arithmetic, string concat, len/type_name/etc. on literals, indexing on literal collections,
// f-string with only literal interpolations, ...), it delegates to tryEvaluateConstant which speculatively compiles
// and runs the subtree.
//
// Rationale for the unified approach:
//   - Kavun is dynamically typed with per-operator coercion rules that differ across type combinations (e.g. `"a"+1`
//     is legal, `1+"a"` is a runtime error; `==` is coercive; decimal arithmetic has precision rules; etc.). Re-
//     implementing those rules in the optimizer is a common source of subtle divergence bugs.
//   - Reusing the compiler + VM guarantees the folded result is byte-identical to the runtime result.
//   - When new types, operators, or builtin methods are added, this pass automatically supports them with no changes.
//
// Requirements on the language for this pass to remain correct (see docs/purity.md for the full contract):
//   - All operators (UnaryOp, BinaryOp) are pure by contract.
//   - All methods (MethodCall) are pure w.r.t. the receiver and external state. Higher-order methods pass
//     through the purity of their function arguments — the method itself is pure; any impurity comes from a
//     supplied callback. The optimizer excludes such calls by refusing to fold any MethodCall with a
//     function-shaped argument.
//   - The escape hatch for mutation is a method name ending in `_in_place`; such methods are treated as impure.
//   - Append is Go-style (may alias the receiver's backing storage) and is not foldable.
//   - Builtin functions expose the Pure metadata bit (see core.BuiltinFunction.Pure); user-defined functions are
//     proven pure by the interprocedural pass before folding calls to them.
//
// Eligibility check (per subtree, before calling tryEvaluateConstant):
//   - Only literal leaves and calls to pure builtins/methods.
//   - No identifier references (they may resolve to shadowed or reassigned builtins, or to closed-over mutable
//     state; see docs/language.md on builtin shadowing).
//   - No FuncLit values (identity-sensitive).
//   - Deterministic operators only (no time, random, I/O, imports).
//   - Container literal results (arrays/records/dicts) are folded ONLY when the result cannot be mutated afterwards,
//     e.g. when the folded expression is immediately consumed by a scalar-producing operator/method (len, indexing,
//     comparison) and does not escape as a stored/returned value.
//
// Runtime-error handling: see tryEvaluateConstant. On error the subtree is left untouched by default.
//
// This pass subsumes what would otherwise be separate FoldConstantExpressions, FoldConstantBuiltinCalls,
// FoldConstantIndexing, FoldConstantFString, and FoldStringConcatChains passes.
func (c *Compiler) foldConstantSubexpressions(node ast.Node) (ast.Node, bool, error) {
	fset := c.file.Set()
	shadowed := shadowedBuiltinsIn(node)

	rewriteExpr := func(e ast.Expression) (ast.Expression, bool) {
		// Skip nodes that are already atomic literals — nothing to gain.
		if isLiteralExpr(e) {
			return e, false
		}
		// Only try to fold if the entire subtree is eligible.
		if !isFoldableExpr(e, shadowed) {
			return e, false
		}
		// Special-case: literals which should be handled by wrapping operators rather than by direct folding.
		switch e.(type) {
		case *composite.Array, *composite.Record:
			return e, false
		}

		v, ok := evalConstantExpr(e, fset)
		if !ok {
			return e, false
		}
		lit, ok := safeValueToLiteral(v, e.Pos())
		if !ok {
			return e, false
		}
		return lit, true
	}

	n, changed := walkFile(node, nil, rewriteExpr)
	return n, changed, nil
}

// foldLogicalShortCircuit simplifies `&&` and `||` when the LHS is a compile-time constant, using Kavun's truthiness
// table (see docs/language.md):
//   - true  && x → x         (LHS truthy: result is RHS)
//   - false && x → false     (LHS falsy: RHS is NOT evaluated)
//   - true  || x → true      (LHS truthy: RHS is NOT evaluated)
//   - false || x → x         (LHS falsy: result is RHS)
//
// Correctness notes:
//   - Runs BEFORE foldConstantSubexpressions so short-circuit can prune an expensive/impure RHS before the folder
//     touches it.
//   - When the RHS is discarded (`false && x`, `true || x`), any side effects in x are ALSO discarded. This matches
//     the language's short-circuit semantics, so no compensating ExprStmt is required.
//   - LHS truthiness follows Kavun's rules: undefined, false, 0 (int), decimal(0), "", [], {}, empty dict are falsy;
//     everything else (including 0.0 float, non-empty containers, non-zero numerics) is truthy.
//   - Do NOT rewrite `x && y` or `x || y` when neither side is a constant — Kavun returns one of the operand values
//     (not a normalized bool), and consumers may depend on that identity.
func (c *Compiler) foldLogicalShortCircuit(node ast.Node) (ast.Node, bool, error) {
	rewriteExpr := func(e ast.Expression) (ast.Expression, bool) {
		be, ok := e.(*expression.Binary)
		if !ok {
			return e, false
		}
		if be.Token != token.LAnd && be.Token != token.LOr {
			return e, false
		}
		// Only fire when LHS is a scalar literal — we need to know its truthiness at compile time.
		truthy, isConst := isTruthyLiteral(be.LHS)
		if !isConst {
			return e, false
		}
		switch be.Token {
		case token.LAnd:
			if truthy {
				// true && x → x
				return be.RHS, true
			}
			// false && x → LHS (short-circuits, discards x)
			return be.LHS, true
		case token.LOr:
			if truthy {
				// true || x → LHS
				return be.LHS, true
			}
			// false || x → x
			return be.RHS, true
		}
		return e, false
	}

	n, changed := walkFile(node, nil, rewriteExpr)
	return n, changed, nil
}

// copyPropagation replaces uses of a variable `y` that is initialized as a bare copy of another variable `x`,
// i.e. `y := x; use(y)`, with `x` at every use site. Safety requirements (all must hold):
//   - Both x and y are top-level file idents.
//   - y is single-assignment.
//   - y is not referenced from inside a FuncLit, defer, or as an assign target.
//   - x is not written after the copy point.
//   - x is not referenced from inside a FuncLit that could rebind it.
//   - x is not `undefined`.
//
// Enables further folding by unifying references, so it combines well with propagateConstants and dead-assignment
// elimination in the same cycle.
func (c *Compiler) copyPropagation(node ast.Node) (ast.Node, bool, error) {
	file, ok := node.(*ast.File)
	if !ok {
		return node, false, nil
	}

	usage := collectNameUsage(file)
	copies := make(map[string]string) // y → x
	for _, s := range file.Stmts {
		as, ok := s.(*statement.Assign)
		if !ok {
			continue
		}
		if len(as.LHS) != 1 || len(as.RHS) != 1 {
			continue
		}
		if as.Token != token.Define {
			continue
		}
		yIdent, yok := as.LHS[0].(*ast.Identifier)
		xIdent, xok := as.RHS[0].(*ast.Identifier)
		if !yok || !xok {
			continue
		}
		if yIdent.Name == xIdent.Name {
			continue
		}
		// x must be a real user variable (not a builtin function shadow) and must be stable.
		yU, yhas := usage[yIdent.Name]
		xU, xhas := usage[xIdent.Name]
		if !yhas || !xhas {
			continue
		}
		if yU.writes != 1 || yU.insideFuncLit || yU.insideDefer || yU.addressed {
			continue
		}
		// x must be writable exactly once (its own declaration) and free of closure capture that could rebind it later.
		if xU.writes != 1 || xU.insideFuncLit || xU.insideDefer || xU.addressed {
			continue
		}
		// x must not resolve to a builtin (builtins can be shadowed but we avoid propagating names that could clash).
		if _, isBuiltin := vm.BuiltinFunctions[xIdent.Name]; isBuiltin {
			continue
		}
		if _, isBuiltin := vm.BuiltinFunctions[yIdent.Name]; isBuiltin {
			continue
		}
		copies[yIdent.Name] = xIdent.Name
	}
	if len(copies) == 0 {
		return node, false, nil
	}

	// Resolve chains y → x → z...
	resolve := func(name string) string {
		seen := map[string]bool{}
		for {
			if seen[name] {
				return name
			}
			seen[name] = true
			nxt, ok := copies[name]
			if !ok {
				return name
			}
			name = nxt
		}
	}

	changed := false
	rewriteExpr := func(e ast.Expression) (ast.Expression, bool) {
		id, ok := e.(*ast.Identifier)
		if !ok {
			return e, false
		}
		target := resolve(id.Name)
		if target == id.Name {
			return e, false
		}
		changed = true
		return &ast.Identifier{Name: target, NamePos: id.NamePos}, true
	}

	n, walkChanged := walkFile(file, nil, rewriteExpr)
	return n, changed || walkChanged, nil
}

// propagateConstants replaces reads of variables declared as `x := <literal>` (or `var x = <literal>`) with the literal
// itself, but only in strictly safe cases:
//   - The declaration is a single-LHS := / assign with a literal RHS.
//   - The variable is NEVER written after that point (only reads).
//   - The variable is NEVER referenced from inside a FuncLit (closures capture
//     by reference).
//   - The variable is NEVER referenced from inside a defer.
//   - The variable is NEVER a for-loop-in bound (each iteration re-binds).
//
// This is conservative: it only fires for top-level declarations in the File stmt list (walk semantics guarantee
// sequential scope), and it does not propagate across function boundaries. Even so, it composes with folding to turn
// `x := 2; y := x + 3` into `x := 2; y := 5` in one optimization cycle.
func (c *Compiler) propagateConstants(node ast.Node) (ast.Node, bool, error) {
	file, ok := node.(*ast.File)
	if !ok {
		return node, false, nil
	}

	usage := collectNameUsage(file)
	consts := make(map[string]ast.Expression) // name → literal
	for _, s := range file.Stmts {
		as, ok := s.(*statement.Assign)
		if !ok {
			continue
		}
		if len(as.LHS) != 1 || len(as.RHS) != 1 {
			continue
		}
		if as.Token != token.Define && as.Token != token.Assign {
			continue
		}
		id, ok := as.LHS[0].(*ast.Identifier)
		if !ok {
			continue
		}
		if !isLiteralExpr(as.RHS[0]) {
			continue
		}
		// Never propagate builtin names — the identifier may be used as a function callee elsewhere (`len(x)`), and
		// replacing it with a value literal would change program semantics.
		if _, isBuiltin := vm.BuiltinFunctions[id.Name]; isBuiltin {
			continue
		}
		u, ok := usage[id.Name]
		if !ok {
			continue
		}
		// Must be single-assignment (exactly the declaration counts), no closure captures, no defer, no loop rebinding.
		if u.writes != 1 || u.insideFuncLit || u.insideDefer || u.addressed {
			continue
		}
		consts[id.Name] = as.RHS[0]
	}
	if len(consts) == 0 {
		return node, false, nil
	}

	// Rewrite reads of tracked idents to the corresponding literal. Avoid rewriting occurrences at LHS positions
	// (walker already skips plain-ident LHS in AssignStmt / IncDecStmt).
	changed := false
	rewriteExpr := func(e ast.Expression) (ast.Expression, bool) {
		id, ok := e.(*ast.Identifier)
		if !ok {
			return e, false
		}
		lit, ok := consts[id.Name]
		if !ok {
			return e, false
		}
		// Clone the literal so each use has its own position (safe since literals are immutable value carriers).
		if v, ok := literalToValue(lit); ok {
			if cloned, ok := safeValueToLiteral(v, id.Pos()); ok {
				changed = true
				return cloned, true
			}
		}
		return e, false
	}

	n, walkChanged := walkFile(file, nil, rewriteExpr)
	return n, changed || walkChanged, nil
}

// simplifyConstantConditions folds `if` and ternary (`?:`) expressions whose condition is a compile-time-constant
// truthiness value:
//   - if C { A } else { B } → A   when C is truthy
//   - if C { A } else { B } → B   when C is falsy (or <empty> if no else clause)
//   - C ? A : B                  → same
//
// Correctness requirements:
//   - Preserve any init statement of the if: `if x := f(); true { A }` → `x := f(); A`. The init may declare a
//     variable used later via smart `=`, and its call may have side effects.
//   - Truthiness follows Kavun's table (docs/language.md): undefined, false, 0 (int), decimal(0), "", [], {}, empty
//     dict are falsy; 0.0 (float) is truthy; every other non-empty value is truthy.
//   - If the removed branch declares variables referenced later via smart `=`, do not eliminate — those references
//     would fail to resolve. In practice: run this pass only after a scope-analysis phase that has bound every
//     identifier to its declaring scope, so we know whether removing a branch strands any later reference.
//   - Do not confuse this pass with foldConstantSubexpressions on the condition: this pass eliminates a whole
//     statement branch, not just an expression. Fold-then-simplify is the intended interaction (folding runs first
//     in the pipeline).
func (c *Compiler) simplifyConstantConditions(node ast.Node) (ast.Node, bool, error) {
	// Statement-level: fold if-statements whose condition is a scalar literal.
	rewriteStmt := func(s ast.Statement) (ast.Statement, bool) {
		is, ok := s.(*statement.If)
		if !ok {
			return s, false
		}
		// A non-nil Init may declare variables referenced later via smart `=` or may have side effects; we cannot
		// silently drop it. If it is present, we conservatively refuse to rewrite the whole statement unless we can
		// safely keep the Init as a sibling. Init is always a simple statement in Kavun (usually AssignStmt); when we
		// keep it, wrap it with the surviving branch in a fresh BlockStmt.
		truthy, isConst := isTruthyLiteral(is.Cond)
		if !isConst {
			return s, false
		}
		var chosen ast.Statement
		if truthy {
			chosen = is.Body
		} else if is.Else != nil {
			chosen = is.Else
		} else {
			// No else and condition falsy: drop the branch. Preserve Init if any.
			if is.Init != nil {
				return &statement.Block{
					LBrace: is.IfPos,
					RBrace: is.End(),
					Stmts:  []ast.Statement{is.Init},
				}, true
			}
			return &statement.Block{LBrace: is.IfPos, RBrace: is.End()}, true
		}
		if is.Init != nil {
			block := stmtToBlock(chosen, is.IfPos)
			// Prepend the Init statement to preserve any variable it declares.
			newStmts := make([]ast.Statement, 0, len(block.Stmts)+1)
			newStmts = append(newStmts, is.Init)
			newStmts = append(newStmts, block.Stmts...)
			return &statement.Block{
				LBrace: is.IfPos,
				RBrace: is.End(),
				Stmts:  newStmts,
			}, true
		}
		return chosen, true
	}

	// Expression-level: fold ternary `c ? a : b` with a constant c.
	rewriteExpr := func(e ast.Expression) (ast.Expression, bool) {
		ce, ok := e.(*expression.Ternary)
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

	n, changed := walkFile(node, rewriteStmt, rewriteExpr)
	return n, changed, nil
}

// eliminateDeadBranches removes unreachable else / else-if branches that were exposed by simplifyConstantConditions
// (or that had a statically-known constant condition to begin with). Distinct from simplifyConstantConditions in
// that it targets chained `if / else if / else` where an earlier branch is provably always taken and later branches
// are provably unreachable.
func (c *Compiler) eliminateDeadBranches(node ast.Node) (ast.Node, bool, error) {
	var simplify func(is *statement.If) (ast.Statement, bool)
	simplify = func(is *statement.If) (ast.Statement, bool) {
		// Recurse first so inner ifs are simplified.
		if is.Else != nil {
			if inner, ok := is.Else.(*statement.If); ok {
				if r, c := simplify(inner); c {
					is.Else = r
				}
			}
		}
		// Only touch chained else-if forms with a compile-time-constant condition. Ignore ifs with Init
		// (see simplifyConstantConditions).
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
		return &statement.Block{LBrace: is.IfPos, RBrace: is.End()}, true
	}

	var globalChanged bool

	rewriteStmt := func(s ast.Statement) (ast.Statement, bool) {
		is, ok := s.(*statement.If)
		if !ok {
			return s, false
		}
		if r, c := simplify(is); c {
			globalChanged = true
			return r, true
		}
		return s, false
	}

	n, changed := walkFile(node, rewriteStmt, nil)
	return n, changed || globalChanged, nil
}

// eliminateUnreachableAfterTerminator removes statements that follow a terminating statement within the same block.
// Terminators: `return`, `break`, `continue`, and (reserved for the future) any call to a builtin annotated with a
// `NoReturn` metadata bit.
//
// Correctness notes:
//   - Removes statements STRICTLY AFTER the terminator; the terminator itself is preserved.
//   - Does not descend into nested blocks past a terminator (any `defer` registered BEFORE the terminator still
//     fires — defer registration is a runtime effect, not a lexical one).
//   - Distinct from eliminateDeadBranches, which handles unreachable else/else-if branches after a constant `if`.
func (c *Compiler) eliminateUnreachableAfterTerminator(node ast.Node) (ast.Node, bool, error) {
	var changed bool
	rewriteStmt := func(s ast.Statement) (ast.Statement, bool) {
		block, ok := s.(*statement.Block)
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

	n, walkChanged := walkFile(node, rewriteStmt, nil)
	return n, changed || walkChanged, nil
}

// eliminateDeadAssignments removes declarations of the form `x := <literal>` or `x := <ident>` where x is:
//   - Never read anywhere.
//   - Never captured by a FuncLit.
//   - Never referenced by a defer.
//   - Never re-assigned or addressed (single-assignment).
//   - Its RHS is side-effect-free (a literal or another identifier).
//
// When the RHS has side effects we do NOT remove the statement (to preserve observable behavior).
func (c *Compiler) eliminateDeadAssignments(node ast.Node) (ast.Node, bool, error) {
	file, ok := node.(*ast.File)
	if !ok {
		return node, false, nil
	}

	usage := collectNameUsage(file)
	changed := false
	out := file.Stmts[:0]
	for _, s := range file.Stmts {
		if as, ok := s.(*statement.Assign); ok {
			if len(as.LHS) == 1 && len(as.RHS) == 1 && as.Token == token.Define {
				if id, ok := as.LHS[0].(*ast.Identifier); ok {
					u, uok := usage[id.Name]
					sideEffectFree := isLiteralExpr(as.RHS[0])
					if !sideEffectFree {
						if _, ok := as.RHS[0].(*ast.Identifier); ok {
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
