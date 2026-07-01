# Virtual Machine

Each bytecode instruction has fixed size: 8 bytes total.

- 1 byte opcode (Op)
- 1 byte operand (Op1)
- 2 byte operand (Op2)
- 4 byte operand (Op3)

## Limits and how to read them

Most limits below are bytecode encoding limits, not practical limits you normally run into while writing scripts.

For example, the array literal limit means you can write a very large number of elements directly in one inline initializer
(encoded in Op3, so up to 4294967295):
`x = [1, 2, 3, ...]`. It does not mean an array can only contain that many elements. Arrays can grow dynamically through
runtime operations, limited by available memory and VM stack/resources.

Likewise, bytecode jump limits apply within a single function body, not to the whole program. Reaching them usually
means a function has become extremely large. In that case, split the code into smaller functions/modules or move
application logic back to the host program. Kavun is designed as an embeddable scripting language; "embeddable" does not
mean the entire application should be implemented in Kavun.

## Architectural limits (instruction format / core runtime layout)

- Maximum number of opcodes is 256, values 0..255.

- Any index/address carried in Op3 is 32-bit unsigned at bytecode level: 0..4294967295.
  This applies to jump targets and most indexes (globals, locals, free vars, static pools, etc.).
- Any count carried in Op2 is 16-bit unsigned at bytecode level: 0..65535.
  This applies to argument counts, selector counts, closure free-var capture count, etc.
- Boolean/small-token fields carried in Op1 are 8-bit unsigned at bytecode level: 0..255.

Because static values are stored in separate typed pools (primitives, strings, bytes, runes, decimals, times,
format-specs, compiled functions), there is no single shared "constant slot" limit anymore.
Each pool is independently addressable through Op3 (32-bit).

Examples derived from operand widths:

- Direct call/defer argument count encoded in bytecode: up to 65535 (Op2).
- Assignment selector chain encoded in bytecode: up to 65535 selectors (Op2).
- Array literal element count encoded in bytecode: up to 4294967295 elements (Op3).
- Record literal encoded field count: up to 4294967295 fields (Op3), i.e. up to 2147483647 key/value pairs.
- Function body jump target index: up to 4294967295 instructions (Op3).

## Runtime resource limits

The default VM constructor values are:

- Stack slots = 2048
- Call frames = 1024
- Global slots when no globals provided = 1024

Embedders can choose a different stack size and frame limit with `vm.NewVM(maxFrames, maxStack)`, and can pass custom
global storage to `VM.Reset`.

These are runtime configuration defaults/guards, not bytecode architecture limits.

## Existing checks that are not architectural limits

- Function parameters are currently compiler-limited to 127 by an explicit check.
  This is not required by instruction encoding (NumParameters is stored as int in compiled functions).

## Practical notes

- On 64-bit Go builds, practical limits are usually memory and configured VM resources.
- On 32-bit Go builds, practical limits are lower because Go slice/map indexing uses int.
