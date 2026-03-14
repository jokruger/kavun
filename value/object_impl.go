package value

import (
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type ObjectImpl struct {
}

func (o *ObjectImpl) TypeName() string {
	panic(gse.ErrNotImplemented)
}

func (o *ObjectImpl) String() string {
	panic(gse.ErrNotImplemented)
}

func (o *ObjectImpl) BinaryOp(token.Token, core.Object) (core.Object, error) {
	return nil, gse.ErrInvalidOperator
}

func (o *ObjectImpl) Copy() core.Object {
	return nil
}

func (o *ObjectImpl) IsFalsy() bool {
	return false
}

func (o *ObjectImpl) Equals(x core.Object) bool {
	return o == x
}

func (o *ObjectImpl) IndexGet(core.Object) (core.Object, error) {
	return nil, gse.ErrNotIndexable
}

func (o *ObjectImpl) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *ObjectImpl) Iterate() core.Iterator {
	return nil
}

func (o *ObjectImpl) CanIterate() bool {
	return false
}

func (o *ObjectImpl) Call(...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *ObjectImpl) CanCall() bool {
	return false
}

func (o *ObjectImpl) ToString() (string, bool) {
	return "", false
}

func (o *ObjectImpl) ToInt() (int, bool) {
	return 0, false
}

func (o *ObjectImpl) ToInt64() (int64, bool) {
	return 0, false
}

func (o *ObjectImpl) ToFloat64() (float64, bool) {
	return 0, false
}

func (o *ObjectImpl) ToBool() (bool, bool) {
	return false, false
}

func (o *ObjectImpl) ToRune() (rune, bool) {
	return 0, false
}

func (o *ObjectImpl) ToByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *ObjectImpl) ToTime() (time.Time, bool) {
	return time.Time{}, false
}

func (o *ObjectImpl) ToInterface() any {
	return o
}
