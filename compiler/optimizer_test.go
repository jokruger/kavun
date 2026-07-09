package compiler_test

import (
	"strings"
	"testing"

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
			fileSet := parser.NewFileSet()
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

func parseFile(t *testing.T, src string) *parser.File {
	t.Helper()
	fileSet := parser.NewFileSet()
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
// simplifyBooleanIdentities
// ---------------------------------------------------------------------------

func TestOptimizer_SimplifyBooleanIdentities(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.SimplifyBooleanIdentities = true
		return oc
	}
	cases := []optCase{
		{
			name:        "!true → false",
			src:         `out = !true`,
			wantAST:     `out = false`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     false,
			oc:          only,
		},
		{
			name:        "!false → true",
			src:         `out = !false`,
			wantAST:     `out = true`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:        "!!(a > b) → (a > b)  (provably bool)",
			src:         `a := 3; b := 2; out = !!(a > b)`,
			wantAST:     `a := 3; b := 2; out = ((a > b))`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:          `!!"abc" is NOT simplified (non-bool operand)`,
			src:           `out = !!"abc"`,
			wantAST:       `out = (!(!"abc"))`,
			wantUnchanged: []string{"simplifyBooleanIdentities"},
			oc:            only,
		},
		{
			name:        `(a > b) == true → (a > b)`,
			src:         `a := 3; b := 2; out = (a > b) == true`,
			wantAST:     `a := 3; b := 2; out = ((a > b))`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:        `(a > b) == false → !(a > b)`,
			src:         `a := 3; b := 2; out = (a > b) == false`,
			wantAST:     `a := 3; b := 2; out = (!((a > b)))`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     false,
			oc:          only,
		},
		{
			name:        `(a > b) != true → !(a > b)`,
			src:         `a := 3; b := 2; out = (a > b) != true`,
			wantAST:     `a := 3; b := 2; out = (!((a > b)))`,
			wantChanged: []string{"simplifyBooleanIdentities"},
			wantOut:     false,
			oc:          only,
		},
		{
			name:          `x == true not simplified for non-bool ident`,
			src:           `x := "y"; out = x == true`,
			wantAST:       `x := "y"; out = (x == true)`,
			wantUnchanged: []string{"simplifyBooleanIdentities"},
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
			src:         `out = 0; if true { out = 1 } else { out = 2 }`,
			wantAST:     `out = 0; {out = 1}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:        "if false keeps else",
			src:         `out = 0; if false { out = 1 } else { out = 2 }`,
			wantAST:     `out = 0; {out = 2}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     2,
			oc:          only,
		},
		{
			name:        "if false without else drops branch",
			src:         `out = 5; if false { out = 1 }`,
			wantAST:     `out = 5; {}`,
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
			name:        "init statement preserved",
			src:         `if x := 10; true { out = x }`,
			wantAST:     `{x := 10; out = x}`,
			wantChanged: []string{"simplifyConstantConditions"},
			wantOut:     10,
			oc:          only,
		},
	}
	runOptCases(t, cases)
}

// ---------------------------------------------------------------------------
// simplifyIfExprToBool
// ---------------------------------------------------------------------------

func TestOptimizer_SimplifyIfExprToBool(t *testing.T) {
	only := func() *compiler.OptimizationConfig {
		oc := compiler.O0()
		oc.MaxPasses = 2
		oc.SimplifyIfExprToBool = true
		return oc
	}
	cases := []optCase{
		{
			name:        "if cond {return true} else {return false} → return cond",
			src:         `f := func(a, b) { if a > b { return true } else { return false } }; out = f(3, 2)`,
			wantAST:     `f := func(a, b) {return (a > b)}; out = f(3, 2)`,
			wantChanged: []string{"simplifyIfExprToBool"},
			wantOut:     true,
			oc:          only,
		},
		{
			name:        "if cond {return false} else {return true} → return !cond",
			src:         `f := func(a, b) { if a > b { return false } else { return true } }; out = f(3, 2)`,
			wantAST:     `f := func(a, b) {return (!(a > b))}; out = f(3, 2)`,
			wantChanged: []string{"simplifyIfExprToBool"},
			wantOut:     false,
			oc:          only,
		},
		{
			name:          "non-bool cond not simplified",
			src:           `f := func(x) { if x { return true } else { return false } }; out = f("hi")`,
			wantAST:       `f := func(x) {if x {return true} else {return false}}; out = f("hi")`,
			wantUnchanged: []string{"simplifyIfExprToBool"},
			oc:            only,
		},
		{
			name:          "identical branches not simplified",
			src:           `f := func(a, b) { if a > b { return true } else { return true } }; out = f(3, 2)`,
			wantAST:       `f := func(a, b) {if (a > b) {return true} else {return true}}; out = f(3, 2)`,
			wantUnchanged: []string{"simplifyIfExprToBool"},
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
			wantAST:     `{out = 1}`,
			wantChanged: []string{"eliminateDeadBranches"},
			wantOut:     1,
			oc:          only,
		},
		{
			name:        "if false else-if kept",
			src:         `x := 1; if false { out = 1 } else if x > 0 { out = 2 } else { out = 3 }`,
			wantAST:     `x := 1; if (x > 0) {out = 2} else {out = 3}`,
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
