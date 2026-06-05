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

const runeTypeName = "rune"

// RuneValue creates new rune value.
func RuneValue(c rune) Value {
	return Value{
		Type:      VT_RUNE,
		Immutable: true,
		Data:      uint64(c),
	}
}

var TypeRune = ValueType{
	Name:         ConstHook(runeTypeName),
	String:       func(_ *Arena, v Value) string { return fmt.Sprintf("%q", rune(v.Data)) },
	Format:       runeTypeFormat,
	Interface:    func(_ *Arena, v Value) any { return rune(v.Data) },
	EncodeJSON:   runeTypeEncodeJSON,
	EncodeBinary: runeTypeEncodeBinary,
	DecodeBinary: runeTypeDecodeBinary,
	IsTrue:       func(_ *Arena, v Value) bool { return v.Data != 0 },
	Equal:        runeTypeEqual,
	Len:          ConstHook(int64(1)),
	BinaryOp:     runeTypeBinaryOp,
	MethodCall:   runeTypeMethodCall,
	AsString:     func(_ *Arena, v Value) (string, bool) { return string(rune(v.Data)), true },
	AsInt:        func(_ *Arena, v Value) (int64, bool) { return int64(v.Data), true },
	AsBool:       func(_ *Arena, v Value) (bool, bool) { return v.Data != 0, true },
	AsRune:       func(_ *Arena, v Value) (rune, bool) { return rune(v.Data), true },
	AsByte:       runeTypeAsByte,
}

func runeTypeEncodeJSON(_ *Arena, v Value) ([]byte, error) {
	c := rune(v.Data)
	s := strconv.FormatInt(int64(c), 10)
	return []byte(s), nil
}

func runeTypeEncodeBinary(_ *Arena, v Value) ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v.Data))
	return b, nil
}

func runeTypeDecodeBinary(_ *Arena, v *Value, data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("rune: expected 4 bytes, got %d", len(data))
	}
	v.Data = uint64(binary.BigEndian.Uint32(data))
	return nil
}

func runeTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return fmt.Sprintf("%q", rune(v.Data)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(runeTypeName, sp, fspec.AlignLeft), nil
	}

	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}

	if sp.HasPrec || sp.CoerceZero || sp.Bare {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}

	r := rune(v.Data)
	verb := sp.Verb
	if verb == 0 {
		verb = 'c'
	}

	switch verb {
	case 'c':
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
		}
		return fspec.ApplyGenerics(string(r), sp, fspec.AlignLeft), nil

	case 'q':
		if sp.Sign != fspec.SignDefault || sp.Grouping != 0 || sp.ZeroPad {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
		}
		return fspec.ApplyGenerics(strconv.QuoteRune(r), sp, fspec.AlignLeft), nil

	case 'd':
		if sp.Grouping == ',' || sp.Grouping == '_' || sp.Grouping == 0 {
			// fine
		} else {
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
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
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
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
			return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
		}
		digits := strings.ToUpper(strconv.FormatUint(uint64(uint32(r)), 16))
		if len(digits) < 4 {
			digits = strings.Repeat("0", 4-len(digits)) + digits
		}
		return fspec.ApplyGenerics("U+"+digits, sp, fspec.AlignRight), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(a), sp)
	}
}

func runeTypeAsByte(_ *Arena, v Value) (byte, bool) {
	c := rune(v.Data)
	if c > 255 {
		return byte(c), false
	}
	return byte(c), true
}

func runeTypeEqual(a *Arena, v Value, rhs Value) bool {
	r, ok := rhs.AsRune(a)
	if !ok {
		return false
	}
	return rune(v.Data) == r
}

func runeTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
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
		return BoolValue(v.Data != 0), nil

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
		b, _ := runeTypeAsByte(a, v)
		return ByteValue(b), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return a.NewStringValue(string(rune(v.Data))), nil

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := runeTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s), nil

	case "repeat":
		n, err := parseRepeatCount(a, name, args)
		if err != nil {
			return Undefined, err
		}
		rs := a.NewRunes(n, true)
		r := rune(v.Data)
		for i := range n {
			rs[i] = r
		}
		return a.NewRunesValue(rs, false), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		elems, err := resolveJoinSeq(a, args[0], name)
		if err != nil {
			return Undefined, err
		}
		s, err := joinElementsToString(a, elems, string(rune(v.Data)))
		if err != nil {
			return Undefined, err
		}
		return a.NewRunesValue([]rune(s), false), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, runeTypeName)
	}
}

func runeTypeBinaryOp(a *Arena, v Value, rhs Value, op token.Token) (Value, error) {
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
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
		}

	case VT_STRING: // rune op string => string
		l := string(rune(v.Data))
		r := *(*string)(rhs.Ptr)
		switch op {
		case token.Add:
			return a.NewStringValue(l + r), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
		}

	default:
		// rune op any => rune
		r, ok := rhs.AsRune(a)
		if !ok {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
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
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), rhs.TypeName(a))
		}
	}
}
