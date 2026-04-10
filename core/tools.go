package core

import (
	"bytes"
	"unicode/utf8"
)

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
