package unit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/tests/require"
	"github.com/jokruger/kavun/vm"
)

// -----------------------------------------------------------------------------
// is_* type-predicate builtins. These were entirely uncovered by the original
// test suite (every is_* function reported 0% line coverage).
// -----------------------------------------------------------------------------

func TestBuiltinIsPredicates(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want bool
	}{
		// is_string
		{"is_string/string", `is_string("a")`, true},
		{"is_string/runes", `is_string(runes("a"))`, false},
		{"is_string/int", `is_string(1)`, false},

		// is_runes
		{"is_runes/runes", `is_runes(runes("a"))`, true},
		{"is_runes/string", `is_runes("a")`, false},

		// is_int
		{"is_int/int", `is_int(1)`, true},
		{"is_int/float", `is_int(1.0)`, false},

		// is_float
		{"is_float/float", `is_float(1.0)`, true},
		{"is_float/int", `is_float(1)`, false},

		// is_decimal
		{"is_decimal/decimal", `is_decimal(decimal("1.5"))`, true},
		{"is_decimal/float", `is_decimal(1.5)`, false},

		// is_bool
		{"is_bool/true", `is_bool(true)`, true},
		{"is_bool/int", `is_bool(0)`, false},

		// is_byte
		{"is_byte/byte", `is_byte(byte(0))`, true},
		{"is_byte/int", `is_byte(0)`, false},

		// is_rune
		{"is_rune/rune", `is_rune('a')`, true},
		{"is_rune/int", `is_rune(97)`, false},

		// is_bytes
		{"is_bytes/bytes", `is_bytes(bytes("a"))`, true},
		{"is_bytes/string", `is_bytes("a")`, false},

		// is_array
		{"is_array/array", `is_array([])`, true},
		{"is_array/dict", `is_array({})`, false},

		// is_record
		{"is_record/record", `is_record({})`, true},
		{"is_record/dict", `is_record(dict({}))`, false},

		// is_dict
		{"is_dict/dict", `is_dict(dict({}))`, true},
		{"is_dict/record", `is_dict({})`, false},

		// is_range
		{"is_range/range", `is_range(range(0, 5, 1))`, true},
		{"is_range/array", `is_range([])`, false},

		// is_immutable
		{"is_immutable/immutable", `is_immutable(immutable([1, 2]))`, true},
		{"is_immutable/mutable", `is_immutable([1, 2])`, false},
		{"is_immutable/string", `is_immutable("x")`, true},
		{"is_immutable/int", `is_immutable(1)`, true},

		// is_time
		{"is_time/time", `is_time(time())`, true},
		{"is_time/int", `is_time(1)`, false},

		// is_error
		{"is_error/error", `is_error(error("oops"))`, true},
		{"is_error/string", `is_error("x")`, false},

		// is_undefined
		{"is_undefined/undef", `is_undefined(undefined)`, true},
		{"is_undefined/zero", `is_undefined(0)`, false},

		// is_function
		{"is_function/lambda", `is_function(func(){})`, true},
		{"is_function/builtin", `is_function(len)`, true},
		{"is_function/int", `is_function(1)`, false},

		// is_callable
		{"is_callable/lambda", `is_callable(func(){})`, true},
		{"is_callable/builtin", `is_callable(len)`, true},
		{"is_callable/int", `is_callable(1)`, false},

		// is_iterable
		{"is_iterable/array", `is_iterable([])`, true},
		{"is_iterable/string", `is_iterable("a")`, true},
		{"is_iterable/range", `is_iterable(range(0, 1, 1))`, true},
		{"is_iterable/int", `is_iterable(1)`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectRun(t, "out = "+c.expr, nil, c.want)
		})
	}
}

func TestBuiltinIsPredicates_WrongArity(t *testing.T) {
	for _, name := range []string{
		"is_string", "is_runes", "is_int", "is_float", "is_decimal",
		"is_bool", "is_byte", "is_rune", "is_bytes", "is_array",
		"is_record", "is_dict", "is_range", "is_immutable", "is_time",
		"is_error", "is_undefined", "is_function", "is_callable", "is_iterable",
	} {
		t.Run(name, func(t *testing.T) {
			expectError(t, name+"()", nil,
				fmt.Sprintf("wrong_num_arguments: (%s) expected 1 argument(s), got 0", name))
		})
	}
}

func TestBuiltinTypeName(t *testing.T) {
	expectRun(t, `out = type_name(1)`, nil, "int")
	expectRun(t, `out = type_name(1.0)`, nil, "float")
	expectRun(t, `out = type_name("x")`, nil, "string")
	expectRun(t, `out = type_name([])`, nil, "array")
	expectRun(t, `out = type_name({})`, nil, "record")
	expectRun(t, `out = type_name(dict({}))`, nil, "dict")
	expectRun(t, `out = type_name(undefined)`, nil, "undefined")
	expectRun(t, `out = type_name(error("x"))`, nil, "error")
	expectRun(t, `out = type_name(func(){})`, nil, "<compiled-function/0>")
	expectRun(t, `out = type_name(len)`, nil, "<builtin-function:len/1>")
	expectError(t, `type_name()`, nil, "wrong_num_arguments: (type_name) expected 1 argument(s), got 0")
}

// -----------------------------------------------------------------------------
// Spread operator edge cases (empty array, non-array, method-call combinations)
// -----------------------------------------------------------------------------

func TestSpread_EmptyArray_OnVariadic(t *testing.T) {
	expectRun(t, `f := func(...a) { return a }; out = f([]...)`, nil, ARR{})
	expectRun(t, `f := func(a, ...b) { return [a, b] }; out = f(1, []...)`, nil, ARR{1, ARR{}})
}

func TestSpread_EmptyArray_OnFixedArity(t *testing.T) {
	expectRun(t, `f := func() { return 42 }; out = f([]...)`, nil, 42)
	expectError(t, `f := func(a) { return a }; f([]...)`, nil,
		"wrong_num_arguments")
}

func TestSpread_NonArray(t *testing.T) {
	expectError(t, `f := func(a) { return a }; r := {a:1}; f(r...)`, nil,
		"invalid_argument_type: (...) argument spread expects type array, got record")
	expectError(t, `f := func(a) { return a }; s := "abc"; f(s...)`, nil,
		"invalid_argument_type: (...) argument spread expects type array, got string")
	expectError(t, `f := func(a) { return a }; n := 1; f(n...)`, nil,
		"invalid_argument_type: (...) argument spread expects type array, got int")
}

func TestSpread_MethodCall_EmptyArray_WrongArgsRaised(t *testing.T) {
	// for_each requires exactly 1 fn argument. An empty spread degrades to zero args.
	expectError(t, `[1,2].for_each([]...)`, nil,
		"wrong_num_arguments: (for_each)")
}

func TestSpread_MethodCall_NonArray(t *testing.T) {
	expectError(t, `[1,2].for_each({a:1}...)`, nil,
		"invalid_argument_type: (...) argument spread expects type array, got record")
}

// -----------------------------------------------------------------------------
// splice edge cases including the integer-overflow regression (delete count
// MaxInt64 used to crash the VM with "slice bounds out of range").
// -----------------------------------------------------------------------------

func TestSplice_HugeDeleteCountClamps(t *testing.T) {
	// Regression: large positive count must be clamped, not overflow startIdx+delCount.
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := splice(a, 2, 9223372036854775807)
		out = [a, d]
	`, nil, ARR{ARR{1, 2}, ARR{3, 4, 5}})
}

func TestSplice_HugeDeleteCountWithInsertClamps(t *testing.T) {
	expectRun(t, `
		a := [1, 2, 3, 4, 5]
		d := splice(a, 1, 9223372036854775807, "x", "y")
		out = [a, d]
	`, nil, ARR{ARR{1, "x", "y"}, ARR{2, 3, 4, 5}})
}

func TestSplice_NegativeStart(t *testing.T) {
	expectError(t, `splice([1,2,3], -1)`, nil,
		"index_out_of_bounds: (splice, start index)")
}

func TestSplice_StartBeyondLen(t *testing.T) {
	expectError(t, `splice([1,2,3], 4)`, nil,
		"index_out_of_bounds: (splice, start index)")
}

func TestSplice_NegativeCount_Recoverable(t *testing.T) {
	// Bug fix: negative-count error is now Recoverable so deferred recover() can catch it.
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			splice([1,2,3], 0, -1)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestSplice_OnConstArray_Errors(t *testing.T) {
	expectError(t, `splice(immutable([1,2,3]), 0)`, nil,
		"invalid_argument_type: (splice) argument first expects type mutable array")
}

// -----------------------------------------------------------------------------
// range edge cases
// -----------------------------------------------------------------------------

func TestRange_StepZero_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			range(0, 5, 0)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestRange_NegativeStep_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			range(0, 5, -1)
			return "not_rescued"
		}
		out = f()
	`, nil, "rescued")
}

func TestRange_WrongArity(t *testing.T) {
	expectError(t, `range()`, nil, "wrong_num_arguments: (range) expected 2 or 3")
	expectError(t, `range(1)`, nil, "wrong_num_arguments: (range) expected 2 or 3")
	expectError(t, `range(1,2,3,4)`, nil, "wrong_num_arguments: (range) expected 2 or 3")
}

func TestRange_NonIntArgs(t *testing.T) {
	expectError(t, `range("a", 1, 1)`, nil,
		"invalid_argument_type: (range) argument start expects type int")
	expectError(t, `range(0, "b", 1)`, nil,
		"invalid_argument_type: (range) argument stop expects type int")
	expectError(t, `range(0, 1, "c")`, nil,
		"invalid_argument_type: (range) argument step expects type int")
}

// -----------------------------------------------------------------------------
// Constructor builtins (string/int/float/byte/rune/bytes/runes/dict/...) that
// were partially covered. Cover the explicit "fallback default value" branch
// that returns args[1] when args[0] cannot be converted.
// -----------------------------------------------------------------------------

func TestConstructorFallback_Defaults(t *testing.T) {
	// Use values that are NOT convertible to the target type, so the fallback kicks in.
	expectRun(t, `out = int("nope", 42)`, nil, 42)
	expectRun(t, `out = float("nope", 1.5)`, nil, 1.5)
	expectRun(t, `out = string(len, "alt")`, nil, "alt")
}

func TestConstructorFallback_NoFallback_ReturnsUndefined(t *testing.T) {
	expectRun(t, `out = is_undefined(int("nope"))`, nil, true)
	expectRun(t, `out = is_undefined(float("nope"))`, nil, true)
}

func TestConstructorWrongArity(t *testing.T) {
	expectError(t, `int(1, 2, 3)`, nil, "wrong_num_arguments: (int) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `float(1, 2, 3)`, nil, "wrong_num_arguments: (float) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `bool(1, 2, 3)`, nil, "wrong_num_arguments: (bool) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `byte(1, 2, 3)`, nil, "wrong_num_arguments: (byte) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `rune(1, 2, 3)`, nil, "wrong_num_arguments: (rune) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `string(1, 2, 3)`, nil, "wrong_num_arguments: (string) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `runes(1, 2, 3)`, nil, "wrong_num_arguments: (runes) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `bytes(1, 2, 3)`, nil, "wrong_num_arguments: (bytes) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `decimal(1, 2, 3)`, nil, "wrong_num_arguments: (decimal) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `time(1, 2, 3)`, nil, "wrong_num_arguments: (time) expected 0, 1 or 2 argument(s), got 3")
	expectError(t, `dict(1, 2, 3)`, nil, "wrong_num_arguments: (dict) expected 0, 1 or 2 argument(s), got 3")
}

func TestBuiltinDict_FromInvalidType(t *testing.T) {
	expectError(t, `dict(123)`, nil,
		"invalid_argument_type: (dict) argument first expects type dict or record")
}

// -----------------------------------------------------------------------------
// error()/raise()/recover() edge cases
// -----------------------------------------------------------------------------

func TestError_FatalFlag(t *testing.T) {
	// error(payload, true) creates a fatal error which bypasses recover.
	expectError(t, `
		f := func() {
			defer func() { recover() }()
			raise(error("boom", true))
		}
		f()
	`, nil, "boom")
}

func TestError_RecoverableFlag(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			raise(error("boom", false))
		}
		out = f()
	`, nil, "rescued")
}

func TestError_WrongFlagType(t *testing.T) {
	// A builtin function value has no AsBool conversion -> triggers the type check.
	expectError(t, `error("x", len)`, nil,
		"invalid_argument_type: (error) argument second expects type bool")
}

func TestError_WrongArity(t *testing.T) {
	expectError(t, `error()`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 0")
	expectError(t, `error("a", true, "extra")`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 3")
}

func TestRaise_PayloadGetsWrapped(t *testing.T) {
	// raise of non-error wraps it.
	expectRun(t, `
		f := func() r {
			defer func() {
				e := recover()
				if is_error(e) { r = "wrapped" }
			}()
			raise("plain")
		}
		out = f()
	`, nil, "wrapped")
}

func TestRaise_FatalFlag_BypassesRecover(t *testing.T) {
	expectError(t, `
		f := func() {
			defer func() { recover() }()
			raise("boom", true)
		}
		f()
	`, nil, "boom")
}

func TestRaise_DemoteFatalFlagToRecoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			raise(error("boom", true), false) // demote
		}
		out = f()
	`, nil, "rescued")
}

func TestRaise_WrongArity(t *testing.T) {
	expectError(t, `raise()`, nil, "wrong_num_arguments: (raise) expected 1 or 2 argument(s), got 0")
	expectError(t, `raise("x", true, "extra")`, nil, "wrong_num_arguments: (raise) expected 1 or 2 argument(s), got 3")
}

func TestRaise_WrongFlagType(t *testing.T) {
	expectError(t, `raise("x", len)`, nil,
		"invalid_argument_type: (raise) argument second expects type bool")
}

func TestRecover_WrongArity(t *testing.T) {
	expectError(t, `func() { defer func() { recover(1) }(); raise("x") }()`, nil,
		"wrong_num_arguments: (recover) expected 0 argument(s), got 1")
}

// -----------------------------------------------------------------------------
// Defer execution corners
// -----------------------------------------------------------------------------

func TestDefer_DeepRecursionWithDefers(t *testing.T) {
	// Each call registers a defer; verifies that the deferred-call slice is correctly
	// reset on each frame across many levels and that recover-eligible frames don't
	// leak in-flight errors between calls.
	expectRun(t, `
		log := []
		f := func() {}
		walker := 0
		walker = func(n) {
			defer f()
			if n > 0 {
				walker(n-1)
			}
			log = append(log, n)
		}
		walker(20)
		out = len(log)
	`, nil, 21)
}

func TestDefer_LaterDeferRunsAfterEarlierRaisedAndRecovered(t *testing.T) {
	// First defer (LIFO last) raises. Earlier defer recovers it; the function returns normally.
	expectRun(t, `
		log := []
		f := func() r {
			defer func() {
				log = append(log, "defer1")
				e := recover()
				if e != undefined { log = append(log, "rescued") }
			}()
			defer func() {
				log = append(log, "defer2")
				raise("from-defer2")
			}()
			r = "ok"
		}
		_ = f()
		out = log
	`, nil, ARR{"defer2", "defer1", "rescued"})
}

func TestDefer_NestedFunctionCallRecoverFails(t *testing.T) {
	// recover() called from a helper INSIDE a defer must return undefined (Go parity).
	expectRun(t, `
		out = "untouched"
		f := func() {
			defer func() {
				helper := func() { return recover() }
				e := helper()
				if e == undefined { out = "no_recover_through_helper" }
			}()
			raise("err")
		}
		// f re-raises since helper.recover() returned undefined.
		// Wrap to swallow.
		g := func() {
			defer func() { recover() }()
			f()
		}
		g()
	`, nil, "no_recover_through_helper")
}

func TestDefer_VariadicDeferredFunction(t *testing.T) {
	expectRun(t, `
		log := []
		f := func(...args) { log = append(log, args) }
		g := func() {
			defer f(1, 2, 3)
		}
		g()
		out = log[0]
	`, nil, ARR{1, 2, 3})
}

// -----------------------------------------------------------------------------
// Tail-call and recursion safety with named results
// -----------------------------------------------------------------------------

func TestTailCall_DeepRecursionDoesNotOverflow(t *testing.T) {
	// 100k iterations: only TCO keeps this within DefaultMaxFrames.
	expectRun(t, `
		f := func(n) {
			if n == 0 { return "done" }
			return f(n-1)
		}
		out = f(100000)
	`, nil, "done")
}

func TestTailCall_DisabledWhenDefersPresent(t *testing.T) {
	// With a defer registered, TCO must be skipped — otherwise the defer slice
	// would leak across the recursive call, doubling-firing or losing entries.
	expectRun(t, `
		log := []
		f := 0
		f = func(n) {
			defer func() { log = append(log, n) }()
			if n == 0 { return }
			f(n-1)
		}
		f(3)
		out = log
	`, nil, ARR{0, 1, 2, 3})
}

// -----------------------------------------------------------------------------
// Closures & free variables: capture-by-reference and mutation through a defer
// -----------------------------------------------------------------------------

func TestClosure_DeferMutatesCapturedVariable(t *testing.T) {
	expectRun(t, `
		x := 1
		f := func() {
			defer func() { x = 99 }()
		}
		f()
		out = x
	`, nil, 99)
}

func TestClosure_NamedResultViaClosure(t *testing.T) {
	// Defer mutates named result through closure capture.
	expectRun(t, `
		f := func() r {
			r = 10
			defer func() { r = r * 2 }()
			return
		}
		out = f()
	`, nil, 20)
}

// -----------------------------------------------------------------------------
// VM.Call: host-side calls to Kavun functions through script-level callbacks
// -----------------------------------------------------------------------------

func TestHostCallback_CallScriptFunction(t *testing.T) {
	// A host-registered builtin that invokes a script function via VM.Call.
	caller := core.NewBuiltinFunctionValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			if len(args) != 2 {
				return core.Undefined, fmt.Errorf("invoke expects (fn, arg)")
			}
			fnVal := args[0]
			if fnVal.Type != core.VT_COMPILED_FUNCTION {
				return core.Undefined, fmt.Errorf("invoke: arg 1 not a function")
			}
			return v.Call((*core.CompiledFunction)(fnVal.Ptr), []core.Value{args[1]})
		}, 2, false)

	expectRun(t,
		`f := func(x) { return x * 3 }; out = invoke(f, 7)`,
		Opts().Symbol("invoke", caller).Skip2ndPass(),
		21)
}

func TestHostCallback_PropagatesRaisedError(t *testing.T) {
	// Errors raised by the script callback must bubble back through VM.Call to the host.
	caller := core.NewBuiltinFunctionValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call((*core.CompiledFunction)(fnVal.Ptr), nil)
		}, 1, false)

	expectError(t,
		`f := func() { raise("script-side") }; invoke(f)`,
		Opts().Symbol("invoke", caller).Skip2ndPass(),
		"script-side")
}

func TestHostCallback_RecoveredByOuterScript(t *testing.T) {
	// If the host-invoked script function defers a recover, the error must be
	// caught at the trampoline boundary and returned cleanly to the host.
	caller := core.NewBuiltinFunctionValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call((*core.CompiledFunction)(fnVal.Ptr), nil)
		}, 1, false)

	expectRun(t, `
		f := func() r {
			defer func() {
				e := recover()
				if e != undefined { r = "rescued" }
			}()
			raise("oops")
		}
		out = invoke(f)
	`, Opts().Symbol("invoke", caller).Skip2ndPass(),
		"rescued")
}

func TestHostCallback_VarargsAndArity(t *testing.T) {
	caller := core.NewBuiltinFunctionValue("invoke3",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call((*core.CompiledFunction)(fnVal.Ptr),
				[]core.Value{core.IntValue(1), core.IntValue(2), core.IntValue(3)})
		}, 1, false)

	// Variadic script function via host VM.Call.
	expectRun(t, `
		f := func(...xs) {
			s := 0
			for _, x in xs { s += x }
			return s
		}
		out = invoke3(f)
	`, Opts().Symbol("invoke3", caller).Skip2ndPass(), 6)

	// Wrong arity from host-side.
	wrong := core.NewBuiltinFunctionValue("invoke",
		func(v core.VM, args []core.Value) (core.Value, error) {
			fnVal := args[0]
			return v.Call((*core.CompiledFunction)(fnVal.Ptr), nil)
		}, 1, false)
	expectError(t, `f := func(a) { return a }; invoke(f)`,
		Opts().Symbol("invoke", wrong).Skip2ndPass(),
		"wrong_num_arguments: (call) expected 1 argument(s), got 0")
}

// -----------------------------------------------------------------------------
// Stack overflow scenarios
// -----------------------------------------------------------------------------

func TestStackOverflow_MutualRecursion(t *testing.T) {
	expectError(t, `
		f := 0
		g := 0
		f = func(n) { return g(n+1) }
		g = func(n) { return f(n+1) }
		f(0)
	`, nil, "stack_overflow")
}

func TestStackOverflow_HostCallback_RespectsFrameLimit(t *testing.T) {
	// Build a small VM with very few frames, then invoke a host-callback that
	// wants to call back into the VM. Eventually exhaust frames.
	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(8, 1024) // tiny frame stack

	var caller core.Value
	callerFn := func(v core.VM, args []core.Value) (core.Value, error) {
		if len(args) != 1 {
			return core.Undefined, fmt.Errorf("invoke needs 1 arg")
		}
		return v.Call((*core.CompiledFunction)(args[0].Ptr), []core.Value{args[0]})
	}
	caller = core.NewBuiltinFunctionValue("invoke", callerFn, 1, false)

	s := kavun.NewScript([]byte(`f := func(self) { return invoke(self) }; out = invoke(f)`))
	require.NoError(t, add(s, "out", nil))
	s.Add("invoke", caller)
	c, err := s.Compile(cta)
	require.NoError(t, err)
	err = c.Run(rta, machine)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "stack_overflow"),
		"expected stack_overflow, got %v", err)
}

// -----------------------------------------------------------------------------
// Iterator / not-iterable error paths
// -----------------------------------------------------------------------------

func TestIterator_OnNonIterable(t *testing.T) {
	expectError(t, `for x in 1 { _ = x }`, nil, "not_iterable")
	expectError(t, `for k, v in true { _ = k; _ = v }`, nil, "not_iterable")
}

// -----------------------------------------------------------------------------
// Format spec & f-string edge cases
// -----------------------------------------------------------------------------

func TestFormatDyn_BadSpec_Recoverable(t *testing.T) {
	// f"{x:{spec}}" with an invalid dynamic spec must produce a recoverable error.
	expectRun(t, `
		f := func() r {
			defer func() { e := recover(); if e != undefined { r = "rescued" } }()
			x := 42; spec := "@"
			_ = f"{x:{spec}}"
		}
		out = f()
	`, nil, "rescued")
}

func TestFormatDyn_NonStringSpec(t *testing.T) {
	// The dynamic-spec inner expression is always coerced to a string by the compiler
	// (via OpFormat with empty spec), so this guard is mostly defensive — verify that
	// even purely non-string-looking values produce a valid (or recoverable) result
	// rather than panicking. Numeric specs parse as width.
	expectRun(t, `x := 1; spec := 5; out = f"{x:{spec}}"`, nil, "    1")
}

func TestBuiltinFormat_TemplateModeMismatch(t *testing.T) {
	expectError(t, `format("{a}", [1])`, nil, "named placeholders but args is array")
	expectError(t, `format("{0}", {a:1})`, nil, "indexed placeholders but args is")
}

func TestBuiltinFormat_MissingKey(t *testing.T) {
	expectError(t, `format("{missing}", {a:1})`, nil, "missing key")
}

func TestBuiltinFormat_IndexOutOfRange(t *testing.T) {
	expectError(t, `format("{5}", [1])`, nil, "out of range")
}

func TestBuiltinFormat_BytesAsTemplate(t *testing.T) {
	expectRun(t, `out = format(bytes("hi {0}!"), ["world"])`, nil, "hi world!")
}

func TestBuiltinFormat_RunesAsTemplate(t *testing.T) {
	expectRun(t, `out = format(runes("hi {0}!"), ["world"])`, nil, "hi world!")
}

func TestBuiltinFormat_NonStringTemplate(t *testing.T) {
	expectError(t, `format(123, [])`, nil,
		"invalid_argument_type: (format) argument template expects type string")
}

func TestBuiltinFormat_WrongArity(t *testing.T) {
	expectError(t, `format("x")`, nil, "wrong_num_arguments: (format) expected 2")
}

// -----------------------------------------------------------------------------
// Record literal: dynamic key error path (OpRecord with non-string key).
// In practice only string-typed expressions are accepted as keys at compile time,
// but the runtime guard exists; ensure it is not silently bypassed.
// -----------------------------------------------------------------------------

func TestRecordLiteral_StringKey_OK(t *testing.T) {
	expectRun(t, `out = {"a": 1, "b": 2}`, nil, MAP{"a": 1, "b": 2})
}

// -----------------------------------------------------------------------------
// Division / arithmetic edge cases
// -----------------------------------------------------------------------------

func TestArith_DivisionByZero_Int(t *testing.T) {
	expectError(t, `1 / 0`, nil, "division_by_zero")
	expectError(t, `1 % 0`, nil, "division_by_zero")
}

func TestArith_DivisionByZero_Recoverable(t *testing.T) {
	expectRun(t, `
		f := func() r {
			defer func() { if recover() != undefined { r = "rescued" } }()
			_ = 1 / 0
			return "no"
		}
		out = f()
	`, nil, "rescued")
}

func TestArith_NegateMinInt_Wraps(t *testing.T) {
	// -MinInt64 wraps to MinInt64 (two's complement); document the behavior.
	expectRun(t, `
		min := -9223372036854775807 - 1
		out = -min == min
	`, nil, true)
}

func TestArith_BitwiseComplement_Int(t *testing.T) {
	expectRun(t, `out = ^0`, nil, -1)
	expectRun(t, `out = ^(-1)`, nil, 0)
}

// -----------------------------------------------------------------------------
// Not-callable / not-iterable / not-sliceable error paths
// -----------------------------------------------------------------------------

func TestNotCallable(t *testing.T) {
	expectError(t, `1()`, nil, "not_callable: type int is not callable")
	expectError(t, `({})()`, nil, "not_callable")
	expectError(t, `"x"()`, nil, "not_callable")
}

// -----------------------------------------------------------------------------
// Global vs. local selector assignment paths (OpSetSelGlobal/OpSetSelLocal/OpSetSelFree)
// -----------------------------------------------------------------------------

func TestSelectorAssign_GlobalRecord(t *testing.T) {
	expectRun(t, `
		g := {a: {b: 1}}
		g.a.b = 99
		out = g.a.b
	`, nil, 99)
}

func TestSelectorAssign_LocalRecord(t *testing.T) {
	expectRun(t, `
		f := func() {
			x := {a: {b: 1}}
			x.a.b = 99
			return x.a.b
		}
		out = f()
	`, nil, 99)
}

func TestSelectorAssign_FreeVar(t *testing.T) {
	expectRun(t, `
		f := func() {
			x := {a: {b: 1}}
			g := func() { x.a.b = 99 }
			g()
			return x.a.b
		}
		out = f()
	`, nil, 99)
}

// -----------------------------------------------------------------------------
// Spread on empty array combined with method-call (covers OpMethodCall spread+empty path)
// -----------------------------------------------------------------------------

func TestSpread_MethodCall_EmptyArray(t *testing.T) {
	// `arr.method(args...)` where args is an empty array — combined with a method
	// that accepts variable arity. dict has a `keys()` method that takes 0 args.
	expectRun(t, `
		d := dict({a:1, b:2})
		out = len(d.keys([]...))
	`, nil, 2)
}

// -----------------------------------------------------------------------------
// VM lifecycle: multiple Reset, abort, isStackEmpty, Clear
// -----------------------------------------------------------------------------

func TestVM_Abort_StopsExecution(t *testing.T) {
	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	c := compile(t, `for true { _ = 1 }`, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	var runErr error
	go func() {
		defer wg.Done()
		runErr = c.Run(rta, machine)
	}()
	time.Sleep(20 * time.Millisecond)
	machine.Abort()
	wg.Wait()
	// VM stopped cleanly via Abort: no error propagated.
	require.NoError(t, runErr)
	_ = cta
}

func TestVM_Clear_ZerosOutSlots(t *testing.T) {
	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	c := compile(t, `out = "ok"`, nil)
	require.NoError(t, c.Run(rta, machine))
	// Should not panic, should not leak references.
	machine.Clear()
	require.True(t, machine.IsStackEmpty())
	_ = cta
}

func TestVM_ReuseAfterAbort(t *testing.T) {
	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// 1: abort an infinite loop
	c1 := compile(t, `for true { _ = 1 }`, nil)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = c1.Run(rta, machine)
	}()
	time.Sleep(10 * time.Millisecond)
	machine.Abort()
	wg.Wait()

	// 2: reuse same VM for a fresh program — must not be poisoned.
	c2 := compile(t, `out = 7`, nil)
	require.NoError(t, c2.Run(rta, machine))
	compiledGet(t, c2, "out", int64(7))
	_ = cta
}

// -----------------------------------------------------------------------------
// kavunErrorWrap: errors raised from the script must implement Unwrap to a *errs.Error
// so that errors.Is(hostErr, errs.ErrXxx) works at the host boundary.
// -----------------------------------------------------------------------------

func TestHostErrorBoundary_ErrorsIsWorks(t *testing.T) {
	cta := core.NewArena(nil)
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	c := compile(t, `1 / 0`, nil)
	err := c.Run(rta, machine)
	require.Error(t, err)
	require.True(t, errors.Is(err, errs.ErrDivisionByZero),
		"expected errors.Is(err, ErrDivisionByZero), got: %v", err)
	_ = cta
}

// -----------------------------------------------------------------------------
// Context cancellation propagation
// -----------------------------------------------------------------------------

func TestRunContext_CancelMidExecution(t *testing.T) {
	rta := core.NewArena(nil)
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
	c := compile(t, `for true {}`, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	err := c.RunContext(ctx, rta, machine)
	require.Equal(t, context.Canceled, err)
}
