package core

import (
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/core/member/members"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/internal/binary"
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
	EncodeBinary: boundMemberTypeEncodeBinary,                            // PURE by contract
	DecodeBinary: boundMemberTypeDecodeBinary,                            // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](true, nil),                     // PURE by contract
	Equal:        boundMemberTypeEqual,                                   // PURE by contract
	IsCallable:   ConstHook(true),                                        // PURE by contract
	IsVariadic:   boundMemberTypeIsVariadic,                              // PURE by contract
	Arity:        boundMemberTypeArity,                                   // PURE by contract
	Call:         boundMemberTypeCall,                                    // CALLABLE-DEPENDENT by contract

	Methods: []Method{
		members.IsTrue: NewPureMethod(MemberIsTrue, 0, Fixed),
		members.Format: NewPureMethod(MemberFormat, 0, Variadic),
		members.Copy:   NewPureMethod(MemberSelf, 0, Fixed),
		members.Freeze: NewPureMethod(MemberSelf, 0, Fixed),
	},
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
	if e.Variadic {
		return fmt.Sprintf("<bound-member:%s.%s/%d+>", recv, o.Name, e.Arity)
	}
	return fmt.Sprintf("<bound-member:%s.%s/%d>", recv, o.Name, e.Arity)
}

func boundMemberTypeIsVariadic(v Value) bool {
	return (*BoundMember)(v.Ptr).entry().Variadic
}

func boundMemberTypeArity(v Value) int {
	return (*BoundMember)(v.Ptr).entry().Arity
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

// The receiver's own encoding, the member id, the name. A receiver that does not encode makes the bound member not
// encode, with the receiver's error.
func boundMemberTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*BoundMember)(v.Ptr)
	rb, err := o.Receiver.EncodeBinary()
	if err != nil {
		return nil, err
	}
	b := binary.AppendBytes(nil, rb)
	b = binary.AppendUint64(b, uint64(o.ID))
	b = binary.AppendBytes(b, []byte(o.Name))
	return b, nil
}

func boundMemberTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	rb, err := binary.ReadBytes(data, &offset, "bound member (receiver)")
	if err != nil {
		return err
	}
	var recv Value
	if err := recv.DecodeBinary(rb); err != nil {
		return err
	}
	id, err := binary.ReadUint64(data, &offset, "bound member (id)")
	if err != nil {
		return err
	}
	name, err := binary.ReadBytes(data, &offset, "bound member (name)")
	if err != nil {
		return err
	}
	if offset != len(data) {
		return fmt.Errorf("bound member: trailing %d bytes", len(data)-offset)
	}
	if id >= uint64(member.Max) {
		return fmt.Errorf("bound member: member id %d out of range", id)
	}
	o := &BoundMember{Receiver: recv, ID: member.ID(id), Name: string(name)}
	if o.ID != member.Unknown {
		if o.ID.String() != o.Name {
			return fmt.Errorf("bound member: member id %d is not %q", id, o.Name)
		}
		if o.entry().Fn == nil {
			return fmt.Errorf("bound member: type %s has no member %q", recv.TypeName(), o.Name)
		}
	}
	*v = Value{Type: value.BoundMember, Immutable: true, Ptr: unsafe.Pointer(o)}
	return nil
}
