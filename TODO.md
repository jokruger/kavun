- place kind at the beginning of the struct (with padding) - it is accessed first, so the data/ptr should cached after that

- use methods instead of properties for all types! this way it will be more consistent
- use pattern is_sorted(), sorted(), to_lower(), to_upper(), etc
- update documentations!

- for int/float/string/etc args, fast path for specific types, only then call .AsX()

- builtin/compiled functions  - separate interfaces and V_BUILTIN_FUNC and V_COMPILED_FUNC, add IsBuiltinFunction, IsCompiledFunction ???

- try value receivers instead of pinter (for core.Value at least)

- add core.Value.IsObject() - true is V_OBJECT or V_OBJECT_PTR, false if scalar (int, bool, float, etc)
- replace "if .Kind() ?? V_ " with .IsX() !

- core.NewObject, .NewImmutableObject, .NewTemporalObject !!!!!!

- .Copy, .Access - add "immutable" argument indicating the immutability of container/wrapper object

- add IsBuiltinFunction, IsCompiledFunction, IsIterator - they are frequently used in VM

- review how immutable flag is used:
  - immutable value can be copied by reference
  - mutable value can be modified inplace
  - add "const" keyword to create immutable scalars/etc

- make copy if object is "temporal"
- use "temporal" flag for temp values to optimize iterators, lambda loops, etc
  - array/map iterator .Value() can return "temporal" shallow copy
  - lambda loops can use "temporal" shallow copies for keys/values
  - string/bytes/etc lambda loops can use "temporal" variables for values

- NewObject - flags must be bitset, will be easier to pass as args to constructor
- map.record must sustain immutability of the record - if map is immutable, record must be immutable as well
- record.record must sustain immutability of the record - if record is immutable, record must be immutable as well
- array.array must sustain immutability of the array - if array is immutable, array must be immutable as well

- VM: case parser.OpImmutable - instead of checking for array/map/record just set immutable flag inplace!
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

- a special mode for "re-usable" objects:
    - regular objects copied by reference when used in assignment / append
    - "re-usable" objects are forced to copy when used in assignment / append
        - this allow re-use same object in loops, iterators, etc

- vector types: bytes, ints, floats

- new type Tuple.
    - map/record to array of tuples
    - map/record from array of tuples

- replace "x := y" with "var x" and "var x = y" syntax

- add "decimal" type

- check if we still need enums package - move missing functions to type properties

- function property "arity" and "variadic"

- add .Json() method to produce JSON representation of the value
- migrate to crypto/rand
- Move strings package functions to the string type member functions
- Iterators - .Key / .Value - check bounds and return undefined if out of bounds

- typed vectors, J core operators
- add Set data type
- add "x in y" syntax
- map.has / .contains
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
- string trim(), lower(), upper(), replace(old, new), contains(sub), startsWith, endsWith
- bytes.hex()
- bytes.base64()

- move type related functions to type member functions; remove duplicates from stdlib (i.e. stdlib must be complimentary extension of type member functions)

- make Len/Append/Delete a Value function, so len() can be used for user defined types too

Recommended Baseline Examples

- Arrays: `map`, `filter`, `reduce`, `sum`, `avg`, `min`, `max`, `len`, `is_empty`, `sort`, `sort_by`
- Strings: `len`, `is_empty`, `upper`, `lower`, `trim`, `contains`, `has_prefix`, `has_suffix`, `to_int`, `to_float`
- Int/Float: `abs`, `pow`, `is_zero`, `to_string`, `to_int`, `to_float`

- add time.is_leap_year(), time.is_weekend(), time.is_weekday(), time.is_holiday() (with holiday calendar)

- make range (builtin) return a generator instead of array, add member function 'to_array()' to convert to array if needed

- map/array/record/string/bytes -> value level?
- string-iterator, array-iterator, etc -> value level?

- value: use unsafe.Pointer and API to register custom types!

- check if use of different storage fields in Value affects performance (due to mem offset), shell we always use d64 first? shell we add bool field instead of d8?

- remove unused obj helper functions (set, len, get, etc)
- range generator
- "v Value" vs "v *Value - check if any performance difference

