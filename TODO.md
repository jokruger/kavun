- update documentations!

- disallow to SetValueType for primitive types because they can be hardcoded in a hot path (int arithmetics, etc)
- do atomic load check for "abort" flag every X cycles, not every cycle
- for int/float/string/etc args, fast path for specific types, only then call .AsX()

- map.record must sustain immutability of the record - if map is immutable, record must be immutable as well
- record.record must sustain immutability of the record - if record is immutable, record must be immutable as well
- array.array must sustain immutability of the array - if array is immutable, array must be immutable as well

- VM: case parser.OpImmutable - instead of checking for array/map/record just set immutable flag inplace! or add Value method to convert anything to immutable!
- VM: case parser.OpSliceIndex - move slicing logic to value member function

- need separate mutable and immutable constructors for primitives - so mutable can be modified inplace, immutable can be copied by reference

===

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
    - map/record to array of tuples
    - map/record from array of tuples

- add "decimal" type
- check if we still need enums package - move missing functions to type properties
- function property "arity" and "variadic"
- migrate to crypto/rand
- Move strings package functions to the string type member functions
- typed vectors, J core operators
- add Set data type
- merge(r1, r2) → new record, map.merge
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
- enumerate() → array[(index, value)] (or map-like pairs)
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
- map/array/record/string/bytes -> value level?
- string-iterator, array-iterator, etc -> value level?
- char - implement methods from https://pkg.go.dev/unicode
- add Hash function for Value (and all types)

- missing ctors(0/1/2): array, record
- range methods: map, filter, reduce, sum, etc (mirror array methods)
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