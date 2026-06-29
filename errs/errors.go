// Package errs defines the structured error type used throughout the Kavun runtime.
//
// Severity policy (encoded as the Recoverable bool flag):
//   - Recoverable (Recoverable=true): ordinary script-level mistakes (wrong types, bad indexing, division by zero,
//     missing methods, etc). Visible to deferred recover().
//   - Fatal (Recoverable=false, the zero value for safety): VM/host invariant violation (resource exhaustion,
//     internal logic errors, host-side setup mistakes). Bypasses recover; the VM stops and the error escapes to
//     the host.
package errs

import (
	"errors"
	"fmt"

	"github.com/jokruger/kavun/core/opcode"
	"github.com/jokruger/kavun/fspec"
)

const (
	// Recoverable kinds.
	KindDivisionByZero        = "division_by_zero"
	KindInvalidArgumentType   = "invalid_argument_type"
	KindIndexOutOfBounds      = "index_out_of_bounds"
	KindWrongNumArguments     = "wrong_num_arguments"
	KindNotAccessible         = "not_accessible"
	KindNotAssignable         = "not_assignable"
	KindNotCallable           = "not_callable"
	KindNotIterable           = "not_iterable"
	KindNotAppendable         = "not_appendable"
	KindNotDeletable          = "not_deletable"
	KindNotSliceable          = "not_sliceable"
	KindInvalidIndexType      = "invalid_index_type"
	KindInvalidSelector       = "invalid_selector"
	KindInvalidUnaryOperator  = "invalid_unary_operator"
	KindInvalidBinaryOperator = "invalid_binary_operator"
	KindInvalidMethod         = "invalid_method"
	KindUnsupportedFormatSpec = "unsupported_format_spec"
	KindNotImplemented        = "not_implemented"
	KindConversion            = "conversion"
	KindInvalidValue          = "invalid_value"
	KindModuleNotFound        = "module_not_found"
	KindUndefinedVariable     = "undefined_variable"
	KindJSONEncoding          = "json_encoding"
	KindBinaryEncoding        = "binary_encoding"
	KindFormatting            = "formatting"

	// Fatal kinds.
	KindInvalidOperand = "invalid_operand"
	KindStackOverflow  = "stack_overflow"
	KindResourceLimit  = "resource_limit"
	KindInternal       = "internal"
	KindHost           = "host"
)

var (
	ErrDivisionByZero        = &Error{Kind: KindDivisionByZero, Recoverable: true}
	ErrInvalidArgumentType   = &Error{Kind: KindInvalidArgumentType, Recoverable: true}
	ErrUnsupportedFormatSpec = &Error{Kind: KindUnsupportedFormatSpec, Recoverable: true}
	ErrStackOverflow         = &Error{Kind: KindStackOverflow}
)

// Error is the structured runtime error used across the Kavun runtime.
// Construction MUST go through one of the constructors in this package so that Kind and Recoverable stay consistent
// for a given Kind. Constructing *Error literals directly is supported but discouraged; the convention is
// "one Kind, one severity". The zero value is a *fatal* error (Recoverable=false): this default biases unfamiliar /
// hand-constructed errors toward stopping the VM rather than silently being swallowed by recover().
type Error struct {
	Message     string // human-readable detail
	Kind        string // stable machine-readable tag (e.g. "division_by_zero")
	Recoverable bool   // if true, visible to deferred recover(); if false, bypasses recover and stops the VM
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Kind
	}
	return e.Kind + ": " + e.Message
}

// IsFatal reports whether the error should bypass recover() and stop the VM.
func (e *Error) IsFatal() bool {
	return !e.Recoverable
}

// IsRecoverable reports whether the error is visible to deferred recover().
func (e *Error) IsRecoverable() bool {
	return e.Recoverable
}

// Is matches by Kind so that errors.Is(err, sentinel) keeps working when callers compare against package-level sentinel
// values.
func (e *Error) Is(target error) bool {
	o, ok := target.(*Error)
	if !ok {
		return false
	}
	return o.Kind == e.Kind
}

// IsCritical reports whether err should bypass deferred recover() and stop the VM.
// Errors that do not implement *Error default to fatal.
func IsCritical(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := errors.AsType[*Error](err); ok {
		return !e.Recoverable
	}
	return true
}

// AsError extracts a *Error from any wrapped error chain. Returns nil if no *Error is present.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := errors.AsType[*Error](err); ok {
		return e
	}
	return nil
}

// NewRecoverableError constructs a recoverable error with the given kind tag.
// Use from third-party builtins/types when no specific helper applies.
func NewRecoverableError(kind, message string) *Error {
	return &Error{
		Message:     message,
		Kind:        kind,
		Recoverable: true,
	}
}

// NewFatalError constructs a fatal (VM-stopping) error with the given kind tag.
func NewFatalError(kind, message string) *Error {
	return &Error{
		Message: message,
		Kind:    kind,
	}
}

func NewDivisionByZeroError() *Error {
	return &Error{
		Kind:        KindDivisionByZero,
		Recoverable: true,
		Message:     "division by zero",
	}
}

func NewInvalidOperandError(op opcode.Opcode, index int, width int8, value int) *Error {
	return &Error{
		Kind:    KindInvalidOperand,
		Message: fmt.Sprintf("invalid operand for opcode %d at index %d: expected width %d byte(s), got value %d", op, index, width, value),
	}
}

func NewInvalidOperandCountError(op opcode.Opcode, expected int, got int) *Error {
	return &Error{
		Kind:    KindInvalidOperand,
		Message: fmt.Sprintf("invalid operand count for opcode %d: expected %d, got %d", op, expected, got),
	}
}

func NewStackOverflowError(context string) *Error {
	return &Error{
		Kind:    KindStackOverflow,
		Message: context,
	}
}

func NewResourceLimitError(detail string) *Error {
	return &Error{
		Kind:    KindResourceLimit,
		Message: detail,
	}
}

func NewAllocationLimitError(typeName string) *Error {
	return &Error{
		Kind:    KindResourceLimit,
		Message: fmt.Sprintf("allocation limit exceeded for type %s", typeName),
	}
}

func NewInternalError(context string) *Error {
	return &Error{
		Kind:    KindInternal,
		Message: context,
	}
}

func NewHostError(context string) *Error {
	return &Error{
		Kind:    KindHost,
		Message: context,
	}
}

func NewInvalidArgumentTypeError(context string, name string, expected string, got string) *Error {
	return &Error{
		Kind:        KindInvalidArgumentType,
		Recoverable: true,
		Message:     fmt.Sprintf("(%s) argument %s expects type %s, got %s", context, name, expected, got),
	}
}

func NewIndexOutOfBoundsError(context string, idx int, size int) *Error {
	return &Error{
		Kind:        KindIndexOutOfBounds,
		Recoverable: true,
		Message:     fmt.Sprintf("(%s) %d out of range [0, %d]", context, idx, size),
	}
}

func NewWrongNumArgumentsError(context string, expected string, got int) *Error {
	return &Error{
		Kind:        KindWrongNumArguments,
		Recoverable: true,
		Message:     fmt.Sprintf("(%s) expected %s argument(s), got %d", context, expected, got),
	}
}

func NewNotAccessibleError(valType string) *Error {
	return &Error{
		Kind:        KindNotAccessible,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support indexing or field access", valType),
	}
}

func NewNotAssignableError(valType string) *Error {
	return &Error{
		Kind:        KindNotAssignable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support assignment via indexing or field access", valType),
	}
}

func NewNotCallableError(valType string) *Error {
	return &Error{
		Kind:        KindNotCallable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s is not callable", valType),
	}
}

func NewNotIterableError(valType string) *Error {
	return &Error{
		Kind:        KindNotIterable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s is not iterable", valType),
	}
}

func NewNotAppendableError(valType string) *Error {
	return &Error{
		Kind:        KindNotAppendable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support append", valType),
	}
}

func NewNotDeletableError(valType string) *Error {
	return &Error{
		Kind:        KindNotDeletable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support delete", valType),
	}
}

func NewNotSliceableError(valType string) *Error {
	return &Error{
		Kind:        KindNotSliceable,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support slicing", valType),
	}
}

func NewSliceStepZeroError() *Error {
	return &Error{
		Kind:        KindNotSliceable,
		Recoverable: true,
		Message:     "step cannot be zero",
	}
}

func NewInvalidIndexTypeError(context string, expected string, got string) *Error {
	return &Error{
		Kind:        KindInvalidIndexType,
		Recoverable: true,
		Message:     fmt.Sprintf("(%s) expected %s, got %s", context, expected, got),
	}
}

func NewInvalidSelectorError(valType string, sel string) *Error {
	return &Error{
		Kind:        KindInvalidSelector,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s has no property %s", valType, sel),
	}
}

func NewNotImplementedError(feature string) *Error {
	return &Error{
		Kind:        KindNotImplemented,
		Recoverable: true,
		Message:     feature,
	}
}

func NewInvalidUnaryOperatorError(op string, valType string) *Error {
	return &Error{
		Kind:        KindInvalidUnaryOperator,
		Recoverable: true,
		Message:     fmt.Sprintf("%s %s", op, valType),
	}
}

func NewInvalidBinaryOperatorError(op string, left string, right string) *Error {
	return &Error{
		Kind:        KindInvalidBinaryOperator,
		Recoverable: true,
		Message:     fmt.Sprintf("%s %s %s", left, op, right),
	}
}

func NewInvalidMethodError(method string, valType string) *Error {
	return &Error{
		Kind:        KindInvalidMethod,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s has no method %s", valType, method),
	}
}

func NewUnsupportedFormatSpec(valType string, spec fspec.FormatSpec) *Error {
	return &Error{
		Kind:        KindUnsupportedFormatSpec,
		Recoverable: true,
		Message:     fmt.Sprintf("type %s does not support format spec %v", valType, spec),
	}
}

func NewConversionError(from string, to string, detail string) *Error {
	msg := fmt.Sprintf("cannot convert %s to %s", from, to)
	if detail != "" {
		msg += ": " + detail
	}
	return &Error{Kind: KindConversion, Recoverable: true, Message: msg}
}

func NewInvalidValueError(detail string) *Error {
	return &Error{Kind: KindInvalidValue, Recoverable: true, Message: detail}
}

func NewModuleNotFoundError(name string, path string) *Error {
	return &Error{
		Kind:        KindModuleNotFound,
		Recoverable: true,
		Message:     fmt.Sprintf("module '%s' not found at: %s", name, path),
	}
}

func NewUndefinedVariableError(name string) *Error {
	return &Error{
		Kind:        KindUndefinedVariable,
		Recoverable: true,
		Message:     fmt.Sprintf("'%s' is not defined", name),
	}
}

func NewJSONEncodingError(context string) *Error {
	return &Error{
		Kind:        KindJSONEncoding,
		Recoverable: true,
		Message:     context,
	}
}

func NewNoJSONEncodingError(valType string) *Error {
	return NewJSONEncodingError(fmt.Sprintf("value type %s does not support JSON encoding", valType))
}

func NewBinaryEncodingError(context string) *Error {
	return &Error{
		Kind:        KindBinaryEncoding,
		Recoverable: true,
		Message:     context,
	}
}

func NewNoBinaryEncodingError(valType string) *Error {
	return NewBinaryEncodingError(fmt.Sprintf("value type %s does not support binary encoding", valType))
}

func NewFormattingError(context string) *Error {
	return &Error{
		Kind:        KindFormatting,
		Recoverable: true,
		Message:     context,
	}
}

func NewNoFormattingError(valType string) *Error {
	return NewFormattingError(fmt.Sprintf("value type %s does not support formatting", valType))
}
