# Virtual Machine

Each bytecode instruction has fixed size: 8 bytes total.

- 1 byte opcode (Op)
- 1 byte operand (Op1)
- 2 byte operand (Op2)
- 4 byte operand (Op3)

## Limits

- Maximum number of opcodes is 256.
- Maximum number of function parameters is 127.
- Maximum number of literal arguments in one member call (`x.name(a, b, …)`, `defer x.name(…)`) is 255: the
  member-call instructions carry the count in `Op1`, beside the member ID in `Op2` and the name's static string
  index in `Op3`. More is a compile error; a spread argument (`x.name(xs...)`) counts as one. A plain function
  call (`f(…)`) carries its count in `Op2` and is not limited this way.
- An assignment target has no depth limit (`a.b[c].d… = x`): the chain is compiled as ordinary reads,
  left to right, and one `AssignProperty` or `AssignIndex` instruction performs the write.
- Maximum length of a count-driven sequence allocation is `4294967296` elements. It bounds the
  allocation a *count* asks for — `repeat(n)`, its `*` operator form, and `pad_start(n)` / `pad_end(n)`
  on `array`, `string`, `runes` and `bytes`, and materializing a `range`'s elements (`array()`, `join()`,
  `for_each`, … — see the range page) — and a count past it raises a catchable
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
right-hand side value and pushes that many results — by position for an array, by name (via the same `AccessIndex` hook
ordinary indexing uses) for a dict/record, filling `undefined` for a name with no matching key.
