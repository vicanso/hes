package hes

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
)

// Error is a structured HTTP error.
//
// After construction, treat values as immutable: use With* / Append / Clone
// to derive new instances instead of mutating fields in place.
type Error struct {
	// StatusCode is the HTTP status code.
	StatusCode int `json:"statusCode,omitempty"`
	// Code is an application-defined error code.
	Code string `json:"code,omitempty"`
	// Category groups related errors (e.g. package or domain name).
	Category string `json:"category,omitempty"`
	// SubCategory further classifies the error within Category.
	SubCategory string `json:"subCategory,omitempty"`
	// Title is a short human-readable summary.
	Title string `json:"title,omitempty"`
	// Message is the primary error description.
	Message string `json:"message,omitempty"`
	// Exception marks unexpected server-side failures.
	Exception bool `json:"exception,omitempty"`
	// File is the source file from caller capture, if enabled.
	File string `json:"file,omitempty"`
	// Line is the source line from caller capture, if enabled.
	Line int `json:"line,omitempty"`
	// Extra holds optional structured metadata.
	Extra map[string]any `json:"extra,omitempty"`
	// Errs holds aggregated sub-errors (e.g. validation).
	Errs []*Error `json:"errs,omitempty"`
	// Cause is the underlying error, omitted from JSON.
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	parts := make([]string, 0, 8)
	if e.File != "" {
		parts = append(parts, fmt.Sprintf("file=%s, line=%d", e.File, e.Line))
	}
	if e.StatusCode != 0 {
		parts = append(parts, fmt.Sprintf("statusCode=%d", e.StatusCode))
	}
	if e.Category != "" {
		parts = append(parts, fmt.Sprintf("category=%s", e.Category))
	}
	if e.SubCategory != "" {
		parts = append(parts, fmt.Sprintf("subCategory=%s", e.SubCategory))
	}
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}
	if e.Title != "" {
		parts = append(parts, fmt.Sprintf("title=%s", e.Title))
	}
	parts = append(parts, fmt.Sprintf("message=%s", e.Message))
	if e.Exception {
		parts = append(parts, "exception=true")
	}

	str := strings.Join(parts, ", ")
	if len(e.Errs) == 0 {
		return str
	}

	arr := make([]string, len(e.Errs))
	for i, err := range e.Errs {
		arr[i] = err.Error()
	}
	return fmt.Sprintf("%s, errs:(%s)", str, strings.Join(arr, ","))
}

// Format implements fmt.Formatter.
//
//	%s, %v   — Error()
//	%q       — quoted Message
//	%+v      — Error() plus cause chain and call site
func (e *Error) Format(s fmt.State, verb rune) {
	if e == nil {
		_, _ = io.WriteString(s, "<nil>")
		return
	}
	switch verb {
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Message)
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.formatVerbose())
			return
		}
		fallthrough
	default:
		_, _ = io.WriteString(s, e.Error())
	}
}

func (e *Error) formatVerbose() string {
	var b strings.Builder
	b.WriteString(e.Error())
	for c := e.Cause; c != nil; {
		fmt.Fprintf(&b, "\n  - cause: %s", c.Error())
		switch u := c.(type) {
		case interface{ Unwrap() error }:
			c = u.Unwrap()
		default:
			c = nil
		}
	}
	return b.String()
}

// Unwrap returns the underlying cause, or joined sub-errors when present.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Cause != nil {
		if len(e.Errs) == 0 {
			return e.Cause
		}
		errs := make([]error, 0, 1+len(e.Errs))
		errs = append(errs, e.Cause)
		for _, he := range e.Errs {
			errs = append(errs, he)
		}
		return errors.Join(errs...)
	}
	if len(e.Errs) == 0 {
		return nil
	}
	if len(e.Errs) == 1 {
		return e.Errs[0]
	}
	errs := make([]error, len(e.Errs))
	for i, he := range e.Errs {
		errs[i] = he
	}
	return errors.Join(errs...)
}

// Is reports whether target matches this error for errors.Is.
//
// Matching rules:
//   - pointer identity is handled by errors.Is before calling Is
//   - if both sides have a non-empty Code: Codes are equal
//   - otherwise: same StatusCode, Category, SubCategory, Code, Title, and Message
//
// The structural rule matches Append dedup so identical client-facing data
// is treated as the same error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok || e == nil || t == nil {
		return false
	}
	return e.same(t)
}

// same reports whether two errors represent the same logical error.
func (e *Error) same(t *Error) bool {
	if e.Code != "" && t.Code != "" {
		return e.Code == t.Code
	}
	return e.StatusCode == t.StatusCode &&
		e.Category == t.Category &&
		e.SubCategory == t.SubCategory &&
		e.Code == t.Code &&
		e.Title == t.Title &&
		e.Message == t.Message
}

// Clone returns a deep copy of the error (Extra map and Errs tree).
func (e *Error) Clone() *Error {
	if e == nil {
		return nil
	}
	clone := e.cloneHeader()
	if len(e.Errs) > 0 {
		clone.Errs = make([]*Error, len(e.Errs))
		for i, err := range e.Errs {
			clone.Errs[i] = err.Clone()
		}
	}
	return clone
}

// cloneHeader copies scalar fields and deep-copies Extra.
// Errs is left nil; callers decide how to populate it.
func (e *Error) cloneHeader() *Error {
	clone := *e
	clone.Errs = nil
	if e.Extra != nil {
		clone.Extra = maps.Clone(e.Extra)
	}
	return &clone
}

// WithMessage returns a clone with an updated message.
func (e *Error) WithMessage(message string) *Error {
	clone := e.Clone()
	clone.Message = message
	return clone
}

// WithStatus returns a clone with an updated HTTP status code.
func (e *Error) WithStatus(statusCode int) *Error {
	clone := e.Clone()
	clone.StatusCode = statusCode
	return clone
}

// WithCode returns a clone with an updated application error code.
func (e *Error) WithCode(code string) *Error {
	clone := e.Clone()
	clone.Code = code
	return clone
}

// WithCategory returns a clone with an updated category.
func (e *Error) WithCategory(category string) *Error {
	clone := e.Clone()
	clone.Category = category
	return clone
}

// WithSubCategory returns a clone with an updated sub-category.
func (e *Error) WithSubCategory(subCategory string) *Error {
	clone := e.Clone()
	clone.SubCategory = subCategory
	return clone
}

// WithTitle returns a clone with an updated title.
func (e *Error) WithTitle(title string) *Error {
	clone := e.Clone()
	clone.Title = title
	return clone
}

// WithException returns a clone with Exception set.
func (e *Error) WithException(exception bool) *Error {
	clone := e.Clone()
	clone.Exception = exception
	return clone
}

// WithExtra returns a clone with one extra field set.
func (e *Error) WithExtra(key string, value any) *Error {
	clone := e.Clone()
	if clone.Extra == nil {
		clone.Extra = make(map[string]any, 1)
	}
	clone.Extra[key] = value
	return clone
}

// WithCause returns a clone with an updated cause.
func (e *Error) WithCause(cause error) *Error {
	clone := e.Clone()
	clone.Cause = cause
	return clone
}

// Append returns a clone with additional sub-errors.
// Nil errors are skipped. Nested multi-errors are flattened.
// Duplicates (same logical error per Is/same) are ignored.
//
// Sub-error pointers are shared (errors are treated as immutable).
// Use Clone on a child first if you need an isolated copy.
func (e *Error) Append(errs ...error) *Error {
	if e == nil {
		return nil
	}
	if len(errs) == 0 {
		return e
	}

	clone := e.cloneHeader()
	if n := len(e.Errs); n > 0 {
		clone.Errs = make([]*Error, n, n+len(errs))
		copy(clone.Errs, e.Errs)
	}

	for _, err := range errs {
		clone.appendOne(err)
	}
	return clone
}

func (e *Error) appendOne(err error) {
	if err == nil {
		return
	}
	he := From(err)
	if he == nil {
		return
	}
	// Flatten aggregates; share leaf pointers under the immutability convention.
	if len(he.Errs) > 0 {
		for _, sub := range he.Errs {
			e.appendOne(sub)
		}
		return
	}
	if e.exists(he) {
		return
	}
	e.Errs = append(e.Errs, he)
}

func (e *Error) exists(he *Error) bool {
	for _, item := range e.Errs {
		if item.same(he) {
			return true
		}
	}
	return false
}

// IsEmpty reports whether there are no sub-errors.
func (e *Error) IsEmpty() bool {
	return e == nil || len(e.Errs) == 0
}

// IsNotEmpty reports whether there is at least one sub-error.
func (e *Error) IsNotEmpty() bool {
	return !e.IsEmpty()
}

// StatusOr returns StatusCode, or fallback when StatusCode is zero.
func (e *Error) StatusOr(fallback int) int {
	if e == nil || e.StatusCode == 0 {
		return fallback
	}
	return e.StatusCode
}

// StatusOrInternal returns StatusCode, or 500 when zero.
func (e *Error) StatusOrInternal() int {
	return e.StatusOr(http.StatusInternalServerError)
}
