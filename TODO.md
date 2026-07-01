# TODO list for Kavun

- static analyzer:
  - check all opcodes are valid
  - check all jumps are valid (address is within bytecode)
- No need to check if opcode is valid in VM - it is already checked by static analyzer
- Use unsafe for vm.ip so no bounds check on each opcode fetch

- range form:
  - f..t
  - f..t:s with step s
  - exact range(...) semantics - f inclusive, t exclusive
  - expression operands allowed, with optional constant folding
    - if expressions/variables are used, then generate builtin range() call
    - if only constants are use, then generate static value and corresponding opcode

- ast optimization - detect expressions which are using only constants and builtin primitives like int(), byte(), etc - calculate in compile time and store single static cons instruction!

- byte opcode
- short rune opcode (2 bytes)
- rune opcode (4 bytes)
- int1 opcode (1 byte), int2 opcode (2 bytes), int4 opcode (4 bytes), int8 opcode (8 bytes)
- float opcode (8 bytes)

- static primitives can be stored as bytecode (opcode + 4 bytes data)

- composite opcodes - some common structures/patterns (loops, calls, assign-inc, etc) are implemented as multiple opcodes - we can implement them as single opcode

- add "reuse" flag to hooks which return value

- SeqIterNextHook, SeqIterKeyHook, etc, and any generics receiving resolve callback can be changed to generic type Target and (*Target)(v.Ptr) directly!

- hooks which return value - accept flag indication that current value can be reused (so we can avoid some allocation) - in future compiler can detect when it can use this!

- NOTE!: do we actually need to do Retain/Release when copy to stack? Think about it. We should call it only when we truly create persistent copy - stack in most cases is temporary. Analyze it in details.

- use pool for low level slices (bytes, runes, arrays)

- enforce value management policy:
  - arguments passed with no ownership transfer:
    - function calls pin if it stores argument to container (i.e. retain/release will not be called properly anymore)
    - function calls retain if creates copy of argument and takes ownership of it
    - function calls release for previously owned value if needed
    - caller calls release after the function call if it passed newly created value as argument
  - values returned from functions with ownership transfer:
    - caller calls release if it does not need returned value anymore
  - vm calls release for values taken from stack if it decrements sp
  - vm calls release for values on stack if it overwrites them
  - in vm check all helper functions which may return core.Value - check policy!

- compiler - ensure we are deduping statics on a fly, and we check the max number of each static type (65536 - 2 bytes for index)
- review vm/unwind/etc - each time we modify stack, decide if we need to call value retain/release/pin, etc
- review all functions which may require Pin (assign, split, partition, map, filter, etc - where new values are created and stored in containers)
- review all functions where temporary values are created (filter, count, map, reduce, etc) ensure they are released
- review stdlib on var management policy

- review how arguments are passed to variadic functions - currently we create new array, so shell we pin values in it?

-  ensure we write some new value to stack each time we increment it

- opcode to load static primitives => Static.Primitives[i]
- opcode to load static decimal => DecimalValue, ref points to Static.Decimals[i], static = true
- opcode to load static strings => StringValue, ref points to Static.Strings[i], static = true
- opcode to load static runes => RunesValue, ref points to Static.Runes[i], static = true
- opcode to load static format specs => FormatSpecValue, ref points to Static.FormatSpecs[i], static = true
- opcode to load static compiled functions => CompiledFunctionValue, ref points to Static.CompiledFunctions[i], static = true

- review / rename opcodes
- replace constants with typed constant primitives and corresponding opcodes which load constant primitives on stack (i.e. build Values from primitives dynamically)

- revisit use of ToImmutable - shell we call Clone? or shell we do Retain?
- on stack increment we must ensure we are writing new value to the stack
- on stack decrement if corresponding ref is not 0 we should release it and set to 0!
- vm.Clear is not needed
- when overwriting value on stack, release old ref, decide on new ref (is it copy? should call Retain?)
- when overwriting global/local/const, release old ref, decide on new ref (is it copy? should call Retain?)
- when storing value to map/array/etc, pin new ref
- on vm reset ensure there is no old ref left in globals/locals/const/stack/etc
- data type Copy => Clone, review usage - the call should always create new value, the caller itself decides on immutable and does a logical copy if needed (i.e. Retain)
- vm.raisedError.Error - returns "error" if payload is not Error - shell we return "error: " + payload.String ?

- builtin types, modules and functions:
  - IDs must be FIRST..LAST..[RESERVED]..[USER_RESERVED], expose first user reserved
  - so the system and user IDs are stable even if new system added
  - API to add user defined

- find a common solution for static (const) and dynamic memory:
  - primitives resolved on opcode level (load const = get preassembled Value from consts)
  - complex types resolved on refpool level (.Resolve) - decide if it from pool or const mem
- migrate to refpool
- improve refcounting and Retain/Release usage

- review all encoders/decoders - store length as uint32
- why bytecode stores main function as pointer?

- sync documentation with new design
- ensure it is documented that if VM.Clear was used, caller must also call Reset before next run!
- document mem management policy - receiving logic (functions) should decide if retain/pin is needed

- shell we release values on stack when Clear is called?

- validate changes to stack pointer when we got error in vm (sp must always be updated same as in success case)

- why we allocate globals as static size array? is it changing during execution? can we make it slice - exactly the required size?

- add test for bytecode serialization - compile complicated script with all types of constants / statics, serialize bytecode, deserialize

- document that if Arena is shared between compiled scripts, before resolving values you must to call Attach to ensure the correct static segment is used for resolving static values (which is script specific)

- check type conversion: string(["a", "b", "c"]) and ["a", "b", "c"].string()

- now primitives are easy to distinguish, so we can have fast path in equal for instance (no call to hook, just compare data)

- let compiler to decide when check for "abort" flag - i.e. add opcode, emit it in loops / recursions ?

- control allowed modules on VM level!!! required for security, so we can allow bytecode execution but disallow some modules!

- piping and flow (`x |> f1(_) |> f2(y, _) ...`)
  - builtin type member functions allow write nice calc pipes, but user defined functions still will require nesting
  - idea is to be able describe a pipe where prev call result is passed as an argument to next call in pipe
  - ideally when describing next function we should be able define the argument to which the prev result is passed, and define other args

- destructuring: `x, y, z = [1, 2, 3]; {a, c} = {a: 1, b: 2, c: 3}`

- type as data + extension methods:
  - array.foo => call array static method
  - array.sum = foo => override array type method (globally)
  - array.myfoo = foo => extend array type with new method (globally)

- add to desc "written in pre Go, no CGo"

- compiler - find a way to analyze expressions and generate a code which does not require new variables on each binary op and can reuse existing.
  - we may need to change interface of hooks so instead of returning value thay will have a receiver as argument, so compiler can decide if new var is needed

- builtins are stored as a map, but max num of builtin functions is 256, so we can use array!
- check if vm limits are enforced (globals, etc)
- knowing vm limits (max nums / sizes), what can be optimized? (i.e. we could potentially use some preallocs, etc)?
- inspect all panics - return errors
- can we de-dupe constants in same time we emit them?

- need a stable dict iterations / map / tostr / etc

- add Hash function for Value (and all types). For ptr based values hash can be cached in .Data, use it in comparison

- refactor core/tools.go , looks like coerceSepToString, coerceSepToBytes, etc can be replaced with .AsString, etc?
- refactor member functions - in many cases we can have generic implementation used from concrete types

- make sure you cannot crash VM from script: limit num of allocs, total size of containers and mem used, catch panics
- for arrays, bytes, runes, strings - store data=leng and ptr=underlying data (&[0] / StringData, etc) to avoid allocation of header struct
- use store underlying array/dict pinter in Value.Ptr instead of using wrapper struct
- try use unsafe.StringData / unsafe.String to store and rebuild strings?
- do atomic load check for "abort" flag every X cycles, not every cycle
- for int/float/string/etc args, fast path for specific types, only then call .AsX()
- string - make it unicode indexed (slice, index and member function work with unicode by iterating! - note on performance in docs)
- runes.trim - custom implementation that uses runes slice from allocator

- array, string, bytes - multi-index get: array[1, 3, 5], or array[x] where x is array of ints
- ... and multi-index set: array[1, 3, 5] = [10, 30, 50]

- vector types: bytes, ints, floats

- new type Tuple.
  - dict/record to array of tuples
  - dict/record from array of tuples

- function property "arity" and "variadic"
- migrate to crypto/rand
- Move strings package functions to the string type member functions
- typed vectors, J core operators
- add Set data type, set specific operations
- merge(r1, r2) → new record, dict.merge
- optimization for "modify and assign" pattern (reuse variable, pass argument to inform type logic)
- array.append (array) => new array
- array.extend (array) => inplace
- fold(f, init) → value (same as reduce-with-init; pick one name)
- array.sort(lambda(a, b) => bool)
- window(n, step=1) → array[array]
- (! first need tuple type) zip(other) → array[tuple] (or array[array] of len 2); unzip ???
- enumerate() → array[(index, value)] (or dict-like pairs)
- string replace(old, new), startsWith, endsWith
- bytes.hex()
- bytes.base64()
- move type related functions to type member functions; remove duplicates from stdlib (i.e. stdlib must be complimentary extension of type member functions)
- Arrays: `sort_by`
- Strings: `has_prefix`, `has_suffix`
- Int/Float: `abs`, `pow`, `is_zero`
- add time.is_leap_year(), time.is_weekend(), time.is_weekday(), time.is_holiday() (with holiday calendar)
- rune - implement methods from <https://pkg.go.dev/unicode>
- missing ctors(0/1/2): array, record
- range methods: dict, filter, reduce, sum, etc (mirror array methods)
- generic range (just like int range but use Value for start/stop/step) - to be used for time, float, etc ranges as well
- splice - use AsArray
- move splice function to container types (methods)
- in VM slice logic, use fast path for Int
- vector/array operations like /+, /-, /\*, etc - elementwise operations for vectors
- format for decimal
- sign member function for int/float/decimal
- abs member function for int/float/decimal
- pow member function for int/float/decimal
- sqrt member function for int/float/decimal
- type() member function for all types, returning type name as string
- container types: .reverse(), .shuffle(), .unique(), .chunk(size), .window(size, step), .enumerate()
- remove dict/record to string conversion - it breaks consistency... complex values should be printed, not converted to string implicitly
- add flag to `immutable` function to do a deep immutability (for arrays/dicts/records) - so all nested structures will be immutable as well
- go style switch with multi-value cases, default, etc

- Array.fill(n, val)`/`Array.fill(n, fn)
- array.intersperse(x)
- array.cycle(n)
- string.pad_left(n, ch)`/`pad_right`/`center
- array.take(n)`/`drop(n)
- array.push/pop,insert

<<<<<<

- string/rune/bytes/array \* int => repeat n times; need to be in sync with global vectorization strategy
- implement hashing for each data type, optimize "dedupe / unique / equal" using hash
- compile time tail call optimization - runtime vm should not be smart, just a stupid loop over switch cases, all decisions should be made at compile time
- inlining and other optimizations
- builtin max/min
- find a way to reuse value envelopes: receiver ptr instead of return value, mark as tmp, on assign copy if tmp, etc - primary usecase = loops
- how to use string value or envelope ptr in map keys, so we can use them when iterating over keys (instead of creating new strings)
- builtin cron support (expressions, next event, etc)
- shell we rename fmt to io ?
- input functions - console input, key, etc
- builtin memoization (for functions)
- use caches for runtime parsing, etc (use cache package with controlled cache size)
- for range var {}
- builtin regex type

- for in range; for range
- array(), array(n) constructor
- types ctors should return error value instead of raising an error (so user code can react)
- all types should have conertor functions to all other types - return err object if it is impossible
- optional static types - does not allow reassign to other types, fail function calls, etc

- cheat-sheet page

- refactor error system
- review all functions returning errors - decide: shell it raise error or return an error object

- builtin logging

- why .byte(), .string(), .decimal(), etc convert without checking for error?

!!! check vm.go, "case opcode.CallFunction" and "case opcode.CallMethod"
it looks like we first put spread args to the stack (and can overflow) but then
immediately reshape it to collapse the tail args into variadic (a single array arg).
It should be possible to avoid temp copying to stack !

<<<<<

## Performance optimizations enabled by precise MaxStack

Now that each `CompiledFunction` knows its exact peak operand-stack height, several optimizations become possible:

1. **Per-frame stack allocation** — currently the VM has one giant shared `stack []Value`. With MaxStack known per function, each frame could carry its own slice (or use a bump allocator), improving cache locality and enabling parallel call stacks for goroutine-style features.

2. **Tighter default stack size** — the default `stackSize` heuristic could shrink for small scripts. Programs that statically never recurse can use exactly `sum(MaxStack)` for the call chain.

3. **Drop residual safety branches** — any leftover defensive stack checks in hot paths (e.g., the OpCall guard itself can become a debug-only assert in release builds, since the compiler proves the invariant).

4. **Smarter inlining** — small callees whose MaxStack + caller's current height fits without growth become candidates for bytecode-level inlining (no new frame, no OpCall overhead).

5. **Disassembler/profiler surface** — expose MaxStack and NumLocals in disassembly so users can spot deep-evaluation hotspots.

6. **Stack pre-touching / zeroing only the needed range** — `Reset()` and frame entry only need to clear `NumLocals+MaxStack` slots, not the whole stack.

7. **Specialized tiny-frame VMs** — for leaf functions with MaxStack ≤ a small N (say 4), a register-style fast dispatch could be generated.
