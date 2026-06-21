package core

import (
	"encoding/binary"
	"fmt"
	"math"
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

// IntValue creates new boxed int value.
func IntValue(i int64) Value {
	return Value{
		Type:      value.Int,
		Immutable: true,
		Data:      uint64(i),
	}
}

var TypeInt = ValueTypeDescr{
	Name:         ConstHook(intTypeName),
	String:       func(v Value) string { return strconv.FormatInt(int64(v.Data), 10) },
	Format:       intTypeFormat,
	Interface:    func(v Value) any { return int64(v.Data) },
	EncodeJSON:   intTypeEncodeJSON,
	EncodeBinary: intTypeEncodeBinary,
	DecodeBinary: intTypeDecodeBinary,
	IsTrue:       func(v Value) bool { return v.Data != 0 },
	Equal:        intTypeEqual,
	Len:          ConstHook(int64(1)),
	UnaryOp:      intTypeUnaryOp,
	BinaryOp:     intTypeBinaryOp,
	MethodCall:   intTypeMethodCall,
	AsString:     func(v Value) (string, bool) { return strconv.FormatInt(int64(v.Data), 10), true },
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },
	AsFloat:      func(v Value) (float64, bool) { return float64(int64(v.Data)), true },
	AsDecimal:    func(v Value) (dec128.Dec128, bool) { return dec128.FromInt64(int64(v.Data)), true },
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },
	AsRune:       intTypeAsRune,
	AsTime:       func(v Value) (time.Time, bool) { return time.Unix(int64(v.Data), 0), true },
	AsByte:       intTypeAsByte,
}

func intTypeEncodeJSON(v Value) ([]byte, error) {
	s := strconv.FormatInt(int64(v.Data), 10)
	return []byte(s), nil
}

func intTypeEncodeBinary(v Value) ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v.Data)
	return b, nil
}

func intTypeDecodeBinary(v *Value, data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("int: expected 8 bytes, got %d", len(data))
	}
	v.Data = binary.BigEndian.Uint64(data)
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
	i := int64(v.Data)
	if i < 0 || i > utf8.MaxRune {
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

func intTypeEqual(v Value, rhs Value) bool {
	r, ok := rhs.AsInt()
	if !ok {
		return false
	}
	return int64(v.Data) == r
}

func intTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := v.AsFloat(a)
		return FloatValue(f), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := v.AsDecimal(a)
		return a.NewDecimalValue(d)

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
		c, _ := v.AsRune(a)
		return RuneValue(c), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := v.AsByte(a)
		return ByteValue(b), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		s, _ := v.AsString()
		return a.NewStringValue(s)

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := v.AsTime(a)
		return a.NewTimeValue(t)

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
		s, err := intTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

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

	case "repeat":
		return repeatScalarToArray(a, v, name, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, intTypeName)
	}
}

func intTypeUnaryOp(v Value, op token.Token) (Value, error) {
	i := int64(v.Data)
	switch op {
	case token.Sub: // see also fast track in VM OpMinus
		return IntValue(-i), nil

	case token.Xor: // see also fast track in VM OpBComplement
		return IntValue(^i), nil

	default:
		return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
	}
}

func intTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	// see also int/int fast track in VM OpBinaryOp

	switch rhs.Type {
	case value.Float: // int op float => float
		l := float64(int64(v.Data))
		r := math.Float64frombits(rhs.Data)
		switch op {
		case token.Add:
			return FloatValue(l + r), nil
		case token.Sub:
			return FloatValue(l - r), nil
		case token.Mul:
			return FloatValue(l * r), nil
		case token.Quo:
			return FloatValue(l / r), nil
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

	case value.Decimal: // int op decimal => decimal
		l := dec128.FromInt64(int64(v.Data))
		r := *a.ResolveDecimalValue(rhs)
		switch op {
		case token.Add:
			return a.NewDecimalValue(l.Add(r))
		case token.Sub:
			return a.NewDecimalValue(l.Sub(r))
		case token.Mul:
			return a.NewDecimalValue(l.Mul(r))
		case token.Quo:
			return a.NewDecimalValue(l.Div(r))
		case token.Less:
			return BoolValue(l.LessThan(r)), nil
		case token.Greater:
			return BoolValue(l.GreaterThan(r)), nil
		case token.LessEq:
			return BoolValue(l.LessThanOrEqual(r)), nil
		case token.GreaterEq:
			return BoolValue(l.GreaterThanOrEqual(r)), nil
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

	default:
		// int op any => int
		r, ok := rhs.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}

		l := int64(v.Data)
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
		default:
			return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
		}
	}
}
