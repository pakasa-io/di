package di

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode identifies a category of DI error.
type ErrorCode int

// ErrorCode constants classify the errors returned by the package.
const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeTypeNotRegistered
	ErrorCodeInvalidFactory
	ErrorCodeCircularDependency
	ErrorCodeDependencyResolution
	ErrorCodeScopeClosed
	ErrorCodeInvalidOption
	ErrorCodeUnsupportedAPIShape
	ErrorCodeMultipleBindings
	ErrorCodeBindingNotFound
	ErrorCodeNoFactory
	ErrorCodeNotAFunction
	ErrorCodeDependencyInjectionFailed
	ErrorCodeInvalidStruct
	ErrorCodeNilScope
	ErrorCodeNoBindingsFound
	ErrorCodeInvalidFunction
	ErrorCodeInvalidLifetime
	ErrorCodeScopeRequired
	ErrorCodeContainerClose
	ErrorCodeInvalidLifetimeGraph
)

func (c ErrorCode) String() string {
	switch c {
	case ErrorCodeTypeNotRegistered:
		return "TYPE_NOT_REGISTERED"
	case ErrorCodeInvalidFactory:
		return "INVALID_FACTORY"
	case ErrorCodeCircularDependency:
		return "CIRCULAR_DEPENDENCY"
	case ErrorCodeDependencyResolution:
		return "DEPENDENCY_RESOLUTION"
	case ErrorCodeScopeClosed:
		return "SCOPE_CLOSED"
	case ErrorCodeInvalidOption:
		return "INVALID_OPTION"
	case ErrorCodeUnsupportedAPIShape:
		return "UNSUPPORTED_API_SHAPE"
	case ErrorCodeMultipleBindings:
		return "MULTIPLE_BINDINGS"
	case ErrorCodeBindingNotFound:
		return "BINDING_NOT_FOUND"
	case ErrorCodeNoFactory:
		return "NO_FACTORY"
	case ErrorCodeNotAFunction:
		return "NOT_A_FUNCTION"
	case ErrorCodeDependencyInjectionFailed:
		return "DEPENDENCY_INJECTION_FAILED"
	case ErrorCodeInvalidStruct:
		return "INVALID_STRUCT"
	case ErrorCodeNilScope:
		return "NIL_SCOPE"
	case ErrorCodeNoBindingsFound:
		return "NO_BINDINGS_FOUND"
	case ErrorCodeInvalidFunction:
		return "INVALID_FUNCTION"
	case ErrorCodeInvalidLifetime:
		return "INVALID_LIFETIME"
	case ErrorCodeScopeRequired:
		return "SCOPE_REQUIRED"
	case ErrorCodeContainerClose:
		return "CONTAINER_CLOSE_ERROR"
	case ErrorCodeInvalidLifetimeGraph:
		return "INVALID_LIFETIME_GRAPH"
	default:
		return "UNKNOWN"
	}
}

// Error types
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
	Trace   []string
}

func (e *Error) Error() string {
	base := fmt.Sprintf("di: %s: %s", e.Code, e.Message)
	if len(e.Trace) > 0 {
		base = fmt.Sprintf("%s [trace: %s]", base, strings.Join(e.Trace, " -> "))
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", base, e.Cause)
	}
	return base
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Is reports whether target is an [Error] with the same [ErrorCode].
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == other.Code
}

// Common error codes
var (
	ErrTypeNotRegistered    = &Error{Code: ErrorCodeTypeNotRegistered, Message: "type not registered"}
	ErrInvalidFactory       = &Error{Code: ErrorCodeInvalidFactory, Message: "invalid factory function"}
	ErrCircularDependency   = &Error{Code: ErrorCodeCircularDependency, Message: "circular dependency detected"}
	ErrDependencyResolution = &Error{Code: ErrorCodeDependencyResolution, Message: "failed to resolve dependency"}
	ErrScopeClosed          = &Error{Code: ErrorCodeScopeClosed, Message: "scope is closed"}
	ErrInvalidOption        = &Error{Code: ErrorCodeInvalidOption, Message: "invalid option"}
	ErrUnsupportedAPIShape  = &Error{Code: ErrorCodeUnsupportedAPIShape, Message: "unsupported API shape"}
	ErrMultipleBindings     = &Error{Code: ErrorCodeMultipleBindings, Message: "multiple bindings found"}
	ErrBindingNotFound      = &Error{Code: ErrorCodeBindingNotFound, Message: "binding not found"}
)

func newError(code ErrorCode, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func newErrorWithTrace(code ErrorCode, message string, cause error, trace []string) error {
	return &Error{Code: code, Message: message, Cause: cause, Trace: append([]string(nil), trace...)}
}

func attachTrace(err error, trace []string) error {
	if err == nil || len(trace) == 0 {
		return err
	}

	var diErr *Error
	if errors.As(err, &diErr) {
		if len(diErr.Trace) > 0 {
			return err
		}
		clone := *diErr
		clone.Trace = append([]string(nil), trace...)
		return &clone
	}

	return &Error{
		Code:    ErrDependencyResolution.Code,
		Message: ErrDependencyResolution.Message,
		Cause:   err,
		Trace:   append([]string(nil), trace...),
	}
}

// ValidationError aggregates one or more graph validation issues.
type ValidationError struct {
	Issues []error
}

func validationTrace(err error) []string {
	var diErr *Error
	if errors.As(err, &diErr) {
		return append([]string(nil), diErr.Trace...)
	}
	return nil
}

func formatTraceSnapshot(trace []string) string {
	if len(trace) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n   graph:")
	for i, node := range trace {
		b.WriteByte('\n')
		b.WriteString("   ")
		b.WriteString(strings.Repeat("  ", i))
		b.WriteString("- ")
		b.WriteString(node)
	}
	return b.String()
}

func (e *ValidationError) Error() string {
	switch len(e.Issues) {
	case 0:
		return "di: validation failed"
	case 1:
		return e.Issues[0].Error()
	default:
		return fmt.Sprintf("di: validation failed with %d issues", len(e.Issues))
	}
}

func (e *ValidationError) Unwrap() error {
	return errors.Join(e.Issues...)
}

// Summary returns a readable multi-line summary of validation issues.
func (e *ValidationError) Summary() string {
	if e == nil || len(e.Issues) == 0 {
		return "di: validation passed"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "di validation failed with %d issue(s):\n", len(e.Issues))
	for i, issue := range e.Issues {
		fmt.Fprintf(&b, "%d. %v", i+1, issue)
		b.WriteString(formatTraceSnapshot(validationTrace(issue)))
		if i < len(e.Issues)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// FormatValidation formats validation errors for startup or CLI output.
func FormatValidation(err error) string {
	if err == nil {
		return "di: validation passed"
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Summary()
	}
	return err.Error()
}
