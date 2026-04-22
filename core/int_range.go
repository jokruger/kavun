package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/jokruger/gs/errs"
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
		Ptr:  unsafe.Pointer(v),
		Type: VT_INT_RANGE,
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
	return fmt.Sprintf("range(%d, %d, %d)", o.Start, o.Stop, o.Step)
}

func intRangeTypeEqual(v Value, r Value) bool {
	if r.Type != VT_INT_RANGE {
		return false
	}

	a := (*IntRange)(v.Ptr)
	b := (*IntRange)(r.Ptr)
	return *a == *b
}

func intRangeTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*IntRange)(v.Ptr)
	return a.NewIntRangeValue(o.Start, o.Stop, o.Step)
}

func intRangeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		a := vm.Allocator()
		t, _ := intRangeTypeAsArray(v, a)
		return a.NewArrayValue(t, false)

	case "to_bytes":
		return intRangeFnToBytes(v, vm, args)

	case "to_string":
		return intRangeFnToString(v, vm, args)

	case "to_record":
		return intRangeFnToRecord(v, vm, args)

	case "to_map":
		return intRangeFnToMap(v, vm, args)

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

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func intRangeFnToBytes(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("to_bytes", "0", len(args))
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
		return vm.Allocator().NewBytesValue(bs)
	}
	for t > o.Stop {
		bs[i] = byte(t)
		i++
		t -= o.Step
	}
	return vm.Allocator().NewBytesValue(bs)
}

func intRangeFnToString(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("to_string", "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	bs := make([]rune, o.Len())
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			bs[i] = rune(t)
			i++
			t += o.Step
		}
		return vm.Allocator().NewStringValue(string(bs))
	}
	for t > o.Stop {
		bs[i] = rune(t)
		i++
		t -= o.Step
	}
	return vm.Allocator().NewStringValue(string(bs))
}

func intRangeFnToRecord(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("to_record", "0", len(args))
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
		return vm.Allocator().NewRecordValue(m, false)
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return vm.Allocator().NewRecordValue(m, false)
}

func intRangeFnToMap(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("to_map", "0", len(args))
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
		return vm.Allocator().NewMapValue(m, false)
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return vm.Allocator().NewMapValue(m, false)
}

func intRangeTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	o := (*IntRange)(v.Ptr)

	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		t, ok := o.Get(i)
		if !ok {
			return Undefined, nil
		}
		return IntValue(t), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func intRangeTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*IntRange)(v.Ptr)
	return a.NewIntRangeIteratorValue(o.Start, o.Stop, o.Step)
}

func intRangeTypeIsTrue(v Value) bool {
	o := (*IntRange)(v.Ptr)
	return o.Start != o.Stop
}

func intRangeTypeAsBool(v Value) (bool, bool) {
	return intRangeTypeIsTrue(v), true
}

func intRangeTypeAsArray(v Value, a Allocator) ([]Value, bool) {
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
