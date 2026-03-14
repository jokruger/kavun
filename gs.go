package gs

import (
	"errors"
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

const (
	// GlobalsSize is the maximum number of global variables for a VM.
	GlobalsSize = 1024

	// StackSize is the maximum stack size for a VM.
	StackSize = 2048

	// MaxFrames is the maximum number of function frames for a VM.
	MaxFrames = 1024

	// SourceFileExtDefault is the default extension for source files.
	SourceFileExtDefault = ".gs"
)

// CountObjects returns the number of objects that a given object o contains.
// For scalar value types, it will always be 1. For compound value types,
// this will include its elements and all of their elements recursively.
func CountObjects(o core.Object) (c int) {
	c = 1
	switch o := o.(type) {
	case *value.Array:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.ImmutableArray:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.Map:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.ImmutableMap:
		for _, v := range o.Value {
			c += CountObjects(v)
		}
	case *value.Error:
		c += CountObjects(o.Value)
	}
	return
}

// ToInterface attempts to convert an object o to an any value
func ToInterface(o core.Object) (res any) {
	switch o := o.(type) {
	case *value.Int:
		res = o.Value
	case *value.String:
		res = o.Value
	case *value.Float:
		res = o.Value
	case *value.Bool:
		res = o == value.TrueValue
	case *value.Char:
		res = o.Value
	case *value.Bytes:
		res = o.Value
	case *value.Array:
		res = make([]any, len(o.Value))
		for i, val := range o.Value {
			res.([]any)[i] = ToInterface(val)
		}
	case *value.ImmutableArray:
		res = make([]any, len(o.Value))
		for i, val := range o.Value {
			res.([]any)[i] = ToInterface(val)
		}
	case *value.Map:
		res = make(map[string]any)
		for key, v := range o.Value {
			res.(map[string]any)[key] = ToInterface(v)
		}
	case *value.ImmutableMap:
		res = make(map[string]any)
		for key, v := range o.Value {
			res.(map[string]any)[key] = ToInterface(v)
		}
	case *value.Time:
		res = o.Value
	case *value.Error:
		res = errors.New(o.String())
	case *value.Undefined:
		res = nil
	case core.Object:
		return o
	}
	return
}

// FromInterface will attempt to convert an any v to a Gs Object
func FromInterface(v any) (core.Object, error) {
	switch v := v.(type) {
	case nil:
		return value.UndefinedValue, nil
	case string:
		if len(v) > core.MaxStringLen {
			return nil, gse.ErrStringLimit
		}
		return &value.String{Value: v}, nil
	case int64:
		return &value.Int{Value: v}, nil
	case int:
		return &value.Int{Value: int64(v)}, nil
	case bool:
		if v {
			return value.TrueValue, nil
		}
		return value.FalseValue, nil
	case rune:
		return &value.Char{Value: v}, nil
	case byte:
		return &value.Char{Value: rune(v)}, nil
	case float64:
		return &value.Float{Value: v}, nil
	case []byte:
		if len(v) > core.MaxBytesLen {
			return nil, gse.ErrBytesLimit
		}
		return &value.Bytes{Value: v}, nil
	case error:
		return &value.Error{Value: &value.String{Value: v.Error()}}, nil
	case map[string]core.Object:
		return &value.Map{Value: v}, nil
	case map[string]any:
		kv := make(map[string]core.Object)
		for vk, vv := range v {
			vo, err := FromInterface(vv)
			if err != nil {
				return nil, err
			}
			kv[vk] = vo
		}
		return &value.Map{Value: kv}, nil
	case []core.Object:
		return &value.Array{Value: v}, nil
	case []any:
		arr := make([]core.Object, len(v))
		for i, e := range v {
			vo, err := FromInterface(e)
			if err != nil {
				return nil, err
			}
			arr[i] = vo
		}
		return &value.Array{Value: arr}, nil
	case time.Time:
		return &value.Time{Value: v}, nil
	case core.Object:
		return v, nil
	case core.CallableFunction:
		return &value.UserFunction{Value: v}, nil
	}
	return nil, fmt.Errorf("cannot convert to object: %T", v)
}
