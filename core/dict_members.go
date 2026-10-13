package core

import (
	"sort"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
)

// dict's members: one function per entry of TypeDict.Methods that is not a shared member (MemberIsTrue,
// MemberFormat, MemberSelf in tools.go).

// PURE by contract
func dictCopy(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return dictTypeCopy(v, true)
}

// PURE by contract
func dictCopyShallow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return dictTypeCopy(v, false)
}

// PURE by contract
func dictFreezeShallow(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.ToImmutable()
}

// PURE by contract
func dictFreeze(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v.Freeze()
}

// PURE by contract
func dictDict(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the same-type conversion constructs — a new, independent, mutable shallow copy, exactly
	// dict(d) / d.copy_shallow(); see the note on array's own case. record_view() is how a
	// script asks for shared storage instead.
	c, err := dictTypeCopy(v, false)
	if err != nil {
		return Undefined, err
	}
	return ConvMember(id.String(), dictTypeName, args, true, c)
}

// PURE by contract
func dictRecord(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return DictToRecord(v, false), nil
}

// PURE by contract
func dictRecordView(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return DictToRecord(v, true), nil
}

// PURE by contract
func dictArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// a map's conversion elements are its ENTRIES, key-sorted, so
	// d.array().dict() round-trips
	o := (*Dict)(v.Ptr)
	return ConvMember(id.String(), dictTypeName, args, true, NewArrayValue(MapToSortedEntries(o.Elements), false))
}

// PURE by contract
func dictTime(_ VM, v Value, id member.ID, args []Value) (Value, error) {
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
	return ConvMember(id.String(), dictTypeName, args, true, NewTimeValue(t))
}

// PURE by contract
func dictDate(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	d, err := DateFromComponents(o.Elements)
	if err != nil {
		if len(args) == 1 {
			return args[0], nil
		}
		return Undefined, err
	}
	return ConvMember(id.String(), dictTypeName, args, true, DateValue(d))
}

// PURE by contract
func dictRange(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	r, err := RangeFromComponents(o.Elements)
	if err != nil {
		if len(args) == 1 {
			return args[0], nil
		}
		return Undefined, err
	}
	return ConvMember(id.String(), dictTypeName, args, true, r)
}

// dictContains is contains(...): does some entry match?
// PURE by contract
func dictContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := dictReadMatchArgs(vm, "contains", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
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
// PURE by contract
func dictCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := dictReadMatchArgs(vm, "count", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	n := int64(0)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// dictKeep is keep(...): a new dict of the matching entries.
// PURE by contract
func dictKeep(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := dictKept(vm, "keep", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewDictValue(out, false), nil
}

// IMPURE: mutates the receiver. dictKeepInPlace is keep_in_place(...): keep the matching entries in the receiver
// itself.
func dictKeepInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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

// dictRemove is remove(...): a new dict without the matching entries.
// PURE by contract
func dictRemove(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	out, err := dictRemaining(vm, "remove", v, args)
	if err != nil {
		return Undefined, err
	}
	return NewDictValue(out, false), nil
}

// IMPURE: mutates the receiver. dictRemoveInPlace is remove_in_place(...): drop the matching entries from the
// receiver itself.
func dictRemoveInPlace(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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
// PURE by contract
func dictAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := dictReadMatchArgs(vm, "any", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
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
// PURE by contract
func dictAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	m, err := dictReadMatchArgs(vm, "all", args)
	if err != nil {
		return Undefined, err
	}
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		hit, err := m.matches(k, o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
}

// PURE by contract
func dictMap(vm VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	// maps the ATTACHMENT, keys fixed — 1:1, answering a dict. Re-keying is a
	// different operation, and one that can collide. The callback follows the
	// map family's bindings: f/1 receives the key, f/2 (key, value)
	cb, err := readElemCallback(vm, id.String(), args)
	if err != nil {
		return Undefined, err
	}
	mapped := make(map[string]Value, len(o.Elements))
	for _, k := range o.sortedKeys() {
		res, err := cb.CallEntry(NewStringValue(k), o.Elements[k])
		if err != nil {
			return Undefined, err
		}
		mapped[k] = res
	}
	return NewDictValue(mapped, false), nil
}

// PURE by contract
func dictReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	// f/2 receives (acc, key), f/3 (acc, key, value) — the key is the element
	// (mirroring array.reduce); keys are visited in sorted order so callback
	// side effects and the fold itself are deterministic
	acc, cb, err := readReduceArgs(vm, args)
	if err != nil {
		return Undefined, err
	}
	for _, k := range o.sortedKeys() {
		acc, err = cb.CallEntry(acc, NewStringValue(k), o.Elements[k])
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// PURE by contract
func dictMerge(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return dictTypeMerge(v, args, false)
}

// IMPURE by contract (mutates the receiver)
func dictMergeInPlace(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return dictTypeMerge(v, args, true)
}

// PURE by contract
func dictIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return BoolValue(len(o.Elements) == 0), nil
}

// PURE by contract
func dictLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	o := (*Dict)(v.Ptr)
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return IntValue(int64(len(o.Elements))), nil
}

// PURE by contract
func dictKeys(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return dictFnKeys(v)
}

// PURE by contract
func dictValues(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return dictFnValues(v)
}

// PURE by contract
func dictForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	cb, err := readElemCallback(vm, "for_each", args)
	if err != nil {
		return Undefined, err
	}

	// a full pass, callback return ignored; returns the receiver (see arrayForEach). Keys are visited in
	// sorted order, like every other member and like `for k in d` — for_each is the side-effecting member,
	// so its order is the most observable of all.
	o := (*Dict)(v.Ptr)
	for _, k := range o.sortedKeys() {
		if _, err := cb.CallEntry(NewStringValue(k), o.Elements[k]); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// dictIndex is the locator on a map: it answers a KEY. A dict's element is its key, so the value reading is
// key equality and the predicate reading yields the first key (in sorted order, for determinism) satisfying the
// callback. There is no absent reading — keys are identities, never filler — and no index_last: unordered.
// PURE by contract
func dictIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
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
		cb, err := newElemCallback(vm, "index", needle)
		if err != nil {
			return Undefined, err
		}
		for _, k := range keys {
			res, err := cb.CallEntry(NewStringValue(k), o.Elements[k])
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
