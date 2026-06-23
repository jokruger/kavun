package core

import (
	"fmt"
	"strings"
	"unsafe"

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

func NewRecordValue(m map[string]Value, immutable bool) Value {
	o := &Record{Elements: m}
	return Value{Type: value.Record, Immutable: immutable, Ptr: unsafe.Pointer(o)}
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

func recordTypeString(v Value) string {
	o := (*Record)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeInterface(v Value) any {
	o := (*Record)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func recordTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)
	var b []byte
	b = append(b, '{')
	len1 := len(o.Elements) - 1
	idx := 0
	for key, value := range o.Elements {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := value.EncodeJSON()
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

func recordTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for key, value := range o.Elements {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := value.EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func recordTypeDecodeBinary(v *Value, data []byte) error {
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
		if err := element.DecodeBinary(eb); err != nil {
			return fmt.Errorf("record value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("record: trailing %d bytes", len(data)-offset)
	}

	*v = NewRecordValue(value, v.Immutable)

	return nil
}

func recordTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return recordTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(recordTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(recordTypeString(v), sp, fspec.AlignLeft), nil
}

func recordTypeClone(v Value) (Value, error) {
	// Deep copy the record (and make it mutable) and its elements
	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone()
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewRecordValue(c, false), nil
}

func recordTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Record)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(v Value, index Value, mode opcode.Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	o := (*Record)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}

func recordTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Record)(v.Ptr).Elements), nil
}

func recordTypeIsTrue(v Value) bool {
	return len((*Record)(v.Ptr).Elements) > 0
}

func recordTypeEqual(v Value, rv Value) bool {
	var r map[string]Value
	switch rv.Type {
	case value.Dict:
		r = (*Dict)(rv.Ptr).Elements
	case value.Record:
		r = (*Record)(rv.Ptr).Elements
	default:
		return false
	}

	l := (*Record)(v.Ptr).Elements
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
}

func recordTypeLen(v Value) int64 {
	o := (*Record)(v.Ptr)
	return int64(len(o.Elements))
}

func recordTypeAssign(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}

	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}

	(*Record)(v.Ptr).Elements[k] = r

	return nil
}

func recordTypeContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Record)(v.Ptr).Elements[s]
	return ok
}

func recordTypeDelete(v Value, key Value) (Value, error) {
	if v.Immutable {
		return Undefined, errs.NewNotDeletableError(v.TypeName())
	}

	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}
	delete((*Record)(v.Ptr).Elements, s)
	return v, nil
}

func recordTypeAsBool(v Value) (bool, bool) {
	return len((*Record)(v.Ptr).Elements) > 0, true
}

func recordTypeAsString(v Value) (string, bool) {
	return v.String(), true
}

func recordTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Record)(v.Ptr).Elements, true
}
