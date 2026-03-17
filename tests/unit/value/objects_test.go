package value_test

/*
func TestObject_TypeName(t *testing.T) {
	var o core.Object = &value.Int{}
	require.Equal(t, "int", o.TypeName())
	o = &value.Float{}
	require.Equal(t, "float", o.TypeName())
	o = &value.Char{}
	require.Equal(t, "char", o.TypeName())
	o = &value.String{}
	require.Equal(t, "string", o.TypeName())
	o = &value.Bool{}
	require.Equal(t, "bool", o.TypeName())
	o = value.NewArray(nil, false)
	require.Equal(t, "array", o.TypeName())
	o = &value.Map{}
	require.Equal(t, "map", o.TypeName())
	o = &value.ArrayIterator{}
	require.Equal(t, "array-iterator", o.TypeName())
	o = &value.StringIterator{}
	require.Equal(t, "string-iterator", o.TypeName())
	o = &value.MapIterator{}
	require.Equal(t, "map-iterator", o.TypeName())
	o = &value.BuiltinFunction{Name: "fn"}
	require.Equal(t, "builtin-function:fn", o.TypeName())
	o = &value.CompiledFunction{}
	require.Equal(t, "compiled-function", o.TypeName())
	o = &value.Undefined{}
	require.Equal(t, "undefined", o.TypeName())
	o = &value.Error{}
	require.Equal(t, "error", o.TypeName())
	o = &value.Bytes{}
	require.Equal(t, "bytes", o.TypeName())
}

func TestObject_IsFalsy(t *testing.T) {
	var o core.Object = &value.Int{Value: 0}
	require.True(t, o.IsFalsy())
	o = &value.Int{Value: 1}
	require.False(t, o.IsFalsy())
	o = &value.Float{Value: 0}
	require.False(t, o.IsFalsy())
	o = &value.Float{Value: 1}
	require.False(t, o.IsFalsy())
	o = &value.Char{Value: ' '}
	require.False(t, o.IsFalsy())
	o = &value.Char{Value: 'T'}
	require.False(t, o.IsFalsy())
	o = &value.String{Value: ""}
	require.True(t, o.IsFalsy())
	o = &value.String{Value: " "}
	require.False(t, o.IsFalsy())
	o = &value.Array{Value: nil}
	require.True(t, o.IsFalsy())
	o = &value.Array{Value: []core.Object{nil}} // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = &value.Map{Value: nil}
	require.True(t, o.IsFalsy())
	o = &value.Map{Value: map[string]core.Object{"a": nil}} // nil is not valid but still count as 1 element
	require.False(t, o.IsFalsy())
	o = &value.StringIterator{}
	require.True(t, o.IsFalsy())
	o = &value.ArrayIterator{}
	require.True(t, o.IsFalsy())
	o = &value.MapIterator{}
	require.True(t, o.IsFalsy())
	o = &value.BuiltinFunction{}
	require.False(t, o.IsFalsy())
	o = &value.CompiledFunction{}
	require.False(t, o.IsFalsy())
	o = &value.Undefined{}
	require.True(t, o.IsFalsy())
	o = &value.Error{}
	require.True(t, o.IsFalsy())
	o = &value.Bytes{}
	require.True(t, o.IsFalsy())
	o = &value.Bytes{Value: []byte{1, 2}}
	require.False(t, o.IsFalsy())
}

func TestObject_String(t *testing.T) {
	var o core.Object = &value.Int{Value: 0}
	require.Equal(t, "0", o.String())
	o = &value.Int{Value: 1}
	require.Equal(t, "1", o.String())
	o = &value.Float{Value: 0}
	require.Equal(t, "0", o.String())
	o = &value.Float{Value: 1}
	require.Equal(t, "1", o.String())
	o = &value.Char{Value: ' '}
	require.Equal(t, " ", o.String())
	o = &value.Char{Value: 'T'}
	require.Equal(t, "T", o.String())
	o = &value.String{Value: ""}
	require.Equal(t, `""`, o.String())
	o = &value.String{Value: " "}
	require.Equal(t, `" "`, o.String())
	o = &value.Array{Value: nil}
	require.Equal(t, "[]", o.String())
	o = &value.Map{Value: nil}
	require.Equal(t, "{}", o.String())
	o = &value.Error{Value: nil}
	require.Equal(t, "error", o.String())
	o = &value.Error{Value: &value.String{Value: "error 1"}}
	require.Equal(t, `error: "error 1"`, o.String())
	o = &value.StringIterator{}
	require.Equal(t, "<string-iterator>", o.String())
	o = &value.ArrayIterator{}
	require.Equal(t, "<array-iterator>", o.String())
	o = &value.MapIterator{}
	require.Equal(t, "<map-iterator>", o.String())
	o = &value.Undefined{}
	require.Equal(t, "<undefined>", o.String())
	o = &value.Bytes{}
	require.Equal(t, "", o.String())
	o = &value.Bytes{Value: []byte("foo")}
	require.Equal(t, "foo", o.String())
}

func TestObject_BinaryOp(t *testing.T) {
	var o core.Object = &value.Char{}
	_, err := o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.Bool{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.Map{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.ArrayIterator{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.StringIterator{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.MapIterator{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.BuiltinFunction{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.CompiledFunction{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.Undefined{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
	o = &value.Error{}
	_, err = o.BinaryOp(token.Add, value.UndefinedValue)
	require.Error(t, err)
}

func TestArray_BinaryOp(t *testing.T) {
	testBinaryOp(t, &value.Array{Value: nil}, token.Add,
		&value.Array{Value: nil}, &value.Array{Value: nil})
	testBinaryOp(t, &value.Array{Value: nil}, token.Add,
		&value.Array{Value: []core.Object{}}, &value.Array{Value: nil})
	testBinaryOp(t, &value.Array{Value: []core.Object{}}, token.Add,
		&value.Array{Value: nil}, &value.Array{Value: []core.Object{}})
	testBinaryOp(t, &value.Array{Value: []core.Object{}}, token.Add,
		&value.Array{Value: []core.Object{}},
		&value.Array{Value: []core.Object{}})
	testBinaryOp(t, &value.Array{Value: nil}, token.Add,
		&value.Array{Value: []core.Object{
			&value.Int{Value: 1},
		}}, &value.Array{Value: []core.Object{
			&value.Int{Value: 1},
		}})
	testBinaryOp(t, &value.Array{Value: nil}, token.Add,
		&value.Array{Value: []core.Object{
			&value.Int{Value: 1},
			&value.Int{Value: 2},
			&value.Int{Value: 3},
		}}, &value.Array{Value: []core.Object{
			&value.Int{Value: 1},
			&value.Int{Value: 2},
			&value.Int{Value: 3},
		}})
	testBinaryOp(t, &value.Array{Value: []core.Object{
		&value.Int{Value: 1},
		&value.Int{Value: 2},
		&value.Int{Value: 3},
	}}, token.Add, &value.Array{Value: nil},
		&value.Array{Value: []core.Object{
			&value.Int{Value: 1},
			&value.Int{Value: 2},
			&value.Int{Value: 3},
		}})
	testBinaryOp(t, &value.Array{Value: []core.Object{
		&value.Int{Value: 1},
		&value.Int{Value: 2},
		&value.Int{Value: 3},
	}}, token.Add, &value.Array{Value: []core.Object{
		&value.Int{Value: 4},
		&value.Int{Value: 5},
		&value.Int{Value: 6},
	}}, &value.Array{Value: []core.Object{
		&value.Int{Value: 1},
		&value.Int{Value: 2},
		&value.Int{Value: 3},
		&value.Int{Value: 4},
		&value.Int{Value: 5},
		&value.Int{Value: 6},
	}})
}

func TestError_Equals(t *testing.T) {
	err1 := &value.Error{Value: &value.String{Value: "some error"}}
	err2 := err1
	require.True(t, err1.Equals(err2))
	require.True(t, err2.Equals(err1))

	err2 = &value.Error{Value: &value.String{Value: "some error"}}
	require.False(t, err1.Equals(err2))
	require.False(t, err2.Equals(err1))
}

func TestFloat_BinaryOp(t *testing.T) {
	// float + float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.Add,
				&value.Float{Value: r}, &value.Float{Value: l + r})
		}
	}

	// float - float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.Sub,
				&value.Float{Value: r}, &value.Float{Value: l - r})
		}
	}

	// float * float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.Mul,
				&value.Float{Value: r}, &value.Float{Value: l * r})
		}
	}

	// float / float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			if r != 0 {
				testBinaryOp(t, &value.Float{Value: l}, token.Quo,
					&value.Float{Value: r}, &value.Float{Value: l / r})
			}
		}
	}

	// float < float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.Less,
				&value.Float{Value: r}, boolValue(l < r))
		}
	}

	// float > float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.Greater,
				&value.Float{Value: r}, boolValue(l > r))
		}
	}

	// float <= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.LessEq,
				&value.Float{Value: r}, boolValue(l <= r))
		}
	}

	// float >= float
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := float64(-2); r <= 2.1; r += 0.4 {
			testBinaryOp(t, &value.Float{Value: l}, token.GreaterEq,
				&value.Float{Value: r}, boolValue(l >= r))
		}
	}

	// float + int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.Add,
				&value.Int{Value: r}, &value.Float{Value: l + float64(r)})
		}
	}

	// float - int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.Sub,
				&value.Int{Value: r}, &value.Float{Value: l - float64(r)})
		}
	}

	// float * int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.Mul,
				&value.Int{Value: r}, &value.Float{Value: l * float64(r)})
		}
	}

	// float / int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, &value.Float{Value: l}, token.Quo,
					&value.Int{Value: r},
					&value.Float{Value: l / float64(r)})
			}
		}
	}

	// float < int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.Less,
				&value.Int{Value: r}, boolValue(l < float64(r)))
		}
	}

	// float > int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.Greater,
				&value.Int{Value: r}, boolValue(l > float64(r)))
		}
	}

	// float <= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.LessEq,
				&value.Int{Value: r}, boolValue(l <= float64(r)))
		}
	}

	// float >= int
	for l := float64(-2); l <= 2.1; l += 0.4 {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Float{Value: l}, token.GreaterEq,
				&value.Int{Value: r}, boolValue(l >= float64(r)))
		}
	}
}

func TestInt_BinaryOp(t *testing.T) {
	// int + int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.Add,
				&value.Int{Value: r}, &value.Int{Value: l + r})
		}
	}

	// int - int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.Sub,
				&value.Int{Value: r}, &value.Int{Value: l - r})
		}
	}

	// int * int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.Mul,
				&value.Int{Value: r}, &value.Int{Value: l * r})
		}
	}

	// int / int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			if r != 0 {
				testBinaryOp(t, &value.Int{Value: l}, token.Quo,
					&value.Int{Value: r}, &value.Int{Value: l / r})
			}
		}
	}

	// int % int
	for l := int64(-4); l <= 4; l++ {
		for r := -int64(-4); r <= 4; r++ {
			if r == 0 {
				testBinaryOp(t, &value.Int{Value: l}, token.Rem,
					&value.Int{Value: r}, &value.Int{Value: l % r})
			}
		}
	}

	// int & int
	testBinaryOp(t,
		&value.Int{Value: 0}, token.And, &value.Int{Value: 0},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.And, &value.Int{Value: 0},
		&value.Int{Value: int64(1) & int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.And, &value.Int{Value: 1},
		&value.Int{Value: int64(0) & int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.And, &value.Int{Value: 1},
		&value.Int{Value: int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.And, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0) & int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.And, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1) & int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: int64(0xffffffff)}, token.And,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1984}, token.And,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1984) & int64(0xffffffff)})
	testBinaryOp(t, &value.Int{Value: -1984}, token.And,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(-1984) & int64(0xffffffff)})

	// int | int
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Or, &value.Int{Value: 0},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Or, &value.Int{Value: 0},
		&value.Int{Value: int64(1) | int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Or, &value.Int{Value: 1},
		&value.Int{Value: int64(0) | int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Or, &value.Int{Value: 1},
		&value.Int{Value: int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Or, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0) | int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Or, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1) | int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: int64(0xffffffff)}, token.Or,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1984}, token.Or,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1984) | int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: -1984}, token.Or,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(-1984) | int64(0xffffffff)})

	// int ^ int
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Xor, &value.Int{Value: 0},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Xor, &value.Int{Value: 0},
		&value.Int{Value: int64(1) ^ int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Xor, &value.Int{Value: 1},
		&value.Int{Value: int64(0) ^ int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Xor, &value.Int{Value: 1},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.Xor, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.Xor, &value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: int64(0xffffffff)}, token.Xor,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1984}, token.Xor,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1984) ^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: -1984}, token.Xor,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(-1984) ^ int64(0xffffffff)})

	// int &^ int
	testBinaryOp(t,
		&value.Int{Value: 0}, token.AndNot, &value.Int{Value: 0},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.AndNot, &value.Int{Value: 0},
		&value.Int{Value: int64(1) &^ int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.AndNot,
		&value.Int{Value: 1}, &value.Int{Value: int64(0) &^ int64(1)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.AndNot, &value.Int{Value: 1},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 0}, token.AndNot,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: 1}, token.AndNot,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: int64(0xffffffff)}, token.AndNot,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(0)})
	testBinaryOp(t,
		&value.Int{Value: 1984}, token.AndNot,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(1984) &^ int64(0xffffffff)})
	testBinaryOp(t,
		&value.Int{Value: -1984}, token.AndNot,
		&value.Int{Value: int64(0xffffffff)},
		&value.Int{Value: int64(-1984) &^ int64(0xffffffff)})

	// int << int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			&value.Int{Value: 0}, token.Shl, &value.Int{Value: s},
			&value.Int{Value: int64(0) << uint(s)})
		testBinaryOp(t,
			&value.Int{Value: 1}, token.Shl, &value.Int{Value: s},
			&value.Int{Value: int64(1) << uint(s)})
		testBinaryOp(t,
			&value.Int{Value: 2}, token.Shl, &value.Int{Value: s},
			&value.Int{Value: int64(2) << uint(s)})
		testBinaryOp(t,
			&value.Int{Value: -1}, token.Shl, &value.Int{Value: s},
			&value.Int{Value: int64(-1) << uint(s)})
		testBinaryOp(t,
			&value.Int{Value: -2}, token.Shl, &value.Int{Value: s},
			&value.Int{Value: int64(-2) << uint(s)})
		testBinaryOp(t,
			&value.Int{Value: int64(0xffffffff)}, token.Shl,
			&value.Int{Value: s},
			&value.Int{Value: int64(0xffffffff) << uint(s)})
	}

	// int >> int
	for s := int64(0); s < 64; s++ {
		testBinaryOp(t,
			&value.Int{Value: 0}, token.Shr, &value.Int{Value: s},
			&value.Int{Value: int64(0) >> uint(s)})
		testBinaryOp(t,
			&value.Int{Value: 1}, token.Shr, &value.Int{Value: s},
			&value.Int{Value: int64(1) >> uint(s)})
		testBinaryOp(t,
			&value.Int{Value: 2}, token.Shr, &value.Int{Value: s},
			&value.Int{Value: int64(2) >> uint(s)})
		testBinaryOp(t,
			&value.Int{Value: -1}, token.Shr, &value.Int{Value: s},
			&value.Int{Value: int64(-1) >> uint(s)})
		testBinaryOp(t,
			&value.Int{Value: -2}, token.Shr, &value.Int{Value: s},
			&value.Int{Value: int64(-2) >> uint(s)})
		testBinaryOp(t,
			&value.Int{Value: int64(0xffffffff)}, token.Shr,
			&value.Int{Value: s},
			&value.Int{Value: int64(0xffffffff) >> uint(s)})
	}

	// int < int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.Less,
				&value.Int{Value: r}, boolValue(l < r))
		}
	}

	// int > int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.Greater,
				&value.Int{Value: r}, boolValue(l > r))
		}
	}

	// int <= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.LessEq,
				&value.Int{Value: r}, boolValue(l <= r))
		}
	}

	// int >= int
	for l := int64(-2); l <= 2; l++ {
		for r := int64(-2); r <= 2; r++ {
			testBinaryOp(t, &value.Int{Value: l}, token.GreaterEq,
				&value.Int{Value: r}, boolValue(l >= r))
		}
	}

	// int + float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.Add,
				&value.Float{Value: r},
				&value.Float{Value: float64(l) + r})
		}
	}

	// int - float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.Sub,
				&value.Float{Value: r},
				&value.Float{Value: float64(l) - r})
		}
	}

	// int * float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.Mul,
				&value.Float{Value: r},
				&value.Float{Value: float64(l) * r})
		}
	}

	// int / float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			if r != 0 {
				testBinaryOp(t, &value.Int{Value: l}, token.Quo,
					&value.Float{Value: r},
					&value.Float{Value: float64(l) / r})
			}
		}
	}

	// int < float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.Less,
				&value.Float{Value: r}, boolValue(float64(l) < r))
		}
	}

	// int > float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.Greater,
				&value.Float{Value: r}, boolValue(float64(l) > r))
		}
	}

	// int <= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.LessEq,
				&value.Float{Value: r}, boolValue(float64(l) <= r))
		}
	}

	// int >= float
	for l := int64(-2); l <= 2; l++ {
		for r := float64(-2); r <= 2.1; r += 0.5 {
			testBinaryOp(t, &value.Int{Value: l}, token.GreaterEq,
				&value.Float{Value: r}, boolValue(float64(l) >= r))
		}
	}
}

func TestMap_Index(t *testing.T) {
	m := &value.Map{Value: make(map[string]core.Object)}
	k := &value.Int{Value: 1}
	v := &value.String{Value: "abcdef"}
	err := m.IndexSet(k, v)

	require.NoError(t, err)

	res, err := m.IndexGet(k)
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
			testBinaryOp(t, &value.String{Value: ls}, token.Add,
				&value.String{Value: rs},
				&value.String{Value: ls + rs})

			rc := []rune(rstr)[r]
			testBinaryOp(t, &value.String{Value: ls}, token.Add,
				&value.Char{Value: rc},
				&value.String{Value: ls + string(rc)})
		}
	}
}

func testBinaryOp(
	t *testing.T,
	lhs core.Object,
	op token.Token,
	rhs core.Object,
	expected core.Object,
) {
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
*/
