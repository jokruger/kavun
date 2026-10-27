package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
)

// BoundMember is a member function read as a value: x.name without a call. The receiver is captured by value at
// bind time — a heap receiver shares its body (a later _in_place on the original is visible through the bound
// member), while the header, Immutable flag included, is the one copied then.
type BoundMember struct {
	Receiver Value
	ID       member.ID // Unknown for a name-path member (embedder types)
	Name     string    // always set; the name is truth
}

func NewBoundMemberValue(recv Value, id member.ID, name string) Value {
	o := &BoundMember{Receiver: recv, ID: id, Name: name}
	return Value{Type: value.BoundMember, Immutable: true, Ptr: unsafe.Pointer(o)}
}

var TypeBoundMember = ValueTypeDescr{
	Name:         boundMemberTypeName,                                    // PURE by contract
	String:       func(v Value) string { return boundMemberTypeName(v) }, // PURE by contract
	Format:       callableFormat,                                         // PURE by contract
	EncodeBinary: codeEncodeBinary,                                       // PURE by contract (refuses: code, not data)
	DecodeBinary: codeDecodeBinary,                                       // IMPURE by contract (refuses: code, not data)
	IsTrue:       Const2Hook[bool, error](true, nil),                     // PURE by contract
	Equal:        boundMemberTypeEqual,                                   // PURE by contract
	IsCallable:   ConstHook(true),                                        // PURE by contract
	IsVariadic:   boundMemberTypeIsVariadic,                              // PURE by contract
	Arity:        boundMemberTypeArity,                                   // PURE by contract
	Call:         boundMemberTypeCall,                                    // CALLABLE-DEPENDENT by contract

	Methods: []Method{
		members.IsTrue: NewPureMethod("()", boundMemberIsTrue),
		members.Format: NewPureMethod("([spec])", boundMemberFormat),
		members.Copy:   NewPureMethod("()", boundMemberCopy),
		members.Freeze: NewPureMethod("()", boundMemberFreeze),
	},
}

// boundMemberIsTrue is a bound member's is_true(): the type's IsTrue hook, as a bool.
// PURE by contract
func boundMemberIsTrue(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	t, err := ValueTypes[v.Type].IsTrue(v)
	if err != nil {
		return Undefined, err
	}
	return BoolValue(t), nil
}

// boundMemberFormat is a bound member's format([spec]).
// PURE by contract
func boundMemberFormat(_ VM, v Value, id member.ID, args []Value) (Value, error) {
	return FormatWithSpec(v, id, args)
}

// boundMemberCopy is a bound member's copy(): the receiver — a bound member is immutable, so a copy of any depth is the
// value itself.
// PURE by contract
func boundMemberCopy(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
}

// boundMemberFreeze is a bound member's freeze(): the receiver — a bound member is immutable already.
// PURE by contract
func boundMemberFreeze(_ VM, v Value, _ member.ID, _ []Value) (Value, error) {
	return v, nil
}

// entry is the receiver type's table entry for the bound member, or a zero entry for a name-path member.
func (o *BoundMember) entry() Method {
	d := &ValueTypes[o.Receiver.Type]
	if o.ID != member.Unknown && int(o.ID) < len(d.Methods) {
		return d.Methods[o.ID]
	}
	return Method{}
}

func boundMemberTypeName(v Value) string {
	o := (*BoundMember)(v.Ptr)
	if o == nil { // a value whose decoding failed is named in the decode error
		return "<bound-member>"
	}
	recv := o.Receiver.TypeName()
	if o.Receiver.IsCallable() {
		recv = "function"
	}
	e := o.entry()
	if e.Sig.IsVariadic() {
		return fmt.Sprintf("<bound-member:%s.%s/%d+>", recv, o.Name, e.Sig.Arity())
	}
	return fmt.Sprintf("<bound-member:%s.%s/%d>", recv, o.Name, e.Sig.Arity())
}

func boundMemberTypeIsVariadic(v Value) bool {
	return (*BoundMember)(v.Ptr).entry().Sig.IsVariadic()
}

func boundMemberTypeArity(v Value) int {
	return (*BoundMember)(v.Ptr).entry().Sig.Arity()
}

// CALLABLE-DEPENDENT: exactly the dispatch of x.name(args), so the call is as pure as the member. See docs/purity.md.
func boundMemberTypeCall(vm VM, v Value, args []Value) (Value, error) {
	o := (*BoundMember)(v.Ptr)
	return o.Receiver.CallMember(vm, o.ID, o.Name, args)
}

// Two bound members are equal when they bind the same member to equal receivers.
func boundMemberTypeEqual(v Value, other Value, final bool) bool {
	if other.Type == value.BoundMember {
		l, r := (*BoundMember)(v.Ptr), (*BoundMember)(other.Ptr)
		return l.ID == r.ID && l.Name == r.Name && l.Receiver.Equal(r.Receiver)
	}
	if final {
		return false
	}
	return ValueTypes[other.Type].Equal(other, v, true)
}
