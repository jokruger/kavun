package core

import (
	"fmt"
	"strings"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/opcode"
)

const (
	recordTypeName          = "record"
	immutableRecordTypeName = "immutable-record"
)

var TypeRecord = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinRecordValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainRecordValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseRecordValue(v) },
	Name:         SeqNameHook(recordTypeName, immutableRecordTypeName),
	String:       recordTypeString,
	Format:       recordTypeFormat,
	Interface:    DictInterface,
	EncodeJSON:   DictEncodeJSON,
	EncodeBinary: DictEncodeBinary,
	DecodeBinary: recordDecodeBinary,
	IsTrue:       DictIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     func(a *Arena, v Value) (Value, error) { return a.NewDictIteratorValue(a.ResolveDictValue(v).Elements) },
	Equal:        DictEqual,
	Clone:        recordTypeClone,
	Len:          DictLen,
	MethodCall:   recordTypeMethodCall,
	Access:       recordTypeAccess,
	Assign:       DictAssign,
	Contains:     DictContains,
	Delete:       DictDelete,
	AsBool:       DictAsBool,
	AsString:     DictAsString,
	AsDict:       DictAsDict,
}

func recordTypeString(a *Arena, v Value) string {
	o := a.ResolveDictValue(v)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String(a)))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordDecodeBinary(a *Arena, v *Value, data []byte) error {
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
	o := a.ResolveDictValue(v)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone(a)
		if err != nil {
			return Undefined, err
		}
		t.Pin(a)
		c[k] = t
	}
	return a.NewRecordValue(c, false)
}

func recordTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := a.ResolveDictValue(v)
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
	o := a.ResolveDictValue(v)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}
