package core

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/jokruger/kavun/core/member/members"

	"github.com/jokruger/kavun/core/token"
	"github.com/jokruger/kavun/core/value"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/fspec"
	"github.com/jokruger/kavun/internal/binary"
)

const errorTypeName = "error"
const KindUser = "user"

type Error struct {
	Payload  Value
	Kind     string
	Category errs.Category
	Fatal    bool
}

func (e *Error) Set(payload Value, kind string, category errs.Category, fatal bool) {
	e.Payload = payload
	e.Kind = kind
	e.Category = category
	e.Fatal = fatal
}

func NewErrorValue(payload Value, kind string, category errs.Category, fatal bool) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: payload, Kind: kind, Category: category, Fatal: fatal}),
	}
}

func NewRuntimeErrorValue(kind string, category errs.Category, fatal bool, message string) Value {
	return Value{
		Type:      value.Error,
		Immutable: true,
		Ptr:       unsafe.Pointer(&Error{Payload: NewStringValue(message), Kind: kind, Category: category, Fatal: fatal}),
	}
}

var TypeError = ValueTypeDescr{
	Name:         ConstHook(errorTypeName),                            // PURE by contract
	String:       errorTypeString,                                     // PURE by contract
	Format:       errorTypeFormat,                                     // PURE by contract
	Interface:    func(v Value) any { return errors.New(v.String()) }, // PURE by contract
	EncodeJSON:   errorTypeEncodeJSON,                                 // PURE by contract
	EncodeBinary: errorTypeEncodeBinary,                               // PURE by contract
	DecodeBinary: errorTypeDecodeBinary,                               // IMPURE by contract (mutates target)
	IsTrue:       Const2Hook[bool, error](true, nil),                  // PURE by contract
	Copy:         errorTypeCopy,                                       // PURE by contract
	Equal:        errorTypeEqual,                                      // PURE by contract
	UnaryOp:      errorTypeUnaryOp,                                    // PURE by contract
	BinaryOp:     errorTypeBinaryOp,                                   // PURE by contract
	AsString:     errorTypeAsString,                                   // PURE by contract
	AsBool:       Const2Hook(true, true),                              // PURE by contract

	Methods: []MethodEntry{
		members.IsTrue:        {Fn: memberIsTrue, Pure: true},
		members.String:        {Fn: errorString, Pure: true},
		members.Format:        {Fn: memberFormat, Pure: true},
		members.Copy:          {Fn: errorCopy, Pure: true},
		members.Freeze:        {Fn: errorFreeze, Pure: true},
		members.Runes:         {Fn: errorRunes, Pure: true},
		members.Bool:          {Fn: errorBool, Pure: true},
		members.Kind:          {Fn: errorKind, Pure: true},
		members.Value:         {Fn: errorValue, Pure: true},
		members.IsUser:        {Fn: errorIsUser, Pure: true},
		members.IsRuntime:     {Fn: errorIsRuntime, Pure: true},
		members.IsRequirement: {Fn: errorIsRequirement, Pure: true},
	},
}

func errorTypeEncodeJSON(v Value) ([]byte, error) {
	// the payload's render (total: a non-string payload renders as its string form), encoded exactly as a string
	// is — JSON escaping, and the same refusal of text holding octets that are not symbols
	s, _ := errorTypeAsString(v)
	b, err := stringTypeEncodeJSON(NewStringValue(s))
	if err != nil {
		return nil, err
	}
	return append(append([]byte(`{"error":`), b...), '}'), nil
}

func errorTypeEncodeBinary(v Value) ([]byte, error) {
	o := (*Error)(v.Ptr)
	pb, err := o.Payload.EncodeBinary()
	if err != nil {
		return nil, fmt.Errorf("error (payload): %w", err)
	}

	b := binary.AppendBytes(nil, []byte(o.Kind))
	b = append(b, byte(o.Category))
	if o.Fatal {
		b = append(b, byte(1))
	} else {
		b = append(b, byte(0))
	}
	b = binary.AppendBytes(b, pb)
	return b, nil
}

func errorTypeDecodeBinary(v *Value, data []byte) error {
	offset := 0
	kb, err := binary.ReadBytes(data, &offset, "error (kind)")
	if err != nil {
		return err
	}
	if len(data)-offset < 2 {
		return fmt.Errorf("error (category, fatal): expected 2 bytes, got %d", len(data)-offset)
	}
	category := errs.Category(data[offset])
	offset++
	fatal := data[offset] != 0
	offset++

	pb, err := binary.ReadBytes(data, &offset, "error (payload)")
	if err != nil {
		return err
	}
	var payload Value
	if err := payload.DecodeBinary(pb); err != nil {
		return fmt.Errorf("error (payload): %w", err)
	}
	if offset != len(data) {
		return fmt.Errorf("error: trailing %d bytes", len(data)-offset)
	}

	*v = NewErrorValue(payload, string(kb), category, fatal)
	return nil
}

func errorTypeString(v Value) string {
	o := (*Error)(v.Ptr)
	if o.Payload.Type == value.Undefined {
		return "error()"
	}
	return fmt.Sprintf("error(%s)", o.Payload.String())
}

func errorTypeFormat(v Value, sp fspec.FormatSpec) (string, error) {
	if sp.HasUnconsumedTail() {
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
	switch sp.Verb {
	case 0:
		s, _ := errorTypeAsString(v) // total: a non-string payload renders as its string form
		return fspec.ApplyGenerics(s, sp, fspec.AlignLeft), nil

	case 'v':
		return errorTypeString(v), nil

	case 'T':
		return fspec.ApplyGenerics(errorTypeName, sp, fspec.AlignLeft), nil

	default:
		return "", errs.NewUnsupportedFormatSpec(v.TypeName(), sp)
	}
}

// deep=true recursively copies the payload (today's copy() semantics); deep=false only allocates a new Error
// wrapper, leaving the payload sharing the source (copy_shallow()).
func errorTypeCopy(v Value, deep bool) (Value, error) {
	o := (*Error)(v.Ptr)
	if !deep {
		return NewErrorValue(o.Payload, o.Kind, o.Category, o.Fatal), nil
	}
	pl, err := o.Payload.Copy(true)
	if err != nil {
		return Undefined, err
	}
	return NewErrorValue(pl, o.Kind, o.Category, o.Fatal), nil
}

func errorTypeEqual(v Value, other Value, final bool) bool {
	switch other.Type {
	case value.Error:
		o := (*Error)(v.Ptr)
		x := (*Error)(other.Ptr)
		return o.Payload.Equal(x.Payload)
	}

	// default to false if final
	if final {
		return false
	}

	// delegate
	return ValueTypes[other.Type].Equal(other, v, true)
}

// PURE by contract.
func errorTypeBinaryOp(v Value, other Value, op token.Token, reflected bool) (Value, error) {
	if reflected {
		return Undefined, errs.NewInvalidBinaryOperatorError(op.String(), other.TypeName(), v.TypeName())
	}
	return ValueTypes[other.Type].BinaryOp(other, v, op, true)
}

// PURE by contract.
// error has no unary operations.
func errorTypeUnaryOp(v Value, op token.Token) (Value, error) {
	return Undefined, errs.NewInvalidUnaryOperatorError(op.String(), v.TypeName())
}

func errorTypeAsString(v Value) (string, bool) {
	o := (*Error)(v.Ptr)
	if s, ok := o.Payload.AsString(); ok {
		return s, true
	}
	return o.Payload.String(), true
}
