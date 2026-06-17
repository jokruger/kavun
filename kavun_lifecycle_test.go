package kavun_test

import (
	"testing"

	"github.com/jokruger/kavun/core"
)

// Lifecycle / memory-management tests.
//
// These tests exercise the ownership policy from docs/memory-management.md, focusing on use-after-release
// scenarios rather than leaks. The refpool is configured with `ZeroOnRelease=true`, so a prematurely freed
// pooled value's slot is zeroed immediately — meaning a UAF on a string becomes the empty string, on an array
// becomes the empty array, on a dict becomes the empty dict, etc. Each test computes a dynamic (non-static)
// value through a lifecycle event and asserts its observed content.
//
// To force refpool slot reuse (so a freed slot would be overwritten by a later allocation) several tests
// allocate many short-lived strings after the value under test.

// --- §1: Compiled function returning local computed value ----------------------------------------------

// OpReturn must Pin the result before releaseFrameLocals; otherwise the local-backed return value would be
// zeroed when its frame slot is Released.
func TestLifecycle_Return_LocalComputedString(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func(a, b) {
			r := a + b
			return r
		}
		out = mk("hello_", "world")
	`, nil, "hello_world")
}

// Same as above but the local goes through several reassignments / mutations before return.
func TestLifecycle_Return_LocalAfterReassign(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func() {
			r := "init"
			r = r + "_step1"
			r = r + "_step2"
			return r
		}
		out = mk()
	`, nil, "init_step1_step2")
}

// --- §2: Compiled function returning an argument unchanged ---------------------------------------------

// Args are stored in frame locals; releaseFrameLocals would zero them at return time. OpReturn must Pin
// the result first.
func TestLifecycle_Return_ArgUnchanged(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		id := func(x) { return x }
		out = id("hello_" + "world")
	`, nil, "hello_world")
}

// Identity passed through several frames. Every intermediate Pin/Release must compose correctly.
func TestLifecycle_Return_ArgThroughDeepCallChain(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		a := func(v) { return v }
		b := func(v) { return a(v) }
		c := func(v) { return b(v) }
		d := func(v) { return c(v) }
		e := func(v) { return d(v) }
		out = e("dyn_" + "string")
	`, nil, "dyn_string")
}

// --- §3: Builtin returning an argument (Stage F Retain) -----------------------------------------------

// string(s) when s is already a string returns args[0]. Without Retain in the builtin the only ref would be
// dropped by OpCall's Release of the arg slot.
func TestLifecycle_Builtin_StringReturnsArg(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "abc" + "def"
		r := string(s)
		out = r + "_tail"
	`, nil, "abcdef_tail")
}

// bytes(b) when b is already bytes follows the same path.
func TestLifecycle_Builtin_BytesReturnsArg(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		b := bytes("xyz" + "qwe")
		r := bytes(b)
		out = string(r)
	`, nil, "xyzqwe")
}

// --- §4: Method returning self (Stage F Retain) ------------------------------------------------------

func TestLifecycle_Method_ArrayArrayReturnsSelf(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		a := [1, 2, 3]
		b := a.array()
		out = b.len()
	`, nil, int64(3))
}

func TestLifecycle_Method_DictDictReturnsSelf(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		d := dict({a: 1, b: 2})
		d2 := d.copy()
		out = d2.len()
	`, nil, int64(2))
}

// --- §5: Container stores computed value (DictAssign Pin / array element Pin) ------------------------

// SetSelLocal Releases the RHS; DictAssign must Pin before storing or the dict element is zeroed.
func TestLifecycle_Dict_StoreComputedValue(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		d := {}
		d["k"] = "hello" + "_world"
		out = d["k"]
	`, nil, "hello_world")
}

// Same but force pool slot reuse via many subsequent allocations.
func TestLifecycle_Dict_StoreComputedValue_WithPoolPressure(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		d := {}
		d["target"] = "preserve_me_" + "across_allocs"
		spam := []
		for i in range(0, 500, 1) {
			spam = append(spam, string(i) + "_x")
		}
		out = d["target"]
	`, nil, "preserve_me_across_allocs")
}

// Array element write via SetSelLocal goes through Assign hook; verify Pin keeps it alive.
func TestLifecycle_Array_StoreComputedValue(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		a := [undefined, undefined, undefined]
		a[1] = "computed_" + "string"
		out = a[1]
	`, nil, "computed_string")
}

// Append builtin pins each arg; verify retrieval after pressure.
func TestLifecycle_Array_AppendComputed_WithPoolPressure(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		a := []
		a = append(a, "first_" + "elem")
		a = append(a, "second_" + "elem")
		spam := []
		for i in range(0, 500, 1) {
			spam = append(spam, string(i))
		}
		out = a[0] + "|" + a[1]
	`, nil, "first_elem|second_elem")
}

// --- §6: Closure free variables (boxed *Value capture) -----------------------------------------------

// Closure captures a free variable; mutating it across calls must not free or alias incorrectly.
func TestLifecycle_Closure_CounterAcrossManyCalls(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func() {
			n := 0
			return func() { n += 1; return n }
		}
		c := mk()
		last := 0
		for i in range(0, 200, 1) { last = c() }
		out = last
	`, nil, int64(200))
}

// Closure captures a string-typed free var; mutation produces new strings, the slot pointer (*Value) stays valid.
func TestLifecycle_Closure_StringFreeVar_Mutation(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func() {
			s := "init"
			return {
				get: func() { return s },
				app: func(t) { s = s + t },
			}
		}
		c := mk()
		c.app("_a")
		c.app("_b")
		c.app("_c")
		out = c.get()
	`, nil, "init_a_b_c")
}

// Closure returned from a closure: outer's free var lives only through the inner closure's capture list.
// OpReturn on the outer must not free the captured slot box.
func TestLifecycle_Closure_NestedReturn(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func(prefix) {
			return func(suffix) {
				return prefix + "_" + suffix
			}
		}
		f := mk("hello")
		spam := []
		for i in range(0, 200, 1) { spam = append(spam, string(i)) }
		out = f("world")
	`, nil, "hello_world")
}

// Closure called via reduce many times — the accumulator string concatenation must not UAF.
func TestLifecycle_Closure_RepeatedInvocationInForEach(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		out = ["a", "b", "c", "d", "e"].reduce("", func(acc, v) { return acc + v + "_" })
	`, nil, "a_b_c_d_e_")
}

// --- §7: Tail-call optimization (Stage C: Release of overwritten locals) -----------------------------

// Tail-recursive accumulator over strings. New args overwrite locals every iteration; old locals must be
// Released exactly once, the new args must not double-Release.
func TestLifecycle_TailCall_StringAccumulator(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		loop := func(n, acc) {
			if n == 0 { return acc }
			return loop(n - 1, acc + "x")
		}
		out = loop(50, "start_")
	`, nil, "start_"+repeatString("x", 50))
}

// --- §8: Iterators over function/expression results -------------------------------------------------

// IteratorInit pops the iterable and Releases it; the iterator captures a Go slice header, and elements are
// pinned, so the iteration must observe each element correctly.
func TestLifecycle_Iterator_OverFunctionResult(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func() { return ["one_" + "1", "two_" + "2", "three_" + "3"] }
		collected := []
		for x in mk() { collected = append(collected, x) }
		out = collected.len() == 3 && collected[0] == "one_1" && collected[2] == "three_3"
	`, nil, true)
}

// Iterator over a method-call result.
func TestLifecycle_Iterator_OverMethodResult(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "a,b,c,d"
		parts := []
		for p in s.split(",") { parts = append(parts, p + "_x") }
		out = parts.len() == 4 && parts[0] == "a_x" && parts[3] == "d_x"
	`, nil, true)
}

// Iterating, then using the iterable again — verify the source survived the Release inside OpIteratorInit.
func TestLifecycle_Iterator_SourceArrayStillUsableAfter(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		a := ["one_" + "1", "two_" + "2"]
		acc := ""
		for x in a { acc = acc + x + "|" }
		out = acc + a[0] + "|" + a[1]
	`, nil, "one_1|two_2|one_1|two_2")
}

// --- §9: Chained method / index / slice expressions --------------------------------------------------

// Several intermediate dynamic values are pushed and Released in chain order. Every step's result must remain
// valid until consumed by the next op.
func TestLifecycle_Chain_MethodCallsOnString(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "alpha,beta,gamma"
		out = s.split(",")[1] + "_" + s.split(",")[2]
	`, nil, "beta_gamma")
}

// Slice on a computed concatenation: OpBinaryOp result is consumed by OpSliceIndex.
func TestLifecycle_Chain_SliceOnConcat(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := ("hello_" + "world")[0:5]
		out = s
	`, nil, "hello")
}

// Index access on a computed array.
func TestLifecycle_Chain_IndexOnArrayLiteral(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		out = ["a_" + "1", "b_" + "2", "c_" + "3"][1]
	`, nil, "b_2")
}

// --- §10: F-strings / formatted output --------------------------------------------------------------

// Format opcode replaces the operand slot with the formatted string; the operand value must be Released
// exactly once and not before Format reads it.
func TestLifecycle_Format_DynamicOperand(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "abc" + "def"
		out = f"<{s}>"
	`, nil, "<abcdef>")
}

func TestLifecycle_Format_ChainedExpression(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "xy" + "zw"
		out = f"len={s.len()}"
	`, nil, "len=4")
}

// --- §11: Defer with captured args ------------------------------------------------------------------

// OpDefer copies args at defer-time into a captured-args slice (ownership transferred from stack). When the
// defer runs (compiled or builtin path), the args must still be live.
func TestLifecycle_Defer_CapturedComputedArg(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		captured := undefined
		log := func(s) { captured = s }
		f := func() {
			s := "deferred_" + "value"
			defer log(s)
		}
		f()
		out = captured
	`, nil, "deferred_value")
}

// Many defers stack up — each must independently hold its captured args.
func TestLifecycle_Defer_ManyDefersCapturedSeparately(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		log := []
		record := func(s) { log = append(log, s) }
		f := func() {
			for i in range(0, 5, 1) {
				defer record("v_" + string(i))
			}
		}
		f()
		out = log.len() == 5 && log[0] == "v_4" && log[4] == "v_0"
	`, nil, true)
}

// --- §12: Recover / unwind paths (Stage E Pin-fallback) ---------------------------------------------

// Inner function raises an error carrying a dynamic payload string. Outer defer recovers it; the payload
// must remain readable even after the unwinder discards intermediate frames.
func TestLifecycle_Recover_PayloadString(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = string(e) + "_caught"
				}
			}()
			raise(error("boom_" + "1"))
		}
		out = f()
	`, nil, "boom_1_caught")
}

// Recovery several frames up.
func TestLifecycle_Recover_DeepUnwind(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		inner := func() { raise(error("deep_" + "boom")) }
		mid := func() { inner() }
		outer := func() res {
			defer func() {
				e := recover()
				if e != undefined { res = string(e) }
			}()
			mid()
		}
		out = outer()
	`, nil, "deep_boom")
}

// Defer that itself allocates after recover sees the error payload.
func TestLifecycle_Recover_AllocationAfterRecover(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					spam := []
					for i in range(0, 200, 1) { spam = append(spam, string(i)) }
					res = string(e) + "_after_spam"
				}
			}()
			raise(error("orig_" + "msg"))
		}
		out = f()
	`, nil, "orig_msg_after_spam")
}

// --- §13: Named result + deferred mutation ----------------------------------------------------------

// `return EXPR` writes EXPR into the named-result slot before defers; a defer may then re-read it. The
// stored value must remain valid across all the defers.
func TestLifecycle_NamedResult_DeferReadsAndMutates(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		f := func() res {
			defer func() { res = res + "_deferred" }()
			return "base_" + "value"
		}
		out = f()
	`, nil, "base_value_deferred")
}

// --- §14: Discarded results / pop opcode ------------------------------------------------------------

// Many calls whose results are discarded by OpPop — each result must be properly Released and not corrupt
// state for the next call.
func TestLifecycle_Pop_DiscardedManyCalls(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func(i) { return "v_" + string(i) }
		for i in range(0, 500, 1) { mk(i) }
		final := mk(999)
		out = final
	`, nil, "v_999")
}

// --- §15: Binary ops Release operands -----------------------------------------------------------------

// OpBinaryOp Releases both operands and produces a new owned result; chained concatenation must compose.
func TestLifecycle_BinaryOp_ChainedConcat(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		out = ("a_" + "1") + ("_b_" + "2") + ("_c_" + "3")
	`, nil, "a_1_b_2_c_3")
}

// --- §16: Comparison / Equal Release operands --------------------------------------------------------

// OpEqual must Release both operands. Verify subsequent allocations don't see stale slots.
func TestLifecycle_Equal_DynamicOperandsReleased(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "hello"
		eq := ("hel" + "lo") == s
		spam := []
		for i in range(0, 200, 1) { spam = append(spam, string(i)) }
		out = eq && s == "hello"
	`, nil, true)
}

// --- §17: Conditional (JumpFalsy) Release condition --------------------------------------------------

// OpJumpFalsy on a dynamic-string condition must Release the condition value when not taking the truthy
// branch, and not corrupt unrelated state.
func TestLifecycle_Cond_DynamicConditionReleased(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		s := "preserve_" + "me"
		for i in range(0, 50, 1) {
			if ("a" + string(i)) == "stop" { break }
		}
		out = s
	`, nil, "preserve_me")
}

// --- §18: Splits and stdlib container builders (Pattern C: Pin on store) -----------------------------

func TestLifecycle_Split_ElementsSurvivePoolPressure(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		text := import("text")
		parts := text.split("alpha,beta,gamma,delta,epsilon", ",")
		spam := []
		for i in range(0, 500, 1) { spam = append(spam, string(i) + "_pad") }
		out = parts[0] + "|" + parts[4]
	`, nil, "alpha|epsilon")
}

func TestLifecycle_Fields_ElementsSurvivePoolPressure(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		text := import("text")
		parts := text.fields("  one two three  four  ")
		spam := []
		for i in range(0, 500, 1) { spam = append(spam, string(i)) }
		out = parts.len() == 4 && parts[0] == "one" && parts[3] == "four"
	`, nil, true)
}

// --- §19: Container element retrieval (pinned-in-container path) -------------------------------------

// Pull element from a container; original container goes out of scope. The element was Pinned when stored
// (per §5) so it survives indefinitely (until arena Reset).
func TestLifecycle_ElementRetrieval_AfterContainerOutOfScope(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		mk := func() {
			d := {}
			d["k"] = "stored_" + "value"
			return d["k"]
		}
		got := mk()
		spam := []
		for i in range(0, 200, 1) { spam = append(spam, string(i)) }
		out = got
	`, nil, "stored_value")
}

// --- §20: Composite stress test ----------------------------------------------------------------------

// A combination: closures with free vars, deferred calls, container mutation, recovery — exercises many
// lifecycle paths together.
func TestLifecycle_Composite_StressMixedPaths(t *testing.T) {
	rta := core.NewArena(nil)
	expectRun(t, rta, `
		state := { log: [], counter: 0 }
		make_logger := func(prefix) {
			return func(msg) {
				state.counter += 1
				state.log = append(state.log, prefix + ":" + msg)
			}
		}
		log_info := make_logger("INFO")
		log_warn := make_logger("WARN")

		work := func(n) res {
			defer log_info(f"done_{n}")
			defer func() {
				e := recover()
				if e != undefined { res = "recovered_" + string(e) }
			}()
			if n < 0 { raise(error("neg_" + string(n))) }
			return f"ok_{n}"
		}

		results := []
		for i in range(-2, 3, 1) {
			results = append(results, work(i))
		}
		log_warn("end")

		out = results.len() == 5 &&
		      results[0] == "recovered_neg_-2" &&
		      results[1] == "recovered_neg_-1" &&
		      results[2] == "ok_0" &&
		      results[4] == "ok_2" &&
		      state.counter == 6 &&
		      state.log.len() == 6 &&
		      state.log[5] == "WARN:end"
	`, nil, true)
}

// --- helpers ----------------------------------------------------------------------------------------

func repeatString(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
