package value_test

import (
	"testing"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/parser"
	"github.com/jokruger/gs/tests/require"
	"github.com/jokruger/gs/token"
	"github.com/jokruger/gs/value"
	"github.com/jokruger/gs/vm"
)

func TestObject_TypeName(t *testing.T) {
	var o core.Object = value.NewInt(0)
	require.Equal(t, "int", o.TypeName())
	o = value.NewFloat(0)
	require.Equal(t, "float", o.TypeName())
	o = value.NewChar(0)
	require.Equal(t, "char", o.TypeName())
	o = value.NewString("")
	require.Equal(t, "string", o.TypeName())
	o = value.NewBool(false)
	require.Equal(t, "bool", o.TypeName())
	o = value.NewArray(nil, false)
	require.Equal(t, "array", o.TypeName())
	o = value.NewRecord(nil, false)
	require.Equal(t, "record", o.TypeName())
	o = value.NewArrayIterator(nil)
	require.Equal(t, "array-iterator", o.TypeName())
	o = value.NewStringIterator(nil)
	require.Equal(t, "string-iterator", o.TypeName())
	o = value.NewMapIterator(nil)
	require.Equal(t, "map-iterator", o.TypeName())
	o = value.NewBuiltinFunction("fn", nil, 0, false)
	require.Equal(t, "builtin-function:fn", o.TypeName())
	o = &vm.CompiledFunction{}
	require.Equal(t, "compiled-function", o.TypeName())
	o = value.UndefinedValue
	require.Equal(t, "undefined", o.TypeName())
	o = value.NewError(nil)
	require.Equal(t, "error", o.TypeName())
	o = value.NewBytes(nil)
	require.Equal(t, "bytes", o.TypeName())
}

func TestObject_IsFalsy(t *testing.T) {
	var o core.Object = value.NewInt(0)
	require.True(t, o.IsFalsy())
	o = value.NewInt(1)
	require.False(t, o.IsFalsy())
	o = value.NewFloat(0)
	require.False(t, o.IsFalsy())
	o = value.NewFloat(1)
	require.False(t, o.IsFalsy())
	o = value.NewChar(' ')
	require.False(t, o.IsFalsy())
	o = value.NewChar('T')
	require.False(t, o.IsFalsy())
	o = value.NewString("")
	require.True(t, o.IsFalsy())
	o = value.NewString(" ")
	require.False(t, o.IsFalsy())
	o = value.NewArray(nil, false)
	require.True(t, o.IsFalsy())
	o = value.NewArray([]core.Object{nil}, false) // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = value.NewRecord(nil, false)
	require.True(t, o.IsFalsy())
	o = value.NewRecord(map[string]core.Object{"a": nil}, false) // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = value.NewStringIterator(nil)
	require.True(t, o.IsFalsy())
	o = value.NewArrayIterator(nil)
	require.True(t, o.IsFalsy())
	o = value.NewMapIterator(nil)
	require.True(t, o.IsFalsy())
	o = value.NewBuiltinFunction("fn", nil, 0, false)
	require.False(t, o.IsFalsy())
	o = &vm.CompiledFunction{}
	require.False(t, o.IsFalsy())
	o = value.UndefinedValue
	require.True(t, o.IsFalsy())
	o = value.NewError(nil)
	require.True(t, o.IsFalsy())
	o = value.NewBytes(nil)
	require.True(t, o.IsFalsy())
	o = value.NewBytes([]byte{1, 2})
	require.False(t, o.IsFalsy())
}

func TestObject_String(t *testing.T) {
	var o core.Object = value.NewInt(0)
	require.Equal(t, "0", o.String())
	o = value.NewInt(1)
	require.Equal(t, "1", o.String())
	o = value.NewFloat(0)
	require.Equal(t, "0", o.String())
	o = value.NewFloat(1)
	require.Equal(t, "1", o.String())
	o = value.NewChar(' ')
	require.Equal(t, "' '", o.String())
	o = value.NewChar('T')
	require.Equal(t, "'T'", o.String())
	o = value.NewString("")
	require.Equal(t, `""`, o.String())
	o = value.NewString(" ")
	require.Equal(t, `" "`, o.String())
	o = value.NewArray(nil, false)
	require.Equal(t, "[]", o.String())
	o = value.NewRecord(nil, false)
	require.Equal(t, "{}", o.String())
	o = value.NewError(nil)
	require.Equal(t, "error(undefined)", o.String())
	o = value.NewError(value.NewString("error 1"))
	require.Equal(t, `error("error 1")`, o.String())
	o = value.NewStringIterator(nil)
	require.Equal(t, "<string-iterator>", o.String())
	o = value.NewArrayIterator(nil)
	require.Equal(t, "<array-iterator>", o.String())
	o = value.NewMapIterator(nil)
	require.Equal(t, "<map-iterator>", o.String())
	o = value.UndefinedValue
	require.Equal(t, "undefined", o.String())
	o = value.NewBytes(nil)
	require.Equal(t, "", o.String())
	o = value.NewBytes([]byte("foo"))
	require.Equal(t, "foo", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Object = value.NewChar(0)
	_, err := o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewBool(false)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewRecord(nil, false)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewArrayIterator(nil)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewStringIterator(nil)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewMapIterator(nil)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewBuiltinFunction("fn", nil, 0, false)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &vm.CompiledFunction{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.UndefinedValue
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = value.NewError(nil)
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, value.NewArray(nil, false), token.Add,
		value.NewArray(nil, false), value.NewArray(nil, false))
	testBinaryOp(t, value.NewArray(nil, false), token.Add,
		value.NewArray([]core.Object{}, false), value.NewArray(nil, false))
	testBinaryOp(t, value.NewArray([]core.Object{}, false), token.Add,
		value.NewArray(nil, false), value.NewArray([]core.Object{}, false))
	testBinaryOp(t, value.NewArray([]core.Object{}, false), token.Add,
		value.NewArray([]core.Object{}, false),
		value.NewArray([]core.Object{}, false))
	testBinaryOp(t, value.NewArray(nil, false), token.Add,
		value.NewArray([]core.Object{
			value.NewInt(1),
		}, false), value.NewArray([]core.Object{
			value.NewInt(1),
		}, false))
	testBinaryOp(t, value.NewArray(nil, false), token.Add,
		value.NewArray([]core.Object{
			value.NewInt(1),
			value.NewInt(2),
			value.NewInt(3),
		}, false), value.NewArray([]core.Object{
			value.NewInt(1),
			value.NewInt(2),
			value.NewInt(3),
		}, false))
	testBinaryOp(t, value.NewArray([]core.Object{
		value.NewInt(1),
		value.NewInt(2),
		value.NewInt(3),
	}, false), token.Add, value.NewArray(nil, false),
		value.NewArray([]core.Object{
			value.NewInt(1),
			value.NewInt(2),
			value.NewInt(3),
		}, false))
	testBinaryOp(t, value.NewArray([]core.Object{
		value.NewInt(1),
		value.NewInt(2),
		value.NewInt(3),
	}, false), token.Add, value.NewArray([]core.Object{
		value.NewInt(4),
		value.NewInt(5),
		value.NewInt(6),
	}, false), value.NewArray([]core.Object{
		value.NewInt(1),
		value.NewInt(2),
		value.NewInt(3),
		value.NewInt(4),
		value.NewInt(5),
		value.NewInt(6),
	}, false))
}

func TestError_Equals(t *testing.T) {
	err1 := value.NewError(value.NewString("some error"))
	err2 := err1
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = value.NewError(value.NewString("some error"))
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = value.NewError(value.NewString("some error 2"))
	require.False(t, err1.Equals(err2))
	require.False(t, err2.Equals(err1))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.Add,
				value.NewFloat(r), value.NewFloat(l+r))
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.Sub,
				value.NewFloat(r), value.NewFloat(l-r))
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.Mul,
				value.NewFloat(r), value.NewFloat(l*r))
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, value.NewFloat(l), token.Quo,
					value.NewFloat(r), value.NewFloat(l/r))
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.Less,
				value.NewFloat(r), boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.Greater,
				value.NewFloat(r), boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.LessEq,
				value.NewFloat(r), boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, value.NewFloat(l), token.GreaterEq,
				value.NewFloat(r), boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.Add,
				value.NewInt(r), value.NewFloat(l+float64(r)))
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.Sub,
				value.NewInt(r), value.NewFloat(l-float64(r)))
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.Mul,
				value.NewInt(r), value.NewFloat(l*float64(r)))
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, value.NewFloat(l), token.Quo,
					value.NewInt(r),
					value.NewFloat(l/float64(r)))
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.Less,
				value.NewInt(r), boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.Greater,
				value.NewInt(r), boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.LessEq,
				value.NewInt(r), boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewFloat(l), token.GreaterEq,
				value.NewInt(r), boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.Add,
				value.NewInt(r), value.NewInt(l+r))
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.Sub,
				value.NewInt(r), value.NewInt(l-r))
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.Mul,
				value.NewInt(r), value.NewInt(l*r))
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, value.NewInt(l), token.Quo,
					value.NewInt(r), value.NewInt(l/r))
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, value.NewInt(l), token.Rem,
					value.NewInt(r), value.NewInt(l%r))
			}
		}
	}

	// int & int
	testBinaryOp(t,
		value.NewInt(0), token.And, value.NewInt(0),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1), token.And, value.NewInt(0),
		value.NewInt(int64(1)&int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.And, value.NewInt(1),
		value.NewInt(int64(0)&int64(1)))
	testBinaryOp(t,
		value.NewInt(1), token.And, value.NewInt(1),
		value.NewInt(int64(1)))
	testBinaryOp(t,
		value.NewInt(0), token.And, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)&int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1), token.And, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1)&int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(int64(0xffffffff)), token.And,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1984), token.And,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1984)&int64(0xffffffff)))
	testBinaryOp(t, value.NewInt(-1984), token.And,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(-1984)&int64(0xffffffff)))

	// int | int
	testBinaryOp(t,
		value.NewInt(0), token.Or, value.NewInt(0),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1), token.Or, value.NewInt(0),
		value.NewInt(int64(1)|int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.Or, value.NewInt(1),
		value.NewInt(int64(0)|int64(1)))
	testBinaryOp(t,
		value.NewInt(1), token.Or, value.NewInt(1),
		value.NewInt(int64(1)))
	testBinaryOp(t,
		value.NewInt(0), token.Or, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)|int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1), token.Or, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1)|int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(int64(0xffffffff)), token.Or,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1984), token.Or,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1984)|int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(-1984), token.Or,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(-1984)|int64(0xffffffff)))

	// int ^ int
	testBinaryOp(t,
		value.NewInt(0), token.Xor, value.NewInt(0),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1), token.Xor, value.NewInt(0),
		value.NewInt(int64(1)^int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.Xor, value.NewInt(1),
		value.NewInt(int64(0)^int64(1)))
	testBinaryOp(t,
		value.NewInt(1), token.Xor, value.NewInt(1),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.Xor, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1), token.Xor, value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1)^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(int64(0xffffffff)), token.Xor,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1984), token.Xor,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1984)^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(-1984), token.Xor,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(-1984)^int64(0xffffffff)))

	// int &^ int
	testBinaryOp(t,
		value.NewInt(0), token.AndNot, value.NewInt(0),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1), token.AndNot, value.NewInt(0),
		value.NewInt(int64(1)&^int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.AndNot,
		value.NewInt(1), value.NewInt(int64(0)&^int64(1)))
	testBinaryOp(t,
		value.NewInt(1), token.AndNot, value.NewInt(1),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(0), token.AndNot,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)&^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(1), token.AndNot,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1)&^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(int64(0xffffffff)), token.AndNot,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(0)))
	testBinaryOp(t,
		value.NewInt(1984), token.AndNot,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(1984)&^int64(0xffffffff)))
	testBinaryOp(t,
		value.NewInt(-1984), token.AndNot,
		value.NewInt(int64(0xffffffff)),
		value.NewInt(int64(-1984)&^int64(0xffffffff)))

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			value.NewInt(0), token.Shl, value.NewInt(s),
			value.NewInt(int64(0)<<uint(s)))
		testBinaryOp(t,
			value.NewInt(1), token.Shl, value.NewInt(s),
			value.NewInt(int64(1)<<uint(s)))
		testBinaryOp(t,
			value.NewInt(2), token.Shl, value.NewInt(s),
			value.NewInt(int64(2)<<uint(s)))
		testBinaryOp(t,
			value.NewInt(-1), token.Shl, value.NewInt(s),
			value.NewInt(int64(-1)<<uint(s)))
		testBinaryOp(t,
			value.NewInt(-2), token.Shl, value.NewInt(s),
			value.NewInt(int64(-2)<<uint(s)))
		testBinaryOp(t,
			value.NewInt(int64(0xffffffff)), token.Shl,
			value.NewInt(s),
			value.NewInt(int64(0xffffffff)<<uint(s)))
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			value.NewInt(0), token.Shr, value.NewInt(s),
			value.NewInt(int64(0)>>uint(s)))
		testBinaryOp(t,
			value.NewInt(1), token.Shr, value.NewInt(s),
			value.NewInt(int64(1)>>uint(s)))
		testBinaryOp(t,
			value.NewInt(2), token.Shr, value.NewInt(s),
			value.NewInt(int64(2)>>uint(s)))
		testBinaryOp(t,
			value.NewInt(-1), token.Shr, value.NewInt(s),
			value.NewInt(int64(-1)>>uint(s)))
		testBinaryOp(t,
			value.NewInt(-2), token.Shr, value.NewInt(s),
			value.NewInt(int64(-2)>>uint(s)))
		testBinaryOp(t,
			value.NewInt(int64(0xffffffff)), token.Shr,
			value.NewInt(s),
			value.NewInt(int64(0xffffffff)>>uint(s)))
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.Less,
				value.NewInt(r), boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.Greater,
				value.NewInt(r), boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.LessEq,
				value.NewInt(r), boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, value.NewInt(l), token.GreaterEq,
				value.NewInt(r), boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.Add,
				value.NewFloat(r),
				value.NewFloat(float64(l)+r))
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.Sub,
				value.NewFloat(r),
				value.NewFloat(float64(l)-r))
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.Mul,
				value.NewFloat(r),
				value.NewFloat(float64(l)*r))
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, value.NewInt(l), token.Quo,
					value.NewFloat(r),
					value.NewFloat(float64(l)/r))
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.Less,
				value.NewFloat(r), boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.Greater,
				value.NewFloat(r), boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.LessEq,
				value.NewFloat(r), boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, value.NewInt(l), token.GreaterEq,
				value.NewFloat(r), boolValue(float64(l) >= r))
		}
	}
}

func TestRecord_Index(t *testing.T) {
	m := value.NewRecord(make(map[string]core.Object), false)
	k := value.NewInt(1)
	v := value.NewString("abcdef")
	err := m.Assign(k, v)

	require.NoError(t, err)

	res, err := m.Access(k, parser.OpIndex)
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
			testBinaryOp(t, value.NewString(ls), token.Add,
				value.NewString(rs),
				value.NewString(ls+rs))

			rc := []rune(rstr)[r]
			testBinaryOp(t, value.NewString(ls), token.Add,
				value.NewChar(rc),
				value.NewString(ls+string(rc)))
		}
	}
}

func testBinaryOp(t *testing.T, lhs core.Object, op token.Token, rhs core.Object, expected core.Object) {
	t.Helper()
	actual, err := lhs.BinaryOp(op, rhs)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func boolValue(b bool) core.Object {
	if b {
		return value.TrueValue
	}
	return value.FalseValue
}
