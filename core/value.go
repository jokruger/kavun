package core

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/jokruger/dec128"
	bc "github.com/jokruger/kavun/core/bytecode"
	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
)

const anyTypeName = "value"

// Value represents a boxed Kavun value.
type Value struct {
	Type      uint8
	Immutable bool
	Data      uint64
	Ptr       unsafe.Pointer
}

// RefValue is a dummy constructor used in internal generics.
func RefValue(v Value) Value {
	return v
}

func (v *Value) Set(val Value) {
	*v = val
}

// PURE by contract
// EncodeJSON is the SINGLE translation point for the EncodeJSON hooks: a hook may answer a raw Go error, and this
// turns it into a recoverable json_encoding error exactly once. Script-reachable via json.encode, so a bare hook
// error must never travel further — errs.IsCritical reads a non-*errs.Error as fatal and would stop the VM.
//
// A json_encoding error coming back up from a nested element is already translated and already carries its path,
// so it passes through untouched; prefixing at every level used to produce
// "json encoding failed for type array: json encoding failed for type array: …".
func (v Value) EncodeJSON() ([]byte, error) {
	b, err := ValueTypes[v.Type].EncodeJSON(v)
	if err == nil {
		return b, nil
	}
	if e := errs.AsError(err); e != nil && e.Kind == errs.KindJSONEncoding {
		return nil, err
	}
	return nil, errs.NewJSONEncodingError(fmt.Sprintf("(%s) %s", v.TypeName(), err.Error()))
}

// binaryCodecError is the SINGLE translation point for the EncodeBinary/DecodeBinary hooks. Those hooks may answer
// raw Go errors internally — the nesting inside a container hook is theirs to shape — but exactly one binary_encoding
// error comes back out, so nothing raw can reach errs.IsCritical and be read as fatal. An error already translated
// by a nested level passes through untouched.
func binaryCodecError(typeName string, err error) error {
	if e := errs.AsError(err); e != nil && e.Kind == errs.KindBinaryEncoding {
		return err
	}
	return errs.NewBinaryEncodingError(fmt.Sprintf("(%s) %s", typeName, err.Error()))
}

// jsonPathPrefix prepends one path segment to a nested json_encoding failure, so the error names WHERE in the
// structure it happened: "[0].price: (function) value type function does not support JSON encoding". A segment is
// "[i]" for an array index and ".key" for a dict/record key; consecutive segments join directly, and the first one
// is separated from the reason by ": ".
func jsonPathPrefix(seg string, err error) error {
	e := errs.AsError(err)
	if e == nil || e.Kind != errs.KindJSONEncoding {
		return err
	}
	if strings.HasPrefix(e.Message, "[") || strings.HasPrefix(e.Message, ".") {
		return errs.NewJSONEncodingError(seg + e.Message)
	}
	return errs.NewJSONEncodingError(seg + ": " + e.Message)
}

// PURE by contract
func (v Value) EncodeBinary() ([]byte, error) {
	i := byte(0)
	if v.Immutable {
		i = byte(1)
	}
	b, err := ValueTypes[v.Type].EncodeBinary(v)
	if err != nil {
		return nil, binaryCodecError(v.TypeName(), err)
	}
	return append([]byte{v.Type, i}, b...), nil
}

// IMPURE by contract (mutates target)
func (v *Value) DecodeBinary(data []byte) error {
	if len(data) < 2 {
		return errs.NewBinaryEncodingError(fmt.Sprintf("(decode) type header: expected at least 2 bytes, got %d", len(data)))
	}
	var t Value
	t.Type = data[0]
	t.Immutable = data[1] != 0
	if err := ValueTypes[t.Type].DecodeBinary(&t, data[2:]); err != nil {
		return binaryCodecError(t.TypeName(), err)
	}
	*v = t
	return nil
}

// GobEncode wraps binary encoding so gob does not reflect over unsafe.Pointer field.
func (v Value) GobEncode() ([]byte, error) {
	return v.EncodeBinary()
}

// GobDecode wraps binary decoding to mirror GobEncode.
func (v *Value) GobDecode(data []byte) error {
	return v.DecodeBinary(data)
}

// LOCALISED-STATE by contract (advances iterator cursor)
func (v Value) Next() bool {
	return ValueTypes[v.Type].Next(v)
}

// LOCALISED-STATE by contract (reads iterator cursor)
func (v Value) Key() (Value, error) {
	return ValueTypes[v.Type].Key(v)
}

// LOCALISED-STATE by contract (reads iterator cursor)
// Elem is the single-variable for-in binding: the container's ELEMENT — the Value hook everywhere except map
// iterators, whose element is the KEY (the value is the attachment; the two-variable form reads both).
func (v Value) Elem() (Value, error) {
	return ValueTypes[v.Type].Elem(v)
}

func (v Value) Value() (Value, error) {
	return ValueTypes[v.Type].Value(v)
}

// PURE by contract
func (v Value) TypeName() string {
	return ValueTypes[v.Type].Name(v)
}

// PURE by contract
func (v Value) Format(sp fspec.FormatSpec) (string, error) {
	return ValueTypes[v.Type].Format(v, sp)
}

// PURE by contract
func (v Value) String() string {
	return ValueTypes[v.Type].String(v)
}

// PURE by contract
func (v Value) Interface() any {
	return ValueTypes[v.Type].Interface(v)
}

// PURE by contract
func (v Value) Arity() int {
	return ValueTypes[v.Type].Arity(v)
}

// PURE by contract
func (v Value) IsPrimitive() bool {
	return v.Type <= value.LastPrimitiveType
}

// PURE by contract
func (v Value) IsUserDefined() bool {
	return v.Type >= value.FirstUserDefinedType
}

// PURE by contract
func (v Value) IsTrue() (bool, error) {
	return ValueTypes[v.Type].IsTrue(v)
}

// PURE by contract
func (v Value) IsIterable() bool {
	return ValueTypes[v.Type].IsIterable(v)
}

// PURE by contract
func (v Value) IsCallable() bool {
	return ValueTypes[v.Type].IsCallable(v)
}

// PURE by contract
func (v Value) IsVariadic() bool {
	return ValueTypes[v.Type].IsVariadic(v)
}

// PURE by contract
// Contains is the `in` operator: exactly the contains member's VALUE readings — element | run | family,
// full member acceptance — raising on an unacceptable operand. A callable operand raises too: an operator
// operand is always a value; the predicate reading is the member's (contains(f) ≡ any(f)).
func (v Value) Contains(e Value) (bool, error) {
	return ValueTypes[v.Type].Contains(v, e)
}

// PURE by contract
func (v Value) AsValue() (Value, bool) {
	return v, true
}

// PURE by contract
func (v Value) AsBool() (bool, bool) {
	return ValueTypes[v.Type].AsBool(v)
}

// PURE by contract
func (v Value) AsRune() (rune, bool) {
	return ValueTypes[v.Type].AsRune(v)
}

// PURE by contract
func (v Value) AsByte() (byte, bool) {
	return ValueTypes[v.Type].AsByte(v)
}

// PURE by contract
func (v Value) AsInt() (int64, bool) {
	return ValueTypes[v.Type].AsInt(v)
}

// PURE by contract
func (v Value) AsFloat() (float64, bool) {
	return ValueTypes[v.Type].AsFloat(v)
}

// PURE by contract
func (v Value) AsDecimal() (dec128.Dec128, bool) {
	return ValueTypes[v.Type].AsDecimal(v)
}

// PURE by contract
func (v Value) AsTime() (time.Time, bool) {
	return ValueTypes[v.Type].AsTime(v)
}

// PURE by contract
func (v Value) AsString() (string, bool) {
	return ValueTypes[v.Type].AsString(v)
}

// PURE by contract
func (v Value) AsRunes() ([]rune, bool) {
	return ValueTypes[v.Type].AsRunes(v)
}

// PURE by contract
func (v Value) AsBytes() ([]byte, bool) {
	return ValueTypes[v.Type].AsBytes(v)
}

// PURE by contract
func (v Value) AsArray() ([]Value, bool) {
	return ValueTypes[v.Type].AsArray(v)
}

// PURE by contract
func (v Value) AsIntRange() (IntRange, bool) {
	return ValueTypes[v.Type].AsIntRange(v)
}

// PURE by contract
func (v Value) AsDict() (map[string]Value, bool) {
	return ValueTypes[v.Type].AsDict(v)
}

// PURE by contract.
func (v Value) Equal(rhs Value) bool {
	return ValueTypes[v.Type].Equal(v, rhs, false)
}

// PURE by contract.
func (v Value) BinaryOp(op token.Token, rhs Value) (Value, error) {
	return ValueTypes[v.Type].BinaryOp(v, rhs, op, false)
}

// PURE by contract
func (v Value) UnaryOp(op token.Token) (Value, error) {
	return ValueTypes[v.Type].UnaryOp(v, op)
}

// PURE by contract: deep=true recursively copies nested Values (copy()); deep=false copies only the top-level
// container/wrapper, sharing nested structure (copy_shallow()).
func (v *Value) Copy(deep bool) (Value, error) {
	return ValueTypes[v.Type].Copy(*v, deep)
}

// METHOD-DEPENDENT by contract: purity varies per method name, reported by IsMethodPure (see docs/purity.md)
func (v Value) MethodCall(vm VM, name string, args []Value) (Value, error) {
	// universal members are answered here, once for every type — builtin and
	// host-defined alike — instead of being repeated in each MethodCall switch
	switch name {
	case "is_true":
		if len(args) != 0 {
			return Undefined, errs.NewWrongNumArgumentsError(name, "0", len(args))
		}
		t, err := ValueTypes[v.Type].IsTrue(v)
		if err != nil {
			return Undefined, err
		}
		return BoolValue(t), nil
	}
	return ValueTypes[v.Type].MethodCall(vm, v, name, args)
}

// PURE by contract
func (v Value) Access(index Value, mode bc.Opcode) (Value, error) {
	return ValueTypes[v.Type].Access(v, index, mode)
}

// IMPURE by contract (mutates target)
func (v Value) Assign(idx Value, val Value, mode bc.Opcode) error {
	return ValueTypes[v.Type].Assign(v, idx, val, mode)
}

// PURE by contract (constructs new iterator)
func (v Value) Iterator() (Value, error) {
	return ValueTypes[v.Type].Iterator(v)
}

// CALLABLE-DEPENDENT by contract
func (v Value) Call(vm VM, args []Value) (Value, error) {
	return ValueTypes[v.Type].Call(vm, v, args)
}

// PURE by contract
func (v Value) Len() int64 {
	return ValueTypes[v.Type].Len(v)
}

// MUTATE-DEPENDENT by contract: mutate=true mutates the receiver in place (append_in_place()); mutate=false
// returns an independent value with the items appended (append()).
func (v Value) Append(args []Value, mutate bool) (Value, error) {
	return ValueTypes[v.Type].Append(v, args, mutate)
}

// MUTATE-DEPENDENT by contract: mutate=true mutates the receiver in place (delete_in_place()); mutate=false
// returns an independent container without the key (delete()).
func (v Value) Delete(key Value, mutate bool) (Value, error) {
	return ValueTypes[v.Type].Delete(v, key, mutate)
}

// PURE by contract
func (v Value) Slice(s Value, e Value) (Value, error) {
	return ValueTypes[v.Type].Slice(v, s, e)
}

// PURE by contract
func (v Value) SliceStep(s Value, e Value, step Value) (Value, error) {
	return ValueTypes[v.Type].SliceStep(v, s, e, step)
}

// PURE by contract: exposed to scripts as freeze_shallow() (member call and free builtin). Despite the naming
// symmetry with freeze()'s "_shallow"/deep split, this never mutates any shared storage — it returns a new
// header (Immutable flag flipped) pointing at the same body, so it's a genuinely pure operation like copy() or
// copy_shallow(), not an "_in_place"-style body mutation: the caller must capture and reassign the result
// (`x = x.freeze_shallow()` / `x = freeze_shallow(x)`) to see any effect on their own variable, and a
// pre-existing sibling binding into the same body is unaffected and stays independently mutable. Renamed from
// freeze_in_place 2026-08-17 — that name wrongly implied the same "mutates without reassignment" behavior as
// append_in_place/splice_in_place/delete_in_place, which this operation structurally cannot do (Immutable lives
// on the header, not the body).
func (v Value) ToImmutable() (Value, error) {
	t := v
	t.Immutable = true
	return t, nil
}

// PURE by contract: never mutates the receiver or affects any existing alias into it. Deep-copies first
// (Copy(true)), then marks only the fresh, not-yet-observable clone immutable throughout (MarkImmutableDeep) —
// safe for the same reason export's codegen is (see MarkImmutableDeep's own precondition): nothing outside this
// call can reach the clone yet. This is freeze()'s definition; freeze_shallow() is ToImmutable() by another
// name — the explicit twin that skips the detach and so does NOT protect against another live, still-mutable
// alias into the same body.
func (v Value) Freeze() (Value, error) {
	c, err := v.Copy(true)
	if err != nil {
		return Undefined, err
	}
	c.MarkImmutableDeep()
	return c, nil
}

// IMPURE by contract (mutates target)
//
// MarkImmutableDeep flips Immutable to true on v and, recursively, on every Value reachable through it —
// array/dict/record elements and an error's payload — without cloning anything. Unlike ToImmutable, which only
// ever flips the top-level flag, this walks into containers so that no reachable nested Value keeps looking
// mutable after the top-level one no longer is. Only safe to call when nothing outside the caller can still
// observe v (or anything under it) as mutable. Deliberately does not recurse into a compiled function's closed-over
// free variables (CompiledFunction.Free) or ValuePtr indirection: those are variable-capture aliasing, a different
// mechanism from container nesting, and out of scope here. Does not guard against cyclic containers, matching every
// other recursive Value walk in this package (e.g. arrayTypeCopy) — Kavun's shared-by-default container model doesn't
// defend against that anywhere today.
func (v *Value) MarkImmutableDeep() {
	v.Immutable = true
	switch v.Type {
	case value.Array:
		o := (*Array)(v.Ptr)
		for i := range o.Elements {
			o.Elements[i].MarkImmutableDeep()
		}
	case value.Dict:
		o := (*Dict)(v.Ptr)
		for k, e := range o.Elements {
			e.MarkImmutableDeep()
			o.Elements[k] = e
		}
	case value.Record:
		o := (*Record)(v.Ptr)
		for k, e := range o.Elements {
			e.MarkImmutableDeep()
			o.Elements[k] = e
		}
	case value.Error:
		o := (*Error)(v.Ptr)
		o.Payload.MarkImmutableDeep()
	}
}
