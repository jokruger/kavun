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

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const (
	bytesTypeName          = "bytes"
	immutableBytesTypeName = "immutable-bytes"
)

type Bytes = Seq[byte]

func NewStaticBytesValue(b *Bytes) Value {
	return Value{Type: value.Bytes, Immutable: true, Ptr: unsafe.Pointer(b)}
}

func NewBytesValue(b []byte, immutable bool) Value {
	o := &Bytes{}
	o.Set(b)
	return Value{Type: value.Bytes, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeBytes = ValueTypeDescr{
	Name:         SeqNameHook(bytesTypeName, immutableBytesTypeName),                                     // PURE by contract
	String:       bytesTypeString,                                                                        // PURE by contract
	Format:       bytesTypeFormat,                                                                        // PURE by contract
	Interface:    func(v Value) any { return (*Bytes)(v.Ptr).Elements },                                  // PURE by contract
	EncodeJSON:   bytesTypeEncodeJSON,                                                                    // PURE by contract
	EncodeBinary: bytesTypeEncodeBinary,                                                                  // PURE by contract
	DecodeBinary: bytesTypeDecodeBinary,                                                                  // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) bool { return len((*Bytes)(v.Ptr).Elements) > 0 },                        // PURE by contract
	IsIterable:   ConstHook(true),                                                                        // PURE by contract
	Iterator:     bytesTypeIterator,                                                                      // PURE by contract (constructs fresh iterator)
	Equal:        bytesTypeEqual,                                                                         // PURE by contract
	Clone:        bytesTypeClone,                                                                         // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Bytes)(v.Ptr).Elements)) },                    // PURE by contract
	BinaryOp:     bytesTypeBinaryOp,                                                                      // PURE by contract
	MethodCall:   bytesTypeMethodCall,                                                                    // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       SeqAccessHook(ByteValue, bytesTypeResolve),                                             // PURE by contract
	Assign:       SeqAssignHook(bytesTypeResolve, Value.AsByte, byteTypeName),                            // IMPURE by contract
	Append:       bytesTypeAppend,                                                                        // GO-STYLE by contract (may share receiver storage)
	Contains:     bytesTypeContains,                                                                      // PURE by contract
	Slice:        SeqSliceHook(NewBytesValue, bytesTypeResolve),                                          // PURE by contract
	SliceStep:    SeqSliceStepHook(NewBytesValue, bytesTypeResolve),                                      // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return conv.ParseBool(string((*Bytes)(v.Ptr).Elements)) }, // PURE by contract
	AsString:     func(v Value) (string, bool) { return string((*Bytes)(v.Ptr).Elements), true },         // PURE by contract
	AsBytes:      func(v Value) ([]byte, bool) { return (*Bytes)(v.Ptr).Elements, true },                 // PURE by contract
	AsArray:      bytesTypeAsArray,                                                                       // PURE by contract

	// No _in_place methods. Higher-order methods (filter/count/all/any/for_each/find/map/reduce) are gated the same
	// way as string's. All methods are expected to be pure.
	IsMethodPure: func(string) bool { return true },
}

func bytesTypeResolve(v Value) *Bytes {
	return (*Bytes)(v.Ptr)
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
	*v = NewBytesValue(value, v.Immutable)
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
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	o := (*Bytes)(v.Ptr)
	return format.FormatStringLike(bytesTypeName, sp, string(o.Elements), true)
}

// GO-STYLE: may reuse the receiver's backing storage (mirrors Go's append). Not required to be pure; callers are
// expected to overwrite the receiver via `x = append(x, ...)`. Not folded by the optimizer. See docs/purity.md.
func bytesTypeAppend(v Value, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	res := append([]byte{}, o.Elements...)
	for i, arg := range args {
		switch arg.Type {
		case value.Bytes:
			res = append(res, (*Bytes)(arg.Ptr).Elements...)
		default:
			b, ok := arg.AsByte()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError("append", fmt.Sprintf("%d", i+1), "byte or bytes", arg.TypeName())
			}
			res = append(res, b)
		}
	}
	return NewBytesValue(res, false), nil
}

// PURE by contract
func bytesTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	o := (*Bytes)(v.Ptr)
	r, ok := rhs.AsBytes()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	switch op {
	case token.Add:
		t := make([]byte, len(o.Elements)+len(r))
		copy(t, o.Elements)
		copy(t[len(o.Elements):], r)
		return NewBytesValue(t, false), nil
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

func bytesTypeClone(v Value) (Value, error) {
	o := (*Bytes)(v.Ptr)
	t := make([]byte, len(o.Elements))
	copy(t, o.Elements)
	return NewBytesValue(t, false), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func bytesTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return bytesTypeClone(v)

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, len(o.Elements))
		for i, b := range o.Elements {
			m[strconv.Itoa(i)] = ByteValue(b)
		}
		return NewDictValue(m, false), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(string(o.Elements)), nil

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
		return NewStringValue(s), nil

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
		sorted := make([]byte, len(o.Elements))
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return NewBytesValue(sorted, false), nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]byte, 0, len(o.Elements))
		for i, b := range o.Elements {
			if i == 0 || b != o.Elements[i-1] {
				out = append(out, b)
			}
		}
		return NewBytesValue(out, false), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]byte, 0, len(o.Elements))
		var seen [256]bool
		for _, b := range o.Elements {
			if !seen[b] {
				seen[b] = true
				out = append(out, b)
			}
		}
		return NewBytesValue(out, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := make([]byte, n)
		for i, b := range o.Elements {
			rev[n-1-i] = b
		}
		return NewBytesValue(rev, false), nil

	case "filter":
		return SeqFilter(vm, v, args, ByteValue, NewBytesValue, bytesTypeResolve)

	case "count":
		return SeqCount(vm, v, args, ByteValue, bytesTypeResolve)

	case "all":
		return SeqAll(vm, v, args, ByteValue, bytesTypeResolve)

	case "any":
		return SeqAny(vm, v, args, ByteValue, bytesTypeResolve)

	case "for_each":
		return SeqForEach(vm, v, args, ByteValue, bytesTypeResolve)

	case "find":
		return SeqFind(vm, v, args, ByteValue, bytesTypeResolve)

	case "chunk":
		return SeqChunk(v, args, NewBytesValue, bytesTypeResolve)

	case "sum":
		return bytesFnSum(v, args)

	case "avg":
		return bytesFnAvg(v, args)

	case "map":
		return SeqMap(vm, v, args, ByteValue, bytesTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, ByteValue, bytesTypeResolve)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := make([]byte, n*sl)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return NewBytesValue(out, false), nil

	case "split":
		return bytesFnSplit(v, args)

	case "split_lines":
		return bytesFnSplitLines(v, args)

	case "partition":
		return bytesFnPartition(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func bytesTypeIterator(v Value) (Value, error) {
	return NewBytesIteratorValue((*Bytes)(v.Ptr).Elements), nil
}

func bytesTypeAsArray(v Value) ([]Value, bool) {
	o := (*Bytes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
	for i, b := range o.Elements {
		arr[i] = ByteValue(b)
	}
	return arr, true
}

func bytesTypeContains(v Value, e Value) bool {
	o := (*Bytes)(v.Ptr)
	switch e.Type {
	case value.Byte:
		b := byte(e.Data)
		return bytes.Contains(o.Elements, []byte{b})

	case value.Int:
		b := int64(e.Data)
		if b < 0 || b > 255 {
			return false
		}
		return bytes.Contains(o.Elements, []byte{byte(b)})

	case value.Bytes:
		return bytes.Contains(o.Elements, (*Bytes)(e.Ptr).Elements)

	default:
		b, ok := e.AsByte()
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

func bytesFnSplit(v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*Bytes)(v.Ptr)
	var pieces [][]byte
	if len(args) == 0 {
		pieces = splitBytesWhitespace(o.Elements)
	} else {
		sep, err := coerceSepToBytes(name, args[0])
		if err != nil {
			return Undefined, err
		}
		if len(sep) == 0 {
			return Undefined, fmt.Errorf("split separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitBytesByLiteral(o.Elements, sep, limit)
	}
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		buf := make([]byte, len(p))
		copy(buf, p)
		arr[i] = NewBytesValue(buf, false)
	}
	return NewArrayValue(arr, false), nil
}

func bytesFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Bytes)(v.Ptr)
	pieces := splitLinesBytes(o.Elements)
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		buf := make([]byte, len(p))
		copy(buf, p)
		arr[i] = NewBytesValue(buf, false)
	}
	return NewArrayValue(arr, false), nil
}

func bytesFnPartition(v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToBytes(name, args[0])
	if err != nil {
		return Undefined, err
	}
	if len(sep) == 0 {
		return Undefined, fmt.Errorf("partition separator must not be empty")
	}
	o := (*Bytes)(v.Ptr)
	arr := make([]Value, 3)
	idx := bytes.Index(o.Elements, sep)
	makeCopy := func(src []byte) Value {
		buf := make([]byte, len(src))
		copy(buf, src)
		return NewBytesValue(buf, false)
	}
	if idx < 0 {
		arr[0] = makeCopy(o.Elements)
		arr[1] = NewBytesValue(nil, false)
		arr[2] = NewBytesValue(nil, false)
	} else {
		arr[0] = makeCopy(o.Elements[:idx])
		arr[1] = makeCopy(o.Elements[idx : idx+len(sep)])
		arr[2] = makeCopy(o.Elements[idx+len(sep):])
	}
	return NewArrayValue(arr, false), nil
}
