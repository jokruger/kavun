# Virtual Machine

Each instruction starts with a one-byte opcode. Operands are either one byte or two bytes wide.

## Limits and how to read them

Most limits below are bytecode encoding limits, not practical limits you normally run into while writing scripts.

For example, the array literal limit means you can write at most 65535 elements directly in one inline initializer:
`x = [1, 2, 3, ...]`. It does not mean an array can only contain 65535 elements. Arrays can grow dynamically through
runtime operations, limited by available memory and VM stack/resources.

Likewise, bytecode jump limits apply within a single function body, not to the whole program. Reaching them usually
means a function has become extremely large. In that case, split the code into smaller functions/modules or move
application logic back to the host program. Kavun is designed as an embeddable scripting language; "embeddable" does not
mean the entire application should be implemented in Kavun.

Limits:

- Maximum number of opcodes is 256, values 0..255.
- Maximum number of addressable constant slots is 65536. This includes ordinary constants, compiled functions, method
  names, format specs, and defer-method names. Compiler emits indexes before deduplication, so repeated constants can
  still hit this limit.
- Maximum number of addressable global variable slots is 65536. The default VM global storage is smaller unless the host
  passes a larger globals slice.
- Maximum number of local variables (per-function) is 256, including parameters and named results.
- Maximum number of captured free variables is 255.
- Maximum builtin function index space is 256, values 0..255.
- Maximum number of arguments used in a direct call is 255. Spread calls (`...`) can expand to more arguments at
  runtime, bounded by the VM stack.
- Maximum number of function parameters is 127.
- Maximum allowed length of an assignment selector chain is 255 selectors.
- Function body bytecode has no direct 65535-byte size cap. Jump instructions can only address positions up to 65535, so
  any function that needs to jump beyond that will fail to compile.
- Arrays can store any number of elements, but an array literal (`x = [1, 2, ...]`) can contain at most 65535 elements.
- Records and dicts can store any number of key/value pairs, but a record literal (`x = {a: 1, b: 2, ...}`) can contain
  at most 32767 pairs.

## Runtime resource limits

The default VM constructor values are:

- Stack slots = 2048
- Call frames = 1024
- Global slots when no globals provided = 1024

Embedders can choose a different stack size and frame limit with `vm.NewVM(maxFrames, maxStack)`, and can pass custom
global storage to `VM.Reset`.
