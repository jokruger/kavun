package core

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/token"
)

// RuneValue creates new rune value.
func RuneValue(c rune) Value {
	return Value{
		Type:  VT_RUNE,
		Const: true,
		Data:  uint64(c),
	}
}

/* Rune type methods */

func runeTypeName(v Value) string {
	return "rune"
}

func runeTypeEncodeJSON(v Value) ([]byte, error) {
	c := rune(v.Data)
	s := strconv.FormatInt(int64(c), 10)
	return []byte(s), nil
}

func runeTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v.Data))
	return b, nil
}

func runeTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("rune: expected 4 bytes, got %d", len(data))
	}
	v.Data = uint64(binary.BigEndian.Uint32(data))
	return nil
}

func runeTypeString(v Value) string {
	return fmt.Sprintf("%q", rune(v.Data))
}

func runeTypeFormat(v Value, s fspec.FormatSpec) (string, error) {
	if s.Verb == 'v' {
		return v.String(), nil
	}
	if s.HasPrec || s.CoerceZero {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}

	r := rune(v.Data)
	verb := s.Verb
	if verb == 0 {
		verb = 'c'
	}

	switch verb {
	case 'c':
		if s.Sign != fspec.SignDefault || s.Grouping != 0 || s.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		return fspec.ApplyGenerics(string(r), s, fspec.AlignLeft), nil

	case 'q':
		if s.Sign != fspec.SignDefault || s.Grouping != 0 || s.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(r), s, fspec.AlignLeft), nil

	case 'd':
		if s.Grouping == ',' || s.Grouping == '_' || s.Grouping == 0 {
			// fine
		} else {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		negative := r < 0
		var digits string
		if negative {
			digits = strconv.FormatUint(uint64(-int64(r)), 10)
		} else {
			digits = strconv.FormatUint(uint64(r), 10)
		}
		if s.Grouping != 0 {
			digits = fspec.GroupDigits(digits, s.Grouping, 3)
		}
		sign := fspec.SignPrefix(s.Sign, negative)
		if negative {
			sign = "-"
		}
		body := sign + digits
		return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil

	case 'x', 'X':
		if s.Grouping == ',' {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		// per docs: rune hex has no "0x" prefix (unlike int/byte).
		digits := strconv.FormatUint(uint64(uint32(r)), 16)
		if verb == 'X' {
			digits = strings.ToUpper(digits)
		}
		if s.Grouping == '_' {
			digits = fspec.GroupDigits(digits, '_', 4)
		}
		body := fspec.SignPrefix(s.Sign, false) + digits
		return fspec.ApplyGenerics(body, s, fspec.AlignRight), nil

	case 'U':
		if s.Sign != fspec.SignDefault || s.Grouping != 0 || s.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
		}
		digits := strings.ToUpper(strconv.FormatUint(uint64(uint32(r)), 16))
		if len(digits) < 4 {
			digits = strings.Repeat("0", 4-len(digits)) + digits
		}
		return fspec.ApplyGenerics("U+"+digits, s, fspec.AlignRight), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), s)
	}
}

func runeTypeInterface(v Value) any {
	return rune(v.Data)
}

func runeTypeIsTrue(v Value) bool {
	return v.Data != 0
}

func runeTypeAsInt(v Value) (int64, bool) {
	return int64(v.Data), true
}

func runeTypeAsString(v Value) (string, bool) {
	return string(rune(v.Data)), true
}

func runeTypeAsBool(v Value) (bool, bool) {
	return v.Data != 0, true
}

func runeTypeAsRune(v Value) (rune, bool) {
	return rune(v.Data), true
}

func runeTypeAsByte(v Value) (byte, bool) {
	c := rune(v.Data)
	if c > 255 {
		return byte(c), false
	}
	return byte(c), true
}

func runeTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsRune()
	if !ok {
		return false
	}
	return rune(v.Data) == r
}

func runeTypeMethodCall(v Value, vm VM, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "rune":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runeTypeAsBool(v)
		return BoolValue(b), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := int64(v.Data), true
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := runeTypeAsByte(v)
		return ByteValue(b), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := runeTypeAsString(v)
		return vm.Allocator().NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, "rune")
	}
}

func runeTypeBinaryOp(v Value, a *Arena, op token.Token, rhs Value) (Value, error) {
	switch rhs.Type {
	case VT_INT: // rune op int => int
		l := int64(v.Data)
		r := int64(rhs.Data)
		switch op {
		case token.Add:
			return IntValue(l + r), nil
		case token.Sub:
			return IntValue(l - r), nil
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

	case VT_STRING: // rune op string => string
		l := string(rune(v.Data))
		r, _ := stringTypeAsString(rhs)
		switch op {
		case token.Add:
			return a.NewStringValue(l + r), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// rune op any => rune
		r, ok := rhs.AsRune()
		if !ok {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := rune(v.Data)
		switch op {
		case token.Add:
			return RuneValue(l + r), nil
		case token.Sub:
			return RuneValue(l - r), nil
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
}
