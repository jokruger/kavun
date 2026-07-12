package compiler

import (
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/parser"
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
	SimplifyIfExprToBool                bool
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
	oc.SimplifyIfExprToBool = true
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
func (c *Compiler) Optimize(node parser.Node) (parser.Node, error) {
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
	fn      func(parser.Node) (parser.Node, bool, error)
}

// passes returns the ordered pipeline for one optimization cycle.
func (c *Compiler) passes() []optimizationPass {
	return []optimizationPass{
		{"foldLogicalShortCircuit", c.oc.FoldLogicalShortCircuit, c.foldLogicalShortCircuit},
		{"foldConstantSubexpressions", c.oc.FoldConstantSubexpressions, c.foldConstantSubexpressions},
		{"copyPropagation", c.oc.CopyPropagation, c.copyPropagation},
		{"propagateConstants", c.oc.PropagateConstants, c.propagateConstants},
		{"simplifyConstantConditions", c.oc.SimplifyConstantConditions, c.simplifyConstantConditions},
		{"simplifyIfExprToBool", c.oc.SimplifyIfExprToBool, c.simplifyIfExprToBool},
		{"eliminateDeadBranches", c.oc.EliminateDeadBranches, c.eliminateDeadBranches},
		{"eliminateUnreachableAfterTerminator", c.oc.EliminateUnreachableAfterTerminator, c.eliminateUnreachableAfterTerminator},
		{"eliminateDeadAssignments", c.oc.EliminateDeadAssignments, c.eliminateDeadAssignments},
	}
}

// optimize runs one full cycle of all enabled AST passes in the order returned by passes().
// Returns the (possibly modified) node and a boolean indicating whether any pass reported changes.
func (c *Compiler) optimize(node parser.Node) (parser.Node, bool, error) {
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

// tryEvaluateConstant attempts to reduce a subtree to a single literal by speculatively compiling and running it.
// This is the core primitive used by foldConstantSubexpressions (and any other pass that needs "if this were a
// constant, what would it be?"). The goal is to avoid duplicating per-type semantics in the optimizer: reusing the
// compiler + VM guarantees the folded result is byte-identical to the runtime result, and new types/operators/
// methods are supported automatically as they are added to core.
//
// Preconditions the caller MUST verify before invoking:
//   - Every leaf of the subtree is a literal (IntLit, FloatLit, DecimalLit, StringLit, BoolLit, RuneLit, ByteLit,
//     ArrayLit, RecordLit, dict/range literal) OR a call to a builtin whose core.BuiltinFunction.Pure == true.
//   - No identifier references — they may resolve to shadowed builtins, closures, or mutable state, and the same
//     name may resolve differently at the optimization site than at the original source site.
//   - No FuncLit — functions are values with identity; two structurally equal FuncLits are not equal at runtime.
//   - The root operation dispatches to a hook categorized as "always pure" by the project purity contract; see
//     docs/purity.md. Concretely: UnaryOp, BinaryOp, MethodCall, Access, Slice, SliceStep, and any AsX conversion
//     are foldable; Assign, Delete, Append, iterator advancement (Next/Key/Value), and Call are not.
//   - For MethodCall subtrees the higher-order rule from docs/purity.md applies: fold only when no argument is a
//     FuncLit, identifier, method value, or CallExpr — i.e. no argument can carry impurity into the method body.
//
// Evaluation strategy:
//  1. Compile the subtree into a short-lived bytecode chunk in an isolated symbol table (no imports, no globals).
//  2. Execute in a sandboxed VM with strict step and allocation budgets so a pathological literal (e.g. a huge
//     string repeat) cannot stall compilation.
//  3. Marshal the resulting core.Value back into an equivalent literal AST node at the original source position.
//
// Runtime-error handling:
//   - Leave the subtree untouched on runtime error.
//
// Non-goals:
//   - Do not evaluate anything that observes external state (time, random, environment, filesystem, imports).
//   - Do not fold results that are mutable references shared across call sites (e.g. an array/record literal
//     produced by a builtin) unless the surrounding code proves the result is never mutated. Folding a mutable
//     container into a single shared constant would change program semantics.
//
// Return values:
//   - (literalNode, true, nil): subtree was reduced.
//   - (node, false, nil):       subtree is not eligible, or the speculative run failed in a way that should preserve
//     the original tree (e.g. runtime error or budget exhausted).
//   - (nil, false, err):        optimizer-internal failure only.
func (c *Compiler) tryEvaluateConstant(node parser.Node) (parser.Node, bool, error) {
	expr, ok := node.(parser.Expr)
	if !ok {
		return node, false, nil
	}
	if isLiteralExpr(expr) {
		return node, false, nil
	}
	shadowed := shadowedBuiltinsIn(expr)
	if !isFoldableExpr(expr, shadowed) {
		return node, false, nil
	}
	v, ok := evalConstantExpr(expr, c.file.Set())
	if !ok {
		return node, false, nil
	}
	lit, ok := safeValueToLiteral(v, expr.Pos())
	if !ok {
		return node, false, nil
	}
	return lit, true, nil
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
func (c *Compiler) foldConstantSubexpressions(node parser.Node) (parser.Node, bool, error) {
	fset := c.file.Set()
	shadowed := shadowedBuiltinsIn(node)

	rewriteExpr := func(e parser.Expr) (parser.Expr, bool) {
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
		case *parser.ArrayLit, *parser.RecordLit:
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
func (c *Compiler) foldLogicalShortCircuit(node parser.Node) (parser.Node, bool, error) {
	rewriteExpr := func(e parser.Expr) (parser.Expr, bool) {
		be, ok := e.(*parser.BinaryExpr)
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
func (c *Compiler) copyPropagation(node parser.Node) (parser.Node, bool, error) {
	file, ok := node.(*parser.File)
	if !ok {
		return node, false, nil
	}

	usage := collectNameUsage(file)
	copies := make(map[string]string) // y → x
	for _, s := range file.Stmts {
		as, ok := s.(*parser.AssignStmt)
		if !ok {
			continue
		}
		if len(as.LHS) != 1 || len(as.RHS) != 1 {
			continue
		}
		if as.Token != token.Define {
			continue
		}
		yIdent, yok := as.LHS[0].(*parser.Ident)
		xIdent, xok := as.RHS[0].(*parser.Ident)
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
	rewriteExpr := func(e parser.Expr) (parser.Expr, bool) {
		id, ok := e.(*parser.Ident)
		if !ok {
			return e, false
		}
		target := resolve(id.Name)
		if target == id.Name {
			return e, false
		}
		changed = true
		return &parser.Ident{Name: target, NamePos: id.NamePos}, true
	}

	n, walkChanged := walkFile(file, nil, rewriteExpr)
	return n, changed || walkChanged, nil
}

// propagateConstants replaces reads of variables whose value is a compile-time constant with the constant literal
// itself.
//
// Kavun-specific safety requirements:
//   - Closures capture free variables BY REFERENCE (see docs/language.md). A variable read by any reachable FuncLit
//     is NOT eligible for propagation unless (a) no reachable FuncLit writes to it and (b) no external write can
//     occur between the declaration and each use point.
//   - Named return values (`func() n { ... }`) are implicitly mutable and read at return time — never propagate the
//     named-result identifier.
//   - `defer` bodies read variables at scope exit — treat defer bodies as reachable use sites.
//   - Smart `=` semantics: the FIRST `x = ...` in a scope is a declaration. Do not remove or replace that
//     declaration in a way that would eliminate the binding used by later smart-`=` references or by closures.
//   - `for k, v in ...` binds fresh values each iteration — `v`/`k` are not constants across iterations even if the
//     collection contents look constant.
//   - Builtins can be shadowed via `:=` per-script (see docs/language.md). Propagating a value that was assigned
//     from a builtin call requires resolving the builtin at the assignment site with the same scope rules the
//     compiler uses.
//
// Conservative rule (sufficient for O1/O2):
//   - Variable is declared via `:=` or `var x = ...`.
//   - Variable is never target of `=`, compound assignment, `++`, `--`, `for k, v in`, or a spread receiver after
//     its declaration.
//   - No enclosing FuncLit references the variable, OR every FuncLit reference is a read AND no reachable code path
//     between the declaration and the use can invoke a closure that writes to it.
//   - RHS is already a literal (or a subtree that foldConstantSubexpressions has reduced to a literal on an earlier
//     pass in the same optimization cycle).
func (c *Compiler) propagateConstants(node parser.Node) (parser.Node, bool, error) {
	return c.runPropagateConstants(node)
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
func (c *Compiler) simplifyConstantConditions(node parser.Node) (parser.Node, bool, error) {
	return c.runSimplifyConstantConditions(node)
}

// simplifyIfExprToBool rewrites `if cond { true } else { false }` to `cond`, and the mirror form
// `if cond { false } else { true }` to `!cond`. Only fires when ALL of the following hold:
//   - cond is provably bool (bool literal, comparison, `!`, `&&`, `||`, `in`/`not in`, or an ident annotated bool).
//   - Both branches are exactly a single bool-literal statement (either an ExprStmt or a `return`).
//   - The if has no init statement, OR the init is preserved as a preceding statement in the surrounding block.
//
// If cond is not provably bool, the rewrite would change the observable value (`if x { true } else { false }`
// returns a bool; `x` might be a non-bool truthy value like a string).
func (c *Compiler) simplifyIfExprToBool(node parser.Node) (parser.Node, bool, error) {
	return c.runSimplifyIfExprToBool(node)
}

// eliminateDeadBranches removes unreachable else / else-if branches that were exposed by simplifyConstantConditions
// (or that had a statically-known constant condition to begin with). Distinct from simplifyConstantConditions in
// that it targets chained `if / else if / else` where an earlier branch is provably always taken and later branches
// are provably unreachable.
func (c *Compiler) eliminateDeadBranches(node parser.Node) (parser.Node, bool, error) {
	return c.runEliminateDeadBranches(node)
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
func (c *Compiler) eliminateUnreachableAfterTerminator(node parser.Node) (parser.Node, bool, error) {
	return c.runEliminateUnreachableAfterTerminator(node)
}

// eliminateDeadAssignments removes assignments whose result is never read on any reachable path AND whose RHS is
// side-effect-free. If the RHS has side effects, rewrites to a bare ExprStmt so the effect is preserved.
//
// Kavun-specific safety requirements:
//   - A variable read by ANY reachable FuncLit (closure) counts as read — closures capture by reference and may be
//     invoked from arbitrary later points.
//   - A variable read by ANY defer body counts as read.
//   - A named return value is read at every explicit or implicit return — never eliminate assignments to it.
//   - Smart `=` semantics: the first `x = ...` in a scope is a declaration; eliminating it also eliminates the
//     binding, potentially changing name resolution for later `x` references (which would fall back to an outer
//     scope or become an error). Only eliminate assignments to strictly local, uniquely-scoped variables.
//   - `for k, v in ...` and named-result parameters are declarations, not eliminable assignments.
//   - Compound assignments (`+=`, `<<=`, ...) both read AND write the LHS — treat as a use of the prior value.
//   - Assignments through indexing (`a[i] = v`) or field selection (`r.k = v`) are stores into a container the
//     caller may observe — do NOT eliminate.
//
// Runs last among the intraprocedural passes because folding, propagation, and branch/condition simplification all
// tend to increase the set of dead assignments.
func (c *Compiler) eliminateDeadAssignments(node parser.Node) (parser.Node, bool, error) {
	return c.runEliminateDeadAssignments(node)
}
