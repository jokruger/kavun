package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/token"
)

type Runes struct {
	Elements []rune
}

func (o *Runes) Set(r []rune) {
	o.Elements = r
	if o.Elements == nil {
		o.Elements = []rune{}
	}
}

// RunesValue creates new boxed runes value.
func RunesValue(v *Runes) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_RUNES,
	}
}

// NewRunesValue creates new (heap-allocated) runes value.
func NewRunesValue(v []rune) Value {
	o := &Runes{}
	o.Set(v)
	return RunesValue(o)
}

/* Runes type methods */

func runesTypeName(v Value) string {
	return "runes"
}

func runesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	var b []byte
	b = EncodeString(b, string(o.Elements))
	return b, nil
}

func runesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	s := string(o.Elements)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("runes: %w", err)
	}
	return buf.Bytes(), nil
}

func runesTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("runes: %w", err)
	}
	o := &Runes{Elements: []rune(s)}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func runesTypeString(v Value) string {
	o := (*Runes)(v.Ptr)
	return strconv.Quote(string(o.Elements))
}

func runesTypeInterface(v Value) any {
	o := (*Runes)(v.Ptr)
	return o.Elements
}

func runesTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsRunes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*Runes)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewRunesValue(append(o.Elements, r...))
	case token.Less:
		return BoolValue(string(o.Elements) < string(r)), nil
	case token.LessEq:
		return BoolValue(string(o.Elements) <= string(r)), nil
	case token.Greater:
		return BoolValue(string(o.Elements) > string(r)), nil
	case token.GreaterEq:
		return BoolValue(string(o.Elements) >= string(r)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func runesTypeEqual(v Value, r Value) bool {
	t, ok := r.AsRunes()
	if !ok {
		return false
	}
	o := (*Runes)(v.Ptr)
	return slices.Equal(o.Elements, t)
}

func runesTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*Runes)(v.Ptr)
	return a.NewRunesValue(o.Elements)
}

func runesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "to_runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(string(o.Elements))

	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false)

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runesTypeAsBool(v)
		return BoolValue(b), nil

	case "to_bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewBytesValue([]byte(string(o.Elements)))

	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := runesTypeAsFloat(v)
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := runesTypeAsInt(v)
		return IntValue(i), nil

	case "to_decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := runesTypeAsDecimal(v)
		return alloc.NewDecimalValue(d)

	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsTime(v)
		return alloc.NewTimeValue(t)

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := o.Elements
		m := make(map[string]Value, len(rs))
		for i, r := range rs {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewRecordValue(m, false)

	case "to_map":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := o.Elements
		m := make(map[string]Value, len(rs))
		for i, r := range rs {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewMapValue(m, false)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Elements) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Elements))), nil

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(o.Elements[0]), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return RuneValue(slices.Max(o.Elements)), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRunesValue([]rune(strings.ToLower(string(o.Elements))))

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRunesValue([]rune(strings.ToUpper(string(o.Elements))))

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(runesTypeContains(v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return alloc.NewRunesValue([]rune(strings.Trim(string(o.Elements), " \t\n")))
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string or runes", args[0].TypeName())
		}
		return alloc.NewRunesValue([]rune(strings.Trim(string(o.Elements), s)))

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := o.Elements
		sorted := make([]rune, len(rs))
		copy(sorted, rs)
		slices.Sort(sorted)
		return alloc.NewRunesValue(sorted)

	case "filter":
		return runesFnFilter(v, vm, args)

	case "count":
		return runesFnCount(v, vm, args)

	case "all":
		return runesFnAll(v, vm, args)

	case "any":
		return runesFnAny(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func runesTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		o := (*Runes)(v.Ptr)
		rs := o.Elements
		if i < 0 || i >= int64(len(rs)) {
			return Undefined, nil
		}
		return RuneValue(rs[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func runesTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*Runes)(v.Ptr)
	return a.NewRunesIteratorValue(o.Elements)
}

func runesTypeIsTrue(v Value) bool {
	o := (*Runes)(v.Ptr)
	return len(o.Elements) > 0
}

func runesTypeAsString(v Value) (string, bool) {
	o := (*Runes)(v.Ptr)
	return string(o.Elements), true
}

func runesTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Runes)(v.Ptr)
	return o.Elements, true
}

func runesTypeAsInt(v Value) (int64, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(string(o.Elements), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func runesTypeAsFloat(v Value) (float64, bool) {
	o := (*Runes)(v.Ptr)
	f, err := strconv.ParseFloat(string(o.Elements), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func runesTypeAsDecimal(v Value) (Decimal, bool) {
	o := (*Runes)(v.Ptr)
	d := dec128.FromString(string(o.Elements))
	return d, !d.IsNaN()
}

func runesTypeAsBool(v Value) (bool, bool) {
	o := (*Runes)(v.Ptr)
	return conv.ParseBool(string(o.Elements))
}

func runesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Runes)(v.Ptr)
	return []byte(string(o.Elements)), true
}

func runesTypeAsTime(v Value) (time.Time, bool) {
	o := (*Runes)(v.Ptr)
	val, err := dateparse.ParseAny(string(o.Elements))
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func runesTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

func runesTypeContains(v Value, e Value) bool {
	o := (*Runes)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return slices.Contains(o.Elements, c)

	case VT_STRING:
		s := (*String)(e.Ptr)
		return strings.Contains(string(o.Elements), s.Value)

	case VT_RUNES:
		runes := (*Runes)(e.Ptr)
		return strings.Contains(string(o.Elements), string(runes.Elements))

	default:
		c, ok := e.AsRune()
		if !ok {
			return false
		}
		return slices.Contains(o.Elements, c)
	}
}

func runesTypeLen(v Value) int64 {
	o := (*Runes)(v.Ptr)
	return int64(len(o.Elements))
}

func runesTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	l := int64(len(rs))

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}

	if e.Type == VT_UNDEFINED {
		ei = l
	} else {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	if si > ei {
		return Undefined, fmt.Errorf("invalid slice index: %d > %d", si, ei)
	}

	if si < 0 {
		si = 0
	} else if si > l {
		si = l
	}

	if ei < 0 {
		ei = 0
	} else if ei > l {
		ei = l
	}

	return a.NewRunesValue(rs[si:ei])
}

func runesFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		filtered := make([]rune, 0, len(rs))
		for _, v := range rs {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewRunesValue(filtered)

	case 2:
		filtered := make([]rune, 0, len(rs))
		for i, v := range rs {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewRunesValue(filtered)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range rs {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		var count int64
		for i, v := range rs {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range rs {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	case 2:
		for i, v := range rs {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

func runesFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Runes)(v.Ptr)
	rs := o.Elements
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range rs {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	case 2:
		for i, v := range rs {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}
