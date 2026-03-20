package value

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type BuiltinFunction struct {
	value    core.NativeFunc
	name     string
	arity    int // number of positional arguments, or minimum number of arguments if variadic is true
	variadic bool
}

func NewBuiltinFunction(name string, value core.NativeFunc, arity int, variadic bool) *BuiltinFunction {
	o := &BuiltinFunction{}
	o.Set(name, value, arity, variadic)
	return o
}

func (o *BuiltinFunction) GobDecode(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	var name string
	if err := dec.Decode(&name); err != nil {
		return err
	}

	var arity int
	if err := dec.Decode(&arity); err != nil {
		return err
	}

	var variadic bool
	if err := dec.Decode(&variadic); err != nil {
		return err
	}

	o.Set(name, nil, arity, variadic)
	return nil
}

func (o *BuiltinFunction) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(o.name); err != nil {
		return nil, err
	}
	if err := enc.Encode(o.arity); err != nil {
		return nil, err
	}
	if err := enc.Encode(o.variadic); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (o *BuiltinFunction) Set(name string, value core.NativeFunc, arity int, variadic bool) {
	o.name = name
	o.value = value
	o.arity = arity
	o.variadic = variadic
}

func (o *BuiltinFunction) Name() string {
	return o.name
}

func (o *BuiltinFunction) TypeName() string {
	return "builtin-function:" + o.name
}

func (o *BuiltinFunction) String() string {
	return "<builtin-function>"
}

func (o *BuiltinFunction) Interface() any {
	return o.value
}

func (o *BuiltinFunction) Arity() int {
	return o.arity
}

func (o *BuiltinFunction) BinaryOp(op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *BuiltinFunction) Equals(x core.Object) bool {
	return o == x
}

func (o *BuiltinFunction) Copy() core.Object {
	return NewBuiltinFunction(o.name, o.value, o.arity, o.variadic)
}

func (o *BuiltinFunction) Access(core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *BuiltinFunction) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *BuiltinFunction) Iterate() core.Iterator {
	return nil
}

func (o *BuiltinFunction) Call(vm core.VM, args ...core.Object) (core.Object, error) {
	if o.value == nil {
		return nil, core.NewLogicError(fmt.Sprintf("built-in function %s is referencing nil", o.name))
	}

	if !o.variadic && len(args) != o.arity {
		return nil, core.NewWrongNumArgumentsError("builtin function '"+o.name+"'", fmt.Sprintf("%d", o.arity), len(args))
	}

	return o.value(args...)
}

func (o *BuiltinFunction) IsFalsy() bool {
	return o == nil
}

func (o *BuiltinFunction) IsIterable() bool {
	return false
}

func (o *BuiltinFunction) IsCallable() bool {
	return true
}

func (o *BuiltinFunction) IsImmutable() bool {
	return true
}

func (o *BuiltinFunction) IsVariadic() bool {
	return o.variadic
}

func (o *BuiltinFunction) AsString() (string, bool) {
	return "", false
}

func (o *BuiltinFunction) AsInt() (int64, bool) {
	return 0, false
}

func (o *BuiltinFunction) AsFloat() (float64, bool) {
	return 0, false
}

func (o *BuiltinFunction) AsBool() (bool, bool) {
	return false, false
}

func (o *BuiltinFunction) AsRune() (rune, bool) {
	return 0, false
}

func (o *BuiltinFunction) AsByteSlice() ([]byte, bool) {
	return nil, false
}

func (o *BuiltinFunction) AsTime() (time.Time, bool) {
	return time.Time{}, false
}
