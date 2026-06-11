package test

import (
	"fmt"
	"time"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core"
)

func FromInterface(alloc *core.Arena, v any) (core.Value, error) {
	if alloc == nil {
		alloc = core.NewArena(nil)
	}

	switch v := v.(type) {
	case nil:
		return core.Undefined, nil

	case string:
		return alloc.NewStringValue(v)

	case int64:
		return core.IntValue(v), nil

	case int:
		return core.IntValue(int64(v)), nil

	case bool:
		return core.BoolValue(v), nil

	case rune:
		return core.RuneValue(v), nil

	case byte:
		return core.ByteValue(v), nil

	case float64:
		return core.FloatValue(v), nil

	case dec128.Dec128:
		return alloc.NewDecimalValue(v)

	case []byte:
		return alloc.NewBytesValue(v, false)

	case []rune:
		return alloc.NewRunesValue(v, false)

	case error:
		nv, err := alloc.NewStringValue(v.Error())
		if err != nil {
			return core.Undefined, err
		}
		return alloc.NewErrorValue(nv, core.KindUser, false)

	case map[string]core.Value:
		return alloc.NewRecordValue(v, false)

	case map[string]any:
		kv := make(map[string]core.Value)
		for vk, vv := range v {
			vo, err := FromInterface(alloc, vv)
			if err != nil {
				return core.Undefined, err
			}
			kv[vk] = vo
		}
		return alloc.NewRecordValue(kv, false)

	case []core.Value:
		return alloc.NewArrayValue(v, false)

	case []any:
		arr := make([]core.Value, len(v))
		for i, e := range v {
			vo, err := FromInterface(alloc, e)
			if err != nil {
				return core.Undefined, err
			}
			arr[i] = vo
		}
		return alloc.NewArrayValue(arr, false)

	case time.Time:
		return alloc.NewTimeValue(v)

	case core.Value:
		return v, nil

	case core.NativeFunc:
		return alloc.NewBuiltinClosureValue("anonymous", v, 0, true)
	}

	return core.Undefined, fmt.Errorf("cannot convert to object: %T", v)
}
