package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/text"
)

const byteTypeName = "byte"

func ByteValue(v byte) Value {
	return Value{
		Type:      value.Byte,
		Immutable: true,
		Data:      uint64(v),
	}
}

// One octet: an unsigned 8-bit value, 0–255.
//
// ## Overview
//
// `byte` is an ordinal scalar with text content. It is two things at once:
//
// - **A number-like value** for binary data — and **the only modular type in the language**: `byte` arithmetic
// wraps modulo 256. Every other numeric type (`int`, `float`, `decimal`, `rune`) raises on overflow.
// - **A symbol in ASCII, an octet above it.** As text content, an octet reads as a symbol in `0x00`–`0x7F`;
// `0x80`–`0xFF` alone is not valid UTF-8, so its text is the **octet itself**, which reads back as that
// octet's [escape](string.md#undecodable-octets) and converts back to this same byte. The conversion is
// total and lossless in both directions. The Latin-1 reading is never assumed — `b'\xFF' == "ÿ"` is `false`,
// and the Latin-1 value is reachable explicitly through `.int()`.
//
// `byte` values are immutable. The element type of [`bytes`](bytes.md).
//
// No sequence members (`len`, `repeat`, …) — a `byte` has no elements.
//
// ## Literals and Construction
//
// ```go
// b := b'A'          // => byte(65)
// nl := b'\n'        // => byte(10) — byte-sized escapes accepted
// ff := b'\xFF'      // => byte(255)
// ```
//
// A `b'...'` literal must resolve to exactly one octet: empty contents, multi-character contents, and code
// points above 255 are compile errors.
//
// ```go
// byte()              // => byte(0) — the zero value
// byte(65)            // => byte(65) — conversion from int, raises outside 0..255
// byte(256)           // Error: conversion: cannot convert int to byte
// (256).byte(b'\x00') // => byte(0) — the member's optional default rescues the failure
// ```
//
// Promotion into a sequence is the count constructor's job (there is no `repeat()` on scalars):
//
// ```go
// bytes(b'A', 3)     // => bytes([65, 65, 65])
// ```
//
// ## Operators
//
// ### Arithmetic — wraps mod 256
//
// `+` and `-` accept a `byte` or a plain `int` on either side; the result is always a `byte`, reduced mod 256
// for *any* magnitude of the `int` operand:
//
// ```go
// byte(10) + byte(3)   // => byte(13)
// b'\xFF' + 1          // => byte(0) — wraps; every other numeric type raises here
// b'\x00' - 1          // => byte(255)
// byte(0) - 300        // => byte(212) — any magnitude, still mod 256
// 1 + b'\xFF'          // => byte(0) — either operand order
// -b'\x01'             // => byte(255) — unary minus is the additive inverse mod 256
// ```
//
// | operator | with `byte` | with `int` | with `float` / `decimal` |
// | --- | --- | --- | --- |
// | `+` `-` | `byte`, wraps | `byte`, wraps | raises |
// | `*` `/` `%` | raises | raises | raises |
//
// ### Bitwise
//
// `& | ^ &^` are same-type only; the shift count additionally accepts a plain `int`:
//
// ```go
// b'\x0F' & b'\x03'    // => byte(3)
// b'\x0F' | b'\x30'    // => byte(63)
// b'\x0F' ^ b'\xFF'    // => byte(240)
// ^b'\x00'             // => byte(255) — unary complement
// b'\x01' << 3         // => byte(8) — int shift count is fine
// b'\x01' & 5          // Error: invalid_binary_operator: byte & int
// ```
//
// ### Comparison
//
// `==`/`!=`/`<`/`<=`/`>`/`>=` compare `byte` with the other numeric-valued scalars by numeric value —
// `byte`, `int`, `rune`, `bool` (as 0/1), and `float`. Next to *text*, equality reads the canonical text —
// the symbol — exactly like `rune` (ordering against text raises):
//
// ```go
// b'A' == 65           // => true
// b'A' < 66            // => true
// 65 == b'A'           // => true
// b'A' < b'B'          // => true
// b'A' == 'A'          // => true — same code point
// b'A' < 65.5          // => true
// "A" == b'A'          // => true — equality against text reads the symbol, like rune
// "65" == b'A'         // => false — the text is the symbol, never the number
// ```
//
// ### As text content
//
// Next to the text types, a `byte` is a *symbol*, and only ASCII octets have one. A scalar on the left takes
// the sequence's type:
//
// ```go
// "abc" + b'a'         // => "abca"
// b'a' + "bc"          // => "abc" — scalar on the left, result is still a string
// u"ab" + b'c'         // => u"abc"
// "abc" + byte(200)    // Error: invalid_value: an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got 200
// u"ab" + byte(200)    // Error: invalid_value — same rule on runes
// ```
//
// Into `bytes` a byte is always exactly one octet — no symbol question arises:
//
// ```go
// bytes("ab") + b'\xFF'   // => bytes([97, 98, 255]) — any octet, unconditionally
// ```
//
// The same acceptance applies to search arguments:
//
// ```go
// "abc".contains(b'a')       // => true
// "abc".index(b'c')          // => 2
// "abc".contains(byte(200))  // Error: invalid_value — no symbol exists
// bytes("abc").contains(b'b') // => true — always fine on bytes
// ```
//
// The symbol is the `byte`'s **canonical text**: every convert-to-string surface answers it (matching `.string()`), and
// a high octet — having no symbol — answers its escape there. Dict keys, `join`, and record membership all read it:
//
// ```go
// d := dict(); d[b'A'] = 1
// d.keys()                   // => ["A"] — a byte key stores under its symbol
// d[b'\xFF'] = 1             // stored under "\xff" — a high octet's text is its escape
// ["x", b'A'].join("-")      // => "x-A"
// ["x", b'\xFF'].join("-")   // => "x-\xff" — the octet's escape
// b'A' in {A: 1}             // => true — a record key is the symbol too
// ```
//
// Display renders — `format()`, f-strings — are a different question and stay *numeric*; see
// [`format`](#format).
//
// ## Conversions
//
// Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given
// (the default is not type-checked).
//
// | member | result | partial? |
// | --- | --- | --- |
// | `int()` | the octet's value, 0–255 | total |
// | `byte()` | identity | total |
// | `rune()` | the same symbol | ASCII only — raises above `0x7F` |
// | `string()` / `runes()` | the one-octet text holding it | **total** — above `0x7F` the text holds the octet |
//
// The `rune` conversion is encoding-based: it succeeds iff the octet is the complete UTF-8 representation of a
// symbol on its own — exactly ASCII. `string`/`runes` are total. There is deliberately no `bool()`, `float()`,
// `decimal()`, or `bytes()` member: `int` is the sole gateway to the numeric and boolean domains
// (`b.int().float()`, `b'0'.int().bool()`), and `string` is the gateway to octets (`b.string().bytes()`).
// Text never parses directly into `byte` either — `"65".byte()` raises; write `"65".int().byte()`.
//
//kavun:type byte

var TypeByte = ValueTypeDescr{
	Name:         ConstHook(byteTypeName),                                         // PURE by contract
	String:       func(v Value) string { return fmt.Sprintf("byte(%d)", v.Data) }, // PURE by contract
	Format:       byteTypeFormat,                                                  // PURE by contract
	Interface:    func(v Value) any { return byte(v.Data) },                       // PURE by contract
	EncodeJSON:   byteTypeEncodeJSON,                                              // PURE by contract
	EncodeBinary: byteTypeEncodeBinary,                                            // PURE by contract
	DecodeBinary: byteTypeDecodeBinary,                                            // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },         // PURE by contract
	Len:          ConstHook(int64(1)),                                             // PURE by contract
	Equal:        byteTypeEqual,                                                   // PURE by contract
	UnaryOp:      byteTypeUnaryOp,                                                 // PURE by contract
	BinaryOp:     byteTypeBinaryOp,                                                // PURE by contract

	// A byte's canonical TEXT matches .string(): its ASCII symbol, or for a high octet (0x80-0xFF) the one-octet
	// text that reads as its escape — so a byte dict key stores under "A" (or "\xff"), and join renders it.
	// Display renders stay numeric: String (byte(65)), Format, EncodeJSON.
	AsString: func(v Value) (string, bool) { return text.ByteSymbolString(byte(v.Data)) }, // PURE by contract
	AsInt:    func(v Value) (int64, bool) { return int64(v.Data), true },                  // PURE by contract
	AsRune:   byteTypeAsRune,                                                              // PURE by contract
	AsByte:   func(v Value) (byte, bool) { return byte(v.Data), true },                    // PURE by contract

	Methods: byteMethods,
}

func byteTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(int64(v.Data), 10)
	return []byte(s), nil
}

func byteTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 1)
	b[0] = byte(v.Data)
	return b, nil
}

func byteTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("byte: expected 1 byte, got %d", len(data))
	}
	v.Data = uint64(data[0])
	return nil
}

func byteTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return fmt.Sprintf("byte(%d)", v.Data), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(byteTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.HasPrec || sp.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	n := uint64(byte(v.Data))
	verb := sp.Verb
	if verb == 0 || verb == 'v' {
		verb = 'd'
	}

	// 'c' renders the byte as an ASCII character; only width/fill/align apply.
	if verb == 'c' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(string(rune(n)), sp, fspec.AlignLeft), nil
	}

	// 'q' renders the byte as a quoted character literal.
	if verb == 'q' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(rune(n)), sp, fspec.AlignLeft), nil
	}

	var base int
	var prefix string
	var groupEvery int
	var upper bool

	switch verb {
	case 'd':
		base = 10
		groupEvery = 3
		if sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}

	case 'b':
		base = 2
		prefix = "0b"
		groupEvery = 4

	case 'o':
		base = 8
		prefix = "0o"
		groupEvery = 4

	case 'x':
		base = 16
		prefix = "0x"
		groupEvery = 4

	case 'X':
		base = 16
		prefix = "0x"
		groupEvery = 4
		upper = true

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.Bare {
		prefix = ""
	}

	// grouping rules: ',' is decimal-only; '_' allowed for any base.
	if sp.Grouping == ',' && base != 10 {
		return "", errs.NewUnsupportedFormatSpecMsg("',' grouping is only supported with decimal verb 'd'; use '_' for base-2/8/16")
	}

	digits := strconv.FormatUint(n, base)
	if upper {
		digits = strings.ToUpper(digits)
	}
	if sp.Grouping != 0 {
		digits = fspec.GroupDigits(digits, sp.Grouping, groupEvery)
	}

	body := fspec.SignPrefix(sp.Sign, false) + prefix + digits
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

// byteTypeAsRune: a byte converts to a rune iff it is the UTF-8 representation
// of a symbol on its own, which is exactly ASCII; 0x80-0xFF alone is not valid
// UTF-8, and the Latin-1 reading is reachable through .int() explicitly.
func byteTypeAsRune(v Value) (rune, bool) {
	if v.Data > 0x7F {
		return rune(v.Data), false
	}
	return rune(v.Data), true
}

func byteTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Byte:
		return byte(v.Data) == byte(other.Data)

	case value.Bool:
		// compare directly — equality is part of the settled comparison design
		// and must not ride on the conversion surface (bool has no byte edge)
		var r byte
		if other.Data != 0 {
			r = 1
		}
		return byte(v.Data) == r
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func byteTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Int:
			switch op {
			case tokens.Add:
				return ByteValue(byte(other.Data) + byte(v.Data)), nil
			case tokens.Sub:
				return ByteValue(byte(other.Data) - byte(v.Data)), nil
			case tokens.Less:
				return BoolValue(int64(other.Data) < int64(byte(v.Data))), nil
			case tokens.Greater:
				return BoolValue(int64(other.Data) > int64(byte(v.Data))), nil
			case tokens.LessEq:
				return BoolValue(int64(other.Data) <= int64(byte(v.Data))), nil
			case tokens.GreaterEq:
				return BoolValue(int64(other.Data) >= int64(byte(v.Data))), nil
			}

		case value.Bool:
			switch op {
			case tokens.Less:
				l, _ := boolTypeAsInt(other)
				return BoolValue(l < int64(byte(v.Data))), nil
			case tokens.Greater:
				l, _ := boolTypeAsInt(other)
				return BoolValue(l > int64(byte(v.Data))), nil
			case tokens.LessEq:
				l, _ := boolTypeAsInt(other)
				return BoolValue(l <= int64(byte(v.Data))), nil
			case tokens.GreaterEq:
				l, _ := boolTypeAsInt(other)
				return BoolValue(l >= int64(byte(v.Data))), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Byte:
		switch op {
		case tokens.Add:
			return ByteValue(byte(v.Data) + byte(other.Data)), nil
		case tokens.Sub:
			return ByteValue(byte(v.Data) - byte(other.Data)), nil
		case tokens.And:
			return ByteValue(byte(v.Data) & byte(other.Data)), nil
		case tokens.Or:
			return ByteValue(byte(v.Data) | byte(other.Data)), nil
		case tokens.Xor:
			return ByteValue(byte(v.Data) ^ byte(other.Data)), nil
		case tokens.AndNot:
			return ByteValue(byte(v.Data) &^ byte(other.Data)), nil
		case tokens.Shl:
			return ByteValue(byte(v.Data) << byte(other.Data)), nil
		case tokens.Shr:
			return ByteValue(byte(v.Data) >> byte(other.Data)), nil
		case tokens.Less:
			return BoolValue(byte(v.Data) < byte(other.Data)), nil
		case tokens.Greater:
			return BoolValue(byte(v.Data) > byte(other.Data)), nil
		case tokens.LessEq:
			return BoolValue(byte(v.Data) <= byte(other.Data)), nil
		case tokens.GreaterEq:
			return BoolValue(byte(v.Data) >= byte(other.Data)), nil
		}

	case value.Int:
		switch op {
		case tokens.Add:
			return ByteValue(byte(v.Data) + byte(other.Data)), nil
		case tokens.Sub:
			return ByteValue(byte(v.Data) - byte(other.Data)), nil
		case tokens.And, tokens.Or, tokens.Xor, tokens.AndNot:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), other.TypeName())
		case tokens.Shl:
			return ByteValue(byte(v.Data) << byte(other.Data)), nil
		case tokens.Shr:
			return ByteValue(byte(v.Data) >> byte(other.Data)), nil
		case tokens.Less:
			return BoolValue(int64(byte(v.Data)) < int64(other.Data)), nil
		case tokens.Greater:
			return BoolValue(int64(byte(v.Data)) > int64(other.Data)), nil
		case tokens.LessEq:
			return BoolValue(int64(byte(v.Data)) <= int64(other.Data)), nil
		case tokens.GreaterEq:
			return BoolValue(int64(byte(v.Data)) >= int64(other.Data)), nil
		}

	case value.Bool:
		switch op {
		case tokens.Less:
			l, _ := boolTypeAsInt(other)
			return BoolValue(int64(byte(v.Data)) < l), nil
		case tokens.Greater:
			l, _ := boolTypeAsInt(other)
			return BoolValue(int64(byte(v.Data)) > l), nil
		case tokens.LessEq:
			l, _ := boolTypeAsInt(other)
			return BoolValue(int64(byte(v.Data)) <= l), nil
		case tokens.GreaterEq:
			l, _ := boolTypeAsInt(other)
			return BoolValue(int64(byte(v.Data)) >= l), nil
		}
	}

	// delegate
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func byteTypeUnaryOp(v Value, op token.Token) (Value, error) {
	i := byte(v.Data)
	switch op {
	case tokens.Sub:
		return ByteValue(-i), nil

	case tokens.Xor:
		return ByteValue(^i), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}
