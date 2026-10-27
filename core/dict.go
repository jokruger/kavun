package core

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

const (
	dictTypeName          = "dict"
	immutableDictTypeName = "immutable-dict"
)

type Dict struct {
	Elements map[string]Value
	// IsView reports whether Elements is shared with another value (a record it
	// was viewed from); set only by the explicit _view constructors
	IsView bool
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

// sortedKeys returns the dict's keys in lexical order. EVERY key enumeration on a map goes through this (or
// through the iterator, which sorts the same way): members, iteration, rendering and encoding alike. Key order
// is part of the language contract, not an implementation detail — never walk o.Elements directly to produce
// an observable sequence, or the Go map's randomized order leaks into script behaviour.
func (o *Dict) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewDictValue(m map[string]Value, immutable bool) Value {
	if m == nil {
		// a nil map reads fine but PANICS the host on assignment — every dict must be writable
		m = make(map[string]Value)
	}
	o := &Dict{Elements: m}
	return Value{Type: value.Dict, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

// Mutable string-keyed maps.
//
// ## Overview
//
// **A dict is a set of keys, each with an attached value: the key is the element, the value is the attachment.**
// That one sentence generates the whole surface. Search members (`contains`, `count`, `index`, `any`, `all`) match
// keys; `keys()` and `values()` name the two axes; `keep`/`remove` keep or drop whole entries; `map` transforms
// the attachment with the keys fixed. Any operation that would have to choose between the key axis and the value
// axis is deliberately absent — instead you name the collection you mean: `d.values().contains(x)`,
// `d.values().sum()`, `d.keys().min()`.
//
// Dicts are reference-typed: `b = a` shares the underlying map, `copy()` makes an independent one — see
// [container semantics](../../container_semantics.md). Keys are always strings. Values can be anything.
//
// `dict` and [`record`](record.md) are the same map shape with two access grammars: on a `dict`, `d[k]` is
// element access and `d.name(...)` is a member-function call; on a `record`, `r.k` is field access and there are no
// member functions at all. Conversions between them are one call in either direction (see [`record()`](#record) and
// [`dict()`](#dict)).
//
// The two failure modes shared by the surface:
//
// - Every `_in_place` twin raises `not_mutable` on a frozen receiver; on success it mutates the shared map and
// returns the receiver, so mutators chain.
// - Every member that takes an argument has **no blank (no-argument) reading**: a map has two axes, so a bare
// `d.keep()` / `d.contains()` / `d.any()` has no single meaning and raises `wrong_num_arguments` rather than
// guessing one.
//
// There is **no `.string()`** on dict, and the free `string(d)` raises too — a map's content has no text form.
// The rendering is `format()` / f-strings.
//
// ## Literals and construction
//
// There is no dict literal — `{a: 1}` is a [record](record.md) literal. A dict is built by the `dict()`
// constructor:
//
// ```go
// d = dict({a: 1, b: 2})       // from a record literal: dict({"a": 1, "b": 2})
// q = dict({"x y": 1})         // string key form for keys that aren't identifiers
// e = dict()                   // empty dict — a real, writable map:
// e["x"] = 1
// e                            // => dict({"x": 1})
//
// // a dict argument is copied, never aliased — a constructor constructs
// c = dict(d)                  // a new, independent dict (shallow: values are shared)
// c["a"] = 9                   // d["a"] is still 1
// dict(freeze(d))              // mutable again
// ```
//
// ### The entries form
//
// `dict([[k, v], ...])` constructs from a list of entries, where each element must be exactly a 2-element array —
// anything else raises (`conversion`). Keys go through their own string conversion; a duplicate key is
// last-one-wins. `arr.dict()` is the same reading as a member on `array`:
//
// ```go
// dict([["a", 1], ["b", 2]])       // => dict({"a": 1, "b": 2})
// [["a", 1], ["b", 2]].dict()      // same
// dict([[1, "x"], [2.5, "y"]])     // => dict({"1": "x", "2.5": "y"}) — keys stringified
// dict([[1, 2, 3]])                // Error: conversion — an entry is exactly [key, value]
// ```
//
// The reverse edge is [`d.array()`](#array), which answers the
// entries key-sorted — the two round-trip up to ordering.
//
// ### From a record
//
// ```go
// dict({a: 1})          // independent dict with the record's entries
// dict_view({a: 1})     // a dict sharing the record's map — see record: views
// ```
//
// The `_view` pair is the one place the constructor rule does **not** apply: `dict()` / `record()` always build
// a new value, and `dict_view()` / `record_view()` always share storage — that is their entire job, and the only
// way to get shared storage in the language. On an argument that is already the target type there is nothing to
// borrow — `d` already owns its map — so `dict_view(d)` answers `d` itself and `is_view(d)` stays `false`.
//
// ## Keys and the index operator
//
// Keys are strings. A non-string key stores under (and looks up by) its **canonical text** — the same reading
// `.string()` answers: `d[65]` keys `"65"`, `d[b'A']` keys `"A"`, `d[true]` keys `"true"`, and a high octet like
// `b'\xFF'` keys its escape text `"\xff"`. A value with no canonical text — a `dict`, a `record`, a callable,
// `undefined` — raises `invalid_index_type`, never silently keys by its rendering. A sequence-typed key (`array`,
// `range`) raises too: a transcode is not a key.
//
// ```go
// d = dict({a: 1})
// d[1] = "x"            // stored under "1"
// d[1.5] = "y"          // stored under "1.5"
// d[true] = "z"         // stored under "true"
// d[b'A'] = "w"         // stored under "A" — a byte's canonical text is its ASCII symbol
// d[b'\xFF'] = "v"      // stored under "\xff" — a high octet's text is its escape
// d["1"]                // => "x" — the same entry d[1] wrote
//
// d[{k: 1}]             // Error: invalid_index_type — record has no string conversion
// d[dict()]             // Error: invalid_index_type
// d[func() { return 1 }]  // Error: invalid_index_type
// d[undefined]          // Error: invalid_index_type
// d[b'\xFF']            // => "v" — the same entry d[b'\xFF'] wrote, under "\xff"
// d[[1, 2]]             // Error: invalid_index_type — a sequence is not a key
// d[range(1, 3)]        // Error: invalid_index_type
// ```
//
// Reading a missing key answers `undefined` — presence and "attached `undefined`" are told apart with
// `contains`:
//
// ```go
// d = dict({a: 1})
// d["nope"]             // => undefined
// d.contains("nope")    // => false
// ```
//
// `d[k] = v` is a statement, not an expression. On a frozen dict it raises `not_assignable`:
//
// ```go
// f = dict({a: 1}).freeze()
// f["b"] = 2            // Error: not_assignable
// ```
//
// Dot access on a dict is **not** element access — that namespace belongs to member functions, and it is refused
// in **both directions**: `d.a` (read) and `d.a = 5` (write) each raise `invalid_selector`
// (`type dict has no property a`). In a chain the refusal hits at the first dot — `d.x.y = 1` raises at `.x`.
// Element access is always `d["a"]` / `d["a"] = 5`; the selector grammar is [`record`](record.md)'s
// feature.
//
// ## Operators
//
// | operator | meaning | result |
// | --- | --- | --- |
// | `d1 + d2` | merge, last (right) wins on key collision | `dict` |
// | `dict + record`, `record + dict` | merge — either operand being a `dict` makes the result a `dict` | `dict` |
// | `d - "key"` | new dict without that entry; missing key is a no-op; non-mutating | `dict` |
// | `k in d` | key membership, with `contains`' rules (below) | `bool` |
// | `==` / `!=` | deep structural equality; crosses `dict`/`record` freely | `bool` |
// | `< > <= >=` | **not defined** — a map has no total order | raises |
//
// ```go
// dict({a: 1, b: 2}) + dict({b: 9, c: 3})   // => dict({"a": 1, "b": 9, "c": 3})
// dict({a: 1}) + {b: 2}                     // => dict({"a": 1, "b": 2})
// {a: 1} + dict({b: 2})                     // => dict({"a": 1, "b": 2})
//
// dict({a: 1, b: 2}) - "a"                  // => dict({"b": 2}) — the receiver is untouched
// dict({a: 1}) - "zz"                       // => dict({"a": 1}) — missing key, no-op
// ```
//
// `-` takes a `string` key only; there is no reflected form (`"key" - d` is not defined).
//
// `in` asks key membership and follows `contains`' acceptance — it converts what a key probe converts and raises
// on everything else, never answering a silent `false` for an operand it cannot read:
//
// ```go
//
//	"a" in dict({a: 1})               // => true
//	"z" in dict({a: 1})               // => false
//	1 in dict([["1", 5]])             // => true — the int converts to the key "1"
//
//	dict({a: 1}) in dict({a: 1, b: 2})     // Error: not_implemented — the submap reading is reserved
//	(func() { return 1 }) in dict({a: 1})  // Error: invalid_value — an operator operand is always a value;
//	                                       // the predicate reading is contains(f)/any(f)
//	undefined in dict({a: 1})              // Error: invalid_argument_type — not convertible to a key
//
// ```
//
// Equality is deep — element values compare by their own equality, key order never matters, and `dict` and
// `record` compare across each other:
//
// ```go
// dict({a: [1, 2]}) == dict({a: [1, 2]})   // => true
// dict({a: [1, 2]}) == dict({a: [1, 3]})   // => false
// dict({a: 1}) == {a: 1}                   // => true — dict/record cross freely
// ```
//
// ## Iteration
//
// A map's element is its key, so the single-variable form yields **keys**; the two-variable form yields
// `(key, value)`; values alone are spelled with the blank:
//
// ```go
// d = dict({b: 2, a: 1})
// for k in d {}             // k: "a", "b" — the keys
// for k, v in d {}          // ("a", 1), ("b", 2)
// for _, v in d {}          // 1, 2 — the values spelling
// ```
//
// **Key order is lexical, everywhere.** Iteration, every member that visits keys (`for_each`, `reduce`, `index`,
// `map`, `keep`, `count`, `any`, `all`, …), `keys()`/`values()`, the `array()` entries, and rendering and
// encoding all walk the same single order, so a map-driven program answers the same thing on every run — a
// display, a `json.encode` payload and a binary blob included. Keys are strings and sort as strings do
// (byte-wise: `"10"` before `"2"`, uppercase before lowercase).
//
// ```go
// d = dict({zebra: 1, apple: 2, mango: 3})
// d.keys()                       // => ["apple", "mango", "zebra"]
// for k in d {}                 // "apple", "mango", "zebra" — the same order
// array(d)                       // => [["apple", 2], ["mango", 3], ["zebra", 1]]
// ```
//
// Order is a *contract*, not an insertion memory: a dict has no insertion order to preserve, and re-inserting a
// key does not move it. It is also not a sortable sequence.
//
// ## Callbacks
//
// A member that takes a function — the predicate reading of the search members, `map`, `for_each`, `reduce` — calls it
// in the **fewest form it accepts**, read from the callable's arity and whether it is variadic, never from what kind of
// callable it is:
//
// - a per-element callback gets the key when it accepts one argument, else `(key, value)`; anything else raises;
// - a `reduce` callback gets `(acc, key)` when it accepts two arguments, else `(acc, key, value)`; anything else
// raises.
//
// So `func(x, ...rest)` and `func(...args)` get the key, and a builtin or a bound member is a callback like any other
// (`min` and `"abc".contains` accept one argument). A lambda is the one way to call a callable in another form — choose
// the form, reorder, fix or drop arguments:
//
// ```go
// d = dict({a: 1, b: 2})
// d.keep(func(k, v) { return v > 1 })                   // => dict({"b": 2}) — accepts two, not one
// d.count(func(k, ...rest) { return k == "a" })         // => 1 — accepts one: the key
// d.keep(["b", "c"].contains)                           // => dict({"b": 2})
// d.reduce(0, func(acc, k, v) { return acc + v })       // => 3
// ```
//
//kavun:type dict

var TypeDict = ValueTypeDescr{
	Name:                MutabilityNameHook(dictTypeName, immutableDictTypeName), // PURE by contract
	String:              dictTypeString,                                          // PURE by contract
	Format:              dictTypeFormat,                                          // PURE by contract
	Interface:           dictTypeInterface,                                       // PURE by contract
	EncodeJSON:          dictTypeEncodeJSON,                                      // PURE by contract
	EncodeBinary:        dictTypeEncodeBinary,                                    // PURE by contract
	DecodeBinary:        dictTypeDecodeBinary,                                    // IMPURE by contract (mutates target)
	IsTrue:              dictTypeIsTrue,                                          // PURE by contract
	IsIterable:          ConstHook(true),                                         // PURE by contract
	Iterator:            dictTypeIterator,                                        // PURE by contract (constructs fresh iterator)
	Equal:               dictTypeEqual,                                           // PURE by contract
	BinaryOp:            dictTypeBinaryOp,                                        // PURE by contract
	Copy:                dictTypeCopy,                                            // PURE by contract
	Len:                 dictTypeLen,                                             // PURE by contract
	AccessIndex:         dictTypeAccessIndex,                                     // PURE by contract
	AccessNamedProperty: dictTypeAccessNamedProperty,                             // PURE by contract
	AssignIndex:         dictTypeAssignIndex,                                     // IMPURE by contract
	AssignNamedProperty: dictTypeAssignNamedProperty,                             // IMPURE by contract
	Contains:            dictTypeContains,                                        // PURE by contract
	Delete:              dictTypeDelete,                                          // MUTATE-DEPENDENT by contract
	AsBool:              dictTypeAsBool,                                          // PURE by contract
	AsDict:              dictTypeAsDict,                                          // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (keep/count/all/any/for_each/find/map/reduce) are gated the same way as string's.

	Methods: dictMethods,
}

func dictTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for _, k := range o.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, o.Elements[k].String()))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeInterface(v Value) any {
	o := (*Dict)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func dictTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var b []byte
	b = append(b, '{')
	keys := o.sortedKeys()
	len1 := len(keys) - 1
	for idx, key := range keys {
		b = text.AppendJSONString(b, key)
		b = append(b, ':')
		eb, err := o.Elements[key].EncodeJSON()
		if err != nil {
			return nil, jsonPathPrefix("."+key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, '}')
	return b, nil
}

func dictTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for _, key := range o.sortedKeys() {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := o.Elements[key].EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func dictTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "dict (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(eb); err != nil {
			return fmt.Errorf("dict value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("dict: trailing %d bytes", len(data)-offset)
	}

	*v = NewDictValue(value, v.Immutable)

	return nil
}

func dictTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return dictTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(dictTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(dictTypeString(v), sp, fspec.AlignLeft), nil
}

// deep=true recursively copies every value (today's copy() semantics); deep=false only clones the top-level
// map header, leaving nested containers sharing the source (copy_shallow()).
func dictTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Dict)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	if !deep {
		for k, e := range o.Elements {
			c[k] = e
		}
		return NewDictValue(c, false), nil
	}
	for k, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewDictValue(c, false), nil
}

// DictToRecord converts a dict to a record. share=true reuses the dict's own map directly (record_view() /
// record_view(dict_val) — the explicit performance opt-in, today's original dict.record() behavior preserved
// under the new name); share=false (record() / dict_val.record()) builds an independent shallow copy — a new
// top-level map, elements copied by reference (not recursively cloned), matching every other type's own
// `.record()` conversion (array/bytes/runes/string/range all shallow-copy the same way). Used by both the
// dict.record()/dict.record_view() member cases and the free record()/record_view() constructors.
func DictToRecord(v Value, share bool) Value {
	o := (*Dict)(v.Ptr)
	if share {
		r := NewRecordValue(o.Elements, v.Immutable)
		(*Record)(r.Ptr).IsView = true
		return r
	}
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		c[k] = e
	}
	return NewRecordValue(c, false)
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func dictTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil
}

func dictTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Dict:
		return mapsEqual((*Dict)(v.Ptr).Elements, (*Dict)(other.Ptr).Elements)
	case value.Record:
		return mapsEqual((*Dict)(v.Ptr).Elements, (*Record)(other.Ptr).Elements)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)

}

// PURE by contract.
func dictTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Record:
			r := (*Dict)(v.Ptr).Elements
			switch op {
			case tokens.Add:
				l := (*Record)(other.Ptr).Elements
				return NewDictValue(mergeMaps(l, r), false), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	l := (*Dict)(v.Ptr).Elements
	switch other.Type {
	case value.Dict:
		r := (*Dict)(other.Ptr).Elements
		switch op {
		case tokens.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.Record:
		r := (*Record)(other.Ptr).Elements
		switch op {
		case tokens.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.String:
		switch op {
		case tokens.Sub:
			return dictTypeDelete(v, other, false)
		}

	default:
		// `-` shares remove's key reading: any operand whose string conversion exists
		// names a key (`d - 1` removes "1"), exactly like d.remove(1) and `1 in d`
		if op == tokens.Sub {
			if _, ok := other.AsString(); ok {
				return dictTypeDelete(v, other, false)
			}
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func dictTypeAccessIndex(v Value, index Value) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	o := (*Dict)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

// PURE by contract — a dict's keys are data, reached by d[k] only; d.name is refused
func dictTypeAccessNamedProperty(vm VM, v Value, name string) (Value, error) {
	if r, ok, err := PropertyByLookup(vm, v, name); ok {
		return r, err
	}
	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), name)
}

func dictFnKeys(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	keys := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		keys = append(keys, NewStringValue(k))
	}
	return NewArrayValue(keys, false), nil
}

func dictFnValues(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	values := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		values = append(values, o.Elements[k])
	}
	return NewArrayValue(values, false), nil
}

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all, all reading the KEY axis (a dict is a set of
// keys, each with an attached value):
//   - a function       a predicate, f/1(key) or f/2(key, value)
//   - a string         one key; several form a set
//   - a dict/record    the submap reading: deferred, raises saying so
//
// A map has no blank reading — it has two axes — so the no-argument form raises; reach the value axis with a
// predicate or via values(). Keys are visited in sorted order, so predicate side effects and short-circuiting
// are deterministic.
// ---------------------------------------------------------------------------

// dictMatch is a match member's argument list after reading; exactly one reading is set.
type dictMatch struct {
	pred *ElemCallback       // a predicate
	keys map[string]struct{} // a set of keys
}

// dictReadMatchArgs reads a match member's arguments.
func dictReadMatchArgs(vm VM, name string, args []Value) (dictMatch, error) {

	if args[0].IsCallable() {
		if len(args) > 1 {
			return dictMatch{}, seq.ErrPredicateAmongMany(name)
		}
		cb, err := newElemCallback(vm, name, args[0])
		if err != nil {
			return dictMatch{}, err
		}
		return dictMatch{pred: cb}, nil
	}

	keys := make(map[string]struct{}, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return dictMatch{}, seq.ErrFunctionInSet(name)
		}
		if a.Type == value.Dict || a.Type == value.Record {
			return dictMatch{}, errs.NewNotImplementedError("(" + name + ") the submap reading is deferred; match keys, or compare entries with a predicate")
		}
		k, ok := a.AsString()
		if !ok {
			return dictMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "a key (string) or a predicate", a.TypeName())
		}
		keys[k] = struct{}{}
	}
	return dictMatch{keys: keys}, nil
}

// matches reports whether the entry k: val matches the reading.
func (m *dictMatch) matches(k string, val Value) (bool, error) {
	if m.pred != nil {
		res, err := m.pred.CallEntry(NewStringValue(k), val)
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	_, hit := m.keys[k]
	return hit, nil
}

// dictKept answers the entries keep(...) keeps: the matches. Shared by keep and keep_in_place; name is the
// member called, for the errors.
func dictKept(vm VM, name string, v Value, args []Value) (map[string]Value, error) {
	m, err := dictReadMatchArgs(vm, name, args)
	if err != nil {
		return nil, err
	}
	o := (*Dict)(v.Ptr)
	out := make(map[string]Value, len(o.Elements))
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
		if err != nil {
			return nil, err
		}
		if hit {
			out[k] = o.Elements[k]
		}
	}
	return out, nil
}

// dictRemaining answers the entries remove(...) leaves: everything but the matches. Shared by remove and
// remove_in_place; name is the member called, for the errors.
func dictRemaining(vm VM, name string, v Value, args []Value) (map[string]Value, error) {
	m, err := dictReadMatchArgs(vm, name, args)
	if err != nil {
		return nil, err
	}
	o := (*Dict)(v.Ptr)
	out := make(map[string]Value, len(o.Elements))
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
		if err != nil {
			return nil, err
		}
		if !hit {
			out[k] = o.Elements[k]
		}
	}
	return out, nil
}

func dictTypeIsTrue(v Value) (bool, error) {
	return len((*Dict)(v.Ptr).Elements) > 0, nil
}

func dictTypeLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
// Selector access is the record's feature: a dict refuses the dot in BOTH directions, mirroring
// dictTypeAccessNamedProperty — d.a = 5 raises exactly like d.a, and d[k] is the dict's data spelling.
func dictTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Dict)(v.Ptr).Elements[k] = r

	return nil
}

// IMPURE by contract — d.name = r is refused, as d.name is
func dictTypeAssignNamedProperty(vm VM, v Value, name string, r Value) error {
	if ok, err := AssignPropertyByLookup(vm, v, name, r); ok {
		return err
	}
	return errs.NewInvalidSelectorError(v.TypeName(), name)
}

// dictTypeContains is the `in` operator on the KEY axis (a map's element is its key); a submap
// operand raises as deferred, a callable raises — an operator operand is always a value.
func dictTypeContains(v Value, e Value) (bool, error) {
	return mapContainsKey((*Dict)(v.Ptr).Elements, e)
}

// mutate=true: IMPURE, removes an entry from the receiver in place (delete_in_place()). Not folded by the
// optimizer. mutate=false: PURE, returns an independent dict without the key (delete()), leaving the receiver
// untouched — works regardless of the receiver's mutability, since nothing is mutated. See docs/purity.md.
// dictTypeMerge implements merge/merge_in_place — the map family's whole add side: variadic over maps (dict
// and record, the family's two members), entries applied in ARGUMENT ORDER with last-wins on key collision,
// exactly the + operator's rule. There is deliberately no single-entry add member: the single-entry spellings
// are d[k] = v (mutating, a statement) and d.merge(dict([[k, v]])) (non-mutating). mutate=false returns a
// fresh dict, receiver untouched; mutate=true writes into the receiver's own map, visible to every live alias,
// and returns the receiver. Both accept zero arguments as a legal no-op.
func dictTypeMerge(v Value, args []Value, mutate bool) (Value, error) {
	name := "merge"
	if mutate {
		name = "merge_in_place"
	}
	if mutate && v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	maps := make([]map[string]Value, 0, len(args))
	for i, a := range args {
		m, ok := a.AsDict()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, seq.ArgOrdinal(i), "dict or record", a.TypeName())
		}
		maps = append(maps, m)
	}

	o := (*Dict)(v.Ptr)
	if mutate {
		for _, m := range maps {
			for k, e := range m {
				o.Elements[k] = e
			}
		}
		return v, nil
	}

	c := make(map[string]Value, len(o.Elements)+len(args))
	for k, e := range o.Elements {
		c[k] = e
	}
	for _, m := range maps {
		for k, e := range m {
			c[k] = e
		}
	}
	return NewDictValue(c, false), nil
}

func dictTypeDelete(v Value, key Value, mutate bool) (Value, error) {
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("remove_in_place", v.TypeName())
		}
		delete((*Dict)(v.Ptr).Elements, s)
		return v, nil
	}

	o := (*Dict)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		if k != s {
			c[k] = e
		}
	}
	return NewDictValue(c, false), nil
}

func dictTypeAsBool(v Value) (bool, bool) {
	return len((*Dict)(v.Ptr).Elements) > 0, true
}

func dictTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
