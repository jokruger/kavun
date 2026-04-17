package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/jokruger/gs/errs"
	"github.com/jokruger/gs/token"
)

type Bytes struct {
	Elements []byte
}

func (o *Bytes) Set(elements []byte) {
	o.Elements = elements
	if o.Elements == nil {
		o.Elements = []byte{}
	}
}

// BytesValue creates new boxed bytes value.
func BytesValue(v *Bytes) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_BYTES,
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

func bytesTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.Elements, r...))
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

func bytesTypeCopy(v Value, a Allocator) (Value, error) {
	o := (*Bytes)(v.Ptr)
	t := make([]byte, len(o.Elements))
	copy(t, o.Elements)
	return a.NewBytesValue(t)
}

func bytesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("to_bytes", "0", len(args))
		}
		return v, nil

	case "to_array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.to_array", "0", len(args))
		}
		a := vm.Allocator()
		t, _ := bytesTypeAsArray(v, a)
		return a.NewArrayValue(t, false)

	case "to_record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.to_record", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		m := make(map[string]Value, len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = IntValue(int64(b))
		}
		return vm.Allocator().NewMapValue(m, false)

	case "to_string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.to_string", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return vm.Allocator().NewStringValue(string(o.Elements))

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.is_empty", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return BoolValue(len(o.Elements) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.len", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return IntValue(int64(len(o.Elements))), nil

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError("bytes.first", v.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[0])), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError("bytes.last", v.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[len(o.Elements)-1])), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError("bytes.contains", "1", len(args))
		}
		return BoolValue(bytesTypeContains(v, args[0])), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func bytesTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("bytes index", "int", index.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if i < 0 || i >= int64(len(o.Elements)) {
			return Undefined, nil
		}
		return IntValue(int64(o.Elements[i])), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

func bytesTypeIsIterable(v Value) bool {
	return true
}

func bytesTypeIterator(v Value, a Allocator) (Value, error) {
	o := (*Bytes)(v.Ptr)
	return a.NewBytesIteratorValue(o.Elements)
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

func bytesTypeAsArray(v Value, a Allocator) ([]Value, bool) {
	o := (*Bytes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
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

func bytesTypeSlice(v Value, a Allocator, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	o := (*Bytes)(v.Ptr)
	l := int64(len(o.Elements))

	if s.Type != VT_UNDEFINED {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("array slice", "int", s.TypeName())
		}
	}

	if e.Type == VT_UNDEFINED {
		ei = l
	} else {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("array slice", "int", e.TypeName())
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

	return a.NewBytesValue(o.Elements[si:ei])
}
