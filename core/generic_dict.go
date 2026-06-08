package core

import (
	"fmt"

	"github.com/jokruger/kavun/errs"
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

	b := appendBinaryUint64(nil, uint64(len(o.Elements)))
	for key, value := range o.Elements {
		b = appendBinaryBytes(b, []byte(key))
		eb, err := value.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = appendBinaryBytes(b, eb)
	}
	return b, nil
}

func DictDecodeBinary(a *Arena, v *Value, data []byte) error {
	offset := 0
	count, err := readBinaryUint64(data, &offset, "dict (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := readBinaryBytes(data, &offset, fmt.Sprintf("dict key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := readBinaryBytes(data, &offset, fmt.Sprintf("dict value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(a, eb); err != nil {
			return fmt.Errorf("dict value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("dict: trailing %d bytes", len(data)-offset)
	}

	o, err := a.NewDictValue(value, v.Immutable)
	if err != nil {
		return err
	}
	*v = o
	return nil
}

func DictIsTrue(a *Arena, v Value) bool {
	return len(a.ResolveDictValue(v).Elements) > 0
}

func DictEqual(a *Arena, v Value, r Value) bool {
	switch r.Type {
	case VT_DICT, VT_RECORD:
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
