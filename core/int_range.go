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

	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token/tokens"
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

// intRangeCount answers how many elements start..stop by step holds, exactly; step must be positive.
func intRangeCount(start, stop, step int64) uint64 {
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
	return int64(intRangeCount(o.Start, o.Stop, o.Step))
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
	if intRangeCount(start, stop, step) > math.MaxInt64 {
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
	CallNamedMethod:     intRangeTypeCallNamedMethod, // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessIndex:         intRangeTypeAccessIndex,     // PURE by contract
	AccessNamedProperty: noNamedProperty,             // PURE by contract
	Contains:            intRangeTypeContains,        // PURE by contract
	AsBool:              intRangeTypeAsBool,          // PURE by contract
	AsArray:             intRangeTypeAsArray,         // PURE by contract
	AsIntRange:          intRangeTypeAsIntRange,      // PURE by contract

	IsNamedMethodPure: func(string) bool { return true }, // all methods are expected to be pure
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

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func intRangeTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value regardless of copy depth
		return v, nil

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable already, so freeze/freeze_shallow are no-ops
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, err := intRangeElements(name, v)
		if err != nil {
			return Undefined, err
		}
		return NewArrayValue(t, false), nil

	case "bytes":
		return intRangeFnToBytes(v, args)

	case "range":
		return convMember(name, intRangeTypeName, args, true, v)

	case "components":
		// the constitutive parts, as a record — exactly what range(rec) rebuilds from
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*IntRange)(v.Ptr)
		return NewRecordValue(map[string]Value{
			"start": IntValue(o.Start),
			"stop":  IntValue(o.Stop),
			"step":  IntValue(o.Step),
		}, false), nil

	case "string":
		return intRangeFnToString(v, args)

	case "runes":
		res, err := intRangeFnToString(v, nil)
		if err != nil {
			return Undefined, err
		}
		str, _ := res.AsString()
		return convMember(name, intRangeTypeName, args, true, NewRunesValue([]rune(str), false))

	case "format":
		return memberFormat(vm, v, members.Format, args)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*IntRange)(v.Ptr)
		return BoolValue(o.Start == o.Stop), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*IntRange)(v.Ptr)
		return IntValue(o.Len()), nil

	case "contains":
		return intRangeContainsMember(vm, v, args)

	case "count":
		return intRangeCountMember(vm, v, args)

	case "any":
		return intRangeAny(vm, v, args)

	case "all":
		return intRangeAll(vm, v, args)

	case "for_each":
		return intRangeForEach(vm, v, args)

	case "index":
		return intRangeIndex(vm, v, args)

	case "index_last":
		return intRangeIndexLast(vm, v, args)

	case "join":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		elems, err := intRangeElements(name, v)
		if err != nil {
			return Undefined, err
		}
		if len(args) == 0 {
			s, err := joinElementsToString(elems, "")
			if err != nil {
				return Undefined, err
			}
			return NewStringValue(s), nil
		}
		return joinSeqWithSep(elems, args[0], name)

	case "first":
		return intRangeFirst(vm, v, args)

	case "last":
		return intRangeLast(vm, v, args)

	case "min":
		return intRangeMin(vm, v, args)

	case "max":
		return intRangeMax(vm, v, args)

	case "sum":
		return intRangeSumMember(vm, v, args)

	case "avg":
		return intRangeAvg(vm, v, args)

	case "reduce":
		return intRangeReduce(vm, v, args)

	case "reverse":
		return intRangeReverse(vm, v, args)

	case "sort":
		return intRangeSort(vm, v, args)

	case "dedup", "unique":
		// the identity: an arithmetic progression never repeats
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "slice":
		// clamps like every slice (reading past the end is harmless); negative
		// indices count from the end. A closed form: a sub-progression is a range
		s, e, err := readSliceArgs(name, args)
		if err != nil {
			return Undefined, err
		}
		o := (*IntRange)(v.Ptr)
		si, ok := int64(0), true
		if s.Type != value.Undefined {
			if si, ok = s.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError(name, "int", s.TypeName())
			}
		}
		ei := int64(0)
		if e.Type != value.Undefined {
			if ei, ok = e.AsInt(); !ok {
				return Undefined, errs.NewInvalidIndexTypeError(name, "int", e.TypeName())
			}
		}
		si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, o.Len())
		return intRangeSub(o, si, ei-si), nil

	case "chunk":
		// array of ranges: the outer array holds no int elements, so nothing materialises
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		size, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
		}
		if size < 1 {
			return Undefined, errs.NewInvalidValueError("chunk size must be positive")
		}
		o := (*IntRange)(v.Ptr)
		n := o.Len()
		count := n / size
		if n%size != 0 {
			count++
		}
		if _, err := SeqAllocLen(name, count); err != nil {
			return Undefined, err
		}
		chunks := make([]Value, 0, count)
		for k := range count {
			at := k * size // < n, so it never overflows; at += size could
			chunks = append(chunks, intRangeSub(o, at, min(size, n-at)))
		}
		return NewArrayValue(chunks, false), nil

	default:
		return CallMemberByLookup(vm, v, name, args)
	}
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

// intRangeContainsMember is contains(...): is there a match anywhere? One int is answered in closed form, exactly
// like the `in` operator — nothing is scanned; every other reading walks the elements.
func intRangeContainsMember(vm VM, v Value, args []Value) (Value, error) {
	if len(args) == 1 && args[0].Type == value.Int {
		return BoolValue((*IntRange)(v.Ptr).Contains(int64(args[0].Data))), nil
	}
	elems, err := intRangeMaterialize("contains", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("contains", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// intRangeCountMember is count(...): how many elements match.
func intRangeCountMember(vm VM, v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("count", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("count", args, true)
	if err != nil {
		return Undefined, err
	}
	n := int64(0)
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// intRangeAny is any(...): does some element match?
func intRangeAny(vm VM, v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("any", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("any", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// intRangeAll is all(...): does every element match? True on an empty range.
func intRangeAll(vm VM, v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("all", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("all", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
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

// intRangeIndex is index(...): the first match.
func intRangeIndex(vm VM, v Value, args []Value) (Value, error) {
	return intRangeLocate(vm, "index", v, args, false)
}

// intRangeIndexLast is index_last(...): the last match.
func intRangeIndexLast(vm VM, v Value, args []Value) (Value, error) {
	return intRangeLocate(vm, "index_last", v, args, true)
}

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

// intRangeFirst is first([default]): the first element.
func intRangeFirst(_ VM, v Value, args []Value) (Value, error) {
	const name = "first"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, _ := intRangeFirstLast(o, n)
	return IntValue(first), nil
}

// intRangeLast is last([default]): the last element.
func intRangeLast(_ VM, v Value, args []Value) (Value, error) {
	const name = "last"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	_, last := intRangeFirstLast(o, n)
	return IntValue(last), nil
}

// intRangeMin is min([default]): the smallest element — one of the two ends.
func intRangeMin(_ VM, v Value, args []Value) (Value, error) {
	const name = "min"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	return IntValue(min(first, last)), nil
}

// intRangeMax is max([default]): the largest element — one of the two ends.
func intRangeMax(_ VM, v Value, args []Value) (Value, error) {
	const name = "max"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	return IntValue(max(first, last)), nil
}

// intRangeSumMember is sum([default]): the sum, in checked int arithmetic like the array member — overflow raises, never
// wraps.
func intRangeSumMember(_ VM, v Value, args []Value) (Value, error) {
	const name = "sum"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	sum, err := intRangeSum(n, first, last)
	if err != nil {
		return Undefined, err
	}
	return IntValue(sum), nil
}

// intRangeAvg is avg([default]): the sum divided by the count — the same division the array member performs on
// int elements.
func intRangeAvg(_ VM, v Value, args []Value) (Value, error) {
	const name = "avg"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	sum, err := intRangeSum(n, first, last)
	if err != nil {
		return Undefined, err
	}
	return IntValue(sum).BinaryOp(tokens.Quo, IntValue(n))
}

// intRangeReverse is reverse(): the same elements in the opposite order, as a range.
func intRangeReverse(_ VM, v Value, args []Value) (Value, error) {
	const name = "reverse"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return v, nil
	}
	return intRangeReversed(name, o, n)
}

// intRangeSort is sort(): the elements ascending, as a range — the receiver itself when it already ascends.
func intRangeSort(_ VM, v Value, args []Value) (Value, error) {
	const name = "sort"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 || o.Start <= o.Stop {
		return v, nil
	}
	return intRangeReversed(name, o, n)
}

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

// intRangeSum is the closed-form sum of an arithmetic progression, n*(first+last)/2 — always a whole number —
// computed exactly and raising when it does not fit int64, like the checked int `+` the array member uses.
func intRangeSum(n, first, last int64) (int64, error) {
	s := new(big.Int).Add(big.NewInt(first), big.NewInt(last))
	s.Mul(s, big.NewInt(n))
	s.Quo(s, big.NewInt(2))
	if !s.IsInt64() {
		return 0, errs.NewInvalidValueError("int overflow")
	}
	return s.Int64(), nil
}

// intRangeFnToBytes is element-wise and all-or-nothing, like every sequence conversion: an element outside
// the octet range fails the whole conversion (answering the optional default or raising) — it never wraps.
func intRangeFnToBytes(v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("bytes", v)
	if err != nil {
		return Undefined, err
	}
	bs := make([]byte, len(elems))
	ok := true
	for i, t := range elems {
		if t < 0 || t > 255 {
			ok = false
			break
		}
		bs[i] = byte(t)
	}
	return convMember("bytes", intRangeTypeName, args, ok, NewBytesValue(bs, false))
}

// intRangeFnToString is element-wise and all-or-nothing: each element must be a valid code point
// (surrogates excluded) — a failing element fails the whole conversion, never a silent U+FFFD.
func intRangeFnToString(v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("string", v)
	if err != nil {
		return Undefined, err
	}
	rs := make([]rune, len(elems))
	ok := true
	for i, t := range elems {
		if t < 0 || t > 0x10FFFF || (t >= 0xD800 && t <= 0xDFFF) {
			ok = false
			break
		}
		rs[i] = rune(t)
	}
	return convMember("string", intRangeTypeName, args, ok, NewStringValue(string(rs)))
}

// intRangeForEach is for_each(f): a full pass whose callback result is ignored; returns the receiver, so it chains.
func intRangeForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	elems, err := intRangeMaterialize("for_each", v)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		if _, err := callElem(vm, fn, i, IntValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// intRangeReduce is reduce(acc, f): folds the elements left to right.
func intRangeReduce(vm VM, v Value, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("reduce", v)
	if err != nil {
		return Undefined, err
	}
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		acc, err = callReduce(vm, fn, acc, i, IntValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
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
