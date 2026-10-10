package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/jokruger/kavun/core/member/members"
	"maps"
	"math"
	"math/big"
	"slices"
	"unsafe"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
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

	Methods: []MethodEntry{
		members.IsTrue:     {Fn: memberIsTrue, Pure: true},
		members.String:     {Fn: intRangeString, Pure: true},
		members.Format:     {Fn: memberFormat, Pure: true},
		members.Copy:       {Fn: memberSelf, Pure: true},
		members.Freeze:     {Fn: memberSelf, Pure: true},
		members.Runes:      {Fn: intRangeRunes, Pure: true},
		members.Array:      {Fn: intRangeArray, Pure: true},
		members.Bytes:      {Fn: intRangeBytes, Pure: true},
		members.Range:      {Fn: intRangeRange, Pure: true},
		members.Len:        {Fn: intRangeLen, Pure: true},
		members.IsEmpty:    {Fn: intRangeIsEmpty, Pure: true},
		members.Contains:   {Fn: intRangeContains, Pure: true},
		members.Index:      {Fn: intRangeIndex, Pure: true},
		members.Count:      {Fn: intRangeCount, Pure: true},
		members.All:        {Fn: intRangeAll, Pure: true},
		members.Any:        {Fn: intRangeAny, Pure: true},
		members.ForEach:    {Fn: intRangeForEach, Pure: true},
		members.Reduce:     {Fn: intRangeReduce, Pure: true},
		members.First:      {Fn: intRangeFirst, Pure: true},
		members.Last:       {Fn: intRangeLast, Pure: true},
		members.IndexLast:  {Fn: intRangeIndexLast, Pure: true},
		members.Min:        {Fn: intRangeMin, Pure: true},
		members.Max:        {Fn: intRangeMax, Pure: true},
		members.Slice:      {Fn: intRangeSlice, Pure: true},
		members.Reverse:    {Fn: intRangeReverse, Pure: true},
		members.Sort:       {Fn: intRangeSort, Pure: true},
		members.Unique:     {Fn: intRangeDedup, Pure: true},
		members.Dedup:      {Fn: intRangeDedup, Pure: true},
		members.Chunk:      {Fn: intRangeChunk, Pure: true},
		members.Join:       {Fn: intRangeJoin, Pure: true},
		members.Sum:        {Fn: intRangeSum, Pure: true},
		members.Avg:        {Fn: intRangeAvg, Pure: true},
		members.Components: {Fn: intRangeComponents, Pure: true},
	},
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
	significant bool    // no argument
	pred        Value   // a predicate
	elems       []int64 // a set of elements
}

// intRangeReadMatchArgs reads a match member's arguments. allowRuns is false for any/all, which never have a run
// reading; for contains/count the run reading exists but is not implemented yet.
func intRangeReadMatchArgs(name string, args []Value, allowRuns bool) (intRangeMatch, error) {
	if len(args) == 0 {
		return intRangeMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return intRangeMatch{}, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return intRangeMatch{}, err
		}
		return intRangeMatch{pred: args[0]}, nil
	}

	var m intRangeMatch
	firstRun := -1
	for i, a := range args {
		switch {
		case a.IsCallable():
			return intRangeMatch{}, errFunctionInSet(name)
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
		return intRangeMatch{}, errMixedSet(name)
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
func (m *intRangeMatch) matches(vm VM, i int, e int64) (bool, error) {
	switch {
	case m.significant:
		return e != 0, nil
	case m.pred.IsCallable():
		res, err := callElem(vm, m.pred, i, IntValue(e))
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
	elems, err := intRangeMaterialize(name, v)
	if err != nil {
		return Undefined, err
	}
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
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
		if err := checkElemCallback(name, needle); err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := callElem(vm, needle, i, IntValue(e))
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
// must be a valid code point (surrogates excluded) — a failing element fails the whole conversion (ok false), never a
// silent U+FFFD.
func intRangeCodePoints(name string, v Value) ([]rune, bool, error) {
	elems, err := intRangeMaterialize(name, v)
	if err != nil {
		return nil, false, err
	}
	rs := make([]rune, len(elems))
	for i, t := range elems {
		if t < 0 || t > 0x10FFFF || (t >= 0xD800 && t <= 0xDFFF) {
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
	i, ok = NormalizeIndex(i, o.Len())
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
// MaxSequenceLen. The loop runs by count, not by comparing against Stop: the value after the last element may
// lie past int64 and wrap, which a `t < Stop` loop would read as more elements.
func intRangeMaterialize(name string, v Value) ([]int64, error) {
	o := (*IntRange)(v.Ptr)
	n, err := SeqAllocLen(name, o.Len())
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
	n, err := SeqAllocLen(name, o.Len())
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

// intRangeTypeAsArray declines (false) a range past MaxSequenceLen: the hook has no error channel, so a caller
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
