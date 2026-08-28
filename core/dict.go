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
	dictTypeName          = "dict"
	immutableDictTypeName = "immutable-dict"
)

type Dict struct {
	Elements map[string]Value
}

func (o *Dict) Set(elements map[string]Value) {
	o.Elements = elements
}

// sortedKeys returns the dict's keys in a deterministic (lexical) order.
func (o *Dict) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewDictValue(m map[string]Value, immutable bool) Value {
	o := &Dict{Elements: m}
	return Value{Type: value.Dict, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeDict = ValueTypeDescr{
	Name:         SeqNameHook(dictTypeName, immutableDictTypeName), // PURE by contract
	String:       dictTypeString,                                   // PURE by contract
	Format:       dictTypeFormat,                                   // PURE by contract
	Interface:    dictTypeInterface,                                // PURE by contract
	EncodeJSON:   dictTypeEncodeJSON,                               // PURE by contract
	EncodeBinary: dictTypeEncodeBinary,                             // PURE by contract
	DecodeBinary: dictTypeDecodeBinary,                             // IMPURE by contract (mutates target)
	IsTrue:       dictTypeIsTrue,                                   // PURE by contract
	IsIterable:   ConstHook(true),                                  // PURE by contract
	Iterator:     dictTypeIterator,                                 // PURE by contract (constructs fresh iterator)
	Equal:        dictTypeEqual,                                    // PURE by contract
	BinaryOp:     dictTypeBinaryOp,                                 // PURE by contract
	Copy:         dictTypeCopy,                                     // PURE by contract
	Len:          dictTypeLen,                                      // PURE by contract
	MethodCall:   dictTypeMethodCall,                               // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       dictTypeAccess,                                   // PURE by contract
	Assign:       dictTypeAssign,                                   // IMPURE by contract
	Contains:     dictTypeContains,                                 // PURE by contract
	Delete:       dictTypeDelete,                                   // MUTATE-DEPENDENT by contract
	AsBool:       dictTypeAsBool,                                   // PURE by contract
	AsString:     dictTypeAsString,                                 // PURE by contract
	AsDict:       dictTypeAsDict,                                   // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (filter/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
}

func dictTypeString(v Value) string {
	o := (*Dict)(v.Ptr)
	pairs := make([]string, 0, len(o.Elements))
	for _, k := range o.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%q: %s", k, o.Elements[k].String()))
	}
	return fmt.Sprintf("dict({%s})", strings.Join(pairs, ", "))
}

func dictTypeInterface(v Value) any {
	o := (*Dict)(v.Ptr)
	res := make(map[string]any)
	for key, v := range o.Elements {
		res[key] = v.Interface()
	}
	return res
}

func dictTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)
	var b []byte
	b = append(b, '{')
	keys := o.sortedKeys()
	len1 := len(keys) - 1
	for idx, key := range keys {
		b = EncodeString(b, key)
		b = append(b, ':')
		eb, err := o.Elements[key].EncodeJSON()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, '}')
	return b, nil
}

func dictTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Dict)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for _, key := range o.sortedKeys() {
		b = binary.AppendBytes(b, []byte(key))
		eb, err := o.Elements[key].EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("dict value at key %q: %w", key, err)
		}
		b = binary.AppendBytes(b, eb)
	}
	return b, nil
}

func dictTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "dict (elements count)")
	if err != nil {
		return err
	}

	value := make(map[string]Value, int(count))
	for i := 0; i < int(count); i++ {
		kb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict key at index %d", i))
		if err != nil {
			return err
		}
		key := string(kb)
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("dict value at key %q", key))
		if err != nil {
			return err
		}
		var element Value
		if err := element.DecodeBinary(eb); err != nil {
			return fmt.Errorf("dict value at key %q: %w", key, err)
		}
		value[key] = element
	}
	if offset != len(data) {
		return fmt.Errorf("dict: trailing %d bytes", len(data)-offset)
	}

	*v = NewDictValue(value, v.Immutable)

	return nil
}

func dictTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return dictTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(dictTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(dictTypeString(v), sp, fspec.AlignLeft), nil
}

// deep=true recursively copies every value (today's copy() semantics); deep=false only clones the top-level
// map header, leaving nested containers sharing the source (copy_shallow()).
func dictTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Dict)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	if !deep {
		for k, e := range o.Elements {
			c[k] = e
		}
		return NewDictValue(c, false), nil
	}
	for k, e := range o.Elements {
		t, err := e.Copy(true)
		if err != nil {
			return Undefined, err
		}
		c[k] = t
	}
	return NewDictValue(c, false), nil
}

// DictToRecord converts a dict to a record. share=true reuses the dict's own map directly (record_view() /
// record_view(dict_val) — the explicit performance opt-in, today's original dict.record() behavior preserved
// under the new name); share=false (record() / dict_val.record()) builds an independent shallow copy — a new
// top-level map, elements copied by reference (not recursively cloned), matching every other type's own
// `.record()` conversion (array/bytes/runes/string/range all shallow-copy the same way). Used by both the
// dict.record()/dict.record_view() member cases and the free record()/record_view() constructors.
func DictToRecord(v Value, share bool) Value {
	o := (*Dict)(v.Ptr)
	if share {
		return NewRecordValue(o.Elements, v.Immutable)
	}
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		c[k] = e
	}
	return NewRecordValue(c, false)
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func dictTypeIterator(v Value) (Value, error) {
	return NewDictIteratorValue((*Dict)(v.Ptr).Elements), nil
}

func dictTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Dict:
		return mapsEqual((*Dict)(v.Ptr).Elements, (*Dict)(other.Ptr).Elements)
	case value.Record:
		return mapsEqual((*Dict)(v.Ptr).Elements, (*Record)(other.Ptr).Elements)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)

}

// PURE by contract.
func dictTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.Record:
			r := (*Dict)(v.Ptr).Elements
			switch op {
			case token.Add:
				l := (*Record)(other.Ptr).Elements
				return NewDictValue(mergeMaps(l, r), false), nil
			}
		}
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	l := (*Dict)(v.Ptr).Elements
	switch other.Type {
	case value.Dict:
		r := (*Dict)(other.Ptr).Elements
		switch op {
		case token.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.Record:
		r := (*Record)(other.Ptr).Elements
		switch op {
		case token.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.String:
		switch op {
		case token.Sub:
			return dictTypeDelete(v, other, false)
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func dictTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictTypeCopy(v, true)

	case "copy_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictTypeCopy(v, false)

	case "freeze_shallow":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.ToImmutable()

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return DictToRecord(v, false), nil

	case "record_view":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return DictToRecord(v, true), nil

	case "array":
		// a map's conversion elements are its ENTRIES, key-sorted, so
		// d.array().dict() round-trips
		o := (*Dict)(v.Ptr)
		return convMember(name, dictTypeName, args, true, NewArrayValue(MapToSortedEntries(o.Elements), false))

	case "time":
		// a components map is a conversion (it has a receiver), unlike the
		// positional construction forms; time(d.components-shaped dict) agrees
		o := (*Dict)(v.Ptr)
		t, err := TimeFromComponents(o.Elements)
		if err != nil {
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, err
		}
		return convMember(name, dictTypeName, args, true, NewTimeValue(t))

	case "range":
		o := (*Dict)(v.Ptr)
		r, err := RangeFromComponents(o.Elements)
		if err != nil {
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, err
		}
		return convMember(name, dictTypeName, args, true, r)

	case "delete_in_place":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return dictTypeDelete(v, args[0], true)

	case "delete":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return dictTypeDelete(v, args[0], false)

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := dictTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(o.Elements) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(o.Elements))), nil

	case "keys":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictFnKeys(v)

	case "values":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return dictFnValues(v)

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(dictTypeContains(v, args[0])), nil

	case "filter":
		return dictFnFilter(vm, v, args)

	case "count":
		return dictFnCount(vm, v, args)

	case "all":
		return dictFnAll(vm, v, args)

	case "any":
		return dictFnAny(vm, v, args)

	case "for_each":
		return dictFnForEach(vm, v, args)

	case "find":
		return dictFnFind(vm, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE by contract
func dictTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	k, ok := index.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("key access", "string", index.TypeName())
	}

	if mode == bc.AccessIndex {
		o := (*Dict)(v.Ptr)
		r, ok := o.Elements[k]
		if !ok {
			return Undefined, nil
		}
		return r, nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), k)
}

func dictFnKeys(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	keys := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		keys = append(keys, NewStringValue(k))
	}
	return NewArrayValue(keys, false), nil
}

func dictFnValues(v Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	sorted := o.sortedKeys()
	values := make([]Value, 0, len(sorted))
	for _, k := range sorted {
		values = append(values, o.Elements[k])
	}
	return NewArrayValue(values, false), nil
}

func dictFnFilter(vm VM, v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "0 or 1", len(args))
	}

	o := (*Dict)(v.Ptr)
	filtered := make(map[string]Value, len(o.Elements))

	if len(args) == 0 {
		for k, v := range o.Elements {
			if v.Type != value.Undefined {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "function", fn.TypeName())
	}

	var buf [2]Value

	switch fn.Arity() {
	case 1:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil

	case 2:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				filtered[k] = v
			}
		}
		return NewDictValue(filtered, false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnCount(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		var count int64
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		o := (*Dict)(v.Ptr)
		var count int64
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if !t {
				return Undefined, nil
			}
		}

	case 2:
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if !t {
				return Undefined, nil
			}
		}
	}
	return Undefined, nil
}

func dictFnFind(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "function", fn.TypeName())
	}

	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			nv := NewStringValue(k)
			buf[0] = nv
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				return nv, nil
			}
		}
		return Undefined, nil

	case 2:
		for k, v := range o.Elements {
			nv := NewStringValue(k)
			buf[0] = nv
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				return nv, nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnAll(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if !t {
				return False, nil
			}
		}
		return True, nil

	case 2:
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if !t {
				return False, nil
			}
		}
		return True, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictFnAny(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "function", fn.TypeName())
	}

	var buf [2]Value
	switch fn.Arity() {
	case 1:
		o := (*Dict)(v.Ptr)
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				return True, nil
			}
		}
		return False, nil

	case 2:
		o := (*Dict)(v.Ptr)
		for k, v := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = v
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				return True, nil
			}
		}
		return False, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}

func dictTypeIsTrue(v Value) (bool, error) {
	return len((*Dict)(v.Ptr).Elements) > 0, nil
}

func dictTypeLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
func dictTypeAssign(v Value, index Value, r Value) error {
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

func dictTypeContains(v Value, e Value) bool {
	s, ok := e.AsString()
	if !ok {
		return false
	}
	_, ok = (*Dict)(v.Ptr).Elements[s]
	return ok
}

// mutate=true: IMPURE, removes an entry from the receiver in place (delete_in_place()). Not folded by the
// optimizer. mutate=false: PURE, returns an independent dict without the key (delete()), leaving the receiver
// untouched — works regardless of the receiver's mutability, since nothing is mutated. See docs/purity.md.
func dictTypeDelete(v Value, key Value, mutate bool) (Value, error) {
	s, ok := key.AsString()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("delete key", "string", key.TypeName())
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotDeletableError(v.TypeName())
		}
		delete((*Dict)(v.Ptr).Elements, s)
		return v, nil
	}

	o := (*Dict)(v.Ptr)
	c := make(map[string]Value, len(o.Elements))
	for k, e := range o.Elements {
		if k != s {
			c[k] = e
		}
	}
	return NewDictValue(c, false), nil
}

func dictTypeAsBool(v Value) (bool, bool) {
	return len((*Dict)(v.Ptr).Elements) > 0, true
}

func dictTypeAsString(v Value) (string, bool) {
	return v.String(), true
}

func dictTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
