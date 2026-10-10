package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/jokruger/kavun/core/member/members"
	"slices"
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
	Name:                MutabilityNameHook(runesTypeName, immutableRunesTypeName),                                 // PURE by contract
	String:              func(v Value) string { return "u" + strconv.Quote(EncodeText((*Runes)(v.Ptr).Elements)) }, // PURE by contract
	Format:              runesTypeFormat,                                                                           // PURE by contract
	Interface:           func(v Value) any { return (*Runes)(v.Ptr).Elements },                                     // PURE by contract
	EncodeJSON:          runesTypeEncodeJSON,                                                                       // PURE by contract
	EncodeBinary:        runesTypeEncodeBinary,                                                                     // PURE by contract
	DecodeBinary:        runesTypeDecodeBinary,                                                                     // IMPURE by contract (mutates target)
	IsTrue:              func(v Value) (bool, error) { return len((*Runes)(v.Ptr).Elements) > 0, nil },             // PURE by contract
	IsIterable:          ConstHook(true),                                                                           // PURE by contract
	Iterator:            runesTypeIterator,                                                                         // PURE by contract (constructs fresh iterator)
	Copy:                runesTypeCopy,                                                                             // PURE by contract
	Len:                 func(v Value) int64 { return int64(len((*Runes)(v.Ptr).Elements)) },                       // PURE by contract
	Equal:               runesTypeEqual,                                                                            // PURE by contract
	BinaryOp:            runesTypeBinaryOp,                                                                         // PURE by contract
	AccessIndex:         runesTypeAccessIndex,                                                                      // PURE by contract
	AccessNamedProperty: noNamedProperty,                                                                           // PURE by contract
	AssignIndex:         runesTypeAssignIndex,                                                                      // IMPURE by contract
	Append:              runesTypeAppend,                                                                           // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Contains:            runesTypeContains,                                                                         // PURE by contract
	Slice:               runesTypeSlice,                                                                            // PURE by contract
	SliceStep:           runesTypeSliceStep,                                                                        // PURE by contract
	AsBool:              runesTypeAsBool,                                                                           // PURE by contract
	AsInt:               runesTypeAsInt,                                                                            // PURE by contract
	AsFloat:             runesTypeAsFloat,                                                                          // PURE by contract
	AsDecimal:           runesTypeAsDecimal,                                                                        // PURE by contract
	AsTime:              runesTypeAsTime,                                                                           // PURE by contract
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

	Methods: []MethodEntry{
		members.IsTrue:              {Fn: memberIsTrue, Pure: true},
		members.String:              {Fn: runesString, Pure: true},
		members.Format:              {Fn: memberFormat, Pure: true},
		members.Copy:                {Fn: runesCopy, Pure: true},
		members.Freeze:              {Fn: runesFreeze, Pure: true},
		members.Runes:               {Fn: runesRunes, Pure: true},
		members.Int:                 {Fn: runesInt, Pure: true},
		members.Bool:                {Fn: runesBool, Pure: true},
		members.Float:               {Fn: runesFloat, Pure: true},
		members.Time:                {Fn: runesTime, Pure: true},
		members.Decimal:             {Fn: runesDecimal, Pure: true},
		members.Date:                {Fn: runesDate, Pure: true},
		members.Array:               {Fn: runesArray, Pure: true},
		members.Bytes:               {Fn: runesBytes, Pure: true},
		members.Len:                 {Fn: runesLen, Pure: true},
		members.IsEmpty:             {Fn: runesIsEmpty, Pure: true},
		members.Contains:            {Fn: runesContains, Pure: true},
		members.Index:               {Fn: runesIndex, Pure: true},
		members.Count:               {Fn: runesCount, Pure: true},
		members.All:                 {Fn: runesAll, Pure: true},
		members.Any:                 {Fn: runesAny, Pure: true},
		members.ForEach:             {Fn: runesForEach, Pure: true},
		members.Reduce:              {Fn: runesReduce, Pure: true},
		members.Keep:                {Fn: runesKeep, Pure: true},
		members.Map:                 {Fn: runesMap, Pure: true},
		members.Remove:              {Fn: runesRemove, Pure: true},
		members.First:               {Fn: runesFirst, Pure: true},
		members.Last:                {Fn: runesLast, Pure: true},
		members.IndexLast:           {Fn: runesIndexLast, Pure: true},
		members.Min:                 {Fn: runesMin, Pure: true},
		members.Max:                 {Fn: runesMax, Pure: true},
		members.Slice:               {Fn: memberSlice, Pure: true},
		members.Reverse:             {Fn: runesReverse, Pure: true},
		members.Sort:                {Fn: runesSort, Pure: true},
		members.Unique:              {Fn: runesUnique, Pure: true},
		members.Dedup:               {Fn: runesDedup, Pure: true},
		members.Chunk:               {Fn: runesChunk, Pure: true},
		members.Append:              {Fn: runesAppend, Pure: true},
		members.Prepend:             {Fn: runesPrepend, Pure: true},
		members.Push:                {Fn: runesPush, Pure: true},
		members.PushFirst:           {Fn: runesPushFirst, Pure: true},
		members.Insert:              {Fn: runesInsert, Pure: true},
		members.Splice:              {Fn: runesSplice, Pure: true},
		members.Repeat:              {Fn: runesRepeat, Pure: true},
		members.PadStart:            {Fn: runesPadStart, Pure: true},
		members.PadEnd:              {Fn: runesPadEnd, Pure: true},
		members.Trim:                {Fn: runesTrim, Pure: true},
		members.TrimStart:           {Fn: runesTrimStart, Pure: true},
		members.TrimEnd:             {Fn: runesTrimEnd, Pure: true},
		members.HasPrefix:           {Fn: runesHasPrefix, Pure: true},
		members.HasSuffix:           {Fn: runesHasSuffix, Pure: true},
		members.RemovePrefix:        {Fn: runesRemovePrefix, Pure: true},
		members.RemoveSuffix:        {Fn: runesRemoveSuffix, Pure: true},
		members.Replace:             {Fn: runesReplace, Pure: true},
		members.Split:               {Fn: runesSplit, Pure: true},
		members.FlatMap:             {Fn: runesFlatMap, Pure: true},
		members.IsASCII:             {Fn: runesIsASCII, Pure: true},
		members.IsValid:             {Fn: runesIsValid, Pure: true},
		members.SplitLines:          {Fn: runesSplitLines, Pure: true},
		members.Partition:           {Fn: runesPartition, Pure: true},
		members.Lower:               {Fn: runesLower, Pure: true},
		members.Upper:               {Fn: runesUpper, Pure: true},
		members.CaseFold:            {Fn: runesCaseFold, Pure: true},
		members.TitleCase:           {Fn: runesTitleCase, Pure: true},
		members.CamelCase:           {Fn: runesCamelCase, Pure: true},
		members.PascalCase:          {Fn: runesPascalCase, Pure: true},
		members.SnakeCase:           {Fn: runesSnakeCase, Pure: true},
		members.KebabCase:           {Fn: runesKebabCase, Pure: true},
		members.KeepInPlace:         {Fn: runesKeepInPlace, Pure: false},
		members.RemoveInPlace:       {Fn: runesRemoveInPlace, Pure: false},
		members.AppendInPlace:       {Fn: runesAppendInPlace, Pure: false},
		members.PrependInPlace:      {Fn: runesPrependInPlace, Pure: false},
		members.PushInPlace:         {Fn: runesPushInPlace, Pure: false},
		members.PushFirstInPlace:    {Fn: runesPushFirstInPlace, Pure: false},
		members.InsertInPlace:       {Fn: runesInsertInPlace, Pure: false},
		members.SpliceInPlace:       {Fn: runesSpliceInPlace, Pure: false},
		members.PadStartInPlace:     {Fn: runesPadStartInPlace, Pure: false},
		members.PadEndInPlace:       {Fn: runesPadEndInPlace, Pure: false},
		members.TrimInPlace:         {Fn: runesTrimInPlace, Pure: false},
		members.TrimStartInPlace:    {Fn: runesTrimStartInPlace, Pure: false},
		members.TrimEndInPlace:      {Fn: runesTrimEndInPlace, Pure: false},
		members.RemovePrefixInPlace: {Fn: runesRemovePrefixInPlace, Pure: false},
		members.RemoveSuffixInPlace: {Fn: runesRemoveSuffixInPlace, Pure: false},
		members.ReplaceInPlace:      {Fn: runesReplaceInPlace, Pure: false},
		members.ReverseInPlace:      {Fn: runesReverseInPlace, Pure: false},
		members.SortInPlace:         {Fn: runesSortInPlace, Pure: false},
		members.UniqueInPlace:       {Fn: runesUniqueInPlace, Pure: false},
		members.DedupInPlace:        {Fn: runesDedupInPlace, Pure: false},
		members.SliceView:           {Fn: runesSliceView, Pure: true},
		members.ChunkView:           {Fn: runesChunkView, Pure: true},
	},
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
			case tokens.Less:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l < r), nil
			case tokens.LessEq:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l <= r), nil
			case tokens.Greater:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l > r), nil
			case tokens.GreaterEq:
				l := *(*string)(other.Ptr)
				r := EncodeText((*Runes)(v.Ptr).Elements)
				return BoolValue(l >= r), nil
			}

		case value.Rune:
			switch op {
			case tokens.Add:
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
			case tokens.Add:
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
		case tokens.Less:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l < r), nil
		case tokens.LessEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l <= r), nil
		case tokens.Greater:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l > r), nil
		case tokens.GreaterEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := EncodeText((*Runes)(other.Ptr).Elements)
			return BoolValue(l >= r), nil
		}

	case value.String:
		switch op {
		case tokens.Less:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l < r), nil
		case tokens.LessEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l <= r), nil
		case tokens.Greater:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l > r), nil
		case tokens.GreaterEq:
			l := EncodeText((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
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
	if op == tokens.Add || op == tokens.Sub {
		s, ok, err := textOperandString(other)
		if err != nil {
			return Undefined, err
		}
		if ok {
			l := EncodeText((*Runes)(v.Ptr).Elements)
			if op == tokens.Add {
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
func runesTypeAccessIndex(v Value, index Value) (Value, error) {
	elems := (*Runes)(v.Ptr).Elements
	i, err := resolveIndex("index access", index, len(elems))
	if err != nil {
		return Undefined, err
	}
	return RuneValue(elems[i]), nil
}

// IMPURE by contract: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func runesTypeAssignIndex(v Value, index Value, r Value) error {
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

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match,
// counted in symbols. The argument's type selects the reading:
//   - no argument      the first / last significant symbol (not IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one symbol, compared with ==; a value that is not one symbol raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// split(...seps) and partition(...seps): a separator is a run (an element being a run of one), a homogeneous set
// of runs, a predicate on single symbols, or — no argument — the blank set (IsBlankRune). Runs match leftmost-longest
// and never overlap; an empty run matches nothing.
// ---------------------------------------------------------------------------

// runesPiece is one piece of split/partition: new runes, independent of the receiver.
func runesPiece(elems []rune) Value {
	return NewRunesValue(slices.Clone(elems), false)
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(symbol) or
// f/2(index, symbol); reduce's is f/2(acc, symbol) or f/3(acc, index, symbol).
// ---------------------------------------------------------------------------

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
