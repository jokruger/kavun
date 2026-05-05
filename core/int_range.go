package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
)

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
		Type:  VT_INT_RANGE,
		Const: true,
		Ptr:   unsafe.Pointer(v),
	}
}

// NewIntRangeValue creates a new (heap-allocated) int-range value.
func NewIntRangeValue(start, stop, step int64) Value {
	t := &IntRange{}
	t.Set(start, stop, step)
	return IntRangeValue(t)
}

/* IntRange type methods */

func intRangeTypeName(v Value) string {
	return "range"
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
	o := &IntRange{
		Start: start,
		Stop:  stop,
		Step:  step,
	}
	v.Ptr = unsafe.Pointer(o)
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
	if err := format.ValidateContainerSpec("range", sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(intRangeTypeString(v), sp, fspec.AlignLeft), nil
}

func intRangeTypeEqual(v Value, r Value) bool {
	if r.Type != VT_INT_RANGE {
		return false
	}

	a := (*IntRange)(v.Ptr)
	b := (*IntRange)(r.Ptr)
	return *a == *b
}

func intRangeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
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
		a := vm.Allocator()
		t, _ := intRangeTypeAsArray(v, a)
		return a.NewArrayValue(t, false), nil

	case "bytes":
		return intRangeFnToBytes(v, vm, args)

	case "string":
		return intRangeFnToString(v, vm, args)

	case "record":
		return intRangeFnToRecord(v, vm, args)

	case "dict":
		return intRangeFnToDict(v, vm, args)

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
		return intRangeFnForEach(v, vm, args)

	case "find":
		return intRangeFnFind(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func intRangeFnToBytes(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("bytes", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	alloc := vm.Allocator()
	bs := alloc.NewBytes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			bs[i] = byte(t)
			i++
			t += o.Step
		}
		return alloc.NewBytesValue(bs, false), nil
	}
	for t > o.Stop {
		bs[i] = byte(t)
		i++
		t -= o.Step
	}
	return alloc.NewBytesValue(bs, false), nil
}

func intRangeFnToString(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("string", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	alloc := vm.Allocator()
	rs := alloc.NewRunes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			rs[i] = rune(t)
			i++
			t += o.Step
		}
		return alloc.NewStringValue(string(rs)), nil
	}
	for t > o.Stop {
		rs[i] = rune(t)
		i++
		t -= o.Step
	}
	return alloc.NewStringValue(string(rs)), nil
}

func intRangeFnToRecord(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("record", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	alloc := vm.Allocator()
	m := alloc.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return alloc.NewRecordValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return alloc.NewRecordValue(m, false), nil
}

func intRangeFnToDict(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dict", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	alloc := vm.Allocator()
	m := alloc.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return alloc.NewDictValue(m, false), nil
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return alloc.NewDictValue(m, false), nil
}

func intRangeFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := forEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*IntRange)(v.Ptr)
	var buf [2]Value
	i := int64(0)
	t := o.Start

	call := func(value int64) (bool, error) {
		switch fn.Arity() {
		case 1:
			buf[0] = IntValue(value)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return false, err
			}
			return res.IsTrue(), nil

		case 2:
			buf[0] = IntValue(i)
			buf[1] = IntValue(value)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return false, err
			}
			return res.IsTrue(), nil
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

func intRangeFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}
	arity := fn.Arity()
	if arity != 1 && arity != 2 {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}

	o := (*IntRange)(v.Ptr)
	var buf [2]Value
	i := int64(0)
	t := o.Start

	call := func(value int64) (bool, error) {
		switch arity {
		case 1:
			buf[0] = IntValue(value)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return false, err
			}
			return res.IsTrue(), nil

		case 2:
			buf[0] = IntValue(i)
			buf[1] = IntValue(value)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return false, err
			}
			return res.IsTrue(), nil
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

func intRangeTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	o := (*IntRange)(v.Ptr)

	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		i, ok = normalizeSequenceIndex(i, o.Len())
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

func intRangeTypeIterator(v Value, a *Arena) (Value, error) {
	o := (*IntRange)(v.Ptr)
	return a.NewIntRangeIteratorValue(o.Start, o.Stop, o.Step), nil
}

func intRangeTypeIsTrue(v Value) bool {
	o := (*IntRange)(v.Ptr)
	return o.Start != o.Stop
}

func intRangeTypeAsBool(v Value) (bool, bool) {
	return intRangeTypeIsTrue(v), true
}

func intRangeTypeAsArray(v Value, a *Arena) ([]Value, bool) {
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
