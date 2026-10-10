package core

import (
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/token/tokens"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
)

// int_range's members: one function per entry of TypeIntRange.Methods that is not a shared member (memberIsTrue,
// memberFormat, memberSelf in tools.go).

// PURE by contract
func intRangeArray(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	t, err := intRangeElements(name, v)
	if err != nil {
		return Undefined, err
	}
	return NewArrayValue(t, false), nil
}

// intRangeBytes is element-wise and all-or-nothing, like every sequence conversion: an element outside
// the octet range fails the whole conversion (answering the optional default or raising) — it never wraps.
// PURE by contract
func intRangeBytes(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("bytes", v)
	if err != nil {
		return Undefined, err
	}
	bs := make([]byte, len(elems))
	ok := true
	for i, t := range elems {
		if t < 0 || t > 255 {
			ok = false
			break
		}
		bs[i] = byte(t)
	}
	return convMember("bytes", intRangeTypeName, args, ok, NewBytesValue(bs, false))
}

// PURE by contract
func intRangeRange(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return convMember(id.String(), intRangeTypeName, args, true, v)
}

// PURE by contract
func intRangeComponents(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the constitutive parts, as a record — exactly what range(rec) rebuilds from
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	return NewRecordValue(map[string]Value{
		"start": IntValue(o.Start),
		"stop":  IntValue(o.Stop),
		"step":  IntValue(o.Step),
	}, false), nil
}

// PURE by contract
func intRangeString(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	rs, ok, err := intRangeCodePoints(name, v)
	if err != nil {
		return Undefined, err
	}
	return convMember(name, intRangeTypeName, args, ok, NewStringValue(EncodeText(rs)))
}

// PURE by contract
func intRangeRunes(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	rs, ok, err := intRangeCodePoints(name, v)
	if err != nil {
		return Undefined, err
	}
	return convMember(name, intRangeTypeName, args, ok, NewRunesValue(rs, false))
}

// PURE by contract
func intRangeIsEmpty(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	return BoolValue(o.Start == o.Stop), nil
}

// PURE by contract
func intRangeLen(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	return IntValue(o.Len()), nil
}

// intRangeContains is contains(...): is there a match anywhere? One int is answered in closed form, exactly
// like the `in` operator — nothing is scanned; every other reading walks the elements.
// PURE by contract
func intRangeContains(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	if len(args) == 1 && args[0].Type == value.Int {
		return BoolValue((*IntRange)(v.Ptr).Contains(int64(args[0].Data))), nil
	}
	elems, err := intRangeMaterialize("contains", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("contains", args, true)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// intRangeCount is count(...): how many elements match.
// PURE by contract
func intRangeCount(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("count", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("count", args, true)
	if err != nil {
		return Undefined, err
	}
	n := int64(0)
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			n++
		}
	}
	return IntValue(n), nil
}

// intRangeAny is any(...): does some element match?
// PURE by contract
func intRangeAny(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("any", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("any", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if hit {
			return True, nil
		}
	}
	return False, nil
}

// intRangeAll is all(...): does every element match? True on an empty range.
// PURE by contract
func intRangeAll(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("all", v)
	if err != nil {
		return Undefined, err
	}
	m, err := intRangeReadMatchArgs("all", args, false)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		hit, err := m.matches(vm, i, e)
		if err != nil {
			return Undefined, err
		}
		if !hit {
			return False, nil
		}
	}
	return True, nil
}

// intRangeForEach is for_each(f): a full pass whose callback result is ignored; returns the receiver, so it chains.
// PURE by contract
func intRangeForEach(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	fn, err := readElemCallback("for_each", args)
	if err != nil {
		return Undefined, err
	}
	elems, err := intRangeMaterialize("for_each", v)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		if _, err := callElem(vm, fn, i, IntValue(e)); err != nil {
			return Undefined, err
		}
	}
	return v, nil
}

// intRangeIndex is index(...): the first match.
// PURE by contract
func intRangeIndex(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return intRangeLocate(vm, "index", v, args, false)
}

// intRangeIndexLast is index_last(...): the last match.
// PURE by contract
func intRangeIndexLast(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	return intRangeLocate(vm, "index_last", v, args, true)
}

// PURE by contract
func intRangeJoin(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	elems, err := intRangeElements(name, v)
	if err != nil {
		return Undefined, err
	}
	if len(args) == 0 {
		s, err := joinElementsToString(elems, "")
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil
	}
	return joinSeqWithSep(elems, args[0], name)
}

// intRangeFirst is first([default]): the first element.
// PURE by contract
func intRangeFirst(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "first"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, _ := intRangeFirstLast(o, n)
	return IntValue(first), nil
}

// intRangeLast is last([default]): the last element.
// PURE by contract
func intRangeLast(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "last"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	_, last := intRangeFirstLast(o, n)
	return IntValue(last), nil
}

// intRangeMin is min([default]): the smallest element — one of the two ends.
// PURE by contract
func intRangeMin(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "min"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	return IntValue(min(first, last)), nil
}

// intRangeMax is max([default]): the largest element — one of the two ends.
// PURE by contract
func intRangeMax(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "max"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	return IntValue(max(first, last)), nil
}

// intRangeSum is sum([default]): the sum, in checked int arithmetic like the array member — overflow raises, never
// wraps.
// PURE by contract
func intRangeSum(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "sum"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	sum, err := intRangeProgressionSum(n, first, last)
	if err != nil {
		return Undefined, err
	}
	return IntValue(sum), nil
}

// intRangeAvg is avg([default]): the sum divided by the count — the same division the array member performs on
// int elements.
// PURE by contract
func intRangeAvg(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "avg"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return emptySeqResult(name, args)
	}
	first, last := intRangeFirstLast(o, n)
	sum, err := intRangeProgressionSum(n, first, last)
	if err != nil {
		return Undefined, err
	}
	return IntValue(sum).BinaryOp(tokens.Quo, IntValue(n))
}

// intRangeReduce is reduce(acc, f): folds the elements left to right.
// PURE by contract
func intRangeReduce(vm VM, v Value, _ member.ID, args []Value) (Value, error) {
	elems, err := intRangeMaterialize("reduce", v)
	if err != nil {
		return Undefined, err
	}
	acc, fn, err := readReduceArgs(args)
	if err != nil {
		return Undefined, err
	}
	for i, e := range elems {
		acc, err = callReduce(vm, fn, acc, i, IntValue(e))
		if err != nil {
			return Undefined, err
		}
	}
	return acc, nil
}

// intRangeReverse is reverse(): the same elements in the opposite order, as a range.
// PURE by contract
func intRangeReverse(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "reverse"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 {
		return v, nil
	}
	return intRangeReversed(name, o, n)
}

// intRangeSort is sort(): the elements ascending, as a range — the receiver itself when it already ascends.
// PURE by contract
func intRangeSort(_ VM, v Value, _ member.ID, args []Value) (Value, error) {
	const name = "sort"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	if n == 0 || o.Start <= o.Stop {
		return v, nil
	}
	return intRangeReversed(name, o, n)
}

// PURE by contract
func intRangeDedup(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	// the identity: an arithmetic progression never repeats
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(id.String(), "0", len(args))
	}
	return v, nil
}

// PURE by contract
func intRangeSlice(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// clamps like every slice (reading past the end is harmless); negative
	// indices count from the end. A closed form: a sub-progression is a range
	s, e, err := readSliceArgs(name, args)
	if err != nil {
		return Undefined, err
	}
	o := (*IntRange)(v.Ptr)
	si, ok := int64(0), true
	if s.Type != value.Undefined {
		if si, ok = s.AsInt(); !ok {
			return Undefined, errs.NewInvalidIndexTypeError(name, "int", s.TypeName())
		}
	}
	ei := int64(0)
	if e.Type != value.Undefined {
		if ei, ok = e.AsInt(); !ok {
			return Undefined, errs.NewInvalidIndexTypeError(name, "int", e.TypeName())
		}
	}
	si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, o.Len())
	return intRangeSub(o, si, ei-si), nil
}

// PURE by contract
func intRangeChunk(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	name := id.String()
	// array of ranges: the outer array holds no int elements, so nothing materialises
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	size, ok := args[0].AsInt()
	if !ok {
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
	}
	if size < 1 {
		return Undefined, errs.NewInvalidValueError("chunk size must be positive")
	}
	o := (*IntRange)(v.Ptr)
	n := o.Len()
	count := n / size
	if n%size != 0 {
		count++
	}
	if _, err := SeqAllocLen(name, count); err != nil {
		return Undefined, err
	}
	chunks := make([]Value, 0, count)
	for k := range count {
		at := k * size // < n, so it never overflows; at += size could
		chunks = append(chunks, intRangeSub(o, at, min(size, n-at)))
	}
	return NewArrayValue(chunks, false), nil
}
