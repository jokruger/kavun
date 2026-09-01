package core

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
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

// parseIntArg reads an int-valued slot (a count, a width, an edit position): any numeric is accepted iff the
// conversion is LOSSLESS — repeat(2.0) is repeat(2), repeat(1.5) raises rather than silently truncating —
// and everything non-convertible is a type error.
func parseIntArg(name, pos string, a Value) (int64, error) {
	i, ok := a.AsInt()
	if !ok {
		return 0, errs.NewInvalidArgumentTypeError(name, pos, "int", a.TypeName())
	}
	switch a.Type {
	case value.Float, value.Decimal:
		if !a.Equal(IntValue(i)) {
			return 0, errs.NewInvalidValueError(fmt.Sprintf("(%s) argument %s must be a whole number, got %s", name, pos, a.String()))
		}
	}
	return i, nil
}

// MaxSequenceLen bounds every count-driven sequence allocation (`repeat`, the `*` operator, `pad_start` and
// `pad_end`). Go's makeslice PANICS — not raises — for a length it cannot represent, and a Go panic escaping
// into the host is exactly what the error model forbids, so the count is checked against this ceiling first
// and answers a catchable error instead. The value is far past any real script and below makeslice's own
// limit for every element type Kavun has (byte, rune, Value), which makes that panic unreachable through
// these paths.
const MaxSequenceLen = 1 << 32

// SeqRepeatTotal computes len(receiver) * count for a repeat, raising rather than overflowing or panicking.
// PURE by contract.
func SeqRepeatTotal(name string, n, elems int) (int, error) {
	if n != 0 && elems > MaxSequenceLen/n {
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf(
			"(%s) result would be %d × %d elements, past the %d limit", name, elems, n, MaxSequenceLen))
	}
	return n * elems, nil
}

// SeqPadWidth reads a pad's target WIDTH as an allocation size, raising rather than panicking. It is the same
// ceiling as `repeat`'s, checked directly instead of as a product: a pad's width IS the resulting element
// count. The caller must have handled the no-op case (a width at or below the length) first, so what reaches
// here is always a real allocation. PURE by contract.
func SeqPadWidth(name string, n int64) (int, error) {
	if n > MaxSequenceLen {
		// argument validation must be catchable by recover()
		return 0, errs.NewInvalidValueError(fmt.Sprintf(
			"(%s) result would be %d elements, past the %d limit", name, n, MaxSequenceLen))
	}
	return int(n), nil
}

// SeqRepeatOperand reads the right operand of a sequence's `*` as a repeat count, so `x * n` is exactly
// `x.repeat(n)`. Only a numeric operand is a count: anything else is not a `*` at all and the caller must fall
// through to its ordinary invalid-operator path (a `false` second result), so `[1] * "ab"` reads as
// `invalid_binary_operator` rather than a count that failed to parse. A numeric one is held to the member's
// contract — whole-valued and non-negative — and its failures are catchable, named for the operator.
// PURE by contract.
func SeqRepeatOperand(other Value) (int, bool, error) {
	switch other.Type {
	case value.Int, value.Float, value.Decimal:
	default:
		return 0, false, nil
	}
	n, err := parseIntArg("*", "right operand", other)
	if err != nil {
		return 0, true, err
	}
	if n < 0 {
		// argument validation must be catchable by recover()
		return 0, true, errs.NewInvalidValueError(fmt.Sprintf("(*) repeat count must be non-negative, got %d", n))
	}
	return int(n), true, nil
}

// parseRepeatCount validates and extracts the count argument for a `repeat` method.
// It expects exactly one int argument and returns an error if the count is negative.
func parseRepeatCount(name string, args []Value) (int, error) {
	if len(args) != 1 {
		return 0, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	n, err := parseIntArg(name, "first", args[0])
	if err != nil {
		return 0, err
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

// mapRunesCase applies a per-symbol case mapping, leaving the octet escapes alone — an octet that is not a
// symbol has no case, and Go's strings.ToUpper/ToLower would replace it with U+FFFD on the way through.
// PURE by contract.
func mapRunesCase(rs []rune, f func(rune) rune) []rune {
	out := make([]rune, len(rs))
	for i, r := range rs {
		if IsEscapeRune(r) {
			out[i] = r
			continue
		}
		out[i] = f(r)
	}
	return out
}

// ByteSymbolString is a byte's TEXT CONTENT — the one-octet text that holds it. TOTAL: below 0x80 that text
// is the octet's ASCII symbol, and at or above it the text holds the octet itself, which reads back as the
// octet's escape and converts back to this same byte. The old Latin-1 leak the partial version guarded against
// (equality reading "\xFF" as "ÿ") cannot happen now: "\xFF" is one ESCAPE symbol, never U+00FF, so
// b'\xff' == "\xff" is true and b'\xff' == "ÿ" is false, which is what both should be.
// The RENDER of a byte stays its number (format(b) -> "65").
func ByteSymbolString(b byte) (string, bool) {
	return string([]byte{b}), true
}

// IsBlankElement reports whether e is "insignificant content" for a general container: undefined, or the
// element type's own zero value. Match-taking members called with NO argument act on this set (count() counts
// the significant elements, keep() answers exactly those, remove() drops the blanks, index() locates the first
// significant one). Each verb reads against the set the way its own name implies, which is why keep() and
// remove() land on the same answer with no argument: two different actions (keep the significant / remove the
// blank), not one operation under two names. It is a DEFAULT, not a policy — the argument forms override it at
// any call site, and a
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

// IsBlankRune: the symbol types' blank set — NUL plus Unicode whitespace. The blank set is one notion,
// NUL ∪ whitespace, projected into each receiver's ELEMENT DOMAIN: symbols (string/runes) take the Unicode
// White_Space class, octets (bytes) the ASCII subset — all the whitespace an octet can express — so the two
// sets agree everywhere both domains overlap and neither type imports the other's limitation.
func IsBlankRune(r rune) bool {
	return r == 0 || unicode.IsSpace(r)
}

// IsBlankByte is the octet projection of the same notion: NUL plus ASCII whitespace, a fixed set of literal
// octets — deciding anything wider would require decoding, which octets never get.
func IsBlankByte(b byte) bool {
	switch b {
	case 0, ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
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
// them with `sep`. A container element raises rather than passing through its element-wise text conversion — a
// nested collection has no canonical text, and joining one silently was the data-loss class this rule removes.
func joinElementsToString(elems []Value, sep string) (string, error) {
	if len(elems) == 0 {
		return "", nil
	}
	parts := make([]string, len(elems))
	total := 0
	for i, e := range elems {
		switch e.Type {
		case value.Array, value.Dict, value.Record, value.IntRange:
			return "", errs.NewConversionError(e.TypeName(), "string", "a nested collection is not renderable in join")
		}
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

// ---------------------------------------------------------------------------
// The casing family: one word segmenter, five renderings. Symbol-class
// operations, so they live on string/runes only — never on bytes.
// ---------------------------------------------------------------------------

// caseSegmentWritten splits text on the WRITTEN boundaries only — runs of whitespace, '_' and '-',
// discarded, empty words dropped. The label rendering (title_case) segments with this: a label preserves the
// author's emphasis, and that covers word boundaries the author did not write — case transitions stay inside
// words ("iPhone" → "IPhone", "hELLO" → "HELLO"). The identifier renderings use the full segmenter below.
func caseSegmentWritten(rs []rune) [][]rune {
	var words [][]rune
	var cur []rune
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			if len(cur) > 0 {
				words = append(words, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		words = append(words, cur)
	}
	return words
}

// caseSegmentWords splits text into words for the *_case members. The boundary set is CLOSED:
//   - runs of whitespace, '_' and '-' delimit words and are discarded;
//   - a lower→upper transition starts a word (helloWorld → hello|World);
//   - in an upper run followed by a lower, the last upper starts the word
//     (HTTPServer → HTTP|Server, parseXMLFile → parse|XML|File);
//   - everything else — digits, apostrophes, periods — stays inside the word,
//     which is what keeps "don't" one word (an OPEN boundary set is how Go's
//     deprecated strings.Title produced "Don'T");
//   - empty words are dropped.
func caseSegmentWords(rs []rune) [][]rune {
	var words [][]rune
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, cur)
			cur = nil
		}
	}
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			flush()
			continue
		}
		if len(cur) > 0 {
			prev := cur[len(cur)-1]
			switch {
			case unicode.IsLower(prev) && unicode.IsUpper(r):
				flush()
			case unicode.IsUpper(prev) && unicode.IsLower(r) && len(cur) >= 2 && unicode.IsUpper(cur[len(cur)-2]):
				last := prev
				cur = cur[:len(cur)-1]
				flush()
				cur = []rune{last}
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

// caseRenderWords renders segmented words per member. Two policies: the identifier renderings
// (snake/kebab/camel/pascal) NORMALISE the interior — an identifier has a canonical case — while the label
// rendering (title_case) PRESERVES it, uppercasing only each word's first symbol ("ATM fee" → "ATM Fee" as a
// title, "atm_fee" as an identifier).
func caseRenderWords(name string, words [][]rune) []rune {
	out := make([]rune, 0, 16)
	for wi, w := range words {
		switch name {
		case "snake_case", "kebab_case":
			if wi > 0 {
				if name == "snake_case" {
					out = append(out, '_')
				} else {
					out = append(out, '-')
				}
			}
			for _, r := range w {
				out = append(out, unicode.ToLower(r))
			}
		case "camel_case", "pascal_case":
			for i, r := range w {
				if i == 0 && (name == "pascal_case" || wi > 0) {
					out = append(out, unicode.ToUpper(r))
				} else {
					out = append(out, unicode.ToLower(r))
				}
			}
		case "title_case":
			if wi > 0 {
				out = append(out, ' ')
			}
			for i, r := range w {
				if i == 0 {
					out = append(out, unicode.ToUpper(r))
				} else {
					out = append(out, r)
				}
			}
		}
	}
	return out
}

// foldRuneCanonical maps a symbol to one canonical member of its simple case-folding orbit — the smallest
// LOWERCASE member, or the smallest member when the orbit has none — so a.case_fold() == b.case_fold() answers
// exactly Go's strings.EqualFold while the render reads like every mainstream casefold ("Hello" → "hello";
// S/s/ſ all → s). The representative must come from INSIDE the orbit: naively lowercasing the minimum would
// merge İ (an orbit of one) into i's orbit and break the equality. A TRANSFORM, which composes (a dict key, a
// sort basis, a dedup argument) where a comparison predicate cannot; not the same as lowering — the fold
// predicates differ from lower-comparison in both directions (ſ/s fold-equal but lower-unequal; İ/i the reverse).
func foldRuneCanonical(r rune) rune {
	minAll := r
	minLower := rune(-1)
	f := r
	for {
		if f < minAll {
			minAll = f
		}
		if unicode.IsLower(f) && (minLower < 0 || f < minLower) {
			minLower = f
		}
		f = unicode.SimpleFold(f)
		if f == r {
			break
		}
	}
	if minLower >= 0 {
		return minLower
	}
	return minAll
}

// ---------------------------------------------------------------------------
// Operator-layer text operands. The receiver — the LEFT operand — decides the
// result type; acceptance mirrors the member layer with one deliberate cut:
// an int is never text in operator position, so the arithmetic readings stay
// unambiguous. ok=false means the operand is not text content at all (the
// caller answers with its invalid-operator error); a data-range failure (a
// non-ASCII octet into symbols, invalid UTF-8) raises directly.
// ---------------------------------------------------------------------------

// textOperandString reads an operand as SYMBOL content (string/runes receivers).
func textOperandString(a Value) (string, bool, error) {
	switch a.Type {
	case value.String:
		return *(*string)(a.Ptr), true, nil
	case value.Runes:
		return string((*Runes)(a.Ptr).Elements), true, nil
	case value.Bytes:
		// TOTAL: an undecodable octet is carried through as itself; a string holds octets
		return string((*Bytes)(a.Ptr).Elements), true, nil
	case value.Rune:
		return string(rune(a.Data)), true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return "", false, errs.NewInvalidValueError(fmt.Sprintf("an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", a.Data))
		}
		return string(rune(a.Data)), true, nil
	}
	return "", false, nil
}

// textOperandOctets reads an operand as OCTET content (bytes receivers) — encoding is total, so every text
// operand is accepted; symbols contribute their UTF-8 encoding.
func textOperandOctets(a Value) ([]byte, bool, error) {
	switch a.Type {
	case value.Bytes:
		return (*Bytes)(a.Ptr).Elements, true, nil
	case value.String:
		return []byte(*(*string)(a.Ptr)), true, nil
	case value.Runes:
		return []byte(string((*Runes)(a.Ptr).Elements)), true, nil
	case value.Rune:
		return []byte(string(rune(a.Data))), true, nil
	case value.Byte:
		return []byte{byte(a.Data)}, true, nil
	}
	return nil, false, nil
}

// removeAllBytes is `-` on an octet receiver: every occurrence of the run, leftmost non-overlapping;
// the empty run removes nothing.
func removeAllBytes(l, r []byte) []byte {
	if len(r) == 0 {
		return slices.Clone(l)
	}
	return bytes.ReplaceAll(l, r, nil)
}

// mapContainsKey is the `in` operator on a map receiver: the KEY axis — a key is accepted iff its
// own string conversion exists (the d[k] rule); a submap operand raises as deferred, a callable
// raises, anything unconvertible raises — never a silent false.
func mapContainsKey(m map[string]Value, e Value) (bool, error) {
	if e.IsCallable() {
		return false, errs.NewInvalidValueError("(in) an operator operand is always a value — the predicate reading is contains(f)/any(f)")
	}
	if e.Type == value.Dict || e.Type == value.Record {
		return false, errs.NewNotImplementedError("(in) the submap reading is deferred; match keys, or compare entries with a predicate")
	}
	s, ok := e.AsString()
	if !ok {
		return false, errs.NewInvalidArgumentTypeError("in", "operand", "a key (string)", e.TypeName())
	}
	_, hit := m[s]
	return hit, nil
}
