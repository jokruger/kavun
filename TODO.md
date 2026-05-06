# TODO list for Kavun

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
- add Set data type
- merge(r1, r2) → new record, dict.merge
- optimization for "modify and assign" pattern (reuse variable, pass argument to inform type logic)
- array.append (array) => new array
- array.extend (array) => inplace
- array.unique
- array.join
- fold(f, init) → value (same as reduce-with-init; pick one name)
- flatten() → array
- array.sort(lambda(a, b) => bool)
- window(n, step=1) → array[array]
- zip(other) → array[tuple] (or array[array] of len 2)
- enumerate() → array[(index, value)] (or dict-like pairs)
- string.split(sep) → array[string]
- array.join
- string replace(old, new), startsWith, endsWith
- bytes.hex()
- bytes.base64()
- move type related functions to type member functions; remove duplicates from stdlib (i.e. stdlib must be complimentary extension of type member functions)
- Arrays: `sort_by`
- Strings: `has_prefix`, `has_suffix`
- Int/Float: `abs`, `pow`, `is_zero`
- add time.is_leap_year(), time.is_weekend(), time.is_weekday(), time.is_holiday() (with holiday calendar)
- rune - implement methods from <https://pkg.go.dev/unicode>
- add Hash function for Value (and all types)
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
- container types: .reverse(), .shuffle(), .unique(), .join(sep), .split(sep), .chunk(size), .window(size, step), .zip(other), .enumerate()
- remove dict/record to string conversion - it breaks consistency... complex values should be printed, not converted to string implicitly
- add flag to `immutable` function to do a deep immutability (for arrays/dicts/records) - so all nested structures will be immutable as well
- go style switch with multi-value cases, default, etc
- string/rune/bytes/array \* int => repeat n times
- slices.compact
- compile time tail call optimization - runtime vm should not be smart, just a stupid loop over switch cases, all decisions should be made at compile time
- inlining and other optimizations
- .index_of or .index - search for element or subsequence (depending on arguments - mirror the .contains method), return index
- find a way to reuse value envelopes: receiver ptr instead of return value, mark as tmp, on assign copy if tmp, etc - primary usecase = loops
- how to use string value or envelope ptr in map keys, so we can use them when iterating over keys (instead of creating new strings)
- builtin cron support (expressions, next event, etc)
- builtin templates

- runtime formatting - fstring body as raw string, variables as map or array ({name} for maps, {1} for array)
- fstring fspec from variable (:{x}) -- see how python does it

- builtin memoization (for functions)
- use caches for parsing, etc (use cache package with controlled cache size)

===

    expectRun(t, `out = format("")`, nil, "")
    expectRun(t, `out = format("foo")`, nil, "foo")
    expectRun(t, `out = format("foo %d %v %s", 1, 2, "bar")`, nil, "foo 1 2 bar")
    expectRun(t, `out = format("foo %v", [1, "bar", true])`, nil, `foo [1, "bar", true]`)
    expectRun(t, `out = format("foo %v %d", [1, "bar", true], 19)`, nil, `foo [1, "bar", true] 19`)
    expectRun(t, `out = format("foo %v", {a: {b: {c: [1, 2, 3]}}})`, nil, `foo {"a": {"b": {"c": [1, 2, 3]}}}`)
    expectRun(t, `out = format("foo %v", {"a": {"b": {"c": [1, 2, 3]}}})`, nil, `foo {"a": {"b": {"c": [1, 2, 3]}}}`)
    expectRun(t, `out = format("%v", [1, [2, [3, 4]]])`, nil, `[1, [2, [3, 4]]]`)
