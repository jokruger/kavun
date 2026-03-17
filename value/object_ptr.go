package value

import (
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/token"
)

type ObjectPtr struct {
	value *core.Object
}

func NewObjectPtr(value *core.Object) *ObjectPtr {
	o := &ObjectPtr{}
	o.Set(value)
	return o
}

func (o *ObjectPtr) Set(value *core.Object) {
	o.value = value
}

func (o *ObjectPtr) Value() *core.Object {
	return o.value
}

func (o *ObjectPtr) TypeName() string {
	return "<free-var>"
}

func (o *ObjectPtr) String() string {
	return "free-var"
}

func (o *ObjectPtr) Interface() any {
	return o.value
}

func (o *ObjectPtr) Arity() int {
	return 0
}

func (o *ObjectPtr) BinaryOp(token.Token, core.Object) (core.Object, error) {
	return nil, gse.ErrInvalidOperator
}

func (o *ObjectPtr) Equals(x core.Object) bool {
	return o == x
}

func (o *ObjectPtr) Copy() core.Object {
	return o
}

func (o *ObjectPtr) IndexGet(core.Object) (core.Object, error) {
	return nil, gse.ErrNotIndexable
}

func (o *ObjectPtr) IndexSet(core.Object, core.Object) error {
	return gse.ErrNotIndexAssignable
}

func (o *ObjectPtr) Iterate() core.Iterator {
	return nil
}

func (o *ObjectPtr) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *ObjectPtr) IsFalsy() bool {
	return o.value == nil
}

func (o *ObjectPtr) IsIterable() bool {
	return false
}

func (o *ObjectPtr) IsCallable() bool {
	return false
}

func (o *ObjectPtr) IsImmutable() bool {
	return false
}

func (o *ObjectPtr) IsVariadic() bool {
	return false
}

func (o *ObjectPtr) AsString() (string, bool) {
	return "", false
}

func (o *ObjectPtr) AsInt() (int64, bool) {
	return 0, false
}

func (o *ObjectPtr) AsFloat() (float64, bool) {
	return 0, false
}

func (o *ObjectPtr) AsBool() (bool, bool) {
	return !o.IsFalsy(), true
}

func (o *ObjectPtr) AsRune() (rune, bool) {
	return 0, false
}

func (o *ObjectPtr) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *ObjectPtr) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
