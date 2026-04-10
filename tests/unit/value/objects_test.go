package value

import (
	"testing"
	"time"

	"github.com/jokruger/gs/core"
	mock "github.com/jokruger/gs/tests"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/token"
	_ "github.com/jokruger/gs/vm"
)

var vm = mock.Vm
var alloc = mock.Alloc

func TestObject_Value(t *testing.T) {
	var v core.Value
	var x core.Value
	var bs []byte
	var err error

	// Undefined
	v = core.UndefinedValue()
	require.True(t, v.IsUndefined())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsUndefined())
	require.Equal(t, true, v.Equal(x))

	// Bool
	v = core.BoolValue(true)
	require.True(t, v.IsBool())
	require.Equal(t, true, v.Bool())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsBool())
	require.Equal(t, true, x.Bool())
	require.Equal(t, true, v.Equal(x))

	v = core.BoolValue(false)
	require.True(t, v.IsBool())
	require.Equal(t, false, v.Bool())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsBool())
	require.Equal(t, false, x.Bool())
	require.Equal(t, true, v.Equal(x))

	// Char
	v = core.CharValue('A')
	require.True(t, v.IsChar())
	require.Equal(t, 'A', v.Char())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsChar())
	require.Equal(t, 'A', x.Char())
	require.Equal(t, true, v.Equal(x))

	v = core.CharValue('₴')
	require.True(t, v.IsChar())
	require.Equal(t, '₴', v.Char())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsChar())
	require.Equal(t, '₴', x.Char())
	require.Equal(t, true, v.Equal(x))

	// Int
	v = core.IntValue(123)
	require.True(t, v.IsInt())
	require.Equal(t, int64(123), v.Int())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsInt())
	require.Equal(t, int64(123), x.Int())
	require.Equal(t, true, v.Equal(x))

	v = core.IntValue(-456)
	require.True(t, v.IsInt())
	require.Equal(t, int64(-456), v.Int())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsInt())
	require.Equal(t, int64(-456), x.Int())
	require.Equal(t, true, v.Equal(x))

	// Float
	v = core.FloatValue(3.14)
	require.True(t, v.IsFloat())
	require.Equal(t, 3.14, v.Float())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsFloat())
	require.Equal(t, 3.14, x.Float())
	require.Equal(t, true, v.Equal(x))

	v = core.FloatValue(-2.71828)
	require.True(t, v.IsFloat())
	require.Equal(t, -2.71828, v.Float())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsFloat())
	require.Equal(t, -2.71828, x.Float())
	require.Equal(t, true, v.Equal(x))

	// String
	v = alloc.NewStringValue("")
	require.True(t, v.IsString())
	s, _ := v.AsString()
	require.Equal(t, "", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsString())
	s, _ = x.AsString()
	require.Equal(t, "", s)
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewStringValue("hello")
	require.True(t, v.IsString())
	s, _ = v.AsString()
	require.Equal(t, "hello", s)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsString())
	s, _ = x.AsString()
	require.Equal(t, "hello", s)
	require.Equal(t, true, v.Equal(x))

	// Bytes
	v = alloc.NewBytesValue([]byte{})
	require.True(t, v.IsBytes())
	b, _ := v.AsBytes()
	require.Equal(t, []byte{}, b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsBytes())
	b, _ = x.AsBytes()
	require.Equal(t, []byte{}, b)
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewBytesValue([]byte("foo"))
	require.True(t, v.IsBytes())
	b, _ = v.AsBytes()
	require.Equal(t, []byte("foo"), b)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsBytes())
	b, _ = x.AsBytes()
	require.Equal(t, []byte("foo"), b)
	require.Equal(t, true, v.Equal(x))

	// Array
	v = alloc.NewArrayValue([]core.Value{}, false)
	require.True(t, v.IsArray())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsArray())
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	require.True(t, v.IsArray())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsArray())
	require.Equal(t, true, v.Equal(x))

	// Record
	v = alloc.NewRecordValue(map[string]core.Value{}, true)
	require.True(t, v.IsRecord())
	require.True(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsRecord())
	require.True(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.IsRecord())
	require.False(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsRecord())
	require.False(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	// Map
	v = alloc.NewMapValue(map[string]core.Value{}, true)
	require.True(t, v.IsMap())
	require.True(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsMap())
	require.True(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewMapValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	require.True(t, v.IsMap())
	require.False(t, v.IsImmutable())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsMap())
	require.False(t, x.IsImmutable())
	require.Equal(t, true, v.Equal(x))

	// Error
	v = alloc.NewErrorValue(core.UndefinedValue())
	require.True(t, v.IsError())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsError())
	require.Equal(t, true, v.Equal(x))

	v = alloc.NewErrorValue(core.NewStringValue("some error"))
	require.True(t, v.IsError())
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsError())
	require.Equal(t, true, v.Equal(x))

	// Time
	v = alloc.NewTimeValue(time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC))
	require.True(t, v.IsTime())
	tm, _ := v.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	bs, err = v.EncodeBinary()
	require.NoError(t, err)
	err = x.DecodeBinary(bs)
	require.NoError(t, err)
	require.True(t, x.IsTime())
	tm, _ = x.AsTime()
	require.Equal(t, time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC), tm)
	require.Equal(t, true, v.Equal(x))
}

func TestObject_TypeName(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, "int", o.TypeName())

	o = core.FloatValue(0)
	require.Equal(t, "float", o.TypeName())

	o = core.CharValue(0)
	require.Equal(t, "char", o.TypeName())

	o = alloc.NewStringValue("")
	require.Equal(t, "string", o.TypeName())

	o = core.BoolValue(false)
	require.Equal(t, "bool", o.TypeName())

	o = alloc.NewArrayValue(nil, false)
	require.Equal(t, "array", o.TypeName())

	o = alloc.NewRecordValue(nil, false)
	require.Equal(t, "record", o.TypeName())

	o = alloc.NewBuiltinFunctionValue("fn", nil, 0, false)
	require.Equal(t, "<builtin-function:fn/0>", o.TypeName())

	o = core.UndefinedValue()
	require.Equal(t, "undefined", o.TypeName())

	o = alloc.NewErrorValue(core.UndefinedValue())
	require.Equal(t, "error", o.TypeName())

	o = alloc.NewBytesValue(nil)
	require.Equal(t, "bytes", o.TypeName())
}

func TestObject_IsTrue(t *testing.T) {
	var o core.Value

	// 0 is false, non-zero is true
	o = core.IntValue(0)
	require.False(t, o.IsTrue())
	o = core.IntValue(1)
	require.True(t, o.IsTrue())
	o = core.IntValue(123)
	require.True(t, o.IsTrue())
	o = core.IntValue(-456)
	require.True(t, o.IsTrue())

	// NaN is false, non-NaN is true
	o = core.FloatValue(0)
	require.True(t, o.IsTrue())
	o = core.FloatValue(1)
	require.True(t, o.IsTrue())

	// non-zero char is true
	o = core.CharValue(' ')
	require.True(t, o.IsTrue())
	o = core.CharValue('T')
	require.True(t, o.IsTrue())

	// empty string is false, non-empty string is true
	o = alloc.NewStringValue("")
	require.False(t, o.IsTrue())
	o = alloc.NewStringValue(" ")
	require.True(t, o.IsTrue())

	// empty array is false, non-empty array is true
	o = alloc.NewArrayValue(nil, false)
	require.False(t, o.IsTrue())
	o = alloc.NewArrayValue([]core.Value{core.UndefinedValue()}, false)
	require.True(t, o.IsTrue())

	// empty record is false, non-empty record is true
	o = alloc.NewRecordValue(nil, false)
	require.False(t, o.IsTrue())
	o = alloc.NewRecordValue(map[string]core.Value{"a": core.UndefinedValue()}, false)
	require.True(t, o.IsTrue())

	// undefined is false
	o = core.UndefinedValue()
	require.False(t, o.IsTrue())

	// error is false
	o = alloc.NewErrorValue(core.UndefinedValue())
	require.False(t, o.IsTrue())

	// empty bytes is false, non-empty bytes is true
	o = alloc.NewBytesValue(nil)
	require.False(t, o.IsTrue())
	o = alloc.NewBytesValue([]byte{1, 2})
	require.True(t, o.IsTrue())
}

func TestObject_String(t *testing.T) {
	var o core.Value

	o = core.IntValue(0)
	require.Equal(t, "0", o.String())

	o = core.IntValue(1)
	require.Equal(t, "1", o.String())

	o = core.FloatValue(0)
	require.Equal(t, "0", o.String())

	o = core.FloatValue(1)
	require.Equal(t, "1", o.String())

	o = core.CharValue(' ')
	require.Equal(t, "' '", o.String())

	o = core.CharValue('T')
	require.Equal(t, "'T'", o.String())

	o = alloc.NewStringValue("")
	require.Equal(t, `""`, o.String())

	o = alloc.NewStringValue(" ")
	require.Equal(t, `" "`, o.String())

	o = alloc.NewArrayValue(nil, false)
	require.Equal(t, "[]", o.String())

	o = alloc.NewRecordValue(nil, false)
	require.Equal(t, "{}", o.String())

	o = alloc.NewErrorValue(core.UndefinedValue())
	require.Equal(t, "error(undefined)", o.String())

	o = alloc.NewErrorValue(alloc.NewStringValue("error 1"))
	require.Equal(t, `error("error 1")`, o.String())

	o = core.UndefinedValue()
	require.Equal(t, "undefined", o.String())

	o = alloc.NewBytesValue(nil)
	require.Equal(t, "bytes([])", o.String())

	o = alloc.NewBytesValue([]byte("foo"))
	require.Equal(t, "bytes([102, 111, 111])", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Value

	o = core.CharValue(0)
	_, err := o.BinaryOp(alloc, token.Add, core.UndefinedValue())
	require.Error(t, err)

	o = core.BoolValue(false)
	_, err = o.BinaryOp(alloc, token.Add, core.UndefinedValue())
	require.Error(t, err)

	o = alloc.NewRecordValue(nil, false)
	_, err = o.BinaryOp(alloc, token.Add, core.UndefinedValue())
	require.Error(t, err)

	o = core.UndefinedValue()
	_, err = o.BinaryOp(alloc, token.Add, core.UndefinedValue())
	require.Error(t, err)

	o = alloc.NewErrorValue(core.UndefinedValue())
	_, err = o.BinaryOp(alloc, token.Add, core.UndefinedValue())
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, alloc.NewArrayValue(nil, false), token.Add,
		alloc.NewArrayValue(nil, false), alloc.NewArrayValue(nil, false))
	testBinaryOp(t, alloc.NewArrayValue(nil, false), token.Add,
		alloc.NewArrayValue([]core.Value{}, false), alloc.NewArrayValue(nil, false))
	testBinaryOp(t, alloc.NewArrayValue([]core.Value{}, false), token.Add,
		alloc.NewArrayValue(nil, false), alloc.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, alloc.NewArrayValue([]core.Value{}, false), token.Add,
		alloc.NewArrayValue([]core.Value{}, false),
		alloc.NewArrayValue([]core.Value{}, false))
	testBinaryOp(t, alloc.NewArrayValue(nil, false), token.Add,
		alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false), alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
		}, false))
	testBinaryOp(t, alloc.NewArrayValue(nil, false), token.Add,
		alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false), alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, alloc.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, alloc.NewArrayValue(nil, false),
		alloc.NewArrayValue([]core.Value{
			core.IntValue(1),
			core.IntValue(2),
			core.IntValue(3),
		}, false))
	testBinaryOp(t, alloc.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
	}, false), token.Add, alloc.NewArrayValue([]core.Value{
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false), alloc.NewArrayValue([]core.Value{
		core.IntValue(1),
		core.IntValue(2),
		core.IntValue(3),
		core.IntValue(4),
		core.IntValue(5),
		core.IntValue(6),
	}, false))
}

func TestError_Equals(t *testing.T) {
	err1 := alloc.NewErrorValue(alloc.NewStringValue("some error"))
	err2 := err1
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = alloc.NewErrorValue(alloc.NewStringValue("some error"))
	require.True(t, err1.Equal(err2))
	require.True(t, err2.Equal(err1))

	err2 = alloc.NewErrorValue(alloc.NewStringValue("some error 2"))
	require.False(t, err1.Equal(err2))
	require.False(t, err2.Equal(err1))

	bool1 := core.BoolValue(true)
	bool2 := core.BoolValue(true)
	bool3 := core.BoolValue(false)

	char1 := core.CharValue('A')
	char2 := core.CharValue('A')
	char3 := core.CharValue('B')

	int1 := core.IntValue(123)
	int2 := core.IntValue(123)
	int3 := core.IntValue(456)

	float1 := core.FloatValue(3.14)
	float2 := core.FloatValue(3.14)
	float3 := core.FloatValue(2.71828)

	string1 := alloc.NewStringValue("hello")
	string2 := alloc.NewStringValue("hello")
	string3 := alloc.NewStringValue("world")

	bytes1 := alloc.NewBytesValue([]byte("foo"))
	bytes2 := alloc.NewBytesValue([]byte("foo"))
	bytes3 := alloc.NewBytesValue([]byte("bar"))

	array1 := alloc.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array2 := alloc.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(2)}, false)
	array3 := alloc.NewArrayValue([]core.Value{core.IntValue(1), core.IntValue(3)}, false)

	map1 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map2 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	map3 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	record1 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record2 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(1)}, false)
	record3 := alloc.NewRecordValue(map[string]core.Value{"a": core.IntValue(2)}, false)

	// compare to undefined
	require.False(t, bool1.Equal(core.UndefinedValue()))
	require.False(t, char1.Equal(core.UndefinedValue()))
	require.False(t, int1.Equal(core.UndefinedValue()))
	require.False(t, float1.Equal(core.UndefinedValue()))
	require.False(t, string1.Equal(core.UndefinedValue()))
	require.False(t, bytes1.Equal(core.UndefinedValue()))
	require.False(t, array1.Equal(core.UndefinedValue()))
	require.False(t, map1.Equal(core.UndefinedValue()))
	require.False(t, record1.Equal(core.UndefinedValue()))

	// compare to equal
	require.True(t, bool1.Equal(bool2))
	require.True(t, char1.Equal(char2))
	require.True(t, int1.Equal(int2))
	require.True(t, float1.Equal(float2))
	require.True(t, string1.Equal(string2))
	require.True(t, bytes1.Equal(bytes2))
	require.True(t, array1.Equal(array2))
	require.True(t, map1.Equal(map2))
	require.True(t, record1.Equal(record2))

	// compare to not equal
	require.False(t, bool1.Equal(bool3))
	require.False(t, char1.Equal(char3))
	require.False(t, int1.Equal(int3))
	require.False(t, float1.Equal(float3))
	require.False(t, string1.Equal(string3))
	require.False(t, bytes1.Equal(bytes3))
	require.False(t, array1.Equal(array3))
	require.False(t, map1.Equal(map3))
	require.False(t, record1.Equal(record3))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Add,
				core.FloatValue(r), core.FloatValue(l+r))
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Sub,
				core.FloatValue(r), core.FloatValue(l-r))
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Mul,
				core.FloatValue(r), core.FloatValue(l*r))
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, core.FloatValue(l), token.Quo,
					core.FloatValue(r), core.FloatValue(l/r))
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Less,
				core.FloatValue(r), boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.Greater,
				core.FloatValue(r), boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.LessEq,
				core.FloatValue(r), boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, core.FloatValue(l), token.GreaterEq,
				core.FloatValue(r), boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Add,
				core.IntValue(r), core.FloatValue(l+float64(r)))
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Sub,
				core.IntValue(r), core.FloatValue(l-float64(r)))
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Mul,
				core.IntValue(r), core.FloatValue(l*float64(r)))
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, core.FloatValue(l), token.Quo,
					core.IntValue(r),
					core.FloatValue(l/float64(r)))
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Less,
				core.IntValue(r), boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.Greater,
				core.IntValue(r), boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.LessEq,
				core.IntValue(r), boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.FloatValue(l), token.GreaterEq,
				core.IntValue(r), boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Add,
				core.IntValue(r), core.IntValue(l+r))
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Sub,
				core.IntValue(r), core.IntValue(l-r))
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Mul,
				core.IntValue(r), core.IntValue(l*r))
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, core.IntValue(l), token.Quo,
					core.IntValue(r), core.IntValue(l/r))
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, core.IntValue(l), token.Rem,
					core.IntValue(r), core.IntValue(l%r))
			}
		}
	}

	// int & int
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(0),
		core.IntValue(int64(1)&int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(1),
		core.IntValue(int64(0)&int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(1),
		core.IntValue(int64(1)))
	testBinaryOp(t,
		core.IntValue(0), token.And, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)&int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.And, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)&int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1984), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)&int64(0xffffffff)))
	testBinaryOp(t, core.IntValue(-1984), token.And,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)&int64(0xffffffff)))

	// int | int
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(0),
		core.IntValue(int64(1)|int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(1),
		core.IntValue(int64(0)|int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(1),
		core.IntValue(int64(1)))
	testBinaryOp(t,
		core.IntValue(0), token.Or, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.Or, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1984), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)|int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.Or,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)|int64(0xffffffff)))

	// int ^ int
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(0),
		core.IntValue(int64(1)^int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(1),
		core.IntValue(int64(0)^int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(1),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.Xor, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.Xor, core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1984), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.Xor,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)^int64(0xffffffff)))

	// int &^ int
	testBinaryOp(t,
		core.IntValue(0), token.AndNot, core.IntValue(0),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot, core.IntValue(0),
		core.IntValue(int64(1)&^int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.AndNot,
		core.IntValue(1), core.IntValue(int64(0)&^int64(1)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot, core.IntValue(1),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(0), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(1), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(int64(0xffffffff)), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(0)))
	testBinaryOp(t,
		core.IntValue(1984), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(1984)&^int64(0xffffffff)))
	testBinaryOp(t,
		core.IntValue(-1984), token.AndNot,
		core.IntValue(int64(0xffffffff)),
		core.IntValue(int64(-1984)&^int64(0xffffffff)))

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			core.IntValue(0), token.Shl, core.IntValue(s),
			core.IntValue(int64(0)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(1), token.Shl, core.IntValue(s),
			core.IntValue(int64(1)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(2), token.Shl, core.IntValue(s),
			core.IntValue(int64(2)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(-1), token.Shl, core.IntValue(s),
			core.IntValue(int64(-1)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(-2), token.Shl, core.IntValue(s),
			core.IntValue(int64(-2)<<uint(s)))
		testBinaryOp(t,
			core.IntValue(int64(0xffffffff)), token.Shl,
			core.IntValue(s),
			core.IntValue(int64(0xffffffff)<<uint(s)))
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			core.IntValue(0), token.Shr, core.IntValue(s),
			core.IntValue(int64(0)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(1), token.Shr, core.IntValue(s),
			core.IntValue(int64(1)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(2), token.Shr, core.IntValue(s),
			core.IntValue(int64(2)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(-1), token.Shr, core.IntValue(s),
			core.IntValue(int64(-1)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(-2), token.Shr, core.IntValue(s),
			core.IntValue(int64(-2)>>uint(s)))
		testBinaryOp(t,
			core.IntValue(int64(0xffffffff)), token.Shr,
			core.IntValue(s),
			core.IntValue(int64(0xffffffff)>>uint(s)))
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Less,
				core.IntValue(r), boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.Greater,
				core.IntValue(r), boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.LessEq,
				core.IntValue(r), boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, core.IntValue(l), token.GreaterEq,
				core.IntValue(r), boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Add,
				core.FloatValue(r),
				core.FloatValue(float64(l)+r))
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Sub,
				core.FloatValue(r),
				core.FloatValue(float64(l)-r))
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Mul,
				core.FloatValue(r),
				core.FloatValue(float64(l)*r))
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, core.IntValue(l), token.Quo,
					core.FloatValue(r),
					core.FloatValue(float64(l)/r))
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Less,
				core.FloatValue(r), boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.Greater,
				core.FloatValue(r), boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.LessEq,
				core.FloatValue(r), boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, core.IntValue(l), token.GreaterEq,
				core.FloatValue(r), boolValue(float64(l) >= r))
		}
	}
}

func TestRecord_Index(t *testing.T) {
	m := alloc.NewRecordValue(make(map[string]core.Value), false)
	k := core.IntValue(1)
	v := alloc.NewStringValue("abcdef")
	err := m.Assign(k, v)

	require.NoError(t, err)

	res, err := m.Access(vm, k, core.OpIndex)
	require.NoError(t, err)
	require.Equal(t, v, res)
}

func TestString_BinaryOp(t *testing.T) {
	lstr := "abcde"
	rstr := "01234"
	for l := 0; l < len(lstr); l++ {
		for r := 0; r < len(rstr); r++ {
			ls := lstr[l:]
			rs := rstr[r:]
			testBinaryOp(t, alloc.NewStringValue(ls), token.Add,
				alloc.NewStringValue(rs),
				alloc.NewStringValue(ls+rs))

			rc := []rune(rstr)[r]
			testBinaryOp(t, alloc.NewStringValue(ls), token.Add,
				core.CharValue(rc),
				alloc.NewStringValue(ls+string(rc)))
		}
	}
}

func testBinaryOp(t *testing.T, lhs core.Value, op token.Token, rhs core.Value, expected core.Value) {
	t.Helper()
	actual, err := lhs.BinaryOp(alloc, op, rhs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func boolValue(b bool) core.Value {
	return core.BoolValue(b)
}
