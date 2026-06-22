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
		return a.NewBytesValue(v, false)

	case rune:
		return core.RuneValue(v), nil

	case []rune:
		return a.NewRunesValue(v, false)

	case int:
		return core.IntValue(int64(v)), nil

	case int64:
		return core.IntValue(v), nil

	case float64:
		return core.FloatValue(v), nil

	case string:
		return a.NewStringValue(v)

	case dec128.Dec128:
		return a.NewDecimalValue(v)

	case time.Time:
		return a.NewTimeValue(v)

	case error:
		nv, err := a.NewStringValue(v.Error())
		if err != nil {
			return core.Undefined, err
		}
		a.PinAny(nv)
		return a.NewErrorValue(nv, core.KindUser, false)

	case []string:
		arr := make([]core.Value, len(v))
		for i, e := range v {
			nv, err := ValueOf(e)
			if err != nil {
				return core.Undefined, err
			}
			a.PinAny(nv)
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
			a.PinAny(nv)
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
			a.PinAny(nv)
			kv[vk] = nv
		}
		return a.NewRecordValue(kv, false)

	case core.NativeFunc:
		return a.NewBuiltinClosureValue("anonymous", v, 0, true)

	case core.Value:
		return v, nil

	case []core.Value:
		for _, e := range v {
			a.PinAny(e)
		}
		return a.NewArrayValue(v, false)

	case map[string]core.Value:
		for _, vv := range v {
			a.PinAny(vv)
		}
		return a.NewRecordValue(v, false)

	default:
		return core.Undefined, fmt.Errorf("cannot convert to object: %T", v)
	}
}
