// Package errs defines the structured error type used throughout the Kavun runtime.
//
// Severity policy:
//   - Fatal: VM/host invariant violation (resource exhaustion, internal logic errors, host-side setup mistakes).
//     Bypasses recover; the VM stops and the error escapes to the host.
//   - Recoverable: ordinary script-level mistakes (wrong types, bad indexing, division by zero, missing methods, etc).
//     Visible to deferred recover().
package errs

import (
	"errors"
	"fmt"

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
	KindEncoding              = "encoding"

	// Fatal kinds.
	KindStackOverflow = "stack_overflow"
	KindResourceLimit = "resource_limit"
	KindInternal      = "internal"
	KindHost          = "host"
)

var (
	ErrDivisionByZero        = &Error{Kind: KindDivisionByZero, Severity: Recoverable}
	ErrInvalidArgumentType   = &Error{Kind: KindInvalidArgumentType, Severity: Recoverable}
	ErrIndexOutOfBounds      = &Error{Kind: KindIndexOutOfBounds, Severity: Recoverable}
	ErrWrongNumArguments     = &Error{Kind: KindWrongNumArguments, Severity: Recoverable}
	ErrNotAccessible         = &Error{Kind: KindNotAccessible, Severity: Recoverable}
	ErrNotAssignable         = &Error{Kind: KindNotAssignable, Severity: Recoverable}
	ErrNotCallable           = &Error{Kind: KindNotCallable, Severity: Recoverable}
	ErrNotIterable           = &Error{Kind: KindNotIterable, Severity: Recoverable}
	ErrNotAppendable         = &Error{Kind: KindNotAppendable, Severity: Recoverable}
	ErrNotDeletable          = &Error{Kind: KindNotDeletable, Severity: Recoverable}
	ErrNotSliceable          = &Error{Kind: KindNotSliceable, Severity: Recoverable}
	ErrInvalidIndexType      = &Error{Kind: KindInvalidIndexType, Severity: Recoverable}
	ErrInvalidSelector       = &Error{Kind: KindInvalidSelector, Severity: Recoverable}
	ErrInvalidUnaryOperator  = &Error{Kind: KindInvalidUnaryOperator, Severity: Recoverable}
	ErrInvalidBinaryOperator = &Error{Kind: KindInvalidBinaryOperator, Severity: Recoverable}
	ErrInvalidMethod         = &Error{Kind: KindInvalidMethod, Severity: Recoverable}
	ErrUnsupportedFormatSpec = &Error{Kind: KindUnsupportedFormatSpec, Severity: Recoverable}
	ErrNotImplemented        = &Error{Kind: KindNotImplemented, Severity: Recoverable}
	ErrConversion            = &Error{Kind: KindConversion, Severity: Recoverable}
	ErrInvalidValue          = &Error{Kind: KindInvalidValue, Severity: Recoverable}
	ErrModuleNotFound        = &Error{Kind: KindModuleNotFound, Severity: Recoverable}
	ErrUndefinedVariable     = &Error{Kind: KindUndefinedVariable, Severity: Recoverable}
	ErrEncoding              = &Error{Kind: KindEncoding, Severity: Recoverable}

	ErrStackOverflow = &Error{Kind: KindStackOverflow, Severity: Fatal}
	ErrResourceLimit = &Error{Kind: KindResourceLimit, Severity: Fatal}
	ErrInternal      = &Error{Kind: KindInternal, Severity: Fatal}
	ErrHost          = &Error{Kind: KindHost, Severity: Fatal}
)

// Severity classifies errors for the recovery mechanism.
type Severity uint8

const (
	Fatal       Severity = 0 // stop the VM immediately; script code defers do NOT run, recover() cannot see them
	Recoverable Severity = 1 // visible to deferred recover() in script code
)

func (s Severity) String() string {
	switch s {
	case Fatal:
		return "fatal"
	case Recoverable:
		return "recoverable"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}

// Error is the structured runtime error used across the Kavun runtime.
// Construction MUST go through one of the constructors in this package so that Kind and Severity stay consistent for a
// given Kind. Constructing *Error literals directly is supported but discouraged; the convention is
// "one Kind, one Severity".
type Error struct {
	Message  string   // human-readable detail
	Kind     string   // stable machine-readable tag (e.g. "division_by_zero")
	Severity Severity // recoverability policy
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Kind
	}
	return e.Kind + ": " + e.Message
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
// Errors that do not implement *Error default to Fatal.
func IsCritical(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Severity == Fatal
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
		Message:  message,
		Kind:     kind,
		Severity: Recoverable,
	}
}

// NewFatalError constructs a fatal (VM-stopping) error with the given kind tag.
func NewFatalError(kind, message string) *Error {
	return &Error{
		Message:  message,
		Kind:     kind,
		Severity: Fatal,
	}
}

func NewDivisionByZeroError() *Error {
	return &Error{Kind: KindDivisionByZero, Severity: Recoverable, Message: "division by zero"}
}

func NewStackOverflowError(context string) *Error {
	return &Error{Kind: KindStackOverflow, Severity: Fatal, Message: context}
}

func NewResourceLimitError(detail string) *Error {
	return &Error{Kind: KindResourceLimit, Severity: Fatal, Message: detail}
}

func NewInternalError(context string) *Error {
	return &Error{Kind: KindInternal, Severity: Fatal, Message: context}
}

func NewHostError(context string) *Error {
	return &Error{Kind: KindHost, Severity: Fatal, Message: context}
}

func NewInvalidArgumentTypeError(context string, name string, expected string, got string) *Error {
	return &Error{
		Kind:     KindInvalidArgumentType,
		Severity: Recoverable,
		Message:  fmt.Sprintf("(%s) argument %s expects type %s, got %s", context, name, expected, got),
	}
}

func NewIndexOutOfBoundsError(context string, idx int, size int) *Error {
	return &Error{
		Kind:     KindIndexOutOfBounds,
		Severity: Recoverable,
		Message:  fmt.Sprintf("(%s) %d out of range [0, %d]", context, idx, size),
	}
}

func NewWrongNumArgumentsError(context string, expected string, got int) *Error {
	return &Error{
		Kind:     KindWrongNumArguments,
		Severity: Recoverable,
		Message:  fmt.Sprintf("(%s) expected %s argument(s), got %d", context, expected, got),
	}
}

func NewNotAccessibleError(valType string) *Error {
	return &Error{
		Kind:     KindNotAccessible,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support indexing or field access", valType),
	}
}

func NewNotAssignableError(valType string) *Error {
	return &Error{
		Kind:     KindNotAssignable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support assignment via indexing or field access", valType),
	}
}

func NewNotCallableError(valType string) *Error {
	return &Error{
		Kind:     KindNotCallable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s is not callable", valType),
	}
}

func NewNotIterableError(valType string) *Error {
	return &Error{
		Kind:     KindNotIterable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s is not iterable", valType),
	}
}

func NewNotAppendableError(valType string) *Error {
	return &Error{
		Kind:     KindNotAppendable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support append", valType),
	}
}

func NewNotDeletableError(valType string) *Error {
	return &Error{
		Kind:     KindNotDeletable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support delete", valType),
	}
}

func NewNotSliceableError(valType string) *Error {
	return &Error{
		Kind:     KindNotSliceable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support slicing", valType),
	}
}

func NewSliceStepZeroError() *Error {
	return &Error{
		Kind:     KindNotSliceable,
		Severity: Recoverable,
		Message:  "step cannot be zero",
	}
}

func NewInvalidIndexTypeError(context string, expected string, got string) *Error {
	return &Error{
		Kind:     KindInvalidIndexType,
		Severity: Recoverable,
		Message:  fmt.Sprintf("(%s) expected %s, got %s", context, expected, got),
	}
}

func NewInvalidSelectorError(valType string, sel string) *Error {
	return &Error{
		Kind:     KindInvalidSelector,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s has no property %s", valType, sel),
	}
}

func NewNotImplementedError(feature string) *Error {
	return &Error{
		Kind:     KindNotImplemented,
		Severity: Recoverable,
		Message:  feature,
	}
}

func NewInvalidUnaryOperatorError(op string, valType string) *Error {
	return &Error{
		Kind:     KindInvalidUnaryOperator,
		Severity: Recoverable,
		Message:  fmt.Sprintf("%s %s", op, valType),
	}
}

func NewInvalidBinaryOperatorError(op string, left string, right string) *Error {
	return &Error{
		Kind:     KindInvalidBinaryOperator,
		Severity: Recoverable,
		Message:  fmt.Sprintf("%s %s %s", left, op, right),
	}
}

func NewInvalidMethodError(method string, valType string) *Error {
	return &Error{
		Kind:     KindInvalidMethod,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s has no method %s", valType, method),
	}
}

func NewUnsupportedFormatSpec(valType string, spec fspec.FormatSpec) *Error {
	return &Error{
		Kind:     KindUnsupportedFormatSpec,
		Severity: Recoverable,
		Message:  fmt.Sprintf("type %s does not support format spec %v", valType, spec),
	}
}

func NewConversionError(from string, to string, detail string) *Error {
	msg := fmt.Sprintf("cannot convert %s to %s", from, to)
	if detail != "" {
		msg += ": " + detail
	}
	return &Error{Kind: KindConversion, Severity: Recoverable, Message: msg}
}

func NewInvalidValueError(detail string) *Error {
	return &Error{Kind: KindInvalidValue, Severity: Recoverable, Message: detail}
}

func NewModuleNotFoundError(name string, path string) *Error {
	return &Error{
		Kind:     KindModuleNotFound,
		Severity: Recoverable,
		Message:  fmt.Sprintf("module '%s' not found at: %s", name, path),
	}
}

func NewUndefinedVariableError(name string) *Error {
	return &Error{
		Kind:     KindUndefinedVariable,
		Severity: Recoverable,
		Message:  fmt.Sprintf("'%s' is not defined", name),
	}
}

func NewEncodingError(detail string) *Error {
	return &Error{Kind: KindEncoding, Severity: Recoverable, Message: detail}
}
