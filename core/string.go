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

type String struct {
	Value string
}

func (o *String) Set(s string) {
	o.Value = s
}

// StringValue creates new boxed string value.
func StringValue(v *String) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_STRING,
	}
}

// NewStringValue creates new (heap-allocated) string value.
func NewStringValue(v string) Value {
	o := &String{}
	o.Set(v)
	return StringValue(o)
}

/* String type methods */

func stringTypeName(v Value) string {
	return "string"
}

func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	var b []byte
	b = EncodeString(b, o.Value)
	return b, nil
}

func stringTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*String)(v.Ptr)
	s := o.Value
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("string: %w", err)
	}
	return buf.Bytes(), nil
}

func stringTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("string: %w", err)
	}
	o := &String{Value: s}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func stringTypeString(v Value) string {
	o := (*String)(v.Ptr)
	return strconv.Quote(o.Value)
}

func stringTypeInterface(v Value) any {
	o := (*String)(v.Ptr)
	return o.Value
}

func stringTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsString()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*String)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(o.Value + r)
	case token.Less:
		return BoolValue(o.Value < r), nil
	case token.LessEq:
		return BoolValue(o.Value <= r), nil
	case token.Greater:
		return BoolValue(o.Value > r), nil
	case token.GreaterEq:
		return BoolValue(o.Value >= r), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func stringTypeEqual(v Value, r Value) bool {
	t, ok := r.AsString()
	if !ok {
		return false
	}
	o := (*String)(v.Ptr)
	return o.Value == t
}

func stringTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewStringValue(o.Value)
}

func stringTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*String)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewRunesValue([]rune(o.Value))

	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false)

	case "to_bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := stringTypeAsBool(v)
		return BoolValue(b), nil

	case "to_bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewBytesValue([]byte(o.Value))

	case "to_float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := stringTypeAsFloat(v)
		return FloatValue(f), nil

	case "to_int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := stringTypeAsInt(v)
		return IntValue(i), nil

	case "to_decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := stringTypeAsDecimal(v)
		r, err := alloc.NewDecimal()
		if err != nil {
			return Undefined, err
		}
		*r = d
		return DecimalValue(r), nil

	case "to_time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsTime(v)
		return alloc.NewTimeValue(t)

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = IntValue(int64(r))
		}
		return alloc.NewRecordValue(m, false)

	case "to_map":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = IntValue(int64(r))
		}
		return alloc.NewMapValue(m, false)

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Value) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Value))), nil

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Value[0])), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Value[len(o.Value)-1])), nil

	case "min":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(slices.Min([]byte(o.Value)))), nil

	case "max":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Value) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(slices.Max([]byte(o.Value)))), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(strings.ToLower(o.Value))

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(strings.ToUpper(o.Value))

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(stringTypeContains(v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return alloc.NewStringValue(strings.Trim(o.Value, " \t\n"))
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
		}
		return alloc.NewStringValue(strings.Trim(o.Value, s))

	case "filter":
		return stringFnFilter(v, vm, args)

	case "count":
		return stringFnCount(v, vm, args)

	case "all":
		return stringFnAll(v, vm, args)

	case "any":
		return stringFnAny(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func stringTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		o := (*String)(v.Ptr)
		if i < 0 || i >= int64(len(o.Value)) {
			return Undefined, nil
		}
		return IntValue(int64(o.Value[i])), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func stringTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewRunesIteratorValue([]rune(o.Value))
}

func stringTypeIsTrue(v Value) bool {
	o := (*String)(v.Ptr)
	return len(o.Value) > 0
}

func stringTypeAsString(v Value) (string, bool) {
	o := (*String)(v.Ptr)
	return o.Value, true
}

func stringTypeAsRunes(v Value) ([]rune, bool) {
	o := (*String)(v.Ptr)
	return []rune(o.Value), true
}

func stringTypeAsInt(v Value) (int64, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func stringTypeAsFloat(v Value) (float64, bool) {
	o := (*String)(v.Ptr)
	f, err := strconv.ParseFloat(o.Value, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func stringTypeAsDecimal(v Value) (Decimal, bool) {
	o := (*String)(v.Ptr)
	d := dec128.FromString(o.Value)
	return d, !d.IsNaN()
}

func stringTypeAsBool(v Value) (bool, bool) {
	o := (*String)(v.Ptr)
	return conv.ParseBool(o.Value)
}

func stringTypeAsBytes(v Value) ([]byte, bool) {
	o := (*String)(v.Ptr)
	return []byte(o.Value), true
}

func stringTypeAsTime(v Value) (time.Time, bool) {
	o := (*String)(v.Ptr)
	val, err := dateparse.ParseAny(o.Value)
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

func stringTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	o := (*String)(v.Ptr)
	arr := make([]Value, len(o.Value))
	for i, r := range o.Value {
		arr[i] = IntValue(int64(r))
	}
	return arr, true
}

func stringTypeContains(v Value, e Value) bool {
	o := (*String)(v.Ptr)
	switch e.Type {
	case VT_RUNE:
		c := rune(e.Data)
		return strings.ContainsRune(o.Value, c)

	case VT_STRING:
		s := (*String)(e.Ptr)
		return strings.Contains(o.Value, s.Value)

	default:
		c, ok := e.AsRune()
		if !ok {
			return false
		}
		return strings.ContainsRune(o.Value, c)
	}
}

func stringTypeLen(v Value) int64 {
	o := (*String)(v.Ptr)
	return int64(len(o.Value))
}

func stringTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*String)(v.Ptr)
	l := int64(len(o.Value))

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

	return a.NewStringValue(o.Value[si:ei])
}

func stringFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	o := (*String)(v.Ptr)
	alloc := vm.Allocator()
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		filtered := make([]rune, 0, len(o.Value))
		for _, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewStringValue(string(filtered))

	case 2:
		filtered := make([]rune, 0, len(o.Value))
		for i, v := range o.Value {
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
		return alloc.NewStringValue(string(filtered))

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func stringFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range o.Value {
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
		for i, v := range o.Value {
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

func stringFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Value {
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
		for i, v := range o.Value {
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

func stringFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Value {
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
		for i, v := range o.Value {
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
