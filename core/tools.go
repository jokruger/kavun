package core

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
)

// MutabilityNameHook answers a container's type name by its mutability: name for a mutable value, immutableName
// for an immutable one ("array" / "immutable-array").
func MutabilityNameHook(name, immutableName string) func(Value) string {
	return func(v Value) string {
		if v.Immutable {
			return immutableName
		}
		return name
	}
}

// readChunkSize reads chunk's / chunk_view's single argument (the dispatcher guarantees it): an int-valued size of
// at least 1.
func readChunkSize(name string, args []Value) (int64, error) {
	size, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return 0, err
	}
	if size < 1 {
		return 0, errs.NewInvalidValueError("chunk size must be positive")
	}
	return size, nil
}

// readSliceArgs reads the member form's optional (start[, end]) as the two bounds the Slice hook takes — undefined
// standing for an omitted bound, exactly like the operator's a[i:] / a[:j] / a[:].
func readSliceArgs(name string, args []Value) (Value, Value, error) {
	if len(args) > 2 {
		return Undefined, Undefined, ErrArgCount
	}
	s, e := Undefined, Undefined
	if len(args) > 0 {
		s = args[0]
	}
	if len(args) > 1 {
		e = args[1]
	}
	return s, e, nil
}

// resolveSliceBounds parses a (start, end) bound pair — undefined meaning omitted — and normalises it against the
// length: negative bounds count from the end, and both are clamped into [0, length] with start <= end.
func resolveSliceBounds(name string, s, e Value, length int) (int, int, error) {
	var si, ei int64
	var ok bool
	if s.Type != value.Undefined {
		if si, ok = s.AsInt(); !ok {
			return 0, 0, errs.NewInvalidIndexTypeError(name, "int", s.TypeName())
		}
	}
	if e.Type != value.Undefined {
		if ei, ok = e.AsInt(); !ok {
			return 0, 0, errs.NewInvalidIndexTypeError(name, "int", e.TypeName())
		}
	}
	si, ei = seq.NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, int64(length))
	return int(si), int(ei), nil
}

// resolveSliceStep parses a stepped slice a[s:e:step] and normalises it against the length; the step must be a
// non-zero int. The answer is the first position, the exclusive stop, and the step.
func resolveSliceStep(s, e, stepVal Value, length int) (int, int, int, error) {
	step, ok := stepVal.AsInt()
	if !ok {
		return 0, 0, 0, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
	}
	if step == 0 {
		return 0, 0, 0, errs.NewSliceStepZeroError()
	}
	var si, ei int64
	if s.Type != value.Undefined {
		if si, ok = s.AsInt(); !ok {
			return 0, 0, 0, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}
	if e.Type != value.Undefined {
		if ei, ok = e.AsInt(); !ok {
			return 0, 0, 0, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}
	si, ei = seq.NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, int64(length))
	return int(si), int(ei), int(step), nil
}

// resolveIndex parses an element index — an int, negative counting from the end — and checks it against the
// length. name is the operation, for the errors ("index access", "index assign").
func resolveIndex(name string, index Value, length int) (int, error) {
	i, ok := index.AsInt()
	if !ok {
		return 0, errs.NewInvalidIndexTypeError(name, "int", index.TypeName())
	}
	if i, ok = seq.NormalizeIndex(i, int64(length)); !ok {
		return 0, errs.NewIndexOutOfBoundsError(name, int(i), length)
	}
	return int(i), nil
}

// readEditPos reads a positional edit's slot (insert's position): int-valued and lossless, negative counting from
// the end, and — editing past the end is not harmless, unlike reading — out of [0, length] raises. The dispatcher
// guarantees the slot (the template's first parameter).
func readEditPos(name string, args []Value, length int) (int, error) {
	i, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return 0, err
	}
	orig := i
	if i < 0 {
		i += int64(length)
	}
	if i < 0 || i > int64(length) {
		return 0, errs.NewIndexOutOfBoundsError(name, int(orig), length)
	}
	return int(i), nil
}

// readSpliceRange reads splice's (start[, count]) against the length and answers the edited span [start, end).
// start counts from the end when negative and must land in [0, length]; count must be non-negative and is clamped
// to what remains; both default to "the rest". name is the calling member, so each twin reports itself.
func readSpliceRange(name string, args []Value, length int) (int, int, error) {
	start := 0
	if len(args) > 0 {
		arg, ok := args[0].AsInt()
		if !ok {
			return 0, 0, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
		}
		start = int(arg)
		if start < 0 {
			start += length
		}
		if start < 0 || start > length {
			return 0, 0, errs.NewIndexOutOfBoundsError(name+", start index", int(arg), length)
		}
	}
	count := length - start
	if len(args) > 1 {
		arg, ok := args[1].AsInt()
		if !ok {
			return 0, 0, errs.NewInvalidArgumentTypeError(name, "second", "int", args[1].TypeName())
		}
		if arg < 0 {
			return 0, 0, errs.NewInvalidValueError("(" + name + ") delete count must be non-negative")
		}
		if arg < int64(count) {
			count = int(arg)
		}
	}
	return start, start + count, nil
}

// readElemCallback reads the single callback argument of a per-element member (map, flat_map, for_each): one
// function called with the element (f/1) or, when it does not accept one argument, the (index, element) pair
// (f/2) — the fewest form wins (CallbackForm). The dispatcher guarantees the one argument.
func readElemCallback(vm VM, name string, args []Value) (*ElemCallback, error) {
	fn := args[0]
	if !fn.IsCallable() {
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "function", fn.TypeName())
	}
	return newElemCallback(vm, name, fn)
}

// readReduceArgs reads reduce(acc, f): the initial accumulator and a function called with (acc, element) (f/2)
// when it accepts two arguments, else with (acc, index, element) (f/3) when it accepts three (CallbackForm). The
// dispatcher guarantees the two arguments.
func readReduceArgs(vm VM, args []Value) (Value, *ReduceCallback, error) {
	acc, fn := args[0], args[1]
	if !fn.IsCallable() {
		return Undefined, nil, errs.NewInvalidArgumentTypeError("reduce", "second", "function", fn.TypeName())
	}
	arity, ok := CallbackForm(fn, 2, 3)
	if !ok {
		return Undefined, nil, errs.NewInvalidArgumentTypeError("reduce", "second", "f/2 or f/3", fn.TypeName())
	}
	return acc, &ReduceCallback{vm: vm, fn: fn, arity: arity}, nil
}

// ReduceCallback is a reduce callback read once per member call by readReduceArgs. Like ElemCallback it reuses
// one argument buffer for every call.
type ReduceCallback struct {
	vm    VM
	fn    Value
	arity int      // 2 or 3
	buf   [3]Value // the callee never retains or mutates its arguments
}

// Call calls the callback for the element e at index i: f/2 receives (acc, element), f/3 (acc, index, element).
func (c *ReduceCallback) Call(acc Value, i int, e Value) (Value, error) {
	c.buf[0] = acc
	if c.arity == 3 {
		c.buf[1] = IntValue(int64(i))
		c.buf[2] = e
		return c.fn.Call(c.vm, c.buf[:3])
	}
	c.buf[1] = e
	return c.fn.Call(c.vm, c.buf[:2])
}

// CallEntry is Call for a map, whose element is its key: f/2 receives (acc, key), f/3 (acc, key, value).
func (c *ReduceCallback) CallEntry(acc Value, k Value, val Value) (Value, error) {
	c.buf[0] = acc
	c.buf[1] = k
	if c.arity == 3 {
		c.buf[2] = val
		return c.fn.Call(c.vm, c.buf[:3])
	}
	return c.fn.Call(c.vm, c.buf[:2])
}

// mapsEqual checks if two maps of string to Value are equal, using Value.Equal for value comparison.
func mapsEqual(a, b map[string]Value) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !av.Equal(bv) {
			return false
		}
	}
	return true
}

// mergeMaps merges two maps of string to Value by copying all entries from both maps into a new map.
// If a key exists in both maps, the value from the second map overwrites the value from the first map.
func mergeMaps(a, b map[string]Value) map[string]Value {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	result := make(map[string]Value, len(a)+len(b))
	maps.Copy(result, a)
	maps.Copy(result, b)
	return result
}

// parseIntArg reads an int-valued slot (a count, a width, an edit position): any numeric is accepted iff the
// conversion is LOSSLESS — repeat(2.0) is repeat(2), repeat(1.5) raises rather than silently truncating —
// and everything non-convertible is a type error.
func parseIntArg(name, pos string, a Value) (int64, error) {
	i, ok := a.AsInt()
	if !ok {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "int", a.TypeName())
	}
	switch a.Type {
	case value.Float, value.Decimal:
		if !a.Equal(IntValue(i)) {
			return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) argument %s must be a whole number, got %s", name, pos, a.String()))
		}
	}
	return i, nil
}

// CallbackForm picks the argument count a callback fn is called with: the FEWEST k in fewest..most that fn
// accepts by its pair (arity N, variadic), where accepts(k) is k == N, or variadic and k > N. ok is false when it
// accepts none of them. The pair alone decides — never the body, never the callable's kind — so a script function,
// a builtin and a bound member with the same pair are called the same way: an element callback (1, 2) takes
// func(x), func(x, ...a), func(...a), min, int or s.contains as f/1 and only func(i, x) as f/2. A lambda is the one
// way to call a callable in another form (choose it, reorder, fix or drop arguments).
func CallbackForm(fn Value, fewest, most int) (k int, ok bool) {
	arity, variadic := fn.Arity(), fn.IsVariadic()
	for k := fewest; k <= most; k++ {
		if k == arity || (variadic && k > arity) {
			return k, true
		}
	}
	return 0, false
}

// ElemCallback is a per-element callback read once per member call (readElemCallback, newElemCallback): f/1
// receives the element, f/2 the (index, element) pair. It reuses one argument buffer for every call, which is
// sound because a callee never retains or mutates its arguments (docs/conventions.md).
type ElemCallback struct {
	vm    VM
	fn    Value
	arity int      // 1 or 2
	buf   [2]Value // the argument buffer
}

// newElemCallback validates a per-element callback argument fn, already known to be callable: it is called with
// the element (f/1) when it accepts one argument, else with the (index, element) pair (f/2) when it accepts two,
// and refused otherwise (CallbackForm).
func newElemCallback(vm VM, name string, fn Value) (*ElemCallback, error) {
	arity, ok := CallbackForm(fn, 1, 2)
	if !ok {
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
	}
	return &ElemCallback{vm: vm, fn: fn, arity: arity}, nil
}

// Call calls the callback for the element e at index i: f/1 receives the element, f/2 (index, element).
func (c *ElemCallback) Call(i int, e Value) (Value, error) {
	if c.arity == 2 {
		c.buf[0] = IntValue(int64(i))
		c.buf[1] = e
		return c.fn.Call(c.vm, c.buf[:2])
	}
	c.buf[0] = e
	return c.fn.Call(c.vm, c.buf[:1])
}

// CallEntry is Call for a map, whose element is its key: f/1 receives the key, f/2 (key, value).
func (c *ElemCallback) CallEntry(k Value, val Value) (Value, error) {
	c.buf[0] = k
	if c.arity == 2 {
		c.buf[1] = val
		return c.fn.Call(c.vm, c.buf[:2])
	}
	return c.fn.Call(c.vm, c.buf[:1])
}

// locatorAnswer is the uniform miss contract of the locators (index/index_last): a position answers itself;
// absence (idx < 0) answers undefined — never an in-band sentinel like -1, which negative indexing would silently
// accept — or the optional trailing default.
func locatorAnswer(idx int, dflt []Value) (Value, error) {
	if idx >= 0 {
		return IntValue(int64(idx)), nil
	}
	if len(dflt) == 1 {
		return dflt[0], nil
	}
	return Undefined, nil
}

// emptySeqResult is the one answer for "this member has no value to give, because the sequence is empty":
// first/last/min/max/sum/avg. It raises, like every other operation that cannot produce a valid value, unless the
// caller supplied the member form's trailing default — the same opt-out the conversions use (`"bad".int(0)`).
// A silent undefined here used to make an empty input indistinguishable from a real undefined element.
// PURE by contract.
func emptySeqResult(name string, args []Value) (Value, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) empty sequence", name))
}

// SeqRepeatOperand reads the right operand of a sequence's `*` as a repeat count, so `x * n` is exactly
// `x.repeat(n)`. Only a numeric operand is a count: anything else is not a `*` at all and the caller must fall
// through to its ordinary invalid-operator path (a `false` second result), so `[1] * "ab"` reads as
// `invalid_binary_operator` rather than a count that failed to parse. A numeric one is held to the member's
// contract — whole-valued and non-negative — and its failures are catchable, named for the operator.
// PURE by contract.
func SeqRepeatOperand(other Value) (int, bool, error) {
	switch other.Type {
	case value.Int, value.Float, value.Decimal:
	default:
		return 0, false, nil
	}
	n, err := parseIntArg("*", "right operand", other)
	if err != nil {
		return 0, true, err
	}
	if n < 0 {
		// argument validation must be catchable by recover()
		return 0, true, errs.NewInvalidValueError(fmt.Sprintf("(*) repeat count must be non-negative, got %d", n))
	}
	return int(n), true, nil
}

// parseRepeatCount validates and extracts the count argument for a `repeat` method.
// Its one int argument is guaranteed by the dispatcher; a negative count raises.
func parseRepeatCount(name string, args []Value) (int, error) {
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return 0, err
	}
	if n < 0 {
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) repeat count must be non-negative, got %d", name, n))
	}
	return int(n), nil
}

// MapToSortedEntries materializes a map's conversion elements — its entries — as [[k, v], ...] in canonical
// key-sorted order, so the two directions of the sequence<->map boundary round-trip up to that ordering.
func MapToSortedEntries(m map[string]Value) []Value {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]Value, len(keys))
	for i, k := range keys {
		entries[i] = NewArrayValue([]Value{NewStringValue(k), m[k]}, false)
	}
	return entries
}

// ElementsToRunes converts an element container's content to symbols: each element through its own rune
// conversion, all-or-nothing — a failing element fails the whole conversion, with no partial result and no
// substituted placeholder (the silent NUL/U+FFFD corruption this replaces).
func ElementsToRunes(elems []Value) ([]rune, bool) {
	rs := make([]rune, len(elems))
	for i, e := range elems {
		r, ok := e.AsRune()
		if !ok {
			return nil, false
		}
		rs[i] = r
	}
	return rs, true
}

// ElementsToBytes is ElementsToRunes' octet twin.
func ElementsToBytes(elems []Value) ([]byte, bool) {
	bs := make([]byte, len(elems))
	for i, e := range elems {
		b, ok := e.AsByte()
		if !ok {
			return nil, false
		}
		bs[i] = b
	}
	return bs, true
}

// ElementsToEntries reads a sequence as a map: each element must be EXACTLY a 2-element array (an entry) — any
// other element fails, including a 2-element text sequence, so a misread never silently becomes a map. The key
// goes through its own string conversion (absent on undefined/dict/record/callables, which therefore fail) and
// later entries overwrite earlier ones, the same last-wins rule as merging maps.
func ElementsToEntries(elems []Value) (map[string]Value, bool) {
	m := make(map[string]Value, len(elems))
	for _, e := range elems {
		if e.Type != value.Array {
			return nil, false
		}
		entry := (*Array)(e.Ptr).Elements
		if len(entry) != 2 {
			return nil, false
		}
		k, ok := entry[0].AsString()
		if !ok {
			return nil, false
		}
		m[k] = entry[1]
	}
	return m, true
}

// IsBlankElement reports whether e is "insignificant content" for a general container: undefined, or the
// element type's own zero value. Match-taking members called with NO argument act on this set (count() counts
// the significant elements, keep() answers exactly those, remove() drops the blanks, index() locates the first
// significant one). Each verb reads against the set the way its own name implies, which is why keep() and
// remove() land on the same answer with no argument: two different actions (keep the significant / remove the
// blank), not one operation under two names. It is a DEFAULT, not a policy — the argument forms override it at
// any call site, and a
// script that means "zeros are data" passes its own set. The text triple uses whitespace sets instead
// (text.IsBlankRune/text.IsBlankByte): these members are about separators and filler, and whitespace is text's filler.
func IsBlankElement(e Value) bool {
	switch e.Type {
	case value.Undefined:
		return true
	case value.Bool, value.Byte, value.Rune, value.Int:
		return e.Data == 0
	case value.Float:
		return math.Float64frombits(e.Data) == 0
	case value.Decimal:
		return (*dec128.Dec128)(e.Ptr).IsZero()
	case value.String:
		return len(*(*string)(e.Ptr)) == 0
	case value.Runes:
		return len((*Runes)(e.Ptr).Elements) == 0
	case value.Bytes:
		return len((*Bytes)(e.Ptr).Elements) == 0
	case value.Array:
		return len((*Array)(e.Ptr).Elements) == 0
	case value.Dict:
		return len((*Dict)(e.Ptr).Elements) == 0
	case value.Record:
		return len((*Record)(e.Ptr).Elements) == 0
	case value.Time:
		return isZeroTime(*(*time.Time)(e.Ptr))
	case value.IntRange:
		o := (*IntRange)(e.Ptr)
		return o.Start == o.Stop
	}
	// callables and error have no zero value and are never blank
	return false
}

// ConvMember implements the uniform conversion-member shape x.T([default]): zero or one argument; a failed
// conversion answers the default when one is supplied and raises otherwise. Every conversion member carries the
// slot — including identities and total conversions, where it can never fire — so generic code never has to
// special-case the receiver's type. The default is deliberately not type-checked: it is an explicit opt-out,
// the caller's responsibility. A second argument is refused with ErrArgCount, for the dispatcher to report.
func ConvMember(name string, from string, args []Value, ok bool, res Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, ErrArgCount
	}
	if ok {
		return res, nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return Undefined, errs.NewConversionError(from, name, "")
}

// joinElementsToString stringifies each element via AsString (the same coercion used by the `+` operator) and joins
// them with `sep`. A container element raises rather than passing through its element-wise text conversion — a
// nested collection has no canonical text, and joining one silently was the data-loss class this rule removes.
func joinElementsToString(elems []Value, sep string) (string, error) {
	if len(elems) == 0 {
		return "", nil
	}
	parts, total, err := joinElementParts(elems)
	if err != nil {
		return "", err
	}
	if len(elems) > 1 {
		total += (len(elems) - 1) * len(sep)
	}
	var b strings.Builder
	b.Grow(total)
	for i, p := range parts {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(p)
	}
	return b.String(), nil
}

// joinElementsToRunes is the runes join: the same acceptance as joinElementsToString, assembled as symbols — every
// element and the separator keep their own symbols across the seams (a runes element its elements, text decoded on
// its own), as runes + does.
func joinElementsToRunes(elems []Value, sep []rune) ([]rune, error) {
	if len(elems) == 0 {
		return []rune{}, nil
	}
	parts, _, err := joinElementParts(elems)
	if err != nil {
		return nil, err
	}
	var out []rune
	for i, p := range parts {
		if i > 0 {
			out = append(out, sep...)
		}
		if elems[i].Type == value.Runes {
			out = append(out, (*Runes)(elems[i].Ptr).Elements...)
		} else {
			out = append(out, text.DecodeText(p)...)
		}
	}
	if out == nil {
		out = []rune{}
	}
	return out, nil
}

// joinElementParts reads every element of a join as its text (AsString); a nested collection and a value with no
// text form raise. It answers the parts and their total octet length.
func joinElementParts(elems []Value) ([]string, int, error) {
	parts := make([]string, len(elems))
	total := 0
	for i, e := range elems {
		switch e.Type {
		case value.Array, value.Dict, value.Record, value.IntRange:
			return nil, 0, errs.NewConversionError(e.TypeName(), "string", "a nested collection is not renderable in join")
		}
		s, ok := e.AsString()
		if !ok {
			return nil, 0, errs.NewConversionError(e.TypeName(), "string", "")
		}
		parts[i] = s
		total += len(s)
	}
	return parts, total, nil
}

// coerceSepToBytes converts the separator argument of split/partition to a
// []byte. Accepted types: bytes, byte, string, rune.
func coerceSepToBytes(name string, sep Value) ([]byte, error) {
	switch sep.Type {
	case value.Bytes:
		return (*Bytes)(sep.Ptr).Elements, nil
	case value.Byte:
		return []byte{byte(sep.Data)}, nil
	case value.String:
		return []byte(*(*string)(sep.Ptr)), nil
	case value.Rune:
		return []byte(text.EncodeRuneText(rune(sep.Data))), nil
	default:
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "bytes, byte, string or rune", sep.TypeName())
	}
}

// PURE by contract
func defaultFormat(v Value, _ fspec.FormatSpec) (string, error) {
	return "", errs.NewNoFormattingError(v.TypeName())
}

// PURE by contract
func defaultUnaryOp(v Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// PURE by contract.
func defaultEqual(v Value, other Value, final bool) bool {
	// default to in-memory value equality
	if final {
		return v == other
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func defaultBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// CallMemberByLookup is the last step of every name path for x.name(args): the name may still be in the type's
// Methods table under an id the call site did not carry (bytecode compiled before the name was bound, a Go caller
// passing member.Unknown); otherwise the member does not exist. Every name switch ends in it, and it is the default
// CallNamedMethod, so the registry is consulted only on a miss — never on the id path or a name-switch hit.
// METHOD-DEPENDENT by contract: purity varies per member (see docs/purity.md)
func CallMemberByLookup(vm VM, v Value, name string, args []Value) (Value, error) {
	d := &ValueTypes[v.Type]
	if id := member.Lookup(name); id != member.Unknown && int(id) < len(d.Methods) && d.Methods[id].Fn != nil {
		return d.Methods[id].Call(vm, v, id, args)
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

// FormatWithSpec is the body of every format([spec]) member, which calls it from a one-line wrapper: it parses the
// spec and renders through the receiver type's own Format hook. id names the member for errors; a second argument
// is refused with ErrArgCount, for the dispatcher to report.
// PURE by contract
func FormatWithSpec(v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) > 1 {
		return Undefined, ErrArgCount
	}
	f := ""
	if len(args) == 1 {
		// the spec is text: string, runes or bytes — not anything that merely renders (format(5) raises)
		switch a := args[0]; a.Type {
		case value.String:
			f = *(*string)(a.Ptr)
		case value.Runes:
			f = text.EncodeText((*Runes)(a.Ptr).Elements)
		case value.Bytes:
			f = string((*Bytes)(a.Ptr).Elements)
		default:
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes or bytes", a.TypeName())
		}
	}
	sp, err := fspec.Parse(f)
	if err != nil {
		return Undefined, errs.FromFormatSpecError(name, err)
	}
	s, err := ValueTypes[v.Type].Format(v, sp)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(s), nil
}

// PropertyByLookup is the last step of every name path for x.name — the mirror of Value.AccessProperty for a
// name the access site carried no id for: the type's Properties[id].Get, else a bound member for Methods[id].
// false is the miss; each type keeps its own miss error.
func PropertyByLookup(vm VM, v Value, name string) (Value, bool, error) {
	d := &ValueTypes[v.Type]
	id := member.Lookup(name)
	if id == member.Unknown {
		return Undefined, false, nil
	}
	if int(id) < len(d.Properties) && d.Properties[id].Get != nil {
		r, err := d.Properties[id].Get(vm, v, id)
		return r, true, err
	}
	if int(id) < len(d.Methods) && d.Methods[id].Fn != nil {
		return NewBoundMemberValue(v, id, name), true, nil
	}
	return Undefined, false, nil
}

// AssignPropertyByLookup is the last step of every name path for x.name = r: the type's Properties[id].Set for a
// name the access site carried no id for. false is the miss; each type keeps its own miss error.
func AssignPropertyByLookup(vm VM, v Value, name string, r Value) (bool, error) {
	d := &ValueTypes[v.Type]
	if id := member.Lookup(name); id != member.Unknown && int(id) < len(d.Properties) && d.Properties[id].Set != nil {
		return true, d.Properties[id].Set(vm, v, id, r)
	}
	return false, nil
}

// MUTATE-DEPENDENT by contract (see ValueTypeDescr.Delete)
func defaultDelete(v Value, _ Value, _ bool) (Value, error) {
	return Undefined, errs.NewNotDeletableError(v.TypeName())
}

// PURE by contract
func defaultAccessNamedProperty(vm VM, v Value, name string) (Value, error) {
	if r, ok, err := PropertyByLookup(vm, v, name); ok {
		return r, err
	}
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

// IMPURE by contract (mutates target)
func defaultAssignNamedProperty(vm VM, v Value, name string, r Value) error {
	if ok, err := AssignPropertyByLookup(vm, v, name, r); ok {
		return err
	}
	return errs.NewNotAssignableError(v.TypeName())
}

// PURE by contract — x.name on a type whose only access is by index: names the property, quoted
func noNamedProperty(vm VM, v Value, name string) (Value, error) {
	if r, ok, err := PropertyByLookup(vm, v, name); ok {
		return r, err
	}
	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), NewStringValue(name).String())
}

// PURE by contract
func defaultAccessIndex(v Value, _ Value) (Value, error) {
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

// MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append) — this type supports neither form.
func defaultAppend(v Value, _ []Value, _ bool) (Value, error) {
	return Undefined, errs.NewNotAppendableError(v.TypeName())
}

// PURE by contract
func defaultSlice(v Value, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

// PURE by contract
func defaultSliceStep(v Value, _, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

func defaultCall(_ VM, v Value, _ []Value) (Value, error) {
	return Undefined, errs.NewNotCallableError(v.TypeName())
}

// PURE by contract
func defaultAsRunes(v Value) ([]rune, bool) {
	s, ok := v.AsString()
	if !ok {
		return nil, false
	}
	return text.DecodeText(s), true
}

// ---------------------------------------------------------------------------
// The casing family: one word segmenter, five renderings. Symbol-class
// operations, so they live on string/runes only — never on bytes.
// ---------------------------------------------------------------------------

// The case renderings come in two policies: the identifier renderings (snake/kebab/camel/pascal) NORMALISE the
// interior — an identifier has a canonical case — while the label rendering (title_case) PRESERVES it, uppercasing
// only each word's first symbol ("ATM fee" → "ATM Fee" as a title, "atm_fee" as an identifier).

// ---------------------------------------------------------------------------
// Operator-layer text operands. The receiver — the LEFT operand — decides the
// result type; acceptance mirrors the member layer with one deliberate cut:
// an int is never text in operator position, so the arithmetic readings stay
// unambiguous. ok=false means the operand is not text content at all (the
// caller answers with its invalid-operator error); a data-range failure (a
// non-ASCII octet into symbols, invalid UTF-8) raises directly.
// ---------------------------------------------------------------------------

// textOperandString reads an operand as SYMBOL content (string/runes receivers).
func textOperandString(a Value) (string, bool, error) {
	switch a.Type {
	case value.String:
		return *(*string)(a.Ptr), true, nil
	case value.Runes:
		return text.EncodeText((*Runes)(a.Ptr).Elements), true, nil
	case value.Bytes:
		// TOTAL: an undecodable octet is carried through as itself; a string holds octets
		return string((*Bytes)(a.Ptr).Elements), true, nil
	case value.Rune:
		return text.EncodeRuneText(rune(a.Data)), true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return "", false, errs.NewInvalidValueError(fmt.Sprintf("an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", a.Data))
		}
		return text.EncodeRuneText(rune(a.Data)), true, nil
	}
	return "", false, nil
}

// textOperandRunes reads an operand as a run of SYMBOLS — the reading of the member that the operator spells
// (append, remove): a runes operand keeps its own symbols, text decodes on its own, never fused with the receiver.
// Acceptance and refusals are textOperandString's.
func textOperandRunes(a Value) ([]rune, bool, error) {
	switch a.Type {
	case value.String:
		return text.DecodeText(*(*string)(a.Ptr)), true, nil
	case value.Runes:
		return (*Runes)(a.Ptr).Elements, true, nil
	case value.Bytes:
		return text.DecodeOctets((*Bytes)(a.Ptr).Elements), true, nil
	case value.Rune:
		return []rune{rune(a.Data)}, true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", a.Data))
		}
		return []rune{rune(a.Data)}, true, nil
	}
	return nil, false, nil
}

// textOperandOctets reads an operand as OCTET content (bytes receivers) — encoding is total, so every text
// operand is accepted; symbols contribute their UTF-8 encoding.
func textOperandOctets(a Value) ([]byte, bool, error) {
	switch a.Type {
	case value.Bytes:
		return (*Bytes)(a.Ptr).Elements, true, nil
	case value.String:
		return []byte(*(*string)(a.Ptr)), true, nil
	case value.Runes:
		return []byte(text.EncodeText((*Runes)(a.Ptr).Elements)), true, nil
	case value.Rune:
		return []byte(text.EncodeRuneText(rune(a.Data))), true, nil
	case value.Byte:
		return []byte{byte(a.Data)}, true, nil
	}
	return nil, false, nil
}

// mapContainsKey is the `in` operator on a map receiver: the KEY axis — a key is accepted iff its
// own string conversion exists (the d[k] rule); a submap operand raises as deferred, a callable
// raises, anything unconvertible raises — never a silent false.
func mapContainsKey(m map[string]Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	if e.Type == value.Dict || e.Type == value.Record {
		return false, errs.NewNotImplementedError("(in) the submap reading is deferred; match keys, or compare entries with a predicate")
	}
	s, ok := e.AsString()
	if !ok {
		return false, errs.NewInvalidArgumentTypeError("in", "operand", "a key (string)", e.TypeName())
	}
	_, hit := m[s]
	return hit, nil
}

// roundingMode is one of the seven rounding modes a script names. The names are Python's `decimal` module's,
// lowercased and without the ROUND_ prefix, because that is the reference a reader brings: `down` is toward zero
// and `up` away from it, while `floor` / `ceiling` are the directions toward −∞ / +∞. A mode is a string argument
// because in a real application it is configuration, not code; the round_<mode>(n) twins spell the same names as
// suffixes for the common case where it is fixed. decimal rounds with dec128 (roundingMode.dec128); float rounds
// its shortest decimal text with roundsUp — one table, two engines, the same seven readings.
type roundingMode uint8

const (
	roundCeiling roundingMode = iota
	roundFloor
	roundDown
	roundUp
	roundHalfDown
	roundHalfUp
	roundHalfEven
)

var roundingModes = map[string]roundingMode{
	"ceiling":   roundCeiling,
	"floor":     roundFloor,
	"down":      roundDown,
	"up":        roundUp,
	"half_down": roundHalfDown,
	"half_up":   roundHalfUp,
	"half_even": roundHalfEven,
}

// roundingModeNames lists the accepted names in the order the docs present them, for the error message.
const roundingModeNames = "ceiling, floor, down, up, half_down, half_up, half_even"

// dec128 answers the dec128 mode with the same reading. dec128's ROUND_NAN has no name on purpose: refusing a loss
// is rescale(n)'s job, and it is meaningless for the division-shaped operations, which lose digits by nature.
func (m roundingMode) dec128() dec128.RoundingMode {
	switch m {
	case roundCeiling:
		return dec128.ROUND_UP
	case roundFloor:
		return dec128.ROUND_DOWN
	case roundDown:
		return dec128.ROUND_TOWARD_ZERO
	case roundUp:
		return dec128.ROUND_AWAY_FROM_ZERO
	case roundHalfDown:
		return dec128.ROUND_HALF_TOWARD_ZERO
	case roundHalfUp:
		return dec128.ROUND_HALF_AWAY_FROM_ZERO
	default:
		return dec128.ROUND_BANK
	}
}

// roundsUp is the rounding decision on a magnitude that lost digits: whether its last kept unit goes up by one.
// negative is the value's sign, odd whether the last kept digit is odd, half how the dropped part compares with
// half a unit (-1 below, 0 exactly, +1 above). The caller has already established that something was dropped.
func (m roundingMode) roundsUp(negative, odd bool, half int) bool {
	switch m {
	case roundCeiling:
		return !negative // toward +∞
	case roundFloor:
		return negative // toward −∞
	case roundDown:
		return false // toward zero
	case roundUp:
		return true // away from zero
	case roundHalfDown:
		return half > 0
	case roundHalfUp:
		return half >= 0
	default: // half_even
		return half > 0 || (half == 0 && odd)
	}
}

// roundingModeArg reads a rounding-mode argument: a string naming one of the seven modes, exactly. An unknown name
// raises and lists the valid ones — a mode read from configuration must fail loudly, never fall back to a default.
//
// PURE by contract.
func roundingModeArg(name, pos string, a Value) (roundingMode, error) {
	if a.Type != value.String {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "string", a.TypeName())
	}
	s, _ := a.AsString()
	mode, ok := roundingModes[s]
	if !ok {
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) unknown rounding mode %q, expected one of: %s", name, s, roundingModeNames))
	}
	return mode, nil
}

// RenderFormat is format() for a value whose render is its String hook: the generic spec over the render, and
// {:T} the type name. PURE by contract.
func RenderFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	return fspec.ApplyGenerics(ValueTypes[v.Type].String(v), sp, fspec.AlignLeft), nil
}

// SamePtrEqual is identity equality for a handle type: the same type and the same Ptr. PURE by contract.
func SamePtrEqual(v Value, other Value, final bool) bool {
	if other.Type == v.Type {
		return other.Ptr == v.Ptr
	}
	if final {
		return false
	}
	return ValueTypes[other.Type].Equal(other, v, true)
}

// --- durations. An int counts nanoseconds when its member says so: int and float build one (30.seconds()), an int
// reads one in a unit (d.in_hours()), a string parses one ("1h30m".nanoseconds()). Each member is its own function
// handing its unit to these helpers.

const (
	day  = 24 * time.Hour
	week = 7 * day
)

// durationFromInt is n units in nanoseconds; int is checked, so a duration past its range raises.
func durationFromInt(id member.ID, n int64, unit time.Duration) (Value, error) {
	ns, ok := IntMulChecked(n, int64(unit))
	if !ok {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the duration overflows int", id.String()))
	}
	return IntValue(ns), nil
}

// durationFromFloat is f units rounded to the nearest nanosecond; NaN, ±Inf and a result past int's range raise.
func durationFromFloat(id member.ID, f float64, unit time.Duration) (Value, error) {
	ns := math.Round(f * float64(unit))
	if math.IsNaN(ns) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the duration is not a number", id.String()))
	}
	if ns < math.MinInt64 || ns >= math.MaxInt64 {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("(%s) the duration overflows int", id.String()))
	}
	return IntValue(int64(ns)), nil
}

// durationIn reads a duration of ns nanoseconds in a unit, as a float.
func durationIn(id member.ID, ns int64, unit time.Duration) (Value, error) {
	return FloatValue(float64(ns) / float64(unit)), nil
}
