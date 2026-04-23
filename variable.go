package kavun

import (
	"errors"
	"time"

	"github.com/jokruger/kavun/core"
)

// Variable is a user-defined variable for the script.
type Variable struct {
	name  string
	value core.Value
}

// NewVariable creates a Variable.
func NewVariable(name string, val core.Value) *Variable {
	return &Variable{
		name:  name,
		value: val,
	}
}

// Name returns the name of the variable.
func (v *Variable) Name() string {
	return v.name
}

// Value returns the value of the variable.
func (v *Variable) Value() core.Value {
	return v.value
}

// ValueType returns the name of the value type.
func (v *Variable) ValueType() string {
	return v.value.TypeName()
}

// Int returns int64 value of the variable value. It returns 0 if the value is not convertible to int64.
func (v *Variable) Int() int64 {
	c, _ := v.value.AsInt()
	return c
}

// Time returns time.Time value of the variable value. It returns zero time if the value is not convertible to time.Time.
func (v *Variable) Time() time.Time {
	c, _ := v.value.AsTime()
	return c
}

// Float returns float64 value of the variable value. It returns 0.0 if the
// value is not convertible to float64.
func (v *Variable) Float() float64 {
	c, _ := v.value.AsFloat()
	return c
}

// Rune returns rune value of the variable value. It returns 0 if the value is not convertible to rune.
func (v *Variable) Rune() rune {
	c, _ := v.value.AsRune()
	return c
}

// Bool returns bool value of the variable value. It returns 0 if the value is not convertible to bool.
func (v *Variable) Bool() bool {
	c, _ := v.value.AsBool()
	return c
}

// Array returns []interface value of the variable value. It returns 0 if the value is not convertible to []interface.
func (v *Variable) Array() []any {
	switch v.value.Type {
	case core.VT_ARRAY:
		val := (*core.Array)(v.value.Ptr).Elements
		arr := make([]any, 0, len(val))
		for _, e := range val {
			arr = append(arr, e.Interface())
		}
		return arr
	default:
		return nil
	}
}

// Map returns map[string]any value of the variable value. It returns 0 if the value is not convertible to map[string]any.
func (v *Variable) Map() map[string]any {
	switch v.value.Type {
	case core.VT_MAP, core.VT_RECORD:
		src := (*core.Map)(v.value.Ptr).Elements
		kv := make(map[string]any, len(src))
		for mk, mv := range src {
			kv[mk] = mv.Interface()
		}
		return kv
	default:
		return nil
	}
}

// String returns string value of the variable value. It returns 0 if the value
// is not convertible to string.
func (v *Variable) String() string {
	c, _ := v.value.AsString()
	return c
}

// Bytes returns a byte slice of the variable value. It returns nil if the
// value is not convertible to byte slice.
func (v *Variable) Bytes() []byte {
	c, _ := v.value.AsBytes()
	return c
}

// Error returns an error if the underlying value is error object. If not, this returns nil.
func (v *Variable) Error() error {
	if v.value.Type == core.VT_ERROR {
		return errors.New(v.value.String())
	}
	return nil
}

// Object returns an underlying Object of the variable value. Note that
// returned Object is a copy of an actual Object used in the script.
func (v *Variable) Object() core.Value {
	return v.value
}

// IsUndefined returns true if the underlying value is undefined.
func (v *Variable) IsUndefined() bool {
	return v.value.Type == core.VT_UNDEFINED
}
