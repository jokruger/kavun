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
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/token"
)

type Bytes struct {
	Elements []byte
}

func (o *Bytes) Set(elements []byte) {
	o.Elements = elements
}

// BytesValue creates new boxed bytes value.
func BytesValue(v *Bytes, immutable bool) Value {
	return Value{
		Ptr:   unsafe.Pointer(v),
		Const: immutable,
		Type:  VT_BYTES,
	}
}

// NewBytesValue creates new (heap-allocated) bytes value.
func NewBytesValue(v []byte, immutable bool) Value {
	t := &Bytes{}
	t.Set(v)
	return BytesValue(t, immutable)
}

/* Bytes type methods */

func bytesTypeName(v Value) string {
	if v.Const {
		return "immutable-bytes"
	}
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

func bytesTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return bytesTypeString(v), nil
	}
	o := (*Bytes)(v.Ptr)
	return format.FormatStringLike("bytes", sp, string(o.Elements), true)
}

func bytesTypeInterface(v Value) any {
	o := (*Bytes)(v.Ptr)
	return o.Elements
}

func bytesTypeAssign(v Value, index Value, r Value) error {
	if v.Const {
		return errs.NewNotAssignableError("immutable-bytes")
	}

	o := (*Bytes)(v.Ptr)
	i, ok := index.AsInt()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign", "int", index.TypeName())
	}
	i, ok = normalizeSequenceIndex(i, int64(len(o.Elements)))
	if !ok {
		return errs.NewIndexOutOfBoundsError("index assign", int(i), len(o.Elements))
	}

	b, ok := r.AsByte()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign value", "byte", r.TypeName())
	}
	o.Elements[i] = b

	return nil
}

func bytesTypeAppend(v Value, a *Arena, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	res := append([]byte{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case VT_BYTES:
			t := (*Bytes)(arg.Ptr)
			res = append(res, t.Elements...)
		default:
			b, ok := arg.AsByte()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "byte or bytes", arg.TypeName())
			}
			res = append(res, b)
		}
	}
	return a.NewBytesValue(res, false), nil
}

func bytesTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.Elements, r...), false), nil
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
	return a.NewBytesValue(t, false), nil
}

func bytesTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	alloc := vm.Allocator()

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return bytesTypeCopy(v, alloc)

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(v, alloc)
		return alloc.NewArrayValue(t, false), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return alloc.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := alloc.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return alloc.NewDictValue(m, false), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return alloc.NewStringValue(string(o.Elements)), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := bytesTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return vm.Allocator().NewStringValue(s), nil

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
		return ByteValue(o.Elements[0]), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return ByteValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return ByteValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return ByteValue(slices.Max(o.Elements)), nil

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
		return alloc.NewBytesValue(sorted, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := alloc.NewBytes(n, true)
		for i, b := range o.Elements {
			rev[n-1-i] = b
		}
		return alloc.NewBytesValue(rev, false), nil

	case "filter":
		return bytesFnFilter(v, vm, args)

	case "count":
		return bytesFnCount(v, vm, args)

	case "all":
		return bytesFnAll(v, vm, args)

	case "any":
		return bytesFnAny(v, vm, args)

	case "for_each":
		return bytesFnForEach(v, vm, args)

	case "find":
		return bytesFnFind(v, vm, args)

	case "chunk":
		return bytesFnChunk(v, vm, args)

	case "sum":
		return bytesFnSum(v, vm, args)

	case "avg":
		return bytesFnAvg(v, vm, args)

	case "map":
		return bytesFnMap(v, vm, args)

	case "reduce":
		return bytesFnReduce(v, vm, args)

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
		i, ok = normalizeSequenceIndex(i, int64(len(o.Elements)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(o.Elements))
		}
		return ByteValue(o.Elements[i]), nil
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
		arr[i] = ByteValue(b)
	}
	return arr, true
}

func bytesTypeContains(v Value, e Value) bool {
	o := (*Bytes)(v.Ptr)
	switch e.Type {
	case VT_BYTE:
		b := byte(e.Data)
		return bytes.Contains(o.Elements, []byte{b})

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
		b, ok := e.AsByte()
		if !ok {
			return false
		}
		return bytes.Contains(o.Elements, []byte{b})
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

	if e.Type != VT_UNDEFINED {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	si, ei = normalizeSliceBounds(si, s.Type != VT_UNDEFINED, ei, e.Type != VT_UNDEFINED, l)
	return a.NewBytesValue(o.Elements[si:ei], v.Const), nil
}

func bytesTypeSliceStep(v Value, a *Arena, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	o := (*Bytes)(v.Ptr)
	l := int64(len(o.Elements))

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
	result := a.NewBytes(0, false)
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, o.Elements[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, o.Elements[i])
		}
	}

	return a.NewBytesValue(result, false), nil
}

func bytesFnChunk(v Value, vm VM, args []Value) (Value, error) {
	size, copyChunks, err := chunkArgs("chunk", args)
	if err != nil {
		return Undefined, err
	}

	o := (*Bytes)(v.Ptr)
	length := len(o.Elements)
	alloc := vm.Allocator()
	chunks := alloc.NewArray(chunkCount(length, size), true)

	if length == 0 {
		return alloc.NewArrayValue(chunks, false), nil
	}

	chunkSize := length
	if size < int64(length) {
		chunkSize = int(size)
	}

	for i, start := 0, 0; start < length; i, start = i+1, start+chunkSize {
		end := start + chunkSize
		if end > length {
			end = length
		}
		chunk := o.Elements[start:end]
		chunkConst := v.Const
		if copyChunks {
			chunk = alloc.NewBytes(end-start, true)
			copy(chunk, o.Elements[start:end])
			chunkConst = false
		}
		chunks[i] = alloc.NewBytesValue(chunk, chunkConst)
	}

	return alloc.NewArrayValue(chunks, false), nil
}

func bytesFnForEach(v Value, vm VM, args []Value) (Value, error) {
	fn, err := forEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Bytes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range o.Elements {
			buf[0] = ByteValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = ByteValue(v)
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

func bytesFnFind(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*Bytes)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for i, v := range o.Elements {
			buf[0] = ByteValue(v)
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
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = ByteValue(v)
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
			buf[0] = ByteValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewBytesValue(filtered, false), nil

	case 2:
		for i, v := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = ByteValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return alloc.NewBytesValue(filtered, false), nil

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
			buf[0] = ByteValue(v)
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
			buf[1] = ByteValue(v)
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
			buf[0] = ByteValue(v)
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
			buf[1] = ByteValue(v)
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
			buf[0] = ByteValue(v)
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
			buf[1] = ByteValue(v)
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

func bytesFnSum(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}
	o := (*Bytes)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	var s int64
	for _, b := range o.Elements {
		s += int64(b)
	}
	return IntValue(s), nil
}

func bytesFnAvg(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}
	o := (*Bytes)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	var s int64
	for _, b := range o.Elements {
		s += int64(b)
	}
	return IntValue(s / int64(len(o.Elements))), nil
}

func bytesFnMap(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("map", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*Bytes)(v.Ptr)
	alloc := vm.Allocator()
	mapped := alloc.NewArray(len(o.Elements), true)

	switch fn.Arity() {
	case 1:
		for i, b := range o.Elements {
			buf[0] = ByteValue(b)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return alloc.NewArrayValue(mapped, false), nil

	case 2:
		for i, b := range o.Elements {
			buf[0] = IntValue(int64(i))
			buf[1] = ByteValue(b)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			mapped[i] = res
		}
		return alloc.NewArrayValue(mapped, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("map", "first", "f/1 or f/2", fn.TypeName())
	}
}

func bytesFnReduce(v Value, vm VM, args []Value) (Value, error) {
	if len(args) != 2 {
		return Undefined, errs.NewWrongNumArgumentsError("reduce", "2", len(args))
	}

	acc := args[0]
	fn := args[1]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "non-variadic function", fn.TypeName())
	}

	o := (*Bytes)(v.Ptr)
	var buf [3]Value
	switch fn.Arity() {
	case 2:
		for _, b := range o.Elements {
			buf[0] = acc
			buf[1] = ByteValue(b)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case 3:
		for i, b := range o.Elements {
			buf[0] = acc
			buf[1] = IntValue(int64(i))
			buf[2] = ByteValue(b)
			res, err := fn.Call(vm, buf[:3])
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("reduce", "second", "f/2 or f/3", fn.TypeName())
	}
}
