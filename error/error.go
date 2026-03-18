package error

import (
	"errors"
)

var (
	ErrIndexOutOfBounds    = errors.New("index out of bounds")
	ErrInvalidAccessMode   = errors.New("invalid access mode")
	ErrInvalidIndexType    = errors.New("invalid index type")
	ErrInvalidIndexOnError = errors.New("invalid index on error")
	ErrInvalidOperator     = errors.New("invalid operator")
	ErrWrongNumArguments   = errors.New("wrong number of arguments")
	ErrNotIndexable        = errors.New("not indexable")
	ErrNotIndexAssignable  = errors.New("not index-assignable")
	ErrNotImplemented      = errors.New("not implemented")
	ErrInvalidRangeStep    = errors.New("range step must be greater than 0")
)
