package gs

import (
	"errors"

	"github.com/jokruger/gs/core"
	"github.com/jokruger/gs/value"
)

// Variable is a user-defined variable for the script.
type Variable struct {
	name  string
	value core.Object
}

// NewVariable creates a Variable.
func NewVariable(name string, value any) (*Variable, error) {
	obj, err := FromInterface(value)
	if err != nil {
		return nil, err
	}
	return &Variable{
		name:  name,
		value: obj,
	}, nil
}

// Name returns the name of the variable.
func (v *Variable) Name() string {
	return v.name
}

// Value returns an empty interface of the variable value.
func (v *Variable) Value() any {
	return ToInterface(v.value)
}

// ValueType returns the name of the value type.
func (v *Variable) ValueType() string {
	return v.value.TypeName()
}

// Int returns int value of the variable value.
// It returns 0 if the value is not convertible to int.
func (v *Variable) Int() int {
	c, _ := v.value.ToInt()
	return c
}

// Int64 returns int64 value of the variable value. It returns 0 if the value
// is not convertible to int64.
func (v *Variable) Int64() int64 {
	c, _ := v.value.ToInt64()
	return c
}

// Float returns float64 value of the variable value. It returns 0.0 if the
// value is not convertible to float64.
func (v *Variable) Float() float64 {
	c, _ := v.value.ToFloat64()
	return c
}

// Char returns rune value of the variable value. It returns 0 if the value is
// not convertible to rune.
func (v *Variable) Char() rune {
	c, _ := v.value.ToRune()
	return c
}

// Bool returns bool value of the variable value. It returns 0 if the value is not convertible to bool.
func (v *Variable) Bool() bool {
	c, _ := v.value.ToBool()
	return c
}

// Array returns []interface value of the variable value. It returns 0 if the value is not convertible to []interface.
func (v *Variable) Array() []any {
	switch val := v.value.(type) {
	case *value.Array:
		var arr []any
		for _, e := range val.Value {
			arr = append(arr, ToInterface(e))
		}
		return arr
	}
	return nil
}

// Map returns map[string]any value of the variable value. It returns 0 if the value is not convertible to map[string]any.
func (v *Variable) Map() map[string]any {
	switch val := v.value.(type) {
	case *value.Map:
		kv := make(map[string]any)
		for mk, mv := range val.Value {
			kv[mk] = ToInterface(mv)
		}
		return kv
	}
	return nil
}

// String returns string value of the variable value. It returns 0 if the value
// is not convertible to string.
func (v *Variable) String() string {
	c, _ := v.value.ToString()
	return c
}

// Bytes returns a byte slice of the variable value. It returns nil if the
// value is not convertible to byte slice.
func (v *Variable) Bytes() []byte {
	c, _ := v.value.ToByteSlice()
	return c
}

// Error returns an error if the underlying value is error object. If not,
// this returns nil.
func (v *Variable) Error() error {
	err, ok := v.value.(*value.Error)
	if ok {
		return errors.New(err.String())
	}
	return nil
}

// Object returns an underlying Object of the variable value. Note that
// returned Object is a copy of an actual Object used in the script.
func (v *Variable) Object() core.Object {
	return v.value
}

// IsUndefined returns true if the underlying value is undefined.
func (v *Variable) IsUndefined() bool {
	return v.value == value.UndefinedValue
}
