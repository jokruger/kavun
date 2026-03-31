package value

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/token"
)

type BuiltinFunction struct {
	Object
	value    core.NativeFunc
	name     string
	arity    int // number of positional arguments, or minimum number of arguments if variadic is true
	variadic bool
}

// Should be used only for static initialization. For dynamic creation of built-in functions, use Allocator.NewBuiltinFunction.
func NewStaticBuiltinFunction(name string, val core.NativeFunc, arity int, variadic bool) *BuiltinFunction {
	o := &BuiltinFunction{}
	o.Set(name, val, arity, variadic)
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
	if o.variadic {
		return fmt.Sprintf("<builtin-function:%s/%d+>", o.name, o.arity)
	}
	return fmt.Sprintf("<builtin-function:%s/%d>", o.name, o.arity)
}

func (o *BuiltinFunction) String() string {
	return o.TypeName()
}

func (o *BuiltinFunction) Interface() any {
	return o.value
}

func (o *BuiltinFunction) Arity() int {
	return o.arity
}

func (o *BuiltinFunction) BinaryOp(vm core.VM, op token.Token, rhs core.Object) (core.Object, error) {
	return nil, core.NewInvalidBinaryOperatorError(op.String(), o, rhs)
}

func (o *BuiltinFunction) Copy(alloc core.Allocator) core.Object {
	return alloc.NewBuiltinFunction(o.name, o.value, o.arity, o.variadic)
}

func (o *BuiltinFunction) Access(core.VM, core.Object, core.Opcode) (core.Object, error) {
	return nil, core.NewNotAccessibleError(o)
}

func (o *BuiltinFunction) Assign(core.Object, core.Object) error {
	return core.NewNotAssignableError(o)
}

func (o *BuiltinFunction) Call(vm core.VM, args ...core.Object) (core.Object, error) {
	if o.value == nil {
		return nil, core.NewLogicError(fmt.Sprintf("built-in function %s is referencing nil", o.name))
	}
	return o.value(vm, args...)
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

func (o *BuiltinFunction) IsBuiltinFunction() bool {
	return true
}
