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
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/internal/seq"
	"github.com/jokruger/kavun/internal/text"
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

// Raw octets — binary data that can also be treated as ASCII-range text.
//
// ## Overview
//
// `bytes` is the low-level member of the text family: a mutable sequence of **octets** with no symbol
// interpretation. The element is the [`byte`](byte.md).
//
// | type | representation | mutability | indexing |
// | --- | --- | --- | --- |
// | `bytes` | raw octets, no symbol interpretation | **mutable** | O(1) by octet |
// | `string` | Unicode text in compact UTF-8 | immutable | O(n) by symbol — see [string](string.md) |
// | `runes` | Unicode text as a symbol array | mutable | O(1) by symbol — see [runes](runes.md) |
//
// The line between `bytes` and the symbol types is **encoding**. Members that need symbol classes — `upper`,
// `lower`, `case_fold`, the casing family — do not exist on `bytes`: interpreting an octet as a cased letter
// means assuming an encoding.
//
// ```go
// bytes("héllo").len()                // => 6 — octets — é is two
// bytes("héllo").upper()              // Error: invalid_method: type bytes has no method upper
// bytes("héllo").string().upper()     // => "HÉLLO" — decode, then the symbol work
// ```
//
// The edge members `first([d])`, `last([d])`, `min([d])` and `max([d])` take the optional trailing default; an
// empty receiver has no answer to give, so without one they raise `invalid_value`. No `sum`/`avg` — see the
// exclusions.
//
// ```go
// bytes("bca").first()        // => byte(98)
// bytes("bca").min()          // => byte(97)
// bytes("").first(b'?')       // => byte(63)
// bytes("").first()           // Error: invalid_value: (first) empty sequence
// ```
//
// Decoding is total too: `bytes` → `string`/`runes` never raises — an octet that is not part of valid UTF-8 rides
// through as its escape rune (U+DC80–U+DCFF) and converts back to the same octet.
//
// There is no `.int()`/`.float()`/`.decimal()`/`.bool()`/`.time()` on `bytes` — octets are not text until
// decoded. Parse through the decode: `bytes("42").string().int()` → `42`. There are no `.dict()`/`.record()`
// conversions either (octets are never key/value entries), and no `.byte()` (`bytes([65])` is a sequence;
// its element is `b[0]`).
//
// ## Literals and construction
//
// The literal form is `b"..."`, with the same escapes as `string` literals plus raw octet escapes:
//
// ```go
// b"ab"               // => bytes([97, 98])
// b"\x00\xff"         // => bytes([0, 255])
// ```
//
// **`bytes` is a mutable sequence — but a literal is an immutable constant.**:
// the *type* is writable, and `b"..."` is a compile-time constant of that type. Its content
// is fixed in the source, so the compiler stores it once in the bytecode's static pool and every evaluation
// loads that one shared value; `bytes` is most often constant text or a fixed byte pattern — a magic prefix,
// a separator, a key — which is what the short `b"..."` form is for. When you want a buffer, you construct one:
//
// ```go
// type_name(b"ab")            // => "immutable-bytes"
// l := b"abc"
// l[0] = b'X'                 // Error: not_assignable: type immutable-bytes does not support assignment ...
//
// m := bytes(b"abc")          // a constructor always builds a new, writable body
// m[0] = b'X'
// m                           // => bytes([88, 98, 99]) — the literal is untouched
// m = b"abc".copy()           // the same thing, spelled as a copy
// m = bytes("abc")            // or build it from a string
// ```
//
// `bytes(...)` is both the conversion (one argument) and the count constructor (two arguments): `bytes(x, n)` is
// `bytes(x)` repeated `n` times, so `bytes(b'\x00', n)` is the preallocation spelling:
//
// ```go
// bytes("héllo")          // => bytes([104, 195, 169, 108, 108, 111]) — UTF-8 octets of the text
// bytes([104, 105])       // => bytes([104, 105]) — an array of in-range ints or bytes, element-wise
// bytes([b'h', b'i'])     // => bytes([104, 105])
// bytes([300])            // Error: conversion — not an octet
// bytes(b'\x07')          // => bytes([7]) — a byte is one octet — any octet
// bytes('8')              // => bytes([56]) — a rune contributes its UTF-8 octets — not the digit 8
// bytes(5)                // Error: conversion: cannot convert int to bytes: no conversion exists
// bytes("ab", 2)          // => bytes([97, 98, 97, 98]) — the count form: bytes(x).repeat(n)
// bytes(b'\x00', 3)       // => bytes([0, 0, 0]) — the preallocation: n explicit fill octets
// ```
//
// Encoding is total — every text-ish value has UTF-8 octets — so `string`/`runes` → `bytes` never fails.
// Non-text scalars have no direct octet reading: `(3).bytes()` does not exist — go through the text,
// `(3).string().bytes()`.
//
// ## Indexing and slicing
//
// All positions are octet positions. Negative indices count from the end; slices clamp. Slicing can split a
// multi-octet symbol — `bytes` does not know or care:
//
// ```go
// bytes("héllo")[1]           // => byte(195) — the first octet of é
// bytes("abc")[-1]            // => byte(99)
// bytes("héllo").slice(0, 3)  // => bytes([104, 195, 169])
// ```
//
// Element writes (`m[i] = b'x'`) mutate the shared body and are visible through every alias; on a frozen or
// literal value they raise.
//
// ## Operators
//
// | operator | meaning |
// | --- | --- |
// | `+` | concatenation; the **receiver** (left operand) decides the result type |
// | `-` | removes **every** occurrence of the right operand (leftmost, non-overlapping); acceptance equals `+`'s |
// | `*` | exactly `repeat(n)`: the right operand is a **count**, not content. No reflected direction — `n * b` raises |
// | `==` `!=` `<` `<=` `>` `>=` | content-based, octet by octet — compares across `string`/`runes`/`bytes` by text content |
// | `in` | membership: `x in b` uses `contains`' value readings (element or run); raises on an unacceptable operand — never a silent `false` |
//
// Operand acceptance — every accepted operand is text content, encoded into octets. Encoding is total, so
// `bytes` accepts every text-ish operand unconditionally:
//
// | right operand | reading |
// | --- | --- |
// | `bytes` | its octets, verbatim |
// | `string` / `runes` | its UTF-8 octets |
// | `rune` | its 1–4 UTF-8 octets |
// | `byte` | that octet |
// | `int` | **raises** — arithmetic keeps the numeric reading of `+`/`-`; the member forms (`push`, `append`) take an in-range int |
// | `bool` / `float` / `decimal` | raises — ambiguous as text |
// | `int` (`*` only) | a repeat **count** — `b * n` is exactly `b.repeat(n)`; there is no reflected `n * b` |
// | `array` / `range` / `dict` / callables | raises — the text types accept only text content. `bytes("ab") + [1, 2]` → `[bytes([97, 98]), 1, 2]` is not an exception to that: `bytes` declines and the array answers `+` by prepending it (see [array](array.md)) |
//
// ```go
// bytes("ab") + "cd"          // => bytes([97, 98, 99, 100]) — bytes receiver → bytes
// bytes("ab") + u"cd"         // => bytes([97, 98, 99, 100])
// bytes("ab") + 'é'           // => bytes([97, 98, 195, 169]) — a rune contributes its UTF-8 octets
// bytes("ab") + b'\xff'       // => bytes([97, 98, 255]) — any octet — no ASCII limit on this side
// bytes("ab") + 98            // Error: invalid_binary_operator: bytes + int
// bytes("ab") + [1]           // => [bytes([97, 98]), 1] — bytes declines; the array prepends it
// bytes("ab") - [1]           // Error: invalid_binary_operator: bytes - array
// "ab" + bytes("cd")          // => "abcd" — string receiver → decodes the bytes
// "ab" + bytes([255])         // => "ab\xff" — the octet is carried through as its escape
// b'a' + bytes("bc")          // => bytes([97, 98, 99]) — a scalar on the left takes the sequence's type
//
// bytes("abcabc") - "bc"      // => bytes([97, 97]) — every occurrence
// bytes("banana") - b'a'      // => bytes([98, 110, 110])
// bytes("héllo") - 'é'        // => bytes([104, 108, 108, 111]) — removes the 2-octet run
// bytes("abc") - 98           // Error: invalid_binary_operator: bytes - int
//
// bytes("ab") * 3             // => bytes([97, 98, 97, 98, 97, 98]) — the count form of repeat
// b"ab" * 2                   // => bytes([97, 98, 97, 98]) — a literal answers a new, mutable value
// 3 * bytes("ab")             // Error: invalid_binary_operator: int * bytes   (no reflected direction)
// bytes("ab") * b'\x03'       // Error: invalid_binary_operator: bytes * byte   (the operand is a count, not content)
//
// bytes("ab") == "ab"         // => true
// bytes("ab") == u"ab"        // => true
// bytes("ab") < bytes("b")    // => true
//
// b'a' in bytes("ab")         // => true — element
// "bc" in bytes("abc")        // => true — run of octets
// 'é' in bytes("héllo")       // => true — the rune's 2-octet run
// 97 in bytes("abc")          // => true — an in-range int reads as one octet
// 300 in bytes("abc")         // Error: invalid_value — not an octet — never a silent false
// ```
//
// A callable on the right of `in` raises: an operator operand is always a value — the predicate reading is
// spelled `contains(f)` / `any(f)`.
//
// ## Argument dispatch — the argument's type selects the reading
//
// Every member that matches or adds content reads its arguments from one menu:
//
// | argument | reading |
// | --- | --- |
// | absent | the **blank set** (below) |
// | function | predicate — it gets the element, or `(index, element)` when it does not accept one argument ([callbacks](#callbacks)) |
// | `string` / `runes` / `bytes` | a **run** — a contiguous octet subsequence (text contributes its UTF-8 octets) |
// | `rune` / `byte` / `int` | one **element** where it fits: a byte always; an int in `0`–`255` (members only — operators refuse int); a rune only if its encoding is one octet (ASCII) |
// | several arguments (match side) | a **set** — "∈ the set"; homogeneous: all elements or all runs, mixing raises |
// | several arguments (add side: `append`/`prepend`) | operands in order — mixing freely is the point |
//
// A multi-octet rune is still fine wherever a **run** is accepted (`append`, `contains`, `replace`, ...) — it
// reads as its 1–4 octets; only the strictly one-element slots (`push`, `insert` items, pad fills, `map`
// results) refuse it.
//
// Run matching is leftmost and non-overlapping; in a variadic run set, the longest wins at a tie.
//
// ## Decoding is total
//
// `bytes` is the type with nothing to validate — every octet is a valid octet, which is why it carries
// `is_ascii()` but no `is_valid()`. Decoding *out* of it never fails either: an octet that is not a symbol
// becomes its reserved escape in the text, and converting back returns the same octet.
//
// ```go
// b = bytes([0x61, 0xFF, 0x62])
// b.string().bytes() == b         // => true
// b.runes().bytes() == b          // => true
// b.is_ascii()                    // => false
// b.string().is_valid()           // => false — ask the type that can answer it
// ```
//
// See [string: Undecodable octets](string.md#undecodable-octets) for the model, and note the one boundary that
// is **not** total: `json.encode` on text holding escapes raises, while `json.encode(b)` on the bytes
// themselves is always fine (base64).
//
// ## The blank set
//
// The text types share one notion of insignificant content — **NUL ∪ whitespace** — projected into each
// type's element domain. For `bytes` that is NUL plus **ASCII whitespace**: the fixed octet set
// `{0x00, ' ', '\t', '\n', '\r', '\v', '\f'}` — all the whitespace a single octet can express. Deciding
// anything wider would require decoding, which octets never get: an NBSP's octets (`0xC2 0xA0`) are
// **content** here, not blank. The default pad fill is the canonical member of the set, the space octet.
//
// The no-argument form of a member reads through this set:
//
// - `trim()` / `trim_start()` / `trim_end()` strip blank octets; `split()` separates on runs of them.
// - The queries — `contains()`, `any()`, `count()`, `index()`, `index_last()`, `all()` — ask about
// **significant** (non-blank) octets.
// - `keep()` keeps the significant octets; `remove()` removes the blank ones — two actions, same answer.
//
// ```go
// bytes("  hi \t ").trim()            // => bytes([104, 105])
// bytes([0, 104, 0]).trim()           // => bytes([104]) — NUL is blank
// bytes("\u00a0x\u00a0").trim()       // => bytes([194, 160, 120, 194, 160]) — NBSP octets are content
// bytes("  a b \t c ").split()        // => [bytes([97]), bytes([98]), bytes([99])]
// bytes("  ").contains()              // => false — no significant octet
// bytes(" a b ").count()              // => 2
// bytes("  ab").index()               // => 2 — first significant octet
// bytes(" a b ").remove()             // => bytes([97, 98])
// bytes("42").pad_start(5)            // => bytes([32, 32, 32, 52, 50]) — default fill = the space octet
// ```
//
// ## In-place twins
//
// `bytes` is a mutable-body type, so every eligible transform has an `_in_place` twin that mutates the shared
// body (visible through every alias), **returns the receiver**, and raises kind `not_mutable` on a frozen
// receiver:
//
// ```go
// t := bytes("  hi  ")
// alias := t
// t.trim_in_place()               // => bytes([104, 105]) — returns the receiver
// // alias is now bytes([104, 105]) too
//
// fz := bytes("abc").freeze()
// fz.push_in_place(b'x')          // Error: not_mutable: (push_in_place) type immutable-bytes is immutable
// fz.append(b'd')                 // => bytes([97, 98, 99, 100]) — the copying form works on a frozen receiver
// ```
//
// The unsuffixed name is always the safe, copying form. `slice` has no twin (`slice_view` names the saving);
// `repeat` has none (its result is n × the receiver); `map`/`split`/`partition`/`chunk` have none.
//
// ## Exclusions
//
// - **`upper` / `lower` / `case_fold` / `title_case` and the identifier casings** — symbol classes and case
// mappings assume an encoding, which octets never get; decode first: `.string().upper()`.
// - **`sum` / `avg`** — elements are octets, not numbers; the checksum spelling is
// `b.reduce(0, func(acc, x) { return acc + x.int() })`.
// - **`join`** — the collection-as-receiver render lives on `array`/`range`; spell it
// `b.array().join(",")` if you want an octet listing.
// - **Scalar conversions (`.int()`, `.float()`, `.decimal()`, `.bool()`, `.time()`)** — octets are not text
// until decoded; `string` is the gateway: `.string().int()`.
// - **`.dict()` / `.record()`** — octets are never key/value entries.
// - **`.byte()`** — wrapping/unwrapping is not conversion; the element access is `b[0]`.
// - **`fields`** — no-arg `split()` *is* the whitespace splitter.
// - **`copy_shallow` / `freeze_shallow`** — elements are scalars; there is no second level to stop at.
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
// bytes("abc").count(func(i, c) { return i > 0 })            // => 2 — accepts two, not one
// bytes("abc").index(func(c, ...rest) { return c == 'b' })  // => 1 — accepts one: the element
// bytes("abcd").keep("bd".contains)                         // => bytes([98, 100])
// ```
//
//kavun:type bytes

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

	Methods: bytesMethods,
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
	if mutate && v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	items, err := bytesAddItems(name, args)
	if err != nil {
		return Undefined, err
	}

	if mutate {
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
				l := []byte(text.EncodeRuneText(rune(other.Data)))
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
			total, terr := seq.RepeatTotal(op.String(), n, sl)
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
		r := []byte(text.EncodeRuneText(rune(other.Data)))
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
		// a rune contributes its encoding; an escape's encoding is its octet
		return []byte(text.EncodeRuneText(rune(a.Data))), true, nil
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
//   - no argument      the significant octets (not text.IsBlankByte)
//   - a function       a predicate, f/1(octet) or f/2(index, octet)
//   - text content     a run; several arguments form a set of runs
//
// any/all ask about single octets, so they take octets, a predicate, or nothing — never a run. A set may not mix
// element-typed arguments (byte/rune/int) with text-typed ones (string/runes/bytes), and a function among several
// arguments raises. Runs match leftmost-longest and never overlap.
// ---------------------------------------------------------------------------

// bytesMatch is a match member's argument list after reading; exactly one reading is set.
type bytesMatch struct {
	significant bool          // no argument
	pred        *ElemCallback // a predicate
	elems       []byte        // any/all: a set of octets
	runs        [][]byte      // contains/count/keep/remove: a set of runs
}

// bytesReadMatchArgs reads a match member's arguments. octets is true for any/all, which read each argument as
// exactly one octet instead of a run.
func bytesReadMatchArgs(vm VM, name string, args []Value, octets bool) (bytesMatch, error) {
	if len(args) == 0 {
		return bytesMatch{significant: true}, nil
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return bytesMatch{}, seq.ErrPredicateAmongMany(name)
		}
		cb, err := newElemCallback(vm, name, args[0])
		if err != nil {
			return bytesMatch{}, err
		}
		return bytesMatch{pred: cb}, nil
	}

	runs := make([][]byte, 0, len(args))
	sawSymbol, sawText := false, false
	for _, a := range args {
		if a.IsCallable() {
			return bytesMatch{}, seq.ErrFunctionInSet(name)
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
		return bytesMatch{}, seq.ErrMixedSet(name)
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
			return bytesMatch{}, seq.ErrNotOneElement(name)
		}
		elems = append(elems, r[0])
	}
	return bytesMatch{elems: elems}, nil
}

// matches reports whether octet r at index i matches an element-wise reading (anything but runs).
func (m *bytesMatch) matches(i int, r byte) (bool, error) {
	switch {
	case m.significant:
		return !text.IsBlankByte(r), nil
	case m.pred != nil:
		res, err := m.pred.Call(i, ByteValue(r))
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	return slices.Contains(m.elems, r), nil
}

// bytesKept answers the octets keep(...) keeps: the matches, in order. Shared by keep and keep_in_place; name is
// the member called, for the errors.
func bytesKept(vm VM, name string, v Value, args []Value) ([]byte, error) {
	m, err := bytesReadMatchArgs(vm, name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := seq.RunLengthAt(elems, i, m.runs); k > 0 {
				out = append(out, elems[i:i+k]...)
				i += k
			} else {
				i++
			}
		}
		return out, nil
	}

	for i, r := range elems {
		hit, err := m.matches(i, r)
		if err != nil {
			return nil, err
		}
		if hit {
			out = append(out, r)
		}
	}
	return out, nil
}

// bytesRemaining answers the octets remove(...) leaves: everything but the matches, in order. With no argument
// remove drops the BLANK octets — so it keeps the significant ones, landing on keep()'s answer by the opposite
// action. Shared by remove and remove_in_place; name is the member called, for the errors.
func bytesRemaining(vm VM, name string, v Value, args []Value) ([]byte, error) {
	m, err := bytesReadMatchArgs(vm, name, args, false)
	if err != nil {
		return nil, err
	}
	elems := (*Bytes)(v.Ptr).Elements
	out := make([]byte, 0, len(elems))

	if m.significant {
		for _, r := range elems {
			if !text.IsBlankByte(r) {
				out = append(out, r)
			}
		}
		return out, nil
	}

	if m.runs != nil {
		for i := 0; i < len(elems); {
			if k := seq.RunLengthAt(elems, i, m.runs); k > 0 {
				i += k
			} else {
				out = append(out, elems[i])
				i++
			}
		}
		return out, nil
	}

	for i, r := range elems {
		hit, err := m.matches(i, r)
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
// counted in octets. The argument's type selects the reading:
//   - no argument      the first / last significant octet (not text.IsBlankByte)
//   - a function       a predicate, f/1(octet) or f/2(index, octet)
//   - text content     a contiguous run (string, runes or bytes)
//   - anything else    one octet, compared with ==; a value that is not one octet raises
// A miss answers undefined, or the trailing default. Never variadic: the second slot is the default.
// ---------------------------------------------------------------------------

// bytesLocate is the body of index and index_last; name is the member called, for the errors.
func bytesLocate(vm VM, name string, v Value, args []Value, last bool) (Value, error) {
	if len(args) > 2 {
		return Undefined, ErrArgCount
	}
	elems := (*Bytes)(v.Ptr).Elements

	if len(args) == 0 {
		idx := -1
		for i, e := range elems {
			if !text.IsBlankByte(e) {
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
		cb, err := newElemCallback(vm, name, needle)
		if err != nil {
			return Undefined, err
		}
		idx := -1
		for i, e := range elems {
			res, err := cb.Call(i, ByteValue(e))
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
		return Undefined, seq.ErrNotOneElement(name)
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

// bytesIndexRun is seq.IndexRun on octets, through the standard library's searcher.
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
//   - trim*            a set of single octets, stripped repeat-while; no argument = the blank set (text.IsBlankByte)
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
			return nil, seq.ErrNotOneElement(name)
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
		return nil, seq.ErrMixedSet(name)
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
			return text.IsBlankByte(e)
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
	if len(args) > 2 {
		return nil, ErrArgCount
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
	width, err := seq.PadWidth(name, n)
	if err != nil {
		return nil, err
	}
	pad := slices.Repeat([]byte{fill}, width-len(elems))
	if start {
		return slices.Concat(pad, elems), nil
	}
	return slices.Concat(elems, pad), nil
}

// --------------------------------------------------------------------------- split(...seps) and partition(...seps): a
// separator is a run (an element being a run of one), a homogeneous set of runs, a predicate on single octets, or — no
// argument — the blank set (text.IsBlankByte). Runs match leftmost-longest and never overlap; an empty run matches
// nothing. ---------------------------------------------------------------------------

// bytesPiece is one piece of split/partition: new bytes, independent of the receiver.
func bytesPiece(elems []byte) Value {
	return NewBytesValue(slices.Clone(elems), false)
}

// ---------------------------------------------------------------------------
// The callback members: for_each, map, flat_map, reduce. A per-element callback is f/1(octet) or
// f/2(index, octet); reduce's is f/2(acc, octet) or f/3(acc, index, octet).
// ---------------------------------------------------------------------------

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
