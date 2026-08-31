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
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
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
	Name:         ConstHook(stringTypeName),                                                   // PURE by contract
	String:       func(v Value) string { return strconv.Quote(*(*string)(v.Ptr)) },            // PURE by contract
	Format:       stringTypeFormat,                                                            // PURE by contract
	Interface:    func(v Value) any { return *(*string)(v.Ptr) },                              // PURE by contract
	EncodeJSON:   stringTypeEncodeJSON,                                                        // PURE by contract
	EncodeBinary: stringTypeEncodeBinary,                                                      // PURE by contract
	DecodeBinary: stringTypeDecodeBinary,                                                      // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return len(*(*string)(v.Ptr)) > 0, nil },      // PURE by contract
	IsIterable:   ConstHook(true),                                                             // PURE by contract
	Iterator:     stringTypeIterator,                                                          // PURE by contract (constructs fresh iterator)
	Len:          func(v Value) int64 { return int64(v.Data) },                                // PURE by contract — symbols, not bytes; the count is cached at construction
	Equal:        stringTypeEqual,                                                             // PURE by contract
	BinaryOp:     stringTypeBinaryOp,                                                          // PURE by contract
	MethodCall:   stringTypeMethodCall,                                                        // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       stringTypeAccess,                                                            // PURE by contract
	Contains:     stringTypeContains,                                                          // PURE by contract
	Slice:        stringTypeSlice,                                                             // PURE by contract
	SliceStep:    stringTypeSliceStep,                                                         // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return conv.ParseBool(*(*string)(v.Ptr)) },     // PURE by contract
	AsInt:        stringTypeAsInt,                                                             // PURE by contract
	AsFloat:      stringTypeAsFloat,                                                           // PURE by contract
	AsDecimal:    stringTypeAsDecimal,                                                         // PURE by contract
	AsTime:       stringTypeAsTime,                                                            // PURE by contract
	AsString:     func(v Value) (string, bool) { return *(*string)(v.Ptr), true },             // PURE by contract
	AsRunes:      func(v Value) ([]rune, bool) { return DecodeText(*(*string)(v.Ptr)), true }, // PURE by contract
	AsBytes:      func(v Value) ([]byte, bool) { return []byte(*(*string)(v.Ptr)), true },     // PURE by contract
	AsArray:      stringTypeAsArray,                                                           // PURE by contract
	IsMethodPure: func(string) bool { return true },                                           // All methods are expected to be pure.
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
			case token.Add:
				l := EncodeRuneText(rune(other.Data))
				r := *(*string)(v.Ptr)
				return NewStringValue(l + r), nil
			}
		case value.Byte:
			// a scalar on the left takes the sequence's type; an octet is a symbol only in ASCII
			switch op {
			case token.Add:
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
		case token.Less:
			return BoolValue(l < r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.GreaterEq:
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
	if op == token.Add || op == token.Sub {
		s, ok, err := textOperandString(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			l := *(*string)(v.Ptr)
			if op == token.Add {
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

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func stringTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

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
		t, ok := stringTypeAsTime(v)
		return convMember(name, stringTypeName, args, ok, NewTimeValue(t))

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
		s, err := stringTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

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

	case "contains", "count", "filter", "remove", "any", "all":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return TripleMatchMember(vm, name, v, args, RuneValue,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq },
			runesEncodeMatchArg, func(a, b rune) bool { return a == b }, IsBlankRune)

	case "append", "prepend":
		// whole-operand concatenation, arguments in order: x.append(a, b) ≡ x + a + b,
		// x.prepend(a, b) ≡ a + b + x. No _in_place twins — the receiver is immutable
		// by construction, so each could only raise.
		items, err := tripleAddItems(name, args, runesEncodeMatchArg)
		if err != nil {
			return Undefined, err
		}
		if name == "append" {
			return NewStringValue(*o + EncodeText(items)), nil
		}
		return NewStringValue(EncodeText(items) + *o), nil

	case "push", "push_first":
		// the VALIDATING element add: each argument must be a single symbol; a sequence
		// argument raises even at length 1 — the refusal is the member's purpose
		items, err := triplePushItems(name, args, runesEncodeMatchArg)
		if err != nil {
			return Undefined, err
		}
		if name == "push" {
			return NewStringValue(*o + EncodeText(items)), nil
		}
		return NewStringValue(EncodeText(items) + *o), nil

	case "trim", "trim_start", "trim_end", "has_prefix", "has_suffix",
		"remove_prefix", "remove_suffix", "replace", "pad_start", "pad_end":
		// symbol width, symbol sets; the default fill and blank set are the space
		// symbol and NUL ∪ ASCII whitespace. Unsuffixed only — the receiver is
		// immutable by construction, so no _in_place twins exist here
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return SeqStructuralMember(name, v, args,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq },
			runesEncodeMatchArg, tripleFillElement(runesEncodeMatchArg), ' ',
			func(a, b rune) bool { return a == b }, IsBlankRune)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		slices.Reverse(rs)
		return NewStringValue(EncodeText(rs)), nil

	case "first", "last", "min", "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		rs := DecodeText(*o)
		if len(rs) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		switch name {
		case "first":
			return RuneValue(rs[0]), nil
		case "last":
			return RuneValue(rs[len(rs)-1]), nil
		case "min":
			return RuneValue(slices.Min(rs)), nil
		default:
			return RuneValue(slices.Max(rs)), nil
		}

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
		return SeqSlice(v, args)

	case "chunk":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return SeqChunk(v, args,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq })

	case "insert":
		// the element-inserting sibling of splice (pure only — the receiver is immutable
		// by construction): each item must be a single symbol; the position is an EDIT
		// and raises out of range
		rs := DecodeText(*o)
		at, err := seqEditPos(name, args, int64(len(rs)))
		if err != nil {
			return Undefined, err
		}
		items, err := triplePushItems(name, args[1:], runesEncodeMatchArg)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(EncodeText(slices.Insert(rs, int(at), items...))), nil

	case "splice":
		// the pure form only — string is immutable by construction, so no splice_in_place
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return SeqSplice(append([]Value{v}, args...), false,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq },
			runesAppendItems, stringTypeName)

	case "map":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return TripleMapMember(vm, name, v, args, RuneValue,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq },
			runesEncodeMatchArg)

	case "flat_map":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return TripleFlatMapMember(vm, name, v, args, RuneValue,
			func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) },
			func(Value) *Seq[rune] { return &seq },
			runesEncodeMatchArg)

	case "reduce":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return SeqReduce(vm, v, args, RuneValue, func(Value) *Seq[rune] { return &seq })

	case "case_fold":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := DecodeText(*o)
		for i, r := range rs {
			rs[i] = foldRuneCanonical(r)
		}
		return NewStringValue(EncodeText(rs)), nil

	case "title_case", "snake_case", "kebab_case", "camel_case", "pascal_case":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// the label rendering segments on WRITTEN boundaries only (case transitions
		// stay inside words); the identifier renderings re-segment fully
		if name == "title_case" {
			return NewStringValue(EncodeText(caseRenderWords(name, caseSegmentWritten(DecodeText(*o))))), nil
		}
		return NewStringValue(EncodeText(caseRenderWords(name, caseSegmentWords(DecodeText(*o))))), nil

	case "for_each":
		return stringFnForEach(vm, v, args)

	case "index", "index_last":
		// offsets are SYMBOL positions (directly usable with [i] and slicing);
		// materializes the runes — the documented cost of compact storage
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		return SeqIndex(vm, v, args, name == "index_last", RuneValue,
			func(Value) *Seq[rune] { return &seq },
			func(a Value) bool { return a.Type == value.String || a.Type == value.Runes || a.Type == value.Bytes },
			func(elems []rune, run Value, last bool) (int64, bool, error) {
				rr, ok := run.AsRunes()
				if !ok {
					return -1, false, errs.NewInvalidArgumentTypeError(name, "first", "text content", run.TypeName())
				}
				idx, found := SeqIndexRun(elems, rr, RuneValue, last)
				return idx, found, nil
			},
			tripleElemCheck(runesEncodeMatchArg),
			IsBlankRune)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		if _, err := SeqRepeatTotal(name, n, len(*o)); err != nil {
			return Undefined, err
		}
		return NewStringValue(strings.Repeat(*o, n)), nil

	case "split", "partition":
		rs := DecodeText(*o)
		seq := Seq[rune]{Elements: rs}
		allocPiece := func(out []rune, _ bool) Value { return NewStringValue(EncodeText(out)) }
		resolve := func(Value) *Seq[rune] { return &seq }
		eq := func(a, b rune) bool { return a == b }
		if name == "split" {
			return SeqSplitMember(vm, name, v, args, RuneValue, allocPiece, resolve, runesEncodeMatchArg, eq, IsBlankRune)
		}
		return SeqPartitionMember(vm, name, v, args, RuneValue, allocPiece, resolve, runesEncodeMatchArg, eq, IsBlankRune)

	case "split_lines":
		return stringFnSplitLines(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE by contract
func stringTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	if mode == bc.AccessIndex {
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

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
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
// PURE by contract with higher-order rule caveat (see docs/purity.md)
// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see SeqForEach)
	o := (*string)(v.Ptr)
	var buf [2]Value
	i := int64(0)
	for _, r := range *o {
		if fn.Arity() == 2 {
			buf[0] = IntValue(i)
			buf[1] = RuneValue(r)
			if _, err := fn.Call(vm, buf[:2]); err != nil {
				return Undefined, err
			}
		} else {
			buf[0] = RuneValue(r)
			if _, err := fn.Call(vm, buf[:1]); err != nil {
				return Undefined, err
			}
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
