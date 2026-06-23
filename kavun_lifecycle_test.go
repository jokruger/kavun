package kavun_test

import (
	"testing"
)

func TestLifecycle_Return_LocalComputedString(t *testing.T) {
	expectRun(t, `
		mk := func(a, b) {
			r := a + b
			return r
		}
		out = mk("hello_", "world")
	`, nil, "hello_world")
}

func TestLifecycle_Return_LocalAfterReassign(t *testing.T) {
	expectRun(t, `
		mk := func() {
			r := "init"
			r = r + "_step1"
			r = r + "_step2"
			return r
		}
		out = mk()
	`, nil, "init_step1_step2")
}

func TestLifecycle_Return_ArgUnchanged(t *testing.T) {
	expectRun(t, `
		id := func(x) { return x }
		out = id("hello_" + "world")
	`, nil, "hello_world")
}

func TestLifecycle_Return_ArgThroughDeepCallChain(t *testing.T) {
	expectRun(t, `
		a := func(v) { return v }
		b := func(v) { return a(v) }
		c := func(v) { return b(v) }
		d := func(v) { return c(v) }
		e := func(v) { return d(v) }
		out = e("dyn_" + "string")
	`, nil, "dyn_string")
}

func TestLifecycle_Builtin_StringReturnsArg(t *testing.T) {
	expectRun(t, `
		s := "abc" + "def"
		r := string(s)
		out = r + "_tail"
	`, nil, "abcdef_tail")
}

func TestLifecycle_Builtin_BytesReturnsArg(t *testing.T) {
	expectRun(t, `
		b := bytes("xyz" + "qwe")
		r := bytes(b)
		out = string(r)
	`, nil, "xyzqwe")
}

func TestLifecycle_Method_ArrayArrayReturnsSelf(t *testing.T) {
	expectRun(t, `
		a := [1, 2, 3]
		b := a.array()
		out = b.len()
	`, nil, int64(3))
}

func TestLifecycle_Method_DictDictReturnsSelf(t *testing.T) {
	expectRun(t, `
		d := dict({a: 1, b: 2})
		d2 := d.copy()
		out = d2.len()
	`, nil, int64(2))
}

func TestLifecycle_Dict_StoreComputedValue(t *testing.T) {
	expectRun(t, `
		d := {}
		d["k"] = "hello" + "_world"
		out = d["k"]
	`, nil, "hello_world")
}

func TestLifecycle_Dict_StoreComputedValue_WithPoolPressure(t *testing.T) {
	expectRun(t, `
		d := {}
		d["target"] = "preserve_me_" + "across_allocs"
		spam := []
		for i in range(0, 500, 1) {
			spam = append(spam, string(i) + "_x")
		}
		out = d["target"]
	`, nil, "preserve_me_across_allocs")
}

func TestLifecycle_Array_StoreComputedValue(t *testing.T) {
	expectRun(t, `
		a := [undefined, undefined, undefined]
		a[1] = "computed_" + "string"
		out = a[1]
	`, nil, "computed_string")
}

func TestLifecycle_Array_AppendComputed_WithPoolPressure(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Closure_CounterAcrossManyCalls(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Closure_StringFreeVar_Mutation(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Closure_NestedReturn(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Closure_RepeatedInvocationInForEach(t *testing.T) {
	expectRun(t, `
		out = ["a", "b", "c", "d", "e"].reduce("", func(acc, v) { return acc + v + "_" })
	`, nil, "a_b_c_d_e_")
}

func TestLifecycle_TailCall_StringAccumulator(t *testing.T) {
	expectRun(t, `
		loop := func(n, acc) {
			if n == 0 { return acc }
			return loop(n - 1, acc + "x")
		}
		out = loop(50, "start_")
	`, nil, "start_"+repeatString("x", 50))
}

func TestLifecycle_Iterator_OverFunctionResult(t *testing.T) {
	expectRun(t, `
		mk := func() { return ["one_" + "1", "two_" + "2", "three_" + "3"] }
		collected := []
		for x in mk() { collected = append(collected, x) }
		out = collected.len() == 3 && collected[0] == "one_1" && collected[2] == "three_3"
	`, nil, true)
}

func TestLifecycle_Iterator_OverMethodResult(t *testing.T) {
	expectRun(t, `
		s := "a,b,c,d"
		parts := []
		for p in s.split(",") { parts = append(parts, p + "_x") }
		out = parts.len() == 4 && parts[0] == "a_x" && parts[3] == "d_x"
	`, nil, true)
}

func TestLifecycle_Iterator_SourceArrayStillUsableAfter(t *testing.T) {
	expectRun(t, `
		a := ["one_" + "1", "two_" + "2"]
		acc := ""
		for x in a { acc = acc + x + "|" }
		out = acc + a[0] + "|" + a[1]
	`, nil, "one_1|two_2|one_1|two_2")
}

func TestLifecycle_Chain_MethodCallsOnString(t *testing.T) {
	expectRun(t, `
		s := "alpha,beta,gamma"
		out = s.split(",")[1] + "_" + s.split(",")[2]
	`, nil, "beta_gamma")
}

func TestLifecycle_Chain_SliceOnConcat(t *testing.T) {
	expectRun(t, `
		s := ("hello_" + "world")[0:5]
		out = s
	`, nil, "hello")
}

func TestLifecycle_Chain_IndexOnArrayLiteral(t *testing.T) {
	expectRun(t, `
		out = ["a_" + "1", "b_" + "2", "c_" + "3"][1]
	`, nil, "b_2")
}

func TestLifecycle_Format_DynamicOperand(t *testing.T) {
	expectRun(t, `
		s := "abc" + "def"
		out = f"<{s}>"
	`, nil, "<abcdef>")
}

func TestLifecycle_Format_ChainedExpression(t *testing.T) {
	expectRun(t, `
		s := "xy" + "zw"
		out = f"len={s.len()}"
	`, nil, "len=4")
}

func TestLifecycle_Defer_CapturedComputedArg(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Defer_ManyDefersCapturedSeparately(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Recover_PayloadString(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Recover_DeepUnwind(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Recover_AllocationAfterRecover(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_NamedResult_DeferReadsAndMutates(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() { res = res + "_deferred" }()
			return "base_" + "value"
		}
		out = f()
	`, nil, "base_value_deferred")
}

func TestLifecycle_Pop_DiscardedManyCalls(t *testing.T) {
	expectRun(t, `
		mk := func(i) { return "v_" + string(i) }
		for i in range(0, 500, 1) { mk(i) }
		final := mk(999)
		out = final
	`, nil, "v_999")
}

func TestLifecycle_BinaryOp_ChainedConcat(t *testing.T) {
	expectRun(t, `
		out = ("a_" + "1") + ("_b_" + "2") + ("_c_" + "3")
	`, nil, "a_1_b_2_c_3")
}

func TestLifecycle_Equal_DynamicOperandsReleased(t *testing.T) {
	expectRun(t, `
		s := "hello"
		eq := ("hel" + "lo") == s
		spam := []
		for i in range(0, 200, 1) { spam = append(spam, string(i)) }
		out = eq && s == "hello"
	`, nil, true)
}

func TestLifecycle_Cond_DynamicConditionReleased(t *testing.T) {
	expectRun(t, `
		s := "preserve_" + "me"
		for i in range(0, 50, 1) {
			if ("a" + string(i)) == "stop" { break }
		}
		out = s
	`, nil, "preserve_me")
}

func TestLifecycle_Split_ElementsSurvivePoolPressure(t *testing.T) {
	expectRun(t, `
		text := import("text")
		parts := text.split("alpha,beta,gamma,delta,epsilon", ",")
		spam := []
		for i in range(0, 500, 1) { spam = append(spam, string(i) + "_pad") }
		out = parts[0] + "|" + parts[4]
	`, nil, "alpha|epsilon")
}

func TestLifecycle_Fields_ElementsSurvivePoolPressure(t *testing.T) {
	expectRun(t, `
		text := import("text")
		parts := text.fields("  one two three  four  ")
		spam := []
		for i in range(0, 500, 1) { spam = append(spam, string(i)) }
		out = parts.len() == 4 && parts[0] == "one" && parts[3] == "four"
	`, nil, true)
}

func TestLifecycle_ElementRetrieval_AfterContainerOutOfScope(t *testing.T) {
	expectRun(t, `
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

func TestLifecycle_Composite_StressMixedPaths(t *testing.T) {
	expectRun(t, `
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

func repeatString(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
