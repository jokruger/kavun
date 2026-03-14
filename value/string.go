package value

import (
	"strconv"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type String struct {
	ObjectImpl
	Value   string
	runeStr []rune
}

func (o *String) TypeName() string {
	return "string"
}

func (o *String) String() string {
	return strconv.Quote(o.Value)
}

func (o *String) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *String:
			if len(o.Value)+len(rhs.Value) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			return &String{Value: o.Value + rhs.Value}, nil
		default:
			rhsStr := rhs.String()
			if len(o.Value)+len(rhsStr) > core.MaxStringLen {
				return nil, gse.ErrStringLimit
			}
			return &String{Value: o.Value + rhsStr}, nil
		}
	case token.Less:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value < rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.LessEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value <= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.Greater:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value > rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case token.GreaterEq:
		switch rhs := rhs.(type) {
		case *String:
			if o.Value >= rhs.Value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *String) IsFalsy() bool {
	return len(o.Value) == 0
}

func (o *String) Copy() core.Object {
	return &String{Value: o.Value}
}

func (o *String) Equals(x core.Object) bool {
	t, ok := x.(*String)
	if !ok {
		return false
	}
	return o.Value == t.Value
}

func (o *String) IndexGet(index core.Object) (res core.Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = gse.ErrInvalidIndexType
		return
	}
	idxVal := int(intIdx.Value)
	if o.runeStr == nil {
		o.runeStr = []rune(o.Value)
	}
	if idxVal < 0 || idxVal >= len(o.runeStr) {
		res = UndefinedValue
		return
	}
	res = &Char{Value: o.runeStr[idxVal]}
	return
}

func (o *String) Iterate() core.Iterator {
	if o.runeStr == nil {
		o.runeStr = []rune(o.Value)
	}
	return &StringIterator{
		v: o.runeStr,
		l: len(o.runeStr),
	}
}

func (o *String) CanIterate() bool {
	return true
}

func (o *String) ToString() (string, bool) {
	return o.Value, true
}

func (o *String) ToInt() (int, bool) {
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		return int(i), true
	}
	return 0, false
}

func (o *String) ToInt64() (int64, bool) {
	i, err := strconv.ParseInt(o.Value, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

func (o *String) ToFloat64() (float64, bool) {
	f, err := strconv.ParseFloat(o.Value, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

func (o *String) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *String) ToByteSlice() ([]byte, bool) {
	return []byte(o.Value), true
}

func (o *String) ToInterface() any {
	return o.Value
}
