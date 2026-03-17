package error

import (
	"errors"
)

var (
	ErrStackOverflow         = errors.New("stack overflow")
	ErrObjectAllocLimit      = errors.New("object allocation limit exceeded")
	ErrIndexOutOfBounds      = errors.New("index out of bounds")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidIndexValueType = errors.New("invalid index value type")
	ErrInvalidIndexOnError   = errors.New("invalid index on error")
	ErrInvalidOperator       = errors.New("invalid operator")
	ErrWrongNumArguments     = errors.New("wrong number of arguments")
	ErrBytesLimit            = errors.New("exceeding bytes size limit")
	ErrStringLimit           = errors.New("exceeding string size limit")
	ErrNotIndexable          = errors.New("not indexable")
	ErrNotIndexAssignable    = errors.New("not index-assignable")
	ErrNotImplemented        = errors.New("not implemented")
	ErrInvalidRangeStep      = errors.New("range step must be greater than 0")
)
