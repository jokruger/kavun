package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
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

func TestRecover_VMError_IsRuntime(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.is_runtime()
				}
			}()
			x := 1 / 0
		}
		out = f()
	`, nil, true)
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

func TestRecover_RaisedUserError_IsNotRuntime(t *testing.T) {
	expectRun(t, `
		f := func() res {
			defer func() {
				e := recover()
				if e != undefined {
					res = e.is_runtime()
				}
			}()
			raise(error("nope"))
		}
		out = f()
	`, nil, false)
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

// --- severity / recoverability tests ---

// is_runtime() returns false for user errors and true for runtime ones.
func TestRecover_IsRuntime_ForRuntimeError(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.is_runtime()
    }
  }()
  x := 1 / 0
}
out = f()
`, nil, true)
}

func TestRecover_IsRuntime_ForUserError(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.is_runtime()
    }
  }()
  raise(error("oops"))
}
out = f()
`, nil, false)
}

// kind() reports specific runtime error kinds; new "not_iterable" tag should surface when iterating a non-iterable value.
func TestRecover_NotIterable_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  for i in true {  // bool is not_iterable
    _ = i
  }
}
out = f()
`, nil, "not_iterable")
}

// not_callable kind is exposed via recover().
func TestRecover_NotCallable_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  x := 42
  x()
}
out = f()
`, nil, "not_callable")
}

// wrong_num_arguments is exposed via recover().
func TestRecover_WrongNumArguments_Kind(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  g := func(a, b) { return a + b }
  g(1)
}
out = f()
`, nil, "wrong_num_arguments")
}

// User-raised errors carry an empty kind (kind() returns "").
func TestRecover_UserError_KindIsUser(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined {
      res = e.kind()
    }
  }()
  raise(error("boom"))
}
out = f()
`, nil, "user")
}

// Critical (Fatal) Go errors raised by host-supplied builtins must bypass deferred recover() and escape directly to the host.
func TestRecover_FatalErrorBypassesRecover(t *testing.T) {
	fatalBuiltin := core.NewBuiltinFunctionValue(
		"do_fatal",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, errs.NewFatalError("custom_fatal", "host requested abort")
		}, 0, false)

	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // tries to swallow but cannot
  do_fatal()
}
f()
`,
		Opts().Symbol("do_fatal", fatalBuiltin).Skip2ndPass(),
		"custom_fatal: host requested abort",
	)
}

// Recoverable Go errors raised by host-supplied builtins are caught by deferred recover().
func TestRecover_RecoverableErrorIsCaught(t *testing.T) {
	recBuiltin := core.NewBuiltinFunctionValue(
		"do_logical",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, errs.NewRecoverableError("custom_kind", "user level mistake")
		}, 0, false)

	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined { res = e.kind() }
  }()
  do_logical()
}
out = f()
`,
		Opts().Symbol("do_logical", recBuiltin).Skip2ndPass(),
		"custom_kind",
	)
}

// Script-level fatal errors raised via `error(payload, true)` must bypass deferred recover() and escape directly to the host.
func TestRecover_ScriptFatalErrorBypassesRecover(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // tries to swallow but cannot
  raise(error("boom", true))
}
f()
`,
		Opts().Skip2ndPass(),
		"boom",
	)
}

// raise(err, true) promotes an otherwise-recoverable error to fatal so recover() cannot catch it.
func TestRecover_RaiseFatalFlagPromotesToFatal(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()
  raise(error("boom"), true)
}
f()
`,
		Opts().Skip2ndPass(),
		"boom",
	)
}

// raise(non_error, true) wraps the payload in a fatal error.
func TestRecover_RaiseFatalFlagOnRawPayload(t *testing.T) {
	expectError(t, `
f := func() {
  defer func() { _ = recover() }()
  raise("plain", true)
}
f()
`,
		Opts().Skip2ndPass(),
		"plain",
	)
}

// raise(err, false) demotes a fatal error back to recoverable so recover() catches it; the original error value is
// left unchanged.
func TestRecover_RaiseFalseFlagDemotesToRecoverable(t *testing.T) {
	expectRun(t, `
e := error("boom", true)
f := func() res {
  defer func() {
    r := recover()
    if r != undefined { res = r.is_fatal() }
  }()
  raise(e, false)
}
out = [f(), e.is_fatal()]
`, nil, ARR{false, true})
}

// Script-level error with explicit fatal=false is still recoverable (matches default).
func TestRecover_ScriptExplicitNonFatalIsRecovered(t *testing.T) {
	expectRun(t, `
f := func() res {
  defer func() {
    e := recover()
    if e != undefined { res = e.kind() }
  }()
  raise(error("boom", false))
}
out = f()
`, nil, "user")
}

// --- regression tests for newly-improved behavior ---

// `return EXPR` in a function with a named result is sugar for `name = EXPR; return`. Defers can observe and mutate
// the returned value through the named result. Matches Go semantics.
func TestReturnExpr_NamedResult_DeferMutates(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { r = r + 1 }()
			return 41
		}
		out = f()
	`, nil, 42)
}

func TestReturnExpr_NamedResult_DeferOverrides(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { r = "deferred" }()
			return "explicit"
		}
		out = f()
	`, nil, "deferred")
}

func TestReturnExpr_NamedResult_NoDefer_UnaffectedByNamedSlot(t *testing.T) {
	// Without defers, `return EXPR` should still produce EXPR — writing to the named-result slot is a no-op for
	// the visible return value when there are no defers to observe it.
	expectRun(t, `
		f := func() r {
			r = "init"
			return "explicit"
		}
		out = f()
	`, nil, "explicit")
}

func TestReturnExpr_NoNamedResult_DeferIrrelevant(t *testing.T) {
	expectRun(t, `
		f := func() {
			defer func() {}()
			return 7
		}
		out = f()
	`, nil, 7)
}

// `defer obj.method()` calls the method when the surrounding function exits. recover() inside such a method does
// NOT catch a raised error (the method dispatch path doesn't push a Kavun-level deferred-for frame). This codifies
// the current limitation; if/when method-call defers gain recover support, this test should be updated.
func TestDeferMethodCall_DoesNotEnableRecover(t *testing.T) {
	expectError(t, `
		// `+"`recover_helper`"+` is reachable as a method of nothing — we just verify recover() inside a deferred
		// method call (acting on a value) cannot swallow a raised error.
		f := func() {
			arr := [1,2,3]
			defer arr.sort()  // a valid deferred method call; sort() can't recover()
			raise(error("escapes_through_method_defer"))
		}
		f()
	`, nil, "escapes_through_method_defer")
}

// recover() invoked from inside a host builtin running as a defer returns Undefined (the builtin is not a Kavun
// deferred-for frame). Therefore the raised error escapes.
func TestRecover_FromHostBuiltinAsDefer_IsIneffective(t *testing.T) {
	probe := core.NewBuiltinFunctionValue(
		"probe_recover",
		func(v core.VM, args []core.Value) (core.Value, error) {
			// Try to recover from inside a deferred builtin — must return Undefined.
			return v.Recover(), nil
		}, 0, false)

	expectError(t, `
f := func() {
  defer probe_recover()
  raise(error("escapes_past_builtin_defer"))
}
f()
`,
		Opts().Symbol("probe_recover", probe).Skip2ndPass(),
		"escapes_past_builtin_defer",
	)
}

// A host builtin that returns a raw (non-*errs.Error) Go error is classified Fatal and bypasses recover(). This
// matches the documented severity policy: any non-*errs.Error defaults to Fatal.
func TestRecover_RawGoErrorFromBuiltin_IsFatal(t *testing.T) {
	rawBuiltin := core.NewBuiltinFunctionValue(
		"do_raw",
		func(v core.VM, args []core.Value) (core.Value, error) {
			return core.Undefined, fmt.Errorf("plain go error")
		}, 0, false)

	expectError(t, `
f := func() {
  defer func() { _ = recover() }()  // cannot catch — error is Fatal
  do_raw()
}
f()
`,
		Opts().Symbol("do_raw", rawBuiltin).Skip2ndPass(),
		"plain go error",
	)
}

// Stress: many defers (1000) all run in LIFO order; the first-registered defer (running last) sees the accumulated
// counter. Exercises arena-allocated args slice and per-defer state cleanup at scale.
func TestDefer_ManyDefers_AllRun(t *testing.T) {
	expectRun(t, `
		f := func() res {
			counter := 0
			defer func() { res = counter }()  // registered FIRST → runs LAST → sees final counter
			for i := 0; i < 1000; i = i + 1 {
				defer func() { counter = counter + 1 }()
			}
		}
		out = f()
	`, nil, 1000)
}

// recover() called from a nested *non-deferred* helper function returns undefined and the error propagates.
// This is the contrapositive of TestRecover_OnlyDirectlyInDeferred phrased in terms of the new Recover() guard.
func TestRecover_NestedHelper_ReturnsUndefined(t *testing.T) {
	expectError(t, `
		helper := func() { _ = recover() }
		f := func() {
			defer func() { helper() }()
			raise(error("nested_helper_cannot_recover"))
		}
		f()
	`, nil, "nested_helper_cannot_recover")
}
