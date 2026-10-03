# PROPOSAL: cheaper native-to-script callbacks

Status: **proposal, not scheduled.** Written 2026-10-03 from measurements in `tmp/dispatchbench/`
(`reentry_test.go`, `prims_test.go`); the numbers below should move into `benchmarks/` as the first step of any
implementation. No behaviour change is proposed: callbacks keep their `defer`/`recover()` semantics, error
propagation and stack-overflow checks.

## 1. The finding, decomposed

Every callback-taking member (`map`, `flat_map`, `reduce`, `for_each`, `keep`/`remove`/`any`/`all`/`count` with a
predicate, `index(f)`, `sort(cmp)`, the `fin` root-finding and bracketing callbacks) calls the script function
through `Value.Call` → `ValueTypes[t].Call` → `compiledFunctionTypeCall` → the `core.VM` interface →
`(*vm.VM).Call`, which re-enters the interpreter with a trampoline frame. `a.map(f)` over eight ints with
`f := e => e + 1` costs about 566 ns, or 70 ns per element. The first write-up in `NOTES.md` called that "the
price of re-entering the VM"; the measurements say it is only a third of it:

| measurement (i7-9750H, go1.27.1) | ns | what it isolates |
| --- | ---: | --- |
| `vm.Call` from Go of `e => e` (LOAD_LOCAL, RETURN) | 32.7 | re-entry + 2 instructions |
| `vm.Call` from Go of `e => e + 1` (4 instructions) | 39.3 | +2 instructions = 6.6 ns, so ~3.3 ns per instruction |
| script-level `x = f(i)` per iteration, loop subtracted | 33 | OpCall + the same 4-instruction body + return + store |
| script-level `for e in a { x = f(e) }` per element | 78 | the hand-written equivalent of `map` |
| `a.map(f)` per element | 70 | the member: re-entry + body + loop + result allocation |

Reading the table:

- **Re-entry overhead is about 25 ns** (32.7 minus two instructions). An in-script `OpCall` plus `Return` is
  about 8 to 10 ns for the same frame work. The callback premium is therefore **about 15 ns per callback**.
- **The body is cheap**: the four instructions of `e => e + 1` are ~13 ns at ~3.3 ns each.
- **About 30 ns per element is the member's own loop**: `t2v`, the two-slot argument buffer, the `Value.Call`
  chain (three indirections), storing the result, and the share of the result array allocation (192 bytes per
  call of eight).
- **`map` already beats the hand-written loop** (70 vs 78 ns per element), because the for-in loop pays its own
  iterator and store instructions.

So the ceiling for this proposal is the 15 ns premium plus a few nanoseconds of indirection: roughly a 20 to 25
percent reduction of the per-element cost of callback-taking members, not a multiple. It is worth doing because
callback members are the idiomatic Kavun way to write loops (`docs/examples/*.kvn` use `map`, `reduce`, `keep`
far more than `for`), but it should not be sold as more than that.

## 2. Where the 25 ns go

`(*VM).Call` in `vm/vm.go`, per invocation:

1. Receiver type check and the `VarArgs` branch, then the argument-count check against `fn.NumParameters`.
2. Six saves of interpreter state (`ip`, `sp`, `curInsts`, `curFrame`, `framesIndex`, `err`).
3. Two overflow checks (frames, operand stack).
4. A **trampoline frame**: four field writes for a synthetic frame whose only instruction is `Suspend`, so that
   the callee's `Return` has somewhere to return to that then exits `run()`.
5. Push of the callee slot and the arguments.
6. Ten field writes to set up the callee frame, `initFrameLocals`, the named-result slot.
7. `runUntilSuspend`: enter `run()`, execute the body, `Return` pops into the trampoline (eight more writes and a
   `releaseFrameLocals`), dispatch `Suspend`, exit; then the `err`/`tryRecover` checks.
8. Result extraction, six restores, error swap.

Plus, before any of it, the three indirect calls of the `Value.Call` chain and the interface call into the VM.

Of these, 1 and 3 are per-call repetitions of facts that are constant for the whole member call (same function,
same arity, same frame need), 4 and the `Suspend` dispatch in 7 exist only to make `Return` exit `run()`, and the
indirections exist because `core` only knows the `VM` interface.

## 3. Proposal A — a prepared callback (contained change)

Resolve the callee once per member call and give the engines a direct, pre-validated entry point.

```go
// in core
type Callback interface {
	// Call runs the prepared function with args. args is read-only (docs/conventions.md).
	Call(args []Value) (Value, error)
}

type VM interface {
	// ... existing methods ...
	// PrepareCallback validates fn once for repeated invocation with exactly arity arguments: callable type,
	// arity match (variadic functions are rolled up per call as today), frame and stack need. The returned
	// Callback is valid for the duration of the current member call only.
	PrepareCallback(fn Value, arity int) (Callback, error)
}
```

`(*vm.VM).PrepareCallback` returns a small struct holding the `*CompiledFunction`, the arity, and the precomputed
stack need. For a `BuiltinFunction`/`BuiltinClosure` callee it returns a wrapper that calls the Go function
directly (today those also go through `Value.Call`, cheaply). Its `Call`:

- skips the type and arity checks (done once), keeps the two overflow checks (they depend on the current `sp`);
- pushes **no trampoline frame**. The callee frame carries a new flag, `hostReturn bool`. The `Return` handler,
  after popping the frame, checks the flag and returns from `run()` the way `Suspend` does, leaving the result in
  the callee slot. Both `unwind.go` sites that test `f.fn == callbackTrampolineFn` test the flag instead, so
  `recover()` stays bounded exactly where it is bounded today — at the frame the host called;
- is a method on `*VM`, reached through one interface call from the engine instead of three indirections plus
  one interface call.

Engines change one line each, hoisting the resolution out of the loop:

```go
cb, err := vm.PrepareCallback(args[0], 1)
if err != nil { return Undefined, err }
for i, e := range o.Elements {
	buf[0] = t2v(e)
	res, err := cb.Call(buf[:1])
	...
}
```

The arity switch `SeqMap` already performs (`f/1` or `f/2`) is where the `arity` argument comes from; engines that
accept either arity prepare for the one they detect, as now.

**What stays the same.** `defer`/`recover()` inside the callback (the bound is the flagged frame); error
propagation out of the callback into the member and from there into the script; the stack-overflow errors and
their text; `(*VM).Call` itself for host code and for `invokeDeferred`, which keep the trampoline path (cold).

**Expected saving.** Skipping the trampoline writes, the `Suspend` dispatch, the per-call validation, and two
indirections is estimated at 10 to 15 ns of the 25 ns re-entry, so `a.map(f)` per element from ~70 to ~55–60 ns.
`BenchmarkReentry` (to be moved into `benchmarks/`) measures it directly; `BenchmarkMemberCall/arr_map8` is the
end-to-end guard.

## 4. Proposal B — the body, through the existing superinstruction roadmap

The callback body `e => e + 1` compiles to `LOAD_LOCAL e; PUSH_INT 1; BINARY_OP +; RETURN`. `TODO.md` already
lists the fusions that shrink it: `LoadLocalReturn`, `PushIntReturn`/`ReturnConst`, fused integer binary
operations. They apply to all code, and for a callback they remove one or two of the four dispatches, about 3 to 7
ns per invocation on the numbers above. Nothing callback-specific is needed; this proposal only notes that the
callback case is one more beneficiary and that `BenchmarkReentry/plus_one` vs `identity` is the measurement that
will show it.

## 5. Considered and rejected

- **Inlining a lambda literal into a loop at compile time** (`a.map(e => e + 1)` rewritten to a `for`). Rejected:
  the receiver's type is unknown at compile time, and the member's semantics are the receiver's (string `map`
  enforces 1:1 symbol results, `flat_map` reads results like an add-side operand). Only a type-specialising
  optimizer could do this safely, and `map` already outruns the hand-written loop it would be rewritten to.
- **A callback-free iteration protocol** where the member returns a cursor the VM drives, so no re-entry occurs.
  Rejected for now: it reshapes every higher-order member and the VM's instruction set for a 15 ns premium.
- **Keeping the trampoline but pooling it.** The trampoline's cost is the writes and the extra dispatch, not an
  allocation (it already lives in the frame array); pooling saves nothing.

## 6. Tests

- `kavun_lifecycle_test.go` and the `defer`/`recover` cases in `kavun_test.go` pin the semantics Proposal A must
  preserve (recover inside a callback, errors escaping a callback, stack overflow inside a callback, abort during
  a callback).
- Add: a test that a callback's `recover()` cannot see the member's or the caller's in-flight error (the bound),
  pinned on the flagged frame as it is today on the trampoline.
- Move `BenchmarkReentry` and `BenchmarkReentrySplit` into `benchmarks/`.

## 7. Risks

- **Unwinder invariants.** `vm/unwind.go` identifies the host boundary by the trampoline function pointer in two
  places; a frame flag must replace both and nothing else may depend on the trampoline frame's presence. The
  `invokeDeferred` path keeps using the trampoline, so the two mechanisms coexist until it, too, is converted.
- **Interface growth.** `core.VM` gains one method; every embedder-side mock of it (tests) gains one method.
- **Over-selling.** The measured ceiling is a quarter of the per-element cost. If the first implementation shows
  less than ~8 ns saved per callback, the change should be reconsidered against its added mechanism.
