- .Native() -> .Value()
- BinaryOp, IndexGet, IndexSet, etc should receive VM as an argument so in future we can construct new objects through VM
- refactor stdlib to use specific implementations or wrappers instead of generic Func* wrappers - it will help with logging and error handling
- implement built-in functions, member functions and constants opcodes so we can refer to built-in functions by IDs instead of names
- implement Record type and make it default when using {} syntax
- make it possible to construct map from record
- use memory pools and arenas inside VM to reduce allocations
- implement lambdas
- implement monad style member functions for collections (map, filter, reduce)
- IsUndefined
- get rid of Immutable* types, use IsImmutable flag on types instead
- migrate to crypto/rand
- Move strings package functions to the string type member functions
- Move time package functions to the time type member functions
- String as Bool (parse)
- String as Time (parse)
- All objects must be constructed through VM (move all New* functions to VM and make them member functions)
- BinaryOp - default rhs cast (after specific type checks) is .AsX()
- Equal - first check ptr eq, then use .AsX()
- index get/set - use .AsInt() .AsString() for index and value
- Iterators - falsy/bool based on whether they can iterate more
- Iterators - .Key / .Value - check bounds and return undefined if out of bounds
- Bool operators (logic and, or, not, etc)
- Investigate how .Copy is used - can we get rid of it?
- Bytes/String - IndexSet
- Review all stdlibs, check names for consistent style (snake-case, etc)

uncomment:
    stdlib,

    objects_test.go

    bytecode_test.go - gob serialization for private fields!

replace with constructors:
    &X{}
    X{}
    new(X)

    for:
        ArrayIterator
        BytesIterator
        MapIterator
        StringIterator
        String
        Bytes
        Char
        Int
        Float
        Bool
        Time
        Map, ImmutableMap => Map
        Array, ImmutableArray => Array
        Undefined
        ObjectPtr
        Error
        BuiltinFunction
        CompiledFunction

replace:
    .Native() => .Value()
