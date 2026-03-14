package value

import (
	"bytes"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Bytes struct {
	ObjectImpl
	Value []byte
}

func (o *Bytes) String() string {
	return string(o.Value)
}

func (o *Bytes) TypeName() string {
	return "bytes"
}

func (o *Bytes) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch op {
	case token.Add:
		switch rhs := rhs.(type) {
		case *Bytes:
			if len(o.Value)+len(rhs.Value) > core.MaxBytesLen {
				return nil, gse.ErrBytesLimit
			}
			return &Bytes{Value: append(o.Value, rhs.Value...)}, nil
		}
	}
	return nil, gse.ErrInvalidOperator
}

func (o *Bytes) Copy() core.Object {
	return &Bytes{Value: append([]byte{}, o.Value...)}
}

func (o *Bytes) IsFalsy() bool {
	return len(o.Value) == 0
}

func (o *Bytes) Equals(x core.Object) bool {
	t, ok := x.(*Bytes)
	if !ok {
		return false
	}
	return bytes.Equal(o.Value, t.Value)
}

func (o *Bytes) IndexGet(index core.Object) (res core.Object, err error) {
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
	res = &Int{Value: int64(o.Value[idxVal])}
	return
}

func (o *Bytes) Iterate() core.Iterator {
	return &BytesIterator{
		v: o.Value,
		l: len(o.Value),
	}
}

func (o *Bytes) CanIterate() bool {
	return true
}

func (o *Bytes) ToString() (string, bool) {
	return o.String(), true
}

func (o *Bytes) ToBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Bytes) ToByteSlice() ([]byte, bool) {
	return o.Value, true
}

func (o *Bytes) ToInterface() any {
	return o.Value
}
