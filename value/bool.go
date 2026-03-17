package value

import (
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type Bool struct {
	value bool
}

func NewBool(value bool) *Bool {
	if value {
		return TrueValue
	}
	return FalseValue
}

func (o *Bool) Value() bool {
	return o.value
}

func (o *Bool) TypeName() string {
	return "bool"
}

func (o *Bool) String() string {
	if o.value {
		return TrueString
	}
	return FalseString
}

func (o *Bool) Interface() any {
	return o.value
}

func (o *Bool) Arity() int {
	return 0
}

func (o *Bool) BinaryOp(token.Token, core.Object) (core.Object, error) {
	return nil, gse.ErrInvalidOperator
}

func (o *Bool) Equals(x core.Object) bool {
	if o == x {
		return true
	}
	t, ok := x.AsBool()
	if !ok {
		return false
	}
	return o.value == t
}

func (o *Bool) Copy() core.Object {
	return NewBool(o.value)
}

func (o *Bool) IndexGet(core.Object) (core.Object, error) {
	return nil, gse.ErrNotIndexable
}

func (o *Bool) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *Bool) Iterate() core.Iterator {
	return nil
}

func (o *Bool) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Bool) IsFalsy() bool {
	return !o.value
}

func (o *Bool) IsIterable() bool {
	return false
}

func (o *Bool) IsCallable() bool {
	return false
}

func (o *Bool) IsImmutable() bool {
	return false
}

func (o *Bool) IsVariadic() bool {
	return false
}

func (o *Bool) AsString() (string, bool) {
	return o.String(), true
}

func (o *Bool) AsInt() (int64, bool) {
	if o.value {
		return 1, true
	}
	return 0, true
}

func (o *Bool) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Bool) AsBool() (bool, bool) {
	return o.value, true
}

func (o *Bool) AsRune() (rune, bool) {
	return 0, false
}

func (o *Bool) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Bool) AsTime() (time.Time, bool) {
	return time.Time{}, false
}

func (o *Bool) GobDecode(b []byte) (err error) {
	o.value = b[0] == 1
	return
}

func (o *Bool) GobEncode() (b []byte, err error) {
	if o.value {
		b = []byte{1}
	} else {
		b = []byte{0}
	}
	return
}
