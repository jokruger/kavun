1) write benchmark tests for most common scenarios (loops, recursion, arithmetics, etc)
2) replace *Object type system with boxed scalars:
    - value is a struct containing type info, flags (immutable, temporal, etc) and 64-bit data - pointer to heap object (for maps, strings, arrays) or int/bool/float/etc packed in the 64-bit data field

===

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
