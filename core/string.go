package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/jokruger/kavun/core/member/members"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

const stringTypeName = "string"

func NewStaticStringValue(s *string) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(text.TextRuneCount(*s)), Ptr: unsafe.Pointer(s)}
}

// NewStaticStringValueCounted is NewStaticStringValue with a precomputed rune count
func NewStaticStringValueCounted(s *string, runeLen int64) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(runeLen), Ptr: unsafe.Pointer(s)}
}

func NewStringValue(s string) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(text.TextRuneCount(s)), Ptr: unsafe.Pointer(&s)}
}

// newStringValueCounted skips the rune recount where the caller already knows it (substring/slice paths).
func newStringValueCounted(s string, runeLen int64) Value {
	return Value{Type: value.String, Immutable: true, Data: uint64(runeLen), Ptr: unsafe.Pointer(&s)}
}

// stringASCIIFastPath reports the fast path: every octet is ASCII, so byte offsets are symbol offsets and
// indexing/slicing stay O(1). It deliberately does NOT compare the rune count to the byte count — an
// undecodable octet also decodes to one symbol from one octet, so that test passes for text the fast path
// would then read wrongly (it would answer rune(0xFF) where every other operation answers the escape).
func stringASCIIFastPath(_ Value, s string) bool {
	return text.IsASCIIText(s)
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
	AsRunes:             func(v Value) ([]rune, bool) { return text.DecodeText(*(*string)(v.Ptr)), true },                      // PURE by contract
	AsBytes:             func(v Value) ([]byte, bool) { return []byte(*(*string)(v.Ptr)), true },                               // PURE by contract
	AsArray:             stringTypeAsArray,                                                                                     // PURE by contract

	Methods: []Method{
		members.IsTrue:       NewPureMethod(MemberIsTrue, 0, Fixed),
		members.String:       NewPureMethod(stringString, 0, Variadic),
		members.Format:       NewPureMethod(MemberFormat, 0, Variadic),
		members.Copy:         NewPureMethod(MemberSelf, 0, Fixed),
		members.Freeze:       NewPureMethod(MemberSelf, 0, Fixed),
		members.Runes:        NewPureMethod(stringRunes, 0, Fixed),
		members.Int:          NewPureMethod(stringInt, 0, Variadic),
		members.Bool:         NewPureMethod(stringBool, 0, Variadic),
		members.Float:        NewPureMethod(stringFloat, 0, Variadic),
		members.Time:         NewPureMethod(stringTime, 1, Variadic),
		members.Decimal:      NewPureMethod(stringDecimal, 0, Variadic),
		members.Date:         NewPureMethod(stringDate, 1, Variadic),
		members.Array:        NewPureMethod(stringArray, 0, Fixed),
		members.Bytes:        NewPureMethod(stringBytes, 0, Fixed),
		members.Len:          NewPureMethod(stringLen, 0, Fixed),
		members.IsEmpty:      NewPureMethod(stringIsEmpty, 0, Fixed),
		members.Contains:     NewPureMethod(stringContains, 0, Variadic),
		members.Index:        NewPureMethod(stringIndex, 0, Variadic),
		members.Count:        NewPureMethod(stringCount, 0, Variadic),
		members.All:          NewPureMethod(stringAll, 0, Variadic),
		members.Any:          NewPureMethod(stringAny, 0, Variadic),
		members.ForEach:      NewPureMethod(stringForEach, 1, Fixed),
		members.Reduce:       NewPureMethod(stringReduce, 2, Fixed),
		members.Keep:         NewPureMethod(stringKeep, 0, Variadic),
		members.Map:          NewPureMethod(stringMap, 1, Fixed),
		members.Remove:       NewPureMethod(stringRemove, 0, Variadic),
		members.First:        NewPureMethod(stringFirst, 0, Variadic),
		members.Last:         NewPureMethod(stringLast, 0, Variadic),
		members.IndexLast:    NewPureMethod(stringIndexLast, 0, Variadic),
		members.Min:          NewPureMethod(stringMin, 0, Variadic),
		members.Max:          NewPureMethod(stringMax, 0, Variadic),
		members.Slice:        NewPureMethod(memberSlice, 0, Variadic),
		members.Reverse:      NewPureMethod(stringReverse, 0, Fixed),
		members.Sort:         NewPureMethod(stringSort, 0, Fixed),
		members.Unique:       NewPureMethod(stringUnique, 0, Fixed),
		members.Dedup:        NewPureMethod(stringDedup, 0, Fixed),
		members.Chunk:        NewPureMethod(stringChunk, 1, Fixed),
		members.Append:       NewPureMethod(stringAppend, 0, Variadic),
		members.Prepend:      NewPureMethod(stringPrepend, 0, Variadic),
		members.Push:         NewPureMethod(stringPush, 0, Variadic),
		members.PushFirst:    NewPureMethod(stringPushFirst, 0, Variadic),
		members.Insert:       NewPureMethod(stringInsert, 1, Variadic),
		members.Splice:       NewPureMethod(stringSplice, 0, Variadic),
		members.Repeat:       NewPureMethod(stringRepeat, 1, Fixed),
		members.PadStart:     NewPureMethod(stringPadStart, 1, Variadic),
		members.PadEnd:       NewPureMethod(stringPadEnd, 1, Variadic),
		members.Trim:         NewPureMethod(stringTrim, 0, Variadic),
		members.TrimStart:    NewPureMethod(stringTrimStart, 0, Variadic),
		members.TrimEnd:      NewPureMethod(stringTrimEnd, 0, Variadic),
		members.HasPrefix:    NewPureMethod(stringHasPrefix, 1, Variadic),
		members.HasSuffix:    NewPureMethod(stringHasSuffix, 1, Variadic),
		members.RemovePrefix: NewPureMethod(stringRemovePrefix, 1, Variadic),
		members.RemoveSuffix: NewPureMethod(stringRemoveSuffix, 1, Variadic),
		members.Replace:      NewPureMethod(stringReplace, 2, Fixed),
		members.Split:        NewPureMethod(stringSplit, 0, Variadic),
		members.FlatMap:      NewPureMethod(stringFlatMap, 1, Fixed),
		members.IsASCII:      NewPureMethod(stringIsASCII, 0, Fixed),
		members.IsValid:      NewPureMethod(stringIsValid, 0, Fixed),
		members.SplitLines:   NewPureMethod(stringSplitLines, 0, Fixed),
		members.Partition:    NewPureMethod(stringPartition, 0, Variadic),
		members.Lower:        NewPureMethod(stringLower, 0, Fixed),
		members.Upper:        NewPureMethod(stringUpper, 0, Fixed),
		members.CaseFold:     NewPureMethod(stringCaseFold, 0, Fixed),
		members.TitleCase:    NewPureMethod(stringTitleCase, 0, Fixed),
		members.CamelCase:    NewPureMethod(stringCamelCase, 0, Fixed),
		members.PascalCase:   NewPureMethod(stringPascalCase, 0, Fixed),
		members.SnakeCase:    NewPureMethod(stringSnakeCase, 0, Fixed),
		members.KebabCase:    NewPureMethod(stringKebabCase, 0, Fixed),
		members.Nanoseconds:  NewPureMethod(stringNanoseconds, 0, Variadic),
	},
}

// PURE by contract
func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	// JSON text is UTF-8 by definition, so an octet that is not a symbol has nowhere to go here. It
	// raises rather than emitting a byte no parser can read back or silently becoming U+FFFD: the
	// conversions inside the language are total, the boundary OUT of the language is not, and a script
	// asks with is_valid() before it encodes. bytes.json() carries arbitrary octets, as base64
	if !text.TextIsValid(*o) {
		return nil, errs.NewConversionError(v.TypeName(), "json", "the text holds octets that are not symbols — encode it as bytes, or repair it (is_valid() finds them)")
	}
	var b []byte
	b = text.AppendJSONString(b, *o)
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
				l := text.EncodeRuneText(rune(other.Data))
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
				return NewStringValue(text.EncodeRuneText(rune(other.Data)) + *(*string)(v.Ptr)), nil
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
			if _, terr := seq.RepeatTotal(op.String(), n, len(src)); terr != nil {
				return Undefined, terr
			}
			return NewStringValue(strings.Repeat(src, n)), nil
		}
	}

	// + and - take text content, and the RECEIVER — the left operand — decides the result type; acceptance
	// mirrors the member layer minus int, whose operator reading stays arithmetic. + joins the octets, as append
	// does (a string is its octets). `-` removes every occurrence of the run as remove reads it — whole symbols,
	// leftmost non-overlapping; the empty run removes nothing
	if op == tokens.Add {
		s, ok, err := textOperandString(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			return NewStringValue(*(*string)(v.Ptr) + s), nil
		}
	}
	if op == tokens.Sub {
		run, ok, err := textOperandRunes(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			l := *(*string)(v.Ptr)
			if len(run) == 0 {
				return NewStringValue(l), nil
			}
			// a valid run matches only whole symbols, so the octet replacement is remove's reading; a run holding
			// an escape is removed symbol by symbol — its octets would also match inside a multi-octet symbol
			if text.RunesAreValid(run) {
				return NewStringValue(strings.ReplaceAll(l, text.EncodeText(run), "")), nil
			}
			return NewStringValue(text.EncodeText(seq.RemoveRun(text.DecodeText(l), run))), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func stringTypeAccessIndex(v Value, index Value) (Value, error) {
	i, ok := index.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
	}
	s := *(*string)(v.Ptr)
	rl := int64(v.Data)
	i, ok = seq.NormalizeIndex(i, rl)
	if !ok {
		return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), int(rl))
	}
	// s[i] is the i-th SYMBOL and yields a rune — never a byte. An undecodable octet is one
	// symbol, its escape, exactly as iteration and .array() answer it
	if stringASCIIFastPath(v, s) {
		return RuneValue(rune(s[i])), nil
	}
	j := int64(0)
	for k := 0; k < len(s); {
		r, w := text.TextFirst(s[k:])
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
	return NewRunesIteratorValue(text.DecodeText(*o)), nil
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
	rs := text.DecodeText(*o)
	arr := make([]Value, len(rs))
	for i, r := range rs {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

// PURE by contract
// stringTypeContains is the `in` operator: every accepted operand is text content read as a run (the member's own
// acceptance — contains(x) and x in s agree symbol for symbol, an undecodable octet included); a callable raises.
// The empty run is contained everywhere.
func stringTypeContains(v Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	run, _, matchable, err := textMatchArg("in", e)
	if err != nil {
		return false, err
	}
	if len(run) == 0 {
		return true, nil
	}
	if !matchable {
		return false, nil
	}
	return text.FindRun(*(*string)(v.Ptr), run, false) >= 0, nil
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

	si, ei = seq.NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, l)
	// bounds are SYMBOL offsets: translate to byte offsets before slicing, so the result can never split a
	// multi-byte rune / be invalid UTF-8
	bs, be := runeSpanToByteSpan(v, str, si, ei)
	return newStringValueCounted(str[bs:be], ei-si), nil
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

	start, end := seq.NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, l)
	// stepping selects SYMBOLS — a byte-wise loop could slice a multi-byte rune apart and emit invalid UTF-8
	if stringASCIIFastPath(v, str) {
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
	rs := text.DecodeText(str)
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
	return newStringValueCounted(text.EncodeText(result), int64(len(result))), nil
}
