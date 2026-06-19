package core

import (
	"fmt"

	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/internal/binary"
)

type Dict struct {
	Elements map[string]Value
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

func DictInterface(a *Arena, v Value) any {
	o := a.ResolveDictValue(v)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface(a)
	}
	return res
}

func DictEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveDictValue(v)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON(a)
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
		idx++
	}
	b = append(b, '}')
	return b, nil
}

func DictEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveDictValue(v)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for key, value := range o.Elements {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := value.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func DictIsTrue(a *Arena, v Value) bool {
	return len(a.ResolveDictValue(v).Elements) > 0
}

func DictEqual(a *Arena, v Value, r Value) bool {
	switch r.Type {
	case value.Dict, value.Record:
		l := a.ResolveDictValue(v).Elements
		r := a.ResolveDictValue(r).Elements
		if len(l) != len(r) {
			return false
		}
		for k, le := range l {
			re, ok := r[k]
			if !ok {
				return false
			}
			if !le.Equal(a, re) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func DictLen(a *Arena, v Value) int64 {
	o := a.ResolveDictValue(v)
	return int64(len(o.Elements))
}

func DictAssign(a *Arena, v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName(a))
	}

	k, ok := index.AsString(a)
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName(a))
	}

	r.Pin(a) // §5: container takes pinned ownership of the value.
	a.ResolveDictValue(v).Elements[k] = r

	return nil
}

func DictContains(a *Arena, v Value, e Value) bool {
	s, ok := e.AsString(a)
	if !ok {
		return false
	}
	_, ok = a.ResolveDictValue(v).Elements[s]
	return ok
}

func DictDelete(a *Arena, v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName(a))
	}

	s, ok := key.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName(a))
	}
	delete(a.ResolveDictValue(v).Elements, s)
	v.Retain(a)
	return v, nil
}

func DictAsBool(a *Arena, v Value) (bool, bool) {
	return len(a.ResolveDictValue(v).Elements) > 0, true
}

func DictAsString(a *Arena, v Value) (string, bool) {
	return v.String(a), true
}

func DictAsDict(a *Arena, v Value) (map[string]Value, bool) {
	return a.ResolveDictValue(v).Elements, true
}
