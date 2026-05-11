package unit

import (
	"testing"

	"github.com/jokruger/kavun/parser"
)

// --- named return value ---

func TestNamedReturn_DefaultUndefined(t *testing.T) {
	expectRun(t, `
		f := func() res {
			// no assignment to res — bare return yields undefined
			return
		}
		out = is_undefined(f())
	`, nil, true)
}

func TestNamedReturn_AssignThenBareReturn(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = 42
			return
		}
		out = f()
	`, nil, 42)
}

func TestNamedReturn_AssignNoReturnStmt(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = "hello"
		}
		out = f()
	`, nil, "hello")
}

func TestNamedReturn_ExplicitReturnOverridesNamed(t *testing.T) {
	expectRun(t, `
		f := func() res {
			res = "named"
			return "explicit"
		}
		out = f()
	`, nil, "explicit")
}

func TestNamedReturn_ParameterCollision_Errors(t *testing.T) {
	expectError(t, `
		f := func(x) x { return }
		out = f(1)
	`, nil, "named result")
}

func TestNamedReturn_UnderscoreNotAllowed(t *testing.T) {
	expectError(t, `
		f := func() _ { return }
		out = f()
	`, nil, "named result cannot be '_'")
}

// Regression: each call must reset the named-result slot to undefined.
// Previously the slot reused whatever stack value the previous call left behind, so a function that didn't assign its
// named result could observe a stale value from an unrelated earlier call.
func TestNamedReturn_SlotResetBetweenCalls(t *testing.T) {
	expectRun(t, `
		sign := func(x) s {
			if x > 0 { s = 1 }
			if x < 0 { s = -1 }
			if x == 0 { s = 0 }
		}
		maybe := func(x) r {
			if x { return }
			r = "set"
		}
		_ = sign(0)         // leaves 0 in the slot region
		out = is_undefined(maybe(true))
	`, nil, true)
}

func TestNamedReturn_ReadBeforeAssignIsUndefined(t *testing.T) {
	expectRun(t, `
		f := func() r {
			before := r
			r = 5
			return before
		}
		out = is_undefined(f())
	`, nil, true)
}

func TestNamedReturn_RecursionUsesOwnSlot(t *testing.T) {
	expectRun(t, `
		fact := func(n) r {
			if n <= 1 { r = 1; return }
			r = n * fact(n - 1)
		}
		out = fact(6)
	`, nil, 720)
}

func TestNamedReturn_ConditionalAssignment(t *testing.T) {
	expectRun(t, `
		sign := func(x) s {
			if x > 0 { s = 1 }
			if x < 0 { s = -1 }
			if x == 0 { s = 0 }
		}
		out = [sign(-7), sign(0), sign(3)]
	`, nil, ARR{-1, 0, 1})
}

func TestNamedReturn_ShadowedInInnerBlock(t *testing.T) {
	// A `:=` inside a nested block introduces a new local that shadows the named-result symbol; the outer slot is
	// untouched.
	expectRun(t, `
		f := func() r {
			r = "outer"
			if true {
				r := "inner"
				_ = r
			}
		}
		out = f()
	`, nil, "outer")
}

func TestNamedReturn_MutateThroughReference(t *testing.T) {
	expectRun(t, `
		build := func() obj {
			obj = {a: 1}
			obj.b = 2
		}
		r := build()
		out = [r.a, r.b]
	`, nil, ARR{1, 2})
}

func TestNamedReturn_CapturedByClosure(t *testing.T) {
	// The named result holds a closure that captures a sibling local.
	// Each invocation of the returned closure must observe the same captured environment (closure-over-local, not over
	// slot value).
	expectRun(t, `
		counter := func() c {
			n := 0
			c = func() { n = n + 1; return n }
		}
		inc := counter()
		out = [inc(), inc(), inc()]
	`, nil, ARR{1, 2, 3})
}

func TestNamedReturn_ImmediatelyInvoked(t *testing.T) {
	expectRun(t, `
		out = (func() r { r = 99 })()
	`, nil, 99)
}

func TestNamedReturn_ForLoopAccumulation(t *testing.T) {
	expectRun(t, `
		sumto := func(n) total {
			total = 0
			for i := 1; i <= n; i = i + 1 { total = total + i }
		}
		out = sumto(10)
	`, nil, 55)
}

func TestNamedReturn_VariadicWithNamedResult(t *testing.T) {
	expectRun(t, `
		joinall := func(sep, ...xs) joined {
			joined = ""
			for x in xs {
				if joined == "" { joined = string(x) } else { joined = joined + sep + string(x) }
			}
		}
		out = joinall(",", 1, 2, 3)
	`, nil, "1,2,3")
}

func TestNamedReturn_NameMayShadowBuiltin(t *testing.T) {
	// The named-result identifier is just a local symbol; it can use the same spelling as a builtin (here `len`)
	// without ambiguity.
	expectRun(t, `
		f := func() len {
			len = 7
		}
		out = f()
	`, nil, 7)
}

func TestNamedReturn_BareReturnInLoopUsesNamedSlot(t *testing.T) {
	// A bare `return` inside a loop must yield the current named-result value, not what the call stack happens to hold.
	expectRun(t, `
		find := func(arr, target) idx {
			idx = -1
			for i := 0; i < len(arr); i = i + 1 {
				if arr[i] == target { idx = i; return }
			}
		}
		out = [find([10, 20, 30, 40], 30), find([10, 20, 30, 40], 99)]
	`, nil, ARR{2, -1})
}

func TestNamedReturn_ExplicitReturnExprIgnoresNamedSlot(t *testing.T) {
	expectRun(t, `
		f := func() r {
			r = 1
			return r + 100  // expression value wins
		}
		out = f()
	`, nil, 101)
}

func TestNamedReturn_ReassignMultipleTimes(t *testing.T) {
	expectRun(t, `
		f := func() r {
			r = 1
			r = r + 10
			r = r * 2
		}
		out = f()
	`, nil, 22)
}

// --- defer ---

func TestDefer_RunsOnExit(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "a") }()
			log = append(log, "b")
		}
		f()
		out = log
	`, nil, ARR{"b", "a"})
}

func TestDefer_LIFOOrder(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, 1) }()
			defer func() { log = append(log, 2) }()
			defer func() { log = append(log, 3) }()
		}
		f()
		out = log
	`, nil, ARR{3, 2, 1})
}

func TestDefer_ArgsCapturedAtDeferTime(t *testing.T) {
	// Plain-call defer evaluates its argument expressions at defer statement time, not at call time.
	expectRun(t, `
		seen := undefined
		record := func(v) { seen = v }
		f := func() {
			x := 10
			defer record(x)
			x = 20
		}
		f()
		out = seen
	`, nil, 10)
}

func TestDefer_RunsOnExplicitReturn(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "deferred") }()
			return
		}
		f()
		out = log
	`, nil, ARR{"deferred"})
}

func TestDefer_OutsideFunction_Errors(t *testing.T) {
	expectError(t, `defer foo()`, nil, "defer not allowed outside function")
}

func TestDefer_NonCall_Errors(t *testing.T) {
	testFileSet := parser.NewFileSet()
	src := `f := func() { defer 1+1 }`
	testFile := testFileSet.AddFile("test", -1, len(src))
	p := parser.NewParser(testFile, []byte(src), nil)
	_, err := p.ParseFile()
	if err == nil {
		t.Fatal("expected parse error for non-call defer, got none")
	}
}

// --- recover() ---

func TestRecover_OutsideDeferred_ReturnsUndefined(t *testing.T) {
	expectRun(t, `
		f := func() { return is_undefined(recover()) }
		out = f()
	`, nil, true)
}

func TestRecover_NoErrorInDeferred_ReturnsUndefined(t *testing.T) {
	expectRun(t, `
		got := undefined
		f := func() {
			defer func() {
				got = recover()
			}()
		}
		f()
		out = is_undefined(got)
	`, nil, true)
}

func TestRecover_CatchesVMError(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = "caught"
				}
			}()
			x := 1 / 0
			res = "no_error"
		}
		out = f()
	`, nil, "caught")
}

func TestRecover_VMError_OriginIsVM(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.origin()
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, "vm")
}

func TestRecover_VMError_HasKind(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.kind()
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, "division_by_zero")
}

func TestRecover_RaiseUserError(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.value()
				}
			}()
			raise(error({code: "boom"}))
		}
		v := f()
		out = v.code
	`, nil, "boom")
}

func TestRecover_RaisedUserError_OriginIsUser(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.origin()
				}
			}()
			raise(error("nope"))
		}
		out = f()
	`, nil, "user")
}

func TestRecover_OnlyDirectlyInDeferred(t *testing.T) {
	// recover() must be called directly from the deferred function; indirection through another call returns undefined,
	// so the raised error is not cleared and propagates out.
	expectError(t, `
		inner := func() { return recover() }
		f := func() {
			defer func() {
				inner()
			}()
			raise(error("escapes_through_inner"))
		}
		f()
	`, nil, "escapes_through_inner")
}

func TestRecover_ErrorEscapesIfNotRecovered(t *testing.T) {
	expectError(t, `
		f := func() {
			defer func() {
				// don't call recover()
			}()
			raise(error("escapes"))
		}
		f()
	`, nil, "escapes")
}

func TestDefer_RunsBeforeUnrecoveredErrorEscapes(t *testing.T) {
	expectError(t, `
		log := []
		f := func() {
			defer func() { log = append(log, "did defer") }()
			raise(error("oops"))
		}
		f()
	`, nil, "oops")
}

func TestRecover_NamedResultUpdatedByDefer(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				if recover() != undefined {
					res = "rescued"
				}
			}()
			res = "ok"
			raise(error("bang"))
		}
		out = f()
	`, nil, "rescued")
}

func TestDefer_AccessAndModifyNamedResult(t *testing.T) {
	expectRun(t, `
		f := func(x) res {
			defer func() {
				res = res + 100
			}()
			res = x
		}
		out = f(5)
	`, nil, 105)
}

// --- multiple defers + recover interaction ---

func TestDefer_LaterDeferStillRunsAfterRecover(t *testing.T) {
	expectRun(t, `
		log := []
		f := func() res {
			defer func() { log = append(log, "outer") }()
			defer func() {
				if recover() != undefined {
					log = append(log, "recovered")
				}
			}()
			raise(error("boom"))
		}
		f()
		out = log
	`, nil, ARR{"recovered", "outer"})
}

func TestDefer_RaisedInsideDefer_CanBeRecoveredByEarlierDefer(t *testing.T) {
	// defers run LIFO; an earlier-registered defer (= later to run) can recover an error raised by a later-registered
	// defer (= run earlier).
	expectRun(t, `
		f := func() res {
			defer func() {
				if recover() != undefined {
					res = "outer_caught"
				}
			}()
			defer func() {
				raise(error("from_inner_defer"))
			}()
			res = "ok"
		}
		out = f()
	`, nil, "outer_caught")
}
