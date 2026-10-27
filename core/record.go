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
	"github.com/jokruger/kavun/internal/text"
)

const (
	recordTypeName          = "record"
	immutableRecordTypeName = "immutable-record"
)

type Record struct {
	Elements map[string]Value
	// IsView reports whether Elements is shared with another value (a dict it
	// was viewed from); set only by the explicit _view constructors
	IsView bool
}

func (o *Record) Set(elements map[string]Value) {
	o.Elements = elements
}

// sortedKeys returns the record's keys in lexical order — the same contract as dict (see Dict.sortedKeys).
// Iteration, rendering and encoding are all ordered, so a display, a JSON payload, a binary blob and a
// `for k in r` pass are all reproducible run to run.
func (o *Record) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewRecordValue(m map[string]Value, immutable bool) Value {
	if m == nil {
		// a nil map reads fine but PANICS the host on assignment — every record must be writable
		m = make(map[string]Value)
	}
	o := &Record{Elements: m}
	return Value{Type: value.Record, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

// The field-access shape.
//
// ## Overview
//
// A record is the same string-keyed map as a [`dict`](dict.md), carried in the **field-access shape**: `{a: 1}`
// is a record literal and `r.a` reads a field. That grammar is the whole design — on a record, `r.anything` is
// field access, so **member functions can never exist on it**: `r.len` would be the field `"len"`, not an
// operation. A record therefore has **no member surface at all**, and every operation that must reach one has a
// free-builtin spelling instead (`len(r)`, `copy(r)`, `remove(r, k)`, …) — which is exactly why those free
// builtins exist for every type.
//
// Records are reference-typed: `b = a` shares the map, `copy()` makes an independent one — see [container
// semantics](../../container_semantics.md). Use a record where the data is object-shaped and read by name; use a
// `dict` where you need the map operations (`merge`, `keep`, `keys`, …) — the two convert into each other with one
// call, and compare equal across the boundary.
//
// ## Literals and construction
//
// ```go
// r = {name: "Alice", age: 30}     // record literal
// q = {"x y": 1}                   // string key form for keys that aren't identifiers
// n = {a: {b: 1}}                  // nests freely; n.a.b is 1
// e = record()                     // the empty record: {}
// ```
//
// `record([[k, v], ...])` constructs from a list of entries — each element exactly a 2-element `[key, value]`
// array, anything else raises (`conversion`); keys go through their own string conversion, last one wins:
//
// ```go
// record([["a", 1], ["b", 2]])     // => {"a": 1, "b": 2}
// record(["bad"])                  // Error: conversion — an entry is exactly [key, value]
// ```
//
// `record(r)` on a record is **not** a pass-through: a constructor constructs, so it answers a new,
// independent, mutable record — a shallow copy (`record(freeze(r))` is writable again; the field *values* keep
// their own state). A record literal is likewise built at run time, which is why it is mutable while a text
// literal is a shared constant — see [Constant literals and constructed
// literals](../../language.md#constant-literals-and-constructed-literals).
//
// ```go
// r = {a: 1}
// c = record(r)                    // a new record
// c.a = 9                          // r.a is still 1
// ```
//
// `record(m)` / `record_view(m)` convert from a dict — see [Conversions](#conversions-to-and-from-dict).
//
// ## Field access
//
// `r.k` and `r["k"]` are the same access. A missing field answers `undefined`. Assignment adds or overwrites a
// field; on a frozen record it raises `not_assignable`:
//
// ```go
// r = {a: 1}
// r.a               // => 1
// r["a"]            // => 1
// r.missing         // => undefined
//
// r.c = 3           // adds a field: {"a": 1, "c": 3}
// r["d"] = 4        // same operation, index spelling
//
// f = freeze({a: 1})
// f.a = 2           // Error: not_assignable
// ```
//
// A field holding a function is called through the same access — `r.f()` is "read field `f`, call it", which is
// precisely the grammar member functions would collide with:
//
// ```go
// r = {f: func() { return 7 }}
// r.f()             // => 7
// ```
//
// That holds for every name, the universal ones included: a record has no `is_true` member, so `r.is_true()` calls a
// field named `is_true` — truthiness is the free `is_true(r)`:
//
// ```go
// {is_true: func() { return "field" }}.is_true()   // => "field"
// {a: 1}.is_true()                                // Error: invalid_method: type record has no method is_true
// is_true({a: 1})                                 // => true
// ```
//
// ## The free-builtin surface
//
// Everything a member would do elsewhere is a free builtin here. Free and member forms share one domain — each of
// these also works on every other type that has the corresponding member:
//
// | free builtin | answer |
// | --- | --- |
// | `len(r)` | number of fields |
// | `copy(r)` / `copy_shallow(r)` | independent deep copy / new map sharing the field values |
// | `freeze(r)` / `freeze_shallow(r)` | deep-frozen / map-only-frozen record |
// | `format(r[, spec])` | the render (records have no `.string()` — `string(r)` raises) |
// | `is_true(r)` | truthiness: non-empty |
// | `remove(r, k)` / `remove_in_place(r, k)` | field removal — non-mutating / mutating |
// | `is_view(r)` | whether `r` is borrowing another value's map (a `_view` result) |
// | `array(r)` | the entries, key-sorted: `[[k, v], ...]` |
// | `is_record(r)` / `is_immutable(r)` | type / storage predicates |
//
// ```go
// r = {name: "Alice", age: 30}
// len(r)                       // => 2
// format(r)                    // => "{\"age\": 30, \"name\": \"Alice\"}" — a map renders key-sorted
// is_true(record())            // => false — empty
// is_true(r)                   // => true
//
// remove(r, "age")             // => {"name": "Alice"} — r untouched
// remove_in_place(r, "age")    // mutates r; raises not_mutable on a frozen record
//
// array({b: 2, a: 1})          // => [["a", 1], ["b", 2]] — entries, key-sorted
// ```
//
// Rendering, encoding **and iteration** are all key-sorted, so a display, a `json.encode` payload, a binary blob
// and a `for k in r` pass are the same on every run (see [Iteration](#iteration)).
//
// `copy`/`copy_shallow` and `freeze`/`freeze_shallow` behave exactly as on [`dict`](dict.md#copy):
// the unsuffixed forms are deep, the `_shallow` twins stop one level down.
//
// ## Operators
//
// | operator | meaning | result |
// | --- | --- | --- |
// | `r1 + r2` | merge, last (right) wins | `record` |
// | `record + dict`, `dict + record` | merge — either operand being a `dict` makes the result a `dict` | `dict` |
// | `k in r` | key membership, same rules as on `dict` | `bool` |
// | `==` / `!=` | deep structural equality; crosses `record`/`dict` freely | `bool` |
// | `r - "key"` | **not defined** — raises; field removal is the free `remove(r, k)` | raises |
// | `< > <= >=` | **not defined** — a map has no total order | raises |
//
// ```go
// {a: 1} + {b: 9, c: 3}        // => {"a": 1, "b": 9, "c": 3}
// {a: 1} + dict({b: 2})        // => dict({"a": 1, "b": 2})
// "a" in {a: 1}                // => true
// {a: [1]} == {a: [1]}         // => true — deep
// dict({a: 1}) == {a: 1}       // => true — dict/record cross freely
// {a: 1, b: 2} - "a"           // Error: invalid_binary_operator — use remove({a: 1, b: 2}, "a")
// ```
//
// ## Iteration
//
// Same as `dict`: the single-variable form yields **keys**, the two-variable form `(key, value)`, and **key
// order is lexical** — a `record` and a `dict` share one iterator, so they agree exactly:
//
// ```go
// for k in {a: 1, b: 2} {}           // "a", "b" — the keys
// for k, v in {a: 1, b: 2} {}        // ("a", 1), ("b", 2)
// for _, v in {a: 1, b: 2} {}        // 1, 2 — the values spelling
// ```
//
// For anything beyond a loop — `keep`, `map`, `merge` with several sources, the key/value arrays — convert to a
// `dict` and use its members.
//
// ## Conversions, to and from dict
//
// Each direction has an independent-copy form and a `_view` form; the view shares the same underlying map — the
// performance opt-in when the copy would be pure overhead:
//
// ```go
// r = {a: 1}
// d = dict(r)          // independent dict — writes to d never reach r
// w = dict_view(r)     // shares r's map:
// w["b"] = 2
// r.b                  // => 2 — the record sees the write
// is_view(w)           // => true
//
// d2 = dict({a: 1})
// r2 = d2.record()         // independent record
// v2 = d2.record_view()    // shares d2's map
// d2["c"] = 9
// v2.c                     // => 9 — live
// ```
//
// The entries boundary works in both directions and round-trips up to ordering: `array(r)` answers the key-sorted
// `[[k, v], ...]` entries, `record([[k, v], ...])` builds from them.
//
//kavun:type record

var TypeRecord = ValueTypeDescr{
	Name:                MutabilityNameHook(recordTypeName, immutableRecordTypeName), // PURE by contract
	String:              recordTypeString,                                            // PURE by contract
	Format:              recordTypeFormat,                                            // PURE by contract
	Interface:           recordTypeInterface,                                         // PURE by contract
	EncodeJSON:          recordTypeEncodeJSON,                                        // PURE by contract
	EncodeBinary:        recordTypeEncodeBinary,                                      // PURE by contract
	DecodeBinary:        recordTypeDecodeBinary,                                      // IMPURE by contract (mutates target)
	IsTrue:              recordTypeIsTrue,                                            // PURE by contract
	IsIterable:          ConstHook(true),                                             // PURE by contract
	Iterator:            recordTypeIterator,                                          // PURE by contract (constructs fresh iterator)
	Copy:                recordTypeCopy,                                              // PURE by contract
	Len:                 recordTypeLen,                                               // PURE by contract
	Equal:               recordTypeEqual,                                             // PURE by contract
	BinaryOp:            recordTypeBinaryOp,                                          // PURE by contract
	CallNamedMethod:     recordTypeCallNamedMethod,                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessNamedProperty: recordTypeAccessNamedProperty,                               // PURE by contract
	AccessIndex:         recordTypeAccessIndex,                                       // PURE by contract
	AssignNamedProperty: recordTypeAssignNamedProperty,                               // IMPURE by contract
	AssignIndex:         recordTypeAssignIndex,                                       // IMPURE by contract
	Contains:            recordTypeContains,                                          // PURE by contract
	Delete:              recordTypeDelete,                                            // MUTATE-DEPENDENT by contract
	AsBool:              recordTypeAsBool,                                            // PURE by contract
	AsDict:              recordTypeAsDict,                                            // PURE by contract
	IsNamedMethodPure:   func(string) bool { return false },                          // method calls are redirected to the value keys, so conservatively assume they are impure
}

func recordTypeString(v Value) string {
	o := (*Record)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for _, k := range o.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, o.Elements[k].String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeInterface(v Value) any {
	o := (*Record)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func recordTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)
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

func recordTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for _, key := range o.sortedKeys() {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := o.Elements[key].EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func recordTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "record (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("record key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("record value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(eb); err != nil {
			return fmt.Errorf("record value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("record: trailing %d bytes", len(data)-offset)
	}

	*v = NewRecordValue(value, v.Immutable)

	return nil
}

func recordTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return recordTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(recordTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(recordTypeString(v), sp, fspec.AlignLeft), nil
}

// deep=true recursively copies every value (copy()); deep=false only clones the top-level map header, leaving nested
// containers sharing the source (copy_shallow()). record has no member surface (docs/reference/types/record.md), so
// neither is reachable as a member call — only via the free copy()/copy_shallow() builtins, through the Value.Copy
// hook.
func recordTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	if !deep {
		for k, e := range o.Elements {
			c[k] = e
		}
		return NewRecordValue(c, false), nil
	}
	for k, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewRecordValue(c, false), nil
}

// RecordToDict converts a record to a dict. share=true reuses the record's own map directly (dict_view(record_val),
// the explicit sharing opt-in); share=false (dict(record_val)) builds an independent shallow copy — a new
// top-level map, elements copied by reference (not recursively cloned), matching every other type's own `.dict()`
// conversion. Only reached via the free `dict`/`dict_view` constructors: record has no member surface
// (docs/reference/types/record.md), so there is no `record_val.dict()` member form.
func RecordToDict(v Value, share bool) Value {
	o := (*Record)(v.Ptr)
	if share {
		d := NewDictValue(o.Elements, v.Immutable)
		(*Dict)(d.Ptr).IsView = true
		return d
	}
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		c[k] = e
	}
	return NewDictValue(c, false)
}

func recordTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Record:
		return mapsEqual((*Record)(v.Ptr).Elements, (*Record)(other.Ptr).Elements)
	case value.Dict:
		return mapsEqual((*Record)(v.Ptr).Elements, (*Dict)(other.Ptr).Elements)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func recordTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Record:
		switch op {
		case tokens.Add:
			return NewRecordValue(mergeMaps((*Record)(v.Ptr).Elements, (*Record)(other.Ptr).Elements), false), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func recordTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	// r.f(args) compiles as a member call, so a callable field is called from here.
	e, ok := recordField(v, name)
	if !ok {
		return CallMemberByLookup(vm, v, name, args)
	}
	if !e.IsCallable() {
		return Undefined, errs.NewRecoverableError(errs.KindNotCallable, fmt.Sprintf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName()))
	}
	return e.Call(vm, args)
}

// recordField is the one field lookup behind every record member and index path: r.f(...), r.f, r["f"].
func recordField(v Value, name string) (Value, bool) {
	e, ok := (*Record)(v.Ptr).Elements[name]
	return e, ok
}

// recordSetField is the one field write behind r.f = x and r["f"] = x.
func recordSetField(v Value, name string, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	(*Record)(v.Ptr).Elements[name] = r
	return nil
}

// PURE by contract — a missing field answers undefined
func recordTypeAccessNamedProperty(_ VM, v Value, name string) (Value, error) {
	e, _ := recordField(v, name)
	return e, nil
}

// PURE by contract — r[k] is r.k for a string k
func recordTypeAccessIndex(v Value, index Value) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	e, _ := recordField(v, k)
	return e, nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func recordTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Record)(v.Ptr).Elements), nil
}

func recordTypeIsTrue(v Value) (bool, error) {
	return len((*Record)(v.Ptr).Elements) > 0, nil
}

func recordTypeLen(v Value) int64 {
	o := (*Record)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes a field into the receiver. Not folded by the optimizer. See docs/purity.md.
func recordTypeAssignNamedProperty(_ VM, v Value, name string, r Value) error {
	return recordSetField(v, name, r)
}

// IMPURE: writes a field into the receiver; r[k] = x is r.k = x for a string k. See docs/purity.md.
func recordTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}
	return recordSetField(v, k, r)
}

// recordTypeContains is the `in` operator on the KEY axis, exactly dict's rule.
func recordTypeContains(v Value, e Value) (bool, error) {
	return mapContainsKey((*Record)(v.Ptr).Elements, e)
}

// mutate=true: IMPURE, removes a field from the receiver in place (the free delete_in_place() builtin — record
// has no CallNamedMethod switch, so this is only ever reached that way, never as a member call). Not folded by the
// optimizer. mutate=false: PURE, returns an independent record without the key (the free delete() builtin),
// leaving the receiver untouched — works regardless of the receiver's mutability. See docs/purity.md.
func recordTypeDelete(v Value, key Value, mutate bool) (Value, error) {
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("remove_in_place", v.TypeName())
		}
		delete((*Record)(v.Ptr).Elements, s)
		return v, nil
	}

	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		if k != s {
			c[k] = e
		}
	}
	return NewRecordValue(c, false), nil
}

func recordTypeAsBool(v Value) (bool, bool) {
	return len((*Record)(v.Ptr).Elements) > 0, true
}

func recordTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Record)(v.Ptr).Elements, true
}
