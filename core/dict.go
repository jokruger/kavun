package core

import (
	"fmt"
	"slices"
	"sort"
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
	// IsView reports whether Elements is shared with another value (a record it
	// was viewed from); set only by the explicit _view constructors
	IsView bool
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
		r := NewRecordValue(o.Elements, v.Immutable)
		(*Record)(r.Ptr).IsView = true
		return r
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

	case "remove_in_place":
		// the twin runs remove's own dispatch (key set or predicate) and applies it to the receiver
		if v.Immutable {
			return Undefined, errs.NewNotDeletableError(v.TypeName())
		}
		res, err := dictMatchMember(vm, name, v, args)
		if err != nil {
			return Undefined, err
		}
		o.Set((*Dict)(res.Ptr).Elements)
		return v, nil

	case "contains", "count", "filter", "remove", "any", "all":
		return dictMatchMember(vm, name, v, args)

	case "filter_in_place":
		// the twin runs filter's own dispatch (key set or predicate) and applies it to the receiver
		if v.Immutable {
			return Undefined, errs.NewNotDeletableError(v.TypeName())
		}
		res, err := dictMatchMember(vm, name, v, args)
		if err != nil {
			return Undefined, err
		}
		o.Set((*Dict)(res.Ptr).Elements)
		return v, nil

	case "map":
		// maps the ATTACHMENT, keys fixed — 1:1, answering a dict. Re-keying is a
		// different operation, and one that can collide. The callback follows the
		// map family's bindings: f/1 receives the key, f/2 (key, value)
		call, err := seqMapCallback(name, args)
		if err != nil {
			return Undefined, err
		}
		fn := args[0]
		mapped := make(map[string]Value, len(o.Elements))
		for _, k := range o.sortedKeys() {
			var res Value
			if fn.Arity() >= 2 {
				res, err = fn.Call(vm, []Value{NewStringValue(k), o.Elements[k]})
			} else {
				res, err = call(vm, 0, NewStringValue(k))
			}
			if err != nil {
				return Undefined, err
			}
			mapped[k] = res
		}
		return NewDictValue(mapped, false), nil

	case "reduce":
		// f/2 receives (acc, key), f/3 (acc, key, value) — the key is the element
		// (mirroring array.reduce); keys are visited in sorted order so callback
		// side effects and the fold itself are deterministic
		if len(args) != 2 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "2", len(args))
		}
		acc := args[0]
		fn := args[1]
		if !fn.IsCallable() {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "second", "function", fn.TypeName())
		}
		arity := fn.Arity()
		if arity != 2 && arity != 3 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "second", "f/2 or f/3", fn.TypeName())
		}
		for _, k := range o.sortedKeys() {
			var res Value
			var err error
			if arity == 3 {
				res, err = fn.Call(vm, []Value{acc, NewStringValue(k), o.Elements[k]})
			} else {
				res, err = fn.Call(vm, []Value{acc, NewStringValue(k)})
			}
			if err != nil {
				return Undefined, err
			}
			acc = res
		}
		return acc, nil

	case "merge":
		return dictTypeMerge(v, args, false)

	case "merge_in_place":
		return dictTypeMerge(v, args, true)

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

	case "for_each":
		return dictFnForEach(vm, v, args)

	case "index":
		return dictFnIndex(vm, v, args)

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

func dictFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see SeqForEach)
	o := (*Dict)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for k := range o.Elements {
			buf[0] = NewStringValue(k)
			if _, err := fn.Call(vm, buf[:1]); err != nil {
				return Undefined, err
			}
		}

	case 2:
		for k, e := range o.Elements {
			buf[0] = NewStringValue(k)
			buf[1] = e
			if _, err := fn.Call(vm, buf[:2]); err != nil {
				return Undefined, err
			}
		}
	}
	return v, nil
}

// dictFnIndex is the locator on a map: it answers a KEY. A dict's element is its key, so the value reading is
// key equality and the predicate reading yields the first key (in sorted order, for determinism) satisfying the
// callback. There is no absent reading — keys are identities, never filler — and no index_last: unordered.
func dictFnIndex(vm VM, v Value, args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError("index", "1 or 2", len(args))
	}
	o := (*Dict)(v.Ptr)
	needle := args[0]
	dflt := args[1:]
	miss := func() (Value, error) {
		if len(dflt) == 1 {
			return dflt[0], nil
		}
		return Undefined, nil
	}

	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if needle.IsCallable() {
		arity := needle.Arity()
		if arity != 1 && arity != 2 {
			return Undefined, errs.NewInvalidArgumentTypeError("index", "first", "f/1 or f/2", needle.TypeName())
		}
		var buf [2]Value
		for _, k := range keys {
			if arity == 2 {
				buf[0] = NewStringValue(k)
				buf[1] = o.Elements[k]
			} else {
				buf[0] = NewStringValue(k)
			}
			res, err := needle.Call(vm, buf[:arity])
			if err != nil {
				return Undefined, err
			}
			t, terr := res.IsTrue()
			if terr != nil {
				return Undefined, terr
			}
			if t {
				return NewStringValue(k), nil
			}
		}
		return miss()
	}

	k, ok := needle.AsString()
	if !ok {
		return Undefined, errs.NewInvalidArgumentTypeError("index", "first", "a key or a predicate", needle.TypeName())
	}
	if _, exists := o.Elements[k]; exists {
		return NewStringValue(k), nil
	}
	return miss()
}

// dictMatchMember is the match family on a map — contains / count / any / all /
// filter / remove, all reading the KEY axis (a dict is a set of keys, each with
// an attached value). Arguments: string keys form the element set; a single
// function is a predicate (f/1 gets the key, f/2 gets key and value); a map
// argument (the submap reading) is deferred and raises saying so. A map has no
// blank reading — it has two axes, so the no-argument form raises; reach the
// value axis with a predicate or via values(). Keys are visited in sorted order
// so predicate side effects and short-circuiting are deterministic.
func dictMatchMember(vm VM, name string, v Value, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)

	// the _in_place twin runs the same dispatch and verb; the caller applies the mutation.
	// The full member name stays in every error message.
	verb := strings.TrimSuffix(name, "_in_place")

	var pred func(k string, val Value) (bool, error)

	switch {
	case len(args) == 0:
		return Undefined, errs.NewWrongNumArgumentsError(name, "1 or more (a map has no blank reading)", 0)

	case args[0].IsCallable():
		if len(args) > 1 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "a single predicate (a function among several arguments has no reading)", "mixed")
		}
		fn := args[0]
		arity := fn.Arity()
		if arity != 1 && arity != 2 {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "f/1 or f/2", fn.TypeName())
		}
		pred = func(k string, val Value) (bool, error) {
			var buf [2]Value
			n := 1
			if arity >= 2 {
				buf[0] = NewStringValue(k)
				buf[1] = val
				n = 2
			} else {
				buf[0] = NewStringValue(k)
			}
			res, err := fn.Call(vm, buf[:n])
			if err != nil {
				return false, err
			}
			return res.IsTrue()
		}

	default:
		keys := make(map[string]struct{}, len(args))
		for _, a := range args {
			if a.IsCallable() {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "arguments", "one reading per call (a function among several arguments always raises)", "mixed")
			}
			if a.Type == value.Dict || a.Type == value.Record {
				return Undefined, errs.NewNotImplementedError("(" + name + ") the submap reading is deferred; match keys, or compare entries with a predicate")
			}
			k, ok := a.AsString()
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "argument", "a key (string) or a predicate", a.TypeName())
			}
			keys[k] = struct{}{}
		}
		pred = func(k string, _ Value) (bool, error) {
			_, hit := keys[k]
			return hit, nil
		}
	}

	sorted := o.sortedKeys()
	switch verb {
	case "contains", "any":
		for _, k := range sorted {
			t, err := pred(k, o.Elements[k])
			if err != nil {
				return Undefined, err
			}
			if t {
				return True, nil
			}
		}
		return False, nil

	case "all":
		for _, k := range sorted {
			t, err := pred(k, o.Elements[k])
			if err != nil {
				return Undefined, err
			}
			if !t {
				return False, nil
			}
		}
		return True, nil

	case "count":
		n := int64(0)
		for _, k := range sorted {
			t, err := pred(k, o.Elements[k])
			if err != nil {
				return Undefined, err
			}
			if t {
				n++
			}
		}
		return IntValue(n), nil

	case "filter", "remove":
		keepMatches := verb == "filter"
		kept := make(map[string]Value, len(o.Elements))
		for _, k := range sorted {
			t, err := pred(k, o.Elements[k])
			if err != nil {
				return Undefined, err
			}
			if t == keepMatches {
				kept[k] = o.Elements[k]
			}
		}
		return NewDictValue(kept, false), nil
	}
	return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
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
// dictTypeMerge implements merge/merge_in_place — the map family's whole add side: variadic over maps (dict
// and record, the family's two members), entries applied in ARGUMENT ORDER with last-wins on key collision,
// exactly the + operator's rule. There is deliberately no single-entry add member: the single-entry spellings
// are d[k] = v (mutating, a statement) and d.merge(dict([[k, v]])) (non-mutating). mutate=false returns a
// fresh dict, receiver untouched; mutate=true writes into the receiver's own map, visible to every live alias,
// and returns the receiver. Both accept zero arguments as a legal no-op.
func dictTypeMerge(v Value, args []Value, mutate bool) (Value, error) {
	name := "merge"
	if mutate {
		name = "merge_in_place"
	}
	maps := make([]map[string]Value, 0, len(args))
	for i, a := range args {
		m, ok := a.AsDict()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, fmt.Sprintf("%d", i+1), "dict or record", a.TypeName())
		}
		maps = append(maps, m)
	}

	o := (*Dict)(v.Ptr)
	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotAppendableError(v.TypeName())
		}
		for _, m := range maps {
			for k, e := range m {
				o.Elements[k] = e
			}
		}
		return v, nil
	}

	c := make(map[string]Value, len(o.Elements)+len(args))
	for k, e := range o.Elements {
		c[k] = e
	}
	for _, m := range maps {
		for k, e := range m {
			c[k] = e
		}
	}
	return NewDictValue(c, false), nil
}

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
