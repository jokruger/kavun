package core

import (
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const undefinedTypeName = "undefined"

// The absence of a value.
//
// ## Overview
//
// `undefined` is what the language answers where there is nothing to answer: a missing dict key, a locator miss,
// a function that returns nothing. (An **aggregation** on an empty sequence is not in that list — `[].first()`
// raises, because an element can itself be `undefined` and the two cases must stay distinguishable. See
// [array § Overview](array.md#overview).) It is not an error and not a zero — it is *absence*, and it
// has the thinnest member surface in the language: **`format()`, `is_true()`, and the conversion members with a
// mandatory default** (`undefined.int(0)` → `0` — see [Absence and defaults](#absence-and-defaults)). Absence can
// be rendered, tested, and materialized with an explicit fallback, but there is no value there to operate on —
// which is why `copy` and `freeze` do not exist on it, as members or as free calls.
//
// ```go
// undefined.format()        // => "undefined"
// undefined.is_true()       // => false
// !!undefined               // => false
// type_name(undefined)      // => "undefined"
// is_undefined(undefined)   // => true
// ```
//
// ## Where it appears
//
// ```go
// d := dict({a: 1})
// d["missing"]         // => undefined — a missed key
// [1, 2].index(9)      // => undefined — a locator miss (never -1)
//
// f := func() { }      // no return value
// f()                  // => undefined
//
// g := func() out { }  // a named result starts as undefined
// g()                  // => undefined — until the body (or a deferred handler) assigns it
// ```
//
// A locator miss also takes an optional trailing default that replaces the `undefined` answer:
// `[1, 2].index(9, -1)` → `-1`. An aggregation on an empty sequence is not absence: it raises unless a trailing
// default is given, which it then answers — `[].first(0)` → `0`.
//
// ## Operators
//
// One rule governs everything here: `undefined` **propagates on the data plane and raises on the action plane**.
// Chaining-shaped *reads* — selectors, indexing, slicing, arithmetic, concatenation, ordering — flow through and
// answer `undefined`, so a chain like `a.b.c` can miss at any level and still answer one `undefined` at the end,
// with no null-check flood. *Actions* — calling, membership, iteration, member calls — raise: absence performs
// nothing.
//
// On the data plane, arithmetic, concatenation, and ordering answer `undefined` rather than raising — with
// equality as the one exception that answers a real `bool`:
//
// ```go
// undefined + 1              // => undefined
// undefined + "a"            // => undefined
// undefined - 1              // => undefined
// undefined < 1              // => undefined — ordering has no answer against absence
// undefined == undefined     // => true — the exception: equality is answerable
// undefined != undefined     // => false
// undefined == 0             // => false — absence equals nothing but itself
// undefined && true          // => undefined
// undefined || 5             // => 5 — the usual "fallback" spelling
// ```
//
// The action plane refuses instead of flowing:
//
// ```go
//
//	1 in undefined          // Error: invalid_binary_operator — membership in an
//	                        // absent container is a program error, not an absent answer
//	u := undefined
//	u()                     // Error: not_callable
//	for x in undefined { }  // Error: not_iterable — absence has no elements,
//	                        // not zero of them; is_iterable(undefined) is false
//
// ```
//
// Indexing and slicing both propagate (`undefined[0]` → `undefined`, `undefined[0:1]` → `undefined`) — they are
// chaining-shaped reads, so a chain of lookups can miss at any level and still answer one `undefined` at the
// end. And since nothing is there to mutate, `is_immutable(undefined)` answers `true`.
//
// ## Absence and defaults
//
// `undefined` converts to nothing on its own: `int(undefined)` raises (`cannot convert undefined to int: value
// is missing`). But `undefined` carries **every conversion member** — `bool`, `byte`, `rune`, `int`, `float`,
// `decimal`, `time`, `date`, `string`, `runes`, `bytes`, `array`, `dict`, `record` — with a **mandatory default**, and
// the member answers the default. This is the terminal step of a propagated chain — materialize the miss with a
// fallback — and it replaces the test-for-undefined dance:
//
// ```go
// d := dict({a: 1})
// d["missing"].int(0)         // => 0 — instead of: v := d["missing"]; if v == undefined { ... }
// d["missing"].string("-")    // => "-"
// d["a"].int(0)               // => 1 — on a present value the member is the ordinary conversion
// ```
//
// The default is answered **as-is** — an explicit opt-out, not a type-checked replacement value
// (`undefined.int("n/a")` answers `"n/a"`). *As-is* includes identity: the default is **not copied**, so a
// container you pass as a default comes back as the very value you handed in, not a new one. This is the one
// place a conversion member does not construct, and deliberately — the default is a value you supplied, like
// the other branch of a conditional; nothing was converted, so nothing was built:
//
// ```go
// def := [1, 2]
// got := undefined.array(def)     // got IS def — the rescue answers your value
// got[0] = 9                      // def is [9, 2]
// array(def)                      // ... whereas the CONVERSION always builds a new array
// undefined.array(def.copy())     // say so explicitly when the default must be private
// ```
//
// And the default is not optional here: with nothing to convert and no fallback there is nothing to answer, so
// the no-default form raises:
//
// ```go
// undefined.int()             // Error: conversion: cannot convert undefined to int: value is missing
// ```
//
// There is no free spelling of the rescue — constructors take one value, so `int(d["missing"], 0)` raises
// `wrong_num_arguments`; the default belongs to the conversion *member*.
//
//kavun:type undefined

var TypeUndefined = ValueTypeDescr{
	Name:         ConstHook(undefinedTypeName),                                  // PURE by contract
	String:       func(Value) string { return undefinedTypeName },               // PURE by contract
	Format:       undefinedTypeFormat,                                           // PURE by contract
	EncodeJSON:   func(Value) ([]byte, error) { return []byte("null"), nil },    // PURE by contract
	EncodeBinary: func(Value) ([]byte, error) { return []byte{}, nil },          // PURE by contract
	DecodeBinary: func(v *Value, _ []byte) error { *v = Undefined; return nil }, // IMPURE by contract (mutates target)
	Interface:    func(Value) any { return nil },                                // PURE by contract
	IsTrue:       Const2Hook[bool, error](false, nil),                           // PURE by contract

	// Undefined propagates on the DATA plane (selectors, indexing, slicing, operators — a chain like a.b.c
	// misses at any level and answers one undefined) and raises on the ACTION plane (call, iteration,
	// membership, members). Getting an iterator is an action, so IsIterable answers false and for-in raises.
	IsIterable: ConstHook(false), // PURE by contract

	Equal:               undefinedTypeEqual,                                                        // PURE by contract
	UnaryOp:             undefinedTypeUnaryOp,                                                      // PURE by contract
	BinaryOp:            undefinedTypeBinaryOp,                                                     // PURE by contract
	AccessIndex:         func(Value, Value) (Value, error) { return Undefined, nil },               // PURE by contract
	AccessNamedProperty: func(VM, Value, string) (Value, error) { return Undefined, nil },          // PURE by contract
	Slice:               func(Value, Value, Value) (Value, error) { return Undefined, nil },        // PURE by contract
	SliceStep:           func(Value, Value, Value, Value) (Value, error) { return Undefined, nil }, // PURE by contract
	AsBool:              func(Value) (bool, bool) { return false, true },                           // PURE by contract

	Methods: undefinedMethods,
}

// PURE by contract
func undefinedTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return undefinedTypeName, nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
	}
	if sp.Verb != 0 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	return fspec.ApplyGenerics(undefinedTypeName, sp, fspec.AlignLeft), nil
}

func undefinedTypeEqual(v Value, other Value, _ bool) bool {
	// undefined is only equal to undefined
	return v.Type == other.Type
}

func undefinedTypeBinaryOp(Value, Value, token.Token, bool) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}

func undefinedTypeUnaryOp(Value, token.Token) (Value, error) {
	// undefined is propagated unconditionally
	return Undefined, nil
}
