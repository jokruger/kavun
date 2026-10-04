package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const (
	runesTypeName          = "runes"
	immutableRunesTypeName = "immutable-runes"
)

// Runes is a runes value's body: its symbols.
type Runes struct {
	Elements []rune
	// IsView reports whether Elements shares backing storage with another value. Only the _view members
	// (slice_view, chunk_view) set it; every other constructor answers an independently-owned body.
	IsView bool
}

// Set replaces the elements — how every _in_place member writes its result back.
func (o *Runes) Set(elements []rune) {
	o.Elements = elements
}

func NewStaticRunesValue(r *Runes) Value {
	return Value{Type: value.Runes, Immutable: true, Ptr: unsafe.Pointer(r)}
}

func NewRunesValue(r []rune, immutable bool) Value {
	o := &Runes{}
	o.Set(r)
	return Value{Type: value.Runes, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeRunes = ValueTypeDescr{
	Name:         MutabilityNameHook(runesTypeName, immutableRunesTypeName),                                 // PURE by contract
	String:       func(v Value) string { return "u" + strconv.Quote(EncodeText((*Runes)(v.Ptr).Elements)) }, // PURE by contract
	Format:       runesTypeFormat,                                                                           // PURE by contract
	Interface:    func(v Value) any { return (*Runes)(v.Ptr).Elements },                                     // PURE by contract
	EncodeJSON:   runesTypeEncodeJSON,                                                                       // PURE by contract
	EncodeBinary: runesTypeEncodeBinary,                                                                     // PURE by contract
	DecodeBinary: runesTypeDecodeBinary,                                                                     // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return len((*Runes)(v.Ptr).Elements) > 0, nil },             // PURE by contract
	IsIterable:   ConstHook(true),                                                                           // PURE by contract
	Iterator:     runesTypeIterator,                                                                         // PURE by contract (constructs fresh iterator)
	Copy:         runesTypeCopy,                                                                             // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Runes)(v.Ptr).Elements)) },                       // PURE by contract
	Equal:        runesTypeEqual,                                                                            // PURE by contract
	BinaryOp:     runesTypeBinaryOp,                                                                         // PURE by contract
	MethodCall:   runesTypeMethodCall,                                                                       // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       runesTypeAccess,                                                                           // PURE by contract
	Assign:       runesTypeAssign,                                                                           // IMPURE by contract
	Append:       runesTypeAppend,                                                                           // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Contains:     runesTypeContains,                                                                         // PURE by contract
	Slice:        runesTypeSlice,                                                                            // PURE by contract
	SliceStep:    runesTypeSliceStep,                                                                        // PURE by contract
	AsBool:       runesTypeAsBool,                                                                           // PURE by contract
	AsInt:        runesTypeAsInt,                                                                            // PURE by contract
	AsFloat:      runesTypeAsFloat,                                                                          // PURE by contract
	AsDecimal:    runesTypeAsDecimal,                                                                        // PURE by contract
	AsTime:       runesTypeAsTime,                                                                           // PURE by contract
	AsDate: func(v Value) (civil.Date, bool) {
		d, err := ParseDateText(EncodeText((*Runes)(v.Ptr).Elements))
		return d, err == nil
	}, // PURE by contract
	AsString: func(v Value) (string, bool) { return EncodeText((*Runes)(v.Ptr).Elements), true }, // PURE by contract
	AsRunes:  func(v Value) ([]rune, bool) { return (*Runes)(v.Ptr).Elements, true },             // PURE by contract
	AsBytes:  runesTypeAsBytes,                                                                   // PURE by contract
	AsArray:  runesTypeAsArray,                                                                   // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (keep/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
}

// runesEncodeMatchArg: acceptance on a symbol receiver — text content as
// SYMBOLS. rune/ASCII byte/in-range int are the element class; string/runes/
// valid-UTF-8 bytes are the run class.
func runesEncodeMatchArg(name string, a Value) ([]rune, bool, error) {
	switch a.Type {
	case value.Rune:
		return []rune{rune(a.Data)}, true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("(%s) an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", name, a.Data))
		}
		return []rune{rune(a.Data)}, true, nil
	case value.Int:
		r, ok := a.AsRune()
		if !ok {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("(%s) an int reads as one symbol and must be a valid code point, got %d", name, int64(a.Data)))
		}
		return []rune{r}, true, nil
	case value.String, value.Runes:
		rs, _ := a.AsRunes()
		return rs, false, nil
	case value.Bytes:
		// TOTAL: an undecodable octet decodes to its escape rather than raising
		return DecodeOctets((*Bytes)(a.Ptr).Elements), false, nil
	}
	return nil, false, errs.NewInvalidArgumentTypeError(name, "argument", "text content (symbols, octets, or text)", a.TypeName())
}

// PURE by contract
func runesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	// same boundary as string's: JSON text is UTF-8, so an escape has no representation here
	if !RunesAreValid(o.Elements) {
		return nil, errs.NewConversionError(v.TypeName(), "json", "the text holds octets that are not symbols — encode it as bytes, or repair it (is_valid() finds them)")
	}
	var b []byte
	b = EncodeString(b, EncodeText(o.Elements))
	return b, nil
}

// PURE by contract
func runesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	s := EncodeText(o.Elements)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("runes: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func runesTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("runes: %w", err)
	}
	*v = NewRunesValue(DecodeText(s), v.Immutable)
	return nil
}

// PURE by contract
func runesTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return "u" + strconv.Quote(EncodeText((*Runes)(v.Ptr).Elements)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	o := (*Runes)(v.Ptr)
	return format.FormatStringLike("runes", sp, EncodeText(o.Elements), false)
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// runes value with the items appended (append()) — never touches the receiver's backing storage, works
// regardless of the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func runesTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Runes)(v.Ptr)
	name := "append"
	if mutate {
		name = "append_in_place"
	}
	items, err := runesAddItems(name, args)
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
	res := make([]rune, 0, len(o.Elements)+len(items))
	res = append(res, o.Elements...)
	res = append(res, items...)
	return NewRunesValue(res, false), nil
}

// PURE by contract. deep is irrelevant here: elements are raw runes, not nested Values, so there's nothing a
// shallow copy could leave shared. Kept for signature parity with the shared Copy hook.
func runesTypeCopy(v Value, _ bool) (Value, error) {
	o := (*Runes)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	copy(rs, o.Elements)
	return NewRunesValue(rs, false), nil
}

func runesTypeEqual(v Value, other Value, final bool) bool {
	o := (*Runes)(v.Ptr)
	switch other.Type {
	case value.Runes:
		t := (*Runes)(other.Ptr).Elements
		return slices.Equal(o.Elements, t)
	case value.String, value.Bool, value.Byte, value.Rune, value.Int, value.Decimal, value.Float:
		t, ok := other.AsString()                // identity for String, canonical text form for the rest
		return ok && EncodeText(o.Elements) == t // no text form (a high octet) equals no text
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func runesTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.String:
			switch op {
			case token.Less:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l < r), nil
			case token.LessEq:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l <= r), nil
			case token.Greater:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l > r), nil
			case token.GreaterEq:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l >= r), nil
			}

		case value.Rune:
			switch op {
			case token.Add:
				l := []rune{rune(other.Data)}
				r := (*Runes)(v.Ptr).Elements
				t := make([]rune, len(l)+len(r))
				copy(t, l)
				copy(t[len(l):], r)
				return NewRunesValue(t, false), nil
			}

		case value.Byte:
			// a scalar on the left takes the sequence's type; an octet is a symbol only in ASCII
			switch op {
			case token.Add:
				if other.Data > 0x7F {
					return Undefined, errs.NewInvalidValueError(fmt.Sprintf("an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", other.Data))
				}
				return NewRunesValue(append([]rune{rune(other.Data)}, (*Runes)(v.Ptr).Elements...), false), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Runes:
		switch op {
		case token.Less:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l < r), nil
		case token.LessEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l <= r), nil
		case token.Greater:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l > r), nil
		case token.GreaterEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l >= r), nil
		}

	case value.String:
		switch op {
		case token.Less:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l < r), nil
		case token.LessEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l <= r), nil
		case token.Greater:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l > r), nil
		case token.GreaterEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l >= r), nil
		}

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
			src := (*Runes)(v.Ptr).Elements
			sl := len(src)
			total, terr := SeqRepeatTotal(op.String(), n, sl)
			if terr != nil {
				return Undefined, terr
			}
			t := make([]rune, total)
			// step by the receiver's length, never by the count (see the member form)
			for i := 0; i < total; i += sl {
				copy(t[i:], src)
			}
			return NewRunesValue(t, false), nil
		}
	}

	// + and - take text content, and the RECEIVER — the left operand — decides the result type; acceptance
	// mirrors the member layer minus int, whose operator reading stays arithmetic. `-` removes every
	// occurrence of the run, leftmost non-overlapping; the empty run removes nothing
	if op == token.Add || op == token.Sub {
		s, ok, err := textOperandString(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			l := EncodeText((*Runes)(v.Ptr).Elements)
			if op == token.Add {
				return NewRunesValue(DecodeText(l+s), false), nil
			}
			if s == "" {
				return NewRunesValue(DecodeText(l), false), nil
			}
			return NewRunesValue(DecodeText(strings.ReplaceAll(l, s, "")), false), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func runesTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	if mode != bc.AccessIndex {
		return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
	}
	elems := (*Runes)(v.Ptr).Elements
	i, err := resolveIndex("index access", index, len(elems))
	if err != nil {
		return Undefined, err
	}
	return RuneValue(elems[i]), nil
}

// IMPURE by contract: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func runesTypeAssign(v Value, index Value, r Value, _ bc.Opcode) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	elems := (*Runes)(v.Ptr).Elements
	i, err := resolveIndex("index assign", index, len(elems))
	if err != nil {
		return err
	}
	c, ok := r.AsRune()
	if !ok {
		return errs.NewInvalidIndexTypeError("index assign value", runeTypeName, r.TypeName())
	}
	elems[i] = c
	return nil
}

// PURE by contract. a[i:j] always answers an independently-owned copy; sharing is slice_view's job.
func runesTypeSlice(v Value, s Value, e Value) (Value, error) {
	elems := (*Runes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds("slice", s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]rune, ei-si)
	copy(out, elems[si:ei])
	return NewRunesValue(out, false), nil
}

// PURE by contract
func runesTypeSliceStep(v Value, s Value, e Value, step Value) (Value, error) {
	elems := (*Runes)(v.Ptr).Elements
	start, end, st, err := resolveSliceStep(s, e, step, len(elems))
	if err != nil {
		return Undefined, err
	}
	out := make([]rune, 0)
	if st > 0 {
		for i := start; i < end; i += st {
			out = append(out, elems[i])
		}
	} else {
		for i := start; i > end; i += st {
			out = append(out, elems[i])
		}
	}
	return NewRunesValue(out, false), nil
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func runesTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return runesTypeCopy(v, true)

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

	case "runes":
		// the same-type conversion constructs — a new, independent, mutable copy, exactly
		// runes(r) / r.copy(); see the note on array's own case.
		c, err := runesTypeCopy(v, false)
		if err != nil {
			return Undefined, err
		}
		return convMember(name, runesTypeName, args, true, c)

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(o.Elements)), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "bool":
		b, ok := runesTypeAsBool(v)
		return convMember(name, runesTypeName, args, ok, BoolValue(b))

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewBytesValue(EncodeOctets(o.Elements), false), nil

	case "float":
		f, ok := runesTypeAsFloat(v)
		return convMember(name, runesTypeName, args, ok, FloatValue(f))

	case "int":
		i, ok := runesTypeAsInt(v)
		return convMember(name, runesTypeName, args, ok, IntValue(i))

	case "decimal":
		d, ok := runesTypeAsDecimal(v)
		return convMember(name, runesTypeName, args, ok, NewDecimalValue(d))

	case "time":
		return textTimeMember(name, runesTypeName, EncodeText((*Runes)(v.Ptr).Elements), args)

	case "date":
		return textDateMember(name, runesTypeName, EncodeText((*Runes)(v.Ptr).Elements), args)

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
			return Undefined, errs.FromFormatSpecError(name, err)
		}
		s, err := runesTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_valid":
		// no escapes anywhere: every element is a real symbol
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(RunesAreValid(o.Elements)), nil

	case "is_ascii":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(RunesAreASCII(o.Elements)), nil

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
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return RuneValue(o.Elements[0]), nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return RuneValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return RuneValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return RuneValue(slices.Max(o.Elements)), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToLower(r)
		}
		return NewRunesValue(rs, false), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToUpper(r)
		}
		return NewRunesValue(rs, false), nil

	case "contains":
		return runesContainsMember(vm, v, args)

	case "count":
		return runesCount(vm, v, args)

	case "keep":
		return runesKeep(vm, v, args)

	case "keep_in_place":
		return runesKeepInPlace(vm, v, args)

	case "remove":
		return runesRemove(vm, v, args)

	case "remove_in_place":
		return runesRemoveInPlace(vm, v, args)

	case "any":
		return runesAny(vm, v, args)

	case "all":
		return runesAll(vm, v, args)

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := make([]rune, len(o.Elements))
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return NewRunesValue(sorted, false), nil

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
		return runesDedup(vm, v, args)

	case "dedup_in_place":
		return runesDedupInPlace(vm, v, args)

	case "unique":
		return runesUnique(vm, v, args)

	case "unique_in_place":
		return runesUniqueInPlace(vm, v, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := make([]rune, n)
		for i, r := range o.Elements {
			rev[n-1-i] = r
		}
		return NewRunesValue(rev, false), nil

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
		return runesForEach(vm, v, args)

	case "index":
		return runesIndex(vm, v, args)

	case "index_last":
		return runesIndexLast(vm, v, args)

	case "chunk":
		return runesChunk(vm, v, args)

	case "chunk_view":
		return runesChunkView(vm, v, args)

	case "slice":
		return sliceMember(v, args)

	case "slice_view":
		return runesSliceView(vm, v, args)

	case "append":
		return runesTypeAppend(v, args, false)

	case "append_in_place":
		return runesTypeAppend(v, args, true)

	case "prepend":
		return runesPrepend(vm, v, args)

	case "prepend_in_place":
		return runesPrependInPlace(vm, v, args)

	case "push":
		return runesPush(vm, v, args)

	case "push_in_place":
		return runesPushInPlace(vm, v, args)

	case "push_first":
		return runesPushFirst(vm, v, args)

	case "push_first_in_place":
		return runesPushFirstInPlace(vm, v, args)

	case "insert":
		return runesInsert(vm, v, args)

	case "insert_in_place":
		return runesInsertInPlace(vm, v, args)

	case "splice":
		return runesSplice(vm, v, args)

	case "splice_in_place":
		return runesSpliceInPlace(vm, v, args)

	case "map":
		return runesMap(vm, v, args)

	case "flat_map":
		return runesFlatMap(vm, v, args)

	case "case_fold":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			out[i] = foldRuneCanonical(r)
		}
		return NewRunesValue(out, false), nil

	case "title_case":
		// the label rendering segments on WRITTEN boundaries only (case transitions stay inside words);
		// the identifier renderings re-segment fully
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(caseJoinTitle(caseSegmentWritten(o.Elements)), false), nil

	case "snake_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(caseJoinLower(caseSegmentWords(o.Elements), '_'), false), nil

	case "kebab_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(caseJoinLower(caseSegmentWords(o.Elements), '-'), false), nil

	case "camel_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(caseJoinCapitalized(caseSegmentWords(o.Elements), true), false), nil

	case "pascal_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(caseJoinCapitalized(caseSegmentWords(o.Elements), false), false), nil

	case "reduce":
		return runesReduce(vm, v, args)

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
		out := make([]rune, total)
		// step by the receiver's length, never by the count: an empty receiver has total 0 and must not
		// spin n times copying nothing
		for i := 0; i < total; i += sl {
			copy(out[i:], src)
		}
		return NewRunesValue(out, false), nil

	case "split":
		return runesSplit(vm, v, args)

	case "split_lines":
		return runesFnSplitLines(v, args)

	case "partition":
		return runesPartition(vm, v, args)

	case "trim":
		return runesTrim(vm, v, args)

	case "trim_in_place":
		return runesTrimInPlace(vm, v, args)

	case "trim_start":
		return runesTrimStart(vm, v, args)

	case "trim_start_in_place":
		return runesTrimStartInPlace(vm, v, args)

	case "trim_end":
		return runesTrimEnd(vm, v, args)

	case "trim_end_in_place":
		return runesTrimEndInPlace(vm, v, args)

	case "has_prefix":
		return runesHasPrefix(vm, v, args)

	case "has_suffix":
		return runesHasSuffix(vm, v, args)

	case "remove_prefix":
		return runesRemovePrefix(vm, v, args)

	case "remove_prefix_in_place":
		return runesRemovePrefixInPlace(vm, v, args)

	case "remove_suffix":
		return runesRemoveSuffix(vm, v, args)

	case "remove_suffix_in_place":
		return runesRemoveSuffixInPlace(vm, v, args)

	case "replace":
		return runesReplace(vm, v, args)

	case "replace_in_place":
		return runesReplaceInPlace(vm, v, args)

	case "pad_start":
		return runesPadStart(vm, v, args)

	case "pad_start_in_place":
		return runesPadStartInPlace(vm, v, args)

	case "pad_end":
		return runesPadEnd(vm, v, args)

	case "pad_end_in_place":
		return runesPadEndInPlace(vm, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func runesTypeIterator(v Value) (Value, error) {
	return NewRunesIteratorValue((*Runes)(v.Ptr).Elements), nil
}

// PURE by contract
func runesTypeAsInt(v Value) (int64, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(EncodeText(o.Elements), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

// PURE by contract
func runesTypeAsFloat(v Value) (float64, bool) {
	o := (*Runes)(v.Ptr)
	f, err := strconv.ParseFloat(EncodeText(o.Elements), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

// PURE by contract
func runesTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*Runes)(v.Ptr)
	d := dec128.FromString(EncodeText(o.Elements))
	return d, !d.IsNaN()
}

// PURE by contract
func runesTypeAsBool(v Value) (bool, bool) {
	o := (*Runes)(v.Ptr)
	return conv.ParseBool(EncodeText(o.Elements))
}

// PURE by contract
func runesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Runes)(v.Ptr)
	return EncodeOctets(o.Elements), true
}

// PURE by contract
func runesTypeAsTime(v Value) (time.Time, bool) {
	return parseTimeText(EncodeText((*Runes)(v.Ptr).Elements))
}

// PURE by contract
func runesTypeAsArray(v Value) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

// PURE by contract
// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all.
//
// On a text receiver every accepted argument is TEXT CONTENT (runesEncodeMatchArg), matched as a run of symbols —
// a single symbol is just a run of one:
//   - no argument      the significant symbols (not IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a run; several arguments form a set of runs
//
// any/all ask about single symbols, so they take symbols, a predicate, or nothing — never a run. A set may not mix
// symbol-typed arguments (rune/byte/int) with text-typed ones (string/runes/bytes), and a function among several
// arguments raises. Runs match leftmost-longest and never overlap.
// ---------------------------------------------------------------------------

// runesMatch is a match member's argument list after reading; exactly one reading is set.
type runesMatch struct {
	significant bool     // no argument
	pred        Value    // a predicate
	elems       []rune   // any/all: a set of symbols
	runs        [][]rune // contains/count/keep/remove: a set of runs
}

// runesReadMatchArgs reads a match member's arguments. symbols is true for any/all, which read each argument as
// exactly one symbol instead of a run.
func runesReadMatchArgs(name string, args []Value, symbols bool) (runesMatch, error) {
	if len(args) == 0 {
		return runesMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return runesMatch{}, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return runesMatch{}, err
		}
		return runesMatch{pred: args[0]}, nil
	}

	runs := make([][]rune, 0, len(args))
	sawSymbol, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return runesMatch{}, errFunctionInSet(name)
		}
		run, isSymbol, err := runesEncodeMatchArg(name, a)
		if err != nil {
			return runesMatch{}, err
		}
		if isSymbol {
			sawSymbol = true
		} else {
			sawText = true
		}
		runs = append(runs, run)
	}
	if sawSymbol && sawText {
		return runesMatch{}, errMixedSet(name)
	}

	if !symbols {
		return runesMatch{runs: runs}, nil
	}
	if sawText {
		return runesMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "a value, a function, or nothing (the contiguous-run query is contains's)", "sequence")
	}
	elems := make([]rune, 0, len(runs))
	for _, r := range runs {
		if len(r) != 1 {
			return runesMatch{}, errNotOneElement(name)
		}
		elems = append(elems, r[0])
	}
	return runesMatch{elems: elems}, nil
}

// matches reports whether symbol r at index i matches an element-wise reading (anything but runs).
func (m *runesMatch) matches(vm VM, i int, r rune) (bool, error) {
	switch {
	case m.significant:
		return !IsBlankRune(r), nil
	case m.pred.IsCallable():
		res, err := callElem(vm, m.pred, i, RuneValue(r))
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	return slices.Contains(m.elems, r), nil
}

// runesContainsMember is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
func runesContainsMember(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements

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

// runesCount is count(...): how many matches. Runs count non-overlapping occurrences.
func runesCount(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements

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

// runesKept answers the symbols keep(...) keeps: the matches, in order. Shared by keep and keep_in_place; name is
// the member called, for the errors.
func runesKept(vm VM, name string, v Value, args []Value) ([]rune, error) {
	m, err := runesReadMatchArgs(name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems))

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

// runesKeep is keep(...): new runes of the matches.
func runesKeep(vm VM, v Value, args []Value) (Value, error) {
	out, err := runesKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesKeepInPlace is keep_in_place(...): keep the matches in the receiver itself.
func runesKeepInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesRemaining answers the symbols remove(...) leaves: everything but the matches, in order. With no argument
// remove drops the BLANK symbols — so it keeps the significant ones, landing on keep()'s answer by the opposite
// action. Shared by remove and remove_in_place; name is the member called, for the errors.
func runesRemaining(vm VM, name string, v Value, args []Value) ([]rune, error) {
	m, err := runesReadMatchArgs(name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems))

	if m.significant {
		for _, r := range elems {
			if !IsBlankRune(r) {
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

// runesRemove is remove(...): new runes without the matches.
func runesRemove(vm VM, v Value, args []Value) (Value, error) {
	out, err := runesRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemoveInPlace is remove_in_place(...): drop the matches from the receiver
// itself.
func runesRemoveInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesAny is any(...): does some symbol match? Symbols, a predicate, or nothing — never a run.
func runesAny(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Runes)(v.Ptr).Elements {
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

// runesAll is all(...): does every symbol match? Symbols, a predicate, or nothing — never a run. True on empty
// runes.
func runesAll(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range (*Runes)(v.Ptr).Elements {
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
// counted in symbols. The argument's type selects the reading:
//   - no argument      the first / last significant symbol (not IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one symbol, compared with ==; a value that is not one symbol raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// runesIndex is index(...): the first match.
func runesIndex(vm VM, v Value, args []Value) (Value, error) {
	return runesLocate(vm, "index", v, args, false)
}

// runesIndexLast is index_last(...): the last match.
func runesIndexLast(vm VM, v Value, args []Value) (Value, error) {
	return runesLocate(vm, "index_last", v, args, true)
}

// runesLocate is the body of index and index_last; name is the member called, for the errors.
func runesLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	elems := (*Runes)(v.Ptr).Elements

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !IsBlankRune(e) {
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
			res, err := callElem(vm, needle, i, RuneValue(e))
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
		run, ok := needle.AsRunes()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "text content", needle.TypeName())
		}
		return locatorAnswer(indexRun(elems, run, last), dflt)
	}

	enc, isElement, err := runesEncodeMatchArg(name, needle)
	if err != nil {
		return Undefined, err
	}
	if !isElement || len(enc) != 1 {
		return Undefined, errNotOneElement(name)
	}
	idx := -1
	for i, e := range elems {
		if RuneValue(e).Equal(needle) {
			idx = i
			if !last {
				break
			}
		}
	}
	return locatorAnswer(idx, dflt)
}

// ---------------------------------------------------------------------------
// The structural members: the trim family, the anchored pair (has_/remove_ prefix/suffix), replace, and the pads.
// Every argument is text content (runesEncodeMatchArg); none of them takes a predicate.
//   - trim*            a set of single symbols, stripped repeat-while; no argument = the blank set (IsBlankRune)
//   - *_prefix/suffix  a homogeneous set of runs; the longest anchored run wins; remove_* removes it once
//   - replace(old, new) every occurrence of the run old, leftmost and non-overlapping; an empty old matches nothing
//   - pad_*(n[, fill]) n counts symbols; the fill is exactly one symbol, the space by default
// ---------------------------------------------------------------------------

// runesReadElementSet reads a set of single symbols (the trim family's set, the pads' fill): each argument must be
// one symbol. A text-typed argument or a function raises with refuse, the member's own statement of what it
// takes.
func runesReadElementSet(name string, args []Value, refuse string) ([]rune, error) {
	set := make([]rune, 0, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", refuse, a.TypeName())
		}
		enc, isElement, err := runesEncodeMatchArg(name, a)
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

// runesReadRunSet reads a homogeneous set of runs (the anchored pair, split, partition): an element-typed argument
// is a run of its encoding, but one call may not mix element-typed and text-typed arguments. A function raises
// with onFunction, the member's own statement of what it takes.
func runesReadRunSet(name string, args []Value, onFunction string) ([][]rune, error) {
	runs := make([][]rune, 0, len(args))
	sawElement, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return nil, errs.NewInvalidArgumentTypeError(name, "argument", onFunction, a.TypeName())
		}
		run, isElement, err := runesEncodeMatchArg(name, a)
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

// runesTrimmed answers the receiver without its leading (start) and/or trailing (end) symbols that belong to the
// set. The name is the member called, for the errors.
func runesTrimmed(name string, v Value, args []Value, start, end bool) ([]rune, error) {
	set, err := runesReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e rune) bool {
		if len(args) == 0 {
			return IsBlankRune(e)
		}
		return slices.Contains(set, e)
	}
	elems := (*Runes)(v.Ptr).Elements
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

// runesAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func runesAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := runesReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := (*Runes)(v.Ptr).Elements
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

// runesWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func runesWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]rune, error) {
	best, err := runesAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := (*Runes)(v.Ptr).Elements
	switch {
	case best <= 0:
		return slices.Clone(elems), nil
	case suffix:
		return slices.Clone(elems[:len(elems)-best]), nil
	default:
		return slices.Clone(elems[best:]), nil
	}
}

// runesReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); each
// argument is read on its own, an element being a run of one. The name is the member called, for the errors.
func runesReplaced(name string, v Value, args []Value) ([]rune, error) {
	if len(args) != 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "2", len(args))
	}
	if args[0].IsCallable() || args[1].IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "argument", "an element or a run (replace is never a predicate)", "function")
	}
	old, _, err := runesEncodeMatchArg(name, args[0])
	if err != nil {
		return nil, err
	}
	repl, _, err := runesEncodeMatchArg(name, args[1])
	if err != nil {
		return nil, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems))
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

// runesPadded answers the receiver filled at the front (start) or the end up to n symbols; a width at or below the
// length leaves it unchanged. The name is the member called, for the errors.
func runesPadded(name string, v Value, args []Value, start bool) ([]rune, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := ' '
	if len(args) == 2 {
		set, err := runesReadElementSet(name, args[1:], "one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return nil, err
		}
		fill = set[0]
	}
	elems := (*Runes)(v.Ptr).Elements
	if n <= int64(len(elems)) {
		return slices.Clone(elems), nil
	}
	width, err := SeqPadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]rune{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// runesTrim is trim(...): without the leading and trailing elements of the set.
func runesTrim(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimInPlace is trim_in_place(...): trim(...) applied to the receiver
// itself.
func runesTrimInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesTrimStart is trim_start(...): without the leading elements of the set.
func runesTrimStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimStartInPlace is trim_start_in_place(...): trim_start(...) applied to the receiver
// itself.
func runesTrimStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesTrimEnd is trim_end(...): without the trailing elements of the set.
func runesTrimEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesTrimEndInPlace is trim_end_in_place(...): trim_end(...) applied to the receiver
// itself.
func runesTrimEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "trim_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesTrimmed(name, v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
func runesRemovePrefix(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemovePrefixInPlace is remove_prefix_in_place(...): remove_prefix(...) applied to the receiver
// itself.
func runesRemovePrefixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_prefix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesWithoutAnchored(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
func runesRemoveSuffix(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesRemoveSuffixInPlace is remove_suffix_in_place(...): remove_suffix(...) applied to the receiver
// itself.
func runesRemoveSuffixInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "remove_suffix_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesWithoutAnchored(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesReplace is replace(...): every occurrence of old replaced by new.
func runesReplace(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesReplaceInPlace is replace_in_place(...): replace(...) applied to the receiver
// itself.
func runesReplaceInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "replace_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesReplaced(name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesPadStart is pad_start(...): filled at the front up to n elements.
func runesPadStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPadStartInPlace is pad_start_in_place(...): pad_start(...) applied to the receiver
// itself.
func runesPadStartInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_start_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesPadded(name, v, args, true)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesPadEnd is pad_end(...): filled at the end up to n elements.
func runesPadEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := runesPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPadEndInPlace is pad_end_in_place(...): pad_end(...) applied to the receiver
// itself.
func runesPadEndInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "pad_end_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := runesPadded(name, v, args, false)
	if err != nil {
		return Undefined, err
	}
	(*Runes)(v.Ptr).Set(out)
	return v, nil
}

// runesHasPrefix is has_prefix(...): does the receiver start with one of the runs?
func runesHasPrefix(_ VM, v Value, args []Value) (Value, error) {
	best, err := runesAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// runesHasSuffix is has_suffix(...): does the receiver end with one of the runs?
func runesHasSuffix(_ VM, v Value, args []Value) (Value, error) {
	best, err := runesAnchoredRun("has_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// ---------------------------------------------------------------------------
// split(...seps) and partition(...seps): a separator is a run (an element being a run of one), a homogeneous set
// of runs, a predicate on single symbols, or — no argument — the blank set (IsBlankRune). Runs match leftmost-longest
// and never overlap; an empty run matches nothing.
// ---------------------------------------------------------------------------

// runesSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant symbols, the
// classic whitespace split.
func runesSplit(vm VM, v Value, args []Value) (Value, error) {
	const name = "split"
	elems := (*Runes)(v.Ptr).Elements

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankRune(e) {
				if start >= 0 {
					pieces = append(pieces, runesPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, runesPiece(elems[start:]))
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
			res, err := callElem(vm, args[0], i, RuneValue(e))
			if err != nil {
				return Undefined, err
			}
			hit, err := res.IsTrue()
			if err != nil {
				return Undefined, err
			}
			if hit {
				pieces = append(pieces, runesPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, runesPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := runesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, runesPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, runesPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// runesPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
func runesPartition(vm VM, v Value, args []Value) (Value, error) {
	const name = "partition"
	elems := (*Runes)(v.Ptr).Elements
	found, n := -1, 0

	switch {
	case len(args) == 0:
		for i, e := range elems {
			if IsBlankRune(e) {
				found, n = i, 1
				for found+n < len(elems) && IsBlankRune(elems[found+n]) {
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
			res, err := callElem(vm, args[0], i, RuneValue(e))
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
		runs, err := runesReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
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
		return NewArrayValue([]Value{runesPiece(elems), runesPiece(nil), runesPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{runesPiece(elems[:found]), runesPiece(elems[found : found+n]), runesPiece(elems[found+n:])}, false), nil
}

// runesPiece is one piece of split/partition: new runes, independent of the receiver.
func runesPiece(elems []rune) Value {
	return NewRunesValue(slices.Clone(elems), false)
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(symbol) or
// f/2(index, symbol); reduce's is f/2(acc, symbol) or f/3(acc, index, symbol).
// ---------------------------------------------------------------------------

// runesForEach is for_each(f): a full pass whose callback result is ignored — early exit belongs to for/break or a
// search member. Returns the receiver, so it chains.
func runesForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Runes)(v.Ptr).Elements {
		if _, err := callElem(vm, fn, i, RuneValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// runesMap is map(f): strictly 1:1, answering runes — each callback result must be exactly one symbol (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
func runesMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			return Undefined, errs.NewInvalidValueError("(" + name + ") the callback answered undefined — map is 1:1; the dropping form is flat_map")
		}
		enc, isElement, err := runesEncodeMatchArg(name, res)
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
	return NewRunesValue(out, false), nil
}

// runesFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
func runesFlatMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems))
	for i, e := range elems {
		res, err := callElem(vm, fn, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
		if res.Type == value.Undefined {
			continue
		}
		enc, _, err := runesEncodeMatchArg(name, res)
		if err != nil {
			return Undefined, err
		}
		out = append(out, enc...)
	}
	return NewRunesValue(out, false), nil
}

// runesReduce is reduce(acc, f): folds the symbols left to right.
func runesReduce(vm VM, v Value, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range (*Runes)(v.Ptr).Elements {
		acc, err = callReduce(vm, fn, acc, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// ---------------------------------------------------------------------------
// The add and edit members: chunk, slice_view, splice, insert, prepend, push, push_first (append is the Append
// hook, runesTypeAppend; slice is the Slice hook's member spelling). The add side reads text content:
// append/prepend/splice's inserts are runs (an element being a run of one), push/push_first/insert take exactly
// one symbol per argument.
// ---------------------------------------------------------------------------

// runesAddItems reads the add side's operands (append, prepend, splice's inserts): every argument is text content,
// concatenated in argument order — mixing elements and runs is fine here (x.append("ab", 'c') is x + "ab" + 'c').
func runesAddItems(name string, args []Value) ([]rune, error) {
	items := make([]rune, 0, len(args))
	for _, a := range args {
		enc, _, err := runesEncodeMatchArg(name, a)
		if err != nil {
			return nil, err
		}
		items = append(items, enc...)
	}
	return items, nil
}

// runesPushItems reads push/push_first/insert's elements: each argument must be exactly one symbol — a text
// argument raises even at length 1, which is the member's purpose (append/prepend take runs).
func runesPushItems(name string, args []Value) ([]rune, error) {
	return runesReadElementSet(name, args, "one element (a sequence argument never reads as an element here; append/prepend take runs)")
}

// runesChunk is chunk(size): the symbols in pieces of size (the last one shorter), each an independent copy.
func runesChunk(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewRunesValue(slices.Clone(elems[start:min(start+step, len(elems))]), false))
	}
	return NewArrayValue(chunks, false), nil
}

// runesChunkView is chunk_view(size): chunk's pieces sharing the receiver's storage, each marked as a view.
func runesChunkView(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk_view", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		c := NewRunesValue(elems[start:min(start+step, len(elems))], v.Immutable)
		(*Runes)(c.Ptr).IsView = true
		chunks = append(chunks, c)
	}
	return NewArrayValue(chunks, false), nil
}

// runesSliceView is slice_view([start[, end]]): the span sharing the receiver's storage, marked as a view.
func runesSliceView(_ VM, v Value, args []Value) (Value, error) {
	const name = "slice_view"
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	si, ei, err := resolveSliceBounds(name, s, e, len(elems))
	if err != nil {
		return Undefined, err
	}
	res := NewRunesValue(elems[si:ei], v.Immutable)
	(*Runes)(res.Ptr).IsView = true
	return res, nil
}

// runesSplice is splice([start[, count[, ...items]]]): a new runes with count symbols at start replaced by
// the items (read like append's operands).
func runesSplice(_ VM, v Value, args []Value) (Value, error) {
	elems := (*Runes)(v.Ptr).Elements
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = runesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]rune, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesSpliceInPlace is splice_in_place(...): splice applied to the receiver itself.
// A side-effecting member returns the RECEIVER, so mutators chain; the removed span is x.slice(i, j) taken
// beforehand.
func runesSpliceInPlace(_ VM, v Value, args []Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotMutableError("splice_in_place", v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	start, end, err := readSpliceRange(args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = runesAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	o.Set(append(o.Elements[:start], append(items, o.Elements[end:]...)...))
	return v, nil
}

// runesInsert is insert(i, ...items): a new runes with the items — one symbol each — at position i, which
// raises out of [0, len].
func runesInsert(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert"
	elems := (*Runes)(v.Ptr).Elements
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := runesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	out := slices.Insert(slices.Clone(elems), at, items...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesInsertInPlace is insert_in_place(i, ...items): insert applied to the receiver
// itself.
func runesInsertInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	at, err := readEditPos(name, args, len(o.Elements))
	if err != nil {
		return Undefined, err
	}
	items, err := runesPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	o.Set(slices.Insert(o.Elements, at, items...))
	return v, nil
}

// runesPrepend is prepend(...items): a new runes with the items (read like append's operands) in front, in
// argument order — x.prepend(a, b) is a + b + x.
func runesPrepend(_ VM, v Value, args []Value) (Value, error) {
	items, err := runesAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPrependInPlace is prepend_in_place(...items): prepend applied to the receiver
// itself.
func runesPrependInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "prepend_in_place"
	items, err := runesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// runesPush is push(...items): a new runes with the items — exactly one symbol each — at the end.
func runesPush(_ VM, v Value, args []Value) (Value, error) {
	items, err := runesPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(elems)+len(items))
	out = append(out, elems...)
	out = append(out, items...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPushInPlace is push_in_place(...items): push applied to the receiver itself.
func runesPushInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "push_in_place"
	items, err := runesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(append(o.Elements, items...))
	return v, nil
}

// runesPushFirst is push_first(...items): a new runes with the items — exactly one symbol each — in front,
// in argument order.
func runesPushFirst(_ VM, v Value, args []Value) (Value, error) {
	items, err := runesPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	elems := (*Runes)(v.Ptr).Elements
	out := make([]rune, 0, len(items)+len(elems))
	out = append(out, items...)
	out = append(out, elems...)
	return NewRunesValue(out, false), nil
}

// IMPURE: mutates the receiver. runesPushFirstInPlace is push_first_in_place(...items): push_first applied to the
// receiver itself.
func runesPushFirstInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "push_first_in_place"
	items, err := runesPushItems(name, args)
	if err != nil {
		return Undefined, err
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(slices.Insert(o.Elements, 0, items...))
	return v, nil
}

// runesDeduped answers the elements with each run of equal neighbours collapsed to one.
func runesDeduped(elems []rune) []rune {
	out := make([]rune, 0, len(elems))
	for i, e := range elems {
		if i == 0 || e != elems[i-1] {
			out = append(out, e)
		}
	}
	return out
}

// runesDedup is dedup(): a new runes with each run of equal neighbours collapsed to one.
func runesDedup(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
	}
	return NewRunesValue(runesDeduped((*Runes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. runesDedupInPlace is dedup_in_place(): dedup applied to the receiver itself.
func runesDedupInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "dedup_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(runesDeduped(o.Elements))
	return v, nil
}

// runesUniqueElements answers the elements without repeats, each kept at its first occurrence.
func runesUniqueElements(elems []rune) []rune {
	out := make([]rune, 0, len(elems))
	seen := make(map[rune]struct{}, len(elems))
	for _, e := range elems {
		if _, ok := seen[e]; !ok {
			seen[e] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}

// runesUnique is unique(): a new runes without repeats, each element kept at its first occurrence.
func runesUnique(_ VM, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}
	return NewRunesValue(runesUniqueElements((*Runes)(v.Ptr).Elements), false), nil
}

// IMPURE: mutates the receiver. runesUniqueInPlace is unique_in_place(): unique applied to the receiver itself.
func runesUniqueInPlace(_ VM, v Value, args []Value) (Value, error) {
	const name = "unique_in_place"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	o := (*Runes)(v.Ptr)
	o.Set(runesUniqueElements(o.Elements))
	return v, nil
}

// runesTypeContains is the `in` operator: every accepted operand is text content encoded into the
// receiver's representation and matched as a run (the member's own acceptance); a callable raises.
func runesTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	run, _, err := runesEncodeMatchArg("in", e)
	if err != nil {
		return false, err
	}
	return strings.Contains(EncodeText((*Runes)(v.Ptr).Elements), string(run)), nil
}

// PURE by contract
func runesFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	pieces := splitLinesString(EncodeText(o.Elements))
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue(DecodeText(p), false)
	}
	return NewArrayValue(arr, false), nil
}
