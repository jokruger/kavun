package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/jokruger/kavun/bc"
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

// IntRangeValue creates boxed int-range value.
func IntRangeValue(v *IntRange) Value {
	return Value{
		Type:      VT_INT_RANGE,
		Immutable: true,
		Ptr:       unsafe.Pointer(v),
	}
}

// NewIntRangeValue creates a new (heap-allocated) int-range value.
func NewIntRangeValue(start, stop, step int64) Value {
	t := &IntRange{}
	t.Set(start, stop, step)
	return IntRangeValue(t)
}

var TypeIntRange = ValueType{
	Name:         ConstHook(intRangeTypeName),
	EncodeBinary: intRangeTypeEncodeBinary,
	DecodeBinary: intRangeTypeDecodeBinary,
	String:       intRangeTypeString,
	Format:       intRangeTypeFormat,
	IsTrue:       intRangeTypeIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     intRangeTypeIterator,
	Equal:        intRangeTypeEqual,
	Len:          intRangeTypeLen,
	MethodCall:   intRangeTypeMethodCall,
	Access:       intRangeTypeAccess,
	Contains:     intRangeTypeContains,
	AsBool:       intRangeTypeAsBool,
	AsArray:      intRangeTypeAsArray,
}

func intRangeTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
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

func intRangeTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
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
	o := &IntRange{
		Start: start,
		Stop:  stop,
		Step:  step,
	}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func intRangeTypeString(a *Arena, v Value) string {
	o := (*IntRange)(v.Ptr)
	if o.Step == 1 {
		return fmt.Sprintf("range(%d, %d)", o.Start, o.Stop)
	}
	return fmt.Sprintf("range(%d, %d, %d)", o.Start, o.Stop, o.Step)
}

func intRangeTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return intRangeTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(intRangeTypeName, sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(intRangeTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(intRangeTypeString(a, v), sp, fspec.AlignLeft), nil
}

func intRangeTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_INT_RANGE {
		return false
	}

	x := (*IntRange)(v.Ptr)
	y := (*IntRange)(r.Ptr)
	return *x == *y
}

func intRangeTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := intRangeTypeAsArray(a, v)
		return a.NewArrayValue(t, false), nil

	case "bytes":
		return intRangeFnToBytes(a, v, args)

	case "string":
		return intRangeFnToString(a, v, args)

	case "record":
		return intRangeFnToRecord(a, v, args)

	case "dict":
		return intRangeFnToDict(a, v, args)

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := intRangeTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

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
		return BoolValue(intRangeTypeContains(a, v, args[0])), nil

	case "for_each":
		return intRangeFnForEach(a, vm, v, args)

	case "find":
		return intRangeFnFind(a, vm, v, args)

	case "join":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		elems, _ := intRangeTypeAsArray(a, v)
		if len(args) == 0 {
			s, err := joinElementsToString(a, elems, "")
			if err != nil {
				return Undefined, err
			}
			return a.NewStringValue(s), nil
		}
		return joinSeqWithSep(elems, args[0], vm, name)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func intRangeFnToBytes(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("bytes", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	bs := a.NewBytes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			bs[i] = byte(t)
			i++
			t += o.Step
		}
		return a.NewBytesValue(bs, false), nil
	}
	for t > o.Stop {
		bs[i] = byte(t)
		i++
		t -= o.Step
	}
	return a.NewBytesValue(bs, false), nil
}

func intRangeFnToString(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("string", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	rs := a.NewRunes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			rs[i] = rune(t)
			i++
			t += o.Step
		}
		return a.NewStringValue(string(rs)), nil
	}
	for t > o.Stop {
		rs[i] = rune(t)
		i++
		t -= o.Step
	}
	return a.NewStringValue(string(rs)), nil
}

func intRangeFnToRecord(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("record", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	m := a.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return a.NewRecordValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return a.NewRecordValue(m, false), nil
}

func intRangeFnToDict(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dict", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	m := a.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return a.NewDictValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return a.NewDictValue(m, false), nil
}

func intRangeFnForEach(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(a, args)
	if err != nil {
		return Undefined, err
	}

	o := (*IntRange)(v.Ptr)
	var buf [2]Value
	i := int64(0)
	t := o.Start

	call := func(value int64) (bool, error) {
		switch fn.Arity(a) {
		case 1:
			buf[0] = IntValue(value)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return false, err
			}
			return res.IsTrue(a), nil

		case 2:
			buf[0] = IntValue(i)
			buf[1] = IntValue(value)
			res, err := fn.Call(a, vm, buf[:2])
			if err != nil {
				return false, err
			}
			return res.IsTrue(a), nil
		}
		return false, nil
	}

	if o.Start <= o.Stop {
		for t < o.Stop {
			ok, err := call(t)
			if err != nil || !ok {
				return Undefined, err
			}
			i++
			t += o.Step
		}
		return Undefined, nil
	}
	for t > o.Stop {
		ok, err := call(t)
		if err != nil || !ok {
			return Undefined, err
		}
		i++
		t -= o.Step
	}
	return Undefined, nil
}

func intRangeFnFind(a *Arena, vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable(a) || fn.IsVariadic(a) {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName(a))
	}
	arity := fn.Arity(a)
	if arity != 1 && arity != 2 {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName(a))
	}

	o := (*IntRange)(v.Ptr)
	var buf [2]Value
	i := int64(0)
	t := o.Start

	call := func(value int64) (bool, error) {
		switch arity {
		case 1:
			buf[0] = IntValue(value)
			res, err := fn.Call(a, vm, buf[:1])
			if err != nil {
				return false, err
			}
			return res.IsTrue(a), nil

		case 2:
			buf[0] = IntValue(i)
			buf[1] = IntValue(value)
			res, err := fn.Call(a, vm, buf[:2])
			if err != nil {
				return false, err
			}
			return res.IsTrue(a), nil
		}
		return false, nil
	}

	if o.Start <= o.Stop {
		for t < o.Stop {
			ok, err := call(t)
			if err != nil {
				return Undefined, err
			}
			if ok {
				return IntValue(i), nil
			}
			i++
			t += o.Step
		}
		return Undefined, nil
	}
	for t > o.Stop {
		ok, err := call(t)
		if err != nil {
			return Undefined, err
		}
		if ok {
			return IntValue(i), nil
		}
		i++
		t -= o.Step
	}
	return Undefined, nil
}

func intRangeTypeAccess(a *Arena, v Value, index Value, mode bc.Opcode) (Value, error) {
	o := (*IntRange)(v.Ptr)

	if mode == bc.OpIndex {
		i, ok := index.AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName(a))
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

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(a), index.String(a))
}

func intRangeTypeIterator(a *Arena, v Value) (Value, error) {
	o := (*IntRange)(v.Ptr)
	return a.NewIntRangeIteratorValue(o.Start, o.Stop, o.Step), nil
}

func intRangeTypeIsTrue(a *Arena, v Value) bool {
	o := (*IntRange)(v.Ptr)
	return o.Start != o.Stop
}

func intRangeTypeAsBool(a *Arena, v Value) (bool, bool) {
	return intRangeTypeIsTrue(a, v), true
}

func intRangeTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := (*IntRange)(v.Ptr)
	arr := a.NewArray(int(o.Len()), true)
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

func intRangeTypeContains(a *Arena, v Value, e Value) bool {
	o := (*IntRange)(v.Ptr)
	i, ok := e.AsInt(a)
	if !ok {
		return false
	}
	return o.Contains(i)
}

func intRangeTypeLen(a *Arena, v Value) int64 {
	o := (*IntRange)(v.Ptr)
	return o.Len()
}
