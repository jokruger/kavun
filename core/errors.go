package core

import (
	"errors"
	"fmt"
)

var (
	ErrStackOverflow    = errors.New("stack overflow")
	ErrObjectAllocLimit = errors.New("object allocation limit exceeded")
	ErrBytesLimit       = errors.New("bytes size limit exceeded")
	ErrStringLimit      = errors.New("string size limit exceeded")
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
