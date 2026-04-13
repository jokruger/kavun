package require

import (
	"fmt"
	"time"

	"github.com/jokruger/gs/core"
)

func FromInterface(alloc core.Allocator, v any) (core.Value, error) {
	switch v := v.(type) {
	case nil:
		return core.Undefined, nil

	case string:
		return alloc.NewStringValue(v), nil

	case int64:
		return core.IntValue(v), nil

	case int:
		return core.IntValue(int64(v)), nil

	case bool:
		return core.BoolValue(v), nil

	case rune:
		return core.CharValue(v), nil

	case byte:
		return core.CharValue(rune(v)), nil

	case float64:
		return core.FloatValue(v), nil

	case []byte:
		return alloc.NewBytesValue(v), nil

	case error:
		return alloc.NewErrorValue(alloc.NewStringValue(v.Error())), nil

	case map[string]core.Value:
		return alloc.NewRecordValue(v, false), nil

	case map[string]any:
		kv := make(map[string]core.Value)
		for vk, vv := range v {
			vo, err := FromInterface(alloc, vv)
			if err != nil {
				return core.Undefined, err
			}
			kv[vk] = vo
		}
		return alloc.NewRecordValue(kv, false), nil

	case []core.Value:
		return alloc.NewArrayValue(v, false), nil

	case []any:
		arr := make([]core.Value, len(v))
		for i, e := range v {
			vo, err := FromInterface(alloc, e)
			if err != nil {
				return core.Undefined, err
			}
			arr[i] = vo
		}
		return alloc.NewArrayValue(arr, false), nil

	case time.Time:
		return alloc.NewTimeValue(v), nil

	case core.Value:
		return v, nil

	case core.NativeFunc:
		return alloc.NewBuiltinFunctionValue("anonymous", v, 0, true), nil
	}

	return core.Undefined, fmt.Errorf("cannot convert to object: %T", v)
}
