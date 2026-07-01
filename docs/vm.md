# Virtual Machine

Each bytecode instruction has fixed size: 8 bytes total.

- 1 byte opcode (Op)
- 1 byte operand (Op1)
- 2 byte operand (Op2)
- 4 byte operand (Op3)

## Limits

- Maximum number of opcodes is 256.
- Maximum number of function parameters is 127.

## Defaults

- Stack slots = 2048
- Call frames = 1024
- Global slots when no globals provided = 1024

Embedders can choose a different stack size and frame limit with `vm.NewVM(maxFrames, maxStack)`, and can pass custom
global storage to `VM.Reset`.
