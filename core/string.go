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
	"unicode/utf8"
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

// stringASCIIFastPath reports the fast path: every octet is ASCII, so byte offsets are symbol offsets and
// indexing/slicing stay O(1). It deliberately does NOT compare the rune count to the byte count — an
// undecodable octet also decodes to one symbol from one octet, so that test passes for text the fast path
// would then read wrongly (it would answer rune(0xFF) where every other operation answers the escape).
func stringASCIIFastPath(_ Value, s string) bool {
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

	Methods: []MethodEntry{
		members.IsTrue:       {Fn: memberIsTrue, Pure: true},
		members.String:       {Fn: stringString, Pure: true},
		members.Format:       {Fn: memberFormat, Pure: true},
		members.Copy:         {Fn: memberSelf, Pure: true},
		members.Freeze:       {Fn: memberSelf, Pure: true},
		members.Runes:        {Fn: stringRunes, Pure: true},
		members.Int:          {Fn: stringInt, Pure: true},
		members.Bool:         {Fn: stringBool, Pure: true},
		members.Float:        {Fn: stringFloat, Pure: true},
		members.Time:         {Fn: stringTime, Pure: true},
		members.Decimal:      {Fn: stringDecimal, Pure: true},
		members.Date:         {Fn: stringDate, Pure: true},
		members.Array:        {Fn: stringArray, Pure: true},
		members.Bytes:        {Fn: stringBytes, Pure: true},
		members.Len:          {Fn: stringLen, Pure: true},
		members.IsEmpty:      {Fn: stringIsEmpty, Pure: true},
		members.Contains:     {Fn: stringContains, Pure: true},
		members.Index:        {Fn: stringIndex, Pure: true},
		members.Count:        {Fn: stringCount, Pure: true},
		members.All:          {Fn: stringAll, Pure: true},
		members.Any:          {Fn: stringAny, Pure: true},
		members.ForEach:      {Fn: stringForEach, Pure: true},
		members.Reduce:       {Fn: stringReduce, Pure: true},
		members.Keep:         {Fn: stringKeep, Pure: true},
		members.Map:          {Fn: stringMap, Pure: true},
		members.Remove:       {Fn: stringRemove, Pure: true},
		members.First:        {Fn: stringFirst, Pure: true},
		members.Last:         {Fn: stringLast, Pure: true},
		members.IndexLast:    {Fn: stringIndexLast, Pure: true},
		members.Min:          {Fn: stringMin, Pure: true},
		members.Max:          {Fn: stringMax, Pure: true},
		members.Slice:        {Fn: memberSlice, Pure: true},
		members.Reverse:      {Fn: stringReverse, Pure: true},
		members.Sort:         {Fn: stringSort, Pure: true},
		members.Unique:       {Fn: stringUnique, Pure: true},
		members.Dedup:        {Fn: stringDedup, Pure: true},
		members.Chunk:        {Fn: stringChunk, Pure: true},
		members.Append:       {Fn: stringAppend, Pure: true},
		members.Prepend:      {Fn: stringPrepend, Pure: true},
		members.Push:         {Fn: stringPush, Pure: true},
		members.PushFirst:    {Fn: stringPushFirst, Pure: true},
		members.Insert:       {Fn: stringInsert, Pure: true},
		members.Splice:       {Fn: stringSplice, Pure: true},
		members.Repeat:       {Fn: stringRepeat, Pure: true},
		members.PadStart:     {Fn: stringPadStart, Pure: true},
		members.PadEnd:       {Fn: stringPadEnd, Pure: true},
		members.Trim:         {Fn: stringTrim, Pure: true},
		members.TrimStart:    {Fn: stringTrimStart, Pure: true},
		members.TrimEnd:      {Fn: stringTrimEnd, Pure: true},
		members.HasPrefix:    {Fn: stringHasPrefix, Pure: true},
		members.HasSuffix:    {Fn: stringHasSuffix, Pure: true},
		members.RemovePrefix: {Fn: stringRemovePrefix, Pure: true},
		members.RemoveSuffix: {Fn: stringRemoveSuffix, Pure: true},
		members.Replace:      {Fn: stringReplace, Pure: true},
		members.Split:        {Fn: stringSplit, Pure: true},
		members.FlatMap:      {Fn: stringFlatMap, Pure: true},
		members.IsASCII:      {Fn: stringIsASCII, Pure: true},
		members.IsValid:      {Fn: stringIsValid, Pure: true},
		members.SplitLines:   {Fn: stringSplitLines, Pure: true},
		members.Partition:    {Fn: stringPartition, Pure: true},
		members.Lower:        {Fn: stringLower, Pure: true},
		members.Upper:        {Fn: stringUpper, Pure: true},
		members.CaseFold:     {Fn: stringCaseFold, Pure: true},
		members.TitleCase:    {Fn: stringTitleCase, Pure: true},
		members.CamelCase:    {Fn: stringCamelCase, Pure: true},
		members.PascalCase:   {Fn: stringPascalCase, Pure: true},
		members.SnakeCase:    {Fn: stringSnakeCase, Pure: true},
		members.KebabCase:    {Fn: stringKebabCase, Pure: true},
	},
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

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all — the same readings as on runes (see the
// runes match members; the arguments are read by runesReadMatchArgs), over the string's SYMBOLS. No _in_place
// twins: a string is immutable by construction.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// The locators: index([x[, default]]) / index_last([x[, default]]) — the position of the first / last match,
// counted in symbols. The argument's type selects the reading:
//   - no argument      the first / last significant symbol (not IsBlankRune)
//   - a function       a predicate, f/1(symbol) or f/2(index, symbol)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one symbol, compared with ==; a value that is not one symbol raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// split(...seps) and partition(...seps): a separator is a run (an element being a run of one), a homogeneous set
// of runs, a predicate on single symbols, or — no argument — the blank set (IsBlankRune). Runs match leftmost-longest
// and never overlap; an empty run matches nothing.
// ---------------------------------------------------------------------------

// stringPiece is one piece of split/partition, as a string.
func stringPiece(elems []rune) Value {
	return NewStringValue(EncodeText(elems))
}

// ---------------------------------------------------------------------------
// The callback members: map, flat_map, reduce (for_each is stringForEach, below). A per-element callback is f/1(symbol) or
// f/2(index, symbol); reduce's is f/2(acc, symbol) or f/3(acc, index, symbol).
// ---------------------------------------------------------------------------

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
	if stringASCIIFastPath(v, s) {
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
	if stringASCIIFastPath(v, str) {
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
