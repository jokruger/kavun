package stdlib

import (
	"regexp"
	"testing"

	"github.com/jokruger/kavun/core"
)

func TestTextRE(t *testing.T) {
	// re_match(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"abc", ""},
		{"abc", "abc"},
		{"a", "abc"},
		{"b", "abc"},
		{"^a", "abc"},
		{"^b", "abc"},
	} {
		expected := regexp.MustCompile(d.pattern).MatchString(d.text)
		module(t, "text").call(rta, "re_match", d.pattern, d.text).expect(rta, expected, "pattern: %q, src: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "match", d.text).expect(rta, expected, "patter: %q, src: %q", d.pattern, d.text)
	}

	// re_find(pattern, text)
	for _, d := range []struct {
		pattern  string
		text     string
		expected any
	}{
		{"a(b)", "", core.Undefined},
		{"a(b)", "ab", ARR{
			ARR{
				IMAP{"text": "ab", "begin": 0, "end": 2},
				IMAP{"text": "b", "begin": 1, "end": 2},
			},
		}},
		{"a(bc)d", "abcdefgabcd", ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "bc", "begin": 1, "end": 3},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
		}},
	} {
		module(t, "text").call(rta, "re_find", d.pattern, d.text).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "find", d.text).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_find(pattern, text, count))
	for _, d := range []struct {
		pattern  string
		text     string
		count    int
		expected any
	}{
		{"a(b)", "", -1, core.Undefined},
		{"a(b)", "ab", -1, ARR{
			ARR{
				IMAP{"text": "ab", "begin": 0, "end": 2},
				IMAP{"text": "b", "begin": 1, "end": 2},
			},
		}},
		{"a(bc)d", "abcdefgabcd", -1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "bc", "begin": 1, "end": 3},
			},
			ARR{
				IMAP{"text": "abcd", "begin": 7, "end": 11},
				IMAP{"text": "bc", "begin": 8, "end": 10},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", -1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
			ARR{
				IMAP{"text": "abcd", "begin": 7, "end": 11},
				IMAP{"text": "a", "begin": 7, "end": 8},
				IMAP{"text": "c", "begin": 9, "end": 10},
			},
		}},
		{"(a)b(c)d", "abcdefgabcd", 0, core.Undefined},
		{"(a)b(c)d", "abcdefgabcd", 1, ARR{
			ARR{
				IMAP{"text": "abcd", "begin": 0, "end": 4},
				IMAP{"text": "a", "begin": 0, "end": 1},
				IMAP{"text": "c", "begin": 2, "end": 3},
			},
		}},
	} {
		module(t, "text").call(rta, "re_find", d.pattern, d.text, d.count).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "find", d.text, d.count).expect(rta, d.expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_replace(pattern, text, repl)
	for _, d := range []struct {
		pattern string
		text    string
		repl    string
	}{
		{"a", "", "b"},
		{"a", "a", "b"},
		{"a", "acac", "b"},
		{"b", "acac", "x"},
		{"a", "acac", "123"},
		{"ac", "acac", "99"},
		{"ac$", "acac", "foo"},
		{"a(b)", "ababab", "$1"},
		{"a(b)(c)", "abcabcabc", "$2$1"},
		{"(a(b)c)", "abcabcabc", "$1$2"},
		{"(일(2)삼)", "일2삼12삼일23", "$1$2"},
		{"((일)(2)3)", "일23\n일이3\n일23", "$1$2$3"},
		{"(a(b)c)", "abc\nabc\nabc", "$1$2"},
	} {
		expected := regexp.MustCompile(d.pattern).
			ReplaceAllString(d.text, d.repl)
		module(t, "text").call(rta, "re_replace", d.pattern, d.text, d.repl).expect(rta, expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "replace", d.text, d.repl).expect(rta, expected, "pattern: %q, text: %q, repl: %q", d.pattern, d.text, d.repl)
	}

	// re_split(pattern, text)
	for _, d := range []struct {
		pattern string
		text    string
	}{
		{"a", ""},
		{"a", "abcabc"},
		{"ab", "abcabc"},
		{"^a", "abcabc"},
	} {
		var expected []any
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, -1) {
			expected = append(expected, ex)
		}
		module(t, "text").call(rta, "re_split", d.pattern, d.text).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "split", d.text).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
	}

	// re_split(pattern, text, count))
	for _, d := range []struct {
		pattern string
		text    string
		count   int
	}{
		{"a", "", -1},
		{"a", "abcabc", -1},
		{"ab", "abcabc", -1},
		{"^a", "abcabc", -1},
		{"a", "abcabc", 0},
		{"a", "abcabc", 1},
		{"a", "abcabc", 2},
		{"a", "abcabc", 3},
		{"b", "abcabc", 1},
		{"b", "abcabc", 2},
		{"b", "abcabc", 3},
	} {
		var expected []any
		for _, ex := range regexp.MustCompile(d.pattern).Split(d.text, d.count) {
			expected = append(expected, ex)
		}
		module(t, "text").call(rta, "re_split", d.pattern, d.text, d.count).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
		module(t, "text").call(rta, "re_compile", d.pattern).call(rta, "split", d.text, d.count).expect(rta, expected, "pattern: %q, text: %q", d.pattern, d.text)
	}
}

func TestText(t *testing.T) {
	module(t, "text").call(rta, "compare", "", "").expect(rta, 0)
	module(t, "text").call(rta, "compare", "", "a").expect(rta, -1)
	module(t, "text").call(rta, "compare", "a", "").expect(rta, 1)
	module(t, "text").call(rta, "compare", "a", "a").expect(rta, 0)
	module(t, "text").call(rta, "compare", "a", "b").expect(rta, -1)
	module(t, "text").call(rta, "compare", "b", "a").expect(rta, 1)
	module(t, "text").call(rta, "compare", "abcde", "abcde").expect(rta, 0)
	module(t, "text").call(rta, "compare", "abcde", "abcdf").expect(rta, -1)
	module(t, "text").call(rta, "compare", "abcdf", "abcde").expect(rta, 1)

	module(t, "text").call(rta, "contains", "", "").expect(rta, true)
	module(t, "text").call(rta, "contains", "", "a").expect(rta, false)
	module(t, "text").call(rta, "contains", "a", "").expect(rta, true)
	module(t, "text").call(rta, "contains", "a", "a").expect(rta, true)
	module(t, "text").call(rta, "contains", "abcde", "a").expect(rta, true)
	module(t, "text").call(rta, "contains", "abcde", "abcde").expect(rta, true)
	module(t, "text").call(rta, "contains", "abc", "abcde").expect(rta, false)
	module(t, "text").call(rta, "contains", "ab cd", "bc").expect(rta, false)

	module(t, "text").call(rta, "replace", "", "", "", -1).expect(rta, "")
	module(t, "text").call(rta, "replace", "abcd", "a", "x", -1).expect(rta, "xbcd")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", -1).expect(rta, "xxxx")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", 0).expect(rta, "aaaa")
	module(t, "text").call(rta, "replace", "aaaa", "a", "x", 2).expect(rta, "xxaa")
	module(t, "text").call(rta, "replace", "abcd", "bc", "x", -1).expect(rta, "axd")

	module(t, "text").call(rta, "format_bool", true).expect(rta, "true")
	module(t, "text").call(rta, "format_bool", false).expect(rta, "false")
	module(t, "text").call(rta, "format_float", -19.84, "f", -1, 64).expect(rta, "-19.84")
	module(t, "text").call(rta, "format_int", -1984, 10).expect(rta, "-1984")
	module(t, "text").call(rta, "format_int", 1984, 8).expect(rta, "3700")
	module(t, "text").call(rta, "parse_bool", "true").expect(rta, true)
	module(t, "text").call(rta, "parse_bool", "0").expect(rta, false)
	module(t, "text").call(rta, "parse_float", "-19.84", 64).expect(rta, -19.84)
	module(t, "text").call(rta, "parse_int", "-1984", 10, 64).expect(rta, -1984)
}

func TestReplace(t *testing.T) {
	module(t, "text").call(rta, "replace", "123456789012", "1", "x", -1).expect(rta, "x234567890x2")
	module(t, "text").call(rta, "replace", "123456789012", "12", "x", -1).expect(rta, "x34567890x")
	module(t, "text").call(rta, "replace", "123456789012", "012", "xyz", -1).expect(rta, "123456789xyz")
	module(t, "text").call(rta, "re_replace", "1", "123456789012", "x").expect(rta, "x234567890x2")
	module(t, "text").call(rta, "re_replace", "12", "123456789012", "x").expect(rta, "x34567890x")
	module(t, "text").call(rta, "re_replace", "1(2)", "123456789012", "x$1").expect(rta, "x234567890x2")
	module(t, "text").call(rta, "re_replace", "(1)(2)", "123456789012", "$2$1").expect(rta, "213456789021")
}

func TestTextRepeat(t *testing.T) {
	module(t, "text").call(rta, "repeat", "1234", "3").expect(rta, "123412341234")
	module(t, "text").call(rta, "repeat", "1", "12").expect(rta, "111111111111")
}

func TestSubstr(t *testing.T) {
	module(t, "text").call(rta, "substr", "", 0, 0).expect(rta, "")
	module(t, "text").call(rta, "substr", "abcdef", 0, 3).expect(rta, "abc")
	module(t, "text").call(rta, "substr", "abcdef", 0, 6).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 0, 10).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", -10, 10).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 0).expect(rta, "abcdef")
	module(t, "text").call(rta, "substr", "abcdef", 3).expect(rta, "def")

	module(t, "text").call(rta, "substr", "", 10, 0).expectError()
	module(t, "text").call(rta, "substr", "", "10", 0).expectError()
	module(t, "text").call(rta, "substr", "", 10, "0").expectError()
	module(t, "text").call(rta, "substr", "", "10", "0").expectError()

	module(t, "text").call(rta, "substr", 0, 0, 1).expect(rta, "0")
	module(t, "text").call(rta, "substr", 123, 0, 1).expect(rta, "1")
	module(t, "text").call(rta, "substr", 123.456, 4, 7).expect(rta, "456")
}

func TestPadLeft(t *testing.T) {
	module(t, "text").call(rta, "pad_left", "ab", 7, 0).expect(rta, "00000ab")
	module(t, "text").call(rta, "pad_right", "ab", 7, 0).expect(rta, "ab00000")
	module(t, "text").call(rta, "pad_left", "ab", 7, "+-").expect(rta, "-+-+-ab")
	module(t, "text").call(rta, "pad_right", "ab", 7, "+-").expect(rta, "ab+-+-+")
}
