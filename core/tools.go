package core

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

// NormalizeIndex normalizes index (-1 = last element, -2 = second to last, etc.) and checks if it's within bounds.
func NormalizeIndex(index int64, length int64) (int64, bool) {
	if index < 0 {
		index += length
	}
	if index < 0 || index >= length {
		return index, false
	}
	return index, true
}

// NormalizeSliceBounds normalizes slice bounds (negative values count from the end, missing start defaults to 0,
// missing end defaults to length) and clamps them to [0, length]. If start > end after normalization, start is set to
// end.
func NormalizeSliceBounds(start int64, hasStart bool, end int64, hasEnd bool, length int64) (int64, int64) {
	if !hasStart {
		start = 0
	} else if start < 0 {
		start += length
	}

	if !hasEnd {
		end = length
	} else if end < 0 {
		end += length
	}

	if start < 0 {
		start = 0
	} else if start > length {
		start = length
	}

	if end < 0 {
		end = 0
	} else if end > length {
		end = length
	}

	if start > end {
		start = end
	}

	return start, end
}

// NormalizeSliceBoundsStep returns the effective start and end for a step-based slice.
// Caller must ensure step != 0. For step > 0 the iteration is start..end (exclusive).
// For step < 0 the iteration is start..end (exclusive, with end possibly -1 to include index 0).
func NormalizeSliceBoundsStep(si int64, hasStart bool, ei int64, hasEnd bool, step int64, length int64) (int64, int64) {
	var start, end int64
	if step > 0 {
		if !hasStart {
			start = 0
		} else {
			start = si
			if start < 0 {
				start += length
			}
			if start < 0 {
				start = 0
			} else if start > length {
				start = length
			}
		}
		if !hasEnd {
			end = length
		} else {
			end = ei
			if end < 0 {
				end += length
			}
			if end < 0 {
				end = 0
			} else if end > length {
				end = length
			}
		}
	} else {
		// step < 0: lower bound is -1, upper bound is length-1
		if !hasStart {
			start = length - 1
		} else {
			start = si
			if start < 0 {
				start += length
			}
			if start < -1 {
				start = -1
			} else if start >= length {
				start = length - 1
			}
		}
		if !hasEnd {
			end = -1
		} else {
			end = ei
			if end < 0 {
				end += length
			}
			if end < -1 {
				end = -1
			} else if end >= length {
				end = length - 1
			}
		}
	}
	return start, end
}

// ForEachCallback validates that the only argument is a callback (non-variadic function of arity 1 or 2) and returns it
// as a Value.
func ForEachCallback(args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("for_each", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("for_each", "first", "non-variadic function", fn.TypeName())
	}
	if arity := fn.Arity(); arity != 1 && arity != 2 {
		return Undefined, errs.NewInvalidArgumentTypeError("for_each", "first", "f/1 or f/2", fn.TypeName())
	}

	return fn, nil
}

// parseRepeatCount validates and extracts the count argument for a `repeat` method.
// It expects exactly one int argument and returns an error if the count is negative.
func parseRepeatCount(name string, args []Value) (int, error) {
	if len(args) != 1 {
		return 0, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	n, ok := args[0].AsInt()
	if !ok {
		return 0, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
	}
	if n < 0 {
		return 0, fmt.Errorf("repeat count must be non-negative, got %d", n)
	}
	return int(n), nil
}

// repeatScalarToArray builds a new array containing n copies of v.
// Used by scalar value types (int, bool, float, decimal, time, undefined)
// whose `repeat(n)` lifts the value into an array.
func repeatScalarToArray(v Value, name string, args []Value) (Value, error) {
	n, err := parseRepeatCount(name, args)
	if err != nil {
		return Undefined, err
	}
	arr := make([]Value, n)
	for i := range n {
		arr[i] = v
	}
	return NewArrayValue(arr, false), nil
}

// joinElementsToString stringifies each element via AsString (the same coercion used by the `+` operator) and joins
// them with `sep`.
func joinElementsToString(elems []Value, sep string) (string, error) {
	if len(elems) == 0 {
		return "", nil
	}
	parts := make([]string, len(elems))
	total := 0
	for i, e := range elems {
		s, ok := e.AsString()
		if !ok {
			return "", fmt.Errorf("cannot convert %s to string", e.TypeName())
		}
		parts[i] = s
		total += len(s)
	}
	if len(elems) > 1 {
		total += (len(elems) - 1) * len(sep)
	}
	var b strings.Builder
	b.Grow(total)
	for i, p := range parts {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(p)
	}
	return b.String(), nil
}

// resolveJoinSeq returns the array of values to be joined for the given seq value.
// `seq` must be array or int_range; otherwise an error is returned.
func resolveJoinSeq(seq Value, name string) ([]Value, error) {
	switch seq.Type {
	case value.Array:
		return (*Array)(seq.Ptr).Elements, nil
	case value.IntRange:
		arr, _ := intRangeTypeAsArray(seq)
		return arr, nil
	default:
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "array or range", seq.TypeName())
	}
}

// joinSeqValueWithSepString joins the elements of a seq value (array or range) using a given string separator and
// returns a string value.
func joinSeqValueWithSepString(seq Value, sep string, name string) (Value, error) {
	elems, err := resolveJoinSeq(seq, name)
	if err != nil {
		return Undefined, err
	}
	s, err := joinElementsToString(elems, sep)
	if err != nil {
		return Undefined, err
	}
	return NewStringValue(s), nil
}

// coerceSepToString converts the separator argument of split/partition to a
// Go string. Accepted types: string, runes, byte, rune.
func coerceSepToString(name string, sep Value) (string, error) {
	switch sep.Type {
	case value.String:
		return *(*string)(sep.Ptr), nil
	case value.Runes:
		return string((*Runes)(sep.Ptr).Elements), nil
	case value.Byte:
		return string([]byte{byte(sep.Data)}), nil
	case value.Rune:
		return string(rune(sep.Data)), nil
	default:
		return "", errs.NewInvalidArgumentTypeError(name, "first", "string, runes, byte or rune", sep.TypeName())
	}
}

// coerceSepToBytes converts the separator argument of split/partition to a
// []byte. Accepted types: bytes, byte, string, rune.
func coerceSepToBytes(name string, sep Value) ([]byte, error) {
	switch sep.Type {
	case value.Bytes:
		return (*Bytes)(sep.Ptr).Elements, nil
	case value.Byte:
		return []byte{byte(sep.Data)}, nil
	case value.String:
		return []byte(*(*string)(sep.Ptr)), nil
	case value.Rune:
		return []byte(string(rune(sep.Data))), nil
	default:
		return nil, errs.NewInvalidArgumentTypeError(name, "first", "bytes, byte, string or rune", sep.TypeName())
	}
}

// parseSplitLimit returns the limit argument for split. -1 means unlimited.
// 0 means no splits at all (return receiver as a single piece).
func parseSplitLimit(name string, args []Value, idx int) (int, error) {
	n, ok := args[idx].AsInt()
	if !ok {
		return 0, errs.NewInvalidArgumentTypeError(name, "second", "int", args[idx].TypeName())
	}
	if n < 0 {
		return -1, nil
	}
	return int(n), nil
}

// splitStringByLiteral splits s by sep with at most limit splits.
// limit == -1 means unlimited. sep must be non-empty. Empty s yields nil.
func splitStringByLiteral(s, sep string, limit int) []string {
	if len(s) == 0 {
		return nil
	}
	if limit == 0 {
		return []string{s}
	}
	if limit < 0 {
		return strings.Split(s, sep)
	}
	return strings.SplitN(s, sep, limit+1)
}

// splitStringWhitespace splits s on runs of Unicode whitespace, dropping empty
// pieces. Equivalent to strings.Fields.
func splitStringWhitespace(s string) []string {
	return strings.Fields(s)
}

// splitBytesByLiteral splits bs by sep with at most limit splits.
// limit == -1 means unlimited. sep must be non-empty. Empty bs yields nil.
func splitBytesByLiteral(bs, sep []byte, limit int) [][]byte {
	if len(bs) == 0 {
		return nil
	}
	if limit == 0 {
		return [][]byte{bs}
	}
	if limit < 0 {
		return bytes.Split(bs, sep)
	}
	return bytes.SplitN(bs, sep, limit+1)
}

// splitBytesWhitespace splits bs on runs of ASCII whitespace, dropping empty
// pieces. Equivalent to bytes.Fields.
func splitBytesWhitespace(bs []byte) [][]byte {
	return bytes.Fields(bs)
}

// splitLinesString splits s on \n, \r\n or \r. A trailing line terminator
// does not produce an extra empty trailing element. Empty s yields nil.
func splitLinesString(s string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, 8)
	i := 0
	start := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case '\n':
			out = append(out, s[start:i])
			i++
			start = i
		case '\r':
			out = append(out, s[start:i])
			i++
			if i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
		default:
			i++
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// splitLinesBytes is the []byte counterpart of splitLinesString.
func splitLinesBytes(bs []byte) [][]byte {
	if len(bs) == 0 {
		return nil
	}
	out := make([][]byte, 0, 8)
	i := 0
	start := 0
	for i < len(bs) {
		c := bs[i]
		switch c {
		case '\n':
			out = append(out, bs[start:i])
			i++
			start = i
		case '\r':
			out = append(out, bs[start:i])
			i++
			if i < len(bs) && bs[i] == '\n' {
				i++
			}
			start = i
		default:
			i++
		}
	}
	if start < len(bs) {
		out = append(out, bs[start:])
	}
	return out
}

// PURE by contract
func defaultFormat(v Value, _ fspec.FormatSpec) (string, error) {
	return "", errs.NewNoFormattingError(v.TypeName())
}

// PURE by contract
func defaultUnaryOp(v Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

// PURE by contract
func defaultBinaryOp(v Value, r Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func defaultMethodCall(_ VM, v Value, name string, _ []Value) (Value, error) {
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

// IMPURE by contract (mutates target)
func defaultDelete(v Value, _ Value) (Value, error) {
	return Undefined, errs.NewNotDeletableError(v.TypeName())
}

// PURE by contract
func defaultAccess(v Value, _ Value, _ bc.Opcode) (Value, error) {
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

// IMPURE by contract (may mutate target)
func defaultAppend(v Value, _ []Value) (Value, error) {
	return Undefined, errs.NewNotAppendableError(v.TypeName())
}

// PURE by contract
func defaultSlice(v Value, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

// PURE by contract
func defaultSliceStep(v Value, _, _, _ Value) (Value, error) {
	return Undefined, errs.NewNotSliceableError(v.TypeName())
}

func defaultCall(_ VM, v Value, _ []Value) (Value, error) {
	return Undefined, errs.NewNotCallableError(v.TypeName())
}

// PURE by contract
func defaultAsRunes(v Value) ([]rune, bool) {
	s, ok := v.AsString()
	if !ok {
		return nil, false
	}
	return []rune(s), true
}

// EncodeString encodes given string as JSON string according to
// https://www.json.org/img/string.png
// Implementation is inspired by https://github.com/json-iterator/go
func EncodeString(b []byte, val string) []byte {
	valLen := len(val)
	buf := bytes.NewBuffer(b)
	buf.WriteByte('"')

	// write string, the fast path, without utf8 and escape support
	i := 0
	for ; i < valLen; i++ {
		c := val[i]
		if c > 31 && c != '"' && c != '\\' {
			buf.WriteByte(c)
		} else {
			break
		}
	}
	if i == valLen {
		buf.WriteByte('"')
		return buf.Bytes()
	}
	encodeStringSlowPath(buf, i, val, valLen)
	buf.WriteByte('"')
	return buf.Bytes()
}

// encodeStringSlowPath is ported from Go 1.14.2 encoding/json package.
// U+2028 U+2029 JSONP security holes can be fixed with addition call to
// json.html_escape() thus it is removed from the implementation below.
// Note: Invalid runes are not checked as they are checked in original
// implementation.
func encodeStringSlowPath(buf *bytes.Buffer, i int, val string, valLen int) {
	start := i
	for i < valLen {
		if b := val[i]; b < utf8.RuneSelf {
			if safeSet[b] {
				i++
				continue
			}
			if start < i {
				buf.WriteString(val[start:i])
			}
			buf.WriteByte('\\')
			switch b {
			case '\\', '"':
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('r')
			case '\t':
				buf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				buf.WriteString(`u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		i++
		continue
	}
	if start < valLen {
		buf.WriteString(val[start:])
	}
}

// safeSet holds the value true if the ASCII character with the given array position can be represented inside a JSON string without any further escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

var hex = "0123456789abcdef"
