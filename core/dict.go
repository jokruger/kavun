package core

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unsafe"

	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/token/tokens"
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

// sortedKeys returns the dict's keys in lexical order. EVERY key enumeration on a map goes through this (or
// through the iterator, which sorts the same way): members, iteration, rendering and encoding alike. Key order
// is part of the language contract, not an implementation detail — never walk o.Elements directly to produce
// an observable sequence, or the Go map's randomized order leaks into script behaviour.
func (o *Dict) sortedKeys() []string {
	keys := make([]string, 0, len(o.Elements))
	for k := range o.Elements {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func NewDictValue(m map[string]Value, immutable bool) Value {
	if m == nil {
		// a nil map reads fine but PANICS the host on assignment — every dict must be writable
		m = make(map[string]Value)
	}
	o := &Dict{Elements: m}
	return Value{Type: value.Dict, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeDict = ValueTypeDescr{
	Name:                MutabilityNameHook(dictTypeName, immutableDictTypeName), // PURE by contract
	String:              dictTypeString,                                          // PURE by contract
	Format:              dictTypeFormat,                                          // PURE by contract
	Interface:           dictTypeInterface,                                       // PURE by contract
	EncodeJSON:          dictTypeEncodeJSON,                                      // PURE by contract
	EncodeBinary:        dictTypeEncodeBinary,                                    // PURE by contract
	DecodeBinary:        dictTypeDecodeBinary,                                    // IMPURE by contract (mutates target)
	IsTrue:              dictTypeIsTrue,                                          // PURE by contract
	IsIterable:          ConstHook(true),                                         // PURE by contract
	Iterator:            dictTypeIterator,                                        // PURE by contract (constructs fresh iterator)
	Equal:               dictTypeEqual,                                           // PURE by contract
	BinaryOp:            dictTypeBinaryOp,                                        // PURE by contract
	Copy:                dictTypeCopy,                                            // PURE by contract
	Len:                 dictTypeLen,                                             // PURE by contract
	CallNamedMethod:     dictTypeCallNamedMethod,                                 // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
	AccessIndex:         dictTypeAccessIndex,                                     // PURE by contract
	AccessNamedProperty: dictTypeAccessNamedProperty,                             // PURE by contract
	AssignIndex:         dictTypeAssignIndex,                                     // IMPURE by contract
	AssignNamedProperty: dictTypeAssignNamedProperty,                             // IMPURE by contract
	Contains:            dictTypeContains,                                        // PURE by contract
	Delete:              dictTypeDelete,                                          // MUTATE-DEPENDENT by contract
	AsBool:              dictTypeAsBool,                                          // PURE by contract
	AsDict:              dictTypeAsDict,                                          // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (keep/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsNamedMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
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
			case tokens.Add:
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
		case tokens.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.Record:
		r := (*Record)(other.Ptr).Elements
		switch op {
		case tokens.Add:
			return NewDictValue(mergeMaps(l, r), false), nil
		}

	case value.String:
		switch op {
		case tokens.Sub:
			return dictTypeDelete(v, other, false)
		}

	default:
		// `-` shares remove's key reading: any operand whose string conversion exists
		// names a key (`d - 1` removes "1"), exactly like d.remove(1) and `1 in d`
		if op == tokens.Sub {
			if _, ok := other.AsString(); ok {
				return dictTypeDelete(v, other, false)
			}
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsNamedMethodPure (see docs/purity.md)
func dictTypeCallNamedMethod(vm VM, v Value, name string, args []Value) (Value, error) {
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
		// the same-type conversion constructs — a new, independent, mutable shallow copy, exactly
		// dict(d) / d.copy_shallow(); see the note on array's own case. record_view() is how a
		// script asks for shared storage instead.
		c, err := dictTypeCopy(v, false)
		if err != nil {
			return Undefined, err
		}
		return convMember(name, dictTypeName, args, true, c)

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

	case "date":
		o := (*Dict)(v.Ptr)
		d, err := DateFromComponents(o.Elements)
		if err != nil {
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, err
		}
		return convMember(name, dictTypeName, args, true, DateValue(d))

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

	case "contains":
		return dictContainsMember(vm, v, args)

	case "count":
		return dictCount(vm, v, args)

	case "keep":
		return dictKeep(vm, v, args)

	case "keep_in_place":
		return dictKeepInPlace(vm, v, args)

	case "remove":
		return dictRemove(vm, v, args)

	case "remove_in_place":
		return dictRemoveInPlace(vm, v, args)

	case "any":
		return dictAny(vm, v, args)

	case "all":
		return dictAll(vm, v, args)

	case "map":
		// maps the ATTACHMENT, keys fixed — 1:1, answering a dict. Re-keying is a
		// different operation, and one that can collide. The callback follows the
		// map family's bindings: f/1 receives the key, f/2 (key, value)
		fn, err := readElemCallback(name, args)
		if err != nil {
			return Undefined, err
		}
		mapped := make(map[string]Value, len(o.Elements))
		for _, k := range o.sortedKeys() {
			var res Value
			if fn.Arity() == 2 {
				res, err = fn.Call(vm, []Value{NewStringValue(k), o.Elements[k]})
			} else {
				res, err = fn.Call(vm, []Value{NewStringValue(k)})
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
		return memberFormat(vm, v, members.Format, args)

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
		return CallMemberByLookup(vm, v, name, args)
	}
}

// PURE by contract
func dictTypeAccessIndex(v Value, index Value) (Value, error) {
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

// PURE by contract — a dict's keys are data, reached by d[k] only; d.name is refused
func dictTypeAccessNamedProperty(vm VM, v Value, name string) (Value, error) {
	if e, id, ok := PropertyByLookup(v, name); ok && e.Get != nil {
		return e.Get(vm, v, id)
	}
	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), name)
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
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see arrayForEach). Keys are visited in
	// sorted order, like every other member and like `for k in d` — for_each is the side-effecting member,
	// so its order is the most observable of all.
	o := (*Dict)(v.Ptr)
	var buf [2]Value
	sorted := o.sortedKeys()
	switch fn.Arity() {
	case 1:
		for _, k := range sorted {
			buf[0] = NewStringValue(k)
			if _, err := fn.Call(vm, buf[:1]); err != nil {
				return Undefined, err
			}
		}

	case 2:
		for _, k := range sorted {
			buf[0] = NewStringValue(k)
			buf[1] = o.Elements[k]
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

// ---------------------------------------------------------------------------
// The match members: contains / count / keep / remove / any / all, all reading the KEY axis (a dict is a set of
// keys, each with an attached value):
//   - a function       a predicate, f/1(key) or f/2(key, value)
//   - a string         one key; several form a set
//   - a dict/record    the submap reading: deferred, raises saying so
//
// A map has no blank reading — it has two axes — so the no-argument form raises; reach the value axis with a
// predicate or via values(). Keys are visited in sorted order, so predicate side effects and short-circuiting
// are deterministic.
// ---------------------------------------------------------------------------

// dictMatch is a match member's argument list after reading; exactly one reading is set.
type dictMatch struct {
	pred Value               // a predicate
	keys map[string]struct{} // a set of keys
}

// dictReadMatchArgs reads a match member's arguments.
func dictReadMatchArgs(name string, args []Value) (dictMatch, error) {
	if len(args) == 0 {
		return dictMatch{}, errs.NewWrongNumArgumentsError(name, "1 or more (a map has no blank reading)", 0)
	}

	if args[0].IsCallable() {
		if len(args) > 1 {
			return dictMatch{}, errPredicateAmongMany(name)
		}
		if err := checkElemCallback(name, args[0]); err != nil {
			return dictMatch{}, err
		}
		return dictMatch{pred: args[0]}, nil
	}

	keys := make(map[string]struct{}, len(args))
	for _, a := range args {
		if a.IsCallable() {
			return dictMatch{}, errFunctionInSet(name)
		}
		if a.Type == value.Dict || a.Type == value.Record {
			return dictMatch{}, errs.NewNotImplementedError("(" + name + ") the submap reading is deferred; match keys, or compare entries with a predicate")
		}
		k, ok := a.AsString()
		if !ok {
			return dictMatch{}, errs.NewInvalidArgumentTypeError(name, "argument", "a key (string) or a predicate", a.TypeName())
		}
		keys[k] = struct{}{}
	}
	return dictMatch{keys: keys}, nil
}

// matches reports whether the entry k: val matches the reading.
func (m *dictMatch) matches(vm VM, k string, val Value) (bool, error) {
	if m.pred.IsCallable() {
		var res Value
		var err error
		if m.pred.Arity() == 2 {
			res, err = m.pred.Call(vm, []Value{NewStringValue(k), val})
		} else {
			res, err = m.pred.Call(vm, []Value{NewStringValue(k)})
		}
		if err != nil {
			return false, err
		}
		return res.IsTrue()
	}
	_, hit := m.keys[k]
	return hit, nil
}

// dictContainsMember is contains(...): does some entry match?
func dictContainsMember(vm VM, v Value, args []Value) (Value, error) {
	m, err := dictReadMatchArgs("contains", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// dictCount is count(...): how many entries match.
func dictCount(vm VM, v Value, args []Value) (Value, error) {
	m, err := dictReadMatchArgs("count", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	n := int64(0)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// dictKept answers the entries keep(...) keeps: the matches. Shared by keep and keep_in_place; name is the
// member called, for the errors.
func dictKept(vm VM, name string, v Value, args []Value) (map[string]Value, error) {
	m, err := dictReadMatchArgs(name, args)
	if err != nil {
		return nil, err
	}
	o := (*Dict)(v.Ptr)
	out := make(map[string]Value, len(o.Elements))
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return nil, err
		}
		if hit {
			out[k] = o.Elements[k]
		}
	}
	return out, nil
}

// dictKeep is keep(...): a new dict of the matching entries.
func dictKeep(vm VM, v Value, args []Value) (Value, error) {
	out, err := dictKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewDictValue(out, false), nil
}

// IMPURE: mutates the receiver. dictKeepInPlace is keep_in_place(...): keep the matching entries in the receiver
// itself.
func dictKeepInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "keep_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := dictKept(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Dict)(v.Ptr).Set(out)
	return v, nil
}

// dictRemaining answers the entries remove(...) leaves: everything but the matches. Shared by remove and
// remove_in_place; name is the member called, for the errors.
func dictRemaining(vm VM, name string, v Value, args []Value) (map[string]Value, error) {
	m, err := dictReadMatchArgs(name, args)
	if err != nil {
		return nil, err
	}
	o := (*Dict)(v.Ptr)
	out := make(map[string]Value, len(o.Elements))
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return nil, err
		}
		if !hit {
			out[k] = o.Elements[k]
		}
	}
	return out, nil
}

// dictRemove is remove(...): a new dict without the matching entries.
func dictRemove(vm VM, v Value, args []Value) (Value, error) {
	out, err := dictRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewDictValue(out, false), nil
}

// IMPURE: mutates the receiver. dictRemoveInPlace is remove_in_place(...): drop the matching entries from the
// receiver itself.
func dictRemoveInPlace(vm VM, v Value, args []Value) (Value, error) {
	const name = "remove_in_place"
	if v.Immutable {
		return Undefined, errs.NewNotMutableError(name, v.TypeName())
	}
	out, err := dictRemaining(vm, name, v, args)
	if err != nil {
		return Undefined, err
	}
	(*Dict)(v.Ptr).Set(out)
	return v, nil
}

// dictAny is any(...): does some entry match? The same question as contains on a map.
func dictAny(vm VM, v Value, args []Value) (Value, error) {
	m, err := dictReadMatchArgs("any", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// dictAll is all(...): does every entry match? True on an empty dict.
func dictAll(vm VM, v Value, args []Value) (Value, error) {
	m, err := dictReadMatchArgs("all", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(vm, k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
}

func dictTypeIsTrue(v Value) (bool, error) {
	return len((*Dict)(v.Ptr).Elements) > 0, nil
}

func dictTypeLen(v Value) int64 {
	o := (*Dict)(v.Ptr)
	return int64(len(o.Elements))
}

// IMPURE: writes into the receiver. Not folded by the optimizer. See docs/purity.md.
// Selector access is the record's feature: a dict refuses the dot in BOTH directions, mirroring
// dictTypeAccess — d.a = 5 raises exactly like d.a, and d[k] is the dict's data spelling.
func dictTypeAssignIndex(v Value, index Value, r Value) error {
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

// IMPURE by contract — d.name = r is refused, as d.name is
func dictTypeAssignNamedProperty(vm VM, v Value, name string, r Value) error {
	if e, id, ok := PropertyByLookup(v, name); ok && e.Set != nil {
		return e.Set(vm, v, id, r)
	}
	return errs.NewInvalidSelectorError(v.TypeName(), name)
}

// dictTypeContains is the `in` operator on the KEY axis (a map's element is its key); a submap
// operand raises as deferred, a callable raises — an operator operand is always a value.
func dictTypeContains(v Value, e Value) (bool, error) {
	return mapContainsKey((*Dict)(v.Ptr).Elements, e)
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
			return Undefined, errs.NewNotMutableError("merge_in_place", v.TypeName())
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
			return Undefined, errs.NewNotMutableError("remove_in_place", v.TypeName())
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

func dictTypeAsDict(v Value) (map[string]Value, bool) {
	return (*Dict)(v.Ptr).Elements, true
}
