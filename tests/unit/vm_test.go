package unit

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/parser"
	"github.com/jokruger/kavun/tests/require"
)

func TestUndefined(t *testing.T) {
	expectRun(t, rta, `out = undefined`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined.a`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined[1]`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined.a.b`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined[1][2]`, nil, core.Undefined)
	expectRun(t, rta, `out = undefined ? 1 : 2`, nil, 2)
	expectRun(t, rta, `out = undefined == undefined`, nil, true)
	expectRun(t, rta, `out = undefined == 1`, nil, false)
	expectRun(t, rta, `out = 1 == undefined`, nil, false)
	expectRun(t, rta, `out = undefined == float([])`, nil, true)
	expectRun(t, rta, `out = float([]) == undefined`, nil, true)
	expectRun(t, rta, `out = undefined.format("v")`, nil, "undefined")

	u := core.Undefined
	s, _ := u.AsString(rta)
	require.Equal(t, rta, "", s)
	require.Equal(t, rta, "undefined", u.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = undefined == %s`, u.String(rta)), nil, true)
}

func TestBoolean(t *testing.T) {
	expectRun(t, rta, `out = bool()`, nil, false)
	expectRun(t, rta, `out = bool(true)`, nil, true)
	expectRun(t, rta, `out = bool(false)`, nil, false)

	expectRun(t, rta, `out = true`, nil, true)
	expectRun(t, rta, `out = false`, nil, false)

	expectRun(t, rta, `out = 1 < 2`, nil, true)
	expectRun(t, rta, `out = 1 > 2`, nil, false)
	expectRun(t, rta, `out = 1 < 1`, nil, false)
	expectRun(t, rta, `out = 1 > 2`, nil, false)
	expectRun(t, rta, `out = 1 == 1`, nil, true)
	expectRun(t, rta, `out = 1 != 1`, nil, false)
	expectRun(t, rta, `out = 1 == 2`, nil, false)
	expectRun(t, rta, `out = 1 != 2`, nil, true)
	expectRun(t, rta, `out = 1 <= 2`, nil, true)
	expectRun(t, rta, `out = 1 >= 2`, nil, false)
	expectRun(t, rta, `out = 1 <= 1`, nil, true)
	expectRun(t, rta, `out = 1 >= 2`, nil, false)

	expectRun(t, rta, `out = true == true`, nil, true)
	expectRun(t, rta, `out = false == false`, nil, true)
	expectRun(t, rta, `out = true == false`, nil, false)
	expectRun(t, rta, `out = true != false`, nil, true)
	expectRun(t, rta, `out = false != true`, nil, true)
	expectRun(t, rta, `out = (1 < 2) == true`, nil, true)
	expectRun(t, rta, `out = (1 < 2) == false`, nil, false)
	expectRun(t, rta, `out = (1 > 2) == true`, nil, false)
	expectRun(t, rta, `out = (1 > 2) == false`, nil, true)
	expectRun(t, rta, `out = 5 + true`, nil, 6)
	expectRun(t, rta, `out = 5 + true; 5`, nil, 6)

	expectError(t, rta, `-true`, nil, "invalid_unary_operator: - bool")
	expectError(t, rta, `true + false`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `5; true + false; 5`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `if (10 > 1) { true + false; }`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, rta, `
func() {
	if (10 > 1) {
		if (10 > 1) {
			return true + false;
		}

		return 1;
	}
}()
`, nil, "invalid_binary_operator: bool + bool")

	expectError(t, rta, `if (true + false) { 10 }`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `10 + (true + false)`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `(true + false) + 20`, nil, "invalid_binary_operator: bool + bool")
	expectError(t, rta, `!(true + false)`, nil, "invalid_binary_operator: bool + bool")

	var v core.Value

	v = core.True
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "true", s)
	v = core.True
	require.Equal(t, rta, "true", v.String(rta))

	v = core.True
	expectRun(t, rta, fmt.Sprintf(`out = true == %s`, v.String(rta)), nil, true)
	v = core.False
	expectRun(t, rta, fmt.Sprintf(`out = false == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = true.bool()`, nil, true)
	expectRun(t, rta, `out = false.bool()`, nil, false)
	expectRun(t, rta, `out = true.byte()`, nil, byte(1))
	expectRun(t, rta, `out = false.byte()`, nil, byte(0))
	expectRun(t, rta, `out = true.int()`, nil, 1)
	expectRun(t, rta, `out = false.int()`, nil, 0)
	expectRun(t, rta, `out = true.string()`, nil, "true")
	expectRun(t, rta, `out = false.string()`, nil, "false")
	expectRun(t, rta, `out = false.format()`, nil, "false")
	expectRun(t, rta, `out = false.format("v")`, nil, "false")
}

func TestByte(t *testing.T) {
	var v core.Value

	expectRun(t, rta, `out = byte(5)`, nil, byte(5))
	expectRun(t, rta, `out = byte(true)`, nil, byte(1))
	expectRun(t, rta, `out = byte(false)`, nil, byte(0))
	expectRun(t, rta, `out = byte('A')`, nil, byte(65))
	expectRun(t, rta, `out = byte("12")`, nil, byte(12))
	expectRun(t, rta, `out = byte(u"12")`, nil, byte(12))
	expectRun(t, rta, `out = byte(u"300", byte(7))`, nil, byte(7))
	expectRun(t, rta, `out = byte(255) + 1`, nil, byte(0))
	expectRun(t, rta, `out = byte(255) + 2`, nil, byte(1))
	expectRun(t, rta, `out = byte(0) - 1`, nil, byte(255))
	expectRun(t, rta, `out = 1 + byte(255)`, nil, int64(256))

	v = core.ByteValue(0)
	expectRun(t, rta, fmt.Sprintf(`out = byte(0) == %s`, v.String(rta)), nil, true)
	v = core.ByteValue(1)
	expectRun(t, rta, fmt.Sprintf(`out = byte(1) == %s`, v.String(rta)), nil, true)
	v = core.ByteValue(123)
	expectRun(t, rta, fmt.Sprintf(`out = byte(123) == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = byte(123).int()`, nil, 123)
	expectRun(t, rta, `out = byte(0).bool()`, nil, false)
	expectRun(t, rta, `out = byte(10).bool()`, nil, true)
	expectRun(t, rta, `out = byte(48).rune()`, nil, '0')
	expectRun(t, rta, `out = byte(48).float()`, nil, 48.0)
	expectRun(t, rta, `out = byte(48).string()`, nil, "48")
	expectRun(t, rta, `out = byte(48).format()`, nil, "48")
	expectRun(t, rta, `out = byte(48).format("v")`, nil, "byte(48)")
}

func TestInteger(t *testing.T) {
	var v core.Value

	expectRun(t, rta, `out = 5`, nil, 5)
	expectRun(t, rta, `out = 10`, nil, 10)
	expectRun(t, rta, `out = -5`, nil, -5)
	expectRun(t, rta, `out = -10`, nil, -10)
	expectRun(t, rta, `out = 5 + 5 + 5 + 5 - 10`, nil, 10)
	expectRun(t, rta, `out = 2 * 2 * 2 * 2 * 2`, nil, 32)
	expectRun(t, rta, `out = -50 + 100 + -50`, nil, 0)
	expectRun(t, rta, `out = 5 * 2 + 10`, nil, 20)
	expectRun(t, rta, `out = 5 + 2 * 10`, nil, 25)
	expectRun(t, rta, `out = 20 + 2 * -10`, nil, 0)
	expectRun(t, rta, `out = 50 / 2 * 2 + 10`, nil, 60)
	expectRun(t, rta, `out = 2 * (5 + 10)`, nil, 30)
	expectRun(t, rta, `out = 3 * 3 * 3 + 10`, nil, 37)
	expectRun(t, rta, `out = 3 * (3 * 3) + 10`, nil, 37)
	expectRun(t, rta, `out = (5 + 10 * 2 + 15 /3) * 2 + -10`, nil, 50)
	expectRun(t, rta, `out = 5 % 3`, nil, 2)
	expectRun(t, rta, `out = 5 % 3 + 4`, nil, 6)
	expectRun(t, rta, `out = +5`, nil, 5)
	expectRun(t, rta, `out = +5 + -5`, nil, 0)

	expectRun(t, rta, `out = 9 + '0'`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, rta, `out = '9' - 5`, nil, 52) // '9' is 57 in ASCII

	v = core.IntValue(0)
	expectRun(t, rta, fmt.Sprintf(`out = 0 == %s`, v.String(rta)), nil, true)
	v = core.IntValue(1)
	expectRun(t, rta, fmt.Sprintf(`out = 1 == %s`, v.String(rta)), nil, true)
	v = core.IntValue(1234567890)
	expectRun(t, rta, fmt.Sprintf(`out = 1234567890 == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = 5 + "-5"`, nil, 0)
	expectRun(t, rta, `out = 5 + "5"`, nil, 10)

	expectRun(t, rta, `out = (12).int()`, nil, 12)
	expectRun(t, rta, `out = (0).bool()`, nil, false)
	expectRun(t, rta, `out = (10).bool()`, nil, true)
	expectRun(t, rta, `out = (48).rune()`, nil, '0')
	expectRun(t, rta, `out = (48).float()`, nil, 48.0)
	expectRun(t, rta, `out = (48).string()`, nil, "48")
	expectRun(t, rta, `out = (1234567890).time().utc().string()`, nil, "2009-02-13 23:31:30 +0000 UTC")
	expectRun(t, rta, `out = (48).byte()`, nil, byte(48))
	expectRun(t, rta, `out = (48).format()`, nil, "48")
	expectRun(t, rta, `out = (48).format("v")`, nil, "48")
}

func TestFloat(t *testing.T) {
	expectRun(t, rta, `out = 0.0`, nil, 0.0)
	expectRun(t, rta, `out = -10.3`, nil, -10.3)
	expectRun(t, rta, `out = 3.2 + 2.0 * -4.0`, nil, -4.8)
	expectRun(t, rta, `out = 4 + 2.3`, nil, 6.3)
	expectRun(t, rta, `out = 2.3 + 4`, nil, 6.3)
	expectRun(t, rta, `out = +5.0`, nil, 5.0)
	expectRun(t, rta, `out = -5.0 + +5.0`, nil, 0.0)

	v := core.FloatValue(0.0)
	expectRun(t, rta, fmt.Sprintf(`out = 0.0 == %s`, v.String(rta)), nil, true)
	v = core.FloatValue(1.0)
	expectRun(t, rta, fmt.Sprintf(`out = 1.0 == %s`, v.String(rta)), nil, true)
	v = core.FloatValue(12345.6789)
	expectRun(t, rta, fmt.Sprintf(`out = 12345.6789 == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = 5.0 + "-5.0"`, nil, 0.0)
	expectRun(t, rta, `out = 5.0 + "5.0"`, nil, 10.0)

	expectRun(t, rta, `out = (1.5).float()`, nil, 1.5)
	expectRun(t, rta, `out = (1.5).int()`, nil, 1)
	expectRun(t, rta, `out = (1.5).string()`, nil, "1.5")

	// f-suffix float literals
	expectRun(t, rta, `out = 1f`, nil, 1.0)
	expectRun(t, rta, `out = 1.5f`, nil, 1.5)
	expectRun(t, rta, `out = type_name(1f)`, nil, "float")
	expectRun(t, rta, `out = type_name(1.5f)`, nil, "float")
	expectRun(t, rta, `out = 2f + 3f`, nil, 5.0)
}

func TestDecimal(t *testing.T) {
	expectRun(t, rta, `out = decimal(123)`, nil, dec128.FromInt64(123))
	expectRun(t, rta, `out = decimal(1.23)`, nil, dec128.FromFloat64(1.23))
	expectRun(t, rta, `out = decimal("1.23")`, nil, dec128.FromString("1.23"))

	expectRun(t, rta, `out = (123).decimal()`, nil, dec128.FromInt64(123))
	expectRun(t, rta, `out = (1.23).decimal()`, nil, dec128.FromFloat64(1.23))
	expectRun(t, rta, `out = "1.23".decimal()`, nil, dec128.FromString("1.23"))

	expectRun(t, rta, `out = decimal(1) + decimal(2)`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = decimal(1) + 2`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1 + decimal(2)`, nil, dec128.FromString("3"))

	expectRun(t, rta, `out = 1.0 + decimal(2)`, nil, 3.0)
	expectRun(t, rta, `out = decimal(1) + 2.0`, nil, dec128.FromString("3"))

	expectRun(t, rta, `out = 1d`, nil, dec128.FromInt64(1))
	expectRun(t, rta, `out = 1.23d`, nil, dec128.FromString("1.23"))
	expectRun(t, rta, `out = type_name(1d)`, nil, "decimal")
	expectRun(t, rta, `out = type_name(1.23d)`, nil, "decimal")
	expectRun(t, rta, `out = 1d + 2d`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1d + 2`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1 + 2d`, nil, dec128.FromString("3"))
	expectRun(t, rta, `out = 1.5d + 0.5d`, nil, dec128.FromString("2"))
	expectRun(t, rta, `out = -1d`, nil, dec128.FromInt64(-1))

	expectRun(t, rta, `out = (1.23d).decimal()`, nil, dec128.FromString("1.23"))
	expectRun(t, rta, `out = (123d).float().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, rta, `out = (123d).int().decimal()`, nil, dec128.FromString("123"))
	expectRun(t, rta, `out = (1.23d).string()`, nil, "1.23")
	expectRun(t, rta, `out = (1.23d).is_zero()`, nil, false)
	expectRun(t, rta, `out = (0d).is_zero()`, nil, true)
	expectRun(t, rta, `out = (0d).is_negative()`, nil, false)
	expectRun(t, rta, `out = (1d).is_negative()`, nil, false)
	expectRun(t, rta, `out = (-1d).is_negative()`, nil, true)
	expectRun(t, rta, `out = (0d).is_positive()`, nil, false)
	expectRun(t, rta, `out = (1d).is_positive()`, nil, true)
	expectRun(t, rta, `out = (-1d).is_positive()`, nil, false)
	expectRun(t, rta, `out = (0d).sign()`, nil, 0)
	expectRun(t, rta, `out = (1d).sign()`, nil, 1)
	expectRun(t, rta, `out = (-1d).sign()`, nil, -1)
	expectRun(t, rta, `out = (123d).rescale(2).scale()`, nil, 2)
	expectRun(t, rta, `out = (123d).rescale(2).canonical().scale()`, nil, 0)
	expectRun(t, rta, `out = (1.23d).format()`, nil, "1.23")
	expectRun(t, rta, `out = (1.23d).format("v")`, nil, "1.23d")
}

func TestRune(t *testing.T) {
	expectRun(t, rta, `out = 'a'`, nil, 'a')
	expectRun(t, rta, `out = 'あ'`, nil, rune(12354))
	expectRun(t, rta, `out = 'Æ'`, nil, rune(198))

	expectRun(t, rta, `out = '0' + '9'`, nil, rune(105))
	expectRun(t, rta, `out = '0' + 9`, nil, 57) // '0' is 48 in ASCII
	expectRun(t, rta, `out = '9' - 4`, nil, 53) // '9' is 57 in ASCII
	expectRun(t, rta, `out = '0' == '0'`, nil, true)
	expectRun(t, rta, `out = '0' != '0'`, nil, false)
	expectRun(t, rta, `out = '2' < '4'`, nil, true)
	expectRun(t, rta, `out = '2' > '4'`, nil, false)
	expectRun(t, rta, `out = '2' <= '4'`, nil, true)
	expectRun(t, rta, `out = '2' >= '4'`, nil, false)
	expectRun(t, rta, `out = '4' < '4'`, nil, false)
	expectRun(t, rta, `out = '4' > '4'`, nil, false)
	expectRun(t, rta, `out = '4' <= '4'`, nil, true)
	expectRun(t, rta, `out = '4' >= '4'`, nil, true)

	v := core.RuneValue('A')
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "A", s)
	v = core.RuneValue('A')
	require.Equal(t, rta, "'A'", v.String(rta))

	v = core.RuneValue('0')
	expectRun(t, rta, fmt.Sprintf(`out = '0' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('A')
	expectRun(t, rta, fmt.Sprintf(`out = 'A' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('₴')
	expectRun(t, rta, fmt.Sprintf(`out = '₴' == %s`, v.String(rta)), nil, true)
	v = core.RuneValue('\'')
	expectRun(t, rta, fmt.Sprintf(`out = '\'' == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = '4' + 4`, nil, 56) // '4' is 52 in ASCII
	expectRun(t, rta, `out = '4' + "4"`, nil, "44")
	expectError(t, rta, `'4' - "4"`, nil, "invalid_binary_operator: rune - string")

	expectRun(t, rta, `out = '4'.rune()`, nil, '4')
	expectRun(t, rta, `out = '4'.bool()`, nil, true)
	expectRun(t, rta, `out = '4'.int()`, nil, 52)
	expectRun(t, rta, `out = '4'.string()`, nil, "4")
	expectRun(t, rta, `out = '4'.format()`, nil, "4")
	expectRun(t, rta, `out = '4'.format("v")`, nil, "'4'")
}

func TestString(t *testing.T) {
	expectRun(t, rta, `out = "Hello World!"`, nil, "Hello World!")
	expectRun(t, rta, `out = "Hello" + " " + "World!"`, nil, "Hello World!")

	expectRun(t, rta, `out = "Hello" == "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello" == "World"`, nil, false)
	expectRun(t, rta, `out = "Hello" != "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello" != "World"`, nil, true)

	expectRun(t, rta, `out = "Hello" > "World"`, nil, false)
	expectRun(t, rta, `out = "World" < "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello" < "World"`, nil, true)
	expectRun(t, rta, `out = "World" > "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello" >= "World"`, nil, false)
	expectRun(t, rta, `out = "Hello" <= "World"`, nil, true)
	expectRun(t, rta, `out = "Hello" >= "Hello"`, nil, true)
	expectRun(t, rta, `out = "World" <= "World"`, nil, true)
	expectRun(t, rta, `out = "el" in "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello".contains("el")`, nil, true)
	expectRun(t, rta, `out = 'e' in "Hello"`, nil, true)
	expectRun(t, rta, `out = "Hello".contains('e')`, nil, true)
	expectRun(t, rta, `out = "z" in "Hello"`, nil, false)
	expectRun(t, rta, `out = "Hello".contains("z")`, nil, false)
	expectRun(t, rta, `out = "z" not in "Hello"`, nil, true)

	// index operator
	str := "abcdef"
	strStr := `"abcdef"`
	strLen := 6
	for idx := range strLen {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1]", strStr, idx), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("idx = %d; out = %s[idx]", idx, strStr), nil, str[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, -idx-1), nil, str[strLen-idx-1])
	}

	expectError(t, rta, fmt.Sprintf("%s[%d]", strStr, -strLen-1), nil, "index_out_of_bounds")
	expectError(t, rta, fmt.Sprintf("%s[%d]", strStr, strLen), nil, "index_out_of_bounds")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d]", strStr, -2), nil, str[strLen-2])

	// slice operator
	for low := 0; low <= strLen; low++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, low, low), nil, "")
		for high := low; high <= strLen; high++ {
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", strStr, low, high), nil, str[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, high), nil, str[:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", strStr, low), nil, str[low:])
		}
	}

	expectRun(t, rta, fmt.Sprintf("out = %s[:]", strStr), nil, str[:])
	expectRun(t, rta, fmt.Sprintf("out = %s[:]", strStr), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", strStr, -1), nil, str[strLen-1:])
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, strLen+1), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 2), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", strStr, -1), nil, str[:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 0, -1), nil, str[:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, -3, -1), nil, str[strLen-3:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 1, -1), nil, str[1:strLen-1])
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 2, 1), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, 10, 20), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", strStr, -100, 100), nil, str)
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:2]", strStr), nil, "bd")
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:-1]", strStr), nil, "")
	expectRun(t, rta, fmt.Sprintf("out = %s[5:1:-1]", strStr), nil, "fedc")
	expectRun(t, rta, fmt.Sprintf("out = %s[0:%d:2]", strStr, strLen), nil, "ace")
	expectRun(t, rta, fmt.Sprintf("out = %s[::-1]", strStr), nil, "fedcba")
	expectError(t, rta, fmt.Sprintf("out = %s[::0]", strStr), nil, "step cannot be zero")

	// string concatenation with other types
	expectRun(t, rta, `out = "foo" + 1`, nil, "foo1")
	// Float.string() returns the smallest number of digits necessary such that ParseFloat will return f exactly.
	expectRun(t, rta, `out = "foo" + 1.0`, nil, "foo1") // <- note '1' instead of '1.0'
	expectRun(t, rta, `out = "foo" + 1.5`, nil, "foo1.5")
	expectRun(t, rta, `out = "foo" + true`, nil, "footrue")
	expectRun(t, rta, `out = "foo" + 'X'`, nil, "fooX")
	expectRun(t, rta, `out = "foo" + error(5)`, nil, "foo5")
	expectRun(t, rta, `out = "foo" + [100, 101]`, nil, "foode")
	// also works with "+=" operator
	expectRun(t, rta, `out = "foo"; out += 1.5`, nil, "foo1.5")

	// string concat works only when string is LHS
	expectError(t, rta, `1 + "foo"`, nil, "invalid_binary_operator: int + string")

	// there is no '-' operator for string
	expectError(t, rta, `"foo" - "bar"`, nil, "invalid_binary_operator: string - string")

	// undefined cannot be added to string
	expectError(t, rta, `"foo" + undefined`, nil, "invalid_binary_operator: string + undefined")

	v := rta.NewStringValue("abc")
	s, _ := v.AsString(rta)
	require.Equal(t, rta, "abc", s)
	v = rta.NewStringValue("abc")
	require.Equal(t, rta, `"abc"`, v.String(rta))

	v = rta.NewStringValue("")
	expectRun(t, rta, fmt.Sprintf(`out = "" == %s`, v.String(rta)), nil, true)
	v = rta.NewStringValue("hello")
	expectRun(t, rta, fmt.Sprintf(`out = "hello" == %s`, v.String(rta)), nil, true)
	v = rta.NewStringValue("hello \"world\"")
	expectRun(t, rta, fmt.Sprintf(`out = "hello \"world\"" == %s`, v.String(rta)), nil, true)
	v = rta.NewStringValue("123₴")
	expectRun(t, rta, fmt.Sprintf(`out = "123₴" == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = "".is_empty()`, nil, true)
	expectRun(t, rta, `out = "abcd".is_empty()`, nil, false)
	expectRun(t, rta, `out = "abcd".len()`, nil, 4)
	expectRun(t, rta, `out = "Abcd".lower()`, nil, "abcd")
	expectRun(t, rta, `out = "Abcd".upper()`, nil, "ABCD")
	expectRun(t, rta, `out = "abcd ".trim()`, nil, "abcd")
	expectRun(t, rta, `out = "abcd".trim("ad")`, nil, "bc")
	expectRun(t, rta, `out = "".reverse()`, nil, "")
	expectRun(t, rta, `out = "a".reverse()`, nil, "a")
	expectRun(t, rta, `out = "hello".reverse()`, nil, "olleh")
	expectRun(t, rta, `out = "їЇґҐ".reverse()`, nil, "ҐґЇї")
	expectRun(t, rta, `out = "こんにちは".reverse()`, nil, "はちにんこ")

	expectRun(t, rta, `out = "abc".string()`, nil, "abc")
	expectRun(t, rta, `out = "abc".array()`, nil, ARR{int64('a'), int64('b'), int64('c')})
	expectRun(t, rta, `out = "abc".array().string()`, nil, "abc")
	expectRun(t, rta, `out = "true".bool()`, nil, true)
	expectRun(t, rta, `out = "false".bool()`, nil, false)
	expectRun(t, rta, `out = "abc".bool()`, nil, false)
	expectRun(t, rta, `out = "true".bool().string()`, nil, "true")
	expectRun(t, rta, `out = "abc".bytes()`, nil, rta.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, rta, `out = "abc".bytes().string()`, nil, "abc")
	expectRun(t, rta, `out = "1.2".float()`, nil, 1.2)
	expectRun(t, rta, `out = "1.2".float().string()`, nil, "1.2")
	expectRun(t, rta, `out = "12".byte()`, nil, byte(12))
	expectRun(t, rta, `out = u"12".byte()`, nil, byte(12))
	expectRun(t, rta, `out = "12".int()`, nil, 12)
	expectRun(t, rta, `out = "12".float().string()`, nil, "12")
	expectRun(t, rta, `out = "abc".int()`, nil, 0)
	expectRun(t, rta, `out = "abc".record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = "abc".dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = "abc".format()`, nil, "abc")
	expectRun(t, rta, `out = "abc".format("v")`, nil, `"abc"`)

	expectRun(t, rta, `out = " їЇґҐ ".trim()`, nil, "їЇґҐ")
	expectRun(t, rta, `out = "їЇґҐ".upper()`, nil, "ЇЇҐҐ")
	expectRun(t, rta, `out = "їЇґҐ".lower()`, nil, "їїґґ")
	expectRun(t, rta, `out = "こんにちはさ"[1]`, nil, byte(129)) // byte index, not rune index
	expectRun(t, rta, `out = "こんにちはさ"[1:2]`, nil, "\x81")  // byte slice, not rune slice
	expectRun(t, rta, `out = "こんにちはさ"[0:3]`, nil, "こ")     // byte slice, not rune slice

	expectRun(t, rta, `out = len("")`, nil, 0)
	expectRun(t, rta, `out = len("hello")`, nil, 5)
	expectRun(t, rta, `out = len("їЇґҐ")`, nil, 8)    // byte length, not rune length
	expectRun(t, rta, `out = len("こんにちはさ")`, nil, 18) // byte length, not rune length

	expectRun(t, rta, `out = "hello".filter(x => x > 'e')`, nil, "hllo")
	expectRun(t, rta, `out = "hello".filter((i, x) => i > 2)`, nil, "lo")
	expectRun(t, rta, `out = "hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = "hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = "hello".all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = "hello".all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = "hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = "hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = "hello".any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = "hello".any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = "hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = "hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = "hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = "hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = "hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = "hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = "".find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = "x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = "x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = "x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `
out = ""
ignored := "hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, rta, `
out = 0
ignored := "abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunes(t *testing.T) {
	expectRun(t, rta, `out = u"Hello World!"`, nil, []rune("Hello World!"))
	expectRun(t, rta, `out = u"Hello" + u" " + "World!"`, nil, []rune("Hello World!"))

	expectRun(t, rta, `out = u"Hello" == "Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" == u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" == u"World"`, nil, false)
	expectRun(t, rta, `out = u"Hello" != u"Hello"`, nil, false)
	expectRun(t, rta, `out = u"Hello" != u"World"`, nil, true)

	expectRun(t, rta, `out = u"Hello" > u"World"`, nil, false)
	expectRun(t, rta, `out = u"World" < u"Hello"`, nil, false)
	expectRun(t, rta, `out = u"Hello" < u"World"`, nil, true)
	expectRun(t, rta, `out = u"World" > u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"Hello" >= u"World"`, nil, false)
	expectRun(t, rta, `out = u"Hello" <= u"World"`, nil, true)
	expectRun(t, rta, `out = u"Hello" >= u"Hello"`, nil, true)
	expectRun(t, rta, `out = u"World" <= u"World"`, nil, true)
	expectRun(t, rta, `out = u"el" in u"Hello"`, nil, true)
	expectRun(t, rta, `out = runes("Hello").contains(u"el")`, nil, true)
	expectRun(t, rta, `out = 'e' in u"Hello"`, nil, true)
	expectRun(t, rta, `out = runes("Hello").contains('e')`, nil, true)
	expectRun(t, rta, `out = runes("z") in u"Hello"`, nil, false)
	expectRun(t, rta, `out = runes("Hello").contains(u"z")`, nil, false)
	expectRun(t, rta, `out = runes("z") not in u"Hello"`, nil, true)

	expectRun(t, rta, `out = runes("").is_empty()`, nil, true)
	expectRun(t, rta, `out = runes("abcd").is_empty()`, nil, false)
	expectRun(t, rta, `out = runes("abcd").len()`, nil, 4)
	expectRun(t, rta, `out = runes("abcd").first()`, nil, 'a')
	expectRun(t, rta, `out = runes("abcd").last()`, nil, 'd')
	expectRun(t, rta, `out = runes("Abcd").lower()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = runes("Abcd").upper()`, nil, []rune("ABCD"))
	expectRun(t, rta, `out = runes("abcd ").trim()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = runes("abcd").trim("ad")`, nil, []rune("bc"))
	expectRun(t, rta, `out = runes("").reverse()`, nil, []rune(""))
	expectRun(t, rta, `out = runes("hello").reverse()`, nil, []rune("olleh"))
	expectRun(t, rta, `out = u"hello".reverse()`, nil, []rune("olleh"))
	expectRun(t, rta, `out = u"їЇґҐ".reverse()`, nil, []rune("ҐґЇї"))
	expectRun(t, rta, `out = u"こんにちは".reverse()`, nil, []rune("はちにんこ"))

	expectRun(t, rta, `out = runes("abc").string()`, nil, "abc")
	expectRun(t, rta, `out = runes("abc").array()`, nil, ARR{'a', 'b', 'c'})
	expectRun(t, rta, `out = runes("abc").array().string()`, nil, "abc")
	expectRun(t, rta, `out = runes("true").bool()`, nil, true)
	expectRun(t, rta, `out = runes("false").bool()`, nil, false)
	expectRun(t, rta, `out = runes("abc").bool()`, nil, false)
	expectRun(t, rta, `out = runes("true").bool().string()`, nil, "true")
	expectRun(t, rta, `out = runes("abc").bytes()`, nil, rta.NewBytesValue([]byte{'a', 'b', 'c'}, false))
	expectRun(t, rta, `out = runes("abc").bytes().string()`, nil, "abc")
	expectRun(t, rta, `out = runes("1.2").float()`, nil, 1.2)
	expectRun(t, rta, `out = runes("1.2").float().string()`, nil, "1.2")
	expectRun(t, rta, `out = runes("12").int()`, nil, 12)
	expectRun(t, rta, `out = runes("12").float().string()`, nil, "12")
	expectRun(t, rta, `out = runes("abc").int()`, nil, 0)
	expectRun(t, rta, `out = runes("abc").record()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})
	expectRun(t, rta, `out = runes("abc").dict()`, nil, MAP{"0": 'a', "1": 'b', "2": 'c'})

	expectRun(t, rta, `out = runes(" їЇґҐ ").trim()`, nil, []rune("їЇґҐ"))
	expectRun(t, rta, `out = u" їЇґҐ ".trim()`, nil, []rune("їЇґҐ"))

	expectRun(t, rta, `out = u"їЇґҐ".upper()`, nil, []rune("ЇЇҐҐ"))
	expectRun(t, rta, `out = u"їЇґҐ".lower()`, nil, []rune("їїґґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1]`, nil, 'Ї')
	expectRun(t, rta, `out = u"їЇґҐ"[-1]`, nil, 'Ґ')
	expectRun(t, rta, `out = u"їЇґҐ"[-2]`, nil, 'ґ')
	expectRun(t, rta, `out = u"їЇґҐ"[1:2]`, nil, []rune("Ї"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:3]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[:-1]`, nil, []rune("їЇґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:-1]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[-3:-1]`, nil, []rune("Їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[10:20]`, nil, []rune(""))
	expectRun(t, rta, `out = u"їЇґҐ"[1:4:2]`, nil, []rune("ЇҐ"))
	expectRun(t, rta, `out = u"їЇґҐ"[1:4:-1]`, nil, []rune(""))
	expectRun(t, rta, `out = u"їЇґҐ"[3:0:-1]`, nil, []rune("ҐґЇ"))
	expectRun(t, rta, `out = u"їЇґҐ"[0:4:2]`, nil, []rune("їґ"))
	expectRun(t, rta, `out = u"їЇґҐ"[::-1]`, nil, []rune("ҐґЇї"))
	expectError(t, rta, `out = u"їЇґҐ"[::0]`, nil, "step cannot be zero")
	expectRun(t, rta, `out = u"こんにちはさ"[1]`, nil, 'ん')
	expectRun(t, rta, `out = u"こんにちはさ"[1:2]`, nil, []rune("ん"))
	expectRun(t, rta, `out = u"こんにちはさ"[1:3]`, nil, []rune("んに"))
	expectRun(t, rta, `out = u"こんにちはさ"[-2:]`, nil, []rune("はさ"))
	expectError(t, rta, `out = u"こんにちはさ"[-7]`, nil, "index_out_of_bounds")

	expectRun(t, rta, `out = len(u"")`, nil, 0)
	expectRun(t, rta, `out = len(u"hello")`, nil, 5)
	expectRun(t, rta, `out = len(u"їЇґҐ")`, nil, 4)
	expectRun(t, rta, `out = len(u"こんにちはさ")`, nil, 6)

	expectRun(t, rta, `out = runes("abc").format()`, nil, "abc")
	expectRun(t, rta, `out = runes("abc").format("v")`, nil, `u"abc"`)

	expectRun(t, rta, `out = u"hello".sort()`, nil, []rune("ehllo"))
	expectRun(t, rta, `out = u"".dedup()`, nil, []rune(""))
	expectRun(t, rta, `out = u"aabbccd".dedup()`, nil, []rune("abcd"))
	expectRun(t, rta, `out = u"abc".dedup()`, nil, []rune("abc"))
	expectRun(t, rta, `out = u"aaaa".dedup()`, nil, []rune("a"))
	expectRun(t, rta, `out = u"abab".dedup()`, nil, []rune("abab"))
	expectRun(t, rta, `out = u"hello".sort().dedup()`, nil, []rune("ehlo"))
	expectRun(t, rta, `out = u"їЇїЇ".dedup()`, nil, []rune("їЇїЇ"))
	expectRun(t, rta, `out = u"їїЇЇ".dedup()`, nil, []rune("їЇ"))
	expectRun(t, rta, `out = u"".unique()`, nil, []rune(""))
	expectRun(t, rta, `out = u"abc".unique()`, nil, []rune("abc"))
	expectRun(t, rta, `out = u"hello".unique()`, nil, []rune("helo"))
	expectRun(t, rta, `out = u"abab".unique()`, nil, []rune("ab"))
	expectRun(t, rta, `out = u"їЇїЇ".unique()`, nil, []rune("їЇ"))
	expectRun(t, rta, `out = u"".chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = u"hello".chunk(2)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, rta, `out = u"hello".chunk(2, true)`, nil, ARR{[]rune("he"), []rune("ll"), []rune("o")})
	expectRun(t, rta, `out = u"hello".chunk(10)`, nil, ARR{[]rune("hello")})
	expectRun(t, rta, `out = u"hello".filter(x => x > 'e')`, nil, []rune("hllo"))
	expectRun(t, rta, `out = u"hello".filter((i, x) => i > 2)`, nil, []rune("lo"))
	expectRun(t, rta, `out = u"hello".count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = u"hello".count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = u"hello".all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = u"hello".all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = u"hello".all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = u"hello".all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = u"hello".any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = u"hello".any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = u"hello".any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = u"hello".any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = u"hello".find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = u"hello".find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = u"hello".find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = u"hello".find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = u"".find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = u"x".find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = u"x".find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = u"x".find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `out = u"hello".min()`, nil, 'e')
	expectRun(t, rta, `out = u"hello".max()`, nil, 'o')
	expectRun(t, rta, `
out = ""
ignored := u"hello".for_each(func(r) {
	out += r.string()
	return r != 'l'
})
`, nil, "hel")
	expectRun(t, rta, `
out = 0
ignored := u"abc".for_each(func(i, r) {
	out += i + r.int()
	return true
})
`, nil, 297)
}

func TestRunesMutability(t *testing.T) {
	// index assignment
	expectRun(t, rta, `r := runes("hello"); r[0] = 'H'; out = r`, nil, []rune("Hello"))
	expectRun(t, rta, `r := runes("hello"); r[-2] = '!'; out = r`, nil, []rune("hel!o"))
	expectRun(t, rta, `r := runes("hello"); r[0] = 0x41; out = r`, nil, []rune("Aello"))

	// append
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c'); out = r2`, nil, []rune("abc"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c', 'd'); out = r2`, nil, []rune("abcd"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, runes("cd")); out = r2`, nil, []rune("abcd"))
	expectRun(t, rta, `r := runes("ab"); r2 := append(r, 'c'); out = r`, nil, []rune("ab"))

	// sum / avg / map / reduce
	expectRun(t, rta, `out = runes("abc").sum()`, nil, 97+98+99)
	expectRun(t, rta, `out = runes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, rta, `out = runes("").sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = runes("").avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = runes("abc").map(func(r) { return r + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, rta, `out = runes("abc").map(func(i, r) { return [i, r] })`, nil, ARR{ARR{0, 'a'}, ARR{1, 'b'}, ARR{2, 'c'}})
	expectRun(t, rta, `out = runes("abc").reduce(0, func(acc, r) { return acc + r })`, nil, int64('a'+'b'+'c'))
	expectRun(t, rta, `out = runes("abc").reduce("", func(acc, i, r) { return acc + i.string() + r.string() })`, nil, "0a1b2c")

	// type names
	expectRun(t, rta, `out = type_name(runes("abc"))`, nil, "runes")
	expectRun(t, rta, `out = type_name(immutable(runes("abc")))`, nil, "immutable-runes")

	// immutable rejects writes
	expectError(t, rta, `r := immutable(runes("abc")); r[0] = 'X'`, nil, "not_assignable: type immutable-runes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, rta, `out = type_name(immutable(runes("abcd"))[1:3])`, nil, "immutable-runes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, rta, `out = type_name(immutable(runes("abcd"))[::-1])`, nil, "runes")
	// slice of mutable stays mutable
	expectRun(t, rta, `out = type_name(runes("abcd")[1:3])`, nil, "runes")

	// copy of immutable yields mutable
	expectRun(t, rta, `r := immutable(runes("abc")); c := copy(r); c[0] = 'X'; out = c`, nil, []rune("Xbc"))

	// append on immutable returns a fresh mutable value (does not mutate source)
	expectRun(t, rta, `r := immutable(runes("ab")); r2 := append(r, 'c'); r2[0] = 'X'; out = r2`, nil, []rune("Xbc"))
	expectRun(t, rta, `r := immutable(runes("ab")); r2 := append(r, 'c'); out = type_name(r2)`, nil, "runes")

	// invalid assignment values
	expectError(t, rta, `r := runes("abc"); r[0] = "xy"`, nil, "invalid_index_type: (index assign value) expected rune, got string")
	expectError(t, rta, `r := runes("abc"); r[10] = 'X'`, nil, "index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestError(t *testing.T) {
	expectError(t, rta, `out = error()`, nil, "wrong_num_arguments: (error) expected 1 or 2 argument(s), got 0")
	expectRun(t, rta, `out = error(1)`, nil, errorObject(rta, 1))
	expectRun(t, rta, `out = error(1).value()`, nil, 1)
	expectRun(t, rta, `out = error("some error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = error("some" + " error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = func() { return error(5) }()`, nil, errorObject(rta, 5))
	expectRun(t, rta, `out = error(error("foo"))`, nil, errorObject(rta, errorObject(rta, "foo")))
	expectRun(t, rta, `out = error("some error")`, nil, errorObject(rta, "some error"))
	expectRun(t, rta, `out = error("some error").value()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").string()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").format()`, nil, "some error")
	expectRun(t, rta, `out = error("some error").format("v")`, nil, `error("some error")`)

	expectRun(t, rta, `out = error("x").is_fatal()`, nil, false)
	expectRun(t, rta, `out = error("x", false).is_fatal()`, nil, false)
	expectRun(t, rta, `out = error("x", true).is_fatal()`, nil, true)
	expectError(t, rta, `out = error("x").is_fatal(1)`, nil, "wrong_num_arguments: (is_fatal) expected 0 argument(s), got 1")

	expectError(t, rta, `error("error").err`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, rta, `error("error").value_`, nil, "not_accessible: type error does not support indexing or field access")
	expectError(t, rta, `error([1,2,3])[1]`, nil, "not_accessible: type error does not support indexing or field access")

	s, _ := rta.NewErrorValue(rta.NewStringValue("abc"), core.KindUser, false).AsString(rta)
	require.Equal(t, rta, "abc", s)
	require.Equal(t, rta, `error("abc")`, rta.NewErrorValue(rta.NewStringValue("abc"), core.KindUser, false).String(rta))

	v := rta.NewErrorValue(core.Undefined, core.KindUser, false)
	require.Equal(t, rta, "error()", v.String(rta))
	expectRun(t, rta, `out = error(undefined) == error(undefined)`, nil, true)
	v = rta.NewErrorValue(rta.NewStringValue("some error"), core.KindUser, false)
	expectRun(t, rta, fmt.Sprintf(`out = error("some error") == %s`, v.String(rta)), nil, true)
}

func TestArray(t *testing.T) {
	expectRun(t, rta, `out = [1, 2 * 2, 3 + 3]`, nil, ARR{1, 4, 6})

	// array copy-by-reference
	expectRun(t, rta, `a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2`, nil, ARR{5, 2, 3})
	expectRun(t, rta, `func () { a1 := [1, 2, 3]; a2 := a1; a1[0] = 5; out = a2 }()`, nil, ARR{5, 2, 3})

	// array index set
	expectError(t, rta, `a1 := [1, 2, 3]; a1[3] = 5`, nil, "index_out_of_bounds")

	// index operator
	arr := ARR{1, 2, 3, 4, 5, 6}
	arrStr := `[1, 2, 3, 4, 5, 6]`
	arrLen := 6
	for idx := 0; idx < arrLen; idx++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1]", arrStr, idx), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("idx := %d; out = %s[idx]", idx, arrStr), nil, arr[idx])
		expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, -idx-1), nil, arr[arrLen-idx-1])
	}

	expectError(t, rta, fmt.Sprintf("%s[%d]", arrStr, -arrLen-1), nil, "index_out_of_bounds")
	expectError(t, rta, fmt.Sprintf("%s[%d]", arrStr, arrLen), nil, "index_out_of_bounds")
	expectRun(t, rta, fmt.Sprintf("out = %s[%d]", arrStr, -2), nil, arr[arrLen-2])
	expectRun(t, rta, `a1 := [1, 2, 3]; a1[-1] = 5; out = a1[2]`, nil, 5)
	expectError(t, rta, `a1 := [1, 2, 3]; a1[-4] = 5`, nil, "index_out_of_bounds")

	// slice operator
	for low := 0; low < arrLen; low++ {
		expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, low), nil, ARR{})
		for high := low; high <= arrLen; high++ {
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[0 + %d : 0 + %d]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[1 + %d - 1 : 1 + %d - 1]", arrStr, low, high), nil, arr[low:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, high), nil, arr[:high])
			expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", arrStr, low), nil, arr[low:])
		}
	}

	expectRun(t, rta, fmt.Sprintf("out = %s[:]", arrStr), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:]", arrStr, -1), nil, ARR{6})
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, arrLen+1), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 2), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[:%d]", arrStr, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 0, -1), nil, ARR{1, 2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 1, -1), nil, ARR{2, 3, 4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, -3, -1), nil, ARR{4, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 2, 1), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, 10, 20), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d]", arrStr, -100, 100), nil, arr)
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:2]", arrStr), nil, ARR{2, 4})
	expectRun(t, rta, fmt.Sprintf("out = %s[1:5:-1]", arrStr), nil, ARR{})
	expectRun(t, rta, fmt.Sprintf("out = %s[5:1:-1]", arrStr), nil, ARR{6, 5, 4, 3})
	expectRun(t, rta, fmt.Sprintf("out = %s[%d:%d:%d]", arrStr, 0, arrLen, 2), nil, ARR{1, 3, 5})
	expectRun(t, rta, fmt.Sprintf("out = %s[::-1]", arrStr), nil, ARR{6, 5, 4, 3, 2, 1})
	expectError(t, rta, fmt.Sprintf("out = %s[::0]", arrStr), nil, "step cannot be zero")

	v := rta.NewArrayValue(nil, false)
	expectRun(t, rta, fmt.Sprintf(`out = [] == %s`, v.String(rta)), nil, true)
	v = rta.NewArrayValue(nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = [] == %s`, v.String(rta)), nil, true)

	v = rta.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.Undefined,
		rta.NewStringValue("3"),
	}, false)
	expectRun(t, rta, fmt.Sprintf(`out = [1, undefined, "3"] == %s`, v.String(rta)), nil, true)

	expectError(t, rta, `[1, 2, 3].q`, nil, "Runtime Error: invalid_selector: type array has no property \"q\"\n\tat test:1:11")

	expectRun(t, rta, `t := []; out = t.sort()`, nil, ARR{})
	expectRun(t, rta, `t := [1, 2, 3]; out = t.sort()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `t := [3, 2, 1]; out = t.sort()`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `out = [].dedup()`, nil, ARR{})
	expectRun(t, rta, `out = [1].dedup()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 1, 2, 2, 3, 3, 3, 1].dedup()`, nil, ARR{1, 2, 3, 1})
	expectRun(t, rta, `out = [1, 2, 3].dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [1, 2, 1, 2].dedup()`, nil, ARR{1, 2, 1, 2})
	expectRun(t, rta, `out = [3, 1, 2, 1, 3, 2].sort().dedup()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = ["a", "a", "b", "a"].dedup()`, nil, ARR{"a", "b", "a"})
	expectRun(t, rta, `out = [1, 1.0, "1"].dedup()`, nil, ARR{1})
	expectRun(t, rta, `out = [[1, 2], [1, 2], [3]].dedup()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, rta, `out = [].unique()`, nil, ARR{})
	expectRun(t, rta, `out = [1].unique()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 2, 3].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [1, 1, 2, 2, 3, 3, 3, 1].unique()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [3, 1, 2, 1, 3, 2].unique()`, nil, ARR{3, 1, 2})
	expectRun(t, rta, `out = ["a", "b", "a", "c", "b"].unique()`, nil, ARR{"a", "b", "c"})
	expectRun(t, rta, `out = [1, 1.0, "1"].unique()`, nil, ARR{1})
	expectRun(t, rta, `out = [[1, 2], [3], [1, 2]].unique()`, nil, ARR{ARR{1, 2}, ARR{3}})

	expectRun(t, rta, `out = [].reverse()`, nil, ARR{})
	expectRun(t, rta, `out = [1].reverse()`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 2, 3].reverse()`, nil, ARR{3, 2, 1})
	expectRun(t, rta, `out = ["a", "b", "c"].reverse()`, nil, ARR{"c", "b", "a"})
	expectRun(t, rta, `out = [1, 2, 3].reverse().reverse()`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `t := []; out = t.is_empty()`, nil, true)
	expectRun(t, rta, `t := [1, 2, 3]; out = t.is_empty()`, nil, false)

	expectRun(t, rta, `t := []; out = t.len()`, nil, 0)
	expectRun(t, rta, `t := [1, 2, 3]; out = t.len()`, nil, 3)

	expectRun(t, rta, `out = [].first()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].first()`, nil, 1)

	expectRun(t, rta, `out = [].last()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].last()`, nil, 3)

	expectRun(t, rta, `out = [].min()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].min()`, nil, 1)

	expectRun(t, rta, `out = [].max()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].max()`, nil, 3)

	expectRun(t, rta, `out = [].sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].sum()`, nil, 6)

	expectRun(t, rta, `out = [].avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = [1, 2, 3].avg()`, nil, 2)

	expectRun(t, rta, `out = [].count(x => x > 0)`, nil, 0)
	expectRun(t, rta, `out = [1, 2, 3, -10].count(x => x > 0)`, nil, 3)
	expectRun(t, rta, `out = [1, 2, 3, -10].count((i, x) => x == i+1)`, nil, 3)

	expectRun(t, rta, `out = [1, 2, 3].filter(x => x == 2)`, nil, ARR{2})
	expectRun(t, rta, `out = [1, 2, 3].filter(x => x != 2)`, nil, ARR{1, 3})
	expectRun(t, rta, `out = [1, undefined, 2, undefined, 3].filter()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [].filter()`, nil, ARR{})
	expectRun(t, rta, `out = [undefined, undefined].filter()`, nil, ARR{})

	expectRun(t, rta, `out = [].all(x => x > 0)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].all(x => x > 0)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].all(x => x > -100)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].all((i, x) => x == i+1)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, 4].all((i, x) => x == i+1)`, nil, true)

	expectRun(t, rta, `out = [].any(x => x > 0)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].any(x => x < 0)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, -10].any(x => x < -100)`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3, -10].any((i, x) => x != i+1)`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3, 4].any((i, x) => x != i+1)`, nil, false)

	expectRun(t, rta, `out = [].map(x => x * x)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3].map(x => x * x)`, nil, ARR{1, 4, 9})

	expectRun(t, rta, `out = [].chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3, 4].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}})
	expectRun(t, rta, `out = [1, 2, 3, 4, 5].chunk(2)`, nil, ARR{ARR{1, 2}, ARR{3, 4}, ARR{5}})
	expectRun(t, rta, `out = [1, 2, 3].chunk(10)`, nil, ARR{ARR{1, 2, 3}})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2, false); c[0][0] = 9; out = a`, nil, ARR{9, 2, 3})
	expectRun(t, rta, `a := [1, 2, 3]; c := a.chunk(2, true); c[0][0] = 9; out = a`, nil, ARR{1, 2, 3})
	expectError(t, rta, `out = [1, 2, 3].chunk()`, nil, "wrong_num_arguments: (chunk) expected 1 or 2 argument(s), got 0")
	expectError(t, rta, `out = [1, 2, 3].chunk("x")`, nil, "invalid_argument_type: (chunk) argument first expects type int, got string")
	expectError(t, rta, `out = [1, 2, 3].chunk(2, 1)`, nil, "invalid_argument_type: (chunk) argument second expects type bool, got int")
	expectError(t, rta, `out = [1, 2, 3].chunk(0)`, nil, "invalid_value: chunk size must be positive")
	expectError(t, rta, `out = [1, 2, 3].chunk(-1)`, nil, "invalid_value: chunk size must be positive")

	expectRun(t, rta, `
out = 0
ignored := [1, 2, 3, 4].for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)

	expectRun(t, rta, `
out = 0
ignored := [10, 20, 30].for_each(func(i, v) {
	out += i * v
	return true
})
`, nil, 80)

	expectRun(t, rta, `out = [1].for_each(func(v) { return true })`, nil, core.Undefined)
	expectError(t, rta, `out = [1].for_each()`, nil, "wrong_num_arguments: (for_each) expected 1 argument(s), got 0")
	expectError(t, rta, `out = [1].for_each(1)`, nil, "invalid_argument_type: (for_each) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = [1].for_each(func() { return true })`, nil, "invalid_argument_type: (for_each) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = [10, 20, 30].find(x => x == 20)`, nil, 1)
	expectRun(t, rta, `out = [10, 20, 30].find(x => x == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = [10, 20, 30].find((i, v) => i == 2)`, nil, 2)
	expectRun(t, rta, `out = [10, 20, 30].find((i, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = [].find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = [1].find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = [1].find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = [1].find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = [].reduce(0, (a, v) => a + v)`, nil, 0)
	expectRun(t, rta, `out = [1, 2, 3].reduce(0, (a, v) => a + v)`, nil, 6)
	expectRun(t, rta, `out = [1, 2, 3].reduce(0, (a, i, v) => a + i)`, nil, 3)
	expectRun(t, rta, `out = [1, 2].reduce(0, (a, v) => a + [10, 20].reduce(0, (b, w) => b + w) + v)`, nil, 63)

	expectRun(t, rta, `out = [1, 2, 3].array()`, nil, ARR{1, 2, 3})
	expectRun(t, rta, `out = [48, 49, -1].bytes()`, nil, rta.NewBytesValue([]byte{48, 49, 255}, false))
	expectRun(t, rta, `out = [48, 49, -1].record()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, rta, `out = [48, 49, -1].dict()`, nil, MAP{"0": 48, "1": 49, "2": -1})
	expectRun(t, rta, `out = [48, 49, 50].string()`, nil, "012")
	expectRun(t, rta, `out = [48, 49, 50].format("v")`, nil, "[48, 49, 50]")
	expectRun(t, rta, `out = [48, 49, 50].format()`, nil, "[48, 49, 50]")

	expectRun(t, rta, `out = 2 in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains(2)`, nil, true)
	expectRun(t, rta, `out = "2" in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains("2")`, nil, true)
	expectRun(t, rta, `out = "z" in [1, 2, 3]`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3].contains("z")`, nil, false)
	expectRun(t, rta, `out = [2, 3] in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains([2, 3])`, nil, true)
	expectRun(t, rta, `out = [] in [1, 2, 3]`, nil, true)
	expectRun(t, rta, `out = [1, 2, 3].contains([])`, nil, true)
	expectRun(t, rta, `out = [1, 3] in [1, 2, 3]`, nil, false)
	expectRun(t, rta, `out = [1, 2, 3].contains([1, 3])`, nil, false)
	expectRun(t, rta, `out = [1, 3] not in [1, 2, 3]`, nil, true)
}

func TestRecord(t *testing.T) {
	expectRun(t, rta, `
out = {
	one: 10 - 9,
	two: 1 + 1,
	three: 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, rta, `
out = {
	"one": 10 - 9,
	"two": 1 + 1,
	"three": 6 / 2
}`, nil, MAP{"one": 1, "two": 2, "three": 3})

	expectRun(t, rta, `out = {foo: 5}["foo"]`, nil, 5)
	expectRun(t, rta, `out = {foo: 5}["bar"]`, nil, core.Undefined)
	expectRun(t, rta, `key := "foo"; out = {foo: 5}[key]`, nil, 5)
	expectRun(t, rta, `out = {}["foo"]`, nil, core.Undefined)

	expectRun(t, rta, `
m := {
	foo: func(x) {
		return x * 2
	}
}
out = m["foo"](2) + m["foo"](3)
`, nil, 10)

	expectRun(t, rta, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1`, nil, 5)
	expectRun(t, rta, `m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1`, nil, 3)
	expectRun(t, rta, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m1.k1 = 5; out = m2.k1 }()`, nil, 5)
	expectRun(t, rta, `func() { m1 := {k1: 1, k2: "foo"}; m2 := m1; m2.k1 = 3; out = m1.k1 }()`, nil, 3)

	v := rta.NewRecordValue(nil, false)
	expectRun(t, rta, fmt.Sprintf(`out = {} == %s`, v.String(rta)), nil, true)
	v = rta.NewRecordValue(nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = {} == %s`, v.String(rta)), nil, true)

	v = rta.NewRecordValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": rta.NewStringValue("3"),
	}, false)
	expectRun(t, rta, fmt.Sprintf(`out = {a: 1, b: undefined, c: "3"} == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = {a: 1, b: 2}["b"]`, nil, 2)
	expectRun(t, rta, `out = {a: 1, b: 2}["q"]`, nil, core.Undefined)
	expectRun(t, rta, `out = {a: 1, b: 2}.b`, nil, 2)
	expectRun(t, rta, `out = {a: 1, b: 2}.q`, nil, core.Undefined)
	expectRun(t, rta, `out = "a" in {a: 1, b: 2}`, nil, true)
	expectRun(t, rta, `out = "q" in {a: 1, b: 2}`, nil, false)
	expectRun(t, rta, `out = "q" not in {a: 1, b: 2}`, nil, true)
	expectRun(t, rta, `t := {a: 1, b: 2}; t["a"] = 3; out = t.a`, nil, 3)
	expectRun(t, rta, `t := {a: 1, b: 2}; t.a = 3; out = t["a"]`, nil, 3)
}

func TestDict(t *testing.T) {
	expectRun(t, rta, fmt.Sprintf(`out = dict() == %s`, rta.NewDictValue(nil, false).String(rta)), nil, true)
	expectRun(t, rta, fmt.Sprintf(`out = dict() == %s`, rta.NewDictValue(nil, true).String(rta)), nil, true)

	expectRun(t, rta, fmt.Sprintf(`out = dict({a: 1, b: undefined, c: "3"}) == %s`, rta.NewDictValue(map[string]core.Value{
		"a": core.IntValue(1),
		"b": core.Undefined,
		"c": rta.NewStringValue("3"),
	}, false).String(rta)), nil, true)

	expectRun(t, rta, `out = dict({a: 1, b: 2})["b"]`, nil, 2)
	expectRun(t, rta, `out = dict({a: 1, b: 2}).record().b`, nil, 2)
	expectRun(t, rta, `out = dict({a: 1, b: 2})["q"]`, nil, core.Undefined)
	expectRun(t, rta, `out = "a" in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, rta, `out = "q" in dict({a: 1, b: 2})`, nil, false)
	expectRun(t, rta, `out = "q" not in dict({a: 1, b: 2})`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2}); t["a"] = 3; out = t["a"]`, nil, 3)
	expectError(t, rta, `dict({a: 1, b: 2}).q`, nil, "Runtime Error: invalid_selector: type dict has no property q\n\tat test:1:20")

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.is_empty()`, nil, false)
	expectRun(t, rta, `t := dict(); out = t.is_empty()`, nil, true)

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.len()`, nil, 2)
	expectRun(t, rta, `t := dict(); out = t.len()`, nil, 0)

	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.keys().sort()`, nil, ARR{"a", "b"})
	expectRun(t, rta, `t := dict({a: 1, b: 2}); out = t.values().sort()`, nil, ARR{1, 2})

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.filter(k => k != "b").keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.filter((k, v) => v > 1).keys().sort()`, nil, ARR{"b", "c"})
	expectRun(t, rta, `t := dict({a: 1, b: undefined, c: 3, d: undefined}); out = t.filter().keys().sort()`, nil, ARR{"a", "c"})
	expectRun(t, rta, `t := dict(); out = t.filter().len()`, nil, 0)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.count(k => k != "b")`, nil, 2)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.count((k, v) => v > 1)`, nil, 2)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "b")`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all(k => k != "q")`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 1)`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.all((k, v) => v > 0)`, nil, true)

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "b")`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any(k => k == "q")`, nil, false)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 1)`, nil, true)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.any((k, v) => v > 10)`, nil, false)

	expectRun(t, rta, `
out = 0
d = dict({a: 1, b: 2, c: 3})
ignored = d.for_each(func(k) {
	out += d[k]
	return true
})
`, nil, 6)

	expectRun(t, rta, `
items = []
ignored = dict({a: 1, b: 2}).for_each(func(k, v) {
	items = append(items, k + v.string())
	return true
})
out = items.sort()
`, nil, ARR{"a1", "b2"})

	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "b")`, nil, "b")
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find(k => k == "q")`, nil, core.Undefined)
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 2)`, nil, "b")
	expectRun(t, rta, `t := dict({a: 1, b: 2, c: 3}); out = t.find((k, v) => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `t := dict(); out = t.find(k => true)`, nil, core.Undefined)
	expectError(t, rta, `dict({a: 1}).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `dict({a: 1}).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `dict({a: 1}).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `out = "a" in dict({a: 1, b: 2, c: 3})`, nil, true)
	expectRun(t, rta, `out = dict({a: 1, b: 2, c: 3}).contains("a")`, nil, true)
	expectRun(t, rta, `out = "q" in dict({a: 1, b: 2, c: 3})`, nil, false)
	expectRun(t, rta, `out = dict({a: 1, b: 2, c: 3}).contains("q")`, nil, false)
	expectRun(t, rta, `out = "q" not in dict({a: 1, b: 2, c: 3})`, nil, true)

	//there is a problem with keys order (it is random) so we cannot test it now
	//expectRun(t, rta, `out = dict({a: 1, b: 2}).format("v")`, nil, `dict({"a": 1, "b": 2})`)
	//expectRun(t, rta, `out = dict({a: 1, b: 2}).format()`, nil, `dict({"a": 1, "b": 2})`)
}

func TestTime(t *testing.T) {
	o := rta.NewTimeValue(time.Date(2020, 6, 20, 1, 2, 3, 4, time.UTC))
	s, _ := o.AsString(rta)
	require.Equal(t, rta, "2020-06-20 01:02:03.000000004 +0000 UTC", s)
	require.Equal(t, rta, `time("2020-06-20T01:02:03.000000004Z")`, o.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = time("2020-06-20 01:02:03.000000004 UTC") == %s`, o.String(rta)), nil, true)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").year()`, nil, 2020)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").month()`, nil, 6)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").day()`, nil, 20)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").hour()`, nil, 1)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").minute()`, nil, 2)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").second()`, nil, 3)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").nanosecond()`, nil, 4)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").unix()`, nil, 1592614923)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").unix_nano()`, nil, 1592614923000000004)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day()`, nil, 6)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").week_day_name()`, nil, "Saturday")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").month_name()`, nil, "June")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 UTC").year_day()`, nil, 172) // June 20 is the 172nd day of the year (173rd in leap years)
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_date()`, nil, "2020-06-20")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_time()`, nil, "01:02:03")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format_datetime()`, nil, "2020-06-20 01:02:03")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").utc().string()`, nil, "2020-06-19 23:02:03.000000004 +0000 UTC")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").zone_offset()`, nil, 7200)

	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").string()`, nil, "2020-06-20 01:02:03.000000004 +0200 +0200")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").int().time().utc().string()`, nil, "2020-06-19 23:02:03 +0000 UTC")

	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format()`, nil, "2020-06-20T01:02:03+02:00")
	expectRun(t, rta, `out = time("2020-06-20 01:02:03.000000004 +0200").format("v")`, nil, `time("2020-06-20T01:02:03.000000004+02:00")`)
}

func TestBytes(t *testing.T) {
	expectRun(t, rta, `out = bytes("Hello World!")`, nil, []byte("Hello World!"))
	expectRun(t, rta, `out = bytes("Hello") + bytes(" ") + bytes("World!")`, nil, []byte("Hello World!"))

	// bytes[] -> byte
	expectRun(t, rta, `out = bytes("abcde")[0]`, nil, byte(97))
	expectRun(t, rta, `out = bytes("abcde")[1]`, nil, byte(98))
	expectRun(t, rta, `out = bytes("abcde")[4]`, nil, byte(101))
	expectRun(t, rta, `out = bytes("abcde")[-1]`, nil, byte(101))
	expectRun(t, rta, `out = bytes("abcde")[-2]`, nil, byte(100))
	expectError(t, rta, `out = bytes("abcde")[-6]`, nil, "index_out_of_bounds")
	expectError(t, rta, `out = bytes("abcde")[10]`, nil, "index_out_of_bounds")

	// bytes[a:b] -> bytes
	expectRun(t, rta, `out = bytes("abcde")[1:4]`, nil, []byte("bcd"))
	expectRun(t, rta, `out = bytes("abcde")[:-1]`, nil, []byte("abcd"))
	expectRun(t, rta, `out = bytes("abcde")[1:-1]`, nil, []byte("bcd"))
	expectRun(t, rta, `out = bytes("abcde")[-2:]`, nil, []byte("de"))
	expectRun(t, rta, `out = bytes("abcde")[-3:-1]`, nil, []byte("cd"))
	expectRun(t, rta, `out = bytes("abcde")[3:1]`, nil, []byte{})
	expectRun(t, rta, `out = bytes("abcde")[10:20]`, nil, []byte{})
	expectRun(t, rta, `out = bytes("abcde")[1:5:2]`, nil, []byte("bd"))
	expectRun(t, rta, `out = bytes("abcde")[1:5:-1]`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("abcde")[4:0:-1]`, nil, []byte("edcb"))
	expectRun(t, rta, `out = bytes("abcde")[0:5:2]`, nil, []byte("ace"))
	expectRun(t, rta, `out = bytes("abcde")[::-1]`, nil, []byte("edcba"))
	expectError(t, rta, `out = bytes("abcde")[::0]`, nil, "step cannot be zero")

	o := rta.NewBytesValue([]byte("Hello World!"), false)
	s, _ := o.AsString(rta)
	require.Equal(t, rta, "Hello World!", s)
	require.Equal(t, rta, "bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33])", o.String(rta))

	expectRun(t, rta, fmt.Sprintf(`out = bytes([72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33]) == %s`, o.String(rta)), nil, true)

	v := rta.NewBytesValue([]byte("hello"), false)
	expectRun(t, rta, fmt.Sprintf(`out = bytes("hello") == %s`, v.String(rta)), nil, true)

	expectRun(t, rta, `out = bytes("abcde").len()`, nil, 5)
	expectRun(t, rta, `out = bytes("abcde").is_empty()`, nil, false)
	expectRun(t, rta, `out = bytes().is_empty()`, nil, true)
	expectRun(t, rta, `out = bytes("abcde").first()`, nil, byte(97))
	expectRun(t, rta, `out = bytes("abcde").last()`, nil, byte(101))

	expectRun(t, rta, `out = bytes("abc").array()`, nil, ARR{97, 98, 99})
	expectRun(t, rta, `out = bytes("abc").record()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, rta, `out = bytes("abc").dict()`, nil, MAP{"0": 97, "1": 98, "2": 99})
	expectRun(t, rta, `out = bytes("abc").string()`, nil, "abc")
	expectRun(t, rta, `out = "abc".bytes().array().string()`, nil, "abc")
	expectRun(t, rta, `out = bytes("abc").format()`, nil, "abc")
	expectRun(t, rta, `out = bytes("abc").format("v")`, nil, "bytes([97, 98, 99])")

	expectRun(t, rta, `out = 98 in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("abc").contains(98)`, nil, true)
	expectRun(t, rta, `out = 255 in bytes("abc")`, nil, false)
	expectRun(t, rta, `out = bytes("abc").contains(255)`, nil, false)
	expectRun(t, rta, `out = bytes("bc") in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("abc").contains(bytes("bc"))`, nil, true)
	expectRun(t, rta, `out = bytes("bd") in bytes("abc")`, nil, false)
	expectRun(t, rta, `out = bytes("abc").contains(bytes("bd"))`, nil, false)
	expectRun(t, rta, `out = bytes("bd") not in bytes("abc")`, nil, true)
	expectRun(t, rta, `out = bytes("hello").sort()`, nil, []byte("ehllo"))
	expectRun(t, rta, `out = bytes("").dedup()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("a").dedup()`, nil, []byte("a"))
	expectRun(t, rta, `out = bytes("aabbccd").dedup()`, nil, []byte("abcd"))
	expectRun(t, rta, `out = bytes("abc").dedup()`, nil, []byte("abc"))
	expectRun(t, rta, `out = bytes("abab").dedup()`, nil, []byte("abab"))
	expectRun(t, rta, `out = bytes("hello").sort().dedup()`, nil, []byte("ehlo"))
	expectRun(t, rta, `out = bytes([1, 1, 2, 2, 3]).dedup()`, nil, []byte{1, 2, 3})
	expectRun(t, rta, `out = bytes("").unique()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("abc").unique()`, nil, []byte("abc"))
	expectRun(t, rta, `out = bytes("hello").unique()`, nil, []byte("helo"))
	expectRun(t, rta, `out = bytes("abab").unique()`, nil, []byte("ab"))
	expectRun(t, rta, `out = bytes([3, 1, 2, 1, 3, 2]).unique()`, nil, []byte{3, 1, 2})
	expectRun(t, rta, `out = bytes("").reverse()`, nil, []byte(""))
	expectRun(t, rta, `out = bytes("hello").reverse()`, nil, []byte("olleh"))
	expectRun(t, rta, `out = bytes([1, 2, 3]).reverse()`, nil, []byte{3, 2, 1})
	expectRun(t, rta, `out = bytes("").chunk(2)`, nil, ARR{})
	expectRun(t, rta, `out = bytes("hello").chunk(2)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, rta, `out = bytes("hello").chunk(2, true)`, nil, ARR{[]byte("he"), []byte("ll"), []byte("o")})
	expectRun(t, rta, `out = bytes("hello").chunk(10)`, nil, ARR{[]byte("hello")})
	expectRun(t, rta, `out = bytes("hello").filter(x => x > 'e')`, nil, []byte("hllo"))
	expectRun(t, rta, `out = bytes("hello").filter((i, x) => i > 2)`, nil, []byte("lo"))
	expectRun(t, rta, `out = bytes("hello").count(x => x > 'e')`, nil, 4)
	expectRun(t, rta, `out = bytes("hello").count((i, x) => i > 2)`, nil, 2)
	expectRun(t, rta, `out = bytes("hello").all(x => x > 'a')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").all(x => x > 'e')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").all((i, x) => i < 5)`, nil, true)
	expectRun(t, rta, `out = bytes("hello").all((i, x) => i < 3)`, nil, false)
	expectRun(t, rta, `out = bytes("hello").any(x => x == 'e')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").any(x => x == 'z')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").any((i, x) => i == 1 && x == 'e')`, nil, true)
	expectRun(t, rta, `out = bytes("hello").any((i, x) => i == 1 && x == 'z')`, nil, false)
	expectRun(t, rta, `out = bytes("hello").find(x => x == 'l')`, nil, 2)
	expectRun(t, rta, `out = bytes("hello").find(x => x == 'z')`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("hello").find((i, x) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = bytes("hello").find((i, x) => i > 100)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("").find(x => true)`, nil, core.Undefined)
	expectError(t, rta, `out = bytes("x").find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = bytes("x").find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = bytes("x").find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")
	expectRun(t, rta, `out = bytes("hello").min()`, nil, byte('e'))
	expectRun(t, rta, `out = bytes("hello").max()`, nil, byte('o'))
	expectRun(t, rta, `
out = 0
ignored := bytes("abc").for_each(func(b) {
	out += b
	return b < 'b'
})
`, nil, 195)
	expectRun(t, rta, `
items := []
ignored := bytes("ABC").for_each(func(i, b) {
	items = append(items, i, b)
	return true
})
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
	expectRun(t, rta, `
items := []
for i, b in bytes("ABC") {
	items = append(items, i, b)
}
out = items
`, nil, ARR{0, byte('A'), 1, byte('B'), 2, byte('C')})
}

func TestBytesMutability(t *testing.T) {
	// index assignment
	expectRun(t, rta, `b := bytes("hello"); b[0] = 'H'; out = b`, nil, []byte("Hello"))
	expectRun(t, rta, `b := bytes("hello"); b[-2] = '!'; out = b`, nil, []byte("hel!o"))
	expectRun(t, rta, `b := bytes("abc"); b[0] = 65; out = b`, nil, []byte("Abc"))

	// append
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c'); out = b2`, nil, []byte("abc"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c', 'd'); out = b2`, nil, []byte("abcd"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, bytes("cd")); out = b2`, nil, []byte("abcd"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 99); out = b2`, nil, []byte("abc"))
	expectRun(t, rta, `b := bytes("ab"); b2 := append(b, 'c'); out = b`, nil, []byte("ab"))

	// sum / avg / map / reduce
	expectRun(t, rta, `out = bytes("abc").sum()`, nil, 97+98+99)
	expectRun(t, rta, `out = bytes("abc").avg()`, nil, (97+98+99)/3)
	expectRun(t, rta, `out = bytes().sum()`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes().avg()`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("abc").map(func(b) { return b + 1 })`, nil, ARR{int64('b'), int64('c'), int64('d')})
	expectRun(t, rta, `out = bytes("abc").map(func(i, b) { return [i, b] })`, nil,
		ARR{ARR{0, byte('a')}, ARR{1, byte('b')}, ARR{2, byte('c')}})
	expectRun(t, rta, `out = bytes("abc").reduce(0, func(acc, b) { return acc + b })`, nil, 97+98+99)
	expectRun(t, rta, `out = bytes("abc").reduce("", func(acc, i, b) { return acc + i.string() + b.string() })`, nil, "097198299")

	// type names
	expectRun(t, rta, `out = type_name(bytes("abc"))`, nil, "bytes")
	expectRun(t, rta, `out = type_name(immutable(bytes("abc")))`, nil, "immutable-bytes")

	// immutable rejects writes
	expectError(t, rta, `b := immutable(bytes("abc")); b[0] = 'X'`, nil, "not_assignable: type immutable-bytes does not support assignment via indexing or field access")

	// slice of immutable stays immutable (shares memory)
	expectRun(t, rta, `out = type_name(immutable(bytes("abcd"))[1:3])`, nil, "immutable-bytes")
	// stepped slice produces a fresh independent buffer, so it is mutable
	expectRun(t, rta, `out = type_name(immutable(bytes("abcd"))[::-1])`, nil, "bytes")
	// slice of mutable stays mutable
	expectRun(t, rta, `out = type_name(bytes("abcd")[1:3])`, nil, "bytes")

	// copy of immutable yields mutable
	expectRun(t, rta, `b := immutable(bytes("abc")); c := copy(b); c[0] = 'X'; out = c`, nil, []byte("Xbc"))

	// append on immutable returns fresh mutable (does not mutate source)
	expectRun(t, rta, `b := immutable(bytes("ab")); b2 := append(b, 'c'); b2[0] = 'X'; out = b2`, nil, []byte("Xbc"))
	expectRun(t, rta, `b := immutable(bytes("ab")); b2 := append(b, 'c'); out = type_name(b2)`, nil, "bytes")

	// invalid assignment values
	expectError(t, rta, `b := bytes("abc"); b[0] = "xy"`, nil,
		"invalid_index_type: (index assign value) expected byte, got string")
	expectError(t, rta, `b := bytes("abc"); b[0] = 256`, nil,
		"invalid_index_type: (index assign value) expected byte, got int")
	expectError(t, rta, `b := bytes("abc"); b[10] = 'X'`, nil,
		"index_out_of_bounds: (index assign) 10 out of range [0, 3]")
}

func TestArrayIterator(t *testing.T) {
	expectRun(t, rta, `
x := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
y := x[2:5]
sum1 := 0
for v in x {
	sum1 += v
}
sum2 := 0
for v in y {
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{55, 12})

	expectRun(t, rta, `
x := [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
y := x[2:5]
isum1 := 0
sum1 := 0
for i, v in x {
	isum1 += i
	sum1 += v
}
isum2 := 0
sum2 := 0
for i, v in y {
	isum2 += i
	sum2 += v
}
out = [isum1, sum1, isum2, sum2]
`, nil, ARR{45, 55, 3, 12})
}

func TestStringIterator(t *testing.T) {
	expectRun(t, rta, `
x := "abcdefg"
y := x[2:5]
res1 := ""
for v in x {
	res1 += v
}
res2 := ""
for v in y {
	res2 += v
}
out = [res1, res2]
`, nil, ARR{"abcdefg", "cde"})

	expectRun(t, rta, `
x := "abcdefg"
y := x[2:5]
isum1 := 0
res1 := ""
for i, v in x {
	isum1 += i
	res1 += v
}
isum2 := 0
res2 := ""
for i, v in y {
	isum2 += i
	res2 += v
}
out = [isum1, res1, isum2, res2]
`, nil, ARR{21, "abcdefg", 3, "cde"})
}

func TestBytesIterator(t *testing.T) {
	expectRun(t, rta, `
x := bytes("abcdefg")
y := x[2:5]
res1 := ""
for v in x {
	res1 += v.rune()
}
res2 := ""
for v in y {
	res2 += v.rune()
}
out = [res1, res2]
`, nil, ARR{"abcdefg", "cde"})

	expectRun(t, rta, `
x := bytes("abcdefg")
y := x[2:5]
isum1 := 0
res1 := ""
for i, v in x {
	isum1 += i
	res1 += v.rune()
}
isum2 := 0
res2 := ""
for i, v in y {
	isum2 += i
	res2 += v.rune()
}
out = [isum1, res1, isum2, res2]
`, nil, ARR{21, "abcdefg", 3, "cde"})
}

func TestRecordIterator(t *testing.T) {
	expectRun(t, rta, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, rta, `
m := {a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10}
sum1 := 0
sum2 := 0
for k, v in m {
	sum1 += k[0] - 'a'
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{45, 55})
}

func TestDictIterator(t *testing.T) {
	expectRun(t, rta, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
for v in m {
	sum1 += v
}
out = sum1
`, nil, 55)

	expectRun(t, rta, `
m := dict({a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10})
sum1 := 0
sum2 := 0
for k, v in m {
	sum1 += k[0] - 'a'
	sum2 += v
}
out = [sum1, sum2]
`, nil, ARR{45, 55})
}

func TestRange(t *testing.T) {
	expectRun(t, rta, `out = range(97, 103, 1).bytes().string()`, nil, "abcdef")
	expectRun(t, rta, `out = range(103, 97, 1).bytes().string()`, nil, "gfedcb")
	expectRun(t, rta, `out = range(97, 103, 1).string()`, nil, "abcdef")
	expectRun(t, rta, `out = range(103, 97, 1).string()`, nil, "gfedcb")
	expectRun(t, rta, `out = range(1, 3, 1).record()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, rta, `out = range(1, 3, 1).dict()`, nil, MAP{"0": 1, "1": 2})
	expectRun(t, rta, `
out = 0
ignored := range(1, 5, 1).for_each(func(v) {
	out += v
	return v < 3
})
`, nil, 6)
	expectRun(t, rta, `
out = 0
ignored := range(10, 13, 1).for_each(func(i, v) {
	out += i + v
	return true
})
`, nil, 36)

	expectRun(t, rta, `out = range(10, 20, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, rta, `out = range(10, 20, 1).find(v => v == 99)`, nil, core.Undefined)
	expectRun(t, rta, `out = range(10, 20, 1).find((i, v) => i == 3)`, nil, 3)
	expectRun(t, rta, `out = range(20, 10, 1).find(v => v == 15)`, nil, 5)
	expectRun(t, rta, `out = range(0, 0, 1).find(v => true)`, nil, core.Undefined)
	expectError(t, rta, `out = range(0, 5, 1).find()`, nil, "wrong_num_arguments: (find) expected 1 argument(s), got 0")
	expectError(t, rta, `out = range(0, 5, 1).find(1)`, nil, "invalid_argument_type: (find) argument first expects type non-variadic function, got int")
	expectError(t, rta, `out = range(0, 5, 1).find(func() { return true })`, nil, "invalid_argument_type: (find) argument first expects type f/1 or f/2")

	expectRun(t, rta, `r := range(0, 10, 1); out = r.len()`, nil, 10)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.len()`, nil, 5)
	expectRun(t, rta, `r := range(0, 10, 3); out = r.len()`, nil, 4)
	expectRun(t, rta, `r := range(0, 10, 4); out = r.len()`, nil, 3)
	expectRun(t, rta, `r := range(0, 10, 5); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 6); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 7); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 8); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 9); out = r.len()`, nil, 2)
	expectRun(t, rta, `r := range(0, 10, 10); out = r.len()`, nil, 1)
	expectRun(t, rta, `r := range(0, 10, 11); out = r.len()`, nil, 1)
	expectRun(t, rta, `r := range(0, 10, 100); out = r.len()`, nil, 1)

	expectRun(t, rta, `r := range(0, 100, 1); out = len(r)`, nil, 100)
	expectRun(t, rta, `r := range(0, 100, 2); out = len(r)`, nil, 50)
	expectRun(t, rta, `r := range(0, 100, 3); out = len(r)`, nil, 34)
	expectRun(t, rta, `r := range(0, 100, 5); out = len(r)`, nil, 20)
	expectRun(t, rta, `r := range(0, 100, 10); out = len(r)`, nil, 10)

	expectRun(t, rta, `r := range(0, 100, 1); out = r.len()`, nil, 100)
	expectRun(t, rta, `r := range(0, 100, 2); out = r.len()`, nil, 50)
	expectRun(t, rta, `r := range(0, 100, 3); out = r.len()`, nil, 34)
	expectRun(t, rta, `r := range(0, 100, 5); out = r.len()`, nil, 20)
	expectRun(t, rta, `r := range(0, 100, 10); out = r.len()`, nil, 10)

	expectRun(t, rta, `r := range(100, 0, 1); out = len(r)`, nil, 100)
	expectRun(t, rta, `r := range(100, 0, 2); out = len(r)`, nil, 50)
	expectRun(t, rta, `r := range(100, 0, 3); out = len(r)`, nil, 34)
	expectRun(t, rta, `r := range(100, 0, 5); out = len(r)`, nil, 20)
	expectRun(t, rta, `r := range(100, 0, 10); out = len(r)`, nil, 10)

	expectRun(t, rta, `r := range(100, 0, 1); out = r.len()`, nil, 100)
	expectRun(t, rta, `r := range(100, 0, 2); out = r.len()`, nil, 50)
	expectRun(t, rta, `r := range(100, 0, 3); out = r.len()`, nil, 34)
	expectRun(t, rta, `r := range(100, 0, 5); out = r.len()`, nil, 20)
	expectRun(t, rta, `r := range(100, 0, 10); out = r.len()`, nil, 10)

	expectRun(t, rta, `r := range(0, 5, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4})
	expectRun(t, rta, `r := range(5, 0, 1); out = r.array()`, nil, ARR{5, 4, 3, 2, 1})
	expectRun(t, rta, `r := range(-5, 5, 1); out = r.array()`, nil, ARR{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4})

	expectRun(t, rta, `r := range(0, 10, 1); out = r.array()`, nil, ARR{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	expectRun(t, rta, `r := range(0, 10, 2); out = r.array()`, nil, ARR{0, 2, 4, 6, 8})
	expectRun(t, rta, `r := range(0, 10, 3); out = r.array()`, nil, ARR{0, 3, 6, 9})
	expectRun(t, rta, `r := range(0, 10, 4); out = r.array()`, nil, ARR{0, 4, 8})
	expectRun(t, rta, `r := range(0, 10, 5); out = r.array()`, nil, ARR{0, 5})

	expectRun(t, rta, `r := range(10, 0, 1); out = r.array()`, nil, ARR{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	expectRun(t, rta, `r := range(10, 0, 2); out = r.array()`, nil, ARR{10, 8, 6, 4, 2})
	expectRun(t, rta, `r := range(10, 0, 3); out = r.array()`, nil, ARR{10, 7, 4, 1})
	expectRun(t, rta, `r := range(10, 0, 4); out = r.array()`, nil, ARR{10, 6, 2})
	expectRun(t, rta, `r := range(10, 0, 5); out = r.array()`, nil, ARR{10, 5})

	expectRun(t, rta, `r := range(0, 100, 1); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[1]`, nil, 1)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[2]`, nil, 2)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[3]`, nil, 3)
	expectRun(t, rta, `r := range(0, 100, 1); out = r[10]`, nil, 10)

	expectRun(t, rta, `r := range(0, 100, 2); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[1]`, nil, 2)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[2]`, nil, 4)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[3]`, nil, 6)
	expectRun(t, rta, `r := range(0, 100, 2); out = r[10]`, nil, 20)

	expectRun(t, rta, `r := range(0, 100, 3); out = r[0]`, nil, 0)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[1]`, nil, 3)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[2]`, nil, 6)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[3]`, nil, 9)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[10]`, nil, 30)
	expectRun(t, rta, `r := range(0, 100, 3); out = r[-1]`, nil, 99)
	expectRun(t, rta, `r := range(10, 0, 2); out = r[-1]`, nil, 2)
	expectError(t, rta, `r := range(0, 100, 3); out = r[-35]`, nil, "index_out_of_bounds")
	expectError(t, rta, `r := range(0, 100, 3); out = r[34]`, nil, "index_out_of_bounds")

	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(0)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(5)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 1); out = r.contains(10)`, nil, false)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(0)`, nil, true)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(1)`, nil, false)
	expectRun(t, rta, `r := range(0, 10, 2); out = r.contains(2)`, nil, true)

	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(0)`, nil, false)
	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(5)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 1); out = r.contains(10)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(10)`, nil, true)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(9)`, nil, false)
	expectRun(t, rta, `r := range(10, 0, 2); out = r.contains(8)`, nil, true)
	expectRun(t, rta, `out = 11 not in range(0, 10, 1)`, nil, true)

	expectRun(t, rta, `
out = 0
for e in range(1, 10, 1) {
	out += e
}
`, nil, 45)

	expectRun(t, rta, `
out = 0
for i, e in range(1, 10, 1) {
	out += i
}
`, nil, 36)

	expectRun(t, rta, `
out = 0
for e in range(1, 10, 2) {
	out += e
}
`, nil, 25)

	expectRun(t, rta, `
out = 0
for i, e in range(1, 10, 2) {
	out += i
}
`, nil, 10)

	expectRun(t, rta, `
r := range(-10, 10, 1)
a := r.array()
s1 := 0
s2 := 0
for i, e in r {
	s1 += r[i] == e
	s2 += a[i] == e
}
out = [s1, s2]
`, nil, ARR{20, 20})

	expectRun(t, rta, `
r := range(10, -10, 1)
a := r.array()
s1 := 0
s2 := 0
for i, e in r {
	s1 += r[i] == e
	s2 += a[i] == e
}
out = [s1, s2]
`, nil, ARR{20, 20})
}

func TestAssignment(t *testing.T) {
	expectRun(t, rta, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, rta, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, rta, `a := 1; a = a + 4; out = a`, nil, 5)
	expectRun(t, rta, `a := 1; f1 := func() { a = 2; return a }; out = f1()`, nil, 2)
	expectRun(t, rta, `a := 1; f1 := func() { a := 3; a = 2; return a }; out = f1()`, nil, 2)

	expectRun(t, rta, `a := 1; out = a`, nil, 1)
	expectRun(t, rta, `a := 1; a = 2; out = a`, nil, 2)
	expectRun(t, rta, `a := 1; func() { a = 2 }(); out = a`, nil, 2)
	expectRun(t, rta, `a := 1; func() { a := 2 }(); out = a`, nil, 1) // "a := 2" defines a new local variable 'a'
	expectRun(t, rta, `a := 1; func() { b := 2; out = b }()`, nil, 2)

	expectRun(t, rta, `
out = func() {
	a := 2
	func() {
		a = 3 // captured from outer scope
	}()
	return a
}()
`, nil, 3)

	expectRun(t, rta, `
func() {
	a := 5
	out = func() {
		a := 4
		return a
	}()
}()`, nil, 4)

	expectError(t, rta, `a := 1; a := 2`, nil, "redeclared")              // redeclared in the same scope
	expectError(t, rta, `func() { a := 1; a := 2 }()`, nil, "redeclared") // redeclared in the same scope

	expectRun(t, rta, `a := 1; a += 2; out = a`, nil, 3)
	expectRun(t, rta, `a := 1; a += 4 - 2;; out = a`, nil, 3)
	expectRun(t, rta, `a := 3; a -= 1;; out = a`, nil, 2)
	expectRun(t, rta, `a := 3; a -= 5 - 4;; out = a`, nil, 2)
	expectRun(t, rta, `a := 2; a *= 4;; out = a`, nil, 8)
	expectRun(t, rta, `a := 2; a *= 1 + 3;; out = a`, nil, 8)
	expectRun(t, rta, `a := 10; a /= 2;; out = a`, nil, 5)
	expectRun(t, rta, `a := 10; a /= 5 - 3;; out = a`, nil, 5)

	// compound assignment operator does not define new variable
	expectError(t, rta, `a += 4`, nil, "unresolved reference")
	expectError(t, rta, `a -= 4`, nil, "unresolved reference")
	expectError(t, rta, `a *= 4`, nil, "unresolved reference")
	expectError(t, rta, `a /= 4`, nil, "unresolved reference")

	expectRun(t, rta, `
f1 := func() {
	f2 := func() {
		a := 1
		a += 2    // it's a statement, not an expression
		return a
	};

	return f2();
};

out = f1();`, nil, 3)

	expectRun(t, rta, `f1 := func() { f2 := func() { a := 1; a += 4 - 2; return a }; return f2(); }; out = f1()`, nil, 3)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 3; a -= 1; return a }; return f2(); }; out = f1()`, nil, 2)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 3; a -= 5 - 4; return a }; return f2(); }; out = f1()`, nil, 2)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 2; a *= 4; return a }; return f2(); }; out = f1()`, nil, 8)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 2; a *= 1 + 3; return a }; return f2(); }; out = f1()`, nil, 8)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 10; a /= 2; return a }; return f2(); }; out = f1()`, nil, 5)
	expectRun(t, rta, `f1 := func() { f2 := func() { a := 10; a /= 5 - 3; return a }; return f2(); }; out = f1()`, nil, 5)

	expectRun(t, rta, `a := 1; f1 := func() { f2 := func() { a += 2; return a }; return f2(); }; out = f1()`, nil, 3)

	expectRun(t, rta, `
	f1 := func(a) {
		return func(b) {
			c := a
			c += b * 2
			return c
		}
	}

	out = f1(3)(4)
	`, nil, 11)

	expectRun(t, rta, `
	out = func() {
		a := 1
		func() {
			a = 2
			func() {
				a = 3
				func() {
					a := 4 // declared new
				}()
			}()
		}()
		return a
	}()
	`, nil, 3)

	// write on free variables
	expectRun(t, rta, `
	f1 := func() {
		a := 5

		return func() {
			a += 3
			return a
		}()
	}
	out = f1()
	`, nil, 8)

	expectRun(t, rta, `
    out = func() {
        f1 := func() {
            a := 5
            add1 := func() { a += 1 }
            add2 := func() { a += 2 }
            a += 3
            return func() { a += 4; add1(); add2(); a += 5; return a }
        }
        return f1()
    }()()
    `, nil, 20)

	expectRun(t, rta, `
		it := func(seq, fn) {
			fn(seq[0])
			fn(seq[1])
			fn(seq[2])
		}

		foo := func(a) {
			b := 0
			it([1, 2, 3], func(x) {
				b = x + a
			})
			return b
		}

		out = foo(2)
		`, nil, 5)

	expectRun(t, rta, `
		it := func(seq, fn) {
			fn(seq[0])
			fn(seq[1])
			fn(seq[2])
		}

		foo := func(a) {
			b := 0
			it([1, 2, 3], func(x) {
				b += x + a
			})
			return b
		}

		out = foo(2)
		`, nil, 12)

	expectRun(t, rta, `
out = func() {
	a := 1
	func() {
		a = 2
	}()
	return a
}()
`, nil, 2)

	expectRun(t, rta, `
f := func() {
	a := 1
	return {
		b: func() { a += 3 },
		c: func() { a += 2 },
		d: func() { return a }
	}
}
m := f()
m.b()
m.c()
out = m.d()
`, nil, 6)

	expectRun(t, rta, `
each := func(s, x) { for i:=0; i<len(s); i++ { x(s[i]) } }

out = func() {
	a := 100
	each([1, 2, 3], func(x) {
		a += x
	})
	a += 10
	return func(b) {
		return a + b
	}
}()(20)
`, nil, 136)

	// assigning different type value
	expectRun(t, rta, `a := 1; a = "foo"; out = a`, nil, "foo")              // global
	expectRun(t, rta, `func() { a := 1; a = "foo"; out = a }()`, nil, "foo") // local

	expectRun(t, rta, `
out = func() {
	a := 5
	return func() {
		a = "foo"
		return a
	}()
}()`, nil, "foo") // free

	// variables declared in if/for blocks
	expectRun(t, rta, `for a:=0; a<5; a++ {}; a := "foo"; out = a`, nil, "foo")
	expectRun(t, rta, `func() { for a:=0; a<5; a++ {}; a := "foo"; out = a }()`, nil, "foo")

	// selectors
	expectRun(t, rta, `a:=[1,2,3]; a[1] = 5; out = a[1]`, nil, 5)
	expectRun(t, rta, `a:=[1,2,3]; a[1] += 5; out = a[1]`, nil, 7)
	expectRun(t, rta, `a:={b:1,c:2}; a.b = 5; out = a.b`, nil, 5)
	expectRun(t, rta, `a:={b:1,c:2}; a.b += 5; out = a.b`, nil, 6)
	expectRun(t, rta, `a:={b:1,c:2}; a.b += a.c; out = a.b`, nil, 3)
	expectRun(t, rta, `a:={b:1,c:2}; a.b += a.c; out = a.c`, nil, 2)

	expectRun(t, rta, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.c.f[1] += 2
out = a["c"]["f"][1]
`, nil, 10)

	expectRun(t, rta, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.c.h = "bar"
out = a.c.h
`, nil, "bar")

	expectError(t, rta, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
a.x.e = "bar"`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
}

func TestFormatting(t *testing.T) {
	// f-string shapes (docs/f-strings.md)
	expectRun(t, rta, `x = 1; y = 2; z = "hello"; out = f"{z}, {x}, {y}"`, nil, "hello, 1, 2")
	expectRun(t, rta, `name = "world"; n = 42; out = f"hello, {name}! n={n:5d}"`, nil, "hello, world! n=   42")
	expectRun(t, rta, `out = f""`, nil, "")
	expectRun(t, rta, `out = f"hello"`, nil, "hello")
	expectRun(t, rta, `x = 10; out = f"{x}"`, nil, "10")
	expectRun(t, rta, `x = 10; out = f"prefix {x}"`, nil, "prefix 10")
	expectRun(t, rta, `x = 10; out = f"{x} suffix"`, nil, "10 suffix")
	expectRun(t, rta, `x = 10; y = 20; out = f"{x}{y}"`, nil, "1020")
	expectRun(t, rta, `x = 1; y = 2; z = 3; out = f"a={x} b={y} c={z}"`, nil, "a=1 b=2 c=3")
	expectRun(t, rta, `a = 1; b = 2; c = 3; out = f"<{a}{b}>{c}"`, nil, "<12>3")

	// escapes inside f-string body (docs/f-strings.md)
	expectRun(t, rta, `p = "/tmp"; out = f"path = \"{p}\""`, nil, `path = "/tmp"`)
	expectRun(t, rta, `out = f"set = {{1, 2, 3}}"`, nil, "set = {1, 2, 3}")
	expectRun(t, rta, `x = 1; out = f"newline -> {x}\n"`, nil, "newline -> 1\n")

	// format specs in f-strings (docs/f-strings.md)
	expectRun(t, rta, `pi = 3.14159; out = f"{pi:.2f}"`, nil, "3.14")
	expectRun(t, rta, `n = 42; out = f"{n:05d}"`, nil, "00042")
	expectRun(t, rta, `x = -42; out = f"{x:05d}"`, nil, "-0042")
	expectRun(t, rta, `n = 1234; out = f"{n:>10,}"`, nil, "     1,234")
	expectRun(t, rta, `x = 255; out = f"{x:06x}"`, nil, "0x00ff")
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#date}"`, nil, "2020-06-20")

	// expressions inside `{...}` (docs/f-strings.md)
	expectRun(t, rta, `x = 1; y = 2; out = f"{x + y}"`, nil, "3")
	expectRun(t, rta, `users = [{name: "alice"}, {name: "bob"}]; i = 1; out = f"{users[i].name}"`, nil, "bob")
	expectRun(t, rta, `out = f"{ dict({a: 1}).values() :v}"`, nil, "[1]")
	expectRun(t, rta, `out = f"{ {a: 1} }"`, nil, `{"a": 1}`)
	expectRun(t, rta, `out = f"{ {a: 1} :v}"`, nil, `{"a": 1}`)
	expectRun(t, rta, `out = f"{[1,2,3]:v}"`, nil, "[1, 2, 3]")
	expectRun(t, rta, `out = f"{[1,2,3]}"`, nil, "[1, 2, 3]")

	// Format Mini-Language: time #-tail templates (docs/format-mini-language.md)
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#%Y-%m-%d %H:%M:%S}"`, nil, "2020-06-20 01:02:03")
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = f"{t:#%Y-%j}"`, nil, "2020-172")
	expectRun(t, rta, `t = time("2020-06-20 13:02:03 +0200"); out = f"{t:#%I:%M %p}"`, nil, "01:02 PM")

	// int / byte verbs
	expectRun(t, rta, `out = (255).format("x")`, nil, "0xff")
	expectRun(t, rta, `out = (255).format("X")`, nil, "0xFF")
	expectRun(t, rta, `out = (42).format("b")`, nil, "0b101010")
	expectRun(t, rta, `out = (42).format("o")`, nil, "0o52")
	expectRun(t, rta, `out = (65).format("c")`, nil, "A")
	expectRun(t, rta, `out = (42).format("d")`, nil, "42")

	// float verbs
	expectRun(t, rta, `out = (1.5).format("e")`, nil, "1.500000e+00")
	expectRun(t, rta, `out = (0.5).format("%")`, nil, "50.000000%")
	expectRun(t, rta, `out = (1.234d).format("s")`, nil, "1.234")

	// bool verbs
	expectRun(t, rta, `out = true.format("t")`, nil, "true")
	expectRun(t, rta, `out = true.format("T")`, nil, "bool")
	expectRun(t, rta, `out = true.format("d")`, nil, "1")
	expectRun(t, rta, `out = false.format("d")`, nil, "0")

	// universal T verb prints the type name
	expectRun(t, rta, `out = (42).format("T")`, nil, "int")
	expectRun(t, rta, `out = (1.5).format("T")`, nil, "float")
	expectRun(t, rta, `out = "abc".format("T")`, nil, "string")
	expectRun(t, rta, `out = 'A'.format("T")`, nil, "rune")

	// rune verbs
	expectRun(t, rta, `out = 'A'.format("d")`, nil, "65")
	expectRun(t, rta, `out = 'A'.format("U")`, nil, "U+0041")
	expectRun(t, rta, `out = 'A'.format("q")`, nil, "'A'")

	// string verbs
	expectRun(t, rta, `out = "abc".format("v")`, nil, `"abc"`)
	expectRun(t, rta, `out = "hello".format("q")`, nil, `"hello"`)
	expectRun(t, rta, `out = "hello".format("b")`, nil, "aGVsbG8=")
	expectRun(t, rta, `out = "hello".format("B")`, nil, "aGVsbG8")
	expectRun(t, rta, `out = "hi".format("x")`, nil, "6869")
	expectRun(t, rta, `out = "a b/c".format("u")`, nil, "a%20b%2Fc")

	// time verbs / aliases
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#date")`, nil, "2020-06-20")
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#time")`, nil, "01:02:03")
	expectRun(t, rta, `t = time("2020-06-20 01:02:03 +0200"); out = t.format("#unix")`, nil, "1592607723")

	// container Kavun-source form via 'v' (docs/format-mini-language.md default-vs-v table)
	expectRun(t, rta, `out = [1, 2, 3].format("v")`, nil, "[1, 2, 3]")

	// --- Edge cases: expressions with conflicting symbols (`:`, `{`, `}`, `?`) with and without fspec ---

	// Slicing uses `:` inside `[]`
	expectRun(t, rta, `a = [1,2,3,4,5]; out = f"{a[1:3]}"`, nil, "[2, 3]")
	expectRun(t, rta, `a = [1,2,3,4,5]; out = f"{a[1:3]:v}"`, nil, "[2, 3]")
	expectRun(t, rta, `a = [1,2,3,4,5]; out = f"{a[::-1]:v}"`, nil, "[5, 4, 3, 2, 1]")
	expectRun(t, rta, `s = "hello"; out = f"{s[1:4]}"`, nil, "ell")
	expectRun(t, rta, `s = "hello"; out = f"{s[1:4]:>6}"`, nil, "   ell")

	// Record literal `{...}` (with internal `:`) directly in expression
	expectRun(t, rta, `out = f"{ {a: 1} }"`, nil, `{"a": 1}`)
	expectRun(t, rta, `out = f"{ {a: 1} :v}"`, nil, `{"a": 1}`)
	expectRun(t, rta, `out = f"{ {a: 1}.a }"`, nil, "1")
	expectRun(t, rta, `out = f"{ {a: 1}.a :>3}"`, nil, "  1")
	expectRun(t, rta, `out = f"{ {a: {b: 1}}.a.b }"`, nil, "1")
	expectRun(t, rta, `out = f"{ {a: {b: 1}}.a.b :05d}"`, nil, "00001")

	// Dict literal expression
	expectRun(t, rta, `out = f"{ dict({a: 1}) :v}"`, nil, `dict({"a": 1})`)
	expectRun(t, rta, `out = f"{ dict({a: 1}).values() }"`, nil, "[1]")

	// Ternary (uses `?` and `:`) — without spec, with spec, nested, chained
	expectRun(t, rta, `cond = true; out = f"{cond ? \"yes\" : \"no\"}"`, nil, "yes")
	expectRun(t, rta, `cond = false; out = f"{cond ? \"yes\" : \"no\"}"`, nil, "no")
	expectRun(t, rta, `cond = true; out = f"{cond ? \"yes\" : \"no\":>5}"`, nil, "  yes")
	expectRun(t, rta, `cond = true; out = f"{cond ? 42 : 7 :>5d}"`, nil, "   42")
	expectRun(t, rta, `cond = false; out = f"{cond ? 42 : 7 :>5d}"`, nil, "    7")
	expectRun(t, rta, `cond = true; out = f"{(cond ? 1 : 2) + 10}"`, nil, "11")
	expectRun(t, rta, `cond = false; out = f"{(cond ? 1 : 2) + 10:>5}"`, nil, "   12")
	expectRun(t, rta, `a = true; b = false; out = f"{a ? (b ? 1 : 2) : 3}"`, nil, "2")
	expectRun(t, rta, `a = true; b = false; out = f"{a ? (b ? 1 : 2) : 3:>5d}"`, nil, "    2")
	expectRun(t, rta, `a = false; b = true; out = f"{a ? 1 : b ? 2 : 3 :>5d}"`, nil, "    2")

	// Strings inside expressions containing `{`, `}`, `:`
	expectRun(t, rta, `s = "{not}"; out = f"prefix {s} suffix"`, nil, "prefix {not} suffix")
	expectRun(t, rta, `s = "a:b"; out = f"{s}"`, nil, "a:b")
	expectRun(t, rta, `s = "a:b"; out = f"{s:>10}"`, nil, "       a:b")
	expectRun(t, rta, `out = f"{\"hi\"}"`, nil, "hi")
	expectRun(t, rta, `out = f"{\"hi\":>5}"`, nil, "   hi")
	expectRun(t, rta, `out = f"{\"a:b\"}"`, nil, "a:b")
	expectRun(t, rta, `out = f"{\"a:b\":>5}"`, nil, "  a:b")

	// Rune literals containing `:`, `{`, `}`
	expectRun(t, rta, `out = f"{':'}"`, nil, ":")
	expectRun(t, rta, `out = f"{'{'}"`, nil, "{")
	expectRun(t, rta, `out = f"{'}'}"`, nil, "}")
	expectRun(t, rta, `out = f"{':':>3}"`, nil, "  :")

	// Multiple interpolations mixing fspec and non-fspec
	expectRun(t, rta, `a = 1; b = 2; out = f"{a} {b:03d} {a + b:>4d}"`, nil, "1 002    3")

	// Function call with embedded string-literal args
	expectRun(t, rta, `out = f"{int(\"42\") + 1}"`, nil, "43")
	expectRun(t, rta, `out = f"{int(\"42\") + 1:>5d}"`, nil, "   43")

	// Literal `{{`/`}}` adjacent to interpolations
	expectRun(t, rta, `x = 5; out = f"{{{x}}}"`, nil, "{5}")
	expectRun(t, rta, `x = 5; out = f"{{{x:03d}}}"`, nil, "{005}")

	// --- Real-world usage patterns ---

	// Log-style messages
	expectRun(t, rta, `id = 42; name = "alice"; out = f"user {name} (id={id}) logged in"`, nil, "user alice (id=42) logged in")
	expectRun(t, rta, `path = "/etc/foo"; err = "permission denied"; out = f"failed to open {path}: {err}"`, nil, "failed to open /etc/foo: permission denied")

	// Tabular alignment
	expectRun(t, rta, `name = "alice"; age = 30; email = "a@x"; out = f"{name:<10} {age:>3} {email}"`, nil, "alice       30 a@x")
	expectRun(t, rta, `out = f"{\"name\":<10}{\"age\":>5}"`, nil, "name        age")
	expectRun(t, rta, `out = f"{\"title\":-^15}"`, nil, "-----title-----")

	// Currency / thousands grouping
	expectRun(t, rta, `amount = 1234567.89; out = f"${amount:,.2f}"`, nil, "$1,234,567.89")
	expectRun(t, rta, `n = 1000000; out = f"{n:,}"`, nil, "1,000,000")
	expectRun(t, rta, `n = 1234567; out = f"{n:_}"`, nil, "1_234_567")

	// Percentage
	expectRun(t, rta, `r = 0.875; out = f"{r:.1%}"`, nil, "87.5%")
	expectRun(t, rta, `r = 0.5; out = f"{r:6.2%}"`, nil, "50.00%")

	// Sign control
	expectRun(t, rta, `x = 42; out = f"{x:+d}"`, nil, "+42")
	expectRun(t, rta, `x = -42; out = f"{x:+d}"`, nil, "-42")
	expectRun(t, rta, `x = 42; out = f"{x: d}"`, nil, " 42")

	// Hex dump style
	expectRun(t, rta, `addr = 255; out = f"{addr:08x}"`, nil, "0x0000ff")
	expectRun(t, rta, `b = 0xab; out = f"{b:02X}"`, nil, "0xAB")

	// Padding identifiers / progress
	expectRun(t, rta, `n = 7; out = f"ID-{n:06d}"`, nil, "ID-000007")
	expectRun(t, rta, `i = 3; total = 100; out = f"[{i:>3}/{total}] processing..."`, nil, "[  3/100] processing...")

	// Building paths and URLs
	expectRun(t, rta, `dir = "/tmp"; name = "foo"; ext = "txt"; out = f"{dir}/{name}.{ext}"`, nil, "/tmp/foo.txt")
	expectRun(t, rta, `host = "example.com"; port = 8080; path = "/api"; out = f"http://{host}:{port}{path}"`, nil, "http://example.com:8080/api")

	// Floating-point precision
	expectRun(t, rta, `pi = 3.14159265358979; out = f"pi = {pi:.4f}"`, nil, "pi = 3.1416")
	expectRun(t, rta, `x = 1234567.89; out = f"{x:.3e}"`, nil, "1.235e+06")
	expectRun(t, rta, `x = 0.00012345; out = f"{x:.2g}"`, nil, "0.00012")

	// Date/time formatting (real-world templates)
	expectRun(t, rta, `ts = time("2026-05-05 18:42:07 +0200"); out = f"[{ts:#%Y-%m-%d %H:%M:%S}] log message"`, nil, "[2026-05-05 18:42:07] log message")
	expectRun(t, rta, `ts = time("2026-05-05 18:42:07 +0200"); out = f"{ts:#%a, %d %b %Y}"`, nil, "Tue, 05 May 2026")
	expectRun(t, rta, `ts = time("2026-05-05 09:42:00 +0200"); out = f"{ts:#%I:%M %p}"`, nil, "09:42 AM")

	// Multi-line via \n inside f-string body
	expectRun(t, rta, `name = "bob"; n = 3; out = f"name: {name}\ncount: {n}"`, nil, "name: bob\ncount: 3")

	// Booleans / mixed types
	expectRun(t, rta, `ok = true; n = 0; out = f"ok={ok} n={n}"`, nil, "ok=true n=0")

	// Method chain (simple)
	expectRun(t, rta, `name = "ALICE"; out = f"hello, {name.lower()}"`, nil, "hello, alice")
	expectRun(t, rta, `s = "  hello  "; out = f"[{s.trim()}]"`, nil, "[hello]")

	// len / common builtins
	expectRun(t, rta, `xs = [1,2,3,4,5]; out = f"got {len(xs)} items"`, nil, "got 5 items")

	// Array rendering inside a sentence
	expectRun(t, rta, `xs = [1, 2, 3]; out = f"items: {xs}"`, nil, "items: [1, 2, 3]")
	expectRun(t, rta, `xs = [1, 2, 3]; out = f"items: {xs:v}"`, nil, "items: [1, 2, 3]")

	// Negative-zero suppression with `~`
	expectRun(t, rta, `x = -0.0001; out = f"{x:.2f}"`, nil, "-0.00")
	expectRun(t, rta, `x = -0.0001; out = f"{x:.2~f}"`, nil, "0.00")

	// Centered text with default fill
	expectRun(t, rta, `s = "ok"; out = f"|{s:^6}|"`, nil, "|  ok  |")

	// Concatenation of multiple f-strings
	expectRun(t, rta, `a = 1; b = 2; out = f"a={a}" + " " + f"b={b}"`, nil, "a=1 b=2")

	// --- Dynamic format specs (Python-style nested `{...}` inside the spec) ---

	// width / precision from variables
	expectRun(t, rta, `v = 3.14159; w = 10; p = 3; out = f"[{v:{w}.{p}f}]"`, nil, "[     3.142]")
	expectRun(t, rta, `v = 3.14159; w = 10; p = 3; out = f"[{v:>{w}.{p}f}]"`, nil, "[     3.142]")

	// fill, align, width all dynamic
	expectRun(t, rta, `n = 42; w = 10; fill = "*"; align = ">"; out = f"[{n:{fill}{align}{w}}]"`, nil, "[********42]")

	// arithmetic in nested spec expression
	expectRun(t, rta, `n = 1; w = 3; out = f"[{n:{w*2}d}]"`, nil, "[     1]")

	// zero-pad via "0" + width
	expectRun(t, rta, `n = 7; w = 4; out = f"[{n:0{w}d}]"`, nil, "[0007]")

	// runtime spec built from a single variable holding the entire spec text
	expectRun(t, rta, `n = 42; spec = "05d"; out = f"[{n:{spec}}]"`, nil, "[00042]")

	// dynamic spec mixed with static specs in the same f-string
	expectRun(t, rta, `x = 1; y = 2; w = 4; out = f"a={x:03d} b={y:{w}d}"`, nil, "a=001 b=   2")

	// dynamic spec where the inner expression returns the empty string -> default formatting
	expectRun(t, rta, `n = 7; s = ""; out = f"[{n:{s}}]"`, nil, "[7]")

	// dynamic-spec fast path is consistent across iterations (cache hit semantics)
	expectRun(t, rta, `w = 5; out = ""; for i in [1, 2, 3] { out += f"[{i:{w}d}]" }`, nil, "[    1][    2][    3]")

	// runtime error when the dynamic spec resolves to invalid fspec text
	expectError(t, rta, `bad = "zzz"; out = f"{1:{bad}}"`, nil, `f-string format spec "zzz"`)
}

func TestFStringDynamicSpecParseErrors(t *testing.T) {
	// Parse-time errors are reported by the parser itself (not by expectError, which uses require.NoError on parse).
	parseErr := func(input, want string) {
		t.Helper()
		fs := parser.NewFileSet()
		f := fs.AddFile("test", -1, len(input))
		p := parser.NewParser(f, []byte(input), nil)
		_, err := p.ParseFile()
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), want), "expected error to contain %q, got: %s", want, err.Error())
	}

	// nested `{` inside a dynamic-spec placeholder is forbidden (only one level of nesting)
	parseErr(`x = f"{1:{{w}}}"`, "fspec")

	// empty placeholder inside a format spec
	parseErr(`x = f"{1:{}}"`, "empty expression in format spec")

	// missing closing `}` inside a format spec
	parseErr(`x = f"{1:{w}"`, "missing")

	// invalid expression inside a dynamic spec
	parseErr(`x = f"{1:{1+}}"`, "f-string")
}

func TestBitwise(t *testing.T) {
	expectRun(t, rta, `out = 1 & 1`, nil, 1)
	expectRun(t, rta, `out = 1 & 0`, nil, 0)
	expectRun(t, rta, `out = 0 & 1`, nil, 0)
	expectRun(t, rta, `out = 0 & 0`, nil, 0)
	expectRun(t, rta, `out = 1 | 1`, nil, 1)
	expectRun(t, rta, `out = 1 | 0`, nil, 1)
	expectRun(t, rta, `out = 0 | 1`, nil, 1)
	expectRun(t, rta, `out = 0 | 0`, nil, 0)
	expectRun(t, rta, `out = 1 ^ 1`, nil, 0)
	expectRun(t, rta, `out = 1 ^ 0`, nil, 1)
	expectRun(t, rta, `out = 0 ^ 1`, nil, 1)
	expectRun(t, rta, `out = 0 ^ 0`, nil, 0)
	expectRun(t, rta, `out = 1 &^ 1`, nil, 0)
	expectRun(t, rta, `out = 1 &^ 0`, nil, 1)
	expectRun(t, rta, `out = 0 &^ 1`, nil, 0)
	expectRun(t, rta, `out = 0 &^ 0`, nil, 0)
	expectRun(t, rta, `out = 1 << 2`, nil, 4)
	expectRun(t, rta, `out = 16 >> 2`, nil, 4)

	expectRun(t, rta, `out = 1; out &= 1`, nil, 1)
	expectRun(t, rta, `out = 1; out |= 0`, nil, 1)
	expectRun(t, rta, `out = 1; out ^= 0`, nil, 1)
	expectRun(t, rta, `out = 1; out &^= 0`, nil, 1)
	expectRun(t, rta, `out = 1; out <<= 2`, nil, 4)
	expectRun(t, rta, `out = 16; out >>= 2`, nil, 4)

	expectRun(t, rta, `out = ^0`, nil, ^0)
	expectRun(t, rta, `out = ^1`, nil, ^1)
	expectRun(t, rta, `out = ^55`, nil, ^55)
	expectRun(t, rta, `out = ^-55`, nil, ^-55)
}

func TestDictRecord(t *testing.T) {
	expectRun(t, rta, `out = len({})`, nil, 0)
	expectRun(t, rta, `out = len(dict())`, nil, 0)
	expectRun(t, rta, `out = len(dict({}))`, nil, 0)

	expectRun(t, rta, `out = len({a: 1})`, nil, 1)
	expectRun(t, rta, `out = len(dict({a: 1}))`, nil, 1)

	expectRun(t, rta, `out = len({a: 1, b: 2})`, nil, 2)
	expectRun(t, rta, `out = len(dict({a: 1, b: 2}))`, nil, 2)

	expectRun(t, rta, `out = dict() == ""`, nil, false)
	expectRun(t, rta, `out = dict() == {}`, nil, true)
	expectRun(t, rta, `out = dict({a: 1}) == {a: 1}`, nil, true)
	expectRun(t, rta, `out = dict({a: 1}) == {a: 1, b: 1}`, nil, false)

	expectRun(t, rta, `out = {a: 1}["a"]`, nil, 1)
	expectRun(t, rta, `out = {a: 1}.a`, nil, 1)

	expectRun(t, rta, `out = dict({a: 1})["a"]`, nil, 1)
}

func TestBuiltinFunctionLen(t *testing.T) {
	expectRun(t, rta, `out = len("")`, nil, 0)
	expectRun(t, rta, `out = len("four")`, nil, 4)
	expectRun(t, rta, `out = len("hello world")`, nil, 11)
	expectRun(t, rta, `out = len([])`, nil, 0)
	expectRun(t, rta, `out = len([1, 2, 3])`, nil, 3)
	expectRun(t, rta, `out = len({})`, nil, 0)
	expectRun(t, rta, `out = len({a:1, b:2})`, nil, 2)
	expectRun(t, rta, `out = len(immutable([]))`, nil, 0)
	expectRun(t, rta, `out = len(immutable([1, 2, 3]))`, nil, 3)
	expectRun(t, rta, `out = len(immutable({}))`, nil, 0)
	expectRun(t, rta, `out = len(immutable({a:1, b:2}))`, nil, 2)
	expectRun(t, rta, `out = len(undefined)`, nil, 0)
	expectRun(t, rta, `out = len(0)`, nil, 1)
	expectRun(t, rta, `out = len(1)`, nil, 1)
	expectError(t, rta, `len("one", "two")`, nil, "wrong_num_arguments")

	// builtins can be reassigned at the top level (smart assignment mode)
	expectRun(t, rta, `len = 10; out = len`, nil, 10)
	expectRun(t, rta, `len := 10; out = len`, nil, 10)
	expectRun(t, rta, `len = func(x) { return 42 }; out = len("hi")`, nil, 42)

	// builtins can be shadowed in function-local scopes; outer scope still sees builtin
	expectRun(t, rta, `f := func() { len := 10; return len }; out = f()`, nil, 10)
	expectRun(t, rta, `f := func() { len := 10; return len }; out = f() + len("hi")`, nil, 12)

	// shadowing in an if-block: outer reference still resolves to builtin
	expectRun(t, rta, `out = 0; if true { len := 10; out = len }`, nil, 10)
	expectRun(t, rta, `if true { len := 10 }; out = len("hi")`, nil, 2)

	// reassignment changes resolution from this point onward; earlier
	// references compiled to OpGetBuiltin keep the builtin semantics
	expectRun(t, rta, `a := len("ab"); len = 99; b := len; out = a + b`, nil, 101)

	// compound assignment to a builtin remains disallowed (no storage)
	expectError(t, rta, `len += 1`, nil, "cannot assign to builtin 'len'")
	expectError(t, rta, `len -= 1`, nil, "cannot assign to builtin 'len'")
}

func TestBuiltinFunctionCopy(t *testing.T) {
	expectRun(t, rta, `out = copy(1)`, nil, 1)
	expectError(t, rta, `copy(1, 2)`, nil, "wrong_num_arguments")
}

func TestBuiltinFunctionAppend(t *testing.T) {
	expectRun(t, rta, `out = append([1, 2, 3], 4)`, nil, ARR{1, 2, 3, 4})
	expectRun(t, rta, `out = append([1, 2, 3], 4, 5, 6)`, nil, ARR{1, 2, 3, 4, 5, 6})
	expectRun(t, rta, `out = append([1, 2, 3], "foo", false)`, nil, ARR{1, 2, 3, "foo", false})
}

func TestBuiltinFunctionInt(t *testing.T) {
	expectRun(t, rta, `out = int(1)`, nil, 1)
	expectRun(t, rta, `out = int(1.8)`, nil, 1)
	expectRun(t, rta, `out = int("-522")`, nil, -522)
	expectRun(t, rta, `out = int(true)`, nil, 1)
	expectRun(t, rta, `out = int(false)`, nil, 0)
	expectRun(t, rta, `out = int('8')`, nil, 56)
	expectRun(t, rta, `out = int([1])`, nil, core.Undefined)
	expectRun(t, rta, `out = int({a: 1})`, nil, core.Undefined)
	expectRun(t, rta, `out = int(time(1))`, nil, 1)
	expectRun(t, rta, `out = int(undefined)`, nil, core.Undefined)
	expectRun(t, rta, `out = int("-522", 1)`, nil, -522)
	expectRun(t, rta, `out = int(undefined, 1)`, nil, 1)
	expectRun(t, rta, `out = int(undefined, 1.8)`, nil, 1.8)
	expectRun(t, rta, `out = int(undefined, string(1))`, nil, "1")
	expectRun(t, rta, `out = int(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionString(t *testing.T) {
	expectRun(t, rta, `out = string(1)`, nil, "1")
	expectRun(t, rta, `out = string(1.8)`, nil, "1.8")
	expectRun(t, rta, `out = string("-522")`, nil, "-522")
	expectRun(t, rta, `out = string(true)`, nil, "true")
	expectRun(t, rta, `out = string(false)`, nil, "false")
	expectRun(t, rta, `out = string('8')`, nil, "8")
	expectRun(t, rta, `out = string([100, 101, 102])`, nil, "def")
	expectRun(t, rta, `out = string({b: "foo"})`, nil, `{"b": "foo"}`)
	expectRun(t, rta, `out = string(undefined)`, nil, core.Undefined) // not "undefined"
	expectRun(t, rta, `out = string(1, "-522")`, nil, "1")
	expectRun(t, rta, `out = string(undefined, "-522")`, nil, "-522") // not "undefined"
}

func TestBuiltinFunctionFloat(t *testing.T) {
	expectRun(t, rta, `out = float(1)`, nil, 1.0)
	expectRun(t, rta, `out = float(1.8)`, nil, 1.8)
	expectRun(t, rta, `out = float("-52.2")`, nil, -52.2)
	expectRun(t, rta, `out = float(true)`, nil, core.Undefined)
	expectRun(t, rta, `out = float(false)`, nil, core.Undefined)
	expectRun(t, rta, `out = float('8')`, nil, core.Undefined)
	expectRun(t, rta, `out = float([1,8.1,true,3])`, nil, core.Undefined)
	expectRun(t, rta, `out = float({a: 1, b: "foo"})`, nil, core.Undefined)
	expectRun(t, rta, `out = float(undefined)`, nil, core.Undefined)
	expectRun(t, rta, `out = float("-52.2", 1.8)`, nil, -52.2)
	expectRun(t, rta, `out = float(undefined, 1)`, nil, 1)
	expectRun(t, rta, `out = float(undefined, 1.8)`, nil, 1.8)
	expectRun(t, rta, `out = float(undefined, "-52.2")`, nil, "-52.2")
	expectRun(t, rta, `out = float(undefined, rune(56))`, nil, '8')
	expectRun(t, rta, `out = float(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionRune(t *testing.T) {
	expectRun(t, rta, `out = rune(56)`, nil, '8')
	expectRun(t, rta, `out = rune(1.8)`, nil, core.Undefined)
	expectRun(t, rta, `out = rune("-52.2")`, nil, core.Undefined)
	expectRun(t, rta, `out = rune(true)`, nil, core.Undefined)
	expectRun(t, rta, `out = rune(false)`, nil, core.Undefined)
	expectRun(t, rta, `out = rune('8')`, nil, '8')
	expectRun(t, rta, `out = rune([1,8.1,true,3])`, nil, core.Undefined)
	expectRun(t, rta, `out = rune({a: 1, b: "foo"})`, nil, core.Undefined)
	expectRun(t, rta, `out = rune(undefined)`, nil, core.Undefined)
	expectRun(t, rta, `out = rune(56, 'a')`, nil, '8')
	expectRun(t, rta, `out = rune(undefined, '8')`, nil, '8')
	expectRun(t, rta, `out = rune(undefined, 56)`, nil, 56)
	expectRun(t, rta, `out = rune(undefined, "-52.2")`, nil, "-52.2")
	expectRun(t, rta, `out = rune(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionBool(t *testing.T) {
	expectRun(t, rta, `out = bool(1)`, nil, true)          // non-zero integer: true
	expectRun(t, rta, `out = bool(0)`, nil, false)         // zero: true
	expectRun(t, rta, `out = bool(1.8)`, nil, true)        // all floats (except for NaN): true
	expectRun(t, rta, `out = bool(0.0)`, nil, true)        // all floats (except for NaN): true
	expectRun(t, rta, `out = bool("false")`, nil, false)   // parsed boolean string: false
	expectRun(t, rta, `out = bool("true")`, nil, true)     // parsed boolean string: true
	expectRun(t, rta, `out = bool("")`, nil, false)        // empty string: false
	expectRun(t, rta, `out = bool(true)`, nil, true)       // true: true
	expectRun(t, rta, `out = bool(false)`, nil, false)     // false: false
	expectRun(t, rta, `out = bool('8')`, nil, true)        // non-zero chars: true
	expectRun(t, rta, `out = bool(rune(0))`, nil, false)   // zero rune: false
	expectRun(t, rta, `out = bool([1])`, nil, true)        // non-empty arrays: true
	expectRun(t, rta, `out = bool([])`, nil, false)        // empty array: false
	expectRun(t, rta, `out = bool({a: 1})`, nil, true)     // non-empty maps: true
	expectRun(t, rta, `out = bool({})`, nil, false)        // empty maps: false
	expectRun(t, rta, `out = bool(undefined)`, nil, false) // undefined: false
}

func TestBuiltinFunctionBytes(t *testing.T) {
	expectRun(t, rta, `out = bytes(1)`, nil, []byte{0})
	expectRun(t, rta, `out = bytes(1.8)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("-522")`, nil, []byte{'-', '5', '2', '2'})
	expectRun(t, rta, `out = bytes(true)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes(false)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes('8')`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes([1])`, nil, rta.NewBytesValue([]byte{1}, false))
	expectRun(t, rta, `out = bytes({a: 1})`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes(undefined)`, nil, core.Undefined)
	expectRun(t, rta, `out = bytes("-522", ['8'])`, nil, []byte{'-', '5', '2', '2'})
	expectRun(t, rta, `out = bytes(undefined, "-522")`, nil, "-522")
	expectRun(t, rta, `out = bytes(undefined, 1)`, nil, 1)
	expectRun(t, rta, `out = bytes(undefined, 1.8)`, nil, 1.8)
	expectRun(t, rta, `out = bytes(undefined, int("-522"))`, nil, -522)
	expectRun(t, rta, `out = bytes(undefined, undefined)`, nil, core.Undefined)
}

func TestBuiltinFunctionIs(t *testing.T) {
	expectRun(t, rta, `out = is_error(error(1))`, nil, true)
	expectRun(t, rta, `out = is_error(1)`, nil, false)

	expectRun(t, rta, `out = is_undefined(undefined)`, nil, true)
	expectRun(t, rta, `out = is_undefined(error(1))`, nil, false)

	// is_function
	expectRun(t, rta, `out = is_function(1)`, nil, false)
	expectRun(t, rta, `out = is_function(func() {})`, nil, true)
	expectRun(t, rta, `out = is_function(func(x) { return x })`, nil, true)
	expectRun(t, rta, `out = is_function(len)`, nil, true)                                               // builtin function
	expectRun(t, rta, `a := func(x) { return func() { return x } }; out = is_function(a)`, nil, true)    // function
	expectRun(t, rta, `a := func(x) { return func() { return x } }; out = is_function(a(5))`, nil, true) // closure

	expectRun(t, rta, `out = is_function(x)`,
		Opts().Symbol("x", NewStringArrayValue([]string{"foo", "bar"})).Skip2ndPass(),
		false) // user object

	// is_callable
	expectRun(t, rta, `out = is_callable(1)`, nil, false)
	expectRun(t, rta, `out = is_callable(func() {})`, nil, true)
	expectRun(t, rta, `out = is_callable(func(x) { return x })`, nil, true)
	expectRun(t, rta, `out = is_callable(len)`, nil, true)                                               // builtin function
	expectRun(t, rta, `a := func(x) { return func() { return x } }; out = is_callable(a)`, nil, true)    // function
	expectRun(t, rta, `a := func(x) { return func() { return x } }; out = is_callable(a(5))`, nil, true) // closure

	expectRun(t, rta, `out = is_callable(x)`,
		Opts().Symbol("x", NewStringArrayValue([]string{"foo", "bar"})).Skip2ndPass(), true) // user object
}

func TestBuiltinFunctionTypeName(t *testing.T) {
	expectRun(t, rta, `out = type_name(1)`, nil, "int")
	expectRun(t, rta, `out = type_name(1.1)`, nil, "float")
	expectRun(t, rta, `out = type_name("a")`, nil, "string")
	expectRun(t, rta, `out = type_name([1,2,3])`, nil, "array")
	expectRun(t, rta, `out = type_name({k:1})`, nil, "record")
	expectRun(t, rta, `out = type_name('a')`, nil, "rune")
	expectRun(t, rta, `out = type_name(true)`, nil, "bool")
	expectRun(t, rta, `out = type_name(false)`, nil, "bool")
	expectRun(t, rta, `out = type_name(bytes( 1))`, nil, "bytes")
	expectRun(t, rta, `out = type_name(undefined)`, nil, "undefined")
	expectRun(t, rta, `out = type_name(error("err"))`, nil, "error")
	expectRun(t, rta, `out = type_name(func() {})`, nil, "<compiled-function/0>")
	expectRun(t, rta, `a := func(x) { return func() { return x } }; out = type_name(a(5))`, nil, "<compiled-function/0>") // closure
}

func TestBuiltinFunctionFormat(t *testing.T) {
	// --- argument validation ---
	expectError(t, rta, `format()`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 0")
	expectError(t, rta, `format("x")`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 1")
	expectError(t, rta, `format("x", [], [])`, nil, "wrong_num_arguments: (format) expected 2 argument(s), got 3")
	expectError(t, rta, `format(1, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got int")
	expectError(t, rta, `format(1.0, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got float")
	expectError(t, rta, `format(undefined, [])`, nil, "invalid_argument_type: (format) argument template expects type string, got undefined")
	expectError(t, rta, `format("x", 1)`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got int")
	expectError(t, rta, `format("x", "y")`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got string")
	expectError(t, rta, `format("x", undefined)`, nil, "invalid_argument_type: (format) argument args expects type array, dict, or record, got undefined")

	// --- pure literal templates (no placeholders) accept any args container ---
	expectRun(t, rta, `out = format("", [])`, nil, "")
	expectRun(t, rta, `out = format("", {})`, nil, "")
	expectRun(t, rta, `out = format("hello", [])`, nil, "hello")
	expectRun(t, rta, `out = format("hello", {})`, nil, "hello")

	// --- {{ and }} brace escapes ---
	expectRun(t, rta, `out = format("a {{ b }} c", [])`, nil, "a { b } c")
	expectRun(t, rta, `out = format("{{}}", [])`, nil, "{}")
	expectRun(t, rta, `out = format("set = {{ {x} }}", {x: 1})`, nil, "set = { 1 }")

	// --- examples from docs/format-function.md ---
	expectRun(t, rta, `out = format("hello {x} from {y}!", {x: "kavun", y: "Kherson"})`, nil, "hello kavun from Kherson!")
	expectRun(t, rta, `out = format("hello {0} from {1}!", ["kavun", "Kherson"])`, nil, "hello kavun from Kherson!")
	expectRun(t, rta, `out = format("pi = {x:.3f}", {x: 3.14159})`, nil, "pi = 3.142")
	expectRun(t, rta, `out = format("n = {x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "n = 00042")
	expectRun(t, rta, `out = format("{x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "00042")
	expectRun(t, rta, `out = format("{0:{1}}", [42, "05d"])`, nil, "00042")

	// --- examples from docs/language.md "Built-in functions" section ---
	expectRun(t, rta, `out = format("hello {x} from {y}!", {x: "kavun", y: "Kherson"})`, nil, "hello kavun from Kherson!")
	expectRun(t, rta, `out = format("hello {0} from {1}!", ["kavun", "Kherson"])`, nil, "hello kavun from Kherson!")
	expectRun(t, rta, `out = format("pi = {x:.3f}", {x: 3.14159})`, nil, "pi = 3.142")
	expectRun(t, rta, `out = format("n = {x:{fmt}}", {x: 42, fmt: "05d"})`, nil, "n = 00042")

	// --- dict and record behave identically for named lookup ---
	expectRun(t, rta, `out = format("hi {x}", dict({x: "world"}))`, nil, "hi world")
	expectRun(t, rta, `out = format("hi {x}", {x: "world"})`, nil, "hi world")

	// --- repeated placeholders, multi-segment templates ---
	expectRun(t, rta, `out = format("{0}-{1}-{0}", ["a", "b"])`, nil, "a-b-a")
	expectRun(t, rta, `out = format("{a}+{b}={a}+{b}", {a: 1, b: 2})`, nil, "1+2=1+2")

	// --- literal fspec variants ---
	expectRun(t, rta, `out = format("{x:>5}", {x: "hi"})`, nil, "   hi")
	expectRun(t, rta, `out = format("{x:*^7}", {x: "hi"})`, nil, "**hi***")

	// --- "Mode is determined by args type" mismatch errors ---
	expectError(t, rta, `format("{x}", [1, 2])`, nil, "invalid_argument_type: (format) argument args expects type dict or record, got array")
	expectError(t, rta, `format("{0}", {a: 1})`, nil, "invalid_argument_type: (format) argument args expects type array, got record")
	expectError(t, rta, `format("{0}", dict({a: 1}))`, nil, "invalid_argument_type: (format) argument args expects type array, got dict")

	// --- "Mixing named and indexed placeholders is an error" ---
	expectError(t, rta, `format("{0} and {x}", [])`, nil, "unsupported_format_spec: format: cannot mix named and indexed placeholders at offset 8")
	expectError(t, rta, `format("{x} and {0}", {})`, nil, "unsupported_format_spec: format: cannot mix named and indexed placeholders at offset 8")

	// --- template syntax errors ---
	expectError(t, rta, `format("a }", [])`, nil, "unsupported_format_spec: format: unmatched '}' at offset 2 (use '}}' for a literal '}')")
	expectError(t, rta, `format("{}", [])`, nil, "unsupported_format_spec: format: empty placeholder '{}' at offset 0 (auto-numbering is not supported)")
	expectError(t, rta, `format("{x", {})`, nil, "unsupported_format_spec: format: unterminated placeholder starting at offset 0")
	expectError(t, rta, `format("{1bad}", {})`, nil, `unsupported_format_spec: format: invalid placeholder "1bad" at offset 0`)
	expectError(t, rta, `format("{x+1}", {})`, nil, `unsupported_format_spec: format: invalid placeholder "x+1" at offset 0`)
	expectError(t, rta, `format("{ x }", {})`, nil, `unsupported_format_spec: format: invalid placeholder " x " at offset 0`)

	// --- spec parse error in literal spec ---
	expectError(t, rta, `format("{x:zzz}", {x: 1})`, nil, `unsupported_format_spec: format: fspec: trailing characters "zz" in "zzz"`)

	// --- nested-{ref} restrictions ---
	expectError(t, rta, `format("{x:>{w}}", {x: 1, w: 5})`, nil, "unsupported_format_spec: format: '{ref}' inside a format spec must stand alone (offset 4)")
	expectError(t, rta, `format("{x:{a}{b}}", {x: 1, a: "0", b: "5d"})`, nil, "unsupported_format_spec: format: '{ref}' inside a format spec must stand alone (offset 6)")
	expectError(t, rta, `format("{x:{}}", {x: 1})`, nil, "unsupported_format_spec: format: empty '{}' inside format spec at offset 3")

	// --- runtime lookup errors ---
	expectError(t, rta, `format("{x}", {})`, nil, `invalid_value: format: missing key "x"`)
	expectError(t, rta, `format("{0}", [])`, nil, "index_out_of_bounds: (format) 0 out of range [0, 0]")
	expectError(t, rta, `format("{2}", ["a", "b"])`, nil, "index_out_of_bounds: (format) 2 out of range [0, 2]")

	// --- spec-by-reference runtime errors ---
	expectError(t, rta, `format("{x:{fmt}}", {x: 1})`, nil, `invalid_value: format: missing spec ref key "fmt"`)
	expectError(t, rta, `format("{0:{1}}", [1])`, nil, "index_out_of_bounds: (format spec ref) 1 out of range [0, 1]")
	expectError(t, rta, `format("{x:{fmt}}", {x: 1, fmt: 2})`, nil, "invalid_argument_type: (format) argument spec ref expects type string, got int")
	expectError(t, rta, `format("{x:{fmt}}", {x: 1, fmt: "zzz"})`, nil, `unsupported_format_spec: format: fspec: trailing characters "zz" in "zzz"`)

	// --- type's Format method rejects an unsupported spec ---
	expectError(t, rta, `format("{x:.2f}", {x: "hi"})`, nil, `unsupported_format_spec: type string does not support format spec {0 0 0 false false 0 0 2 true false false 102 }`)
}

func TestBuiltinFunctionDelete(t *testing.T) {
	expectError(t, rta, `delete()`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 0")
	expectError(t, rta, `delete(1)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 1")
	expectError(t, rta, `delete(1, 2, 3)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 3")
	expectError(t, rta, `delete({}, "", 3)`, nil, "wrong_num_arguments: (delete) expected 2 argument(s), got 3")
	expectError(t, rta, `delete(1, 1)`, nil, `not_deletable: type int does not support delete`)
	expectError(t, rta, `delete(1.0, 1)`, nil, `not_deletable: type float does not support delete`)
	expectError(t, rta, `delete("str", 1)`, nil, `not_deletable: type string does not support delete`)
	expectError(t, rta, `delete(bytes("str"), 1)`, nil, `not_deletable: type bytes does not support delete`)
	expectError(t, rta, `delete(error("err"), 1)`, nil, `not_deletable: type error does not support delete`)
	expectError(t, rta, `delete(true, 1)`, nil, `not_deletable: type bool does not support delete`)
	expectError(t, rta, `delete(rune('c'), 1)`, nil, `not_deletable: type rune does not support delete`)
	expectError(t, rta, `delete(undefined, 1)`, nil, `not_deletable: type undefined does not support delete`)
	expectError(t, rta, `delete(time(1257894000), 1)`, nil, `not_deletable: type time does not support delete`)
	expectError(t, rta, `delete(immutable({}), "key")`, nil, `not_deletable: type immutable-record does not support delete`)
	expectError(t, rta, `delete(immutable([]), "")`, nil, `not_deletable: type immutable-array does not support delete`)
	expectError(t, rta, `delete([], "")`, nil, `not_deletable: type array does not support delete`)
	expectError(t, rta, `delete({}, undefined)`, nil, `invalid_index_type: (delete key) expected string, got undefined`)

	expectRun(t, rta, `out = delete({}, "")`, nil, MAP{})
	expectRun(t, rta, `out = {key1: 1}; delete(out, "key1")`, nil, MAP{})
	expectRun(t, rta, `out = {key1: 1, key2: "2"}; delete(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, rta, `out = dict({key1: 1}); delete(out, "key1")`, nil, MAP{})
	expectRun(t, rta, `out = dict({key1: 1, key2: "2"}); delete(out, "key1")`, nil, MAP{"key2": "2"})
	expectRun(t, rta, `out = [1, "2", {a: "b", c: 10}]; delete(out[2], "c")`, nil, ARR{1, "2", MAP{"a": "b"}})
}

func TestBuiltinFunctionSplice(t *testing.T) {
	expectError(t, rta, `splice()`, nil, "wrong_num_arguments: (splice) expected at least 1 argument(s), got 0")
	expectError(t, rta, `splice(1)`, nil, `invalid_argument_type: (splice) argument first expects type array, got int`)
	expectError(t, rta, `splice(1.0)`, nil, `invalid_argument_type: (splice) argument first expects type array, got float`)
	expectError(t, rta, `splice("str")`, nil, `invalid_argument_type: (splice) argument first expects type array, got string`)
	expectError(t, rta, `splice(bytes("str"))`, nil, `invalid_argument_type: (splice) argument first expects type array, got bytes`)
	expectError(t, rta, `splice(error("err"))`, nil, `invalid_argument_type: (splice) argument first expects type array, got error`)
	expectError(t, rta, `splice(true)`, nil, `invalid_argument_type: (splice) argument first expects type array, got bool`)
	expectError(t, rta, `splice(rune('c'))`, nil, `invalid_argument_type: (splice) argument first expects type array, got rune`)
	expectError(t, rta, `splice(undefined)`, nil, `invalid_argument_type: (splice) argument first expects type array, got undefined`)
	expectError(t, rta, `splice(time(1257894000))`, nil, `invalid_argument_type: (splice) argument first expects type array, got time`)
	expectError(t, rta, `splice(immutable({}))`, nil, `invalid_argument_type: (splice) argument first expects type array, got immutable-record`)
	expectError(t, rta, `splice(immutable([]))`, nil, `invalid_argument_type: (splice) argument first expects type mutable array, got immutable-array`)
	expectError(t, rta, `splice({})`, nil, `invalid_argument_type: (splice) argument first expects type array, got record`)
	expectError(t, rta, `splice([], "str")`, nil, `invalid_argument_type: (splice) argument second expects type int, got string`)
	expectError(t, rta, `splice([], bytes("str"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got bytes`)
	expectError(t, rta, `splice([], error("error"))`, nil, `invalid_argument_type: (splice) argument second expects type int, got error`)
	expectError(t, rta, `splice([], undefined)`, nil, `invalid_argument_type: (splice) argument second expects type int, got undefined`)
	//expectError(t, rta, `splice([], time(0))`, nil, `invalid_argument_type: (splice) argument second expects type int, got time`)
	expectError(t, rta, `splice([], [])`, nil, `invalid_argument_type: (splice) argument second expects type int, got array`)
	expectError(t, rta, `splice([], {})`, nil, `invalid_argument_type: (splice) argument second expects type int, got record`)
	expectError(t, rta, `splice([], immutable([]))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-array`)
	expectError(t, rta, `splice([], immutable({}))`, nil, `invalid_argument_type: (splice) argument second expects type int, got immutable-record`)
	expectError(t, rta, `splice([], 0, "string")`, nil, `invalid_argument_type: (splice) argument third expects type int, got string`)
	expectError(t, rta, `splice([], 0, bytes("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got bytes`)
	expectError(t, rta, `splice([], 0, error("string"))`, nil, `invalid_argument_type: (splice) argument third expects type int, got error`)
	expectError(t, rta, `splice([], 0, undefined)`, nil, `invalid_argument_type: (splice) argument third expects type int, got undefined`)
	//expectError(t, rta, `splice([], 0, time(0))`, nil, `invalid_argument_type: (splice) argument third expects type int, got time`)
	expectError(t, rta, `splice([], 0, [])`, nil, `invalid_argument_type: (splice) argument third expects type int, got array`)
	expectError(t, rta, `splice([], 0, {})`, nil, `invalid_argument_type: (splice) argument third expects type int, got record`)
	expectError(t, rta, `splice([], 0, immutable([]))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-array`)
	expectError(t, rta, `splice([], 0, immutable({}))`, nil, `invalid_argument_type: (splice) argument third expects type int, got immutable-record`)
	expectError(t, rta, `splice([], 1)`, nil, "index_out_of_bounds")
	expectError(t, rta, `splice([1, 2, 3], 0, -1)`, nil, "invalid_value: splice delete count must be non-negative")
	expectError(t, rta, `splice([1, 2, 3], 99, 0, "a", "b")`, nil, "index_out_of_bounds")
	expectRun(t, rta, `out = []; splice(out)`, nil, ARR{})
	expectRun(t, rta, `out = ["a"]; splice(out, 1)`, nil, ARR{"a"})
	expectRun(t, rta, `out = ["a"]; out = splice(out, 1)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 0, 1)`, nil, ARR{2, 3})
	expectRun(t, rta, `out = [1, 2, 3]; out = splice(out, 0, 1)`, nil, ARR{1})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 0, 0, "a", "b")`, nil, ARR{"a", "b", 1, 2, 3})
	expectRun(t, rta, `out = [1, 2, 3]; out = splice(out, 0, 0, "a", "b")`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 1, 0, "a", "b")`, nil, ARR{1, "a", "b", 2, 3})
	expectRun(t, rta, `out = [1, 2, 3]; out = splice(out, 1, 0, "a", "b")`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 1, 0, "a", "b")`, nil, ARR{1, "a", "b", 2, 3})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 2, 0, "a", "b")`, nil, ARR{1, 2, "a", "b", 3})
	expectRun(t, rta, `out = [1, 2, 3]; splice(out, 3, 0, "a", "b")`, nil, ARR{1, 2, 3, "a", "b"})

	expectRun(t, rta, `array := [1, 2, 3]; deleted := splice(array, 1, 1, "a", "b");
				out = [deleted, array]`, nil, ARR{ARR{2}, ARR{1, "a", "b", 3}})

	expectRun(t, rta, `array := [1, 2, 3]; deleted := splice(array, 1);
		out = [deleted, array]`, nil, ARR{ARR{2, 3}, ARR{1}})

	expectRun(t, rta, `out = []; splice(out, 0, 0, "a", "b")`, nil, ARR{"a", "b"})
	expectRun(t, rta, `out = []; splice(out, 0, 1, "a", "b")`, nil, ARR{"a", "b"})
	expectRun(t, rta, `out = []; out = splice(out, 0, 0, "a", "b")`, nil, ARR{})
	expectRun(t, rta, `out = splice(splice([1, 2, 3], 0, 3), 1, 3)`, nil, ARR{2, 3})

	// splice doc examples
	expectRun(t, rta, `v := [1, 2, 3]; deleted := splice(v, 0);
		out = [deleted, v]`, nil, ARR{ARR{1, 2, 3}, ARR{}})

	expectRun(t, rta, `v := [1, 2, 3]; deleted := splice(v, 1);
		out = [deleted, v]`, nil, ARR{ARR{2, 3}, ARR{1}})

	expectRun(t, rta, `v := [1, 2, 3]; deleted := splice(v, 0, 1);
		out = [deleted, v]`, nil, ARR{ARR{1}, ARR{2, 3}})

	expectRun(t, rta, `v := ["a", "b", "c"]; deleted := splice(v, 1, 2);
		out = [deleted, v]`, nil, ARR{ARR{"b", "c"}, ARR{"a"}})

	expectRun(t, rta, `v := ["a", "b", "c"]; deleted := splice(v, 2, 1, "d");
		out = [deleted, v]`, nil, ARR{ARR{"c"}, ARR{"a", "b", "d"}})

	expectRun(t, rta, `v := ["a", "b", "c"]; deleted := splice(v, 0, 0, "d", "e");
		out = [deleted, v]`, nil, ARR{ARR{}, ARR{"d", "e", "a", "b", "c"}})

	expectRun(t, rta, `v := ["a", "b", "c"]; deleted := splice(v, 1, 1, "d", "e");
		out = [deleted, v]`, nil, ARR{ARR{"b"}, ARR{"a", "d", "e", "c"}})
}

func TestBytesN(t *testing.T) {
	expectRun(t, rta, `out = bytes(0)`, nil, make([]byte, 0))
	expectRun(t, rta, `out = bytes(10)`, nil, make([]byte, 10))
	expectRun(t, rta, `out = bytes(1000)`, nil, make([]byte, 1000))
}

func TestCall(t *testing.T) {
	expectRun(t, rta, `a := { b: func(x) { return x + 2 } }; out = a.b(5)`, nil, 7)
	expectRun(t, rta, `a := { b: { c: func(x) { return x + 2 } } }; out = a.b.c(5)`, nil, 7)
	expectRun(t, rta, `a := { b: { c: func(x) { return x + 2 } } }; out = a["b"].c(5)`, nil, 7)
	expectError(t, rta, `a := 1
b := func(a, c) {
   c(a)
}

c := func(a) {
   a()
}
b(a, c)
`, nil, "Runtime Error: not_callable: type int is not callable\n\tat test:7:4\n\tat test:3:4\n\tat test:9:1")
}

func TestCondExpr(t *testing.T) {
	expectRun(t, rta, `out = true ? 5 : 10`, nil, 5)
	expectRun(t, rta, `out = false ? 5 : 10`, nil, 10)
	expectRun(t, rta, `out = (1 == 1) ? 2 + 3 : 12 - 2`, nil, 5)
	expectRun(t, rta, `out = (1 != 1) ? 2 + 3 : 12 - 2`, nil, 10)
	expectRun(t, rta, `out = (1 == 1) ? true ? 10 - 8 : 1 + 3 : 12 - 2`, nil, 2)
	expectRun(t, rta, `out = (1 == 1) ? false ? 10 - 8 : 1 + 3 : 12 - 2`, nil, 4)

	expectRun(t, rta, `
out = 0
f1 := func() { out += 10 }
f2 := func() { out = -out }
true ? f1() : f2()
`, nil, 10)
	expectRun(t, rta, `
out = 5
f1 := func() { out += 10 }
f2 := func() { out = -out }
false ? f1() : f2()
`, nil, -5)
	expectRun(t, rta, `
f1 := func(a) { return a + 2 }
f2 := func(a) { return a - 2 }
f3 := func(a) { return a + 10 }
f4 := func(a) { return -a }

f := func(c) {
	return c == 0 ? f1(c) : f2(c) ? f3(c) : f4(c)
}

out = [f(0), f(1), f(2)]
`, nil, ARR{2, 11, -2})

	expectRun(t, rta, `f := func(a) { return -a }; out = f(true ? 5 : 3)`, nil, -5)
	expectRun(t, rta, `out = [false?5:10, true?1:2]`, nil, ARR{10, 1})

	expectRun(t, rta, `
out = 1 > 2 ?
	1 + 2 + 3 :
	10 - 5`, nil, 5)
}

func TestEquality(t *testing.T) {
	testEquality(t, `1`, `1`, true)
	testEquality(t, `1`, `2`, false)

	testEquality(t, `1.0`, `1.0`, true)
	testEquality(t, `1.0`, `1.1`, false)

	testEquality(t, `true`, `true`, true)
	testEquality(t, `true`, `false`, false)

	testEquality(t, `"foo"`, `"foo"`, true)
	testEquality(t, `"foo"`, `"bar"`, false)

	testEquality(t, `'f'`, `'f'`, true)
	testEquality(t, `'f'`, `'b'`, false)

	testEquality(t, `[]`, `[]`, true)
	testEquality(t, `[1]`, `[1]`, true)
	testEquality(t, `[1]`, `[1, 2]`, false)
	testEquality(t, `["foo", "bar"]`, `["foo", "bar"]`, true)
	testEquality(t, `["foo", "bar"]`, `["bar", "foo"]`, false)

	testEquality(t, `{}`, `{}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2, a: 1}`, true)
	testEquality(t, `{a: 1, b: 2}`, `{b: 2}`, false)
	testEquality(t, `{a: 1, b: {}}`, `{b: {}, a: 1}`, true)

	testEquality(t, `1`, `"foo"`, false)

	expectRun(t, rta, "out = true == true", nil, true)
	expectRun(t, rta, "out = true != false", nil, true)
	expectRun(t, rta, "out = false != true", nil, true)

	expectRun(t, rta, "out = true == 1", nil, true)
	expectRun(t, rta, "out = 1 == true", nil, true)

	expectRun(t, rta, "out = true == 2", nil, true)
	expectRun(t, rta, "out = 2 != true", nil, true)
	expectRun(t, rta, "out = true != 2", nil, false)
	expectRun(t, rta, "out = 2 == true", nil, false)

	expectRun(t, rta, "out = 0 == false", nil, true)
	expectRun(t, rta, "out = 0 != true", nil, true)
	expectRun(t, rta, "out = false == 0", nil, true)
	expectRun(t, rta, "out = true != 0", nil, true)

	expectRun(t, rta, `out = [1] == ["1"]`, nil, true)
	expectRun(t, rta, `out = [1] != ["2"]`, nil, true)

	expectRun(t, rta, `out = [1, [2]] == [1, ["2"]]`, nil, true)
	expectRun(t, rta, `out = [1, [2]] != [1, ["3"]]`, nil, true)

	expectRun(t, rta, `out = {a: 1} == {a: "1"}`, nil, true)
	expectRun(t, rta, `out = {a: 1} != {a: "2"}`, nil, true)

	expectRun(t, rta, `out = {a: 1, b: {c: 2}} == {a: 1, b: {c: "2"}}`, nil, true)
	expectRun(t, rta, `out = {a: 1, b: {c: 2}} != {a: 1, b: {c: "3"}}`, nil, true)
}

func testEquality(t *testing.T, lhs, rhs string, expected bool) {
	// 1. equality is commutative
	// 2. equality and inequality must be always opposite
	expectRun(t, rta, fmt.Sprintf("out = %s == %s", lhs, rhs), nil, expected)
	expectRun(t, rta, fmt.Sprintf("out = %s == %s", rhs, lhs), nil, expected)
	expectRun(t, rta, fmt.Sprintf("out = %s != %s", lhs, rhs), nil, !expected)
	expectRun(t, rta, fmt.Sprintf("out = %s != %s", rhs, lhs), nil, !expected)
}

func TestVMErrorInfo(t *testing.T) {
	expectError(t, rta, `a := 5
a + "boo"`,
		nil, "Runtime Error: invalid_binary_operator: int + string\n\tat test:2:1")

	expectError(t, rta, `a := 5
b := a(5)`,
		nil, "Runtime Error: not_callable: type int is not callable\n\tat test:2:6")

	expectError(t, rta, `a := 5
b := {}
b.x.y = 10`,
		nil, "Runtime Error: not_assignable: type undefined does not support assignment via indexing or field access\n\tat test:3:1")

	expectError(t, rta, `
a := func() {
	b := 5
	b += "foo"
}
a()`,
		nil, "Runtime Error: invalid_binary_operator: int + string\n\tat test:4:2")

	expectError(t, rta, `a := 5
a + import("mod1")`, Opts().Module(
		"mod1", `export "foo"`,
	), ": invalid_binary_operator: int + string\n\tat test:2:1")

	expectError(t, rta, `a := import("mod1")()`,
		Opts().Module(
			"mod1", `
export func() {
	b := 5
	return b + "foo"
}`), "Runtime Error: invalid_binary_operator: int + string\n\tat mod1:4:9")

	expectError(t, rta, `a := import("mod1")()`,
		Opts().Module(
			"mod1", `export import("mod2")()`).
			Module(
				"mod2", `
export func() {
	b := 5
	return b + "foo"
}`), "Runtime Error: invalid_binary_operator: int + string\n\tat mod2:4:9")

	expectError(t, rta, `a := [1, 2, 3]; b := a[:"invalid"];`, nil, "Runtime Error: invalid_index_type: (slice) expected int, got string")

	//expectError(t, rta, `a := immutable([4, 5, 6]); b := a[:false];`, nil, "Runtime Error: invalid slice index type: bool")
	expectRun(t, rta, `a := immutable([4, 5, 6]); out = string(a[:false]);`, nil, "")

	//expectError(t, rta, `a := "hello"; b := a[:1.23];`, nil, "Runtime Error: invalid slice index type: float")
	expectRun(t, rta, `a := "hello"; out = a[:1.23];`, nil, "h")

	//expectError(t, rta, `a := bytes("world"); b := a[:time(1)];`, nil, "Runtime Error: invalid slice index type: time")
	expectRun(t, rta, `a := bytes("world"); out = string(a[:time(1)]);`, nil, "w")
}

func TestVMErrorUnwrap(t *testing.T) {
	userErr := errors.New("user runtime error")

	userFunc := func(err error) core.Value {
		return core.NewBuiltinClosureValue(
			"user_func",
			func(_ *core.Arena, v core.VM, args []core.Value) (core.Value, error) {
				return core.Undefined, err
			},
			0,
			false,
		)
	}

	expectError(t, rta, `user_func()`, Opts().Symbol("user_func", userFunc(userErr)), "Runtime Error: "+userErr.Error())
	expectErrorIs(t, rta, `user_func()`, Opts().Symbol("user_func", userFunc(userErr)), userErr)

	wrapUserErr := &customError{err: userErr, str: "custom error"}
	expectErrorIs(t, rta, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), wrapUserErr)
	expectErrorIs(t, rta, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), userErr)

	var asErr1 *customError
	expectErrorAs(t, rta, `user_func()`, Opts().Symbol("user_func", userFunc(wrapUserErr)), &asErr1)
	require.True(t, asErr1.Error() == wrapUserErr.Error(), "expected error as:%v, got:%v", wrapUserErr, asErr1)

	userModule := func(err error) module {
		return module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"afunction",
					func(_ *core.Arena, v core.VM, a []core.Value) (core.Value, error) {
						return core.Undefined, err
					},
					0,
					false,
				),
			},
		}
	}

	expectError(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(userErr)), "Runtime Error: "+userErr.Error())
	expectErrorIs(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(userErr)), userErr)
	expectError(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), "Runtime Error: "+wrapUserErr.Error())
	expectErrorIs(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), wrapUserErr)
	expectErrorIs(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), userErr)

	var asErr2 *customError
	expectErrorAs(t, rta, `import("mod1").afunction()`, Opts().BuiltinModule("mod1", userModule(wrapUserErr)), &asErr2)
	require.True(t, asErr2.Error() == wrapUserErr.Error(), "expected error as:%v, got:%v", wrapUserErr, asErr2)
}

func TestForIn(t *testing.T) {
	// array
	expectRun(t, rta, `out = 0; for x in [1, 2, 3] { out += x }`, nil, 6)                     // value
	expectRun(t, rta, `out = 0; for i, x in [1, 2, 3] { out += i + x }`, nil, 9)              // index, value
	expectRun(t, rta, `out = 0; func() { for i, x in [1, 2, 3] { out += i + x } }()`, nil, 9) // index, value
	expectRun(t, rta, `out = 0; for i, _ in [1, 2, 3] { out += i }`, nil, 3)                  // index, _
	expectRun(t, rta, `out = 0; func() { for i, _ in [1, 2, 3] { out += i  } }()`, nil, 3)    // index, _

	// record
	expectRun(t, rta, `out = 0; for v in {a:2,b:3,c:4} { out += v }`, nil, 9)                                      // value
	expectRun(t, rta, `out = ""; for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } }`, nil, "b")              // key, value
	expectRun(t, rta, `out = ""; for k, _ in {a:2} { out += k }`, nil, "a")                                        // key, _
	expectRun(t, rta, `out = 0; for _, v in {a:2,b:3,c:4} { out += v }`, nil, 9)                                   // _, value
	expectRun(t, rta, `out = ""; func() { for k, v in {a:2,b:3,c:4} { out = k; if v==3 { break } } }()`, nil, "b") // key, value

	// string
	expectRun(t, rta, `out = ""; for c in "abcde" { out += c }`, nil, "abcde")
	expectRun(t, rta, `out = ""; for i, c in "abcde" { if i == 2 { continue }; out += c }`, nil, "abde")
}

func TestFor(t *testing.T) {
	expectRun(t, rta, `
	out = 0
	for {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, rta, `
	out = 0
	for {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, rta, `
	out = 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}`, nil, 7) // 1 + 2 + 4

	expectRun(t, rta, `
	out = 0
	a := 0
	for {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, rta, `
	out = 0
	for true {
		out++
		if out == 5 {
			break
		}
	}`, nil, 5)

	expectRun(t, rta, `
	a := 0
	for true {
		a++
		if a == 5 {
			break
		}
	}
	out = a`, nil, 5)

	expectRun(t, rta, `
	out = 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		if a == 5 { break }
		out += a
	}`, nil, 7) // 1 + 2 + 4

	expectRun(t, rta, `
	out = 0
	a := 0
	for true {
		a++
		if a == 3 { continue }
		out += a
		if a == 5 { break }
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, rta, `
	out = 0
	func() {
		for true {
			out++
			if out == 5 {
				return
			}
		}
	}()`, nil, 5)

	expectRun(t, rta, `
	out = 0
	for a:=1; a<=10; a++ {
		out += a
	}`, nil, 55)

	expectRun(t, rta, `
	out = 0
	for a:=1; a<=3; a++ {
		for b:=3; b<=6; b++ {
			out += b
		}
	}`, nil, 54)

	expectRun(t, rta, `
	out = 0
	func() {
		for {
			out++
			if out == 5 {
				break
			}
		}
	}()`, nil, 5)

	expectRun(t, rta, `
	out = 0
	func() {
		for true {
			out++
			if out == 5 {
				break
			}
		}
	}()`, nil, 5)

	expectRun(t, rta, `
	out = func() {
		a := 0
		for {
			a++
			if a == 5 {
				break
			}
		}
		return a
	}()`, nil, 5)

	expectRun(t, rta, `
	out = func() {
		a := 0
		for true {
			a++
			if a== 5 {
				break
			}
		}
		return a
	}()`, nil, 5)

	expectRun(t, rta, `
	out = func() {
		a := 0
		func() {
			for {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, 5)

	expectRun(t, rta, `
	out = func() {
		a := 0
		func() {
			for true {
				a++
				if a == 5 {
					break
				}
			}
		}()
		return a
	}()`, nil, 5)

	expectRun(t, rta, `
	out = func() {
		sum := 0
		for a:=1; a<=10; a++ {
			sum += a
		}
		return sum
	}()`, nil, 55)

	expectRun(t, rta, `
	out = func() {
		sum := 0
		for a:=1; a<=4; a++ {
			for b:=3; b<=5; b++ {
				sum += b
			}
		}
		return sum
	}()`, nil, 48) // (3+4+5) * 4

	expectRun(t, rta, `
	a := 1
	for ; a<=10; a++ {
		if a == 5 {
			break
		}
	}
	out = a`, nil, 5)

	expectRun(t, rta, `
	out = 0
	for a:=1; a<=10; a++ {
		if a == 3 {
			continue
		}
		out += a
		if a == 5 {
			break
		}
	}`, nil, 12) // 1 + 2 + 4 + 5

	expectRun(t, rta, `
	out = 0
	for a:=1; a<=10; {
		if a == 3 {
			a++
			continue
		}
		out += a
		if a == 5 {
			break
		}
		a++
	}`, nil, 12) // 1 + 2 + 4 + 5
}

func TestFunction(t *testing.T) {
	// function with no "return" statement returns "invalid" value.
	expectRun(t, rta, `f1 := func() {}; out = f1();`,
		nil, core.Undefined)
	expectRun(t, rta, `f1 := func() {}; f2 := func() { return f1(); }; f1(); out = f2();`,
		nil, core.Undefined)
	expectRun(t, rta, `f := func(x) { x; }; out = f(5);`,
		nil, core.Undefined)

	expectRun(t, rta, `f := func(...x) { return x; }; out = f(1,2,3);`,
		nil, IARR{1, 2, 3})

	expectRun(t, rta, `f := func(a, b, ...x) { return [a, b, x]; }; out = f(8,9,1,2,3);`,
		nil, ARR{8, 9, ARR{1, 2, 3}})

	expectRun(t, rta, `f := func(v) { x := 2; return func(a, ...b){ return [a, b, v+x]}; }; out = f(5)("a", "b");`,
		nil, ARR{"a", ARR{"b"}, 7})

	expectRun(t, rta, `f := func(...x) { return x; }; out = f();`,
		nil, rta.NewArrayValue([]core.Value{}, true))

	expectRun(t, rta, `f := func(a, b, ...x) { return [a, b, x]; }; out = f(8, 9);`,
		nil, ARR{8, 9, ARR{}})

	expectRun(t, rta, `f := func(v) { x := 2; return func(a, ...b){ return [a, b, v+x]}; }; out = f(5)("a");`,
		nil, ARR{"a", ARR{}, 7})

	expectError(t, rta, `f := func(a, b, ...x) { return [a, b, x]; }; f();`, nil,
		"Runtime Error: wrong_num_arguments: (call) expected >=2 argument(s), got 0\n\tat test:1:46")

	expectError(t, rta, `f := func(a, b, ...x) { return [a, b, x]; }; f(1);`, nil,
		"Runtime Error: wrong_num_arguments: (call) expected >=2 argument(s), got 1\n\tat test:1:46")

	expectRun(t, rta, `f := func(x) { return x; }; out = f(5);`, nil, 5)
	expectRun(t, rta, `f := func(x) { return x * 2; }; out = f(5);`, nil, 10)
	expectRun(t, rta, `f := func(x, y) { return x + y; }; out = f(5, 5);`, nil, 10)
	expectRun(t, rta, `f := func(x, y) { return x + y; }; out = f(5 + 5, f(5, 5));`, nil, 20)
	expectRun(t, rta, `out = func(x) { return x; }(5)`, nil, 5)
	expectRun(t, rta, `x := 10; f := func(x) { return x; }; f(5); out = x;`, nil, 10)

	expectRun(t, rta, `
	f2 := func(a) {
		f1 := func(a) {
			return a * 2;
		};

		return f1(a) * 3;
	};

	out = f2(10);
	`, nil, 60)

	expectRun(t, rta, `
		f1 := func(f) {
			a := [undefined]
			a[0] = func() { return f(a) }
			return a[0]()
		}

		out = f1(func(a) { return 2 })
	`, nil, 2)

	// closures
	expectRun(t, rta, `
		newAdder := func(x) {
			return func(y) { return x + y };
		};

		add2 := newAdder(2);
		out = add2(5);
		`, nil, 7)
	expectRun(t, rta, `
		m := {a: 1}
		for k,v in m {
			func(){
				out = k
			}()
		}
		`, nil, "a")

	expectRun(t, rta, `
		m := {a: 1}
		for k,v in m {
			func(){
				out = v
			}()
		}
		`, nil, 1)
	// function as a argument
	expectRun(t, rta, `
	add := func(a, b) { return a + b };
	sub := func(a, b) { return a - b };
	applyFunc := func(a, b, f) { return f(a, b) };

	out = applyFunc(applyFunc(2, 2, add), 3, sub);
	`, nil, 1)

	expectRun(t, rta, `f1 := func() { return 5 + 10; }; out = f1();`,
		nil, 15)
	expectRun(t, rta, `f1 := func() { return 1 }; f2 := func() { return 2 }; out = f1() + f2()`,
		nil, 3)
	expectRun(t, rta, `f1 := func() { return 1 }; f2 := func() { return f1() + 2 }; f3 := func() { return f2() + 3 }; out = f3()`,
		nil, 6)
	expectRun(t, rta, `f1 := func() { return 99; 100 }; out = f1();`,
		nil, 99)
	expectRun(t, rta, `f1 := func() { return 99; return 100 }; out = f1();`,
		nil, 99)
	expectRun(t, rta, `f1 := func() { return 33; }; f2 := func() { return f1 }; out = f2()();`,
		nil, 33)
	expectRun(t, rta, `one := func() { one = 1; return one }; out = one()`,
		nil, 1)
	expectRun(t, rta, `three := func() { one := 1; two := 2; return one + two }; out = three()`,
		nil, 3)
	expectRun(t, rta, `three := func() { one := 1; two := 2; return one + two }; seven := func() { three := 3; four := 4; return three + four }; out = three() + seven()`,
		nil, 10)
	expectRun(t, rta, `
	foo1 := func() {
		foo := 50
		return foo
	}
	foo2 := func() {
		foo := 100
		return foo
	}
	out = foo1() + foo2()`, nil, 150)
	expectRun(t, rta, `
	g := 50;
	minusOne := func() {
		n := 1;
		return g - n;
	};
	minusTwo := func() {
		n := 2;
		return g - n;
	};
	out = minusOne() + minusTwo()
	`, nil, 97)
	expectRun(t, rta, `
	f1 := func() {
		f2 := func() { return 1; }
		return f2
	};
	out = f1()()
	`, nil, 1)

	expectRun(t, rta, `
	f1 := func(a) { return a; };
	out = f1(4)`, nil, 4)
	expectRun(t, rta, `
	f1 := func(a, b) { return a + b; };
	out = f1(1, 2)`, nil, 3)

	expectRun(t, rta, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	out = sum(1, 2);`, nil, 3)

	expectRun(t, rta, `
	sum := func(a, b) {
		c := a + b;
		return c;
	};
	out = sum(1, 2) + sum(3, 4);`, nil, 10)

	expectRun(t, rta, `
	sum := func(a, b) {
		c := a + b
		return c
	};
	outer := func() {
		return sum(1, 2) + sum(3, 4)
	};
	out = outer();`, nil, 10)

	expectRun(t, rta, `
	g := 10;

	sum := func(a, b) {
		c := a + b;
		return c + g;
	}

	outer := func() {
		return sum(1, 2) + sum(3, 4) + g;
	}

	out = outer() + g
	`, nil, 50)

	expectError(t, rta, `func() { return 1; }(1)`,
		nil, "wrong_num_arguments")
	expectError(t, rta, `func(a) { return a; }()`,
		nil, "wrong_num_arguments")
	expectError(t, rta, `func(a, b) { return a + b; }(1)`,
		nil, "wrong_num_arguments")

	expectRun(t, rta, `
		f1 := func(a) {
			return func() { return a; };
		};
		f2 := f1(99);
		out = f2()
		`, nil, 99)

	expectRun(t, rta, `
		f1 := func(a, b) {
			return func(c) { return a + b + c };
		};

		f2 := f1(1, 2);
		out = f2(8);
		`, nil, 11)
	expectRun(t, rta, `
		f1 := func(a, b) {
			c := a + b;
			return func(d) { return c + d };
		};
		f2 := f1(1, 2);
		out = f2(8);
		`, nil, 11)
	expectRun(t, rta, `
		f1 := func(a, b) {
			c := a + b;
			return func(d) {
				e := d + c;
				return func(f) { return e + f };
			}
		};
		f2 := f1(1, 2);
		f3 := f2(3);
		out = f3(8);
		`, nil, 14)
	expectRun(t, rta, `
		a := 1;
		f1 := func(b) {
			return func(c) {
				return func(d) { return a + b + c + d }
			};
		};
		f2 := f1(2);
		f3 := f2(3);
		out = f3(8);
		`, nil, 14)
	expectRun(t, rta, `
		f1 := func(a, b) {
			one := func() { return a; };
			two := func() { return b; };
			return func() { return one() + two(); }
		};
		f2 := f1(9, 90);
		out = f2();
		`, nil, 99)

	// global function recursion
	expectRun(t, rta, `
		fib := func(x) {
			if x == 0 {
				return 0
			} else if x == 1 {
				return 1
			} else {
				return fib(x-1) + fib(x-2)
			}
		}
		out = fib(15)`, nil, 610)

	// local function recursion
	expectRun(t, rta, `
out = func() {
	sum := func(x) {
		return x == 0 ? 0 : x + sum(x-1)
	}
	return sum(5)
}()`, nil, 15)

	expectError(t, rta, `return 5`, nil, "return not allowed outside function")

	// closure and block scopes
	expectRun(t, rta, `
func() {
	a := 10
	func() {
		b := 5
		if true {
			out = a + 5
		}
	}()
}()`, nil, 15)
	expectRun(t, rta, `
func() {
	a := 10
	b := func() { return 5 }
	func() {
		if b() {
			out = a + b()
		}
	}()
}()`, nil, 15)
	expectRun(t, rta, `
func() {
	a := 10
	func() {
		b := func() { return 5 }
		func() {
			if true {
				out = a + b()
			}
		}()
	}()
}()`, nil, 15)

	// function skipping return
	expectRun(t, rta, `out = func() {}()`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { if v { return true } }(1)`,
		nil, true)
	expectRun(t, rta, `out = func(v) { if v { return true } }(0)`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { if v { } else { return true } }(1)`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { if v { return } }(1)`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { if v { return } }(0)`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { if v { } else { return } }(1)`,
		nil, core.Undefined)
	expectRun(t, rta, `out = func(v) { for ;;v++ { if v == 3 { return true } } }(1)`,
		nil, true)
	expectRun(t, rta, `out = func(v) { for ;;v++ { if v == 3 { break } } }(1)`,
		nil, core.Undefined)

	// 'f' in RHS at line 4 must reference global variable 'f'
	expectRun(t, rta, `
f := func() { return 2 }
out = (func() {
	f := f()
	return f
})()
	`, nil, 2)
}

func TestBlocksInGlobalScope(t *testing.T) {
	expectRun(t, rta, `
f := undefined
if true {
	a := 1
	f = func() {
		a = 2
	}
}
b := 3
f()
out = b`,
		nil, 3)

	expectRun(t, rta, `
func() {
	f := undefined
	if true {
		a := 10
		f = func() {
			a = 20
		}
	}
	b := 5
	f()
	out = b
}()
	`,
		nil, 5)

	expectRun(t, rta, `
f := undefined
if true {
	a := 1
	b := 2
	f = func() {
		a = 3
		b = 4
	}
}
c := 5
d := 6
f()
out = c + d`,
		nil, 11)

	expectRun(t, rta, `
fn := undefined
if true {
	a := 1
	b := 2
	if true {
		c := 3
		d := 4
		fn = func() {
			a = 5
			b = 6
			c = 7
			d = 8
		}
	}
}
e := 9
f := 10
fn()
out = e + f`,
		nil, 19)

	expectRun(t, rta, `
out = 0
func() {
	for x in [1, 2, 3] {
		out += x
	}
}()`,
		nil, 6)

	expectRun(t, rta, `
out = 0
for x in [1, 2, 3] {
	out += x
}`,
		nil, 6)
}

func TestIf(t *testing.T) {

	expectRun(t, rta, `if (true) { out = 10 }`, nil, 10)
	expectRun(t, rta, `if (false) { out = 10 }`, nil, core.Undefined)
	expectRun(t, rta, `if (false) { out = 10 } else { out = 20 }`, nil, 20)
	expectRun(t, rta, `if (1) { out = 10 }`, nil, 10)
	expectRun(t, rta, `if (0) { out = 10 } else { out = 20 }`, nil, 20)
	expectRun(t, rta, `if (1 < 2) { out = 10 }`, nil, 10)
	expectRun(t, rta, `if (1 > 2) { out = 10 }`, nil, core.Undefined)
	expectRun(t, rta, `if (1 < 2) { out = 10 } else { out = 20 }`, nil, 10)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else { out = 20 }`, nil, 20)

	expectRun(t, rta, `if (1 < 2) { out = 10 } else if (1 > 2) { out = 20 } else { out = 30 }`,
		nil, 10)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 < 2) { out = 20 } else { out = 30 }`,
		nil, 20)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30 }`,
		nil, 30)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else if (1 < 2) { out = 30 } else { out = 40 }`,
		nil, 30)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 < 2) { out = 20; out = 21; out = 22 } else { out = 30 }`,
		nil, 22)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { out = 30; out = 31; out = 32}`,
		nil, 32)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else { out = 22 } } else { out = 30 }`,
		nil, 22)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 < 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }`,
		nil, 23)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 == 2) { if (1 == 2) { out = 21 } else if (2 == 3) { out = 22 } else { out = 23 } } else { out = 30 }`,
		nil, 30)
	expectRun(t, rta, `if (1 > 2) { out = 10 } else if (1 == 2) { out = 20 } else { if (1 == 2) { out = 31 } else if (2 == 3) { out = 32 } else { out = 33 } }`,
		nil, 33)

	expectRun(t, rta, `if a:=0; a<1 { out = 10 }`, nil, 10)
	expectRun(t, rta, `a:=0; if a++; a==1 { out = 10 }`, nil, 10)
	expectRun(t, rta, `
func() {
	a := 1
	if a++; a > 1 {
		out = a
	}
}()
`, nil, 2)
	expectRun(t, rta, `
func() {
	a := 1
	if a++; a == 1 {
		out = 10
	} else {
		out = 20
	}
}()
`, nil, 20)
	expectRun(t, rta, `
func() {
	a := 1

	func() {
		if a++; a > 1 {
			a++
		}
	}()

	out = a
}()
`, nil, 3)

	// expression statement in init (should not leave objects on stack)
	expectRun(t, rta, `a := 1; if a; a { out = a }`, nil, 1)
	expectRun(t, rta, `a := 1; if a + 4; a { out = a }`, nil, 1)

	// dead code elimination
	expectRun(t, rta, `
out = func() {
	if false { return 1 }

	a := undefined

	a = 2
	if !a {
		b := func() {
			return is_callable(a) ? a(8) : a
		}()
		if is_error(b) {
			return b
		} else if !is_undefined(b) {
			return immutable(b)
		}
	}

	a = 3
	if a {
		b := func() {
			return is_callable(a) ? a(9) : a
		}()
		if is_error(b) {
			return b
		} else if !is_undefined(b) {
			return immutable(b)
		}
	}

	return a
}()
`, nil, 3)
}

func TestImmutable(t *testing.T) {
	// primitive types are already immutable values
	// immutable expression has no effects.
	expectRun(t, rta, `a := immutable(1); out = a`, nil, 1)
	expectRun(t, rta, `a := 5; b := immutable(a); out = b`, nil, 5)
	expectRun(t, rta, `a := immutable(1); a = 5; out = a`, nil, 5)

	// array
	expectError(t, rta, `a := immutable([1, 2, 3]); a[1] = 5`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, rta, `a := immutable(["foo", [1,2,3]]); a[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, rta, `a := immutable(["foo", [1,2,3]]); a[1][1] = "bar"; out = a`, nil, IARR{"foo", ARR{1, "bar", 3}})
	expectError(t, rta, `a := immutable(["foo", immutable([1,2,3])]); a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, rta, `a := ["foo", immutable([1,2,3])]; a[1][1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, rta, `a := immutable([1,2,3]); b := copy(a); b[1] = 5; out = b`, nil, ARR{1, 5, 3})
	expectRun(t, rta, `a := immutable([1,2,3]); b := copy(a); b[1] = 5; out = a`, nil, IARR{1, 2, 3})
	expectRun(t, rta, `out = immutable([1,2,3]) == [1,2,3]`, nil, true)
	expectRun(t, rta, `out = immutable([1,2,3]) == immutable([1,2,3])`, nil, true)
	expectRun(t, rta, `out = [1,2,3] == immutable([1,2,3])`, nil, true)
	expectRun(t, rta, `out = immutable([1,2,3]) == [1,2]`, nil, false)
	expectRun(t, rta, `out = immutable([1,2,3]) == immutable([1,2])`, nil, false)
	expectRun(t, rta, `out = [1,2,3] == immutable([1,2])`, nil, false)
	expectRun(t, rta, `out = immutable([1, 2, 3, 4])[1]`, nil, 2)
	expectRun(t, rta, `out = immutable([1, 2, 3, 4])[1:3]`, nil, ARR{2, 3})
	expectRun(t, rta, `a := immutable([1,2,3]); a = 5; out = a`, nil, 5)

	// map
	expectError(t, rta, `a := immutable({b: 1, c: 2}); a.b = 5`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectError(t, rta, `a := immutable({b: 1, c: 2}); a["b"] = "bar"`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectRun(t, rta, `a := immutable({b: 1, c: [1,2,3]}); a.c[1] = "bar"; out = a`, nil, IMAP{"b": 1, "c": ARR{1, "bar", 3}})
	expectError(t, rta, `a := immutable({b: 1, c: immutable([1,2,3])}); a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectError(t, rta, `a := {b: 1, c: immutable([1,2,3])}; a.c[1] = "bar"`, nil, "not_assignable: type immutable-array does not support assignment via indexing or field access")
	expectRun(t, rta, `out = immutable({a:1,b:2}) == {a:1,b:2}`, nil, true)
	expectRun(t, rta, `out = immutable({a:1,b:2}) == immutable({a:1,b:2})`, nil, true)
	expectRun(t, rta, `out = {a:1,b:2} == immutable({a:1,b:2})`, nil, true)
	expectRun(t, rta, `out = immutable({a:1,b:2}) == {a:1,b:3}`, nil, false)
	expectRun(t, rta, `out = immutable({a:1,b:2}) == immutable({a:1,b:3})`, nil, false)
	expectRun(t, rta, `out = {a:1,b:2} == immutable({a:1,b:3})`, nil, false)
	expectRun(t, rta, `out = immutable({a:1,b:2}).b`, nil, 2)
	expectRun(t, rta, `out = immutable({a:1,b:2})["b"]`, nil, 2)
	expectRun(t, rta, `a := immutable({a:1,b:2}); a = 5; out = 5`, nil, 5)
	expectRun(t, rta, `a := immutable({a:1,b:2}); out = a.c`, nil, core.Undefined)

	expectRun(t, rta, `a := immutable({b: 5, c: "foo"}); out = a.b`, nil, 5)
	expectError(t, rta, `a := immutable({b: 5, c: "foo"}); a.b = 10`, nil, "not_assignable: type immutable-record does not support assignment via indexing or field access")
}

func TestIncDec(t *testing.T) {
	expectRun(t, rta, `out = 0; out++`, nil, 1)
	expectRun(t, rta, `out = 0; out--`, nil, -1)
	expectRun(t, rta, `a := 0; a++; out = a`, nil, 1)
	expectRun(t, rta, `a := 0; a++; a--; out = a`, nil, 0)

	// this seems strange but it works because 'a += b' is
	// translated into 'a = a + b' and string type takes other types for + operator.
	expectRun(t, rta, `a := "foo"; a++; out = a`, nil, "foo1")
	expectError(t, rta, `a := "foo"; a--`, nil, "invalid_binary_operator: string - int")

	expectError(t, rta, `a++`, nil, "unresolved reference") // not declared
	expectError(t, rta, `a--`, nil, "unresolved reference") // not declared
	expectError(t, rta, `4++`, nil, "unresolved reference")
}

func TestIndexable(t *testing.T) {
	dict := func() core.Value {
		return NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, rta, `out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "foo")
	expectRun(t, rta, `out = d["B"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "bar")
	expectRun(t, rta, `out = d["x"]`, Opts().Symbol("d", dict()).Skip2ndPass(), core.Undefined)

	strCir := func() core.Value {
		return NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectRun(t, rta, `out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, rta, `out = cir[-1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "three")
	expectRun(t, rta, `out = cir[-2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "two")
	expectRun(t, rta, `out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "one")
	expectError(t, rta, `cir["a"]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `out = arr["one"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 0)
	expectRun(t, rta, `out = arr["three"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 2)
	expectRun(t, rta, `out = arr["four"]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), core.Undefined)
	expectRun(t, rta, `out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one")
	expectRun(t, rta, `out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "two")
	expectError(t, rta, `arr[-1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "index_out_of_bounds")
}

func TestIndexAssignable(t *testing.T) {
	dict := func() core.Value {
		return NewStringDictValue(map[string]string{"a": "foo", "b": "bar"})
	}

	expectRun(t, rta, `d["a"] = "1984"; out = d["a"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, rta, `d["c"] = "1984"; out = d["c"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")
	expectRun(t, rta, `d["c"] = 1984; out = d["C"]`, Opts().Symbol("d", dict()).Skip2ndPass(), "1984")

	strCir := func() core.Value {
		return NewStringCircleValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `cir[0] = "ONE"; out = cir[0]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectRun(t, rta, `cir[1] = "TWO"; out = cir[1]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "TWO")
	expectRun(t, rta, `cir[-1] = "THREE"; out = cir[2]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "THREE")
	expectRun(t, rta, `cir[0] = "ONE"; out = cir[3]`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "ONE")
	expectError(t, rta, `cir["a"] = "ONE"`, Opts().Symbol("cir", strCir()).Skip2ndPass(), "invalid_index_type")

	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `arr[0] = "ONE"; out = arr[0]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "ONE")
	expectRun(t, rta, `arr[1] = "TWO"; out = arr[1]`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "TWO")
	expectError(t, rta, `arr["one"] = "ONE"`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "invalid_index_type")
}

func TestIterable(t *testing.T) {
	strArr := func() core.Value {
		return NewStringArrayValue([]string{"one", "two", "three"})
	}

	expectRun(t, rta, `for i, s in arr { out += i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), 3)
	expectRun(t, rta, `for i, s in arr { out += s }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "onetwothree")
	expectRun(t, rta, `for i, s in arr { out += s + i }`, Opts().Symbol("arr", strArr()).Skip2ndPass(), "one0two1three2")
}

func TestLogical(t *testing.T) {
	expectRun(t, rta, `out = true && true`, nil, true)
	expectRun(t, rta, `out = true && false`, nil, false)
	expectRun(t, rta, `out = false && true`, nil, false)
	expectRun(t, rta, `out = false && false`, nil, false)
	expectRun(t, rta, `out = !true && true`, nil, false)
	expectRun(t, rta, `out = !true && false`, nil, false)
	expectRun(t, rta, `out = !false && true`, nil, true)
	expectRun(t, rta, `out = !false && false`, nil, false)

	expectRun(t, rta, `out = true || true`, nil, true)
	expectRun(t, rta, `out = true || false`, nil, true)
	expectRun(t, rta, `out = false || true`, nil, true)
	expectRun(t, rta, `out = false || false`, nil, false)
	expectRun(t, rta, `out = !true || true`, nil, true)
	expectRun(t, rta, `out = !true || false`, nil, false)
	expectRun(t, rta, `out = !false || true`, nil, true)
	expectRun(t, rta, `out = !false || false`, nil, true)

	expectRun(t, rta, `out = 1 && 2`, nil, 2)
	expectRun(t, rta, `out = 1 || 2`, nil, 1)
	expectRun(t, rta, `out = 1 && 0`, nil, 0)
	expectRun(t, rta, `out = 1 || 0`, nil, 1)
	expectRun(t, rta, `out = 1 && (0 || 2)`, nil, 2)
	expectRun(t, rta, `out = 0 || (0 || 2)`, nil, 2)
	expectRun(t, rta, `out = 0 || (0 && 2)`, nil, 0)
	expectRun(t, rta, `out = 0 || (2 && 0)`, nil, 0)

	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() && f()`, nil, 7)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() && t()`, nil, 7)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; f() || t()`, nil, 3)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; t() || f()`, nil, 3)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() && f()`, nil, 3)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() && t()`, nil, 3)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !f() || t()`, nil, 7)
	expectRun(t, rta, `t:=func() {out = 3; return true}; f:=func() {out = 7; return false}; !t() || f()`, nil, 7)
}

func TestCustomBuiltin(t *testing.T) {
	m := Opts().BuiltinModule("math1",
		module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"abs",
					func(alc *core.Arena, v core.VM, a []core.Value) (core.Value, error) {
						r, _ := a[0].AsFloat(alc)
						return core.FloatValue(math.Abs(r)), nil
					},
					1,
					false,
				),
			},
		})

	// builtin
	expectRun(t, rta, `math := import("math1"); out = math.abs(1)`, m, 1.0)
	expectRun(t, rta, `math := import("math1"); out = math.abs(-1)`, m, 1.0)
	expectRun(t, rta, `math := import("math1"); out = math.abs(1.0)`, m, 1.0)
	expectRun(t, rta, `math := import("math1"); out = math.abs(-1.0)`, m, 1.0)
}

func TestUserModules(t *testing.T) {
	// export none
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `fn := func() { return 5.0 }; a := 2`),
		core.Undefined)

	// export values
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `export 5`), 5)
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `export "foo"`), "foo")

	// export compound types
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `export [1, 2, 3]`), IARR{1, 2, 3})
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `export {a: 1, b: 2}`), IMAP{"a": 1, "b": 2})

	// export value is immutable
	expectError(t, rta, `m1 := import("mod1"); m1.a = 5`, Opts().Module("mod1", `export {a: 1, b: 2}`), "not_assignable: type immutable-record does not support assignment via indexing or field access")
	expectError(t, rta, `m1 := import("mod1"); m1[1] = 5`, Opts().Module("mod1", `export [1, 2, 3]`), "not_assignable: type immutable-array does not support assignment via indexing or field access")

	// code after export statement will not be executed
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `a := 10; export a; a = 20`), 10)
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `a := 10; export a; a = 20; export a`), 10)

	// export function
	expectRun(t, rta, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { return 5.0 }`), 5.0)
	// export function that reads module-global variable
	expectRun(t, rta, `out = import("mod1")()`,
		Opts().Module("mod1", `a := 1.5; export func() { return a + 5.0 }`), 6.5)
	// export function that read local variable
	expectRun(t, rta, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { a := 1.5; return a + 5.0 }`), 6.5)
	// export function that read free variables
	expectRun(t, rta, `out = import("mod1")()`,
		Opts().Module("mod1", `export func() { a := 1.5; return func() { return a + 5.0 }() }`), 6.5)

	// recursive function in module
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module(
			"mod1", `
a := func(x) {
	return x == 0 ? 0 : x + a(x-1)
}

export a(5)
`), 15)
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module(
			"mod1", `
export func() {
	a := func(x) {
		return x == 0 ? 0 : x + a(x-1)
	}

	return a(5)
}()
`), 15)

	// (main) -> mod1 -> mod2
	expectRun(t, rta, `out = import("mod1")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export func() { return 5.0 }`),
		5.0)
	// (main) -> mod1 -> mod2
	//        -> mod2
	expectRun(t, rta, `import("mod1"); out = import("mod2")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export func() { return 5.0 }`),
		5.0)
	// (main) -> mod1 -> mod2 -> mod3
	//        -> mod2 -> mod3
	expectRun(t, rta, `import("mod1"); out = import("mod2")()`,
		Opts().Module("mod1", `export import("mod2")`).
			Module("mod2", `export import("mod3")`).
			Module("mod3", `export func() { return 5.0 }`),
		5.0)

	// cyclic imports
	// (main) -> mod1 -> mod2 -> mod1
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod1")`),
		"Compile Error: cyclic module import: mod1\n\tat mod2:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod1
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod1")`),
		"Compile Error: cyclic module import: mod1\n\tat mod3:1:1")
	// (main) -> mod1 -> mod2 -> mod3 -> mod2
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`).
			Module("mod2", `import("mod3")`).
			Module("mod3", `import("mod2")`),
		"Compile Error: cyclic module import: mod2\n\tat mod3:1:1")

	// unknown modules
	expectError(t, rta, `import("mod0")`,
		Opts().Module("mod1", `a := 5`), "module 'mod0' not found")
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `import("mod2")`), "module 'mod2' not found")

	// module is immutable but its variables is not necessarily immutable.
	expectRun(t, rta, `m1 := import("mod1"); m1.a.b = 5; out = m1.a.b`,
		Opts().Module("mod1", `export {a: {b: 3}}`),
		5)

	// make sure module has same builtin functions
	expectRun(t, rta, `out = import("mod1")`,
		Opts().Module("mod1", `export func() { return type_name(0) }()`),
		"int")

	// 'export' statement is ignored outside module
	expectRun(t, rta, `a := 5; export func() { a = 10 }(); out = a`,
		Opts().Skip2ndPass(), 5)

	// 'export' must be in the top-level
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `func() { export 5 }()`),
		"Compile Error: export not allowed inside function\n\tat mod1:1:10")
	expectError(t, rta, `import("mod1")`,
		Opts().Module("mod1", `func() { func() { export 5 }() }()`),
		"Compile Error: export not allowed inside function\n\tat mod1:1:19")

	// module cannot access outer scope
	expectError(t, rta, `a := 5; import("mod1")`,
		Opts().Module("mod1", `export a`),
		"Compile Error: unresolved reference 'a'\n\tat mod1:1:8")

	// runtime error within modules
	expectError(t, rta, `
a := 1;
b := import("mod1");
b(a)`,
		Opts().Module("mod1", `
export func(a) {
   a()
}
`), "Runtime Error: not_callable: type int is not callable\n\tat mod1:3:4\n\tat test:4:1")

	// module skipping export
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", ``), core.Undefined)
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", `if 1 { export true }`), true)
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", `if 0 { export true }`),
		core.Undefined)
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", `if 1 { } else { export true }`),
		core.Undefined)
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", `for v:=0;;v++ { if v == 3 { export true } }`),
		true)
	expectRun(t, rta, `out = import("mod0")`,
		Opts().Module("mod0", `for v:=0;;v++ { if v == 3 { break } }`),
		core.Undefined)

	// duplicate compiled functions
	// NOTE: module "mod" has a function with some local variable, and it's
	//  imported twice by the main script. That causes the same CompiledFunction
	//  put in constants twice and the Bytecode optimization (removing duplicate
	//  constants) should still work correctly.
	expectRun(t, rta, `
m1 := import("mod")
m2 := import("mod")
out = m1.x
	`,
		Opts().Module("mod", `
f1 := func(a, b) {
	c := a + b + 1
	return a + b + 1
}
export { x: 1 }
`),
		1)
}

func TestCustomModuleBlockScopes(t *testing.T) {
	m := Opts().BuiltinModule("rand1",
		module{
			fns: map[uint64]*core.BuiltinFunction{
				0: core.NewBuiltinFunction(
					"intn",
					func(alc *core.Arena, v core.VM, a []core.Value) (core.Value, error) {
						r, _ := a[0].AsInt(alc)
						return core.IntValue(rand.Int63n(r)), nil
					},
					1,
					false,
				),
			},
		})

	// block scopes in module
	expectRun(t, rta, `out = import("mod1")()`, m.Module(
		"mod1", `
	rand := import("rand1")
	foo := func() { return 1 }
	export func() {
		rand.intn(3)
		return foo()
	}`), 1)

	expectRun(t, rta, `out = import("mod1")()`, m.Module(
		"mod1", `
rand := import("rand1")
foo := func() { return 1 }
export func() {
	rand.intn(3)
	if foo() {}
	return 10
}
`), 10)

	expectRun(t, rta, `out = import("mod1")()`, m.Module(
		"mod1", `
	rand := import("rand1")
	foo := func() { return 1 }
	export func() {
		rand.intn(3)
		if true { foo() }
		return 10
	}
	`), 10)
}

func TestBangOperator(t *testing.T) {
	expectRun(t, rta, `out = !true`, nil, false)
	expectRun(t, rta, `out = !false`, nil, true)
	expectRun(t, rta, `out = !0`, nil, true)
	expectRun(t, rta, `out = !5`, nil, false)
	expectRun(t, rta, `out = !!true`, nil, true)
	expectRun(t, rta, `out = !!false`, nil, false)
	expectRun(t, rta, `out = !!5`, nil, true)
}

func TestReturn(t *testing.T) {
	expectRun(t, rta, `out = func() { return 10; }()`, nil, 10)
	expectRun(t, rta, `out = func() { return 10; return 9; }()`, nil, 10)
	expectRun(t, rta, `out = func() { return 2 * 5; return 9 }()`, nil, 10)
	expectRun(t, rta, `out = func() { 9; return 2 * 5; return 9 }()`, nil, 10)
	expectRun(t, rta, `
	out = func() {
		if (10 > 1) {
			if (10 > 1) {
				return 10;
	  		}

	  		return 1;
		}
	}()`, nil, 10)

	expectRun(t, rta, `f1 := func() { return 2 * 5; }; out = f1()`, nil, 10)
}

func TestVMScopes(t *testing.T) {
	// shadowed global variable
	expectRun(t, rta, `
c := 5
if a := 3; a {
	c := 6
} else {
	c := 7
}
out = c
`, nil, 5)

	// shadowed local variable
	expectRun(t, rta, `
func() {
	c := 5
	if a := 3; a {
		c := 6
	} else {
		c := 7
	}
	out = c
}()
`, nil, 5)

	// 'b' is declared in 2 separate blocks
	expectRun(t, rta, `
c := 5
if a := 3; a {
	b := 8
	c = b
} else {
	b := 9
	c = b
}
out = c
`, nil, 8)

	// shadowing inside for statement
	expectRun(t, rta, `
a := 4
b := 5
for i:=0;i<3;i++ {
	b := 6
	for j:=0;j<2;j++ {
		b := 7
		a = i*j
	}
}
out = a`, nil, 2)

	// shadowing inside for statement with var init
	expectRun(t, rta, `
a := 0
for var i = 0; i < 3; i++ {
	a += i
}
out = a`, nil, 3)

	// shadowing variable declared in init statement
	expectRun(t, rta, `
if a := 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, rta, `
a := 4
if a := 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, rta, `
a := 4
if a := 0; a {
	a := 6
	out = a
} else {
	a := 7
	out = a
}`, nil, 7)
	expectRun(t, rta, `
a := 4
if a := 0; a {
	out = a
} else {
	out = a
}`, nil, 0)

	// shadowing variable declared in init statement using var
	expectRun(t, rta, `
a := 4
if var a = 5; a {
	a := 6
	out = a
}`, nil, 6)
	expectRun(t, rta, `
a := 4
if var a = 0; a {
	out = 1
} else {
	out = a
}`, nil, 0)

	// shadowing function level
	expectRun(t, rta, `
a := 5
func() {
	a := 6
	a = 7
}()
out = a
`, nil, 5)
	expectRun(t, rta, `
a := 5
func() {
	if a := 7; true {
		a = 8
	}
}()
out = a
`, nil, 5)
}

func TestSelector(t *testing.T) {
	expectRun(t, rta, `a := {k1: 5, k2: "foo"}; out = a.k1`, nil, 5)
	expectRun(t, rta, `a := {k1: 5, k2: "foo"}; out = a.k2`, nil, "foo")
	expectRun(t, rta, `a := {k1: 5, k2: "foo"}; out = a.k3`, nil, core.Undefined)

	expectRun(t, rta, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
out = a.b.c`, nil, 4)

	expectRun(t, rta, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
b := a.x.c`, nil, core.Undefined)

	expectRun(t, rta, `
a := {
	b: {
		c: 4,
		a: false
	},
	c: "foo bar"
}
b := a.x.y`, nil, core.Undefined)

	expectRun(t, rta, `a := {b: 1, c: "foo"}; a.b = 2; out = a.b`, nil, 2)
	expectRun(t, rta, `a := {b: 1, c: "foo"}; a.c = 2; out = a.c`, nil, 2) // type not checked on sub-field
	expectRun(t, rta, `a := {b: {c: 1}}; a.b.c = 2; out = a.b.c`, nil, 2)
	expectRun(t, rta, `a := {b: 1}; a.c = 2; out = a`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, rta, `a := {b: {c: 1}}; a.b.d = 2; out = a`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, rta, `func() { a := {b: 1, c: "foo"}; a.b = 2; out = a.b }()`, nil, 2)
	expectRun(t, rta, `func() { a := {b: 1, c: "foo"}; a.c = 2; out = a.c }()`, nil, 2) // type not checked on sub-field
	expectRun(t, rta, `func() { a := {b: {c: 1}}; a.b.c = 2; out = a.b.c }()`, nil, 2)
	expectRun(t, rta, `func() { a := {b: 1}; a.c = 2; out = a }()`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, rta, `func() { a := {b: {c: 1}}; a.b.d = 2; out = a }()`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, rta, `func() { a := {b: 1, c: "foo"}; func() { a.b = 2 }(); out = a.b }()`, nil, 2)
	expectRun(t, rta, `func() { a := {b: 1, c: "foo"}; func() { a.c = 2 }(); out = a.c }()`, nil, 2) // type not checked on sub-field
	expectRun(t, rta, `func() { a := {b: {c: 1}}; func() { a.b.c = 2 }(); out = a.b.c }()`, nil, 2)
	expectRun(t, rta, `func() { a := {b: 1}; func() { a.c = 2 }(); out = a }()`, nil, MAP{"b": 1, "c": 2})
	expectRun(t, rta, `func() { a := {b: {c: 1}}; func() { a.b.d = 2 }(); out = a }()`, nil, MAP{"b": MAP{"c": 1, "d": 2}})

	expectRun(t, rta, `
a := {
	b: [1, 2, 3],
	c: {
		d: 8,
		e: "foo",
		f: [9, 8]
	}
}
out = [a.b[2], a.c.d, a.c.e, a.c.f[1]]
`, nil, ARR{3, 8, "foo", 8})

	expectRun(t, rta, `
func() {
	a := [1, 2, 3]
	b := 9
	a[1] = b
	b = 7     // make sure a[1] has a COPY of value of 'b'
	out = a[1]
}()
`, nil, 9)

	expectError(t, rta, `a := {b: {c: 1}}; a.d.c = 2`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
	expectError(t, rta, `a := [1, 2, 3]; a.b = 2`, nil, "invalid_index_type: (index assign) expected int, got string")
	expectError(t, rta, `a := "foo"; a.b = 2`, nil, "not_assignable: type string does not support assignment via indexing or field access")
	expectError(t, rta, `func() { a := {b: {c: 1}}; a.d.c = 2 }()`, nil, "not_assignable: type undefined does not support assignment via indexing or field access")
	expectError(t, rta, `func() { a := [1, 2, 3]; a.b = 2 }()`, nil, "invalid_index_type")
	expectError(t, rta, `func() { a := "foo"; a.b = 2 }()`, nil, "not_assignable: type string does not support assignment via indexing or field access")
}

func TestVMNewStackOverflowError(t *testing.T) {
	expectError(t, rta, `f := func() { return f() + 1 }; f()`, nil, "stack_overflow")
}

func TestTailCall(t *testing.T) {
	expectRun(t, rta, `
	fac := func(n, a) {
		if n == 1 {
			return a
		}
		return fac(n-1, n*a)
	}
	out = fac(5, 1)`, nil, 120)

	expectRun(t, rta, `
	fac := func(n, a) {
		if n == 1 {
			return a
		}
		x := {foo: fac} // indirection for test
		return x.foo(n-1, n*a)
	}
	out = fac(5, 1)`, nil, 120)

	expectRun(t, rta, `
	fib := func(x, s) {
		if x == 0 {
			return 0 + s
		} else if x == 1 {
			return 1 + s
		}
		return fib(x-1, fib(x-2, s))
	}
	out = fib(15, 0)`, nil, 610)

	expectRun(t, rta, `
	fib := func(n, a, b) {
		if n == 0 {
			return a
		} else if n == 1 {
			return b
		}
		return fib(n-1, b, a + b)
	}
	out = fib(15, 0, 1)`, nil, 610)

	// global variable and no return value
	expectRun(t, rta, `
			out = 0
			foo := func(a) {
			   if a == 0 {
			       return
			   }
			   out += a
			   foo(a-1)
			}
			foo(10)`, nil, 55)

	expectRun(t, rta, `
	f1 := func() {
		f2 := 0    // TODO: this might be fixed in the future
		f2 = func(n, s) {
			if n == 0 { return s }
			return f2(n-1, n + s)
		}
		return f2(5, 0)
	}
	out = f1()`, nil, 15)

	// tail-call replacing loop
	// without tail-call optimization, this code will cause stack_overflow
	expectRun(t, rta, `
iter := func(n, max) {
	if n == max {
		return n
	}

	return iter(n+1, max)
}
out = iter(0, 9999)
`, nil, 9999)
	expectRun(t, rta, `
c := 0
iter := func(n, max) {
	if n == max {
		return
	}

	c++
	iter(n+1, max)
}
iter(0, 9999)
out = c
`, nil, 9999)
}

// tail call with free vars
func TestTailCallFreeVars(t *testing.T) {
	expectRun(t, rta, `
func() {
	a := 10
	f2 := 0
	f2 = func(n, s) {
		if n == 0 {
			return s + a
		}
		return f2(n-1, n+s)
	}
	out = f2(5, 0)
}()`, nil, 25)
}

func TestSpread(t *testing.T) {
	expectRun(t, rta, `
	f := func(...a) {
		return append(a, 3)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
	f := func(a, ...b) {
		return append([a], append(b, 3)...)
	}
	out = f([1, 2]...)
	`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
	f := func(a, ...b) {
		return append(append([a], b), 3)
	}
	out = f(1, [2]...)
	`, nil, ARR{1, ARR{2}, 3})

	expectRun(t, rta, `
	f1 := func(...a){
		return append([3], a...)
	}
	f2 := func(a, ...b) {
		return f1(append([a], b...)...)
	}
	out = f2([1, 2]...)
	`, nil, ARR{3, 1, 2})

	expectRun(t, rta, `
	f := func(a, ...b) {
		return func(...a) {
			return append([3], append(a, 4)...)
		}(a, b...)
	}
	out = f([1, 2]...)
	`, nil, ARR{3, 1, 2, 4})

	expectRun(t, rta, `
	f := func(a, ...b) {
		c := append(b, 4)
		return func(){
			return append(append([a], b...), c...)
		}()
	}
	out = f(1, immutable([2, 3])...)
	`, nil, ARR{1, 2, 3, 2, 3, 4})

	expectError(t, rta, `func(a) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 1 argument(s), got 2")
	expectError(t, rta, `func(a, b, c) {}([1, 2]...)`, nil, "Runtime Error: wrong_num_arguments: (call) expected 3 argument(s), got 2")
}

func TestSliceIndex(t *testing.T) {
	expectError(t, rta, `undefined[:1]`, nil, "Runtime Error: not_sliceable: type undefined does not support slicing")
	expectError(t, rta, `123[-1:2]`, nil, "Runtime Error: not_sliceable: type int does not support slicing")
	expectError(t, rta, `{}[:]`, nil, "Runtime Error: not_sliceable: type record does not support slicing")
	expectError(t, rta, `a := 123[-1:2] ; a += 1`, nil, "Runtime Error: not_sliceable: type int does not support slicing")
}

func TestLambdas(t *testing.T) {
	expectRun(t, rta, `
	foo := (a, b) => { return a + b }
	out = foo(1, 2)`, nil, 3)

	expectRun(t, rta, `
	foo := (a) => { return a + 2 }
	out = foo(1)`, nil, 3)

	expectRun(t, rta, `
	foo := a => { return a + 2 }
	out = foo(1)`, nil, 3)

	expectRun(t, rta, `
	foo := () => { return 3 }
	out = foo()`, nil, 3)

	expectRun(t, rta, `
	foo := (a, b) => a + b
	out = foo(1, 2)`, nil, 3)

	expectRun(t, rta, `
	foo := (a) => a + 2
	out = foo(1)`, nil, 3)

	expectRun(t, rta, `
	foo := a => a + 2
	out = foo(1)`, nil, 3)

	expectRun(t, rta, `
	foo := () => 3
	out = foo()`, nil, 3)

	expectRun(t, rta, `
	foo := (a, f) => f(a)
	out = foo(3, x => x*2)`, nil, 6)

	expectRun(t, rta, `
	foo := (f, a) => f(a)
	out = foo(x => x*2, 3)`, nil, 6)
}

func TestIntegrity(t *testing.T) {
	expectRun(t, rta, `
		x := [9, 8, 7, 6, 5, 4, 3, 2, 1]
		r1 := x.sort().filter(e => e % 2 == 0).last()
		y := dict({a: 1, b: 2, c: 3})
		r2 := y.values().sort().filter(e => e == 2).first()

		out = string([r1, r2])
	`, nil, string([]byte{8, 2}))

	expectRun(t, rta, `
		x = [9, 8, 7, 6, 5, 4, 3, 2, 1]
		r1 = x.sort().filter(e => e % 2 == 0).last()
		y = dict({a: 1, b: 2, c: 3})
		r2 = y.values().sort().filter(e => e == 2).first()

		out = string([r1, r2])
	`, nil, string([]byte{8, 2}))

	expectRun(t, rta, `
		out = [1, 2, 3]
			.sort()
			.filter(e => e > 1)
			.sum()
	`, nil, 5)
}

func TestInSyntax(t *testing.T) {
	// element iterator
	expectRun(t, rta, `
		y := [1, 2, 3]
		out = 0
		for x in y {
			out += x
		}
	`, nil, 6)

	// index and element iterator
	expectRun(t, rta, `
		y := [1, 2, 3]
		s1 := 0
		s2 := 0
		for i, x in y {
			s1 += i
			s2 += x
		}
		out = [s1, s2]
	`, nil, ARR{3, 6})

	// loop with condition
	expectRun(t, rta, `
		y := {a: 1, b: 2, c: 3}
		c := 0
		s := 0
		ks := ["a", "b", "c"]
		for i, x in ks {
			if !(x in y) { break }
			c += 1
			s += y[x]
			delete(y, x)
		}
		out = [c, s]
	`, nil, ARR{3, 6})

	// condition
	expectRun(t, rta, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if x in y {
			out = 1
		} else {
			out = 0
		}
	`, nil, 1)

	expectRun(t, rta, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if (x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 1)

	expectRun(t, rta, `
		y := {a: 1, b: 2, c: 3}
		x := "a"
		if !(x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 0)

	expectRun(t, rta, `
		y := {a: 1, b: 2, c: 3}
		x := "z"
		if (x in y) {
			out = 1
		} else {
			out = 0
		}
	`, nil, 0)
}

func TestVarSyntax(t *testing.T) {
	expectRun(t, rta, `
		var x = 1
		var y = 2
		out = x + y
	`, nil, 3)

	expectRun(t, rta, `
		var x = 1
		x = 2
		out = x
	`, nil, 2)

	expectRun(t, rta, `
		var x
		x = 2
		out = x
	`, nil, 2)

	expectRun(t, rta, `
		var x = 1
		func() {
			x = 2
		}()
		out = x
	`, nil, 2)

	expectRun(t, rta, `
		var x = 1
		func() {
			var x = 2
			out = x
		}()
	`, nil, 2)

	expectRun(t, rta, `
		var x = 1
		func() {
			var x = 2
			func() {
				x = 3
			}()
			out = x
		}()
	`, nil, 3)
}

func TestDivBy0(t *testing.T) {
	expectRun(t, rta, `out = 1.0 / 0.0`, nil, math.Inf(0))
	expectRun(t, rta, `out = 1.0 / 0`, nil, math.Inf(0))
	expectRun(t, rta, `out = 1 / 0.0`, nil, math.Inf(0))
	expectError(t, rta, `1 / 0`, nil, "division_by_zero")
}

func TestExamples(t *testing.T) {
	expectRun(t, rta, `
out = {a: 1, b: 2}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, rta, `
out = {a: 1,
	b: 2}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, rta, `
out = {
	a: 1,
	b: 2
}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, rta, `
out = {
	a: 1,
	b: 2,
}
`, nil, MAP{"a": 1, "b": 2})

	expectRun(t, rta, `
out = [1, 2, 3].sum()
`, nil, 6)

	expectRun(t, rta, `
out = [1, 2, 3]
	.sum()
`, nil, 6)

	expectRun(t, rta, `
out = [1, 2, 3].map(x => x*x).sum()
`, nil, 14)

	expectRun(t, rta, `
out = [1, 2, 3]
	.map(x => x*x)
	.sum()
`, nil, 14)

	expectRun(t, rta, `
out = [1, 2, 3]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
out = [1,
	2,
	3]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
out = [1,
	2,
	3]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
out = [
	1,
	2,
	3
]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
out = [
	1,
	2,
	3,
]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
out =
	[
		1,
		2,
		3,
	]
`, nil, ARR{1, 2, 3})

	expectRun(t, rta, `
result := [1, 2, 3, 4, 5, 6]
  .filter(x => x % 2 == 0)
  .map(x => x * x)
  .reduce(0, (sum, x) => sum + x)
out = result
`, nil, 56)

	expectRun(t, rta, `
orders := [
  {customer: "Ada", total: 120, paid: true},
  {customer: "Linus", total: 75, paid: false},
  {customer: "Grace", total: 210, paid: true},
  {customer: "Ken", total: 95, paid: true},
]

paid_total := orders
  .filter(order => order.paid)
  .map(order => order.total)
  .sum()

vip_customers := orders
  .filter(order => order.total >= 100)
  .map(order => order.customer)

out = [paid_total, vip_customers]
`, nil, ARR{425, ARR{"Ada", "Grace"}})
}

func TestVariableDeclarationAndShadowing(t *testing.T) {
	expectRun(t, rta, `
x := 1
out = x
`, nil, 1)

	expectRun(t, rta, `
x = 1
out = x
`, nil, 1)

	expectRun(t, rta, `
x := 1
for i in [0, 1, 2] {
	x = i // assignment to outer variable
}
out = x
`, nil, 2)

	expectRun(t, rta, `
x = 1
for i in [0, 1, 2] {
	x = i // assignment to outer variable
}
out = x
`, nil, 2)

	expectRun(t, rta, `
x := 1
for i in [0, 1, 2] {
	x := i // declaration of new variable that shadows outer variable, so outer variable is not modified
}
out = x
`, nil, 1)

	expectRun(t, rta, `
x = 1
for i in [0, 1, 2] {
	x := i // declaration of new variable that shadows outer variable, so outer variable is not modified
}
out = x
`, nil, 1)

	expectRun(t, rta, `
x := 1
foo := func() {
	x = 2 // assignment to outer variable
}
foo()
out = x
`, nil, 2)

	expectRun(t, rta, `
x = 1
foo = func() {
	x = 2 // assignment to outer variable
}
foo()
out = x
`, nil, 2)

	expectRun(t, rta, `
x := 1
foo := func() {
	x := 2 // declaration of new variable that shadows outer variable, so outer variable is not modified
}
foo()
out = x
`, nil, 1)

	expectRun(t, rta, `
x = 1
foo = func() {
	x := 2 // declaration of new variable that shadows outer variable, so outer variable is not modified
}
foo()
out = x
`, nil, 1)

	expectRun(t, rta, `
x = 0
y = 0
if x = 10; x > 0 {
    y = 1
} else {
    y = 2
}
out = [x, y]
`, nil, ARR{10, 1}) // x == 10, y == 1 (= modifies outer x)

	expectRun(t, rta, `
x = 0
y = 0
if x := 10; x > 0 {
    y = 1
} else {
    y = 2
}
out = [x, y]
`, nil, ARR{0, 1}) // x == 0, y == 1 (:= declares new local x in if block)
}

func TestRepeat(t *testing.T) {
	// Scalars -> array of n copies
	expectRun(t, rta, `x := 1; out = x.repeat(3)`, nil, ARR{1, 1, 1})
	expectRun(t, rta, `x := 0; out = x.repeat(0)`, nil, ARR{})
	expectRun(t, rta, `x := 7; out = x.repeat(1)`, nil, ARR{7})
	expectRun(t, rta, `b := true; out = b.repeat(2)`, nil, ARR{true, true})
	expectRun(t, rta, `f := 1.5; out = f.repeat(2)`, nil, ARR{1.5, 1.5})
	expectRun(t, rta, `out = undefined.repeat(3)`, nil, ARR{core.Undefined, core.Undefined, core.Undefined})

	// decimal & time -> array of n copies (reference scalars are immutable in user-land)
	expectRun(t, rta, `d := decimal("1.5"); out = d.repeat(2).len()`, nil, 2)
	expectRun(t, rta, `d := decimal("1.5"); out = d.repeat(2)[0] == d`, nil, true)
	expectRun(t, rta, `d := decimal("1.5"); out = d.repeat(2)[1] == d`, nil, true)
	expectRun(t, rta, `d := decimal("0").repeat(0); out = d`, nil, ARR{})
	expectRun(t, rta, `t := time(0); out = t.repeat(3).len()`, nil, 3)

	// byte -> bytes (specialized concat)
	expectRun(t, rta, `out = byte(65).repeat(3)`, nil, rta.NewBytesValue([]byte{65, 65, 65}, false))
	expectRun(t, rta, `out = byte(0).repeat(0)`, nil, rta.NewBytesValue([]byte{}, false))
	expectRun(t, rta, `out = byte(255).repeat(2)`, nil, rta.NewBytesValue([]byte{255, 255}, false))

	// rune -> runes (specialized concat)
	expectRun(t, rta, `out = 'a'.repeat(3)`, nil, []rune("aaa"))
	expectRun(t, rta, `out = 'a'.repeat(0)`, nil, []rune(""))
	expectRun(t, rta, `out = 'こ'.repeat(2)`, nil, []rune("ここ"))

	// string -> string concat
	expectRun(t, rta, `out = "ab".repeat(3)`, nil, "ababab")
	expectRun(t, rta, `out = "".repeat(5)`, nil, "")
	expectRun(t, rta, `out = "x".repeat(0)`, nil, "")
	expectRun(t, rta, `out = "-".repeat(5)`, nil, "-----")
	expectRun(t, rta, `out = "їЇ".repeat(2)`, nil, "їЇїЇ")

	// bytes -> bytes concat
	expectRun(t, rta, `out = "AB".bytes().repeat(3)`, nil, rta.NewBytesValue([]byte{65, 66, 65, 66, 65, 66}, false))
	expectRun(t, rta, `out = "".bytes().repeat(5)`, nil, rta.NewBytesValue([]byte{}, false))
	expectRun(t, rta, `out = "x".bytes().repeat(0)`, nil, rta.NewBytesValue([]byte{}, false))

	// runes -> runes concat
	expectRun(t, rta, `out = u"ab".repeat(3)`, nil, []rune("ababab"))
	expectRun(t, rta, `out = u"".repeat(5)`, nil, []rune(""))
	expectRun(t, rta, `out = u"x".repeat(0)`, nil, []rune(""))

	// array -> array concat
	expectRun(t, rta, `out = [1, 2].repeat(3)`, nil, ARR{1, 2, 1, 2, 1, 2})
	expectRun(t, rta, `out = [].repeat(5)`, nil, ARR{})
	expectRun(t, rta, `out = [1, 2, 3].repeat(0)`, nil, ARR{})
	expectRun(t, rta, `out = [1].repeat(1)`, nil, ARR{1})

	// chains and idioms
	expectRun(t, rta, `out = "ab".repeat(3).len()`, nil, 6)
	expectRun(t, rta, `out = [1, 2].repeat(3).sum()`, nil, 9)

	// negative count -> error
	expectError(t, rta, `"ab".repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, rta, `[1].repeat(-2)`, nil, "repeat count must be non-negative")
	expectError(t, rta, `byte(1).repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, rta, `'a'.repeat(-1)`, nil, "repeat count must be non-negative")
	expectError(t, rta, `(1).repeat(-1)`, nil, "repeat count must be non-negative")

	// wrong arity / arg type
	expectError(t, rta, `"ab".repeat()`, nil, "wrong_num_arguments")
	expectError(t, rta, `"ab".repeat(1, 2)`, nil, "wrong_num_arguments")
	expectError(t, rta, `"ab".repeat([])`, nil, "invalid_argument_type")
}

func TestJoin(t *testing.T) {
	// array seq with string sep
	expectRun(t, rta, `out = [1, 2, 3].join(", ")`, nil, "1, 2, 3")
	// string sep, array arg (sep-as-receiver)
	expectRun(t, rta, `out = ", ".join([1, 2, 3])`, nil, "1, 2, 3")
	// default sep
	expectRun(t, rta, `out = [1, 2, 3].join()`, nil, "123")
	// empty seq
	expectRun(t, rta, `out = [].join(", ")`, nil, "")
	expectRun(t, rta, `out = ", ".join([])`, nil, "")
	// single element
	expectRun(t, rta, `out = [42].join(", ")`, nil, "42")
	// mixed types stringified via AsString (same as `+` operator)
	expectRun(t, rta, `out = [1, "a", true].join(" | ")`, nil, "1 | a | true")
	// undefined is not string-coercible (consistent with `+`)
	expectError(t, rta, `[1, undefined].join(",")`, nil, "cannot convert undefined to string")

	// runes sep (both directions) -> runes result; encode to bytes("aXbXc")
	expectRun(t, rta, `out = bytes([1, 2, 3].join(u","))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, rta, `out = bytes(u",".join([1, 2, 3]))`, nil, []byte{'1', ',', '2', ',', '3'})

	// rune sep -> runes result
	expectRun(t, rta, `out = bytes([1, 2, 3].join(','))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, rta, `out = bytes(','.join([1, 2, 3]))`, nil, []byte{'1', ',', '2', ',', '3'})

	// byte sep -> bytes result
	expectRun(t, rta, `out = [1, 2, 3].join(byte(0x2C))`, nil, []byte{'1', ',', '2', ',', '3'})
	expectRun(t, rta, `out = byte(0x2C).join([1, 2, 3])`, nil, []byte{'1', ',', '2', ',', '3'})

	// range as seq
	expectRun(t, rta, `out = range(1, 4).join(",")`, nil, "1,2,3")
	expectRun(t, rta, `out = ",".join(range(1, 4))`, nil, "1,2,3")
	expectRun(t, rta, `out = range(0, 0).join(",")`, nil, "")

	// errors: wrong sep type for array.join
	expectError(t, rta, `[1, 2].join(123)`, nil, "invalid_argument_type")
	// errors: wrong seq type for sep.join
	expectError(t, rta, `", ".join("ab")`, nil, "invalid_argument_type")
	expectError(t, rta, `", ".join(123)`, nil, "invalid_argument_type")
	// errors: arity
	expectError(t, rta, `", ".join()`, nil, "wrong_num_arguments")
	expectError(t, rta, `", ".join([1], [2])`, nil, "wrong_num_arguments")
	expectError(t, rta, `[1, 2].join(",", "x")`, nil, "wrong_num_arguments")
}

func TestSplit(t *testing.T) {
	// string.split — basic literal
	expectRun(t, rta, `out = "a,b,c".split(",")`, nil, ARR{"a", "b", "c"})
	expectRun(t, rta, `out = "a,b,c".split(",", 1)`, nil, ARR{"a", "b,c"})
	expectRun(t, rta, `out = "a,b,c".split(",", 0)`, nil, ARR{"a,b,c"})
	expectRun(t, rta, `out = "a,b,c".split(",", -1)`, nil, ARR{"a", "b", "c"})
	// string.split — whitespace default
	expectRun(t, rta, `out = "  hello  world  ".split()`, nil, ARR{"hello", "world"})
	// string.split — leading/trailing/consecutive seps preserved
	expectRun(t, rta, `out = ",a,".split(",")`, nil, ARR{"", "a", ""})
	expectRun(t, rta, `out = "a,,b".split(",")`, nil, ARR{"a", "", "b"})
	// string.split — sep not found
	expectRun(t, rta, `out = "abc".split("x")`, nil, ARR{"abc"})
	// string.split — empty receiver
	expectRun(t, rta, `out = "".split(",")`, nil, ARR{})
	expectRun(t, rta, `out = "".split()`, nil, ARR{})
	// string.split — cross-type sep
	expectRun(t, rta, `out = "a,b".split(',')`, nil, ARR{"a", "b"})
	expectRun(t, rta, `out = "a,b".split(byte(0x2C))`, nil, ARR{"a", "b"})
	expectRun(t, rta, `out = "a,b".split(u",")`, nil, ARR{"a", "b"})

	// runes.split
	expectRun(t, rta, `out = bytes(u"a,b,c".split(",")[1])`, nil, []byte{'b'})
	expectRun(t, rta, `out = u"a b c".split().len()`, nil, int64(3))
	expectRun(t, rta, `out = u"".split(",").len()`, nil, int64(0))

	// bytes.split
	expectRun(t, rta, `out = bytes("a,b,c").split(",").len()`, nil, int64(3))
	expectRun(t, rta, `out = bytes("a,b,c").split(byte(0x2C)).len()`, nil, int64(3))
	expectRun(t, rta, `out = bytes("a b c").split().len()`, nil, int64(3))
	expectRun(t, rta, `out = bytes("").split(",").len()`, nil, int64(0))
	expectRun(t, rta, `out = bytes("a,b,c").split(",", 1)[1]`, nil, []byte("b,c"))

	// errors
	expectError(t, rta, `"a,b".split("")`, nil, "split separator must not be empty")
	expectError(t, rta, `"a,b".split([])`, nil, "invalid_argument_type")
	expectError(t, rta, `"a,b".split(",", "x")`, nil, "invalid_argument_type")
	expectError(t, rta, `"a,b".split(",", 1, 2)`, nil, "wrong_num_arguments")
	expectError(t, rta, `bytes("a,b").split([])`, nil, "invalid_argument_type")
}

func TestSplitLines(t *testing.T) {
	expectRun(t, rta, `out = "a\nb\nc".split_lines()`, nil, ARR{"a", "b", "c"})
	expectRun(t, rta, `out = "a\r\nb\rc\nd".split_lines()`, nil, ARR{"a", "b", "c", "d"})
	expectRun(t, rta, `out = "trail\n".split_lines()`, nil, ARR{"trail"})
	expectRun(t, rta, `out = "no_newline".split_lines()`, nil, ARR{"no_newline"})
	expectRun(t, rta, `out = "".split_lines()`, nil, ARR{})
	expectRun(t, rta, `out = "\n\n".split_lines()`, nil, ARR{"", ""})

	// runes / bytes
	expectRun(t, rta, `out = u"a\nb".split_lines().len()`, nil, int64(2))
	expectRun(t, rta, `out = bytes("a\nb").split_lines().len()`, nil, int64(2))

	expectError(t, rta, `"x".split_lines("y")`, nil, "wrong_num_arguments")
}

func TestPartition(t *testing.T) {
	expectRun(t, rta, `out = "a=1=b".partition("=")`, nil, ARR{"a", "=", "1=b"})
	expectRun(t, rta, `out = "abc".partition("x")`, nil, ARR{"abc", "", ""})
	expectRun(t, rta, `out = "".partition(",")`, nil, ARR{"", "", ""})
	expectRun(t, rta, `out = "a,b".partition(',')`, nil, ARR{"a", ",", "b"})
	expectRun(t, rta, `out = "a,b".partition(byte(0x2C))`, nil, ARR{"a", ",", "b"})

	// runes
	expectRun(t, rta, `out = u"a=b".partition("=").len()`, nil, int64(3))
	expectRun(t, rta, `out = bytes(u"a=b".partition("=")[1])`, nil, []byte{'='})

	// bytes
	expectRun(t, rta, `out = bytes("k=v").partition("=").len()`, nil, int64(3))
	expectRun(t, rta, `out = bytes("k=v").partition("=")[0]`, nil, []byte("k"))
	expectRun(t, rta, `out = bytes("k=v").partition("=")[1]`, nil, []byte("="))
	expectRun(t, rta, `out = bytes("k=v").partition("=")[2]`, nil, []byte("v"))
	expectRun(t, rta, `out = bytes("abc").partition("x")[0]`, nil, []byte("abc"))

	// errors
	expectError(t, rta, `"a".partition("")`, nil, "partition separator must not be empty")
	expectError(t, rta, `"a".partition([])`, nil, "invalid_argument_type")
	expectError(t, rta, `"a".partition()`, nil, "wrong_num_arguments")
	expectError(t, rta, `bytes("a").partition([])`, nil, "invalid_argument_type")
}

func TestFlatten(t *testing.T) {
	// no nested arrays — no-op (but still produces a fresh array)
	expectRun(t, rta, `out = [1, 2, 3].flatten()`, nil, ARR{int64(1), int64(2), int64(3)})
	// one level nesting
	expectRun(t, rta, `out = [[1, 2], [3, 4]].flatten()`, nil, ARR{int64(1), int64(2), int64(3), int64(4)})
	// default depth = 1: deeper nesting preserved
	expectRun(t, rta, `out = [1, [2, 3], [4, [5, 6]]].flatten()`, nil, ARR{int64(1), int64(2), int64(3), int64(4), ARR{int64(5), int64(6)}})
	// explicit depth
	expectRun(t, rta, `out = [1, [2, 3], [4, [5, 6]]].flatten(2)`, nil, ARR{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)})
	// unbounded (negative)
	expectRun(t, rta, `out = [1, [[2, [[3]]]]].flatten(-1)`, nil, ARR{int64(1), int64(2), int64(3)})
	expectRun(t, rta, `out = [1, [[2, [[3]]]]].flatten(-100)`, nil, ARR{int64(1), int64(2), int64(3)})
	// depth 0 = shallow copy (no unwrap)
	expectRun(t, rta, `out = [1, [2, [3]]].flatten(0)`, nil, ARR{int64(1), ARR{int64(2), ARR{int64(3)}}})
	// empty
	expectRun(t, rta, `out = [].flatten()`, nil, ARR{})
	expectRun(t, rta, `out = [].flatten(5)`, nil, ARR{})
	// non-array elements stay intact
	expectRun(t, rta, `out = ["ab", [1, 2]].flatten()`, nil, ARR{"ab", int64(1), int64(2)})
	expectRun(t, rta, `out = [[1], "abc", [[2, 3]]].flatten(1)`, nil, ARR{int64(1), "abc", ARR{int64(2), int64(3)}})
	// fresh top-level array (mutating result doesn't affect original)
	expectRun(t, rta, `
		x = [[1, 2], [3, 4]]
		y = x.flatten()
		y[0] = 99
		out = x[0][0]
	`, nil, int64(1))

	// errors
	expectError(t, rta, `[1, 2].flatten("x")`, nil, "invalid_argument_type")
	expectError(t, rta, `[1, 2].flatten(1, 2)`, nil, "wrong_num_arguments")
}
