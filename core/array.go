package core

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/internal/seq"
)

const (
	arrayTypeName          = "array"
	immutableArrayTypeName = "immutable-array"
)

// Array is an array's body: its elements.
type Array struct {
	Elements []Value
	// IsView reports whether Elements shares backing storage with another value. Only the _view members
	// (slice_view, chunk_view) set it; every other constructor answers an independently-owned body.
	IsView bool
}

// Set replaces the elements — how every _in_place member writes its result back.
func (o *Array) Set(elements []Value) {
	o.Elements = elements
}

func NewArrayValue(arr []Value, immutable bool) Value {
	o := &Array{}
	o.Set(arr)
	return Value{Type: value.Array, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

// Mutable, heterogeneous sequences of values.
//
// ## Overview
//
// Arrays are ordered, mutable sequences whose elements can be values of any type, including other arrays.
//
// Arrays are reference-typed: `b = a` makes both names point at the same body; use `copy()` for an independent
// value. Every member without an `_in_place` suffix is non-mutating and returns a new array; the `_in_place` twins
// mutate the shared body (see [In-place twins](#in-place-twins)).
//
// The structural edits (`trim`, `pad_start`/`pad_end`, `replace`) are sequence verbs, not text verbs: a fill
// *element*, a set of *elements*, a run-or-element substitution.
//
// `first`, `last`, `min`, `max`, `sum`, `avg` all take the uniform optional trailing default. An **empty receiver
// has no answer to give**, so without a default they raise `invalid_value` — exactly like a failed conversion, and
// with the same one opt-out. `min`/`max` need mutually comparable elements; `sum`/`avg` need numeric ones. With
// all-`int` input `avg` is an `int` (integer division) — include a float for a fractional mean.
//
// ```go
// [1, 2, 3].first()        // => 1
// [].first()               // Error: invalid_value: (first) empty sequence
// [].first(0)              // => 0
// [].sum(0)                // => 0 — the empty sum is the caller's to name, not the language's
// [1, 2, 3].sum()          // => 6
// [1, 2].avg()             // => 1 — all-int mean is an int
// [1, 2.0].avg()           // => 1.5
// [undefined, 7].first()   // => undefined — first answers the plain first element, which may itself be undefined
// ```
//
// That last line is why the empty case raises rather than answering `undefined`: an element *can* be `undefined`,
// so a silent `undefined` made "the sequence was empty" and "the first element is undefined" indistinguishable.
//
// Container conversions are the element conversion applied element-wise, **all-or-nothing** — one inconvertible
// element fails the whole call.
//
// ## Literals and construction
//
// ```go
// uniform = [1, 2, 3]
// mixed = [1, "two", 3.0, true] // elements are any type
// ```
//
// An array literal is **constructed**, not constant: its elements are expressions evaluated when execution
// reaches them (`[f(x), i + 1]`), so each evaluation builds a fresh body — which is why an array literal is
// mutable while a `b"..."` / `u"..."` text literal is a shared, immutable constant. See [Constant literals and
// constructed literals](../../language.md#constant-literals-and-constructed-literals).
//
// The free `array(...)` constructor has two forms with two different jobs:
// - **Arity 1**: it decomposes a convertible sequence into elements and wraps any other value as one element.
// An array argument is *not* passed through — a constructor constructs, so `array(a)` is a new, independent,
// mutable array (a shallow copy, exactly `a.copy_shallow()`; `a.copy()` is the deep spelling).
// - **Arity 2**: `array(x, n)` is `n` copies of `x` as one element each and never spreads, whatever `x` is.
//
// ```go
// array()                  // => []
// array(range(1, 4))       // => [1, 2, 3]
// array("abc")             // => ['a', 'b', 'c'] — a string materializes as its runes
// array(5)                 // => [5] — a non-sequence value is one element
// array(-1)                // => [-1]
// array(undefined)         // Error: conversion: cannot convert undefined to array: value is missing
//
// // an array argument is copied, never aliased — and a frozen one comes back writable
// a = [1, 2]
// b = array(a)             // => [1, 2] — a new array
// b[0] = 9                 // b is [9, 2]; a is still [1, 2]
// array(freeze([1, 2]))    // => [1, 2] — mutable again (shallow: frozen ELEMENTS stay frozen)
//
// // the count form: n copies of x, x stays whole
// array("ab", 2)           // => ["ab", "ab"]
// array([1, 2], 2)         // => [[1, 2], [1, 2]] — never spreads
// array(undefined, 3)      // => [undefined, undefined, undefined] — the preallocation spelling
// ```
//
// ### Indexing and slicing
//
// ```go
// a = [10, 20, 30, 40, 50]
// a[0]             // => 10
// a[-1]            // => 50 — negative indices count from the end
// a[1:3]           // => [20, 30]
// a[::-1]          // => [50, 40, 30, 20, 10] — three-part slice with negative step
// a[7]             // Error: index_out_of_bounds: (index access) 7 out of range [0, 5]
// ```
//
// Slice bounds clamp; single-element access raises out of range. Index assignment (`a[0] = 99`) mutates in place
// and raises on an immutable array.
//
// ## Operators
//
// | operator | reading |
// | --- | --- |
// | `a + x` | exactly `append(x)`: an `array` operand spreads its elements in as a run; **every other value** is appended as one element; `undefined` propagates (`a + undefined` → `undefined`); an `error` operand raises |
// | `x + a` | exactly `prepend(x)` — the mirror, for a left operand that has no reading of its own for an array |
// | `a * n` | exactly `repeat(n)`: the right operand is a **count**, not an element. No reflected direction — `n * a` raises |
// | `a - x` | exactly `remove(x)`: an `array` operand removes every occurrence of that contiguous run — never set difference; every other value removes every equal element; `undefined` propagates |
// | `x in a` | the value readings of `contains`: element or run; a callable operand raises — an operator operand is always a value |
// | `==` / `!=` | deep equality, element by element, through any nesting |
// | `<` `>` `<=` `>=` | **not defined** — raises; an array has no total order. `sort()` orders the *elements*, which must be mutually comparable |
//
// ```go
// [1, 2] + [3, 4]          // => [1, 2, 3, 4]
// [1, 2] + 3               // => [1, 2, 3] — one element
// [1, 2] + "ab"            // => [1, 2, "ab"] — not an array: one element
// [9] + (1..4)             // => [9, range(1, 4)] — a range is one element too; see below
// [1, 2] + undefined       // => undefined
// [1, 2] + error("x")      // Error: invalid_binary_operator: array + error
// ```
//
// An array on the **right** works too, and means `prepend`. A left operand that has no reading of its own for an
// array hands the operation over, and the array adds it at the front:
//
// ```go
// 3 + [1, 2]               // => [3, 1, 2] — the mirror of [1, 2] + 3
// "ab" + [1, 2]            // => ["ab", 1, 2]
// (1..3) + [1]             // => [range(1, 3), 1]
// undefined + [1]          // => undefined — the universal contracts still come first
// error("x") + [1]         // Error: invalid_binary_operator: error + array
// ```
//
// Only `+` has this reflected form, because only the add side has a front member. Removal has no `prepend`
// counterpart, so every other operator raises rather than inventing one:
//
// ```go
// 3 - [1, 2]               // Error: invalid_binary_operator: int - array
// 3 * [1, 2]               // Error: invalid_binary_operator: int * array
// ```
//
// The operator adds **one** element at the front, and `+` is left-associative, so it does not chain the way
// `prepend` does — `1 + 2 + [3]` is `(1 + 2) + [3]` = `[3, 3]`. Use `prepend` for more than one:
//
// ```go
// [1].prepend(2, 3)        // => [2, 3, 1]
// 2 + 3 + [1]              // => [5, 1] — the ints add first
// ```
//
// `*` is `repeat`'s operator form — the right operand is a **count**, not an element:
//
// ```go
// [1, 2] * 3               // => [1, 2, 1, 2, 1, 2]
// [1, 2] * 0               // => []
// [1, 2] * 2.0             // => [1, 2, 1, 2] — a lossless numeric is a count
// [1, 2] * 2.5             // Error: invalid_value: (*) argument right operand must be a whole number
// [1, 2] * -1              // Error: invalid_value: (*) repeat count must be non-negative
// [1, 2] * [3]             // Error: invalid_binary_operator: array * array — only a number is a count
// 3 * [1, 2]               // Error: invalid_binary_operator: int * array
// ```
//
// There is **no** reflected direction here, unlike `+`: `a * n` reads as "apply `n` to the sequence", while
// `n * a` would read as "apply the sequence to `n`", which has no meaning. Element-wise arithmetic — multiplying
// each element by `n` — is a separate, future operator family (`.+`, `.-`, `.*`, `./`), never plain `*`.
//
// `-` removes by value, every occurrence:
//
// ```go
// [1, 2, 1, 2, 3] - 1      // => [2, 2, 3] — every equal element
// [1, 2, 3, 1, 2] - [1, 2] // => [3] — every occurrence of the contiguous run
// [1, 2, 3] - (1..3)       // => [1, 2, 3] — a range is one element, and no element equals it
// [1, [2], 1] - [2]        // => [1, [2], 1] — [2] means the element 2, which does not occur
// ```
//
// `in` accepts values only:
//
// ```go
// 2 in [1, 2, 3]                     // => true
// [2, 3] in [1, 2, 3]                // => true — a run
// undefined in [undefined, 1]        // => true
// [{a: [1]}] == [{a: [1]}]           // => true — deep equality, through any nesting
// f = func(x) { return true }
// f in [1, 2]    // Error: invalid_value — in) an operator operand is always a value — the predicate reading is contains(f)/any(f
// ```
//
// `+=`, `-=` and `*=` are the usual sugar: `a += 3` appends one element, `a -= 1` removes every `1`, `a *= 3`
// repeats.
//
// ## Argument dispatch
//
// Every member that searches, matches, or adds selects its *reading* from the argument's type — one menu, shared
// by members and operators alike:
//
// | argument | reading |
// | --- | --- |
// | absent | the [blank set](#the-blank-set) |
// | a function | a predicate — it gets the element, or `(index, element)` when it does not accept one argument ([callbacks](#callbacks)) |
// | an `array` — the receiver's own kind, and nothing else | a contiguous **run** of elements |
// | anything else, `range` and the text types included | **one element** |
// | a one-element array `[x]` | the element `x` — so wrapping is the uniform way to spell "this array, as one element": `[[2, 3]]` means the single element `[2, 3]` |
//
// **Only an `array` spreads — nothing else, however sequence-like.** A `range`, a `string`, a `runes`, a `bytes`
// are each one element. Spreading is never inferred from a type; it is spelled at the call site:
//
// ```go
// [9] + (1..4)                  // => [9, range(1, 4)]
// [9] + (1..4).array()          // => [9, 1, 2, 3]
// [1] + "ab"                    // => [1, "ab"]
// [1] + "ab".array()            // => [1, 'a', 'b']
// ```
//
// Two reasons, and they will still hold when the language grows typed vectors (`ints`, `floats`, …):
//
// - `string` and `runes` hold the *same* text and are interchangeable everywhere (see [runes](runes.md)),
// so neither can spread while the other stays an element — and `arr + "text"` must keep meaning *append the
// text*.
// - a `range` that quietly became an `array` would answer an `array` today and an `ints` tomorrow, silently, in a
// script that named neither. `.array()` puts the target where a reader — and a migration — can find it.
//
// ```go
// [1, 2, 3].contains(2)              // => true — element
// [1, 2, 3].contains([2, 3])         // => true — run
// [1, [2, 3], 4].contains([[2, 3]])  // => true — the wrap: the element [2, 3]
// [1, 2, 3].contains(func(x) { return x > 2 })   // => true — predicate
// ```
//
// A variadic call is a homogeneous **set** — "equals the argument" becomes "is in the set". Every argument in one
// call must have the same reading; mixing elements with runs raises, and a function among several arguments always
// raises:
//
// ```go
// [1, 2, 3].contains(4, 2)           // => true — any of the set
// [1, 2, 1, 3].count(1, 3)           // => 3
// [1, 2, 3].contains(9, [2, 3])      // Error: invalid_argument_type: a HOMOGENEOUS set — one reading per call
// ```
//
// Each member declares which readings it takes; a reading the member does not declare raises an error naming the
// ones it has (see `any`/`all` and `trim` below for members that refuse runs).
//
// Run matching is **leftmost-longest and non-overlapping**:
//
// ```go
// [1, 1, 1, 1].count([1, 1])                       // => 2 — non-overlapping
// ["a", "b", "c", "d"].remove(["a", "b"], ["a", "b", "c"])  // => ["d"] — longest wins at a tie
// ```
//
// The empty run matches nothing for counting and keeping verbs, but is "contained" everywhere:
//
// ```go
// [1, 2].contains([])      // => true
// [1, 2].count([])         // => 0
// [1, 2].remove([])        // => [1, 2]
// ```
//
// ## The blank set
//
// The no-argument form of `remove`/`keep`, `trim`/`trim_start`/`trim_end`, `contains`/`any`/`all`, `count`, and
// `index`/`index_last` matches the array's **blank set**: `undefined` together with each element type's own zero
// value — `0`, `0.0`, `""`, empty containers, and so on. Blankness is about separators and filler, not arithmetic:
// a script in which zeros are data names its own set instead.
//
// ```go
// [1, 0, undefined, "", 0.0, [], 2].remove()   // => [1, 2]
// [undefined, 0, 5, 0, undefined].trim(undefined)   // => [0, 5, 0] — zeros are data here
// ```
//
// The default pad fill is `undefined` — the blank set's canonical member.
//
// ## Copies, freezing, rendering, truthiness
//
// ```go
// a = [[1]]
// b = a.copy()
// b[0][0] = 9            // a is still [[1]]
// c = a.copy_shallow()
// c[0][0] = 7            // a is now [[7]] — nested values shared
//
// d = [1]
// f = d.freeze()
// d[0] = 9               // d is [9]; f stays [1]
// is_immutable(f)        // => true
//
// [0].is_true()          // => true — non-empty, even if every element is blank
// ```
//
// The free forms `len(x)`, `copy(x)`, `freeze(x)`, `format(x)`, `is_true(x)` are the same operations by another
// spelling.
//
// ## In-place twins
//
// Every mutating member is the `_in_place` twin of a non-mutating one with the same arguments and readings.
//
// A twin mutates the shared body — the change is visible through every alias, no reassignment needed — and
// returns the receiver, so twins chain and `y = x.m(args)` leaves the same content in `x` as `x.m_in_place(args)`.
// On a frozen receiver every twin raises kind `not_mutable`:
//
// ```go
// a = [1]
// b = a                        // b shares a's body
// a.append_in_place(2, 3)
// b                            // => [1, 2, 3] — no reassignment needed
//
// [1].freeze().append_in_place(2)   // Error: not_mutable: (append_in_place) type immutable-array is immutable
// ```
//
// ## Callbacks
//
// A member that takes a function — the predicate reading of the match members, `map`, `flat_map`, `for_each`, `reduce`
// — calls it in the **fewest form it accepts**, read from the callable's arity and whether it is variadic, never from
// what kind of callable it is:
//
// - a per-element callback gets the element when it accepts one argument, else `(index, element)`; anything else
// raises;
// - a `reduce` callback gets `(acc, element)` when it accepts two arguments, else `(acc, index, element)`; anything
// else raises.
//
// So `func(x, ...rest)` and `func(...args)` get the element, and a builtin or a bound member is a callback like any
// other (`min` and `"abc".contains` accept one argument). A lambda is the one way to call a callable in another form —
// choose the form, reorder, fix or drop arguments:
//
// ```go
// [5, 6].map(func(i, x) { return i })             // => [0, 1] — accepts two, not one
// [5, 6].map(func(x, ...rest) { return x * 2 })   // => [10, 12] — accepts one: the element
// [3, 1].map(min)                                 // => [3, 1]
// ["a", "z"].keep("abc".contains)                 // => ["a"]
// [5, 6].map((i, x) => [x, i])                    // => [[5, 0], [6, 1]] — a lambda picks the form and the order
// [1, 2].reduce(0, max)                           // => 2
// ```
//
//kavun:type array

var TypeArray = ValueTypeDescr{
	Name:         MutabilityNameHook(arrayTypeName, immutableArrayTypeName),                     // PURE by contract
	String:       arrayTypeString,                                                               // PURE by contract
	Format:       arrayTypeFormat,                                                               // PURE by contract
	Interface:    arrayTypeInterface,                                                            // PURE by contract
	EncodeJSON:   arrayTypeEncodeJSON,                                                           // PURE by contract
	EncodeBinary: arrayTypeEncodeBinary,                                                         // PURE by contract
	DecodeBinary: arrayTypeDecodeBinary,                                                         // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return len((*Array)(v.Ptr).Elements) > 0, nil }, // PURE by contract
	IsIterable:   ConstHook(true),                                                               // PURE by contract
	Iterator:     arrayTypeIterator,                                                             // PURE by contract (constructs fresh iterator)
	Equal:        arrayTypeEqual,                                                                // PURE by contract
	BinaryOp:     arrayTypeBinaryOp,                                                             // PURE by contract
	Copy:         arrayTypeCopy,                                                                 // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Array)(v.Ptr).Elements)) },           // PURE by contract
	Contains:     arrayTypeContains,                                                             // PURE by contract
	Append:       arrayTypeAppend,                                                               // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Slice:        arrayTypeSlice,                                                                // PURE by contract
	SliceStep:    arrayTypeSliceStep,                                                            // PURE by contract

	AccessIndex:         arrayTypeAccessIndex, // PURE by contract
	AccessNamedProperty: noNamedProperty,      // PURE by contract
	AssignIndex:         arrayTypeAssignIndex, // IMPURE by contract

	AsBool: func(v Value) (bool, bool) { return len((*Array)(v.Ptr).Elements) > 0, true }, // PURE by contract
	// No AsString hook: an array has no canonical text (its element-wise conversion is a transcoding
	// constructor, not a render), so a dict key, a join element, or any implicit to-string consumer raises
	// instead of silently keying/rendering the transcode. The explicit conversions stay: .string(), string(a).
	AsRunes: arrayTypeAsRunes,                                                        // PURE by contract
	AsBytes: arrayTypeAsBytes,                                                        // PURE by contract
	AsArray: func(v Value) ([]Value, bool) { return (*Array)(v.Ptr).Elements, true }, // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order methods
	// (keep/map/reduce/for_each/all/any/find/count) are pure in isolation — impurity can only enter via a
	// function-valued argument.

	Methods: arrayMethods,
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	parts := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		parts[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func arrayTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return arrayTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(arrayTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(arrayTypeString(v), sp, fspec.AlignLeft), nil
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var b []byte
	b = append(b, '[')
	len1 := len(o.Elements) - 1
	for idx, elem := range o.Elements {
		eb, err := elem.EncodeJSON()
		if err != nil {
			return nil, jsonPathPrefix(fmt.Sprintf("[%d]", idx), err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, ']')
	return b, nil
}

func arrayTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for i, elem := range o.Elements {
		eb, err := elem.EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", i, err)
		}
		b = binary.AppendBytes(b, eb)
	}

	return b, nil
}

func arrayTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "array (elements count)")
	if err != nil {
		return err
	}

	arr := make([]Value, int(count))
	for i := range arr {
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("array element at index %d", i))
		if err != nil {
			return err
		}
		if err := arr[i].DecodeBinary(eb); err != nil {
			return fmt.Errorf("array element at index %d: %w", i, err)
		}
	}
	if offset != len(data) {
		return fmt.Errorf("array: trailing %d bytes", len(data)-offset)
	}

	*v = NewArrayValue(arr, v.Immutable)
	return nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func arrayTypeIterator(v Value) (Value, error) {
	return NewArrayIteratorValue((*Array)(v.Ptr).Elements), nil
}

func arrayTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Array:
		l := (*Array)(v.Ptr).Elements
		r := (*Array)(other.Ptr).Elements
		if len(l) != len(r) {
			return false
		}
		for i, e := range l {
			if !e.Equal(r[i]) {
				return false
			}
		}
		return true
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func arrayTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		// the left operand had no reading for an array and handed the operation over. Only `+` has a
		// reflected form, because only the add side has a front spelling: `x + a` is exactly
		// `a.prepend(x)`, the mirror of `a + x` = `a.append(x)`. Removal has no front member, so `-`
		// — and every other operator — raises here rather than inventing one.
		if op != tokens.Add {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
		}
		// the universal contracts outrank the element reading on this side too: an error raises
		// through every operator rather than becoming the first element (undefined never reaches
		// here — it propagates without handing over)
		if other.Type == value.Error {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
		}
		l := (*Array)(v.Ptr)
		items := arrayAddItems([]Value{other})
		t := make([]Value, 0, len(items)+len(l.Elements))
		t = append(t, items...)
		t = append(t, l.Elements...)
		return NewArrayValue(t, false), nil
	}

	// the universal contracts outrank the element reading: undefined propagates through
	// every operator and error raises through every operator — appending either as an
	// element is the member spelling (push)
	if other.Type == value.Undefined || other.Type == value.Error {
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}

	l := (*Array)(v.Ptr)
	switch op {
	case tokens.Add:
		// exactly append's reading: an operand of the receiver's OWN KIND — another array —
		// contributes its elements as a run; anything else is one element
		items := arrayAddItems([]Value{other})
		t := make([]Value, 0, len(l.Elements)+len(items))
		t = append(t, l.Elements...)
		t = append(t, items...)
		return NewArrayValue(t, false), nil

	case tokens.Mul:
		// exactly repeat's reading: the right operand is a COUNT, not an element — a sequence
		// times a number is that sequence n times over. There is no reflected direction:
		// `seq * n` reads as "apply n to the sequence", `n * seq` has no such reading
		if n, isCount, err := SeqRepeatOperand(other); isCount {
			if err != nil {
				return Undefined, err
			}
			src := l.Elements
			sl := len(src)
			total, terr := seq.RepeatTotal(op.String(), n, sl)
			if terr != nil {
				return Undefined, terr
			}
			t := make([]Value, total)
			// step by the receiver's length, never by the count (see the member form)
			for i := 0; i < total; i += sl {
				copy(t[i:], src)
			}
			return NewArrayValue(t, false), nil
		}

	case tokens.Sub:
		// exactly remove's value readings: an array operand removes every occurrence of
		// the contiguous run (never set difference), anything else every equal element
		switch other.Type {
		case value.Array:
			runs := [][]Value{(*Array)(other.Ptr).Elements}
			kept := make([]Value, 0, len(l.Elements))
			for i := 0; i < len(l.Elements); {
				if k := arrayRunLengthAt(l.Elements, i, runs); k > 0 {
					i += k
				} else {
					kept = append(kept, l.Elements[i])
					i++
				}
			}
			return NewArrayValue(kept, false), nil
		}
		kept := make([]Value, 0, len(l.Elements))
		for _, e := range l.Elements {
			if !e.Equal(other) {
				kept = append(kept, e)
			}
		}
		return NewArrayValue(kept, false), nil
	}

	if other.Type != value.Array {
		return ValueTypes[other.Type].BinaryOp(other, v, op, true)
	}
	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
}

// deep=true recursively copies every element (today's copy() semantics); deep=false only clones the top-level
// slice header, leaving nested containers sharing the source (copy_shallow()).
func arrayTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Array)(v.Ptr)
	c := make([]Value, len(o.Elements))
	if !deep {
		copy(c, o.Elements)
		return NewArrayValue(c, false), nil
	}
	for i, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[i] = t
	}
	return NewArrayValue(c, false), nil
}

// PURE by contract
func arrayTypeAccessIndex(v Value, index Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	i, err := resolveIndex("index access", index, len(elems))
	if err != nil {
		return Undefined, err
	}
	return (elems[i]), nil
}

// IMPURE by contract: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func arrayTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	elems := (*Array)(v.Ptr).Elements
	i, err := resolveIndex("index assign", index, len(elems))
	if err != nil {
		return err
	}
	elems[i] = r
	return nil
}

// PURE by contract. a[i:j] always answers an independently-owned copy; sharing is slice_view's job.
func arrayTypeSlice(v Value, s Value, e Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds("slice", s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]Value, ei-si)
	copy(out, elems[si:ei])
	return NewArrayValue(out, false), nil
}

// PURE by contract
func arrayTypeSliceStep(v Value, s Value, e Value, step Value) (Value, error) {
	elems := (*Array)(v.Ptr).Elements
	start, end, st, err := resolveSliceStep(s, e, step, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]Value, 0)
	if st > 0 {
		for i := start; i < end; i += st {
			out = append(out, elems[i])
		}
	} else {
		for i := start; i > end; i += st {
			out = append(out, elems[i])
		}
	}
	return NewArrayValue(out, false), nil
}

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all.
//
// One name per operation; the ARGUMENT'S TYPE selects the reading:
//   - no argument      the significant elements (not IsBlankElement)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an array         a contiguous run (not for any/all)
//   - anything else    one element, compared with ==
//
// Several arguments form a SET of one reading (all elements, or all runs); mixing the two, or putting a function
// among them, raises. Runs match leftmost-longest and never overlap.
// ---------------------------------------------------------------------------

// arrayMatch is a match member's argument list after reading; exactly one reading is set.
type arrayMatch struct {
	significant bool          // no argument
	pred        *ElemCallback // a predicate
	elems       []Value       // a set of elements
	runs        [][]Value     // a set of runs
}

// arrayReadMatchArgs reads a match member's arguments. allowRuns is false for any/all: "some element is this
// subsequence" is contains' question, not theirs.
func arrayReadMatchArgs(vm VM, name string, args []Value, allowRuns bool) (arrayMatch, error) {
	if len(args) == 0 {
		return arrayMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return arrayMatch{}, seq.ErrPredicateAmongMany(name)
		}
		cb, err := newElemCallback(vm, name, args[0])
		if err != nil {
			return arrayMatch{}, err
		}
		return arrayMatch{pred: cb}, nil
	}

	var m arrayMatch
	firstRun := -1
	for i, a := range args {
		switch {
		case a.IsCallable():
			return arrayMatch{}, seq.ErrFunctionInSet(name)
		case a.Type == value.Array:
			if firstRun < 0 {
				firstRun = i
			}
			m.runs = append(m.runs, (*Array)(a.Ptr).Elements)
		default:
			m.elems = append(m.elems, a)
		}
	}
	if m.runs != nil && m.elems != nil {
		return arrayMatch{}, seq.ErrMixedSet(name)
	}
	if m.runs != nil && !allowRuns {
		return arrayMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a predicate (this member declares no run reading)", args[firstRun].TypeName())
	}
	return m, nil
}

// matches reports whether element e at index i matches an element-wise reading (anything but runs).
func (m *arrayMatch) matches(i int, e Value) (bool, error) {
	switch {
	case m.significant:
		return !IsBlankElement(e), nil
	case m.pred != nil:
		res, err := m.pred.Call(i, e)
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	for _, x := range m.elems {
		if e.Equal(x) {
			return true, nil
		}
	}
	return false, nil
}

// arrayRunLengthAt reports the length of the longest non-empty run in runs that matches elems at position i
// (0 when none does) — seq.RunLengthAt with Value equality.
func arrayRunLengthAt(elems []Value, i int, runs [][]Value) int {
	best := 0
	for _, r := range runs {
		if len(r) <= best || i+len(r) > len(elems) {
			continue
		}
		if arrayEqualRun(elems[i:i+len(r)], r) {
			best = len(r)
		}
	}
	return best
}

// arrayKept answers the elements keep(...) keeps: the matches, in order. Shared by keep and keep_in_place; name
// is the member called, for the errors.
func arrayKept(vm VM, name string, v Value, args []Value) ([]Value, error) {
	m, err := arrayReadMatchArgs(vm, name, args, true)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := arrayRunLengthAt(elems, i, m.runs); k > 0 {
				out = append(out, elems[i:i+k]...)
				i += k
			} else {
				i++
			}
		}
		return out, nil
	}

	for i, e := range elems {
		hit, err := m.matches(i, e)
		if err != nil {
			return nil, err
		}
		if hit {
			out = append(out, e)
		}
	}
	return out, nil
}

// arrayRemaining answers the elements remove(...) leaves: everything but the matches, in order. With no argument
// remove drops the BLANK elements — so it keeps the significant ones, landing on keep()'s answer by the opposite
// action. Shared by remove and remove_in_place; name is the member called, for the errors.
func arrayRemaining(vm VM, name string, v Value, args []Value) ([]Value, error) {
	m, err := arrayReadMatchArgs(vm, name, args, true)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))

	if m.significant {
		for _, e := range elems {
			if !IsBlankElement(e) {
				out = append(out, e)
			}
		}
		return out, nil
	}

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := arrayRunLengthAt(elems, i, m.runs); k > 0 {
				i += k
			} else {
				out = append(out, elems[i])
				i++
			}
		}
		return out, nil
	}

	for i, e := range elems {
		hit, err := m.matches(i, e)
		if err != nil {
			return nil, err
		}
		if !hit {
			out = append(out, e)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match. The
// argument's type selects the reading:
//   - no argument      the first / last significant element (not IsBlankElement)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an array         a contiguous run
//   - anything else    one element, compared with == (an array can hold one of anything)
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// arrayLocate is the body of index and index_last; name is the member called, for the errors.
func arrayLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
	}
	elems := (*Array)(v.Ptr).Elements

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !IsBlankElement(e) {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, nil)
	}

	needle, dflt := args[0], args[1:]

	if needle.IsCallable() {
		cb, err := newElemCallback(vm, name, needle)
		if err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := cb.Call(i, e)
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, dflt)
	}

	if needle.Type == value.Array {
		return locatorAnswer(arrayIndexRun(elems, (*Array)(needle.Ptr).Elements, last), dflt)
	}

	idx := -1
	for i, e := range elems {
		if e.Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// arrayIndexRun is seq.IndexRun with Value equality.
func arrayIndexRun(elems []Value, run []Value, last bool) int {
	n, m := len(elems), len(run)
	if m == 0 || m > n {
		return -1
	}
	idx := -1
	for i := 0; i+m <= n; i++ {
		if arrayEqualRun(elems[i:i+m], run) {
			idx = i
			if !last {
				break
			}
		}
	}
	return idx
}

// ---------------------------------------------------------------------------
// The structural members: the trim family, the anchored pair (has_/remove_ prefix/suffix), replace, and the pads.
// They are sequence verbs, not text verbs: an array argument is a run, anything else is one element; none of
// them takes a predicate.
//   - trim*            a set of single elements, stripped repeat-while; no argument = the blank set (IsBlankElement)
//   - *_prefix/suffix  a homogeneous set of runs; the longest anchored run wins; remove_* removes it once
//   - replace(old, new) every occurrence of the run old, leftmost and non-overlapping; an empty old matches nothing
//   - pad_*(n[, fill]) n counts elements; the fill is any one value, undefined by default
// ---------------------------------------------------------------------------

// arrayReadElementSet reads the trim family's set of single elements: an array argument or a function raises with
// refuse, the member's own statement of what it takes.
func arrayReadElementSet(name string, args []Value, refuse string) ([]Value, error) {
	for _, a := range args {
		if a.IsCallable() || a.Type == value.Array {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
	}
	return args, nil
}

// arrayReadRunSet reads the anchored pair's homogeneous set of runs: an array argument is a run, anything else a
// run of one element, but one call may not mix the two. A function raises with onFunction.
func arrayReadRunSet(name string, args []Value, onFunction string) ([][]Value, error) {
	runs := make([][]Value, 0, len(args))
	sawElement, sawRun := false, false
	for _, a := range args {
		switch {
		case a.IsCallable():
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		case a.Type == value.Array:
			sawRun = true
			runs = append(runs, (*Array)(a.Ptr).Elements)
		default:
			sawElement = true
			runs = append(runs, []Value{a})
		}
	}
	if sawElement && sawRun {
		return nil, seq.ErrMixedSet(name)
	}
	return runs, nil
}

// arrayEqualRun reports whether two runs hold equal elements, in order.
func arrayEqualRun(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// arrayTrimmed answers the receiver without its leading (start) and/or trailing (end) elements that belong to the
// set. The name is the member called, for the errors.
func arrayTrimmed(name string, v Value, args []Value, start, end bool) ([]Value, error) {
	set, err := arrayReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e Value) bool {
		if len(args) == 0 {
			return IsBlankElement(e)
		}
		for _, x := range set {
			if e.Equal(x) {
				return true
			}
		}
		return false
	}
	elems := (*Array)(v.Ptr).Elements
	lo, hi := 0, len(elems)
	if start {
		for lo < hi && inSet(elems[lo]) {
			lo++
		}
	}
	if end {
		for hi > lo && inSet(elems[hi-1]) {
			hi--
		}
	}
	return slices.Clone(elems[lo:hi]), nil
}

// arrayAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func arrayAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	runs, err := arrayReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := (*Array)(v.Ptr).Elements
	best := -1
	for _, r := range runs {
		if len(r) > len(elems) || len(r) <= best {
			continue
		}
		at := elems[:len(r)]
		if suffix {
			at = elems[len(elems)-len(r):]
		}
		if arrayEqualRun(at, r) {
			best = len(r)
		}
	}
	return best, nil
}

// arrayWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func arrayWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]Value, error) {
	best, err := arrayAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := (*Array)(v.Ptr).Elements
	switch {
	case best <= 0:
		return slices.Clone(elems), nil
	case suffix:
		return slices.Clone(elems[:len(elems)-best]), nil
	default:
		return slices.Clone(elems[best:]), nil
	}
}

// arrayReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); an
// array argument is a run, anything else a run of one element. The name is the member called, for the errors.
func arrayReplaced(name string, v Value, args []Value) ([]Value, error) {
	if args[0].IsCallable() || args[1].IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	asRun := func(a Value) []Value {
		if a.Type == value.Array {
			return (*Array)(a.Ptr).Elements
		}
		return []Value{a}
	}
	old, repl := asRun(args[0]), asRun(args[1])
	elems := (*Array)(v.Ptr).Elements
	out := make([]Value, 0, len(elems))
	for i := 0; i < len(elems); {
		if len(old) > 0 && i+len(old) <= len(elems) && arrayEqualRun(elems[i:i+len(old)], old) {
			out = append(out, repl...)
			i += len(old)
		} else {
			out = append(out, elems[i])
			i++
		}
	}
	return out, nil
}

// arrayPadded answers the receiver filled at the front (start) or the end up to n elements; a width at or below
// the length leaves it unchanged. The fill is any one value. The name is the member called, for the errors.
func arrayPadded(name string, v Value, args []Value, start bool) ([]Value, error) {
	if len(args) > 2 {
		return nil, ErrArgCount
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := Undefined
	if len(args) == 2 {
		fill = args[1]
	}
	elems := (*Array)(v.Ptr).Elements
	if n <= int64(len(elems)) {
		return slices.Clone(elems), nil
	}
	width, err := seq.PadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]Value{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(element) or
// f/2(index, element); reduce's is f/2(acc, element) or f/3(acc, index, element).
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// The add and edit members: chunk, slice_view, splice, insert, prepend, push, push_first (append is the Append
// hook, arrayTypeAppend; slice is the Slice hook's member spelling). append/prepend/splice's inserts read like
// the + operator (an array operand spreads, see arrayAddItems); push/push_first/insert never spread.
// ---------------------------------------------------------------------------

// arrayDeduped answers the elements with each run of equal neighbours collapsed to one.
func arrayDeduped(elems []Value) []Value {
	out := make([]Value, 0, len(elems))
	for i, e := range elems {
		if i == 0 || !out[len(out)-1].Equal(e) {
			out = append(out, e)
		}
	}
	return out
}

// arrayUniqueElements answers the elements without repeats, each kept at its first occurrence.
func arrayUniqueElements(elems []Value) []Value {
	out := make([]Value, 0, len(elems))
	for _, e := range elems {
		seen := false
		for _, u := range out {
			if u.Equal(e) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, e)
		}
	}
	return out
}

// arrayTypeContains is the `in` operator: contains' VALUE readings — an operand of the receiver's own
// FAMILY (array or range) is a contiguous run, anything else one element; the empty run is contained
// everywhere. A callable raises — an operator operand is always a value.
func arrayTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	o := (*Array)(v.Ptr)
	switch e.Type {
	case value.Array:
		runs := [][]Value{(*Array)(e.Ptr).Elements}
		if len(runs[0]) == 0 {
			return true, nil
		}
		for i := range o.Elements {
			if arrayRunLengthAt(o.Elements, i, runs) > 0 {
				return true, nil
			}
		}
		return false, nil
	}
	for i := range o.Elements {
		if o.Elements[i].Equal(e) {
			return true, nil
		}
	}
	return false, nil
}

// arrayEncodeStructuralArg reads a structural member's argument on an array receiver: an argument of the
// receiver's own kind (another array) is a run; anything else is one element. Callables never reach it —
// the classifiers dispatch them first (a predicate where the member declares one, a refusal elsewhere).
func arrayEncodeStructuralArg(_ string, a Value) ([]Value, bool, error) {
	if a.Type == value.Array {
		return (*Array)(a.Ptr).Elements, false, nil
	}
	return []Value{a}, true, nil
}

// arrayAddItems flattens append/prepend's variadic operands: an argument of the receiver's OWN KIND — another
// array — contributes its elements as a run, so the member agrees with the + operator; every other value is one
// element, an array can hold one of anything. A range is one element like the rest: materializing it is spelled
// at the call site (`a + r.array()`), never inferred, so a script that never names `array` never gets one. The
// element spelling for a nested array is push(row) or the wrap append([row]).
func arrayAddItems(args []Value) []Value {
	items := make([]Value, 0, len(args))
	for _, a := range args {
		if a.Type == value.Array {
			items = append(items, (*Array)(a.Ptr).Elements...)
			continue
		}
		items = append(items, a)
	}
	return items
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// array with the items appended (append()) — never touches the receiver's backing storage, works regardless of
// the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func arrayTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Array)(v.Ptr)
	items := arrayAddItems(args)

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("append_in_place", v.TypeName())
		}
		o.Set(append(o.Elements, items...))
		return v, nil
	}

	// Pure: build a fresh, independent array — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule; append(o.Elements, ...) would risk writing into o's own array).
	t := make([]Value, 0, len(o.Elements)+len(items))
	t = append(t, o.Elements...)
	t = append(t, items...)
	return NewArrayValue(t, false), nil
}

func arrayTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Array)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		r, ok := e.AsInt()
		if !ok || r < 0 || r > unicode.MaxRune {
			return nil, false
		}
		rs[i] = rune(r)
	}
	return rs, true
}

func arrayTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

// mutate=false: PURE, returns a fresh, independently-owned sorted array, source untouched. mutate=true: sorts
// the receiver's own backing storage directly, visible to every other alias sharing it; rejects an immutable
// receiver.
func arrayFnSort(name string, v Value, args []Value, mutate bool) (Value, error) {
	if mutate && v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}

	var err error
	o := (*Array)(v.Ptr)
	var t []Value
	if mutate {
		t = o.Elements
	} else {
		t = make([]Value, len(o.Elements))
		copy(t, o.Elements)
	}
	slices.SortFunc(t, func(x, y Value) int {
		less, e := x.BinaryOp(tokens.Less, y)
		if e != nil {
			err = e
			return 0
		}
		lt, e2 := less.IsTrue()
		if e2 != nil {
			err = e2
			return 0
		}
		if !lt {
			if x.Equal(y) {
				return 0
			}
			return 1
		}
		return -1
	})
	if err != nil {
		return Undefined, err
	}
	if mutate {
		return v, nil
	}

	return NewArrayValue(t, false), nil
}

// joinSeqWithSep performs the join given pre-resolved seq elements and a separator value.
// Returns a value whose type is determined by the sep type.
func joinSeqWithSep(elems []Value, sep Value, name string) (Value, error) {
	switch sep.Type {
	case value.String:
		s, err := joinElementsToString(elems, *(*string)(sep.Ptr))
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case value.Runes:
		rs, err := joinElementsToRunes(elems, (*Runes)(sep.Ptr).Elements)
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue(rs, false), nil

	case value.Rune:
		rs, err := joinElementsToRunes(elems, []rune{rune(sep.Data)})
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue(rs, false), nil

	case value.Byte:
		s, err := joinElementsToString(elems, string([]byte{byte(sep.Data)}))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	case value.Bytes:
		s, err := joinElementsToString(elems, string((*Bytes)(sep.Ptr).Elements))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes, bytes, byte, or rune", sep.TypeName())
	}
}

// flattenAppend appends each element of src to dst, unwrapping nested arrays up to `depth` levels.
// depth == 0 means no unwrapping (shallow copy).
// depth < 0 means unbounded (fully recursive).
func flattenAppend(dst []Value, src []Value, depth int) []Value {
	if depth == 0 {
		return append(dst, src...)
	}
	next := depth
	if next > 0 {
		next--
	}
	for _, e := range src {
		if e.Type == value.Array {
			inner := (*Array)(e.Ptr).Elements
			dst = flattenAppend(dst, inner, next)
		} else {
			dst = append(dst, e)
		}
	}
	return dst
}
