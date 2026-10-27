package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

const intRangeTypeName = "range"

type IntRange struct {
	Start int64
	Stop  int64
	Step  int64
}

func (o *IntRange) Set(start, stop, step int64) {
	o.Start = start
	o.Stop = stop
	o.Step = step
}

func (o *IntRange) Empty() bool {
	return o.Start == o.Stop
}

// The arithmetic below works in uint64 wherever a distance between two int64s is taken: stop - start spans up to
// 2^64-1, which no int64 holds. An element offset i*step, on the other hand, may wrap in int64 freely — the true
// element lies between Start and Stop, so the two's-complement result is exact (Go defines signed wrap-around).

// intRangeSize answers how many elements start..stop by step holds, exactly; step must be positive.
func intRangeSize(start, stop, step int64) uint64 {
	var span uint64
	if start <= stop {
		span = uint64(stop) - uint64(start)
	} else {
		span = uint64(start) - uint64(stop)
	}
	n := span / uint64(step)
	if span%uint64(step) != 0 {
		n++
	}
	return n
}

// Len answers the element count. Exact for every range a script can hold: construction refuses a range whose
// count does not fit int64 (see NewIntRange).
func (o *IntRange) Len() int64 {
	return int64(intRangeSize(o.Start, o.Stop, o.Step))
}

// Get answers element i, 0 <= i < Len().
func (o *IntRange) Get(i int64) (int64, bool) {
	if i < 0 || i >= o.Len() {
		return 0, false
	}
	if o.Start <= o.Stop {
		return o.Start + i*o.Step, true
	}
	return o.Start - i*o.Step, true
}

func (o *IntRange) Contains(i int64) bool {
	if o.Start <= o.Stop {
		return i >= o.Start && i < o.Stop && (uint64(i)-uint64(o.Start))%uint64(o.Step) == 0
	}
	return i <= o.Start && i > o.Stop && (uint64(o.Start)-uint64(i))%uint64(o.Step) == 0
}

// NewIntRange is the checked constructor behind every script-side spelling (range(a, b[, step]), a..b, the
// components map, the binary decoder): the step must be positive, and the element count must fit int64 — a range
// is an int sequence, so its len() must be an int. NewIntRangeValue stays the unchecked Go-side builder.
// PURE by contract.
func NewIntRange(start, stop, step int64) (Value, error) {
	if step <= 0 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("range step must be greater than 0, got %d", step))
	}
	if intRangeSize(start, stop, step) > math.MaxInt64 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(range) %s holds more elements than an int can count",
			intRangeSource(start, stop, step)))
	}
	return NewIntRangeValue(start, stop, step), nil
}

func NewIntRangeValue(start, stop, step int64) Value {
	o := &IntRange{}
	o.Set(start, stop, step)
	return Value{Type: value.IntRange, Immutable: true, Ptr: unsafe.Pointer(o)}
}

// NewStaticIntRangeValue wraps a range backed by the compiler's static pool (see compiler/static.go), sharing the
// pool's storage directly instead of allocating a fresh IntRange. Safe because IntRange is always immutable and has
// no reachable mutable substructure — mirrors NewStaticDecimalValue/NewStaticTimeValue.
func NewStaticIntRangeValue(o *IntRange) Value {
	return Value{Type: value.IntRange, Immutable: true, Ptr: unsafe.Pointer(o)}
}

// Lazy integer sequence defined by `start`, `stop`, and `step`.
//
// ## Overview
//
// A `range` is an arithmetic progression of `int` values that is never stored: every element is computed from
// three components — `start` (inclusive), `stop` (exclusive), and `step` (always positive). Direction comes from
// the relation of `start` to `stop`, not from the sign of `step`:
//
// ```go
// range(2, 8)          // 2, 3, 4, 5, 6, 7        (ascending, step 1)
// range(1, 10, 2)      // 1, 3, 5, 7, 9           (ascending, step 2)
// range(10, 0, 3)      // 10, 7, 4, 1             (descending — start > stop)
// range(3, 3)          // empty
// ```
//
// `range` is always immutable, its elements are always `int`, and `len()`, membership, aggregation, and every
// transform are O(1) closed forms computed on the three components — nothing is ever materialized unless you ask
// for it with `.array()` (or another conversion).
//
// That laziness fixes the member roster: **no member of `range` answers a new materialized sequence of its own
// elements.** Every transform that exists (`slice`, `reverse`, `sort`, `dedup`, `unique`) is a closed form on
// `start`/`stop`/`step` and answers another `range`; `chunk(n)` answers an array *of* ranges. Anything that could
// produce arbitrary elements (`map`, `keep`, `append`, …) is deliberately absent — spell it `.array().map(...)`.
//
// The element answers (`first`, `last`, `min`, `max`) and the aggregations (`sum`, `avg`): all six carry the uniform
// optional trailing `[default]`: an empty range has no answer to give, so without a
// default they raise `invalid_value`.
//
// The closed-form transforms (`slice`, `reverse`, `sort`, `dedup`, `unique`): each answers a **`range`** — computed on
// the components, nothing materialized. The result's printed
// components may differ from what you would write by hand while generating exactly the intended elements.
//
// ## Construction
//
// ### Constructor forms
//
// ```go
// range()              // the empty range — range(0, 0)
// range(2, 8)          // start, stop; step defaults to 1
// range(1, 10, 2)      // start, stop, step
// ```
//
// - There is **no one-argument count form**: one argument is the components record, so `range(5)` raises
// `invalid_argument_type` (`argument components expects type dict or record, got int`). Write `range(0, 5)`.
// - `step` must be greater than zero; `range(1, 10, 0)` and `range(1, 10, -2)` raise `invalid_value`
// (`range step must be greater than 0`). Descending sequences come from `start > stop`, never from a
// negative step.
// - The element count must fit an `int`: a range is an int sequence, so its `len()` is an int.
// `range(-9223372036854775807, 9223372036854775807)` holds 2⁶⁴−2 elements and raises `invalid_value`
// (`… holds more elements than an int can count`) — in every spelling (`a..b`, `range(rec)` too). With a step of
// 2 or more the same bounds are fine: `range(-9223372036854775807, 9223372036854775807, 2).len()` is
// `9223372036854775807`. Every closed form on a range that constructed is exact up to the int64 edges.
//
// ### Materializing members and the sequence limit
//
// A range is lazy, but some members need its elements one by one: `array()`, `join()`, `string()`/`runes()`/
// `bytes()`, `for_each`, `reduce`, `index`/`index_last`, and the set/predicate/blank readings of
// `contains`/`count`/`any`/`all`, as well as `chunk(n)`'s array of ranges. These hold to the same ceiling as
// `repeat` (`4294967296` elements, see [vm.md](../../vm.md)) and raise `invalid_value` past it instead of
// allocating — `range(0, 1 << 62).array()` raises `(array) result would be 4611686018427387904 elements, past
// the 4294967296 limit`. The free constructors `array(r)`/`string(r)`/`runes(r)` decline such a range as a
// `conversion` error. `len()`, indexing, `in`, `contains(x)`, `first`/`last`/`min`/`max`/`sum`/`avg` and the
// transforms never materialize and have no such limit.
//
// ### The `..` literal
//
// `a..b` is the literal form of `range(a, b)` (step 1, either direction), and `a..b:s` of `range(a, b, s)`:
//
// ```go
// 1..4                 // => range(1, 4) — 1, 2, 3
// 5..1                 // => range(5, 1) — 5, 4, 3, 2
// 1..10:3              // => range(1, 10, 3) — 1, 4, 7
// (1..4).array()       // => [1, 2, 3] — parenthesize before calling a member
// ```
//
// The bounds and the step can be any expression (`1..n+2`, `a..b:s`); `..` binds looser than `+`. The literal is
// the constructor call itself, so the step follows the same rule — greater than zero — and a descending sequence is
// still spelled with `start > stop`: `10..1:3` is `10, 7, 4`. See [language.md](../../language.md) for the grammar.
//
// ### From a components record
//
// `range(rec)` rebuilds a range from a record with keys `start`, `stop`, and optionally `step`; an unknown key
// raises. `components()` is the reverse direction. A `dict` with the same keys is accepted too.
//
// ```go
// range({start: 1, stop: 10, step: 2})        // => range(1, 10, 2)
// range({start: 1, stop: 5})                  // => range(1, 5) — step defaults to 1
// range({start: 1, stop: 5, bogus: 1})        // Error: invalid_value: unknown component "bogus"
//
// r := range(1, 10, 2)
// format(r.components())                      // => "{\"start\": 1, \"step\": 2, \"stop\": 10}"
// range(r.components()) == r                  // => true — the round trip
// ```
//
// `components()` answers a `record`, which has no member surface — render it with the free `format(x)` and read
// fields with `rec.start` etc.
//
// ## Iteration and indexing
//
// `for x in r` yields the elements lazily, in the range's direction:
//
// ```go
//
//	acc := []
//	for x in range(10, 0, 3) {
//	    acc = acc + x
//	}
//	// acc == [10, 7, 4, 1]
//
// ```
//
// Single elements are O(1)-indexable, with negative indices counting from the end:
//
// ```go
// r := range(1, 10)
// r[2]                 // => 3
// r[-1]                // => 9
// r[99]                // Error: index_out_of_bounds
// ```
//
// Slice *syntax* is not supported — `r[1:3]` raises `not_sliceable`. The member `slice(i, j)` is the spelling,
// and unlike the syntax form it clamps instead of raising.
//
// ## Operators
//
// ### Equality
//
// Ranges compare by their **definition** — the `start`/`stop`/`step` components — not by the element sequence
// they generate:
//
// ```go
// range(1, 4) == range(1, 4)          // => true
// range(1, 4) == range(1, 4, 1)       // => true — step 1 is the default, same definition
// range(1, 4) != range(1, 5)          // => true
// range(1, 11, 3) == range(1, 12, 3)  // => false — both generate 1, 4, 7, 10, but the definitions differ
// range(1, 4) == [1, 2, 3]            // => false — never equal to another type
// ```
//
// Ordering (`<`, `<=`, …) is not defined between ranges and raises.
//
// ### The operators a range does not have
//
// `+`, `-` and `*` are **not defined** on a range and raise (`range + int`, `range - int`, `range * int`) — a
// range is a read-only formula, so there is nothing to append to, remove from, or repeat. Nor does a range
// spread into another sequence's operator: it is **one element**, like any other non-`array` value
// (`[9] + (1..4)` → `[9, range(1, 4)]`). Materialize it first when you want the elements — `[9] + (1..4).array()`
// → `[9, 1, 2, 3]`. The members mirror this exactly: a range has `contains`, and no `append`/`remove`/`repeat`.
//
// ### Membership: `x in r`
//
// An `int` operand is a closed-form membership test — arithmetic on the components, no materialization:
//
// ```go
// 3 in range(1, 10, 2)     // => true
// 4 in range(1, 10, 2)     // => false
// ```
//
// A sequence operand (`array` or `range`) asks for the *run* reading, which is deferred on `range` and raises
// saying so (`the run reading on a range is deferred until the vectorised integer sequence type exists; write
// .array() explicitly`). A callable operand raises too — an operator operand is always a value; the predicate
// reading is spelled `r.contains(f)` / `r.any(f)`:
//
// ```go
// [2, 3] in range(1, 10)                    // Error: not_implemented — deferred (not_implemented)
// (2..4) in range(1, 10)                    // Error: not_implemented — deferred
// func(x) { return x > 2 } in range(1, 10)  // Error: invalid_value — write r.any(fn)
// ```
//
// ### A range as the other side's operand
//
// On `array`'s side a `range` is **one element**, like every value that is not itself an `array` — a range is
// never spread implicitly. Materializing is spelled at the call site, and then it is an ordinary array:
//
// ```go
// [9] + range(1, 4)                  // => [9, range(1, 4)] — one element
// [9] + range(1, 4).array()          // => [9, 1, 2, 3] — spelled
// [1, 2, 3] - range(2, 4)            // => [1, 2, 3] — no element equals the range
// [1, 2, 3] - range(2, 4).array()    // => [1] — removes the run 2, 3
// ```
//
// The reason is forward-looking: a range that quietly became an `array` would answer an `array` today and an
// `ints` tomorrow — silently, in a script that named neither. It is the same rule that keeps `map`/`keep` off
// this type.
//
// `range` itself has no add (or remove) operator — a range cannot be extended and stay a closed form:
//
// ```go
// range(1, 4) + 5             // Error: invalid_binary_operator: range + int
// range(1, 4) + range(4, 7)   // Error: invalid_binary_operator: range + range
// ```
//
// `range + array` is the one exception, and it is the array's doing, not the range's: the range has no reading
// for an array, so it hands the operation over and the array prepends it as one element.
//
// ```go
// range(1, 4) + [1]           // => [range(1, 4), 1]
// ```
//
// On the text types (`string`/`runes`/`bytes`), a `range` argument or operand raises — they accept text content
// only. Convert first: `bytes("ab") + range(1, 4).bytes()`.
//
// ## Argument dispatch
//
// Argument-taking search members (`contains`, `count`, `any`, `all`, `index`, `index_last`) share one reading
// menu: an **`int` element**, a **variadic set** of ints (match = "∈ the set"), a **predicate function** ([callbacks](#callbacks)), or
// **no argument** — the blank-set reading, which for `range` is the set `{0}` ("significant" = non-zero). All
// readings are closed forms. A sequence argument (`array` or `range`) would be the run reading, which is
// **deferred** on `range` and raises telling you to write `.array()` explicitly — except on `any`/`all`, where a
// run argument is refused **permanently** (there is no universal reading of "any equals this run"). Mixing a
// function with other arguments in one variadic call raises.
//
// ## Callbacks
//
// A member that takes a function — the predicate reading of the search members, `for_each`, `reduce` — calls it in the
// **fewest form it accepts**, read from the callable's arity and whether it is variadic, never from what kind of
// callable it is:
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
// range(0, 6).count(func(i, x) { return i % 2 == 0 })     // => 3 — accepts two, not one
// range(0, 10).index(func(x, ...rest) { return x > 6 })   // => 7 — accepts one: the element
// range(1, 5).reduce(0, max)                              // => 4
// ```
//
//kavun:type range

var TypeIntRange = ValueTypeDescr{
	Name:                ConstHook(intRangeTypeName), // PURE by contract
	EncodeBinary:        intRangeTypeEncodeBinary,    // PURE by contract
	DecodeBinary:        intRangeTypeDecodeBinary,    // IMPURE by contract (mutates target)
	String:              intRangeTypeString,          // PURE by contract
	Format:              intRangeTypeFormat,          // PURE by contract
	IsTrue:              intRangeTypeIsTrue,          // PURE by contract
	IsIterable:          ConstHook(true),             // PURE by contract
	Iterator:            intRangeTypeIterator,        // PURE by contract (constructs fresh iterator)
	Equal:               intRangeTypeEqual,           // PURE by contract
	Len:                 intRangeTypeLen,             // PURE by contract
	AccessIndex:         intRangeTypeAccessIndex,     // PURE by contract
	AccessNamedProperty: noNamedProperty,             // PURE by contract
	Contains:            intRangeTypeContains,        // PURE by contract
	AsBool:              intRangeTypeAsBool,          // PURE by contract
	AsArray:             intRangeTypeAsArray,         // PURE by contract
	AsIntRange:          intRangeTypeAsIntRange,      // PURE by contract

	Methods: intRangeMethods,
}

func intRangeTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*IntRange)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Start); err != nil {
		return nil, fmt.Errorf("int-range (start): %w", err)
	}
	if err := enc.Encode(o.Stop); err != nil {
		return nil, fmt.Errorf("int-range (stop): %w", err)
	}
	if err := enc.Encode(o.Step); err != nil {
		return nil, fmt.Errorf("int-range (step): %w", err)
	}
	return buf.Bytes(), nil
}

func intRangeTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var start int64
	if err := dec.Decode(&start); err != nil {
		return fmt.Errorf("int-range (start): %w", err)
	}
	var stop int64
	if err := dec.Decode(&stop); err != nil {
		return fmt.Errorf("int-range (stop): %w", err)
	}
	var step int64
	if err := dec.Decode(&step); err != nil {
		return fmt.Errorf("int-range (step): %w", err)
	}
	r, err := NewIntRange(start, stop, step)
	if err != nil {
		return fmt.Errorf("int-range: %w", err)
	}
	*v = r
	return nil
}

func intRangeTypeString(v Value) string {
	o := (*IntRange)(v.Ptr)
	return intRangeSource(o.Start, o.Stop, o.Step)
}

// intRangeSource is a range's source form, its constructor call.
func intRangeSource(start, stop, step int64) string {
	if step == 1 {
		return fmt.Sprintf("range(%d, %d)", start, stop)
	}
	return fmt.Sprintf("range(%d, %d, %d)", start, stop, step)
}

func intRangeTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return intRangeTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(intRangeTypeName, sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(intRangeTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(intRangeTypeString(v), sp, fspec.AlignLeft), nil
}

func intRangeTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.IntRange:
		x := (*IntRange)(v.Ptr)
		y := (*IntRange)(other.Ptr)
		return *x == *y
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// ---------------------------------------------------------------------------
// The match members: contains / count / any / all. A range has no keep/remove (a lazy sequence never answers a
// materialised sequence of its own elements). The readings:
//   - no argument      the significant elements (not 0)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an int           one element; several form a set
//   - array or range   a run: deferred until the vectorised integer sequence type exists, so it raises
// ---------------------------------------------------------------------------

// intRangeMatch is a match member's argument list after reading; exactly one reading is set.
type intRangeMatch struct {
	significant bool          // no argument
	pred        *ElemCallback // a predicate
	elems       []int64       // a set of elements
}

// intRangeReadMatchArgs reads a match member's arguments. allowRuns is false for any/all, which never have a run
// reading; for contains/count the run reading exists but is not implemented yet.
func intRangeReadMatchArgs(vm VM, name string, args []Value, allowRuns bool) (intRangeMatch, error) {
	if len(args) == 0 {
		return intRangeMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return intRangeMatch{}, seq.ErrPredicateAmongMany(name)
		}
		cb, err := newElemCallback(vm, name, args[0])
		if err != nil {
			return intRangeMatch{}, err
		}
		return intRangeMatch{pred: cb}, nil
	}

	var m intRangeMatch
	firstRun := -1
	for i, a := range args {
		switch {
		case a.IsCallable():
			return intRangeMatch{}, seq.ErrFunctionInSet(name)
		case a.Type == value.Array || a.Type == value.IntRange:
			if firstRun < 0 {
				firstRun = i
			}
		case a.Type == value.Int:
			m.elems = append(m.elems, int64(a.Data))
		default:
			return intRangeMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "int (the element type), a predicate, or nothing", a.TypeName())
		}
	}
	if firstRun >= 0 && m.elems != nil {
		return intRangeMatch{}, seq.ErrMixedSet(name)
	}
	if firstRun >= 0 {
		if !allowRuns {
			return intRangeMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a predicate (this member declares no run reading)", args[firstRun].TypeName())
		}
		return intRangeMatch{}, errs.NewNotImplementedError("(" + name + ") the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
	}
	return m, nil
}

// matches reports whether element e at index i matches the reading.
func (m *intRangeMatch) matches(i int, e int64) (bool, error) {
	switch {
	case m.significant:
		return e != 0, nil
	case m.pred != nil:
		res, err := m.pred.Call(i, IntValue(e))
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	return slices.Contains(m.elems, e), nil
}

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match:
//   - no argument      the first / last significant element (not 0)
//   - a function       a predicate, f/1(element) or f/2(index, element)
//   - an int           one element, compared with ==
//   - array or range   a run: deferred until the vectorised integer sequence type exists, so it raises (never
//                      approximated by an array)
// A miss answers undefined, or the trailing default.
// ---------------------------------------------------------------------------

// intRangeLocate is the body of index and index_last; name is the member called, for the errors.
func intRangeLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
	}
	elems, err := intRangeMaterialize(name, v)
	if err != nil {
		return Undefined, err
	}

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if e != 0 {
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
			res, err := cb.Call(i, IntValue(e))
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

	if needle.Type == value.Array || needle.Type == value.IntRange {
		return Undefined, errs.NewNotImplementedError("(" + name + ") the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
	}

	if needle.Type != value.Int {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "argument", "int (the element type), a predicate, or nothing", needle.TypeName())
	}
	idx := -1
	for i, e := range elems {
		if IntValue(e).Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// ---------------------------------------------------------------------------
// The element answers and aggregates (first, last, min, max, sum, avg) and the reorderings (reverse, sort) — all
// closed forms on Start/Stop/Step: the elements are an arithmetic progression, so nothing is materialised, and a
// reordering answers a RANGE again. An empty range answers the trailing default or raises (emptySeqResult).
// ---------------------------------------------------------------------------

// intRangeReversed answers the non-empty range running from its last element back to its first. The result's
// exclusive stop sits one past the first element — which no int64 holds when that element is itself an int64
// bound: such a result cannot be encoded and raises.
func intRangeReversed(name string, o *IntRange, n int64) (Value, error) {
	first, last := intRangeFirstLast(o, n)
	if o.Start > o.Stop { // descending → ascending encoding
		if first == math.MaxInt64 {
			return Undefined, intRangeUnencodable(name, first)
		}
		return NewIntRangeValue(last, first+1, o.Step), nil
	}
	// ascending → descending encoding
	if first == math.MinInt64 {
		return Undefined, intRangeUnencodable(name, first)
	}
	return NewIntRangeValue(last, first-1, o.Step), nil
}

// intRangeFirstLast answers the first and last elements of a non-empty range — closed forms on Start/Stop/Step.
func intRangeFirstLast(o *IntRange, n int64) (int64, int64) {
	if o.Start <= o.Stop {
		return o.Start, o.Start + (n-1)*o.Step
	}
	return o.Start, o.Start - (n-1)*o.Step
}

// intRangeSub answers the sub-range of `count` elements starting at element offset `at` (both already validated
// against the length), keeping the source's direction and step — the closed form behind slice and chunk.
//
// Its exclusive stop is one step past its last element, saturated at the int64 bound when that step would leave
// int64: any stop after the last element and no further than one step past it encodes the same elements, and the
// last element of a sub-range lies strictly inside the source's bounds, so the bound itself always qualifies.
func intRangeSub(o *IntRange, at, count int64) Value {
	if count <= 0 {
		return NewIntRangeValue(o.Stop, o.Stop, o.Step)
	}
	start, _ := o.Get(at)
	last, _ := o.Get(at + count - 1)
	if o.Start <= o.Stop {
		stop := int64(math.MaxInt64)
		if last <= math.MaxInt64-o.Step {
			stop = last + o.Step
		}
		return NewIntRangeValue(start, stop, o.Step)
	}
	stop := int64(math.MinInt64)
	if last >= math.MinInt64+o.Step {
		stop = last - o.Step
	}
	return NewIntRangeValue(start, stop, o.Step)
}

// intRangeUnencodable is the error for a derived range whose exclusive stop would lie past int64: its end
// element `bound` is math.MinInt64 or math.MaxInt64 itself.
func intRangeUnencodable(name string, bound int64) error {
	return errs.NewInvalidValueError(fmt.Sprintf("(%s) the result ends at %d, so its exclusive stop lies past the int range", name, bound))
}

// intRangeProgressionSum is the closed-form sum of an arithmetic progression, n*(first+last)/2 — always a whole number —
// computed exactly and raising when it does not fit int64, like the checked int `+` the array member uses.
func intRangeProgressionSum(n, first, last int64) (int64, error) {
	s := new(big.Int).Add(big.NewInt(first), big.NewInt(last))
	s.Mul(s, big.NewInt(n))
	s.Quo(s, big.NewInt(2))
	if !s.IsInt64() {
		return 0, errs.NewInvalidValueError("int overflow")
	}
	return s.Int64(), nil
}

// intRangeCodePoints reads the elements as code points for string()/runes(): element-wise and all-or-nothing, each
// must be in the rune domain (a scalar value or an octet escape — the int → rune conversion's rule, which array
// shares) — a failing element fails the whole conversion (ok false), never a silent U+FFFD.
func intRangeCodePoints(name string, v Value) ([]rune, bool, error) {
	elems, err := intRangeMaterialize(name, v)
	if err != nil {
		return nil, false, err
	}
	rs := make([]rune, len(elems))
	for i, t := range elems {
		if !text.IntInRuneDomain(t) {
			return nil, false, nil
		}
		rs[i] = rune(t)
	}
	return rs, true, nil
}

// PURE by contract
func intRangeTypeAccessIndex(v Value, index Value) (Value, error) {
	o := (*IntRange)(v.Ptr)
	i, ok := index.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
	}
	i, ok = seq.NormalizeIndex(i, o.Len())
	if !ok {
		return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), int(o.Len()))
	}
	t, ok := o.Get(i)
	if !ok {
		return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), int(o.Len()))
	}
	return IntValue(t), nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func intRangeTypeIterator(v Value) (Value, error) {
	o := (*IntRange)(v.Ptr)
	return NewIntRangeIteratorValue(o.Start, o.Stop, o.Step), nil
}

// RangeFromComponents rebuilds a range from {start, stop[, step]}: start and stop are required, step defaults
// to 1, an unknown key raises. The way back from r.components().
func RangeFromComponents(m map[string]Value) (Value, error) {
	// sorted, so several unknown keys always name the same one (see TimeFromComponents)
	for _, k := range slices.Sorted(maps.Keys(m)) {
		switch k {
		case "start", "stop", "step":
		default:
			return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(range) unknown component %q", k))
		}
	}
	get := func(key string) (int64, bool, error) {
		v, ok := m[key]
		if !ok {
			return 0, false, nil
		}
		i, ok := v.AsInt()
		if !ok {
			return 0, false, errs.NewInvalidArgumentTypeError("range", key, "int", v.TypeName())
		}
		return i, true, nil
	}
	start, ok, err := get("start")
	if err != nil {
		return Undefined, err
	}
	if !ok {
		return Undefined, errs.NewInvalidValueError("(range) component start is required")
	}
	stop, ok, err := get("stop")
	if err != nil {
		return Undefined, err
	}
	if !ok {
		return Undefined, errs.NewInvalidValueError("(range) component stop is required")
	}
	step, ok, err := get("step")
	if err != nil {
		return Undefined, err
	}
	if !ok {
		step = 1
	}
	return NewIntRange(start, stop, step)
}

// intRangeMaterialize expands the range's elements for members that need positional scans, holding the count to
// seq.MaxSequenceLen. The loop runs by count, not by comparing against Stop: the value after the last element may
// lie past int64 and wrap, which a `t < Stop` loop would read as more elements.
func intRangeMaterialize(name string, v Value) ([]int64, error) {
	o := (*IntRange)(v.Ptr)
	n, err := seq.AllocLen(name, o.Len())
	if err != nil {
		return nil, err
	}
	elems := make([]int64, n)
	t, step := o.Start, o.Step
	if o.Start > o.Stop {
		step = -step
	}
	for i := range elems {
		elems[i] = t
		t += step
	}
	return elems, nil
}

// intRangeElements is intRangeMaterialize answering the elements as values — the range's array form — built
// directly rather than through an []int64.
func intRangeElements(name string, v Value) ([]Value, error) {
	o := (*IntRange)(v.Ptr)
	n, err := seq.AllocLen(name, o.Len())
	if err != nil {
		return nil, err
	}
	arr := make([]Value, n)
	t, step := o.Start, o.Step
	if o.Start > o.Stop {
		step = -step
	}
	for i := range arr {
		arr[i] = IntValue(t)
		t += step
	}
	return arr, nil
}

func intRangeTypeIsTrue(v Value) (bool, error) {
	o := (*IntRange)(v.Ptr)
	return o.Start != o.Stop, nil
}

func intRangeTypeAsBool(v Value) (bool, bool) {
	t, err := intRangeTypeIsTrue(v)
	if err != nil {
		return false, false
	}
	return t, true
}

func intRangeTypeAsIntRange(v Value) (IntRange, bool) {
	return *(*IntRange)(v.Ptr), true
}

// intRangeTypeAsArray declines (false) a range past seq.MaxSequenceLen: the hook has no error channel, so a caller
// that converts a range must check ok rather than read an empty array.
func intRangeTypeAsArray(v Value) ([]Value, bool) {
	arr, err := intRangeElements("array", v)
	return arr, err == nil
}

// intRangeTypeContains is the `in` operator: an int is the element (a closed form on
// Start/Stop/Step); the run reading is deferred with the member's own wording; a callable raises.
func intRangeTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	switch e.Type {
	case value.Int:
		return (*IntRange)(v.Ptr).Contains(int64(e.Data)), nil
	case value.Array, value.IntRange:
		return false, errs.NewNotImplementedError("(in) the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
	}
	return false, errs.NewInvalidArgumentTypeError("in", "operand", "int (the element type)", e.TypeName())
}

func intRangeTypeLen(v Value) int64 {
	o := (*IntRange)(v.Ptr)
	return o.Len()
}
