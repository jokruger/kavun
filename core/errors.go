package core

import (
	"errors"
	"fmt"
)

var (
	ErrLogicError            = errors.New("logic error")
	ErrStackOverflow         = errors.New("stack overflow")
	ErrObjectAllocLimit      = errors.New("object allocation limit exceeded")
	ErrBytesLimit            = errors.New("bytes size limit exceeded")
	ErrStringLimit           = errors.New("string size limit exceeded")
	ErrDecodeBinarySize      = errors.New("invalid binary size")
	ErrBinaryNotSupported    = errors.New("binary serialization not supported")
	ErrInvalidArgumentType   = errors.New("invalid argument type")
	ErrIndexOutOfBounds      = errors.New("index out of bounds")
	ErrWrongNumArguments     = errors.New("wrong number of arguments")
	ErrInvalidAccessMode     = errors.New("invalid access mode")
	ErrNotAccessible         = errors.New("object is not accessible")
	ErrNotAssignable         = errors.New("object is not assignable")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidSelector       = errors.New("invalid selector")
	ErrNotImplemented        = errors.New("not implemented")
	ErrInvalidBinaryOperator = errors.New("invalid binary operator")
)

func NewLogicError(context string) error {
	return fmt.Errorf("%w: %s", ErrLogicError, context)
}

func NewStackOverflowError(context string) error {
	return fmt.Errorf("%w: %s", ErrStackOverflow, context)
}

func NewObjectAllocLimitError(context string) error {
	return fmt.Errorf("%w: %s", ErrObjectAllocLimit, context)
}

func NewBytesLimitError(context string) error {
	return fmt.Errorf("%w: %s", ErrBytesLimit, context)
}

func NewStringLimitError(context string) error {
	return fmt.Errorf("%w: %s", ErrStringLimit, context)
}

func NewDecodeBinarySizeError(obj Object, expected int, got int) error {
	return fmt.Errorf("%w: type %s expects %d bytes, got %d", ErrDecodeBinarySize, obj.TypeName(), expected, got)
}

func NewBinaryNotSupportedError(obj Object) error {
	return fmt.Errorf("%w: type %s", ErrBinaryNotSupported, obj.TypeName())
}

func NewInvalidArgumentTypeError(context string, name string, expected string, got Object) error {
	return fmt.Errorf("%w: %s argument '%s' expects type %s, got %s", ErrInvalidArgumentType, context, name, expected, got.TypeName())
}

func NewIndexOutOfBoundsError(context string, idx int, size int) error {
	return fmt.Errorf("%w: %s: index %d out of range [0,%d)", ErrIndexOutOfBounds, context, idx, size)
}

func NewWrongNumArgumentsError(context string, expected string, got int) error {
	return fmt.Errorf("%w: %s: expected %s argument(s), got %d", ErrWrongNumArguments, context, expected, got)
}

func NewInvalidAccessModeError(dt string, mode string) error {
	return fmt.Errorf("%w: type %s does not support access mode '%s'", ErrInvalidAccessMode, dt, mode)
}

func NewNotAccessibleError(obj Object) error {
	return fmt.Errorf("%w: type %s does not support indexing or field access", ErrNotAccessible, obj.TypeName())
}

func NewNotAssignableError(obj Object) error {
	return fmt.Errorf("%w: type %s does not support assignment via indexing or field access", ErrNotAssignable, obj.TypeName())
}

func NewInvalidIndexTypeError(context string, expected string, got Object) error {
	return fmt.Errorf("%w: %s: expected %s, got %s", ErrInvalidIndexType, context, expected, got.TypeName())
}

func NewInvalidSelectorError(obj Object, sel string) error {
	return fmt.Errorf("%w: type %s has no selector '%s'", ErrInvalidSelector, obj.TypeName(), sel)
}

func NewNotImplementedError(feature string) error {
	return fmt.Errorf("%w: %s", ErrNotImplemented, feature)
}

func NewInvalidBinaryOperatorError(op string, left Object, right Object) error {
	return fmt.Errorf("%w: %s %s %s", ErrInvalidBinaryOperator, left.TypeName(), op, right.TypeName())
}
