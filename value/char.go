package value

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Char struct {
	value rune
}

func NewChar(v rune) *Char {
	o := &Char{}
	o.Set(v)
	return o
}

func (o *Char) GobDecode(b []byte) error {
	if len(b) != 4 {
		return core.NewDecodeBinarySizeError(o, 4, len(b))
	}
	o.Set(rune(int32(binary.BigEndian.Uint32(b))))
	return nil
}

func (o *Char) GobEncode() ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(int32(o.value)))
	return b, nil
}

func (o *Char) Set(v rune) {
	o.value = v
}

func (o *Char) Value() rune {
	return o.value
}

func (o *Char) TypeName() string {
	return "char"
}

func (o *Char) String() string {
	return fmt.Sprintf("%q", o.value)
}

func (o *Char) Interface() any {
	return o.value
}

func (o *Char) Arity() int {
	return 0
}

func (o *Char) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	switch rhs := rhs.(type) {
	case *Char:
		switch op {
		case token.Add:
			r := o.value + rhs.value
			if r == o.value {
				return o, nil
			}
			return NewChar(r), nil
		case token.Sub:
			r := o.value - rhs.value
			if r == o.value {
				return o, nil
			}
			return NewChar(r), nil
		case token.Less:
			if o.value < rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if o.value > rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if o.value <= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if o.value >= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	case *Int:
		switch op {
		case token.Add:
			r := o.value + rune(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewChar(r), nil
		case token.Sub:
			r := o.value - rune(rhs.value)
			if r == o.value {
				return o, nil
			}
			return NewChar(r), nil
		case token.Less:
			if int64(o.value) < rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.Greater:
			if int64(o.value) > rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.LessEq:
			if int64(o.value) <= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		case token.GreaterEq:
			if int64(o.value) >= rhs.value {
				return TrueValue, nil
			}
			return FalseValue, nil
		}
	}
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Char) Equals(x core.Object) bool {
	t, ok := x.AsRune()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Char) Copy() core.Object {
	return NewChar(o.value)
}

func (o *Char) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *Char) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Char) Iterate() core.Iterator {
	return nil
}

func (o *Char) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Char) IsFalsy() bool {
	return o.value == 0
}

func (o *Char) IsIterable() bool {
	return false
}

func (o *Char) IsCallable() bool {
	return false
}

func (o *Char) IsImmutable() bool {
	return true
}

func (o *Char) IsVariadic() bool {
	return false
}

func (o *Char) AsString() (string, bool) {
	return string(o.value), true
}

func (o *Char) AsInt() (int64, bool) {
	return int64(o.value), true
}

func (o *Char) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Char) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *Char) AsRune() (rune, bool) {
	return o.value, true
}

func (o *Char) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Char) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
