# TODO list for Kavun

- for arrays, bytes, runes, strings - store data=leng and ptr=underlying data (&[0] / StringData, etc) to avoid allocation of header struct
- use store underlying array/dict pinter in Value.Ptr instead of using wrapper struct
- try use unsafe.StringData / unsafe.String to store and rebuild strings?

<<<<<<<

- do atomic load check for "abort" flag every X cycles, not every cycle
- for int/float/string/etc args, fast path for specific types, only then call .AsX()

- dict.record must sustain immutability of the record - if dict is immutable, record must be immutable as well
- record.record must sustain immutability of the record - if record is immutable, record must be immutable as well
- array.array must sustain immutability of the array - if array is immutable, array must be immutable as well

- VM: case parser.OpImmutable - instead of checking for array/dict/record just set immutable flag inplace! or add Value method to convert anything to immutable!
- VM: case parser.OpSliceIndex - move slicing logic to value member function

- need separate mutable and immutable constructors for primitives - so mutable can be modified inplace, immutable can be copied by reference

===

- string, add method to get len in runes (utf8.RuneCountInString)
- runes.trim - custom implementation that uses runes slice from allocator

- implement correct allocations limit control (dynamic objects, buffers, strings, etc) - is it even possible to control taking into account user types?

- variable.go - use AsX instead of type assertion

- vm: analyze "switch by (.type)" and replace with value member functions (.Immutable(), .Slice(), etc)
- replace all "switch by (.type)" - should use .IsX() functions instead
- mutable variables must be copied when assigned!

- bytes and string should be similar to array (immutable flag, assign by index, etc) - ensure constructors from other copies create new bytes/string!
- Bytes/String - IndexSet

- array, string, bytes - multi-index get: array[1, 3, 5], or array[x] where x is array of ints
- ... and multi-index set: array[1, 3, 5] = [10, 30, 50]

- analyze VM, design allocator.Release use strategy - is it even possible (or worth it) to know when object can be released?
  - Maybe it is better to minimize creation of new objects and just use arena allocator?
- implement arenas (no release object, just pre-allocated pool) for small number of objects

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
- reduce(f, init?) → value
- fold(f, init) → value (same as reduce-with-init; pick one name)
- find(pred) → value? (or null if not found)
- flatten() → array
- array.sort(lambda(a, b) => bool)
- chunk(n) → array[array]
- window(n, step=1) → array[array]
- zip(other) → array[tuple] (or array[array] of len 2)
- enumerate() → array[(index, value)] (or dict-like pairs)
- string.split(sep) → array[string]
- array.join
- string trim(), lower(), upper(), replace(old, new), startsWith, endsWith
- bytes.hex()
- bytes.base64()
- move type related functions to type member functions; remove duplicates from stdlib (i.e. stdlib must be complimentary extension of type member functions)
- Arrays: `sort_by`
- Strings: `has_prefix`, `has_suffix`
- Int/Float: `abs`, `pow`, `is_zero`
- add time.is_leap_year(), time.is_weekend(), time.is_weekday(), time.is_holiday() (with holiday calendar)
- dict/array/record/string/bytes -> value level?
- string-iterator, array-iterator, etc -> value level?
- char - implement methods from <https://pkg.go.dev/unicode>
- add Hash function for Value (and all types)

- missing ctors(0/1/2): array, record
- range methods: dict, filter, reduce, sum, etc (mirror array methods)
- generic range (just like int range but use Value for start/stop/step) - to be used for time, float, etc ranges as well
- splice - use AsArray
- move splice function to container types (methods)

- in VM slice logic, use fast path for VT_INT
- smart arena allocator:
  - used for complex (ptr-based) types only, no need to pre-allocate ints, bools, etc
  - use preallocated buffers
  - use .data to store index, use max-buff to indicate the value was allocated on the heap
  - when buff allocated value release, mark corresponding ptr as nil
  - if released value is last in buff, decrease buff cursor (til non-nil value found)

- add tests for AsX methods
- add mutex in VM

- vector/array operations like /+, /-, /\*, etc - elementwise operations for vectors
- format for decimal

- sign member function for int/float/decimal
- abs member function for int/float/decimal
- pow member function for int/float/decimal
- sqrt member function for int/float/decimal
- type() member function for all types, returning type name as string

- container types: .reverse(), .shuffle(), .unique(), .join(sep), .split(sep), .chunk(size), .window(size, step), .zip(other), .enumerate()
- byte type
- separate string and unicode
- shell we use ".to\_" names?
- why we need immutable arrays/records/dicts?

- remove dict/record to string conversion - it breaks consistency... complex values should be printed, not converted to string implicitly
- add flag to `immutable` function to do a deep immutability (for arrays/dicts/records) - so all nested structures will be immutable as well
- go style switch with multi-value cases, default, etc
- "not in" operator
- string/rune/bytes/array \* int => repeat n times
- slices.compact
- "byte" member functions (similar to .int)

- compile time tail call optimization - runtime vm should not be smart, just a stupid loop over switch cases, all decisions should be made at compile time
- inlining and other optimizations

- .index_of or .index - search for element or subsequence (depending on arguments - mirror the .contains method), return index

- find a way to reuse value envelopes: receiver ptr instead of return value, mark as tmp, on assign copy if tmp, etc - primary usecase = loops

- "len = 10" fails with cryptic error
- builtin cron support (expressions, next event, etc)

- make sure you cannot crash VM from script: limit num of allocs, total size of containers and mem used, catch panics
