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
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/token"
)

const (
	bytesTypeName          = "bytes"
	immutableBytesTypeName = "immutable-bytes"
)

type Bytes = Seq[byte]

// BytesValue creates new boxed bytes value.
func BytesValue(v *Bytes, immutable bool) Value {
	return Value{
		Ptr:       unsafe.Pointer(v),
		Immutable: immutable,
		Type:      VT_BYTES,
	}
}

// NewBytesValue creates new (heap-allocated) bytes value.
func NewBytesValue(v []byte, immutable bool) Value {
	t := &Bytes{}
	t.Set(v)
	return BytesValue(t, immutable)
}

var TypeBytes = ValueType{
	Name:         SeqNameHook(bytesTypeName, immutableBytesTypeName),
	String:       bytesTypeString,
	Format:       bytesTypeFormat,
	Interface:    func(a *Arena, v Value) any { return (*Bytes)(v.Ptr).Elements },
	EncodeJSON:   bytesTypeEncodeJSON,
	EncodeBinary: bytesTypeEncodeBinary,
	DecodeBinary: bytesTypeDecodeBinary,
	IsTrue:       SeqIsTrue[byte],
	IsIterable:   ConstHook(true),
	Iterator:     bytesTypeIterator,
	Equal:        bytesTypeEqual,
	Clone:        bytesTypeClone,
	Len:          SeqLen[byte],
	BinaryOp:     bytesTypeBinaryOp,
	MethodCall:   bytesTypeMethodCall,
	Access:       SeqAccessHook(ByteValue),
	Assign:       SeqAssignHook(Value.AsByte, byteTypeName),
	Append:       bytesTypeAppend,
	Contains:     bytesTypeContains,
	Slice:        SeqSliceHook(ArenaNewBytesValue),
	SliceStep:    SeqSliceStepHook(ArenaNewBytes, ArenaNewBytesValue),
	AsBool:       func(a *Arena, v Value) (bool, bool) { return conv.ParseBool(string((*Bytes)(v.Ptr).Elements)) },
	AsString:     func(a *Arena, v Value) (string, bool) { return string((*Bytes)(v.Ptr).Elements), true },
	AsBytes:      func(a *Arena, v Value) ([]byte, bool) { return (*Bytes)(v.Ptr).Elements, true },
	AsArray:      bytesTypeAsArray,
}

func bytesTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
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

func bytesTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := (*Bytes)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("bytes: %w", err)
	}
	return buf.Bytes(), nil
}

func bytesTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
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

func bytesTypeString(a *Arena, v Value) string {
	o := (*Bytes)(v.Ptr)
	es := make([]string, len(o.Elements))
	for i, b := range o.Elements {
		es[i] = fmt.Sprintf("%d", b)
	}
	return fmt.Sprintf("bytes([%s])", strings.Join(es, ", "))
}

func bytesTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return bytesTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(a), sp, fspec.AlignLeft), nil
	}
	o := (*Bytes)(v.Ptr)
	return format.FormatStringLike(bytesTypeName, sp, string(o.Elements), true)
}

func bytesTypeAppend(a *Arena, v Value, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	res := append([]byte{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case VT_BYTES:
			t := (*Bytes)(arg.Ptr)
			res = append(res, t.Elements...)
		default:
			b, ok := arg.AsByte(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "byte or bytes", arg.TypeName(a))
			}
			res = append(res, b)
		}
	}
	return a.NewBytesValue(res, false), nil
}

func bytesTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.Elements, r...), false), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func bytesTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsBytes(a)
	if !ok {
		return false
	}
	o := (*Bytes)(v.Ptr)
	return bytes.Equal(o.Elements, t)
}

func bytesTypeClone(a *Arena, v Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	t := a.NewBytes(len(o.Elements), true)
	copy(t, o.Elements)
	return a.NewBytesValue(t, false), nil
}

func bytesTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return bytesTypeClone(a, v)

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(a, v)
		return a.NewArrayValue(t, false), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return a.NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return a.NewDictValue(m, false), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(string(o.Elements)), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := bytesTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

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
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return ByteValue(o.Elements[0]), nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
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
		return BoolValue(bytesTypeContains(a, v, args[0])), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := a.NewBytes(len(o.Elements), true)
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return a.NewBytesValue(sorted, false), nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := a.NewBytes(len(o.Elements), false)
		for i, b := range o.Elements {
			if i == 0 || b != o.Elements[i-1] {
				out = append(out, b)
			}
		}
		return a.NewBytesValue(out, false), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := a.NewBytes(len(o.Elements), false)
		var seen [256]bool
		for _, b := range o.Elements {
			if !seen[b] {
				seen[b] = true
				out = append(out, b)
			}
		}
		return a.NewBytesValue(out, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := a.NewBytes(n, true)
		for i, b := range o.Elements {
			rev[n-1-i] = b
		}
		return a.NewBytesValue(rev, false), nil

	case "filter":
		return SeqFilter(a, vm, v, args, ByteValue, ArenaNewBytes, ArenaNewBytesValue)

	case "count":
		return SeqCount(a, vm, v, args, ByteValue)

	case "all":
		return SeqAll(a, vm, v, args, ByteValue)

	case "any":
		return SeqAny(a, vm, v, args, ByteValue)

	case "for_each":
		return SeqForEach(a, vm, v, args, ByteValue)

	case "find":
		return SeqFind(a, vm, v, args, ByteValue)

	case "chunk":
		return SeqChunk(a, v, args, ArenaNewBytes, ArenaNewBytesValue)

	case "sum":
		return bytesFnSum(v, args)

	case "avg":
		return bytesFnAvg(v, args)

	case "map":
		return SeqMap(a, vm, v, args, ByteValue)

	case "reduce":
		return SeqReduce(a, vm, v, args, ByteValue)

	case "repeat":
		n, err := parseRepeatCount(a, name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := a.NewBytes(n*sl, true)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return a.NewBytesValue(out, false), nil

	case "split":
		return bytesFnSplit(a, v, args)

	case "split_lines":
		return bytesFnSplitLines(a, v, args)

	case "partition":
		return bytesFnPartition(a, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func bytesTypeIterator(a *Arena, v Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	return a.NewBytesIteratorValue(o.Elements), nil
}

func bytesTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := (*Bytes)(v.Ptr)
	arr := a.NewArray(len(o.Elements), true)
	for i, b := range o.Elements {
		arr[i] = ByteValue(b)
	}
	return arr, true
}

func bytesTypeContains(a *Arena, v Value, e Value) bool {
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
		b, ok := e.AsByte(a)
		if !ok {
			return false
		}
		return bytes.Contains(o.Elements, []byte{b})
	}
}

func bytesFnSum(v Value, args []Value) (Value, error) {
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

func bytesFnAvg(v Value, args []Value) (Value, error) {
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

func bytesFnSplit(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*Bytes)(v.Ptr)
	var pieces [][]byte
	if len(args) == 0 {
		pieces = splitBytesWhitespace(o.Elements)
	} else {
		sep, err := coerceSepToBytes(a, name, args[0])
		if err != nil {
			return Undefined, err
		}
		if len(sep) == 0 {
			return Undefined, fmt.Errorf("split separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(a, name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitBytesByLiteral(o.Elements, sep, limit)
	}
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		buf := a.NewBytes(len(p), true)
		copy(buf, p)
		arr[i] = a.NewBytesValue(buf, false)
	}
	return a.NewArrayValue(arr, false), nil
}

func bytesFnSplitLines(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Bytes)(v.Ptr)
	pieces := splitLinesBytes(o.Elements)
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		buf := a.NewBytes(len(p), true)
		copy(buf, p)
		arr[i] = a.NewBytesValue(buf, false)
	}
	return a.NewArrayValue(arr, false), nil
}

func bytesFnPartition(a *Arena, v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToBytes(a, name, args[0])
	if err != nil {
		return Undefined, err
	}
	if len(sep) == 0 {
		return Undefined, fmt.Errorf("partition separator must not be empty")
	}
	o := (*Bytes)(v.Ptr)
	arr := a.NewArray(3, true)
	idx := bytes.Index(o.Elements, sep)
	makeCopy := func(src []byte) Value {
		buf := a.NewBytes(len(src), true)
		copy(buf, src)
		return a.NewBytesValue(buf, false)
	}
	if idx < 0 {
		arr[0] = makeCopy(o.Elements)
		arr[1] = a.NewBytesValue(nil, false)
		arr[2] = a.NewBytesValue(nil, false)
	} else {
		arr[0] = makeCopy(o.Elements[:idx])
		arr[1] = makeCopy(o.Elements[idx : idx+len(sep)])
		arr[2] = makeCopy(o.Elements[idx+len(sep):])
	}
	return a.NewArrayValue(arr, false), nil
}
