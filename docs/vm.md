# Virtual Machine

Each bytecode instruction has fixed size: 8 bytes total.

- 1 byte opcode (Op)
- 1 byte operand (Op1)
- 2 byte operand (Op2)
- 4 byte operand (Op3)

## Limits

- Maximum number of opcodes is 256.
- One count limit, 255, applies to every count a program writes out: the parameters of one function (a
  variadic `...rest` counts as one), the literal arguments of one call — plain or member, deferred or not
  (`f(a, b, …)`, `x.name(a, b, …)`, `defer f(…)`) — and the names of one destructuring assignment
  (`a, b, … := x`). It fits `Op1`, the narrowest operand that carries one of these counts. More is a compile
  error.
- A spread argument (`f(xs...)`, `x.name(xs...)`) counts as one literal argument; its elements are not counted
  and are bounded only by memory. They are never expanded on the operand stack: a builtin, a member or a host
  callable receives them in a fresh argument slice, and a compiled function as its parameters, the rest in a fresh
  variadic array, so a spread behaves the same at every call depth. A spread the callee does not accept raises
  the callee's recoverable `wrong_num_arguments`. `defer` with a spread argument is a compile error.
- Maximum number of variables one closure captures is 65535 (`Op2` of `MakeClosure`). More is a compile error.
- The operand stack holds `vm.DefaultStackSize` (2048) values unless the host passes another size to `vm.NewVM`.
  Each function's maximum stack depth is computed at compile time and checked before the function runs — on
  entry for a call, before the first instruction for the main function — so a body that needs more (a literal
  with thousands of elements) fails with the fatal `stack_overflow: operand stack` before anything is pushed.
- An assignment target has no depth limit (`a.b[c].d… = x`): the chain is compiled as ordinary reads,
  left to right, and one `AssignProperty` or `AssignIndex` instruction performs the write.
- Maximum length of a count-driven sequence allocation is `4294967296` elements. It bounds the
  allocation a *count* asks for — `repeat(n)`, its `*` operator form, and `pad_start(n)` / `pad_end(n)`
  on `array`, `string`, `runes` and `bytes`, materializing a `range`'s elements (`array()`, `join()`,
  `for_each`, … — see the range page), `rand.generator`'s `bytes(n)` / `perm(n)` and a `re.pattern`'s
  replacement expansion — and a count past it raises a catchable
  `invalid_value` rather than panicking the host. It is not a limit on a sequence's length: a
  sequence grown by appending or concatenation is bounded only by memory.

## Bytecode format

`Bytecode.Encode(w, signer)` writes an envelope around the gob stream of the `Bytecode`; integers are
little-endian:

```text
"KVN" | tag length (u16) | tag | core major (u16) | core minor | host major | host minor | gob(Bytecode)
                               |<----------------------- signed region ---------------------------->|
```

- **tag** — the embedder's `vm.Signer` output over the signed region, at most `vm.MaxTagLen` bytes; length 0 means
  unsigned. The versions sit inside the signed region, so a changed version is a trust failure.
- **core version** — `vm.CoreVersion()`, the engine's. A *minor* bump is an addition to a frozen, append-only table
  (an opcode, a member id, a builtin-function slot, a type id, a static pool kind); a *major* bump is a change of
  meaning (an operand layout, a moved or reused id or slot, a `Bytecode`/`core.Static`/`core.CompiledFunction`
  field, the envelope or gob format). A behaviour fix bumps nothing.
- **host version** — `core.SetHostVersion`, the embedder's, by the same two rules for its own types, modules and
  grant policy. `0.0` means unset, which is legal only while no embedder type or module is registered.

`Bytecode.Decode(r, signer)` reads to EOF and checks, in this order, each refusal a distinct sentinel error for
`errors.Is`:

1. **trust** — with a signer, the tag must be present and verify (`vm.ErrUntrusted`); with `nil`, the tag is
   ignored.
2. **host identity** — embedder types or modules registered and the host version unset: `vm.ErrHostVersionUnset`
   (`Encode` refuses the same way).
3. **compatibility** — the core version, then the host version: accepted iff the major is equal and the file's
   minor is not above the engine's; a host version `0.0` is accepted everywhere, since such bytecode can reference
   only what the engine provides. Otherwise `vm.ErrIncompatible`, naming both versions — recompile from source.
   The pre-envelope format (`KVN002`) is refused the same way.
4. **decode** — the gob stream (`vm.ErrNotBytecode` for a foreign, truncated or corrupt stream).

Nothing else is checked: a loaded bytecode is **trusted or refused**, and trusted bytecode is our compiler's output,
valid by construction (the compiler's invariants — the static primitive pool holds only `undefined`, `bool`,
`byte`, `rune`, `int`, `float` and `date` — are pinned by tests, not re-checked at load). A party you do not trust
hands you source, not bytecode. Code is serialized only here: the static pool holds no `core.Value`, and the value
codec (`Value.EncodeBinary`) refuses every function and module value. The module grant is not part of the
envelope: `Bytecode.Imports` lists the builtin modules the code can import, and the VM checks it against its own
grant (`VM.SetAllowedModules`) at `Reset`, once per bytecode; `Run` answers a refusal before the first
instruction, so no import is checked while a script runs.

## Defaults

- Stack slots = 2048
- Call frames = 1024
- Global slots when no globals provided = 1024

Embedders can choose a different stack size and frame limit with `vm.NewVM(maxFrames, maxStack)`, and can pass custom
global storage to `VM.Reset`.

A call from Go back into the script — a callback member (`map`, `for_each`, `reduce`, …) calling its function, a
host builtin calling `VM.Call`, a deferred call — takes two frames: the callee's, and one beneath it whose only
instruction hands control back to Go. Nested callback members (`a.map(e => b.map(f))`, or a function that maps
with itself) therefore reach the frame limit at about half the depth of plain calls; running out raises a fatal
`stack_overflow: native callback frames`. The boundary bounds error handling the way a function call does: a
callback's own deferred `recover()` catches its own raise and the member continues; an error the callback does
not recover leaves the member as the member's error; `recover()` in a function that a deferred body calls through
a member answers `undefined`, as in any function that is not itself deferred.

## Static data pools

Compiled constants live in per-kind pools on `core.Static` (strings, decimals, format specs, compiled functions,
etc.), each referenced from bytecode by pool index (`Op3`). `NameLists` is one such pool: each entry is the ordered
list of LHS names for one destructuring-assignment statement (see `docs/language.md`), with `""` marking a `_`
position. It backs the `Unpack` opcode (`Op1` = number of positions, `Op3` = `NameLists` index), which pops the
right-hand side value and pushes that many results — by position for an array, by name (via the same `AccessIndex` hook
ordinary indexing uses) for a dict/record, filling `undefined` for a name with no matching key.

## Type ids

Every value carries a one-byte type id (`core/value`); `core.ValueTypes[256]` holds one descriptor per id. Bytecode
records type ids (in the static pool's primitives) and builtin-function slots, so moving either is a bytecode break.

| ids | group |
| --- | --- |
| 0–15 | data-only primitives: `undefined` 0 (must be zero), `bool`, `byte`, `rune`, `int`, `float`, `date`; the rest reserved. Contiguous from 0: only these go into the static pool as primitives |
| 16–31 | internal values and callables: value pointer, builtin function, builtin closure, compiled function, bound member, format spec |
| 32–47 | heap scalars: `string`, `decimal`, `time`, `error` |
| 48–79 | containers: `array`, `dict`, `record`, `range`, `bytes`, `runes`; the rest reserved for planned containers and vectors |
| 80–111 | iterators, each at its container's id + 32 |
| 112–143 | builtin module types |
| 144–191 | module-owned types (`fin.year_fraction`, the `fin.*` tables, …) |
| 192–255 | embedder types (`kavun.UserDefinedType` is 192) |

The language owns ids 0–191, embedders 192–255. Ids 112–191 are frozen: allocated alphabetically on first use,
then append-only, never renumbered. A builtin module's id is its type id (`ImportBuiltinModule`'s `Op3`); its
functions live in `core.BuiltinFunctions` at slot *type id* × `core.ModuleSlotSize` (128) + their index, and the
global builtins use slot base 0.
