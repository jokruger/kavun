package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/errs"
)

type Dict struct {
	Elements map[string]Value
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

func DictInterface(v Value) any {
	o := (*Dict)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func DictEncodeJSON(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON()
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

func DictEncodeBinary(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(o.Elements); err != nil {
		return nil, fmt.Errorf("dict (elements): %w", err)
	}
	return buf.Bytes(), nil
}

func DictDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var value map[string]Value
	if err := dec.Decode(&value); err != nil {
		return fmt.Errorf("dict (elements): %w", err)
	}
	if value == nil {
		value = make(map[string]Value)
	}
	o := &Dict{Elements: value}
	v.Ptr = unsafe.Pointer(o)
	return nil
}

func DictIsTrue(v Value) bool {
	return len((*Dict)(v.Ptr).Elements) > 0
}

func DictEqual(v Value, r Value) bool {
	switch r.Type {
	case VT_DICT, VT_RECORD:
		l := (*Dict)(v.Ptr).Elements
		r := (*Dict)(r.Ptr).Elements
		if len(l) != len(r) {
			return false
		}
		for k, le := range l {
			re, ok := r[k]
			if !ok {
				return false
			}
			if !le.Equal(re) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func DictLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

func DictAssign(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Dict)(v.Ptr).Elements[k] = r

	return nil
}

func DictContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Dict)(v.Ptr).Elements[s]
	return ok
}

func DictDelete(v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName())
	}

	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}
	delete((*Dict)(v.Ptr).Elements, s)
	return v, nil
}

func DictAsBool(v Value) (bool, bool) {
	return len((*Dict)(v.Ptr).Elements) > 0, true
}

func DictAsString(v Value) (string, bool) {
	return v.String(), true
}

func DictAsDict(v Value, a *Arena) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
