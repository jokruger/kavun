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
	value []byte
}

func (o *Bytes) Set(v []byte) {
	o.value = v

	if o.value == nil {
		o.value = []byte{}
	}
}

func (o *Bytes) Value() []byte {
	return o.value
}

func (o *Bytes) IsEmpty() bool {
	return len(o.value) == 0
}

func (o *Bytes) Len() int {
	return len(o.value)
}

func (o *Bytes) Append(v []byte) {
	o.value = append(o.value, v...)
}

func (o *Bytes) At(i int) byte {
	return o.value[i]
}

func (o *Bytes) Clear() {
	o.value = o.value[:0]
}

func (o *Bytes) Slice(start, end int) []byte {
	return o.value[start:end]
}

func BytesValue(v *Bytes) Value {
	return Value{
		Ptr:  unsafe.Pointer(v),
		Type: VT_BYTES,
	}
}

func NewBytesValue(v []byte) Value {
	t := &Bytes{}
	t.Set(v)
	return BytesValue(t)
}

func bytesTypeName(v Value) string {
	return "bytes"
}

func bytesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Bytes)(v.Ptr)
	b := make([]byte, 0, 2+base64.StdEncoding.EncodedLen(o.Len()))
	b = append(b, '"')
	encodedLen := base64.StdEncoding.EncodedLen(o.Len())
	dst := make([]byte, encodedLen)
	base64.StdEncoding.Encode(dst, o.Value())
	b = append(b, dst...)
	b = append(b, '"')
	return b, nil
}

func bytesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Bytes)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.value); err != nil {
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
	o := &Bytes{value: value}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func bytesTypeString(v Value) string {
	o := (*Bytes)(v.Ptr)
	es := make([]string, len(o.value))
	for i, b := range o.value {
		es[i] = fmt.Sprintf("%d", b)
	}
	return fmt.Sprintf("bytes([%s])", strings.Join(es, ", "))
}

func bytesTypeInterface(v Value) any {
	o := (*Bytes)(v.Ptr)
	return o.value
}

func bytesTypeBinaryOp(v Value, a Allocator, op token.Token, rhs Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes()
	if !ok {
		return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.value, r...)), nil
	}

	return UndefinedValue(), errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

func bytesTypeEqual(v Value, r Value) bool {
	t, ok := r.AsBytes()
	if !ok {
		return false
	}
	o := (*Bytes)(v.Ptr)
	return bytes.Equal(o.value, t)
}

func bytesTypeCopy(v Value, a Allocator) Value {
	o := (*Bytes)(v.Ptr)
	t := make([]byte, len(o.value))
	copy(t, o.value)
	return a.NewBytesValue(t)
}

func bytesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "to_bytes":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("to_bytes", "0", len(args))
		}
		return v, nil

	case "to_array":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bytes.to_array", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		arr := make([]Value, len(o.value))
		for i, b := range o.value {
			arr[i] = IntValue(int64(b))
		}
		return vm.Allocator().NewArrayValue(arr, false), nil

	case "to_record":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bytes.to_record", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		m := make(map[string]Value, len(o.value))
		for i, b := range o.value {
			m[strconv.Itoa(i)] = IntValue(int64(b))
		}
		return vm.Allocator().NewMapValue(m, false), nil

	case "to_string":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bytes.to_string", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return vm.Allocator().NewStringValue(string(o.value)), nil

	case "is_empty":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bytes.is_empty", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return BoolValue(o.IsEmpty()), nil

	case "len":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewWrongNumArgumentsError("bytes.len", "0", len(args))
		}
		o := (*Bytes)(v.Ptr)
		return IntValue(int64(o.Len())), nil

	case "first":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewInvalidMethodError("bytes.first", v.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if len(o.value) == 0 {
			return UndefinedValue(), nil
		}
		return IntValue(int64(o.value[0])), nil

	case "last":
		if len(args) != 0 {
			return UndefinedValue(), errs.NewInvalidMethodError("bytes.last", v.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if len(o.value) == 0 {
			return UndefinedValue(), nil
		}
		return IntValue(int64(o.value[len(o.value)-1])), nil

	default:
		return UndefinedValue(), errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func bytesTypeAccess(v Value, a Allocator, index Value, mode Opcode) (Value, error) {
	if mode == OpIndex {
		i, ok := index.AsInt()
		if !ok {
			return UndefinedValue(), errs.NewInvalidIndexTypeError("bytes index", "int", index.TypeName())
		}
		o := (*Bytes)(v.Ptr)
		if i < 0 || i >= int64(len(o.value)) {
			return UndefinedValue(), nil
		}
		return IntValue(int64(o.value[i])), nil
	}

	k, ok := index.AsString()
	if !ok {
		return UndefinedValue(), errs.NewInvalidIndexTypeError("bytes selector access", "string", index.TypeName())
	}

	return UndefinedValue(), errs.NewInvalidSelectorError(v.TypeName(), k)
}

func bytesTypeIsIterable(v Value) bool {
	return true
}

func bytesTypeIterator(v Value, a Allocator) Value {
	o := (*Bytes)(v.Ptr)
	return a.NewBytesIteratorValue(o.value)
}

func bytesTypeIsTrue(v Value) bool {
	o := (*Bytes)(v.Ptr)
	return len(o.value) > 0
}

func bytesTypeAsString(v Value) (string, bool) {
	o := (*Bytes)(v.Ptr)
	return string(o.value), true
}

func bytesTypeAsBool(v Value) (bool, bool) {
	return bytesTypeIsTrue(v), true
}

func bytesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Bytes)(v.Ptr)
	return o.value, true
}
