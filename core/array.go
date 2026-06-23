package core

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unsafe"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
	"github.com/jokruger/kavun/internal/format"
)

const (
	arrayTypeName          = "array"
	immutableArrayTypeName = "immutable-array"
)

type Array = Seq[Value]

func NewArrayValue(arr []Value, immutable bool) Value {
	o := &Array{}
	o.Set(arr)
	return Value{Type: value.Array, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeArray = ValueTypeDescr{
	Name:         SeqNameHook(arrayTypeName, immutableArrayTypeName),
	String:       arrayTypeString,
	Format:       arrayTypeFormat,
	Interface:    arrayTypeInterface,
	EncodeJSON:   arrayTypeEncodeJSON,
	EncodeBinary: arrayTypeEncodeBinary,
	DecodeBinary: arrayTypeDecodeBinary,
	IsTrue:       func(v Value) bool { return len((*Array)(v.Ptr).Elements) > 0 },
	IsIterable:   ConstHook(true),
	Iterator:     arrayTypeIterator,
	Equal:        arrayTypeEqual,
	Clone:        arrayTypeClone,
	Len:          func(v Value) int64 { return int64(len((*Array)(v.Ptr).Elements)) },
	BinaryOp:     arrayTypeBinaryOp,
	MethodCall:   arrayTypeMethodCall,
	Access:       SeqAccessHook(RefValue, arrayTypeResolve),
	Assign:       SeqAssignHook(arrayTypeResolve, Value.AsValue, anyTypeName),
	Contains:     arrayTypeContains,
	Append:       arrayTypeAppend,
	Slice:        SeqSliceHook(NewArrayValue, arrayTypeResolve),
	SliceStep:    SeqSliceStepHook(NewArrayValue, arrayTypeResolve),
	AsBool:       func(v Value) (bool, bool) { return len((*Array)(v.Ptr).Elements) > 0, true },
	AsString:     arrayTypeAsString,
	AsRunes:      arrayTypeAsRunes,
	AsBytes:      arrayTypeAsBytes,
	AsArray:      func(v Value) ([]Value, bool) { return (*Array)(v.Ptr).Elements, true },
}

func arrayTypeResolve(v Value) *Array {
	return (*Array)(v.Ptr)
}

func arrayTypeString(v Value) string {
	o := (*Array)(v.Ptr)
	parts := make([]string, len(o.Elements))
	for i, e := range o.Elements {
		parts[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func arrayTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return arrayTypeString(v), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	if err := format.ValidateContainerSpec(arrayTypeName, sp); err != nil {
		return "", err
	}
	return fspec.ApplyGenerics(arrayTypeString(v), sp, fspec.AlignLeft), nil
}

func arrayTypeInterface(v Value) any {
	o := (*Array)(v.Ptr)
	res := make([]any, len(o.Elements))
	for i, val := range o.Elements {
		res[i] = val.Interface()
	}
	return res
}

func arrayTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)
	var b []byte
	b = append(b, '[')
	len1 := len(o.Elements) - 1
	for idx, elem := range o.Elements {
		eb, err := elem.EncodeJSON()
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

func arrayTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Array)(v.Ptr)

	b := binary.AppendUint64(nil, uint64(len(o.Elements)))
	for i, elem := range o.Elements {
		eb, err := elem.EncodeBinary()
		if err != nil {
			return nil, fmt.Errorf("array element at index %d: %w", i, err)
		}
		b = binary.AppendBytes(b, eb)
	}

	return b, nil
}

func arrayTypeDecodeBinary(v *Value, data []byte) error {
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
		if err := arr[i].DecodeBinary(eb); err != nil {
			return fmt.Errorf("array element at index %d: %w", i, err)
		}
	}
	if offset != len(data) {
		return fmt.Errorf("array: trailing %d bytes", len(data)-offset)
	}

	*v = NewArrayValue(arr, v.Immutable)
	return nil
}

func arrayTypeIterator(v Value) (Value, error) {
	return NewArrayIteratorValue((*Array)(v.Ptr).Elements), nil
}

func arrayTypeEqual(v Value, r Value) bool {
	if r.Type != value.Array {
		return false
	}

	la := (*Array)(v.Ptr).Elements
	ra := (*Array)(r.Ptr).Elements

	if len(la) != len(ra) {
		return false
	}

	for i, e := range la {
		if !e.Equal(ra[i]) {
			return false
		}
	}

	return true
}

func arrayTypeClone(v Value) (Value, error) {
	// Deep copy the array (and make it mutable) and its elements
	o := (*Array)(v.Ptr)
	c := make([]Value, len(o.Elements))
	for i, e := range o.Elements {
		t, err := e.Clone()
		if err != nil {
			return Undefined, err
		}
		c[i] = t
	}
	return NewArrayValue(c, false), nil
}

func arrayTypeBinaryOp(v Value, r Value, op token.Token) (Value, error) {
	if r.Type != value.Array {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
	}

	la := (*Array)(v.Ptr)
	ra := (*Array)(r.Ptr)
	switch op {
	case token.Add:
		return NewArrayValue(append(la.Elements, ra.Elements...), false), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), r.TypeName())
}

func arrayTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Array)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return arrayTypeClone(v)

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		bs := make([]byte, len(o.Elements))
		for i, e := range o.Elements {
			bs[i], _ = e.AsByte()
		}
		return NewBytesValue(bs, false), nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := make([]rune, len(o.Elements))
		for i, e := range o.Elements {
			r[i], _ = e.AsRune()
		}
		return NewStringValue(string(r)), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := make(map[string]Value, len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return NewRecordValue(r, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		r := make(map[string]Value, len(o.Elements))
		for i, v := range o.Elements {
			r[strconv.Itoa(i)] = v
		}
		return NewDictValue(r, false), nil

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
		s, err := arrayTypeFormat(v, sp)
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
		return BoolValue(arrayTypeContains(v, args[0])), nil

	case "min":
		return arrayFnMin(v, args)

	case "max":
		return arrayFnMax(v, args)

	case "sum":
		return arrayFnSum(v, args)

	case "avg":
		return arrayFnAvg(v, args)

	case "sort":
		return arrayFnSort(v, args)

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError("dedup", "0", len(args))
		}
		o := (*Array)(v.Ptr)
		out := make([]Value, 0, len(o.Elements))
		for i, e := range o.Elements {
			if i == 0 || !out[len(out)-1].Equal(e) {
				out = append(out, e)
			}
		}
		return NewArrayValue(out, false), nil

	case "unique":
		return arrayFnUnique(v, args)

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		o := (*Array)(v.Ptr)
		n := len(o.Elements)
		t := make([]Value, n)
		for i, x := range o.Elements {
			t[n-1-i] = x
		}
		return NewArrayValue(t, false), nil

	case "filter":
		return SeqFilter(vm, v, args, RefValue, NewArrayValue, arrayTypeResolve)

	case "count":
		return SeqCount(vm, v, args, RefValue, arrayTypeResolve)

	case "all":
		return SeqAll(vm, v, args, RefValue, arrayTypeResolve)

	case "any":
		return SeqAny(vm, v, args, RefValue, arrayTypeResolve)

	case "map":
		return SeqMap(vm, v, args, RefValue, arrayTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, RefValue, arrayTypeResolve)

	case "for_each":
		return SeqForEach(vm, v, args, RefValue, arrayTypeResolve)

	case "find":
		return SeqFind(vm, v, args, RefValue, arrayTypeResolve)

	case "chunk":
		return SeqChunk(v, args, NewArrayValue, arrayTypeResolve)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := make([]Value, n*sl)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return NewArrayValue(out, false), nil

	case "join":
		return arrayFnJoin(v, args)

	case "flatten":
		return arrayFnFlatten(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

func arrayTypeContains(v Value, e Value) bool {
	o := (*Array)(v.Ptr)
	switch e.Type {
	case value.Array:
		t := (*Array)(e.Ptr)
		if len(t.Elements) == 0 {
			return true
		}
		if len(o.Elements) < len(t.Elements) {
			return false
		}
		for i := range o.Elements {
			if o.Elements[i].Equal(t.Elements[0]) {
				match := true
				for j := 1; j < len(t.Elements); j++ {
					if i+j >= len(o.Elements) || !o.Elements[i+j].Equal(t.Elements[j]) {
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
			if o.Elements[i].Equal(e) {
				return true
			}
		}
		return false
	}
}

func arrayTypeAppend(v Value, args []Value) (Value, error) {
	return NewArrayValue(append((*Array)(v.Ptr).Elements, args...), false), nil
}

func arrayTypeAsString(v Value) (string, bool) {
	rs, ok := arrayTypeAsRunes(v)
	if !ok {
		return "", false
	}
	return string(rs), true
}

func arrayTypeAsRunes(v Value) ([]rune, bool) {
	o := (*Array)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	for i, e := range o.Elements {
		r, ok := e.AsInt()
		if !ok || r < 0 || r > unicode.MaxRune {
			return nil, false
		}
		rs[i] = rune(r)
	}
	return rs, true
}

func arrayTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Array)(v.Ptr)
	bs := make([]byte, len(o.Elements))
	for i, e := range o.Elements {
		b, ok := e.AsInt()
		if !ok || b < 0 || b > 255 {
			return nil, false
		}
		bs[i] = byte(b)
	}
	return bs, true
}

func arrayFnSort(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sort", "0", len(args))
	}

	var err error
	o := (*Array)(v.Ptr)
	t := make([]Value, len(o.Elements))
	copy(t, o.Elements)
	slices.SortFunc(t, func(x, y Value) int {
		less, e := x.BinaryOp(token.Less, y)
		if e != nil {
			err = e
			return 0
		}
		if !less.IsTrue() {
			if x.Equal(y) {
				return 0
			}
			return 1
		}
		return -1
	})
	if err != nil {
		return Undefined, err
	}

	return NewArrayValue(t, false), nil
}

// unique returns a new array with duplicate elements removed, regardless of their position in the array. This is less
// efficient than dedup, but does not require the input array to be sorted.
func arrayFnUnique(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("unique", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	out := make([]Value, 0, len(o.Elements))
	for _, e := range o.Elements {
		seen := false
		for _, u := range out {
			if u.Equal(e) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, e)
		}
	}

	return NewArrayValue(out, false), nil
}

func arrayFnMin(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("min", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		less, err := o.Elements[i].BinaryOp(token.Less, e)
		if err != nil {
			return Undefined, err
		}
		if less.IsTrue() {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnMax(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("max", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	e := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		greater, err := o.Elements[i].BinaryOp(token.Greater, e)
		if err != nil {
			return Undefined, err
		}
		if greater.IsTrue() {
			e = o.Elements[i]
		}
	}

	return e, nil
}

func arrayFnSum(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("sum", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	s := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		s, err = s.BinaryOp(token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	return s, nil
}

func arrayFnAvg(v Value, args []Value) (Value, error) {
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError("avg", "0", len(args))
	}

	o := (*Array)(v.Ptr)
	if len(o.Elements) == 0 {
		return Undefined, nil
	}

	var err error
	sum := o.Elements[0]
	for i := 1; i < len(o.Elements); i++ {
		sum, err = sum.BinaryOp(token.Add, o.Elements[i])
		if err != nil {
			return Undefined, err
		}
	}

	length := IntValue(int64(len(o.Elements)))
	avg, err := sum.BinaryOp(token.Quo, length)
	if err != nil {
		return Undefined, err
	}

	return avg, nil
}

// arrayFnJoin implements `array.join(sep)`.
// sep types: string | runes | byte | rune.
// Result type follows sep: string→string, runes→runes, byte→bytes, rune→runes.
// With no argument, defaults to empty string separator.
func arrayFnJoin(v Value, args []Value) (Value, error) {
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError("join", "0 or 1", len(args))
	}
	o := (*Array)(v.Ptr)
	if len(args) == 0 {
		s, err := joinElementsToString(o.Elements, "")
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil
	}
	return joinSeqWithSep(o.Elements, args[0], "join")
}

// joinSeqWithSep performs the join given pre-resolved seq elements and a separator value.
// Returns a value whose type is determined by the sep type.
func joinSeqWithSep(elems []Value, sep Value, name string) (Value, error) {
	switch sep.Type {
	case value.String:
		s, err := joinElementsToString(elems, *(*string)(sep.Ptr))
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case value.Runes:
		s, err := joinElementsToString(elems, string((*Runes)(sep.Ptr).Elements))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case value.Rune:
		s, err := joinElementsToString(elems, string(rune(sep.Data)))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case value.Byte:
		s, err := joinElementsToString(elems, string([]byte{byte(sep.Data)}))
		if err != nil {
			return Undefined, err
		}
		return NewBytesValue([]byte(s), false), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string, runes, byte, or rune", sep.TypeName())
	}
}

func arrayFnFlatten(v Value, args []Value) (Value, error) {
	const name = "flatten"
	if len(args) > 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
	}
	depth := 1
	if len(args) == 1 {
		d, ok := args[0].AsInt()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "int", args[0].TypeName())
		}
		if d < 0 {
			depth = -1
		} else {
			depth = int(d)
		}
	}
	o := (*Array)(v.Ptr)
	out := make([]Value, 0, len(o.Elements))
	out = flattenAppend(out, o.Elements, depth)
	arr := make([]Value, len(out))
	copy(arr, out)
	return NewArrayValue(arr, false), nil
}

// flattenAppend appends each element of src to dst, unwrapping nested arrays up to `depth` levels.
// depth == 0 means no unwrapping (shallow copy).
// depth < 0 means unbounded (fully recursive).
func flattenAppend(dst []Value, src []Value, depth int) []Value {
	if depth == 0 {
		return append(dst, src...)
	}
	next := depth
	if next > 0 {
		next--
	}
	for _, e := range src {
		if e.Type == value.Array {
			inner := (*Array)(e.Ptr).Elements
			dst = flattenAppend(dst, inner, next)
		} else {
			dst = append(dst, e)
		}
	}
	return dst
}
