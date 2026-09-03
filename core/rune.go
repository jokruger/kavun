package core

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const runeTypeName = "rune"

func RuneValue(c rune) Value {
	return Value{
		Type:      value.Rune,
		Immutable: true,
		Data:      uint64(c),
	}
}

var TypeRune = ValueTypeDescr{
	Name:         ConstHook(runeTypeName),                                                    // PURE by contract
	String:       func(v Value) string { return fmt.Sprintf("%q", rune(v.Data)) },            // PURE by contract
	Format:       runeTypeFormat,                                                             // PURE by contract
	Interface:    func(v Value) any { return rune(v.Data) },                                  // PURE by contract
	EncodeJSON:   runeTypeEncodeJSON,                                                         // PURE by contract
	EncodeBinary: runeTypeEncodeBinary,                                                       // PURE by contract
	DecodeBinary: runeTypeDecodeBinary,                                                       // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return v.Data != 0, nil },                    // PURE by contract
	Len:          ConstHook(int64(1)),                                                        // PURE by contract
	Equal:        runeTypeEqual,                                                              // PURE by contract
	BinaryOp:     runeTypeBinaryOp,                                                           // PURE by contract
	MethodCall:   runeTypeMethodCall,                                                         // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	AsString:     func(v Value) (string, bool) { return EncodeRuneText(rune(v.Data)), true }, // PURE by contract
	AsInt:        func(v Value) (int64, bool) { return int64(v.Data), true },                 // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return v.Data != 0, true },                    // PURE by contract
	AsRune:       func(v Value) (rune, bool) { return rune(v.Data), true },                   // PURE by contract
	AsByte:       runeTypeAsByte,                                                             // PURE by contract
	IsMethodPure: func(string) bool { return true },                                          // All methods are expected to be pure.
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
		return fspec.ApplyGenerics(string(r), sp, fspec.AlignLeft), nil

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
	if IsEscapeRune(c) {
		return EscapeRuneOctet(c), true
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
	if !IntInRuneDomain(r) {
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
			case token.Add:
				return runeArithResult(l + r)
			case token.Less:
				return BoolValue(l < r), nil
			case token.Greater:
				return BoolValue(l > r), nil
			case token.LessEq:
				return BoolValue(l <= r), nil
			case token.GreaterEq:
				return BoolValue(l >= r), nil
			}

		case value.Byte:
			l := int64(other.Data)
			r := int64(v.Data)
			switch op {
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
			}

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
	case value.Rune, value.Byte:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
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
		}

	case value.Int:
		l := int64(v.Data)
		r := int64(other.Data)
		switch op {
		case token.Add:
			return runeArithResult(l + r)
		case token.Sub:
			return runeArithResult(l - r)
		case token.Less:
			return BoolValue(l < r), nil
		case token.Greater:
			return BoolValue(l > r), nil
		case token.LessEq:
			return BoolValue(l <= r), nil
		case token.GreaterEq:
			return BoolValue(l >= r), nil
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

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func runeTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
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

	case "rune":
		return convMember(name, runeTypeName, args, true, v)

	case "int":
		return convMember(name, runeTypeName, args, true, IntValue(int64(v.Data)))

	case "byte":
		b, ok := v.AsByte()
		return convMember(name, runeTypeName, args, ok, ByteValue(b))

	case "string":
		// total — the default slot never fires, but every conversion carries it. An escape rune
		// answers the single octet it stands for, so it round-trips back to the data it came from
		return convMember(name, runeTypeName, args, true, NewStringValue(EncodeRuneText(rune(v.Data))))

	case "runes":
		// the text targets compose through string: 'A'.runes() ≡ 'A'.string().runes()
		return convMember(name, runeTypeName, args, true, NewRunesValue([]rune{rune(v.Data)}, false))

	case "is_valid":
		// a real symbol, as opposed to an escape standing for an octet that is not one
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(RuneIsValid(rune(v.Data))), nil

	case "is_ascii":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(rune(v.Data) < 0x80), nil

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
			return Undefined, errs.FromFormatSpecError(name, err)
		}
		s, err := runeTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	default:
		return Undefined, errs.NewInvalidMethodError(name, runeTypeName)
	}
}
