package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/jokruger/kavun/core/opcode"
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

func (a *Arena) MustNewIntRangeValue(start, stop, step int64) Value {
	v, err := a.NewIntRangeValue(start, stop, step)
	if err != nil {
		panic(err)
	}
	return v
}

func (a *Arena) NewIntRangeValue(start, stop, step int64) (Value, error) {
	if ref, p, ok := a.arena.New(value.IntRange); ok {
		(*IntRange)(p).Set(start, stop, step)
		return Value{Type: value.IntRange, Immutable: true, Data: ref}, nil
	}
	return Undefined, errs.NewAllocationLimitError(intRangeTypeName)
}

var TypeIntRange = ValueTypeDescr{
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

func intRangeTypeEncodeBinary(v Value) ([]byte, error) {
	o := a.ResolveIntRangeValue(v)
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
	o, err := a.NewIntRangeValue(start, stop, step)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o
	return nil
}

func intRangeTypeString(v Value) string {
	o := a.ResolveIntRangeValue(v)
	if o.Step == 1 {
		return fmt.Sprintf("range(%d, %d)", o.Start, o.Stop)
	}
	return fmt.Sprintf("range(%d, %d, %d)", o.Start, o.Stop, o.Step)
}

func intRangeTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
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

func intRangeTypeEqual(v Value, r Value) bool {
	if r.Type != value.IntRange {
		return false
	}

	x := a.ResolveIntRangeValue(v)
	y := a.ResolveIntRangeValue(r)
	return *x == *y
}

func intRangeTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
		return a.NewArrayValue(t, false)

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
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
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
		return a.NewStringValue(s)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveIntRangeValue(v)
		return BoolValue(o.Start == o.Stop), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveIntRangeValue(v)
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
			return a.NewStringValue(s)
		}
		return joinSeqWithSep(a, elems, args[0], name)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func intRangeFnToBytes(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("bytes", "0", len(args))
	}
	o := a.ResolveIntRangeValue(v)
	bs := a.NewBytes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			bs[i] = byte(t)
			i++
			t += o.Step
		}
		return a.NewBytesValue(bs, false)
	}
	for t > o.Stop {
		bs[i] = byte(t)
		i++
		t -= o.Step
	}
	return a.NewBytesValue(bs, false)
}

func intRangeFnToString(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("string", "0", len(args))
	}
	o := a.ResolveIntRangeValue(v)
	rs := a.NewRunes(int(o.Len()), true)
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			rs[i] = rune(t)
			i++
			t += o.Step
		}
		return a.NewStringValue(string(rs))
	}
	for t > o.Stop {
		rs[i] = rune(t)
		i++
		t -= o.Step
	}
	return a.NewStringValue(string(rs))
}

func intRangeFnToRecord(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("record", "0", len(args))
	}
	o := a.ResolveIntRangeValue(v)
	m := a.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return a.NewRecordValue(m, false)
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return a.NewRecordValue(m, false)
}

func intRangeFnToDict(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dict", "0", len(args))
	}
	o := a.ResolveIntRangeValue(v)
	m := a.NewDict(int(o.Len()))
	i := 0
	t := o.Start
	if o.Start <= o.Stop {
		for t < o.Stop {
			m[strconv.Itoa(i)] = IntValue(t)
			i++
			t += o.Step
		}
		return a.NewDictValue(m, false)
	}
	for t > o.Stop {
		m[strconv.Itoa(i)] = IntValue(t)
		i++
		t -= o.Step
	}
	return a.NewDictValue(m, false)
}

func intRangeFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(a, args)
	if err != nil {
		return Undefined, err
	}

	o := a.ResolveIntRangeValue(v)
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

func intRangeFnFind(vm VM, v Value, args []Value) (Value, error) {
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

	o := a.ResolveIntRangeValue(v)
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

func intRangeTypeAccess(v Value, index Value, mode opcode.Opcode) (Value, error) {
	o := a.ResolveIntRangeValue(v)

	if mode == opcode.Index {
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

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String(a))
}

func intRangeTypeIterator(v Value) (Value, error) {
	o := a.ResolveIntRangeValue(v)
	return a.NewIntRangeIteratorValue(o.Start, o.Stop, o.Step)
}

func intRangeTypeIsTrue(v Value) bool {
	o := a.ResolveIntRangeValue(v)
	return o.Start != o.Stop
}

func intRangeTypeAsBool(v Value) (bool, bool) {
	return intRangeTypeIsTrue(a, v), true
}

func intRangeTypeAsArray(v Value) ([]Value, bool) {
	o := a.ResolveIntRangeValue(v)
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
	o := a.ResolveIntRangeValue(v)
	i, ok := e.AsInt()
	if !ok {
		return false
	}
	return o.Contains(i)
}

func intRangeTypeLen(v Value) int64 {
	o := a.ResolveIntRangeValue(v)
	return o.Len()
}
