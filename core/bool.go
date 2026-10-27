package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const boolTypeName = "bool"

func BoolValue(b bool) Value {
	if b {
		return True
	}
	return False
}

// The two truth values, `true` and `false`.
//
// ## Overview
//
// `bool` is a scalar with conversions only — no arithmetic, no elements. Two distinct questions meet here
// and stay separate:
//
// - **Conversion** — `bool(x)` / `x.bool()`: a numeric zero test, or a text *parse*.
// - **Truthiness** — `is_true(x)` / `x.is_true()` / `!!x`: is this value different from its type's zero value?
// Truthiness is defined for *every* type; the logical operators `&&`/`||`/`!` and `if`/`for` conditions use
// it, not the conversion.
//
// The two disagree on purpose: `bool("false")` → `false` (parses the word), `is_true("false")` → `true`
// (a non-empty string).
//
// `bool` values are immutable.
//
// ## Literals and Construction
//
// ```go
// t := true
// f := false
// bool()          // => false — the zero value
// ```
//
// ## Operators
//
// ### No arithmetic
//
// `bool` takes part in no arithmetic at all — every combination raises:
//
// ```go
// true + 1        // Error: invalid_binary_operator: bool + int
// true + true     // Error: invalid_binary_operator: bool + bool
// true * 2        // Error: invalid_binary_operator: bool * int
// -true           // Error: invalid_unary_operator: - bool
// ```
//
// Unary `^` is boolean negation of a `bool` (same-type complement); `!` is the universal truthiness negation:
//
// ```go
// ^true           // => false
// !true           // => false
// ```
//
// ### Comparison
//
// `false < true`; against the numeric scalars a `bool` compares as 0/1:
//
// ```go
// false < true    // => true
// true <= true    // => true
// true == 1       // => true
// false == 0      // => true
// true < 2        // => true
// true < 1.5      // => true
// true == b'\x01' // => true
// true == 2       // => false — equality is by value, not truthiness
// true == "true"  // => true — against text, equality compares the bool's text form
// ```
//
// ### Logical operators — truthiness, any type, short-circuiting
//
// `&&` and `||` evaluate *truthiness* of any operand type, short-circuit, and answer the deciding **operand
// value** (not a forced `bool`); `!` always answers a `bool`:
//
// ```go
// 1 && "a"        // => "a" — left is truthy, right decides
// 0 && "a"        // => 0 — left decides, right never evaluated
// "" || "x"       // => "x"
// "y" || "x"      // => "y"
// !5              // => false
// !""             // => true
// !!"x"           // => true — the idiomatic "as bool" spelling
// ```
//
// A value whose truthiness is an error state raises rather than answering:
//
// ```go
// float("nan") && true   // Error: invalid_value: float NaN is neither true nor false in a boolean context
// ```
//
// ## Conversions
//
// Every conversion is `x.T([default])`: a valid `T`, or a catchable raise, or the default if one is given.
//
// There is no `float()`, `decimal()`, `byte()`, or `rune()` member — `int` is the sole gateway
// (`true.int().float()`), and no `time()`/container targets exist.
//
// ## The free `bool(x)` conversion — zero test or parse
//
// `bool(x)` (≡ `x.bool()` where the member exists) is defined on exactly two domains:
//
// **Numeric** — the `bool → int` edge read backwards, a zero test:
//
// ```go
// bool(0)               // => false
// bool(2)               // => true
// bool(-1)              // => true
// bool(0.0)             // => false
// bool(0.5)             // => true
// bool(decimal("0"))    // => false
// ```
//
// **Text** — a parse of the wide literal set, case-insensitive: `true`/`false`, `1`/`0`, `t`/`f`, `yes`/`no`.
// Invalid text **raises** (never a silent `false`); the member's optional default rescues it:
//
// ```go
// bool("false")         // => false
// bool("FALSE")         // => false
// bool("Yes")           // => true
// bool("t")             // => true
// bool("0")             // => false
// bool("abc")           // Error: conversion: cannot convert string to bool
// bool("")              // Error: conversion — empty is not a boolean word
// "abc".bool(false)     // => false — the member's default rescues bad data
// "false".bool()        // => false — same operation, member spelling
// ```
//
// Nothing else converts: `bool([])` and `bool(b'\x00')` raise — for every other type the question is
// truthiness, spelled `is_true`/`!!x`.
//
//kavun:type bool

var TypeBool = ValueTypeDescr{
	Name:         ConstHook(boolTypeName),                                 // PURE by contract
	String:       boolTypeString,                                          // PURE by contract
	Format:       boolTypeFormat,                                          // PURE by contract
	Interface:    func(v Value) any { return v.Data != 0 },                // PURE by contract
	EncodeJSON:   boolTypeEncodeJSON,                                      // PURE by contract
	EncodeBinary: boolTypeEncodeBinary,                                    // PURE by contract
	DecodeBinary: boolTypeDecodeBinary,                                    // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil }, // PURE by contract
	Equal:        boolTypeEqual,                                           // PURE by contract
	BinaryOp:     boolTypeBinaryOp,                                        // PURE by contract
	UnaryOp:      boolTypeUnaryOp,                                         // PURE by contract
	Len:          ConstHook(int64(1)),                                     // PURE by contract
	AsString:     boolTypeAsString,                                        // PURE by contract
	AsInt:        boolTypeAsInt,                                           // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true }, // PURE by contract

	Methods: boolMethods,
}

func boolTypeEncodeJSON(v Value) ([]byte, error) {
	s := boolTypeString(v)
	return []byte(s), nil
}

func boolTypeEncodeBinary(v Value) ([]byte, error) {
	return []byte{uint8(v.Data)}, nil
}

func boolTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("bool: expected 1 byte, got %d", len(data))
	}
	v.Data = uint64(data[0])
	return nil
}

func boolTypeString(v Value) string {
	if v.Data == 0 {
		return "false"
	}
	return "true"
}

func boolTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	var body string
	switch sp.Verb {
	case 'v':
		return boolTypeString(v), nil

	case 'T':
		body = boolTypeName

	case 0, 't':
		if v.Data == 0 {
			body = "false"
		} else {
			body = "true"
		}

	case 'd':
		if v.Data == 0 {
			body = "0"
		} else {
			body = "1"
		}

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	return fspec.ApplyGenerics(body, sp, fspec.AlignLeft), nil
}

func boolTypeAsString(v Value) (string, bool) {
	if v.Data == 0 {
		return "false", true
	}
	return "true", true
}

func boolTypeAsInt(v Value) (int64, bool) {
	if v.Data == 0 {
		return 0, true
	}
	return 1, true
}

func boolTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Bool:
		return v.Data == other.Data
	}

	// default to false if final
	if final {
		return false
	}

	// delegate to other type's Equal if not final
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func boolTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		// bool always recognizes same-type non-reflected, so it can never be reached reflected — nothing ever needs to
		// delegate into bool.
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Bool:
		l := v.Data != 0
		r := other.Data != 0
		switch op {
		case tokens.Less:
			return BoolValue(!l && r), nil
		case tokens.Greater:
			return BoolValue(l && !r), nil
		case tokens.LessEq:
			return BoolValue(!l || r), nil
		case tokens.GreaterEq:
			return BoolValue(l || !r), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract.
func boolTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case tokens.Xor:
		return BoolValue(v.Data == 0), nil
	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}
