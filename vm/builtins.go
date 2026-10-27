package vm

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"

	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/text"
)

var BuiltinFunctions = make(map[string]core.Value)
var BuiltinFunctionNames []string

// This page is the complete, authoritative list of global built-in functions (everything callable without an
// `import(...)`).
//
// For the full list of *member* functions on each type (e.g. `[1,2,3].sort()`), see
// [Detailed type documentation](../language.md#detailed-type-documentation) — this page only covers top-level
// functions.
//
// ## Value constructors / conversions
//
// `bool`, `byte`, `rune`, `int`, `float`, `decimal`, `time`, `date`, `string`, `runes`, `bytes`, `array`, `dict`,
// `record`, and `range` are all callable as top-level functions named after the type. Conversion is
// construction — the free form `T(x)` and the member form `x.T()` are one operation, and they share one
// failure contract:
//
// - **0 args** — the type's zero value (`int()` → `0`, `array()` → `[]`, `range()` → the empty range).
// - **1 arg, already the target type** — a **new, independent value**: a constructor constructs. For the five
// types with a mutable body (`array`, `dict`, `record`, `bytes`, `runes`) the result is a fresh **mutable**
// **shallow** copy — exactly `x.copy_shallow()` — so `b := array(a)` never writes through to `a`, and
// `bytes(b"ab")` gives a writable body built from a constant literal's content. The elements are the values
// handed in, so a frozen element stays frozen; `x.copy()` is the deep spelling. `string` and the scalars are
// immutable and carry no identity, so there is nothing to observe: they answer the same value.
// - **1 arg, any other type** — converted. A conversion that cannot succeed **raises** a catchable error — it
// never answers a silent `undefined` or a silent zero.
// - **The explicit fallback is the member form's**: `x.T(default)` answers the default instead of raising on a
// failed conversion. The fallback is the *opt-out*, so it is not type-checked, and it is answered **as-is** —
// a container default is **not** copied, unlike a converted value: nothing was converted, so nothing was
// built. `undefined.array(def)` answers `def` itself; `array(def)` answers a new array. The free form takes
// no fallback — `int("bad", 0)` raises `wrong_num_arguments`.
// - **A free constructor's second argument is a construction parameter, never a default.** Only two families
// take one: the sequence constructors' **count** (`T(x, n)`, below) and `time`'s / `date`'s **layout** for text
// (`time("29/08/2026", "%d/%m/%Y")`). The member conversion `s.time([default])` reads the canonical grammar like
// `time(s)`; the layout reading has its own member, `s.parse_time(layout[, default])` (`s.parse_date` for a
// date), whose default comes after the layout; see [time.md](types/time.md#text-layouts).
//
// Parsing always raises on invalid input, for every type — there is no in-band NaN-on-failed-parse:
// `decimal("bad")` raises exactly like `int("bad")` does (use the member form's default when a bad parse is
// expected: `"bad".decimal(0d)` → `0`).
//
// ## Building a sequence from copies: `T(x, count)`
//
// The four sequence constructors — `array`, `string`, `bytes`, `runes` — take a second **count** argument:
// `T(x, n)` builds a sequence of `n` copies of `x`, with `x` kept whole whatever its type (like `push`/`insert`,
// never spreading). For the text types, whose elements are scalars, this degenerates to content repetition —
// `T(x, n)` equals `T(x).repeat(n)`; for `array` it equals `[x].repeat(n)`. There is no sizing form — a single
// argument is always the conversion (`array(3)` → `[3]`, `runes(10)` → `u"10"`); preallocation is the explicit
// `undefined` fill:
//
// ```go
// array(undefined, 3)    // => [undefined, undefined, undefined] — preallocation, fill explicit
// array(0, 3)            // => [0, 0, 0]
// array([1, 2], 2)       // => [[1, 2], [1, 2]] — x stays whole, never spreads
// bytes(b'A', 3)         // => bytes([65, 65, 65])
// string("ab", 2)        // => "abab"
// runes("ab", 2)         // => u"abab"
// array(3)               // => [3] — arity 1 is the conversion, never a size
// runes(10)              // => u"10"
// array([1, 2])          // => [1, 2] — passthrough
// array(range(1, 4))     // => [1, 2, 3] — conversion
// ```
//
// The count must be a lossless whole number (`2.0` is fine, `1.5` raises) and non-negative (a negative count
// raises a recoverable `invalid_value` error); a count of `0` answers the empty sequence.
//
// ## `dict`/`record` are outliers
//
// `dict()` and `record()` — mirror images, converting in opposite directions — follow the shared convention above
// (the zero value, a same-type copy, a raise on any other type) and add two shapes of their own: the `_view`
// constructors, which share storage instead of copying, and the entries form, which builds from an array of
// `[key, value]` pairs:
//
// - `dict()` / `record()` — empty dict / record.
// - `dict(d)` / `record(r)` where the argument is already the target type — a new, independent shallow copy, as
// every constructor answers.
// - `dict(r)` where `r` is a `record`, and `record(d)` where `d` is a `dict` — converted to an independent
// **shallow** copy: a fresh top-level container, but nested values are not recursively cloned (same convention
// every other type's own `.record()`/`.dict()` conversion already follows). Mutating the result's own keys never
// affects the source's keys, or vice versa; mutating a *nested* container reachable from both still affects both
// (shallow, not deep). Result is always mutable, regardless of the source's mutability — same convention as
// `copy()`.
// - `dict_view(r)` / `record_view(d)` are the explicit `_view` twins: share the source's underlying storage
// directly instead of copying — both the top-level key set and nested values are the *same* backing map, so
// mutating either side through either wrapper is visible through both. Result's mutability is inherited from
// the source's (shares its immutability, not always-mutable like the copying form). Maximum performance when
// you've confirmed nothing else needs to observe the source independently.
// - `dict(entries)` / `record(entries)` where `entries` is an array — the ENTRIES reading: each element must be
// exactly a 2-element array `[key, value]` (`dict([["a", 1]])` → `dict({"a": 1})`); keys go through their
// string conversion, later entries overwrite earlier ones. The same reading powers `arr.dict()` and
// `d.array()` (which answers key-sorted entries), so the two directions round-trip up to ordering.
// - `dict(x)` / `record(x)` for any other type — **raises a runtime error**, like every failed conversion.
//
// The same member-call spellings exist on the relevant receiver: `dict_val.record()` / `dict_val.record_view()`.
// `record` has no member functions at all (a dot on a record is field access — see `docs/reference/types/record.md`),
// so `record_val.dict()` doesn't exist; `dict(record_val)`/`dict_view(record_val)` are its only spellings for that
// direction.
//
// ```go
// dict()                 // => dict({})
// dict({a: 1})            // => dict({"a": 1}) — from record, independent shallow copy
// record(dict({a: 1}))    // => {"a": 1} — from dict, independent shallow copy
// dict_view({a: 1})       // => dict({"a": 1}) — shares storage with the record literal
// dict(42)                // Error: conversion: cannot convert int to dict
// ```
//
// ## Collections and helpers
//
// A free function exists exactly where a member cannot do the job: nullary construction (`int()`), a multi-input
// subject (`min(a, b, c)` selects among its ARGUMENTS; `arr.min()` aggregates over elements — one meaning, two
// delivery mechanisms), or a receiver that cannot have members — `record` has no member surface at all, which is
// why `len`, the copy/freeze family, `remove`/`remove_in_place`, `format`, `is_true` and `is_view` all keep free
// spellings. A free form that would merely duplicate a member does not exist (there is no free `append`).
//
// ## Type predicates
//
// Each `is_T` predicate (except `is_callable`/`is_iterable`/`is_immutable`) checks the value's *exact*
// runtime type — no coercion, and no "is-a" relationship between related types (e.g. `is_int(byte(1))` is `false`).
//
//kavun:builtins

// Answer the runtime type name of `x` (`"function"` for every callable).
//
// ```go
// type_name({})      // => "record"
// ```
//
//kavun:builtin type_name slot=28 pure (x) -> string
func builtinTypeName(vm core.VM, args []core.Value) (core.Value, error) {
	// classification names types: scripts branch on bare lowercase names, so the
	// four function kinds answer "function" uniformly and the kind/arity detail
	// lives in the render (format(), f-strings). Custom host-defined callable
	// types keep their own registered name — is_callable is their unifier.
	switch args[0].Type {
	case value.CompiledFunction, value.BuiltinFunction, value.BuiltinClosure, value.BoundMember:
		return core.NewStringValue("function"), nil
	}
	return core.NewStringValue(args[0].TypeName()), nil
}

// Report whether `x` is exactly a `string`, with no coercion.
//
//kavun:builtin is_string slot=14 pure (x) -> bool
func builtinIsString(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.String {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `runes`, with no coercion.
//
//kavun:builtin is_runes slot=37 pure (x) -> bool
func builtinIsRunes(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Runes {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly an `int`, with no coercion.
//
//kavun:builtin is_int slot=12 pure (x) -> bool
func builtinIsInt(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Int {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `float`, with no coercion.
//
//kavun:builtin is_float slot=13 pure (x) -> bool
func builtinIsFloat(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Float {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `decimal`, with no coercion.
//
//kavun:builtin is_decimal slot=35 pure (x) -> bool
func builtinIsDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Decimal {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `bool`, with no coercion.
//
//kavun:builtin is_bool slot=15 pure (x) -> bool
func builtinIsBool(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Bool {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `byte`, with no coercion.
//
//kavun:builtin is_byte slot=39 pure (x) -> bool
func builtinIsByte(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Byte {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `rune`, with no coercion.
//
//kavun:builtin is_rune slot=16 pure (x) -> bool
func builtinIsRune(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Rune {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `bytes`, with no coercion.
//
//kavun:builtin is_bytes slot=17 pure (x) -> bool
func builtinIsBytes(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Bytes {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly an `array`, with no coercion.
//
// ```go
// is_array([1, 2])   // => true
// ```
//
//kavun:builtin is_array slot=18 pure (x) -> bool
func builtinIsArray(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Array {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `record`, with no coercion.
//
//kavun:builtin is_record slot=20 pure (x) -> bool
func builtinIsRecord(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Record {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `dict`, with no coercion.
//
//kavun:builtin is_dict slot=31 pure (x) -> bool
func builtinIsDict(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Dict {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly a `range`, with no coercion.
//
//kavun:builtin is_range slot=32 pure (x) -> bool
func builtinIsRange(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.IntRange {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` cannot be mutated through this reference.
//
// `is_immutable(x)` answers true unless the value can be mutated **through that reference** — it reads
// the header, exceptionlessly, so `is_immutable(undefined)` is `true`. It is not a promise that the content
// cannot change: a frozen header over a body something else still owns (a frozen view, or any
// `freeze_shallow`ed sibling) refuses *your* writes while the other holder's writes remain visible through it —
// see [container semantics: views](../container_semantics.md#views).
//
//kavun:builtin is_immutable slot=19 pure (x) -> bool
func builtinIsImmutable(vm core.VM, args []core.Value) (core.Value, error) {
	// true unless the value can be mutated — exception-less; undefined cannot be, so it answers true
	// even though the constant's header does not carry the flag
	return core.BoolValue(args[0].Immutable || args[0].Type == value.Undefined), nil
}

// Report whether `x` is exactly a `date`, with no coercion.
//
//kavun:builtin is_date slot=54 pure (x) -> bool
func builtinIsDate(vm core.VM, args []core.Value) (core.Value, error) {
	return core.BoolValue(args[0].Type == value.Date), nil
}

// Report whether `x` is exactly a `time`, with no coercion.
//
//kavun:builtin is_time slot=23 pure (x) -> bool
func builtinIsTime(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Time {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is exactly an `error`, with no coercion.
//
//kavun:builtin is_error slot=24 pure (x) -> bool
func builtinIsError(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Error {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is `undefined`.
//
//kavun:builtin is_undefined slot=25 pure (x) -> bool
func builtinIsUndefined(vm core.VM, args []core.Value) (core.Value, error) {
	if args[0].Type == value.Undefined {
		return core.True, nil
	}
	return core.False, nil
}

// Report whether `x` is a function: `type_name(x) == "function"`.
//
// `is_function(x)` is the type predicate (`type_name(x) == "function"`, true for functions, closures and builtin
// functions alike); `is_callable(x)` is the capability predicate — it additionally covers host-defined callable
// types, which keep their own `type_name`.
//
//kavun:builtin is_function slot=26 pure (x) -> bool
func builtinIsFunction(vm core.VM, args []core.Value) (core.Value, error) {

	switch args[0].Type {
	case value.BuiltinFunction, value.BuiltinClosure, value.CompiledFunction:
		return core.True, nil
	default:
		return core.False, nil
	}
}

// Report whether `x` can be called, host-defined callable types included; see [`is_function`](#is_function).
//
// ```go
// is_callable(len)   // => true
// ```
//
//kavun:builtin is_callable slot=27 pure (x) -> bool
func builtinIsCallable(vm core.VM, args []core.Value) (core.Value, error) {
	return core.BoolValue(args[0].IsCallable()), nil
}

// Report whether `x` can be iterated with `for … in`.
//
//kavun:builtin is_iterable slot=22 pure (x) -> bool
func builtinIsIterable(vm core.VM, args []core.Value) (core.Value, error) {
	return core.BoolValue(args[0].IsIterable()), nil
}

// Report whether `x` is a borrowing header — a `_view` result.
//
// ```go norun — a signature summary
// is_view(x)                                   // is x a borrowing header — a `_view` result?
// ```
//
// `is_view(x)` reports whether `x` is a **borrowing header** — a value produced by
// `slice_view`/`chunk_view`/`dict_view`/`record_view`, which reads and writes another value's body instead of
// owning one. It is a property of `x` alone: it does not say whether anything is viewing `x` (after
// `r = d.record_view()`, `is_view(r)` is `true` and `is_view(d)` is `false`), and it is not a general "is this
// body shared" test — plain assignment (`e := d`) and `freeze_shallow` also share a body without producing a
// view.
//
//kavun:builtin is_view slot=51 pure (x) -> bool
func builtinIsView(vm core.VM, args []core.Value) (core.Value, error) {
	// Free-only with universal domain (is_view(5) is false, not an error): predicates ABOUT a value — type,
	// capability, storage — read the header and must answer honestly on every value, record views included,
	// which no member could reach.
	switch args[0].Type {
	case value.Array:
		return core.BoolValue((*core.Array)(args[0].Ptr).IsView), nil
	case value.Bytes:
		return core.BoolValue((*core.Bytes)(args[0].Ptr).IsView), nil
	case value.Runes:
		return core.BoolValue((*core.Runes)(args[0].Ptr).IsView), nil
	case value.Dict:
		return core.BoolValue((*core.Dict)(args[0].Ptr).IsView), nil
	case value.Record:
		return core.BoolValue((*core.Record)(args[0].Ptr).IsView), nil
	}
	return core.False, nil
}

// Report the truthiness of `x`, the free spelling of `x.is_true()`.
//
// ```go norun — a signature summary
// is_true(x)                                   // truthiness (the free spelling of x.is_true())
// ```
//
//kavun:builtin is_true slot=50 pure (x) -> bool
func builtinIsTrue(vm core.VM, args []core.Value) (core.Value, error) {
	// NOT equality with `true`: is_true([1]) is true while [1] == true is false. Raises on an error-state
	// value (NaN).
	t, err := args[0].IsTrue()
	if err != nil {
		return core.Undefined, err
	}
	return core.BoolValue(t), nil
}

// Answer the length of a collection, text or range.
//
// ```go norun — a signature summary
// len(x)                                       // length of a collection/text/range; raises on types with no length
// ```
//
// `len(x)` raises on a type with no length — it never answers a made-up number.
//
//kavun:builtin len slot=0 pure (x) -> int
func builtinLen(vm core.VM, args []core.Value) (core.Value, error) {
	// the free form and the member are one operation with one domain: the types
	// that have a length. len(5) answering 1 was total and nonsensical — record
	// is why the free form exists at all (it can never have members)
	switch args[0].Type {
	case value.String, value.Runes, value.Bytes, value.Array, value.Dict, value.Record, value.IntRange:
		return core.IntValue(args[0].Len()), nil
	}
	if core.IsExtensionType(args[0].Type) || core.ModuleByType(args[0].Type) != nil { // a module: its member count
		return core.IntValue(args[0].Len()), nil
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("len", "first", "a value with a length", args[0].TypeName())
}

// Answer the smallest argument.
//
// ```go norun — a signature summary
// min(a, b, ...); max(a, b, ...)               // smallest/largest ARGUMENT (see below)
// ```
//
// `min`/`max` are variadic selection over their **arguments** — deliberately never looking inside one:
//
// ```go
// min(3, 1, 2)           // => 1
// min("banana", "apple") // => "apple"
// min([3, 1, 2])         // => [3, 1, 2] — ONE argument, returned unchanged; not a lookup inside it
// min([3, 1, 2]...)      // => 1 — spread it, which always agrees with [3, 1, 2].min()
// min()                  // Error: wrong_num_arguments — there is nothing to select
// ```
//
//kavun:builtin min slot=43 pure (first, ...rest) -> value
func builtinMin(vm core.VM, args []core.Value) (core.Value, error) {
	// variadic selection over ARGUMENTS (min(a, b, ...)); the member arr.min()
	// is the aggregation over elements — one meaning, two delivery mechanisms.
	// A zero-argument selection has no answer and raises.
	return minMaxReduce(args, tokens.Less)
}

// Answer the largest argument; see [`min`](#min).
//
//kavun:builtin max slot=44 pure (first, ...rest) -> value
func builtinMax(vm core.VM, args []core.Value) (core.Value, error) {
	return minMaxReduce(args, tokens.Greater)
}

func minMaxReduce(args []core.Value, op token.Token) (core.Value, error) {
	if len(args) == 0 {
		return core.Undefined, nil
	}

	best := args[0]
	for i := 1; i < len(args); i++ {
		better, err := args[i].BinaryOp(op, best)
		if err != nil {
			return core.Undefined, err
		}
		bt, terr := better.IsTrue()
		if terr != nil {
			return core.Undefined, terr
		}
		if bt {
			best = args[i]
		}
	}

	return best, nil
}

// Wrap `payload` as an error value; with `fatal` true, raising it bypasses every `recover()`.
//
// ```go
// error("msg")                                 // error value with a string payload
// error({code: 42})                            // error value with a structured payload
// ```
//
// `error(...)` requires at least one argument (an error without a payload carries no information — and note every
// error value is truthy).
//
//kavun:builtin error slot=33 pure (payload[, fatal]) -> error
func builtinError(vm core.VM, args []core.Value) (core.Value, error) {
	switch len(args) {
	case 1:
		if args[0].Type == value.Error {
			return args[0], nil
		}
		return core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, false), nil
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("error", "second", "bool", args[1].TypeName())
		}
		if args[0].Type == value.Error {
			o := (*core.Error)(args[0].Ptr)
			if o.Fatal == fatal {
				return args[0], nil
			}
			return core.NewErrorValue(o.Payload, o.Kind, o.Category, fatal), nil
		}
		return core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, fatal), nil
	default:
		return core.Undefined, core.ErrArgCount
	}
}

// Raise `err` — wrapped as an error first if it is not one — so a deferred `recover()` can catch it.
//
// ```go norun — a signature summary
// raise(err)                                   // raise an error so a deferred recover() can catch it
// ```
//
//kavun:builtin raise slot=40 impure (err[, fatal]) -> undefined
func builtinRaise(vm core.VM, args []core.Value) (core.Value, error) {
	var val core.Value
	switch len(args) {
	case 1:
		val = args[0]
		if val.Type != value.Error {
			val = core.NewErrorValue(val, core.KindUser, errs.CategoryUser, false)
		}
	case 2:
		fatal, ok := args[1].AsBool()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("raise", "second", "bool", args[1].TypeName())
		}
		if args[0].Type == value.Error {
			o := (*core.Error)(args[0].Ptr)
			val = core.NewErrorValue(o.Payload, o.Kind, o.Category, fatal)
		} else if fatal {
			val = core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, true)
		} else {
			val = core.NewErrorValue(args[0], core.KindUser, errs.CategoryUser, false)
		}
	default:
		return core.Undefined, core.ErrArgCount
	}
	return core.Undefined, newRaisedError(val)
}

// Answer `undefined` when `cond` is true; otherwise raise a recoverable error of kind `requirement` carrying `payload`
// untouched.
//
// ```go norun — a signature summary
// require(cond, payload)                       // input check: undefined if cond is true, else raise kind "requirement"
// ```
//
//kavun:builtin require slot=52 impure (cond, payload) -> undefined
func builtinRequire(vm core.VM, args []core.Value) (core.Value, error) {
	// Not pure: whether it raises depends on runtime state, so the optimizer must never fold it away.
	ok, err := args[0].IsTrue()
	if err != nil {
		return core.Undefined, err
	}
	if ok {
		return core.Undefined, nil
	}
	return core.Undefined, newRaisedError(
		core.NewErrorValue(args[1], errs.KindRequirement, errs.CategoryRequirement, false))
}

// Inside a deferred function, answer the in-flight error and stop the unwinding; otherwise `undefined`.
//
// ```go
// recover()                                    // inside a deferred function, return & clear the in-flight error
// ```
//
// It must be called directly inside the deferred function — any indirection answers `undefined`.
//
//kavun:builtin recover slot=41 impure () -> value
func builtinRecover(vm core.VM, args []core.Value) (core.Value, error) {
	return vm.Recover(), nil
}

// Build a `range`: empty with no argument, from a components record, or from `start`, `stop` and an optional `step`.
//
// ```go
// range({start: 1, stop: 4})  // => range(1, 4) — rebuilds from a components record
// ```
//
// ```go
// range(0, 10)                                 // range(start, stop[, step]) — sugar: 0..10, 0..10:step
// ```
//
//kavun:builtin range slot=30 pure () | (components) | (start, stop[, step]) -> range
func builtinRange(vm core.VM, args []core.Value) (core.Value, error) {
	numArgs := len(args)
	switch numArgs {
	case 0:
		// range() is the type's zero form — the empty range — so generic code can
		// spell "the zero value of range" the way it can for every other type
		return core.NewIntRangeValue(0, 0, 1), nil
	case 1:
		// a components MAP rebuilds the range: {start, stop[, step]}, strict keys
		var m map[string]core.Value
		switch args[0].Type {
		case value.Dict:
			m = (*core.Dict)(args[0].Ptr).Elements
		case value.Record:
			m = (*core.Record)(args[0].Ptr).Elements
		default:
			return core.Undefined, errs.NewInvalidArgumentTypeError("range", "components", "dict or record", args[0].TypeName())
		}
		return core.RangeFromComponents(m)
	case 2, 3:
	default:
		return core.Undefined, core.ErrArgCount
	}

	start, ok := args[0].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "start", "int", args[0].TypeName())
	}

	stop, ok := args[1].AsInt()
	if !ok {
		return core.Undefined, errs.NewInvalidArgumentTypeError("range", "stop", "int", args[1].TypeName())
	}

	step := int64(1)
	if numArgs == 3 {
		step, ok = args[2].AsInt()
		if !ok {
			return core.Undefined, errs.NewInvalidArgumentTypeError("range", "step", "int", args[2].TypeName())
		}
	}

	return core.NewIntRange(start, stop, step)
}

// Render one value, or fill a template's placeholders from `args`.
//
// ```go norun — a signature summary
// format(x); format(template, args)            // render one value / runtime template formatting (see below)
// ```
//
// ```go
// format("hello {x} from {y}!", {x: "Kavun", y: "Kherson"})  // => "hello Kavun from Kherson!"
// format("hello {0} from {1}!", ["Kavun", "Kherson"])        // => "hello Kavun from Kherson!"
// format("pi = {x:.3f}", {x: 3.14159})                       // => "pi = 3.142"
// format("n = {x:{fmt}}", {x: 42, fmt: "05d"})               // => "n = 00042"
// ```
//
// `format(template, args)` is the runtime counterpart to f-strings. The template uses the same `{name}` / `{0}`
// placeholder syntax and the same [Format Mini-Language](../format_mini_language.md) for `:fspec`. `args` must be an
// `array` (for indexed placeholders) or a `dict` / `record` (for named placeholders); the two modes cannot be mixed
// in one template, and expressions are not allowed inside `{...}`. See [`format`](../format_function.md) for the full
// reference.
//
//kavun:builtin format slot=29 pure (x) | (template, args) -> string
func builtinFormat(vm core.VM, args []core.Value) (core.Value, error) {
	// one operation graded by arity: format(x) renders any value (a template
	// with nothing to fill is its own rendering) — the render's callable form,
	// and record's render spelling; format(tmpl, args) fills placeholders
	if len(args) > 2 {
		return core.Undefined, core.ErrArgCount
	}
	if len(args) == 1 {
		out, err := args[0].Format(fspec.FormatSpec{})
		if err != nil {
			return core.Undefined, err
		}
		return core.NewStringValue(out), nil
	}
	if args[0].Type != value.String && args[0].Type != value.Runes && args[0].Type != value.Bytes {
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "template", "string", args[0].TypeName())
	}
	tmplStr, _ := args[0].AsString()

	var arr []core.Value
	var dict map[string]core.Value
	switch args[1].Type {
	case value.Array:
		arr = (*core.Array)(args[1].Ptr).Elements
	case value.Dict:
		dict = (*core.Dict)(args[1].Ptr).Elements
	case value.Record:
		dict = (*core.Record)(args[1].Ptr).Elements
	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array, dict, or record", args[1].TypeName())
	}

	tmpl, err := fspec.ParseTemplate(tmplStr)
	if err != nil {
		return core.Undefined, errs.FromFormatSpecError("format", err)
	}

	switch tmpl.Mode {
	case fspec.TemplateModeIndexed:
		if args[1].Type != value.Array {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "array", args[1].TypeName())
		}
	case fspec.TemplateModeNamed:
		if args[1].Type == value.Array {
			return core.Undefined, errs.NewInvalidArgumentTypeError("format", "args", "dict or record", args[1].TypeName())
		}
	}

	lookup := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.Index < 0 || seg.Index >= len(arr) {
				return core.Undefined, errs.NewIndexOutOfBoundsError("format", seg.Index, len(arr))
			}
			return arr[seg.Index], nil
		}
		v, ok := dict[seg.Name]
		if !ok {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("format: missing key %q", seg.Name))
		}
		return v, nil
	}

	lookupRef := func(seg fspec.TemplateSegment) (core.Value, error) {
		if tmpl.Mode == fspec.TemplateModeIndexed {
			if seg.SpecRefIndex < 0 || seg.SpecRefIndex >= len(arr) {
				return core.Undefined, errs.NewIndexOutOfBoundsError("format spec ref", seg.SpecRefIndex, len(arr))
			}
			return arr[seg.SpecRefIndex], nil
		}
		v, ok := dict[seg.SpecRefName]
		if !ok {
			return core.Undefined, errs.NewInvalidValueError(fmt.Sprintf("format: missing spec ref key %q", seg.SpecRefName))
		}
		return v, nil
	}

	var sb strings.Builder
	for _, seg := range tmpl.Segments {
		if seg.Kind == fspec.TemplateLiteral {
			sb.WriteString(seg.Literal)
			continue
		}
		val, err := lookup(seg)
		if err != nil {
			return core.Undefined, err
		}
		spec := seg.Spec
		if seg.HasSpec && seg.SpecIsRef {
			refVal, err := lookupRef(seg)
			if err != nil {
				return core.Undefined, err
			}
			// the spec is text: string, runes or bytes, as for the format member
			var specStr string
			switch refVal.Type {
			case value.String:
				specStr = *(*string)(refVal.Ptr)
			case value.Runes:
				specStr = text.EncodeText((*core.Runes)(refVal.Ptr).Elements)
			case value.Bytes:
				specStr = string((*core.Bytes)(refVal.Ptr).Elements)
			default:
				return core.Undefined, errs.NewInvalidArgumentTypeError("format", "spec ref", "string, runes or bytes", refVal.TypeName())
			}
			parsed, ferr := fspec.Parse(specStr)
			if ferr != nil {
				return core.Undefined, errs.FromFormatSpecError("format", ferr)
			}
			spec = parsed
		}
		out, ferr := val.Format(spec)
		if ferr != nil {
			return core.Undefined, ferr
		}
		sb.WriteString(out)
	}
	return core.NewStringValue(sb.String()), nil
}

// Answer a deep, mutable copy of `x`.
//
// ```go norun — a signature summary
// copy(x)                                      // deep mutable copy
// ```
//
// `copy`/`freeze` work on every type except `undefined` (absence has no identity to copy).
//
//kavun:builtin copy slot=1 pure (x) -> value
func builtinCopy(vm core.VM, args []core.Value) (core.Value, error) {
	// absence has no identity to copy; handle missing data explicitly (is_undefined)
	if args[0].Type == value.Undefined {
		return core.Undefined, errs.NewInvalidArgumentTypeError("copy", "first", "a value", "undefined")
	}
	return args[0].Copy(true)
}

// Answer a shallow, mutable copy of an `array`, `dict` or `record`.
//
// ```go norun — a signature summary
// copy_shallow(x)                              // shallow mutable copy (array/dict/record only — elsewhere unobservable)
// ```
//
//kavun:builtin copy_shallow slot=2 pure (x) -> value
func builtinCopyShallow(vm core.VM, args []core.Value) (core.Value, error) {
	// the twin exists only where the distinction is observable: the types whose
	// elements can themselves be containers
	switch args[0].Type {
	case value.Array, value.Dict, value.Record:
		return args[0].Copy(false)
	}
	if core.IsExtensionType(args[0].Type) {
		return args[0].Copy(false)
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("copy_shallow", "first", "array, dict, or record", args[0].TypeName())
}

// Answer a deep-immutable copy of `x`, leaving the source untouched.
//
// ```go norun — a signature summary
// freeze(x)                                    // deep copy, then deep-immutable; source untouched
// ```
//
// `freeze`/`freeze_shallow` differ in shape because immutability lives on the value header, not the shared body:
// `freeze(x)` detaches first (a deep copy) and marks the clone immutable, never affecting other bindings;
// `freeze_shallow(x)` returns `x` with its own header flagged — the caller must reassign
// (`x = freeze_shallow(x)`), and a sibling binding that still points at the same body stays mutable.
//
//kavun:builtin freeze slot=48 pure (x) -> value
func builtinFreeze(vm core.VM, args []core.Value) (core.Value, error) {
	// absence has no mutability; freezing it was a header-flag artifact
	if args[0].Type == value.Undefined {
		return core.Undefined, errs.NewInvalidArgumentTypeError("freeze", "first", "a value", "undefined")
	}
	return args[0].Freeze()
}

// Answer `x` with its own header marked immutable; see [`freeze`](#freeze).
//
// ```go norun — a signature summary
// freeze_shallow(x)                            // x, header marked immutable; needs `x = freeze_shallow(x)` to stick
// ```
//
//kavun:builtin freeze_shallow slot=49 pure (x) -> value
func builtinFreezeShallow(vm core.VM, args []core.Value) (core.Value, error) {
	switch args[0].Type {
	case value.Array, value.Dict, value.Record:
		return args[0].ToImmutable()
	}
	if core.IsExtensionType(args[0].Type) {
		return args[0].ToImmutable()
	}
	return core.Undefined, errs.NewInvalidArgumentTypeError("freeze_shallow", "first", "array, dict, or record", args[0].TypeName())
}

// Convert `x` to a `string`, or build `count` copies of it; with no argument, `""`.
//
// ```go
// string(99)            // => "99"
// string(undefined)     // Error: conversion — absence has no textual form
// ```
//
//kavun:builtin string slot=5 pure ([x[, count]]) -> string
func builtinString(vm core.VM, args []core.Value) (core.Value, error) {
	// .string() is a CONVERSION — the receiver's text content — not the render:
	// dict/record/callables have no text content and raise; format(x) renders.
	// string(x, n) is the count form: convert, then repeat the content n times.
	if len(args) == 2 {
		r, err := builtinString(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("string", args, core.EmptyString, func(t uint8) bool {
		switch t {
		case value.Bool, value.Byte, value.Rune, value.Int, value.Float, value.Decimal,
			value.String, value.Runes, value.Bytes, value.Array, value.IntRange, value.Time, value.Error:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.String:
			return src, true
		case value.Byte:
			s, ok := text.ByteSymbolString(byte(src.Data))
			return core.NewStringValue(s), ok
		case value.Bytes:
			// the UTF-8 decode is partial: invalid input declines
			b, _ := src.AsBytes()
			if !utf8.Valid(b) {
				return core.Undefined, false
			}
			return core.NewStringValue(string(b)), true
		case value.Array, value.IntRange:
			// a range past MaxSequenceLen declines
			elems, ok := src.AsArray()
			if !ok {
				return core.Undefined, false
			}
			rs, ok := core.ElementsToRunes(elems)
			return core.NewStringValue(text.EncodeText(rs)), ok
		case value.Error:
			o := (*core.Error)(src.Ptr)
			s, err := o.Payload.Format(fspec.FormatSpec{})
			return core.NewStringValue(s), err == nil
		}
		s, ok := src.AsString()
		return core.NewStringValue(s), ok
	})
}

// Convert `x` to `runes`, or build `count` copies of it; with no argument, empty `runes`.
//
// ```go
// runes("abc")          // => u"abc"
// ```
//
//kavun:builtin runes slot=36 pure ([x[, count]]) -> runes
func builtinRunes(vm core.VM, args []core.Value) (core.Value, error) {
	// mirrors .string() as symbols on every receiver that has text content —
	// note runes(65) is the CONVERSION runes("65"), never sizing; runes(x, n)
	// is the count form: convert, then repeat the content n times.
	if len(args) == 2 {
		r, err := builtinRunes(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("runes", args, core.NewRunesValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Bool, value.Byte, value.Rune, value.Int, value.Float, value.Decimal,
			value.String, value.Runes, value.Bytes, value.Array, value.IntRange, value.Time, value.Error:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Runes:
			return ctorSameType(src)
		case value.Byte:
			s, ok := text.ByteSymbolString(byte(src.Data))
			return core.NewRunesValue(text.DecodeText(s), false), ok
		case value.Bytes:
			// TOTAL: an undecodable octet decodes to its escape, as .runes() answers
			b, _ := src.AsBytes()
			return core.NewRunesValue(text.DecodeOctets(b), false), true
		case value.Array, value.IntRange:
			// a range past MaxSequenceLen declines
			elems, ok := src.AsArray()
			if !ok {
				return core.Undefined, false
			}
			rs, ok := core.ElementsToRunes(elems)
			return core.NewRunesValue(rs, false), ok
		case value.Error:
			o := (*core.Error)(src.Ptr)
			s, err := o.Payload.Format(fspec.FormatSpec{})
			return core.NewRunesValue(text.DecodeText(s), false), err == nil
		}
		s, ok := src.AsString()
		return core.NewRunesValue(text.DecodeText(s), false), ok
	})
}

// Convert `x` to `bytes`, or build `count` copies of it; with no argument, empty `bytes`.
//
// ```go
// bytes(b"ab")          // => bytes([97, 98]) — mutable; the literal itself stays an immutable constant
// ```
//
//kavun:builtin bytes slot=10 pure ([x[, count]]) -> bytes
func builtinBytes(vm core.VM, args []core.Value) (core.Value, error) {
	// bytes(x) is the CONVERSION — x's content as octets; a byte is one octet, a rune its UTF-8
	// encoding. bytes(int) RAISES: bytes plays a double role (ASCII text vs memory chunk), so an
	// int argument is ambiguous — spell the octet explicitly (bytes(b'A')) or the text (bytes("65")).
	// bytes(x, n) is the count form: convert, then repeat the content n times — the old sizing
	// form's spelling is an explicit fill, bytes(b'\x00', 5).
	if len(args) == 2 {
		r, err := builtinBytes(vm, args[:1])
		if err != nil {
			return core.Undefined, err
		}
		return ctorRepeat(vm, r, args[1])
	}
	return convertBuiltin("bytes", args, core.NewBytesValue(nil, false), func(t uint8) bool {
		switch t {
		case value.String, value.Runes, value.Bytes, value.Array, value.Byte, value.Rune:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Bytes:
			return ctorSameType(src)
		case value.Array:
			elems, _ := src.AsArray()
			bs, ok := core.ElementsToBytes(elems)
			return core.NewBytesValue(bs, false), ok
		case value.Byte:
			return core.NewBytesValue([]byte{byte(src.Data)}, false), true
		case value.Rune:
			// a rune contributes its encoding; an escape's encoding is its octet
			return core.NewBytesValue([]byte(text.EncodeRuneText(rune(src.Data))), false), true
		}
		b, ok := src.AsBytes()
		return core.NewBytesValue(b, false), ok
	})
}

// Convert `x` to an `array`, or build `count` copies of it as elements; with no argument, `[]`.
//
// ```go
// array([1, 2])         // => [1, 2] — a NEW array: writing into it never touches the argument
// ```
//
//kavun:builtin array slot=42 pure ([x[, count]]) -> array
func builtinArray(vm core.VM, args []core.Value) (core.Value, error) {
	// Two arities, two operations. array(x) is "represent x as a sequence": a convertible
	// (text, range, map entries) DECOMPOSES into its elements, anything else is one element —
	// array(5) is [5]; undefined raises (a conversion slot). array(x, n) is "n copies of x as
	// ONE element" — the never-spreading (push-family) reading, exactly [x].repeat(n):
	// array("ab", 2) is ["ab", "ab"], and undefined IS allowed there — array(undefined, 5)
	// is the explicit preallocation. The decomposing repetition is spelled convert-then-repeat:
	// array("ab").repeat(2).
	if len(args) == 2 {
		wrapped := core.NewArrayValue([]core.Value{args[0]}, false)
		return ctorRepeat(vm, wrapped, args[1])
	}
	return convertBuiltin("array", args, core.NewArrayValue(nil, false), func(t uint8) bool {
		// every present value has an array reading: decompose or wrap
		return true
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Array:
			return ctorSameType(src)
		case value.Dict:
			// a map's conversion elements are its entries, key-sorted
			return core.NewArrayValue(core.MapToSortedEntries((*core.Dict)(src.Ptr).Elements), false), true
		case value.Record:
			return core.NewArrayValue(core.MapToSortedEntries((*core.Record)(src.Ptr).Elements), false), true
		case value.String, value.Runes, value.Bytes, value.IntRange:
			elems, ok := src.AsArray()
			return core.NewArrayValue(elems, false), ok
		}
		if core.IsExtensionType(src.Type) {
			if elems, ok := src.AsArray(); ok {
				return core.NewArrayValue(elems, false), true
			}
		}
		// a non-sequence value is one element
		return core.NewArrayValue([]core.Value{src}, false), true
	})
}

// Convert `x` to a `dict`; with no argument, the empty dict; see [the outliers](#dictrecord-are-outliers).
//
//kavun:builtin dict slot=21 pure ([x]) -> dict
func builtinDict(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("dict", args, core.NewDictValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Dict, value.Record, value.Array:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Dict:
			return ctorSameType(src)
		case value.Record:
			return core.RecordToDict(src, false), true
		case value.Array:
			// the entries reading, agreeing with arr.dict()
			m, ok := core.ElementsToEntries((*core.Array)(src.Ptr).Elements)
			return core.NewDictValue(m, false), ok
		}
		return core.Undefined, false
	})
}

// Convert `x` to a `record`; with no argument, the empty record; see [the outliers](#dictrecord-are-outliers).
//
//kavun:builtin record slot=46 pure ([x]) -> record
func builtinRecord(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("record", args, core.NewRecordValue(nil, false), func(t uint8) bool {
		switch t {
		case value.Dict, value.Record, value.Array:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		switch src.Type {
		case value.Record:
			return ctorSameType(src)
		case value.Dict:
			return core.DictToRecord(src, false), true
		case value.Array:
			m, ok := core.ElementsToEntries((*core.Array)(src.Ptr).Elements)
			return core.NewRecordValue(m, false), ok
		}
		return core.Undefined, false
	})
}

// Convert `x` to a `time`, or read `text` under a `layout`; with no argument, the zero time (unix 0, UTC).
//
// ```go
// time("2024-01-01")    // time value (the canonical grammar; anything else names its layout)
// time("01/02/2024", "%d/%m/%Y")  // time value -- the second argument is the layout
// ```
//
//kavun:builtin time slot=11 pure ([x]) | (text, layout) -> time
func builtinTime(vm core.VM, args []core.Value) (core.Value, error) {
	if err := checkTextLayoutReading("time", args); err != nil {
		return core.Undefined, err
	}
	if len(args) == 2 {
		if args[1].Type != value.String {
			return core.Undefined, errs.NewInvalidArgumentTypeError("time", "second (layout)", "string", args[1].TypeName())
		}
		text, _ := args[0].AsString()
		layout, _ := args[1].AsString()
		t, err := core.ParseTimeLayout(text, layout)
		if err != nil {
			var le *core.LayoutError
			if errors.As(err, &le) {
				return core.Undefined, errs.NewInvalidValueError("(time) " + err.Error())
			}
			return core.Undefined, errs.NewConversionError(args[0].TypeName(), "time", err.Error())
		}
		return core.NewTimeValue(t), nil
	}
	// a components MAP is a conversion and rebuilds the instant; unknown keys
	// raise inside TimeFromComponents, so a typo never silently means year 1
	if len(args) == 1 && (args[0].Type == value.Dict || args[0].Type == value.Record) {
		var m map[string]core.Value
		if args[0].Type == value.Dict {
			m = (*core.Dict)(args[0].Ptr).Elements
		} else {
			m = (*core.Record)(args[0].Ptr).Elements
		}
		t, err := core.TimeFromComponents(m)
		if err != nil {
			return core.Undefined, err
		}
		return core.NewTimeValue(t), nil
	}
	if len(args) == 1 && (args[0].Type == value.String || args[0].Type == value.Runes) {
		// the canonical grammar, with the reason a text was refused in the message
		text, _ := args[0].AsString()
		t, err := core.ParseTimeText(text)
		if err != nil {
			return core.Undefined, errs.NewConversionError(args[0].TypeName(), "time", err.Error())
		}
		return core.NewTimeValue(t), nil
	}
	return convertBuiltin("time", args, core.NewTimeValue(core.ZeroTime), func(t uint8) bool {
		switch t {
		case value.Time, value.Date, value.String, value.Runes, value.Int, value.Float, value.Decimal:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Time {
			return src, true
		}
		t, ok := src.AsTime()
		return core.NewTimeValue(t), ok
	})
}

// Convert `x` to a `date`, or read `text` under a `layout`; with no argument, `1970-01-01`.
//
//kavun:builtin date slot=53 pure ([x]) | (text, layout) -> date
func builtinDate(vm core.VM, args []core.Value) (core.Value, error) {
	if err := checkTextLayoutReading("date", args); err != nil {
		return core.Undefined, err
	}
	if len(args) >= 1 && (args[0].Type == value.String || args[0].Type == value.Runes) {
		text, _ := args[0].AsString()
		var d civil.Date
		var err error
		if len(args) == 1 {
			d, err = core.ParseDateText(text)
		} else {
			if args[1].Type != value.String {
				return core.Undefined, errs.NewInvalidArgumentTypeError("date", "second (layout)", "string", args[1].TypeName())
			}
			layout, _ := args[1].AsString()
			d, err = core.ParseDateLayout(text, layout)
		}
		if err != nil {
			var le *core.LayoutError
			if errors.As(err, &le) {
				return core.Undefined, errs.NewInvalidValueError("(date) " + err.Error())
			}
			return core.Undefined, errs.NewConversionError(args[0].TypeName(), "date", err.Error())
		}
		return core.DateValue(d), nil
	}
	if len(args) == 1 && (args[0].Type == value.Dict || args[0].Type == value.Record) {
		var m map[string]core.Value
		if args[0].Type == value.Dict {
			m = (*core.Dict)(args[0].Ptr).Elements
		} else {
			m = (*core.Record)(args[0].Ptr).Elements
		}
		d, err := core.DateFromComponents(m)
		if err != nil {
			return core.Undefined, err
		}
		return core.DateValue(d), nil
	}
	return convertBuiltin("date", args, core.DateValue(core.ZeroDate), func(t uint8) bool {
		switch t {
		case value.Date, value.Time, value.Int:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		d, ok := src.AsDate()
		return core.DateValue(d), ok
	})
}

// checkTextLayoutReading bounds the `([x]) | (text, layout)` template of time and date: three or more arguments
// are refused by count, and two are the layout reading, whose first argument must be text — a two-argument call
// on anything else is a type error, never a count error (the template accepts two).
func checkTextLayoutReading(name string, args []core.Value) error {
	switch {
	case len(args) > 2:
		return core.ErrArgCount
	case len(args) == 2 && args[0].Type != value.String && args[0].Type != value.Runes:
		return errs.NewInvalidArgumentTypeError(name, "first (text)", "string or runes", args[0].TypeName())
	}
	return nil
}

// convertBuiltin implements the uniform free-constructor shape T([x]).
// T() is the zero value; T(x) builds the value from x's CONTENT — a safe, unambiguous
// conversion — and raises on failure: there is no free default form. The fallible-conversion
// idiom is the member's, where the receiver opts into recovery: x.T(default).
//
// Host-defined types always get an attempt: their hooks decide.
func convertBuiltin(name string, args []core.Value, zero core.Value, hasEdge func(uint8) bool, conv func(core.Value) (core.Value, bool)) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return zero, nil
	}
	if l > 1 {
		return core.Undefined, core.ErrArgCount
	}
	src := args[0]
	if src.Type == value.Undefined {
		return core.Undefined, errs.NewConversionError(src.TypeName(), name, "value is missing")
	}
	if !hasEdge(src.Type) && !core.IsExtensionType(src.Type) {
		return core.Undefined, errs.NewConversionError(src.TypeName(), name, "no conversion exists")
	}
	if r, ok := conv(src); ok {
		return r, nil
	}
	return core.Undefined, errs.NewConversionError(src.TypeName(), name, "")
}

// ctorSameType answers a constructor's same-type argument. A constructor CONSTRUCTS: `T(x)` where `x` is
// already a `T` builds a NEW, independent, mutable value rather than handing back the argument, so
// `b := array(a)` never writes through to `a` and `bytes(b"ab")` gives a writable body from a literal's
// shared constant. The copy is SHALLOW — exactly `x.copy_shallow()`: the elements are the values handed in,
// and a frozen element stays frozen. The deep spelling is `x.copy()`. Immutable types (string, the scalars)
// have no observable form of this: they cannot be written to and carry no identity, so their converters
// return the argument.
// PURE by contract: a fresh value that no one else holds is not external state.
func ctorSameType(src core.Value) (core.Value, bool) {
	c, err := src.Copy(false)
	if err != nil {
		return core.Undefined, false
	}
	return c, true
}

// ctorRepeat implements the sequence constructors' count form: T(x, n) is n copies of x-as-element.
// For the text types the element reading degenerates to content repetition (their elements are scalars
// only), so it is exactly T(x).repeat(n); array never spreads, so it is exactly [x].repeat(n). Both are
// implemented AS that member call, so validation (lossless int count, non-negative) and semantics can
// never drift from repeat's.
func ctorRepeat(vm core.VM, seq core.Value, count core.Value) (core.Value, error) {
	return seq.CallMember(vm, members.Repeat, member.ID(members.Repeat).String(), []core.Value{count})
}

// the numeric targets share one source set: the numerics themselves, text, and time
func numericConvEdge(t uint8) bool {
	switch t {
	case value.Int, value.Float, value.Decimal, value.String, value.Runes, value.Time:
		return true
	}
	return false
}

// Convert `x` to an `int`; with no argument, `0`.
//
// ```go
// int("42")             // => 42
// int("bad")            // Error: conversion: cannot convert string to int
// "bad".int(0)          // => 0 — explicit fallback: the member form's trailing default
// ```
//
//kavun:builtin int slot=6 pure ([x]) -> int
func builtinInt(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("int", args, core.IntValue(0), func(t uint8) bool {
		return numericConvEdge(t) || t == value.Bool || t == value.Byte || t == value.Rune
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Int {
			return src, true
		}
		i, ok := src.AsInt()
		return core.IntValue(i), ok
	})
}

// Convert `x` to a `float`; with no argument, `0.0`.
//
// ```go
// float("3.14")         // => 3.14
// ```
//
//kavun:builtin float slot=8 pure ([x]) -> float
func builtinFloat(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("float", args, core.FloatValue(0), numericConvEdge, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Float {
			return src, true
		}
		f, ok := src.AsFloat()
		return core.FloatValue(f), ok
	})
}

// Convert `x` to a `decimal`; with no argument, `0d`.
//
// ```go
// decimal("bad")        // Error: conversion — parse always raises on invalid input
// ```
//
//kavun:builtin decimal slot=34 pure ([x]) -> decimal
func builtinDecimal(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("decimal", args, core.NewDecimalValue(dec128.Decimal0), numericConvEdge, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Decimal {
			return src, true
		}
		d, ok := src.AsDecimal()
		if ok && d.IsNaN() {
			// a NaN decimal is an error state, never a produced value
			return core.Undefined, false
		}
		return core.NewDecimalValue(d), ok
	})
}

// Convert `x` to a `bool`: a numeric zero test or a text parse; with no argument, `false`.
//
// ```go
// bool(0.0)             // => false — bool(x) is a numeric zero test / text parse
// bool("false")         // => false — parses; distinct from is_true("false"), which is true (non-empty)
// ```
//
//kavun:builtin bool slot=7 pure ([x]) -> bool
func builtinBool(vm core.VM, args []core.Value) (core.Value, error) {
	// the conversion, not truthiness: a text receiver parses a boolean literal,
	// a numeric receiver is a zero check; everything else has no edge and raises.
	// Truthiness is is_true(x) / !!x.
	return convertBuiltin("bool", args, core.False, func(t uint8) bool {
		switch t {
		case value.Bool, value.Int, value.Float, value.Decimal, value.String, value.Runes:
			return true
		}
		return false
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Bool {
			return src, true
		}
		b, ok := src.AsBool()
		return core.BoolValue(b), ok
	})
}

// Convert `x` to a `byte`; with no argument, `byte(0)`.
//
// ```go
// byte(999)             // Error: conversion — out of the octet range
// ```
//
//kavun:builtin byte slot=38 pure ([x]) -> byte
func builtinByte(vm core.VM, args []core.Value) (core.Value, error) {
	// int is the sole gateway from the numeric domain; a rune converts iff its
	// UTF-8 form is one octet (ASCII); text does not parse into an ordinal type
	return convertBuiltin("byte", args, core.ByteValue(0), func(t uint8) bool {
		return t == value.Byte || t == value.Int || t == value.Rune
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Byte {
			return src, true
		}
		b, ok := src.AsByte()
		return core.ByteValue(b), ok
	})
}

// Convert `x` to a `rune`; with no argument, NUL.
//
//kavun:builtin rune slot=9 pure ([x]) -> rune
func builtinRune(vm core.VM, args []core.Value) (core.Value, error) {
	return convertBuiltin("rune", args, core.RuneValue(0), func(t uint8) bool {
		return t == value.Rune || t == value.Int || t == value.Byte
	}, func(src core.Value) (core.Value, bool) {
		if src.Type == value.Rune {
			return src, true
		}
		c, ok := src.AsRune()
		return core.RuneValue(c), ok
	})
}

// Answer a `dict` sharing a record's storage; see [the outliers](#dictrecord-are-outliers).
//
//kavun:builtin dict_view slot=45 pure ([x]) -> dict
func builtinDictView(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewDictValue(nil, false), nil
	}
	if l > 1 {
		return core.Undefined, core.ErrArgCount
	}

	switch args[0].Type {
	case value.Dict:
		return args[0], nil

	case value.Record:
		return core.RecordToDict(args[0], true), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("dict_view", "first", "dict or record", args[0].TypeName())
	}
}

// Answer a `record` sharing a dict's storage; see [the outliers](#dictrecord-are-outliers).
//
//kavun:builtin record_view slot=47 pure ([x]) -> record
func builtinRecordView(vm core.VM, args []core.Value) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NewRecordValue(nil, false), nil
	}
	if l > 1 {
		return core.Undefined, core.ErrArgCount
	}

	switch args[0].Type {
	case value.Record:
		return args[0], nil

	case value.Dict:
		return core.DictToRecord(args[0], true), nil

	default:
		return core.Undefined, errs.NewInvalidArgumentTypeError("record_view", "first", "dict or record", args[0].TypeName())
	}
}

// Answer a `dict` or `record` without `key`, leaving `obj` untouched.
//
// ```go norun — a signature summary
// remove(obj, "key")                           // returns obj without "key"; does not mutate obj
// ```
//
//kavun:builtin remove slot=3 pure (obj, key) -> value
func builtinRemove(vm core.VM, args []core.Value) (core.Value, error) {
	return args[0].Delete(args[1], false)
}

// Remove `key` from a `dict` or `record` in place; see [`remove`](#remove).
//
// ```go norun — a signature summary
// remove_in_place(obj, "key")                  // mutates a record/dict in place
// ```
//
//kavun:builtin remove_in_place slot=4 impure (obj, key) -> value
func builtinRemoveInPlace(vm core.VM, args []core.Value) (core.Value, error) {
	return args[0].Delete(args[1], true)
}
