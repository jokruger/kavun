package compiler_test

import (
	"strings"
	"testing"

	"github.com/jokruger/kavun/compiler"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/internal/require"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/vm"
)

// Compiles a Kavun source snippet and returns the resulting bytecode. The compiler runs with builtins (so `len`,
// `range`, etc. resolve) but no host symbols.
func compileSrc(t *testing.T, src string) *vm.Bytecode {
	t.Helper()
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("test", -1, len(src))

	p := parser.NewParser(srcFile, []byte(src), nil)
	file, err := p.ParseFile()
	require.NoError(t, err, "parse error for src: %s", src)

	c := compiler.NewCompiler(nil, srcFile, nil, nil, nil, nil)
	err = c.Compile(file)
	require.NoError(t, err, "compile error for src: %s", src)
	return c.Bytecode()
}

// Returns the main function followed by every nested compiled function reachable from the constants table, in encounter
// order. This lets tests assert MaxStack of individual lambdas without depending on the exact constant index assigned
// by the compiler.
func collectFuncs(bc *vm.Bytecode) []*core.CompiledFunction {
	out := []*core.CompiledFunction{bc.MainFunction}
	for i := range bc.Static.CompiledFunctions {
		out = append(out, &bc.Static.CompiledFunctions[i])
	}
	return out
}

// Compiles and runs a snippet via the standard VM. It panics-recovers and returns the run error (if any). Used to
// confirm that MaxStack is at least sufficient — if it were undersized, the OpCall stack-bounds guard would reject
// calls and the script would fail.
func runOK(t *testing.T, src string) {
	t.Helper()
	bc := compileSrc(t, src)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	globals := make([]core.Value, vm.GlobalsSize)
	machine.Reset(bc, globals)
	err := machine.Run()
	require.NoError(t, err, "run error for src: %s", src)
}

// Produces a short identifier from the first non-empty line of src.
func scriptName(src string, idx int) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 40 {
			line = line[:40]
		}
		return line
	}
	return "script-" + string(rune('0'+idx))
}

// Static cases — small hand-built bytecode snippets with known stack heights.
func TestComputeMaxStack_Static(t *testing.T) {
	cases := []struct {
		name string
		ins  []byte
		want int
	}{
		{
			"empty",
			[]byte{},
			0,
		},
		{
			"single constant push",
			[]byte{byte(opcode.LoadStaticString8), 0, 0},
			1,
		},
		{
			"push and pop balances to zero peak of 1",
			[]byte{
				byte(opcode.LoadStaticString8), 0, 0,
				byte(opcode.Pop),
			},
			1,
		},
		{
			"three pushes then pop reaches peak 3",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticString8), 0, 1,
				byte(opcode.LoadStaticRunes8), 0, 2,
				byte(opcode.Pop),
				byte(opcode.Pop),
				byte(opcode.Pop),
			},
			3,
		},
		{
			"binary op: a+b peaks at 2",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.BinaryOp), 1,
			},
			2,
		},
		{
			"array of 4 elements peaks at 4",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.LoadStaticPrimitive8), 0, 2,
				byte(opcode.LoadStaticPrimitive8), 0, 3,
				byte(opcode.MakeArray8), 4, 0,
			},
			4,
		},
		{
			"call with 3 args peaks at 4 (callee + 3 args)",
			[]byte{
				byte(opcode.LoadGlobal8), 0, 0, // callee
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.LoadStaticPrimitive8), 0, 2,
				byte(opcode.CallFunction), 3, 0,
			},
			4,
		},
		{
			"short-circuit AND balances",
			// Push a, AndJump END, push b, END: result on stack -> peak 1
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0, // push a
				byte(opcode.AndJump), 9, 0, // jump to END if false
				byte(opcode.LoadStaticPrimitive8), 0, 1, // push b (fall-through)
				// END: result is one value
			},
			1,
		},
		{
			"if/else both arms balance",
			// 0: push cond           (3 bytes)
			// 3: JumpFalsy -> 16     (5 bytes)
			// 8: push then           (3 bytes)
			// 11: Jump -> 19         (5 bytes)
			// 16: push else          (3 bytes)
			// 19: <end>
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0, // cond
				byte(opcode.JumpFalsy), 16, 0, // -> ELSE
				byte(opcode.LoadStaticPrimitive8), 0, 1, // then
				byte(opcode.Jump8), 19, 0, // -> END
				byte(opcode.LoadStaticPrimitive8), 0, 2, // else
				// END
			},
			1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compiler.ComputeMaxStack(tc.ins)
			require.Equal(t, tc.want, got)
		})
	}
}

// Ensures that analyzeOp panics on an unknown opcode. This is a guard against forgetting to extend the analyzer
// when a new opcode is introduced.
func TestComputeMaxStack_UnknownOpcodePanics(t *testing.T) {
	// 0xFF is well outside the range of currently defined opcodes.
	ins := []byte{0xFF}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic on unknown opcode, got nil")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "unknown opcode") {
			t.Fatalf("expected panic message to mention 'unknown opcode', got %q", msg)
		}
	}()
	_ = compiler.ComputeMaxStack(ins)
}

// Asserts exact MaxStack values for hand-traced sources. The expected `main` is MaxStack of the top-level function
// (which always ends with OpSuspend). The `inner` slice lists expected MaxStacks for nested compiled functions in the
// order they appear in the constants table.
func TestComputeMaxStack_Compile_Exact(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		main  int
		inner []int
	}{
		// --- Trivial bodies ---
		{
			"empty top-level",
			``,
			0,
			nil,
		},
		{
			"single literal expression",
			`1`,
			1,
			nil,
		},
		{
			"simple add",
			`a := 1 + 2`,
			2,
			nil,
		},

		// --- Lambdas ---
		{
			"lambda no params, returns literal",
			`f := func() { return 1 }`,
			1, // OpConstant<f>, OpSetGlobal
			[]int{1},
		},
		{
			"lambda two params, returns sum",
			`f := func(a, b) { return a + b }`,
			1,
			// inner: GetLocal a, GetLocal b, OpBinaryOp, OpReturn 1 -> peak 2
			[]int{2},
		},
		{
			"lambda with 4-arg nested call",
			// inner peak: callee + 4 args = 5
			`g := func(a,b,c,d) { return a }
			 f := func() { return g(1, 2, 3, 4) }`,
			1,
			[]int{1, 5}, // g first (peak 1: just GetLocal+Return), then f
		},

		// --- Closures ---
		{
			"closure captures 1 local",
			`outer := func() {
				x := 1
				return func() { return x }
			}`,
			1,
			// inner-most: GetFree 0, OpReturn 1 -> peak 1
			// outer: push 1 / DefLocal, GetLocalPtr(x), OpClosure(...,1), OpReturn 1 -> peak 1
			[]int{1, 1},
		},
		{
			"closure captures 3 locals",
			`outer := func() {
				a := 1; b := 2; c := 3
				return func() { return a + b + c }
			}`,
			1,
			// inner-most: GetFree 0, GetFree 1, +, GetFree 2, +, OpReturn 1 -> peak 2
			// outer: pushing 3 free-var pointers before OpClosure -> peak 3
			[]int{2, 3},
		},

		// --- Method calls ---
		{
			"method call with 3 args",
			`a := [1].slice(0, 1, 1)`,
			// receiver + 3 args = 4
			4,
			nil,
		},

		// --- Array & record literals ---
		{
			"array of 5",
			`a := [1, 2, 3, 4, 5]`,
			5,
			nil,
		},
		{
			"record of 3 keys",
			// for each key,value pair: push key (const), compile value (1)
			// pairs accumulate; for 3 pairs: 6 slots peak
			`r := {a: 1, b: 2, c: 3}`,
			6,
			nil,
		},

		// --- Nested call chains ---
		{
			"nested call: f(g(1,2), h(3,4))",
			// outer callee + (inner result) + (inner callee + 2 args = 3) — peak observed = 5
			`f := func(p,q) { return p }
			 g := func(p,q) { return p }
			 h := func(p,q) { return p }
			 x := f(g(1, 2), h(3, 4))`,
			5,
			[]int{1, 1, 1},
		},

		// --- Short-circuit boolean chains ---
		{
			"and chain a && b && c",
			// AndJump leaves value on stack on jump-taken arm; fall-through pops it
			// before RHS push. Net peak = 1 at any single point.
			`a := true; b := true; c := true
			 x := a && b && c`,
			1,
			nil,
		},
		{
			"or chain a || b || c",
			`a := false; b := false; c := true
			 x := a || b || c`,
			1,
			nil,
		},

		// --- Conditional / ternary ---
		{
			"ternary a ? b : c",
			// both arms produce exactly 1 value on stack
			`a := true; b := 1; c := 2
			 x := a ? b : c`,
			1,
			nil,
		},

		// --- if/else statements ---
		{
			"if/else assigning result",
			// Both arms: 1 push + SetGlobal/Pop. Peak 1.
			`a := true
			 if a { x := 1 } else { x := 2 }`,
			1,
			nil,
		},
		{
			"if without else",
			`a := true
			 if a { x := 1 }`,
			1,
			nil,
		},

		// --- Selectors / index chains ---
		{
			"selector chain a.b.c.d",
			// pushes accumulate as we walk the chain: peak 4
			`a := {b: {c: {d: 1}}}
			 x := a.b.c.d`,
			4,
			nil,
		},
		{
			"index chain a[1][2][3]",
			`a := [[[1,2,3,4]]]
			 x := a[0][0][0]`,
			4,
			nil,
		},

		// --- Defer ---
		{
			"defer call with 2 args",
			// inner f body: callee + 2 args = 3
			`g := func(p,q) { return p }
			 f := func() { defer g(1, 2) }`,
			1,
			[]int{1, 3},
		},
		{
			"defer method call with 2 args",
			// inner f body: receiver + 2 args = 3
			`a := [1,2,3]
			 f := func() { defer a.slice(1, 2) }`,
			3, // [1,2,3] literal peaks at 3 in the main function
			[]int{3},
		},

		// --- Slicing ---
		{
			"slice a[1:2:3]",
			// a, low, high, step on stack = 4
			`a := [1,2,3,4,5]
			 x := a[1:2:3]`,
			5,
			nil,
		},
		{
			"slice a[1:2]",
			// a, low, high = 3
			`a := [1,2,3,4,5]
			 x := a[1:2]`,
			5,
			nil,
		},

		// --- Loops ---
		{
			"plain for loop",
			// for i := 0; i < 3; i = i + 1 { x := i }
			// Peak inside body: at most 2 (i + literal for comparison or addition)
			`for i := 0; i < 3; i = i + 1 { x := i }`,
			2,
			nil,
		},
		{
			"for-in over array",
			`for k, v in [1, 2, 3] { x := k + v }`,
			3, // [1,2,3] literal pushes 3 slots before OpArray
			nil,
		},

		// --- f-strings ---
		{
			"f-string empty",
			`x := f""`,
			1,
			nil,
		},
		{
			"f-string single literal",
			`x := f"hello"`,
			1,
			nil,
		},
		{
			"f-string with one interpolation",
			`a := 1
			 x := f"val={a}"`,
			// "val=" (1) ; push a, OpFormat (1) -> total 2 before Add
			2,
			nil,
		},
		{
			"f-string with three interpolations",
			// Each iteration after the first: push part, Add (peak 2). So peak stays 2.
			`a := 1; b := 2; c := 3
			 x := f"a={a} b={b} c={c}"`,
			2,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bc := compileSrc(t, tc.src)
			funcs := collectFuncs(bc)
			require.Equal(t, tc.main, funcs[0].MaxStack, "main MaxStack mismatch; got %d want %d for src:\n%s", funcs[0].MaxStack, tc.main, tc.src)

			gotInner := funcs[1:]
			require.Equal(t, len(tc.inner), len(gotInner), "inner function count mismatch (got %d, want %d) for src:\n%s", len(gotInner), len(tc.inner), tc.src)
			for i, want := range tc.inner {
				require.Equal(t, want, gotInner[i].MaxStack, "inner[%d] MaxStack mismatch; got %d want %d for src:\n%s", i, gotInner[i].MaxStack, want, tc.src)
			}
		})
	}
}

// Verifies that scripts using closures, nested calls, recursion, deferred calls, etc. compile *and* run cleanly.
// If MaxStack were undersized for any inner function, the VM's OpCall stack-bound check would trip and Run() would
// return an error.
func TestComputeMaxStack_Compile_RunOK(t *testing.T) {
	scripts := []string{
		// Closures: capture, mutate, return through a free var
		`make := func() {
			x := 0
			return func() { x = x + 1; return x }
		}
		c := make()
		a := c(); b := c(); cc := c()
		if a != 1 || b != 2 || cc != 3 { raise("fail") }`,

		// Nested closures (3 levels)
		`f := func() {
			x := 10
			return func() {
				y := 20
				return func() { return x + y }
			}
		}
		v := f()()()
		if v != 30 { raise("fail") }`,

		// Closure with many captured locals
		`f := func() {
			a := 1; b := 2; c := 3; d := 4; e := 5; g := 6; h := 7
			return func() { return a + b + c + d + e + g + h }
		}
		if f()() != 28 { raise("fail") }`,

		// Recursive lambda via DEFL-before-CLOSURE trick (named result optional)
		`fact := func(n) {
			if n <= 1 { return 1 }
			return n * fact(n - 1)
		}
		if fact(6) != 720 { raise("fail") }`,

		// Mutual recursion via forward declarations.
		// (Mutually-recursive let-bindings need an explicit `undefined` seed
		// for the second name so the first can capture it as a free var.)
		`isOdd := undefined
		isEven := func(n) { if n == 0 { return true } else { return isOdd(n - 1) } }
		isOdd = func(n) { if n == 0 { return false } else { return isEven(n - 1) } }
		if !isEven(10) || isOdd(10) { raise("fail") }`,

		// Deeply nested call expressions
		`f := func(a, b, c) { return a + b + c }
		v := f(f(1, 2, 3), f(4, 5, 6), f(7, 8, 9))
		if v != 45 { raise("fail") }`,

		// Method calls + chained selectors
		`s := "hello world"
		if !s.contains("world") { raise("fail") }`,

		// Defer + recover with multi-arg deferred call
		`f := func() {
			defer func(a, b, c) { }(1, 2, 3)
			return 42
		}
		if f() != 42 { raise("fail") }`,

		// Long f-string concatenation
		`a := 1; b := 2; c := 3; d := 4; e := 5
		s := f"{a}-{b}-{c}-{d}-{e}"
		if s != "1-2-3-4-5" { raise("fail") }`,

		// Big array literal
		`arr := [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]
		if len(arr) != 20 { raise("fail") }`,

		// Big record literal
		`r := {a:1, b:2, c:3, d:4, e:5, f:6, g:7, h:8}
		if r.a + r.h != 9 { raise("fail") }`,

		// Short-circuit operator chains
		`f := func() {
			a := true; b := false
			x := a && a && a && a && a
			y := b || b || b || a
			return x && y
		}
		if !f() { raise("fail") }`,

		// Spread arguments
		`f := func(a, b, c) { return a + b + c }
		args := [10, 20, 30]
		if f(args...) != 60 { raise("fail") }`,

		// Variadic function
		`f := func(...args) { s := 0; for _, v in args { s = s + v }; return s }
		if f(1, 2, 3, 4, 5) != 15 { raise("fail") }`,

		// Named result and bare return
		`f := func() x { x = 7; return }
		if f() != 7 { raise("fail") }`,

		// For-in with break/continue
		`total := 0
		for _, v in [1, 2, 3, 4, 5] {
			if v == 3 { continue }
			if v == 5 { break }
			total = total + v
		}
		if total != 1 + 2 + 4 { raise("fail") }`,

		// Nested for + break
		`hits := 0
		for i := 0; i < 5; i = i + 1 {
			for j := 0; j < 5; j = j + 1 {
				if j == 2 { break }
				hits = hits + 1
			}
		}
		if hits != 10 { raise("fail") }`,

		// Deep recursion (must not blow MaxStack — loops/recursion don't grow per-frame stack)
		`sum := func(n) {
			if n <= 0 { return 0 }
			return n + sum(n - 1)
		}
		if sum(50) != 1275 { raise("fail") }`,

		// Selector assignment with chain
		`r := {a: {b: {c: 0}}}
		r.a.b.c = 42
		if r.a.b.c != 42 { raise("fail") }`,

		// Compound assign to selector
		`r := {n: 10}
		r.n += 5
		if r.n != 15 { raise("fail") }`,

		// Closure inside a loop (creates fresh closure each iteration)
		`fns := []
		for i := 0; i < 3; i = i + 1 {
			fns = append(fns, func() { return i })
		}
		if len(fns) != 3 { raise("fail") }`,

		// Dynamic-spec f-string
		`w := 5
		s := f"{42:0{w}d}"
		if s != "00042" { raise("fail") }`,

		// Recover with non-trivial body
		`safe := func() x {
			defer func() {
				if e := recover(); e != undefined {
					x = -1
				}
			}()
			a := 1
			b := 0
			x = a / b
			return
		}
		if safe() != -1 { raise("fail") }`,
	}

	for i, src := range scripts {
		t.Run(scriptName(src, i), func(t *testing.T) {
			runOK(t, src)
		})
	}
}

// Adds coverage for opcodes not exercised by the original static cases — defer, method calls, closures with frees,
// selector assignments, and big array literals.
func TestComputeMaxStack_StaticExtended(t *testing.T) {
	cases := []struct {
		name string
		ins  []byte
		want int
	}{
		{
			// receiver + 2 args, then OpMethodCall pops them all and pushes 1
			"method call receiver+2 args -> peak 3",
			[]byte{
				byte(opcode.LoadGlobal8), 0, 0, // receiver
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.CallMethod8), 0, 0, 2, 0, // methodIdx, nargs=2, ellipsis=0
			},
			3,
		},
		{
			// defer fn(a, b): push fn, a, b; OpDefer pops all 3
			"defer with 2 args -> peak 3",
			[]byte{
				byte(opcode.LoadGlobal8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.Defer), 2,
			},
			3,
		},
		{
			// defer obj.m(a, b): push receiver, a, b; OpDeferMethod pops 3
			"defer method with 2 args -> peak 3",
			[]byte{
				byte(opcode.LoadGlobal8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
				byte(opcode.DeferMethod8), 0, 0, 2,
			},
			3,
		},
		{
			// OpClosure NF=3: 3 free-var pointers must be on stack before
			"closure with 3 free vars -> peak 3",
			[]byte{
				byte(opcode.LoadLocalPtr), 0,
				byte(opcode.LoadLocalPtr), 1,
				byte(opcode.LoadLocalPtr), 2,
				byte(opcode.MakeClosure8), 0, 0, 3,
			},
			3,
		},
		{
			// OpSetSelGlobal NS=2: value + 2 selectors on stack -> peak 3
			"selector set global with 2 selectors -> peak 3",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0, // value
				byte(opcode.LoadStaticPrimitive8), 0, 1, // sel1
				byte(opcode.LoadStaticPrimitive8), 0, 2, // sel2
				byte(opcode.StoreIndexedGlobal8), 0, 0, 2,
			},
			3,
		},
		{
			// 8-element array
			"array of 8 -> peak 8",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.MakeArray8), 8, 0,
			},
			8,
		},
		{
			// SliceIndexStep pops 4 (target+lo+hi+step), pushes 1
			"slice with step -> peak 4",
			[]byte{
				byte(opcode.LoadGlobal8), 0, 0, // target
				byte(opcode.LoadStaticPrimitive8), 0, 0, // lo
				byte(opcode.LoadStaticPrimitive8), 0, 1, // hi
				byte(opcode.LoadStaticPrimitive8), 0, 2, // step
				byte(opcode.SliceStep),
			},
			4,
		},
		{
			// OpOrJump: same behaviour as OpAndJump for MaxStack
			"or chain a || b -> peak 1",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.OrJump), 9, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 1,
			},
			1,
		},
		{
			// Dead-code after OpReturn is skipped (analyzer treats Return as terminator)
			"unreachable code after return is ignored",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.Return), 1,
				// these instructions are dead — must not raise peak
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
			},
			1,
		},
		{
			// Unconditional jump over code that pushes a lot
			"unconditional jump skips high-push region",
			// 0: push 1 const
			// 3: Jump -> 15 (skip dead pushes)
			// 6..14: dead area
			// 15: push 1 const
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.Jump8), 15, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0, // dead 6
				byte(opcode.LoadStaticPrimitive8), 0, 0, // dead 9
				byte(opcode.LoadStaticPrimitive8), 0, 0, // dead 12
				byte(opcode.LoadStaticPrimitive8), 0, 0, // 15 (target)
			},
			2, // first push (1), then jump preserves, then target push -> peak 2 at merge
		},
		{
			// Empty array / record arity zero
			"empty array literal -> peak 1",
			[]byte{
				byte(opcode.MakeArray8), 0, 0,
			},
			1,
		},
		{
			// Pop-only ops shouldn't raise peak past entry height
			"pure pop sequence at height 0",
			[]byte{
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.LoadStaticPrimitive8), 0, 0,
				byte(opcode.Equal), // 2 -> 1
				byte(opcode.Pop),   // 1 -> 0
			},
			2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compiler.ComputeMaxStack(tc.ins)
			require.Equal(t, tc.want, got)
		})
	}
}
