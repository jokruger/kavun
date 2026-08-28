package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"unsafe"

	bc "github.com/jokruger/kavun/core/bytecode"
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

func (o *IntRange) Len() int64 {
	if o.Start == o.Stop {
		return 0
	}
	if o.Start < o.Stop {
		return (o.Stop - o.Start + o.Step - 1) / o.Step
	}
	return (o.Start - o.Stop + o.Step - 1) / o.Step
}

func (o *IntRange) Get(i int64) (int64, bool) {
	if o.Start <= o.Stop {
		t := o.Start + i*o.Step
		if t >= o.Stop {
			return 0, false
		}
		return t, true
	}
	t := o.Start - i*o.Step
	if t <= o.Stop {
		return 0, false
	}
	return t, true
}

func (o *IntRange) Contains(i int64) bool {
	if o.Start <= o.Stop {
		return i >= o.Start && i < o.Stop && (i-o.Start)%o.Step == 0
	}
	return i <= o.Start && i > o.Stop && (o.Start-i)%o.Step == 0
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
	*v = NewIntRangeValue(start, stop, step)
	return nil
}

func intRangeTypeString(v Value) string {
	o := (*IntRange)(v.Ptr)
	if o.Step == 1 {
		return fmt.Sprintf("range(%d, %d)", o.Start, o.Stop)
	}
	return fmt.Sprintf("range(%d, %d, %d)", o.Start, o.Stop, o.Step)
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
		t, _ := intRangeTypeAsArray(v)
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
			return Undefined, err
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

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(intRangeTypeContains(v, args[0])), nil

	case "for_each":
		return intRangeFnForEach(vm, v, args)

	case "index", "index_last":
		// element | predicate | absent(blank {0}), plus [default]. The RUN reading
		// is deferred: it targets the vectorised int sequence type, which does not
		// exist yet, and is never approximated by an array
		elems := intRangeMaterialize(v)
		seq := Seq[int64]{Elements: elems}
		return SeqIndex(vm, v, args, name == "index_last", IntValue,
			func(Value) *Seq[int64] { return &seq },
			func(a Value) bool { return a.Type == value.Array || a.Type == value.IntRange },
			func(_ []int64, run Value, _ bool) (int64, bool, error) {
				return -1, false, errs.NewNotImplementedError("(" + name + ") the run reading on a range is deferred until the vectorised integer sequence type exists; write .array() explicitly")
			},
			func(i int64) bool { return i == 0 })

	case "join":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		elems, _ := intRangeTypeAsArray(v)
		if len(args) == 0 {
			s, err := joinElementsToString(elems, "")
			if err != nil {
				return Undefined, err
			}
			return NewStringValue(s), nil
		}
		return joinSeqWithSep(elems, args[0], name)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func intRangeFnToBytes(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("bytes", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	bs := make([]byte, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			bs[i] = byte(t)
			i++
			t += o.Step
		}
		return NewBytesValue(bs, false), nil
	}
	for t > o.Stop {
		bs[i] = byte(t)
		i++
		t -= o.Step
	}
	return NewBytesValue(bs, false), nil
}

func intRangeFnToString(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("string", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	rs := make([]rune, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			rs[i] = rune(t)
			i++
			t += o.Step
		}
		return NewStringValue(string(rs)), nil
	}
	for t > o.Stop {
		rs[i] = rune(t)
		i++
		t -= o.Step
	}
	return NewStringValue(string(rs)), nil
}

func intRangeFnToRecord(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("record", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	m := make(map[string]Value, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return NewRecordValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return NewRecordValue(m, false), nil
}

func intRangeFnToDict(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dict", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	m := make(map[string]Value, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return NewDictValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return NewDictValue(m, false), nil
}

func intRangeFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see SeqForEach)
	var buf [2]Value
	for i, e := range intRangeMaterialize(v) {
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
	for k := range m {
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
	if step <= 0 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("range step must be greater than 0, got %d", step))
	}
	return NewIntRangeValue(start, stop, step), nil
}

// intRangeMaterialize expands the range's elements for members that need positional scans.
func intRangeMaterialize(v Value) []int64 {
	o := (*IntRange)(v.Ptr)
	elems := make([]int64, 0, o.Len())
	if o.Start <= o.Stop {
		for t := o.Start; t < o.Stop; t += o.Step {
			elems = append(elems, t)
		}
	} else {
		for t := o.Start; t > o.Stop; t -= o.Step {
			elems = append(elems, t)
		}
	}
	return elems
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

func intRangeTypeAsArray(v Value) ([]Value, bool) {
	o := (*IntRange)(v.Ptr)
	arr := make([]Value, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			arr[i] = IntValue(t)
			i++
			t += o.Step
		}
		return arr, true
	}
	for t > o.Stop {
		arr[i] = IntValue(t)
		i++
		t -= o.Step
	}
	return arr, true
}

func intRangeTypeContains(v Value, e Value) bool {
	o := (*IntRange)(v.Ptr)
	i, ok := e.AsInt()
	if !ok {
		return false
	}
	return o.Contains(i)
}

func intRangeTypeLen(v Value) int64 {
	o := (*IntRange)(v.Ptr)
	return o.Len()
}
