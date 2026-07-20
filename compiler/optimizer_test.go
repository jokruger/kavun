package compiler_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// optCase describes one pass-under-test scenario.
type optCase struct {
	name string
	src  string
	// expected AST after Optimize().String(). Whitespace is normalized.
	wantAST string
	// pass names that must have reported changed=true at least once.
	wantChanged []string
	// pass names that must NOT have reported changed=true.
	wantUnchanged []string
	// optional: expected runtime value of variable `out` after running the
	// optimized program (nil to skip).
	wantOut any
	oc      func() *compiler.OptimizationConfig
}

// runOptCases parses src, runs the optimizer with the supplied config, and
// asserts the resulting AST + which passes reported changes.
func runOptCases(t *testing.T, cases []optCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file := parseFile(t, tc.src)
			seen := map[string]bool{}
			oc := tc.oc()
			oc.OnPass = func(name string, changed bool) {
				if changed {
					seen[name] = true
				}
			}
			// Compile to force optimization + verify it still produces valid bytecode.
			fileSet := ast.NewFileSet()
			srcFile := fileSet.AddFile("opt-test", -1, len(tc.src))
			symTable := compiler.NewSymbolTable()
			for idx, name := range vm.BuiltinFunctionNames {
				symTable.DefineBuiltin(idx, name)
			}
			// Pre-declare `out` as a global so scripts that write to it compile.
			outSym := symTable.Define("out")
			c := compiler.NewCompiler(oc, nil, srcFile, symTable, nil, nil, nil)

			// Reparse using the srcFile so the compiler's file pointer is consistent.
			p := parser.NewParser(srcFile, []byte(tc.src), nil)
			parsed, err := p.ParseFile()
			require.NoError(t, err, "parse")
			_ = file
			optimized, err := c.Optimize(parsed)
			require.NoError(t, err, "optimize")
			if tc.wantAST != "" {
				got := normalizeStr(optimized.String())
				want := normalizeStr(tc.wantAST)
				require.Equal(t, want, got, "AST mismatch for %s\nSRC=%s", tc.name, tc.src)
			}
			for _, p := range tc.wantChanged {
				require.True(t, seen[p], "expected pass %q to report changed=true", p)
			}
			for _, p := range tc.wantUnchanged {
				require.False(t, seen[p], "expected pass %q to NOT report changed=true", p)
			}
			if tc.wantOut != nil {
				err = c.CompileNode(optimized)
				require.NoError(t, err, "compile after optimize")
				bc := c.Bytecode()
				globals := make([]core.Value, vm.GlobalsSize)
				machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
				machine.Reset(bc, globals)
				err = machine.Run()
				require.NoError(t, err, "run")
				got := globals[outSym.Index]
				want := wrapPrimitive(t, tc.wantOut)
				require.True(t, got.Equal(want), "runtime value mismatch for %s: got %s (%s) want %s (%s)", tc.name, got.String(), got.TypeName(), want.String(), want.TypeName())
			}
		})
	}
}

func parseFile(t *testing.T, src string) *ast.File {
	t.Helper()
	fileSet := ast.NewFileSet()
	srcFile := fileSet.AddFile("opt-test", -1, len(src))
	p := parser.NewParser(srcFile, []byte(src), nil)
	f, err := p.ParseFile()
	require.NoError(t, err, "parse")
	return f
}

// wrapPrimitive turns a Go primitive into a core.Value for equality comparison
// against the runtime `out` global. Supports only the subset needed by the
// optimizer tests.
func wrapPrimitive(t *testing.T, v any) core.Value {
	t.Helper()
	switch x := v.(type) {
	case int:
		return core.IntValue(int64(x))
	case int64:
		return core.IntValue(x)
	case float64:
		return core.FloatValue(x)
	case bool:
		if x {
			return core.True
		}
		return core.False
	case string:
		return core.NewStringValue(x)
	case dec128.Dec128:
		return core.NewDecimalValue(x)
	default:
		t.Fatalf("wrapPrimitive: unsupported %T", v)
		return core.Undefined
	}
}

// normalizeStr collapses whitespace so AST comparisons are robust against
// formatting differences.
func normalizeStr(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// ---------------------------------------------------------------------------
// tryEvaluateConstant / foldConstantSubexpressions
// ---------------------------------------------------------------------------

func TestOptimizer_FoldConstantSubexpressions(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 3
		oc.FoldConstantSubexpressions = true
		return oc
	}
	cases := []optCase{
		{
			name:        "arithmetic",
			src:         `out = 1 + 2 * 3`,
			wantAST:     `out = 7`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     7,
			oc:          only,
		},
		{
			name:        "string concat",
			src:         `out = "a" + "b" + "c"`,
			wantAST:     `out = "abc"`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     "abc",
			oc:          only,
		},
		{
			name:        "unary neg",
			src:         `out = -(2 + 3)`,
			wantAST:     `out = -5`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     -5,
			oc:          only,
		},
		{
			name:        "builtin len on literal string",
			src:         `out = len("hello")`,
			wantAST:     `out = 5`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:        "method call on string literal",
			src:         `out = "abc".upper()`,
			wantAST:     `out = "ABC"`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     "ABC",
			oc:          only,
		},
		{
			name:        "comparison",
			src:         `out = (1 + 1) == 2`,
			wantAST:     `out = true`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:          "identifier not folded",
			src:           `x := 1; out = x + 2`,
			wantAST:       `x := 1; out = (x + 2)`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			oc:            only,
		},
		{
			name:          "shadowed builtin not folded",
			src:           `len = func(x) { return 42 }; out = len("hi")`,
			wantAST:       `len = func(x) {return 42}; out = len("hi")`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			oc:            only,
		},
		{
			name:          "runtime error preserved (division by zero)",
			src:           `out = 1 / 0`,
			wantAST:       `out = (1 / 0)`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			oc:            only,
		},
		{
			name:          "array literal not folded (mutable identity)",
			src:           `out = [1, 2, 3]`,
			wantAST:       `out = [1, 2, 3]`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// foldLogicalShortCircuit
// ---------------------------------------------------------------------------

func TestOptimizer_FoldLogicalShortCircuit(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.FoldLogicalShortCircuit = true
		return oc
	}
	cases := []optCase{
		{
			name:        "true && x → x",
			src:         `x := 5; out = true && x`,
			wantAST:     `x := 5; out = x`,
			wantChanged: []string{"foldLogicalShortCircuit"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:        "false && x → false (RHS discarded)",
			src:         `out = false && (1 / 0)`,
			wantAST:     `out = false`,
			wantChanged: []string{"foldLogicalShortCircuit"},
			wantOut:     false,
			oc:          only,
		},
		{
			name:        "true || x → true (RHS discarded)",
			src:         `out = true || (1 / 0)`,
			wantAST:     `out = true`,
			wantChanged: []string{"foldLogicalShortCircuit"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:        "false || x → x",
			src:         `x := 5; out = false || x`,
			wantAST:     `x := 5; out = x`,
			wantChanged: []string{"foldLogicalShortCircuit"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:          "non-constant LHS not simplified",
			src:           `x := 5; out = x && true`,
			wantAST:       `x := 5; out = (x && true)`,
			wantUnchanged: []string{"foldLogicalShortCircuit"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// simplifyConstantConditions
// ---------------------------------------------------------------------------

func TestOptimizer_SimplifyConstantConditions(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 3
		oc.SimplifyConstantConditions = true
		return oc
	}
	cases := []optCase{
		{
			name:        "if true keeps body only",
			src:         `out = 0; if true {out = 1} else {out = 2}`,
			wantAST:     `out = 0; if true {out = 1}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:        "if false keeps else",
			src:         `out = 0; if false {out = 1} else {out = 2}`,
			wantAST:     `out = 0; if false {} else {out = 2}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     2,
			oc:          only,
		},
		{
			name:        "if false without else drops branch",
			src:         `out = 5; if false {out = 1}`,
			wantAST:     `out = 5; if false {}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:        "ternary with constant condition",
			src:         `out = (true ? 1 : 2)`,
			wantAST:     `out = (1)`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:          "non-constant condition preserved",
			src:           `x := 1; if x > 0 { out = 1 } else { out = 2 }`,
			wantAST:       `x := 1; if (x > 0) {out = 1} else {out = 2}`,
			wantUnchanged: []string{"simplifyConstantConditions"},
			oc:            only,
		},
		{
			// Truthy with no Else: nothing to prune, so the if-statement (and the scope layer its own header
			// contributes) is left completely untouched. See the correctness note on simplifyConstantConditions -
			// flattening this into a bare Block would still behave the same here, but would silently change scoping for
			// other programs (see compiler_test.go's redeclaration tests), so this pass never does it.
			name:          "init statement preserved (if kept intact, nothing to prune)",
			src:           `if x := 10; true {out = x}`,
			wantAST:       `if x := 10; true {out = x}`,
			wantUnchanged: []string{"simplifyConstantConditions"},
			wantOut:       10,
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// eliminateDeadBranches (chained if / else if)
// ---------------------------------------------------------------------------

func TestOptimizer_EliminateDeadBranches(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 3
		oc.EliminateDeadBranches = true
		return oc
	}
	cases := []optCase{
		{
			name:        "if-else-if constant true drops later branches",
			src:         `if true { out = 1 } else if x > 0 { out = 2 } else { out = 3 }`,
			wantAST:     `if true {out = 1}`,
			wantChanged: []string{"eliminateDeadBranches"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:        "if false else-if kept",
			src:         `x := 1; if false { out = 1 } else if x > 0 { out = 2 } else { out = 3 }`,
			wantAST:     `x := 1; if false {} else if (x > 0) {out = 2} else {out = 3}`,
			wantChanged: []string{"eliminateDeadBranches"},
			wantOut:     2,
			oc:          only,
		},
		{
			name:          "non-constant condition preserved",
			src:           `x := 1; if x > 0 { out = 1 } else if x > 5 { out = 2 } else { out = 3 }`,
			wantAST:       `x := 1; if (x > 0) {out = 1} else if (x > 5) {out = 2} else {out = 3}`,
			wantUnchanged: []string{"eliminateDeadBranches"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// eliminateUnreachableAfterTerminator
// ---------------------------------------------------------------------------

func TestOptimizer_EliminateUnreachableAfterTerminator(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.EliminateUnreachableAfterTerminator = true
		return oc
	}
	cases := []optCase{
		{
			name:        "return terminates block",
			src:         `f := func() { return 1; return 2 }; out = f()`,
			wantAST:     `f := func() {return 1}; out = f()`,
			wantChanged: []string{"eliminateUnreachableAfterTerminator"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:        "break inside loop terminates block",
			src:         `for i := 0; i < 10; i = i + 1 { break; out = i }; out = 7`,
			wantAST:     `for i := 0 ; (i < 10) ; i = (i + 1){break}; out = 7`,
			wantChanged: []string{"eliminateUnreachableAfterTerminator"},
			wantOut:     7,
			oc:          only,
		},
		{
			name:          "no terminator, no change",
			src:           `f := func() { x := 1; return x }; out = f()`,
			wantAST:       `f := func() {x := 1; return x}; out = f()`,
			wantUnchanged: []string{"eliminateUnreachableAfterTerminator"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// propagateConstants
// ---------------------------------------------------------------------------

func TestOptimizer_PropagateConstants(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.PropagateConstants = true
		return oc
	}
	cases := []optCase{
		{
			name:        "simple literal propagation",
			src:         `x := 5; out = x`,
			wantAST:     `x := 5; out = 5`,
			wantChanged: []string{"propagateConstants"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:        "propagation into expression",
			src:         `x := 5; y := 3; out = x + y`,
			wantAST:     `x := 5; y := 3; out = (5 + 3)`,
			wantChanged: []string{"propagateConstants"},
			wantOut:     8,
			oc:          only,
		},
		{
			name:          "variable reassigned: not propagated",
			src:           `x := 5; x = 6; out = x`,
			wantAST:       `x := 5; x = 6; out = x`,
			wantUnchanged: []string{"propagateConstants"},
			oc:            only,
		},
		{
			name:          "variable used in closure: not propagated",
			src:           `x := 5; f := func() { return x }; out = f()`,
			wantAST:       `x := 5; f := func() {return x}; out = f()`,
			wantUnchanged: []string{"propagateConstants"},
			oc:            only,
		},
		{
			name:          "non-literal RHS: not propagated",
			src:           `x := 1 + 2; out = x`,
			wantAST:       `x := (1 + 2); out = x`,
			wantUnchanged: []string{"propagateConstants"},
			oc:            only,
		},
		{
			name:          "variable used as LHS base: not propagated",
			src:           `a := "foo"; out = a`,
			wantAST:       `a := "foo"; out = "foo"`,
			wantChanged:   []string{"propagateConstants"},
			wantUnchanged: nil,
			wantOut:       "foo",
			oc:            only,
		},
		{
			name:          "builtin name not propagated",
			src:           `len = 99; out = len`,
			wantAST:       `len = 99; out = len`,
			wantUnchanged: []string{"propagateConstants"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// copyPropagation
// ---------------------------------------------------------------------------

func TestOptimizer_CopyPropagation(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.CopyPropagation = true
		return oc
	}
	cases := []optCase{
		{
			name:        "simple copy replaced",
			src:         `x := 5; y := x; out = y`,
			wantAST:     `x := 5; y := x; out = x`,
			wantChanged: []string{"copyPropagation"},
			wantOut:     5,
			oc:          only,
		},
		{
			name:          "copy source reassigned: not propagated",
			src:           `x := 5; y := x; x = 6; out = y`,
			wantAST:       `x := 5; y := x; x = 6; out = y`,
			wantUnchanged: []string{"copyPropagation"},
			oc:            only,
		},
		{
			name:          "copy captured in closure: not propagated",
			src:           `x := 5; y := x; f := func() { return y }; out = f()`,
			wantAST:       `x := 5; y := x; f := func() {return y}; out = f()`,
			wantUnchanged: []string{"copyPropagation"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// eliminateDeadAssignments
// ---------------------------------------------------------------------------

func TestOptimizer_EliminateDeadAssignments(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.EliminateDeadAssignments = true
		return oc
	}
	cases := []optCase{
		{
			name:        "literal unused ident removed",
			src:         `x := 5; out = 1`,
			wantAST:     `out = 1`,
			wantChanged: []string{"eliminateDeadAssignments"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:          "used ident retained",
			src:           `x := 5; out = x`,
			wantAST:       `x := 5; out = x`,
			wantUnchanged: []string{"eliminateDeadAssignments"},
			oc:            only,
		},
		{
			name:          "side-effecting RHS retained",
			src:           `x := len("hi"); out = 1`,
			wantAST:       `x := len("hi"); out = 1`,
			wantUnchanged: []string{"eliminateDeadAssignments"},
			oc:            only,
		},
		{
			name:          "captured by closure: retained",
			src:           `x := 5; f := func() { return x }; out = f()`,
			wantAST:       `x := 5; f := func() {return x}; out = f()`,
			wantUnchanged: []string{"eliminateDeadAssignments"},
			oc:            only,
		},
		{
			name:          "binding consumed by export is retained",
			src:           `res := 5; export res`,
			wantAST:       `res := 5; export res`,
			wantUnchanged: []string{"eliminateDeadAssignments"},
			oc:            only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// Combined pipelines (O2 / O3) — smoke tests
// ---------------------------------------------------------------------------

func TestOptimizer_O2_Pipeline(t *testing.T) {
	cases := []optCase{
		{
			name:    "constant folded through propagation and dead-assign",
			src:     `x := 2 * 3; y := x + 4; out = y * 2`,
			wantOut: 20,
			oc:      compiler.O2,
		},
		{
			name:    "constant-false branch pruned",
			src:     `if false { out = 1 } else { out = 42 }`,
			wantOut: 42,
			oc:      compiler.O2,
		},
		{
			name:    "if-to-bool short-form",
			src:     `x := 3; y := 2; if x > y { out = true } else { out = false }`,
			wantOut: true,
			oc:      compiler.O2,
		},
	}
	runOptCases(t, cases)
}

func TestOptimizer_O3_Pipeline_NoOp_Passes_AreSafe(t *testing.T) {
	// foldPureFunctionCalls / inlinePureFunctions are safe no-ops today.
	// The test below verifies they don't corrupt otherwise-valid programs.
	cases := []optCase{
		{
			name:    "user pure function preserved",
			src:     `f := func(x) { return x + 1 }; out = f(41)`,
			wantOut: 42,
			oc:      compiler.O3,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// FoldConstantSubexpressions gating and ordering (round-2 safety review finding #2 fix).
//
// FoldConstantSubexpressions is the only pass that speculatively compiles+runs candidate subtrees in a real VM, so
// its cost isn't bounded by AST size the way every other pass's is (OPTIMIZER_REVIEW.md finding #2: a 200M-char
// `.repeat()` inside a provably-dead `if false { ... }` branch took ~1.4s to compile at O2 before this fix, vs.
// ~136µs at O0 — for code that never executes). Two changes close this: (a) FoldConstantSubexpressions is now O3
// only, not O1/O2, so a caller can get all of O2's dead-code/branch elimination and constant propagation without
// paying for speculative VM execution; (b) within a cycle, it now runs LAST, after every pass that can shrink or
// eliminate code, so a branch that's already dead by a literal condition (no folding needed to see that) is removed
// before the expensive pass ever reaches it in the same cycle.
// ---------------------------------------------------------------------------

func TestOptimizer_FoldConstantSubexpressions_O3Only(t *testing.T) {
	require.False(t, compiler.O1().FoldConstantSubexpressions, "O1 must not enable speculative constant folding")
	require.False(t, compiler.O2().FoldConstantSubexpressions, "O2 must not enable speculative constant folding")
	require.True(t, compiler.O3().FoldConstantSubexpressions, "O3 must enable speculative constant folding")
}

func TestOptimizer_FoldConstantSubexpressions_RunsAfterDeadCodeElimination(t *testing.T) {
	cases := []optCase{
		{
			// The only foldable subexpression in this program lives inside a branch whose condition is already a
			// literal `false` — eliminateDeadBranches/simplifyConstantConditions can and must prune it without any
			// folding. foldConstantSubexpressions must report unchanged: by the time it runs (last), the dead body
			// is already gone, so it never touches (and never speculatively evaluates) the expression that was there.
			name:          "expression inside an already-dead if-false branch is never folded",
			src:           `out = 0; if false { out = 1 + 2 }`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			wantChanged:   []string{"simplifyConstantConditions"},
			wantOut:       0,
			oc:            compiler.O3,
		},
		{
			// Same shape for code after a terminating return: eliminateUnreachableAfterTerminator removes it purely
			// structurally (no folding needed to know it's unreachable), so foldConstantSubexpressions never sees it.
			name:          "expression after a return is never folded",
			src:           `f := func() { return 1; x := 1 + 2 }; out = f()`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			wantChanged:   []string{"eliminateUnreachableAfterTerminator"},
			wantOut:       1,
			oc:            compiler.O3,
		},
		{
			// Sanity: ordinary, live constant folding still works at O3 once dead-code passes have nothing left to
			// prune first.
			name:        "live constant expression still folds",
			src:         `out = 1 + 2`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     3,
			oc:          compiler.O3,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// Dynamic-typing / shadowing corner cases (round-2 safety review regressions).
//
// collectNameUsage (optimizer_impl.go) tracks name usage in one flat, file-wide map keyed by identifier string, with
// no real lexical-scope awareness. propagateConstants/copyPropagation/eliminateDeadAssignments only collect
// substitution *candidates* from top-level file.Stmts, but their rewrite step walks the ENTIRE tree, including
// nested blocks and FuncLit bodies. These cases pin down that shadowing a candidate name in a nested block, a FuncLit
// parameter, or a for-in loop variable never leaks the outer literal into the shadowed scope: every such shadowing
// construct also counts as a "write" (or sets insideFuncLit) in the same flat usage record, which conservatively
// disqualifies the outer name from substitution. Verified empirically at O0 vs O3 before writing these down.
// ---------------------------------------------------------------------------

func TestOptimizer_ShadowingCornerCases(t *testing.T) {
	cases := []optCase{
		{
			name: "block-scope shadow of outer var not clobbered by propagation",
			src: `x := 5
if true {
	x := 100
	out = x
}`,
			wantOut: 100,
			oc:      compiler.O3,
		},
		{
			name: "nested multi-level block shadow resolves to innermost",
			src: `x := 1
if true {
	x := 2
	if true {
		x := 3
		out = x
	}
}`,
			wantOut: 3,
			oc:      compiler.O3,
		},
		{
			name: "function parameter shadows outer constant candidate",
			src: `x := 5
f := func(x) { return x * 2 }
out = f(10) + x`,
			wantOut: 25,
			oc:      compiler.O3,
		},
		{
			name: "for-in loop variable shadows same-named outer var",
			src: `x := 99
sum := 0
for x in [1, 2, 3] {
	sum = sum + x
}
out = sum + x`,
			wantOut: 105,
			oc:      compiler.O3,
		},
		{
			name: "copy source reassigned after copy point keeps pre-reassignment value",
			src: `x := 5
y := x
x = 10
out = y`,
			wantOut: 5,
			oc:      compiler.O3,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// Dynamic-typing coercion / truthiness / precision corner cases.
//
// foldConstantSubexpressions never hand-rolls operator semantics: it speculatively compiles+runs the candidate
// subtree in the real VM (evalConstantExpr) and only materializes a literal via safeValueToLiteral on success. That
// design should make it correct-by-construction for coercion/truthiness/precision rules — these cases exist to pin
// that down for the trickiest parts of Kavun's dynamic type system (asymmetric string+int coercion, decimal
// precision, float-vs-int/decimal truthiness divergence, NaN, integer overflow) so a future change to the folder
// can't quietly regress them.
// ---------------------------------------------------------------------------

func TestOptimizer_DynamicTypingCornerCases(t *testing.T) {
	cases := []optCase{
		{
			name:    "decimal precision preserved through propagation",
			src:     `x := 1.50d; out = x`,
			wantOut: dec128.FromString("1.50"),
			oc:      compiler.O3,
		},
		{
			name:    "decimal precision preserved through arithmetic folding",
			src:     `out = 1.10d + 2.00d`,
			wantOut: dec128.FromString("3.10"),
			oc:      compiler.O3,
		},
		{
			// docs/language.md truthiness table: int 0 / decimal 0 are falsy, but float 0.0 is truthy.
			name:    "float zero is truthy (unlike int/decimal zero)",
			src:     `if 0.0 { out = "truthy" } else { out = "falsy" }`,
			wantOut: "truthy",
			oc:      compiler.O3,
		},
		{
			name:    "decimal zero is falsy",
			src:     `if 0.00d { out = "truthy" } else { out = "falsy" }`,
			wantOut: "falsy",
			oc:      compiler.O3,
		},
		{
			name:    "string + int coercion folds",
			src:     `out = "a" + 1`,
			wantOut: "a1",
			oc:      compiler.O3,
		},
		{
			name:          "int + string is a runtime error, left unfolded",
			src:           `out = 1 + "a"`,
			wantUnchanged: []string{"foldConstantSubexpressions"},
			oc: func() *compiler.OptimizationConfig {
				oc := compiler.O0()
				oc.MaxPasses = 3
				oc.FoldConstantSubexpressions = true
				return oc
			},
		},
		{
			name:    "int overflow wraps identically whether folded or not",
			src:     `out = 9223372036854775807 + 1`,
			wantOut: int64(-9223372036854775808),
			oc:      compiler.O3,
		},
		{
			name:    "ternary does not evaluate the untaken side effect/error branch",
			src:     `out = true ? 1 : (1 / 0)`,
			wantOut: 1,
			oc:      compiler.O3,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// MethodCall purity gate (round-2 safety review finding #1 fix).
//
// isFoldableExpr's *expression.MethodCall case used to have no purity check on the method name at all — unlike its
// *expression.Call case, which gates on isBuiltinPureName. It relied entirely on (a) container-literal receivers
// having no matching AST case and (b) safeValueToLiteral refusing non-scalar return types, neither of which catches
// a method that returns a scalar value that is nonetheless environment-dependent. core.ValueTypeDescr.IsMethodPure
// closes that gap: isFoldableExpr now requires the receiver to already be a literal (so its concrete type is known
// without evaluating anything — true immediately for a literal written in source, and true after this same
// bottom-up pass has already folded an eligible receiver into one) and consults that type's IsMethodPure(name).
// ---------------------------------------------------------------------------

func TestOptimizer_MethodCallPurityGate(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.FoldConstantSubexpressions = true
		return oc
	}
	cases := []optCase{
		{
			name:        "ordinary string method still folds",
			src:         `out = "abc".upper()`,
			wantAST:     `out = "ABC"`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     "ABC",
			oc:          only,
		},
		{
			// core/int.go's AsTime now normalizes to UTC (time.Unix(...).UTC()), so this chain has no environment
			// dependence left and is safe to fold: 1718445600 == 2024-06-15T10:00:00Z.
			name:        "int-to-time wall-clock accessor folds (UTC by default, no environment dependence)",
			src:         `out = (1718445600).time().hour()`,
			wantAST:     `out = 10`,
			wantChanged: []string{"foldConstantSubexpressions"},
			wantOut:     10,
			oc:          only,
		},
		{
			// The receiver's OWN .time() call is independently foldable (String's IsMethodPure allows "time", and
			// dateparse.ParseAny defaults to UTC), so the pass does report changed=true — but .local() itself must
			// never fold (see TestOptimizer_TimeLocalFoldingDoesNotLeakCompileTimeZone for the behavioral proof), and
			// once .local() fails to fold, .hour() sees a non-literal receiver and is transitively blocked too.
			name:    "time.local() and anything chained after it are never folded",
			src:     `out = "2024-06-15T10:00:00Z".time().local().hour()`,
			wantAST: `out = t"2024-06-15T10:00:00Z".local().hour()`,
			oc:      only,
		},
	}
	runOptCases(t, cases)
}

// TestOptimizer_TimeLocalFoldingDoesNotLeakCompileTimeZone is the behavioral regression test for round-2 finding #1:
// before the IsMethodPure gate, `some_time.local().hour()` (or any wall-clock read reached through .local()) would
// be evaluated once at compile time and frozen into the bytecode using whatever timezone the COMPILING process
// happened to be in, diverging from the correct behavior of re-evaluating "local time" against the RUNNING process's
// zone on every execution. Proven by compiling while time.Local is UTC, then switching time.Local to a different
// zone before running the already-compiled bytecode (simulating "compiled here, executed there," e.g. a compile
// farm vs. execution nodes in a different region, or a cached Compiled reused across a DST transition): a correct
// implementation must reflect the zone active at RUN time for both O0 and O3.
func TestOptimizer_TimeLocalFoldingDoesNotLeakCompileTimeZone(t *testing.T) {
	originalLocal := time.Local
	defer func() { time.Local = originalLocal }()

	utc, err := time.LoadLocation("UTC")
	require.NoError(t, err, "load UTC")
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err, "load Asia/Tokyo")

	src := `out = "2024-06-15T10:00:00Z".time().local().hour()`

	for _, oc := range []*compiler.OptimizationConfig{compiler.O0(), compiler.O3()} {
		time.Local = utc

		fileSet := ast.NewFileSet()
		srcFile := fileSet.AddFile("tz-test", -1, len(src))
		symTable := compiler.NewSymbolTable()
		for idx, name := range vm.BuiltinFunctionNames {
			symTable.DefineBuiltin(idx, name)
		}
		outSym := symTable.Define("out")
		c := compiler.NewCompiler(oc, nil, srcFile, symTable, nil, nil, nil)
		require.NoError(t, c.Compile(srcFile, []byte(src), nil), "compile")
		bc := c.Bytecode()

		time.Local = tokyo // switch AFTER compiling, BEFORE running

		globals := make([]core.Value, vm.GlobalsSize)
		machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
		machine.Reset(bc, globals)
		require.NoError(t, machine.Run(), "run")

		got := globals[outSym.Index]
		want := core.IntValue(19) // 10:00 UTC == 19:00 Asia/Tokyo (UTC+9) — must reflect the RUN-time zone
		require.True(t, got.Equal(want), "hour mismatch: got %s want %s", got.String(), want.String())
	}
}

// ---------------------------------------------------------------------------
// Ensure O0 disables everything
// ---------------------------------------------------------------------------

func TestOptimizer_O0_NoChanges(t *testing.T) {
	// With O0, MaxPasses = 0 → Optimize returns node unchanged. Verify the
	// AST matches the raw parse output.
	src := `out = 1 + 2`
	file := parseFile(t, src)
	c := compiler.NewCompiler(compiler.O0(), nil, file.InputFile, nil, nil, nil, nil)
	optimized, err := c.Optimize(file)
	require.NoError(t, err, "optimize")
	require.Equal(t, normalizeStr(file.String()), normalizeStr(optimized.String()))
}
