package core

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
	"github.com/jokruger/kavun/token"
)

const (
	arrayTypeName          = "array"
	immutableArrayTypeName = "immutable-array"
)

type Array = Seq[Value]

var TypeArray = ValueTypeDescr{
	Pin:          func(a *Arena, v Value) { a.PinArrayValue(v) },
	Retain:       func(a *Arena, v Value) { a.RetainArrayValue(v) },
	Release:      func(a *Arena, v Value) { a.ReleaseArrayValue(v) },
	Name:         SeqNameHook(arrayTypeName, immutableArrayTypeName),
	String:       arrayTypeString,
	Format:       arrayTypeFormat,
	Interface:    arrayTypeInterface,
	EncodeJSON:   arrayTypeEncodeJSON,
	EncodeBinary: arrayTypeEncodeBinary,
	DecodeBinary: arrayTypeDecodeBinary,
	IsTrue:       func(a *Arena, v Value) bool { return len(a.ResolveArrayValue(v).Elements) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     arrayTypeIterator,
	Equal:        arrayTypeEqual,
	Clone:        arrayTypeClone,
	Len:          func(a *Arena, v Value) int64 { return int64(len(a.ResolveArrayValue(v).Elements)) },
	BinaryOp:     arrayTypeBinaryOp,
	MethodCall:   arrayTypeMethodCall,
	Access:       SeqAccessHook(RefValue, arrayTypeResolve),
	Assign:       SeqAssignHook(arrayTypeResolve, Value.AsValue, Value.Pin, anyTypeName),
	Contains:     arrayTypeContains,
	Append:       arrayTypeAppend,
	Slice:        SeqSliceHook(ArenaNewArrayValue, arrayTypeResolve),
	SliceStep:    SeqSliceStepHook(ArenaNewArray, ArenaNewArrayValue, arrayTypeResolve),
	AsBool:       func(a *Arena, v Value) (bool, bool) { return len(a.ResolveArrayValue(v).Elements) > 0, true },
	AsString:     arrayTypeAsString,
	AsRunes:      arrayTypeAsRunes,
	AsBytes:      arrayTypeAsBytes,
	AsArray:      func(a *Arena, v Value) ([]Value, bool) { return a.ResolveArrayValue(v).Elements, true },
}

func arrayTypeResolve(a *Arena, v Value) *Array {
	return a.ResolveArrayValue(v)
}

func arrayTypeString(a *Arena, v Value) string {
	o := a.ResolveArrayValue(v)
	parts := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		parts[i] = e.String(a)
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func arrayTypeFormat(a *Arena, v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return arrayTypeString(a, v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(a), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(arrayTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(arrayTypeString(a, v), sp, fspec.AlignLeft), nil
}

func arrayTypeInterface(a *Arena, v Value) any {
	o := a.ResolveArrayValue(v)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface(a)
	}
	return res
}

func arrayTypeEncodeJSON(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveArrayValue(v)
	var b []byte
	b = append(b, '[')
	len1 := len(o.Elements) - 1
	for idx, elem := range o.Elements {
		eb, err := elem.EncodeJSON(a)
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", idx, err)
		}
		b = append(b, eb...)
		if idx < len1 {
			b = append(b, ',')
		}
	}
	b = append(b, ']')
	return b, nil
}

func arrayTypeEncodeBinary(a *Arena, v Value) ([]byte, error) {
	o := a.ResolveArrayValue(v)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for i, elem := range o.Elements {
		eb, err := elem.EncodeBinary(a)
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", i, err)
		}
		b = binary.AppendBytes(b, eb)
	}

	return b, nil
}

func arrayTypeDecodeBinary(a *Arena, v *Value, data []byte) error {
	offset := 0
	count, err := binary.ReadUint64(data, &offset, "array (elements count)")
	if err != nil {
		return err
	}

	arr := make([]Value, int(count))
	for i := range arr {
		eb, err := binary.ReadBytes(data, &offset, fmt.Sprintf("array element at index %d", i))
		if err != nil {
			return err
		}
		if err := arr[i].DecodeBinary(a, eb); err != nil {
			return fmt.Errorf("array element at index %d: %w", i, err)
		}
	}
	if offset != len(data) {
		return fmt.Errorf("array: trailing %d bytes", len(data)-offset)
	}

	o, err := a.NewArrayValue(arr, v.Immutable)
	if err != nil {
		return err
	}
	// we are not releasing old value here because it should be managed by caller Value.DecodeBinary
	*v = o
	return nil
}

func arrayTypeIterator(a *Arena, v Value) (Value, error) {
	o := a.ResolveArrayValue(v)
	return a.NewArrayIteratorValue(o.Elements)
}

func arrayTypeEqual(a *Arena, v Value, r Value) bool {
	if r.Type != VT_ARRAY {
		return false
	}

	la := a.ResolveArrayValue(v).Elements
	ra := a.ResolveArrayValue(r).Elements

	if len(la) != len(ra) {
		return false
	}

	for i, e := range la {
		if !e.Equal(a, ra[i]) {
			return false
		}
	}

	return true
}

func arrayTypeClone(a *Arena, v Value) (Value, error) {
	// Deep copy the array (and make it mutable) and its elements
	o := a.ResolveArrayValue(v)
	c := a.NewArray(len(o.Elements), true)
	for i, e := range o.Elements {
		t, err := e.Clone(a)
		if err != nil {
			return Undefined, err
		}
		t.Pin(a)
		c[i] = t
	}
	return a.NewArrayValue(c, false)
}

func arrayTypeBinaryOp(a *Arena, v Value, r Value, op token.Token) (Value, error) {
	if r.Type != VT_ARRAY {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), r.TypeName(a))
	}

	la := a.ResolveArrayValue(v)
	ra := a.ResolveArrayValue(r)
	switch op {
	case token.Add:
		return a.NewArrayValue(append(la.Elements, ra.Elements...), false)
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(a), r.TypeName(a))
}

func arrayTypeMethodCall(a *Arena, vm VM, v Value, name string, args []Value) (Value, error) {
	o := a.ResolveArrayValue(v)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return arrayTypeClone(a, v)

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		v.Retain(a)
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		bs := a.NewBytes(len(o.Elements), true)
		for i, e := range o.Elements {
			bs[i], _ = e.AsByte(a)
		}
		return a.NewBytesValue(bs, false)

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := a.NewRunes(len(o.Elements), true)
		for i, e := range o.Elements {
			r[i], _ = e.AsRune(a)
		}
		return a.NewStringValue(string(r))

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := a.NewDict(len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return a.NewRecordValue(r, false)

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := a.NewDict(len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return a.NewDictValue(r, false)

	case "format":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		f := ""
		if len(args) == 1 {
			var ok bool
			f, ok = args[0].AsString(a)
			if !ok {
				return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName(a))
			}
		}
		sp, err := fspec.Parse(f)
		if err != nil {
			return Undefined, err
		}
		s, err := arrayTypeFormat(a, v, sp)
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

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

	case "first":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return o.Elements[0], nil

	case "last":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if len(o.Elements) == 0 {
			return Undefined, nil
		}
		return o.Elements[len(o.Elements)-1], nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(arrayTypeContains(a, v, args[0])), nil

	case "min":
		return arrayFnMin(a, v, args)

	case "max":
		return arrayFnMax(a, v, args)

	case "sum":
		return arrayFnSum(a, v, args)

	case "avg":
		return arrayFnAvg(a, v, args)

	case "sort":
		return arrayFnSort(a, v, args)

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
		}
		o := a.ResolveArrayValue(v)
		out := a.NewArray(len(o.Elements), false)
		for i, e := range o.Elements {
			if i == 0 || !out[len(out)-1].Equal(a, e) {
				out = append(out, e)
			}
		}
		return a.NewArrayValue(out, false)

	case "unique":
		return arrayFnUnique(a, v, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := a.ResolveArrayValue(v)
		n := len(o.Elements)
		t := a.NewArray(n, true)
		for i, x := range o.Elements {
			t[n-1-i] = x
		}
		return a.NewArrayValue(t, false)

	case "filter":
		return SeqFilter(a, vm, v, args, RefValue, ArenaNewArray, ArenaNewArrayValue, arrayTypeResolve)

	case "count":
		return SeqCount(a, vm, v, args, RefValue, arrayTypeResolve)

	case "all":
		return SeqAll(a, vm, v, args, RefValue, arrayTypeResolve)

	case "any":
		return SeqAny(a, vm, v, args, RefValue, arrayTypeResolve)

	case "map":
		return SeqMap(a, vm, v, args, RefValue, arrayTypeResolve)

	case "reduce":
		return SeqReduce(a, vm, v, args, RefValue, arrayTypeResolve)

	case "for_each":
		return SeqForEach(a, vm, v, args, RefValue, arrayTypeResolve)

	case "find":
		return SeqFind(a, vm, v, args, RefValue, arrayTypeResolve)

	case "chunk":
		return SeqChunk(a, v, args, ArenaNewArray, ArenaNewArrayValue, arrayTypeResolve)

	case "repeat":
		n, err := parseRepeatCount(a, name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := a.NewArray(n*sl, true)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return a.NewArrayValue(out, false)

	case "join":
		return arrayFnJoin(a, v, args)

	case "flatten":
		return arrayFnFlatten(a, v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName(a))
	}
}

func arrayTypeContains(a *Arena, v Value, e Value) bool {
	o := a.ResolveArrayValue(v)
	switch e.Type {
	case VT_ARRAY:
		t := a.ResolveArrayValue(e)
		if len(t.Elements) == 0 {
			return true
		}
		if len(o.Elements) < len(t.Elements) {
			return false
		}
		for i := range o.Elements {
			if o.Elements[i].Equal(a, t.Elements[0]) {
				match := true
				for j := 1; j < len(t.Elements); j++ {
					if i+j >= len(o.Elements) || !o.Elements[i+j].Equal(a, t.Elements[j]) {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
		return false

	default:
		for i := range o.Elements {
			if o.Elements[i].Equal(a, e) {
				return true
			}
		}
		return false
	}
}

func arrayTypeAppend(a *Arena, v Value, args []Value) (Value, error) {
	o := a.ResolveArrayValue(v)
	for _, arg := range args {
		arg.Pin(a) // mark appended values as unmanaged because they are now also owned by the array
	}
	return a.NewArrayValue(append(o.Elements, args...), false)
}

func arrayTypeAsString(a *Arena, v Value) (string, bool) {
	rs, ok := arrayTypeAsRunes(a, v)
	if !ok {
		return "", false
	}
	return string(rs), true
}

func arrayTypeAsRunes(a *Arena, v Value) ([]rune, bool) {
	o := a.ResolveArrayValue(v)
	rs := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		r, ok := e.AsInt(a)
		if !ok || r < 0 || r > unicode.MaxRune {
			return nil, false
		}
		rs[i] = rune(r)
	}
	return rs, true
}

func arrayTypeAsBytes(a *Arena, v Value) ([]byte, bool) {
	o := a.ResolveArrayValue(v)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt(a)
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

func arrayFnSort(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sort", "0", len(args))
	}

	var err error
	o := a.ResolveArrayValue(v)
	t := a.NewArray(len(o.Elements), true)
	copy(t, o.Elements)
	slices.SortFunc(t, func(x, y Value) int {
		less, e := x.BinaryOp(a, token.Less, y)
		if e != nil {
			err = e
			return 0
		}
		if !less.IsTrue(a) {
			if x.Equal(a, y) {
				return 0
			}
			return 1
		}
		return -1
	})
	if err != nil {
		return Undefined, err
	}

	return a.NewArrayValue(t, false)
}

// unique returns a new array with duplicate elements removed, regardless of their position in the array. This is less
// efficient than dedup, but does not require the input array to be sorted.
func arrayFnUnique(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}

	o := a.ResolveArrayValue(v)
	out := a.NewArray(len(o.Elements), false)
	for _, e := range o.Elements {
		seen := false
		for _, u := range out {
			if u.Equal(a, e) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, e)
		}
	}

	return a.NewArrayValue(out, false)
}

func arrayFnMin(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("min", "0", len(args))
	}

	o := a.ResolveArrayValue(v)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		less, err := o.Elements[i].BinaryOp(a, token.Less, e)
		if err != nil {
			return Undefined, err
		}
		if less.IsTrue(a) {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnMax(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("max", "0", len(args))
	}

	o := a.ResolveArrayValue(v)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		greater, err := o.Elements[i].BinaryOp(a, token.Greater, e)
		if err != nil {
			return Undefined, err
		}
		if greater.IsTrue(a) {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnSum(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}

	o := a.ResolveArrayValue(v)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	s := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		s, err = s.BinaryOp(a, token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	return s, nil
}

func arrayFnAvg(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}

	o := a.ResolveArrayValue(v)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	sum := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		sum, err = sum.BinaryOp(a, token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	length := IntValue(int64(len(o.Elements)))
	avg, err := sum.BinaryOp(a, token.Quo, length)
	if err != nil {
		return Undefined, err
	}

	return avg, nil
}

// arrayFnJoin implements `array.join(sep)`.
// sep types: string | runes | byte | rune.
// Result type follows sep: string→string, runes→runes, byte→bytes, rune→runes.
// With no argument, defaults to empty string separator.
func arrayFnJoin(a *Arena, v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("join", "0 or 1", len(args))
	}
	o := a.ResolveArrayValue(v)
	if len(args) == 0 {
		s, err := joinElementsToString(a, o.Elements, "")
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)
	}
	return joinSeqWithSep(a, o.Elements, args[0], "join")
}

// joinSeqWithSep performs the join given pre-resolved seq elements and a separator value.
// Returns a value whose type is determined by the sep type.
func joinSeqWithSep(a *Arena, elems []Value, sep Value, name string) (Value, error) {
	switch sep.Type {
	case VT_STRING:
		s, err := joinElementsToString(a, elems, *a.ResolveStringValue(sep))
		if err != nil {
			return Undefined, err
		}
		return a.NewStringValue(s)

	case VT_RUNES:
		s, err := joinElementsToString(a, elems, string(a.ResolveRunesValue(sep).Elements))
		if err != nil {
			return Undefined, err
		}
		return a.NewRunesValue([]rune(s), false)

	case VT_RUNE:
		s, err := joinElementsToString(a, elems, string(rune(sep.Data)))
		if err != nil {
			return Undefined, err
		}
		return a.NewRunesValue([]rune(s), false)

	case VT_BYTE:
		s, err := joinElementsToString(a, elems, string([]byte{byte(sep.Data)}))
		if err != nil {
			return Undefined, err
		}
		return a.NewBytesValue([]byte(s), false)

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes, byte, or rune", sep.TypeName(a))
	}
}

func arrayFnFlatten(a *Arena, v Value, args []Value) (Value, error) {
	const name = "flatten"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	depth := 1
	if len(args) == 1 {
		d, ok := args[0].AsInt(a)
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName(a))
		}
		if d < 0 {
			depth = -1
		} else {
			depth = int(d)
		}
	}
	o := a.ResolveArrayValue(v)
	out := make([]Value, 0, len(o.Elements))
	out = flattenAppend(a, out, o.Elements, depth)
	arr := a.NewArray(len(out), true)
	copy(arr, out)
	return a.NewArrayValue(arr, false)
}

// flattenAppend appends each element of src to dst, unwrapping nested arrays up to `depth` levels.
// depth == 0 means no unwrapping (shallow copy).
// depth < 0 means unbounded (fully recursive).
func flattenAppend(a *Arena, dst []Value, src []Value, depth int) []Value {
	if depth == 0 {
		return append(dst, src...)
	}
	next := depth
	if next > 0 {
		next--
	}
	for _, e := range src {
		if e.Type == VT_ARRAY {
			inner := a.ResolveArrayValue(e).Elements
			dst = flattenAppend(a, dst, inner, next)
		} else {
			dst = append(dst, e)
		}
	}
	return dst
}
