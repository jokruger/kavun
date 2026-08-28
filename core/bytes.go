package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
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
	IsTrue:       func(v Value) (bool, error) { return len((*Bytes)(v.Ptr).Elements) > 0, nil },          // PURE by contract
	IsIterable:   ConstHook(true),                                                                        // PURE by contract
	Iterator:     bytesTypeIterator,                                                                      // PURE by contract (constructs fresh iterator)
	Equal:        bytesTypeEqual,                                                                         // PURE by contract
	BinaryOp:     bytesTypeBinaryOp,                                                                      // PURE by contract
	Copy:         bytesTypeCopy,                                                                          // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Bytes)(v.Ptr).Elements)) },                    // PURE by contract
	MethodCall:   bytesTypeMethodCall,                                                                    // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       SeqAccessHook(ByteValue, bytesTypeResolve),                                             // PURE by contract
	Assign:       SeqAssignHook(bytesTypeResolve, Value.AsByte, byteTypeName),                            // IMPURE by contract
	Append:       bytesTypeAppend,                                                                        // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Contains:     bytesTypeContains,                                                                      // PURE by contract
	Slice:        SeqSliceHook(NewBytesValue, bytesTypeResolve),                                          // PURE by contract
	SliceStep:    SeqSliceStepHook(NewBytesValue, bytesTypeResolve),                                      // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return conv.ParseBool(string((*Bytes)(v.Ptr).Elements)) }, // PURE by contract
	AsString:     func(v Value) (string, bool) { return string((*Bytes)(v.Ptr).Elements), true },         // PURE by contract
	AsBytes:      func(v Value) ([]byte, bool) { return (*Bytes)(v.Ptr).Elements, true },                 // PURE by contract
	AsArray:      bytesTypeAsArray,                                                                       // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (filter/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
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

// bytesAppendItems flattens append's variadic args (byte or bytes values) into a single []byte, using methodName
// in any argument-type error so it reads correctly whether called from append() or append_in_place().
func bytesAppendItems(args []Value, methodName string) ([]byte, error) {
	items := make([]byte, 0, len(args))
	for i, arg := range args {
		switch arg.Type {
		case value.Bytes:
			items = append(items, (*Bytes)(arg.Ptr).Elements...)
		default:
			b, ok := arg.AsByte()
			if !ok {
				return nil, errs.NewInvalidArgumentTypeError(methodName, fmt.Sprintf("%d", i+1), "byte or bytes", arg.TypeName())
			}
			items = append(items, b)
		}
	}
	return items, nil
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// bytes value with the items appended (append()) — never touches the receiver's backing storage, works
// regardless of the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func bytesTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Bytes)(v.Ptr)
	name := "append"
	if mutate {
		name = "append_in_place"
	}
	items, err := bytesAppendItems(args, name)
	if err != nil {
		return Undefined, err
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotAppendableError(v.TypeName())
		}
		o.Set(append(o.Elements, items...))
		return v, nil
	}

	// Pure: build a fresh, independent slice — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule).
	res := make([]byte, 0, len(o.Elements)+len(items))
	res = append(res, o.Elements...)
	res = append(res, items...)
	return NewBytesValue(res, false), nil
}

func bytesTypeEqual(v Value, other Value, final bool) bool {
	o := (*Bytes)(v.Ptr)
	switch other.Type {
	case value.Bytes, value.String, value.Runes:
		t, _ := other.AsBytes() // always exact for Bytes/String/Runes
		return bytes.Equal(o.Elements, t)
	case value.Bool, value.Byte, value.Rune, value.Int, value.Decimal, value.Float:
		s, _ := other.AsString() // canonical text form
		return bytes.Equal(o.Elements, []byte(s))
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func bytesTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	o := (*Bytes)(v.Ptr)

	if reflected {
		switch other.Type {
		case value.Byte:
			switch op {
			case token.Add:
				l := []byte{byte(other.Data)}
				t := make([]byte, len(l)+len(o.Elements))
				copy(t, l)
				copy(t[len(l):], o.Elements)
				return NewBytesValue(t, false), nil
			}

		case value.Rune:
			switch op {
			case token.Add:
				l := []byte(string(rune(other.Data)))
				t := make([]byte, len(l)+len(o.Elements))
				copy(t, l)
				copy(t[len(l):], o.Elements)
				return NewBytesValue(t, false), nil
			}

		case value.String, value.Runes:
			l, _ := other.AsBytes() // always succeeds for String/Runes
			switch op {
			case token.Add:
				t := make([]byte, len(l)+len(o.Elements))
				copy(t, l)
				copy(t[len(l):], o.Elements)
				return NewBytesValue(t, false), nil
			case token.Less:
				return BoolValue(bytes.Compare(l, o.Elements) < 0), nil
			case token.LessEq:
				return BoolValue(bytes.Compare(l, o.Elements) <= 0), nil
			case token.Greater:
				return BoolValue(bytes.Compare(l, o.Elements) > 0), nil
			case token.GreaterEq:
				return BoolValue(bytes.Compare(l, o.Elements) >= 0), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Byte:
		switch op {
		case token.Add:
			r := []byte{byte(other.Data)}
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case token.Sub:
			b := byte(other.Data)
			t := make([]byte, 0, len(o.Elements))
			for _, e := range o.Elements {
				if e != b {
					t = append(t, e)
				}
			}
			return NewBytesValue(t, false), nil
		}

	case value.Rune:
		r := []byte(string(rune(other.Data)))
		switch op {
		case token.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case token.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		}

	case value.String:
		r, _ := other.AsBytes() // always succeeds for String
		switch op {
		case token.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case token.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case token.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case token.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case token.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case token.GreaterEq:
			return BoolValue(bytes.Compare(o.Elements, r) >= 0), nil
		}

	case value.Bytes:
		r := (*Bytes)(other.Ptr).Elements
		switch op {
		case token.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case token.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case token.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case token.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case token.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case token.GreaterEq:
			return BoolValue(bytes.Compare(o.Elements, r) >= 0), nil
		}

	case value.Runes:
		r, _ := other.AsBytes() // always succeeds for Runes
		switch op {
		case token.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case token.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case token.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case token.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case token.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case token.GreaterEq:
			return BoolValue(bytes.Compare(o.Elements, r) >= 0), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// bytesRemoveSubsequence returns a copy of elements with every non-overlapping occurrence of sub removed. An
// empty sub is a no-op (removing "nothing" everywhere is otherwise ill-defined) rather than looping forever.
func bytesRemoveSubsequence(elements, sub []byte) []byte {
	if len(sub) == 0 {
		return append([]byte{}, elements...)
	}
	t := make([]byte, 0, len(elements))
	rest := elements
	for {
		i := bytes.Index(rest, sub)
		if i < 0 {
			t = append(t, rest...)
			break
		}
		t = append(t, rest[:i]...)
		rest = rest[i+len(sub):]
	}
	return t
}

// deep is irrelevant here: elements are raw bytes, not nested Values, so there's nothing a shallow copy could
// leave shared. Kept for signature parity with the shared Copy hook.
func bytesTypeCopy(v Value, _ bool) (Value, error) {
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
		return bytesTypeCopy(v, true)

	case "copy_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return bytesTypeCopy(v, false)

	case "freeze_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.ToImmutable()

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

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

	case "string":
		// the UTF-8 decode; decoding is partial, so invalid UTF-8 raises rather
		// than silently substituting U+FFFD (the default slot is the escape)
		ok := utf8.Valid(o.Elements)
		var res Value
		if ok {
			res = NewStringValue(string(o.Elements))
		}
		return convMember(name, bytesTypeName, args, ok, res)

	case "runes":
		// the same decode, materialized as symbols — the mirror of .bytes()
		ok := utf8.Valid(o.Elements)
		var res Value
		if ok {
			res = NewRunesValue([]rune(string(o.Elements)), false)
		}
		return convMember(name, bytesTypeName, args, ok, res)

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

	case "sort_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Immutable {
			return Undefined, errs.NewNotSortableError(v.TypeName())
		}
		slices.Sort(o.Elements)
		return v, nil

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

	case "reverse_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Immutable {
			return Undefined, errs.NewNotReversibleError(v.TypeName())
		}
		slices.Reverse(o.Elements)
		return v, nil

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

	case "chunk_view":
		return SeqChunkView(v, args, NewBytesValue, bytesTypeResolve)

	case "slice":
		return SeqSlice(v, args)

	case "slice_view":
		return SeqSliceView(v, args, NewBytesValue, bytesTypeResolve)

	case "is_view":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(o.IsView), nil

	case "append":
		return bytesTypeAppend(v, args, false)

	case "append_in_place":
		return bytesTypeAppend(v, args, true)

	case "splice_in_place":
		return SeqSplice(append([]Value{v}, args...), true, NewBytesValue, bytesTypeResolve, bytesAppendItems, bytesTypeName)

	case "splice":
		return SeqSplice(append([]Value{v}, args...), false, NewBytesValue, bytesTypeResolve, bytesAppendItems, bytesTypeName)

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
			return Undefined, errs.NewInvalidValueError("(split) separator must not be empty")
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
		return Undefined, errs.NewInvalidValueError("(partition) separator must not be empty")
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
