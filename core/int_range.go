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

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
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
	Name:         ConstHook(intRangeTypeName), // PURE by contract
	EncodeBinary: intRangeTypeEncodeBinary,    // PURE by contract
	DecodeBinary: intRangeTypeDecodeBinary,    // IMPURE by contract (mutates target)
	String:       intRangeTypeString,          // PURE by contract
	Format:       intRangeTypeFormat,          // PURE by contract
	IsTrue:       intRangeTypeIsTrue,          // PURE by contract
	IsIterable:   ConstHook(true),             // PURE by contract
	Iterator:     intRangeTypeIterator,        // PURE by contract (constructs fresh iterator)
	Equal:        intRangeTypeEqual,           // PURE by contract
	Len:          intRangeTypeLen,             // PURE by contract
	MethodCall:   intRangeTypeMethodCall,      // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       intRangeTypeAccess,          // PURE by contract
	Contains:     intRangeTypeContains,        // PURE by contract
	AsBool:       intRangeTypeAsBool,          // PURE by contract
	AsArray:      intRangeTypeAsArray,         // PURE by contract
	AsIntRange:   intRangeTypeAsIntRange,      // PURE by contract

	IsMethodPure: func(string) bool { return true }, // all methods are expected to be pure
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

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func intRangeTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, errs.FromFormatSpecError(name, err)
		}
		s, err := intRangeTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

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

	case "contains", "count", "any", "all":
		if name == "contains" && len(args) == 1 && args[0].Type == value.Int {
			// the element reading is a closed form, exactly the `in` operator's — nothing is scanned
			return BoolValue((*IntRange)(v.Ptr).Contains(int64(args[0].Data))), nil
		}
		elems, err := intRangeMaterialize(name, v)
		if err != nil {
			return Undefined, err
		}
		seq := Seq[int64]{Elements: elems}
		return SeqMatchMember(vm, name, v, args, IntValue, nil,
			func(Value) *Seq[int64] { return &seq },
			func(a Value) (int64, bool, error) {
				if a.Type != value.Int {
					return 0, false, errs.NewInvalidArgumentTypeError(name, "argument", "int (the element type), a predicate, or nothing", a.TypeName())
				}
				return int64(a.Data), true, nil
			},
			func(a Value) bool { return a.Type == value.Array || a.Type == value.IntRange },
			func(a Value) ([]int64, error) {
				return nil, errs.NewNotImplementedError("(" + name + ") the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
			},
			func(a, b int64) bool { return a == b },
			func(i int64) bool { return i == 0 })

	case "for_each":
		return intRangeFnForEach(vm, v, args)

	case "index", "index_last":
		// element | predicate | absent(blank {0}), plus [default]. The RUN reading
		// is deferred: it targets the vectorised int sequence type, which does not
		// exist yet, and is never approximated by an array
		elems, err := intRangeMaterialize(name, v)
		if err != nil {
			return Undefined, err
		}
		seq := Seq[int64]{Elements: elems}
		return SeqIndex(vm, v, args, name == "index_last", IntValue,
			func(Value) *Seq[int64] { return &seq },
			func(a Value) bool { return a.Type == value.Array || a.Type == value.IntRange },
			func(_ []int64, run Value, _ bool) (int64, bool, error) {
				return -1, false, errs.NewNotImplementedError("(" + name + ") the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
			},
			func(name string, a Value) error {
				if a.Type != value.Int {
					return errs.NewInvalidArgumentTypeError(name, "argument", "int (the element type), a predicate, or nothing", a.TypeName())
				}
				return nil
			},
			func(i int64) bool { return i == 0 })

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

	case "first", "last", "min", "max", "sum", "avg":
		// element answers and aggregation, all closed forms — the elements are an
		// arithmetic progression, so nothing is materialised
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		o := (*IntRange)(v.Ptr)
		n := o.Len()
		if n == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		first, last := intRangeFirstLast(o, n)
		switch name {
		case "first":
			return IntValue(first), nil
		case "last":
			return IntValue(last), nil
		case "min":
			return IntValue(min(first, last)), nil
		case "max":
			return IntValue(max(first, last)), nil
		}
		// sum and avg: checked int arithmetic like the array members — overflow raises, never wraps
		sum, err := intRangeSum(n, first, last)
		if err != nil {
			return Undefined, err
		}
		if name == "sum" {
			return IntValue(sum), nil
		}
		// avg — the same division the array member performs on int elements
		return IntValue(sum).BinaryOp(token.Quo, IntValue(n))

	case "reduce":
		elems, err := intRangeMaterialize(name, v)
		if err != nil {
			return Undefined, err
		}
		seq := Seq[int64]{Elements: elems}
		return SeqReduce(vm, v, args, IntValue, func(Value) *Seq[int64] { return &seq })

	case "reverse", "sort", "dedup", "unique":
		// closed forms on Start/Stop/Step, answering a RANGE — a lazy sequence
		// never answers a new sequence of its own elements in a materialised type.
		// dedup/unique are the identity: an arithmetic progression never repeats
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*IntRange)(v.Ptr)
		n := o.Len()
		if n == 0 || name == "dedup" || name == "unique" {
			return v, nil
		}
		first, last := intRangeFirstLast(o, n)
		// the result runs from `last` back to `first`, so its exclusive stop sits one past `first` — which no
		// int64 holds when `first` is itself an int64 bound: such a result cannot be encoded and raises
		switch {
		case name == "sort" && o.Start <= o.Stop:
			return v, nil // already ascending
		case o.Start > o.Stop: // sort or reverse of descending → ascending encoding
			if first == math.MaxInt64 {
				return Undefined, intRangeUnencodable(name, first)
			}
			return NewIntRangeValue(last, first+1, o.Step), nil
		default: // reverse of ascending → descending encoding
			if first == math.MinInt64 {
				return Undefined, intRangeUnencodable(name, first)
			}
			return NewIntRangeValue(last, first-1, o.Step), nil
		}

	case "slice":
		// clamps like every slice (reading past the end is harmless); negative
		// indices count from the end. A closed form: a sub-progression is a range
		s, e, err := seqOptionalSliceArgs(name, args)
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
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
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

func intRangeFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see SeqForEach)
	elems, err := intRangeMaterialize("for_each", v)
	if err != nil {
		return Undefined, err
	}
	var buf [2]Value
	for i, e := range elems {
		if fn.Arity() == 2 {
			buf[0] = IntValue(int64(i))
			buf[1] = IntValue(e)
			if _, err := fn.Call(vm, buf[:2]); err != nil {
				return Undefined, err
			}
		} else {
			buf[0] = IntValue(e)
			if _, err := fn.Call(vm, buf[:1]); err != nil {
				return Undefined, err
			}
		}
	}
	return v, nil
}

// PURE by contract
func intRangeTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	o := (*IntRange)(v.Ptr)

	if mode == bc.AccessIndex {
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

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
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
