package kavun

import (
	"fmt"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
)

func MustValueOf(v any) core.Value {
	nv, err := ValueOf(v)
	if err != nil {
		panic(err)
	}
	return nv
}

func ValueOf(v any) (core.Value, error) {
	switch v := v.(type) {
	case nil:
		return core.Undefined, nil

	case bool:
		return core.BoolValue(v), nil

	case byte:
		return core.ByteValue(v), nil

	case []byte:
		return core.NewBytesValue(v, false), nil

	case rune:
		return core.RuneValue(v), nil

	case []rune:
		return core.NewRunesValue(v, false), nil

	case int:
		return core.IntValue(int64(v)), nil

	case int64:
		return core.IntValue(v), nil

	case float64:
		return core.FloatValue(v), nil

	case string:
		return core.NewStringValue(v), nil

	case dec128.Dec128:
		return core.NewDecimalValue(v), nil

	case time.Time:
		return core.NewTimeValue(v), nil

	case error:
		return core.NewErrorValue(core.NewStringValue(v.Error()), core.KindUser, false), nil

	case []string:
		arr := make([]core.Value, len(v))
		for i, e := range v {
			nv, err := ValueOf(e)
			if err != nil {
				return core.Undefined, err
			}
			arr[i] = nv
		}
		return core.NewArrayValue(arr, false), nil

	case []any:
		arr := make([]core.Value, len(v))
		for i, e := range v {
			nv, err := ValueOf(e)
			if err != nil {
				return core.Undefined, err
			}
			arr[i] = nv
		}
		return core.NewArrayValue(arr, false), nil

	case map[string]any:
		kv := make(map[string]core.Value)
		for vk, vv := range v {
			nv, err := ValueOf(vv)
			if err != nil {
				return core.Undefined, err
			}
			kv[vk] = nv
		}
		return core.NewRecordValue(kv, false), nil

	case core.NativeFunc:
		return core.NewBuiltinClosureValue("anonymous", v, 0, true), nil

	case core.Value:
		return v, nil

	case []core.Value:
		return core.NewArrayValue(v, false), nil

	case map[string]core.Value:
		return core.NewRecordValue(v, false), nil

	default:
		return core.Undefined, fmt.Errorf("cannot convert to object: %T", v)
	}
}
