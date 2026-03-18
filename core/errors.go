package core

import (
	"errors"
	"fmt"
)

var (
	ErrStackOverflow       = errors.New("stack overflow")
	ErrObjectAllocLimit    = errors.New("object allocation limit exceeded")
	ErrBytesLimit          = errors.New("bytes size limit exceeded")
	ErrStringLimit         = errors.New("string size limit exceeded")
	ErrDecodeBinarySize    = errors.New("invalid binary size")
	ErrBinaryNotSupported  = errors.New("binary serialization not supported")
	ErrInvalidArgumentType = errors.New("invalid argument type")
)

func StackOverflow(context string) error {
	return fmt.Errorf("%w: %s", ErrStackOverflow, context)
}

func ObjectAllocLimit(context string) error {
	return fmt.Errorf("%w: %s", ErrObjectAllocLimit, context)
}

func BytesLimit(context string) error {
	return fmt.Errorf("%w: %s", ErrBytesLimit, context)
}

func StringLimit(context string) error {
	return fmt.Errorf("%w: %s", ErrStringLimit, context)
}

func DecodeBinarySize(obj Object, expected int, got int) error {
	return fmt.Errorf("%w: type %s expects %d bytes, got %d", ErrDecodeBinarySize, obj.TypeName(), expected, got)
}

func BinaryNotSupported(obj Object) error {
	return fmt.Errorf("%w: type %s", ErrBinaryNotSupported, obj.TypeName())
}

func InvalidArgumentType(context string, name string, expected string, got Object) error {
	return fmt.Errorf("%w: %s argument '%s' expects type %s, got %s", ErrInvalidArgumentType, context, name, expected, got.TypeName())
}
