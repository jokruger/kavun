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
	"unicode/utf8"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const stringTypeName = "string"

func NewStaticStringValue(s *string) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(TextRuneCount(*s)), Ptr: unsafe.Pointer(s)}
}

// NewStaticStringValueCounted is NewStaticStringValue with a precomputed rune count
func NewStaticStringValueCounted(s *string, runeLen int64) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(runeLen), Ptr: unsafe.Pointer(s)}
}

func NewStringValue(s string) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(TextRuneCount(s)), Ptr: unsafe.Pointer(&s)}
}

// newStringValueCounted skips the rune recount where the caller already knows it (substring/slice paths).
func newStringValueCounted(s string, runeLen int64) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(runeLen), Ptr: unsafe.Pointer(&s)}
}

// stringIsASCII reports the fast path: every octet is ASCII, so byte offsets are symbol offsets and
// indexing/slicing stay O(1). It deliberately does NOT compare the rune count to the byte count — an
// undecodable octet also decodes to one symbol from one octet, so that test passes for text the fast path
// would then read wrongly (it would answer rune(0xFF) where every other operation answers the escape).
func stringIsASCII(_ Value, s string) bool {
	return IsASCIIText(s)
}

// TypeString is a string type descriptor.
var TypeString = ValueTypeDescr{
	Name:                ConstHook(stringTypeName),                                                                             // PURE by contract
	String:              func(v Value) string { return strconv.Quote(*(*string)(v.Ptr)) },                                      // PURE by contract
	Format:              stringTypeFormat,                                                                                      // PURE by contract
	Interface:           func(v Value) any { return *(*string)(v.Ptr) },                                                        // PURE by contract
	EncodeJSON:          stringTypeEncodeJSON,                                                                                  // PURE by contract
	EncodeBinary:        stringTypeEncodeBinary,                                                                                // PURE by contract
	DecodeBinary:        stringTypeDecodeBinary,                                                                                // IMPURE by contract (mutates target)
	IsTrue:              func(v Value) (bool, error) { return len(*(*string)(v.Ptr)) > 0, nil },                                // PURE by contract
	IsIterable:          ConstHook(true),                                                                                       // PURE by contract
	Iterator:            stringTypeIterator,                                                                                    // PURE by contract (constructs fresh iterator)
	Len:                 func(v Value) int64 { return int64(v.Data) },                                                          // PURE by contract — symbols, not bytes; the count is cached at construction
	Equal:               stringTypeEqual,                                                                                       // PURE by contract
	BinaryOp:            stringTypeBinaryOp,                                                                                    // PURE by contract
	CallNamedMethod:     stringTypeCallNamedMethod,                                                                             // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessIndex:         stringTypeAccessIndex,                                                                                 // PURE by contract
	AccessNamedProperty: noNamedProperty,                                                                                       // PURE by contract
	Contains:            stringTypeContains,                                                                                    // PURE by contract
	Slice:               stringTypeSlice,                                                                                       // PURE by contract
	SliceStep:           stringTypeSliceStep,                                                                                   // PURE by contract
	AsBool:              func(v Value) (bool, bool) { return conv.ParseBool(*(*string)(v.Ptr)) },                               // PURE by contract
	AsInt:               stringTypeAsInt,                                                                                       // PURE by contract
	AsFloat:             stringTypeAsFloat,                                                                                     // PURE by contract
	AsDecimal:           stringTypeAsDecimal,                                                                                   // PURE by contract
	AsTime:              stringTypeAsTime,                                                                                      // PURE by contract
	AsDate:              func(v Value) (civil.Date, bool) { d, err := ParseDateText(*(*string)(v.Ptr)); return d, err == nil }, // PURE by contract
	AsString:            func(v Value) (string, bool) { return *(*string)(v.Ptr), true },                                       // PURE by contract
	AsRunes:             func(v Value) ([]rune, bool) { return DecodeText(*(*string)(v.Ptr)), true },                           // PURE by contract
	AsBytes:             func(v Value) ([]byte, bool) { return []byte(*(*string)(v.Ptr)), true },                               // PURE by contract
	AsArray:             stringTypeAsArray,                                                                                     // PURE by contract
	IsNamedMethodPure:   func(string) bool { return true },                                                                     // All methods are expected to be pure.
}

// PURE by contract
func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	// JSON text is UTF-8 by definition, so an octet that is not a symbol has nowhere to go here. It
	// raises rather than emitting a byte no parser can read back or silently becoming U+FFFD: the
	// conversions inside the language are total, the boundary OUT of the language is not, and a script
	// asks with is_valid() before it encodes. bytes.json() carries arbitrary octets, as base64
	if !TextIsValid(*o) {
		return nil, errs.NewConversionError(v.TypeName(), "json", "the text holds octets that are not symbols — encode it as bytes, or repair it (is_valid() finds them)")
	}
	var b []byte
	b = EncodeString(b, *o)
	return b, nil
}

// PURE by contract
func stringTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("string: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func stringTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("string: %w", err)
	}
	*v = NewStringValue(s)
	return nil
}

// PURE by contract
func stringTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	o := (*string)(v.Ptr)
	if sp.Verb == 'v' {
		return strconv.Quote(*o), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(stringTypeName, sp, fspec.AlignLeft), nil
	}
	return format.FormatStringLike(stringTypeName, sp, *o, false)
}

func stringTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.String, value.Bool, value.Byte, value.Rune, value.Int, value.Decimal, value.Float:
		t, ok := other.AsString()           // identity for String, canonical text form for the rest
		return ok && *(*string)(v.Ptr) == t // no text form (a high octet) equals no string
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

func stringTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Rune:
			switch op {
			case tokens.Add:
				l := EncodeRuneText(rune(other.Data))
				r := *(*string)(v.Ptr)
				return NewStringValue(l + r), nil
			}
		case value.Byte:
			// a scalar on the left takes the sequence's type; an octet is a symbol only in ASCII
			switch op {
			case tokens.Add:
				if other.Data > 0x7F {
					return Undefined, errs.NewInvalidValueError(fmt.Sprintf("an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", other.Data))
				}
				return NewStringValue(EncodeRuneText(rune(other.Data)) + *(*string)(v.Ptr)), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.String:
		l := *(*string)(v.Ptr)
		r := *(*string)(other.Ptr)
		switch op {
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}
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
			src := *(*string)(v.Ptr)
			if _, terr := SeqRepeatTotal(op.String(), n, len(src)); terr != nil {
				return Undefined, terr
			}
			return NewStringValue(strings.Repeat(src, n)), nil
		}
	}

	// + and - take text content, and the RECEIVER — the left operand — decides the result type; acceptance
	// mirrors the member layer minus int, whose operator reading stays arithmetic. `-` removes every
	// occurrence of the run, leftmost non-overlapping; the empty run removes nothing
	if op == tokens.Add || op == tokens.Sub {
		s, ok, err := textOperandString(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			l := *(*string)(v.Ptr)
			if op == tokens.Add {
				return NewStringValue(l + s), nil
			}
			if s == "" {
				return NewStringValue(l), nil
			}
			return NewStringValue(strings.ReplaceAll(l, s, "")), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func stringTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*string)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value regardless of copy depth
		return v, nil

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable already, so freeze/freeze_shallow are no-ops
		return v, nil

	case "string":
		// immutable and identity-less, so there is nothing to construct: the receiver IS the
		// independent value. It still takes the trailing default like every other conversion
		// cell, so generic x.string(fallback) code works on a string receiver too.
		return convMember(name, stringTypeName, args, true, v)

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewBytesValue([]byte(*o), false), nil

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewRunesValue(DecodeText(*o), false), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "bool":
		b, ok := conv.ParseBool(*(*string)(v.Ptr))
		return convMember(name, stringTypeName, args, ok, BoolValue(b))

	case "float":
		f, ok := stringTypeAsFloat(v)
		return convMember(name, stringTypeName, args, ok, FloatValue(f))

	case "int":
		i, ok := stringTypeAsInt(v)
		return convMember(name, stringTypeName, args, ok, IntValue(i))

	case "decimal":
		d, ok := stringTypeAsDecimal(v)
		return convMember(name, stringTypeName, args, ok, NewDecimalValue(d))

	case "time":
		return textTimeMember(name, stringTypeName, *(*string)(v.Ptr), args)

	case "date":
		return textDateMember(name, stringTypeName, *(*string)(v.Ptr), args)

	case "format":
		return memberFormat(vm, v, members.Format, args)

	case "is_valid":
		// no escapes anywhere: the text is well-formed UTF-8 end to end
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(TextIsValid(*o)), nil

	case "is_ascii":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(IsASCIIText(*o)), nil

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(*o) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(v.Data)), nil // symbols, not bytes

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(mapRunesCase(DecodeText(*o), unicode.ToLower))), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(mapRunesCase(DecodeText(*o), unicode.ToUpper))), nil

	case "contains":
		return stringContainsMember(vm, v, args)

	case "count":
		return stringCount(vm, v, args)

	case "keep":
		return stringKeep(vm, v, args)

	case "remove":
		return stringRemove(vm, v, args)

	case "any":
		return stringAny(vm, v, args)

	case "all":
		return stringAll(vm, v, args)

	case "append":
		return stringAppend(vm, v, args)

	case "prepend":
		return stringPrepend(vm, v, args)

	case "push":
		return stringPush(vm, v, args)

	case "push_first":
		return stringPushFirst(vm, v, args)

	case "trim":
		return stringTrim(vm, v, args)

	case "trim_start":
		return stringTrimStart(vm, v, args)

	case "trim_end":
		return stringTrimEnd(vm, v, args)

	case "has_prefix":
		return stringHasPrefix(vm, v, args)

	case "has_suffix":
		return stringHasSuffix(vm, v, args)

	case "remove_prefix":
		return stringRemovePrefix(vm, v, args)

	case "remove_suffix":
		return stringRemoveSuffix(vm, v, args)

	case "replace":
		return stringReplace(vm, v, args)

	case "pad_start":
		return stringPadStart(vm, v, args)

	case "pad_end":
		return stringPadEnd(vm, v, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		slices.Reverse(rs)
		return NewStringValue(EncodeText(rs)), nil

	case "first":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		rs := DecodeText(*o)
		if len(rs) == 0 {
			// absence is data: undefined, or the optional trailing default
			return emptySeqResult(name, args)
		}
		return RuneValue(rs[0]), nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		rs := DecodeText(*o)
		if len(rs) == 0 {
			return emptySeqResult(name, args)
		}
		return RuneValue(rs[len(rs)-1]), nil

	case "min":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		rs := DecodeText(*o)
		if len(rs) == 0 {
			return emptySeqResult(name, args)
		}
		return RuneValue(slices.Min(rs)), nil

	case "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		rs := DecodeText(*o)
		if len(rs) == 0 {
			return emptySeqResult(name, args)
		}
		return RuneValue(slices.Max(rs)), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		slices.Sort(rs)
		return NewStringValue(EncodeText(rs)), nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		out := make([]rune, 0, len(rs))
		for i, r := range rs {
			if i == 0 || r != rs[i-1] {
				out = append(out, r)
			}
		}
		return NewStringValue(EncodeText(out)), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		out := make([]rune, 0, len(rs))
		seen := make(map[rune]struct{}, len(rs))
		for _, r := range rs {
			if _, ok := seen[r]; !ok {
				seen[r] = struct{}{}
				out = append(out, r)
			}
		}
		return NewStringValue(EncodeText(out)), nil

	case "slice":
		return sliceMember(v, args)

	case "chunk":
		return stringChunk(vm, v, args)

	case "insert":
		return stringInsert(vm, v, args)

	case "splice":
		return stringSplice(vm, v, args)

	case "map":
		return stringMap(vm, v, args)

	case "flat_map":
		return stringFlatMap(vm, v, args)

	case "reduce":
		return stringReduce(vm, v, args)

	case "case_fold":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		for i, r := range rs {
			rs[i] = foldRuneCanonical(r)
		}
		return NewStringValue(EncodeText(rs)), nil

	case "title_case":
		// the label rendering segments on WRITTEN boundaries only (case transitions stay inside words);
		// the identifier renderings re-segment fully
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(caseJoinTitle(caseSegmentWritten(DecodeText(*o))))), nil

	case "snake_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(caseJoinLower(caseSegmentWords(DecodeText(*o)), '_'))), nil

	case "kebab_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(caseJoinLower(caseSegmentWords(DecodeText(*o)), '-'))), nil

	case "camel_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(caseJoinCapitalized(caseSegmentWords(DecodeText(*o)), true))), nil

	case "pascal_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(EncodeText(caseJoinCapitalized(caseSegmentWords(DecodeText(*o)), false))), nil

	case "for_each":
		return stringForEach(vm, v, args)

	case "index":
		return stringIndex(vm, v, args)

	case "index_last":
		return stringIndexLast(vm, v, args)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		if _, err := SeqRepeatTotal(name, n, len(*o)); err != nil {
			return Undefined, err
		}
		return NewStringValue(strings.Repeat(*o, n)), nil

	case "split":
		return stringSplit(vm, v, args)

	case "partition":
		return stringPartition(vm, v, args)

	case "split_lines":
		return stringFnSplitLines(v, args)

	default:
		return CallMemberByLookup(vm, v, name, args)
	}
}

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all — the same readings as on runes (see the
// runes match members; the arguments are read by runesReadMatchArgs), over the string's SYMBOLS. No _in_place
// twins: a string is immutable by construction.
// ---------------------------------------------------------------------------

// stringContainsMember is contains(...): is there a match anywhere? The empty run is contained everywhere, the
// same answer `in` gives.
func stringContainsMember(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("contains", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))

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

// stringCount is count(...): how many matches. Runs count non-overlapping occurrences.
func stringCount(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("count", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))

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

// stringKeep is keep(...): a new string of the matches, in order.
func stringKeep(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("keep", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
		return NewStringValue(EncodeText(out)), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if hit {
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringRemove is remove(...): a new string without the matches, in order. With no argument remove drops the
// BLANK symbols — so it keeps the significant ones, landing on keep()'s answer by the opposite action.
func stringRemove(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("remove", args, false)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	out := make([]rune, 0, len(elems))

	if m.significant {
		for _, r := range elems {
			if !IsBlankRune(r) {
				out = append(out, r)
			}
		}
		return NewStringValue(EncodeText(out)), nil
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
		return NewStringValue(EncodeText(out)), nil
	}

	for i, r := range elems {
		hit, err := m.matches(vm, i, r)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			out = append(out, r)
		}
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringAny is any(...): does some symbol match? Symbols, a predicate, or nothing — never a run.
func stringAny(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("any", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range DecodeText(*(*string)(v.Ptr)) {
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

// stringAll is all(...): does every symbol match? Symbols, a predicate, or nothing — never a run. True on the empty
// string.
func stringAll(vm VM, v Value, args []Value) (Value, error) {
	m, err := runesReadMatchArgs("all", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, r := range DecodeText(*(*string)(v.Ptr)) {
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

// stringIndex is index(...): the first match.
func stringIndex(vm VM, v Value, args []Value) (Value, error) {
	return stringLocate(vm, "index", v, args, false)
}

// stringIndexLast is index_last(...): the last match.
func stringIndexLast(vm VM, v Value, args []Value) (Value, error) {
	return stringLocate(vm, "index_last", v, args, true)
}

// stringLocate is the body of index and index_last; name is the member called, for the errors.
func stringLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	elems := DecodeText(*(*string)(v.Ptr))

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

// stringReadElementSet reads a set of single symbols (the trim family's set, the pads' fill): each argument must be
// one symbol. A text-typed argument or a function raises with refuse, the member's own statement of what it
// takes.
func stringReadElementSet(name string, args []Value, refuse string) ([]rune, error) {
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

// stringReadRunSet reads a homogeneous set of runs (the anchored pair, split, partition): an element-typed argument
// is a run of its encoding, but one call may not mix element-typed and text-typed arguments. A function raises
// with onFunction, the member's own statement of what it takes.
func stringReadRunSet(name string, args []Value, onFunction string) ([][]rune, error) {
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

// stringTrimmed answers the receiver without its leading (start) and/or trailing (end) symbols that belong to the
// set. The name is the member called, for the errors.
func stringTrimmed(name string, v Value, args []Value, start, end bool) ([]rune, error) {
	set, err := stringReadElementSet(name, args, "a set of elements (the anchored run form is remove_prefix/remove_suffix; no predicate reading)")
	if err != nil {
		return nil, err
	}
	inSet := func(e rune) bool {
		if len(args) == 0 {
			return IsBlankRune(e)
		}
		return slices.Contains(set, e)
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
	return (elems[lo:hi]), nil
}

// stringAnchoredRun answers the length of the longest run of the argument set found at the start (or, with suffix,
// the end) of the receiver, or -1 when none is. The empty run is anchored everywhere. The name is the member
// called, for the errors.
func stringAnchoredRun(name string, v Value, args []Value, suffix bool) (int, error) {
	if len(args) == 0 {
		return 0, errs.NewWrongNumArgumentsError(name, "1 or more", 0)
	}
	runs, err := stringReadRunSet(name, args, "an element or a run (no predicate reading — \"the first element satisfies f\" is index(f) == 0)")
	if err != nil {
		return 0, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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

// stringWithoutAnchored answers the receiver without its longest matching prefix (or suffix) — removed once;
// unchanged when nothing matches. The name is the member called, for the errors.
func stringWithoutAnchored(name string, v Value, args []Value, suffix bool) ([]rune, error) {
	best, err := stringAnchoredRun(name, v, args, suffix)
	if err != nil {
		return nil, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	switch {
	case best <= 0:
		return (elems), nil
	case suffix:
		return (elems[:len(elems)-best]), nil
	default:
		return (elems[best:]), nil
	}
}

// stringReplaced answers the receiver with every occurrence of the run old (args[0]) replaced by new (args[1]); each
// argument is read on its own, an element being a run of one. The name is the member called, for the errors.
func stringReplaced(name string, v Value, args []Value) ([]rune, error) {
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
	elems := DecodeText(*(*string)(v.Ptr))
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

// stringPadded answers the receiver filled at the front (start) or the end up to n symbols; a width at or below the
// length leaves it unchanged. The name is the member called, for the errors.
func stringPadded(name string, v Value, args []Value, start bool) ([]rune, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errs.NewWrongNumArgumentsError(name, "1 or 2", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return nil, err
	}
	fill := ' '
	if len(args) == 2 {
		set, err := stringReadElementSet(name, args[1:], "one fill element (a run fill hides a truncation rule; build the run and append it instead)")
		if err != nil {
			return nil, err
		}
		fill = set[0]
	}
	elems := DecodeText(*(*string)(v.Ptr))
	if n <= int64(len(elems)) {
		return (elems), nil
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

// stringTrim is trim(...): without the leading and trailing elements of the set.
func stringTrim(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringTrimmed("trim", v, args, true, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringTrimStart is trim_start(...): without the leading elements of the set.
func stringTrimStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringTrimmed("trim_start", v, args, true, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringTrimEnd is trim_end(...): without the trailing elements of the set.
func stringTrimEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringTrimmed("trim_end", v, args, false, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringRemovePrefix is remove_prefix(...): without the longest matching prefix, once.
func stringRemovePrefix(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringWithoutAnchored("remove_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringRemoveSuffix is remove_suffix(...): without the longest matching suffix, once.
func stringRemoveSuffix(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringWithoutAnchored("remove_suffix", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringReplace is replace(...): every occurrence of old replaced by new.
func stringReplace(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringReplaced("replace", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringPadStart is pad_start(...): filled at the front up to n elements.
func stringPadStart(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringPadded("pad_start", v, args, true)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringPadEnd is pad_end(...): filled at the end up to n elements.
func stringPadEnd(_ VM, v Value, args []Value) (Value, error) {
	out, err := stringPadded("pad_end", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(out)), nil
}

// stringHasPrefix is has_prefix(...): does the receiver start with one of the runs?
func stringHasPrefix(_ VM, v Value, args []Value) (Value, error) {
	best, err := stringAnchoredRun("has_prefix", v, args, false)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(best >= 0), nil
}

// stringHasSuffix is has_suffix(...): does the receiver end with one of the runs?
func stringHasSuffix(_ VM, v Value, args []Value) (Value, error) {
	best, err := stringAnchoredRun("has_suffix", v, args, true)
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

// stringSplit is split(...): the pieces between the separators. Explicit separators keep the empty pieces between
// adjacent hits (n hits answer n+1 pieces); the blank form answers the maximal runs of significant symbols, the
// classic whitespace split.
func stringSplit(vm VM, v Value, args []Value) (Value, error) {
	const name = "split"
	elems := DecodeText(*(*string)(v.Ptr))

	if len(args) == 0 {
		var pieces []Value
		start := -1
		for i, e := range elems {
			if IsBlankRune(e) {
				if start >= 0 {
					pieces = append(pieces, stringPiece(elems[start:i]))
					start = -1
				}
			} else if start < 0 {
				start = i
			}
		}
		if start >= 0 {
			pieces = append(pieces, stringPiece(elems[start:]))
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
				pieces = append(pieces, stringPiece(elems[start:i]))
				start = i + 1
			}
		}
		pieces = append(pieces, stringPiece(elems[start:]))
		return NewArrayValue(pieces, false), nil
	}

	runs, err := stringReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
	if err != nil {
		return Undefined, err
	}
	for i := 0; i < len(elems); {
		if k := runLengthAt(elems, i, runs); k > 0 {
			pieces = append(pieces, stringPiece(elems[start:i]))
			i += k
			start = i
		} else {
			i++
		}
	}
	pieces = append(pieces, stringPiece(elems[start:]))
	return NewArrayValue(pieces, false), nil
}

// stringPartition is partition(...): the one-split form, [before, separator, after] around the first hit (the
// longest run at that position); a miss answers [receiver, empty, empty]. The blank form takes the whole run of
// blanks as the separator.
func stringPartition(vm VM, v Value, args []Value) (Value, error) {
	const name = "partition"
	elems := DecodeText(*(*string)(v.Ptr))
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
		runs, err := stringReadRunSet(name, args, "one reading per call (a function among several arguments always raises)")
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
		return NewArrayValue([]Value{stringPiece(elems), stringPiece(nil), stringPiece(nil)}, false), nil
	}
	return NewArrayValue([]Value{stringPiece(elems[:found]), stringPiece(elems[found : found+n]), stringPiece(elems[found+n:])}, false), nil
}

// stringPiece is one piece of split/partition, as a string.
func stringPiece(elems []rune) Value {
	return NewStringValue(EncodeText(elems))
}

// ---------------------------------------------------------------------------
// The callback members: map, flat_map, reduce (for_each is stringForEach, below). A per-element callback is f/1(symbol) or
// f/2(index, symbol); reduce's is f/2(acc, symbol) or f/3(acc, index, symbol).
// ---------------------------------------------------------------------------

// stringMap is map(f): strictly 1:1, answering a string — each callback result must be exactly one symbol (an
// in-range int, byte or rune); a run or undefined raises, because widening and dropping are flat_map's job.
func stringMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
	return NewStringValue(EncodeText(out)), nil
}

// stringFlatMap is flat_map(f): map, then concatenate — each callback result is text content appended as a run (a
// single element being a run of one); undefined contributes nothing.
func stringFlatMap(vm VM, v Value, args []Value) (Value, error) {
	const name = "flat_map"
	fn, err := readElemCallback(name, args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
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
	return NewStringValue(EncodeText(out)), nil
}

// stringReduce is reduce(acc, f): folds the symbols left to right.
func stringReduce(vm VM, v Value, args []Value) (Value, error) {
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range DecodeText(*(*string)(v.Ptr)) {
		acc, err = callReduce(vm, fn, acc, i, RuneValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// ---------------------------------------------------------------------------
// The add and edit members: append, prepend, push, push_first, chunk, splice, insert (slice is the Slice hook's
// member spelling). No _in_place twins and no views: a string is immutable by construction. The add side reads
// text content: append/prepend/splice's inserts are runs (an element being a run of one), push/push_first/insert
// take exactly one symbol per argument.
// ---------------------------------------------------------------------------

// stringAddItems reads the add side's operands (append, prepend, splice's inserts): every argument is text
// content, concatenated in argument order — mixing elements and runs is fine here (x.append("ab", 'c') is
// x + "ab" + 'c').
func stringAddItems(name string, args []Value) ([]rune, error) {
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

// stringPushItems reads push/push_first/insert's elements: each argument must be exactly one symbol — a text
// argument raises even at length 1, which is the member's purpose (append/prepend take runs).
func stringPushItems(name string, args []Value) ([]rune, error) {
	return stringReadElementSet(name, args, "one element (a sequence argument never reads as an element here; append/prepend take runs)")
}

// stringAppend is append(...items): x.append(a, b) is x + a + b.
func stringAppend(_ VM, v Value, args []Value) (Value, error) {
	items, err := stringAddItems("append", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(*(*string)(v.Ptr) + EncodeText(items)), nil
}

// stringPrepend is prepend(...items): x.prepend(a, b) is a + b + x.
func stringPrepend(_ VM, v Value, args []Value) (Value, error) {
	items, err := stringAddItems("prepend", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(items) + *(*string)(v.Ptr)), nil
}

// stringPush is push(...items): the items — exactly one symbol each — appended.
func stringPush(_ VM, v Value, args []Value) (Value, error) {
	items, err := stringPushItems("push", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(*(*string)(v.Ptr) + EncodeText(items)), nil
}

// stringPushFirst is push_first(...items): the items — exactly one symbol each — in front, in argument order.
func stringPushFirst(_ VM, v Value, args []Value) (Value, error) {
	items, err := stringPushItems("push_first", args)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(items) + *(*string)(v.Ptr)), nil
}

// stringChunk is chunk(size): the symbols in strings of size symbols (the last one shorter).
func stringChunk(_ VM, v Value, args []Value) (Value, error) {
	size, err := readChunkSize("chunk", args)
	if err != nil {
		return Undefined, err
	}
	elems := DecodeText(*(*string)(v.Ptr))
	step := len(elems)
	if size < int64(step) {
		step = int(size)
	}
	chunks := make([]Value, 0)
	for start := 0; start < len(elems); start += step {
		chunks = append(chunks, NewStringValue(EncodeText(elems[start:min(start+step, len(elems))])))
	}
	return NewArrayValue(chunks, false), nil
}

// stringSplice is splice([start[, count[, ...items]]]): a new string with count symbols at start replaced by the
// items (read like append's operands).
func stringSplice(_ VM, v Value, args []Value) (Value, error) {
	elems := DecodeText(*(*string)(v.Ptr))
	start, end, err := readSpliceRange(args, len(elems))
	if err != nil {
		return Undefined, err
	}
	var items []rune
	if len(args) > 2 {
		if items, err = stringAddItems("splice", args[2:]); err != nil {
			return Undefined, err
		}
	}
	out := make([]rune, 0, start+len(items)+len(elems)-end)
	out = append(out, elems[:start]...)
	out = append(out, items...)
	out = append(out, elems[end:]...)
	return NewStringValue(EncodeText(out)), nil
}

// stringInsert is insert(i, ...items): a new string with the items — one symbol each — at symbol position i, which
// raises out of [0, len].
func stringInsert(_ VM, v Value, args []Value) (Value, error) {
	const name = "insert"
	elems := DecodeText(*(*string)(v.Ptr))
	at, err := readEditPos(name, args, len(elems))
	if err != nil {
		return Undefined, err
	}
	items, err := stringPushItems(name, args[1:])
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(EncodeText(slices.Insert(elems, at, items...))), nil
}

// PURE by contract
func stringTypeAccessIndex(v Value, index Value) (Value, error) {
	i, ok := index.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
	}
	s := *(*string)(v.Ptr)
	rl := int64(v.Data)
	i, ok = NormalizeIndex(i, rl)
	if !ok {
		return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), int(rl))
	}
	// s[i] is the i-th SYMBOL and yields a rune — never a byte. An undecodable octet is one
	// symbol, its escape, exactly as iteration and .array() answer it
	if stringIsASCII(v, s) {
		return RuneValue(rune(s[i])), nil
	}
	j := int64(0)
	for k := 0; k < len(s); {
		r, w := utf8.DecodeRuneInString(s[k:])
		if r == utf8.RuneError && w <= 1 {
			r, w = OctetEscapeRune(s[k]), 1
		}
		if j == i {
			return RuneValue(r), nil
		}
		j++
		k += w
	}
	return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), int(rl))
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func stringTypeIterator(v Value) (Value, error) {
	o := (*string)(v.Ptr)
	return NewRunesIteratorValue(DecodeText(*o)), nil
}

// PURE by contract
func stringTypeAsInt(v Value) (int64, bool) {
	o := (*string)(v.Ptr)
	i, err := strconv.ParseInt(*o, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

// PURE by contract
func stringTypeAsFloat(v Value) (float64, bool) {
	o := (*string)(v.Ptr)
	f, err := strconv.ParseFloat(*o, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

// PURE by contract
func stringTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*string)(v.Ptr)
	d := dec128.FromString(*o)
	return d, !d.IsNaN()
}

// PURE by contract
func stringTypeAsTime(v Value) (time.Time, bool) {
	return parseTimeText(*(*string)(v.Ptr))
}

// PURE by contract
func stringTypeAsArray(v Value) ([]Value, bool) {
	o := (*string)(v.Ptr)
	rs := DecodeText(*o)
	arr := make([]Value, len(rs))
	for i, r := range rs {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

// PURE by contract
// stringTypeContains is the `in` operator: every accepted operand is text content encoded into the
// receiver's representation and matched as a run (the member's own acceptance); a callable raises.
func stringTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	run, _, err := runesEncodeMatchArg("in", e)
	if err != nil {
		return false, err
	}
	return strings.Contains(*(*string)(v.Ptr), string(run)), nil
}

// PURE by contract
func stringTypeSlice(v Value, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(v.Data) // symbol count, not byte count

	if s.Type != value.Undefined {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}

	if e.Type != value.Undefined {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, l)
	// bounds are SYMBOL offsets: translate to byte offsets before slicing, so the result can never split a
	// multi-byte rune / be invalid UTF-8
	bs, be := runeSpanToByteSpan(v, str, si, ei)
	return newStringValueCounted(str[bs:be], ei-si), nil
}

// runeSpanToByteSpan translates the rune-offset span [si, ei) into the byte-offset span of str that contains
// exactly those symbols. Offsets must already be normalized. O(1) on ASCII, one scan otherwise.
func runeSpanToByteSpan(v Value, str string, si, ei int64) (int64, int64) {
	if stringIsASCII(v, str) {
		return si, ei
	}
	bs, be := int64(len(str)), int64(len(str))
	j := int64(0)
	for bi := range str { // range yields the byte offset of each rune in turn
		if j == si {
			bs = int64(bi)
		}
		if j == ei {
			be = int64(bi)
			break
		}
		j++
	}
	return bs, be
}

// PURE by contract
func stringTypeSliceStep(v Value, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(v.Data) // symbol count, not byte count

	step, ok := stepVal.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
	}
	if step == 0 {
		return Undefined, errs.NewSliceStepZeroError()
	}

	if s.Type != value.Undefined {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}
	if e.Type != value.Undefined {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	start, end := NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, l)
	// stepping selects SYMBOLS — a byte-wise loop could slice a multi-byte rune apart and emit invalid UTF-8
	if stringIsASCII(v, str) {
		bs := []byte(str)
		result := make([]byte, 0, len(bs))
		if step > 0 {
			for i := start; i < end; i += step {
				result = append(result, bs[i])
			}
		} else {
			for i := start; i > end; i += step {
				result = append(result, bs[i])
			}
		}
		return newStringValueCounted(string(result), int64(len(result))), nil
	}
	rs := DecodeText(str)
	result := make([]rune, 0, len(rs))
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, rs[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, rs[i])
		}
	}
	return newStringValueCounted(EncodeText(result), int64(len(result))), nil
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
//
// stringForEach is for_each(f): a full pass whose callback result is ignored; returns the receiver, so it chains.
// It walks the Go string directly rather than decoding it first.
func stringForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	i := 0
	for _, r := range *(*string)(v.Ptr) {
		if _, err := callElem(vm, fn, i, RuneValue(r)); err != nil {
			return Undefined, err
		}
		i++
	}
	return v, nil
}

// PURE by contract
func stringFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*string)(v.Ptr)
	pieces := splitLinesString(*o)
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewStringValue(p)
	}
	return NewArrayValue(arr, false), nil
}
