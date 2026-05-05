package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// ByteValue creates new boxed byte value.
func ByteValue(v byte) Value {
	return Value{
		Type:  VT_BYTE,
		Const: true,
		Data:  uint64(v),
	}
}

/* Byte type methods */

func byteTypeName(v Value) string {
	return "byte"
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

func byteTypeString(v Value) string {
	return fmt.Sprintf("byte(%d)", v.Data)
}

func byteTypeFormat(v Value, s fspec.FormatSpec) (string, error) {
	if s.Verb == 'v' {
		return v.String(), nil
	}
	if s.HasPrec || s.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	n := uint64(byte(v.Data))
	verb := s.Verb
	if verb == 0 || verb == 'v' {
		verb = 'd'
	}

	// 'c' renders the byte as an ASCII character; only width/fill/align apply.
	if verb == 'c' {
		if s.Sign != fspec.SignDefault || s.Grouping != 0 || s.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		return fspec.ApplyGenerics(string(rune(n)), s, fspec.AlignLeft), nil
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
		prefix = "0X"
		groupEvery = 4
		upper = true
	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	// grouping rules: ',' is decimal-only; '_' allowed for any base.
	if s.Grouping == ',' && base != 10 {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	digits := strconv.FormatUint(n, base)
	if upper {
		digits = strings.ToUpper(digits)
	}
	if s.Grouping != 0 {
		digits = fspec.GroupDigits(digits, s.Grouping, groupEvery)
	}

	body := fspec.SignPrefix(s.Sign, false) + prefix + digits
	return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil
}

func byteTypeInterface(v Value) any {
	return byte(v.Data)
}

func byteTypeIsTrue(v Value) bool {
	return v.Data != 0
}

func byteTypeAsInt(v Value) (int64, bool) {
	return int64(v.Data), true
}

func byteTypeAsString(v Value) (string, bool) {
	return strconv.FormatInt(int64(v.Data), 10), true
}

func byteTypeAsFloat(v Value) (float64, bool) {
	return float64(int64(v.Data)), true
}

func byteTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	return dec128.FromInt64(int64(v.Data)), true
}

func byteTypeAsBool(v Value) (bool, bool) {
	return v.Data != 0, true
}

func byteTypeAsByte(v Value) (byte, bool) {
	return byte(v.Data), true
}

func byteTypeAsRune(v Value) (rune, bool) {
	return rune(v.Data), true
}

func byteTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsByte()
	if !ok {
		return false
	}
	return byte(v.Data) == r
}

func byteTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := v.AsInt()
		return IntValue(i), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := v.AsFloat()
		return FloatValue(f), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := v.AsDecimal()
		r := vm.Allocator().NewDecimal()
		*r = d
		return DecimalValue(r), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := v.AsBool()
		return BoolValue(b), nil

	case "rune":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		c, _ := v.AsRune()
		return RuneValue(c), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return vm.Allocator().NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "byte")
	}
}

func byteTypeUnaryOp(v Value, a *Arena, op token.Token) (Value, error) {
	i := byte(v.Data)
	switch op {
	case token.Sub:
		return ByteValue(-i), nil

	case token.Xor:
		return ByteValue(^i), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func byteTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	// byte op any => byte
	r, ok := rhs.AsByte()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := byte(v.Data)
	switch op {
	case token.Add:
		return ByteValue(l + r), nil
	case token.Sub:
		return ByteValue(l - r), nil
	case token.And:
		return ByteValue(l & r), nil
	case token.Or:
		return ByteValue(l | r), nil
	case token.Xor:
		return ByteValue(l ^ r), nil
	case token.AndNot:
		return ByteValue(l &^ r), nil
	case token.Shl:
		return ByteValue(l << uint64(r)), nil
	case token.Shr:
		return ByteValue(l >> uint64(r)), nil
	case token.Less:
		return BoolValue(l < r), nil
	case token.Greater:
		return BoolValue(l > r), nil
	case token.LessEq:
		return BoolValue(l <= r), nil
	case token.GreaterEq:
		return BoolValue(l >= r), nil
	default:
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}
}
