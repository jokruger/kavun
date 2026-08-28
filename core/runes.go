package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/jokruger/dec128"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/conv"
	"github.com/jokruger/kavun/internal/format"
)

const (
	runesTypeName          = "runes"
	immutableRunesTypeName = "immutable-runes"
)

type Runes = Seq[rune]

func NewStaticRunesValue(r *Runes) Value {
	return Value{Type: value.Runes, Immutable: true, Ptr: unsafe.Pointer(r)}
}

func NewRunesValue(r []rune, immutable bool) Value {
	o := &Runes{}
	o.Set(r)
	return Value{Type: value.Runes, Immutable: immutable, Ptr: unsafe.Pointer(o)}
}

var TypeRunes = ValueTypeDescr{
	Name:         SeqNameHook(runesTypeName, immutableRunesTypeName),                                    // PURE by contract
	String:       func(v Value) string { return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)) }, // PURE by contract
	Format:       runesTypeFormat,                                                                       // PURE by contract
	Interface:    func(v Value) any { return (*Runes)(v.Ptr).Elements },                                 // PURE by contract
	EncodeJSON:   runesTypeEncodeJSON,                                                                   // PURE by contract
	EncodeBinary: runesTypeEncodeBinary,                                                                 // PURE by contract
	DecodeBinary: runesTypeDecodeBinary,                                                                 // IMPURE by contract (mutates target)
	IsTrue:       func(v Value) (bool, error) { return len((*Runes)(v.Ptr).Elements) > 0, nil },         // PURE by contract
	IsIterable:   ConstHook(true),                                                                       // PURE by contract
	Iterator:     runesTypeIterator,                                                                     // PURE by contract (constructs fresh iterator)
	Copy:         runesTypeCopy,                                                                         // PURE by contract
	Len:          func(v Value) int64 { return int64(len((*Runes)(v.Ptr).Elements)) },                   // PURE by contract
	Equal:        runesTypeEqual,                                                                        // PURE by contract
	BinaryOp:     runesTypeBinaryOp,                                                                     // PURE by contract
	MethodCall:   runesTypeMethodCall,                                                                   // METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
	Access:       SeqAccessHook(RuneValue, runesTypeResolve),                                            // PURE by contract
	Assign:       SeqAssignHook(runesTypeResolve, Value.AsRune, runeTypeName),                           // IMPURE by contract
	Append:       runesTypeAppend,                                                                       // MUTATE-DEPENDENT by contract (see ValueTypeDescr.Append)
	Contains:     runesTypeContains,                                                                     // PURE by contract
	Slice:        SeqSliceHook(NewRunesValue, runesTypeResolve),                                         // PURE by contract
	SliceStep:    SeqSliceStepHook(NewRunesValue, runesTypeResolve),                                     // PURE by contract
	AsBool:       runesTypeAsBool,                                                                       // PURE by contract
	AsInt:        runesTypeAsInt,                                                                        // PURE by contract
	AsFloat:      runesTypeAsFloat,                                                                      // PURE by contract
	AsDecimal:    runesTypeAsDecimal,                                                                    // PURE by contract
	AsTime:       runesTypeAsTime,                                                                       // PURE by contract
	AsString:     func(v Value) (string, bool) { return string((*Runes)(v.Ptr).Elements), true },        // PURE by contract
	AsRunes:      func(v Value) ([]rune, bool) { return (*Runes)(v.Ptr).Elements, true },                // PURE by contract
	AsBytes:      runesTypeAsBytes,                                                                      // PURE by contract
	AsArray:      runesTypeAsArray,                                                                      // PURE by contract

	// _in_place are the mutating methods; every other method, including append/splice, is pure. Higher-order
	// methods (filter/count/all/any/for_each/find/map/reduce) are gated the same way as string's.
	IsMethodPure: func(name string) bool { return !strings.HasSuffix(name, "_in_place") },
}

// runesEncodeMatchArg: acceptance on a symbol receiver — text content as
// SYMBOLS. rune/ASCII byte/in-range int are the element class; string/runes/
// valid-UTF-8 bytes are the run class.
func runesEncodeMatchArg(name string, a Value) ([]rune, bool, error) {
	switch a.Type {
	case value.Rune:
		return []rune{rune(a.Data)}, true, nil
	case value.Byte:
		if a.Data > 0x7F {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("(%s) an octet reads as one symbol only in [0x00, 0x7F] (ASCII), got %d", name, a.Data))
		}
		return []rune{rune(a.Data)}, true, nil
	case value.Int:
		r, ok := a.AsRune()
		if !ok {
			return nil, false, errs.NewInvalidValueError(fmt.Sprintf("(%s) an int reads as one symbol and must be a valid code point, got %d", name, int64(a.Data)))
		}
		return []rune{r}, true, nil
	case value.String, value.Runes:
		rs, _ := a.AsRunes()
		return rs, false, nil
	case value.Bytes:
		b := (*Bytes)(a.Ptr).Elements
		if !utf8.Valid(b) {
			return nil, false, errs.NewInvalidValueError("(" + name + ") the bytes argument is not valid UTF-8")
		}
		return []rune(string(b)), false, nil
	}
	return nil, false, errs.NewInvalidArgumentTypeError(name, "argument", "text content (symbols, octets, or text)", a.TypeName())
}

func runesTypeResolve(v Value) *Runes {
	return (*Runes)(v.Ptr)
}

// PURE by contract
func runesTypeEncodeJSON(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	var b []byte
	b = EncodeString(b, string(o.Elements))
	return b, nil
}

// PURE by contract
func runesTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Runes)(v.Ptr)
	s := string(o.Elements)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return nil, fmt.Errorf("runes: %w", err)
	}
	return buf.Bytes(), nil
}

// IMPURE by contract (mutates target)
func runesTypeDecodeBinary(v *Value, data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var s string
	if err := dec.Decode(&s); err != nil {
		return fmt.Errorf("runes: %w", err)
	}
	*v = NewRunesValue([]rune(s), v.Immutable)
	return nil
}

// PURE by contract
func runesTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.Verb == 'v' {
		return "u" + strconv.Quote(string((*Runes)(v.Ptr).Elements)), nil
	}
	if sp.Verb == 'T' {
		return fspec.ApplyGenerics(v.TypeName(), sp, fspec.AlignLeft), nil
	}
	o := (*Runes)(v.Ptr)
	return format.FormatStringLike("runes", sp, string(o.Elements), false)
}

// runesAppendItems flattens append's variadic args (rune or runes values) into a single []rune, using methodName
// in any argument-type error so it reads correctly whether called from append() or append_in_place().
func runesAppendItems(args []Value, methodName string) ([]rune, error) {
	items := make([]rune, 0, len(args))
	for i, arg := range args {
		switch arg.Type {
		case value.Runes:
			items = append(items, (*Runes)(arg.Ptr).Elements...)
		default:
			c, ok := arg.AsRune()
			if !ok {
				return nil, errs.NewInvalidArgumentTypeError(methodName, fmt.Sprintf("%d", i+1), "rune or runes", arg.TypeName())
			}
			items = append(items, c)
		}
	}
	return items, nil
}

// mutate=true: IMPURE, mutates the receiver's own backing struct in place via Set (append_in_place()) — reuses
// spare capacity or reallocates exactly like Go's append, visible to every other live alias into this body.
// Rejects an immutable receiver. Not folded by the optimizer. mutate=false: PURE, returns a fresh, independent
// runes value with the items appended (append()) — never touches the receiver's backing storage, works
// regardless of the receiver's mutability. Both accept zero item arguments as a legal no-op. See docs/purity.md.
func runesTypeAppend(v Value, args []Value, mutate bool) (Value, error) {
	o := (*Runes)(v.Ptr)
	name := "append"
	if mutate {
		name = "append_in_place"
	}
	items, err := runesAppendItems(args, name)
	if err != nil {
		return Undefined, err
	}

	if mutate {
		if v.Immutable {
			return Undefined, errs.NewNotAppendableError(v.TypeName())
		}
		o.Set(append(o.Elements, items...))
		return v, nil
	}

	// Pure: build a fresh, independent slice — never touch o's own backing storage (per docs/conventions.md's
	// variadic/slice argument immutability rule).
	res := make([]rune, 0, len(o.Elements)+len(items))
	res = append(res, o.Elements...)
	res = append(res, items...)
	return NewRunesValue(res, false), nil
}

// PURE by contract. deep is irrelevant here: elements are raw runes, not nested Values, so there's nothing a
// shallow copy could leave shared. Kept for signature parity with the shared Copy hook.
func runesTypeCopy(v Value, _ bool) (Value, error) {
	o := (*Runes)(v.Ptr)
	rs := make([]rune, len(o.Elements))
	copy(rs, o.Elements)
	return NewRunesValue(rs, false), nil
}

func runesTypeEqual(v Value, other Value, final bool) bool {
	o := (*Runes)(v.Ptr)
	switch other.Type {
	case value.Runes:
		t := (*Runes)(other.Ptr).Elements
		return slices.Equal(o.Elements, t)
	case value.String, value.Bool, value.Byte, value.Rune, value.Int, value.Decimal, value.Float:
		t, _ := other.AsString() // identity for String, canonical text form for the rest
		return string(o.Elements) == t
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func runesTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		switch other.Type {
		case value.String:
			switch op {
			case token.Add:
				l := []rune(*(*string)(other.Ptr))
				r := (*Runes)(v.Ptr).Elements
				t := make([]rune, len(l)+len(r))
				copy(t, l)
				copy(t[len(l):], r)
				return NewRunesValue(t, false), nil
			case token.Less:
				l := *(*string)(other.Ptr)
				r := string((*Runes)(v.Ptr).Elements)
				return BoolValue(l < r), nil
			case token.LessEq:
				l := *(*string)(other.Ptr)
				r := string((*Runes)(v.Ptr).Elements)
				return BoolValue(l <= r), nil
			case token.Greater:
				l := *(*string)(other.Ptr)
				r := string((*Runes)(v.Ptr).Elements)
				return BoolValue(l > r), nil
			case token.GreaterEq:
				l := *(*string)(other.Ptr)
				r := string((*Runes)(v.Ptr).Elements)
				return BoolValue(l >= r), nil
			}

		case value.Rune:
			switch op {
			case token.Add:
				l := []rune{rune(other.Data)}
				r := (*Runes)(v.Ptr).Elements
				t := make([]rune, len(l)+len(r))
				copy(t, l)
				copy(t[len(l):], r)
				return NewRunesValue(t, false), nil
			}
		}

		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}

	switch other.Type {
	case value.Runes:
		switch op {
		case token.Add:
			l := (*Runes)(v.Ptr).Elements
			r := (*Runes)(other.Ptr).Elements
			t := make([]rune, len(l)+len(r))
			copy(t, l)
			copy(t[len(l):], r)
			return NewRunesValue(t, false), nil
		case token.Sub:
			l := (*Runes)(v.Ptr).Elements
			r := (*Runes)(other.Ptr).Elements
			if len(r) == 0 {
				return NewRunesValue(slices.Clone(l), false), nil
			}
			s := string(l)
			res := strings.ReplaceAll(s, string(r), "")
			return NewRunesValue([]rune(res), false), nil
		case token.Less:
			l := string((*Runes)(v.Ptr).Elements)
			r := string((*Runes)(other.Ptr).Elements)
			return BoolValue(l < r), nil
		case token.LessEq:
			l := string((*Runes)(v.Ptr).Elements)
			r := string((*Runes)(other.Ptr).Elements)
			return BoolValue(l <= r), nil
		case token.Greater:
			l := string((*Runes)(v.Ptr).Elements)
			r := string((*Runes)(other.Ptr).Elements)
			return BoolValue(l > r), nil
		case token.GreaterEq:
			l := string((*Runes)(v.Ptr).Elements)
			r := string((*Runes)(other.Ptr).Elements)
			return BoolValue(l >= r), nil
		}

	case value.String:
		switch op {
		case token.Add:
			l := (*Runes)(v.Ptr).Elements
			r := []rune(*(*string)(other.Ptr))
			t := make([]rune, len(l)+len(r))
			copy(t, l)
			copy(t[len(l):], r)
			return NewRunesValue(t, false), nil
		case token.Less:
			l := string((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l < r), nil
		case token.LessEq:
			l := string((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l <= r), nil
		case token.Greater:
			l := string((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l > r), nil
		case token.GreaterEq:
			l := string((*Runes)(v.Ptr).Elements)
			r := *(*string)(other.Ptr)
			return BoolValue(l >= r), nil
		}

	case value.Rune:
		switch op {
		case token.Add:
			l := (*Runes)(v.Ptr).Elements
			r := []rune{rune(other.Data)}
			t := make([]rune, len(l)+len(r))
			copy(t, l)
			copy(t[len(l):], r)
			return NewRunesValue(t, false), nil
		case token.Sub:
			l := (*Runes)(v.Ptr).Elements
			r := rune(other.Data)
			t := make([]rune, 0, len(l))
			for _, e := range l {
				if e != r {
					t = append(t, e)
				}
			}
			return NewRunesValue(t, false), nil
		}
	}

	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func runesTypeMethodCall(vm VM, v Value, name string, args []Value) (Value, error) {
	o := (*Runes)(v.Ptr)

	switch name {
	case "copy":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return runesTypeCopy(v, true)

	case "freeze":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v.Freeze()

	case "runes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return v, nil

	case "string":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewStringValue(string(o.Elements)), nil

	case "array":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, _ := runesTypeAsArray(v)
		return NewArrayValue(t, false), nil

	case "bool":
		b, ok := runesTypeAsBool(v)
		return convMember(name, runesTypeName, args, ok, BoolValue(b))

	case "bytes":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		return NewBytesValue([]byte(string(o.Elements)), false), nil

	case "float":
		f, ok := runesTypeAsFloat(v)
		return convMember(name, runesTypeName, args, ok, FloatValue(f))

	case "int":
		i, ok := runesTypeAsInt(v)
		return convMember(name, runesTypeName, args, ok, IntValue(i))

	case "decimal":
		d, ok := runesTypeAsDecimal(v)
		return convMember(name, runesTypeName, args, ok, NewDecimalValue(d))

	case "time":
		t, ok := runesTypeAsTime(v)
		return convMember(name, runesTypeName, args, ok, NewTimeValue(t))

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
		s, err := runesTypeFormat(v, sp)
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
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return RuneValue(o.Elements[0]), nil

	case "last":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return RuneValue(o.Elements[len(o.Elements)-1]), nil

	case "min":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return RuneValue(slices.Min(o.Elements)), nil

	case "max":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(o.Elements) == 0 {
			// absence is data: undefined, or the optional trailing default
			if len(args) == 1 {
				return args[0], nil
			}
			return Undefined, nil
		}
		return RuneValue(slices.Max(o.Elements)), nil

	case "lower":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToLower(r)
		}
		return NewRunesValue(rs, false), nil

	case "upper":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		rs := make([]rune, len(o.Elements))
		for i, r := range o.Elements {
			rs[i] = unicode.ToUpper(r)
		}
		return NewRunesValue(rs, false), nil

	case "contains", "count", "filter", "remove", "any", "all":
		return TripleMatchMember(vm, name, v, args, RuneValue, NewRunesValue, runesTypeResolve,
			runesEncodeMatchArg, func(a, b rune) bool { return a == b }, IsBlankRune)

	case "trim":
		if len(args) > 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0 or 1", len(args))
		}
		if len(args) == 0 {
			return NewRunesValue([]rune(strings.Trim(string(o.Elements), " \t\n")), false), nil
		}
		s, ok := args[0].AsString()
		if !ok {
			return Undefined, errs.NewInvalidArgumentTypeError(name, "first", "string or runes", args[0].TypeName())
		}
		return NewRunesValue([]rune(strings.Trim(string(o.Elements), s)), false), nil

	case "sort":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		sorted := make([]rune, len(o.Elements))
		copy(sorted, o.Elements)
		slices.Sort(sorted)
		return NewRunesValue(sorted, false), nil

	case "sort_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Immutable {
			return Undefined, errs.NewNotSortableError(v.TypeName())
		}
		slices.Sort(o.Elements)
		return v, nil

	case "dedup":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]rune, 0, len(o.Elements))
		for i, r := range o.Elements {
			if i == 0 || r != o.Elements[i-1] {
				out = append(out, r)
			}
		}
		return NewRunesValue(out, false), nil

	case "unique":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		out := make([]rune, 0, len(o.Elements))
		seen := make(map[rune]struct{}, len(o.Elements))
		for _, r := range o.Elements {
			if _, ok := seen[r]; !ok {
				seen[r] = struct{}{}
				out = append(out, r)
			}
		}
		return NewRunesValue(out, false), nil

	case "reverse":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		n := len(o.Elements)
		rev := make([]rune, n)
		for i, r := range o.Elements {
			rev[n-1-i] = r
		}
		return NewRunesValue(rev, false), nil

	case "reverse_in_place":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		if v.Immutable {
			return Undefined, errs.NewNotReversibleError(v.TypeName())
		}
		slices.Reverse(o.Elements)
		return v, nil

	case "for_each":
		return SeqForEach(vm, v, args, RuneValue, runesTypeResolve)

	case "index", "index_last":
		// offsets are rune positions on this receiver
		return SeqIndex(vm, v, args, name == "index_last", RuneValue, runesTypeResolve,
			func(a Value) bool { return a.Type == value.String || a.Type == value.Runes || a.Type == value.Bytes },
			func(elems []rune, run Value, last bool) (int64, bool, error) {
				rs, ok := run.AsRunes()
				if !ok {
					return -1, false, errs.NewInvalidArgumentTypeError(name, "first", "text content", run.TypeName())
				}
				idx, found := SeqIndexRun(elems, rs, RuneValue, last)
				return idx, found, nil
			},
			IsBlankRune)

	case "chunk":
		return SeqChunk(v, args, NewRunesValue, runesTypeResolve)

	case "chunk_view":
		return SeqChunkView(v, args, NewRunesValue, runesTypeResolve)

	case "slice":
		return SeqSlice(v, args)

	case "slice_view":
		return SeqSliceView(v, args, NewRunesValue, runesTypeResolve)

	case "append":
		return runesTypeAppend(v, args, false)

	case "append_in_place":
		return runesTypeAppend(v, args, true)

	case "splice_in_place":
		return SeqSplice(append([]Value{v}, args...), true, NewRunesValue, runesTypeResolve, runesAppendItems, runesTypeName)

	case "splice":
		return SeqSplice(append([]Value{v}, args...), false, NewRunesValue, runesTypeResolve, runesAppendItems, runesTypeName)

	case "map":
		return SeqMap(vm, v, args, RuneValue, runesTypeResolve)

	case "reduce":
		return SeqReduce(vm, v, args, RuneValue, runesTypeResolve)

	case "repeat":
		n, err := parseRepeatCount(name, args)
		if err != nil {
			return Undefined, err
		}
		src := o.Elements
		sl := len(src)
		out := make([]rune, n*sl)
		for i := range n {
			copy(out[i*sl:], src)
		}
		return NewRunesValue(out, false), nil

	case "join":
		if len(args) != 1 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
		}
		elems, err := resolveJoinSeq(args[0], name)
		if err != nil {
			return Undefined, err
		}
		s, err := joinElementsToString(elems, string(o.Elements))
		if err != nil {
			return Undefined, err
		}
		return NewRunesValue([]rune(s), false), nil

	case "split":
		return runesFnSplit(v, args)

	case "split_lines":
		return runesFnSplitLines(v, args)

	case "partition":
		return runesFnPartition(v, args)

	default:
		return Undefined, errs.NewInvalidMethodError(name, v.TypeName())
	}
}

// PURE: constructs a fresh iterator. Iterator advancement is a separate hook. See docs/purity.md.
func runesTypeIterator(v Value) (Value, error) {
	return NewRunesIteratorValue((*Runes)(v.Ptr).Elements), nil
}

// PURE by contract
func runesTypeAsInt(v Value) (int64, bool) {
	o := (*Runes)(v.Ptr)
	i, err := strconv.ParseInt(string(o.Elements), 10, 64)
	if err == nil {
		return i, true
	}
	return 0, false
}

// PURE by contract
func runesTypeAsFloat(v Value) (float64, bool) {
	o := (*Runes)(v.Ptr)
	f, err := strconv.ParseFloat(string(o.Elements), 64)
	if err == nil {
		return f, true
	}
	return 0, false
}

// PURE by contract
func runesTypeAsDecimal(v Value) (dec128.Dec128, bool) {
	o := (*Runes)(v.Ptr)
	d := dec128.FromString(string(o.Elements))
	return d, !d.IsNaN()
}

// PURE by contract
func runesTypeAsBool(v Value) (bool, bool) {
	o := (*Runes)(v.Ptr)
	return conv.ParseBool(string(o.Elements))
}

// PURE by contract
func runesTypeAsBytes(v Value) ([]byte, bool) {
	o := (*Runes)(v.Ptr)
	return []byte(string(o.Elements)), true
}

// PURE by contract
func runesTypeAsTime(v Value) (time.Time, bool) {
	return parseTimeText(string((*Runes)(v.Ptr).Elements))
}

// PURE by contract
func runesTypeAsArray(v Value) ([]Value, bool) {
	o := (*Runes)(v.Ptr)
	arr := make([]Value, len(o.Elements))
	for i, r := range o.Elements {
		arr[i] = RuneValue(r)
	}
	return arr, true
}

// PURE by contract
func runesTypeContains(v Value, e Value) bool {
	o := (*Runes)(v.Ptr)
	switch e.Type {
	case value.Rune:
		c := rune(e.Data)
		return slices.Contains(o.Elements, c)

	case value.String:
		return strings.Contains(string(o.Elements), *(*string)(e.Ptr))

	case value.Runes:
		return strings.Contains(string(o.Elements), string((*Runes)(e.Ptr).Elements))

	default:
		c, ok := e.AsRune()
		if !ok {
			return false
		}
		return slices.Contains(o.Elements, c)
	}
}

// PURE by contract
// PURE by contract
// PURE by contract
func runesFnSplit(v Value, args []Value) (Value, error) {
	const name = "split"
	if len(args) > 2 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0, 1 or 2", len(args))
	}
	o := (*Runes)(v.Ptr)
	src := string(o.Elements)
	var pieces []string
	if len(args) == 0 {
		pieces = splitStringWhitespace(src)
	} else {
		sep, err := coerceSepToString(name, args[0])
		if err != nil {
			return Undefined, err
		}
		if sep == "" {
			return Undefined, errs.NewInvalidValueError("(split) separator must not be empty")
		}
		limit := -1
		if len(args) == 2 {
			limit, err = parseSplitLimit(name, args, 1)
			if err != nil {
				return Undefined, err
			}
		}
		pieces = splitStringByLiteral(src, sep, limit)
	}
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue([]rune(p), false)
	}
	return NewArrayValue(arr, false), nil
}

// PURE by contract
func runesFnSplitLines(v Value, args []Value) (Value, error) {
	const name = "split_lines"
	if len(args) != 0 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
	}
	o := (*Runes)(v.Ptr)
	pieces := splitLinesString(string(o.Elements))
	arr := make([]Value, len(pieces))
	for i, p := range pieces {
		arr[i] = NewRunesValue([]rune(p), false)
	}
	return NewArrayValue(arr, false), nil
}

// PURE by contract
func runesFnPartition(v Value, args []Value) (Value, error) {
	const name = "partition"
	if len(args) != 1 {
		return Undefined, errs.NewWrongNumArgumentsError(name, "1", len(args))
	}
	sep, err := coerceSepToString(name, args[0])
	if err != nil {
		return Undefined, err
	}
	if sep == "" {
		return Undefined, errs.NewInvalidValueError("(partition) separator must not be empty")
	}
	o := (*Runes)(v.Ptr)
	src := string(o.Elements)
	arr := make([]Value, 3)
	idx := strings.Index(src, sep)
	if idx < 0 {
		arr[0] = NewRunesValue([]rune(src), false)
		arr[1] = NewRunesValue(nil, false)
		arr[2] = NewRunesValue(nil, false)
	} else {
		arr[0] = NewRunesValue([]rune(src[:idx]), false)
		arr[1] = NewRunesValue([]rune(src[idx:idx+len(sep)]), false)
		arr[2] = NewRunesValue([]rune(src[idx+len(sep):]), false)
	}
	return NewArrayValue(arr, false), nil
}
