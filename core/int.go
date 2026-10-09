package core

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jokruger/kavun/core/member/members"

	"github.com/jokruger/dec128"
	"github.com/jokruger/fin128/civil"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const intTypeName = "int"

func IntValue(i int64) Value {
	return Value{
		Type:      value.Int,
		Immutable: true,
		Data:      uint64(i),
	}
}

var TypeInt = ValueTypeDescr{
	Name:         ConstHook(intTypeName),                                                               // PURE by contract
	String:       func(v Value) string { return strconv.FormatInt(int64(v.Data), 10) },                 // PURE by contract
	Format:       intTypeFormat,                                                                        // PURE by contract
	Interface:    func(v Value) any { return int64(v.Data) },                                           // PURE by contract
	EncodeJSON:   intTypeEncodeJSON,                                                                    // PURE by contract
	EncodeBinary: intTypeEncodeBinary,                                                                  // PURE by contract
	DecodeBinary: intTypeDecodeBinary,                                                                  // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                              // PURE by contract
	Len:          ConstHook(int64(1)),                                                                  // PURE by contract
	Equal:        intTypeEqual,                                                                         // PURE by contract
	UnaryOp:      intTypeUnaryOp,                                                                       // PURE by contract
	BinaryOp:     intTypeBinaryOp,                                                                      // PURE by contract
	AsString:     func(v Value) (string, bool) { return strconv.FormatInt(int64(v.Data), 10), true },   // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                           // PURE by contract
	AsFloat:      func(v Value) (float64, bool) { return float64(int64(v.Data)), true },                // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return dec128.FromInt64(int64(v.Data)), true }, // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                              // PURE by contract
	AsRune:       intTypeAsRune,                                                                        // PURE by contract
	AsTime:       func(v Value) (time.Time, bool) { return time.Unix(int64(v.Data), 0).UTC(), true },   // PURE by contract
	AsDate:       intTypeAsDate,                                                                        // PURE by contract
	AsByte:       intTypeAsByte,                                                                        // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:    {Fn: memberIsTrue, Pure: true},
		members.String:    {Fn: intString, Pure: true},
		members.Format:    {Fn: memberFormat, Pure: true},
		members.Copy:      {Fn: memberSelf, Pure: true},
		members.Freeze:    {Fn: memberSelf, Pure: true},
		members.Runes:     {Fn: intRunes, Pure: true},
		members.Int:       {Fn: intInt, Pure: true},
		members.Bool:      {Fn: intBool, Pure: true},
		members.Float:     {Fn: intFloat, Pure: true},
		members.Time:      {Fn: intTime, Pure: true},
		members.Decimal:   {Fn: intDecimal, Pure: true},
		members.Date:      {Fn: intDate, Pure: true},
		members.Byte:      {Fn: intByte, Pure: true},
		members.Rune:      {Fn: intRune, Pure: true},
		members.Abs:       {Fn: intAbs, Pure: true},
		members.Sign:      {Fn: intSign, Pure: true},
		members.IsNaN:     {Fn: intIsNaN, Pure: true},
		members.IsInf:     {Fn: intIsInf, Pure: true},
		members.TimeMs:    {Fn: intTimeMs, Pure: true},
		members.TimeMicro: {Fn: intTimeMicro, Pure: true},
		members.TimeNano:  {Fn: intTimeNano, Pure: true},
	},
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(int64(v.Data), 10)
	return []byte(s), nil
}

func intTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v.Data)
	return b, nil
}

func intTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("int: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.LittleEndian.Uint64(data)
	return nil
}

func intTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return strconv.FormatInt(int64(v.Data), 10), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(intTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	if sp.HasPrec || sp.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}

	i := int64(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'd'
	}

	// 'c' renders the code point as a UTF-8 character.
	if verb == 'c' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		if i < 0 || i > utf8.MaxRune {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(string(rune(i)), sp, fspec.AlignLeft), nil
	}

	// 'q' renders the code point as a quoted character literal: 'A', '\n', etc.
	if verb == 'q' {
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad || sp.Bare {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		if i < 0 || i > utf8.MaxRune {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(rune(i)), sp, fspec.AlignLeft), nil
	}

	var (
		base       int
		prefix     string
		groupEvery int
		upper      bool
	)
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

	if sp.Grouping == ',' && base != 10 {
		return "", errs.NewUnsupportedFormatSpecMsg("',' grouping is only supported with decimal verb 'd'; use '_' for base-2/8/16")
	}

	negative := i < 0
	var u uint64
	if negative {
		// safely negate, including math.MinInt64
		u = uint64(-(i + 1)) + 1
	} else {
		u = uint64(i)
	}

	digits := strconv.FormatUint(u, base)
	if upper {
		digits = strings.ToUpper(digits)
	}
	if sp.Grouping != 0 {
		digits = fspec.GroupDigits(digits, sp.Grouping, groupEvery)
	}

	sign := fspec.SignPrefix(sp.Sign, negative)
	if negative {
		sign = "-"
	}
	body := sign + prefix + digits
	return fspec.ApplyGenerics(body, sp, fspec.AlignRight), nil
}

func intTypeAsRune(v Value) (rune, bool) {
	// a rune is a Unicode scalar value OR one of the reserved octet escapes (see text_escape.go);
	// negatives, values past MaxRune and the non-escape surrogates are excluded by the same domain
	// rule, not as special cases. The domain is exactly the set that can be encoded back to octets
	i := int64(v.Data)
	if !IntInRuneDomain(i) {
		return rune(i), false
	}
	return rune(i), true
}

func intTypeAsByte(v Value) (byte, bool) {
	i := int64(v.Data)
	if i < 0 || i > math.MaxUint8 {
		return byte(i), false
	}
	return byte(i), true
}

func intTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Int, value.Rune, value.Byte, value.Bool:
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
func intTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
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
	case value.Int:
		// int is a CHECKED numeric: overflow raises, catchable — wide modular
		// arithmetic left the language (byte is the only modular type)
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case tokens.Add:
			if s, ok := IntAddChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Sub:
			if s, ok := IntSubChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Mul:
			if s, ok := IntMulChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Quo:
			if r == 0 {
				return Undefined, errs.NewDivisionByZeroError()
			}
			if l == math.MinInt64 && r == -1 {
				return Undefined, errs.NewInvalidValueError("int overflow")
			}
			return IntValue(l / r), nil
		case tokens.Rem:
			if r == 0 {
				return Undefined, errs.NewDivisionByZeroError()
			}
			return IntValue(l % r), nil
		case tokens.And:
			return IntValue(l & r), nil
		case tokens.Or:
			return IntValue(l | r), nil
		case tokens.Xor:
			return IntValue(l ^ r), nil
		case tokens.AndNot:
			return IntValue(l &^ r), nil
		case tokens.Shl:
			if s, ok := IntShlChecked(l, r); ok {
				return IntValue(s), nil
			}
			return Undefined, errs.NewInvalidValueError("int overflow")
		case tokens.Shr:
			if r < 0 {
				return Undefined, errs.NewInvalidValueError("int overflow")
			}
			if r >= 64 {
				// arithmetic shift saturates at the sign bit — no information invented
				return IntValue(l >> 63), nil
			}
			return IntValue(l >> uint64(r)), nil
		case tokens.Less:
			return BoolValue(l < r), nil
		case tokens.Greater:
			return BoolValue(l > r), nil
		case tokens.LessEq:
			return BoolValue(l <= r), nil
		case tokens.GreaterEq:
			return BoolValue(l >= r), nil
		}

	case value.Float:
		l := float64(int64(v.Data))
		r := math.Float64frombits(other.Data)
		switch op {
		case tokens.Add:
			return floatArithResult(l + r)
		case tokens.Sub:
			return floatArithResult(l - r)
		case tokens.Mul:
			return floatArithResult(l * r)
		case tokens.Quo:
			return floatArithResult(l / r)
		case tokens.Rem:
			return floatArithResult(math.Mod(l, r))
		case tokens.Less, tokens.Greater, tokens.LessEq, tokens.GreaterEq:
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(int64(v.Data)), r)
			return exactOrderFloat(cmp, op)
		}

	case value.Decimal:
		l := dec128.FromInt64(int64(v.Data))
		r := *(*dec128.Dec128)(other.Ptr)
		switch op {
		case tokens.Add:
			return decimalArithResult(l.Add(r), op, r.IsZero())
		case tokens.Sub:
			return decimalArithResult(l.Sub(r), op, r.IsZero())
		case tokens.Mul:
			return decimalArithResult(l.Mul(r), op, r.IsZero())
		case tokens.Quo:
			return decimalArithResult(l.Div(r), op, r.IsZero())
		case tokens.Rem:
			return decimalArithResult(l.Mod(r), op, r.IsZero())
		case tokens.Less:
			return BoolValue(l.LessThan(r)), nil
		case tokens.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case tokens.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case tokens.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
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

// PURE by contract
// IntAddChecked / IntSubChecked / IntMulChecked / IntShlChecked are int's checked arithmetic — ok=false means
// the mathematical result does not fit int64 and the operation must raise instead of wrapping. Shared by the
// operator implementations here and the VM's integer fast paths.
func IntAddChecked(l, r int64) (int64, bool) {
	s := l + r
	// overflow iff the operands share a sign and the sum does not
	return s, (l >= 0) != (r >= 0) || (s >= 0) == (l >= 0)
}

func IntSubChecked(l, r int64) (int64, bool) {
	s := l - r
	return s, (l >= 0) == (r >= 0) || (s >= 0) == (l >= 0)
}

func IntMulChecked(l, r int64) (int64, bool) {
	if l == 0 || r == 0 {
		return 0, true
	}
	if l == math.MinInt64 || r == math.MinInt64 {
		if l == 1 {
			return r, true
		}
		if r == 1 {
			return l, true
		}
		return 0, false
	}
	p := l * r
	return p, p/r == l
}

func IntShlChecked(l, r int64) (int64, bool) {
	if r < 0 || r >= 64 {
		return 0, false
	}
	s := l << uint64(r)
	return s, s>>uint64(r) == l
}

func intTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case tokens.Sub: // see also fast track in VM OpMinus
		i := int64(v.Data)
		if i == math.MinInt64 {
			return Undefined, errs.NewInvalidValueError("int overflow")
		}
		return IntValue(-i), nil

	case tokens.Xor: // see also fast track in VM OpBComplement
		i := int64(v.Data)
		return IntValue(^i), nil
	}

	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// intTypeAsDate reads an int as epoch days (civil.FromDays), failing outside 0001-01-01…9999-12-31.
//
// PURE by contract
func intTypeAsDate(v Value) (civil.Date, bool) {
	n := int64(v.Data)
	if n < -1<<31 || n > 1<<31-1 {
		return civil.Date{}, false
	}
	return civil.FromDays(int32(n))
}
