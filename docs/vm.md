# Virtual Machine

Each bytecode instruction has fixed size: 8 bytes total.

- 1 byte opcode (Op)
- 1 byte operand (Op1)
- 2 byte operand (Op2)
- 4 byte operand (Op3)

## Limits

- Maximum number of opcodes is 256.
- Maximum number of function parameters is 127.
- Maximum selector depth in one assignment target is 8 (`a.b[c].d… = x`): the `StoreIndexed*`
  instructions carry each selector's spelling (dot vs bracket) as a bitmask in `Op1`, so the runtime
  can distinguish selector access from index access on the write path exactly as on the read path.
  Deeper chains are a compile error; reads are unlimited.
- Maximum length of a count-driven sequence allocation is `4294967296` elements. It bounds the
  allocation a *count* asks for — `repeat(n)`, its `*` operator form, and `pad_start(n)` / `pad_end(n)`
  on `array`, `string`, `runes` and `bytes` — and a count past it raises a catchable
  `invalid_value` rather than panicking the host. It is not a limit on a sequence's length: a
  sequence grown by appending or concatenation is bounded only by memory.

## Defaults

- Stack slots = 2048
- Call frames = 1024
- Global slots when no globals provided = 1024

Embedders can choose a different stack size and frame limit with `vm.NewVM(maxFrames, maxStack)`, and can pass custom
global storage to `VM.Reset`.

## Static data pools

Compiled constants live in per-kind pools on `core.Static` (strings, decimals, format specs, compiled functions,
etc.), each referenced from bytecode by pool index (`Op3`). `NameLists` is one such pool: each entry is the ordered
list of LHS names for one destructuring-assignment statement (see `docs/language.md`), with `""` marking a `_`
position. It backs the `Unpack` opcode (`Op1` = number of positions, `Op3` = `NameLists` index), which pops the
right-hand side value and pushes that many results — by position for an array, by name (via the same `Access` hook
ordinary indexing uses) for a dict/record, filling `undefined` for a name with no matching key.
