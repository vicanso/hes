package hes

import (
	"errors"
	"net/http"
)

// Option configures an Error at construction time.
type Option func(*Error)

// WithStatus sets the HTTP status code.
func WithStatus(statusCode int) Option {
	return func(e *Error) {
		e.StatusCode = statusCode
	}
}

// WithCode sets the application error code.
func WithCode(code string) Option {
	return func(e *Error) {
		e.Code = code
	}
}

// WithCategory sets the error category.
func WithCategory(category string) Option {
	return func(e *Error) {
		e.Category = category
	}
}

// WithSubCategory sets the error sub-category.
func WithSubCategory(subCategory string) Option {
	return func(e *Error) {
		e.SubCategory = subCategory
	}
}

// WithTitle sets the error title.
func WithTitle(title string) Option {
	return func(e *Error) {
		e.Title = title
	}
}

// WithException marks the error as an unexpected exception.
func WithException() Option {
	return func(e *Error) {
		e.Exception = true
	}
}

// WithExtra sets a single extra metadata entry.
func WithExtra(key string, value any) Option {
	return func(e *Error) {
		if e.Extra == nil {
			e.Extra = make(map[string]any, 1)
		}
		e.Extra[key] = value
	}
}

// WithExtras merges metadata entries into Extra.
func WithExtras(extra map[string]any) Option {
	return func(e *Error) {
		if len(extra) == 0 {
			return
		}
		if e.Extra == nil {
			e.Extra = make(map[string]any, len(extra))
		}
		for k, v := range extra {
			e.Extra[k] = v
		}
	}
}

// WithCause sets the underlying cause.
func WithCause(cause error) Option {
	return func(e *Error) {
		e.Cause = cause
	}
}

// WithCaller captures the call site when the option is created.
// skip is additional frames above the WithCaller call itself (0 = direct caller).
func WithCaller(skip int) Option {
	// +1 = captureCaller, +1 = WithCaller.
	file, line, ok := captureCaller(skip + 2)
	return func(e *Error) {
		if !ok {
			return
		}
		e.File = file
		e.Line = line
	}
}

// New creates a client-facing HTTP error (default status 400).
func New(message string, opts ...Option) *Error {
	e := &Error{
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
	apply(e, opts)
	if defaultCaller.Load() && e.File == "" {
		// skip=1 → New's caller (see setCaller docs).
		e.setCaller(1)
	}
	return e
}

// Wrap converts err into an *Error.
//
//   - nil → nil
//   - error chain containing *Error, no opts → that *Error (same pointer)
//   - error chain containing *Error, with opts → deep clone with options applied
//   - other errors → new *Error with status 500, Exception=true, Cause=err
func Wrap(err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	var he *Error
	if errors.As(err, &he) {
		if len(opts) == 0 {
			return he
		}
		clone := he.Clone()
		apply(clone, opts)
		return clone
	}
	e := &Error{
		Message:    err.Error(),
		StatusCode: http.StatusInternalServerError,
		Exception:  true,
		Cause:      err,
	}
	apply(e, opts)
	if defaultCaller.Load() && e.File == "" {
		e.setCaller(1)
	}
	return e
}

// From returns err as *Error using errors.As when possible; otherwise Wrap(err).
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var he *Error
	if errors.As(err, &he) {
		return he
	}
	return Wrap(err)
}

// Is reports whether err's chain contains an *Error.
func Is(err error) bool {
	var he *Error
	return errors.As(err, &he)
}

// As extracts the first *Error from err's chain.
func As(err error) (*Error, bool) {
	var he *Error
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

// BadRequest creates a 400 error (alias of New).
func BadRequest(message string, opts ...Option) *Error {
	return New(message, opts...)
}

// Unauthorized creates a 401 error.
func Unauthorized(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusUnauthorized), opts)...)
}

// Forbidden creates a 403 error.
func Forbidden(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusForbidden), opts)...)
}

// NotFound creates a 404 error.
func NotFound(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusNotFound), opts)...)
}

// RequestTimeout creates a 408 error.
func RequestTimeout(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusRequestTimeout), opts)...)
}

// Conflict creates a 409 error.
func Conflict(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusConflict), opts)...)
}

// UnsupportedMediaType creates a 415 error.
func UnsupportedMediaType(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusUnsupportedMediaType), opts)...)
}

// UnprocessableEntity creates a 422 error.
func UnprocessableEntity(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusUnprocessableEntity), opts)...)
}

// TooManyRequests creates a 429 error.
func TooManyRequests(message string, opts ...Option) *Error {
	return New(message, withLeading(WithStatus(http.StatusTooManyRequests), opts)...)
}

// Internal creates a 500 exception error.
func Internal(message string, opts ...Option) *Error {
	leading := []Option{
		WithStatus(http.StatusInternalServerError),
		WithException(),
	}
	return New(message, append(leading, opts...)...)
}

// BadGateway creates a 502 exception error.
func BadGateway(message string, opts ...Option) *Error {
	leading := []Option{
		WithStatus(http.StatusBadGateway),
		WithException(),
	}
	return New(message, append(leading, opts...)...)
}

// ServiceUnavailable creates a 503 exception error.
func ServiceUnavailable(message string, opts ...Option) *Error {
	leading := []Option{
		WithStatus(http.StatusServiceUnavailable),
		WithException(),
	}
	return New(message, append(leading, opts...)...)
}

// GatewayTimeout creates a 504 exception error.
func GatewayTimeout(message string, opts ...Option) *Error {
	leading := []Option{
		WithStatus(http.StatusGatewayTimeout),
		WithException(),
	}
	return New(message, append(leading, opts...)...)
}

func apply(e *Error, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
}

func withLeading(opt Option, opts []Option) []Option {
	out := make([]Option, 0, len(opts)+1)
	out = append(out, opt)
	out = append(out, opts...)
	return out
}
