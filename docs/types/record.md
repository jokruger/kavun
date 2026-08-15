# record

Object type with string keys and values of any type.

## Overview

Records are the primary object type in Kavun. They are collections of key-value pairs where keys are always strings.
Both dot notation (`.field_name`) and index notation (`["field_name"]`) can be used to access fields. Records are
reference-typed.

**Key Characteristic:** Records expose fields via selector/index access only - they have NO member functions. Use dot
notation for both reading and writing fields.

## Declaration and Usage

### Record Literals

```go
r = {name: "Alice", age: 30}
r2 = {x: 10, y: 20, z: 30}
empty = {}
```

### Accessing Fields

Both dot notation and index notation work:

```go
r = {name: "Alice", age: 30}

// Dot notation (for valid identifiers)
name = r.name       // "Alice"
age = r.age         // 30

// Index notation (for any string key)
name = r["name"]    // "Alice"
age = r["age"]      // 30

// With special characters, must use index notation
r2 = {"first-name": "Bob"}
first_name = r2["first-name"]  // "Bob"
```

### Accessing Non-Existent Fields

Accessing non-existent fields returns `undefined`:

```go
r = {name: "Alice"}
missing = r.missing    // undefined
also_missing = r["missing"]  // undefined
```

### Adding and Modifying Fields

```go
r = {name: "Alice"}

// Add new field
r.city = "Berlin"
r["country"] = "Germany"

// Modify existing field
r.name = "Alicia"
r["name"] = "Alicia"
```

### Removing Fields

`delete`/`delete_in_place` are kept as free functions specifically because `record` has no member functions at
all (see [container semantics](container-semantics.md)) — they're the only way to remove a key from a record.
`delete` is pure (does not mutate `r`); `delete_in_place` does:

```go
r = {name: "Alice", age: 30, city: "Berlin"}

r2 = delete(r, "age")   // r2 is {name: "Alice", city: "Berlin"}; r is untouched

delete_in_place(r, "age")   // mutates r directly
// r is now {name: "Alice", city: "Berlin"}
```

### Reference Semantics

```go
fmt = import("fmt")

r1 = {name: "Alice"}
r2 = r1

r1.name = "Alicia"
fmt.println(r2.name)    // "Alicia" (both point to same record)

r3 = copy(r1)       // Independent copy
r1.name = "Alice"
fmt.println(r3.name)    // "Alicia" (r3 is unchanged)
```

## Checking Field Existence

```go
r = {name: "Alice", age: 30}

"name" in r         // true
"age" in r          // true
"missing" in r      // false
```

## Key Features

- **No member functions**: Records don't have methods; use global functions
- **String keys only**: All keys are strings
- **Heterogeneous values**: Values can be of any type
- **Dot notation**: For field access (when key is valid identifier)
- **Index notation**: For any string key
- **Reference type**: Assignments create references, not copies

## Differences from Dict

| Feature          | Record                    | Dict                                               |
| ---------------- | ------------------------- | -------------------------------------------------- |
| Field Access     | Dot and index             | Index only                                         |
| Member Functions | None                      | Many (keys, values, filter, etc.)                  |
| Use Case         | Object representation     | Map/dictionary operations                          |
| Access Syntax    | `r.field` or `r["field"]` | `d["key"]` for access; `d.method()` for operations |

## Record to Dict Conversion

Records and dicts represent the same logical structure internally. When you convert a record to a dict with
`dict(record)`, no copy is made—they point to the same data:

```go
fmt = import("fmt")

r = {name: "Alice", age: 30}
d = dict(r)

// Both point to the same data
r.name = "Alicia"
fmt.println(d.keys())    // ["name", "age"] (with updated name)
```

## Examples

### Data Representation

```go
fmt = import("fmt")

// Represent structured data
user = {
    id: 1,
    name: "Alice",
    email: "alice@example.com",
    active: true
}

fmt.println("User:", user.name)
fmt.println("Email:", user.email)
```

### Building Records Dynamically

```go
fmt = import("fmt")

// Create record with computed fields
person = {}
person.first_name = "John"
person.last_name = "Doe"
person.age = 30

// Add computed field
person.full_name = person.first_name + " " + person.last_name

fmt.println(person.full_name)  // "John Doe"
```

### Nested Records

```go
// Records can contain records
company = {
    name: "TechCorp",
    headquarters: {
        city: "San Francisco",
        country: "USA"
    },
    employees: 150
}

hq_city = company.headquarters.city  // "San Francisco"
```

### Record Modification

```go
// Update and modify records
config = {debug: false, timeout: 30, port: 8080}

// Modify existing
config.debug = true

// Add new
config.env = "production"

// Remove (delete_in_place mutates directly; delete() is the pure twin, requires reassignment)
delete_in_place(config, "debug")
```

### Iterating Over Records

Since records don't have iteration methods, convert to dict to iterate:

```go
fmt = import("fmt")

r = {name: "Alice", age: 30, city: "Berlin"}
d = dict(r)

for key in d.keys() {
    value = d[key]
    fmt.println(key + ": " + value.string())
}
```

### Field Presence Checking

```go
fmt = import("fmt")

// Check if field exists before accessing
data = {x: 10}

y_value = undefined
if "y" in data {
    y_value = data.y
} else {
    y_value = 0  // default
}

fmt.println("Y value: " + y_value.string())
```

## Notes

- Records have **no member functions** - all field access is through dot/index notation
- To perform operations on record data, convert to `dict` first if needed
- Records are mutable: fields can be added, modified, and deleted
- Records are reference-typed for assignment efficiency
- Use `copy()` to create independent copies
- Field names must be valid identifiers for dot notation; use index notation for special characters
