package require

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
	gse "github.com/jokruger/gs/error"
	"github.com/jokruger/gs/value"
)

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
