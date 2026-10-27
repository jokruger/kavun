package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
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

// Immutable Unicode text.
//
// ## Overview
//
// A `string` is a sequence of **symbols** (Unicode code points). The symbol is the element everywhere on the
// type: `len()` counts symbols, `s[i]` yields a `rune`, `s[i:j]` slices at symbol boundaries, and every member
// offset — `index()`, `slice()`, `splice()`, `insert()` — is symbol-based.
//
// ```go
// "héllo".len()        // => 5 — symbols, not bytes
// "héllo"[1]           // => 'é' — a rune
// "héllo"[1:3]         // => "él"
// "abc"[-1]            // => 'c' — negative indices count from the end
// "кавун".len()        // => 5
// ```
//
// Symbols are code points, **not** grapheme clusters. A flag emoji is two regional-indicator symbols; a
// combining sequence is a base symbol plus combining marks. What a reader perceives as one character can be
// several symbols:
//
// ```go
// "🇺🇦".len()          // => 2 — two regional indicators
// "e\u0301".len()     // => 2 — é written as e + U+0301 (combining acute)
// ```
//
// **Storage and cost.** A `string` is stored as compact UTF-8 with a cached symbol count and an ASCII fast
// path: indexing and slicing are O(1) on pure-ASCII strings and O(n) worst case on multibyte content. For heavy
// positional work on non-ASCII text, convert to [`runes`](runes.md) first — `runes` is the O(1)-indexing,
// materialized form of the same text.
// Members whose answer depends on the ends of the text or on the argument alone — `first`/`last`, the anchored pair
// and their `remove_` twins, the `trim` family, the pads, and `contains`/`count`/`index`/`index_last`/`replace` —
// read the text in place and never decode the whole receiver; the members that visit or rebuild every symbol decode
// it once per call.
//
// **Immutability.** A `string` is immutable by construction. It is the only sequence type with no mutating
// member at all: no `_in_place` twins, and no `slice_view`/`chunk_view` (on an immutable value sharing is
// unobservable, so `slice` *is* the view). Element assignment raises:
//
// ```go
// s := "abc"
// s[0] = 'x'           // Error: not_assignable: type string does not support assignment
// ```
//
// A `string` literal is a compile-time constant as well: its content is fixed in the source, so the compiler
// stores it once in the bytecode's static pool and every evaluation loads that one shared value. For `string`
// this is invisible — the type has no mutating member either way — but it is the same mechanism that makes
// `b"..."` and `u"..."` immutable *constants* of otherwise mutable types, and the reason all three short forms
// exist: text written down in a program is usually constant text. See [Constant literals and constructed
// literals](../../language.md#constant-literals-and-constructed-literals); when you want a mutable buffer of the
// same content, construct one — `runes("abc")` or `bytes("abc")`.
//
// **`string` and `runes` are semantically interchangeable** — the same text, the same member surface, the same
// operator behavior, and they compare equal on equal content (`"ab" == u"ab"` is `true`). The difference is
// storage (UTF-8 vs. a symbol array) and mutability. Choose `string` by default; choose `runes` when you need
// in-place mutation or guaranteed O(1) indexing.
//
// Every transforming member returns a new `string`; partitioning members (`split`, `partition`, `split_lines`,
// `chunk`) return an `array` of strings.
//
// The edge members `first([d])`, `last([d])`, `min([d])` and `max([d])` answer a `rune`; an empty string has no edge
// to give, so without the optional trailing default they raise `invalid_value`:
//
// ```go
// "abc".first()        // => 'a'
// "".first()           // Error: invalid_value: (first) empty sequence
// "".first('?')        // => '?'
// "banana".min()       // => 'a'
// "banana".max()       // => 'n'
// ```
//
// The parsing conversions (`int`, `float`, `decimal`, `bool`, `time`, `date`, `nanoseconds`) follow the uniform
// failure policy: a valid result or a catchable raise, and with the optional trailing default, the default when the
// text does not parse. A malformed `time`/`date` layout is the caller's error, not the data's, and raises even with a
// default. The re-representations `runes()`, `bytes()` and `array()` are total — they cannot fail — so they take no
// default.
//
// ## Literals
//
// ```go
// s := "hello"
// esc := "line1\nline2 \"quoted\" \u00a0"   // escape sequences
// raw := r"C:\Users\Bob \d+"                 // raw string: backslashes literal
// uni := u"кавун"                             // a runes literal — same text, runes type
// ```
//
// The free `string(x)` constructor is the conversion — the free spelling of `x.string()`. The two-argument
// `string(x, n)` is the count form: the conversion repeated `n` times, `string(x).repeat(n)`:
//
// ```go
// string()             // => ""
// string("ab", 2)      // => "abab" — the count form: string(x).repeat(n)
// string('x', 3)       // => "xxx"
// string(65, 2)        // => "6565" — the conversion first: string(65) is "65"
// ```
//
// F-strings interpolate expressions and format specs into a `string`: `f"n={n:5d}"`. See
// [f-strings](../../f_strings.md) and the [format mini-language](../../format_mini_language.md).
//
// ## Operators
//
// | operator | meaning |
// | --- | --- |
// | `+` | concatenation of text content |
// | `-` | remove every occurrence of the run (leftmost, non-overlapping), read as symbols as `remove` reads it — the octets of an escape never match inside a symbol; acceptance equals `+`'s |
// | `*` | exactly `repeat(n)`: the right operand is a **count**, not text. No reflected direction — `n * s` raises |
// | `==` `!=` | content equality; `string` and `runes` with the same text are equal |
// | `<` `<=` `>` `>=` | lexicographic by symbol |
// | `x in s` | membership — runs `contains`' value readings |
// | `s[i]` | the symbol at `i`, as a `rune` |
// | `s[i:j]` | symbol-boundary slice, a `string` |
//
// Both operands must carry **text content**. The **receiver (left operand) decides the result type**; a scalar
// on the left takes the sequence's type:
//
// ```go
// "ab" + "cd"          // => "abcd"
// "ab" + u"cd"         // => "abcd" — string — the left operand decides
// u"ab" + "cd"         // => u"abcd" — runes on the left
// "ab" + bytes("cd")   // => "abcd" — the bytes decode; the decode never fails
// b'a' + "bc"          // => "abc" — scalar left takes the sequence's type
// 'é' + "bc"           // => "ébc"
// "abc" + b'\xff'      // Error: invalid_value: a byte is a symbol only in ASCII (0x00–0x7F)
// "ab" + 1             // Error: invalid_binary_operator: operators never take int — arithmetic keeps its reading
// "ab" + 1.5           // Error: invalid_binary_operator
// "ab" + true          // Error: invalid_binary_operator
// "ab" + ["cd"]        // => ["ab", "cd"] — string declines; the array prepends it as one element
// "ab" - ["cd"]        // Error: invalid_binary_operator: string - array (there is no reflected form for removal)
// ```
//
// `-` removes runs or single symbols, never set-difference:
//
// ```go
// "banana" - "na"      // => "ba" — every occurrence, leftmost non-overlapping
// "banana" - 'a'       // => "bnn"
// "banana" - 97        // Error: invalid_binary_operator: operators never take int
//
// "-" * 20             // => "--------------------" — the count form of repeat
// "ab" * 2.0           // => "abab" — a whole-valued count converts; 1.5 raises
// 2 * "ab"             // Error: invalid_binary_operator: int * string   (no reflected direction)
// ```
//
// `in` accepts exactly the values `contains` accepts and raises on anything else — a callable raises, because
// an operator operand is always a value, never a predicate:
//
// ```go
// 'a' in "abc"                       // => true
// "bc" in "abc"                      // => true
// 97 in "abc"                        // => true — 'a' — members read a valid int as the symbol
// 1.5 in "abc"                       // Error: invalid_argument_type
// (func(c) { return true }) in "abc" // Error: invalid_value — the predicate spelling is contains(f)
// ```
//
// Comparisons are lexicographic by symbol, not by length:
//
// ```go
// "abc" < "abd"        // => true
// "b" > "aaaa"         // => true
// ```
//
// ## Argument dispatch
//
// Every member and operator on `string` reads its arguments the same way. An argument is accepted iff it is
// **text content encoded into the receiver's representation**:
//
// | argument | reading |
// | --- | --- |
// | `string` / `runes` | text, verbatim — a contiguous **run** |
// | `bytes` | decoded as UTF-8; **total** — an octet that is not a symbol becomes its [escape](#undecodable-octets) |
// | `rune` | itself — one symbol |
// | `byte` | one symbol iff ASCII (`0x00`–`0x7F`); beyond raises |
// | `int` (members only) | one symbol iff a valid code point; **operators never take `int`** |
// | `bool` / `float` / `decimal` | raise — a number as text is ambiguous (`65` could be `"65"` or `'A'`) |
// | `int` (`*` only) | a repeat **count** — `s * n` is exactly `s.repeat(n)`; there is no reflected `n * s` |
// | `array` / `range` / `dict` / callables / `undefined` | raise — the text family is closed. One exception is not `string`'s doing: `"ab" + [1, 2]` → `["ab", 1, 2]`, because a left operand that declines hands `+` to the array, which prepends it as one element (see [array](array.md)) |
//
// The match members (`contains`, `count`, `index`, `index_last`, `keep`, `remove`, `any`, `all`, `split`, …)
// share one argument menu:
//
// - **no argument** — the [blank set](#the-blank-set);
// - **a function** — a predicate: it gets the element, or `(index, element)` when it does not accept one argument
// ([callbacks](#callbacks));
// - **`string`/`runes`/`bytes`** — a contiguous run;
// - **`rune`/`byte`/valid `int`** — one element;
// - **variadic** — a homogeneous set: all elements or all runs. Mixing the two classes raises, and a function
// among several arguments raises.
//
// ```go
// "abc".contains('x', 'b')                       // => true — element set
// "abc".count(func(i, c) { return i > 0 })       // => 2 — accepts two, not one: gets (index, element)
// "abcd".remove("ab", "abc")                     // => "d" — run set; at a tie the longest run wins
// "banana".remove("na", 'b')                     // Error: invalid_argument_type: mixed element and run classes
// "abc".remove("a", func(c) { return true })     // Error: invalid_argument_type: a function among several arguments
// ```
//
// The one exception is the **add side** (`append`/`prepend`), where operands mix freely in order:
// `x.append("ab", 'c')` ≡ `x + "ab" + 'c'`.
//
// Run matching is leftmost and non-overlapping: `"aaa".count("aa")` is `1`.
//
// ## Undecodable octets
//
// Text that is not valid UTF-8 is data too — a file off disk, a legacy export, a truncated frame. So **every
// conversion among `byte`, `bytes`, `string` and `runes` is total and lossless**: nothing raises, nothing is
// substituted, and the octets you started with are the octets you get back.
//
// An octet no decoder can read as a symbol becomes its own reserved rune — **U+DC80–U+DCFF, one per octet
// 0x80–0xFF** — and encoding turns that rune straight back into the octet it came from.
//
// ```go
// b = bytes([0x61, 0xFF, 0x62])   // 'a', a stray octet, 'b'
// s = b.string()                  // no raise
// s.len()                         // => 3 — the octet is one symbol
// s[1].int()                      // => 56575 — U+DCFF, the escape for 0xFF
// s.bytes() == b                  // => true
// s.runes().bytes() == b          // => true — through runes and back, unchanged
// ```
//
// Asking, and repairing:
//
// ```go
// s := bytes([0x61, 0xFF, 0x62]).string()
// s.is_valid()                    // => false — some symbol is an escape
// "abc".is_valid()                // => true
// s[1].is_valid()                 // => false — this one
// s[1].byte()                     // => byte(255) — the escape converts back to its octet
// s.replace(s[1], '?')            // => "a?b"
// s.keep(c => c.is_valid())     // => "ab"
// ```
//
// **Why this range, and why it cannot go wrong.** Only an octet ≥ 0x80 can ever be undecodable — an ASCII octet
// is always a symbol — so 128 values suffice. Every undecodable octet decodes with width 1, so *n* bad octets
// always become *n* escapes. And U+DC80–U+DCFF are low surrogates: never Unicode scalar values, so well-formed
// text can never produce one and an escape can never be mistaken for content. (The scheme is Python's
// `surrogateescape`, PEP 383; Rust's WTF-8 answers the same problem the same way.)
//
// **A string is its octets, so building one joins octets.** Every member and operator that builds a string — `+`,
// `append`, `prepend`, `push`, `replace`, `pad_start`/`pad_end`, f-strings — concatenates the octets, and the result is
// read afresh: escapes whose octets meet to complete a sequence read as the one symbol they spell.
// [`runes`](runes.md) holds symbols, so the same building keeps them apart; the octets are the same either
// way.
//
// ```go
// a = bytes([0xE2, 0x82]).string()  // two escapes: the first two octets of "₩"
// b = bytes([0xA9]).string()        // one escape: its last octet
// (a + b).len()                     // => 1 — the octets meet and read "₩"
// a + b == "₩"                      // => true
// a.runes().append(b).len()         // => 3 — runes keeps the three escapes
// ```
//
// **The two places this is visible elsewhere.** A [`rune`](rune.md)'s domain is exactly what can be encoded
// — scalar values plus the 128 escapes — so `rune(0xDCFF)` exists while `rune(0xD800)` raises where it is written, and
// no conversion *out* of `rune`/`runes` can fail. And the boundary **out** of the language is not total: `json.encode`
// raises on text holding escapes, because JSON text is UTF-8 by definition. Encode the `bytes` instead (they go as
// base64), or repair the text first.
//
// ## The blank set
//
// The no-argument form of the match members acts on the type's **blank set** — its notion of insignificant
// content. For `string` that is **NUL ∪ Unicode whitespace** (the Unicode `White_Space` class): this is what
// no-arg `trim()`, `split()`, `partition()`, `index()`, `count()`, `keep()`, and `remove()` act on, and the
// default `pad_start`/`pad_end` fill is the set's canonical member, the space.
//
// Being the Unicode class, it includes the non-breaking space and every other Unicode space — not just ASCII
// whitespace:
//
// ```go
// "\u00a0x".trim()     // => "x" — the NBSP is blank
// "\x00x\x00".trim()   // => "x" — NUL is blank
// ```
//
// `bytes` projects the same notion into its element domain (the ASCII subset — all the whitespace an octet can
// express); the sets agree wherever the domains overlap.
//
// ## Callbacks
//
// A member that takes a function — the predicate reading of the match members, `map`, `flat_map`, `for_each`, `reduce`
// — calls it in the **fewest form it accepts**, read from the callable's arity and whether it is variadic, never from
// what kind of callable it is:
//
// - a per-element callback gets the element when it accepts one argument, else `(index, element)`; anything else
// raises;
// - a `reduce` callback gets `(acc, element)` when it accepts two arguments, else `(acc, index, element)`; anything
// else raises.
//
// So `func(x, ...rest)` and `func(...args)` get the element, and a builtin or a bound member is a callback like any
// other (`min` and `"abc".contains` accept one argument). A lambda is the one way to call a callable in another form —
// choose the form, reorder, fix or drop arguments:
//
// ```go
// "abc".count(func(i, c) { return i > 0 })               // => 2 — accepts two, not one
// "abc".index(func(c, ...rest) { return c == 'b' })     // => 1 — accepts one: the element
// "abcd".keep("bd".contains)                            // => "bd"
// "abc".reduce("", (s, c) => c + s)                      // => "cba"
// ```
//
//kavun:type string

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

	Methods: stringMethods,
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
