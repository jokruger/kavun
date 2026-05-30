package core

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/bc"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/format"
)

const (
	recordTypeName          = "record"
	immutableRecordTypeName = "immutable-record"
)

// RecordValue creates new boxed record value.
func RecordValue(v *Dict, immutable bool) Value {
	return Value{
		Type:      VT_RECORD,
		Immutable: immutable,
		Ptr:       unsafe.Pointer(v),
	}
}

// NewRecordValue creates new (heap-allocated) record value.
func NewRecordValue(vals map[string]Value, immutable bool) Value {
	t := &Dict{}
	t.Set(vals)
	return RecordValue(t, immutable)
}

var TypeRecord = ValueType{
	Name:         SeqNameHook(recordTypeName, immutableRecordTypeName),
	String:       recordTypeString,
	Format:       recordTypeFormat,
	Interface:    DictInterface,
	EncodeJSON:   DictEncodeJSON,
	EncodeBinary: DictEncodeBinary,
	DecodeBinary: DictDecodeBinary,
	IsTrue:       DictIsTrue,
	IsIterable:   ConstHook(true),
	Iterator:     func(a *Arena, v Value) (Value, error) { return a.NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil },
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
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for k, v := range o.Elements {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, v.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func recordTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
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

func recordTypeClone(a *Arena, v Value) (Value, error) {
	// Deep copy the record (and make it mutable) and its elements
	o := (*Dict)(v.Ptr)
	c := a.NewDict(len(o.Elements))
	for k, v := range o.Elements {
		t, err := v.Clone(a)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return a.NewRecordValue(c, false), nil
}

func recordTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Dict)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, fmt.Errorf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName())
	}
	return e.Call(vm, args)
}

func recordTypeAccess(a *Arena, v Value, index Value, mode bc.Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	o := (*Dict)(v.Ptr)
	r, ok := o.Elements[k]
	if !ok {
		return Undefined, nil
	}
	return r, nil
}
