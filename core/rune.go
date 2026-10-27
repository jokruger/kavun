package core

import (
	"encoding/binary"
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

const runeTypeName = "rune"

func RuneValue(c rune) Value {
	return Value{
		Type:      value.Rune,
		Immutable: true,
		Data:      uint64(c),
	}
}

// One Unicode code point.
//
// ## Overview
//
// `rune` is an ordinal scalar with text content: the element type of [`string`](string.md) and
// [`runes`](runes.md). It holds a single code point (`U+0000`–`U+10FFFF`, excluding the surrogate range),
// whatever its UTF-8 length — `'A'` and `'є'` are both one rune.
//
// Its **domain** is exactly what can be encoded back to octets: a Unicode scalar value, **or** one of the 128
// [octet escapes](string.md#undecodable-octets) (`U+DC80`–`U+DCFF`) standing for an octet that is not a symbol.
// Everything else — a high surrogate, a negative, anything past `U+10FFFF` — raises where it is written, so no
// conversion *out* of a `rune` can fail. `is_valid()` distinguishes the two: a real symbol, or an escape.
//
// `rune` is ordinal, not numeric: it is comparable and supports offset arithmetic, but has no `*`/`/`, no
// `sum`/`avg` participation, and **its overflow policy is raise** — an arithmetic result that leaves the
// code-point space is an error, never a silent `U+FFFD`.
//
// `rune` values are immutable.
//
// No sequence members — one code point has no elements.
//
// ## Literals and Construction
//
// ```go
// a := 'A'             // U+0041
// ye := 'є'            // U+0454 — any code point, directly
// nl := '\n'           // escapes accepted
// x := '\x41'          // => 'A'
// u := '\u0454'        // => 'є'
// top := '\U0010FFFF'
// ```
//
// ```go
// rune()               // => '\x00' — the zero value (NUL)
// rune(65)             // => 'A' — conversion from int
// rune(1114112)        // Error: conversion — above U+10FFFF
// rune(55296)          // Error: conversion — the surrogate range U+D800–U+DFFF is not a code point
// (1114112).rune('?')  // => '?' — the member's optional default rescues the failure
// ```
//
// Promotion into a sequence: there is no `repeat()` on scalars — go through the one-symbol string:
//
// ```go
// 'a'.string().repeat(3)   // => "aaa"
// u"".pad_end(3, 'a')      // => u"aaa"
// ```
//
// ## Operators
//
// ### Arithmetic — offsets raise on overflow
//
// `rune + int` and `rune - int` move along the code-point axis and return a `rune`; a result outside
// `U+0000`–`U+10FFFF`, or inside the surrogate range, **raises** — never a replacement character:
//
// ```go
// 'a' + 1              // => 'b'
// 'a' - 1              // => '`'
// 1 + 'a'              // => 'b'
// '\U0010FFFF' + 1     // Error: invalid_value: rune overflow: 1114112 is outside a rune's domain
// '\uD7FF' + 1         // Error: invalid_value: rune overflow: 55296 is outside a rune's domain — a surrogate
// 'a' - 98             // Error: invalid_value: rune overflow: -1 is outside a rune's domain
// ```
//
// `rune - rune` is the distance between two code points — a plain `int`, not a rune (`rune - byte` reads the
// same way):
//
// ```go
// 'd' - 'a'            // => 3
// 'a' - b'a'           // => 0
// ```
//
// | operator | with `int` | with `rune` | with `byte` | with `float` / `decimal` |
// | --- | --- | --- | --- | --- |
// | `+` | `rune` (guarded) | raises | raises | raises |
// | `-` | `rune` (guarded) | `int` — the distance | `int` — the distance | raises |
// | `*` `/` `%` | raises | raises | raises | raises |
//
// There is no `rune + rune`: adding two code points has no meaning.
//
// ### Comparison
//
// Comparisons order by code point and accept the numeric-valued scalars — `rune`, `byte`, `int`, `bool`
// (as 0/1), `float`:
//
// ```go
// 'a' < 'b'            // => true
// 'a' == 97            // => true
// 97 == 'a'            // => true
// 'a' == b'a'          // => true
// 'a' < 97.5           // => true
// ```
//
// ### As text content
//
// Next to `string`/`runes` a rune is itself — one symbol; into `bytes` it is its UTF-8 octets (1–4 of them).
// A scalar on the left takes the sequence's type:
//
// ```go
// "ab" + 'c'             // => "abc"
// 'a' + "bc"             // => "abc" — scalar on the left, result is still a string
// u"ab" + 'є'            // => u"abє"
// bytes("ab") + 'є'      // => bytes([97, 98, 209, 148]) — the 2-octet UTF-8 encoding
// "абв".contains('б')    // => true — same acceptance as the operators
// ```
//
// ## Conversions
//
// Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given.
//
// | member | result | partial? |
// | --- | --- | --- |
// | `int()` | the code point's value | total |
// | `rune()` | identity | total |
// | `byte()` | the same symbol as one octet | ASCII, **or an escape** — raises otherwise |
// | `string()` / `runes()` | the symbol as text; an escape gives its octet | total (the default is never used) |
//
// `byte()` is encoding-based: it succeeds iff the symbol's UTF-8 representation is a single octet — exactly
// ASCII. `U+0080`–`U+00FF` do *not* convert (their UTF-8 form is two octets); the Latin-1 numeric reading is
// `.int()` explicitly.
//
// There is deliberately no `bool()`, `float()`, or `decimal()` (through `.int()` — `int` is the sole gateway)
// and no `bytes()` (write `.string().bytes()`). Text never parses into `rune` — `"65".rune()` raises; write
// `"65".int().rune()`.
//
//kavun:type rune

var TypeRune = ValueTypeDescr{
	Name:         ConstHook(runeTypeName),                                                         // PURE by contract
	String:       func(v Value) string { return fmt.Sprintf("%q", rune(v.Data)) },                 // PURE by contract
	Format:       runeTypeFormat,                                                                  // PURE by contract
	Interface:    func(v Value) any { return rune(v.Data) },                                       // PURE by contract
	EncodeJSON:   runeTypeEncodeJSON,                                                              // PURE by contract
	EncodeBinary: runeTypeEncodeBinary,                                                            // PURE by contract
	DecodeBinary: runeTypeDecodeBinary,                                                            // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                         // PURE by contract
	Len:          ConstHook(int64(1)),                                                             // PURE by contract
	Equal:        runeTypeEqual,                                                                   // PURE by contract
	BinaryOp:     runeTypeBinaryOp,                                                                // PURE by contract
	AsString:     func(v Value) (string, bool) { return text.EncodeRuneText(rune(v.Data)), true }, // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                      // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                         // PURE by contract
	AsRune:       func(v Value) (rune, bool) { return rune(v.Data), true },                        // PURE by contract
	AsByte:       runeTypeAsByte,                                                                  // PURE by contract

	Methods: runeMethods,
}

func runeTypeEncodeJSON(v Value) ([]byte, error) {
	c := rune(v.Data)
	s := strconv.FormatInt(int64(c), 10)
	return []byte(s), nil
}

func runeTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v.Data))
	return b, nil
}

func runeTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("rune: expected 4 bytes, got %d", len(data))
	}
	v.Data = uint64(binary.LittleEndian.Uint32(data))
	return nil
}

func runeTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return fmt.Sprintf("%q", rune(v.Data)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(runeTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.HasPrec || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	r := rune(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'c'
	}

	switch verb {
	case 'c':
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(text.EncodeRuneText(r), sp, fspec.AlignLeft), nil

	case 'q':
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(r), sp, fspec.AlignLeft), nil

	case 'd':
		if sp.Grouping == ',' || sp.Grouping == '_' || sp.Grouping == 0 {
			// fine
		} else {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		negative := r < 0
		var digits string
		if negative {
			digits = strconv.FormatUint(uint64(-int64(r)), 10)
		} else {
			digits = strconv.FormatUint(uint64(r), 10)
		}
		if sp.Grouping != 0 {
			digits = fspec.GroupDigits(digits, sp.Grouping, 3)
		}
		sign := fspec.SignPrefix(sp.Sign, negative)
		if negative {
			sign = "-"
		}
		body := sign + digits
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil

	case 'x', 'X':
		if sp.Grouping == ',' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		// per docs: rune hex has no "0x" prefix (unlike int/byte).
		digits := strconv.FormatUint(uint64(uint32(r)), 16)
		if verb == 'X' {
			digits = strings.ToUpper(digits)
		}
		if sp.Grouping == '_' {
			digits = fspec.GroupDigits(digits, '_', 4)
		}
		body := fspec.SignPrefix(sp.Sign, false) + digits
		return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil

	case 'U':
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		digits := strings.ToUpper(strconv.FormatUint(uint64(uint32(r)), 16))
		if len(digits) < 4 {
			digits = strings.Repeat("0", 4-len(digits)) + digits
		}
		return fspec.ApplyGenerics("U+"+digits, sp, fspec.AlignRight), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
}

func runeTypeAsByte(v Value) (byte, bool) {
	// a symbol converts to one octet iff its UTF-8 form IS one octet — ASCII;
	// U+0080-U+00FF would produce an octet that is not a representation of the
	// symbol (its UTF-8 form is two octets). The Latin-1 reading is .int() explicitly.
	// An ESCAPE converts too, and to the octet it stands for: that is the whole point of
	// the escape, and it is the repair path — b'\xff' back out of the text that holds it
	c := rune(v.Data)
	if text.IsEscapeRune(c) {
		return text.EscapeRuneOctet(c), true
	}
	if c > 0x7F {
		return byte(c), false
	}
	return byte(c), true
}

func runeTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Rune, value.Byte, value.Bool:
		return v.Data == other.Data
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
// runeArithResult guards rune arithmetic: a result outside the code-point space (or inside the surrogate
// range) raises instead of silently becoming U+FFFD — one overflow policy per type, and rune's is raise.
func runeArithResult(r int64) (Value, error) {
	if !text.IntInRuneDomain(r) {
		return Undefined, errs.NewInvalidValueError(fmt.Sprintf("rune overflow: %d is outside a rune's domain", r))
	}
	return RuneValue(rune(r)), nil
}

func runeTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Int:
			l := int64(other.Data)
			r := int64(v.Data)
			switch op {
			case tokens.Add:
				return runeArithResult(l + r)
			case tokens.Less:
				return BoolValue(l < r), nil
			case tokens.Greater:
				return BoolValue(l > r), nil
			case tokens.LessEq:
				return BoolValue(l <= r), nil
			case tokens.GreaterEq:
				return BoolValue(l >= r), nil
			}

		case value.Byte:
			l := int64(other.Data)
			r := int64(v.Data)
			switch op {
			case tokens.Sub:
				return IntValue(l - r), nil
			case tokens.Less:
				return BoolValue(l < r), nil
			case tokens.Greater:
				return BoolValue(l > r), nil
			case tokens.LessEq:
				return BoolValue(l <= r), nil
			case tokens.GreaterEq:
				return BoolValue(l >= r), nil
			}

		case value.Bool:
			l := int64(other.Data)
			r := int64(v.Data)
			switch op {
			case tokens.Less:
				return BoolValue(l < r), nil
			case tokens.Greater:
				return BoolValue(l > r), nil
			case tokens.LessEq:
				return BoolValue(l <= r), nil
			case tokens.GreaterEq:
				return BoolValue(l >= r), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Rune, value.Byte:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Sub:
			return IntValue(l - r), nil
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}

	case value.Int:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Add:
			return runeArithResult(l + r)
		case tokens.Sub:
			return runeArithResult(l - r)
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}

	case value.Bool:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}
