package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
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

// Bytes is a bytes value's body: its octets.
type Bytes struct {
	Elements []byte
	// IsView reports whether Elements shares backing storage with another value. Only the _view members
	// (slice_view, chunk_view) set it; every other constructor answers an independently-owned body.
	IsView bool
}

// Set replaces the elements — how every _in_place member writes its result back.
func (o *Bytes) Set(elements []byte) {
	o.Elements = elements
}

func NewStaticBytesValue(b *Bytes) Value {
	return Value{Type: value.Bytes, Immutable: true, Ptr: unsafe.Pointer(b)}
}

func NewBytesValue(b []byte, immutable bool) Value {
	o := &Bytes{}
	o.Set(b)
	return Value{Type: value.Bytes, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeBytes = ValueTypeDescr{
	Name:                MutabilityNameHook(bytesTypeName, immutableBytesTypeName),                              // PURE by contract
	String:              bytesTypeString,                                                                        // PURE by contract
	Format:              bytesTypeFormat,                                                                        // PURE by contract
	Interface:           func(v Value) any { return (*Bytes)(v.Ptr).Elements },                                  // PURE by contract
	EncodeJSON:          bytesTypeEncodeJSON,                                                                    // PURE by contract
	EncodeBinary:        bytesTypeEncodeBinary,                                                                  // PURE by contract
	DecodeBinary:        bytesTypeDecodeBinary,                                                                  // IMPURE by contract (mutates target)
	IsTrue:              func(v Value) (bool, error) { return len((*Bytes)(v.Ptr).Elements) > 0, nil },          // PURE by contract
	IsIterable:          ConstHook(true),                                                                        // PURE by contract
	Iterator:            bytesTypeIterator,                                                                      // PURE by contract (constructs fresh iterator)
	Equal:               bytesTypeEqual,                                                                         // PURE by contract
	BinaryOp:            bytesTypeBinaryOp,                                                                      // PURE by contract
	Copy:                bytesTypeCopy,                                                                          // PURE by contract
	Len:                 func(v Value) int64 { return int64(len((*Bytes)(v.Ptr).Elements)) },                    // PURE by contract
	CallNamedMethod:     bytesTypeCallNamedMethod,                                                               // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessIndex:         bytesTypeAccessIndex,                                                                   // PURE by contract
	AccessNamedProperty: noNamedProperty,                                                                        // PURE by contract
	AssignIndex:         bytesTypeAssignIndex,                                                                   // IMPURE by contract
	Append:              bytesTypeAppend,                                                                        // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Contains:            bytesTypeContains,                                                                      // PURE by contract
	Slice:               bytesTypeSlice,                                                                         // PURE by contract
	SliceStep:           bytesTypeSliceStep,                                                                     // PURE by contract
	AsBool:              func(v Value) (bool, bool) { return conv.ParseBool(string((*Bytes)(v.Ptr).Elements)) }, // PURE by contract
	AsString:            func(v Value) (string, bool) { return string((*Bytes)(v.Ptr).Elements), true },         // PURE by contract
	AsBytes:             func(v Value) ([]byte, bool) { return (*Bytes)(v.Ptr).Elements, true },                 // PURE by contract
	AsArray:             bytesTypeAsArray,                                                                       // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (keep/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsNamedMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
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
	items, err := bytesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
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
		s, ok := other.AsString()                       // canonical text form
		return ok && bytes.Equal(o.Elements, []byte(s)) // no text form (a high octet) equals no text
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
			case tokens.Add:
				l := []byte{byte(other.Data)}
				t := make([]byte, len(l)+len(o.Elements))
				copy(t, l)
				copy(t[len(l):], o.Elements)
				return NewBytesValue(t, false), nil
			}

		case value.Rune:
			switch op {
			case tokens.Add:
				l := []byte(string(rune(other.Data)))
				t := make([]byte, len(l)+len(o.Elements))
				copy(t, l)
				copy(t[len(l):], o.Elements)
				return NewBytesValue(t, false), nil
			}

		case value.String, value.Runes:
			// no reflected Add: the RECEIVER — the left operand — decides the result
			// type, so "ab" + bytes("cd") is string's own cell and answers a string
			l, _ := other.AsBytes() // always succeeds for String/Runes
			switch op {
			case tokens.Less:
				return BoolValue(bytes.Compare(l, o.Elements) < 0), nil
			case tokens.LessEq:
				return BoolValue(bytes.Compare(l, o.Elements) <= 0), nil
			case tokens.Greater:
				return BoolValue(bytes.Compare(l, o.Elements) > 0), nil
			case tokens.GreaterEq:
				return BoolValue(bytes.Compare(l, o.Elements) >= 0), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	// `*` is repeat's operator form: the right operand is a COUNT, not text content — a sequence times a
	// number is that sequence n times over. There is no reflected direction: `seq * n` reads as "apply n to
	// the sequence", `n * seq` has no such reading
	if op == tokens.Mul {
		n, isCount, err := SeqRepeatOperand(other)
		if err != nil {
			return Undefined, err
		}
		if isCount {
			src := o.Elements
			sl := len(src)
			total, terr := SeqRepeatTotal(op.String(), n, sl)
			if terr != nil {
				return Undefined, terr
			}
			t := make([]byte, total)
			// step by the receiver's length, never by the count (see the member form)
			for i := 0; i < total; i += sl {
				copy(t[i:], src)
			}
			return NewBytesValue(t, false), nil
		}
	}

	switch other.Type {
	case value.Byte:
		switch op {
		case tokens.Add:
			r := []byte{byte(other.Data)}
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case tokens.Sub:
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
		case tokens.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case tokens.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		}

	case value.String:
		r, _ := other.AsBytes() // always succeeds for String
		switch op {
		case tokens.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case tokens.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case tokens.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case tokens.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case tokens.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case tokens.GreaterEq:
			return BoolValue(bytes.Compare(o.Elements, r) >= 0), nil
		}

	case value.Bytes:
		r := (*Bytes)(other.Ptr).Elements
		switch op {
		case tokens.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case tokens.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case tokens.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case tokens.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case tokens.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case tokens.GreaterEq:
			return BoolValue(bytes.Compare(o.Elements, r) >= 0), nil
		}

	case value.Runes:
		r, _ := other.AsBytes() // always succeeds for Runes
		switch op {
		case tokens.Add:
			t := make([]byte, len(o.Elements)+len(r))
			copy(t, o.Elements)
			copy(t[len(o.Elements):], r)
			return NewBytesValue(t, false), nil
		case tokens.Sub:
			return NewBytesValue(bytesRemoveSubsequence(o.Elements, r), false), nil
		case tokens.Less:
			return BoolValue(bytes.Compare(o.Elements, r) < 0), nil
		case tokens.LessEq:
			return BoolValue(bytes.Compare(o.Elements, r) <= 0), nil
		case tokens.Greater:
			return BoolValue(bytes.Compare(o.Elements, r) > 0), nil
		case tokens.GreaterEq:
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

// PURE by contract
func bytesTypeAccessIndex(v Value, index Value) (Value, error) {
	elems := (*Bytes)(v.Ptr).Elements
	i, err := resolveIndex("index access", index, len(elems))
	if err != nil {
		return Undefined, err
	}
	return ByteValue(elems[i]), nil
}

// IMPURE by contract: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func bytesTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	elems := (*Bytes)(v.Ptr).Elements
	i, err := resolveIndex("index assign", index, len(elems))
	if err != nil {
		return err
	}
	c, ok := r.AsByte()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign value", byteTypeName, r.TypeName())
	}
	elems[i] = c
	return nil
}

// PURE by contract. a[i:j] always answers an independently-owned copy; sharing is slice_view's job.
func bytesTypeSlice(v Value, s Value, e Value) (Value, error) {
	elems := (*Bytes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds("slice", s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]byte, ei-si)
	copy(out, elems[si:ei])
	return NewBytesValue(out, false), nil
}

// PURE by contract
func bytesTypeSliceStep(v Value, s Value, e Value, step Value) (Value, error) {
	elems := (*Bytes)(v.Ptr).Elements
	start, end, st, err := resolveSliceStep(s, e, step, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]byte, 0)
	if st > 0 {
		for i := start; i < end; i += st {
			out = append(out, elems[i])
		}
	} else {
		for i := start; i > end; i += st {
			out = append(out, elems[i])
		}
	}
	return NewBytesValue(out, false), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func bytesTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Bytes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return bytesTypeCopy(v, true)

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

	case "bytes":
		// the same-type conversion constructs — a new, independent, mutable copy, exactly
		// bytes(b) / b.copy(); see the note on array's own case. b"..." literals reach a
		// writable body through this too.
		c, err := bytesTypeCopy(v, false)
		if err != nil {
			return Undefined, err
		}
		return convMember(name, bytesTypeName, args, true, c)

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := bytesTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "string":
		// TOTAL: the decode never fails and never loses an octet — an undecodable one becomes its
		// reserved escape (see text_escape.go), so .string().bytes() returns these octets exactly.
		// is_valid() is how a script asks whether any escape is in there
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(string(o.Elements)), nil

	case "runes":
		// the same decode, materialized as symbols — the mirror of .bytes()
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(DecodeOctets(o.Elements), false), nil

	case "format":
		return memberFormat(vm, v, members.Format, args)

	case "is_ascii":
		// bytes has no is_valid: every octet is a valid octet. The decode question is
		// b.string().is_valid(), which asks it of the type that can answer it
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(OctetsAreASCII(o.Elements)), nil

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
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			return emptySeqResult(name, args)
		}
		return ByteValue(o.Elements[0]), nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			return emptySeqResult(name, args)
		}
		return ByteValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return ByteValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return ByteValue(slices.Max(o.Elements)), nil

	case "contains":
		return bytesContainsMember(vm, v, args)

	case "count":
		return bytesCount(vm, v, args)

	case "keep":
		return bytesKeep(vm, v, args)

	case "keep_in_place":
		return bytesKeepInPlace(vm, v, args)

	case "remove":
		return bytesRemove(vm, v, args)

	case "remove_in_place":
		return bytesRemoveInPlace(vm, v, args)

	case "any":
		return bytesAny(vm, v, args)

	case "all":
		return bytesAll(vm, v, args)

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
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		slices.Sort(o.Elements)
		return v, nil

	case "dedup":
		return bytesDedup(vm, v, args)

	case "dedup_in_place":
		return bytesDedupInPlace(vm, v, args)

	case "unique":
		return bytesUnique(vm, v, args)

	case "unique_in_place":
		return bytesUniqueInPlace(vm, v, args)

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
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		slices.Reverse(o.Elements)
		return v, nil

	case "for_each":
		return bytesForEach(vm, v, args)

	case "index":
		return bytesIndex(vm, v, args)

	case "index_last":
		return bytesIndexLast(vm, v, args)

	case "chunk":
		return bytesChunk(vm, v, args)

	case "chunk_view":
		return bytesChunkView(vm, v, args)

	case "slice":
		return memberSlice(vm, v, members.Slice, args)

	case "slice_view":
		return bytesSliceView(vm, v, args)

	case "append":
		return bytesTypeAppend(v, args, false)

	case "append_in_place":
		return bytesTypeAppend(v, args, true)

	case "prepend":
		return bytesPrepend(vm, v, args)

	case "prepend_in_place":
		return bytesPrependInPlace(vm, v, args)

	case "push":
		return bytesPush(vm, v, args)

	case "push_in_place":
		return bytesPushInPlace(vm, v, args)

	case "push_first":
		return bytesPushFirst(vm, v, args)

	case "push_first_in_place":
		return bytesPushFirstInPlace(vm, v, args)

	case "insert":
		return bytesInsert(vm, v, args)

	case "insert_in_place":
		return bytesInsertInPlace(vm, v, args)

	case "splice":
		return bytesSplice(vm, v, args)

	case "splice_in_place":
		return bytesSpliceInPlace(vm, v, args)

	case "map":
		return bytesMap(vm, v, args)

	case "flat_map":
		return bytesFlatMap(vm, v, args)

	case "reduce":
		return bytesReduce(vm, v, args)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		total, err := SeqRepeatTotal(name, n, sl)
		if err != nil {
			return Undefined, err
		}
		out := make([]byte, total)
		// step by the receiver's length, never by the count: an empty receiver has total 0 and must not
		// spin n times copying nothing
		for i := 0; i < total; i += sl {
			copy(out[i:], src)
		}
		return NewBytesValue(out, false), nil

	case "split":
		return bytesSplit(vm, v, args)

	case "split_lines":
		return bytesFnSplitLines(v, args)

	case "partition":
		return bytesPartition(vm, v, args)

	case "trim":
		return bytesTrim(vm, v, args)

	case "trim_in_place":
		return bytesTrimInPlace(vm, v, args)

	case "trim_start":
		return bytesTrimStart(vm, v, args)

	case "trim_start_in_place":
		return bytesTrimStartInPlace(vm, v, args)

	case "trim_end":
		return bytesTrimEnd(vm, v, args)

	case "trim_end_in_place":
		return bytesTrimEndInPlace(vm, v, args)

	case "has_prefix":
		return bytesHasPrefix(vm, v, args)

	case "has_suffix":
		return bytesHasSuffix(vm, v, args)

	case "remove_prefix":
		return bytesRemovePrefix(vm, v, args)

	case "remove_prefix_in_place":
		return bytesRemovePrefixInPlace(vm, v, args)

	case "remove_suffix":
		return bytesRemoveSuffix(vm, v, args)

	case "remove_suffix_in_place":
		return bytesRemoveSuffixInPlace(vm, v, args)

	case "replace":
		return bytesReplace(vm, v, args)

	case "replace_in_place":
		return bytesReplaceInPlace(vm, v, args)

	case "pad_start":
		return bytesPadStart(vm, v, args)

	case "pad_start_in_place":
		return bytesPadStartInPlace(vm, v, args)

	case "pad_end":
		return bytesPadEnd(vm, v, args)

	case "pad_end_in_place":
		return bytesPadEndInPlace(vm, v, args)

	default:
		return CallMemberByLookup(vm, v, name, args)
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

// bytesEncodeMatchArg: acceptance on a bytes receiver — every accepted argument
// is text content as OCTETS. byte/rune/in-range int are the element class (a
// rune contributes its UTF-8 octets, 1-4 of them); string/runes/bytes are the
// run class; everything else has no reading here and raises. Range failures
// name the range, not the type.
func bytesEncodeMatchArg(name string, a Value) ([]byte, bool, error) {
	switch a.Type {
	case value.Byte:
		return []byte{byte(a.Data)}, true, nil
	case value.Int:
		i := int64(a.Data)
		if i < 0 || i > 255 {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("(%s) an int reads as one octet and must be in [0, 255], got %d", name, i))
		}
		return []byte{byte(i)}, true, nil
	case value.Rune:
		return []byte(string(rune(a.Data))), true, nil
	case value.String, value.Runes:
		b, _ := a.AsBytes()
		return b, false, nil
	case value.Bytes:
		return (*Bytes)(a.Ptr).Elements, false, nil
	}
	return nil, false, errs.NewInvalidArgumentTypeError(name, "argument", "text content (octets, symbols, or text)", a.TypeName())
}

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all.
//
// On a text receiver every accepted argument is TEXT CONTENT (bytesEncodeMatchArg), matched as a run of octets —
// a single octet is just a run of one, and a multi-octet rune is a run:
//   - no argument      the significant octets (not IsBlankByte)
//   - a function       a predicate, f/1(octet) or f/2(index, octet)
//   - text content     a run; several arguments form a set of runs
//
// any/all ask about single octets, so they take octets, a predicate, or nothing — never a run. A set may not mix
// element-typed arguments (byte/rune/int) with text-typed ones (string/runes/bytes), and a function among several
// arguments raises. Runs match leftmost-longest and never overlap.
// ---------------------------------------------------------------------------

// bytesMatch is a match member's argument list after reading; exactly one reading is set.
type bytesMatch struct {
	significant bool     // no argument
	pred        Value    // a predicate
	elems       []byte   // any/all: a set of octets
	runs        [][]byte // contains/count/keep/remove: a set of runs
}

// bytesReadMatchArgs reads a match member's arguments. octets is true for any/all, which read each argument as
// exactly one octet instead of a run.
func bytesReadMatchArgs(name string, args []Value, octets bool) (bytesMatch, error) {
	if len(args) == 0 {
		return bytesMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return bytesMatch{}, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return bytesMatch{}, err
		}
		return bytesMatch{pred: args[0]}, nil
	}

	runs := make([][]byte, 0, len(args))
	sawSymbol, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return bytesMatch{}, errFunctionInSet(name)
		}
		run, isSymbol, err := bytesEncodeMatchArg(name, a)
		if err != nil {
			return bytesMatch{}, err
		}
		if isSymbol {
			sawSymbol = true
		} else {
			sawText = true
		}
		runs = append(runs, run)
	}
	if sawSymbol && sawText {
		return bytesMatch{}, errMixedSet(name)
	}

	if !octets {
		return bytesMatch{runs: runs}, nil
	}
	if sawText {
		return bytesMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "a value, a function, or nothing (the contiguous-run query is contains's)", "sequence")
	}
	elems := make([]byte, 0, len(runs))
	for _, r := range runs {
		if len(r) != 1 {
			return bytesMatch{}, errNotOneElement(name)
		}
		elems = append(elems, r[0])
	}
	return bytesMatch{elems: elems}, nil
}

// matches reports whether octet r at index i matches an element-wise reading (anything but runs).
func (m *bytesMatch) matches(vm VM, i int, r byte) (bool, error) {
	switch {
	case m.significant:
		return !IsBlankByte(r), nil
	case m.pred.IsCallable():
		res, err := callElem(vm, m.pred, i, ByteValue(r))
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	return slices.Contains(m.elems, r), nil
}

// bytesContainsMember is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
func bytesContainsMember(vm VM, v Value, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements

	if m.runs != nil {
		for _, r := range m.runs {
			if len(r) == 0 {
				return True, nil
			}
		}
		for i := range elems {
			if runLengthAt(elems, i, m.runs) > 0 {
				return True, nil
			}
		}
		return False, nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// bytesCount is count(...): how many matches. Runs count non-overlapping occurrences.
func bytesCount(vm VM, v Value, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements

	n := int64(0)
	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				n++
				i += k
			} else {
				i++
			}
		}
		return IntValue(n), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// bytesKept answers the octets keep(...) keeps: the matches, in order. Shared by keep and keep_in_place; name is
// the member called, for the errors.
func bytesKept(vm VM, name string, v Value, args []Value) ([]byte, error) {
	m, err := bytesReadMatchArgs(name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				out = append(out, elems[i:i+k]...)
				i += k
			} else {
				i++
			}
		}
		return out, nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return nil, err
		}
		if hit {
			out = append(out, r)
		}
	}
	return out, nil
}

// bytesKeep is keep(...): new bytes of the matches.
func bytesKeep(vm VM, v Value, args []Value) (Value, error) {
	out, err := bytesKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func bytesKeepInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesRemaining answers the octets remove(...) leaves: everything but the matches, in order. With no argument
// remove drops the BLANK octets — so it keeps the significant ones, landing on keep()'s answer by the opposite
// action. Shared by remove and remove_in_place; name is the member called, for the errors.
func bytesRemaining(vm VM, name string, v Value, args []Value) ([]byte, error) {
	m, err := bytesReadMatchArgs(name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))

	if m.significant {
		for _, r := range elems {
			if !IsBlankByte(r) {
				out = append(out, r)
			}
		}
		return out, nil
	}

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := runLengthAt(elems, i, m.runs); k > 0 {
				i += k
			} else {
				out = append(out, elems[i])
				i++
			}
		}
		return out, nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return nil, err
		}
		if !hit {
			out = append(out, r)
		}
	}
	return out, nil
}

// bytesRemove is remove(...): new bytes without the matches.
func bytesRemove(vm VM, v Value, args []Value) (Value, error) {
	out, err := bytesRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func bytesRemoveInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesAny is any(...): does some octet match? Symbols, a predicate, or nothing — never a run.
func bytesAny(vm VM, v Value, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Bytes)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// bytesAll is all(...): does every octet match? Symbols, a predicate, or nothing — never a run. True on empty
// bytes.
func bytesAll(vm VM, v Value, args []Value) (Value, error) {
	m, err := bytesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Bytes)(v.Ptr).Elements {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
}

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match,
// counted in octets. The argument's type selects the reading:
//   - no argument      the first / last significant octet (not IsBlankByte)
//   - a function       a predicate, f/1(octet) or f/2(index, octet)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one octet, compared with ==; a value that is not one octet raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// bytesIndex is index(...): the first match.
func bytesIndex(vm VM, v Value, args []Value) (Value, error) {
	return bytesLocate(vm, "index", v, args, false)
}

// bytesIndexLast is index_last(...): the last match.
func bytesIndexLast(vm VM, v Value, args []Value) (Value, error) {
	return bytesLocate(vm, "index_last", v, args, true)
}

// bytesLocate is the body of index and index_last; name is the member called, for the errors.
func bytesLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	elems := (*Bytes)(v.Ptr).Elements

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !IsBlankByte(e) {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, nil)
	}

	needle, dflt := args[0], args[1:]

	if needle.IsCallable() {
		if err := checkElemCallback(name, needle); err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := callElem(vm, needle, i, ByteValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				idx = i
				if !last {
					break
				}
			}
		}
		return locatorAnswer(idx, dflt)
	}

	if needle.Type == value.String || needle.Type == value.Runes || needle.Type == value.Bytes {
		run, ok := needle.AsBytes()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "text content", needle.TypeName())
		}
		return locatorAnswer(bytesIndexRun(elems, run, last), dflt)
	}

	enc, isElement, err := bytesEncodeMatchArg(name, needle)
	if err != nil {
		return Undefined, err
	}
	if !isElement || len(enc) != 1 {
		return Undefined, errNotOneElement(name)
	}
	idx := -1
	for i, e := range elems {
		if ByteValue(e).Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// bytesIndexRun is indexRun on octets, through the standard library's searcher.
func bytesIndexRun(elems, run []byte, last bool) int {
	if len(run) == 0 || len(run) > len(elems) {
		return -1
	}
	if last {
		return bytes.LastIndex(elems, run)
	}
	return bytes.Index(elems, run)
}

// ---------------------------------------------------------------------------
// The structural members: the trim family, the anchored pair (has_/remove_ prefix/suffix), replace, and the pads.
// Every argument is text content (bytesEncodeMatchArg); none of them takes a predicate.
//   - trim*            a set of single octets, stripped repeat-while; no argument = the blank set (IsBlankByte)
//   - *_prefix/suffix  a homogeneous set of runs; the longest anchored run wins; remove_* removes it once
//   - replace(old, new) every occurrence of the run old, leftmost and non-overlapping; an empty old matches nothing
//   - pad_*(n[, fill]) n counts octets; the fill is exactly one octet, the space by default
// ---------------------------------------------------------------------------

// bytesReadElementSet reads a set of single octets (the trim family's set, the pads' fill): each argument must be
// one octet. A text-typed argument or a function raises with refuse, the member's own statement of what it
// takes.
func bytesReadElementSet(name string, args []Value, refuse string) ([]byte, error) {
	set := make([]byte, 0, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
		enc, isElement, err := bytesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		if !isElement {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
		if len(enc) != 1 {
			return nil, errNotOneElement(name)
		}
		set = append(set, enc[0])
	}
	return set, nil
}

// bytesReadRunSet reads a homogeneous set of runs (the anchored pair, split, partition): an element-typed argument
// is a run of its encoding, but one call may not mix element-typed and text-typed arguments. A function raises
// with onFunction, the member's own statement of what it takes.
func bytesReadRunSet(name string, args []Value, onFunction string) ([][]byte, error) {
	runs := make([][]byte, 0, len(args))
	sawElement, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		}
		run, isElement, err := bytesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		if isElement {
			sawElement = true
		} else {
			sawText = true
		}
		runs = append(runs, run)
	}
	if sawElement && sawText {
		return nil, errMixedSet(name)
	}
	return runs, nil
}

// bytesTrimmed answers the receiver without its leading (start) and/or trailing (end) octets that belong to the
// set. The name is the member called, for the errors.
func bytesTrimmed(name string, v Value, args []Value, start, end bool) ([]byte, error) {
	set, err := bytesReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e byte) bool {
		if len(args) == 0 {
			return IsBlankByte(e)
		}
		return slices.Contains(set, e)
	}
	elems := (*Bytes)(v.Ptr).Elements
	lo, hi := 0, len(elems)
	if start {
		for lo < hi && inSet(elems[lo]) {
			lo++
		}
	}
	if end {
		for hi > lo && inSet(elems[hi-1]) {
			hi--
		}
	}
	return slices.Clone(elems[lo:hi]), nil
}

// bytesAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func bytesAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := bytesReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	best := -1
	for _, r := range runs {
		if len(r) > len(elems) || len(r) <= best {
			continue
		}
		at := elems[:len(r)]
		if suffix {
			at = elems[len(elems)-len(r):]
		}
		if slices.Equal(at, r) {
			best = len(r)
		}
	}
	return best, nil
}

// bytesWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func bytesWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]byte, error) {
	best, err := bytesAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	switch {
	case best <= 0:
		return slices.Clone(elems), nil
	case suffix:
		return slices.Clone(elems[:len(elems)-best]), nil
	default:
		return slices.Clone(elems[best:]), nil
	}
}

// bytesReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); each
// argument is read on its own, an element being a run of one. The name is the member called, for the errors.
func bytesReplaced(name string, v Value, args []Value) ([]byte, error) {
	if len(args) != 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].IsCallable() || args[1].IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	old, _, err := bytesEncodeMatchArg(name, args[0])
	if err != nil {
		return nil, err
	}
	repl, _, err := bytesEncodeMatchArg(name, args[1])
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))
	for i := 0; i < len(elems); {
		if len(old) > 0 && i+len(old) <= len(elems) && slices.Equal(elems[i:i+len(old)], old) {
			out = append(out, repl...)
			i += len(old)
		} else {
			out = append(out, elems[i])
			i++
		}
	}
	return out, nil
}

// bytesPadded answers the receiver filled at the front (start) or the end up to n octets; a width at or below the
// length leaves it unchanged. The name is the member called, for the errors.
func bytesPadded(name string, v Value, args []Value, start bool) ([]byte, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := byte(' ')
	if len(args) == 2 {
		set, err := bytesReadElementSet(name, args[1:], "one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return nil, err
		}
		fill = set[0]
	}
	elems := (*Bytes)(v.Ptr).Elements
	if n <= int64(len(elems)) {
		return slices.Clone(elems), nil
	}
	width, err := SeqPadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]byte{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// bytesTrim is trim(...): without the leading and trailing elements of the set.
func bytesTrim(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func bytesTrimInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesTrimStart is trim_start(...): without the leading elements of the set.
func bytesTrimStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func bytesTrimStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesTrimEnd is trim_end(...): without the trailing elements of the set.
func bytesTrimEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func bytesTrimEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesTrimmed(name, v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
func bytesRemovePrefix(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func bytesRemovePrefixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_prefix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesWithoutAnchored(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
func bytesRemoveSuffix(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func bytesRemoveSuffixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_suffix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesWithoutAnchored(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesReplace is replace(...): every occurrence of old replaced by new.
func bytesReplace(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func bytesReplaceInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "replace_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesReplaced(name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesPadStart is pad_start(...): filled at the front up to n elements.
func bytesPadStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func bytesPadStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesPadded(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesPadEnd is pad_end(...): filled at the end up to n elements.
func bytesPadEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := bytesPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func bytesPadEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := bytesPadded(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Bytes)(v.Ptr).Set(out)
	return v, nil
}

// bytesHasPrefix is has_prefix(...): does the receiver start with one of the runs?
func bytesHasPrefix(_ VM, v Value, args []Value) (Value, error) {
	best, err := bytesAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// bytesHasSuffix is has_suffix(...): does the receiver end with one of the runs?
func bytesHasSuffix(_ VM, v Value, args []Value) (Value, error) {
	best, err := bytesAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// ---------------------------------------------------------------------------
// split(...seps) and partition(...seps): a separator is a run (an element being a run of one), a homogeneous set
// of runs, a predicate on single octets, or — no argument — the blank set (IsBlankByte). Runs match leftmost-longest
// and never overlap; an empty run matches nothing.
// ---------------------------------------------------------------------------

// bytesSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant octets, the
// classic whitespace split.
func bytesSplit(vm VM, v Value, args []Value) (Value, error) {
	const name = "split"
	elems := (*Bytes)(v.Ptr).Elements

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankByte(e) {
				if start >= 0 {
					pieces = append(pieces, bytesPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, bytesPiece(elems[start:]))
		}
		return NewArrayValue(pieces, false), nil
	}

	pieces := make([]Value, 0, 4)
	start := 0

	if args[0].IsCallable() {
		if len(args) > 1 {
			return Undefined, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return Undefined, err
		}
		for i, e := range elems {
			res, err := callElem(vm, args[0], i, ByteValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				pieces = append(pieces, bytesPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, bytesPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := bytesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, bytesPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, bytesPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// bytesPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
func bytesPartition(vm VM, v Value, args []Value) (Value, error) {
	const name = "partition"
	elems := (*Bytes)(v.Ptr).Elements
	found, n := -1, 0

	switch {
	case len(args) == 0:
		for i, e := range elems {
			if IsBlankByte(e) {
				found, n = i, 1
				for found+n < len(elems) && IsBlankByte(elems[found+n]) {
					n++
				}
				break
			}
		}

	case args[0].IsCallable():
		if len(args) > 1 {
			return Undefined, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return Undefined, err
		}
		for i, e := range elems {
			res, err := callElem(vm, args[0], i, ByteValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				found, n = i, 1
				break
			}
		}

	default:
		runs, err := bytesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
		if err != nil {
			return Undefined, err
		}
		for i := range elems {
			if k := runLengthAt(elems, i, runs); k > 0 {
				found, n = i, k
				break
			}
		}
	}

	if found < 0 {
		return NewArrayValue([]Value{bytesPiece(elems), bytesPiece(nil), bytesPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{bytesPiece(elems[:found]), bytesPiece(elems[found : found+n]), bytesPiece(elems[found+n:])}, false), nil
}

// bytesPiece is one piece of split/partition: new bytes, independent of the receiver.
func bytesPiece(elems []byte) Value {
	return NewBytesValue(slices.Clone(elems), false)
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(octet) or
// f/2(index, octet); reduce's is f/2(acc, octet) or f/3(acc, index, octet).
// ---------------------------------------------------------------------------

// bytesForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
func bytesForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Bytes)(v.Ptr).Elements {
		if _, err := callElem(vm, fn, i, ByteValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// bytesMap is map(f): strictly 1:1, answering bytes — each callback result must be exactly one octet (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
func bytesMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered undefined — map is 1:1; the dropping form is flat_map")
		}
		enc, isElement, err := bytesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		if !isElement {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered a sequence — map is 1:1; the concatenating form is flat_map")
		}
		if len(enc) != 1 {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback result does not fit a single element of the receiver")
		}
		out[i] = enc[0]
	}
	return NewBytesValue(out, false), nil
}

// bytesFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
func bytesFlatMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			continue
		}
		enc, _, err := bytesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		out = append(out, enc...)
	}
	return NewBytesValue(out, false), nil
}

// bytesReduce is reduce(acc, f): folds the octets left to right.
func bytesReduce(vm VM, v Value, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Bytes)(v.Ptr).Elements {
		acc, err = callReduce(vm, fn, acc, i, ByteValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// ---------------------------------------------------------------------------
// The add and edit members: chunk, slice_view, splice, insert, prepend, push, push_first (append is the Append
// hook, bytesTypeAppend; slice is the Slice hook's member spelling). The add side reads text content:
// append/prepend/splice's inserts are runs (an element being a run of one), push/push_first/insert take exactly
// one octet per argument.
// ---------------------------------------------------------------------------

// bytesAddItems reads the add side's operands (append, prepend, splice's inserts): every argument is text content,
// concatenated in argument order — mixing elements and runs is fine here (x.append("ab", 'c') is x + "ab" + 'c').
func bytesAddItems(name string, args []Value) ([]byte, error) {
	items := make([]byte, 0, len(args))
	for _, a := range args {
		enc, _, err := bytesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		items = append(items, enc...)
	}
	return items, nil
}

// bytesPushItems reads push/push_first/insert's elements: each argument must be exactly one octet — a text
// argument raises even at length 1, which is the member's purpose (append/prepend take runs).
func bytesPushItems(name string, args []Value) ([]byte, error) {
	return bytesReadElementSet(name, args, "one element (a sequence argument never reads as an element here; append/prepend take runs)")
}

// bytesChunk is chunk(size): the octets in pieces of size (the last one shorter), each an independent copy.
func bytesChunk(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewBytesValue(slices.Clone(elems[start:min(start+step, len(elems))]), false))
	}
	return NewArrayValue(chunks, false), nil
}

// bytesChunkView is chunk_view(size): chunk's pieces sharing the receiver's storage, each marked as a view.
func bytesChunkView(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk_view", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		c := NewBytesValue(elems[start:min(start+step, len(elems))], v.Immutable)
		(*Bytes)(c.Ptr).IsView = true
		chunks = append(chunks, c)
	}
	return NewArrayValue(chunks, false), nil
}

// bytesSliceView is slice_view([start[, end]]): the span sharing the receiver's storage, marked as a view.
func bytesSliceView(_ VM, v Value, args []Value) (Value, error) {
	const name = "slice_view"
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds(name, s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	res := NewBytesValue(elems[si:ei], v.Immutable)
	(*Bytes)(res.Ptr).IsView = true
	return res, nil
}

// bytesSplice is splice([start[, count[, ...items]]]): a new bytes with count octets at start replaced by
// the items (read like append's operands).
func bytesSplice(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Bytes)(v.Ptr).Elements
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []byte
	if len(args) > 2 {
		if items, err = bytesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]byte, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesSpliceInPlace is splice_in_place(...): splice applied to the receiver itself.
// A side-effecting member returns the RECEIVER, so mutators chain; the removed span is x.slice(i, j) taken
// beforehand.
func bytesSpliceInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	start, end, err := readSpliceRange(args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	var items []byte
	if len(args) > 2 {
		if items, err = bytesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	o.Set(append(o.Elements[:start], append(items, o.Elements[end:]...)...))
	return v, nil
}

// bytesInsert is insert(i, ...items): a new bytes with the items — one octet each — at position i, which
// raises out of [0, len].
func bytesInsert(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert"
	elems := (*Bytes)(v.Ptr).Elements
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := bytesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	out := slices.Insert(slices.Clone(elems), at, items...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func bytesInsertInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	at, err := readEditPos(name, args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	items, err := bytesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	o.Set(slices.Insert(o.Elements, at, items...))
	return v, nil
}

// bytesPrepend is prepend(...items): a new bytes with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
func bytesPrepend(_ VM, v Value, args []Value) (Value, error) {
	items, err := bytesAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func bytesPrependInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "prepend_in_place"
	items, err := bytesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// bytesPush is push(...items): a new bytes with the items — exactly one octet each — at the end.
func bytesPush(_ VM, v Value, args []Value) (Value, error) {
	items, err := bytesPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems)+len(items))
	out = append(out, elems...)
	out = append(out, items...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPushInPlace is push_in_place(...items): push applied to the receiver itself.
func bytesPushInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "push_in_place"
	items, err := bytesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(append(o.Elements, items...))
	return v, nil
}

// bytesPushFirst is push_first(...items): a new bytes with the items — exactly one octet each — in front,
// in argument order.
func bytesPushFirst(_ VM, v Value, args []Value) (Value, error) {
	items, err := bytesPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewBytesValue(out, false), nil
}

// IMPURE: mutates the receiver. bytesPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func bytesPushFirstInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "push_first_in_place"
	items, err := bytesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// bytesDeduped answers the elements with each run of equal neighbours collapsed to one.
func bytesDeduped(elems []byte) []byte {
	out := make([]byte, 0, len(elems))
	for i, e := range elems {
		if i == 0 || e != elems[i-1] {
			out = append(out, e)
		}
	}
	return out
}

// bytesDedup is dedup(): a new bytes with each run of equal neighbours collapsed to one.
func bytesDedup(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewBytesValue(bytesDeduped((*Bytes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. bytesDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func bytesDedupInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "dedup_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(bytesDeduped(o.Elements))
	return v, nil
}

// bytesUniqueElements answers the elements without repeats, each kept at its first occurrence.
func bytesUniqueElements(elems []byte) []byte {
	out := make([]byte, 0, len(elems))
	var seen [256]bool
	for _, e := range elems {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

// bytesUnique is unique(): a new bytes without repeats, each element kept at its first occurrence.
func bytesUnique(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewBytesValue(bytesUniqueElements((*Bytes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. bytesUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func bytesUniqueInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "unique_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Bytes)(v.Ptr)
	o.Set(bytesUniqueElements(o.Elements))
	return v, nil
}

// bytesTypeContains is the `in` operator: every accepted operand is text content as OCTETS, matched
// as a run (the member's own acceptance — an out-of-range int now raises, never a silent false); a
// callable raises.
func bytesTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	run, _, err := bytesEncodeMatchArg("in", e)
	if err != nil {
		return false, err
	}
	return bytes.Contains((*Bytes)(v.Ptr).Elements, run), nil
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
