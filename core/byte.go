package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const byteTypeName = "byte"

func ByteValue(v byte) Value {
	return Value{
		Type:      value.Byte,
		Immutable: true,
		Data:      uint64(v),
	}
}

var TypeByte = ValueTypeDescr{
	Name:         ConstHook(byteTypeName),                                                              // PURE by contract
	String:       func(v Value) string { return fmt.Sprintf("byte(%d)", v.Data) },                      // PURE by contract
	Format:       byteTypeFormat,                                                                       // PURE by contract
	Interface:    func(v Value) any { return byte(v.Data) },                                            // PURE by contract
	EncodeJSON:   byteTypeEncodeJSON,                                                                   // PURE by contract
	EncodeBinary: byteTypeEncodeBinary,                                                                 // PURE by contract
	DecodeBinary: byteTypeDecodeBinary,                                                                 // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) bool { return v.Data != 0 },                                            // PURE by contract
	Equal:        byteTypeEqual,                                                                        // PURE by contract
	Len:          ConstHook(int64(1)),                                                                  // PURE by contract
	UnaryOp:      byteTypeUnaryOp,                                                                      // PURE by contract
	BinaryOp:     byteTypeBinaryOp,                                                                     // PURE by contract
	MethodCall:   byteTypeMethodCall,                                                                   // PURE by contract with higher-order rule caveat (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return strconv.FormatInt(int64(v.Data), 10), true },   // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                           // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                              // PURE by contract
	AsRune:       func(v Value) (rune, bool) { return rune(v.Data), true },                             // PURE by contract
	AsByte:       func(v Value) (byte, bool) { return byte(v.Data), true },                             // PURE by contract
	AsFloat:      func(v Value) (float64, bool) { return float64(int64(v.Data)), true },                // PURE by contract
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return dec128.FromInt64(int64(v.Data)), true }, // PURE by contract
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
		return "", fmt.Errorf("%w: ',' grouping is only supported with decimal verb 'd'; use '_' for base-2/8/16",
			errs.ErrUnsupportedFormatSpec)
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

func byteTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsByte()
	if !ok {
		return false
	}
	return byte(v.Data) == r
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func byteTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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
		return NewDecimalValue(d), nil

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
		return NewStringValue(s), nil

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
		s, err := byteTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		bs := make([]byte, n)
		b := byte(v.Data)
		for i := range n {
			bs[i] = b
		}
		return NewBytesValue(bs, false), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		elems, err := resolveJoinSeq(args[0], name)
		if err != nil {
			return Undefined, err
		}
		s, err := joinElementsToString(elems, string([]byte{byte(v.Data)}))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, byteTypeName)
	}
}

// PURE by contract
func byteTypeUnaryOp(v Value, op token.Token) (Value, error) {
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

// PURE by contract
func byteTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
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
