package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/token"
)

type Bytes struct {
	Elements []byte
}

func (o *Bytes) Set(elements []byte) {
	o.Elements = elements
}

// BytesValue creates new boxed bytes value.
func BytesValue(v *Bytes) Value {
	return Value{
		Ptr:   unsafe.Pointer(v),
		Const: true,
		Type:  VT_BYTES,
	}
}

// NewBytesValue creates new (heap-allocated) bytes value.
func NewBytesValue(v []byte) Value {
	t := &Bytes{}
	t.Set(v)
	return BytesValue(t)
}

/* Bytes type methods */

func bytesTypeName(v Value) string {
	return "bytes"
}

func bytesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Bytes)(v.Ptr)
	b := make([]byte, 0, 2+base64.StdEncoding.EncodedLen(len(o.Elements)))
	b = append(b, '"')
	encodedLen := base64.StdEncoding.EncodedLen(len(o.Elements))
	dst := make([]byte, encodedLen)
	base64.StdEncoding.Encode(dst, o.Elements)
	b = append(b, dst...)
	b = append(b, '"')
	return b, nil
}

func bytesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Bytes)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("bytes: %w", err)
	}
	return buf.Bytes(), nil
}

func bytesTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var value []byte
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("bytes: %w", err)
	}
	if value == nil {
		value = []byte{}
	}
	o := &Bytes{Elements: value}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func bytesTypeString(v Value) string {
	o := (*Bytes)(v.Ptr)
	es := make([]string, len(o.Elements))
	for i, b := range o.Elements {
		es[i] = fmt.Sprintf("%d", b)
	}
	return fmt.Sprintf("bytes([%s])", strings.Join(es, ", "))
}

func bytesTypeInterface(v Value) any {
	o := (*Bytes)(v.Ptr)
	return o.Elements
}

func bytesTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.Elements, r...)), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func bytesTypeEqual(v Value, r Value) bool {
	t, ok := r.AsBytes()
	if !ok {
		return false
	}
	o := (*Bytes)(v.Ptr)
	return bytes.Equal(o.Elements, t)
}

func bytesTypeCopy(v Value, a *Arena) (Value, error) {
	o := (*Bytes)(v.Ptr)
	t := a.NewBytes(len(o.Elements), true)
	copy(t, o.Elements)
	return a.NewBytesValue(t), nil
}

func bytesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "to_bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false), nil

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewMap(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = IntValue(int64(b))
		}
		return alloc.NewRecordValue(m, false), nil

	case "to_map":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewMap(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = IntValue(int64(b))
		}
		return alloc.NewMapValue(m, false), nil

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(string(o.Elements)), nil

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
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[0])), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[len(o.Elements)-1])), nil

	case "min":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(slices.Min(o.Elements))), nil

	case "max":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(slices.Max(o.Elements))), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(bytesTypeContains(v, args[0])), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := alloc.NewBytes(len(o.Elements), true)
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return alloc.NewBytesValue(sorted), nil

	case "filter":
		return bytesFnFilter(v, vm, args)

	case "count":
		return bytesFnCount(v, vm, args)

	case "all":
		return bytesFnAll(v, vm, args)

	case "any":
		return bytesFnAny(v, vm, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func bytesTypeAccess(v Value, a *Arena, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if i < 0 || i >= int64(len(o.Elements)) {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[i])), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func bytesTypeIterator(v Value, a *Arena) (Value, error) {
	o := (*Bytes)(v.Ptr)
	return a.NewBytesIteratorValue(o.Elements), nil
}

func bytesTypeIsTrue(v Value) bool {
	o := (*Bytes)(v.Ptr)
	return len(o.Elements) > 0
}

func bytesTypeAsString(v Value) (string, bool) {
	o := (*Bytes)(v.Ptr)
	return string(o.Elements), true
}

func bytesTypeAsBool(v Value) (bool, bool) {
	return bytesTypeIsTrue(v), true
}

func bytesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Bytes)(v.Ptr)
	return o.Elements, true
}

func bytesTypeAsArray(v Value, a *Arena) ([]Value, bool) {
	o := (*Bytes)(v.Ptr)
	arr := a.NewArray(len(o.Elements), true)
	for i, b := range o.Elements {
		arr[i] = IntValue(int64(b))
	}
	return arr, true
}

func bytesTypeContains(v Value, e Value) bool {
	o := (*Bytes)(v.Ptr)
	switch e.Type {
	case VT_INT:
		b := int64(e.Data)
		if b < 0 || b > 255 {
			return false
		}
		return bytes.Contains(o.Elements, []byte{byte(b)})

	case VT_BYTES:
		t := (*Bytes)(e.Ptr)
		return bytes.Contains(o.Elements, t.Elements)

	default:
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return false
		}
		return bytes.Contains(o.Elements, []byte{byte(b)})
	}
}

func bytesTypeLen(v Value) int64 {
	o := (*Bytes)(v.Ptr)
	return int64(len(o.Elements))
}

func bytesTypeSlice(v Value, a *Arena, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Bytes)(v.Ptr)
	l := int64(len(o.Elements))

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

	return a.NewBytesValue(o.Elements[si:ei]), nil
}

func bytesFnFilter(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*Bytes)(v.Ptr)
	alloc := vm.Allocator()
	filtered := alloc.NewBytes(len(o.Elements), false)

	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = IntValue(int64(v))
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewBytesValue(filtered), nil

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = IntValue(int64(v))
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewBytesValue(filtered), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func bytesFnCount(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Bytes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range o.Elements {
			buf[0] = IntValue(int64(v))
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = IntValue(int64(v))
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

func bytesFnAll(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Bytes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = IntValue(int64(v))
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = IntValue(int64(v))
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

func bytesFnAny(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Bytes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = IntValue(int64(v))
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = IntValue(int64(v))
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
