package core

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
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

// sortedKeys returns the record's keys in a deterministic (lexical) order. Iteration order over a
// map is deliberately undefined, but everything that RENDERS or ENCODES one is deterministic, so a
// display, a JSON payload, and a binary blob are reproducible run to run (same as dict).
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
	Name:         SeqNameHook(recordTypeName, immutableRecordTypeName), // PURE by contract
	String:       recordTypeString,                                     // PURE by contract
	Format:       recordTypeFormat,                                     // PURE by contract
	Interface:    recordTypeInterface,                                  // PURE by contract
	EncodeJSON:   recordTypeEncodeJSON,                                 // PURE by contract
	EncodeBinary: recordTypeEncodeBinary,                               // PURE by contract
	DecodeBinary: recordTypeDecodeBinary,                               // IMPURE by contract (mutates target)
	IsTrue:       recordTypeIsTrue,                                     // PURE by contract
	IsIterable:   ConstHook(true),                                      // PURE by contract
	Iterator:     recordTypeIterator,                                   // PURE by contract (constructs fresh iterator)
	Copy:         recordTypeCopy,                                       // PURE by contract
	Len:          recordTypeLen,                                        // PURE by contract
	Equal:        recordTypeEqual,                                      // PURE by contract
	BinaryOp:     recordTypeBinaryOp,                                   // PURE by contract
	MethodCall:   recordTypeMethodCall,                                 // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       recordTypeAccess,                                     // PURE by contract
	Assign:       recordTypeAssign,                                     // IMPURE by contract
	Contains:     recordTypeContains,                                   // PURE by contract
	Delete:       recordTypeDelete,                                     // MUTATE-DEPENDENT by contract
	AsBool:       recordTypeAsBool,                                     // PURE by contract
	AsDict:       recordTypeAsDict,                                     // PURE by contract
	IsMethodPure: func(string) bool { return false },                   // method calls are redirected to the value keys, so conservatively assume they are impure
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
			return nil, fmt.Errorf("record value at key %q: %w", key, err)
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
// map header, leaving nested containers sharing the source (copy_shallow()). record has no MethodCall switch
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
// via the free `dict`/`dict_view` constructors: record has no `MethodCall` switch (see P14), so there is no
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
		case token.Add:
			return NewRecordValue(mergeMaps((*Record)(v.Ptr).Elements, (*Record)(other.Ptr).Elements), false), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func recordTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	// Function call on selector will be compiled as method call, so we need to process it here.
	o := (*Record)(v.Ptr)
	e, ok := o.Elements[name]
	if !ok {
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
	if !e.IsCallable() {
		return Undefined, errs.NewRecoverableError(errs.KindNotCallable, fmt.Sprintf("%s.%s is not callable, got %s", v.TypeName(), name, e.TypeName()))
	}
	return e.Call(vm, args)
}

// PURE by contract
func recordTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
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
func recordTypeAssign(v Value, index Value, r Value, _ bc.Opcode) error {
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

// recordTypeContains is the `in` operator on the KEY axis, exactly dict's rule.
func recordTypeContains(v Value, e Value) (bool, error) {
	return mapContainsKey((*Record)(v.Ptr).Elements, e)
}

// mutate=true: IMPURE, removes a field from the receiver in place (the free delete_in_place() builtin — record
// has no MethodCall switch, so this is only ever reached that way, never as a member call). Not folded by the
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
