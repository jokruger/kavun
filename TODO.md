- hardcode builtin function indexes so we can add new functions, change their order in source code without breaking compatibility

- remove is_immutable_array, is_immutable_map
- add is_immutable
- return IsImmutable = true for basic types (int, string, bool, time, etc)

- rename Map to Record
- add Map, add builtin function to create a map from record (deep copy)

- BinaryOp, IndexGet, IndexSet, etc should receive VM as an argument so in future we can construct new objects through VM
- use memory pools and arenas inside VM to reduce allocations
- implement lambdas
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
- Investigate how .Copy is used - can we get rid of it?
- Bytes/String - IndexSet
- Review all stdlibs, check names for consistent style (snake-case, etc)
