# TODO: preparation for refpool migration

- use of types (must be allocated in arena only - no direct allocations):
  - value ptr ???
  - arena only allocated: 
    - error
    - decimal
    - string
    - bytes
    - runes
    - array
    - dict
    - record
    - int range
    - array iterator
    - dict iterator
    - int range iterator
    - runes iterator
    - bytes iterator
    - builtin closure
    - compiled function
    - format spec

- review / rename opcodes
- review arena allocated values - check what is used by compiler (ensure only basic types are used)
  - split compiled function into two (same as builtin functions) - function and closure, closure is dynamically allocated
- remove static constructors (except basic types) - all complex types should be created using arena (even if it is just wrapper around heap)
- replace constants with typed constant primitives and corresponding opcodes which load constant primitives on stack (i.e. build Values from primitives dynamically)

- review how arena is used during compile time
  - it looks like we need to distinguish compile time and runtime arenas!
  - because in compile time we allocate some values, but the in runtime we may reset arena
  - so, need to separate compile and run time values and use different arenas for them!!!!
  - can we use a flag in Value for this; or maybe force clone all compile time values to runtime? - need separate function for this - move between arenas!!!
    => or modify refpool so it is a multi-pool, and pool index is part of Reference..

- require same arena for compile and run - change docs
  - new compiler => error if arena is nil (there is no default arena anymore)
- Complex types static ctors - remove because in case of refpool they can be created only in arena!
- revisit use of ToImmutable - shell we call Clone? or shell we do Retain?
- on stack increment we must ensure we are writing new value to the stack
- on stack decrement if corresponding ref is not 0 we should release it and set to 0!
- vm.Clear is not needed
- when overwriting value on stack, release old ref, decide on new ref (is it copy? should call Retain?)
- when overwriting global/local/const, release old ref, decide on new ref (is it copy? should call Retain?)
- when storing value to map/array/etc, pin new ref
- on vm reset ensure there is no old ref left in globals/locals/const/stack/etc
- document that on vm reset any allocated refs are not released - i.e. it is caller responsibility to reset arena!
- data type Copy => Clone, review usage - the call should always create new value, the caller itself decides on immutable and does a logical copy if needed (i.e. Retain)
- vm.raisedError.Error - returns "error" if payload is not VT_ERROR - shell we return "error: " + payload.String ?

- builtin types, modules and functions:
  - IDs must be FIRST..LAST..[RESERVED]..[USER_RESERVED], expose first user reserved
  - so the system and user IDs are stable even if new system added
  - API to add user defined

- add Retain/Release/Resolve to arena, so we start using it instead of ptr cast, so client code already prepared for refpool
- find a common solution for static (const) and dynamic memory:
  - primitives resolved on opcode level (load const = get preassembled Value from consts)
  - complex types resolved on refpool level (.Resolve) - decide if it from pool or const mem
- migrate to refpool
- improve refcounting and Retain/Release usage

# TODO list for Kavun

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
- capturing closures still have heap pieces: free-var slices are made with make, and captured locals can escape - can we use arena or pool?

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
- in VM slice logic, use fast path for VT_INT
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

- b"" format for bytes (i.e. string converted to bytes)
- range form f..l and f..l/s , i.e. range from f to l with step 1, and range from f to l witj step s

!!! check vm.go, "case bc.OpCall" and "case bc.OpMethodCall"
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
