# Kavun value ownership & lifetime policy (draft)

## 1. Vocabulary and core invariant

A `core.Value` is a tagged handle. For refcounted types it holds a `refpool.Reference`; for static/primitive types it is self-contained. The pool gives every live, non-pinned reference a counter; `Pin` removes a reference from refcount tracking until the next `Arena.Reset`.

We classify every place a `Value` is stored as either:

- **Owning slot (+1)** — the slot is one of the owners that keeps the underlying object alive. The slot is responsible for one matching `Release` when it stops referring to the object.
- **Borrowed slot (+0)** — the slot points at an object kept alive by someone else for at least the duration of this borrow. The slot must not `Release`.

The **single core invariant** is:

> Every value stored in an owning slot was either (a) freshly produced by `arena.New…Value` (which returns +1) or (b) followed by an explicit `Retain`. Every owning slot, when overwritten or dropped, performs exactly one `Release` against the old contents.

`Pin` is a one-way escape hatch from this invariant: once a value is pinned, `Retain`/`Release` against it are no-ops, and the slot's owning/borrowed status no longer matters until arena reset.

For static/primitive types, `Retain`/`Release`/`Pin` are all no-ops, so the rules below are vacuously satisfied — they have cost only for pooled types (`string`, `decimal`, `time`, `array`, `bytes`, `runes`, `dict`, `record`, `error`, iterators, closures, etc.).

## 2. Slot classification

| Slot | Class | Notes |
|---|---|---|
| `vm.globals[i]` | **owning** | one owner per live global |
| `vm.stack[i]` for `i < sp` | **owning** | the stack holds +1 references between calls |
| frame locals (`stack[bp..bp+numLocals]`) | **owning** | same backing array as the stack |
| `freeVars[i]` (`*Value` indirection) | **owning** for the pointed-to slot; the `*Value` itself is pinned at creation | shared cell |
| Container element (`Array.Elements[i]`, `Record.Fields[k]`, `Dict.Entries[k]`, `Error.Payload`) | **pinned** | see §5 |
| Iterator's reference to its source | **owning** | iterator holds +1 on the source container |
| Native function argument slice `args []Value` | **borrowed** | see §3 |
| Native function return value | **owning** at the call site | see §3 |
| `frame.inFlightErr` / raised error in flight | **owning** | one owner during unwind |
| `defers[i].args` | **owning** | captured eagerly at defer time |
| Constants/statics (`v.Static == true`) | **N/A** | refcount ops are no-ops |

## 3. Native functions (builtins, stdlib, type hooks: `Method`, `BinaryOp`, `Access`, etc.)

**Arguments are borrowed.** The caller (VM) keeps owning the values for the duration of the call. The callee:

- **MUST NOT** call `Release` on any argument.
- **MUST NOT** call `Retain` unless it needs to extend the value's lifetime beyond the call (i.e. it stores a copy of the handle somewhere outside the call frame).
- May freely call `Resolve…` and read the underlying data.

**The return value is +1 owned** by the caller. The four legal ways to produce the return:

1. **Fresh allocation** (`arena.NewXxxValue(...)`) — already +1; return as-is.
2. **Returning a primitive / static value** — no ref ops needed.
3. **Returning one of the arguments unchanged** — `arg.Retain(a)` then return. The returned value must be independent of the borrowed argument lifetime.
4. **Returning a value pulled out of a container** (e.g. `array[i]`, `dict["k"]`, iterator `Value()`) — the container holds it as **pinned** (§5), so the value cannot be reclaimed before the next arena reset. Returning it as-is is safe; the caller treats it as +1 and may `Release` (no-op for pinned). No `Retain` is required.

**Storing an argument into a container** (e.g. `array.append(x)`, `dict.set("k", v)`, building an error payload): call `v.Pin(a)` before placing it inside the container. This is the price/benefit trade-off discussed in §5 — the function does **not** need to Retain, the container does **not** track refs, and the caller's ownership is unaffected.

**On error return:** if the function allocated intermediate values that will not be returned, it must `Release` them. The returned `error` is itself a `Value` and follows rule (1) above.

## 4. VM operand stack

Treat every slot below `sp` as +1 owned. From this, all opcode rules fall out mechanically:

### 4.1 Pushes

- **Push from an owning cell** (`GetGlobal`, `GetLocal`, `GetFree`, `GetBuiltin` for variable refs, `GetConst` when the const is non-static): `v.Retain(a)`, then write to `stack[sp]`, `sp++`.
- **Push from a static/primitive constant** (`GetConst` with `Static==true`, `True`/`False`/`Null`, immediate int/float/etc.): no Retain.
- **Push a freshly allocated value** (`Array`, `Record`, `BinaryOp`, builtin/native call result, `Iterator`, fstring result, ...): no Retain — `New…Value` already returned +1.

### 4.2 Pops

- **Pop into an owning cell** (`SetGlobal`, `SetLocal`, `DefineLocal`, `SetFree`): this is a **move**. `target.Release(a)` (release old contents), `target = stack[--sp]`, do **not** Release the source slot.
  - For `SetFree`/free variables: the slot pointed to by `*ValuePtr` is the owning cell; same rule.
- **Pop and discard** (`Pop`, conditional jump that drops the test value, error-path cleanup, abnormal frame teardown): `stack[--sp].Release(a)`.
- **Pop into a container** (`Array`, `Record`, `Dict` literal opcodes, `Append`, `SetIndex`): **Pin** the popped value as it enters the container (§5), then `sp--`. No Release of the popped slot (Pin is in lieu of refcount management).

### 4.3 Overwrites mid-stack

`SetLocal`/`DefineLocal`/named-result write into a slot below `sp`. Always `old.Release(a)` before assignment, even if the new value is a primitive — the old value might be pooled.

This is what `initFrameLocals` exists for: stale slots from previous frames must be cleared to `Undefined` (a no-op `Release`) before any `DefineLocal` runs against them.

### 4.4 Calls (compiled function callee)

Arguments live on the stack at `stack[sp-N..sp]` as +1 owned, by §4.1. On entry the VM reframes so those slots become locals 0..N-1 of the callee. **This is a move, not a borrow** — no Retain on entry, no Release on exit. The callee's `Return` opcode:

- Reads the result (+1 from §4.1/§4.3).
- For each local slot still owning a value (locals `0..numLocals-1` of the popped frame): `Release`. These are overwritten on the next call's `initFrameLocals` regardless; releasing on return makes refcount reuse possible immediately rather than at frame reuse. This is implemented as `frame.releaseLocals(a)` invoked from `OpReturn` (and from the unwinder when a frame is skipped); it unconditionally Releases every slot in `bp..bp+numLocals`, relying on `Undefined` being a no-op Release for any slot already cleared.
- Places the result in the caller's slot of the callee value (which is itself an owning cell — `callee.Release(a)` then write the result).

### 4.5 Calls (native callee — builtin / type hook / closure)

Arguments at `stack[sp-N..sp]` are **borrowed** by the callee (§3). On return:

- Release each of the N argument slots (the +1 ownership stays in the stack slots through the call, then the slots collapse).
- Release the callee value slot.
- Move the returned value into the slot previously holding the callee.

### 4.6 Spread args, variadic packing, defer capture

The variadic packing code in `OpCall` collects tail args into a fresh array. Per §5, packing a Value into a container Pins it. The packed-array Value itself is +1 from `NewArrayValue` and replaces one stack slot — the slots it pinned are then dropped by `sp -= varArgs` with no further Release (Pin already covers them).

Spread (`opcode.Call` with `spread==1`): the spread array slot is popped (`Release`), its elements are pushed onto the stack. Element values come from a container (pinned) — pushing them onto the stack without Retain is consistent with §3 rule 4: they are pinned, so the stack slot effectively owns nothing reclaimable, and any later Release is a no-op.

`OpDefer`/`OpDeferMethod` capture args into an arena-allocated slice. This slice is **owning** (+1 per element) — the defer queue takes ownership from the stack by moving (no Retain, no Release of the stack source). When the defer runs, the captured args feed the call as borrowed args per §4.5, and the defer slot Releases them after.

### 4.7 Errors and unwinding

A raised error (`OpRaise`, runtime panic, native function error return) starts as a +1 owned value. The unwinder transfers ownership through `frame.inFlightErr`. When `recover()` reads it, the value moves to the recovering frame's stack (no Retain, no Release). If the error escapes `Run()`, the embedder receives it and must arrange for arena reset (or explicit `Release`) when done.

During unwind, the unwinder must `Release` every still-live owning slot in skipped frames (stack slots between `bp` and the frame's top of stack, and the callee value below). Without this, a recovered error means the skipped frames' pool entries leak until arena reset — correct but wasteful.

## 5. Containers: the Pin trade-off

Arrays, dicts, records, and error payloads hold their elements as **pinned** values:

- Inserting a value: `v.Pin(a)` before storing.
- Reading a value: return as-is; the value is pinned so it cannot be reclaimed before arena reset.
- Removing a value (delete, slice that drops elements): no Release — Pin is one-way.
- The container itself is a normal refcounted value.

Trade-off: an element that enters a container can no longer be reused via the free-list. The container hot path stays branchless (no traversal during Release of the container itself); reuse still applies to the bulk of allocations, which live on the stack, in locals, and as short-lived temporaries from binary ops.

When this proves too wasteful (e.g. very long-lived dicts in a service script), the policy allows a future per-container "owned" mode (`Retain` on insert / `Release` on remove or container destruction), but that is a follow-up — not part of the initial policy.

## 5a. The Pin escape hatch

In code paths that are exceptional (error handling, unwinding, defer rollback, recover) or where the lifetime of a value is hard to predict, **prefer `Pin` over best-effort Retain/Release**. Pin is always safe — it cannot cause double-free or use-after-free — and costs nothing on the hot path because hot paths shouldn't reach it. The price is that pinned values leak until the next arena reset, which is acceptable for cold paths.

Use Retain/Release only where ownership is clear and the path is hot enough that pool free-list reuse matters. When in doubt, Pin.

Concrete examples of "use Pin":

- Stack slots that the unwinder is about to abandon in a skipped frame after a recovered error.
- A value captured into a defer queue when the defer's eventual execution path is conditional.
- Native code that allocates several intermediates and may fail at various points — pin the intermediates instead of tracking which to Release on each error path.

## 6. VM lifecycle

- **VM.Reset(arena, bytecode, globals)**: caller is responsible for the arena state. The VM does **not** Release globals/stack before resetting state. Document that the recommended pattern is `arena.Reset()` followed by `vm.Reset(...)` between runs.
- **VM.Clear**: drops Go references to help GC; does not Release pool refs (arena reset is the only correct way).
- **arena.Reset** in the middle of execution invalidates **all** live references everywhere — globals must be reseeded from a fresh allocation if used in the next run.

## 7. Compiler vs VM split

**Default policy: rules above are enforced inside the VM and inside native functions**, not by the compiler. The rules are local per opcode and need no flow analysis, so the VM owns correctness end-to-end.

**Compiler-level optimisations are layered on top** as new opcodes whenever a static analysis proves the VM is doing redundant work. Each must be optional — removing it must never break correctness.

Initial candidates worth a separate opcode (track as follow-up work, not part of the policy itself):

| Optimisation | Replaces | Win |
|---|---|---|
| `MoveLocal` (move-out on last use) | `GetLocal` + skip Retain; source slot then becomes `Undefined` and is not Released later | save one Retain/Release pair per last-use load |
| `PopDrop` (drop value the compiler proved is +0) | `Pop` | save one Release |
| `GetConstStatic` (compile-time-known static) | `GetConst` with runtime branch on `Static` | save the branch |
| `CallTailMove` | `Call` for compiled callees in tail position | already partially present; extend to skip per-local Release of locals known dead |

Until these exist, the VM follows §4 unconditionally.
