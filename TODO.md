# TODO list for Kavun

- for arrays, bytes, runes, strings - store data=leng and ptr=underlying data (&[0] / StringData, etc) to avoid allocation of header struct
- allocate/release underlying arrays and dicts through allocator
- use store underlying array/dict pinter in Value.Ptr instead of using wrapper struct
- remove bool argument from NewDict/NewRecord/NewArray - use separate constructors for mutable/immutable

- try use unsafe.StringData / unsafe.String to store and rebuild strings?

- make bytes/runes mutable - add assign operator and check for immutability

- disallow to SetValueType for primitive types because they can be hardcoded in a hot path (int arithmetics, etc)
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

- check if we still need enums package - move missing functions to type properties
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
- string/rune/bytes/array * int => repeat n times
  
# enum module
implement following function from enums, then remove this module

enumerable = array, record, dict

  // chunk returns an array of elements split into groups the length of size.
  // If `x` can't be split evenly, the final chunk will be the remaining elements.
  chunk: func(x, size) {
    if !is_array_like(x) || !size { return undefined }

    numElements := len(x)
    if !numElements { return [] }

    res := []
    idx := 0
    for idx < numElements {
      res = append(res, x[idx:idx+size])
      idx += size
    }

    return res
  },

  // each iterates over elements of `x` and invokes `fn` for each element. `fn` is
  // invoked with two arguments: `key` and `value`. `key` is an int index
  // if `x` is array. `key` is a string key if `x` is dict. It does not iterate
  // and returns undefined if `x` is not enumerable.
  each: func(x, fn) {
    if !is_enumerable(x) { return undefined }

    for k, v in x {
      fn(k, v)
    }
  },

  // filter iterates over elements of `x`, returning an array of all elements `fn`
  // returns truthy for. `fn` is invoked with two arguments: `key` and `value`.
  // `key` is an int index if `x` is array. It returns undefined if `x` is not array.
  filter: func(x, fn) {
    if !is_array_like(x) { return undefined }

    dst := []
    for k, v in x {
      if fn(k, v) { dst = append(dst, v) }
    }

    return dst
  },

  // find iterates over elements of `x`, returning value of the first element `fn`
  // returns truthy for. `fn` is invoked with two arguments: `key` and `value`.
  // `key` is an int index if `x` is array. `key` is a string key if `x` is dict.
  // It returns undefined if `x` is not enumerable.
  find: func(x, fn) {
    if !is_enumerable(x) { return undefined }

    for k, v in x {
      if fn(k, v) { return v }
    }
  },

  // find_key iterates over elements of `x`, returning key or index of the first
  // element `fn` returns truthy for. `fn` is invoked with two arguments: `key`
  // and `value`. `key` is an int index if `x` is array. `key` is a string key if
  // `x` is dict. It returns undefined if `x` is not enumerable.
  find_key: func(x, fn) {
    if !is_enumerable(x) { return undefined }

    for k, v in x {
      if fn(k, v) { return k }
    }
  },

===

sequence[start:stop:step].
start: The beginning index (inclusive). Defaults to 0.
stop: The ending index (exclusive). The slice goes up to, but does not include, this index.
step: (Optional) The increment between each index. Defaults to 1. A negative step (e.g., ::-1) reverses the sequence.

Operation 	Syntax	Description
Get Element	seq[i]	Returns the item at index i.
Simple Slice	seq[start:stop]	Items from start to stop-1.
From Start	seq[:stop]	Items from index 0 up to stop-1.
To End	seq[start:]	Items from start to the end of the sequence.
With Step	seq[start:stop:step]	Items from start to stop-1, jumping by step.
Reverse	seq[::-1]	Returns a reversed version of the sequence.

[-1] → last element (index len - 1)
[-2] → second from end (index len - 2)
[-n] → n-th from end (index len - n)
[-10] on a 5-element list raises error
[0] → first element
[n] → element at position n
[5] on a 5-element list raises error (not wrapping)
[0:5] on 3-element list → all 3 elements (stop clamps to 3)
[-2:] → last 2 elements
[:-1] → all but last element
[::-1] → full reverse
[1:-1] → middle elements (excludes first and last)
[10:20] on 5-element list → empty (both out of bounds, clamped)
