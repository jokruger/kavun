package errs

import (
	"errors"
	"fmt"

	"github.com/jokruger/kavun/fspec"
)

var (
	ErrDivisionByZero        = errors.New("division by zero")
	ErrLogic                 = errors.New("logic error")
	ErrStackOverflow         = errors.New("stack overflow")
	ErrObjectAllocLimit      = errors.New("object allocation limit exceeded")
	ErrBytesLimit            = errors.New("bytes size limit exceeded")
	ErrStringLimit           = errors.New("string size limit exceeded")
	ErrInvalidArgumentType   = errors.New("invalid argument type")
	ErrIndexOutOfBounds      = errors.New("index out of bounds")
	ErrWrongNumArguments     = errors.New("wrong number of arguments")
	ErrInvalidAccessMode     = errors.New("invalid access mode")
	ErrNotAccessible         = errors.New("object is not accessible")
	ErrNotAssignable         = errors.New("object is not assignable")
	ErrNotCallable           = errors.New("object is not callable")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidSelector       = errors.New("invalid selector")
	ErrNotImplemented        = errors.New("not implemented")
	ErrInvalidUnaryOperator  = errors.New("invalid unary operator")
	ErrInvalidBinaryOperator = errors.New("invalid binary operator")
	ErrInvalidMethod         = errors.New("invalid method error")
	ErrInvalidAppend         = errors.New("invalid append error")
	ErrInvalidDelete         = errors.New("invalid delete error")
	ErrInvalidSlice          = errors.New("invalid slice error")
	ErrUnsupportedFormatSpec = errors.New("unsupported format spec")
)

func NewLogicError(context string) error {
	return fmt.Errorf("%w: %s", ErrLogic, context)
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

func NewInvalidArgumentTypeError(context string, name string, expected string, got string) error {
	return fmt.Errorf("%w: (%s) argument %s expects type %s, got %s", ErrInvalidArgumentType, context, name, expected, got)
}

func NewIndexOutOfBoundsError(context string, idx int, size int) error {
	return fmt.Errorf("%w: (%s) %d out of range [0, %d]", ErrIndexOutOfBounds, context, idx, size)
}

func NewWrongNumArgumentsError(context string, expected string, got int) error {
	return fmt.Errorf("%w: (%s) expected %s argument(s), got %d", ErrWrongNumArguments, context, expected, got)
}

func NewInvalidAccessModeError(dt string, mode string) error {
	return fmt.Errorf("%w: type %s does not support %s access", ErrInvalidAccessMode, dt, mode)
}

func NewNotAccessibleError(valType string) error {
	return fmt.Errorf("%w: type %s does not support indexing or field access", ErrNotAccessible, valType)
}

func NewNotAssignableError(valType string) error {
	return fmt.Errorf("%w: type %s does not support assignment via indexing or field access", ErrNotAssignable, valType)
}

func NewNotCallableError(valType string) error {
	return fmt.Errorf("%w: type %s does not support function call", ErrNotCallable, valType)
}

func NewInvalidIndexTypeError(context string, expected string, got string) error {
	return fmt.Errorf("%w: (%s) expected %s, got %s", ErrInvalidIndexType, context, expected, got)
}

func NewInvalidSelectorError(valType string, sel string) error {
	return fmt.Errorf("%w: type %s has no property %s", ErrInvalidSelector, valType, sel)
}

func NewNotImplementedError(feature string) error {
	return fmt.Errorf("%w: %s", ErrNotImplemented, feature)
}

func NewInvalidUnaryOperatorError(op string, valType string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidUnaryOperator, op, valType)
}

func NewInvalidBinaryOperatorError(op string, left string, right string) error {
	return fmt.Errorf("%w: %s %s %s", ErrInvalidBinaryOperator, left, op, right)
}

func NewInvalidMethodError(method string, valType string) error {
	return fmt.Errorf("%w: type %s has no method %s", ErrInvalidMethod, valType, method)
}

func NewInvalidAppendError(valType string) error {
	return fmt.Errorf("%w: type %s does not support append", ErrInvalidAppend, valType)
}

func NewInvalidDeleteError(valType string) error {
	return fmt.Errorf("%w: type %s does not support delete", ErrInvalidDelete, valType)
}

func NewInvalidSliceError(valType string) error {
	return fmt.Errorf("%w: type %s does not support slicing", ErrInvalidSlice, valType)
}

func NewSliceStepZeroError() error {
	return fmt.Errorf("%w: step cannot be zero", ErrInvalidSlice)
}

func NewUnsupportedFormatSpec(valType string, spec fspec.FormatSpec) error {
	return fmt.Errorf("%w: type %s does not support format spec %v", ErrUnsupportedFormatSpec, valType, spec)
}
