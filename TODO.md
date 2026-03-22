- Allocator interface and dummy implementation
- .Access, .BinaryOp, .Copy, .Call, .Iterate, and all built-in functions should receive MM as an argument, so in future we can construct new objects through MM and use pools and arenas to reduce allocations

- analyze VM, design allocator.Release use strategy

- Guarantee there is no aliases - any assignment must do a deep copy! This will allow implement effective pool allocator!
  - ??? is it worth it? Maybe it is better to minimize creation of new objects and just use arena allocator?

- remove object ptr comparison in Equal - it is too rare case, better to just check type and value
- remove "optimizations" like "int + 0 = same object" - it is not worth the complexity
- ensure all object methods and built-in functions always return new objects, even max(a, b) should return new object, not a or b - this will allow VM know when it can release objects and will allow use pools and arenas to reduce allocations
- analyze VM if it is possible to know when object can be released (reassignments, stack-only objects, etc)
- implement arenas (no release object, just pre-allocated pool) for small number of objects
- try pools - check if it worth it

- add .Json() method to produce JSON representation of the value
- add function "record" to make records from maps
- add function array to make arrays from bytes / strings
- bytes and string should be similar to array (immutable flag, assign by index, etc) - ensure constructors from other copies create new bytes/string!
- use memory pools and arenas inside VM to reduce allocations
- implement monad style member functions for collections (map, filter, reduce)
- migrate to crypto/rand
- Move strings package functions to the string type member functions
- Move time package functions to the time type member functions
- String as Bool (parse)
- String as Time (parse)
- All objects must be constructed through VM (move all New* functions to VM and make them member functions)
- BinaryOp - default rhs cast (after specific type checks) is .AsX()
- Equal - first check ptr eq, then use .AsX()
- index get/set - use .AsInt() .AsString() for index and value
- Iterators - .Key / .Value - check bounds and return undefined if out of bounds
- Bool operators (logic and, or, not, etc)
- Bytes/String - IndexSet
- Review all stdlibs, check names for consistent style (snake-case, etc)
- Implement time parsing (string as time, etc) - use github.com/araddon/dateparse
