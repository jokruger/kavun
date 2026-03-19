package value

import (
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type Undefined struct {
}

func NewUndefined() *Undefined {
	return &Undefined{}
}

func (o *Undefined) GobDecode(b []byte) error {
	if len(b) != 0 {
		core.NewDecodeBinarySizeError(o, 0, len(b))
	}
	return nil
}

func (o *Undefined) GobEncode() ([]byte, error) {
	return []byte{}, nil
}

func (o *Undefined) Next() bool {
	return false
}

func (o *Undefined) Key() core.Object {
	return o
}

func (o *Undefined) Value() core.Object {
	return o
}

func (o *Undefined) TypeName() string {
	return "undefined"
}

func (o *Undefined) String() string {
	return "<undefined>"
}

func (o *Undefined) Interface() any {
	return nil
}

func (o *Undefined) Arity() int {
	return 0
}

func (o *Undefined) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *Undefined) Equals(x core.Object) bool {
	return o == x
}

func (o *Undefined) Copy() core.Object {
	return o
}

func (o *Undefined) Access(core.Object, core.Opcode) (core.Object, error) {
	return UndefinedValue, nil
}

func (o *Undefined) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *Undefined) Iterate() core.Iterator {
	return o
}

func (o *Undefined) Call(core.VM, ...core.Object) (core.Object, error) {
	return nil, nil
}

func (o *Undefined) IsFalsy() bool {
	return true
}

func (o *Undefined) IsIterable() bool {
	return true
}

func (o *Undefined) IsCallable() bool {
	return false
}

func (o *Undefined) IsImmutable() bool {
	return false
}

func (o *Undefined) IsVariadic() bool {
	return false
}

func (o *Undefined) AsString() (string, bool) {
	return "", false
}

func (o *Undefined) AsInt() (int64, bool) {
	return 0, false
}

func (o *Undefined) AsFloat() (float64, bool) {
	return 0, false
}

func (o *Undefined) AsBool() (bool, bool) {
	return false, true
}

func (o *Undefined) AsRune() (rune, bool) {
	return 0, false
}

func (o *Undefined) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *Undefined) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
