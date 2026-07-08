package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/araddon/dateparse"
	"github.com/jokruger/dec128"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const stringTypeName = "string"

func NewStaticStringValue(s *string) Value {
	return Value{Type: value.String, Immutable: true, Ptr: unsafe.Pointer(s)}
}

func NewStringValue(s string) Value {
	return Value{Type: value.String, Immutable: true, Ptr: unsafe.Pointer(&s)}
}

// TypeString is a string type descriptor.
var TypeString = ValueTypeDescr{
	Name:         ConstHook(stringTypeName),                                               // PURE by contract
	String:       func(v Value) string { return strconv.Quote(*(*string)(v.Ptr)) },        // PURE by contract
	Format:       stringTypeFormat,                                                        // PURE by contract
	Interface:    func(v Value) any { return *(*string)(v.Ptr) },                          // PURE by contract
	EncodeJSON:   stringTypeEncodeJSON,                                                    // PURE by contract
	EncodeBinary: stringTypeEncodeBinary,                                                  // PURE by contract
	DecodeBinary: stringTypeDecodeBinary,                                                  // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) bool { return len(*(*string)(v.Ptr)) > 0 },                // PURE by contract
	IsIterable:   ConstHook(true),                                                         // PURE by contract
	Iterator:     stringTypeIterator,                                                      // PURE by contract (constructs fresh iterator)
	Equal:        stringTypeEqual,                                                         // PURE by contract
	Len:          func(v Value) int64 { return int64(len(*(*string)(v.Ptr))) },            // PURE by contract
	BinaryOp:     stringTypeBinaryOp,                                                      // PURE by contract
	MethodCall:   stringTypeMethodCall,                                                    // PURE by contract with higher-order rule caveat (see docs/purity.md)
	Access:       stringTypeAccess,                                                        // PURE by contract
	Contains:     stringTypeContains,                                                      // PURE by contract
	Slice:        stringTypeSlice,                                                         // PURE by contract
	SliceStep:    stringTypeSliceStep,                                                     // PURE by contract
	AsBool:       func(v Value) (bool, bool) { return conv.ParseBool(*(*string)(v.Ptr)) }, // PURE by contract
	AsInt:        stringTypeAsInt,                                                         // PURE by contract
	AsByte:       stringTypeAsByte,                                                        // PURE by contract
	AsFloat:      stringTypeAsFloat,                                                       // PURE by contract
	AsDecimal:    stringTypeAsDecimal,                                                     // PURE by contract
	AsTime:       stringTypeAsTime,                                                        // PURE by contract
	AsString:     func(v Value) (string, bool) { return *(*string)(v.Ptr), true },         // PURE by contract
	AsRunes:      func(v Value) ([]rune, bool) { return []rune(*(*string)(v.Ptr)), true }, // PURE by contract
	AsBytes:      func(v Value) ([]byte, bool) { return []byte(*(*string)(v.Ptr)), true }, // PURE by contract
	AsArray:      stringTypeAsArray,                                                       // PURE by contract
}

// PURE by contract
func stringTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	var b []byte
	b = EncodeString(b, *o)
	return b, nil
}

// PURE by contract
func stringTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*string)(v.Ptr)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(*o); err != nil {
		return nil, fmt.Errorf("string: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func stringTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("string: %w", err)
	}
	*v = NewStringValue(s)
	return nil
}

// PURE by contract
func stringTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	o := (*string)(v.Ptr)
	if sp.Verb == 'v' {
		return strconv.Quote(*o), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(stringTypeName, sp, fspec.AlignLeft), nil
	}
	return format.FormatStringLike(stringTypeName, sp, *o, false)
}

// PURE by contract
func stringTypeBinaryOp(v Value, rhs Value, op token.Token) (Value, error) {
	r, ok := rhs.AsString()
	if !ok {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
	}

	l := *(*string)(v.Ptr)
	switch op {
	case token.Add:
		return NewStringValue(l + r), nil
	case token.Less:
		return BoolValue(l < r), nil
	case token.LessEq:
		return BoolValue(l <= r), nil
	case token.Greater:
		return BoolValue(l > r), nil
	case token.GreaterEq:
		return BoolValue(l >= r), nil
	}

	return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), v.TypeName(), rhs.TypeName())
}

// PURE by contract
func stringTypeEqual(v Value, r Value) bool {
	t, ok := r.AsString()
	if !ok {
		return false
	}
	return *(*string)(v.Ptr) == t
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*string)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		// it is always immutable, so we can return the same value
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewBytesValue([]byte(*o), false), nil

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, utf8.RuneCountInString(*o))
		for i, r := range *o {
			rs[i] = r
		}
		return NewRunesValue(rs, false), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "bool":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := conv.ParseBool(*(*string)(v.Ptr))
		return BoolValue(b), nil

	case "float":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		f, _ := stringTypeAsFloat(v)
		return FloatValue(f), nil

	case "int":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		i, _ := stringTypeAsInt(v)
		return IntValue(i), nil

	case "byte":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		b, _ := stringTypeAsByte(v)
		return ByteValue(b), nil

	case "decimal":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		d, _ := stringTypeAsDecimal(v)
		return NewDecimalValue(d), nil

	case "time":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := stringTypeAsTime(v)
		return NewTimeValue(t), nil

	case "record":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, utf8.RuneCountInString(*o))
		for i, r := range *o {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return NewRecordValue(m, false), nil

	case "dict":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		m := make(map[string]Value, utf8.RuneCountInString(*o))
		for i, r := range *o {
			m[strconv.Itoa(i)] = RuneValue(r)
		}
		return NewDictValue(m, false), nil

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
		s, err := stringTypeFormat(v, sp)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(s), nil

	case "is_empty":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return BoolValue(len(*o) == 0), nil

	case "len":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return IntValue(int64(len(*o))), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(strings.ToLower(*o)), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(strings.ToUpper(*o)), nil

	case "contains":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return BoolValue(stringTypeContains(v, args[0])), nil

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return NewStringValue(strings.Trim(*o, " \t\n")), nil
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string", args[0].TypeName())
		}
		return NewStringValue(strings.Trim(*o, s)), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := []rune(*o)
		slices.Reverse(rs)
		return NewStringValue(string(rs)), nil

	case "filter":
		return stringFnFilter(vm, v, args)

	case "count":
		return stringFnCount(vm, v, args)

	case "all":
		return stringFnAll(vm, v, args)

	case "any":
		return stringFnAny(vm, v, args)

	case "for_each":
		return stringFnForEach(vm, v, args)

	case "find":
		return stringFnFind(vm, v, args)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		return NewStringValue(strings.Repeat(*o, n)), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		return joinSeqValueWithSepString(args[0], *o, name)

	case "split":
		return stringFnSplit(v, args)

	case "split_lines":
		return stringFnSplitLines(v, args)

	case "partition":
		return stringFnPartition(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE by contract
func stringTypeAccess(v Value, index Value, mode bc.Opcode) (Value, error) {
	if mode == bc.AccessIndex {
		i, ok := index.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("index access", "int", index.TypeName())
		}
		s := *(*string)(v.Ptr)
		i, ok = NormalizeIndex(i, int64(len(s)))
		if !ok {
			return Undefined, errs.NewIndexOutOfBoundsError("index access", int(i), len(s))
		}
		return ByteValue(s[i]), nil
	}

	return Undefined, errs.NewInvalidSelectorError(v.TypeName(), index.String())
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func stringTypeIterator(v Value) (Value, error) {
	o := (*string)(v.Ptr)
	return NewRunesIteratorValue([]rune(*o)), nil
}

// PURE by contract
func stringTypeAsInt(v Value) (int64, bool) {
	o := (*string)(v.Ptr)
	i, err := strconv.ParseInt(*o, 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

// PURE by contract
func stringTypeAsByte(v Value) (byte, bool) {
	o := (*string)(v.Ptr)
	i, err := strconv.ParseInt(*o, 10, 64)
	if err == nil {
		if i < 0 || i > 255 {
			return byte(i), false
		}
		return byte(i), true
	}
	return 0, false
}

// PURE by contract
func stringTypeAsFloat(v Value) (float64, bool) {
	o := (*string)(v.Ptr)
	f, err := strconv.ParseFloat(*o, 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

// PURE by contract
func stringTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*string)(v.Ptr)
	d := dec128.FromString(*o)
	return d, !d.IsNaN()
}

// PURE by contract
func stringTypeAsTime(v Value) (time.Time, bool) {
	o := (*string)(v.Ptr)
	val, err := dateparse.ParseAny(*o)
	if err != nil {
		return time.Time{}, false
	}
	return val, true
}

// PURE by contract
func stringTypeAsArray(v Value) ([]Value, bool) {
	o := (*string)(v.Ptr)
	arr := make([]Value, utf8.RuneCountInString(*o))
	for i, r := range *o {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

// PURE by contract
func stringTypeContains(v Value, e Value) bool {
	o := (*string)(v.Ptr)
	switch e.Type {
	case value.Rune:
		c := rune(e.Data)
		return strings.ContainsRune(*o, c)

	case value.String:
		return strings.Contains(*o, *(*string)(e.Ptr))

	default:
		c, ok := e.AsRune()
		if !ok {
			return false
		}
		return strings.ContainsRune(*o, c)
	}
}

// PURE by contract
func stringTypeSlice(v Value, s Value, e Value) (Value, error) {
	var si int64
	var ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(len(str))

	if s.Type != value.Undefined {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}

	if e.Type != value.Undefined {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	si, ei = NormalizeSliceBounds(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, l)
	return NewStringValue(str[si:ei]), nil
}

// PURE by contract
func stringTypeSliceStep(v Value, s Value, e Value, stepVal Value) (Value, error) {
	var si, ei int64
	var ok bool

	str := *(*string)(v.Ptr)
	l := int64(len(str))

	step, ok := stepVal.AsInt()
	if !ok {
		return Undefined, errs.NewInvalidIndexTypeError("slice step", "int", stepVal.TypeName())
	}
	if step == 0 {
		return Undefined, errs.NewSliceStepZeroError()
	}

	if s.Type != value.Undefined {
		si, ok = s.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", s.TypeName())
		}
	}
	if e.Type != value.Undefined {
		ei, ok = e.AsInt()
		if !ok {
			return Undefined, errs.NewInvalidIndexTypeError("slice", "int", e.TypeName())
		}
	}

	start, end := NormalizeSliceBoundsStep(si, s.Type != value.Undefined, ei, e.Type != value.Undefined, step, l)
	bs := []byte(str)
	result := make([]byte, 0, len(bs))
	if step > 0 {
		for i := start; i < end; i += step {
			result = append(result, bs[i])
		}
	} else {
		for i := start; i > end; i += step {
			result = append(result, bs[i])
		}
	}
	return NewStringValue(string(result)), nil
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnFilter(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("filter", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "non-variadic function", fn.TypeName())
	}

	var buf [2]Value
	o := (*string)(v.Ptr)
	filtered := make([]rune, 0, utf8.RuneCountInString(*o))

	switch fn.Arity() {
	case 1:
		for _, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return NewStringValue(string(filtered)), nil

	case 2:
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				filtered = append(filtered, v)
			}
		}
		return NewStringValue(string(filtered)), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("filter", "first", "f/1 or f/2", fn.TypeName())
	}
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnCount(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("count", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "non-variadic function", fn.TypeName())
	}

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		var count int64
		for _, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	case 2:
		var count int64
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				count++
			}
		}
		return IntValue(count), nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("count", "first", "f/1 or f/2", fn.TypeName())
	}
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnForEach(vm VM, v Value, args []Value) (Value, error) {
	fn, err := ForEachCallback(args)
	if err != nil {
		return Undefined, err
	}

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}

	case 2:
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return Undefined, nil
			}
		}
	}
	return Undefined, nil
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnFind(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("find", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "non-variadic function", fn.TypeName())
	}

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for i, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	case 2:
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return IntValue(int64(i)), nil
			}
		}
		return Undefined, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("find", "first", "f/1 or f/2", fn.TypeName())
	}
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnAll(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("all", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "non-variadic function", fn.TypeName())
	}

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	case 2:
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if !res.IsTrue() {
				return False, nil
			}
		}
		return True, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("all", "first", "f/1 or f/2", fn.TypeName())
	}
}

// PURE by contract with higher-order rule caveat (see docs/purity.md)
func stringFnAny(vm VM, v Value, args []Value) (Value, error) {
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError("any", "1", len(args))
	}

	fn := args[0]
	if !fn.IsCallable() || fn.IsVariadic() {
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "non-variadic function", fn.TypeName())
	}

	o := (*string)(v.Ptr)
	var buf [2]Value
	switch fn.Arity() {
	case 1:
		for _, v := range *o {
			buf[0] = RuneValue(v)
			res, err := fn.Call(vm, buf[:1])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	case 2:
		for i, v := range *o {
			buf[0] = IntValue(int64(i))
			buf[1] = RuneValue(v)
			res, err := fn.Call(vm, buf[:2])
			if err != nil {
				return Undefined, err
			}
			if res.IsTrue() {
				return True, nil
			}
		}
		return False, nil

	default:
		return Undefined, errs.NewInvalidArgumentTypeError("any", "first", "f/1 or f/2", fn.TypeName())
	}
}

// PURE by contract
func stringFnSplit(v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*string)(v.Ptr)
	var pieces []string
	if len(args) == 0 {
		pieces = splitStringWhitespace(*o)
	} else {
		sep, err := coerceSepToString(name, args[0])
		if err != nil {
			return Undefined, err
		}
		if sep == "" {
			return Undefined, fmt.Errorf("split separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitStringByLiteral(*o, sep, limit)
	}
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewStringValue(p)
	}
	return NewArrayValue(arr, false), nil
}

// PURE by contract
func stringFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*string)(v.Ptr)
	pieces := splitLinesString(*o)
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewStringValue(p)
	}
	return NewArrayValue(arr, false), nil
}

// PURE by contract
func stringFnPartition(v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToString(name, args[0])
	if err != nil {
		return Undefined, err
	}
	if sep == "" {
		return Undefined, fmt.Errorf("partition separator must not be empty")
	}
	s := *(*string)(v.Ptr)
	arr := make([]Value, 3)
	idx := strings.Index(s, sep)
	if idx < 0 {
		arr[0] = NewStringValue(s)
		arr[1] = EmptyString
		arr[2] = EmptyString
	} else {
		arr[0] = NewStringValue(s[:idx])
		arr[1] = NewStringValue(s[idx : idx+len(sep)])
		arr[2] = NewStringValue(s[idx+len(sep):])
	}
	return NewArrayValue(arr, false), nil
}
