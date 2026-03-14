package value

import (
	"fmt"
	"strings"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type ImmutableArray struct {
	ObjectImpl
	Value []core.Object
}

func (o *ImmutableArray) TypeName() string {
	return "immutable-array"
}

func (o *ImmutableArray) String() string {
	var elements []string
	for _, e := range o.Value {
		elements = append(elements, e.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
}

func (o *ImmutableArray) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	if rhs, ok := rhs.(*ImmutableArray); ok {
		switch op {
		case token.Add:
			return &Array{Value: append(o.Value, rhs.Value...)}, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *ImmutableArray) Copy() core.Object {
	var c []core.Object
	for _, elem := range o.Value {
		c = append(c, elem.Copy())
	}
	return &Array{Value: c}
}

func (o *ImmutableArray) IsFalsy() bool {
	return len(o.Value) == 0
}

func (o *ImmutableArray) Equals(x core.Object) bool {
	var xVal []core.Object
	switch x := x.(type) {
	case *Array:
		xVal = x.Value
	case *ImmutableArray:
		xVal = x.Value
	default:
		return false
	}
	if len(o.Value) != len(xVal) {
		return false
	}
	for i, e := range o.Value {
		if !e.Equals(xVal[i]) {
			return false
		}
	}
	return true
}

func (o *ImmutableArray) IndexGet(index core.Object) (res core.Object, err error) {
	intIdx, ok := index.(*Int)
	if !ok {
		err = gse.ErrInvalidIndexType
		return
	}
	idxVal := int(intIdx.Value)
	if idxVal < 0 || idxVal >= len(o.Value) {
		res = UndefinedValue
		return
	}
	res = o.Value[idxVal]
	return
}

func (o *ImmutableArray) Iterate() core.Iterator {
	return &ArrayIterator{
		v: o.Value,
		l: len(o.Value),
	}
}

func (o *ImmutableArray) CanIterate() bool {
	return true
}

func (o *ImmutableArray) ToString() (string, bool) {
	return o.String(), true
}

func (o *ImmutableArray) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *ImmutableArray) ToInterface() any {
	res := make([]any, len(o.Value))
	for i, val := range o.Value {
		res[i] = val.ToInterface()
	}
	return res
}
