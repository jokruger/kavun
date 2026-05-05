package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
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
		Type:  VT_STRING,
		Const: true,
		Ptr:   unsafe.Pointer(v),
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

func stringTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return stringTypeString(v), nil
	}
	o := (*String)(v.Ptr)
	return formatStringLike(v, sp, o.Value, false)
}

func stringTypeInterface(v Value) any {
	o := (*String)(v.Ptr)
	return o.Value
}

func stringTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	r, ok := rhs.AsString()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	o := (*String)(v.Ptr)
	switch op {
	case token.Add:
		return a.NewStringValue(o.Value + r), nil
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

func stringTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*String)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewBytesValue([]byte(o.Value), false), nil

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := alloc.NewRunes(utf8.RuneCountInString(o.Value), true)
		for i, r := range o.Value {
			rs[i] = r
		}
		return alloc.NewRunesValue(rs, false), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := stringTypeAsBool(v)
		return BoolValue(b), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := stringTypeAsFloat(v)
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := stringTypeAsInt(v)
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := stringTypeAsByte(v)
		return ByteValue(b), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := stringTypeAsDecimal(v)
		r := alloc.NewDecimal()
		*r = d
		return DecimalValue(r), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsTime(v)
		d := alloc.NewTime()
		*d = t
		return TimeValue(d), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(utf8.RuneCountInString(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(utf8.RuneCountInString(o.Value))
		for i, r := range o.Value {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return alloc.NewDictValue(m, false), nil

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

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(strings.ToLower(o.Value)), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(strings.ToUpper(o.Value)), nil

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
			return alloc.NewStringValue(strings.Trim(o.Value, " \t\n")), nil
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
		}
		return alloc.NewStringValue(strings.Trim(o.Value, s)), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := []rune(o.Value)
		slices.Reverse(rs)
		return alloc.NewStringValue(string(rs)), nil

	case "filter":
		return stringFnFilter(v, vm, args)

	case "count":
		return stringFnCount(v, vm, args)

	case "all":
		return stringFnAll(v, vm, args)

	case "any":
		return stringFnAny(v, vm, args)

	case "for_each":
		return stringFnForEach(v, vm, args)

	case "find":
		return stringFnFind(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func stringTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		o := (*String)(v.Ptr)
		i, ok = normalizeSequenceIndex(i, int64(len(o.Value)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(o.Value))
		}
		return ByteValue(o.Value[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func stringTypeIterator(v Value, a *Arena) (Value, error) {
	o := (*String)(v.Ptr)
	return a.NewRunesIteratorValue([]rune(o.Value)), nil
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

func stringTypeAsByte(v Value) (byte, bool) {
	o := (*String)(v.Ptr)
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		if i < 0 || i > 255 {
			return byte(i), false
		}
		return byte(i), true
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

func stringTypeAsDecimal(v Value) (dec128.Dec128, bool) {
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

func stringTypeAsArray(v Value, a *Arena) ([]Value, bool) {
	o := (*String)(v.Ptr)
	arr := a.NewArray(utf8.RuneCountInString(o.Value), true)
	for i, r := range o.Value {
		arr[i] = RuneValue(r)
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

func stringTypeSlice(v Value, a *Arena, s Value, e Value) (Value, error) {
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

	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	si, ei = normalizeSliceBounds(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, l)
	return a.NewStringValue(o.Value[si:ei]), nil
}

func stringTypeSliceStep(v Value, a *Arena, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	o := (*String)(v.Ptr)
	l := int64(len(o.Value))

	step, ok := stepVal.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
	}
	if step == 0 {
		return Undefined, errs.NewSliceStepZeroError()
	}

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}
	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	start, end := normalizeSliceBoundsStep(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, step, l)
	bs := []byte(o.Value)
	result := a.NewBytes(0, false)
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, bs[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, bs[i])
		}
	}
	return a.NewStringValue(string(result)), nil
}

func stringFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*String)(v.Ptr)
	alloc := vm.Allocator()
	filtered := alloc.NewRunes(utf8.RuneCountInString(o.Value), false)

	switch fn.Arity() {
	case 1:
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
		return alloc.NewStringValue(string(filtered)), nil

	case 2:
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
		return alloc.NewStringValue(string(filtered)), nil

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

func stringFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := forEachCallback(args)
	if err != nil {
		return Undefined, err
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
				return Undefined, nil
			}
		}

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}
	}
	return Undefined, nil
}

func stringFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*String)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for i, v := range o.Value {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	case 2:
		for i, v := range o.Value {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
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
