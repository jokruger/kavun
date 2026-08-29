package core

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jokruger/dec128"
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
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("for_each", "first", "function", fn.TypeName())
	}
	if arity := fn.Arity(); arity != 1 && arity != 2 {
		return Undefined, errs.NewInvalidArgumentTypeError("for_each", "first", "f/1 or f/2", fn.TypeName())
	}

	return fn, nil
}

// mapsEqual checks if two maps of string to Value are equal, using Value.Equal for value comparison.
func mapsEqual(a, b map[string]Value) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !av.Equal(bv) {
			return false
		}
	}
	return true
}

// mergeMaps merges two maps of string to Value by copying all entries from both maps into a new map.
// If a key exists in both maps, the value from the second map overwrites the value from the first map.
func mergeMaps(a, b map[string]Value) map[string]Value {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	result := make(map[string]Value, len(a)+len(b))
	maps.Copy(result, a)
	maps.Copy(result, b)
	return result
}

// decimalToExactRat converts d to its true exact value as ±coefficient/10^scale — never through a
// lossy intermediate (no ToString()/parse, no float64) — for use in exact cross-type comparisons
// against float, where dec128's own fixed ~34-digit precision can't hold an arbitrary float64's
// exact decimal expansion but math/big.Rat, being arbitrary-precision, always can. Caller must
// exclude NaN first — a NaN decimal has no coefficient/scale worth converting.
func decimalToExactRat(d *dec128.Dec128) *big.Rat {
	coef := d.Coefficient().BigInt()
	if d.IsNegative() {
		coef = new(big.Int).Neg(coef)
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale())), nil)
	return new(big.Rat).SetFrac(coef, denom)
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
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) repeat count must be non-negative, got %d", name, n))
	}
	return int(n), nil
}

// MapToSortedEntries materializes a map's conversion elements — its entries — as [[k, v], ...] in canonical
// key-sorted order, so the two directions of the sequence<->map boundary round-trip up to that ordering.
func MapToSortedEntries(m map[string]Value) []Value {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]Value, len(keys))
	for i, k := range keys {
		entries[i] = NewArrayValue([]Value{NewStringValue(k), m[k]}, false)
	}
	return entries
}

// ElementsToRunes converts an element container's content to symbols: each element through its own rune
// conversion, all-or-nothing — a failing element fails the whole conversion, with no partial result and no
// substituted placeholder (the silent NUL/U+FFFD corruption this replaces).
func ElementsToRunes(elems []Value) ([]rune, bool) {
	rs := make([]rune, len(elems))
	for i, e := range elems {
		r, ok := e.AsRune()
		if !ok {
			return nil, false
		}
		rs[i] = r
	}
	return rs, true
}

// ElementsToBytes is ElementsToRunes' octet twin.
func ElementsToBytes(elems []Value) ([]byte, bool) {
	bs := make([]byte, len(elems))
	for i, e := range elems {
		b, ok := e.AsByte()
		if !ok {
			return nil, false
		}
		bs[i] = b
	}
	return bs, true
}

// ElementsToEntries reads a sequence as a map: each element must be EXACTLY a 2-element array (an entry) — any
// other element fails, including a 2-element text sequence, so a misread never silently becomes a map. The key
// goes through its own string conversion (absent on undefined/dict/record/callables, which therefore fail) and
// later entries overwrite earlier ones, the same last-wins rule as merging maps.
func ElementsToEntries(elems []Value) (map[string]Value, bool) {
	m := make(map[string]Value, len(elems))
	for _, e := range elems {
		if e.Type != value.Array {
			return nil, false
		}
		entry := (*Array)(e.Ptr).Elements
		if len(entry) != 2 {
			return nil, false
		}
		k, ok := entry[0].AsString()
		if !ok {
			return nil, false
		}
		m[k] = entry[1]
	}
	return m, true
}

// ByteSymbolString is a byte's TEXT CONTENT — its ASCII symbol. It succeeds iff the octet is a UTF-8
// representation of a symbol on its own (0x00-0x7F); 0x80-0xFF alone is not valid UTF-8 and declines.
// The RENDER of a byte stays its number (format(b) -> "65") and stays total — a conversion may be partial,
// the universal render may not.
func ByteSymbolString(b byte) (string, bool) {
	if b > 0x7F {
		return string(rune(b)), false
	}
	return string(rune(b)), true
}

// IsBlankElement reports whether e is "insignificant content" for a general container: undefined, or the
// element type's own zero value. Match-taking members called with NO argument act on this set (count() counts
// the significant elements, filter() keeps them, remove() drops the blanks, index() locates the first
// significant one). It is a DEFAULT, not a policy — the argument forms override it at any call site, and a
// script that means "zeros are data" passes its own set. The text triple uses whitespace sets instead
// (IsBlankRune/IsBlankByte): these members are about separators and filler, and whitespace is text's filler.
func IsBlankElement(e Value) bool {
	switch e.Type {
	case value.Undefined:
		return true
	case value.Bool, value.Byte, value.Rune, value.Int:
		return e.Data == 0
	case value.Float:
		return math.Float64frombits(e.Data) == 0
	case value.Decimal:
		return (*dec128.Dec128)(e.Ptr).IsZero()
	case value.String:
		return len(*(*string)(e.Ptr)) == 0
	case value.Runes:
		return len((*Runes)(e.Ptr).Elements) == 0
	case value.Bytes:
		return len((*Bytes)(e.Ptr).Elements) == 0
	case value.Array:
		return len((*Array)(e.Ptr).Elements) == 0
	case value.Dict:
		return len((*Dict)(e.Ptr).Elements) == 0
	case value.Record:
		return len((*Record)(e.Ptr).Elements) == 0
	case value.Time:
		return (*time.Time)(e.Ptr).IsZero()
	case value.IntRange:
		o := (*IntRange)(e.Ptr)
		return o.Start == o.Stop
	}
	// callables and error have no zero value and are never blank
	return false
}

// IsBlankRune: the text triple's blank set — NUL plus ASCII whitespace, a fixed set of literal code points.
func IsBlankRune(r rune) bool {
	switch r {
	case 0, ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// IsBlankByte is IsBlankRune on octets — identical content, so bytes and the symbol types agree.
func IsBlankByte(b byte) bool {
	return b <= 0x7F && IsBlankRune(rune(b))
}

// convMember implements the uniform conversion-member shape x.T([default]): zero or one argument; a failed
// conversion answers the default when one is supplied and raises otherwise. Every conversion member carries the
// slot — including identities and total conversions, where it can never fire — so generic code never has to
// special-case the receiver's type. The default is deliberately not type-checked: it is an explicit opt-out,
// the caller's responsibility.
func convMember(name string, from string, args []Value, ok bool, res Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	if ok {
		return res, nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return Undefined, errs.NewConversionError(from, name, "")
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
			return "", errs.NewConversionError(e.TypeName(), "string", "")
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

// PURE by contract.
func defaultEqual(v Value, other Value, final bool) bool {
	// default to in-memory value equality
	if final {
		return v == other
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func defaultBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func defaultMethodCall(_ VM, v Value, name string, _ []Value) (Value, error) {
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
}

// MUTATE-DEPENDENT by contract (see ValueTypeDescr.Delete)
func defaultDelete(v Value, _ Value, _ bool) (Value, error) {
	return Undefined, errs.NewNotDeletableError(v.TypeName())
}

// PURE by contract
func defaultAccess(v Value, _ Value, _ bc.Opcode) (Value, error) {
	return Undefined, errs.NewNotAccessibleError(v.TypeName())
}

// MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append) — this type supports neither form.
func defaultAppend(v Value, _ []Value, _ bool) (Value, error) {
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
