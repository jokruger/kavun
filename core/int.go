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

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/token"
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
	BinaryOp:     intTypeBinaryOp,                                                                      // PURE by contract
	UnaryOp:      intTypeUnaryOp,                                                                       // PURE by contract
	MethodCall:   intTypeMethodCall,                                                                    // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return strconv.FormatInt(int64(v.Data), 10), true },   // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                           // PURE by contract
	AsFloat:      func(v Value) (float64, bool) { return float64(int64(v.Data)), true },                // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return dec128.FromInt64(int64(v.Data)), true }, // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                              // PURE by contract
	AsRune:       intTypeAsRune,                                                                        // PURE by contract
	AsTime:       func(v Value) (time.Time, bool) { return time.Unix(int64(v.Data), 0).UTC(), true },   // PURE by contract
	AsByte:       intTypeAsByte,                                                                        // PURE by contract
	IsMethodPure: func(string) bool { return true },                                                    // All methods are expected to be pure.
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
		return "", fmt.Errorf("%w: ',' grouping is only supported with decimal verb 'd'; use '_' for base-2/8/16",
			errs.ErrUnsupportedFormatSpec)
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
	// a rune is a Unicode scalar value: surrogates are excluded by the same
	// range rule as negatives and values past MaxRune, not as a special case
	i := int64(v.Data)
	if i < 0 || i > utf8.MaxRune || (i >= 0xD800 && i <= 0xDFFF) {
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
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Int:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case token.Add:
			return IntValue(l + r), nil
		case token.Sub:
			return IntValue(l - r), nil
		case token.Mul:
			return IntValue(l * r), nil
		case token.Quo:
			if r == 0 {
				return Undefined, errs.ErrDivisionByZero
			}
			return IntValue(l / r), nil
		case token.Rem:
			return IntValue(l % r), nil
		case token.And:
			return IntValue(l & r), nil
		case token.Or:
			return IntValue(l | r), nil
		case token.Xor:
			return IntValue(l ^ r), nil
		case token.AndNot:
			return IntValue(l &^ r), nil
		case token.Shl:
			return IntValue(l << uint64(r)), nil
		case token.Shr:
			return IntValue(l >> uint64(r)), nil
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		}

	case value.Float:
		l := float64(int64(v.Data))
		r := math.Float64frombits(other.Data)
		switch op {
		case token.Add:
			return floatArithResult(l + r)
		case token.Sub:
			return floatArithResult(l - r)
		case token.Mul:
			return floatArithResult(l * r)
		case token.Quo:
			return floatArithResult(l / r)
		case token.Rem:
			return floatArithResult(math.Mod(l, r))
		case token.Less, token.Greater, token.LessEq, token.GreaterEq:
			cmp := compareExactAndFloat(new(big.Rat).SetInt64(int64(v.Data)), r)
			return exactOrderFloat(cmp, op)
		}

	case value.Decimal:
		l := dec128.FromInt64(int64(v.Data))
		r := *(*dec128.Dec128)(other.Ptr)
		switch op {
		case token.Add:
			return NewDecimalValue(l.Add(r)), nil
		case token.Sub:
			return NewDecimalValue(l.Sub(r)), nil
		case token.Mul:
			return NewDecimalValue(l.Mul(r)), nil
		case token.Quo:
			return NewDecimalValue(l.Div(r)), nil
		case token.Rem:
			return NewDecimalValue(l.Mod(r)), nil
		case token.Less:
			return BoolValue(l.LessThan(r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		}

	case value.Bool:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract
func intTypeUnaryOp(v Value, op token.Token) (Value, error) {
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		i := int64(v.Data)
		return IntValue(-i), nil

	case token.Xor: // see also fast track in VM OpBComplement
		i := int64(v.Data)
		return IntValue(^i), nil
	}

	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func intTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value regardless of copy depth
		return v, nil

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable already, so freeze/freeze_shallow are no-ops
		return v, nil

	case "int":
		return convMember(name, intTypeName, args, true, v)

	case "float":
		f, ok := v.AsFloat()
		return convMember(name, intTypeName, args, ok, FloatValue(f))

	case "decimal":
		d, ok := v.AsDecimal()
		return convMember(name, intTypeName, args, ok, NewDecimalValue(d))

	case "bool":
		b, ok := v.AsBool()
		return convMember(name, intTypeName, args, ok, BoolValue(b))

	case "rune":
		c, ok := v.AsRune()
		return convMember(name, intTypeName, args, ok, RuneValue(c))

	case "byte":
		b, ok := v.AsByte()
		return convMember(name, intTypeName, args, ok, ByteValue(b))

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return NewStringValue(s), nil

	case "runes":
		s, ok := v.AsString()
		return convMember(name, intTypeName, args, ok, NewRunesValue([]rune(s), false))

	// The int -> time family. In conversion context an int is a unix timestamp, never a duration
	// (that reading belongs to operator context — `t + n` is nanoseconds; see docs/types/time.md).
	// Each of these names the encoding it reads, and each is the exact inverse of the time accessor
	// with the matching suffix: time_ms <-> unix_ms, time_micro <-> unix_micro, time_nano <->
	// unix_nano, and the unsuffixed time() <-> int()/unix(), which are seconds. All produce UTC, so
	// the result never depends on the host's timezone.
	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := v.AsTime()
		return NewTimeValue(t), nil

	case "time_ms":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewTimeValue(time.UnixMilli(int64(v.Data)).UTC()), nil

	case "time_micro":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewTimeValue(time.UnixMicro(int64(v.Data)).UTC()), nil

	case "time_nano":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewTimeValue(time.Unix(0, int64(v.Data)).UTC()), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := intTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_nan":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return False, nil

	case "is_inf":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return False, nil

	case "sign":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Data == 0 {
			return IntValue(0), nil
		} else if int64(v.Data) > 0 {
			return IntValue(1), nil
		} else {
			return IntValue(-1), nil
		}

	case "abs":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i := int64(v.Data)
		if i < 0 {
			return IntValue(-i), nil
		}
		return v, nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, intTypeName)
	}
}
