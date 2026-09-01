package core

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"slices"
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
	// methods (keep/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
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

// bytesAppendItems flattens the add side's variadic operands (append/prepend/splice inserts) into octets via
// the receiver's acceptance table — every accepted argument is text content, an element contributing its
// encoding and a run its content, in argument order. methodName keeps errors reading correctly from every
// caller.
func bytesAppendItems(args []Value, methodName string) ([]byte, error) {
	return tripleAddItems(methodName, args, bytesEncodeMatchArg)
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

// bytesTypeAddFront implements prepend/prepend_in_place: whole-operand concatenation at the FRONT, arguments
// staying in order — x.prepend(a, b) ≡ a + b + x. Same purity split as bytesTypeAppend.
func bytesTypeAddFront(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Bytes)(v.Ptr)
	name := "prepend"
	if mutate {
		name = "prepend_in_place"
	}
	items, err := bytesAppendItems(args, name)
	if err != nil {
		return Undefined, err
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		// slices.Insert reuses the receiver's backing array whenever capacity allows
		o.Set(slices.Insert(o.Elements, 0, items...))
		return v, nil
	}

	res := make([]byte, 0, len(items)+len(o.Elements))
	res = append(res, items...)
	res = append(res, o.Elements...)
	return NewBytesValue(res, false), nil
}

// bytesTypePush implements push/push_first and their _in_place twins: the VALIDATING element add — each
// argument must be a single octet (a sequence argument raises even at length 1, and so does a multi-octet
// rune); the refusal is the member's purpose. Arguments stay in order at the front too. Same purity split as
// bytesTypeAppend.
func bytesTypePush(v Value, args []Value, mutate bool, front bool) (Value, error) {
	o := (*Bytes)(v.Ptr)
	name := "push"
	if front {
		name = "push_first"
	}
	if mutate {
		name += "_in_place"
	}
	items, err := triplePushItems(name, args, bytesEncodeMatchArg)
	if err != nil {
		return Undefined, err
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		if front {
			o.Set(slices.Insert(o.Elements, 0, items...))
		} else {
			o.Set(append(o.Elements, items...))
		}
		return v, nil
	}

	res := make([]byte, 0, len(o.Elements)+len(items))
	if front {
		res = append(res, items...)
		res = append(res, o.Elements...)
	} else {
		res = append(res, o.Elements...)
		res = append(res, items...)
	}
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
			// no reflected Add: the RECEIVER — the left operand — decides the result
			// type, so "ab" + bytes("cd") is string's own cell and answers a string
			l, _ := other.AsBytes() // always succeeds for String/Runes
			switch op {
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

	// `*` is repeat's operator form: the right operand is a COUNT, not text content — a sequence times a
	// number is that sequence n times over. There is no reflected direction: `seq * n` reads as "apply n to
	// the sequence", `n * seq` has no such reading
	if op == token.Mul {
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
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return ByteValue(o.Elements[0]), nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return ByteValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return ByteValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return ByteValue(slices.Max(o.Elements)), nil

	case "contains", "count", "keep", "remove", "any", "all", "remove_in_place", "keep_in_place":
		mutate := strings.HasSuffix(name, "_in_place")
		if mutate && v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		res, err := TripleMatchMember(vm, name, v, args, ByteValue, NewBytesValue, bytesTypeResolve,
			bytesEncodeMatchArg, func(a, b byte) bool { return a == b }, IsBlankByte)
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set((*Bytes)(res.Ptr).Elements)
			return v, nil
		}
		return res, nil

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

	case "dedup", "dedup_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if name == "dedup_in_place" && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		out := make([]byte, 0, len(o.Elements))
		for i, b := range o.Elements {
			if i == 0 || b != o.Elements[i-1] {
				out = append(out, b)
			}
		}
		if name == "dedup_in_place" {
			o.Set(out)
			return v, nil
		}
		return NewBytesValue(out, false), nil

	case "unique", "unique_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if name == "unique_in_place" && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		out := make([]byte, 0, len(o.Elements))
		var seen [256]bool
		for _, b := range o.Elements {
			if !seen[b] {
				seen[b] = true
				out = append(out, b)
			}
		}
		if name == "unique_in_place" {
			o.Set(out)
			return v, nil
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
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		slices.Reverse(o.Elements)
		return v, nil

	case "for_each":
		return SeqForEach(vm, v, args, ByteValue, bytesTypeResolve)

	case "index", "index_last":
		// the locator: element | run | predicate | absent(blank), plus [default];
		// offsets are octet positions on this receiver
		return SeqIndex(vm, v, args, name == "index_last", ByteValue, bytesTypeResolve,
			func(a Value) bool { return a.Type == value.String || a.Type == value.Runes || a.Type == value.Bytes },
			func(elems []byte, run Value, last bool) (int64, bool, error) {
				b, ok := run.AsBytes()
				if !ok {
					return -1, false, errs.NewInvalidArgumentTypeError(name, "first", "text content", run.TypeName())
				}
				if len(b) == 0 || len(b) > len(elems) {
					return -1, false, nil
				}
				if last {
					i := bytes.LastIndex(elems, b)
					return int64(i), i >= 0, nil
				}
				i := bytes.Index(elems, b)
				return int64(i), i >= 0, nil
			},
			tripleElemCheck(bytesEncodeMatchArg),
			IsBlankByte)

	case "chunk":
		return SeqChunk(v, args, NewBytesValue, bytesTypeResolve)

	case "chunk_view":
		return SeqChunkView(v, args, NewBytesValue, bytesTypeResolve)

	case "slice":
		return SeqSlice(v, args)

	case "slice_view":
		return SeqSliceView(v, args, NewBytesValue, bytesTypeResolve)

	case "append":
		return bytesTypeAppend(v, args, false)

	case "append_in_place":
		return bytesTypeAppend(v, args, true)

	case "prepend":
		return bytesTypeAddFront(v, args, false)

	case "prepend_in_place":
		return bytesTypeAddFront(v, args, true)

	case "push":
		return bytesTypePush(v, args, false, false)

	case "push_in_place":
		return bytesTypePush(v, args, true, false)

	case "push_first":
		return bytesTypePush(v, args, false, true)

	case "push_first_in_place":
		return bytesTypePush(v, args, true, true)

	case "insert", "insert_in_place":
		// the element-inserting sibling of splice: each item must be a single element
		// (a sequence raises even at length 1 — splice takes runs); the position is a
		// positional EDIT and raises out of range
		mutate := name == "insert_in_place"
		if mutate && v.Immutable {
			return Undefined, errs.NewNotMutableError(name, v.TypeName())
		}
		at, err := seqEditPos(name, args, int64(len(o.Elements)))
		if err != nil {
			return Undefined, err
		}
		items, err := triplePushItems(name, args[1:], bytesEncodeMatchArg)
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set(slices.Insert(o.Elements, int(at), items...))
			return v, nil
		}
		return NewBytesValue(slices.Insert(slices.Clone(o.Elements), int(at), items...), false), nil

	case "splice_in_place":
		return SeqSplice(append([]Value{v}, args...), true, NewBytesValue, bytesTypeResolve, bytesAppendItems, bytesTypeName)

	case "splice":
		return SeqSplice(append([]Value{v}, args...), false, NewBytesValue, bytesTypeResolve, bytesAppendItems, bytesTypeName)

	case "map":
		// strictly 1:1, answering the receiver's type — a sequence or undefined
		// callback result raises; the concatenating/dropping form is flat_map
		return TripleMapMember(vm, name, v, args, ByteValue, NewBytesValue, bytesTypeResolve, bytesEncodeMatchArg)

	case "flat_map":
		return TripleFlatMapMember(vm, name, v, args, ByteValue, NewBytesValue, bytesTypeResolve, bytesEncodeMatchArg)

	case "reduce":
		return SeqReduce(vm, v, args, ByteValue, bytesTypeResolve)

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
		return SeqSplitMember(vm, name, v, args, ByteValue, NewBytesValue, bytesTypeResolve,
			bytesEncodeMatchArg, func(a, b byte) bool { return a == b }, IsBlankByte)

	case "split_lines":
		return bytesFnSplitLines(v, args)

	case "partition":
		return SeqPartitionMember(vm, name, v, args, ByteValue, NewBytesValue, bytesTypeResolve,
			bytesEncodeMatchArg, func(a, b byte) bool { return a == b }, IsBlankByte)

	case "trim", "trim_start", "trim_end", "has_prefix", "has_suffix",
		"remove_prefix", "remove_suffix", "replace", "pad_start", "pad_end",
		"trim_in_place", "trim_start_in_place", "trim_end_in_place",
		"remove_prefix_in_place", "remove_suffix_in_place", "replace_in_place",
		"pad_start_in_place", "pad_end_in_place":
		// octet width, literal-octet fills and sets — well-defined on binary data.
		// The default fill and blank set are the space octet and NUL ∪ ASCII whitespace
		mutate := strings.HasSuffix(name, "_in_place")
		if mutate && v.Immutable {
			return Undefined, immutableTwinError(name, v.TypeName())
		}
		res, err := SeqStructuralMember(name, v, args, NewBytesValue, bytesTypeResolve,
			bytesEncodeMatchArg, tripleFillElement(bytesEncodeMatchArg), byte(' '),
			func(a, b byte) bool { return a == b }, IsBlankByte)
		if err != nil {
			return Undefined, err
		}
		if mutate {
			o.Set((*Bytes)(res.Ptr).Elements)
			return v, nil
		}
		return res, nil

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
