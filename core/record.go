package core

import (
	"fmt"
	"strings"

	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
)

const (
	recordTypeName          = "record"
	immutableRecordTypeName = "immutable-record"
)

type Record struct {
	Elements map[string]Value
}

func (o *Record) Set(elements map[string]Value) {
	o.Elements = elements
}

var TypeRecord = ValueTypeDescr{
	Name:         SeqNameHook(recordTypeName, immutableRecordTypeName),
	String:       recordTypeString,
	Format:       recordTypeFormat,
	Interface:    recordTypeInterface,
	EncodeJSON:   recordTypeEncodeJSON,
	EncodeBinary: recordTypeEncodeBinary,
	DecodeBinary: recordTypeDecodeBinary,
	IsTrue:       recordTypeIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     recordTypeIterator,
	Equal:        recordTypeEqual,
	Clone:        recordTypeClone,
	Len:          recordTypeLen,
	MethodCall:   recordTypeMethodCall,
	Access:       recordTypeAccess,
	Assign:       recordTypeAssign,
	Contains:     recordTypeContains,
	Delete:       recordTypeDelete,
	AsBool:       recordTypeAsBool,
	AsString:     recordTypeAsString,
	AsDict:       recordTypeAsDict,
}

func recordTypeString(a *Arena, v Value) string {
	o := a.ResolveRecordValue(v)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String(a)))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeInterface(a *Arena, v Value) any {
	o := a.ResolveRecordValue(v)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface(a)
	}
	return res
}

func recordTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveRecordValue(v)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON(a)
		if err != nil {
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
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

func recordTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveRecordValue(v)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for key, value := range o.Elements {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := value.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func recordTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "record (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("record key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("record value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(a, eb); err != nil {
			return fmt.Errorf("record value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("record: trailing %d bytes", len(data)-offset)
	}

	o, err := a.NewRecordValue(value, v.Immutable)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o

	return nil
}

func recordTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return recordTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(a), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(recordTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(recordTypeString(a, v), sp, fspec.AlignLeft), nil
}

func recordTypeClone(a *Arena, v Value) (Value, error) {
	// Deep copy the record (and make it mutable) and its elements
	o := a.ResolveRecordValue(v)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone(a)
		if err != nil {
			return Undefined, err
		}
		a.PinAny(t)
		c[k] = t
	}
	return a.NewRecordValue(c, false)
}

func recordTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := a.ResolveRecordValue(v)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
	if !e.IsCallable(a) {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(a), name, e.TypeName(a))
	}
	return e.Call(a, vm, args)
}

func recordTypeAccess(a *Arena, v Value, index Value, mode opcode.Opcode) (Value, error) {
	k, ok := index.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName(a))
	}
	o := a.ResolveRecordValue(v)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

func recordTypeIterator(a *Arena, v Value) (Value, error) {
	return a.NewDictIteratorValue(a.ResolveRecordValue(v).Elements)
}

func recordTypeIsTrue(a *Arena, v Value) bool {
	return len(a.ResolveRecordValue(v).Elements) > 0
}

func recordTypeEqual(a *Arena, v Value, rv Value) bool {
	var r map[string]Value
	switch rv.Type {
	case value.Dict:
		r = a.ResolveDictValue(rv).Elements
	case value.Record:
		r = a.ResolveRecordValue(rv).Elements
	default:
		return false
	}

	l := a.ResolveRecordValue(v).Elements
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
}

func recordTypeLen(a *Arena, v Value) int64 {
	o := a.ResolveRecordValue(v)
	return int64(len(o.Elements))
}

func recordTypeAssign(a *Arena, v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName(a))
	}

	k, ok := index.AsString(a)
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName(a))
	}

	a.PinAny(r) // §5: container takes pinned ownership of the value.
	a.ResolveRecordValue(v).Elements[k] = r

	return nil
}

func recordTypeContains(a *Arena, v Value, e Value) bool {
	s, ok := e.AsString(a)
	if !ok {
		return false
	}
	_, ok = a.ResolveRecordValue(v).Elements[s]
	return ok
}

func recordTypeDelete(a *Arena, v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName(a))
	}

	s, ok := key.AsString(a)
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName(a))
	}
	delete(a.ResolveRecordValue(v).Elements, s)
	return v, nil
}

func recordTypeAsBool(a *Arena, v Value) (bool, bool) {
	return len(a.ResolveRecordValue(v).Elements) > 0, true
}

func recordTypeAsString(a *Arena, v Value) (string, bool) {
	return v.String(a), true
}

func recordTypeAsDict(a *Arena, v Value) (map[string]Value, bool) {
	return a.ResolveRecordValue(v).Elements, true
}
