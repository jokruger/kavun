package core

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
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
	// IsView reports whether Elements is shared with another value (a dict it
	// was viewed from); set only by the explicit _view constructors
	IsView bool
}

func (o *Record) Set(elements map[string]Value) {
	o.Elements = elements
}

// sortedKeys returns the record's keys in lexical order — the same contract as dict (see Dict.sortedKeys).
// Iteration, rendering and encoding are all ordered, so a display, a JSON payload, a binary blob and a
// `for k in r` pass are all reproducible run to run.
func (o *Record) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewRecordValue(m map[string]Value, immutable bool) Value {
	if m == nil {
		// a nil map reads fine but PANICS the host on assignment — every record must be writable
		m = make(map[string]Value)
	}
	o := &Record{Elements: m}
	return Value{Type: value.Record, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeRecord = ValueTypeDescr{
	Name:                MutabilityNameHook(recordTypeName, immutableRecordTypeName), // PURE by contract
	String:              recordTypeString,                                            // PURE by contract
	Format:              recordTypeFormat,                                            // PURE by contract
	Interface:           recordTypeInterface,                                         // PURE by contract
	EncodeJSON:          recordTypeEncodeJSON,                                        // PURE by contract
	EncodeBinary:        recordTypeEncodeBinary,                                      // PURE by contract
	DecodeBinary:        recordTypeDecodeBinary,                                      // IMPURE by contract (mutates target)
	IsTrue:              recordTypeIsTrue,                                            // PURE by contract
	IsIterable:          ConstHook(true),                                             // PURE by contract
	Iterator:            recordTypeIterator,                                          // PURE by contract (constructs fresh iterator)
	Copy:                recordTypeCopy,                                              // PURE by contract
	Len:                 recordTypeLen,                                               // PURE by contract
	Equal:               recordTypeEqual,                                             // PURE by contract
	BinaryOp:            recordTypeBinaryOp,                                          // PURE by contract
	CallNamedMethod:     recordTypeCallNamedMethod,                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessNamedProperty: recordTypeAccessNamedProperty,                               // PURE by contract
	AccessIndex:         recordTypeAccessIndex,                                       // PURE by contract
	AssignNamedProperty: recordTypeAssignNamedProperty,                               // IMPURE by contract
	AssignIndex:         recordTypeAssignIndex,                                       // IMPURE by contract
	Contains:            recordTypeContains,                                          // PURE by contract
	Delete:              recordTypeDelete,                                            // MUTATE-DEPENDENT by contract
	AsBool:              recordTypeAsBool,                                            // PURE by contract
	AsDict:              recordTypeAsDict,                                            // PURE by contract
	IsNamedMethodPure:   func(string) bool { return false },                          // method calls are redirected to the value keys, so conservatively assume they are impure
}

func recordTypeString(v Value) string {
	o := (*Record)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for _, k := range o.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, o.Elements[k].String()))
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
	keys := o.sortedKeys()
	len1 := len(keys) - 1
	for idx, key := range keys {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := o.Elements[key].EncodeJSON()
		if err != nil {
			return nil, jsonPathPrefix("."+key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, '}')
	return b, nil
}

func recordTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Record)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for _, key := range o.sortedKeys() {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := o.Elements[key].EncodeBinary()
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

// deep=true recursively copies every value (today's copy() semantics); deep=false only clones the top-level
// map header, leaving nested containers sharing the source (copy_shallow()). record has no CallNamedMethod switch
// (see P14/function-matrix.md), so neither is reachable as a member call — only via the free copy() builtin,
// which dispatches here through the Value.Copy hook regardless.
func recordTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	if !deep {
		for k, e := range o.Elements {
			c[k] = e
		}
		return NewRecordValue(c, false), nil
	}
	for k, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewRecordValue(c, false), nil
}

// RecordToDict converts a record to a dict. share=true reuses the record's own map directly (dict_view(record_val)
// — the explicit performance opt-in, today's original dict(record_val) behavior preserved under the new name);
// share=false (dict(record_val)) builds an independent shallow copy — a new top-level map, elements copied by
// reference (not recursively cloned), matching every other type's own `.dict()` conversion. Only ever reached
// via the free `dict`/`dict_view` constructors: record has no `CallNamedMethod` switch (see P14), so there is no
// `record_val.dict()` member form and never was.
func RecordToDict(v Value, share bool) Value {
	o := (*Record)(v.Ptr)
	if share {
		d := NewDictValue(o.Elements, v.Immutable)
		(*Dict)(d.Ptr).IsView = true
		return d
	}
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		c[k] = e
	}
	return NewDictValue(c, false)
}

func recordTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Record:
		return mapsEqual((*Record)(v.Ptr).Elements, (*Record)(other.Ptr).Elements)
	case value.Dict:
		return mapsEqual((*Record)(v.Ptr).Elements, (*Dict)(other.Ptr).Elements)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func recordTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Record:
		switch op {
		case tokens.Add:
			return NewRecordValue(mergeMaps((*Record)(v.Ptr).Elements, (*Record)(other.Ptr).Elements), false), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func recordTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
	// r.f(args) compiles as a member call, so a callable field is called from here.
	e, ok := recordField(v, name)
	if !ok {
		return CallMemberByLookup(vm, v, name, args)
	}
	if !e.IsCallable() {
		return Undefined, errs.NewRecoverableError(errs.KindNotCallable, fmt.Sprintf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName()))
	}
	return e.Call(vm, args)
}

// recordField is the one field lookup behind every record member and index path: r.f(...), r.f, r["f"].
func recordField(v Value, name string) (Value, bool) {
	e, ok := (*Record)(v.Ptr).Elements[name]
	return e, ok
}

// recordSetField is the one field write behind r.f = x and r["f"] = x.
func recordSetField(v Value, name string, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	(*Record)(v.Ptr).Elements[name] = r
	return nil
}

// PURE by contract — a missing field answers undefined
func recordTypeAccessNamedProperty(_ VM, v Value, name string) (Value, error) {
	e, _ := recordField(v, name)
	return e, nil
}

// PURE by contract — r[k] is r.k for a string k
func recordTypeAccessIndex(v Value, index Value) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}
	e, _ := recordField(v, k)
	return e, nil
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func recordTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Record)(v.Ptr).Elements), nil
}

func recordTypeIsTrue(v Value) (bool, error) {
	return len((*Record)(v.Ptr).Elements) > 0, nil
}

func recordTypeLen(v Value) int64 {
	o := (*Record)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes a field into the receiver. Not folded by the optimizer. See docs/purity.md.
func recordTypeAssignNamedProperty(_ VM, v Value, name string, r Value) error {
	return recordSetField(v, name, r)
}

// IMPURE: writes a field into the receiver; r[k] = x is r.k = x for a string k. See docs/purity.md.
func recordTypeAssignIndex(v Value, index Value, r Value) error {
	if v.Immutable {
		return errs.NewNotAssignableError(v.TypeName())
	}
	k, ok := index.AsString()
	if !ok {
		return errs.NewInvalidIndexTypeError("key assign", "string", index.TypeName())
	}
	return recordSetField(v, k, r)
}

// recordTypeContains is the `in` operator on the KEY axis, exactly dict's rule.
func recordTypeContains(v Value, e Value) (bool, error) {
	return mapContainsKey((*Record)(v.Ptr).Elements, e)
}

// mutate=true: IMPURE, removes a field from the receiver in place (the free delete_in_place() builtin — record
// has no CallNamedMethod switch, so this is only ever reached that way, never as a member call). Not folded by the
// optimizer. mutate=false: PURE, returns an independent record without the key (the free delete() builtin),
// leaving the receiver untouched — works regardless of the receiver's mutability. See docs/purity.md.
func recordTypeDelete(v Value, key Value, mutate bool) (Value, error) {
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotMutableError("remove_in_place", v.TypeName())
		}
		delete((*Record)(v.Ptr).Elements, s)
		return v, nil
	}

	o := (*Record)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		if k != s {
			c[k] = e
		}
	}
	return NewRecordValue(c, false), nil
}

func recordTypeAsBool(v Value) (bool, bool) {
	return len((*Record)(v.Ptr).Elements) > 0, true
}

func recordTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Record)(v.Ptr).Elements, true
}
