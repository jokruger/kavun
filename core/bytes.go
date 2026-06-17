package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"

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

var TypeBytes = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinBytesValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainBytesValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseBytesValue(v) },
	Name:         SeqNameHook(bytesTypeName, immutableBytesTypeName),
	String:       bytesTypeString,
	Format:       bytesTypeFormat,
	Interface:    func(a *Arena, v Value) any { return a.ResolveBytesValue(v).Elements },
	EncodeJSON:   bytesTypeEncodeJSON,
	EncodeBinary: bytesTypeEncodeBinary,
	DecodeBinary: bytesTypeDecodeBinary,
	IsTrue:       func(a *Arena, v Value) bool { return len(a.ResolveBytesValue(v).Elements) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     bytesTypeIterator,
	Equal:        bytesTypeEqual,
	Clone:        bytesTypeClone,
	Len:          func(a *Arena, v Value) int64 { return int64(len(a.ResolveBytesValue(v).Elements)) },
	BinaryOp:     bytesTypeBinaryOp,
	MethodCall:   bytesTypeMethodCall,
	Access:       SeqAccessHook(ByteValue, bytesTypeResolve),
	Assign:       SeqAssignHook(bytesTypeResolve, Value.AsByte, func(byte, *Arena) {}, byteTypeName),
	Append:       bytesTypeAppend,
	Contains:     bytesTypeContains,
	Slice:        SeqSliceHook(ArenaNewBytesValue, bytesTypeResolve),
	SliceStep:    SeqSliceStepHook(ArenaNewBytes, ArenaNewBytesValue, bytesTypeResolve),
	AsBool:       func(a *Arena, v Value) (bool, bool) { return conv.ParseBool(string(a.ResolveBytesValue(v).Elements)) },
	AsString:     func(a *Arena, v Value) (string, bool) { return string(a.ResolveBytesValue(v).Elements), true },
	AsBytes:      func(a *Arena, v Value) ([]byte, bool) { return a.ResolveBytesValue(v).Elements, true },
	AsArray:      bytesTypeAsArray,
}

func bytesTypeResolve(a *Arena, v Value) *Bytes {
	return a.ResolveBytesValue(v)
}

func bytesTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveBytesValue(v)
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
	o := a.ResolveBytesValue(v)
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
	o, err := a.NewBytesValue(value, v.Immutable)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o
	return nil
}

func bytesTypeString(a *Arena, v Value) string {
	o := a.ResolveBytesValue(v)
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
	o := a.ResolveBytesValue(v)
	return format.FormatStringLike(bytesTypeName, sp, string(o.Elements), true)
}

func bytesTypeAppend(a *Arena, v Value, args []Value) (Value, error) {
	o := a.ResolveBytesValue(v)
	res := append([]byte{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case VT_BYTES:
			t := a.ResolveBytesValue(arg)
			res = append(res, t.Elements...)
		default:
			b, ok := arg.AsByte(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "byte or bytes", arg.TypeName(a))
			}
			res = append(res, b)
		}
	}
	return a.NewBytesValue(res, false)
}

func bytesTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
	o := a.ResolveBytesValue(v)
	r, ok := rhs.AsBytes(a)
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
	}

	switch op {
	case token.Add:
		return a.NewBytesValue(append(o.Elements, r...), false)
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
}

func bytesTypeEqual(a *Arena, v Value, r Value) bool {
	t, ok := r.AsBytes(a)
	if !ok {
		return false
	}
	o := a.ResolveBytesValue(v)
	return bytes.Equal(o.Elements, t)
}

func bytesTypeClone(a *Arena, v Value) (Value, error) {
	o := a.ResolveBytesValue(v)
	t := a.NewBytes(len(o.Elements), true)
	copy(t, o.Elements)
	return a.NewBytesValue(t, false)
}

func bytesTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := a.ResolveBytesValue(v)

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
		v.Retain(a)
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(a, v)
		return a.NewArrayValue(t, false)

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return a.NewRecordValue(m, false)

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := a.NewDict(len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return a.NewDictValue(m, false)

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(string(o.Elements))

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
		return a.NewStringValue(s)

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
		return a.NewBytesValue(sorted, false)

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
		return a.NewBytesValue(out, false)

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
		return a.NewBytesValue(out, false)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := a.NewBytes(n, true)
		for i, b := range o.Elements {
			rev[n-1-i] = b
		}
		return a.NewBytesValue(rev, false)

	case "filter":
		return SeqFilter(a, vm, v, args, ByteValue, ArenaNewBytes, ArenaNewBytesValue, bytesTypeResolve)

	case "count":
		return SeqCount(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "all":
		return SeqAll(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "any":
		return SeqAny(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "for_each":
		return SeqForEach(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "find":
		return SeqFind(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "chunk":
		return SeqChunk(a, v, args, ArenaNewBytes, ArenaNewBytesValue, bytesTypeResolve)

	case "sum":
		return bytesFnSum(a, v, args)

	case "avg":
		return bytesFnAvg(a, v, args)

	case "map":
		return SeqMap(a, vm, v, args, ByteValue, bytesTypeResolve)

	case "reduce":
		return SeqReduce(a, vm, v, args, ByteValue, bytesTypeResolve)

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
		return a.NewBytesValue(out, false)

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
	o := a.ResolveBytesValue(v)
	return a.NewBytesIteratorValue(o.Elements)
}

func bytesTypeAsArray(a *Arena, v Value) ([]Value, bool) {
	o := a.ResolveBytesValue(v)
	arr := a.NewArray(len(o.Elements), true)
	for i, b := range o.Elements {
		arr[i] = ByteValue(b)
	}
	return arr, true
}

func bytesTypeContains(a *Arena, v Value, e Value) bool {
	o := a.ResolveBytesValue(v)
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
		t := a.ResolveBytesValue(e)
		return bytes.Contains(o.Elements, t.Elements)

	default:
		b, ok := e.AsByte(a)
		if !ok {
			return false
		}
		return bytes.Contains(o.Elements, []byte{b})
	}
}

func bytesFnSum(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}
	o := a.ResolveBytesValue(v)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}
	var s int64
	for _, b := range o.Elements {
		s += int64(b)
	}
	return IntValue(s), nil
}

func bytesFnAvg(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}
	o := a.ResolveBytesValue(v)
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
	o := a.ResolveBytesValue(v)
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
		nv, err := a.NewBytesValue(buf, false)
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[i] = nv
	}
	return a.NewArrayValue(arr, false)
}

func bytesFnSplitLines(a *Arena, v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := a.ResolveBytesValue(v)
	pieces := splitLinesBytes(o.Elements)
	arr := a.NewArray(len(pieces), true)
	for i, p := range pieces {
		buf := a.NewBytes(len(p), true)
		copy(buf, p)
		nv, err := a.NewBytesValue(buf, false)
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[i] = nv
	}
	return a.NewArrayValue(arr, false)
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
	o := a.ResolveBytesValue(v)
	arr := a.NewArray(3, true)
	idx := bytes.Index(o.Elements, sep)
	makeCopy := func(src []byte) (Value, error) {
		buf := a.NewBytes(len(src), true)
		copy(buf, src)
		return a.NewBytesValue(buf, false)
	}
	if idx < 0 {
		nv, err := makeCopy(o.Elements)
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[0] = nv
		nv, err = a.NewBytesValue(nil, false)
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[1] = nv
		nv, err = a.NewBytesValue(nil, false)
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[2] = nv
	} else {
		nv, err := makeCopy(o.Elements[:idx])
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[0] = nv
		nv, err = makeCopy(o.Elements[idx : idx+len(sep)])
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[1] = nv
		nv, err = makeCopy(o.Elements[idx+len(sep):])
		if err != nil {
			return Undefined, err
		}
		a.PinBytesValue(nv)
		arr[2] = nv
	}
	return a.NewArrayValue(arr, false)
}
